package server

import (
	"maps"
	"reflect"
	"slices"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
)

type financialIssuanceApplication struct {
	Schema        string
	Configuration financialIssuanceConfiguration
	Source        financialIssuanceSourceView
	Digest        string
}

func copyIssuanceSource(p financialIssuanceSourceView) financialIssuanceSourceView {
	p.Basis.Subject.Digests = maps.Clone(p.Basis.Subject.Digests)
	p.Basis.Subject.Entries = slices.Clone(p.Basis.Subject.Entries)
	p.Basis.Decision.Decision.Exclude = slices.Clone(p.Basis.Decision.Decision.Exclude)
	p.Basis.Contract.Components = slices.Clone(p.Basis.Contract.Components)
	p.Basis.Rows = slices.Clone(p.Basis.Rows)
	p.Basis.Contributors = slices.Clone(p.Basis.Contributors)
	return p
}

func copyIssuanceApplication(p financialIssuanceApplication) financialIssuanceApplication {
	p.Configuration = copyConfiguration(p.Configuration)
	p.Source = copyIssuanceSource(p.Source)
	return p
}

func applicationDigest(p financialIssuanceApplication) (string, error) {
	p.Digest = ""
	return financialEvidenceDigest("issuance_application_v1", p)
}

func applicationApplicability(p financialIssuanceApplication) error {
	bad := func() error { return commands.Fail(commands.EvidenceStale) }
	c, s := p.Configuration, p.Source.Basis
	if p.Schema != "hb_issuance_application_v1" || c.Schema != "hb_issuance_configuration_v1" || c.State != "confirmed" || !configurationActive(c) || c.ActivatedAt != c.Active.At {
		return bad()
	}
	kind, err := configurationKind(c.Spec)
	if err != nil || !reflect.DeepEqual(kind, c.Kind) || !reflect.DeepEqual(kind.Languages, c.Spec.Languages) {
		return bad()
	}
	digest, err := configurationDigest(c)
	if err != nil || digest != c.Digest {
		return bad()
	}
	digest, err = financialEvidenceDigest("issuance_source", s)
	if err != nil || digest != p.Source.Digest || s.Schema != "hb_issuance_source_v1" || s.Profile != financialFinalAmountProfile {
		return bad()
	}
	scope := c.Scope
	if c.Workspace != s.Workspace || scope.Workspace.ID != s.Workspace || !reflect.DeepEqual(scope.Contract, s.Contract) || scope.BillingProfile != s.BillingProfile || scope.BillingVersion != s.BillingRevision || scope.Document != s.Document || s.Customer != scope.Contract.CustomerID || s.Customer != scope.BillingProfile.BuyerID || s.Subject.Currency != scope.BillingProfile.Currency || s.Subject.Currency != s.Document.Currency || s.Document.IssuerLegalEntityID != scope.Document.IssuerLegalEntityID {
		return bad()
	}
	return nil
}

func (a *financialIssuance) ownedConfirmedConfiguration(workspace, id, digest string) (financialIssuanceConfiguration, error) {
	p, ok := a.configurations[id]
	if !ok || p.Spec.ID != id || p.Workspace != workspace || p.Digest != digest || p.State != "confirmed" || p.Proposer == "" || p.Confirmer == "" || p.Proposer == p.Confirmer {
		return financialIssuanceConfiguration{}, commands.Fail(commands.EvidenceStale)
	}
	if err := a.configurationIntegrity(workspace, p); err != nil {
		return financialIssuanceConfiguration{}, err
	}
	for _, governor := range []string{p.Proposer, p.Confirmer} {
		if err := a.server.issuanceProfileGovernorStanding(workspace, governor); err != nil {
			return financialIssuanceConfiguration{}, err
		}
	}
	p, err := a.configurationReadback(workspace, p)
	if err != nil {
		return financialIssuanceConfiguration{}, err
	}
	if a.server.clock().Before(p.ActivatedAt) {
		return financialIssuanceConfiguration{}, commands.Fail(commands.EvidenceStale)
	}
	return p, nil
}

func (a *financialIssuance) draftApplication(workspace, actor, capability, configurationID, configurationDigest, finalRequest, finalDigest string) (financialIssuanceApplication, error) {
	if err := a.draftOperation(workspace, actor, capability); err != nil {
		return financialIssuanceApplication{}, err
	}
	c, err := a.ownedConfirmedConfiguration(workspace, configurationID, configurationDigest)
	if err != nil {
		return financialIssuanceApplication{}, err
	}
	source, err := a.server.financialIssuanceSourceLocked(workspace, finalRequest, finalDigest, actor)
	if err != nil {
		return financialIssuanceApplication{}, err
	}
	p := financialIssuanceApplication{Schema: "hb_issuance_application_v1", Configuration: c, Source: source}
	if err := applicationApplicability(p); err != nil {
		return financialIssuanceApplication{}, err
	}
	p.Digest, err = applicationDigest(p)
	if err != nil {
		return financialIssuanceApplication{}, err
	}
	return copyIssuanceApplication(p), nil
}

func issuanceDraftFromApplication(p financialIssuanceApplication, id, language string) (issuance.Draft, error) {
	if err := applicationApplicability(p); err != nil {
		return issuance.Draft{}, err
	}
	digest, err := applicationDigest(p)
	if err != nil || digest != p.Digest {
		return issuance.Draft{}, commands.Fail(commands.EvidenceStale)
	}
	return issuanceDraftFromEvidence(p.Configuration.Kind, p.Configuration.Spec.Mode, p.Configuration.Spec.Profile, p.Source.Basis, id, language)
}
