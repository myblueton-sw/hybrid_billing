package server

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
	"github.com/myblueton-sw/hybrid_billing/modules/protectedauthority"
)

func TestFinancialProtectedChangeMultipleIndependentApprovals(t *testing.T) {
	f, requester, verifier, first, spec := protectedChangeWorld(t, 24*time.Hour, 3)
	sixth := activateProtectedMember(f, "protected-sixth")
	seventh := activateProtectedMember(f, "protected-seventh")
	grantChannelCapabilityAt(f, "sixth-", protectedauthority.ApproveChange, "protected-sixth")
	grantChannelCapabilityAt(f, "seventh-", protectedauthority.ApproveChange, "protected-seventh")
	second, third := f.as(sixth), f.as(seventh)
	p, err := requestChangeAs(f, requester, spec)
	f.must(err)
	p, err = verifyChangeAs(f, verifier, p)
	f.must(err)
	p, err = decideChangeAs(f, first, p)
	f.must(err)
	if p.Module.State != protected.StateConfirmed {
		t.Fatal("scheduled before configured approval count")
	}
	f.s.fault = func(at string) bool { return at == "issuance-change-after-decide" }
	_, err = decideChangeAs(f, second, p)
	documentFailure(t, err, commands.DependencyUnavailable)
	f.s.fault = nil
	saved := copyChange(f.s.financialIssuance.changes[spec.ID])
	_, err = decideChangeAs(f, first, p)
	documentFailure(t, err, commands.PermissionDenied)
	_, err = verifyChangeAs(f, verifier, p)
	documentFailure(t, err, commands.DependencyUnavailable)
	if !reflect.DeepEqual(saved, f.s.financialIssuance.changes[spec.ID]) {
		t.Fatal("earlier phase replay cleared unknown decision")
	}
	p, err = decideChangeAs(f, second, p)
	f.must(err)
	if p.Module.State != protected.StateConfirmed || len(p.Module.Decisions) != 2 {
		t.Fatal("partial decision recovery wrong")
	}
	p, err = decideChangeAs(f, third, p)
	f.must(err)
	if p.Module.State != protected.StateScheduled || len(p.Module.Decisions) != 3 {
		t.Fatal("three distinct approvals missing")
	}
}

func TestFinancialProtectedChangeUnknownLapseRecovery(t *testing.T) {
	for _, phase := range []string{"request", "verify", "decide"} {
		t.Run(phase, func(t *testing.T) {
			f, requester, verifier, approver, spec := protectedChangeWorld(t, 24*time.Hour, 2)
			var p financialChangeRecord
			var err error
			if phase != "request" {
				p, err = requestChangeAs(f, requester, spec)
				f.must(err)
			}
			if phase == "decide" {
				p, err = verifyChangeAs(f, verifier, p)
				f.must(err)
			}
			f.s.fault = func(at string) bool { return at == "issuance-change-after-"+phase }
			switch phase {
			case "request":
				_, err = requestChangeAs(f, requester, spec)
			case "verify":
				_, err = verifyChangeAs(f, verifier, p)
			case "decide":
				_, err = decideChangeAs(f, approver, p)
			}
			documentFailure(t, err, commands.DependencyUnavailable)
			f.s.fault = nil
			saved := copyChange(f.s.financialIssuance.changes[spec.ID])
			f.advance(time.Minute)
			atDeadline, err := f.s.financialIssuance.fields.Of(SampleWorkspace, spec.ID)
			f.must(err)
			if atDeadline.State == protected.StateLapsed {
				t.Fatal("lapsed at exact deadline")
			}
			f.advance(time.Nanosecond)
			switch phase {
			case "request":
				p, err = requestChangeAs(f, requester, spec)
			case "verify":
				p, err = verifyChangeAs(f, verifier, saved)
			case "decide":
				p, err = decideChangeAs(f, approver, saved)
			}
			f.must(err)
			if p.Phase != "recorded" || p.Module.State != protected.StateLapsed || !p.Module.Ended.Equal(spec.Effective) || !changeViewEqual(p.Module, saved.Module) {
				t.Fatal("natural lapse lost write evidence")
			}
			if len(p.Decisions) != len(saved.Decisions) || !reflect.DeepEqual(p.Verification, saved.Verification) {
				t.Fatal("lapse fabricated intent evidence")
			}
		})
	}
}
func TestFinancialProtectedChangeForeignTrailAndFuturePredecessor(t *testing.T) {
	f, requester, verifier, approver, spec := protectedChangeWorld(t, 24*time.Hour, 1)
	p, err := requestChangeAs(f, requester, spec)
	f.must(err)
	p, err = verifyChangeAs(f, verifier, p)
	f.must(err)
	p, err = decideChangeAs(f, approver, p)
	f.must(err)
	next := spec
	next.ID = "future-successor"
	next.Effective = spec.Effective.Add(time.Minute)
	next.After = protected.Ref{ID: "next-template", Fingerprint: strings.Repeat("c", 64)}
	q, err := requestChangeAs(f, requester, next)
	f.must(err)
	if q.Module.Before != p.Spec.After || !f.s.clock().Before(p.Spec.Effective) {
		t.Fatal("future predecessor Before was not retained")
	}
	_, err = f.s.financialIssuance.fields.Withdraw(SampleWorkspace, q.Spec.ID, protected.Actor{ID: SampleAdmin2, Kind: protected.KindHuman})
	f.must(err)
	next.ID = "foreign-trail-attempt"
	next.Effective = next.Effective.Add(time.Minute)
	_, err = requestChangeAs(f, requester, next)
	documentFailure(t, err, commands.EvidenceStale)
	if len(f.s.financialIssuance.changes) != 2 {
		t.Fatal("foreign mutation admitted new ownership")
	}
}
func TestFinancialProtectedChangeCurrentEvidenceAndRights(t *testing.T) {
	for _, mode := range []string{"requester", "channel", "source", "orphan-order", "foreign-trail", "lookup"} {
		t.Run(mode, func(t *testing.T) {
			f, requester, verifier, _, spec := protectedChangeWorld(t, 24*time.Hour, 1)
			p, err := requestChangeAs(f, requester, spec)
			f.must(err)
			a := f.s.financialIssuance
			expected := commands.EvidenceStale
			switch mode {
			case "requester":
				for _, g := range f.s.admin.Grants() {
					if g.SubjectID == SampleAdmin2 && g.Terms.Action == gate.RequestIssuanceProtectedChange {
						f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
					}
				}
				expected = commands.PermissionDenied
			case "channel":
				_, err = a.fields.Retire(SampleWorkspace, p.Spec.Channel, protected.Actor{ID: SampleMember, Kind: protected.KindHuman})
				f.must(err)
			case "source":
				key := [2]string{SampleWorkspace, "public-result"}
				v := f.s.documentERP.resultApplications[key]
				v.receipt = "changed"
				f.s.documentERP.resultApplications[key] = v
			case "orphan-order":
				delete(a.changeOrders, issuanceChangeScope(p.Channel.Subject.Subject, p.Spec.Field))
			case "foreign-trail":
				spec.Field = protected.FieldDocumentRecipient
				spec.Channel = "mature-customer"
				spec.ChannelDigest = a.channels[spec.Channel].Digest
				spec.ID = "recipient-change"
				foreign := protected.Change{ID: "foreign", Subject: a.channels[spec.Channel].Subject.Subject, Field: spec.Field, After: spec.After, Provenance: spec.Provenance, Instruction: spec.Instruction, Effective: spec.Effective}
				_, err = a.fields.Request(SampleWorkspace, foreign, protected.Actor{ID: SampleAdmin2, Kind: protected.KindHuman})
				f.must(err)
				_, err = requestChangeAs(f, requester, spec)
				documentFailure(t, err, commands.EvidenceStale)
				return
			case "lookup":
				f.s.fault = func(at string) bool { return at == "issuance-change-lookup" }
				expected = commands.DependencyUnavailable
			}
			_, err = verifyChangeAs(f, verifier, p)
			documentFailure(t, err, expected)
			if a.changes[p.Spec.ID].Digest != p.Digest || a.changes[p.Spec.ID].Verification != nil {
				t.Fatal("refusal lost original intent or verified")
			}
		})
	}
}

func TestFinancialProtectedChangeUndeclaredInstructionNoReservation(t *testing.T) {
	for _, mode := range []string{"foreign", "retired", "nonmatching", "unavailable"} {
		t.Run(mode, func(t *testing.T) {
			f, requester, _, _, spec := protectedChangeWorld(t, 24*time.Hour, 1)
			a := f.s.financialIssuance
			foreign := protected.Channel{ID: "foreign-incoming", Subject: a.channels[spec.Channel].Subject.Subject, Kind: protected.ChannelEmail, Ref: protected.Ref{ID: "foreign-ref", Fingerprint: strings.Repeat("c", 64)}, Evidence: "foreign-registration"}
			_, err := a.fields.Register(SampleWorkspace, foreign, protected.Actor{ID: SampleMember, Kind: protected.KindHuman})
			f.must(err)
			if mode == "retired" {
				_, err = a.fields.Retire(SampleWorkspace, foreign.ID, protected.Actor{ID: SampleAdmin, Kind: protected.KindHuman})
				f.must(err)
			}
			spec.Instruction = protected.Instruction{Kind: protected.ChannelEmail, Sender: foreign.Ref.Fingerprint, Evidence: "synthetic-incoming"}
			if mode == "nonmatching" {
				spec.Instruction.Sender = strings.Repeat("d", 64)
			}
			if mode == "unavailable" {
				f.s.fault = func(at string) bool { return at == "issuance-change-channel-lookup" }
			}
			_, err = requestChangeAs(f, requester, spec)
			if mode == "nonmatching" {
				f.must(err)
				return
			}
			expected := commands.InvalidRequest
			if mode == "unavailable" {
				expected = commands.DependencyUnavailable
			}
			documentFailure(t, err, expected)
			if len(a.changes) != 0 || len(a.changeOrders) != 0 {
				t.Fatal("known invalid instruction locked scope")
			}
			trail, err := a.fields.Trail(SampleWorkspace, foreign.Subject, spec.Field)
			f.must(err)
			if len(trail) != 0 {
				t.Fatal("invalid instruction wrote change")
			}
		})
	}
}
func TestFinancialProtectedChangeIntentAssociationCorruption(t *testing.T) {
	for _, mode := range []string{"verification", "decision"} {
		t.Run(mode, func(t *testing.T) {
			f, requester, verifier, approver, spec := protectedChangeWorld(t, 24*time.Hour, 1)
			p, err := requestChangeAs(f, requester, spec)
			f.must(err)
			p, err = verifyChangeAs(f, verifier, p)
			f.must(err)
			p, err = decideChangeAs(f, approver, p)
			f.must(err)
			changed := copyChange(p)
			if mode == "verification" {
				changed.Verification.Evidence = "changed"
			} else {
				changed.Decisions[0].Approval.Digest = strings.Repeat("e", 64)
			}
			f.s.financialIssuance.changes[spec.ID] = changed
			_, err = decideChangeAs(f, approver, p)
			documentFailure(t, err, commands.EvidenceStale)
		})
	}
}

func TestFinancialProtectedChangeAllCurrentProfileFields(t *testing.T) {
	f, requester, verifier, approver, spec := protectedChangeWorld(t, 24*time.Hour, 1)
	for _, entry := range []struct{ field, kind string }{{protected.FieldDocumentTemplate, "kind"}, {protected.FieldDocumentRecipient, "customer"}, {protected.FieldPaymentAccount, "entity"}} {
		spec.ID = "change-" + entry.kind
		spec.Field = entry.field
		spec.Channel = "mature-" + entry.kind
		spec.ChannelDigest = f.s.financialIssuance.channels[spec.Channel].Digest
		p, err := requestChangeAs(f, requester, spec)
		f.must(err)
		p, err = verifyChangeAs(f, verifier, p)
		f.must(err)
		p, err = decideChangeAs(f, approver, p)
		f.must(err)
		if p.Module.State != protected.StateScheduled || p.Module.Change.Subject != p.Channel.Subject.Subject {
			t.Fatal("wrong protected field subject")
		}
	}
}
func TestFinancialProtectedChangeVerificationValidityLapse(t *testing.T) {
	rules := issuanceRules()
	rules.Validity = time.Hour
	f, requester, verifier, approver, spec := protectedChangeWorld(t, 24*time.Hour, 2, rules)
	spec.Effective = f.s.clock().Add(2 * time.Hour)
	p, err := requestChangeAs(f, requester, spec)
	f.must(err)
	p, err = verifyChangeAs(f, verifier, p)
	f.must(err)
	f.s.fault = func(at string) bool { return at == "issuance-change-after-decide" }
	_, err = decideChangeAs(f, approver, p)
	documentFailure(t, err, commands.DependencyUnavailable)
	f.s.fault = nil
	f.advance(time.Hour)
	exact, err := f.s.financialIssuance.fields.Of(SampleWorkspace, spec.ID)
	f.must(err)
	if exact.State != protected.StateConfirmed {
		t.Fatal("lapsed at verification validity equality")
	}
	f.advance(time.Nanosecond)
	for _, login := range []string{AdminLogin, Admin2Login, Admin3Login, MemberLogin, "protected-fifth@example.invalid"} {
		for _, v := range f.s.sessions.byLogin(login) {
			f.s.endProbe(v)
		}
	}
	for _, entry := range []struct{ cap, recipient string }{{protectedauthority.RegisterChannel, SampleMember}, {protectedauthority.AdmitChannel, SampleAdmin3}, {protectedauthority.RequestChange, SampleAdmin2}, {protectedauthority.VerifyChange, SampleAdmin}, {protectedauthority.ApproveChange, "protected-fifth"}} {
		grantChannelCapabilityAt(f, "renew-", entry.cap, entry.recipient)
	}
	p, err = decideChangeAs(f, f.as("protected-fifth@example.invalid"), p)
	f.must(err)
	if p.Module.State != protected.StateLapsed || !p.Module.Ended.Equal(p.Module.Verification.At.Add(time.Hour)) || len(p.Module.Decisions) != 1 {
		t.Fatal("verification deadline did not preserve partial decisions")
	}
	f.advance(-time.Minute)
	module, err := f.s.financialIssuance.fields.Of(SampleWorkspace, spec.ID)
	f.must(err)
	if !reflect.DeepEqual(module, p.Module) {
		t.Fatal("raw clock rollback revived module change")
	}
}
