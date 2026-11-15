package sandbox

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

	ChangeEnvironment = "environment"
	ChangeDestination = "destination"
	ChangeSchema      = "schema"

	ProposalPending   = "pending"
	ProposalApplied   = "applied"
	ProposalWithdrawn = "withdrawn"

	ModeDemo       = "demo"
	ModeProduction = "production"

	DestinationERP          = "erp"
	DestinationMailbox      = "mailbox"
	DestinationWebhook      = "webhook"
	DestinationPayment      = "payment"
	DestinationProvider     = "provider"
	DestinationNotification = "notification"

	FieldEnum        = "enum"
	FieldInteger     = "integer"
	FieldFlag        = "flag"
	FieldDestination = "destination"

	TargetDatabase   = "database"
	TargetKeyRing    = "keyring"
	TargetEvidence   = "evidence"
	TargetNumbers    = "numbers"
	TargetConnection = "connection"
	TargetVolume     = "volume"

	OriginReal      = "real"
	OriginPublic    = "public"
	OriginConverted = "converted"
	OriginSynthetic = "synthetic"

	MarkingDemo = "DEMO"

	ReasonAINotAllowed     = "ai_not_allowed"
	ReasonNotHuman         = "not_human"
	ReasonSelfApproval     = "self_approval"
	ReasonNotPending       = "not_pending"
	ReasonNotProposer      = "not_proposer"
	ReasonNamespaceShared  = "namespace_shared"
	ReasonNoProduction     = "no_production"
	ReasonNotDemo          = "not_demo"
	ReasonNotReserved      = "not_reserved"
	ReasonNotMock          = "not_mock"
	ReasonMockDestination  = "mock_destination"
	ReasonSensitiveKey     = "sensitive_key"
	ReasonUndeclared       = "undeclared"
	ReasonOutsideEnv       = "outside_env"
	ReasonStale            = "stale"
	ReasonCrossEnvironment = "cross_environment"
	ReasonForeignNumber    = "foreign_number"
	ReasonNotReal          = "not_real"
	ReasonDemoPrefix       = "demo_prefix"
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

type Environment struct {
	ID           string
	Mode         string
	Database     string
	KeyRing      string
	EvidencePath string
	NumberPrefix string
}

type Destination struct {
	ID          string
	Environment string
	Kind        string
	Host        string
	Port        int
}

type FieldSpec struct {
	Key    string
	Kind   string
	Values []string
}

type Schema struct {
	ID     string
	Fields []FieldSpec
}

type Change struct {
	Kind        string
	Environment Environment
	Destination Destination
	Schema      Schema
}

type ProposalView struct {
	Number   uint64
	Change   Change
	State    string
	Proposer string
	Approver string
}

type DestinationView struct {
	Destination Destination
	Enabled     bool
}

type Request struct {
	Environment string
	Kind        string
	Host        string
	Port        int
}

type Setting struct {
	Key   string
	Value string
}

type Config struct {
	Environment string
	Schema      string
	Settings    []Setting
}

type Target struct {
	Kind string
	Name string
}

type Plan struct {
	Environment string
	Key         string
	Targets     []Target
}

type ResetView struct {
	Environment string
	ID          uint64
	Key         string
	Targets     []Target
	Done        []bool
	Completed   bool
	By          string
	At          time.Time
}

type Input struct {
	Environment string
	Source      string
	Database    string
	Origin      string
	Number      string
}

type LabelView struct {
	Environment string
	Demo        bool
	Marking     string
	Origin      string
	Source      string
}
