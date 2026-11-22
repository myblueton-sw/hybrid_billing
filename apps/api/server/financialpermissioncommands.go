package server

import (
	"errors"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/financialpermission"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
)

type financialPermissionCommand struct {
	s    *Server
	kind string
}

func (s *Server) financialPermissionBoundary(command string, call commands.Call) commands.Code {
	var a *serverGrantAuthority
	switch command {
	case "result_authority.propose_console", "result_authority.propose_human", "result_authority.approve", "result_authority.reject", "result_authority.withdraw", "result_authority.inspect", "result_authority.history", "result_authority.context":
		a = s.resultAuthority
	case "document_authority.propose_console", "document_authority.propose_human", "document_authority.approve", "document_authority.reject", "document_authority.withdraw", "document_authority.inspect":
		a = s.documentAuthority
	case "financial_permission.propose", "financial_permission.approve", "financial_permission.reject", "financial_permission.withdraw", "financial_permission.inspect":
		a = s.financialPermission
	case "financial_request_permission.propose_console", "financial_request_permission.propose_human", "financial_request_permission.approve", "financial_request_permission.reject", "financial_request_permission.withdraw", "financial_request_permission.inspect":
		a = s.financialRequestPermission
	default:
		return ""
	}
	if a == nil {
		return ""
	}
	if err := a.caller(call.Answer.Workspace(), call.Answer.User()); err != nil {
		if errors.Is(err, gate.ErrUnavailable) {
			return commands.DependencyUnavailable
		}
		return commands.PermissionDenied
	}
	if _, err := a.Check(call.Answer.Workspace(), call.Answer.User()); err != nil {
		if errors.Is(err, gate.ErrUnavailable) {
			return commands.DependencyUnavailable
		}
		return commands.PermissionDenied
	}
	return ""
}

func (h financialPermissionCommand) Resolve(workspace, target string) (commands.Target, commands.Resolution) {
	if workspace != h.s.bootWorkspace || !validRequestID(target) {
		return commands.Target{}, commands.NotFound
	}
	return commands.Target{Revision: h.s.Revision()}, commands.Found
}
func (h financialPermissionCommand) Apply(ctx commands.Context) (commands.Outcome, error) {
	a := h.s.financialPermission
	if a == nil || h.s.cfg.Commands.Fields < 20 || h.s.cfg.Commands.Value < 1024 {
		return commands.Outcome{}, commands.Fail(commands.UnsupportedCapability)
	}
	if err := a.caller(ctx.Workspace, ctx.User); err != nil {
		return commands.Outcome{}, inputAuthorityError(err)
	}
	if _, err := a.Check(ctx.Workspace, ctx.User); err != nil {
		return commands.Outcome{}, inputAuthorityError(err)
	}
	var record inputauthority.Record
	var err error
	switch h.kind {
	case "propose":
		before, e1 := time.Parse(time.RFC3339Nano, ctx.Fields["not_before"])
		after, e2 := time.Parse(time.RFC3339Nano, ctx.Fields["not_after"])
		if e1 != nil || e2 != nil {
			return commands.Outcome{}, commands.Fail(commands.InvalidRequest, "not_before", "not_after")
		}
		spec := inputauthority.Spec{Capability: financialpermission.Approve, Workspace: ctx.Workspace, Kind: financialpermission.DirectHuman, Recipient: ctx.Fields["recipient"], NotBefore: before, NotAfter: after}
		record, err = a.proposeLocked(inputauthority.Proposal{ID: ctx.Target, Spec: spec, Proposer: ctx.User, Rationale: ctx.Rationale})
	case "inspect":
		record, err = a.recordLocked(ctx.Workspace, ctx.Target)
	default:
		i := inputauthority.Intent{Workspace: ctx.Workspace, ID: ctx.Target, Actor: ctx.User, Digest: ctx.Fields["digest"], Rationale: ctx.Rationale}
		switch h.kind {
		case "approve":
			record, err = a.approveLocked(i)
		case "reject":
			record, err = a.rejectLocked(i)
		case "withdraw":
			record, err = a.withdrawLocked(i)
		default:
			return commands.Outcome{}, commands.Fail(commands.UnsupportedCommand)
		}
	}
	if err != nil {
		if errors.Is(err, inputauthority.ErrConflict) && record.Proposal.ID != "" && (record.State == inputauthority.Expired || record.State == inputauthority.Superseded) {
			h.s.counter++
		}
		return commands.Outcome{}, inputAuthorityError(err)
	}
	fields := inputAuthorityFields(record)
	fields["receipt"] = "historical_decision"
	out := commands.Outcome{Fields: fields}
	if h.kind != "inspect" {
		out.ResultRevision, out.BusinessState = h.s.Next(), record.State
	}
	return out, nil
}
func (s *Server) financialPermissionDefinitions() []commands.Definition {
	var out []commands.Definition
	for _, kind := range []string{"propose", "approve", "reject", "withdraw", "inspect"} {
		var fields []commands.Field
		if kind == "propose" {
			fields = append(fields, commands.Field{Name: "recipient", Type: commands.ID, Required: true})
			for _, name := range []string{"not_before", "not_after"} {
				fields = append(fields, commands.Field{Name: name, Type: commands.Text, Required: true})
			}
		} else if kind != "inspect" {
			fields = []commands.Field{{Name: "digest", Type: commands.Text, Required: true}}
		}
		out = append(out, commands.Definition{ID: "financial_permission." + kind, Permission: commands.Permission{Action: gate.AdministerInstallation, Kind: gate.InstallationKind}, Mutating: kind != "inspect", Rationale: kind != "inspect", Fields: fields, Handler: financialPermissionCommand{s: s, kind: kind}})
	}
	return out
}
