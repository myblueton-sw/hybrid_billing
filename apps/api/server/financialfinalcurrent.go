package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func (s *Server) currentFinancialFinalLocked(workspace, request, digest, consumer string) (approvals.Subject, error) {
	fail := func(code commands.Code) (approvals.Subject, error) { return approvals.Subject{}, commands.Fail(code) }
	if s.cfg.Seed != SeedSample || s.approvals == nil || s.financialFinals == nil {
		return fail(commands.UnsupportedCapability)
	}
	if consumer == "" || consumer != s.answer.User() || workspace != s.answer.Workspace() || workspace != s.bootWorkspace || s.answer.Kind() != identity.Human {
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
	if actual.User() != consumer || actual.Workspace() != workspace || actual.Kind() != identity.Human {
		return fail(commands.PermissionDenied)
	}
	switch s.gate.Admit(s.answer) {
	case gate.Admitted:
	case gate.AdmitUnavailable:
		return fail(commands.AuthorityUnavailable)
	default:
		return fail(commands.Unauthenticated)
	}
	if s.held() {
		return fail(commands.PrerequisiteFailed)
	}
	v, b, err := s.financialFinalRecord(workspace, request)
	if err != nil || v.Digest != digest {
		return fail(commands.EvidenceStale)
	}
	if v.State != approvals.StateApproved || len(v.Decisions) != 1 || v.Decisions[0].Decision.Verdict != approvals.VerdictApprove || len(v.Subject.Entries) != 0 || len(v.Excluded) != 0 {
		return fail(commands.PrerequisiteFailed)
	}
	approver := v.Decisions[0].Actor
	if err := financialFinalContributorCheck(b.Candidate, b.Requester, approver); err != nil {
		return approvals.Subject{}, err
	}
	decision := s.gate.InspectApprovalAuthority(approver, v.Subject.Author, gate.Request{Action: gate.Approve, Kind: "erp_final_approval", Resource: gate.Resource{WorkspaceID: workspace}})
	if decision.Code == gate.AuthorityUnavailable {
		return fail(commands.AuthorityUnavailable)
	}
	if !decision.Allowed {
		return fail(commands.PermissionDenied)
	}
	if err := s.currentOwnedFinancialAuthorityCoverage(workspace, approver, erpFinalPolicyStage, v.Subject.Currency, v.Subject.Amount); err != nil {
		return approvals.Subject{}, err
	}
	if len(s.stranded) > 0 {
		return fail(commands.DependencyUnavailable)
	}
	if err := s.checkFinancialFinalEvidence(workspace, request, consumer, b); err != nil {
		return approvals.Subject{}, err
	}
	execution := approvals.Execution{ID: request, Request: request, Stage: v.Subject.Stage, Target: v.Subject.Target, Digests: v.Subject.Digests, Amount: v.Subject.Amount, Currency: v.Subject.Currency}
	if err := s.approvals.CheckCurrent(workspace, execution, approvals.Actor{ID: consumer, Kind: approvals.KindHuman}); err != nil {
		return fail(commands.PrerequisiteFailed)
	}
	return v.Subject, nil
}
