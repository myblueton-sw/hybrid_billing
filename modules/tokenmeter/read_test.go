package tokenmeter

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func (h *harness) sequence(monthNumber int) uint64 {
	h.t.Helper()
	from, to := month(monthNumber)
	out, err := h.r.Counting("ws1", "tm-1", from, to)
	if err != nil && err != ErrNotFound {
		h.t.Fatal(err)
	}
	return out.Sequence
}

func TestCountingAndTheSequenceNumber(t *testing.T) {
	h := newHarness(t, nil)
	from, to := month(10)
	_, err := h.r.Counting("ws1", "tm-1", from, to)
	wantErr(t, err, ErrNotFound, "a source that is not known")
	spec := export("r1", "snap-1", 10)
	h.open(spec)
	if got := h.sequence(10); got != 1 || h.counting(10) != "" {
		t.Fatalf("after opening: %d", got)
	}
	h.open(spec)
	h.post("r1", event("e1", 10), event("e2", 10))
	h.post("r1", event("e1", 10), event("e2", 10))
	if got := h.sequence(10); got != 2 {
		t.Fatalf("repeats do not move the number: %d", got)
	}
	h.post("r1", event("e2", 10), event("e3", 10))
	for name, call := range map[string]func() error{
		"a refused header":   func() error { return third(h.r.Open(export("r1", "snap-2", 10))) },
		"an occupied export": func() error { return third(h.r.Open(export("r2", "snap-1", 10))) },
		"a refused batch":    func() error { return second(h.r.Post("ws1", "r1", []Event{event("e4", 10)})) },
		"a refused void":     func() error { return h.r.Void("ws1", "r1", "because") },
		"a read":             func() error { return second(h.r.Events("ws1", "r1", 0, 1)) },
	} {
		if call() == nil {
			t.Fatal(name)
		}
	}
	if got := h.sequence(10); got != 3 || h.counting(10) != "" {
		t.Fatalf("after three events: %d", got)
	}
	h.close("r1")
	h.close("r1")
	counted, err := h.r.Counting("ws1", "tm-1", from.In(time.FixedZone("kst", 9*3600)), to)
	if err != nil || counted != (Counted{Reception: "r1", Snapshot: "snap-1", Sequence: 4}) {
		t.Fatalf("%+v %v", counted, err)
	}
	for name, period := range map[string][2]time.Time{
		"a day of the month":     {from, from.Add(day)},
		"its last day":           {to.Add(-day), to},
		"the month and a second": {from, to.Add(time.Second)},
		"another month":          {to, to.Add(30 * day)},
	} {
		got, err := h.r.Counting("ws1", "tm-1", period[0], period[1])
		wantErr(t, err, ErrNotFound, name)
		if got != (Counted{Sequence: 4}) {
			t.Fatalf("%s: %+v", name, got)
		}
	}
	for name, period := range map[string][2]time.Time{
		"a period that runs backwards": {to, from},
		"an empty period":              {from, from},
		"a period with a fraction":     {from, to.Add(time.Millisecond)},
		"a period that is too long":    {from, from.Add(40*day + time.Second)},
		"no period":                    {},
	} {
		_, err := h.r.Counting("ws1", "tm-1", period[0], period[1])
		wantErr(t, err, ErrInvalid, name)
	}
	if _, err := h.r.Counting("ws1", "tm-1", from, from.Add(40*day)); err != ErrNotFound {
		t.Fatalf("the longest period: %v", err)
	}
	for _, c := range []struct {
		ws, source string
		want       error
	}{{"ws2", "tm-1", ErrNotFound}, {"ws1", "tm-2", ErrNotFound}, {"", "tm-1", ErrInvalid}, {"ws1", "tm 1", ErrInvalid}} {
		_, err := h.r.Counting(c.ws, c.source, from, to)
		wantErr(t, err, c.want, "counting")
	}
	other := export("x1", "snap-x", 10)
	other.Source = "tm-2"
	h.open(other)
	far := export("r1", "snap-1", 10)
	far.WorkspaceID = "ws2"
	if _, _, err := h.r.Open(far); err != nil {
		t.Fatal(err)
	}
	if got := h.sequence(10); got != 4 || h.counting(10) != "r1" {
		t.Fatalf("another source and another workspace have their own numbers: %d", got)
	}
	if err := h.r.Void("ws1", "r1", RegeneratedAtSource); err != nil {
		t.Fatal(err)
	}
	if err := h.r.Void("ws1", "r1", RegeneratedAtSource); err != nil {
		t.Fatal(err)
	}
	if got := h.sequence(10); got != 5 || h.counting(10) != "" {
		t.Fatalf("after the void: %d", got)
	}
	h.full(export("r2", "snap-1", 10))
	if got := h.sequence(10); got != 8 || h.counting(10) != "r2" {
		t.Fatalf("the export received again: %d %s", got, h.verdict("r2"))
	}
	var unbuilt Registry
	var missing *Registry
	for _, r := range []*Registry{&unbuilt, missing} {
		_, err := r.Counting("ws1", "tm-1", from, to)
		wantErr(t, err, ErrInvalid, "a registry that was not built")
		wantErr(t, second(r.Eligibility("ws1", "r1")), ErrInvalid, "eligibility")
		wantErr(t, second(r.Reception("ws1", "r1")), ErrInvalid, "reception")
		wantErr(t, second(r.Events("ws1", "r1", 0, 1)), ErrInvalid, "events")
		wantErr(t, second(r.Receptions("ws1", "tm-1", "", 1)), ErrInvalid, "receptions")
		wantErr(t, second(r.Post("ws1", "r1", []Event{event("e", 10)})), ErrInvalid, "post")
		wantErr(t, second(r.Close("ws1", "r1")), ErrInvalid, "close")
		wantErr(t, r.Void("ws1", "r1", ByOperator), ErrInvalid, "void")
		wantErr(t, third(r.Open(spec)), ErrInvalid, "open")
	}
}

func TestVoidingAndClosing(t *testing.T) {
	h := newHarness(t, nil)
	h.open(export("r1", "snap-1", 10))
	h.post("r1", event("b", 10), event("a", 10))
	for _, c := range []struct {
		ws, id string
		reason VoidReason
		want   error
	}{{"ws1", "r1", "", ErrInvalid}, {"ws1", "r1", "mistake", ErrInvalid}, {"ws1", "r9", "mistake", ErrInvalid}, {"ws1", "r9", ByOperator, ErrNotFound}, {"ws2", "r1", ByOperator, ErrNotFound}, {"", "r1", ByOperator, ErrInvalid}, {"ws1", "", ByOperator, ErrInvalid}} {
		before := h.r.dump()
		wantErr(t, h.r.Void(c.ws, c.id, c.reason), c.want, "void")
		h.unchanged(before, "a refused void")
	}
	for _, c := range []struct {
		ws, id string
		want   error
	}{{"ws1", "r9", ErrNotFound}, {"ws2", "r1", ErrNotFound}, {"", "r1", ErrInvalid}, {"ws1", "", ErrInvalid}} {
		wantErr(t, second(h.r.Close(c.ws, c.id)), c.want, "close")
		wantErr(t, second(h.r.Eligibility(c.ws, c.id)), c.want, "eligibility")
		wantErr(t, second(h.r.Reception(c.ws, c.id)), c.want, "reception")
		wantErr(t, second(h.r.Events(c.ws, c.id, 0, 1)), c.want, "events")
	}
	wantErr(t, second(h.r.Events("ws1", "r1", 0, 1)), ErrRejected, "the events of an open reception")
	h.clock.advance(time.Minute)
	h.verdict("r1")
	h.clock.advance(-time.Hour)
	for _, reason := range []VoidReason{WithdrawnAtSource, RegeneratedAtSource, WrongReport, ByOperator} {
		made := newHarness(t, nil)
		made.open(export("r1", "snap-1", 10))
		made.full(export("r2", "snap-2", 9))
		for _, id := range []string{"r1", "r2"} {
			if err := made.r.Void("ws1", id, reason); err != nil || made.view(id).VoidReason != reason || made.view(id).State != Voided || made.view(id).WasClosed != (id == "r2") {
				t.Fatal(reason, id, err)
			}
			if err := made.r.Void("ws1", id, reason); err != nil {
				t.Fatal(reason, id, err)
			}
		}
	}
	if err := h.r.Void("ws1", "r1", WithdrawnAtSource); err != nil {
		t.Fatal(err)
	}
	view := h.view("r1")
	if view.State != Voided || view.WasClosed || view.Figures != nil || view.Events != 2 || view.VoidReason != WithdrawnAtSource || !view.Changed.Equal(start.Add(time.Minute)) || !view.Opened.Equal(start) {
		t.Fatalf("the registry keeps the latest reading of its clock: %+v", view)
	}
	if got := h.events("r1"); len(got) != 2 || got[0].Event.ID != "a" || got[1].Event.ID != "b" {
		t.Fatalf("a voided reception shows what it received: %+v", got)
	}
	before := h.r.dump()
	wantErr(t, h.r.Void("ws1", "r1", ByOperator), ErrConflict, "a second reason")
	wantErr(t, second(h.r.Close("ws1", "r1")), ErrRejected, "closing what was voided open")
	wantErr(t, second(h.r.Post("ws1", "r1", []Event{event("c", 10)})), ErrRejected, "an event for a voided reception")
	if state, err := h.r.Post("ws1", "r1", []Event{event("a", 10)}); err != nil || state != Voided {
		t.Fatal(state, err)
	}
	if state, _, err := h.r.Open(export("r1", "snap-1", 10)); err != nil || state != Voided {
		t.Fatal(state, err)
	}
	h.unchanged(before, "a voided reception")
	h.clock.advance(2 * time.Hour)
	h.full(export("r2", "snap-1", 10))
	if err := h.r.Void("ws1", "r2", ByOperator); err != nil {
		t.Fatal(err)
	}
	view = h.view("r2")
	if view.State != Voided || !view.WasClosed || view.Figures == nil || flat(view.Figures) == "" || !view.Changed.Equal(start.Add(time.Hour+time.Minute)) {
		t.Fatalf("%+v", view)
	}
	if state, err := h.r.Close("ws1", "r2"); err != nil || state != Voided {
		t.Fatalf("closing again what was closed and voided is a repeat: %v %v", state, err)
	}
	mutable := h.view("r2")
	mutable.Spec.Tokens[Input], mutable.Spec.Costs["USD"] = "9", "9"
	if again := h.view("r2"); again.Spec.Tokens[Input] != "300" || again.Spec.Costs["USD"] != "3" {
		t.Fatal("a view shares the header")
	}
	rows := h.events("r2")
	rows[0].Event.Cost.Amount = "999"
	rows[0].Findings = append(rows[0].Findings, "x")
	if again := h.events("r2"); again[0].Event.Cost.Amount != "1" || again[0].Findings != nil {
		t.Fatal("a page shares an event")
	}
}

func TestPagesOfEventsAndReceptions(t *testing.T) {
	h := newHarness(t, nil)
	spec := export("r1", "snap-1", 10)
	spec.Events = 12
	h.open(spec)
	var want []string
	for index := 11; index >= 0; index-- {
		id := fmt.Sprintf("e%02d", index)
		h.post("r1", event(id, 10))
		want = append([]string{id}, want...)
	}
	h.close("r1")
	var got []string
	for _, row := range h.events("r1") {
		got = append(got, row.Event.ID)
	}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("events in the order of their identifiers: %v", got)
	}
	for _, c := range []struct{ after, limit, first, count int }{{0, 5, 0, 5}, {0, 1, 0, 1}, {10, 5, 10, 2}, {11, 1, 11, 1}, {12, 5, 0, 0}, {99, 5, 0, 0}, {3, 4, 3, 4}} {
		page, err := h.r.Events("ws1", "r1", c.after, c.limit)
		if err != nil || len(page) != c.count || (c.count > 0 && page[0].Event.ID != want[c.first]) {
			t.Fatalf("%+v: %v %v", c, page, err)
		}
	}
	for _, c := range []struct{ after, limit int }{{-1, 1}, {0, 0}, {0, 6}, {0, -1}} {
		wantErr(t, second(h.r.Events("ws1", "r1", c.after, c.limit)), ErrInvalid, "a page of events")
	}
	for _, id := range []string{"m", "c", "x", "a", "k", "b", "z"} {
		made := export(id, "snap-"+id, 9)
		h.open(made)
	}
	if err := h.r.Void("ws1", "c", ByOperator); err != nil {
		t.Fatal(err)
	}
	other := export("d", "snap-d", 9)
	other.Source = "tm-2"
	h.open(other)
	list := func(after string, limit int) string {
		rows, err := h.r.Receptions("ws1", "tm-1", after, limit)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, row := range rows {
			out = append(out, fmt.Sprintf("%s:%s:%s:%d", row.ID, row.Snapshot, row.State, row.PeriodStart.Month()*100+row.PeriodEnd.Month()))
		}
		return strings.Join(out, " ")
	}
	for _, c := range []struct {
		after string
		limit int
		want  string
	}{
		{"", 5, "a:snap-a:open:910 b:snap-b:open:910 c:snap-c:voided:910 k:snap-k:open:910 m:snap-m:open:910"},
		{"m", 5, "r1:snap-1:closed:1011 x:snap-x:open:910 z:snap-z:open:910"},
		{"a", 2, "b:snap-b:open:910 c:snap-c:voided:910"},
		{"bb", 1, "c:snap-c:voided:910"},
		{"z", 5, ""},
		{"zz", 1, ""},
	} {
		if got := list(c.after, c.limit); got != c.want {
			t.Fatalf("after %q:\n got %s\nwant %s", c.after, got, c.want)
		}
	}
	if rows, err := h.r.Receptions("ws1", "tm-9", "", 5); err != nil || rows != nil {
		t.Fatal(rows, err)
	}
	if rows, err := h.r.Receptions("ws2", "tm-1", "", 5); err != nil || rows != nil {
		t.Fatalf("another workspace sees nothing: %v %v", rows, err)
	}
	for _, c := range []struct {
		ws, source, after string
		limit             int
	}{{"", "tm-1", "", 1}, {"ws1", "", "", 1}, {"ws1", "tm-1", "a b", 1}, {"ws1", "tm-1", "", 0}, {"ws1", "tm-1", "", 6}} {
		wantErr(t, second(h.r.Receptions(c.ws, c.source, c.after, c.limit)), ErrInvalid, "a page of receptions")
	}
}

func TestTheBoundaryOfAWorkspace(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.Receptions = 2 })
	h.full(export("r1", "snap-1", 10))
	alone := h.r.dump()
	theirs := export("r1", "snap-1", 10)
	theirs.WorkspaceID = "ws2"
	theirs.Produced = ListInterface
	if state, occupant, err := h.r.Open(theirs); err != nil || state != Open || occupant != "" {
		t.Fatalf("the same identifiers in another workspace are free: %v %q %v", state, occupant, err)
	}
	if _, err := h.r.Post("ws2", "r1", []Event{with(event("r1-e1", 10), func(e *Event) { e.Outcome = Failed })}); err != nil {
		t.Fatal(err)
	}
	second := export("r2", "snap-2", 10)
	second.WorkspaceID = "ws2"
	if _, _, err := h.r.Open(second); err != nil {
		t.Fatal(err)
	}
	third := export("r3", "snap-3", 11)
	third.WorkspaceID = "ws2"
	if _, _, err := h.r.Open(third); err != ErrLimit {
		t.Fatalf("the limit of receptions is per workspace: %v", err)
	}
	h.open(export("r3", "snap-3", 11))
	if _, _, err := h.r.Open(export("r4", "snap-4", 12)); err != ErrLimit {
		t.Fatal(err)
	}
	if err := h.r.Void("ws2", "r1", ByOperator); err != nil {
		t.Fatal(err)
	}
	if got := h.verdict("r1"); got != "eligible[]" || h.counting(10) != "r1" || h.sequence(10) != 4 {
		t.Fatalf("what another workspace does changes nothing here: %s", got)
	}
	var mine []string
	for _, line := range strings.Split(h.r.dump(), "\n") {
		if strings.Contains(line, " ws1/r1 ") || strings.HasPrefix(line, "reception ws1/r1 ") {
			mine = append(mine, line)
		}
	}
	for _, line := range mine {
		if !strings.Contains(alone, line) {
			t.Fatalf("the reception changed: %s", line)
		}
	}
	if len(mine) != 1 {
		t.Fatal(mine)
	}
}

func TestRefusalsAreFixedValuesAndNothingIsComputedWithFloats(t *testing.T) {
	h := newHarness(t, nil)
	secret := "customer-9f3a"
	spec := export(secret, secret, 10)
	spec.Revision = 0
	_, occupant, err := h.r.Open(spec)
	if err != ErrInvalid || occupant != "" || strings.Contains(err.Error(), secret) {
		t.Fatal(err)
	}
	for _, kind := range []error{ErrInvalid, ErrNotFound, ErrConflict, ErrRejected, ErrLimit} {
		if strings.ContainsAny(kind.Error(), "%0123456789") {
			t.Fatalf("%q", kind)
		}
	}
	h.open(export("r1", "snap-1", 10))
	if _, occupant, err := h.r.Open(export("r2", "snap-1", 10)); err != ErrConflict || occupant != "r1" {
		t.Fatalf("the occupant of an export is named: %q %v", occupant, err)
	}
	far := export("r2", "snap-1", 10)
	far.Source = "tm-2"
	h.open(far)
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) < 4 {
		t.Fatal(files, err)
	}
	banned := regexp.MustCompile(`float32|float64|math\.|\.Float\(|ParseFloat|big\.Float|big\.Rat|\.Quo\(|\.Div\(|\.Exp\(|time\.Now|rand\.`)
	primitive := regexp.MustCompile(`decimal\.[A-Za-z]+\(`)
	scanned := 0
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		content, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		scanned++
		for number, line := range strings.Split(string(content), "\n") {
			if found := banned.FindString(line); found != "" && !strings.Contains(line, `"math/big"`) {
				t.Errorf("%s:%d uses %s", name, number+1, found)
			}
			for _, call := range primitive.FindAllString(line, -1) {
				if call != "decimal.New(" {
					t.Errorf("%s:%d calls %s", name, number+1, call)
				}
			}
			if strings.Count(line, ".Mul(") != strings.Count(line, ", ten)") || strings.Count(line, "QuoRem(") != strings.Count(line, "QuoRem(value, ten, remainder)") {
				t.Errorf("%s:%d multiplies or divides by something other than ten", name, number+1)
			}
			if strings.Contains(line, "//") || strings.Contains(line, "/*") {
				t.Errorf("%s:%d has a comment", name, number+1)
			}
		}
	}
	if scanned != 4 {
		t.Fatalf("scanned %d source files", scanned)
	}
}

func TestAnExportOfAHundredThousandEvents(t *testing.T) {
	h := newHarness(t, nil)
	const count, batch = 100000, 1000
	spec := export("big", "snap-big", 10)
	spec.Events = count
	spec.Tokens = map[Kind]string{Input: "10000000", Output: "1000000", CacheRead: "0", CacheWrite: "0"}
	spec.Costs = map[string]string{"USD": "1000"}
	h.open(spec)
	from, _ := month(10)
	began := time.Now()
	for first := 0; first < count; first += batch {
		events := make([]Event, batch)
		for index := range events {
			made := priced(fmt.Sprintf("e%06d", first+index), 10, "0.01", Billed, Provider)
			made.Time = from.Add(time.Duration(first+index) * time.Second)
			events[index] = made
		}
		h.post("big", events...)
	}
	figures := h.close("big")
	want := "error=0 net=10000000/0 cost/unconfirmed/USD=1000 outcome/succeeded=100000 tokens/provider_reported/cache_read=0 tokens/provider_reported/cache_write=0 tokens/provider_reported/input=10000000 tokens/provider_reported/output=1000000 "
	if got := flat(figures) + " "; got != want {
		t.Fatalf("\n got %s\nwant %s", got, want)
	}
	if got := h.verdict("big"); got != "eligible[]" {
		t.Fatal(got)
	}
	page, err := h.r.Events("ws1", "big", count-2, 5)
	if err != nil || len(page) != 2 || page[1].Event.ID != "e099999" {
		t.Fatal(page, err)
	}
	for round := 0; round < 200; round++ {
		if got := h.verdict("big"); got != "eligible[]" {
			t.Fatal(got)
		}
	}
	t.Logf("received, closed and judged in %s", time.Since(began).Round(time.Millisecond))
}
