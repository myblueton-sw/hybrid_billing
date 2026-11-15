package webhooks

import (
	"errors"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/events"
)

var (
	ErrInvalid  = errors.New("a request or an input is incomplete or malformed")
	ErrNotFound = errors.New("what the request names does not exist in this workspace")
	ErrConflict = errors.New("another fact is already recorded under this identifier")
	ErrRejected = errors.New("a recorded state or rule refuses this request")
	ErrLimit    = errors.New("a configured limit or bound is exceeded")
	ErrStale    = errors.New("the lease is not the open, latest lease of the delivery")
)

const (
	maxIDBytes      = 128
	maxWorkspaces   = 100000
	maxSubscription = 10000
	maxTypes        = 32
	maxAttempts     = 100
	maxDead         = 10000
	maxBatch        = 10000
	maxWork         = 10000
	minLease        = time.Second
	maxLease        = time.Hour
	minBackoff      = time.Second
	maxMaxBackoff   = 24 * time.Hour
	maxMaxAge       = 30 * 24 * time.Hour
	jitterPercent   = 20
	signatureTag    = "hybrid-billing/webhook/v1"

	KindHuman     = "human"
	KindAssistant = "assistant"
	KindSystem    = "system"

	DestinationActive   = "active"
	DestinationInactive = "inactive"
	DestinationRevoked  = "revoked"

	SubscriptionProposed  = "proposed"
	SubscriptionActive    = "active"
	SubscriptionPaused    = "paused"
	SubscriptionSuspended = "suspended"
	SubscriptionRevoked   = "revoked"

	StatusDelivered = "delivered"
	StatusRejected  = "rejected"
	StatusThrottled = "throttled"
	StatusFailed    = "failed"
	StatusUnknown   = "unknown"

	JobQueued    = "queued"
	JobRunning   = "running"
	JobWaiting   = "waiting"
	JobSucceeded = "succeeded"
	JobFailed    = "failed"
	JobCancelled = "cancelled"

	KindOpen       = "open"
	KindRedelivery = "redelivery"
	KindDead       = "dead"
	KindCancelled  = "cancelled"

	ReasonAINotAllowed        = "ai_not_allowed"
	ReasonNotHuman            = "not_human"
	ReasonNotSystem           = "not_system"
	ReasonSelfApproval        = "self_approval"
	ReasonNotPending          = "not_pending"
	ReasonNotActive           = "not_active"
	ReasonRevoked             = "revoked"
	ReasonDestinationInactive = "destination_inactive"
	ReasonDestinationRevoked  = "destination_revoked"
	ReasonSubscribeRefused    = "subscribe_refused"
	ReasonPermissionRevoked   = "permission_revoked"
	ReasonSubscriptionRevoked = "subscription_revoked"
	ReasonDependency          = "dependency_unavailable"
	ReasonGap                 = "gap"
	ReasonBacklog             = "backlog"
	ReasonRejected            = "rejected"
	ReasonAttempts            = "attempts_exhausted"
	ReasonMaxAge              = "max_age"
	ReasonUnapprovedRedirect  = "unapproved_redirect"
	ReasonBackoff             = "backoff"
	ReasonThrottled           = "throttled"
	ReasonPaused              = "paused"
)

type Refusal struct {
	Reason string
}

func (e Refusal) Error() string { return ErrRejected.Error() + ": " + e.Reason }

func (e Refusal) Unwrap() error { return ErrRejected }

type Outbox interface {
	Read(workspace string, after uint64, limit int) (events.Page, error)
	Bounds(workspace string) (uint64, uint64, error)
}

type Destinations interface {
	State(workspace, destination string) (string, error)
}

type Permits interface {
	Subscribe(workspace, subscriber string, types []string) (bool, error)
	Allowed(workspace, subscriber string, event events.Stored) (bool, error)
}

type Signer interface {
	Current(workspace, subscription string) (string, error)
	Sign(workspace, subscription, version string, message []byte) ([]byte, error)
}

type Redirects interface {
	Check(workspace, destination, current string, hop int, location string) (string, string, int, error)
}

type Config struct {
	Outbox       Outbox
	Destinations Destinations
	Permits      Permits
	Signer       Signer
	Redirects    Redirects
}

type Limits struct {
	Workspaces    int
	Subscriptions int
	Types         int
	Lease         time.Duration
	Backoff       time.Duration
	MaxBackoff    time.Duration
	Attempts      int
	MaxAge        time.Duration
	Dead          int
	Batch         int
	Work          int
}

type Actor struct {
	ID   string
	Kind string
}

type Subscription struct {
	ID          string
	Subscriber  string
	Destination string
	Types       []string
}

type Gap struct {
	After uint64
	To    uint64
}

type SubscriptionView struct {
	Subscription Subscription
	State        string
	Reason       string
	Cursor       uint64
	Gaps         []Gap
	Proposer     string
	Approver     string
	Resumer      string
}

type Lease struct {
	Subscription string
	Delivery     string
	Token        string
	Sequence     uint64
	Source       string
	EventID      string
	Destination  string
}

type Signed struct {
	Delivery  string
	Attempt   int
	Version   string
	At        time.Time
	Body      []byte
	Signature []byte
	Headers   map[string]string
}

type Outcome struct {
	Status     string
	RetryAfter time.Duration
}

type DeliveryView struct {
	ID             string
	Subscription   string
	Sequence       uint64
	Source         string
	EventID        string
	Kind           string
	State          string
	Reason         string
	Attempts       int
	OutcomeUnknown bool
}
