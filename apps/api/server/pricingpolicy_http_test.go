package server

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/pricingpolicy"
	"github.com/myblueton-sw/hybrid_billing/modules/rating"
)

func policyHTTPSpec() pricingpolicy.Spec {
	return pricingpolicy.Spec{Mode: "synthetic", Scope: pricingpolicy.Scope{Workspace: SampleWorkspace, Customer: "sample-le-north", Entity: "sample-le-seller", Contract: "sample-contract-01", BillingProfile: "sample-profile-north"},
		From: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), Profile: candidateProfile(),
		Limits: rating.Limits{Rules: 20, Tiers: 20, Discounts: 20, Lines: 1000, Intermediate: 60},
		Policy: rating.Policy{ID: "http-policy", Version: "1", Currency: "USD", CostBasis: rating.Billed, Quantum: "0.01", Mode: "half_even", Path: rating.Resale, Eligible: []rating.Class{rating.CSPUsage}, Rules: []rating.Rule{{ID: "pass", Level: rating.Default, Type: rating.PassThrough}}}}
}

func policySpecText(t *testing.T, spec pricingpolicy.Spec) string {
	t.Helper()
	text, err := pricingSpecJSON(spec)
	if err != nil {
		t.Fatal(err)
	}
	return text
}

func policyRead(f world, a actor, kind, id string, fields map[string]string) reply {
	if fields == nil {
		fields = map[string]string{}
	}
	body, _ := json.Marshal(map[string]any{"request_id": "policy-read", "target_id": id, "body": fields})
	return f.call("pricing_policy."+kind, string(body), a.cookie, a.csrf)
}

func policyProposalFields(prepared reply, spec string) map[string]string {
	fields := map[string]string{"spec": spec}
	for _, name := range []string{"contract_revision", "billing_revision", "target_fingerprint", "preparation_digest"} {
		fields[name] = field(prepared, name)
	}
	return fields
}

func policyHTTPWorld(t *testing.T, beforeFinance ...func(world)) (world, actor, actor, string, string) {
	t.Helper()
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	if len(beforeFinance) > 1 {
		t.Fatal("multiple financial setup callbacks")
	}
	if len(beforeFinance) == 1 && beforeFinance[0] != nil {
		beforeFinance[0](f)
	}
	provisionPricingCapability(t, f, "inspect", SampleMember)
	_, proposerGrant := provisionPricingCapability(t, f, "propose", SampleMember)
	provisionPricingCapability(t, f, "approve", SampleAdmin3)
	provisionPricingCapability(t, f, "revoke", SampleAdmin3)
	return f, f.as(MemberLogin), f.as(Admin3Login), policySpecText(t, policyHTTPSpec()), proposerGrant
}

func TestPricingPolicyHTTPPrepareProposeApproveRevoke(t *testing.T) {
	f, maker, checker, spec, _ := policyHTTPWorld(t)
	id := "http-policy-record"
	before := f.s.Revision()
	prepared := policyRead(f, maker, "prepare", id, map[string]string{"spec": spec})
	if prepared.status != 200 || prepared.body["result_revision"] != before || f.s.Revision() != before || len(field(prepared, "preparation_digest")) != 64 || field(prepared, "contract_revision") != "2" || field(prepared, "billing_revision") != "1" {
		t.Fatal("preparation", prepared.raw)
	}
	if _, err := f.s.chargeProducer.pricingPolicies.Record(SampleWorkspace, id); !errors.Is(err, pricingpolicy.ErrNotFound) {
		t.Fatal("prepare stored a proposal", err)
	}
	f.code(t, "admin has no policy cost", policyRead(f, f.as(AdminLogin), "prepare", id, map[string]string{"spec": spec}), 403, string(commands.PermissionDenied))
	f.code(t, "approver lacks inspection", policyRead(f, checker, "prepare", id, map[string]string{"spec": spec}), 403, string(commands.PermissionDenied))
	proposal := f.runWith(maker, before, "pricing_policy.propose", id, policyProposalFields(prepared, spec))
	f.ok(t, "proposed", proposal, pricingpolicy.Pending, "")
	if field(proposal, "spec") != "" || field(proposal, "receipt") != "historical_decision" {
		t.Fatal("mutation leaked cost terms", proposal.raw)
	}
	f.code(t, "maker lacks approval", policyMutate(f, maker, "approve", id, map[string]string{"digest": field(proposal, "digest")}), 403, string(commands.PermissionDenied))
	approved := policyMutate(f, checker, "approve", id, map[string]string{"digest": field(proposal, "digest")})
	f.ok(t, "approved", approved, pricingpolicy.Approved, "")
	before = f.s.Revision()
	inspected := policyRead(f, maker, "inspect", id, nil)
	if inspected.status != 200 || field(inspected, "current_target") != "matched" || field(inspected, "approval_by") != SampleAdmin3 || field(inspected, "spec") != spec || f.s.Revision() != before {
		t.Fatal("inspection", inspected.raw)
	}
	f.code(t, "approver lacks record cost", policyRead(f, checker, "inspect", id, nil), 403, string(commands.PermissionDenied))
	profile, revision, err := f.s.orgs.BillingProfile(SampleWorkspace, "sample-profile-north")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.s.orgs.PutBillingProfile(profile, revision); err != nil {
		t.Fatal(err)
	}
	inspected = policyRead(f, maker, "inspect", id, nil)
	if inspected.status != 200 || field(inspected, "current_target") != "changed" {
		t.Fatal("stale inspection", inspected.raw)
	}
	revoked := policyMutate(f, checker, "revoke", id, map[string]string{"digest": field(proposal, "digest")})
	f.ok(t, "stale target can revoke", revoked, pricingpolicy.Revoked, "")
	inspected = policyRead(f, maker, "inspect", id, nil)
	if inspected.status != 200 || field(inspected, "recorded_state") != pricingpolicy.Revoked || field(inspected, "approval_by") != SampleAdmin3 || field(inspected, "revocation_by") != SampleAdmin3 || field(inspected, "spec") != spec {
		t.Fatal("history changed", inspected.raw)
	}
}

func TestPricingPolicyHTTPPreparationBinding(t *testing.T) {
	for _, cause := range []string{"terms", "target-id", "profile-revision", "fingerprint", "canonical-revision", "proposal-grant", "global-revision"} {
		t.Run(cause, func(t *testing.T) {
			f, maker, _, spec, grant := policyHTTPWorld(t)
			id := "prepared-policy"
			prepared := policyRead(f, maker, "prepare", id, map[string]string{"spec": spec})
			if prepared.status != 200 {
				t.Fatal(prepared.raw)
			}
			fields := policyProposalFields(prepared, spec)
			expected := prepared.body["result_revision"].(string)
			status, code := 409, commands.EvidenceStale
			switch cause {
			case "terms":
				fields["spec"] = strings.Replace(spec, `"Version":"1"`, `"Version":"2"`, 1)
			case "target-id":
				id = "other-policy"
			case "profile-revision":
				profile, rev, err := f.s.orgs.BillingProfile(SampleWorkspace, "sample-profile-north")
				if err != nil {
					t.Fatal(err)
				}
				if _, err = f.s.orgs.PutBillingProfile(profile, rev); err != nil {
					t.Fatal(err)
				}
			case "fingerprint":
				fields["target_fingerprint"] = strings.Repeat("0", 64)
			case "canonical-revision":
				fields["contract_revision"] = "02"
			case "proposal-grant":
				if err := f.s.admin.RevokeGrant(grant, 1); err != nil {
					t.Fatal(err)
				}
				status, code = 403, commands.PermissionDenied
			case "global-revision":
				provisionPricingCapability(t, f, "use", SampleMember)
				code = commands.RevisionConflict
			}
			f.code(t, cause, f.runWith(maker, expected, "pricing_policy.propose", id, fields), status, string(code))
			if _, err := f.s.chargeProducer.pricingPolicies.Record(SampleWorkspace, id); !errors.Is(err, pricingpolicy.ErrNotFound) {
				t.Fatal("failed proposal stored", err)
			}
		})
	}
}

func TestPricingPolicyHTTPCurrentProposerAndSelfApproval(t *testing.T) {
	f, maker, checker, spec, grant := policyHTTPWorld(t)
	id := "current-proposer"
	p := policyRead(f, maker, "prepare", id, map[string]string{"spec": spec})
	proposal := policyMutate(f, maker, "propose", id, policyProposalFields(p, spec))
	f.ok(t, "proposal", proposal, pricingpolicy.Pending, "")
	before, _ := f.s.chargeProducer.pricingPolicies.Record(SampleWorkspace, id)
	if err := f.s.admin.RevokeGrant(grant, 1); err != nil {
		t.Fatal(err)
	}
	f.code(t, "revoked proposer", policyMutate(f, checker, "approve", id, map[string]string{"digest": field(proposal, "digest")}), 403, string(commands.PermissionDenied))
	after, _ := f.s.chargeProducer.pricingPolicies.Record(SampleWorkspace, id)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("failed approval mutated history")
	}
	f = setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	for _, capability := range []string{"inspect", "propose", "approve"} {
		provisionPricingCapability(t, f, capability, SampleMember)
	}
	maker = f.as(MemberLogin)
	p = policyRead(f, maker, "prepare", id, map[string]string{"spec": spec})
	proposal = policyMutate(f, maker, "propose", id, policyProposalFields(p, spec))
	f.ok(t, "self proposal", proposal, pricingpolicy.Pending, "")
	f.code(t, "self approval", policyMutate(f, maker, "approve", id, map[string]string{"digest": field(proposal, "digest")}), 403, string(commands.SelfApprovalDenied))
}

func policyMutate(f world, a actor, kind, id string, fields map[string]string) reply {
	return f.runWith(a, f.s.Revision(), "pricing_policy."+kind, id, fields)
}
