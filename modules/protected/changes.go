package protected

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"strconv"
	"time"
)

func (c Change) valid() bool {
	return validID(c.ID) && validID(c.Subject) && fields[c.Field] && c.After.valid() && provenances[c.Provenance] &&
		inbound[c.Instruction.Kind] && (c.Instruction.Channel == "" || validID(c.Instruction.Channel)) &&
		(c.Instruction.Kind == ChannelInPerson && c.Instruction.Sender == "" || c.Instruction.Kind != ChannelInPerson && validDigest(c.Instruction.Sender)) && validID(c.Instruction.Evidence) && !c.Effective.IsZero()
}

func hashOf(texts ...string) string {
	sum := sha256.New()
	for _, text := range texts {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(text)))
		sum.Write(size[:])
		sum.Write([]byte(text))
	}
	return hex.EncodeToString(sum.Sum(nil))
}

func stamp(t time.Time) string { return t.Format(time.RFC3339Nano) }

func span(d time.Duration) string { return strconv.FormatInt(int64(d), 10) }

func requestDigest(workspace string, v View) string {
	c := v.Change
	return hashOf(
		changeLabel, workspace, c.ID, c.Subject, c.Field,
		v.Before.ID, v.Before.Fingerprint, c.After.ID, c.After.Fingerprint,
		c.Provenance, c.Instruction.Kind, c.Instruction.Channel, c.Instruction.Sender, c.Instruction.Evidence,
		stamp(c.Effective), v.Requester, stamp(v.Requested),
		strconv.Itoa(v.Revision), strconv.Itoa(v.Rules.Approvals),
		span(v.Rules.Validity), span(v.Rules.MaxLead), span(v.Rules.ChannelAge),
	)
}

func confirmDigest(request string, v VerificationView) string {
	return hashOf(confirmLabel, request, v.Verifier, v.Verification.Channel, v.Verification.Evidence, v.Verification.Outcome, stamp(v.At))
}

func open(state string) bool {
	return state == StateRequested || state == StateConfirmed
}

func (c *change) deadline() time.Time {
	end := c.view.Change.Effective
	if c.view.State == StateConfirmed {
		if expiry := c.view.Verification.At.Add(c.view.Rules.Validity); expiry.Before(end) {
			end = expiry
		}
	}
	return end
}

func (c *change) settle(now time.Time) {
	if open(c.view.State) {
		if end := c.deadline(); now.After(end) {
			c.view.State, c.view.Ended = StateLapsed, end
		}
	}
}

func (s *space) settle(k key, now time.Time) []*change {
	list := s.byField[k]
	for _, c := range list {
		c.settle(now)
	}
	return list
}

func (s *space) latest(list []*change) *change {
	var last *change
	for _, c := range list {
		if c.view.State == StateScheduled {
			last = c
		}
	}
	return last
}

func (s *space) undeclared(subject, sender string) bool {
	for _, ch := range s.holders[subject] {
		if ch.Channel.Ref.Fingerprint == sender {
			return true
		}
	}
	return false
}

func (r *Registry) Request(workspace string, c Change, actor Actor) (View, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !c.valid() || !actor.valid() {
		return View{}, ErrInvalid
	}
	c.Effective = c.Effective.UTC().Round(0)
	s := r.workspaces[workspace]
	if s != nil {
		if seen := s.changes[c.ID]; seen != nil {
			if seen.view.Change == c && actor.is(seen.view.Requester) {
				seen.settle(now)
				return copyView(seen.view), nil
			}
			return View{}, ErrConflict
		}
	}
	if err := actor.refusal(); err != nil {
		return View{}, err
	}
	if c.Provenance == ProvenanceAssistant {
		return View{}, refuse(ReasonLLMGenerated)
	}
	if s == nil || s.active == nil {
		return View{}, refuse(ReasonNoRules)
	}
	if name := c.Instruction.Channel; name != "" {
		ch := s.channels[name]
		if ch == nil || ch.Channel.Subject != c.Subject || ch.Channel.Kind != c.Instruction.Kind || ch.Channel.Ref.Fingerprint != c.Instruction.Sender {
			return View{}, refuse(ReasonUnknownChannel)
		}
	} else if c.Instruction.Sender != "" && s.undeclared(c.Subject, c.Instruction.Sender) {
		return View{}, refuse(ReasonUndeclaredChannel)
	}
	k := key{c.Subject, c.Field}
	list := s.settle(k, now)
	for _, other := range list {
		if open(other.view.State) {
			return View{}, refuse(ReasonChangePending)
		}
	}
	rules := s.active.Rules
	if !c.Effective.After(now) || c.Effective.After(now.Add(rules.MaxLead)) {
		return View{}, refuse(ReasonEffectiveOutOfRange)
	}
	var before Ref
	if last := s.latest(list); last != nil {
		if !c.Effective.After(last.view.Change.Effective) {
			return View{}, refuse(ReasonOutOfOrder)
		}
		before = last.view.Change.After
	}
	if before == c.After {
		return View{}, refuse(ReasonNoChange)
	}
	if len(s.changes) >= r.limits.Changes {
		return View{}, ErrLimit
	}
	made := &change{approved: map[string]bool{}}
	made.view = View{Change: c, State: StateRequested, Before: before, Requester: actor.ID, Requested: now, Rules: rules, Revision: s.active.Revision}
	made.view.Digest = requestDigest(workspace, made.view)
	s.changes[c.ID] = made
	s.byField[k] = append(s.byField[k], made)
	return copyView(made.view), nil
}

func (r *Registry) lookup(workspace, id string, now time.Time) (*space, *change, error) {
	s := r.workspaces[workspace]
	if s == nil || s.changes[id] == nil {
		return nil, nil, ErrNotFound
	}
	c := s.changes[id]
	c.settle(now)
	return s, c, nil
}

func (r *Registry) Verify(workspace, id string, v Verification, actor Actor) (View, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !validID(v.Channel) || !validID(v.Evidence) || !outcomes[v.Outcome] || !validDigest(v.Digest) || !actor.valid() {
		return View{}, ErrInvalid
	}
	s, c, err := r.lookup(workspace, id, now)
	if err != nil {
		return View{}, err
	}
	if seen := c.view.Verification; seen != nil && seen.Verification == v && actor.is(seen.Verifier) {
		return copyView(c.view), nil
	}
	if err := actor.refusal(); err != nil {
		return View{}, err
	}
	switch c.view.State {
	case StateLapsed:
		return View{}, refuse(ReasonLapsed)
	case StateRequested:
	default:
		return View{}, refuse(ReasonNotPending)
	}
	if v.Digest != c.view.Digest {
		return View{}, refuse(ReasonDigestChanged)
	}
	if actor.ID == c.view.Requester {
		return View{}, refuse(ReasonSelfVerification)
	}
	ch := s.channels[v.Channel]
	if ch == nil || ch.Channel.Subject != c.view.Change.Subject {
		return View{}, refuse(ReasonUnknownChannel)
	}
	if ch.Admitter == "" {
		return View{}, refuse(ReasonChannelNotAdmitted)
	}
	if v.Channel == c.view.Change.Instruction.Channel || ch.Channel.Ref.Fingerprint == c.view.Change.Instruction.Sender {
		return View{}, refuse(ReasonSameChannel)
	}
	if ch.RetiredBy != "" {
		return View{}, refuse(ReasonChannelRetired)
	}
	if ch.Admitted.After(c.view.Requested.Add(-c.view.Rules.ChannelAge)) {
		return View{}, refuse(ReasonChannelTooNew)
	}
	if ch.custodian(c.view.Requester) || ch.custodian(actor.ID) {
		return View{}, refuse(ReasonSelfChannel)
	}
	seen := &VerificationView{Verification: v, Verifier: actor.ID, At: now}
	seen.Digest = confirmDigest(c.view.Digest, *seen)
	c.view.Verification = seen
	if v.Outcome == OutcomeDenied {
		c.view.State, c.view.EndedBy, c.view.Ended = StateRejected, actor.ID, now
	} else {
		c.view.State = StateConfirmed
	}
	return copyView(c.view), nil
}

func (r *Registry) Approve(workspace, id string, a Approval, actor Actor) (View, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !validDigest(a.Digest) || !actor.valid() {
		return View{}, ErrInvalid
	}
	if !(a.Verdict == VerdictApprove && a.Reason == "" || a.Verdict == VerdictReject && rejectReasons[a.Reason]) {
		return View{}, ErrInvalid
	}
	s, c, err := r.lookup(workspace, id, now)
	if err != nil {
		return View{}, err
	}
	for _, seen := range c.view.Decisions {
		if actor.is(seen.Approver) && seen.Approval == a {
			return copyView(c.view), nil
		}
	}
	if err := actor.refusal(); err != nil {
		return View{}, err
	}
	switch c.view.State {
	case StateLapsed:
		return View{}, refuse(ReasonLapsed)
	case StateRequested:
		return View{}, refuse(ReasonNotConfirmed)
	case StateConfirmed:
	default:
		return View{}, refuse(ReasonNotPending)
	}
	if a.Digest != c.view.Verification.Digest {
		return View{}, refuse(ReasonDigestChanged)
	}
	ch := s.channels[c.view.Verification.Verification.Channel]
	if ch.RetiredBy != "" {
		return View{}, refuse(ReasonChannelRetired)
	}
	if actor.ID == c.view.Requester || actor.ID == c.view.Verification.Verifier || ch.custodian(actor.ID) {
		return View{}, refuse(ReasonSelfApproval)
	}
	if c.approved[actor.ID] {
		return View{}, refuse(ReasonAlreadyDecided)
	}
	c.view.Decisions = append(c.view.Decisions, DecisionView{Approval: a, Approver: actor.ID, At: now})
	if a.Verdict == VerdictReject {
		c.view.State, c.view.EndedBy, c.view.Ended = StateRejected, actor.ID, now
		return copyView(c.view), nil
	}
	c.approved[actor.ID] = true
	if len(c.approved) >= c.view.Rules.Approvals {
		c.view.State, c.view.Scheduled = StateScheduled, now
	}
	return copyView(c.view), nil
}

func (r *Registry) Withdraw(workspace, id string, actor Actor) (View, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(id) || !actor.valid() {
		return View{}, ErrInvalid
	}
	s, c, err := r.lookup(workspace, id, now)
	if err != nil {
		return View{}, err
	}
	if c.view.State == StateWithdrawn && actor.is(c.view.EndedBy) {
		return copyView(c.view), nil
	}
	if err := actor.refusal(); err != nil {
		return View{}, err
	}
	list := s.settle(key{c.view.Change.Subject, c.view.Change.Field}, now)
	switch c.view.State {
	case StateRequested, StateConfirmed:
	case StateScheduled:
		if !now.Before(c.view.Change.Effective) {
			return View{}, refuse(ReasonAlreadyEffective)
		}
		for index := len(list) - 1; list[index] != c; index-- {
			if open(list[index].view.State) || list[index].view.State == StateScheduled {
				return View{}, refuse(ReasonHasSuccessor)
			}
		}
	default:
		return View{}, refuse(ReasonNotPending)
	}
	c.view.State, c.view.EndedBy, c.view.Ended = StateWithdrawn, actor.ID, now
	return copyView(c.view), nil
}

func copyView(v View) View {
	if v.Verification != nil {
		seen := *v.Verification
		v.Verification = &seen
	}
	v.Decisions = append([]DecisionView(nil), v.Decisions...)
	return v
}

func (r *Registry) Of(workspace, id string) (View, error) {
	now := r.enter()
	defer r.leave()
	_, c, err := r.lookup(workspace, id, now)
	if err != nil {
		return View{}, err
	}
	return copyView(c.view), nil
}

func (r *Registry) Trail(workspace, subject, field string) ([]View, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(subject) || !fields[field] {
		return nil, ErrInvalid
	}
	s := r.workspaces[workspace]
	if s == nil {
		return nil, ErrNotFound
	}
	list := s.settle(key{subject, field}, now)
	out := make([]View, 0, len(list))
	for _, c := range list {
		out = append(out, copyView(c.view))
	}
	return out, nil
}

func (r *Registry) Resolve(workspace, subject, field string, at time.Time) (Resolution, error) {
	r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(subject) || !fields[field] || at.IsZero() {
		return Resolution{}, ErrInvalid
	}
	s := r.workspaces[workspace]
	if s == nil {
		return Resolution{}, ErrNotFound
	}
	var found *change
	for _, c := range s.byField[key{subject, field}] {
		if c.view.State == StateScheduled && !c.view.Change.Effective.After(at) {
			found = c
		}
	}
	if found == nil {
		return Resolution{}, ErrNotFound
	}
	return Resolution{Change: found.view.Change.ID, Ref: found.view.Change.After, Effective: found.view.Change.Effective}, nil
}
