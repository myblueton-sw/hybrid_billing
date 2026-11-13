package tokenmeter

import (
	"math/big"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

type stored struct {
	event    Event
	print    string
	findings []string
	class    string
	cost     *number
	counts   map[Kind]*big.Int
}

type source struct {
	exports    map[string]*reception
	receptions []*reception
	holders    map[string][]*reception
	sequence   uint64
}

type reception struct {
	spec       Spec
	print      string
	origin     *source
	tokens     map[Kind]*big.Int
	costs      map[string]*number
	imprecise  bool
	state      State
	wasClosed  bool
	events     map[string]*stored
	pairs      map[string]string
	order      []string
	currencies map[string]bool
	documents  map[string]bool
	shared     map[*reception]int
	figures    *Figures
	differ     bool
	outside    bool
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
	profile    decimal.Profile
	clock      func() time.Time
	now        time.Time
	workspaces map[string]*workspace
}

func NewRegistry(limits Limits, profile decimal.Profile, clock func() time.Time) (*Registry, error) {
	if !limits.valid() || !profile.Valid() || clock == nil {
		return nil, ErrInvalid
	}
	return &Registry{built: true, limits: limits, profile: profile, clock: clock, workspaces: map[string]*workspace{}}, nil
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
	for _, id := range []string{s.WorkspaceID, s.ID, s.Source, s.Snapshot, s.Schema, s.Connection} {
		if !validID(id) {
			return nil, false
		}
	}
	if !r.period(s.PeriodStart, s.PeriodEnd) || s.Revision < 1 || s.Revision > maxRevision || s.Unit != UnitToken || s.Events < 1 {
		return nil, false
	}
	if (s.Produced != SnapshotExport && s.Produced != ListInterface) || (s.Numeric != Decimal && s.Numeric != Float) || (s.Declaration != SourceManifest && s.Declaration != Derived) {
		return nil, false
	}
	made := &reception{spec: s, tokens: map[Kind]*big.Int{}, costs: map[string]*number{}, currencies: map[string]bool{}}
	made.spec.PeriodStart, made.spec.PeriodEnd = s.PeriodStart.UTC().Round(0), s.PeriodEnd.UTC().Round(0)
	made.spec.Tokens, made.spec.Costs = map[Kind]string{}, map[string]string{}
	prints := []string{s.ID, s.Source, s.Snapshot, made.spec.PeriodStart.Format(time.RFC3339), made.spec.PeriodEnd.Format(time.RFC3339), strconv.Itoa(s.Revision), string(s.Produced), string(s.Numeric), strconv.Itoa(s.Events), string(s.Declaration)}
	stated := 0
	for _, kind := range kinds {
		text, given := s.Tokens[kind]
		if !given {
			prints = append(prints, "")
			continue
		}
		total, formed := parseWhole(text, maxTotalDigits)
		if !formed {
			return nil, false
		}
		stated++
		made.tokens[kind], made.spec.Tokens[kind] = total, text
		prints = append(prints, "="+text)
	}
	if stated != len(s.Tokens) {
		return nil, false
	}
	currencies := make([]string, 0, len(s.Costs))
	for currency := range s.Costs {
		currencies = append(currencies, currency)
	}
	sort.Strings(currencies)
	for _, currency := range currencies {
		total, digits, formed := parseAmount(s.Costs[currency])
		if !validCurrency(currency) || !formed {
			return nil, false
		}
		if s.Numeric == Float && digits > floatDigits {
			made.imprecise = true
		}
		made.costs[currency], made.currencies[currency], made.spec.Costs[currency] = total, true, s.Costs[currency]
		prints = append(prints, currency, s.Costs[currency])
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
	if spec.Events > r.limits.Events || len(spec.Costs) > r.limits.Currencies {
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
	if ws != nil {
		if origin = ws.sources[spec.Source]; origin != nil && origin.exports[spec.Snapshot] != nil {
			return "", origin.exports[spec.Snapshot].spec.ID, ErrConflict
		}
		if len(ws.receptions) >= r.limits.Receptions {
			return "", "", ErrLimit
		}
	}
	if ws == nil {
		ws = &workspace{receptions: map[string]*reception{}, sources: map[string]*source{}}
		r.workspaces[spec.WorkspaceID] = ws
	}
	if origin == nil {
		origin = &source{exports: map[string]*reception{}, holders: map[string][]*reception{}}
		ws.sources[spec.Source] = origin
	}
	made.origin, made.state = origin, Open
	made.events, made.pairs, made.documents, made.shared = map[string]*stored{}, map[string]string{}, map[string]bool{}, map[*reception]int{}
	made.opened, made.changed = r.now, r.now
	origin.exports[spec.Snapshot] = made
	origin.receptions = append(origin.receptions, made)
	ws.receptions[spec.ID] = made
	origin.sequence++
	return Open, "", nil
}

func (r *Registry) form(event Event) (*stored, bool) {
	for _, id := range []string{event.ID, event.Request, event.Attempt, event.Consumer, event.Provider, event.Model} {
		if !validID(id) {
			return nil, false
		}
	}
	if (event.Version != "" && !validID(event.Version)) || (event.Endpoint != "" && !validID(event.Endpoint)) {
		return nil, false
	}
	at := event.Time.Round(0)
	if at.IsZero() || at.Nanosecond() != 0 || !event.Outcome.known() || !event.Usage.known() || !event.Cache.known() {
		return nil, false
	}
	made := &stored{event: event, counts: map[Kind]*big.Int{}}
	made.event.Time = at.UTC()
	prints := []string{event.ID, event.Request, event.Attempt, strconv.FormatInt(at.Unix(), 10), event.Consumer, event.Provider, event.Model, event.Version, event.Endpoint, string(event.Outcome), string(event.Usage), string(event.Cache)}
	for index, text := range []string{event.Input, event.Output, event.CacheRead, event.CacheWrite} {
		prints = append(prints, "="+text)
		if text == "" {
			continue
		}
		count, formed := parseWhole(text, maxCountDigits)
		if !formed {
			return nil, false
		}
		made.counts[kinds[index]] = count
	}
	if event.Cost != nil {
		cost := *event.Cost
		amount, _, formed := parseAmount(cost.Amount)
		if !formed || !validCurrency(cost.Currency) || !cost.Grade.known() || !cost.Origin.known() {
			return nil, false
		}
		if cost.Grade == Invoiced && (cost.Origin != Provider || !validID(cost.Document)) || cost.Grade != Invoiced && cost.Document != "" {
			return nil, false
		}
		made.event.Cost, made.cost = &cost, amount
		switch {
		case cost.Grade == Estimated:
			made.class = ClassEstimated
		case cost.Origin == SelfHostedGPU:
			made.class = ClassSelfHosted
		case cost.Grade == Invoiced:
			made.class = ClassInvoiceReferenced
		default:
			made.class = ClassUnconfirmed
		}
		prints = append(prints, "cost", cost.Amount, cost.Currency, string(cost.Grade), string(cost.Origin), cost.Document)
	}
	made.print = join(prints...)
	return made, true
}

func (r *Registry) examine(held *reception, made *stored) {
	if made.event.Time.Before(held.spec.PeriodStart) || !made.event.Time.Before(held.spec.PeriodEnd) {
		made.findings = append(made.findings, OutsidePeriod)
	}
	if cached := made.counts[CacheRead]; made.event.Cache == Included && cached != nil && (made.counts[Input] == nil || made.counts[Input].Cmp(cached) < 0) {
		made.findings = append(made.findings, CacheRelationBroken)
	}
	if made.cost != nil {
		if _, digits, _ := parseAmount(made.event.Cost.Amount); held.spec.Numeric == Float && digits > floatDigits {
			made.findings = append(made.findings, FloatPrecisionExceeded)
		}
		if !made.cost.fits(r.profile) {
			made.findings = append(made.findings, CostOutsideProfile)
		}
	}
}

func identities(event Event) []string {
	return []string{join("event", event.ID), join("pair", event.Request, event.Attempt)}
}

func (r *Registry) Post(workspaceID, id string, events []Event) (State, error) {
	defer r.leave()
	if !r.enter() || !validID(workspaceID) || !validID(id) {
		return "", ErrInvalid
	}
	held := r.find(workspaceID, id)
	if held == nil {
		return "", ErrNotFound
	}
	if len(events) == 0 {
		return "", ErrInvalid
	}
	batch := make([]*stored, len(events))
	ids, pairs := map[string]bool{}, map[string]bool{}
	for index, event := range events {
		made, formed := r.form(event)
		pair := join(event.Request, event.Attempt)
		if !formed || ids[event.ID] || pairs[pair] {
			return "", ErrInvalid
		}
		ids[event.ID], pairs[pair], batch[index] = true, true, made
	}
	if len(events) > r.limits.Batch {
		return "", ErrLimit
	}
	var fresh []*stored
	for _, made := range batch {
		earlier := held.events[made.event.ID]
		switch {
		case earlier != nil && earlier.print != made.print:
			return "", ErrConflict
		case earlier != nil:
		case held.pairs[join(made.event.Request, made.event.Attempt)] != "":
			return "", ErrConflict
		default:
			fresh = append(fresh, made)
		}
	}
	if len(fresh) == 0 {
		return held.state, nil
	}
	if held.state != Open || len(held.events)+len(fresh) > held.spec.Events {
		return "", ErrRejected
	}
	currencies, documents := len(held.currencies), len(held.documents)
	seen := map[string]bool{}
	for _, made := range fresh {
		cost := made.event.Cost
		if cost == nil {
			continue
		}
		if cost.Origin == TokenMeterFee {
			return "", ErrRejected
		}
		if !held.currencies[cost.Currency] && !seen["c"+cost.Currency] {
			seen["c"+cost.Currency] = true
			currencies++
		}
		if cost.Document != "" && !held.documents[cost.Document] && !seen["d"+cost.Document] {
			seen["d"+cost.Document] = true
			documents++
		}
	}
	if currencies > r.limits.Currencies || documents > r.limits.Documents {
		return "", ErrRejected
	}
	for _, made := range fresh {
		r.examine(held, made)
		held.events[made.event.ID] = made
		held.pairs[join(made.event.Request, made.event.Attempt)] = made.event.ID
		if cost := made.event.Cost; cost != nil {
			held.currencies[cost.Currency] = true
			if cost.Document != "" {
				held.documents[cost.Document] = true
			}
		}
		for _, identity := range identities(made.event) {
			for _, other := range held.origin.holders[identity] {
				held.shared[other]++
				other.shared[held]++
			}
			held.origin.holders[identity] = append(held.origin.holders[identity], held)
		}
	}
	held.origin.sequence++
	return held.state, nil
}

func (held *reception) arrange() {
	held.order = make([]string, 0, len(held.events))
	for id := range held.events {
		held.order = append(held.order, id)
	}
	sort.Strings(held.order)
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
	if held.state != Open {
		return "", ErrRejected
	}
	held.arrange()
	r.summarise(held)
	held.state, held.wasClosed, held.changed = Closed, true, r.now
	held.origin.sequence++
	return held.state, nil
}

func (r *Registry) summarise(held *reception) {
	figures := &Figures{Outcomes: map[Outcome]int{}, Tokens: map[Usage]map[Kind]string{}, Unknown: map[Kind]int{}, Costs: map[string]map[string]string{}, Documents: map[string]DocumentSum{}}
	tokens := map[Usage]map[Kind]*big.Int{}
	perKind := map[Kind]*big.Int{}
	costs := map[string]map[string]*number{}
	all := map[string]*number{}
	documents := map[string]map[string]*number{}
	net := new(big.Int)
	plus := func(sums map[string]*number, currency string, amount *number) {
		if sums[currency] == nil {
			sums[currency] = zero()
		}
		sums[currency].add(amount)
	}
	for _, made := range held.events {
		figures.Outcomes[made.event.Outcome]++
		broken := false
		for _, finding := range made.findings {
			broken = broken || finding == CacheRelationBroken
		}
		if len(made.findings) > 0 {
			figures.InError++
		}
		for _, kind := range kinds {
			count := made.counts[kind]
			if count == nil {
				figures.Unknown[kind]++
				continue
			}
			if tokens[made.event.Usage] == nil {
				tokens[made.event.Usage] = map[Kind]*big.Int{}
			}
			if tokens[made.event.Usage][kind] == nil {
				tokens[made.event.Usage][kind] = new(big.Int)
			}
			if perKind[kind] == nil {
				perKind[kind] = new(big.Int)
			}
			tokens[made.event.Usage][kind].Add(tokens[made.event.Usage][kind], count)
			perKind[kind].Add(perKind[kind], count)
		}
		input, cached := made.counts[Input], made.counts[CacheRead]
		switch {
		case broken:
			figures.NetUnknown++
		case made.event.Cache == Included && input != nil && cached != nil:
			net.Add(net, input).Sub(net, cached)
		case made.event.Cache == Separate && input != nil:
			net.Add(net, input)
		default:
			figures.NetUnknown++
		}
		if made.cost == nil || len(made.findings) > 0 {
			continue
		}
		cost := made.event.Cost
		if costs[made.class] == nil {
			costs[made.class] = map[string]*number{}
		}
		plus(costs[made.class], cost.Currency, made.cost)
		plus(all, cost.Currency, made.cost)
		if cost.Document != "" {
			if documents[cost.Document] == nil {
				documents[cost.Document] = map[string]*number{}
			}
			plus(documents[cost.Document], cost.Currency, made.cost)
			entry := figures.Documents[cost.Document]
			entry.Events++
			figures.Documents[cost.Document] = entry
		}
	}
	figures.NetInput = net.String()
	for usage, byKind := range tokens {
		figures.Tokens[usage] = map[Kind]string{}
		for kind, sum := range byKind {
			figures.Tokens[usage][kind] = sum.String()
		}
	}
	for class, byCurrency := range costs {
		figures.Costs[class] = map[string]string{}
		for currency, sum := range byCurrency {
			figures.Costs[class][currency] = sum.text()
			if !sum.fits(r.profile) {
				held.outside = true
			}
		}
	}
	for document, byCurrency := range documents {
		entry := figures.Documents[document]
		entry.Sums = map[string]string{}
		for currency, sum := range byCurrency {
			entry.Sums[currency] = sum.text()
			if !sum.fits(r.profile) {
				held.outside = true
			}
		}
		figures.Documents[document] = entry
	}
	held.differ = len(all) != len(held.costs)
	for currency, sum := range all {
		if declared := held.costs[currency]; declared == nil || !declared.equals(sum) {
			held.differ = true
		}
	}
	for kind, declared := range held.tokens {
		sum := perKind[kind]
		if sum == nil {
			sum = new(big.Int)
		}
		if sum.Cmp(declared) != 0 {
			held.differ = true
		}
	}
	held.figures = figures
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
		held.arrange()
	}
	held.state, held.reason, held.changed = Voided, reason, r.now
	delete(held.origin.exports, held.spec.Snapshot)
	held.origin.sequence++
	return nil
}
