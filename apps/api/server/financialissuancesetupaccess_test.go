package server

import (
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/protectedauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/termsauthority"
)

func TestFinancialSetupInspectionAuthority(t *testing.T) {
	for _, domain := range []string{"term", "protected"} {
		for _, mode := range []string{"hold", "human_revoked", "ceiling_revoked", "binding", "ceiling", "session", "actor", "foreign_workspace", "unknown", "mutation_only"} {
			t.Run(domain+"/"+mode, func(t *testing.T) {
				f := setup(t, SeedSample).ready()
				f.setPassword(MemberLogin, pw)
				root, second, member := f.as(AdminLogin), f.as(Admin2Login), f.as(MemberLogin)
				proposal, decision, spec := termPermissionProposal, termPermissionDecision, termPermissionSpec(f, "inspect")
				authority := f.s.termAuthority
				read := f.s.financialIssuance.termReadCaller
				write := f.s.financialIssuance.termCaller
				other := f.s.financialIssuance.protectedReadCaller
				mutation := termsauthority.RecordTerm
				if domain == "protected" {
					proposal, decision, spec = protectedPermissionProposal, protectedPermissionDecision, protectedPermissionSpec(f, "inspect")
					authority = f.s.protectedAuthority
					read, write, other = f.s.financialIssuance.protectedReadCaller, f.s.financialIssuance.protectedCaller, f.s.financialIssuance.termReadCaller
					mutation = protectedauthority.RegisterChannel
				}
				for _, who := range []actor{root, member} {
					f.must(inputAuthorityAs(f, who, func() error {
						documentFailure(t, read(SampleWorkspace, f.s.answer.User()), commands.PermissionDenied)
						return nil
					}))
				}
				if mode == "mutation_only" {
					spec.Capability = mutation
				}
				p, err := proposal(f, root, inputauthority.Proposal{ID: "read-ceiling", Spec: spec, Rationale: "Explicit setup evidence ceiling"})
				f.must(err)
				ceiling, err := decision(f, second, p)
				f.must(err)
				spec.Kind, spec.Recipient, spec.CeilingID = inputauthority.Human, SampleMember, ceiling.Decision.GrantID
				p, err = proposal(f, root, inputauthority.Proposal{ID: "read-human", Spec: spec, Rationale: "Explicit setup evidence operator"})
				f.must(err)
				if mode == "unknown" {
					faultPoint := "issuance-" + domain + "-authority-after-grant"
					f.s.fault = func(at string) bool { return at == faultPoint }
					_, failed := decision(f, second, p)
					if failed == nil {
						t.Fatal("interrupted grant reported success")
					}
					f.s.fault = nil
					f.must(inputAuthorityAs(f, member, func() error {
						documentFailure(t, read(SampleWorkspace, SampleMember), commands.PermissionDenied)
						return nil
					}))
				}
				human, err := decision(f, second, p)
				f.must(err)
				f.must(inputAuthorityAs(f, member, func() error {
					if mode == "mutation_only" {
						f.must(write(SampleWorkspace, SampleMember, mutation))
						documentFailure(t, read(SampleWorkspace, SampleMember), commands.PermissionDenied)
						return nil
					}
					f.must(read(SampleWorkspace, SampleMember))
					documentFailure(t, other(SampleWorkspace, SampleMember), commands.PermissionDenied)
					documentFailure(t, write(SampleWorkspace, SampleMember, mutation), commands.PermissionDenied)
					expected := commands.PermissionDenied
					workspace, actorID := SampleWorkspace, SampleMember
					switch mode {
					case "unknown":
						return nil
					case "hold":
						f.s.enterHold("Synthetic setup read hold")
						f.must(read(workspace, actorID))
						documentFailure(t, write(workspace, actorID, "inspect"), commands.PrerequisiteFailed)
						return nil
					case "human_revoked", "ceiling_revoked":
						id := human.Decision.GrantID
						if mode == "ceiling_revoked" {
							id = ceiling.Decision.GrantID
						}
						grant, ok := f.s.grantView(id)
						if !ok {
							t.Fatal("missing actual grant")
						}
						f.must(f.s.admin.RevokeGrant(id, grant.Version))
					case "binding":
						delete(authority.bindings, human.Decision.GrantID)
					case "ceiling":
						delete(authority.ceilings, ceiling.Decision.GrantID)
					case "session":
						f.s.endProbe(member.verifier(f))
						expected = commands.Unauthenticated
					case "actor":
						actorID = SampleAdmin
					case "foreign_workspace":
						workspace = "foreign-workspace"
					}
					documentFailure(t, read(workspace, actorID), expected)
					return nil
				}))
			})
		}
	}
}

func TestFinancialSetupEvidenceCannotDowngradeOrUseGenericGrants(t *testing.T) {
	fields := []string{termsauthority.EvidenceField, protectedauthority.EvidenceField}
	catalog := Catalog(nil, map[string]gate.FieldClass{fields[0]: gate.General, fields[1]: gate.General})
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	for _, field := range fields {
		if catalog.Fields[field] != gate.Cost {
			t.Fatal("setup evidence downgraded", field)
		}
		f.code(t, "generic evidence field bypass", f.requestGrant(root, SampleMember, map[string]string{"action": string(gate.ViewCost), "kind": ReaderKind, "fields": field}), 422, "invalid_request")
	}
}
