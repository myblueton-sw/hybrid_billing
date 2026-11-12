package gate

import (
	"github.com/myblueton-sw/hybrid_billing/modules/auth"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

type (
	Catalog        = auth.Catalog
	FieldClass     = auth.FieldClass
	Request        = auth.Request
	Resource       = auth.Resource
	Selector       = auth.Selector
	Month          = auth.Month
	Period         = auth.Period
	Scope          = auth.Scope
	Validity       = auth.Validity
	Terms          = auth.Terms
	Grant          = auth.Grant
	Delegation     = auth.Delegation
	SupportSession = auth.SupportSession
	Client         = auth.Client
	Action         = auth.Action
	Code           = auth.Code
	Reason         = auth.Reason
)

const (
	General = auth.General
	Cost    = auth.Cost

	View                  = auth.View
	ViewCost              = auth.ViewCost
	ChangeCustomerPolicy  = auth.ChangeCustomerPolicy
	ChangeSellingPolicy   = auth.ChangeSellingPolicy
	RequestApproval       = auth.RequestApproval
	Approve               = auth.Approve
	Issue                 = auth.Issue
	Download              = auth.Download
	Deliver               = auth.Deliver
	ApproveSupportSession = auth.ApproveSupportSession

	Active    = auth.Active
	Suspended = auth.Suspended

	Allowed              = auth.Allowed
	PermissionDenied     = auth.PermissionDenied
	SelfApprovalDenied   = auth.SelfApprovalDenied
	AuthorityUnavailable = auth.AuthorityUnavailable

	ReasonAllowed          = auth.ReasonAllowed
	ReasonUnavailable      = auth.ReasonUnavailable
	ReasonInvalidRequest   = auth.ReasonInvalidRequest
	ReasonPrincipal        = auth.ReasonPrincipal
	ReasonStamp            = auth.ReasonStamp
	ReasonNoGrant          = auth.ReasonNoGrant
	ReasonClient           = auth.ReasonClient
	ReasonSupportScope     = auth.ReasonSupportScope
	ReasonSupportSession   = auth.ReasonSupportSession
	ReasonApprovalRequired = auth.ReasonApprovalRequired
	ReasonNotHuman         = auth.ReasonNotHuman
	ReasonAuthor           = auth.ReasonAuthor
	ReasonSelfApproval     = auth.ReasonSelfApproval
	ReasonIdentity         = auth.ReasonIdentity
	ReasonWorkspace        = auth.ReasonWorkspace
)

var (
	ErrInvalid     = auth.ErrInvalid
	ErrConflict    = auth.ErrConflict
	ErrNotFound    = auth.ErrNotFound
	ErrNotBacked   = auth.ErrNotBacked
	ErrDenied      = auth.ErrDenied
	ErrUnavailable = auth.ErrUnavailable
	ErrChanged     = auth.ErrChanged

	All       = auth.All
	One       = auth.One
	NewMonth  = auth.NewMonth
	AnyPeriod = auth.AnyPeriod
	Months    = auth.Months
)

type Stamp struct {
	issued     bool
	permission auth.Stamp
	answer     identity.Answer
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
