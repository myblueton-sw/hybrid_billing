package fx

import (
	"math/big"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/exact"
)

func (r *Registry) Convert(workspace, id string, lines []Line) (Result, error) {
	if len(lines) > r.limits.Lines {
		return Result{}, ErrLimit
	}
	seen := map[string]bool{}
	amounts := make([]exact.Fraction, len(lines))
	for index, line := range lines {
		_, amount, formed := r.number(line.Amount)
		if !validID(line.ID) || seen[line.ID] || !validID(line.Need) || !formed || !validCurrency(line.Currency) ||
			(line.State != StateSource && line.State != StateAlreadyInTarget) {
			return Result{}, ErrInvalid
		}
		seen[line.ID] = true
		amounts[index] = amount
	}
	if !validID(workspace) || !validID(id) {
		return Result{}, ErrInvalid
	}
	r.enter()
	s := r.workspaces[workspace]
	var found *stored
	if s != nil {
		found = s.snapshots[id]
	}
	r.leave()
	if found == nil {
		return Result{}, ErrNotFound
	}
	index := map[string]int{}
	for position, entry := range found.snapshot.Entries {
		index[entry.Need.ID] = position
	}
	for _, line := range lines {
		if _, ok := index[line.Need]; !ok {
			return Result{}, ErrNotFound
		}
	}
	kinds := make([]string, len(lines))
	for position, line := range lines {
		entry := found.snapshot.Entries[index[line.Need]]
		switch {
		case entry.Kind == EntrySameCurrency && line.Currency == entry.Need.Source:
			kinds[position] = ResultSameCurrency
		case entry.Kind != EntrySameCurrency && line.State == StateAlreadyInTarget && line.Currency == entry.Need.Target:
			kinds[position] = ResultAlreadyInTarget
		case entry.Kind == EntryHeld && line.State == StateSource && line.Currency == entry.Need.Source:
			kinds[position] = ResultHeld
		case entry.Kind == EntryRate && line.State == StateSource && line.Currency == entry.Need.Source:
			kinds[position] = ResultConverted
		default:
			return Result{}, ErrRejected
		}
	}
	out := Result{Label: Label, Snapshot: id, ContentDigest: found.snapshot.ContentDigest, Digest: found.snapshot.Digest, Lines: make([]Converted, 0, len(lines))}
	for position, line := range lines {
		at := index[line.Need]
		entry := found.snapshot.Entries[at]
		result := Converted{Line: line.ID, Kind: kinds[position]}
		switch kinds[position] {
		case ResultSameCurrency, ResultAlreadyInTarget:
			result.Currency, result.Amount = line.Currency, line.Amount
		case ResultHeld:
			result.Reason = entry.Reason
			out.Held++
		default:
			rated := found.rates[at]
			exactAmount := amounts[position].Times(rated.sales)
			converted, err := r.round(exactAmount, rated.scale, entry.AmountMode)
			if err != nil {
				return Result{}, err
			}
			atQuoted, err := r.round(amounts[position].Times(rated.perSource), rated.scale, entry.AmountMode)
			if err != nil {
				return Result{}, err
			}
			fee, err := r.round(exactAmount.Times(rated.fee), rated.scale, entry.AmountMode)
			if err != nil {
				return Result{}, err
			}
			total, err := decimal.New(r.profile, new(big.Int).Add(converted, fee), rated.scale)
			if err != nil {
				return Result{}, ErrLimit
			}
			result.Currency, result.Exact = entry.Need.Target, exactAmount.Text()
			result.Amount, result.AtQuoted, result.Fee, result.Total = r.text(converted, rated.scale), r.text(atQuoted, rated.scale), r.text(fee, rated.scale), total.String()
		}
		out.Lines = append(out.Lines, result)
	}
	return out, nil
}

func (r *Registry) round(value exact.Fraction, scale int, mode string) (*big.Int, error) {
	if value.Digits() > r.limits.Digits {
		return nil, ErrLimit
	}
	units, _ := value.Quanta(scale, mode)
	if _, err := decimal.New(r.profile, units, scale); err != nil {
		return nil, ErrLimit
	}
	return units, nil
}

func (r *Registry) text(units *big.Int, scale int) string {
	value, _ := decimal.New(r.profile, units, scale)
	return value.String()
}
