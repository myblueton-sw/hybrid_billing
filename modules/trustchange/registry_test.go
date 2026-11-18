package trustchange

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestNewAndRunPrefixes(t *testing.T) {
	for _, run := range []string{"", "0A1B2C3D", "0a1b2c3", "zzzzzzzz", "0a1b2c3d4"} {
		if _, err := New(run, time.Now); err != ErrInvalid {
			t.Errorf("run %q: %v", run, err)
		}
	}
	if _, err := New("0a1b2c3d", nil); err != ErrInvalid {
		t.Error("no clock")
	}
	r, err := New("0a1b2c3d", time.Now)
	if err != nil || r.Run() != "0a1b2c3d" {
		t.Fatal(err)
	}
	for id, want := range map[string]bool{"trust-0a1b2c3d-1": true, "trust-0a1b2c3d-12": true, "trust-0a1b2c3d-0": false, "trust-0a1b2c3d-01": false,
		"trust-0a1b2c3d-": false, "trust-0a1b2c3d": false, "trust-0A1B2C3D-1": false, "proposal-1": false, "trust-0a1b2c3d-1x": false, "trust-zzzzzzzz-1": false, "0a1b2c3d-1": false} {
		if _, ok := RunOf(id); ok != want {
			t.Errorf("%s: %v", id, ok)
		}
	}
	run, err := Draw(bytes.NewReader([]byte{0, 0, 0, 0, 0, 0, 0, 1}), map[string]bool{"00000000": true})
	if err != nil || run != "00000001" {
		t.Errorf("a colliding prefix is redrawn: %s %v", run, err)
	}
	if _, err := Draw(bytes.NewReader([]byte{1}), nil); err == nil {
		t.Error("a short random source")
	}
}

func TestDigests(t *testing.T) {
	c := candidate(60)
	if ConfigDigest(c) != ConfigDigest(candidate(60)) || ConfigDigest(c) == ConfigDigest(candidate(61)) || EmptyDigest() == ConfigDigest(c) || len(EmptyDigest()) != 64 {
		t.Error("config digest")
	}
	variants := []Candidate{
		{Methods: []string{"password"}, MFA: "administrators", MaxAge: time.Hour},
		{Methods: []string{"oidc", "password"}, MFA: "none", MaxAge: time.Hour},
		{Methods: []string{"password"}, MFA: "none", MaxAge: time.Hour, StepUp: []Rule{{Action: "policy.change", MaxAge: time.Minute}}},
		{Methods: []string{"password"}, MFA: "none", MaxAge: time.Hour, StepUp: []Rule{{Action: "policy.change", MaxAge: time.Minute, MFA: true}}},
		{Methods: []string{"password"}, MFA: "none", MaxAge: time.Hour, StepUp: []Rule{{Action: "policy.change", MaxAge: 2 * time.Minute}}},
		{Methods: []string{"password"}, MFA: "none", MaxAge: time.Hour, StepUp: []Rule{{Action: "other", MaxAge: time.Minute}}},
	}
	seen := map[string]bool{ConfigDigest(c): true}
	for _, v := range variants {
		if seen[ConfigDigest(v)] {
			t.Errorf("collision %+v", v)
		}
		seen[ConfigDigest(v)] = true
	}
	v := View{Workspace: "w", Route: "r", ID: "i", Revision: 1, OldEpoch: 1, OldPolicy: 1, OldDigest: "o", NewDigest: "n", Proposer: "p"}
	d := changeDigest(v)
	for name, change := range map[string]func(*View){
		"workspace": func(x *View) { x.Workspace = "w2" }, "route": func(x *View) { x.Route = "r2" }, "id": func(x *View) { x.ID = "i2" },
		"revision": func(x *View) { x.Revision = 2 }, "old epoch": func(x *View) { x.OldEpoch = 2 }, "old policy": func(x *View) { x.OldPolicy = 2 },
		"old digest": func(x *View) { x.OldDigest = "o2" }, "new digest": func(x *View) { x.NewDigest = "n2" }, "proposer": func(x *View) { x.Proposer = "p2" },
	} {
		x := v
		change(&x)
		if changeDigest(x) == d {
			t.Errorf("%s does not change the change digest", name)
		}
	}
	if changeDigest(v) != d {
		t.Error("equal inputs")
	}
	if EvidenceDigest("c", []string{"a", "b"}) != EvidenceDigest("c", []string{"b", "a"}) || EvidenceDigest("c", []string{"a"}) == EvidenceDigest("d", []string{"a"}) ||
		EvidenceDigest("c", []string{"a"}) == EvidenceDigest("c", []string{"a", ""}) {
		t.Error("evidence digest")
	}
}

func TestProposeAndRevise(t *testing.T) {
	f := setup(t)
	v := f.propose("ann", 60)
	if v.ID != "trust-0a1b2c3d-1" || v.State != StateDraft || v.Revision != 1 || v.OldEpoch != 1 || v.OldDigest != base0.Digest || v.NewDigest != ConfigDigest(candidate(60)) ||
		v.ChangeDigest != changeDigest(v) || len(v.History) != 1 || v.History[0].Rationale != "why" || v.Run != "0a1b2c3d" {
		t.Fatalf("proposed %+v", v)
	}
	p := Proposal{Workspace: "w1", Route: RouteLocalPassword, Proposer: "ann", Candidate: candidate(60), Base: base0}
	refused(t, "duplicate", second(f.r.Propose(p)), ReasonDuplicate)
	p.Proposer = "bob"
	if _, err := f.r.Propose(p); err != nil {
		t.Errorf("same candidate from another proposer %v", err)
	}
	p.Candidate = candidate(720)
	refused(t, "unchanged", second(f.r.Propose(p)), ReasonUnchanged)
	for name, change := range map[string]func(*Proposal){
		"no methods": func(x *Proposal) { x.Candidate.Methods = nil }, "unsorted": func(x *Proposal) { x.Candidate.Methods = []string{"password", "oidc"} },
		"empty method": func(x *Proposal) { x.Candidate.Methods = []string{""} }, "no mfa": func(x *Proposal) { x.Candidate.MFA = "" },
		"repeated method": func(x *Proposal) { x.Candidate.Methods = []string{"password", "password"} },
		"unnamed rule":    func(x *Proposal) { x.Candidate.StepUp = []Rule{{MaxAge: time.Minute}} },
		"no age":          func(x *Proposal) { x.Candidate.MaxAge = 0 }, "bad rule": func(x *Proposal) { x.Candidate.StepUp = []Rule{{Action: "x"}} },
		"no workspace": func(x *Proposal) { x.Workspace = "" }, "no route": func(x *Proposal) { x.Route = "" }, "no proposer": func(x *Proposal) { x.Proposer = "" },
	} {
		x := Proposal{Workspace: "w1", Route: RouteLocalPassword, Proposer: "cat", Candidate: candidate(30), Base: base0}
		change(&x)
		refused(t, name, second(f.r.Propose(x)), ReasonInvalid)
	}
	refused(t, "revise by another", second(f.r.Revise(v.ID, "w1", "bob", 1, candidate(45), base0, "")), ReasonNotProposer)
	refused(t, "revise unknown", second(f.r.Revise("trust-0a1b2c3d-99", "w1", "ann", 1, candidate(45), base0, "")), ReasonNotFound)
	refused(t, "revise in another workspace", second(f.r.Revise(v.ID, "w2", "ann", 1, candidate(45), base0, "")), ReasonNotFound)
	refused(t, "revise stale", second(f.r.Revise(v.ID, "w1", "ann", 2, candidate(45), base0, "")), ReasonApprovalStale)
	refused(t, "revise old epoch", second(f.r.Revise(v.ID, "w1", "ann", 1, candidate(45), Base{Epoch: 2, Policy: 2, Digest: base0.Digest}, "")), ReasonEpoch)
	refused(t, "revise invalid", second(f.r.Revise(v.ID, "w1", "ann", 1, Candidate{}, base0, "")), ReasonInvalid)
	refused(t, "revise to current", second(f.r.Revise(v.ID, "w1", "ann", 1, candidate(720), base0, "")), ReasonUnchanged)
	refused(t, "revise to same", second(f.r.Revise(v.ID, "w1", "ann", 1, candidate(60), base0, "")), ReasonDuplicate)
	approved := f.approved("bob", f.validated("cat", v))
	revised, err := f.r.Revise(v.ID, "w1", "ann", 1, candidate(45), base0, "revised")
	if err != nil || revised.State != StateDraft || revised.Revision != 2 || revised.Approver != "" || revised.Validation != nil || revised.Validator != "" ||
		revised.ChangeDigest == approved.ChangeDigest || revised.NewDigest != ConfigDigest(candidate(45)) {
		t.Fatalf("revised %+v %v", revised, err)
	}
	refused(t, "approve the old digest", second(f.r.Approve(approval(approved, "bob"))), ReasonStage)
	again := f.validated("cat", revised)
	stale := approval(approved, "bob")
	stale.EvidenceDigest = again.Validation.Digest
	stale.Recomputed = again.Validation.Digest
	refused(t, "an old change digest", second(f.r.Approve(stale)), ReasonApprovalStale)
	got, _ := f.r.Get(v.ID, "w1")
	got.Candidate.Methods[0] = "zzz"
	got.History[0].By = "zzz"
	if back, _ := f.r.Get(v.ID, "w1"); back.Candidate.Methods[0] != "password" || back.History[0].By != "ann" {
		t.Error("views share memory")
	}
	if _, ok := f.r.Get(v.ID, "w2"); ok {
		t.Error("another workspace")
	}
	if list := f.r.List("w1"); len(list) != 2 || list[0].Number != 2 || len(f.r.List("w2")) != 0 {
		t.Errorf("list %+v", list)
	}
}

func TestValidateApproveRejectCancel(t *testing.T) {
	f := setup(t)
	v := f.propose("ann", 60)
	refused(t, "approve a draft", second(f.r.Approve(Approval{ID: v.ID, Workspace: "w1", Actor: "bob", Base: base0})), ReasonStage)
	refused(t, "validate stale revision", second(f.r.Validate(v.ID, "w1", "ann", 2, v.ChangeDigest, base0, evidence("e"), "")), ReasonApprovalStale)
	refused(t, "validate stale digest", second(f.r.Validate(v.ID, "w1", "ann", 1, "x", base0, evidence("e"), "")), ReasonApprovalStale)
	refused(t, "validate old epoch", second(f.r.Validate(v.ID, "w1", "ann", 1, v.ChangeDigest, Base{Epoch: 2}, evidence("e"), "")), ReasonEpoch)
	refused(t, "validate without evidence", second(f.r.Validate(v.ID, "w1", "ann", 1, v.ChangeDigest, base0, Evidence{}, "")), ReasonInvalid)
	refused(t, "validate unknown", second(f.r.Validate("trust-x", "w1", "ann", 1, v.ChangeDigest, base0, evidence("e"), "")), ReasonNotFound)
	blocked := evidence("b")
	blocked.Blocked = true
	b, err := f.r.Validate(v.ID, "w1", "ann", 1, v.ChangeDigest, base0, blocked, "")
	if err != nil || b.State != StateDraft || b.Validation.Result != ResultBlocked || b.Validator != "ann" {
		t.Fatalf("blocked %+v %v", b, err)
	}
	a := approval(b, "bob")
	refused(t, "approve a blocked draft", second(f.r.Approve(a)), ReasonStage)
	val := f.validated("ann", v)
	if val.State != StateValidated || val.Validation.Result != ResultValidated || val.Validation.Items[0].Item != "test_login" {
		t.Fatalf("validated %+v", val)
	}
	refused(t, "self approval", second(f.r.Approve(approval(val, "ann"))), ReasonSelf)
	wrong := approval(val, "bob")
	wrong.Base = Base{Epoch: 9}
	refused(t, "approve old epoch", second(f.r.Approve(wrong)), ReasonEpoch)
	wrong = approval(val, "bob")
	wrong.ChangeDigest = "x"
	refused(t, "approve stale change", second(f.r.Approve(wrong)), ReasonApprovalStale)
	wrong = approval(val, "bob")
	wrong.EvidenceDigest = "x"
	refused(t, "approve other evidence", second(f.r.Approve(wrong)), ReasonEvidenceStale)
	wrong = approval(val, "bob")
	wrong.Recomputed = "x"
	refused(t, "approve changed evidence", second(f.r.Approve(wrong)), ReasonEvidenceStale)
	app := f.approved("bob", val)
	if app.State != StateApproved || app.Approver != "bob" || !app.ApprovedAt.Equal(start) {
		t.Fatalf("approved %+v", app)
	}
	refused(t, "approve twice", second(f.r.Approve(approval(app, "cat"))), ReasonStage)
	*f.now = start.Add(time.Minute)
	re, err := f.r.Validate(v.ID, "w1", "cat", 1, v.ChangeDigest, base0, evidence("changed"), "again")
	if err != nil || re.State != StateValidated || re.Approver != "" || re.Validation.Digest == app.Validation.Digest || re.History[len(re.History)-2].Rationale != HistoryInvalidated {
		t.Fatalf("revalidated %+v %v", re, err)
	}
	refused(t, "reject by proposer", second(f.r.Reject(v.ID, "w1", "ann", 1, "")), ReasonSelf)
	refused(t, "cancel by another", second(f.r.Cancel(v.ID, "w1", "bob", 1, "")), ReasonNotProposer)
	refused(t, "reject stale", second(f.r.Reject(v.ID, "w1", "bob", 2, "")), ReasonApprovalStale)
	refused(t, "reject unknown", second(f.r.Reject("x", "w1", "bob", 1, "")), ReasonNotFound)
	rj, err := f.r.Reject(v.ID, "w1", "bob", 1, "not now")
	if err != nil || rj.State != StateRejected || rj.History[len(rj.History)-1].Rationale != "not now" {
		t.Fatalf("rejected %+v", rj)
	}
	for name, call := range map[string]func() error{
		"revise": func() error { return second(f.r.Revise(v.ID, "w1", "ann", 1, candidate(5), base0, "")) },
		"validate": func() error {
			return second(f.r.Validate(v.ID, "w1", "ann", 1, v.ChangeDigest, base0, evidence("e"), ""))
		},
		"reject":   func() error { return second(f.r.Reject(v.ID, "w1", "bob", 1, "")) },
		"cancel":   func() error { return second(f.r.Cancel(v.ID, "w1", "ann", 1, "")) },
		"approve":  func() error { return second(f.r.Approve(approval(app, "bob"))) },
		"activate": func() error { return second(f.r.Activate(approval(app, "cat"), receipt(2))) },
	} {
		refused(t, "terminal "+name, call(), ReasonStage)
	}
	w := f.propose("ann", 61)
	c, err := f.r.Cancel(w.ID, "w1", "ann", 1, "withdrawn")
	if err != nil || c.State != StateCancelled {
		t.Fatalf("cancelled %+v", c)
	}
}

func TestBoundsAndHistory(t *testing.T) {
	f := setup(t)
	for n := 0; n < MaxOpenPerProposer; n++ {
		f.propose("ann", n+1)
	}
	refused(t, "per proposer", second(f.r.Propose(Proposal{Workspace: "w1", Route: "r", Proposer: "ann", Candidate: candidate(100), Base: base0})), ReasonLimit)
	for _, who := range []string{"bob", "cat", "dan"} {
		for n := 0; n < MaxOpenPerProposer; n++ {
			f.propose(who, n+1)
		}
	}
	refused(t, "per workspace", second(f.r.Propose(Proposal{Workspace: "w1", Route: "r", Proposer: "eve", Candidate: candidate(100), Base: base0})), ReasonLimit)
	if _, err := f.r.Propose(Proposal{Workspace: "w2", Route: "r", Proposer: "eve", Candidate: candidate(100), Base: base0}); err != nil {
		t.Errorf("another workspace %v", err)
	}
	for _, v := range f.r.List("w1") {
		f.r.Cancel(v.ID, "w1", v.Proposer, 1, "")
	}
	for n := 0; n < MaxKept+10; n++ {
		v := f.propose("ann", 200+n)
		f.r.Cancel(v.ID, "w1", "ann", 1, "")
	}
	if len(f.r.order) != MaxKept || len(f.r.changes) != MaxKept {
		t.Errorf("kept %d", len(f.r.order))
	}
	g := setup(t)
	for n := 0; n < MaxKept+5; n++ {
		g.r.Propose(Proposal{Workspace: "w" + strings.Repeat("x", n), Route: "r", Proposer: "p", Candidate: candidate(9), Base: base0})
	}
	if len(g.r.order) != MaxKept+5 {
		t.Errorf("open changes are never pruned: %d", len(g.r.order))
	}
	h := setup(t)
	v := h.propose("ann", 60)
	for n := 0; n < MaxHistory+5; n++ {
		v = h.validated("bob", v)
	}
	if len(v.History) != MaxHistory || v.History[MaxHistory-1].State != StateValidated {
		t.Errorf("history %d", len(v.History))
	}
}

func TestRestorePrunesInOneCall(t *testing.T) {
	f := setup(t)
	views := []View{}
	for n := 1; n <= MaxKept+5; n++ {
		views = append(views, View{ID: "trust-11111111-" + itoa(n), Workspace: "w1", State: StateActivated})
	}
	if err := f.r.Restore(views); err != nil || len(f.r.order) != MaxKept {
		t.Errorf("restored %d %v", len(f.r.order), err)
	}
}

func itoa(n int) string {
	out := ""
	for ; n > 0; n /= 10 {
		out = string(rune('0'+n%10)) + out
	}
	return out
}

func TestRestoreEarlierRuns(t *testing.T) {
	f := setup(t)
	earlier := View{ID: "trust-11111111-3", Workspace: "w1", State: StateActivated, Activation: &Activation{Epoch: 2}}
	if err := f.r.Restore([]View{earlier}); err != nil {
		t.Fatal(err)
	}
	got, ok := f.r.Get(earlier.ID, "w1")
	if !ok || !got.EarlierRun || got.Run != "11111111" {
		t.Errorf("restored %+v", got)
	}
	for name, v := range map[string]View{
		"the same run":   {ID: "trust-0a1b2c3d-1", State: StateActivated},
		"a repeated id":  earlier,
		"a malformed id": {ID: "x", State: StateActivated},
		"not activated":  {ID: "trust-22222222-1", State: StateDraft},
	} {
		if err := f.r.Restore([]View{v}); err != ErrInvalid {
			t.Errorf("%s: %v", name, err)
		}
	}
}
