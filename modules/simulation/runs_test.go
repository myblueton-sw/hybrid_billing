package simulation

import (
	"sync"
	"testing"
	"time"
)

func TestRunOutcomes(t *testing.T) {
	f := standard(t)
	f.expect("s1", standardChecks()...)
	list := results(standardChecks())
	list[2].Currency = "EUR"
	list = append(list[:1], list[2], Result{Name: "zz-extra", Value: "1"})
	f.advance(time.Minute)
	v := f.record("s1", "1.0.0", "r", "o", list)
	want := []Outcome{{Name: "krw-total", Outcome: OutcomeMatch}, {Name: "lines", Outcome: OutcomeMissing}, {Name: "usd-total", Outcome: OutcomeMismatch, Value: "-3.25", Currency: "EUR"}}
	if len(v.Outcomes) != 3 {
		t.Fatalf("outcomes %+v", v.Outcomes)
	}
	for index := range want {
		if v.Outcomes[index] != want[index] {
			t.Errorf("outcome %d %+v", index, v.Outcomes[index])
		}
	}
	if v.Unexpected != 1 || v.Verdict != VerdictFail || v.Key != KeySingle || v.Number != 1 || v.Harness != "h1" || v.Expectation != 1 || !v.At.Equal(start.Add(time.Minute)) {
		t.Errorf("view %+v", v)
	}
	if v.Results != digest(encodeResults(list)) || v.Output != sum("o") || v.Rules != sum("r") || v.Engine != "1.0.0" || v.Scenario != "s1" {
		t.Errorf("fields %+v", v)
	}
	value := results(standardChecks())
	value[0].Value = "1000.51"
	if w := f.record("s1", "1.0.0", "r2", "o2", value); w.Verdict != VerdictFail || w.Outcomes[0].Outcome != OutcomeMismatch || w.Outcomes[0].Value != "1000.51" {
		t.Errorf("value mismatch %+v", w)
	}
	extra := append(results(standardChecks()), Result{Name: "zz", Value: "1"})
	if w := f.record("s1", "1.0.0", "r3", "o3", extra); w.Verdict != VerdictFail || w.Unexpected != 1 {
		t.Errorf("unexpected %+v", w)
	}
	got, err := f.r.Run(1)
	if err != nil || got.Outcomes[2] != want[2] {
		t.Errorf("read %+v %v", got, err)
	}
	got.Outcomes[0].Name = "mutated"
	if again, _ := f.r.Run(1); again.Outcomes[0].Name != "krw-total" {
		t.Error("run read shares memory")
	}
	for _, n := range []uint64{0, 4} {
		expect(t, "run missing", second(f.r.Run(n)), ErrNotFound)
	}
}

func TestMismatchValueLimit(t *testing.T) {
	f := standard(t)
	checks := []Check{amount("a-total", "1", "KRW")}
	for index := 0; index < 10; index++ {
		checks = append(checks, Check{Name: "c" + itoa(index), Kind: CheckCount, Value: "1"})
	}
	f.expect("s1", checks...)
	list := results(checks)
	for index := range list {
		list[index].Value = "2"
	}
	v := f.record("s1", "1.0.0", "r", "o", list)
	withValue := 0
	for _, o := range v.Outcomes {
		if o.Outcome != OutcomeMismatch {
			t.Fatalf("outcome %+v", o)
		}
		if o.Value != "" {
			withValue++
		}
	}
	if withValue != maxValues || v.Outcomes[maxValues-1].Value != "2" || v.Outcomes[maxValues].Value != "" || len(f.r.runs[0].values) != maxValues {
		t.Errorf("values kept %d %+v", withValue, v.Outcomes)
	}
}

func TestReproducibility(t *testing.T) {
	f := standard(t)
	f.expect("s1", standardChecks()...)
	good := results(standardChecks())
	first := f.record("s1", "1.0.0", "r", "o", good)
	if first.Verdict != VerdictUnconfirmed || first.Key != KeySingle {
		t.Errorf("single %+v", first)
	}
	second := f.record("s1", "1.0.0", "r", "o", good)
	if second.Verdict != VerdictPass || second.Key != KeyReproduced {
		t.Errorf("reproduced %+v", second)
	}
	if again, _ := f.r.Run(1); again.Verdict != VerdictPass {
		t.Errorf("earlier run %+v", again)
	}
	shifted := results(standardChecks())
	shifted[1].Value = "13"
	third := f.record("s1", "1.0.0", "r", "o", shifted)
	if third.Verdict != VerdictNonReproducible || third.MarkedBy != 3 || third.Key != KeyNonReproducible {
		t.Errorf("results differ %+v", third)
	}
	fourth := f.record("s1", "1.0.0", "r", "o2", good)
	if fourth.MarkedBy != 3 {
		t.Errorf("mark moved %+v", fourth)
	}
	for _, n := range []uint64{1, 2} {
		if v, _ := f.r.Run(n); v.Verdict != VerdictNonReproducible || v.MarkedBy != 3 {
			t.Errorf("run %d %+v", n, v)
		}
	}
	other := f.record("s1", "1.0.0", "r9", "o", good)
	if other.Key != KeySingle {
		t.Errorf("separate key %+v", other)
	}
	g := standard(t)
	g.expect("s1", standardChecks()...)
	g.record("s1", "1.0.0", "r", "o", good)
	if v := g.record("s1", "1.0.0", "r", "o-changed", good); v.Verdict != VerdictNonReproducible || v.MarkedBy != 2 {
		t.Errorf("output differs %+v", v)
	}
}

func TestCopiedAnswer(t *testing.T) {
	f := standard(t)
	f.expect("s1", standardChecks()...)
	x := Run{Scenario: "s1", Engine: "1.0.0", Rules: sum("r"), Output: sum("sheet-s11"), Results: results(standardChecks())}
	for index := 0; index < 2; index++ {
		v, err := f.r.Record(x, runner)
		f.must(err)
		if !v.Copied || v.Verdict != VerdictFail {
			t.Errorf("copied %+v", v)
		}
	}
}

func TestCurrent(t *testing.T) {
	f := standard(t)
	f.expect("s1", standardChecks()...)
	c, err := f.r.Current("s1")
	if err != nil || c.Verdict != VerdictNone || c.Run != 0 {
		t.Errorf("none %+v %v", c, err)
	}
	expect(t, "missing", second(f.r.Current("nope")), ErrNotFound)
	good, bad := results(standardChecks()), results(standardChecks())
	bad[0].Value = "1"
	f.record("s1", "9.0.0", "r", "o", good)
	f.record("s1", "9.0.0", "r", "o", good)
	if c, _ := f.r.Current("s1"); c.Verdict != VerdictPass || c.Run != 2 || c.Engine != "9.0.0" || c.Rules != sum("r") {
		t.Errorf("pass %+v", c)
	}
	f.record("s1", "10.0.0", "r", "x", bad)
	if c, _ := f.r.Current("s1"); c.Verdict != VerdictFail || c.Engine != "10.0.0" {
		t.Errorf("newer engine %+v", c)
	}
	f.record("s1", "9.0.0", "r", "o", good)
	if c, _ := f.r.Current("s1"); c.Verdict != VerdictFail || c.Run != 3 {
		t.Errorf("older engine masks %+v", c)
	}
	f.record("s1", "10.0.0", "r2", "y", good)
	f.record("s1", "10.0.0", "r2", "y", good)
	if c, _ := f.r.Current("s1"); c.Verdict != VerdictFail || c.Run != 3 {
		t.Errorf("other rule set masks %+v", c)
	}
	f.record("s1", "10.0.0", "r", "x", bad)
	if c, _ := f.r.Current("s1"); c.Verdict != VerdictFail || c.Run != 7 {
		t.Errorf("latest of rule set %+v", c)
	}
	g := standard(t)
	g.expect("s1", standardChecks()...)
	g.record("s1", "1.0.0", "a", "o", good)
	g.record("s1", "1.0.0", "b", "p", good)
	g.record("s1", "1.0.0", "b", "p", good)
	if c, _ := g.r.Current("s1"); c.Verdict != VerdictUnconfirmed || c.Run != 1 {
		t.Errorf("worst is unconfirmed %+v", c)
	}
	g.record("s1", "1.0.0", "a", "o", good)
	if c, _ := g.r.Current("s1"); c.Verdict != VerdictPass || c.Run != 4 {
		t.Errorf("equal verdicts pick latest %+v", c)
	}
	g.must(g.r.RevokeHarness("h1", ann))
	if c, _ := g.r.Current("s1"); c.Verdict != VerdictRevoked {
		t.Errorf("revoked %+v", c)
	}
}

func TestRunsPaging(t *testing.T) {
	f := standard(t)
	f.expect("s1", standardChecks()...)
	for index := 0; index < 5; index++ {
		f.record("s1", "1.0.0", "r", "o", nil)
	}
	page, err := f.r.Runs("s1", 2, 2)
	if err != nil || len(page) != 2 || page[0].Number != 3 || page[1].Number != 4 {
		t.Errorf("page %+v %v", page, err)
	}
	if all, _ := f.r.Runs("s1", 0, maxPage); len(all) != 5 {
		t.Errorf("all %d", len(all))
	}
	if none, _ := f.r.Runs("f1", 0, 1); len(none) != 0 || none == nil {
		t.Errorf("empty %+v", none)
	}
	for _, limit := range []int{0, maxPage + 1} {
		expect(t, "limit", second(f.r.Runs("s1", 0, limit)), ErrInvalid)
	}
	expect(t, "missing", second(f.r.Runs("nope", 0, 1)), ErrNotFound)
}

func TestConcurrentRecord(t *testing.T) {
	f := standard(t)
	f.expect("s1", standardChecks()...)
	var wg sync.WaitGroup
	for index := 0; index < 8; index++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < 20; n++ {
				if _, err := f.r.Record(Run{Scenario: "s1", Engine: "1.0.0", Rules: sum("r"), Output: sum("o"), Results: results(standardChecks())}, runner); err != nil {
					t.Error(err)
				}
				f.r.Current("s1")
				f.r.Summary()
			}
		}()
	}
	wg.Wait()
	if c, _ := f.r.Current("s1"); c.Verdict != VerdictPass || len(f.r.runs) != 160 || f.r.keys[[3]string{"s1", "1.0.0", sum("r")}].count != 160 {
		t.Errorf("concurrent %+v %d", c, len(f.r.runs))
	}
}
