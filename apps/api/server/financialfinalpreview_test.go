package server

import (
	"errors"
	"reflect"
	"strconv"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func TestFinancialFinalPreviewPermissionsBeforeLookup(t *testing.T) {
	for _, mode := range []string{"view-only", "request-only"} {
		t.Run(mode, func(t *testing.T) {
			w, digest := finalHTTPWorld(t)
			f := w.f
			if mode == "request-only" {
				provisionFinalHTTPRights(t, w)
				revoked := 0
				for _, grant := range f.s.admin.Grants() {
					if grant.SubjectID == SampleAdmin3 && grant.Terms.Action == gate.View && grant.Terms.Kind == approvalKind {
						f.must(f.s.admin.RevokeGrant(grant.ID, grant.Version))
						revoked++
					}
				}
				if revoked == 0 {
					t.Fatal("no read grants revoked")
				}
			}
			f.s.documentERP = nil
			r := documentRead(f, w.actors[Admin3Login], "erp_final.preview", "preview-permission", map[string]string{"result": "public-result", "digest": digest})
			f.code(t, "permission checked before unavailable producer", r, 403, "permission_denied")
		})
	}
}

func TestFinancialFinalPreviewReadOnlyAndStaleRevision(t *testing.T) {
	w, digest := finalHTTPWorld(t)
	f := w.f
	provisionFinalHTTPRights(t, w)
	body := map[string]string{"result": "public-result", "digest": digest}
	preview := func(id string) reply { return documentRead(f, w.actors[Admin3Login], "erp_final.preview", id, body) }
	before, err := w.ledger.ResultOf(SampleWorkspace, "result-preparation")
	f.must(err)
	revision := f.s.Revision()
	p := preview("preview-request")
	if p.status != 200 || field(p, "amount") != "16" || field(p, "net") != "15.00" || field(p, "tax") != "1.00" || field(p, "request_status") != "evidence_ready" || field(p, "revision") != revision || !validPlanHash(field(p, "candidate_digest")) {
		t.Fatal(p.raw)
	}
	if len(f.s.financialFinals) != 0 || f.s.Revision() != revision {
		t.Fatal("preview mutated state")
	}
	_, err = f.s.approvals.RequestOf(SampleWorkspace, "preview-request")
	if !errors.Is(err, approvals.ErrNotFound) {
		t.Fatal("preview inserted request")
	}
	after, err := w.ledger.ResultOf(SampleWorkspace, "result-preparation")
	f.must(err)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("preview changed ledger")
	}
	if got := preview("document-charge"); got.status != 200 || field(got, "request_status") != "target_unavailable" {
		t.Fatal(got.raw)
	}
	policy := f.run(w.actors[AdminLogin], "erp_final_policy.propose", "preview-policy-three", financialFields("3"))
	f.ok(t, "policy proposal", policy, "pending", "")
	f.ok(t, "policy confirmation", f.run(w.actors[Admin2Login], "erp_final_policy.confirm", "preview-policy-three", map[string]string{"digest": field(policy, "digest")}), "confirmed", "")
	f.code(t, "frozen preview revision invalidated", f.runWith(w.actors[Admin3Login], revision, "erp_final.request", "preview-request", body), 409, "revision_conflict")
	f.s.enterHold("preview-held-test")
	heldRevision := f.s.Revision()
	held := preview("held-preview")
	if held.status != 200 || field(held, "request_status") != "hold_active" || f.s.Revision() != heldRevision || len(f.s.financialFinals) != 0 {
		t.Fatal(held.raw)
	}
}

func TestFinancialFinalPreviewOutputAndCapacity(t *testing.T) {
	w, digest := finalHTTPWorld(t)
	f := w.f
	provisionFinalHTTPRights(t, w)
	read := func() reply {
		return documentRead(f, w.actors[Admin3Login], "erp_final.preview", "capacity-preview", map[string]string{"result": "public-result", "digest": digest})
	}
	before := f.s.Revision()
	saved := f.s.cfg.Commands
	f.s.cfg.Commands.Fields = 23
	f.code(t, "insufficient output fields", read(), 501, "unsupported_capability")
	f.s.cfg.Commands = saved
	f.s.cfg.Commands.Value = 1023
	f.code(t, "insufficient output value", read(), 501, "unsupported_capability")
	f.s.cfg.Commands = saved
	if f.s.Revision() != before || len(f.s.financialFinals) != 0 {
		t.Fatal("output refusal changed state")
	}
	for i := 0; i < approvalCap; i++ {
		f.s.financialFinals["capacity-"+strconv.Itoa(i)] = financialFinalBinding{}
	}
	got := read()
	if got.status != 200 || field(got, "request_status") != "capacity_reached" || f.s.Revision() != before || len(f.s.financialFinals) != approvalCap {
		t.Fatal(got.raw)
	}
	_, err := f.s.approvals.RequestOf(SampleWorkspace, "capacity-preview")
	if !errors.Is(err, approvals.ErrNotFound) {
		t.Fatal("capacity preview inserted request")
	}
}
