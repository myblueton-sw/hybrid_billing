package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/documentauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/financialpermission"
	"github.com/myblueton-sw/hybrid_billing/modules/financialrequestauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/pricingauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/resultauthority"
)

type grantAuthorityDomain uint8

const (
	resultGrantAuthority      grantAuthorityDomain = 6
	documentGrantAuthority    grantAuthorityDomain = 5
	inputGrantAuthority       grantAuthorityDomain = 1
	pricingGrantAuthority     grantAuthorityDomain = 2
	financialPermissionDomain grantAuthorityDomain = 3
	financialRequestDomain    grantAuthorityDomain = 4
)

func (d grantAuthorityDomain) grantPrefix() string {
	if d == resultGrantAuthority {
		return "result-grant-"
	}
	if d == documentGrantAuthority {
		return "document-grant-"
	}
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
	if d == resultGrantAuthority {
		return "result-authority-"
	}
	if d == documentGrantAuthority {
		return "document-authority-"
	}
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
	if d == resultGrantAuthority {
		return resultAuthorityTerms(workspace, capability)
	}
	if d == documentGrantAuthority {
		return documentAuthorityTerms(workspace, capability)
	}
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
	case resultGrantAuthority:
		return s.resultAuthority
	case documentGrantAuthority:
		return s.documentAuthority
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

func documentAuthorityTerms(workspace, capability string) (gate.Terms, error) {
	terms := gate.Terms{Kind: documentauthority.ResourceKind, Scope: gate.Scope{WorkspaceID: workspace, Customer: gate.All(), LegalEntity: gate.All(), Contract: gate.All()}, Period: gate.AnyPeriod()}
	switch capability {
	case documentauthority.Propose:
		terms.Action = gate.ProposeDocumentEvidence
	case documentauthority.Confirm:
		terms.Action = gate.ConfirmDocumentEvidence
	case documentauthority.Revoke:
		terms.Action = gate.RevokeDocumentEvidence
	case documentauthority.Prepare:
		terms.Action = gate.PrepareERPRecord
	case documentauthority.Inspect:
		terms.Action = gate.ViewCost
		terms.Fields = []string{documentauthority.EvidenceField}
	default:
		return gate.Terms{}, inputauthority.ErrInvalid
	}
	return terms, nil
}

func resultAuthorityTerms(workspace, capability string) (gate.Terms, error) {
	terms := gate.Terms{Kind: resultauthority.ResourceKind, Scope: gate.Scope{WorkspaceID: workspace, Customer: gate.All(), LegalEntity: gate.All(), Contract: gate.All()}, Period: gate.AnyPeriod()}
	switch capability {
	case resultauthority.Propose:
		terms.Action = gate.ProposeERPResult
	case resultauthority.Confirm:
		terms.Action = gate.ConfirmERPResult
	case resultauthority.Revoke:
		terms.Action = gate.RevokeERPResult
	case resultauthority.Apply:
		terms.Action = gate.ApplyERPResult
	case resultauthority.Inspect:
		terms.Action = gate.ViewCost
		terms.Fields = []string{resultauthority.EvidenceField}
	default:
		return gate.Terms{}, inputauthority.ErrInvalid
	}
	return terms, nil
}
