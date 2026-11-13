package tokenmeter

import (
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type span struct{ from, to time.Time }

func spans() []span {
	september, october := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	november, december := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	return []span{{september, october}, {october, november}, {november, december}, {october, october.Add(15 * day)}, {october.Add(10 * day), november.Add(4 * day)}}
}

var (
	workspaces = []string{"ws1", "ws2"}
	sourceIDs  = []string{"tm-1", "tm-2"}
	errorKinds = []error{ErrInvalid, ErrNotFound, ErrConflict, ErrRejected, ErrLimit}
)

type generator struct {
	t       *testing.T
	rnd     *rand.Rand
	r       *Registry
	w       *world
	limits  Limits
	serial  int
	reached map[string]int
	counted map[string]bool
}

func (g *generator) once(key, where string) {
	if !g.counted[key+"|"+where] {
		g.counted[key+"|"+where] = true
		g.reached[key]++
	}
}

func kindOf(err error) string {
	if err == nil {
		return "ok"
	}
	for _, kind := range errorKinds {
		if errors.Is(err, kind) {
			return kind.Error()[:12]
		}
	}
	return "other"
}

func (g *generator) pick(options ...string) string { return options[g.rnd.Intn(len(options))] }

func (g *generator) event(spec Spec, id string) Event {
	rnd := g.rnd
	length := int64(spec.PeriodEnd.Sub(spec.PeriodStart) / time.Second)
	made := Event{ID: id, Request: "req-" + id, Attempt: "1", Time: spec.PeriodStart.Add(time.Duration(rnd.Int63n(length)) * time.Second), Consumer: "user-" + g.pick("1", "2"), Provider: "prov-a", Model: g.pick("m1", "m2"),
		Version: g.pick("", "v1"), Endpoint: g.pick("", "chat"), Outcome: Outcome(g.pick("succeeded", "succeeded", "failed", "cancelled", "stream_incomplete")),
		Input: g.pick("", "0", "7", "100", "100", "100"), Output: g.pick("", "0", "7", "100"), CacheRead: g.pick("", "0", "7", "7"), CacheWrite: g.pick("", "0", "7"),
		Usage: Usage(g.pick("provider_reported", "provider_reported", "estimated", "not_known")), Cache: Relation(g.pick("included", "separate", "separate", "not_known"))}
	if rnd.Intn(14) == 0 {
		made.Time = spec.PeriodEnd
	}
	if rnd.Intn(12) == 0 {
		made.ID = "g" + strconv.Itoa(rnd.Intn(6))
	}
	if rnd.Intn(12) == 0 {
		made.Request = "q" + strconv.Itoa(rnd.Intn(4))
	}
	if rnd.Intn(60) == 0 {
		made.Outcome = ""
	}
	if rnd.Intn(60) == 0 {
		made.Output = "01"
	}
	if rnd.Intn(5) > 0 {
		cost := &Cost{Amount: g.pick("1", "1", "1", "3", "3", "1.50", "0.25", "0.25", "-0.5", "2e1", "0.0000001", "60000000000000", "1234567890.123456"), Currency: g.pick("USD", "USD", "USD", "USD", "EUR", "KRW", "JPY"),
			Grade: Grade(g.pick("invoiced", "invoiced", "billed", "computed", "estimated", "not_known")), Origin: Origin(g.pick("provider", "provider", "provider", "provider", "self_hosted_gpu"))}
		if cost.Grade == Invoiced {
			cost.Origin, cost.Document = Provider, "inv-"+strconv.Itoa(rnd.Intn(3))
			if rnd.Intn(15) == 0 {
				cost.Document = "inv-" + strconv.Itoa(3+rnd.Intn(4))
			}
		}
		switch rnd.Intn(80) {
		case 0:
			cost.Origin = TokenMeterFee
		case 1:
			cost.Amount = "1."
		case 2:
			cost.Document = map[bool]string{true: "", false: "inv-0"}[cost.Grade == Invoiced]
		}
		made.Cost = cost
	}
	return made
}

func (g *generator) spec(ws, id string) (Spec, []Event) {
	rnd := g.rnd
	period := spans()[rnd.Intn(len(spans()))]
	if rnd.Intn(3) > 0 {
		period = spans()[rnd.Intn(3)]
	}
	g.serial++
	spec := Spec{WorkspaceID: ws, ID: id, Source: sourceIDs[rnd.Intn(5)/4], Snapshot: "s" + strconv.Itoa(rnd.Intn(8)), Schema: g.pick("v1", "v2"), Connection: "conn", PeriodStart: period.from, PeriodEnd: period.to, Revision: 1,
		Produced: SnapshotExport, Numeric: Decimal, Unit: UnitToken, Declaration: SourceManifest}
	if rnd.Intn(12) == 0 {
		spec.Produced = ListInterface
	}
	if rnd.Intn(12) == 0 {
		spec.Declaration = Derived
	}
	if rnd.Intn(7) == 0 {
		spec.Numeric = Float
	}
	plan := make([]Event, 1+rnd.Intn(4))
	taken, pairs := map[string]bool{}, map[string]bool{}
	for index := range plan {
		for {
			plan[index] = g.event(spec, fmt.Sprintf("%s-%d-e%d", id, g.serial, index))
			if rnd.Intn(4) > 0 {
				plan[index].Outcome, plan[index].Output = Succeeded, "7"
				if cost := plan[index].Cost; cost != nil && rnd.Intn(3) > 0 {
					cost.Amount, cost.Currency = g.pick("1", "3", "0.25", "1.50"), "USD"
					if cost.Origin == TokenMeterFee {
						cost.Origin = Provider
					}
					if (cost.Grade == Invoiced) != (cost.Document != "") {
						cost.Grade, cost.Document = Billed, ""
					}
				}
				if rnd.Intn(3) > 0 {
					plan[index].Time, plan[index].Cache = spec.PeriodStart.Add(time.Hour), Separate
				}
			}
			pair := plan[index].Request + "/" + plan[index].Attempt
			if !taken[plan[index].ID] && !pairs[pair] {
				taken[plan[index].ID], pairs[pair] = true, true
				break
			}
		}
	}
	if len(plan) > 1 && rnd.Intn(12) == 0 {
		for index := 0; index < 2; index++ {
			plan[index].Outcome, plan[index].Output, plan[index].Time, plan[index].Cache = Succeeded, "7", spec.PeriodStart.Add(time.Hour), Separate
			plan[index].Cost = &Cost{Amount: "60000000000000", Currency: "USD", Grade: Billed, Origin: Provider}
		}
	}
	spec.Events = len(plan)
	var formed []Event
	for _, e := range plan {
		if modelFormed(e) && (e.Cost == nil || e.Cost.Origin != TokenMeterFee) {
			formed = append(formed, e)
		}
	}
	sums := modelFigures(spec, formed)
	spec.Tokens, spec.Costs = map[Kind]string{}, map[string]string{}
	for _, kind := range kindOrder {
		spec.Tokens[kind] = strconv.FormatInt(sums.perKind[kind], 10)
	}
	currencies := make([]string, 0, len(sums.all))
	for currency := range sums.all {
		currencies = append(currencies, currency)
	}
	sort.Strings(currencies)
	for _, currency := range currencies {
		spec.Costs[currency] = ratText(sums.all[currency])
	}
	switch rnd.Intn(28) {
	case 0:
		delete(spec.Tokens, kindOrder[rnd.Intn(4)])
	case 1:
		spec.Tokens[kindOrder[rnd.Intn(4)]] = "999"
	case 2:
		spec.Costs["USD"] = "77"
	case 3:
		spec.Costs = map[string]string{}
	case 4:
		spec.Events++
	case 5:
		spec.Revision = 0
	case 6:
		spec.PeriodStart, spec.PeriodEnd = spec.PeriodEnd, spec.PeriodStart
	case 7:
		spec.PeriodEnd = spec.PeriodStart.Add(41 * day)
	case 8:
		spec.Events = g.limits.Events + 1
	case 9:
		spec.Costs = map[string]string{"USD": "1", "EUR": "1", "KRW": "1", "JPY": "1"}
	case 10:
		spec.Costs["USD"] = "1."
	case 12, 13:
		if total, stated := spec.Costs["USD"]; stated {
			spec.Numeric = Float
			spec.Costs["USD"] = total + map[bool]string{true: "000000000000001", false: ".000000000000001"}[strings.Contains(total, ".")]
		}
	case 11:
		if len(currencies) > 0 {
			spec.Costs[currencies[0]] += "00"
			if !strings.Contains(spec.Costs[currencies[0]], ".") {
				spec.Costs[currencies[0]] = strings.TrimSuffix(spec.Costs[currencies[0]], "00") + ".00"
			}
		}
	}
	return spec, plan
}

func (g *generator) same(what string, got, want error) {
	g.t.Helper()
	if !errors.Is(got, want) || (want == nil && got != nil) {
		g.t.Fatalf("%s: the registry said %v, the model %v", what, got, want)
	}
	g.reached[strings.Fields(what)[0]+"/"+kindOf(want)]++
}

func (g *generator) open(spec Spec, plan []Event) {
	state, occupant, err := g.r.Open(spec)
	wantState, wantOccupant, wantErr := g.w.open(spec)
	g.same("open "+spec.ID, err, wantErr)
	if state != wantState || occupant != wantOccupant {
		g.t.Fatalf("open %s: %v %q, model %v %q", spec.ID, state, occupant, wantState, wantOccupant)
	}
	if held := g.w.find(spec.WorkspaceID, spec.ID); held != nil && held.plan == nil && err == nil {
		held.plan = plan
	}
	if occupant != "" {
		g.reached["occupant"]++
	}
}

func (g *generator) post(ws, id string, events []Event) {
	state, err := g.r.Post(ws, id, events)
	wantState, wantErr := g.w.post(ws, id, events)
	g.same("post "+id, err, wantErr)
	if state != wantState {
		g.t.Fatalf("post %s: %v, model %v", id, state, wantState)
	}
}

func (g *generator) batch(held *modelHeld) []Event {
	rnd := g.rnd
	var remaining, stored []Event
	for _, e := range held.plan {
		if _, kept := held.events[e.ID]; kept {
			stored = append(stored, e)
		} else {
			remaining = append(remaining, e)
		}
	}
	var batch []Event
	size := 1 + rnd.Intn(3)
	if rnd.Intn(30) == 0 {
		size = 4
	}
	for len(batch) < size && len(remaining) > 0 {
		batch = append(batch, remaining[0])
		remaining = remaining[1:]
	}
	if len(stored) > 0 && rnd.Intn(5) == 0 {
		batch = append(batch, stored[rnd.Intn(len(stored))])
	}
	if len(stored) > 0 && rnd.Intn(14) == 0 {
		altered := stored[rnd.Intn(len(stored))]
		switch rnd.Intn(3) {
		case 0:
			altered.Model = "m9"
		case 1:
			altered.ID = "other-" + altered.ID
		default:
			altered.Input = "5"
		}
		batch = append(batch, altered)
	}
	if rnd.Intn(14) == 0 || len(batch) == 0 {
		g.serial++
		batch = append(batch, g.event(held.spec, fmt.Sprintf("x-%d", g.serial)))
	}
	if rnd.Intn(40) == 0 {
		batch = append(batch, batch[0])
	}
	if rnd.Intn(60) == 0 {
		batch = nil
	}
	return batch
}

func (g *generator) step() {
	rnd := g.rnd
	ws := workspaces[rnd.Intn(6)/5]
	ids := g.w.order[ws]
	if len(ids) == 0 || rnd.Intn(4) == 0 {
		id := "r" + strconv.Itoa(rnd.Intn(14))
		spec, plan := g.spec(ws, id)
		if held := g.w.find(ws, id); held != nil && rnd.Intn(2) == 0 {
			spec, plan = held.spec, nil
		}
		g.open(spec, plan)
		return
	}
	if rnd.Intn(40) == 0 {
		switch rnd.Intn(3) {
		case 0:
			g.post(ws, "nope", []Event{g.event(g.w.find(ws, ids[0]).spec, "n1")})
		case 1:
			_, err := g.r.Close(ws, "nope")
			_, want := g.w.close(ws, "nope")
			g.same("close nope", err, want)
		default:
			g.same("void nope", g.r.Void(ws, "nope", ByOperator), g.w.void(ws, "nope", ByOperator))
		}
		return
	}
	held := g.w.find(ws, ids[rnd.Intn(len(ids))])
	id := held.spec.ID
	complete := true
	for _, e := range held.plan {
		_, kept := held.events[e.ID]
		complete = complete && kept
	}
	choice := rnd.Intn(100)
	closing, voiding := 8, 4
	switch {
	case held.state == Open && complete:
		closing, voiding = 70, 6
	case held.state == Closed:
		closing, voiding = 10, 8
	case held.state == Voided:
		closing, voiding = 20, 25
	}
	switch {
	case choice < closing:
		state, err := g.r.Close(ws, id)
		want, wantErr := g.w.close(ws, id)
		g.same("close "+id, err, wantErr)
		if state != want {
			g.t.Fatalf("close %s: %v, model %v", id, state, want)
		}
	case choice < closing+voiding:
		reason := VoidReason(g.pick("withdrawn_at_source", "regenerated_at_source", "wrong_report", "operator", "operator", "operator", ""))
		g.same("void "+id, g.r.Void(ws, id, reason), g.w.void(ws, id, reason))
	default:
		g.post(ws, id, g.batch(held))
	}
	got, err := g.r.Eligibility(ws, id)
	if err != nil || show(got) != show(g.w.verdict(ws, held)) {
		g.t.Fatalf("%s/%s: the registry judged %s (%v), the model %s", ws, id, show(got), err, show(g.w.verdict(ws, held)))
	}
}

func (g *generator) read(ws, id string) []EventView {
	var out []EventView
	for {
		page, err := g.r.Events(ws, id, len(out), g.limits.Page)
		if err != nil {
			g.t.Fatal(err)
		}
		if len(page) == 0 {
			return out
		}
		out = append(out, page...)
	}
}

func (g *generator) rows(ws, name string) []Row {
	var out []Row
	after := ""
	for {
		page, err := g.r.Receptions(ws, name, after, g.limits.Page)
		if err != nil {
			g.t.Fatal(err)
		}
		if len(page) == 0 {
			return out
		}
		out = append(out, page...)
		after = page[len(page)-1].ID
	}
}

func (g *generator) audit() {
	t := g.t
	for _, ws := range workspaces {
		if len(g.w.order[ws]) > g.limits.Receptions {
			t.Fatal("more receptions than the limit")
		}
		for _, name := range sourceIDs {
			rows := g.rows(ws, name)
			var want []string
			for _, id := range g.w.order[ws] {
				if g.w.held[ws][id].spec.Source == name {
					want = append(want, id)
				}
			}
			sort.Strings(want)
			read := map[string][]Event{}
			var seenRows []seen
			var listed []string
			for _, row := range rows {
				listed = append(listed, row.ID)
				made := seen{id: row.ID, state: row.State, from: row.PeriodStart, to: row.PeriodEnd, readable: row.State != Open}
				if made.readable {
					views := g.read(ws, row.ID)
					spec := g.w.held[ws][row.ID].spec
					for index, view := range views {
						if index > 0 && views[index-1].Event.ID >= view.Event.ID {
							t.Fatalf("%s: events out of order", row.ID)
						}
						if !reflect.DeepEqual(view.Findings, modelFindings(spec, view.Event)) || view.Class != modelClass(view.Event) {
							t.Fatalf("%s/%s: findings %v class %q", row.ID, view.Event.ID, view.Findings, view.Class)
						}
						for _, finding := range view.Findings {
							g.once("finding/"+finding, ws+"/"+row.ID+"/"+view.Event.ID)
						}
						made.events = append(made.events, view.Event)
					}
					read[row.ID] = made.events
				}
				seenRows = append(seenRows, made)
			}
			if !reflect.DeepEqual(listed, want) {
				t.Fatalf("%s/%s lists %v, the model %v", ws, name, listed, want)
			}
			var eligible []seen
			exact := map[span]string{}
			exports := map[string]string{}
			for _, row := range rows {
				if row.State == Voided {
					continue
				}
				if other := exports[row.Snapshot]; other != "" {
					t.Fatalf("%s and %s both hold the export %s", other, row.ID, row.Snapshot)
				}
				exports[row.Snapshot] = row.ID
			}
			for index, row := range seenRows {
				held := g.w.held[ws][row.id]
				view, err := g.r.Reception(ws, row.id)
				if err != nil || view.State != held.state || view.WasClosed != held.wasClosed || view.VoidReason != held.reason || view.Events != len(held.events) || view.Events > view.Spec.Events || specKey(view.Spec) != specKey(held.spec) {
					t.Fatalf("%s: %+v %v against %+v", row.id, view, err, held)
				}
				if (view.Figures != nil) != held.wasClosed {
					t.Fatalf("%s: figures exist exactly after a close", row.id)
				}
				if row.readable {
					var keys, modelKeys []string
					for _, e := range row.events {
						keys = append(keys, eventKey(e))
					}
					for _, e := range held.list() {
						modelKeys = append(modelKeys, eventKey(e))
					}
					if !reflect.DeepEqual(keys, modelKeys) {
						t.Fatalf("%s: events\n%v\n%v", row.id, keys, modelKeys)
					}
				}
				if held.wasClosed {
					sums := modelFigures(view.Spec, row.events)
					total := 0
					for _, count := range view.Figures.Outcomes {
						total += count
					}
					if flat(view.Figures) != flat(sums.figures) || total != view.Events || view.Figures.InError > view.Events || strings.HasPrefix(view.Figures.NetInput, "-") {
						t.Fatalf("%s: figures\n got %s\nfrom the events read %s", row.id, flat(view.Figures), flat(sums.figures))
					}
					for class := range view.Figures.Costs {
						g.once("class/"+class, ws+"/"+row.id)
					}
				}
				got, err := g.r.Eligibility(ws, row.id)
				if err != nil {
					t.Fatal(err)
				}
				model := g.w.verdict(ws, held)
				if show(got) != show(model) {
					t.Fatalf("%s/%s: the registry judged %s, the model %s", ws, row.id, show(got), show(model))
				}
				again, known := modelVerdict(g.limits, view.Spec, seenRows[index], seenRows)
				switch {
				case known && show(again) != show(got):
					t.Fatalf("%s/%s: the registry judged %s, its own reads give %s", ws, row.id, show(got), show(again))
				case !known && withoutSharing(again) != withoutSharing(got):
					t.Fatalf("%s/%s: the registry judged %s, its own reads give %s apart from sharing", ws, row.id, show(got), show(again))
				}
				g.reached[fmt.Sprintf("reread/%v", known)]++
				g.once("result/"+got.Result, ws+"/"+row.id)
				for _, reason := range got.Reasons {
					g.once("reason/"+reason, ws+"/"+row.id)
				}
				if len(got.Reasons) > 3 {
					g.once("reasons/many", ws+"/"+row.id)
				}
				if got.Conflicts.Total > g.limits.Listed || got.Shared.Total > g.limits.Listed {
					g.once("listed/cut", ws+"/"+row.id)
				}
				if got.Result == Eligible {
					for _, other := range eligible {
						if row.from.Before(other.to) && other.from.Before(row.to) || common(row.events, other.events) {
							t.Fatalf("%s and %s are both eligible", row.id, other.id)
						}
					}
					if row.state != Closed || len(row.events) != view.Spec.Events || view.Figures.InError != 0 {
						t.Fatalf("%s is eligible: %+v", row.id, view)
					}
					eligible = append(eligible, row)
					exact[span{row.from, row.to}] = row.id
				}
			}
			for _, period := range spans() {
				counted, err := g.r.Counting(ws, name, period.from, period.to)
				want := exact[period]
				if counted.Reception != want || (err == nil) != (want != "") || (err != nil && err != ErrNotFound) || counted.Sequence != g.w.sequence[ws+"/"+name] {
					t.Fatalf("%s/%s counting %+v %v, want %q at %d", ws, name, counted, err, want, g.w.sequence[ws+"/"+name])
				}
				if want != "" {
					g.once("counting", ws+"/"+want)
				}
			}
		}
	}
}

func TestRandomHistoriesAgainstAModelAndInvariants(t *testing.T) {
	reached := map[string]int{}
	limits := Limits{Receptions: 10, Events: 6, Batch: 3, Currencies: 3, Documents: 4, Page: 2, Listed: 2, MaxPeriod: 40 * day}
	for seed := int64(1); seed <= 260; seed++ {
		clock := &testClock{at: start}
		r, err := NewRegistry(limits, testProfile(t), clock.read)
		if err != nil {
			t.Fatal(err)
		}
		g := &generator{t: t, rnd: rand.New(rand.NewSource(seed)), r: r, w: newWorld(limits), limits: limits, reached: reached, counted: map[string]bool{}}
		for step := 0; step < 90; step++ {
			g.step()
			if step%15 == 14 {
				g.audit()
			}
			clock.advance(time.Second)
		}
		g.audit()
	}
	least := map[string]int{"result/eligible": 300, "counting": 300, "reread/true": 2000, "reread/false": 500, "occupant": 20, "reasons/many": 50, "listed/cut": 20,
		"open/ok": 1000, "post/ok": 2000, "close/ok": 1000, "void/ok": 300}
	for _, key := range []string{"open/" + kindOf(ErrInvalid), "open/" + kindOf(ErrLimit), "open/" + kindOf(ErrConflict), "post/" + kindOf(ErrInvalid), "post/" + kindOf(ErrNotFound), "post/" + kindOf(ErrLimit), "post/" + kindOf(ErrConflict),
		"post/" + kindOf(ErrRejected), "close/" + kindOf(ErrNotFound), "close/" + kindOf(ErrRejected), "void/" + kindOf(ErrInvalid), "void/" + kindOf(ErrNotFound), "void/" + kindOf(ErrConflict)} {
		least[key] = 20
	}
	for _, reason := range []string{IsVoided, NotClosed, FromListInterface, DeclarationDerived, FloatTotalImprecise, PeriodConflict, EventShared, EventsInError, Incomplete, TotalsNotStated, TotalsDiffer, SumOutsideProfile} {
		least["reason/"+reason] = 20
	}
	for _, finding := range []string{OutsidePeriod, CacheRelationBroken, FloatPrecisionExceeded, CostOutsideProfile} {
		least["finding/"+finding] = 20
	}
	for _, class := range []string{ClassEstimated, ClassSelfHosted, ClassInvoiceReferenced, ClassUnconfirmed} {
		least["class/"+class] = 20
	}
	keys := make([]string, 0, len(least))
	for key := range least {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if reached[key] < least[key] {
			t.Errorf("%s reached %d times, want at least %d", key, reached[key], least[key])
		}
	}
	if t.Failed() {
		t.Logf("%v", reached)
	}
}

func TestConcurrentUse(t *testing.T) {
	from, to := month(10)
	for round := 0; round < 30; round++ {
		r, err := NewRegistry(testLimits(), testProfile(t), func() time.Time { return start })
		if err != nil {
			t.Fatal(err)
		}
		const writers, batches, size = 6, 8, 5
		spec := export("r1", "snap-1", 10)
		spec.Events = writers * batches * size
		spec.Tokens = map[Kind]string{Input: strconv.Itoa(spec.Events * 100), Output: strconv.Itoa(spec.Events * 10), CacheRead: "0", CacheWrite: "0"}
		spec.Costs = map[string]string{"USD": strconv.Itoa(spec.Events)}
		if _, _, err := r.Open(spec); err != nil {
			t.Fatal(err)
		}
		begin := make(chan struct{})
		var group sync.WaitGroup
		accepted := make([]int, writers)
		closed := make([]bool, 3)
		for writer := 0; writer < writers; writer++ {
			group.Add(1)
			go func(writer int) {
				defer group.Done()
				<-begin
				for batch := 0; batch < batches; batch++ {
					events := make([]Event, size)
					for index := range events {
						events[index] = event(fmt.Sprintf("w%d-b%d-e%d", writer, batch, index), 10)
					}
					switch _, err := r.Post("ws1", "r1", events); err {
					case nil:
						accepted[writer] += size
					case ErrRejected:
					default:
						t.Errorf("post: %v", err)
					}
					own := export(fmt.Sprintf("own-%d-%d", writer, batch), fmt.Sprintf("own-%d-%d", writer, batch), 10)
					own.Source = fmt.Sprintf("own-tm-%d", writer)
					if _, _, err := r.Open(own); err != nil {
						t.Errorf("open: %v", err)
					}
				}
			}(writer)
		}
		for closer := range closed {
			group.Add(1)
			go func(closer int) {
				defer group.Done()
				<-begin
				for spin := 0; spin < round*40; spin++ {
					if _, err := r.Eligibility("ws1", "r1"); err != nil {
						t.Errorf("eligibility: %v", err)
					}
				}
				state, err := r.Close("ws1", "r1")
				closed[closer] = err == nil && state == Closed
			}(closer)
		}
		for reader := 0; reader < 3; reader++ {
			group.Add(1)
			go func() {
				defer group.Done()
				<-begin
				for spin := 0; spin < 60; spin++ {
					if _, err := r.Reception("ws1", "r1"); err != nil {
						t.Errorf("reception: %v", err)
					}
					if _, err := r.Counting("ws1", "tm-1", from, to); err != nil && err != ErrNotFound {
						t.Errorf("counting: %v", err)
					}
					if _, err := r.Receptions("ws1", "tm-1", "", 5); err != nil {
						t.Errorf("receptions: %v", err)
					}
					if _, err := r.Events("ws1", "r1", 0, 5); err != nil && err != ErrRejected {
						t.Errorf("events: %v", err)
					}
				}
			}()
		}
		close(begin)
		group.Wait()
		total := 0
		for _, count := range accepted {
			total += count
		}
		view, err := r.Reception("ws1", "r1")
		if err != nil || view.State != Closed || view.Events != total || !closed[0] || !closed[1] || !closed[2] {
			t.Fatalf("round %d: %+v %v, accepted %d, closed %v", round, view, err, total, closed)
		}
		input, _ := new(big.Int).SetString(view.Figures.Tokens[ProviderReported][Input], 10)
		if total > 0 && (input == nil || input.Int64() != int64(total)*100 || view.Figures.Costs[ClassInvoiceReferenced]["USD"] != strconv.Itoa(total) || view.Figures.Outcomes[Succeeded] != total) {
			t.Fatalf("round %d: figures %s for %d events", round, flat(view.Figures), total)
		}
		verdict, err := r.Eligibility("ws1", "r1")
		wantResult := map[bool]string{true: "eligible[]", false: "not_eligible[incomplete totals_differ]"}[total == spec.Events]
		if err != nil || show(verdict) != wantResult {
			t.Fatalf("round %d: %s with %d of %d events", round, show(verdict), total, spec.Events)
		}
		counted, err := r.Counting("ws1", "tm-1", from, to)
		if (err == nil) != (total == spec.Events) || counted.Sequence != uint64(2+total/size) {
			t.Fatalf("round %d: %+v %v after %d accepted batches", round, counted, err, total/size)
		}
	}
}

func TestConcurrentRepeatsExportsAndVoids(t *testing.T) {
	from, to := month(10)
	voided := 0
	defer func() {
		if voided < 10 && !t.Failed() {
			t.Errorf("the void was raced %d times", voided)
		}
	}()
	for round := 0; round < 40; round++ {
		r, err := NewRegistry(testLimits(), testProfile(t), func() time.Time { return start })
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err := r.Open(export("r1", "snap-1", 10)); err != nil {
			t.Fatal(err)
		}
		batch := []Event{event("e1", 10), event("e2", 10), event("e3", 10)}
		begin := make(chan struct{})
		var group sync.WaitGroup
		const racers = 8
		added := make([]bool, racers)
		winners := make([]string, racers)
		occupants := make([]string, racers)
		for racer := 0; racer < racers; racer++ {
			group.Add(2)
			go func(racer int) {
				defer group.Done()
				<-begin
				switch _, err := r.Post("ws1", "r1", batch); err {
				case nil:
					added[racer] = true
				case ErrRejected:
				default:
					t.Errorf("post: %v", err)
				}
			}(racer)
			go func(racer int) {
				defer group.Done()
				<-begin
				id := fmt.Sprintf("x%d", racer)
				switch _, occupant, err := r.Open(export(id, "snap-x", 11)); err {
				case nil:
					winners[racer] = id
				case ErrConflict:
					occupants[racer] = occupant
				default:
					t.Errorf("open: %v", err)
				}
			}(racer)
		}
		group.Add(1)
		go func() {
			defer group.Done()
			<-begin
			for spin := 0; spin < round*20; spin++ {
				if _, err := r.Reception("ws1", "r1"); err != nil {
					t.Errorf("reception: %v", err)
				}
			}
			if _, err := r.Close("ws1", "r1"); err != nil {
				t.Errorf("close: %v", err)
			}
		}()
		close(begin)
		group.Wait()
		view, err := r.Reception("ws1", "r1")
		any := false
		for _, did := range added {
			any = any || did
		}
		if err != nil || view.State != Closed || (view.Events != 3 && view.Events != 0) || any != (view.Events == 3) {
			t.Fatalf("round %d: %+v %v, added %v", round, view, err, added)
		}
		counted, err := r.Counting("ws1", "tm-1", from, to)
		wantSequence := uint64(3)
		if view.Events == 3 {
			wantSequence = 4
			for _, did := range added {
				if !did {
					t.Fatalf("round %d: an identical batch was refused although it was stored: %v", round, added)
				}
			}
		}
		if (err == nil) != (view.Events == 3) || counted.Sequence != wantSequence {
			t.Fatalf("round %d: the same batch from %d goroutines moved the number to %d, want %d (%v)", round, racers, counted.Sequence, wantSequence, err)
		}
		winner := ""
		for _, id := range winners {
			if id != "" && winner != "" {
				t.Fatalf("round %d: %s and %s both received the export", round, winner, id)
			}
			if id != "" {
				winner = id
			}
		}
		for racer, occupant := range occupants {
			if winner == "" || (winners[racer] == "") != (occupant == winner) {
				t.Fatalf("round %d: winner %q, occupants %v", round, winner, occupants)
			}
		}
		rows, err := r.Receptions("ws1", "tm-1", "r1", 5)
		if err != nil || len(rows) != 1 || rows[0].ID != winner {
			t.Fatalf("round %d: %+v %v", round, rows, err)
		}
		if view.Events != 3 {
			continue
		}
		if err := r.Void("ws1", winner, ByOperator); err != nil {
			t.Fatal(err)
		}
		voided++
		begin = make(chan struct{})
		for reader := 0; reader < 4; reader++ {
			group.Add(1)
			go func() {
				defer group.Done()
				<-begin
				gone := false
				for spin := 0; spin < 200; spin++ {
					verdict, err := r.Eligibility("ws1", "r1")
					counted, found := r.Counting("ws1", "tm-1", from, to)
					if err != nil || (found != nil && found != ErrNotFound) {
						t.Errorf("reads: %v %v", err, found)
						return
					}
					switch text := show(verdict); {
					case text == "not_eligible[voided]":
						gone = true
					case text != "eligible[]" || gone:
						t.Errorf("a verdict of %s after the void was seen: %v", text, gone)
						return
					}
					if gone && found == nil {
						t.Errorf("%+v counts after the void was seen", counted)
						return
					}
					if found == nil && counted.Reception != "r1" {
						t.Errorf("%+v", counted)
						return
					}
				}
			}()
		}
		group.Add(1)
		go func() {
			defer group.Done()
			<-begin
			for spin := 0; spin < round*10; spin++ {
				if _, err := r.Eligibility("ws1", "r1"); err != nil {
					t.Errorf("eligibility: %v", err)
				}
			}
			if err := r.Void("ws1", "r1", WithdrawnAtSource); err != nil {
				t.Errorf("void: %v", err)
			}
		}()
		close(begin)
		group.Wait()
		if verdict, err := r.Eligibility("ws1", "r1"); err != nil || show(verdict) != "not_eligible[voided]" {
			t.Fatalf("round %d: %s %v", round, show(verdict), err)
		}
		if counted, err := r.Counting("ws1", "tm-1", from, to); err != ErrNotFound || counted.Sequence != 6 {
			t.Fatalf("round %d: %+v %v", round, counted, err)
		}
	}
}
