package server

import (
	"errors"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/erpledger"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

type documentLedgerObservation struct {
	financialERPLedger
	mode string
}

func (l *documentLedgerObservation) SubmissionOf(workspace, key string) (erpledger.SubmissionView, error) {
	if l.mode == "unavailable" {
		return erpledger.SubmissionView{}, errors.New("synthetic lookup failure")
	}
	if l.mode == "missing" {
		return erpledger.SubmissionView{}, erpledger.ErrNotFound
	}
	v, err := l.financialERPLedger.SubmissionOf(workspace, key)
	if l.mode == "conflict" {
		v.Author = "foreign"
	}
	return v, err
}

func TestFinancialDocumentPublicUncertainObservationIsNotSettlement(t *testing.T) {
	w, digest := publicDocument(t)
	f := w.f
	a := w.actors[Admin2Login]
	f.ok(t, "confirm", f.run(w.actors[Admin3Login], "document_evidence.confirm", "public-document", map[string]string{"digest": digest}), "confirmed", "")
	w.a.afterPrepare = func() error { return errors.New("synthetic lost response") }
	f.code(t, "unknown effect", f.run(a, "erp_preparation.prepare", "observed", map[string]string{"document": "public-document", "digest": digest}), 500, string(commands.InternalError))
	w.a.afterPrepare = nil
	probe := &documentLedgerObservation{financialERPLedger: w.a.ledger}
	w.a.ledger = probe
	for _, mode := range []string{"unavailable", "missing", "conflict"} {
		probe.mode = mode
		r := documentRead(f, a, "erp_preparation.inspect", "observed", map[string]string{})
		if r.status != 200 || field(r, "ledger_observation") != mode || field(r, "recorded_receipt") != "" || field(r, "observed_digest") != "" {
			t.Fatal(r.raw)
		}
		if mode != "missing" {
			f.code(t, "owned observation remains uncertain", f.run(a, "erp_preparation.prepare", "observed", map[string]string{"document": "public-document", "digest": digest}), 500, string(commands.InternalError))
		}
		if w.a.owners[[2]string{SampleWorkspace, "observed"}].receipt != "" {
			t.Fatal("read settled owner")
		}
	}
	probe.mode = ""
	for _, g := range f.s.admin.Grants() {
		if g.SubjectID == SampleAdmin2 && g.Terms.Action == gate.PrepareERPRecord {
			f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
		}
	}
	f.code(t, "revoked before unresolved recovery", f.run(a, "erp_preparation.prepare", "observed", map[string]string{"document": "public-document", "digest": digest}), 403, string(commands.PermissionDenied))
	if w.a.owners[[2]string{SampleWorkspace, "observed"}].receipt != "" {
		t.Fatal("permission refusal erased uncertainty")
	}
}

func TestFinancialDocumentPublicSessionEnvelopeAndHold(t *testing.T) {
	w, digest := publicDocument(t)
	f := w.f
	a := w.actors[Admin3Login]
	body := chargeBody(f, "confirm-envelope", "public-document", map[string]string{"digest": digest})
	noCSRF := f.call("document_evidence.confirm", body, a.cookie, "")
	if noCSRF.status != 403 {
		t.Fatal(noCSRF.raw)
	}
	f.code(t, "stale revision", f.runWith(a, "r-0", "document_evidence.confirm", "public-document", map[string]string{"digest": digest}), 409, string(commands.RevisionConflict))
	f.advance(16 * time.Minute)
	f.code(t, "stepup", f.run(a, "document_evidence.confirm", "public-document", map[string]string{"digest": digest}), 403, CodeReauthentication)
	a = f.as(Admin3Login)
	f.s.enterHold("synthetic test")
	f.code(t, "held confirmation", f.run(a, "document_evidence.confirm", "public-document", map[string]string{"digest": digest}), 503, string(commands.DependencyUnavailable))
	if got := documentRead(f, a, "document_evidence.inspect", "public-document", map[string]string{"digest": digest}); got.status != 200 {
		t.Fatal(got.raw)
	}
	f.ok(t, "hold revocation", f.run(a, "document_evidence.revoke", "public-document", map[string]string{"digest": digest}), "revoked", "")
}

func TestFinancialDocumentPublicImpactAndCatalogue(t *testing.T) {
	w, _ := publicDocument(t)
	f := w.f
	a := w.actors[AdminLogin]
	staged := f.stage(a, "", SampleAdmin2)
	f.ok(t, "stage", staged, "draft", "")
	before := f.s.Revision()
	f.s.fault = func(at string) bool { return at == "document-evidence-impact" }
	f.code(t, "impact fails closed", f.prepare(a, field(staged, "draft"), "suspend_membership"), 503, string(commands.DependencyUnavailable))
	if f.s.Revision() != before {
		t.Fatal("impact failure mutated revision")
	}
	f.s.fault = nil
	p := f.prepare(a, field(staged, "draft"), "suspend_membership")
	f.ok(t, "prepare impact", p, "prepared", "")
	rows := f.bulkRead(a, "/"+field(p, "bulk")).body["impact"].(map[string]any)["pending"].([]any)
	found := false
	for _, raw := range rows {
		row := raw.(map[string]any)
		if row["store"] == "document_evidence_proposals" {
			found = true
			if row["count"] != 1.0 || row["rechecked"] != true {
				t.Fatal(row)
			}
		}
	}
	if !found {
		t.Fatal("pending document omitted")
	}
	for _, id := range []string{"document_evidence.propose", "document_evidence.confirm", "document_evidence.revoke", "document_evidence.inspect", "erp_preparation.prepare", "erp_preparation.inspect", "document_evidence.history"} {
		found = false
		for _, d := range f.s.commands.Catalogue() {
			if d.ID == id {
				found = true
				if d.Permission.Action != documentCommandAction(id) || !d.Supported {
					t.Fatal(d)
				}
			}
		}
		if !found {
			t.Fatal("missing command", id)
		}
	}
}

func TestFinancialDocumentPublicSeedNone(t *testing.T) {
	f := setup(t, SeedNone).ready()
	defer f.s.Close()
	login := "first@example.invalid"
	if got := f.auth("redeem", activationBody(f.links()[""], login), f.pre()); got.status != 204 {
		t.Fatal(got.raw)
	}
	cookie, csrf, err := f.s.openFor(LoginTriple(login), EmptyWorkspace, false)
	f.must(err)
	a := actor{cookie, csrf}
	if f.s.documentERP != nil {
		t.Fatal("SeedNone assembled fixture")
	}
	if code := f.s.documentBoundary("document_evidence.propose", commands.Call{}); code != commands.UnsupportedCapability {
		t.Fatal(code)
	}
	for _, command := range []string{"document_evidence.propose", "document_evidence.confirm", "document_evidence.revoke", "document_evidence.inspect", "erp_preparation.prepare", "erp_preparation.inspect", "document_evidence.history"} {
		if documentCommandAction(command) == gate.ViewCost {
			fields := map[string]string{}
			if command == "document_evidence.inspect" || command == "document_evidence.history" {
				fields["digest"] = "none"
				if command == "document_evidence.history" {
					fields["kind"] = "documents"
				}
			}
			f.code(t, "unseeded "+command, documentRead(f, a, command, "unseeded", fields), 501, string(commands.UnsupportedCapability))
		} else {
			f.code(t, "unseeded "+command, f.run(a, command, "unseeded", map[string]string{}), 501, string(commands.UnsupportedCapability))
		}
	}
}
