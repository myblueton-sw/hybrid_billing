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
	GrantView      = auth.GrantView
	DelegationView = auth.DelegationView
	Explanation    = auth.Explanation
	Delegation     = auth.Delegation
	SupportSession = auth.SupportSession
	Client         = auth.Client
	Action         = auth.Action
	Code           = auth.Code
	Reason         = auth.Reason
)

const (
	SubmitIssuanceDraft            = auth.SubmitIssuanceDraft
	ApproveIssuanceDraft           = auth.ApproveIssuanceDraft
	RegisterIssuanceChannel        = auth.RegisterIssuanceChannel
	AdmitIssuanceChannel           = auth.AdmitIssuanceChannel
	RequestIssuanceProtectedChange = auth.RequestIssuanceProtectedChange
	VerifyIssuanceProtectedChange  = auth.VerifyIssuanceProtectedChange
	ApproveIssuanceProtectedChange = auth.ApproveIssuanceProtectedChange
	RecordIssuanceTerm             = auth.RecordIssuanceTerm
	ApproveIssuanceTerm            = auth.ApproveIssuanceTerm
	BindIssuanceTerm               = auth.BindIssuanceTerm
	ApproveIssuanceTermBinding     = auth.ApproveIssuanceTermBinding
	ProposeERPResult               = auth.ProposeERPResult
	ConfirmERPResult               = auth.ConfirmERPResult
	RevokeERPResult                = auth.RevokeERPResult
	ApplyERPResult                 = auth.ApplyERPResult
	ProposeDocumentEvidence        = auth.ProposeDocumentEvidence
	ConfirmDocumentEvidence        = auth.ConfirmDocumentEvidence
	RevokeDocumentEvidence         = auth.RevokeDocumentEvidence
	PrepareERPRecord               = auth.PrepareERPRecord

	General = auth.General
	Cost    = auth.Cost

	View                   = auth.View
	ViewCost               = auth.ViewCost
	ChangeCustomerPolicy   = auth.ChangeCustomerPolicy
	ChangeSellingPolicy    = auth.ChangeSellingPolicy
	RequestApproval        = auth.RequestApproval
	ConfirmInput           = auth.ConfirmInput
	ProposePricingPolicy   = auth.ProposePricingPolicy
	ApprovePricingPolicy   = auth.ApprovePricingPolicy
	RevokePricingPolicy    = auth.RevokePricingPolicy
	UsePricingPolicy       = auth.UsePricingPolicy
	Approve                = auth.Approve
	Issue                  = auth.Issue
	Download               = auth.Download
	Deliver                = auth.Deliver
	ApproveSupportSession  = auth.ApproveSupportSession
	AdministerInstallation = auth.AdministerInstallation
	TrustPropose           = auth.TrustPropose
	TrustValidate          = auth.TrustValidate
	TrustApprove           = auth.TrustApprove
	TrustActivate          = auth.TrustActivate
	RevokeAccessInBulk     = auth.RevokeAccessInBulk
	AdministerAPIClients   = auth.AdministerAPIClients
	AdministerModules      = auth.AdministerModules
	ApproveModules         = auth.ApproveModules
	StopModules            = auth.StopModules
	AdministerChannels     = auth.AdministerChannels
	ApproveChannels        = auth.ApproveChannels
	StopChannels           = auth.StopChannels
	ReprocessEvents        = auth.ReprocessEvents
	InstallationKind       = auth.InstallationKind

	Active    = auth.Active
	Suspended = auth.Suspended

	Allowed              = auth.Allowed
	PermissionDenied     = auth.PermissionDenied
	SelfApprovalDenied   = auth.SelfApprovalDenied
	AuthorityUnavailable = auth.AuthorityUnavailable

	ReasonAllowed           = auth.ReasonAllowed
	ReasonUnavailable       = auth.ReasonUnavailable
	ReasonInvalidRequest    = auth.ReasonInvalidRequest
	ReasonPrincipal         = auth.ReasonPrincipal
	ReasonStamp             = auth.ReasonStamp
	ReasonNoGrant           = auth.ReasonNoGrant
	ReasonClient            = auth.ReasonClient
	ReasonSupportScope      = auth.ReasonSupportScope
	ReasonSupportSession    = auth.ReasonSupportSession
	ReasonApprovalRequired  = auth.ReasonApprovalRequired
	ReasonNotHuman          = auth.ReasonNotHuman
	ReasonAuthor            = auth.ReasonAuthor
	ReasonSelfApproval      = auth.ReasonSelfApproval
	ReasonIdentity          = auth.ReasonIdentity
	ReasonWorkspace         = auth.ReasonWorkspace
	ReasonInstallationScope = auth.ReasonInstallationScope
	ReasonNotAuthor         = auth.ReasonNotAuthor
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

type Answer = identity.Answer

type Admission string

const (
	Admitted         Admission = "admitted"
	NotCurrent       Admission = "identity_not_current"
	AdmitUnavailable Admission = "authority_unavailable"
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

func InstallationAction(action Action) bool { return auth.InstallationAction(action) }

func IntegrationRights() []Action { return auth.IntegrationRights() }
