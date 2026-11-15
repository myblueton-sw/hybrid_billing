package corrections

import (
	"errors"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
)

var (
	ErrInvalid  = errors.New("a request or an input is incomplete or malformed")
	ErrNotFound = errors.New("what the request names does not exist in this workspace")
	ErrConflict = errors.New("another fact is already recorded under this identifier")
	ErrRejected = errors.New("a recorded state or rule refuses this request")
	ErrLimit    = errors.New("a configured limit or bound is exceeded")
)

const (
	maxIDBytes     = 128
	maxDigits      = 30
	maxWorkspaces  = 100000
	maxCorrections = 1000000
	maxRecords     = 1000000
	requestLabel   = "corrections_request_v1"

	KindHuman     = "human"
	KindAssistant = "assistant"
	KindSystem    = "system"

	ReasonLateSource     = "late_source"
	ReasonRetroactive    = "retroactive_change"
	ReasonQuantityError  = "quantity_error"
	ReasonPriceError     = "price_error"
	ReasonOther          = "other"
	DirectionCredit      = "credit"
	DirectionDebit       = "debit"
	StateRequested       = "requested"
	StateApproved        = "approved"
	StateApplied         = "applied"
	StateWithdrawn       = "withdrawn"
	OutcomeMatched       = "matched"
	OutcomeDiverged      = "diverged"
	RefusalAINotAllowed  = "ai_not_allowed"
	RefusalNotHuman      = "not_human"
	RefusalNotOriginal   = "not_original"
	RefusalUnavailable   = "documents_unavailable"
	RefusalScale         = "scale_mismatch"
	RefusalKeyReused     = "key_reused"
	RefusalPending       = "correction_pending"
	RefusalNoChange      = "no_change"
	RefusalSelfApproval  = "self_approval"
	RefusalDigestChanged = "digest_changed"
	RefusalBaseChanged   = "base_changed"
	RefusalNotRequested  = "not_requested"
	RefusalNotApproved   = "not_approved"
	RefusalClosed        = "closed"
	RefusalLinked        = "document_linked"
	RefusalUnlinked      = "unlinked_credit"
	RefusalDebit         = "debit_unsupported"
	RefusalCreditIssued  = "credit_issued"
	RefusalMismatch      = "document_mismatch"
	RefusalAmount        = "amount_mismatch"
	RefusalFuture        = "future_observation"
	RefusalNotUnlinked   = "not_unlinked"
	RefusalSelfExplain   = "self_explanation"
)

type Refusal struct {
	Reason string
}

func (e Refusal) Error() string { return ErrRejected.Error() + ": " + e.Reason }

func (e Refusal) Unwrap() error { return ErrRejected }

type Actor struct {
	ID   string
	Kind string
}

type Documents interface {
	DocumentOf(workspace, id string) (issuance.DocumentView, error)
	CreditsOf(workspace, id string) ([]string, error)
}

type Config struct {
	Documents Documents
}

type Limits struct {
	Workspaces  int
	Corrections int
	Records     int
}

type Correction struct {
	ID       string
	Original string
	Key      string
	Target   string
	Reason   string
	Evidence string
	Period   Period
}

type Period struct {
	From string
	To   string
}

type Base struct {
	Effective string
	Credits   int
	Digest    string
}

type View struct {
	Correction Correction
	Base       Base
	Difference string
	Direction  string
	State      string
	Requester  string
	Requested  time.Time
	Approver   string
	Approved   time.Time
	Closer     string
	CloserKind string
	Closed     time.Time
	Document   string
	Digest     string
	Duplicate  bool
}

type Explanation struct {
	Reason   string
	Evidence string
}

type Note struct {
	Credit      string
	Explanation Explanation
	Approver    string
	Issuer      string
	Actor       string
	At          time.Time
}

type Final struct {
	Reference string
	Amount    string
	Observed  time.Time
}

type Record struct {
	Final    Final
	Actor    string
	Kind     string
	At       time.Time
	Outcome  string
	Expected string
}

type Effective struct {
	Original  string
	Total     string
	Effective string
	Applied   []View
	Unlinked  []string
	Notes     []Note
	Open      *View
	Finals    []Record
}
