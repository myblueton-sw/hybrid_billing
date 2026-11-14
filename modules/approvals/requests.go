package approvals

import (
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/exact"
)

func (r *Registry) subjectForm(s Subject) (Subject, *request, bool) {
	if !validID(s.ID) || !stages[s.Stage] || !validID(s.Target) || !validID(s.Author) || !validCurrency(s.Currency) || len(s.Digests) == 0 || len(s.Entries) > r.limits.Items {
		return s, nil, false
	}
	for k, v := range s.Digests {
		if !digestKinds[k] || !validDigest(v) {
			return s, nil, false
		}
	}
	amount, printed, ok := r.number(s.Amount, true)
	if !ok {
		return s, nil, false
	}
	made := &request{amount: amount, entries: map[string]*entry{}, excluded: map[string]string{}, deciders: map[string]int{}, executed: map[string]string{}}
	s.Amount = printed
	s.Digests = copyDigests(s.Digests)
	s.Entries = append([]Entry(nil), s.Entries...)
	if len(s.Entries) > 0 {
		sum := exact.Whole(0)
		for index, e := range s.Entries {
			value, text, ok := r.number(e.Amount, true)
			if !ok || !validID(e.Item) || !validID(e.Scope) || !validDigest(e.Digest) || made.entries[e.Item] != nil {
				return s, nil, false
			}
			s.Entries[index].Amount = text
			made.entries[e.Item] = &entry{e: s.Entries[index], amount: value}
			sum = sum.Plus(value)
		}
		if !same(sum, amount) {
			return s, nil, false
		}
	}
	made.view.Subject = s
	return s, made, true
}

func sameSubject(a, b Subject) bool {
	if a.ID != b.ID || a.Stage != b.Stage || a.Target != b.Target || a.Author != b.Author || a.Amount != b.Amount || a.Currency != b.Currency ||
		!sameDigests(a.Digests, b.Digests) || len(a.Entries) != len(b.Entries) {
		return false
	}
	for index := range a.Entries {
		if a.Entries[index] != b.Entries[index] {
			return false
		}
	}
	return true
}

func viewRequest(req *request) RequestView {
	v := req.view
	v.Subject.Digests = copyDigests(req.view.Subject.Digests)
	v.Subject.Entries = append([]Entry(nil), req.view.Subject.Entries...)
	v.Decisions = make([]DecisionView, 0, len(req.view.Decisions))
	for _, d := range req.view.Decisions {
		d.Decision.Exclude = append([]Exclusion(nil), d.Decision.Exclude...)
		v.Decisions = append(v.Decisions, d)
	}
	v.Excluded = []Exclusion{}
	for _, item := range sortedKeys(req.excluded) {
		v.Excluded = append(v.Excluded, Exclusion{Item: item, Reason: req.excluded[item]})
	}
	v.Executed = sortedKeys(req.executed)
	return v
}

func (req *request) end(state, actor string, now time.Time) {
	req.view.State, req.view.EndedBy, req.view.Ended = state, actor, now
}

func (r *Registry) Request(workspace string, subject Subject, actor Actor) (RequestView, error) {
	if !validID(workspace) || !validActor(actor) {
		return RequestView{}, ErrInvalid
	}
	subject, made, ok := r.subjectForm(subject)
	if !ok {
		return RequestView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil {
		return RequestView{}, ErrNotFound
	}
	if old := s.requests[subject.ID]; old != nil {
		if !sameSubject(old.view.Subject, subject) || old.view.Requester != actor.ID {
			return RequestView{}, ErrConflict
		}
		return viewRequest(old), nil
	}
	policy, ok := s.policies[subject.Stage]
	if !ok {
		return RequestView{}, refuse(ReasonNoPolicy)
	}
	if len(policy.Required) != len(subject.Digests) {
		return RequestView{}, ErrRejected
	}
	for _, k := range policy.Required {
		if subject.Digests[k] == "" {
			return RequestView{}, ErrRejected
		}
	}
	if len(s.requests) >= r.limits.Requests {
		return RequestView{}, ErrLimit
	}
	key := [2]string{subject.Stage, subject.Target}
	for _, old := range s.targets[key] {
		if old.view.State == StatePending || old.view.State == StateApproved {
			old.end(StateSuperseded, actor.ID, now)
		}
	}
	made.view.Policy, made.view.State, made.view.Requester, made.view.Written = policy.Revision, StatePending, actor.ID, now
	made.view.Digest = subjectDigest(workspace, subject, policy.Revision)
	s.requests[subject.ID] = made
	s.targets[key] = append(s.targets[key], made)
	return viewRequest(made), nil
}

func (r *Registry) RequestOf(workspace, id string) (RequestView, error) {
	if !validID(workspace) || !validID(id) {
		return RequestView{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.requests[id] == nil {
		return RequestView{}, ErrNotFound
	}
	return viewRequest(s.requests[id]), nil
}

func decisionForm(d Decision) bool {
	seen := map[string]bool{}
	for _, x := range d.Exclude {
		if !validID(x.Item) || !excludeReasons[x.Reason] || seen[x.Item] {
			return false
		}
		seen[x.Item] = true
	}
	switch d.Verdict {
	case VerdictApprove:
		return d.Reason == ""
	case VerdictReject:
		return rejectReasons[d.Reason] && len(d.Exclude) == 0
	}
	return false
}

func sameDecision(a, b Decision) bool {
	if a.Verdict != b.Verdict || a.Reason != b.Reason || len(a.Exclude) != len(b.Exclude) {
		return false
	}
	for index := range a.Exclude {
		if a.Exclude[index] != b.Exclude[index] {
			return false
		}
	}
	return true
}

func (req *request) covered(extra map[string]string) exact.Fraction {
	if len(req.entries) == 0 {
		return req.amount
	}
	total := exact.Whole(0)
	for item, e := range req.entries {
		if req.excluded[item] == "" && extra[item] == "" {
			total = total.Plus(absolute(e.amount))
		}
	}
	return total
}

func (req *request) remaining(extra map[string]string) int {
	count := 0
	for item := range req.entries {
		if req.excluded[item] == "" && extra[item] == "" {
			count++
		}
	}
	return count
}

func (r *Registry) Decide(workspace, id string, d Decision, actor Actor) (RequestView, error) {
	if !validID(workspace) || !validID(id) || !validActor(actor) || !decisionForm(d) {
		return RequestView{}, ErrInvalid
	}
	d.Exclude = append([]Exclusion(nil), d.Exclude...)
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.requests[id] == nil {
		return RequestView{}, ErrNotFound
	}
	req := s.requests[id]
	for _, x := range d.Exclude {
		if req.entries[x.Item] == nil {
			return RequestView{}, ErrNotFound
		}
	}
	if index, ok := req.deciders[actor.ID]; ok {
		if !sameDecision(req.view.Decisions[index].Decision, d) {
			return RequestView{}, ErrConflict
		}
		return viewRequest(req), nil
	}
	if actor.Kind != KindHuman {
		return RequestView{}, refuse(ReasonAINotAllowed)
	}
	if req.view.State != StatePending {
		return RequestView{}, refuse(ReasonNotPending)
	}
	if actor.ID == req.view.Subject.Author || actor.ID == req.view.Requester {
		return RequestView{}, refuse(ReasonSelfApproval)
	}
	extra := map[string]string{}
	for _, x := range d.Exclude {
		extra[x.Item] = x.Reason
	}
	if d.Verdict == VerdictApprove {
		if !s.covering(actor.ID, req.view.Subject.Stage, req.view.Subject.Currency, req.covered(extra), now) {
			return RequestView{}, refuse(ReasonNoAuthority)
		}
		if len(req.entries) > 0 && req.remaining(extra) == 0 {
			return RequestView{}, refuse(ReasonNothingLeft)
		}
	}
	view := DecisionView{Decision: d, Actor: actor.ID, At: now}
	view.Digest = decisionDigest(workspace, id, view)
	req.deciders[actor.ID] = len(req.view.Decisions)
	req.view.Decisions = append(req.view.Decisions, view)
	if d.Verdict == VerdictReject {
		req.end(StateRejected, actor.ID, now)
		return viewRequest(req), nil
	}
	for item, reason := range extra {
		if req.excluded[item] == "" {
			req.excluded[item] = reason
		}
	}
	req.approvers = append(req.approvers, actor.ID)
	if len(req.approvers) >= s.policies[req.view.Subject.Stage].Approvals && s.policies[req.view.Subject.Stage].Revision == req.view.Policy {
		req.view.State, req.view.ApprovedAt = StateApproved, now
	}
	return viewRequest(req), nil
}

func (r *Registry) Withdraw(workspace, id string, actor Actor) (RequestView, error) {
	if !validID(workspace) || !validID(id) || !validActor(actor) {
		return RequestView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.requests[id] == nil {
		return RequestView{}, ErrNotFound
	}
	req := s.requests[id]
	if req.view.State == StateWithdrawn && req.view.EndedBy == actor.ID {
		return viewRequest(req), nil
	}
	if actor.ID != req.view.Requester || req.view.State != StatePending && req.view.State != StateApproved {
		return RequestView{}, ErrRejected
	}
	if len(req.entries) == 0 && len(req.executed) > 0 {
		return RequestView{}, ErrRejected
	}
	req.end(StateWithdrawn, actor.ID, now)
	return viewRequest(req), nil
}
