package server

import (
	"errors"
	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/erpledger"
	"github.com/myblueton-sw/hybrid_billing/modules/normalize"
	"reflect"
	"strconv"
	"testing"
	"time"
)

func finalizeInvoiceInWorld(t *testing.T, f world, prefix, period, candidateID, candidateDigest string, declaredRows ...invoiceRowOracle) (string, string) {
	requester, decider := f.as(Admin3Login), f.as(AdminLogin)
	defer closeInvoiceActors(f, requester, decider)
	chargeID := prefix + "-charge"
	requested := f.runWith(requester, f.s.Revision(), "financial_charge.request", chargeID, map[string]string{"candidate": candidateID, "digest": candidateDigest})
	f.ok(t, "second charge request", requested, approvals.StatePending, "")
	f.ok(t, "second charge approval", f.runWith(decider, f.s.Revision(), "financial_charge.approve", chargeID, map[string]string{"digest": field(requested, "digest")}), approvals.StateApproved, "")
	candidate, err := f.s.chargeProducer.candidates.financialCandidate(SampleWorkspace, candidateID)
	f.must(err)
	a := f.s.documentERP
	ledger, ok := a.ledger.(*erpledger.Registry)
	if !ok {
		t.Fatal("actual fixture ledger unavailable")
	}
	conn, err := ledger.ConnectionOf(SampleWorkspace, "synthetic-document-erp")
	f.must(err)
	mappings := []erpledger.MappingView{}
	for _, entry := range []struct{ kind, platform string }{{"customer", candidate.Envelope.Input.Customer}, {"product", "document-product"}, {"tax", "document-tax"}} {
		mapping, err := ledger.MappingOf(SampleWorkspace, conn.Connection.ID, entry.kind, entry.platform)
		f.must(err)
		mappings = append(mappings, mapping)
	}
	survey, err := ledger.Survey(SampleWorkspace, erpledger.Survey{Connection: conn.Connection.ID, Customer: candidate.Envelope.Input.Customer, Period: period, Outcome: erpledger.SurveyComplete, Evidence: prefix + "-survey"}, erpledger.Actor{ID: "synthetic-document-author", Kind: erpledger.KindHuman})
	f.must(err)
	basis := financialDocumentBasis{ID: prefix + "-document", Mode: "synthetic", Profile: "synthetic_individual_net_v1", Evidence: prefix + "-document-evidence", Workspace: SampleWorkspace, Request: chargeID, RequestDigest: field(requested, "digest"), Candidate: candidateID, CandidateDigest: candidateDigest, Path: erpledger.PathDirect, Document: candidate.Envelope.Input.Normalized.Document, Connection: conn, Mappings: mappings, Survey: survey, ExpectedNet: "15", ExpectedTax: "1"}
	oracle := map[string]invoiceRowOracle{}
	erpOracle := map[string]invoiceRowOracle{}
	charge, err := f.s.approvals.RequestOf(SampleWorkspace, chargeID)
	f.must(err)
	declared := map[string]invoiceRowOracle{}
	for i, row := range invoiceRows(declaredRows) {
		declared[prefix+"-line-"+strconv.Itoa(i)] = row
	}
	for _, origin := range candidate.Envelope.Input.Normalized.Rows {
		expected, ok := declared[origin.Key.Key]
		if !ok || origin.Line == "" {
			t.Fatal("unrecognized actual source line")
		}
		oracle[origin.Line] = expected
		source, err := normalizedDigest(struct {
			Schema, Workspace, Target string
			Row                       normalize.RowKey
		}{"hb_erp_economic_source_v1", SampleWorkspace, charge.Subject.Target, origin.Key})
		f.must(err)
		erpOracle["line-"+source] = expected
	}
	if len(oracle) != len(declared) {
		t.Fatal("missing source-line oracle")
	}
	for _, c := range candidate.Envelope.Result.Charges {
		expected, ok := oracle[c.Line]
		if !ok || c.Amount != expected.net {
			t.Fatal("rated charge differs from declared source amount")
		}
		basis.Lines = append(basis.Lines, financialDocumentLine{Line: c.Line, Rule: c.Rule, Kind: c.Kind, Product: "document-product", Quantity: expected.quantity, Unit: "document-unit", TaxCode: "document-tax", Tax: expected.tax, Evidence: prefix + "-line-evidence-" + c.Line})
	}
	w := documentWorld{f: f, a: a, ledger: ledger, basis: basis, actors: map[string]actor{AdminLogin: decider, Admin2Login: f.as(Admin2Login), Admin3Login: requester}}
	w.consumer = w.actors[Admin2Login]
	defer closeInvoiceActors(f, w.consumer)
	proposed, err := documentAs(w, Admin2Login, func(actor string) (financialDocumentView, error) {
		return a.proposeDocumentLocked(basis, actor, "Second invoice document evidence")
	})
	f.must(err)
	confirmed, err := decideDocument(w, Admin3Login, proposed, false, "Second invoice independent confirmation")
	f.must(err)
	w.digest = confirmed.Review.Digest
	preparationID := prefix + "-preparation"
	sub, err := w.prepare(preparationID)
	f.must(err)
	packet := financialERPResultPacket{Mode: "synthetic", Profile: "synthetic_result_one_to_one_v1", Evidence: prefix + "-reported-source", Preparation: preparationID, PreparationDigest: sub.Digest, Connection: conn.Connection.ID, External: prefix + "-reported-document", Currency: "USD", Customer: sub.Submission.Customer, Period: period, Revision: 1}
	for i, line := range sub.Submission.Lines {
		expected, ok := erpOracle[line.ID]
		if !ok || line.Net != expected.net || line.Tax != expected.tax || line.Quantity != expected.quantity {
			t.Fatal("preparation differs from declared source-line amounts")
		}
		packet.Rows = append(packet.Rows, financialERPResultRow{ID: prefix + "-row-" + strconv.Itoa(i), Line: line.ID, Product: "synthetic-product", Unit: "document-unit", Net: expected.net, Tax: expected.tax, Quantity: expected.quantity})
	}
	result := confirmResult(t, w, proposeResult(t, w, packet, prefix+"-result"))
	if len(result.Basis.Comparison) != 0 {
		t.Fatalf("second invoice comparison: %+v; lines: %+v", result.Basis.Comparison, sub.Submission.Lines)
	}
	_, err = applyResult(w, result)
	f.must(err)
	finalID := prefix + "-final"
	final := f.runWith(w.actors[Admin3Login], f.s.Revision(), "erp_final.request", finalID, map[string]string{"result": result.Basis.ID, "digest": result.Review.Digest})
	f.ok(t, "second invoice final request", final, approvals.StatePending, "")
	digest := field(final, "digest")
	f.ok(t, "second invoice final approval", f.runWith(w.actors[AdminLogin], f.s.Revision(), "erp_final.approve", finalID, map[string]string{"digest": digest}), approvals.StateApproved, "")
	return finalID, digest
}

func TestFinancialSecondInvoiceActualFinalPreservesFirst(t *testing.T) {
	f, _, _, draft, _ := draftApprovalWorld(t)
	first, err := f.s.approvals.RequestOf(SampleWorkspace, "issuance-final")
	f.must(err)
	adapter := f.s.documentERP
	oldResult, err := adapter.ledger.ResultOf(SampleWorkspace, "result-preparation")
	f.must(err)
	id, digest := captureInvoiceInWorld(t, f, f.as(MemberLogin), f.as(Admin3Login), "invoice-b", "2026-11", time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC))
	finalID, finalDigest := finalizeInvoiceInWorld(t, f, "invoice-b", "2026-11", id, digest)
	next, err := f.s.approvals.RequestOf(SampleWorkspace, finalID)
	f.must(err)
	after, err := f.s.approvals.RequestOf(SampleWorkspace, "issuance-final")
	f.must(err)
	currentResult, err := adapter.ledger.ResultOf(SampleWorkspace, "result-preparation")
	f.must(err)
	if next.Digest != finalDigest || next.Subject.Target == first.Subject.Target || next.Subject.Amount != "16" || !reflect.DeepEqual(after, first) || !reflect.DeepEqual(oldResult, currentResult) || f.s.documentERP != adapter || !reflect.DeepEqual(f.s.financialIssuance.drafts[draft.Spec.ID], draft) {
		t.Fatal("second invoice final reused economic target or changed first history")
	}
}

func TestFinancialSecondInvoiceAfterFirstApprovalExpiry(t *testing.T) {
	f, member, reviewer, draft, binding := draftApprovalWorld(t)
	a := f.s.financialIssuance
	reviewA, err := prepareDraftReviewAs(f, reviewer, "invoice-a-review", draft, binding)
	f.must(err)
	approvedA, err := approveDraftAs(f, reviewer, reviewA.Review)
	f.must(err)
	specB := issuanceSpec()
	specB.ID = "invoice-b-profile"
	reservation, err := proposeKindReservationForTest(f, f.as(AdminLogin), specB)
	f.must(err)
	reservation, err = confirmKindReservationForTest(f, f.as(Admin3Login), reservation)
	f.must(err)
	channelSpec := a.channels["mature-kind"].Spec
	channelSpec.ID = "invoice-b-kind-channel"
	channelSpec.Selector = financialChannelSelector{Kind: "kind", Reservation: reservation.Spec.ID, ReservationDigest: reservation.Digest}
	channelB, err := registerChannelAs(f, member, channelSpec)
	f.must(err)
	channelB, err = admitChannelAs(f, f.as(Admin3Login), channelB)
	f.must(err)
	first, err := f.s.approvals.RequestOf(SampleWorkspace, "issuance-final")
	f.must(err)
	execution := approvals.Execution{ID: first.Subject.ID, Request: first.Subject.ID, Stage: first.Subject.Stage, Target: first.Subject.Target, Digests: first.Subject.Digests, Amount: first.Subject.Amount, Currency: first.Subject.Currency}
	f.must(f.s.approvals.CheckCurrent(SampleWorkspace, execution, approvals.Actor{ID: SampleAdmin3, Kind: approvals.KindHuman}))
	f.advance(24*time.Hour + time.Minute)
	renewInvoiceAuthorities(t, f)
	err = f.s.approvals.CheckCurrent(SampleWorkspace, execution, approvals.Actor{ID: SampleAdmin3, Kind: approvals.KindHuman})
	var refusal approvals.Refusal
	if !errors.As(err, &refusal) || refusal.Reason != approvals.ReasonExpired {
		t.Fatal("first financial approval did not actually expire", err)
	}
	id, digest := captureInvoiceInWorld(t, f, f.as(MemberLogin), f.as(Admin3Login), "invoice-expiry-b", "2026-11", time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC))
	finalID, finalDigest := finalizeInvoiceInWorld(t, f, "invoice-expiry-b", "2026-11", id, digest)
	var profileB financialIssuanceProfile
	f.must(inputAuthorityAs(f, f.as(AdminLogin), func() error {
		var err error
		profileB, err = a.proposeProfile(SampleWorkspace, finalID, finalDigest, f.s.answer.User(), "Second invoice independent profile", specB)
		return err
	}))
	profileB, err = confirmIssuanceProfile(f, f.as(Admin3Login), profileB)
	f.must(err)
	changeSpec := a.changes["consume-kind"].Spec
	changeSpec.ID, changeSpec.Profile, changeSpec.ProfileDigest = "invoice-b-template", profileB.Spec.ID, profileB.Digest
	changeSpec.Channel, changeSpec.ChannelDigest = channelB.Spec.ID, channelB.Digest
	changeSpec.After.ID = "invoice-b-template-ref"
	changeSpec.Effective = f.s.clock().Add(time.Minute)
	change, err := requestChangeAs(f, f.as(Admin2Login), changeSpec)
	f.must(err)
	change, err = verifyChangeAs(f, f.as(AdminLogin), change)
	f.must(err)
	_, err = decideChangeAs(f, f.as("protected-fifth@example.invalid"), change)
	f.must(err)
	f.advance(time.Minute)
	draftB, err := submitDraftAs(f, f.as(MemberLogin), financialDraftSpec{ID: "invoice-b-draft", Profile: profileB.Spec.ID, ProfileDigest: profileB.Digest, Language: "en"})
	f.must(err)
	reviewer = f.as("draft-reviewer@example.invalid")
	reviewB, err := prepareDraftReviewAs(f, reviewer, "invoice-b-review", draftB, binding)
	f.must(err)
	approvedB, err := approveDraftAs(f, reviewer, reviewB.Review)
	f.must(err)
	if approvedB.Review.Module.References[0] != approvedA.Review.Module.References[0] || approvedB.Review.Module.References[1] != approvedA.Review.Module.References[1] || approvedB.Review.Coverage.Binding != approvedA.Review.Coverage.Binding || approvedB.Review.Coverage.Term != approvedA.Review.Coverage.Term || approvedB.Review.Module.References[2].Change != changeSpec.ID || draftB.Profile.Source.Basis.Subject.Target == draft.Profile.Source.Basis.Subject.Target {
		t.Fatal("second invoice did not reuse historical parties/terms with its own template and source")
	}
	replayA, err := approveDraftAs(f, reviewer, reviewA.Review)
	f.must(err)
	if !reflect.DeepEqual(replayA, approvedA) {
		t.Fatal("second invoice changed first frozen approval")
	}
	next, err := f.s.approvals.RequestOf(SampleWorkspace, finalID)
	f.must(err)
	after, err := f.s.approvals.RequestOf(SampleWorkspace, "issuance-final")
	f.must(err)
	if next.Digest != finalDigest || next.Subject.Target == first.Subject.Target || !reflect.DeepEqual(after, first) || !reflect.DeepEqual(f.s.financialIssuance.drafts[draft.Spec.ID], draft) {
		t.Fatal("second invoice after expiry changed first history")
	}
}
