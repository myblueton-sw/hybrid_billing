package adjustments

import "github.com/myblueton-sw/hybrid_billing/packages/primitives/exact"

type computed struct {
	differences []Difference
	amounts     []exact.Fraction
	adjusted    []exact.Fraction
	source      exact.Fraction
	adjustment  exact.Fraction
	total       exact.Fraction
	gross       exact.Fraction
}

func (r *Registry) compute(d *draft, active []*entry) (computed, bool) {
	out := computed{source: exact.Whole(0), adjustment: exact.Whole(0), gross: exact.Whole(0)}
	quantity := map[string]exact.Fraction{}
	amount := map[string]exact.Fraction{}
	for _, l := range d.lines {
		quantity[l.source.ID] = l.quantity
		amount[l.source.ID] = l.amount
		out.source = out.source.Plus(l.amount)
	}
	total := out.source
	values := []exact.Fraction{out.source}
	for _, e := range active {
		v := e.view.Entry
		diff := Difference{Entry: v.ID, Kind: v.Kind, Line: v.Line}
		if v.Line != "" {
			diff.Charge = d.byLine[v.Line].source.Charge
		}
		var current, target, change exact.Fraction
		switch {
		case v.Kind == EntryQuantity:
			current, target = quantity[v.Line], e.value
			delta := target.Minus(current)
			price, base := d.byLine[v.Line].price, d.view.Draft.Base
			change = round(target.Times(price), base).Minus(round(current.Times(price), base))
			quantity[v.Line] = target
			diff.Quantity = show(delta)
			values = append(values, current, target, delta)
		case v.Line != "":
			current = amount[v.Line]
		default:
			current = total
		}
		switch v.Kind {
		case EntryFixed:
			change, target = e.value, current.Plus(e.value)
		case EntryTargetTotal:
			change, target = e.value.Minus(current), e.value
		}
		if v.Line != "" {
			amount[v.Line] = amount[v.Line].Plus(change)
			values = append(values, amount[v.Line])
		}
		total = total.Plus(change)
		out.adjustment = out.adjustment.Plus(change)
		if change.Sign() < 0 {
			out.gross = out.gross.Minus(change)
		} else {
			out.gross = out.gross.Plus(change)
		}
		diff.Current, diff.Target, diff.Amount = show(current), show(target), show(change)
		values = append(values, current, target, change, total, out.adjustment, out.gross)
		out.differences = append(out.differences, diff)
		out.amounts = append(out.amounts, change)
	}
	for _, l := range d.lines {
		out.adjusted = append(out.adjusted, amount[l.source.ID])
	}
	out.total = total
	for _, value := range values {
		if !r.bounded(value) {
			return computed{}, false
		}
	}
	return out, true
}

func activeOf(list []*entry) []*entry {
	var out []*entry
	for _, e := range list {
		if e.view.Active {
			out = append(out, e)
		}
	}
	return out
}

func (d *draft) blocked() error {
	if len(d.result.amounts) == 0 {
		return refuse(RefuseNothingToAdjust)
	}
	if d.zero() {
		return refuse(RefuseZeroDifference)
	}
	if d.result.total.Sign() < 0 {
		return refuse(RefuseNegativeTotal)
	}
	return nil
}

func (d *draft) zero() bool {
	for _, change := range d.result.amounts {
		if change.Sign() == 0 {
			return true
		}
	}
	return false
}
