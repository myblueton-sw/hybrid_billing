package server

import (
	"errors"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func currentCharge(f world, a actor, workspace, id, digest, consumer string) (approvals.Subject, error) {
	var subject approvals.Subject
	err := inputAuthorityAs(f, a, func() error {
		var err error
		subject, err = f.s.currentFinancialChargeLocked(workspace, id, digest, consumer)
		return err
	})
	return subject, err
}

func approvedCharge(t *testing.T) (world, actor, string) {
	t.Helper()
	f, requester, decider, digest := chargeWorld(t)
	p := chargeRequest(f, requester, "current-charge", digest)
	f.ok(t, "request", p, approvals.StatePending, "")
	f.ok(t, "approve", f.run(decider, "financial_charge.approve", "current-charge", map[string]string{"digest": field(p, "digest")}), approvals.StateApproved, "")
	return f, f.as(Admin2Login), field(p, "digest")
}

func TestCurrentFinancialChargeDoesNotConsumeOrAuthorizeConsumer(t *testing.T) {
	f, consumer, digest := approvedCharge(t)
	before := f.s.Revision()
	for i := 0; i < 2; i++ {
		s, err := currentCharge(f, consumer, SampleWorkspace, "current-charge", digest, SampleAdmin2)
		if err != nil || s.Amount != "15" || s.Currency != "USD" || s.Author != SampleMember {
			t.Fatal(s, err)
		}
		s.Digests["input"] = "mutated"
	}
	v, err := f.s.approvals.RequestOf(SampleWorkspace, "current-charge")
	if err != nil || len(v.Executed) != 0 || f.s.Revision() != before {
		t.Fatal(v, err)
	}
	for _, action := range []gate.Action{gate.Approve, gate.Issue} {
		for _, grant := range f.s.admin.Grants() {
			if grant.SubjectID == SampleAdmin2 && grant.Terms.Action == action && grant.Terms.Kind == "charge_approval" && grant.Active {
				t.Fatal("consumer gained authority")
			}
		}
	}
}

func TestCurrentFinancialChargeRefusesChangedAuthorityAndBindings(t *testing.T) {
	for _, mode := range []string{"gate-revoked", "gate-and-producer", "limit-revoked", "wrong-consumer", "foreign", "digest", "binding", "policy", "producer"} {
		t.Run(mode, func(t *testing.T) {
			f, a, digest := approvedCharge(t)
			workspace, consumer := SampleWorkspace, SampleAdmin2
			switch mode {
			case "gate-revoked", "gate-and-producer":
				if mode == "gate-and-producer" {
					f.s.chargeProducer = nil
				}
				for _, grant := range f.s.admin.Grants() {
					if grant.SubjectID == SampleAdmin && grant.Terms.Action == gate.Approve {
						f.must(f.s.admin.RevokeGrant(grant.ID, grant.Version))
					}
				}
			case "limit-revoked":
				r, err := f.s.financialAuthorityRecord(SampleWorkspace, "charge-limit")
				f.must(err)
				f.ok(t, "revoke", f.run(a, "financial_authority.revoke", "charge-limit", map[string]string{"digest": r.proposal.Digest}), "revoked", "")
			case "wrong-consumer":
				consumer = SampleAdmin
			case "foreign":
				workspace = "foreign"
			case "digest":
				digest = "changed"
			case "binding":
				b := f.s.financialCharges["current-charge"]
				b.Subject.Amount = "16"
				f.s.financialCharges["current-charge"] = b
			case "policy":
				subjectPolicy(t, f, "replacement", "3", "attribution,input,mapping,result,rule")
			case "producer":
				f.s.chargeProducer = nil
			}
			_, err := currentCharge(f, a, workspace, "current-charge", digest, consumer)
			var failure *commands.Failure
			if !errors.As(err, &failure) {
				t.Fatal("expected closed refusal", err)
			}
			expected := map[string]commands.Code{"gate-revoked": commands.PermissionDenied, "gate-and-producer": commands.PermissionDenied, "limit-revoked": commands.PrerequisiteFailed, "wrong-consumer": commands.PermissionDenied, "foreign": commands.PermissionDenied, "digest": commands.EvidenceStale, "binding": commands.EvidenceStale, "policy": commands.PrerequisiteFailed, "producer": commands.UnsupportedCapability}[mode]
			if failure.Code != expected {
				t.Fatal(failure)
			}
			v, err := f.s.approvals.RequestOf(SampleWorkspace, "current-charge")
			if err != nil || len(v.Executed) != 0 {
				t.Fatal("inspection consumed approval", err)
			}
		})
	}
}

func TestCurrentFinancialChargeRejectsHistoricalReplayAfterRevocation(t *testing.T) {
	f, a, digest := approvedCharge(t)
	subject, err := currentCharge(f, a, SampleWorkspace, "current-charge", digest, SampleAdmin2)
	f.must(err)
	e := approvals.Execution{ID: "historical-effect", Request: subject.ID, Stage: subject.Stage, Target: subject.Target, Amount: subject.Amount, Currency: subject.Currency, Digests: subject.Digests}
	actor := approvals.Actor{ID: SampleAdmin2, Kind: approvals.KindHuman}
	_, err = f.s.approvals.Execute(SampleWorkspace, e, actor)
	f.must(err)
	_, err = f.s.approvals.Report(SampleWorkspace, e.ID, approvals.Outcome{Result: approvals.ResultSucceeded}, actor)
	f.must(err)
	for _, grant := range f.s.admin.Grants() {
		if grant.SubjectID == SampleAdmin && grant.Terms.Action == gate.Approve {
			f.must(f.s.admin.RevokeGrant(grant.ID, grant.Version))
		}
	}
	if _, err = f.s.approvals.Execute(SampleWorkspace, e, actor); err != nil {
		t.Fatal("historical module replay changed", err)
	}
	if _, err = currentCharge(f, a, SampleWorkspace, "current-charge", digest, SampleAdmin2); err == nil {
		t.Fatal("historical execution bypassed current grant")
	}
}

func TestCurrentFinancialChargePendingRejectedAndMissing(t *testing.T) {
	for _, mode := range []string{"pending", "rejected", "missing"} {
		t.Run(mode, func(t *testing.T) {
			f, requester, decider, digest := chargeWorld(t)
			p := chargeRequest(f, requester, "current-charge", digest)
			f.ok(t, "request", p, approvals.StatePending, "")
			if mode == "rejected" {
				f.ok(t, "reject", f.run(decider, "financial_charge.reject", "current-charge", map[string]string{"digest": field(p, "digest"), "reason": "other"}), approvals.StateRejected, "")
			}
			id := "current-charge"
			if mode == "missing" {
				id = "missing"
			}
			if _, err := currentCharge(f, f.as(Admin2Login), SampleWorkspace, id, field(p, "digest"), SampleAdmin2); err == nil {
				t.Fatal(mode)
			}
		})
	}
}

func TestCurrentFinancialChargeMissingAndInactiveIdentity(t *testing.T) {
	f, a, digest := approvedCharge(t)
	f.s.domain.Lock()
	_, err := f.s.currentFinancialChargeLocked(SampleWorkspace, "current-charge", digest, SampleAdmin2)
	f.s.domain.Unlock()
	if err == nil {
		t.Fatal("no actual caller")
	}
	u, _, err := f.s.findUser(SampleAdmin)
	f.must(err)
	f.must(f.s.identity.Installation().SuspendUser(seedOperator, SampleAdmin, u.Epoch))
	_, err = currentCharge(f, a, SampleWorkspace, "current-charge", digest, SampleAdmin2)
	if err == nil {
		t.Fatal("inactive recorded approver")
	}
}

func TestCurrentFinancialChargeDomainLockOrdersRevocation(t *testing.T) {
	f, a, digest := approvedCharge(t)
	var grantID string
	var version uint64
	for _, g := range f.s.admin.Grants() {
		if g.SubjectID == SampleAdmin && g.Terms.Action == gate.Approve {
			grantID, version = g.ID, g.Version
		}
	}
	if grantID == "" {
		t.Fatal("missing fixture grant")
	}
	revoking, revoked := make(chan struct{}), make(chan error, 1)
	err := inputAuthorityAs(f, a, func() error {
		go func() {
			close(revoking)
			f.s.domain.Lock()
			defer f.s.domain.Unlock()
			revoked <- f.s.admin.RevokeGrant(grantID, version)
		}()
		<-revoking
		_, err := f.s.currentFinancialChargeLocked(SampleWorkspace, "current-charge", digest, SampleAdmin2)
		select {
		case <-revoked:
			t.Fatal("revocation crossed the held domain lock")
		default:
		}
		return err
	})
	f.must(err)
	f.must(<-revoked)
	_, err = currentCharge(f, a, SampleWorkspace, "current-charge", digest, SampleAdmin2)
	var failure *commands.Failure
	if !errors.As(err, &failure) || failure.Code != commands.PermissionDenied {
		t.Fatal(err)
	}
}
