package server

import (
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/retention"
)

const (
	retentionShown       = 50
	retentionWindow      = 1000
	retentionOutcomePage = 1000
	countsWorkspace      = "workspace"
	countsOtherScope     = "other_scope"
	retentionPopulation  = "registered_sample_artifacts"
	retentionOrigin      = "artifact_anchor"
	retentionLedger      = "in_memory"
	retentionCheckText   = "Release check"
	noEligibleApprover   = "no_eligible_approver"
	targetTooLong        = "target_too_long"
	runFinished          = "finished"
	standingDeleted      = "deleted"
	standingInRun        = "in_run"
	standingNoPolicy     = "no_policy"
	standingNotDue       = "not_due"
	standingHeld         = "held"
	standingGuarding     = "guarding"
	standingReferenced   = "referenced"
	standingEligible     = "eligible"
	maxTargetBytes       = 128
)

var (
	retentionStandings = []string{standingEligible, standingHeld, standingGuarding, standingReferenced, standingNotDue, standingNoPolicy, standingInRun, standingDeleted}
	retentionOutcomes  = []string{retention.OutcomePending, retention.OutcomeDeleting, retention.OutcomeDeleted, retention.OutcomeFailed, retention.OutcomeHeld,
		retention.OutcomeGuarding, retention.OutcomeReferenced, retention.OutcomeNotDue, retention.OutcomeStopped}
	retentionHoldCounts = []string{retention.HoldNotPreserved, retention.HoldOutcomePending, retention.HoldPartlyDeleted}
)

type retentionPerson struct {
	ID    string  `json:"id"`
	Login *string `json:"login"`
}

type retentionRunRef struct {
	Target string `json:"target"`
	Label  string `json:"label"`
}

type retentionPolicyView struct {
	Revision   uint64           `json:"revision"`
	Period     int              `json:"period_days"`
	Grace      int              `json:"grace_days"`
	Proposer   *retentionPerson `json:"proposer"`
	Approver   *retentionPerson `json:"approver"`
	ApprovedAt string           `json:"approved_at"`
}

type retentionPolicyRow struct {
	Class          string               `json:"class"`
	Target         string               `json:"target"`
	Policy         *retentionPolicyView `json:"policy"`
	Origin         string               `json:"origin"`
	Scope          string               `json:"scope"`
	ArchiveRule    string               `json:"archive_rule"`
	Pending        *uint64              `json:"pending"`
	CanPropose     bool                 `json:"can_propose"`
	ProposeBlocked string               `json:"propose_blocked"`
}

type retentionArtifactRow struct {
	ID          string           `json:"id"`
	Anchor      string           `json:"anchor"`
	DueAt       *string          `json:"due_at"`
	Stores      []string         `json:"stores"`
	Access      string           `json:"access"`
	Standing    string           `json:"standing"`
	Held        bool             `json:"held"`
	Damaged     bool             `json:"damaged"`
	Run         *retentionRunRef `json:"run"`
	OtherScope  bool             `json:"other_scope"`
	Shared      bool             `json:"shared"`
	Decision    string           `json:"decision,omitempty"`
	Holdable    bool             `json:"holdable"`
	HoldBlocked string           `json:"hold_blocked"`
	HoldTarget  string           `json:"hold_target"`
}

type retentionClassGroup struct {
	Class      string                 `json:"class"`
	Population string                 `json:"population"`
	Counts     map[string]int         `json:"counts"`
	Total      int                    `json:"total"`
	More       bool                   `json:"more"`
	Artifacts  []retentionArtifactRow `json:"artifacts"`
	Usage      string                 `json:"usage"`
	Growth     string                 `json:"growth"`
	Bytes      string                 `json:"bytes"`
}

type retentionHoldScope struct {
	Artifacts []string `json:"artifacts"`
	Workspace bool     `json:"workspace"`
	Classes   []string `json:"classes"`
}

type retentionOutcomeRow struct {
	Artifact string            `json:"artifact"`
	Outcome  string            `json:"outcome"`
	Stores   map[string]string `json:"stores"`
}

type retentionHoldRow struct {
	ID                string                `json:"id"`
	Target            string                `json:"target"`
	Owner             *retentionPerson      `json:"owner"`
	Reason            string                `json:"reason"`
	Placer            *retentionPerson      `json:"placer"`
	PlacedAt          string                `json:"placed_at"`
	Rationale         string                `json:"rationale"`
	Released          bool                  `json:"released"`
	Scope             retentionHoldScope    `json:"scope"`
	Counts            map[string]int        `json:"counts"`
	CountsScope       string                `json:"counts_scope"`
	Outcomes          []retentionOutcomeRow `json:"outcomes"`
	OutcomesMore      bool                  `json:"outcomes_more"`
	Release           *uint64               `json:"release"`
	CanProposeRelease bool                  `json:"can_propose_release"`
	ReleaseBlocked    string                `json:"release_blocked"`
}

type retentionReferenceRow struct {
	ID    string `json:"id"`
	Class string `json:"class"`
}

type retentionReferences struct {
	Population string                  `json:"population"`
	Total      int                     `json:"total"`
	More       bool                    `json:"more"`
	Rows       []retentionReferenceRow `json:"rows"`
}

type retentionHoldsSection struct {
	Total        int                 `json:"total"`
	More         bool                `json:"more"`
	Rows         []retentionHoldRow  `json:"rows"`
	TenantTarget string              `json:"tenant_target"`
	References   retentionReferences `json:"references"`
}

type retentionStoreRow struct {
	Store    string `json:"store"`
	Target   int    `json:"target"`
	Deleted  int    `json:"deleted"`
	Failed   int    `json:"failed"`
	Excluded int    `json:"excluded"`
	Pending  int    `json:"pending"`
}

type retentionRunArtifactRow struct {
	ID      string `json:"id"`
	Outcome string `json:"outcome"`
}

type retentionRunArtifacts struct {
	Total int                       `json:"total"`
	More  bool                      `json:"more"`
	Rows  []retentionRunArtifactRow `json:"rows"`
}

type retentionRunRow struct {
	Run          retentionRunRef       `json:"run"`
	Class        string                `json:"class"`
	Revision     uint64                `json:"revision"`
	State        string                `json:"state"`
	Decision     string                `json:"decision"`
	NextChunk    int                   `json:"next_chunk"`
	Chunks       int                   `json:"chunks"`
	ChunkSize    int                   `json:"chunk_size"`
	LeaseSeconds int64                 `json:"lease_seconds"`
	Reason       string                `json:"reason"`
	Damaged      int                   `json:"damaged"`
	Proposer     *retentionPerson      `json:"proposer"`
	Approver     *retentionPerson      `json:"approver"`
	Stopper      *retentionPerson      `json:"stopper"`
	Counts       map[string]int        `json:"counts"`
	Stores       []retentionStoreRow   `json:"stores"`
	Artifacts    retentionRunArtifacts `json:"artifacts"`
	CanStop      bool                  `json:"can_stop"`
	StopBlocked  string                `json:"stop_blocked"`
}

type retentionDeletionSection struct {
	Total        int               `json:"total"`
	More         bool              `json:"more"`
	Rows         []retentionRunRow `json:"rows"`
	Worker       string            `json:"worker"`
	Retry        string            `json:"retry"`
	BackupExpiry string            `json:"backup_expiry"`
	ProposePlan  string            `json:"propose_plan"`
}

type retentionRestoreSection struct {
	State string `json:"state"`
}

type retentionTombstoneRow struct {
	Sequence   uint64           `json:"sequence"`
	Artifact   string           `json:"artifact"`
	Class      string           `json:"class"`
	Run        *retentionRunRef `json:"run"`
	OtherScope bool             `json:"other_scope"`
	At         string           `json:"at"`
	Decision   string           `json:"decision"`
}

type retentionTombstones struct {
	Window int                     `json:"window"`
	Total  int                     `json:"total"`
	More   bool                    `json:"more"`
	Rows   []retentionTombstoneRow `json:"rows"`
}

type retentionDecisionsSection struct {
	Watermark     uint64              `json:"watermark"`
	Tombstones    retentionTombstones `json:"tombstones"`
	Ledger        string              `json:"ledger"`
	Epoch         string              `json:"epoch"`
	DurableCommit string              `json:"durable_commit"`
	RestoreCheck  string              `json:"restore_check"`
}

type retentionImpactRow struct {
	AsOf   string `json:"as_of"`
	Count  int    `json:"count"`
	Digest string `json:"digest"`
}

type retentionImpactNow struct {
	AsOf           string `json:"as_of"`
	Count          int    `json:"count"`
	Digest         string `json:"digest"`
	WorkspaceCount int    `json:"workspace_count"`
}

type retentionPolicySummary struct {
	Class      string              `json:"class"`
	Period     int                 `json:"period_days"`
	Grace      int                 `json:"grace_days"`
	AtProposal *retentionImpactRow `json:"impact_at_proposal"`
}

type retentionPlanSummary struct {
	Class         string `json:"class"`
	ArtifactCount int    `json:"artifact_count"`
	ChunkSize     int    `json:"chunk_size"`
	LeaseSeconds  int64  `json:"lease_seconds"`
	Reason        string `json:"reason"`
}

type retentionReleaseSummary struct {
	Hold   string `json:"hold"`
	Reason string `json:"reason"`
}

type retentionSummary struct {
	Policy  *retentionPolicySummary  `json:"policy"`
	Plan    *retentionPlanSummary    `json:"plan"`
	Release *retentionReleaseSummary `json:"release"`
}

type retentionProposalRow struct {
	Number           uint64              `json:"number"`
	Target           string              `json:"target"`
	Kind             string              `json:"kind"`
	State            string              `json:"state"`
	Created          string              `json:"created_at"`
	Proposer         *retentionPerson    `json:"proposer"`
	Approver         *retentionPerson    `json:"approver"`
	Rationale        string              `json:"rationale"`
	ClosedBy         *retentionPerson    `json:"closed_by"`
	ClosingRationale string              `json:"closing_rationale"`
	Summary          retentionSummary    `json:"summary"`
	ImpactRequired   bool                `json:"impact_required"`
	ImpactNow        *retentionImpactNow `json:"impact_now"`
	CanApprove       bool                `json:"can_approve"`
	ApproveBlocked   string              `json:"approve_blocked"`
	CanWithdraw      bool                `json:"can_withdraw"`
	WithdrawBlocked  string              `json:"withdraw_blocked"`
}

type retentionProposals struct {
	Total int                    `json:"total"`
	More  bool                   `json:"more"`
	Rows  []retentionProposalRow `json:"rows"`
}

type retentionPage struct {
	Revision  string                    `json:"revision"`
	Now       string                    `json:"now"`
	State     string                    `json:"state"`
	Policies  []retentionPolicyRow      `json:"policies"`
	Expiry    []retentionClassGroup     `json:"expiry"`
	Holds     retentionHoldsSection     `json:"holds"`
	Deletion  retentionDeletionSection  `json:"deletion"`
	Restore   retentionRestoreSection   `json:"restore"`
	Decisions retentionDecisionsSection `json:"decisions"`
	Proposals retentionProposals        `json:"proposals"`
}

type retentionFacts struct {
	s         *Server
	reader    string
	workspace string
	approvers []string
	policies  map[string]*retention.PolicyView
	artifacts map[string]retention.Artifact
	standings map[string]retention.Standing
}

func (s *Server) personOf(id string) *retentionPerson {
	if id == "" {
		return nil
	}
	p := &retentionPerson{ID: id}
	if login, ok := s.dir.loginOf(id); ok {
		p.Login = &login
	}
	return p
}

func retentionReason(err error) string {
	var refusal retention.Refusal
	switch {
	case err == nil:
		return ""
	case errors.As(err, &refusal):
		return refusal.Reason
	case errors.Is(err, retention.ErrNotFound):
		return "not_found"
	case errors.Is(err, retention.ErrLimit):
		return "limit"
	case errors.Is(err, retention.ErrConflict):
		return "conflict"
	}
	return "invalid"
}

func (s *Server) retentionCheck(c retention.Change, proposer, approver string) string {
	return retentionReason(s.retentionCall("check", func() error { return s.retention.Check(c, proposer, approver, retention.Impact{}) }))
}

func runTarget(id string) string {
	return retentionRunTarget + strings.TrimPrefix(id, retentionRunPrefix)
}

func (f *retentionFacts) runRef(id string) (*retentionRunRef, bool) {
	switch {
	case id == "":
		return nil, false
	case f.s.ret.visibleRun(id, f.workspace):
		return &retentionRunRef{Target: runTarget(id), Label: id}, false
	}
	return nil, true
}

func (f *retentionFacts) of(id string) (retention.Artifact, retention.Standing, error) {
	if a, ok := f.artifacts[id]; ok {
		return a, f.standings[id], nil
	}
	a, err := f.s.retentionArtifact(id)
	if err != nil {
		return retention.Artifact{}, retention.Standing{}, err
	}
	v, err := f.s.retentionStanding(id)
	if err != nil {
		return retention.Artifact{}, retention.Standing{}, err
	}
	f.artifacts[id], f.standings[id] = a, v
	return a, v, nil
}

func standingOf(v retention.Standing) string {
	switch {
	case v.Access == retention.AccessDeleted:
		return standingDeleted
	case v.Run != "":
		return standingInRun
	case v.Guarding:
		return standingGuarding
	case v.DueAt.IsZero():
		return standingNoPolicy
	case !v.Due:
		return standingNotDue
	case v.Held:
		return standingHeld
	case v.Referenced:
		return standingReferenced
	}
	return standingEligible
}

func decisionOf(access string) string {
	if access == retention.AccessOK {
		return ""
	}
	return notConnected
}

func zeroCounts(keys []string) map[string]int {
	out := map[string]int{}
	for _, k := range keys {
		out[k] = 0
	}
	return out
}

func (f *retentionFacts) visibleIDs() []string {
	out := []string{}
	for _, a := range f.s.ret.artifacts {
		if f.s.ret.visibleArtifact(a.id, f.workspace) {
			out = append(out, a.id)
		}
	}
	sort.Strings(out)
	return out
}

func (f *retentionFacts) artifactRow(id string) (retentionArtifactRow, error) {
	a, v, err := f.of(id)
	if err != nil {
		return retentionArtifactRow{}, err
	}
	row := retentionArtifactRow{ID: id, Anchor: stamp(a.Anchor), Stores: append([]string{}, a.Stores...), Access: v.Access, Standing: standingOf(v), Held: v.Held,
		Damaged: v.Damaged, Decision: decisionOf(v.Access), Holdable: true}
	row.Run, row.OtherScope = f.runRef(v.Run)
	if !v.DueAt.IsZero() {
		due := stamp(v.DueAt)
		row.DueAt = &due
	}
	for _, tenant := range f.s.ret.artifact(id).tenants {
		row.Shared = row.Shared || tenant != f.workspace
	}
	if target := retentionArtifactTarget + id; len(target) > maxTargetBytes {
		row.Holdable, row.HoldBlocked = false, targetTooLong
	} else {
		row.HoldTarget = target
	}
	return row, nil
}

func (f *retentionFacts) expiry(ids []string) ([]retentionClassGroup, error) {
	out := []retentionClassGroup{}
	for _, class := range retention.Classes {
		g := retentionClassGroup{Class: class, Population: retentionPopulation, Counts: zeroCounts(retentionStandings), Artifacts: []retentionArtifactRow{}, Usage: notConnected,
			Growth: notConnected, Bytes: notConnected}
		for _, id := range ids {
			if f.s.ret.artifact(id).class != class {
				continue
			}
			row, err := f.artifactRow(id)
			if err != nil {
				return nil, err
			}
			g.Counts[row.Standing]++
			g.Total++
			if len(g.Artifacts) < retentionShown {
				g.Artifacts = append(g.Artifacts, row)
			}
		}
		g.More = g.Total > len(g.Artifacts)
		out = append(out, g)
	}
	return out, nil
}

func (f *retentionFacts) releaseBlocked(id string, v retention.HoldView) string {
	switch {
	case v.Released:
		return retention.ReasonReleased
	case f.s.ret.pendingRelease(id) != nil:
		return alreadyProposed
	}
	eligible := 0
	for _, user := range f.approvers {
		if user != v.Placer {
			eligible++
		}
	}
	if eligible < 2 {
		return noEligibleApprover
	}
	return f.s.retentionCheck(retention.Change{Kind: retention.ChangeRelease, Release: retention.Release{Hold: id, Reason: retentionCheckText}}, f.reader, "")
}

func (f *retentionFacts) visibleOutcomes(id string) ([]retention.HoldOutcome, error) {
	s := f.s
	out, after := []retention.HoldOutcome{}, ""
	for {
		var page []retention.HoldOutcome
		if err := s.retentionCall("hold_outcomes", func() (err error) { page, err = s.retention.HoldOutcomes(id, after, retentionOutcomePage); return }); err != nil {
			return nil, err
		}
		for _, o := range page {
			if s.ret.visibleArtifact(o.Artifact, f.workspace) {
				out = append(out, o)
			}
		}
		if len(out) > retentionShown || len(page) < retentionOutcomePage {
			return out, nil
		}
		after = page[len(page)-1].Artifact
	}
}

func (f *retentionFacts) holdRow(h *retentionHold) (retentionHoldRow, error) {
	s := f.s
	v, err := s.retentionHold(h.id)
	if err != nil {
		return retentionHoldRow{}, err
	}
	outcomes, err := f.visibleOutcomes(h.id)
	if err != nil {
		return retentionHoldRow{}, err
	}
	row := retentionHoldRow{ID: h.id, Target: retentionHoldTarget + h.id, Owner: s.personOf(v.Hold.Owner), Reason: v.Hold.Reason, Placer: s.personOf(v.Placer),
		PlacedAt: stamp(h.created), Rationale: h.rationale, Released: v.Released, Scope: retentionHoldScope{Artifacts: []string{}, Classes: append([]string{}, v.Hold.Classes...)},
		Counts: zeroCounts(retentionHoldCounts), Outcomes: []retentionOutcomeRow{}, OutcomesMore: len(outcomes) > retentionShown}
	for _, id := range v.Hold.Artifacts {
		if s.ret.visibleArtifact(id, f.workspace) {
			row.Scope.Artifacts = append(row.Scope.Artifacts, id)
		}
	}
	other := len(row.Scope.Artifacts) != len(v.Hold.Artifacts)
	for _, tenant := range v.Hold.Tenants {
		row.Scope.Workspace = row.Scope.Workspace || tenant == f.workspace
		other = other || tenant != f.workspace
	}
	row.CountsScope = countsWorkspace
	if other {
		row.Counts, row.CountsScope = nil, countsOtherScope
	}
	for key, n := range v.Counts {
		if row.Counts != nil {
			row.Counts[key] = n
		}
	}
	for _, o := range outcomes[:min(len(outcomes), retentionShown)] {
		row.Outcomes = append(row.Outcomes, retentionOutcomeRow{Artifact: o.Artifact, Outcome: o.Outcome, Stores: o.Stores})
	}
	if e := s.ret.pendingRelease(h.id); e != nil {
		number := e.view.Number
		row.Release = &number
	}
	row.ReleaseBlocked = f.releaseBlocked(h.id, v)
	row.CanProposeRelease = row.ReleaseBlocked == ""
	return row, nil
}

func (f *retentionFacts) holds(ids []string) (retentionHoldsSection, error) {
	section := retentionHoldsSection{Rows: []retentionHoldRow{}, TenantTarget: retentionTenantTarget + f.workspace,
		References: retentionReferences{Population: retentionPopulation, Rows: []retentionReferenceRow{}}}
	for index := len(f.s.ret.holds) - 1; index >= 0; index-- {
		h := f.s.ret.holds[index]
		if h.workspace != f.workspace {
			continue
		}
		section.Total++
		if len(section.Rows) >= retentionShown {
			continue
		}
		row, err := f.holdRow(h)
		if err != nil {
			return retentionHoldsSection{}, err
		}
		section.Rows = append(section.Rows, row)
	}
	section.More = section.Total > len(section.Rows)
	refs := &section.References
	for _, id := range ids {
		if !f.standings[id].Referenced {
			continue
		}
		refs.Total++
		if len(refs.Rows) < retentionShown {
			refs.Rows = append(refs.Rows, retentionReferenceRow{ID: id, Class: f.s.ret.artifact(id).class})
		}
	}
	refs.More = refs.Total > len(refs.Rows)
	return section, nil
}

func excludedOutcome(outcome string) bool {
	switch outcome {
	case retention.OutcomeHeld, retention.OutcomeGuarding, retention.OutcomeReferenced, retention.OutcomeNotDue, retention.OutcomeStopped:
		return true
	}
	return false
}

func (f *retentionFacts) storeRows(v retention.RunView) ([]retentionStoreRow, error) {
	excluded := map[string]int{}
	for _, id := range v.Plan.Artifacts {
		if !excludedOutcome(v.Outcomes[id]) {
			continue
		}
		a, standing, err := f.of(id)
		if err != nil {
			return nil, err
		}
		for _, store := range a.Stores {
			if standing.Stores[store] == "" {
				excluded[store]++
			}
		}
	}
	names := []string{}
	for store := range v.Stores {
		names = append(names, store)
	}
	sort.Strings(names)
	out := []retentionStoreRow{}
	for _, store := range names {
		c := v.Stores[store]
		out = append(out, retentionStoreRow{Store: store, Target: c.Target, Deleted: c.Deleted, Failed: c.Failed, Excluded: excluded[store], Pending: c.Pending - excluded[store]})
	}
	return out, nil
}

func stopBlocked(state string) string {
	switch state {
	case retention.RunRunning:
		return ""
	case retention.RunStopped:
		return retention.RunStopped
	}
	return runFinished
}

func (f *retentionFacts) runRow(r *retentionRun) (retentionRunRow, error) {
	s := f.s
	v, err := s.retentionRunView(r.id)
	if err != nil {
		return retentionRunRow{}, err
	}
	stores, err := f.storeRows(v)
	if err != nil {
		return retentionRunRow{}, err
	}
	row := retentionRunRow{Run: retentionRunRef{Target: runTarget(r.id), Label: r.id}, Class: v.Plan.Class, Revision: v.Revision, State: v.State, Decision: notConnected,
		NextChunk: v.Next, Chunks: v.Chunks, ChunkSize: v.Plan.Chunk, LeaseSeconds: int64(v.Plan.Lease / time.Second), Reason: v.Plan.Reason, Damaged: v.Damaged,
		Proposer: s.personOf(v.Proposer), Approver: s.personOf(v.Approver), Stopper: s.personOf(v.Stopper), Counts: zeroCounts(retentionOutcomes), Stores: stores,
		Artifacts: retentionRunArtifacts{Rows: []retentionRunArtifactRow{}}, StopBlocked: stopBlocked(v.State)}
	row.CanStop = row.StopBlocked == ""
	for key, n := range v.Counts {
		row.Counts[key] = n
	}
	list := &row.Artifacts
	for _, id := range v.Plan.Artifacts {
		if !s.ret.visibleArtifact(id, f.workspace) {
			continue
		}
		list.Total++
		if len(list.Rows) < retentionShown {
			list.Rows = append(list.Rows, retentionRunArtifactRow{ID: id, Outcome: v.Outcomes[id]})
		}
	}
	list.More = list.Total > len(list.Rows)
	return row, nil
}

func (f *retentionFacts) deletion() (retentionDeletionSection, error) {
	section := retentionDeletionSection{Rows: []retentionRunRow{}, Worker: notConnected, Retry: notConnected, BackupExpiry: notConnected, ProposePlan: notConnected}
	for index := len(f.s.ret.runs) - 1; index >= 0; index-- {
		r := f.s.ret.runs[index]
		if r.workspace != f.workspace {
			continue
		}
		section.Total++
		if len(section.Rows) >= retentionShown {
			continue
		}
		row, err := f.runRow(r)
		if err != nil {
			return retentionDeletionSection{}, err
		}
		section.Rows = append(section.Rows, row)
	}
	section.More = section.Total > len(section.Rows)
	return section, nil
}

func (f *retentionFacts) decisions() (retentionDecisionsSection, error) {
	s := f.s
	section := retentionDecisionsSection{Tombstones: retentionTombstones{Window: retentionWindow, Rows: []retentionTombstoneRow{}}, Ledger: retentionLedger,
		Epoch: notConnected, DurableCommit: notConnected, RestoreCheck: notConnected}
	if err := s.retentionCall("watermark", func() error { section.Watermark = s.retention.Watermark(); return nil }); err != nil {
		return retentionDecisionsSection{}, err
	}
	after := uint64(0)
	if section.Watermark > retentionWindow {
		after = section.Watermark - retentionWindow
	}
	var list []retention.Tombstone
	if err := s.retentionCall("tombstones", func() (err error) { list, err = s.retention.Tombstones(after, retentionWindow); return }); err != nil {
		return retentionDecisionsSection{}, err
	}
	t := &section.Tombstones
	for index := len(list) - 1; index >= 0; index-- {
		x := list[index]
		if !s.ret.visibleArtifact(x.Artifact, f.workspace) {
			continue
		}
		t.Total++
		if len(t.Rows) < retentionShown {
			row := retentionTombstoneRow{Sequence: x.Sequence, Artifact: x.Artifact, Class: x.Class, At: stamp(x.At), Decision: notConnected}
			row.Run, row.OtherScope = f.runRef(x.Run)
			t.Rows = append(t.Rows, row)
		}
	}
	t.More = t.Total > len(t.Rows)
	return section, nil
}

func impactRow(i retention.Impact) *retentionImpactRow {
	if i.AsOf.IsZero() {
		return nil
	}
	return &retentionImpactRow{AsOf: i.AsOf.UTC().Format(time.RFC3339Nano), Count: i.Count, Digest: i.Digest}
}

func retentionSummaryOf(c retention.Change) retentionSummary {
	switch c.Kind {
	case retention.ChangePolicy:
		return retentionSummary{Policy: &retentionPolicySummary{Class: c.Policy.Class, Period: c.Policy.Period, Grace: c.Policy.Grace, AtProposal: impactRow(c.Impact)}}
	case retention.ChangePlan:
		p := c.Plan
		return retentionSummary{Plan: &retentionPlanSummary{Class: p.Class, ArtifactCount: len(p.Artifacts), ChunkSize: p.Chunk, LeaseSeconds: int64(p.Lease / time.Second),
			Reason: p.Reason}}
	}
	return retentionSummary{Release: &retentionReleaseSummary{Hold: c.Release.Hold, Reason: c.Release.Reason}}
}

func (f *retentionFacts) shortening(p retention.Policy) bool {
	cur := f.policies[p.Class]
	return cur != nil && p.Period+p.Grace < cur.Policy.Period+cur.Policy.Grace
}

func (f *retentionFacts) blockers(v retention.ProposalView) (string, string) {
	if v.State != retention.ProposalPending {
		blocked := retention.ReasonNotPending
		if v.State == retentionRetired {
			blocked = retentionRetired
		}
		return blocked, blocked
	}
	withdraw := ""
	if v.Proposer != f.reader {
		withdraw = retention.ReasonNotProposer
	}
	switch {
	case v.Change.Kind == retention.ChangePlan:
		return notConsoleKind, withdraw
	case v.Proposer == f.reader:
		return ownProposal, withdraw
	}
	approve := f.s.retentionCheck(v.Change, v.Proposer, f.reader)
	if approve == retention.ReasonImpactRequired {
		approve = ""
	}
	return approve, withdraw
}

func (f *retentionFacts) proposalRow(e *retentionEntry) (retentionProposalRow, error) {
	s, v := f.s, e.view
	row := retentionProposalRow{Number: v.Number, Target: retentionProposalTarget + strconv.FormatUint(v.Number, 10), Kind: v.Change.Kind, State: v.State,
		Created: stamp(e.created), Proposer: s.personOf(v.Proposer), Approver: s.personOf(v.Approver), Rationale: e.rationale, ClosedBy: s.personOf(e.closedBy),
		ClosingRationale: e.closing, Summary: retentionSummaryOf(v.Change)}
	if v.State == retention.ProposalPending && v.Change.Kind == retention.ChangePolicy && f.shortening(v.Change.Policy) {
		var impact retention.ImpactView
		if err := s.retentionCall("impact", func() (err error) { impact, err = s.retention.Impact(v.Change.Policy); return }); err != nil {
			return retentionProposalRow{}, err
		}
		row.ImpactRequired = true
		row.ImpactNow = &retentionImpactNow{AsOf: impact.Impact.AsOf.UTC().Format(time.RFC3339Nano), Count: impact.Impact.Count, Digest: impact.Impact.Digest,
			WorkspaceCount: impact.Tenants[f.workspace]}
	}
	row.ApproveBlocked, row.WithdrawBlocked = f.blockers(v)
	row.CanApprove, row.CanWithdraw = row.ApproveBlocked == "", row.WithdrawBlocked == ""
	return row, nil
}

func (f *retentionFacts) proposals() (retentionProposals, error) {
	section := retentionProposals{Rows: []retentionProposalRow{}}
	for index := len(f.s.ret.entries) - 1; index >= 0; index-- {
		e := f.s.ret.entries[index]
		if !f.s.ret.visibleEntry(e, f.workspace) {
			continue
		}
		section.Total++
		if len(section.Rows) >= retentionShown {
			continue
		}
		row, err := f.proposalRow(e)
		if err != nil {
			return retentionProposals{}, err
		}
		section.Rows = append(section.Rows, row)
	}
	section.More = section.Total > len(section.Rows)
	return section, nil
}

func (f *retentionFacts) policyRows() ([]retentionPolicyRow, error) {
	s := f.s
	out := []retentionPolicyRow{}
	for _, class := range retention.Classes {
		row := retentionPolicyRow{Class: class, Target: retentionClassTarget + class, Origin: retentionOrigin, Scope: notConnected, ArchiveRule: notConnected}
		v, err := s.retentionPolicy(class)
		switch {
		case err == nil:
			f.policies[class] = &v
			row.Policy = &retentionPolicyView{Revision: v.Number, Period: v.Policy.Period, Grace: v.Policy.Grace, Proposer: s.personOf(v.Proposer),
				Approver: s.personOf(v.Approver), ApprovedAt: stamp(v.At)}
		case !errors.Is(err, retention.ErrNotFound):
			return nil, err
		}
		if e := s.ret.pendingPolicy(class); e != nil {
			number := e.view.Number
			row.Pending, row.ProposeBlocked = &number, alreadyProposed
		}
		row.CanPropose = row.ProposeBlocked == ""
		out = append(out, row)
	}
	return out, nil
}

func (s *Server) retentionApprovers(verifier identity.Verifier, now time.Time) ([]string, error) {
	members, err := s.roster(verifier)
	if err != nil {
		return nil, err
	}
	out := []string{}
	for user, m := range members {
		if m.Effective && m.UserState == identity.Active && s.dir.holds(user, now) {
			out = append(out, user)
		}
	}
	return out, nil
}

func (s *Server) retentionPage(verifier identity.Verifier, reader, workspace string, now time.Time) (retentionPage, error) {
	if s.faulty("retention-read") {
		return retentionPage{}, retention.ErrInvalid
	}
	approvers, err := s.retentionApprovers(verifier, now)
	if err != nil {
		return retentionPage{}, err
	}
	f := &retentionFacts{s: s, reader: reader, workspace: workspace, approvers: approvers, policies: map[string]*retention.PolicyView{},
		artifacts: map[string]retention.Artifact{}, standings: map[string]retention.Standing{}}
	page := retentionPage{Revision: s.Revision(), Now: stamp(now), State: stateConfigured, Restore: retentionRestoreSection{State: notConnected}}
	ids := f.visibleIDs()
	if page.Policies, err = f.policyRows(); err != nil {
		return retentionPage{}, err
	}
	if page.Expiry, err = f.expiry(ids); err != nil {
		return retentionPage{}, err
	}
	if page.Holds, err = f.holds(ids); err != nil {
		return retentionPage{}, err
	}
	if page.Deletion, err = f.deletion(); err != nil {
		return retentionPage{}, err
	}
	if page.Decisions, err = f.decisions(); err != nil {
		return retentionPage{}, err
	}
	page.Proposals, err = f.proposals()
	return page, err
}

func (s *Server) readRetention(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	defer s.domain.RUnlock()
	found, answer, ok := s.admitted(w, r)
	if !ok {
		return
	}
	now := s.clock()
	if answer.Workspace() != s.bootWorkspace {
		writeJSON(w, http.StatusOK, request, installationEmpty{Revision: s.Revision(), Now: stamp(now), State: stateNotConfigured})
		return
	}
	page, err := s.retentionPage(found.verifier, answer.User(), answer.Workspace(), now)
	if err != nil {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	writeJSON(w, http.StatusOK, request, page)
}
