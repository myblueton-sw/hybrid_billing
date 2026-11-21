package server

import (
	"errors"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/resultauthority"
)

type resultCommand struct {
	s       *Server
	command string
}

func resultCommandAction(command string) gate.Action {
	switch command {
	case "result_evidence.propose":
		return gate.ProposeERPResult
	case "result_evidence.confirm":
		return gate.ConfirmERPResult
	case "result_evidence.revoke":
		return gate.RevokeERPResult
	case "erp_result.apply":
		return gate.ApplyERPResult
	case "result_evidence.inspect", "result_evidence.history", "erp_result.inspect":
		return gate.ViewCost
	}
	return ""
}

func (s *Server) resultBoundary(command string, call commands.Call) commands.Code {
	action := resultCommandAction(command)
	if action == "" {
		return ""
	}
	if s.cfg.Seed != SeedSample || s.documentERP == nil {
		return commands.UnsupportedCapability
	}
	if call.Client != ConsoleClient {
		return commands.PermissionDenied
	}
	err := s.documentERP.resultBoundaryCheck(command, call)
	if err == nil {
		return ""
	}
	var failure *commands.Failure
	if errors.As(err, &failure) {
		return failure.Code
	}
	return commands.DependencyUnavailable
}

func (a *financialERP) resultBoundaryCheck(command string, call commands.Call) error {
	workspace, actor := call.Answer.Workspace(), call.Answer.User()
	if err := a.resultPermission(workspace, actor, resultCommandAction(command)); err != nil {
		return err
	}
	if err := a.boundedResultFields(nil); err != nil {
		return err
	}
	if command != "result_evidence.confirm" && command != "erp_result.apply" {
		return nil
	}
	if len(call.Fields) != 1 || call.Fields[0].Name != "digest" {
		return commands.Fail(commands.InvalidRequest)
	}
	r, b, err := a.resultRecord(workspace, call.Target, call.Fields[0].Value)
	if err != nil {
		return err
	}
	if command == "result_evidence.confirm" {
		if actor == r.review.Proposer {
			return commands.Fail(commands.SelfApprovalDenied)
		}
		if r.review.State == "revoked" {
			return commands.Fail(commands.PrerequisiteFailed)
		}
		if err := a.resultStanding(workspace, r.review.Proposer, gate.ProposeERPResult); err != nil {
			return err
		}
		_, _, err = a.resultWatermark(b)
		return err
	}
	if err := a.currentResult(b, r, actor); err != nil {
		return err
	}
	app, owned, err := a.resultApplicationOwner(b, r.review.Digest)
	if err != nil {
		return err
	}
	if owned && app.actor != actor {
		return commands.Fail(commands.EvidenceStale)
	}
	if owned && app.receipt != "" {
		return a.checkResultReceipt(b, app)
	}
	return nil
}

func (h resultCommand) Resolve(workspace, target string) (commands.Target, commands.Resolution) {
	if workspace != h.s.bootWorkspace || !validRequestID(target) {
		return commands.Target{}, commands.NotFound
	}
	return commands.Target{Revision: h.s.Revision()}, commands.Found
}

func (h resultCommand) Apply(ctx commands.Context) (commands.Outcome, error) {
	a := h.s.documentERP
	if a == nil {
		return commands.Outcome{}, commands.Fail(commands.UnsupportedCapability)
	}
	if err := a.resultPermission(ctx.Workspace, ctx.User, resultCommandAction(h.command)); err != nil {
		return commands.Outcome{}, err
	}
	if err := a.boundedResultFields(nil); err != nil {
		return commands.Outcome{}, err
	}
	if h.command == "result_evidence.history" {
		fields, err := a.resultHistoryLocked(ctx)
		if err != nil {
			return commands.Outcome{}, err
		}
		return commands.Outcome{Fields: fields}, nil
	}
	var v financialERPResultView
	var err error
	if h.command == "result_evidence.propose" {
		b, e := a.buildResultBasis(ctx.Workspace, ctx.Target, ctx.Fields["packet"])
		if e != nil {
			return commands.Outcome{}, e
		}
		if e = a.admitPublicResult(b, ctx.Fields["packet"]); e != nil {
			return commands.Outcome{}, e
		}
		v, err = a.proposeResultLocked(ctx.Workspace, ctx.Target, ctx.User, ctx.Rationale, ctx.Fields["packet"])
	} else {
		r, b, e := a.resultRecord(ctx.Workspace, ctx.Target, ctx.Fields["digest"])
		if e != nil {
			return commands.Outcome{}, e
		}
		if _, _, e = a.resultApplicationOwner(b, r.review.Digest); e != nil {
			return commands.Outcome{}, e
		}
		if h.command == "result_evidence.inspect" || h.command == "erp_result.inspect" {
			fields, e := a.resultFields(a.resultView(r, b))
			if e != nil {
				return commands.Outcome{}, e
			}
			return commands.Outcome{Fields: fields}, nil
		}
		if e = a.admitPublicResult(b, r.raw); e != nil {
			return commands.Outcome{}, e
		}
		if h.command == "erp_result.apply" {
			v, err = a.applyResultLocked(ctx.Workspace, ctx.Target, ctx.Fields["digest"], ctx.User)
		} else {
			v, err = a.decideResultLocked(ctx.Workspace, ctx.Target, ctx.Fields["digest"], ctx.User, ctx.Rationale, h.command == "result_evidence.revoke")
		}
	}
	if err != nil {
		return commands.Outcome{}, err
	}
	state := v.Review.State
	if h.command == "erp_result.apply" {
		state = v.ApplicationState
	}
	return commands.Outcome{Fields: resultReceipt(v), BusinessState: state, ResultRevision: h.s.Next()}, nil
}

func (s *Server) resultDefinitions() []commands.Definition {
	var out []commands.Definition
	for _, id := range []string{"result_evidence.propose", "result_evidence.confirm", "result_evidence.revoke", "result_evidence.inspect", "result_evidence.history", "erp_result.apply", "erp_result.inspect"} {
		action := resultCommandAction(id)
		mutation := action != gate.ViewCost
		permission := commands.Permission{Action: action, Kind: resultauthority.ResourceKind}
		if !mutation {
			permission.Fields = []string{resultauthority.EvidenceField}
		}
		fields := []commands.Field{{Name: "digest", Type: commands.Text, Required: true}}
		if id == "result_evidence.propose" {
			fields = []commands.Field{{Name: "packet", Type: commands.Text, Required: true}}
		}
		if id == "result_evidence.history" {
			fields = append(fields, commands.Field{Name: "after", Type: commands.ID}, commands.Field{Name: "snapshot", Type: commands.ID})
		}
		refusals := []commands.Code{commands.EvidenceStale}
		if id == "result_evidence.confirm" {
			refusals = append(refusals, commands.SelfApprovalDenied)
		}
		out = append(out, commands.Definition{ID: id, Permission: permission, Mutating: mutation, Rationale: mutation, Fields: fields, Refusals: refusals, Handler: resultCommand{s: s, command: id}})
	}
	return out
}
