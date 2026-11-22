package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

type financialFinalCommand struct {
	s    *Server
	kind string
}

func (h financialFinalCommand) Resolve(workspace, target string) (commands.Target, commands.Resolution) {
	if workspace != h.s.bootWorkspace || !validRequestID(target) {
		return commands.Target{}, commands.NotFound
	}
	t := commands.Target{Revision: h.s.Revision()}
	if h.kind != "request" && h.kind != "preview" {
		v, _, err := h.s.financialFinalRecord(workspace, target)
		if err != nil {
			return commands.Target{}, commands.NotFound
		}
		t.AuthorID = v.Subject.Author
	}
	return t, commands.Found
}

func (h financialFinalCommand) Apply(ctx commands.Context) (commands.Outcome, error) {
	s := h.s
	if s.cfg.Seed != SeedSample || s.financialFinals == nil || s.approvals == nil || s.cfg.Commands.Fields < 24 || s.cfg.Commands.Value < 1024 {
		return commands.Outcome{}, commands.Fail(commands.UnsupportedCapability)
	}
	if ctx.User != s.answer.User() || ctx.Workspace != s.answer.Workspace() {
		return commands.Outcome{}, commands.Fail(commands.PermissionDenied)
	}
	if h.kind == "preview" {
		return s.previewFinancialFinal(ctx)
	}
	if h.kind == "prepare" {
		return s.prepareFinancialFinal(ctx)
	}
	if h.kind == "review" {
		v, b, err := s.financialFinalRecord(ctx.Workspace, ctx.Target)
		if err != nil {
			return commands.Outcome{}, financialPolicyError(err)
		}
		return commands.Outcome{Fields: financialFinalReview(v, b)}, nil
	}
	if h.kind == "request" {
		v, err := s.requestFinancialFinal(ctx)
		if err != nil {
			return commands.Outcome{}, err
		}
		return commands.Outcome{ResultRevision: s.Next(), BusinessState: v.State, Fields: financialFinalReceipt(v)}, nil
	}
	v, err := s.decideFinancialFinal(ctx, h.kind)
	if err != nil {
		return commands.Outcome{}, err
	}
	return commands.Outcome{ResultRevision: s.Next(), BusinessState: v.State, Fields: financialFinalReceipt(v)}, nil
}

func (s *Server) financialFinalDefinitions() []commands.Definition {
	out := []commands.Definition{}
	for _, kind := range []string{"request", "preview", "review", "prepare", "approve", "reject"} {
		permission := commands.Permission{Action: gate.RequestApproval, Kind: "erp_final_approval"}
		var fields []commands.Field
		approval := kind == "approve" || kind == "reject"
		if kind == "review" || kind == "prepare" {
			permission = commands.Permission{Action: gate.View, Kind: approvalKind, Fields: []string{approvalField}}
		}
		if kind == "request" || kind == "preview" {
			fields = []commands.Field{{Name: "result", Type: commands.ID, Required: true}, {Name: "digest", Type: commands.Text, Required: true}}
		}
		if approval {
			permission.Action = gate.Approve
		}
		if approval || kind == "prepare" {
			fields = []commands.Field{{Name: "digest", Type: commands.Text, Required: true}}
		}
		if kind == "reject" {
			fields = append(fields, commands.Field{Name: "reason", Type: commands.Text, Required: true})
		}
		out = append(out, commands.Definition{ID: "erp_final." + kind, Permission: permission, Approval: approval, Mutating: kind != "review" && kind != "prepare" && kind != "preview", Rationale: kind != "review" && kind != "prepare" && kind != "preview", Fields: fields, Refusals: []commands.Code{commands.EvidenceStale}, Handler: financialFinalCommand{s: s, kind: kind}})
	}
	return out
}

func (s *Server) financialFinalBoundary(command string, call *commands.Call) commands.Code {
	var action gate.Action
	switch command {
	case "erp_final.request", "erp_final.preview":
		action = gate.RequestApproval
	case "erp_final.review", "erp_final.prepare":
		action = gate.View
	case "erp_final.approve", "erp_final.reject":
		action = gate.Approve
	default:
		return ""
	}
	if admitted := s.gate.Admit(call.Answer); admitted != gate.Admitted {
		if admitted == gate.AdmitUnavailable {
			return commands.AuthorityUnavailable
		}
		return commands.PermissionDenied
	}
	request := gate.Request{Action: action, Kind: "erp_final_approval", Resource: gate.Resource{WorkspaceID: call.Answer.Workspace()}}
	if action == gate.View {
		request.Kind = approvalKind
		request.Fields = []string{approvalField}
	}
	if command == "erp_final.preview" {
		read := gate.Request{Action: gate.View, Kind: approvalKind, Fields: []string{approvalField}, Resource: gate.Resource{WorkspaceID: call.Answer.Workspace()}}
		if code := financialGateCode(s.gate.Decide(call.Answer, call.Client, read)); code != "" {
			return code
		}
	}
	if action != gate.Approve {
		return financialGateCode(s.gate.Decide(call.Answer, call.Client, request))
	}
	registered := false
	for _, d := range s.commands.Catalogue() {
		if d.ID == command {
			registered = d.Approval && d.Mutating && d.Permission.Action == gate.Approve && d.Permission.Kind == "erp_final_approval" && len(d.Permission.Fields) == 0
		}
	}
	if !registered || call.Client != ConsoleClient {
		return commands.PermissionDenied
	}
	v, binding, err := s.financialFinalRecord(call.Answer.Workspace(), call.Target)
	if err != nil {
		return commands.PermissionDenied
	}
	d := s.gate.CheckApproval(call.Answer, v.Subject.Author, request)
	if code := financialGateCode(d); code != "" {
		return code
	}
	if err := financialFinalContributorCheck(binding.Candidate, v.Requester, call.Answer.User()); err != nil {
		return commands.SelfApprovalDenied
	}
	call.Client = ""
	return ""
}
