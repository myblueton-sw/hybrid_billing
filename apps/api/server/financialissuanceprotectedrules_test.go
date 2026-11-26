package server

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
)

func issuanceRules() protected.Rules {
	return protected.Rules{Approvals: 1, Validity: 48 * time.Hour, MaxLead: 7 * 24 * time.Hour, ChannelAge: 24 * time.Hour}
}
func proposeRulesAs(f world, who actor, id string, rules protected.Rules) (financialProtectedRules, error) {
	var p financialProtectedRules
	err := inputAuthorityAs(f, who, func() error {
		var err error
		p, err = f.s.financialIssuance.proposeProtectedRules(SampleWorkspace, id, f.s.answer.User(), "Explicit synthetic protected rules", rules)
		return err
	})
	return p, err
}
func confirmRulesAs(f world, who actor, p financialProtectedRules) (financialProtectedRules, error) {
	var out financialProtectedRules
	err := inputAuthorityAs(f, who, func() error {
		var err error
		out, err = f.s.financialIssuance.confirmProtectedRules(SampleWorkspace, p.ID, p.Digest, f.s.answer.User())
		return err
	})
	return out, err
}
func TestFinancialIssuanceProtectedRulesIndependent(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	if f.s.financialIssuance.protectedRules != nil {
		t.Fatal("rules seeded")
	}
	p, err := proposeRulesAs(f, root, "rules-1", issuanceRules())
	f.must(err)
	if p.State != "proposed" || p.Module.Revision != 1 || p.Module.Proposer != SampleAdmin {
		t.Fatal("actual rule proposal missing")
	}
	_, err = confirmRulesAs(f, root, p)
	documentFailure(t, err, commands.SelfApprovalDenied)
	p, err = confirmRulesAs(f, second, p)
	f.must(err)
	if p.State != "confirmed" || p.Confirmer != SampleAdmin2 || p.Module.Confirmer != SampleAdmin2 {
		t.Fatal("independent confirmation missing")
	}
	actual, err := f.s.financialIssuance.fields.RulesOf(SampleWorkspace)
	f.must(err)
	if actual.Pending != nil || actual.Active == nil || *actual.Active != p.Module || len(actual.History) != 1 {
		t.Fatal("actual rules do not match retained snapshot")
	}
	again, err := proposeRulesAs(f, root, "rules-1", issuanceRules())
	f.must(err)
	if again != p {
		t.Fatal("proposal replay changed rules")
	}
	again, err = confirmRulesAs(f, second, p)
	f.must(err)
	if again != p {
		t.Fatal("confirmation replay changed rules")
	}
	_, err = proposeRulesAs(f, root, "competing", issuanceRules())
	documentFailure(t, err, commands.EvidenceStale)
	if len(f.s.financialIssuance.profiles) != 0 || len(f.s.financialIssuance.termIntents) != 0 || len(f.s.protectedAuthority.bindings) != 0 {
		t.Fatal("rules granted business authority or source eligibility")
	}
}
func TestFinancialIssuanceProtectedRulesFaultRecovery(t *testing.T) {
	for _, operation := range []string{"propose", "confirm"} {
		for _, point := range []string{"before", "after", "publication"} {
			t.Run(operation+"/"+point, func(t *testing.T) {
				f := setup(t, SeedSample).ready()
				root, second := f.as(AdminLogin), f.as(Admin2Login)
				var p financialProtectedRules
				var err error
				if operation == "confirm" {
					p, err = proposeRulesAs(f, root, "rules-1", issuanceRules())
					f.must(err)
				}
				fault := "issuance-protected-rules-" + point + "-" + operation
				if point == "publication" {
					fault = "issuance-protected-rules-after-" + operation + "-publication"
				}
				run := func() error {
					if operation == "propose" {
						_, err = proposeRulesAs(f, root, "rules-1", issuanceRules())
					} else {
						_, err = confirmRulesAs(f, second, p)
					}
					return err
				}
				f.s.fault = func(at string) bool { return at == fault }
				documentFailure(t, run(), commands.DependencyUnavailable)
				f.s.fault = nil
				if operation == "propose" && point == "before" {
					if f.s.financialIssuance.protectedRules != nil {
						t.Fatal("prewrite fault retained rule intent")
					}
					_, err = f.s.financialIssuance.fields.RulesOf(SampleWorkspace)
					if !errors.Is(err, protected.ErrNotFound) {
						t.Fatal("prewrite fault wrote module rules")
					}
				} else {
					expected := "registration_unknown"
					if operation == "confirm" {
						expected = "confirmation_unknown"
					}
					if point == "before" {
						expected = "proposed"
					}
					if point == "publication" {
						expected = "proposed"
						if operation == "confirm" {
							expected = "confirmed"
						}
					}
					if f.s.financialIssuance.protectedRules.State != expected {
						t.Fatal("lost rule recovery state")
					}
					_, err = proposeRulesAs(f, root, "competing", issuanceRules())
					documentFailure(t, err, commands.EvidenceStale)
				}
				f.must(run())
				f.must(run())
				stored := *f.s.financialIssuance.protectedRules
				active, err := f.s.financialIssuance.protectedRulesModule(SampleWorkspace, stored)
				f.must(err)
				if active != (operation == "confirm") {
					t.Fatal("recovery module state mismatch")
				}
			})
		}
	}
}
func TestFinancialIssuanceProtectedRulesForeignAndForms(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	for _, mutate := range []func(*protected.Rules){func(r *protected.Rules) { r.Approvals = 0 }, func(r *protected.Rules) { r.Approvals = 4 }, func(r *protected.Rules) { r.Validity = time.Minute }, func(r *protected.Rules) { r.MaxLead = 367 * 24 * time.Hour }, func(r *protected.Rules) { r.ChannelAge = 24*time.Hour - time.Nanosecond }, func(r *protected.Rules) { r.ChannelAge = 91 * 24 * time.Hour }} {
		rules := issuanceRules()
		mutate(&rules)
		_, err := proposeRulesAs(f, root, "bad", rules)
		documentFailure(t, err, commands.InvalidRequest)
	}
	if f.s.financialIssuance.protectedRules != nil {
		t.Fatal("invalid rules reserved")
	}
	_, err := f.s.financialIssuance.fields.Propose(SampleWorkspace, issuanceRules(), protected.Actor{ID: SampleAdmin, Kind: protected.KindHuman})
	f.must(err)
	_, err = proposeRulesAs(f, root, "foreign", issuanceRules())
	documentFailure(t, err, commands.EvidenceStale)
	if f.s.financialIssuance.protectedRules != nil {
		t.Fatal("foreign module rules adopted")
	}
}
func TestFinancialIssuanceProtectedRulesCurrentCallerAndGovernors(t *testing.T) {
	for _, mode := range []string{"ended_session", "proposer_revoked", "other_confirmer", "changed_module"} {
		t.Run(mode, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			root, second := f.as(AdminLogin), f.as(Admin2Login)
			p, err := proposeRulesAs(f, root, "rules-1", issuanceRules())
			f.must(err)
			f.s.fault = func(at string) bool { return at == "issuance-protected-rules-after-confirm" }
			_, err = confirmRulesAs(f, second, p)
			documentFailure(t, err, commands.DependencyUnavailable)
			f.s.fault = nil
			saved := *f.s.financialIssuance.protectedRules
			before, err := f.s.financialIssuance.fields.RulesOf(SampleWorkspace)
			f.must(err)
			switch mode {
			case "ended_session":
				err = inputAuthorityAs(f, second, func() error {
					f.s.sessions.remove(f.s.caller)
					_, err := f.s.financialIssuance.confirmProtectedRules(SampleWorkspace, p.ID, p.Digest, SampleAdmin2)
					return err
				})
				documentFailure(t, err, commands.Unauthenticated)
			case "proposer_revoked":
				removed := 0
				for _, g := range f.s.admin.Grants() {
					if g.SubjectID == SampleAdmin && g.Terms.Action == gate.AdministerInstallation {
						f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
						removed++
					}
				}
				if removed == 0 {
					t.Fatal("no proposer grant revoked")
				}
				_, err = confirmRulesAs(f, second, p)
				if !errors.Is(err, inputauthority.ErrDenied) {
					t.Fatal("revoked proposer accepted", err)
				}
			case "other_confirmer":
				_, err = confirmRulesAs(f, f.as(Admin3Login), p)
				documentFailure(t, err, commands.PermissionDenied)
			case "changed_module":
				rules := issuanceRules()
				rules.Approvals = 2
				_, err = f.s.financialIssuance.fields.Propose(SampleWorkspace, rules, protected.Actor{ID: SampleAdmin, Kind: protected.KindHuman})
				f.must(err)
				_, err = confirmRulesAs(f, second, p)
				documentFailure(t, err, commands.EvidenceStale)
			}
			if *f.s.financialIssuance.protectedRules != saved {
				t.Fatal("refusal released pinned rule intent")
			}
			if mode != "changed_module" {
				after, err := f.s.financialIssuance.fields.RulesOf(SampleWorkspace)
				f.must(err)
				if !reflect.DeepEqual(before, after) {
					t.Fatal("refusal changed actual module rules")
				}
			}
		})
	}
}
