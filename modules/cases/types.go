package cases

import (
	"errors"
	"time"
)

var (
	ErrInvalid  = errors.New("a request or an input is incomplete or malformed")
	ErrNotFound = errors.New("what the request names does not exist or is not visible to this actor")
	ErrConflict = errors.New("another fact is already recorded under this identifier")
	ErrRejected = errors.New("a recorded state or rule refuses this request")
	ErrLimit    = errors.New("a configured limit or bound is exceeded")
)

const (
	maxIDBytes     = 128
	maxNumberBytes = 64
	maxFraction    = 18
	maxDueAhead    = 366 * 24 * time.Hour
	findingLabel   = "billing_finding_v1"
	stepLabel      = "billing_finding_step_v1"
	caseLabel      = "billing_case_v1"
	entryLabel     = "billing_case_entry_v1"
	moveLabel      = "billing_case_step_v1"
)

const (
	SideOperator = "operator"
	SideCustomer = "customer"

	SourceManual   = "manual"
	SourceDetector = "detector"

	MeaningCost       = "cost"
	MeaningSale       = "sale"
	MeaningDifference = "difference"
	MeaningNone       = "none"

	FindingInReview         = "in_review"
	FindingConfirmedNormal  = "confirmed_normal"
	FindingAdjustmentNeeded = "adjustment_needed"
	FindingResolved         = "resolved"

	StepConfirm = "confirm_normal"
	StepAdjust  = "require_adjustment"
	StepResolve = "resolve"
	StepReopen  = "reopen"
	StepAssign  = "assign"

	KindInquiry = "inquiry"
	KindDispute = "dispute"

	CaseOpen      = "open"
	CaseAnswered  = "answered"
	CaseClosed    = "closed"
	CaseWithdrawn = "withdrawn"

	EntryMessage = "message"
	EntryAnswer  = "answer"
	EntryNote    = "note"

	OutcomeAnswered  = "answered"
	OutcomeUpheld    = "upheld"
	OutcomeNotUpheld = "not_upheld"
)

var (
	sides         = map[string]bool{SideOperator: true, SideCustomer: true}
	findingKinds  = map[string]bool{"cost_spike": true, "new_charge": true, "missing_credit_or_discount": true, "unattributed": true, "reconciliation_difference": true, "rate_conflict": true, "metering_gap": true}
	meanings      = map[string]bool{MeaningCost: true, MeaningSale: true, MeaningDifference: true, MeaningNone: true}
	severities    = map[string]bool{"low": true, "medium": true, "high": true, "critical": true}
	sources       = map[string]bool{SourceManual: true, SourceDetector: true}
	normalReasons = map[string]bool{"expected_change": true, "already_corrected": true, "duplicate": true, "out_of_scope": true}
	adjustReasons = map[string]bool{"rate_error": true, "missing_credit": true, "missing_discount": true, "attribution_error": true, "metering_error": true, "source_error": true}
	reopenReasons = map[string]bool{"new_evidence": true, "resolution_failed": true, "decision_revised": true}
	remedyKinds   = map[string]bool{"adjustment_draft": true, "correction_document": true, "credit_document": true, "source_correction": true}
	caseKinds     = map[string]bool{KindInquiry: true, KindDispute: true}
	topics        = map[string]bool{"amount": true, "quantity": true, "rate": true, "tax": true, "credit": true, "delivery": true, "access": true, "other": true}
	dueReasons    = map[string]bool{"initial": true, "customer_request": true, "information_pending": true, "escalation": true, "owner_change": true, "follow_up": true}
	caseRemedies  = map[string]bool{"adjustment_draft": true, "correction_document": true, "credit_document": true}
)

type Actor struct {
	ID       string
	Side     string
	Customer string
}

type Target struct {
	Customer string
	Contract string
	Document string
	Line     string
	Resource string
}

type Finding struct {
	ID          string
	Kind        string
	Target      Target
	PeriodStart string
	PeriodEnd   string
	Meaning     string
	Amount      string
	Currency    string
	Evidence    string
	Severity    string
	Owner       string
	Source      string
	Detector    string
}

type Step struct {
	ID       string
	Kind     string
	Reason   string
	Evidence string
	Owner    string
	Remedy   Remedy
	Actor    string
	From     string
	To       string
	At       time.Time
	Digest   string
}

type Remedy struct {
	Kind      string
	Reference string
}

type FindingView struct {
	Finding Finding
	State   string
	Owner   string
	Author  string
	Digest  string
	Written time.Time
	Steps   []Step
}

type Subject struct {
	Document string
	Line     string
}

type Case struct {
	ID       string
	Kind     string
	Customer string
	Subjects []Subject
	Topic    string
	Amount   string
	Currency string
	Body     string
}

type Entry struct {
	ID   string
	Body string
}

type CaseStep struct {
	Step    string
	Owner   string
	Due     time.Time
	Reason  string
	Closing Closing
}

type Closing struct {
	Outcome string
	Remedy  Remedy
}

type EntryView struct {
	ID     string
	Kind   string
	Body   string
	Actor  string
	Side   string
	At     time.Time
	Digest string
}

type DueChange struct {
	Due    time.Time
	Reason string
	Actor  string
	At     time.Time
	Digest string
}

type OwnerChange struct {
	Step   string
	Owner  string
	Actor  string
	At     time.Time
	Digest string
}

type ClosingView struct {
	Outcome  string
	Remedy   Remedy
	ClosedBy string
	Closed   time.Time
	Digest   string
	Reopened time.Time
	Reopener string
	Reopen   string
}

type CaseView struct {
	Case            Case
	State           string
	OpenedBy        string
	OpenedSide      string
	Opened          time.Time
	Owner           string
	OriginalDue     time.Time
	Due             time.Time
	DueHistory      []DueChange
	Owners          []OwnerChange
	Entries         []EntryView
	Outcome         string
	Remedy          Remedy
	ClosedBy        string
	Closed          time.Time
	Closings        []ClosingView
	ResponseOverdue bool
	Digest          string
}

type FindingFilter struct {
	State        string
	Customer     string
	Unattributed bool
	Severity     string
	After        string
}

type CaseFilter struct {
	State         string
	Customer      string
	Kind          string
	Document      string
	AfterCustomer string
	After         string
}

type Limits struct {
	Workspaces    int
	Findings      int
	Cases         int
	CustomerCases int
	Steps         int
	Messages      int
	Entries       int
	Subjects      int
	Page          int
}

func within(value, cap int) bool { return value > 0 && value <= cap }

func (l Limits) valid() bool {
	return within(l.Workspaces, 100000) && within(l.Findings, 1000000) && within(l.Cases, 1000000) && within(l.CustomerCases, l.Cases) &&
		within(l.Steps, 1000) && within(l.Messages, 1000) && within(l.Entries, 1000) && within(l.Subjects, 100) && within(l.Page, 1000)
}
