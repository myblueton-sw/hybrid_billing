package server

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/protectedauthority"
)

func setupMutationWorld(t *testing.T) (world, actor, actor, actor, map[string]string) {
	t.Helper()
	f := setup(t, SeedSample).ready()
	root, second, third := f.as(AdminLogin), f.as(Admin2Login), f.as(Admin3Login)
	spec := protectedPermissionSpec(f, protectedauthority.Inspect)
	p, err := protectedPermissionProposal(f, root, inputauthority.Proposal{ID: "setup-inspect-ceiling", Spec: spec, Rationale: "Explicit setup inspection ceiling"})
	f.must(err)
	ceiling, err := protectedPermissionDecision(f, second, p)
	f.must(err)
	for _, recipient := range []string{SampleAdmin, SampleAdmin2, SampleAdmin3} {
		spec.Kind, spec.Recipient, spec.CeilingID = inputauthority.Human, recipient, ceiling.Decision.GrantID
		proposer, approver := root, second
		if recipient == SampleAdmin {
			proposer, approver = second, third
		} else if recipient == SampleAdmin2 {
			approver = third
		}
		p, err = protectedPermissionProposal(f, proposer, inputauthority.Proposal{ID: "setup-inspect-" + recipient, Spec: spec, Rationale: "Explicit setup inspection governor"})
		f.must(err)
		_, err = protectedPermissionDecision(f, approver, p)
		f.must(err)
	}
	context := setupContextRead(f, root, "sample-contract-01", "2026-11-16")
	if context.status != 200 {
		t.Fatal(context.raw)
	}
	s := configurationSpec()
	fields := map[string]string{"contract": s.Contract, "on": "2026-11-16", "scope_digest": field(context, "scope_digest"), "mode": s.Mode, "profile": s.Profile, "region": s.Region, "tax": s.Tax, "zone": s.Zone, "languages": "en,ko,ar"}
	return f, root, second, third, fields
}

func TestFinancialSetupMutationJourney(t *testing.T) {
	f, root, second, third, fields := setupMutationWorld(t)
	before := f.s.Revision()
	body := chargeBody(f, "setup-proposal-envelope", "public-configuration", fields)
	proposed := f.call("issuance_setup.propose_configuration", body, root.cookie, root.csrf)
	f.ok(t, "proposed", proposed, "proposed", "")
	if field(proposed, "setup_on") != fields["on"] || field(proposed, "scope_digest") != fields["scope_digest"] || field(proposed, "proposal_reason") != "Synthetic financial review" || f.s.Revision() == before {
		t.Fatal("missing immutable proposal receipt", proposed.raw)
	}
	current := f.s.Revision()
	replay := f.call("issuance_setup.propose_configuration", body, root.cookie, root.csrf)
	f.ok(t, "same envelope", replay, "proposed", "")
	if f.s.Revision() != current {
		t.Fatal("replay changed revision")
	}
	args := map[string]string{"digest": field(proposed, "digest")}
	f.code(t, "self confirm", draftRun(f, root, "issuance_setup.confirm_configuration", "public-configuration", args), 403, string(commands.SelfApprovalDenied))
	f.code(t, "other proposer", draftRun(f, third, "issuance_setup.propose_configuration", "public-configuration", fields), 403, string(commands.PermissionDenied))
	confirmed := draftRun(f, second, "issuance_setup.confirm_configuration", "public-configuration", args)
	f.ok(t, "independent confirmation", confirmed, "confirmed", "")
	f.code(t, "pinned confirmer", draftRun(f, third, "issuance_setup.confirm_configuration", "public-configuration", args), 403, string(commands.PermissionDenied))
	f.ok(t, "fresh owned confirmation", draftRun(f, second, "issuance_setup.confirm_configuration", "public-configuration", args), "confirmed", "")
	row := configurationPageRow(t, setupConfigurationsRead(f, root, "sample-contract-01", "", ""), 0)
	if row["setup_on"] != fields["on"] || row["proposal_reason"] != "Synthetic financial review" || row["state"] != "confirmed" {
		t.Fatal("history lacks retry identity")
	}
	retained := f.s.financialIssuance.configurations["public-configuration"]
	clone := copyConfiguration(retained)
	clone.Setup.On = "changed"
	if retained.Setup.On != fields["on"] {
		t.Fatal("setup context aliased")
	}
}

func TestFinancialSetupMutationInterruptionRecovery(t *testing.T) {
	for _, kind := range []string{"propose", "confirm"} {
		for _, point := range []string{"before-intent", "after-intent", "after-module", "after-publication"} {
			t.Run(kind+"/"+point, func(t *testing.T) {
				f, root, second, third, fields := setupMutationWorld(t)
				who, wrong := root, third
				command := "issuance_setup.propose_configuration"
				args := fields
				if kind == "confirm" {
					p := draftRun(f, root, command, "interrupted-configuration", fields)
					f.ok(t, "proposal", p, "proposed", "")
					command = "issuance_setup.confirm_configuration"
					who = second
					args = map[string]string{"digest": field(p, "digest")}
				}
				key := fmt.Sprintf("configuration-%s-%s", kind, point)
				body := chargeBody(f, key, "interrupted-configuration", args)
				f.s.fault = func(at string) bool { return at == "issuance-configuration-"+kind+"-"+point }
				got := f.call(command, body, who.cookie, who.csrf)
				status, code := 500, commands.InternalError
				if point == "before-intent" {
					status, code = 503, commands.DependencyUnavailable
				}
				f.code(t, "interrupted", got, status, string(code))
				f.s.fault = nil
				replay := f.call(command, body, who.cookie, who.csrf)
				if point != "before-intent" {
					f.code(t, "same envelope stays uncertain", replay, 500, string(commands.InternalError))
				} else {
					want := "proposed"
					if kind == "confirm" {
						want = "confirmed"
					}
					f.ok(t, "clean retry", replay, want, "")
				}
				f.code(t, "wrong actor recovery", draftRun(f, wrong, command, "interrupted-configuration", args), 403, string(commands.PermissionDenied))
				recovered := draftRun(f, who, command, "interrupted-configuration", args)
				want := "proposed"
				if kind == "confirm" {
					want = "confirmed"
				}
				f.ok(t, "fresh owned recovery", recovered, want, "")
				if len(f.s.financialIssuance.configurations) != 1 || len(f.s.financialIssuance.configurationScopes) != 1 {
					t.Fatal("recovery duplicated setup")
				}
			})
		}
	}
}

func TestFinancialSetupMutationPreflightAndLegacy(t *testing.T) {
	f, root, second, _, fields := setupMutationWorld(t)
	a := f.s.financialIssuance
	f.s.cfg.Commands.Value = 4095
	got := draftRun(f, root, "issuance_setup.propose_configuration", "budget-config", fields)
	f.code(t, "receipt capacity", got, 501, string(commands.UnsupportedCapability))
	if len(a.configurations) != 0 || len(a.configurationScopes) != 0 {
		t.Fatal("preflight reserved configuration")
	}
	f.s.cfg.Commands.Value = 4096
	old, err := proposeConfigurationAs(f, root, configurationSpec())
	f.must(err)
	f.code(t, "legacy closed", draftRun(f, second, "issuance_setup.confirm_configuration", old.Spec.ID, map[string]string{"digest": old.Digest}), 501, string(commands.UnsupportedCapability))
	if !reflect.DeepEqual(old, a.configurations[old.Spec.ID]) {
		t.Fatal("legacy adopted")
	}
	body := chargeBody(f, "escaped-reason", "escaped-config", fields)
	var envelope map[string]any
	f.must(json.Unmarshal([]byte(body), &envelope))
	envelope["rationale"] = strings.Repeat("\"", 1900)
	raw, err := json.Marshal(envelope)
	f.must(err)
	f.code(t, "escaped history row admission", f.call("issuance_setup.propose_configuration", string(raw), root.cookie, root.csrf), 422, string(commands.PrerequisiteFailed))
	if len(a.configurations) != 1 || len(a.configurationScopes) != 1 {
		t.Fatal("oversized history receipt retained an intent")
	}
}
