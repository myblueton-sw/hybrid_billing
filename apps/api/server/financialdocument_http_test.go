package server

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/erpledger"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func documentRead(f world, a actor, command, id string, fields map[string]string) reply {
	body, _ := json.Marshal(map[string]any{"request_id": "document-read", "target_id": id, "body": fields})
	return f.call(command, string(body), a.cookie, a.csrf)
}

func publicDocument(t *testing.T) (documentWorld, string) {
	t.Helper()
	w := explicitDocumentWorld(t)
	w.a = w.f.s.documentERP
	if w.a == nil || len(w.a.documents) != 0 || len(w.a.owners) != 0 {
		t.Fatal("runtime absent or seeded document effects")
	}
	p := w.f.run(w.actors[Admin2Login], "document_evidence.propose", "public-document", map[string]string{"packet": packetJSON(t, packetFromBasis(w.basis))})
	w.f.ok(t, "public proposal", p, "pending", "")
	return w, field(p, "digest")
}

func TestFinancialDocumentPublicRuntimeJourney(t *testing.T) {
	w, digest := publicDocument(t)
	f := w.f
	proposer, decider := w.actors[Admin2Login], w.actors[Admin3Login]
	p := documentRead(f, proposer, "document_evidence.inspect", "public-document", map[string]string{"digest": digest})
	if p.status != 200 || field(p, "current_eligibility") != "unchecked" || field(p, "basis") == "" || field(p, "recorded_state") != "pending" {
		t.Fatal(p.raw)
	}
	var b financialDocumentBasis
	f.must(json.Unmarshal([]byte(field(p, "basis")), &b))
	if b.Document != w.basis.Document || b.Candidate != w.basis.Candidate || b.Lines[0].Rule != w.basis.Lines[0].Rule {
		t.Fatal("unowned derived evidence")
	}
	f.code(t, "pending cannot prepare", f.run(proposer, "erp_preparation.prepare", "public-preparation", map[string]string{"document": "public-document", "digest": digest}), 422, string(commands.PrerequisiteFailed))
	c := f.run(decider, "document_evidence.confirm", "public-document", map[string]string{"digest": digest})
	f.ok(t, "independent confirm", c, "confirmed", "")
	if field(c, "basis") != "" {
		t.Fatal("mutation disclosed Cost evidence")
	}
	prepared := f.run(proposer, "erp_preparation.prepare", "public-preparation", map[string]string{"document": "public-document", "digest": digest})
	f.ok(t, "actual runtime ERP prepare", prepared, erpledger.StatePrepared, "")
	actual, err := w.a.ledger.SubmissionOf(SampleWorkspace, "public-preparation")
	f.must(err)
	if actual.Total != (erpledger.Amounts{Net: "15.00", Tax: "1.00", Quantity: "5.00"}) || actual.Digest != field(prepared, "digest") || len(actual.Commands) != 0 {
		t.Fatal(actual)
	}
	r := documentRead(f, proposer, "erp_preparation.inspect", "public-preparation", map[string]string{})
	if r.status != 200 || field(r, "recorded_receipt") != actual.Digest || field(r, "ledger_observation") != "matching_prepared" || field(r, "observed_digest") != actual.Digest {
		t.Fatal(r.raw)
	}
	charge, err := f.s.approvals.RequestOf(SampleWorkspace, w.basis.Request)
	f.must(err)
	if len(charge.Executed) != 0 {
		t.Fatal("preparation consumed financial approval")
	}
	f.s.enterHold("synthetic test")
	f.ok(t, "revoke during hold", f.run(decider, "document_evidence.revoke", "public-document", map[string]string{"digest": digest}), "revoked", "")
	r = documentRead(f, proposer, "erp_preparation.inspect", "public-preparation", map[string]string{})
	if r.status != 200 || field(r, "recorded_receipt") != actual.Digest {
		t.Fatal(r.raw)
	}
	f.s.hold = nil
	f.code(t, "revoked document cannot recover", f.run(proposer, "erp_preparation.prepare", "public-preparation", map[string]string{"document": "public-document", "digest": digest}), 422, string(commands.PrerequisiteFailed))
}

func TestFinancialDocumentPublicUnknownRecoveryAndPermission(t *testing.T) {
	w, digest := publicDocument(t)
	f := w.f
	a := w.actors[Admin2Login]
	f.ok(t, "confirm", f.run(w.actors[Admin3Login], "document_evidence.confirm", "public-document", map[string]string{"digest": digest}), "confirmed", "")
	w.a.afterPrepare = func() error { return errors.New("synthetic lost response") }
	body := chargeBody(f, "unknown-prepare", "unknown-preparation", map[string]string{"document": "public-document", "digest": digest})
	unknown := f.call("erp_preparation.prepare", body, a.cookie, a.csrf)
	if unknown.status != 500 || unknown.body["operation_id"] == nil || unknown.body["next_action"] != string(commands.NextOperation) {
		t.Fatal(unknown.raw)
	}
	owner := w.a.owners[[2]string{SampleWorkspace, "unknown-preparation"}]
	if owner.receipt != "" {
		t.Fatal("invented receipt")
	}
	r := documentRead(f, a, "erp_preparation.inspect", "unknown-preparation", map[string]string{})
	if r.status != 200 || field(r, "recorded_receipt") != "" || field(r, "observed_digest") == "" || field(r, "ledger_observation") != "matching_prepared" || w.a.owners[[2]string{SampleWorkspace, "unknown-preparation"}].receipt != "" {
		t.Fatal(r.raw)
	}
	w.a.afterPrepare = nil
	replay := f.call("erp_preparation.prepare", body, a.cookie, a.csrf)
	if replay.status != unknown.status || replay.body["operation_id"] != unknown.body["operation_id"] {
		t.Fatal(replay.raw)
	}
	recovered := f.run(a, "erp_preparation.prepare", "unknown-preparation", map[string]string{"document": "public-document", "digest": digest})
	f.ok(t, "fresh key recovers exact owned effect", recovered, erpledger.StatePrepared, "")
	if field(recovered, "digest") != field(r, "observed_digest") || len(w.a.owners) != 1 {
		t.Fatal(recovered.raw)
	}
	for _, g := range f.s.admin.Grants() {
		if g.SubjectID == SampleAdmin2 && g.Terms.Action == gate.PrepareERPRecord {
			f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
		}
	}
	f.code(t, "revoked permission before cached replay", f.call("erp_preparation.prepare", body, a.cookie, a.csrf), 403, string(commands.PermissionDenied))
	f.code(t, "revoked permission before recovery", f.run(a, "erp_preparation.prepare", "unknown-preparation", map[string]string{"document": "public-document", "digest": digest}), 403, string(commands.PermissionDenied))
}

func TestFinancialDocumentPublicAdmissionAndCostBoundary(t *testing.T) {
	w := explicitDocumentWorld(t)
	f := w.f
	a := w.actors[Admin2Login]
	runtime := f.s.documentERP
	p := packetFromBasis(w.basis)
	p.Evidence = strings.Repeat("e", 129)
	bad := f.run(a, "document_evidence.propose", "oversized", map[string]string{"packet": packetJSON(t, p)})
	if bad.status == 200 || len(runtime.documents) != 0 {
		t.Fatal(bad.raw)
	}
	p = packetFromBasis(w.basis)
	limits := f.s.cfg.Commands
	f.s.cfg.Commands.Value = 2000
	bad = f.run(a, "document_evidence.propose", "output-too-large", map[string]string{"packet": packetJSON(t, p)})
	if bad.status == 200 || len(runtime.documents) != 0 {
		t.Fatal(bad.raw)
	}
	f.s.cfg.Commands = limits
	for _, g := range f.s.admin.Grants() {
		if g.SubjectID == SampleAdmin2 && g.Terms.Action == gate.ViewCost && g.Terms.Kind == "synthetic_document_evidence" {
			f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
		}
	}
	proposed := f.run(a, "document_evidence.propose", "without-cost", map[string]string{"packet": packetJSON(t, p)})
	f.ok(t, "separate proposal capability", proposed, "pending", "")
	for _, id := range []string{"without-cost", "missing"} {
		f.code(t, "no Cost before lookup", documentRead(f, a, "document_evidence.inspect", id, map[string]string{"digest": field(proposed, "digest")}), 403, string(commands.PermissionDenied))
	}
	if field(proposed, "basis") != "" || field(proposed, "proposal_reason") != "" {
		t.Fatal(proposed.raw)
	}
}
