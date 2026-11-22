package server

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
	"github.com/myblueton-sw/hybrid_billing/modules/protectedauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/receivables"
	"github.com/myblueton-sw/hybrid_billing/modules/termsauthority"
)

type configuredSetupWorld struct {
	f                             world
	requester, verifier, approver actor
	configuration                 financialIssuanceConfiguration
	channels                      map[string]financialChannelRecord
	change                        financialChangeSpec
	term                          financialTermSpec
}

func configuredSetup(t *testing.T) configuredSetupWorld {
	t.Helper()
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	return configuredSetupIn(t, f)
}

func configuredSetupIn(t *testing.T, existing world) configuredSetupWorld {
	t.Helper()
	f, registrar, admitter, spec := configurationChannelWorldIn(t, existing)
	channels := map[string]financialChannelRecord{}
	for _, sel := range []financialChannelSelector{spec.Selector, {Kind: "customer", Contract: "sample-contract-01", Revision: 2}, {Kind: "entity", BillingProfile: "sample-profile-north"}} {
		s := spec
		s.ID = "setup-" + sel.Kind
		s.Selector = sel
		p, err := registerChannelAs(f, registrar, s)
		f.must(err)
		p, err = admitChannelAs(f, admitter, p)
		f.must(err)
		channels[sel.Kind] = p
	}
	f.advance(24 * time.Hour)
	renewInvoiceAuthorities(t, f)
	logins := map[string]string{}
	for _, entry := range []struct{ id, capability string }{{"setup-requester", protectedauthority.RequestChange}, {"setup-verifier", protectedauthority.VerifyChange}, {"setup-approver", protectedauthority.ApproveChange}} {
		logins[entry.id] = activateProtectedMember(f, entry.id)
		grantChannelCapabilityAt(f, entry.id+"-", entry.capability, entry.id)
	}
	for _, right := range []struct{ capability, recipient string }{{termsauthority.RecordTerm, "setup-requester"}, {termsauthority.BindTerm, "setup-requester"}, {termsauthority.ApproveTerm, "setup-approver"}, {termsauthority.ApproveBinding, "setup-approver"}} {
		grantTermCapability(t, f, right.capability, right.recipient)
	}
	c := *channels["configuration_kind"].Subject.Configuration
	r := configurationReference(c)
	change := financialChangeSpec{ID: "setup-template", Configuration: &r, Field: protected.FieldDocumentTemplate, Channel: channels["configuration_kind"].Spec.ID, ChannelDigest: channels["configuration_kind"].Digest, Provenance: protected.ProvenanceStaff, Reason: "Reusable synthetic protected reference", After: protected.Ref{ID: "synthetic-template", Fingerprint: strings.Repeat("b", 64)}, Instruction: protected.Instruction{Kind: protected.ChannelInPerson, Evidence: "synthetic-setup-instruction"}, Effective: f.s.clock().Add(time.Minute)}
	term := financialTermSpec{ID: "setup-term", Configuration: &r, Revision: 1, NetDays: 30, Mode: receivables.ModeCalendar, Reason: "Reusable synthetic calendar terms"}
	return configuredSetupWorld{f: f, requester: f.as(logins["setup-requester"]), verifier: f.as(logins["setup-verifier"]), approver: f.as(logins["setup-approver"]), configuration: c, channels: channels, change: change, term: term}
}

func completeConfiguredSetup(t *testing.T, w configuredSetupWorld) (map[string]financialChangeRecord, financialTermBinding) {
	t.Helper()
	f := w.f
	changes := map[string]financialChangeRecord{}
	for _, entry := range []struct{ field, kind string }{{protected.FieldPaymentAccount, "entity"}, {protected.FieldDocumentRecipient, "customer"}, {protected.FieldDocumentTemplate, "configuration_kind"}} {
		spec := w.change
		spec.ID = "configured-change-" + entry.kind
		spec.Field = entry.field
		spec.Channel = w.channels[entry.kind].Spec.ID
		spec.ChannelDigest = w.channels[entry.kind].Digest
		p, err := requestChangeAs(f, w.requester, spec)
		f.must(err)
		if p.Module.Requested.Sub(p.Channel.Module.Admitted) != 24*time.Hour {
			t.Fatal("actual channel maturity missing")
		}
		p, err = verifyChangeAs(f, w.verifier, p)
		f.must(err)
		p, err = decideChangeAs(f, w.approver, p)
		f.must(err)
		changes[entry.field] = p
	}
	term, err := recordTermAs(f, w.requester, w.term)
	f.must(err)
	term, err = approveTermAs(f, w.approver, term)
	f.must(err)
	spec := bindingSpec(term)
	spec.Zone = w.configuration.Spec.Zone
	binding, err := bindTermAs(f, w.requester, spec)
	f.must(err)
	binding, err = approveBindingAs(f, w.approver, binding)
	f.must(err)
	f.advance(time.Minute)
	return changes, binding
}

func TestFinancialConfiguredSetupBeforeFinancialApproval(t *testing.T) {
	w := configuredSetup(t)
	f := w.f
	a := f.s.financialIssuance
	changes, binding := completeConfiguredSetup(t, w)
	grantDraftCapability(f, draftauthority.Submit)
	grantDraftCapability(f, draftauthority.Inspect)
	referenceCount := 0
	f.must(inputAuthorityAs(f, f.as(MemberLogin), func() error {
		if err := a.draftOperation(SampleWorkspace, SampleMember, draftauthority.Submit); err != nil {
			return err
		}
		resolved, err := a.configuredReferences(w.configuration, f.s.clock())
		if err != nil {
			return err
		}
		for _, r := range resolved {
			p := changes[r.Field]
			if r.Change != p.Spec.ID || r.Ref != p.Spec.After || !reflect.DeepEqual(p.Configuration, &w.configuration) {
				t.Fatal("configured reference provenance differs")
			}
			referenceCount++
		}
		coverage, err := a.configuredCoverage(w.configuration, binding.Spec.ID, binding.Digest, f.s.clock())
		if err != nil {
			return err
		}
		if coverage.Binding != binding.Binding.ID || coverage.Contract != w.configuration.Scope.Contract.ContractID || coverage.NetDays != 30 || coverage.Mode != receivables.ModeCalendar || coverage.Date != "2026-11-17" || coverage.Due != "2026-12-17" {
			t.Fatal("actual configured calendar coverage differs")
		}
		return nil
	}))
	if referenceCount != 3 || !reflect.DeepEqual(binding.Term.Configuration, &w.configuration) || !reflect.DeepEqual(binding.Term.Profile, financialIssuanceProfile{}) || len(a.profiles) != 0 || len(a.kindReservations) != 0 || len(a.drafts) != 0 {
		t.Fatal("setup requires or fabricated financial profile")
	}
	_, err := f.s.approvals.RequestOf(SampleWorkspace, "issuance-final")
	if !errors.Is(err, approvals.ErrNotFound) {
		t.Fatal("setup created final evidence")
	}
	original := copyFinancialBinding(binding)
	binding.Term.Spec.Configuration.ID = "changed"
	binding.Term.Configuration.Kind.Languages[0] = "changed"
	replay, err := approveBindingAs(f, w.approver, original)
	f.must(err)
	if !reflect.DeepEqual(replay, original) {
		t.Fatal("configured binding snapshot aliased")
	}
	for _, p := range changes {
		original := copyChange(p)
		p.Spec.Configuration.ID = "changed"
		p.Configuration.Scope.Parties[0].Party.ID = "changed"
		again, err := decideChangeAs(f, w.approver, original)
		f.must(err)
		if !reflect.DeepEqual(original, again) {
			t.Fatal("configured change snapshot aliased")
		}
	}
}
