package server

import (
	"errors"
	"testing"
)

func TestFinancialERPResultImpactRetainsUnknownAndRejectsCorruption(t *testing.T) {
	w, p := resultWorld(t)
	w.f.s.documentERP = w.a
	v := proposeResult(t, w, p, "impact")
	pending, unknown, err := resultEvidenceCounts(w.f.s)
	w.f.must(err)
	if pending[SampleAdmin2] != 1 || len(unknown) != 0 {
		t.Fatal("pending category")
	}
	v = confirmResult(t, w, v)
	w.a.afterResultWrite = func(string) error { return errors.New("interrupted") }
	_, err = applyResult(w, v)
	if !errors.Is(err, errFinancialERPResultUncertain) {
		t.Fatal(err)
	}
	w.a.afterResultWrite = nil
	_, err = resultAs(w, Admin3Login, func(actor string) (financialERPResultView, error) {
		return w.a.decideResultLocked(SampleWorkspace, v.Basis.ID, v.Review.Digest, actor, "Withdraw unresolved result", true)
	})
	w.f.must(err)
	pending, unknown, err = resultEvidenceCounts(w.f.s)
	w.f.must(err)
	if len(pending) != 0 || unknown[SampleAdmin2] != 1 {
		t.Fatal("revoked unknown omitted")
	}
	key := [3]string{SampleWorkspace, p.Connection, p.External}
	saved := w.a.resultExternals[key]
	w.a.resultExternals[key] = "foreign"
	if _, _, err = resultEvidenceCounts(w.f.s); err == nil {
		t.Fatal("corrupt reverse ownership accepted")
	}
	w.a.resultExternals[key] = saved
	app := w.a.resultApplications[[2]string{SampleWorkspace, v.Basis.ID}]
	w.a.resultApplications[[2]string{SampleWorkspace, "orphan"}] = app
	if _, _, err = resultEvidenceCounts(w.f.s); err == nil {
		t.Fatal("orphan application accepted")
	}
	delete(w.a.resultApplications, [2]string{SampleWorkspace, "orphan"})
	w.f.s.fault = func(at string) bool { return at == "result-evidence-impact" }
	if _, _, err = resultEvidenceCounts(w.f.s); err == nil {
		t.Fatal("failed enumeration treated as empty")
	}
}

func TestFinancialERPResultImpactExcludesRecordedReceipts(t *testing.T) {
	w, p := resultWorld(t)
	w.f.s.documentERP = w.a
	v := confirmResult(t, w, proposeResult(t, w, p, "recorded-impact"))
	_, err := applyResult(w, v)
	w.f.must(err)
	pending, unknown, err := resultEvidenceCounts(w.f.s)
	w.f.must(err)
	if len(pending) != 0 || len(unknown) != 0 {
		t.Fatal("recorded receipt counted pending")
	}
}

func TestFinancialERPResultImpactPublicPreview(t *testing.T) {
	w, p, digest := publicResult(t)
	f := w.f
	pendingPacket := cloneDocumentValue(t, p)
	pendingPacket.External = "pending-impact-document"
	proposeResult(t, w, pendingPacket, "pending-impact")
	w.a.afterResultWrite = func(string) error { return errors.New("interrupted") }
	f.code(t, "unknown application", f.run(w.actors[Admin2Login], "erp_result.apply", "public-result", map[string]string{"digest": digest}), 500, "internal_error")
	w.a.afterResultWrite = nil
	admin := w.actors[AdminLogin]
	staged := f.stage(admin, "", SampleAdmin2)
	f.ok(t, "stage impact", staged, "draft", "")
	prepared := f.prepare(admin, field(staged, "draft"), "suspend_membership")
	f.ok(t, "prepare impact", prepared, "prepared", "")
	rows := f.bulkRead(admin, "/"+field(prepared, "bulk")).body["impact"].(map[string]any)["pending"].([]any)
	found := map[string]bool{}
	for _, raw := range rows {
		row := raw.(map[string]any)
		name := row["store"].(string)
		if name == "result_evidence_proposals" || name == "result_applications_unknown" {
			if row["count"] != 1.0 || row["rechecked"] != true {
				t.Fatal(row)
			}
			found[name] = true
		}
	}
	if len(found) != 2 {
		t.Fatal("result impact categories missing")
	}
	staged = f.stage(admin, "", SampleAdmin2)
	f.ok(t, "new draft for unavailable preview", staged, "draft", "")
	before := f.s.Revision()
	f.s.fault = func(at string) bool { return at == "result-evidence-impact" }
	f.code(t, "impact unavailable", f.prepare(admin, field(staged, "draft"), "suspend_membership"), 503, "dependency_unavailable")
	if f.s.Revision() != before {
		t.Fatal("failed impact mutated revision")
	}
}
