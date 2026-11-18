package trustchange

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestActivation(t *testing.T) {
	f := setup(t)
	v := f.propose("ann", 60)
	other := f.propose("bob", 30)
	elsewhere, _ := f.r.Propose(Proposal{Workspace: "w2", Route: "r", Proposer: "bob", Candidate: candidate(30), Base: base0})
	older, _ := f.r.Propose(Proposal{Workspace: "w1", Route: "r", Proposer: "cat", Candidate: candidate(31), Base: Base{Epoch: 0, Policy: 0, Digest: EmptyDigest()}})
	calls := 0
	counting := func(out Outcome) func(View) Outcome {
		return func(intent View) Outcome {
			calls++
			if intent.Activator != "cat" || intent.State != StateApproved {
				t.Errorf("intent %+v", intent)
			}
			return out
		}
	}
	refused(t, "activate a draft", second(f.r.Activate(Approval{ID: v.ID, Workspace: "w1", Actor: "cat", Base: base0}, counting(Outcome{}))), ReasonNeedsApproval)
	val := f.validated("dan", v)
	refused(t, "activate validated", second(f.r.Activate(approval(val, "cat"), counting(Outcome{}))), ReasonNeedsApproval)
	app := f.approved("bob", val)
	refused(t, "the proposer activates", second(f.r.Activate(approval(app, "ann"), counting(Outcome{}))), ReasonSelf)
	refused(t, "the approver activates", second(f.r.Activate(approval(app, "bob"), counting(Outcome{}))), ReasonSelf)
	refused(t, "unknown", second(f.r.Activate(Approval{ID: "x", Workspace: "w1"}, counting(Outcome{}))), ReasonNotFound)
	stale := approval(app, "cat")
	stale.Recomputed = "changed"
	refused(t, "recomputed evidence", second(f.r.Activate(stale, counting(Outcome{}))), ReasonEvidenceStale)
	stale = approval(app, "cat")
	stale.Base.Epoch = 5
	refused(t, "old epoch", second(f.r.Activate(stale, counting(Outcome{}))), ReasonEpoch)
	if calls != 0 {
		t.Fatalf("commit called %d times before the checks passed", calls)
	}
	failed := errors.New("ledger refused")
	if _, err := f.r.Activate(approval(app, "cat"), counting(Outcome{Err: failed})); err != failed || calls != 1 {
		t.Fatalf("a refused commit %v", err)
	}
	if got, _ := f.r.Get(v.ID, "w1"); got.State != StateApproved || got.Activator != "" {
		t.Fatalf("a refused commit changed the change %+v", got)
	}
	done, err := f.r.Activate(approval(app, "cat"), counting(Outcome{Activation: Activation{Epoch: 2, Sequence: 9, Watermark: "9:x"}}))
	if err != nil || calls != 2 || done.State != StateActivated || done.Activator != "cat" || done.Activation.Epoch != 2 || done.Activation.Sequence != 9 {
		t.Fatalf("activated %+v %v", done, err)
	}
	if got, _ := f.r.Get(other.ID, "w1"); got.State != StateSuperseded || got.SupersededBy != v.ID || got.SupersededEpoch != 2 {
		t.Errorf("superseded %+v", got)
	}
	if got, _ := f.r.Get(elsewhere.ID, "w2"); got.State != StateDraft {
		t.Errorf("another workspace was superseded %+v", got)
	}
	if got, _ := f.r.Get(older.ID, "w1"); got.State != StateDraft {
		t.Errorf("a change on another epoch was superseded %+v", got)
	}
	refused(t, "activate twice", second(f.r.Activate(approval(app, "cat"), counting(Outcome{}))), ReasonStage)
	g := setup(t)
	u := g.approved("bob", g.validated("dan", g.propose("ann", 60)))
	sibling := g.propose("bob", 30)
	unknown, err := g.r.Activate(approval(u, "cat"), receiptUnknown("commit_unknown"))
	if err != nil || unknown.State != StateUnknown || unknown.Unknown != "commit_unknown" || unknown.Activation != nil {
		t.Fatalf("unknown %+v %v", unknown, err)
	}
	if got, _ := g.r.Get(sibling.ID, "w1"); got.State != StateDraft {
		t.Errorf("an unknown activation superseded %+v", got)
	}
	refused(t, "an unknown activation is terminal", second(g.r.Activate(approval(u, "cat"), receipt(3))), ReasonStage)
}

func receiptUnknown(reason string) func(View) Outcome {
	return func(View) Outcome { return Outcome{Unknown: reason} }
}

func TestRevalidationWithUnchangedEvidenceKeepsTheApproval(t *testing.T) {
	f := setup(t)
	app := f.approved("bob", f.validated("cat", f.propose("ann", 60)))
	*f.now = start.Add(time.Minute)
	same, err := f.r.Validate(app.ID, "w1", "dan", app.Revision, app.ChangeDigest, base0, evidence("e"), "again")
	if err != nil || same.State != StateApproved || same.Approver != "bob" || same.Validator != "dan" || !same.Validation.At.Equal(start.Add(time.Minute)) {
		t.Fatalf("same evidence %+v %v", same, err)
	}
	last := same.History[len(same.History)-1]
	if last.State != HistoryKept || last.Rationale != "again" {
		t.Errorf("history %+v", same.History)
	}
	for _, e := range same.History {
		if e.Rationale == HistoryInvalidated {
			t.Error("an unchanged digest invalidated the approval")
		}
	}
	if _, err := f.r.Activate(approval(same, "eve"), receipt(2)); err != nil {
		t.Errorf("the kept approval activates %v", err)
	}
	g := setup(t)
	kept := g.approved("bob", g.validated("cat", g.propose("ann", 60)))
	blocked := evidence("e")
	blocked.Blocked = true
	b, _ := g.r.Validate(kept.ID, "w1", "dan", kept.Revision, kept.ChangeDigest, base0, blocked, "")
	if b.State != StateDraft || b.Approver != "" || b.History[len(b.History)-2].Rationale != HistoryInvalidated {
		t.Errorf("a blocked result with the same digest clears the approval %+v", b)
	}
}

func TestConcurrentActivationsOnOneEpoch(t *testing.T) {
	for round := 0; round < 50; round++ {
		f := setup(t)
		one := f.approved("bob", f.validated("cat", f.propose("ann", 60)))
		two := f.approved("bob", f.validated("cat", f.propose("ann", 61)))
		var wg sync.WaitGroup
		errs := make([]error, 2)
		for i, v := range []View{one, two} {
			wg.Add(1)
			go func(i int, v View) {
				defer wg.Done()
				_, errs[i] = f.r.Activate(approval(v, "dan"), receipt(2))
			}(i, v)
		}
		wg.Wait()
		ok, stage := 0, 0
		for _, err := range errs {
			var r Refusal
			switch {
			case err == nil:
				ok++
			case errors.As(err, &r) && r.Reason == ReasonStage:
				stage++
			}
		}
		if ok != 1 || stage != 1 {
			t.Fatalf("round %d: %v", round, errs)
		}
	}
}
