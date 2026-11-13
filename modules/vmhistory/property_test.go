package vmhistory

import (
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"
)

var errorKinds = []error{ErrInvalid, ErrNotFound, ErrConflict, ErrRejected, ErrLimit}

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

func tick(seconds int64) time.Time { return at(0).Add(time.Duration(seconds) * time.Second) }

type generator struct {
	t       *testing.T
	rnd     *rand.Rand
	r       *Registry
	w       *world
	seed    int64
	reached map[string]int
	counted map[string]bool
	spans   int
}

func (g *generator) pick(options ...string) string { return options[g.rnd.Intn(len(options))] }

func (g *generator) once(key string, where any) {
	if mark := fmt.Sprint(key, "|", g.seed, "|", where); !g.counted[mark] {
		g.counted[mark] = true
		g.reached[key]++
	}
}

func (g *generator) same(what string, where any, got, want error) {
	g.t.Helper()
	if got != want {
		g.t.Fatalf("seed %d %s %v: the registry said %v, the model %v", g.seed, what, where, got, want)
	}
	if want == nil {
		g.reached[what+"/ok"]++
		return
	}
	g.once(what+"/"+kindOf(want), where)
}

func (g *generator) asset() string { return g.pick("a0", "a1", "a2", "a3", "a4") }

func (g *generator) meter() string { return g.pick("c0", "c1", "c2", "c3") }

func (g *generator) putAsset() {
	rnd := g.rnd
	id := g.asset()
	kind := AssetKind(g.pick("vm", "vm", "host"))
	created, deleted, revision := int64(rnd.Intn(6)), int64(-1), 1
	if rnd.Intn(3) == 0 {
		deleted = 28 + int64(rnd.Intn(14))
	}
	if held := g.w.assets[id]; held != nil {
		switch rnd.Intn(4) {
		case 0:
			kind, created, deleted, revision = held.kind, held.created, held.deleted, held.revision
		case 1:
			revision = held.revision + 1
		case 2:
			revision = held.revision + 1 + rnd.Intn(2)
			created = held.created
		default:
			revision = held.revision - rnd.Intn(2)
		}
	}
	switch rnd.Intn(25) {
	case 0:
		kind = ""
	case 1:
		deleted = created
	case 2:
		created = g.w.now + 5
	}
	spec := Asset{ID: id, Kind: kind, Created: tick(created), Revision: revision}
	if deleted >= 0 {
		spec.Deleted = tick(deleted)
	}
	g.same("asset", id, g.r.PutAsset("ws1", "vc-1", spec), g.w.putAsset(id, kind, created, deleted, revision))
}

func (g *generator) putSpan(asset string, from, to int64, plain bool) {
	rnd := g.rnd
	g.spans++
	spec := Span{ID: fmt.Sprintf("s%d", g.spans), Asset: asset, From: tick(from), To: tick(to), CPUs: g.pick("", "0", "2", "2", "4"), Organization: g.pick("", "o1", "o1", "o2")}
	if size := g.pick("", "1 KiB", "1 kB", "3 kB"); size != "" {
		fmt.Sscan(size, &spec.Memory.Value, &spec.Memory.Unit)
	}
	if size := g.pick("", "2 KiB", "2 kB"); size != "" {
		fmt.Sscan(size, &spec.Disk.Value, &spec.Disk.Unit)
	}
	if plain {
		spec.CPUs, spec.Memory, spec.Disk = "2", Size{Value: "1", Unit: "kB"}, Size{Value: "2", Unit: "KiB"}
	}
	if !plain {
		switch rnd.Intn(30) {
		case 0:
			spec.CPUs = "x"
		case 1:
			spec.Memory.Unit = ""
		case 2:
			spec.ID = fmt.Sprintf("s%d", 1+rnd.Intn(g.spans))
		case 3:
			spec.Asset = "nowhere"
		}
	}
	state, err := g.r.PutSpan("ws1", "vc-1", spec)
	want, wantErr := g.w.putSpan(spec, from, to)
	g.same("span", spec.Asset, err, wantErr)
	if state != want {
		g.t.Fatalf("span: %v, model %v", state, want)
	}
}

func (g *generator) putReadings(id string, batch []mInput) {
	specs := make([]Reading, len(batch))
	for index, input := range batch {
		specs[index] = Reading{At: tick(input.at), Value: input.text, Generation: input.generation, Revision: input.revision}
	}
	states, err := g.r.PutReadings("ws1", "vc-1", id, specs)
	want, wantErr := g.w.putReadings(id, batch)
	g.same("readings", id, err, wantErr)
	if !reflect.DeepEqual(states, want) {
		g.t.Fatalf("readings: %v, model %v", states, want)
	}
}

func (g *generator) step() {
	rnd := g.rnd
	switch choice := rnd.Intn(100); {
	case choice < 10:
		g.putAsset()
	case choice < 24:
		from := int64(rnd.Intn(40))
		to := from + 1 + int64(rnd.Intn(12))
		if rnd.Intn(30) == 0 {
			to = from
		}
		if rnd.Intn(40) == 0 {
			to = g.w.now + 3
		}
		g.putSpan(g.asset(), from, to, false)
	case choice < 30:
		asset := g.asset()
		if held := g.w.assets[asset]; held != nil {
			from := held.created
			for from < 40 {
				to := from + 4 + int64(rnd.Intn(10))
				g.putSpan(asset, from, to, rnd.Intn(3) > 0)
				from = to
			}
		}
	case choice < 36:
		id := fmt.Sprintf("s%d", 1+rnd.Intn(g.spans+1))
		reason := VoidReason(g.pick("wrong_report", "operator", "operator", ""))
		state, err := g.r.VoidSpan("ws1", "vc-1", id, reason)
		want, wantErr := g.w.voidSpan(id, reason)
		g.same("voidspan", id, err, wantErr)
		if state != want {
			g.t.Fatal(state, want)
		}
		if wantErr == nil && rnd.Intn(3) == 0 {
			_, err := g.r.VoidSpan("ws1", "vc-1", id, WithdrawnAtSource)
			_, wantErr := g.w.voidSpan(id, WithdrawnAtSource)
			g.same("voidspan", id, err, wantErr)
		}
	case choice < 48:
		spec := Counter{ID: g.meter(), Asset: g.asset(), Unit: g.pick("Wh", "Wh", "kWh", "kW"), MaxGap: int64(rnd.Intn(8))}
		if held := g.w.counter(spec.ID); held != nil && rnd.Intn(2) == 0 {
			spec.Asset, spec.MaxGap, spec.Unit = held.asset, held.gap, map[int64]string{1: "Wh", 1000: "kWh"}[held.factor]
		}
		state, err := g.r.PutCounter("ws1", "vc-1", spec)
		want, wantErr := g.w.putCounter(spec)
		g.same("counter", spec.ID, err, wantErr)
		if state != want {
			g.t.Fatal(state, want)
		}
	case choice < 52:
		id := g.meter()
		reason := VoidReason(g.pick("wrong_report", "operator", "operator", "operator", ""))
		state, err := g.r.VoidCounter("ws1", "vc-1", id, reason)
		want, wantErr := g.w.voidCounter(id, reason)
		g.same("voidcounter", id, err, wantErr)
		if state != want {
			g.t.Fatal(state, want)
		}
		if wantErr == nil && rnd.Intn(3) == 0 {
			_, err := g.r.VoidCounter("ws1", "vc-1", id, WithdrawnAtSource)
			_, wantErr := g.w.voidCounter(id, WithdrawnAtSource)
			g.same("voidcounter", id, err, wantErr)
		}
	case choice < 64:
		id := g.meter()
		meter := g.w.counter(id)
		if meter == nil {
			g.putReadings(id, []mInput{{at: 1, text: "1", generation: 1, revision: 1}})
			return
		}
		owner := g.w.assets[meter.asset]
		at, generation := owner.created, 1
		if rnd.Intn(3) == 0 {
			at += int64(rnd.Intn(5))
		}
		for at <= 42 {
			var batch []mInput
			for len(batch) < 4 && at <= 42 {
				batch = append(batch, mInput{at: at, text: fmt.Sprintf("%d.%d", at*3, rnd.Intn(2)*5), generation: generation, revision: 1})
				at += 1 + int64(rnd.Intn(int(meter.gap)))
				if rnd.Intn(25) == 0 {
					at += meter.gap
				}
				if rnd.Intn(40) == 0 {
					generation++
				}
			}
			g.putReadings(id, batch)
		}
	case choice < 80:
		id := g.meter()
		var batch []mInput
		for index, count := 0, rnd.Intn(6); index < count; index++ {
			input := mInput{at: int64(rnd.Intn(44)), generation: 1 + rnd.Intn(8)/7, revision: 1}
			input.text = g.pick(fmt.Sprint(input.at*3), fmt.Sprint(input.at*3), fmt.Sprint(input.at*3), "7", "0.5", "x", "1.")
			if earlier := g.w.reading(id, input.at); earlier != nil {
				switch rnd.Intn(4) {
				case 0:
					input = mInput{at: earlier.at, text: earlier.text, generation: earlier.generation, revision: earlier.revision}
				case 1:
					input.revision = earlier.revision + 1
				case 2:
					input.revision = earlier.revision
				}
			}
			if rnd.Intn(50) == 0 {
				input.at = g.w.now + 9
			}
			batch = append(batch, input)
		}
		g.putReadings(id, batch)
	default:
		id := g.meter()
		at, revision := int64(rnd.Intn(44)), 1
		var all []*mReading
		for _, held := range g.w.readings {
			if held.counter == id {
				all = append(all, held)
			}
		}
		if len(all) > 0 && rnd.Intn(5) > 0 {
			held := all[rnd.Intn(len(all))]
			at, revision = held.at, held.revision-rnd.Intn(6)/5
		}
		reason := VoidReason(g.pick("wrong_report", "operator", "operator", "operator", ""))
		state, err := g.r.VoidReading("ws1", "vc-1", id, tick(at), revision, reason)
		want, wantErr := g.w.voidReading(id, at, revision, reason)
		g.same("voidreading", id, err, wantErr)
		if state != want {
			g.t.Fatal(state, want)
		}
		if wantErr == nil && rnd.Intn(3) == 0 {
			_, err := g.r.VoidReading("ws1", "vc-1", id, tick(at), revision, WithdrawnAtSource)
			_, wantErr := g.w.voidReading(id, at, revision, WithdrawnAtSource)
			g.same("voidreading", id, err, wantErr)
		}
	}
}

func total(out Result) (*big.Rat, bool) {
	if out.Energy.Total == "" {
		return nil, false
	}
	value, _ := new(big.Rat).SetString(out.Energy.Total)
	return value, true
}

func (g *generator) check() {
	t := g.t
	rnd := g.rnd
	ids := make([]string, 0, len(g.w.assets))
	for id := range g.w.assets {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		from := int64(rnd.Intn(46)) - 3
		to := from + 1 + int64(rnd.Intn(46))
		var meters []*mCounter
		for _, meter := range g.w.counters {
			if meter.asset == id && !meter.voided && len(g.w.live(meter.id)) > 1 {
				meters = append(meters, meter)
			}
		}
		if len(meters) > 0 && rnd.Intn(3) == 0 {
			live := g.w.live(meters[rnd.Intn(len(meters))].id)
			first := rnd.Intn(len(live) - 1)
			from, to = live[first].at, live[first+1+rnd.Intn(len(live)-first-1)].at
		} else if rnd.Intn(3) == 0 {
			owner := g.w.assets[id]
			from, to = owner.created, 42
			if owner.deleted >= 0 {
				to = owner.deleted
			}
			if rnd.Intn(2) == 0 {
				from += int64(rnd.Intn(10))
				to = from + 2 + int64(rnd.Intn(20))
			}
		}
		got, err := g.r.Usage("ws1", "vc-1", id, tick(from), tick(to))
		want, metered := g.w.usage(id, from, to)
		if err != nil || cfg(got) != cfg(want) || en(got) != en(want) || got.Window != want.Window || got.Sequence != want.Sequence || got.Change != want.Change {
			t.Fatalf("seed %d asset %s over [%d,%d):\n got %s | %s | %d %d/%d\nwant %s | %s | %d %d/%d (%v)", g.seed, id, from, to, cfg(got), en(got), got.Window, got.Sequence, got.Change, cfg(want), en(want), want.Window, want.Sequence, want.Change, err)
		}
		g.once("config/"+got.Configuration.Result, id)
		g.once("energy/"+got.Energy.Result, id)
		for _, reason := range got.Configuration.Reasons {
			g.once("reason/"+reason, id)
		}
		for _, reason := range got.Energy.Reasons {
			g.once("reason/"+reason, id)
		}
		if got.Window == 0 {
			continue
		}
		for _, part := range []struct {
			result           string
			reasons          int
			covered, missing int64
		}{{got.Configuration.Result, len(got.Configuration.Reasons), got.Configuration.Covered, got.Configuration.Uncovered}, {got.Energy.Result, len(got.Energy.Reasons), got.Energy.Covered, got.Energy.Uncovered}} {
			if part.covered+part.missing != got.Window || part.covered < 0 || part.missing < 0 || (part.result == Complete) != (part.reasons == 0) || (part.result == Complete && part.missing != 0) {
				t.Fatalf("seed %d asset %s: %+v in a window of %d", g.seed, id, part, got.Window)
			}
		}
		if metered && got.Energy.Result == Complete {
			sum := new(big.Rat)
			owner := g.w.assets[id]
			first, last := from, to
			if owner.created > first {
				first = owner.created
			}
			if owner.deleted >= 0 && owner.deleted < last {
				last = owner.deleted
			}
			for _, meter := range g.w.counters {
				if meter.asset != id || meter.voided {
					continue
				}
				var inside []*mReading
				for _, held := range g.w.live(meter.id) {
					if held.at >= first && held.at <= last {
						inside = append(inside, held)
					}
				}
				if len(inside) > 1 {
					sum.Add(sum, new(big.Rat).SetFrac(new(big.Int).Sub(inside[len(inside)-1].micro, inside[0].micro), big.NewInt(1000000)))
				}
			}
			if value, present := total(got); !present || value.Cmp(sum) != 0 {
				t.Fatalf("seed %d asset %s: a complete total of %s, last less first gives %s", g.seed, id, got.Energy.Total, sum.FloatString(6))
			}
			g.once("oracle", id)
		}
		if to-from < 2 {
			continue
		}
		middle := from + 1 + int64(rnd.Intn(int(to-from-1)))
		left, _ := g.r.Usage("ws1", "vc-1", id, tick(from), tick(middle))
		right, _ := g.r.Usage("ws1", "vc-1", id, tick(middle), tick(to))
		if left.Configuration.Covered+right.Configuration.Covered != got.Configuration.Covered || left.Window+right.Window != got.Window {
			t.Fatalf("seed %d asset %s: configuration is not additive at %d", g.seed, id, middle)
		}
		whole, parts := map[string]*big.Int{}, map[string]*big.Int{}
		collect := func(into map[string]*big.Int, out Result) {
			entries := append([]Entry(nil), out.Configuration.Entries...)
			if out.Configuration.None != nil {
				entries = append(entries, *out.Configuration.None)
			}
			for _, entry := range entries {
				for name, text := range map[string]string{"cpu": entry.CPU.Integral, "mem": entry.Memory.Integral, "disk": entry.Disk.Integral, "seconds": fmt.Sprint(entry.Seconds)} {
					if text == "" {
						continue
					}
					value, _ := new(big.Int).SetString(text, 10)
					key := entry.Organization + "/" + name
					if into[key] == nil {
						into[key] = new(big.Int)
					}
					into[key].Add(into[key], value)
				}
			}
		}
		collect(whole, got)
		collect(parts, left)
		collect(parts, right)
		if fmt.Sprint(whole) != fmt.Sprint(parts) {
			t.Fatalf("seed %d asset %s: integrals of [%d,%d) and [%d,%d) do not add up: %v against %v", g.seed, id, from, middle, middle, to, parts, whole)
		}
		perCounter := func(out Result) map[string]*big.Rat {
			found := map[string]*big.Rat{}
			for _, entry := range out.Energy.Counters {
				if entry.Energy != "" {
					found[entry.Counter], _ = new(big.Rat).SetString(entry.Energy)
				}
			}
			return found
		}
		union, a, b := perCounter(got), perCounter(left), perCounter(right)
		crossing := map[string]bool{}
		segments, _ := g.w.segments(id)
		for _, part := range segments {
			if part.kind == "paired" && part.from < middle && part.to > middle {
				crossing[part.meter] = true
			}
		}
		for _, entry := range got.Energy.Counters {
			sum := new(big.Rat)
			for _, side := range []map[string]*big.Rat{a, b} {
				if side[entry.Counter] != nil {
					sum.Add(sum, side[entry.Counter])
				}
			}
			over := union[entry.Counter]
			if over == nil {
				over = new(big.Rat)
			}
			if order := over.Cmp(sum); order < 0 || (order != 0 && !crossing[entry.Counter]) {
				t.Fatalf("seed %d asset %s counter %s: %s over the union, %s over the parts at %d", g.seed, id, entry.Counter, over.FloatString(6), sum.FloatString(6), middle)
			}
			if crossing[entry.Counter] {
				g.once("crossing", id)
			}
		}
	}
	for _, id := range ids {
		var live []Span
		after := time.Time{}
		for {
			page, err := g.r.Spans("ws1", "vc-1", id, after, 3)
			if err != nil {
				t.Fatal(err)
			}
			live = append(live, page.Items...)
			if !page.More {
				break
			}
			after = page.Items[len(page.Items)-1].From
		}
		owner := g.w.assets[id]
		count := 0
		for _, held := range g.w.spans {
			if held.asset == id && !held.voided {
				count++
			}
		}
		if len(live) != count {
			t.Fatalf("seed %d asset %s lists %d live spans, the model has %d", g.seed, id, len(live), count)
		}
		for index, held := range live {
			if index > 0 && held.From.Before(live[index-1].To) {
				t.Fatalf("seed %d asset %s: live spans overlap", g.seed, id)
			}
			if held.From.Before(tick(owner.created)) || (owner.deleted >= 0 && held.To.After(tick(owner.deleted))) || held.To.After(start) {
				t.Fatalf("seed %d asset %s: a live span outside the life", g.seed, id)
			}
		}
	}
	for _, meter := range g.w.counters {
		if meter.voided {
			continue
		}
		var listed []Reading
		after := time.Time{}
		for {
			page, err := g.r.Readings("ws1", "vc-1", meter.id, after, 3)
			if err != nil {
				t.Fatal(err)
			}
			listed = append(listed, page.Items...)
			if !page.More {
				break
			}
			after = page.Items[len(page.Items)-1].At
		}
		live := g.w.live(meter.id)
		owner := g.w.assets[meter.asset]
		if len(listed) != len(live) {
			t.Fatalf("seed %d counter %s lists %d readings, the model has %d", g.seed, meter.id, len(listed), len(live))
		}
		for index, held := range listed {
			if !held.At.Equal(tick(live[index].at)) || held.Value != live[index].text || held.Generation != live[index].generation || held.Revision != live[index].revision {
				t.Fatalf("seed %d counter %s: %+v against %+v", g.seed, meter.id, held, live[index])
			}
			if index > 0 && held.Generation < listed[index-1].Generation {
				t.Fatalf("seed %d counter %s: generations fall", g.seed, meter.id)
			}
			if held.At.Before(tick(owner.created)) || (owner.deleted >= 0 && held.At.After(tick(owner.deleted))) {
				t.Fatalf("seed %d counter %s: a live reading outside the life", g.seed, meter.id)
			}
		}
	}
}

func TestRandomHistoriesAgainstAModelAndInvariants(t *testing.T) {
	reached := map[string]int{}
	limits := Limits{Sources: 1, Assets: 4, Spans: 5, Counters: 2, Readings: 60, Batch: 4, Page: 3, MaxPeriod: 40 * day, MaxGap: 6}
	now := int64(start.Sub(at(0)) / time.Second)
	for seed := int64(1); seed <= 300; seed++ {
		r, err := NewRegistry(limits, func() time.Time { return start })
		if err != nil {
			t.Fatal(err)
		}
		g := &generator{t: t, rnd: rand.New(rand.NewSource(seed)), r: r, w: newWorld(limits, now), seed: seed, reached: reached, counted: map[string]bool{}}
		for step := 0; step < 110; step++ {
			g.step()
			if step%6 == 5 {
				g.check()
			}
		}
		g.check()
	}
	least := map[string]int{"oracle": 60, "crossing": 60, "config/complete": 60, "config/incomplete": 200, "config/not_applicable": 100, "energy/complete": 60, "energy/incomplete": 200, "energy/not_applicable": 100}
	for _, reason := range []string{NotCovered, CPUNotStated, MemoryNotStated, DiskNotStated, NoCounter, CountersOverlap, CounterReset, CounterDecrease, CounterGap, BoundaryNotRead, NotRead} {
		least["reason/"+reason] = 30
	}
	for _, key := range []string{"asset/ok", "span/ok", "counter/ok", "readings/ok", "voidspan/ok", "voidcounter/ok", "voidreading/ok"} {
		least[key] = 150
	}
	for _, key := range []string{"asset/" + kindOf(ErrInvalid), "asset/" + kindOf(ErrConflict), "asset/" + kindOf(ErrRejected), "asset/" + kindOf(ErrLimit),
		"span/" + kindOf(ErrInvalid), "span/" + kindOf(ErrNotFound), "span/" + kindOf(ErrConflict), "span/" + kindOf(ErrRejected), "span/" + kindOf(ErrLimit),
		"counter/" + kindOf(ErrInvalid), "counter/" + kindOf(ErrNotFound), "counter/" + kindOf(ErrConflict), "counter/" + kindOf(ErrLimit),
		"readings/" + kindOf(ErrInvalid), "readings/" + kindOf(ErrNotFound), "readings/" + kindOf(ErrConflict), "readings/" + kindOf(ErrRejected), "readings/" + kindOf(ErrLimit),
		"voidspan/" + kindOf(ErrInvalid), "voidspan/" + kindOf(ErrNotFound), "voidspan/" + kindOf(ErrConflict),
		"voidcounter/" + kindOf(ErrInvalid), "voidcounter/" + kindOf(ErrNotFound), "voidcounter/" + kindOf(ErrConflict),
		"voidreading/" + kindOf(ErrInvalid), "voidreading/" + kindOf(ErrNotFound), "voidreading/" + kindOf(ErrConflict), "voidreading/" + kindOf(ErrRejected)} {
		least[key] = 15
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
	for round := 0; round < 40; round++ {
		r, err := NewRegistry(testLimits(), func() time.Time { return start })
		if err != nil {
			t.Fatal(err)
		}
		h := &harness{t: t, r: r}
		h.asset(machine("vm"))
		h.counter("c", "vm", "Wh")
		h.span(sized("base", "vm", 0, 2, "1", "", "", ""))
		batch := []Reading{rd(0, "1", 1), rd(1, "2", 1), rd(2, "3", 1)}
		begin := make(chan struct{})
		var group sync.WaitGroup
		run := func(work func()) {
			group.Add(1)
			go func() {
				defer group.Done()
				<-begin
				work()
			}()
		}
		for writer := 0; writer < 6; writer++ {
			run(func() {
				if _, err := r.PutReadings("ws1", "vc-1", "c", batch); err != nil {
					t.Errorf("readings: %v", err)
				}
			})
		}
		results := make([]error, 4)
		for index := range results {
			run(func() {
				_, results[index] = r.PutSpan("ws1", "vc-1", sized(fmt.Sprintf("x%d", index), "vm", 2+index, 6+index, "1", "", "", ""))
			})
		}
		for reader := 0; reader < 3; reader++ {
			run(func() {
				var last, change uint64
				for spin := 0; spin < 100; spin++ {
					out, err := r.Usage("ws1", "vc-1", "vm", at(0), at(2))
					if err != nil || out.Sequence < last || out.Change < change || out.Change > out.Sequence {
						t.Errorf("usage: %+v %v after %d/%d", out, err, last, change)
						return
					}
					last, change = out.Sequence, out.Change
					covered := out.Configuration.Covered
					if covered != 7200 && covered != 0 {
						t.Errorf("a result with the span or without it: %d", covered)
						return
					}
					if text := en(out); text != "incomplete[not_read] 0/7200 total=- c=-/0" && text != "complete[] 7200/0 total=2 c=2/7200" {
						t.Errorf("a batch is seen whole or not at all: %s", text)
						return
					}
				}
			})
		}
		run(func() {
			for spin := 0; spin < round*5; spin++ {
				if _, err := r.Assets("ws1", "vc-1", "", 1); err != nil {
					t.Errorf("assets: %v", err)
				}
			}
			if _, err := r.VoidSpan("ws1", "vc-1", "base", WrongReport); err != nil {
				t.Errorf("void: %v", err)
			}
		})
		close(begin)
		group.Wait()
		stored := 0
		for index, err := range results {
			if err == nil {
				stored++
				for other := range results[:index] {
					if results[other] == nil && index-other < 4 {
						t.Fatalf("round %d: the overlapping spans %d and %d were both stored", round, other, index)
					}
				}
			} else if err != ErrRejected {
				t.Fatalf("round %d: %v", round, err)
			}
		}
		out := h.usage("vm", at(0), at(2))
		if stored != 1 || out.Sequence != uint64(3+1+stored+1) || en(out) != "complete[] 7200/0 total=2 c=2/7200" || out.Configuration.Covered != 0 {
			t.Fatalf("round %d: %d spans stored, %d, %s", round, stored, out.Sequence, en(out))
		}
		page, err := r.Readings("ws1", "vc-1", "c", time.Time{}, 4)
		if err != nil || len(page.Items) != 3 {
			t.Fatalf("round %d: one batch from six goroutines stored once: %+v %v", round, page, err)
		}
	}
}
