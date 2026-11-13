package rating

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestTheCasesOfTheAcceptance(t *testing.T) {
	markup := policy(byDefault(Markup, func(r *Rule) { r.Rate = "0.2" }))
	if out := rate(t, markup, cost("a", "100")); charges(out) != "default/rated/a=120" || out.Total != "120" || steps(out, "default") != "costs_above_zero(;)=100 costs_below_zero(;)=0 base(100;0.2)=120 base_amount(120;1)=120 passed(;)=0 passed_amount(0;1)=0 total(;)=120 | 120" {
		t.Fatalf("100 with a markup of 20%% is 120: %s | %s", charges(out), steps(out, "default"))
	}
	margin := policy(byDefault(TargetMargin, func(r *Rule) { r.Margin = "0.2" }))
	if out := rate(t, margin, cost("a", "100")); charges(out) != "default/rated/a=125" || out.Total != "125" {
		t.Fatalf("100 with a target margin of 20%% is 125: %s", charges(out))
	}
	before, after := rate(t, markup, cost("a", "100"), cost("b", "55.5")), rate(t, alter(markup, func(p *Policy) { p.Version = "v2" }), cost("a", "100"), cost("b", "55.5"))
	if before.Version != "v1" || after.Version != "v2" {
		t.Fatal(before.Version, after.Version)
	}
	after.Version = "v1"
	if !reflect.DeepEqual(before, after) {
		t.Fatal("a change of the version changes nothing but the label")
	}
	for name, c := range map[string]struct {
		rule Rule
		want string
	}{
		"credits pass through at cost":             {byDefault(PassThrough, nil), "default/rated/a=100 default/rated/z=0 default/passed_at_cost/b=-30 default/passed_at_cost/c=-20 | 50"},
		"credits take no markup":                   {byDefault(Markup, func(r *Rule) { r.Rate = "0.2" }), "default/rated/a=120 default/rated/z=0 default/passed_at_cost/b=-30 default/passed_at_cost/c=-20 | 70"},
		"credits take no margin":                   {byDefault(TargetMargin, func(r *Rule) { r.Margin = "0.2" }), "default/rated/a=125 default/rated/z=0 default/passed_at_cost/b=-30 default/passed_at_cost/c=-20 | 75"},
		"credits are outside discounts and bounds": {byDefault(Markup, func(r *Rule) { r.Rate = "0.2"; r.Discounts = []Discount{{Percent: "0.5"}}; r.Maximum = "40" }), "default/rated/a=120 default/rated/z=0 default/passed_at_cost/b=-30 default/passed_at_cost/c=-20 default/discount/-=-60 default/maximum_cap/-=-20 | -10"},
	} {
		out := rate(t, policy(c.rule), cost("a", "100"), cost("b", "-30"), cost("c", "-20"), cost("z", "0"))
		if got := charges(out) + " | " + out.Total; got != c.want {
			t.Fatalf("%s:\n got %s\nwant %s", name, got, c.want)
		}
	}
	if out := rate(t, markup, cost("a", "100"), cost("b", "-100")); out.Total != "20" {
		t.Fatalf("a cost and its reversal as two lines are a cost and a credit: %s", out.Total)
	}
	if out := rate(t, margin, cost("a", "100"), cost("b", "-100")); out.Total != "25" {
		t.Fatal(out.Total)
	}
	if out := rate(t, policy(byDefault(PassThrough, func(r *Rule) { r.Minimum = "100" })), cost("a", "-50"), cost("b", "-0")); charges(out) != "default/rated/b=0 default/passed_at_cost/a=-50 default/minimum_top_up/-=100" || out.Total != "50" {
		t.Fatalf("a rule of credits only is active and charges its minimum: %s", charges(out))
	}
	stacked := policy(byDefault(PassThrough, func(r *Rule) { r.Discounts = []Discount{{Percent: "0.1"}, {Fixed: "50"}, {Percent: "0.5"}} }))
	out := rate(t, stacked, cost("a", "1000"))
	if charges(out) != "default/rated/a=1000 default/discount/-=-100 default/discount/-=-50 default/discount/-=-425" || out.Total != "425" {
		t.Fatalf("discounts stack on what is left: %s", charges(out))
	}
	if got := steps(out, "default"); !strings.Contains(got, "discount(1000;0.1)=-100 discount(900;50)=-50 discount(850;0.5)=-425 total(;)=425") {
		t.Fatal(got)
	}
	lines := []Line{cost("usage", "1000"), alter(cost("fee", "200"), func(l *Line) { l.Class = ServiceFee }), alter(cost("lic", "50"), func(l *Line) { l.Class = Licence }), alter(cost("misc", "5"), func(l *Line) { l.Class = Other })}
	direct := alter(markup, func(p *Policy) { p.Path, p.Eligible = Direct, nil })
	if out := rate(t, direct, lines...); charges(out) != "default/rated/fee=240" || excluded(out) != "lic:class_not_eligible misc:class_not_eligible usage:class_not_eligible" || out.Path != Direct {
		t.Fatalf("under a direct contract only service fees are charged: %s | %s", charges(out), excluded(out))
	}
	if out := rate(t, markup, lines...); charges(out) != "default/rated/fee=240 default/rated/lic=60 default/rated/misc=6 default/rated/usage=1200" || excluded(out) != "" {
		t.Fatal(charges(out))
	}
	resale := alter(markup, func(p *Policy) { p.Eligible = []Class{CSPUsage, ServiceFee} })
	if out := rate(t, resale, lines...); charges(out) != "default/rated/fee=240 default/rated/usage=1200" || excluded(out) != "lic:class_not_eligible misc:class_not_eligible" || fmt.Sprint(out.Exclusions) != "[{class_not_eligible 2}]" {
		t.Fatal(charges(out), excluded(out))
	}
	if out := rate(t, alter(markup, func(p *Policy) { p.Eligible = nil }), lines...); charges(out) != "" || len(out.Excluded) != 4 || out.Total != "0" || out.Traces != nil {
		t.Fatal(charges(out))
	}
}

func TestChargeTypes(t *testing.T) {
	per := func(amount string, denominator int) Price { return Price{Amount: amount, Per: denominator} }
	tiers := []Tier{{UpTo: "100", Price: per("10", 1)}, {UpTo: "1000", Price: per("8000", 1000)}, {Price: per("5000000", 1000000)}}
	cases := []struct {
		name  string
		rule  Rule
		lines []Line
		want  string
		trace string
	}{
		{"a unit price", byDefault(UnitPrice, func(r *Rule) { r.Price = per("12.5", 1) }), []Line{used("a", "3"), used("b", "1")}, "default/rated/a=38 default/rated/b=12 | 50", "quantity(;)=4 included(;)=0 above_included(4;0)=4 base(4;)=50 base_amount(50;1)=50 total(;)=50 | 50"},
		{"a price per thousand", byDefault(UnitPrice, func(r *Rule) { r.Price = per("2.5", 1000) }), []Line{used("a", "123456")}, "default/rated/a=309 | 309", ""},
		{"a price per million", byDefault(UnitPrice, func(r *Rule) { r.Price = per("3", 1000000) }), []Line{used("a", "2500000.5")}, "default/rated/a=8 | 8", "quantity(;)=5000001/2 included(;)=0 above_included(5000001/2;0)=5000001/2 base(5000001/2;)=15000003/2000000 base_amount(15000003/2000000;1)=8 total(;)=8 | 8"},
		{"an included quantity", byDefault(UnitPrice, func(r *Rule) { r.Price, r.Included = per("10", 1), "100" }), []Line{used("a", "60"), used("b", "70")}, "default/rated/a=138 default/rated/b=162 | 300", "quantity(;)=130 included(;)=100 above_included(130;100)=30 base(30;)=300 base_amount(300;1)=300 total(;)=300 | 300"},
		{"an included quantity above the quantity", byDefault(UnitPrice, func(r *Rule) { r.Price, r.Included = per("10", 1), "100" }), []Line{used("a", "60"), used("b", "40")}, "default/rated/a=0 default/rated/b=0 | 0", ""},
		{"graduated tiers", byDefault(Graduated, func(r *Rule) { r.Tiers = tiers }), []Line{used("a", "1500")}, "default/rated/a=10700 | 10700",
			"quantity(;)=1500 included(;)=0 above_included(1500;0)=1500 tier(100;10/1)=1000 tier(900;8000/1000)=7200 tier(500;5000000/1000000)=2500 base(1500;)=10700 base_amount(10700;1)=10700 total(;)=10700 | 10700"},
		{"a graduated quantity on a bound", byDefault(Graduated, func(r *Rule) { r.Tiers = tiers }), []Line{used("a", "100")}, "default/rated/a=1000 | 1000", "quantity(;)=100 included(;)=0 above_included(100;0)=100 tier(100;10/1)=1000 base(100;)=1000 base_amount(1000;1)=1000 total(;)=1000 | 1000"},
		{"graduated tiers measured above the included quantity", byDefault(Graduated, func(r *Rule) { r.Tiers, r.Included = tiers, "50" }), []Line{used("a", "160")}, "default/rated/a=1080 | 1080", ""},
		{"graduated tiers without usage", byDefault(Graduated, func(r *Rule) { r.Tiers = tiers }), []Line{used("a", "0")}, "default/rated/a=0 | 0", "quantity(;)=0 included(;)=0 above_included(0;0)=0 base(0;)=0 base_amount(0;1)=0 total(;)=0 | 0"},
		{"volume tiers", byDefault(Volume, func(r *Rule) { r.Tiers = tiers }), []Line{used("a", "1500")}, "default/rated/a=7500 | 7500", "quantity(;)=1500 included(;)=0 above_included(1500;0)=1500 tier(1500;5000000/1000000)=7500 base(1500;)=7500 base_amount(7500;1)=7500 total(;)=7500 | 7500"},
		{"a volume quantity on a bound belongs to the lower tier", byDefault(Volume, func(r *Rule) { r.Tiers = tiers }), []Line{used("a", "100")}, "default/rated/a=1000 | 1000", ""},
		{"a volume quantity just above a bound", byDefault(Volume, func(r *Rule) { r.Tiers = tiers }), []Line{used("a", "100.0001")}, "default/rated/a=800 | 800", ""},
		{"a volume quantity on the last bound", byDefault(Volume, func(r *Rule) { r.Tiers = tiers }), []Line{used("a", "1000")}, "default/rated/a=8000 | 8000", ""},
		{"volume tiers without usage", byDefault(Volume, func(r *Rule) { r.Tiers = tiers }), []Line{used("a", "0")}, "default/rated/a=0 | 0", "quantity(;)=0 included(;)=0 above_included(0;0)=0 base(0;)=0 base_amount(0;1)=0 total(;)=0 | 0"},
		{"one tier", byDefault(Volume, func(r *Rule) { r.Tiers = []Tier{{Price: per("3", 1)}} }), []Line{used("a", "7")}, "default/rated/a=21 | 21", ""},
		{"a minimum that adds what is lacking", byDefault(UnitPrice, func(r *Rule) { r.Price, r.Minimum = per("10", 1), "500" }), []Line{used("a", "30")}, "default/rated/a=300 default/minimum_top_up/-=200 | 500", "quantity(;)=30 included(;)=0 above_included(30;0)=30 base(30;)=300 base_amount(300;1)=300 minimum(300;500)=200 total(;)=500 | 500"},
		{"a minimum that is met", byDefault(UnitPrice, func(r *Rule) { r.Price, r.Minimum = per("10", 1), "300" }), []Line{used("a", "30")}, "default/rated/a=300 | 300", "quantity(;)=30 included(;)=0 above_included(30;0)=30 base(30;)=300 base_amount(300;1)=300 minimum(300;300)=0 total(;)=300 | 300"},
		{"a maximum that takes off what exceeds", byDefault(UnitPrice, func(r *Rule) { r.Price, r.Maximum = per("10", 1), "250" }), []Line{used("a", "30")}, "default/rated/a=300 default/maximum_cap/-=-50 | 250", ""},
		{"a maximum that is met", byDefault(UnitPrice, func(r *Rule) { r.Price, r.Maximum = per("10", 1), "300" }), []Line{used("a", "30")}, "default/rated/a=300 | 300", ""},
		{"a minimum and a maximum that are equal", byDefault(UnitPrice, func(r *Rule) { r.Price, r.Minimum, r.Maximum = per("10", 1), "400", "400" }), []Line{used("a", "30")}, "default/rated/a=300 default/minimum_top_up/-=100 | 400",
			"quantity(;)=30 included(;)=0 above_included(30;0)=30 base(30;)=300 base_amount(300;1)=300 minimum(300;400)=100 maximum(400;400)=0 total(;)=400 | 400"},
		{"a maximum after a minimum that added", byDefault(UnitPrice, func(r *Rule) { r.Price, r.Minimum, r.Maximum = per("10", 1), "400", "450" }), []Line{used("a", "30")}, "default/rated/a=300 default/minimum_top_up/-=100 | 400",
			"quantity(;)=30 included(;)=0 above_included(30;0)=30 base(30;)=300 base_amount(300;1)=300 minimum(300;400)=100 maximum(400;450)=0 total(;)=400 | 400"},
		{"a maximum that applies beside a minimum that does not", byDefault(UnitPrice, func(r *Rule) { r.Price, r.Minimum, r.Maximum = per("10", 1), "100", "250" }), []Line{used("a", "30")}, "default/rated/a=300 default/maximum_cap/-=-50 | 250",
			"quantity(;)=30 included(;)=0 above_included(30;0)=30 base(30;)=300 base_amount(300;1)=300 minimum(300;100)=0 maximum(300;250)=-50 total(;)=250 | 250"},
		{"a minimum without usage", byDefault(UnitPrice, func(r *Rule) { r.Price, r.Minimum = per("10", 1), "500" }), []Line{used("a", "0")}, "default/rated/a=0 default/minimum_top_up/-=500 | 500", ""},
		{"a discount after which the minimum applies", byDefault(UnitPrice, func(r *Rule) { r.Price, r.Minimum, r.Discounts = per("10", 1), "250", []Discount{{Percent: "0.5"}} }), []Line{used("a", "30")}, "default/rated/a=300 default/discount/-=-150 default/minimum_top_up/-=100 | 250", ""},
		{"a fixed discount larger than what is left", byDefault(UnitPrice, func(r *Rule) {
			r.Price, r.Discounts = per("10", 1), []Discount{{Fixed: "200"}, {Fixed: "200"}, {Fixed: "200"}}
		}), []Line{used("a", "30")},
			"default/rated/a=300 default/discount/-=-200 default/discount/-=-100 | 0", "quantity(;)=30 included(;)=0 above_included(30;0)=30 base(30;)=300 base_amount(300;1)=300 discount(300;200)=-200 discount(100;200)=-100 discount(0;200)=0 total(;)=0 | 0"},
	}
	for _, c := range cases {
		out := rate(t, policy(c.rule), c.lines...)
		if got := charges(out) + " | " + out.Total; got != c.want {
			t.Fatalf("%s:\n got %s\nwant %s", c.name, got, c.want)
		}
		if got := steps(out, "default"); c.trace != "" && got != c.trace {
			t.Fatalf("%s:\n got %s\nwant %s", c.name, got, c.trace)
		}
	}
	mixed := byDefault(Graduated, func(r *Rule) {
		r.Tiers = []Tier{{UpTo: "1", Price: per("0.4", 1)}, {UpTo: "2", Price: per("400", 1000)}, {Price: per("400000", 1000000)}}
	})
	if out := rate(t, policy(mixed), used("a", "3")); out.Total != "1" {
		t.Fatalf("three parts of 0.4 are rounded once, to 1, and not each to 0: %s", out.Total)
	}
	flat := policy(Rule{ID: "fee", Level: Service, Key: "support", Type: Flat, Amount: "300000", Discounts: []Discount{{Percent: "0.1"}}, Maximum: "250000"}, byDefault(PassThrough, nil))
	support := func(id, amount, quantity string) Line {
		return Line{ID: id, Class: ServiceFee, Service: "support", Cost: amount, Quantity: quantity, Currency: "KRW"}
	}
	out := rate(t, flat, support("s1", "", ""), support("s2", "-40", "3"), cost("u", "10"))
	if charges(out) != "default/rated/u=10 fee/flat/-=300000 fee/discount/-=-30000 fee/maximum_cap/-=-20000" || out.Total != "250010" || fmt.Sprint(out.Covered) != "[{s1 fee} {s2 fee}]" || !out.NoRuleHeld {
		t.Fatalf("a flat fee needs neither input and covers its lines: %s %v", charges(out), out.Covered)
	}
	if got := steps(out, "fee"); got != "base_amount(300000;)=300000 discount(300000;0.1)=-30000 maximum(270000;250000)=-20000 total(;)=250000 | 250000" {
		t.Fatal(got)
	}
	if out := rate(t, flat, cost("u", "10")); charges(out) != "default/rated/u=10" || len(out.Traces) != 1 {
		t.Fatalf("a flat fee is not charged for a rule without a line: %s", charges(out))
	}
	zero := policy(Rule{ID: "fee", Level: Group, Key: "g", Type: Flat, Amount: "0", Minimum: "0"})
	if out := rate(t, zero, alter(cost("a", "1"), func(l *Line) { l.Group = "g" })); charges(out) != "" || out.Total != "0" || steps(out, "fee") != "base_amount(0;)=0 minimum(0;0)=0 total(;)=0 | 0" || len(out.Covered) != 1 {
		t.Fatalf("a charge of a rule that is zero is not returned, its step is: %s", steps(out, "fee"))
	}
}

func TestRoundingQuantaAndShares(t *testing.T) {
	modes := map[string][2]string{"toward_zero": {"2", "-2"}, "away_from_zero": {"3", "-3"}, "floor": {"2", "-3"}, "ceiling": {"3", "-2"}, "half_up": {"3", "-3"}, "half_even": {"2", "-2"}}
	for mode, want := range modes {
		p := alter(policy(byDefault(PassThrough, nil)), func(p *Policy) { p.Mode = mode })
		out := rate(t, p, cost("a", "2.5"), cost("b", "-2.5"))
		if got := charges(out); got != fmt.Sprintf("default/rated/a=%s default/passed_at_cost/b=%s", want[0], want[1]) {
			t.Fatalf("%s: %s", mode, got)
		}
	}
	for mode, want := range map[string][2]string{"half_even": {"4", "-4"}, "half_up": {"4", "-4"}, "floor": {"3", "-4"}, "ceiling": {"4", "-3"}, "toward_zero": {"3", "-3"}, "away_from_zero": {"4", "-4"}} {
		p := alter(policy(byDefault(PassThrough, nil)), func(p *Policy) { p.Mode = mode })
		out := rate(t, p, cost("a", "3.5"), cost("b", "-3.5"))
		if got := charges(out); got != fmt.Sprintf("default/rated/a=%s default/passed_at_cost/b=%s", want[0], want[1]) {
			t.Fatalf("%s at an odd count: %s", mode, got)
		}
	}
	for mode, want := range map[string]string{"half_even": "2", "half_up": "2", "floor": "2", "ceiling": "3", "toward_zero": "2", "away_from_zero": "3"} {
		p := alter(policy(byDefault(PassThrough, nil)), func(p *Policy) { p.Mode = mode })
		if out := rate(t, p, cost("a", "2.4999")); out.Total != want {
			t.Fatalf("%s below a tie: %s", mode, out.Total)
		}
		if out := rate(t, p, cost("a", "2")); out.Total != "2" {
			t.Fatalf("%s on a whole count: %s", mode, out.Total)
		}
	}
	for quantum, want := range map[string]string{"1": "143", "0.1": "142.9", "0.01": "142.86", "0.001": "142.857", "0.0001": "142.8571"} {
		p := alter(policy(byDefault(TargetMargin, func(r *Rule) { r.Margin = "0.3" })), func(p *Policy) { p.Quantum = quantum })
		out := rate(t, p, cost("a", "100"))
		if out.Total != want || out.Quantum != quantum || !strings.Contains(steps(out, "default"), "base(100;0.3)=1000/7 base_amount(1000/7;"+quantum+")="+want) {
			t.Fatalf("a ratio that does not end, at %s: %s %s", quantum, out.Total, steps(out, "default"))
		}
	}
	discount := func(mode string) string {
		p := alter(policy(byDefault(PassThrough, func(r *Rule) { r.Discounts = []Discount{{Percent: "0.5"}} })), func(p *Policy) { p.Mode = mode })
		return charges(rate(t, p, cost("a", "1")))
	}
	if discount("ceiling") != "default/rated/a=1 default/discount/-=-1" || discount("floor") != "default/rated/a=1" || discount("half_even") != "default/rated/a=1" || discount("half_up") != "default/rated/a=1 default/discount/-=-1" {
		t.Fatalf("the size of a discount is rounded as a value above zero: %s %s", discount("ceiling"), discount("floor"))
	}
	full := profileOf(t, 12, 4, 14)
	compiled, err := Compile(full, testLimits(), alter(policy(byDefault(Markup, func(r *Rule) { r.Rate = "0.1755" })), func(p *Policy) { p.Quantum = "0.0001" }))
	if err != nil {
		t.Fatal(err)
	}
	out, err := compiled.Rate([]Line{cost("a", "100.1234")})
	if err != nil || out.Total != "117.6951" {
		t.Fatalf("a product whose scale the profile cannot hold is still rated exactly and rounded once: %s %v", out.Total, err)
	}
	shares := policy(byDefault(PassThrough, nil))
	if got := charges(rate(t, shares, cost("c", "0.5"), cost("a", "0.5"), cost("b", "0.5"), cost("z", "0"))); got != "default/rated/a=1 default/rated/b=1 default/rated/c=0 default/rated/z=0" {
		t.Fatalf("a rule's amount is rounded once and its units go to lines by identifier on a tie: %s", got)
	}
	if got := charges(rate(t, shares, cost("a", "-0.5"), cost("b", "-0.5"), cost("c", "-0.5"), cost("d", "-0.5"))); got != "default/passed_at_cost/a=-1 default/passed_at_cost/b=-1 default/passed_at_cost/c=0 default/passed_at_cost/d=0" {
		t.Fatalf("credits are rounded once and shared by size: %s", got)
	}
	if got := charges(rate(t, shares, cost("a", "-0.004"))); got != "default/passed_at_cost/a=0" {
		t.Fatalf("a credit that rounds to zero is returned for its line: %s", got)
	}
	units := policy(byDefault(UnitPrice, func(r *Rule) { r.Price = Price{Amount: "1", Per: 1} }))
	if got := charges(rate(t, units, used("a", "0.5"), used("b", "0.5"), used("c", "0.5"))); got != "default/rated/a=1 default/rated/b=1 default/rated/c=0" {
		t.Fatalf("three halves cost two, whatever the cut: %s", got)
	}
	whole := rate(t, units, used("a", "1.5"))
	split := rate(t, units, used("a1", "0.7"), used("a2", "0.8"))
	if whole.Total != "2" || split.Total != "2" || steps(whole, "default") != steps(split, "default") {
		t.Fatalf("cutting a usage into lines leaves the rule's amounts unchanged:\n%s\n%s", steps(whole, "default"), steps(split, "default"))
	}
}

func TestPrecedenceExclusionsAndHeldRules(t *testing.T) {
	p := policy(
		Rule{ID: "r-sku", Level: SKU, Key: "sku-1", Type: PassThrough},
		Rule{ID: "r-service", Level: Service, Key: "svc-1", Type: Markup, Rate: "1"},
		Rule{ID: "r-group", Level: Group, Key: "grp-1", Type: UnitPrice, Price: Price{Amount: "7", Per: 1}},
		byDefault(TargetMargin, func(r *Rule) { r.Margin = "0.5" }),
	)
	line := func(id, sku, service, group string) Line {
		return Line{ID: id, Class: Other, SKU: sku, Service: service, Group: group, Cost: "10", Quantity: "2", Currency: "KRW"}
	}
	out := rate(t, p, line("all", "sku-1", "svc-1", "grp-1"), line("svc", "sku-9", "svc-1", "grp-1"), line("grp", "", "svc-9", "grp-1"), line("none", "sku-9", "svc-9", "grp-9"), line("bare", "", "", ""))
	if got := charges(out); got != "default/rated/bare=20 default/rated/none=20 r-group/rated/grp=14 r-service/rated/svc=20 r-sku/rated/all=10" {
		t.Fatalf("the most specific rule that names a line rates it:\n%s", got)
	}
	var order []string
	for _, trace := range out.Traces {
		order = append(order, trace.Rule)
	}
	if strings.Join(order, " ") != "default r-group r-service r-sku" || out.Total != "84" || out.Lines != 5 {
		t.Fatal(order, out.Total)
	}
	without := alter(p, func(p *Policy) { p.Rules = p.Rules[:3]; p.Eligible = []Class{Other} })
	out = rate(t, without, line("none", "sku-9", "svc-9", "grp-9"), alter(line("fee", "sku-9", "", ""), func(l *Line) { l.Class = ServiceFee }), line("all", "sku-1", "", ""))
	if excluded(out) != "fee:class_not_eligible none:no_rule" || fmt.Sprint(out.Exclusions) != "[{class_not_eligible 1} {no_rule 1}]" || !out.NoRuleHeld {
		t.Fatalf("a class is judged before the rule: %s", excluded(out))
	}
	held := policy(
		Rule{ID: "units", Level: Group, Key: "g", Type: Volume, Tiers: []Tier{{UpTo: "100", Price: Price{Amount: "10", Per: 1}}, {Price: Price{Amount: "1", Per: 1}}}, Minimum: "500"},
		byDefault(Markup, func(r *Rule) { r.Rate = "0.2"; r.Minimum = "100" }),
	)
	grouped := func(id, quantity string) Line {
		return Line{ID: id, Class: Other, Group: "g", Quantity: quantity, Currency: "KRW"}
	}
	out = rate(t, held, grouped("q1", "80"), grouped("q2", ""), grouped("q3", ""), cost("c1", "50"), alter(cost("c2", ""), func(l *Line) { l.Quantity = "9" }), alter(grouped("x", ""), func(l *Line) { l.Class = CSPUsage; l.Group = "g" }))
	if charges(out) != "" || out.Total != "0" || out.NoRuleHeld {
		t.Fatalf("a rule with an unknown input charges nothing, not even its minimum: %s", charges(out))
	}
	if excluded(out) != "c1:rule_held c2:cost_not_known q1:rule_held q2:quantity_not_known q3:quantity_not_known x:quantity_not_known" {
		t.Fatal(excluded(out))
	}
	if fmt.Sprint(out.Exclusions) != "[{cost_not_known 1} {quantity_not_known 3} {rule_held 2}]" || steps(out, "units") != "held(;)=3 | 0" || steps(out, "default") != "held(;)=1 | 0" {
		t.Fatal(out.Exclusions, steps(out, "units"))
	}
	out = rate(t, held, grouped("q1", "80"), alter(cost("c2", ""), func(l *Line) { l.Quantity = "9" }), alter(grouped("known", "20"), func(l *Line) { l.Cost = "" }))
	if charges(out) != "units/rated/known=200 units/rated/q1=800" || excluded(out) != "c2:cost_not_known" || out.NoRuleHeld || out.Total != "1000" {
		t.Fatalf("a held rule does not hold another, and an input a type does not need may be unknown: %s | %s", charges(out), excluded(out))
	}
	ineligible := alter(held, func(p *Policy) { p.Eligible = []Class{Other} })
	out = rate(t, ineligible, grouped("q1", "80"), alter(grouped("usage", ""), func(l *Line) { l.Class = CSPUsage }))
	if charges(out) != "units/rated/q1=800" || excluded(out) != "usage:class_not_eligible" || !out.NoRuleHeld {
		t.Fatalf("a line that is not eligible does not hold its rule: %s | %s", charges(out), excluded(out))
	}
	out = rate(t, ineligible, alter(grouped("zz", "5"), func(l *Line) { l.Class = CSPUsage }), grouped("aa", ""), grouped("mm", "1"))
	if excluded(out) != "aa:quantity_not_known mm:rule_held zz:class_not_eligible" {
		t.Fatalf("excluded lines are listed by identifier: %s", excluded(out))
	}
	if out := rate(t, p); out.Lines != 0 || out.Charges != nil || out.Traces != nil || out.Excluded != nil || out.Covered != nil || out.Exclusions != nil || out.Total != "0" || !out.NoRuleHeld {
		t.Fatalf("no lines: %+v", out)
	}
	labels := rate(t, alter(p, func(p *Policy) { p.CostBasis, p.Mode, p.Quantum = Effective, "floor", "0.01" }))
	if labels.PolicyID != "pol-1" || labels.Version != "v1" || labels.Currency != "KRW" || labels.CostBasis != Effective || labels.Path != Resale || labels.Quantum != "0.01" || labels.Mode != "floor" || labels.ProfileID != "synthetic" || labels.ProfileRevision != "7" || labels.Algorithm != "rating_v1" {
		t.Fatalf("%+v", labels)
	}
}
