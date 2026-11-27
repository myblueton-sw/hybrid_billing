package server

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/retention"
)

func TestRetentionRunsListEveryRunningRunAndPageTheFinishedOnes(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	seeded, err := (&retentionFacts{s: f.s, workspace: SampleWorkspace}).workspaceRuns()
	if err != nil {
		t.Fatal(err)
	}
	running, finished := 0, 0
	for _, r := range seeded {
		if runRunning(r) {
			running++
		} else {
			finished++
		}
	}
	made := []string{}
	for n := 0; n < 130; n++ {
		id := fmt.Sprintf("run-art-%03d", n)
		f.retRegister(id, retention.ClassConversation, 1, SampleWorkspace)
		run := f.retPlan(retention.ClassConversation, SampleWorkspace, id)
		made = append(made, run)
		if n < 70 {
			if _, err := f.s.retention.Stop(run, retBob); err != nil {
				t.Fatal(err)
			}
			finished++
		} else {
			running++
		}
	}
	section := retSection(f.retentionRead(root), "deletion")
	facts, _ := section["finished"].(map[string]any)
	rows := retRows(section)
	listed := 0
	for _, row := range rows {
		if row["state"] == retention.RunRunning {
			listed++
		}
	}
	if listed != running || len(rows) != running || whole(section["total"]) != running+finished || section["more"] != true || whole(facts["total"]) != finished || facts["next"] == "" {
		t.Fatalf("every running run is listed and the finished ones are counted: %d of %d running in %d rows, facts %v", listed, running, len(rows), facts)
	}
	path := "/api/v1/retention/runs"
	seen, next, pages := []string{}, "", 0
	for {
		query := "?limit=30"
		if next != "" {
			query += "&after=" + next
		}
		page := f.readAs(root, path+query)
		if page.status != 200 || whole(page.body["total"]) != finished || whole(page.body["matching"]) != finished || page.body["order"] != "newest_first" {
			t.Fatalf("%s: %s", query, page.raw[:min(len(page.raw), 300)])
		}
		for _, raw := range page.body["runs"].([]any) {
			row := raw.(map[string]any)
			if row["state"] == retention.RunRunning || row["can_stop"] != false {
				t.Fatalf("a running run among the finished ones: %v", row["run"])
			}
			seen = append(seen, fmt.Sprint(row["number"]))
		}
		next = fmt.Sprint(page.body["next"])
		if pages++; next == "" || pages > 10 {
			break
		}
	}
	if len(seen) != finished || pages != 3 {
		t.Fatalf("%d finished runs over %d pages, want %d", len(seen), pages, finished)
	}
	for index := 1; index < len(seen); index++ {
		a, _ := strconv.Atoi(seen[index-1])
		b, _ := strconv.Atoi(seen[index])
		if b >= a {
			t.Fatalf("the finished runs are not newest first: %s then %s", seen[index-1], seen[index])
		}
	}
	found := f.readAs(root, path+"?q="+strings.ToUpper(made[3]))
	if rows, _ := found.body["runs"].([]any); found.status != 200 || len(rows) == 0 || rows[len(rows)-1].(map[string]any)["run"].(map[string]any)["label"] != made[3] {
		t.Errorf("a search by label: %s", found.raw[:min(len(found.raw), 300)])
	}
	if none := f.readAs(root, path+"?q="+made[100]); whole(none.body["matching"]) != 0 || whole(none.body["total"]) != finished {
		t.Errorf("a running run answers a search of the finished ones: %s", none.raw[:min(len(none.raw), 200)])
	}
	for _, query := range []string{"?after=0", "?after=x", "?limit=0", "?limit=101", "?q=", "?state=x", "?limit=5&limit=6"} {
		if refused := f.readAs(root, path+query); refused.status != 422 {
			t.Errorf("%s: %d", query, refused.status)
		}
	}
	f.code(t, "member", f.readAs(f.as(MemberLogin), path), 403, "permission_denied")
	f.code(t, "anonymous", f.readAs(actor{}, path), 401, "unauthenticated")
	f.s.fault = func(at string) bool { return at == "retention-read" }
	if failed := f.readAs(root, path); failed.status != 503 {
		t.Errorf("a fault: %d", failed.status)
	}
	f.s.fault = nil
	f.s.ret.runs = append(f.s.ret.runs, &retentionRun{id: "run/other", workspace: "other-workspace"})
	if same := f.readAs(root, path); whole(same.body["total"]) != finished {
		t.Errorf("another workspace's run is counted: %v", same.body["total"])
	}
	f.s.ret.runs = append(f.s.ret.runs, &retentionRun{id: "run/never", workspace: SampleWorkspace})
	if failed := f.readAs(root, path); failed.status != 503 {
		t.Errorf("an unknown stored run: %d", failed.status)
	}
	f.s.ret.runs = f.s.ret.runs[:len(f.s.ret.runs)-1]
	f.s.bootWorkspace = "elsewhere"
	if other := f.readAs(root, path); other.status != 200 || len(other.body["runs"].([]any)) != 0 || whole(other.body["total"]) != 0 {
		t.Errorf("another workspace: %d", other.status)
	}
}

func TestRetentionRunsMixTheNewestFinishedOnesWhenFewAreRunning(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	for n := 0; n < 60; n++ {
		id := fmt.Sprintf("done-art-%03d", n)
		f.retRegister(id, retention.ClassConversation, 1, SampleWorkspace)
		if _, err := f.s.retention.Stop(f.retPlan(retention.ClassConversation, SampleWorkspace, id), retBob); err != nil {
			t.Fatal(err)
		}
	}
	section := retSection(f.retentionRead(root), "deletion")
	facts, _ := section["finished"].(map[string]any)
	rows := retRows(section)
	oldest, listedFinished := 0, 0
	for _, row := range rows {
		if row["state"] != retention.RunRunning {
			oldest, listedFinished = whole(row["number"]), listedFinished+1
		}
	}
	if len(rows) != retentionShown || section["more"] != true || facts["next"] != fmt.Sprint(oldest) || listedFinished == 0 || listedFinished == len(rows) {
		t.Fatalf("fifty rows, running and finished, with the cursor at the oldest finished one listed: %d rows, %d finished, facts %v, oldest %d", len(rows), listedFinished, facts, oldest)
	}
	for index := 1; index < len(rows); index++ {
		if whole(rows[index]["number"]) >= whole(rows[index-1]["number"]) {
			t.Fatalf("the main read is not newest first at %d", index)
		}
	}
	rest := f.readAs(root, "/api/v1/retention/runs?after="+fmt.Sprint(facts["next"]))
	older, _ := rest.body["runs"].([]any)
	if rest.status != 200 || len(older) == 0 || rest.body["next"] != "" || whole(older[0].(map[string]any)["number"]) >= oldest {
		t.Fatalf("the rest follows the main read: %s", rest.raw[:min(len(rest.raw), 300)])
	}
	if listedFinished+len(older) != whole(facts["total"]) || len(rows)+len(older) != whole(section["total"]) {
		t.Errorf("%d listed and %d older of %v", len(rows), len(older), section["total"])
	}
}

func TestRetentionTombstonesOfTheWindowArePagedNewestFirst(t *testing.T) {
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
	before := whole(retSection(f.retentionRead(root), "decisions")["tombstones"].(map[string]any)["total"])
	mine, theirs := []string{}, []string{}
	for n := 0; n < 120; n++ {
		id := fmt.Sprintf("gone-%03d", n)
		mine = append(mine, id)
		f.retRegister(id, retention.ClassClaim, 10, SampleWorkspace)
	}
	for n := 0; n < 30; n++ {
		id := fmt.Sprintf("elsewhere-%03d", n)
		theirs = append(theirs, id)
		f.retRegister(id, retention.ClassClaim, 10, "other-workspace")
	}
	first := f.retPlan(retention.ClassClaim, SampleWorkspace, mine[:60]...)
	f.retDelete(first, 0, mine[:60]...)
	other := f.retPlan(retention.ClassClaim, "other-workspace", theirs...)
	f.retDelete(other, 0, theirs...)
	second := f.retPlan(retention.ClassClaim, SampleWorkspace, mine[60:]...)
	f.retDelete(second, 0, mine[60:]...)
	total := before + 120
	decisions := retSection(f.retentionRead(root), "decisions")
	section := decisions["tombstones"].(map[string]any)
	rows := retRows(section)
	if len(rows) != retentionShown || whole(section["total"]) != total || section["more"] != true || section["next"] != f.s.tombstoneCursor(fmt.Sprint(rows[len(rows)-1]["artifact"])) {
		t.Fatalf("the main read lists the newest fifty and says where it stopped: %d rows of %v, next %v", len(rows), section["total"], section["next"])
	}
	path := "/api/v1/retention/tombstones"
	all, next, pages := append([]map[string]any{}, rows...), fmt.Sprint(section["next"]), 0
	for next != "" {
		page := f.readAs(root, path+"?limit=30&after="+next)
		if page.status != 200 || whole(page.body["total"]) != total || whole(page.body["matching"]) != total || page.body["order"] != "newest_first" || whole(page.body["window"]) != retentionWindow ||
			page.body["watermark"] != nil || decisions["watermark"] != nil {
			t.Fatalf("after %s: %s", next, page.raw[:min(len(page.raw), 300)])
		}
		for _, raw := range page.body["tombstones"].([]any) {
			all = append(all, raw.(map[string]any))
		}
		next = fmt.Sprint(page.body["next"])
		if pages++; pages > 10 {
			t.Fatal("the records do not end")
		}
	}
	if len(all) != total {
		t.Fatalf("the pages hold %d of %d", len(all), total)
	}
	kept, _ := f.s.retention.Tombstones(0, int(f.s.retention.Watermark()))
	expected := []string{}
	for index := len(kept) - 1; index >= 0; index-- {
		if f.s.ret.visibleArtifact(kept[index].Artifact, SampleWorkspace) {
			expected = append(expected, kept[index].Artifact)
		}
	}
	listed := []string{}
	for _, row := range all {
		listed = append(listed, fmt.Sprint(row["artifact"]))
	}
	if strings.Join(listed, " ") != strings.Join(expected, " ") {
		t.Fatal("the records are not the workspace's, newest first, each once")
	}
	for index, row := range all {
		if strings.HasPrefix(fmt.Sprint(row["artifact"]), "elsewhere-") || row["other_scope"] != false {
			t.Fatalf("another workspace's record is listed: %v", row)
		}
		if row["sequence"] != nil {
			t.Fatalf("a sequence is sent at %d", index)
		}
	}
	whole50 := f.readAs(root, path)
	if got, _ := whole50.body["tombstones"].([]any); len(got) != retentionShown || fmt.Sprint(got[0]) != fmt.Sprint(map[string]any(rows[0])) || fmt.Sprint(got[49]) != fmt.Sprint(map[string]any(rows[49])) {
		t.Error("the first page of the further read differs from the rows of the main read")
	}
	found := f.readAs(root, path+"?q=GONE-07")
	if got, _ := found.body["tombstones"].([]any); found.status != 200 || len(got) != 10 || whole(found.body["matching"]) != 10 || got[0].(map[string]any)["artifact"] != "gone-079" {
		t.Errorf("a search by artifact: %s", found.raw[:min(len(found.raw), 300)])
	}
	if byRun := f.readAs(root, path+"?q="+first+"&limit=100"); whole(byRun.body["matching"]) != 60 {
		t.Errorf("a search by run label: %v", byRun.body["matching"])
	}
	if hidden := f.readAs(root, path+"?q=elsewhere"); whole(hidden.body["matching"]) != 0 || strings.Contains(hidden.raw, "elsewhere-0") {
		t.Errorf("a search reaches another workspace's records: %s", hidden.raw[:min(len(hidden.raw), 200)])
	}
	if hidden := f.readAs(root, path+"?q="+other); whole(hidden.body["matching"]) != 0 {
		t.Errorf("a search by another workspace's run label matches %v", hidden.body["matching"])
	}
	for _, query := range []string{"?after=0", "?after=x", "?limit=0", "?limit=101", "?q=", "?state=x", "?limit=5&limit=6"} {
		if refused := f.readAs(root, path+query); refused.status != 422 {
			t.Errorf("%s: %d", query, refused.status)
		}
	}
	f.code(t, "member", f.readAs(f.as(MemberLogin), path), 403, "permission_denied")
	f.code(t, "anonymous", f.readAs(actor{}, path), 401, "unauthenticated")
	f.s.fault = func(at string) bool { return at == "retention-read" }
	if failed := f.readAs(root, path); failed.status != 503 {
		t.Errorf("a fault: %d", failed.status)
	}
	f.s.fault = nil
	f.s.bootWorkspace = "elsewhere"
	if out := f.readAs(root, path); out.status != 200 || len(out.body["tombstones"].([]any)) != 0 || whole(out.body["total"]) != 0 || out.body["watermark"] != nil {
		t.Errorf("another workspace: %d %s", out.status, out.raw)
	}
}
