package server

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/inputreview"
	"github.com/myblueton-sw/hybrid_billing/modules/normalize"
)

const sampleInputReview = "sample-input-review"

type serverInputReviewAuthority struct{ s *Server }

func (a serverInputReviewAuthority) Check(scope inputreview.Scope, actor, action string) error {
	s := a.s
	if scope.Workspace != s.bootWorkspace || actor == "" || actor != s.answer.User() || scope.Workspace != s.answer.Workspace() || action != inputreview.ConfirmAction && action != inputreview.RejectAction {
		return inputreview.ErrRejected
	}
	decision := s.gate.Decide(s.answer, ConsoleClient, gate.Request{Action: gate.ConfirmInput, Kind: inputauthority.ResourceKind, Resource: gate.Resource{WorkspaceID: scope.Workspace}})
	if decision.Allowed {
		return nil
	}
	if decision.Code == gate.AuthorityUnavailable {
		return gate.ErrUnavailable
	}
	return inputreview.ErrRejected
}

func (s *Server) seedInputReview() error {
	document := candidateDocument{Mode: "synthetic", Scope: inputreview.Scope{Workspace: SampleWorkspace, Provider: "sample-provider-01", Account: "sample-acct-01", Period: "2026-10"}, ID: "fixture-provider-document", Revision: 1, Currency: "USD", Field: "billed", Basis: normalize.Billed, Total: "15.0000"}
	body, err := json.Marshal(document)
	if err != nil {
		return err
	}
	_, err = s.inputReviews.prepareLocked(SampleWorkspace, "sample-normalized-charge", sampleInputReview, body, nil)
	return err
}

type inputReviewCommand struct {
	s    *Server
	kind string
}

func (h inputReviewCommand) Resolve(workspace, target string) (commands.Target, commands.Resolution) {
	if workspace != h.s.bootWorkspace || !validRequestID(target) {
		return commands.Target{}, commands.NotFound
	}
	return commands.Target{Revision: h.s.Revision()}, commands.Found
}

func inputReviewError(err error) error {
	switch {
	case errors.Is(err, inputreview.ErrInvalid), errors.Is(err, errCandidateInvalid):
		return commands.Fail(commands.InvalidRequest)
	case errors.Is(err, inputreview.ErrNotFound), errors.Is(err, inputreview.ErrRejected):
		return commands.Fail(commands.PermissionDenied)
	case errors.Is(err, inputreview.ErrConflict), errors.Is(err, errCandidateConflict):
		return commands.Fail(commands.EvidenceStale)
	case errors.Is(err, inputreview.ErrLimit), errors.Is(err, errCandidateLimit):
		return commands.Fail(commands.PrerequisiteFailed)
	default:
		return commands.Fail(commands.DependencyUnavailable)
	}
}

func inputReviewFields(record inputreview.Record) map[string]string {
	e := record.Evidence
	fields := map[string]string{"mode": "synthetic", "review": record.ID, "recorded_state": record.State, "digest": e.Digest, "workspace": e.Scope.Workspace, "provider": e.Scope.Provider, "account": e.Scope.Account, "period": e.Scope.Period, "method": e.Method, "recorded_verdict": e.Verdict, "compared_at": record.Compared.Format(time.RFC3339Nano)}
	if d := record.Decision; d != nil {
		fields["decided_by"] = d.Actor
		fields["decision_reason"] = d.Rationale
		fields["decided_at"] = d.At.Format(time.RFC3339Nano)
		fields["recorded_scope_revision"] = strconv.FormatUint(d.ScopeRevision, 10)
	}
	return fields
}

func (h inputReviewCommand) inspect(record inputreview.Record) (commands.Outcome, error) {
	r := h.s.inputReviews
	fields := inputReviewFields(record)
	publication, err := r.registry.Current(record.Evidence.Scope)
	if err != nil && !errors.Is(err, inputreview.ErrNotFound) {
		return commands.Outcome{}, inputReviewError(err)
	}
	fields["scope_revision"] = strconv.FormatUint(publication.Revision, 10)
	fields["current_eligibility"] = "unverified"
	fresh, _, err := r.currentEvidenceLocked(record.Evidence.Scope.Workspace, record.Evidence.Run)
	fields["current_evidence"] = "unavailable"
	if err == nil {
		fields["current_evidence"] = "changed"
		if fresh.Digest == record.Evidence.Digest && fresh.Verdict == record.Evidence.Verdict {
			fields["current_evidence"] = "matched"
			fields["current_eligibility"] = "not_eligible"
			if record.State == inputreview.Eligible && fresh.Verdict == inputreview.Match && publication.Record == record.ID && publication.Digest == record.Evidence.Digest && record.Decision != nil && publication.Revision == record.Decision.ScopeRevision {
				fields["current_eligibility"] = "eligible"
			}
		}
	} else if errors.Is(err, errCandidateConflict) || errors.Is(err, inputreview.ErrConflict) {
		fields["current_evidence"] = "changed"
	}
	held := r.bindings[[2]string{record.Evidence.Scope.Workspace, record.Evidence.Run}]
	if held == nil {
		return commands.Outcome{}, commands.Fail(commands.DependencyUnavailable)
	}
	fields["document"] = held.Document.ID
	fields["document_revision"] = strconv.Itoa(held.Document.Revision)
	fields["document_currency"] = held.Document.Currency
	fields["document_total"] = held.Document.Total
	fields["document_digest"] = held.Digest
	return commands.Outcome{Fields: fields}, nil
}

func (h inputReviewCommand) Apply(ctx commands.Context) (commands.Outcome, error) {
	r := h.s.inputReviews
	if r == nil {
		return commands.Outcome{}, commands.Fail(commands.UnsupportedCapability)
	}
	record, err := r.registry.Record(ctx.Workspace, ctx.Target)
	if err != nil {
		return commands.Outcome{}, inputReviewError(err)
	}
	if h.kind == "inspect" {
		return h.inspect(record)
	}
	revision, err := strconv.ParseUint(ctx.Fields["scope_revision"], 10, 32)
	if err != nil || strconv.FormatUint(revision, 10) != ctx.Fields["scope_revision"] {
		return commands.Outcome{}, commands.Fail(commands.InvalidRequest, "scope_revision")
	}
	intent := inputreview.Intent{Workspace: ctx.Workspace, ID: ctx.Target, Actor: ctx.User, Rationale: ctx.Rationale, Digest: ctx.Fields["digest"], ExpectedScope: revision}
	if h.kind == "confirm_publish" {
		record, err = h.confirmInput(intent)
	} else {
		record, err = r.registry.Reject(intent)
	}
	if err != nil {
		return commands.Outcome{}, inputReviewError(err)
	}
	fields := inputReviewFields(record)
	fields["receipt"] = "historical_decision"
	return commands.Outcome{Fields: fields, BusinessState: record.State, ResultRevision: h.s.Next()}, nil
}

func (h inputReviewCommand) confirmInput(intent inputreview.Intent) (inputreview.Record, error) {
	return h.s.inputReviews.registry.Confirm(intent)
}

func (s *Server) inputReviewDefinitions() []commands.Definition {
	out := []commands.Definition{}
	for _, kind := range []string{"inspect", "confirm_publish", "reject"} {
		mutation := kind != "inspect"
		var refusals []commands.Code
		if mutation {
			refusals = []commands.Code{commands.EvidenceStale}
		}
		permission := commands.Permission{Action: gate.ConfirmInput, Kind: inputauthority.ResourceKind}
		fields := []commands.Field{{Name: "digest", Type: commands.Text, Required: true}, {Name: "scope_revision", Type: commands.Text, Required: true}}
		if !mutation {
			permission.Action = gate.ViewCost
			permission.Fields = []string{inputauthority.EvidenceField}
			fields = nil
		}
		out = append(out, commands.Definition{ID: "input." + kind, Permission: permission, Mutating: mutation, Rationale: mutation, Fields: fields, Refusals: refusals, Handler: inputReviewCommand{s: s, kind: kind}})
	}
	return out
}
