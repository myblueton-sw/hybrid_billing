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
	Version   uint64
	Closed    bool
	Active    bool
	Effective bool
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

func insertOrdered(ids []string, id string) []string {
	at := sort.SearchStrings(ids, id)
	if at < len(ids) && ids[at] == id {
		return ids
	}
	ids = append(ids, "")
	copy(ids[at+1:], ids[at:])
	ids[at] = id
	return ids
}

func afterOrdered(ids []string, after string) int {
	if after == "" {
		return 0
	}
	return sort.Search(len(ids), func(i int) bool { return ids[i] > after })
}

func (r *Registry) grantView(id string, at time.Time) GrantView {
	stored := r.grants[id]
	g := stored.value
	g.Terms = copyTerms(g.Terms)
	active := !stored.closed && g.Validity.at(at)
	effective := active && (g.SubjectID == "" || r.memberWith(g.Terms.Scope.WorkspaceID, g.SubjectID, r.grantJoined[id]))
	return GrantView{Grant: g, Version: stored.version, Closed: stored.closed, Active: active, Effective: effective}
}

func (r *Registry) delegationView(id string, at time.Time) DelegationView {
	stored := r.delegations[id]
	d := stored.value
	d.Terms = copyTerms(d.Terms)
	return DelegationView{Delegation: d, Version: stored.version, Closed: stored.closed, Active: !stored.closed && d.Validity.at(at), Effective: r.effective(stored, at, "")}
}

func (r *Registry) Grants() []GrantView {
	r.mu.RLock()
	defer r.mu.RUnlock()
	now := r.time()
	out := make([]GrantView, 0, len(r.grantOrder))
	for _, id := range r.grantOrder {
		out = append(out, r.grantView(id, now))
	}
	return out
}

func (r *Registry) Delegations() []DelegationView {
	r.mu.RLock()
	defer r.mu.RUnlock()
	now := r.time()
	out := make([]DelegationView, 0, len(r.delegationOrder))
	for _, id := range r.delegationOrder {
		out = append(out, r.delegationView(id, now))
	}
	return out
}

func (r *Registry) GrantPage(after string, limit int, at time.Time) ([]GrantView, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit < 1 {
		return nil, false
	}
	start := afterOrdered(r.grantOrder, after)
	end := min(start+limit, len(r.grantOrder))
	out := make([]GrantView, 0, end-start)
	for _, id := range r.grantOrder[start:end] {
		out = append(out, r.grantView(id, at))
	}
	return out, end < len(r.grantOrder)
}

func (r *Registry) DelegationPage(after string, limit int, at time.Time) ([]DelegationView, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit < 1 {
		return nil, false
	}
	start := afterOrdered(r.delegationOrder, after)
	end := min(start+limit, len(r.delegationOrder))
	out := make([]DelegationView, 0, end-start)
	for _, id := range r.delegationOrder[start:end] {
		out = append(out, r.delegationView(id, at))
	}
	return out, end < len(r.delegationOrder)
}

func (r *Registry) GrantAt(id string, at time.Time) (GrantView, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, known := r.grants[id]; !known {
		return GrantView{}, false
	}
	return r.grantView(id, at), true
}

func (r *Registry) DelegationAt(id string, at time.Time) (DelegationView, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, known := r.delegations[id]; !known {
		return DelegationView{}, false
	}
	return r.delegationView(id, at), true
}

func (r *Registry) Counts() (int, int) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.grantOrder), len(r.delegationOrder)
}

func (r *Registry) Generation() uint64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.generation
}

func (r *Registry) linkHolds(stored versioned[Delegation], now time.Time, except string) bool {
	d := stored.value
	return !stored.closed && d.Validity.at(now) && r.activeHuman(d.DelegatorID) && r.activeHuman(d.DelegateID) &&
		r.memberWith(d.Terms.Scope.WorkspaceID, d.DelegatorID, r.chainJoined[d.ID][0]) &&
		r.memberWith(d.Terms.Scope.WorkspaceID, d.DelegateID, r.chainJoined[d.ID][1]) &&
		r.holdsInWorkspace(d.DelegateID, d.Terms.Scope.WorkspaceID, now, except)
}

func (r *Registry) rootBacked(d Delegation, now time.Time, except string) bool {
	if !r.backedDirectly(d.DelegatorID, d.Terms, now, except) {
		return false
	}
	if d.Terms.Action != Download {
		return true
	}
	for _, field := range d.Terms.Fields {
		view := d.Terms
		view.Action, view.Fields = r.classView(field), []string{field}
		if !r.backedDirectly(d.DelegatorID, view, now, except) {
			return false
		}
	}
	return true
}

func (r *Registry) effective(stored versioned[Delegation], now time.Time, except string) bool {
	for {
		if !r.linkHolds(stored, now, except) {
			return false
		}
		if stored.value.ParentID == "" {
			return r.rootBacked(stored.value, now, except)
		}
		stored = r.delegations[stored.value.ParentID]
	}
}

func (r *Registry) Explain(subjectID string, request Request) Explanation {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if InstallationAction(request.Action) || humanOnly(request.Action) {
		return Explanation{Code: PermissionDenied, Reason: ReasonInvalidRequest}
	}
	subject, _ := r.subject(subjectID)
	principal := Principal{Installation: r.installation, SubjectID: subjectID, Epoch: subject.Epoch}
	d := r.consistent(principal, func(now time.Time) Decision { return r.decide(principal, request, now) })
	return Explanation{Allowed: d.Allowed, Code: d.Code, Reason: d.Reason, Delegations: d.Delegations}
}
