package costpool

import (
	"errors"
	"time"
)

var (
	ErrInvalid   = errors.New("request is incomplete or malformed")
	ErrNotFound  = errors.New("pool or version does not exist in this workspace, or nothing counts")
	ErrConflict  = errors.New("request conflicts with what is stored")
	ErrRejected  = errors.New("the stored state or a rule does not allow this step")
	ErrLimit     = errors.New("a configured limit or bound is exceeded")
	ErrSelfCheck = errors.New("a result failed its own check and nothing was written")
)

const (
	maxIDBytes        = 128
	maxVersion        = 999999
	maxAmountFraction = 4
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

func validQuantum(text string) bool {
	return text == "1" || text == "0.1" || text == "0.01" || text == "0.001" || text == "0.0001"
}

type Spec struct {
	WorkspaceID string
	Pool        string
	Version     int
	Currency    string
	PeriodStart time.Time
	PeriodEnd   time.Time
	Owner       string
	Quantum     string
	Supersedes  int
}

type State string

const (
	Draft            State = "draft"
	ApprovalRecorded State = "approval_recorded"
	Withdrawn        State = "withdrawn"
)

type AssetKind string

const (
	Server       AssetKind = "server"
	GPU          AssetKind = "gpu"
	Storage      AssetKind = "storage"
	Network      AssetKind = "network"
	FacilityRoom AssetKind = "facility"
	OtherAsset   AssetKind = "other"
)

func (k AssetKind) known() bool {
	return k == Server || k == GPU || k == Storage || k == Network || k == FacilityRoom || k == OtherAsset
}

type Kind string

const (
	Depreciation Kind = "depreciation"
	Lease        Kind = "lease"
	PowerCost    Kind = "power"
	Facility     Kind = "facility"
	Licence      Kind = "licence"
	Operations   Kind = "operations"
)

func (k Kind) known() bool {
	return k == Depreciation || k == Lease || k == PowerCost || k == Facility || k == Licence || k == Operations
}

type Asset struct {
	ID          string
	Kind        AssetKind
	Method      Kind
	Acquisition string
}

type Grade string

const (
	Measured      Grade = "measured"
	Estimated     Grade = "estimated"
	GradeNotKnown Grade = "not_known"
)

type Power struct {
	Grade  Grade
	Source string
	PUE    string
	From   time.Time
	To     time.Time
}

type Component struct {
	ID         string
	Kind       Kind
	Amount     string
	Asset      string
	Evidence   string
	IncludedIn string
	Power      *Power
}

type Measure string

const (
	ReservedCapacity Measure = "reserved_capacity"
	FixedAgreement   Measure = "fixed_agreement"
)

type Bearer string

const (
	Organization Bearer = "organization"
	MSP          Bearer = "msp"
	Unattributed Bearer = "unattributed"
)

func (b Bearer) known() bool { return b == Organization || b == MSP || b == Unattributed }

type Target struct {
	ID     string
	Bearer Bearer
	Weight string
	Idle   bool
}

type Basis struct {
	ID       string
	Measure  Measure
	Capacity string
	Targets  []Target
}

type Approval struct {
	Approver  string
	Reference string
}

type WithdrawReason string

const (
	EnteredInError WithdrawReason = "entered_in_error"
	ByOperator     WithdrawReason = "operator"
)

const (
	Current    = "current"
	NotCurrent = "not_current"

	IsWithdrawn        = "withdrawn"
	NoApprovalRecorded = "no_approval_recorded"
	IsSuperseded       = "superseded"

	MultipleYes           = "yes"
	MultipleNo            = "no"
	MultipleNotDetermined = "not_determined"
)

type Verdict struct {
	Result  string
	Reasons []string
}

type Listed struct {
	Total int
	Items []int
}

type Blockers struct {
	Summed                 int
	WithoutAmount          int
	AssetsWithoutComponent int
	BasisSet               bool
	NamedSuperseded        bool
	Multiple               string
	Overlapping            Listed
}

type Figures struct {
	Total            string
	ByKind           map[Kind]string
	PoolLevel        string
	HasPoolLevel     bool
	AssetSums        int
	ByGrade          map[Grade]string
	IncludedKnown    int
	IncludedNotKnown int
	PricesStated     int
	PricesNotStated  int
	ByBearer         map[Bearer]string
	Idle             string
	HasIdle          bool
}

type View struct {
	Spec           Spec
	State          State
	Recorded       bool
	Superseded     bool
	SupersededBy   int
	Approval       Approval
	WithdrawReason WithdrawReason
	Opened         time.Time
	Changed        time.Time
	Assets         int
	Components     int
	Targets        int
	Basis          *Basis
	Blockers       *Blockers
	Figures        *Figures
}

type Share struct {
	Target string
	Bearer Bearer
	Weight string
	Idle   bool
	Amount string
}

type AssetSum struct {
	Asset string
	Sum   string
}

type Row struct {
	Version     int
	PeriodStart time.Time
	PeriodEnd   time.Time
	State       State
	Superseded  bool
}

type Counted struct {
	Version  int
	Sequence uint64
}

type Limits struct {
	Pools        int
	Versions     int
	Assets       int
	Components   int
	Targets      int
	Page         int
	Listed       int
	MaxPeriod    time.Duration
	Intermediate int
}

func (l Limits) valid() bool {
	return l.Pools > 0 && l.Versions > 0 && l.Assets > 0 && l.Components > 0 && l.Targets > 0 && l.Page > 0 && l.Listed > 0 && l.MaxPeriod > 0 && l.Intermediate > 0
}
