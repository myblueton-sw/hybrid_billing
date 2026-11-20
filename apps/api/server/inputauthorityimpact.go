package server

import (
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

var fallibleProposalStores = []fallibleProposalStore{{name: "input_authority_proposals", prefix: "input_authority", rechecked: true, tally: inputAuthorityTally}, {name: "pricing_authority_proposals", prefix: "pricing_authority", rechecked: true, tally: pricingAuthorityTally}, {name: "pricing_policy_proposals", prefix: "pricing_policy", rechecked: true, tally: pricingPolicyTally}}

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
