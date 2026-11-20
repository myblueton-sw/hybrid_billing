package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

type financialChargeCommand struct {
	s    *Server
	kind string
}

func (h financialChargeCommand) Resolve(workspace, target string) (commands.Target, commands.Resolution) {
	if workspace != h.s.bootWorkspace || !validRequestID(target) {
		return commands.Target{}, commands.NotFound
	}
	t := commands.Target{Revision: h.s.Revision()}
	if h.kind != "request" {
		v, _, err := h.s.financialChargeRecord(workspace, target)
		if err != nil {
			return commands.Target{}, commands.NotFound
		}
		t.AuthorID = v.Subject.Author
	}
	return t, commands.Found
}

func (h financialChargeCommand) Apply(ctx commands.Context) (commands.Outcome, error) {
	s := h.s
	if s.cfg.Seed != SeedSample || s.financialCharges == nil || s.approvals == nil || s.cfg.Commands.Fields < 24 || s.cfg.Commands.Value < 1024 {
		return commands.Outcome{}, commands.Fail(commands.UnsupportedCapability)
	}
	if ctx.User != s.answer.User() || ctx.Workspace != s.answer.Workspace() {
		return commands.Outcome{}, commands.Fail(commands.PermissionDenied)
	}
	if h.kind == "review" {
		v, b, err := s.financialChargeRecord(ctx.Workspace, ctx.Target)
		if err != nil {
			return commands.Outcome{}, financialPolicyError(err)
		}
		return commands.Outcome{Fields: financialChargeReview(v, b)}, nil
	}
	if h.kind == "request" {
		v, err := s.requestFinancialCharge(ctx)
		if err != nil {
			return commands.Outcome{}, err
		}
		return commands.Outcome{ResultRevision: s.Next(), BusinessState: v.State, Fields: financialChargeReceipt(v)}, nil
	}
	v, err := s.decideFinancialCharge(ctx, h.kind)
	if err != nil {
		return commands.Outcome{}, err
	}
	return commands.Outcome{ResultRevision: s.Next(), BusinessState: v.State, Fields: financialChargeReceipt(v)}, nil
}

func (s *Server) financialChargeDefinitions() []commands.Definition {
	out := []commands.Definition{}
	for _, kind := range []string{"request", "review", "approve", "reject"} {
		permission := commands.Permission{Action: gate.RequestApproval, Kind: "charge_approval"}
		var fields []commands.Field
		approval := kind == "approve" || kind == "reject"
		if kind == "review" {
			permission = commands.Permission{Action: gate.View, Kind: approvalKind, Fields: []string{approvalField}}
		}
		if kind == "request" {
			fields = []commands.Field{{Name: "candidate", Type: commands.ID, Required: true}, {Name: "digest", Type: commands.Text, Required: true}}
		}
		if approval {
			permission.Action = gate.Approve
			fields = []commands.Field{{Name: "digest", Type: commands.Text, Required: true}}
		}
		if kind == "reject" {
			fields = append(fields, commands.Field{Name: "reason", Type: commands.Text, Required: true})
		}
		out = append(out, commands.Definition{ID: "financial_charge." + kind, Permission: permission, Approval: approval, Mutating: kind != "review", Rationale: kind != "review", Fields: fields, Refusals: []commands.Code{commands.EvidenceStale}, Handler: financialChargeCommand{s: s, kind: kind}})
	}
	return out
}

func financialGateCode(d gate.Decision) commands.Code {
	if d.Allowed {
		return ""
	}
	if d.Code == gate.AuthorityUnavailable {
		return commands.AuthorityUnavailable
	}
	if d.Code == gate.SelfApprovalDenied {
		return commands.SelfApprovalDenied
	}
	return commands.PermissionDenied
}

func (s *Server) financialChargeBoundary(command string, call *commands.Call) commands.Code {
	var action gate.Action
	switch command {
	case "financial_charge.request":
		action = gate.RequestApproval
	case "financial_charge.review":
		action = gate.View
	case "financial_charge.approve", "financial_charge.reject":
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
	request := gate.Request{Action: action, Kind: "charge_approval", Resource: gate.Resource{WorkspaceID: call.Answer.Workspace()}}
	if action == gate.View {
		request.Kind = approvalKind
		request.Fields = []string{approvalField}
	}
	if action != gate.Approve {
		return financialGateCode(s.gate.Decide(call.Answer, call.Client, request))
	}
	registered := false
	for _, d := range s.commands.Catalogue() {
		if d.ID == command {
			registered = d.Approval && d.Mutating && d.Permission.Action == gate.Approve && d.Permission.Kind == "charge_approval" && len(d.Permission.Fields) == 0
		}
	}
	if !registered || call.Client != ConsoleClient {
		return commands.PermissionDenied
	}
	v, _, err := s.financialChargeRecord(call.Answer.Workspace(), call.Target)
	if err != nil {
		return commands.PermissionDenied
	}
	d := s.gate.CheckApproval(call.Answer, v.Subject.Author, request)
	if code := financialGateCode(d); code != "" {
		return code
	}
	if v.Requester == call.Answer.User() {
		return commands.SelfApprovalDenied
	}
	call.Client = ""
	return ""
}
