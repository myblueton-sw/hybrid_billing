package support

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

	ChangeSite        = "site"
	ChangeSchema      = "schema"
	ChangeDestination = "destination"
	ChangeUpload      = "upload"

	ProposalPending   = "pending"
	ProposalApplied   = "applied"
	ProposalWithdrawn = "withdrawn"

	SourceLog        = "log"
	SourceConfig     = "config"
	SourceDiagnostic = "diagnostic"
	SourceMetric     = "metric"
	SourceVersion    = "version"

	ClassEnum       = "enum"
	ClassInteger    = "integer"
	ClassDuration   = "duration"
	ClassTimestamp  = "timestamp"
	ClassVersion    = "version"
	ClassHost       = "host"
	ClassAddress    = "address"
	ClassEmail      = "email"
	ClassUser       = "user"
	ClassIdentifier = "identifier"

	StateDraft     = "draft"
	StatePreviewed = "previewed"

	DropUnknownSource = "unknown_source"
	DropUndeclared    = "undeclared"
	DropWrongShape    = "wrong_shape"
	DropOutOfRange    = "out_of_range"
	DropDuplicateKey  = "duplicate_key"
	DropNoFields      = "no_fields"

	ReasonAINotAllowed        = "ai_not_allowed"
	ReasonNotHuman            = "not_human"
	ReasonSelfApproval        = "self_approval"
	ReasonNotPending          = "not_pending"
	ReasonNotProposer         = "not_proposer"
	ReasonSensitiveKey        = "sensitive_key"
	ReasonDestinationDisabled = "destination_disabled"
	ReasonEmpty               = "empty"
	ReasonExpired             = "expired"
	ReasonNotPreviewed        = "not_previewed"
	ReasonDigestMismatch      = "digest_mismatch"
	ReasonUploadDisabled      = "upload_disabled"
)

type Refusal struct {
	Reason string
}

func (e Refusal) Error() string { return ErrRejected.Error() + ": " + e.Reason }

func (e Refusal) Unwrap() error { return ErrRejected }

type Limits struct {
	Sites        int
	Destinations int
	Bundles      int
	SiteBytes    int64
	Bytes        int64
	BundleLife   time.Duration
	Proposals    int
}

type Actor struct {
	ID   string
	Kind string
}

type Site struct {
	ID     string
	Tenant string
}

type FieldSpec struct {
	Key    string
	Class  string
	Values []string
}

type Schema struct {
	Source string
	Fields []FieldSpec
}

type Destination struct {
	ID   string
	Host string
	Port int
}

type Upload struct {
	Site        string
	Destination string
}

type Change struct {
	Kind        string
	Site        Site
	Schema      Schema
	Destination Destination
	Upload      Upload
}

type ProposalView struct {
	Number   uint64
	Change   Change
	State    string
	Proposer string
	Approver string
}

type Field struct {
	Key      string
	Int      int64
	Text     string
	Time     time.Time
	Duration time.Duration
}

type Item struct {
	Source string
	Fields []Field
}

type Rendered struct {
	Key   string
	Class string
	Value string
}

type Kept struct {
	Number int
	Source string
	Fields []Rendered
}

type Drop struct {
	Source string
	Reason string
	Count  int
}

type Summary struct {
	Kept       int
	Pseudonyms map[string]int
	Drops      []Drop
}

type BundleView struct {
	ID      string
	Site    string
	State   string
	Items   []Kept
	Summary Summary
	Digest  string
	Bytes   []byte
	Expiry  time.Time
}

type UploadRecord struct {
	Destination string
	Digest      string
	By          string
	At          time.Time
}
