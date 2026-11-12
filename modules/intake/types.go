package intake

import (
	"errors"
	"time"
)

var (
	ErrInvalid  = errors.New("request is incomplete or malformed")
	ErrNotFound = errors.New("artifact, subject or manifest does not exist in this workspace")
	ErrConflict = errors.New("request conflicts with what is stored")
	ErrRejected = errors.New("the state of the artifact or subject does not allow this step")
	ErrLimit    = errors.New("a configured limit is exceeded")
)

const maxIDBytes = 128

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

func validDigest(sum string) bool {
	if len(sum) < 8 || len(sum) > 128 {
		return false
	}
	for index := 0; index < len(sum); index++ {
		c := sum[index]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func validCurrency(code string) bool {
	return len(code) == 3 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= 'Z' && code[2] >= 'A' && code[2] <= 'Z'
}

type Subject struct {
	WorkspaceID string
	Provider    string
	Scope       string
	Period      string
}

func (s Subject) valid() bool {
	return validID(s.WorkspaceID) && validID(s.Provider) && validID(s.Scope) && validID(s.Period)
}

type KeyKind string

const (
	EventKeys   KeyKind = "event"
	LocatorKeys KeyKind = "locator"
)

type Role string

const (
	Original Role = "original"
	Derived  Role = "derived"
)

type Holds string

const (
	Rows               Holds = "rows"
	Empty              Holds = "empty"
	AwaitingExtraction Holds = "awaiting_extraction"
)

type Check string

const (
	Passed Check = "passed"
	Failed Check = "failed"
	NotRun Check = "not_run"
)

func (c Check) known() bool { return c == Passed || c == Failed || c == NotRun }

type TotalsOrigin string

const (
	SourceStated   TotalsOrigin = "source_stated"
	CallerComputed TotalsOrigin = "caller_computed"
)

type Spec struct {
	ID         string
	Subject    Subject
	Document   string
	Kind       KeyKind
	Revision   uint64
	Bytes      string
	Role       Role
	Holds      Holds
	Original   string
	Method     string
	Chunks     int
	Rows       int
	Sums       map[string]string
	Totals     TotalsOrigin
	Size       int64
	Expanded   int64
	Location   string
	Collection string
	Scan       Check
	Syntax     Check
	Bodies     Check
}

func (s Spec) carriesRows() bool { return s.Role == Derived || s.Holds == Rows }

type Record struct {
	Key      string
	Digest   string
	Amount   string
	Currency string
}

type State string

const (
	Staging      State = "staging"
	Sealed       State = "sealed"
	Rejected     State = "rejected"
	Inconsistent State = "inconsistent"
	Abandoned    State = "abandoned"
	Retired      State = "retired"
)

const (
	TooLarge      = "too_large"
	RatioExceeded = "ratio_exceeded"
	ScanFailed    = "scan_failed"
	ScanNotRun    = "scan_not_run"
	SyntaxFailed  = "syntax_failed"
	SyntaxNotRun  = "syntax_not_run"
	BodiesFound   = "bodies_found"
	BodiesNotRun  = "bodies_not_run"
	GapChunks     = "chunks_missing"
	GapRows       = "rows_differ"
	GapCurrency   = "currency_not_declared"
	GapNoRows     = "declared_currency_without_rows"
	GapSum        = "sum_differs"
	GapOriginal   = "original_not_sealed"
	GapVerified   = "not_verified"
	Unreadable    = "unreadable"
	InConflict    = "document_conflict"
	KeyConflict   = "key_conflict"
	SumOutside    = "sum_outside_profile"
)

type AbandonReason string

const (
	TransferFailed AbandonReason = "transfer_failed"
	Superseded     AbandonReason = "superseded"
	Unsealable     AbandonReason = "unsealable"
	AbandonedBy    AbandonReason = "operator"
)

func (r AbandonReason) known() bool {
	return r == TransferFailed || r == Superseded || r == Unsealable || r == AbandonedBy
}

type RetireReason string

const (
	WrongSource           RetireReason = "wrong_source"
	Reissued              RetireReason = "reissued"
	VerificationWithdrawn RetireReason = "verification_withdrawn"
	RetiredBy             RetireReason = "operator"
)

func (r RetireReason) known() bool {
	return r == WrongSource || r == Reissued || r == VerificationWithdrawn || r == RetiredBy
}

type Gap struct {
	What     string
	Currency string
}

type Verification struct {
	Digest   string
	Verifier string
	Evidence string
	At       time.Time
}

type ArtifactView struct {
	Spec          Spec
	State         State
	Reasons       []string
	AbandonReason AbandonReason
	RetireReason  RetireReason
	Rows          int
	Sums          map[string]string
	Digest        string
	Sequence      uint64
	Verification  *Verification
	Contradiction int
	Registered    time.Time
	Changed       time.Time
}

type StagedView struct {
	Chunks   int
	Rows     int
	Sums     map[string]string
	Digest   string
	Complete bool
}

type ManifestRecord struct {
	Kind     KeyKind
	Document string
	Key      string
	Amount   string
	Currency string
	Digest   string
	Artifact string
}

type Listed struct {
	Total int
	Items []string
}

type Blocker struct {
	What     string
	Document string
	Key      string
	HeldBy   string
	Currency string
}

type ManifestView struct {
	Subject     Subject
	Number      uint64
	Records     int
	Sums        map[string]string
	Content     string
	Attribution string
	Staging     int
	Behind      Listed
	SameBytes   Listed
	At          time.Time
}

type StatusView struct {
	Latest       uint64
	Differs      string
	Blockers     []Blocker
	BlockerTotal int
	Staging      int
	Behind       Listed
	SameBytes    Listed
}

const (
	Same               = ""
	DiffersContent     = "content"
	DiffersAttribution = "attribution"
	Blocked            = "blocked"
)

type Published struct {
	Number       uint64
	Written      bool
	Blockers     []Blocker
	BlockerTotal int
}

type Limits struct {
	MaxSize      int64
	MaxRatio     int64
	Chunks       int
	ChunkRecords int
	Artifacts    int
	Records      int
	Manifests    int
	Page         int
	Listed       int
}

func (l Limits) valid() bool {
	return l.MaxSize > 0 && l.MaxRatio > 0 && l.Chunks > 0 && l.ChunkRecords > 0 && l.Artifacts > 0 && l.Records > 0 && l.Manifests > 0 && l.Page > 0 && l.Listed > 0
}
