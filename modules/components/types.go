package components

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
	maxIDBytes     = 128
	maxNumberBytes = 64
	maxScale       = 8
	componentLabel = "dc_component_v1"
	pricingLabel   = "dc_pricing_v1"
	incidentLabel  = "dc_incident_v1"
	clauseLabel    = "dc_sla_clause_v1"
	creditLabel    = "dc_sla_credit_v1"
)

const (
	KindRack        = "rack"
	KindSpace       = "space"
	KindLine        = "line"
	KindGPUReserved = "gpu_reserved"
	KindUsage       = "usage"
	KindService     = "service"

	All = "all"

	StatePending   = "pending"
	StateApproved  = "approved"
	StateWithdrawn = "withdrawn"

	LineFixed    = "fixed"
	LineUsage    = "usage"
	LineIncluded = "included"

	CreditLinked   = "linked"
	CreditUnlinked = "unlinked"

	ReasonMeterOverlap          = "meter_overlap"
	ReasonInclusionOverlap      = "inclusion_overlap"
	ReasonPartialPeriod         = "partial_period"
	ReasonCurrencyMismatch      = "currency_mismatch"
	ReasonMissingReading        = "missing_reading"
	ReasonNoPricing             = "no_pricing"
	ReasonClauseOverlap         = "clause_overlap"
	ReasonClauseOutside         = "clause_outside_component"
	ReasonCreditOverlap         = "credit_overlap"
	ReasonIncidentUsed          = "incident_used"
	ReasonIncidentOutsidePeriod = "incident_outside_period"
	ReasonZeroCredit            = "zero_credit"
	ReasonLinkUsed              = "link_used"
	ReasonComponentAbsent       = "component_absent"
)

var fixedKinds = map[string]bool{KindRack: true, KindSpace: true, KindLine: true, KindGPUReserved: true, KindService: true}

type Refusal struct {
	Reason string
}

func (e Refusal) Error() string { return ErrRejected.Error() + ": " + e.Reason }

func (e Refusal) Unwrap() error { return ErrRejected }

type Inclusion struct {
	Meter    string
	Quantity string
}

type Component struct {
	ID       string
	Contract string
	Currency string
	Kind     string
	From     string
	To       string
	Fee      string
	Includes []Inclusion
	Meter    string
	Rate     string
}

type ComponentView struct {
	Component  Component
	State      string
	Author     string
	Approver   string
	Written    time.Time
	ApprovedAt time.Time
	Digest     string
}

type Pricing struct {
	Contract string
	Scale    int
	Mode     string
}

type PricingView struct {
	Pricing Pricing
	Author  string
	Written time.Time
	Digest  string
}

type Reading struct {
	Meter    string
	Quantity string
}

type Request struct {
	Contract    string
	PeriodStart string
	PeriodEnd   string
	Usage       []Reading
}

type Line struct {
	Kind      string
	Component string
	Amount    string
	Meter     string
	Measured  string
	Included  string
	Billable  string
	Rate      string
	Includes  []Inclusion
}

type Unpriced struct {
	Meter    string
	Measured string
	Included string
	Excess   string
}

type CreditLine struct {
	Credit    string
	Component string
	Amount    string
	State     string
	Document  string
	Increase  string
}

type Statement struct {
	Contract    string
	Currency    string
	PeriodStart string
	PeriodEnd   string
	Lines       []Line
	Unpriced    []Unpriced
	Credits     []CreditLine
	Charges     string
	Net         string
	Evaluated   time.Time
}

type Incident struct {
	ID        string
	Contract  string
	Component string
	Start     time.Time
	End       time.Time
	Evidence  string
}

type IncidentView struct {
	Incident Incident
	Author   string
	Written  time.Time
	Credit   string
	Digest   string
}

type Clause struct {
	ID        string
	Contract  string
	Component string
	Rate      string
	From      string
	To        string
}

type ClauseView struct {
	Clause     Clause
	State      string
	Author     string
	Approver   string
	Written    time.Time
	ApprovedAt time.Time
	Digest     string
}

type Credit struct {
	ID          string
	Clause      string
	Incidents   []string
	PeriodStart string
	PeriodEnd   string
}

type Link struct {
	Document string
	Increase string
}

type CreditView struct {
	Credit      Credit
	Component   string
	Amount      string
	State       string
	Author      string
	Approver    string
	WithdrawnBy string
	Written     time.Time
	ApprovedAt  time.Time
	Link        Link
	LinkedBy    string
	LinkedAt    time.Time
	Digest      string
}

type Limits struct {
	Workspaces         int
	Components         int
	Inclusions         int
	Readings           int
	Incidents          int
	IncidentsPerCredit int
	Clauses            int
	Credits            int
}

func within(value, cap int) bool { return value > 0 && value <= cap }

func (l Limits) valid() bool {
	return within(l.Workspaces, 100000) && within(l.Components, 1000) && within(l.Inclusions, 100) && within(l.Readings, 1000) &&
		within(l.Incidents, 100000) && within(l.IncidentsPerCredit, 100) && within(l.Clauses, 1000) && within(l.Credits, 10000)
}
