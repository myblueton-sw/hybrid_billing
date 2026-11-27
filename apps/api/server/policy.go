package server

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/authpolicy"
	"github.com/myblueton-sw/hybrid_billing/modules/decisionledger"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/trustchange"
)

const (
	minPolicyAge = 5
	maxPolicyAge = 720
	needPeople   = 3
)

type stepRow struct {
	Action        string `json:"action"`
	MaxAgeSeconds int64  `json:"max_age_seconds"`
	MFA           bool   `json:"mfa"`
}

type revisionRow struct {
	Methods       []string  `json:"methods"`
	MFA           string    `json:"mfa"`
	MaxAgeSeconds int64     `json:"max_age_seconds"`
	StepUp        []stepRow `json:"step_up"`
}

type epochRow struct {
	Value      string `json:"value"`
	Generation uint64 `json:"generation"`
	Number     uint64 `json:"number"`
}

type policyRow struct {
	revisionRow
	Number    uint64   `json:"number"`
	Epoch     epochRow `json:"epoch"`
	Proposer  string   `json:"proposer"`
	Validator string   `json:"validator"`
	Approver  string   `json:"approver"`
	Activator string   `json:"activator"`
	At        string   `json:"at"`
}

type holdRow struct {
	Reason string `json:"reason"`
	Since  string `json:"since"`
}

type trustRow struct {
	Epoch              string   `json:"epoch"`
	EpochGeneration    uint64   `json:"epoch_generation"`
	EpochNumber        uint64   `json:"epoch_number"`
	PolicyNumber       uint64   `json:"policy_number"`
	Digest             string   `json:"digest"`
	Origin             string   `json:"origin"`
	CarriedFrom        string   `json:"carried_from"`
	CarriedLine        int      `json:"carried_line"`
	DurableUnconfirmed bool     `json:"durable_unconfirmed"`
	ActivatedAt        string   `json:"activated_at"`
	ResetAt            string   `json:"reset_at"`
	OlderIgnored       bool     `json:"older_record_ignored"`
	StricterFields     []string `json:"stricter_fields"`
	Proposer           string   `json:"proposer"`
	Validator          string   `json:"validator"`
	Approver           string   `json:"approver"`
	Activator          string   `json:"activator"`
	Change             string   `json:"change"`
	Hold               *holdRow `json:"hold"`
}

type trustLedgerRow struct {
	State            string `json:"state"`
	IsolationReason  string `json:"isolation_reason"`
	ProblemLine      int    `json:"problem_line"`
	Records          uint64 `json:"records"`
	Watermark        string `json:"watermark"`
	Generation       uint64 `json:"generation"`
	GenerationStated bool   `json:"generation_stated"`
	NotDurable       bool   `json:"not_durable"`
}

type rightsRow struct {
	Propose  bool `json:"propose"`
	Validate bool `json:"validate"`
	Approve  bool `json:"approve"`
	Activate bool `json:"activate"`
}

type holdersRow struct {
	Qualifying int      `json:"qualifying_people"`
	Members    int      `json:"members"`
	Ready      bool     `json:"ready"`
	Missing    []string `json:"missing"`
	Needed     int      `json:"needed"`
	Enrolment  string   `json:"enrolment"`
}

type policyPage struct {
	Revision        string         `json:"revision"`
	Now             string         `json:"now"`
	SignedInAt      string         `json:"signed_in_at"`
	AuthenticatedAt string         `json:"authenticated_at"`
	ActUntil        string         `json:"act_until"`
	StepUpSeconds   int64          `json:"step_up_seconds"`
	You             string         `json:"you"`
	StartedAt       string         `json:"started_at"`
	Trust           trustRow       `json:"trust"`
	Policy          *policyRow     `json:"policy"`
	Ledger          trustLedgerRow `json:"ledger"`
	Rights          rightsRow      `json:"rights"`
	Holders         holdersRow     `json:"holders"`
	Waiting         string         `json:"waiting_activation"`
	EarlierRunKept  bool           `json:"earlier_run_kept"`
	Query           []settingRow   `json:"query_settings"`
}

func revisionOf(r authpolicy.Revision) revisionRow {
	out := revisionRow{Methods: append([]string{}, r.Methods...), MFA: r.MFA, MaxAgeSeconds: int64(r.MaxAge / time.Second), StepUp: []stepRow{}}
	for _, rule := range r.StepUp {
		out.StepUp = append(out.StepUp, stepRow{Action: rule.Action, MaxAgeSeconds: int64(rule.MaxAge / time.Second), MFA: rule.MFA})
	}
	return out
}

func epochOf(e uint64) epochRow {
	return epochRow{Value: strconv.FormatUint(e, 10), Generation: authpolicy.EpochGeneration(e), Number: authpolicy.EpochNumber(e)}
}

func window(r authpolicy.Revision) time.Duration {
	limit := r.MaxAge
	for _, rule := range r.StepUp {
		if rule.Action == authpolicy.ActionPolicyChange && rule.MaxAge < limit {
			limit = rule.MaxAge
		}
	}
	return limit
}

func (s *Server) listPolicy(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	defer s.domain.RUnlock()
	found, answer, ok := s.admitted(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, request, s.policyPage(found, answer))
}

func (s *Server) trustRow(base trustchange.Base) trustRow {
	o := s.trustOrigin
	row := trustRow{Epoch: strconv.FormatUint(base.Epoch, 10), EpochGeneration: authpolicy.EpochGeneration(base.Epoch), EpochNumber: authpolicy.EpochNumber(base.Epoch),
		PolicyNumber: base.Policy, Digest: base.Digest, Origin: o.origin, CarriedFrom: o.carriedFrom, CarriedLine: o.carriedLine, DurableUnconfirmed: o.unconfirmed,
		ActivatedAt: stampOrEmpty(o.activatedAt), ResetAt: stampOrEmpty(o.resetAt), Change: o.change, OlderIgnored: o.olderIgnored, StricterFields: append([]string{}, o.stricter...)}
	if current, err := s.policy.Policy(s.bootWorkspace); err == nil {
		row.Proposer, row.Validator, row.Approver, row.Activator = current.Proposer, current.Validator, current.Approver, current.Activator
	}
	if s.hold != nil {
		row.Hold = &holdRow{Reason: s.hold.reason, Since: stamp(s.hold.since)}
	}
	return row
}

func (s *Server) trustLedger() trustLedgerRow {
	if s.ledger == nil {
		return trustLedgerRow{State: string(decisionledger.StateIsolated), IsolationReason: string(decisionledger.IsolationMissing)}
	}
	st := s.ledger.Status()
	return trustLedgerRow{State: string(st.State), IsolationReason: string(st.Isolation), ProblemLine: st.ProblemLine, Records: st.Records, Watermark: st.Watermark,
		Generation: st.Generation, GenerationStated: st.Genesis.GenerationStated, NotDurable: st.NotDurable}
}

func (s *Server) policyPage(found session, answer identity.Answer) policyPage {
	now := s.clock()
	you := answer.User()
	page := policyPage{Revision: s.Revision(), Now: stamp(now), SignedInAt: stamp(found.created), AuthenticatedAt: stamp(found.authenticated),
		You: you, StartedAt: stamp(s.startedAt), Ledger: s.trustLedger(), Rights: s.rightsOf(you, now),
		Holders: holdersRow{Needed: needPeople, Enrolment: "not_decided", Missing: []string{}}}
	ctx := s.trustRows(found, you, now, page.Rights)
	page.Trust = s.trustRow(ctx.base)
	limit := ctx.limit
	var candidate *authpolicy.Revision
	if view, err := s.policy.Policy(s.bootWorkspace); err == nil {
		page.Policy = &policyRow{revisionRow: revisionOf(view.Revision), Number: view.Number, Epoch: epochOf(view.Epoch), Proposer: view.Proposer,
			Validator: view.Validator, Approver: view.Approver, Activator: view.Activator, At: stamp(view.At)}
		candidate = &ctx.current
		page.ActUntil = stamp(found.authenticated.Add(limit))
	}
	page.StepUpSeconds = int64(limit / time.Second)
	if f, err := s.trustFacts(found.verifier, s.bootWorkspace, candidate, now); err == nil {
		page.Holders.Qualifying, page.Holders.Members = f.qualifying(), f.members
		ready, detail := f.recovery()
		page.Holders.Ready = ready
		if missing, ok := strings.CutPrefix(detail, "missing:"); ok {
			page.Holders.Missing = strings.Split(missing, ",")
		}
	}
	summary := s.trust.Summary(s.bootWorkspace)
	page.Waiting, page.EarlierRunKept = summary.Unknown, summary.EarlierRun
	page.Query = s.querySettingRows()
	return page
}

func (s *Server) trustRows(found session, you string, now time.Time, rights rightsRow) rowContext {
	base, current := s.trustBase(s.bootWorkspace)
	limit := stepUpDefault
	if view, err := s.policy.Policy(s.bootWorkspace); err == nil {
		limit = min(limit, window(view.Revision))
	}
	return rowContext{s: s, you: you, base: base, current: current, now: now, verifier: found.verifier, since: found.authenticated, limit: limit,
		rights: rights, unavailable: s.trustUnavailable(false), activateUnavailable: s.trustUnavailable(true)}
}

func minutes(raw string, low, high int) (int, bool) {
	if len(raw) == 0 || len(raw) > 4 || (len(raw) > 1 && raw[0] == '0') || strings.TrimLeft(raw, "0123456789") != "" {
		return 0, false
	}
	n, _ := strconv.Atoi(raw)
	return n, n >= low && n <= high
}
