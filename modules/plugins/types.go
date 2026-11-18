package plugins

import (
	"errors"
	"io"
	"time"
)

var (
	ErrInvalid  = errors.New("a request or an input is incomplete or malformed")
	ErrNotFound = errors.New("what the request names is not registered")
	ErrConflict = errors.New("another fact is already recorded under this identifier")
	ErrRejected = errors.New("a recorded state or rule refuses this request")
	ErrLimit    = errors.New("a configured limit or bound is exceeded")
	ErrStale    = errors.New("the state changed while a dependency was consulted")
)

const (
	KindHuman     = "human"
	KindAssistant = "assistant"
	KindSystem    = "system"

	ChangePublisher = "publisher"
	ChangeKey       = "key"
	ChangeVersion   = "version"
	ChangeWorkspace = "workspace"
	ChangeRuntime   = "runtime"
	ChangeInstall   = "install"
	ChangeUpgrade   = "upgrade"
	ChangeRollback  = "rollback"
	ChangeEnable    = "enable"
	ChangeRemove    = "remove"
	ChangeEndpoint  = "endpoint"

	MarkVulnerable = "vulnerable"
	MarkCompromise = "compromise"
	MarkRevocation = "revocation"

	ProposalPending   = "pending"
	ProposalApplied   = "applied"
	ProposalWithdrawn = "withdrawn"

	StateInstalled = "installed"
	StateEnabled   = "enabled"
	StateStopped   = "stopped"
	StateRemoved   = "removed"

	CapabilityReadEvents  = "read_events"
	CapabilityCallCommand = "call_command"
	CapabilityServeScreen = "serve_screen"

	ComponentTable  = "table"
	ComponentForm   = "form"
	ComponentDetail = "detail"
	ComponentChart  = "chart"

	DeleteOnRemoval   = "delete_on_removal"
	RetainForEvidence = "retain_for_evidence"

	MigrationNone        = "none"
	MigrationAdditive    = "additive"
	MigrationCopyForward = "copy_forward"

	ReasonAINotAllowed        = "ai_not_allowed"
	ReasonNotHuman            = "not_human"
	ReasonNotRuntime          = "not_runtime"
	ReasonSelfApproval        = "self_approval"
	ReasonNotPending          = "not_pending"
	ReasonNotProposer         = "not_proposer"
	ReasonWeakKey             = "weak_key"
	ReasonKeyLife             = "key_life"
	ReasonKeyMismatch         = "key_mismatch"
	ReasonKeyInactive         = "key_inactive"
	ReasonPublisherInactive   = "publisher_inactive"
	ReasonSignatureInvalid    = "signature_invalid"
	ReasonIncompatible        = "incompatible"
	ReasonVulnerable          = "vulnerable"
	ReasonRevoked             = "revoked"
	ReasonPublisherMismatch   = "publisher_mismatch"
	ReasonStorageChanged      = "storage_changed"
	ReasonIsolated            = "isolated"
	ReasonNotAllowedState     = "not_allowed_state"
	ReasonNamespaceTaken      = "namespace_taken"
	ReasonRuntimeTaken        = "runtime_taken"
	ReasonDirection           = "direction"
	ReasonNotDeclared         = "not_declared"
	ReasonFingerprintMismatch = "fingerprint_mismatch"
	ReasonDestinationInactive = "destination_inactive"
	ReasonDependency          = "dependency_unavailable"
	ReasonExpired             = "expired"
	ReasonUnknownToken        = "unknown_token"
	ReasonStaleToken          = "stale_token"
	ReasonScope               = "scope"
	ReasonNotEnabled          = "not_enabled"
	ReasonTimeout             = "timeout"
	ReasonOperator            = "operator"
	ReasonLicenseNotPermitted = "license_not_permitted"
)

type Refusal struct {
	Reason string
}

func (e Refusal) Error() string { return ErrRejected.Error() + ": " + e.Reason }

func (e Refusal) Unwrap() error { return ErrRejected }

type Destinations interface {
	Active(workspace, destination string) (bool, error)
}

type Config struct {
	Core         string
	Destinations Destinations
	Random       io.Reader
	Licenses     []string
}

type Limits struct {
	Publishers    int
	Keys          int
	KeyLife       time.Duration
	Plugins       int
	Versions      int
	Workspaces    int
	Installations int
	Tokens        int
	Fingerprints  int
	TokenLife     time.Duration
	Endpoints     int
	Proposals     int
}

type Actor struct {
	ID   string
	Kind string
}

type Publisher struct {
	ID   string
	Name string
}

type PublisherKey struct {
	Publisher string
	Version   string
	Public    []byte
	NotBefore time.Time
	NotAfter  time.Time
	Proof     []byte
}

type Capability struct {
	Kind   string
	Target string
}

type CoreRange struct {
	Min string
	Max string
}

type RuntimeLimits struct {
	Timeout     time.Duration
	MemoryMiB   int
	Concurrency int
}

type Storage struct {
	Namespace     string
	Owner         string
	RetentionDays int
	Deletion      string
	Migration     string
}

type Field struct {
	Name   string
	Label  string
	Source string
}

type Screen struct {
	ID        string
	Component string
	Label     string
	Fields    []Field
}

type Manifest struct {
	Plugin       string
	Version      string
	Publisher    string
	Key          string
	Digest       string
	License      string
	Core         CoreRange
	Capabilities []Capability
	Egress       []string
	Endpoints    []string
	Limits       RuntimeLimits
	Storage      Storage
	Screens      []Screen
}

type Workspace struct {
	ID           string
	Verification bool
	Runtime      string
}

type Endpoint struct {
	ID          string
	Fingerprint string
	Evidence    string
}

type Change struct {
	Kind      string
	Publisher Publisher
	Key       PublisherKey
	Manifest  Manifest
	Signature []byte
	Workspace Workspace
	Plugin    string
	Version   string
	Endpoint  Endpoint
}

type ProposalView struct {
	Number   uint64
	Change   Change
	State    string
	Proposer string
	Approver string
}

type Mark struct {
	By          string
	Reason      string
	ConfirmedBy string
}

type Target struct {
	Kind      string
	Publisher string
	Key       string
	Plugin    string
	Version   string
}

type PublisherView struct {
	Publisher  Publisher
	Revocation *Mark
	Keys       []KeyView
}

type KeyView struct {
	Version    string
	Public     []byte
	NotBefore  time.Time
	NotAfter   time.Time
	Retired    bool
	Compromise *Mark
}

type VersionView struct {
	Manifest   Manifest
	Signature  []byte
	Vulnerable *Mark
	Permitted  bool
}

type EndpointView struct {
	Endpoint Endpoint
	Approver string
}

type InstallationView struct {
	Workspace string
	Plugin    string
	Version   string
	State     string
	Reason    string
	Revision  uint64
	Endpoints []EndpointView
	Disposal  Storage
}

type Grant struct {
	Workspace string
	Plugin    string
	Version   string
	Digest    string
	Revision  uint64
	Scopes    []Capability
}
