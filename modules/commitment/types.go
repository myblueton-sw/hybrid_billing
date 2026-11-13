package commitment

import (
	"errors"
	"time"
)

var (
	ErrInvalid   = errors.New("a request or an input is incomplete or malformed")
	ErrNotFound  = errors.New("what the request names does not exist in this workspace")
	ErrConflict  = errors.New("another fact is already recorded under this identifier")
	ErrRejected  = errors.New("a recorded state or rule refuses this request")
	ErrLimit     = errors.New("a configured limit or bound is exceeded")
	ErrSelfCheck = errors.New("a result failed its own check and nothing was returned")
)

const (
	maxIDBytes     = 128
	maxNumberBytes = 64
	periodLabel    = "commitment_period_v1"
	attributionLbl = "commitment_attribution_v1"
	allocPolicyLbl = "commitment_allocation_policy_v1"
	salePolicyLbl  = "commitment_sale_policy_v1"
	AllocationLbl  = "commitment_allocation_v1"
	SaleLbl        = "commitment_sale_v1"
)

const (
	KindReserved  = "reserved_instance"
	KindSavings   = "savings_plan"
	KindCommitted = "committed_use_discount"

	PaymentAll     = "all_upfront"
	PaymentPartial = "partial_upfront"
	PaymentNone    = "no_upfront"

	RecoveryCharged   = "charged"
	RecoveryRecovered = "recovered"

	LineUsed   = "used"
	LineUnused = "unused"

	StateWritten    = "written"
	StateSuperseded = "superseded"
	StateSealed     = "sealed"
	StateHeld       = "held"

	CurrentSealed         = "sealed"
	CurrentCorrectionHeld = "correction_held"
	CurrentNotJudged      = "not_judged"

	ReasonForeignCommitment = "foreign_commitment"
	ReasonDuplicateSource   = "duplicate_source"
	ReasonCostNotPreserved  = "cost_not_preserved"
	ReasonPrepaidExceeded   = "prepaid_exceeded"

	BearMSP          = "msp"
	BearCustomer     = "customer"
	BearProRataUsed  = "pro_rata_used"
	BearUnattributed = "unattributed"

	HoldAccountNotAttributed = "account_not_attributed"
	HoldNoBasis              = "no_basis"

	MethodCostPlus     = "cost_plus"
	MethodSavingsShare = "savings_share"

	HoldNoSalePolicy              = "no_sale_policy"
	HoldNegativeSavings           = "negative_savings"
	HoldRecoveredWithSavingsShare = "recovered_with_savings_share"
)

var (
	kinds      = map[string]bool{KindReserved: true, KindSavings: true, KindCommitted: true}
	payments   = map[string]bool{PaymentAll: true, PaymentPartial: true, PaymentNone: true}
	recoveries = map[string]bool{RecoveryCharged: true, RecoveryRecovered: true}
	bearings   = map[string]bool{BearMSP: true, BearCustomer: true, BearProRataUsed: true, BearUnattributed: true}
	quanta     = map[string]int{"1": 0, "0.1": 1, "0.01": 2, "0.001": 3, "0.0001": 4}
)

type Commitment struct {
	ID           string
	Provider     string
	PayerAccount string
	Kind         string
	Currency     string
	First        string
	Last         string
	Payment      string
	Prepaid      string
	Quantum      string
	Recovery     string
}

type Line struct {
	Source     string
	Kind       string
	Commitment string
	Account    string
	Recurring  string
	Amortized  string
	List       string
}

type Facts struct {
	Recurring string
	Amortized string
	Lines     []Line
}

type RevisionState struct {
	Revision  int
	State     string
	Reasons   []string
	Digest    string
	WrittenAt time.Time
	JudgedAt  time.Time
}

type CurrentView struct {
	State    string
	Revision int
	Digest   string
}

type Mapping struct {
	Account  string
	Customer string
}

type Attribution struct {
	ID       string
	Version  string
	Mappings []Mapping
}

type AllocationPolicy struct {
	ID       string
	Version  string
	Bearing  string
	Customer string
}

type SalePolicy struct {
	Customer    string
	ID          string
	Version     string
	Method      string
	Fee         string
	FeeOnUnused bool
	Share       string
	Mode        string
}

type CustomerPart struct {
	Customer        string
	UsedRecurring   string
	UsedAmortized   string
	List            string
	Savings         string
	NegativeSavings int
	UnusedRecurring string
	UnusedAmortized string
}

type AllocationView struct {
	Label                 string
	ProfileID             string
	ProfileRevision       string
	Commitment            string
	Month                 string
	Revision              int
	PeriodDigest          string
	Recovery              string
	AttributionID         string
	AttributionVersion    string
	AttributionDigest     string
	PolicyID              string
	PolicyVersion         string
	PolicyDigest          string
	Bearing               string
	Hold                  string
	Customers             []CustomerPart
	MSPRecurring          string
	MSPAmortized          string
	RoundingRecurring     string
	RoundingAmortized     string
	UnattributedRecurring string
	UnattributedAmortized string
	Digest                string
}

type Charge struct {
	Customer      string
	PolicyID      string
	PolicyVersion string
	PolicyDigest  string
	Hold          string
	Amount        string
	Exact         string
}

type Sale struct {
	Label              string
	ProfileID          string
	ProfileRevision    string
	AllocationDigest   string
	Charges            []Charge
	Total              string
	CostCompared       string
	Recovered          string
	AllCustomersPriced bool
	MSPResult          string
	BelowCost          bool
	Digest             string
}

type Limits struct {
	Commitments int
	Revisions   int
	Lines       int
	Customers   int
	Digits      int
}

func (l Limits) valid() bool {
	return l.Commitments > 0 && l.Commitments <= 100000 && l.Revisions > 0 && l.Revisions <= 1000 && l.Lines > 0 && l.Lines <= 100000 &&
		l.Customers > 0 && l.Customers <= 10000 && l.Digits > 0 && l.Digits <= 10000
}

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

func validMonth(text string) bool {
	if len(text) != 7 || text[4] != '-' {
		return false
	}
	for _, index := range []int{0, 1, 2, 3, 5, 6} {
		if text[index] < '0' || text[index] > '9' {
			return false
		}
	}
	return text >= "2000-01" && text <= "2099-12" && text[5:] >= "01" && text[5:] <= "12"
}
