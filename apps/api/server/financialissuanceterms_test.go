package server

import (
	"reflect"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/receivables"
	"github.com/myblueton-sw/hybrid_billing/modules/termsauthority"
)

func termsWorld(t *testing.T) (world, actor, actor, financialTermSpec) {
	t.Helper()
	f, second, first, digest := issuanceSourceWorld(t)
	spec := issuanceSpec()
	spec.Zone = "Asia/Tokyo"
	profile, err := proposeIssuanceProfile(f, first, digest, spec)
	f.must(err)
	profile, err = confirmIssuanceProfile(f, second, profile)
	f.must(err)
	for _, right := range []struct{ capability, recipient string }{{termsauthority.RecordTerm, SampleMember}, {termsauthority.ApproveTerm, SampleAdmin3}, {termsauthority.BindTerm, SampleMember}, {termsauthority.ApproveBinding, SampleAdmin3}} {
		grantTermCapability(t, f, right.capability, right.recipient)
	}
	return f, f.as(MemberLogin), f.as(Admin3Login), financialTermSpec{ID: "terms-1", Profile: profile.Spec.ID, ProfileDigest: profile.Digest, Mode: receivables.ModeCalendar, Revision: 1, NetDays: 30, Reason: "Explicit synthetic calendar terms"}
}
func recordTermAs(f world, who actor, spec financialTermSpec) (financialTermRecord, error) {
	var p financialTermRecord
	err := inputAuthorityAs(f, who, func() error {
		var err error
		p, err = f.s.financialIssuance.recordTerm(SampleWorkspace, f.s.answer.User(), spec)
		return err
	})
	return p, err
}
func approveTermAs(f world, who actor, p financialTermRecord) (financialTermRecord, error) {
	var out financialTermRecord
	err := inputAuthorityAs(f, who, func() error {
		var err error
		out, err = f.s.financialIssuance.approveTerm(SampleWorkspace, p.Spec.ID, p.Digest, f.s.answer.User())
		return err
	})
	return out, err
}
func bindTermAs(f world, who actor, spec financialTermBindingSpec) (financialTermBinding, error) {
	var out financialTermBinding
	err := inputAuthorityAs(f, who, func() error {
		var err error
		out, err = f.s.financialIssuance.bindTerm(SampleWorkspace, f.s.answer.User(), spec)
		return err
	})
	return out, err
}
func approveBindingAs(f world, who actor, p financialTermBinding) (financialTermBinding, error) {
	var out financialTermBinding
	err := inputAuthorityAs(f, who, func() error {
		var err error
		out, err = f.s.financialIssuance.approveTermBinding(SampleWorkspace, p.Spec.ID, p.Digest, f.s.answer.User())
		return err
	})
	return out, err
}
func coverageAs(f world, who actor, p financialTermBinding, at time.Time) (receivables.Coverage, error) {
	var out receivables.Coverage
	err := inputAuthorityAs(f, who, func() error {
		var err error
		out, err = f.s.financialIssuance.termCoverage(SampleWorkspace, p.Spec.ID, p.Digest, f.s.answer.User(), at)
		return err
	})
	return out, err
}
func bindingSpec(p financialTermRecord) financialTermBindingSpec {
	return financialTermBindingSpec{ID: "binding-1", Term: p.Spec.ID, TermDigest: p.Digest, From: "2026-01-01", To: "2026-12-31", Zone: "Asia/Tokyo", Reason: "initial"}
}
func approvedTermWorld(t *testing.T) (world, actor, actor, financialTermRecord) {
	t.Helper()
	f, author, decider, spec := termsWorld(t)
	p, err := recordTermAs(f, author, spec)
	f.must(err)
	p, err = approveTermAs(f, decider, p)
	f.must(err)
	return f, author, decider, p
}
func TestFinancialIssuanceTermsActualComposition(t *testing.T) {
	f, author, decider, spec := termsWorld(t)
	before, err := f.s.approvals.RequestOf(SampleWorkspace, "issuance-final")
	f.must(err)
	p, err := recordTermAs(f, author, spec)
	f.must(err)
	if p.State != "pending" || p.Module.Term.NetDays != 30 || p.Module.Author != SampleMember {
		t.Fatal("missing real recorded term")
	}
	_, err = approveTermAs(f, author, p)
	documentFailure(t, err, commands.PermissionDenied)
	p, err = approveTermAs(f, decider, p)
	f.must(err)
	if p.State != "approved" || p.Module.Approver != SampleAdmin3 {
		t.Fatal("missing real independent term decision")
	}
	replay, err := recordTermAs(f, author, spec)
	f.must(err)
	if !reflect.DeepEqual(p, replay) {
		t.Fatal("term replay changed")
	}
	b, err := bindTermAs(f, author, bindingSpec(p))
	f.must(err)
	if b.Binding.Contract != p.Profile.Source.Basis.Contract.ContractID || b.Binding.Term != p.Term.ID || b.State != "pending" {
		t.Fatal("binding lost source contract")
	}
	b, err = approveBindingAs(f, decider, b)
	f.must(err)
	at := time.Date(2026, 1, 31, 16, 0, 0, 0, time.UTC)
	c, err := coverageAs(f, decider, b, at)
	f.must(err)
	if c.Date != "2026-02-01" || c.Due != "2026-03-03" || c.Zone != "Asia/Tokyo" || c.Binding != b.Binding.ID || c.NetDays != 30 {
		t.Fatalf("wrong actual timezone coverage: %+v", c)
	}
	actual, err := f.s.financialIssuance.terms.Cover(SampleWorkspace, b.Binding.Contract, at)
	f.must(err)
	if c != actual {
		t.Fatal("service substituted coverage")
	}
	_, err = coverageAs(f, author, b, at)
	documentFailure(t, err, commands.PermissionDenied)
	b.Term.Profile.Source.Basis.Rows[0].Net = "999"
	stored, err := approveBindingAs(f, decider, f.s.financialIssuance.bindingIntents[b.Spec.ID])
	f.must(err)
	if stored.Term.Profile.Source.Basis.Rows[0].Net == "999" {
		t.Fatal("binding copy aliases source")
	}
	after, err := f.s.approvals.RequestOf(SampleWorkspace, "issuance-final")
	f.must(err)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("configuration consumed financial approval")
	}
}
func TestFinancialIssuanceTermsAllInterruptionBoundaries(t *testing.T) {
	for _, operation := range []string{"term-register", "term-approve", "binding-register", "binding-approve"} {
		for _, point := range []string{"before", "after", "publication"} {
			t.Run(operation+"/"+point, func(t *testing.T) {
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
				family, verb := "term", "register"
				if operation == "binding-register" || operation == "binding-approve" {
					family = "binding"
				}
				if operation == "term-approve" || operation == "binding-approve" {
					verb = "approve"
				}
				fault := "issuance-" + family + "-" + point + "-" + verb
				if point == "publication" {
					fault = "issuance-" + family + "-after-" + verb + "-publication"
				}
				f.s.fault = func(at string) bool { return at == fault }
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
				documentFailure(t, run(), commands.DependencyUnavailable)
				f.s.fault = nil
				if verb == "register" && point == "before" {
					if family == "term" && (len(a.termIntents) != 0 || len(a.termScopes) != 0) || family == "binding" && len(a.bindingIntents) != 0 {
						t.Fatal("before-write reserved ownership")
					}
				} else {
					expected := "registration_unknown"
					if verb == "approve" {
						expected = "approval_unknown"
					}
					if point == "before" {
						expected = "pending"
					}
					if point == "publication" {
						expected = "pending"
						if verb == "approve" {
							expected = "approved"
						}
					}
					state := a.termIntents[spec.ID].State
					if family == "binding" {
						state = a.bindingIntents[bs.ID].State
					}
					if state != expected {
						t.Fatalf("retained state=%s want=%s", state, expected)
					}
					if family == "term" {
						other := spec
						other.ID = "competing-term"
						_, e := recordTermAs(f, author, other)
						documentFailure(t, e, commands.PrerequisiteFailed)
					} else {
						other := bs
						other.ID = "competing-binding"
						_, e := bindTermAs(f, author, other)
						documentFailure(t, e, commands.PrerequisiteFailed)
					}
				}
				f.must(run())
				f.must(run())
				if family == "term" {
					v, err := a.terms.Term(SampleWorkspace, a.termIntents[spec.ID].Term.ID, spec.Revision)
					f.must(err)
					if v != a.termIntents[spec.ID].Module {
						t.Fatal("term recovery lacks exact readback")
					}
				} else {
					v, err := a.terms.BindingOf(SampleWorkspace, a.bindingIntents[bs.ID].Binding.ID)
					f.must(err)
					if !reflect.DeepEqual(v, a.bindingIntents[bs.ID].Module) {
						t.Fatal("binding recovery lacks exact readback")
					}
				}
			})
		}
	}
}
func TestFinancialIssuanceBindingIntervals(t *testing.T) {
	for _, open := range []bool{false, true} {
		t.Run(map[bool]string{false: "bounded", true: "open"}[open], func(t *testing.T) {
			f, author, decider, p := approvedTermWorld(t)
			spec := bindingSpec(p)
			spec.To = "2026-01-31"
			if open {
				spec.To = ""
			}
			b, err := bindTermAs(f, author, spec)
			f.must(err)
			_, err = approveBindingAs(f, decider, b)
			f.must(err)
			overlap := spec
			overlap.ID = "overlap"
			overlap.From = "2026-01-31"
			overlap.To = "2026-02-28"
			_, err = bindTermAs(f, author, overlap)
			documentFailure(t, err, commands.PrerequisiteFailed)
			adjacent := overlap
			adjacent.ID = "adjacent"
			adjacent.From = "2026-02-01"
			next, err := bindTermAs(f, author, adjacent)
			if open {
				documentFailure(t, err, commands.PrerequisiteFailed)
			} else {
				f.must(err)
				_, err = approveBindingAs(f, decider, next)
				f.must(err)
			}
		})
	}
}
func TestFinancialIssuanceTermsForeignAndMalformed(t *testing.T) {
	f, author, _, spec := termsWorld(t)
	for _, mutate := range []func(*financialTermSpec){func(p *financialTermSpec) { p.NetDays = -1 }, func(p *financialTermSpec) { p.NetDays = 366 }, func(p *financialTermSpec) { p.Revision = 0 }, func(p *financialTermSpec) { p.Mode = receivables.ModeBusinessDay }} {
		bad := spec
		mutate(&bad)
		_, err := recordTermAs(f, author, bad)
		documentFailure(t, err, commands.InvalidRequest)
	}
	if len(f.s.financialIssuance.termIntents) != 0 {
		t.Fatal("invalid term reserved")
	}
	id, err := financialEvidenceDigest("issuance_term", spec.ID)
	f.must(err)
	_, err = f.s.financialIssuance.terms.RecordTerm(SampleWorkspace, receivables.Term{ID: "issued-term-" + id, Revision: 1, NetDays: 30, Mode: receivables.ModeCalendar}, SampleMember)
	f.must(err)
	_, err = recordTermAs(f, author, spec)
	documentFailure(t, err, commands.EvidenceStale)
	if len(f.s.financialIssuance.termIntents) != 0 {
		t.Fatal("foreign term adopted")
	}
}
