package termsauthority

import "github.com/myblueton-sw/hybrid_billing/modules/internal/grantauthority"

var (
	ErrInvalid  = grantauthority.ErrInvalid
	ErrDenied   = grantauthority.ErrDenied
	ErrConflict = grantauthority.ErrConflict
	ErrNotFound = grantauthority.ErrNotFound
	ErrLimit    = grantauthority.ErrLimit
	ErrUnknown  = grantauthority.ErrUnknown
)

const (
	Console        = "console_ceiling"
	Human          = "human_grant"
	Pending        = "pending"
	Applying       = "applying"
	Unknown        = "decision_unknown"
	Applied        = "applied"
	Rejected       = "rejected"
	Withdrawn      = "withdrawn"
	Expired        = "expired"
	Superseded     = "superseded"
	ActiveGrant    = "active"
	RevokedGrant   = "revoked"
	ExpiredGrant   = "expired"
	RecordTerm     = "record"
	ApproveTerm    = "approve"
	BindTerm       = "bind"
	ApproveBinding = "approve_binding"
	Inspect        = "inspect"
	EvidenceField  = "issuance_terms"
	ResourceKind   = "synthetic_issuance_terms"
)

type Limits = grantauthority.Limits
type Spec = grantauthority.Spec
type Proposal = grantauthority.Proposal
type Intent = grantauthority.Intent
type Effect = grantauthority.Effect
type Authority = grantauthority.Authority
type Targets = grantauthority.Targets
type Effects = grantauthority.Effects
type Decision = grantauthority.Decision
type Record = grantauthority.Record
type Registry = grantauthority.Registry
