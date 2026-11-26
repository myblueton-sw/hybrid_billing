package server

import (
	"reflect"
	"slices"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
)

type financialConfigurationSpec struct {
	ID, Contract, BillingProfile                string
	Revision, ContractRevision, BillingRevision uint64
	Mode, Profile, Region, Tax, Zone            string
	Languages                                   []string
}

type financialConfigurationScope struct {
	Workspace        organizations.Workspace
	WorkspaceVersion uint64
	Contract         organizations.ContractRevision
	BillingProfile   organizations.BillingProfile
	BillingVersion   uint64
	Document         organizations.DocumentKey
	Parties          []financialChannelParty
}

type financialConfigurationSetupContext struct {
	On, ScopeDigest string
}

type financialIssuanceConfiguration struct {
	Schema, Workspace        string
	Spec                     financialConfigurationSpec
	Scope                    financialConfigurationScope
	Kind                     issuance.Kind
	Proposer, Reason, Digest string
	State, Confirmer         string
	Proposed, Active         issuance.KindView
	ActivatedAt              time.Time
	Setup                    *financialConfigurationSetupContext `json:"setup,omitempty"`
}

func configurationKind(spec financialConfigurationSpec) (issuance.Kind, error) {
	if spec.Revision != 1 || !validRequestID(spec.Contract) || spec.ContractRevision == 0 || !validRequestID(spec.BillingProfile) || spec.BillingRevision == 0 {
		return issuance.Kind{}, commands.Fail(commands.InvalidRequest)
	}
	kind, err := issuanceProfileKind(financialIssuanceProfileSpec{ID: spec.ID, Mode: spec.Mode, Profile: spec.Profile, Region: spec.Region, Tax: spec.Tax, Zone: spec.Zone, Languages: spec.Languages})
	if err != nil {
		return issuance.Kind{}, err
	}
	identity, err := financialEvidenceDigest("issuance_configuration_kind_v1", struct {
		ID       string
		Revision uint64
	}{spec.ID, spec.Revision})
	if err != nil {
		return issuance.Kind{}, err
	}
	kind.ID = "configured-kind-" + identity
	return kind, nil
}

func configurationScopeKey(workspace string, spec financialConfigurationSpec) (string, error) {
	return financialEvidenceDigest("issuance_configuration_scope_v1", struct {
		Workspace, Contract, BillingProfile string
		ContractRevision, BillingRevision   uint64
	}{workspace, spec.Contract, spec.BillingProfile, spec.ContractRevision, spec.BillingRevision})
}

func copyConfiguration(p financialIssuanceConfiguration) financialIssuanceConfiguration {
	if p.Setup != nil {
		setup := *p.Setup
		p.Setup = &setup
	}
	p.Spec.Languages = slices.Clone(p.Spec.Languages)
	p.Kind.Languages = slices.Clone(p.Kind.Languages)
	p.Scope.Contract.Components = slices.Clone(p.Scope.Contract.Components)
	p.Scope.Parties = slices.Clone(p.Scope.Parties)
	p.Proposed.Kind.Languages = slices.Clone(p.Proposed.Kind.Languages)
	p.Active.Kind.Languages = slices.Clone(p.Active.Kind.Languages)
	return p
}

func configurationDigest(p financialIssuanceConfiguration) (string, error) {
	p.Digest, p.State, p.Confirmer = "", "", ""
	p.Proposed, p.Active, p.ActivatedAt = issuance.KindView{}, issuance.KindView{}, time.Time{}
	return financialEvidenceDigest("issuance_configuration_v1", p)
}

func (a *financialIssuance) configurationScope(workspace string, spec financialConfigurationSpec) (financialConfigurationScope, error) {
	p := financialConfigurationScope{}
	bad := func() (financialConfigurationScope, error) {
		return financialConfigurationScope{}, commands.Fail(commands.EvidenceStale)
	}
	if workspace != a.server.bootWorkspace {
		return financialConfigurationScope{}, commands.Fail(commands.PermissionDenied)
	}
	var err error
	p.Workspace, p.WorkspaceVersion, err = a.server.orgs.Workspace(workspace)
	if err != nil || p.Workspace.ID != workspace || p.Workspace.State != organizations.Active || p.WorkspaceVersion == 0 {
		return bad()
	}
	p.Contract, err = a.server.orgs.ContractRevision(workspace, spec.Contract, spec.ContractRevision)
	if err != nil || p.Contract.WorkspaceID != workspace || p.Contract.ContractID != spec.Contract || p.Contract.Revision != spec.ContractRevision {
		return bad()
	}
	p.BillingProfile, p.BillingVersion, err = a.server.orgs.BillingProfile(workspace, spec.BillingProfile)
	b := p.BillingProfile
	if err != nil || p.BillingVersion != spec.BillingRevision || b.ID != spec.BillingProfile || b.WorkspaceID != workspace || b.State != organizations.Active || b.Path != organizations.ExternalInvoice || p.Contract.CustomerID != b.BuyerID || p.Contract.SellerID != b.IssuerID || (len(b.Currency) != 3 || b.Currency[0] < 'A' || b.Currency[0] > 'Z' || b.Currency[1] < 'A' || b.Currency[1] > 'Z' || b.Currency[2] < 'A' || b.Currency[2] > 'Z') {
		return bad()
	}
	parties := map[string]financialChannelParty{}
	for _, id := range []string{p.Contract.CustomerID, p.Contract.SellerID, b.IssuerID, b.BuyerID, b.PayerID, b.RecipientID} {
		if id == "" {
			continue
		}
		snapshots, _, err := a.channelParty(workspace, id)
		if err != nil {
			return bad()
		}
		for _, snapshot := range snapshots {
			parties[snapshot.Party.ID] = snapshot
		}
	}
	for _, snapshot := range parties {
		p.Parties = append(p.Parties, snapshot)
	}
	slices.SortFunc(p.Parties, func(a, b financialChannelParty) int {
		if a.Party.ID < b.Party.ID {
			return -1
		}
		if a.Party.ID > b.Party.ID {
			return 1
		}
		return 0
	})
	p.Document, err = a.server.orgs.DocumentKey(workspace, spec.BillingProfile)
	if err != nil || p.Document.WorkspaceID != workspace || p.Document.Path != organizations.ExternalInvoice || p.Document.Currency != b.Currency {
		return bad()
	}
	return p, nil
}

func (a *financialIssuance) configurationIntegrity(workspace string, p financialIssuanceConfiguration) error {
	bad := func() error { return commands.Fail(commands.EvidenceStale) }
	if !validConfigurationSetup(p.Setup) || p.Schema != "hb_issuance_configuration_v1" || p.Workspace != workspace || p.Proposer == "" || !documentReason(p.Reason) {
		return bad()
	}
	kind, err := configurationKind(p.Spec)
	if err != nil || !reflect.DeepEqual(kind, p.Kind) || !reflect.DeepEqual(kind.Languages, p.Spec.Languages) {
		return bad()
	}
	digest, err := configurationDigest(p)
	if err != nil || digest != p.Digest {
		return bad()
	}
	key, err := configurationScopeKey(workspace, p.Spec)
	if err != nil || a.configurationScopes[key] != p.Spec.ID {
		return bad()
	}
	count := 0
	for k, id := range a.configurationScopes {
		if id == p.Spec.ID {
			count++
			if k != key {
				return bad()
			}
		}
	}
	if count != 1 {
		return bad()
	}
	for id, other := range a.configurations {
		otherKey, err := configurationScopeKey(other.Workspace, other.Spec)
		if err != nil || (otherKey == key && id != p.Spec.ID) || (other.Spec.ID == p.Spec.ID && id != p.Spec.ID) {
			return bad()
		}
	}
	scope, err := a.configurationScope(workspace, p.Spec)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(scope, p.Scope) {
		return bad()
	}
	return nil
}

func (a *financialIssuance) currentConfiguration(workspace string, p financialIssuanceConfiguration) error {
	if err := a.configurationIntegrity(workspace, p); err != nil {
		return err
	}
	for _, actor := range []string{p.Proposer, p.Confirmer} {
		if actor != "" {
			if _, err := a.server.governor(workspace, actor); err != nil {
				return err
			}
		}
	}
	return nil
}
