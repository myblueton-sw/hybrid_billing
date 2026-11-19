package accesshistory

import (
	"io"
	"io/fs"
	"sync"
	"time"
)

type handle interface {
	io.ReaderAt
	io.Writer
	io.Seeker
	Sync() error
	Truncate(size int64) error
	Close() error
	Stat() (fs.FileInfo, error)
	Fd() uintptr
}

type system interface {
	openFile(path string, create bool) (handle, error)
	lock(h handle) error
	checkDir(info fs.FileInfo) error
	syncDir(path string) error
	rename(from, to string) error
	remove(path string) error
}

type Hold struct {
	ID       string
	Owner    string
	Reason   string
	Tenants  []string
	Classes  []string
	PlacedAt time.Time
	Released bool
	Covers   func(tenant string) bool
}

type Options struct {
	Dir              string
	Days             int
	DaysSource       string
	MaxBytes         int64
	ClockGap         time.Duration
	AcceptShortening int
	AcceptClock      string
	ReleaseMarker    string
	ReleaseReason    string
	Clock            func() time.Time
	ClockUnverified  func() bool
	Holds            func() []Hold
	Log              func(event string, args ...any)
	Manual           bool
}

type Event struct {
	At        time.Time
	Kind      string
	Reason    string
	Workspace string
	Handle    string
	Browser   string
	OS        string
	Count     int64
}

type Note struct {
	Kind  string
	From  time.Time
	To    time.Time
	Count *int64
}

type Recording struct {
	State  string
	Since  time.Time
	Reason string
}

type Query struct {
	Limit     int
	Workspace string
	CursorW   uint64
	CursorR   uint64
	From      time.Time
	To        time.Time
}

type Page struct {
	Events        []Event
	More          bool
	NextW         uint64
	NextR         uint64
	Days          int
	Source        string
	PeriodSince   time.Time
	EffectiveDays int
	AvailableFrom time.Time
	Recording     Recording
	Notes         []Note
}

type entry struct {
	seq       uint64
	n         uint64
	at        int64
	kind      string
	reason    string
	workspace string
	handle    string
	browser   string
	os        string
	count     int64
}

type sequence struct {
	entries []entry
	next    uint64
}

type person struct {
	seqs map[string]*sequence
}

type lossNote struct {
	kind         string
	from, to     int64
	per          map[string]int64
	unattributed int64
}

type periodEntry struct {
	Days  int    `json:"days"`
	Since string `json:"since"`
}

type deferral struct {
	Reference string `json:"reference"`
	Logged    string `json:"logged"`
}

type snapshot struct {
	Owner    string   `json:"owner"`
	Reason   string   `json:"reason"`
	Tenants  []string `json:"tenants"`
	Classes  []string `json:"classes"`
	PlacedAt string   `json:"placed_at"`
}

type marker struct {
	ID          string   `json:"id"`
	Hold        string   `json:"hold"`
	Snapshot    snapshot `json:"snapshot"`
	Run         string   `json:"run"`
	FirstSeen   string   `json:"first_observed"`
	Days        []string `json:"days"`
	HeldDamaged []string `json:"held_damaged,omitempty"`
}

type stateFile struct {
	Days     int           `json:"days"`
	History  []periodEntry `json:"history"`
	Seq      uint64        `json:"seq"`
	LastAt   string        `json:"last_at,omitempty"`
	Deferral *deferral     `json:"deferral,omitempty"`
	Markers  []marker      `json:"markers"`
}

type peopleLine struct {
	Kind     string    `json:"k"`
	User     string    `json:"u,omitempty"`
	Floor    uint64    `json:"f,omitempty"`
	At       int64     `json:"t"`
	Marker   string    `json:"m,omitempty"`
	Hold     string    `json:"hold,omitempty"`
	Snapshot *snapshot `json:"snap,omitempty"`
	Why      string    `json:"why,omitempty"`
	Run      string    `json:"run,omitempty"`
	Year     int       `json:"y,omitempty"`
	Count    int64     `json:"c,omitempty"`
}

type item struct {
	rec     Record
	control func()
	done    chan struct{}
}

type History struct {
	o       Options
	sys     system
	dir     string
	run     string
	lockH   handle
	stateMu sync.Mutex
	offerMu sync.Mutex
	mu      sync.RWMutex
	queue   chan item
	quit    chan struct{}
	wg      sync.WaitGroup

	seq          uint64
	closed       bool
	stopped      bool
	stopSince    int64
	stopReason   string
	dropped      lossCounts
	droppedAll   int64
	dropFrom     int64
	dropTo       int64
	afterClose   int64
	state        stateFile
	people       handle
	peopleSize   int64
	audits       []peopleLine
	floors       map[string]uint64
	floorAt      map[string]int64
	persons      map[string]*person
	notes        []lossNote
	overdue      int64
	deferred     string
	futureFile   bool
	files        map[string]handle
	sizes        map[string]int64
	total        int64
	lastAt       int64
	refused      map[string]int
	more         map[string]int64
	moreSeq      map[string]uint64
	refusedDay   string
	heldCount    lossCounts
	lastPurge    string
	lastRetry    int64
	effective    int
	workspaces   map[string]string
	runMarkers   map[string]bool
	periodChange *Record
	pendingRecs  []Record
	afterDrain   func()
	writtenSeq   uint64
	failPeople   func() bool
	beforeWrite  func()
}
