package pricingpolicy

import (
	"errors"
	"sync"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/rating"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

var (
	ErrInvalid  = errors.New("invalid synthetic pricing policy")
	ErrNotFound = errors.New("pricing policy not found")
	ErrConflict = errors.New("pricing policy evidence conflicts")
	ErrRejected = errors.New("pricing policy action refused")
	ErrLimit    = errors.New("pricing policy capacity reached")
)

const (
	Pending       = "pending"
	Approved      = "approved"
	Revoked       = "revoked"
	ProposeAction = "pricing_policy.propose"
	ApproveAction = "pricing_policy.approve"
	RevokeAction  = "pricing_policy.revoke"
	UseAction     = "pricing_policy.use"
	Schema        = "synthetic_pricing_policy_v1"
	MaxBytes      = 1 << 20
)

type Scope struct{ Workspace, Customer, Entity, Contract, BillingProfile string }
type Target struct {
	ContractRevision, BillingRevision uint64
	Currency                          string
	Path                              rating.Path
	Fingerprint                       string
}
type Spec struct {
	Mode     string
	Scope    Scope
	From, To time.Time
	Policy   rating.Policy
	Profile  decimal.ProfileSpec
	Limits   rating.Limits
}
type Proposal struct {
	Schema, Algorithm, ID, Proposer, Rationale string
	Spec                                       Spec
	Target                                     Target
}
type Decision struct {
	Actor, Rationale string
	At               time.Time
}
type Record struct {
	Proposal             Proposal
	Digest, State        string
	ProposedAt           time.Time
	Approval, Revocation *Decision
}
type Authority interface {
	Check(Scope, string, string) error
}
type Targets interface {
	Current(Scope, time.Time, time.Time) (Target, error)
}
type Registry struct {
	mu        sync.Mutex
	records   map[[2]string]Record
	authority Authority
	targets   Targets
	clock     func() time.Time
	last      time.Time
	limit     int
}
