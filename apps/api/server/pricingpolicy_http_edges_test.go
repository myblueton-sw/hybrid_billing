package server

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/pricingauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/pricingpolicy"
)

func TestPricingPolicyHTTPHoldAndStepUp(t *testing.T) {
	f, maker, checker, spec, _ := policyHTTPWorld(t)
	id := "hold-policy"
	prepared := policyRead(f, maker, "prepare", id, map[string]string{"spec": spec})
	proposal := policyMutate(f, maker, "propose", id, policyProposalFields(prepared, spec))
	f.ok(t, "proposed", proposal, pricingpolicy.Pending, "")
	f.advance(16 * time.Minute)
	f.code(t, "approval step-up", policyMutate(f, checker, "approve", id, map[string]string{"digest": field(proposal, "digest")}), 403, CodeReauthentication)
	if got := policyRead(f, maker, "inspect", id, nil); got.status != 200 {
		t.Fatal("read requires step-up", got.raw)
	}
	checker = f.as(Admin3Login)
	f.ok(t, "fresh approval", policyMutate(f, checker, "approve", id, map[string]string{"digest": field(proposal, "digest")}), pricingpolicy.Approved, "")
	f.s.domain.Lock()
	f.s.enterHold("synthetic-test")
	f.s.domain.Unlock()
	f.code(t, "hold approve", policyMutate(f, checker, "approve", id, map[string]string{"digest": field(proposal, "digest")}), 503, string(commands.DependencyUnavailable))
	f.code(t, "hold propose", policyMutate(f, maker, "propose", "new-held-policy", policyProposalFields(prepared, spec)), 503, string(commands.DependencyUnavailable))
	if got := policyRead(f, maker, "inspect", id, nil); got.status != 200 {
		t.Fatal("held read", got.raw)
	}
	if got := policyRead(f, maker, "prepare", "new-held-policy", map[string]string{"spec": spec}); got.status != 200 {
		t.Fatal("held preparation", got.raw)
	}
	f.ok(t, "held revoke", policyMutate(f, checker, "revoke", id, map[string]string{"digest": field(proposal, "digest")}), pricingpolicy.Revoked, "")
}

func TestPricingPolicyHTTPHistoricalReplayAndCurrentAuthority(t *testing.T) {
	f, maker, checker, spec, grant := policyHTTPWorld(t)
	id := "replay-policy"
	prepared := policyRead(f, maker, "prepare", id, map[string]string{"spec": spec})
	body, _ := json.Marshal(map[string]any{"request_id": "policy-replay", "target_id": id, "idempotency_key": "policy-replay-key", "expected_revision": f.s.Revision(), "rationale": "sample reason", "body": policyProposalFields(prepared, spec)})
	proposal := f.call("pricing_policy.propose", string(body), maker.cookie, maker.csrf)
	f.ok(t, "proposed", proposal, pricingpolicy.Pending, "")
	f.ok(t, "approve", policyMutate(f, checker, "approve", id, map[string]string{"digest": field(proposal, "digest")}), pricingpolicy.Approved, "")
	before := f.s.counter
	replay := f.call("pricing_policy.propose", string(body), maker.cookie, maker.csrf)
	if replay.status != 200 || field(replay, "recorded_state") != pricingpolicy.Pending || field(replay, "receipt") != "historical_decision" || f.s.counter != before {
		t.Fatal("replay was not historical", replay.raw)
	}
	current := policyRead(f, maker, "inspect", id, nil)
	if current.status != 200 || field(current, "recorded_state") != pricingpolicy.Approved {
		t.Fatal("current record", current.raw)
	}
	if err := f.s.admin.RevokeGrant(grant, 1); err != nil {
		t.Fatal(err)
	}
	f.code(t, "revoked replay", f.call("pricing_policy.propose", string(body), maker.cookie, maker.csrf), 403, string(commands.PermissionDenied))
}

func TestPricingPolicyHTTPInspectionGuardsAndInputErrors(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	member := f.as(MemberLogin)
	ceiling, grant := provisionPricingCapability(t, f, "inspect", SampleAdmin3)
	provisionPricingCapability(t, f, "use", SampleMember)
	terms, _ := pricingGrantAuthority.terms(SampleWorkspace, "inspect")
	if err := f.s.admin.AddDelegation(gate.Delegation{ID: "http-policy-delegation", DelegatorID: SampleAdmin3, DelegateID: SampleMember, Terms: terms, Validity: gate.Validity{NotBefore: f.s.clock(), NotAfter: f.s.clock().Add(time.Hour)}}); err != nil {
		t.Fatal(err)
	}
	spec := policySpecText(t, policyHTTPSpec())
	f.code(t, "delegated prepare", policyRead(f, member, "prepare", "guard-policy", map[string]string{"spec": spec}), 403, string(commands.PermissionDenied))
	f.code(t, "delegated inspect", policyRead(f, member, "inspect", "guard-policy", nil), 403, string(commands.PermissionDenied))
	third := f.as(Admin3Login)
	for _, raw := range []string{`{}`, `{"Mode":"synthetic","Mode":"synthetic"}`, strings.Repeat(" ", 4097), strings.Replace(spec, `"Lines":1000`, `"Lines":1000.5`, 1)} {
		f.code(t, "invalid JSON", policyRead(f, third, "prepare", "guard-policy", map[string]string{"spec": raw}), 422, string(commands.InvalidRequest))
	}
	foreign := strings.Replace(spec, `"Workspace":"`+SampleWorkspace+`"`, `"Workspace":"foreign"`, 1)
	f.code(t, "foreign workspace", policyRead(f, third, "prepare", "guard-policy", map[string]string{"spec": foreign}), 403, string(commands.PermissionDenied))
	for _, id := range []string{grant, ceiling} {
		if err := f.s.admin.RevokeGrant(id, 1); err != nil {
			t.Fatal(err)
		}
		f.code(t, "revoked inspection", policyRead(f, third, "prepare", "guard-policy", map[string]string{"spec": spec}), 403, string(commands.PermissionDenied))
	}
	if _, err := f.s.chargeProducer.pricingPolicies.Record(SampleWorkspace, "guard-policy"); !errors.Is(err, pricingpolicy.ErrNotFound) {
		t.Fatal("read stored proposal", err)
	}
}

func TestPricingPolicyHTTPConcurrentApprovalSerializes(t *testing.T) {
	f, maker, checker, spec, _ := policyHTTPWorld(t)
	id := "concurrent-policy"
	prepared := policyRead(f, maker, "prepare", id, map[string]string{"spec": spec})
	proposal := policyMutate(f, maker, "propose", id, policyProposalFields(prepared, spec))
	f.ok(t, "proposed", proposal, pricingpolicy.Pending, "")
	revision := f.s.Revision()
	results := make([]reply, 2)
	var wg sync.WaitGroup
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i] = f.runWith(checker, revision, "pricing_policy.approve", id, map[string]string{"digest": field(proposal, "digest")})
		}(i)
	}
	wg.Wait()
	success, conflict := 0, 0
	for _, result := range results {
		if result.status == 200 {
			success++
		} else if result.status == 409 && result.body["code"] == string(commands.RevisionConflict) {
			conflict++
		} else {
			t.Fatal(result.raw)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatal("concurrent outcomes", success, conflict)
	}
	record, err := f.s.chargeProducer.pricingPolicies.Record(SampleWorkspace, id)
	if err != nil || record.State != pricingpolicy.Approved || record.Approval == nil {
		t.Fatal(record, err)
	}
}

func TestPricingPolicyDefinitionsAndSeedNone(t *testing.T) {
	f := setup(t, SeedNone)
	for _, definition := range f.s.pricingPolicyDefinitions() {
		h := definition.Handler.(pricingPolicyCommand)
		if _, err := h.Apply(commands.Context{Workspace: f.s.bootWorkspace}); err == nil {
			t.Fatal("SeedNone supported", definition.ID)
		}
		if definition.Permission.Kind != pricingauthority.ResourceKind {
			t.Fatal("wrong kind", definition.ID)
		}
		if !definition.Mutating && (definition.Permission.Action != gate.ViewCost || len(definition.Permission.Fields) != 1 || definition.Permission.Fields[0] != pricingauthority.EvidenceField) {
			t.Fatal("wrong evidence permission", definition.ID)
		}
	}
}
