package simulation

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

	ChangeDataset     = "dataset"
	ChangeScenario    = "scenario"
	ChangeExpectation = "expectation"
	ChangeProfile     = "profile"
	ChangeCell        = "cell"
	ChangeTarget      = "target"
	ChangeProof       = "proof"
	ChangeHarness     = "harness"

	ProposalPending   = "pending"
	ProposalApplied   = "applied"
	ProposalWithdrawn = "withdrawn"

	OriginPublic    = "public"
	OriginConverted = "converted"
	OriginSynthetic = "synthetic"

	PurposeFormat  = "format"
	PurposeBilling = "billing"

	TopologySingle = "single"
	TopologyGroup  = "group"
	TopologyMSP    = "msp"

	DeploymentCloud  = "cloud"
	DeploymentOnprem = "onprem"
	DeploymentMixed  = "mixed"

	MethodManual      = "manual_sheet"
	MethodIndependent = "independent_logic"

	CheckAmount = "amount"
	CheckCount  = "count"
	CheckDigest = "digest"
	CheckStatus = "status"

	OutcomeMatch    = "match"
	OutcomeMismatch = "mismatch"
	OutcomeMissing  = "missing"

	KeySingle          = "single"
	KeyReproduced      = "reproduced"
	KeyNonReproducible = "non_reproducible"

	VerdictNone            = "none"
	VerdictPass            = "pass"
	VerdictFail            = "fail"
	VerdictUnconfirmed     = "unconfirmed"
	VerdictNonReproducible = "non_reproducible"
	VerdictRevoked         = "revoked"

	CoverageTested      = "tested"
	CoverageUnsupported = "unsupported"
	CoverageUndecided   = "undecided"

	CellOpen        = "open"
	CellUnsupported = "unsupported"
	CellUndecided   = "undecided"
	CellFailed      = "failed"
	CellPassed      = "passed"
	CellPending     = "pending"

	TargetCSP     = "csp"
	TargetERP     = "erp"
	TargetProduct = "product"

	DemoPassed  = "passed"
	DemoFailed  = "failed"
	DemoPending = "pending"

	IntegrationNotProven    = "not_proven"
	IntegrationUnclassified = "proven_unclassified"

	EventFullUpfront     = "full_upfront"
	EventPartialUpfront  = "partial_upfront"
	EventNoUpfront       = "no_upfront"
	EventUtilizationZero = "utilization_zero"
	EventUtilizationFull = "utilization_full"
	EventOverage         = "overage"
	EventConcurrent      = "concurrent_commitments"
	EventMidMonthStart   = "mid_month_start"
	EventMidMonthExpiry  = "mid_month_expiry"
	EventMidMonthMove    = "mid_month_move"
	EventShareScope      = "share_scope_change"
	EventLateCredit      = "late_credit"
	EventRefund          = "refund"
	EventExchange        = "exchange"
	EventCancelCorrect   = "cancel_then_correct"
	EventNegativeSavings = "negative_savings"
	EventFXChange        = "fx_change"
	EventPartialCSVError = "partial_csv_error"

	ReasonAINotAllowed     = "ai_not_allowed"
	ReasonNotHuman         = "not_human"
	ReasonSelfApproval     = "self_approval"
	ReasonNotPending       = "not_pending"
	ReasonNotProposer      = "not_proposer"
	ReasonIdentityTaken    = "identity_taken"
	ReasonNotHarness       = "not_harness"
	ReasonSelfVerification = "self_verification"
	ReasonNoticeChanged    = "notice_changed"
	ReasonPublicAnswer     = "public_answer"
	ReasonRunsRecorded     = "runs_recorded"
	ReasonNoExpectation    = "no_expectation"
	ReasonRunOutput        = "run_output"
	ReasonNotCovered       = "not_covered"
	ReasonNoAmount         = "no_amount"
	ReasonNoTotal          = "no_total"
	ReasonWrongPurpose     = "wrong_purpose"
	ReasonWrongOrigin      = "wrong_origin"
)

var Events = []string{EventFullUpfront, EventPartialUpfront, EventNoUpfront, EventUtilizationZero, EventUtilizationFull, EventOverage, EventConcurrent,
	EventMidMonthStart, EventMidMonthExpiry, EventMidMonthMove, EventShareScope, EventLateCredit, EventRefund, EventExchange, EventCancelCorrect,
	EventNegativeSavings, EventFXChange, EventPartialCSVError}

type Refusal struct {
	Reason string
}

func (e Refusal) Error() string { return ErrRejected.Error() + ": " + e.Reason }

func (e Refusal) Unwrap() error { return ErrRejected }

type Actor struct {
	ID   string
	Kind string
}

type Dataset struct {
	ID          string
	Version     string
	Origin      string
	Digest      string
	Source      string
	License     string
	Notice      string
	Parent      string
	Tool        string
	ToolVersion string
	Seed        string
	Coverage    []string
}

type Scenario struct {
	ID             string
	FixtureVersion string
	Purpose        string
	Topology       string
	Deployment     string
	Seed           string
	Start          time.Time
	Zone           string
	Relations      string
	Policy         string
	FX             string
	Datasets       []string
	Events         string
	EventCount     int64
	Profile        string
	Covers         []string
}

type Check struct {
	Name     string
	Kind     string
	Value    string
	Currency string
	Document string
	Period   string
}

type Expectation struct {
	Scenario string
	Version  int
	Method   string
	Evidence string
	Checks   []Check
}

type Profile struct {
	ID string
}

type Cell struct {
	Profile  string
	Event    string
	Coverage string
	Scenario string
	Checks   []string
	Dataset  string
	Evidence string
	Decision string
}

type Target struct {
	ID        string
	Kind      string
	Scenarios []string
}

type Proof struct {
	Target      string
	Environment string
	Evidence    string
}

type Harness struct {
	ID       string
	Identity string
}

type Change struct {
	Kind        string
	Dataset     Dataset
	Scenario    Scenario
	Expectation Expectation
	Profile     Profile
	Cell        Cell
	Target      Target
	Proof       Proof
	Harness     Harness
}

type ProposalView struct {
	Number   uint64
	Change   Change
	State    string
	Proposer string
	Approver string
}

type Result struct {
	Name     string
	Value    string
	Currency string
}

type Run struct {
	Scenario string
	Engine   string
	Rules    string
	Output   string
	Results  []Result
}

type Outcome struct {
	Name     string
	Outcome  string
	Value    string
	Currency string
}

type RunView struct {
	Number      uint64
	Scenario    string
	Engine      string
	Rules       string
	Output      string
	Results     string
	Harness     string
	At          time.Time
	Expectation int
	Outcomes    []Outcome
	Unexpected  int
	Copied      bool
	Key         string
	MarkedBy    uint64
	Verdict     string
}

type CurrentView struct {
	Scenario string
	Verdict  string
	Run      uint64
	Engine   string
	Rules    string
	Expected int
}

type CellChange struct {
	Replaced Cell
	Status   string
}

type CellView struct {
	Cell      Cell
	Status    string
	Revisions int
	History   []CellChange
	Failures  int
	Engine    string
	Rules     string
}

type ProofView struct {
	Environment string
	Evidence    string
	Approver    string
	At          time.Time
	Count       int
}

type StatusView struct {
	Target      Target
	Demo        string
	Integration string
	Proof       ProofView
}
