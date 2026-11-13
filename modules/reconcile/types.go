package reconcile

import (
	"errors"
	"time"
)

var (
	ErrInvalid  = errors.New("request is incomplete or malformed")
	ErrNotFound = errors.New("run, document or subject does not exist in this workspace")
	ErrConflict = errors.New("request conflicts with what is stored")
	ErrRejected = errors.New("the stored state or a rule does not allow this step")
	ErrLimit    = errors.New("a configured limit is exceeded")
)

const (
	maxIDBytes  = 128
	maxRevision = 999999
)

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
	return len(code) == 3 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= 'Z' && code[2] >= 'A' && code[2] <= 'Z'
}

type Basis string

const (
	ProviderDocument Basis = "provider_document"
	IndependentPrice Basis = "independent_price"
	InternalCost     Basis = "internal_cost"
)

type RunSpec struct {
	WorkspaceID        string
	ID                 string
	Basis              Basis
	Issuer             string
	Account            string
	Document           string
	Revision           int
	Currency           string
	Total              string
	Input              string
	Parts              int
	Rounding           string
	Customer           string
	Period             string
	Charge             string
	Pool               string
	QuantityComplete   bool
	QuantityEvidence   string
	PriceEvidence      string
	Cost               string
	CostEvidence       string
	PoolEvidence       string
	AllocationEvidence string
}

type Subject struct {
	Basis    Basis
	Issuer   string
	Account  string
	Document string
	Revision int
	Currency string
	Customer string
	Period   string
	Charge   string
	Pool     string
}

type CostKind string

const (
	Billed     CostKind = "billed"
	Effective  CostKind = "effective"
	List       CostKind = "list"
	Contracted CostKind = "contracted"
	Sale       CostKind = "sale"
)

func (k CostKind) known() bool {
	return k == Billed || k == Effective || k == List || k == Contracted || k == Sale
}

type Part struct {
	ID       string
	Amount   string
	Currency string
	Kind     CostKind
	Source   string
}

type Category string

const (
	Tax                Category = "tax"
	Support            Category = "support"
	Credit             Category = "credit"
	CommitmentPurchase Category = "commitment_purchase"
	Marketplace        Category = "marketplace"
	Rounding           Category = "rounding"
)

var categories = []Category{Tax, Support, Credit, CommitmentPurchase, Marketplace, Rounding}

func (c Category) known() bool {
	for _, listed := range categories {
		if c == listed {
			return true
		}
	}
	return false
}

type ItemType string

const (
	DocumentLine ItemType = "document_line"
	InternalItem ItemType = "internal_item"
)

type Line struct {
	ID       string
	Category Category
	Amount   string
	Currency string
	ItemType ItemType
	Item     string
	Evidence string
}

type State string

const (
	Open   State = "open"
	Closed State = "closed"
	Voided State = "voided"
)

type VoidReason string

const (
	VoidSuperseded VoidReason = "superseded"
	WrongFigures   VoidReason = "wrong_figures"
	VoidedBy       VoidReason = "operator"
)

func (r VoidReason) known() bool {
	return r == VoidSuperseded || r == WrongFigures || r == VoidedBy
}

const (
	Explained   = "explained"
	Unexplained = "unexplained"
	Ready       = "ready"
	NotReady    = "not_ready"

	NotClosed          = "not_closed"
	UnexplainedResidue = "unexplained_residual"
	Superseded         = "superseded"
	SumOutsideProfile  = "sum_outside_profile"
	QuantityIncomplete = "quantity_incomplete"
	IsVoided           = "voided"

	Determined   = "determined"
	Declared     = "declared"
	Undetermined = "undetermined"
)

type CategorySum struct {
	Category Category
	Lines    int
	Sum      string
}

type RunView struct {
	Spec       RunSpec
	State      State
	WasClosed  bool
	Superseded bool
	Posted     int
	Lines      int
	Internal   string
	Categories []CategorySum
	Explained  string
	Residual   string
	Verdict    string
	VoidReason VoidReason
	Opened     time.Time
	Changed    time.Time
}

type Readiness struct {
	Verdict string
	Reasons []string
	Cost    string
}

type Listed struct {
	Total int
	Items []string
}

type CurrentRun struct {
	ID       string
	Currency string
	State    State
	Verdict  string
}

type CurrentView struct {
	Revision   int
	Runs       []CurrentRun
	Superseded Listed
}

type Limits struct {
	Runs   int
	Parts  int
	Lines  int
	Page   int
	Listed int
}

func (l Limits) valid() bool {
	return l.Runs > 0 && l.Parts > 0 && l.Lines > 0 && l.Page > 0 && l.Listed > 0
}
