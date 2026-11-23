package server

import "github.com/myblueton-sw/hybrid_billing/modules/gate"

func issuanceDraftTally(s *Server) (map[string]int, error) {
	out := map[string]int{}
	if s.cfg.Seed != SeedSample {
		return out, nil
	}
	a := s.financialIssuance
	if a == nil || s.faulty("issuance-draft-impact") || len(a.drafts) > financialIssuanceLimit {
		return nil, gate.ErrUnavailable
	}
	for id, p := range a.drafts {
		if p.Workspace != s.bootWorkspace || p.Spec.ID != id || !validRequestID(p.Submitter) {
			return nil, gate.ErrUnavailable
		}
		switch p.State {
		case "submission_refused":
			continue
		case "submission_unknown", "submitted":
		default:
			return nil, gate.ErrUnavailable
		}
		last, reviewed, err := a.draftReviewChain(p)
		if err != nil {
			return nil, gate.ErrUnavailable
		}
		approved := reviewed && last.State == "approved"
		if !approved {
			out[p.Submitter]++
		}
	}
	return out, nil
}
