package connectors

import (
	"errors"
	"math"
	"time"
)

var (
	ErrInvalid  = errors.New("request is incomplete or malformed")
	ErrNotFound = errors.New("collection or subject does not exist in this workspace")
	ErrConflict = errors.New("request conflicts with what is stored")
	ErrRejected = errors.New("the state of the collection or capability does not allow this step")
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

func validChecksum(sum string) bool {
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

type Operation string

const (
	Discover           Operation = "discover"
	ValidateConnection Operation = "validate_connection"
	FetchCosts         Operation = "fetch_costs"
	FetchInvoices      Operation = "fetch_invoices"
	Checkpoint         Operation = "checkpoint"
)

func (o Operation) fetches() bool { return o == FetchCosts || o == FetchInvoices }

func (o Operation) known() bool {
	return o.fetches() || o == Discover || o == ValidateConnection || o == Checkpoint
}

type Kind string

const (
	ProviderOriginal Kind = "provider_original"
	ProForma         Kind = "pro_forma"
	DistributorSales Kind = "distributor_sales"
	NoKind           Kind = "none"
)

func (k Kind) data() bool { return k == ProviderOriginal || k == ProForma || k == DistributorSales }

func paired(operation Operation, kind Kind) bool {
	return operation.known() && ((operation.fetches() && kind.data()) || (!operation.fetches() && kind == NoKind))
}

type Condition struct {
	Contract string
	Region   string
	Source   string
}

func (c Condition) valid() bool { return validID(c.Contract) && validID(c.Region) && validID(c.Source) }

type Support string

const (
	Verified    Support = "verified"
	Limited     Support = "limited"
	Unsupported Support = "unsupported"
)

type Capability struct {
	Provider    string
	Version     string
	Operation   Operation
	Kind        Kind
	Condition   Condition
	State       Support
	Evidence    string
	Reason      string
	Alternative string
}

func (c Capability) valid() bool {
	if !validID(c.Provider) || !validID(c.Version) || !paired(c.Operation, c.Kind) || !c.Condition.valid() {
		return false
	}
	switch c.State {
	case Verified:
		return validID(c.Evidence) && c.Reason == "" && c.Alternative == ""
	case Limited:
		return validID(c.Evidence) && validID(c.Reason) && c.Alternative == ""
	case Unsupported:
		return c.Evidence == "" && validID(c.Reason) && (c.Alternative == "" || validID(c.Alternative))
	}
	return false
}

type Answer string

const (
	Allowed           Answer = "allowed"
	AnswerLimited     Answer = "limited"
	AnswerUnsupported Answer = "unsupported"
	Unknown           Answer = "unknown"
)

const (
	GapCapability = "capability_changed"
	GapManifest   = "no_manifest"
	GapMissing    = "part_missing"
	GapMismatched = "part_mismatched"
	GapChain      = "chain_unfinished"
	GapTotals     = "totals_differ"
	counterCap    = 1 << 20
)

type Permission struct {
	Answer      Answer
	Reason      string
	Alternative string
	Revision    uint64
}

type Subject struct {
	WorkspaceID  string
	ConnectionID string
	Operation    Operation
	ScopeID      string
	PeriodID     string
	Kind         Kind
}

func (s Subject) valid() bool {
	return validID(s.WorkspaceID) && validID(s.ConnectionID) && s.Operation.fetches() && validID(s.ScopeID) && validID(s.PeriodID) && s.Kind.data()
}

type Shape string

const (
	Files Shape = "files"
	Pages Shape = "pages"
)

type Spec struct {
	ID        string
	Subject   Subject
	Provider  string
	Version   string
	Condition Condition
	Revision  uint64
	Shape     Shape
	Cursor    string
}

func (s Spec) valid() bool {
	if !validID(s.ID) || !s.Subject.valid() || !validID(s.Provider) || !validID(s.Version) || !s.Condition.valid() || s.Revision == 0 {
		return false
	}
	return (s.Shape == Files && s.Cursor == "") || (s.Shape == Pages && validID(s.Cursor))
}

type State string

const (
	Collecting   State = "collecting"
	Interrupted  State = "interrupted"
	Completed    State = "completed"
	Superseded   State = "superseded"
	Inconsistent State = "inconsistent"
	Abandoned    State = "abandoned"
	Retired      State = "retired"
)

func (s State) running() bool { return s == Collecting || s == Interrupted }

type Origin string

const (
	SourceListed   Origin = "source_listed"
	AdapterDerived Origin = "adapter_derived"
)

type Part struct {
	ID       string
	Bytes    int64
	Rows     int64
	Checksum string
}

type LossReason string

const (
	PermissionWithdrawn LossReason = "permission_withdrawn"
	CredentialExpired   LossReason = "credential_expired"
	ConnectionDisabled  LossReason = "connection_disabled"
)

func (r LossReason) known() bool {
	return r == PermissionWithdrawn || r == CredentialExpired || r == ConnectionDisabled
}

type AbandonReason string

const (
	CapabilityChanged AbandonReason = "capability_changed"
	CursorExpired     AbandonReason = "cursor_expired"
	LimitReached      AbandonReason = "limit_reached"
	PartUnverifiable  AbandonReason = "part_unverifiable"
	ByOperator        AbandonReason = "operator"
)

func (r AbandonReason) known() bool {
	return r == CapabilityChanged || r == CursorExpired || r == LimitReached || r == PartUnverifiable || r == ByOperator
}

type RetireReason string

const (
	RetireConflict     RetireReason = "conflict"
	CapabilityWithdrew RetireReason = "capability_withdrawn"
	WrongSource        RetireReason = "wrong_source"
	RetireByOperator   RetireReason = "operator"
)

func (r RetireReason) known() bool {
	return r == RetireConflict || r == CapabilityWithdrew || r == WrongSource || r == RetireByOperator
}

type Gap struct {
	What string
	Part string
}

type Totals struct {
	Units int
	Bytes int64
	Rows  int64
}

type PartView struct {
	Part       Part
	Verified   bool
	Mismatches int
	Last       Part
}

type CollectionView struct {
	Spec          Spec
	State         State
	Capability    uint64
	Origin        Origin
	Declared      bool
	Parts         []PartView
	Pages         int
	Waiting       string
	Ended         bool
	Interruption  uint64
	LossReason    LossReason
	AbandonReason AbandonReason
	RetireReason  RetireReason
	Throttles     int
	LastWait      time.Duration
	Totals        Totals
	Digest        string
	Empty         bool
	Completion    uint64
	Contradiction string
	Started       time.Time
	Changed       time.Time
}

type Standing string

const (
	Duplicate   Standing = "duplicate"
	Conflicting Standing = "conflicting"
	Newer       Standing = "newer"
	Older       Standing = "older"
	Candidate   Standing = "candidate"
	IsRetired   Standing = "retired"
)

type Other struct {
	ID           string
	Standing     Standing
	Revision     uint64
	Digest       string
	Empty        bool
	Origin       Origin
	RetireReason RetireReason
}

type EffectiveView struct {
	Collection *CollectionView
	Pointer    uint64
	Moved      time.Time
	Known      uint64
	Stale      bool
	Present    Permission
	Others     []Other
}

type Limits struct {
	Capabilities int
	Collections  int
	Parts        int
	Pages        int
	MaxBytes     int64
	MaxRows      int64
	MaxWait      time.Duration
}

func (l Limits) valid() bool {
	if l.Capabilities <= 0 || l.Collections <= 0 || l.Parts <= 0 || l.Pages <= 0 || l.MaxBytes <= 0 || l.MaxRows <= 0 || l.MaxWait <= 0 {
		return false
	}
	units := int64(max(l.Parts, l.Pages))
	return l.MaxBytes <= math.MaxInt64/units && l.MaxRows <= math.MaxInt64/units
}
