package vmhistory

import (
	"errors"
	"time"
)

var (
	ErrInvalid  = errors.New("request is incomplete or malformed")
	ErrNotFound = errors.New("source, asset, span, counter or reading does not exist in this workspace")
	ErrConflict = errors.New("request conflicts with what is stored")
	ErrRejected = errors.New("the stored state or a rule does not allow this step")
	ErrLimit    = errors.New("a configured limit is exceeded")
)

const (
	maxIDBytes    = 128
	maxNumber     = 999999
	maxCPUs       = 1000000
	maxCPUDigits  = 7
	longestPeriod = 366 * 24 * time.Hour
	maxSizeDigits = 18
	maxWholePart  = 15
	maxFraction   = 6
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

type State string

const (
	Live   State = "live"
	Voided State = "voided"
)

type AssetKind string

const (
	VM   AssetKind = "vm"
	Host AssetKind = "host"
)

type Asset struct {
	ID       string
	Kind     AssetKind
	Created  time.Time
	Deleted  time.Time
	Revision int
}

type Size struct {
	Value string
	Unit  string
}

type Span struct {
	ID           string
	Asset        string
	From         time.Time
	To           time.Time
	CPUs         string
	Memory       Size
	Disk         Size
	Organization string
}

type Counter struct {
	ID     string
	Asset  string
	Unit   string
	MaxGap int64
}

type Reading struct {
	At         time.Time
	Value      string
	Generation int
	Revision   int
}

type VoidReason string

const (
	WithdrawnAtSource VoidReason = "withdrawn_at_source"
	WrongReport       VoidReason = "wrong_report"
	ByOperator        VoidReason = "operator"
)

func (r VoidReason) known() bool {
	return r == WithdrawnAtSource || r == WrongReport || r == ByOperator
}

const (
	Complete      = "complete"
	Incomplete    = "incomplete"
	NotApplicable = "not_applicable"

	NotCovered      = "not_covered"
	CPUNotStated    = "cpu_not_stated"
	MemoryNotStated = "memory_not_stated"
	DiskNotStated   = "disk_not_stated"

	NoCounter       = "no_counter"
	CountersOverlap = "counters_overlap"
	CounterReset    = "counter_reset"
	CounterDecrease = "counter_decrease"
	CounterGap      = "counter_gap"
	BoundaryNotRead = "boundary_not_read"
	NotRead         = "not_read"
)

type Dimension struct {
	Integral         string
	NotStatedSeconds int64
}

type Entry struct {
	Organization string
	Seconds      int64
	CPU          Dimension
	Memory       Dimension
	Disk         Dimension
}

type Configuration struct {
	Result    string
	Reasons   []string
	Covered   int64
	Uncovered int64
	Entries   []Entry
	None      *Entry
}

type CounterEnergy struct {
	Counter string
	Energy  string
	Seconds int64
}

type Energy struct {
	Result    string
	Reasons   []string
	Covered   int64
	Uncovered int64
	Counters  []CounterEnergy
	Total     string
}

type Result struct {
	Asset         string
	WindowStart   time.Time
	WindowEnd     time.Time
	Window        int64
	Configuration Configuration
	Energy        Energy
	Sequence      uint64
	Change        uint64
}

type AssetPage struct {
	Items    []Asset
	More     bool
	Sequence uint64
}

type SpanPage struct {
	Items    []Span
	More     bool
	Sequence uint64
	Change   uint64
}

type CounterView struct {
	Counter Counter
	State   State
}

type CounterPage struct {
	Items    []CounterView
	More     bool
	Sequence uint64
	Change   uint64
}

type ReadingPage struct {
	Items    []Reading
	More     bool
	Sequence uint64
	Change   uint64
}

type Limits struct {
	Sources   int
	Assets    int
	Spans     int
	Counters  int
	Readings  int
	Batch     int
	Page      int
	MaxPeriod time.Duration
	MaxGap    int64
}

func (l Limits) valid() bool {
	return l.Sources > 0 && l.Assets > 0 && l.Spans > 0 && l.Counters > 0 && l.Readings > 0 && l.Batch > 0 && l.Page > 0 && l.MaxPeriod <= longestPeriod && l.MaxGap > 0 && l.MaxGap <= int64(l.MaxPeriod/time.Second)
}
