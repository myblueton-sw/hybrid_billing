package receivables

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
	maxZoneBytes   = 64
	maxNetDays     = 3650
	termLabel      = "receivable_term_v1"
	bindingLabel   = "receivable_binding_v1"
	documentLabel  = "receivable_document_v1"
	extensionLabel = "receivable_extension_v1"
	movementLabel  = "receivable_allocation_v1"
	reversalLabel  = "receivable_reversal_v1"
	holdLabel      = "receivable_hold_v1"
)

const (
	ModeCalendar    = "calendar"
	ModeBusinessDay = "business_day"

	ObligationPayable  = "payable"
	ObligationInternal = "internal"

	BasisInherit = "inherit"
	BasisReissue = "reissue"

	SourceAuthority = "authority"
	SourceManual    = "manual"

	StatePending   = "pending"
	StateApproved  = "approved"
	StateWithdrawn = "withdrawn"
	StateReleased  = "released"

	StatusNotApplicable = "not_applicable"
	StatusSettled       = "settled"
	StatusUnknown       = "unknown"
	StatusNotDue        = "not_due"
	StatusDueToday      = "due_today"
	StatusOverdue       = "overdue"

	BucketNotDue   = "not_due"
	BucketDueToday = "due_today"
	Bucket1to30    = "1_30"
	Bucket31to60   = "31_60"
	Bucket61to90   = "61_90"
	Bucket91Plus   = "91_plus"
	BucketUnknown  = "unknown"
	BucketReplaced = "replaced"

	ReasonTermMissing        = "term_missing"
	ReasonTermPending        = "term_pending"
	ReasonTermConflict       = "term_conflict"
	ReasonZoneMismatch       = "zone_mismatch"
	ReasonExceedsOutstanding = "exceeds_outstanding"
	ReasonFutureEffective    = "future_effective"
)

var (
	modes           = map[string]bool{ModeCalendar: true, ModeBusinessDay: true}
	bindReasons     = map[string]bool{"initial": true, "renegotiation": true, "correction": true}
	obligations     = map[string]bool{ObligationPayable: true, ObligationInternal: true}
	extendReasons   = map[string]bool{"customer_request": true, "dispute_resolution": true, "billing_error": true, "commercial_decision": true}
	allocationKinds = map[string]bool{"receipt": true, "credit": true, "cancellation": true, "correction": true}
	holdKinds       = map[string]bool{"dispute": true, "collection_hold": true}
	buckets         = []string{BucketNotDue, BucketDueToday, Bucket1to30, Bucket31to60, Bucket61to90, Bucket91Plus, BucketUnknown, BucketReplaced}
)

type Refusal struct {
	Reason string
}

func (e Refusal) Error() string { return ErrRejected.Error() + ": " + e.Reason }

func (e Refusal) Unwrap() error { return ErrRejected }

type Term struct {
	ID       string
	Revision int
	NetDays  int
	Mode     string
}

type TermView struct {
	Term       Term
	State      string
	Author     string
	Approver   string
	Written    time.Time
	ApprovedAt time.Time
	Digest     string
}

type Binding struct {
	ID       string
	Contract string
	Term     string
	Revision int
	Zone     string
	From     string
	To       string
	Reason   string
}

type BindingView struct {
	Binding     Binding
	State       string
	Author      string
	Approver    string
	WithdrawnBy string
	Written     time.Time
	ApprovedAt  time.Time
	PendingEnd  string
	EndAuthor   string
	Ends        []EndView
	Digest      string
}

type EndView struct {
	From     string
	To       string
	Author   string
	Approver string
	At       time.Time
}

type Receivable struct {
	Document   string
	Customer   string
	Entity     string
	Currency   string
	Contract   string
	Amount     string
	IssuedAt   time.Time
	Obligation string
	Replaces   string
	DueBasis   string
}

type Snapshot struct {
	Binding      string
	Term         string
	Revision     int
	NetDays      int
	Mode         string
	Zone         string
	IssueDate    string
	OriginalDue  string
	EffectiveDue string
}

type Extension struct {
	ID       string
	Document string
	Due      string
	Reason   string
}

type ExtensionView struct {
	Extension   Extension
	State       string
	Author      string
	Approver    string
	WithdrawnBy string
	Written     time.Time
	ApprovedAt  time.Time
	PreviousDue string
	DaysOverdue int
	Digest      string
}

type Allocation struct {
	ID        string
	Document  string
	Kind      string
	Amount    string
	Effective string
	Source    string
	Ledger    string
	Revision  string
	Evidence  string
}

type AllocationView struct {
	Allocation  Allocation
	State       string
	Author      string
	Approver    string
	WithdrawnBy string
	Arrived     time.Time
	ApprovedAt  time.Time
	Reversal    string
	Digest      string
}

type Reversal struct {
	ID         string
	Allocation string
	Effective  string
	Source     string
	Ledger     string
	Revision   string
	Evidence   string
}

type ReversalView struct {
	Reversal    Reversal
	State       string
	Author      string
	Approver    string
	WithdrawnBy string
	Arrived     time.Time
	ApprovedAt  time.Time
	Digest      string
}

type Hold struct {
	ID        string
	Document  string
	Kind      string
	Reference string
}

type HoldView struct {
	Hold       Hold
	State      string
	Author     string
	ReleasedBy string
	Written    time.Time
	ReleasedAt time.Time
	Digest     string
}

type Confirmation struct {
	Entity   string
	Currency string
	AsOf     time.Time
	Actor    string
	Recorded time.Time
}

type DocumentView struct {
	Receivable  Receivable
	Author      string
	Written     time.Time
	Snapshot    Snapshot
	ReplacedBy  string
	Extensions  []ExtensionView
	Allocations []AllocationView
	Reversals   []ReversalView
	Holds       []HoldView
	Digest      string
}

type StatusView struct {
	Document    DocumentView
	Status      string
	Outstanding string
	Days        int
	Stale       bool
	Confirmed   time.Time
	Evaluated   time.Time
}

type Estimate struct {
	Contract string
	Binding  string
	Date     string
	Due      string
}

type Filter struct {
	Customer string
	Entity   string
	Currency string
	Contract string
}

type Bucket struct {
	Count       int
	Amount      string
	StaleCount  int
	StaleAmount string
}

type AgingView struct {
	Currency string
	Buckets  map[string]Bucket
	Total    string
	Count    int
}

type Aging struct {
	Evaluated  time.Time
	Currencies []AgingView
}

type Limits struct {
	Workspaces  int
	Terms       int
	Bindings    int
	Receivables int
	Allocations int
	Extensions  int
	Holds       int
	NetDays     int
	Freshness   time.Duration
}

func BucketNames() []string { return append([]string(nil), buckets...) }

func within(value, cap int) bool { return value > 0 && value <= cap }

func (l Limits) valid() bool {
	return within(l.Workspaces, 100000) && within(l.Terms, 100000) && within(l.Bindings, 1000) && within(l.Receivables, 10000000) &&
		within(l.Allocations, 10000) && within(l.Extensions, 100) && within(l.Holds, 100) && within(l.NetDays, maxNetDays) &&
		l.Freshness >= time.Minute && l.Freshness <= 31*24*time.Hour
}
