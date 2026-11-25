package server

import (
	"reflect"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/receivables"
)

func termScopeKey(p financialTermRecord) (string, error) {
	if err := setupOriginShape(p.Spec.Profile, p.Spec.ProfileDigest, p.Spec.Configuration, p.Profile, p.Configuration); err != nil {
		return "", err
	}
	if p.Configuration == nil {
		return p.Profile.Source.Basis.Subject.Target, nil
	}
	digest, err := financialEvidenceDigest("issuance_configuration_term_scope_v1", struct{ Workspace, ID, Digest string }{p.Configuration.Workspace, p.Configuration.Spec.ID, p.Configuration.Digest})
	if err != nil {
		return "", err
	}
	return "configured-term-" + digest, nil
}

func termContractAndZone(p financialTermRecord) (string, string, error) {
	if err := setupOriginShape(p.Spec.Profile, p.Spec.ProfileDigest, p.Spec.Configuration, p.Profile, p.Configuration); err != nil {
		return "", "", err
	}
	if p.Configuration != nil {
		return p.Configuration.Scope.Contract.ContractID, p.Configuration.Spec.Zone, nil
	}
	return p.Profile.Source.Basis.Contract.ContractID, p.Profile.Spec.Zone, nil
}

func configuredTermDigest(p financialTermRecord) (string, error) {
	return financialEvidenceDigest("issuance_configuration_term_v1", struct {
		Spec          financialTermSpec
		Configuration financialIssuanceConfiguration
		Term          receivables.Term
		Author        string
	}{p.Spec, *p.Configuration, p.Term, p.Author})
}

func (a *financialIssuance) termScopeOwned(p financialTermRecord) error {
	key, err := termScopeKey(p)
	if err != nil || a.termScopes[key] != p.Spec.ID {
		return commands.Fail(commands.EvidenceStale)
	}
	for otherKey, id := range a.termScopes {
		if id == p.Spec.ID && otherKey != key {
			return commands.Fail(commands.EvidenceStale)
		}
	}
	for id, other := range a.termIntents {
		otherKey, e := termScopeKey(other)
		if e != nil || (otherKey == key && id != p.Spec.ID) {
			return commands.Fail(commands.EvidenceStale)
		}
	}
	return nil
}

func (a *financialIssuance) currentTermOrigin(workspace string, p financialTermRecord, actor, capability string) error {
	if err := a.termCaller(workspace, actor, capability); err != nil {
		return err
	}
	profile, configured, err := a.freshSetupOrigin(workspace, p.Spec.Profile, p.Spec.ProfileDigest, actor, p.Spec.Configuration)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(profile, p.Profile) || !reflect.DeepEqual(configured, p.Configuration) {
		return commands.Fail(commands.EvidenceStale)
	}
	return nil
}
