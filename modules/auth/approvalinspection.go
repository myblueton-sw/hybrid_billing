package auth

import "time"

func (r *Registry) InspectApprovalAuthority(subjectID, authorID string, request Request) Explanation {
	r.mu.RLock()
	defer r.mu.RUnlock()
	subject, _ := r.subject(subjectID)
	principal := Principal{Installation: r.installation, SubjectID: subjectID, Epoch: subject.Epoch}
	d := r.consistent(principal, func(now time.Time) Decision {
		return r.checkApproval(principal, authorID, request, now)
	})
	return Explanation{Allowed: d.Allowed, Code: d.Code, Reason: d.Reason}
}
