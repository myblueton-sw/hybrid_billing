package server

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/protectedauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/termsauthority"
)

var issuanceAuthorityNamespaces = []struct {
	name         string
	domain       grantAuthorityDomain
	capabilities []string
}{
	{"issuance_protected_authority", protectedGrantAuthority, []string{protectedauthority.RegisterChannel, protectedauthority.AdmitChannel, protectedauthority.RequestChange, protectedauthority.VerifyChange, protectedauthority.ApproveChange, protectedauthority.Inspect}},
	{"issuance_term_authority", termGrantAuthority, []string{termsauthority.RecordTerm, termsauthority.ApproveTerm, termsauthority.BindTerm, termsauthority.ApproveBinding, termsauthority.Inspect}},
	{"issuance_draft_authority", draftGrantAuthority, []string{draftauthority.Submit, draftauthority.Approve, draftauthority.Inspect}},
}

func issuanceAuthorityTimes(f world, capability string) map[string]string {
	return map[string]string{"capability": capability, "not_before": f.s.clock().Format(time.RFC3339Nano), "not_after": f.s.clock().Add(12 * time.Hour).Format(time.RFC3339Nano)}
}

func issuanceGrantCount(f world, domain grantAuthorityDomain) int {
	n := 0
	for id := range f.s.grantAuthority(domain).bindings {
		if _, found := f.s.grantView(id); found {
			n++
		}
	}
	return n
}

func issuanceAuthorityRead(f world, a actor, namespace, id string) reply {
	body, _ := json.Marshal(map[string]any{"request_id": "authority-read", "target_id": id, "body": map[string]string{}})
	return f.call(namespace+".inspect", string(body), a.cookie, a.csrf)
}

func provisionIssuanceCapability(t *testing.T, f world, namespace, capability, recipient, prefix string) reply {
	t.Helper()
	logins := []string{}
	for _, u := range []struct{ id, login string }{{SampleAdmin, AdminLogin}, {SampleAdmin2, Admin2Login}, {SampleAdmin3, Admin3Login}} {
		if u.id != recipient {
			logins = append(logins, u.login)
		}
	}
	first, second := f.as(logins[0]), f.as(logins[1])
	ceiling := f.run(first, namespace+".propose_console", prefix+"ceiling", issuanceAuthorityTimes(f, capability))
	f.ok(t, "ceiling proposal", ceiling, inputauthority.Pending, "")
	applied := f.run(second, namespace+".approve", prefix+"ceiling", map[string]string{"digest": field(ceiling, "digest")})
	f.ok(t, "ceiling approval", applied, inputauthority.Applied, "")
	human := issuanceAuthorityTimes(f, capability)
	human["recipient"], human["ceiling"] = recipient, field(applied, "grant")
	proposed := f.run(first, namespace+".propose_human", prefix+"human", human)
	f.ok(t, "human proposal", proposed, inputauthority.Pending, "")
	granted := f.run(second, namespace+".approve", prefix+"human", map[string]string{"digest": field(proposed, "digest")})
	f.ok(t, "human approval", granted, inputauthority.Applied, "")
	if field(granted, "receipt") != "historical_decision" || field(granted, "recipient") != recipient || field(granted, "capability") != capability || field(granted, "grant") == "" {
		t.Fatal(granted.raw)
	}
	return granted
}

func TestIssuanceAuthorityPublicProvisioningGrantsEveryCapability(t *testing.T) {
	for _, ns := range issuanceAuthorityNamespaces {
		for _, capability := range ns.capabilities {
			t.Run(ns.name+"/"+capability, func(t *testing.T) {
				f := setup(t, SeedSample).ready()
				f.setPassword(MemberLogin, pw)
				terms, err := ns.domain.terms(SampleWorkspace, capability)
				f.must(err)
				request := gate.Request{Action: terms.Action, Kind: terms.Kind, Fields: terms.Fields, Resource: gate.Resource{WorkspaceID: SampleWorkspace}}
				allowed := func() bool {
					var got bool
					f.must(inputAuthorityAs(f, f.as(MemberLogin), func() error {
						got = f.s.gate.Decide(f.s.answer, ConsoleClient, request).Allowed
						return nil
					}))
					return got
				}
				if allowed() || issuanceGrantCount(f, ns.domain) != 0 {
					t.Fatal("capability held before provisioning")
				}
				granted := provisionIssuanceCapability(t, f, ns.name, capability, SampleMember, "public-")
				if !allowed() || issuanceGrantCount(f, ns.domain) != 2 {
					t.Fatal("public provisioning did not produce an effective grant")
				}
				if got := issuanceAuthorityRead(f, f.as(Admin3Login), ns.name, "public-human"); got.status != 200 || field(got, "state") != inputauthority.Applied || field(got, "digest") != field(granted, "digest") {
					t.Fatal(got.raw)
				}
				for _, other := range issuanceAuthorityNamespaces {
					if other.name != ns.name && issuanceGrantCount(f, other.domain) != 0 {
						t.Fatalf("grant leaked into %s", other.name)
					}
				}
			})
		}
	}
}

func TestIssuanceAuthorityPublicProvisioningEnablesRealCommands(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	p, err := proposeConfigurationAs(f, f.as(AdminLogin), configurationSpec())
	f.must(err)
	c, err := confirmConfigurationAs(f, f.as(Admin3Login), p)
	f.must(err)
	member := f.as(MemberLogin)
	claim := registrationStatement("public-claim", "future-channel")
	f.code(t, "terms read before the grant", termsRead(f, member, c, nil), 403, string(commands.PermissionDenied))
	f.code(t, "protected read before the grant", protectedSetupRead(f, member, "protected", c, nil), 403, string(commands.PermissionDenied))
	f.code(t, "attestation before the grant", attestRun(f, member, "attest_channel_registration", c, claim), 403, string(commands.PermissionDenied))
	f.code(t, "term before the grant", termWrite(f, member, "record_term", c, map[string]string{"term": "public-term", "net_days": "30"}), 403, string(commands.PermissionDenied))
	provisionIssuanceCapability(t, f, "issuance_term_authority", termsauthority.Inspect, SampleMember, "terms-inspect-")
	provisionIssuanceCapability(t, f, "issuance_protected_authority", protectedauthority.Inspect, SampleMember, "protected-inspect-")
	provisionIssuanceCapability(t, f, "issuance_protected_authority", protectedauthority.RegisterChannel, SampleMember, "register-")
	provisionIssuanceCapability(t, f, "issuance_term_authority", termsauthority.RecordTerm, SampleMember, "record-")
	member = f.as(MemberLogin)
	if got := termsRead(f, member, c, nil); got.status != 200 || field(got, "term_state") != "none" {
		t.Fatal(got.raw)
	}
	if got := protectedSetupRead(f, member, "protected", c, nil); got.status != 200 || field(got, "rules_state") != "none" {
		t.Fatal(got.raw)
	}
	f.ok(t, "attestation after the grant", attestRun(f, member, "attest_channel_registration", c, claim), "recorded", "")
	f.ok(t, "term after the grant", termWrite(f, member, "record_term", c, map[string]string{"term": "public-term", "net_days": "30"}), "pending", "")
	f.code(t, "another step still refused", attestRun(f, member, "attest_change_instruction", c, map[string]string{"attestation": "other", "target": "change", "statement": "Not granted"}), 403, string(commands.PermissionDenied))
	f.code(t, "another terms capability still refused", termWrite(f, member, "approve_term", c, map[string]string{"term": "public-term", "term_digest": strings.Repeat("a", 64)}), 403, string(commands.PermissionDenied))
}

func TestIssuanceAuthorityPublicProvisioningProtocolAndAuthority(t *testing.T) {
	for _, ns := range issuanceAuthorityNamespaces {
		t.Run(ns.name, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			f.setPassword(MemberLogin, pw)
			root, second, third, member := f.as(AdminLogin), f.as(Admin2Login), f.as(Admin3Login), f.as(MemberLogin)
			capability := ns.capabilities[0]
			times := issuanceAuthorityTimes(f, capability)
			f.code(t, "non-governor proposes", f.runWith(member, f.s.Revision(), ns.name+".propose_console", "member-ceiling", times), 403, string(commands.PermissionDenied))
			f.code(t, "non-governor inspects", issuanceAuthorityRead(f, member, ns.name, "member-ceiling"), 403, string(commands.PermissionDenied))
			unknown := issuanceAuthorityTimes(f, "administer")
			f.code(t, "unknown capability", f.run(root, ns.name+".propose_console", "unknown-ceiling", unknown), 422, string(commands.InvalidRequest))
			for _, other := range issuanceAuthorityNamespaces {
				foreign := other.capabilities[0]
				known := false
				for _, own := range ns.capabilities {
					known = known || own == foreign
				}
				if !known {
					f.code(t, "capability of "+other.name, f.run(root, ns.name+".propose_console", "foreign-ceiling", issuanceAuthorityTimes(f, foreign)), 422, string(commands.InvalidRequest))
				}
			}
			for _, name := range []string{"recipient", "ceiling", "action", "kind", "workspace"} {
				forged := issuanceAuthorityTimes(f, capability)
				forged[name] = "approve"
				f.code(t, "forged "+name, f.run(root, ns.name+".propose_console", "forged-"+name, forged), 422, string(commands.InvalidRequest))
			}
			saved := f.s.cfg.Commands
			for _, change := range []func(){func() { f.s.cfg.Commands.Fields = 19 }, func() { f.s.cfg.Commands.Value = 1023 }} {
				change()
				f.code(t, "transport budget", f.run(root, ns.name+".propose_console", "budget", times), 501, string(commands.UnsupportedCapability))
				f.s.cfg.Commands = saved
			}
			if len(f.s.grantAuthority(ns.domain).bindings) != 0 {
				t.Fatal("a refusal reserved a proposal")
			}
			ceiling := f.run(root, ns.name+".propose_console", "ceiling", times)
			f.ok(t, "ceiling proposal", ceiling, inputauthority.Pending, "")
			f.code(t, "proposer approves", f.run(root, ns.name+".approve", "ceiling", map[string]string{"digest": field(ceiling, "digest")}), 403, string(commands.PermissionDenied))
			f.code(t, "stale digest", f.run(second, ns.name+".approve", "ceiling", map[string]string{"digest": strings.Repeat("a", 64)}), 422, string(commands.InvalidStage))
			human := issuanceAuthorityTimes(f, capability)
			human["recipient"], human["ceiling"] = SampleMember, "issuance-missing-ceiling"
			f.code(t, "human grant without an approved ceiling", f.run(root, ns.name+".propose_human", "early-human", human), 403, string(commands.PermissionDenied))
			applied := f.run(second, ns.name+".approve", "ceiling", map[string]string{"digest": field(ceiling, "digest")})
			f.ok(t, "ceiling approval", applied, inputauthority.Applied, "")
			own := issuanceAuthorityTimes(f, capability)
			own["recipient"], own["ceiling"] = SampleAdmin, field(applied, "grant")
			f.code(t, "recipient proposes own grant", f.run(root, ns.name+".propose_human", "own-human", own), 403, string(commands.PermissionDenied))
			selfApproved := f.run(second, ns.name+".propose_human", "own-human", own)
			f.ok(t, "grant for a governor", selfApproved, inputauthority.Pending, "")
			f.code(t, "recipient approves own grant", f.run(root, ns.name+".approve", "own-human", map[string]string{"digest": field(selfApproved, "digest")}), 403, string(commands.PermissionDenied))
			f.code(t, "recipient rejects own grant", f.run(root, ns.name+".reject", "own-human", map[string]string{"digest": field(selfApproved, "digest")}), 403, string(commands.PermissionDenied))
			f.ok(t, "withdraw", f.run(second, ns.name+".withdraw", "own-human", map[string]string{"digest": field(selfApproved, "digest")}), inputauthority.Withdrawn, "")
			f.code(t, "approve a withdrawn proposal", f.run(third, ns.name+".approve", "own-human", map[string]string{"digest": field(selfApproved, "digest")}), 422, string(commands.InvalidStage))
			long := issuanceAuthorityTimes(f, capability)
			long["recipient"], long["ceiling"], long["not_after"] = SampleMember, field(applied, "grant"), f.s.clock().Add(13*time.Hour).Format(time.RFC3339Nano)
			f.code(t, "validity beyond the ceiling", f.run(root, ns.name+".propose_human", "long-human", long), 403, string(commands.PermissionDenied))
			mismatched := issuanceAuthorityTimes(f, ns.capabilities[1])
			mismatched["recipient"], mismatched["ceiling"] = SampleMember, field(applied, "grant")
			f.code(t, "ceiling of another capability", f.run(root, ns.name+".propose_human", "mismatched-human", mismatched), 403, string(commands.PermissionDenied))
			for _, other := range issuanceAuthorityNamespaces {
				if other.name != ns.name {
					crossed := issuanceAuthorityTimes(f, other.capabilities[0])
					crossed["recipient"], crossed["ceiling"] = SampleMember, field(applied, "grant")
					f.code(t, "ceiling of "+ns.name+" under "+other.name, f.run(root, other.name+".propose_human", "crossed-human", crossed), 403, string(commands.PermissionDenied))
					if issuanceGrantCount(f, other.domain) != 0 {
						t.Fatalf("a %s ceiling produced a %s grant", ns.name, other.name)
					}
				}
			}
			if issuanceGrantCount(f, ns.domain) != 1 {
				t.Fatal("a refused proposal produced a grant")
			}

			pending := issuanceAuthorityTimes(f, capability)
			pending["recipient"], pending["ceiling"] = SampleMember, field(applied, "grant")
			proposed := f.run(second, ns.name+".propose_human", "held-human", pending)
			f.ok(t, "human proposal", proposed, inputauthority.Pending, "")
			staged := f.stage(root, "", SampleAdmin2)
			f.ok(t, "stage", staged, "draft", "")
			prepared := f.prepare(root, field(staged, "draft"), "suspend_membership")
			f.ok(t, "prepare impact", prepared, "prepared", "")
			counted := false
			for _, raw := range f.bulkRead(root, "/"+field(prepared, "bulk")).body["impact"].(map[string]any)["pending"].([]any) {
				if row := raw.(map[string]any); row["store"] == ns.name+"_proposals" {
					counted = whole(row["count"]) == 1 && row["rechecked"] == true
				}
			}
			if !counted {
				t.Fatal("pending provisioning proposal missing from bulk revocation impact")
			}
			f.s.enterHold("Provisioning held")
			digest := map[string]string{"digest": field(proposed, "digest")}
			f.code(t, "held proposal", f.run(root, ns.name+".propose_console", "held-ceiling", times), 503, string(commands.DependencyUnavailable))
			f.code(t, "held human proposal", f.run(root, ns.name+".propose_human", "held-human-2", pending), 503, string(commands.DependencyUnavailable))
			f.code(t, "held approval", f.run(root, ns.name+".approve", "held-human", digest), 503, string(commands.DependencyUnavailable))
			if got := issuanceAuthorityRead(f, root, ns.name, "held-human"); got.status != 200 || field(got, "digest") != field(proposed, "digest") {
				t.Fatal("inspection refused during hold", got.raw)
			}
			f.ok(t, "held rejection", f.run(root, ns.name+".reject", "held-human", digest), inputauthority.Rejected, "")
		})
	}
}

func TestIssuanceAuthorityPublicProvisioningMatchesPrivateHelper(t *testing.T) {
	public := setup(t, SeedSample).ready()
	private := setup(t, SeedSample).ready()
	times := issuanceAuthorityTimes(public, protectedauthority.RegisterChannel)
	got := public.run(public.as(AdminLogin), "issuance_protected_authority.propose_console", "same-ceiling", times)
	public.ok(t, "public proposal", got, inputauthority.Pending, "")
	spec := protectedPermissionSpec(private, protectedauthority.RegisterChannel)
	spec.NotAfter = private.s.clock().Add(12 * time.Hour)
	helper, err := protectedPermissionProposal(private, private.as(AdminLogin), inputauthority.Proposal{ID: "same-ceiling", Spec: spec, Rationale: "sample reason"})
	private.must(err)
	if field(got, "digest") != helper.Digest {
		t.Fatal("public and helper proposals differ for the same input")
	}
}

func TestIssuanceAuthorityPublicProvisioningUnseeded(t *testing.T) {
	f := setup(t, SeedNone).ready()
	defer f.s.Close()
	login := "first@example.invalid"
	if got := f.auth("redeem", activationBody(f.links()[""], login), f.pre()); got.status != 204 {
		t.Fatal(got.status)
	}
	cookie, csrf, err := f.s.openFor(LoginTriple(login), EmptyWorkspace, false)
	f.must(err)
	root := actor{cookie, csrf}
	before := len(f.s.admin.Grants())
	for _, ns := range issuanceAuthorityNamespaces {
		for _, kind := range []string{"propose_console", "propose_human", "approve", "reject", "withdraw"} {
			fields := map[string]string{"digest": "none"}
			if kind == "propose_console" || kind == "propose_human" {
				fields = issuanceAuthorityTimes(f, ns.capabilities[0])
			}
			if kind == "propose_human" {
				fields["recipient"], fields["ceiling"] = "member", "none"
			}
			f.code(t, ns.name+"."+kind, f.run(root, ns.name+"."+kind, "unseeded", fields), 501, string(commands.UnsupportedCapability))
		}
		f.code(t, ns.name+".inspect", issuanceAuthorityRead(f, root, ns.name, "unseeded"), 501, string(commands.UnsupportedCapability))
	}
	if len(f.s.admin.Grants()) != before {
		t.Fatal("an unseeded installation gained a grant")
	}
}

func TestIssuanceAuthorityPublicProvisioningUncertainWriteAndImpactFault(t *testing.T) {
	for _, ns := range issuanceAuthorityNamespaces {
		t.Run(ns.name, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			root, second, third := f.as(AdminLogin), f.as(Admin2Login), f.as(Admin3Login)
			ceiling := f.run(second, ns.name+".propose_console", "uncertain", issuanceAuthorityTimes(f, ns.capabilities[0]))
			f.ok(t, "proposal", ceiling, inputauthority.Pending, "")
			digest := map[string]string{"digest": field(ceiling, "digest")}
			staged := f.stage(root, "", SampleAdmin2)
			f.s.fault = func(at string) bool { return at == ns.domain.faultPrefix()+"impact" }
			f.code(t, "impact fails closed", f.prepare(root, field(staged, "draft"), "suspend_membership"), 503, string(commands.DependencyUnavailable))
			f.s.fault = func(at string) bool { return at == ns.domain.faultPrefix()+"after-grant" }
			lost := f.run(root, ns.name+".approve", "uncertain", digest)
			f.s.fault = nil
			if lost.status == 200 {
				t.Fatal("a lost grant write reported success", lost.raw)
			}
			if got := issuanceAuthorityRead(f, third, ns.name, "uncertain"); field(got, "state") != inputauthority.Unknown {
				t.Fatal("uncertain write not retained as unknown", got.raw)
			}
			if other := f.run(third, ns.name+".approve", "uncertain", digest); other.status == 200 {
				t.Fatal("another governor recovered the decision", other.raw)
			}
			f.ok(t, "recovery by the same governor", f.run(root, ns.name+".approve", "uncertain", digest), inputauthority.Applied, "")
			if issuanceGrantCount(f, ns.domain) != 1 {
				t.Fatal("recovery did not leave exactly one grant")
			}
		})
	}
}
