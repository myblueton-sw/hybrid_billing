package trust

import (
	"errors"
	"net/netip"
	"time"
)

var (
	ErrInvalid  = errors.New("a request or an input is incomplete or malformed")
	ErrNotFound = errors.New("what the request names does not exist in this workspace")
	ErrConflict = errors.New("another fact is already recorded under this identifier")
	ErrRejected = errors.New("a recorded state or rule refuses this request")
	ErrLimit    = errors.New("a configured limit or bound is exceeded")
)

const (
	maxIDBytes                  = 128
	maxLocation                 = 2048
	maxWorkspaces               = 100000
	maxConnections              = 10000
	maxRetired                  = 1000000
	maxEndpoints                = 100
	maxKeys                     = 16
	maxDestinations             = 10000
	maxEvents                   = 1000000
	maxItems                    = 10000
	maxQuarantine               = 1000000
	maxAllowlist                = 16
	maxRedirects                = 10
	maxBody                     = 64 << 20
	minSkew                     = time.Second
	maxSkew                     = time.Hour
	minKeyLife                  = time.Hour
	maxKeyLife                  = 3650 * 24 * time.Hour
	maxFingerprints             = 2
	signatureBytes              = 64
	pushTag                     = "trust_push_v1"
	defaultPort                 = 443
	maxHostBytes                = 253
	maxLabelBytes               = 63
	KindHuman                   = "human"
	KindAssistant               = "assistant"
	KindSystem                  = "system"
	StateProposed               = "proposed"
	StateActive                 = "active"
	StateRetired                = "retired"
	StateRevoked                = "revoked"
	ReasonAINotAllowed          = "ai_not_allowed"
	ReasonNotHuman              = "not_human"
	ReasonSelfApproval          = "self_approval"
	ReasonNotPending            = "not_pending"
	ReasonProposalChanged       = "proposal_changed"
	ReasonRevoked               = "revoked"
	ReasonNotActive             = "not_active"
	ReasonFingerprintInUse      = "fingerprint_in_use"
	ReasonFingerprintRetired    = "fingerprint_retired"
	ReasonUnknownCertificate    = "unknown_certificate"
	ReasonWorkspaceMismatch     = "workspace_mismatch"
	ReasonEndpointNotRegistered = "endpoint_not_registered"
	ReasonWeakKey               = "weak_key"
	ReasonKeyLife               = "key_life"
	ReasonConnectionChanged     = "connection_changed"
	ReasonUnknownKey            = "unknown_key"
	ReasonForged                = "forged"
	ReasonKeyExpired            = "key_expired"
	ReasonStale                 = "stale"
	ReasonAlreadyAccepted       = "already_accepted"
	ReasonDuplicate             = "duplicate"
	ReasonReplayCapacity        = "replay_capacity"
	ReasonPortMismatch          = "port_mismatch"
	ReasonAddressMismatch       = "address_mismatch"
	ReasonMetadataAddress       = "metadata_address"
	ReasonForbiddenAddress      = "forbidden_address"
	ReasonPrivateAddress        = "private_address"
	ReasonUnapprovedRedirect    = "unapproved_redirect"
	ReasonRedirectLimit         = "redirect_limit"
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

type Limits struct {
	Workspaces   int
	Connections  int
	Retired      int
	Endpoints    int
	Keys         int
	Destinations int
	Events       int
	Items        int
	Quarantine   int
	Allowlist    int
	Redirects    int
	Body         int
	Skew         time.Duration
	KeyLife      time.Duration
}

type Fingerprint [32]byte

type Connection struct {
	ID           string
	Fingerprints []Fingerprint
	Endpoints    []string
}

type ConnectionView struct {
	Active   Connection
	Revision int
	Proposal int
	State    string
	Pending  *Connection
	Proposer string
	Approver string
}

type Presented struct {
	Fingerprint Fingerprint
	Workspace   string
	Endpoint    string
}

type Admission struct {
	Workspace  string
	Connection string
	Revision   int
}

type Key struct {
	Version   string
	Public    []byte
	NotBefore time.Time
	NotAfter  time.Time
}

type KeyView struct {
	Key      Key
	State    string
	Proposer string
	Approver string
}

type Push struct {
	EventIDs   []string
	KeyVersion string
	Signed     time.Time
	Body       []byte
	Signature  []byte
}

type Entry struct {
	Sequence   int
	Connection string
	EventIDs   []string
	KeyVersion string
	Reason     string
	Signed     time.Time
	Received   time.Time
	BodyDigest [32]byte
}

type Destination struct {
	ID      string
	URL     string
	Private []netip.Prefix
}

type DestinationView struct {
	Destination Destination
	State       string
	Proposer    string
	Approver    string
}

type Redirect struct {
	Destination string
	URL         string
	Hop         int
}
