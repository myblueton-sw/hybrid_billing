package server

import (
	"errors"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

type issuanceDraftCommand struct {
	s    *Server
	kind string
}

func (s *Server) issuanceDraftBoundary(command string, call commands.Call) commands.Code {
	if command != "issuance_draft.list" && command != "issuance_draft.inspect" {
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
	var fields map[string]string
	var err error
	switch h.kind {
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
	for _, kind := range []string{"list", "inspect"} {
		after := "after"
		if kind == "inspect" {
			after = "after_line"
		}
		out = append(out, commands.Definition{ID: "issuance_draft." + kind, Permission: commands.Permission{Action: gate.ViewCost, Kind: draftauthority.ResourceKind, Fields: []string{draftauthority.EvidenceField}}, Fields: []commands.Field{{Name: after, Type: commands.Text}, {Name: "snapshot", Type: commands.Text}}, Refusals: []commands.Code{commands.EvidenceStale}, Handler: issuanceDraftCommand{s: s, kind: kind}})
	}
	return out
}
