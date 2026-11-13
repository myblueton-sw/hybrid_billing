package commitment

import (
	"math/big"
	"sort"
	"strings"

	"github.com/myblueton-sw/hybrid_billing/modules/allocation"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/exact"
)

type parts struct {
	usedRecurring   exact.Fraction
	usedAmortized   exact.Fraction
	list            exact.Fraction
	negative        int
	unusedRecurring exact.Fraction
	unusedAmortized exact.Fraction
}

type Allocation struct {
	made      bool
	limits    Limits
	scale     int
	recovered bool
	cost      exact.Fraction
	amortized exact.Fraction
	customers map[string]*parts
	view      AllocationView
}

func (a Allocation) View() AllocationView {
	view := a.view
	view.Customers = append([]CustomerPart{}, a.view.Customers...)
	return view
}

func format(f exact.Fraction) (string, bool) {
	text := f.Text()
	num, den := text, "1"
	if index := strings.IndexByte(text, '/'); index >= 0 {
		num, den = text[:index], text[index+1:]
	}
	value, _ := new(big.Int).SetString(num, 10)
	divisor, _ := new(big.Int).SetString(den, 10)
	rest, twos, fives := new(big.Int).Set(divisor), 0, 0
	two, five, zero := big.NewInt(2), big.NewInt(5), new(big.Int)
	for new(big.Int).Rem(rest, two).Cmp(zero) == 0 {
		rest.Quo(rest, two)
		twos++
	}
	for new(big.Int).Rem(rest, five).Cmp(zero) == 0 {
		rest.Quo(rest, five)
		fives++
	}
	if rest.Cmp(big.NewInt(1)) != 0 {
		return "", false
	}
	scale := twos
	if fives > scale {
		scale = fives
	}
	units := new(big.Int).Mul(value, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil))
	units.Quo(units, divisor)
	negative := units.Sign() < 0
	digits := new(big.Int).Abs(units).String()
	for len(digits) <= scale {
		digits = "0" + digits
	}
	out := digits
	if scale > 0 {
		out = digits[:len(digits)-scale] + "." + digits[len(digits)-scale:]
	}
	if negative {
		out = "-" + out
	}
	return out, true
}

func decimalOf(profile decimal.Profile, f exact.Fraction) (decimal.Decimal, bool) {
	text, ok := format(f)
	if !ok {
		return decimal.Decimal{}, false
	}
	scale := 0
	if index := strings.IndexByte(text, '.'); index >= 0 {
		scale = len(text) - index - 1
	}
	units, _ := new(big.Int).SetString(strings.Replace(text, ".", "", 1), 10)
	value, err := decimal.New(profile, units, scale)
	return value, err == nil
}

func attributionDigest(a Attribution) string {
	p := newPrinter(attributionLbl)
	p.fields(a.ID, a.Version)
	p.number(len(a.Mappings))
	for _, m := range a.Mappings {
		p.fields(m.Account, m.Customer)
	}
	return p.hex()
}

func policyDigest(policy AllocationPolicy) string {
	p := newPrinter(allocPolicyLbl)
	p.fields(policy.ID, policy.Version, policy.Bearing, policy.Customer)
	return p.hex()
}

func allocationDigest(v AllocationView) string {
	p := newPrinter(AllocationLbl)
	p.fields(v.ProfileID, v.ProfileRevision, v.Commitment, v.Month)
	p.number(v.Revision)
	p.fields(v.PeriodDigest, v.Recovery, v.AttributionID, v.AttributionVersion, v.AttributionDigest, v.PolicyID, v.PolicyVersion, v.PolicyDigest, v.Bearing, v.Hold)
	p.number(len(v.Customers))
	for _, c := range v.Customers {
		p.fields(c.Customer, c.UsedRecurring, c.UsedAmortized, c.List, c.Savings)
		p.number(c.NegativeSavings)
		p.fields(c.UnusedRecurring, c.UnusedAmortized)
	}
	p.fields(v.MSPRecurring, v.MSPAmortized, v.RoundingRecurring, v.RoundingAmortized, v.UnattributedRecurring, v.UnattributedAmortized)
	return p.hex()
}

func Allocate(period Period, attribution Attribution, policy AllocationPolicy) (Allocation, error) {
	if !period.made {
		return Allocation{}, ErrInvalid
	}
	if len(attribution.Mappings) > period.limits.Customers {
		return Allocation{}, ErrLimit
	}
	mapped := map[string]string{}
	for _, m := range attribution.Mappings {
		if !validID(m.Account) || !validID(m.Customer) || mapped[m.Account] != "" {
			return Allocation{}, ErrInvalid
		}
		mapped[m.Account] = m.Customer
	}
	if !validID(attribution.ID) || !validID(attribution.Version) || !validID(policy.ID) || !validID(policy.Version) ||
		!bearings[policy.Bearing] || (policy.Bearing == BearCustomer) != (policy.Customer != "") || (policy.Customer != "" && !validID(policy.Customer)) {
		return Allocation{}, ErrInvalid
	}
	sorted := append([]Mapping{}, attribution.Mappings...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Account < sorted[j].Account })
	attribution.Mappings = sorted
	c := period.commitment
	a := Allocation{made: true, limits: period.limits, scale: quanta[c.Quantum], recovered: c.Recovery == RecoveryRecovered, customers: map[string]*parts{}}
	a.cost = sumOf(period.profile, period.facts.Recurring, period.facts.Amortized)
	a.amortized = sumOf(period.profile, period.facts.Amortized)
	v := AllocationView{Label: AllocationLbl, ProfileID: period.profile.ID(), ProfileRevision: period.profile.Revision(), Commitment: c.ID, Month: period.month,
		Revision: period.revision, PeriodDigest: period.digest, Recovery: c.Recovery, AttributionID: attribution.ID, AttributionVersion: attribution.Version,
		AttributionDigest: attributionDigest(attribution), PolicyID: policy.ID, PolicyVersion: policy.Version, PolicyDigest: policyDigest(policy), Bearing: policy.Bearing, Customers: []CustomerPart{}}
	fresh := func() *parts {
		return &parts{usedRecurring: exact.Whole(0), usedAmortized: exact.Whole(0), list: exact.Whole(0), unusedRecurring: exact.Whole(0), unusedAmortized: exact.Whole(0)}
	}
	unusedRec, unusedAm := exact.Whole(0), exact.Whole(0)
	for _, line := range period.facts.Lines {
		rec, am := sumOf(period.profile, line.Recurring), sumOf(period.profile, line.Amortized)
		if line.Kind == LineUnused {
			unusedRec, unusedAm = unusedRec.Plus(rec), unusedAm.Plus(am)
			continue
		}
		customer, ok := mapped[line.Account]
		if !ok {
			v.Hold = HoldAccountNotAttributed
			continue
		}
		part := a.customers[customer]
		if part == nil {
			part = fresh()
			a.customers[customer] = part
		}
		list := sumOf(period.profile, line.List)
		part.usedRecurring, part.usedAmortized, part.list = part.usedRecurring.Plus(rec), part.usedAmortized.Plus(am), part.list.Plus(list)
		if list.Less(rec.Plus(am)) {
			part.negative++
		}
	}
	zero := exact.Whole(0)
	msp := [2]exact.Fraction{zero, zero}
	rounding := [2]exact.Fraction{zero, zero}
	unattributed := [2]exact.Fraction{zero, zero}
	if v.Hold == "" {
		switch policy.Bearing {
		case BearMSP:
			msp = [2]exact.Fraction{unusedRec, unusedAm}
		case BearUnattributed:
			unattributed = [2]exact.Fraction{unusedRec, unusedAm}
		case BearCustomer:
			if unusedRec.Sign() == 0 && unusedAm.Sign() == 0 {
				break
			}
			part := a.customers[policy.Customer]
			if part == nil {
				part = fresh()
				a.customers[policy.Customer] = part
			}
			part.unusedRecurring, part.unusedAmortized = unusedRec, unusedAm
		default:
			basis := zero
			for _, part := range a.customers {
				basis = basis.Plus(part.usedRecurring).Plus(part.usedAmortized)
			}
			if (unusedRec.Sign() > 0 || unusedAm.Sign() > 0) && basis.Sign() == 0 {
				v.Hold = HoldNoBasis
				break
			}
			for index, total := range []exact.Fraction{unusedRec, unusedAm} {
				if total.Sign() == 0 {
					continue
				}
				shares, left, err := a.split(period, total)
				if err != nil {
					return Allocation{}, err
				}
				rounding[index] = left
				for customer, share := range shares {
					if index == 0 {
						a.customers[customer].unusedRecurring = share
					} else {
						a.customers[customer].unusedAmortized = share
					}
				}
			}
		}
	}
	if v.Hold != "" {
		a.customers = map[string]*parts{}
		v.Digest = allocationDigest(v)
		a.view = v
		return a, nil
	}
	names := make([]string, 0, len(a.customers))
	for customer := range a.customers {
		names = append(names, customer)
	}
	sort.Strings(names)
	if len(names) > period.limits.Customers {
		return Allocation{}, ErrLimit
	}
	total := msp[0].Plus(msp[1]).Plus(rounding[0]).Plus(rounding[1]).Plus(unattributed[0]).Plus(unattributed[1])
	texts := []string{}
	out := func(f exact.Fraction) string {
		text, ok := format(f)
		if !ok || f.Digits() > period.limits.Digits {
			texts = append(texts, "")
		}
		return text
	}
	for _, name := range names {
		part := a.customers[name]
		total = total.Plus(part.usedRecurring).Plus(part.usedAmortized).Plus(part.unusedRecurring).Plus(part.unusedAmortized)
		savings := part.list.Minus(part.usedRecurring).Minus(part.usedAmortized)
		v.Customers = append(v.Customers, CustomerPart{Customer: name, UsedRecurring: out(part.usedRecurring), UsedAmortized: out(part.usedAmortized), List: out(part.list),
			Savings: out(savings), NegativeSavings: part.negative, UnusedRecurring: out(part.unusedRecurring), UnusedAmortized: out(part.unusedAmortized)})
	}
	v.MSPRecurring, v.MSPAmortized = out(msp[0]), out(msp[1])
	v.RoundingRecurring, v.RoundingAmortized = out(rounding[0]), out(rounding[1])
	v.UnattributedRecurring, v.UnattributedAmortized = out(unattributed[0]), out(unattributed[1])
	if len(texts) > 0 {
		return Allocation{}, ErrLimit
	}
	if !same(total, a.cost) {
		return Allocation{}, ErrSelfCheck
	}
	v.Digest = allocationDigest(v)
	a.view = v
	return a, nil
}

func (a Allocation) split(period Period, total exact.Fraction) (map[string]exact.Fraction, exact.Fraction, error) {
	amount, ok := decimalOf(period.profile, total)
	quantum, _ := decimal.Parse(period.profile, period.commitment.Quantum)
	if !ok {
		return nil, exact.Fraction{}, ErrLimit
	}
	targets := []allocation.Target{}
	for customer, part := range a.customers {
		weight, ok := decimalOf(period.profile, part.usedRecurring.Plus(part.usedAmortized))
		if !ok {
			return nil, exact.Fraction{}, ErrLimit
		}
		targets = append(targets, allocation.Target{ID: customer, Weight: weight})
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].ID < targets[j].ID })
	result, err := allocation.LargestRemainderV1(allocation.Request{Profile: period.profile, Limits: allocation.Limits{MaxTargets: period.limits.Customers, MaxIntermediateDigits: period.profile.IntermediateDigits()},
		Encoding: allocation.EncodingASCIIBytewiseV1, Total: amount, Quantum: quantum, Mode: decimal.TowardZero, Targets: targets})
	if err == allocation.ErrConservation {
		return nil, exact.Fraction{}, ErrSelfCheck
	}
	if err != nil {
		return nil, exact.Fraction{}, ErrLimit
	}
	shares := map[string]exact.Fraction{}
	for _, share := range result.Shares {
		shares[share.ID], _ = exact.Of(share.Amount)
	}
	left, _ := exact.Of(result.RoundingComponent)
	return shares, left, nil
}
