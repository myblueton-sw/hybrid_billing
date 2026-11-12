package contracts

import (
	"math/big"
	"sort"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

type DayBasis string

const ActualDays DayBasis = "actual_days"

type SuspendedRule string

const (
	BillSuspended SuspendedRule = "bill_suspended"
	SkipSuspended SuspendedRule = "skip_suspended"
)

type Basis struct {
	WorkspaceID string
	ContractID  string
	ChargeID    string
	Period      Interval
	Point       uint64
	Days        DayBasis
	Suspended   SuspendedRule
	Scale       int
	Mode        decimal.Mode
}

func (b Basis) valid() bool {
	return validID(b.WorkspaceID) && validID(b.ContractID) && validID(b.ChargeID) && b.Period.Valid() && b.Days == ActualDays &&
		(b.Suspended == BillSuspended || b.Suspended == SkipSuspended) && b.Scale >= 0 && b.Mode.Valid()
}

type Line struct {
	Interval Interval
	Revision uint64
	Days     int
	Amount   decimal.Decimal
}

type RevisionTotal struct {
	Revision uint64
	Price    decimal.Decimal
	Total    decimal.Decimal
}

type Result struct {
	Basis           Basis
	ProfileID       string
	ProfileRevision string
	PeriodDays      int
	Lines           []Line
	Revisions       []RevisionTotal
	Total           decimal.Decimal
}

func covers(period Interval, segments []Segment) bool {
	if len(segments) == 0 || segments[0].Interval.start != period.start || segments[len(segments)-1].Interval.end != period.end {
		return false
	}
	for index, segment := range segments {
		if !segment.Interval.Valid() || (index > 0 && segments[index-1].Interval.end != segment.Interval.start) {
			return false
		}
		switch segment.State {
		case NotStarted, Active, Suspended, Ended:
		default:
			return false
		}
	}
	return true
}

func whole(p decimal.Profile, value int) (decimal.Decimal, error) {
	return decimal.New(p, big.NewInt(int64(value)), 0)
}

func Prorate(p decimal.Profile, basis Basis, amounts map[uint64]decimal.Decimal, segments []Segment) (Result, error) {
	if !p.Valid() {
		return Result{}, decimal.ErrInvalidProfile
	}
	if !basis.valid() || !covers(basis.Period, segments) {
		return Result{}, ErrInvalid
	}
	zero, err := whole(p, 0)
	if err != nil {
		return Result{}, err
	}
	days, err := whole(p, basis.Period.Days())
	if err != nil {
		return Result{}, err
	}
	if _, err := decimal.DivideRound(p, zero, days, basis.Scale, basis.Mode); err != nil {
		return Result{}, err
	}
	result := Result{Basis: basis, ProfileID: p.ID(), ProfileRevision: p.Revision(), PeriodDays: basis.Period.Days(), Lines: []Line{}, Revisions: []RevisionTotal{}, Total: zero}
	totals := map[uint64]int{}
	numerator, rounded, sign := zero, zero, 0
	for _, segment := range segments {
		if segment.State != Active && !(segment.State == Suspended && basis.Suspended == BillSuspended) {
			continue
		}
		price := amounts[segment.Revision]
		if !price.Valid() || price.Scale() > basis.Scale {
			return Result{}, ErrInvalid
		}
		if price.Sign() != 0 {
			if sign != 0 && sign != price.Sign() {
				return Result{}, ErrInvalid
			}
			sign = price.Sign()
		}
		count, err := whole(p, segment.Interval.Days())
		if err != nil {
			return Result{}, err
		}
		share, err := decimal.Mul(p, price, count)
		if err != nil {
			return Result{}, err
		}
		if numerator, err = decimal.Add(p, numerator, share); err != nil {
			return Result{}, err
		}
		running, err := decimal.DivideRound(p, numerator, days, basis.Scale, basis.Mode)
		if err != nil {
			return Result{}, err
		}
		amount, err := decimal.Sub(p, running, rounded)
		if err != nil {
			return Result{}, err
		}
		rounded = running
		result.Lines = append(result.Lines, Line{Interval: segment.Interval, Revision: segment.Revision, Days: segment.Interval.Days(), Amount: amount})
		index, listed := totals[segment.Revision]
		if !listed {
			index = len(result.Revisions)
			totals[segment.Revision] = index
			result.Revisions = append(result.Revisions, RevisionTotal{Revision: segment.Revision, Price: price, Total: zero})
		}
		if result.Revisions[index].Total, err = decimal.Add(p, result.Revisions[index].Total, amount); err != nil {
			return Result{}, err
		}
	}
	sort.Slice(result.Revisions, func(a, b int) bool { return result.Revisions[a].Revision < result.Revisions[b].Revision })
	result.Total = rounded
	return result, nil
}

type RevisionDifference struct {
	Revision uint64
	Amount   decimal.Decimal
}

type Difference struct {
	Total     decimal.Decimal
	Revisions []RevisionDifference
	Before    Result
	After     Result
}

func consistent(p decimal.Profile, result Result) bool {
	sum, err := whole(p, 0)
	seen := map[uint64]bool{}
	for _, entry := range result.Revisions {
		if err != nil || seen[entry.Revision] || !entry.Price.Valid() || !entry.Total.Valid() {
			return false
		}
		seen[entry.Revision] = true
		sum, err = decimal.Add(p, sum, entry.Total)
	}
	return err == nil && sum.Equal(result.Total)
}

func comparable(p decimal.Profile, before, after Result) bool {
	a, b := before.Basis, after.Basis
	a.Point, b.Point = 0, 0
	if a != b || !a.valid() || !before.Total.Valid() || !after.Total.Valid() {
		return false
	}
	for _, result := range []Result{before, after} {
		if result.ProfileID != p.ID() || result.ProfileRevision != p.Revision() {
			return false
		}
	}
	if !consistent(p, before) || !consistent(p, after) {
		return false
	}
	prices := map[uint64]decimal.Decimal{}
	for _, entry := range before.Revisions {
		prices[entry.Revision] = entry.Price
	}
	for _, entry := range after.Revisions {
		if price, both := prices[entry.Revision]; both && !price.Equal(entry.Price) {
			return false
		}
	}
	return true
}

func Correction(p decimal.Profile, before, after Result) (Difference, error) {
	if !p.Valid() {
		return Difference{}, decimal.ErrInvalidProfile
	}
	if !comparable(p, before, after) {
		return Difference{}, ErrInvalid
	}
	total, err := decimal.Sub(p, after.Total, before.Total)
	if err != nil {
		return Difference{}, err
	}
	zero, err := whole(p, 0)
	if err != nil {
		return Difference{}, err
	}
	amounts := map[uint64]decimal.Decimal{}
	for _, entry := range after.Revisions {
		amounts[entry.Revision] = entry.Total
	}
	for _, entry := range before.Revisions {
		current, listed := amounts[entry.Revision]
		if !listed {
			current = zero
		}
		if amounts[entry.Revision], err = decimal.Sub(p, current, entry.Total); err != nil {
			return Difference{}, err
		}
	}
	difference := Difference{Total: total, Revisions: []RevisionDifference{}, Before: before, After: after}
	for revision, amount := range amounts {
		difference.Revisions = append(difference.Revisions, RevisionDifference{Revision: revision, Amount: amount})
	}
	sort.Slice(difference.Revisions, func(a, b int) bool { return difference.Revisions[a].Revision < difference.Revisions[b].Revision })
	return difference, nil
}
