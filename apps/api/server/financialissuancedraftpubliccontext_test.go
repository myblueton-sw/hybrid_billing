package server

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func draftContextRead(f world, who actor, request, digest string) reply {
	return documentRead(f, who, "issuance_draft.context", request, map[string]string{"digest": digest})
}

func assertDraftContext(t *testing.T, read reply, expected map[string]string) {
	t.Helper()
	if read.status != 200 {
		t.Fatal(read.raw)
	}
	for key, value := range expected {
		if field(read, key) != value {
			t.Fatalf("%s: got %q, want %q: %s", key, field(read, key), value, read.raw)
		}
	}
	for _, key := range []string{"mode", "final_request", "final_digest", "final_state", "customer_id", "period", "currency", "scale", "total", "current_eligibility", "readiness", "submit_permission", "approve_permission", "submit_action", "prepare_action", "approve_action", "step_up", "snapshot", "binding_count"} {
		if field(read, key) == "" {
			t.Fatal("missing required context field", key)
		}
	}
}

func TestFinancialDraftPublicContextActualJourney(t *testing.T) {
	f, member, reviewer, spec, binding := configuredDraftWorld(t)
	a := f.s.financialIssuance
	request, digest := spec.Application.FinalRequest, spec.Application.FinalDigest
	revision := f.s.Revision()
	read := draftContextRead(f, member, request, digest)
	assertDraftContext(t, read, map[string]string{"readiness": "ready", "submit_action": "ready", "approve_action": "permission_denied", "current_eligibility": "eligible", "total": "16", "scale": "2", "binding_count": "1", "next_binding": ""})
	var choice map[string]string
	f.must(json.Unmarshal([]byte(field(read, "binding_00")), &choice))
	if choice["binding"] != binding.Spec.ID || choice["binding_digest"] != binding.Digest || choice["net_days"] != "30" || len(a.drafts) != 0 || f.s.Revision() != revision {
		t.Fatal("context used a module binding or reserved an intent")
	}
	draft, err := submitDraftAs(f, member, spec)
	f.must(err)
	read = draftContextRead(f, reviewer, request, digest)
	assertDraftContext(t, read, map[string]string{"readiness": "existing_draft", "prepare_action": "ready", "approve_action": "review_required", "submit_permission": "denied", "existing_draft": spec.ID})
	prepared, err := prepareDraftReviewAs(f, reviewer, "context-actual-review", draft, binding)
	f.must(err)
	revision = f.s.Revision()
	read = draftContextRead(f, reviewer, request, digest)
	assertDraftContext(t, read, map[string]string{"prepare_action": "unresolved_operation", "approve_action": "ready"})
	if !reflect.DeepEqual(a.draftReviews[prepared.Review.ID], prepared.Review) || f.s.Revision() != revision {
		t.Fatal("context changed saved review")
	}
	f.s.fault = func(at string) bool { return at == "issuance-draft-review-after-module" }
	_, err = approveDraftAs(f, reviewer, prepared.Review)
	documentFailure(t, err, commands.DependencyUnavailable)
	f.s.fault = nil
	unknown := copyDraftReview(a.draftReviews[prepared.Review.ID])
	read = draftContextRead(f, reviewer, request, digest)
	assertDraftContext(t, read, map[string]string{"approve_action": "unresolved_operation"})
	if !reflect.DeepEqual(a.draftReviews[prepared.Review.ID], unknown) {
		t.Fatal("context reconciled interrupted approval")
	}
	_, err = approveDraftAs(f, reviewer, prepared.Review)
	f.must(err)
	read = draftContextRead(f, reviewer, request, digest)
	assertDraftContext(t, read, map[string]string{"prepare_action": "completed", "approve_action": "completed"})
}

func TestFinancialDraftPublicContextMissingSetupAndPermissions(t *testing.T) {
	f, member, _, digest := applicationWorld(t)
	a := f.s.financialIssuance
	read := draftContextRead(f, member, "issuance-final", digest)
	assertDraftContext(t, read, map[string]string{"readiness": "terms_missing", "submit_action": "ready", "current_eligibility": "eligible", "binding_count": "0"})
	removed := 0
	for _, grant := range f.s.admin.Grants() {
		if grant.SubjectID == SampleMember && grant.Terms.Kind == draftauthority.ResourceKind && grant.Terms.Action == gate.SubmitIssuanceDraft {
			f.must(f.s.admin.RevokeGrant(grant.ID, grant.Version))
			removed++
		}
	}
	if removed == 0 {
		t.Fatal("no actual Submit grant revoked")
	}
	read = draftContextRead(f, member, "issuance-final", digest)
	assertDraftContext(t, read, map[string]string{"readiness": "terms_missing", "submit_permission": "denied", "submit_action": "permission_denied", "current_eligibility": "eligible"})
	f.s.fault = func(at string) bool { return at == "issuance-configuration-lookup" }
	read = draftContextRead(f, member, "issuance-final", digest)
	assertDraftContext(t, read, map[string]string{"readiness": "unavailable", "submit_action": "permission_denied", "binding_count": "0", "configuration": ""})
	f.s.fault = nil
	if len(a.drafts) != 0 || len(a.draftReviews) != 0 {
		t.Fatal("setup inspection retained business effects")
	}
}

func TestFinancialDraftPublicContextExpiredAndHeld(t *testing.T) {
	f, member, _, digest := applicationWorld(t)
	f.advance(24*time.Hour + time.Minute)
	renewInvoiceAuthorities(t, f, "-public-context")
	member = f.as(MemberLogin)
	read := draftContextRead(f, member, "issuance-final", digest)
	assertDraftContext(t, read, map[string]string{"readiness": "source_ineligible", "submit_action": "source_ineligible", "current_eligibility": "ineligible", "total": "16", "configuration": "", "binding_count": "0"})
	f.s.enterHold("synthetic context hold")
	read = draftContextRead(f, member, "issuance-final", digest)
	assertDraftContext(t, read, map[string]string{"readiness": "installation_held", "submit_action": "installation_held", "current_eligibility": "unchecked", "configuration": "", "total": "16"})
	for _, grant := range f.s.admin.Grants() {
		if grant.SubjectID == SampleMember && grant.Terms.Kind == draftauthority.ResourceKind && grant.Terms.Action == gate.ViewCost {
			f.must(f.s.admin.RevokeGrant(grant.ID, grant.Version))
		}
	}
	read = draftContextRead(f, member, "issuance-final", digest)
	f.code(t, "held context still needs Inspect", read, 403, string(commands.PermissionDenied))
	if read.body["fields"] != nil {
		t.Fatal("lost inspection disclosed context")
	}
}
