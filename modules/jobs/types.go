package jobs

import (
	"errors"
	"time"
)

var (
	ErrInvalid  = errors.New("request is incomplete or malformed")
	ErrNotFound = errors.New("job or task does not exist in this workspace")
	ErrConflict = errors.New("request conflicts with what is stored")
	ErrStale    = errors.New("the attempt is not the open, latest attempt of the task")
	ErrRejected = errors.New("the state of the job or task does not allow this step")
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

type TaskState string

const (
	Pending       TaskState = "pending"
	Leased        TaskState = "leased"
	TaskSucceeded TaskState = "succeeded"
	TaskFailed    TaskState = "failed"
	Unknown       TaskState = "unknown"
	TaskCancelled TaskState = "cancelled"
)

type JobState string

const (
	Queued    JobState = "queued"
	Running   JobState = "running"
	Waiting   JobState = "waiting"
	Succeeded JobState = "succeeded"
	Failed    JobState = "failed"
	Cancelled JobState = "cancelled"
)

func (s JobState) Terminal() bool { return s == Succeeded || s == Failed || s == Cancelled }

type WaitReason string

const (
	OutcomeUnknown WaitReason = "outcome_unknown"
	CancelPending  WaitReason = "cancel_pending"
)

type CancelReason string

const (
	Requested           CancelReason = "requested"
	PermissionWithdrawn CancelReason = "permission_withdrawn"
	Superseded          CancelReason = "superseded"
)

type Resolution string

const (
	ResolveSucceeded Resolution = "succeeded"
	ResolveFailed    Resolution = "failed"
	ResolveRetry     Resolution = "retry"
)

const (
	CodeLeaseExpired       = "lease_expired"
	CodeCancelledWhileOpen = "cancelled_while_open"
)

type Policy struct {
	Lease       time.Duration
	Life        time.Duration
	Attempts    int
	BackoffBase time.Duration
	BackoffCap  time.Duration
}

func (p Policy) valid() bool {
	return p.Lease > 0 && p.Life >= p.Lease && p.Attempts >= 1 && p.BackoffBase > 0 && p.BackoffCap >= p.BackoffBase
}

func (p Policy) backoff(attempt int) time.Duration {
	delay := p.BackoffBase
	for step := 1; step < attempt; step++ {
		if delay >= p.BackoffCap-delay {
			return p.BackoffCap
		}
		delay *= 2
	}
	return delay
}

type TaskSpec struct {
	ID         string
	Input      string
	Repeatable bool
}

type Lease struct {
	WorkspaceID string
	JobID       string
	TaskID      string
	Input       string
	Checkpoint  string
	Token       uint64
	Attempt     int
	Expiry      time.Time
}

type Ref struct {
	WorkspaceID string
	JobID       string
	TaskID      string
	Token       uint64
}

func (l Lease) Ref() Ref {
	return Ref{WorkspaceID: l.WorkspaceID, JobID: l.JobID, TaskID: l.TaskID, Token: l.Token}
}

type Report struct {
	State  TaskState
	Replay bool
}

type TaskView struct {
	ID         string
	Input      string
	Repeatable bool
	State      TaskState
	Attempts   int
	Token      uint64
	Open       bool
	Code       string
	Checkpoint string
	NotBefore  time.Time
	Expiry     time.Time
}

type JobView struct {
	WorkspaceID string
	ID          string
	Type        string
	Policy      Policy
	State       JobState
	Reason      WaitReason
	Attention   int
	Cancel      CancelReason
	Revision    uint64
	Counts      map[TaskState]int
	Tasks       []TaskView
	Overdue     int
	AsOf        time.Time
}

type Event string

const (
	EventCreated   Event = "created"
	EventGranted   Event = "granted"
	EventExpired   Event = "expired"
	EventCompleted Event = "completed"
	EventFailed    Event = "failed"
	EventUnknown   Event = "unknown"
	EventResolved  Event = "resolved"
	EventCancel    Event = "cancel_requested"
)

type Transition struct {
	Revision uint64
	At       time.Time
	Task     string
	Worker   string
	Event    Event
	State    JobState
}

type Limits struct {
	Jobs         int
	Tasks        int
	Attempts     int
	PerType      int
	PerWorkspace int
	Batch        int
}

func (l Limits) valid() bool {
	return l.Jobs > 0 && l.Tasks > 0 && l.Attempts > 0 && l.PerType > 0 && l.PerWorkspace > 0 && l.PerWorkspace <= l.PerType && l.Batch > 0
}

func derive(counts map[TaskState]int, cancelRequested, granted bool) (JobState, WaitReason) {
	total := 0
	for _, count := range counts {
		total += count
	}
	switch {
	case cancelRequested && counts[Leased] == 0 && counts[Unknown] == 0:
		return Cancelled, ""
	case cancelRequested:
		return Waiting, CancelPending
	case counts[TaskSucceeded] == total:
		return Succeeded, ""
	case counts[Pending] == 0 && counts[Leased] == 0 && counts[Unknown] == 0:
		return Failed, ""
	case counts[Pending] == 0 && counts[Leased] == 0:
		return Waiting, OutcomeUnknown
	case !granted:
		return Queued, ""
	}
	return Running, ""
}
