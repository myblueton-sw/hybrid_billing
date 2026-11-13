package tokenmeter

import (
	"errors"
	"time"
)

var (
	ErrInvalid  = errors.New("request is incomplete or malformed")
	ErrNotFound = errors.New("reception or source does not exist in this workspace, or nothing counts")
	ErrConflict = errors.New("request conflicts with what is stored")
	ErrRejected = errors.New("the stored state or a rule does not allow this step")
	ErrLimit    = errors.New("a configured limit is exceeded")
)

const (
	maxIDBytes  = 128
	maxRevision = 999999
)

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

func validCurrency(code string) bool {
	return len(code) == 3 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= 'Z' && code[2] >= 'A' && code[2] <= 'Z'
}

type Produced string

const (
	SnapshotExport Produced = "snapshot_export"
	ListInterface  Produced = "list_interface"
)

type Numeric string

const (
	Decimal Numeric = "decimal"
	Float   Numeric = "float"
)

const UnitToken = "token"

type Declaration string

const (
	SourceManifest Declaration = "source_manifest"
	Derived        Declaration = "derived"
)

type Kind string

const (
	Input      Kind = "input"
	Output     Kind = "output"
	CacheRead  Kind = "cache_read"
	CacheWrite Kind = "cache_write"
)

var kinds = []Kind{Input, Output, CacheRead, CacheWrite}

type Spec struct {
	WorkspaceID string
	ID          string
	Source      string
	Snapshot    string
	Schema      string
	Connection  string
	PeriodStart time.Time
	PeriodEnd   time.Time
	Revision    int
	Produced    Produced
	Numeric     Numeric
	Unit        string
	Events      int
	Tokens      map[Kind]string
	Costs       map[string]string
	Declaration Declaration
}

type Outcome string

const (
	Succeeded        Outcome = "succeeded"
	Failed           Outcome = "failed"
	Cancelled        Outcome = "cancelled"
	StreamIncomplete Outcome = "stream_incomplete"
)

func (o Outcome) known() bool {
	return o == Succeeded || o == Failed || o == Cancelled || o == StreamIncomplete
}

type Usage string

const (
	ProviderReported Usage = "provider_reported"
	UsageEstimated   Usage = "estimated"
	UsageNotKnown    Usage = "not_known"
)

func (u Usage) known() bool {
	return u == ProviderReported || u == UsageEstimated || u == UsageNotKnown
}

type Relation string

const (
	Included         Relation = "included"
	Separate         Relation = "separate"
	RelationNotKnown Relation = "not_known"
)

func (r Relation) known() bool { return r == Included || r == Separate || r == RelationNotKnown }

type Grade string

const (
	Invoiced      Grade = "invoiced"
	Billed        Grade = "billed"
	Computed      Grade = "computed"
	Estimated     Grade = "estimated"
	GradeNotKnown Grade = "not_known"
)

func (g Grade) known() bool {
	return g == Invoiced || g == Billed || g == Computed || g == Estimated || g == GradeNotKnown
}

type Origin string

const (
	Provider      Origin = "provider"
	SelfHostedGPU Origin = "self_hosted_gpu"
	TokenMeterFee Origin = "tokenmeter_fee"
)

func (o Origin) known() bool { return o == Provider || o == SelfHostedGPU || o == TokenMeterFee }

type Cost struct {
	Amount   string
	Currency string
	Grade    Grade
	Origin   Origin
	Document string
}

type Event struct {
	ID         string
	Request    string
	Attempt    string
	Time       time.Time
	Consumer   string
	Provider   string
	Model      string
	Version    string
	Endpoint   string
	Outcome    Outcome
	Input      string
	Output     string
	CacheRead  string
	CacheWrite string
	Usage      Usage
	Cache      Relation
	Cost       *Cost
}

const (
	OutsidePeriod          = "outside_period"
	CacheRelationBroken    = "cache_relation_broken"
	FloatPrecisionExceeded = "float_precision_exceeded"
	CostOutsideProfile     = "cost_outside_profile"

	ClassEstimated         = "estimated"
	ClassSelfHosted        = "self_hosted"
	ClassInvoiceReferenced = "invoice_referenced"
	ClassUnconfirmed       = "unconfirmed"
)

type EventView struct {
	Event    Event
	Findings []string
	Class    string
}

type State string

const (
	Open   State = "open"
	Closed State = "closed"
	Voided State = "voided"
)

type VoidReason string

const (
	WithdrawnAtSource   VoidReason = "withdrawn_at_source"
	RegeneratedAtSource VoidReason = "regenerated_at_source"
	WrongReport         VoidReason = "wrong_report"
	ByOperator          VoidReason = "operator"
)

func (r VoidReason) known() bool {
	return r == WithdrawnAtSource || r == RegeneratedAtSource || r == WrongReport || r == ByOperator
}

const (
	Eligible    = "eligible"
	NotEligible = "not_eligible"

	IsVoided            = "voided"
	NotClosed           = "not_closed"
	FromListInterface   = "list_interface"
	DeclarationDerived  = "declaration_derived"
	FloatTotalImprecise = "float_total_imprecise"
	PeriodConflict      = "period_conflict"
	EventShared         = "event_shared"
	EventsInError       = "events_in_error"
	Incomplete          = "incomplete"
	TotalsNotStated     = "totals_not_stated"
	TotalsDiffer        = "totals_differ"
	SumOutsideProfile   = "sum_outside_profile"
)

type Listed struct {
	Total int
	Items []string
}

type Verdict struct {
	Result    string
	Reasons   []string
	Conflicts Listed
	Shared    Listed
}

type DocumentSum struct {
	Events int
	Sums   map[string]string
}

type Figures struct {
	InError    int
	Outcomes   map[Outcome]int
	Tokens     map[Usage]map[Kind]string
	Unknown    map[Kind]int
	NetInput   string
	NetUnknown int
	Costs      map[string]map[string]string
	Documents  map[string]DocumentSum
}

type View struct {
	Spec       Spec
	State      State
	WasClosed  bool
	Events     int
	Figures    *Figures
	VoidReason VoidReason
	Opened     time.Time
	Changed    time.Time
}

type Row struct {
	ID          string
	Snapshot    string
	PeriodStart time.Time
	PeriodEnd   time.Time
	State       State
}

type Counted struct {
	Reception string
	Snapshot  string
	Sequence  uint64
}

type Limits struct {
	Receptions int
	Events     int
	Batch      int
	Currencies int
	Documents  int
	Page       int
	Listed     int
	MaxPeriod  time.Duration
}

func (l Limits) valid() bool {
	return l.Receptions > 0 && l.Events > 0 && l.Batch > 0 && l.Currencies > 0 && l.Documents > 0 && l.Page > 0 && l.Listed > 0 && l.MaxPeriod > 0
}
