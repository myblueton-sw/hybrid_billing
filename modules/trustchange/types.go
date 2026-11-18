package trustchange

import (
	"errors"
	"time"
)

const (
	RouteLocalPassword = "local_password"

	StateDraft      = "draft"
	StateValidated  = "validated"
	StateApproved   = "independently_approved"
	StateActivated  = "activated"
	StateRejected   = "rejected"
	StateCancelled  = "cancelled"
	StateSuperseded = "superseded"
	StateUnknown    = "activation_unknown"

	ResultValidated = "validated"
	ResultBlocked   = "blocked"

	ReasonNotFound      = "not_found"
	ReasonStage         = "invalid_stage"
	ReasonSelf          = "self_approval"
	ReasonEpoch         = "epoch"
	ReasonApprovalStale = "approval_stale"
	ReasonEvidenceStale = "evidence_stale"
	ReasonNeedsApproval = "needs_approval"
	ReasonNotProposer   = "not_proposer"
	ReasonDuplicate     = "duplicate"
	ReasonUnchanged     = "unchanged"
	ReasonLimit         = "limit"
	ReasonInvalid       = "invalid"

	HistoryInvalidated = "approval_invalidated"
	HistoryKept        = "revalidated_approval_kept"

	MaxOpenPerProposer  = 25
	MaxOpenPerWorkspace = 100
	MaxKept             = 200
	MaxHistory          = 20
	RunBytes            = 4
)

var ErrInvalid = errors.New("trust change registry options are invalid")

type Refusal struct{ Reason string }

func (r Refusal) Error() string { return "trust change refused: " + r.Reason }

type Rule struct {
	Action string
	MaxAge time.Duration
	MFA    bool
}

type Candidate struct {
	Methods []string
	MFA     string
	MaxAge  time.Duration
	StepUp  []Rule
}

type Base struct {
	Epoch  uint64
	Policy uint64
	Digest string
}

type Item struct {
	Item   string
	Result string
	Reason string
	Detail string
}

type Evidence struct {
	Digest  string
	Blocked bool
	Items   []Item
}

type Validation struct {
	Result string
	At     time.Time
	Digest string
	Items  []Item
}

type Entry struct {
	State     string
	By        string
	At        time.Time
	Rationale string
}

type Activation struct {
	Epoch     uint64
	Sequence  uint64
	Watermark string
	At        time.Time
}

type View struct {
	ID              string
	Number          uint64
	Run             string
	Workspace       string
	Route           string
	Revision        uint64
	OldEpoch        uint64
	OldPolicy       uint64
	OldDigest       string
	NewDigest       string
	ChangeDigest    string
	Candidate       Candidate
	State           string
	Proposer        string
	Validator       string
	Approver        string
	Activator       string
	Validation      *Validation
	ApprovedAt      time.Time
	SupersededBy    string
	SupersededEpoch uint64
	Activation      *Activation
	Unknown         string
	EarlierRun      bool
	RunStartedAt    time.Time
	History         []Entry
}

type Proposal struct {
	Workspace string
	Route     string
	Proposer  string
	Candidate Candidate
	Base      Base
	Rationale string
}

type Approval struct {
	ID             string
	Workspace      string
	Actor          string
	ChangeDigest   string
	EvidenceDigest string
	Recomputed     string
	Base           Base
	Rationale      string
}

type Outcome struct {
	Err        error
	Unknown    string
	Activation Activation
}

func Open(state string) bool {
	return state == StateDraft || state == StateValidated || state == StateApproved
}
