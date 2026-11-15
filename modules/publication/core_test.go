package publication

import (
	"errors"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func TestConfig(t *testing.T) {
	f := setup(t)
	if f.r.limits != defaults {
		t.Errorf("defaults %+v", f.r.limits)
	}
	want := Limits{ChunkRows: 10000, VersionRows: 1000000, StagedRows: 2000000, TenantStagedRows: 1000000, HeldRows: 10000000, TenantHeldRows: 5000000, Groups: 100000,
		Retained: 2, Page: 1000, TenantDatasets: 10000, Datasets: 100000, StagingTTL: 24 * time.Hour}
	most := Limits{ChunkRows: 100000, VersionRows: 10000000, StagedRows: 20000000, TenantStagedRows: 20000000, HeldRows: 100000000, TenantHeldRows: 100000000,
		Groups: 1000000, Retained: 16, Page: 10000, TenantDatasets: 100000, Datasets: 1000000, StagingTTL: 30 * 24 * time.Hour}
	if defaults != want || maximums != most {
		t.Errorf("tables %+v %+v", defaults, maximums)
	}
	g := setupWith(t, most)
	if g.r.limits != most {
		t.Error("maximums refused")
	}
	clock := func() time.Time { return start }
	setters := []func(*Limits, int){
		func(l *Limits, n int) { l.ChunkRows = n }, func(l *Limits, n int) { l.VersionRows = n }, func(l *Limits, n int) { l.StagedRows = n },
		func(l *Limits, n int) { l.TenantStagedRows = n }, func(l *Limits, n int) { l.HeldRows = n }, func(l *Limits, n int) { l.TenantHeldRows = n },
		func(l *Limits, n int) { l.Groups = n }, func(l *Limits, n int) { l.Retained = n },
		func(l *Limits, n int) { l.Page = n }, func(l *Limits, n int) { l.TenantDatasets = n }, func(l *Limits, n int) { l.Datasets = n },
	}
	highs := []int{most.ChunkRows, most.VersionRows, most.StagedRows, most.TenantStagedRows, most.HeldRows, most.TenantHeldRows, most.Groups, most.Retained,
		most.Page, most.TenantDatasets, most.Datasets}
	for index, set := range setters {
		for _, value := range []int{-1, highs[index] + 1} {
			var l Limits
			set(&l, value)
			expect(t, "limit", second(NewRegistry(Config{Profile: profile(t), Limits: l}, clock)), ErrInvalid)
		}
		var l Limits
		set(&l, 1)
		if r, err := NewRegistry(Config{Profile: profile(t), Limits: l}, clock); err != nil {
			t.Errorf("limit %d of 1 refused", index)
		} else if r.limits == defaults {
			t.Errorf("limit %d of 1 ignored", index)
		}
	}
	for _, ttl := range []time.Duration{-1, most.StagingTTL + 1} {
		expect(t, "ttl", second(NewRegistry(Config{Profile: profile(t), Limits: Limits{StagingTTL: ttl}}, clock)), ErrInvalid)
	}
	if r, _ := NewRegistry(Config{Profile: profile(t), Limits: Limits{StagingTTL: time.Minute}}, clock); r.limits.StagingTTL != time.Minute {
		t.Error("ttl ignored")
	}
	expect(t, "clock", second(NewRegistry(Config{Profile: profile(t)}, nil)), ErrInvalid)
	expect(t, "profile", second(NewRegistry(Config{Profile: decimal.Profile{}}, clock)), ErrInvalid)
	if !errors.Is(Refusal{Reason: "x"}, ErrRejected) || (Refusal{Reason: "x"}).Error() != ErrRejected.Error()+": x" {
		t.Error("refusal")
	}
}

func TestForms(t *testing.T) {
	f := setup(t)
	good := Spec{Tenant: "t1", Dataset: "usage", Version: "v1", Rows: 1, Digest: digestOf([][]Row{{row("a", "g", "KRW", "1")}})}
	bad := map[string]func(*Spec){
		"tenant": func(s *Spec) { s.Tenant = "" }, "dataset": func(s *Spec) { s.Dataset = "a b" }, "version": func(s *Spec) { s.Version = "" },
		"rows": func(s *Spec) { s.Rows = 0 }, "many": func(s *Spec) { s.Rows = f.r.limits.VersionRows + 1 }, "digest": func(s *Spec) { s.Digest = "x" },
	}
	for name, mutate := range bad {
		s := good
		mutate(&s)
		expect(t, name, second(f.r.Begin(s)), ErrInvalid)
	}
	edge := good
	edge.Rows = f.r.limits.VersionRows
	f.must(second(f.r.Begin(edge)))
	f.must(f.r.Abort("t1", "usage", "v1"))
	for _, c := range []string{"KRW", "AAA", "ZZZ"} {
		if !validCurrency(c) {
			t.Errorf("currency %s", c)
		}
	}
	for _, c := range []string{"", "KR", "krw", "K1W", "KRWX", "@AA", "[AA", "A@A", "AA["} {
		if validCurrency(c) {
			t.Errorf("currency %q accepted", c)
		}
	}
	if !validHex(digest(nil)) || validHex("") || validHex(digest(nil)[:63]+"G") || validHex("g"+digest(nil)[1:]) {
		t.Error("hex")
	}
	long := "a"
	for len(long) < 128 {
		long += "b"
	}
	for _, id := range []string{"A", "Z", "a", "z", "0", "9", "a.b_c:d-e", long} {
		if !validID(id) {
			t.Errorf("id %q refused", id)
		}
	}
	for _, id := range []string{"", "!a", " a", "a b", "a@", "a[", "a`", "a{", "a/", long + "c"} {
		if validID(id) {
			t.Errorf("id %q accepted", id)
		}
	}
	for _, args := range [][3]string{{"", "d", "v1"}, {"t", "", "v1"}, {"t", "d", ""}} {
		expect(t, "version form", second(f.r.State(args[0], args[1], args[2])), ErrInvalid)
	}
	rowsBad := map[string][]Row{
		"key":       {row("", "g", "KRW", "1")},
		"group":     {row("a", "", "KRW", "1")},
		"currency":  {row("a", "g", "krw", "1")},
		"amount":    {row("a", "g", "KRW", "x")},
		"canonical": {row("a", "g", "KRW", "1.50")},
		"zero":      {row("a", "g", "KRW", "0.0")},
		"order":     {row("b", "g", "KRW", "1"), row("a", "g", "KRW", "1")},
		"repeat":    {row("a", "g", "KRW", "1"), row("a", "g", "KRW", "1")},
		"input":     {row("a", "g", "KRW", "1234567")},
		"scale":     {row("a", "g", "KRW", "0.00001")},
	}
	for name, list := range rowsBad {
		if _, ok := f.r.parse(list); ok {
			t.Errorf("rows %s accepted", name)
		}
	}
	if _, ok := f.r.parse([]Row{row("a", "g", "KRW", "-999999.9999"), row("b", "g", "KRW", "0")}); !ok {
		t.Error("edge amounts refused")
	}
}
