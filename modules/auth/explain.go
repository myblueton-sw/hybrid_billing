package auth

import "time"

func (r *Registry) ExplainInstallation(subjectID, clientID string, request Request, now time.Time) Explanation {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !InstallationAction(request.Action) {
		return Explanation{Code: PermissionDenied, Reason: ReasonInvalidRequest}
	}
	subject, _ := r.subject(subjectID)
	principal := Principal{Installation: r.installation, SubjectID: subjectID, ClientID: clientID, Epoch: subject.Epoch}
	d := r.decide(principal, request, now)
	return Explanation{Allowed: d.Allowed, Code: d.Code, Reason: d.Reason}
}
