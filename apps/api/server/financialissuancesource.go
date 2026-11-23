package server

import (
	"reflect"
	"slices"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
)

type financialIssuanceSource struct {
	Schema, Workspace, Profile, FinalRequest, FinalDigest string
	Subject                                               approvals.Subject
	Decision                                              approvals.DecisionView
	Contract                                              organizations.ContractRevision
	BillingProfile                                        organizations.BillingProfile
	BillingRevision                                       uint64
	Document                                              organizations.DocumentKey
	Candidate, CandidateDigest, Customer, Period          string
	Result, ResultDigest, ApplicationReceipt, Net, Tax    string
	Scale                                                 int
	Rows                                                  []financialERPResultRow
	Contributors                                          []string
}

type financialIssuanceSourceView struct {
	Basis  financialIssuanceSource
	Digest string
}

func (s *Server) financialIssuanceSourceLocked(workspace, request, digest, consumer string) (financialIssuanceSourceView, error) {
	fail := func(code commands.Code) (financialIssuanceSourceView, error) {
		return financialIssuanceSourceView{}, commands.Fail(code)
	}
	subject, err := s.currentFinancialFinalLocked(workspace, request, digest, consumer)
	if err != nil {
		return financialIssuanceSourceView{}, err
	}
	final, binding, err := s.financialFinalRecord(workspace, request)
	if err != nil || final.Digest != digest || !reflect.DeepEqual(subject, final.Subject) || len(final.Decisions) != 1 {
		return fail(commands.EvidenceStale)
	}
	_, result, err := s.documentERP.resultRecord(workspace, binding.Candidate.ResultID, binding.Candidate.ResultDigest)
	if err != nil {
		return financialIssuanceSourceView{}, err
	}
	app, ok := s.documentERP.resultApplications[[2]string{workspace, result.ID}]
	if !ok || app.receipt == "" || app.receipt != binding.Candidate.ApplicationReceipt || result.ID != binding.Candidate.ResultID || result.Packet.Currency != subject.Currency || result.Total.Net != binding.Candidate.Net || result.Total.Tax != binding.Candidate.Tax {
		return fail(commands.EvidenceStale)
	}
	charge, owned, err := s.financialChargeRecord(workspace, result.Document.Request)
	if err != nil || charge.Digest != result.Document.RequestDigest || owned.Candidate != result.Document.Candidate || owned.CandidateDigest != result.Document.CandidateDigest {
		return fail(commands.EvidenceStale)
	}
	if s.chargeProducer == nil {
		return fail(commands.DependencyUnavailable)
	}
	candidate, err := s.chargeProducer.candidates.financialCandidate(workspace, owned.Candidate)
	if err != nil {
		return financialIssuanceSourceView{}, financialChargeEvidenceError(err)
	}
	input := candidate.Envelope.Input
	origin := input.Normalized
	contract, profile, link := origin.Contract, origin.BillingProfile, origin.Binding
	if candidate.Envelope.Schema != "hb_charge_candidate_v7" || candidate.Digest != owned.CandidateDigest || input.Workspace != workspace || input.ID != owned.Candidate || origin.Document != result.Document.Document || contract.WorkspaceID != workspace || contract.ContractID != link.Contract || contract.Revision != link.ContractRevision || contract.CustomerID != input.Customer || input.Customer != result.Packet.Customer || profile.WorkspaceID != workspace || profile.ID != link.BillingProfile || link.BillingRevision == 0 || profile.Currency != subject.Currency || profile.Path != origin.Document.Path || origin.Document.WorkspaceID != workspace || origin.Document.Currency != subject.Currency || link.Period != result.Packet.Period || !validRequestID(contract.ContractID) || !validRequestID(profile.ID) {
		return fail(commands.EvidenceStale)
	}
	basis := financialIssuanceSource{
		Schema: "hb_issuance_source_v1", Workspace: workspace, Profile: binding.Candidate.AmountProfile, FinalRequest: request, FinalDigest: digest,
		Subject: subject, Decision: final.Decisions[0], Contract: contract, BillingProfile: profile, BillingRevision: link.BillingRevision, Document: origin.Document,
		Candidate: input.ID, CandidateDigest: candidate.Digest, Customer: input.Customer, Period: result.Packet.Period,
		Result: result.ID, ResultDigest: binding.Candidate.ResultDigest, ApplicationReceipt: app.receipt, Net: result.Total.Net, Tax: result.Total.Tax,
		Scale: result.Document.Connection.Connection.Scale, Rows: slices.Clone(result.Packet.Rows), Contributors: slices.Clone(binding.Candidate.Contributors),
	}
	basis.Contract.Components = slices.Clone(contract.Components)
	remaining := financialDocumentBytes
	if !candidateText(reflect.ValueOf(basis), &remaining) {
		return fail(commands.UnsupportedCapability)
	}
	bound, err := financialEvidenceDigest("issuance_source", basis)
	if err != nil {
		return financialIssuanceSourceView{}, err
	}
	return financialIssuanceSourceView{Basis: basis, Digest: bound}, nil
}
