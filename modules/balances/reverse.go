package balances

import (
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/exact"
)

func viewReversal(rev *reversal) ReversalView {
	return ReversalView{Reversal: rev.r, State: rev.state, Author: rev.author, Approver: rev.approver, Digest: rev.digest, WrittenAt: rev.writtenAt, ApprovedAt: rev.approvedAt}
}

type reversed struct {
	app  *application
	lot  *lot
	acct *account
	list *[]*reversal
	ok   bool
}

func (s *space) reversedRecord(rev Reversal) reversed {
	switch rev.Kind {
	case ReverseApplication:
		if app := s.applications[rev.Record]; app != nil {
			return reversed{app: app, acct: app.acct, list: &app.reversals, ok: true}
		}
	case ReverseIncrease:
		if i := s.increases[rev.Record]; i != nil {
			return reversed{lot: i.lot, acct: i.acct, list: &i.reversals, ok: i.state == StateApproved}
		}
	case ReverseOpening:
		if o := s.openings[rev.Record]; o != nil {
			return reversed{lot: o.lot, acct: o.acct, list: &o.reversals, ok: o.state == StateApproved}
		}
	}
	return reversed{}
}

func (s *space) exists(rev Reversal) bool {
	switch rev.Kind {
	case ReverseApplication:
		return s.applications[rev.Record] != nil
	case ReverseIncrease:
		return s.increases[rev.Record] != nil
	}
	return s.openings[rev.Record] != nil
}

func (r *Registry) reversible(target reversed, amount exact.Fraction, now time.Time) bool {
	if !target.ok {
		return false
	}
	if target.app != nil {
		app := target.app
		return stateOf(app, now) == StateConfirmed && !app.applied.Minus(r.reversedOf(app)).Less(amount)
	}
	l := target.lot
	return l != nil && !l.expired(now) && !l.remainder.Minus(r.held(target.acct, l, now)).Less(amount)
}

func (r *Registry) Reverse(workspace string, rev Reversal, author string) (ReversalView, error) {
	amount, formed := r.amount(rev.Amount, true)
	if !validID(workspace) || !validID(author) || !formed || !validID(rev.ID) || !reverseKinds[rev.Kind] || !validID(rev.Record) {
		return ReversalView{}, ErrInvalid
	}
	rev.Amount = amount
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || !s.exists(rev) {
		return ReversalView{}, ErrNotFound
	}
	if existing := s.reversals[rev.ID]; existing != nil {
		if existing.r != rev {
			return ReversalView{}, ErrConflict
		}
		return viewReversal(existing), nil
	}
	target := s.reversedRecord(rev)
	if !r.reversible(target, r.value(amount), now) {
		return ReversalView{}, ErrRejected
	}
	if len(*target.list) >= r.limits.Reversals || len(s.reversals) >= r.limits.Workspace {
		return ReversalView{}, ErrLimit
	}
	stored := &reversal{r: rev, state: StatePending, author: author, writtenAt: now, digest: reversalDigest(workspace, rev)}
	*target.list = append(*target.list, stored)
	s.reversals[rev.ID] = stored
	return viewReversal(stored), nil
}

func (r *Registry) ApproveReversal(workspace, id, approver string) (ReversalView, error) {
	if !validID(workspace) || !validID(id) || !validID(approver) {
		return ReversalView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.reversals[id] == nil {
		return ReversalView{}, ErrNotFound
	}
	rev := s.reversals[id]
	if rev.state == StateEffective {
		if rev.approver != approver {
			return ReversalView{}, ErrConflict
		}
		return viewReversal(rev), nil
	}
	amount := r.value(rev.r.Amount)
	target := s.reversedRecord(rev.r)
	if approver == rev.author || !r.reversible(target, amount, now) {
		return ReversalView{}, ErrRejected
	}
	if target.app != nil {
		left := amount
		for index := len(target.app.holds) - 1; index >= 0 && left.Sign() > 0; index-- {
			h := target.app.holds[index]
			back := least(h.amount.Minus(h.returned), left)
			h.returned = h.returned.Plus(back)
			h.lot.remainder = h.lot.remainder.Plus(back)
			left = left.Minus(back)
		}
	} else {
		target.lot.remainder = target.lot.remainder.Minus(amount)
	}
	rev.state, rev.approver, rev.approvedAt = StateEffective, approver, now
	return viewReversal(rev), nil
}

func (r *Registry) Reversal(workspace, id string) (ReversalView, error) {
	if !validID(workspace) || !validID(id) {
		return ReversalView{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.reversals[id] == nil {
		return ReversalView{}, ErrNotFound
	}
	return viewReversal(s.reversals[id]), nil
}
