package approvals

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
)

const (
	maxIDBytes     = 128
	maxNumberBytes = 64
	minValidity    = time.Hour
	maxValidity    = 90 * 24 * time.Hour
	subjectLabel   = "approval_subject_v1"
	decisionLabel  = "approval_decision_v1"
	executionLabel = "approval_execution_v1"
	proposalLabel  = "approval_proposal_v1"
)

const (
	KindHuman     = "human"
	KindAssistant = "assistant"
	KindSystem    = "system"

	StageCharge   = "charge_approval"
	StageERPFinal = "erp_final_approval"
	StageIssue    = "issue"
	StageDeliver  = "deliver"

	ProposePolicy      = "policy"
	ProposeGrant       = "grant"
	ProposeAddAdmin    = "add_admin"
	ProposeRemoveAdmin = "remove_admin"

	StatePending    = "pending"
	StateApproved   = "approved"
	StateRejected   = "rejected"
	StateWithdrawn  = "withdrawn"
	StateSuperseded = "superseded"
	StateConfirmed  = "confirmed"

	VerdictApprove = "approve"
	VerdictReject  = "reject"

	ResultSucceeded = "succeeded"
	ResultFailed    = "failed"

	ReasonAINotAllowed    = "ai_not_allowed"
	ReasonNotAdmin        = "not_admin"
	ReasonNotPending      = "not_pending"
	ReasonSelfApproval    = "self_approval"
	ReasonNoAuthority     = "no_authority"
	ReasonNothingLeft     = "nothing_left"
	ReasonNotApproved     = "not_approved"
	ReasonSuperseded      = "superseded"
	ReasonWithdrawn       = "withdrawn"
	ReasonPolicyChanged   = "policy_changed"
	ReasonExpired         = "expired"
	ReasonSubjectChanged  = "subject_changed"
	ReasonAuthorityLost   = "authority_lost"
	ReasonItemIneligible  = "item_ineligible"
	ReasonAlreadyExecuted = "already_executed"
	ReasonNoPolicy        = "no_policy"
	ReasonStaleProposal   = "stale_proposal"
	ReasonGranteeActs     = "grantee_acts"
	ReasonTooFewAdmins    = "too_few_admins"
)

var (
	kinds          = map[string]bool{KindHuman: true, KindAssistant: true, KindSystem: true}
	stages         = map[string]bool{StageCharge: true, StageERPFinal: true, StageIssue: true, StageDeliver: true}
	digestKinds    = map[string]bool{"input": true, "rule": true, "mapping": true, "fx": true, "attribution": true, "erp_draft": true, "result": true}
	excludeReasons = map[string]bool{"not_eligible": true, "blocked_error": true, "customer_hold": true, "data_missing": true}
	rejectReasons  = map[string]bool{"incorrect_amount": true, "missing_evidence": true, "policy_violation": true, "other": true}
	failReasons    = map[string]bool{"external_error": true, "timeout": true, "refused_downstream": true}
)

type Refusal struct {
	Reason string
}

func (e Refusal) Error() string { return ErrRejected.Error() + ": " + e.Reason }

func (e Refusal) Unwrap() error { return ErrRejected }

type Actor struct {
	ID   string
	Kind string
}

type Policy struct {
	Stage     string
	Revision  int
	Approvals int
	Required  []string
	Validity  time.Duration
}

type Authority struct {
	ID       string
	Approver string
	Stage    string
	Currency string
	Limit    string
	From     string
	To       string
}

type AuthorityView struct {
	Authority Authority
	Proposer  string
	Confirmer string
	Granted   time.Time
	RevokedBy string
	Revoked   time.Time
}

type Proposal struct {
	ID        string
	Kind      string
	Policy    Policy
	Authority Authority
	Admin     string
}

type ProposalView struct {
	Proposal  Proposal
	State     string
	Proposer  string
	Confirmer string
	Written   time.Time
	Confirmed time.Time
	Digest    string
}

type Entry struct {
	Item   string
	Scope  string
	Amount string
	Digest string
}

type Subject struct {
	ID       string
	Stage    string
	Target   string
	Author   string
	Digests  map[string]string
	Amount   string
	Currency string
	Entries  []Entry
}

type Exclusion struct {
	Item   string
	Reason string
}

type Decision struct {
	Verdict string
	Reason  string
	Exclude []Exclusion
}

type DecisionView struct {
	Decision Decision
	Actor    string
	At       time.Time
	Digest   string
}

type RequestView struct {
	Subject    Subject
	Policy     int
	State      string
	Requester  string
	Written    time.Time
	ApprovedAt time.Time
	EndedBy    string
	Ended      time.Time
	Decisions  []DecisionView
	Excluded   []Exclusion
	Executed   []string
	Digest     string
}

type Execution struct {
	ID         string
	Request    string
	Stage      string
	Target     string
	Item       string
	Digests    map[string]string
	ItemDigest string
	Amount     string
	Currency   string
}

type Outcome struct {
	Result string
	Reason string
}

type ExecutionView struct {
	Execution Execution
	Actor     string
	At        time.Time
	Outcome   Outcome
	Reported  time.Time
	Earlier   Outcome
	Digest    string
}

type Limits struct {
	Workspaces  int
	Admins      int
	Proposals   int
	Authorities int
	Requests    int
	Items       int
	Executions  int
}

func within(value, cap int) bool { return value > 0 && value <= cap }

func (l Limits) valid() bool {
	return within(l.Workspaces, 100000) && l.Admins >= 2 && l.Admins <= 100 && within(l.Proposals, 100000) && within(l.Authorities, 100000) &&
		within(l.Requests, 1000000) && within(l.Items, 100000) && within(l.Executions, 10000000)
}
