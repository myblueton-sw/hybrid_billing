package server

import (
	"reflect"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func issuanceSourceWorld(t *testing.T, beforeFinance ...func(world)) (world, actor, actor, string) {
	t.Helper()
	f, requester, decider, result := finalCommandWorld(t, beforeFinance...)
	p := finalRequest(f, requester, "issuance-final", result)
	f.ok(t, "request final", p, approvals.StatePending, "")
	digest := field(p, "digest")
	f.ok(t, "approve final", f.run(decider, "erp_final.approve", "issuance-final", map[string]string{"digest": digest}), approvals.StateApproved, "")
	return f, requester, decider, digest
}

func readIssuanceSource(f world, a actor, workspace, digest, consumer string) (financialIssuanceSourceView, error) {
	var out financialIssuanceSourceView
	err := inputAuthorityAs(f, a, func() error {
		var err error
		out, err = f.s.financialIssuanceSourceLocked(workspace, "issuance-final", digest, consumer)
		return err
	})
	return out, err
}

func TestFinancialIssuanceSourceOwnedFactsAndIsolation(t *testing.T) {
	f, requester, _, digest := issuanceSourceWorld(t)
	revision := f.s.Revision()
	before, err := f.s.approvals.RequestOf(SampleWorkspace, "issuance-final")
	f.must(err)
	first, err := readIssuanceSource(f, requester, SampleWorkspace, digest, SampleAdmin3)
	f.must(err)
	b := first.Basis
	if !validPlanHash(first.Digest) || b.Schema != "hb_issuance_source_v1" || b.Profile != financialFinalAmountProfile || b.Subject.Amount != "16" || b.Subject.Currency != "USD" || b.Net != "15.00" || b.Tax != "1.00" || b.Result != "public-result" || b.ApplicationReceipt == "" || b.Scale != 2 || len(b.Rows) != 2 || b.Rows[0].ID != "row-0" || b.Rows[0].Net != "10.25" || b.Rows[0].Tax != "0.25" || b.Rows[1].ID != "row-1" || b.Rows[1].Net != "4.75" || b.Rows[1].Tax != "0.75" || b.Decision.Actor != SampleAdmin {
		t.Fatalf("source facts mismatch: %+v", b)
	}
	contract, err := f.s.orgs.ContractRevision(SampleWorkspace, b.Contract.ContractID, b.Contract.Revision)
	f.must(err)
	document, err := f.s.orgs.DocumentKey(SampleWorkspace, b.BillingProfile.ID)
	f.must(err)
	if !reflect.DeepEqual(contract, b.Contract) || document != b.Document || b.Customer != contract.CustomerID || b.BillingRevision == 0 || b.FinalDigest != digest {
		t.Fatal("source party/contract provenance mismatch")
	}
	again, err := readIssuanceSource(f, requester, SampleWorkspace, digest, SampleAdmin3)
	f.must(err)
	if !reflect.DeepEqual(first, again) {
		t.Fatal("unchanged source is unstable")
	}
	first.Basis.Rows[0].Net = "999"
	first.Basis.Contributors[0] = "other"
	first.Basis.Subject.Digests["result"] = "changed"
	first.Basis.Contract.Components[0].ID = "changed"
	fresh, err := readIssuanceSource(f, requester, SampleWorkspace, digest, SampleAdmin3)
	f.must(err)
	if !reflect.DeepEqual(fresh, again) {
		t.Fatal("caller mutated retained provenance")
	}
	after, err := f.s.approvals.RequestOf(SampleWorkspace, "issuance-final")
	f.must(err)
	if f.s.Revision() != revision || !reflect.DeepEqual(before, after) || len(after.Executed) != 0 {
		t.Fatal("source read changed or consumed final approval")
	}
}

func TestFinancialIssuanceSourceRefusesUnapprovedAndWrongCaller(t *testing.T) {
	f, requester, decider, result := finalCommandWorld(t)
	p := finalRequest(f, requester, "issuance-final", result)
	f.ok(t, "request", p, "pending", "")
	_, err := readIssuanceSource(f, requester, SampleWorkspace, field(p, "digest"), SampleAdmin3)
	documentFailure(t, err, commands.PrerequisiteFailed)
	f.ok(t, "approve", f.run(decider, "erp_final.approve", "issuance-final", map[string]string{"digest": field(p, "digest")}), "approved", "")
	for _, c := range []struct {
		workspace, digest, consumer string
		code                        commands.Code
	}{{"foreign", field(p, "digest"), SampleAdmin3, commands.PermissionDenied}, {SampleWorkspace, field(p, "digest"), SampleAdmin, commands.PermissionDenied}, {SampleWorkspace, "wrong", SampleAdmin3, commands.EvidenceStale}} {
		_, err := readIssuanceSource(f, requester, c.workspace, c.digest, c.consumer)
		documentFailure(t, err, c.code)
	}
	err = inputAuthorityAs(f, requester, func() error {
		f.s.sessions.remove(f.s.caller)
		_, failure := f.s.financialIssuanceSourceLocked(SampleWorkspace, "issuance-final", field(p, "digest"), SampleAdmin3)
		return failure
	})
	documentFailure(t, err, commands.Unauthenticated)
}

func TestFinancialIssuanceSourceCurrentAuthorityAndProducer(t *testing.T) {
	for _, mode := range []string{"approver_right", "producer", "receipt"} {
		t.Run(mode, func(t *testing.T) {
			f, requester, _, digest := issuanceSourceWorld(t)
			switch mode {
			case "approver_right":
				for _, g := range f.s.admin.Grants() {
					if g.Terms.Kind == "erp_final_approval" && g.SubjectID == SampleAdmin {
						f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
					}
				}
			case "producer":
				f.s.chargeProducer = nil
			case "receipt":
				key := [2]string{SampleWorkspace, "public-result"}
				app := f.s.documentERP.resultApplications[key]
				app.receipt = "changed"
				f.s.documentERP.resultApplications[key] = app
			}
			revision := f.s.Revision()
			_, err := readIssuanceSource(f, requester, SampleWorkspace, digest, SampleAdmin3)
			documentFailure(t, err, map[string]commands.Code{"approver_right": commands.PermissionDenied, "producer": commands.UnsupportedCapability, "receipt": commands.EvidenceStale}[mode])
			if f.s.Revision() != revision {
				t.Fatal("invalid current source admitted or mutated")
			}
		})
	}
}
