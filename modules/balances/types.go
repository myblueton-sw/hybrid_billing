package balances

import (
	"errors"
	"time"
)

var (
	ErrInvalid  = errors.New("a request or an input is incomplete or malformed")
	ErrNotFound = errors.New("what the request names does not exist in this workspace")
	ErrConflict = errors.New("another fact is already recorded under this identifier")
	ErrRejected = errors.New("a recorded state or rule refuses this request")
	ErrLimit    = errors.New("a configured limit or bound is exceeded")
)

const (
	maxIDBytes       = 128
	maxNumberBytes   = 64
	maxLifetime      = 86400
	accountLabel     = "balance_account_v1"
	openingLabel     = "balance_opening_v1"
	increaseLabel    = "balance_increase_v1"
	applicationLabel = "balance_application_v1"
	reversalLabel    = "balance_reversal_v1"
)

const (
	KindCash   = "cash_prepayment"
	KindCredit = "credit"

	AuthorityPlatform = "platform"
	AuthorityExternal = "external"

	IncreaseCashReceipt    = "cash_receipt"
	IncreaseCreditDocument = "credit_document"
	IncreasePromotion      = "promotion"
	IncreaseSLACredit      = "sla_credit"

	StatePending   = "pending"
	StateApproved  = "approved"
	StateWithdrawn = "withdrawn"
	StateEffective = "effective"

	StateReserved  = "reserved"
	StateConfirmed = "confirmed"
	StateReleased  = "released"
	StateExpired   = "expired"

	ReverseApplication = "application"
	ReverseIncrease    = "increase"
	ReverseOpening     = "opening"

	ReasonExternalAuthority = "external_authority"
	ReasonOpeningPending    = "opening_pending"
	ReasonCustomerMismatch  = "customer_mismatch"
	ReasonCurrencyMismatch  = "currency_mismatch"
	ReasonTargetMismatch    = "target_mismatch"
	ReasonTargetSettled     = "target_settled"
	ReasonExceedsTarget     = "exceeds_target"
	ReasonNothingAvailable  = "nothing_available"
	ReasonInsufficient      = "insufficient"
)

var (
	balanceKinds  = map[string]bool{KindCash: true, KindCredit: true}
	authorities   = map[string]bool{AuthorityPlatform: true, AuthorityExternal: true}
	increaseKinds = map[string]string{IncreaseCashReceipt: KindCash, IncreaseCreditDocument: KindCredit, IncreasePromotion: KindCredit, IncreaseSLACredit: KindCredit}
	reverseKinds  = map[string]bool{ReverseApplication: true, ReverseIncrease: true, ReverseOpening: true}
)

type Account struct {
	ID        string
	Customer  string
	Entity    string
	Currency  string
	Kind      string
	Authority string
	Ledger    string
}

type AccountView struct {
	Account   Account
	Author    string
	Digest    string
	WrittenAt time.Time
}

type Opening struct {
	ID           string
	Account      string
	PriorLedger  string
	AsOf         string
	Amount       string
	Evidence     string
	MigrationKey string
	IncludedFrom string
	ExpiresOn    string
}

type OpeningView struct {
	Opening     Opening
	State       string
	Author      string
	Approver    string
	WithdrawnBy string
	Digest      string
	WrittenAt   time.Time
	ApprovedAt  time.Time
	WithdrawnAt time.Time
}

type Import struct {
	Ledger      string
	Transaction string
	Occurred    string
}

type Increase struct {
	ID        string
	Account   string
	Kind      string
	Amount    string
	Reference string
	ExpiresOn string
	Import    Import
}

type IncreaseView struct {
	Increase    Increase
	State       string
	Author      string
	Approver    string
	WithdrawnBy string
	Digest      string
	WrittenAt   time.Time
	ApprovedAt  time.Time
	WithdrawnAt time.Time
}

type Target struct {
	Document string
	Line     string
	Amount   string
}

type Application struct {
	ID        string
	Account   string
	Target    Target
	Requested string
	Partial   bool
	Lifetime  int
	Import    Import
}

type ApplicationView struct {
	Application Application
	State       string
	Applied     string
	Reversed    string
	Expiry      time.Time
	ReservedBy  string
	ReservedAt  time.Time
	ConfirmedBy string
	ConfirmedAt time.Time
	ReleasedBy  string
	ReleasedAt  time.Time
	Digest      string
}

type Decision struct {
	Reason      string
	Application ApplicationView
}

type Reversal struct {
	ID     string
	Kind   string
	Record string
	Amount string
}

type ReversalView struct {
	Reversal   Reversal
	State      string
	Author     string
	Approver   string
	Digest     string
	WrittenAt  time.Time
	ApprovedAt time.Time
}

type LotView struct {
	Number    int
	Source    string
	Record    string
	Amount    string
	Remainder string
	Held      string
	Available string
	Expired   string
	Expiry    time.Time
}

type Statement struct {
	Account              string
	Currency             string
	Opening              string
	Increases            string
	Applications         string
	Expired              string
	ApplicationReversals string
	LotReversals         string
	Closing              string
	Held                 string
	Available            string
}

type Limits struct {
	Accounts     int
	Openings     int
	Increases    int
	Applications int
	Targets      int
	Reversals    int
	Workspace    int
}

func (l Limits) valid() bool {
	return l.Accounts > 0 && l.Accounts <= 100000 && l.Openings > 0 && l.Openings <= 100 && l.Increases > 0 && l.Increases <= 100000 &&
		l.Applications > 0 && l.Applications <= 1000000 && l.Targets > 0 && l.Targets <= 1000000 && l.Reversals > 0 && l.Reversals <= 1000 &&
		l.Workspace > 0 && l.Workspace <= 1000000
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

func optionalID(id string) bool { return id == "" || validID(id) }

func validCurrency(code string) bool {
	return len(code) == 3 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= 'Z' && code[2] >= 'A' && code[2] <= 'Z'
}

func validDate(text string) bool {
	if len(text) != 10 || text < "2000-01-01" || text > "2099-12-31" {
		return false
	}
	parsed, err := time.Parse("2006-01-02", text)
	return err == nil && parsed.Format("2006-01-02") == text
}

func optionalDate(text string) bool { return text == "" || validDate(text) }

func validDigest(text string) bool {
	if len(text) != 64 {
		return false
	}
	for index := 0; index < len(text); index++ {
		c := text[index]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func validImport(i Import) bool {
	if i == (Import{}) {
		return true
	}
	return validID(i.Ledger) && validID(i.Transaction) && validDate(i.Occurred)
}
