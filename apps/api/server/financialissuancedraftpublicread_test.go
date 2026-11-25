package server

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
)

func TestFinancialDraftPublicReadUnknownSubmission(t *testing.T) {
	for _, configured := range []bool{false, true} {
		name := "legacy"
		if configured {
			name = "configured"
		}
		t.Run(name, func(t *testing.T) {
			var f world
			var member actor
			var spec financialDraftSpec
			if configured {
				var c financialIssuanceConfiguration
				var digest string
				f, member, c, digest = applicationWorld(t)
				spec = financialDraftSpec{ID: "unknown-public-draft", Language: "en", Application: &financialDraftApplicationRequest{Schema: "hb_issuance_application_request_v1", Configuration: configurationReference(c), FinalRequest: "issuance-final", FinalDigest: digest}}
			} else {
				f, member, spec, _ = draftSubmissionWorld(t)
			}
			a := f.s.financialIssuance
			f.s.fault = func(at string) bool { return at == "issuance-draft-after-intent" }
			_, err := submitDraftAs(f, member, spec)
			documentFailure(t, err, commands.DependencyUnavailable)
			f.s.fault = nil
			before := copyFinancialDraft(a.drafts[spec.ID])
			revision := f.s.Revision()
			read := documentRead(f, member, "issuance_draft.inspect", spec.ID, map[string]string{})
			if read.status != 200 || field(read, "module_observation") != "missing" || field(read, "submission_state") != "submission_unknown" || field(read, "current_eligibility") != "unchecked" || field(read, "total") != "16" || field(read, "scale") != "2" || field(read, "line_count") != "4" {
				t.Fatal(read.raw)
			}
			if configured {
				var recovery struct {
					Operation, OriginalActor, BusinessID, State, Permission string
					Arguments                                               map[string]string
				}
				var raw map[string]json.RawMessage
				f.must(json.Unmarshal([]byte(field(read, "recovery_submit")), &raw))
				f.must(json.Unmarshal(raw["arguments"], &recovery.Arguments))
				if recovery.Arguments["configuration"] != spec.Application.Configuration.ID || recovery.Arguments["final_request"] != spec.Application.FinalRequest || recovery.Arguments["language"] != "en" {
					t.Fatal("recovery omitted immutable submission arguments")
				}
			} else if field(read, "recovery_submit") != "" || field(read, "origin") != "legacy" {
				t.Fatal("legacy mutation recovery exposed")
			}
			listed := documentRead(f, member, "issuance_draft.list", SampleWorkspace, map[string]string{})
			if listed.status != 200 || field(listed, "count") != "1" || field(listed, "row_00") == "" {
				t.Fatal(listed.raw)
			}
			if !reflect.DeepEqual(a.drafts[spec.ID], before) || len(a.draftReviews) != 0 || f.s.Revision() != revision {
				t.Fatal("read settled uncertainty or changed revision")
			}
			_, err = a.documents.Submit(SampleWorkspace, before.Draft, issuance.Actor{ID: SampleMember, Kind: issuance.KindHuman})
			f.must(err)
			conflict := documentRead(f, member, "issuance_draft.inspect", spec.ID, map[string]string{})
			if conflict.status != 200 || field(conflict, "module_observation") != "conflict" || !reflect.DeepEqual(a.drafts[spec.ID], before) {
				t.Fatal("read adopted foreign unobserved module", conflict.raw)
			}
			f.s.fault = func(at string) bool { return at == "issuance-draft-lookup" }
			unavailable := documentRead(f, member, "issuance_draft.inspect", spec.ID, map[string]string{})
			if unavailable.status != 200 || field(unavailable, "module_observation") != "unavailable" || !reflect.DeepEqual(a.drafts[spec.ID], before) {
				t.Fatal(unavailable.raw)
			}
			f.s.fault = nil
		})
	}
}

func TestFinancialDraftPublicReadFrozenReviewAndHold(t *testing.T) {
	f, member, reviewer, spec, binding := configuredDraftWorld(t)
	a := f.s.financialIssuance
	draft, err := submitDraftAs(f, member, spec)
	f.must(err)
	prepared, err := prepareDraftReviewAs(f, reviewer, "public-read-review", draft, binding)
	f.must(err)
	read := documentRead(f, reviewer, "issuance_draft.inspect", spec.ID, map[string]string{})
	if read.status != 200 || field(read, "recovery_prepare") == "" || field(read, "recovery_approve") == "" || field(read, "coverage") == "" || field(read, "reference_02") == "" {
		t.Fatal(read.raw)
	}
	var descriptor struct {
		Arguments  map[string]string `json:"arguments"`
		Permission string            `json:"permission"`
	}
	f.must(json.Unmarshal([]byte(field(read, "recovery_prepare")), &descriptor))
	if descriptor.Permission != "allowed" || descriptor.Arguments["binding"] != binding.Spec.ID || descriptor.Arguments["binding_digest"] != binding.Digest || descriptor.Arguments["draft_digest"] != draft.Digest {
		t.Fatal("wrong frozen service binding recovery")
	}
	other := documentRead(f, member, "issuance_draft.inspect", spec.ID, map[string]string{})
	f.must(json.Unmarshal([]byte(field(other, "recovery_approve")), &descriptor))
	if descriptor.Permission != "denied" {
		t.Fatal("viewer gained recovery ownership")
	}
	f.s.fault = func(at string) bool { return at == "issuance-draft-review-after-module" }
	_, err = approveDraftAs(f, reviewer, prepared.Review)
	documentFailure(t, err, commands.DependencyUnavailable)
	f.s.fault = nil
	unknown := copyDraftReview(a.draftReviews[prepared.Review.ID])
	revision := f.s.Revision()
	read = documentRead(f, reviewer, "issuance_draft.inspect", spec.ID, map[string]string{})
	if read.status != 200 || field(read, "module_observation") != "matching_approved" || field(read, "receipt") != "review_attempt" || !reflect.DeepEqual(a.draftReviews[unknown.ID], unknown) || f.s.Revision() != revision {
		t.Fatal("inspection settled unknown review", read.raw)
	}
	approved, err := approveDraftAs(f, reviewer, prepared.Review)
	f.must(err)
	f.advance(24*time.Hour + time.Minute)
	renewInvoiceAuthorities(t, f, "-read-expiry")
	reviewer = f.as("configured-draft-reviewer@example.invalid")
	f.s.enterHold("synthetic frozen read")
	revision = f.s.Revision()
	read = documentRead(f, reviewer, "issuance_draft.inspect", spec.ID, map[string]string{})
	listed := documentRead(f, reviewer, "issuance_draft.list", SampleWorkspace, map[string]string{})
	if read.status != 200 || listed.status != 200 || field(read, "receipt") != "historical_approval" || field(read, "current_eligibility") != "unchecked" || field(read, "recovery_approve") != "" || !reflect.DeepEqual(a.draftReviews[unknown.ID], approved.Review) || f.s.Revision() != revision {
		t.Fatal("historical held read reopened eligibility", read.raw, listed.raw)
	}
	revoked := 0
	for _, g := range f.s.admin.Grants() {
		if g.SubjectID == "configured-draft-reviewer" && g.Terms.Kind == draftauthority.ResourceKind && g.Terms.Action == gate.ViewCost {
			f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
			revoked++
		}
	}
	if revoked == 0 {
		t.Fatal("inspect grant not revoked")
	}
	read = documentRead(f, reviewer, "issuance_draft.inspect", spec.ID, map[string]string{})
	f.code(t, "held read still needs current inspection", read, 403, string(commands.PermissionDenied))
	if read.body["fields"] != nil {
		t.Fatal("lost inspection leaked evidence")
	}
}

func TestFinancialDraftPublicReadAdmissionBeforeEvidence(t *testing.T) {
	f, member, c, digest := applicationWorld(t)
	a := f.s.financialIssuance
	spec := financialDraftSpec{ID: "public-admission", Language: "en", Application: &financialDraftApplicationRequest{Schema: "hb_issuance_application_request_v1", Configuration: configurationReference(c), FinalRequest: "issuance-final", FinalDigest: digest}}
	draft, err := submitDraftAs(f, member, spec)
	f.must(err)
	lookups := 0
	f.s.fault = func(at string) bool {
		if at == "issuance-draft-lookup" {
			lookups++
		}
		return false
	}
	stranger := f.as(activateProtectedMember(f, "unprivileged-reader"))
	for _, entry := range []struct{ command, target string }{{"issuance_draft.inspect", spec.ID}, {"issuance_draft.list", SampleWorkspace}} {
		result := documentRead(f, stranger, entry.command, entry.target, map[string]string{})
		f.code(t, "ordinary member requires explicit inspect", result, 403, string(commands.PermissionDenied))
		if result.body["fields"] != nil {
			t.Fatal("unprivileged read disclosed data")
		}
	}
	session := member.verifier(f)
	f.s.sessions.remove(session)
	result := documentRead(f, member, "issuance_draft.inspect", spec.ID, map[string]string{})
	f.code(t, "ended session cannot read", result, 401, string(commands.Unauthenticated))
	if lookups != 0 || result.body["fields"] != nil || !reflect.DeepEqual(a.drafts[spec.ID], draft) {
		t.Fatal("admission failure observed or changed evidence")
	}
	f.s.fault = nil
}
