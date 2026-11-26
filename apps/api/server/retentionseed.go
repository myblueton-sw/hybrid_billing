package server

import (
	"errors"
	"strconv"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/retention"
)

const (
	retentionDatabase    = "sample-database"
	retentionObjectStore = "sample-object-store"
	retentionIngest      = "sample-retention-ingest"
	retentionWorker      = "sample-retention-worker"
	retentionDispute     = "sample-hold-dispute"
	retentionTemporary   = "sample-hold-temporary"
	retentionRunPrefix   = "run/"
	retentionHoldPrefix  = "retention-hold-"
	retentionRetired     = "retired"
)

type retentionArtifact struct {
	id      string
	class   string
	tenants []string
}

type retentionHold struct {
	id        string
	workspace string
	created   time.Time
	rationale string
	released  bool
	number    int
}

type retentionRun struct {
	id        string
	workspace string
}

type retentionEntry struct {
	view      retention.ProposalView
	workspace string
	created   time.Time
	rationale string
	closedBy  string
	closing   string
}

type retentionStore struct {
	artifacts []*retentionArtifact
	holds     []*retentionHold
	runs      []*retentionRun
	entries   []*retentionEntry
	holdSeq   uint64
	plans     uint64
	dropped   int
}

func newRetentionStore() *retentionStore { return &retentionStore{} }

func (c *retentionStore) artifact(id string) *retentionArtifact {
	for _, a := range c.artifacts {
		if a.id == id {
			return a
		}
	}
	return nil
}

func (c *retentionStore) hold(id string) *retentionHold {
	for _, h := range c.holds {
		if h.id == id {
			return h
		}
	}
	return nil
}

func (c *retentionStore) run(id string) *retentionRun {
	for _, r := range c.runs {
		if r.id == id {
			return r
		}
	}
	return nil
}

func (c *retentionStore) entry(number uint64) *retentionEntry {
	for _, e := range c.entries {
		if e.view.Number == number {
			return e
		}
	}
	return nil
}

func (c *retentionStore) keep(e *retentionEntry) {
	c.entries = append(c.entries, e)
	if len(c.entries) <= installKept {
		return
	}
	for index, old := range c.entries {
		if old.view.State != retention.ProposalPending {
			c.entries = append(c.entries[:index], c.entries[index+1:]...)
			c.dropped++
			return
		}
	}
}

func (c *retentionStore) visibleArtifact(id, workspace string) bool {
	a := c.artifact(id)
	if a == nil {
		return false
	}
	for _, tenant := range a.tenants {
		if tenant == workspace {
			return true
		}
	}
	return false
}

func (c *retentionStore) visibleHold(id, workspace string) bool {
	h := c.hold(id)
	return h != nil && h.workspace == workspace
}

func (c *retentionStore) visibleRun(id, workspace string) bool {
	r := c.run(id)
	return r != nil && r.workspace == workspace
}

func (c *retentionStore) visibleArtifacts(ids []string, workspace string) bool {
	for _, id := range ids {
		if !c.visibleArtifact(id, workspace) {
			return false
		}
	}
	return true
}

func (c *retentionStore) visibleEntry(e *retentionEntry, workspace string) bool {
	switch x := e.view.Change; {
	case x.Kind == retention.ChangePolicy:
		return true
	case e.workspace != workspace:
		return false
	case x.Kind == retention.ChangeRelease:
		return c.visibleHold(x.Release.Hold, workspace)
	}
	return c.visibleArtifacts(e.view.Change.Plan.Artifacts, workspace)
}

func (c *retentionStore) recordRun(id, workspace string, artifacts []string) {
	if c.visibleArtifacts(artifacts, workspace) {
		c.runs = append(c.runs, &retentionRun{id: id, workspace: workspace})
	}
}

func (c *retentionStore) liveHolds(workspace string, released func(id string) (bool, error)) (int, error) {
	live := 0
	for _, h := range c.holds {
		if h.workspace != workspace {
			continue
		}
		gone, err := released(h.id)
		if err != nil {
			return 0, err
		}
		if !gone {
			live++
		}
	}
	return live, nil
}

func (c *retentionStore) pendingPolicy(class string) *retentionEntry {
	for _, e := range c.entries {
		if v := e.view; v.State == retention.ProposalPending && v.Change.Kind == retention.ChangePolicy && v.Change.Policy.Class == class {
			return e
		}
	}
	return nil
}

func (c *retentionStore) pendingRelease(hold string) *retentionEntry {
	for _, e := range c.entries {
		if v := e.view; v.State == retention.ProposalPending && v.Change.Kind == retention.ChangeRelease && v.Change.Release.Hold == hold {
			return e
		}
	}
	return nil
}

func (s *Server) retentionCall(name string, call func() error) error {
	return s.moduleCall("retention-"+name, retention.ErrInvalid, call)
}

func (s *Server) retentionPropose(c retention.Change, by retention.Actor, workspace, rationale string) (*retentionEntry, error) {
	var view retention.ProposalView
	if err := s.retentionCall("propose", func() (err error) { view, err = s.retention.Propose(c, by); return }); err != nil {
		return nil, err
	}
	e := s.ret.entry(view.Number)
	if e == nil {
		e = &retentionEntry{view: view, workspace: workspace, created: s.clock(), rationale: rationale}
		s.ret.keep(e)
	}
	return e, nil
}

func (s *Server) retireRetention(e *retentionEntry, err error) error {
	var refusal retention.Refusal
	gone := errors.Is(err, retention.ErrNotFound) || errors.As(err, &refusal) && refusal.Reason == retention.ReasonNotPending
	if gone && e.view.State == retention.ProposalPending {
		e.view.State = retentionRetired
	}
	return err
}

func (s *Server) retentionApprove(e *retentionEntry, by retention.Actor, reviewed retention.Impact, rationale string) error {
	var view retention.ProposalView
	err := s.retentionCall("approve", func() (err error) { view, err = s.retention.Approve(e.view.Number, by, reviewed); return })
	if err != nil {
		return s.retireRetention(e, err)
	}
	if e.view.State == retention.ProposalPending {
		e.closedBy, e.closing = by.ID, rationale
	}
	e.view = view
	if view.Change.Kind == retention.ChangeRelease && view.State == retention.ProposalApplied {
		if h := s.ret.hold(view.Change.Release.Hold); h != nil {
			h.released = true
		}
	}
	return nil
}

func (s *Server) retentionWithdraw(e *retentionEntry, by retention.Actor, rationale string) error {
	var view retention.ProposalView
	err := s.retentionCall("withdraw", func() (err error) { view, err = s.retention.Withdraw(e.view.Number, by); return })
	if err != nil {
		return s.retireRetention(e, err)
	}
	e.view, e.closedBy, e.closing = view, by.ID, rationale
	return nil
}

func (s *Server) retentionPlaceHold(h retention.Hold, by retention.Actor, workspace, rationale string) (retention.HoldView, error) {
	var view retention.HoldView
	if err := s.retentionCall("place_hold", func() (err error) { view, err = s.retention.PlaceHold(h, by); return }); err != nil {
		return retention.HoldView{}, err
	}
	created := s.clock()
	s.ret.holds = append(s.ret.holds, &retentionHold{id: h.ID, workspace: workspace, created: created, rationale: rationale, number: len(s.ret.holds) + 1})
	s.holdPlaced.Store(h.ID, created)
	return view, nil
}

func (s *Server) retentionHold(id string) (retention.HoldView, error) {
	var view retention.HoldView
	err := s.retentionCall("hold", func() (err error) { view, err = s.retention.Hold(id); return })
	return view, err
}

func (s *Server) retentionRunView(id string) (retention.RunView, error) {
	var view retention.RunView
	err := s.retentionCall("run", func() (err error) { view, err = s.retention.Run(id); return })
	return view, err
}

func (s *Server) retentionPolicy(class string) (retention.PolicyView, error) {
	var view retention.PolicyView
	err := s.retentionCall("policy", func() (err error) { view, err = s.retention.Policy(class); return })
	return view, err
}

func (s *Server) retentionStanding(id string) (retention.Standing, error) {
	var view retention.Standing
	err := s.retentionCall("standing", func() (err error) { view, err = s.retention.Standing(id); return })
	return view, err
}

func (s *Server) retentionArtifact(id string) (retention.Artifact, error) {
	var view retention.Artifact
	err := s.retentionCall("artifact", func() (err error) { view, err = s.retention.Artifact(id); return })
	return view, err
}

func samePlan(a, b retention.DeletionPlan) bool {
	if a.Class != b.Class || a.Chunk != b.Chunk || a.Lease != b.Lease || a.Reason != b.Reason || len(a.Artifacts) != len(b.Artifacts) {
		return false
	}
	for index := range a.Artifacts {
		if a.Artifacts[index] != b.Artifacts[index] {
			return false
		}
	}
	return true
}

type retentionSample struct {
	ids    []string
	class  string
	age    int
	stores []string
}

func retentionSamples() []retentionSample {
	both := []string{retentionDatabase, retentionObjectStore}
	database, object := []string{retentionDatabase}, []string{retentionObjectStore}
	return []retentionSample{
		{[]string{"sample-source-1", "sample-source-2"}, retention.ClassSource, 400, both},
		{[]string{"sample-source-3", "sample-source-4"}, retention.ClassSource, 400, database},
		{[]string{"sample-source-5"}, retention.ClassSource, 300, database},
		{[]string{"sample-source-6"}, retention.ClassSource, 250, database},
		{[]string{"sample-source-7"}, retention.ClassSource, 30, database},
		{[]string{"sample-final-1"}, retention.ClassFinal, 100, database},
		{[]string{"sample-claim-1"}, retention.ClassClaim, 500, database},
		{[]string{"sample-temporary-1", "sample-temporary-2", "sample-temporary-3"}, retention.ClassTemporary, 20, object},
		{[]string{"sample-conversation-1"}, retention.ClassConversation, 10, database},
		{[]string{"sample-projection-1"}, retention.ClassProjection, 5, object},
	}
}

func (s *Server) seedRetention(now time.Time) error {
	ws := s.Workspace()
	first := retention.Actor{ID: SampleAdmin, Kind: retention.KindHuman}
	second := retention.Actor{ID: SampleAdmin2, Kind: retention.KindHuman}
	member := retention.Actor{ID: SampleMember, Kind: retention.KindHuman}
	ingest := retention.Actor{ID: retentionIngest, Kind: retention.KindSystem}
	worker := retention.Actor{ID: retentionWorker, Kind: retention.KindSystem}
	approved := func(c retention.Change) func() error {
		return func() error {
			e, err := s.retentionPropose(c, first, ws, installSeedReason)
			if err != nil {
				return err
			}
			return s.retentionApprove(e, second, e.view.Change.Impact, installSeedReason)
		}
	}
	policy := func(class string, period, grace int) func() error {
		return approved(retention.Change{Kind: retention.ChangePolicy, Policy: retention.Policy{Class: class, Period: period, Grace: grace}})
	}
	plan := func(p retention.DeletionPlan) func() error {
		return func() error {
			if err := approved(retention.Change{Kind: retention.ChangePlan, Plan: p})(); err != nil {
				return err
			}
			s.ret.plans++
			id := retentionRunPrefix + strconv.FormatUint(s.ret.plans, 10)
			v, err := s.retentionRunView(id)
			switch {
			case err != nil:
				return err
			case !samePlan(v.Plan, p):
				return retention.ErrConflict
			}
			s.ret.recordRun(id, ws, p.Artifacts)
			return nil
		}
	}
	var token uint64
	claim := func(run string, chunk int) func() error {
		return func() error {
			return s.retentionCall("claim", func() error {
				c, err := s.retention.Claim(run, chunk, worker)
				token = c.Token
				return err
			})
		}
	}
	proceed := func(run, id string) func() error {
		return func() error {
			return s.retentionCall("proceed", func() error { _, err := s.retention.Proceed(run, 0, token, id, worker); return err })
		}
	}
	report := func(run string, chunk int, store string, deleted, failed []string) func() error {
		return func() error {
			return s.retentionCall("report", func() error { return s.retention.Report(run, chunk, token, store, deleted, failed, worker) })
		}
	}
	hold := func(h retention.Hold, by retention.Actor) func() error {
		return func() error { _, err := s.retentionPlaceHold(h, by, ws, installSeedReason); return err }
	}
	steps := []func() error{}
	for _, sample := range retentionSamples() {
		for _, id := range sample.ids {
			x := retention.Artifact{ID: id, Class: sample.class, Tenants: []string{ws}, Anchor: now.Add(-time.Duration(sample.age) * 24 * time.Hour), Stores: sample.stores}
			steps = append(steps, func() error {
				if err := s.retentionCall("register", func() error { return s.retention.Register(x, ingest) }); err != nil {
					return err
				}
				s.ret.artifacts = append(s.ret.artifacts, &retentionArtifact{id: x.ID, class: x.Class, tenants: append([]string{}, x.Tenants...)})
				return nil
			})
		}
	}
	runOne, runTwo := retentionRunPrefix+"1", retentionRunPrefix+"2"
	steps = append(steps,
		func() error {
			return s.retentionCall("refer", func() error {
				return s.retention.Refer(retention.Reference{From: "sample-final-1", To: "sample-source-3", Kind: retention.Requires}, ingest)
			})
		},
		func() error {
			return s.retentionCall("refer", func() error {
				return s.retention.Refer(retention.Reference{From: "sample-claim-1", To: "sample-source-1", Kind: retention.Guards}, ingest)
			})
		},
		policy(retention.ClassSource, 365, 30),
		policy(retention.ClassFinal, 3650, 0),
		policy(retention.ClassTemporary, 7, 0),
		hold(retention.Hold{ID: retentionDispute, Owner: SampleAdmin2, Reason: "Sample billing dispute hold", Artifacts: []string{"sample-source-4"}}, first),
		plan(retention.DeletionPlan{Class: retention.ClassSource, Artifacts: []string{"sample-source-1", "sample-source-2", "sample-source-3", "sample-source-4"}, Chunk: 2,
			Lease: time.Hour, Reason: "Sample expiry deletion"}),
		claim(runOne, 0),
		proceed(runOne, "sample-source-1"),
		proceed(runOne, "sample-source-2"),
		report(runOne, 0, retentionDatabase, []string{"sample-source-1", "sample-source-2"}, nil),
		report(runOne, 0, retentionObjectStore, []string{"sample-source-1"}, []string{"sample-source-2"}),
		claim(runOne, 1),
		plan(retention.DeletionPlan{Class: retention.ClassTemporary, Artifacts: []string{"sample-temporary-1", "sample-temporary-2", "sample-temporary-3"}, Chunk: 1,
			Lease: time.Hour, Reason: "Sample temporary file expiry"}),
		claim(runTwo, 0),
		report(runTwo, 0, retentionObjectStore, []string{"sample-temporary-1"}, nil),
		hold(retention.Hold{ID: retentionTemporary, Owner: SampleMember, Reason: "Sample export review hold", Tenants: []string{ws},
			Classes: []string{retention.ClassTemporary}}, member),
		func() error {
			_, err := s.retentionPropose(retention.Change{Kind: retention.ChangeRelease, Release: retention.Release{Hold: retentionTemporary,
				Reason: "Sample review finished"}}, first, ws, installSeedReason)
			return err
		},
		func() error {
			_, err := s.retentionPropose(retention.Change{Kind: retention.ChangePolicy, Policy: retention.Policy{Class: retention.ClassSource, Period: 180, Grace: 30}},
				second, ws, installSeedReason)
			return err
		},
	)
	return run(steps...)
}
