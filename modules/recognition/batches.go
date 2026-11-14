package recognition

import (
	"math/big"
	"strconv"
	"time"
)

func (b *batch) posted() bool {
	for _, sub := range b.subs {
		if sub.Result.State == ResultPosted {
			return true
		}
	}
	return false
}

func (b *batch) external() string {
	for _, sub := range b.subs {
		if sub.Result.State == ResultPosted {
			return sub.Result.External
		}
	}
	return ""
}

func (b *batch) gone() bool {
	return b.view.State == BatchVoid || b.view.State == BatchAbandoned
}

func (b *batch) carrying() bool {
	return b.carrier != nil && b.carrier.live()
}

func (b *batch) reversed() bool {
	return b.reverse != nil && b.reverse.posted()
}

func (b *batch) pending() bool {
	for _, sub := range b.subs {
		switch sub.Result.State {
		case ResultSent, ResultAccepted, ResultUnknown:
			return true
		}
	}
	return false
}

func (b *batch) copy() BatchView {
	v := b.view
	v.Batch.Lines = append([]Line(nil), v.Batch.Lines...)
	v.Posted = b.posted()
	v.External = b.external()
	v.ReversedBy, v.Carries = "", ""
	if b.reverse != nil {
		v.ReversedBy = b.reverse.view.Batch.ID
	}
	if b.carrying() {
		v.Carries = b.carrier.view.Adjustment.ID
	}
	return v
}

func (s *space) liveInitial(k monthKey) *batch {
	for _, b := range s.byMonth[k] {
		if b.view.Batch.Kind == BatchInitial && !b.gone() && !b.reversed() {
			return b
		}
	}
	return nil
}

func sameBatch(a, b Batch) bool {
	if a.ID != b.ID || a.Profile != b.Profile || a.Month != b.Month || a.Posting != b.Posting || a.Kind != b.Kind ||
		a.Reverses != b.Reverses || a.Provisional != b.Provisional || a.Snapshot != b.Snapshot || len(a.Lines) != len(b.Lines) {
		return false
	}
	for index := range a.Lines {
		if a.Lines[index] != b.Lines[index] {
			return false
		}
	}
	return true
}

func (r *Registry) batchForm(b Batch) bool {
	if !validID(b.ID) || !validID(b.Profile) || !validMonth(b.Month) || !validMonth(b.Posting) || !kinds[b.Kind] || !validDigest(b.Snapshot) {
		return false
	}
	if (b.Kind == BatchReversal) != (b.Reverses != "") || b.Reverses != "" && !validID(b.Reverses) {
		return false
	}
	if b.Kind == BatchReversal && len(b.Lines) > 0 {
		return false
	}
	seen := map[string]bool{}
	for _, line := range b.Lines {
		if !validID(line.ID) || seen[line.ID] || !validID(line.Lineage) || !r.number(line.Net) || !r.number(line.Tax) {
			return false
		}
		seen[line.ID] = true
	}
	return true
}

func batchDigest(workspace string, v BatchView) string {
	b := v.Batch
	texts := []string{batchLabel, workspace, b.ID, b.Profile, b.Month, b.Posting, b.Kind, b.Reverses, flag(b.Provisional), b.Snapshot, strconv.Itoa(len(b.Lines))}
	for _, line := range b.Lines {
		texts = append(texts, line.ID, line.Lineage, line.Net, line.Tax)
	}
	texts = append(texts, v.Total.Net, v.Total.Tax, v.Author, v.Opened.Format(time.RFC3339Nano))
	return hashOf(texts...)
}

func (r *Registry) Open(workspace string, b Batch, actor Actor) (BatchView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !r.batchForm(b) || !actor.valid() {
		return BatchView{}, ErrInvalid
	}
	s := r.workspace(workspace)
	if s != nil {
		if seen := s.batches[b.ID]; seen != nil {
			if sameBatch(seen.view.Batch, b) && actor.is(seen.view.Author) {
				return seen.copy(), nil
			}
			return BatchView{}, ErrConflict
		}
	}
	if err := actor.refusal(); err != nil {
		return BatchView{}, err
	}
	if s == nil || s.profiles[b.Profile] == nil {
		return BatchView{}, refuse(ReasonUnknownProfile)
	}
	p := s.profiles[b.Profile]
	scale := p.view.Profile.Scale
	net, tax := new(big.Int), new(big.Int)
	for _, line := range b.Lines {
		n, ok := r.units(line.Net, scale)
		x, fine := r.units(line.Tax, scale)
		if !ok || !fine {
			return BatchView{}, ErrInvalid
		}
		net.Add(net, n)
		tax.Add(tax, x)
	}
	k := monthKey{b.Profile, b.Month}
	var target *batch
	if b.Kind == BatchReversal {
		target = s.batches[b.Reverses]
	}
	if b.Provisional && !p.view.Profile.Provisional {
		return BatchView{}, refuse(ReasonProvisionalNotAllowed)
	}
	if s.cleared[k] != "" {
		return BatchView{}, refuse(ReasonMonthCleared)
	}
	if p.closed[b.Posting] {
		return BatchView{}, refuse(ReasonPeriodClosed)
	}
	if b.Kind == BatchInitial && b.Posting != b.Month || b.Kind != BatchInitial && b.Posting < b.Month {
		return BatchView{}, refuse(ReasonPostingMismatch)
	}
	live := s.liveInitial(k)
	if b.Kind == BatchInitial {
		if live != nil {
			return BatchView{}, refuse(ReasonInitialExists)
		}
	} else if live == nil || live.view.State != BatchApproved || b.Kind == BatchAmendment && !live.posted() {
		return BatchView{}, refuse(ReasonNoInitial)
	} else if b.Kind == BatchAmendment && live.reverse != nil {
		return BatchView{}, refuse(ReasonInitialReversing)
	}
	if b.Kind == BatchReversal {
		if err := s.reversible(target, k, b.Provisional); err != nil {
			return BatchView{}, err
		}
		net.Neg(target.net)
		tax.Neg(target.tax)
	} else if len(b.Lines) == 0 || b.Kind == BatchAmendment && net.Sign() == 0 && tax.Sign() == 0 {
		return BatchView{}, refuse(ReasonEmpty)
	}
	if !r.bounded(net, scale) || !r.bounded(tax, scale) {
		return BatchView{}, ErrLimit
	}
	if len(s.batches) >= r.limits.Batches || len(b.Lines) > r.limits.Lines {
		return BatchView{}, ErrLimit
	}
	b.Lines = append([]Line(nil), b.Lines...)
	made := &batch{net: net, tax: tax, target: target}
	made.view = BatchView{Batch: b, State: BatchOpen, Total: Amounts{Net: show(net, scale), Tax: show(tax, scale)}, Author: actor.ID, Opened: now}
	made.view.Digest = batchDigest(workspace, made.view)
	if target != nil {
		target.reverse = made
	}
	s.batches[b.ID] = made
	s.byMonth[k] = append(s.byMonth[k], made)
	return made.copy(), nil
}

func (s *space) reversible(target *batch, k monthKey, provisional bool) error {
	if target == nil || target.view.Batch.Profile != k.profile || target.view.Batch.Month != k.month {
		return refuse(ReasonUnknownTarget)
	}
	if target.view.Batch.Provisional != provisional {
		return ErrInvalid
	}
	if target.view.Batch.Kind == BatchReversal {
		return refuse(ReasonTargetIsReversal)
	}
	if !target.posted() {
		return refuse(ReasonTargetNotPosted)
	}
	if target.reverse != nil {
		return refuse(ReasonAlreadyReversed)
	}
	if target.carrying() {
		return refuse(ReasonCarriesAdjustment)
	}
	if target.view.Batch.Kind == BatchInitial {
		for _, other := range s.byMonth[k] {
			if other.view.Batch.Kind == BatchAmendment && !other.gone() && !other.reversed() {
				return refuse(ReasonHasLiveAmendments)
			}
		}
	}
	return nil
}

func (s *space) release(b *batch) {
	if b.target != nil && b.target.reverse == b {
		b.target.reverse = nil
	}
}

func (r *Registry) Approve(workspace, id, digest string, actor Actor) (BatchView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !validDigest(digest) || !actor.valid() {
		return BatchView{}, ErrInvalid
	}
	s := r.workspace(workspace)
	if s == nil || s.batches[id] == nil {
		return BatchView{}, ErrNotFound
	}
	b := s.batches[id]
	if b.view.Approver != "" && actor.is(b.view.Approver) && digest == b.view.Digest {
		return b.copy(), nil
	}
	if err := actor.refusal(); err != nil {
		return BatchView{}, err
	}
	if b.view.State != BatchOpen {
		return BatchView{}, refuse(ReasonNotPending)
	}
	if digest != b.view.Digest {
		return BatchView{}, refuse(ReasonDigestChanged)
	}
	if actor.ID == b.view.Author {
		return BatchView{}, refuse(ReasonSelfApproval)
	}
	if s.profiles[b.view.Batch.Profile].closed[b.view.Batch.Posting] {
		return BatchView{}, refuse(ReasonPeriodClosed)
	}
	b.view.State, b.view.Approver, b.view.Approved = BatchApproved, actor.ID, now
	return b.copy(), nil
}

func (r *Registry) Void(workspace, id string, actor Actor) (BatchView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !actor.valid() {
		return BatchView{}, ErrInvalid
	}
	s := r.workspace(workspace)
	if s == nil || s.batches[id] == nil {
		return BatchView{}, ErrNotFound
	}
	b := s.batches[id]
	if b.view.State == BatchVoid && actor.is(b.view.EndedBy) {
		return b.copy(), nil
	}
	if err := actor.refusal(); err != nil {
		return BatchView{}, err
	}
	if b.view.State != BatchOpen {
		return BatchView{}, refuse(ReasonNotPending)
	}
	if b.carrying() {
		return BatchView{}, refuse(ReasonCarriesAdjustment)
	}
	b.view.State, b.view.EndedBy, b.view.Ended = BatchVoid, actor.ID, now
	s.release(b)
	return b.copy(), nil
}

func (r *Registry) Abandon(workspace, id string, actor Actor) (BatchView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !actor.valid() {
		return BatchView{}, ErrInvalid
	}
	s := r.workspace(workspace)
	if s == nil || s.batches[id] == nil {
		return BatchView{}, ErrNotFound
	}
	b := s.batches[id]
	if b.view.State == BatchAbandoned && actor.is(b.view.EndedBy) {
		return b.copy(), nil
	}
	if err := actor.refusal(); err != nil {
		return BatchView{}, err
	}
	if b.view.State != BatchApproved || b.posted() {
		return BatchView{}, refuse(ReasonNotPending)
	}
	if b.pending() {
		return BatchView{}, refuse(ReasonOutcomeUnknown)
	}
	if b.carrying() {
		return BatchView{}, refuse(ReasonCarriesAdjustment)
	}
	if actor.ID == b.view.Author {
		return BatchView{}, refuse(ReasonSelfApproval)
	}
	b.view.State, b.view.EndedBy, b.view.Ended = BatchAbandoned, actor.ID, now
	s.release(b)
	return b.copy(), nil
}

func (r *Registry) BatchOf(workspace, id string) (BatchView, error) {
	r.enter()
	defer r.leave()
	s := r.workspace(workspace)
	if s == nil || s.batches[id] == nil {
		return BatchView{}, ErrNotFound
	}
	return s.batches[id].copy(), nil
}
