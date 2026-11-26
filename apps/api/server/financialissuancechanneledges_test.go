package server

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
	"github.com/myblueton-sw/hybrid_billing/modules/protectedauthority"
)

func TestFinancialChannelCurrentOrganizationEvidence(t *testing.T) {
	for _, mode := range []string{"parent", "issuer-version", "profile", "workspace"} {
		t.Run(mode, func(t *testing.T) {
			f, registrar, admitter, spec := channelWorld(t)
			_, err := f.s.orgs.PutBillingProfile(organizations.BillingProfile{ID: "unit-issuer", WorkspaceID: SampleWorkspace, IssuerID: "sample-unit-north-ops", BuyerID: "sample-unit-north-data", Currency: "USD", Path: organizations.InternalChargeback, State: organizations.Active}, 0)
			f.must(err)
			spec.Selector = financialChannelSelector{Kind: "entity", BillingProfile: "unit-issuer"}
			p, err := registerChannelAs(f, registrar, spec)
			f.must(err)
			if p.Subject.Subject != "entity:sample-le-north" || len(p.Subject.Parties) != 2 {
				t.Fatal("active legal parent not retained")
			}
			switch mode {
			case "parent", "issuer-version":
				id := "sample-le-north"
				if mode == "issuer-version" {
					id = "sample-unit-north-ops"
				}
				party, v, err := f.s.orgs.Party(SampleWorkspace, id)
				f.must(err)
				if mode == "parent" {
					party.State = organizations.Suspended
				}
				_, err = f.s.orgs.PutParty(party, v)
				f.must(err)
			case "profile":
				profile, v, err := f.s.orgs.BillingProfile(SampleWorkspace, "unit-issuer")
				f.must(err)
				profile.State = organizations.Suspended
				_, err = f.s.orgs.PutBillingProfile(profile, v)
				f.must(err)
			case "workspace":
				workspace, v, err := f.s.orgs.Workspace(SampleWorkspace)
				f.must(err)
				workspace.State = organizations.Suspended
				_, err = f.s.orgs.PutWorkspace(workspace, v)
				f.must(err)
			}
			_, err = admitChannelAs(f, admitter, p)
			documentFailure(t, err, commands.EvidenceStale)
			module, err := f.s.financialIssuance.fields.ChannelOf(SampleWorkspace, spec.ID)
			f.must(err)
			if module.Admitter != "" || f.s.financialIssuance.channels[spec.ID].State != "registered" {
				t.Fatal("stale evidence admitted channel")
			}
		})
	}
}

func TestFinancialChannelKindStableAcrossConsumption(t *testing.T) {
	f, second, first, digest := issuanceSourceWorld(t)
	f, registrar, admitter, spec := configuredChannelWorld(f)
	p, err := registerChannelAs(f, registrar, spec)
	f.must(err)
	before := copyChannelRecord(p)
	profile, err := proposeIssuanceProfile(f, first, digest, issuanceSpec())
	f.must(err)
	_, err = confirmIssuanceProfile(f, second, profile)
	f.must(err)
	replay, err := registerChannelAs(f, registrar, spec)
	f.must(err)
	if !reflect.DeepEqual(before, replay) {
		t.Fatal("profile consumption changed matured-channel identity")
	}
	_, err = admitChannelAs(f, admitter, p)
	f.must(err)
}

func TestFinancialChannelWrongAuthorizedAdmissionRecovery(t *testing.T) {
	f, registrar, admitter, spec := channelWorld(t)
	authority := f.s.protectedAuthority
	var ceiling string
	var permission inputauthority.Spec
	for id, s := range authority.ceilings {
		if s.Capability == protectedauthority.AdmitChannel {
			ceiling = id
			permission = s
		}
	}
	permission.Kind, permission.Recipient, permission.CeilingID = inputauthority.Human, SampleAdmin, ceiling
	grant, err := protectedPermissionProposal(f, f.as(Admin2Login), inputauthority.Proposal{ID: "alternate-admitter", Spec: permission, Rationale: "Independent alternate synthetic custodian"})
	f.must(err)
	_, err = protectedPermissionDecision(f, f.as(Admin3Login), grant)
	f.must(err)
	other := f.as(AdminLogin)
	p, err := registerChannelAs(f, registrar, spec)
	f.must(err)
	f.s.fault = func(at string) bool { return at == "issuance-channel-after-admit" }
	_, err = admitChannelAs(f, admitter, p)
	documentFailure(t, err, commands.DependencyUnavailable)
	f.s.fault = nil
	f.must(inputAuthorityAs(f, other, func() error {
		return f.s.financialIssuance.protectedCaller(SampleWorkspace, SampleAdmin, protectedauthority.AdmitChannel)
	}))
	_, err = admitChannelAs(f, other, p)
	documentFailure(t, err, commands.PermissionDenied)
	recovered, err := admitChannelAs(f, admitter, p)
	f.must(err)
	if recovered.Module.Admitter != SampleAdmin3 {
		t.Fatal("changed pinned admitter")
	}
}

func TestFinancialChannelCapacityAndWorkspace(t *testing.T) {
	f, registrar, _, spec := channelWorld(t)
	err := inputAuthorityAs(f, registrar, func() error {
		_, err := f.s.financialIssuance.registerChannel("other-workspace", f.s.answer.User(), spec)
		return err
	})
	documentFailure(t, err, commands.PermissionDenied)
	if len(f.s.financialIssuance.channels) != 0 {
		t.Fatal("cross-workspace reservation")
	}
	for i := 0; i < financialIssuanceLimit; i++ {
		spec.ID = fmt.Sprintf("capacity-channel-%d", i)
		_, err := registerChannelAs(f, registrar, spec)
		f.must(err)
	}
	spec.ID = "over-capacity"
	_, err = registerChannelAs(f, registrar, spec)
	documentFailure(t, err, commands.PrerequisiteFailed)
	if _, err := f.s.financialIssuance.fields.ChannelOf(SampleWorkspace, spec.ID); !errors.Is(err, protected.ErrNotFound) {
		t.Fatal("capacity refusal mutated module")
	}
}

func TestFinancialChannelAdmissionRecoveryRechecksRegistrar(t *testing.T) {
	f, registrar, admitter, spec := channelWorld(t)
	p, err := registerChannelAs(f, registrar, spec)
	f.must(err)
	f.s.fault = func(at string) bool { return at == "issuance-channel-after-admit" }
	_, err = admitChannelAs(f, admitter, p)
	documentFailure(t, err, commands.DependencyUnavailable)
	f.s.fault = nil
	saved := f.s.financialIssuance.channels[spec.ID]
	for _, g := range f.s.admin.Grants() {
		if g.SubjectID == SampleMember && g.Terms.Kind == protectedauthority.ResourceKind {
			f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
		}
	}
	_, err = admitChannelAs(f, admitter, p)
	documentFailure(t, err, commands.PermissionDenied)
	if !reflect.DeepEqual(saved, f.s.financialIssuance.channels[spec.ID]) {
		t.Fatal("lost uncertain admission ownership")
	}
	actual, err := f.s.financialIssuance.fields.ChannelOf(SampleWorkspace, spec.ID)
	f.must(err)
	if actual != saved.Module {
		t.Fatal("refusal mutated module")
	}
}

func TestFinancialChannelAdmissionRequiresOwnedCurrentRights(t *testing.T) {
	for _, mode := range []string{"unowned", "ceiling", "ended-session"} {
		t.Run(mode, func(t *testing.T) {
			f, registrar, admitter, spec := channelWorld(t)
			p, err := registerChannelAs(f, registrar, spec)
			f.must(err)
			a := f.s.financialIssuance
			switch mode {
			case "unowned":
				for id, s := range f.s.protectedAuthority.bindings {
					if s.Kind == inputauthority.Human && s.Capability == protectedauthority.AdmitChannel {
						delete(f.s.protectedAuthority.bindings, id)
					}
				}
			case "ceiling":
				for id, s := range f.s.protectedAuthority.ceilings {
					if s.Capability == protectedauthority.AdmitChannel {
						g, ok := f.s.grantView(id)
						if !ok {
							t.Fatal("missing ceiling")
						}
						f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
					}
				}
			}
			if mode == "ended-session" {
				err = inputAuthorityAs(f, admitter, func() error {
					f.s.endProbe(f.s.caller)
					_, err := a.admitChannel(SampleWorkspace, p.Spec.ID, p.Digest, f.s.answer.User(), "synthetic-admission-evidence")
					return err
				})
				documentFailure(t, err, commands.Unauthenticated)
			} else {
				_, err = admitChannelAs(f, admitter, p)
				documentFailure(t, err, commands.PermissionDenied)
			}
			actual, err := a.fields.ChannelOf(SampleWorkspace, spec.ID)
			f.must(err)
			if actual != p.Module || !reflect.DeepEqual(a.channels[spec.ID], p) {
				t.Fatal("unauthorized admission changed ownership or module")
			}
		})
	}
}
