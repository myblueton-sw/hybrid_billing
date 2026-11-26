package retention

import (
	"errors"
	"time"
)

var (
	ErrInvalid  = errors.New("request is incomplete or inconsistent")
	ErrNotFound = errors.New("no record is known under this identifier")
	ErrConflict = errors.New("another record is already kept under this identifier")
	ErrLimit    = errors.New("a configured limit is reached")
	ErrRejected = errors.New("a recorded state or rule refuses this request")
)

const (
	KindHuman     = "human"
	KindSystem    = "system"
	KindAssistant = "assistant"

	ClassSource       = "source"
	ClassFinal        = "final"
	ClassClaim        = "claim"
	ClassIdentity     = "identity"
	ClassConversation = "conversation"
	ClassTemporary    = "temporary"
	ClassProjection   = "projection"

	Requires = "requires"
	Mentions = "mentions"
	Guards   = "guards"

	ChangePolicy  = "policy"
	ChangePlan    = "plan"
	ChangeRelease = "release"

	ProposalPending   = "pending"
	ProposalApplied   = "applied"
	ProposalWithdrawn = "withdrawn"

	OutcomePending    = "pending"
	OutcomeDeleting   = "deleting"
	OutcomeDeleted    = "deleted"
	OutcomeFailed     = "failed"
	OutcomeHeld       = "held"
	OutcomeReferenced = "referenced"
	OutcomeGuarding   = "guarding"
	OutcomeNotDue     = "not_due"
	OutcomeStopped    = "stopped"

	StoreDeleted = "deleted"
	StoreFailed  = "failed"

	HoldNotPreserved   = "not_preserved"
	HoldOutcomePending = "outcome_pending"
	HoldPartlyDeleted  = "partially_deleted"

	RunRunning  = "running"
	RunStopped  = "stopped"
	RunDone     = "done"
	RunComplete = "complete"

	AccessOK      = "ok"
	AccessBlocked = "blocked"
	AccessDeleted = "deleted"
)

const (
	ReasonAINotAllowed   = "ai_not_allowed"
	ReasonNotHuman       = "not_human"
	ReasonNotSystem      = "not_system"
	ReasonSelfApproval   = "self_approval"
	ReasonNotProposer    = "not_proposer"
	ReasonNotPending     = "not_pending"
	ReasonDeleted        = "deleted"
	ReasonDeleting       = "deleting"
	ReasonImpactRequired = "impact_required"
	ReasonImpactChanged  = "impact_changed"
	ReasonAlreadyDeleted = "already_deleted"
	ReasonInPlan         = "in_plan"
	ReasonOutOfOrder     = "out_of_order"
	ReasonLeased         = "leased"
	ReasonStaleToken     = "stale_token"
	ReasonNotClaimed     = "not_claimed"
	ReasonNothingToRetry = "nothing_to_retry"
	ReasonPlacer         = "placer"
	ReasonReleased       = "released"
	ReasonBackward       = "backward"
	ReasonChunkDone      = "chunk_done"
	ReasonNotInChunk     = "not_in_chunk"
)

var Classes = []string{ClassClaim, ClassConversation, ClassFinal, ClassIdentity, ClassProjection, ClassSource, ClassTemporary}

type Refusal struct {
	Reason string
}

func (r Refusal) Error() string { return ErrRejected.Error() + ": " + r.Reason }

func (r Refusal) Unwrap() error { return ErrRejected }

type Actor struct {
	ID   string
	Kind string
}

type Artifact struct {
	ID      string
	Class   string
	Tenants []string
	Anchor  time.Time
	Stores  []string
}

type Reference struct {
	From string
	To   string
	Kind string
}

type Policy struct {
	Class  string
	Period int
	Grace  int
}

type Impact struct {
	AsOf   time.Time
	Count  int
	Digest string
}

type ImpactView struct {
	Impact     Impact
	Tenants    map[string]int
	Held       int
	Referenced int
	Guarding   int
}

type PolicyView struct {
	Policy   Policy
	Number   uint64
	Proposer string
	Approver string
	At       time.Time
}

type Hold struct {
	ID        string
	Owner     string
	Reason    string
	Artifacts []string
	Tenants   []string
	Classes   []string
}

type HoldOutcome struct {
	Artifact string
	Outcome  string
	Stores   map[string]string
}

type HoldView struct {
	Hold     Hold
	Placer   string
	Released bool
	Counts   map[string]int
}

type DeletionPlan struct {
	Class     string
	Artifacts []string
	Chunk     int
	Lease     time.Duration
	Reason    string
}

type Release struct {
	Hold   string
	Reason string
}

type Change struct {
	Kind    string
	Policy  Policy
	Impact  Impact
	Plan    DeletionPlan
	Release Release
}

type ProposalView struct {
	Number   uint64
	Change   Change
	State    string
	Proposer string
	Approver string
}

type Claim struct {
	Run      string
	Chunk    int
	Token    uint64
	Deleting []string
	Until    time.Time
}

type StoreCounts struct {
	Target  int
	Deleted int
	Failed  int
	Pending int
}

type RunView struct {
	ID       string
	Plan     DeletionPlan
	Revision uint64
	State    string
	Next     int
	Chunks   int
	Damaged  int
	Counts   map[string]int
	Stores   map[string]StoreCounts
	Outcomes map[string]string
	Proposer string
	Approver string
	Stopper  string
}

type PreviewView struct {
	Eligible   []string
	More       bool
	Held       int
	Referenced int
	Guarding   int
}

type Tombstone struct {
	Sequence uint64
	Artifact string
	Class    string
	Run      string
	At       time.Time
}
