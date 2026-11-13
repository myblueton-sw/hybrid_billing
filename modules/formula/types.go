package formula

import (
	"errors"
	"time"
)

var (
	ErrInvalid  = errors.New("a request, a set or a configuration is incomplete or malformed")
	ErrNotFound = errors.New("what the request names does not exist in this workspace")
	ErrConflict = errors.New("another fact is already recorded under this identifier")
	ErrRejected = errors.New("a recorded state or rule refuses this request")
	ErrLimit    = errors.New("a configured limit or bound is exceeded")
)

const (
	Semantics      = "formula_v1"
	subjectLabel   = "formula_approval_subject_v1"
	maxIDBytes     = 128
	maxNumberBytes = 64
	maxRank        = 999
	maxRevision    = 999999
	maxCount       = 1000000000
)

const (
	KindNumber   = "number"
	KindMoney    = "money"
	KindQuantity = "quantity"
	KindPrice    = "price"
)

const (
	NodeConstant  = "constant"
	NodeVariable  = "variable"
	NodeReference = "reference"
	NodeAdd       = "add"
	NodeSub       = "sub"
	NodeMul       = "mul"
	NodeDiv       = "div"
	NodeMin       = "min"
	NodeMax       = "max"
	NodeGrossUp   = "gross_up"
)

const (
	CodeTooMany        = "too_many"
	CodeTooLarge       = "too_large"
	CodeTooDeep        = "too_deep"
	CodeMalformed      = "malformed"
	CodeUnknownName    = "unknown_name"
	CodeTypeMismatch   = "type_mismatch"
	CodeConstantZero   = "constant_division_by_zero"
	CodeMarginRange    = "margin_out_of_range"
	CodeCycle          = "cycle"
	CodeRankConflict   = "rank_conflict"
	ReasonNotKnown     = "variable_not_known"
	ReasonDivideByZero = "division_by_zero"
	ReasonMarginRange  = "margin_out_of_range"
)

const (
	StateDraft            = "draft"
	StateCheckFailed      = "check_failed"
	StateChecked          = "checked"
	StateImpactRecorded   = "impact_recorded"
	StateApprovalRecorded = "approval_recorded"
	StateActive           = "active"
	StateSuperseded       = "superseded"
	StateWithdrawn        = "withdrawn"
	CoverageFull          = "full"
	CoverageSample        = "sample"
)

var codeOrder = map[string]int{
	CodeTooMany: 1, CodeTooLarge: 2, CodeTooDeep: 3, CodeMalformed: 4, CodeUnknownName: 5,
	CodeTypeMismatch: 6, CodeConstantZero: 7, CodeMarginRange: 8, CodeCycle: 9, CodeRankConflict: 10,
}

var operations = map[string]bool{NodeAdd: true, NodeSub: true, NodeMul: true, NodeDiv: true, NodeMin: true, NodeMax: true, NodeGrossUp: true}

var withdrawals = map[string]bool{"mistake": true, "replaced": true, "refused": true, "stopped": true}

type Type struct {
	Kind     string
	Currency string
	Unit     string
}

type Node struct {
	Kind     string
	Number   string
	Type     Type
	Name     string
	Formula  string
	Operands []Node
}

type Variable struct {
	Name string
	Type Type
}

type Formula struct {
	ID      string
	Target  string
	Rank    int
	Output  Type
	Quantum string
	Mode    string
	Root    Node
}

type Set struct {
	Variables []Variable
	Formulas  []Formula
}

type Limits struct {
	Variables int
	Formulas  int
	Nodes     int
	Depth     int
	Digits    int
}

func (l Limits) valid() bool {
	return l.Variables > 0 && l.Variables <= 1000 && l.Formulas > 0 && l.Formulas <= 1000 &&
		l.Nodes > 0 && l.Nodes <= 10000 && l.Depth > 0 && l.Depth <= 100 && l.Digits > 0 && l.Digits <= 10000
}

type RegistryLimits struct {
	Limits
	Sets      int
	Revisions int
	History   int
	Page      int
}

func (l RegistryLimits) valid() bool {
	return l.Limits.valid() && l.Sets > 0 && l.Sets <= 100000 && l.Revisions > 0 && l.Revisions <= maxRevision &&
		l.History > 0 && l.History <= 1000 && l.Page > 0 && l.Page <= 1000
}

type Finding struct {
	Code    string
	Formula string
}

type Input struct {
	Name  string
	Value string
}

type Outcome struct {
	Formula string
	Type    Type
	Value   string
	Exact   string
	Reason  string
}

type Result struct {
	Semantics       string
	ProfileID       string
	ProfileRevision string
	Target          string
	Outcome         Outcome
	Reached         []Outcome
	Set             string
	Revision        int
	Digest          string
}

type Impact struct {
	Manifest       string
	ManifestDigest string
	Epoch          int
	Coverage       string
	Evaluated      int
	Changed        int
	WithoutValue   int
	Failed         int
}

type Approval struct {
	Approver  string
	Reference string
	Subject   string
	At        time.Time
}

type ImpactRecord struct {
	Impact   Impact
	Subject  string
	At       time.Time
	Approval *Approval
}

type RevisionView struct {
	Set          string
	Number       int
	State        string
	Author       string
	Digest       string
	Findings     []Finding
	Impact       *ImpactRecord
	Current      bool
	History      []ImpactRecord
	Withdrawal   string
	DraftedAt    time.Time
	CheckedAt    time.Time
	ActivatedAt  time.Time
	SupersededAt time.Time
	WithdrawnAt  time.Time
}

type ActiveView struct {
	Revision int
	Digest   string
	Epoch    int
	Sequence int
}

type Listed struct {
	Number int
	State  string
}

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

func validDigest(text string) bool {
	if len(text) != 64 {
		return false
	}
	for index := 0; index < len(text); index++ {
		c := text[index]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func validType(t Type) bool {
	switch t.Kind {
	case KindNumber:
		return t.Currency == "" && t.Unit == ""
	case KindMoney:
		return validCurrency(t.Currency) && t.Unit == ""
	case KindQuantity:
		return t.Currency == "" && validID(t.Unit)
	case KindPrice:
		return validCurrency(t.Currency) && validID(t.Unit)
	}
	return false
}
