package server

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/passwords"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
	"github.com/myblueton-sw/hybrid_billing/modules/protectedauthority"
)

func protectedChangeWorld(t *testing.T, age time.Duration, approvals int, ruleOverride ...protected.Rules) (world, actor, actor, actor, financialChangeSpec) {
	t.Helper()
	f, second, first, digest, channels := matureIssuanceWorld(t, age, approvals, ruleOverride...)
	profile, err := proposeIssuanceProfile(f, first, digest, issuanceSpec())
	f.must(err)
	profile, err = confirmIssuanceProfile(f, second, profile)
	f.must(err)
	activateProtectedMember(f, "protected-fifth")
	grantChannelCapabilityAt(f, "request-", protectedauthority.RequestChange, SampleAdmin2)
	grantChannelCapabilityAt(f, "verify-", protectedauthority.VerifyChange, SampleAdmin)
	grantChannelCapabilityAt(f, "approve-", protectedauthority.ApproveChange, "protected-fifth")
	for _, login := range []string{AdminLogin, Admin2Login, Admin3Login, MemberLogin} {
		for _, v := range f.s.sessions.byLogin(login) {
			f.s.endProbe(v)
		}
	}
	spec := financialChangeSpec{ID: "protected-change", Profile: profile.Spec.ID, ProfileDigest: profile.Digest, Field: protected.FieldDocumentTemplate, Channel: channels["kind"].Spec.ID, ChannelDigest: channels["kind"].Digest, Provenance: protected.ProvenanceStaff, Reason: "Explicit synthetic template reference", After: protected.Ref{ID: "synthetic-template", Fingerprint: strings.Repeat("b", 64)}, Instruction: protected.Instruction{Kind: protected.ChannelInPerson, Evidence: "synthetic-incoming-instruction"}, Effective: f.s.clock().Add(time.Minute)}
	return f, f.as(Admin2Login), f.as(AdminLogin), f.as("protected-fifth@example.invalid"), spec
}
func requestChangeAs(f world, who actor, spec financialChangeSpec) (financialChangeRecord, error) {
	var p financialChangeRecord
	err := inputAuthorityAs(f, who, func() error {
		var err error
		p, err = f.s.financialIssuance.requestProtectedChange(SampleWorkspace, f.s.answer.User(), spec)
		return err
	})
	return p, err
}
func verifyChangeAs(f world, who actor, p financialChangeRecord) (financialChangeRecord, error) {
	var out financialChangeRecord
	err := inputAuthorityAs(f, who, func() error {
		var err error
		out, err = f.s.financialIssuance.verifyProtectedChange(SampleWorkspace, p.Spec.ID, p.Digest, f.s.answer.User(), "declared-synthetic-verification", protected.OutcomeConfirmed)
		return err
	})
	return out, err
}
func decideChangeAs(f world, who actor, p financialChangeRecord) (financialChangeRecord, error) {
	var out financialChangeRecord
	err := inputAuthorityAs(f, who, func() error {
		var err error
		out, err = f.s.financialIssuance.decideProtectedChange(SampleWorkspace, p.Spec.ID, p.Digest, f.s.answer.User(), protected.VerdictApprove, "")
		return err
	})
	return out, err
}
func TestFinancialProtectedChangeActualMatureFlow(t *testing.T) {
	f, requester, verifier, approver, spec := protectedChangeWorld(t, 24*time.Hour, 1)
	p, err := requestChangeAs(f, requester, spec)
	f.must(err)
	if p.Module.Requested.Sub(p.Channel.Module.Admitted) != 24*time.Hour {
		t.Fatal("channel maturity was not actual elapsed time")
	}
	p, err = verifyChangeAs(f, verifier, p)
	f.must(err)
	if p.Module.Verification.Verification.Digest != p.Module.Digest || p.Module.Verification.Digest == p.Module.Digest {
		t.Fatal("request and verification digest confusion")
	}
	p, err = decideChangeAs(f, approver, p)
	f.must(err)
	if p.Module.State != protected.StateScheduled || len(p.Module.Decisions) != 1 {
		t.Fatal("actual protected approval missing")
	}
	replay, err := decideChangeAs(f, approver, p)
	f.must(err)
	if !reflect.DeepEqual(p, replay) {
		t.Fatal("approval replay changed evidence")
	}
	f.advance(time.Minute)
	actual, err := f.s.financialIssuance.fields.Resolve(SampleWorkspace, p.Channel.Subject.Subject, p.Spec.Field, f.s.clock())
	f.must(err)
	if actual.Change != p.Spec.ID || actual.Ref != p.Spec.After || !actual.Effective.Equal(p.Spec.Effective) {
		t.Fatal("actual resolution differs")
	}
	p.Module.Verification.Verifier = "changed"
	p.Module.Decisions[0].Approver = "changed"
	replay, err = decideChangeAs(f, approver, replay)
	f.must(err)
	if replay.Module.Verification.Verifier == "changed" || replay.Module.Decisions[0].Approver == "changed" {
		t.Fatal("returned view alias")
	}
}
func TestFinancialProtectedChangeTooYoungNoEffect(t *testing.T) {
	f, requester, _, _, spec := protectedChangeWorld(t, 24*time.Hour-time.Nanosecond, 1)
	_, err := requestChangeAs(f, requester, spec)
	documentFailure(t, err, commands.PrerequisiteFailed)
	if len(f.s.financialIssuance.changes) != 0 || len(f.s.financialIssuance.changeOrders) != 0 {
		t.Fatal("too-young request reserved ownership")
	}
	f.advance(time.Nanosecond)
	_, err = requestChangeAs(f, requester, spec)
	f.must(err)
}
func TestFinancialProtectedChangeInterruptionRecovery(t *testing.T) {
	for _, point := range []string{"before-request", "after-request", "after-request-publication", "before-verify", "after-verify", "after-verification-publication", "before-decide", "after-decide", "after-decision-publication"} {
		t.Run(point, func(t *testing.T) {
			f, requester, verifier, approver, spec := protectedChangeWorld(t, 24*time.Hour, 1)
			verify := strings.Contains(point, "verif")
			decide := strings.Contains(point, "decid") || strings.Contains(point, "decision")
			var p financialChangeRecord
			var err error
			if verify || decide {
				p, err = requestChangeAs(f, requester, spec)
				f.must(err)
			}
			if decide {
				p, err = verifyChangeAs(f, verifier, p)
				f.must(err)
			}
			f.s.fault = func(at string) bool { return at == "issuance-change-"+point }
			if decide {
				_, err = decideChangeAs(f, approver, p)
			} else if verify {
				_, err = verifyChangeAs(f, verifier, p)
			} else {
				_, err = requestChangeAs(f, requester, spec)
			}
			documentFailure(t, err, commands.DependencyUnavailable)
			f.s.fault = nil
			if point == "before-request" && len(f.s.financialIssuance.changes) != 0 {
				t.Fatal("before-request retained intent")
			}
			p, err = requestChangeAs(f, requester, spec)
			f.must(err)
			if decide {
				p, err = decideChangeAs(f, approver, p)
				f.must(err)
			} else {
				p, err = verifyChangeAs(f, verifier, p)
				f.must(err)
				p, err = decideChangeAs(f, approver, p)
				f.must(err)
			}
			if p.Module.State != protected.StateScheduled || len(p.Module.Decisions) != 1 || len(f.s.financialIssuance.changes) != 1 {
				t.Fatal("lost exact recovery")
			}
		})
	}
}

func activateProtectedMember(f world, id string) string {
	login := id + "@example.invalid"
	people := newPopulation(f)
	people.member(id, login, identity.Member)
	token, err := f.s.recordLink(login, passwords.PurposeSet, f.s.clock())
	f.must(err)
	body, err := json.Marshal(map[string]string{"token": token, "password": pw})
	f.must(err)
	if reply := f.auth("redeem", string(body), f.pre()); reply.status != 204 {
		f.t.Fatalf("fifth member activation: status %d", reply.status)
	}
	return login
}
