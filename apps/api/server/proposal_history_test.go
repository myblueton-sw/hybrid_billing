package server

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/installer"
	"github.com/myblueton-sw/hybrid_billing/modules/perfgate"
	"github.com/myblueton-sw/hybrid_billing/modules/plugins"
	"github.com/myblueton-sw/hybrid_billing/modules/recovery"
	"github.com/myblueton-sw/hybrid_billing/modules/releasetrace"
	"github.com/myblueton-sw/hybrid_billing/modules/retention"
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

func lockNumbers(rows any) []int { return keptNumbers(rows) }

func TestRecoveryLockLogIsCutCountedAndRead(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	install := f.s.rec.install(recoveryStandby)
	seeded := len(install.locks)
	row := func(got reply) map[string]any {
		t.Helper()
		rows, _ := got.body["isolation"].(map[string]any)["rows"].([]any)
		for _, raw := range rows {
			if r := raw.(map[string]any); r["installation"] == recoveryStandby {
				return r
			}
		}
		t.Fatalf("no isolation row for the standby installation: %s", got.raw[:min(len(got.raw), 300)])
		return nil
	}
	few := row(f.recoveryRead(root))
	if facts := few["lock_history"].(map[string]any); whole(facts["total"]) != seeded || facts["next"] != "" || len(lockNumbers(few["locks"])) != seeded {
		t.Fatalf("a short log is whole and says nothing further exists: %v", facts)
	}
	at := install.locks
	for n := 0; n < 130; n++ {
		at = append(at, recoveryLockRecord{by: SampleAdmin, rationale: fmt.Sprintf("Drill %03d", n), at: f.s.clock()})
	}
	install.locks = at
	total := seeded + 130
	main := row(f.recoveryRead(root))
	facts := main["lock_history"].(map[string]any)
	shown := lockNumbers(main["locks"])
	if whole(facts["total"]) != total || len(shown) != recoveryShown || shown[0] != total-recoveryShown+1 || shown[len(shown)-1] != total || fmt.Sprint(facts["next"]) != fmt.Sprint(shown[0]) {
		t.Fatalf("the main read lists the newest fifty in the order recorded and says where older ones start: %v first %d last %d", facts, shown[0], shown[len(shown)-1])
	}
	path := "/api/v1/recovery/locks?installation=" + recoveryStandby
	all, next, pages := []int{}, fmt.Sprint(facts["next"]), 0
	for next != "" {
		got := f.readAs(root, path+"&limit=40&after="+next)
		if got.status != 200 || whole(got.body["total"]) != len(install.locks) || whole(got.body["matching"]) != len(install.locks) || got.body["order"] != "newest_first" || got.body["installation"] != recoveryStandby {
			t.Fatalf("after %s: %s", next, got.raw[:min(len(got.raw), 300)])
		}
		all = append(all, lockNumbers(got.body["locks"])...)
		next = fmt.Sprint(got.body["next"])
		if pages++; pages == 1 {
			install.locks = append(install.locks, recoveryLockRecord{by: SampleAdmin, rationale: "added meanwhile", at: f.s.clock()})
		}
	}
	if len(all) != total-recoveryShown || all[0] != total-recoveryShown || all[len(all)-1] != 1 {
		t.Fatalf("the older locks in pages: %d from %d to %d", len(all), all[0], all[len(all)-1])
	}
	for index := 1; index < len(all); index++ {
		if all[index] != all[index-1]-1 {
			t.Fatalf("a gap or a repeat at %d", index)
		}
	}
	found := f.readAs(root, path+"&q=drill%20129")
	if n := lockNumbers(found.body["locks"]); found.status != 200 || len(n) != 1 || n[0] != seeded+130 || whole(found.body["matching"]) != 1 {
		t.Errorf("a search by reason: %s", found.raw[:min(len(found.raw), 300)])
	}
	if by := f.readAs(root, path+"&q=admin%40example&limit=1"); by.status != 200 || whole(by.body["matching"]) < 131 || by.body["next"] == "" {
		t.Errorf("a search by sign-in name: %d %v", by.status, by.body["matching"])
	}
	for _, query := range []string{"", "?installation=", "?installation=sample-absent", "?installation=" + recoveryStandby + "&installation=" + recoveryStandby, "?installation=" + recoveryStandby + "&after=0",
		"?installation=" + recoveryStandby + "&limit=101", "?installation=" + recoveryStandby + "&q=", "?installation=" + recoveryStandby + "&state=x"} {
		if got := f.readAs(root, "/api/v1/recovery/locks"+query); got.status != 422 {
			t.Errorf("%s: %d", query, got.status)
		}
	}
	if got := f.readAs(actor{}, path); got.status != 401 {
		t.Errorf("anonymous: %d", got.status)
	}
	member, theirs := f.readAs(f.as(MemberLogin), path), f.recoveryRead(f.as(MemberLogin))
	if member.status != theirs.status || (member.status == 200 && theirs.body["state"] != "configured") {
		t.Errorf("a member reads the lock log (%d) but not the recovery read (%d %v)", member.status, theirs.status, theirs.body["state"])
	}
}

func TestSubscriptionActionHistoryIsCutCountedAndRead(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	e := f.s.integ.subscription(sampleSubscription)
	seeded := len(e.history)
	row := func(got reply) map[string]any {
		t.Helper()
		rows, _ := got.body["channels"].(map[string]any)["subscriptions"].([]any)
		for _, raw := range rows {
			if r := raw.(map[string]any); r["id"] == sampleSubscription {
				return r
			}
		}
		t.Fatalf("no row for the sample subscription: %s", got.raw[:min(len(got.raw), 300)])
		return nil
	}
	few := row(f.integrations(root))
	if facts := few["history_facts"].(map[string]any); seeded == 0 || whole(facts["total"]) != seeded || facts["next"] != "" || len(keptNumbers(few["history"])) != seeded {
		t.Fatalf("a short history is whole and says nothing further exists: %v", facts)
	}
	for n := 0; n < 130; n++ {
		e.history = append(e.history, actionRecord{action: "pause", actor: SampleAdmin, rationale: fmt.Sprintf("Window %03d", n)})
	}
	total := seeded + 130
	main := row(f.integrations(root))
	facts := main["history_facts"].(map[string]any)
	shown := keptNumbers(main["history"])
	if whole(facts["total"]) != total || len(shown) != installShown || shown[0] != total-installShown+1 || shown[len(shown)-1] != total || fmt.Sprint(facts["next"]) != fmt.Sprint(shown[0]) {
		t.Fatalf("the main read lists the newest fifty in the order recorded: %v first %d last %d", facts, shown[0], shown[len(shown)-1])
	}
	path := "/api/v1/integrations/subscription-history?subscription=" + sampleSubscription
	all, next, pages := []int{}, fmt.Sprint(facts["next"]), 0
	for next != "" {
		got := f.readAs(root, path+"&limit=40&after="+next)
		if got.status != 200 || whole(got.body["total"]) != len(e.history) || whole(got.body["matching"]) != len(e.history) || got.body["order"] != "newest_first" || got.body["subscription"] != sampleSubscription {
			t.Fatalf("after %s: %s", next, got.raw[:min(len(got.raw), 300)])
		}
		all = append(all, keptNumbers(got.body["history"])...)
		next = fmt.Sprint(got.body["next"])
		if pages++; pages == 1 {
			e.history = append(e.history, actionRecord{action: "request_resume", actor: SampleAdmin, rationale: "added meanwhile"})
		}
	}
	if len(all) != total-installShown || all[0] != total-installShown || all[len(all)-1] != 1 {
		t.Fatalf("the older actions in pages: %d from %d to %d", len(all), all[0], all[len(all)-1])
	}
	for index := 1; index < len(all); index++ {
		if all[index] != all[index-1]-1 {
			t.Fatalf("a gap or a repeat at %d", index)
		}
	}
	found := f.readAs(root, path+"&q=window%20129")
	if n := keptNumbers(found.body["history"]); found.status != 200 || len(n) != 1 || n[0] != seeded+130 || whole(found.body["matching"]) != 1 {
		t.Errorf("a search by reason: %s", found.raw[:min(len(found.raw), 300)])
	}
	if by := f.readAs(root, path+"&q=request_resume"); by.status != 200 || whole(by.body["matching"]) != 1 {
		t.Errorf("a search by action: %d %v", by.status, by.body["matching"])
	}
	for _, query := range []string{"", "?subscription=", "?subscription=sample-absent", "?subscription=" + sampleSubscription + "&subscription=" + sampleSubscription, "?subscription=" + sampleSubscription + "&after=0",
		"?subscription=" + sampleSubscription + "&limit=101", "?subscription=" + sampleSubscription + "&q=", "?subscription=" + sampleSubscription + "&state=x"} {
		if got := f.readAs(root, "/api/v1/integrations/subscription-history"+query); got.status != 422 {
			t.Errorf("%s: %d", query, got.status)
		}
	}
	if got := f.readAs(actor{}, path); got.status != 401 {
		t.Errorf("anonymous: %d", got.status)
	}
	if denied := f.readAs(f.as(MemberLogin), path); denied.status != 403 || strings.Contains(denied.raw, "Window") {
		t.Errorf("a member without channel rights reads the history: %d", denied.status)
	}
	f.s.fault = func(at string) bool { return at == "integrations-read" }
	if got := f.readAs(root, path); got.status != 503 {
		t.Errorf("a fault: %d", got.status)
	}
	f.s.fault = func(at string) bool { return at == "webhooks-subscription" }
	if got := f.readAs(root, path); got.status != 503 {
		t.Errorf("a failing module read: %d", got.status)
	}
}

func TestSubscriptionActionHistoryFollowsTheChannelsSection(t *testing.T) {
	path := "/api/v1/integrations/subscription-history?subscription=" + sampleSubscription
	for _, right := range gate.IntegrationRights() {
		f := setup(t, SeedSample).ready()
		f.setRights(t, SampleAdmin3, right)
		reader := f.as(Admin3Login)
		shown := false
		for _, r := range channelRights {
			shown = shown || r == right
		}
		main, got := f.integrations(reader), f.readAs(reader, path)
		_, listed := main.body["channels"].(map[string]any)["subscriptions"]
		if main.status != 200 || listed != shown || (got.status == 200) != shown || (!shown && got.status != 403) {
			t.Errorf("only %s: the channels section listed %v, the history read answered %d", right, listed, got.status)
		}
	}
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.s.integ.subscriptions = append(f.s.integ.subscriptions, &subscriptionEntry{id: "no-longer-held", history: []actionRecord{{action: "revoke", actor: SampleAdmin, rationale: "Kept by the server only"}}})
	gone := f.readAs(root, "/api/v1/integrations/subscription-history?subscription=no-longer-held")
	absent := f.readAs(root, "/api/v1/integrations/subscription-history?subscription=never-existed")
	if gone.status != 422 || absent.status != 422 || strings.Contains(gone.raw, "Kept by the server") || gone.body["code"] != absent.body["code"] {
		t.Errorf("a subscription the module no longer holds: %d %s", gone.status, gone.raw)
	}
	f.s.bootWorkspace = "elsewhere"
	if other := f.readAs(root, path); other.status != 422 || strings.Contains(other.raw, "Sample data") {
		t.Errorf("another workspace: %d %s", other.status, other.raw)
	}
}

func TestRetentionProposalHistoryIsCountedAndReadByKind(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	seeded := len(f.s.ret.entries)
	closedOf := func(kind string) int {
		n := 0
		for _, e := range f.s.ret.entries {
			if !retentionPending(e) && f.s.ret.visibleEntry(e, SampleWorkspace) && (kind == "" || e.view.Change.Kind == kind) {
				n++
			}
		}
		return n
	}
	for i := 0; i < installKept+5; i++ {
		f.s.ret.keep(&retentionEntry{view: retention.ProposalView{Number: uint64(1000 + i), State: retention.ProposalWithdrawn, Proposer: SampleAdmin,
			Change: retention.Change{Kind: retention.ChangePolicy, Policy: retention.Policy{Class: retention.ClassClaim, Period: 1}}}, rationale: fmt.Sprintf("Review %03d", i)})
	}
	for i := 0; i < 3; i++ {
		f.s.ret.keep(&retentionEntry{workspace: SampleWorkspace, view: retention.ProposalView{Number: uint64(2000 + i), State: retention.ProposalWithdrawn, Proposer: SampleAdmin,
			Change: retention.Change{Kind: retention.ChangeRelease, Release: retention.Release{Hold: retentionDispute}}}})
		f.s.ret.keep(&retentionEntry{workspace: "other-workspace", view: retention.ProposalView{Number: uint64(2100 + i), State: retention.ProposalWithdrawn, Proposer: SampleAdmin,
			Change: retention.Change{Kind: retention.ChangeRelease, Release: retention.Release{Hold: retentionDispute}}}, rationale: "Another workspace"})
	}
	section := retSection(f.retentionRead(root), "proposals")
	facts, _ := section["history"].(map[string]any)
	byKind, _ := section["closed_by_kind"].(map[string]any)
	rows := retRows(section)
	waiting, shown := 0, []int{}
	for _, row := range rows {
		if row["state"] == retention.ProposalPending {
			waiting++
		} else {
			shown = append(shown, whole(row["number"]))
		}
	}
	if waiting == 0 || len(rows) != retentionShown || whole(facts["total"]) != closedOf("") || whole(facts["kept"]) != installKept || whole(facts["dropped"]) != f.s.ret.dropped ||
		f.s.ret.dropped != seeded+installKept+5+6-len(f.s.ret.entries) || f.s.ret.dropped == 0 || fmt.Sprint(facts["next"]) != fmt.Sprint(shown[len(shown)-1]) ||
		whole(byKind[retention.ChangePolicy]) != closedOf(retention.ChangePolicy) || whole(section["total"]) != len(f.s.ret.entries)-3 || closedOf(retention.ChangeRelease) < 3 || whole(byKind[retention.ChangeRelease]) != closedOf(retention.ChangeRelease) {
		t.Fatalf("the retention read: %v by kind %v, %d waiting, %d closed shown", facts, byKind, waiting, len(shown))
	}
	all := f.walkHistory(root, "/api/v1/retention/proposals", facts, shown)
	if len(all) != closedOf("") {
		t.Errorf("the pages hold %d of %d closed", len(all), closedOf(""))
	}
	policy := f.readAs(root, "/api/v1/retention/proposals?kind=policy&limit=100&after="+fmt.Sprint(facts["next"]))
	policyFacts, _ := policy.body["proposal_history"].(map[string]any)
	if closedOf(retention.ChangePolicy) == closedOf("") || whole(policyFacts["total"]) != closedOf(retention.ChangePolicy) || whole(policyFacts["dropped"]) != f.s.ret.dropped {
		t.Errorf("with a kind the total is the closed proposals of that kind: %v, policy %d of %d closed", policyFacts, closedOf(retention.ChangePolicy), closedOf(""))
	}
	if policy.status != 200 || policy.body["kind"] != "policy" || whole(policy.body["matching"]) != closedOf(retention.ChangePolicy) {
		t.Errorf("by kind: %d matching %v of %d", policy.status, policy.body["matching"], closedOf(retention.ChangePolicy))
	}
	for _, raw := range policy.body["proposals"].([]any) {
		if row := raw.(map[string]any); row["kind"] != "policy" || row["state"] == retention.ProposalPending {
			t.Fatalf("a row of another kind or a waiting one: %v %v", row["kind"], row["state"])
		}
	}
	release := f.readAs(root, "/api/v1/retention/proposals?kind=release")
	if release.status != 200 || whole(release.body["matching"]) != closedOf(retention.ChangeRelease) {
		t.Errorf("releases: %d %v of %d", release.status, release.body["matching"], closedOf(retention.ChangeRelease))
	}
	if searched := f.readAs(root, "/api/v1/retention/proposals?kind=policy&q=review%20204"); whole(searched.body["matching"]) != 1 ||
		whole(searched.body["proposal_history"].(map[string]any)["total"]) != closedOf(retention.ChangePolicy) {
		t.Errorf("a search does not change the total of the kind: %s", searched.raw[:min(len(searched.raw), 200)])
	}
	if hidden := f.readAs(root, "/api/v1/retention/proposals?q=another"); whole(hidden.body["matching"]) != 0 || strings.Contains(hidden.raw, "Another workspace") {
		t.Errorf("another workspace's release proposal is found: %s", hidden.raw[:min(len(hidden.raw), 200)])
	}
	for _, n := range all {
		if n >= 2100 && n < 2200 {
			t.Fatalf("another workspace's release proposal %d is listed", n)
		}
	}
	found := f.readAs(root, "/api/v1/retention/proposals?q=review%20204")
	if n := keptNumbers(found.body["proposals"]); found.status != 200 || len(n) != 1 || n[0] != 1204 {
		t.Errorf("a search by rationale: %s", found.raw[:min(len(found.raw), 300)])
	}
	f.refusedHistory(root, "/api/v1/retention/proposals")
	for _, query := range []string{"?kind=", "?kind=other", "?kind=policy&kind=plan"} {
		if got := f.readAs(root, "/api/v1/retention/proposals"+query); got.status != 422 {
			t.Errorf("%s: %d", query, got.status)
		}
	}
	member, theirs := f.readAs(f.as(MemberLogin), "/api/v1/retention/proposals"), f.retentionRead(f.as(MemberLogin))
	if member.status != theirs.status {
		t.Errorf("a member: the further read %d, the main read %d", member.status, theirs.status)
	}
	if got := f.readAs(actor{}, "/api/v1/retention/proposals"); got.status != 401 {
		t.Errorf("anonymous: %d", got.status)
	}
	f.s.fault = func(at string) bool { return at == "retention-read" }
	if got := f.readAs(root, "/api/v1/retention/proposals"); got.status != 503 {
		t.Errorf("a fault: %d", got.status)
	}
	f.s.fault = nil
	f.s.bootWorkspace = "elsewhere"
	if other := f.readAs(root, "/api/v1/retention/proposals"); other.status != 200 || len(keptNumbers(other.body["proposals"])) != 0 || whole(other.body["matching"]) != 0 {
		t.Errorf("another workspace: %d %s", other.status, other.raw[:min(len(other.raw), 200)])
	}
}

func TestEvidenceProposalHistoryIsCountedAndReadAcrossModules(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	seeded := len(f.s.ev.entries)
	for i := 0; i < installKept+5; i++ {
		module, kind := moduleTrace, releasetrace.ChangeBuild
		e := &evidenceEntry{module: module, number: uint64(1000 + i/2), kind: kind, state: releasetrace.ProposalWithdrawn, proposer: SampleAdmin, rationale: fmt.Sprintf("Round %03d", i),
			trace: releasetrace.Change{Kind: releasetrace.ChangeBuild, Build: releasetrace.Build{Build: "9.9.9"}}}
		if i%2 == 1 {
			e.module, e.kind, e.trace, e.perf = modulePerf, perfgate.ChangeProfile, releasetrace.Change{}, perfgate.Change{Kind: perfgate.ChangeProfile}
		}
		f.s.ev.keep(e)
	}
	closedNow := func() int {
		n := 0
		for _, e := range f.s.ev.entries {
			if !evidencePending(e) {
				n++
			}
		}
		return n
	}
	decisions := evMap(f.evidenceRead(root).body, "decisions")
	facts := evMap(decisions, "proposal_history")
	rows := evRows(evMap(decisions, "proposals"))
	waiting, shown := 0, []string{}
	for _, row := range rows {
		if row["state"] == entryPending {
			waiting++
		} else {
			shown = append(shown, fmt.Sprint(row["target"]))
		}
	}
	if waiting == 0 || len(rows) != evidenceShown || whole(facts["total"]) != closedNow() || whole(facts["kept"]) != installKept || f.s.ev.dropped != seeded+installKept+5-len(f.s.ev.entries) ||
		whole(facts["dropped"]) != f.s.ev.dropped || f.s.ev.dropped == 0 || facts["next"] == "" || whole(evMap(decisions, "proposals")["total"]) != len(f.s.ev.entries) {
		t.Fatalf("the evidence read: %v, %d waiting, %d closed shown", facts, waiting, len(shown))
	}
	all, next, pages := append([]string{}, shown...), fmt.Sprint(facts["next"]), 0
	for next != "" {
		got := f.readAs(root, "/api/v1/release-evidence/proposals?limit=60&after="+next)
		if got.status != 200 || whole(got.body["matching"]) != closedNow() || whole(evMap(got.body, "proposal_history")["total"]) != closedNow() || got.body["order"] != "newest_first" {
			t.Fatalf("after %s: %s", next, got.raw[:min(len(got.raw), 300)])
		}
		for _, raw := range got.body["proposals"].([]any) {
			row := raw.(map[string]any)
			if row["state"] == entryPending {
				t.Fatalf("a waiting proposal on a further page: %v", row["target"])
			}
			all = append(all, fmt.Sprint(row["target"]))
		}
		next = fmt.Sprint(got.body["next"])
		if pages++; pages > 10 {
			t.Fatal("the history does not end")
		}
	}
	seen := map[string]bool{}
	for _, target := range all {
		if seen[target] {
			t.Fatalf("%s is listed twice", target)
		}
		seen[target] = true
	}
	if len(all) != closedNow() || !seen[evidenceProposalTarget(moduleTrace, 1050)] || !seen[evidenceProposalTarget(modulePerf, 1050)] {
		t.Errorf("the pages hold %d of %d closed, and two modules' proposals with the same number are both kept apart", len(all), closedNow())
	}
	found := f.readAs(root, "/api/v1/release-evidence/proposals?q=round%20203")
	if rows, _ := found.body["proposals"].([]any); found.status != 200 || len(rows) != 1 || rows[0].(map[string]any)["module"] != modulePerf || whole(found.body["matching"]) != 1 {
		t.Errorf("a search by rationale: %s", found.raw[:min(len(found.raw), 300)])
	}
	f.refusedHistory(root, "/api/v1/release-evidence/proposals")
	member, theirs := f.readAs(f.as(MemberLogin), "/api/v1/release-evidence/proposals"), f.evidenceRead(f.as(MemberLogin))
	if member.status != theirs.status || member.status != 403 {
		t.Errorf("a member: the further read %d, the main read %d", member.status, theirs.status)
	}
	if got := f.readAs(actor{}, "/api/v1/release-evidence/proposals"); got.status != 401 {
		t.Errorf("anonymous: %d", got.status)
	}
	f.s.fault = func(at string) bool { return at == "release-evidence-read" }
	if got := f.readAs(root, "/api/v1/release-evidence/proposals"); got.status != 503 {
		t.Errorf("a fault: %d", got.status)
	}
	f.s.fault = nil
	f.s.bootWorkspace = "elsewhere"
	if other := f.readAs(root, "/api/v1/release-evidence/proposals"); other.status != 200 || len(other.body["proposals"].([]any)) != 0 || whole(other.body["matching"]) != 0 {
		t.Errorf("another workspace: %d", other.status)
	}
}
