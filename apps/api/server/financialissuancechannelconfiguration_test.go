package server

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
	"github.com/myblueton-sw/hybrid_billing/modules/protectedauthority"
)

func configurationChannelWorld(t *testing.T) (world, actor, actor, financialChannelSpec) {
	t.Helper()
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	return configurationChannelWorldIn(t, f)
}

func configurationChannelWorldIn(t *testing.T, f world) (world, actor, actor, financialChannelSpec) {
	t.Helper()
	p, err := proposeConfigurationAs(f, f.as(AdminLogin), configurationSpec())
	f.must(err)
	c, err := confirmConfigurationAs(f, f.as(Admin3Login), p)
	f.must(err)
	rules, err := proposeRulesAs(f, f.as(AdminLogin), "configuration-channel-rules", issuanceRules())
	f.must(err)
	_, err = confirmRulesAs(f, f.as(Admin2Login), rules)
	f.must(err)
	login := activateProtectedMember(f, "configured-channel-admitter")
	grantChannelCapability(f, protectedauthority.RegisterChannel, SampleMember)
	grantChannelCapability(f, protectedauthority.AdmitChannel, "configured-channel-admitter")
	reference := configurationReference(c)
	return f, f.as(MemberLogin), f.as(login), financialChannelSpec{ID: "configuration-channel", Kind: protected.ChannelPortal, Ref: protected.Ref{ID: "synthetic-configured-channel", Fingerprint: strings.Repeat("a", 64)}, Evidence: "synthetic-registration", Reason: "Reusable document template verification", Selector: financialChannelSelector{Kind: "configuration_kind", Configuration: &reference}}
}

func TestFinancialConfiguredChannelIndependentSetupAndCopies(t *testing.T) {
	f, registrar, admitter, spec := configurationChannelWorld(t)
	a := f.s.financialIssuance
	p, err := registerChannelAs(f, registrar, spec)
	f.must(err)
	f.advance(time.Minute)
	confirmed, err := admitChannelAs(f, admitter, p)
	f.must(err)
	c := *confirmed.Subject.Configuration
	if confirmed.Subject.Subject != "kind:"+c.Kind.ID || confirmed.State != "admitted" || confirmed.Module.Admitter != "configured-channel-admitter" || !confirmed.Module.Admitted.Equal(f.s.clock()) || len(a.kindReservations) != 0 || len(a.profiles) != 0 || len(a.drafts) != 0 {
		t.Fatal("configured channel did not use independent actual setup")
	}
	equalSpec := spec
	r := *spec.Selector.Configuration
	equalSpec.Selector.Configuration = &r
	replay, err := registerChannelAs(f, registrar, equalSpec)
	f.must(err)
	if !reflect.DeepEqual(replay, confirmed) {
		t.Fatal("structurally equal independently allocated selector changed replay")
	}
	spec.Selector.Configuration.ID = "caller-changed"
	replay.Spec.Selector.Configuration.Digest = "changed"
	replay.Subject.Selector.Configuration.ID = "changed"
	replay.Subject.Configuration.Spec.Languages[0] = "changed"
	replay.Subject.Configuration.Scope.Parties[0].Party.ID = "changed"
	replay.Subject.ReservedKind.Languages[0] = "changed"
	again, err := admitChannelAs(f, admitter, confirmed)
	f.must(err)
	if !reflect.DeepEqual(again, confirmed) {
		t.Fatal("caller mutated retained configured channel")
	}
	actual, err := a.fields.ChannelOf(SampleWorkspace, p.Spec.ID)
	f.must(err)
	if actual != confirmed.Module {
		t.Fatal("channel module does not match retained admission")
	}
}

func TestFinancialConfiguredChannelRejectsMixedOrigins(t *testing.T) {
	f, registrar, _, spec := configurationChannelWorld(t)
	for _, mode := range []string{"missing", "unknown", "legacy-kind", "reservation", "contract", "billing", "revision", "digest"} {
		t.Run(mode, func(t *testing.T) {
			s := spec
			s.Selector = copyChannelSelector(spec.Selector)
			switch mode {
			case "missing":
				s.Selector.Configuration = nil
			case "unknown":
				s.Selector.Configuration.Schema = "unknown"
			case "legacy-kind":
				s.Selector.Kind = "kind"
			case "reservation":
				s.Selector.Reservation = "legacy"
			case "contract":
				s.Selector.Contract = "sample-contract-01"
			case "billing":
				s.Selector.BillingProfile = "sample-profile-north"
			case "revision":
				s.Selector.Revision = 1
			case "digest":
				s.Selector.Configuration.Digest = "invalid"
			}
			_, err := registerChannelAs(f, registrar, s)
			documentFailure(t, err, commands.InvalidRequest)
			if len(f.s.financialIssuance.channels) != 0 {
				t.Fatal("invalid origin reserved channel")
			}
		})
	}
}

func TestFinancialConfiguredChannelRecovery(t *testing.T) {
	for _, point := range []string{"before-register", "after-register", "after-registration-publication", "before-admit", "after-admit", "after-admission-publication"} {
		t.Run(point, func(t *testing.T) {
			f, registrar, admitter, spec := configurationChannelWorld(t)
			a := f.s.financialIssuance
			admission := strings.Contains(point, "admit") || strings.Contains(point, "admission")
			var p financialChannelRecord
			var err error
			if admission {
				p, err = registerChannelAs(f, registrar, spec)
				f.must(err)
			}
			f.s.fault = func(at string) bool { return at == "issuance-channel-"+point }
			if admission {
				_, err = admitChannelAs(f, admitter, p)
			} else {
				_, err = registerChannelAs(f, registrar, spec)
			}
			documentFailure(t, err, commands.DependencyUnavailable)
			f.s.fault = nil
			observed, exists := a.channels[spec.ID]
			module, moduleErr := a.fields.ChannelOf(SampleWorkspace, spec.ID)
			if point == "before-register" {
				if exists || !errors.Is(moduleErr, protected.ErrNotFound) {
					t.Fatal("prewrite register created channel")
				}
			} else {
				f.must(moduleErr)
				if module != observed.Module {
					t.Fatal("unknown operation lost actual module snapshot")
				}
			}
			f.advance(time.Minute)
			p, err = registerChannelAs(f, registrar, spec)
			f.must(err)
			confirmed, err := admitChannelAs(f, admitter, p)
			f.must(err)
			if confirmed.State != "admitted" || len(a.channels) != 1 || confirmed.Subject.Configuration == nil {
				t.Fatal("configured recovery did not preserve owned channel")
			}
			if !module.Admitted.IsZero() && confirmed.Module.Admitted != module.Admitted {
				t.Fatal("recovery manufactured a new admission time")
			}
		})
	}
}

func TestFinancialConfiguredChannelCurrentScopeAndOwnership(t *testing.T) {
	for _, mode := range []string{"governor", "party", "legacy-payload", "missing-snapshot", "foreign-module"} {
		t.Run(mode, func(t *testing.T) {
			f, registrar, admitter, spec := configurationChannelWorld(t)
			a := f.s.financialIssuance
			if mode == "foreign-module" {
				c := a.configurations[spec.Selector.Configuration.ID]
				_, err := a.fields.Register(SampleWorkspace, protected.Channel{ID: spec.ID, Subject: "kind:" + c.Kind.ID, Kind: spec.Kind, Ref: spec.Ref, Evidence: spec.Evidence}, protected.Actor{ID: SampleMember, Kind: protected.KindHuman})
				f.must(err)
				_, err = registerChannelAs(f, registrar, spec)
				documentFailure(t, err, commands.EvidenceStale)
				if len(a.channels) != 0 {
					t.Fatal("foreign module adopted")
				}
				return
			}
			f.s.fault = func(at string) bool { return at == "issuance-channel-after-admit" }
			p, err := registerChannelAs(f, registrar, spec)
			f.must(err)
			_, err = admitChannelAs(f, admitter, p)
			documentFailure(t, err, commands.DependencyUnavailable)
			f.s.fault = nil
			saved := copyChannelRecord(a.channels[spec.ID])
			module := saved.Module
			expected := commands.EvidenceStale
			switch mode {
			case "governor":
				removed := 0
				for _, g := range f.s.admin.Grants() {
					if g.SubjectID == SampleAdmin3 && g.Terms.Action == gate.AdministerInstallation {
						f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
						removed++
					}
				}
				if removed == 0 {
					t.Fatal("actual governor grant not removed")
				}
				expected = commands.PermissionDenied
				f.must(inputAuthorityAs(f, admitter, func() error {
					if err := a.protectedCaller(SampleWorkspace, "configured-channel-admitter", protectedauthority.AdmitChannel); err != nil {
						return err
					}
					_, err := a.channelRules(SampleWorkspace)
					return err
				}))
			case "party":
				party, v, e := f.s.orgs.Party(SampleWorkspace, "sample-le-north")
				f.must(e)
				party.State = organizations.Suspended
				_, err = f.s.orgs.PutParty(party, v)
				f.must(err)
			case "legacy-payload":
				saved.Subject.Contract.ContractID = "legacy-payload"
				a.channels[spec.ID] = saved
			case "missing-snapshot":
				saved.Subject.Configuration = nil
				a.channels[spec.ID] = saved
			}
			if mode == "legacy-payload" || mode == "missing-snapshot" {
				if _, err := channelDigest(saved); err == nil {
					t.Fatal("mixed retained shape received digest")
				}
			}
			_, err = admitChannelAs(f, admitter, p)
			documentFailure(t, err, expected)
			actual, err := a.fields.ChannelOf(SampleWorkspace, spec.ID)
			f.must(err)
			if actual != module || !reflect.DeepEqual(a.channels[spec.ID], saved) {
				t.Fatal("refused recovery changed state")
			}
		})
	}
}

func TestFinancialConfiguredChannelSharesLegacyIdentityOwnership(t *testing.T) {
	f, registrar, _, configured := configurationChannelWorld(t)
	a := f.s.financialIssuance
	reservation, err := proposeKindReservationForTest(f, f.as(AdminLogin), issuanceSpec())
	f.must(err)
	reservation, err = confirmKindReservationForTest(f, f.as(Admin2Login), reservation)
	f.must(err)
	legacy := configured
	legacy.ID = "legacy-owned-channel"
	legacy.Selector = financialChannelSelector{Kind: "kind", Reservation: reservation.Spec.ID, ReservationDigest: reservation.Digest}
	old, err := registerChannelAs(f, registrar, legacy)
	f.must(err)
	current, err := registerChannelAs(f, registrar, configured)
	f.must(err)
	foreign := configured
	foreign.ID = old.Spec.ID
	_, err = registerChannelAs(f, registrar, foreign)
	documentFailure(t, err, commands.EvidenceStale)
	foreign = legacy
	foreign.ID = current.Spec.ID
	_, err = registerChannelAs(f, registrar, foreign)
	documentFailure(t, err, commands.EvidenceStale)
	if len(a.channels) != 2 || !reflect.DeepEqual(a.channels[old.Spec.ID], old) || !reflect.DeepEqual(a.channels[current.Spec.ID], current) {
		t.Fatal("cross-format retry changed owned channels")
	}
}
