package server

import "github.com/myblueton-sw/hybrid_billing/modules/gate"

func resultEvidenceCounts(s *Server) (map[string]int, map[string]int, error) {
	pending, unknown := map[string]int{}, map[string]int{}
	if s.cfg.Seed != SeedSample {
		return pending, unknown, nil
	}
	a := s.documentERP
	if a == nil || s.faulty("result-evidence-impact") {
		return nil, nil, gate.ErrUnavailable
	}
	if len(a.results) > financialDocumentLimit || len(a.resultApplications) > financialDocumentLimit || len(a.resultExternals) != len(a.resultApplications) || len(a.resultPreparations) != len(a.resultApplications) {
		return nil, nil, gate.ErrUnavailable
	}
	for key, record := range a.results {
		if key[0] != s.bootWorkspace {
			return nil, nil, gate.ErrUnavailable
		}
		r, b, err := a.resultRecord(key[0], key[1], record.review.Digest)
		if err != nil {
			return nil, nil, err
		}
		if _, _, err = a.resultApplicationOwner(b, r.review.Digest); err != nil {
			return nil, nil, err
		}
		if r.review.State == "pending" {
			pending[r.review.Proposer]++
		}
	}
	for key, app := range a.resultApplications {
		if key[0] != s.bootWorkspace {
			return nil, nil, gate.ErrUnavailable
		}
		r, b, err := a.resultRecord(key[0], key[1], app.digest)
		if err != nil {
			return nil, nil, err
		}
		checked, owned, err := a.resultApplicationOwner(b, r.review.Digest)
		if err != nil {
			return nil, nil, err
		}
		if !owned || r.review.Confirmation.Actor == "" {
			return nil, nil, gate.ErrUnavailable
		}
		if checked.receipt == "" {
			unknown[checked.actor]++
		}
	}
	return pending, unknown, nil
}

func resultEvidenceTally(s *Server) (map[string]int, error) {
	pending, _, err := resultEvidenceCounts(s)
	return pending, err
}
func resultApplicationTally(s *Server) (map[string]int, error) {
	_, unknown, err := resultEvidenceCounts(s)
	return unknown, err
}
