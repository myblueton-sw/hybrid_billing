package auth

import "time"

func (r *Registry) checkAuthor(principal Principal, authorID string, request Request, now time.Time) Decision {
	if !r.available {
		return denied(AuthorityUnavailable, ReasonUnavailable, principal)
	}
	if !InstallationAction(request.Action) || !r.validRequest(request) || !r.inInstallation(request.Resource.WorkspaceID) {
		return denied(PermissionDenied, ReasonInvalidRequest, principal)
	}
	if !r.current(principal) || principal.SessionID != "" || principal.ClientID == "" {
		return denied(PermissionDenied, ReasonPrincipal, principal)
	}
	if !r.activeHuman(principal.SubjectID) {
		return denied(PermissionDenied, ReasonNotHuman, principal)
	}
	if authorID == "" || authorID != principal.SubjectID {
		return denied(PermissionDenied, ReasonNotAuthor, principal)
	}
	if r.authorClient == "" || principal.ClientID != r.authorClient || r.clients[principal.ClientID].value.State != Active || !r.direct("", principal.ClientID, request.Action, request.Kind, request.Resource, "", now) {
		return denied(PermissionDenied, ReasonClient, principal)
	}
	return Decision{Allowed: true, Code: Allowed, Reason: ReasonAllowed, ActorID: principal.SubjectID, ClientID: principal.ClientID,
		Stamp: r.stamp(principal, request, "author:"+authorID, false)}
}

func (r *Registry) CheckAuthor(principal Principal, authorID string, request Request) Decision {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.consistent(principal, func(now time.Time) Decision { return r.checkAuthor(principal, authorID, request, now) })
}

func (r *Registry) RecheckAuthor(stamp Stamp, principal Principal, authorID string, request Request) Decision {
	r.mu.RLock()
	defer r.mu.RUnlock()
	decision := r.consistent(principal, func(now time.Time) Decision { return r.checkAuthor(principal, authorID, request, now) })
	if decision.Allowed && (!stamp.issued || stamp.approval || stamp != decision.Stamp) {
		return denied(PermissionDenied, ReasonStamp, principal)
	}
	return decision
}
