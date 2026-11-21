package server

import (
	"errors"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/documentauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/erpledger"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

type documentCommand struct {
	s       *Server
	command string
}

func documentCommandAction(command string) gate.Action {
	switch command {
	case "document_evidence.propose":
		return gate.ProposeDocumentEvidence
	case "document_evidence.confirm":
		return gate.ConfirmDocumentEvidence
	case "document_evidence.revoke":
		return gate.RevokeDocumentEvidence
	case "erp_preparation.prepare":
		return gate.PrepareERPRecord
	case "document_evidence.inspect", "erp_preparation.inspect", "document_evidence.history", "document_evidence.advice", "document_evidence.context":
		return gate.ViewCost
	}
	return ""
}

func (s *Server) documentBoundary(command string, call commands.Call) commands.Code {
	action := documentCommandAction(command)
	if action == "" {
		return ""
	}
	if s.cfg.Seed != SeedSample || s.documentERP == nil {
		return commands.UnsupportedCapability
	}
	if call.Client != ConsoleClient {
		return commands.PermissionDenied
	}
	if err := s.documentERP.documentPermission(call.Answer.Workspace(), call.Answer.User(), action); err != nil {
		var failure *commands.Failure
		if errors.As(err, &failure) {
			return failure.Code
		}
		return commands.DependencyUnavailable
	}
	return ""
}

func (h documentCommand) Resolve(workspace, target string) (commands.Target, commands.Resolution) {
	if workspace != h.s.bootWorkspace || !validRequestID(target) {
		return commands.Target{}, commands.NotFound
	}
	return commands.Target{Revision: h.s.Revision()}, commands.Found
}

func documentCommandError(err error) error {
	if err == nil || errors.Is(err, errFinancialERPUncertain) {
		return err
	}
	var failure *commands.Failure
	if errors.As(err, &failure) {
		return failure
	}
	if errors.Is(err, erpledger.ErrRejected) || errors.Is(err, erpledger.ErrLimit) {
		return commands.Fail(commands.PrerequisiteFailed)
	}
	return financialChargeEvidenceError(err)
}

func (h documentCommand) Apply(ctx commands.Context) (commands.Outcome, error) {
	a := h.s.documentERP
	if a == nil {
		return commands.Outcome{}, commands.Fail(commands.UnsupportedCapability)
	}
	if err := a.documentPermission(ctx.Workspace, ctx.User, documentCommandAction(h.command)); err != nil {
		return commands.Outcome{}, err
	}
	if err := a.boundedDocumentFields(nil); err != nil {
		return commands.Outcome{}, err
	}
	if h.command == "document_evidence.context" {
		fields, err := a.documentContextLocked(ctx)
		if err != nil {
			return commands.Outcome{}, err
		}
		return commands.Outcome{Fields: fields}, nil
	}
	if h.command == "document_evidence.advice" {
		fields, err := a.documentAdviceLocked(ctx)
		if err != nil {
			return commands.Outcome{}, err
		}
		return commands.Outcome{Fields: fields}, nil
	}
	if h.command == "document_evidence.history" {
		fields, err := a.documentHistoryLocked(ctx)
		if err != nil {
			return commands.Outcome{}, err
		}
		return commands.Outcome{Fields: fields}, nil
	}
	if h.command == "erp_preparation.inspect" {
		fields, err := a.inspectPreparationLocked(ctx.Workspace, ctx.Target, ctx.User)
		if err != nil {
			return commands.Outcome{}, err
		}
		return commands.Outcome{Fields: fields}, nil
	}
	if h.command == "erp_preparation.prepare" {
		v, err := a.prepareLocked(ctx.Workspace, ctx.Fields["document"], ctx.Fields["digest"], ctx.Target, ctx.User)
		if err != nil {
			return commands.Outcome{}, documentCommandError(err)
		}
		fields := map[string]string{"mode": "synthetic", "preparation": ctx.Target, "document_digest": ctx.Fields["digest"], "digest": v.Digest, "recorded_state": v.State, "receipt": "historical_preparation"}
		return commands.Outcome{Fields: fields, BusinessState: v.State, ResultRevision: h.s.Next()}, nil
	}
	var v financialDocumentView
	var err error
	if h.command == "document_evidence.propose" {
		b, e := a.buildDocumentPacketLocked(ctx.Workspace, ctx.Target, ctx.User, ctx.Fields["packet"])
		if e != nil {
			return commands.Outcome{}, e
		}
		if e = a.admitPublicDocument(b); e != nil {
			return commands.Outcome{}, e
		}
		v, err = a.proposeDocumentLocked(b, ctx.User, ctx.Rationale)
	} else if h.command == "document_evidence.inspect" {
		v, err = a.inspectDocumentLocked(ctx.Workspace, ctx.Target, ctx.Fields["digest"], ctx.User)
		if err != nil {
			return commands.Outcome{}, err
		}
		fields, e := a.documentFields(v)
		if e != nil {
			return commands.Outcome{}, e
		}
		return commands.Outcome{Fields: fields}, nil
	} else {
		_, b, e := a.governedDocument(ctx.Workspace, ctx.Target, ctx.Fields["digest"])
		if e != nil {
			return commands.Outcome{}, e
		}
		if e = a.admitPublicDocument(b); e != nil {
			return commands.Outcome{}, e
		}
		v, err = a.decideDocumentLocked(ctx.Workspace, ctx.Target, ctx.Fields["digest"], ctx.User, ctx.Rationale, h.command == "document_evidence.revoke")
	}
	if err != nil {
		return commands.Outcome{}, documentCommandError(err)
	}
	return commands.Outcome{Fields: documentReceipt(v), BusinessState: v.Review.State, ResultRevision: h.s.Next()}, nil
}

func (s *Server) documentDefinitions() []commands.Definition {
	var out []commands.Definition
	for _, id := range []string{"document_evidence.propose", "document_evidence.confirm", "document_evidence.revoke", "document_evidence.inspect", "erp_preparation.prepare", "erp_preparation.inspect", "document_evidence.history", "document_evidence.advice", "document_evidence.context"} {
		action := documentCommandAction(id)
		permission := commands.Permission{Action: action, Kind: documentauthority.ResourceKind}
		mutation := action != gate.ViewCost
		if !mutation {
			permission.Fields = []string{documentauthority.EvidenceField}
		}
		var fields []commands.Field
		if id == "document_evidence.propose" {
			fields = []commands.Field{{Name: "packet", Type: commands.Text, Required: true}}
		} else if id != "erp_preparation.inspect" {
			fields = []commands.Field{{Name: "digest", Type: commands.Text, Required: true}}
		}
		if id == "document_evidence.history" {
			fields = append(fields, commands.Field{Name: "kind", Type: commands.Text, Required: true}, commands.Field{Name: "after", Type: commands.ID}, commands.Field{Name: "snapshot", Type: commands.ID})
		}
		if id == "document_evidence.context" {
			fields = append(fields, commands.Field{Name: "after", Type: commands.ID}, commands.Field{Name: "snapshot", Type: commands.ID})
		}
		if id == "erp_preparation.prepare" {
			fields = append(fields, commands.Field{Name: "document", Type: commands.ID, Required: true})
		}
		refusals := []commands.Code{commands.EvidenceStale}
		if id == "document_evidence.confirm" {
			refusals = append(refusals, commands.SelfApprovalDenied)
		}
		out = append(out, commands.Definition{ID: id, Permission: permission, Mutating: mutation, Rationale: mutation, Fields: fields, Refusals: refusals, Handler: documentCommand{s: s, command: id}})
	}
	return out
}
