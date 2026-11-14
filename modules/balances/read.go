package balances

import "github.com/myblueton-sw/hybrid_billing/packages/primitives/exact"

func text(f exact.Fraction) string {
	out, _ := format(f)
	return out
}

func (r *Registry) Lots(workspace, id string) ([]LotView, error) {
	if !validID(workspace) || !validID(id) {
		return nil, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	acct := r.lookup(workspace, id)
	if acct == nil {
		return nil, ErrNotFound
	}
	out := []LotView{}
	for _, l := range acct.lots {
		v := LotView{Number: l.number, Source: l.source, Record: l.record, Amount: text(l.amount), Remainder: "0", Held: "0", Available: "0", Expired: "0", Expiry: l.expiry}
		if l.expired(now) {
			v.Expired = text(l.remainder)
		} else {
			held := r.held(acct, l, now)
			v.Remainder, v.Held, v.Available = text(l.remainder), text(held), text(l.remainder.Minus(held))
		}
		out = append(out, v)
	}
	return out, nil
}

func (r *Registry) effective(list []*reversal) exact.Fraction {
	total := exact.Whole(0)
	for _, rev := range list {
		if rev.state == StateEffective {
			total = total.Plus(r.value(rev.r.Amount))
		}
	}
	return total
}

func (r *Registry) Statement(workspace, id string) (Statement, error) {
	if !validID(workspace) || !validID(id) {
		return Statement{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	acct := r.lookup(workspace, id)
	if acct == nil {
		return Statement{}, ErrNotFound
	}
	if acct.a.Authority == AuthorityExternal {
		return Statement{}, ErrRejected
	}
	opening, increases, applied, expired := exact.Whole(0), exact.Whole(0), exact.Whole(0), exact.Whole(0)
	appReversed, lotReversed, closing, held := exact.Whole(0), exact.Whole(0), exact.Whole(0), exact.Whole(0)
	for _, o := range acct.openings {
		if o.state == StateApproved {
			opening = opening.Plus(r.value(o.o.Amount))
			lotReversed = lotReversed.Plus(r.effective(o.reversals))
		}
	}
	for _, i := range acct.increases {
		if i.state == StateApproved {
			increases = increases.Plus(r.value(i.i.Amount))
			lotReversed = lotReversed.Plus(r.effective(i.reversals))
		}
	}
	for _, app := range acct.apps {
		if app.state == StateConfirmed {
			applied = applied.Plus(app.applied)
			appReversed = appReversed.Plus(r.reversedOf(app))
		}
	}
	for _, l := range acct.lots {
		if l.expired(now) {
			expired = expired.Plus(l.remainder)
			continue
		}
		closing = closing.Plus(l.remainder)
		held = held.Plus(r.held(acct, l, now))
	}
	return Statement{Account: id, Currency: acct.a.Currency, Opening: text(opening), Increases: text(increases), Applications: text(applied), Expired: text(expired),
		ApplicationReversals: text(appReversed), LotReversals: text(lotReversed), Closing: text(closing), Held: text(held), Available: text(closing.Minus(held))}, nil
}
