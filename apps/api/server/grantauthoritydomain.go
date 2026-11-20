package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/financialpermission"
	"github.com/myblueton-sw/hybrid_billing/modules/financialrequestauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/pricingauthority"
)

type grantAuthorityDomain uint8

const (
	inputGrantAuthority       grantAuthorityDomain = 1
	pricingGrantAuthority     grantAuthorityDomain = 2
	financialPermissionDomain grantAuthorityDomain = 3
	financialRequestDomain    grantAuthorityDomain = 4
)

func (d grantAuthorityDomain) grantPrefix() string {
	if d == financialRequestDomain {
		return "financial-request-"
	}
	if d == financialPermissionDomain {
		return "financial-permission-"
	}
	if d == pricingGrantAuthority {
		return "pricing-grant-"
	}
	if d == inputGrantAuthority {
		return "input-grant-"
	}
	return ""
}
func (d grantAuthorityDomain) faultPrefix() string {
	if d == financialRequestDomain {
		return "financial-request-"
	}
	if d == financialPermissionDomain {
		return "financial-permission-"
	}
	if d == pricingGrantAuthority {
		return "pricing-authority-"
	}
	if d == inputGrantAuthority {
		return "input-authority-"
	}
	return ""
}
func (d grantAuthorityDomain) terms(workspace, capability string) (gate.Terms, error) {
	if d == financialRequestDomain {
		if capability != financialrequestauthority.Request {
			return gate.Terms{}, inputauthority.ErrInvalid
		}
		return gate.Terms{Action: gate.RequestApproval, Kind: financialrequestauthority.ResourceKind, Scope: gate.Scope{WorkspaceID: workspace, Customer: gate.All(), LegalEntity: gate.All(), Contract: gate.All()}, Period: gate.AnyPeriod()}, nil
	}
	if d == financialPermissionDomain {
		if capability != financialpermission.Approve {
			return gate.Terms{}, inputauthority.ErrInvalid
		}
		return gate.Terms{Action: gate.Approve, Kind: financialpermission.ResourceKind, Scope: gate.Scope{WorkspaceID: workspace, Customer: gate.All(), LegalEntity: gate.All(), Contract: gate.All()}, Period: gate.AnyPeriod()}, nil
	}
	if d == pricingGrantAuthority {
		return pricingAuthorityTerms(workspace, capability)
	}
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

func pricingAuthorityTerms(workspace, capability string) (gate.Terms, error) {
	terms := gate.Terms{Kind: pricingauthority.ResourceKind, Scope: gate.Scope{WorkspaceID: workspace, Customer: gate.All(), LegalEntity: gate.All(), Contract: gate.All()}, Period: gate.AnyPeriod()}
	switch capability {
	case pricingauthority.Propose:
		terms.Action = gate.ProposePricingPolicy
	case pricingauthority.Approve:
		terms.Action = gate.ApprovePricingPolicy
	case pricingauthority.Revoke:
		terms.Action = gate.RevokePricingPolicy
	case pricingauthority.Use:
		terms.Action = gate.UsePricingPolicy
	case pricingauthority.Inspect:
		terms.Action = gate.ViewCost
		terms.Fields = []string{pricingauthority.EvidenceField}
	default:
		return gate.Terms{}, pricingauthority.ErrInvalid
	}
	return terms, nil
}
func (s *Server) grantAuthority(domain grantAuthorityDomain) *serverGrantAuthority {
	switch domain {
	case inputGrantAuthority:
		return s.inputAuthority
	case pricingGrantAuthority:
		return s.pricingAuthority
	case financialPermissionDomain:
		return s.financialPermission
	case financialRequestDomain:
		return s.financialRequestPermission
	}
	return nil
}
