package claims

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
	maxIDBytes       = 128
	maxNumberBytes   = 64
	maxLifetime      = 86400
	maxInstances     = 1000
	applicationLabel = "billing_application_v1"
	eligibilityLabel = "billing_eligibility_v1"
	reversalLabel    = "billing_reversal_v1"
)

const (
	PathDirect = "direct"
	PathResale = "resale"

	KindCSPUsage   = "csp_usage"
	KindMSPService = "msp_service"
	KindOther      = "other"

	RoutePlatform = "platform"
	RouteExternal = "external_order"
	RouteUnknown  = "unknown"

	CorrectsApplication = "application"
	CorrectsReversal    = "reversal"

	ModeInterval = "interval"
	ModeSource   = "source"

	StateReserved  = "reserved"
	StateConfirmed = "confirmed"
	StateReleased  = "released"
	StateReplaced  = "replaced"
	StateExpired   = "expired"
	StateReversed  = "reversed"

	ReversalPending   = "pending"
	ReversalEffective = "effective"

	ReasonNoEligibility      = "no_eligibility"
	ReasonNotEligible        = "not_eligible"
	ReasonUnknownInstance    = "unknown_instance"
	ReasonRoutedExternally   = "routed_externally"
	ReasonRouteUnknown       = "route_unknown"
	ReasonModeMismatch       = "mode_mismatch"
	ReasonCurrencyMismatch   = "currency_mismatch"
	ReasonAlreadyClaimed     = "already_claimed"
	ReasonCapacityMismatch   = "capacity_mismatch"
	ReasonSourceInOtherScope = "source_in_other_scope"
	ReasonCapacityExceeded   = "capacity_exceeded"
)

var (
	paths  = map[string]bool{PathDirect: true, PathResale: true}
	kinds  = map[string]bool{KindCSPUsage: true, KindMSPService: true, KindOther: true}
	routes = map[string]bool{RoutePlatform: true, RouteExternal: true, RouteUnknown: true}
	modes  = map[string]bool{ModeInterval: true, ModeSource: true}
)

type Component struct {
	ID        string
	Kind      string
	Billable  bool
	Route     string
	Reference string
	Instances []string
}

type Revision struct {
	From       string
	To         string
	Path       string
	Components []Component
}

type EligibilityView struct {
	Contract    string
	Number      int
	Revision    Revision
	EffectiveTo string
	Author      string
	Approver    string
	Digest      string
	WrittenAt   time.Time
	ApprovedAt  time.Time
}

type Scope struct {
	Contract  string
	Component string
	Instance  string
}

type Application struct {
	ID           string
	Scope        Scope
	Start        string
	End          string
	Mode         string
	Source       string
	CapQuantity  string
	CapAmount    string
	Quantity     string
	Amount       string
	Currency     string
	Calculation  string
	Replaces     string
	CorrectsKind string
	Corrects     string
	Lifetime     int
}

type ApplicationView struct {
	Application      Application
	State            string
	Expiry           time.Time
	ReversedQuantity string
	ReversedAmount   string
	ReservedBy       string
	ReservedAt       time.Time
	ConfirmedBy      string
	ConfirmedAt      time.Time
	ReleasedBy       string
	ReleasedAt       time.Time
	ReplacedBy       string
	ReplacedAt       time.Time
	Digest           string
}

type Decision struct {
	Reason      string
	Application ApplicationView
}

type Reversal struct {
	ID          string
	Application string
	Quantity    string
	Amount      string
}

type ReversalView struct {
	Reversal   Reversal
	State      string
	Author     string
	Approver   string
	Digest     string
	WrittenAt  time.Time
	ApprovedAt time.Time
}

type SourceView struct {
	Source       string
	Quantity     string
	Amount       string
	Currency     string
	Remaining    []Remaining
	UsedQuantity string
	UsedAmount   string
}

type Remaining struct {
	Application string
	Scope       Scope
	Quantity    string
	Amount      string
}

type Limits struct {
	Contracts    int
	Revisions    int
	Components   int
	Applications int
	Sources      int
	Reversals    int
	Batch        int
}

func (l Limits) valid() bool {
	return l.Contracts > 0 && l.Contracts <= 100000 && l.Revisions > 0 && l.Revisions <= 1000 && l.Components > 0 && l.Components <= 1000 &&
		l.Applications > 0 && l.Applications <= 1000000 && l.Sources > 0 && l.Sources <= 1000000 && l.Reversals > 0 && l.Reversals <= 1000 &&
		l.Batch > 0 && l.Batch <= 10000
}

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

func optionalID(id string) bool { return id == "" || validID(id) }

func validCurrency(code string) bool {
	return len(code) == 3 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= 'Z' && code[2] >= 'A' && code[2] <= 'Z'
}

func validDate(text string) bool {
	if len(text) != 10 || text < "2000-01-01" || text > "2099-12-31" {
		return false
	}
	parsed, err := time.Parse("2006-01-02", text)
	return err == nil && parsed.Format("2006-01-02") == text
}
