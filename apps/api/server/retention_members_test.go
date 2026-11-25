package server

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/retention"
)

func TestRetentionRunArtifactsAreReadInIdentifierOrder(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	ids := []string{}
	for n := 0; n < 120; n++ {
		id := fmt.Sprintf("plan-%03d", n)
		f.retRegister(id, retention.ClassConversation, 1, SampleWorkspace)
		ids = append(ids, id)
		if n%30 == 0 {
			hidden := fmt.Sprintf("plan-%03d-hidden", n)
			f.retRegister(hidden, retention.ClassConversation, 1, "other-workspace")
			ids = append(ids, hidden)
		}
	}
	run := f.retPlan(retention.ClassConversation, SampleWorkspace, ids...)
	if f.s.ret.run(run) != nil {
		t.Fatal("a run with another workspace's artifacts was stored for this workspace")
	}
	f.s.ret.runs = append(f.s.ret.runs, &retentionRun{id: run, workspace: SampleWorkspace})
	row := retRun(t, f.retentionRead(root), run)
	list := row["artifacts"].(map[string]any)
	main := retRows(list)
	if len(main) != retentionShown || whole(list["total"]) != 120 || list["more"] != true || list["next"] != "50" || main[0]["id"] != "plan-000" {
		t.Fatalf("the main read lists the first fifty in identifier order and says where it stopped: %d of %v, next %v, first %v", len(main), list["total"], list["next"], main[0]["id"])
	}
	path := "/api/v1/retention/run-artifacts?run=" + runTarget(run)
	got, next, pages := []string{}, "50", 0
	for _, r := range main {
		got = append(got, fmt.Sprint(r["id"]))
	}
	for next != "" {
		page := f.readAs(root, path+"&limit=30&after="+next)
		if page.status != 200 || whole(page.body["total"]) != 120 || page.body["order"] != "identifier" || page.body["run"] != runTarget(run) {
			t.Fatalf("after %s: %s", next, page.raw[:min(len(page.raw), 300)])
		}
		for _, raw := range page.body["artifacts"].([]any) {
			got = append(got, fmt.Sprint(raw.(map[string]any)["id"]))
		}
		next = fmt.Sprint(page.body["next"])
		if pages++; pages > 10 {
			t.Fatal("the artifacts do not end")
		}
	}
	want := []string{}
	for _, id := range ids {
		if !strings.HasSuffix(id, "-hidden") {
			want = append(want, id)
		}
	}
	if !reflect.DeepEqual(got, want) || pages != 3 {
		t.Fatalf("%d artifacts over %d pages, want %d in identifier order without the hidden ones", len(got), pages, len(want))
	}
	first := f.readAs(root, path)
	if rows, _ := first.body["artifacts"].([]any); len(rows) != retentionShown || !reflect.DeepEqual(rows[0], map[string]any(main[0])) || !reflect.DeepEqual(rows[49], map[string]any(main[49])) || first.body["next"] != "50" {
		t.Error("the first page of the further read differs from the main read's rows")
	}
	if strings.Contains(first.raw+f.readAs(root, path+"&after=100").raw, "hidden") {
		t.Error("another workspace's artifact is in the run's artifacts")
	}
	f.retRegister("their-own", retention.ClassConversation, 1, "other-workspace")
	other := f.retPlan(retention.ClassConversation, "other-workspace", "their-own")
	base := "/api/v1/retention/run-artifacts"
	for _, query := range []string{"", "?run=", "?run=retention-run-", "?run=retention-run-999", "?run=" + runTarget(other), "?run=run/1", "?run=retention-run-01", "?run=" + runTarget(run) + "&after=0",
		"?run=" + runTarget(run) + "&after=x", "?run=" + runTarget(run) + "&limit=0", "?run=" + runTarget(run) + "&limit=51", "?run=" + runTarget(run) + "&q=a", "?run=" + runTarget(run) + "&run=" + runTarget(run),
		"?after=5"} {
		if refused := f.readAs(root, base+query); refused.status != 422 {
			t.Errorf("%q: %d", query, refused.status)
		}
	}
	hiddenRun, unknownRun := f.readAs(root, base+"?run="+runTarget(other)), f.readAs(root, base+"?run=retention-run-999")
	delete(hiddenRun.body, "request_id")
	delete(unknownRun.body, "request_id")
	if !reflect.DeepEqual(hiddenRun.body, unknownRun.body) || strings.Contains(hiddenRun.raw, "their-own") {
		t.Errorf("another workspace's run answers differently from an unknown one: %s / %s", hiddenRun.raw, unknownRun.raw)
	}
	if past := f.readAs(root, path+"&after=500"); past.status != 200 || len(past.body["artifacts"].([]any)) != 0 || past.body["next"] != "" {
		t.Errorf("past the end: %s", past.raw[:min(len(past.raw), 200)])
	}
	f.code(t, "member", f.readAs(f.as(MemberLogin), path), 403, "permission_denied")
	f.code(t, "anonymous", f.readAs(actor{}, path), 401, "unauthenticated")
	f.s.fault = func(at string) bool { return at == "retention-read" }
	if failed := f.readAs(root, path); failed.status != 503 {
		t.Errorf("a fault: %d", failed.status)
	}
	f.s.fault = nil
	f.s.bootWorkspace = "elsewhere"
	if out := f.readAs(root, path); out.status != 422 {
		t.Errorf("another workspace: %d", out.status)
	}
}

func TestRetentionHoldOutcomesAreReadByArtifact(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	p, err := f.s.retention.Propose(retention.Change{Kind: retention.ChangePolicy, Policy: retention.Policy{Class: retention.ClassClaim, Period: 1}}, retAnn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.retention.Approve(p.Number, retBob, retention.Impact{}); err != nil {
		t.Fatal(err)
	}
	mine, theirs := []string{}, []string{}
	for n := 0; n < 120; n++ {
		mine = append(mine, fmt.Sprintf("cover-%03d", n))
		f.retRegister(mine[n], retention.ClassClaim, 10, SampleWorkspace)
		if n%30 == 0 {
			theirs = append(theirs, fmt.Sprintf("cover-%03d-hidden", n))
			f.retRegister(theirs[len(theirs)-1], retention.ClassClaim, 10, "other-workspace")
		}
	}
	f.retDelete(f.retPlan(retention.ClassClaim, SampleWorkspace, mine...), 0, mine...)
	f.retDelete(f.retPlan(retention.ClassClaim, "other-workspace", theirs...), 0, theirs...)
	if _, err := f.s.retentionPlaceHold(retention.Hold{ID: "wide-hold", Owner: SampleAdmin, Reason: "bulk", Tenants: []string{"other-workspace", SampleWorkspace}, Classes: []string{retention.ClassClaim}},
		retAnn, SampleWorkspace, "bulk"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.retentionPlaceHold(retention.Hold{ID: "their-hold", Owner: SampleAdmin, Reason: "bulk", Tenants: []string{"other-workspace"}, Classes: []string{retention.ClassClaim}},
		retAnn, "other-workspace", "bulk"); err != nil {
		t.Fatal(err)
	}
	hold := retHold(t, f.retentionRead(root), "wide-hold")
	main, _ := hold["outcomes"].([]any)
	if len(main) != retentionShown || hold["outcomes_more"] != true || hold["outcomes_next"] != main[len(main)-1].(map[string]any)["artifact"] {
		t.Fatalf("the main read lists the first fifty outcomes and says where it stopped: %d, next %v", len(main), hold["outcomes_next"])
	}
	path := "/api/v1/retention/hold-outcomes?hold=wide-hold"
	got, next, pages := []string{}, fmt.Sprint(hold["outcomes_next"]), 0
	for _, raw := range main {
		got = append(got, fmt.Sprint(raw.(map[string]any)["artifact"]))
	}
	for next != "" {
		page := f.readAs(root, path+"&limit=30&after="+next)
		if page.status != 200 || page.body["order"] != "identifier" || page.body["hold"] != "wide-hold" {
			t.Fatalf("after %s: %s", next, page.raw[:min(len(page.raw), 300)])
		}
		for _, raw := range page.body["outcomes"].([]any) {
			got = append(got, fmt.Sprint(raw.(map[string]any)["artifact"]))
		}
		next = fmt.Sprint(page.body["next"])
		if pages++; pages > 10 {
			t.Fatal("the outcomes do not end")
		}
	}
	covered := 0
	for index, id := range got {
		if strings.HasSuffix(id, "-hidden") || index > 0 && id <= got[index-1] {
			t.Fatalf("outcome %d is %s after %s", index, id, got[max(index-1, 0)])
		}
		if strings.HasPrefix(id, "cover-") {
			covered++
		}
	}
	if covered != 120 {
		t.Fatalf("%d of the 120 covered artifacts are listed over %d pages", covered, pages)
	}
	first := f.readAs(root, path)
	if rows, _ := first.body["outcomes"].([]any); len(rows) != retentionShown || !reflect.DeepEqual(rows[0], main[0]) || !reflect.DeepEqual(rows[49], main[49]) || first.body["next"] != hold["outcomes_next"] {
		t.Error("the first page of the further read differs from the main read's outcomes")
	}
	base := "/api/v1/retention/hold-outcomes"
	for _, query := range []string{"", "?hold=", "?hold=never-placed", "?hold=their-hold", "?hold=wide-hold&after=", "?hold=wide-hold&after=a,b", "?hold=wide-hold&limit=0", "?hold=wide-hold&limit=51",
		"?hold=wide-hold&q=a", "?hold=wide-hold&hold=wide-hold", "?after=cover-001"} {
		if refused := f.readAs(root, base+query); refused.status != 422 {
			t.Errorf("%q: %d", query, refused.status)
		}
	}
	hiddenHold, unknownHold := f.readAs(root, base+"?hold=their-hold"), f.readAs(root, base+"?hold=never-placed")
	delete(hiddenHold.body, "request_id")
	delete(unknownHold.body, "request_id")
	if !reflect.DeepEqual(hiddenHold.body, unknownHold.body) {
		t.Errorf("another workspace's hold answers differently from an unknown one: %s / %s", hiddenHold.raw, unknownHold.raw)
	}
	if past := f.readAs(root, path+"&after=zzzz"); past.status != 200 || len(past.body["outcomes"].([]any)) != 0 || past.body["next"] != "" {
		t.Errorf("past the end: %s", past.raw[:min(len(past.raw), 200)])
	}
	f.code(t, "member", f.readAs(f.as(MemberLogin), path), 403, "permission_denied")
	f.code(t, "anonymous", f.readAs(actor{}, path), 401, "unauthenticated")
	f.s.fault = func(at string) bool { return at == "retention-read" }
	if failed := f.readAs(root, path); failed.status != 503 {
		t.Errorf("a fault: %d", failed.status)
	}
	f.s.fault = nil
	f.s.bootWorkspace = "elsewhere"
	if out := f.readAs(root, path); out.status != 422 {
		t.Errorf("another workspace: %d", out.status)
	}
}
