package organizations

import (
	"math"
	"sync"
)

type scoped struct {
	workspace string
	id        string
}

type versioned[T any] struct {
	value   T
	version uint64
}

type Registry struct {
	mu           sync.RWMutex
	installation string
	workspaces   map[string]versioned[Workspace]
	parties      map[scoped]versioned[Party]
	profiles     map[scoped]versioned[BillingProfile]
	contracts    map[scoped][]ContractRevision
}

func NewRegistry(installationID string) (*Registry, error) {
	if !validID(installationID) {
		return nil, ErrInvalid
	}
	return &Registry{
		installation: installationID,
		workspaces:   make(map[string]versioned[Workspace]),
		parties:      make(map[scoped]versioned[Party]),
		profiles:     make(map[scoped]versioned[BillingProfile]),
		contracts:    make(map[scoped][]ContractRevision),
	}, nil
}

func (r *Registry) InstallationID() string { return r.installation }

func next(current, expected uint64) (uint64, error) {
	if current != expected || current == math.MaxUint64 {
		return 0, ErrConflict
	}
	return current + 1, nil
}

func (r *Registry) PutWorkspace(workspace Workspace, expected uint64) (uint64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !validID(workspace.ID) || !validState(workspace.State) ||
		(workspace.Profile != ProfileMSP && workspace.Profile != ProfileSingleEnterprise && workspace.Profile != ProfileGroup) {
		return 0, ErrInvalid
	}
	version, err := next(r.workspaces[workspace.ID].version, expected)
	if err != nil {
		return 0, err
	}
	r.workspaces[workspace.ID] = versioned[Workspace]{workspace, version}
	return version, nil
}

func (r *Registry) Workspace(id string) (Workspace, uint64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	found := r.workspaces[id]
	if found.version == 0 {
		return Workspace{}, 0, ErrNotFound
	}
	return found.value, found.version, nil
}

func (r *Registry) writable(workspaceID string) error {
	workspace := r.workspaces[workspaceID]
	if workspace.version == 0 {
		return ErrNotFound
	}
	if workspace.value.State != Active {
		return ErrSuspended
	}
	return nil
}

func (r *Registry) activeParty(workspaceID, id string) (Party, error) {
	if !validID(id) {
		return Party{}, ErrInvalid
	}
	party := r.parties[scoped{workspaceID, id}]
	if party.version == 0 {
		return Party{}, ErrNotFound
	}
	if party.value.State != Active {
		return Party{}, ErrSuspended
	}
	if party.value.Kind == InternalUnit && r.parties[scoped{workspaceID, party.value.ParentID}].value.State != Active {
		return Party{}, ErrSuspended
	}
	return party.value, nil
}

func (r *Registry) PutParty(party Party, expected uint64) (uint64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !validID(party.ID) || !validID(party.WorkspaceID) || !validState(party.State) {
		return 0, ErrInvalid
	}
	if err := r.writable(party.WorkspaceID); err != nil {
		return 0, err
	}
	key := scoped{party.WorkspaceID, party.ID}
	old := r.parties[key]
	if suspension := old.value; old.version != 0 && party.State == Suspended {
		suspension.State = Suspended
		if suspension == party {
			version, err := next(old.version, expected)
			if err != nil {
				return 0, err
			}
			r.parties[key] = versioned[Party]{party, version}
			return version, nil
		}
	}
	switch party.Kind {
	case LegalEntity:
		if party.ParentID != "" {
			return 0, ErrInvalid
		}
	case InternalUnit:
		if party.ParentID == party.ID {
			return 0, ErrInvalid
		}
		parent, err := r.activeParty(party.WorkspaceID, party.ParentID)
		if err != nil {
			return 0, err
		}
		if parent.Kind != LegalEntity {
			return 0, ErrInvalid
		}
	default:
		return 0, ErrInvalid
	}
	version, err := next(old.version, expected)
	if err != nil {
		return 0, err
	}
	if old.version != 0 && (old.value.Kind != party.Kind || old.value.ParentID != party.ParentID) {
		return 0, ErrInvalid
	}
	r.parties[key] = versioned[Party]{party, version}
	return version, nil
}

func (r *Registry) Party(workspaceID, id string) (Party, uint64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	found := r.parties[scoped{workspaceID, id}]
	if found.version == 0 {
		return Party{}, 0, ErrNotFound
	}
	return found.value, found.version, nil
}

func legalEntityOf(party Party) string {
	if party.Kind == InternalUnit {
		return party.ParentID
	}
	return party.ID
}

func (r *Registry) LegalEntityOf(workspaceID, partyID string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	party := r.parties[scoped{workspaceID, partyID}]
	if party.version == 0 {
		return "", ErrNotFound
	}
	return legalEntityOf(party.value), nil
}

func (r *Registry) PutBillingProfile(profile BillingProfile, expected uint64) (uint64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !validID(profile.ID) || !validID(profile.WorkspaceID) || !validState(profile.State) || !validCurrency(profile.Currency) {
		return 0, ErrInvalid
	}
	if err := r.writable(profile.WorkspaceID); err != nil {
		return 0, err
	}
	key := scoped{profile.WorkspaceID, profile.ID}
	old := r.profiles[key]
	if old.version != 0 {
		frozen := old.value
		frozen.State = profile.State
		if frozen != profile {
			return 0, ErrInvalid
		}
		if profile.State == Suspended {
			version, err := next(old.version, expected)
			if err != nil {
				return 0, err
			}
			r.profiles[key] = versioned[BillingProfile]{profile, version}
			return version, nil
		}
	}
	issuer, err := r.activeParty(profile.WorkspaceID, profile.IssuerID)
	if err != nil {
		return 0, err
	}
	buyer, err := r.activeParty(profile.WorkspaceID, profile.BuyerID)
	if err != nil {
		return 0, err
	}
	if issuer.ID == buyer.ID {
		return 0, ErrInvalid
	}
	switch profile.Path {
	case ExternalInvoice:
		payer, err := r.activeParty(profile.WorkspaceID, profile.PayerID)
		if err != nil {
			return 0, err
		}
		if issuer.Kind != LegalEntity || buyer.Kind != LegalEntity || payer.Kind != LegalEntity || payer.ID == issuer.ID {
			return 0, ErrInvalid
		}
		if profile.RecipientID != "" {
			recipient, err := r.activeParty(profile.WorkspaceID, profile.RecipientID)
			if err != nil {
				return 0, err
			}
			if recipient.Kind != LegalEntity || recipient.ID == issuer.ID {
				return 0, ErrInvalid
			}
		}
	case InternalChargeback:
		if profile.PayerID != "" || profile.RecipientID != "" {
			return 0, ErrInvalid
		}
		if legalEntityOf(issuer) != legalEntityOf(buyer) {
			return 0, ErrIntercompanyUndecided
		}
	default:
		return 0, ErrInvalid
	}
	version, err := next(old.version, expected)
	if err != nil {
		return 0, err
	}
	r.profiles[key] = versioned[BillingProfile]{profile, version}
	return version, nil
}

func (r *Registry) BillingProfile(workspaceID, id string) (BillingProfile, uint64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	found := r.profiles[scoped{workspaceID, id}]
	if found.version == 0 {
		return BillingProfile{}, 0, ErrNotFound
	}
	return found.value, found.version, nil
}

func (r *Registry) DocumentKey(workspaceID, profileID string) (DocumentKey, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	found := r.profiles[scoped{workspaceID, profileID}]
	if found.version == 0 {
		return DocumentKey{}, ErrNotFound
	}
	profile := found.value
	return DocumentKey{
		WorkspaceID:         workspaceID,
		IssuerLegalEntityID: legalEntityOf(r.parties[scoped{workspaceID, profile.IssuerID}].value),
		BuyerLegalEntityID:  legalEntityOf(r.parties[scoped{workspaceID, profile.BuyerID}].value),
		PayerID:             profile.PayerID,
		Currency:            profile.Currency,
		Path:                profile.Path,
	}, nil
}

func (r *Registry) AppendContractRevision(revision ContractRevision) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !validID(revision.ContractID) || !validID(revision.WorkspaceID) || !validID(revision.ProviderRef) ||
		(revision.Path != Direct && revision.Path != Resale) || len(revision.Components) == 0 {
		return ErrInvalid
	}
	if err := r.writable(revision.WorkspaceID); err != nil {
		return err
	}
	customer, err := r.activeParty(revision.WorkspaceID, revision.CustomerID)
	if err != nil {
		return err
	}
	if customer.Kind != LegalEntity {
		return ErrInvalid
	}
	if revision.SellerID != "" {
		seller, err := r.activeParty(revision.WorkspaceID, revision.SellerID)
		if err != nil {
			return err
		}
		if seller.Kind != LegalEntity || seller.ID == customer.ID {
			return ErrInvalid
		}
	}
	if revision.Path == Resale && revision.SellerID == "" {
		return ErrInvalid
	}
	seen := make(map[string]struct{}, len(revision.Components))
	for _, component := range revision.Components {
		if !validID(component.ID) {
			return ErrInvalid
		}
		if _, repeated := seen[component.ID]; repeated {
			return ErrInvalid
		}
		seen[component.ID] = struct{}{}
		switch component.Kind {
		case ProviderCharge:
			responsible := component.ProviderPaymentResponsibleID
			if responsible == "" || (responsible != revision.CustomerID && responsible != revision.SellerID) {
				return ErrInvalid
			}
			if revision.Path == Direct && (responsible != revision.CustomerID || component.SellerBillable) {
				return ErrInvalid
			}
			if component.SellerBillable && responsible != revision.SellerID {
				return ErrInvalid
			}
		case SellerService:
			if component.ProviderPaymentResponsibleID != "" || revision.SellerID == "" {
				return ErrInvalid
			}
		default:
			return ErrInvalid
		}
	}
	key := scoped{revision.WorkspaceID, revision.ContractID}
	history := r.contracts[key]
	if len(history) > 0 && (history[0].CustomerID != revision.CustomerID || history[0].ProviderRef != revision.ProviderRef) {
		return ErrInvalid
	}
	if revision.Revision != uint64(len(history))+1 {
		return ErrConflict
	}
	stored := revision
	stored.Components = append([]Component(nil), revision.Components...)
	r.contracts[key] = append(history, stored)
	return nil
}

func (r *Registry) ContractRevision(workspaceID, contractID string, revision uint64) (ContractRevision, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	history := r.contracts[scoped{workspaceID, contractID}]
	if revision == 0 || revision > uint64(len(history)) {
		return ContractRevision{}, ErrNotFound
	}
	found := history[revision-1]
	found.Components = append([]Component(nil), found.Components...)
	return found, nil
}
