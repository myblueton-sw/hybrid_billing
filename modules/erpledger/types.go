package erpledger

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
	maxIDBytes        = 128
	maxNumberBytes    = 64
	maxScale          = 8
	minInterval       = time.Minute
	maxInterval       = 24 * time.Hour
	maxWorkspaces     = 100000
	maxConnections    = 10000
	maxMappings       = 1000000
	maxAnchors        = 1000000
	maxSubmissions    = 1000000
	maxLines          = 10000
	maxDocuments      = 1000000
	maxRows           = 10000
	maxObservations   = 10000000
	maxCommands       = 1000
	maxLinks          = 10000
	maxSurveys        = 1000000
	postedWatch       = 30 * 24 * time.Hour
	ResolveAccept     = "accept"
	ResolveKeep       = "keep"
	ResolveSuperseded = "superseded"
	submissionLabel   = "erp_submission_v1"
	TransportAPI      = "api"
	TransportFile     = "file"
	TransportDB       = "database"
	PurposeInvoice    = "customer_invoice"
	PurposePayable    = "payable"
	PathDirect        = "direct"
	PathOrderBased    = "order_based"
	SourceWebhook     = "webhook"
	SourcePoll        = "poll"
	SourceResultFile  = "result_file"
)

const (
	KindHuman     = "human"
	KindAssistant = "assistant"
	KindSystem    = "system"

	CommandConnect   = "connect"
	CommandMapping   = "validate_mapping"
	CommandPrepare   = "prepare"
	CommandSubmit    = "submit_draft"
	CommandReadDraft = "read_draft"
	CommandPost      = "post"
	CommandStatus    = "get_status"
	CommandCorrect   = "correct"
	CommandBalance   = "get_balance"

	AnchorSalesOrder = "sales_order"
	AnchorContract   = "contract"
	AnchorPrepaid    = "prepaid"
	AnchorPayable    = "ap_document"

	SurveyComplete = "complete"
	SurveyUnknown  = "unknown"

	StatePrepared   = "prepared"
	StateSent       = "sent"
	StateDraft      = "draft"
	StatePosted     = "posted"
	StateAbsent     = "absent"
	StateSuperseded = "superseded"
	StateWithdrawn  = "withdrawn"
	StateRejected   = "rejected"
	StateUnknown    = "unknown"

	ReasonAINotAllowed       = "ai_not_allowed"
	ReasonNotHuman           = "not_human"
	ReasonDirectDatabase     = "direct_database_forbidden"
	ReasonUnsupported        = "unsupported"
	ReasonSelfApproval       = "self_approval"
	ReasonUnknownConnection  = "unknown_connection"
	ReasonNotSurveyed        = "not_surveyed"
	ReasonLedgerUnknown      = "ledger_unknown"
	ReasonMappingMissing     = "mapping_missing"
	ReasonUnknownAnchor      = "unknown_anchor"
	ReasonAlreadyReflected   = "already_reflected"
	ReasonAlreadySubmitted   = "already_submitted"
	ReasonEmpty              = "empty"
	ReasonWrongState         = "wrong_state"
	ReasonOverAllocated      = "over_allocated"
	ReasonUnknownSubmission  = "unknown_submission"
	ReasonUnknownDocument    = "unknown_document"
	ReasonConnectionMismatch = "connection_mismatch"
	ReasonDiverged           = "diverged"
	ReasonTemplateUnverified = "template_unverified"
)

var (
	capabilities  = map[string]bool{CommandConnect: true, CommandMapping: true, CommandPrepare: true, CommandSubmit: true, CommandReadDraft: true, CommandPost: true, CommandStatus: true, CommandCorrect: true, CommandBalance: true}
	mappingKinds  = map[string]bool{"customer": true, "product": true, "tax": true}
	anchorKinds   = map[string]bool{AnchorSalesOrder: true, AnchorContract: true, AnchorPrepaid: true, AnchorPayable: true}
	outcomes      = map[string]bool{SurveyComplete: true, SurveyUnknown: true}
	sources       = map[string]bool{SourceWebhook: true, SourcePoll: true, SourceResultFile: true}
	observed      = map[string]bool{StateDraft: true, StatePosted: true, StateRejected: true, StateAbsent: true, StateUnknown: true}
	terminal      = map[string]bool{StatePosted: true, StateRejected: true, StateAbsent: true, StateSuperseded: true, StateWithdrawn: true}
	retired       = map[string]bool{StateRejected: true, StateAbsent: true, StateSuperseded: true, StateWithdrawn: true}
	checkOutcomes = map[string]bool{"timeout": true, "error": true}
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
	Workspaces   int
	Connections  int
	Mappings     int
	Anchors      int
	Submissions  int
	Lines        int
	Documents    int
	Rows         int
	Observations int
	Links        int
	Surveys      int
}

func (l Limits) valid() bool {
	pairs := [][2]int{{l.Workspaces, maxWorkspaces}, {l.Connections, maxConnections}, {l.Mappings, maxMappings}, {l.Anchors, maxAnchors},
		{l.Submissions, maxSubmissions}, {l.Lines, maxLines}, {l.Documents, maxDocuments}, {l.Rows, maxRows}, {l.Observations, maxObservations}, {l.Links, maxLinks}, {l.Surveys, maxSurveys}}
	for _, p := range pairs {
		if p[0] < 1 || p[0] > p[1] {
			return false
		}
	}
	return true
}

type Connection struct {
	ID           string
	Entity       string
	System       string
	Transport    string
	Capabilities []string
	Currency     string
	Scale        int
	Interval     time.Duration
	Owner        string
	Template     string
}

type ConnectionView struct {
	Connection Connection
	Author     string
	At         time.Time
	Verified   bool
	Evidence   string
	Verifier   string
}

type Mapping struct {
	Connection string
	Kind       string
	Platform   string
	External   string
}

type MappingView struct {
	Mapping  Mapping
	Revision int
	Author   string
	At       time.Time
}

type Anchor struct {
	ID         string
	Connection string
	Kind       string
	External   string
	Customer   string
	Sources    []string
}

type Survey struct {
	Connection string
	Customer   string
	Period     string
	Outcome    string
	Evidence   string
}

type Line struct {
	ID       string
	Source   string
	Product  string
	TaxCode  string
	Quantity string
	Net      string
	Tax      string
}

type Submission struct {
	Key        string
	Connection string
	Purpose    string
	Customer   string
	Period     string
	Path       string
	Anchor     string
	Lines      []Line
}

type CommandView struct {
	Name  string
	Actor string
	At    time.Time
}

type SubmissionView struct {
	Submission  Submission
	State       string
	Total       Amounts
	Mappings    []string
	Author      string
	Prepared    time.Time
	External    string
	Revision    int
	Posted      time.Time
	Diverged    bool
	LastState   string
	LastRef     string
	Resolution  *ResolutionView
	WithdrawnBy string
	Commands    []CommandView
	Watch       *Watch
	Digest      string
}

type Amounts struct {
	Net      string
	Tax      string
	Quantity string
}

type Watch struct {
	Last     time.Time
	Next     time.Time
	Revision int
	Retries  int
	Owner    string
}

type Observation struct {
	ID         string
	Connection string
	Key        string
	External   string
	Revision   int
	State      string
	Source     string
}

type ObservationView struct {
	Observation Observation
	Actor       string
	At          time.Time
	Outcome     string
}

type Row struct {
	ID       string
	Product  string
	Net      string
	Tax      string
	Quantity string
}

type Document struct {
	Connection string
	External   string
	Revision   int
	Rows       []Row
}

type Link struct {
	Connection string
	External   string
	Row        string
	Key        string
	Line       string
	Net        string
	Tax        string
	Quantity   string
}

type LinkView struct {
	Link   Link
	Stale  bool
	Author string
}

type DocumentView struct {
	Document Document
	Total    Amounts
	Links    []LinkView
	Author   string
	At       time.Time
}

type LineResult struct {
	Line    string
	Linked  Amounts
	Matched bool
}

type DocumentTotals struct {
	Connection string
	External   string
	Total      Amounts
	Attributed Amounts
}

type Reconciliation struct {
	Key       string
	Lines     []LineResult
	Documents []DocumentTotals
	Matched   bool
}

type Resolution struct {
	Outcome  string
	Evidence string
}

type ResolutionView struct {
	Resolution Resolution
	Actor      string
	At         time.Time
}
