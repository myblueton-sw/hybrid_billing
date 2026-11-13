package evidence

import (
	"errors"
	"time"
)

var (
	ErrInvalid  = errors.New("request is incomplete or malformed")
	ErrNotFound = errors.New("artifact, instance, bundle, item, hold or gap does not exist in this workspace")
	ErrConflict = errors.New("request conflicts with what is stored")
	ErrRejected = errors.New("the stored state does not allow this step")
	ErrLimit    = errors.New("a configured limit is exceeded")
)

const maxIDBytes = 128

func validID(id string) bool {
	if id == "" || len(id) > maxIDBytes {
		return false
	}
	for index := 0; index < len(id); index++ {
		c := id[index]
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == ':' || c == '-') {
			return false
		}
	}
	return true
}

func validDigest(sum string) bool {
	if len(sum) < 8 || len(sum) > 128 {
		return false
	}
	for index := 0; index < len(sum); index++ {
		c := sum[index]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

type Kind string

const (
	SourceOriginal Kind = "source_original"
	InputManifest  Kind = "input_manifest"
	Mapping        Kind = "mapping"
	Attribution    Kind = "attribution"
	Policy         Kind = "policy"
	FX             Kind = "fx"
	Engine         Kind = "engine"
	Module         Kind = "module"
	Result         Kind = "result"
	Document       Kind = "document"
)

var kinds = []Kind{SourceOriginal, InputManifest, Mapping, Attribution, Policy, FX, Engine, Module, Result, Document}

var required = []Kind{SourceOriginal, InputManifest, Mapping, Attribution, Policy, Engine, Result}

func (k Kind) rank() int {
	for position, known := range kinds {
		if k == known {
			return position
		}
	}
	return -1
}

type CustodyKind string

const (
	Product CustodyKind = "product"
	Remote  CustodyKind = "remote"
)

type Custody struct {
	Kind          CustodyKind
	Location      string
	Receipt       string
	VerifiedUntil time.Time
	Evidence      string
}

func (c Custody) valid() bool {
	if !validID(c.Location) {
		return false
	}
	switch c.Kind {
	case Product:
		return validID(c.Receipt) && c.VerifiedUntil.IsZero() && c.Evidence == ""
	case Remote:
		return c.Receipt == "" && c.VerifiedUntil.IsZero() == (c.Evidence == "") && (c.Evidence == "" || validID(c.Evidence))
	}
	return false
}

type ArtifactSpec struct {
	WorkspaceID string
	ID          string
	Kind        Kind
	Version     string
	Digest      string
	Size        int64
	Custody     Custody
}

type ArtifactState string

const (
	Registered ArtifactState = "registered"
	Deleted    ArtifactState = "deleted"
)

type LossReason string

const (
	Expired               LossReason = "expired"
	LostDeleted           LossReason = "deleted"
	AccessRevoked         LossReason = "access_revoked"
	Corrupted             LossReason = "corrupted"
	VerificationWithdrawn LossReason = "verification_withdrawn"
)

func (r LossReason) known() bool {
	return r == Expired || r == LostDeleted || r == AccessRevoked || r == Corrupted || r == VerificationWithdrawn
}

type DeleteReason string

const (
	DeleteRetentionEnded DeleteReason = "retention_ended"
	WrongRegistration    DeleteReason = "wrong_registration"
	DeletedBy            DeleteReason = "operator"
)

func (r DeleteReason) known() bool {
	return r == DeleteRetentionEnded || r == WrongRegistration || r == DeletedBy
}

type PurgeReason string

const (
	PurgeRetentionEnded PurgeReason = "retention_ended"
	PurgedBy            PurgeReason = "operator"
)

func (r PurgeReason) known() bool { return r == PurgeRetentionEnded || r == PurgedBy }

type ReleaseReason string

const (
	MatterClosed  ReleaseReason = "matter_closed"
	PlacedInError ReleaseReason = "placed_in_error"
	ReleasedBy    ReleaseReason = "operator"
)

func (r ReleaseReason) known() bool {
	return r == MatterClosed || r == PlacedInError || r == ReleasedBy
}

type GapReason string

const (
	CaptureFailed       GapReason = "capture_failed"
	EmergencyRevocation GapReason = "emergency_revocation"
	ReceivedAfterEnd    GapReason = "received_after_end"
)

func (r GapReason) known() bool {
	return r == CaptureFailed || r == EmergencyRevocation || r == ReceivedAfterEnd
}

type InstanceView struct {
	Custody Custody
	Lost    bool
	Reason  LossReason
}

type ArtifactView struct {
	Spec         ArtifactSpec
	State        ArtifactState
	DeleteReason DeleteReason
	Instances    []InstanceView
	Restricted   bool
	Bundles      int
	Registered   time.Time
	Changed      time.Time
}

type Item struct {
	Role     Kind
	Artifact string
}

type BundleSpec struct {
	WorkspaceID string
	ID          string
	Customer    string
	Period      string
	RetainUntil time.Time
	FX          bool
	Modules     bool
	Items       []Item
}

type BundleState string

const (
	Confirmed BundleState = "confirmed"
	Purged    BundleState = "purged"
)

const (
	UnknownArtifact   = "unknown_artifact"
	DeletedArtifact   = "deleted_artifact"
	WrongKind         = "wrong_kind"
	NamedTwice        = "named_twice"
	RoleMissing       = "role_missing"
	FXMismatch        = "fx_mismatch"
	ModuleMismatch    = "module_mismatch"
	RetentionNotAhead = "retention_not_ahead"
	NotHeld           = "not_held"
)

type Reason struct {
	Code     string
	Role     Kind
	Artifact string
}

type Listed struct {
	Total int
	Items []string
}

type Confirmation struct {
	State   BundleState
	Reasons []Reason
	Gaps    Listed
}

type Extension struct {
	Until   time.Time
	NotHeld int
	AtRisk  int
}

type HoldSpec struct {
	WorkspaceID string
	ID          string
	Reason      string
	Bundles     []string
}

type HoldState string

const (
	Active   HoldState = "active"
	Released HoldState = "released"
)

type Holding struct {
	State     HoldState
	Positions []int
	Exposed   Listed
}

type GapSpec struct {
	WorkspaceID string
	ID          string
	Customer    string
	Period      string
	Role        Kind
	Reason      GapReason
	Owner       string
	Path        string
}

type GapState string

const (
	GapOpen   GapState = "open"
	GapClosed GapState = "closed"
)

type GapView struct {
	ID       string
	State    GapState
	Artifact string
}

const (
	Held            = "held"
	Broken          = "broken"
	CauseLost       = "lost"
	CauseUnverified = "verification_ended"
	Operational     = "operational"
	IsolatedOnly    = "isolated_only"
)

type BrokenItem struct {
	Role     Kind
	Artifact string
	Cause    string
	Reason   LossReason
}

type Inspection struct {
	Verdict     string
	Broken      []BrokenItem
	BrokenTotal int
	AtRisk      []Item
	AtRiskTotal int
	Rerun       string
	Protected   bool
	Bound       []GapView
	BoundTotal  int
	Open        Listed
}

type BundleView struct {
	Spec        BundleSpec
	State       BundleState
	PurgeReason PurgeReason
	Retained    time.Time
	Counts      map[Kind]int
	Digest      string
	Gaps        []string
	Holds       int
	Confirmed   time.Time
	Changed     time.Time
}

type Download struct {
	ID        string
	Version   string
	Digest    string
	Size      int64
	Instances []Custody
}

type Limits struct {
	Artifacts   int
	Bundles     int
	Holds       int
	Gaps        int
	Instances   int
	Items       int
	HoldBundles int
	Restricted  int
	Page        int
	Listed      int
}

func (l Limits) valid() bool {
	return l.Artifacts > 0 && l.Bundles > 0 && l.Holds > 0 && l.Gaps > 0 && l.Instances > 0 && l.Items > 0 && l.HoldBundles > 0 && l.Restricted > 0 && l.Page > 0 && l.Listed > 0
}
