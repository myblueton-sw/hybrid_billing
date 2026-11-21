package server

import (
	"errors"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/erpledger"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

type documentWorld struct {
	f        world
	a        *financialERP
	ledger   *erpledger.Registry
	consumer actor
	basis    financialDocumentBasis
	digest   string
}

func explicitDocumentWorld(t *testing.T) documentWorld {
	t.Helper()
	f, requester, decider, _ := chargeWorld(t)
	original, err := f.s.chargeProducer.candidates.financialCandidate(SampleWorkspace, "subject-candidate")
	f.must(err)
	origin := original.Envelope.Input.Normalized
	fields := map[string]string{"binding": origin.Binding.ID, "input_review": origin.InputEligibility.Review.ID, "input_digest": origin.InputEligibility.Review.Evidence.Digest, "scope_revision": "1", "policy": origin.Pricing.Proposal.ID, "policy_digest": origin.Pricing.Digest}
	captured := policyMutate(f, f.as(MemberLogin), "capture", "document-candidate", fields)
	f.ok(t, "explicit document candidate", captured, "captured", "")
	requested := f.run(requester, "financial_charge.request", "document-charge", map[string]string{"candidate": "document-candidate", "digest": field(captured, "digest")})
	f.ok(t, "document charge", requested, approvals.StatePending, "")
	f.ok(t, "approve charge", f.run(decider, "financial_charge.approve", "document-charge", map[string]string{"digest": field(requested, "digest")}), approvals.StateApproved, "")
	candidate, err := f.s.chargeProducer.candidates.financialCandidate(SampleWorkspace, "document-candidate")
	f.must(err)
	numbers, err := decimal.NewProfile(candidateProfile())
	f.must(err)
	ledger, err := erpledger.NewRegistry(numbers, erpledger.Limits{Workspaces: 10, Connections: 10, Mappings: 1000, Anchors: 100, Submissions: 100, Lines: 100, Documents: 100, Rows: 100, Observations: 100, Links: 100, Surveys: 100}, f.s.clock)
	f.must(err)
	fixture := erpledger.Actor{ID: "synthetic-document-author", Kind: erpledger.KindHuman}
	conn, err := ledger.Connect(SampleWorkspace, erpledger.Connection{ID: "synthetic-document-erp", Entity: candidate.Envelope.Input.Normalized.Document.IssuerLegalEntityID, System: "synthetic-erp", Transport: erpledger.TransportAPI, Capabilities: []string{erpledger.CommandPrepare, erpledger.CommandMapping}, Currency: "USD", Scale: 2, Interval: time.Hour, Owner: fixture.ID}, fixture)
	f.must(err)
	mappings := []erpledger.MappingView{}
	for _, m := range []erpledger.Mapping{{Connection: conn.Connection.ID, Kind: "customer", Platform: candidate.Envelope.Input.Customer, External: "synthetic-customer"}, {Connection: conn.Connection.ID, Kind: "product", Platform: "document-product", External: "synthetic-product"}, {Connection: conn.Connection.ID, Kind: "tax", Platform: "document-tax", External: "synthetic-tax"}} {
		v, err := ledger.Map(SampleWorkspace, m, fixture)
		f.must(err)
		mappings = append(mappings, v)
	}
	survey, err := ledger.Survey(SampleWorkspace, erpledger.Survey{Connection: conn.Connection.ID, Customer: candidate.Envelope.Input.Customer, Period: "2026-10", Outcome: erpledger.SurveyComplete, Evidence: "synthetic-survey"}, fixture)
	f.must(err)
	b := financialDocumentBasis{ID: "explicit-document-basis", Mode: "synthetic", Profile: "synthetic_individual_net_v1", Evidence: "explicit-synthetic-document-v1", Workspace: SampleWorkspace, Request: "document-charge", RequestDigest: field(requested, "digest"), Candidate: "document-candidate", CandidateDigest: field(captured, "digest"), Path: erpledger.PathDirect, Document: candidate.Envelope.Input.Normalized.Document, Connection: conn, Mappings: mappings, Survey: survey, ExpectedNet: "15", ExpectedTax: "1"}
	for i, c := range candidate.Envelope.Result.Charges {
		b.Lines = append(b.Lines, financialDocumentLine{Line: c.Line, Rule: c.Rule, Kind: c.Kind, Product: "document-product", Quantity: []string{"2", "3"}[i], Unit: "document-unit", TaxCode: "document-tax", Tax: []string{"0.25", "0.75"}[i], Evidence: []string{"explicit-line-a", "explicit-line-b"}[i]})
	}
	a, err := newFinancialERP(f.s, ledger)
	f.must(err)
	f.s.domain.Lock()
	digest, err := a.retainDocumentLocked(b)
	f.s.domain.Unlock()
	f.must(err)
	return documentWorld{f: f, a: a, ledger: ledger, consumer: f.as(Admin2Login), basis: b, digest: digest}
}

func (w documentWorld) prepare(key string) (erpledger.SubmissionView, error) {
	var v erpledger.SubmissionView
	err := inputAuthorityAs(w.f, w.consumer, func() error {
		var err error
		v, err = w.a.prepareLocked(SampleWorkspace, w.basis.ID, w.digest, key, SampleAdmin2)
		return err
	})
	return v, err
}

func TestFinancialERPActualPreparationAndExactReplay(t *testing.T) {
	w := explicitDocumentWorld(t)
	before := w.f.s.Revision()
	v, err := w.prepare("document-preparation")
	w.f.must(err)
	if v.State != erpledger.StatePrepared || v.Total != (erpledger.Amounts{Net: "15.00", Tax: "1.00", Quantity: "5.00"}) || v.Author != SampleAdmin2 || len(v.Submission.Lines) != 2 || len(v.Commands) != 0 {
		t.Fatal(v)
	}
	actual, err := w.ledger.SubmissionOf(SampleWorkspace, "document-preparation")
	w.f.must(err)
	if actual.Digest != v.Digest || w.a.owners[[2]string{SampleWorkspace, "document-preparation"}].receipt != v.Digest {
		t.Fatal("missing actual retained receipt")
	}
	v.Submission.Lines[0].Tax = "99"
	again, err := w.prepare("document-preparation")
	w.f.must(err)
	if again.Submission.Lines[0].Tax == "99" || again.Digest != actual.Digest || w.f.s.Revision() != before {
		t.Fatal("history changed")
	}
	charge, err := w.f.s.approvals.RequestOf(SampleWorkspace, w.basis.Request)
	w.f.must(err)
	if len(charge.Executed) != 0 {
		t.Fatal("preparation consumed approval")
	}
	if _, err = w.prepare("different-preparation"); err == nil {
		t.Fatal("new key bypassed source coverage")
	}
	if len(w.a.owners) != 1 {
		t.Fatal("failed preflight reserved ownership")
	}
}

func TestFinancialERPRecoversOwnedInterruptedPreparation(t *testing.T) {
	w := explicitDocumentWorld(t)
	w.a.afterPrepare = func() error { return errors.New("synthetic interruption") }
	if _, err := w.prepare("interrupted"); err == nil {
		t.Fatal("expected interrupted receipt")
	}
	if w.a.owners[[2]string{SampleWorkspace, "interrupted"}].receipt != "" {
		t.Fatal("invented receipt")
	}
	actual, err := w.ledger.SubmissionOf(SampleWorkspace, "interrupted")
	w.f.must(err)
	w.a.afterPrepare = nil
	recovered, err := w.prepare("interrupted")
	w.f.must(err)
	if recovered.Digest != actual.Digest || len(w.a.owners) != 1 {
		t.Fatal("not exact recovery")
	}
	for _, g := range w.f.s.admin.Grants() {
		if g.SubjectID == SampleAdmin && g.Terms.Action == gate.Approve {
			w.f.must(w.f.s.admin.RevokeGrant(g.ID, g.Version))
		}
	}
	if _, err = w.prepare("interrupted"); err == nil {
		t.Fatal("history bypassed revoked authority")
	}
}
