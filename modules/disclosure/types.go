package disclosure

import (
	"errors"
	"io"
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
	maxIDBytes      = 128
	maxValueBytes   = 4096
	maxWorkspaces   = 100000
	maxFields       = 10000
	maxPolicies     = 10000
	maxRecipients   = 100000
	maxProjections  = 1000000
	maxPackages     = 1000000
	maxLinks        = 1000000
	maxRows         = 100000
	maxCombinations = 10000
	minCombination  = 2
	maxCombination  = 8
	minLifetime     = time.Hour
	maxLifetime     = 30 * 24 * time.Hour
	maxPage         = 500
	projectionLabel = "disclosure_projection_v1"
	inputLabel      = "disclosure_input_v1"
	CustomerField   = "customer"

	KindHuman     = "human"
	KindAssistant = "assistant"
	KindSystem    = "system"

	ClassPublic   = "public"
	ClassCustomer = "customer"
	ClassInternal = "internal"
	ClassSecret   = "secret"

	RoleCustomer = "customer"
	RolePartner  = "partner"
	RoleAuditor  = "auditor"

	ReasonAINotAllowed       = "ai_not_allowed"
	ReasonNotHuman           = "not_human"
	ReasonSelfApproval       = "self_approval"
	ReasonStaleProposal      = "stale_proposal"
	ReasonUnclassifiedField  = "unclassified_field"
	ReasonClassNotAllowed    = "class_not_allowed"
	ReasonInferenceConflict  = "inference_conflict"
	ReasonLocatorClass       = "locator_class"
	ReasonPolicySuspended    = "policy_suspended"
	ReasonPolicySuperseded   = "policy_superseded"
	ReasonNoActivePolicy     = "no_active_policy"
	ReasonRecipientRevoked   = "recipient_revoked"
	ReasonRecipientChanged   = "recipient_changed"
	ReasonRecipientMismatch  = "recipient_mismatch"
	ReasonReferenceCollision = "reference_collision"
	ReasonLinkExpired        = "link_expired"
	ReasonLinkRevoked        = "link_revoked"
	ReasonKeyUnavailable     = "key_unavailable"
	ReasonRecipientActor     = "recipient_actor"
	ReasonPolicyNotGranted   = "policy_not_granted"
	ReasonNoFields           = "no_fields"
)

var (
	classes = map[string]bool{ClassPublic: true, ClassCustomer: true, ClassInternal: true, ClassSecret: true}
	roles   = map[string]bool{RoleCustomer: true, RolePartner: true, RoleAuditor: true}
	allowed = map[string]map[string]bool{
		RoleCustomer: {ClassPublic: true, ClassCustomer: true},
		RolePartner:  {ClassPublic: true, ClassCustomer: true},
		RoleAuditor:  {ClassPublic: true, ClassCustomer: true, ClassInternal: true},
	}
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

type Config struct {
	Keys   func(workspace string) ([]byte, error)
	Random io.Reader
}

type Limits struct {
	Workspaces   int
	Fields       int
	Policies     int
	Recipients   int
	Projections  int
	Packages     int
	Links        int
	Rows         int
	Combinations int
}

type Field struct {
	Name    string
	Class   string
	Locator bool
}

type FieldView struct {
	Field  Field
	Author string
	At     time.Time
}

type Combination struct {
	ID     string
	Fields []string
}

type CombinationView struct {
	Combination Combination
	Proposer    string
	Confirmer   string
	At          time.Time
}

type Policy struct {
	ID    string
	Roles map[string][]string
}

type PolicyView struct {
	Policy    Policy
	Revision  int
	State     string
	Proposer  string
	Confirmer string
	At        time.Time
}

type Recipient struct {
	ID        string
	Role      string
	Customers []string
	Policy    string
}

type GrantView struct {
	Recipient Recipient
	Revision  int
	State     string
	Proposer  string
	Confirmer string
	RevokedBy string
	RevokedAt time.Time
	At        time.Time
}

type Row map[string]string

type Request struct {
	Policy    string
	Recipient string
	Rows      []Row
}

type Projection struct {
	Digest string
	Fields []string
	Rows   [][]string
}

type Audit struct {
	Projection    Projection
	Workspace     string
	Recipient     string
	Grant         int
	Policy        string
	Revision      int
	Input         string
	DroppedRows   int
	DroppedFields []string
	Actor         string
	At            time.Time
}

type PackageView struct {
	ID         string
	Projection string
	Input      string
	Policy     string
	Revision   int
	Recipient  string
	Grant      int
	File       string
	FileDigest string
	Bytes      []byte
	Actor      string
	At         time.Time
}

type Delivery struct {
	ID         string
	File       string
	FileDigest string
	Bytes      []byte
}

type LinkView struct {
	Digest    string
	Package   string
	Expires   time.Time
	Issuer    string
	RevokedBy string
}
