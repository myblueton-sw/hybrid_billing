package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/inputreview"
	"github.com/myblueton-sw/hybrid_billing/modules/normalize"
	"github.com/myblueton-sw/hybrid_billing/modules/reconcile"
)

type candidateFixtureAuthority struct {
	Scope   inputreview.Scope
	Allowed bool
}

func (a *candidateFixtureAuthority) Check(scope inputreview.Scope, actor, action string) error {
	if !a.Allowed || scope != a.Scope || actor != "fixture-operator" || action != inputreview.ConfirmAction && action != inputreview.RejectAction {
		return inputreview.ErrRejected
	}
	return nil
}

func candidateBasisFixture(t *testing.T) (*candidateInputReviews, *candidateFixtureAuthority, candidateDocument) {
	t.Helper()
	f := setup(t, SeedSample).ready()
	t.Cleanup(func() { f.s.Close() })
	f.s.domain.Lock()
	t.Cleanup(f.s.domain.Unlock)
	document := candidateDocument{Mode: "synthetic", Scope: inputreview.Scope{Workspace: SampleWorkspace, Provider: "sample-provider-01", Account: "sample-acct-01", Period: "2026-10"}, ID: "fixture-provider-document", Revision: 1, Currency: "USD", Field: "billed", Basis: normalize.Billed, Total: "15.0000"}
	authority := &candidateFixtureAuthority{Scope: document.Scope, Allowed: true}
	r, err := newCandidateInputReviews(f.s.chargeProducer, authority, f.s.clock)
	if err != nil {
		t.Fatal(err)
	}
	return r, authority, document
}

func candidateDocumentBytes(t *testing.T, document candidateDocument) []byte {
	t.Helper()
	data, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func candidateReviewIntent(record inputreview.Record, revision uint64) inputreview.Intent {
	return inputreview.Intent{Workspace: record.Evidence.Scope.Workspace, ID: record.ID, Actor: "fixture-operator", Rationale: "Reviewed the separately declared synthetic provider document", Digest: record.Evidence.Digest, ExpectedScope: revision}
}

func TestCandidateBasisActualConfirmationAndReadOnlyRecheck(t *testing.T) {
	r, authority, document := candidateBasisFixture(t)
	body := candidateDocumentBytes(t, document)
	record, err := r.prepareLocked(SampleWorkspace, "sample-normalized-charge", "review-1", body, nil)
	if err != nil || record.State != inputreview.Awaiting {
		t.Fatal(record, err)
	}
	held := r.bindings[[2]string{SampleWorkspace, "review-1"}]
	before, err := r.basis.Run(SampleWorkspace, held.Spec.ID)
	if err != nil || before.Internal != "15" || before.Spec.Total != "15" || before.Residual != "0" || before.Posted != 2 {
		t.Fatal(before, err)
	}
	if !bytes.Equal(held.Bytes, body) || held.Document.Total != "15.0000" {
		t.Fatal("original document changed")
	}
	body[0] = 'x'
	intent := candidateReviewIntent(record, 0)
	confirmed, err := r.registry.Confirm(intent)
	if err != nil || confirmed.State != inputreview.Eligible || confirmed.Decision.ScopeRevision != 1 {
		t.Fatal(confirmed, err)
	}
	publication, err := r.registry.Current(document.Scope)
	if err != nil || publication.Record != "review-1" || publication.Revision != 1 {
		t.Fatal(publication, err)
	}
	after, err := r.basis.Run(SampleWorkspace, held.Spec.ID)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("confirmation changed reconciliation", after, err)
	}
	repeated, err := r.registry.Confirm(intent)
	if err != nil || !reflect.DeepEqual(confirmed, repeated) {
		t.Fatal("confirmation retry", repeated, err)
	}
	authority.Allowed = false
	if _, err := r.registry.Confirm(intent); !errors.Is(err, inputreview.ErrRejected) {
		t.Fatal("revoked fixture authority", err)
	}
	if history, _ := r.registry.Record(SampleWorkspace, "review-1"); !reflect.DeepEqual(confirmed, history) {
		t.Fatal("history mutated")
	}
	if len(r.envelopes) != 1 || len(r.envelopes[record.Evidence.Digest]) == 0 {
		t.Fatal("comparison envelope not retained")
	}
}

func TestCandidateBasisRejectsSameTotalWrongMeaning(t *testing.T) {
	for _, kind := range []string{"basis", "currency", "scope", "field", "mode"} {
		t.Run(kind, func(t *testing.T) {
			r, _, document := candidateBasisFixture(t)
			switch kind {
			case "basis":
				document.Basis = normalize.Effective
			case "currency":
				document.Currency = "EUR"
			case "scope":
				document.Scope.Account = "foreign"
			case "field":
				document.Field = "effective"
			case "mode":
				document.Mode = "production"
			}
			if _, err := r.prepareLocked(SampleWorkspace, "sample-normalized-charge", "review-1", candidateDocumentBytes(t, document), nil); err == nil {
				t.Fatal("wrong document accepted")
			}
			if len(r.bindings) != 0 || len(r.documents) != 0 {
				t.Fatal("invalid document reserved state")
			}
		})
	}
	r, _, document := candidateBasisFixture(t)
	document.Total = "16"
	record, err := r.prepareLocked(SampleWorkspace, "sample-normalized-charge", "difference", candidateDocumentBytes(t, document), nil)
	if err != nil || record.State != inputreview.Blocked {
		t.Fatal("unexplained document", record, err)
	}
	if _, err := r.registry.Confirm(candidateReviewIntent(record, 0)); !errors.Is(err, inputreview.ErrRejected) {
		t.Fatal("unexplained published", err)
	}
}

func TestCandidateBasisNewDocumentAndVoidPreserveHistory(t *testing.T) {
	r, _, document := candidateBasisFixture(t)
	record, err := r.prepareLocked(SampleWorkspace, "sample-normalized-charge", "review-1", candidateDocumentBytes(t, document), nil)
	if err != nil {
		t.Fatal(err)
	}
	first, err := r.registry.Confirm(candidateReviewIntent(record, 0))
	if err != nil {
		t.Fatal(err)
	}
	document.Revision = 2
	next, err := r.prepareLocked(SampleWorkspace, "sample-normalized-charge", "review-2", candidateDocumentBytes(t, document), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.registry.Confirm(candidateReviewIntent(record, 0)); err == nil {
		t.Fatal("older document remained current")
	}
	if _, err := r.registry.Confirm(candidateReviewIntent(next, 1)); err != nil {
		t.Fatal(err)
	}
	held := r.bindings[[2]string{SampleWorkspace, "review-2"}]
	if err := r.basis.Void(SampleWorkspace, held.Spec.ID, reconcile.WrongFigures); err != nil {
		t.Fatal(err)
	}
	if _, err := r.registry.Confirm(candidateReviewIntent(next, 1)); err == nil {
		t.Fatal("voided basis replay")
	}
	if history, _ := r.registry.Record(SampleWorkspace, "review-1"); !reflect.DeepEqual(first, history) {
		t.Fatal("historical receipt changed")
	}
}

func TestCandidateBasisCannotAdoptUnrelatedRun(t *testing.T) {
	r, _, document := candidateBasisFixture(t)
	body := candidateDocumentBytes(t, document)
	prepared, err := r.prepareBindingLocked(SampleWorkspace, "sample-normalized-charge", "review-1", body, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.basis.Open(prepared.Spec); err != nil {
		t.Fatal(err)
	}
	if _, err := r.prepareLocked(SampleWorkspace, "sample-normalized-charge", "review-1", body, nil); !errors.Is(err, errCandidateConflict) {
		t.Fatal("unrelated run adopted", err)
	}
	if len(r.bindings) != 0 {
		t.Fatal("foreign run reserved")
	}
}
