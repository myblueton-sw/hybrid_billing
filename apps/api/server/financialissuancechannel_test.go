package server

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
	"github.com/myblueton-sw/hybrid_billing/modules/protectedauthority"
)

func grantChannelCapability(f world, capability, recipient string) {
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	spec := protectedPermissionSpec(f, capability)
	p, err := protectedPermissionProposal(f, root, inputauthority.Proposal{ID: capability + "-ceiling", Spec: spec, Rationale: "Explicit synthetic channel ceiling"})
	f.must(err)
	c, err := protectedPermissionDecision(f, second, p)
	f.must(err)
	spec.Kind, spec.Recipient, spec.CeilingID = inputauthority.Human, recipient, c.Decision.GrantID
	p, err = protectedPermissionProposal(f, root, inputauthority.Proposal{ID: capability + "-human", Spec: spec, Rationale: "Explicit synthetic channel operator"})
	f.must(err)
	_, err = protectedPermissionDecision(f, second, p)
	f.must(err)
}
func channelWorld(t *testing.T) (world, actor, actor, financialChannelSpec) {
	t.Helper()
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	return configuredChannelWorld(f)
}
func configuredChannelWorld(f world) (world, actor, actor, financialChannelSpec) {
	rules, err := proposeRulesAs(f, f.as(AdminLogin), "channel-rules", issuanceRules())
	f.must(err)
	_, err = confirmRulesAs(f, f.as(Admin2Login), rules)
	f.must(err)
	r, err := proposeKindReservationForTest(f, f.as(AdminLogin), issuanceSpec())
	f.must(err)
	r, err = confirmKindReservationForTest(f, f.as(Admin2Login), r)
	f.must(err)
	grantChannelCapability(f, protectedauthority.RegisterChannel, SampleMember)
	grantChannelCapability(f, protectedauthority.AdmitChannel, SampleAdmin3)
	return f, f.as(MemberLogin), f.as(Admin3Login), financialChannelSpec{ID: "channel-1", Kind: protected.ChannelPortal, Ref: protected.Ref{ID: "synthetic-channel-ref", Fingerprint: strings.Repeat("a", 64)}, Evidence: "synthetic-registration-evidence", Reason: "Synthetic channel configuration", Selector: financialChannelSelector{Kind: "kind", Reservation: r.Spec.ID, ReservationDigest: r.Digest}}
}
func registerChannelAs(f world, who actor, spec financialChannelSpec) (financialChannelRecord, error) {
	var p financialChannelRecord
	err := inputAuthorityAs(f, who, func() error {
		var err error
		p, err = f.s.financialIssuance.registerChannel(SampleWorkspace, f.s.answer.User(), spec)
		return err
	})
	return p, err
}
func admitChannelAs(f world, who actor, p financialChannelRecord) (financialChannelRecord, error) {
	var out financialChannelRecord
	err := inputAuthorityAs(f, who, func() error {
		var err error
		out, err = f.s.financialIssuance.admitChannel(SampleWorkspace, p.Spec.ID, p.Digest, f.s.answer.User(), "synthetic-admission-evidence")
		return err
	})
	return out, err
}
func TestFinancialChannelActualIndependentFlow(t *testing.T) {
	f, registrar, admitter, spec := channelWorld(t)
	before := f.s.clock()
	p, err := registerChannelAs(f, registrar, spec)
	f.must(err)
	if p.State != "registered" || p.Module.Registrar != SampleMember || !p.Module.Registered.Equal(before) {
		t.Fatal("actual registration missing")
	}
	p, err = admitChannelAs(f, admitter, p)
	f.must(err)
	if p.State != "admitted" || p.Module.Admitter != SampleAdmin3 || !p.Module.Admitted.Equal(f.s.clock()) {
		t.Fatal("actual admission missing")
	}
	v, err := f.s.financialIssuance.fields.ChannelOf(SampleWorkspace, spec.ID)
	f.must(err)
	if p.Module != v {
		t.Fatal("actual channel differs")
	}
	replay, err := registerChannelAs(f, registrar, spec)
	f.must(err)
	if !reflect.DeepEqual(replay, p) {
		t.Fatal("registration replay changed admission")
	}
	p.Subject.ReservedKind.Languages[0] = "changed"
	replay.Subject.ReservationSpec.Languages[0] = "changed"
	again, err := registerChannelAs(f, registrar, spec)
	f.must(err)
	if again.Subject.ReservedKind.Languages[0] == "changed" || again.Subject.ReservationSpec.Languages[0] == "changed" {
		t.Fatal("mutable returned evidence")
	}
	if len(f.s.financialIssuance.profiles) != 0 {
		t.Fatal("channel required financial approval")
	}
	f.must(inputAuthorityAs(f, registrar, func() error {
		d := f.s.gate.Decide(f.s.answer, ConsoleClient, gate.Request{Action: gate.AdministerInstallation, Kind: gate.InstallationKind, Resource: gate.Resource{WorkspaceID: SampleWorkspace}})
		if d.Allowed {
			t.Fatal("ordinary channel operator gained installation authority")
		}
		return nil
	}))
}
func TestFinancialChannelFaultRecovery(t *testing.T) {
	for _, point := range []string{"before-register", "after-register", "after-registration-publication", "before-admit", "after-admit", "after-admission-publication"} {
		t.Run(point, func(t *testing.T) {
			f, registrar, admitter, spec := channelWorld(t)
			var p financialChannelRecord
			admission := strings.Contains(point, "admit") || strings.Contains(point, "admission")
			if admission {
				var err error
				p, err = registerChannelAs(f, registrar, spec)
				f.must(err)
			}
			f.s.fault = func(at string) bool { return at == "issuance-channel-"+point }
			var err error
			if admission {
				_, err = admitChannelAs(f, admitter, p)
			} else {
				_, err = registerChannelAs(f, registrar, spec)
			}
			documentFailure(t, err, commands.DependencyUnavailable)
			f.s.fault = nil
			a := f.s.financialIssuance
			saved, exists := a.channels[spec.ID]
			if point == "before-register" {
				if exists {
					t.Fatal("before-register retained intent")
				}
				if _, err := a.fields.ChannelOf(SampleWorkspace, spec.ID); !errors.Is(err, protected.ErrNotFound) {
					t.Fatal("before-register wrote channel")
				}
			} else if !exists {
				t.Fatal("lost channel intent")
			}
			if point == "after-register" && saved.State != "registration_unknown" || point == "after-admit" && saved.State != "admission_unknown" {
				t.Fatal("lost uncertain state")
			}
			p, err = registerChannelAs(f, registrar, spec)
			f.must(err)
			p, err = admitChannelAs(f, admitter, p)
			f.must(err)
			if p.State != "admitted" || len(a.channels) != 1 {
				t.Fatal("recovery failed")
			}
		})
	}
}
func TestFinancialChannelRefusesForeignAndRetired(t *testing.T) {
	for _, mode := range []string{"foreign", "retired", "lookup", "registrar", "admitter", "rules"} {
		t.Run(mode, func(t *testing.T) {
			f, registrar, admitter, spec := channelWorld(t)
			a := f.s.financialIssuance
			if mode == "foreign" {
				subject, err := a.channelSubject(SampleWorkspace, spec.Selector)
				f.must(err)
				_, err = a.fields.Register(SampleWorkspace, protected.Channel{ID: spec.ID, Subject: subject.Subject, Kind: spec.Kind, Ref: spec.Ref, Evidence: spec.Evidence}, protected.Actor{ID: SampleMember, Kind: protected.KindHuman})
				f.must(err)
				_, err = registerChannelAs(f, registrar, spec)
				documentFailure(t, err, commands.EvidenceStale)
				if len(a.channels) != 0 {
					t.Fatal("foreign channel adopted")
				}
				return
			}
			p, err := registerChannelAs(f, registrar, spec)
			f.must(err)
			p, err = admitChannelAs(f, admitter, p)
			f.must(err)
			expected := commands.EvidenceStale
			switch mode {
			case "retired":
				_, err = a.fields.Retire(SampleWorkspace, spec.ID, protected.Actor{ID: SampleAdmin, Kind: protected.KindHuman})
				f.must(err)
			case "lookup":
				f.s.fault = func(at string) bool { return at == "issuance-channel-lookup" }
				expected = commands.DependencyUnavailable
			case "registrar", "admitter":
				action, actor := gate.RegisterIssuanceChannel, SampleMember
				if mode == "admitter" {
					action, actor = gate.AdmitIssuanceChannel, SampleAdmin3
				}
				for _, g := range f.s.admin.Grants() {
					if g.SubjectID == actor && g.Terms.Action == action {
						f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
					}
				}
				expected = commands.PermissionDenied
			case "rules":
				a.protectedRules.Digest = "changed"
			}
			_, err = registerChannelAs(f, registrar, spec)
			documentFailure(t, err, expected)
			if a.channels[spec.ID].Digest != p.Digest || a.channels[spec.ID].State != "admitted" {
				t.Fatal("refusal released ownership")
			}
		})
	}
}
func TestFinancialChannelOrganizationSelectors(t *testing.T) {
	f, registrar, admitter, spec := channelWorld(t)
	f.must(f.s.orgs.AppendContractRevision(organizations.ContractRevision{WorkspaceID: SampleWorkspace, ContractID: "channel-contract", Revision: 1, ProviderRef: "synthetic-provider", CustomerID: "sample-le-north", Path: organizations.Direct, Components: []organizations.Component{{ID: "channel-component", Kind: organizations.ProviderCharge, ProviderPaymentResponsibleID: "sample-le-north"}}}))
	for _, sel := range []financialChannelSelector{{Kind: "customer", Contract: "channel-contract", Revision: 1}, {Kind: "entity", BillingProfile: "sample-profile-north"}} {
		spec.ID = "channel-" + sel.Kind
		spec.Selector = sel
		p, err := registerChannelAs(f, registrar, spec)
		f.must(err)
		want := "customer:sample-le-north"
		if sel.Kind == "entity" {
			want = "entity:sample-le-seller"
		}
		if p.Subject.Subject != want {
			t.Fatal("wrong derived subject")
		}
		_, err = admitChannelAs(f, admitter, p)
		f.must(err)
		if sel.Kind == "customer" {
			p.Subject.Contract.Components[0].ID = "changed"
			replay, err := registerChannelAs(f, registrar, spec)
			f.must(err)
			if replay.Subject.Contract.Components[0].ID == "changed" {
				t.Fatal("contract evidence alias")
			}
		}
	}
}
func TestFinancialChannelClosedSelectorAndShape(t *testing.T) {
	for _, mode := range []string{"mixed", "missing", "bad-ref", "bad-kind", "long-id", "wrong-reservation"} {
		t.Run(mode, func(t *testing.T) {
			f, registrar, _, spec := channelWorld(t)
			expected := commands.InvalidRequest
			switch mode {
			case "mixed":
				spec.Selector.Contract = "mixed"
			case "missing":
				spec.Selector = financialChannelSelector{Kind: "entity", BillingProfile: "missing"}
				expected = commands.EvidenceStale
			case "bad-ref":
				spec.Ref.Fingerprint = "invalid"
			case "bad-kind":
				spec.Kind = protected.ChannelInPerson
			case "long-id":
				spec.ID = strings.Repeat("x", 129)
			case "wrong-reservation":
				spec.Selector.ReservationDigest = strings.Repeat("b", 64)
				expected = commands.EvidenceStale
			}
			_, err := registerChannelAs(f, registrar, spec)
			documentFailure(t, err, expected)
			if len(f.s.financialIssuance.channels) != 0 {
				t.Fatal("invalid request reserved ownership")
			}
			if _, err := f.s.financialIssuance.fields.ChannelOf(SampleWorkspace, spec.ID); !errors.Is(err, protected.ErrNotFound) {
				t.Fatal("invalid request wrote module")
			}
		})
	}
}
