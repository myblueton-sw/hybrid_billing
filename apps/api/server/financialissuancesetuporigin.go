package server

import (
	"reflect"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
)

func validSetupReference(id, digest string, r *financialConfigurationReference) bool {
	if r != nil {
		return id == "" && digest == "" && validConfigurationReference(r)
	}
	return validRequestID(id) && validPlanHash(digest)
}

func copyConfigurationReference(r *financialConfigurationReference) *financialConfigurationReference {
	if r == nil {
		return nil
	}
	v := *r
	return &v
}

func copyConfigurationSnapshot(c *financialIssuanceConfiguration) *financialIssuanceConfiguration {
	if c == nil {
		return nil
	}
	v := copyConfiguration(*c)
	return &v
}

func setupOriginShape(id, digest string, r *financialConfigurationReference, p financialIssuanceProfile, c *financialIssuanceConfiguration) error {
	bad := func() error { return commands.Fail(commands.EvidenceStale) }
	if !validSetupReference(id, digest, r) {
		return bad()
	}
	if r == nil {
		if c != nil || p.Spec.ID != id || p.Digest != digest {
			return bad()
		}
		return nil
	}
	if c == nil || !reflect.DeepEqual(p, financialIssuanceProfile{}) || configurationReference(*c) != *r || c.Schema != "hb_issuance_configuration_v1" || c.State != "confirmed" || !configurationActive(*c) || c.ActivatedAt != c.Active.At {
		return bad()
	}
	expected, err := configurationDigest(*c)
	if err != nil || expected != c.Digest {
		return bad()
	}
	kind, err := configurationKind(c.Spec)
	if err != nil || !reflect.DeepEqual(kind, c.Kind) || !reflect.DeepEqual(kind.Languages, c.Spec.Languages) {
		return bad()
	}
	return nil
}

func (a *financialIssuance) setupOrigin(workspace, id, digest string, r *financialConfigurationReference) (financialIssuanceProfile, *financialIssuanceConfiguration, error) {
	if !validSetupReference(id, digest, r) {
		return financialIssuanceProfile{}, nil, commands.Fail(commands.InvalidRequest)
	}
	if r == nil {
		p, err := a.ownedConfirmedProfile(workspace, id, digest)
		return p, nil, err
	}
	c, err := a.ownedConfirmedConfiguration(workspace, r.ID, r.Digest)
	if err != nil {
		return financialIssuanceProfile{}, nil, err
	}
	return financialIssuanceProfile{}, &c, nil
}

func (a *financialIssuance) freshSetupOrigin(workspace, id, digest, actor string, r *financialConfigurationReference) (financialIssuanceProfile, *financialIssuanceConfiguration, error) {
	p, c, err := a.setupOrigin(workspace, id, digest, r)
	if err != nil {
		return financialIssuanceProfile{}, nil, err
	}
	if c == nil {
		p, err = a.profileCurrentFinancialSource(workspace, actor, p)
	}
	return p, c, err
}

func (a *financialIssuance) retainedSetupOrigin(workspace, id, digest string, r *financialConfigurationReference, p financialIssuanceProfile, c *financialIssuanceConfiguration) error {
	if err := setupOriginShape(id, digest, r, p, c); err != nil {
		return err
	}
	actual, configured, err := a.setupOrigin(workspace, id, digest, r)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(actual, p) || !reflect.DeepEqual(configured, c) {
		return commands.Fail(commands.EvidenceStale)
	}
	return nil
}

func (a *financialIssuance) configuredChangeSubject(c financialIssuanceConfiguration, field string) (financialChannelSubject, error) {
	var sel financialChannelSelector
	switch field {
	case protected.FieldDocumentRecipient:
		sel = financialChannelSelector{Kind: "customer", Contract: c.Scope.Contract.ContractID, Revision: c.Scope.Contract.Revision}
	case protected.FieldPaymentAccount:
		sel = financialChannelSelector{Kind: "entity", BillingProfile: c.Scope.BillingProfile.ID}
	case protected.FieldDocumentTemplate:
		r := configurationReference(c)
		sel = financialChannelSelector{Kind: "configuration_kind", Configuration: &r}
	default:
		return financialChannelSubject{}, commands.Fail(commands.InvalidRequest)
	}
	p, err := a.channelSubject(c.Workspace, sel)
	if err != nil {
		return financialChannelSubject{}, err
	}
	bad := func() (financialChannelSubject, error) {
		return financialChannelSubject{}, commands.Fail(commands.EvidenceStale)
	}
	if p.Workspace != c.Scope.Workspace || p.WorkspaceVersion != c.Scope.WorkspaceVersion {
		return bad()
	}
	switch field {
	case protected.FieldDocumentRecipient:
		if !reflect.DeepEqual(p.Contract, c.Scope.Contract) || p.Subject != "customer:"+c.Scope.Contract.CustomerID {
			return bad()
		}
	case protected.FieldPaymentAccount:
		if p.BillingProfile != c.Scope.BillingProfile || p.BillingVersion != c.Scope.BillingVersion || p.Subject != "entity:"+c.Scope.Document.IssuerLegalEntityID {
			return bad()
		}
	case protected.FieldDocumentTemplate:
		if !reflect.DeepEqual(p.Configuration, &c) || !reflect.DeepEqual(p.ReservedKind, c.Kind) {
			return bad()
		}
	}
	return p, nil
}

func (a *financialIssuance) setupChangeSubject(p financialIssuanceProfile, c *financialIssuanceConfiguration, field string) (financialChannelSubject, error) {
	if c != nil {
		return a.configuredChangeSubject(*c, field)
	}
	return a.changeSubject(p, field)
}
