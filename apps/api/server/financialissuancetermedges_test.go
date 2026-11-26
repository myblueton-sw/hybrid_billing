package server

import (
	"errors"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/receivables"
	"github.com/myblueton-sw/hybrid_billing/modules/termsauthority"
)

func TestFinancialIssuanceBindingMalformedAndForeign(t *testing.T) {
	f, author, _, p := approvedTermWorld(t)
	for _, mutate := range []func(*financialTermBindingSpec){
		func(s *financialTermBindingSpec) { s.From = "1999-12-31" }, func(s *financialTermBindingSpec) { s.To = "2100-01-01" }, func(s *financialTermBindingSpec) { s.From = "2026-02-30" }, func(s *financialTermBindingSpec) { s.To = "2025-12-31" }, func(s *financialTermBindingSpec) { s.Zone = "UTC" }, func(s *financialTermBindingSpec) { s.Reason = "default" },
	} {
		spec := bindingSpec(p)
		mutate(&spec)
		_, err := bindTermAs(f, author, spec)
		documentFailure(t, err, commands.InvalidRequest)
	}
	if len(f.s.financialIssuance.bindingIntents) != 0 {
		t.Fatal("malformed binding reserved")
	}
	spec := bindingSpec(p)
	id, err := financialEvidenceDigest("issuance_binding", spec.ID)
	f.must(err)
	_, err = f.s.financialIssuance.terms.Bind(SampleWorkspace, receivables.Binding{ID: "issued-binding-" + id, Contract: p.Profile.Source.Basis.Contract.ContractID, Term: p.Term.ID, Revision: p.Term.Revision, Zone: spec.Zone, From: spec.From, To: spec.To, Reason: spec.Reason}, SampleMember)
	f.must(err)
	_, err = bindTermAs(f, author, spec)
	documentFailure(t, err, commands.EvidenceStale)
	if len(f.s.financialIssuance.bindingIntents) != 0 {
		t.Fatal("foreign binding adopted")
	}
}

func TestFinancialIssuanceTermsRecoveryRechecksSourceAndRights(t *testing.T) {
	for _, operation := range []string{"term-register", "term-approve", "binding-register", "binding-approve"} {
		for _, change := range []string{"right", "source"} {
			t.Run(operation+"/"+change, func(t *testing.T) {
				f, author, decider, spec := termsWorld(t)
				a := f.s.financialIssuance
				var p financialTermRecord
				var b financialTermBinding
				var err error
				if operation != "term-register" {
					p, err = recordTermAs(f, author, spec)
					f.must(err)
				}
				if operation == "binding-register" || operation == "binding-approve" {
					p, err = approveTermAs(f, decider, p)
					f.must(err)
				}
				bs := bindingSpec(p)
				if operation == "binding-approve" {
					b, err = bindTermAs(f, author, bs)
					f.must(err)
				}
				point := map[string]string{"term-register": "issuance-term-after-register", "term-approve": "issuance-term-after-approve", "binding-register": "issuance-binding-after-register", "binding-approve": "issuance-binding-after-approve"}[operation]
				run := func() error {
					switch operation {
					case "term-register":
						_, err = recordTermAs(f, author, spec)
					case "term-approve":
						_, err = approveTermAs(f, decider, p)
					case "binding-register":
						_, err = bindTermAs(f, author, bs)
					case "binding-approve":
						_, err = approveBindingAs(f, decider, b)
					}
					return err
				}
				f.s.fault = func(at string) bool { return at == point }
				documentFailure(t, run(), commands.DependencyUnavailable)
				f.s.fault = nil
				savedTerm := copyFinancialTerm(a.termIntents[spec.ID])
				savedBinding := copyFinancialBinding(a.bindingIntents[bs.ID])
				expected := commands.EvidenceStale
				if change == "source" {
					key := [2]string{SampleWorkspace, "public-result"}
					app := f.s.documentERP.resultApplications[key]
					app.receipt = "changed"
					f.s.documentERP.resultApplications[key] = app
				} else {
					expected = commands.PermissionDenied
					action := map[string]gate.Action{"term-register": gate.RecordIssuanceTerm, "term-approve": gate.ApproveIssuanceTerm, "binding-register": gate.BindIssuanceTerm, "binding-approve": gate.ApproveIssuanceTermBinding}[operation]
					target := SampleMember
					if operation == "term-approve" || operation == "binding-approve" {
						target = SampleAdmin3
					}
					removed := 0
					for _, g := range f.s.admin.Grants() {
						if g.SubjectID == target && g.Terms.Action == action {
							f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
							removed++
						}
					}
					if removed == 0 {
						t.Fatal("no actual right revoked")
					}
				}
				documentFailure(t, run(), expected)
				if !reflect.DeepEqual(savedTerm, a.termIntents[spec.ID]) || !reflect.DeepEqual(savedBinding, a.bindingIntents[bs.ID]) || len(a.termScopes) != 1 {
					t.Fatal("currentness refusal released or rewrote ownership")
				}
			})
		}
	}
}

func TestFinancialIssuanceTermsSelfApprovalAndPending(t *testing.T) {
	f, author, _, spec := termsWorld(t)
	grantAdditionalTermCapability(t, f, termsauthority.ApproveTerm, SampleMember)
	grantAdditionalTermCapability(t, f, termsauthority.ApproveBinding, SampleMember)
	p, err := recordTermAs(f, author, spec)
	f.must(err)
	_, err = approveTermAs(f, author, p)
	documentFailure(t, err, commands.SelfApprovalDenied)
	_, err = bindTermAs(f, author, bindingSpec(p))
	documentFailure(t, err, commands.PrerequisiteFailed)
	p, err = approveTermAs(f, f.as(Admin3Login), p)
	f.must(err)
	b, err := bindTermAs(f, author, bindingSpec(p))
	f.must(err)
	_, err = approveBindingAs(f, author, b)
	documentFailure(t, err, commands.SelfApprovalDenied)
	_, err = coverageAs(f, f.as(Admin3Login), b, f.s.clock())
	documentFailure(t, err, commands.EvidenceStale)
}

func TestFinancialIssuanceBindingRefusesChangedModuleHistory(t *testing.T) {
	for _, change := range []string{"pending_end", "ended", "withdrawn"} {
		t.Run(change, func(t *testing.T) {
			f, author, decider, p := approvedTermWorld(t)
			b, err := bindTermAs(f, author, bindingSpec(p))
			f.must(err)
			if change == "withdrawn" {
				_, err = f.s.financialIssuance.terms.WithdrawBinding(SampleWorkspace, b.Binding.ID, SampleMember)
				f.must(err)
				_, err = approveBindingAs(f, decider, b)
				documentFailure(t, err, commands.EvidenceStale)
				return
			}
			b, err = approveBindingAs(f, decider, b)
			f.must(err)
			_, err = f.s.financialIssuance.terms.EndBinding(SampleWorkspace, b.Binding.ID, "2026-11-30", SampleMember)
			f.must(err)
			if change == "ended" {
				_, err = f.s.financialIssuance.terms.ApproveEnd(SampleWorkspace, b.Binding.ID, SampleAdmin3)
				f.must(err)
			}
			_, err = coverageAs(f, decider, b, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC))
			documentFailure(t, err, commands.EvidenceStale)
		})
	}
}

func grantAdditionalTermCapability(t *testing.T, f world, capability, recipient string) {
	t.Helper()
	for id, ceiling := range f.s.termAuthority.ceilings {
		if ceiling.Capability != capability {
			continue
		}
		spec := ceiling
		spec.Kind = inputauthority.Human
		spec.Recipient = recipient
		spec.CeilingID = id
		p, err := termPermissionProposal(f, f.as(AdminLogin), inputauthority.Proposal{ID: capability + "-extra-" + recipient, Spec: spec, Rationale: "Independent additional term operator"})
		f.must(err)
		_, err = termPermissionDecision(f, f.as(Admin2Login), p)
		f.must(err)
		return
	}
	t.Fatal("missing actual governed ceiling")
}

func TestFinancialIssuanceBindingActualCapacity(t *testing.T) {
	f, author, _, p := approvedTermWorld(t)
	day := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < financialIssuanceLimit; i++ {
		spec := bindingSpec(p)
		spec.ID = "bounded-" + strconv.Itoa(i)
		spec.From = day.AddDate(0, 0, i).Format("2006-01-02")
		spec.To = spec.From
		_, err := bindTermAs(f, author, spec)
		f.must(err)
	}
	spec := bindingSpec(p)
	spec.ID = "over-capacity"
	spec.From = "2026-12-31"
	spec.To = spec.From
	_, err := bindTermAs(f, author, spec)
	documentFailure(t, err, commands.PrerequisiteFailed)
	id, err := financialEvidenceDigest("issuance_binding", spec.ID)
	f.must(err)
	_, err = f.s.financialIssuance.terms.BindingOf(SampleWorkspace, "issued-binding-"+id)
	if !errors.Is(err, receivables.ErrNotFound) || len(f.s.financialIssuance.bindingIntents) != financialIssuanceLimit {
		t.Fatal("capacity refusal wrote a binding")
	}
}

func TestFinancialIssuanceTermsUnknownOwnerCannotBeReplaced(t *testing.T) {
	for _, binding := range []bool{false, true} {
		t.Run(map[bool]string{false: "term", true: "binding"}[binding], func(t *testing.T) {
			f, author, decider, spec := termsWorld(t)
			p, err := recordTermAs(f, author, spec)
			f.must(err)
			var b financialTermBinding
			capability := termsauthority.ApproveTerm
			point := "issuance-term-after-approve"
			if binding {
				p, err = approveTermAs(f, decider, p)
				f.must(err)
				b, err = bindTermAs(f, author, bindingSpec(p))
				f.must(err)
				capability = termsauthority.ApproveBinding
				point = "issuance-binding-after-approve"
			}
			for id, ceiling := range f.s.termAuthority.ceilings {
				if ceiling.Capability == capability {
					grant := ceiling
					grant.Kind = inputauthority.Human
					grant.Recipient = SampleAdmin2
					grant.CeilingID = id
					proposed, err := termPermissionProposal(f, f.as(AdminLogin), inputauthority.Proposal{ID: "other-authorized-approver", Spec: grant, Rationale: "Independent alternate decision right"})
					f.must(err)
					_, err = termPermissionDecision(f, decider, proposed)
					f.must(err)
				}
			}
			f.s.fault = func(at string) bool { return at == point }
			if binding {
				_, err = approveBindingAs(f, decider, b)
			} else {
				_, err = approveTermAs(f, decider, p)
			}
			documentFailure(t, err, commands.DependencyUnavailable)
			f.s.fault = nil
			savedTerm := copyFinancialTerm(f.s.financialIssuance.termIntents[p.Spec.ID])
			savedBinding := copyFinancialBinding(f.s.financialIssuance.bindingIntents[b.Spec.ID])
			other := f.as(Admin2Login)
			f.must(inputAuthorityAs(f, other, func() error { return f.s.financialIssuance.termCaller(SampleWorkspace, SampleAdmin2, capability) }))
			if binding {
				_, err = approveBindingAs(f, other, b)
			} else {
				_, err = approveTermAs(f, other, p)
			}
			documentFailure(t, err, commands.PermissionDenied)
			if !reflect.DeepEqual(savedTerm, f.s.financialIssuance.termIntents[p.Spec.ID]) || !reflect.DeepEqual(savedBinding, f.s.financialIssuance.bindingIntents[b.Spec.ID]) {
				t.Fatal("alternate approver changed pinned intent")
			}
			if binding {
				actual, readErr := f.s.financialIssuance.terms.BindingOf(SampleWorkspace, savedBinding.Binding.ID)
				f.must(readErr)
				if !reflect.DeepEqual(actual, savedBinding.Module) {
					t.Fatal("alternate approver changed actual binding")
				}
			} else {
				actual, readErr := f.s.financialIssuance.terms.Term(SampleWorkspace, savedTerm.Term.ID, savedTerm.Term.Revision)
				f.must(readErr)
				if actual != savedTerm.Module {
					t.Fatal("alternate approver changed actual term")
				}
			}
			if binding {
				_, err = approveBindingAs(f, decider, b)
			} else {
				_, err = approveTermAs(f, decider, p)
			}
			f.must(err)
		})
	}
}
