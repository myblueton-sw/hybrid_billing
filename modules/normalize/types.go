package normalize

import (
	"errors"
	"time"
)

var (
	ErrInvalid  = errors.New("request is incomplete or malformed")
	ErrNotFound = errors.New("target, mapping, revision or run does not exist here")
	ErrConflict = errors.New("request conflicts with what is stored")
	ErrRejected = errors.New("the stored state or declaration does not allow this step")
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

func validCurrency(code string) bool {
	return len(code) == 3 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= 'Z' && code[2] >= 'A' && code[2] <= 'Z'
}

type Kind string

const (
	Cost     Kind = "cost"
	Quantity Kind = "quantity"
	Text     Kind = "text"
	Time     Kind = "time"
	Currency Kind = "currency"
)

func (k Kind) known() bool {
	return k == Cost || k == Quantity || k == Text || k == Time || k == Currency
}

func (k Kind) numeric() bool { return k == Cost || k == Quantity }

type CostKind string

const (
	Billed     CostKind = "billed"
	Effective  CostKind = "effective"
	List       CostKind = "list"
	Contracted CostKind = "contracted"
	Sale       CostKind = "sale"
)

func (c CostKind) target() bool {
	return c == Billed || c == Effective || c == List || c == Contracted
}

type Field struct {
	Name    string
	Kind    Kind
	Cost    CostKind
	Format  bool
	Billing bool
}

type Target struct {
	ID           string
	Fields       []Field
	CostCurrency string
	PeriodStart  string
	PeriodEnd    string
}

type Transform string

const (
	Copy   Transform = "copy"
	Negate Transform = "negate"
	Scale  Transform = "scale"
)

type MissingReason string

const (
	NotProvidedBySource MissingReason = "not_provided_by_source"
	NotSupported        MissingReason = "not_supported"
	NotApplicable       MissingReason = "not_applicable"
)

func (m MissingReason) known() bool {
	return m == NotProvidedBySource || m == NotSupported || m == NotApplicable
}

type Entry struct {
	Field      string
	Source     string
	SourceKind Kind
	SourceCost CostKind
	Transform  Transform
	Power      int
	Missing    MissingReason
}

type MappingKey struct {
	Provider      string
	SourceVersion string
	Target        string
}

func (k MappingKey) valid() bool {
	return validID(k.Provider) && validID(k.SourceVersion) && validID(k.Target)
}

type RevisionView struct {
	Key             MappingKey
	Number          uint64
	Entries         []Entry
	Complete        bool
	Partial         []string
	NotApplicable   []string
	BillingPossible bool
	BillingGaps     []string
	NoRequiredCost  bool
	At              time.Time
}

type KeyKind string

const (
	EventKeys   KeyKind = "event"
	LocatorKeys KeyKind = "locator"
)

type RowKey struct {
	Kind     KeyKind
	Document string
	Key      string
}

func (k RowKey) valid() bool {
	switch k.Kind {
	case EventKeys:
		return k.Document == "" && validID(k.Key)
	case LocatorKeys:
		return validID(k.Document) && validID(k.Key)
	}
	return false
}

type Cell struct {
	Null bool
	Text string
}

type Row struct {
	Key   RowKey
	Cells map[string]Cell
}

type Input struct {
	Scope    string
	Period   string
	Manifest uint64
}

type Window struct {
	From time.Time
	To   time.Time
}

type RunSpec struct {
	WorkspaceID string
	ID          string
	Mapping     MappingKey
	Revision    uint64
	Input       Input
	Chunks      int
	Rows        int
	Window      *Window
	Expected    map[string]map[string]string
}

type State string

const (
	Open         State = "open"
	Closed       State = "closed"
	Discarded    State = "discarded"
	Inconsistent State = "inconsistent"
	Abandoned    State = "abandoned"
)

type AbandonReason string

const (
	TransferFailed AbandonReason = "transfer_failed"
	Superseded     AbandonReason = "superseded"
	AbandonedBy    AbandonReason = "operator"
)

func (r AbandonReason) known() bool {
	return r == TransferFailed || r == Superseded || r == AbandonedBy
}

type DiscardReason string

const (
	DiscardSuperseded DiscardReason = "superseded"
	WrongInput        DiscardReason = "wrong_input"
	DiscardedBy       DiscardReason = "operator"
)

func (r DiscardReason) known() bool {
	return r == DiscardSuperseded || r == WrongInput || r == DiscardedBy
}

const (
	NotDecimal          = "not_decimal"
	OutsideProfile      = "outside_profile"
	NotTime             = "not_time"
	NotCurrency         = "not_currency"
	EmptyText           = "empty"
	RequiredNull        = "required_null"
	RequiredNotProvided = "required_not_provided"
	CurrencyMissing     = "currency_missing"
	PeriodMissing       = "period_missing"
	PeriodInverted      = "period_inverted"
	OutsideWindow       = "outside_window"
)

const (
	Conformant    = "conformant"
	Partial       = "partial"
	Nonconformant = "nonconformant"
	Ready         = "ready"
	NotReady      = "not_ready"
)

const (
	MappingPartial    = "mapping_partial"
	RowsInError       = "rows_in_error"
	RowsQuarantined   = "rows_quarantined"
	MappingGap        = "mapping_gap"
	RowsLacking       = "rows_lacking"
	TotalsNotExpected = "totals_not_expected"
	TotalsDiffer      = "totals_differ"
	WindowNotDeclared = "window_not_declared"
	SumOutsideProfile = "sum_outside_profile"
	ChunksMissing     = "chunks_missing"
	RowCountDiffers   = "rows_differ"
)

type Verdict struct {
	Result  string
	Reasons []string
}

type CellState string

const (
	Value       CellState = "value"
	Null        CellState = "null"
	NotProvided CellState = "not_provided"
	Unmapped    CellState = "unmapped"
)

type OutCell struct {
	State CellState
	Text  string
}

type OutRow struct {
	Chunk    int
	Position int
	Key      RowKey
	Cells    map[string]OutCell
	Lacking  []string
}

type RowsPage struct {
	Format  Verdict
	Billing Verdict
	Rows    []OutRow
}

type Finding struct {
	Chunk    int
	Position int
	Key      RowKey
	Field    string
	Code     string
}

type QuarantinedRow struct {
	Chunk    int
	Position int
	Key      RowKey
}

type Listed struct {
	Total int
	Items []string
}

type RunView struct {
	Spec            RunSpec
	State           State
	Checked         int
	Normal          int
	Errors          int
	Quarantined     int
	Sums            map[string]map[string]string
	Format          *Verdict
	Billing         *Verdict
	FindingsTotal   int
	Unused          Listed
	ProfileID       string
	ProfileRevision string
	AbandonReason   AbandonReason
	DiscardReason   DiscardReason
	Contradiction   int
	Opened          time.Time
	Changed         time.Time
}

type Limits struct {
	Targets   int
	Fields    int
	Mappings  int
	Revisions int
	Runs      int
	Chunks    int
	ChunkRows int
	Rows      int
	Cells     int
	Text      int
	Findings  int
	Unused    int
	Page      int
}

func (l Limits) valid() bool {
	return l.Targets > 0 && l.Fields > 0 && l.Mappings > 0 && l.Revisions > 0 && l.Runs > 0 && l.Chunks > 0 && l.ChunkRows > 0 && l.Rows > 0 && l.Cells > 0 && l.Text > 0 && l.Findings > 0 && l.Unused > 0 && l.Page > 0
}
