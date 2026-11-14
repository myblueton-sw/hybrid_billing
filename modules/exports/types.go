package exports

import (
	"errors"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
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
	maxTextBytes   = 512
	maxScale       = 8
	maxIntDigits   = 18
	maxWorkspaces  = 100000
	maxArtifacts   = 1000000
	maxLines       = 200000
	maxSettlement  = 10000
	maxArtifactID  = 100
	maxNumberBytes = 64
	sheetRows      = 1048575
	dataLabel      = "export_data_v1"
	inputsLabel    = "export_inputs_v1"
	machineMarker  = "#machine-values"
	KindSettlement = "settlement"
	KindCustomer   = "customer"
	KindHuman      = "human"
	KindAssistant  = "assistant"
	KindSystem     = "system"

	ReasonAINotAllowed       = "ai_not_allowed"
	ReasonNotHuman           = "not_human"
	ReasonConfirmed          = "confirmed"
	ReasonNotConfirmed       = "not_confirmed"
	ReasonNotSettlement      = "not_settlement"
	ReasonUnknownRow         = "unknown_row"
	ReasonSubmissionMismatch = "submission_mismatch"
	ReasonAmountChanged      = "amount_changed"
	ReasonSelfApproval       = "self_approval"
	ReasonNotUTF8            = "not_utf8"
	ReasonWrongFormat        = "wrong_format"
	ReasonMixedResults       = "mixed_results"
	maxResultBytes           = 32 << 20
)

var (
	resultStates = map[string]bool{"": true, "draft": true, "posted": true, "rejected": true}
	forward      = map[string]map[string]bool{"": {"draft": true, "posted": true, "rejected": true}, "draft": {"posted": true, "rejected": true}, "posted": {"rejected": true}}
)

type Refusal struct {
	Reason string
	Row    string
}

func (e Refusal) Error() string {
	if e.Row != "" {
		return ErrRejected.Error() + ": " + e.Reason + " at " + e.Row
	}
	return ErrRejected.Error() + ": " + e.Reason
}

func (e Refusal) Unwrap() error { return ErrRejected }

type Actor struct {
	ID   string
	Kind string
}

type Config struct {
	Generator string
	Schema    string
	Numbers   decimal.Profile
}

type Limits struct {
	Workspaces int
	Artifacts  int
	Lines      int
}

type Line struct {
	Row      string
	Source   string
	Product  string
	TaxCode  string
	Quantity string
	Net      string
	Tax      string
}

type Spec struct {
	Connection string
	Submission string
	Approval   string
	Customer   string
	Period     string
	Currency   string
	Scale      int
	Lines      []Line
}

type Request struct {
	ID       string
	Kind     string
	TimeZone string
	Spec     Spec
}

type Contract struct {
	Encoding         string
	Separator        string
	LineEnd          string
	Decimal          string
	Sign             string
	Period           string
	TimeZone         string
	Scale            int
	SortKey          string
	ViewingBOM       bool
	ViewingPrecision string
	Profile          string
	ProfileRevision  string
}

type File struct {
	Name   string
	Media  string
	Bytes  []byte
	Digest string
}

type Volatile struct {
	Actor string
	At    time.Time
}

type Artifact struct {
	Request   Request
	Files     []File
	Contract  Contract
	Generator string
	Schema    string
	Data      string
	Inputs    string
	Volatile  Volatile
	Confirmed *Volatile
}

type Comparison struct {
	DataEqual        bool
	GeneratorChanged bool
	SchemaChanged    bool
	Data             string
}

type Result struct {
	Row       string
	Reference string
	State     string
}

type Handoff struct {
	Observation string
	Connection  string
	Submission  string
	Reference   string
	State       string
	Revision    int
}

type Registration struct {
	Rows    []Result
	Handoff *Handoff
}
