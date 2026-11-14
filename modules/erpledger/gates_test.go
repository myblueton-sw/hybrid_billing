package erpledger

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func TestRegistryForms(t *testing.T) {
	clock := &testClock{now: start}
	one := Limits{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	must(NewRegistry(testNumbers(t), one, clock.read))
	must(NewRegistry(testNumbers(t), Limits{100000, 10000, 1000000, 1000000, 1000000, 10000, 1000000, 10000, 10000000, 10000, 1000000}, clock.read))
	if _, err := NewRegistry(decimal.Profile{}, one, clock.read); err != ErrInvalid {
		t.Fatal(err)
	}
	if _, err := NewRegistry(testNumbers(t), one, nil); err != ErrInvalid {
		t.Fatal(err)
	}
	maxes := []int{100000, 10000, 1000000, 1000000, 1000000, 10000, 1000000, 10000, 10000000, 10000, 1000000}
	for i := range maxes {
		for _, v := range []int{0, maxes[i] + 1} {
			l := []int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
			l[i] = v
			lim := Limits{l[0], l[1], l[2], l[3], l[4], l[5], l[6], l[7], l[8], l[9], l[10]}
			if _, err := NewRegistry(testNumbers(t), lim, clock.read); err != ErrInvalid {
				t.Fatalf("%d %d: %v", i, v, err)
			}
		}
	}
	err := refuse(ReasonEmpty)
	if !errors.Is(err, ErrRejected) || err.Error() != ErrRejected.Error()+": empty" {
		t.Fatal(err)
	}
}

func TestConnectionForms(t *testing.T) {
	f := newFixture(t)
	bad := []func(*Connection){
		func(c *Connection) { c.ID = "" },
		func(c *Connection) { c.Entity = "" },
		func(c *Connection) { c.System = "a b" },
		func(c *Connection) { c.Transport = "ftp" },
		func(c *Connection) { c.Capabilities = nil },
		func(c *Connection) { c.Capabilities = []string{"delete"} },
		func(c *Connection) { c.Capabilities = []string{CommandPost, CommandPost} },
		func(c *Connection) { c.Currency = "krw" },
		func(c *Connection) { c.Scale = -1 },
		func(c *Connection) { c.Scale = 9 },
		func(c *Connection) { c.Interval = time.Minute - 1 },
		func(c *Connection) { c.Interval = 24*time.Hour + 1 },
		func(c *Connection) { c.Owner = "" },
		func(c *Connection) { c.Template = "t" },
		func(c *Connection) { c.Transport = TransportFile },
		func(c *Connection) { c.Transport, c.Template = TransportFile, "a b" },
	}
	for i, change := range bad {
		c := apiConnection("x")
		change(&c)
		if got := kind(second(f.r.Connect("w1", c, human("admin")))); got != "invalid" {
			t.Fatalf("case %d: %s", i, got)
		}
	}
	edge := apiConnection("edge")
	edge.Scale, edge.Interval, edge.Capabilities = 8, 24*time.Hour, []string{CommandStatus, CommandConnect}
	v := must(f.r.Connect("w1", edge, human("admin")))
	if strings.Join(v.Connection.Capabilities, ",") != "connect,get_status" {
		t.Fatalf("sorted %v", v.Connection.Capabilities)
	}
	if again := must(f.r.Connect("w1", edge, human("admin"))); again.At != v.At {
		t.Fatal("repeat")
	}
	v.Connection.Capabilities[0] = "x"
	if must(f.r.ConnectionOf("w1", "edge")).Connection.Capabilities[0] != "connect" {
		t.Fatal("aliasing")
	}
	low := apiConnection("low")
	low.Interval = time.Minute
	must(f.r.Connect("w1", low, human("admin")))
	must(f.r.Connect("w1", apiConnection("more"), human("admin")))
	expect(t, "limit", second(f.r.Connect("w1", apiConnection("most"), human("admin"))), "limit")
	for _, change := range []func(*Connection){
		func(c *Connection) { c.Entity = "x" }, func(c *Connection) { c.System = "x" }, func(c *Connection) { c.Capabilities = []string{CommandConnect} },
		func(c *Connection) { c.Currency = "USD" }, func(c *Connection) { c.Scale = 2 }, func(c *Connection) { c.Interval = 2 * time.Hour },
		func(c *Connection) { c.Owner = "x" }, func(c *Connection) { c.Transport, c.Template = TransportFile, "t" },
	} {
		c := apiConnection("api")
		change(&c)
		expect(t, "changed", second(f.r.Connect("w1", c, human("admin"))), "conflict")
	}
	ft := fileConnection("file")
	g := newFixture(t)
	must(g.r.Connect("w1", ft, human("admin")))
	ft.Template = "other"
	expect(t, "template changed", second(g.r.Connect("w1", ft, human("admin"))), "conflict")
	expect(t, "other author", second(f.r.Connect("w1", apiConnection("api"), human("x"))), "conflict")
	expect(t, "system repeat", second(f.r.Connect("w1", apiConnection("api"), system("admin"))), "conflict")
	expect(t, "assistant", second(f.r.Connect("w2", apiConnection("api"), assistant("bot"))), ReasonAINotAllowed)
	expect(t, "system", second(f.r.Connect("w2", apiConnection("api"), system("job"))), ReasonNotHuman)
	expect(t, "ws", second(f.r.Connect("", apiConnection("api"), human("a"))), "invalid")
	expect(t, "actor", second(f.r.Connect("w1", apiConnection("api"), Actor{ID: "a", Kind: "robot"})), "invalid")
	must(f.r.Connect("w2", apiConnection("api"), human("a")))
	must(f.r.Connect("w3", apiConnection("api"), human("a")))
	expect(t, "workspaces", second(f.r.Connect("w4", apiConnection("api"), human("a"))), "limit")
	expect(t, "connection of", second(f.r.ConnectionOf("w1", "nope")), "not_found")
	expect(t, "connection of ws", second(f.r.ConnectionOf("w9", "api")), "not_found")
}

func TestVerifyTemplateGates(t *testing.T) {
	f := newFixture(t)
	must(f.r.Connect("w1", fileConnection("file"), human("admin")))
	expect(t, "form", second(f.r.VerifyTemplate("w1", "file", "", human("qa"))), "invalid")
	expect(t, "ws", second(f.r.VerifyTemplate("", "file", "e", human("qa"))), "invalid")
	expect(t, "id", second(f.r.VerifyTemplate("w1", "", "e", human("qa"))), "invalid")
	expect(t, "actor", second(f.r.VerifyTemplate("w1", "file", "e", human(""))), "invalid")
	expect(t, "missing", second(f.r.VerifyTemplate("w1", "nope", "e", human("qa"))), "not_found")
	expect(t, "missing ws", second(f.r.VerifyTemplate("w9", "file", "e", human("qa"))), "not_found")
	expect(t, "assistant", second(f.r.VerifyTemplate("w1", "file", "e", assistant("bot"))), ReasonAINotAllowed)
	expect(t, "api", second(f.r.VerifyTemplate("w1", "api", "e", human("qa"))), ReasonUnsupported)
	v := must(f.r.VerifyTemplate("w1", "file", "e", human("qa")))
	if !v.Verified || v.Evidence != "e" || v.Verifier != "qa" {
		t.Fatalf("verified %+v", v)
	}
	must(f.r.VerifyTemplate("w1", "file", "e", human("qa")))
	expect(t, "other evidence", second(f.r.VerifyTemplate("w1", "file", "e2", human("qa"))), "conflict")
	expect(t, "other verifier", second(f.r.VerifyTemplate("w1", "file", "e", human("qb"))), "conflict")
}

func TestMappingGates(t *testing.T) {
	f := newFixture(t)
	good := Mapping{"api", "customer", "beta", "C-200"}
	for i, m := range []Mapping{{"", "customer", "b", "x"}, {"api", "account", "b", "x"}, {"api", "customer", "", "x"}, {"api", "customer", "b", ""}} {
		if got := kind(second(f.r.Map("w1", m, human("a")))); got != "invalid" {
			t.Fatalf("case %d: %s", i, got)
		}
	}
	expect(t, "ws", second(f.r.Map("", good, human("a"))), "invalid")
	expect(t, "actor", second(f.r.Map("w1", good, human(""))), "invalid")
	expect(t, "unknown", second(f.r.Map("w1", Mapping{"nope", "customer", "b", "x"}, human("a"))), ReasonUnknownConnection)
	expect(t, "unknown ws", second(f.r.Map("w9", good, human("a"))), ReasonUnknownConnection)
	expect(t, "assistant", second(f.r.Map("w1", good, assistant("bot"))), ReasonAINotAllowed)
	v := must(f.r.Map("w1", good, human("a")))
	if v.Revision != 1 {
		t.Fatalf("rev %+v", v)
	}
	if again := must(f.r.Map("w1", good, human("a"))); again.Revision != 1 {
		t.Fatal("repeat")
	}
	expect(t, "other same", second(f.r.Map("w1", good, human("b"))), "conflict")
	good.External = "C-201"
	if v := must(f.r.Map("w1", good, human("b"))); v.Revision != 2 {
		t.Fatalf("remap %+v", v)
	}
	narrow := apiConnection("narrow")
	narrow.Capabilities = []string{CommandPrepare}
	must(f.r.Connect("w1", narrow, human("admin")))
	expect(t, "unsupported", second(f.r.Map("w1", Mapping{"narrow", "tax", "vat10", "T"}, human("a"))), ReasonUnsupported)
	for i := 0; i < 14; i++ {
		_, _ = f.r.Map("w1", Mapping{"api", "product", "p" + string(rune('a'+i)), "x"}, human("a"))
	}
	expect(t, "limit", second(f.r.Map("w1", Mapping{"api", "product", "zz", "x"}, human("a"))), "limit")
}

func TestAnchorAndSurveyGates(t *testing.T) {
	f := newFixture(t)
	good := Anchor{ID: "a1", Connection: "api", Kind: AnchorContract, External: "K-1", Customer: "acme", Sources: []string{"s1", "s2"}}
	bad := []func(*Anchor){
		func(a *Anchor) { a.ID = "" }, func(a *Anchor) { a.Connection = "" }, func(a *Anchor) { a.Kind = "invoice" },
		func(a *Anchor) { a.External = "" }, func(a *Anchor) { a.Customer = "" }, func(a *Anchor) { a.Sources = nil },
		func(a *Anchor) { a.Sources = []string{"s1", "s1"} }, func(a *Anchor) { a.Sources = []string{""} },
	}
	for i, change := range bad {
		a := good
		a.Sources = append([]string(nil), good.Sources...)
		change(&a)
		if got := kind(second(f.r.Anchor("w1", a, human("a")))); got != "invalid" {
			t.Fatalf("case %d: %s", i, got)
		}
	}
	expect(t, "ws", second(f.r.Anchor("", good, human("a"))), "invalid")
	expect(t, "actor", second(f.r.Anchor("w1", good, human(""))), "invalid")
	expect(t, "unknown", second(f.r.Anchor("w9", good, human("a"))), ReasonUnknownConnection)
	expect(t, "assistant", second(f.r.Anchor("w1", good, assistant("bot"))), ReasonAINotAllowed)
	v := must(f.r.Anchor("w1", good, human("a")))
	v.Sources[0] = "x"
	if again := must(f.r.Anchor("w1", good, human("a"))); again.Sources[0] != "s1" {
		t.Fatal("aliasing")
	}
	expect(t, "other author", second(f.r.Anchor("w1", good, human("b"))), "conflict")
	for _, change := range []func(*Anchor){
		func(a *Anchor) { a.Kind = AnchorPrepaid }, func(a *Anchor) { a.External = "x" }, func(a *Anchor) { a.Customer = "x" },
		func(a *Anchor) { a.Sources = []string{"s1"} },
	} {
		a := good
		a.Sources = append([]string(nil), good.Sources...)
		change(&a)
		expect(t, "changed", second(f.r.Anchor("w1", a, human("a"))), "conflict")
	}
	f.connect(t, apiConnection("api2"))
	a := good
	a.Connection = "api2"
	expect(t, "connection changed", second(f.r.Anchor("w1", a, human("a"))), "conflict")
	for i := 2; i <= 6; i++ {
		_, _ = f.r.Anchor("w1", Anchor{ID: "a" + string(rune('0'+i)), Connection: "api", Kind: AnchorPayable, External: "x", Customer: "acme", Sources: []string{"z"}}, human("a"))
	}
	expect(t, "limit", second(f.r.Anchor("w1", Anchor{ID: "a9", Connection: "api", Kind: AnchorPayable, External: "x", Customer: "acme", Sources: []string{"z"}}, human("a"))), "limit")
	sv := Survey{Connection: "api", Customer: "acme", Period: "2026-12", Outcome: SurveyComplete, Evidence: "e"}
	for i, s := range []Survey{
		{Connection: "", Customer: "acme", Period: "2026-12", Outcome: SurveyComplete, Evidence: "e"},
		{Connection: "api", Customer: "", Period: "2026-12", Outcome: SurveyComplete, Evidence: "e"},
		{Connection: "api", Customer: "acme", Period: "2026-13", Outcome: SurveyComplete, Evidence: "e"},
		{Connection: "api", Customer: "acme", Period: "2026-12", Outcome: "maybe", Evidence: "e"},
		{Connection: "api", Customer: "acme", Period: "2026-12", Outcome: SurveyComplete, Evidence: ""},
	} {
		if got := kind(second(f.r.Survey("w1", s, human("a")))); got != "invalid" {
			t.Fatalf("survey %d: %s", i, got)
		}
	}
	expect(t, "survey ws", second(f.r.Survey("", sv, human("a"))), "invalid")
	expect(t, "survey actor", second(f.r.Survey("w1", sv, human(""))), "invalid")
	expect(t, "survey unknown", second(f.r.Survey("w9", sv, human("a"))), ReasonUnknownConnection)
	expect(t, "survey assistant", second(f.r.Survey("w1", sv, assistant("bot"))), ReasonAINotAllowed)
	first := must(f.r.Survey("w1", sv, human("a")))
	f.clock.advance(time.Minute)
	if again := must(f.r.Survey("w1", sv, human("a"))); again.At != first.At {
		t.Fatal("survey repeat")
	}
	if other := must(f.r.Survey("w1", sv, human("b"))); other.Author != "b" {
		t.Fatal("survey other")
	}
}

func TestPrepareForms(t *testing.T) {
	f := newFixture(t)
	good := invoice("k", "api", line("l1", "s1", "1"))
	bad := []func(*Submission){
		func(s *Submission) { s.Key = "" }, func(s *Submission) { s.Connection = "" }, func(s *Submission) { s.Purpose = "credit" },
		func(s *Submission) { s.Customer = "" }, func(s *Submission) { s.Period = "2026-1" }, func(s *Submission) { s.Path = "x" },
		func(s *Submission) { s.Anchor = "a" }, func(s *Submission) { s.Path = PathOrderBased },
		func(s *Submission) { s.Path, s.Anchor, s.Purpose = PathOrderBased, "a", PurposePayable },
		func(s *Submission) { s.Path, s.Anchor = PathOrderBased, "a b" },
		func(s *Submission) { s.Lines = []Line{line("l1", "s1", "1"), line("l1", "s2", "1")} },
		func(s *Submission) { s.Lines = []Line{line("l1", "s1", "1"), line("l2", "s1", "1")} },
		func(s *Submission) {
			s.Lines = []Line{{ID: "", Source: "s", Product: "p", TaxCode: "t", Quantity: "1", Net: "1", Tax: "0"}}
		},
		func(s *Submission) {
			s.Lines = []Line{{ID: "l", Source: "", Product: "p", TaxCode: "t", Quantity: "1", Net: "1", Tax: "0"}}
		},
		func(s *Submission) {
			s.Lines = []Line{{ID: "l", Source: "s", Product: "", TaxCode: "t", Quantity: "1", Net: "1", Tax: "0"}}
		},
		func(s *Submission) {
			s.Lines = []Line{{ID: "l", Source: "s", Product: "p", TaxCode: "", Quantity: "1", Net: "1", Tax: "0"}}
		},
		func(s *Submission) {
			s.Lines = []Line{{ID: "l", Source: "s", Product: "p", TaxCode: "t", Quantity: "x", Net: "1", Tax: "0"}}
		},
		func(s *Submission) {
			s.Lines = []Line{{ID: "l", Source: "s", Product: "p", TaxCode: "t", Quantity: "1", Net: "x", Tax: "0"}}
		},
		func(s *Submission) {
			s.Lines = []Line{{ID: "l", Source: "s", Product: "p", TaxCode: "t", Quantity: "1", Net: "1", Tax: "x"}}
		},
		func(s *Submission) { s.Lines = []Line{line("l1", "s1", "1.5")} },
	}
	for i, change := range bad {
		s := good
		change(&s)
		if got := kind(second(f.r.Prepare("w1", s, human("b")))); got != "invalid" && i < len(bad)-1 {
			t.Fatalf("case %d: %s", i, got)
		}
	}
	scaled := good
	scaled.Lines = []Line{line("l1", "s1", "1.5")}
	expect(t, "scale", second(f.r.Prepare("w1", scaled, human("b"))), "invalid")
	qty := good
	qty.Lines = []Line{{ID: "l1", Source: "s1", Product: "compute", TaxCode: "vat10", Quantity: "0.5", Net: "1", Tax: "0"}}
	expect(t, "qty scale", second(f.r.Prepare("w1", qty, human("b"))), "invalid")
	tax := good
	tax.Lines = []Line{{ID: "l1", Source: "s1", Product: "compute", TaxCode: "vat10", Quantity: "1", Net: "1", Tax: "0.5"}}
	expect(t, "tax scale", second(f.r.Prepare("w1", tax, human("b"))), "invalid")
	expect(t, "ws", second(f.r.Prepare("", good, human("b"))), "invalid")
	expect(t, "actor", second(f.r.Prepare("w1", good, human(""))), "invalid")
	expect(t, "unknown connection", second(f.r.Prepare("w1", invoice("k", "nope", line("l1", "s1", "1")), human("b"))), ReasonUnknownConnection)
	expect(t, "unknown ws", second(f.r.Prepare("w9", good, human("b"))), ReasonUnknownConnection)
	expect(t, "assistant", second(f.r.Prepare("w1", good, assistant("bot"))), ReasonAINotAllowed)
	expect(t, "system", second(f.r.Prepare("w1", good, system("job"))), ReasonNotHuman)
	empty := good
	empty.Lines = nil
	expect(t, "empty", second(f.r.Prepare("w1", empty, human("b"))), ReasonEmpty)
	for _, m := range []Submission{
		{Key: "k", Connection: "api", Purpose: PurposeInvoice, Customer: "beta", Period: "2026-10", Path: PathDirect, Lines: []Line{line("l1", "s1", "1")}},
		invoice("k", "api", Line{ID: "l1", Source: "s1", Product: "gpu", TaxCode: "vat10", Quantity: "1", Net: "1", Tax: "0"}),
		invoice("k", "api", Line{ID: "l1", Source: "s1", Product: "compute", TaxCode: "vat0", Quantity: "1", Net: "1", Tax: "0"}),
	} {
		if m.Customer == "beta" {
			must(f.r.Survey("w1", Survey{Connection: "api", Customer: "beta", Period: "2026-10", Outcome: SurveyComplete, Evidence: "e"}, human("a")))
		}
		expect(t, "mapping", second(f.r.Prepare("w1", m, human("b"))), ReasonMappingMissing)
	}
	narrow := apiConnection("narrow")
	narrow.Capabilities = []string{CommandSubmit}
	must(f.r.Connect("w1", narrow, human("admin")))
	expect(t, "unsupported", second(f.r.Prepare("w1", invoice("k", "narrow", line("l1", "s1", "1")), human("b"))), ReasonUnsupported)
	five := good
	five.Lines = []Line{line("1", "a", "1"), line("2", "b", "1"), line("3", "c", "1"), line("4", "d", "1"), line("5", "e", "1")}
	expect(t, "lines", second(f.r.Prepare("w1", five, human("b"))), "limit")
	big := good
	big.Lines = []Line{line("1", "a", "99999999999999"), line("2", "b", "99999999999999"), line("3", "c", "99999999999999"), line("4", "d", "99999999999999")}
	v := must(f.r.Prepare("w1", big, human("b")))
	if v.Total.Net != "399999999999996" || v.State != StatePrepared {
		t.Fatalf("totals %+v", v)
	}
	v.Submission.Lines[0].Net = "x"
	v.Mappings[0] = "x"
	got := must(f.r.SubmissionOf("w1", "k"))
	if got.Submission.Lines[0].Net == "x" || got.Mappings[0] == "x" {
		t.Fatal("aliasing")
	}
	for _, change := range []func(*Submission){
		func(s *Submission) { s.Connection = "narrow" }, func(s *Submission) { s.Purpose = PurposePayable },
		func(s *Submission) { s.Customer = "beta" }, func(s *Submission) { s.Period = "2026-11" },
		func(s *Submission) { s.Path, s.Anchor = PathOrderBased, "a" }, func(s *Submission) { s.Lines = s.Lines[:3] },
		func(s *Submission) { s.Lines = append([]Line(nil), s.Lines...); s.Lines[0].Tax = "1" },
	} {
		s := big
		change(&s)
		expect(t, "changed key", second(f.r.Prepare("w1", s, human("b"))), "conflict")
	}
	expect(t, "other author", second(f.r.Prepare("w1", big, human("c"))), "conflict")
	expect(t, "system repeat", second(f.r.Prepare("w1", big, system("b"))), "conflict")
	expect(t, "submission of", second(f.r.SubmissionOf("w1", "nope")), "not_found")
	expect(t, "submission of ws", second(f.r.SubmissionOf("w9", "k")), "not_found")
}

func TestPrepareAnchorsAndLimits(t *testing.T) {
	f := newFixture(t)
	must(f.r.Anchor("w1", Anchor{ID: "ap", Connection: "api", Kind: AnchorPayable, External: "AP", Customer: "acme", Sources: []string{"s1"}}, human("a")))
	must(f.r.Anchor("w1", Anchor{ID: "other", Connection: "api", Kind: AnchorSalesOrder, External: "SO", Customer: "beta", Sources: []string{"s2"}}, human("a")))
	f.connect(t, apiConnection("api2"))
	must(f.r.Anchor("w1", Anchor{ID: "far", Connection: "api2", Kind: AnchorSalesOrder, External: "SO2", Customer: "acme", Sources: []string{"s3"}}, human("a")))
	must(f.r.Anchor("w1", Anchor{ID: "so", Connection: "api", Kind: AnchorSalesOrder, External: "SO3", Customer: "acme", Sources: []string{"s4"}}, human("a")))
	for _, anchor := range []string{"nope", "ap", "other", "far"} {
		s := invoice("k", "api", line("l1", "s9", "1"))
		s.Path, s.Anchor = PathOrderBased, anchor
		expect(t, "anchor "+anchor, second(f.r.Prepare("w1", s, human("b"))), ReasonUnknownAnchor)
	}
	s := invoice("k", "api", line("l1", "s4", "1"), line("l2", "s3", "1"))
	s.Path, s.Anchor = PathOrderBased, "so"
	expect(t, "other anchor still blocks", second(f.r.Prepare("w1", s, human("b"))), ReasonAlreadyReflected)
	s.Lines = s.Lines[:1]
	must(f.r.Prepare("w1", s, human("b")))
	for i := 0; i < 11; i++ {
		_, _ = f.r.Prepare("w1", invoice("x"+string(rune('a'+i)), "api", line("l1", "t"+string(rune('a'+i)), "1")), human("b"))
	}
	expect(t, "limit", second(f.r.Prepare("w1", invoice("zz", "api", line("l1", "zz", "1")), human("b"))), "limit")
}
