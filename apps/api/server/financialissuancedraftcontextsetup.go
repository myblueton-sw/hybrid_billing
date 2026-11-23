package server

import (
	"errors"
	"reflect"
	"sort"
	"strconv"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
	"github.com/myblueton-sw/hybrid_billing/modules/receivables"
)

func draftContextSourceStatus(err error) string {
	var failure *commands.Failure
	if errors.As(err, &failure) && (failure.Code == commands.PrerequisiteFailed || failure.Code == commands.PermissionDenied) {
		return "ineligible"
	}
	return "unavailable"
}

func draftContextMissingSetup(err error) bool {
	var refusal receivables.Refusal
	return errors.Is(err, protected.ErrNotFound) || errors.As(err, &refusal) && refusal.Reason == receivables.ReasonTermMissing
}

func (a *financialIssuance) contextConfiguration(source financialIssuanceSourceView, existing *financialDraftObservation) (*financialIssuanceConfiguration, string, error) {
	if a.documents == nil || len(a.configurations) > financialIssuanceLimit || len(a.configurationScopes) > financialIssuanceLimit {
		return nil, "unavailable", commands.Fail(commands.DependencyUnavailable)
	}
	if existing != nil {
		if existing.Draft.Application == nil {
			return nil, "existing_draft", nil
		}
		p := existing.Draft.Application
		c, err := a.ownedConfirmedConfiguration(source.Basis.Workspace, p.Configuration.Spec.ID, p.Configuration.Digest)
		if err != nil {
			return nil, "unavailable", err
		}
		if !reflect.DeepEqual(c, p.Configuration) || !reflect.DeepEqual(source, p.Source) {
			return nil, "unavailable", commands.Fail(commands.EvidenceStale)
		}
		return &c, "ready", nil
	}

	expectedScope, err := configurationScopeKey(source.Basis.Workspace, financialConfigurationSpec{Contract: source.Basis.Contract.ContractID, ContractRevision: source.Basis.Contract.Revision, BillingProfile: source.Basis.BillingProfile.ID, BillingRevision: source.Basis.BillingRevision})
	if err != nil {
		return nil, "unavailable", err
	}
	if id, reserved := a.configurationScopes[expectedScope]; reserved {
		owned, exists := a.configurations[id]
		actualScope, err := configurationScopeKey(owned.Workspace, owned.Spec)
		if !exists || err != nil || actualScope != expectedScope || owned.Spec.ID != id {
			return nil, "unavailable", commands.Fail(commands.EvidenceStale)
		}
	}
	var found *financialIssuanceConfiguration
	ids := make([]string, 0, len(a.configurations))
	for id := range a.configurations {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		p := a.configurations[id]
		hash, err := configurationDigest(p)
		if err != nil || hash != p.Digest || p.Spec.ID != id {
			return nil, "unavailable", commands.Fail(commands.EvidenceStale)
		}
		s := source.Basis
		if p.Workspace != s.Workspace || p.Spec.Contract != s.Contract.ContractID || p.Spec.ContractRevision != s.Contract.Revision || p.Spec.BillingProfile != s.BillingProfile.ID || p.Spec.BillingRevision != s.BillingRevision {
			continue
		}
		c, err := a.ownedConfirmedConfiguration(s.Workspace, id, p.Digest)
		if err != nil {
			return nil, "unavailable", err
		}
		if err := applicationApplicability(financialIssuanceApplication{Schema: "hb_issuance_application_v1", Configuration: c, Source: source}); err != nil {
			return nil, "unavailable", err
		}
		if found != nil {
			return nil, "configuration_ambiguous", nil
		}
		found = &c
	}
	if found == nil {
		return nil, "configuration_missing", nil
	}
	return found, "ready", nil
}

func (a *financialIssuance) contextBindingChoices(c financialIssuanceConfiguration, at time.Time) ([]map[string]string, error) {
	if a.terms == nil || len(a.bindingIntents) > financialIssuanceLimit {
		return nil, commands.Fail(commands.DependencyUnavailable)
	}
	location, err := time.LoadLocation(c.Spec.Zone)
	if err != nil {
		return nil, commands.Fail(commands.DependencyUnavailable)
	}
	date := at.In(location).Format(time.DateOnly)
	ids := make([]string, 0, len(a.bindingIntents))
	for id := range a.bindingIntents {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var choices []map[string]string
	for _, id := range ids {
		p := a.bindingIntents[id]
		if p.Term.Configuration == nil || p.Term.Configuration.Workspace != c.Workspace || p.Term.Configuration.Spec.ID != c.Spec.ID {
			continue
		}
		digest, err := bindingIntentDigest(p)
		if err != nil || digest != p.Digest || p.Spec.ID != id || !reflect.DeepEqual(*p.Term.Configuration, c) {
			return nil, commands.Fail(commands.EvidenceStale)
		}
		if err := a.contextRetainedBinding(c, p); err != nil {
			return nil, err
		}
		if p.State == "pending" || date < p.Binding.From || date > termBindingEnd(p.Binding.To) {
			continue
		}
		coverage, err := a.configuredCoverage(c, id, p.Digest, at)
		if err != nil {
			return nil, err
		}
		choices = append(choices, map[string]string{"binding": id, "binding_digest": p.Digest, "term_revision": strconv.Itoa(coverage.Revision), "net_days": strconv.Itoa(coverage.NetDays), "calendar_mode": coverage.Mode, "zone": coverage.Zone, "from": p.Spec.From, "to": p.Spec.To})
	}
	return choices, nil
}

func (a *financialIssuance) contextRetainedBinding(c financialIssuanceConfiguration, p financialTermBinding) error {
	term, ok := a.termIntents[p.Spec.Term]
	if !ok || !reflect.DeepEqual(term, p.Term) || term.State != "approved" || p.Binding.Contract != c.Scope.Contract.ContractID || p.Binding.Zone != c.Spec.Zone || p.Binding.From != p.Spec.From || p.Binding.To != p.Spec.To || !termBindingDate(p.Binding.From) || p.Binding.To != "" && (!termBindingDate(p.Binding.To) || p.Binding.To < p.Binding.From) {
		return commands.Fail(commands.EvidenceStale)
	}
	if err := a.termScopeOwned(term); err != nil {
		return err
	}
	if _, err := a.termReadback(c.Workspace, term); err != nil {
		return err
	}
	_, err := a.bindingReadback(c.Workspace, p)
	return err
}

func draftContextConstructible(c financialIssuanceConfiguration, source financialIssuanceSourceView) error {
	if len(c.Spec.Languages) == 0 {
		return commands.Fail(commands.PrerequisiteFailed)
	}
	p := financialIssuanceApplication{Schema: "hb_issuance_application_v1", Configuration: c, Source: source}
	var err error
	p.Digest, err = applicationDigest(p)
	if err != nil {
		return err
	}
	_, err = issuanceDraftFromApplication(p, "issuance-context-probe", c.Spec.Languages[0])
	return err
}
