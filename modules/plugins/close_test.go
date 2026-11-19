package plugins

import "testing"

func TestCloseAPendingProposal(t *testing.T) {
	f := setup(t)
	c := Change{Kind: ChangePublisher, Publisher: Publisher{ID: "acme", Name: "Acme"}}
	p, err := f.r.Propose(c, ann)
	f.must(err)
	expect(t, "close for another reason", second(f.r.Close(p.Number, ReasonOperator)), ErrInvalid)
	expect(t, "close a missing proposal", second(f.r.Close(99, ReasonRequesterLostRight)), ErrNotFound)
	closed, err := f.r.Close(p.Number, ReasonRequesterLostRight)
	if err != nil || closed.State != ProposalClosed || closed.Proposer != "ann" {
		t.Fatalf("closed %+v %v", closed, err)
	}
	expect(t, "approve a closed proposal", second(f.r.Approve(p.Number, bob)), ErrNotFound)
	expect(t, "close twice", second(f.r.Close(p.Number, ReasonRequesterLostRight)), ErrNotFound)
	again, err := f.r.Propose(c, ann)
	f.must(err)
	if again.Number == p.Number || again.State != ProposalPending {
		t.Errorf("the slot was not freed %+v", again)
	}
	done, err := f.r.Approve(again.Number, bob)
	f.must(err)
	reason(t, "close an applied proposal", second(f.r.Close(done.Number, ReasonRequesterLostRight)), ReasonNotPending)
}

func TestClosingFreesTheProposerShare(t *testing.T) {
	l := limits
	l.Proposals = 4
	f := setupWith(t, l)
	c := Change{Kind: ChangePublisher, Publisher: Publisher{ID: "acme", Name: "Acme"}}
	p, err := f.r.Propose(c, ann)
	f.must(err)
	other := Change{Kind: ChangePublisher, Publisher: Publisher{ID: "beta", Name: "Beta"}}
	expect(t, "over the proposer share", second(f.r.Propose(other, ann)), ErrLimit)
	f.must(second(f.r.Close(p.Number, ReasonRequesterLostRight)))
	f.must(second(f.r.Propose(other, ann)))
}
