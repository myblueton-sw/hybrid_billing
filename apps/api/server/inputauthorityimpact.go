package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/pricingpolicy"
)

type fallibleProposalStore struct {
	name      string
	prefix    string
	rechecked bool
	tally     func(*Server) (map[string]int, error)
}

var fallibleProposalStores = []fallibleProposalStore{{name: "result_evidence_proposals", prefix: "result_evidence", rechecked: true, tally: resultEvidenceTally}, {name: "result_applications_unknown", prefix: "erp_result", rechecked: true, tally: resultApplicationTally}, {name: "result_authority_proposals", prefix: "result_authority", rechecked: true, tally: resultAuthorityTally}, {name: "document_evidence_proposals", prefix: "document_evidence", rechecked: true, tally: documentEvidenceTally}, {name: "document_authority_proposals", prefix: "document_authority", rechecked: true, tally: documentAuthorityTally}, {name: "financial_charge_requests", prefix: "financial_charge", rechecked: false, tally: financialChargeTally}, {name: "financial_request_permission_proposals", prefix: "financial_request_permission", rechecked: true, tally: financialRequestPermissionTally}, {name: "financial_permission_proposals", prefix: "financial_permission", rechecked: true, tally: financialPermissionTally}, {name: "financial_authority_proposals", prefix: "financial_authority", rechecked: true, tally: financialAuthorityTally}, {name: "financial_policy_proposals", prefix: "financial_policy", rechecked: true, tally: financialPolicyTally}, {name: "input_authority_proposals", prefix: "input_authority", rechecked: true, tally: inputAuthorityTally}, {name: "pricing_authority_proposals", prefix: "pricing_authority", rechecked: true, tally: pricingAuthorityTally}, {name: "pricing_policy_proposals", prefix: "pricing_policy", rechecked: true, tally: pricingPolicyTally}}

func pricingPolicyTally(s *Server) (map[string]int, error) {
	out := map[string]int{}
	if s.chargeProducer == nil || s.chargeProducer.pricingPolicies == nil {
		return out, nil
	}
	if s.faulty("pricing-policy-impact") {
		return nil, gate.ErrUnavailable
	}
	records, err := s.chargeProducer.pricingPolicies.Records(s.bootWorkspace)
	if err != nil {
		return nil, err
	}
	for _, record := range records {
		if record.State == pricingpolicy.Pending {
			out[record.Proposal.Proposer]++
		}
	}
	return out, nil
}

func inputAuthorityTally(s *Server) (map[string]int, error) {
	return grantAuthorityTally(s, inputGrantAuthority)
}
func pricingAuthorityTally(s *Server) (map[string]int, error) {
	return grantAuthorityTally(s, pricingGrantAuthority)
}
func grantAuthorityTally(s *Server, domain grantAuthorityDomain) (map[string]int, error) {
	a := s.grantAuthority(domain)
	out := map[string]int{}
	if a == nil {
		return out, nil
	}
	if s.faulty(domain.faultPrefix() + "impact") {
		return nil, gate.ErrUnavailable
	}
	records, err := a.registry.Records(s.bootWorkspace)
	if err != nil {
		return nil, err
	}
	for _, record := range records {
		if record.State == inputauthority.Pending || record.State == inputauthority.Applying || record.State == inputauthority.Unknown {
			out[record.Proposal.Proposer]++
		}
	}
	return out, nil
}

func financialPolicyTally(s *Server) (map[string]int, error) {
	out := map[string]int{}
	if s.cfg.Seed != SeedSample {
		return out, nil
	}
	if s.approvals == nil || s.financialPolicies == nil || s.faulty("financial-policy-impact") {
		return nil, gate.ErrUnavailable
	}
	for id := range s.financialPolicies {
		record, _, err := s.financialPolicyRecord(s.bootWorkspace, id)
		if err != nil {
			return nil, err
		}
		if record.State == approvals.StatePending {
			out[record.Proposer]++
		}
	}
	return out, nil
}

func financialAuthorityTally(s *Server) (map[string]int, error) {
	out := map[string]int{}
	if s.cfg.Seed != SeedSample {
		return out, nil
	}
	if s.approvals == nil || s.financialAuthorities == nil || s.faulty("financial-authority-impact") {
		return nil, gate.ErrUnavailable
	}
	for id := range s.financialAuthorities {
		record, err := s.financialAuthorityRecord(s.bootWorkspace, id)
		if err != nil {
			return nil, err
		}
		if record.proposal.State == approvals.StatePending {
			out[record.proposal.Proposer]++
		}
	}
	return out, nil
}

func financialPermissionTally(s *Server) (map[string]int, error) {
	return grantAuthorityTally(s, financialPermissionDomain)
}

func financialRequestPermissionTally(s *Server) (map[string]int, error) {
	return grantAuthorityTally(s, financialRequestDomain)
}

func financialChargeTally(s *Server) (map[string]int, error) {
	out := map[string]int{}
	if s.cfg.Seed != SeedSample {
		return out, nil
	}
	if s.approvals == nil || s.financialCharges == nil || s.faulty("financial-charge-impact") {
		return nil, gate.ErrUnavailable
	}
	for id := range s.financialCharges {
		v, _, err := s.financialChargeRecord(s.bootWorkspace, id)
		if err != nil {
			return nil, err
		}
		if v.State == approvals.StatePending {
			out[v.Requester]++
		}
	}
	return out, nil
}

func documentAuthorityTally(s *Server) (map[string]int, error) {
	return grantAuthorityTally(s, documentGrantAuthority)
}

func documentEvidenceTally(s *Server) (map[string]int, error) {
	out := map[string]int{}
	if s.cfg.Seed != SeedSample {
		return out, nil
	}
	if s.documentERP == nil || s.faulty("document-evidence-impact") {
		return nil, gate.ErrUnavailable
	}
	for key, record := range s.documentERP.documents {
		if key[0] != s.bootWorkspace {
			return nil, gate.ErrUnavailable
		}
		if _, _, err := s.documentERP.governedDocument(key[0], key[1], record.review.Digest); err != nil {
			return nil, err
		}
		if record.review.State == "pending" {
			out[record.review.Proposer]++
		}
	}
	return out, nil
}

func resultAuthorityTally(s *Server) (map[string]int, error) {
	return grantAuthorityTally(s, resultGrantAuthority)
}
