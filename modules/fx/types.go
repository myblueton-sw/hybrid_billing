package fx

import (
	"errors"
	"time"
)

var (
	ErrInvalid  = errors.New("a request or a configuration is incomplete or malformed")
	ErrNotFound = errors.New("what the request names does not exist in this workspace")
	ErrConflict = errors.New("another fact is already recorded under this identifier")
	ErrRejected = errors.New("a recorded state or rule refuses this request")
	ErrLimit    = errors.New("a configured limit or bound is exceeded")
)

const (
	Label          = "fx_v1"
	contentLabel   = "fx_snapshot_content_v1"
	snapshotLabel  = "fx_snapshot_v1"
	maxIDBytes     = 128
	maxNumberBytes = 64
	maxQuoteScale  = 8
	minOffset      = -720
	maxOffset      = 840
	maxDays        = 31
	maxRevision    = 1000
	maxVersion     = 10000
	firstYear      = 2000
	lastYear       = 2099
)

const (
	OriginFeed   = "feed"
	OriginManual = "manual"
	OriginFile   = "file"

	ObservationRecorded     = "recorded"
	ObservationAwaiting     = "awaiting_approval"
	ObservationApproved     = "approval_recorded"
	ObservationVoidAwaiting = "void_awaiting_approval"
	ObservationVoid         = "void"
	VersionAwaiting         = "awaiting_approval"
	VersionApproved         = "approval_recorded"
	VersionRecorded         = "recorded"
	LevelCustomer           = "customer"
	LevelContract           = "contract"
	LevelMSP                = "msp"
	MethodEnded             = "ended"
	MethodDefaultAllowed    = "default_allowed"
	MethodFixed             = "fixed"
	MethodDated             = "dated"
	MethodMonthlyAverage    = "monthly_average"
	ConstructionDirect      = "direct"
	ConstructionInverse     = "inverse"
	ConstructionCross       = "cross"
	RuleHold                = "hold"
	RulePreviousRecorded    = "previous_recorded"
	MonthSame               = "same_month"
	MonthPrevious           = "previous_month"
	Averaging               = "arithmetic_mean_of_recorded_days"
	Order                   = "convert_after_source_charges"
	StateSource             = "source"
	StateAlreadyInTarget    = "already_in_target"
	EntrySameCurrency       = "same_currency"
	EntryRate               = "rate"
	EntryHeld               = "held"
	ResultSameCurrency      = "not_converted_same_currency"
	ResultAlreadyInTarget   = "not_converted_already_in_target"
	ResultHeld              = "held"
	ResultConverted         = "converted"
	ReasonCustomerException = "customer_exception"
	ReasonContract          = "contract"
	ReasonMSPDefault        = "msp_default"
	ReasonContractNotStated = "contract_not_stated"
	ReasonNoDefaultPolicy   = "no_default_policy"
	ReasonWithdrawnInForce  = "withdrawn_in_force"
	ReasonDateNotReached    = "date_not_reached"
	ReasonMonthNotEnded     = "month_not_ended"
	ReasonAwaitingApproval  = "awaiting_approval"
	ReasonNoRateRecorded    = "no_rate_recorded"
	ReasonLegsNotOnOneDate  = "legs_not_on_one_date"
	ReasonNotEnoughDays     = "not_enough_days"
	ReasonRateRoundsToZero  = "rate_rounds_to_zero"
	ReasonRatePrecisionLost = "rate_precision_lost"
)

var (
	units          = map[int]bool{1: true, 10: true, 100: true, 1000: true, 10000: true}
	origins        = map[string]bool{OriginFeed: true, OriginManual: true, OriginFile: true}
	voidReasons    = map[string]bool{"mistake": true, "duplicate": true, "source_withdrew": true}
	withdrawReason = map[string]bool{"mistake": true, "replaced": true}
	levels         = map[string]bool{LevelCustomer: true, LevelContract: true, LevelMSP: true}
	purposes       = map[string]bool{"billing": true, "display": true, "accounting": true}
	quanta         = map[string]int{"1": 0, "0.1": 1, "0.01": 2, "0.001": 3, "0.0001": 4}
)

type Key struct {
	Source string
	Series string
	Base   string
	Quote  string
	Date   string
}

type ObservationInput struct {
	Key      Key
	Revision int
	Unit     int
	Value    string
	Origin   string
	Evidence string
	Recorder string
}

type Approval struct {
	Approver  string
	Reference string
	At        time.Time
}

type Void struct {
	Voider   string
	Evidence string
	Reason   string
	At       time.Time
	Approval *Approval
}

type RevisionView struct {
	Revision   int
	State      string
	Unit       int
	Value      string
	Origin     string
	Evidence   string
	Recorder   string
	RecordedAt time.Time
	Approval   *Approval
	Void       *Void
}

type PolicyAttributes struct {
	Level   string
	Key     string
	Purpose string
	Source  string
	Target  string
}

type VersionInput struct {
	Version      int
	Effective    string
	Author       string
	Method       string
	Unit         int
	Value        string
	Evidence     string
	RateSource   string
	Series       string
	Construction string
	Pivot        string
	Rule         string
	MaxDays      int
	Month        string
	Averaging    string
	LeastDays    int
	Spread       string
	Fee          string
	QuoteScale   int
	QuoteMode    string
	Tolerance    string
	Quantum      string
	AmountMode   string
	Order        string
}

type Withdrawal struct {
	Withdrawer string
	Reason     string
	At         time.Time
}

type VersionView struct {
	VersionInput
	State      string
	RecordedAt time.Time
	Approval   *Approval
	Withdrawal *Withdrawal
}

type PolicyView struct {
	ID         string
	Attributes PolicyAttributes
	Versions   []VersionView
}

type Need struct {
	ID       string
	Purpose  string
	Source   string
	Target   string
	Customer string
	Contract string
	Date     string
	Offset   int
}

type Used struct {
	Key       Key
	Revision  int
	Unit      int
	Value     string
	Origin    string
	Evidence  string
	Reference string
}

type Entry struct {
	Need            Need
	Kind            string
	Reason          string
	Level           string
	Policy          string
	Version         int
	AllowingPolicy  string
	AllowingVersion int
	Method          string
	Construction    string
	Pivot           string
	Month           string
	Averaging       string
	LeastDays       int
	Rule            string
	MaxDays         int
	Observations    []Used
	FixedUnit       int
	FixedValue      string
	FixedEvidence   string
	FixedReference  string
	DateUsed        string
	DaysElapsed     int
	VoidsPassed     int
	PerSource       string
	QuotedRate      string
	Unit            int
	Spread          string
	ExactQuote      string
	SalesQuote      string
	QuoteScale      int
	QuoteMode       string
	Tolerance       string
	Difference      string
	SalesRate       string
	Fee             string
	Quantum         string
	AmountMode      string
	Order           string
}

type Snapshot struct {
	ID            string
	PreparedAt    time.Time
	Entries       []Entry
	Held          int
	ContentDigest string
	Digest        string
}

type Listed struct {
	ID            string
	ContentDigest string
	Digest        string
}

type Line struct {
	ID       string
	Need     string
	Amount   string
	Currency string
	State    string
}

type Converted struct {
	Line     string
	Kind     string
	Reason   string
	Currency string
	Amount   string
	Exact    string
	AtQuoted string
	Fee      string
	Total    string
}

type Result struct {
	Label         string
	Snapshot      string
	ContentDigest string
	Digest        string
	Held          int
	Lines         []Converted
}

type Limits struct {
	Keys      int
	Revisions int
	Policies  int
	Versions  int
	Needs     int
	Snapshots int
	Lines     int
	Digits    int
	Page      int
}

func (l Limits) valid() bool {
	return l.Keys > 0 && l.Keys <= 1000000 && l.Revisions > 0 && l.Revisions <= 1000 &&
		l.Policies > 0 && l.Policies <= 100000 && l.Versions > 0 && l.Versions <= 10000 &&
		l.Needs > 0 && l.Needs <= 10000 && l.Snapshots > 0 && l.Snapshots <= 100000 &&
		l.Lines > 0 && l.Lines <= 100000 && l.Digits > 0 && l.Digits <= 10000 && l.Page > 0 && l.Page <= 1000
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

func validCurrency(code string) bool {
	return len(code) == 3 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= 'Z' && code[2] >= 'A' && code[2] <= 'Z'
}

func digitsOf(text string) (int, bool) {
	value := 0
	if text == "" {
		return 0, false
	}
	for index := 0; index < len(text); index++ {
		if text[index] < '0' || text[index] > '9' {
			return 0, false
		}
		value = value*10 + int(text[index]-'0')
	}
	return value, true
}

func parseDate(text string) (time.Time, bool) {
	if len(text) != 10 || text[4] != '-' || text[7] != '-' {
		return time.Time{}, false
	}
	year, okYear := digitsOf(text[0:4])
	month, okMonth := digitsOf(text[5:7])
	day, okDay := digitsOf(text[8:10])
	if !okYear || !okMonth || !okDay || year < firstYear || year > lastYear || month < 1 || month > 12 || day < 1 {
		return time.Time{}, false
	}
	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if date.Day() != day {
		return time.Time{}, false
	}
	return date, true
}

func dateText(date time.Time) string { return date.Format("2006-01-02") }
