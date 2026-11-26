package server

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func (f world) closedRead(a actor, query string) reply {
	f.t.Helper()
	return f.do("GET", "/api/v1/organizations/plans/closed"+query, "", func(r *http.Request) {
		if a.cookie != nil {
			r.AddCookie(a.cookie)
		}
		r.Header.Set(csrfHeader, a.csrf)
	})
}

func planIDs(got reply) []string {
	out := []string{}
	rows, _ := got.body["plans"].([]any)
	for _, row := range rows {
		out = append(out, row.(map[string]any)["id"].(string))
	}
	return out
}

func (f world) closeMany(ws, prefix string, n int) {
	for i := 0; i < n; i++ {
		f.s.plans.put(&planMeta{workspace: ws, id: fmt.Sprintf("%s-%03d", prefix, i), proposer: SampleAdmin, change: planChange{Member: fmt.Sprintf("acct-%03d", i), From: "2026-12-01"}})
	}
}

func TestClosedPlansArePagedAndCounted(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	open := planID(t, f.proposeChange(root, "hierarchy", "sample-acct-01", "set", "sample-unit-north-data", "2026-12-01"))
	f.closeMany(SampleWorkspace, "plan-closed", closedPlansKept+5)
	f.closeMany("other-workspace", "plan-other", closedPlansKept+7)
	f.s.prunePlans()
	if f.s.plans.dropped[SampleWorkspace] != 5 || f.s.plans.dropped["other-workspace"] != 7 || len(f.s.plans.byKey) != 2*closedPlansKept+1 {
		t.Fatalf("each workspace keeps its own history: dropped %v, kept %d", f.s.plans.dropped, len(f.s.plans.byKey))
	}
	first := f.plansRead(root)
	facts := first.body["closed"].(map[string]any)
	shown := planIDs(first)
	if whole(facts["total"]) != closedPlansKept || whole(facts["kept"]) != closedPlansKept || whole(facts["dropped"]) != 5 || facts["next"] == "" || len(shown) != plansShown || shown[0] != "plan-closed-204" ||
		!strings.Contains(strings.Join(shown, " "), open) {
		t.Fatalf("the plans read %v, shown %d first %s", facts, len(shown), shown[0])
	}
	all := []string{}
	for _, id := range shown {
		if id != open {
			all = append(all, id)
		}
	}
	next, pages := facts["next"].(string), 0
	for next != "" {
		got := f.closedRead(root, "?limit=40&after="+next)
		if got.status != 200 || got.body["order"] != closedPlansOrder || whole(got.body["matching"]) != closedPlansKept || whole(got.body["closed"].(map[string]any)["total"]) != closedPlansKept {
			t.Fatalf("page %d %s", pages, got.raw)
		}
		if pages == 1 {
			f.closeMany(SampleWorkspace, "plan-later", 3)
			f.s.prunePlans()
		}
		all = append(all, planIDs(got)...)
		next, _ = got.body["next"].(string)
		if pages++; pages > 20 {
			t.Fatal("the pages do not end")
		}
	}
	seen := map[string]bool{}
	for i, id := range all {
		if seen[id] || strings.HasPrefix(id, "plan-other") || strings.HasPrefix(id, "plan-later") || (i > 0 && id >= all[i-1]) {
			t.Fatalf("row %d %s after %v", i, id, all[max(0, i-2):i])
		}
		seen[id] = true
	}
	if len(all) != closedPlansKept-3 || all[0] != "plan-closed-204" || all[len(all)-1] != "plan-closed-008" {
		t.Errorf("read %d closed plans from %s to %s in %d pages", len(all), all[0], all[len(all)-1], pages)
	}
	top := f.closedRead(root, "")
	if ids := planIDs(top); len(ids) != plansShown || ids[0] != "plan-later-002" || top.body["next"] == "" || whole(top.body["closed"].(map[string]any)["dropped"]) != 8 || top.body["q"] != "" || top.body["server_time"] == "" {
		t.Errorf("the newest page %v %s", ids[:1], top.raw[:200])
	}
	gone := f.closedRead(root, "?after=1")
	if gone.status != 200 || len(planIDs(gone)) != 0 || gone.body["next"] != "" || whole(gone.body["closed"].(map[string]any)["dropped"]) != 8 {
		t.Errorf("a cursor below what is kept %s", gone.raw)
	}
	if got := f.closedRead(f.as(Admin2Login), "?limit=1"); got.status != 200 || len(planIDs(got)) != 1 {
		t.Errorf("another administrator of the workspace %s", got.raw)
	}
	if got := f.closedRead(f.as(MemberLogin), ""); got.status != 403 {
		t.Errorf("a member %d", got.status)
	}
	if got := f.closedRead(actor{}, ""); got.status != 401 {
		t.Errorf("no session %d", got.status)
	}
}

func TestClosedPlansSearchAndRefusals(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.closeMany(SampleWorkspace, "plan-closed", 60)
	f.closeMany("other-workspace", "plan-closed", 60)
	f.s.plans.put(&planMeta{workspace: SampleWorkspace, id: "plan-special", proposer: SampleAdmin2, change: planChange{Member: "sample-acct-03", Target: "sample-unit-north-ops", From: "2026-12-01"}})
	for query, want := range map[string]string{
		"?q=closed-01":          "10 plan-closed-019",
		"?q=CLOSED-059":         "1 plan-closed-059",
		"?q=acct-007":           "1 plan-closed-007",
		"?q=Account%2003":       "1 plan-special",
		"?q=north%20operations": "1 plan-special",
		"?q=admin2%40example":   "1 plan-special",
		"?q=nothing-like-this":  "0 ",
		"?q=plan-other":         "0 ",
	} {
		got := f.closedRead(root, query)
		ids := planIDs(got)
		first := ""
		if len(ids) > 0 {
			first = ids[0]
		}
		if got.status != 200 || fmt.Sprint(whole(got.body["matching"]), " ", first) != want || whole(got.body["closed"].(map[string]any)["total"]) != 61 {
			t.Errorf("%s: %d matching %v first %q", query, got.status, got.body["matching"], first)
		}
	}
	paged := []string{}
	for next := ""; ; {
		query := "?q=closed-0&limit=7"
		if next != "" {
			query += "&after=" + next
		}
		got := f.closedRead(root, query)
		if got.status != 200 || whole(got.body["matching"]) != 60 || got.body["q"] != "closed-0" {
			t.Fatalf("a searched page %s", got.raw[:200])
		}
		paged = append(paged, planIDs(got)...)
		if next, _ = got.body["next"].(string); next == "" {
			break
		}
	}
	if len(paged) != 60 || paged[0] != "plan-closed-059" || paged[59] != "plan-closed-000" {
		t.Errorf("searched pages gave %d rows", len(paged))
	}
	for _, query := range []string{"?after=0", "?after=x", "?after=01", "?after=", "?limit=0", "?limit=101", "?limit=5x", "?q=", "?q=%20", "?q=" + strings.Repeat("a", 65), "?state=closed", "?limit=5&limit=6"} {
		if got := f.closedRead(root, query); got.status != 422 {
			t.Errorf("%s: %d", query, got.status)
		}
	}
	f.s.fault = func(at string) bool { return at == "plan-active" }
	if got := f.closedRead(root, ""); got.status != 503 {
		t.Errorf("a fault %d", got.status)
	}
}

func TestPlansReadSaysWhenNoClosedPlanIsShown(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	empty := f.plansRead(root).body["closed"].(map[string]any)
	if whole(empty["total"]) != 0 || empty["next"] != "" || whole(empty["dropped"]) != 0 {
		t.Errorf("no closed plan %v", empty)
	}
	f.closeMany(SampleWorkspace, "plan-closed", 3)
	few := f.plansRead(root)
	if facts := few.body["closed"].(map[string]any); whole(facts["total"]) != 3 || facts["next"] != "" || len(planIDs(few)) != 3 {
		t.Errorf("three closed plans, all shown %v", facts)
	}
}

func TestAPlanIsOrderedAndKeptByWhenItClosed(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	old := planID(t, f.proposeChange(root, "hierarchy", "sample-acct-01", "set", "sample-unit-north-data", "2026-12-01"))
	f.closeMany(SampleWorkspace, "plan-closed", closedPlansKept+10)
	if got := f.run(root, "relationship.discard", old, nil); got.status != 200 {
		t.Fatalf("discard %s", got.raw)
	}
	if f.s.plans.get(SampleWorkspace, old) == nil {
		t.Fatal("a plan that was just closed was dropped as the oldest")
	}
	top := f.closedRead(root, "?limit=2")
	if ids := planIDs(top); len(ids) != 2 || ids[0] != old || ids[1] != fmt.Sprintf("plan-closed-%03d", closedPlansKept+9) {
		t.Errorf("the plan closed last is listed first: %v", planIDs(top))
	}
	first := f.plansRead(root)
	if ids := planIDs(first); ids[0] != old || whole(first.body["closed"].(map[string]any)["dropped"]) != 11 {
		t.Errorf("the plans read %v %v", planIDs(first)[:2], first.body["closed"])
	}
}
