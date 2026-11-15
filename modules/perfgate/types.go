package perfgate

import (
	"errors"
	"time"
)

var (
	ErrInvalid  = errors.New("a request or an input is incomplete or malformed")
	ErrNotFound = errors.New("what the request names is not registered")
	ErrConflict = errors.New("another fact is already recorded under this identifier")
	ErrRejected = errors.New("a recorded state or rule refuses this request")
	ErrLimit    = errors.New("a configured limit or bound is exceeded")
)

const (
	KindHuman     = "human"
	KindAssistant = "assistant"
	KindSystem    = "system"

	ChangeProfile  = "profile"
	ChangeHardware = "hardware"
	ChangeBudget   = "budget"
	ChangeApproval = "approval"
	ChangeHarness  = "harness"
	ChangeRetire   = "retire"

	ProposalPending   = "pending"
	ProposalApplied   = "applied"
	ProposalWithdrawn = "withdrawn"

	GateE2E             = "e2e"
	GateDailyInflow     = "daily_inflow"
	GateMonthlyClose    = "monthly_close"
	GateInterruptResume = "interrupt_resume"
	GateCatchUp         = "catch_up"

	StorageSSD     = "ssd"
	StorageHDD     = "hdd"
	StorageNetwork = "network"

	MetricMaxRSS     = "max_rss_bytes"
	MetricDBIO       = "db_io_bytes"
	MetricWAL        = "wal_bytes"
	MetricLockWait   = "lock_wait_ms"
	MetricP95        = "p95_ms"
	MetricP99        = "p99_ms"
	MetricThroughput = "throughput_rows_s"
	MetricCompletion = "completion_ms"
	MetricResume     = "resume_ms"

	BudgetProposed = "proposed"
	BudgetApproved = "approved"

	VerdictOtherData      = "other_data"
	VerdictIncomplete     = "incomplete"
	VerdictIncorrect      = "incorrect"
	VerdictRevoked        = "revoked"
	VerdictSelfVerified   = "self_verified"
	VerdictMissingMetric  = "missing_metric"
	VerdictUnbudgeted     = "unbudgeted"
	VerdictOtherHardware  = "other_hardware"
	VerdictOverBudget     = "over_budget"
	VerdictPredatesBudget = "predates_budget"
	VerdictProvisional    = "provisional"
	VerdictPassed         = "passed"

	StatusNotRun      = "not_run"
	StatusFailing     = "failing"
	StatusProvisional = "provisional"
	StatusPassed      = "passed"

	ReasonAINotAllowed     = "ai_not_allowed"
	ReasonNotHuman         = "not_human"
	ReasonSelfApproval     = "self_approval"
	ReasonNotPending       = "not_pending"
	ReasonNotProposer      = "not_proposer"
	ReasonIdentityTaken    = "identity_taken"
	ReasonNotHarness       = "not_harness"
	ReasonSelfVerification = "self_verification"
	ReasonBelowP0          = "below_p0"
	ReasonMissingMetric    = "missing_metric"
	ReasonNotActive        = "not_active"
	ReasonSamePeople       = "same_people"
	ReasonRetired          = "retired"
)

var (
	Gates   = []string{GateE2E, GateDailyInflow, GateMonthlyClose, GateInterruptResume, GateCatchUp}
	Metrics = []string{MetricCompletion, MetricDBIO, MetricLockWait, MetricMaxRSS, MetricP95, MetricP99, MetricResume, MetricThroughput, MetricWAL}
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

type Axes struct {
	RowsRun           int64
	RowsDay           int64
	RowsMonth         int64
	Customers         int64
	Accounts          int64
	SharedPools       int64
	SkewPermille      int64
	FanOut            int64
	ConcurrentCloses  int64
	PeakWindowMinutes int64
	RetentionMonths   int64
	FailureInjections int64
	BacklogRows       int64
}

type Profile struct {
	ID       string
	Gate     string
	Axes     Axes
	Dataset  string
	Expected string
}

type Hardware struct {
	ID        string
	CPUs      int64
	MemoryMiB int64
	DiskGiB   int64
	Storage   string
	Spec      string
}

type Limit struct {
	Metric string
	Value  int64
}

type Budget struct {
	ID       string
	Profile  string
	Hardware string
	Limits   []Limit
}

type Approval struct {
	Budget       string
	Distribution string
}

type Harness struct {
	ID       string
	Identity string
}

type Retire struct {
	Profile string
}

type Change struct {
	Kind     string
	Profile  Profile
	Hardware Hardware
	Budget   Budget
	Approval Approval
	Harness  Harness
	Retire   Retire
}

type ProposalView struct {
	Number   uint64
	Change   Change
	State    string
	Proposer string
	Approver string
}

type BudgetView struct {
	Budget       Budget
	State        string
	Distribution string
	Proposer     string
	Approver     string
	ApprovalBy   string
	ApprovedBy   string
	ApprovedAt   time.Time
}

type Measure struct {
	Metric string
	Value  int64
}

type Run struct {
	Profile     string
	Hardware    string
	Build       string
	Dataset     string
	Output      string
	Rows        int64
	DurationS   int64
	Mismatches  int64
	Interrupted bool
	Metrics     []Measure
}

type RunView struct {
	Number  uint64
	Run     Run
	Harness string
	At      time.Time
	Verdict string
}

type ProfileStatus struct {
	Profile  string
	Status   string
	Run      uint64
	Failures int
}

type GateView struct {
	Gate     string
	Build    string
	Status   string
	Profiles []ProfileStatus
	Retired  []ProfileStatus
}

type ReadinessView struct {
	Build string
	Ready bool
	Gates []GateView
}
