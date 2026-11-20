package server

import (
	"bytes"
	"log/slog"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

var holdAllowed = []string{"access.reject_request", "access.request_revoke", "access.revoke_delegation", "access.withdraw_request", "bulk_revocation.add_matching", "bulk_revocation.add_targets",
	"bulk_revocation.discard", "bulk_revocation.execute", "bulk_revocation.prepare", "identity.depart_user",
	"identity.remove_membership", "identity.revoke_invitation", "identity.suspend_membership", "identity.suspend_user", "input.reject", "input_authority.reject", "input_authority.withdraw", "installation.withdraw",
	"integrations.letter_withdraw_dismiss", "integrations.letter_withdraw_redeliver", "integrations.plugin_disable", "integrations.plugin_stop", "integrations.plugin_withdraw",
	"integrations.subscription_pause", "integrations.subscription_revoke", "integrations.subscription_withdraw", "integrations.subscription_withdraw_resume",
	"pricing_authority.reject", "pricing_authority.withdraw", "pricing_policy.revoke", "recovery.lock", "recovery.withdraw", "relationship.discard", "release_evidence.withdraw", "retention.place_hold", "retention.stop_run", "retention.withdraw",
	"service.retire", "service.revoke_credential", "service.suspend", "view_group.discard_draft"}

var holdConditional = []string{"access.approve_request", "identity.change_role", "identity.set_role_period"}

var holdRefused = []string{"access.request_grant", "contract.end", "contract.evidence", "contract.renew", "contract.reopen", "contract.resume", "contract.settle",
	"contract.suspend", "contract.withdraw", "financial_policy.confirm", "financial_policy.propose", "identity.create_user", "identity.invite", "identity.issue_reset_link", "identity.issue_set_link",
	"identity.reinstate_membership", "identity.reinstate_user", "input.confirm_publish", "input_authority.approve", "input_authority.propose_console", "input_authority.propose_human", "installation.approve", "installation.propose_plan", "integrations.letter_approve_dismiss",
	"integrations.letter_approve_redeliver", "integrations.letter_request_dismiss", "integrations.letter_request_redeliver", "integrations.plugin_approve",
	"integrations.plugin_propose", "integrations.redelivery_reapprove", "integrations.subscription_approve", "integrations.subscription_approve_resume",
	"integrations.subscription_propose", "integrations.subscription_request_resume", "pricing_authority.approve", "pricing_authority.propose_console", "pricing_authority.propose_human", "pricing_policy.approve", "pricing_policy.capture", "pricing_policy.propose", "recovery.approve", "recovery.propose_unlock", "relationship.activate", "relationship.propose",
	"release_evidence.approve", "release_evidence.retire_profile", "release_evidence.retire_requirement", "retention.approve", "retention.propose_policy",
	"retention.propose_release", "service.create", "service.issue_credential", "service.reinstate", "trust.activate", "trust.approve", "trust.cancel", "trust.propose", "trust.reject", "trust.validate",
	"view_group.draft", "view_group.edit", "view_group.propose"}

func TestHoldTableIsExplicitAndComplete(t *testing.T) {
	f := setup(t, SeedSample).ready()
	registered := []string{}
	for id, mutating := range f.s.mutating {
		if !mutating {
			continue
		}
		registered = append(registered, id)
		if _, ok := holdTable[id]; !ok {
			t.Errorf("registered command %s has no hold row", id)
		}
	}
	sort.Strings(registered)
	byRule := map[string][]string{}
	for id, rule := range holdTable {
		if !f.s.mutating[id] {
			t.Errorf("hold row %s is not a registered mutating command", id)
		}
		if rule.reason == "" {
			t.Errorf("hold row %s has no reason", id)
		}
		key := rule.rule
		if key == holdRoleDecrease || key == holdRevokeApproval || key == holdPeriodDecrease {
			key = "conditional"
		}
		byRule[key] = append(byRule[key], id)
	}
	for key, want := range map[string][]string{holdAllow: holdAllowed, "conditional": holdConditional, holdRefuse: holdRefused} {
		got := byRule[key]
		sort.Strings(got)
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("%s rows\n%v\nwant\n%v", key, got, want)
		}
	}
	if len(registered) != 106 || len(holdTable) != 106 {
		t.Errorf("registered %d rows %d", len(registered), len(holdTable))
	}
	data, err := os.ReadFile("boundary.go")
	f.must(err)
	routes := regexp.MustCompile(`"(POST|PUT|PATCH|DELETE) (/[^"]+)"`).FindAllStringSubmatch(string(data), -1)
	for _, m := range routes {
		if rule, ok := holdRoutes[m[2]]; !ok || rule.reason == "" {
			t.Errorf("route %s %s has no hold row", m[1], m[2])
		}
	}
	for _, r := range postRoutes {
		if _, ok := holdRoutes[r.path]; !ok {
			t.Errorf("post route %s has no hold row", r.path)
		}
	}
	if len(routes) != len(holdRoutes) || len(holdRoutes) != 8 {
		t.Errorf("routes %d rows %d", len(routes), len(holdRoutes))
	}
	if f.s.holdAllows("synthetic.unknown", commands.Call{}, identity.Verifier{}) {
		t.Error("an unlisted command is allowed")
	}
}

func (f world) holdLogs(logs *bytes.Buffer, command string) int {
	n := 0
	for _, entry := range logLines(f.t, logs.String(), "trust_hold_refused") {
		if entry["command"] == command {
			n++
		}
	}
	return n
}

func TestHoldAllowsDecreasesOnly(t *testing.T) {
	var logs bytes.Buffer
	f := setupWith(t, func(c *Config) { c.Log = slog.New(slog.NewJSONHandler(&logs, nil)) })
	p := f.trustAdmins()
	f.setPassword(MemberLogin, pw)
	invited, _ := f.created(p.a, "invited@example.invalid", "member")
	_, link := f.created(p.a, "other@example.invalid", "member")
	pending := f.requestGrant(p.a, SampleMember, nil)
	withdrawn := f.requestGrant(p.a, SampleAdmin3, nil)
	resetRef := f.run(p.a, "identity.issue_reset_link", SampleMember, nil).body["fields"].(map[string]any)["link_ref"].(string)
	resetToken := linkToken(f.collect(p.a, resetRef).body["url"].(string))
	f.s.domain.Lock()
	f.s.enterHold("commit_unknown")
	f.s.domain.Unlock()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	for _, id := range append(append([]string{}, holdRefused...), "access.request_grant") {
		got := f.run(root, id, "any-target", nil)
		f.code(t, id, got, 503, string(commands.DependencyUnavailable))
	}
	before := map[string]int{}
	for _, id := range holdAllowed {
		before[id] = f.holdLogs(&logs, id)
		f.run(root, id, "any-target", nil)
		if f.holdLogs(&logs, id) != before[id] {
			t.Errorf("the hold refused %s", id)
		}
	}
	if got := f.run(root, "identity.revoke_invitation", invited, nil); got.status != 200 {
		t.Errorf("revoke invitation %s", got.raw)
	}
	if got := f.run(root, "access.withdraw_request", requestOf(withdrawn), nil); got.status != 200 {
		t.Errorf("withdraw %s", got.raw)
	}
	f.code(t, "an add approval", f.approveRequest(second, pending), 503, string(commands.DependencyUnavailable))
	if got := f.run(second, "access.reject_request", requestOf(pending), nil); got.status != 200 {
		t.Errorf("reject %s", got.raw)
	}
	revoke := f.run(root, "access.request_revoke", "reader-"+SampleAdmin2, nil)
	refusedApprovals := f.holdLogs(&logs, "access.approve_request")
	if got := f.approveRequest(f.as(Admin3Login), revoke); got.status != 202 || f.holdLogs(&logs, "access.approve_request") != refusedApprovals {
		t.Errorf("a revoke approval %s", got.raw)
	}
	f.code(t, "a role increase", f.run(root, "identity.change_role", SampleMember, map[string]string{"role": "administrator"}), 503, string(commands.DependencyUnavailable))
	if got := f.run(root, "identity.change_role", SampleAdmin2, map[string]string{"role": "member"}); got.status != 200 {
		t.Errorf("a role decrease %s", got.raw)
	}
	f.code(t, "a role without a value", f.run(root, "identity.change_role", SampleAdmin2, nil), 503, string(commands.DependencyUnavailable))
	if got := f.selfPost(root, "sessions/end", f.othersBody(root)); got.status != 202 {
		t.Errorf("ending sessions %s", got.raw)
	}
	if got := f.reauth(root, pw); got.status != 204 {
		t.Errorf("password confirmation %s", got.raw)
	}
	if got := f.signin(AdminLogin, pw); got.status != 200 {
		t.Errorf("sign-in during the hold %s", got.raw)
	}
	f.code(t, "a set link", f.redeem(link, pw), 503, string(commands.DependencyUnavailable))
	if _, err := f.s.passwords.Peek(link); err != nil {
		t.Errorf("a refused set link was consumed %v", err)
	}
	if got := f.redeem(resetToken, "another password for the member"); got.status != 204 {
		t.Errorf("a reset link %s", got.raw)
	}
	f.s.domain.Lock()
	f.s.hold = nil
	f.s.domain.Unlock()
	if got := f.redeem(link, pw); got.status != 204 {
		t.Errorf("the refused link is still usable %s", got.raw)
	}
}

func TestHoldStopsInvitationAcceptance(t *testing.T) {
	f := setup(t, SeedSample).ready()
	p := f.trustAdmins()
	if got := f.run(p.a, "identity.remove_membership", SampleAdmin3, nil); got.status != 200 {
		t.Fatal(got.raw)
	}
	if got := f.run(p.a, "identity.invite", SampleAdmin3, map[string]string{"role": "member"}); got.status != 200 {
		t.Fatal(got.raw)
	}
	f.s.domain.Lock()
	f.s.enterHold("commit_unknown")
	f.s.domain.Unlock()
	f.code(t, "sign-in without a membership", f.signin(Admin3Login, pw), 401, string(commands.Unauthenticated))
	if _, ok := f.s.dir.invitation(Admin3Login, f.s.clock()); !ok {
		t.Error("the invitation was accepted during the hold")
	}
	f.s.domain.Lock()
	f.s.hold = nil
	f.s.domain.Unlock()
	if got := f.signin(Admin3Login, pw); got.status != 200 {
		t.Errorf("after the hold %s", got.raw)
	}
}

func TestHoldRaceBetweenReserveAndCommit(t *testing.T) {
	f := setup(t, SeedSample).ready()
	link := f.links()[Admin2Login]
	f.s.pause = func() {
		f.s.pause = nil
		f.s.domain.Lock()
		f.s.enterHold("commit_unknown")
		f.s.domain.Unlock()
	}
	f.code(t, "a hold that starts while hashing", f.redeem(link, pw), 503, string(commands.DependencyUnavailable))
	if _, err := f.s.passwords.Peek(link); err != nil || f.s.passwords.Has(LoginKey(Admin2Login)) {
		t.Errorf("the link was consumed %v", err)
	}
	g := setup(t, SeedNone).ready()
	g.s.domain.Lock()
	g.s.enterHold("commit_unknown")
	g.s.domain.Unlock()
	g.code(t, "bootstrap during the hold", g.auth("redeem", `{"token":"`+g.links()[""]+`","password":"`+pw+`","login":"first@example.invalid","email":"first@example.invalid"}`, g.pre()), 503,
		string(commands.DependencyUnavailable))
	g.s.domain.Lock()
	g.s.hold = nil
	g.s.domain.Unlock()
	g.s.pause = func() {
		g.s.pause = nil
		g.s.domain.Lock()
		g.s.enterHold("commit_unknown")
		g.s.domain.Unlock()
	}
	g.code(t, "a hold that starts while the bootstrap hashes", g.auth("redeem", `{"token":"`+g.links()[""]+`","password":"`+pw+`","login":"first@example.invalid","email":"first@example.invalid"}`, g.pre()), 503,
		string(commands.DependencyUnavailable))
	if !g.s.boot.live {
		t.Error("the bootstrap link was used")
	}
}
