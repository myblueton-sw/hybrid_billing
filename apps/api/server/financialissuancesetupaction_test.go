package server

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
)

func setupActionRead(f world, who actor, id, operation string, body map[string]string) reply {
	fields := map[string]string{"configuration": id, "operation": operation}
	for k, v := range body {
		fields[k] = v
	}
	return documentRead(f, who, "issuance_setup.action", "sample-contract-01", fields)
}

func actionWant(t *testing.T, got reply, permission, step, action, code string) {
	t.Helper()
	if got.status != 200 {
		t.Fatal(got.raw)
	}
	for key, want := range map[string]string{"permission": permission, "step_up": step, "action": action, "code": code, "current_eligibility": "unchecked"} {
		if field(got, key) != want {
			t.Fatal(key, got.raw)
		}
	}
}

func TestFinancialSetupActionJourney(t *testing.T) {
	f, root, second, third, fields := setupMutationWorld(t)
	args := map[string]string{"on": fields["on"], "scope_digest": fields["scope_digest"]}
	before := f.s.Revision()
	actionWant(t, setupActionRead(f, root, "action-config", "propose_configuration", args), "allowed", "satisfied", "recheck_required", "")
	if f.s.Revision() != before || len(f.s.financialIssuance.configurations) != 0 {
		t.Fatal("read reserved state")
	}
	p := draftRun(f, root, "issuance_setup.propose_configuration", "action-config", fields)
	f.ok(t, "proposal", p, "proposed", "")
	digest := map[string]string{"digest": field(p, "digest")}
	actionWant(t, setupActionRead(f, root, "action-config", "propose_configuration", args), "allowed", "satisfied", "recorded", "")
	actionWant(t, setupActionRead(f, root, "other-config", "propose_configuration", args), "allowed", "satisfied", "blocked", "configuration_exists")
	actionWant(t, setupActionRead(f, root, "action-config", "confirm_configuration", digest), "allowed", "satisfied", "blocked", "self_approval_denied")
	actionWant(t, setupActionRead(f, second, "action-config", "confirm_configuration", digest), "allowed", "satisfied", "recheck_required", "")
	f.ok(t, "confirmed", draftRun(f, second, "issuance_setup.confirm_configuration", "action-config", digest), "confirmed", "")
	actionWant(t, setupActionRead(f, second, "action-config", "confirm_configuration", digest), "allowed", "satisfied", "recorded", "")
	actionWant(t, setupActionRead(f, third, "action-config", "confirm_configuration", digest), "allowed", "satisfied", "blocked", "permission_denied")
}

func TestFinancialSetupActionUnknownPublication(t *testing.T) {
	for _, operation := range []string{"propose_configuration", "confirm_configuration"} {
		t.Run(operation, func(t *testing.T) {
			f, root, second, _, fields := setupMutationWorld(t)
			actor := root
			args := fields
			point := "propose"
			if operation == "confirm_configuration" {
				p := draftRun(f, root, "issuance_setup.propose_configuration", "unknown-action", fields)
				f.ok(t, "proposal", p, "proposed", "")
				args = map[string]string{"digest": field(p, "digest")}
				actor = second
				point = "confirm"
			}
			f.s.fault = func(at string) bool { return at == "issuance-configuration-"+point+"-after-module" }
			f.code(t, "interruption", draftRun(f, actor, "issuance_setup."+operation, "unknown-action", args), 500, string(commands.InternalError))
			f.s.fault = nil
			before := copyConfiguration(f.s.financialIssuance.configurations["unknown-action"])
			rev := f.s.Revision()
			readArgs := map[string]string{"on": fields["on"], "scope_digest": fields["scope_digest"]}
			if point == "confirm" {
				readArgs = args
			}
			actionWant(t, setupActionRead(f, actor, "unknown-action", operation, readArgs), "allowed", "satisfied", "recheck_required", "")
			if point == "propose" {
				actionWant(t, setupActionRead(f, second, "unknown-action", "confirm_configuration", map[string]string{"digest": before.Digest}), "allowed", "satisfied", "blocked", "configuration_pending")
			}
			if f.s.Revision() != rev || !reflect.DeepEqual(before, f.s.financialIssuance.configurations["unknown-action"]) {
				t.Fatal("read promoted unknown operation")
			}
		})
	}
}

func TestFinancialSetupActionPrecedence(t *testing.T) {
	f, member := setupContextWorld(t)
	actionWant(t, setupActionRead(f, member, "new", "propose_configuration", map[string]string{"on": "2026-11-16", "scope_digest": strings.Repeat("a", 64)}), "denied", "unchecked", "blocked", "permission_denied")
	f, root, _, _, fields := setupMutationWorld(t)
	args := map[string]string{"on": fields["on"], "scope_digest": fields["scope_digest"]}
	f.advance(16 * time.Minute)
	actionWant(t, setupActionRead(f, root, "new", "propose_configuration", args), "allowed", "reauthentication_required", "blocked", "reauthentication_required")
	f.s.enterHold("Action projection hold")
	actionWant(t, setupActionRead(f, root, "new", "propose_configuration", args), "allowed", "unchecked", "blocked", "installation_held")
}

func TestFinancialSetupActionScopeRefusals(t *testing.T) {
	f, root, second, _, fields := setupMutationWorld(t)
	args := map[string]string{"on": fields["on"], "scope_digest": strings.Repeat("a", 64)}
	actionWant(t, setupActionRead(f, root, "new", "propose_configuration", args), "allowed", "satisfied", "blocked", "evidence_stale")
	missing := setupActionRead(f, second, "absent", "confirm_configuration", map[string]string{"digest": strings.Repeat("a", 64)})
	actionWant(t, missing, "allowed", "satisfied", "blocked", "permission_denied")
	p := draftRun(f, root, "issuance_setup.propose_configuration", "scoped-action", fields)
	f.ok(t, "proposal", p, "proposed", "")
	foreign := documentRead(f, second, "issuance_setup.action", "sample-contract-02", map[string]string{"operation": "confirm_configuration", "configuration": "scoped-action", "digest": field(p, "digest")})
	actionWant(t, foreign, "allowed", "satisfied", "blocked", "permission_denied")
	profile, version, err := f.s.orgs.BillingProfile(SampleWorkspace, "sample-profile-north")
	f.must(err)
	profile.State = organizations.Suspended
	_, err = f.s.orgs.PutBillingProfile(profile, version)
	f.must(err)
	before := copyConfiguration(f.s.financialIssuance.configurations["scoped-action"])
	revision := f.s.Revision()
	actionWant(t, setupActionRead(f, second, "scoped-action", "confirm_configuration", map[string]string{"digest": field(p, "digest")}), "allowed", "satisfied", "blocked", "evidence_stale")
	if f.s.Revision() != revision || !reflect.DeepEqual(before, f.s.financialIssuance.configurations["scoped-action"]) {
		t.Fatal("stale action inspection changed retained state")
	}
}

func TestFinancialSetupActionLegacyAndInvalidInput(t *testing.T) {
	f, root, second, _, fields := setupMutationWorld(t)
	p, err := proposeConfigurationAs(f, root, configurationSpec())
	f.must(err)
	actionWant(t, setupActionRead(f, second, p.Spec.ID, "confirm_configuration", map[string]string{"digest": p.Digest}), "allowed", "satisfied", "blocked", "unsupported_capability")
	for _, args := range []map[string]string{
		{"on": fields["on"], "scope_digest": fields["scope_digest"], "digest": p.Digest},
		{"on": "2026-02-30", "scope_digest": fields["scope_digest"]},
		{"on": fields["on"], "scope_digest": "bad"},
	} {
		f.code(t, "invalid fields", setupActionRead(f, root, "new", "propose_configuration", args), 422, string(commands.InvalidRequest))
	}
}
