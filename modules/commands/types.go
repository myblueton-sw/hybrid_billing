package commands

import (
	"errors"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

var (
	ErrInvalid     = errors.New("definition or limits are incomplete or inconsistent")
	ErrConflict    = errors.New("command identifier is already registered")
	ErrUnavailable = errors.New("command registry is unavailable")
)

type Code string

const (
	Unauthenticated       Code = "unauthenticated"
	PermissionDenied      Code = "permission_denied"
	SelfApprovalDenied    Code = "self_approval_denied"
	RevisionConflict      Code = "revision_conflict"
	IdempotencyConflict   Code = "idempotency_conflict"
	InvalidRequest        Code = "invalid_request"
	InvalidStage          Code = "invalid_stage"
	PrerequisiteFailed    Code = "prerequisite_failed"
	UnsupportedCommand    Code = "unsupported_command"
	UnsupportedCapability Code = "unsupported_capability"
	AuthorityUnavailable  Code = "authority_unavailable"
	DependencyUnavailable Code = "dependency_unavailable"
	InternalError         Code = "internal_error"
)

type NextAction string

const (
	NextNone      NextAction = "none"
	NextAnswer    NextAction = "obtain_new_access_answer"
	NextInspect   NextAction = "inspect_current_state"
	NextCorrect   NextAction = "correct_request"
	NextRetry     NextAction = "retry_later"
	NextOperation NextAction = "inspect_operation"
)

type errorRule struct {
	status int
	retry  bool
	next   NextAction
}

var errorRules = map[Code]errorRule{
	Unauthenticated:       {401, false, NextAnswer},
	PermissionDenied:      {403, false, NextNone},
	SelfApprovalDenied:    {403, false, NextNone},
	RevisionConflict:      {409, false, NextInspect},
	IdempotencyConflict:   {409, false, NextInspect},
	InvalidRequest:        {422, false, NextCorrect},
	InvalidStage:          {422, false, NextCorrect},
	PrerequisiteFailed:    {422, false, NextCorrect},
	UnsupportedCommand:    {404, false, NextNone},
	UnsupportedCapability: {501, false, NextNone},
	AuthorityUnavailable:  {503, true, NextRetry},
	DependencyUnavailable: {503, true, NextRetry},
	InternalError:         {500, false, NextNone},
}

var handlerCodes = map[Code]bool{
	UnsupportedCapability: true, InvalidStage: true, RevisionConflict: true, PrerequisiteFailed: true,
	DependencyUnavailable: true, PermissionDenied: true, InvalidRequest: true,
}

type Error struct {
	Code      Code
	Status    int
	Retry     bool
	Next      NextAction
	Request   string
	Fields    []string
	Operation string
}

type Failure struct {
	Code   Code
	Fields []string
}

func (f *Failure) Error() string { return string(f.Code) }

func Fail(code Code, fields ...string) error {
	return &Failure{Code: code, Fields: append([]string(nil), fields...)}
}

type FieldType string

const (
	ID        FieldType = "id"
	Decimal   FieldType = "decimal"
	Integer   FieldType = "integer"
	Boolean   FieldType = "boolean"
	Text      FieldType = "text"
	Timestamp FieldType = "timestamp"
	Enum      FieldType = "enum"
)

type Field struct {
	Name     string
	Type     FieldType
	Required bool
	Values   []string
}

type Permission struct {
	Action gate.Action
	Kind   string
	Fields []string
}

type Resolution int

const (
	NotFound Resolution = iota
	Found
	Unavailable
)

type Target struct {
	CustomerID    string
	LegalEntityID string
	ContractID    string
	Month         gate.Month
	HasMonth      bool
	Revision      string
	AuthorID      string
}

type Context struct {
	Workspace string
	User      string
	Client    string
	Operation string
	Request   string
	Target    string
	Resolved  string
	Expected  string
	Rationale string
	Fields    map[string]string
	At        time.Time
}

type Outcome struct {
	Accepted       bool
	JobID          string
	ResultRevision string
	BusinessState  string
	Fields         map[string]string
}

type Handler interface {
	Resolve(workspace, target string) (Target, Resolution)
	Apply(Context) (Outcome, error)
}

type Definition struct {
	ID         string
	Permission Permission
	Approval   bool
	Mutating   bool
	Rationale  bool
	Fields     []Field
	Handler    Handler
}

type Value struct {
	Name  string
	Value string
}

type Call struct {
	Answer    gate.Answer
	Client    string
	Request   string
	Command   string
	Target    string
	Key       string
	Expected  string
	Rationale string
	Fields    []Value
}

type Status string

const (
	Completed Status = "completed"
	Accepted  Status = "accepted"
)

type Result struct {
	Err       *Error
	Status    Status
	Request   string
	Operation string
	Target    string
	Revision  string
	State     string
	Job       string
	Fields    map[string]string
	AsOf      time.Time
	Replay    bool
}

type Limits struct {
	Fields     int
	Value      int
	Rationale  int
	Operations int
	Journal    int
}

func (l Limits) valid() bool {
	for _, value := range []int{l.Fields, l.Value, l.Rationale, l.Operations, l.Journal} {
		if value < 1 {
			return false
		}
	}
	return true
}

const Marker = "not_recorded"

type Record struct {
	Sequence  uint64
	At        time.Time
	Request   string
	Command   string
	User      string
	Client    string
	Workspace string
	Target    string
	Outcome   string
	Operation string
	Replay    bool
	Digest    string
}

type Entry struct {
	ID         string
	Permission Permission
	Approval   bool
	Mutating   bool
	Rationale  bool
	Fields     []Field
	Supported  bool
}

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

func marked(value string) string {
	if validID(value) {
		return value
	}
	return Marker
}
