package authpolicy

import (
	"errors"
	"strings"
	"testing"
)

func TestRevocation(t *testing.T) {
	f := standard(t)
	users := []string{"u1", "u2", "u3", "u4", "u5"}
	for _, u := range users {
		f.must(second(f.r.Evaluate(member(u), f.weak(), "")))
	}
	id := f.block(users, 2)
	if id != "w1/1" {
		t.Fatalf("id %s", id)
	}
	for _, u := range users {
		reason(t, "blocked before any chunk "+u, second(f.r.Evaluate(member(u), f.strong(), "")), ReasonBlocked)
	}
	f.must(second(f.r.Evaluate(member("u6"), f.weak(), "")))
	v, err := f.r.Plan(id)
	if err != nil || v.State != PlanRunning || v.Chunks != 3 || v.Completed != 0 || v.Next != 0 || v.Proposer != "cat" || v.Approver != "dan" || v.Plan.ID != id {
		t.Fatalf("plan %+v %v", v, err)
	}
	for index, want := range []string{"u1 u2", "u3 u4", "u5"} {
		got, err := f.r.Chunk(id, index)
		if err != nil || strings.Join(got, " ") != want {
			t.Errorf("chunk %d %v", index, got)
		}
	}
	got, _ := f.r.Chunk(id, 0)
	got[0] = "zz"
	if again, _ := f.r.Chunk(id, 0); again[0] != "u1" {
		t.Error("chunk shares memory")
	}
	expect(t, "chunk range", second(f.r.Chunk(id, 3)), ErrInvalid)
	expect(t, "chunk negative", second(f.r.Chunk(id, -1)), ErrInvalid)
	expect(t, "chunk missing", second(f.r.Chunk("w1/9", 0)), ErrNotFound)
	reason(t, "human worker", second(f.r.Complete(id, 0, ann)), ReasonNotSystem)
	reason(t, "assistant worker", second(f.r.Complete(id, 0, Principal{User: "x", Kind: KindAssistant})), ReasonAINotAllowed)
	expect(t, "worker form", second(f.r.Complete(id, 0, Principal{Kind: KindSystem})), ErrInvalid)
	expect(t, "worker kind", second(f.r.Complete(id, 0, Principal{User: "x"})), ErrInvalid)
	expect(t, "complete missing", second(f.r.Complete("w1/9", 0, worker)), ErrNotFound)
	expect(t, "complete range", second(f.r.Complete(id, 3, worker)), ErrInvalid)
	expect(t, "complete negative", second(f.r.Complete(id, -1, worker)), ErrInvalid)
	reason(t, "out of order", second(f.r.Complete(id, 1, worker)), ReasonOutOfOrder)
	if v, err := f.r.Complete(id, 0, worker); err != nil || v.Completed != 1 || v.Next != 1 {
		t.Errorf("first %+v %v", v, err)
	}
	if v, err := f.r.Complete(id, 0, worker); err != nil || v.Completed != 1 {
		t.Error("idempotent")
	}
	f.must(second(f.r.Complete(id, 1, worker)))
	if v, _ := f.r.Complete(id, 2, worker); v.State != PlanComplete || v.Next != 3 {
		t.Errorf("complete %+v", v)
	}
	reason(t, "complete keeps block", second(f.r.Evaluate(member("u5"), f.strong(), "")), ReasonBlocked)
	next := f.block([]string{"u1", "u9"}, 1000)
	if next != "w1/2" {
		t.Errorf("second id %s", next)
	}
	reason(t, "lift in scope proposer", f.tryLift(id, admin("u1"), bob), ReasonBlocked)
	f.must(second(f.r.Propose(Change{Kind: ChangeLift, Lift: Lift{Plan: id, Reason: "x"}}, ann, f.strong())))
	if f.r.blocked[[2]string{"w1", "u1"}] != 2 || f.r.usage["w1"] != 7 {
		t.Errorf("counts %d %d", f.r.blocked[[2]string{"w1", "u1"}], f.r.usage["w1"])
	}
	f.by(Change{Kind: ChangeLift, Lift: Lift{Plan: id, Reason: "restored"}}, ann, bob)
	reason(t, "still in other plan", second(f.r.Evaluate(member("u1"), f.strong(), "")), ReasonBlocked)
	f.must(second(f.r.Evaluate(member("u2"), f.weak(), "")))
	if v, _ := f.r.Plan(id); v.State != PlanLifted || v.Lifted != [2]string{"ann", "bob"} || v.Completed != 3 {
		t.Errorf("lifted %+v", v)
	}
	if f.r.usage["w1"] != 2 || f.r.entries != 2 || f.r.workspaces["w1"].active != 1 {
		t.Errorf("released %d %d", f.r.usage["w1"], f.r.entries)
	}
	reason(t, "lift twice", f.try(Change{Kind: ChangeLift, Lift: Lift{Plan: id, Reason: "x"}}), ReasonLifted)
	reason(t, "complete lifted", second(f.r.Complete(id, 0, worker)), ReasonLifted)
	if got, err := f.r.Chunk(id, 2); err != nil || len(got) != 1 {
		t.Error("chunk of lifted")
	}
	g := standard(t)
	gid := g.block([]string{"bob", "zed"}, 1)
	reason(t, "blocked approver", second(g.tryBy(Change{Kind: ChangeLift, Lift: Lift{Plan: gid, Reason: "x"}}, ann, bob, g.strong(), g.strong())), ReasonBlocked)
	p, _ := g.r.Propose(Change{Kind: ChangeLift, Lift: Lift{Plan: gid, Reason: "x"}}, ann, g.strong())
	g.r.blocked = map[[2]string]int{}
	reason(t, "approver in scope", second(g.r.Approve(p.Number, bob, g.strong())), ReasonInScope)
	reason(t, "proposer in scope", second(g.r.Propose(Change{Kind: ChangeLift, Lift: Lift{Plan: gid, Reason: "y"}}, admin("zed"), g.strong())), ReasonInScope)
	expect(t, "lift missing", g.try(Change{Kind: ChangeLift, Lift: Lift{Plan: "w1/99", Reason: "x"}}), ErrNotFound)
	expect(t, "plan missing", second(g.r.Plan("w1/9")), ErrNotFound)
	h := standard(t)
	h.block([]string{"cat"}, 1)
	reason(t, "blocked admin cannot act", second(h.r.Propose(Change{Kind: ChangeRevision, Revision: open()}, cat, h.strong())), ReasonBlocked)
	if !errors.Is(Refusal{Reason: "x"}, ErrRejected) || (Refusal{Reason: "x"}).Error() != ErrRejected.Error()+": x" {
		t.Error("refusal")
	}
}

func (f world) tryLift(id string, proposer, approver Principal) error {
	_, err := f.tryBy(Change{Kind: ChangeLift, Lift: Lift{Plan: id, Reason: "x"}}, proposer, approver, f.strong(), f.strong())
	return err
}

func TestLiftedKept(t *testing.T) {
	f := standard(t)
	first := f.block([]string{"a"}, 1)
	f.by(Change{Kind: ChangeLift, Lift: Lift{Plan: first, Reason: "x"}}, ann, bob)
	for n := 2; n < 1000; n++ {
		f.r.lifted = append(f.r.lifted, "w1/x"+itoa(n))
	}
	keep := f.block([]string{"b"}, 1)
	f.by(Change{Kind: ChangeLift, Lift: Lift{Plan: keep, Reason: "x"}}, ann, bob)
	if _, err := f.r.Plan(first); err != nil || len(f.r.lifted) != 1000 {
		t.Fatal("evicted early")
	}
	more := f.block([]string{"d"}, 1)
	f.by(Change{Kind: ChangeLift, Lift: Lift{Plan: more, Reason: "x"}}, ann, bob)
	expect(t, "evicted", second(f.r.Plan(first)), ErrNotFound)
	if _, err := f.r.Plan(keep); err != nil || len(f.r.lifted) != 1000 {
		t.Error("kept")
	}
	if again := f.block([]string{"c"}, 1); again != "w1/4" {
		t.Errorf("never reused %s", again)
	}
}
