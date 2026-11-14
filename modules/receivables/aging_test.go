package receivables

import (
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestAgingBoundaries(t *testing.T) {
	l := testLimits()
	l.Receivables = 20
	r, clock := openWith(t, l)
	setup(t, r, "k", "UTC", 0)
	today := time.Date(2026, 12, 31, 12, 0, 0, 0, time.UTC)
	clock.set(today)
	want := map[int]string{-1: BucketNotDue, 0: BucketDueToday, 1: Bucket1to30, 30: Bucket1to30, 31: Bucket31to60, 60: Bucket31to60, 61: Bucket61to90, 90: Bucket61to90, 91: Bucket91Plus, 300: Bucket91Plus}
	for days := range want {
		if days < 0 {
			continue
		}
		id := "d" + itoa(days)
		must(r.Issue("ws", invoice(id, "k", "1.5", today.AddDate(0, 0, -days)), "clerk"))
	}
	setup(t, r, "fut", "UTC", 1)
	must(r.Issue("ws", invoice("dm1", "fut", "1.5", today), "clerk"))
	must(r.Confirm("ws", "en", "USD", today, "sync"))
	for days, bucket := range want {
		id := "d" + itoa(days)
		if days < 0 {
			id = "dm1"
		}
		v := must(r.Status("ws", id))
		if v.Days != days || bucketOf(v) != bucket {
			t.Errorf("%d days: %d %s", days, v.Days, bucketOf(v))
		}
	}
	a := must(r.Aging("ws", Filter{}))
	if len(a.Currencies) != 1 || a.Currencies[0].Count != 10 || a.Currencies[0].Total != "15" || !a.Evaluated.Equal(today) {
		t.Fatalf("aging: %+v", a)
	}
	b := a.Currencies[0].Buckets
	for bucket, count := range map[string]int{BucketNotDue: 1, BucketDueToday: 1, Bucket1to30: 2, Bucket31to60: 2, Bucket61to90: 2, Bucket91Plus: 2, BucketUnknown: 0, BucketReplaced: 0} {
		if b[bucket].Count != count || b[bucket].StaleCount != 0 {
			t.Errorf("%s: %+v", bucket, b[bucket])
		}
	}
	if b[Bucket1to30].Amount != "3" || b[BucketUnknown].Amount != "0" {
		t.Errorf("sums: %+v", b)
	}
	clock.set(today.Add(49 * time.Hour))
	a = must(r.Aging("ws", Filter{}))
	b = a.Currencies[0].Buckets
	if b[BucketUnknown].Count != 10 || b[Bucket1to30].Count != 0 || b[BucketNotDue].Count != 0 || b[BucketDueToday].Count != 0 || a.Currencies[0].Total != "15" {
		t.Errorf("stale: %+v", b)
	}
	if b[BucketUnknown].StaleCount != 10 || b[BucketUnknown].StaleAmount != "15" {
		t.Errorf("stale sums: %+v", b[BucketUnknown])
	}
}

func TestAgingFiltersAndCurrencies(t *testing.T) {
	r, clock := openWith(t, testLimits())
	clock.set(time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC))
	setup(t, r, "k", "UTC", 30)
	setup(t, r, "j", "UTC", 30)
	at := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	must(r.Issue("ws", invoice("a", "k", "10", at), "clerk"))
	eur := invoice("b", "k", "7", at)
	eur.Currency = "EUR"
	must(r.Issue("ws", eur, "clerk"))
	other := invoice("c", "j", "5", at)
	other.Customer, other.Entity = "cv", "e2"
	must(r.Issue("ws", other, "clerk"))
	paid := invoice("p", "k", "3", at)
	must(r.Issue("ws", paid, "clerk"))
	must(r.Allocate("ws", receipt("x", "p", "3", "2026-10-02"), "sync"))
	all := must(r.Aging("ws", Filter{}))
	if len(all.Currencies) != 2 || all.Currencies[0].Currency != "EUR" || all.Currencies[1].Total != "15" || all.Currencies[1].Count != 2 {
		t.Fatalf("currencies: %+v", all)
	}
	if stale := all.Currencies[1].Buckets[BucketNotDue]; stale.StaleCount != 2 || stale.StaleAmount != "15" {
		t.Errorf("stale before due: %+v", stale)
	}
	for f, total := range map[Filter]string{{Customer: "cv"}: "5", {Entity: "en", Currency: "USD"}: "10", {Contract: "j"}: "5", {Currency: "EUR"}: "7"} {
		got := must(r.Aging("ws", f))
		if len(got.Currencies) != 1 || got.Currencies[0].Total != total {
			t.Errorf("%+v: %+v", f, got)
		}
	}
	if got := must(r.Aging("ws", Filter{Customer: "none"})); len(got.Currencies) != 0 {
		t.Error("an empty filter result")
	}
	if got := must(r.Aging("none", Filter{})); got.Currencies == nil || len(got.Currencies) != 0 {
		t.Error("an unknown workspace")
	}
	expect(t, "invalid", second(r.Aging("ws", Filter{Currency: "usd"})), "currency filter")
	expect(t, "invalid", second(r.Aging("", Filter{})), "workspace")
}

func TestConfirmations(t *testing.T) {
	r, clock := openWith(t, testLimits())
	clock.set(time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC))
	setup(t, r, "k", "UTC", 30)
	must(r.Issue("ws", invoice("d", "k", "10", time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)), "clerk"))
	if v := must(r.Status("ws", "d")); v.Status != StatusUnknown || !v.Stale || !v.Confirmed.IsZero() || v.Days != 5 || v.Outstanding != "10" {
		t.Errorf("no confirmation: %+v", v)
	}
	expect(t, "rejected", second(r.Confirm("ws", "en", "USD", clock.read().Add(time.Second), "sync")), "a future confirmation")
	old := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	clock.set(old.Add(47 * time.Hour))
	clock.set(time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC))
	must(r.Confirm("ws", "en", "USD", time.Date(2026, 11, 3, 0, 0, 1, 0, time.UTC), "sync"))
	if v := must(r.Status("ws", "d")); v.Status != StatusOverdue || v.Stale {
		t.Errorf("fresh: %+v", v)
	}
	expect(t, "rejected", second(r.Confirm("ws", "en", "USD", time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC), "sync")), "a confirmation that moves back")
	expect(t, "ok", second(r.Confirm("ws", "en", "USD", time.Date(2026, 11, 3, 0, 0, 1, 0, time.UTC), "sync")), "repeat")
	expect(t, "rejected", second(r.Confirm("ws", "en", "USD", time.Date(2026, 11, 3, 0, 0, 1, 0, time.UTC), "other")), "the same time by another actor")
	clock.set(time.Date(2026, 11, 5, 0, 0, 2, 0, time.UTC))
	if v := must(r.Status("ws", "d")); v.Status != StatusUnknown || !v.Stale || !v.Confirmed.Equal(time.Date(2026, 11, 3, 0, 0, 1, 0, time.UTC)) {
		t.Errorf("just beyond the freshness: %+v", v)
	}
	clock.set(time.Date(2026, 11, 5, 0, 0, 1, 0, time.UTC))
	if v := must(r.Status("ws", "d")); v.Status != StatusUnknown {
		t.Errorf("the clock never goes back: %+v", v)
	}
	r2, c2 := openWith(t, testLimits())
	c2.set(time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC))
	setup(t, r2, "k", "UTC", 30)
	must(r2.Confirm("ws", "en", "USD", time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC), "sync"))
	must(r2.Issue("ws", invoice("d", "k", "10", time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)), "clerk"))
	must(r2.Issue("ws", invoice("late", "k", "10", time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC)), "clerk"))
	if v := must(r2.Status("ws", "d")); v.Status != StatusOverdue || v.Stale {
		t.Errorf("exactly at the freshness limit: %+v", v)
	}
	if v := must(r2.Status("ws", "late")); v.Stale {
		t.Error("a confirmation older than the issue counts by its age")
	}
	other := invoice("eur", "k", "10", time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC))
	other.Currency = "EUR"
	must(r2.Issue("ws", other, "clerk"))
	if v := must(r2.Status("ws", "eur")); v.Status != StatusUnknown {
		t.Error("a confirmation of another currency counts")
	}
	expect(t, "invalid", second(r.Confirm("ws", "en", "USD", time.Time{}, "sync")), "zero time")
	expect(t, "invalid", second(r.Confirm("ws", "en", "usd", start, "sync")), "currency")
	expect(t, "not_found", second(r.Status("ws", "zz")), "unknown document")
	expect(t, "invalid", second(r.Status("ws", "")), "malformed document")
}

func TestRegistryAndRooms(t *testing.T) {
	zones := func(string) (*time.Location, error) { return time.UTC, nil }
	if _, err := NewRegistry(testProfile(t), testLimits(), nil, zones); kind(err) != "invalid" {
		t.Error("no clock")
	}
	if _, err := NewRegistry(testProfile(t), testLimits(), time.Now, nil); kind(err) != "invalid" {
		t.Error("no zones")
	}
	caps := Limits{Workspaces: 100000, Terms: 100000, Bindings: 1000, Receivables: 10000000, Allocations: 10000, Extensions: 100, Holds: 100, NetDays: 3650, Freshness: 31 * 24 * time.Hour}
	if _, err := NewRegistry(testProfile(t), caps, time.Now, zones); err != nil {
		t.Error(err)
	}
	for name, change := range map[string]func(*Limits){
		"workspaces": func(l *Limits) { l.Workspaces++ }, "terms": func(l *Limits) { l.Terms++ }, "bindings": func(l *Limits) { l.Bindings++ },
		"receivables": func(l *Limits) { l.Receivables++ }, "allocations": func(l *Limits) { l.Allocations++ }, "extensions": func(l *Limits) { l.Extensions++ },
		"holds": func(l *Limits) { l.Holds++ }, "net": func(l *Limits) { l.NetDays++ }, "fresh": func(l *Limits) { l.Freshness++ },
		"short": func(l *Limits) { l.Freshness = time.Minute - 1 }, "zero": func(l *Limits) { l.Workspaces = 0 },
	} {
		l := caps
		change(&l)
		if _, err := NewRegistry(testProfile(t), l, time.Now, zones); kind(err) != "invalid" {
			t.Errorf("%s: %v", name, err)
		}
	}
	short := caps
	short.Freshness = time.Minute
	if _, err := NewRegistry(testProfile(t), short, time.Now, zones); err != nil {
		t.Error("a minute of freshness")
	}
	l := testLimits()
	l.Workspaces, l.Terms, l.Receivables, l.Bindings = 2, 2, 2, 2
	r, clock := openWith(t, l)
	clock.set(time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC))
	setup(t, r, "k", "UTC", 30)
	must(r.Confirm("w2", "en", "USD", start, "sync"))
	expect(t, "limit", second(r.Confirm("w3", "en", "USD", start, "sync")), "workspaces")
	expect(t, "limit", second(r.RecordTerm("w3", Term{ID: "t", Revision: 1, Mode: ModeCalendar}, "c")), "workspaces by a term")
	must(r.RecordTerm("ws", Term{ID: "t", Revision: 1, Mode: ModeCalendar}, "c"))
	expect(t, "limit", second(r.RecordTerm("ws", Term{ID: "u", Revision: 1, Mode: ModeCalendar}, "c")), "terms")
	must(r.Bind("ws", Binding{ID: "b2", Contract: "k", Term: "t", Revision: 1, Zone: "UTC", From: "2030-01-01", Reason: "initial"}, "c"))
	expect(t, "limit", second(r.Bind("ws", Binding{ID: "b3", Contract: "k", Term: "t", Revision: 1, Zone: "UTC", From: "2031-01-01", Reason: "initial"}, "c")), "bindings")
	must(r.WithdrawBinding("ws", "b2", "c"))
	expect(t, "limit", second(r.Bind("ws", Binding{ID: "b3", Contract: "k", Term: "t", Revision: 1, Zone: "UTC", From: "2031-01-01", Reason: "initial"}, "c")), "withdrawn bindings count")
	must(r.Issue("ws", invoice("a", "k", "1", start.Add(time.Hour)), "c"))
	must(r.Issue("ws", invoice("b", "k", "1", start.Add(time.Hour)), "c"))
	expect(t, "limit", second(r.Issue("ws", invoice("c", "k", "1", start.Add(time.Hour)), "c")), "receivables")
	r2, c2 := openWith(t, testLimits())
	c2.set(time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC))
	setup(t, r2, "k", "UTC", 30)
	must(r2.Issue("ws", invoice("d", "k", "100", start.Add(time.Hour)), "c"))
	for i := 0; i < 4; i++ {
		id := "a" + itoa(i)
		must(r2.Allocate("ws", Allocation{ID: id, Document: "d", Kind: "receipt", Amount: "1", Effective: "2026-10-02", Source: SourceManual, Evidence: digest('a')}, "clerk"))
		if i < 3 {
			must(r2.ApproveAllocation("ws", id, "lead"))
		} else {
			must(r2.WithdrawAllocation("ws", id, "clerk"))
		}
	}
	for i := 0; i < 3; i++ {
		id := "v" + itoa(i)
		must(r2.Reverse("ws", Reversal{ID: id, Allocation: "a" + itoa(i), Effective: "2026-10-02", Source: SourceManual, Evidence: digest('a')}, "clerk"))
		must(r2.WithdrawReversal("ws", id, "clerk"))
	}
	must(r2.Reverse("ws", Reversal{ID: "v3", Allocation: "a0", Effective: "2026-10-02", Source: SourceManual, Evidence: digest('a')}, "clerk"))
	expect(t, "limit", second(r2.Reverse("ws", Reversal{ID: "v9", Allocation: "a1", Effective: "2026-10-02", Source: SourceManual, Evidence: digest('a')}, "clerk")), "reversals with withdrawn ones")
	expect(t, "limit", second(r2.Allocate("ws", Allocation{ID: "m", Document: "d", Kind: "receipt", Amount: "1", Effective: "2026-10-02", Source: SourceManual, Evidence: digest('a')}, "clerk")), "allocations")
}

func TestFormsAndAmounts(t *testing.T) {
	r, clock := openWith(t, testLimits())
	clock.set(time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC))
	setup(t, r, "k", "UTC", 30)
	good := invoice("d", "k", "1", start)
	for name, change := range map[string]func(*Receivable){
		"document": func(v *Receivable) { v.Document = "" }, "customer": func(v *Receivable) { v.Customer = "a b" }, "entity": func(v *Receivable) { v.Entity = "" },
		"currency": func(v *Receivable) { v.Currency = "US" }, "contract": func(v *Receivable) { v.Contract = "" }, "zero time": func(v *Receivable) { v.IssuedAt = time.Time{} },
		"obligation": func(v *Receivable) { v.Obligation = "maybe" }, "basis alone": func(v *Receivable) { v.DueBasis = BasisInherit },
		"self": func(v *Receivable) { v.Replaces, v.DueBasis = "d", BasisInherit }, "basis": func(v *Receivable) { v.Replaces, v.DueBasis = "x", "fresh" },
		"amount": func(v *Receivable) { v.Amount = "0" }, "digits": func(v *Receivable) { v.Amount = "123456789012345" }, "text": func(v *Receivable) { v.Amount = "1e2" },
		"long": func(v *Receivable) { v.Amount = "1." + strings.Repeat("0", 70) }, "old": func(v *Receivable) { v.IssuedAt = time.Date(1999, 12, 31, 12, 0, 0, 0, time.UTC) },
	} {
		v := good
		change(&v)
		expect(t, "invalid", second(r.Issue("ws", v, "clerk")), name)
	}
	v := good
	v.Amount = "12345678901234.12345678"
	if d := must(r.Issue("ws", v, "clerk")); d.Receivable.Amount != "12345678901234.12345678" {
		t.Error(d.Receivable.Amount)
	}
	v.Document, v.Amount = "e", "1.50"
	if d := must(r.Issue("ws", v, "clerk")); d.Receivable.Amount != "1.5" || len(d.Digest) != 64 {
		t.Errorf("canonical amount: %+v", d.Receivable)
	}
	expect(t, "ok", second(r.Issue("ws", v, "clerk")), "repeat with the same text")
	for name, call := range map[string]func() error{
		"approve term":  func() error { return second(r.ApproveTerm("", "a", 1, "b")) },
		"bind":          func() error { return second(r.Bind("ws", Binding{}, "c")) },
		"approve":       func() error { return second(r.ApproveBinding("ws", "", "c")) },
		"withdraw":      func() error { return second(r.WithdrawBinding("ws", "b", "")) },
		"end":           func() error { return second(r.EndBinding("ws", "b", "2026-13-01", "c")) },
		"approve end":   func() error { return second(r.ApproveEnd("ws", "", "c")) },
		"binding":       func() error { return second(r.BindingOf("", "b")) },
		"document":      func() error { return second(r.Document("ws", "")) },
		"extend":        func() error { return second(r.Extend("ws", Extension{}, "c")) },
		"approve ext":   func() error { return second(r.ApproveExtension("ws", "", "c")) },
		"withdraw ext":  func() error { return second(r.WithdrawExtension("ws", "x", "")) },
		"approve alloc": func() error { return second(r.ApproveAllocation("ws", "", "c")) },
		"withdraw a":    func() error { return second(r.WithdrawAllocation("", "x", "c")) },
		"reverse":       func() error { return second(r.Reverse("ws", Reversal{}, "c")) },
		"approve rev":   func() error { return second(r.ApproveReversal("ws", "", "c")) },
		"withdraw rev":  func() error { return second(r.WithdrawReversal("ws", "x", "")) },
		"hold":          func() error { return second(r.Hold("ws", Hold{}, "c")) },
		"release":       func() error { return second(r.ReleaseHold("ws", "", "c")) },
		"record term":   func() error { return second(r.RecordTerm("ws", Term{ID: "a", Revision: 1, Mode: ModeCalendar}, "")) },
		"issue actor":   func() error { return second(r.Issue("ws", good, "")) },
		"alloc actor":   func() error { return second(r.Allocate("ws", receipt("a", "d", "1", "2026-10-02"), "")) },
	} {
		if kind(call()) != "invalid" {
			t.Errorf("%s accepts a malformed request", name)
		}
	}
	for name, call := range map[string]func() error{
		"approve":     func() error { return second(r.ApproveBinding("ws", "zz", "c")) },
		"withdraw":    func() error { return second(r.WithdrawBinding("ws", "zz", "c")) },
		"approve end": func() error { return second(r.ApproveEnd("ws", "zz", "c")) },
		"binding":     func() error { return second(r.BindingOf("ws", "zz")) },
		"document":    func() error { return second(r.Document("ws", "zz")) },
		"approve ext": func() error { return second(r.ApproveExtension("ws", "zz", "c")) },
		"withdraw x":  func() error { return second(r.WithdrawExtension("ws", "zz", "c")) },
		"approve a":   func() error { return second(r.ApproveAllocation("ws", "zz", "c")) },
		"withdraw a":  func() error { return second(r.WithdrawAllocation("ws", "zz", "c")) },
		"approve r":   func() error { return second(r.ApproveReversal("ws", "zz", "c")) },
		"withdraw r":  func() error { return second(r.WithdrawReversal("ws", "zz", "c")) },
		"term":        func() error { return second(r.ApproveTerm("none", "a", 1, "c")) },
		"bind": func() error {
			return second(r.Bind("none", Binding{ID: "b", Contract: "k", Term: "t", Revision: 1, Zone: "UTC", From: "2026-01-01", Reason: "initial"}, "c"))
		},
		"space": func() error { return second(r.Document("none", "d")) },
	} {
		if kind(call()) != "not_found" {
			t.Errorf("%s finds an unknown record", name)
		}
	}
}

func TestConcurrentAllocationsNeverExceed(t *testing.T) {
	l := testLimits()
	l.Allocations = 100
	r, clock := openWith(t, l)
	clock.set(time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC))
	setup(t, r, "k", "UTC", 30)
	must(r.Issue("ws", invoice("d", "k", "100", start.Add(time.Hour)), "c"))
	var wg sync.WaitGroup
	for i := 0; i < 40; i++ {
		wg.Add(2)
		id := itoa(i)
		go func() { defer wg.Done(); r.Allocate("ws", receipt("a"+id, "d", "7", "2026-10-02"), "sync") }()
		go func() { defer wg.Done(); r.Aging("ws", Filter{}) }()
	}
	wg.Wait()
	if v := must(r.Status("ws", "d")); v.Outstanding != "2" || len(v.Document.Allocations) != 14 {
		t.Errorf("%s %d", v.Outstanding, len(v.Document.Allocations))
	}
}

func TestSourceRules(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		data, err := os.ReadFile(e.Name())
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		if strings.Contains(text, "flo"+"at") || strings.Contains(text, "/"+"/ ") || strings.Contains(text, "/"+"*") {
			t.Errorf("%s uses a binary fraction type or comments", e.Name())
		}
		if strings.Count(text, "\n") >= 1500 {
			t.Errorf("%s is too long", e.Name())
		}
		if strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		if strings.Contains(text, "time/tz"+"data") {
			t.Errorf("%s embeds zone data", e.Name())
		}
		for _, line := range strings.Split(text, "\n") {
			if strings.Contains(line, "hybrid_bil"+"ling/") && !strings.Contains(line, "packages/primitives/") {
				t.Errorf("%s imports %s", e.Name(), strings.TrimSpace(line))
			}
		}
	}
}
