package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/termsauthority"
)

func termGrantTerms(action gate.Action, kind string, fields []string) bool {
	switch action {
	case gate.RecordIssuanceTerm, gate.ApproveIssuanceTerm, gate.BindIssuanceTerm, gate.ApproveIssuanceTermBinding:
		return true
	}
	if kind == termsauthority.ResourceKind {
		return true
	}
	return false
}

func (a *serverGrantAuthority) termShape(spec inputauthority.Spec) bool {
	if a.s == nil || spec.Workspace != a.s.bootWorkspace || spec.NotBefore.IsZero() || !spec.NotBefore.Before(spec.NotAfter) {
		return false
	}
	if _, err := termAuthorityTerms(spec.Workspace, spec.Capability); err != nil {
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
