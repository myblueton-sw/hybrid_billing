package server

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/events"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/plugins"
	"github.com/myblueton-sw/hybrid_billing/modules/webhooks"
)

const (
	sampleLetter   = "delivery-3"
	samplePluginAt = pluginTarget + samplePlugin
)

func (f world) integrations(a actor) reply {
	f.t.Helper()
	return f.do("GET", "/api/v1/integrations", "", func(r *http.Request) {
		if a.cookie != nil {
			r.AddCookie(a.cookie)
		}
		r.Header.Set(csrfHeader, a.csrf)
	})
}

func part(got reply, name string) map[string]any {
	section, _ := got.body[name].(map[string]any)
	return section
}

func rowWith(t testing.TB, got reply, section, list, key, value string) map[string]any {
	t.Helper()
	for _, item := range part(got, section)[list].([]any) {
		if row := item.(map[string]any); fmt.Sprint(row[key]) == value {
			return row
		}
	}
	t.Fatalf("no %s %s=%s in %s", list, key, value, got.raw)
	return nil
}

func pluginProposal(t testing.TB, got reply, number int) map[string]any {
	t.Helper()
	return rowWith(t, got, "modules", "proposals", "number", fmt.Sprint(number))
}

func subscriptionRowFor(t testing.TB, got reply, id string) map[string]any {
	t.Helper()
	return rowWith(t, got, "channels", "subscriptions", "id", id)
}

func deliveryRowFor(t testing.TB, got reply, id string) map[string]any {
	t.Helper()
	return rowWith(t, got, "events", "deliveries", "id", id)
}

func actionsOf(row map[string]any) string {
	out := []string{}
	actions := row["actions"].(map[string]any)
	for _, name := range []string{"approve", "pause", "request_resume", "approve_resume", "revoke", "request_redeliver", "approve_redeliver", "request_dismiss", "approve_dismiss"} {
		if a, ok := actions[name].(map[string]any); ok {
			out = append(out, fmt.Sprintf("%s=%v/%v", name, a["allowed"], a["blocked"]))
		}
	}
	return strings.Join(out, " ")
}

func fieldsFor(kind string) map[string]string {
	switch kind {
	case pluginProposeKind:
		return map[string]string{"kind": "enable"}
	case subProposeKind:
		return map[string]string{"destination": "sample-erp-webhook", "types": typePayment}
	}
	return map[string]string{}
}

func pluginProposalAt(number int) string { return fmt.Sprintf("%s%d", pluginProposalTarget, number) }

func subscriptionAt(id string) string { return subscriptionTarget + id }

func letterAt(id string) string { return letterTarget + id }

func TestIntegrationsReadShape(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	got := f.integrations(root)
	if got.status != 200 || got.body["state"] != "configured" || got.body["now"] != stamp(start) || got.body["revision"] != f.s.Revision() {
		t.Fatalf("read %s", got.raw)
	}
	clients := part(got, "clients")
	if s := fmt.Sprint(clients["state"], " ", clients["manage_on"], " ", clients["oauth_clients"], " ", clients["service_users"]); s != "configured M04 not_connected []" {
		t.Errorf("clients %s", s)
	}
	if s := fmt.Sprint(clients["clients"]); s != "[map[grants:[map[action:view id:reader-console kind:focus_data not_after:2026-12-16T09:00:00Z not_before:2026-11-16T09:00:00Z "+
		"scope:map[contract:all customer:all legal_entity:all period:any] state:active]] id:console kind:first_party_console state:active]]" {
		t.Errorf("console %s", s)
	}
	modules := part(got, "modules")
	if s := fmt.Sprint(modules["state"], " ", modules["workspace"], " ", modules["publishers"]); s != "configured map[id:sample-workspace runtime:sample-plugin-runtime verification:false] "+
		"[map[id:sample-publisher keys:[map[compromise:<nil> not_after:2027-05-15T09:00:00Z not_before:2026-11-16T09:00:00Z retired:false version:sample-plugin-key-1]] name:Sample Publisher revocation:<nil>]]" {
		t.Errorf("modules %s", s)
	}
	versions := modules["plugins"].([]any)
	first := versions[0].(map[string]any)
	if len(versions) != 3 || first["version"] != "1.0.0" || versions[1].(map[string]any)["version"] != "1.1.0" || first["digest"] != sampleHash("sample-ledger-export-1.0.0") {
		t.Fatalf("versions %v", versions)
	}
	licensing := []string{}
	for _, v := range versions {
		row := v.(map[string]any)
		licensing = append(licensing, fmt.Sprint(row["version"], " ", row["license"], " ", row["license_state"]))
	}
	if s := strings.Join(licensing, ","); s != "1.0.0 Apache-2.0 permitted,1.1.0 Apache-2.0 permitted,1.2.0 LicenseRef-sample-proprietary not_permitted" {
		t.Errorf("licenses %s", s)
	}
	if s := fmt.Sprint(modules["licenses"]); s != "map[permitted:[Apache-2.0 BSD-2-Clause BSD-3-Clause ISC LGPL-2.1-only LGPL-3.0-only MIT MPL-2.0 Zlib] source:default]" {
		t.Errorf("license list %s", s)
	}
	if s := fmt.Sprint(first["capabilities"], first["egress"], first["endpoints"], first["timeout_seconds"], first["memory_mib"], first["concurrency"], first["core_min"], first["core_max"],
		first["key_version"], first["publisher"], first["vulnerable"]); s != "[read_events:invoice.issued serve_screen:overview] [sample-erp] [api] 30 128 21.0.01.9.9sample-plugin-key-1sample-publisher<nil>" {
		t.Errorf("manifest %s", s)
	}
	if s := fmt.Sprint(first["storage"], first["screens"]); s != "map[deletion:retain_for_evidence migration:additive namespace:sample-ledger-export-data owner:sample-publisher retention_days:30] "+
		"[map[component:table fields:[map[label:Total name:total source:sample-ledger-export-data] map[label:Event name:event source:invoice.issued]] id:overview label:Overview]]" {
		t.Errorf("storage and screens %s", s)
	}
	if s := fmt.Sprint(modules["installations"]); s != "[map[can_disable:true can_stop:true disable_blocked: disposal:<nil> endpoints:[] plugin:sample-ledger-export reason: revision:2 "+
		"runtime:not_connected state:enabled stop_blocked: tokens:not_issued version:1.0.0]]" {
		t.Errorf("installations %s", s)
	}
	kinds := []string{}
	for _, p := range modules["proposals"].([]any) {
		row := p.(map[string]any)
		kinds = append(kinds, fmt.Sprintf("%v:%v:%v", row["number"], row["kind"], row["state"]))
	}
	if strings.Join(kinds, ",") != "10:version:applied,9:endpoint:pending,8:upgrade:pending,7:enable:applied,6:install:applied,5:workspace:applied,4:version:applied,3:version:applied,2:key:applied,1:publisher:applied" {
		t.Errorf("proposals %v", kinds)
	}
	endpoint, upgrade := pluginProposal(t, got, 9), pluginProposal(t, got, 8)
	if s := blockedAs(endpoint) + "|" + blockedAs(upgrade) + "|" + blockedAs(pluginProposal(t, got, 2)); s != "false own_proposal true |true  false not_proposer|false not_pending false not_pending" {
		t.Errorf("plugin blockers %s", s)
	}
	if s := fmt.Sprint(endpoint["summary"]); s != "map[endpoint:api fingerprint:"+sampleHash("sample-certificate-api")+" key_not_after: key_not_before: key_version: plugin:sample-ledger-export "+
		"publisher: publisher_name: runtime: version: workspace:sample-workspace]" {
		t.Errorf("endpoint summary %s", s)
	}
	key := pluginProposal(t, got, 2)
	if s := fmt.Sprint(key["summary"], key["approver_login"], key["closed_by"], key["closing_rationale"], key["target"]); s != "map[endpoint: fingerprint: key_not_after:2027-05-15T09:00:00Z "+
		"key_not_before:2026-11-16T09:00:00Z key_version:sample-plugin-key-1 plugin: publisher:sample-publisher publisher_name: runtime: version: workspace:]admin2@example.invalidsample-admin-2Sample dataplugin-proposal-2" {
		t.Errorf("key %s", s)
	}
	channels := part(got, "channels")
	if s := fmt.Sprint(channels["state"], channels["subscribers"], channels["destinations"]); s != "configured[sample-erp-connector] [map[id:sample-erp-webhook source:sample_allowlist state:active] "+
		"map[id:sample-legacy-webhook source:sample_allowlist state:inactive] map[id:sample-payments-webhook source:sample_allowlist state:active]]" {
		t.Errorf("channels %s", s)
	}
	invoices := subscriptionRowFor(t, got, sampleSubscription)
	if s := fmt.Sprint(invoices["state"], invoices["cursor"], invoices["lag"], invoices["types"], invoices["destination"], invoices["proposer_login"], invoices["approver_login"], invoices["signer"],
		invoices["history"], invoices["target"]); s != "active3 0 [invoice.issued]sample-erp-webhookadmin@example.invalidadmin2@example.invalidnot_connected"+
		"[map[action:approve actor:sample-admin-2 actor_login:admin2@example.invalid rationale:Sample data reason: request:]]webhook-subscription:sample-erp-invoices" {
		t.Errorf("invoices %s", s)
	}
	if s := fmt.Sprint(invoices["keys"]); s != "[map[approver:sample-admin-2 approver_login:admin2@example.invalid not_after:2027-05-15T09:00:00Z not_before:2026-11-16T09:00:00Z "+
		"proposer:sample-admin proposer_login:admin@example.invalid state:active version:sample-webhook-key-1]]" {
		t.Errorf("keys %s", s)
	}
	if s := actionsOf(invoices); s != "approve=false/not_pending pause=true/ request_resume=false/not_pending approve_resume=false/not_pending revoke=true/" {
		t.Errorf("invoice actions %s", s)
	}
	payments := subscriptionRowFor(t, got, samplePayments)
	if s := fmt.Sprint(payments["state"], " ", payments["proposer"], " ", payments["keys"], " ", actionsOf(payments)); s != "proposed sample-admin-2 [] approve=true/ pause=false/not_active "+
		"request_resume=false/not_pending approve_resume=false/not_pending revoke=true/" {
		t.Errorf("payments %s", s)
	}
	feed := part(got, "events")
	if s := fmt.Sprint(feed["state"], feed["outbox"], feed["dispatcher"], feed["types"]); s != "configuredmap[floor:0 head:3]not_connected[map[approver:sample-admin-2 approver_login:admin2@example.invalid "+
		"fields:[invoice_id] name:invoice.issued proposer:sample-admin proposer_login:admin@example.invalid schema_version:1 state:active] map[approver:sample-admin-2 approver_login:admin2@example.invalid "+
		"fields:[payment_id] name:payment.received proposer:sample-admin proposer_login:admin@example.invalid schema_version:1 state:active]]" {
		t.Errorf("events %s", s)
	}
	if n := len(feed["deliveries"].([]any)); n != 2 {
		t.Fatalf("deliveries %d", n)
	}
	dead := deliveryRowFor(t, got, sampleLetter)
	if s := fmt.Sprint(dead["kind"], dead["state"], dead["reason"], dead["sequence"], dead["event_id"], dead["source"], dead["redeliver_request"], dead["dismiss_request"], dead["target"]); s !=
		"deadfailedrejected2sample-invoice-2urn:sample-billingmap[by:sample-admin by_login:admin@example.invalid id:letter-request-1 rationale:Sample data target:webhook-letter-request:letter-request-1] <nil>webhook-letter:delivery-3" {
		t.Errorf("dead letter %s", s)
	}
	if s := actionsOf(dead); s != "request_redeliver=false/already_requested approve_redeliver=false/own_request request_dismiss=true/ approve_dismiss=false/not_requested" {
		t.Errorf("letter actions %s", s)
	}
	waiting := deliveryRowFor(t, got, "delivery-5")
	if s := fmt.Sprint(waiting["kind"], waiting["state"], waiting["attempts"], waiting["outcome_unknown"], " ", actionsOf(waiting)); s != "openwaiting1 false request_redeliver=false/not_dead "+
		"approve_redeliver=false/not_dead request_dismiss=false/not_dead approve_dismiss=false/not_dead" {
		t.Errorf("waiting %s", s)
	}
	second := f.integrations(f.as(Admin2Login))
	if s := blockedAs(pluginProposal(t, second, 9)) + "|" + blockedAs(pluginProposal(t, second, 8)); s != "true  false not_proposer|false own_proposal true " {
		t.Errorf("second plugin blockers %s", s)
	}
	if s := actionsOf(subscriptionRowFor(t, second, samplePayments)); !strings.HasPrefix(s, "approve=false/own_proposal") {
		t.Errorf("second payments %s", s)
	}
	if s := actionsOf(deliveryRowFor(t, second, sampleLetter)); s != "request_redeliver=false/already_requested approve_redeliver=true/ request_dismiss=true/ approve_dismiss=false/not_requested" {
		t.Errorf("second letter %s", s)
	}
}

func TestIntegrationsReadServiceUsers(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	id := f.newService(root, "ledger-feed")
	f.issued(root, id, "10")
	f.issued(root, id, "10")
	got := f.integrations(root)
	if s := fmt.Sprint(part(got, "clients")["service_users"]); s != "[map[id:svc-ledger-feed owner:sample-admin owner_login:admin@example.invalid state:active usable_credentials:2]]" {
		t.Errorf("service users %s", s)
	}
	if strings.Contains(got.raw, "hbsk_") || strings.Contains(got.raw, "credential\"") {
		t.Errorf("service users carry credentials %s", got.raw)
	}
	f.advance(11 * 24 * time.Hour)
	if s := fmt.Sprint(part(f.integrations(f.as(AdminLogin)), "clients")["service_users"]); !strings.Contains(s, "usable_credentials:0") {
		t.Errorf("expired credentials %s", s)
	}
}

func TestIntegrationsReadWithoutSeed(t *testing.T) {
	none := setup(t, SeedNone).ready()
	tok := none.links()[""]
	if got := none.auth("redeem", `{"token":"`+tok+`","password":"`+pw+`","login":"first@example.invalid","email":"first@example.invalid"}`, none.pre()); got.status != 204 {
		t.Fatalf("activate %+v", got)
	}
	cookie, csrf, err := none.s.openFor(LoginTriple("first@example.invalid"), EmptyWorkspace, false)
	if err != nil {
		t.Fatal(err)
	}
	a := actor{cookie, csrf}
	got := none.integrations(a)
	if got.status != 200 || got.body["state"] != "configured" || part(got, "clients")["state"] != "configured" || len(part(got, "clients")["clients"].([]any)) != 1 {
		t.Fatalf("empty %s", got.raw)
	}
	for _, name := range []string{"modules", "channels", "events"} {
		if part(got, name)["state"] != "not_configured" {
			t.Errorf("%s %v", name, part(got, name))
		}
	}
	if s := fmt.Sprint(part(got, "modules")["workspace"], part(got, "modules")["proposals"], part(got, "channels")["subscriptions"], part(got, "events")["deliveries"]); s != "<nil> [] [] []" {
		t.Errorf("empty sections %s", s)
	}
	if !strings.Contains(fmt.Sprint(part(got, "clients")["clients"]), "grants:[]") {
		t.Errorf("console grants hide installation administration %s", got.raw)
	}
	for _, c := range []struct{ command, target string }{{"integrations.subscription_propose", webhooksTarget}, {"integrations.plugin_propose", samplePluginAt}} {
		none.code(t, c.command, none.runWith(a, none.revision(a), c.command, c.target, fieldsFor(strings.TrimPrefix(c.command, integrationCommand))), 403, "permission_denied")
	}
}

func TestIntegrationsAccess(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	member, root := f.as(MemberLogin), f.as(AdminLogin)
	f.code(t, "member read", f.integrations(member), 403, "permission_denied")
	f.code(t, "anonymous", f.integrations(actor{}), 401, "unauthenticated")
	commandTargets := map[string]string{pluginApproveKind: pluginProposalAt(8), pluginWithdrawKind: pluginProposalAt(9), pluginProposeKind: samplePluginAt,
		pluginStopKind: samplePluginAt, pluginDisableKind: samplePluginAt, subProposeKind: webhooksTarget, subApproveKind: subscriptionAt(samplePayments),
		subPauseKind: subscriptionAt(sampleSubscription), subRequestKind: subscriptionAt(sampleSubscription), subResumeKind: subscriptionAt(sampleSubscription),
		subRevokeKind: subscriptionAt(sampleSubscription), letterRedeliverKind: letterAt(sampleLetter), letterAgainKind: letterAt(sampleLetter),
		letterDismissKind: letterAt(sampleLetter), letterDropKind: letterAt(sampleLetter)}
	before := f.s.Revision()
	for kind, target := range commandTargets {
		f.code(t, "member "+kind, f.runWith(member, f.revision(root), integrationCommand+kind, target, fieldsFor(kind)), 403, "permission_denied")
		h := integrationHandler{s: f.s, kind: kind}
		if _, res := h.Resolve("other-ws", target); res != commands.NotFound {
			t.Errorf("%s resolved in another workspace", kind)
		}
		if _, res := h.Resolve(SampleWorkspace, target); res != commands.Found {
			t.Errorf("%s did not resolve %s", kind, target)
		}
	}
	f.s.bootWorkspace = "elsewhere"
	if got := f.integrations(root); got.status != 200 || got.body["state"] != "not_configured" || got.body["clients"] != nil {
		t.Errorf("other workspace %s", got.raw)
	}
	for kind, target := range commandTargets {
		f.code(t, "other workspace "+kind, f.runWith(root, f.s.Revision(), integrationCommand+kind, target, fieldsFor(kind)), 403, "permission_denied")
	}
	f.s.bootWorkspace = SampleWorkspace
	if f.s.Revision() != before {
		t.Errorf("refused commands moved the revision %s %s", before, f.s.Revision())
	}
	for _, c := range []struct{ command, target string }{
		{"integrations.plugin_approve", "plugin-proposal-08"}, {"integrations.plugin_approve", "plugin-proposal-x"}, {"integrations.plugin_approve", "plugin-proposal-99"},
		{"integrations.plugin_approve", "8"}, {"integrations.plugin_withdraw", "plugin-proposal-"}, {"integrations.plugin_stop", "plugin:other"},
		{"integrations.plugin_stop", samplePlugin}, {"integrations.subscription_pause", "webhook-subscription:other"}, {"integrations.subscription_pause", sampleSubscription},
		{"integrations.letter_request_dismiss", "webhook-letter:delivery-5"}, {"integrations.letter_request_dismiss", sampleLetter},
		{"integrations.subscription_propose", "webhook"}, {"integrations.plugin_propose", webhooksTarget},
	} {
		f.code(t, c.command+" "+c.target, f.run(root, c.command, c.target, fieldsFor(strings.TrimPrefix(c.command, integrationCommand))), 403, "permission_denied")
	}
	long := strings.Repeat("r", maxRationale+1)
	f.code(t, "long reason", f.admin(root, "integrations.plugin_approve", pluginProposalAt(8), long, map[string]string{}), 422, "invalid_request")
	f.code(t, "no reason", f.admin(root, "integrations.plugin_approve", pluginProposalAt(8), "", map[string]string{}), 422, "invalid_request")
	f.code(t, "unknown kind", f.run(root, "integrations.plugin_propose", samplePluginAt, map[string]string{"kind": "install_all"}), 422, "invalid_request")
	f.s.integ.types = nil
	f.code(t, "no approved types", f.run(root, "integrations.subscription_propose", webhooksTarget, fieldsFor(subProposeKind)), 403, "permission_denied")
}

func leakValues(t *testing.T, f world, captured []byte) []string {
	t.Helper()
	secrets := []string{}
	add := func(b []byte) {
		if len(b) > 0 {
			secrets = append(secrets, fmt.Sprintf("%x", b), base64.StdEncoding.EncodeToString(b), base64.RawURLEncoding.EncodeToString(b), string(b))
		}
	}
	p, err := f.s.plugins.Publisher(samplePublisher)
	if err != nil {
		t.Fatal(err)
	}
	add(p.Keys[0].Public)
	for _, version := range []string{"1.0.0", "1.1.0", "1.2.0"} {
		v, err := f.s.plugins.Version(samplePlugin, version)
		if err != nil {
			t.Fatal(err)
		}
		add(v.Signature)
	}
	proof, err := f.s.plugins.Proposal(2)
	if err != nil {
		t.Fatal(err)
	}
	add(proof.Change.Key.Proof)
	for _, e := range f.s.integ.entries {
		add(e.view.Change.Key.Public)
		add(e.view.Change.Signature)
	}
	if len(f.s.integ.subscriptions) > 0 {
		keys, err := f.s.webhooks.Keys(SampleWorkspace, sampleSubscription, sampleAdminWebhook)
		if err != nil || len(keys) != 1 {
			t.Fatalf("webhook keys %v %v", keys, err)
		}
		add(keys[0].Key.Public)
	}
	add(captured)
	page, err := f.s.events.Read(SampleWorkspace, 0, 10, events.Actor{ID: "leak-probe", Kind: events.KindSystem})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range page.Events {
		add(events.Body(e))
	}
	return secrets
}

func assertNoLeak(t *testing.T, name, raw string, secrets []string) {
	t.Helper()
	for _, secret := range secrets {
		if strings.Contains(raw, secret) {
			t.Errorf("%s carries %q", name, secret)
		}
	}
	for _, word := range []string{"public", "proof", "signature", "private", "lease-", "hbp_", "specversion", "webhook-key-version", "webhook-time", "\"body\"", "token\""} {
		if strings.Contains(raw, word) {
			t.Errorf("%s names %s", name, word)
		}
	}
}

func TestIntegrationsReadMarksAndWorkspace(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	if _, err := f.s.plugins.MarkVulnerable(samplePlugin, "1.0.0", "sample finding", sampleAdminPlugin); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.plugins.ConfirmMark(plugins.Target{Kind: plugins.MarkVulnerable, Plugin: samplePlugin, Version: "1.0.0"}, sampleAdmin2Plugin); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.plugins.CompromiseKey(samplePublisher, samplePluginKey, "sample compromise", sampleAdminPlugin); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.plugins.RevokePublisher(samplePublisher, "sample revocation", sampleAdmin2Plugin); err != nil {
		t.Fatal(err)
	}
	got := f.integrations(root)
	modules := part(got, "modules")
	if s := fmt.Sprint(modules["plugins"].([]any)[0].(map[string]any)["vulnerable"]); s != "map[by:sample-admin by_login:admin@example.invalid confirmed_by:sample-admin-2 "+
		"confirmed_by_login:admin2@example.invalid reason:sample finding]" {
		t.Errorf("vulnerable %s", s)
	}
	publisher := modules["publishers"].([]any)[0].(map[string]any)
	if s := fmt.Sprint(publisher["revocation"], " ", publisher["keys"]); s != "map[by:sample-admin-2 by_login:admin2@example.invalid confirmed_by: confirmed_by_login: reason:sample revocation] "+
		"[map[compromise:map[by:sample-admin by_login:admin@example.invalid confirmed_by: confirmed_by_login: reason:sample compromise] not_after:2027-05-15T09:00:00Z "+
		"not_before:2026-11-16T09:00:00Z retired:true version:sample-plugin-key-1]]" {
		t.Errorf("publisher %s", s)
	}
	if v := installedRow(t, got); v["state"] != "stopped" || v["reason"] != "vulnerable" {
		t.Errorf("revoked installation %v", v)
	}
	if s := blockedAs(pluginProposal(t, got, 8)); s != "false revoked false not_proposer" {
		t.Errorf("upgrade under a revoked publisher %s", s)
	}
	if s := blockedAs(pluginProposal(t, f.integrations(f.as(Admin2Login)), 9)); s != "true  false not_proposer" {
		t.Errorf("endpoint %s", s)
	}
	f.s.integ.workspace = ""
	got = f.integrations(root)
	if s := fmt.Sprint(part(got, "modules")["workspace"], part(got, "modules")["installations"], part(got, "modules")["state"]); s != "<nil> []configured" {
		t.Errorf("no workspace %s", s)
	}
}

func TestIntegrationsReadServiceUserFault(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.newService(root, "ledger-feed")
	f.newService(root, "ledger-audit")
	f.s.fault = func(step string) bool { return step == "credentials" }
	f.code(t, "credentials", f.integrations(root), 503, "dependency_unavailable")
}

func TestIntegrationsReadGaps(t *testing.T) {
	f := setup(t, SeedSample).ready()
	billing := events.Actor{ID: "gap-probe", Kind: events.KindSystem}
	relay := webhooks.Actor{ID: "gap-relay", Kind: webhooks.KindSystem}
	publish := func(id string) {
		e := events.Event{ID: id, Source: sampleSource, Type: typePayment, SchemaVersion: schemaVersion, Data: map[string]string{"payment_id": id}}
		if _, err := f.s.events.Publish(SampleWorkspace, e, billing); err != nil {
			t.Fatal(err)
		}
	}
	f.ok(t, "approve payments", f.run(f.as(AdminLogin), "integrations.subscription_approve", subscriptionAt(samplePayments), nil), "active", "")
	publish("sample-payment-1")
	f.advance(31 * 24 * time.Hour)
	publish("sample-payment-2")
	if _, err := f.s.webhooks.Dispatch(SampleWorkspace, 10, relay); err != nil {
		t.Fatal(err)
	}
	now := f.s.clock()
	terms := gate.Terms{Action: gate.AdministerInstallation, Kind: gate.InstallationKind, Period: gate.AnyPeriod(),
		Scope: gate.Scope{WorkspaceID: SampleWorkspace, Customer: gate.All(), LegalEntity: gate.All(), Contract: gate.All()}}
	for _, g := range []gate.Grant{{ID: "late-console", ClientID: ConsoleClient}, {ID: "late-" + SampleAdmin, SubjectID: SampleAdmin}, {ID: "late-" + SampleAdmin2, SubjectID: SampleAdmin2}} {
		g.Terms, g.Validity = terms, gate.Validity{NotBefore: now, NotAfter: now.Add(time.Hour)}
		if err := f.s.admin.AddGrant(g); err != nil {
			t.Fatal(err)
		}
	}
	for _, right := range gate.IntegrationRights() {
		lateTerms := terms
		lateTerms.Action = right
		for _, g := range []gate.Grant{{ID: "late-console-" + string(right), ClientID: ConsoleClient}, {ID: "late-" + string(right) + "-" + SampleAdmin, SubjectID: SampleAdmin},
			{ID: "late-" + string(right) + "-" + SampleAdmin2, SubjectID: SampleAdmin2}} {
			g.Terms, g.Validity = lateTerms, gate.Validity{NotBefore: now, NotAfter: now.Add(time.Hour)}
			if err := f.s.admin.AddGrant(g); err != nil {
				t.Fatal(err)
			}
		}
	}
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	row := subscriptionRowFor(t, f.integrations(root), samplePayments)
	if s := fmt.Sprint(row["state"], " ", row["reason"], " ", actionsOf(row)); s != "suspended gap approve=false/not_pending pause=false/not_active request_resume=true/ "+
		"approve_resume=false/not_pending revoke=true/" {
		t.Errorf("suspended %s", s)
	}
	f.ok(t, "request resume", f.run(root, "integrations.subscription_request_resume", subscriptionAt(samplePayments), nil), "suspended", "")
	f.ok(t, "approve resume", f.run(second, "integrations.subscription_approve_resume", subscriptionAt(samplePayments), nil), "active", "")
	row = subscriptionRowFor(t, f.integrations(root), samplePayments)
	if s := fmt.Sprint(row["gaps"], " ", row["cursor"], " ", row["lag"]); s != "[map[after:3 to:4]] 4 1" {
		t.Errorf("gaps %s", s)
	}
}
