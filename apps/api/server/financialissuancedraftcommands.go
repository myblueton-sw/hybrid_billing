package server

import (
	"errors"
	"strings"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

type issuanceDraftCommand struct {
	s    *Server
	kind string
}

func (s *Server) issuanceDraftBoundary(command string, call commands.Call) commands.Code {
	kind := strings.TrimPrefix(command, "issuance_draft.")
	capability := draftCommandCapability(kind)
	if command != "issuance_draft."+kind || kind != "list" && kind != "inspect" && kind != "context" && capability == "" {
		return ""
	}
	if s.cfg.Seed != SeedSample || s.financialIssuance == nil {
		return commands.UnsupportedCapability
	}
	if call.Client != ConsoleClient {
		return commands.PermissionDenied
	}
	if err := s.financialIssuance.draftReadCaller(call.Answer.Workspace(), call.Answer.User()); err != nil {
		var failure *commands.Failure
		if errors.As(err, &failure) {
			return failure.Code
		}
		return commands.DependencyUnavailable
	}

	if capability != "" {
		fields := map[string]string{}
		for _, field := range call.Fields {
			fields[field.Name] = field.Value
		}
		if err := s.financialIssuance.draftMutationIdentity(call.Answer.Workspace(), call.Answer.User(), kind, call.Target, fields); err != nil {
			var failure *commands.Failure
			if errors.As(err, &failure) {
				return failure.Code
			}
			return commands.DependencyUnavailable
		}
	}
	return ""
}

func (h issuanceDraftCommand) Resolve(workspace, target string) (commands.Target, commands.Resolution) {
	if workspace != h.s.bootWorkspace || !validRequestID(target) || h.kind == "list" && target != workspace {
		return commands.Target{}, commands.NotFound
	}
	return commands.Target{Revision: h.s.Revision()}, commands.Found
}

func (h issuanceDraftCommand) Apply(ctx commands.Context) (commands.Outcome, error) {
	a := h.s.financialIssuance
	if h.s.cfg.Seed != SeedSample || a == nil {
		return commands.Outcome{}, commands.Fail(commands.UnsupportedCapability)
	}
	if draftCommandCapability(h.kind) != "" {
		return a.applyDraftMutation(h.kind, ctx)
	}
	var fields map[string]string
	var err error
	switch h.kind {
	case "context":
		fields, err = a.contextDraftFields(ctx.Workspace, ctx.User, ctx.Target, ctx.Fields["digest"], ctx.Fields["after_binding"], ctx.Fields["snapshot"])
	case "list":
		fields, err = a.listDraftFields(ctx.Workspace, ctx.User, ctx.Fields["after"], ctx.Fields["snapshot"])
	case "inspect":
		fields, err = a.inspectDraftFields(ctx.Workspace, ctx.User, ctx.Target, ctx.Fields["after_line"], ctx.Fields["snapshot"])
	default:
		return commands.Outcome{}, commands.Fail(commands.UnsupportedCommand)
	}
	if err != nil {
		return commands.Outcome{}, err
	}
	return commands.Outcome{Fields: fields}, nil
}

func (s *Server) issuanceDraftDefinitions() []commands.Definition {
	var out []commands.Definition
	for _, kind := range []string{"list", "inspect", "context"} {
		after := "after"
		if kind == "inspect" {
			after = "after_line"
		}
		fields := []commands.Field{{Name: after, Type: commands.Text}, {Name: "snapshot", Type: commands.Text}}
		if kind == "context" {
			fields[0].Name = "after_binding"
			fields = append(fields, commands.Field{Name: "digest", Type: commands.Text, Required: true})
		}
		out = append(out, commands.Definition{ID: "issuance_draft." + kind, Permission: commands.Permission{Action: gate.ViewCost, Kind: draftauthority.ResourceKind, Fields: []string{draftauthority.EvidenceField}}, Fields: fields, Refusals: []commands.Code{commands.EvidenceStale}, Handler: issuanceDraftCommand{s: s, kind: kind}})
	}

	for _, kind := range []string{"submit", "prepare_approval", "approve"} {
		names := []string{"digest"}
		if kind == "submit" {
			names = []string{"configuration", "configuration_digest", "final_request", "final_digest", "language"}
		}
		if kind == "prepare_approval" {
			names = []string{"draft", "draft_digest", "binding", "binding_digest"}
		}
		fields := make([]commands.Field, 0, len(names))
		for _, name := range names {
			fields = append(fields, commands.Field{Name: name, Type: commands.Text, Required: true})
		}
		action := gate.ApproveIssuanceDraft
		if kind == "submit" {
			action = gate.SubmitIssuanceDraft
		}
		out = append(out, commands.Definition{ID: "issuance_draft." + kind, Permission: commands.Permission{Action: action, Kind: draftauthority.ResourceKind}, Mutating: true, Rationale: true, Fields: fields, Refusals: []commands.Code{commands.EvidenceStale, commands.SelfApprovalDenied}, Handler: issuanceDraftCommand{s: s, kind: kind}})
	}
	return out
}
