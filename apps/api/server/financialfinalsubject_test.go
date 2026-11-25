package server

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/erpledger"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func TestFinancialFinalGrossExactFixedScale(t *testing.T) {
	for _, c := range []struct {
		net, tax, want string
		scale          int
	}{
		{"15.00", "1.00", "16", 2}, {"0.00", "0.00", "0", 2}, {"0.10", "0.20", "0.3", 2},
		{"15", "1", "", 2}, {"-1.00", "1.00", "", 2}, {"1.001", "0.000", "", 2},
		{"1000000000000.00", "0.00", "", 2},
		{"NaN", "0.00", "", 2}, {"1.00", "0.00", "", 5},
	} {
		t.Run(c.net+"/"+c.tax, func(t *testing.T) {
			got, err := financialFinalGross(erpledger.Amounts{Net: c.net, Tax: c.tax}, c.scale)
			if c.want == "" {
				documentFailure(t, err, commands.PrerequisiteFailed)
				return
			}
			if err != nil || got != c.want {
				t.Fatalf("gross=%q error=%v", got, err)
			}
		})
	}
}

func TestFinancialFinalSubjectOwnedEvidenceAndDetachedOutput(t *testing.T) {
	w, packet, digest := publicResult(t)
	f := w.f
	build := func(id string) (financialFinalCandidate, error) {
		var result financialFinalCandidate
		err := inputAuthorityAs(f, w.actors[Admin3Login], func() error {
			var err error
			result, err = f.s.financialFinalSubjectLocked(SampleWorkspace, id, "public-result", digest, SampleAdmin3)
			return err
		})
		return result, err
	}
	_, err := build("final-request")
	documentFailure(t, err, commands.PrerequisiteFailed)
	f.ok(t, "apply reported result", f.run(w.actors[Admin2Login], "erp_result.apply", "public-result", map[string]string{"digest": digest}), "recorded", "")
	_, err = build("final-request")
	documentFailure(t, err, commands.PrerequisiteFailed)
	p := f.run(w.actors[AdminLogin], "erp_final_policy.propose", "subject-final-policy", financialFields("2"))
	f.ok(t, "propose final policy", p, approvals.StatePending, "")
	f.ok(t, "confirm final policy", f.run(w.actors[Admin2Login], "erp_final_policy.confirm", "subject-final-policy", map[string]string{"digest": field(p, "digest")}), "confirmed", "")
	first, err := build("final-request")
	f.must(err)
	if first.Subject.Amount != "16" || first.Subject.Currency != "USD" || first.Subject.Stage != approvals.StageERPFinal || first.Subject.Author != SampleAdmin2 || first.Net != "15.00" || first.Tax != "1.00" || first.ApplicationReceipt == "" {
		t.Fatalf("unexpected candidate: %+v", first)
	}
	next, err := build("another-final-request")
	f.must(err)
	if next.Subject.Target != first.Subject.Target || !reflect.DeepEqual(next.Subject.Digests, first.Subject.Digests) {
		t.Fatal("request identity changed economic target or evidence")
	}
	first.Subject.Digests["result"] = "changed"
	first.Policy.Required[0] = "changed"
	first.Contributors[0] = "changed"
	again, err := build("another-final-request")
	f.must(err)
	if !reflect.DeepEqual(next, again) {
		t.Fatal("returned candidate aliases authoritative state")
	}
	for _, author := range again.Contributors {
		documentFailure(t, financialFinalContributorCheck(again, "independent-requester", author), commands.SelfApprovalDenied)
	}
	documentFailure(t, financialFinalContributorCheck(again, "requester", "requester"), commands.SelfApprovalDenied)
	f.must(financialFinalContributorCheck(again, "independent-requester", "independent-decider"))
	packet.Revision = 2
	for i := range packet.Rows {
		packet.Rows[i].ID = "new-" + packet.Rows[i].ID
	}
	replacement := confirmResult(t, w, proposeResult(t, w, packet, "replacement-result"))
	_, err = applyResult(w, replacement)
	f.must(err)
	_, err = build("final-request")
	documentFailure(t, err, commands.EvidenceStale)
	var replaced financialFinalCandidate
	f.must(inputAuthorityAs(f, w.actors[Admin3Login], func() error {
		var err error
		replaced, err = f.s.financialFinalSubjectLocked(SampleWorkspace, "replacement-request", replacement.Basis.ID, replacement.Review.Digest, SampleAdmin3)
		return err
	}))
	if replaced.Subject.Target != again.Subject.Target || replaced.Subject.Digests["result"] == again.Subject.Digests["result"] {
		t.Fatal("replacement must preserve target and change evidence")
	}
	for i, required := range []string{"erp_draft,result", "fx,result", "input,result"} {
		fields := financialFields(strconv.Itoa(i + 3))
		fields["required"] = required
		if i == 2 {
			fields["approvals"] = "2"
		}
		id := "unsupported-policy-" + strconv.Itoa(i)
		proposal := f.run(w.actors[AdminLogin], "erp_final_policy.propose", id, fields)
		f.ok(t, "propose unsupported consumer policy", proposal, "pending", "")
		f.ok(t, "confirm unsupported consumer policy", f.run(w.actors[Admin2Login], "erp_final_policy.confirm", id, map[string]string{"digest": field(proposal, "digest")}), "confirmed", "")
		err = inputAuthorityAs(f, w.actors[Admin3Login], func() error {
			_, err := f.s.financialFinalSubjectLocked(SampleWorkspace, "unsupported-request", replacement.Basis.ID, replacement.Review.Digest, SampleAdmin3)
			return err
		})
		documentFailure(t, err, commands.PrerequisiteFailed)
	}
	err = inputAuthorityAs(f, w.actors[Admin3Login], func() error {
		_, err := f.s.financialFinalSubjectLocked("foreign-workspace", "foreign-request", replacement.Basis.ID, replacement.Review.Digest, SampleAdmin3)
		return err
	})
	documentFailure(t, err, commands.PermissionDenied)
	err = inputAuthorityAs(f, w.actors[Admin3Login], func() error {
		caller := f.s.caller
		defer func() { f.s.caller = caller }()
		f.s.caller = identity.Verifier{}
		_, err := f.s.financialFinalSubjectLocked(SampleWorkspace, "missing-session-request", replacement.Basis.ID, replacement.Review.Digest, SampleAdmin3)
		return err
	})
	documentFailure(t, err, commands.Unauthenticated)

}
