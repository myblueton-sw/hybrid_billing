package server

import (
	"strconv"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

type financialDraftContextEvidence struct {
	Fields   map[string]string
	Existing *financialDraftObservation
}

func (a *financialIssuance) draftContextEvidence(workspace, actor, request, digest string) (financialDraftContextEvidence, error) {
	if err := a.draftReadCaller(workspace, actor); err != nil {
		return financialDraftContextEvidence{}, err
	}
	if !validRequestID(request) || !validPlanHash(digest) {
		return financialDraftContextEvidence{}, commands.Fail(commands.InvalidRequest)
	}
	s := a.server
	if s.approvals == nil || s.documentERP == nil {
		return financialDraftContextEvidence{}, commands.Fail(commands.UnsupportedCapability)
	}
	bad := func() (financialDraftContextEvidence, error) {
		return financialDraftContextEvidence{}, commands.Fail(commands.EvidenceStale)
	}
	final, binding, err := s.financialFinalRecord(workspace, request)
	if err != nil || final.Digest != digest {
		return bad()
	}
	candidate := binding.Candidate
	_, result, err := s.documentERP.resultRecord(workspace, candidate.ResultID, candidate.ResultDigest)
	if err != nil {
		return financialDraftContextEvidence{}, err
	}
	scale := result.Document.Connection.Connection.Scale
	gross, err := financialFinalGross(result.Total, scale)
	if err != nil || candidate.AmountProfile != financialFinalAmountProfile || candidate.Net != result.Total.Net || candidate.Tax != result.Total.Tax || gross != final.Subject.Amount || result.Packet.Currency != final.Subject.Currency || result.Document.Connection.Connection.Currency != final.Subject.Currency || final.Subject.Target == "" {
		return bad()
	}
	existing, err := a.draftForEconomicTarget(workspace, final.Subject.Target)
	if err != nil {
		return financialDraftContextEvidence{}, err
	}
	fields := map[string]string{
		"mode": "synthetic", "final_request": request, "final_digest": digest, "final_state": final.State,
		"customer_id": result.Packet.Customer, "period": result.Packet.Period, "currency": final.Subject.Currency,
		"scale": strconv.Itoa(scale), "total": gross, "current_eligibility": "unchecked",
	}
	if existing != nil {
		fields["existing_draft"], fields["existing_digest"] = existing.Draft.Spec.ID, existing.Draft.Digest
	}
	return financialDraftContextEvidence{Fields: fields, Existing: existing}, nil
}

func (a *financialIssuance) draftForEconomicTarget(workspace, target string) (*financialDraftObservation, error) {
	if target == "" || len(a.drafts) > financialIssuanceLimit || len(a.draftScopes) > financialIssuanceLimit {
		return nil, commands.Fail(commands.EvidenceStale)
	}
	var found *financialDraftObservation
	for id, record := range a.drafts {
		p, err := a.retainedDraft(record.Workspace, id)
		if err != nil {
			return nil, err
		}
		source, err := draftOriginSource(p.Draft)
		if err != nil {
			return nil, err
		}
		if record.Workspace != workspace || source.Basis.Subject.Target != target {
			continue
		}
		if found != nil {
			return nil, commands.Fail(commands.EvidenceStale)
		}
		found = &p
	}
	id, reserved := a.draftScopes[target]
	if reserved && (found == nil || found.Draft.Spec.ID != id) || !reserved && found != nil {
		return nil, commands.Fail(commands.EvidenceStale)
	}
	if found != nil {
		observed := a.observeDraft(*found)
		return &observed, nil
	}
	return nil, nil
}
