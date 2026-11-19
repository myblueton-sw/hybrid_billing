package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/webhooks"
)

func TestTheStartingRightAloneKeepsWorkApprovable(t *testing.T) {
	for _, c := range []struct {
		approval string
		right    gate.Action
		state    string
	}{
		{pluginApproveKind, gate.AdministerModules, "applied"},
		{subApproveKind, gate.AdministerChannels, "active"},
		{subResumeKind, gate.AdministerChannels, "active"},
		{letterAgainKind, gate.ReprocessEvents, "queued"},
		{letterDropKind, gate.ReprocessEvents, "dismissed"},
	} {
		f := setup(t, SeedSample).ready()
		f.setRights(t, SampleAdmin3, append(gate.IntegrationRights(), gate.AdministerInstallation)...)
		target, approver := f.pendingBy3(t, c.approval)
		f.setRights(t, SampleAdmin3, gate.AdministerInstallation, c.right)
		f.ok(t, c.approval, f.run(approver, integrationCommand+c.approval, target, nil), c.state, "")
	}
	f := setup(t, SeedSample).ready()
	id, root, _ := f.heldRedelivery(t)
	f.setRights(t, SampleAdmin2, gate.AdministerInstallation)
	if err := f.dispatch(); err != nil {
		t.Fatal(err)
	}
	f.setRights(t, SampleAdmin3, gate.AdministerInstallation, gate.ReprocessEvents)
	f.ok(t, "re-approval with the requester's right", f.run(root, integrationCommand+reapproveKind, redeliveryTarget+id, nil), "queued", "")
}

func TestClosingsLeaveHistory(t *testing.T) {
	for _, approval := range []string{letterAgainKind, letterDropKind, subResumeKind} {
		f := setup(t, SeedSample).ready()
		f.setRights(t, SampleAdmin3, append(gate.IntegrationRights(), gate.AdministerInstallation)...)
		target, approver := f.pendingBy3(t, approval)
		f.setRights(t, SampleAdmin3, gate.AdministerInstallation)
		got := f.run(approver, integrationCommand+approval, target, nil)
		f.ok(t, approval, got, integrationClosed, "")
		history := fmt.Sprint(subscriptionRowFor(t, f.integrations(approver), sampleSubscription)["history"])
		if !strings.Contains(history, "reason:requester_lost_right") || !strings.Contains(history, "request:"+fmt.Sprint(got.body["request_id"])) ||
			!strings.Contains(history, "actor:"+SampleAdmin2) {
			t.Errorf("%s history %s", approval, history)
		}
	}
}

func TestNewCommandFaultsChangeNothing(t *testing.T) {
	type step struct {
		fault, kind string
		prepare     func(f world) (string, actor)
	}
	steps := []step{
		{"plugins-close", pluginApproveKind, func(f world) (string, actor) { return f.pendingWithLoss(t, pluginApproveKind) }},
		{"webhooks-close", subApproveKind, func(f world) (string, actor) { return f.pendingWithLoss(t, subApproveKind) }},
		{"webhooks-close_resume", subResumeKind, func(f world) (string, actor) { return f.pendingWithLoss(t, subResumeKind) }},
		{"webhooks-close_request", letterDropKind, func(f world) (string, actor) { return f.pendingWithLoss(t, letterDropKind) }},
		{"webhooks-withdraw_request", redeliverWithdrawKind, func(f world) (string, actor) { return letterRequestTarget + "letter-request-1", f.as(AdminLogin) }},
		{"webhooks-withdraw", subWithdrawKind, func(f world) (string, actor) { return subscriptionAt(samplePayments), f.as(Admin2Login) }},
		{"webhooks-withdraw_resume", resumeWithdrawKind, func(f world) (string, actor) {
			f.mustOK(t, "pause", f.run(f.as(AdminLogin), integrationCommand+subPauseKind, subscriptionAt(sampleSubscription), nil))
			f.mustOK(t, "request", f.run(f.as(AdminLogin), integrationCommand+subRequestKind, subscriptionAt(sampleSubscription), nil))
			return resumeRequestTarget + f.s.integ.subscription(sampleSubscription).resume.id, f.as(AdminLogin)
		}},
	}
	for _, c := range steps {
		f := setup(t, SeedSample).ready()
		target, who := c.prepare(f)
		f.s.fault = func(s string) bool { return s == c.fault }
		f.code(t, c.fault, f.run(who, integrationCommand+c.kind, target, nil), 500, "internal_error")
		f.s.fault = nil
		if got := f.run(who, integrationCommand+c.kind, target, nil); got.status != 200 {
			t.Errorf("%s left the work changed: %s", c.fault, got.raw)
		}
	}
	for _, fault := range []string{"webhooks-held", "webhooks-reapprove", "webhooks-close_redelivery"} {
		f := setup(t, SeedSample).ready()
		id, root, _ := f.heldRedelivery(t)
		f.setRights(t, SampleAdmin2, gate.AdministerInstallation)
		if err := f.dispatch(); err != nil {
			t.Fatal(err)
		}
		if fault == "webhooks-close_redelivery" {
			f.setRights(t, SampleAdmin3, gate.AdministerInstallation)
		}
		f.s.fault = func(s string) bool { return s == fault }
		f.code(t, fault, f.run(root, integrationCommand+reapproveKind, redeliveryTarget+id, nil), 500, "internal_error")
		f.s.fault = nil
		if f.s.integ.held[id] == "" {
			t.Errorf("%s dropped the hold record", fault)
		}
	}
}

func (f world) pendingWithLoss(t *testing.T, approval string) (string, actor) {
	f.setRights(t, SampleAdmin3, append(gate.IntegrationRights(), gate.AdministerInstallation)...)
	target, approver := f.pendingBy3(t, approval)
	f.setRights(t, SampleAdmin3, gate.AdministerInstallation)
	return target, approver
}

func TestResumeRequestsAreWithdrawnByTheirAuthor(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	f.mustOK(t, "pause", f.run(root, integrationCommand+subPauseKind, subscriptionAt(sampleSubscription), nil))
	f.mustOK(t, "request", f.run(second, integrationCommand+subRequestKind, subscriptionAt(sampleSubscription), nil))
	first := f.s.integ.subscription(sampleSubscription).resume
	if first == nil || !strings.HasPrefix(first.id, "resume-request-") || first.by != SampleAdmin2 {
		t.Fatalf("resume record %+v", first)
	}
	target := resumeRequestTarget + first.id
	read := f.integrations(second)
	row := subscriptionRowFor(t, read, sampleSubscription)
	if r := row["resume_request"].(map[string]any); r["id"] != first.id || r["target"] != target || r["by"] != SampleAdmin2 {
		t.Errorf("resume row %v", r)
	}
	if a := row["actions"].(map[string]any)["withdraw_resume"].(map[string]any); a["allowed"] != true {
		t.Errorf("own withdraw %v", a)
	}
	if a := subscriptionRowFor(t, f.integrations(root), sampleSubscription)["actions"].(map[string]any)["withdraw_resume"].(map[string]any); a["allowed"] != false {
		t.Errorf("another's withdraw %v", a)
	}
	f.code(t, "another person", f.run(root, integrationCommand+resumeWithdrawKind, target, nil), 403, "permission_denied")
	f.code(t, "a letter request target", f.run(second, integrationCommand+resumeWithdrawKind, letterRequestTarget+first.id, nil), 403, "permission_denied")
	n := keys.Add(1)
	body, _ := json.Marshal(map[string]any{"request_id": fmt.Sprintf("req-%d", n), "target_id": target, "idempotency_key": fmt.Sprintf("key-%d", n),
		"expected_revision": f.s.Revision(), "rationale": "sample reason", "body": map[string]string{}})
	done := f.call(integrationCommand+resumeWithdrawKind, string(body), second.cookie, second.csrf)
	if done.status != 200 || done.body["business_state"] != "withdrawn" {
		t.Fatalf("withdraw %s", done.raw)
	}
	if v := subscriptionRowFor(t, f.integrations(root), sampleSubscription); v["resumer"] != "" || v["resume_request"] != nil {
		t.Errorf("after withdrawal %v %v", v["resumer"], v["resume_request"])
	}
	f.mustOK(t, "B requests", f.run(root, integrationCommand+subRequestKind, subscriptionAt(sampleSubscription), nil))
	again := f.call(integrationCommand+resumeWithdrawKind, string(body), second.cookie, second.csrf)
	if again.status != 200 || again.body["replay"] != true {
		t.Errorf("replay %s", again.raw)
	}
	f.mustOK(t, "B withdraws", f.run(root, integrationCommand+resumeWithdrawKind, resumeRequestTarget+f.s.integ.subscription(sampleSubscription).resume.id, nil))
	f.mustOK(t, "A requests again", f.run(second, integrationCommand+subRequestKind, subscriptionAt(sampleSubscription), nil))
	f.unchanged(t, "a fresh key on the older request", f.run(second, integrationCommand+resumeWithdrawKind, target, nil), "invalid_stage")
}

func TestSubscriptionWithdrawalAvailability(t *testing.T) {
	f := setup(t, SeedSample).ready()
	own := subscriptionRowFor(t, f.integrations(f.as(Admin2Login)), samplePayments)["actions"].(map[string]any)
	other := subscriptionRowFor(t, f.integrations(f.as(AdminLogin)), samplePayments)["actions"].(map[string]any)
	if own["withdraw"].(map[string]any)["allowed"] != true || other["withdraw"].(map[string]any)["blocked"] != "not_requester" {
		t.Errorf("withdraw %v %v", own["withdraw"], other["withdraw"])
	}
	active := subscriptionRowFor(t, f.integrations(f.as(AdminLogin)), sampleSubscription)["actions"].(map[string]any)
	if active["withdraw"].(map[string]any)["blocked"] != "not_pending" || active["withdraw_resume"].(map[string]any)["blocked"] != "not_pending" {
		t.Errorf("active %v", active)
	}
	f.unchanged(t, "withdraw an active subscription", f.run(f.as(AdminLogin), integrationCommand+subWithdrawKind, subscriptionAt(sampleSubscription), nil), "invalid_stage")
	f.code(t, "a missing destination", f.run(f.as(AdminLogin), integrationCommand+subProposeKind, webhooksTarget, map[string]string{"types": typePayment}), 422, "invalid_request")
}

func TestApproveResumeChecksAdmissionOnlyForApprovers(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.mustOK(t, "pause", f.run(root, integrationCommand+subPauseKind, subscriptionAt(sampleSubscription), nil))
	f.mustOK(t, "request", f.run(root, integrationCommand+subRequestKind, subscriptionAt(sampleSubscription), nil))
	f.s.fault = func(s string) bool { return s == "destinations" }
	got := f.integrations(f.as(Admin2Login))
	f.s.fault = nil
	if a := subscriptionRowFor(t, got, sampleSubscription)["actions"].(map[string]any)["approve_resume"].(map[string]any); a["blocked"] != "dependency_unavailable" {
		t.Errorf("approve resume %v", a)
	}
}

func TestHeldRecordsWithoutAHeldDeliveryAreDropped(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.s.integ.held["delivery-1"] = sampleSubscription
	f.unchanged(t, "not held", f.run(f.as(AdminLogin), integrationCommand+reapproveKind, redeliveryTarget+"delivery-1", nil), "invalid_stage")
	if _, kept := f.s.integ.held["delivery-1"]; kept {
		t.Error("the stale hold record was kept")
	}
}

func TestAuthorKindsFindOnlyTheirOwnRecords(t *testing.T) {
	f := setup(t, SeedSample).ready()
	for _, kind := range []string{pluginWithdrawKind, subWithdrawKind, resumeWithdrawKind, redeliverWithdrawKind, dismissWithdrawKind} {
		h := integrationHandler{s: f.s, kind: kind}
		for _, target := range []string{"", "plugin-proposal-99", subscriptionTarget + "missing", resumeRequestTarget + "letter-request-1", letterRequestTarget + "missing"} {
			if author, ok := h.author(target); ok || author != "" {
				t.Errorf("%s %s: %q %v", kind, target, author, ok)
			}
		}
	}
	if _, ok := (integrationHandler{s: f.s, kind: dismissWithdrawKind}).author(letterRequestTarget + "letter-request-1"); ok {
		t.Error("a redelivery request resolved for a dismissal withdrawal")
	}
	if author, ok := (integrationHandler{s: f.s, kind: redeliverWithdrawKind}).author(letterRequestTarget + "letter-request-1"); !ok || author != SampleAdmin {
		t.Errorf("own kind %q %v", author, ok)
	}
}

func TestStandingIsFailClosedAndExact(t *testing.T) {
	f := setup(t, SeedSample).ready()
	now := f.s.clock()
	if got, code := f.s.standingOf(SampleAdmin, gate.AdministerModules, now); got != qualifies || code != "" {
		t.Errorf("qualifies %v %s", got, code)
	}
	if got, code := f.s.standingOf(SampleAdmin3, gate.AdministerModules, now); got != lost || code != "" {
		t.Errorf("lost %v %s", got, code)
	}
	if got, code := f.s.standingOf("bad id", gate.AdministerModules, now); got != unknown || code != "dependency_unavailable" {
		t.Errorf("an unreadable user %v %s", got, code)
	}
	boot := f.s.bootWorkspace
	f.s.bootWorkspace = ""
	if got, code := f.s.standingOf(SampleAdmin, gate.AdministerModules, now); got != unknown || code != "dependency_unavailable" {
		t.Errorf("an unreadable membership %v %s", got, code)
	}
	f.s.bootWorkspace = boot
	if err := f.s.admin.SetAvailable(false); err != nil {
		t.Fatal(err)
	}
	if got, code := f.s.standingOf(SampleAdmin, gate.AdministerModules, now); got != unknown || code != "authority_unavailable" {
		t.Errorf("an unavailable authority %v %s", got, code)
	}
	_ = f.s.admin.SetAvailable(true)
	if gone, err := f.s.starterCheck("", gate.AdministerModules); gone || err == nil {
		t.Errorf("an empty starter %v %v", gone, err)
	}
	f.s.fault = func(s string) bool { return s == "starter-identity" }
	if gone, err := f.s.starterCheck(SampleAdmin, gate.AdministerModules); gone || err == nil {
		t.Errorf("an unknown starter %v %v", gone, err)
	}
}

func TestTheReadUsesEachStarterRight(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setRights(t, SampleAdmin3, append(gate.IntegrationRights(), gate.AdministerInstallation)...)
	ptarget, approver := f.pendingBy3(t, pluginApproveKind)
	starget, _ := f.pendingBy3(t, subApproveKind)
	f.pendingBy3(t, letterAgainKind)
	f.setRights(t, SampleAdmin3, gate.AdministerInstallation, gate.AdministerModules, gate.AdministerChannels, gate.ReprocessEvents)
	got := f.integrations(approver)
	if b := rowWith(t, got, "modules", "proposals", "target", ptarget)["approve_blocked"]; b == blockedLostRight {
		t.Errorf("plugin %v", b)
	}
	if b := subscriptionRowFor(t, got, strings.TrimPrefix(starget, subscriptionTarget))["actions"].(map[string]any)["approve"].(map[string]any)["blocked"]; b == blockedLostRight {
		t.Errorf("subscription %v", b)
	}
	if b := deliveryRowFor(t, got, sampleLetter)["actions"].(map[string]any)["approve_redeliver"].(map[string]any)["blocked"]; b == blockedLostRight {
		t.Errorf("letter %v", b)
	}
}

func TestTheReadRefusesBadSessions(t *testing.T) {
	f := setup(t, SeedSample).ready()
	a := f.as(AdminLogin)
	f.code(t, "no csrf", f.integrations(actor{cookie: a.cookie, csrf: "wrong"}), 403, CodeCSRF)
	f.code(t, "no session", f.integrations(actor{}), 401, "unauthenticated")
	f.mustOK(t, "suspend", f.run(f.as(Admin2Login), "identity.suspend_user", SampleAdmin3, nil))
	if got := f.integrations(f.as3OrNone()); got.status == 200 {
		t.Errorf("a suspended user read P02 %s", got.raw)
	}
}

func (f world) as3OrNone() actor {
	a, _ := f.as3()
	return a
}

func TestRequestEvictionKeepsLiveRequestsAndStopsAtTheLimit(t *testing.T) {
	f := setup(t, SeedSample).ready()
	c := f.s.integ
	c.requests.limit = 1
	old1 := c.newRequest(requestDismiss, SampleAdmin, "x1", sampleSubscription)
	if c.requests.records[old1.id] == nil {
		t.Fatal("a single request was evicted at the limit")
	}
	c.requests.limit = 10
	old2 := c.newRequest(requestDismiss, SampleAdmin, "x2", sampleSubscription)
	old3 := c.newRequest(requestDismiss, SampleAdmin, "x3", sampleSubscription)
	c.dismiss["x3"] = old3
	c.requests.limit = 1
	fresh := c.newRequest(requestDismiss, SampleAdmin, "x4", sampleSubscription)
	if c.requests.records[old1.id] != nil || c.requests.records[old2.id] != nil {
		t.Errorf("not every evictable record went %v", c.requests.order[requestDismiss])
	}
	if c.requests.records[old3.id] == nil || c.requests.records[fresh.id] == nil {
		t.Errorf("a live or new record went %v", c.requests.order[requestDismiss])
	}
	g := setup(t, SeedSample).ready()
	root := g.as(AdminLogin)
	g.mustOK(t, "pause", g.run(root, integrationCommand+subPauseKind, subscriptionAt(sampleSubscription), nil))
	g.mustOK(t, "request 1", g.run(root, integrationCommand+subRequestKind, subscriptionAt(sampleSubscription), nil))
	first := g.s.integ.subscription(sampleSubscription).resume.id
	g.mustOK(t, "withdraw 1", g.run(root, integrationCommand+resumeWithdrawKind, resumeRequestTarget+first, nil))
	g.s.integ.requests.limit = 1
	g.mustOK(t, "request 2", g.run(root, integrationCommand+subRequestKind, subscriptionAt(sampleSubscription), nil))
	second := g.s.integ.subscription(sampleSubscription).resume.id
	if g.s.integ.requests.records[first] != nil || g.s.integ.requests.records[second] == nil {
		t.Errorf("resume eviction %v", g.s.integ.requests.order[requestResume])
	}
	if !g.s.integ.liveRequest(g.s.integ.requests.records[second]) || g.s.integ.liveRequest(&requestRecord{id: first, kind: requestResume, object: sampleSubscription}) ||
		g.s.integ.liveRequest(&requestRecord{id: second, kind: requestResume, object: "missing"}) {
		t.Error("resume liveness")
	}
	if (integrationHandler{s: g.s, kind: resumeWithdrawKind}).found(second) {
		t.Error("a bare request ID resolved without its prefix")
	}
}

func TestOwnPendingListsOnlyTheCallersLiveWork(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	f.mustOK(t, "dismissal", f.run(second, integrationCommand+letterDismissKind, letterAt(sampleLetter), nil))
	f.mustOK(t, "pause", f.run(root, integrationCommand+subPauseKind, subscriptionAt(sampleSubscription), nil))
	f.mustOK(t, "resume", f.run(second, integrationCommand+subRequestKind, subscriptionAt(sampleSubscription), nil))
	kinds := func(a actor) string {
		out := []string{}
		for _, item := range f.integrations(a).body["own_pending"].([]any) {
			m := item.(map[string]any)
			out = append(out, fmt.Sprint(m["kind"], ":", m["id"]))
		}
		return strings.Join(out, " ")
	}
	if got := kinds(root); got != "plugin_proposal:9 redeliver_request:letter-request-1" {
		t.Errorf("admin %s", got)
	}
	resume := f.s.integ.subscription(sampleSubscription).resume.id
	if got := kinds(second); got != "plugin_proposal:8 resume_request:"+resume+" subscription_proposal:"+samplePayments+" dismiss_request:letter-request-2" {
		t.Errorf("admin2 %s", got)
	}
	f.mustOK(t, "approve resume", f.run(root, integrationCommand+subResumeKind, subscriptionAt(sampleSubscription), nil))
	f.mustOK(t, "approve payments", f.run(root, integrationCommand+subApproveKind, subscriptionAt(samplePayments), nil))
	f.mustOK(t, "withdraw redelivery", f.run(root, integrationCommand+redeliverWithdrawKind, letterRequestTarget+"letter-request-1", nil))
	if got := kinds(second); got != "plugin_proposal:8 dismiss_request:letter-request-2" {
		t.Errorf("admin2 after %s", got)
	}
	if got := kinds(root); got != "plugin_proposal:9" {
		t.Errorf("admin after %s", got)
	}
	if n := len(f.s.integ.redeliver); n != 0 || f.s.integ.subscription(sampleSubscription).resume != nil || f.s.integ.subscription(samplePayments).proposed {
		t.Errorf("settled state %d", n)
	}
}

func TestGrantFailuresAreReportedExactly(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	f.s.fault = func(s string) bool { return s == "integration-grant" }
	if err := f.s.grantIntegration(SampleMember, gate.IntegrationRights(), f.s.clock()); !errors.Is(err, errPartial) {
		t.Errorf("the grant fault %v", err)
	}
	f.s.fault = func(s string) bool { return s == "integration-grant-5" }
	if err := f.s.grantIntegration(SampleMember, gate.IntegrationRights(), f.s.clock()); !errors.Is(err, errPartial) || errors.Is(err, errRollback) {
		t.Errorf("a clean rollback %v", err)
	}
	f.s.fault = nil
	g := setup(t, SeedSample).ready()
	g.setPassword(MemberLogin, pw)
	if _, err := g.s.admin.PutClient(gate.Client{ID: ConsoleClient, State: gate.Suspended}, 1); err != nil {
		t.Fatal(err)
	}
	if err := g.s.grantIntegration(SampleMember, gate.IntegrationRights(), g.s.clock()); err == nil {
		t.Error("a client grant failure was ignored")
	}
	for _, gr := range g.s.admin.Grants() {
		if gr.SubjectID == SampleMember && strings.HasPrefix(gr.ID, integrationGrantID) {
			t.Errorf("a person grant was added after a client failure %s", gr.ID)
		}
	}
}

func TestScreeningSkipsAndFailsTruthfully(t *testing.T) {
	f := setup(t, SeedSample).ready()
	id, _, _ := f.heldRedelivery(t)
	f.s.integ.subscriptions = append(f.s.integ.subscriptions, &subscriptionEntry{id: "missing-subscription"})
	f.setRights(t, SampleAdmin2, gate.AdministerInstallation, gate.StopChannels)
	for _, fault := range []string{"webhooks-screen", "webhooks-hold"} {
		f.s.fault = func(s string) bool { return s == fault }
		if err := f.dispatch(); err == nil {
			t.Errorf("%s did not fail the round", fault)
		}
		f.s.fault = nil
		if _, held := f.s.integ.held[id]; held {
			t.Errorf("%s recorded a hold", fault)
		}
	}
	if err := f.dispatch(); err != nil {
		t.Fatal(err)
	}
	if f.s.integ.held[id] != sampleSubscription {
		t.Errorf("an approver with only the stop right was not held %v", f.s.integ.held)
	}
	g := setup(t, SeedSample).ready()
	gid, groot, _ := g.heldRedelivery(t)
	g.mustOK(t, "demote", g.run(groot, "identity.change_role", SampleAdmin3, map[string]string{"role": "member"}))
	g.s.fault = func(s string) bool { return s == "webhooks-close_queued" }
	if err := g.dispatch(); err == nil {
		t.Error("a close fault did not fail the round")
	}
	g.s.fault = nil
	if row := deliveryRowFor(t, g.integrations(groot), gid); row["kind"] != "redelivery" {
		t.Errorf("closed despite the fault %v", row["kind"])
	}
}

func TestBulkImpactCountsResumeRequests(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.mustOK(t, "pause", f.run(root, integrationCommand+subPauseKind, subscriptionAt(sampleSubscription), nil))
	f.mustOK(t, "resume", f.run(f.as(Admin2Login), integrationCommand+subRequestKind, subscriptionAt(sampleSubscription), nil))
	for _, store := range proposalStores {
		if store.name == "integration_proposals" {
			if n := store.tally(f.s)[SampleAdmin2]; n != 3 {
				t.Errorf("admin2 integration work %d", n)
			}
		}
	}
}

func TestLetterAndHeldRowsReportTheirReasons(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	f.mustOK(t, "dismissal", f.run(second, integrationCommand+letterDismissKind, letterAt(sampleLetter), nil))
	mine := deliveryRowFor(t, f.integrations(root), sampleLetter)["actions"].(map[string]any)
	if mine["withdraw_redeliver"].(map[string]any)["allowed"] != true || mine["withdraw_dismiss"].(map[string]any)["blocked"] != "not_requester" {
		t.Errorf("admin letter actions %v", mine)
	}
	theirs := deliveryRowFor(t, f.integrations(second), sampleLetter)["actions"].(map[string]any)
	if theirs["withdraw_dismiss"].(map[string]any)["allowed"] != true || theirs["withdraw_redeliver"].(map[string]any)["blocked"] != "not_requester" {
		t.Errorf("admin2 letter actions %v", theirs)
	}
	f.ok(t, "withdraw the dismissal", f.run(second, integrationCommand+dismissWithdrawKind, letterRequestTarget+"letter-request-2", nil), "withdrawn", "")
	if _, kept := f.s.integ.dismiss[sampleLetter]; kept || f.s.integ.redeliver[sampleLetter].id != "letter-request-1" {
		t.Error("the dismissal withdrawal touched the wrong request")
	}
	f.setRights(t, SampleAdmin, gate.AdministerInstallation)
	f.setPassword(Admin3Login, pw)
	lostRow := deliveryRowFor(t, f.integrations(second), sampleLetter)["actions"].(map[string]any)["approve_redeliver"].(map[string]any)
	if lostRow["blocked"] != blockedLostRight {
		t.Errorf("letter starter lost %v", lostRow)
	}
	f.code(t, "a plugin proposal without kind", f.run(root, integrationCommand+pluginProposeKind, samplePluginAt, map[string]string{}), 422, "invalid_request")
	g := setup(t, SeedSample).ready()
	id, groot, _ := g.heldRedelivery(t)
	g.setRights(t, SampleAdmin2, gate.AdministerInstallation)
	if err := g.dispatch(); err != nil {
		t.Fatal(err)
	}
	if b := deliveryRowFor(t, g.integrations(g.as(Admin3Login)), id)["actions"].(map[string]any)["reapprove"].(map[string]any)["blocked"]; b != ownRequest {
		t.Errorf("the requester sees %v", b)
	}
	g.setRights(t, SampleAdmin3, gate.AdministerInstallation)
	if b := deliveryRowFor(t, g.integrations(groot), id)["actions"].(map[string]any)["reapprove"].(map[string]any)["blocked"]; b != blockedLostRight {
		t.Errorf("the requester lost the right but root sees %v", b)
	}
	g.ok(t, "close by re-approval", g.run(groot, integrationCommand+reapproveKind, redeliveryTarget+id, nil), integrationClosed, "")
	history := fmt.Sprint(subscriptionRowFor(t, g.integrations(groot), sampleSubscription)["history"])
	if !strings.Contains(history, "action:redelivery_closed") || !strings.Contains(history, "reason:requester_lost_right") {
		t.Errorf("history %s", history)
	}
	for approval, action := range map[string]string{letterAgainKind: "redeliver_request_closed", letterDropKind: "dismiss_request_closed", subResumeKind: "resume_closed"} {
		w := setup(t, SeedSample).ready()
		target, approver := w.pendingWithLoss(t, approval)
		w.ok(t, approval, w.run(approver, integrationCommand+approval, target, nil), integrationClosed, "")
		if h := fmt.Sprint(subscriptionRowFor(t, w.integrations(approver), sampleSubscription)["history"]); !strings.Contains(h, "action:"+action) {
			t.Errorf("%s history %s", approval, h)
		}
	}
}

func TestStarterChecksFailClosedOnEveryApproval(t *testing.T) {
	for _, approval := range []string{subApproveKind, subResumeKind, letterAgainKind, letterDropKind} {
		for step, code := range map[string]string{"starter-identity": "dependency_unavailable", "users": "dependency_unavailable", "starter-authority": "authority_unavailable"} {
			f := setup(t, SeedSample).ready()
			f.setRights(t, SampleAdmin3, append(gate.IntegrationRights(), gate.AdministerInstallation)...)
			target, approver := f.pendingBy3(t, approval)
			f.s.fault = func(s string) bool { return s == step }
			f.code(t, approval+" "+step, f.run(approver, integrationCommand+approval, target, nil), 503, code)
			f.s.fault = nil
			f.mustOK(t, approval+" after "+step, f.run(approver, integrationCommand+approval, target, nil))
		}
	}
	f := setup(t, SeedSample).ready()
	id, root, _ := f.heldRedelivery(t)
	f.setRights(t, SampleAdmin2, gate.AdministerInstallation)
	if err := f.dispatch(); err != nil {
		t.Fatal(err)
	}
	f.s.fault = func(s string) bool { return s == "starter-identity" }
	f.code(t, "re-approval with an unknown requester", f.run(root, integrationCommand+reapproveKind, redeliveryTarget+id, nil), 503, "dependency_unavailable")
	f.s.fault = nil
	if f.s.integ.held[id] == "" {
		t.Error("an unknown requester released or closed the hold")
	}
}

func TestEvictionStartsOnlyBeyondTheLimit(t *testing.T) {
	f := setup(t, SeedSample).ready()
	c := f.s.integ
	c.requests.limit = 2
	old := c.newRequest(requestDismiss, SampleAdmin, "y1", sampleSubscription)
	c.newRequest(requestDismiss, SampleAdmin, "y2", sampleSubscription)
	if c.requests.records[old.id] == nil {
		t.Error("a record was evicted at exactly the limit")
	}
}

func TestScreeningUsesTheReprocessRight(t *testing.T) {
	f := setup(t, SeedSample).ready()
	id, root, _ := f.heldRedelivery(t)
	f.setRights(t, SampleAdmin3, gate.AdministerInstallation, gate.StopChannels)
	if err := f.dispatch(); err != nil {
		t.Fatal(err)
	}
	if row := deliveryRowFor(t, f.integrations(root), id); row["kind"] != "cancelled" {
		t.Errorf("a requester with only the stop right was not closed out %v", row["kind"])
	}
}

func TestAnEndedSessionCannotReadP02(t *testing.T) {
	f := setup(t, SeedSample).ready()
	a := f.as(Admin3Login)
	f.mustOK(t, "suspend", f.run(f.as(AdminLogin), "identity.suspend_user", SampleAdmin3, nil))
	f.code(t, "the suspended user's session", f.integrations(a), 401, "unauthenticated")
}

func TestTheFilterScreensOnlyQueuedUnheldRedeliveries(t *testing.T) {
	base := webhooks.DeliveryView{Kind: webhooks.KindRedelivery, State: webhooks.JobQueued}
	if !screenable(base) {
		t.Error("a queued redelivery is not screened")
	}
	for name, d := range map[string]webhooks.DeliveryView{
		"open":    {Kind: webhooks.KindOpen, State: webhooks.JobQueued},
		"dead":    {Kind: webhooks.KindDead, State: webhooks.JobQueued},
		"running": {Kind: webhooks.KindRedelivery, State: webhooks.JobRunning},
		"held":    {Kind: webhooks.KindRedelivery, State: webhooks.JobQueued, Hold: webhooks.ReasonApproverLostRight},
	} {
		if screenable(d) {
			t.Errorf("%s was screened", name)
		}
	}
}

func TestOneUnknownStarterSkipsTheWholeRound(t *testing.T) {
	for _, who := range []string{SampleAdmin3, SampleAdmin2} {
		f := setup(t, SeedSample).ready()
		id, root, _ := f.heldRedelivery(t)
		rounds := 0
		f.s.fault = func(s string) bool {
			if s == "webhooks-dispatch_round" {
				rounds++
			}
			return s == "starter-identity-"+who
		}
		if err := f.dispatch(); err == nil {
			t.Errorf("%s unknown: the round ran", who)
		}
		f.s.fault = nil
		row := deliveryRowFor(t, f.integrations(root), id)
		if rounds != 0 || row["hold"] != "" || row["kind"] != webhooks.KindRedelivery {
			t.Errorf("%s unknown: rounds %d, row %v %v", who, rounds, row["hold"], row["kind"])
		}
	}
}

func TestAHeldRedeliveryIsNotScreenedAgain(t *testing.T) {
	f := setup(t, SeedSample).ready()
	id, root, _ := f.heldRedelivery(t)
	f.setRights(t, SampleAdmin2, gate.AdministerInstallation)
	if err := f.dispatch(); err != nil {
		t.Fatal(err)
	}
	f.mustOK(t, "demote the requester", f.run(root, "identity.change_role", SampleAdmin3, map[string]string{"role": "member"}))
	if err := f.dispatch(); err != nil {
		t.Fatal(err)
	}
	if row := deliveryRowFor(t, f.integrations(root), id); row["hold"] != webhooks.ReasonApproverLostRight || row["kind"] != webhooks.KindRedelivery {
		t.Errorf("a held redelivery was screened again %v %v", row["hold"], row["kind"])
	}
}

func TestDeliveryListsArePagedPastOnePage(t *testing.T) {
	saved := deliveryPage
	deliveryPage = 1
	defer func() { deliveryPage = saved }()
	f := setup(t, SeedSample).ready()
	id, root, _ := f.heldRedelivery(t)
	f.setRights(t, SampleAdmin2, gate.AdministerInstallation)
	if err := f.dispatch(); err != nil {
		t.Fatal(err)
	}
	if f.s.integ.held[id] == "" {
		t.Fatal("a redelivery beyond the first page was not screened")
	}
	f.ok(t, "re-approve beyond the first page", f.run(root, integrationCommand+reapproveKind, redeliveryTarget+id, nil), "queued", "")
}
