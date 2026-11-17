package server

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/plugins"
	"github.com/myblueton-sw/hybrid_billing/modules/webhooks"
)

var (
	sampleAdminWebhook  = webhooks.Actor{ID: SampleAdmin, Kind: webhooks.KindHuman}
	sampleAdmin2Webhook = webhooks.Actor{ID: SampleAdmin2, Kind: webhooks.KindHuman}
	sampleAdminPlugin   = plugins.Actor{ID: SampleAdmin, Kind: plugins.KindHuman}
	sampleAdmin2Plugin  = plugins.Actor{ID: SampleAdmin2, Kind: plugins.KindHuman}
)

func (f world) unchanged(t *testing.T, name string, got reply, code string) {
	t.Helper()
	before := f.s.Revision()
	f.code(t, name, got, 422, code)
	if f.s.Revision() != before {
		t.Errorf("%s moved the revision", name)
	}
}

func (f world) ok(t *testing.T, name string, got reply, state, fields string) {
	t.Helper()
	if got.status != 200 || got.body["business_state"] != state || fields != "" && fmt.Sprint(got.body["fields"]) != fields {
		t.Fatalf("%s: %s", name, got.raw)
	}
}

func installedRow(t testing.TB, got reply) map[string]any {
	t.Helper()
	return rowWith(t, got, "modules", "installations", "plugin", samplePlugin)
}

func TestIntegrationPluginApproveAndWithdraw(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	f.ok(t, "approve", f.run(root, "integrations.plugin_approve", pluginProposalAt(8), nil), "applied", "map[proposal:8 target:plugin-proposal-8]")
	read := f.integrations(root)
	upgrade := pluginProposal(t, read, 8)
	if s := fmt.Sprint(upgrade["state"], " ", upgrade["approver"], " ", upgrade["closed_by_login"], " ", upgrade["closing_rationale"], " ", blockedAs(upgrade)); s !=
		"applied sample-admin admin@example.invalid sample reason false not_pending false not_pending" {
		t.Errorf("approved %s", s)
	}
	if v := installedRow(t, read); v["version"] != "1.1.0" || v["state"] != "enabled" {
		t.Errorf("upgraded %v", v)
	}
	f.unchanged(t, "approve applied again", f.run(root, "integrations.plugin_approve", pluginProposalAt(8), nil), "invalid_stage")
	f.unchanged(t, "approve applied by another", f.run(second, "integrations.plugin_approve", pluginProposalAt(8), nil), "invalid_stage")
	f.unchanged(t, "self approval", f.run(root, "integrations.plugin_approve", pluginProposalAt(9), nil), "prerequisite_failed")
	f.unchanged(t, "not proposer", f.run(second, "integrations.plugin_withdraw", pluginProposalAt(9), nil), "prerequisite_failed")
	f.ok(t, "withdraw", f.run(root, "integrations.plugin_withdraw", pluginProposalAt(9), nil), "withdrawn", "map[proposal:9 target:plugin-proposal-9]")
	gone := pluginProposal(t, f.integrations(second), 9)
	if s := fmt.Sprint(gone["state"], " ", gone["closed_by"], " ", blockedAs(gone)); s != "withdrawn sample-admin false not_pending false not_pending" {
		t.Errorf("withdrawn %s", s)
	}
	if _, err := f.s.plugins.Proposal(9); !errors.Is(err, plugins.ErrNotFound) {
		t.Errorf("module kept a withdrawn proposal %v", err)
	}
	f.unchanged(t, "approve after withdraw", f.run(second, "integrations.plugin_approve", pluginProposalAt(9), nil), "invalid_stage")
	f.unchanged(t, "withdraw twice", f.run(root, "integrations.plugin_withdraw", pluginProposalAt(9), nil), "invalid_stage")
	g := setup(t, SeedSample).ready()
	g.ok(t, "approve endpoint", g.run(g.as(Admin2Login), "integrations.plugin_approve", pluginProposalAt(9), nil), "applied", "")
	got := g.integrations(g.as(AdminLogin))
	if s := fmt.Sprint(installedRow(t, got)["endpoints"]); s != "[map[approver:sample-admin-2 approver_login:admin2@example.invalid fingerprint:"+sampleHash("sample-certificate-api")+" id:api]]" {
		t.Errorf("endpoints %s", s)
	}
	if strings.Contains(got.raw, sampleHash("sample-evidence-api")) {
		t.Error("read carries endpoint evidence")
	}
}

func TestIntegrationPluginPropose(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	f.unchanged(t, "rollback below the installed version", f.run(root, "integrations.plugin_propose", samplePluginAt, map[string]string{"kind": "rollback", "version": "1.0.0"}), "invalid_stage")
	f.unchanged(t, "install while installed", f.run(root, "integrations.plugin_propose", samplePluginAt, map[string]string{"kind": "install", "version": "1.0.0"}), "invalid_stage")
	f.ok(t, "approve upgrade", f.run(root, "integrations.plugin_approve", pluginProposalAt(8), nil), "applied", "")
	fields := map[string]string{"kind": "rollback", "version": "1.0.0"}
	f.ok(t, "propose rollback", f.run(root, "integrations.plugin_propose", samplePluginAt, fields), "pending", "map[proposal:10 target:plugin-proposal-10]")
	f.ok(t, "duplicate", f.admin(root, "integrations.plugin_propose", samplePluginAt, "again", fields), "pending", "map[proposal:10 target:plugin-proposal-10]")
	mine := pluginProposal(t, f.integrations(root), 10)
	if s := fmt.Sprint(mine["rationale"], " ", mine["summary"].(map[string]any)["version"], " ", blockedAs(mine), " ", len(f.s.integ.entries)); s !=
		"sample reason 1.0.0 false own_proposal true  10" {
		t.Errorf("proposal %s", s)
	}
	if s := blockedAs(pluginProposal(t, f.integrations(second), 10)); s != "true  false not_proposer" {
		t.Errorf("second %s", s)
	}
	for _, c := range []map[string]string{{"kind": "install", "version": "9.9.9"}, {"kind": "upgrade"}, {"kind": "rollback", "version": "1.0"}, {"kind": "enable", "version": "1.0.0"},
		{"kind": "remove", "version": "x"}} {
		bad := f.run(root, "integrations.plugin_propose", samplePluginAt, c)
		if bad.status != 422 || bad.body["code"] != "invalid_request" || fmt.Sprint(bad.body["fields"]) != "[version]" {
			t.Errorf("%v: %s", c, bad.raw)
		}
	}
	f.s.integ.workspace = ""
	f.unchanged(t, "no plugin workspace", f.run(root, "integrations.plugin_propose", samplePluginAt, map[string]string{"kind": "enable"}), "invalid_stage")
	f.s.integ.workspace = SampleWorkspace
	f.ok(t, "approve rollback", f.run(second, "integrations.plugin_approve", pluginProposalAt(10), nil), "applied", "")
	if v := installedRow(t, f.integrations(root)); v["version"] != "1.0.0" {
		t.Errorf("rolled back %v", v)
	}
}

func TestIntegrationPluginBlockersFromRecheck(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	f.ok(t, "second upgrade", f.run(root, "integrations.plugin_propose", samplePluginAt, map[string]string{"kind": "upgrade", "version": "1.1.0"}), "pending", "map[proposal:10 target:plugin-proposal-10]")
	endpoint := f.s.integ.entry(9).view.Change
	if _, err := f.s.pluginPropose(endpoint, sampleAdmin2Plugin, "twin"); err != nil {
		t.Fatal(err)
	}
	f.ok(t, "approve upgrade", f.run(root, "integrations.plugin_approve", pluginProposalAt(8), nil), "applied", "")
	f.ok(t, "approve endpoint", f.run(second, "integrations.plugin_approve", pluginProposalAt(9), nil), "applied", "")
	f.s.integ.keep(&pluginEntry{view: plugins.ProposalView{Number: 999, State: plugins.ProposalPending, Proposer: SampleAdmin2, Change: plugins.Change{Kind: plugins.ChangeEnable}}})
	read := f.integrations(second)
	if s := blockedAs(pluginProposal(t, read, 10)); s != "false direction false not_proposer" {
		t.Errorf("direction %s", s)
	}
	read = f.integrations(root)
	if s := blockedAs(pluginProposal(t, read, 11)) + "|" + blockedAs(pluginProposal(t, read, 999)); s != "false conflict false not_proposer|false not_found false not_proposer" {
		t.Errorf("conflict and not found %s", s)
	}
	f.unchanged(t, "approve direction", f.run(second, "integrations.plugin_approve", pluginProposalAt(10), nil), "invalid_stage")
	f.unchanged(t, "approve conflict", f.run(root, "integrations.plugin_approve", pluginProposalAt(11), nil), "invalid_stage")
	f.unchanged(t, "approve missing", f.run(root, "integrations.plugin_approve", pluginProposalAt(999), nil), "invalid_stage")
	g := setup(t, SeedSample).ready()
	groot, gsecond := g.as(AdminLogin), g.as(Admin2Login)
	g.ok(t, "stop", g.run(groot, "integrations.plugin_stop", samplePluginAt, nil), "stopped", "map[target:plugin:sample-ledger-export]")
	g.ok(t, "enable by first", g.run(groot, "integrations.plugin_propose", samplePluginAt, map[string]string{"kind": "enable"}), "pending", "")
	g.ok(t, "enable by second", g.run(gsecond, "integrations.plugin_propose", samplePluginAt, map[string]string{"kind": "enable"}), "pending", "")
	g.ok(t, "approve enable", g.run(gsecond, "integrations.plugin_approve", pluginProposalAt(10), nil), "applied", "")
	if s := blockedAs(pluginProposal(t, g.integrations(groot), 11)); s != "false not_allowed_state false not_proposer" {
		t.Errorf("not allowed state %s", s)
	}
}

func TestIntegrationPluginLimitMapsToPrerequisite(t *testing.T) {
	f := setup(t, SeedSample).ready()
	small := PluginLimits()
	small.Publishers = 2
	r, err := plugins.NewRegistry(plugins.Config{Core: pluginCore, Destinations: integrationDestinations{s: f.s}, Random: strings.NewReader(strings.Repeat("x", 64))}, small, f.s.clock)
	if err != nil {
		t.Fatal(err)
	}
	f.s.plugins, f.s.integ = r, newIntegrationStore()
	if err := f.s.seedPlugins(f.s.clock()); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		id string
		by plugins.Actor
	}{{"sample-b", sampleAdminPlugin}, {"sample-c", sampleAdminPlugin}} {
		if _, err := f.s.pluginPropose(plugins.Change{Kind: plugins.ChangePublisher, Publisher: plugins.Publisher{ID: c.id, Name: "N"}}, c.by, "r"); err != nil {
			t.Fatal(err)
		}
	}
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	f.ok(t, "approve b", f.run(second, "integrations.plugin_approve", pluginProposalAt(10), nil), "applied", "")
	if s := blockedAs(pluginProposal(t, f.integrations(second), 11)); s != "false limit false not_proposer" {
		t.Errorf("limit %s", s)
	}
	f.unchanged(t, "limit", f.run(second, "integrations.plugin_approve", pluginProposalAt(11), nil), "prerequisite_failed")
	if n := len(part(f.integrations(root), "modules")["publishers"].([]any)); n != 2 {
		t.Errorf("publishers %d", n)
	}
}

func TestIntegrationPluginStopAndDisable(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	f.ok(t, "stop", f.run(root, "integrations.plugin_stop", samplePluginAt, nil), "stopped", "map[target:plugin:sample-ledger-export]")
	v := installedRow(t, f.integrations(root))
	if s := fmt.Sprint(v["state"], v["reason"], v["can_stop"], v["stop_blocked"], v["can_disable"], v["disable_blocked"]); s != "stoppedoperatorfalsenot_enabledtrue" {
		t.Errorf("stopped %s", s)
	}
	f.unchanged(t, "stop stopped", f.run(second, "integrations.plugin_stop", samplePluginAt, nil), "invalid_stage")
	f.ok(t, "disable", f.run(second, "integrations.plugin_disable", samplePluginAt, nil), "installed", "map[target:plugin:sample-ledger-export]")
	v = installedRow(t, f.integrations(root))
	if s := fmt.Sprint(v["state"], v["can_stop"], v["can_disable"], v["disable_blocked"]); s != "installedfalse falsenot_allowed_state" {
		t.Errorf("disabled %s", s)
	}
	f.unchanged(t, "disable installed", f.run(root, "integrations.plugin_disable", samplePluginAt, nil), "invalid_stage")
	f.unchanged(t, "stop installed", f.run(root, "integrations.plugin_stop", samplePluginAt, nil), "invalid_stage")
	f.ok(t, "propose remove", f.run(root, "integrations.plugin_propose", samplePluginAt, map[string]string{"kind": "remove"}), "pending", "")
	f.ok(t, "approve remove", f.run(second, "integrations.plugin_approve", pluginProposalAt(10), nil), "applied", "")
	v = installedRow(t, f.integrations(root))
	if s := fmt.Sprint(v["state"], " ", v["disposal"], " ", v["disable_blocked"]); s != "removed map[deletion:retain_for_evidence migration:additive namespace:sample-ledger-export-data "+
		"owner:sample-publisher retention_days:30] not_allowed_state" {
		t.Errorf("removed %s", s)
	}
	f.unchanged(t, "disable removed", f.run(root, "integrations.plugin_disable", samplePluginAt, nil), "invalid_stage")
	f.s.integ.plugins = append(f.s.integ.plugins, &pluginRecord{id: "sample-absent"})
	f.unchanged(t, "stop absent", f.run(root, "integrations.plugin_stop", pluginTarget+"sample-absent", nil), "invalid_stage")
	if n := len(part(f.integrations(root), "modules")["installations"].([]any)); n != 1 {
		t.Errorf("absent installation listed %d", n)
	}
}

func TestIntegrationSubscriptionLifecycle(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	payments := subscriptionAt(samplePayments)
	f.unchanged(t, "self approval", f.run(second, "integrations.subscription_approve", payments, nil), "prerequisite_failed")
	f.ok(t, "approve", f.run(root, "integrations.subscription_approve", payments, nil), "active", "map[subscription:sample-payments-feed target:webhook-subscription:sample-payments-feed]")
	f.unchanged(t, "approve active", f.run(root, "integrations.subscription_approve", payments, nil), "invalid_stage")
	f.unchanged(t, "request resume while active", f.run(root, "integrations.subscription_request_resume", payments, nil), "invalid_stage")
	f.ok(t, "pause", f.run(second, "integrations.subscription_pause", payments, nil), "paused", "")
	f.unchanged(t, "pause paused", f.run(second, "integrations.subscription_pause", payments, nil), "invalid_stage")
	f.unchanged(t, "approve resume without a request", f.run(second, "integrations.subscription_approve_resume", payments, nil), "invalid_stage")
	if s := actionsOf(subscriptionRowFor(t, f.integrations(root), samplePayments)); s != "approve=false/not_pending pause=false/not_active request_resume=true/ approve_resume=false/not_pending revoke=true/" {
		t.Errorf("paused %s", s)
	}
	f.ok(t, "request resume", f.run(root, "integrations.subscription_request_resume", payments, nil), "paused", "")
	f.unchanged(t, "repeat request", f.run(root, "integrations.subscription_request_resume", payments, nil), "invalid_stage")
	f.unchanged(t, "takeover request", f.run(second, "integrations.subscription_request_resume", payments, nil), "invalid_stage")
	if v, _ := f.s.webhooks.SubscriptionOf(SampleWorkspace, samplePayments); v.Resumer != SampleAdmin {
		t.Errorf("resumer %q", v.Resumer)
	}
	read := subscriptionRowFor(t, f.integrations(root), samplePayments)
	if s := fmt.Sprint(read["resumer_login"], " ", actionsOf(read)); s != "admin@example.invalid approve=false/not_pending pause=false/not_active request_resume=false/already_requested "+
		"approve_resume=false/own_request revoke=true/" {
		t.Errorf("requested %s", s)
	}
	if s := actionsOf(subscriptionRowFor(t, f.integrations(second), samplePayments)); !strings.Contains(s, "approve_resume=true/") {
		t.Errorf("second may approve resume %s", s)
	}
	f.unchanged(t, "self resume", f.run(root, "integrations.subscription_approve_resume", payments, nil), "prerequisite_failed")
	f.ok(t, "approve resume", f.run(second, "integrations.subscription_approve_resume", payments, nil), "active", "")
	f.ok(t, "revoke", f.run(root, "integrations.subscription_revoke", payments, nil), "revoked", "")
	f.unchanged(t, "revoke revoked", f.run(second, "integrations.subscription_revoke", payments, nil), "invalid_stage")
	f.unchanged(t, "pause revoked", f.run(second, "integrations.subscription_pause", payments, nil), "invalid_stage")
	row := subscriptionRowFor(t, f.integrations(root), samplePayments)
	history := []string{}
	for _, h := range row["history"].([]any) {
		entry := h.(map[string]any)
		history = append(history, fmt.Sprintf("%v:%v:%v", entry["action"], entry["actor"], entry["rationale"]))
	}
	if strings.Join(history, ",") != "approve:sample-admin:sample reason,pause:sample-admin-2:sample reason,request_resume:sample-admin:sample reason,"+
		"approve_resume:sample-admin-2:sample reason,revoke:sample-admin:sample reason" {
		t.Errorf("history %v", history)
	}
	if s := fmt.Sprint(row["state"], " ", row["reason"], " ", actionsOf(row)); s != "revoked subscription_revoked approve=false/not_pending pause=false/not_active "+
		"request_resume=false/not_pending approve_resume=false/not_pending revoke=false/not_pending" {
		t.Errorf("revoked %s", s)
	}
}

func TestIntegrationSubscriptionPropose(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	legacy := map[string]string{"destination": "sample-legacy-webhook", "types": typeInvoice}
	f.ok(t, "propose", f.run(root, "integrations.subscription_propose", webhooksTarget, legacy), "proposed", "map[subscription:subscription-1 target:webhook-subscription:subscription-1]")
	f.ok(t, "duplicate", f.admin(root, "integrations.subscription_propose", webhooksTarget, "again", legacy), "proposed", "map[subscription:subscription-1 target:webhook-subscription:subscription-1]")
	if len(f.s.integ.subscriptions) != 3 || f.s.integ.counter != 1 || f.s.integ.subscription("subscription-1").rationale != "sample reason" {
		t.Errorf("duplicate stored %d %d", len(f.s.integ.subscriptions), f.s.integ.counter)
	}
	f.unchanged(t, "identical from another proposer", f.run(second, "integrations.subscription_propose", webhooksTarget, legacy), "invalid_stage")
	f.unchanged(t, "identical to an active subscription", f.run(root, "integrations.subscription_propose", webhooksTarget,
		map[string]string{"destination": "sample-erp-webhook", "types": typeInvoice}), "invalid_stage")
	paymentsFields := map[string]string{"destination": "sample-payments-webhook", "types": typePayment}
	f.ok(t, "identical proposal by its proposer", f.run(second, "integrations.subscription_propose", webhooksTarget, paymentsFields), "proposed",
		"map[subscription:sample-payments-feed target:webhook-subscription:sample-payments-feed]")
	f.ok(t, "approve payments", f.run(root, "integrations.subscription_approve", subscriptionAt(samplePayments), nil), "active", "")
	f.unchanged(t, "identical to an active subscription by its proposer", f.run(second, "integrations.subscription_propose", webhooksTarget, paymentsFields), "invalid_stage")
	if s := actionsOf(subscriptionRowFor(t, f.integrations(second), "subscription-1")); !strings.HasPrefix(s, "approve=false/destination_inactive") {
		t.Errorf("inactive destination %s", s)
	}
	f.unchanged(t, "approve inactive destination", f.run(second, "integrations.subscription_approve", subscriptionAt("subscription-1"), nil), "prerequisite_failed")
	both := map[string]string{"destination": "sample-erp-webhook", "types": typePayment + "," + typeInvoice}
	f.ok(t, "two types", f.run(second, "integrations.subscription_propose", webhooksTarget, both), "proposed", "map[subscription:subscription-2 target:webhook-subscription:subscription-2]")
	f.s.integ.types = f.s.integ.types[:1]
	if s := actionsOf(subscriptionRowFor(t, f.integrations(root), "subscription-2")); !strings.HasPrefix(s, "approve=false/subscribe_refused") {
		t.Errorf("refused types %s", s)
	}
	f.unchanged(t, "approve refused types", f.run(root, "integrations.subscription_approve", subscriptionAt("subscription-2"), nil), "prerequisite_failed")
	for _, c := range []struct {
		fields map[string]string
		field  string
	}{
		{map[string]string{"destination": "https://example.invalid", "types": typeInvoice}, "destination"},
		{map[string]string{"destination": "sample-erp", "types": typeInvoice}, "destination"},
		{map[string]string{"destination": "sample-erp-webhook", "types": ""}, "types"},
		{map[string]string{"destination": "sample-erp-webhook", "types": typeInvoice + "," + typeInvoice}, "types"},
		{map[string]string{"destination": "sample-erp-webhook", "types": typePayment}, "types"},
		{map[string]string{"destination": "sample-erp-webhook", "types": typeInvoice + ","}, "types"},
		{map[string]string{"destination": "sample-erp-webhook", "types": " " + typeInvoice}, "types"},
	} {
		bad := f.run(root, "integrations.subscription_propose", webhooksTarget, c.fields)
		if bad.status != 422 || bad.body["code"] != "invalid_request" || fmt.Sprint(bad.body["fields"]) != "["+c.field+"]" {
			t.Errorf("%v: %s", c.fields, bad.raw)
		}
	}
	if f.s.integ.counter != 2 {
		t.Errorf("refused proposals took identifiers %d", f.s.integ.counter)
	}
}

func TestIntegrationSubscriptionDependencies(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	payments := subscriptionAt(samplePayments)
	at := ""
	f.s.fault = func(step string) bool { return step == at }
	at = "destinations"
	if s := actionsOf(subscriptionRowFor(t, f.integrations(root), samplePayments)); !strings.HasPrefix(s, "approve=false/dependency_unavailable") {
		t.Errorf("destinations fault %s", s)
	}
	f.code(t, "destinations", f.run(root, "integrations.subscription_approve", payments, nil), 503, "dependency_unavailable")
	at = "permits"
	if s := actionsOf(subscriptionRowFor(t, f.integrations(root), samplePayments)); !strings.HasPrefix(s, "approve=false/dependency_unavailable") {
		t.Errorf("permits fault %s", s)
	}
	f.code(t, "permits", f.run(root, "integrations.subscription_approve", payments, nil), 503, "dependency_unavailable")
	at = "events-bounds"
	f.code(t, "outbox", f.run(root, "integrations.subscription_approve", payments, nil), 503, "dependency_unavailable")
	at = ""
	f.s.fault = func(step string) bool {
		if step == "destinations" {
			f.s.fault = nil
			if _, err := f.s.webhooks.RevokeSubscription(SampleWorkspace, samplePayments, sampleAdminWebhook); err != nil {
				t.Error(err)
			}
		}
		return false
	}
	f.unchanged(t, "stale approval", f.run(root, "integrations.subscription_approve", payments, nil), "invalid_stage")
	if v, _ := f.s.webhooks.SubscriptionOf(SampleWorkspace, samplePayments); v.State != webhooks.SubscriptionRevoked || v.Approver != "" {
		t.Errorf("stale approval applied %+v", v)
	}
	invoices := subscriptionAt(sampleSubscription)
	f.ok(t, "pause", f.run(root, "integrations.subscription_pause", invoices, nil), "paused", "")
	f.ok(t, "request", f.run(root, "integrations.subscription_request_resume", invoices, nil), "paused", "")
	f.s.fault = func(step string) bool {
		if step == "permits" {
			f.s.fault = nil
			if _, err := f.s.webhooks.PauseSubscription(SampleWorkspace, sampleSubscription, sampleAdminWebhook); err == nil {
				t.Error("paused twice")
			}
			if _, err := f.s.webhooks.RevokeSubscription(SampleWorkspace, sampleSubscription, sampleAdminWebhook); err != nil {
				t.Error(err)
			}
		}
		return false
	}
	f.unchanged(t, "stale resume", f.run(second, "integrations.subscription_approve_resume", invoices, nil), "invalid_stage")
	f.s.integ.subscriptions = append(f.s.integ.subscriptions, &subscriptionEntry{id: "sample-purged"})
	f.unchanged(t, "purged subscription", f.run(root, "integrations.subscription_pause", subscriptionAt("sample-purged"), nil), "invalid_stage")
	if n := len(part(f.integrations(root), "channels")["subscriptions"].([]any)); n != 2 {
		t.Errorf("purged subscription listed %d", n)
	}
	f.ok(t, "propose after a purge", f.run(root, "integrations.subscription_propose", webhooksTarget, map[string]string{"destination": "sample-erp-webhook", "types": typePayment}),
		"proposed", "")
}

func TestIntegrationLetterRequests(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	letter := letterAt(sampleLetter)
	f.unchanged(t, "repeat redelivery request", f.run(root, "integrations.letter_request_redeliver", letter, nil), "invalid_stage")
	f.unchanged(t, "takeover redelivery request", f.run(second, "integrations.letter_request_redeliver", letter, nil), "invalid_stage")
	f.unchanged(t, "self redelivery", f.run(root, "integrations.letter_approve_redeliver", letter, nil), "prerequisite_failed")
	f.unchanged(t, "dismiss without a request", f.run(root, "integrations.letter_approve_dismiss", letter, nil), "invalid_stage")
	f.ok(t, "request dismiss", f.run(second, "integrations.letter_request_dismiss", letter, nil), "requested", "map[delivery:delivery-3 target:webhook-letter:delivery-3]")
	f.unchanged(t, "takeover dismiss", f.run(root, "integrations.letter_request_dismiss", letter, nil), "invalid_stage")
	f.unchanged(t, "self dismiss", f.run(second, "integrations.letter_approve_dismiss", letter, nil), "prerequisite_failed")
	if s := fmt.Sprint(f.s.integ.redeliver[sampleLetter], f.s.integ.dismiss[sampleLetter]); s != "{sample-admin Sample data} {sample-admin-2 sample reason}" {
		t.Errorf("requests %s", s)
	}
	row := deliveryRowFor(t, f.integrations(root), sampleLetter)
	if s := actionsOf(row); s != "request_redeliver=false/already_requested approve_redeliver=false/own_request request_dismiss=false/already_requested approve_dismiss=true/" {
		t.Errorf("letter %s", s)
	}
	at := "webhooks-approve_redeliver"
	f.s.fault = func(step string) bool { return step == at }
	f.code(t, "redelivery fault", f.run(second, "integrations.letter_approve_redeliver", letter, nil), 500, "internal_error")
	at = "webhooks-request_dismiss"
	f.s.integ.dismiss = map[string]letterRequest{}
	f.code(t, "dismiss request fault", f.run(root, "integrations.letter_request_dismiss", letter, nil), 500, "internal_error")
	if len(f.s.integ.dismiss) != 0 || len(f.s.integ.subscription(sampleSubscription).letters) != 1 {
		t.Errorf("failed calls changed the mirror %v", f.s.integ.dismiss)
	}
	f.s.fault = nil
	f.s.integ.dismiss[sampleLetter] = letterRequest{by: SampleAdmin2, rationale: "sample reason"}
	got := f.run(second, "integrations.letter_approve_redeliver", letter, nil)
	f.ok(t, "approve redelivery", got, "queued", "map[delivery:delivery-3 redelivery:delivery-7 target:webhook-letter:delivery-3]")
	if len(f.s.integ.redeliver) != 0 || len(f.s.integ.dismiss) != 0 || len(f.s.integ.subscription(sampleSubscription).letters) != 0 {
		t.Errorf("mirror after redelivery %v %v", f.s.integ.redeliver, f.s.integ.dismiss)
	}
	read := f.integrations(root)
	if fresh := deliveryRowFor(t, read, "delivery-7"); fresh["kind"] != "redelivery" || fresh["state"] != "queued" || fresh["event_id"] != "sample-invoice-2" {
		t.Errorf("redelivery %v", fresh)
	}
	f.code(t, "letter gone", f.run(root, "integrations.letter_request_dismiss", letter, nil), 403, "permission_denied")
	f.s.integ.subscription(sampleSubscription).letters = []string{sampleLetter}
	f.unchanged(t, "no longer dead", f.run(root, "integrations.letter_request_dismiss", letter, nil), "invalid_stage")
}

func TestIntegrationLetterDismissAndRevocation(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	letter := letterAt(sampleLetter)
	f.ok(t, "request dismiss", f.run(second, "integrations.letter_request_dismiss", letter, nil), "requested", "")
	f.ok(t, "approve dismiss", f.run(root, "integrations.letter_approve_dismiss", letter, nil), "dismissed", "map[delivery:delivery-3 target:webhook-letter:delivery-3]")
	if len(f.s.integ.redeliver) != 0 || len(f.s.integ.dismiss) != 0 || len(f.s.integ.subscription(sampleSubscription).letters) != 0 {
		t.Errorf("mirror after dismissal")
	}
	for _, d := range part(f.integrations(root), "events")["deliveries"].([]any) {
		if d.(map[string]any)["id"] == sampleLetter {
			t.Error("dismissed letter listed")
		}
	}
	g := setup(t, SeedSample).ready()
	groot, gsecond := g.as(AdminLogin), g.as(Admin2Login)
	g.ok(t, "revoke", g.run(groot, "integrations.subscription_revoke", subscriptionAt(sampleSubscription), nil), "revoked", "")
	read := g.integrations(gsecond)
	if s := actionsOf(deliveryRowFor(t, read, sampleLetter)); !strings.Contains(s, "approve_redeliver=false/revoked") {
		t.Errorf("revoked letter %s", s)
	}
	if keys := subscriptionRowFor(t, read, sampleSubscription)["keys"]; fmt.Sprint(keys) != "[]" {
		t.Errorf("revoked keys %v", keys)
	}
	g.unchanged(t, "redeliver on revoked", g.run(gsecond, "integrations.letter_approve_redeliver", letterAt(sampleLetter), nil), "invalid_stage")
	g.ok(t, "dismiss on revoked", g.run(gsecond, "integrations.letter_request_dismiss", letterAt(sampleLetter), nil), "requested", "")
	g.ok(t, "approve dismiss on revoked", g.run(groot, "integrations.letter_approve_dismiss", letterAt(sampleLetter), nil), "dismissed", "")
}

func TestIntegrationStaleRevisionAndReplay(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	old := f.revision(root)
	if got := f.runWith(root, old, "integrations.plugin_stop", samplePluginAt, map[string]string{}); got.status != 200 {
		t.Fatal(got.raw)
	}
	f.code(t, "stale", f.runWith(second, old, "integrations.subscription_pause", subscriptionAt(sampleSubscription), map[string]string{}), 409, "revision_conflict")
	body := `{"request_id":"req-p","target_id":"` + subscriptionAt(sampleSubscription) + `","idempotency_key":"key-pause","expected_revision":"` + f.revision(second) +
		`","rationale":"r","body":{}}`
	first := f.call("integrations.subscription_pause", body, second.cookie, second.csrf)
	again := f.call("integrations.subscription_pause", body, second.cookie, second.csrf)
	if first.status != 200 || again.status != 200 || again.body["replay"] != true || fmt.Sprint(first.body["fields"]) != fmt.Sprint(again.body["fields"]) {
		t.Fatalf("replay %s %s", first.raw, again.raw)
	}
	if n := len(f.s.integ.subscription(sampleSubscription).history); n != 2 {
		t.Errorf("replay applied twice %d", n)
	}
}

func TestIntegrationsLeakNothing(t *testing.T) {
	f := setup(t, SeedSample)
	var captured []byte
	f.s.fault = func(step string) bool {
		if step == "webhooks-approve_key" {
			for _, key := range f.s.signer.keys {
				captured = append([]byte(nil), key...)
			}
		}
		return false
	}
	f = f.ready()
	f.s.fault = nil
	if len(captured) != 64 {
		t.Fatalf("captured %d", len(captured))
	}
	secrets := leakValues(t, f, captured)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	raws := map[string]string{"read": f.integrations(root).raw, "second read": f.integrations(second).raw}
	steps := []struct {
		who            actor
		command        string
		target         string
		fields         map[string]string
		expectedStatus int
	}{
		{root, "plugin_approve", pluginProposalAt(8), nil, 200},
		{second, "plugin_approve", pluginProposalAt(9), nil, 200},
		{root, "plugin_propose", samplePluginAt, map[string]string{"kind": "rollback", "version": "1.0.0"}, 200},
		{root, "plugin_withdraw", pluginProposalAt(10), nil, 200},
		{root, "plugin_stop", samplePluginAt, nil, 200},
		{root, "plugin_disable", samplePluginAt, nil, 200},
		{root, "subscription_approve", subscriptionAt(samplePayments), nil, 200},
		{root, "subscription_pause", subscriptionAt(samplePayments), nil, 200},
		{root, "subscription_request_resume", subscriptionAt(samplePayments), nil, 200},
		{second, "subscription_approve_resume", subscriptionAt(samplePayments), nil, 200},
		{root, "subscription_propose", webhooksTarget, map[string]string{"destination": "sample-legacy-webhook", "types": typePayment}, 200},
		{second, "letter_request_dismiss", letterAt(sampleLetter), nil, 200},
		{second, "letter_approve_redeliver", letterAt(sampleLetter), nil, 200},
		{root, "subscription_revoke", subscriptionAt(samplePayments), nil, 200},
		{root, "letter_request_redeliver", letterAt(sampleLetter), nil, 403},
		{root, "letter_approve_dismiss", letterAt(sampleLetter), nil, 403},
	}
	for _, c := range steps {
		got := f.run(c.who, integrationCommand+c.command, c.target, c.fields)
		if got.status != c.expectedStatus {
			t.Fatalf("%s: %s", c.command, got.raw)
		}
		raws[c.command] = got.raw
	}
	raws["final read"] = f.integrations(root).raw
	for name, raw := range raws {
		assertNoLeak(t, name, raw, secrets)
	}
	if !strings.Contains(raws["read"], sampleHash("sample-ledger-export-1.0.0")) {
		t.Error("read hides the manifest digest")
	}
}

func TestIntegrationDuplicateNeedsSameTypeSet(t *testing.T) {
	if sameTypes([]string{"a", "b"}, []string{"a"}) || sameTypes([]string{"a"}, []string{"a", "b"}) || !sameTypes([]string{"b", "a"}, []string{"a", "b"}) {
		t.Error("type set comparison")
	}
}

func TestIntegrationModulesStateFromAnyPart(t *testing.T) {
	f := setup(t, SeedNone).ready()
	for name, set := range map[string]func(){
		"workspace":  func() { f.s.integ.workspace = f.s.bootWorkspace },
		"publishers": func() { f.s.integ.publishers = []string{"p"} },
		"plugins":    func() { f.s.integ.plugins = []*pluginRecord{{}} },
	} {
		saved := *f.s.integ
		set()
		out, _ := f.s.modulesOf(SampleAdmin)
		if out.State != stateConfigured {
			t.Errorf("%s state %s", name, out.State)
		}
		*f.s.integ = saved
	}
}
