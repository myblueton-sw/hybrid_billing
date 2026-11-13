package fx

import (
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/exact"
)

type rates struct {
	sales     exact.Fraction
	perSource exact.Fraction
	fee       exact.Fraction
	scale     int
}

type resolver struct {
	r     *Registry
	s     *space
	local time.Time
}

type give int

const (
	giveNone give = iota
	giveValue
	givePending
	giveVoid
)

func (z *resolver) fit(f exact.Fraction) error {
	if f.Digits() > z.r.limits.Digits {
		return ErrLimit
	}
	return nil
}

func (z *resolver) at(key Key) (give, *revision) {
	if z.s == nil {
		return giveNone, nil
	}
	revisions := z.s.keys[key]
	if len(revisions) == 0 {
		return giveNone, nil
	}
	top := revisions[len(revisions)-1]
	switch top.state {
	case ObservationRecorded, ObservationApproved:
		return giveValue, top
	case ObservationVoid:
		return giveVoid, top
	}
	return givePending, top
}

func (z *resolver) policy(level, key string, need Need) *policy {
	if z.s == nil {
		return nil
	}
	return z.s.policies[z.s.places[place{level, key, need.Purpose, need.Source, need.Target}]]
}

func normalised(rev *revision, r *Registry) exact.Fraction {
	_, value, _ := r.number(rev.input.Value)
	made, _ := value.Over(exact.Whole(int64(rev.input.Unit)))
	return made
}

func used(rev *revision) Used {
	reference := ""
	if rev.approval != nil {
		reference = rev.approval.Reference
	}
	return Used{Key: rev.input.Key, Revision: rev.input.Revision, Unit: rev.input.Unit, Value: rev.input.Value, Origin: rev.input.Origin, Evidence: rev.input.Evidence, Reference: reference}
}

func (z *resolver) keys(v *version, need Need, date time.Time) []Key {
	day := dateText(date)
	in := v.input
	switch in.Construction {
	case ConstructionDirect:
		return []Key{{in.RateSource, in.Series, need.Source, need.Target, day}}
	case ConstructionInverse:
		return []Key{{in.RateSource, in.Series, need.Target, need.Source, day}}
	}
	return []Key{{in.RateSource, in.Series, need.Source, in.Pivot, day}, {in.RateSource, in.Series, need.Target, in.Pivot, day}}
}

func (z *resolver) hold(entry *Entry, reason string) (Entry, rates, error) {
	entry.Kind, entry.Reason, entry.Observations = EntryHeld, reason, []Used{}
	return *entry, rates{}, nil
}

func (z *resolver) choose(entry *Entry, need Need, date time.Time) (*version, string) {
	if p := z.policy(LevelCustomer, need.Customer, need); p != nil {
		if v := p.on(date); v != nil {
			entry.Level, entry.Policy, entry.Version = LevelCustomer, p.id, v.input.Version
			if v.withdrawal != nil {
				return nil, ReasonWithdrawnInForce
			}
			if v.input.Method != MethodEnded {
				entry.Reason = ReasonCustomerException
				return v, ""
			}
			entry.Level, entry.Policy, entry.Version = "", "", 0
		}
	}
	p := z.policy(LevelContract, need.Contract, need)
	if p == nil {
		return nil, ReasonContractNotStated
	}
	v := p.on(date)
	if v == nil || (v.withdrawal == nil && v.input.Method == MethodEnded) {
		return nil, ReasonContractNotStated
	}
	entry.Level, entry.Policy, entry.Version = LevelContract, p.id, v.input.Version
	if v.withdrawal != nil {
		return nil, ReasonWithdrawnInForce
	}
	if v.input.Method != MethodDefaultAllowed {
		entry.Reason = ReasonContract
		return v, ""
	}
	entry.Level, entry.Policy, entry.Version = "", "", 0
	entry.AllowingPolicy, entry.AllowingVersion = p.id, v.input.Version
	p = z.policy(LevelMSP, "", need)
	if p == nil {
		return nil, ReasonNoDefaultPolicy
	}
	if v = p.on(date); v == nil {
		return nil, ReasonNoDefaultPolicy
	}
	entry.Level, entry.Policy, entry.Version = LevelMSP, p.id, v.input.Version
	if v.withdrawal != nil {
		return nil, ReasonWithdrawnInForce
	}
	entry.Reason = ReasonMSPDefault
	return v, ""
}

func (z *resolver) resolve(need Need) (Entry, rates, error) {
	entry := Entry{Need: need, Observations: []Used{}}
	if need.Source == need.Target {
		entry.Kind = EntrySameCurrency
		return entry, rates{}, nil
	}
	date, _ := parseDate(need.Date)
	if date.After(z.local) {
		return z.hold(&entry, ReasonDateNotReached)
	}
	v, reason := z.choose(&entry, need, date)
	if v == nil {
		return z.hold(&entry, reason)
	}
	in := v.input
	entry.Method, entry.Construction, entry.Pivot, entry.Month, entry.Averaging = in.Method, in.Construction, in.Pivot, in.Month, in.Averaging
	entry.LeastDays, entry.Rule, entry.MaxDays = in.LeastDays, in.Rule, in.MaxDays
	entry.Spread, entry.QuoteScale, entry.QuoteMode, entry.Tolerance = in.Spread, in.QuoteScale, in.QuoteMode, in.Tolerance
	entry.Fee, entry.Quantum, entry.AmountMode, entry.Order = in.Fee, in.Quantum, in.AmountMode, in.Order
	var quoted exact.Fraction
	var unit int
	switch in.Method {
	case MethodFixed:
		entry.FixedUnit, entry.FixedValue, entry.FixedEvidence = in.Unit, in.Value, in.Evidence
		if v.approval == nil {
			return z.hold(&entry, ReasonAwaitingApproval)
		}
		entry.FixedReference = v.approval.Reference
		_, quoted, _ = z.r.number(in.Value)
		unit = in.Unit
	case MethodDated:
		var held string
		quoted, unit, held = z.dated(&entry, v, need, date)
		if held != "" {
			return z.hold(&entry, held)
		}
	default:
		var held string
		var err error
		quoted, unit, held, err = z.average(&entry, v, need, date)
		if err != nil {
			return Entry{}, rates{}, err
		}
		if held != "" {
			return z.hold(&entry, held)
		}
	}
	return z.price(&entry, in, quoted, unit)
}

func (z *resolver) dated(entry *Entry, v *version, need Need, date time.Time) (exact.Fraction, int, string) {
	window := v.input.MaxDays
	seen := make([]bool, len(z.keys(v, need, date)))
	for back := 0; back <= window; back++ {
		day := date.AddDate(0, 0, -back)
		keys := z.keys(v, need, day)
		found := make([]*revision, 0, len(keys))
		states := make([]give, len(keys))
		for index, key := range keys {
			state, rev := z.at(key)
			states[index] = state
			if state == giveValue {
				seen[index] = true
				found = append(found, rev)
			}
		}
		if len(found) != len(keys) && !date.Before(z.local) {
			return exact.Fraction{}, 0, ReasonDateNotReached
		}
		for _, state := range states {
			if state == givePending {
				return exact.Fraction{}, 0, ReasonAwaitingApproval
			}
		}
		for _, state := range states {
			if state == giveVoid {
				entry.VoidsPassed++
			}
		}
		if len(found) == len(keys) {
			if back > 0 {
				entry.DateUsed, entry.DaysElapsed = dateText(day), back
			}
			for _, rev := range found {
				entry.Observations = append(entry.Observations, used(rev))
			}
			if v.input.Construction == ConstructionCross {
				made, _ := normalised(found[0], z.r).Over(normalised(found[1], z.r))
				return made, 1, ""
			}
			_, value, _ := z.r.number(found[0].input.Value)
			return value, found[0].input.Unit, ""
		}
	}
	for _, leg := range seen {
		if !leg {
			return exact.Fraction{}, 0, ReasonNoRateRecorded
		}
	}
	return exact.Fraction{}, 0, ReasonLegsNotOnOneDate
}

func (z *resolver) average(entry *Entry, v *version, need Need, date time.Time) (exact.Fraction, int, string, error) {
	start := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, time.UTC)
	if v.input.Month == MonthPrevious {
		start = start.AddDate(0, -1, 0)
	}
	end := start.AddDate(0, 1, 0)
	if z.local.Before(end) {
		return exact.Fraction{}, 0, ReasonMonthNotEnded, nil
	}
	legs := len(z.keys(v, need, start))
	sums := make([]exact.Fraction, legs)
	counts := make([]int, legs)
	largest := make([]int, legs)
	for index := range sums {
		sums[index] = exact.Whole(0)
	}
	for day := start; day.Before(end); day = day.AddDate(0, 0, 1) {
		for index, key := range z.keys(v, need, day) {
			state, rev := z.at(key)
			switch state {
			case givePending:
				return exact.Fraction{}, 0, ReasonAwaitingApproval, nil
			case giveVoid:
				entry.VoidsPassed++
			case giveValue:
				sums[index] = sums[index].Plus(normalised(rev, z.r))
				if err := z.fit(sums[index]); err != nil {
					return exact.Fraction{}, 0, "", err
				}
				counts[index]++
				if rev.input.Unit > largest[index] {
					largest[index] = rev.input.Unit
				}
				entry.Observations = append(entry.Observations, used(rev))
			}
		}
	}
	for _, count := range counts {
		if count < v.input.LeastDays {
			return exact.Fraction{}, 0, ReasonNotEnoughDays, nil
		}
	}
	means := make([]exact.Fraction, legs)
	for index := range sums {
		means[index], _ = sums[index].Over(exact.Whole(int64(counts[index])))
	}
	if legs == 2 {
		made, _ := means[0].Over(means[1])
		return made, 1, "", z.fit(made)
	}
	made := means[0].Times(exact.Whole(int64(largest[0])))
	return made, largest[0], "", z.fit(made)
}

func (z *resolver) price(entry *Entry, in VersionInput, quoted exact.Fraction, unit int) (Entry, rates, error) {
	inverse := in.Construction == ConstructionInverse
	_, spread, _ := z.r.number(in.Spread)
	_, fee, _ := z.r.number(in.Fee)
	_, tolerance, _ := z.r.number(in.Tolerance)
	factor := exact.Whole(1).Plus(spread)
	per := exact.Whole(int64(unit))
	exactQuote := quoted.Times(factor)
	perSource, _ := quoted.Over(per)
	if inverse {
		exactQuote, _ = quoted.Over(factor)
		perSource, _ = per.Over(quoted)
	}
	for _, f := range []exact.Fraction{quoted, exactQuote, perSource} {
		if err := z.fit(f); err != nil {
			return Entry{}, rates{}, err
		}
	}
	entry.QuotedRate, entry.Unit, entry.ExactQuote, entry.PerSource = quoted.Text(), unit, exactQuote.Text(), perSource.Text()
	count, _ := exactQuote.Quanta(in.QuoteScale, in.QuoteMode)
	shown, err := decimal.New(z.r.profile, count, in.QuoteScale)
	if err != nil {
		return Entry{}, rates{}, ErrLimit
	}
	entry.SalesQuote = shown.String()
	rounded, _ := exact.Units(count, in.QuoteScale)
	if rounded.Sign() <= 0 {
		return z.hold(entry, ReasonRateRoundsToZero)
	}
	distance := rounded.Minus(exactQuote)
	if distance.Sign() < 0 {
		distance = exact.Whole(0).Minus(distance)
	}
	smaller := rounded
	if exactQuote.Less(rounded) {
		smaller = exactQuote
	}
	difference, _ := distance.Over(smaller)
	if err := z.fit(difference); err != nil {
		return Entry{}, rates{}, err
	}
	entry.Difference = difference.Text()
	if tolerance.Less(difference) {
		return z.hold(entry, ReasonRatePrecisionLost)
	}
	sales, _ := rounded.Over(per)
	if inverse {
		sales, _ = per.Over(rounded)
	}
	if err := z.fit(sales); err != nil {
		return Entry{}, rates{}, err
	}
	entry.SalesRate, entry.Kind = sales.Text(), EntryRate
	return *entry, rates{sales: sales, perSource: perSource, fee: fee, scale: quanta[in.Quantum]}, nil
}
