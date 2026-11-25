package server

import (
	"reflect"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/protectedauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/termsauthority"
)

func TestFinancialConfiguredSetupInterruptedWrites(t *testing.T) {
	for _, phase := range []string{"request", "verify", "decide", "term-record", "term-approve", "binding-record", "binding-approve"} {
		t.Run(phase, func(t *testing.T) {
			w := configuredSetup(t)
			f := w.f
			a := f.s.financialIssuance
			otherLogin := activateProtectedMember(f, "setup-other")
			var change financialChangeRecord
			var term financialTermRecord
			var binding financialTermBinding
			var err error
			var point string
			var original actor
			var call func(actor) error
			switch phase {
			case "request", "verify", "decide":
				capability := protectedauthority.RequestChange
				original = w.requester
				point = "issuance-change-after-request"
				call = func(who actor) error { var e error; change, e = requestChangeAs(f, who, w.change); return e }
				if phase != "request" {
					change, err = requestChangeAs(f, w.requester, w.change)
					f.must(err)
					capability = protectedauthority.VerifyChange
					original = w.verifier
					point = "issuance-change-after-verify"
					call = func(who actor) error {
						var e error
						change, e = verifyChangeAs(f, who, a.changes[w.change.ID])
						return e
					}
				}
				if phase == "decide" {
					change, err = verifyChangeAs(f, w.verifier, change)
					f.must(err)
					capability = protectedauthority.ApproveChange
					original = w.approver
					point = "issuance-change-after-decide"
					call = func(who actor) error {
						var e error
						change, e = decideChangeAs(f, who, a.changes[w.change.ID])
						return e
					}
				}
				grantChannelCapabilityAt(f, "other-", capability, "setup-other")
				f.must(inputAuthorityAs(f, f.as(otherLogin), func() error { return a.protectedCaller(SampleWorkspace, "setup-other", capability) }))
			default:
				capability := termsauthority.RecordTerm
				original = w.requester
				point = "issuance-term-after-register"
				call = func(who actor) error { var e error; term, e = recordTermAs(f, who, w.term); return e }
				if phase != "term-record" {
					term, err = recordTermAs(f, w.requester, w.term)
					f.must(err)
					capability = termsauthority.ApproveTerm
					original = w.approver
					point = "issuance-term-after-approve"
					call = func(who actor) error {
						var e error
						term, e = approveTermAs(f, who, a.termIntents[w.term.ID])
						return e
					}
				}
				if phase == "binding-record" || phase == "binding-approve" {
					term, err = approveTermAs(f, w.approver, term)
					f.must(err)
					spec := bindingSpec(term)
					spec.Zone = w.configuration.Spec.Zone
					capability = termsauthority.BindTerm
					original = w.requester
					point = "issuance-binding-after-register"
					call = func(who actor) error { var e error; binding, e = bindTermAs(f, who, spec); return e }
					if phase == "binding-approve" {
						binding, err = bindTermAs(f, w.requester, spec)
						f.must(err)
						capability = termsauthority.ApproveBinding
						original = w.approver
						point = "issuance-binding-after-approve"
						call = func(who actor) error {
							var e error
							binding, e = approveBindingAs(f, who, a.bindingIntents[spec.ID])
							return e
						}
					}
				}
				grantConfiguredTermCapability(f, "other-", capability, "setup-other")
				f.must(inputAuthorityAs(f, f.as(otherLogin), func() error { return a.termCaller(SampleWorkspace, "setup-other", capability) }))
			}
			f.s.fault = func(at string) bool { return at == point }
			documentFailure(t, call(original), commands.DependencyUnavailable)
			f.s.fault = nil
			changeBefore := copyChange(a.changes[w.change.ID])
			termBefore := copyFinancialTerm(a.termIntents[w.term.ID])
			bindingBefore := copyFinancialBinding(a.bindingIntents["binding-1"])
			expected := commands.PermissionDenied
			if phase == "request" || phase == "term-record" || phase == "binding-record" {
				expected = commands.EvidenceStale
			}
			documentFailure(t, call(f.as(otherLogin)), expected)
			if !reflect.DeepEqual(changeBefore, a.changes[w.change.ID]) || !reflect.DeepEqual(termBefore, a.termIntents[w.term.ID]) || !reflect.DeepEqual(bindingBefore, a.bindingIntents["binding-1"]) {
				t.Fatal("different authorized actor changed retained uncertainty")
			}
			f.must(call(original))
			if phase == "request" || phase == "verify" || phase == "decide" {
				if change.Phase != "recorded" || !reflect.DeepEqual(change.Configuration, &w.configuration) || !reflect.DeepEqual(change.Module, changeBefore.Module) {
					t.Fatal("configured change did not reconcile exact module")
				}
			} else if phase == "term-record" || phase == "term-approve" {
				if term.State == "registration_unknown" || term.State == "approval_unknown" || term.Module != termBefore.Module || !reflect.DeepEqual(term.Configuration, &w.configuration) {
					t.Fatal("configured term did not reconcile exact module")
				}
			} else if binding.State == "registration_unknown" || binding.State == "approval_unknown" || !reflect.DeepEqual(binding.Module, bindingBefore.Module) || !reflect.DeepEqual(binding.Term.Configuration, &w.configuration) {
				t.Fatal("configured binding did not reconcile exact module")
			}
		})
	}
}

func grantConfiguredTermCapability(f world, prefix, capability, recipient string) {
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	spec := termPermissionSpec(f, capability)
	p, err := termPermissionProposal(f, root, inputauthority.Proposal{ID: prefix + capability + "-ceiling", Spec: spec, Rationale: "Explicit configured term ceiling"})
	f.must(err)
	ceiling, err := termPermissionDecision(f, second, p)
	f.must(err)
	spec.Kind, spec.Recipient, spec.CeilingID = inputauthority.Human, recipient, ceiling.Decision.GrantID
	p, err = termPermissionProposal(f, root, inputauthority.Proposal{ID: prefix + capability + "-human", Spec: spec, Rationale: "Explicit configured term operator"})
	f.must(err)
	_, err = termPermissionDecision(f, second, p)
	f.must(err)
	f.s.endProbe(root.verifier(f))
	f.s.endProbe(second.verifier(f))
}
