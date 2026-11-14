package receivables

import (
	"sort"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/exact"
)

func (r *Registry) Confirm(workspace, entity, currency string, asOf time.Time, actor string) (Confirmation, error) {
	if !validID(workspace) || !validID(entity) || !validCurrency(currency) || asOf.IsZero() || !validID(actor) {
		return Confirmation{}, ErrInvalid
	}
	asOf = asOf.UTC().Round(0)
	now := r.enter()
	defer r.leave()
	key := [2]string{entity, currency}
	if s := r.workspaces[workspace]; s != nil {
		if old, ok := s.confirmations[key]; ok {
			if old.AsOf.Equal(asOf) && old.Actor == actor {
				return old, nil
			}
			if !old.AsOf.Before(asOf) {
				return Confirmation{}, ErrRejected
			}
		}
	}
	if asOf.After(now) {
		return Confirmation{}, ErrRejected
	}
	s, err := r.open(workspace)
	if err != nil {
		return Confirmation{}, err
	}
	c := Confirmation{Entity: entity, Currency: currency, AsOf: asOf, Actor: actor, Recorded: now}
	s.confirmations[key] = c
	return c, nil
}

func (r *Registry) evaluate(s *space, d *document, now time.Time) StatusView {
	v := StatusView{Document: viewDocument(d), Evaluated: now}
	outstanding := d.outstanding()
	v.Outstanding = r.print(outstanding)
	if d.view.Receivable.Obligation != ObligationPayable {
		v.Status = StatusNotApplicable
		return v
	}
	c, confirmed := s.confirmations[[2]string{d.view.Receivable.Entity, d.view.Receivable.Currency}]
	v.Confirmed = c.AsOf
	v.Stale = !confirmed || now.Sub(c.AsOf) > r.limits.Freshness
	v.Days = daysBetween(d.view.Snapshot.EffectiveDue, today(d, now))
	switch {
	case outstanding.Sign() == 0:
		v.Status = StatusSettled
	case v.Days < 0:
		v.Status = StatusNotDue
	case v.Days == 0:
		v.Status = StatusDueToday
	case v.Stale:
		v.Status = StatusUnknown
	default:
		v.Status = StatusOverdue
	}
	return v
}

func (r *Registry) Status(workspace, id string) (StatusView, error) {
	if !validID(workspace) || !validID(id) {
		return StatusView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	d := r.find(workspace, id)
	if d == nil {
		return StatusView{}, ErrNotFound
	}
	return r.evaluate(r.workspaces[workspace], d, now), nil
}

func bucketOf(v StatusView) string {
	switch {
	case v.Document.ReplacedBy != "":
		return BucketReplaced
	case v.Status == StatusNotDue:
		return BucketNotDue
	case v.Status == StatusDueToday:
		return BucketDueToday
	case v.Status == StatusUnknown:
		return BucketUnknown
	case v.Days <= 30:
		return Bucket1to30
	case v.Days <= 60:
		return Bucket31to60
	case v.Days <= 90:
		return Bucket61to90
	}
	return Bucket91Plus
}

func filterForm(f Filter) bool {
	return optionalID(f.Customer) && optionalID(f.Entity) && (f.Currency == "" || validCurrency(f.Currency)) && optionalID(f.Contract)
}

func (f Filter) matches(v Receivable) bool {
	return (f.Customer == "" || v.Customer == f.Customer) && (f.Entity == "" || v.Entity == f.Entity) && (f.Currency == "" || v.Currency == f.Currency) && (f.Contract == "" || v.Contract == f.Contract)
}

type tally struct {
	count, stale     int
	amount, staleSum exact.Fraction
}

func (r *Registry) Aging(workspace string, f Filter) (Aging, error) {
	if !validID(workspace) || !filterForm(f) {
		return Aging{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	out := Aging{Evaluated: now, Currencies: []AgingView{}}
	s := r.workspaces[workspace]
	if s == nil {
		return out, nil
	}
	sums := map[string]map[string]*tally{}
	for _, d := range s.documents {
		if !f.matches(d.view.Receivable) {
			continue
		}
		v := r.evaluate(s, d, now)
		if v.Status == StatusNotApplicable || v.Status == StatusSettled {
			continue
		}
		currency := d.view.Receivable.Currency
		if sums[currency] == nil {
			sums[currency] = map[string]*tally{}
			for _, name := range buckets {
				sums[currency][name] = &tally{amount: exact.Whole(0), staleSum: exact.Whole(0)}
			}
		}
		t := sums[currency][bucketOf(v)]
		amount := d.outstanding()
		t.count++
		t.amount = t.amount.Plus(amount)
		if v.Stale {
			t.stale++
			t.staleSum = t.staleSum.Plus(amount)
		}
	}
	currencies := make([]string, 0, len(sums))
	for currency := range sums {
		currencies = append(currencies, currency)
	}
	sort.Strings(currencies)
	for _, currency := range currencies {
		view := AgingView{Currency: currency, Buckets: map[string]Bucket{}}
		total := exact.Whole(0)
		for _, name := range buckets {
			t := sums[currency][name]
			view.Buckets[name] = Bucket{Count: t.count, Amount: r.print(t.amount), StaleCount: t.stale, StaleAmount: r.print(t.staleSum)}
			view.Count += t.count
			total = total.Plus(t.amount)
		}
		view.Total = r.print(total)
		out.Currencies = append(out.Currencies, view)
	}
	return out, nil
}
