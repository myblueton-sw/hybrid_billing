package server

import (
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func TestFinancialFinalOwnedRequestIdentityAndRetry(t *testing.T) {
	w, _, digest := publicResult(t)
	f := w.f
	f.ok(t, "apply result", f.run(w.actors[Admin2Login], "erp_result.apply", "public-result", map[string]string{"digest": digest}), "recorded", "")
	proposal := f.run(w.actors[AdminLogin], "erp_final_policy.propose", "request-final-policy", financialFields("2"))
	f.ok(t, "propose final policy", proposal, "pending", "")
	f.ok(t, "confirm final policy", f.run(w.actors[Admin2Login], "erp_final_policy.confirm", "request-final-policy", map[string]string{"digest": field(proposal, "digest")}), "confirmed", "")
	ctx := commands.Context{Workspace: SampleWorkspace, User: SampleAdmin3, Target: "actual-final-request", Rationale: "Review the reported final gross", Fields: map[string]string{"result": "public-result", "digest": digest}}
	request := func(c commands.Context) (approvals.RequestView, error) {
		var out approvals.RequestView
		err := inputAuthorityAs(f, w.actors[Admin3Login], func() error { var err error; out, err = f.s.requestFinancialFinal(c); return err })
		return out, err
	}
	first, err := request(ctx)
	f.must(err)
	if first.State != approvals.StatePending || first.Subject.Amount != "16" || first.Subject.Author != SampleAdmin2 || first.Requester != SampleAdmin3 {
		t.Fatalf("unexpected final request: %+v", first)
	}
	replay, err := request(ctx)
	f.must(err)
	if replay.Digest != first.Digest || len(f.s.financialFinals) != 1 {
		t.Fatal("retry created a different request")
	}
	changed := ctx
	changed.Rationale = "Changed request rationale"
	_, err = request(changed)
	documentFailure(t, err, commands.EvidenceStale)
	changed = ctx
	changed.Target = "document-charge"
	_, err = request(changed)
	documentFailure(t, err, commands.EvidenceStale)
	_, binding, err := f.s.financialFinalRecord(SampleWorkspace, ctx.Target)
	f.must(err)
	binding.Candidate.Subject.Digests["result"] = "changed"
	binding.Candidate.Contributors[0] = "changed"
	_, _, err = f.s.financialFinalRecord(SampleWorkspace, ctx.Target)
	f.must(err)
	if len(f.s.financialFinals) != 1 {
		t.Fatal("failed request changed ownership")
	}
}
