package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
)

type fallibleProposalStore struct {
	name      string
	prefix    string
	rechecked bool
	tally     func(*Server) (map[string]int, error)
}

var fallibleProposalStores = []fallibleProposalStore{{name: "input_authority_proposals", prefix: "input_authority", rechecked: true, tally: inputAuthorityTally}}

func inputAuthorityTally(s *Server) (map[string]int, error) {
	out := map[string]int{}
	if s.inputAuthority == nil {
		return out, nil
	}
	if s.faulty("input-authority-impact") {
		return nil, gate.ErrUnavailable
	}
	records, err := s.inputAuthority.registry.Records(s.bootWorkspace)
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
