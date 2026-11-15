package licensemode

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

	ChangeKey        = "key"
	ChangeRetire     = "retire_key"
	ChangeCompromise = "compromise_key"
	ChangeReset      = "reset_high_water"

	ProposalPending   = "pending"
	ProposalApplied   = "applied"
	ProposalWithdrawn = "withdrawn"

	ModeNoncommercial = "noncommercial"
	ModeEvaluation    = "evaluation"
	ModeCommercial    = "commercial"

	DocEvaluation = "evaluation"
	DocCommercial = "commercial"

	ChannelDirect      = "direct"
	ChannelMarketplace = "marketplace"

	MetricInstallations = "installations"
	MetricSeats         = "seats"
	MetricTenants       = "tenants"
	MetricDocuments     = "documents"

	OutcomeInvalid         = "invalid"
	OutcomeUnverifiable    = "unverifiable"
	OutcomeWrongKind       = "wrong_kind"
	OutcomeScopeMismatch   = "scope_mismatch"
	OutcomeTimeUnavailable = "time_unavailable"
	OutcomeClockAnomaly    = "clock_anomaly"
	OutcomeNotYetValid     = "not_yet_valid"
	OutcomeExpired         = "expired"
	OutcomeSuperseded      = "superseded"
	OutcomeDuplicateRight  = "duplicate_right"
	OutcomeBadSupersedes   = "bad_supersedes"
	OutcomeAccepted        = "accepted"
	OutcomeRepeat          = "repeat"

	StateNoncommercial     = "noncommercial"
	StateEvaluationPending = "evaluation_pending"
	StateEvaluationActive  = "evaluation_active"
	StateEvaluationExpired = "evaluation_expired"
	StateCommercialPending = "commercial_pending"
	StateCommercialActive  = "commercial_active"
	StateCommercialExpired = "commercial_expired"
	StateUnverified        = "unverified"
	StateUnselected        = "unselected"

	ReasonKeyCompromised   = "key_compromised"
	ReasonLocalClockBehind = "local_clock_behind"
	ReasonResetTooEarly    = "reset_too_early"

	ActionRegister = "register"
	ActionContact  = "contact"

	ReasonAINotAllowed     = "ai_not_allowed"
	ReasonNotHuman         = "not_human"
	ReasonSelfApproval     = "self_approval"
	ReasonNotPending       = "not_pending"
	ReasonNotProposer      = "not_proposer"
	ReasonPrivateKey       = "private_key"
	ReasonNoProof          = "no_proof"
	ReasonRetirePast       = "retire_past"
	ReasonCompromised      = "compromised"
	ReasonTermsMismatch    = "terms_mismatch"
	ReasonModeMismatch     = "mode_mismatch"
	ReasonStaleGrant       = "stale_grant"
	ReasonAlreadyEvaluated = "already_evaluated"
	ReasonRetired          = "retired"
)

var Metrics = []string{MetricDocuments, MetricInstallations, MetricSeats, MetricTenants}

type Refusal struct {
	Reason string
}

func (e Refusal) Error() string { return ErrRejected.Error() + ": " + e.Reason }

func (e Refusal) Unwrap() error { return ErrRejected }

type Authority interface {
	Now() (time.Time, bool)
}

type Config struct {
	Authority Authority
}

type Actor struct {
	ID   string
	Kind string
}

type Key struct {
	ID          string
	Issuer      string
	Public      string
	NotBefore   int64
	NotAfter    int64
	AcceptUntil int64
	Proof       string
}

type Retire struct {
	Key string
	At  int64
}

type Reset struct {
	At     int64
	Reason string
}

type Change struct {
	Kind       string
	Key        Key
	Retire     Retire
	Compromise string
	Reset      Reset
}

type ProposalView struct {
	Number   uint64
	Change   Change
	State    string
	Proposer string
	Approver string
}

type Claims struct {
	Kind         string
	ID           string
	Issuer       string
	KeyID        string
	IssuedAt     int64
	SubjectID    string
	Subject      string
	Agreement    string
	Supersedes   string
	Product      string
	Installation string
	Scope        []string
	Channel      string
	Metric       string
	Start        int64
	End          int64
	Perpetual    bool
	Terms        string
	ActivateBy   int64
}

type Signed struct {
	Body      []byte
	Signature []byte
}

type Acceptance struct {
	Actor string
	Terms string
}

type KeyView struct {
	Key         Key
	Compromised bool
	Retired     bool
}

type Activation struct {
	Grant      string
	Subject    string
	Terms      string
	Scope      []string
	AcceptedBy string
	Start      time.Time
	End        time.Time
	KeyID      string
}

type ModeChange struct {
	Mode string
	By   string
	At   time.Time
}

type Submission struct {
	Outcome string
	Digest  string
	At      time.Time
}

type StatusView struct {
	Installation   string
	Mode           string
	State          string
	Reason         string
	Warning        bool
	Actions        []string
	Entitlement    *Claims
	Activation     *Activation
	Remaining      time.Duration
	LastOutcome    string
	LastVerified   string
	LastVerifiedAt time.Time
}
