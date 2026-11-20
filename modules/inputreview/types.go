package inputreview

import (
	"errors"
	"time"
)

var (
	ErrInvalid  = errors.New("input review request is malformed")
	ErrNotFound = errors.New("input review record is absent")
	ErrConflict = errors.New("input review evidence or revision changed")
	ErrRejected = errors.New("input review prerequisite refuses the operation")
	ErrLimit    = errors.New("input review capacity or revision is exhausted")
)

const (
	Match                = "match"
	Difference           = "difference"
	Awaiting             = "awaiting_confirmation"
	Blocked              = "blocked"
	Eligible             = "eligible"
	Rejected             = "rejected"
	ConfirmAction        = "input.confirm_publish"
	RejectAction         = "input.reject"
	MaxRevision   uint64 = 1<<32 - 1
)

type Scope struct {
	Workspace string
	Provider  string
	Account   string
	Period    string
}

type Evidence struct {
	Scope    Scope
	Run      string
	Digest   string
	Method   string
	Verdict  string
	Findings []string
}

type Authority interface {
	Check(scope Scope, actor, action string) error
}

type CurrentEvidence interface {
	Compare(workspace, run string) (Evidence, error)
}

type Intent struct {
	Workspace     string
	ID            string
	Actor         string
	Rationale     string
	Digest        string
	ExpectedScope uint64
}

type Decision struct {
	Actor         string
	Rationale     string
	Digest        string
	ExpectedScope uint64
	ScopeRevision uint64
	At            time.Time
}

type Record struct {
	ID       string
	Evidence Evidence
	State    string
	Decision *Decision
	Compared time.Time
}

type Publication struct {
	Scope    Scope
	Revision uint64
	Record   string
	Digest   string
}
