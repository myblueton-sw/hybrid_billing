package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func documentHistoryRead(w documentWorld, a actor, kind, after, snapshot string) reply {
	fields := map[string]string{"digest": w.basis.RequestDigest, "kind": kind}
	if after != "" {
		fields["after"] = after
	}
	if snapshot != "" {
		fields["snapshot"] = snapshot
	}
	return documentRead(w.f, a, "document_evidence.history", w.basis.Request, fields)
}
func documentHistoryValue(t *testing.T, r reply, index int) map[string]string {
	t.Helper()
	var v map[string]string
	if err := json.Unmarshal([]byte(field(r, fmt.Sprintf("row_%02d", index))), &v); err != nil {
		t.Fatal(r.raw, err)
	}
	return v
}
func TestDocumentHistoryPublicDiscoveryAndUncertainty(t *testing.T) {
	w, digest := publicDocument(t)
	f := w.f
	proposer, decider := w.actors[Admin2Login], w.actors[Admin3Login]
	before := f.s.Revision()
	r := documentHistoryRead(w, decider, "documents", "", "")
	if r.status != 200 || field(r, "snapshot") != before || field(r, "current_eligibility") != "unchecked" {
		t.Fatal(r.raw)
	}
	row := documentHistoryValue(t, r, 0)
	if row["document"] != "public-document" || row["digest"] != digest || row["proposed_by"] != SampleAdmin2 || row["recorded_state"] != "pending" {
		t.Fatal(row)
	}
	if f.s.Revision() != before || len(w.a.owners) != 0 {
		t.Fatal("read changed state")
	}
	f.ok(t, "confirm discovered document", f.run(decider, "document_evidence.confirm", row["document"], map[string]string{"digest": row["digest"]}), "confirmed", "")
	w.a.afterPrepare = func() error { return errors.New("synthetic response loss") }
	f.code(t, "unknown", f.run(proposer, "erp_preparation.prepare", "discover-unknown", map[string]string{"document": row["document"], "digest": row["digest"]}), 500, string(commands.InternalError))
	w.a.afterPrepare = nil
	f.s.enterHold("synthetic history test")
	f.ok(t, "revoke", f.run(decider, "document_evidence.revoke", row["document"], map[string]string{"digest": row["digest"]}), "revoked", "")
	for _, g := range f.s.admin.Grants() {
		if g.SubjectID == SampleAdmin2 && g.Terms.Action == gate.PrepareERPRecord {
			f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
		}
	}
	for _, session := range f.s.sessions.byLogin(Admin2Login) {
		f.s.endProbe(session)
	}
	fresh := f.as(Admin2Login)
	before = f.s.Revision()
	r = documentHistoryRead(w, fresh, "preparations", "", "")
	if r.status != 200 {
		t.Fatal(r.raw)
	}
	owner := documentHistoryValue(t, r, 0)
	if owner["preparation"] != "discover-unknown" || owner["document_digest"] != digest || owner["recorded_receipt"] != "" {
		t.Fatal(owner)
	}
	if f.s.Revision() != before || w.a.owners[[2]string{SampleWorkspace, "discover-unknown"}].receipt != "" {
		t.Fatal("history settled uncertainty")
	}
	r = documentHistoryRead(w, fresh, "documents", "", "")
	if r.status != 200 || documentHistoryValue(t, r, 0)["recorded_state"] != "revoked" {
		t.Fatal(r.raw)
	}
	for _, g := range f.s.admin.Grants() {
		if g.SubjectID == SampleAdmin2 && g.Terms.Kind == "synthetic_document_evidence" && g.Terms.Action == gate.ViewCost {
			f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
		}
	}
	for _, id := range []string{w.basis.Request, "missing"} {
		f.code(t, "Cost before lookup", documentRead(f, fresh, "document_evidence.history", id, map[string]string{"digest": w.basis.RequestDigest, "kind": "documents"}), 403, string(commands.PermissionDenied))
	}
}
func TestDocumentHistoryPagingAndCorruption(t *testing.T) {
	w, _ := publicDocument(t)
	f := w.f
	a := w.actors[Admin2Login]
	for i := 0; i < 11; i++ {
		f.ok(t, "proposal", f.run(a, "document_evidence.propose", fmt.Sprintf("history-%02d", i), map[string]string{"packet": packetJSON(t, packetFromBasis(w.basis))}), "pending", "")
	}
	first := documentHistoryRead(w, a, "documents", "", "")
	if first.status != 200 || field(first, "next") != "history-09" || documentHistoryValue(t, first, 9)["document"] != "history-09" {
		t.Fatal(first.raw)
	}
	fields, ok := first.body["fields"].(map[string]any)
	if !ok || len(fields) != 17 {
		t.Fatal("page does not have exact bounded fields", first.raw)
	}
	second := documentHistoryRead(w, a, "documents", field(first, "next"), field(first, "snapshot"))
	if second.status != 200 || field(second, "next") != "" || documentHistoryValue(t, second, 0)["document"] != "history-10" || documentHistoryValue(t, second, 1)["document"] != "public-document" || field(second, "row_02") != "" {
		t.Fatal(second.raw)
	}
	f.code(t, "unknown cursor", documentHistoryRead(w, a, "documents", "missing", field(first, "snapshot")), 409, string(commands.EvidenceStale))
	f.code(t, "partial cursor", documentHistoryRead(w, a, "documents", field(first, "next"), ""), 422, string(commands.InvalidRequest))
	f.ok(t, "insert", f.run(a, "document_evidence.propose", "history-new", map[string]string{"packet": packetJSON(t, packetFromBasis(w.basis))}), "pending", "")
	f.code(t, "restart after change", documentHistoryRead(w, a, "documents", field(first, "next"), field(first, "snapshot")), 409, string(commands.EvidenceStale))
	empty := documentHistoryRead(w, a, "preparations", "", "")
	if empty.status != 200 || field(empty, "row_00") != "" || field(empty, "next") != "" {
		t.Fatal(empty.raw)
	}
	w.a.owners[[2]string{SampleWorkspace, "orphan"}] = financialERPOwnership{basis: "unknown"}
	f.code(t, "orphan association", documentHistoryRead(w, a, "documents", "", ""), 409, string(commands.EvidenceStale))
	delete(w.a.owners, [2]string{SampleWorkspace, "orphan"})
	f.s.cfg.Commands.Value = 1999
	f.code(t, "bounded projection", documentHistoryRead(w, a, "documents", "", ""), 501, string(commands.UnsupportedCapability))
}

func TestDocumentHistoryMemberAndRequestIsolation(t *testing.T) {
	w, digest := publicDocument(t)
	f := w.f
	documentGrantTo(t, f, "inspect", SampleMember)
	for _, g := range f.s.admin.Grants() {
		if g.SubjectID == SampleMember && g.Terms.Action == gate.ViewCost && g.Terms.Kind != "synthetic_document_evidence" {
			f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
		}
	}
	member := f.as(MemberLogin)
	before := f.s.Revision()
	got := documentHistoryRead(w, member, "documents", "", "")
	if got.status != 200 || documentHistoryValue(t, got, 0)["digest"] != digest || f.s.Revision() != before {
		t.Fatal(got.raw)
	}
	denied := documentRead(f, member, "document_evidence.history", "missing-request", map[string]string{"digest": w.basis.RequestDigest, "kind": "documents"})
	f.code(t, "unknown owned request", denied, 409, string(commands.EvidenceStale))
	f.code(t, "wrong digest", documentRead(f, member, "document_evidence.history", w.basis.Request, map[string]string{"digest": "wrong", "kind": "documents"}), 409, string(commands.EvidenceStale))
	f.code(t, "invalid kind", documentHistoryRead(w, member, "everything", "", ""), 422, string(commands.InvalidRequest))
	f.s.chargeProducer = nil
	f.s.enterHold("synthetic producer unavailable")
	got = documentHistoryRead(w, member, "documents", "", "")
	if got.status != 200 || documentHistoryValue(t, got, 0)["digest"] != digest {
		t.Fatal(got.raw)
	}
	w.a.documents[[2]string{"foreign-workspace", "foreign-document"}] = financialDocumentRecord{}
	w.a.owners[[2]string{"foreign-workspace", "foreign-preparation"}] = financialERPOwnership{}
	got = documentHistoryRead(w, member, "documents", "", "")
	if got.status != 200 || field(got, "row_01") != "" {
		t.Fatal(got.raw)
	}
}

func TestDocumentHistoryOtherRequestAndOwnerIntegrity(t *testing.T) {
	w, _ := publicDocument(t)
	f := w.f
	requester, approver, proposer := w.actors[Admin3Login], w.actors[AdminLogin], w.actors[Admin2Login]
	request := f.run(requester, "financial_charge.request", "other-document-charge", map[string]string{"candidate": w.basis.Candidate, "digest": w.basis.CandidateDigest})
	f.ok(t, "other request", request, "pending", "")
	f.ok(t, "other approval", f.run(approver, "financial_charge.approve", "other-document-charge", map[string]string{"digest": field(request, "digest")}), "approved", "")
	packet := packetFromBasis(w.basis)
	packet.Request, packet.RequestDigest = "other-document-charge", field(request, "digest")
	proposed := f.run(proposer, "document_evidence.propose", "other-document", map[string]string{"packet": packetJSON(t, packet)})
	f.ok(t, "other document", proposed, "pending", "")
	digest := field(proposed, "digest")
	f.ok(t, "other confirmation", f.run(requester, "document_evidence.confirm", "other-document", map[string]string{"digest": digest}), "confirmed", "")
	f.ok(t, "other preparation", f.run(proposer, "erp_preparation.prepare", "other-preparation", map[string]string{"document": "other-document", "digest": digest}), "prepared", "")
	before := f.s.Revision()
	key := [2]string{SampleWorkspace, "other-preparation"}
	original := w.a.owners[key]
	got := documentHistoryRead(w, proposer, "preparations", "", "")
	if got.status != 200 || field(got, "row_00") != "" {
		t.Fatal("other request leaked or refused", got.raw)
	}
	other := w
	other.basis.Request, other.basis.RequestDigest = packet.Request, packet.RequestDigest
	got = documentHistoryRead(other, proposer, "preparations", "", "")
	if got.status != 200 || documentHistoryValue(t, got, 0)["preparation"] != "other-preparation" {
		t.Fatal(got.raw)
	}
	for name, change := range map[string]func(*financialERPOwnership){
		"key":         func(o *financialERPOwnership) { o.submission.Key = "wrong" },
		"actor":       func(o *financialERPOwnership) { o.actor = "" },
		"receipt":     func(o *financialERPOwnership) { o.receipt = "wrong" },
		"association": func(o *financialERPOwnership) { o.basis = "wrong" },
	} {
		changed := original
		change(&changed)
		w.a.owners[key] = changed
		f.code(t, name, documentHistoryRead(w, proposer, "preparations", "", ""), 409, string(commands.EvidenceStale))
		if f.s.Revision() != before || !reflect.DeepEqual(w.a.owners[key], changed) {
			t.Fatal("refusal mutated ownership or revision")
		}
		w.a.owners[key] = original
	}
	if len(w.a.owners) != 1 || !reflect.DeepEqual(w.a.owners[key], original) || f.s.Revision() != before {
		t.Fatal("history changed state")
	}
}
