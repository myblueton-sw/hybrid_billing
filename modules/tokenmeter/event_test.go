package tokenmeter

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func with(base Event, change func(*Event)) Event {
	made := base
	if base.Cost != nil {
		cost := *base.Cost
		made.Cost = &cost
	}
	change(&made)
	return made
}

func TestEventRefusals(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.Batch = 4 })
	spec := export("r1", "s1", 10)
	spec.Events = 5
	h.open(spec)
	good := event("e1", 10)
	malformed := map[string]func(*Event){
		"id":                          func(e *Event) { e.ID = "" },
		"request":                     func(e *Event) { e.Request = "" },
		"attempt":                     func(e *Event) { e.Attempt = "" },
		"consumer":                    func(e *Event) { e.Consumer = "kim@example" },
		"provider":                    func(e *Event) { e.Provider = "" },
		"model":                       func(e *Event) { e.Model = "" },
		"a version with a space":      func(e *Event) { e.Version = "v 1" },
		"an endpoint with a slash":    func(e *Event) { e.Endpoint = "v1/chat" },
		"no time":                     func(e *Event) { e.Time = time.Time{} },
		"a time with a fraction":      func(e *Event) { e.Time = e.Time.Add(time.Millisecond) },
		"outcome":                     func(e *Event) { e.Outcome = "ok" },
		"no outcome":                  func(e *Event) { e.Outcome = "" },
		"usage grade":                 func(e *Event) { e.Usage = "" },
		"cache relation":              func(e *Event) { e.Cache = "" },
		"a count with a point":        func(e *Event) { e.Input = "1.5" },
		"a count with a leading zero": func(e *Event) { e.Output = "01" },
		"a negative count":            func(e *Event) { e.CacheRead = "-1" },
		"a count of nineteen digits":  func(e *Event) { e.CacheWrite = "1234567890123456789" },
		"an amount that is no number": func(e *Event) { e.Cost.Amount = "NaN" },
		"an amount with a sign":       func(e *Event) { e.Cost.Amount = "+1" },
		"no amount":                   func(e *Event) { e.Cost.Amount = "" },
		"a currency":                  func(e *Event) { e.Cost.Currency = "usd" },
		"a grade":                     func(e *Event) { e.Cost.Grade = "paid" },
		"no grade":                    func(e *Event) { e.Cost.Grade = "" },
		"an origin":                   func(e *Event) { e.Cost.Origin = "cloud" },
		"invoiced without a document": func(e *Event) { e.Cost.Document = "" },
		"a document that is text":     func(e *Event) { e.Cost.Document = "invoice 1" },
		"invoiced and self-hosted":    func(e *Event) { e.Cost.Origin = SelfHostedGPU },
		"invoiced and the fee":        func(e *Event) { e.Cost.Origin = TokenMeterFee },
		"a document on a billed cost": func(e *Event) { e.Cost.Grade = Billed },
		"a grade without a document":  func(e *Event) { e.Cost.Grade, e.Cost.Document = "paid", "" },
		"no grade and no document":    func(e *Event) { e.Cost.Grade, e.Cost.Document = "", "" },
		"an origin of a billed cost":  func(e *Event) { e.Cost.Grade, e.Cost.Document, e.Cost.Origin = Billed, "", "cloud" },
		"no origin":                   func(e *Event) { e.Cost.Grade, e.Cost.Document, e.Cost.Origin = Billed, "", "" },
	}
	for name, change := range malformed {
		before := h.r.dump()
		_, err := h.r.Post("ws1", "r1", []Event{with(good, change)})
		wantErr(t, err, ErrInvalid, name)
		h.unchanged(before, name)
	}
	other := with(event("e2", 10), func(e *Event) { e.Request, e.Attempt = good.Request, good.Attempt })
	for name, batch := range map[string][]Event{
		"an empty batch":                 nil,
		"an event twice in a batch":      {good, good},
		"a pair twice in a batch":        {good, other},
		"a malformed event after a good": {good, with(event("e2", 10), func(e *Event) { e.Model = "" })},
	} {
		before := h.r.dump()
		_, err := h.r.Post("ws1", "r1", batch)
		wantErr(t, err, ErrInvalid, name)
		h.unchanged(before, name)
	}
	for _, c := range []struct {
		ws, id string
		want   error
	}{{"ws2", "r1", ErrNotFound}, {"ws1", "r9", ErrNotFound}, {"", "r1", ErrInvalid}, {"ws1", "", ErrInvalid}} {
		_, err := h.r.Post(c.ws, c.id, []Event{good})
		wantErr(t, err, c.want, "post")
	}
	_, err := h.r.Post("ws1", "r9", nil)
	wantErr(t, err, ErrNotFound, "the reception is looked up first")
	five := []Event{event("a", 10), event("b", 10), event("c", 10), event("d", 10), event("e", 10)}
	_, err = h.r.Post("ws1", "r1", five)
	wantErr(t, err, ErrLimit, "a batch over the limit")
	_, err = h.r.Post("ws1", "r1", append(five[:4:4], with(event("f", 10), func(e *Event) { e.Model = "" })))
	wantErr(t, err, ErrInvalid, "a malformed event in a batch over the limit")
	bare := with(good, func(e *Event) {
		e.Version, e.Endpoint, e.Cost = "", "", nil
		e.Input, e.Output, e.CacheRead, e.CacheWrite = "", "", "", ""
	})
	h.post("r1", bare)
	before := h.r.dump()
	h.post("r1", bare)
	for name, change := range map[string]func(*Event){
		"a count known":   func(e *Event) { e.Input = "0" },
		"a version":       func(e *Event) { e.Version = "v1" },
		"an endpoint":     func(e *Event) { e.Endpoint = "chat" },
		"the time":        func(e *Event) { e.Time = e.Time.Add(time.Second) },
		"the consumer":    func(e *Event) { e.Consumer = "user-2" },
		"the provider":    func(e *Event) { e.Provider = "prov-b" },
		"the model":       func(e *Event) { e.Model = "model-y" },
		"the outcome":     func(e *Event) { e.Outcome = Failed },
		"the usage grade": func(e *Event) { e.Usage = UsageEstimated },
		"the relation":    func(e *Event) { e.Cache = Included },
		"a cost":          func(e *Event) { e.Cost = &Cost{Amount: "1", Currency: "USD", Grade: Billed, Origin: Provider} },
		"the request":     func(e *Event) { e.Request = "req-other" },
		"the attempt":     func(e *Event) { e.Attempt = "2" },
		"counts moved":    func(e *Event) { e.Output = ""; e.Input = ""; e.CacheRead = "0" },
	} {
		_, err := h.r.Post("ws1", "r1", []Event{with(bare, change)})
		wantErr(t, err, ErrConflict, name)
	}
	_, err = h.r.Post("ws1", "r1", []Event{other})
	wantErr(t, err, ErrConflict, "one attempt under two event identifiers")
	_, err = h.r.Post("ws1", "r1", append(five[:4:4], with(bare, func(e *Event) { e.Outcome = Failed })))
	wantErr(t, err, ErrLimit, "a batch over the limit before its conflict is looked at")
	_, err = h.r.Post("ws1", "r1", []Event{event("e2", 10), other})
	wantErr(t, err, ErrInvalid, "the same pair twice in a batch is malformed before it is a conflict")
	_, err = h.r.Post("ws1", "r1", []Event{event("e3", 10), with(bare, func(e *Event) { e.Outcome = Failed })})
	wantErr(t, err, ErrConflict, "a batch with one conflicting event is refused whole")
	h.unchanged(before, "repeats and conflicts")

	costly := priced("e2", 10, "1.50", Billed, Provider)
	h.post("r1", costly)
	before = h.r.dump()
	for name, change := range map[string]func(*Event){
		"another spelling of the amount": func(e *Event) { e.Cost.Amount = "1.5" },
		"the currency":                   func(e *Event) { e.Cost.Currency = "EUR" },
		"the grade":                      func(e *Event) { e.Cost.Grade = Computed },
		"the origin":                     func(e *Event) { e.Cost.Origin = SelfHostedGPU },
		"a document":                     func(e *Event) { e.Cost.Grade, e.Cost.Document = Invoiced, "inv-9" },
		"no cost":                        func(e *Event) { e.Cost = nil },
	} {
		_, err := h.r.Post("ws1", "r1", []Event{with(costly, change)})
		wantErr(t, err, ErrConflict, name)
	}
	zero := with(event("e3", 10), func(e *Event) { e.Cost.Amount, e.Model, e.Version = "0", "m:x", "y" })
	caller := with(zero, func(*Event) {})
	h.post("r1", caller)
	caller.Cost.Amount = "999"
	_, err = h.r.Post("ws1", "r1", []Event{with(zero, func(e *Event) { e.Cost.Document = "inv-2" })})
	wantErr(t, err, ErrConflict, "another document")
	_, err = h.r.Post("ws1", "r1", []Event{with(zero, func(e *Event) { e.Model, e.Version = "m", "x:y" })})
	wantErr(t, err, ErrConflict, "a colon that moved between two fields")
	_, err = h.r.Post("ws1", "r1", []Event{with(zero, func(e *Event) { e.Cost.Amount = "-0" })})
	wantErr(t, err, ErrConflict, "a negative zero against a zero")
	before = h.r.dump()
	fee := priced("e4", 10, "5", Billed, TokenMeterFee)
	_, err = h.r.Post("ws1", "r1", []Event{event("e5", 10), fee})
	wantErr(t, err, ErrRejected, "the fee of TokenMeter itself")
	h.unchanged(before, "a batch with a fee row")
	h.post("r1", event("e4", 10), bare, zero)
	if view := h.view("r1"); view.Events != 4 {
		t.Fatalf("a batch partly reported before adds only what is new: %d", view.Events)
	}
	before = h.r.dump()
	_, err = h.r.Post("ws1", "r1", []Event{event("e5", 10), event("e6", 10)})
	wantErr(t, err, ErrRejected, "events beyond the declared number")
	h.unchanged(before, "too many events")
	h.post("r1", event("e5", 10))
	h.close("r1")
	before = h.r.dump()
	if state, err := h.r.Post("ws1", "r1", []Event{bare, zero}); err != nil || state != Closed {
		t.Fatal(state, err)
	}
	if kept := h.events("r1"); kept[2].Event.ID != "e3" || kept[2].Event.Cost.Amount != "0" {
		t.Fatalf("a stored event shares the cost of the caller: %+v", kept[2].Event.Cost)
	}
	_, err = h.r.Post("ws1", "r1", []Event{bare, event("e9", 10)})
	wantErr(t, err, ErrRejected, "a new event for a closed reception")
	h.unchanged(before, "after closing")

	bounded := newHarness(t, nil)
	wide := export("r1", "s1", 10)
	wide.Events, wide.Costs = 20, map[string]string{"USD": "1", "EUR": "1"}
	bounded.open(wide)
	money := func(id, currency string) Event {
		return with(priced(id, 10, "1", Billed, Provider), func(e *Event) { e.Cost.Currency = currency })
	}
	bounded.post("r1", money("a", "KRW"), money("b", "KRW"), money("c", "USD"))
	before = bounded.r.dump()
	_, err = bounded.r.Post("ws1", "r1", []Event{money("d", "JPY")})
	wantErr(t, err, ErrRejected, "a fourth currency among declared ones and events")
	paper := func(id, document string) Event {
		return with(event(id, 10), func(e *Event) { e.Cost.Document = document })
	}
	bounded.post("r1", paper("d1", "inv-1"), paper("d2", "inv-2"), paper("d3", "inv-3"), paper("d4", "inv-4"), paper("d5", "inv-1"))
	before = bounded.r.dump()
	_, err = bounded.r.Post("ws1", "r1", []Event{paper("d6", "inv-4"), paper("d7", "inv-5")})
	wantErr(t, err, ErrRejected, "a fifth document")
	bounded.unchanged(before, "the limits of currencies and documents")
	bounded.post("r1", paper("d8", "inv-4"), money("d9", "USD"))
}

func TestFindingsAndClasses(t *testing.T) {
	from, to := month(10)
	at := func(id string, when time.Time) Event {
		return with(event(id, 10), func(e *Event) { e.Time = when })
	}
	tokens := func(id string, relation Relation, input, cached string) Event {
		return with(event(id, 10), func(e *Event) { e.Cache, e.Input, e.CacheRead = relation, input, cached })
	}
	cases := []struct {
		name    string
		numeric Numeric
		event   Event
		want    []string
	}{
		{"at the start of the period", Decimal, at("e", from), nil},
		{"a second before the end", Decimal, at("e", to.Add(-time.Second)), nil},
		{"at the end of the period", Decimal, at("e", to), []string{OutsidePeriod}},
		{"a second before the start", Decimal, at("e", from.Add(-time.Second)), []string{OutsidePeriod}},
		{"the same instant in another zone", Decimal, at("e", from.In(time.FixedZone("kst", 9*3600))), nil},
		{"cache larger than the input that includes it", Decimal, tokens("e", Included, "10", "50"), []string{CacheRelationBroken}},
		{"cache known beside an input that is not", Decimal, tokens("e", Included, "", "5"), []string{CacheRelationBroken}},
		{"cache equal to the input", Decimal, tokens("e", Included, "10", "10"), nil},
		{"cache not known", Decimal, tokens("e", Included, "10", ""), nil},
		{"a separate cache larger than the input", Decimal, tokens("e", Separate, "1", "50"), nil},
		{"a relation that is not known", Decimal, tokens("e", RelationNotKnown, "1", "50"), nil},
		{"sixteen digits from a float", Float, priced("e", 10, "1234567890.123456", Billed, Provider), []string{FloatPrecisionExceeded}},
		{"fifteen digits from a float", Float, priced("e", 10, "123456789.123456", Billed, Provider), nil},
		{"sixteen digits from a decimal", Decimal, priced("e", 10, "1234567890.123456", Billed, Provider), nil},
		{"zeros are not significant", Float, priced("e", 10, "100000000000.000000", Billed, Provider), nil},
		{"seven places after the point", Decimal, priced("e", 10, "0.0000001", Billed, Provider), []string{CostOutsideProfile}},
		{"the largest value of the profile", Decimal, priced("e", 10, "99999999999999.999999", Billed, Provider), nil},
		{"the most negative value of the profile", Decimal, priced("e", 10, "-99999999999999.999999", Billed, Provider), nil},
		{"fifteen whole digits", Decimal, priced("e", 10, "100000000000000", Billed, Provider), []string{CostOutsideProfile}},
		{"an exponent inside the profile", Decimal, priced("e", 10, "1e13", Billed, Provider), nil},
		{"an exponent beyond the profile", Decimal, priced("e", 10, "1e14", Billed, Provider), []string{CostOutsideProfile}},
		{"a small exponent inside the profile", Decimal, priced("e", 10, "1E-6", Billed, Provider), nil},
		{"everything at once", Float, with(priced("e", 10, "0.12345678901234567", Billed, Provider), func(e *Event) { e.Time, e.Cache, e.Input, e.CacheRead = to, Included, "1", "2" }),
			[]string{OutsidePeriod, CacheRelationBroken, FloatPrecisionExceeded, CostOutsideProfile}},
	}
	for _, c := range cases {
		h := newHarness(t, nil)
		figures := h.one(func(s *Spec) { s.Numeric = c.numeric }, c.event)
		got := h.events("r1")
		if len(got) != 1 || !reflect.DeepEqual(got[0].Findings, c.want) || got[0].Class == "" || (figures.InError == 1) != (c.want != nil) || got[0].Event.Time.Location() != time.UTC {
			t.Fatalf("%s: %+v, want %v", c.name, got, c.want)
		}
		if c.want != nil {
			got[0].Findings[0] = "changed"
			if again := h.events("r1"); !reflect.DeepEqual(again[0].Findings, c.want) {
				t.Fatalf("%s: a page shares the findings", c.name)
			}
		}
		if summed := len(figures.Costs) == 1; summed != (c.want == nil) {
			t.Fatalf("%s: the cost of an event with a finding entered a sum: %s", c.name, flat(figures))
		}
	}
	classes := []struct {
		grade  Grade
		origin Origin
		want   string
	}{
		{Estimated, Provider, ClassEstimated}, {Estimated, SelfHostedGPU, ClassEstimated}, {Billed, SelfHostedGPU, ClassSelfHosted}, {Computed, SelfHostedGPU, ClassSelfHosted}, {GradeNotKnown, SelfHostedGPU, ClassSelfHosted},
		{Invoiced, Provider, ClassInvoiceReferenced}, {Billed, Provider, ClassUnconfirmed}, {Computed, Provider, ClassUnconfirmed}, {GradeNotKnown, Provider, ClassUnconfirmed},
	}
	for _, c := range classes {
		h := newHarness(t, nil)
		figures := h.one(nil, priced("e", 10, "2.5", c.grade, c.origin), with(priced("f", 10, "9", c.grade, c.origin), func(e *Event) { e.Time = to }))
		got := h.events("r1")
		if got[0].Class != c.want || got[1].Class != c.want || !reflect.DeepEqual(figures.Costs, map[string]map[string]string{c.want: {"USD": "2.5"}}) {
			t.Fatalf("%s from %s: %+v %s", c.grade, c.origin, got, flat(figures))
		}
	}
	h := newHarness(t, nil)
	h.one(nil, with(event("e", 10), func(e *Event) { e.Cost = nil }))
	if got := h.events("r1"); got[0].Class != "" || got[0].Event.Cost != nil {
		t.Fatalf("%+v", got)
	}
}

func TestTheCasesOfTheAcceptance(t *testing.T) {
	h := newHarness(t, nil)
	spec := export("r1", "snap-1", 10)
	spec.Events = 7
	spec.Tokens = map[Kind]string{Input: "415", Output: "56", CacheRead: "120", CacheWrite: "5"}
	spec.Costs = map[string]string{"USD": "8.5", "KRW": "2"}
	h.open(spec)
	attempt := func(id, request, number string, outcome Outcome, usage Usage, relation Relation, counts [4]string, cost *Cost) Event {
		return with(event(id, 10), func(e *Event) {
			e.Request, e.Attempt, e.Outcome, e.Usage, e.Cache, e.Cost = request, number, outcome, usage, relation, cost
			e.Input, e.Output, e.CacheRead, e.CacheWrite = counts[0], counts[1], counts[2], counts[3]
		})
	}
	h.post("r1",
		attempt("e1", "req-1", "1", Failed, ProviderReported, RelationNotKnown, [4]string{"100", "", "", ""}, nil),
		attempt("e2", "req-1", "2", Cancelled, ProviderReported, Included, [4]string{"100", "0", "40", "0"}, &Cost{Amount: "0.5", Currency: "USD", Grade: Billed, Origin: Provider}),
		attempt("e3", "req-1", "3", Succeeded, UsageEstimated, Separate, [4]string{"200", "50", "30", "5"}, &Cost{Amount: "1.25", Currency: "USD", Grade: Invoiced, Origin: Provider, Document: "inv-1"}),
		attempt("e4", "req-2", "1", Succeeded, ProviderReported, Included, [4]string{"10", "1", "50", ""}, &Cost{Amount: "9", Currency: "USD", Grade: Invoiced, Origin: Provider, Document: "inv-2"}),
	)
	h.post("r1",
		attempt("e5", "req-3", "1", StreamIncomplete, UsageNotKnown, RelationNotKnown, [4]string{"", "", "", ""}, &Cost{Amount: "7", Currency: "USD", Grade: Estimated, Origin: Provider}),
		attempt("e6", "req-4", "1", Succeeded, ProviderReported, Separate, [4]string{"0", "0", "0", "0"}, &Cost{Amount: "2", Currency: "KRW", Grade: Computed, Origin: SelfHostedGPU}),
		attempt("e7", "req-5", "1", Succeeded, ProviderReported, Included, [4]string{"5", "5", "", ""}, &Cost{Amount: "-0.25", Currency: "USD", Grade: Invoiced, Origin: Provider, Document: "inv-1"}),
	)
	_, err := h.r.Post("ws1", "r1", []Event{attempt("e8", "req-1", "2", Cancelled, ProviderReported, Included, [4]string{"100", "0", "40", "0"}, nil)})
	wantErr(t, err, ErrConflict, "the same attempt under a second event identifier")
	h.clock.advance(time.Hour)
	figures := h.close("r1")
	want := "error=1 net=260/4 cost/estimated/USD=7 cost/invoice_referenced/USD=1 cost/self_hosted/KRW=2 cost/unconfirmed/USD=0.5 doc/inv-1/USD=1 doc/inv-1=2 " +
		"outcome/cancelled=1 outcome/failed=1 outcome/stream_incomplete=1 outcome/succeeded=4 " +
		"tokens/estimated/cache_read=30 tokens/estimated/cache_write=5 tokens/estimated/input=200 tokens/estimated/output=50 " +
		"tokens/provider_reported/cache_read=90 tokens/provider_reported/cache_write=0 tokens/provider_reported/input=215 tokens/provider_reported/output=6 " +
		"unknown/cache_read=3 unknown/cache_write=4 unknown/input=1 unknown/output=2"
	if got := flat(figures); got != want {
		t.Fatalf("\n got %s\nwant %s", got, want)
	}
	if got := h.verdict("r1"); got != "not_eligible[events_in_error]" {
		t.Fatal(got)
	}
	view := h.view("r1")
	if view.Events != 7 || !view.WasClosed || !view.Changed.Equal(start.Add(time.Hour)) || !view.Opened.Equal(start) {
		t.Fatalf("%+v", view)
	}
	figures.Costs[ClassInvoiceReferenced]["USD"], figures.Tokens[ProviderReported][Input], figures.Documents["inv-1"].Sums["USD"] = "999", "999", "999"
	figures.Outcomes[Failed], figures.Unknown[Input] = 99, 99
	if again := flat(h.view("r1").Figures); again != want {
		t.Fatal("figures share state")
	}
	empty := newHarness(t, nil)
	nothing := empty.one(func(s *Spec) { s.Events = 3 })
	if nothing.NetInput != "0" || nothing.Outcomes == nil || nothing.Tokens == nil || nothing.Unknown == nil || nothing.Costs == nil || nothing.Documents == nil || flat(nothing) != "error=0 net=0/0 " {
		t.Fatalf("%+v", nothing)
	}
	if got := empty.verdict("r1"); got != "not_eligible[incomplete totals_differ]" {
		t.Fatal(got)
	}
	zeroed := newHarness(t, nil)
	credit := zeroed.one(func(s *Spec) { s.Tokens = nil; s.Costs = map[string]string{"USD": "0"} }, priced("a", 10, "1.50", Billed, Provider), priced("b", 10, "-1.5", Billed, Provider))
	if !reflect.DeepEqual(credit.Costs, map[string]map[string]string{ClassUnconfirmed: {"USD": "0"}}) {
		t.Fatalf("a class that credits bring to zero has an entry: %s", flat(credit))
	}
}

func TestEveryReasonOfEligibility(t *testing.T) {
	_, to := month(10)
	three := func() []Event { return []Event{event("e1", 10), event("e2", 10), event("e3", 10)} }
	costs := func(amounts ...string) []Event {
		events := three()
		for index, amount := range amounts {
			events[index].Cost.Amount = amount
		}
		return events
	}
	late := with(event("e3", 10), func(e *Event) { e.Time, e.Cost = to, nil })
	cases := []struct {
		name   string
		set    func(*Spec)
		events []Event
		want   string
	}{
		{"eligible", nil, three(), "eligible[]"},
		{"read from a list", func(s *Spec) { s.Produced = ListInterface }, three(), "not_eligible[list_interface]"},
		{"a derived declaration", func(s *Spec) { s.Declaration = Derived }, three(), "not_eligible[declaration_derived]"},
		{"a float total of sixteen digits", func(s *Spec) { s.Numeric, s.Costs = Float, map[string]string{"USD": "1000000001.000001"} }, costs("999999999.999999", "0.000002", "1"), "not_eligible[float_total_imprecise]"},
		{"the same total from a decimal source", func(s *Spec) { s.Costs = map[string]string{"USD": "1000000001.000001"} }, costs("999999999.999999", "0.000002", "1"), "eligible[]"},
		{"an event in error", func(s *Spec) { s.Costs = map[string]string{"USD": "2"} }, []Event{event("e1", 10), event("e2", 10), late}, "not_eligible[events_in_error]"},
		{"fewer events than declared", func(s *Spec) { s.Events = 4 }, three(), "not_eligible[incomplete]"},
		{"a token total not stated", func(s *Spec) { delete(s.Tokens, CacheWrite) }, three(), "not_eligible[totals_not_stated]"},
		{"no token total stated", func(s *Spec) { s.Tokens = nil }, three(), "not_eligible[totals_not_stated]"},
		{"a total not stated and another that differs", func(s *Spec) { delete(s.Tokens, CacheWrite); s.Tokens[Input] = "1" }, three(), "not_eligible[totals_not_stated totals_differ]"},
		{"a token total that differs", func(s *Spec) { s.Tokens[Output] = "29" }, three(), "not_eligible[totals_differ]"},
		{"a zero total against known counts", func(s *Spec) { s.Tokens[Input] = "0" }, three(), "not_eligible[totals_differ]"},
		{"a cost total that differs by the smallest unit", func(s *Spec) { s.Costs["USD"] = "3.000001" }, three(), "not_eligible[totals_differ]"},
		{"a cost total that is ten times the sum", func(s *Spec) { s.Costs["USD"] = "3" }, costs("0.1", "0.1", "0.1"), "not_eligible[totals_differ]"},
		{"a cost total written otherwise", func(s *Spec) { s.Costs["USD"] = "3.00" }, three(), "eligible[]"},
		{"a cost total with an exponent", func(s *Spec) { s.Costs["USD"] = "0.3e1" }, three(), "eligible[]"},
		{"a declared currency without a cost", func(s *Spec) { s.Costs["EUR"] = "0" }, three(), "not_eligible[totals_differ]"},
		{"a cost in an undeclared currency", func(s *Spec) { s.Costs = map[string]string{"EUR": "3"} }, three(), "not_eligible[totals_differ]"},
		{"costs without a declared total", func(s *Spec) { s.Costs = nil }, three(), "not_eligible[totals_differ]"},
		{"a sum outside the profile", func(s *Spec) { s.Costs["USD"] = "1.5e14" }, costs("50000000000000", "50000000000000", "50000000000000"), "not_eligible[sum_outside_profile]"},
		{"everything at once", func(s *Spec) {
			s.Produced, s.Declaration, s.Numeric, s.Events = ListInterface, Derived, Float, 4
			s.Costs = map[string]string{"USD": "1.000000000000001"}
			delete(s.Tokens, Output)
		}, []Event{event("e1", 10), late}, "not_eligible[list_interface declaration_derived float_total_imprecise events_in_error incomplete totals_not_stated totals_differ]"},
	}
	for _, c := range cases {
		h := newHarness(t, nil)
		spec := export("r1", "snap-1", 10)
		if c.set != nil {
			c.set(&spec)
		}
		h.open(spec)
		h.post("r1", c.events...)
		open := []string{NotClosed}
		for _, reason := range strings.Fields(strings.TrimSuffix(strings.SplitN(c.want, "[", 2)[1], "]")) {
			if reason != EventsInError && reason != Incomplete && reason != TotalsNotStated && reason != TotalsDiffer && reason != SumOutsideProfile {
				open = append(open, reason)
			}
		}
		if got, want := h.verdict("r1"), "not_eligible["+strings.Join(open, " ")+"]"; got != want {
			t.Fatalf("%s while open:\n got %s\nwant %s", c.name, got, want)
		}
		h.close("r1")
		if got := h.verdict("r1"); got != c.want {
			t.Fatalf("%s:\n got %s\nwant %s\n%s", c.name, got, c.want, flat(h.view("r1").Figures))
		}
		if got, want := h.counting(10), map[bool]string{true: "r1", false: ""}[c.want == "eligible[]"]; got != want {
			t.Fatalf("%s: counting %q", c.name, got)
		}
		if err := h.r.Void("ws1", "r1", ByOperator); err != nil {
			t.Fatal(err)
		}
		if got := h.verdict("r1"); got != "not_eligible[voided]" || h.counting(10) != "" {
			t.Fatalf("%s voided: %s", c.name, got)
		}
	}
}

func TestPeriodConflictsAndSharedEvents(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.Listed = 2 })
	from, _ := month(10)
	for index, id := range []string{"sep", "oct", "nov"} {
		h.full(export(id, "snap-"+id, 9+index))
	}
	for _, id := range []string{"sep", "oct", "nov"} {
		if got := h.verdict(id); got != "eligible[]" {
			t.Fatalf("periods that only touch do not overlap: %s %s", id, got)
		}
	}
	elsewhere := export("other", "snap-oct", 10)
	elsewhere.Source = "tm-2"
	h.open(elsewhere)
	h.post("other", event("oct-e1", 10))
	far := export("oct", "snap-oct", 10)
	far.WorkspaceID = "ws2"
	if _, _, err := h.r.Open(far); err != nil {
		t.Fatal(err)
	}
	if _, err := h.r.Post("ws2", "oct", []Event{event("oct-e1", 10)}); err != nil {
		t.Fatal(err)
	}
	if got := h.verdict("oct"); got != "eligible[]" {
		t.Fatalf("another source and another workspace neither conflict nor share: %s", got)
	}
	daily := func(id string, offset int) Spec {
		spec := export(id, "snap-"+id, 10)
		spec.PeriodStart, spec.PeriodEnd, spec.Produced = from.Add(time.Duration(offset)*day), from.Add(time.Duration(offset+1)*day), ListInterface
		return spec
	}
	h.open(daily("day-b", 3))
	if got := h.verdict("oct"); got != "not_eligible[period_conflict] with [day-b] of 1" {
		t.Fatalf("a list of one day knocks out the snapshot of the month: %s", got)
	}
	if got := h.verdict("day-b"); got != "not_eligible[not_closed list_interface period_conflict] with [oct] of 1" {
		t.Fatal(got)
	}
	h.open(daily("day-a", 4))
	h.open(daily("day-c", 30))
	if got := h.verdict("oct"); got != "not_eligible[period_conflict] with [day-a day-b] of 3" || h.counting(10) != "" {
		t.Fatal(got)
	}
	for _, id := range []string{"day-a", "day-b", "day-c"} {
		if err := h.r.Void("ws1", id, WrongReport); err != nil {
			t.Fatal(err)
		}
	}
	if got := h.verdict("oct"); got != "eligible[]" || h.counting(10) != "oct" {
		t.Fatal(got)
	}
	december := export("dec", "snap-dec", 12)
	h.open(december)
	same := with(event("oct-e1", 12), func(e *Event) { e.Request = "req-fresh" })
	h.post("dec", same)
	if got := h.verdict("oct"); got != "not_eligible[event_shared] sharing [dec] of 1" || h.counting(10) != "" {
		t.Fatalf("an event identifier held by two receptions: %s", got)
	}
	if got := h.verdict("dec"); got != "not_eligible[not_closed event_shared] sharing [oct] of 1" {
		t.Fatal(got)
	}
	pair := with(event("dec-own", 12), func(e *Event) { e.Request = "req-nov-e2" })
	h.post("dec", pair)
	if got := h.verdict("nov"); got != "not_eligible[event_shared] sharing [dec] of 1" {
		t.Fatalf("a pair of request and attempt held by two receptions: %s", got)
	}
	if got := h.verdict("dec"); got != "not_eligible[not_closed event_shared] sharing [nov oct] of 2" {
		t.Fatal(got)
	}
	h.post("dec", with(event("dec-two", 12), func(e *Event) { e.Request, e.Attempt = "req-sep-e1", "2" }))
	if got := h.verdict("sep"); got != "eligible[]" {
		t.Fatalf("another attempt of the same request is another event: %s", got)
	}
	if err := h.r.Void("ws1", "dec", WrongReport); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"oct", "nov"} {
		if got := h.verdict(id); got != "eligible[]" {
			t.Fatalf("a void is the exit: %s %s", id, got)
		}
	}
	again := export("dec2", "snap-dec", 12)
	h.open(again)
	h.post("dec2", event("dec-own", 12))
	if got := h.verdict("dec2"); got != "not_eligible[not_closed]" {
		t.Fatalf("a voided reception of the same export does not share: %s", got)
	}
	overlap := export("oct-again", "snap-oct-2", 10)
	h.open(overlap)
	h.post("oct-again", event("oct-e1", 10), event("oct-e2", 10))
	if got := h.verdict("oct"); got != "not_eligible[period_conflict event_shared] with [oct-again] of 1 sharing [oct-again] of 1" {
		t.Fatalf("a regenerated snapshot of the month: %s", got)
	}
	if got := h.verdict("oct-again"); got != "not_eligible[not_closed period_conflict event_shared] with [oct] of 1 sharing [oct] of 1" {
		t.Fatal(got)
	}
}

func TestEveryReasonInOneVerdict(t *testing.T) {
	h := newHarness(t, nil)
	_, to := month(10)
	neighbour := export("r0", "snap-0", 10)
	h.open(neighbour)
	h.post("r0", event("e1", 10))
	spec := export("r1", "snap-1", 10)
	spec.Produced, spec.Declaration, spec.Numeric, spec.Events = ListInterface, Derived, Float, 5
	spec.Tokens = map[Kind]string{Input: "1", CacheRead: "0", CacheWrite: "0"}
	spec.Costs = map[string]string{"USD": "1.000000000000001"}
	h.open(spec)
	large := func(id string) Event { return priced(id, 10, "50000000000000", Billed, Provider) }
	h.post("r1", large("e1"), large("e2"), with(event("e3", 10), func(e *Event) { e.Time, e.Cost = to, nil }))
	if got, want := h.verdict("r1"), "not_eligible[not_closed list_interface declaration_derived float_total_imprecise period_conflict event_shared] with [r0] of 1 sharing [r0] of 1"; got != want {
		t.Fatalf("\n got %s\nwant %s", got, want)
	}
	h.close("r1")
	want := "not_eligible[list_interface declaration_derived float_total_imprecise period_conflict event_shared events_in_error incomplete totals_not_stated totals_differ sum_outside_profile] with [r0] of 1 sharing [r0] of 1"
	if got := h.verdict("r1"); got != want {
		t.Fatalf("\n got %s\nwant %s", got, want)
	}
	papers := newHarness(t, nil)
	credit := func(id, amount, document string) Event {
		return with(event(id, 10), func(e *Event) { e.Cost.Amount, e.Cost.Document = amount, document })
	}
	figures := papers.one(func(s *Spec) {
		s.Tokens = map[Kind]string{Input: "400", Output: "40", CacheRead: "0", CacheWrite: "0"}
		s.Costs = map[string]string{"USD": "0"}
	},
		credit("a", "90000000000000", "inv-a"), credit("b", "90000000000000", "inv-a"), credit("c", "-90000000000000", "inv-b"), credit("d", "-90000000000000", "inv-b"))
	if figures.Costs[ClassInvoiceReferenced]["USD"] != "0" || figures.Documents["inv-a"].Sums["USD"] != "180000000000000" || figures.Documents["inv-b"].Sums["USD"] != "-180000000000000" {
		t.Fatal(flat(figures))
	}
	if got := papers.verdict("r1"); got != "not_eligible[sum_outside_profile]" {
		t.Fatalf("the sum of a document outside the profile while its class sums to zero: %s", got)
	}
}
