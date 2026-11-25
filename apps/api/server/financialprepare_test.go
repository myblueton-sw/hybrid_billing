package server

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/authpolicy"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func chargePrepare(f world, a actor, id, digest string) reply {
	body, _ := json.Marshal(map[string]any{"request_id": "prepare-read", "target_id": id, "body": map[string]string{"digest": digest}})
	return f.call("financial_charge.prepare", string(body), a.cookie, a.csrf)
}

func prepareOutcomes(t *testing.T, r reply, approve, reject string) {
	t.Helper()
	if r.status != 200 || field(r, "approve") != approve || field(r, "reject") != reject {
		t.Fatal(r.raw)
	}
}

func TestFinancialPrepareExactFactsAndNoMutation(t *testing.T) {
	f, requester, decider, digest := chargeWorld(t)
	p := chargeRequest(f, requester, "prepare", digest)
	f.ok(t, "request", p, approvals.StatePending, "")
	before, err := f.s.approvals.RequestOf(SampleWorkspace, "prepare")
	f.must(err)
	revision := f.s.Revision()
	for range 2 {
		r := chargePrepare(f, decider, "prepare", field(p, "digest"))
		prepareOutcomes(t, r, "eligible", "eligible")
		for key, want := range map[string]string{"receipt": "current_decision_preparation", "amount": "15", "currency": "USD", "amount_basis": "rated_candidate_total", "author": SampleMember, "requester": SampleAdmin3, "policy_revision": "2", "revision": revision, "digest": field(p, "digest")} {
			if field(r, key) != want {
				t.Fatal(key, r.raw)
			}
		}
		if _, err := time.Parse(time.RFC3339Nano, field(r, "checked_at")); err != nil {
			t.Fatal(err)
		}
	}
	after, err := f.s.approvals.RequestOf(SampleWorkspace, "prepare")
	f.must(err)
	if !reflect.DeepEqual(before, after) || revision != f.s.Revision() {
		t.Fatal("preparation mutated request or revision")
	}
	f.code(t, "stale selection", chargePrepare(f, decider, "prepare", "stale"), 409, string(commands.EvidenceStale))
	f.ok(t, "approve", f.run(decider, "financial_charge.approve", "prepare", map[string]string{"digest": field(p, "digest")}), approvals.StateApproved, "")
	prepareOutcomes(t, chargePrepare(f, decider, "prepare", field(p, "digest")), "not_pending", "not_pending")
}

func TestFinancialPrepareSeparatePermissionsAndNondisclosure(t *testing.T) {
	f, requester, decider, digest := chargeWorld(t)
	p := chargeRequest(f, requester, "prepare", digest)
	f.ok(t, "request", p, approvals.StatePending, "")
	f.s.inputReviews = nil
	prepareOutcomes(t, chargePrepare(f, requester, "prepare", field(p, "digest")), "permission_denied", "permission_denied")
	for _, id := range []string{"prepare", "missing"} {
		r := chargePrepare(f, f.as(MemberLogin), id, field(p, "digest"))
		f.code(t, "no read", r, 403, string(commands.PermissionDenied))
		if field(r, "amount") != "" {
			t.Fatal("amount leaked", r.raw)
		}
	}
	f.must(f.s.admin.RevokeGrant("approval-"+SampleAdmin, 1))
	f.code(t, "decision grant not read", chargePrepare(f, decider, "prepare", field(p, "digest")), 403, string(commands.PermissionDenied))
}

func TestFinancialPrepareCurrentConditions(t *testing.T) {
	for _, mode := range []string{"amount", "permission", "source", "unavailable", "hold", "stepup", "capacity"} {
		t.Run(mode, func(t *testing.T) {
			f, requester, decider, digest := chargeWorld(t)
			p := chargeRequest(f, requester, "prepare", digest)
			f.ok(t, "request", p, approvals.StatePending, "")
			prepareOutcomes(t, chargePrepare(f, decider, "prepare", field(p, "digest")), "eligible", "eligible")
			a, b := "eligible", "eligible"
			switch mode {
			case "amount":
				v, err := f.s.financialAuthorityRecord(SampleWorkspace, "charge-limit")
				f.must(err)
				f.ok(t, "revoke amount", f.run(f.as(Admin2Login), "financial_authority.revoke", "charge-limit", map[string]string{"digest": v.proposal.Digest}), "revoked", "")
				a = "amount_authority_missing"
			case "permission":
				for _, g := range f.s.admin.Grants() {
					if g.Terms.Action == gate.Approve && g.Terms.Kind == "charge_approval" && g.SubjectID == SampleAdmin {
						f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
					}
				}
				a, b = "permission_denied", "permission_denied"
			case "source":
				for _, binding := range f.s.inputReviews.bindings {
					binding.Bytes = append([]byte(nil), binding.Bytes...)
					binding.Bytes[0] ^= 1
					break
				}
				a, b = "evidence_stale", "evidence_stale"
			case "unavailable":
				f.s.inputReviews = nil
				a, b = "unsupported_capability", "unsupported_capability"
			case "hold":
				f.s.enterHold("synthetic-test")
				a = "hold_active"
			case "stepup":
				f.advance(16 * time.Minute)
				a, b = CodeReauthentication, CodeReauthentication
			case "capacity":
				f.s.cfg.Commands.Fields = 14
				f.code(t, "bounded output", chargePrepare(f, decider, "prepare", field(p, "digest")), 501, string(commands.UnsupportedCapability))
				return
			}
			prepareOutcomes(t, chargePrepare(f, decider, "prepare", field(p, "digest")), a, b)
			r := f.runWith(decider, f.s.Revision(), "financial_charge.approve", "prepare", map[string]string{"digest": field(p, "digest")})
			if r.status == 200 {
				t.Fatal("preparation became future authorization", r.raw)
			}
		})
	}
}

func TestFinancialPrepareRequesterExclusion(t *testing.T) {
	f, requester, _, digest := chargeWorld(t)
	fields := financialPermissionFields(f)
	fields["recipient"] = SampleAdmin3
	proposal := f.run(f.as(AdminLogin), "financial_permission.propose", "requester-decision", fields)
	f.ok(t, "permission proposal", proposal, "pending", "")
	result := f.run(f.as(Admin2Login), "financial_permission.approve", "requester-decision", map[string]string{"digest": field(proposal, "digest")})
	f.ok(t, "permission", result, "applied", "")
	p := chargeRequest(f, requester, "prepare", digest)
	f.ok(t, "request", p, approvals.StatePending, "")
	prepareOutcomes(t, chargePrepare(f, requester, "prepare", field(p, "digest")), "self_approval_denied", "self_approval_denied")
}

func TestFinancialPrepareAuthorAndBindingBoundaries(t *testing.T) {
	f, requester, decider, digest := chargeWorld(t)
	chargeDecisionGrant(t, f, SampleMember, "author-decision")
	terms := gate.Terms{Action: gate.View, Kind: approvalKind, Fields: []string{approvalField}, Period: gate.AnyPeriod(), Scope: gate.Scope{WorkspaceID: SampleWorkspace, Customer: gate.All(), LegalEntity: gate.All(), Contract: gate.All()}}
	f.must(f.s.admin.AddGrant(gate.Grant{ID: "author-read", SubjectID: SampleMember, Terms: terms, Validity: gate.Validity{NotBefore: f.s.clock(), NotAfter: f.s.clock().Add(time.Hour)}}))
	p := chargeRequest(f, requester, "prepare", digest)
	f.ok(t, "request", p, approvals.StatePending, "")
	prepareOutcomes(t, chargePrepare(f, f.as(MemberLogin), "prepare", field(p, "digest")), "self_approval_denied", "self_approval_denied")
	for _, mode := range []string{"missing", "foreign", "corrupt"} {
		id := "missing"
		if mode != "missing" {
			id = "prepare"
			binding := f.s.financialCharges[id]
			if mode == "foreign" {
				binding.Workspace = "foreign"
			} else {
				binding.Workspace = SampleWorkspace
				binding.Digest = "bad"
			}
			f.s.financialCharges[id] = binding
		}
		got := chargePrepare(f, decider, id, field(p, "digest"))
		f.code(t, mode, got, 403, string(commands.PermissionDenied))
		if field(got, "amount") != "" {
			t.Fatal("binding disclosed", got.raw)
		}
	}
}

func TestFinancialPreparePolicyAndStrandedEffects(t *testing.T) {
	f, requester, decider, digest := chargeWorld(t)
	p := chargeRequest(f, requester, "prepare", digest)
	f.ok(t, "request", p, approvals.StatePending, "")
	f.s.stranded = []uint64{1}
	prepareOutcomes(t, chargePrepare(f, decider, "prepare", field(p, "digest")), "dependency_unavailable", "dependency_unavailable")
	f.s.stranded = nil
	subjectPolicy(t, f, "replacement-policy", "3", "attribution,input,mapping,result,rule")
	prepareOutcomes(t, chargePrepare(f, f.as(AdminLogin), "prepare", field(p, "digest")), "not_pending", "not_pending")
}

func TestFinancialPrepareRequiredMFAFailsClosed(t *testing.T) {
	f, requester, decider, digest := chargeWorld(t)
	p := chargeRequest(f, requester, "prepare", digest)
	f.ok(t, "request", p, approvals.StatePending, "")
	f.directPolicy(authpolicy.Revision{MaxAge: time.Hour, StepUp: []authpolicy.StepUp{{Action: ActionSensitive, MaxAge: 30 * time.Minute, MFA: true}}})
	f.code(t, "policy invalidated session", chargePrepare(f, decider, "prepare", field(p, "digest")), 401, string(commands.Unauthenticated))
	prepareOutcomes(t, chargePrepare(f, f.as(AdminLogin), "prepare", field(p, "digest")), "prerequisite_failed", "prerequisite_failed")
}
