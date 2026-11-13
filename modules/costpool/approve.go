package costpool

import (
	"errors"
	"sort"

	"github.com/myblueton-sw/hybrid_billing/modules/allocation"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

type sums struct {
	total     decimal.Decimal
	byKind    map[Kind]decimal.Decimal
	byAsset   map[string]decimal.Decimal
	byGrade   map[Grade]decimal.Decimal
	poolLevel decimal.Decimal
	hasPool   bool
	summed    int
	unknown   int
}

func summed(made *component) bool { return made.spec.IncludedIn == "" }

func (r *Registry) add(target *decimal.Decimal, amount decimal.Decimal) bool {
	sum, err := decimal.Add(r.profile, *target, amount)
	*target = sum
	return err == nil
}

func (r *Registry) total(held *version) (sums, bool) {
	out := sums{total: r.zero, poolLevel: r.zero, byKind: map[Kind]decimal.Decimal{}, byAsset: map[string]decimal.Decimal{}, byGrade: map[Grade]decimal.Decimal{}}
	fits := true
	for _, made := range held.components {
		if !summed(made) {
			continue
		}
		out.summed++
		if !made.known {
			out.unknown++
			continue
		}
		fits = r.add(&out.total, made.amount) && fits
		kind, ok := out.byKind[made.spec.Kind]
		if !ok {
			kind = r.zero
		}
		fits = r.add(&kind, made.amount) && fits
		out.byKind[made.spec.Kind] = kind
		if made.spec.Asset == "" {
			out.hasPool = true
			fits = r.add(&out.poolLevel, made.amount) && fits
		} else {
			owned, ok := out.byAsset[made.spec.Asset]
			if !ok {
				owned = r.zero
			}
			fits = r.add(&owned, made.amount) && fits
			out.byAsset[made.spec.Asset] = owned
		}
		if made.spec.Power != nil {
			graded, ok := out.byGrade[made.spec.Power.Grade]
			if !ok {
				graded = r.zero
			}
			fits = r.add(&graded, made.amount) && fits
			out.byGrade[made.spec.Power.Grade] = graded
		}
	}
	return out, fits
}

func (r *Registry) quantum(held *version) decimal.Decimal {
	value, _ := r.number(held.spec.Quantum)
	return value
}

func (held *version) without() int {
	count := 0
	for _, owned := range held.assets {
		if !owned.component {
			count++
		}
	}
	return count
}

func (r *Registry) overlapping(held *version) []int {
	var out []int
	for number, other := range held.home.versions {
		if number != held.spec.Supersedes && other.current() && held.overlaps(other) {
			out = append(out, number)
		}
	}
	sort.Ints(out)
	return out
}

func (r *Registry) RecordApproval(workspaceID, name string, number int, approval Approval) (State, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(name) || number < 1 || !validID(approval.Approver) || !validID(approval.Reference) {
		return "", ErrInvalid
	}
	held := r.find(workspaceID, name, number)
	if held == nil {
		return "", ErrNotFound
	}
	if held.recorded {
		if held.approval != approval {
			return "", ErrConflict
		}
		return held.state, nil
	}
	if held.state != Draft {
		return "", ErrRejected
	}
	figures, fits := r.total(held)
	if figures.summed == 0 || figures.unknown > 0 || held.without() > 0 || held.basis == nil {
		return "", ErrRejected
	}
	var named *version
	if held.spec.Supersedes != 0 {
		if named = held.home.versions[held.spec.Supersedes]; named.superseded {
			return "", ErrRejected
		}
	}
	if len(r.overlapping(held)) > 0 {
		return "", ErrRejected
	}
	if !fits {
		return "", ErrLimit
	}
	quantum := r.quantum(held)
	if figures.total.Scale() > quantum.Scale() {
		return "", ErrRejected
	}
	targets := make([]allocation.Target, 0, len(held.basis.spec.Targets))
	for _, target := range held.basis.spec.Targets {
		targets = append(targets, allocation.Target{ID: target.ID, Weight: held.basis.weights[target.ID]})
	}
	result, err := allocation.LargestRemainderV1(allocation.Request{Profile: r.profile, Limits: allocation.Limits{MaxTargets: r.limits.Targets, MaxIntermediateDigits: r.limits.Intermediate},
		Encoding: allocation.EncodingASCIIBytewiseV1, Total: figures.total, Quantum: quantum, Mode: decimal.HalfEven, Targets: targets})
	switch {
	case errors.Is(err, decimal.ErrResultBound) || errors.Is(err, decimal.ErrIntermediateBound) || errors.Is(err, allocation.ErrIntermediateBound):
		return "", ErrLimit
	case err != nil || len(result.Shares) != len(targets):
		return "", ErrSelfCheck
	}
	stored := &Figures{Total: figures.total.String(), ByKind: map[Kind]string{}, ByGrade: map[Grade]string{}, ByBearer: map[Bearer]string{}, AssetSums: len(figures.byAsset), HasPoolLevel: figures.hasPool}
	shares := make([]Share, len(result.Shares))
	classes := map[Bearer]decimal.Decimal{}
	all, classed, idle := r.zero, r.zero, r.zero
	sound := true
	for index, target := range held.basis.spec.Targets {
		share := result.Shares[index]
		if share.ID != target.ID || share.Amount.Sign() < 0 {
			return "", ErrSelfCheck
		}
		shares[index] = Share{Target: target.ID, Bearer: target.Bearer, Weight: target.Weight, Idle: target.Idle, Amount: share.Amount.String()}
		class, ok := classes[target.Bearer]
		if !ok {
			class = r.zero
		}
		sound = r.add(&class, share.Amount) && r.add(&all, share.Amount) && sound
		classes[target.Bearer] = class
		if target.Idle {
			stored.HasIdle = true
			sound = r.add(&idle, share.Amount) && sound
		}
	}
	for class, sum := range classes {
		stored.ByBearer[class] = sum.String()
		sound = r.add(&classed, sum) && sound
	}
	if order, err := decimal.Cmp(r.profile, all, figures.total); !sound || err != nil || order != 0 {
		return "", ErrSelfCheck
	}
	if order, err := decimal.Cmp(r.profile, classed, figures.total); err != nil || order != 0 {
		return "", ErrSelfCheck
	}
	for kind, sum := range figures.byKind {
		stored.ByKind[kind] = sum.String()
	}
	for grade, sum := range figures.byGrade {
		stored.ByGrade[grade] = sum.String()
	}
	if figures.hasPool {
		stored.PoolLevel = figures.poolLevel.String()
	}
	if stored.HasIdle {
		stored.Idle = idle.String()
	}
	for _, made := range held.components {
		switch {
		case summed(made):
		case made.known:
			stored.IncludedKnown++
		default:
			stored.IncludedNotKnown++
		}
	}
	for _, owned := range held.assets {
		if owned.spec.Acquisition != "" {
			stored.PricesStated++
		} else {
			stored.PricesNotStated++
		}
	}
	held.assetSums = make([]AssetSum, 0, len(figures.byAsset))
	for id, sum := range figures.byAsset {
		held.assetSums = append(held.assetSums, AssetSum{Asset: id, Sum: sum.String()})
	}
	sort.Slice(held.assetSums, func(a, b int) bool { return held.assetSums[a].Asset < held.assetSums[b].Asset })
	held.figures, held.shares, held.approval = stored, shares, approval
	held.state, held.recorded = ApprovalRecorded, true
	if named != nil {
		named.superseded, named.supersededBy, named.changed = true, number, r.now
	}
	held.touch(r)
	return held.state, nil
}

func (r *Registry) Withdraw(workspaceID, name string, number int, reason WithdrawReason) error {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(name) || number < 1 || (reason != EnteredInError && reason != ByOperator) {
		return ErrInvalid
	}
	held := r.find(workspaceID, name, number)
	if held == nil {
		return ErrNotFound
	}
	if held.state == Withdrawn {
		if held.reason != reason {
			return ErrConflict
		}
		return nil
	}
	held.state, held.reason = Withdrawn, reason
	held.touch(r)
	return nil
}
