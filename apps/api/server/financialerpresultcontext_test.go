package server

import (
	"encoding/json"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func TestFinancialERPResultDiscoveryEndCursorAndHistoricalContext(t *testing.T) {
	w, p := resultWorld(t)
	f := w.f
	f.s.documentERP = w.a

	a := w.actors[Admin2Login]
	first := documentRead(f, a, "result_evidence.preparations", SampleWorkspace, map[string]string{})
	if first.status != 200 || field(first, "next") != "" || field(first, "row_00") == "" {
		t.Fatal(first.raw)
	}
	fields := map[string]string{"after": p.Preparation, "snapshot": field(first, "snapshot")}
	second := documentRead(f, a, "result_evidence.preparations", SampleWorkspace, fields)
	if second.status != 200 || field(second, "row_00") != "" || field(second, "next") != "" || field(second, "workspace") != field(first, "workspace") || field(second, "snapshot") != field(first, "snapshot") {
		t.Fatal(second.raw)
	}
	_, err := decideDocument(w, Admin3Login, financialDocumentView{Basis: w.basis, Review: financialDocumentReview{Digest: w.digest}}, true, "Historical only")
	f.must(err)
	w.a.ledger = &documentLedgerObservation{financialERPLedger: w.a.ledger, mode: "unavailable"}
	context := documentRead(f, a, "result_evidence.context", p.Preparation, map[string]string{"digest": p.PreparationDigest})
	if context.status != 200 || field(context, "row_00") == "" {
		t.Fatal(context.raw)
	}
	var ready documentAdviceState
	f.must(json.Unmarshal([]byte(field(context, "proposal_readiness")), &ready))
	if ready.State != "unavailable" || ready.Code != commands.DependencyUnavailable {
		t.Fatal(ready)
	}
	if got := documentRead(f, a, "result_evidence.preparations", SampleWorkspace, map[string]string{}); got.status != 200 {
		t.Fatal(got.raw)
	}
	fields["snapshot"] = "r-0"
	f.code(t, "stale discovery cursor", documentRead(f, a, "result_evidence.preparations", SampleWorkspace, fields), 409, string(commands.EvidenceStale))
}

func TestFinancialERPResultDiscoveryWithoutDocumentRights(t *testing.T) {
	w, p := resultWorld(t)
	f := w.f
	f.s.documentERP = w.a
	for _, g := range f.s.admin.Grants() {
		if g.SubjectID == SampleAdmin2 && g.Terms.Kind == "synthetic_document_evidence" {
			f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
		}
	}
	a := w.actors[Admin2Login]
	discovery := documentRead(f, a, "result_evidence.preparations", SampleWorkspace, map[string]string{})
	if discovery.status != 200 || field(discovery, "row_00") == "" {
		t.Fatal(discovery.raw)
	}
	var row map[string]string
	f.must(json.Unmarshal([]byte(field(discovery, "row_00")), &row))
	if row["preparation"] != p.Preparation || row["preparation_digest"] != p.PreparationDigest {
		t.Fatal(row)
	}
	context := documentRead(f, a, "result_evidence.context", p.Preparation, map[string]string{"digest": p.PreparationDigest})
	if context.status != 200 || field(context, "current_eligibility") != "unchecked" {
		t.Fatal(context.raw)
	}
	var reference map[string]string
	f.must(json.Unmarshal([]byte(field(context, "row_00")), &reference))
	if len(reference) != 3 || reference["expected_product"] != "synthetic-product" || reference["expected_unit"] != "document-unit" || reference["line"] == "" {
		t.Fatal(reference)
	}
	var ready documentAdviceState
	f.must(json.Unmarshal([]byte(field(context, "proposal_readiness")), &ready))
	if ready.State != "ready" {
		t.Fatal(ready)
	}
	f.code(t, "document Cost remains denied", documentRead(f, a, "document_evidence.history", w.basis.Request, map[string]string{"digest": w.basis.RequestDigest, "kind": "preparations"}), 403, string(commands.PermissionDenied))
	f.code(t, "foreign discovery target", documentRead(f, a, "result_evidence.preparations", "other-workspace", map[string]string{}), 409, string(commands.EvidenceStale))
	if len(w.a.results) != 0 || len(w.a.resultApplications) != 0 {
		t.Fatal("reads created result effects")
	}
}

func TestFinancialERPResultContextProposalPermissionAndHold(t *testing.T) {
	w, p := resultWorld(t)
	f := w.f
	f.s.documentERP = w.a
	a := w.actors[Admin2Login]
	f.s.enterHold("result-context")
	read := func() documentAdviceState {
		t.Helper()
		got := documentRead(f, a, "result_evidence.context", p.Preparation, map[string]string{"digest": p.PreparationDigest})
		if got.status != 200 {
			t.Fatal(got.raw)
		}
		var out documentAdviceState
		f.must(json.Unmarshal([]byte(field(got, "proposal_readiness")), &out))
		return out
	}
	if got := read(); got.State != "blocked" || got.Code != commands.PrerequisiteFailed {
		t.Fatal(got)
	}
	f.s.hold = nil
	for _, g := range f.s.admin.Grants() {
		if g.SubjectID == SampleAdmin2 && g.Terms.Action == gate.ProposeERPResult {
			f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
		}
	}
	if got := read(); got.State != "unchecked" || got.Code != commands.PermissionDenied {
		t.Fatal(got)
	}
	for _, g := range f.s.admin.Grants() {
		if g.SubjectID == SampleAdmin2 && g.Terms.Action == gate.ViewCost && g.Terms.Kind == "synthetic_erp_result_evidence" {
			f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
		}
	}
	for _, target := range []string{p.Preparation, "missing"} {
		f.code(t, "Cost before lookup", documentRead(f, a, "result_evidence.context", target, map[string]string{"digest": p.PreparationDigest}), 403, string(commands.PermissionDenied))
	}
}

func TestFinancialERPResultDiscoveryCorruptOwnership(t *testing.T) {
	w, p := resultWorld(t)
	f := w.f
	f.s.documentERP = w.a
	key := [2]string{SampleWorkspace, p.Preparation}
	owner := w.a.owners[key]
	owner.submission.Lines[0].Product = "tampered"
	w.a.owners[key] = owner
	f.code(t, "corrupt association", documentRead(f, w.actors[Admin2Login], "result_evidence.preparations", SampleWorkspace, map[string]string{}), 409, string(commands.EvidenceStale))
}
