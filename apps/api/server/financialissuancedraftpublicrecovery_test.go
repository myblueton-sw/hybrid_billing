package server

import (
	"reflect"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
)

func TestFinancialDraftPublicCleanFailureBesideOlderUncertainty(t *testing.T) {
	f, member, reviewer, spec, binding := configuredDraftWorld(t)
	a := f.s.financialIssuance
	f.s.fault = func(at string) bool { return at == "issuance-draft-after-intent" }
	first := draftRun(f, member, "issuance_draft.submit", spec.ID, draftSubmitArguments(spec))
	if first.status != 500 {
		t.Fatal(first.raw)
	}
	before := copyFinancialDraft(a.drafts[spec.ID])
	revision := f.s.Revision()
	second := draftRun(f, member, "issuance_draft.submit", spec.ID, draftSubmitArguments(spec))
	f.code(t, "unchanged old uncertainty is not new effect", second, 503, string(commands.DependencyUnavailable))
	if second.body["operation_id"] != nil || f.s.Revision() != revision || !reflect.DeepEqual(a.drafts[spec.ID], before) {
		t.Fatal("invented new submission effect", second.raw)
	}
	f.s.fault = nil
	f.ok(t, "recover", draftRun(f, member, "issuance_draft.submit", spec.ID, draftSubmitArguments(spec)), "submitted", "")
	p := a.drafts[spec.ID]
	f.ok(t, "prepare", draftRun(f, reviewer, "issuance_draft.prepare_approval", "old-unknown-review", draftPrepareArguments(p, binding)), "prepared", "")
	r := a.draftReviews["old-unknown-review"]
	args := map[string]string{"digest": r.Digest}
	f.s.fault = func(at string) bool { return at == "issuance-draft-review-after-intent" }
	first = draftRun(f, reviewer, "issuance_draft.approve", r.ID, args)
	if first.status != 500 {
		t.Fatal(first.raw)
	}
	reviewBefore := copyDraftReview(a.draftReviews[r.ID])
	revision = f.s.Revision()
	second = draftRun(f, reviewer, "issuance_draft.approve", r.ID, args)
	f.code(t, "unchanged review uncertainty is not new effect", second, 503, string(commands.DependencyUnavailable))
	if second.body["operation_id"] != nil || f.s.Revision() != revision || !reflect.DeepEqual(a.draftReviews[r.ID], reviewBefore) {
		t.Fatal("invented new review effect", second.raw)
	}
	f.s.fault = nil
}

func TestFinancialDraftPublicKnownDriftRefusal(t *testing.T) {
	f, member, reviewer, spec, binding := configuredDraftWorld(t)
	p, err := submitDraftAs(f, member, spec)
	f.must(err)
	prepared, err := prepareDraftReviewAs(f, reviewer, "public-drift", p, binding)
	f.must(err)
	f.advance(16 * time.Hour)
	renewInvoiceAuthorities(t, f, "-public-drift")
	reviewer = f.as("configured-draft-reviewer@example.invalid")
	want := f.s.Next()
	result := draftRun(f, reviewer, "issuance_draft.approve", prepared.Review.ID, map[string]string{"digest": prepared.Review.Digest})
	f.ok(t, "explicit recheck records known refusal", result, "refused", "")
	if f.s.Revision() != want || field(result, "receipt") != "review_attempt" || f.s.financialIssuance.draftReviews[prepared.Review.ID].Refusal != issuance.ReasonCoverageMismatch {
		t.Fatal(result.raw)
	}
	actual, err := f.s.financialIssuance.documents.DraftOf(SampleWorkspace, p.Spec.ID)
	f.must(err)
	if actual.Approver != "" {
		t.Fatal("known refusal approved module")
	}
}

func TestFinancialDraftPublicPinnedActorAndInspectBeforeCache(t *testing.T) {
	f, member, reviewer, spec, binding := configuredDraftWorld(t)
	grantDraftReviewer(f, "public-member-review-", SampleMember)
	member = f.as(MemberLogin)
	p, err := submitDraftAs(f, member, spec)
	f.must(err)
	args := draftPrepareArguments(p, binding)
	body := chargeBody(f, "cached-review-envelope", "cached-review", args)
	first := f.call("issuance_draft.prepare_approval", body, reviewer.cookie, reviewer.csrf)
	f.ok(t, "prepare", first, "prepared", "")
	revision := f.s.Revision()
	other := f.call("issuance_draft.prepare_approval", body, member.cookie, member.csrf)
	f.code(t, "another authorized actor cannot replay review", other, 403, string(commands.PermissionDenied))
	for _, grant := range f.s.admin.Grants() {
		if grant.SubjectID == "configured-draft-reviewer" && grant.Terms.Kind == draftauthority.ResourceKind && grant.Terms.Action == gate.ViewCost {
			f.must(f.s.admin.RevokeGrant(grant.ID, grant.Version))
		}
	}
	lost := f.call("issuance_draft.prepare_approval", body, reviewer.cookie, reviewer.csrf)
	f.code(t, "Inspect before cached prepare receipt", lost, 403, string(commands.PermissionDenied))
	if lost.body["fields"] != nil || other.body["fields"] != nil || f.s.Revision() != revision {
		t.Fatal("cache bypassed actor or inspection")
	}
}

func TestFinancialDraftPublicLegacyMutationClosure(t *testing.T) {
	f, _, reviewer, p, binding := draftApprovalWorld(t)
	prepared, err := prepareDraftReviewAs(f, reviewer, "legacy-public-review", p, binding)
	f.must(err)
	revision := f.s.Revision()
	for _, test := range []struct {
		kind, target string
		args         map[string]string
	}{
		{"prepare_approval", prepared.Review.ID, draftPrepareArguments(p, binding)},
		{"approve", prepared.Review.ID, map[string]string{"digest": prepared.Review.Digest}},
	} {
		result := draftRun(f, reviewer, "issuance_draft."+test.kind, test.target, test.args)
		f.code(t, "legacy is public read only", result, 501, string(commands.UnsupportedCapability))
		if result.body["fields"] != nil || result.body["operation_id"] != nil || f.s.Revision() != revision {
			t.Fatal(result.raw)
		}
	}
}
