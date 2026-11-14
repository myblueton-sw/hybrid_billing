package recognition

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
	maxIDBytes      = 128
	maxNumberBytes  = 64
	maxScale        = 8
	maxMonths       = 12
	maxWorkspaces   = 100000
	maxProfiles     = 100000
	maxBatches      = 1000000
	maxLines        = 10000
	maxAdjustments  = 1000000
	batchLabel      = "recognition_batch_v1"
	adjustmentLabel = "recognition_adjustment_v1"
	reconcileLabel  = "recognition_reconciliation_v1"
	Purpose         = "monthly_recognition"
)

const (
	KindHuman     = "human"
	KindAssistant = "assistant"
	KindSystem    = "system"

	BatchInitial   = "initial"
	BatchAmendment = "amendment"
	BatchReversal  = "reversal"

	BatchOpen      = "open"
	BatchApproved  = "approved"
	BatchVoid      = "void"
	BatchAbandoned = "abandoned"

	ResultSent     = "sent"
	ResultAccepted = "accepted"
	ResultPosted   = "posted"
	ResultFailed   = "failed"
	ResultRejected = "rejected"
	ResultUnknown  = "unknown"

	AdjustmentPending   = "pending"
	AdjustmentApproved  = "approved"
	AdjustmentRejected  = "rejected"
	AdjustmentWithdrawn = "withdrawn"

	VerdictApprove = "approve"
	VerdictReject  = "reject"

	MethodNone = "none"

	StateCollecting         = "collecting"
	StatePostingUnconfirmed = "posting_unconfirmed"
	StateMixedBasis         = "mixed_basis"
	StateAdjustmentPending  = "adjustment_pending"
	StateAmountDifference   = "amount_difference"
	StateAdjustmentPosting  = "adjustment_posting"
	StateMatched            = "matched"

	ReasonAINotAllowed          = "ai_not_allowed"
	ReasonNotHuman              = "not_human"
	ReasonUnknownProfile        = "unknown_profile"
	ReasonProvisionalNotAllowed = "provisional_not_allowed"
	ReasonMonthCleared          = "month_cleared"
	ReasonPeriodClosed          = "period_closed"
	ReasonPostingMismatch       = "posting_mismatch"
	ReasonInitialExists         = "initial_exists"
	ReasonNoInitial             = "no_initial"
	ReasonUnknownTarget         = "unknown_target"
	ReasonTargetIsReversal      = "target_is_reversal"
	ReasonTargetNotPosted       = "target_not_posted"
	ReasonAlreadyReversed       = "already_reversed"
	ReasonCarriesAdjustment     = "carries_adjustment"
	ReasonEmpty                 = "empty"
	ReasonNotPending            = "not_pending"
	ReasonDigestChanged         = "digest_changed"
	ReasonSelfApproval          = "self_approval"
	ReasonNotApproved           = "not_approved"
	ReasonAlreadyPosted         = "already_posted"
	ReasonOutcomeUnknown        = "outcome_unknown"
	ReasonUnknownBatch          = "unknown_batch"
	ReasonCarrierMismatch       = "carrier_mismatch"
	ReasonCarrierPosted         = "carrier_posted"
	ReasonHasLiveAmendments     = "has_live_amendments"
	ReasonInitialReversing      = "initial_reversing"
	ReasonBasisMismatch         = "basis_mismatch"
	ReasonNoMethod              = "no_double_recognition_method"
	ReasonMonthAlreadyCleared   = "month_already_cleared"
	ReasonNotMatched            = "not_matched"
)

var (
	bases    = map[string]bool{"sales_billing": true, "provider_cost": true, "internal_allocation": true}
	taxBases = map[string]bool{"tax_separate": true, "tax_free": true}
	methods  = map[string]bool{"offset": true, "replace": true, "reverse_then_confirm": true, "reference_link": true, MethodNone: true}
	causes   = map[string]bool{"tier_pricing": true, "discount": true, "commitment": true, "late_source": true, "credit": true, "rounding": true, "other": true}
	kinds    = map[string]bool{BatchInitial: true, BatchAmendment: true, BatchReversal: true}
	reports  = map[string]map[string]bool{
		ResultSent:     {ResultAccepted: true, ResultPosted: true, ResultFailed: true, ResultRejected: true, ResultUnknown: true},
		ResultAccepted: {ResultPosted: true, ResultFailed: true, ResultRejected: true, ResultUnknown: true},
		ResultUnknown:  {ResultAccepted: true, ResultPosted: true, ResultFailed: true, ResultRejected: true},
	}
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

type Limits struct {
	Workspaces  int
	Profiles    int
	Batches     int
	Lines       int
	Adjustments int
}

func (l Limits) valid() bool {
	return l.Workspaces >= 1 && l.Workspaces <= maxWorkspaces && l.Profiles >= 1 && l.Profiles <= maxProfiles &&
		l.Batches >= 1 && l.Batches <= maxBatches && l.Lines >= 1 && l.Lines <= maxLines &&
		l.Adjustments >= 1 && l.Adjustments <= maxAdjustments
}

type Profile struct {
	ID          string
	Entity      string
	Ledger      string
	Contract    string
	Basis       string
	TaxBasis    string
	Currency    string
	Scale       int
	Provisional bool
	Method      string
}

type ProfileView struct {
	Profile Profile
	Author  string
	At      time.Time
	Closed  []string
}

type Amounts struct {
	Net string
	Tax string
}

type Line struct {
	ID      string
	Lineage string
	Net     string
	Tax     string
}

type Batch struct {
	ID          string
	Profile     string
	Month       string
	Posting     string
	Kind        string
	Reverses    string
	Provisional bool
	Snapshot    string
	Lines       []Line
}

type BatchView struct {
	Batch      Batch
	State      string
	Total      Amounts
	Author     string
	Opened     time.Time
	Approver   string
	Approved   time.Time
	EndedBy    string
	Ended      time.Time
	Posted     bool
	External   string
	ReversedBy string
	Carries    string
	Digest     string
}

type Result struct {
	State    string
	External string
	Reason   string
}

type SubmissionView struct {
	Key       string
	Purpose   string
	Batch     string
	Digest    string
	Submitter string
	Sent      time.Time
	Result    Result
	Reporter  string
	Reported  time.Time
}

type Adjustment struct {
	ID      string
	Profile string
	Month   string
	Cause   string
	Before  Amounts
	After   Amounts
	Carrier string
}

type Decision struct {
	Verdict string
	Digest  string
}

type AdjustmentView struct {
	Adjustment Adjustment
	Carrier    string
	Linker     string
	Delta      Amounts
	State      string
	Proposer   string
	Proposed   time.Time
	Decider    string
	Decided    time.Time
	EndedBy    string
	Ended      time.Time
	Digest     string
}

type Quarter struct {
	ID       string
	Months   []string
	Currency string
	Basis    string
	TaxBasis string
	Billed   Amounts
	Final    bool
}

type MonthView struct {
	Month       string
	Net         string
	Tax         string
	Live        bool
	Open        int
	Unposted    int
	Provisional int
	Final       bool
}

type Reconciliation struct {
	Profile    string
	Quarter    Quarter
	Months     []MonthView
	Recognised Amounts
	Term       Amounts
	Base       Amounts
	BaseGross  string
	Difference Amounts
	Gross      string
	Pending    []string
	Unposted   []string
	State      string
	Digest     string
}

type ClearanceView struct {
	Quarter Quarter
	Method  string
	Digest  string
	Actor   string
	At      time.Time
}
