package server

import (
	"errors"
	"reflect"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
)

func frozenConfigurationScope(p financialIssuanceConfiguration) bool {
	s, b, c := p.Scope, p.Scope.BillingProfile, p.Scope.Contract
	if s.Workspace.ID != p.Workspace || s.Workspace.State != organizations.Active || s.WorkspaceVersion == 0 || s.BillingVersion != p.Spec.BillingRevision || s.BillingVersion == 0 || b.ID != p.Spec.BillingProfile || b.WorkspaceID != p.Workspace || b.State != organizations.Active || b.Path != organizations.ExternalInvoice || c.ContractID != p.Spec.Contract || c.WorkspaceID != p.Workspace || c.Revision != p.Spec.ContractRevision || c.Revision == 0 || c.CustomerID != b.BuyerID || c.SellerID != b.IssuerID || len(s.Parties) == 0 || len(s.Parties) > 12 {
		return false
	}
	frozen, err := organizations.NewRegistry("retained-configuration-validation")
	if err != nil {
		return false
	}
	if _, err = frozen.PutWorkspace(s.Workspace, 0); err != nil {
		return false
	}
	parties := map[string]organizations.Party{}
	for i, snapshot := range s.Parties {
		party := snapshot.Party
		if snapshot.Version == 0 || party.WorkspaceID != p.Workspace || party.State != organizations.Active || i > 0 && s.Parties[i-1].Party.ID >= party.ID {
			return false
		}
		parties[party.ID] = party
	}
	for _, kind := range []organizations.PartyKind{organizations.LegalEntity, organizations.InternalUnit} {
		for _, snapshot := range s.Parties {
			if snapshot.Party.Kind == kind {
				if _, err = frozen.PutParty(snapshot.Party, 0); err != nil {
					return false
				}
			}
		}
	}
	required := map[string]bool{}
	for _, id := range []string{c.CustomerID, c.SellerID, b.BuyerID, b.IssuerID, b.PayerID, b.RecipientID} {
		if id == "" {
			continue
		}
		party, ok := parties[id]
		if !ok {
			return false
		}
		required[id] = true
		if party.ParentID != "" {
			required[party.ParentID] = true
		}
	}
	if len(required) != len(parties) {
		return false
	}
	if _, err = frozen.PutBillingProfile(b, 0); err != nil {
		return false
	}
	c.Revision = 1
	if err = frozen.AppendContractRevision(c); err != nil {
		return false
	}
	key, err := frozen.DocumentKey(p.Workspace, b.ID)
	return err == nil && key == s.Document
}

func (a *financialIssuance) configurationHistoryIndex() error {
	bad := func() error { return commands.Fail(commands.EvidenceStale) }
	if len(a.configurations) > financialIssuanceLimit || len(a.configurationScopes) > financialIssuanceLimit {
		return commands.Fail(commands.PrerequisiteFailed)
	}
	if len(a.configurations) != len(a.configurationScopes) {
		return bad()
	}
	seen := map[string]bool{}
	for key, id := range a.configurationScopes {
		p, ok := a.configurations[id]
		actual, err := configurationScopeKey(p.Workspace, p.Spec)
		if !ok || err != nil || actual != key || p.Spec.ID != id || seen[id] {
			return bad()
		}
		seen[id] = true
	}
	for id, p := range a.configurations {
		if !seen[id] || !validRequestID(id) || !validRequestID(p.Workspace) || !validRequestID(p.Spec.Contract) {
			return bad()
		}
	}
	return nil
}

func (a *financialIssuance) observeConfigurationHistory(p financialIssuanceConfiguration) (string, error) {
	bad := func() (string, error) { return "", commands.Fail(commands.EvidenceStale) }
	if !validConfigurationSetup(p.Setup) || p.Schema != "hb_issuance_configuration_v1" || !validRequestID(p.Proposer) || !documentReason(p.Reason) || !frozenConfigurationScope(p) {
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
	emptyProposed := reflect.DeepEqual(p.Proposed, issuance.KindView{})
	emptyActive := reflect.DeepEqual(p.Active, issuance.KindView{})
	switch p.State {
	case "proposal_unknown":
		if p.Confirmer != "" || !emptyActive || !p.ActivatedAt.IsZero() || !emptyProposed && !configurationProposed(p) {
			return bad()
		}
	case "proposed":
		if !configurationProposed(p) || p.Confirmer != "" || !emptyActive || !p.ActivatedAt.IsZero() {
			return bad()
		}
	case "confirmation_unknown":
		if !configurationProposed(p) || !validRequestID(p.Confirmer) || p.Confirmer == p.Proposer || !p.ActivatedAt.IsZero() || !emptyActive && !configurationActive(p) {
			return bad()
		}
	case "confirmed":
		if !validRequestID(p.Confirmer) || !configurationActive(p) || p.ActivatedAt != p.Active.At {
			return bad()
		}
	default:
		return bad()
	}
	if a.documents == nil {
		return "", commands.Fail(commands.DependencyUnavailable)
	}
	actual, err := a.configurationLookup(p.Workspace, p.Kind.ID)
	if emptyProposed {
		if errors.Is(err, issuance.ErrNotFound) {
			return "absent", nil
		}
		if err != nil {
			return "", commands.Fail(commands.DependencyUnavailable)
		}
		return bad()
	}
	if errors.Is(err, issuance.ErrNotFound) {
		return bad()
	}
	if err != nil {
		return "", commands.Fail(commands.DependencyUnavailable)
	}
	expected := p.Proposed
	observation := "proposed"
	if !emptyActive {
		expected = p.Active
		observation = "active"
	}
	if !reflect.DeepEqual(actual, expected) {
		return bad()
	}
	return observation, nil
}
