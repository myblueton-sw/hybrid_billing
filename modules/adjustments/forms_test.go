package adjustments

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func TestNewRegistry(t *testing.T) {
	clock := &testClock{now: start}
	expect(t, "invalid", second(NewRegistry(decimal.Profile{}, testLimits(), clock.read)), "profile")
	expect(t, "invalid", second(NewRegistry(testProfile(t), testLimits(), nil)), "clock")
	for index := range 5 {
		l := testLimits()
		fields := []*int{&l.Workspaces, &l.Drafts, &l.Lines, &l.Entries, &l.Evidence}
		*fields[index] = 0
		expect(t, "invalid", second(NewRegistry(testProfile(t), l, clock.read)), "zero limit "+strconv.Itoa(index))
	}
	caps := []int{100000, 1000000, 10000, 1000, 64}
	for index := range 5 {
		l := testLimits()
		fields := []*int{&l.Workspaces, &l.Drafts, &l.Lines, &l.Entries, &l.Evidence}
		*fields[index] = caps[index]
		expect(t, "ok", second(NewRegistry(testProfile(t), l, clock.read)), "cap "+strconv.Itoa(index))
		*fields[index] = caps[index] + 1
		expect(t, "invalid", second(NewRegistry(testProfile(t), l, clock.read)), "over cap "+strconv.Itoa(index))
	}
}

func TestOpenForms(t *testing.T) {
	r, _ := setup(t)
	bad := []func(*Draft){
		func(d *Draft) { d.ID = "" },
		func(d *Draft) { d.ID = "bad id" },
		func(d *Draft) { d.Customer = "" },
		func(d *Draft) { d.Contract = "" },
		func(d *Draft) { d.From = "2026-13-01" },
		func(d *Draft) { d.To = "" },
		func(d *Draft) { d.To = "2026-09-30" },
		func(d *Draft) { d.Base.Run = "" },
		func(d *Draft) { d.Base.Digest = digest('A') },
		func(d *Draft) { d.Base.Digest = "abc" },
		func(d *Draft) { d.Base.Currency = "usd" },
		func(d *Draft) { d.Base.Scale = -1 },
		func(d *Draft) { d.Base.Scale = 9 },
		func(d *Draft) { d.Base.Mode = "nearest" },
		func(d *Draft) { d.Base.Issued = true },
		func(d *Draft) { d.Base.Document = "INV-1" },
		func(d *Draft) { d.Base.Issued, d.Base.Document = true, "bad doc" },
		func(d *Draft) { d.Lines = nil },
		func(d *Draft) { d.Lines[1].ID = "l1" },
		func(d *Draft) { d.Lines[0].ID = "" },
		func(d *Draft) { d.Lines[0].Charge = "" },
		func(d *Draft) { d.Lines[0].Quantity = "-1" },
		func(d *Draft) { d.Lines[0].Quantity = "x" },
		func(d *Draft) { d.Lines[0].Price = "-1" },
		func(d *Draft) { d.Lines[0].Price = "1e3" },
		func(d *Draft) { d.Lines[0].Amount = "25.001" },
		func(d *Draft) { d.Lines[0].Amount = "" },
		func(d *Draft) { d.Lines[0].Amount = strings.Repeat("1", 65) },
	}
	for index, change := range bad {
		d := sample("d1", false)
		change(&d)
		expect(t, "invalid", second(r.Open("ws", d, human("writer"))), "open form "+strconv.Itoa(index))
	}
	expect(t, "invalid", second(r.Open("", sample("d1", false), human("writer"))), "workspace")
	expect(t, "invalid", second(r.Open("ws", sample("d1", false), Actor{ID: "x", Kind: "robot"})), "actor kind")
	expect(t, "invalid", second(r.Open("ws", sample("d1", false), Actor{Kind: KindHuman})), "actor id")
	d := sample("d1", false)
	d.Lines[0].Amount, d.Lines[1].Amount = "-25", "0"
	v := must(r.Open("ws", d, assistant("bot")))
	if v.Source != "-25" || v.Opener != assistant("bot") || v.SourceDigest == "" || v.Digest == "" {
		t.Fatalf("open %+v", v)
	}
	again := must(r.Open("ws", d, assistant("bot")))
	if again.Digest != v.Digest {
		t.Fatal("replay digest")
	}
	expect(t, "conflict", second(r.Open("ws", d, human("bot"))), "other opener")
	d.Lines[0].Charge = "ch-9"
	expect(t, "conflict", second(r.Open("ws", d, assistant("bot"))), "other content")
	d.Lines = d.Lines[:1]
	expect(t, "conflict", second(r.Open("ws", d, assistant("bot"))), "fewer lines")
	for _, change := range []func(*Draft){
		func(d *Draft) { d.Customer = "c2" },
		func(d *Draft) { d.Contract = "k2" },
		func(d *Draft) { d.From = "2026-10-02" },
		func(d *Draft) { d.To = "2026-10-30" },
		func(d *Draft) { d.Base.Scale = 3 },
	} {
		e := sample("d1", false)
		e.Lines[0].Amount, e.Lines[1].Amount = "-25", "0"
		change(&e)
		expect(t, "conflict", second(r.Open("ws", e, assistant("bot"))), "changed field")
	}
}

func TestEntryForms(t *testing.T) {
	r, _ := setup(t)
	opened(t, r, sample("d1", false))
	bad := []Entry{
		{},
		func() Entry { e := item("e", EntryFixed, "", "", "1"); e.ID = "bad id"; return e }(),
		func() Entry { e := item("e", "bonus", "", "", "1"); return e }(),
		func() Entry { e := item("e", EntryFixed, "", "", "1"); e.Reason = "whim"; return e }(),
		func() Entry { e := item("e", EntryFixed, "", "", "1"); e.Evidence = nil; return e }(),
		func() Entry { e := item("e", EntryFixed, "", "", "1"); e.Evidence = []string{"a", "a"}; return e }(),
		func() Entry { e := item("e", EntryFixed, "", "", "1"); e.Evidence = []string{"bad ref"}; return e }(),
		func() Entry { e := item("e", EntryFixed, "", "", "1"); e.From = "2026-10-32"; return e }(),
		func() Entry { e := item("e", EntryFixed, "", "", "1"); e.To = ""; return e }(),
		func() Entry {
			e := item("e", EntryFixed, "", "", "1")
			e.From, e.To = "2026-10-05", "2026-10-04"
			return e
		}(),
		func() Entry { e := item("e", EntryFixed, "", "", "1"); e.Line = "bad line"; return e }(),
		func() Entry { e := item("e", EntryFixed, "", "", "1"); e.Finding = "bad finding"; return e }(),
		item("e", EntryFixed, "", "", "0"),
		item("e", EntryFixed, "", "", ""),
		item("e", EntryFixed, "", "1", "1"),
		item("e", EntryQuantity, "", "2", ""),
		item("e", EntryQuantity, "l1", "-2", ""),
		item("e", EntryQuantity, "l1", "2", "1"),
		item("e", EntryQuantity, "l1", "", ""),
		item("e", EntryTargetTotal, "", "", ""),
		item("e", EntryTargetTotal, "", "120", "1"),
		item("e", EntryTargetTotal, "", "1.2.3", ""),
	}
	for index, e := range bad {
		expect(t, "invalid", second(r.Add("ws", "d1", 1, e, human("clerk"))), "entry form "+strconv.Itoa(index))
	}
	good := item("e", EntryFixed, "", "", "1")
	expect(t, "invalid", second(r.Add("", "d1", 1, good, human("clerk"))), "workspace")
	expect(t, "invalid", second(r.Add("ws", "", 1, good, human("clerk"))), "draft")
	expect(t, "invalid", second(r.Add("ws", "d1", 0, good, human("clerk"))), "revision")
	expect(t, "invalid", second(r.Add("ws", "d1", 1, good, Actor{ID: "x"})), "actor")
	expect(t, "invalid", second(r.Remove("", "d1", 1, "e", human("clerk"))), "remove workspace")
	expect(t, "invalid", second(r.Remove("ws", "", 1, "e", human("clerk"))), "remove draft")
	expect(t, "invalid", second(r.Remove("ws", "d1", 0, "e", human("clerk"))), "remove revision")
	expect(t, "invalid", second(r.Remove("ws", "d1", 1, "", human("clerk"))), "remove entry")
	expect(t, "invalid", second(r.Remove("ws", "d1", 1, "e", Actor{ID: "x"})), "remove actor")
	expect(t, "not_found", second(r.Remove("ws", "zz", 1, "e", human("clerk"))), "remove draft unknown")
	expect(t, "invalid", second(r.Of("", "d1")), "of workspace")
	expect(t, "invalid", second(r.Of("ws", "")), "of draft")
	expect(t, "not_found", second(r.Of("ws", "zz")), "of unknown")
	expect(t, "invalid", second(r.Withdraw("", "d1", human("lead"))), "withdraw workspace")
	expect(t, "invalid", second(r.Withdraw("ws", "", human("lead"))), "withdraw draft")
	expect(t, "invalid", second(r.Withdraw("ws", "d1", Actor{ID: "x"})), "withdraw actor")
}

func TestCheckAndReleaseForms(t *testing.T) {
	r, _ := setup(t)
	v := ready(t, r, "d1", item("e1", EntryFixed, "", "", "5"))
	good := check(v, "0", "105")
	for index, change := range []func(*Check){
		func(c *Check) { c.Revision = 0 },
		func(c *Check) { c.Digest = "x" },
		func(c *Check) { c.Tax = "" },
		func(c *Check) { c.Allocation = digest('Z') },
		func(c *Check) { c.TaxAmount = "" },
		func(c *Check) { c.Final = "" },
	} {
		c := good
		change(&c)
		expect(t, "invalid", second(r.Validate("ws", "d1", c, system("tax"))), "check form "+strconv.Itoa(index))
	}
	expect(t, "invalid", second(r.Validate("", "d1", good, system("tax"))), "check workspace")
	expect(t, "invalid", second(r.Validate("ws", "", good, system("tax"))), "check draft")
	expect(t, "invalid", second(r.Validate("ws", "d1", good, Actor{ID: "x"})), "check actor")
	q := release("r1", v, human("ann"))
	for index, change := range []func(*Release){
		func(q *Release) { q.Request = "" },
		func(q *Release) { q.Revision = 0 },
		func(q *Release) { q.Subject = "" },
		func(q *Release) { q.Approval = "bad ref" },
		func(q *Release) { q.Approvers = nil },
		func(q *Release) { q.Approvers = []Actor{human("a"), human("b"), human("c"), human("d")} },
		func(q *Release) { q.Approvers = []Actor{human("a"), human("a")} },
		func(q *Release) { q.Approvers = []Actor{{ID: "a"}} },
		func(q *Release) { q.Base.Run = "" },
		func(q *Release) { q.Base.Digest = "" },
		func(q *Release) { q.Base.Issued = true },
		func(q *Release) { q.Base.Document = "INV-1" },
		func(q *Release) { q.Base.Issued, q.Base.Document = true, "bad doc" },
	} {
		c := q
		c.Approvers = append([]Actor(nil), q.Approvers...)
		change(&c)
		expect(t, "invalid", r.Ready("ws", "d1", c, system("runner")), "release form "+strconv.Itoa(index))
		expect(t, "invalid", second(r.Release("ws", "d1", c, system("runner"))), "release form "+strconv.Itoa(index))
	}
	expect(t, "invalid", r.Ready("", "d1", q, system("runner")), "release workspace")
	expect(t, "invalid", r.Ready("ws", "", q, system("runner")), "release draft")
	expect(t, "invalid", r.Ready("ws", "d1", q, Actor{ID: "x"}), "release actor")
	three := q
	three.Approvers = []Actor{human("a"), human("b"), human("c")}
	expect(t, "ok", r.Ready("ws", "d1", three, system("runner")), "three approvers")
}

func TestLimits(t *testing.T) {
	r, _ := setup(t)
	d := sample("d1", false)
	for index := range 3 {
		d.Lines = append(d.Lines, SourceLine{ID: "x" + strconv.Itoa(index), Charge: "c", Quantity: "1", Amount: "1"})
	}
	expect(t, "limit", second(r.Open("ws", d, human("writer"))), "lines")
	d.Lines = d.Lines[:4]
	expect(t, "ok", second(r.Open("ws", d, human("writer"))), "four lines")
	for index := 2; index <= 6; index++ {
		expect(t, "ok", second(r.Open("ws", sample("d"+strconv.Itoa(index), false), human("writer"))), "draft")
	}
	expect(t, "limit", second(r.Open("ws", sample("d7", false), human("writer"))), "drafts")
	expect(t, "ok", second(r.Open("w2", sample("d1", false), human("writer"))), "second workspace")
	expect(t, "ok", second(r.Open("w3", sample("d1", false), human("writer"))), "third workspace")
	expect(t, "limit", second(r.Open("w4", sample("d1", false), human("writer"))), "workspaces")
	expect(t, "not_found", second(r.Of("w4", "d1")), "nothing recorded")
	e := item("e", EntryFixed, "", "", "1")
	e.Evidence = []string{"a", "b", "c"}
	expect(t, "limit", second(r.Add("ws", "d2", 1, e, human("clerk"))), "evidence")
	e.Evidence = e.Evidence[:2]
	expect(t, "ok", second(r.Add("ws", "d2", 1, e, human("clerk"))), "two evidence")
	v := must(r.Of("ws", "d2"))
	for index := range 5 {
		v = must(r.Add("ws", "d2", v.Revision, item("f"+strconv.Itoa(index), EntryFixed, "", "", "1"), human("clerk")))
	}
	v = must(r.Remove("ws", "d2", v.Revision, "f0", human("clerk")))
	expect(t, "limit", second(r.Add("ws", "d2", v.Revision, item("g", EntryFixed, "", "", "1"), human("clerk"))), "entries include removed")
}

func TestProfileBounds(t *testing.T) {
	r, _ := setup(t)
	d := sample("d1", false)
	d.Lines = []SourceLine{
		{ID: "l1", Charge: "c1", Quantity: "1", Price: "99999999999999", Amount: "99999999999999"},
		{ID: "l2", Charge: "c2", Quantity: "1", Amount: "99999999999999"},
	}
	v := must(r.Open("ws", d, human("writer")))
	expect(t, "limit", second(r.Add("ws", "d1", v.Revision, item("e", EntryQuantity, "l1", "99999999999999", ""), human("clerk"))), "product beyond profile")
	expect(t, "ok", second(r.Add("ws", "d1", v.Revision, item("e", EntryFixed, "", "", "99999999999999"), human("clerk"))), "within")
	d.ID = "d2"
	d.Lines = []SourceLine{{ID: "l1", Charge: "c1", Quantity: "1", Amount: "99999999999999"}}
	for index := range 99 {
		d.Lines = append(d.Lines, SourceLine{ID: "x" + strconv.Itoa(index), Charge: "c", Quantity: "1", Amount: "99999999999999"})
	}
	big, _ := NewRegistry(testProfile(t), Limits{Workspaces: 1, Drafts: 1, Lines: 200, Entries: 1, Evidence: 1}, (&testClock{now: start}).read)
	expect(t, "ok", second(big.Open("ws", d, human("writer"))), "sum within result digits")
	tiny, _ := decimal.NewProfile(decimal.ProfileSpec{ID: "tiny", Revision: "1", InputIntegerDigits: 3, InputScale: 2, ResultIntegerDigits: 3, ResultScale: 2, IntermediateDigits: 10})
	small, _ := NewRegistry(tiny, testLimits(), (&testClock{now: start}).read)
	sum := Draft{ID: "d1", Customer: "c", Contract: "k", From: "2026-10-01", To: "2026-10-31", Base: base(false), Lines: []SourceLine{
		{ID: "a", Charge: "c", Quantity: "1", Amount: "999"}, {ID: "b", Charge: "c", Quantity: "1", Amount: "1"},
	}}
	expect(t, "limit", second(small.Open("ws", sum, human("writer"))), "source total beyond profile")
	expect(t, "not_found", second(small.Of("ws", "d1")), "nothing recorded")
}

func TestClockAndConcurrency(t *testing.T) {
	r, clock := setup(t)
	clock.advance(time.Hour)
	v := opened(t, r, sample("d1", false))
	clock.advance(-2 * time.Hour)
	v = must(r.Add("ws", "d1", v.Revision, item("e1", EntryFixed, "", "", "5"), human("clerk")))
	if !v.Opened.Equal(start.Add(time.Hour)) || !v.Entries[0].Written.Equal(start.Add(time.Hour)) {
		t.Fatalf("clock %+v", v)
	}
	v = must(r.Validate("ws", "d1", check(v, "0", "105"), system("tax")))
	var wg sync.WaitGroup
	var mu sync.Mutex
	ok := 0
	for index := range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := r.Release("ws", "d1", release("r"+strconv.Itoa(index), v, human("ann")), system("runner")); err == nil {
				mu.Lock()
				ok++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if ok != 1 {
		t.Errorf("%d releases", ok)
	}
	w := opened(t, r, sample("d2", false))
	wg = sync.WaitGroup{}
	ok = 0
	for index := range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := r.Add("ws", "d2", w.Revision, item("e"+strconv.Itoa(index), EntryFixed, "", "", "1"), human("clerk")); err == nil {
				mu.Lock()
				ok++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if ok != 1 {
		t.Errorf("%d adds at one revision", ok)
	}
}

func TestSmallHelpers(t *testing.T) {
	err := refuse(RefuseExpiredForTest)
	if !strings.HasSuffix(err.Error(), ": "+RefuseExpiredForTest) || !strings.HasPrefix(err.Error(), ErrRejected.Error()) {
		t.Fatal(err)
	}
	if !validDate("2000-01-01") || !validDate("2099-12-31") || validDate("1999-12-31") || validDate("2100-01-01") || validDate("2026-02-30") {
		t.Fatal("dates")
	}
	if !validID(strings.Repeat("a", 128)) || validID(strings.Repeat("a", 129)) {
		t.Fatal("ids")
	}
	if !validID("A.z_0:9-Z") || validID("a/b") || validID("a b") || validID("é") {
		t.Fatal("id characters")
	}
	if !validCurrency("USD") || validCurrency("US") || validCurrency("USDX") || validCurrency("uSD") || validCurrency("UsD") || validCurrency("USd") {
		t.Fatal("currency")
	}
	if !validDigest(digest('0')) || !validDigest(digest('9')) || !validDigest(digest('a')) || !validDigest(digest('f')) || validDigest(digest('g')) || validDigest(digest('/')) || validDigest(digest(':')) || validDigest(digest('`')) || validDigest(digest('a')[:63]) {
		t.Fatal("digest")
	}
}

const RefuseExpiredForTest = "expired_for_test"

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
		for _, line := range strings.Split(text, "\n") {
			if strings.Contains(line, "hybrid_bil"+"ling/") && !strings.Contains(line, "packages/primitives/") {
				t.Errorf("%s imports %s", e.Name(), strings.TrimSpace(line))
			}
		}
	}
}
