package tokenmeter

import (
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"
)

var amountFormed = map[string]bool{"1": true, "3": true, "1.50": true, "0.25": true, "-0.5": true, "2e1": true, "0.0000001": true, "60000000000000": true, "1234567890.123456": true, "1.": false}

func ratOf(text string) *big.Rat {
	value, read := new(big.Rat).SetString(text)
	if !read {
		panic(text)
	}
	return value
}

func ratDigits(text string) int {
	if cut := strings.IndexAny(text, "eE"); cut >= 0 {
		text = text[:cut]
	}
	text = strings.Trim(strings.NewReplacer("-", "", ".", "").Replace(text), "0")
	if text == "" {
		return 1
	}
	return len(text)
}

func ratFits(value *big.Rat) bool {
	bound := new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(14), nil))
	scaled := new(big.Rat).Mul(value, big.NewRat(1000000, 1))
	return new(big.Rat).Abs(value).Cmp(bound) < 0 && scaled.IsInt()
}

func ratText(value *big.Rat) string {
	text := value.FloatString(30)
	text = strings.TrimRight(strings.TrimRight(text, "0"), ".")
	if text == "-0" || text == "" {
		return "0"
	}
	return text
}

func countOf(text string) int64 {
	value, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		panic(text)
	}
	return value
}

func countsOf(e Event) map[Kind]string {
	return map[Kind]string{Input: e.Input, Output: e.Output, CacheRead: e.CacheRead, CacheWrite: e.CacheWrite}
}

var kindOrder = []Kind{Input, Output, CacheRead, CacheWrite}

func modelFormed(e Event) bool {
	if e.Outcome == "" {
		return false
	}
	for _, kind := range kindOrder {
		if countsOf(e)[kind] == "01" {
			return false
		}
	}
	if e.Cost == nil {
		return true
	}
	if !amountFormed[e.Cost.Amount] {
		return false
	}
	if e.Cost.Grade == Invoiced {
		return e.Cost.Origin == Provider && e.Cost.Document != ""
	}
	return e.Cost.Document == ""
}

func modelFindings(spec Spec, e Event) []string {
	var out []string
	if e.Time.Before(spec.PeriodStart) || !e.Time.Before(spec.PeriodEnd) {
		out = append(out, OutsidePeriod)
	}
	if e.Cache == Included && e.CacheRead != "" && (e.Input == "" || countOf(e.Input) < countOf(e.CacheRead)) {
		out = append(out, CacheRelationBroken)
	}
	if e.Cost != nil {
		if spec.Numeric == Float && ratDigits(e.Cost.Amount) > 15 {
			out = append(out, FloatPrecisionExceeded)
		}
		if !ratFits(ratOf(e.Cost.Amount)) {
			out = append(out, CostOutsideProfile)
		}
	}
	return out
}

func modelClass(e Event) string {
	switch {
	case e.Cost == nil:
		return ""
	case e.Cost.Grade == Estimated:
		return ClassEstimated
	case e.Cost.Origin == SelfHostedGPU:
		return ClassSelfHosted
	case e.Cost.Grade == Invoiced:
		return ClassInvoiceReferenced
	}
	return ClassUnconfirmed
}

type modelSums struct {
	figures *Figures
	perKind map[Kind]int64
	all     map[string]*big.Rat
	outside bool
}

func modelFigures(spec Spec, events []Event) modelSums {
	figures := &Figures{Outcomes: map[Outcome]int{}, Tokens: map[Usage]map[Kind]string{}, Unknown: map[Kind]int{}, Costs: map[string]map[string]string{}, Documents: map[string]DocumentSum{}}
	out := modelSums{figures: figures, perKind: map[Kind]int64{}, all: map[string]*big.Rat{}}
	tokens := map[string]int64{}
	costs, documents := map[string]*big.Rat{}, map[string]*big.Rat{}
	plus := func(sums map[string]*big.Rat, key string, amount *big.Rat) {
		if sums[key] == nil {
			sums[key] = new(big.Rat)
		}
		sums[key].Add(sums[key], amount)
	}
	var net int64
	for _, e := range events {
		found := modelFindings(spec, e)
		figures.Outcomes[e.Outcome]++
		if len(found) > 0 {
			figures.InError++
		}
		for _, kind := range kindOrder {
			text := countsOf(e)[kind]
			if text == "" {
				figures.Unknown[kind]++
				continue
			}
			if figures.Tokens[e.Usage] == nil {
				figures.Tokens[e.Usage] = map[Kind]string{}
			}
			tokens[string(e.Usage)+"/"+string(kind)] += countOf(text)
			figures.Tokens[e.Usage][kind] = strconv.FormatInt(tokens[string(e.Usage)+"/"+string(kind)], 10)
			out.perKind[kind] += countOf(text)
		}
		broken := strings.Contains(strings.Join(found, " "), CacheRelationBroken)
		switch {
		case !broken && e.Cache == Included && e.Input != "" && e.CacheRead != "":
			net += countOf(e.Input) - countOf(e.CacheRead)
		case !broken && e.Cache == Separate && e.Input != "":
			net += countOf(e.Input)
		default:
			figures.NetUnknown++
		}
		if e.Cost == nil || len(found) > 0 {
			continue
		}
		amount := ratOf(e.Cost.Amount)
		plus(costs, modelClass(e)+"/"+e.Cost.Currency, amount)
		plus(out.all, e.Cost.Currency, amount)
		if e.Cost.Document != "" {
			plus(documents, e.Cost.Document+"/"+e.Cost.Currency, amount)
			entry := figures.Documents[e.Cost.Document]
			entry.Events++
			figures.Documents[e.Cost.Document] = entry
		}
	}
	figures.NetInput = strconv.FormatInt(net, 10)
	for key, sum := range costs {
		parts := strings.Split(key, "/")
		if figures.Costs[parts[0]] == nil {
			figures.Costs[parts[0]] = map[string]string{}
		}
		figures.Costs[parts[0]][parts[1]] = ratText(sum)
		out.outside = out.outside || !ratFits(sum)
	}
	for key, sum := range documents {
		parts := strings.Split(key, "/")
		entry := figures.Documents[parts[0]]
		if entry.Sums == nil {
			entry.Sums = map[string]string{}
		}
		entry.Sums[parts[1]] = ratText(sum)
		out.outside = out.outside || !ratFits(sum)
		figures.Documents[parts[0]] = entry
	}
	return out
}

func (sums modelSums) differ(spec Spec) bool {
	if len(sums.all) != len(spec.Costs) {
		return true
	}
	for currency, sum := range sums.all {
		declared, stated := spec.Costs[currency]
		if !stated || ratOf(declared).Cmp(sum) != 0 {
			return true
		}
	}
	for kind, declared := range spec.Tokens {
		if strconv.FormatInt(sums.perKind[kind], 10) != declared {
			return true
		}
	}
	return false
}

type seen struct {
	id       string
	state    State
	from, to time.Time
	events   []Event
	readable bool
}

func common(a, b []Event) bool {
	held := map[string]bool{}
	for _, e := range a {
		held["e/"+e.ID], held["p/"+e.Request+"/"+e.Attempt] = true, true
	}
	for _, e := range b {
		if held["e/"+e.ID] || held["p/"+e.Request+"/"+e.Attempt] {
			return true
		}
	}
	return false
}

func modelVerdict(limits Limits, spec Spec, self seen, others []seen) (Verdict, bool) {
	out := Verdict{Result: NotEligible}
	if self.state == Voided {
		out.Reasons = []string{IsVoided}
		return out, true
	}
	note := func(applies bool, reason string) {
		if applies {
			out.Reasons = append(out.Reasons, reason)
		}
	}
	note(self.state == Open, NotClosed)
	note(spec.Produced == ListInterface, FromListInterface)
	note(spec.Declaration == Derived, DeclarationDerived)
	imprecise := false
	for _, total := range spec.Costs {
		imprecise = imprecise || (spec.Numeric == Float && ratDigits(total) > 15)
	}
	note(imprecise, FloatTotalImprecise)
	known := true
	var over, sharing []string
	for _, other := range others {
		if other.id == self.id || other.state == Voided {
			continue
		}
		if self.from.Before(other.to) && other.from.Before(self.to) {
			over = append(over, other.id)
		}
		if !self.readable || !other.readable {
			known = false
			continue
		}
		if common(self.events, other.events) {
			sharing = append(sharing, other.id)
		}
	}
	list := func(items []string) Listed {
		sort.Strings(items)
		total := len(items)
		if total > limits.Listed {
			items = items[:limits.Listed]
		}
		return Listed{Total: total, Items: items}
	}
	out.Conflicts, out.Shared = list(over), list(sharing)
	note(len(over) > 0, PeriodConflict)
	note(len(sharing) > 0, EventShared)
	if self.state == Closed {
		sums := modelFigures(spec, self.events)
		note(sums.figures.InError > 0, EventsInError)
		note(len(self.events) != spec.Events, Incomplete)
		note(len(spec.Tokens) != 4, TotalsNotStated)
		note(sums.differ(spec), TotalsDiffer)
		note(sums.outside, SumOutsideProfile)
	}
	if len(out.Reasons) == 0 {
		out.Result = Eligible
	}
	return out, known
}

func withoutSharing(out Verdict) string {
	var kept []string
	for _, reason := range out.Reasons {
		if reason != EventShared {
			kept = append(kept, reason)
		}
	}
	out.Reasons, out.Shared, out.Result = kept, Listed{}, ""
	return show(out)
}

type modelHeld struct {
	spec      Spec
	state     State
	wasClosed bool
	reason    VoidReason
	events    map[string]Event
	pairs     map[string]bool
	plan      []Event
}

func (held *modelHeld) list() []Event {
	ids := make([]string, 0, len(held.events))
	for id := range held.events {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]Event, 0, len(ids))
	for _, id := range ids {
		out = append(out, held.events[id])
	}
	return out
}

func (held *modelHeld) seen(readable bool) seen {
	return seen{id: held.spec.ID, state: held.state, from: held.spec.PeriodStart, to: held.spec.PeriodEnd, events: held.list(), readable: readable}
}

type world struct {
	limits   Limits
	held     map[string]map[string]*modelHeld
	order    map[string][]string
	sequence map[string]uint64
}

func newWorld(limits Limits) *world {
	return &world{limits: limits, held: map[string]map[string]*modelHeld{}, order: map[string][]string{}, sequence: map[string]uint64{}}
}

func specKey(s Spec) string {
	return fmt.Sprintf("%s|%s|%s|%d|%d|%d|%s|%s|%d|%s|%v|%v", s.ID, s.Source, s.Snapshot, s.PeriodStart.Unix(), s.PeriodEnd.Unix(), s.Revision, s.Produced, s.Numeric, s.Events, s.Declaration, s.Tokens, s.Costs)
}

func eventKey(e Event) string {
	cost := ""
	if e.Cost != nil {
		cost = fmt.Sprintf("%+v", *e.Cost)
	}
	at := e.Time.Unix()
	e.Cost, e.Time = nil, time.Time{}
	return fmt.Sprintf("%+v|%d|%s", e, at, cost)
}

func (w *world) find(ws, id string) *modelHeld { return w.held[ws][id] }

func (w *world) others(ws string, self *modelHeld) []seen {
	var out []seen
	for _, id := range w.order[ws] {
		if other := w.held[ws][id]; other.spec.Source == self.spec.Source {
			out = append(out, other.seen(true))
		}
	}
	return out
}

func (w *world) verdict(ws string, held *modelHeld) Verdict {
	out, _ := modelVerdict(w.limits, held.spec, held.seen(true), w.others(ws, held))
	return out
}

func (w *world) open(spec Spec) (State, string, error) {
	if spec.Revision < 1 || spec.Events < 1 || !spec.PeriodStart.Before(spec.PeriodEnd) || spec.PeriodEnd.Sub(spec.PeriodStart) > w.limits.MaxPeriod {
		return "", "", ErrInvalid
	}
	for _, total := range spec.Costs {
		if !amountFormed[total] {
			if _, read := new(big.Rat).SetString(total); !read || strings.HasSuffix(total, ".") {
				return "", "", ErrInvalid
			}
		}
	}
	if spec.Events > w.limits.Events || len(spec.Costs) > w.limits.Currencies {
		return "", "", ErrLimit
	}
	ws := spec.WorkspaceID
	if held := w.find(ws, spec.ID); held != nil {
		if specKey(held.spec) != specKey(spec) {
			return "", "", ErrConflict
		}
		return held.state, "", nil
	}
	for _, id := range w.order[ws] {
		if other := w.held[ws][id]; other.spec.Source == spec.Source && other.spec.Snapshot == spec.Snapshot && other.state != Voided {
			return "", id, ErrConflict
		}
	}
	if len(w.order[ws]) >= w.limits.Receptions {
		return "", "", ErrLimit
	}
	if w.held[ws] == nil {
		w.held[ws] = map[string]*modelHeld{}
	}
	w.held[ws][spec.ID] = &modelHeld{spec: spec, state: Open, events: map[string]Event{}, pairs: map[string]bool{}}
	w.order[ws] = append(w.order[ws], spec.ID)
	w.sequence[ws+"/"+spec.Source]++
	return Open, "", nil
}

func (w *world) post(ws, id string, events []Event) (State, error) {
	held := w.find(ws, id)
	if held == nil {
		return "", ErrNotFound
	}
	if len(events) == 0 {
		return "", ErrInvalid
	}
	ids, pairs := map[string]bool{}, map[string]bool{}
	for _, e := range events {
		pair := e.Request + "/" + e.Attempt
		if !modelFormed(e) || ids[e.ID] || pairs[pair] {
			return "", ErrInvalid
		}
		ids[e.ID], pairs[pair] = true, true
	}
	if len(events) > w.limits.Batch {
		return "", ErrLimit
	}
	var fresh []Event
	for _, e := range events {
		earlier, stored := held.events[e.ID]
		switch {
		case stored && eventKey(earlier) != eventKey(e):
			return "", ErrConflict
		case stored:
		case held.pairs[e.Request+"/"+e.Attempt]:
			return "", ErrConflict
		default:
			fresh = append(fresh, e)
		}
	}
	if len(fresh) == 0 {
		return held.state, nil
	}
	if held.state != Open || len(held.events)+len(fresh) > held.spec.Events {
		return "", ErrRejected
	}
	currencies, documents := map[string]bool{}, map[string]bool{}
	for currency := range held.spec.Costs {
		currencies[currency] = true
	}
	note := func(e Event) {
		if e.Cost != nil {
			currencies[e.Cost.Currency] = true
			if e.Cost.Document != "" {
				documents[e.Cost.Document] = true
			}
		}
	}
	for _, e := range held.events {
		note(e)
	}
	for _, e := range fresh {
		if e.Cost != nil && e.Cost.Origin == TokenMeterFee {
			return "", ErrRejected
		}
		note(e)
	}
	if len(currencies) > w.limits.Currencies || len(documents) > w.limits.Documents {
		return "", ErrRejected
	}
	for _, e := range fresh {
		held.events[e.ID], held.pairs[e.Request+"/"+e.Attempt] = e, true
	}
	w.sequence[ws+"/"+held.spec.Source]++
	return held.state, nil
}

func (w *world) close(ws, id string) (State, error) {
	held := w.find(ws, id)
	switch {
	case held == nil:
		return "", ErrNotFound
	case held.wasClosed:
		return held.state, nil
	case held.state != Open:
		return "", ErrRejected
	}
	held.state, held.wasClosed = Closed, true
	w.sequence[ws+"/"+held.spec.Source]++
	return Closed, nil
}

func (w *world) void(ws, id string, reason VoidReason) error {
	if reason != WithdrawnAtSource && reason != RegeneratedAtSource && reason != WrongReport && reason != ByOperator {
		return ErrInvalid
	}
	held := w.find(ws, id)
	switch {
	case held == nil:
		return ErrNotFound
	case held.state == Voided && held.reason != reason:
		return ErrConflict
	case held.state == Voided:
		return nil
	}
	held.state, held.reason = Voided, reason
	w.sequence[ws+"/"+held.spec.Source]++
	return nil
}
