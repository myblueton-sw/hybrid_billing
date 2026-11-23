package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func (s *Server) currentFinancialChargeLocked(workspace, request, digest, consumer string) (approvals.Subject, error) {
	fail := func(code commands.Code) (approvals.Subject, error) { return approvals.Subject{}, commands.Fail(code) }
	if s.cfg.Seed != SeedSample || s.approvals == nil || s.financialCharges == nil {
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
	if actual.User() != consumer || actual.Workspace() != workspace {
		return fail(commands.PermissionDenied)
	}
	switch s.gate.Admit(s.answer) {
	case gate.Admitted:
	case gate.AdmitUnavailable:
		return fail(commands.AuthorityUnavailable)
	default:
		return fail(commands.Unauthenticated)
	}
	v, b, err := s.financialChargeRecord(workspace, request)
	if err != nil || v.Digest != digest {
		return fail(commands.EvidenceStale)
	}
	if v.State != approvals.StateApproved || len(v.Decisions) != 1 || v.Decisions[0].Decision.Verdict != approvals.VerdictApprove || len(v.Subject.Entries) != 0 || len(v.Excluded) != 0 {
		return fail(commands.PrerequisiteFailed)
	}
	approver := v.Decisions[0].Actor
	if approver == v.Subject.Author || approver == v.Requester {
		return fail(commands.SelfApprovalDenied)
	}
	d := s.gate.InspectApprovalAuthority(approver, v.Subject.Author, gate.Request{Action: gate.Approve, Kind: "charge_approval", Resource: gate.Resource{WorkspaceID: workspace}})
	if d.Code == gate.AuthorityUnavailable {
		return fail(commands.AuthorityUnavailable)
	}
	if !d.Allowed {
		return fail(commands.PermissionDenied)
	}
	if len(s.stranded) > 0 {
		return fail(commands.DependencyUnavailable)
	}
	if err := s.checkFinancialChargeEvidence(workspace, request, b); err != nil {
		return approvals.Subject{}, err
	}
	e := approvals.Execution{ID: request, Request: request, Stage: b.Subject.Stage, Target: b.Subject.Target, Digests: b.Subject.Digests, Amount: b.Subject.Amount, Currency: b.Subject.Currency}
	if err := s.approvals.CheckCurrent(workspace, e, approvals.Actor{ID: consumer, Kind: approvals.KindHuman}); err != nil {
		return fail(commands.PrerequisiteFailed)
	}
	return b.Subject, nil
}
