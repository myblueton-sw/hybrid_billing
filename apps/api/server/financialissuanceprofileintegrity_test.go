package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/protectedauthority"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestOwnedConfirmedProfileEmbeddedIntegrity(t *testing.T) {
	for _, mode := range []string{"inner-digest", "repaired-outer", "equal-governors", "empty-proposer", "empty-confirmer"} {
		t.Run(mode, func(t *testing.T) {
			f, _, p := termProfileWorld(t)
			a := f.s.financialIssuance
			changed := copyIssuanceProfile(p)
			switch mode {
			case "inner-digest":
				changed.Source.Digest = strings.Repeat("a", 64)
			case "repaired-outer":
				changed.Source.Basis.Rows[0].Net = "999"
				immutable := copyIssuanceProfile(changed)
				immutable.Digest, immutable.State, immutable.Confirmer = "", "", ""
				var err error
				changed.Digest, err = financialEvidenceDigest("issuance_profile", immutable)
				f.must(err)
				r := a.kindReservations[p.Spec.ID]
				r.ProfileDigest = changed.Digest
				a.kindReservations[p.Spec.ID] = r
			case "equal-governors":
				changed.Confirmer = changed.Proposer
			case "empty-proposer":
				changed.Proposer = ""
			case "empty-confirmer":
				changed.Confirmer = ""
			}
			a.profiles[p.Spec.ID] = changed
			_, err := a.ownedConfirmedProfile(SampleWorkspace, p.Spec.ID, changed.Digest)
			documentFailure(t, err, commands.EvidenceStale)
		})
	}
}

func TestOwnedConfirmedProfileDoesNotReopenFinancialSource(t *testing.T) {
	f, member, p := termProfileWorld(t)
	a := f.s.financialIssuance
	got, err := a.ownedConfirmedProfile(SampleWorkspace, p.Spec.ID, p.Digest)
	f.must(err)
	got.Source.Basis.Rows[0].Net = "999"
	got.Spec.Languages[0] = "changed"
	got, err = a.ownedConfirmedProfile(SampleWorkspace, p.Spec.ID, p.Digest)
	f.must(err)
	if !reflect.DeepEqual(got, p) {
		t.Fatal("historical profile aliases retained evidence")
	}
	key := [2]string{SampleWorkspace, "public-result"}
	app := f.s.documentERP.resultApplications[key]
	app.receipt = "changed"
	f.s.documentERP.resultApplications[key] = app
	historical, err := a.ownedConfirmedProfile(SampleWorkspace, p.Spec.ID, p.Digest)
	f.must(err)
	if !reflect.DeepEqual(historical, p) {
		t.Fatal("historical source changed")
	}
	_, err = consumeTermProfile(f, member, p)
	documentFailure(t, err, commands.EvidenceStale)
}

func TestProtectedProfileStillRequiresCurrentSource(t *testing.T) {
	f, requester, _, _, spec := protectedChangeWorld(t, 24*time.Hour, 1)
	key := [2]string{SampleWorkspace, "public-result"}
	app := f.s.documentERP.resultApplications[key]
	app.receipt = "changed"
	f.s.documentERP.resultApplications[key] = app
	_, err := f.s.financialIssuance.ownedConfirmedProfile(SampleWorkspace, spec.Profile, spec.ProfileDigest)
	f.must(err)
	err = inputAuthorityAs(f, requester, func() error {
		_, err := f.s.financialIssuance.protectedProfile(SampleWorkspace, spec.Profile, spec.ProfileDigest, f.s.answer.User(), protectedauthority.RequestChange)
		return err
	})
	documentFailure(t, err, commands.EvidenceStale)
}

func TestDraftProfileAuthorizationBeforeLookup(t *testing.T) {
	f, member, p := termProfileWorld(t)
	check := func(profile, digest string) error {
		return inputAuthorityAs(f, member, func() error {
			_, err := f.s.financialIssuance.draftProfile(SampleWorkspace, profile, digest, SampleMember, draftauthority.Submit)
			return err
		})
	}
	documentFailure(t, check("missing", "invalid"), commands.PermissionDenied)
	grantDraftCapability(f, draftauthority.Submit)
	documentFailure(t, check("missing", "invalid"), commands.PermissionDenied)
	grantDraftCapability(f, draftauthority.Inspect)
	f.must(check(p.Spec.ID, p.Digest))
	documentFailure(t, check("missing", "invalid"), commands.EvidenceStale)
	key := [2]string{SampleWorkspace, "public-result"}
	app := f.s.documentERP.resultApplications[key]
	app.receipt = "changed"
	f.s.documentERP.resultApplications[key] = app
	documentFailure(t, check(p.Spec.ID, p.Digest), commands.EvidenceStale)
}
