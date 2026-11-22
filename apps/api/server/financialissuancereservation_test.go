package server

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
)

func proposeKindReservationForTest(f world, who actor, spec financialIssuanceProfileSpec) (financialKindReservation, error) {
	var r financialKindReservation
	err := inputAuthorityAs(f, who, func() error {
		var err error
		r, err = f.s.financialIssuance.proposeKindReservation(SampleWorkspace, f.s.answer.User(), "synthetic prospective kind", spec)
		return err
	})
	return r, err
}

func confirmKindReservationForTest(f world, who actor, r financialKindReservation) (financialKindReservation, error) {
	var out financialKindReservation
	err := inputAuthorityAs(f, who, func() error {
		var err error
		out, err = f.s.financialIssuance.confirmKindReservation(SampleWorkspace, r.Spec.ID, r.Digest, f.s.answer.User())
		return err
	})
	return out, err
}

func TestFinancialKindReservationIndependentAndMandatory(t *testing.T) {
	f, second, first, digest := issuanceSourceWorld(t)
	a := f.s.financialIssuance
	err := inputAuthorityAs(f, first, func() error {
		_, err := a.proposeProfile(SampleWorkspace, "issuance-final", digest, f.s.answer.User(), "synthetic profile configuration", issuanceSpec())
		return err
	})
	documentFailure(t, err, commands.EvidenceStale)
	r, err := proposeKindReservationForTest(f, first, issuanceSpec())
	f.must(err)
	_, err = confirmKindReservationForTest(f, first, r)
	documentFailure(t, err, commands.SelfApprovalDenied)
	confirmed, err := confirmKindReservationForTest(f, second, r)
	f.must(err)
	if confirmed.Digest != r.Digest || confirmed.State != "confirmed" {
		t.Fatal("reservation identity changed")
	}
	if _, err := a.documents.KindOf(SampleWorkspace, r.Kind.ID); !errors.Is(err, issuance.ErrNotFound) || len(a.profiles) != 0 || len(a.scopes) != 0 {
		t.Fatal("reservation created issuance effects")
	}
	r.Spec.Languages[0], r.Kind.Languages[0] = "changed", "changed"
	replay, err := proposeKindReservationForTest(f, first, issuanceSpec())
	f.must(err)
	if !reflect.DeepEqual(replay, confirmed) {
		t.Fatal("caller mutated reservation")
	}
	p, err := proposeIssuanceProfile(f, first, digest, issuanceSpec())
	f.must(err)
	if p.State != "proposed" || p.Confirmer != "" {
		t.Fatal("reservation approved financial profile")
	}
	consumed, err := proposeKindReservationForTest(f, first, issuanceSpec())
	f.must(err)
	if consumed.Digest != confirmed.Digest || consumed.ProfileID != p.Spec.ID || consumed.ProfileDigest != p.Digest {
		t.Fatal("missing stable consumption")
	}
	_, err = confirmIssuanceProfile(f, second, p)
	f.must(err)
	_, err = confirmKindReservationForTest(f, second, confirmed)
	f.must(err)
}

func TestFinancialKindReservationPublicationRecovery(t *testing.T) {
	for _, point := range []string{"before-propose", "after-propose", "before-confirm", "after-confirm"} {
		t.Run(point, func(t *testing.T) {
			f, second, first, _ := issuanceSourceWorld(t)
			a := f.s.financialIssuance
			var r financialKindReservation
			if point == "before-confirm" || point == "after-confirm" {
				var err error
				r, err = proposeKindReservationForTest(f, first, issuanceSpec())
				f.must(err)
			}
			f.s.fault = func(at string) bool { return at == "issuance-reservation-"+point }
			var err error
			if r.Digest == "" {
				_, err = proposeKindReservationForTest(f, first, issuanceSpec())
			} else {
				_, err = confirmKindReservationForTest(f, second, r)
			}
			documentFailure(t, err, commands.DependencyUnavailable)
			f.s.fault = nil
			saved, exists := a.kindReservations[issuanceSpec().ID]
			if point == "before-propose" && exists || point != "before-propose" && !exists {
				t.Fatal("wrong publication ownership")
			}
			if point == "before-confirm" && saved.State != "proposed" || point == "after-confirm" && saved.State != "confirmed" {
				t.Fatal("wrong confirmation publication")
			}
			r, err = proposeKindReservationForTest(f, first, issuanceSpec())
			f.must(err)
			_, err = confirmKindReservationForTest(f, second, r)
			f.must(err)
			if len(a.profiles) != 0 || len(a.scopes) != 0 {
				t.Fatal("unexpected financial effects")
			}
		})
	}
}

func TestFinancialKindReservationConsumptionIntegrity(t *testing.T) {
	for _, mode := range []string{"orphan", "reverse_digest", "profile_reservation", "scope", "spec", "lookup"} {
		t.Run(mode, func(t *testing.T) {
			f, second, first, digest := issuanceSourceWorld(t)
			p, err := proposeIssuanceProfile(f, first, digest, issuanceSpec())
			f.must(err)
			a := f.s.financialIssuance
			r := a.kindReservations[p.Spec.ID]
			switch mode {
			case "orphan":
				delete(a.profiles, p.Spec.ID)
			case "reverse_digest":
				r.ProfileDigest = "changed"
				a.kindReservations[p.Spec.ID] = r
			case "profile_reservation":
				p.ReservationDigest = "changed"
				a.profiles[p.Spec.ID] = p
			case "scope":
				delete(a.scopes, p.Source.Basis.Subject.Target)
			case "spec":
				r.Spec.Zone = "Asia/Tokyo"
				a.kindReservations[p.Spec.ID] = r
			case "lookup":
				f.s.fault = func(at string) bool { return at == "issuance-reservation-kind-lookup" }
			}
			_, err = proposeKindReservationForTest(f, first, issuanceSpec())
			expected := commands.EvidenceStale
			if mode == "lookup" {
				expected = commands.DependencyUnavailable
			}
			documentFailure(t, err, expected)
			_, err = confirmIssuanceProfile(f, second, p)
			documentFailure(t, err, expected)
		})
	}
}

func TestFinancialKindReservationBeforeFinancialSource(t *testing.T) {
	f := setup(t, SeedSample).ready()
	a := f.s.financialIssuance
	first, second := f.as(AdminLogin), f.as(Admin2Login)
	if _, err := f.s.approvals.RequestOf(SampleWorkspace, "issuance-final"); err == nil {
		t.Fatal("unexpected final approval")
	}
	r, err := proposeKindReservationForTest(f, first, issuanceSpec())
	f.must(err)
	r, err = confirmKindReservationForTest(f, second, r)
	f.must(err)
	if len(a.profiles) != 0 || len(a.scopes) != 0 || r.ProfileID != "" {
		t.Fatal("prospective reservation required financial source")
	}
	if _, err := a.documents.KindOf(SampleWorkspace, r.Kind.ID); !errors.Is(err, issuance.ErrNotFound) {
		t.Fatal("prospective reservation created kind")
	}
	for _, g := range f.s.admin.Grants() {
		if g.SubjectID == SampleAdmin2 && g.Terms.Action == gate.AdministerInstallation {
			f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
		}
	}
	_, err = proposeKindReservationForTest(f, first, issuanceSpec())
	documentFailure(t, err, commands.PermissionDenied)
}

func TestFinancialKindReservationUnknownBeforeModuleEffect(t *testing.T) {
	f, _, first, digest := issuanceSourceWorld(t)
	a := f.s.financialIssuance
	kind, err := issuanceProfileKind(issuanceSpec())
	f.must(err)
	for i := 0; i < financialIssuanceLimit; i++ {
		kind.ID = fmt.Sprintf("foreign-kind-%d", i)
		_, err := a.documents.Define(SampleWorkspace, kind, issuance.Actor{ID: SampleAdmin, Kind: issuance.KindHuman})
		f.must(err)
	}
	_, err = proposeIssuanceProfile(f, first, digest, issuanceSpec())
	if !errors.Is(err, issuance.ErrLimit) {
		t.Fatalf("expected module kind capacity failure: %v", err)
	}
	p := a.profiles[issuanceSpec().ID]
	if p.State != "registration_unknown" || a.scopes[p.Source.Basis.Subject.Target] != p.Spec.ID {
		t.Fatal("lost failed registration ownership")
	}
	r, err := proposeKindReservationForTest(f, first, issuanceSpec())
	f.must(err)
	if r.ProfileID != p.Spec.ID || r.ProfileDigest != p.Digest {
		t.Fatal("lost consumption before module effect")
	}
	if _, err := a.documents.KindOf(SampleWorkspace, p.Kind.ID); !errors.Is(err, issuance.ErrNotFound) {
		t.Fatal("expected actual absent kind")
	}
	f.s.fault = func(at string) bool { return at == "issuance-reservation-kind-lookup" }
	_, err = proposeKindReservationForTest(f, first, issuanceSpec())
	documentFailure(t, err, commands.DependencyUnavailable)
	f.s.fault = nil
	_, err = proposeKindReservationForTest(f, first, issuanceSpec())
	f.must(err)
}

func TestFinancialKindReservationForeignAndCapacity(t *testing.T) {
	f := setup(t, SeedSample).ready()
	a := f.s.financialIssuance
	first := f.as(AdminLogin)
	kind, err := issuanceProfileKind(issuanceSpec())
	f.must(err)
	_, err = a.documents.Define(SampleWorkspace, kind, issuance.Actor{ID: SampleAdmin, Kind: issuance.KindHuman})
	f.must(err)
	_, err = proposeKindReservationForTest(f, first, issuanceSpec())
	documentFailure(t, err, commands.EvidenceStale)
	for i := 0; i < financialIssuanceLimit; i++ {
		spec := issuanceSpec()
		spec.ID = fmt.Sprintf("reservation-%d", i)
		_, err := proposeKindReservationForTest(f, first, spec)
		f.must(err)
	}
	spec := issuanceSpec()
	spec.ID = "over-capacity"
	_, err = proposeKindReservationForTest(f, first, spec)
	documentFailure(t, err, commands.PrerequisiteFailed)
}

func TestFinancialKindReservationOrdinaryConsumerRejectsLostAssociation(t *testing.T) {
	f, member, p := termProfileWorld(t)
	a := f.s.financialIssuance
	r := a.kindReservations[p.Spec.ID]
	r.ProfileDigest = "changed"
	a.kindReservations[p.Spec.ID] = r
	_, err := consumeTermProfile(f, member, p)
	documentFailure(t, err, commands.EvidenceStale)
	if a.profiles[p.Spec.ID].Digest != p.Digest || a.scopes[p.Source.Basis.Subject.Target] != p.Spec.ID {
		t.Fatal("refusal released profile ownership")
	}
}
