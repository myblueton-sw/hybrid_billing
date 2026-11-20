package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
)

type grantAuthorityDomain uint8

const inputGrantAuthority grantAuthorityDomain = 1

func (d grantAuthorityDomain) grantPrefix() string {
	if d == inputGrantAuthority {
		return "input-grant-"
	}
	return ""
}
func (d grantAuthorityDomain) faultPrefix() string {
	if d == inputGrantAuthority {
		return "input-authority-"
	}
	return ""
}
func (d grantAuthorityDomain) terms(workspace, capability string) (gate.Terms, error) {
	if d != inputGrantAuthority {
		return gate.Terms{}, inputauthority.ErrInvalid
	}
	terms := gate.Terms{Kind: inputauthority.ResourceKind, Scope: gate.Scope{WorkspaceID: workspace, Customer: gate.All(), LegalEntity: gate.All(), Contract: gate.All()}, Period: gate.AnyPeriod()}
	switch capability {
	case inputauthority.Confirm:
		terms.Action = gate.ConfirmInput
	case inputauthority.Inspect:
		terms.Action = gate.ViewCost
		terms.Fields = []string{inputauthority.EvidenceField}
	default:
		return gate.Terms{}, inputauthority.ErrInvalid
	}
	return terms, nil
}
