package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/protectedauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/termsauthority"
)

func (d grantAuthorityDomain) readNamespace() string {
	switch d {
	case protectedGrantAuthority:
		return "issuance_protected_authority"
	case termGrantAuthority:
		return "issuance_term_authority"
	case draftGrantAuthority:
		return "issuance_draft_authority"
	case resultGrantAuthority:
		return "result_authority"
	}
	return ""
}

func (d grantAuthorityDomain) readVersion() string {
	switch d {
	case protectedGrantAuthority:
		return "synthetic-issuance-protected-authority-v1"
	case termGrantAuthority:
		return "synthetic-issuance-term-authority-v1"
	case draftGrantAuthority:
		return "synthetic-issuance-draft-authority-v1"
	case resultGrantAuthority:
		return "synthetic-erp-result-authority-v1"
	}
	return ""
}

func (d grantAuthorityDomain) readCapabilities() []string {
	switch d {
	case protectedGrantAuthority:
		return []string{protectedauthority.RegisterChannel, protectedauthority.AdmitChannel, protectedauthority.RequestChange, protectedauthority.VerifyChange, protectedauthority.ApproveChange, protectedauthority.Inspect}
	case termGrantAuthority:
		return []string{termsauthority.RecordTerm, termsauthority.ApproveTerm, termsauthority.BindTerm, termsauthority.ApproveBinding, termsauthority.Inspect}
	case draftGrantAuthority:
		return []string{draftauthority.Submit, draftauthority.Approve, draftauthority.Inspect}
	case resultGrantAuthority:
		return []string{"propose", "confirm", "revoke", "apply", "inspect"}
	}
	return nil
}

func (d grantAuthorityDomain) readCapability(capability string) bool {
	for _, known := range d.readCapabilities() {
		if known == capability {
			return true
		}
	}
	return false
}

func (a *serverGrantAuthority) readShape(spec inputauthority.Spec) bool {
	switch a.domain {
	case protectedGrantAuthority:
		return a.protectedShape(spec)
	case termGrantAuthority:
		return a.termShape(spec)
	case draftGrantAuthority:
		return a.draftShape(spec)
	case resultGrantAuthority:
		return a.resultShape(spec)
	}
	return false
}
