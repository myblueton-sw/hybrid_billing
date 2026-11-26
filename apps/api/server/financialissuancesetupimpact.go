package server

import "github.com/myblueton-sw/hybrid_billing/modules/gate"

func issuanceSetupTally(s *Server) (map[string]int, error) {
	out := map[string]int{}
	if s.cfg.Seed != SeedSample {
		return out, nil
	}
	a := s.financialIssuance
	if a == nil || s.faulty("issuance-setup-impact") || a.configurationHistoryIndex() != nil {
		return nil, gate.ErrUnavailable
	}
	for _, p := range a.configurations {
		if p.Workspace != s.bootWorkspace || !validRequestID(p.Proposer) {
			return nil, gate.ErrUnavailable
		}
		switch p.State {
		case "proposal_unknown", "proposed", "confirmation_unknown":
			out[p.Proposer]++
		case "confirmed":
		default:
			return nil, gate.ErrUnavailable
		}
	}
	return out, nil
}
