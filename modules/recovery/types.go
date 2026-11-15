package recovery

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

	ChangeInstallation = "installation"
	ChangeWriter       = "writer"
	ChangeExternal     = "external"
	ChangeVerifier     = "verifier"
	ChangeBegin        = "begin"
	ChangeUnlock       = "unlock"
	ChangePrune        = "prune"
	ChangeAbandon      = "abandon"

	ProposalPending   = "pending"
	ProposalApplied   = "applied"
	ProposalWithdrawn = "withdrawn"

	StateServing    = "serving"
	StateLocked     = "locked"
	StateRecovering = "recovering"

	WriterERP      = "erp"
	WriterIssuance = "issuance"
	WriterDelivery = "delivery"
	WriterPayment  = "payment"
	WriterMessage  = "message"

	OutcomeConfirmed = "confirmed"
	OutcomeFailed    = "failed"
	OutcomeUnknown   = "unknown"

	DomainDatabase    = "database"
	DomainEvidence    = "evidence"
	DomainKeys        = "keys"
	DomainPolicies    = "policies"
	DomainModules     = "modules"
	DomainNumbering   = "numbering"
	DomainSubmissions = "submissions"
	DomainOutbox      = "outbox"
	DomainDeletions   = "deletions"

	DomainPending  = "pending"
	DomainVerified = "verified"
	DomainMismatch = "mismatch"

	PhaseVerify  = "verify"
	PhaseReapply = "reapply"
	PhaseReady   = "ready"

	ItemSubmitted = "submitted"
	ItemIssued    = "issued"
	ItemDelivered = "delivered"
	ItemPaid      = "paid"

	ExternalPending    = "pending"
	ExternalReconciled = "reconciled"

	EventDelete       = "delete"
	EventPurge        = "purge"
	EventHold         = "hold"
	EventExtend       = "extend"
	EventRevokeKey    = "revoke_key"
	EventRevokeAccess = "revoke_access"

	ReasonAINotAllowed     = "ai_not_allowed"
	ReasonNotHuman         = "not_human"
	ReasonSelfApproval     = "self_approval"
	ReasonNotPending       = "not_pending"
	ReasonNotProposer      = "not_proposer"
	ReasonIdentityTaken    = "identity_taken"
	ReasonNotWriter        = "not_writer"
	ReasonNotVerifier      = "not_verifier"
	ReasonLocked           = "locked"
	ReasonRecovering       = "recovering"
	ReasonFenced           = "fenced"
	ReasonExpired          = "expired"
	ReasonFencedKey        = "fenced_key"
	ReasonBelowHighWater   = "below_high_water"
	ReasonGeneration       = "generation"
	ReasonManifestMismatch = "manifest_mismatch"
	ReasonOpenRecovery     = "open_recovery"
	ReasonPruned           = "pruned"
	ReasonNotRecovering    = "not_recovering"
	ReasonNoRestorePoint   = "no_restore_point"
	ReasonPhase            = "phase"
	ReasonOutOfOrder       = "out_of_order"
	ReasonCoverage         = "coverage"
	ReasonMustApply        = "must_apply"
	ReasonIncomplete       = "incomplete"
	ReasonLost             = "lost"
	ReasonOpenCalls        = "open_calls"
	ReasonKindCoverage     = "kind_coverage"
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

type Installation struct {
	ID     string
	Tenant string
	Skew   time.Duration
}

type Writer struct {
	Installation string
	ID           string
	Kind         string
	MaxCall      time.Duration
}

type External struct {
	Installation string
	ID           string
	Kind         string
}

type Verifier struct {
	Installation string
	ID           string
}

type Chunk struct {
	Start    uint64
	End      uint64
	Checksum string
}

type Inventory struct {
	Domain string
	Items  uint64
	Chunks []Chunk
}

type RestorePoint struct {
	Installation   string
	Generation     uint64
	Backup         string
	ManifestDigest string
	AsOf           time.Time
	Manifest       []Inventory
}

type Unlock struct {
	Installation string
	Generation   uint64
	Lost         string
}

type Prune struct {
	Installation string
	Before       time.Time
}

type Abandon struct {
	Installation string
	Recovery     string
	Generation   uint64
}

type Change struct {
	Kind         string
	Installation Installation
	Writer       Writer
	External     External
	Verifier     Verifier
	Begin        RestorePoint
	Unlock       Unlock
	Prune        Prune
	Abandon      Abandon
}

type ProposalView struct {
	Number   uint64
	Change   Change
	State    string
	Proposer string
	Approver string
}

type Token struct {
	Generation uint64
	Sequence   uint64
	Expiry     time.Time
}

type Target struct {
	Workspace string
	Key       string
	Series    string
	Number    int64
}

type CallView struct {
	ID       string
	Writer   string
	Target   Target
	Deadline time.Time
	Outcome  string
}

type Item struct {
	Workspace string
	Key       string
	Kind      string
	At        time.Time
	Series    string
	Number    int64
}

type Page struct {
	Number  int
	From    time.Time
	Through time.Time
	Final   bool
	Rotated bool
	Items   []Item
}

type Event struct {
	Kind   string
	Domain string
	Key    string
}

type EventView struct {
	Sequence uint64
	Event    Event
	At       time.Time
}

type DomainStatus struct {
	Domain   string
	Status   string
	Mismatch []int
}

type ExternalStatus struct {
	ID     string
	Kind   string
	Status string
}

type RecoveryView struct {
	ID         string
	Phase      string
	State      string
	Generation uint64
	StoppedAt  time.Time
	AsOf       time.Time
	Backup     string
	Domains    []DomainStatus
	Externals  []ExternalStatus
	Pending    []EventView
	Lost       []uint64
	Open       int
	OpenCalls  []CallView
	Unknown    []Target
	Skipped    int

	ManifestDigest  string
	PointGeneration uint64
}

type History struct {
	Recovery       string
	ManifestDigest string
	Generation     uint64
	Outcome        string
	Lost           int
}
