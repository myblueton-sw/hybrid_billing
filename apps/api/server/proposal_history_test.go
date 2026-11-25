package server

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/installer"
	"github.com/myblueton-sw/hybrid_billing/modules/plugins"
	"github.com/myblueton-sw/hybrid_billing/modules/recovery"
)

func (f world) readAs(a actor, path string) reply {
	f.t.Helper()
	return f.do("GET", path, "", func(r *http.Request) {
		if a.cookie != nil {
			r.AddCookie(a.cookie)
		}
		r.Header.Set(csrfHeader, a.csrf)
	})
}

func keptNumbers(rows any) []int {
	out := []int{}
	list, _ := rows.([]any)
	for _, row := range list {
		out = append(out, whole(row.(map[string]any)["number"]))
	}
	return out
}

func (f world) walkHistory(a actor, path string, facts map[string]any, shown []int) []int {
	f.t.Helper()
	all := append([]int{}, shown...)
	next, pages := fmt.Sprint(facts["next"]), 0
	for next != "" {
		got := f.readAs(a, path+"?limit=60&after="+next)
		if got.status != 200 || whole(got.body["matching"]) != whole(facts["total"]) || got.body["order"] != "newest_first" {
			f.t.Fatalf("%s after %s: %s", path, next, got.raw)
		}
		all = append(all, keptNumbers(got.body["proposals"])...)
		next = fmt.Sprint(got.body["next"])
		if pages++; pages > 10 {
			f.t.Fatal("the history does not end")
		}
	}
	for index := 1; index < len(all); index++ {
		if all[index] >= all[index-1] {
			f.t.Fatalf("%s is not newest first without repeats at %d: %d then %d", path, index, all[index-1], all[index])
		}
	}
	return all
}

func (f world) refusedHistory(a actor, path string) {
	f.t.Helper()
	for _, query := range []string{"?after=0", "?after=x", "?after=01", "?after=", "?limit=0", "?limit=101", "?limit=5x", "?q=", "?q=%20", "?state=closed", "?limit=5&limit=6"} {
		if got := f.readAs(a, path+query); got.status != 422 {
			f.t.Errorf("%s%s: %d", path, query, got.status)
		}
	}
	if got := f.readAs(a, path+"?limit=100&after=1"); got.status != 200 || len(keptNumbers(got.body["proposals"])) != 0 || got.body["next"] != "" {
		f.t.Errorf("%s below the oldest proposal: %s", path, got.raw)
	}
}

func TestInstallationProposalHistoryIsCountedAndRead(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	seeded := len(f.s.install.entries)
	waiting := 0
	for _, e := range f.s.install.entries {
		if installPending(e) {
			waiting++
		}
	}
	for i := 0; i < installKept+5; i++ {
		e := &installEntry{view: installer.ProposalView{Number: uint64(1000 + i), State: installer.ProposalApplied, Proposer: SampleAdmin}}
		if i%80 == 10 {
			e.rationale = "Quarterly window"
		}
		f.s.install.keep(e)
	}
	got := f.readAs(root, "/api/v1/installation")
	facts, _ := got.body["proposal_history"].(map[string]any)
	shown := keptNumbers(got.body["proposals"])
	if got.status != 200 || waiting == 0 || whole(facts["total"]) != installKept-waiting || whole(facts["kept"]) != installKept || whole(facts["dropped"]) != seeded+5 || len(shown) != installShown ||
		fmt.Sprint(facts["next"]) != fmt.Sprint(shown[installShown-waiting-1]) || shown[0] != 1000+installKept+4 || shown[len(shown)-1] >= 1000 {
		t.Fatalf("the installation read: %v, shown %d", facts, len(shown))
	}
	all := f.walkHistory(root, "/api/v1/installation/proposals", facts, shown[:installShown-waiting])
	if len(all) != installKept-waiting || all[len(all)-1] < 1000 {
		t.Errorf("the pages hold %d closed of %d kept", len(all), len(f.s.install.entries))
	}
	found := f.readAs(root, "/api/v1/installation/proposals?q=QUARTERLY&limit=2")
	if found.status != 200 || whole(found.body["matching"]) != 3 || len(keptNumbers(found.body["proposals"])) != 2 || found.body["next"] == "" || found.body["q"] != "QUARTERLY" {
		t.Errorf("a search by rationale: %s", found.raw)
	}
	rest := f.readAs(root, "/api/v1/installation/proposals?q=quarterly&limit=2&after="+fmt.Sprint(found.body["next"]))
	if n := keptNumbers(rest.body["proposals"]); len(n) != 1 || rest.body["next"] != "" || whole(rest.body["matching"]) != 3 {
		t.Errorf("the rest of a search: %s", rest.raw)
	}
	one := f.readAs(root, "/api/v1/installation/proposals?q=installation-proposal-1010")
	if n := keptNumbers(one.body["proposals"]); len(n) != 1 || n[0] != 1010 {
		t.Errorf("a search by target: %s", one.raw)
	}
	f.refusedHistory(root, "/api/v1/installation/proposals")
	for i := 0; i < installShown+5; i++ {
		f.s.install.keep(&installEntry{view: installer.ProposalView{Number: uint64(3000 + i), State: installer.ProposalPending, Proposer: SampleAdmin}})
	}
	crowded := f.readAs(root, "/api/v1/installation")
	crowdedFacts := crowded.body["proposal_history"].(map[string]any)
	if n := keptNumbers(crowded.body["proposals"]); len(n) != waiting+installShown+5 || n[len(n)-1] >= 1000 || fmt.Sprint(crowdedFacts["next"]) != fmt.Sprint(3000+installShown+5) {
		t.Fatalf("every waiting proposal is listed and no closed one when they fill the list: %d listed, %v", len(n), crowdedFacts)
	}
	top := f.readAs(root, "/api/v1/installation/proposals?limit=3&after="+fmt.Sprint(crowdedFacts["next"]))
	if n := keptNumbers(top.body["proposals"]); len(n) != 3 || n[0] >= 3000 || whole(top.body["matching"]) != whole(crowdedFacts["total"]) {
		t.Errorf("the further read starts from the newest closed proposal and lists no waiting one: %v", n)
	}
	few := setup(t, SeedSample).ready()
	short := few.readAs(few.as(AdminLogin), "/api/v1/installation")
	if facts := short.body["proposal_history"].(map[string]any); facts["next"] != "" || whole(facts["total"]) != len(keptNumbers(short.body["proposals"]))-waiting || whole(facts["dropped"]) != 0 {
		t.Errorf("a short history has nothing further: %v", facts)
	}
}

func TestIntegrationProposalHistoryIsCountedAndRead(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	seeded := len(f.s.integ.entries)
	for i := 0; i < installKept+5; i++ {
		f.s.integ.keep(&pluginEntry{view: plugins.ProposalView{Number: uint64(1000 + i), State: plugins.ProposalApplied, Proposer: SampleAdmin}, rationale: fmt.Sprintf("Batch %03d", i)})
	}
	got := f.readAs(root, "/api/v1/integrations")
	modules, _ := got.body["modules"].(map[string]any)
	facts, _ := modules["proposal_history"].(map[string]any)
	shown := keptNumbers(modules["proposals"])
	_, closed, _ := listKept(f.s.integ.entries, pluginPending, pluginNumber, 0)
	waiting := len(f.s.integ.entries) - closed
	if got.status != 200 || whole(facts["total"]) != closed || whole(facts["kept"]) != installKept || whole(facts["dropped"]) != f.s.integ.dropped || f.s.integ.dropped == 0 ||
		seeded+installKept+5-f.s.integ.dropped != len(f.s.integ.entries) || len(shown) != installShown || fmt.Sprint(facts["next"]) != fmt.Sprint(shown[installShown-waiting-1]) {
		t.Fatalf("the integrations read: %v, shown %d, kept %d", facts, len(shown), len(f.s.integ.entries))
	}
	all := f.walkHistory(root, "/api/v1/integrations/proposals", facts, shown[:installShown-waiting])
	if len(all) != closed {
		t.Errorf("the pages hold %d of %d kept", len(all), len(f.s.integ.entries))
	}
	found := f.readAs(root, "/api/v1/integrations/proposals?q=batch%20204")
	if n := keptNumbers(found.body["proposals"]); found.status != 200 || len(n) != 1 || n[0] != 1204 || whole(found.body["matching"]) != 1 {
		t.Errorf("a search by rationale: %s", found.raw)
	}
	f.refusedHistory(root, "/api/v1/integrations/proposals")
	if denied := f.readAs(f.as(MemberLogin), "/api/v1/integrations/proposals"); denied.status != 403 || strings.Contains(denied.raw, "Batch") {
		t.Errorf("a member without module rights reads the history: %d %s", denied.status, denied.raw)
	}
	f.s.fault = func(at string) bool { return at == "integrations-read" }
	if got := f.readAs(root, "/api/v1/integrations/proposals"); got.status != 503 {
		t.Errorf("a fault %d", got.status)
	}
}

func TestRecoveryProposalHistoryIsCountedPerTenant(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	mine := f.s.rec.held(SampleWorkspace)
	add := func(tenant string, base, n int) {
		for i := 0; i < n; i++ {
			f.s.rec.keep(&recoveryEntry{view: recovery.ProposalView{Number: uint64(base + i), State: recovery.ProposalApplied, Proposer: SampleAdmin,
				Change: recovery.Change{Kind: recovery.ChangeInstallation, Installation: recovery.Installation{ID: fmt.Sprintf("site-%d", base+i), Tenant: tenant}}}})
		}
	}
	add("other-workspace", 5000, installKept+7)
	add(SampleWorkspace, 9000, installKept+5)
	if f.s.rec.dropped["other-workspace"] != 7 || f.s.rec.held("other-workspace") != installKept || f.s.rec.held(SampleWorkspace) != mine+installKept+5-f.s.rec.dropped[SampleWorkspace] ||
		f.s.rec.dropped[SampleWorkspace] == 0 {
		t.Fatalf("each tenant keeps its own history: dropped %v", f.s.rec.dropped)
	}
	got := f.readAs(root, "/api/v1/recovery")
	facts, _ := got.body["proposal_history"].(map[string]any)
	shown := keptNumbers(got.body["proposals"])
	_, closed, _ := listKept(f.s.rec.of(SampleWorkspace), recoveryPending, recoveryNumber, 0)
	waiting := f.s.rec.held(SampleWorkspace) - closed
	if got.status != 200 || waiting == 0 || whole(facts["total"]) != closed || whole(facts["dropped"]) != f.s.rec.dropped[SampleWorkspace] || whole(facts["kept"]) != installKept ||
		len(shown) != recoveryShown || fmt.Sprint(facts["next"]) != fmt.Sprint(shown[recoveryShown-waiting-1]) || shown[len(shown)-1] >= 9000 {
		t.Fatalf("the recovery read: %v, shown %d", facts, len(shown))
	}
	all := f.walkHistory(root, "/api/v1/recovery/proposals", facts, shown[:recoveryShown-waiting])
	if len(all) != closed {
		t.Errorf("the pages hold %d of %d kept", len(all), f.s.rec.held(SampleWorkspace))
	}
	for _, n := range all {
		if n >= 5000 && n < 9000 {
			t.Fatalf("another tenant's proposal %d is listed", n)
		}
	}
	found := f.readAs(root, "/api/v1/recovery/proposals?q=site-9204")
	if n := keptNumbers(found.body["proposals"]); len(n) != 1 || n[0] != 9204 || whole(found.body["matching"]) != 1 {
		t.Errorf("a search by installation: %s", found.raw)
	}
	if other := f.readAs(root, "/api/v1/recovery/proposals?q=site-5100"); whole(other.body["matching"]) != 0 || len(keptNumbers(other.body["proposals"])) != 0 {
		t.Errorf("another tenant's proposal is found: %s", other.raw)
	}
	f.refusedHistory(root, "/api/v1/recovery/proposals")
}

func listedShape(rows []any) (int, int, int, int) {
	waiting, oldestClosed, first := 0, 0, 0
	for index, raw := range rows {
		row := raw.(map[string]any)
		if index == 0 {
			first = whole(row["number"])
		}
		if row["state"] == "pending" {
			waiting++
		} else {
			oldestClosed = whole(row["number"])
		}
	}
	return len(rows), first, waiting, oldestClosed
}
