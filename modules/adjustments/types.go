package adjustments

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
	maxIDBytes     = 128
	maxNumberBytes = 64
	maxScale       = 8
	sourceLabel    = "adjustment_source_v1"
	draftLabel     = "adjustment_draft_v1"
	releaseLabel   = "adjustment_release_v1"
	maxApprovers   = 3
)

const (
	KindHuman     = "human"
	KindAssistant = "assistant"
	KindSystem    = "system"

	EntryQuantity    = "quantity"
	EntryFixed       = "fixed"
	EntryTargetTotal = "target_total"

	ReasonSourceError  = "source_error"
	ReasonContractTerm = "contract_term"
	ReasonGoodwill     = "goodwill"
	ReasonRounding     = "rounding"
	ReasonMeasurement  = "measurement"
	ReasonOther        = "other"

	StateOpen      = "open"
	StateReleased  = "released"
	StateWithdrawn = "withdrawn"

	DispositionNewRun     = "new_run"
	DispositionCorrection = "correction"

	RefuseStaleRevision   = "stale_revision"
	RefuseDigestChanged   = "digest_changed"
	RefuseZeroDifference  = "zero_difference"
	RefuseNegativeTotal   = "negative_total"
	RefuseNotValidated    = "not_validated"
	RefuseAINotAllowed    = "ai_not_allowed"
	RefuseSelfApproval    = "self_approval"
	RefuseNothingToAdjust = "nothing_to_adjust"
	RefuseFinalMismatch   = "final_mismatch"
	RefuseReleased        = "released"
	RefuseWithdrawn       = "withdrawn"
	RefuseAlreadyReleased = "already_released"
	RefuseNoUnitPrice     = "no_unit_price"
	RefuseEntryRemoved    = "entry_removed"
	RefuseOutsidePeriod   = "outside_period"
	RefuseBaseChanged     = "base_changed"
	RefuseBaseConsumed    = "base_consumed"
)

var (
	kinds   = map[string]bool{KindHuman: true, KindAssistant: true, KindSystem: true}
	entries = map[string]bool{EntryQuantity: true, EntryFixed: true, EntryTargetTotal: true}
	reasons = map[string]bool{ReasonSourceError: true, ReasonContractTerm: true, ReasonGoodwill: true, ReasonRounding: true, ReasonMeasurement: true, ReasonOther: true}
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

type Base struct {
	Run      string
	Digest   string
	Issued   bool
	Document string
	Currency string
	Scale    int
	Mode     string
}

type SourceLine struct {
	ID       string
	Charge   string
	Quantity string
	Price    string
	Amount   string
}

type Draft struct {
	ID       string
	Customer string
	Contract string
	From     string
	To       string
	Base     Base
	Lines    []SourceLine
}

type Entry struct {
	ID       string
	Kind     string
	Line     string
	Target   string
	Amount   string
	Reason   string
	Evidence []string
	Finding  string
	From     string
	To       string
}

type EntryView struct {
	Entry     Entry
	Author    Actor
	Written   time.Time
	Active    bool
	RemovedBy Actor
	Removed   time.Time
}

type Difference struct {
	Entry    string
	Kind     string
	Line     string
	Charge   string
	Current  string
	Target   string
	Quantity string
	Amount   string
}

type LineView struct {
	Line     SourceLine
	Adjusted string
}

type Check struct {
	Revision   int
	Digest     string
	Tax        string
	Allocation string
	TaxAmount  string
	Final      string
}

type CheckView struct {
	Check    Check
	Actor    Actor
	Recorded time.Time
}

type Release struct {
	Request   string
	Revision  int
	Subject   string
	Base      Base
	Approval  string
	Approvers []Actor
}

type Outcome struct {
	Release     Release
	Disposition string
	Run         string
	Document    string
	Currency    string
	Scale       int
	Differences []Difference
	Source      string
	Adjustment  string
	Total       string
	Gross       string
	Digest      string
	Subject     string
	Tax         string
	Allocation  string
	TaxAmount   string
	Final       string
	Actor       Actor
	Released    time.Time
}

type View struct {
	Draft        Draft
	Opener       Actor
	Opened       time.Time
	SourceDigest string
	Revision     int
	State        string
	Entries      []EntryView
	Differences  []Difference
	Lines        []LineView
	Source       string
	Adjustment   string
	Total        string
	Gross        string
	Findings     []string
	Digest       string
	Subject      string
	Check        *CheckView
	Outcome      *Outcome
	Withdrawer   Actor
	Withdrawn    time.Time
}

type Limits struct {
	Workspaces int
	Drafts     int
	Lines      int
	Entries    int
	Evidence   int
}

func within(value, cap int) bool { return value > 0 && value <= cap }

func (l Limits) valid() bool {
	return within(l.Workspaces, 100000) && within(l.Drafts, 1000000) && within(l.Lines, 10000) && within(l.Entries, 1000) && within(l.Evidence, 64)
}
