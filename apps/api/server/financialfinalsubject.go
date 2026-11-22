package server

import (
	"reflect"
	"slices"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/erpledger"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
)

const financialFinalAmountProfile = "synthetic_reported_net_plus_tax_v1"

type financialFinalCandidate struct {
	Subject                                                             approvals.Subject
	Policy                                                              approvals.Policy
	Contributors                                                        []string
	ResultID, ResultDigest, ApplicationReceipt, AmountProfile, Net, Tax string
}

func (s *Server) financialFinalSubjectLocked(workspace, request, resultID, resultDigest, actor string) (financialFinalCandidate, error) {
	fail := func(code commands.Code) (financialFinalCandidate, error) {
		return financialFinalCandidate{}, commands.Fail(code)
	}
	if s.cfg.Seed != SeedSample || s.documentERP == nil || s.approvals == nil {
		return fail(commands.UnsupportedCapability)
	}
	if workspace != s.bootWorkspace || actor != s.answer.User() || s.answer.Workspace() != workspace || s.answer.Kind() != identity.Human || !validRequestID(actor) {
		return fail(commands.PermissionDenied)
	}
	found, ok := s.sessions.get(s.caller)
	if !ok || found.workspace != workspace {
		return fail(commands.Unauthenticated)
	}
	actual, code := s.access(found)
	if code != "" {
		return fail(code)
	}
	if actual.User() != actor || actual.Workspace() != workspace || actual.Kind() != identity.Human {
		return fail(commands.PermissionDenied)
	}
	if !validRequestID(request) || !validRequestID(resultID) || !validPlanHash(resultDigest) {
		return fail(commands.InvalidRequest)
	}
	a := s.documentERP
	record, basis, err := a.resultRecord(workspace, resultID, resultDigest)
	if err != nil {
		return financialFinalCandidate{}, err
	}
	if err := a.validateResultChains(); err != nil {
		return financialFinalCandidate{}, err
	}
	app, ok := a.resultApplications[[2]string{workspace, resultID}]
	if !ok || app.receipt == "" || app.after == nil {
		return fail(commands.PrerequisiteFailed)
	}
	if err := a.checkResultReceipt(basis, app); err != nil {
		return financialFinalCandidate{}, err
	}
	if err := a.currentResultEvidence(basis, record, actor); err != nil {
		return financialFinalCandidate{}, err
	}
	upstream, err := s.currentFinancialChargeLocked(workspace, basis.Document.Request, basis.Document.RequestDigest, actor)
	if err != nil {
		return financialFinalCandidate{}, err
	}
	policy, err := s.currentOwnedFinancialPolicyForStage(workspace, erpFinalPolicyStage)
	if err != nil {
		return financialFinalCandidate{}, err
	}
	if policy.Approvals != 1 || len(upstream.Entries) != 0 || basis.Packet.Mode != "synthetic" || basis.Packet.Profile != "synthetic_result_one_to_one_v1" || basis.Document.Mode != "synthetic" || basis.Document.Profile != "synthetic_individual_net_v1" || basis.Packet.Currency != upstream.Currency || basis.Packet.Currency != basis.Document.Connection.Connection.Currency || basis.Packet.Connection != basis.Document.Connection.Connection.ID {
		return fail(commands.PrerequisiteFailed)
	}
	gross, err := financialFinalGross(basis.Total, basis.Document.Connection.Connection.Scale)
	if err != nil {
		return financialFinalCandidate{}, err
	}
	contributors := []string{record.review.Proposer, upstream.Author}
	for _, id := range contributors {
		if !validRequestID(id) {
			return fail(commands.EvidenceStale)
		}
	}
	slices.Sort(contributors)
	contributors = slices.Compact(contributors)
	target, err := financialEvidenceDigest("erp_final_target", struct {
		ChargeTarget        string
		Document            organizations.DocumentKey
		Connection, Purpose string
	}{upstream.Target, basis.Document.Document, basis.Packet.Connection, basis.Preparation.Submission.Purpose})
	if err != nil {
		return financialFinalCandidate{}, err
	}
	subject := approvals.Subject{ID: request, Stage: approvals.StageERPFinal, Target: "erp-final-target-" + target, Author: record.review.Proposer, Amount: gross, Currency: basis.Packet.Currency, Digests: map[string]string{}}
	mandatory, err := financialEvidenceDigest("erp_final_result", struct {
		Basis                      financialERPResultBasis
		Review                     financialDocumentReview
		Snapshot                   erpledger.ResultView
		Receipt, Profile, Net, Tax string
		Upstream                   approvals.Subject
		Policy                     approvals.Policy
		Contributors               []string
	}{basis, record.review, *app.after, app.receipt, financialFinalAmountProfile, basis.Total.Net, basis.Total.Tax, upstream, policy, contributors})
	if err != nil {
		return financialFinalCandidate{}, err
	}
	for _, kind := range policy.Required {
		switch kind {
		case "result":
			subject.Digests[kind] = mandatory
		case "input", "rule":
			digest := upstream.Digests[kind]
			if !validPlanHash(digest) {
				return fail(commands.PrerequisiteFailed)
			}
			subject.Digests[kind] = digest
		case "mapping":
			subject.Digests[kind], err = financialEvidenceDigest("erp_final_mapping", struct {
				Connection  erpledger.ConnectionView
				Mappings    []erpledger.MappingView
				Preparation erpledger.SubmissionView
			}{basis.Document.Connection, basis.Document.Mappings, basis.Preparation})
		case "attribution":
			subject.Digests[kind], err = financialEvidenceDigest("erp_final_attribution", struct {
				Document     organizations.DocumentKey
				Contributors []string
			}{basis.Document.Document, contributors})
		default:
			return fail(commands.PrerequisiteFailed)
		}
		if err != nil {
			return financialFinalCandidate{}, err
		}
	}
	if subject.Digests["result"] != mandatory {
		return fail(commands.PrerequisiteFailed)
	}
	prepared, err := s.approvals.PrepareSubject(subject)
	if err != nil {
		return financialFinalCandidate{}, err
	}
	if !reflect.DeepEqual(subject, prepared) {
		return fail(commands.EvidenceStale)
	}
	policy.Required = slices.Clone(policy.Required)
	return financialFinalCandidate{Subject: prepared, Policy: policy, Contributors: slices.Clone(contributors), ResultID: resultID, ResultDigest: resultDigest, ApplicationReceipt: app.receipt, AmountProfile: financialFinalAmountProfile, Net: basis.Total.Net, Tax: basis.Total.Tax}, nil
}

func financialFinalContributorCheck(candidate financialFinalCandidate, requester, actor string) error {
	if !validRequestID(requester) || !validRequestID(actor) {
		return commands.Fail(commands.PermissionDenied)
	}
	if !validRequestID(candidate.Subject.Author) || len(candidate.Contributors) < 1 || !slices.IsSorted(candidate.Contributors) || !slices.Contains(candidate.Contributors, candidate.Subject.Author) {
		return commands.Fail(commands.EvidenceStale)
	}
	for i, id := range candidate.Contributors {
		if !validRequestID(id) || i > 0 && id == candidate.Contributors[i-1] {
			return commands.Fail(commands.EvidenceStale)
		}
	}
	if actor == requester || slices.Contains(candidate.Contributors, actor) {
		return commands.Fail(commands.SelfApprovalDenied)
	}
	return nil
}
