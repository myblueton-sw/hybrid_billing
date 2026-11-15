package authpolicy

import (
	"errors"
	"time"
)

var (
	ErrInvalid  = errors.New("request is incomplete or inconsistent")
	ErrNotFound = errors.New("no record is known under this identifier")
	ErrLimit    = errors.New("a configured limit is reached")
	ErrRejected = errors.New("a recorded state or rule refuses this request")
)

const (
	KindHuman     = "human"
	KindSystem    = "system"
	KindAssistant = "assistant"

	RoleMember        = "member"
	RoleAdministrator = "administrator"

	MethodLDAP     = "ldap"
	MethodOIDC     = "oidc"
	MethodPassword = "password"
	MethodSAML     = "saml"

	FactorIdP      = "idp_mfa"
	FactorOTP      = "otp"
	FactorWebAuthn = "webauthn"

	MFANone           = "none"
	MFAAdministrators = "administrators"
	MFAEveryone       = "everyone"

	ActionPolicyChange = "policy.change"

	ChangeRevision = "revision"
	ChangePlan     = "plan"
	ChangeLift     = "lift"

	ProposalPending   = "pending"
	ProposalApplied   = "applied"
	ProposalWithdrawn = "withdrawn"

	PlanRunning  = "running"
	PlanComplete = "complete"
	PlanLifted   = "lifted"
)

const (
	ReasonAINotAllowed         = "ai_not_allowed"
	ReasonNotHuman             = "not_human"
	ReasonNotSystem            = "not_system"
	ReasonOtherWorkspace       = "other_workspace"
	ReasonNotAdministrator     = "not_administrator"
	ReasonSelfApproval         = "self_approval"
	ReasonNotProposer          = "not_proposer"
	ReasonNotPending           = "not_pending"
	ReasonApproverLockedOut    = "approver_locked_out"
	ReasonInScope              = "in_scope"
	ReasonNoPolicy             = "no_policy"
	ReasonBlocked              = "blocked"
	ReasonMethodNotAllowed     = "method_not_allowed"
	ReasonIssuerNotAllowed     = "issuer_not_allowed"
	ReasonFutureAuthentication = "future_authentication"
	ReasonStale                = "stale"
	ReasonMFARequired          = "mfa_required"
	ReasonStepUpRequired       = "step_up_required"
	ReasonStepUpMFARequired    = "step_up_mfa_required"
	ReasonOutOfOrder           = "out_of_order"
	ReasonLifted               = "lifted"
)

var (
	Methods      = []string{MethodLDAP, MethodOIDC, MethodPassword, MethodSAML}
	Factors      = []string{FactorIdP, FactorOTP, FactorWebAuthn}
	MFAScopes    = []string{MFANone, MFAAdministrators, MFAEveryone}
	MinAge       = time.Minute
	MaxPolicyAge = 30 * 24 * time.Hour
)

type Refusal struct {
	Reason string
}

func (r Refusal) Error() string { return ErrRejected.Error() + ": " + r.Reason }

func (r Refusal) Unwrap() error { return ErrRejected }

type Principal struct {
	User      string
	Workspace string
	Role      string
	Kind      string
}

type Authentication struct {
	Method  string
	Issuer  string
	Factors []string
	At      time.Time
}

type StepUp struct {
	Action string
	MaxAge time.Duration
	MFA    bool
}

type Revision struct {
	Workspace string
	Methods   []string
	Issuers   []string
	MFA       string
	MaxAge    time.Duration
	StepUp    []StepUp
}

type RevisionView struct {
	Revision Revision
	Number   uint64
	Proposer string
	Approver string
	At       time.Time
}

type Decision struct {
	Revision uint64
	Version  uint64
}

type Plan struct {
	ID        string
	Workspace string
	Users     []string
	Chunk     int
	Reason    string
}

type Lift struct {
	Plan   string
	Reason string
}

type Change struct {
	Kind     string
	Revision Revision
	Plan     Plan
	Lift     Lift
}

type PlanView struct {
	Plan      Plan
	State     string
	Chunks    int
	Completed int
	Next      int
	Proposer  string
	Approver  string
	Lifted    [2]string
}

type ProposalView struct {
	Number    uint64
	Change    Change
	Workspace string
	State     string
	Proposer  string
	Approver  string
}
