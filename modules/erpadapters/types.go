package erpadapters

import (
	"errors"
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
	maxIDBytes       = 128
	maxModules       = 32
	maxWorkspaces    = 100000
	maxProfiles      = 10000
	maxAdapters      = 10000
	maxDiagnoses     = 100000
	maxRowsPerDoc    = 1000000
	maxCalls         = 1000000
	maxPlanRows      = 100000000
	maxPlanDocuments = 100000

	KindHuman     = "human"
	KindAssistant = "assistant"
	KindSystem    = "system"

	LevelDeclared   = "declared"
	LevelSimulated  = "simulated"
	LevelSandbox    = "sandbox"
	LevelProduction = "production"

	CertUncertified = "uncertified"

	DeliveryBlocked = "blocked"
	DeliveryAllowed = "allowed"
	DeliveryManual  = "manual_only"

	ReasonAINotAllowed       = "ai_not_allowed"
	ReasonNotHuman           = "not_human"
	ReasonDirectDatabase     = "direct_database_forbidden"
	ReasonVendorMismatch     = "vendor_mismatch"
	ReasonUnknownProfile     = "unknown_profile"
	ReasonNotMapped          = "not_mapped"
	ReasonLevelDown          = "level_down"
	ReasonSelfApproval       = "self_approval"
	ReasonDeliveryUnsafe     = "delivery_unsafe"
	ReasonLimitUnproven      = "limit_unproven"
	ReasonInternalAttachment = "internal_attachment"
	ReasonRecipientUnproven  = "recipient_unproven"
	ReasonAttachmentUnproven = "attachment_unproven"
	ReasonRevoked            = "revoked"
	ReasonStaleDiagnosis     = "stale_diagnosis"
	LimitRows                = "max_rows"
	LimitCalls               = "calls_per_minute"
)

var (
	vendors     = map[string]bool{"odoo": true, "sap": true, "other": true}
	products    = map[string]string{"odoo": "odoo", "s4hana": "sap", "ecc": "sap", "business_one": "sap", "other": "other"}
	deployments = map[string]bool{"cloud": true, "on_premise": true, "private_cloud": true}
	commands    = []string{"connect", "validate_mapping", "prepare", "submit_draft", "read_draft", "post", "get_status", "correct", "get_balance"}
	lifecycle   = []string{"submit_draft", "read_draft", "post", "get_status", "correct", "get_balance"}
	levels      = map[string]int{LevelDeclared: 0, LevelSimulated: 1, LevelSandbox: 2, LevelProduction: 3}
	attachments = map[string]bool{"invoice_pdf": true, "invoice_xml": true, "statement": true, "internal_cost": true}
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
	Workspaces int
	Profiles   int
	Adapters   int
	Diagnoses  int
}

func (l Limits) valid() bool {
	return l.Workspaces >= 1 && l.Workspaces <= maxWorkspaces && l.Profiles >= 1 && l.Profiles <= maxProfiles &&
		l.Adapters >= 1 && l.Adapters <= maxAdapters && l.Diagnoses >= 1 && l.Diagnoses <= maxDiagnoses
}

type Profile struct {
	ID         string
	Vendor     string
	Product    string
	Version    string
	Deployment string
	Modules    []string
}

type ProfileView struct {
	Profile Profile
	Author  string
	At      time.Time
}

type Method struct {
	Command   string
	Name      string
	Transport string
}

type Adapter struct {
	ID             string
	Profile        string
	Methods        []Method
	MaxRows        int
	CallsPerMinute int
}

type Evidence struct {
	Level     string
	Reference string
}

type EvidenceView struct {
	Evidence  Evidence
	Actor     string
	At        time.Time
	Revoked   bool
	Reason    string
	RevokedBy string
	RevokedAt time.Time
}

type MethodView struct {
	Method   Method
	Evidence EvidenceView
}

type AdapterView struct {
	Adapter Adapter
	Methods []MethodView
	Rows    EvidenceView
	Calls   EvidenceView
	Author  string
	At      time.Time
}

type Enablement struct {
	Diagnosis int
	Actor     string
	At        time.Time
}

type TargetView struct {
	Target    string
	Diagnosis *Diagnosis
	Enabled   *Enablement
	Basis     Delivery
}

type Certification struct {
	Level    string
	Blocking []string
}

type Delivery struct {
	AutoAttach        bool
	AutoMail          bool
	Portal            bool
	Attachments       []string
	RecipientControl  string
	AttachmentControl string
}

type Diagnosis struct {
	Number   int
	Target   string
	Delivery Delivery
	Result   string
	Reasons  []string
	Actor    string
	At       time.Time
}

type Chunk struct {
	First int
	Last  int
}
