package protected

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
	maxIDBytes    = 128
	maxWorkspaces = 100000
	maxChannels   = 1000000
	maxChanges    = 1000000
	minValidity   = time.Hour
	maxValidity   = 30 * 24 * time.Hour
	minLead       = time.Hour
	maxLead       = 366 * 24 * time.Hour
	minChannelAge = 24 * time.Hour
	maxChannelAge = 90 * 24 * time.Hour
	maxApprovals  = 3
	maxProposals  = 1000
	changeLabel   = "protected_change_v1"
	confirmLabel  = "protected_confirmation_v1"
)

const (
	KindHuman     = "human"
	KindAssistant = "assistant"
	KindSystem    = "system"

	FieldPaymentAccount    = "payment_account"
	FieldDocumentRecipient = "document_recipient"
	FieldTaxIdentifier     = "tax_identifier"
	FieldDocumentTemplate  = "document_template"

	ProvenanceStaff     = "staff_entry"
	ProvenanceDocument  = "customer_document"
	ProvenancePortal    = "customer_portal"
	ProvenanceAssistant = "assistant_generated"

	ChannelEmail    = "email"
	ChannelPhone    = "phone"
	ChannelPortal   = "portal"
	ChannelPostal   = "postal"
	ChannelInPerson = "in_person"

	OutcomeConfirmed = "confirmed"
	OutcomeDenied    = "denied"

	VerdictApprove = "approve"
	VerdictReject  = "reject"

	StateRequested = "requested"
	StateConfirmed = "confirmed"
	StateScheduled = "scheduled"
	StateRejected  = "rejected"
	StateWithdrawn = "withdrawn"
	StateLapsed    = "lapsed"

	ReasonAINotAllowed        = "ai_not_allowed"
	ReasonNotHuman            = "not_human"
	ReasonLLMGenerated        = "llm_generated"
	ReasonNoRules             = "no_rules"
	ReasonUnknownChannel      = "unknown_channel"
	ReasonChangePending       = "change_pending"
	ReasonEffectiveOutOfRange = "effective_out_of_range"
	ReasonOutOfOrder          = "out_of_order"
	ReasonNoChange            = "no_change"
	ReasonNotPending          = "not_pending"
	ReasonDigestChanged       = "digest_changed"
	ReasonSelfVerification    = "self_verification"
	ReasonSameChannel         = "same_channel"
	ReasonChannelRetired      = "channel_retired"
	ReasonChannelTooNew       = "channel_too_new"
	ReasonSelfChannel         = "self_channel"
	ReasonNotConfirmed        = "not_confirmed"
	ReasonSelfApproval        = "self_approval"
	ReasonAlreadyDecided      = "already_decided"
	ReasonLapsed              = "lapsed"
	ReasonProposalPending     = "proposal_pending"
	ReasonStaleProposal       = "stale_proposal"
	ReasonUndeclaredChannel   = "undeclared_channel"
	ReasonChannelNotAdmitted  = "channel_not_admitted"
	ReasonAlreadyEffective    = "already_effective"
	ReasonHasSuccessor        = "has_successor"
)

var (
	fields        = map[string]bool{FieldPaymentAccount: true, FieldDocumentRecipient: true, FieldTaxIdentifier: true, FieldDocumentTemplate: true}
	provenances   = map[string]bool{ProvenanceStaff: true, ProvenanceDocument: true, ProvenancePortal: true, ProvenanceAssistant: true}
	channelKinds  = map[string]bool{ChannelEmail: true, ChannelPhone: true, ChannelPortal: true, ChannelPostal: true}
	inbound       = map[string]bool{ChannelEmail: true, ChannelPhone: true, ChannelPortal: true, ChannelPostal: true, ChannelInPerson: true}
	outcomes      = map[string]bool{OutcomeConfirmed: true, OutcomeDenied: true}
	rejectReasons = map[string]bool{"incorrect_details": true, "missing_evidence": true, "suspected_fraud": true, "other": true}
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

type Limits struct {
	Workspaces int
	Channels   int
	Changes    int
}

func (l Limits) valid() bool {
	return l.Workspaces >= 1 && l.Workspaces <= maxWorkspaces && l.Channels >= 1 && l.Channels <= maxChannels && l.Changes >= 1 && l.Changes <= maxChanges
}

type Rules struct {
	Approvals  int
	Validity   time.Duration
	MaxLead    time.Duration
	ChannelAge time.Duration
}

func (r Rules) valid() bool {
	return r.Approvals >= 1 && r.Approvals <= maxApprovals &&
		r.Validity >= minValidity && r.Validity <= maxValidity &&
		r.MaxLead >= minLead && r.MaxLead <= maxLead &&
		r.ChannelAge >= minChannelAge && r.ChannelAge <= maxChannelAge
}

type RulesView struct {
	Rules     Rules
	Revision  int
	Proposer  string
	Proposed  time.Time
	Confirmer string
	Confirmed time.Time
	Discarder string
	Discarded time.Time
}

type RulesState struct {
	Active  *RulesView
	Pending *RulesView
	History []RulesView
}

type Ref struct {
	ID          string
	Fingerprint string
}

type Channel struct {
	ID       string
	Subject  string
	Kind     string
	Ref      Ref
	Evidence string
}

type ChannelView struct {
	Channel    Channel
	Registrar  string
	Registered time.Time
	Admitter   string
	Admitted   time.Time
	Admission  string
	RetiredBy  string
	Retired    time.Time
}

type Instruction struct {
	Kind     string
	Channel  string
	Sender   string
	Evidence string
}

type Change struct {
	ID          string
	Subject     string
	Field       string
	After       Ref
	Provenance  string
	Instruction Instruction
	Effective   time.Time
}

type Verification struct {
	Channel  string
	Evidence string
	Outcome  string
	Digest   string
}

type VerificationView struct {
	Verification Verification
	Verifier     string
	At           time.Time
	Digest       string
}

type Approval struct {
	Verdict string
	Reason  string
	Digest  string
}

type DecisionView struct {
	Approval Approval
	Approver string
	At       time.Time
}

type View struct {
	Change       Change
	State        string
	Before       Ref
	Requester    string
	Requested    time.Time
	Rules        Rules
	Revision     int
	Verification *VerificationView
	Decisions    []DecisionView
	Scheduled    time.Time
	EndedBy      string
	Ended        time.Time
	Digest       string
}

type Resolution struct {
	Change    string
	Ref       Ref
	Effective time.Time
}
