package server

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
)

func configuredDraftWorld(t *testing.T) (world, actor, actor, financialDraftSpec, financialTermBinding) {
	t.Helper()
	var setupWorld configuredSetupWorld
	var binding financialTermBinding
	f, _, _, digest := issuanceSourceWorld(t, func(f world) {
		setupWorld = configuredSetupIn(t, f)
		_, binding = completeConfiguredSetup(t, setupWorld)
		if _, err := f.s.approvals.RequestOf(SampleWorkspace, "issuance-final"); !errors.Is(err, approvals.ErrNotFound) {
			t.Fatal("setup followed finance")
		}
	})
	grantDraftCapability(f, draftauthority.Submit)
	grantDraftCapability(f, draftauthority.Inspect)
	login := activateProtectedMember(f, "configured-draft-reviewer")
	grantDraftReviewer(f, "configured-review-", "configured-draft-reviewer")
	spec := financialDraftSpec{ID: "configured-draft-a", Language: "en", Application: &financialDraftApplicationRequest{Schema: "hb_issuance_application_request_v1", Configuration: configurationReference(setupWorld.configuration), FinalRequest: "issuance-final", FinalDigest: digest}}
	return f, f.as(MemberLogin), f.as(login), spec, binding
}

func TestFinancialConfiguredDraftTwoInvoicesReuseSetup(t *testing.T) {
	f, member, reviewer, spec, binding := configuredDraftWorld(t)
	a := f.s.financialIssuance
	first, err := submitDraftAs(f, member, spec)
	f.must(err)
	review, err := prepareDraftReviewAs(f, reviewer, "configured-review-a", first, binding)
	f.must(err)
	approved, err := approveDraftAs(f, reviewer, review.Review)
	f.must(err)
	if first.Application == nil || !reflect.DeepEqual(first.Profile, financialIssuanceProfile{}) || first.Draft.Total != "16" || approved.Review.State != "approved" || len(approved.Review.References) != 3 {
		t.Fatal("first configured draft lacks actual approval")
	}
	before, err := f.s.approvals.RequestOf(SampleWorkspace, "issuance-final")
	f.must(err)
	e := approvals.Execution{ID: before.Subject.ID, Request: before.Subject.ID, Stage: before.Subject.Stage, Target: before.Subject.Target, Digests: before.Subject.Digests, Amount: before.Subject.Amount, Currency: before.Subject.Currency}
	f.must(f.s.approvals.CheckCurrent(SampleWorkspace, e, approvals.Actor{ID: SampleMember, Kind: approvals.KindHuman}))
	channels := map[string]financialChannelRecord{}
	changes := map[string]financialChangeRecord{}
	for id, p := range a.channels {
		channels[id] = copyChannelRecord(p)
	}
	for id, p := range a.changes {
		changes[id] = copyChange(p)
	}
	f.advance(24*time.Hour + time.Minute)
	renewInvoiceAuthorities(t, f, "-next-invoice")
	err = f.s.approvals.CheckCurrent(SampleWorkspace, e, approvals.Actor{ID: SampleMember, Kind: approvals.KindHuman})
	var refusal approvals.Refusal
	if !errors.As(err, &refusal) || refusal.Reason != approvals.ReasonExpired {
		t.Fatalf("first financial approval remains live: %v", err)
	}
	_, err = submitDraftAs(f, f.as(MemberLogin), spec)
	documentFailure(t, err, commands.PrerequisiteFailed)
	historical, err := approveDraftAs(f, f.as("configured-draft-reviewer@example.invalid"), review.Review)
	f.must(err)
	if !reflect.DeepEqual(historical, approved) || historical.Eligibility != "unchecked" {
		t.Fatal("historical approval refreshed expired source")
	}
	from := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	candidate, candidateDigest := captureInvoiceInWorld(t, f, f.as(MemberLogin), f.as(Admin3Login), "configured-b", "2026-11", from, to)
	final, finalDigest := finalizeInvoiceInWorld(t, f, "configured-b", "2026-11", candidate, candidateDigest)
	secondSpec := spec
	r := *spec.Application
	secondSpec.Application = &r
	secondSpec.ID = "configured-draft-b"
	r.FinalRequest, r.FinalDigest = final, finalDigest
	second, err := submitDraftAs(f, f.as(MemberLogin), secondSpec)
	f.must(err)
	secondReview, err := prepareDraftReviewAs(f, f.as("configured-draft-reviewer@example.invalid"), "configured-review-b", second, binding)
	f.must(err)
	secondApproved, err := approveDraftAs(f, f.as("configured-draft-reviewer@example.invalid"), secondReview.Review)
	f.must(err)
	if secondApproved.Review.State != "approved" || !reflect.DeepEqual(first.Application.Configuration, second.Application.Configuration) || first.Application.Digest == second.Application.Digest || first.Application.Source.Digest == second.Application.Source.Digest || first.Application.Source.Basis.Subject.Target == second.Application.Source.Basis.Subject.Target || second.Application.Source.Basis.Period != "2026-11" || !reflect.DeepEqual(secondApproved.Review.References, approved.Review.References) || secondApproved.Review.Coverage.Binding != approved.Review.Coverage.Binding {
		t.Fatal("next invoice failed exact setup reuse")
	}
	after, err := f.s.approvals.RequestOf(SampleWorkspace, "issuance-final")
	f.must(err)
	if !reflect.DeepEqual(before, after) || !reflect.DeepEqual(a.drafts[first.Spec.ID], first) || !reflect.DeepEqual(a.draftReviews[review.Review.ID], approved.Review) || !reflect.DeepEqual(a.channels, channels) || !reflect.DeepEqual(a.changes, changes) || !reflect.DeepEqual(a.bindingIntents[binding.Spec.ID], binding) || len(a.configurations) != 1 || len(a.configurationScopes) != 1 || len(a.channels) != 3 || len(a.changes) != 3 || len(a.termIntents) != 1 || len(a.bindingIntents) != 1 || len(a.profiles) != 0 || len(a.kindReservations) != 0 || len(a.drafts) != 2 {
		t.Fatal("next invoice altered history or duplicated setup")
	}
}
