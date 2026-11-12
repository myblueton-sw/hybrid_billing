package auth

import (
	"crypto/sha256"
	"encoding/binary"
	"sort"
	"time"
)

func (r *Registry) direct(subjectID, clientID string, action Action, kind string, resource Resource, field string, now time.Time) bool {
	for _, grant := range r.grants {
		g := grant.value
		if grant.closed || g.SubjectID != subjectID || g.ClientID != clientID || !g.Validity.at(now) ||
			g.Terms.Action != action || g.Terms.Kind != kind || !g.Terms.Scope.matches(resource) ||
			!g.Terms.Period.matches(resource) || (field != "" && !contains(g.Terms.Fields, field)) {
			continue
		}
		return true
	}
	return false
}

func (r *Registry) classView(field string) Action {
	if r.fields[field] == Cost {
		return ViewCost
	}
	return View
}

func (r *Registry) covers(delegation versioned[Delegation], action Action, kind string, resource Resource, field string, now time.Time) bool {
	d := delegation.value
	return !delegation.closed && d.Validity.at(now) && d.Terms.Action == action && d.Terms.Kind == kind &&
		d.Terms.Scope.matches(resource) && d.Terms.Period.matches(resource) && (field == "" || contains(d.Terms.Fields, field)) &&
		r.activeHuman(d.DelegatorID) && r.activeHuman(d.DelegateID) &&
		r.holdsInWorkspace(d.DelegateID, d.Terms.Scope.WorkspaceID, now)
}

func (r *Registry) through(id string, action Action, kind string, resource Resource, field string, now time.Time) ([]string, bool) {
	chain := []string{}
	seen := map[string]struct{}{}
	for current := r.delegations[id]; ; {
		d := current.value
		if current.version == 0 || len(chain) >= maxChain || !r.covers(current, action, kind, resource, field, now) {
			return nil, false
		}
		if _, repeated := seen[d.DelegateID]; repeated {
			return nil, false
		}
		seen[d.DelegateID] = struct{}{}
		if len(chain) > 0 && !d.AllowRedelegation {
			return nil, false
		}
		chain = append(chain, d.ID)
		if d.ParentID == "" {
			if _, repeated := seen[d.DelegatorID]; repeated {
				return nil, false
			}
			if !r.direct(d.DelegatorID, "", action, kind, resource, field, now) {
				return nil, false
			}
			if action == Download {
				if !r.direct(d.DelegatorID, "", r.classView(field), kind, resource, field, now) {
					return nil, false
				}
			}
			return chain, true
		}
		parent := r.delegations[d.ParentID]
		if parent.version == 0 || parent.value.DelegateID != d.DelegatorID {
			return nil, false
		}
		current = parent
	}
}

func (r *Registry) single(subjectID string, action Action, kind string, resource Resource, field string, now time.Time) ([]string, bool) {
	if r.direct(subjectID, "", action, kind, resource, field, now) {
		return nil, true
	}
	ids := make([]string, 0)
	for id, delegation := range r.delegations {
		if delegation.value.DelegateID == subjectID {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		if chain, ok := r.through(id, action, kind, resource, field, now); ok {
			return chain, true
		}
	}
	return nil, false
}

func (r *Registry) subjectAllowed(subjectID string, request Request, now time.Time) ([]string, bool) {
	if !fieldAction(request.Action) {
		chain, ok := r.single(subjectID, request.Action, request.Kind, request.Resource, "", now)
		if !ok || len(chain) == 0 {
			return nil, ok
		}
		sorted := append([]string(nil), chain...)
		sort.Strings(sorted)
		return sorted, true
	}
	used := map[string]struct{}{}
	for _, field := range request.Fields {
		needed := []Action{request.Action}
		if request.Action == Download {
			needed = append(needed, r.classView(field))
		}
		for _, action := range needed {
			chain, ok := r.single(subjectID, action, request.Kind, request.Resource, field, now)
			if !ok {
				return nil, false
			}
			for _, id := range chain {
				used[id] = struct{}{}
			}
		}
	}
	if len(used) == 0 {
		return nil, true
	}
	chains := make([]string, 0, len(used))
	for id := range used {
		chains = append(chains, id)
	}
	sort.Strings(chains)
	return chains, true
}

func (r *Registry) clientAllowed(clientID string, request Request, now time.Time) bool {
	fields := request.Fields
	if !fieldAction(request.Action) {
		fields = []string{""}
	}
	for _, field := range fields {
		if !r.direct("", clientID, request.Action, request.Kind, request.Resource, field, now) {
			return false
		}
		if request.Action == Download {
			if !r.direct("", clientID, r.classView(field), request.Kind, request.Resource, field, now) {
				return false
			}
		}
	}
	return true
}

func (r *Registry) validRequest(request Request) bool {
	if _, known := actions[request.Action]; !known {
		return false
	}
	if _, known := r.kinds[request.Kind]; !known {
		return false
	}
	return validID(request.Resource.WorkspaceID) && r.validFields(request.Action, request.Fields)
}

func (r *Registry) current(principal Principal) bool {
	subject := r.subjects[principal.SubjectID]
	return principal.Installation == r.installation && subject.version != 0 && subject.value.State == Active && subject.value.Epoch == principal.Epoch
}

func (r *Registry) stamp(principal Principal, request Request, author string, approval bool) Stamp {
	hash := sha256.New()
	write := func(parts ...string) {
		for _, part := range parts {
			var size [8]byte
			binary.BigEndian.PutUint64(size[:], uint64(len(part)))
			hash.Write(size[:])
			hash.Write([]byte(part))
		}
	}
	fields := append([]string(nil), request.Fields...)
	sort.Strings(fields)
	var numbers [17]byte
	binary.BigEndian.PutUint64(numbers[:8], principal.Epoch)
	binary.BigEndian.PutUint64(numbers[8:16], uint64(request.Resource.Month))
	if request.Resource.HasMonth {
		numbers[16] = 1
	}
	write(principal.Installation, principal.SubjectID, principal.ClientID, principal.SessionID, string(request.Action), request.Kind,
		request.Resource.WorkspaceID, request.Resource.CustomerID, request.Resource.LegalEntityID, request.Resource.ContractID,
		string(numbers[:]), author)
	write(fields...)
	stamp := Stamp{instance: r.instance, generation: r.generation, approval: approval, issued: true}
	copy(stamp.digest[:], hash.Sum(nil))
	return stamp
}

func denied(code Code, reason Reason, principal Principal) Decision {
	return Decision{Code: code, Reason: reason, ActorID: principal.SubjectID, ClientID: principal.ClientID, SessionID: principal.SessionID}
}

func (r *Registry) decide(principal Principal, request Request, now time.Time) Decision {
	if !r.available {
		return denied(AuthorityUnavailable, ReasonUnavailable, principal)
	}
	if !r.validRequest(request) {
		return denied(PermissionDenied, ReasonInvalidRequest, principal)
	}
	if !r.current(principal) || (principal.ClientID != "" && principal.SessionID != "") {
		return denied(PermissionDenied, ReasonPrincipal, principal)
	}
	if humanOnly(request.Action) {
		return denied(PermissionDenied, ReasonApprovalRequired, principal)
	}
	allowed := Decision{Allowed: true, Code: Allowed, Reason: ReasonAllowed, ActorID: principal.SubjectID, ClientID: principal.ClientID, SessionID: principal.SessionID}
	if principal.SessionID != "" {
		session := r.sessions[principal.SessionID]
		s := session.value
		if session.version == 0 || session.closed || !s.Validity.at(now) || s.ActorID != principal.SubjectID || !r.activeHuman(s.ActorID) {
			return denied(PermissionDenied, ReasonSupportSession, principal)
		}
		if request.Action != View || request.Kind != s.Kind || request.Resource.WorkspaceID != s.WorkspaceID ||
			request.Resource.CustomerID == "" || request.Resource.CustomerID != s.CustomerID {
			return denied(PermissionDenied, ReasonSupportScope, principal)
		}
		for _, field := range request.Fields {
			if !contains(s.Fields, field) {
				return denied(PermissionDenied, ReasonSupportScope, principal)
			}
		}
		allowed.Stamp = r.stamp(principal, request, "", false)
		return allowed
	}
	chains, ok := r.subjectAllowed(principal.SubjectID, request, now)
	if !ok {
		return denied(PermissionDenied, ReasonNoGrant, principal)
	}
	if principal.ClientID != "" {
		client := r.clients[principal.ClientID]
		if client.version == 0 || client.value.State != Active || !r.clientAllowed(principal.ClientID, request, now) {
			return denied(PermissionDenied, ReasonClient, principal)
		}
	}
	allowed.Delegations = chains
	allowed.Stamp = r.stamp(principal, request, "", false)
	return allowed
}

func (r *Registry) time() time.Time {
	if r.clock == nil {
		return time.Time{}
	}
	return r.clock()
}

func (r *Registry) Decide(principal Principal, request Request) Decision {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.decide(principal, request, r.time())
}

func (r *Registry) Recheck(stamp Stamp, principal Principal, request Request) Decision {
	r.mu.RLock()
	defer r.mu.RUnlock()
	decision := r.decide(principal, request, r.time())
	if decision.Allowed && (!stamp.issued || stamp.approval || stamp != decision.Stamp) {
		return denied(PermissionDenied, ReasonStamp, principal)
	}
	return decision
}

func (r *Registry) checkApproval(approver Principal, authorID string, request Request, now time.Time) Decision {
	if !r.available {
		return denied(AuthorityUnavailable, ReasonUnavailable, approver)
	}
	if request.Action != Approve || !r.validRequest(request) {
		return denied(PermissionDenied, ReasonInvalidRequest, approver)
	}
	if !r.current(approver) || approver.ClientID != "" || approver.SessionID != "" {
		return denied(PermissionDenied, ReasonPrincipal, approver)
	}
	if !r.activeHuman(approver.SubjectID) {
		return denied(PermissionDenied, ReasonNotHuman, approver)
	}
	if r.subjects[authorID].version == 0 {
		return denied(PermissionDenied, ReasonAuthor, approver)
	}
	if !r.direct(approver.SubjectID, "", Approve, request.Kind, request.Resource, "", now) {
		return denied(PermissionDenied, ReasonNoGrant, approver)
	}
	if authorID == approver.SubjectID {
		return denied(SelfApprovalDenied, ReasonSelfApproval, approver)
	}
	return Decision{Allowed: true, Code: Allowed, Reason: ReasonAllowed, ActorID: approver.SubjectID, Stamp: r.stamp(approver, request, authorID, true)}
}

func (r *Registry) CheckApproval(approver Principal, authorID string, request Request) Decision {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.checkApproval(approver, authorID, request, r.time())
}

func (r *Registry) RecheckApproval(stamp Stamp, approver Principal, authorID string, request Request) Decision {
	r.mu.RLock()
	defer r.mu.RUnlock()
	decision := r.checkApproval(approver, authorID, request, r.time())
	if decision.Allowed && (!stamp.issued || !stamp.approval || stamp != decision.Stamp) {
		return denied(PermissionDenied, ReasonStamp, approver)
	}
	return decision
}
