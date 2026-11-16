package auth

import (
	"sort"
	"time"
)

type GrantView struct {
	Grant
	Version   uint64
	Closed    bool
	Active    bool
	Effective bool
}

type DelegationView struct {
	Delegation
	Version uint64
	Closed  bool
	Active  bool
}

type Explanation struct {
	Allowed     bool
	Code        Code
	Reason      Reason
	Delegations []string
}

func (s Selector) Any() bool { return s.all }

func (s Selector) ID() string { return s.id }

func (p Period) Any() bool { return p.all }

func (p Period) From() Month { return p.from }

func (p Period) To() Month { return p.to }

func (r *Registry) Grants() []GrantView {
	r.mu.RLock()
	defer r.mu.RUnlock()
	now := r.time()
	out := make([]GrantView, 0, len(r.grants))
	for id, stored := range r.grants {
		g := stored.value
		g.Terms = copyTerms(g.Terms)
		active := !stored.closed && g.Validity.at(now)
		effective := active && (g.SubjectID == "" || r.memberWith(g.Terms.Scope.WorkspaceID, g.SubjectID, r.grantJoined[id]))
		out = append(out, GrantView{Grant: g, Version: stored.version, Closed: stored.closed, Active: active, Effective: effective})
	}
	sort.Slice(out, func(a, b int) bool { return out[a].ID < out[b].ID })
	return out
}

func (r *Registry) Delegations() []DelegationView {
	r.mu.RLock()
	defer r.mu.RUnlock()
	now := r.time()
	out := make([]DelegationView, 0, len(r.delegations))
	for _, stored := range r.delegations {
		d := stored.value
		d.Terms = copyTerms(d.Terms)
		out = append(out, DelegationView{Delegation: d, Version: stored.version, Closed: stored.closed, Active: !stored.closed && d.Validity.at(now)})
	}
	sort.Slice(out, func(a, b int) bool { return out[a].ID < out[b].ID })
	return out
}

func (r *Registry) Explain(subjectID string, request Request) Explanation {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if installationAction(request.Action) || humanOnly(request.Action) {
		return Explanation{Code: PermissionDenied, Reason: ReasonInvalidRequest}
	}
	subject, _ := r.subject(subjectID)
	principal := Principal{Installation: r.installation, SubjectID: subjectID, Epoch: subject.Epoch}
	d := r.consistent(principal, func(now time.Time) Decision { return r.decide(principal, request, now) })
	return Explanation{Allowed: d.Allowed, Code: d.Code, Reason: d.Reason, Delegations: d.Delegations}
}
