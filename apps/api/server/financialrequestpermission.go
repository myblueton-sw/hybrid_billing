package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/financialrequestauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
)

func (a *serverGrantAuthority) financialRequestShape(spec inputauthority.Spec) bool {
	if a.s == nil || spec.Workspace != a.s.bootWorkspace || spec.Capability != financialrequestauthority.Request {
		return false
	}
	if spec.Kind == inputauthority.Console {
		return spec.Recipient == ConsoleClient && spec.CeilingID == ""
	}
	if spec.Kind != inputauthority.Human || !validRequestID(spec.Recipient) || spec.CeilingID == "" {
		return false
	}
	ceiling, ok := a.ceilings[spec.CeilingID]
	return ok && ceiling.Kind == inputauthority.Console && ceiling.Recipient == ConsoleClient && ceiling.CeilingID == "" && ceiling.Capability == spec.Capability && ceiling.Workspace == spec.Workspace && !spec.NotBefore.Before(ceiling.NotBefore) && !spec.NotAfter.After(ceiling.NotAfter)
}
