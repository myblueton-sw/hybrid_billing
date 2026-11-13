package hawkeye

import (
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

type posted struct {
	stated Line
	print  string
	cost   *number
}

type statement struct {
	supersedes   string
	supersededBy string
	live         *reception
}

type source struct {
	statements map[string]*statement
	receptions []*reception
	sequence   uint64
}

type reception struct {
	spec       Spec
	print      string
	origin     *source
	statement  *statement
	declared   map[string]*number
	state      State
	wasClosed  bool
	lines      map[string]*posted
	currencies map[string]bool
	teams      []string
	sums       map[string]*number
	unmeasured bool
	reason     VoidReason
	opened     time.Time
	changed    time.Time
}

type workspace struct {
	receptions map[string]*reception
	sources    map[string]*source
}

type Registry struct {
	mu         sync.Mutex
	built      bool
	limits     Limits
	coverage   int
	profile    decimal.Profile
	clock      func() time.Time
	now        time.Time
	workspaces map[string]*workspace
}

func NewRegistry(limits Limits, requiredCoverage int, profile decimal.Profile, clock func() time.Time) (*Registry, error) {
	if !limits.valid() || requiredCoverage < 0 || requiredCoverage > maxCoverage || !profile.Valid() || clock == nil {
		return nil, ErrInvalid
	}
	return &Registry{built: true, limits: limits, coverage: requiredCoverage, profile: profile, clock: clock, workspaces: map[string]*workspace{}}, nil
}

func (r *Registry) enter() bool {
	if r == nil {
		return false
	}
	r.mu.Lock()
	if !r.built {
		return false
	}
	if read := r.clock().UTC().Round(0); read.After(r.now) {
		r.now = read
	}
	return true
}

func (r *Registry) leave() {
	if r != nil {
		r.mu.Unlock()
	}
}

func join(fields ...string) string {
	var out strings.Builder
	for _, field := range fields {
		out.WriteString(strconv.Itoa(len(field)))
		out.WriteByte(':')
		out.WriteString(field)
	}
	return out.String()
}

func (r *Registry) period(from, to time.Time) bool {
	from, to = from.Round(0), to.Round(0)
	return from.Nanosecond() == 0 && to.Nanosecond() == 0 && from.Before(to) && to.Sub(from) <= r.limits.MaxPeriod
}

func (r *Registry) shape(s Spec) (*reception, bool) {
	for _, id := range []string{s.WorkspaceID, s.ID, s.Source, s.Statement, s.CostOrigin, s.Schema, s.Connection} {
		if !validID(id) {
			return nil, false
		}
	}
	if !r.period(s.PeriodStart, s.PeriodEnd) || s.Revision < 1 || s.Revision > maxRevision || s.Coverage < 0 || s.Coverage > maxCoverage || s.Lines < 1 || len(s.Totals) == 0 {
		return nil, false
	}
	if (s.Supersedes == "") != (s.CorrectionReason == "") || s.Supersedes == s.Statement || s.Mock && s.Supersedes != "" {
		return nil, false
	}
	if s.Supersedes != "" && (!validID(s.Supersedes) || !validID(s.CorrectionReason)) {
		return nil, false
	}
	if (s.Status != Final && s.Status != Provisional) || (s.Tier != Measured && s.Tier != Estimated && s.Tier != Unavailable) || (s.Declaration != SourceHeader && s.Declaration != Derived) {
		return nil, false
	}
	made := &reception{spec: s, declared: map[string]*number{}, currencies: map[string]bool{}}
	made.spec.PeriodStart, made.spec.PeriodEnd = s.PeriodStart.UTC(), s.PeriodEnd.UTC()
	made.spec.Totals = map[string]string{}
	currencies := make([]string, 0, len(s.Totals))
	for currency := range s.Totals {
		currencies = append(currencies, currency)
	}
	sort.Strings(currencies)
	prints := []string{s.ID, s.Source, s.Statement, s.CostOrigin, made.spec.PeriodStart.Format(time.RFC3339), made.spec.PeriodEnd.Format(time.RFC3339), strconv.Itoa(s.Revision), s.Supersedes, s.CorrectionReason,
		strconv.FormatBool(s.Mock), string(s.Status), string(s.Tier), strconv.Itoa(s.Coverage), strconv.Itoa(s.Lines), string(s.Declaration)}
	for _, currency := range currencies {
		total, parsed := parse(r.profile, s.Totals[currency])
		if !validCurrency(currency) || !parsed || total.sign() < 0 {
			return nil, false
		}
		made.declared[currency], made.currencies[currency], made.spec.Totals[currency] = total, true, total.text()
		prints = append(prints, currency, total.text())
	}
	made.print = join(prints...)
	return made, true
}

func (r *Registry) find(workspaceID, id string) *reception {
	if ws := r.workspaces[workspaceID]; ws != nil {
		return ws.receptions[id]
	}
	return nil
}

func (r *Registry) Open(spec Spec) (State, string, error) {
	defer r.leave()
	if !r.enter() {
		return "", "", ErrInvalid
	}
	made, formed := r.shape(spec)
	if !formed {
		return "", "", ErrInvalid
	}
	if spec.Lines > r.limits.Lines || len(spec.Totals) > r.limits.Currencies {
		return "", "", ErrLimit
	}
	if held := r.find(spec.WorkspaceID, spec.ID); held != nil {
		if held.print != made.print {
			return "", "", ErrConflict
		}
		return held.state, "", nil
	}
	ws := r.workspaces[spec.WorkspaceID]
	var origin *source
	var known *statement
	if ws != nil {
		if origin = ws.sources[spec.Source]; origin != nil {
			known = origin.statements[spec.Statement]
		}
	}
	if known != nil && known.live != nil {
		return "", known.live.spec.ID, ErrConflict
	}
	var named *statement
	switch {
	case known != nil && known.supersedes != "":
		if spec.Supersedes != known.supersedes {
			return "", "", ErrRejected
		}
	case spec.Supersedes != "":
		if origin != nil {
			named = origin.statements[spec.Supersedes]
		}
		if named == nil || named.supersededBy != "" {
			return "", "", ErrRejected
		}
		for at := known; at != nil && at.supersededBy != ""; at = origin.statements[at.supersededBy] {
			if at.supersededBy == spec.Supersedes {
				return "", "", ErrRejected
			}
		}
	}
	if ws != nil && len(ws.receptions) >= r.limits.Receptions {
		return "", "", ErrLimit
	}
	if ws == nil {
		ws = &workspace{receptions: map[string]*reception{}, sources: map[string]*source{}}
		r.workspaces[spec.WorkspaceID] = ws
	}
	if origin == nil {
		origin = &source{statements: map[string]*statement{}}
		ws.sources[spec.Source] = origin
	}
	if known == nil {
		known = &statement{}
		origin.statements[spec.Statement] = known
	}
	if named != nil {
		known.supersedes, named.supersededBy = spec.Supersedes, spec.Statement
	}
	made.origin, made.statement, made.state, made.lines = origin, known, Open, map[string]*posted{}
	made.opened, made.changed = r.now, r.now
	known.live = made
	ws.receptions[spec.ID] = made
	origin.receptions = append(origin.receptions, made)
	origin.sequence++
	return Open, "", nil
}

func (r *Registry) Post(workspaceID, id string, stated Line) (State, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(id) || !validID(stated.Team) || !validCurrency(stated.Currency) {
		return "", ErrInvalid
	}
	cost, parsed := parse(r.profile, stated.Cost)
	if !parsed || cost.sign() < 0 {
		return "", ErrInvalid
	}
	energy := ""
	if stated.Energy != "" {
		amount, measured := parse(r.profile, stated.Energy)
		if !measured || amount.sign() < 0 {
			return "", ErrInvalid
		}
		energy = amount.text()
	}
	held := r.find(workspaceID, id)
	if held == nil {
		return "", ErrNotFound
	}
	print := join(stated.Team, cost.text(), stated.Currency, strconv.FormatBool(stated.Measured), energy)
	if earlier := held.lines[stated.Team]; earlier != nil {
		if earlier.print != print {
			return "", ErrConflict
		}
		return held.state, nil
	}
	if held.state != Open || held.statement.supersededBy != "" || len(held.lines) >= held.spec.Lines {
		return "", ErrRejected
	}
	if !held.currencies[stated.Currency] && len(held.currencies) >= r.limits.Currencies {
		return "", ErrRejected
	}
	stated.Cost, stated.Energy = cost.text(), energy
	held.lines[stated.Team] = &posted{stated: stated, print: print, cost: cost}
	held.currencies[stated.Currency] = true
	held.origin.sequence++
	return held.state, nil
}

func (r *Registry) Close(workspaceID, id string) (State, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(id) {
		return "", ErrInvalid
	}
	held := r.find(workspaceID, id)
	if held == nil {
		return "", ErrNotFound
	}
	if held.wasClosed {
		return held.state, nil
	}
	if held.state != Open || held.statement.supersededBy != "" {
		return "", ErrRejected
	}
	held.sums = map[string]*number{}
	for team, line := range held.lines {
		held.teams = append(held.teams, team)
		if held.sums[line.stated.Currency] == nil {
			held.sums[line.stated.Currency] = zero()
		}
		held.sums[line.stated.Currency].add(line.cost, 1)
		if !line.stated.Measured {
			held.unmeasured = true
		}
	}
	sort.Strings(held.teams)
	held.state, held.wasClosed, held.changed = Closed, true, r.now
	held.origin.sequence++
	return held.state, nil
}

func (r *Registry) Void(workspaceID, id string, reason VoidReason) error {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(id) || !reason.known() {
		return ErrInvalid
	}
	held := r.find(workspaceID, id)
	if held == nil {
		return ErrNotFound
	}
	if held.state == Voided {
		if held.reason != reason {
			return ErrConflict
		}
		return nil
	}
	if !held.wasClosed {
		for team := range held.lines {
			held.teams = append(held.teams, team)
		}
		sort.Strings(held.teams)
	}
	held.state, held.reason, held.changed = Voided, reason, r.now
	held.statement.live = nil
	held.origin.sequence++
	return nil
}
