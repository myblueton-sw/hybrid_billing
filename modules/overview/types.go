package overview

import (
	"errors"
	"io"
	"time"
)

var (
	ErrInvalid  = errors.New("a request or an input is incomplete or malformed")
	ErrNotFound = errors.New("what the request names does not exist in this workspace")
	ErrConflict = errors.New("another fact is already recorded under this identifier")
	ErrRejected = errors.New("a recorded state or rule refuses this request")
	ErrLimit    = errors.New("a configured limit or bound is exceeded")
)

const (
	maxIDBytes    = 128
	maxDigits     = 30
	maxScale      = 6
	maxWorkspaces = 100000
	maxSubjects   = 1000000
	maxFacts      = 10000000
	maxEvidence   = 16
	maxFilters    = 100000
	maxPage       = 1000
	minStale      = time.Minute
	maxStale      = 90 * 24 * time.Hour
	cursorLabel   = "overview_cursor_v1"
	asOfLabel     = "overview_asof_v1"

	KindHuman     = "human"
	KindAssistant = "assistant"
	KindSystem    = "system"

	StageCollected = "collected"
	StageRated     = "rated"
	StageReviewed  = "reviewed"
	StageApproved  = "approved"
	StageIssued    = "issued"
	StageDelivered = "delivered"
	StageSettled   = "settled"
	StageBlocked   = "blocked"

	MeaningCost        = "cost"
	MeaningSales       = "sales"
	MeaningAllocation  = "allocation"
	MeaningOutstanding = "outstanding"

	StatusCurrent = "current"
	StatusStale   = "stale"
	StatusAbsent  = "absent"

	SortDue      = "due"
	SortCustomer = "customer"
	SortPeriod   = "period"
	SortReceived = "received"

	ReasonAINotAllowed     = "ai_not_allowed"
	ReasonNotHuman         = "not_human"
	ReasonFutureOccurrence = "future_occurrence"
	ReasonRevisionConflict = "revision_conflict"
	ReasonScopeUnavailable = "scope_unavailable"
	ReasonCursorMismatch   = "cursor_mismatch"
	ReasonMeaningOwned     = "meaning_owned"
	ReasonSharedIDs        = "shared_identifiers"
	ReasonRandom           = "random_unavailable"
)

type Refusal struct {
	Reason string
}

func (e Refusal) Error() string { return ErrRejected.Error() + ": " + e.Reason }

func (e Refusal) Unwrap() error { return ErrRejected }

type Actor struct {
	ID   string
	Kind string
}

type Selector struct {
	Customer string
	Entity   string
	Contract string
	From     string
	To       string
}

type Visibility struct {
	All       bool
	Selectors []Selector
	Cost      bool
}

type Scope interface {
	Visible(workspace, viewer string) (Visibility, error)
}

type Config struct {
	Scope  Scope
	Random io.Reader
}

type Limits struct {
	Workspaces int
	Subjects   int
	Facts      int
	Filters    int
	Page       int
	Stale      time.Duration
}

type Subject struct {
	Customer string
	Contract string
	Entity   string
	Period   string
	Currency string
}

type Source struct {
	ID       string
	Revision int
}

type Fact struct {
	ID       string
	Subject  Subject
	Source   Source
	Stage    string
	Due      string
	Owner    string
	Next     string
	Amounts  map[string]string
	Evidence []string
	Occurred time.Time
}

type Entry struct {
	Fact       Fact
	Received   time.Time
	Actor      string
	Superseded bool
}

type Row struct {
	Subject  Subject
	Stage    string
	Due      string
	Owner    string
	Next     string
	Amounts  map[string]string
	Status   string
	Received time.Time
	Occurred time.Time
}

type Filter struct {
	Customers  []string
	Contracts  []string
	Entities   []string
	Periods    []string
	Currencies []string
	Stages     []string
	Owners     []string
	DueBy      string
	Status     string
}

type Sort struct {
	Key        string
	Descending bool
}

type Query struct {
	AsOf        string
	Filter      Filter
	FilterOwner string
	FilterID    string
	Sort        Sort
	Cursor      string
	Limit       int
}

type Total struct {
	Currency  string
	Meaning   string
	Current   string
	Stale     string
	Rows      int
	StaleRows int
	Unknown   int
}

type Page struct {
	AsOf      string
	Evaluated time.Time
	Rows      []Row
	Totals    []Total
	Count     int
	Stale     int
	Next      string
}

type Detail struct {
	AsOf      string
	Evaluated time.Time
	Row       Row
	Timeline  []Entry
}

type Saved struct {
	ID     string
	Filter Filter
	Sort   Sort
	Shared bool
}

type SavedView struct {
	Saved Saved
	Owner string
	At    time.Time
}
