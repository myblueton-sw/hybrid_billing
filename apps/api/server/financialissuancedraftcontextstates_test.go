package server

import (
	"reflect"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
)

func TestFinancialDraftPublicContextConfigurationStates(t *testing.T) {
	for _, pending := range []bool{false, true} {
		name := "missing"
		if pending {
			name = "pending"
		}
		t.Run(name, func(t *testing.T) {
			f, _, _, digest := issuanceSourceWorld(t, func(f world) {
				if pending {
					_, err := proposeConfigurationAs(f, f.as(AdminLogin), configurationSpec())
					f.must(err)
				}
			})
			grantDraftCapability(f, draftauthority.Submit)
			grantDraftCapability(f, draftauthority.Inspect)
			read := draftContextRead(f, f.as(MemberLogin), "issuance-final", digest)
			readiness, action := "configuration_missing", "setup_missing"
			if pending {
				readiness, action = "unavailable", "evidence_unavailable"
			}
			assertDraftContext(t, read, map[string]string{"readiness": readiness, "submit_action": action, "current_eligibility": "eligible", "configuration": "", "binding_count": "0"})
		})
	}
}

func TestFinancialDraftPublicContextIndependentReviewerAndOwnership(t *testing.T) {
	f, member, reviewer, spec, binding := configuredDraftWorld(t)
	grantDraftReviewer(f, "context-member-", SampleMember)
	member = f.as(MemberLogin)
	draft, err := submitDraftAs(f, member, spec)
	f.must(err)
	read := draftContextRead(f, member, spec.Application.FinalRequest, spec.Application.FinalDigest)
	assertDraftContext(t, read, map[string]string{"prepare_action": "independent_reviewer_required", "approve_permission": "allowed"})
	prepared, err := prepareDraftReviewAs(f, reviewer, "context-owner-review", draft, binding)
	f.must(err)
	read = draftContextRead(f, member, spec.Application.FinalRequest, spec.Application.FinalDigest)
	assertDraftContext(t, read, map[string]string{"prepare_action": "owned_by_other", "approve_action": "owned_by_other"})
	if !reflect.DeepEqual(f.s.financialIssuance.draftReviews[prepared.Review.ID], prepared.Review) {
		t.Fatal("other viewer took over review")
	}
}

func TestFinancialDraftPublicContextSavedCoverageDrift(t *testing.T) {
	f, member, reviewer, spec, binding := configuredDraftWorld(t)
	draft, err := submitDraftAs(f, member, spec)
	f.must(err)
	prepared, err := prepareDraftReviewAs(f, reviewer, "context-drift-review", draft, binding)
	f.must(err)
	f.advance(16 * time.Hour)
	renewInvoiceAuthorities(t, f, "-context-drift")
	reviewer = f.as("configured-draft-reviewer@example.invalid")
	revision := f.s.Revision()
	read := draftContextRead(f, reviewer, spec.Application.FinalRequest, spec.Application.FinalDigest)
	assertDraftContext(t, read, map[string]string{"approve_action": "recheck_required", "current_eligibility": "eligible", "readiness": "existing_draft"})
	if !reflect.DeepEqual(f.s.financialIssuance.draftReviews[prepared.Review.ID], prepared.Review) || f.s.Revision() != revision {
		t.Fatal("read published drift refusal")
	}
	refused, err := approveDraftAs(f, reviewer, prepared.Review)
	f.must(err)
	if refused.Review.State != "refused" || refused.Review.Refusal != issuance.ReasonCoverageMismatch {
		t.Fatal("actual recheck did not publish changed coverage refusal")
	}
}

func TestFinancialDraftPublicContextStepUpAndUnavailable(t *testing.T) {
	f, member, _, digest := applicationWorld(t)
	f.advance(16 * time.Minute)
	read := draftContextRead(f, member, "issuance-final", digest)
	assertDraftContext(t, read, map[string]string{"step_up": "reauthentication_required", "submit_action": "reauthentication_required", "approve_action": "permission_denied"})
	member = f.as(MemberLogin)
	f.s.financialIssuance.fields = nil
	read = draftContextRead(f, member, "issuance-final", digest)
	assertDraftContext(t, read, map[string]string{"readiness": "unavailable", "submit_action": "evidence_unavailable", "current_eligibility": "eligible"})
	f.s.cfg.Commands.Value = 4095
	read = draftContextRead(f, member, "issuance-final", digest)
	f.code(t, "context requires output capacity", read, 501, string(commands.UnsupportedCapability))
	if read.body["fields"] != nil {
		t.Fatal("unsupported output leaked context")
	}
}

func TestFinancialDraftPublicContextDanglingConfigurationAndEndedSession(t *testing.T) {
	f, member, c, digest := applicationWorld(t)
	delete(f.s.financialIssuance.configurations, c.Spec.ID)
	read := draftContextRead(f, member, "issuance-final", digest)
	assertDraftContext(t, read, map[string]string{"readiness": "unavailable", "submit_action": "evidence_unavailable", "current_eligibility": "eligible"})
	f.s.sessions.remove(member.verifier(f))
	read = draftContextRead(f, member, "issuance-final", digest)
	f.code(t, "ended session cannot read context", read, 401, string(commands.Unauthenticated))
	if read.body["fields"] != nil {
		t.Fatal("ended session disclosed summary")
	}
}
