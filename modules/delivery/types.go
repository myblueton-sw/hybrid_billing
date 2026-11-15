package delivery

import (
	"errors"
	"io"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/disclosure"
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
	maxChannels   = 100000
	maxDeliveries = 1000000
	maxQueue      = 1000000
	maxAttempts   = 100
	minLease      = time.Minute
	maxLease      = 24 * time.Hour

	KindHuman     = "human"
	KindAssistant = "assistant"
	KindSystem    = "system"

	ChannelERP    = "erp_attachment"
	ChannelMail   = "mail"
	ChannelPortal = "portal"

	EvidenceChannelTest    = "channel_test"
	EvidencePlatformExport = "platform_export"

	StateWaiting   = "waiting"
	StateInFlight  = "in_flight"
	StateAccepted  = "accepted"
	StateConfirmed = "confirmed"
	StateStopped   = "stopped"
	StateExhausted = "exhausted"
	StateCancelled = "cancelled"

	ResultAccepted = "accepted"
	ResultRejected = "rejected"
	ResultUnknown  = "unknown"

	ReasonAINotAllowed       = "ai_not_allowed"
	ReasonNotHuman           = "not_human"
	ReasonRecipientActor     = "recipient_actor"
	ReasonRandomUnavailable  = "random_unavailable"
	ReasonExportEvidence     = "export_evidence_insufficient"
	ReasonSelfApproval       = "self_approval"
	ReasonStaleDiagnosis     = "stale_diagnosis"
	ReasonCapabilityUnproven = "capability_unproven"
	ReasonAddsAttachments    = "adds_attachments"
	ReasonERPGateUnbound     = "erp_gate_unbound"
	ReasonPackageNotLive     = "package_not_live"
	ReasonRecipientMismatch  = "recipient_mismatch"
	ReasonNotWaiting         = "not_waiting"
	ReasonNotApproved        = "not_approved"
	ReasonAttemptRunning     = "attempt_running"
	ReasonFinal              = "final"
	ReasonNotAccepted        = "not_accepted"
	ReasonNotResendable      = "not_resendable"
	ReasonAlreadyApproved    = "already_approved"
	ReasonUnavailable        = "packages_unavailable"
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

type Packages interface {
	Validity(workspace, pkg string) (disclosure.Validity, error)
	Recipient(id string) bool
}

type Config struct {
	Packages Packages
	Random   io.Reader
}

type Limits struct {
	Workspaces int
	Channels   int
	Deliveries int
	Queue      int
	Attempts   int
	Lease      time.Duration
}

type Channel struct {
	ID        string
	Kind      string
	Recipient string
	Target    string
}

type Evidence struct {
	Kind      string
	Reference string
}

type Diagnosis struct {
	FieldControl      bool
	AttachmentControl bool
	AddsAttachments   bool
	Evidence          Evidence
}

type DiagnosisView struct {
	Diagnosis Diagnosis
	Revision  int
	Author    string
	At        time.Time
}

type ChannelView struct {
	Channel    Channel
	Registrar  string
	Registered time.Time
	Diagnoses  []DiagnosisView
	Open       bool
	Opener     string
	Opened     int
	Closer     string
}

type Delivery struct {
	ID      string
	Package string
	Channel string
}

type Attempt struct {
	Number            int
	Actor             string
	Kind              string
	Started           time.Time
	Result            string
	Reference         string
	Reporter          string
	Reported          time.Time
	Late              bool
	Superseded        bool
	PossibleDuplicate bool
}

type View struct {
	Delivery          Delivery
	Recipient         string
	FileDigest        string
	Policy            string
	Revision          int
	Grant             int
	State             string
	Reason            string
	Requester         string
	Requested         time.Time
	Approver          string
	Resends           string
	ResendReason      string
	Closer            string
	Receipt           string
	Confirmer         string
	Attempts          []Attempt
	PossibleDuplicate bool
	Duplicate         bool
}

type Start struct {
	Delivery          string
	Token             string
	Package           string
	Recipient         string
	FileDigest        string
	Channel           string
	Number            int
	PossibleDuplicate bool
}

type Outcome struct {
	Result    string
	Reference string
}
