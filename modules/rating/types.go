package rating

import "errors"

var (
	ErrInvalid   = errors.New("policy or line is incomplete or malformed")
	ErrLimit     = errors.New("a configured limit or bound is exceeded")
	ErrSelfCheck = errors.New("a result failed its own check and nothing was returned")
)

const (
	Algorithm  = "rating_v1"
	maxIDBytes = 128
)

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

func validQuantum(text string) bool {
	return text == "1" || text == "0.1" || text == "0.01" || text == "0.001" || text == "0.0001"
}

type Path string

const (
	Direct Path = "direct"
	Resale Path = "resale"
)

type Class string

const (
	CSPUsage   Class = "csp_usage"
	ServiceFee Class = "service_fee"
	Licence    Class = "licence"
	Other      Class = "other"
)

func (c Class) known() bool { return c == CSPUsage || c == ServiceFee || c == Licence || c == Other }

type Level string

const (
	SKU     Level = "sku"
	Service Level = "service"
	Group   Level = "group"
	Default Level = "default"
)

type Type string

const (
	PassThrough  Type = "pass_through"
	Markup       Type = "markup"
	TargetMargin Type = "target_margin"
	UnitPrice    Type = "unit_price"
	Graduated    Type = "graduated"
	Volume       Type = "volume"
	Flat         Type = "flat"
)

func (t Type) costBased() bool { return t == PassThrough || t == Markup || t == TargetMargin }

func (t Type) quantityBased() bool { return t == UnitPrice || t == Graduated || t == Volume }

type Basis string

const (
	Billed     Basis = "billed"
	Effective  Basis = "effective"
	List       Basis = "list"
	Contracted Basis = "contracted"
)

type Price struct {
	Amount string
	Per    int
}

type Tier struct {
	UpTo  string
	Price Price
}

type Discount struct {
	Percent string
	Fixed   string
}

type Rule struct {
	ID        string
	Level     Level
	Key       string
	Type      Type
	Rate      string
	Margin    string
	Price     Price
	Included  string
	Tiers     []Tier
	Amount    string
	Discounts []Discount
	Minimum   string
	Maximum   string
}

type Policy struct {
	ID        string
	Version   string
	Currency  string
	CostBasis Basis
	Quantum   string
	Mode      string
	Path      Path
	Eligible  []Class
	Rules     []Rule
}

type Line struct {
	ID       string
	Class    Class
	SKU      string
	Service  string
	Group    string
	Cost     string
	Quantity string
	Currency string
}

type Limits struct {
	Rules        int
	Tiers        int
	Discounts    int
	Lines        int
	Intermediate int
}

func (l Limits) valid() bool {
	return l.Rules > 0 && l.Tiers > 0 && l.Discounts > 0 && l.Lines > 0 && l.Intermediate > 0
}

const (
	Rated          = "rated"
	PassedAtCost   = "passed_at_cost"
	FlatCharge     = "flat"
	DiscountCharge = "discount"
	MinimumTopUp   = "minimum_top_up"
	MaximumCap     = "maximum_cap"

	ClassNotEligible = "class_not_eligible"
	NoRule           = "no_rule"
	CostNotKnown     = "cost_not_known"
	QuantityNotKnown = "quantity_not_known"
	RuleHeld         = "rule_held"

	StepCostsAbove    = "costs_above_zero"
	StepCostsBelow    = "costs_below_zero"
	StepQuantity      = "quantity"
	StepIncluded      = "included"
	StepAboveIncluded = "above_included"
	StepTier          = "tier"
	StepBase          = "base"
	StepBaseAmount    = "base_amount"
	StepPassed        = "passed"
	StepPassedAmount  = "passed_amount"
	StepDiscount      = "discount"
	StepMinimum       = "minimum"
	StepMaximum       = "maximum"
	StepTotal         = "total"
	StepHeld          = "held"
)

type Charge struct {
	Rule   string
	Kind   string
	Line   string
	Amount string
}

type Covered struct {
	Line string
	Rule string
}

type Excluded struct {
	Line   string
	Reason string
}

type Step struct {
	Name      string
	Basis     string
	Parameter string
	Outcome   string
}

type Trace struct {
	Rule  string
	Steps []Step
	Total string
}

type Count struct {
	Reason string
	Lines  int
}

type Result struct {
	PolicyID        string
	Version         string
	Currency        string
	CostBasis       Basis
	Path            Path
	Quantum         string
	Mode            string
	ProfileID       string
	ProfileRevision string
	Algorithm       string
	Lines           int
	Charges         []Charge
	Covered         []Covered
	Excluded        []Excluded
	Traces          []Trace
	Total           string
	NoRuleHeld      bool
	Exclusions      []Count
}
