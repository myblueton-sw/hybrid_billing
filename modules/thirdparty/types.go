package thirdparty

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

	ChangeComponent = "component"
	ChangeAdopt     = "adopt"
	ChangeReject    = "reject"
	ChangeRevoke    = "revoke"

	ProposalPending   = "pending"
	ProposalApplied   = "applied"
	ProposalWithdrawn = "withdrawn"

	StateCandidate = "candidate"
	StateAdopted   = "adopted"
	StateRejected  = "rejected"
	StateRevoked   = "revoked"

	TelemetryNone     = "none"
	TelemetryOptional = "optional_off"
	TelemetryRequired = "required"

	ModUnmodified = "unmodified"
	ModPatched    = "patched"
	ModForked     = "forked"

	DistBundled    = "bundled"
	DistSeparate   = "separate"
	DistToolOnly   = "tool_only"
	DistNotShipped = "not_shipped"

	ProblemUnknown           = "unknown"
	ProblemNotAdopted        = "not_adopted"
	ProblemRevoked           = "revoked"
	ProblemMissingDependency = "missing_dependency"
	ProblemNotShipped        = "not_shipped"
	ProblemDuplicateName     = "duplicate_name"

	ReasonAINotAllowed         = "ai_not_allowed"
	ReasonNotHuman             = "not_human"
	ReasonSelfApproval         = "self_approval"
	ReasonNotPending           = "not_pending"
	ReasonNotProposer          = "not_proposer"
	ReasonUnknownLicense       = "unknown_license"
	ReasonNotCandidate         = "not_candidate"
	ReasonNotAdopted           = "not_adopted"
	ReasonTestsFailed          = "tests_failed"
	ReasonNotOffline           = "not_offline"
	ReasonTelemetryRequired    = "telemetry_required"
	ReasonLicenseBoundary      = "license_boundary"
	ReasonDependencyNotAdopted = "dependency_not_adopted"
	ReasonDependencyBoundary   = "dependency_boundary"
	ReasonUnpinnedVersion      = "unpinned_version"
	ReasonToolDependency       = "tool_dependency"
	ReasonNotReady             = "not_ready"
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

type Component struct {
	ID                string
	Name              string
	Version           string
	Commit            string
	Source            string
	License           string
	LicenseText       string
	Notice            string
	SBOM              string
	Artifact          string
	Patch             string
	Dependencies      []string
	Supersedes        string
	Offline           bool
	Telemetry         string
	TelemetryEvidence string
	Modification      string
	Distribution      string
	Tests             string
	Passed            bool
}

type Decision struct {
	Component string
	Reason    string
}

type Change struct {
	Kind      string
	Component Component
	Decision  Decision
}

type ProposalView struct {
	Number   uint64
	Change   Change
	State    string
	Proposer string
	Approver string
}

type ComponentView struct {
	Component Component
	State     string
	Reason    string
}

type Release struct {
	Version    string
	Commit     string
	Components []string
}

type Problem struct {
	Component string
	Problem   string
}

type Check struct {
	Ready    bool
	Problems []Problem
	Bundle   string
}

type ReleaseView struct {
	Release Release
	Bundle  string
	By      string
	At      time.Time
}
