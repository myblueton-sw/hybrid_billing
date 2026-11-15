package closing

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
	maxIDBytes    = 128
	maxWorkspaces = 100000
	maxCalendars  = 10000
	maxCycles     = 100000
	maxTargets    = 1000000
	maxHistory    = 10000000
	maxClaims     = 10000
	maxHolidays   = 1000
	maxWeekend    = 6
	maxDays       = 31
	maxSearch     = 400
	minLead       = time.Hour
	maxLead       = 7 * 24 * time.Hour
	overdueAfter  = 24 * time.Hour
	endOfDay      = 1440

	KindHuman     = "human"
	KindAssistant = "assistant"
	KindSystem    = "system"

	StageCollect  = "collect"
	StageValidate = "validate"
	StageApprove  = "approve"
	StageIssue    = "issue"
	StageDeliver  = "deliver"
	StageSettle   = "settle"

	BasisInternal = "internal"
	BasisERP      = "erp"
	BasisFile     = "file"

	EvidenceInternal = "internal_approval"
	EvidenceERP      = "erp_posting"
	EvidenceFile     = "file_receipt"

	StateProposed = "proposed"
	StateActive   = "active"

	ItemOpen   = "open"
	ItemDone   = "done"
	ItemWaived = "waived"
	ItemHeld   = "held"

	EventAdded            = "added"
	EventCompleted        = "completed"
	EventReopened         = "reopened"
	EventHeld             = "held"
	EventReleased         = "released"
	EventWaiverRequested  = "waiver_requested"
	EventWaiverApproved   = "waiver_approved"
	EventWaiverWithdrawn  = "waiver_withdrawn"
	EventPostponeRequest  = "postponement_requested"
	EventPostponeApproved = "postponement_approved"
	EventPostponeWithdraw = "postponement_withdrawn"

	StepDueSoon = "due_soon"
	StepDue     = "due"
	StepOverdue = "overdue"

	RefusalAINotAllowed    = "ai_not_allowed"
	RefusalNotHuman        = "not_human"
	RefusalNotSystem       = "not_system"
	RefusalSelfApproval    = "self_approval"
	RefusalNotPending      = "not_pending"
	RefusalNotApproved     = "calendar_not_approved"
	RefusalOutsideCalendar = "outside_calendar"
	RefusalStageOrder      = "stage_order"
	RefusalBasisMismatch   = "basis_mismatch"
	RefusalWaiverPending   = "waiver_pending"
	RefusalPostponePending = "postponement_pending"
	RefusalNotLater        = "not_later"
	RefusalState           = "state"
	RefusalNotRequester    = "not_requester"
	RefusalZoneUnavailable = "zone_unavailable"
	RefusalTargetPresent   = "target_present"
	RefusalCalendarPending = "calendar_pending"
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

type Config struct {
	Zones func(string) (*time.Location, error)
}

type Limits struct {
	Workspaces int
	Calendars  int
	Cycles     int
	Targets    int
	History    int
	Claims     int
	Lead       time.Duration
}

type Range struct {
	From string
	To   string
}

type Calendar struct {
	ID       string
	Zone     string
	Weekend  []time.Weekday
	Holidays []string
	Close    int
	Covers   Range
}

type CalendarView struct {
	Calendar Calendar
	Revision int
	State    string
	Proposer string
	Approver string
}

type Rule struct {
	Days     int
	Business bool
}

type StageSpec struct {
	Stage string
	Rule  Rule
	Basis string
}

type Cycle struct {
	ID       string
	Period   string
	Calendar string
	Revision int
	Stages   []StageSpec
}

type StageView struct {
	Spec        StageSpec
	OriginalDue time.Time
	CurrentDue  time.Time
	Pending     *Postponement
	Requester   string
	History     []DueChange
}

type DueChange struct {
	Sequence int
	Kind     string
	Due      time.Time
	Reason   string
	Actor    string
	At       time.Time
}

type CycleView struct {
	Cycle    Cycle
	Opener   string
	Opened   time.Time
	Sequence int
	Stages   []StageView
	Targets  []string
}

type Addition struct {
	Targets []string
	Reason  string
}

type Evidence struct {
	Kind      string
	Reference string
}

type Waiver struct {
	Reason   string
	Evidence string
}

type Postponement struct {
	Due    time.Time
	Reason string
}

type ItemEvent struct {
	Sequence     int
	Kind         string
	Actor        string
	At           time.Time
	Evidence     Evidence
	Reason       string
	LateOriginal time.Duration
	LateCurrent  time.Duration
}

type ItemView struct {
	Target   string
	Stage    string
	State    string
	Waiver   *Waiver
	History  []ItemEvent
	LateOpen time.Duration
}

type AsOf struct {
	Sequence int
	At       time.Time
}

type ProgressView struct {
	Sequence    int
	Denominator int
	Targets     int
	Done        int
	Waived      int
	Held        int
	Open        int
	OpenPastDue int
	HeldPastDue int
	DoneLate    int
	OriginalDue time.Time
	CurrentDue  time.Time
}

type Event struct {
	Cycle   string
	Stage   string
	Step    string
	Due     time.Time
	At      time.Time
	Targets []string
	Skipped []string
}
