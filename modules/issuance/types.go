package issuance

import (
	"errors"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/protected"
	"github.com/myblueton-sw/hybrid_billing/modules/receivables"
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
	maxSubject    = 100
	maxNumbers    = 10000000
	maxAttempts   = 100
	maxDownloads  = 100000
	minLease      = time.Minute
	maxLease      = time.Hour
	maxTextBytes  = 4000
	maxWorkspaces = 100000
	maxKinds      = 10000
	maxDrafts     = 1000000
	maxDocuments  = 1000000
	maxLines      = 200
	maxDetailRows = 10000000
	maxDigits     = 30
	maxScale      = 6
	maxNetDays    = 366
	maxWidth      = 12
	maxFileBytes  = 20 << 20
	draftLabel    = "issuance_draft_v1"
	snapshotLabel = "issuance_snapshot_v1"

	KindHuman     = "human"
	KindAssistant = "assistant"
	KindSystem    = "system"

	ClassSelf     = "self_issued"
	ClassInternal = "internal_settlement"
	ClassERP      = "erp_issued"

	LanguageEnglish = "en"
	LanguageKorean  = "ko"
	LanguageArabic  = "ar"

	StateProposed = "proposed"
	StateActive   = "active"

	NumberReserved  = "reserved"
	NumberIssued    = "issued"
	NumberCancelled = "cancelled"

	CancelAbandoned = "draft_abandoned"
	CancelOperator  = "operator_error"
	CancelOther     = "other"

	FileNone      = "none"
	FileRendering = "rendering"
	FileRendered  = "rendered"

	OutcomeAccepted = "accepted"
	OutcomeVerified = "verified"

	DueApplicable    = "applicable"
	DueNotApplicable = "not_applicable"

	SubjectEntity   = "entity:"
	SubjectCustomer = "customer:"
	SubjectKind     = "kind:"

	ReasonAINotAllowed        = "ai_not_allowed"
	ReasonNotHuman            = "not_human"
	ReasonSelfApproval        = "self_approval"
	ReasonSelfReview          = "self_review"
	ReasonNotApproved         = "not_approved"
	ReasonKindPending         = "kind_pending"
	ReasonTaxUnverified       = "tax_unverified"
	ReasonNotStatutory        = "not_statutory"
	ReasonERPNumbered         = "erp_numbered"
	ReasonNotERP              = "not_erp"
	ReasonNotSelfIssued       = "not_self_issued"
	ReasonSeriesExists        = "series_exists"
	ReasonSeriesPending       = "series_pending"
	ReasonNoSeries            = "no_series"
	ReasonPrefixInUse         = "prefix_in_use"
	ReasonAlreadyIssued       = "already_issued"
	ReasonAlreadyReserved     = "already_reserved"
	ReasonNotReserved         = "not_reserved"
	ReasonDigestChanged       = "digest_changed"
	ReasonTotalMismatch       = "total_mismatch"
	ReasonDetailMismatch      = "detail_mismatch"
	ReasonLanguage            = "language_unsupported"
	ReasonCreditMismatch      = "credit_mismatch"
	ReasonExceedsOriginal     = "exceeds_original"
	ReasonOriginalNotIssued   = "original_not_issued"
	ReasonAccountUnresolved   = "payment_account_unresolved"
	ReasonRecipientUnresolved = "document_recipient_unresolved"
	ReasonTaxIDUnresolved     = "tax_identifier_unresolved"
	ReasonTemplateUnresolved  = "document_template_unresolved"
	ReasonRenderInProgress    = "render_in_progress"
	ReasonRenderFailed        = "render_failed"
	ReasonUnsafeOutput        = "unsafe_output"
	ReasonMalformedOutput     = "malformed_output"
	ReasonDigestMismatch      = "digest_mismatch"
	ReasonNotRendered         = "not_rendered"
	ReasonExternalInUse       = "external_in_use"
	ReasonReservationsOff     = "reservations_disabled"
	ReasonReferenceChanged    = "reference_changed"
	ReasonDetailUnverified    = "detail_unverified"
	ReasonModeUnsupported     = "mode_unsupported"
	ReasonTermMissing         = "term_missing"
	ReasonTermPending         = "term_pending"
	ReasonTermConflict        = "term_conflict"
	ReasonTermUnavailable     = "term_unavailable"
	ReasonAttemptSuperseded   = "attempt_superseded"
	ReasonBeforeApproval      = "before_approval"
	ReasonFutureInstant       = "future_instant"
	ReasonZoneUnavailable     = "zone_unavailable"
	ReasonCoverageMismatch    = "coverage_mismatch"
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

type Fields interface {
	Resolve(workspace, subject, field string, at time.Time) (protected.Resolution, error)
}

type Terms interface {
	Cover(workspace, contract string, at time.Time) (receivables.Coverage, error)
}

type DetailFacts struct {
	Digest   string
	Rows     int64
	Total    string
	Currency string
	Scale    int
}

type Details interface {
	Detail(workspace, artifact string) (DetailFacts, error)
}

type Renderer interface {
	Render(Snapshot) ([]byte, error)
}

type Config struct {
	Fields   Fields
	Terms    Terms
	Details  Details
	Renderer Renderer
	Zones    func(string) (*time.Location, error)
}

type Limits struct {
	Workspaces int
	Kinds      int
	Drafts     int
	Documents  int
	Numbers    int
	Attempts   int
	Downloads  int
	Lease      time.Duration
}

type Kind struct {
	ID         string
	Class      string
	Region     string
	Tax        string
	Statutory  bool
	Credit     bool
	Sequential bool
	Zone       string
	Languages  []string
}

type KindView struct {
	Kind        Kind
	State       string
	Definer     string
	Approver    string
	TaxReviewer string
	TaxEvidence string
	At          time.Time
}

type Series struct {
	ID     string
	Kind   string
	Prefix string
	Width  int
	First  int64
}

type SeriesView struct {
	Series   Series
	State    string
	Proposer string
	Approver string
	Next     int64
	At       time.Time
}

type NumberView struct {
	Number string
	Value  int64
	Draft  string
	State  string
	Actor  string
	At     time.Time
	Reason string
	Taker  string
	Taken  time.Time
}

type Line struct {
	Text   string
	Amount string
	Tax    bool
}

type Detail struct {
	Artifact string
	Digest   string
	Rows     int64
	Total    string
}

type Draft struct {
	ID       string
	Kind     string
	Customer string
	Entity   string
	Contract string
	Currency string
	Scale    int
	Language string
	Lines    []Line
	Total    string
	Detail   *Detail
	Original string
}

type DraftView struct {
	Draft      Draft
	Revision   int
	Digest     string
	Submitter  string
	Approver   string
	References []Reference
	Coverage   *receivables.Coverage
	Number     string
	At         time.Time
}

type External struct {
	Number   string
	IssuedAt time.Time
}

type Reference struct {
	Field     string
	Subject   string
	Change    string
	Ref       protected.Ref
	Effective time.Time
}

type Due struct {
	Status       string
	Zone         string
	IssueDate    string
	Binding      string
	Term         string
	Revision     int
	NetDays      int
	Mode         string
	OriginalDue  string
	EffectiveDue string
}

type Snapshot struct {
	Workspace   string
	Number      string
	External    bool
	Draft       Draft
	Revision    int
	DraftDigest string
	Submitter   string
	Approver    string
	Issuer      string
	IssuerKind  string
	IssuedAt    time.Time
	Class       string
	Region      string
	Tax         string
	Statutory   bool
	TaxEvidence string
	TaxReviewer string
	References  []Reference
	Due         Due
	Digest      string
}

type Attempt struct {
	Number   int
	Actor    string
	Started  time.Time
	Finished time.Time
	Outcome  string
	Digest   string
}

type DocumentView struct {
	Snapshot  Snapshot
	File      string
	Digest    string
	Size      int
	Attempts  []Attempt
	Downloads []Download
}

type Download struct {
	Actor string
	Kind  string
	At    time.Time
}
