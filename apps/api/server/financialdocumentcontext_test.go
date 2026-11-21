package server

import (
	"encoding/json"
	"errors"
	"github.com/myblueton-sw/hybrid_billing/modules/erpledger"
	"reflect"
	"sort"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func documentContextRead(w documentWorld, a actor, fields map[string]string) reply {
	if fields == nil {
		fields = map[string]string{"digest": w.basis.RequestDigest}
	}
	return documentRead(w.f, a, "document_evidence.context", w.basis.Request, fields)
}
func TestDocumentContextActualMemberProjection(t *testing.T) {
	w, _ := publicDocument(t)
	f := w.f
	documentGrantTo(t, f, "inspect", SampleMember)
	member := f.as(MemberLogin)
	for _, g := range f.s.admin.Grants() {
		if g.SubjectID == SampleMember && g.Terms.Action == gate.ViewCost && g.Terms.Kind != "synthetic_document_evidence" {
			f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
		}
	}
	before := f.s.Revision()
	got := documentContextRead(w, member, nil)
	if got.status != 200 || got.body["result_revision"] != before || field(got, "snapshot") != before || field(got, "next") != "" {
		t.Fatal(got.raw)
	}
	var permission documentAdviceState
	f.must(json.Unmarshal([]byte(field(got, "proposal_permission")), &permission))
	if permission.State != "denied" || permission.Code != commands.PermissionDenied {
		t.Fatal(got.raw)
	}
	proposerRead := documentContextRead(w, w.actors[Admin2Login], nil)
	f.must(json.Unmarshal([]byte(field(proposerRead, "proposal_permission")), &permission))
	if proposerRead.status != 200 || permission.State != "allowed" || permission.Code != "" || field(proposerRead, "header") != field(got, "header") {
		t.Fatal(proposerRead.raw)
	}
	projectionFields, ok := got.body["fields"].(map[string]any)
	if !ok || len(projectionFields) != 9 {
		t.Fatal("two rows plus seven metadata expected", got.raw)
	}
	var header map[string]json.RawMessage
	f.must(json.Unmarshal([]byte(field(got, "header")), &header))
	keys := []string{}
	for key := range header {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if !reflect.DeepEqual(keys, []string{"candidate", "candidate_digest", "currency", "customer", "erp", "organization", "period", "profile", "total"}) {
		t.Fatal(header)
	}
	var total, candidate, profile string
	f.must(json.Unmarshal(header["total"], &total))
	f.must(json.Unmarshal(header["candidate"], &candidate))
	f.must(json.Unmarshal(header["profile"], &profile))
	if total != "15" || candidate != w.basis.Candidate || profile != "synthetic_individual_net_v1" {
		t.Fatal(string(header["total"]), candidate, profile)
	}
	for i := 0; i < 2; i++ {
		row := documentHistoryValue(t, got, i)
		if len(row) != 4 || row["line"] == "" || row["amount"] == "" || row["kind"] != "rated" {
			t.Fatal(row)
		}
	}
	var erp map[string]json.RawMessage
	f.must(json.Unmarshal(header["erp"], &erp))
	keys = nil
	for key := range erp {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if !reflect.DeepEqual(keys, []string{"connection", "product", "scale", "tax_code"}) {
		t.Fatal(erp)
	}

	firstRow := documentHistoryValue(t, got, 0)
	continuation := documentContextRead(w, member, map[string]string{"digest": w.basis.RequestDigest, "after": firstRow["line"], "snapshot": field(got, "snapshot")})
	if continuation.status != 200 || field(continuation, "header") != field(got, "header") || documentHistoryValue(t, continuation, 0)["line"] != documentHistoryValue(t, got, 1)["line"] || field(continuation, "row_01") != "" {
		t.Fatal("context continuation changed header or rows", continuation.raw)
	}
	stale := documentContextRead(w, member, map[string]string{"digest": w.basis.RequestDigest, "after": firstRow["line"], "snapshot": "r0"})
	f.code(t, "stale context continuation", stale, 409, string(commands.EvidenceStale))

	if f.s.Revision() != before || len(w.a.documents) != 1 || len(w.a.owners) != 0 {
		t.Fatal("context changed state")
	}
	f.code(t, "wrong digest", documentContextRead(w, member, map[string]string{"digest": "wrong"}), 409, string(commands.EvidenceStale))
	for _, g := range f.s.admin.Grants() {
		if g.SubjectID == SampleAdmin && g.Terms.Action == gate.Approve {
			f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
		}
	}
	f.code(t, "current approver authority", documentContextRead(w, member, nil), 403, string(commands.PermissionDenied))
	if r := documentHistoryRead(w, member, "documents", "", ""); r.status != 200 {
		t.Fatal("history tied to current approval", r.raw)
	}
	for _, g := range f.s.admin.Grants() {
		if g.SubjectID == SampleMember && g.Terms.Kind == "synthetic_document_evidence" {
			f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
		}
	}
	f.code(t, "Cost before current approval", documentContextRead(w, member, nil), 403, string(commands.PermissionDenied))
}
func TestDocumentContextBounds(t *testing.T) {
	w, _ := publicDocument(t)
	a := w.actors[Admin2Login]
	f := w.f
	f.code(t, "missing cursor", documentContextRead(w, a, map[string]string{"digest": w.basis.RequestDigest, "after": "missing", "snapshot": f.s.Revision()}), 409, string(commands.EvidenceStale))
	f.code(t, "partial cursor", documentContextRead(w, a, map[string]string{"digest": w.basis.RequestDigest, "after": "missing"}), 422, string(commands.InvalidRequest))
	f.s.cfg.Commands.Value = 1999
	f.code(t, "output capability", documentContextRead(w, a, nil), 501, string(commands.UnsupportedCapability))
}

type documentContextLedger struct {
	financialERPLedger
	mode string
}

func (l documentContextLedger) ConnectionOf(workspace, id string) (erpledger.ConnectionView, error) {
	if l.mode == "unavailable" {
		return erpledger.ConnectionView{}, errors.New("synthetic unavailable")
	}
	v, err := l.financialERPLedger.ConnectionOf(workspace, id)
	if l.mode == "capability" {
		v.Connection.Capabilities = nil
	}
	if l.mode == "entity" {
		v.Connection.Entity = "other-entity"
	}
	return v, err
}
func TestDocumentContextERPFailuresAndAdvice(t *testing.T) {
	w, digest := publicDocument(t)
	original := w.a.ledger
	for _, mode := range []string{"unavailable", "capability", "entity"} {
		w.a.ledger = documentContextLedger{financialERPLedger: original, mode: mode}
		r := documentContextRead(w, w.actors[Admin2Login], nil)
		if r.status == 200 || field(r, "header") != "" {
			t.Fatal("invalid connector projection", mode, r.raw)
		}
		advice := documentAdviceRead(w, w.actors[Admin3Login], digest)
		state := documentAdviceValues(t, advice, "readiness")["confirm"]
		if mode == "unavailable" && (state.State != "unavailable" || state.Code != commands.DependencyUnavailable) {
			t.Fatal(advice.raw)
		}
		if mode != "unavailable" && (state.State != "blocked" || state.Code != commands.EvidenceStale) {
			t.Fatal(advice.raw)
		}
		if documentAdviceValues(t, advice, "readiness")["revoke"].State != "ready" {
			t.Fatal("revocation tied to ERP", advice.raw)
		}
	}
	if len(w.a.owners) != 0 {
		t.Fatal("read created owner")
	}
}
