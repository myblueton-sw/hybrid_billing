package gate

import "github.com/myblueton-sw/hybrid_billing/modules/auth"

func (g *Gate) InspectApprovalAuthority(subjectID, authorID string, request Request) Explanation {
	if !g.ready() {
		return Explanation{Code: AuthorityUnavailable, Reason: ReasonUnavailable}
	}
	generation := g.view.Generation()
	d := g.permissions.InspectApprovalAuthority(subjectID, authorID, request)
	owned := g.ownAuthor(subjectID, authorID, auth.Decision{Allowed: d.Allowed, Code: d.Code, Reason: d.Reason})
	if g.view.Generation() != generation && owned.Code != AuthorityUnavailable {
		return Explanation{Code: PermissionDenied, Reason: ReasonStamp}
	}
	return Explanation{Allowed: owned.Allowed, Code: owned.Code, Reason: owned.Reason}
}
