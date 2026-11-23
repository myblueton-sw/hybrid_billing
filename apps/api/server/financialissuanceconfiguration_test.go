package server

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
)

func configurationSpec() financialConfigurationSpec {
	s := issuanceSpec()
	return financialConfigurationSpec{ID: "monthly-configuration", Revision: 1, Contract: "sample-contract-01", ContractRevision: 2, BillingProfile: "sample-profile-north", BillingRevision: 1, Mode: s.Mode, Profile: s.Profile, Region: s.Region, Tax: s.Tax, Zone: s.Zone, Languages: s.Languages}
}

func proposeConfigurationAs(f world, who actor, spec financialConfigurationSpec) (financialIssuanceConfiguration, error) {
	var p financialIssuanceConfiguration
	err := inputAuthorityAs(f, who, func() error {
		var err error
		p, err = f.s.financialIssuance.proposeConfiguration(SampleWorkspace, f.s.answer.User(), "Reusable monthly invoice configuration", spec)
		return err
	})
	return p, err
}

func confirmConfigurationAs(f world, who actor, p financialIssuanceConfiguration) (financialIssuanceConfiguration, error) {
	var out financialIssuanceConfiguration
	err := inputAuthorityAs(f, who, func() error {
		var err error
		out, err = f.s.financialIssuance.confirmConfiguration(SampleWorkspace, p.Spec.ID, p.Digest, f.s.answer.User())
		return err
	})
	return out, err
}

func TestFinancialConfigurationIndependentSetup(t *testing.T) {
	f := setup(t, SeedSample).ready()
	a := f.s.financialIssuance
	proposer, confirmer := f.as(AdminLogin), f.as(Admin2Login)
	p, err := proposeConfigurationAs(f, proposer, configurationSpec())
	f.must(err)
	if p.State != "proposed" || !configurationProposed(p) || p.Proposer != SampleAdmin || len(p.Scope.Parties) != 2 || p.Scope.Document.Currency != "USD" || !reflect.DeepEqual(p.Spec.Languages, []string{"ar", "en", "ko"}) {
		t.Fatal("incorrect independently derived configuration")
	}
	legacy, err := issuanceProfileKind(issuanceSpec())
	f.must(err)
	if p.Kind.ID == legacy.ID {
		t.Fatal("legacy kind namespace reused")
	}
	_, err = confirmConfigurationAs(f, proposer, p)
	documentFailure(t, err, commands.SelfApprovalDenied)
	f.advance(time.Minute)
	confirmed, err := confirmConfigurationAs(f, confirmer, p)
	f.must(err)
	if !configurationActive(confirmed) || confirmed.State != "confirmed" || confirmed.ActivatedAt != confirmed.Active.At || !confirmed.ActivatedAt.After(p.Proposed.At) || confirmed.Digest != p.Digest {
		t.Fatal("activation is not immutable module time")
	}
	f.advance(time.Minute)
	again, err := confirmConfigurationAs(f, confirmer, p)
	f.must(err)
	if !reflect.DeepEqual(again, confirmed) {
		t.Fatal("confirmation replay drifted")
	}
	spec := configurationSpec()
	replay, err := proposeConfigurationAs(f, proposer, spec)
	f.must(err)
	if !reflect.DeepEqual(replay, confirmed) {
		t.Fatal("proposal replay reset confirmed configuration")
	}
	spec.Languages[0] = "changed"
	replay.Spec.Languages[0] = "changed"
	replay.Kind.Languages[0] = "changed"
	replay.Proposed.Kind.Languages[0] = "changed"
	replay.Active.Kind.Languages[0] = "changed"
	replay.Scope.Contract.Components[0].ID = "changed"
	replay.Scope.Parties[0].Party.ID = "changed"
	again, err = confirmConfigurationAs(f, confirmer, p)
	f.must(err)
	if !reflect.DeepEqual(again, confirmed) {
		t.Fatal("caller mutated retained snapshots")
	}
	if len(a.profiles) != 0 || len(a.kindReservations) != 0 || len(a.drafts) != 0 || len(a.configurations) != 1 || len(a.configurationScopes) != 1 {
		t.Fatal("setup created invoice-specific objects")
	}
	other := configurationSpec()
	other.ID = "another-month"
	_, err = proposeConfigurationAs(f, proposer, other)
	documentFailure(t, err, commands.PrerequisiteFailed)
}

func TestFinancialConfigurationProposalRecovery(t *testing.T) {
	for _, point := range []string{"before-intent", "after-intent", "after-module", "after-publication"} {
		t.Run(point, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			a := f.s.financialIssuance
			proposer, other := f.as(AdminLogin), f.as(Admin3Login)
			spec := configurationSpec()
			kind, err := configurationKind(spec)
			f.must(err)
			f.s.fault = func(at string) bool { return at == "issuance-configuration-propose-"+point }
			_, err = proposeConfigurationAs(f, proposer, spec)
			documentFailure(t, err, commands.DependencyUnavailable)
			f.s.fault = nil
			saved, exists := a.configurations[spec.ID]
			actual, lookup := a.documents.KindOf(SampleWorkspace, kind.ID)
			switch point {
			case "before-intent":
				if exists || len(a.configurationScopes) != 0 || !errors.Is(lookup, issuance.ErrNotFound) {
					t.Fatal("prewrite proposal mutated ownership")
				}
			case "after-intent":
				if !exists || saved.State != "proposal_unknown" || !reflect.DeepEqual(saved.Proposed, issuance.KindView{}) || !errors.Is(lookup, issuance.ErrNotFound) {
					t.Fatal("intent not retained before module")
				}
			default:
				f.must(lookup)
				if !exists || !reflect.DeepEqual(actual, saved.Proposed) {
					t.Fatal("successful full module snapshot not retained")
				}
			}
			if exists {
				_, err = proposeConfigurationAs(f, other, spec)
				documentFailure(t, err, commands.EvidenceStale)
				competitor := spec
				competitor.ID = "competing"
				_, err = proposeConfigurationAs(f, proposer, competitor)
				documentFailure(t, err, commands.PrerequisiteFailed)
			}
			f.advance(time.Minute)
			p, err := proposeConfigurationAs(f, proposer, spec)
			f.must(err)
			if p.State != "proposed" || !configurationProposed(p) || len(a.configurations) != 1 || len(a.configurationScopes) != 1 {
				t.Fatal("proposal recovery failed")
			}
			if exists && !saved.Proposed.At.IsZero() && !reflect.DeepEqual(p.Proposed, saved.Proposed) {
				t.Fatal("proposal timestamp rewritten")
			}
		})
	}
}

func TestFinancialConfigurationConfirmationRecovery(t *testing.T) {
	for _, point := range []string{"before-intent", "after-intent", "after-module", "after-publication"} {
		t.Run(point, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			a := f.s.financialIssuance
			proposer, confirmer, other := f.as(AdminLogin), f.as(Admin2Login), f.as(Admin3Login)
			p, err := proposeConfigurationAs(f, proposer, configurationSpec())
			f.must(err)
			f.advance(time.Minute)
			f.s.fault = func(at string) bool { return at == "issuance-configuration-confirm-"+point }
			_, err = confirmConfigurationAs(f, confirmer, p)
			documentFailure(t, err, commands.DependencyUnavailable)
			f.s.fault = nil
			saved := a.configurations[p.Spec.ID]
			actual, err := a.documents.KindOf(SampleWorkspace, p.Kind.ID)
			f.must(err)
			if point == "before-intent" {
				if !reflect.DeepEqual(saved, p) || !reflect.DeepEqual(actual, p.Proposed) {
					t.Fatal("prewrite confirmation mutated record")
				}
			} else {
				if saved.Confirmer != SampleAdmin2 {
					t.Fatal("confirmer not pinned")
				}
				_, err = confirmConfigurationAs(f, other, p)
				documentFailure(t, err, commands.PermissionDenied)
				if point == "after-intent" {
					if !reflect.DeepEqual(actual, p.Proposed) || !reflect.DeepEqual(saved.Active, issuance.KindView{}) {
						t.Fatal("intent fault reached module")
					}
				} else if !reflect.DeepEqual(saved.Active, actual) {
					t.Fatal("active snapshot lost")
				}
			}
			f.advance(time.Minute)
			confirmed, err := confirmConfigurationAs(f, confirmer, p)
			f.must(err)
			if confirmed.State != "confirmed" || !configurationActive(confirmed) || confirmed.ActivatedAt != confirmed.Active.At {
				t.Fatal("confirmation recovery failed")
			}
			if !saved.Active.At.IsZero() && !reflect.DeepEqual(saved.Active, confirmed.Active) {
				t.Fatal("recovery changed actual activation")
			}
		})
	}
}
