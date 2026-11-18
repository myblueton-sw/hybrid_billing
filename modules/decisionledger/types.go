package decisionledger

import (
	"errors"
	"io"
	"time"
)

const (
	RecordLimit   = 2048
	MaxGeneration = 1<<32 - 1
	LineLimit     = 4096
	FormatV1      = 1
	DirName       = "decisions"
	FileName      = "ledger.jsonl"
	MarkerName    = "decision-ledger.created"
	RelativeFile  = DirName + "/" + FileName
	ReplacedTag   = ".replaced-"

	KindGenesis  = "genesis"
	KindRecovery = "recovery"
	KindDecision = "decision"

	RevokeGrant      = "revoke_grant"
	RevokeDelegation = "revoke_delegation"
	TrustActivation  = "trust_activation"
	UnknownLedger    = "unknown"

	OriginActivation = "activation"
	OriginCarried    = "carried"
)

type State string

const (
	StateOK        State = "ok"
	StateRecovered State = "recovered"
	StateIsolated  State = "isolated"
	StateStopped   State = "stopped"
	StateFull      State = "full"
)

type Isolation string

const (
	IsolationMissing   Isolation = "missing"
	IsolationIntegrity Isolation = "integrity_failed"
	IsolationMismatch  Isolation = "id_mismatch"
	IsolationOrder     Isolation = "epoch_order"
)

const (
	ReasonCommitFailed  = "commit_failed"
	ReasonCommitUnknown = "commit_unknown"
	ReasonIsolated      = "ledger_isolated"
	ReasonStopped       = "ledger_stopped"
	ReasonFull          = "ledger_full"
)

var (
	ErrLocked           = errors.New("the decision ledger is in use by another process")
	ErrUnsafe           = errors.New("the decision ledger location is not owner-only")
	ErrBound            = errors.New("the decision ledger is larger than its configured bound")
	ErrNothingToReplace = errors.New("there is no isolated decision ledger to replace")
	ErrInvalid          = errors.New("invalid decision ledger options")
	ErrGenerations      = errors.New("the decision ledger generations are exhausted")
	errStartSync        = errors.New("start sync refused by option")
)

type CommitError struct{ Reason string }

func (e *CommitError) Error() string { return "decision not committed: " + e.Reason }

func (e *CommitError) CommitReason() string { return e.Reason }

type Decision struct {
	Workspace        string
	Decision         string
	Target           string
	TargetVersion    uint64
	Revocation       string
	Request          string
	RequestDigest    string
	RequestedBy      string
	ApprovedBy       string
	DenyGeneration   uint64
	EndedDelegations []string
	Trust            *Trust
}

type StepRule struct {
	Action        string
	MaxAgeSeconds int64
	MFA           bool
}

type Config struct {
	Methods       []string
	MFA           string
	MaxAgeSeconds int64
	StepUp        []StepRule
}

type Trust struct {
	Change           string
	ChangeRevision   uint64
	ChangeDigest     string
	EvidenceDigest   string
	OldEpoch         uint64
	NewEpoch         uint64
	OldPolicy        uint64
	NewPolicy        uint64
	OldDigest        string
	NewDigest        string
	Config           Config
	ProposedBy       string
	ValidatedBy      string
	ApprovedBy       string
	ActivatedBy      string
	ApprovedAt       time.Time
	RunStartedAt     time.Time
	InvalidateBelow  uint64
	AffectedSessions uint64
	Origin           string
	CarriedFrom      string
	CarriedLine      int
}

type Retained struct {
	File         string
	Generation   uint64
	GenesisRead  bool
	HighestEpoch uint64
}

type Prior struct {
	Isolation        Isolation
	ProblemLine      int
	MarkerUsable     bool
	MarkerID         string
	MarkerGeneration uint64
	LedgerID         string
	Generation       uint64
	Verified         []Record
	Retained         []Retained
	Unverified       uint64
	UnverifiedRead   bool
}

type Successor struct {
	Generation       uint64
	TrustReset       bool
	GenerationStated bool
	CarriedFrom      string
	ReplacedLine     int
	Carry            []Decision
}

type Record struct {
	Sequence  uint64
	Watermark string
	At        time.Time
	Decision  Decision
}

type Receipt struct {
	Sequence  uint64
	Watermark string
}

type Recovery struct {
	At             time.Time
	TruncatedBytes int64
}

type Status struct {
	State         State
	Isolation     Isolation
	LedgerID      string
	CreatedAt     time.Time
	Replaces      string
	ReplaceReason string
	ReplacedFile  string
	Records       uint64
	Watermark     string
	CheckedAt     time.Time
	ProblemLine   int
	Recovery      *Recovery
	NotDurable    bool
	File          string
	Generation    uint64
	MarkerUsable  bool
	Genesis       Successor
}

type Options struct {
	Dir           string
	MaxBytes      int64
	AcceptNew     bool
	FailStartSync bool
	FailSync      func() bool
	Clock         func() time.Time
	Random        io.Reader
	Successor     func(Prior) (Successor, error)
}
