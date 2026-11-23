package server

import (
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func issuanceDraftPending(t *testing.T, f world, a actor, bulk string) (int, any) {
	t.Helper()
	for _, raw := range f.bulkRead(a, "/"+bulk).body["impact"].(map[string]any)["pending"].([]any) {
		if row := raw.(map[string]any); row["store"] == "issuance_draft_submissions" {
			return whole(row["count"]), row["rechecked"]
		}
	}
	t.Fatal("issuance draft submissions omitted from impact")
	return 0, nil
}

func TestIssuanceDraftSubmissionsCountInBulkImpact(t *testing.T) {
	f, member, reviewer, spec, binding := configuredDraftWorld(t)
	a, root := f.s.financialIssuance, f.as(AdminLogin)
	counted := func(name string, want int) {
		t.Helper()
		tally, err := issuanceDraftTally(f.s)
		if err != nil || len(tally) > 1 || tally[SampleMember] != want || want == 0 && len(tally) != 0 {
			t.Fatal(name, tally, err)
		}
	}
	counted("before any draft", 0)
	f.ok(t, "submit", draftRun(f, member, "issuance_draft.submit", spec.ID, draftSubmitArguments(spec)), "submitted", "")
	counted("submitted draft counted for its submitter only", 1)
	prepared := draftRun(f, reviewer, "issuance_draft.prepare_approval", "impact-review", draftPrepareArguments(a.drafts[spec.ID], binding))
	f.ok(t, "prepare", prepared, "prepared", "")
	counted("draft with a prepared review", 1)

	staged := f.stage(root, "", SampleMember)
	f.ok(t, "stage", staged, "draft", "")
	before := f.s.Revision()
	fails := func(name string) {
		t.Helper()
		f.code(t, name, f.prepare(root, field(staged, "draft"), "suspend_membership"), 503, string(commands.DependencyUnavailable))
	}
	f.s.fault = func(at string) bool { return at == "issuance-draft-impact" }
	fails("impact fails closed")
	f.s.fault = nil
	saved := a.drafts[spec.ID]
	corrupt := saved
	corrupt.State = "unrecognized"
	a.drafts[spec.ID] = corrupt
	fails("unknown draft state fails closed")
	foreign := saved
	foreign.Workspace = "another-workspace"
	a.drafts[spec.ID] = foreign
	fails("foreign draft fails closed")
	refused := saved
	refused.State = "submission_refused"
	a.drafts[spec.ID] = refused
	counted("refused submission", 0)
	a.drafts[spec.ID] = saved
	review := a.draftReviews["impact-review"]
	broken := review
	broken.State = "unrecognized"
	a.draftReviews["impact-review"] = broken
	fails("unknown review state fails closed")
	delete(a.draftReviews, "impact-review")
	fails("missing review fails closed")
	a.draftReviews["impact-review"] = review
	if f.s.Revision() != before {
		t.Fatal("impact failure mutated revision")
	}
	list := f.prepare(root, field(staged, "draft"), "suspend_membership")
	f.ok(t, "prepare impact", list, "prepared", "")
	if count, rechecked := issuanceDraftPending(t, f, root, field(list, "bulk")); count != 1 || rechecked != true {
		t.Fatal("bulk impact row", count, rechecked)
	}

	f.ok(t, "approve", draftRun(f, reviewer, "issuance_draft.approve", "impact-review", map[string]string{"digest": field(prepared, "review_digest")}), "approved", "")
	counted("approved draft", 0)
}

func TestIssuanceDraftApprovalRechecksSuspendedSubmitter(t *testing.T) {
	f, member, reviewer, spec, binding := configuredDraftWorld(t)
	a, root := f.s.financialIssuance, f.as(AdminLogin)
	f.ok(t, "submit", draftRun(f, member, "issuance_draft.submit", spec.ID, draftSubmitArguments(spec)), "submitted", "")
	prepared := draftRun(f, reviewer, "issuance_draft.prepare_approval", "recheck-review", draftPrepareArguments(a.drafts[spec.ID], binding))
	f.ok(t, "prepare", prepared, "prepared", "")
	bulk, digest := f.prepared(t, root, "suspend_membership", SampleMember)
	if count, rechecked := issuanceDraftPending(t, f, root, bulk); count != 1 || rechecked != true {
		t.Fatal("impact before execution", count, rechecked)
	}
	if got := f.execute(root, bulk, digest); got.status != 202 {
		t.Fatal(got.raw)
	}
	approval := draftRun(f, reviewer, "issuance_draft.approve", "recheck-review", map[string]string{"digest": field(prepared, "review_digest")})
	f.code(t, "approve after the submitter is suspended", approval, 403, string(commands.PermissionDenied))
	if a.draftReviews["recheck-review"].State == "approved" {
		t.Fatal("a draft of a suspended submitter was approved")
	}
}
