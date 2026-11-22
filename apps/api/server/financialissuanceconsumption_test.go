package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
	"reflect"
	"testing"
	"time"
)

func referenceConsumptionWorld(t *testing.T, ages ...time.Duration) (world, actor, actor, actor, financialIssuanceProfile, financialChangeSpec) {
	t.Helper()
	age := 24 * time.Hour
	if len(ages) > 1 {
		t.Fatal("multiple fixture ages")
	}
	if len(ages) == 1 {
		age = ages[0]
	}
	f, requester, verifier, approver, spec := protectedChangeWorld(t, age, 1)
	a := f.s.financialIssuance
	for _, entry := range []struct{ field, kind string }{{protected.FieldPaymentAccount, "entity"}, {protected.FieldDocumentRecipient, "customer"}, {protected.FieldDocumentTemplate, "kind"}} {
		spec.ID = "consume-" + entry.kind
		spec.Field, spec.Channel = entry.field, "mature-"+entry.kind
		spec.ChannelDigest = a.channels[spec.Channel].Digest
		p, err := requestChangeAs(f, requester, spec)
		f.must(err)
		p, err = verifyChangeAs(f, verifier, p)
		f.must(err)
		_, err = decideChangeAs(f, approver, p)
		f.must(err)
	}
	f.advance(time.Minute)
	return f, requester, verifier, approver, copyIssuanceProfile(a.profiles[spec.Profile]), spec
}

func TestFinancialDraftReferencesActualResolution(t *testing.T) {
	f, _, _, _, p, _ := referenceConsumptionWorld(t)
	grantDraftCapability(f, draftauthority.Submit)
	grantDraftCapability(f, draftauthority.Inspect)
	member := f.as(MemberLogin)
	before, err := f.s.approvals.RequestOf(SampleWorkspace, "issuance-final")
	f.must(err)
	f.must(inputAuthorityAs(f, member, func() error {
		current, err := f.s.financialIssuance.draftProfile(SampleWorkspace, p.Spec.ID, p.Digest, SampleMember, draftauthority.Submit)
		if err != nil {
			return err
		}
		refs, err := f.s.financialIssuance.draftReferences(current, f.s.clock())
		if err != nil {
			return err
		}
		for i, field := range []string{protected.FieldPaymentAccount, protected.FieldDocumentRecipient, protected.FieldDocumentTemplate} {
			if refs[i].Field != field || refs[i].Change == "" {
				t.Fatal("wrong actual reference ordering")
			}
		}
		refs[0].Ref.ID = "changed"
		again, err := f.s.financialIssuance.draftReferences(current, f.s.clock())
		if err != nil {
			return err
		}
		if again[0].Ref.ID == "changed" {
			t.Fatal("reference alias")
		}
		return nil
	}))
	after, err := f.s.approvals.RequestOf(SampleWorkspace, "issuance-final")
	f.must(err)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("reference consumption changed financial approval")
	}
}

func TestFinancialDraftReferenceFutureSuccessorAndUnknown(t *testing.T) {
	for _, unknown := range []bool{false, true} {
		f, requester, verifier, approver, p, spec := referenceConsumptionWorld(t)
		a := f.s.financialIssuance
		old, err := a.draftReferences(p, f.s.clock())
		f.must(err)
		spec.ID = "future-template"
		spec.After.ID = "future-template-reference"
		spec.Effective = f.s.clock().Add(time.Minute)
		if unknown {
			f.s.fault = func(at string) bool { return at == "issuance-change-after-request" }
		}
		next, err := requestChangeAs(f, requester, spec)
		f.s.fault = nil
		if unknown {
			documentFailure(t, err, commands.DependencyUnavailable)
			_, err = a.draftReferences(p, f.s.clock())
			documentFailure(t, err, commands.DependencyUnavailable)
			continue
		}
		f.must(err)
		next, err = verifyChangeAs(f, verifier, next)
		f.must(err)
		_, err = decideChangeAs(f, approver, next)
		f.must(err)
		now, err := a.draftReferences(p, f.s.clock())
		f.must(err)
		if !reflect.DeepEqual(now, old) {
			t.Fatal("future successor replaced present reference")
		}
		f.advance(time.Minute)
		future, err := a.draftReferences(p, f.s.clock())
		f.must(err)
		if future[2].Change != spec.ID || future[2].Ref != spec.After {
			t.Fatal("successor not selected at effective instant")
		}
	}
}

func TestFinancialDraftReferenceChangedProvenanceAndForeignTrail(t *testing.T) {
	for _, mode := range []string{"contract", "billing", "foreign"} {
		f, _, _, _, p, spec := referenceConsumptionWorld(t)
		a := f.s.financialIssuance
		switch mode {
		case "contract":
			p.Source.Basis.Contract.Revision++
		case "billing":
			p.Source.Basis.BillingRevision++
		case "foreign":
			_, err := a.fields.Request(SampleWorkspace, protected.Change{ID: "foreign-template", Subject: "kind:" + p.Kind.ID, Field: protected.FieldDocumentTemplate, After: protected.Ref{ID: "foreign-reference", Fingerprint: spec.After.Fingerprint}, Provenance: spec.Provenance, Instruction: spec.Instruction, Effective: f.s.clock().Add(time.Minute)}, protected.Actor{ID: SampleAdmin2, Kind: protected.KindHuman})
			f.must(err)
		}
		_, err := a.draftReferences(p, f.s.clock())
		if err == nil {
			t.Fatal("inapplicable or foreign reference accepted", mode)
		}
	}
}

func TestFinancialDraftCoverageActualAndApplicable(t *testing.T) {
	f, author, decider, term := approvedTermWorld(t)
	binding, err := bindTermAs(f, author, bindingSpec(term))
	f.must(err)
	binding, err = approveBindingAs(f, decider, binding)
	f.must(err)
	a := f.s.financialIssuance
	current := copyIssuanceProfile(term.Profile)
	actual, err := a.draftCoverage(current, binding.Spec.ID, binding.Digest, f.s.clock())
	f.must(err)
	expected, err := a.terms.Cover(SampleWorkspace, current.Source.Basis.Contract.ContractID, f.s.clock())
	f.must(err)
	if actual != expected || actual.Zone != "Asia/Tokyo" {
		t.Fatal("coverage not actual calendar result")
	}
	for _, mode := range []string{"contract", "customer", "zone", "binding"} {
		changed := copyIssuanceProfile(current)
		digest := binding.Digest
		switch mode {
		case "contract":
			changed.Source.Basis.Contract.Revision++
		case "customer":
			changed.Source.Basis.Customer = "other"
		case "zone":
			changed.Spec.Zone = "UTC"
		case "binding":
			digest = "wrong"
		}
		_, err := a.draftCoverage(changed, binding.Spec.ID, digest, f.s.clock())
		documentFailure(t, err, commands.EvidenceStale)
	}
	key := [2]string{SampleWorkspace, "public-result"}
	app := f.s.documentERP.resultApplications[key]
	app.receipt = "changed"
	f.s.documentERP.resultApplications[key] = app
	_, err = a.draftCoverage(current, binding.Spec.ID, binding.Digest, f.s.clock())
	f.must(err)
	_, err = coverageAs(f, decider, binding, f.s.clock())
	documentFailure(t, err, commands.EvidenceStale)
}

func TestFinancialDraftReferencesHistoricalStanding(t *testing.T) {
	for _, mode := range []string{"actor", "channel", "governor"} {
		t.Run(mode, func(t *testing.T) {
			f, _, _, _, p, _ := referenceConsumptionWorld(t)
			a := f.s.financialIssuance
			before, err := a.draftReferences(p, f.s.clock())
			f.must(err)
			expected := commands.PermissionDenied
			if mode == "channel" {
				expected = commands.EvidenceStale
				_, err = a.fields.Retire(SampleWorkspace, "mature-entity", protected.Actor{ID: SampleMember, Kind: protected.KindHuman})
				f.must(err)
			} else {
				action, target := gate.ApproveIssuanceProtectedChange, "protected-fifth"
				if mode == "governor" {
					action, target = gate.AdministerInstallation, p.Proposer
				}
				revoked := 0
				for _, g := range f.s.admin.Grants() {
					if g.Terms.Action == action && g.SubjectID == target {
						f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
						revoked++
					}
				}
				if revoked == 0 {
					t.Fatal("fixture revoked no actual grant")
				}
			}
			_, err = a.draftReferences(p, f.s.clock())
			documentFailure(t, err, expected)
			for _, ref := range before {
				v, err := a.fields.Of(SampleWorkspace, ref.Change)
				f.must(err)
				if v.State != protected.StateScheduled {
					t.Fatal("standing refusal changed historical approval")
				}
			}
		})
	}
}

func TestFinancialDraftCoverageUnknownAndRevoked(t *testing.T) {
	for _, mode := range []string{"unknown", "actor", "governor"} {
		t.Run(mode, func(t *testing.T) {
			f, author, decider, term := approvedTermWorld(t)
			a := f.s.financialIssuance
			binding, err := bindTermAs(f, author, bindingSpec(term))
			f.must(err)
			if mode == "unknown" {
				f.s.fault = func(at string) bool { return at == "issuance-binding-after-approve" }
			}
			approved, err := approveBindingAs(f, decider, binding)
			f.s.fault = nil
			expected := commands.PermissionDenied
			if mode == "unknown" {
				documentFailure(t, err, commands.DependencyUnavailable)
				expected = commands.EvidenceStale
			} else {
				f.must(err)
				binding = approved
				_, err = a.draftCoverage(term.Profile, binding.Spec.ID, binding.Digest, f.s.clock())
				f.must(err)
				action, target := gate.ApproveIssuanceTermBinding, binding.Approver
				if mode == "governor" {
					action, target = gate.AdministerInstallation, term.Profile.Proposer
				}
				revoked := 0
				for _, g := range f.s.admin.Grants() {
					if g.Terms.Action == action && g.SubjectID == target {
						f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
						revoked++
					}
				}
				if revoked == 0 {
					t.Fatal("fixture revoked no actual grant")
				}
			}
			_, err = a.draftCoverage(term.Profile, binding.Spec.ID, binding.Digest, f.s.clock())
			documentFailure(t, err, expected)
			if mode == "unknown" {
				recovered, err := approveBindingAs(f, decider, binding)
				f.must(err)
				_, err = a.draftCoverage(term.Profile, recovered.Spec.ID, recovered.Digest, f.s.clock())
				f.must(err)
			}
		})
	}
}
