package releasetrace

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

	ChangeRequirement = "requirement"
	ChangeRetire      = "retire"
	ChangeImplemented = "implemented"
	ChangeBuild       = "build"
	ChangeHarness     = "harness"
	ChangeProof       = "proof"
	ChangePath        = "path"
	ChangeReconcile   = "reconcile"
	ChangeActivate    = "activate"

	ProposalPending   = "pending"
	ProposalApplied   = "applied"
	ProposalWithdrawn = "withdrawn"

	CategoryAmount    = "amount"
	CategoryIsolation = "isolation"
	CategoryApproval  = "approval"
	CategoryDuplicate = "duplicate"
	CategoryRecovery  = "recovery"
	CategoryBulk      = "bulk"

	IssuanceAPI  = "api"
	IssuanceFile = "file"
	IssuanceXLSX = "xlsx"

	StateDesigned       = "designed"
	StateImplemented    = "implemented"
	StateFailing        = "failing"
	StateTested         = "tested"
	StateCustomerProven = "customer_proven"

	PathDesigned = "designed"
	PathActive   = "active"

	ProblemUnknown            = "unknown"
	ProblemCommonUnregistered = "common_unregistered"
	ProblemRetired            = "retired"
	ProblemCommonMissing      = "common_missing"
	ProblemNotTested          = "not_tested"
	ProblemBlocking           = "blocking"
	ProblemPathInactive       = "path_inactive"
	ProblemPathBuild          = "path_build"
	ProblemPathReconcile      = "path_reconciliation"
	ProblemPathRequirement    = "path_requirement"
	ProblemPathEvidence       = "path_evidence"

	ReasonAINotAllowed           = "ai_not_allowed"
	ReasonNotHuman               = "not_human"
	ReasonSelfApproval           = "self_approval"
	ReasonNotPending             = "not_pending"
	ReasonNotProposer            = "not_proposer"
	ReasonIdentityTaken          = "identity_taken"
	ReasonNotHarness             = "not_harness"
	ReasonUnknownTest            = "unknown_test"
	ReasonUnknownBuild           = "unknown_build"
	ReasonSelfVerification       = "self_verification"
	ReasonWeakerReplacement      = "weaker_replacement"
	ReasonRetired                = "retired"
	ReasonReconciliationMissing  = "reconciliation_missing"
	ReasonCategoryMissing        = "category_missing"
	ReasonRequirementNotTested   = "requirement_not_tested"
	ReasonNotTested              = "not_tested"
	ReasonFuturePeriod           = "future_period"
	ReasonCommonNeedsReplacement = "common_needs_replacement"
)

var Categories = []string{CategoryAmount, CategoryApproval, CategoryBulk, CategoryDuplicate, CategoryIsolation, CategoryRecovery}

type Refusal struct {
	Reason string
}

func (e Refusal) Error() string { return ErrRejected.Error() + ": " + e.Reason }

func (e Refusal) Unwrap() error { return ErrRejected }

type Actor struct {
	ID   string
	Kind string
}

type Key struct {
	Namespace string
	ID        string
}

type TestRef struct {
	ID       string
	Category string
}

type Requirement struct {
	Namespace  string
	ID         string
	Source     string
	Owner      string
	Interfaces []string
	Tests      []TestRef
	Common     bool
	Blocking   bool
	Design     string
}

type Retire struct {
	Key         Key
	Replacement Key
	Reason      string
}

type Implemented struct {
	Key    Key
	Commit string
}

type Build struct {
	Build  string
	Commit string
}

type Harness struct {
	ID       string
	Identity string
}

type Proof struct {
	Key         Key
	Build       string
	Environment string
	Customer    string
	Period      string
	Evidence    string
}

type Path struct {
	ID           string
	Provider     string
	Contract     string
	Role         string
	Version      string
	Issuance     string
	Template     string
	Requirements []Key
}

type Reconcile struct {
	Path        string
	Period      string
	Evidence    string
	Matched     bool
	Differences int
}

type Activate struct {
	Path  string
	Build string
	Proof string
}

type Change struct {
	Kind        string
	Requirement Requirement
	Retire      Retire
	Implemented Implemented
	Build       Build
	Harness     Harness
	Proof       Proof
	Path        Path
	Reconcile   Reconcile
	Activate    Activate
}

type ProposalView struct {
	Number   uint64
	Change   Change
	State    string
	Proposer string
	Approver string
}

type Result struct {
	Namespace string
	Test      string
	Build     string
	Passed    bool
	Evidence  string
}

type ResultView struct {
	Result   Result
	Harness  string
	At       time.Time
	Passes   int
	Failures int
	Counted  bool
}

type TestState struct {
	Test     string
	Counted  bool
	Passed   bool
	Passes   int
	Failures int
}

type RequirementView struct {
	Requirement Requirement
	State       string
	Retired     bool
	Replacement Key
	Commit      string
	Tests       []TestState
}

type PathView struct {
	Path            Path
	State           string
	Build           string
	Proof           string
	Reconciliations []Reconcile
}

type Release struct {
	Version      string
	Build        string
	Requirements []Key
	Paths        []string
}

type Problem struct {
	Problem string
	Subject string
}

type GateView struct {
	Ready           bool
	Problems        []Problem
	States          map[Key]string
	Templates       []string
	RetiredBlocking []Key
}
