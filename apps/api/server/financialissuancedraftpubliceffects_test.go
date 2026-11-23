package server

import (
	"reflect"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func TestFinancialDraftPublicSubmissionEffectBoundaries(t *testing.T) {
	for _, point := range []string{"before-submit", "after-intent", "after-submit", "after-submission-publication"} {
		t.Run(point, func(t *testing.T) {
			f, member, c, digest := applicationWorld(t)
			spec := financialDraftSpec{ID: "public-fault-draft", Language: "en", Application: &financialDraftApplicationRequest{Schema: "hb_issuance_application_request_v1", Configuration: configurationReference(c), FinalRequest: "issuance-final", FinalDigest: digest}}
			a := f.s.financialIssuance
			body := chargeBody(f, "public-fault-envelope", spec.ID, draftSubmitArguments(spec))
			before, want := f.s.Revision(), f.s.Next()
			f.s.fault = func(at string) bool { return at == "issuance-draft-"+point }
			result := f.call("issuance_draft.submit", body, member.cookie, member.csrf)
			f.s.fault = nil
			if point == "before-submit" {
				f.code(t, "clean before effect", result, 503, string(commands.DependencyUnavailable))
				if f.s.Revision() != before || result.body["operation_id"] != nil || len(a.drafts) != 0 {
					t.Fatal(result.raw)
				}
			} else {
				if result.status != 500 || result.body["operation_id"] == nil || result.body["next_action"] != string(commands.NextOperation) || f.s.Revision() != want {
					t.Fatal("effect failure was clean or advanced twice", result.raw)
				}
			}
			revision := f.s.Revision()
			cached := f.call("issuance_draft.submit", body, member.cookie, member.csrf)
			if point == "before-submit" {
				if cached.status != 200 || f.s.Revision() != want {
					t.Fatal(cached.raw)
				}
			} else if cached.status != result.status || cached.body["operation_id"] != result.body["operation_id"] || f.s.Revision() != revision {
				t.Fatal("changed same-envelope outcome", cached.raw)
			}
			f.ok(t, "fresh envelope recovers owned draft", draftRun(f, member, "issuance_draft.submit", spec.ID, draftSubmitArguments(spec)), "submitted", "")
			if len(a.drafts) != 1 || len(a.draftScopes) != 1 {
				t.Fatal("recovery duplicated economic intent")
			}
		})
	}
}

func TestFinancialDraftPublicApprovalEffectBoundaries(t *testing.T) {
	for _, point := range []string{"before-intent", "after-intent", "after-module", "after-publication"} {
		t.Run(point, func(t *testing.T) {
			f, member, reviewer, spec, binding := configuredDraftWorld(t)
			p, err := submitDraftAs(f, member, spec)
			f.must(err)
			prepared, err := prepareDraftReviewAs(f, reviewer, "public-effect-review", p, binding)
			f.must(err)
			a := f.s.financialIssuance
			args := map[string]string{"digest": prepared.Review.Digest}
			body := chargeBody(f, "public-review-envelope", prepared.Review.ID, args)
			before, want := f.s.Revision(), f.s.Next()
			f.s.fault = func(at string) bool { return at == "issuance-draft-review-"+point }
			result := f.call("issuance_draft.approve", body, reviewer.cookie, reviewer.csrf)
			f.s.fault = nil
			if point == "before-intent" {
				f.code(t, "clean review refusal", result, 503, string(commands.DependencyUnavailable))
				if f.s.Revision() != before || !reflect.DeepEqual(a.draftReviews[prepared.Review.ID], prepared.Review) || result.body["operation_id"] != nil {
					t.Fatal(result.raw)
				}
			} else if result.status != 500 || result.body["operation_id"] == nil || f.s.Revision() != want {
				t.Fatal("lost review uncertainty", result.raw)
			}
			revision := f.s.Revision()
			cached := f.call("issuance_draft.approve", body, reviewer.cookie, reviewer.csrf)
			if point == "before-intent" {
				if cached.status != 200 || f.s.Revision() != want {
					t.Fatal(cached.raw)
				}
			} else if cached.status != result.status || f.s.Revision() != revision {
				t.Fatal(cached.raw)
			}
			f.ok(t, "explicit same-business recovery", draftRun(f, reviewer, "issuance_draft.approve", prepared.Review.ID, args), "approved", "")
			if len(a.draftReviews) != 1 || len(a.draftReviewOrders[p.Spec.ID]) != 1 {
				t.Fatal("approval recovery created another review")
			}
		})
	}
}
