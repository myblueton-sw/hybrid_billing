package decisionledger

import (
	"errors"
	"io"
	"time"
)

const (
	RecordLimit  = 2048
	LineLimit    = 4096
	FormatV1     = 1
	DirName      = "decisions"
	FileName     = "ledger.jsonl"
	MarkerName   = "decision-ledger.created"
	RelativeFile = DirName + "/" + FileName
	ReplacedTag  = ".replaced-"

	KindGenesis  = "genesis"
	KindRecovery = "recovery"
	KindDecision = "decision"

	RevokeGrant      = "revoke_grant"
	RevokeDelegation = "revoke_delegation"
	UnknownLedger    = "unknown"
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
}

type Options struct {
	Dir           string
	MaxBytes      int64
	AcceptNew     bool
	FailStartSync bool
	Clock         func() time.Time
	Random        io.Reader
}
