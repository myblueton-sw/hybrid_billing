package auth

import (
	"errors"
	"time"
)

var (
	ErrInvalid     = errors.New("record is incomplete or inconsistent")
	ErrConflict    = errors.New("record version conflict")
	ErrNotFound    = errors.New("record does not exist")
	ErrNotBacked   = errors.New("delegation is not backed by its delegator")
	ErrDenied      = errors.New("approver lacks the required authority")
	ErrUnavailable = errors.New("authority is unavailable")
)

type Action string

const (
	View                  Action = "view"
	ViewCost              Action = "view_cost"
	ChangeCustomerPolicy  Action = "change_customer_policy"
	ChangeSellingPolicy   Action = "change_selling_policy"
	RequestApproval       Action = "request_approval"
	Approve               Action = "approve"
	Issue                 Action = "issue"
	Download              Action = "download"
	Deliver               Action = "deliver"
	ApproveSupportSession Action = "approve_support_session"
)

var actions = map[Action]struct{}{
	View: {}, ViewCost: {}, ChangeCustomerPolicy: {}, ChangeSellingPolicy: {}, RequestApproval: {},
	Approve: {}, Issue: {}, Download: {}, Deliver: {}, ApproveSupportSession: {},
}

func fieldAction(action Action) bool {
	return action == View || action == ViewCost || action == Download
}

func humanOnly(action Action) bool { return action == Approve || action == ApproveSupportSession }

type FieldClass string

const (
	General FieldClass = "general"
	Cost    FieldClass = "cost"
)

type Catalog struct {
	Kinds  []string
	Fields map[string]FieldClass
}

type SubjectKind string

const (
	Human   SubjectKind = "human"
	Service SubjectKind = "service"
)

type State string

const (
	Active    State = "active"
	Suspended State = "suspended"
)

type Subject struct {
	ID    string
	Kind  SubjectKind
	State State
	Epoch uint64
}

type Client struct {
	ID    string
	State State
}

type Selector struct {
	set bool
	all bool
	id  string
}

func All() Selector { return Selector{set: true, all: true} }

func One(id string) Selector { return Selector{set: true, id: id} }

func (s Selector) valid() bool { return s.set && (s.all || validID(s.id)) }

func (s Selector) matches(id string) bool { return s.set && (s.all || (id != "" && id == s.id)) }

func (s Selector) within(parent Selector) bool {
	return s.set && parent.set && (parent.all || (!s.all && s.id == parent.id))
}

type Month int

func NewMonth(year, month int) (Month, error) {
	if year < 1900 || year > 9999 || month < 1 || month > 12 {
		return 0, ErrInvalid
	}
	return Month(year*12 + month - 1), nil
}

type Period struct {
	set  bool
	all  bool
	from Month
	to   Month
}

func AnyPeriod() Period { return Period{set: true, all: true} }

func Months(from, to Month) Period { return Period{set: true, from: from, to: to} }

func (p Period) valid() bool { return p.set && (p.all || p.from < p.to) }

func (p Period) matches(resource Resource) bool {
	return p.set && (p.all || (resource.HasMonth && p.from <= resource.Month && resource.Month < p.to))
}

func (p Period) within(parent Period) bool {
	return p.set && parent.set && (parent.all || (!p.all && parent.from <= p.from && p.to <= parent.to))
}

type Scope struct {
	WorkspaceID string
	Customer    Selector
	LegalEntity Selector
	Contract    Selector
}

func (s Scope) matches(resource Resource) bool {
	return s.WorkspaceID != "" && s.WorkspaceID == resource.WorkspaceID &&
		s.Customer.matches(resource.CustomerID) && s.LegalEntity.matches(resource.LegalEntityID) && s.Contract.matches(resource.ContractID)
}

func (s Scope) within(parent Scope) bool {
	return s.WorkspaceID == parent.WorkspaceID && s.Customer.within(parent.Customer) &&
		s.LegalEntity.within(parent.LegalEntity) && s.Contract.within(parent.Contract)
}

type Validity struct {
	NotBefore time.Time
	NotAfter  time.Time
}

func (v Validity) valid() bool {
	return !v.NotBefore.IsZero() && !v.NotAfter.IsZero() && v.NotBefore.Before(v.NotAfter)
}

func (v Validity) at(now time.Time) bool { return !now.Before(v.NotBefore) && now.Before(v.NotAfter) }

func (v Validity) within(parent Validity) bool {
	return !v.NotBefore.Before(parent.NotBefore) && !v.NotAfter.After(parent.NotAfter)
}

type Terms struct {
	Action Action
	Kind   string
	Scope  Scope
	Fields []string
	Period Period
}

type Grant struct {
	ID        string
	SubjectID string
	ClientID  string
	Terms     Terms
	Validity  Validity
}

type Delegation struct {
	ID                string
	DelegatorID       string
	DelegateID        string
	Terms             Terms
	Validity          Validity
	AllowRedelegation bool
	ParentID          string
}

type SupportSession struct {
	ID          string
	ActorID     string
	WorkspaceID string
	CustomerID  string
	Reason      string
	Kind        string
	Fields      []string
	Validity    Validity
	ApproverID  string
}

type Principal struct {
	Installation string
	SubjectID    string
	Epoch        uint64
	ClientID     string
	SessionID    string
}

type Resource struct {
	WorkspaceID   string
	CustomerID    string
	LegalEntityID string
	ContractID    string
	Month         Month
	HasMonth      bool
}

type Request struct {
	Action   Action
	Kind     string
	Resource Resource
	Fields   []string
}

type Code string

const (
	Allowed              Code = "allowed"
	PermissionDenied     Code = "permission_denied"
	SelfApprovalDenied   Code = "self_approval_denied"
	AuthorityUnavailable Code = "authority_unavailable"
)

type Reason string

const (
	ReasonAllowed          Reason = "allowed"
	ReasonUnavailable      Reason = "authority_unavailable"
	ReasonInvalidRequest   Reason = "invalid_request"
	ReasonPrincipal        Reason = "principal_not_current"
	ReasonStamp            Reason = "stamp_not_current"
	ReasonNoGrant          Reason = "no_matching_grant"
	ReasonClient           Reason = "client_not_allowed"
	ReasonSupportScope     Reason = "support_scope"
	ReasonSupportSession   Reason = "support_session_not_current"
	ReasonApprovalRequired Reason = "approval_check_required"
	ReasonNotHuman         Reason = "approver_not_human"
	ReasonAuthor           Reason = "author_unknown"
	ReasonSelfApproval     Reason = "author_is_approver"
)

type Stamp struct {
	instance   [16]byte
	generation uint64
	digest     [32]byte
	approval   bool
	issued     bool
}

type Decision struct {
	Allowed     bool
	Code        Code
	Reason      Reason
	ActorID     string
	ClientID    string
	SessionID   string
	Delegations []string
	Stamp       Stamp
}

const (
	maxIDBytes = 128
	maxChain   = 4
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
