package migration

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

	ChangeSource   = "source"
	ChangeImporter = "importer"
	ChangeMapping  = "mapping"
	ChangeBatch    = "batch"
	ChangeAccept   = "accept"

	ProposalPending   = "pending"
	ProposalApplied   = "applied"
	ProposalWithdrawn = "withdrawn"

	DocumentInvoice = "invoice"
	DocumentCredit  = "credit_note"
	DocumentDebit   = "debit_note"
	DocumentReceipt = "receipt"

	KeyCertificate = "certificate"
	KeyPublic      = "public_key"

	BatchOpen     = "open"
	BatchVerified = "verified"
	BatchMismatch = "mismatch"
	BatchMissing  = "evidence_missing"
	BatchAccepted = "accepted"
	BatchRejected = "rejected"

	ClassNew              = "new"
	ClassDuplicate        = "duplicate"
	ClassConflict         = "conflict"
	ClassNumberConflict   = "number_conflict"
	ClassExternalConflict = "external_conflict"

	ReasonAINotAllowed        = "ai_not_allowed"
	ReasonNotHuman            = "not_human"
	ReasonSelfApproval        = "self_approval"
	ReasonNotPending          = "not_pending"
	ReasonNotProposer         = "not_proposer"
	ReasonIdentityTaken       = "identity_taken"
	ReasonWorkspaceClaimed    = "workspace_claimed"
	ReasonWorkspaceNotAllowed = "workspace_not_allowed"
	ReasonNotImporter         = "not_importer"
	ReasonBatchOpen           = "batch_open"
	ReasonNotOpen             = "not_open"
	ReasonNotVerified         = "not_verified"
	ReasonUnmappedSeries      = "unmapped_series"
	ReasonEffectChanged       = "effect_changed"
	ReasonSecretMaterial      = "secret_material"
	ReasonDependency          = "dependency_unavailable"
	ReasonStale               = "stale"
)

type Refusal struct {
	Reason string
}

func (e Refusal) Error() string { return ErrRejected.Error() + ": " + e.Reason }

func (e Refusal) Unwrap() error { return ErrRejected }

type Evidence interface {
	Present(tenant, digest string) (bool, error)
}

type Config struct {
	Evidence Evidence
}

type Actor struct {
	ID   string
	Kind string
}

type Source struct {
	ID         string
	Tenant     string
	Importer   string
	Workspaces []string
}

type Mapping struct {
	Source       string
	SourceSeries string
	Workspace    string
	Series       string
}

type BatchSpec struct {
	Source string
	ID     string
	Count  int
	Digest string
}

type Accept struct {
	Source string
	Batch  string
	Effect string
}

type Change struct {
	Kind    string
	Source  Source
	Mapping Mapping
	Batch   BatchSpec
	Accept  Accept
}

type ProposalView struct {
	Number   uint64
	Change   Change
	State    string
	Proposer string
	Approver string
}

type Document struct {
	SourceID     string
	Kind         string
	Workspace    string
	SourceSeries string
	Number       int64
	Display      string
	IssuedAt     time.Time
	Currency     string
	Total        int64
	Content      string
	Delivered    bool
	Submitted    bool
	ExternalKey  string
}

type KeyReference struct {
	SourceID string
	Kind     string
	DER      []byte
}

type Item struct {
	Document *Document
	Key      *KeyReference
}

type KeyView struct {
	SourceID    string
	Kind        string
	DER         []byte
	Fingerprint string
}

type Classified struct {
	SourceID string
	Class    string
	Incoming string
	Stored   string
}

type SeriesChange struct {
	Workspace string
	Series    string
	Number    int64
}

type BatchView struct {
	Source   string
	ID       string
	State    string
	Count    int
	Digest   string
	Computed string
	Classes  map[string]int
	Effect   string
	Series   []SeriesChange
	Printed  int
	Keys     int
	By       string
	Reason   string
}

type Export struct {
	HighWater []SeriesChange
	Keys      []string
	Printed   []string
}
