package server

import (
	"errors"
	"reflect"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
)

func configurationProposed(p financialIssuanceConfiguration) bool {
	expected := issuance.KindView{Kind: p.Kind, State: issuance.StateProposed, Definer: p.Proposer, At: p.Proposed.At}
	return !p.Proposed.At.IsZero() && reflect.DeepEqual(p.Proposed, expected)
}

func configurationActive(p financialIssuanceConfiguration) bool {
	expected := issuance.KindView{Kind: p.Kind, State: issuance.StateActive, Definer: p.Proposer, Approver: p.Confirmer, At: p.Active.At}
	return configurationProposed(p) && p.Confirmer != "" && p.Confirmer != p.Proposer && !p.Active.At.IsZero() && !p.Active.At.Before(p.Proposed.At) && reflect.DeepEqual(p.Active, expected)
}

func (a *financialIssuance) configurationLookup(workspace, id string) (issuance.KindView, error) {
	if a.server.faulty("issuance-configuration-lookup") {
		return issuance.KindView{}, commands.Fail(commands.DependencyUnavailable)
	}
	return a.documents.KindOf(workspace, id)
}

func (a *financialIssuance) configurationReadback(workspace string, p financialIssuanceConfiguration) (financialIssuanceConfiguration, error) {
	expected := p.Proposed
	switch p.State {
	case "proposed":
		if !configurationProposed(p) || p.Confirmer != "" || !reflect.DeepEqual(p.Active, issuance.KindView{}) || !p.ActivatedAt.IsZero() {
			return financialIssuanceConfiguration{}, commands.Fail(commands.EvidenceStale)
		}
	case "confirmed":
		if !configurationActive(p) || p.ActivatedAt != p.Active.At {
			return financialIssuanceConfiguration{}, commands.Fail(commands.EvidenceStale)
		}
		expected = p.Active
	default:
		return financialIssuanceConfiguration{}, commands.Fail(commands.DependencyUnavailable)
	}
	actual, err := a.configurationLookup(workspace, p.Kind.ID)
	if err != nil {
		return financialIssuanceConfiguration{}, err
	}
	if !reflect.DeepEqual(actual, expected) {
		return financialIssuanceConfiguration{}, commands.Fail(commands.EvidenceStale)
	}
	return copyConfiguration(p), nil
}

func (a *financialIssuance) proposeConfiguration(workspace, actor, reason string, spec financialConfigurationSpec) (financialIssuanceConfiguration, error) {
	return a.proposeConfigurationTracked(workspace, actor, reason, spec, nil, nil, nil)
}

func (a *financialIssuance) proposeConfigurationTracked(workspace, actor, reason string, spec financialConfigurationSpec, setup *financialConfigurationSetupContext, effect *financialConfigurationEffect, admit func(financialIssuanceConfiguration) error) (financialIssuanceConfiguration, error) {
	if err := a.protectedRuleCaller(workspace, actor); err != nil {
		return financialIssuanceConfiguration{}, err
	}
	if !validConfigurationSetup(setup) {
		return financialIssuanceConfiguration{}, commands.Fail(commands.InvalidRequest)
	}
	kind, err := configurationKind(spec)
	if err != nil || !documentReason(reason) {
		return financialIssuanceConfiguration{}, commands.Fail(commands.InvalidRequest)
	}
	spec.Languages = kind.Languages
	scope, err := a.configurationScope(workspace, spec)
	if err != nil {
		return financialIssuanceConfiguration{}, err
	}
	p := financialIssuanceConfiguration{Schema: "hb_issuance_configuration_v1", Workspace: workspace, Spec: spec, Scope: scope, Kind: kind, Proposer: actor, Reason: reason, Setup: setup}
	p.Digest, err = configurationDigest(p)
	if err != nil {
		return financialIssuanceConfiguration{}, err
	}
	key, err := configurationScopeKey(workspace, spec)
	if err != nil {
		return financialIssuanceConfiguration{}, err
	}
	if admit != nil {
		if err := admit(p); err != nil {
			return financialIssuanceConfiguration{}, err
		}
	}
	if old, ok := a.configurations[spec.ID]; ok {
		if old.Spec.ID != spec.ID || old.Proposer != actor || old.Digest != p.Digest {
			return financialIssuanceConfiguration{}, commands.Fail(commands.EvidenceStale)
		}
		if err := a.currentConfiguration(workspace, old); err != nil {
			return financialIssuanceConfiguration{}, err
		}
		if old.State != "proposal_unknown" {
			return a.configurationReadback(workspace, old)
		}
		if old.Confirmer != "" || !reflect.DeepEqual(old.Active, issuance.KindView{}) || !old.ActivatedAt.IsZero() {
			return financialIssuanceConfiguration{}, commands.Fail(commands.EvidenceStale)
		}
		p = old
	} else {
		if len(a.configurations) >= financialIssuanceLimit || len(a.configurationScopes) >= financialIssuanceLimit || a.configurationScopes[key] != "" {
			return financialIssuanceConfiguration{}, commands.Fail(commands.PrerequisiteFailed)
		}
		for _, id := range a.configurationScopes {
			if id == spec.ID {
				return financialIssuanceConfiguration{}, commands.Fail(commands.EvidenceStale)
			}
		}
		for id, other := range a.configurations {
			otherKey, e := configurationScopeKey(other.Workspace, other.Spec)
			if e != nil || otherKey == key || other.Spec.ID == spec.ID || other.Spec.ID != id {
				return financialIssuanceConfiguration{}, commands.Fail(commands.EvidenceStale)
			}
		}
		if _, err := a.configurationLookup(workspace, kind.ID); !errors.Is(err, issuance.ErrNotFound) {
			if err != nil {
				return financialIssuanceConfiguration{}, err
			}
			return financialIssuanceConfiguration{}, commands.Fail(commands.EvidenceStale)
		}
		if a.server.faulty("issuance-configuration-propose-before-intent") {
			return financialIssuanceConfiguration{}, commands.Fail(commands.DependencyUnavailable)
		}
		p.State = "proposal_unknown"
		a.storeConfiguration(p, effect)
		a.configurationScopes[key] = spec.ID
		effect.mark()
		if a.server.faulty("issuance-configuration-propose-after-intent") {
			return financialIssuanceConfiguration{}, commands.Fail(commands.DependencyUnavailable)
		}
	}
	actual, err := a.configurationLookup(workspace, kind.ID)
	if reflect.DeepEqual(p.Proposed, issuance.KindView{}) {
		if !errors.Is(err, issuance.ErrNotFound) {
			if err != nil {
				return financialIssuanceConfiguration{}, err
			}
			return financialIssuanceConfiguration{}, commands.Fail(commands.EvidenceStale)
		}
		effect.mark()
		actual, err = a.documents.Define(workspace, kind, issuance.Actor{ID: actor, Kind: issuance.KindHuman})
		if err != nil {
			return financialIssuanceConfiguration{}, err
		}
		p.Proposed = actual
		a.storeConfiguration(p, effect)
	} else if err != nil || !configurationProposed(p) || !reflect.DeepEqual(actual, p.Proposed) {
		if err != nil {
			return financialIssuanceConfiguration{}, err
		}
		return financialIssuanceConfiguration{}, commands.Fail(commands.EvidenceStale)
	}
	if !configurationProposed(p) {
		return financialIssuanceConfiguration{}, commands.Fail(commands.EvidenceStale)
	}
	if a.server.faulty("issuance-configuration-propose-after-module") {
		return financialIssuanceConfiguration{}, commands.Fail(commands.DependencyUnavailable)
	}
	p.State = "proposed"
	a.storeConfiguration(p, effect)
	if a.server.faulty("issuance-configuration-propose-after-publication") {
		return financialIssuanceConfiguration{}, commands.Fail(commands.DependencyUnavailable)
	}
	return copyConfiguration(p), nil
}

func (a *financialIssuance) confirmConfiguration(workspace, id, digest, actor string) (financialIssuanceConfiguration, error) {
	return a.confirmConfigurationTracked(workspace, id, digest, actor, nil, nil)
}

func (a *financialIssuance) confirmConfigurationTracked(workspace, id, digest, actor string, effect *financialConfigurationEffect, admit func(financialIssuanceConfiguration) error) (financialIssuanceConfiguration, error) {
	if err := a.protectedRuleCaller(workspace, actor); err != nil {
		return financialIssuanceConfiguration{}, err
	}
	p, ok := a.configurations[id]
	if !ok || p.Spec.ID != id || p.Digest != digest {
		return financialIssuanceConfiguration{}, commands.Fail(commands.EvidenceStale)
	}
	if p.Proposer == actor {
		return financialIssuanceConfiguration{}, commands.Fail(commands.SelfApprovalDenied)
	}
	if p.Confirmer != "" && p.Confirmer != actor {
		return financialIssuanceConfiguration{}, commands.Fail(commands.PermissionDenied)
	}
	if err := a.currentConfiguration(workspace, p); err != nil {
		return financialIssuanceConfiguration{}, err
	}
	if admit != nil {
		if err := admit(p); err != nil {
			return financialIssuanceConfiguration{}, err
		}
	}
	if p.State == "confirmed" {
		return a.configurationReadback(workspace, p)
	}
	if p.State != "proposed" && p.State != "confirmation_unknown" {
		return financialIssuanceConfiguration{}, commands.Fail(commands.PrerequisiteFailed)
	}
	if !configurationProposed(p) || !p.ActivatedAt.IsZero() || (p.State == "proposed" && p.Confirmer != "") || (p.State == "confirmation_unknown" && p.Confirmer != actor) {
		return financialIssuanceConfiguration{}, commands.Fail(commands.EvidenceStale)
	}
	actual, err := a.configurationLookup(workspace, p.Kind.ID)
	if err != nil {
		return financialIssuanceConfiguration{}, err
	}
	observed := !reflect.DeepEqual(p.Active, issuance.KindView{})
	if observed {
		if p.State != "confirmation_unknown" || !configurationActive(p) || !reflect.DeepEqual(actual, p.Active) {
			return financialIssuanceConfiguration{}, commands.Fail(commands.EvidenceStale)
		}
	} else {
		if !reflect.DeepEqual(actual, p.Proposed) {
			return financialIssuanceConfiguration{}, commands.Fail(commands.EvidenceStale)
		}
		if a.server.faulty("issuance-configuration-confirm-before-intent") {
			return financialIssuanceConfiguration{}, commands.Fail(commands.DependencyUnavailable)
		}
		p.State, p.Confirmer = "confirmation_unknown", actor
		a.storeConfiguration(p, effect)
		if a.server.faulty("issuance-configuration-confirm-after-intent") {
			return financialIssuanceConfiguration{}, commands.Fail(commands.DependencyUnavailable)
		}
		effect.mark()
		p.Active, err = a.documents.ApproveKind(workspace, p.Kind.ID, issuance.Actor{ID: actor, Kind: issuance.KindHuman})
		if err != nil {
			return financialIssuanceConfiguration{}, err
		}
		a.storeConfiguration(p, effect)
	}
	if !configurationActive(p) {
		return financialIssuanceConfiguration{}, commands.Fail(commands.EvidenceStale)
	}
	if a.server.faulty("issuance-configuration-confirm-after-module") {
		return financialIssuanceConfiguration{}, commands.Fail(commands.DependencyUnavailable)
	}
	p.State, p.ActivatedAt = "confirmed", p.Active.At
	a.storeConfiguration(p, effect)
	if a.server.faulty("issuance-configuration-confirm-after-publication") {
		return financialIssuanceConfiguration{}, commands.Fail(commands.DependencyUnavailable)
	}
	return copyConfiguration(p), nil
}
