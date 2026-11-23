package server

import (
	"reflect"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
	"github.com/myblueton-sw/hybrid_billing/modules/protectedauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/termsauthority"
)

func TestFinancialConfiguredSetupMixedOriginsAndSnapshots(t *testing.T) {
	w := configuredSetup(t)
	f := w.f
	a := f.s.financialIssuance
	for _, mode := range []string{"missing", "unknown", "legacy-id", "legacy-digest"} {
		t.Run(mode, func(t *testing.T) {
			change := w.change
			change.Configuration = copyConfigurationReference(change.Configuration)
			term := w.term
			term.Configuration = copyConfigurationReference(term.Configuration)
			switch mode {
			case "missing":
				change.Configuration = nil
				term.Configuration = nil
			case "unknown":
				change.Configuration.Schema = "unknown"
				term.Configuration.Schema = "unknown"
			case "legacy-id":
				change.Profile = "legacy"
				term.Profile = "legacy"
			case "legacy-digest":
				change.ProfileDigest = w.configuration.Digest
				term.ProfileDigest = w.configuration.Digest
			}
			_, err := requestChangeAs(f, w.requester, change)
			documentFailure(t, err, commands.InvalidRequest)
			_, err = recordTermAs(f, w.requester, term)
			documentFailure(t, err, commands.InvalidRequest)
			if len(a.changes) != 0 || len(a.termIntents) != 0 || len(a.termScopes) != 0 {
				t.Fatal("mixed origin reserved state")
			}
		})
	}
	changes, binding := completeConfiguredSetup(t, w)
	for _, mode := range []string{"missing", "legacy-payload", "changed-reference", "changed-snapshot"} {
		t.Run(mode, func(t *testing.T) {
			change := copyChange(changes[protected.FieldDocumentTemplate])
			term := copyFinancialTerm(binding.Term)
			switch mode {
			case "missing":
				change.Configuration = nil
				term.Configuration = nil
			case "legacy-payload":
				change.Profile.Reason = "legacy"
				term.Profile.Reason = "legacy"
			case "changed-reference":
				change.Spec.Configuration.ID = "other"
				term.Spec.Configuration.ID = "other"
			case "changed-snapshot":
				change.Configuration.Reason = "changed"
				term.Configuration.Reason = "changed"
			}
			_, err := changeDigest(change)
			documentFailure(t, err, commands.EvidenceStale)
			_, err = termIntentDigest(term)
			documentFailure(t, err, commands.EvidenceStale)
			corrupted := copyFinancialBinding(binding)
			corrupted.Term = term
			_, err = bindingIntentDigest(corrupted)
			documentFailure(t, err, commands.EvidenceStale)
		})
	}
}

func TestFinancialConfiguredConsumptionExactConfiguration(t *testing.T) {
	w := configuredSetup(t)
	f := w.f
	a := f.s.financialIssuance
	_, binding := completeConfiguredSetup(t, w)
	spec := configurationSpec()
	spec.ID = "different-revision"
	spec.ContractRevision = 1
	p, err := proposeConfigurationAs(f, f.as(AdminLogin), spec)
	f.must(err)
	other, err := confirmConfigurationAs(f, f.as(Admin3Login), p)
	f.must(err)
	grantDraftCapability(f, draftauthority.Submit)
	grantDraftCapability(f, draftauthority.Inspect)
	f.must(inputAuthorityAs(f, f.as(MemberLogin), func() error {
		if err := a.draftOperation(SampleWorkspace, SampleMember, draftauthority.Submit); err != nil {
			return err
		}
		_, err := a.configuredReferences(other, f.s.clock())
		documentFailure(t, err, commands.EvidenceStale)
		_, err = a.configuredCoverage(other, binding.Spec.ID, binding.Digest, f.s.clock())
		documentFailure(t, err, commands.EvidenceStale)
		_, err = a.configuredCoverage(w.configuration, binding.Spec.ID, binding.Digest, f.s.clock())
		return err
	}))
}

func TestFinancialConfiguredRecoveryRechecksOnlyConfigurationGovernor(t *testing.T) {
	w := configuredSetup(t)
	f := w.f
	a := f.s.financialIssuance
	f.s.fault = func(at string) bool {
		return at == "issuance-change-after-request" || at == "issuance-term-after-register"
	}
	_, err := requestChangeAs(f, w.requester, w.change)
	documentFailure(t, err, commands.DependencyUnavailable)
	_, err = recordTermAs(f, w.requester, w.term)
	documentFailure(t, err, commands.DependencyUnavailable)
	f.s.fault = nil
	change := copyChange(a.changes[w.change.ID])
	term := copyFinancialTerm(a.termIntents[w.term.ID])
	removed := 0
	for _, g := range f.s.admin.Grants() {
		if g.SubjectID == SampleAdmin3 && g.Terms.Action == gate.AdministerInstallation {
			f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
			removed++
		}
	}
	if removed == 0 {
		t.Fatal("configuration governor was not revoked")
	}
	f.must(inputAuthorityAs(f, w.requester, func() error {
		if err := a.protectedCaller(SampleWorkspace, "setup-requester", protectedauthority.RequestChange); err != nil {
			return err
		}
		if err := a.termCaller(SampleWorkspace, "setup-requester", termsauthority.RecordTerm); err != nil {
			return err
		}
		_, err := a.channelRules(SampleWorkspace)
		return err
	}))
	_, err = requestChangeAs(f, w.requester, w.change)
	documentFailure(t, err, commands.PermissionDenied)
	_, err = recordTermAs(f, w.requester, w.term)
	documentFailure(t, err, commands.PermissionDenied)
	actualChange, err := a.fields.Of(SampleWorkspace, w.change.ID)
	f.must(err)
	actualTerm, err := a.terms.Term(SampleWorkspace, term.Term.ID, term.Term.Revision)
	f.must(err)
	if !reflect.DeepEqual(a.changes[w.change.ID], change) || !reflect.DeepEqual(a.termIntents[w.term.ID], term) || !reflect.DeepEqual(actualChange, change.Module) || actualTerm != term.Module {
		t.Fatal("governor loss changed unknown ownership or actual effects")
	}
}

func TestFinancialConfiguredTermMissingReverseOwnership(t *testing.T) {
	w := configuredSetup(t)
	f := w.f
	a := f.s.financialIssuance
	p, err := recordTermAs(f, w.requester, w.term)
	f.must(err)
	key, err := termScopeKey(p)
	f.must(err)
	delete(a.termScopes, key)
	competitor := w.term
	competitor.ID = "competing-scope"
	_, err = recordTermAs(f, w.requester, competitor)
	documentFailure(t, err, commands.EvidenceStale)
	if len(a.termIntents) != 1 || !reflect.DeepEqual(a.termIntents[p.Spec.ID], p) {
		t.Fatal("missing reverse entry allowed competing term")
	}
}
