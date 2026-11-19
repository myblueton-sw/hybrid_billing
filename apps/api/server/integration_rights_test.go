package server

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func fullViewer(s *Server, user string) *viewer {
	v := &viewer{s: s, id: user, rights: map[gate.Action]bool{}, now: s.clock(), memo: map[string]standing{}}
	for _, right := range gate.IntegrationRights() {
		v.rights[right] = true
	}
	return v
}

var wantRights = map[string]string{
	"plugin_approve": "approve_modules", "plugin_withdraw": "administer_modules", "plugin_propose": "administer_modules", "plugin_stop": "stop_modules",
	"plugin_disable": "stop_modules", "subscription_propose": "administer_channels", "subscription_approve": "approve_channels", "subscription_pause": "stop_channels",
	"subscription_request_resume": "administer_channels", "subscription_approve_resume": "approve_channels", "subscription_revoke": "stop_channels",
	"subscription_withdraw": "administer_channels", "subscription_withdraw_resume": "administer_channels", "letter_request_redeliver": "reprocess_events",
	"letter_approve_redeliver": "reprocess_events", "letter_request_dismiss": "reprocess_events", "letter_approve_dismiss": "reprocess_events",
	"letter_withdraw_redeliver": "reprocess_events", "letter_withdraw_dismiss": "reprocess_events", "redelivery_reapprove": "reprocess_events",
}

var wantAuthor = map[string]bool{"plugin_withdraw": true, "subscription_withdraw": true, "subscription_withdraw_resume": true, "letter_withdraw_redeliver": true,
	"letter_withdraw_dismiss": true}

func TestIntegrationDefinitionsCarryTheirRights(t *testing.T) {
	s := freshInstaller(t)
	seen := map[string]bool{}
	for _, d := range s.integrationDefinitions() {
		kind := strings.TrimPrefix(d.ID, integrationCommand)
		seen[kind] = true
		if string(d.Permission.Action) != wantRights[kind] || d.Permission.Kind != gate.InstallationKind || d.Author != wantAuthor[kind] || !d.Mutating || !d.Rationale {
			t.Errorf("%s: %+v author %v", d.ID, d.Permission, d.Author)
		}
		if d.Permission.Action == gate.AdministerInstallation {
			t.Errorf("%s still uses installation administration", d.ID)
		}
	}
	if len(seen) != len(wantRights) || len(seen) != 20 {
		t.Errorf("definitions %v", seen)
	}
	f := setup(t, SeedSample).ready()
	for _, e := range f.s.commands.Catalogue() {
		integration := strings.HasPrefix(e.ID, integrationCommand)
		for _, right := range gate.IntegrationRights() {
			if e.Permission.Action == right && !integration {
				t.Errorf("%s accepts the integration right %s", e.ID, right)
			}
		}
		if integration && string(e.Permission.Action) != wantRights[strings.TrimPrefix(e.ID, integrationCommand)] {
			t.Errorf("catalogue %s %s", e.ID, e.Permission.Action)
		}
	}
	defer func() {
		if recover() == nil {
			t.Error("a command without a right built a definition")
		}
	}()
	rightOf("integrations_unknown")
}

func (f world) setRights(t testing.TB, user string, rights ...gate.Action) {
	t.Helper()
	keep := map[gate.Action]bool{}
	for _, r := range rights {
		keep[r] = true
	}
	held := map[gate.Action]bool{}
	for _, g := range f.s.admin.Grants() {
		if g.SubjectID != user || g.Closed || !gate.InstallationAction(g.Terms.Action) {
			continue
		}
		if keep[g.Terms.Action] && !held[g.Terms.Action] {
			held[g.Terms.Action] = true
			continue
		}
		if err := f.s.admin.RevokeGrant(g.ID, g.Version); err != nil {
			t.Fatal(err)
		}
	}
	now := f.s.clock()
	scope := gate.Scope{WorkspaceID: SampleWorkspace, Customer: gate.All(), LegalEntity: gate.All(), Contract: gate.All()}
	for _, right := range rights {
		terms := gate.Terms{Action: right, Kind: gate.InstallationKind, Period: gate.AnyPeriod(), Scope: scope}
		_ = f.s.admin.AddGrant(gate.Grant{ID: "test-console-" + string(right), ClientID: ConsoleClient, Terms: terms, Validity: gate.Validity{NotBefore: now, NotAfter: now.Add(time.Hour)}})
		if held[right] {
			continue
		}
		id := fmt.Sprintf("test-%s-%s-%d", right, user, keys.Add(1))
		if err := f.s.admin.AddGrant(gate.Grant{ID: id, SubjectID: user, Terms: terms, Validity: gate.Validity{NotBefore: now, NotAfter: now.Add(time.Hour)}}); err != nil {
			t.Fatal(err)
		}
	}
}

func matrixTarget(kind string) string {
	switch {
	case kind == pluginApproveKind:
		return pluginProposalAt(8)
	case strings.HasPrefix(kind, "plugin_"):
		return samplePluginAt
	case kind == subProposeKind:
		return webhooksTarget
	case kind == subApproveKind:
		return subscriptionAt(samplePayments)
	case strings.HasPrefix(kind, "subscription_"):
		return subscriptionAt(sampleSubscription)
	}
	return letterAt(sampleLetter)
}

func profiles(kind string) map[string][]gate.Action {
	out := map[string][]gate.Action{
		"installation administration": {gate.AdministerInstallation},
		"trust and bulk":              {gate.TrustPropose, gate.TrustValidate, gate.TrustApprove, gate.TrustActivate, gate.RevokeAccessInBulk},
		"nothing":                     {},
	}
	others := []gate.Action{gate.AdministerInstallation, gate.TrustPropose, gate.TrustValidate, gate.TrustApprove, gate.TrustActivate, gate.RevokeAccessInBulk}
	for _, right := range gate.IntegrationRights() {
		out["only "+string(right)] = []gate.Action{right}
		if string(right) != wantRights[kind] {
			others = append(others, right)
		}
	}
	out["every other right"] = others
	return out
}

func TestEveryIntegrationCommandNeedsExactlyItsRight(t *testing.T) {
	kinds := []string{}
	for kind := range wantRights {
		if !wantAuthor[kind] && kind != reapproveKind {
			kinds = append(kinds, kind)
		}
	}
	sort.Strings(kinds)
	cases := 0
	for _, kind := range kinds {
		for name, rights := range profiles(kind) {
			f := setup(t, SeedSample).ready()
			f.setRights(t, SampleAdmin3, rights...)
			got := f.runWith(f.as(Admin3Login), f.s.Revision(), integrationCommand+kind, matrixTarget(kind), fieldsFor(kind))
			holds := false
			for _, r := range rights {
				holds = holds || string(r) == wantRights[kind]
			}
			denied := got.status == 403 && got.body["code"] == string(commands.PermissionDenied)
			if denied == holds {
				t.Errorf("%s with %s: %d %s", kind, name, got.status, got.raw)
			}
			cases++
		}
	}
	if cases != 14*12 {
		t.Errorf("%d cases", cases)
	}
}

func TestTheRightNeedsBothThePersonAndTheConsoleGrant(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setRights(t, SampleAdmin3, gate.ApproveModules)
	for _, g := range f.s.admin.Grants() {
		if g.ClientID == ConsoleClient && g.Terms.Action == gate.ApproveModules && !g.Closed {
			if err := f.s.admin.RevokeGrant(g.ID, g.Version); err != nil {
				t.Fatal(err)
			}
		}
	}
	f.code(t, "person without client", f.runWith(f.as(Admin3Login), f.s.Revision(), integrationCommand+pluginApproveKind, pluginProposalAt(8), map[string]string{}), 403, "permission_denied")
	g := setup(t, SeedSample).ready()
	g.setRights(t, SampleAdmin3)
	g.code(t, "client without person", g.runWith(g.as(Admin3Login), g.s.Revision(), integrationCommand+pluginApproveKind, pluginProposalAt(8), map[string]string{}), 403, "permission_denied")
}

func areaState(got reply, name string) string {
	section := part(got, name)
	if len(section) == 1 && section["state"] == stateNotPermitted {
		return "withheld"
	}
	if section["state"] == stateNotPermitted {
		return "leaky"
	}
	return "shown"
}

func rightsText(got reply) string {
	m, _ := got.body["rights"].(map[string]any)
	out := []string{}
	for _, right := range gate.IntegrationRights() {
		if m[string(right)] == true {
			out = append(out, string(right))
		}
	}
	return strings.Join(out, ",")
}

func noRightOn(row map[string]any, key string) bool {
	a, _ := row["actions"].(map[string]any)[key].(map[string]any)
	return a != nil && a["blocked"] == blockedNoRight && a["allowed"] == false
}

func TestTheReadShowsEachAreaOnlyToItsRights(t *testing.T) {
	set := func(rights ...gate.Action) []gate.Action { return rights }
	cases := map[string][]gate.Action{
		"api":            set(gate.AdministerAPIClients),
		"modules pair":   set(gate.AdministerModules, gate.ApproveModules),
		"channels pair":  set(gate.ApproveChannels, gate.StopChannels),
		"cross pair":     set(gate.StopModules, gate.ReprocessEvents),
		"operator":       operatorRights,
		"all":            gate.IntegrationRights(),
		"administration": set(gate.AdministerInstallation),
	}
	for _, right := range gate.IntegrationRights() {
		cases["only "+string(right)] = set(right)
	}
	for name, rights := range cases {
		f := setup(t, SeedSample).ready()
		f.setRights(t, SampleAdmin3, rights...)
		has := map[gate.Action]bool{}
		for _, r := range rights {
			has[r] = true
		}
		got := f.integrations(f.as(Admin3Login))
		if got.status != 200 {
			t.Fatalf("%s: %s", name, got.raw)
		}
		held := []string{}
		for _, r := range gate.IntegrationRights() {
			if has[r] {
				held = append(held, string(r))
			}
		}
		if rightsText(got) != strings.Join(held, ",") {
			t.Errorf("%s rights %s", name, rightsText(got))
		}
		anyOf := func(list []gate.Action) bool {
			for _, r := range list {
				if has[r] {
					return true
				}
			}
			return false
		}
		want := map[string]bool{"clients": has[gate.AdministerAPIClients], "modules": anyOf(moduleRights), "channels": anyOf(channelRights), "events": anyOf(channelRights)}
		for area, shown := range want {
			if s := areaState(got, area); (s == "shown") != shown || s == "leaky" {
				t.Errorf("%s %s %s", name, area, s)
			}
		}
		if want["modules"] {
			install := installedRow(t, got)
			if (install["stop_blocked"] == blockedNoRight) == has[gate.StopModules] || (install["disable_blocked"] == blockedNoRight) == has[gate.StopModules] {
				t.Errorf("%s install %v", name, install)
			}
			if (pluginProposal(t, got, 8)["approve_blocked"] == blockedNoRight) == has[gate.ApproveModules] {
				t.Errorf("%s proposal %v", name, pluginProposal(t, got, 8)["approve_blocked"])
			}
			if noRightOn(part(got, "modules"), "propose") == has[gate.AdministerModules] {
				t.Errorf("%s propose", name)
			}
		}
		if want["channels"] {
			row := subscriptionRowFor(t, got, sampleSubscription)
			for key, right := range map[string]gate.Action{"approve": gate.ApproveChannels, "pause": gate.StopChannels, "request_resume": gate.AdministerChannels,
				"approve_resume": gate.ApproveChannels, "revoke": gate.StopChannels} {
				if noRightOn(row, key) == has[right] {
					t.Errorf("%s subscription %s", name, key)
				}
			}
			letter := deliveryRowFor(t, got, sampleLetter)
			for _, key := range []string{"request_redeliver", "approve_redeliver", "request_dismiss", "approve_dismiss"} {
				if noRightOn(letter, key) == has[gate.ReprocessEvents] {
					t.Errorf("%s letter %s", name, key)
				}
			}
			if noRightOn(part(got, "channels"), "propose") == has[gate.AdministerChannels] {
				t.Errorf("%s subscription propose", name)
			}
		}
	}
	f := setup(t, SeedSample).ready()
	f.setRights(t, SampleAdmin3)
	f.code(t, "no right at all", f.integrations(f.as(Admin3Login)), 403, "permission_denied")
}

func TestAdmissionDoesNotChangeTheAreas(t *testing.T) {
	read := func(rights ...gate.Action) reply {
		f := setup(t, SeedSample).ready()
		f.setRights(t, SampleAdmin3, rights...)
		return f.integrations(f.as(Admin3Login))
	}
	with := read(gate.AdministerInstallation, gate.StopModules)
	without := read(gate.StopModules)
	for _, area := range []string{"clients", "modules", "channels", "events"} {
		if areaState(with, area) != areaState(without, area) {
			t.Errorf("%s differs by admission", area)
		}
	}
	if rightsText(with) != rightsText(without) || fmt.Sprint(installedRow(t, with)) != fmt.Sprint(installedRow(t, without)) {
		t.Errorf("admission changed the rights or the flags")
	}
	f := setup(t, SeedSample).ready()
	f.setRights(t, SampleAdmin3, gate.StopModules)
	before := f.integrations(f.as(Admin3Login))
	f.s.integ.entry(8).view.Proposer = SampleAdmin3
	after := f.integrations(f.as(Admin3Login))
	if rightsText(before) != rightsText(after) || areaState(before, "clients") != areaState(after, "clients") {
		t.Errorf("own pending items changed the rights or areas")
	}
	if n := len(after.body["own_pending"].([]any)); n != 1 {
		t.Errorf("own pending %d", n)
	}
}

func TestAnUnavailableDecisionFailsTheWholeRead(t *testing.T) {
	for _, right := range append(gate.IntegrationRights(), gate.AdministerInstallation) {
		f := setup(t, SeedSample).ready()
		f.s.fault = func(step string) bool { return step == "integrations-authority-"+string(right) }
		got := f.integrations(f.as(AdminLogin))
		f.code(t, string(right), got, 503, "authority_unavailable")
		if _, present := got.body["rights"]; present {
			t.Errorf("%s leaked rights", right)
		}
	}
	f := setup(t, SeedSample).ready()
	f.setRights(t, SampleAdmin3, gate.StopModules)
	f.s.fault = func(step string) bool { return step == "plugins-licenses" }
	if got := f.integrations(f.as(Admin3Login)); got.status != 503 {
		t.Errorf("a shown area's failure was hidden %s", got.raw)
	}
	g := setup(t, SeedSample).ready()
	g.setRights(t, SampleAdmin3, gate.AdministerAPIClients)
	g.s.fault = func(step string) bool {
		return strings.HasPrefix(step, "plugins-") || strings.HasPrefix(step, "webhooks-") || strings.HasPrefix(step, "events-")
	}
	if got := g.integrations(g.as(Admin3Login)); got.status != 200 {
		t.Errorf("a withheld area's failure failed the read %s", got.raw)
	}
}

func TestReadersWhoCannotApproveTriggerNoAdmissionChecks(t *testing.T) {
	count := func(rights ...gate.Action) int {
		f := setup(t, SeedSample).ready()
		f.setRights(t, SampleAdmin3, rights...)
		n := 0
		f.s.fault = func(step string) bool {
			if step == "destinations" || step == "permits" {
				n++
			}
			return false
		}
		if got := f.integrations(f.as(Admin3Login)); got.status != 200 {
			t.Fatal(got.raw)
		}
		return n
	}
	if n := count(gate.StopChannels); n != 0 {
		t.Errorf("a reader without the approval right caused %d admission reads", n)
	}
	if n := count(gate.ApproveChannels); n == 0 {
		t.Error("an approver caused no admission read")
	}
}

func TestUnknownTargetsAndMissingRightsLookAlike(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setRights(t, SampleAdmin3, gate.StopModules)
	strip := func(r reply) string {
		delete(r.body, "request_id")
		raw, _ := json.Marshal(r.body)
		return strconv.Itoa(r.status) + string(raw)
	}
	a := f.as(Admin3Login)
	want := strip(f.runWith(a, f.s.Revision(), integrationCommand+pluginApproveKind, pluginProposalAt(8), map[string]string{}))
	for _, c := range [][2]string{{pluginStopKind, pluginTarget + "missing"}, {pluginApproveKind, pluginProposalAt(99)}, {subPauseKind, subscriptionAt("missing")},
		{letterRedeliverKind, letterAt("missing")}, {resumeWithdrawKind, resumeRequestTarget + "resume-request-9"}, {redeliverWithdrawKind, letterRequestTarget + "letter-request-9"},
		{reapproveKind, redeliveryTarget + "delivery-9"}} {
		if got := strip(f.runWith(a, f.s.Revision(), integrationCommand+c[0], c[1], fieldsFor(c[0]))); got != want {
			t.Errorf("%s %s: %s; want %s", c[0], c[1], got, want)
		}
	}
	now := f.s.clock()
	scope := gate.Scope{WorkspaceID: SampleWorkspace, Customer: gate.All(), LegalEntity: gate.All(), Contract: gate.All()}
	terms := gate.Terms{Action: gate.ApproveModules, Kind: gate.InstallationKind, Period: gate.AnyPeriod(), Scope: scope}
	if err := f.s.admin.AddGrant(gate.Grant{ID: "brief", SubjectID: SampleAdmin3, Terms: terms, Validity: gate.Validity{NotBefore: now, NotAfter: now.Add(time.Minute)}}); err != nil {
		t.Fatal(err)
	}
	f.advance(2 * time.Minute)
	if got := strip(f.runWith(f.as(Admin3Login), f.s.Revision(), integrationCommand+pluginApproveKind, pluginProposalAt(8), map[string]string{})); got != want {
		t.Errorf("an expired grant: %s", got)
	}
}
