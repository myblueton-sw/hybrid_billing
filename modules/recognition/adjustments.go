package recognition

import (
	"math/big"
	"time"
)

func (a *adjustment) live() bool {
	return a.view.State == AdjustmentPending || a.view.State == AdjustmentApproved
}

func (a *adjustment) copy() AdjustmentView { return a.view }

func (r *Registry) adjustmentForm(a Adjustment) bool {
	return validID(a.ID) && validID(a.Profile) && validMonth(a.Month) && causes[a.Cause] &&
		r.number(a.Before.Net) && r.number(a.Before.Tax) && r.number(a.After.Net) && r.number(a.After.Tax) &&
		(a.Carrier == "" || validID(a.Carrier))
}

func adjustmentDigest(workspace string, v AdjustmentView) string {
	a := v.Adjustment
	return hashOf(adjustmentLabel, workspace, a.ID, a.Profile, a.Month, a.Cause, a.Before.Net, a.Before.Tax, a.After.Net, a.After.Tax,
		v.Delta.Net, v.Delta.Tax, v.Proposer, v.Proposed.Format(time.RFC3339Nano))
}

func (s *space) carrierFor(a *adjustment, id string) (*batch, error) {
	b := s.batches[id]
	if b == nil {
		return nil, refuse(ReasonUnknownBatch)
	}
	if b.gone() || b.reverse != nil || b.view.Batch.Kind != BatchAmendment || b.view.Batch.Profile != a.view.Adjustment.Profile ||
		b.view.Batch.Month != a.view.Adjustment.Month || b.net.Cmp(a.net) != 0 || b.tax.Cmp(a.tax) != 0 {
		return nil, refuse(ReasonCarrierMismatch)
	}
	if b.carrying() {
		return nil, ErrConflict
	}
	return b, nil
}

func (r *Registry) Propose(workspace string, a Adjustment, actor Actor) (AdjustmentView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !r.adjustmentForm(a) || !actor.valid() {
		return AdjustmentView{}, ErrInvalid
	}
	s := r.workspace(workspace)
	if s != nil {
		if seen := s.adjustments[a.ID]; seen != nil {
			if seen.view.Adjustment == a && actor.is(seen.view.Proposer) {
				return seen.copy(), nil
			}
			return AdjustmentView{}, ErrConflict
		}
	}
	if err := actor.refusal(); err != nil {
		return AdjustmentView{}, err
	}
	if s == nil || s.profiles[a.Profile] == nil {
		return AdjustmentView{}, refuse(ReasonUnknownProfile)
	}
	scale := s.profiles[a.Profile].view.Profile.Scale
	var amounts [4]*big.Int
	for index, text := range []string{a.Before.Net, a.Before.Tax, a.After.Net, a.After.Tax} {
		units, ok := r.units(text, scale)
		if !ok {
			return AdjustmentView{}, ErrInvalid
		}
		amounts[index] = units
	}
	net := new(big.Int).Sub(amounts[2], amounts[0])
	tax := new(big.Int).Sub(amounts[3], amounts[1])
	k := monthKey{a.Profile, a.Month}
	if s.cleared[k] != "" {
		return AdjustmentView{}, refuse(ReasonMonthCleared)
	}
	if net.Sign() == 0 && tax.Sign() == 0 {
		return AdjustmentView{}, refuse(ReasonEmpty)
	}
	if !r.bounded(net, scale) || !r.bounded(tax, scale) {
		return AdjustmentView{}, ErrLimit
	}
	made := &adjustment{net: net, tax: tax}
	made.view = AdjustmentView{Adjustment: a, Delta: Amounts{Net: show(net, scale), Tax: show(tax, scale)}, State: AdjustmentPending, Proposer: actor.ID, Proposed: now}
	if a.Carrier != "" {
		carrier, err := s.carrierFor(made, a.Carrier)
		if err != nil {
			return AdjustmentView{}, err
		}
		made.carrier = carrier
	}
	if len(s.adjustments) >= r.limits.Adjustments {
		return AdjustmentView{}, ErrLimit
	}
	made.view.Digest = adjustmentDigest(workspace, made.view)
	if made.carrier != nil {
		made.carrier.carrier = made
		made.view.Carrier, made.view.Linker = a.Carrier, actor.ID
	}
	s.adjustments[a.ID] = made
	s.adjByMonth[k] = append(s.adjByMonth[k], made)
	return made.copy(), nil
}

func (r *Registry) findAdjustment(workspace, id string) (*space, *adjustment, error) {
	s := r.workspace(workspace)
	if s == nil || s.adjustments[id] == nil {
		return nil, nil, ErrNotFound
	}
	return s, s.adjustments[id], nil
}

func (r *Registry) Decide(workspace, id string, d Decision, actor Actor) (AdjustmentView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || d.Verdict != VerdictApprove && d.Verdict != VerdictReject || !validDigest(d.Digest) || !actor.valid() {
		return AdjustmentView{}, ErrInvalid
	}
	_, a, err := r.findAdjustment(workspace, id)
	if err != nil {
		return AdjustmentView{}, err
	}
	verdicts := map[string]string{AdjustmentApproved: VerdictApprove, AdjustmentRejected: VerdictReject}
	if a.view.Decider != "" && actor.is(a.view.Decider) && verdicts[a.view.State] == d.Verdict && d.Digest == a.view.Digest {
		return a.copy(), nil
	}
	if err := actor.refusal(); err != nil {
		return AdjustmentView{}, err
	}
	if a.view.State != AdjustmentPending {
		return AdjustmentView{}, refuse(ReasonNotPending)
	}
	if d.Digest != a.view.Digest {
		return AdjustmentView{}, refuse(ReasonDigestChanged)
	}
	if actor.ID == a.view.Proposer {
		return AdjustmentView{}, refuse(ReasonSelfApproval)
	}
	a.view.Decider, a.view.Decided = actor.ID, now
	if d.Verdict == VerdictApprove {
		a.view.State = AdjustmentApproved
	} else {
		a.view.State, a.view.EndedBy, a.view.Ended = AdjustmentRejected, actor.ID, now
	}
	return a.copy(), nil
}

func (r *Registry) Withdraw(workspace, id string, actor Actor) (AdjustmentView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !actor.valid() {
		return AdjustmentView{}, ErrInvalid
	}
	_, a, err := r.findAdjustment(workspace, id)
	if err != nil {
		return AdjustmentView{}, err
	}
	if a.view.State == AdjustmentWithdrawn && actor.is(a.view.EndedBy) {
		return a.copy(), nil
	}
	if err := actor.refusal(); err != nil {
		return AdjustmentView{}, err
	}
	if !a.live() {
		return AdjustmentView{}, refuse(ReasonNotPending)
	}
	if a.carrier != nil && a.carrier.posted() {
		return AdjustmentView{}, refuse(ReasonCarrierPosted)
	}
	a.view.State, a.view.EndedBy, a.view.Ended = AdjustmentWithdrawn, actor.ID, now
	return a.copy(), nil
}

func (r *Registry) Link(workspace, id, carrier string, actor Actor) (AdjustmentView, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !validID(carrier) || !actor.valid() {
		return AdjustmentView{}, ErrInvalid
	}
	s, a, err := r.findAdjustment(workspace, id)
	if err != nil {
		return AdjustmentView{}, err
	}
	if a.carrier != nil {
		if a.carrier.view.Batch.ID == carrier && actor.is(a.view.Linker) {
			return a.copy(), nil
		}
		return AdjustmentView{}, ErrConflict
	}
	if err := actor.refusal(); err != nil {
		return AdjustmentView{}, err
	}
	if !a.live() {
		return AdjustmentView{}, refuse(ReasonNotPending)
	}
	b, err := s.carrierFor(a, carrier)
	if err != nil {
		return AdjustmentView{}, err
	}
	a.carrier, b.carrier = b, a
	a.view.Carrier, a.view.Linker = carrier, actor.ID
	return a.copy(), nil
}

func (r *Registry) AdjustmentOf(workspace, id string) (AdjustmentView, error) {
	r.enter()
	defer r.leave()
	_, a, err := r.findAdjustment(workspace, id)
	if err != nil {
		return AdjustmentView{}, err
	}
	return a.copy(), nil
}
