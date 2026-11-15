package installer

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
)

const (
	KindHuman     = "human"
	KindAssistant = "assistant"
	KindSystem    = "system"

	ChangeKey         = "key"
	ChangeRelease     = "release"
	ChangeSite        = "site"
	ChangeProbe       = "probe"
	ChangeConfig      = "config"
	ChangeDestination = "destination"
	ChangePlan        = "plan"

	ProposalPending   = "pending"
	ProposalApplied   = "applied"
	ProposalWithdrawn = "withdrawn"

	MarkRelease    = "release"
	MarkCompromise = "compromise"

	ArtifactContainer = "container_image"
	ArtifactArchive   = "archive"
	ArtifactPackage   = "package"

	ProtocolTCP = "tcp"
	ProtocolUDP = "udp"

	PermissionRunService   = "run_service"
	PermissionWriteConfig  = "write_config"
	PermissionListenPort   = "listen_port"
	PermissionCreateVolume = "create_volume"

	ActionCreate = "create"
	ActionBind   = "bind"

	EffectNew       = "new"
	EffectUntouched = "untouched"

	ConflictServiceExists   = "service_exists"
	ConflictPortInUse       = "port_in_use"
	ConflictPortDuplicate   = "port_duplicate"
	ConflictExistingVolume  = "existing_volume"
	ConflictVolumeDuplicate = "volume_duplicate"
	ConflictConfigMismatch  = "config_mismatch"

	RunOpen        = "open"
	RunInterrupted = "interrupted"
	RunFailed      = "failed"
	RunInvalidated = "invalidated"
	RunCancelled   = "cancelled"
	RunCompleted   = "completed"

	StepPreflight     = "preflight"
	StepRegisterAdmin = "register_admin"

	DiagnoseDNS   = "dns"
	DiagnoseTCP   = "tcp"
	DiagnoseProxy = "proxy"
	DiagnoseTLS   = "tls"

	ReasonAINotAllowed       = "ai_not_allowed"
	ReasonNotHuman           = "not_human"
	ReasonNotProbe           = "not_probe"
	ReasonSelfApproval       = "self_approval"
	ReasonNotPending         = "not_pending"
	ReasonNotProposer        = "not_proposer"
	ReasonWeakKey            = "weak_key"
	ReasonKeyLife            = "key_life"
	ReasonKeyMismatch        = "key_mismatch"
	ReasonKeyInactive        = "key_inactive"
	ReasonSignatureInvalid   = "signature_invalid"
	ReasonRevoked            = "revoked"
	ReasonProbeTaken         = "probe_taken"
	ReasonNotDeclared        = "not_declared"
	ReasonUnsupported        = "unsupported_platform"
	ReasonPlatformMismatch   = "platform_mismatch"
	ReasonIncomplete         = "incomplete"
	ReasonObservationStale   = "observation_stale"
	ReasonAlreadyInstalled   = "already_installed"
	ReasonPlanChanged        = "plan_changed"
	ReasonConflicts          = "conflicts"
	ReasonRunOpen            = "run_open"
	ReasonRunClosed          = "run_closed"
	ReasonOutOfOrder         = "out_of_order"
	ReasonDigestMismatch     = "digest_mismatch"
	ReasonProbeChanged       = "probe_changed"
	ReasonConfigChanged      = "config_changed"
	ReasonReleaseRevoked     = "release_revoked"
	ReasonHostChanged        = "host_changed"
	ReasonAcceptanceUsed     = "acceptance_used"
	ReasonNotAllowedState    = "not_allowed_state"
	ReasonInterrupted        = "interrupted"
	ReasonDependency         = "dependency_unavailable"
	ReasonUnknownCode        = "unknown_code"
	ReasonExpired            = "expired"
	ReasonStaleCode          = "stale_code"
	ReasonNotNominee         = "not_nominee"
	ReasonNotApproved        = "not_approved"
	ReasonAddressNotApproved = "address_not_approved"
)

type Refusal struct {
	Reason string
}

func (e Refusal) Error() string { return ErrRejected.Error() + ": " + e.Reason }

func (e Refusal) Unwrap() error { return ErrRejected }

type Config struct {
	Random io.Reader
}

type Limits struct {
	Keys            int
	KeyLife         time.Duration
	Releases        int
	Sites           int
	Destinations    int
	Reports         int
	Runs            int
	ObservationLife time.Duration
	BootstrapLife   time.Duration
	Proposals       int
}

type Actor struct {
	ID   string
	Kind string
}

type ReleaseKey struct {
	Version   string
	Public    []byte
	NotBefore time.Time
	NotAfter  time.Time
	Proof     []byte
}

type Platform struct {
	OS   string
	Arch string
}

type Artifact struct {
	Name   string
	Kind   string
	Digest string
}

type Port struct {
	Number   int
	Protocol string
}

type Service struct {
	Name     string
	Artifact string
	Ports    []Port
	Volumes  []string
}

type Volume struct {
	Name string
	Data bool
}

type Release struct {
	Version   string
	Key       string
	Platforms []Platform
	Artifacts []Artifact
	Services  []Service
	Volumes   []Volume
}

type Site struct {
	ID       string
	Tenant   string
	Platform Platform
	Probe    string
}

type PortOverride struct {
	Service string
	Port    Port
	Number  int
}

type VolumeOverride struct {
	Volume string
	Host   string
}

type SiteConfig struct {
	Site    string
	Version string
	Ports   []PortOverride
	Volumes []VolumeOverride
}

type Destination struct {
	Site   string
	ID     string
	Host   string
	Port   int
	Ranges []string
}

type Acceptance struct {
	Site    string
	Version string
	Hash    string
	Admin   string
}

type Change struct {
	Kind        string
	Key         ReleaseKey
	Release     Release
	Signature   []byte
	Site        Site
	Config      SiteConfig
	Destination Destination
	Acceptance  Acceptance
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
	Kind    string
	Version string
	Key     string
}

type Observation struct {
	Platform Platform
	Ports    []Port
	Volumes  []string
	Services []string
	Granted  []string
	Complete bool
}

type Item struct {
	Kind     string
	Target   string
	Action   string
	Conflict string
}

type DataEffect struct {
	Volume string
	Effect string
}

type PreviewView struct {
	Site        string
	Version     string
	Hash        string
	Services    []Item
	Ports       []Item
	Volumes     []Item
	Data        []DataEffect
	Permissions []string
	Conflicts   []Item
}

type RunView struct {
	ID       string
	Site     string
	Version  string
	Hash     string
	Revision uint64
	Probe    string
	Admin    string
	Steps    []string
	Done     int
	State    string
	Reason   string
	Note     string
}

type Evidence struct {
	Digest string
}

type Result struct {
	Kind    string
	Failure string
	Address string
	Via     string
}

type Guidance struct {
	Cause string
	Steps []string
}
