package server

import (
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func TestIssuanceDraftReviewRechecksSetupActors(t *testing.T) {
	for _, role := range []string{"submitter", "admitter", "requester", "verifier", "approver"} {
		t.Run(role, func(t *testing.T) {
			var j *journeySetup
			f, _, _, final := issuanceSourceWorld(t, func(f world) { j = completeJourneySetup(t, f, "") })
			reviewer := provisionJourneyDrafts(t, f, j)
			a := f.s.financialIssuance
			f.ok(t, "submit", draftRun(f, f.as(MemberLogin), "issuance_draft.submit", "recheck-draft", journeySubmission(j.configuration, "issuance-final", final)), "submitted", "")
			target := "journey-draft-" + role
			if role == "submitter" {
				target = SampleMember
			}
			suspended := f.run(f.as(AdminLogin), "identity.suspend_membership", target, nil)
			if suspended.status != 200 {
				t.Fatal(suspended.raw)
			}
			got := draftRun(f, f.as(reviewer), "issuance_draft.prepare_approval", "recheck-review", map[string]string{"draft": "recheck-draft", "draft_digest": a.drafts["recheck-draft"].Digest, "binding": j.binding, "binding_digest": j.bindingDigest})
			f.code(t, "prepare after the "+role+" lost access", got, 403, string(commands.PermissionDenied))
			if len(a.draftReviews) != 0 {
				t.Fatal("a review was retained")
			}
		})
	}
}
