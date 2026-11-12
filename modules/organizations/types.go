package organizations

import "errors"

var (
	ErrInvalid               = errors.New("record is incomplete or inconsistent")
	ErrNotFound              = errors.New("record does not exist in this workspace")
	ErrConflict              = errors.New("record version conflict")
	ErrSuspended             = errors.New("record refers to a suspended workspace or party")
	ErrIntercompanyUndecided = errors.New("settlement between legal entities without an invoice is not decided")
	ErrMixedDocument         = errors.New("lines do not belong to one billing document")
)

type OperatingProfile string

const (
	ProfileMSP              OperatingProfile = "msp"
	ProfileSingleEnterprise OperatingProfile = "single_enterprise"
	ProfileGroup            OperatingProfile = "group"
)

type State string

const (
	Active    State = "active"
	Suspended State = "suspended"
)

type PartyKind string

const (
	LegalEntity  PartyKind = "legal_entity"
	InternalUnit PartyKind = "internal_unit"
)

type IssuancePath string

const (
	ExternalInvoice    IssuancePath = "external_invoice"
	InternalChargeback IssuancePath = "internal_chargeback"
)

type TransactionPath string

const (
	Direct TransactionPath = "direct"
	Resale TransactionPath = "resale"
)

type ComponentKind string

const (
	ProviderCharge ComponentKind = "provider_charge"
	SellerService  ComponentKind = "seller_service"
)

type Workspace struct {
	ID      string
	Profile OperatingProfile
	State   State
}

type Party struct {
	ID          string
	WorkspaceID string
	Kind        PartyKind
	ParentID    string
	State       State
}

type BillingProfile struct {
	ID          string
	WorkspaceID string
	IssuerID    string
	BuyerID     string
	PayerID     string
	RecipientID string
	Path        IssuancePath
	Currency    string
	State       State
}

type Component struct {
	ID                           string
	Kind                         ComponentKind
	ProviderPaymentResponsibleID string
	SellerBillable               bool
}

type ContractRevision struct {
	ContractID  string
	Revision    uint64
	WorkspaceID string
	ProviderRef string
	CustomerID  string
	SellerID    string
	Path        TransactionPath
	Components  []Component
}

type Reason string

const (
	EligiblePendingApproval Reason = "eligible_pending_approval"
	NotSeller               Reason = "not_seller"
	NoPaymentResponsibility Reason = "no_payment_responsibility"
	OutsideEligibleScope    Reason = "outside_eligible_scope"
	UnknownComponent        Reason = "unknown_component"
	InvalidContract         Reason = "invalid_contract"
)

type Decision struct {
	Eligible bool
	Reason   Reason
}

type DocumentKey struct {
	WorkspaceID         string
	IssuerLegalEntityID string
	BuyerLegalEntityID  string
	PayerID             string
	Currency            string
	Path                IssuancePath
}

const maxIDBytes = 128

func validID(id string) bool {
	if id == "" || len(id) > maxIDBytes {
		return false
	}
	for index := 0; index < len(id); index++ {
		c := id[index]
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == ':' || c == '-') {
			return false
		}
	}
	return true
}

func validCurrency(code string) bool {
	if len(code) != 3 {
		return false
	}
	for index := 0; index < 3; index++ {
		if code[index] < 'A' || code[index] > 'Z' {
			return false
		}
	}
	return true
}

func validState(state State) bool { return state == Active || state == Suspended }
