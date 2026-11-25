package server

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/installer"
	"github.com/myblueton-sw/hybrid_billing/modules/perfgate"
	"github.com/myblueton-sw/hybrid_billing/modules/plugins"
	"github.com/myblueton-sw/hybrid_billing/modules/recovery"
	"github.com/myblueton-sw/hybrid_billing/modules/releasetrace"
	"github.com/myblueton-sw/hybrid_billing/modules/retention"
	"github.com/myblueton-sw/hybrid_billing/modules/webhooks"
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

func TestRetentionHoldsListEveryLiveHoldAndPageTheReleasedOnes(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	for _, h := range f.s.ret.holds {
		v, err := f.s.retentionHold(h.id)
		if err != nil || v.Released != h.released {
			t.Fatalf("after the seed the server's note of %s (released %v) differs from the module's (%v, %v)", h.id, h.released, v.Released, err)
		}
	}
	seeded := len(f.s.ret.holds)
	place := func(id, workspace string, released bool) {
		t.Helper()
		if _, err := f.s.retentionPlaceHold(retention.Hold{ID: id, Owner: SampleAdmin, Reason: "bulk", Tenants: []string{workspace}, Classes: []string{retention.ClassClaim}}, retAnn, workspace, "Bulk "+id); err != nil {
			t.Fatal(err)
		}
		f.s.ret.hold(id).released = released
	}
	for i := 0; i < 130; i++ {
		place(fmt.Sprintf("old-%03d", i), SampleWorkspace, true)
	}
	for i := 0; i < 60; i++ {
		place(fmt.Sprintf("live-%03d", i), SampleWorkspace, false)
	}
	for i := 0; i < 5; i++ {
		place(fmt.Sprintf("other-%03d", i), "other-workspace", true)
	}
	live, released := 0, 0
	for _, h := range f.s.ret.holds {
		if h.workspace == SampleWorkspace && h.released {
			released++
		} else if h.workspace == SampleWorkspace {
			live++
		}
	}
	section := retSection(f.retentionRead(root), "holds")
	facts, _ := section["released"].(map[string]any)
	rows := retRows(section)
	listedLive := 0
	for _, row := range rows {
		if row["released"] == false {
			listedLive++
		}
		if strings.HasPrefix(fmt.Sprint(row["id"]), "other-") {
			t.Fatalf("another workspace's hold %v is listed", row["id"])
		}
	}
	if live < 60 || listedLive != live || len(rows) != live || whole(section["total"]) != live+released || section["more"] != true || whole(facts["total"]) != released ||
		fmt.Sprint(facts["next"]) != fmt.Sprint(seeded+130+60+1) {
		t.Fatalf("every live hold is listed, and no released one when the live ones fill the list: %d rows, %d live of %d, released %v", len(rows), listedLive, live, facts)
	}
	path := "/api/v1/retention/holds"
	all, next, pages := []int{}, fmt.Sprint(facts["next"]), 0
	for next != "" {
		got := f.readAs(root, path+"?limit=50&after="+next)
		if got.status != 200 || whole(got.body["total"]) != released || whole(got.body["matching"]) != released || got.body["order"] != "newest_first" {
			t.Fatalf("after %s: %s", next, got.raw[:min(len(got.raw), 300)])
		}
		for _, raw := range got.body["holds"].([]any) {
			row := raw.(map[string]any)
			if id := fmt.Sprint(row["id"]); strings.HasPrefix(id, "live-") || strings.HasPrefix(id, "other-") {
				t.Fatalf("a live hold or another workspace's hold on a further page: %v", id)
			}
			all = append(all, whole(row["number"]))
		}
		next = fmt.Sprint(got.body["next"])
		if pages++; pages > 10 {
			t.Fatal("the released holds do not end")
		}
	}
	for index := 1; index < len(all); index++ {
		if all[index] >= all[index-1] {
			t.Fatalf("not newest first without repeats at %d", index)
		}
	}
	if len(all) != released {
		t.Errorf("the pages hold %d of %d released", len(all), released)
	}
	found := f.readAs(root, path+"?q=old-129")
	if rows, _ := found.body["holds"].([]any); found.status != 200 || len(rows) != 1 || rows[0].(map[string]any)["id"] != "old-129" || whole(found.body["matching"]) != 1 || whole(found.body["total"]) != released {
		t.Errorf("a search by identifier: %s", found.raw[:min(len(found.raw), 300)])
	}
	if hidden := f.readAs(root, path+"?q=other-00"); whole(hidden.body["matching"]) != 0 {
		t.Errorf("another workspace's hold is found: %v", hidden.body["matching"])
	}
	f.refusedHistory(root, path)
	member, theirs := f.readAs(f.as(MemberLogin), path), f.retentionRead(f.as(MemberLogin))
	if member.status != theirs.status {
		t.Errorf("a member: the further read %d, the main read %d", member.status, theirs.status)
	}
	if got := f.readAs(actor{}, path); got.status != 401 {
		t.Errorf("anonymous: %d", got.status)
	}
	f.s.fault = func(at string) bool { return at == "retention-read" }
	if got := f.readAs(root, path); got.status != 503 {
		t.Errorf("a fault: %d", got.status)
	}
	f.s.fault = nil
	f.s.bootWorkspace = "elsewhere"
	if other := f.readAs(root, path); other.status != 200 || len(other.body["holds"].([]any)) != 0 || whole(other.body["total"]) != 0 {
		t.Errorf("another workspace: %d", other.status)
	}
}

func TestRetentionHoldsMixTheNewestReleasedOnesWhenFewAreInForce(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	live := len(f.s.ret.holds)
	for i := 0; i < 60; i++ {
		id := fmt.Sprintf("done-%03d", i)
		if _, err := f.s.retentionPlaceHold(retention.Hold{ID: id, Owner: SampleAdmin, Reason: "bulk", Tenants: []string{SampleWorkspace}, Classes: []string{retention.ClassClaim}}, retAnn, SampleWorkspace,
			fmt.Sprintf("Closed case %03d", i)); err != nil {
			t.Fatal(err)
		}
		f.s.ret.hold(id).released = true
	}
	section := retSection(f.retentionRead(root), "holds")
	facts, _ := section["released"].(map[string]any)
	rows := retRows(section)
	done, first, last := 0, "", ""
	for _, row := range rows {
		if id := fmt.Sprint(row["id"]); strings.HasPrefix(id, "done-") {
			if done++; first == "" {
				first = id
			}
			last = id
		}
	}
	if len(rows) != retentionShown || done != retentionShown-live || first != "done-059" || last != fmt.Sprintf("done-%03d", 60-done) || whole(facts["total"]) != 60 ||
		fmt.Sprint(facts["next"]) != fmt.Sprint(f.s.ret.hold(last).number) || whole(section["total"]) != live+60 {
		t.Fatalf("with %d in force: %d rows, %d released from %s to %s, facts %v", live, len(rows), done, first, last, facts)
	}
	rest := f.readAs(root, "/api/v1/retention/holds?limit=100&after="+fmt.Sprint(facts["next"]))
	if got, _ := rest.body["holds"].([]any); rest.status != 200 || len(got) != 60-done || got[0].(map[string]any)["id"] != fmt.Sprintf("done-%03d", 60-done-1) || rest.body["next"] != "" {
		t.Errorf("the rest of the released holds: %d", rest.status)
	}
	found := f.readAs(root, "/api/v1/retention/holds?q=closed%20case%20007")
	if got, _ := found.body["holds"].([]any); len(got) != 1 || got[0].(map[string]any)["id"] != "done-007" {
		t.Errorf("a search by the reason given when placing: %s", found.raw[:min(len(found.raw), 200)])
	}
}

func TestSubscriptionEntriesOfPurgedSubscriptionsAreDropped(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	by := webhooks.Actor{ID: SampleAdmin, Kind: webhooks.KindHuman}
	seeded := len(f.s.integ.subscriptions)
	limit := WebhookLimits().Subscriptions
	held := func() int {
		n := 0
		for _, e := range f.s.integ.subscriptions {
			if _, err := f.s.subscriptionOf(e.id); err == nil {
				n++
			}
		}
		return n
	}
	for i := 0; i < 5*limit; i++ {
		id := fmt.Sprintf("round-%03d", i)
		if _, _, err := f.s.subscriptionPropose(webhooks.Subscription{ID: id, Subscriber: sampleSubscriber, Destination: "sample-erp-webhook", Types: []string{typeInvoice}}, by, "round"); err != nil {
			t.Fatalf("propose %s: %v", id, err)
		}
		if _, err := f.s.webhooks.WithdrawSubscription(SampleWorkspace, id, by); err != nil {
			t.Fatalf("withdraw %s: %v", id, err)
		}
		if n := len(f.s.integ.subscriptions); n > 2*limit+1 || n != held() {
			t.Fatalf("after %d rounds the server keeps %d entries of which the module holds %d", i+1, n, held())
		}
	}
	if n := len(f.s.integ.subscriptions); n >= seeded+5*limit || n > 2*limit+1 || f.s.integ.subscription(sampleSubscription) == nil || f.s.integ.subscription("round-000") != nil {
		t.Errorf("the entries are bounded by what the module holds: %d kept, the seeded one kept %v, the first round kept %v", n, f.s.integ.subscription(sampleSubscription) != nil,
			f.s.integ.subscription("round-000") != nil)
	}
	gone := subscriptionTarget + "round-000"
	if e := f.s.subscriptionEntryOf(gone); e == nil || e.proposer != SampleAdmin || len(e.history) != 0 || len(f.s.integ.purged) == 0 || len(f.s.integ.purged) != 5*limit+seeded-len(f.s.integ.subscriptions) {
		t.Fatalf("a purged subscription keeps its identifier and proposer for commands: %v, %d noted", e, len(f.s.integ.purged))
	}
	f.unchanged(t, "revoke a purged subscription", f.run(root, "integrations.subscription_revoke", gone, nil), "invalid_stage")
	f.unchanged(t, "withdraw a purged subscription", f.run(root, "integrations.subscription_withdraw", gone, nil), "invalid_stage")
	f.code(t, "a subscription that never existed", f.run(root, "integrations.subscription_revoke", subscriptionTarget+"never-proposed", nil), 403, "permission_denied")
	got := f.integrations(root)
	rows, _ := got.body["channels"].(map[string]any)["subscriptions"].([]any)
	if got.status != 200 || len(rows) != held() {
		t.Errorf("the read lists %d subscriptions, the module holds %d", len(rows), held())
	}
	f.s.fault = func(at string) bool { return at == "webhooks-subscription" }
	before := len(f.s.integ.subscriptions)
	f.s.dropPurgedSubscriptions(nil)
	if len(f.s.integ.subscriptions) != before {
		t.Errorf("a failing module read dropped entries: %d of %d kept", len(f.s.integ.subscriptions), before)
	}
}

func TestEvidenceResultsAreReadNewestFirstWithoutReadingAll(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	seeded := len(f.s.ev.results)
	seededCalls := 0
	f.s.fault = func(step string) bool {
		if step == "trace-results" {
			seededCalls++
		}
		return false
	}
	before := evRows(evMap(f.evidenceRead(root).body, "artifacts", "results"))
	f.s.fault = nil
	for index, k := range f.s.ev.results {
		if f.s.ev.resultSeq[k] == 0 || (index > 0 && f.s.ev.resultSeq[k] <= f.s.ev.resultSeq[f.s.ev.results[index-1]]) {
			t.Fatalf("the seeded results are not in the order of their latest record at %d", index)
		}
	}
	if len(before) != seeded || fmt.Sprint(before[0]["number"]) != fmt.Sprint(f.s.ev.resultSeq[f.s.ev.results[seeded-1]]) {
		t.Fatalf("the seeded results are listed newest first: %d rows", len(before))
	}
	owned := map[string]bool{}
	record := func(test string) {
		t.Helper()
		if !owned[test] {
			owned[test] = true
			f.evAddRequirement("owner-of-"+test, false, false, releasetrace.TestRef{ID: test})
		}
		x := releasetrace.Result{Namespace: evidenceNamespace, Test: test, Build: evidenceBuild, Passed: true, Evidence: sampleHash("bulk:" + test + fmt.Sprint(f.s.ev.resultCount))}
		if _, err := f.s.trace.Record(x, releasetrace.Actor{ID: evidenceTraceRunner, Kind: releasetrace.KindSystem}); err != nil {
			t.Fatalf("record %s: %v", test, err)
		}
		f.s.ev.addResult(evidenceResultKey{namespace: evidenceNamespace, test: test, build: evidenceBuild})
	}
	for i := 0; i < 130; i++ {
		record(fmt.Sprintf("bulk-%03d", i))
	}
	total := seeded + 130
	calls := map[string]int{}
	f.s.fault = func(step string) bool { calls[step]++; return false }
	got := f.evidenceRead(root)
	f.s.fault = nil
	section := evMap(got.body, "artifacts")
	rows := evRows(evMap(section, "results"))
	if got.status != 200 || len(rows) != evidenceShown || rows[0]["test"] != "bulk-129" || rows[evidenceShown-1]["test"] != "bulk-080" || whole(evMap(section, "results")["total"]) != total ||
		evMap(section, "results")["more"] != true || fmt.Sprint(section["results_next"]) != fmt.Sprint(rows[evidenceShown-1]["number"]) {
		t.Fatalf("the main read lists the newest fifty: first %v last %v next %v", rows[0]["test"], rows[evidenceShown-1]["test"], section["results_next"])
	}
	if calls["trace-results"] >= total || calls["trace-results"] > evidenceShown+seededCalls {
		t.Errorf("the main read asked the module for %d results with %d stored; with the seed alone it asked for %d, and it should ask for the fifty shown and no more for the %d added", calls["trace-results"], total,
			seededCalls, 130)
	}
	path := "/api/v1/release-evidence/results"
	all, next, pages := []string{}, fmt.Sprint(section["results_next"]), 0
	for next != "" {
		page := f.readAs(root, path+"?limit=60&after="+next)
		if page.status != 200 || whole(page.body["total"]) != len(f.s.ev.results) || whole(page.body["matching"]) != len(f.s.ev.results) || page.body["order"] != "latest_record_first" || page.body["rows_can_move"] != true {
			t.Fatalf("after %s: %s", next, page.raw[:min(len(page.raw), 300)])
		}
		for _, raw := range page.body["results"].([]any) {
			row := raw.(map[string]any)
			all = append(all, fmt.Sprint(row["test"], "@", row["build"]))
		}
		next = fmt.Sprint(page.body["next"])
		if pages++; pages == 1 {
			record("bulk-005")
			record("added-meanwhile")
		}
		if pages > 10 {
			t.Fatal("the results do not end")
		}
	}
	seen := map[string]bool{}
	for _, id := range all {
		if seen[id] {
			t.Fatalf("%s is listed twice", id)
		}
		seen[id] = true
	}
	if len(all) != total-evidenceShown-1 || seen["added-meanwhile@"+evidenceBuild] || seen["bulk-005@"+evidenceBuild] {
		t.Errorf("the older pages hold %d results; a result recorded again between two pages moved above the cursor and is not on them", len(all))
	}
	top := evRows(evMap(f.evidenceRead(root).body, "artifacts", "results"))
	if top[0]["test"] != "added-meanwhile" || top[1]["test"] != "bulk-005" || whole(top[1]["passes"]) != 2 {
		t.Errorf("a result recorded again is listed at the top with its counts: %v %v", top[0]["test"], top[1]["test"])
	}
	found := f.readAs(root, path+"?q=BULK-077")
	if rows, _ := found.body["results"].([]any); found.status != 200 || len(rows) != 1 || rows[0].(map[string]any)["test"] != "bulk-077" || whole(found.body["matching"]) != 1 {
		t.Errorf("a search by test: %s", found.raw[:min(len(found.raw), 300)])
	}
	f.refusedHistory(root, path)
	member, theirs := f.readAs(f.as(MemberLogin), path), f.evidenceRead(f.as(MemberLogin))
	if member.status != theirs.status || member.status != 403 {
		t.Errorf("a member: the further read %d, the main read %d", member.status, theirs.status)
	}
	if anonymous := f.readAs(actor{}, path); anonymous.status != 401 {
		t.Errorf("anonymous: %d", anonymous.status)
	}
	f.s.fault = func(at string) bool { return at == "release-evidence-read" }
	if failed := f.readAs(root, path); failed.status != 503 {
		t.Errorf("a fault: %d", failed.status)
	}
	f.s.fault = nil
	f.s.bootWorkspace = "elsewhere"
	if other := f.readAs(root, path); other.status != 200 || len(other.body["results"].([]any)) != 0 || whole(other.body["total"]) != 0 {
		t.Errorf("another workspace: %d", other.status)
	}
}

func TestEvidenceBuildsArePagedAndEveryHarnessIsListed(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	seeded := len(f.s.ev.builds)
	short := evMap(f.evidenceRead(root).body, "artifacts")
	if short["builds_next"] != "" || len(evRows(evMap(short, "builds"))) != seeded {
		t.Fatalf("a short list of builds is whole and says nothing further exists: %v", short["builds_next"])
	}
	for i := 0; i < 130; i++ {
		f.s.ev.builds = append(f.s.ev.builds, evidenceBuildRecord{id: fmt.Sprintf("9.%d.0", i), commit: sampleHash(fmt.Sprint("bulk-build", i))[:40]})
	}
	for i := 0; i < 70; i++ {
		f.s.ev.harnesses = append(f.s.ev.harnesses, evidenceHarnessRecord{module: moduleTrace, id: fmt.Sprintf("bulk-harness-%03d", i)})
	}
	total := seeded + 130
	section := evMap(f.evidenceRead(root).body, "artifacts")
	rows := evRows(evMap(section, "builds"))
	if len(rows) != evidenceShown || rows[0]["build"] != "9.129.0" || whole(rows[0]["order"]) != total || fmt.Sprint(section["builds_next"]) != fmt.Sprint(rows[evidenceShown-1]["order"]) ||
		whole(evMap(section, "builds")["total"]) != total || evMap(section, "builds")["more"] != true {
		t.Fatalf("the main read lists the newest fifty builds: first %v, next %v", rows[0]["build"], section["builds_next"])
	}
	harnesses := evMap(section, "harnesses")
	if len(evRows(harnesses)) != len(f.s.ev.harnesses) || harnesses["more"] != false || whole(harnesses["total"]) != len(f.s.ev.harnesses) || len(f.s.ev.harnesses) < 70 {
		t.Errorf("every harness is listed: %d of %d", len(evRows(harnesses)), len(f.s.ev.harnesses))
	}
	path := "/api/v1/release-evidence/builds"
	orders, next, pages := []int{}, fmt.Sprint(section["builds_next"]), 0
	for next != "" {
		page := f.readAs(root, path+"?limit=40&after="+next)
		if page.status != 200 || whole(page.body["total"]) != len(f.s.ev.builds) || whole(page.body["matching"]) != len(f.s.ev.builds) || page.body["order"] != "newest_first" {
			t.Fatalf("after %s: %s", next, page.raw[:min(len(page.raw), 300)])
		}
		for _, raw := range page.body["builds"].([]any) {
			orders = append(orders, whole(raw.(map[string]any)["order"]))
		}
		next = fmt.Sprint(page.body["next"])
		if pages++; pages == 1 {
			f.s.ev.builds = append(f.s.ev.builds, evidenceBuildRecord{id: "9.999.0", commit: sampleHash("added")[:40]})
		}
		if pages > 10 {
			t.Fatal("the builds do not end")
		}
	}
	if len(orders) != total-evidenceShown || orders[0] != total-evidenceShown || orders[len(orders)-1] != 1 {
		t.Fatalf("the older builds in pages: %d from %d to %d", len(orders), orders[0], orders[len(orders)-1])
	}
	for index := 1; index < len(orders); index++ {
		if orders[index] != orders[index-1]-1 {
			t.Fatalf("a gap or a repeat at %d", index)
		}
	}
	found := f.readAs(root, path+"?q=9.77.0")
	if rows, _ := found.body["builds"].([]any); found.status != 200 || len(rows) != 1 || rows[0].(map[string]any)["build"] != "9.77.0" {
		t.Errorf("a search by build: %s", found.raw[:min(len(found.raw), 300)])
	}
	commit := sampleHash("bulk-build5")[:12]
	if byCommit := f.readAs(root, path+"?q="+commit); whole(byCommit.body["matching"]) != 1 {
		t.Errorf("a search by commit: %v", byCommit.body["matching"])
	}
	f.refusedHistory(root, path)
	member, theirs := f.readAs(f.as(MemberLogin), path), f.evidenceRead(f.as(MemberLogin))
	if member.status != theirs.status || member.status != 403 {
		t.Errorf("a member: the further read %d, the main read %d", member.status, theirs.status)
	}
	if anonymous := f.readAs(actor{}, path); anonymous.status != 401 {
		t.Errorf("anonymous: %d", anonymous.status)
	}
	f.s.fault = func(at string) bool { return at == "release-evidence-read" }
	if failed := f.readAs(root, path); failed.status != 503 {
		t.Errorf("a fault: %d", failed.status)
	}
	f.s.fault = nil
	f.s.bootWorkspace = "elsewhere"
	if other := f.readAs(root, path); other.status != 200 || len(other.body["builds"].([]any)) != 0 || whole(other.body["total"]) != 0 {
		t.Errorf("another workspace: %d", other.status)
	}
}

func TestEvidenceProfilesAreReadInRecordedOrderAndByTarget(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	seeded := len(f.s.ev.profiles)
	for i := 0; i < 70; i++ {
		id := fmt.Sprintf("bulk-profile-%03d", i)
		f.pfApproved(perfgate.Change{Kind: perfgate.ChangeProfile, Profile: perfgate.Profile{ID: id, Gate: perfgate.GateE2E, Axes: perfgate.Axes{RowsRun: 100000}, Dataset: sampleHash("d" + id),
			Expected: sampleHash("e" + id)}})
		f.s.ev.profiles = append(f.s.ev.profiles, id)
	}
	total := seeded + 70
	load := evMap(f.evidenceRead(root).body, "load")
	mainRows := evRows(evMap(load, "profiles"))
	if len(mainRows) != evidenceShown || load["profiles_next"] != "50" || whole(mainRows[0]["number"]) != 1 || whole(mainRows[49]["number"]) != 50 || whole(evMap(load, "profiles")["total"]) != total {
		t.Fatalf("the main read keeps its order and says where it stopped: next %v, %d rows", load["profiles_next"], len(mainRows))
	}
	path := "/api/v1/release-evidence/profiles"
	all, next, pages := append([]map[string]any{}, mainRows...), "50", 0
	for next != "" {
		page := f.readAs(root, path+"?limit=15&after="+next)
		if page.status != 200 || whole(page.body["total"]) != len(f.s.ev.profiles) || whole(page.body["matching"]) != len(f.s.ev.profiles) || page.body["order"] != "recorded" {
			t.Fatalf("after %s: %s", next, page.raw[:min(len(page.raw), 300)])
		}
		for _, raw := range page.body["profiles"].([]any) {
			all = append(all, raw.(map[string]any))
		}
		next = fmt.Sprint(page.body["next"])
		if pages++; pages == 1 {
			f.pfApproved(perfgate.Change{Kind: perfgate.ChangeProfile, Profile: perfgate.Profile{ID: "added-meanwhile", Gate: perfgate.GateE2E, Axes: perfgate.Axes{RowsRun: 100000}, Dataset: sampleHash("dm"),
				Expected: sampleHash("em")}})
			f.s.ev.profiles = append(f.s.ev.profiles, "added-meanwhile")
		}
		if pages > 10 {
			t.Fatal("the profiles do not end")
		}
	}
	if len(all) != total+1 || all[len(all)-1]["id"] != "added-meanwhile" {
		t.Fatalf("the pages hold %d of %d", len(all), total+1)
	}
	for index, row := range all {
		if whole(row["number"]) != index+1 || row["id"] != f.s.ev.profiles[index] {
			t.Fatalf("row %d is %v number %v", index, row["id"], row["number"])
		}
	}
	first := f.readAs(root, path+"?limit=50")
	fresh := evRows(evMap(f.evidenceRead(root).body, "load", "profiles"))
	if rows, _ := first.body["profiles"].([]any); len(rows) != len(fresh) || !reflect.DeepEqual(rows[0], map[string]any(fresh[0])) || !reflect.DeepEqual(rows[49], map[string]any(fresh[49])) {
		t.Error("the first page of the further read differs from the rows of the main read")
	}
	last := perfProfileTarget + "bulk-profile-069"
	byTarget := f.readAs(root, path+"?target="+last+","+perfProfileTarget+f.s.ev.profiles[1]+","+perfProfileTarget+"never-recorded")
	if rows, _ := byTarget.body["profiles"].([]any); byTarget.status != 200 || len(rows) != 2 || rows[0].(map[string]any)["id"] != f.s.ev.profiles[1] || rows[1].(map[string]any)["target"] != last ||
		whole(byTarget.body["matching"]) != 2 || byTarget.body["next"] != "" || !reflect.DeepEqual(rows[0], map[string]any(fresh[1])) {
		t.Fatalf("by target: %s", byTarget.raw[:min(len(byTarget.raw), 300)])
	}
	found := f.readAs(root, path+"?q=PROFILE-033")
	if rows, _ := found.body["profiles"].([]any); found.status != 200 || len(rows) != 1 || rows[0].(map[string]any)["id"] != "bulk-profile-033" || whole(found.body["matching"]) != 1 {
		t.Errorf("a search by identifier: %s", found.raw[:min(len(found.raw), 300)])
	}
	many := []string{}
	for n := 0; n <= profileTargets; n++ {
		many = append(many, perfProfileTarget+fmt.Sprintf("bulk-profile-%03d", n))
	}
	for _, query := range []string{"?target=", "?target=profile:x", "?target=" + perfProfileTarget, "?target=" + last + "," + last, "?target=" + last + ",", "?target=" + strings.Join(many, ","),
		"?target=" + last + "&limit=5", "?target=" + last + "&q=a", "?target=" + last + "&target=" + last, "?after=0", "?after=x", "?limit=0", "?limit=101", "?q=", "?state=x", "?limit=5&limit=6"} {
		if refused := f.readAs(root, path+query); refused.status != 422 {
			t.Errorf("%s: %d", query, refused.status)
		}
	}
	if twenty := f.readAs(root, path+"?target="+strings.Join(many[:profileTargets], ",")); twenty.status != 200 || len(twenty.body["profiles"].([]any)) != profileTargets {
		t.Errorf("twenty targets: %d", twenty.status)
	}
	f.code(t, "member", f.readAs(f.as(MemberLogin), path), 403, "permission_denied")
	f.code(t, "anonymous", f.readAs(actor{}, path), 401, "unauthenticated")
	f.s.fault = func(at string) bool { return at == "release-evidence-read" }
	if failed := f.readAs(root, path); failed.status != 503 {
		t.Errorf("a fault: %d", failed.status)
	}
	f.s.fault = nil
	f.s.ev.profiles = append(f.s.ev.profiles, "never-recorded")
	if failed := f.readAs(root, path+"?target="+perfProfileTarget+"never-recorded"); failed.status != 503 {
		t.Errorf("an unknown stored profile: %d", failed.status)
	}
	f.s.bootWorkspace = "elsewhere"
	if other := f.readAs(root, path); other.status != 200 || len(other.body["profiles"].([]any)) != 0 || whole(other.body["total"]) != 0 {
		t.Errorf("another workspace: %d", other.status)
	}
}
