package marketplace

import (
	"errors"
	"time"
)

var (
	ErrInvalid   = errors.New("a request or an input is incomplete or malformed")
	ErrNotFound  = errors.New("what the request names does not exist in this workspace")
	ErrConflict  = errors.New("another fact is already recorded under this identifier")
	ErrRejected  = errors.New("a recorded state or rule refuses this request")
	ErrLimit     = errors.New("a configured limit or bound is exceeded")
	ErrSelfCheck = errors.New("a result failed its own check and nothing was returned")
)

const (
	maxIDBytes     = 128
	maxNumberBytes = 64
	statementLabel = "marketplace_statement_v1"
	settlementLbl  = "marketplace_settlement_v1"
	policyLabel    = "marketplace_policy_v1"
	ChargeLabel    = "marketplace_charge_v1"
)

const (
	RoleBuyer    = "buyer"
	RoleReseller = "reseller"
	RoleSeller   = "seller"

	ChannelDirect      = "direct"
	ChannelMarketplace = "marketplace"
	ChannelDistributor = "distributor"

	TypePublic         = "public_offer"
	TypePrivate        = "private_offer"
	TypeChannelPrivate = "channel_private_offer"

	CollectorMSP         = "msp"
	CollectorMarketplace = "marketplace"
	CollectorDistributor = "distributor"

	KindSoftware       = "software"
	KindInfrastructure = "infrastructure"
	KindOther          = "other"

	Applied    = "applied"
	NotApplied = "not_applied"
	Unknown    = "unknown"

	StateWritten    = "written"
	StateSuperseded = "superseded"
	StateSealed     = "sealed"
	StateHeld       = "held"

	CurrentSealed         = "sealed"
	CurrentCorrectionHeld = "correction_held"
	CurrentNotJudged      = "not_judged"

	ReasonDuplicateSource          = "duplicate_source"
	ReasonRecognitionExceedsCharge = "recognition_exceeds_charge"

	SettlementExpected = "expected"
	SettlementReported = "reported"
	SettlementPaid     = "paid"

	JudgedConsistent = "consistent"
	JudgedHeld       = "held"

	ReasonNetNotPreserved = "net_not_preserved"
	ReasonReceivedDiffers = "received_differs"
	ReasonPayoutReused    = "payout_reused"

	Excluded = "excluded"
	Included = "included"

	HoldSellerTransaction  = "seller_transaction"
	HoldCollectedElsewhere = "collected_elsewhere"
	HoldSourceUnknown      = "source_discount_unknown"
)

var (
	roles      = map[string]bool{RoleBuyer: true, RoleReseller: true, RoleSeller: true}
	channels   = map[string]bool{ChannelDirect: true, ChannelMarketplace: true, ChannelDistributor: true}
	types      = map[string]bool{TypePublic: true, TypePrivate: true, TypeChannelPrivate: true}
	collectors = map[string]bool{CollectorMSP: true, CollectorMarketplace: true, CollectorDistributor: true}
	lineKinds  = map[string]bool{KindSoftware: true, KindInfrastructure: true, KindOther: true}
	eligible   = map[string]bool{Applied: true, NotApplied: true, Unknown: true}
	rank       = map[string]int{SettlementExpected: 1, SettlementReported: 2, SettlementPaid: 3}
	quanta     = map[string]int{"1": 0, "0.1": 1, "0.01": 2, "0.001": 3, "0.0001": 4}
)

type Participants struct {
	SoftwareProvider string
	Seller           string
	Reseller         string
	Issuer           string
	Payee            string
}

type References struct {
	Product      string
	Offer        string
	Subscription string
	Order        string
}

type Term struct {
	Accepted string
	Start    string
	End      string
	Currency string
}

type Transaction struct {
	ID           string
	Provider     string
	Agreement    string
	Role         string
	Channel      string
	Type         string
	Collector    string
	Participants Participants
	References   References
	Term         Term
}

type Benefit struct {
	State    string
	Amount   string
	Evidence string
}

type Line struct {
	Source   string
	Kind     string
	Account  string
	Charged  string
	List     string
	Discount Benefit
}

type Recognition struct {
	Program  string
	State    string
	Amount   string
	Evidence string
}

type Facts struct {
	Lines        []Line
	Recognitions []Recognition
}

type RevisionState struct {
	Revision  int
	State     string
	Reasons   []string
	Digest    string
	WrittenAt time.Time
	JudgedAt  time.Time
}

type CurrentView struct {
	State    string
	Revision int
	Digest   string
}

type SettlementFacts struct {
	State    string
	Currency string
	Gross    string
	Fee      string
	Refund   string
	Tax      string
	Hold     string
	Net      string
	Report   string
	Payout   string
	Received string
}

type SettlementView struct {
	Transaction string
	Settlement  string
	Revision    int
	Facts       SettlementFacts
	Judged      string
	Reasons     []string
	Received    bool
	Digest      string
	WrittenAt   time.Time
}

type PolicyTerms struct {
	ID               string
	Version          string
	Customer         string
	Transaction      string
	First            string
	Last             string
	Rate             string
	Scope            []string
	SourceDiscounted string
	Evidence         string
	SourceBasis      string
	Quantum          string
}

type ChargeLine struct {
	Source   string
	Kind     string
	Charged  string
	Discount string
	Amount   string
}

type ChargeView struct {
	Label         string
	ProfileID     string
	ProfileRev    string
	Transaction   string
	Month         string
	Revision      int
	StatementHash string
	PolicyID      string
	PolicyVersion string
	PolicyDigest  string
	PolicyFirst   string
	PolicyLast    string
	Customer      string
	Hold          string
	Lines         []ChargeLine
	Recognitions  []Recognition
	Total         string
	ChargedTotal  string
	Margin        string
	Digest        string
}

type CustomerLine struct {
	Source string
	Kind   string
	Amount string
}

type CustomerView struct {
	Transaction string
	Month       string
	Customer    string
	Lines       []CustomerLine
	Total       string
	Digest      string
}

type KindSums struct {
	Software       string
	Infrastructure string
	Other          string
}

type AgreementEntry struct {
	Transaction string
	Role        string
	Currency    string
	State       string
	Revision    int
	Digest      string
	Sums        KindSums
}

type GroupSums struct {
	Role     string
	Currency string
	Sums     KindSums
}

type AgreementView struct {
	Provider  string
	Agreement string
	Month     string
	Entries   []AgreementEntry
	Groups    []GroupSums
	Complete  bool
}

type Limits struct {
	Transactions int
	Revisions    int
	Lines        int
	Recognitions int
	Settlements  int
	Policies     int
	Digits       int
}

func (l Limits) valid() bool {
	return l.Transactions > 0 && l.Transactions <= 100000 && l.Revisions > 0 && l.Revisions <= 1000 && l.Lines > 0 && l.Lines <= 100000 &&
		l.Recognitions > 0 && l.Recognitions <= 1000 && l.Settlements > 0 && l.Settlements <= 10000 && l.Policies > 0 && l.Policies <= 100000 &&
		l.Digits > 0 && l.Digits <= 10000
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

func validMonth(text string) bool {
	if len(text) != 7 || text[4] != '-' {
		return false
	}
	for _, index := range []int{0, 1, 2, 3, 5, 6} {
		if text[index] < '0' || text[index] > '9' {
			return false
		}
	}
	return text >= "2000-01" && text <= "2099-12" && text[5:] >= "01" && text[5:] <= "12"
}

func validDigest(text string) bool {
	if len(text) != 64 {
		return false
	}
	for index := 0; index < len(text); index++ {
		c := text[index]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
