package sharedpool

import (
	"errors"
	"time"
)

var (
	ErrInvalid   = errors.New("request is incomplete or malformed")
	ErrNotFound  = errors.New("pool, version, run or chunk does not exist in this workspace, or nothing is published")
	ErrConflict  = errors.New("request conflicts with what is stored")
	ErrRejected  = errors.New("the stored state or a rule does not allow this step")
	ErrLimit     = errors.New("a configured limit or bound is exceeded")
	ErrSelfCheck = errors.New("a result failed its own check and nothing was written")
)

const (
	maxIDBytes        = 128
	maxVersion        = 999999
	maxAmountFraction = 4
	residualID        = ":residual"
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

type Bearer string

const (
	Organization Bearer = "organization"
	MSP          Bearer = "msp"
	Unattributed Bearer = "unattributed"
)

func (b Bearer) known() bool { return b == Organization || b == MSP || b == Unattributed }

func (b Bearer) residual() bool { return b == MSP || b == Unattributed }

type Spec struct {
	WorkspaceID    string
	Pool           string
	Version        int
	Currency       string
	PeriodStart    time.Time
	PeriodEnd      time.Time
	Amount         string
	Quantum        string
	Capacity       string
	ResidualBearer Bearer
}

type Purpose string

const (
	Showback   Purpose = "showback"
	Simulation Purpose = "simulation"
	Chargeback Purpose = "chargeback"
)

type RunSpec struct {
	ID      string
	Purpose Purpose
	Chunks  int
}

type Weight struct {
	Target string
	Bearer Bearer
	Weight string
}

type State string

const (
	Staging   State = "staging"
	Held      State = "held"
	Sealed    State = "sealed"
	Published State = "published"
	Withdrawn State = "withdrawn"
)

const (
	WeightNotKnown  = "weight_not_known"
	ZeroDenominator = "zero_denominator"
	OverCapacity    = "over_capacity"

	BelowCapacity       = "below_capacity"
	RecordedAlternative = "recorded_alternative"
)

type Alternative struct {
	Approver  string
	Reference string
	Bearer    Bearer
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

type RunView struct {
	ID             string
	Purpose        Purpose
	State          State
	Declared       int
	Staged         int
	Targets        int
	NotKnown       int
	Hold           string
	WeightSum      string
	Amount         string
	Denominator    string
	Alternative    *Alternative
	Approval       Approval
	WasPublished   bool
	WithdrawReason WithdrawReason
	Started        time.Time
	Changed        time.Time
	ByBearer       map[Bearer]string
	Residual       string
	ResidualReason string
}

type Share struct {
	Target   string
	Bearer   Bearer
	Weight   string
	Amount   string
	Residual bool
}

type SharePage struct {
	Items    []Share
	More     bool
	State    State
	Purpose  Purpose
	Sequence uint64
}

type WeightPage struct {
	Items []Weight
	More  bool
}

type VersionRow struct {
	Version     int
	Currency    string
	PeriodStart time.Time
	PeriodEnd   time.Time
}

type RunRow struct {
	ID      string
	Purpose Purpose
	State   State
}

type PublishedChargeback struct {
	Version  int
	Run      string
	Sequence uint64
}

type Limits struct {
	Pools        int
	Versions     int
	Runs         int
	Chunks       int
	ChunkWeights int
	Targets      int
	Page         int
	MaxPeriod    time.Duration
	Intermediate int
}

func (l Limits) valid() bool {
	return l.Pools > 0 && l.Versions > 0 && l.Runs > 0 && l.Chunks > 0 && l.ChunkWeights > 0 && l.Targets > 0 && l.Page > 0 && l.MaxPeriod > 0 && l.Intermediate > 0
}
