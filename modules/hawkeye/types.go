package hawkeye

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
	maxCoverage = 10000
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

type CostStatus string

const (
	Final       CostStatus = "final"
	Provisional CostStatus = "provisional"
)

type Tier string

const (
	Measured    Tier = "measured"
	Estimated   Tier = "estimated"
	Unavailable Tier = "unavailable"
)

type Declaration string

const (
	SourceHeader Declaration = "source_header"
	Derived      Declaration = "derived"
)

type Spec struct {
	WorkspaceID      string
	ID               string
	Source           string
	Statement        string
	CostOrigin       string
	Schema           string
	Connection       string
	PeriodStart      time.Time
	PeriodEnd        time.Time
	Revision         int
	Supersedes       string
	CorrectionReason string
	Mock             bool
	Status           CostStatus
	Tier             Tier
	Coverage         int
	Lines            int
	Totals           map[string]string
	Declaration      Declaration
}

type Line struct {
	Team     string
	Cost     string
	Currency string
	Measured bool
	Energy   string
}

type State string

const (
	Open   State = "open"
	Closed State = "closed"
	Voided State = "voided"
)

type VoidReason string

const (
	WithdrawnAtSource VoidReason = "withdrawn_at_source"
	ReissuedAtSource  VoidReason = "reissued_at_source"
	WrongReport       VoidReason = "wrong_report"
	VoidedBy          VoidReason = "operator"
)

func (r VoidReason) known() bool {
	return r == WithdrawnAtSource || r == ReissuedAtSource || r == WrongReport || r == VoidedBy
}

const (
	Eligible    = "eligible"
	NotEligible = "not_eligible"

	IsVoided           = "voided"
	NotClosed          = "not_closed"
	IsMock             = "mock"
	IsProvisional      = "provisional"
	NotMeasured        = "not_measured"
	CoverageBelow      = "coverage_below_required"
	DeclarationDerived = "declaration_derived"
	IsSuperseded       = "superseded"
	PeriodConflict     = "period_conflict"
	UnmeasuredLines    = "unmeasured_lines"
	IncompleteList     = "incomplete_list"
	TotalsDiffer       = "totals_differ"
	SumOutsideProfile  = "sum_outside_profile"
)

type Listed struct {
	Total int
	Items []string
}

type Verdict struct {
	Result    string
	Reasons   []string
	Conflicts Listed
}

type View struct {
	Spec         Spec
	State        State
	WasClosed    bool
	Lines        int
	Sums         map[string]string
	SupersededBy string
	VoidReason   VoidReason
	Opened       time.Time
	Changed      time.Time
}

type Row struct {
	ID          string
	Statement   string
	PeriodStart time.Time
	PeriodEnd   time.Time
	State       State
	Mock        bool
	Superseded  bool
}

type Counted struct {
	Reception string
	Statement string
	Sequence  uint64
}

type Limits struct {
	Receptions int
	Lines      int
	Currencies int
	Page       int
	Listed     int
	MaxPeriod  time.Duration
}

func (l Limits) valid() bool {
	return l.Receptions > 0 && l.Lines > 0 && l.Currencies > 0 && l.Page > 0 && l.Listed > 0 && l.MaxPeriod > 0
}
