package publication

import (
	"errors"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

var (
	ErrInvalid  = errors.New("a request or an input is incomplete or malformed")
	ErrNotFound = errors.New("what the request names is not registered")
	ErrConflict = errors.New("another fact is already recorded under this identifier")
	ErrRejected = errors.New("a recorded state or rule refuses this request")
	ErrLimit    = errors.New("a configured limit or bound is exceeded")
)

const (
	StateStaging   = "staging"
	StateSealed    = "sealed"
	StatePublished = "published"
	StateRetained  = "retained"

	ReasonUnpublishedOpen = "unpublished_open"
	ReasonGap             = "gap"
	ReasonCountMismatch   = "count_mismatch"
	ReasonDigestMismatch  = "digest_mismatch"
	ReasonDuplicateKey    = "duplicate_key"
	ReasonOverflow        = "overflow"
	ReasonNotSealed       = "not_sealed"
	ReasonRetained        = "retained"
	ReasonStale           = "stale"
	ReasonPublished       = "published"
	ReasonSealed          = "sealed"
)

type Refusal struct {
	Reason string
}

func (e Refusal) Error() string { return ErrRejected.Error() + ": " + e.Reason }

func (e Refusal) Unwrap() error { return ErrRejected }

type Limits struct {
	ChunkRows        int
	VersionRows      int
	StagedRows       int
	TenantStagedRows int
	HeldRows         int
	TenantHeldRows   int
	Groups           int
	Retained         int
	Page             int
	TenantDatasets   int
	Datasets         int
	StagingTTL       time.Duration
}

type Config struct {
	Profile decimal.Profile
	Limits  Limits
}

type Spec struct {
	Tenant  string
	Dataset string
	Version string
	Rows    int
	Digest  string
}

type Row struct {
	Key      string
	Group    string
	Currency string
	Amount   string
}

type Chunk struct {
	Tenant  string
	Dataset string
	Version string
	Index   int
	Rows    []Row
}

type TotalKey struct {
	Group    string
	Currency string
}

type Total struct {
	Key    TotalKey
	Amount string
}

type View struct {
	Tenant       string
	Dataset      string
	Version      string
	State        string
	Sequence     uint64
	Rows         int
	Digest       string
	NextIndex    int
	ReceivedRows int
	Begun        time.Time
	Deadline     time.Time
	Sealed       time.Time
	Published    time.Time
	Repeat       bool
}

type RowPage struct {
	Rows []Row
	Next string
}

type TotalPage struct {
	Totals []Total
	Next   *TotalKey
}
