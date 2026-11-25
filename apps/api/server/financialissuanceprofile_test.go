package server

import (
	"errors"
	"reflect"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
	"github.com/myblueton-sw/hybrid_billing/modules/receivables"
)

func issuanceSpec() financialIssuanceProfileSpec {
	return financialIssuanceProfileSpec{ID: "profile-1", Mode: "erp_issuance", Profile: "synthetic_erp_document_v1", Region: "US", Tax: "synthetic-tax", Zone: "UTC", Languages: []string{"ko", "en", "ar"}}
}

func proposeIssuanceProfile(f world, who actor, digest string, spec financialIssuanceProfileSpec) (financialIssuanceProfile, error) {
	if _, ok := f.s.financialIssuance.kindReservations[spec.ID]; !ok {
		r, err := proposeKindReservationForTest(f, f.as(AdminLogin), spec)
		if err != nil {
			return financialIssuanceProfile{}, err
		}
		if _, err := confirmKindReservationForTest(f, f.as(Admin3Login), r); err != nil {
			return financialIssuanceProfile{}, err
		}
	}
	var p financialIssuanceProfile
	err := inputAuthorityAs(f, who, func() error {
		var err error
		p, err = f.s.financialIssuance.proposeProfile(SampleWorkspace, "issuance-final", digest, f.s.answer.User(), "synthetic profile configuration", spec)
		return err
	})
	return p, err
}

func confirmIssuanceProfile(f world, who actor, p financialIssuanceProfile) (financialIssuanceProfile, error) {
	var out financialIssuanceProfile
	err := inputAuthorityAs(f, who, func() error {
		var err error
		out, err = f.s.financialIssuance.confirmProfile(SampleWorkspace, p.Spec.ID, p.Digest, f.s.answer.User())
		return err
	})
	return out, err
}

func TestFinancialIssuanceProfileIndependentConfirmation(t *testing.T) {
	f, second, first, digest := issuanceSourceWorld(t)
	before, err := f.s.approvals.RequestOf(SampleWorkspace, "issuance-final")
	f.must(err)
	p, err := proposeIssuanceProfile(f, first, digest, issuanceSpec())
	f.must(err)
	if p.State != "proposed" || p.Kind.Class != issuance.ClassERP || p.Kind.Statutory || !reflect.DeepEqual(p.Spec.Languages, []string{"ar", "en", "ko"}) {
		t.Fatalf("unexpected profile: %+v", p)
	}
	_, err = confirmIssuanceProfile(f, first, p)
	documentFailure(t, err, commands.SelfApprovalDenied)
	confirmed, err := confirmIssuanceProfile(f, second, p)
	f.must(err)
	if confirmed.State != "confirmed" || confirmed.Confirmer != SampleAdmin3 {
		t.Fatal("independent confirmation missing")
	}
	replay, err := confirmIssuanceProfile(f, second, p)
	f.must(err)
	if !reflect.DeepEqual(replay, confirmed) {
		t.Fatal("confirmation replay changed")
	}
	confirmed.Spec.Languages[0] = "changed"
	confirmed.Source.Basis.Rows[0].Net = "999"
	confirmed.Source.Basis.Subject.Digests["result"] = "changed"
	again, err := proposeIssuanceProfile(f, first, digest, issuanceSpec())
	f.must(err)
	if !reflect.DeepEqual(again, replay) {
		t.Fatal("caller mutated retained profile")
	}
	other := issuanceSpec()
	other.ID = "competing-profile"
	_, err = proposeIssuanceProfile(f, first, digest, other)
	documentFailure(t, err, commands.PrerequisiteFailed)
	after, err := f.s.approvals.RequestOf(SampleWorkspace, "issuance-final")
	f.must(err)
	if !reflect.DeepEqual(before, after) || len(after.Executed) != 0 {
		t.Fatal("configuration consumed final approval")
	}
}

func TestFinancialIssuanceProfileRegistrationRecovery(t *testing.T) {
	for _, point := range []string{"before-register", "after-register", "after-proposal-publication"} {
		t.Run(point, func(t *testing.T) {
			f, second, first, digest := issuanceSourceWorld(t)
			a := f.s.financialIssuance
			f.s.fault = func(at string) bool { return at == "issuance-profile-"+point }
			_, err := proposeIssuanceProfile(f, first, digest, issuanceSpec())
			documentFailure(t, err, commands.DependencyUnavailable)
			f.s.fault = nil
			kind, err := issuanceProfileKind(issuanceSpec())
			f.must(err)
			v, kindErr := a.documents.KindOf(SampleWorkspace, kind.ID)
			if point == "before-register" {
				if len(a.profiles) != 0 || len(a.scopes) != 0 || !errors.Is(kindErr, issuance.ErrNotFound) {
					t.Fatal("prewrite failure left effects")
				}
			} else {
				f.must(kindErr)
				expected := "registration_unknown"
				if point == "after-proposal-publication" {
					expected = "proposed"
				}
				if a.profiles["profile-1"].State != expected || v.State != issuance.StateProposed || len(a.scopes) != 1 {
					t.Fatal("lost registration ownership")
				}
				other := issuanceSpec()
				other.ID = "competing"
				_, err = proposeIssuanceProfile(f, first, digest, other)
				documentFailure(t, err, commands.PrerequisiteFailed)
				_, err = proposeIssuanceProfile(f, second, digest, issuanceSpec())
				documentFailure(t, err, commands.EvidenceStale)
			}
			reserved, err := proposeKindReservationForTest(f, first, issuanceSpec())
			f.must(err)
			if point == "before-register" && reserved.ProfileID != "" || point != "before-register" && reserved.ProfileID != "profile-1" {
				t.Fatal("wrong consumption at interruption")
			}
			p, err := proposeIssuanceProfile(f, first, digest, issuanceSpec())
			f.must(err)
			if p.State != "proposed" || len(a.profiles) != 1 || len(a.scopes) != 1 {
				t.Fatal("registration did not reconcile")
			}
		})
	}
}

func TestFinancialIssuanceProfileConfirmationRecovery(t *testing.T) {
	for _, point := range []string{"before-confirm", "after-confirm", "after-confirmation-publication"} {
		t.Run(point, func(t *testing.T) {
			f, second, first, digest := issuanceSourceWorld(t)
			p, err := proposeIssuanceProfile(f, first, digest, issuanceSpec())
			f.must(err)
			f.s.fault = func(at string) bool { return at == "issuance-profile-"+point }
			_, err = confirmIssuanceProfile(f, second, p)
			documentFailure(t, err, commands.DependencyUnavailable)
			f.s.fault = nil
			saved := f.s.financialIssuance.profiles[p.Spec.ID]
			v, err := f.s.financialIssuance.documents.KindOf(SampleWorkspace, p.Kind.ID)
			f.must(err)
			if point == "before-confirm" {
				if saved.State != "proposed" || saved.Confirmer != "" || v.State != issuance.StateProposed {
					t.Fatal("prewrite confirmation changed state")
				}
			} else {
				expected := "confirmation_unknown"
				if point == "after-confirmation-publication" {
					expected = "confirmed"
				}
				if saved.State != expected || saved.Confirmer != SampleAdmin3 || v.State != issuance.StateActive || v.Approver != SampleAdmin3 {
					t.Fatal("lost confirmation ownership")
				}
			}
			if point != "before-confirm" {
				_, wrongErr := confirmIssuanceProfile(f, f.as(Admin2Login), p)
				documentFailure(t, wrongErr, commands.PermissionDenied)
			}
			_, err = proposeKindReservationForTest(f, first, issuanceSpec())
			f.must(err)
			confirmed, err := confirmIssuanceProfile(f, second, p)
			f.must(err)
			if confirmed.State != "confirmed" || confirmed.Digest != p.Digest {
				t.Fatal("confirmation did not reconcile")
			}
		})
	}
}

func TestFinancialIssuanceProfileRejectsUnownedKind(t *testing.T) {
	f, _, first, digest := issuanceSourceWorld(t)
	kind, err := issuanceProfileKind(issuanceSpec())
	f.must(err)
	_, err = f.s.financialIssuance.documents.Define(SampleWorkspace, kind, issuance.Actor{ID: SampleAdmin, Kind: issuance.KindHuman})
	f.must(err)
	_, err = proposeIssuanceProfile(f, first, digest, issuanceSpec())
	documentFailure(t, err, commands.EvidenceStale)
	if len(f.s.financialIssuance.profiles) != 0 || len(f.s.financialIssuance.scopes) != 0 {
		t.Fatal("adopted unowned kind")
	}
}

func TestFinancialIssuanceProfileChangedSourceRetainsLock(t *testing.T) {
	for _, mode := range []string{"proposer", "source"} {
		t.Run(mode, func(t *testing.T) {
			f, second, first, digest := issuanceSourceWorld(t)
			p, err := proposeIssuanceProfile(f, first, digest, issuanceSpec())
			f.must(err)
			f.s.fault = func(at string) bool { return at == "issuance-profile-after-confirm" }
			_, err = confirmIssuanceProfile(f, second, p)
			documentFailure(t, err, commands.DependencyUnavailable)
			f.s.fault = nil
			if mode == "proposer" {
				removed := 0
				for _, g := range f.s.admin.Grants() {
					if g.SubjectID == SampleAdmin && g.Terms.Action == gate.AdministerInstallation {
						f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
						removed++
					}
				}
				if removed == 0 {
					t.Fatal("no actual proposer grant revoked")
				}
			} else {
				key := [2]string{SampleWorkspace, "public-result"}
				app := f.s.documentERP.resultApplications[key]
				app.receipt = "changed"
				f.s.documentERP.resultApplications[key] = app
			}
			_, err = confirmIssuanceProfile(f, second, p)
			if mode == "proposer" {
				documentFailure(t, err, commands.PermissionDenied)
			} else {
				documentFailure(t, err, commands.EvidenceStale)
			}
			saved := f.s.financialIssuance.profiles[p.Spec.ID]
			if saved.State != "confirmation_unknown" || f.s.financialIssuance.scopes[p.Source.Basis.Subject.Target] != p.Spec.ID {
				t.Fatal("refusal released original ownership")
			}
		})
	}
}

func TestFinancialIssuanceProfileClosedShape(t *testing.T) {
	for _, mutate := range []func(*financialIssuanceProfileSpec){
		func(p *financialIssuanceProfileSpec) { p.Mode = "self_issuance" },
		func(p *financialIssuanceProfileSpec) { p.Profile = "production" },
		func(p *financialIssuanceProfileSpec) { p.Region = "us" },
		func(p *financialIssuanceProfileSpec) { p.Tax = "" },
		func(p *financialIssuanceProfileSpec) { p.Zone = "Local" },
		func(p *financialIssuanceProfileSpec) { p.Languages = []string{"en", "en"} },
		func(p *financialIssuanceProfileSpec) { p.Languages = []string{"xx"} },
	} {
		p := issuanceSpec()
		mutate(&p)
		_, err := issuanceProfileKind(p)
		documentFailure(t, err, commands.InvalidRequest)
	}
}

func TestFinancialIssuanceEmptyAssembly(t *testing.T) {
	f := setup(t, SeedSample).ready()
	a := f.s.financialIssuance
	if a == nil || a.fields == nil || a.terms == nil || a.documents == nil || len(a.profiles) != 0 || len(a.scopes) != 0 {
		t.Fatal("missing empty actual registries")
	}
	_, err := a.fields.RulesOf(SampleWorkspace)
	if !errors.Is(err, protected.ErrNotFound) {
		t.Fatal("protected rules were seeded")
	}
	_, err = a.terms.Term(SampleWorkspace, "default", 1)
	if !errors.Is(err, receivables.ErrNotFound) {
		t.Fatal("default term exists")
	}
	_, err = a.documents.KindOf(SampleWorkspace, "default")
	if !errors.Is(err, issuance.ErrNotFound) {
		t.Fatal("default kind exists")
	}
	_, err = (unavailableIssuanceContent{}).Detail(SampleWorkspace, "detail")
	documentFailure(t, err, commands.UnsupportedCapability)
	_, err = (unavailableIssuanceContent{}).Render(issuance.Snapshot{})
	documentFailure(t, err, commands.UnsupportedCapability)
	none := setup(t, SeedNone).ready()
	if none.s.financialIssuance != nil {
		t.Fatal("unconfigured installation has issuance adapter")
	}
	_, err = newFinancialIssuance(none.s)
	documentFailure(t, err, commands.UnsupportedCapability)
}
