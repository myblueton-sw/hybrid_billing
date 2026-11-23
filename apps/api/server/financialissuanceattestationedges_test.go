package server

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
	"github.com/myblueton-sw/hybrid_billing/modules/protectedauthority"
)

func TestFinancialSetupAttestationRequiresOwnedStepCapability(t *testing.T) {
	f, reader, c := confirmedSetupConfiguration(t)
	claim := registrationStatement("inspect-only", "future-channel")
	f.code(t, "inspection is not a step capability", attestRun(f, reader, "attest_channel_registration", c, claim), 403, string(commands.PermissionDenied))
	f.code(t, "inspection does not withdraw", attestRun(f, reader, "withdraw_channel_registration", c, map[string]string{"attestation": "inspect-only"}), 403, string(commands.PermissionDenied))
	for _, step := range setupAttestationSteps {
		terms, err := protectedAuthorityTerms(SampleWorkspace, step.Capability)
		f.must(err)
		validity := gate.Validity{NotBefore: f.s.clock(), NotAfter: f.s.clock().Add(time.Hour)}
		if f.s.admin.AddDelegation(gate.Delegation{ID: "delegated-" + step.Purpose, DelegatorID: SampleAdmin, DelegateID: SampleMember, Terms: terms, Validity: validity}) == nil {
			t.Fatalf("%s capability could be delegated", step.Purpose)
		}
	}
	if len(f.s.financialIssuance.attestations) != 0 {
		t.Fatal("refused caller retained a claim")
	}

	w := configuredSetup(t)
	g, k := w.f, w.configuration
	holders := map[string]actor{"attest_change_instruction": w.requester, "attest_change_verification": w.verifier}
	for command := range holders {
		for other, who := range map[string]actor{"requester": w.requester, "verifier": w.verifier, "approver": w.approver} {
			if who == holders[command] {
				continue
			}
			fields := map[string]string{"attestation": "matrix-" + other, "target": "matrix-target", "statement": "Capability matrix"}
			if command == "attest_change_verification" {
				fields["target_digest"] = strings.Repeat("a", 64)
			}
			g.code(t, command+" by "+other, attestRun(g, who, command, k, fields), 403, string(commands.PermissionDenied))
		}
	}
	for _, command := range []string{"attest_channel_registration", "attest_channel_admission"} {
		fields := map[string]string{"attestation": "matrix-channel", "target": "matrix-target", "statement": "Capability matrix"}
		if command == "attest_channel_admission" {
			fields["target_digest"] = strings.Repeat("a", 64)
		}
		for _, who := range []actor{w.requester, w.verifier, w.approver} {
			g.code(t, command, attestRun(g, who, command, k, fields), 403, string(commands.PermissionDenied))
		}
	}
	if len(g.s.financialIssuance.attestations) != 0 {
		t.Fatal("another capability retained a claim")
	}
}

func TestFinancialSetupAttestationCurrentAuthorityAndConfiguration(t *testing.T) {
	for _, mode := range []string{"ceiling", "grant", "session", "non_sample", "foreign_contract", "unconfirmed"} {
		t.Run(mode, func(t *testing.T) {
			f, registrar, _, spec, c := attestationWorld(t)
			f.ok(t, "control", attestRun(f, registrar, "attest_channel_registration", c, registrationStatement("control", spec.ID)), "recorded", "")
			status, code, target := 403, string(commands.PermissionDenied), c
			switch mode {
			case "ceiling", "grant":
				revoked := 0
				for _, grant := range f.s.admin.Grants() {
					_, ceiling := f.s.protectedAuthority.ceilings[grant.ID]
					if grant.Terms.Action == gate.RegisterIssuanceChannel && ceiling == (mode == "ceiling") {
						f.must(f.s.admin.RevokeGrant(grant.ID, grant.Version))
						revoked++
					}
				}
				if revoked != 1 {
					t.Fatalf("revoked %d grants", revoked)
				}
			case "session":
				f.s.endProbe(registrar.verifier(f))
				status, code = 401, string(commands.Unauthenticated)
			case "non_sample":
				f.s.cfg.Seed = SeedNone
				status, code = 501, string(commands.UnsupportedCapability)
			case "foreign_contract":
				target.Spec.Contract = "sample-contract-02"
			case "unconfirmed":
				base, err := f.s.orgs.ContractRevision(SampleWorkspace, c.Spec.Contract, c.Spec.ContractRevision)
				f.must(err)
				base.Revision++
				f.must(f.s.orgs.AppendContractRevision(base))
				pending := configurationSpec()
				pending.ID, pending.ContractRevision = "pending-configuration", base.Revision
				target, err = proposeConfigurationAs(f, f.as(AdminLogin), pending)
				f.must(err)
				status, code = 422, string(commands.PrerequisiteFailed)
			}
			f.code(t, "attest", attestRun(f, registrar, "attest_channel_registration", target, registrationStatement("refused", spec.ID)), status, code)
			f.code(t, "withdraw", attestRun(f, registrar, "withdraw_channel_registration", target, map[string]string{"attestation": "control"}), status, code)
			p := f.s.financialIssuance.attestations["control"]
			if len(f.s.financialIssuance.attestations) != 1 || !p.WithdrawnAt.IsZero() {
				t.Fatal("refused request changed retained claims")
			}
		})
	}
}

func TestFinancialSetupAttestationSerializedBoundaries(t *testing.T) {
	f, registrar, _, spec, c := attestationWorld(t)
	boundary := func(unit, id string) int {
		low, high := 0, 2000/len(unit)
		for low < high {
			n := (low + high + 1) / 2
			if attestationAdmitted(financialAttestation{ID: id, Purpose: "channel_registration", Target: spec.ID, Statement: strings.Repeat(unit, n)}) {
				low = n
			} else {
				high = n - 1
			}
		}
		return low
	}
	grantSetupInspectionForTest(f, true)
	reader := f.as(MemberLogin)
	for name, unit := range map[string]string{"row": "<", "envelope": `"`} {
		n := boundary(unit, "bound-"+name)
		if n == 0 || n >= 2000/len(unit) {
			t.Fatalf("%s boundary %d is not inside the raw limit", name, n)
		}
		over := map[string]string{"attestation": "bound-" + name, "target": spec.ID, "statement": strings.Repeat(unit, n+1)}
		f.code(t, name+" one past the bound", attestRun(f, registrar, "attest_channel_registration", c, over), 422, string(commands.InvalidRequest))
		at := map[string]string{"attestation": "bound-" + name, "target": spec.ID, "statement": strings.Repeat(unit, n)}
		f.ok(t, name+" at the bound", attestRun(f, registrar, "attest_channel_registration", c, at), "recorded", "")
	}
	for i := 0; i < 8; i++ {
		fields := map[string]string{"attestation": fmt.Sprintf("heavy-%02d", i), "target": spec.ID, "statement": strings.Repeat(`"`, boundary(`"`, fmt.Sprintf("heavy-%02d", i)))}
		f.ok(t, "heavy", attestRun(f, registrar, "attest_channel_registration", c, fields), "recorded", "")
	}
	page := protectedSetupRead(f, reader, "attestations", c, nil)
	if page.status != 200 || field(page, "count") != "10" || field(page, "row_09") == "" {
		t.Fatal("ten worst-case rows do not fit one page", page.status)
	}
	for i := 0; i < 10; i++ {
		if len(field(page, fmt.Sprintf("row_%02d", i))) > 4096 {
			t.Fatal("projected row exceeds the admitted bound")
		}
	}
}

func TestFinancialSetupAttestationOwnershipTotalAndWithdrawalFault(t *testing.T) {
	f, registrar, admitter, spec, c := attestationWorld(t)
	a := f.s.financialIssuance
	claim := registrationStatement("owned-claim", spec.ID)
	f.ok(t, "attest", attestRun(f, registrar, "attest_channel_registration", c, claim), "recorded", "")
	grantChannelCapabilityAt(f, "second-", protectedauthority.RegisterChannel, "configured-channel-admitter")
	second := attestRun(f, admitter, "attest_channel_registration", c, claim)
	f.code(t, "same identifier by another actor", second, 409, string(commands.EvidenceStale))
	if strings.Contains(second.raw, SampleMember) || strings.Contains(second.raw, "onboarding") {
		t.Fatal("collision disclosed another actor's claim")
	}
	f.code(t, "another actor withdraws", attestRun(f, admitter, "withdraw_channel_registration", c, map[string]string{"attestation": "owned-claim"}), 403, string(commands.PermissionDenied))
	grantChannelCapabilityAt(f, "purpose-", protectedauthority.AdmitChannel, SampleMember)
	f.code(t, "another purpose withdraws", attestRun(f, registrar, "withdraw_channel_admission", c, map[string]string{"attestation": "owned-claim"}), 403, string(commands.PermissionDenied))
	if !a.attestations["owned-claim"].WithdrawnAt.IsZero() {
		t.Fatal("refused withdrawal changed the claim")
	}

	revision := f.s.Revision()
	f.s.fault = func(at string) bool { return at == "issuance-attestation-after-withdrawal" }
	body := attestBody(f, "uncertain-withdrawal", c, map[string]string{"attestation": "owned-claim"})
	uncertain := f.call("issuance_setup.withdraw_channel_registration", body, registrar.cookie, registrar.csrf)
	f.s.fault = nil
	if uncertain.status != 500 || uncertain.body["operation_id"] == nil || a.attestations["owned-claim"].WithdrawnAt.IsZero() || f.s.Revision() == revision {
		t.Fatal("uncertain withdrawal", uncertain.raw)
	}
	replay := f.call("issuance_setup.withdraw_channel_registration", body, registrar.cookie, registrar.csrf)
	if replay.status != 500 || replay.body["operation_id"] != uncertain.body["operation_id"] {
		t.Fatal("same key did not replay the stored outcome", replay.raw)
	}
	f.ok(t, "new identical withdrawal", attestRun(f, registrar, "withdraw_channel_registration", c, map[string]string{"attestation": "owned-claim"}), "withdrawn", "")
	retried := attestRun(f, registrar, "attest_channel_registration", c, claim)
	f.ok(t, "identical attest after withdrawal", retried, "recorded", "")
	if field(retried, "state") != "withdrawn" {
		t.Fatal("retry hid the withdrawal", retried.raw)
	}

	for i := len(a.attestations); i < financialAttestationLimit; i++ {
		id := fmt.Sprintf("filled-%04d", i)
		a.attestations[id] = financialAttestation{ID: id, Workspace: SampleWorkspace, Purpose: "channel_registration", Target: "filled", Actor: fmt.Sprintf("other-actor-%04d", i)}
	}
	f.code(t, "total bound", attestRun(f, registrar, "attest_channel_registration", c, registrationStatement("over-total", spec.ID)), 422, string(commands.PrerequisiteFailed))
	if len(a.attestations) != financialAttestationLimit {
		t.Fatal("total bound exceeded")
	}
}

func TestFinancialSetupAttestationWithdrawalByEveryPurpose(t *testing.T) {
	f, registrar, admitter, spec, c := attestationWorld(t)
	channel, err := registerChannelAs(f, registrar, spec)
	f.must(err)
	f.ok(t, "admission", attestRun(f, admitter, "attest_channel_admission", c, map[string]string{"attestation": "admission", "target": spec.ID, "target_digest": channel.Digest, "statement": "Compared with the independent record."}), "recorded", "")
	done := attestRun(f, admitter, "withdraw_channel_admission", c, map[string]string{"attestation": "admission"})
	f.ok(t, "withdraw admission", done, "withdrawn", "")
	if field(done, "purpose") != "channel_admission" || field(done, "state") != "withdrawn" {
		t.Fatal(done.raw)
	}

	w := configuredSetup(t)
	g, k := w.f, w.configuration
	g.ok(t, "instruction", attestRun(g, w.requester, "attest_change_instruction", k, map[string]string{"attestation": "instruction", "target": w.change.ID, "statement": "Received in person."}), "recorded", "")
	g.ok(t, "withdraw instruction", attestRun(g, w.requester, "withdraw_change_instruction", k, map[string]string{"attestation": "instruction"}), "withdrawn", "")
	p, err := requestChangeAs(g, w.requester, w.change)
	g.must(err)
	g.ok(t, "verification", attestRun(g, w.verifier, "attest_change_verification", k, map[string]string{"attestation": "verification", "target": w.change.ID, "target_digest": p.Digest, "statement": "Called the registered contact."}), "recorded", "")
	g.ok(t, "withdraw verification", attestRun(g, w.verifier, "withdraw_change_verification", k, map[string]string{"attestation": "verification"}), "withdrawn", "")
	for id, purpose := range map[string]string{"instruction": "change_instruction", "verification": "change_verification"} {
		if got := g.s.financialIssuance.attestations[id]; got.Purpose != purpose || got.WithdrawnAt.IsZero() || got.Consumed != nil {
			t.Fatalf("%s not withdrawn", id)
		}
	}
}

func TestFinancialSetupAttestationPagingAndSharedChannelScope(t *testing.T) {
	f, registrar, _, spec, c := attestationWorld(t)
	a := f.s.financialIssuance
	for i := 1; i <= 12; i++ {
		id := fmt.Sprintf("page-claim-%02d", i)
		f.ok(t, id, attestRun(f, registrar, "attest_channel_registration", c, registrationStatement(id, spec.ID)), "recorded", "")
		if i <= 2 {
			f.ok(t, "withdraw", attestRun(f, registrar, "withdraw_channel_registration", c, map[string]string{"attestation": id}), "withdrawn", "")
		}
	}
	grantSetupInspectionForTest(f, true)
	reader := f.as(MemberLogin)
	first := protectedSetupRead(f, reader, "attestations", c, nil)
	if first.status != 200 || field(first, "count") != "12" || field(first, "next") != "page-claim-10" || field(first, "row_10") != "" {
		t.Fatal(first.raw)
	}
	cursor := map[string]string{"after": field(first, "next"), "snapshot": field(first, "snapshot")}
	last := protectedSetupRead(f, reader, "attestations", c, cursor)
	if protectedSetupRow(t, last, 0)["attestation"] != "page-claim-11" || protectedSetupRow(t, last, 1)["attestation"] != "page-claim-12" || field(last, "next") != "" || field(last, "row_02") != "" {
		t.Fatal(last.raw)
	}
	f.code(t, "cursor without snapshot", protectedSetupRead(f, reader, "attestations", c, map[string]string{"after": "page-claim-10"}), 422, string(commands.InvalidRequest))
	f.code(t, "unknown cursor", protectedSetupRead(f, reader, "attestations", c, map[string]string{"after": "absent-claim", "snapshot": cursor["snapshot"]}), 409, string(commands.EvidenceStale))
	f.ok(t, "withdraw later row", attestRun(f, registrar, "withdraw_channel_registration", c, map[string]string{"attestation": "page-claim-12"}), "withdrawn", "")
	f.code(t, "changed snapshot", protectedSetupRead(f, reader, "attestations", c, cursor), 409, string(commands.EvidenceStale))

	base, err := f.s.orgs.ContractRevision(SampleWorkspace, c.Spec.Contract, c.Spec.ContractRevision)
	f.must(err)
	base.Revision++
	f.must(f.s.orgs.AppendContractRevision(base))
	next := configurationSpec()
	next.ID, next.ContractRevision = "second-configuration", base.Revision
	proposed, err := proposeConfigurationAs(f, f.as(AdminLogin), next)
	f.must(err)
	other, err := confirmConfigurationAs(f, f.as(Admin3Login), proposed)
	f.must(err)
	if other.Scope.BillingVersion != c.Scope.BillingVersion || other.Scope.BillingProfile.State != organizations.Active {
		t.Fatal("fixture configurations do not share the billing profile version")
	}
	entity := spec
	entity.ID, entity.Selector = "shared-entity-channel", financialChannelSelector{Kind: "entity", BillingProfile: c.Spec.BillingProfile}
	channel, err := registerChannelAs(f, registrar, entity)
	f.must(err)
	if protectedSetupChannelKind(c, channel) != "entity" || protectedSetupChannelKind(other, channel) != "entity" {
		t.Fatal("fixture entity channel is not shared by both configurations")
	}
	used := a.attestations["page-claim-03"]
	used.Target = entity.ID
	used.Digest, err = attestationDigest(used)
	f.must(err)
	used.Consumed = &financialAttestationUse{Purpose: used.Purpose, Target: entity.ID, TargetDigest: channel.Digest}
	a.attestations["page-claim-03"] = used
	revision := f.s.Revision()
	shared := protectedSetupRead(f, reader, "attestations", other, nil)
	row := protectedSetupRow(t, shared, 0)
	if field(shared, "count") != "1" || row["attestation"] != "page-claim-03" || row["state"] != "consumed" || row["target"] != entity.ID || used.Context != configurationReference(c) {
		t.Fatal("claim consumed by a shared channel is missing under the second configuration", shared.raw)
	}
	if own := protectedSetupRead(f, reader, "attestations", c, nil); own.status != 200 || field(own, "count") != "12" {
		t.Fatal(own.raw)
	}
	stale := other
	stale.Digest = strings.Repeat("0", 64)
	f.code(t, "stale references digest", protectedSetupRead(f, reader, "references", stale, nil), 409, string(commands.EvidenceStale))
	if got := protectedSetupRead(f, reader, "references", other, nil); got.status != 200 || f.s.Revision() != revision {
		t.Fatal("references read changed the revision", got.raw)
	}
}
