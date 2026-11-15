package events

import (
	"errors"
	"time"
)

var (
	ErrInvalid  = errors.New("a request or an input is incomplete or malformed")
	ErrNotFound = errors.New("what the request names does not exist in this workspace")
	ErrConflict = errors.New("another fact is already recorded under this identifier")
	ErrRejected = errors.New("a recorded state or rule refuses this request")
	ErrLimit    = errors.New("a configured limit or bound is exceeded")
	ErrExpired  = errors.New("the cursor points before the retained events")
	ErrStale    = errors.New("the reservation was replaced or does not exist")
)

const (
	maxIDBytes        = 128
	maxSourceBytes    = 256
	maxFieldName      = 64
	maxFields         = 16
	maxRevisionDigits = 20
	maxVerified       = 10000
	maxWorkspaces     = 100000
	maxTypes          = 1000
	maxEvents         = 10000000
	maxPage           = 10000
	maxSources        = 10000
	maxSeen           = 10000000
	maxRevisions      = 10000000
	minRetention      = time.Hour
	maxRetention      = 3650 * 24 * time.Hour
	minReservation    = time.Second
	maxReservation    = 24 * time.Hour
	specVersion       = "1.0"

	KindHuman     = "human"
	KindAssistant = "assistant"
	KindSystem    = "system"

	StateProposed = "proposed"
	StateActive   = "active"
	StateRevoked  = "revoked"

	DecisionAccepted   = "accepted"
	DecisionDuplicate  = "duplicate"
	DecisionConflict   = "conflict"
	DecisionInProgress = "in_progress"
	DecisionStale      = "stale"

	ReasonAINotAllowed   = "ai_not_allowed"
	ReasonNotHuman       = "not_human"
	ReasonNotSystem      = "not_system"
	ReasonSelfApproval   = "self_approval"
	ReasonNotPending     = "not_pending"
	ReasonUnknownType    = "unknown_type"
	ReasonUnknownField   = "unknown_field"
	ReasonUnverified     = "unverified"
	ReasonSourceNotBound = "source_not_bound"
	ReasonSourceBound    = "source_bound"
	ReasonInboxCapacity  = "inbox_capacity"
	ReasonRevisionLimit  = "revision_capacity"
)

type Refusal struct {
	Reason string
}

func (e Refusal) Error() string { return ErrRejected.Error() + ": " + e.Reason }

func (e Refusal) Unwrap() error { return ErrRejected }

type Expired struct {
	Resync Resync
}

func (e Expired) Error() string { return ErrExpired.Error() }

func (e Expired) Unwrap() error { return ErrExpired }

type Resync struct {
	Floor uint64
	Head  uint64
}

type Actor struct {
	ID   string
	Kind string
}

type Limits struct {
	Workspaces     int
	Types          int
	Events         int
	Retention      time.Duration
	Page           int
	Sources        int
	Seen           int
	InboxRetention time.Duration
	Revisions      int
	Reservation    time.Duration
}

type Type struct {
	Name          string
	SchemaVersion string
	Fields        []string
}

type TypeView struct {
	Type     Type
	State    string
	Proposer string
	Approver string
}

type Event struct {
	ID             string
	Source         string
	Type           string
	SchemaVersion  string
	Subject        string
	ObjectRevision string
	Data           map[string]string
}

type Stored struct {
	Event       Event
	Sequence    uint64
	Time        time.Time
	SpecVersion string
}

type Page struct {
	Events []Stored
	Head   uint64
}

type Binding struct {
	Source     string
	Connection string
	State      string
	Proposer   string
	Approver   string
}

type Reception struct {
	Decision string
	Token    string
	Event    Event
}
