package server

import (
	"encoding/hex"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/contracts"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

const (
	recordsShown    = 100
	maxPeriodDays   = 366
	maxResolves     = 8
	horizonDays     = 3660
	contractCommand = "contract."
)

var contractImpact = map[string]string{"money": "not_computed", "proration": "not_computed", "publication": "not_computed", "erp": "not_computed",
	"final_amount_decision": "not_computed"}

func parseContractDay(raw string) (contracts.Day, bool) {
	t, err := time.Parse(time.DateOnly, raw)
	if err != nil || t.Format(time.DateOnly) != raw {
		return contracts.Day{}, false
	}
	d, err := contracts.NewDay(t.Year(), int(t.Month()), t.Day())
	return d, err == nil
}

func contractToday(now time.Time) contracts.Day {
	u := now.UTC()
	return contractDay(u.Year(), int(u.Month()), u.Day())
}

type contractRow struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Profile     string `json:"profile"`
	ProfileName string `json:"profile_name"`
	Buyer       string `json:"buyer"`
	Payer       string `json:"payer"`
	Issuer      string `json:"issuer"`
	Path        string `json:"path"`
	Currency    string `json:"currency"`
	Timezone    string `json:"timezone"`
	Cycle       string `json:"cycle"`
	State       string `json:"state"`
	Revision    uint64 `json:"lifecycle_revision"`
	Settlement  string `json:"settlement"`
	Sequence    uint64 `json:"sequence"`
	Started     string `json:"start"`
	Ended       string `json:"end"`
	OpenGaps    int    `json:"open_gaps"`
}

type contractsPage struct {
	Revision  string            `json:"revision"`
	On        string            `json:"on"`
	Today     string            `json:"today"`
	Impact    map[string]string `json:"impact"`
	Contracts []contractRow     `json:"contracts"`
}

type contractView struct {
	seq        uint64
	events     []contracts.Event
	settlement contracts.Settlement
	records    []contracts.Record
	position   contracts.Position
}

func (s *Server) contractRead(ws, id string, on contracts.Day) (contractView, error) {
	if s.faulty("contract-read") {
		return contractView{}, contracts.ErrInvalid
	}
	var v contractView
	var errs [5]error
	v.seq, errs[0] = s.lifecycles.Sequence(ws, id)
	v.events, errs[1] = s.lifecycles.Lifecycle(ws, id, v.seq)
	v.settlement, errs[2] = s.lifecycles.Settlement(ws, id, v.seq)
	v.records, errs[3] = s.lifecycles.Records(ws, id)
	v.position, errs[4] = s.lifecycles.StateOn(ws, id, v.seq, on)
	if len(v.events) == 0 {
		errs[0] = errors.Join(errs[0], contracts.ErrNotFound)
	}
	return v, errors.Join(errs[:]...)
}

func openGaps(settlement contracts.Settlement) int {
	open := 0
	for _, e := range settlement.Evidence {
		if e.Evidence.Kind == contracts.Missing && !e.Resolved {
			open++
		}
	}
	return open
}

func finalState(events []contracts.Event) contracts.State {
	last := events[len(events)-1]
	period, _ := contracts.Between(last.From, mustAdd(last.From, 1))
	segments, _ := contracts.Segments(events, period)
	return segments[0].State
}

func mustAdd(day contracts.Day, days int) contracts.Day {
	next, _ := day.Add(days)
	return next
}

func (s *Server) contractRow(m *contractMeta, on contracts.Day) (contractRow, contractView, error) {
	v, err := s.contractRead(m.workspace, m.id, on)
	if err != nil {
		return contractRow{}, v, err
	}
	row := contractRow{ID: m.id, Name: m.name, Profile: m.profile, ProfileName: orgName(m.profile), Timezone: m.timezone, Cycle: m.cycle,
		Settlement: string(v.settlement.State), Sequence: v.seq, OpenGaps: openGaps(v.settlement), State: string(v.position.State), Revision: v.position.Revision,
		Started: v.events[0].From.String()}
	p, _, err := s.orgs.BillingProfile(m.workspace, m.profile)
	if err != nil {
		return contractRow{}, v, err
	}
	row.Buyer, row.Payer, row.Issuer, row.Path, row.Currency = p.BuyerID, p.PayerID, p.IssuerID, string(p.Path), p.Currency
	if day, ended := endDay(v.events); ended {
		row.Ended = day.String()
	}
	return row, v, nil
}

func (s *Server) workspaceContracts(ws string) []*contractMeta {
	mine := []*contractMeta{}
	for _, m := range s.contracts.byKey {
		if m.workspace == ws {
			mine = append(mine, m)
		}
	}
	sort.Slice(mine, func(a, b int) bool { return mine[a].id < mine[b].id })
	return mine
}

func (s *Server) listContracts(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	defer s.domain.RUnlock()
	_, answer, ok := s.admitted(w, r)
	if !ok {
		return
	}
	today := contractToday(s.clock())
	on := today
	if raw := r.URL.Query().Get("on"); raw != "" {
		day, valid := parseContractDay(raw)
		if !valid {
			failCode(w, commands.InvalidRequest, request)
			return
		}
		on = day
	}
	page := contractsPage{Revision: s.Revision(), On: on.String(), Today: today.String(), Impact: contractImpact, Contracts: []contractRow{}}
	mine := s.workspaceContracts(answer.Workspace())
	for _, m := range mine {
		row, _, err := s.contractRow(m, on)
		if err != nil {
			failCode(w, commands.DependencyUnavailable, request)
			return
		}
		page.Contracts = append(page.Contracts, row)
	}
	writeJSON(w, http.StatusOK, request, page)
}

type eventRow struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	From     string `json:"from"`
	Revision uint64 `json:"revision"`
	Withdraw string `json:"withdraw_blocked"`
	Cancel   bool   `json:"cancel_required"`
}

type segmentRow struct {
	From     string `json:"from"`
	Until    string `json:"until"`
	Days     int    `json:"days"`
	State    string `json:"state"`
	Revision uint64 `json:"revision"`
}

type evidenceRow struct {
	Sequence     uint64   `json:"sequence"`
	Write        string   `json:"write"`
	Kind         string   `json:"kind"`
	Reference    string   `json:"reference"`
	CoveredUntil string   `json:"covered_until"`
	Resolves     []string `json:"resolves"`
	Reason       string   `json:"reason"`
	Owner        string   `json:"owner"`
	OwnerLogin   string   `json:"owner_login"`
	FollowUp     string   `json:"follow_up"`
	Resolved     bool     `json:"resolved"`
}

type recordRow struct {
	Sequence   uint64   `json:"sequence"`
	At         string   `json:"at"`
	Write      string   `json:"write"`
	Kind       string   `json:"kind"`
	Added      []string `json:"added"`
	Withdrawn  []string `json:"withdrawn"`
	Cancel     string   `json:"cancel"`
	AsOf       string   `json:"as_of"`
	Reopen     string   `json:"reopen"`
	State      string   `json:"state"`
	Actor      string   `json:"actor"`
	ActorLogin string   `json:"actor_login"`
	Reason     string   `json:"reason"`
}

type actionRow struct {
	Allowed  bool   `json:"allowed"`
	Blocked  string `json:"blocked"`
	Earliest string `json:"earliest"`
}

type componentRow struct {
	ID             string `json:"id"`
	Kind           string `json:"kind"`
	Responsible    string `json:"provider_payment_responsible"`
	SellerBillable bool   `json:"seller_billable"`
}

type termsRow struct {
	Revision   uint64         `json:"revision"`
	Customer   string         `json:"customer"`
	Seller     string         `json:"seller"`
	Path       string         `json:"path"`
	Components []componentRow `json:"components"`
}

type contractDetail struct {
	Revision   string               `json:"revision"`
	Today      string               `json:"today"`
	From       string               `json:"from"`
	Until      string               `json:"until"`
	Impact     map[string]string    `json:"impact"`
	Contract   contractRow          `json:"contract"`
	Terms      []termsRow           `json:"terms"`
	Events     []eventRow           `json:"events"`
	Segments   []segmentRow         `json:"segments"`
	Reopenings int                  `json:"reopenings"`
	Evidence   []evidenceRow        `json:"evidence"`
	Records    []recordRow          `json:"records"`
	Earlier    int                  `json:"earlier_records"`
	Actions    map[string]actionRow `json:"actions"`
}

func withdrawable(e contracts.Event, today contracts.Day) bool {
	return e.Kind != contracts.Start && e.Kind != contracts.Change && !e.From.Before(today)
}

func monthStart(day contracts.Day) contracts.Day {
	t, _ := time.Parse(time.DateOnly, day.String())
	return contractDay(t.Year(), int(t.Month()), 1)
}

func nextMonth(day contracts.Day) contracts.Day {
	t, _ := time.Parse(time.DateOnly, day.String())
	n := time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	return contractDay(n.Year(), int(n.Month()), 1)
}

func detailPeriod(r *http.Request, today contracts.Day) (contracts.Interval, bool) {
	q := r.URL.Query()
	from, until := monthStart(today), nextMonth(today)
	if raw := q.Get("from"); raw != "" {
		from, _ = parseContractDay(raw)
		until = nextMonth(from)
	}
	if raw := q.Get("until"); raw != "" {
		until, _ = parseContractDay(raw)
	}
	period, err := contracts.Between(from, until)
	return period, err == nil && period.Days() <= maxPeriodDays
}

func (s *Server) readContract(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	defer s.domain.RUnlock()
	_, answer, ok := s.admitted(w, r)
	if !ok {
		return
	}
	m := s.contracts.get(answer.Workspace(), r.PathValue("id"))
	if m == nil {
		failCode(w, commands.UnsupportedCommand, request)
		return
	}
	today := contractToday(s.clock())
	period, valid := detailPeriod(r, today)
	if !valid {
		failCode(w, commands.InvalidRequest, request)
		return
	}
	row, v, err := s.contractRow(m, today)
	if err != nil {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	events, settlement, records := v.events, v.settlement, v.records
	detail := contractDetail{Revision: s.Revision(), Today: today.String(), From: period.Start().String(), Until: period.End().String(), Impact: contractImpact,
		Contract: row, Terms: s.contractTerms(m), Events: []eventRow{}, Segments: []segmentRow{}, Reopenings: settlement.Reopenings,
		Evidence: []evidenceRow{}, Records: []recordRow{}}
	for _, e := range events {
		blocked, cancel := withdrawHint(events, settlement, e.ID, today, roomOf(s.lifecycleLimits, records))
		detail.Events = append(detail.Events, eventRow{ID: e.ID, Kind: string(e.Kind), From: e.From.String(), Revision: e.Revision, Withdraw: blocked, Cancel: cancel})
	}
	segments, _ := contracts.Segments(events, period)
	for _, g := range segments {
		detail.Segments = append(detail.Segments, segmentRow{From: g.Interval.Start().String(), Until: g.Interval.End().String(), Days: g.Interval.Days(),
			State: string(g.State), Revision: g.Revision})
	}
	for _, e := range settlement.Evidence {
		detail.Evidence = append(detail.Evidence, evidenceRow{Sequence: e.Sequence, Write: e.Write, Kind: string(e.Evidence.Kind), Reference: e.Evidence.Reference,
			CoveredUntil: e.Evidence.CoveredUntil.String(), Resolves: append([]string{}, e.Evidence.Resolves...), Reason: string(e.Evidence.Reason),
			Owner: e.Evidence.OwnerID, OwnerLogin: s.loginOf(e.Evidence.OwnerID), FollowUp: e.Evidence.FollowUp, Resolved: e.Resolved})
	}
	if len(records) > recordsShown {
		detail.Earlier = len(records) - recordsShown
		records = records[len(records)-recordsShown:]
	}
	for index := len(records) - 1; index >= 0; index-- {
		detail.Records = append(detail.Records, s.recordRow(m, records[index]))
	}
	detail.Actions = actionsFor(events, settlement, v.records, today, roomOf(s.lifecycleLimits, v.records))
	writeJSON(w, http.StatusOK, request, detail)
}

func (s *Server) contractTerms(m *contractMeta) []termsRow {
	terms := []termsRow{}
	for number := uint64(1); ; number++ {
		rev, err := s.orgs.ContractRevision(m.workspace, m.id, number)
		if err != nil {
			return terms
		}
		row := termsRow{Revision: rev.Revision, Customer: rev.CustomerID, Seller: rev.SellerID, Path: string(rev.Path), Components: []componentRow{}}
		for _, c := range rev.Components {
			row.Components = append(row.Components, componentRow{ID: c.ID, Kind: string(c.Kind), Responsible: c.ProviderPaymentResponsibleID, SellerBillable: c.SellerBillable})
		}
		terms = append(terms, row)
	}
}

func (s *Server) recordRow(m *contractMeta, record contracts.Record) recordRow {
	who := s.contracts.written(m.workspace, m.id, record.Write)
	row := recordRow{Sequence: record.Sequence, At: stamp(record.At), Write: record.Write, Kind: string(record.Kind), Added: []string{},
		Withdrawn: append([]string{}, record.Withdrawals...), Cancel: string(record.Cancel), AsOf: record.AsOf.String(), Reopen: string(record.Reopen),
		State: string(record.State), Actor: who.actor, ActorLogin: s.loginOf(who.actor), Reason: who.reason}
	for _, e := range record.Events {
		row.Added = append(row.Added, e.ID)
	}
	return row
}

func earliest(events []contracts.Event, today contracts.Day) contracts.Day {
	last := events[len(events)-1].From
	if last.Before(today) {
		return today
	}
	return mustAdd(last, 1)
}

func validAfter(events []contracts.Event) bool {
	first := events[0].From
	period, _ := contracts.Between(first, mustAdd(first, 1))
	_, err := contracts.Segments(events, period)
	return err == nil
}

func endDay(events []contracts.Event) (contracts.Day, bool) {
	last := events[len(events)-1]
	return last.From, last.Kind == contracts.End
}

func lifecycleHint(events []contracts.Event, kind contracts.Kind) string {
	switch final := finalState(events); {
	case final == contracts.Ended:
		return "ended"
	case kind == contracts.Suspend && final != contracts.Active:
		return "not_active"
	case kind == contracts.Resume && final != contracts.Suspended:
		return "not_suspended"
	}
	return ""
}

type room struct {
	used, writes, reserve int
}

func roomOf(limits contracts.Limits, records []contracts.Record) room {
	return room{used: len(records), writes: limits.Writes, reserve: limits.Reserve}
}

func (r room) full(reserved bool) bool {
	if reserved {
		return r.used >= r.writes+r.reserve
	}
	return r.used >= r.writes
}

func withdrawHint(events []contracts.Event, settlement contracts.Settlement, id string, today contracts.Day, space room) (string, bool) {
	remaining := []contracts.Event{}
	allowed := false
	for _, e := range events {
		if e.ID == id {
			allowed = withdrawable(e, today)
			continue
		}
		remaining = append(remaining, e)
	}
	switch {
	case !allowed:
		return "not_withdrawable", false
	case settlement.State == contracts.Settled:
		return "settled", false
	case !validAfter(remaining):
		return "breaks_lifecycle", false
	case space.full(false):
		return "write_limit", false
	}
	_, ended := endDay(remaining)
	return "", settlement.State == contracts.PendingFinal && !ended
}

func securedHint(events []contracts.Event, settlement contracts.Settlement, today contracts.Day) string {
	if settlement.State != contracts.PendingFinal {
		return "not_pending_final"
	}
	if day, ended := endDay(events); ended && today.Before(day) {
		return "end_not_reached"
	}
	return ""
}

func settleHint(events []contracts.Event, settlement contracts.Settlement, records []contracts.Record, today contracts.Day) string {
	if blocked := securedHint(events, settlement, today); blocked != "" {
		return blocked
	}
	if openGaps(settlement) > 0 {
		return "open_gaps"
	}
	day, _ := endDay(events)
	fence := uint64(0)
	for _, r := range records {
		if r.Kind == contracts.WriteBatch || r.Kind == contracts.WriteReopen {
			fence = r.Sequence
		}
	}
	if len(settlement.Evidence) == 0 {
		return "evidence_needed"
	}
	latest := settlement.Evidence[len(settlement.Evidence)-1]
	if latest.Evidence.Kind != contracts.Secured || latest.Sequence <= fence || latest.Evidence.CoveredUntil.Before(day) {
		return "evidence_needed"
	}
	return ""
}

func reservedKind(kind contracts.Kind) bool {
	return kind == contracts.Suspend || kind == contracts.End
}

func limited(blocked string, space room, reserved bool) string {
	if blocked == "" && space.full(reserved) {
		return "write_limit"
	}
	return blocked
}

func actionsFor(events []contracts.Event, settlement contracts.Settlement, records []contracts.Record, today contracts.Day, space room) map[string]actionRow {
	row := func(blocked string) actionRow { return actionRow{Allowed: blocked == "", Blocked: blocked} }
	settled := settlement.State == contracts.Settled
	lifecycle := func(kind contracts.Kind) actionRow {
		if settled {
			return row("settled")
		}
		blocked := lifecycleHint(events, kind)
		day := earliest(events, today)
		if blocked == "" && mustAdd(today, horizonDays).Before(day) {
			blocked = "horizon"
		}
		r := row(limited(blocked, space, reservedKind(kind)))
		if r.Allowed {
			r.Earliest = day.String()
		}
		return r
	}
	withdraw := "nothing_withdrawable"
	for _, e := range events {
		if blocked, _ := withdrawHint(events, settlement, e.ID, today, space); blocked == "" {
			withdraw = ""
		}
	}
	if settled {
		withdraw = "settled"
	}
	missing, reopen := "", ""
	if settled {
		missing = "settled"
	} else {
		reopen = "not_settled"
	}
	return map[string]actionRow{
		"contract.suspend":          lifecycle(contracts.Suspend),
		"contract.resume":           lifecycle(contracts.Resume),
		"contract.renew":            lifecycle(contracts.Renew),
		"contract.end":              lifecycle(contracts.End),
		"contract.withdraw":         row(withdraw),
		"contract.evidence.secured": row(limited(securedHint(events, settlement, today), space, false)),
		"contract.evidence.missing": row(limited(missing, space, true)),
		"contract.settle":           row(limited(settleHint(events, settlement, records, today), space, false)),
		"contract.reopen":           row(limited(reopen, space, true)),
	}
}

type lifecycleCommand struct {
	s    *Server
	kind string
}

func (h lifecycleCommand) Resolve(workspace, target string) (commands.Target, commands.Resolution) {
	s := h.s
	if workspace != s.bootWorkspace || s.contracts.get(workspace, target) == nil {
		return commands.Target{}, commands.NotFound
	}
	return commands.Target{Revision: s.Revision()}, commands.Found
}

func newWrite() string { return "cw-" + hex.EncodeToString(random(8)) }

func newEvent() string { return "ev-" + hex.EncodeToString(random(8)) }

func (h lifecycleCommand) Apply(ctx commands.Context) (commands.Outcome, error) {
	s := h.s
	if len(ctx.Rationale) > maxRationale {
		return commands.Outcome{}, refuse(commands.InvalidRequest)
	}
	m := s.contracts.get(s.bootWorkspace, ctx.Target)
	seq, err := s.lifecycles.Sequence(m.workspace, m.id)
	if err != nil || s.faulty("contract-sequence") {
		return commands.Outcome{}, refuse(commands.DependencyUnavailable)
	}
	write := newWrite()
	today := contractToday(s.clock())
	var fields map[string]string
	switch h.kind {
	case "withdraw":
		fields, err = h.withdraw(ctx, m, seq, write, today)
	case "evidence":
		err = h.evidence(ctx, m, seq, write, today)
	case "settle":
		err = h.settle(ctx, m, seq, write, today)
	case "reopen":
		err = h.reopen(ctx, m, seq, write)
	default:
		fields, err = h.add(ctx, m, seq, write, today)
	}
	if err != nil {
		return commands.Outcome{}, err
	}
	s.contracts.note(m.workspace, m.id, write, ctx.User, ctx.Rationale)
	settlement, _ := s.lifecycles.Settlement(m.workspace, m.id, seq+1)
	if fields == nil {
		fields = map[string]string{}
	}
	fields["write"], fields["sequence"] = write, strconv.FormatUint(seq+1, 10)
	return commands.Outcome{ResultRevision: s.Next(), BusinessState: string(settlement.State), Fields: fields}, nil
}

func mapContract(err error, rejected commands.Code, fields ...string) error {
	switch {
	case errors.Is(err, contracts.ErrRejected):
		return refuse(rejected, fields...)
	case errors.Is(err, contracts.ErrState):
		return refuse(commands.InvalidStage)
	case errors.Is(err, contracts.ErrLimit):
		return refuse(commands.PrerequisiteFailed)
	case errors.Is(err, contracts.ErrConflict):
		return refuse(commands.RevisionConflict)
	}
	return refuse(commands.InternalError)
}

func (h lifecycleCommand) add(ctx commands.Context, m *contractMeta, seq uint64, write string, today contracts.Day) (map[string]string, error) {
	from, ok := parseContractDay(ctx.Fields["from"])
	if !ok || from.Before(today) || mustAdd(today, horizonDays).Before(from) {
		return nil, refuse(commands.InvalidRequest, "from")
	}
	event := contracts.Event{ID: newEvent(), Kind: contracts.Kind(h.kind), From: from}
	var err error
	if h.s.faulty("contract-amend") {
		err = contracts.ErrInvalid
	} else {
		_, err = h.s.lifecycles.Amend(m.workspace, m.id, seq, write, nil, []contracts.Event{event}, "")
	}
	if err != nil {
		return nil, mapContract(err, commands.InvalidStage)
	}
	return map[string]string{"event": event.ID}, nil
}

func (h lifecycleCommand) withdraw(ctx commands.Context, m *contractMeta, seq uint64, write string, today contracts.Day) (map[string]string, error) {
	v, err := h.s.contractRead(m.workspace, m.id, today)
	if err != nil {
		return nil, refuse(commands.DependencyUnavailable)
	}
	events, settlement := v.events, v.settlement
	name := ctx.Fields["event"]
	blocked, needs := withdrawHint(events, settlement, name, today, room{writes: 1})
	switch blocked {
	case "not_withdrawable":
		return nil, refuse(commands.InvalidRequest, "event")
	case "":
		cancel := contracts.CancelReason(ctx.Fields["cancel"])
		if (cancel != "") != needs {
			return nil, refuse(commands.InvalidRequest, "cancel")
		}
		if h.s.faulty("contract-amend") {
			err = contracts.ErrInvalid
		} else {
			_, err = h.s.lifecycles.Amend(m.workspace, m.id, seq, write, []string{name}, nil, cancel)
		}
	case "settled":
		err = contracts.ErrState
	default:
		err = contracts.ErrRejected
	}
	if err != nil {
		return nil, mapContract(err, commands.InvalidStage)
	}
	return map[string]string{"event": name}, nil
}

func evidenceOf(f map[string]string, user string) (contracts.Evidence, string) {
	evidence := contracts.Evidence{Kind: contracts.EvidenceKind(f["kind"])}
	if evidence.Kind == contracts.Missing {
		for _, name := range []string{"reference", "covered_until", "resolves"} {
			if f[name] != "" {
				return evidence, name
			}
		}
		if f["reason"] == "" {
			return evidence, "reason"
		}
		if !validWorkspace(f["follow_up"]) {
			return evidence, "follow_up"
		}
		evidence.Reason, evidence.OwnerID, evidence.FollowUp = contracts.GapReason(f["reason"]), user, f["follow_up"]
		return evidence, ""
	}
	for _, name := range []string{"reason", "follow_up"} {
		if f[name] != "" {
			return evidence, name
		}
	}
	if !validWorkspace(f["reference"]) {
		return evidence, "reference"
	}
	covered, ok := parseContractDay(f["covered_until"])
	if !ok {
		return evidence, "covered_until"
	}
	evidence.Reference, evidence.CoveredUntil = f["reference"], covered
	if f["resolves"] == "" {
		return evidence, ""
	}
	seen := map[string]bool{}
	for _, name := range strings.Split(f["resolves"], ",") {
		if !validWorkspace(name) || seen[name] || len(seen) == maxResolves {
			return evidence, "resolves"
		}
		seen[name] = true
		evidence.Resolves = append(evidence.Resolves, name)
	}
	return evidence, ""
}

func (h lifecycleCommand) evidence(ctx commands.Context, m *contractMeta, seq uint64, write string, today contracts.Day) error {
	evidence, bad := evidenceOf(ctx.Fields, ctx.User)
	if bad == "" && evidence.Kind == contracts.Secured && today.Before(evidence.CoveredUntil) {
		bad = "covered_until"
	}
	if bad != "" {
		return refuse(commands.InvalidRequest, bad)
	}
	v, err := h.s.contractRead(m.workspace, m.id, today)
	if err != nil {
		return refuse(commands.DependencyUnavailable)
	}
	day, ended := endDay(v.events)
	if (evidence.Kind == contracts.Secured && ended && today.Before(day)) || (evidence.Kind == contracts.Missing && v.settlement.State == contracts.Settled) {
		return refuse(commands.InvalidStage)
	}
	if h.s.faulty("contract-evidence") {
		err = contracts.ErrInvalid
	} else {
		_, err = h.s.lifecycles.RecordEvidence(m.workspace, m.id, seq, write, evidence)
	}
	if err != nil {
		return mapContract(err, commands.InvalidRequest, "resolves")
	}
	return nil
}

func (h lifecycleCommand) settle(ctx commands.Context, m *contractMeta, seq uint64, write string, today contracts.Day) error {
	asOf, ok := parseContractDay(ctx.Fields["as_of"])
	if !ok || today.Before(asOf) {
		return refuse(commands.InvalidRequest, "as_of")
	}
	var err error
	if h.s.faulty("contract-settle") {
		err = contracts.ErrInvalid
	} else {
		_, err = h.s.lifecycles.Settle(m.workspace, m.id, seq, write, asOf)
	}
	if err != nil {
		return mapContract(err, commands.PrerequisiteFailed)
	}
	return nil
}

func (h lifecycleCommand) reopen(ctx commands.Context, m *contractMeta, seq uint64, write string) error {
	var err error
	if h.s.faulty("contract-reopen") {
		err = contracts.ErrInvalid
	} else {
		_, err = h.s.lifecycles.Reopen(m.workspace, m.id, seq, write, contracts.ReopenReason(ctx.Fields["reason"]))
	}
	if err != nil {
		return mapContract(err, commands.PrerequisiteFailed)
	}
	return nil
}

func (s *Server) contractDefinitions() []commands.Definition {
	permission := commands.Permission{Action: gate.AdministerInstallation, Kind: gate.InstallationKind}
	define := func(kind string, f ...commands.Field) commands.Definition {
		return commands.Definition{ID: contractCommand + kind, Permission: permission, Mutating: true, Rationale: true, Fields: f,
			Handler: lifecycleCommand{s: s, kind: kind}}
	}
	from := commands.Field{Name: "from", Type: commands.Text, Required: true}
	return []commands.Definition{
		define("suspend", from),
		define("resume", from),
		define("renew", from),
		define("end", from),
		define("withdraw", commands.Field{Name: "event", Type: commands.ID, Required: true},
			commands.Field{Name: "cancel", Type: commands.Enum, Values: []string{string(contracts.RecordedInError), string(contracts.ContractContinues)}}),
		define("evidence", commands.Field{Name: "kind", Type: commands.Enum, Required: true, Values: []string{string(contracts.Secured), string(contracts.Missing)}},
			commands.Field{Name: "reference", Type: commands.Text}, commands.Field{Name: "covered_until", Type: commands.Text},
			commands.Field{Name: "resolves", Type: commands.Text},
			commands.Field{Name: "reason", Type: commands.Enum, Values: []string{string(contracts.SourceUnavailable), string(contracts.AccessWithdrawnEmergency),
				string(contracts.ProviderRefused), string(contracts.NotAttempted), string(contracts.LateSourceReceived)}},
			commands.Field{Name: "follow_up", Type: commands.Text}),
		define("settle", commands.Field{Name: "as_of", Type: commands.Text, Required: true}),
		define("reopen", commands.Field{Name: "reason", Type: commands.Enum, Required: true,
			Values: []string{string(contracts.LateSource), string(contracts.LateCredit), string(contracts.Correcting)}}),
	}
}
