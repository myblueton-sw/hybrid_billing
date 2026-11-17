package accesschange

import (
	"errors"
	"time"
)

const (
	ChangeAdd    = "add"
	ChangeRevoke = "revoke"

	StatePending    = "pending"
	StateApplied    = "applied"
	StateRejected   = "rejected"
	StateWithdrawn  = "withdrawn"
	StateExpired    = "expired"
	StateSuperseded = "superseded"

	RevocationReceived    = "received"
	RevocationConfirmed   = "deny_confirmed"
	RevocationCleaning    = "cleanup_running"
	RevocationComplete    = "revocation_complete"
	RevocationUnconfirmed = "decision_unconfirmed"

	JobRunning   = "running"
	JobWaiting   = "waiting"
	JobSucceeded = "succeeded"

	OriginProcess = "this_process"
	OriginReplay  = "ledger_replay"

	TargetGrant      = "grant"
	TargetDelegation = "delegation"

	ItemDone          = "done"
	ItemNotApplicable = "not_applicable"
	ItemOutside       = "outside_scope"
	ItemNotStarted    = "not_started"

	ReplayReapplied = "reapplied"
	ReplayAbsent    = "absent"
	ReplayProtected = "protected"

	CommitFailed  = "commit_failed"
	CommitUnknown = "commit_unknown"

	RequestPrefix    = "access-request-"
	RevocationPrefix = "revocation-"

	digestLabel = "access_change_v1"
)

type Reason string

const (
	ReasonNotHuman        Reason = "not_human"
	ReasonProposerSubject Reason = "proposer_is_subject"
	ReasonProposerHolder  Reason = "proposer_is_holder"
	ReasonSelfDecision    Reason = "self_decision"
	ReasonApproverSubject Reason = "approver_is_subject"
	ReasonHolderReject    Reason = "holder_reject"
	ReasonNotAllowed      Reason = "not_allowed"
	ReasonNotProposer     Reason = "not_proposer"
	ReasonNotPending      Reason = "not_pending"
	ReasonExpired         Reason = "expired"
	ReasonDigest          Reason = "digest_mismatch"
	ReasonSuperseded      Reason = "superseded"
	ReasonPendingTarget   Reason = "pending_for_target"
	ReasonLimit           Reason = "pending_limit"
	ReasonExpiry          Reason = "expiry_passed"
	ReasonNotFound        Reason = "not_found"
)

type Refusal struct{ Reason Reason }

func (r Refusal) Error() string { return "access change refused: " + string(r.Reason) }

var ErrInvalid = errors.New("invalid access change input")

type Limits struct {
	Lifetime        time.Duration
	PerProposer     int
	PerWorkspace    int
	EndedKept       int
	RevocationsKept int
}

func (l Limits) valid() bool {
	return l.Lifetime > 0 && l.PerProposer >= 1 && l.PerWorkspace >= l.PerProposer && l.EndedKept >= 1 && l.RevocationsKept >= 1
}

type Change struct {
	Kind         string
	Subject      string
	Action       string
	ResourceKind string
	Fields       []string
	Customer     string
	Days         int
	Grant        string
}

type Actor struct {
	User  string
	Human bool
}

type Proposal struct {
	Workspace    string
	Change       Change
	Holder       string
	Basis        string
	CeilingUntil time.Time
	Proposer     Actor
	Allowed      bool
	Rationale    string
}

type Standing struct {
	Basis   string
	Allowed bool
}

type View struct {
	Number         uint64
	ID             string
	Workspace      string
	Change         Change
	Holder         string
	Basis          string
	Digest         string
	State          string
	Proposer       string
	ProposedAt     time.Time
	ExpiresAt      time.Time
	Rationale      string
	DecidedBy      string
	DecidedAt      time.Time
	DecisionReason string
	Result         string
}

type Target struct {
	Kind    string
	ID      string
	Version uint64
}

type Denied struct {
	Generation uint64
	Ended      []string
}

type Item struct {
	Item   string
	State  string
	Reason string
	Refs   []string
}

type Receipt struct {
	Sequence  uint64
	Watermark string
}

type Entry struct {
	Workspace      string
	Target         Target
	Revocation     string
	Request        string
	RequestDigest  string
	RequestedBy    string
	ApprovedBy     string
	DenyGeneration uint64
	Ended          []string
}

type Ledger interface {
	Commit(Entry) (Receipt, error)
}

type Revocation struct {
	Workspace     string
	Target        Target
	RequestedBy   string
	ApprovedBy    string
	Request       string
	RequestDigest string
}

type Effects struct {
	Deny    func() (Denied, error)
	Ledger  Ledger
	Cleanup func(Denied) []Item
	Planned []Item
}

type Transition struct {
	State string
	At    time.Time
}

type RevocationView struct {
	ID             string
	Origin         string
	RecordedAt     time.Time
	Workspace      string
	Target         Target
	TargetPresent  bool
	Replay         string
	State          string
	History        []Transition
	RequestedBy    string
	ApprovedBy     string
	Request        string
	RequestDigest  string
	DenyAt         time.Time
	DenyGeneration uint64
	Ended          []string
	Receipt        Receipt
	Reason         string
	Cleanup        []Item
	CompletedAt    time.Time
}

type Restored struct {
	ID         string
	RecordedAt time.Time
	Revocation Revocation
	Generation uint64
	Ended      []string
	Receipt    Receipt
	Replay     string
	Present    bool
	Cleanup    []Item
}

type Applier interface {
	Revoke(Revocation, Effects) (RevocationView, error)
}

func JobState(state string) (string, string) {
	switch state {
	case RevocationComplete:
		return JobSucceeded, ""
	case RevocationUnconfirmed:
		return JobWaiting, RevocationUnconfirmed
	}
	return JobRunning, ""
}
