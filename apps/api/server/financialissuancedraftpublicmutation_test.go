package server

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func draftRun(f world, who actor, command, target string, fields map[string]string) reply {
	f.t.Helper()
	body := chargeBody(f, fmt.Sprintf("draft-command-%d", keys.Add(1)), target, fields)
	return f.call(command, body, who.cookie, who.csrf)
}

func draftSubmitArguments(spec financialDraftSpec) map[string]string {
	r := spec.Application
	return map[string]string{"configuration": r.Configuration.ID, "configuration_digest": r.Configuration.Digest, "final_request": r.FinalRequest, "final_digest": r.FinalDigest, "language": spec.Language}
}

func draftPrepareArguments(p financialDraftRecord, b financialTermBinding) map[string]string {
	return map[string]string{"draft": p.Spec.ID, "draft_digest": p.Digest, "binding": b.Spec.ID, "binding_digest": b.Digest}
}

func TestFinancialDraftPublicMutationActualJourneyAndHistoricalReplay(t *testing.T) {
	f, member, reviewer, spec, binding := configuredDraftWorld(t)
	a := f.s.financialIssuance
	submitBody := chargeBody(f, "public-submit-envelope", spec.ID, draftSubmitArguments(spec))
	want := f.s.Next()
	submitted := f.call("issuance_draft.submit", submitBody, member.cookie, member.csrf)
	f.ok(t, "public submit", submitted, "submitted", "")
	if field(submitted, "receipt") != "submission_record" || f.s.Revision() != want {
		t.Fatal(submitted.raw)
	}
	revision := f.s.Revision()
	replay := f.call("issuance_draft.submit", submitBody, member.cookie, member.csrf)
	if replay.status != 200 || replay.body["replay"] != true || f.s.Revision() != revision {
		t.Fatal("cached replay changed revision", replay.raw)
	}
	p := a.drafts[spec.ID]
	prepared := draftRun(f, reviewer, "issuance_draft.prepare_approval", "public-actual-review", draftPrepareArguments(p, binding))
	f.ok(t, "save independent public review", prepared, "prepared", "")
	r := a.draftReviews["public-actual-review"]
	if field(prepared, "review_digest") != r.Digest || field(prepared, "receipt") != "review_attempt" {
		t.Fatal(prepared.raw)
	}
	approveArgs := map[string]string{"digest": r.Digest}
	approveBody := chargeBody(f, "public-approval-envelope", r.ID, approveArgs)
	approved := f.call("issuance_draft.approve", approveBody, reviewer.cookie, reviewer.csrf)
	f.ok(t, "public approve", approved, "approved", "")
	if field(approved, "receipt") != "historical_approval" {
		t.Fatal(approved.raw)
	}
	actual, err := a.documents.DraftOf(SampleWorkspace, p.Spec.ID)
	f.must(err)
	if actual.Approver != r.Actor || actual.Coverage == nil || *actual.Coverage != r.Coverage {
		t.Fatal("receipt lacks real module approval")
	}
	before := copyDraftReview(a.draftReviews[r.ID])
	f.advance(24*time.Hour + time.Minute)
	renewInvoiceAuthorities(t, f, "-public-mutation-expiry")
	member, reviewer = f.as(MemberLogin), f.as("configured-draft-reviewer@example.invalid")
	for _, test := range []struct {
		who          actor
		kind, target string
		args         map[string]string
	}{
		{member, "submit", spec.ID, draftSubmitArguments(spec)},
		{reviewer, "prepare_approval", r.ID, draftPrepareArguments(p, binding)},
		{reviewer, "approve", r.ID, approveArgs},
	} {
		want = f.s.Next()
		read := draftRun(f, test.who, "issuance_draft."+test.kind, test.target, test.args)
		if read.status != 200 || f.s.Revision() != want {
			t.Fatal("fresh-envelope historical replay", read.raw)
		}
	}
	if !reflect.DeepEqual(a.draftReviews[r.ID], before) || len(a.drafts) != 1 || len(a.draftReviews) != 1 {
		t.Fatal("replay changed retained approval")
	}
	for _, grant := range f.s.admin.Grants() {
		if grant.SubjectID == r.Actor && grant.Terms.Kind == draftauthority.ResourceKind && grant.Terms.Action == gate.ApproveIssuanceDraft {
			f.must(f.s.admin.RevokeGrant(grant.ID, grant.Version))
		}
	}
	revoked := f.call("issuance_draft.approve", approveBody, reviewer.cookie, reviewer.csrf)
	f.code(t, "current grant before cached approval", revoked, 403, string(commands.PermissionDenied))
	if revoked.body["fields"] != nil {
		t.Fatal("revoked replay disclosed receipt")
	}
}

func TestFinancialDraftPublicMutationAdmissionAndOutputBeforeEffects(t *testing.T) {
	f, member, reviewer, spec, binding := configuredDraftWorld(t)
	a := f.s.financialIssuance
	f.s.cfg.Commands.Value = 4095
	revision := f.s.Revision()
	read := draftRun(f, member, "issuance_draft.submit", spec.ID, draftSubmitArguments(spec))
	f.code(t, "preflight output capacity", read, 501, string(commands.UnsupportedCapability))
	if len(a.drafts) != 0 || len(a.draftScopes) != 0 || f.s.Revision() != revision || read.body["operation_id"] != nil {
		t.Fatal("preflight reserved business state", read.raw)
	}
	f.s.cfg.Commands.Value = 4096
	f.ok(t, "submit after output correction", draftRun(f, member, "issuance_draft.submit", spec.ID, draftSubmitArguments(spec)), "submitted", "")
	p := a.drafts[spec.ID]
	f.s.cfg.Commands.Value = 4095
	revision = f.s.Revision()
	read = draftRun(f, reviewer, "issuance_draft.prepare_approval", "public-cap-review", draftPrepareArguments(p, binding))
	f.code(t, "prepare output before reservation", read, 501, string(commands.UnsupportedCapability))
	if len(a.draftReviews) != 0 || f.s.Revision() != revision {
		t.Fatal("prepare preflight retained review")
	}
	f.s.cfg.Commands.Value = 4096
	f.ok(t, "prepare after output correction", draftRun(f, reviewer, "issuance_draft.prepare_approval", "public-cap-review", draftPrepareArguments(p, binding)), "prepared", "")
	r := a.draftReviews["public-cap-review"]
	f.s.cfg.Commands.Value = 4095
	revision = f.s.Revision()
	read = draftRun(f, reviewer, "issuance_draft.approve", r.ID, map[string]string{"digest": r.Digest})
	f.code(t, "approve output before effect", read, 501, string(commands.UnsupportedCapability))
	if !reflect.DeepEqual(a.draftReviews[r.ID], r) || f.s.Revision() != revision {
		t.Fatal("approval preflight changed review")
	}
}
