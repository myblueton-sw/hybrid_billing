package server

import (
	"reflect"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
	"github.com/myblueton-sw/hybrid_billing/modules/protectedauthority"
)

func matureIssuanceWorld(t *testing.T, age time.Duration, approvalCount int, ruleOverride ...protected.Rules) (world, actor, actor, string, map[string]financialChannelRecord) {
	t.Helper()
	var recorded map[string]financialChannelRecord
	f, second, first, digest := issuanceSourceWorld(t, func(f world) {
		configured := issuanceRules()
		if len(ruleOverride) > 1 {
			t.Fatal("multiple maturity rule configurations")
		}
		if len(ruleOverride) == 1 {
			configured = ruleOverride[0]
		}
		configured.Approvals = approvalCount
		var registrar, admitter actor
		var spec financialChannelSpec
		f, registrar, admitter, spec = configuredChannelWorld(f, configured)
		recorded = map[string]financialChannelRecord{}
		for _, sel := range []financialChannelSelector{spec.Selector, {Kind: "customer", Contract: "sample-contract-01", Revision: 2}, {Kind: "entity", BillingProfile: "sample-profile-north"}} {
			spec.ID = "mature-" + sel.Kind
			spec.Selector = sel
			p, err := registerChannelAs(f, registrar, spec)
			f.must(err)
			p, err = admitChannelAs(f, admitter, p)
			f.must(err)
			recorded[sel.Kind] = p
		}
		if len(f.s.financialIssuance.profiles) != 0 {
			t.Fatal("financial source created before channel setup")
		}
		if _, err := f.s.approvals.RequestOf(SampleWorkspace, "issuance-final"); err == nil {
			t.Fatal("final request exists before clock advance")
		}
		f.advance(age)
		for _, login := range []string{AdminLogin, Admin2Login, Admin3Login, MemberLogin} {
			for _, v := range f.s.sessions.byLogin(login) {
				f.s.endProbe(v)
			}
		}
		_, err := registerChannelAs(f, f.as(MemberLogin), recorded["kind"].Spec)
		documentFailure(t, err, commands.PermissionDenied)
		grantChannelCapabilityAt(f, "fresh-", protectedauthority.RegisterChannel, SampleMember)
		grantChannelCapabilityAt(f, "fresh-", protectedauthority.AdmitChannel, SampleAdmin3)
	})
	return f, second, first, digest, recorded
}

func TestFinancialChannelMaturesBeforeFreshFinancialSource(t *testing.T) {
	f, second, first, digest, recorded := matureIssuanceWorld(t, 24*time.Hour, 1)
	initial := recorded["kind"].Module.Admitted
	if !f.s.clock().Equal(initial.Add(24 * time.Hour)) {
		t.Fatal("fixture changed monotonic advancement")
	}
	p, err := proposeIssuanceProfile(f, first, digest, issuanceSpec())
	f.must(err)
	p, err = confirmIssuanceProfile(f, second, p)
	f.must(err)
	if !reflect.DeepEqual(p.Source.Basis.Contract, recorded["customer"].Subject.Contract) || !reflect.DeepEqual(p.Source.Basis.BillingProfile, recorded["entity"].Subject.BillingProfile) || p.Source.Basis.BillingRevision != recorded["entity"].Subject.BillingVersion || "entity:"+p.Source.Basis.Document.IssuerLegalEntityID != recorded["entity"].Subject.Subject {
		t.Fatal("fresh source differs from original channel provenance")
	}
	for _, saved := range recorded {
		current, err := registerChannelAs(f, f.as(MemberLogin), saved.Spec)
		f.must(err)
		if !reflect.DeepEqual(current, saved) || f.s.clock().Sub(current.Module.Admitted) < 24*time.Hour {
			t.Fatal("channel changed or failed to age")
		}
	}
}
