package grantauthority

import (
	"errors"
	"time"
)

var (
	ErrInvalid  = errors.New("input authority request is malformed")
	ErrDenied   = errors.New("input authority governance refuses the request")
	ErrConflict = errors.New("input authority proposal or effect conflicts")
	ErrNotFound = errors.New("input authority proposal is absent")
	ErrLimit    = errors.New("input authority capacity is exhausted")
	ErrUnknown  = errors.New("input authority application is unresolved")
)

const (
	Console      = "console_ceiling"
	Human        = "human_grant"
	Pending      = "pending"
	Applying     = "applying"
	Unknown      = "decision_unknown"
	Applied      = "applied"
	Rejected     = "rejected"
	Withdrawn    = "withdrawn"
	Expired      = "expired"
	Superseded   = "superseded"
	ActiveGrant  = "active"
	RevokedGrant = "revoked"
	ExpiredGrant = "expired"
)

type Limits struct {
	Lifetime         time.Duration
	MaxGrantValidity time.Duration
	Capacity         int
}

type Spec struct {
	Capability string
	Workspace  string
	Kind       string
	Recipient  string
	CeilingID  string
	NotBefore  time.Time
	NotAfter   time.Time
}

type Proposal struct {
	ID        string
	Spec      Spec
	Proposer  string
	Rationale string
}

type Intent struct {
	Workspace string
	ID        string
	Actor     string
	Digest    string
	Rationale string
}

type Effect struct {
	Found  bool
	Spec   Spec
	Status string
}

type Authority interface {
	Check(workspace, actor string) (string, error)
}
type Targets interface{ Current(Spec) (string, error) }
type Effects interface {
	Lookup(id string) (Effect, error)
	Apply(id string, spec Spec) error
}

type Decision struct {
	Actor       string
	Rationale   string
	At          time.Time
	GrantID     string
	GrantStatus string
}

type Record struct {
	Proposal   Proposal
	Digest     string
	State      string
	ProposedAt time.Time
	ExpiresAt  time.Time
	Decision   *Decision
}

type Domain struct {
	Version, GrantPrefix string
	Capabilities         []string
}
