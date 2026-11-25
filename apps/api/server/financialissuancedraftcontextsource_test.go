package server

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
)

func draftContextEvidenceAs(f world, who actor, request, digest string) (financialDraftContextEvidence, error) {
	var out financialDraftContextEvidence
	err := inputAuthorityAs(f, who, func() error {
		var err error
		out, err = f.s.financialIssuance.draftContextEvidence(SampleWorkspace, f.s.answer.User(), request, digest)
		return err
	})
	return out, err
}

func TestFinancialDraftContextFrozenSummary(t *testing.T) {
	f, member, c, digest := applicationWorld(t)
	a := f.s.financialIssuance
	before, err := draftContextEvidenceAs(f, member, "issuance-final", digest)
	f.must(err)
	if before.Existing != nil || before.Fields["total"] != "16" || before.Fields["scale"] != "2" || before.Fields["currency"] != "USD" || before.Fields["customer_id"] == "" || before.Fields["period"] == "" || before.Fields["current_eligibility"] != "unchecked" {
		t.Fatal("invalid frozen financial summary", before.Fields)
	}
	spec := financialDraftSpec{ID: "context-owned-draft", Language: "en", Application: &financialDraftApplicationRequest{Schema: "hb_issuance_application_request_v1", Configuration: configurationReference(c), FinalRequest: "issuance-final", FinalDigest: digest}}
	f.s.fault = func(at string) bool { return at == "issuance-draft-after-intent" }
	_, err = submitDraftAs(f, member, spec)
	documentFailure(t, err, commands.DependencyUnavailable)
	f.s.fault = nil
	retained := copyFinancialDraft(a.drafts[spec.ID])
	revision := f.s.Revision()
	unknown, err := draftContextEvidenceAs(f, member, "issuance-final", digest)
	f.must(err)
	if unknown.Existing == nil || unknown.Existing.Module != "missing" || unknown.Existing.Draft.State != "submission_unknown" || unknown.Fields["existing_draft"] != spec.ID || !reflect.DeepEqual(a.drafts[spec.ID], retained) || f.s.Revision() != revision {
		t.Fatal("context failed to discover uncertainty without settlement")
	}
	unknown.Existing.Draft.Application.Source.Basis.Rows[0].Net = "999"
	if !reflect.DeepEqual(a.drafts[spec.ID], retained) {
		t.Fatal("context leaked mutable source")
	}
	f.advance(24*time.Hour + time.Minute)
	renewInvoiceAuthorities(t, f, "-context-source")
	member = f.as(MemberLogin)
	_, err = applicationAs(f, member, c, "issuance-final", digest)
	documentFailure(t, err, commands.PrerequisiteFailed)
	f.s.enterHold("synthetic context history")
	for _, g := range f.s.admin.Grants() {
		if g.SubjectID == SampleMember && g.Terms.Kind == draftauthority.ResourceKind && g.Terms.Action == gate.SubmitIssuanceDraft {
			f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
		}
	}
	revision = f.s.Revision()
	held, err := draftContextEvidenceAs(f, member, "issuance-final", digest)
	f.must(err)
	if held.Fields["total"] != before.Fields["total"] || held.Fields["period"] != before.Fields["period"] || held.Fields["current_eligibility"] != "unchecked" || !reflect.DeepEqual(a.drafts[spec.ID], retained) || f.s.Revision() != revision {
		t.Fatal("held history required Submit or refreshed expired eligibility")
	}
}

func TestFinancialDraftContextIntegrityAndDependencies(t *testing.T) {
	for _, mode := range []string{"digest", "final", "result", "dangling_scope", "missing_scope", "approval_adapter", "result_adapter"} {
		t.Run(mode, func(t *testing.T) {
			f, member, spec, _ := draftSubmissionWorld(t)
			p, err := submitDraftAs(f, member, spec)
			f.must(err)
			a := f.s.financialIssuance
			source, err := draftOriginSource(p)
			f.must(err)
			request, digest := source.Basis.FinalRequest, source.Basis.FinalDigest
			code := commands.EvidenceStale
			switch mode {
			case "digest":
				digest = strings.Repeat("0", 64)
			case "final":
				b := f.s.financialFinals[request]
				b.Candidate.Net = "999"
				f.s.financialFinals[request] = b
			case "result":
				delete(f.s.documentERP.results, [2]string{SampleWorkspace, source.Basis.Result})
			case "dangling_scope":
				a.draftScopes[source.Basis.Subject.Target] = "not-owned"
			case "missing_scope":
				delete(a.draftScopes, source.Basis.Subject.Target)
			case "approval_adapter":
				f.s.approvals = nil
				code = commands.UnsupportedCapability
			case "result_adapter":
				f.s.documentERP = nil
				code = commands.UnsupportedCapability
			}
			revision := f.s.Revision()
			out, err := draftContextEvidenceAs(f, member, request, digest)
			documentFailure(t, err, code)
			if out.Fields != nil || out.Existing != nil || f.s.Revision() != revision {
				t.Fatal("invalid context disclosed partial facts or changed revision")
			}
		})
	}
}

func TestFinancialDraftContextAdmissionBeforeFrozenEvidence(t *testing.T) {
	f, _, _, digest := applicationWorld(t)
	stranger := f.as(activateProtectedMember(f, "context-stranger"))
	f.s.approvals = nil
	out, err := draftContextEvidenceAs(f, stranger, "issuance-final", digest)
	documentFailure(t, err, commands.PermissionDenied)
	if out.Fields != nil || out.Existing != nil {
		t.Fatal("unprivileged context disclosed frozen summary")
	}
}

func TestFinancialDraftContextEconomicOwnerAcrossFinalRequests(t *testing.T) {
	f, member, spec, _ := draftSubmissionWorld(t)
	p, err := submitDraftAs(f, member, spec)
	f.must(err)
	source, err := draftOriginSource(p)
	f.must(err)
	request := finalRequest(f, f.as(Admin3Login), "context-second-final", source.Basis.ResultDigest)
	f.ok(t, "another request for the same economic target", request, "pending", "")
	digest := field(request, "digest")
	revision := f.s.Revision()
	out, err := draftContextEvidenceAs(f, member, "context-second-final", digest)
	f.must(err)
	if out.Fields["final_request"] != "context-second-final" || out.Fields["final_state"] != "pending" || out.Existing == nil || out.Existing.Draft.Spec.ID != p.Spec.ID || out.Existing.Module != "matching_unapproved" || out.Existing.Draft.Profile.Source.Basis.FinalRequest != "issuance-final" || f.s.Revision() != revision {
		t.Fatal("economic owner was lost or reassigned through another final request")
	}
	a := f.s.financialIssuance
	duplicate := copyFinancialDraft(p)
	duplicate.Spec.ID = "context-duplicate"
	duplicate.Draft, err = issuanceDraftFromSource(duplicate.Profile, duplicate.Spec.ID, duplicate.Spec.Language)
	f.must(err)
	duplicate.Module = issuance.DraftView{}
	duplicate.State = "submission_unknown"
	duplicate.Digest, err = draftIntentDigest(duplicate)
	f.must(err)
	a.drafts[duplicate.Spec.ID] = duplicate
	out, err = draftContextEvidenceAs(f, member, "context-second-final", digest)
	documentFailure(t, err, commands.EvidenceStale)
	if out.Fields != nil || out.Existing != nil || f.s.Revision() != revision {
		t.Fatal("duplicate economic ownership was hidden")
	}
}
