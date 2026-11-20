package server

import (
	"errors"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func TestFinancialChargeDependencyMappingAndStaleProducer(t *testing.T) {
	for _, code := range []commands.Code{commands.DependencyUnavailable, commands.AuthorityUnavailable, commands.UnsupportedCapability, commands.PrerequisiteFailed} {
		var failure *commands.Failure
		if !errors.As(financialChargeEvidenceError(commands.Fail(code)), &failure) || failure.Code != code {
			t.Fatal(code, failure)
		}
	}
	f, requester, decider, digest := chargeWorld(t)
	p := chargeRequest(f, requester, "source-current", digest)
	f.ok(t, "request", p, approvals.StatePending, "")
	saved := f.s.inputReviews
	f.s.inputReviews = nil
	f.code(t, "producer unavailable request", chargeRequest(f, requester, "source-unavailable", digest), 501, string(commands.UnsupportedCapability))
	f.code(t, "producer unavailable decision", f.run(decider, "financial_charge.approve", "source-current", map[string]string{"digest": field(p, "digest")}), 501, string(commands.UnsupportedCapability))
	f.s.inputReviews = saved
	for _, b := range saved.bindings {
		b.Bytes = append([]byte(nil), b.Bytes...)
		b.Bytes[0] ^= 1
		break
	}
	f.code(t, "source stale request", chargeRequest(f, requester, "source-stale", digest), 409, string(commands.EvidenceStale))
	f.code(t, "source stale decision", f.run(decider, "financial_charge.approve", "source-current", map[string]string{"digest": field(p, "digest")}), 409, string(commands.EvidenceStale))
	v, err := f.s.approvals.RequestOf(SampleWorkspace, "source-current")
	if err != nil || v.State != approvals.StatePending || len(v.Decisions) != 0 {
		t.Fatal(v, err)
	}
}

func TestFinancialChargeBindingDefensiveAndNondisclosing(t *testing.T) {
	f, requester, decider, digest := chargeWorld(t)
	p := chargeRequest(f, requester, "bound", digest)
	f.ok(t, "request", p, approvals.StatePending, "")
	_, b, err := f.s.financialChargeRecord(SampleWorkspace, "bound")
	f.must(err)
	b.Subject.Digests["input"] = "changed"
	if _, _, err = f.s.financialChargeRecord(SampleWorkspace, "bound"); err != nil {
		t.Fatal("binding alias", err)
	}
	for _, mode := range []string{"missing", "foreign", "corrupt"} {
		id := "missing"
		if mode != "missing" {
			id = "bound"
			v := f.s.financialCharges[id]
			if mode == "foreign" {
				v.Workspace = "foreign"
			} else {
				v.Workspace = SampleWorkspace
				v.Subject.Author = "different"
			}
			f.s.financialCharges[id] = v
		}
		f.code(t, mode, f.run(decider, "financial_charge.approve", id, map[string]string{"digest": field(p, "digest")}), 403, string(commands.PermissionDenied))
	}
}

func TestFinancialChargeImpactAndHold(t *testing.T) {
	f, requester, decider, digest := chargeWorld(t)
	p := chargeRequest(f, requester, "pending-impact", digest)
	f.ok(t, "request", p, approvals.StatePending, "")
	counts, err := financialChargeTally(f.s)
	if err != nil || counts[SampleAdmin3] != 1 {
		t.Fatal(counts, err)
	}
	f.s.fault = func(at string) bool { return at == "financial-charge-impact" }
	if _, err := financialChargeTally(f.s); err == nil {
		t.Fatal("impact unavailable became zero")
	}
	f.s.fault = nil
	f.s.domain.Lock()
	f.s.enterHold("synthetic-test")
	f.s.domain.Unlock()
	f.code(t, "held request", chargeRequest(f, requester, "held", digest), 503, string(commands.DependencyUnavailable))
	f.code(t, "held approve", f.run(decider, "financial_charge.approve", "pending-impact", map[string]string{"digest": field(p, "digest")}), 503, string(commands.DependencyUnavailable))
	if got := chargeRead(f, decider, "pending-impact"); got.status != 200 {
		t.Fatal(got.raw)
	}
	f.ok(t, "held reject", f.run(decider, "financial_charge.reject", "pending-impact", map[string]string{"digest": field(p, "digest"), "reason": "missing_evidence"}), approvals.StateRejected, "")
	counts, err = financialChargeTally(f.s)
	if err != nil || counts[SampleAdmin3] != 0 {
		t.Fatal(counts, err)
	}
}

func TestFinancialChargeStepUp(t *testing.T) {
	f, requester, decider, digest := chargeWorld(t)
	p := chargeRequest(f, requester, "limit", digest)
	f.ok(t, "request", p, approvals.StatePending, "")
	f.advance(16 * time.Minute)
	f.code(t, "step-up", f.run(decider, "financial_charge.approve", "limit", map[string]string{"digest": field(p, "digest")}), 403, CodeReauthentication)
}

func TestFinancialChargeAmountCurrencyAndDates(t *testing.T) {
	for _, mode := range []string{"below", "currency", "expired", "future"} {
		t.Run(mode, func(t *testing.T) {
			f, requester, _, digest := chargeWorld(t)
			p := chargeRequest(f, requester, "bounded", digest)
			f.ok(t, "request", p, approvals.StatePending, "")
			old, err := f.s.financialAuthorityRecord(SampleWorkspace, "charge-limit")
			f.must(err)
			f.ok(t, "remove original limit", f.run(f.as(Admin2Login), "financial_authority.revoke", "charge-limit", map[string]string{"digest": old.proposal.Digest}), "revoked", "")
			fields := financialAuthorityFields(f)
			fields["recipient"] = SampleAdmin
			fields["limit"] = "15"
			switch mode {
			case "below":
				fields["limit"] = "14.99"
			case "currency":
				fields["currency"] = "EUR"
			case "expired":
				fields["from"] = f.s.clock().Add(-48 * time.Hour).Format(time.DateOnly)
				fields["to"] = f.s.clock().Add(-24 * time.Hour).Format(time.DateOnly)
			case "future":
				fields["from"] = f.s.clock().Add(24 * time.Hour).Format(time.DateOnly)
			}
			q := f.run(f.as(Admin2Login), "financial_authority.propose", "bounded-limit", fields)
			f.ok(t, "bounded proposal", q, approvals.StatePending, "")
			f.ok(t, "bounded confirm", f.run(f.as(Admin3Login), "financial_authority.confirm", "bounded-limit", map[string]string{"digest": field(q, "digest")}), approvals.StateConfirmed, "")
			f.code(t, mode, f.run(f.as(AdminLogin), "financial_charge.approve", "bounded", map[string]string{"digest": field(p, "digest")}), 403, string(commands.PermissionDenied))
			v, err := f.s.approvals.RequestOf(SampleWorkspace, "bounded")
			if err != nil || len(v.Decisions) != 0 {
				t.Fatal(v, err)
			}
		})
	}
}
