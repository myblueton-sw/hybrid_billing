package sharedpool

import (
	"errors"
	"sort"

	"github.com/myblueton-sw/hybrid_billing/modules/allocation"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func (held *run) entries() []staged {
	out := make([]staged, 0, len(held.targets))
	for _, part := range held.chunks {
		out = append(out, part.weights...)
	}
	return out
}

func (held *run) unknown() int {
	count := 0
	for _, part := range held.chunks {
		for _, entry := range part.weights {
			if !entry.known {
				count++
			}
		}
	}
	return count
}

func (r *Registry) same(a, b decimal.Decimal) bool {
	order, err := decimal.Cmp(r.profile, a, b)
	return err == nil && order == 0
}

func (r *Registry) store(held *run, shares []Share, amounts []decimal.Decimal) bool {
	classes := map[Bearer]decimal.Decimal{}
	all, classed, others := r.zero, r.zero, r.zero
	residual, has := r.zero, false
	var err error
	for index, share := range shares {
		class, known := classes[share.Bearer]
		if !known {
			class = r.zero
		}
		if class, err = decimal.Add(r.profile, class, amounts[index]); err != nil {
			return false
		}
		classes[share.Bearer] = class
		if all, err = decimal.Add(r.profile, all, amounts[index]); err != nil {
			return false
		}
		if share.Residual {
			residual, has = amounts[index], true
			continue
		}
		if others, err = decimal.Add(r.profile, others, amounts[index]); err != nil {
			return false
		}
	}
	byBearer := map[Bearer]string{}
	for class, sum := range classes {
		byBearer[class] = sum.String()
		if classed, err = decimal.Add(r.profile, classed, sum); err != nil {
			return false
		}
	}
	whole, err := decimal.Add(r.profile, others, residual)
	if err != nil || !r.same(all, held.home.amount) || !r.same(classed, held.home.amount) || !r.same(whole, held.home.amount) {
		return false
	}
	held.shares, held.byBearer = shares, byBearer
	if has {
		held.residual = residual.String()
	}
	return true
}

func (r *Registry) Seal(workspaceID, name string, number int, id string) (State, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(name) || number < 1 || !validID(id) {
		return "", ErrInvalid
	}
	held := r.lookup(workspaceID, name, number, id)
	if held == nil {
		return "", ErrNotFound
	}
	if held.judged {
		return held.state, nil
	}
	if held.state != Staging || len(held.chunks) != held.spec.Chunks {
		return "", ErrRejected
	}
	home := held.home
	if held.unknown() > 0 {
		held.state, held.judged, held.hold = Held, true, WeightNotKnown
		held.touch(r)
		return Held, nil
	}
	entries := held.entries()
	sum := r.zero
	var err error
	for _, entry := range entries {
		if sum, err = decimal.Add(r.profile, sum, entry.value); err != nil {
			return "", ErrLimit
		}
	}
	denominator := sum
	stated := home.spec.Capacity != ""
	if stated {
		denominator = home.capacity
	}
	over := false
	if stated {
		order, err := decimal.Cmp(r.profile, sum, home.capacity)
		if err != nil {
			return "", ErrSelfCheck
		}
		over = order > 0
	}
	if (!stated && sum.Sign() == 0) || over {
		held.state, held.judged, held.hold, held.weightSum = Held, true, ZeroDenominator, sum.String()
		if over {
			held.hold = OverCapacity
		}
		held.touch(r)
		return Held, nil
	}
	targets := make([]allocation.Target, 0, len(entries)+1)
	shares := make([]Share, 0, len(entries)+1)
	for _, entry := range entries {
		targets = append(targets, allocation.Target{ID: entry.weight.Target, Weight: entry.value})
		shares = append(shares, Share{Target: entry.weight.Target, Bearer: entry.weight.Bearer, Weight: entry.weight.Weight})
	}
	why := ""
	if stated && !r.same(sum, home.capacity) {
		unused, err := decimal.Sub(r.profile, home.capacity, sum)
		if err != nil {
			return "", ErrSelfCheck
		}
		targets = append(targets, allocation.Target{ID: residualID, Weight: unused})
		shares = append(shares, Share{Target: residualID, Bearer: home.spec.ResidualBearer, Weight: unused.String(), Residual: true})
		why = BelowCapacity
	}
	result, err := allocation.LargestRemainderV1(allocation.Request{Profile: r.profile, Limits: allocation.Limits{MaxTargets: r.limits.Targets + 1, MaxIntermediateDigits: r.limits.Intermediate},
		Encoding: allocation.EncodingASCIIBytewiseV1, Total: home.amount, Quantum: home.quantum, Mode: decimal.HalfEven, Targets: targets})
	switch {
	case errors.Is(err, decimal.ErrResultBound) || errors.Is(err, decimal.ErrIntermediateBound) || errors.Is(err, allocation.ErrIntermediateBound):
		return "", ErrLimit
	case err != nil || len(result.Shares) != len(shares) || !r.same(result.Denominator, denominator) || result.RoundingComponent.Sign() != 0:
		return "", ErrSelfCheck
	}
	sort.Slice(shares, func(a, b int) bool { return shares[a].Target < shares[b].Target })
	amounts := make([]decimal.Decimal, len(shares))
	for index := range shares {
		share := result.Shares[index]
		if share.ID != shares[index].Target || share.Amount.Sign() < 0 {
			return "", ErrSelfCheck
		}
		shares[index].Amount, amounts[index] = share.Amount.String(), share.Amount
	}
	if !r.store(held, shares, amounts) {
		return "", ErrSelfCheck
	}
	held.state, held.judged, held.weightSum, held.denominator, held.residualWhy = Sealed, true, sum.String(), denominator.String(), why
	held.touch(r)
	return Sealed, nil
}

func (r *Registry) Settle(workspaceID, name string, number int, id string, alternative Alternative) (State, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(name) || number < 1 || !validID(id) || !validID(alternative.Approver) || !validID(alternative.Reference) || !alternative.Bearer.residual() {
		return "", ErrInvalid
	}
	held := r.lookup(workspaceID, name, number, id)
	if held == nil {
		return "", ErrNotFound
	}
	if held.alternative != nil {
		if *held.alternative != alternative {
			return "", ErrConflict
		}
		return held.state, nil
	}
	if held.state != Held || held.hold != ZeroDenominator {
		return "", ErrRejected
	}
	entries := held.entries()
	shares := make([]Share, 0, len(entries)+1)
	amounts := make([]decimal.Decimal, 0, len(entries)+1)
	for _, entry := range entries {
		shares = append(shares, Share{Target: entry.weight.Target, Bearer: entry.weight.Bearer, Weight: entry.weight.Weight, Amount: r.zero.String()})
		amounts = append(amounts, r.zero)
	}
	shares = append(shares, Share{Target: residualID, Bearer: alternative.Bearer, Weight: r.zero.String(), Amount: held.home.amount.String(), Residual: true})
	amounts = append(amounts, held.home.amount)
	order := make([]int, len(shares))
	for index := range order {
		order[index] = index
	}
	sort.Slice(order, func(a, b int) bool { return shares[order[a]].Target < shares[order[b]].Target })
	sorted, values := make([]Share, len(shares)), make([]decimal.Decimal, len(shares))
	for position, index := range order {
		sorted[position], values[position] = shares[index], amounts[index]
	}
	if !r.store(held, sorted, values) {
		return "", ErrSelfCheck
	}
	held.state, held.alternative, held.residualWhy = Sealed, &alternative, RecordedAlternative
	held.touch(r)
	return Sealed, nil
}

func (r *Registry) Publish(workspaceID, name string, number int, id string, approval Approval) (State, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(name) || number < 1 || !validID(id) || (approval.Approver != "" && !validID(approval.Approver)) || (approval.Reference != "" && !validID(approval.Reference)) {
		return "", ErrInvalid
	}
	held := r.lookup(workspaceID, name, number, id)
	if held == nil {
		return "", ErrNotFound
	}
	if held.published {
		if held.approval != approval {
			return "", ErrConflict
		}
		return held.state, nil
	}
	charged := held.spec.Purpose == Chargeback
	if held.state != Sealed || charged != (approval.Approver != "") || charged != (approval.Reference != "") {
		return "", ErrRejected
	}
	home := held.home.home
	if charged {
		for _, other := range home.charged {
			if held.home.spec.PeriodStart.Before(other.home.spec.PeriodEnd) && other.home.spec.PeriodStart.Before(held.home.spec.PeriodEnd) {
				return "", ErrRejected
			}
		}
		home.charged = append(home.charged, held)
	}
	held.state, held.published, held.approval = Published, true, approval
	held.touch(r)
	return Published, nil
}
