package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
)

func issuanceStepSource(s *Server, fault string) (*financialIssuance, bool, error) {
	if s.cfg.Seed != SeedSample {
		return nil, false, nil
	}
	a := s.financialIssuance
	if a == nil || s.faulty(fault) {
		return nil, false, gate.ErrUnavailable
	}
	return a, true, nil
}

func issuanceProtectedStepTally(s *Server) (map[string]int, error) {
	out := map[string]int{}
	a, ok, err := issuanceStepSource(s, "issuance-protected-step-impact")
	if err != nil || !ok {
		return out, err
	}
	if len(a.channels) > financialIssuanceLimit || len(a.changes) > financialIssuanceLimit {
		return nil, gate.ErrUnavailable
	}
	if p := a.protectedRules; p != nil {
		if p.Workspace != s.bootWorkspace || !validRequestID(p.Proposer) {
			return nil, gate.ErrUnavailable
		}
		switch p.State {
		case "registration_unknown", "proposed", "confirmation_unknown":
			out[p.Proposer]++
		case "confirmed":
		default:
			return nil, gate.ErrUnavailable
		}
	}
	for id, p := range a.channels {
		if p.Workspace != s.bootWorkspace || p.Spec.ID != id || !validRequestID(p.Registrar) {
			return nil, gate.ErrUnavailable
		}
		switch p.State {
		case "registration_unknown", "registered", "admission_unknown":
			out[p.Registrar]++
		case "admitted":
		default:
			return nil, gate.ErrUnavailable
		}
	}
	for id, p := range a.changes {
		if p.Workspace != s.bootWorkspace || p.Spec.ID != id || !validRequestID(p.Requester) {
			return nil, gate.ErrUnavailable
		}
		switch p.Phase {
		case "request_unknown":
			out[p.Requester]++
			continue
		case "recorded", "verification_unknown", "decision_unknown":
		default:
			return nil, gate.ErrUnavailable
		}
		switch p.Module.State {
		case protected.StateRequested, protected.StateConfirmed:
			out[p.Requester]++
		case protected.StateScheduled, protected.StateRejected, protected.StateWithdrawn, protected.StateLapsed:
		default:
			return nil, gate.ErrUnavailable
		}
	}
	return out, nil
}

func issuanceTermStepTally(s *Server) (map[string]int, error) {
	out := map[string]int{}
	a, ok, err := issuanceStepSource(s, "issuance-term-step-impact")
	if err != nil || !ok {
		return out, err
	}
	if len(a.termIntents) > financialIssuanceLimit || len(a.bindingIntents) > financialIssuanceLimit {
		return nil, gate.ErrUnavailable
	}
	count := func(author, state string) error {
		if !validRequestID(author) {
			return gate.ErrUnavailable
		}
		switch state {
		case "registration_unknown", "pending", "approval_unknown":
			out[author]++
		case "approved":
		default:
			return gate.ErrUnavailable
		}
		return nil
	}
	for id, p := range a.termIntents {
		if p.Spec.ID != id || termRecordWorkspace(p) != s.bootWorkspace {
			return nil, gate.ErrUnavailable
		}
		if err := count(p.Author, p.State); err != nil {
			return nil, err
		}
	}
	for id, p := range a.bindingIntents {
		if p.Spec.ID != id || termRecordWorkspace(p.Term) != s.bootWorkspace {
			return nil, gate.ErrUnavailable
		}
		if err := count(p.Author, p.State); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func termRecordWorkspace(p financialTermRecord) string {
	if p.Configuration != nil {
		return p.Configuration.Workspace
	}
	return p.Profile.Source.Basis.Workspace
}
