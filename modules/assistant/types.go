package assistant

import (
	"errors"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

var (
	ErrInvalid  = errors.New("a request or an input is incomplete or malformed")
	ErrNotFound = errors.New("what the request names does not exist in this workspace")
	ErrConflict = errors.New("another fact is already recorded under this identifier")
	ErrRejected = errors.New("a recorded state or rule refuses this request")
	ErrLimit    = errors.New("a configured limit or bound is exceeded")
)

const (
	maxIDBytes         = 128
	maxTextBytes       = 2000
	maxWorkspaces      = 100000
	maxTools           = 10000
	maxJobs            = 1000000
	maxConversations   = 1000000
	maxCards           = 1000
	maxCalls           = 100000
	maxTargets         = 100000
	maxFields          = 100
	maxBytes           = 4096
	minDuration        = time.Minute
	maxDuration        = 30 * 24 * time.Hour
	minRetention       = time.Hour
	maxRetention       = 3650 * 24 * time.Hour
	maxConversationAge = 365 * 24 * time.Hour
	maxAttempts        = 10
	callTag            = "assistant_call_v1"
	cardTag            = "assistant_card_v1"
	confirmTag         = "assistant_confirm_v1"

	KindHuman     = "human"
	KindAssistant = "assistant"
	KindSystem    = "system"

	StageRead     = "read"
	StageDraft    = "draft"
	StageSimulate = "simulate"
	StageRequest  = "request"

	StateProposed  = "proposed"
	StateActive    = "active"
	StateWithdrawn = "withdrawn"

	CardPending   = "pending"
	CardConfirmed = "confirmed"
	CardUncertain = "uncertain"
	CardDeclined  = "declined"
	CardExpired   = "expired"

	EntryCall    = "call"
	EntryCard    = "card"
	EntryConfirm = "confirm"
	EntryDecline = "decline"

	OutcomeDeclined = "declined"

	ReasonAINotAllowed       = "ai_not_allowed"
	ReasonNotHuman           = "not_human"
	ReasonNotAssistant       = "not_assistant"
	ReasonNotSystem          = "not_system"
	ReasonSelfApproval       = "self_approval"
	ReasonUnsupported        = "unsupported_command"
	ReasonStage              = "stage_not_allowed"
	ReasonProtected          = "protected_command"
	ReasonNotExposed         = "not_exposed"
	ReasonRequestOnly        = "request_only"
	ReasonNotRequest         = "not_request"
	ReasonNotOwner           = "not_owner"
	ReasonNotConfirmer       = "not_confirmer"
	ReasonJobClosed          = "job_closed"
	ReasonConversation       = "conversation_expired"
	ReasonNotAttached        = "not_attached"
	ReasonBudget             = "budget_exhausted"
	ReasonModelAmount        = "model_amount"
	ReasonDigestChanged      = "digest_changed"
	ReasonAlreadyDecided     = "already_decided"
	ReasonOutcomeUncertain   = "outcome_uncertain"
	ReasonCardExpired        = "card_expired"
	ReasonNotPending         = "not_pending"
	ReasonExposureChanged    = "exposure_changed"
	ReasonConversationExpiry = "conversation_retention"
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

type Commands interface {
	Catalogue() []commands.Entry
	Execute(commands.Call) commands.Result
}

type Config struct {
	Commands  Commands
	Client    string
	Protected []string
}

type Limits struct {
	Workspaces    int
	Tools         int
	Jobs          int
	Conversations int
	Cards         int
	Calls         int
	Targets       int
	Duration      time.Duration
	Retention     time.Duration
	Fields        int
	Bytes         int
}

type Tool struct {
	Command string
	Stage   string
	Results []string
}

type ToolView struct {
	Tool     Tool
	Revision int
	State    string
	Proposer string
	Approver string
}

type Conversation struct {
	ID      string
	Owner   string
	Expires time.Time
}

type Job struct {
	ID      string
	Purpose string
}

type Entry struct {
	Sequence  int
	Kind      string
	Tool      string
	Target    string
	Card      string
	Operation string
	Outcome   string
	Replay    bool
	At        time.Time
}

type JobView struct {
	Job      Job
	Owner    string
	Created  time.Time
	Closed   bool
	ClosedAt time.Time
	Calls    int
	Targets  int
	Cards    int
	Ledger   []Entry
}

type Invocation struct {
	Conversation string
	Tool         string
	Target       string
	Expected     string
	Input        []commands.Value
	Answer       gate.Answer
}

type Reply struct {
	Status    string
	Code      string
	Operation string
	Replay    bool
	Fields    map[string]string
	Truncated []string
	Calls     int
	Remaining int
	Targets   int
}

type Card struct {
	ID       string
	Tool     string
	Target   string
	Expected string
	Input    []commands.Value
	Note     string
}

type CardView struct {
	Card      Card
	Job       string
	Digest    string
	State     string
	Confirmer string
	Operation string
	Code      string
	Attempts  int
	Decliner  string
	Reason    string
}

type Confirmation struct {
	Digest    string
	Rationale string
	Answer    gate.Answer
}
