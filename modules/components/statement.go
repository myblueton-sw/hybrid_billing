package components

import (
	"sort"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/exact"
)

func exactOne() exact.Fraction { return exact.Whole(1) }

func (r *Registry) requestForm(q Request) (map[string]exact.Fraction, []string, bool) {
	if !validID(q.Contract) || !month(q.PeriodStart, q.PeriodEnd) || len(q.Usage) > r.limits.Readings {
		return nil, nil, false
	}
	readings := map[string]exact.Fraction{}
	order := []string{}
	for _, reading := range q.Usage {
		quantity, _, ok := r.number(reading.Quantity, false)
		if !validID(reading.Meter) || !ok || readings[reading.Meter].Valid() {
			return nil, nil, false
		}
		readings[reading.Meter] = quantity
		order = append(order, reading.Meter)
	}
	sort.Strings(order)
	return readings, order, true
}

func (r *Registry) Statement(workspace string, q Request) (Statement, error) {
	if !validID(workspace) {
		return Statement{}, ErrInvalid
	}
	readings, meters, ok := r.requestForm(q)
	if !ok {
		return Statement{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	out := Statement{Contract: q.Contract, PeriodStart: q.PeriodStart, PeriodEnd: q.PeriodEnd, Lines: []Line{}, Unpriced: []Unpriced{}, Credits: []CreditLine{}, Evaluated: now}
	s := r.workspaces[workspace]
	if s == nil || s.pricing[q.Contract] == nil {
		return Statement{}, refuse(ReasonNoPricing)
	}
	pricing := s.pricing[q.Contract].Pricing
	var taking []*component
	for _, c := range s.contracts[q.Contract] {
		k := c.view.Component
		if c.view.State != StateApproved || !overlap(k.From, k.To, q.PeriodStart, q.PeriodEnd) {
			continue
		}
		if k.From > q.PeriodStart || endOf(k.To) < q.PeriodEnd {
			return Statement{}, refuse(ReasonPartialPeriod)
		}
		if out.Currency != "" && out.Currency != k.Currency {
			return Statement{}, refuse(ReasonCurrencyMismatch)
		}
		out.Currency = k.Currency
		taking = append(taking, c)
	}
	sort.Slice(taking, func(a, b int) bool { return taking[a].view.Component.ID < taking[b].view.Component.ID })
	charges := exact.Whole(0)
	priced := map[string]bool{}
	inclusion := func(meter string) (exact.Fraction, bool) {
		for _, c := range taking {
			if c.all[meter] {
				return exact.Fraction{}, true
			}
			if q, ok := c.included[meter]; ok {
				return q, false
			}
		}
		return exact.Whole(0), false
	}
	for _, c := range taking {
		k := c.view.Component
		if fixedKinds[k.Kind] {
			out.Lines = append(out.Lines, Line{Kind: LineFixed, Component: k.ID, Amount: show(c.fee), Includes: append([]Inclusion(nil), k.Includes...)})
			charges = charges.Plus(c.fee)
			continue
		}
		measured, ok := readings[k.Meter]
		if !ok {
			return Statement{}, refuse(ReasonMissingReading)
		}
		priced[k.Meter] = true
		included, all := inclusion(k.Meter)
		line := Line{Kind: LineUsage, Component: k.ID, Meter: k.Meter, Measured: show(measured), Rate: k.Rate}
		billable := exact.Whole(0)
		if all {
			line.Included = All
		} else {
			line.Included = show(included)
			if included.Less(measured) {
				billable = measured.Minus(included)
			}
		}
		amount := round(billable.Times(c.rate), pricing)
		line.Billable, line.Amount = show(billable), show(amount)
		out.Lines = append(out.Lines, line)
		charges = charges.Plus(amount)
	}
	for _, meter := range meters {
		if priced[meter] {
			continue
		}
		measured := readings[meter]
		included, all := inclusion(meter)
		switch {
		case all:
			out.Lines = append(out.Lines, Line{Kind: LineIncluded, Meter: meter, Measured: show(measured), Included: All, Billable: "0", Amount: "0"})
		case included.Sign() > 0 && (measured.Less(included) || same(measured, included)):
			out.Lines = append(out.Lines, Line{Kind: LineIncluded, Meter: meter, Measured: show(measured), Included: show(included), Billable: "0", Amount: "0"})
		case included.Sign() > 0:
			out.Lines = append(out.Lines, Line{Kind: LineIncluded, Meter: meter, Measured: show(measured), Included: show(included), Billable: "0", Amount: "0"})
			out.Unpriced = append(out.Unpriced, Unpriced{Meter: meter, Measured: show(measured), Included: show(included), Excess: show(measured.Minus(included))})
		default:
			out.Unpriced = append(out.Unpriced, Unpriced{Meter: meter, Measured: show(measured), Included: "0", Excess: show(measured)})
		}
	}
	ids := make([]string, 0)
	for id, c := range s.credits {
		if c.view.State == StateApproved && c.clause.view.Clause.Contract == q.Contract && c.view.Credit.PeriodStart == q.PeriodStart {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		c := s.credits[id]
		state := CreditUnlinked
		if c.view.Link != (Link{}) {
			state = CreditLinked
		}
		out.Credits = append(out.Credits, CreditLine{Credit: id, Component: c.view.Component, Amount: c.view.Amount, State: state, Document: c.view.Link.Document, Increase: c.view.Link.Increase})
	}
	out.Charges = show(charges)
	out.Net = out.Charges
	return out, nil
}
