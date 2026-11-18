package bulkrevoke

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalid   = errors.New("request is incomplete or malformed")
	ErrNotFound  = errors.New("draft or bulk revocation does not exist here")
	ErrTooMany   = errors.New("the draft would exceed the people limit")
	ErrOpenDraft = errors.New("the caller already has an open draft")
	ErrPending   = errors.New("the pending draft and list limit is reached")
	ErrState     = errors.New("the draft or bulk revocation is not in a state that allows this step")
	ErrDigest    = errors.New("the digest does not match the prepared list")
	ErrEmpty     = errors.New("the prepared list has no people to revoke")
)

type Effect string

const (
	SuspendMembership Effect = "suspend_membership"
	SuspendUser       Effect = "suspend_user"
)

func (e Effect) Valid() bool { return e == SuspendMembership || e == SuspendUser }

type State string

const (
	Draft     State = "draft"
	Prepared  State = "prepared"
	Executed  State = "executed"
	Discarded State = "discarded"
	Expired   State = "expired"
)

type Outcome string

const (
	OutcomeExcluded    Outcome = "excluded"
	OutcomeHeld        Outcome = "held"
	OutcomeSucceeded   Outcome = "succeeded"
	OutcomeFailed      Outcome = "failed"
	OutcomeUnprocessed Outcome = "unprocessed"
)

const (
	Received       = "received"
	DenyConfirmed  = "deny_confirmed"
	CleanupRunning = "cleanup_running"
	Complete       = "revocation_complete"
	Unconfirmed    = "decision_unconfirmed"
)

const (
	ExcludedSelf           = "self"
	ExcludedNotFound       = "not_found"
	ExcludedNoMembership   = "no_membership"
	ExcludedAlreadyRevoked = "already_revoked"
	HeldChanged            = "changed_since_preview"
	HeldSelf               = "self"
	RecordNotConfirmed     = "record_not_confirmed"
)

const (
	Explicit       = "explicit"
	OriginCommand  = "command"
	OriginReplay   = "ledger_replay"
	DraftPrefix    = "bulk-draft-"
	ManifestPrefix = "bulk-revocation-"
	digestLabel    = "bulk_revocation_v1"
	maxIDBytes     = 128
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

func ValidUser(id string) bool { return validID(id) }

func Number(id, prefix string) (uint64, bool) {
	rest, ok := strings.CutPrefix(id, prefix)
	if !ok || rest == "" || rest[0] == '0' {
		return 0, false
	}
	n, err := strconv.ParseUint(rest, 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

func DraftID(n uint64) string { return DraftPrefix + strconv.FormatUint(n, 10) }

func ManifestID(n uint64) string { return ManifestPrefix + strconv.FormatUint(n, 10) }

type Limits struct {
	MaxTargets int
	Pending    int
	Kept       int
	Lifetime   time.Duration
}

func (l Limits) valid() bool {
	return l.MaxTargets > 0 && l.Pending > 0 && l.Kept > 0 && l.Lifetime > 0
}

type Target struct {
	User            string
	Login           string
	Role            string
	Until           time.Time
	Version         uint64
	Epoch           uint64
	UserState       string
	MembershipState string
	Administrator   bool
}

type Exclusion struct {
	User   string
	Login  string
	Reason string
}

type PendingStore struct {
	Store     string
	Count     int
	Rechecked bool
}

type Impact struct {
	ComputedAt                 time.Time
	People                     int
	Administrators             int
	Members                    int
	InstallationAdministrators int
	TrustHolders               int
	BulkHolders                int
	Sessions                   int
	Pickups                    int
	ServiceUsers               int
	AnchorsAfter               int
	Pending                    []PendingStore
}

type Receipt struct {
	Sequence  uint64
	Watermark string
	Reason    string
}

type Cleanup struct {
	Item   string
	State  string
	Reason string
}

type Item struct {
	Index   int
	Target  Target
	Outcome Outcome
	Reason  string
	State   string
	Receipt Receipt
	Cleanup []Cleanup
	Note    string
}

type Counts struct {
	Targets     int
	Excluded    int
	Duplicates  int
	Succeeded   int
	Held        int
	Failed      int
	Unprocessed int
}

type View struct {
	ID              string
	Number          uint64
	Workspace       string
	State           State
	Origin          string
	Owner           string
	Effect          Effect
	Selection       string
	Revision        string
	Count           int
	Duplicates      int
	Updated         time.Time
	PreparedAt      time.Time
	ExpiresAt       time.Time
	Targets         []Target
	Exclusions      []Exclusion
	Impact          Impact
	Digest          string
	Executor        string
	ExecutedAt      time.Time
	DenyAt          time.Time
	DenyGeneration  uint64
	Header          Receipt
	Headed          bool
	CleanupCreated  bool
	CleanupNotReady bool
	RecordedAt      time.Time
	ReplayHeld      int
	ReplayExcluded  int
	ReplayMissing   int
	Items           []Item
	Counts          Counts
}

type Cleaning struct {
	State  string
	Reason string
}

const (
	JobRunning         = "running"
	JobWaiting         = "waiting"
	JobSucceeded       = "succeeded"
	ReasonUnconfirmed  = "decision_unconfirmed"
	ReasonNotStarted   = "cleanup_not_started"
	CleaningQueued     = "queued"
	CleaningRunning    = "running"
	CleaningWaiting    = "waiting"
	CleaningSucceeded  = "succeeded"
	CleaningFailed     = "failed"
	CleaningCancelled  = "cancelled"
	CleaningNotCreated = ""
)
