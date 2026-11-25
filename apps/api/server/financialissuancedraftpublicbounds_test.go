package server

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func TestFinancialDraftPublicRetainedCorruptionIsEmpty(t *testing.T) {
	f, member, c, digest := applicationWorld(t)
	a := f.s.financialIssuance
	spec := financialDraftSpec{ID: "public-corrupt", Language: "en", Application: &financialDraftApplicationRequest{Schema: "hb_issuance_application_request_v1", Configuration: configurationReference(c), FinalRequest: "issuance-final", FinalDigest: digest}}
	original, err := submitDraftAs(f, member, spec)
	f.must(err)
	p := copyFinancialDraft(original)
	p.Draft.Total = "999"
	p.Digest, err = draftIntentDigest(p)
	f.must(err)
	a.drafts[spec.ID] = p
	revision := f.s.Revision()
	for _, entry := range []struct{ command, target string }{{"issuance_draft.inspect", spec.ID}, {"issuance_draft.list", SampleWorkspace}} {
		result := documentRead(f, member, entry.command, entry.target, map[string]string{})
		f.code(t, "corrupt amount refuses", result, 409, string(commands.EvidenceStale))
		if result.body["fields"] != nil {
			t.Fatal("corrupt retained amount disclosed partial data")
		}
	}
	actual, err := a.documents.DraftOf(SampleWorkspace, spec.ID)
	f.must(err)
	if !reflect.DeepEqual(actual, original.Module) || !reflect.DeepEqual(a.drafts[spec.ID], p) || f.s.Revision() != revision {
		t.Fatal("read repaired or adopted corrupt retained evidence")
	}
}

func TestFinancialDraftPublicProjectionBudget(t *testing.T) {
	f, member, c, digest := applicationWorld(t)
	a := f.s.financialIssuance
	spec := financialDraftSpec{ID: "public-budget", Language: "en", Application: &financialDraftApplicationRequest{Schema: "hb_issuance_application_request_v1", Configuration: configurationReference(c), FinalRequest: "issuance-final", FinalDigest: digest}}
	draft, err := submitDraftAs(f, member, spec)
	f.must(err)
	f.s.cfg.Commands.Value = 4095
	result := documentRead(f, member, "issuance_draft.inspect", spec.ID, map[string]string{})
	f.code(t, "unsupported read budget", result, 501, string(commands.UnsupportedCapability))
	if result.body["fields"] != nil || !reflect.DeepEqual(a.drafts[spec.ID], draft) {
		t.Fatal("unsupported projection leaked or changed record")
	}
	f.s.cfg.Commands.Value = 4096
	fields := map[string]string{}
	for i := 0; i < 17; i++ {
		fields[fmt.Sprintf("row_%02d", i)] = strings.Repeat("\\", 4096)
	}
	raw, err := json.Marshal(fields)
	f.must(err)
	if len(raw) <= 65536 {
		t.Fatal("fixture lacks escaped aggregate overflow")
	}
	documentFailure(t, a.boundedDraftFields(fields), commands.PrerequisiteFailed)
	fields = map[string]string{"value": strings.Repeat("a", 4097)}
	documentFailure(t, a.boundedDraftFields(fields), commands.PrerequisiteFailed)
	fields = map[string]string{}
	for i := 0; i < 65; i++ {
		fields[fmt.Sprintf("field_%02d", i)] = "value"
	}
	documentFailure(t, a.boundedDraftFields(fields), commands.PrerequisiteFailed)
}

func TestFinancialDraftPublicRejectsIncoherentRetainedState(t *testing.T) {
	for _, mode := range []string{"refused-submission-with-module", "refused-review-empty", "refused-review-arbitrary", "prepared-review-refusal"} {
		t.Run(mode, func(t *testing.T) {
			f, member, reviewer, spec, binding := configuredDraftWorld(t)
			a := f.s.financialIssuance
			draft, err := submitDraftAs(f, member, spec)
			f.must(err)
			var review financialDraftReview
			if mode == "refused-submission-with-module" {
				draft.State = "submission_refused"
				a.drafts[spec.ID] = copyFinancialDraft(draft)
			} else {
				prepared, err := prepareDraftReviewAs(f, reviewer, "corrupt-state-review", draft, binding)
				f.must(err)
				review = prepared.Review
				if mode == "prepared-review-refusal" {
					review.Refusal = "changed"
				} else {
					review.State = "refused"
					if mode == "refused-review-arbitrary" {
						review.Refusal = "untrusted-reason"
					}
				}
				a.draftReviews[review.ID] = copyDraftReview(review)
			}
			before := copyFinancialDraft(a.drafts[spec.ID])
			revision := f.s.Revision()
			for _, entry := range []struct{ command, target string }{{"issuance_draft.inspect", spec.ID}, {"issuance_draft.list", SampleWorkspace}} {
				result := documentRead(f, member, entry.command, entry.target, map[string]string{})
				f.code(t, "incoherent state refused", result, 409, string(commands.EvidenceStale))
				if result.body["fields"] != nil {
					t.Error("incoherent state disclosed fields")
				}
			}
			module, err := a.documents.DraftOf(SampleWorkspace, spec.ID)
			f.must(err)
			if !reflect.DeepEqual(a.drafts[spec.ID], before) || !reflect.DeepEqual(module, draft.Module) || review.ID != "" && !reflect.DeepEqual(a.draftReviews[review.ID], review) || f.s.Revision() != revision {
				t.Fatal("read changed corrupt state")
			}
		})
	}
}
