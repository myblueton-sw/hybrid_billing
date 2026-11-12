package relationships

import (
	"errors"
	"fmt"
	"math"
	"time"
)

var (
	ErrInvalid    = errors.New("request or record is incomplete or inconsistent")
	ErrNotFound   = errors.New("record does not exist in this workspace")
	ErrConflict   = errors.New("record conflicts with the stored state")
	ErrState      = errors.New("plan is not in a state that allows this step")
	ErrIncomplete = errors.New("plan chunks do not match the stated count")
	ErrRejected   = errors.New("plan was rejected by validation")
	ErrLimit      = errors.New("a configured limit is exceeded")
)

type Kind string

const (
	Hierarchy      Kind = "hierarchy"
	Payer          Kind = "payer"
	ContractPool   Kind = "contract_pool"
	AllocationPool Kind = "allocation_pool"
	ViewGroup      Kind = "view_group"
)

var kinds = []Kind{AllocationPool, ContractPool, Hierarchy, Payer, ViewGroup}

func known(kind Kind) bool {
	for _, candidate := range kinds {
		if candidate == kind {
			return true
		}
	}
	return false
}

func exclusive(kind Kind) bool { return kind != ViewGroup }

func disjointRoles(kind Kind) bool {
	return kind == ContractPool || kind == AllocationPool || kind == ViewGroup
}

const (
	maxIDBytes = 128
	MaxDepth   = 32
	openEnd    = math.MaxUint32
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

type Day struct{ n uint32 }

func NewDay(year, month, day int) (Day, error) {
	if year < 1 || year > 9999 || month < 1 || month > 12 || day < 1 || day > 31 {
		return Day{}, ErrInvalid
	}
	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if date.Year() != year || int(date.Month()) != month || date.Day() != day {
		return Day{}, ErrInvalid
	}
	return Day{n: uint32(year*10000 + month*100 + day)}, nil
}

func (d Day) valid() bool { return d.n != 0 && d.n != openEnd }

func (d Day) Before(other Day) bool { return d.n < other.n }

func (d Day) String() string { return fmt.Sprintf("%08d", d.n) }

type Interval struct {
	from uint32
	end  uint32
}

func Between(from, to Day) Interval { return Interval{from: from.n, end: to.n} }

func From(from Day) Interval { return Interval{from: from.n, end: openEnd} }

func (i Interval) valid() bool { return i.from != 0 && i.from != openEnd && i.from < i.end }

func (i Interval) bounded() bool { return i.end != openEnd }

func (i Interval) Start() Day { return Day{n: i.from} }

func (i Interval) End() (Day, bool) {
	if !i.bounded() {
		return Day{}, false
	}
	return Day{n: i.end}, true
}

func (i Interval) contains(day Day) bool { return i.from <= day.n && day.n < i.end }

func (i Interval) overlaps(other Interval) bool { return i.from < other.end && other.from < i.end }

func (i Interval) intersect(other Interval) (Interval, bool) {
	result := Interval{from: max(i.from, other.from), end: min(i.end, other.end)}
	return result, result.from < result.end
}

type Entry struct {
	Target   string
	Interval Interval
}

type Timeline struct {
	Kind    Kind
	Member  string
	Entries []Entry
}

type Limits struct {
	Depth              int
	EntriesPerTimeline int
	TimelinesPerChunk  int
	ChunksPerPlan      int
	TimelinesPerPlan   int
	EntriesPerPlan     int
	OpenPlans          int
	Violations         int
	PageSize           int
}

func (l Limits) valid() bool {
	for _, value := range []int{l.Depth, l.EntriesPerTimeline, l.TimelinesPerChunk, l.ChunksPerPlan, l.TimelinesPerPlan, l.EntriesPerPlan, l.OpenPlans, l.Violations, l.PageSize} {
		if value < 1 {
			return false
		}
	}
	return l.Depth <= MaxDepth
}

type Rule string

const (
	RuleRoles      Rule = "member_and_target"
	RulePayerChain Rule = "payer_chain"
	RuleCycle      Rule = "cycle"
	RuleDepth      Rule = "depth"
	RulePlanLimit  Rule = "plan_limit"
	RuleNoChange   Rule = "no_change"
)

type Violation struct {
	Kind   Kind
	Member string
	Rule   Rule
}

type PlanState string

const (
	PlanDraft     PlanState = "draft"
	PlanSealed    PlanState = "sealed"
	PlanActivated PlanState = "activated"
	PlanRejected  PlanState = "rejected"
	PlanDiscarded PlanState = "discarded"
)

type KindSummary struct {
	Changed       int
	EntriesBefore int
	EntriesAfter  int
	Earliest      Day
	HasEarliest   bool
}

type Preview struct {
	Digest     [32]byte
	Kinds      map[Kind]KindSummary
	Violations []Violation
	Truncated  bool
}

type PlanInfo struct {
	State      PlanState
	Base       uint64
	Chunks     []int
	Digest     [32]byte
	Kinds      map[Kind]KindSummary
	Revision   uint64
	Violations []Violation
	Truncated  bool
}

type MemberChange struct {
	Member  string
	Differs []Interval
	Before  []string
	After   []string
}

type Segment struct {
	Interval Interval
	Target   string
	Has      bool
}

type RevisionInfo struct {
	Number   uint64
	Recorded time.Time
	Plan     string
	Digest   [32]byte
}
