package formula

import (
	"strings"
	"testing"
)

const twelve = "999999999999"

func common() []Variable {
	return vars(decl("cost", usd), decl("rate", number), decl("hours", hour), decl("unit", usdHour), decl("a", number), decl("b", number))
}

func TestEveryKindOfNode(t *testing.T) {
	cases := []struct {
		name   string
		output Type
		root   Node
		inputs []string
		want   string
	}{
		{"cost times one plus a rate", usd, op(NodeMul, vr("cost"), op(NodeAdd, con("1", number), vr("rate"))), []string{"cost", "100", "rate", "0.2"}, "f=120(120)"},
		{"a gross-up", usd, op(NodeGrossUp, vr("cost"), vr("rate")), []string{"cost", "100", "rate", "0.2"}, "f=125(125)"},
		{"a gross-up of a credit", usd, op(NodeGrossUp, vr("cost"), vr("rate")), []string{"cost", "-100", "rate", "0.2"}, "f=-125(-125)"},
		{"a gross-up by nothing", usd, op(NodeGrossUp, vr("cost"), vr("rate")), []string{"cost", "100", "rate", "0"}, "f=100(100)"},
		{"a gross-up just below one", usd, op(NodeGrossUp, vr("cost"), vr("rate")), []string{"cost", "1", "rate", "0.999999"}, "f=1000000(1000000)"},
		{"a quantity times a price", usd, op(NodeMul, vr("hours"), vr("unit")), []string{"hours", "3", "unit", "0.105"}, "f=0.32(63/200)"},
		{"a price times a quantity", usd, op(NodeMul, vr("unit"), vr("hours")), []string{"hours", "3", "unit", "0.105"}, "f=0.32(63/200)"},
		{"a share of an amount", usd, op(NodeMul, vr("cost"), op(NodeDiv, vr("a"), vr("b"))), []string{"cost", "100", "a", "1", "b", "3"}, "f=33.33(100/3)"},
		{"a minimum charge", usd, op(NodeMax, vr("cost"), con("50", usd)), []string{"cost", "30"}, "f=50(50)"},
		{"a minimum charge not reached", usd, op(NodeMax, vr("cost"), con("50", usd)), []string{"cost", "70"}, "f=70(70)"},
		{"a cap", usd, op(NodeMin, vr("cost"), con("50", usd)), []string{"cost", "70"}, "f=50(50)"},
		{"a cap not reached", usd, op(NodeMin, vr("cost"), con("50", usd)), []string{"cost", "30"}, "f=30(30)"},
		{"equal operands of a maximum", usd, op(NodeMax, vr("cost"), con("50.0", usd)), []string{"cost", "50"}, "f=50(50)"},
		{"a difference below zero", usd, op(NodeSub, vr("cost"), con("150", usd)), []string{"cost", "100"}, "f=-50(-50)"},
		{"a negative constant", usd, op(NodeAdd, vr("cost"), con("-0.005", usd)), []string{"cost", "1"}, "f=1(199/200)"},
		{"an amount by a number", usd, op(NodeDiv, vr("cost"), vr("a")), []string{"cost", "10", "a", "4"}, "f=2.5(5/2)"},
		{"an amount by a quantity", usdHour, op(NodeDiv, vr("cost"), vr("hours")), []string{"cost", "1", "hours", "8"}, "f=0.12(1/8)"},
		{"an amount by a price", hour, op(NodeDiv, vr("cost"), vr("unit")), []string{"cost", "10", "unit", "4"}, "f=2.5(5/2)"},
		{"an amount by an amount", number, op(NodeDiv, vr("cost"), con("8", usd)), []string{"cost", "1"}, "f=0.12(1/8)"},
		{"a price by a price", number, op(NodeDiv, vr("unit"), con("3", usdHour)), []string{"unit", "2"}, "f=0.67(2/3)"},
		{"a constant alone", usd, con("12.345", usd), nil, "f=12.34(2469/200)"},
		{"a variable alone", usd, vr("cost"), []string{"cost", "0.015"}, "f=0.02(3/200)"},
		{"zero", usd, op(NodeMul, vr("cost"), con("0", number)), []string{"cost", "7"}, "f=0(0)"},
		{"a variable that is not used", usd, vr("cost"), []string{"cost", "1", "rate", "x"[:0]}, "f=1(1)"},
	}
	for _, c := range cases {
		if got := value(t, setOf(common(), fm("f", c.output, c.root)), "f", c.inputs...); got != c.want {
			t.Errorf("%s: %s", c.name, got)
		}
	}
}

func TestRoundingAndNegativeDivisors(t *testing.T) {
	rounded := func(quantum, mode string, root Node, inputs ...string) string {
		formula := fm("f", number, root)
		formula.Quantum, formula.Mode = quantum, mode
		return value(t, setOf(common(), formula), "f", inputs...)
	}
	quotient := op(NodeDiv, vr("a"), vr("b"))
	cases := []struct {
		a, b  string
		exact string
		want  map[string]string
	}{
		{"10", "-4", "-5/2", map[string]string{"toward_zero": "-2", "away_from_zero": "-3", "floor": "-3", "ceiling": "-2", "half_up": "-3", "half_even": "-2"}},
		{"7", "-2", "-7/2", map[string]string{"toward_zero": "-3", "away_from_zero": "-4", "floor": "-4", "ceiling": "-3", "half_up": "-4", "half_even": "-4"}},
		{"-7", "-2", "7/2", map[string]string{"toward_zero": "3", "away_from_zero": "4", "floor": "3", "ceiling": "4", "half_up": "4", "half_even": "4"}},
		{"5", "2", "5/2", map[string]string{"toward_zero": "2", "away_from_zero": "3", "floor": "2", "ceiling": "3", "half_up": "3", "half_even": "2"}},
		{"7", "3", "7/3", map[string]string{"toward_zero": "2", "away_from_zero": "3", "floor": "2", "ceiling": "3", "half_up": "2", "half_even": "2"}},
		{"8", "3", "8/3", map[string]string{"toward_zero": "2", "away_from_zero": "3", "floor": "2", "ceiling": "3", "half_up": "3", "half_even": "3"}},
		{"-8", "3", "-8/3", map[string]string{"toward_zero": "-2", "away_from_zero": "-3", "floor": "-3", "ceiling": "-2", "half_up": "-3", "half_even": "-3"}},
		{"6", "-3", "-2", map[string]string{"toward_zero": "-2", "away_from_zero": "-2", "floor": "-2", "ceiling": "-2", "half_up": "-2", "half_even": "-2"}},
	}
	for _, c := range cases {
		if len(c.want) != len(modes) {
			t.Fatalf("modes of %s/%s", c.a, c.b)
		}
		for mode, want := range c.want {
			if got := rounded("1", mode, quotient, "a", c.a, "b", c.b); got != "f="+want+"("+c.exact+")" {
				t.Errorf("%s/%s %s: %s", c.a, c.b, mode, got)
			}
		}
	}
	pair := func(kind string) Node {
		return op(kind, op(NodeDiv, con("10", number), con("-4", number)), op(NodeDiv, con("7", number), con("-2", number)))
	}
	if got := rounded("0.1", "half_even", pair(NodeMax)); got != "f=-2.5(-5/2)" {
		t.Fatalf("a maximum after negative divisors: %s", got)
	}
	if got := rounded("0.1", "half_even", pair(NodeMin)); got != "f=-3.5(-7/2)" {
		t.Fatalf("a minimum after negative divisors: %s", got)
	}
	if got := rounded("0.1", "half_even", op(NodeGrossUp, con("-1", number), op(NodeDiv, con("1", number), con("-4", number)))); got != "f:margin_out_of_range" {
		t.Fatalf("a margin below zero through a negative divisor: %s", got)
	}
	quanta := map[string]string{"1": "1235", "0.1": "1234.6", "0.01": "1234.57", "0.001": "1234.568", "0.0001": "1234.5678"}
	for quantum, want := range quanta {
		if got := rounded(quantum, "half_even", vr("a"), "a", "1234.56785"); got != "f="+want+"(24691357/20000)" {
			t.Errorf("quantum %s: %s", quantum, got)
		}
	}
	if got := rounded("0.01", "half_even", vr("a"), "a", "2.355"); got != "f=2.36(471/200)" {
		t.Fatalf("a half to the even side upward: %s", got)
	}
	if got := rounded("0.01", "away_from_zero", vr("a"), "a", "2.34"); got != "f=2.34(117/50)" {
		t.Fatalf("an exact value is left: %s", got)
	}
}

func TestReferences(t *testing.T) {
	third := fm("g", usd, op(NodeDiv, vr("cost"), con("3", number)))
	amount := setOf(common(), fm("f", usd, op(NodeMul, ref("g"), con("3", number))), third)
	if got := value(t, amount, "f", "cost", "100"); got != "f=99.99(9999/100) g=33.33(100/3)" {
		t.Fatalf("an amount is seen rounded: %s", got)
	}
	share := fm("s", number, op(NodeDiv, vr("a"), vr("b")))
	exact := setOf(common(), fm("f", usd, op(NodeMul, vr("cost"), ref("s"))), share)
	if got := value(t, exact, "f", "cost", "100", "a", "1", "b", "3"); got != "f=33.33(100/3) s=0.33(1/3)" {
		t.Fatalf("a share is seen exact: %s", got)
	}
	small := setOf(common(), fm("f", number, op(NodeDiv, vr("cost"), ref("g"))), fm("g", usd, op(NodeMul, vr("cost"), con("0.001", number))))
	if got := value(t, small, "f", "cost", "1"); got != "f:division_by_zero g=0(1/1000)" {
		t.Fatalf("an amount that rounds to zero as a divisor: %s", got)
	}
	price := setOf(common(), fm("f", usd, op(NodeMul, vr("hours"), ref("p"))), fm("p", usdHour, op(NodeDiv, vr("cost"), vr("hours"))))
	if got := value(t, price, "f", "cost", "1", "hours", "1000000"); got != "f=1(1) p=0(1/1000000)" {
		t.Fatalf("a price shown as zero and used exact: %s", got)
	}
	margin := fm("m", number, op(NodeDiv, vr("a"), vr("b")))
	margin.Quantum = "0.0001"
	grossed := setOf(common(), fm("f", usd, op(NodeGrossUp, vr("cost"), ref("m"))), margin)
	if got := value(t, grossed, "f", "cost", "1", "a", "99996", "b", "100000"); got != "f=25000(25000) m=1(24999/25000)" {
		t.Fatalf("a margin shown as one and below it: %s", got)
	}
	if got := value(t, grossed, "f", "cost", "1", "a", "100004", "b", "100000"); got != "f:margin_out_of_range m=1(25001/25000)" {
		t.Fatalf("a margin shown as one and above it: %s", got)
	}
	nested := setOf(common(),
		fm("f", usd, op(NodeAdd, ref("g"), ref("h"))),
		fm("g", usd, op(NodeAdd, ref("h"), ref("h"))),
		fm("h", usd, vr("cost")),
		fm("z", usd, vr("hours")))
	nested.Formulas[3].Output, nested.Formulas[3].Root = hour, vr("hours")
	if got := value(t, nested, "f", "cost", "1.004"); got != "f=3(3) g=2(2) h=1(251/250)" {
		t.Fatalf("formulas reached once and listed by identifier: %s", got)
	}
	ranked := func(id string, rank int, root Node) Formula {
		formula := fm(id, usd, root)
		formula.Target, formula.Rank = "t", rank
		return formula
	}
	ranks := setOf(common(), ranked("a", 2, op(NodeMul, vr("cost"), con("2", number))), ranked("b", 1, op(NodeAdd, ref("a"), vr("cost"))), ranked("c", 3, vr("cost")))
	if got := value(t, ranks, "t", "cost", "10"); got != "b=30(30) a=20(20)" {
		t.Fatalf("the lowest rank, referring to another: %s", got)
	}
	alone := setOf(common(), ranked("a", 2, con("5", usd)), ranked("b", 1, vr("cost")))
	if got := value(t, alone, "t"); got != "b:variable_not_known" {
		t.Fatalf("nothing falls back to another rank: %s", got)
	}
}

func TestReasonsAndTheirOrder(t *testing.T) {
	variables := vars(decl("cost", usd), decl("x", usd), decl("z", number), decl("m", number), decl("big", number))
	product := op(NodeMul, op(NodeMul, vr("big"), vr("big")), op(NodeMul, vr("big"), vr("big")))
	given := []string{"cost", "10", "z", "0", "m", "2", "big", twelve}
	cases := []struct {
		name   string
		output Type
		root   Node
		want   string
	}{
		{"a variable not given", usd, vr("x"), "f:variable_not_known"},
		{"a divisor of zero", usd, op(NodeDiv, vr("cost"), vr("z")), "f:division_by_zero"},
		{"a dividend of zero", number, op(NodeDiv, vr("z"), vr("m")), "f=0(0)"},
		{"a margin above one", usd, op(NodeGrossUp, vr("cost"), vr("m")), "f:margin_out_of_range"},
		{"the left operand first", usd, op(NodeAdd, vr("x"), op(NodeDiv, vr("cost"), vr("z"))), "f:variable_not_known"},
		{"the left operand first, the other way", usd, op(NodeAdd, op(NodeDiv, vr("cost"), vr("z")), vr("x")), "f:division_by_zero"},
		{"an operand before the node's own test", usd, op(NodeDiv, vr("x"), vr("z")), "f:variable_not_known"},
		{"the right operand before the node's own test", usd, op(NodeDiv, vr("cost"), op(NodeGrossUp, vr("z"), vr("m"))), "f:margin_out_of_range"},
		{"a margin after its value", usd, op(NodeGrossUp, op(NodeDiv, vr("cost"), vr("z")), vr("m")), "f:division_by_zero"},
		{"a division by one less a rate is not a margin", usd, op(NodeDiv, vr("cost"), op(NodeSub, con("1", number), vr("m"))), "f=-10(-10)"},
		{"a reason on the left of an overflow", number, op(NodeAdd, op(NodeDiv, vr("m"), vr("z")), product), "f:division_by_zero"},
	}
	for _, c := range cases {
		if got := value(t, setOf(variables, fm("f", c.output, c.root)), "f", given...); got != c.want {
			t.Errorf("%s: %s", c.name, got)
		}
	}
	margins := map[string]string{"1": "f:margin_out_of_range", "1.000001": "f:margin_out_of_range", "-0.000001": "f:margin_out_of_range", "0": "f=10(10)", "0.5": "f=20(20)"}
	for margin, want := range margins {
		if got := value(t, setOf(variables, fm("f", usd, op(NodeGrossUp, vr("cost"), vr("m")))), "f", "cost", "10", "m", margin); got != want {
			t.Errorf("margin %s: %s", margin, got)
		}
	}
	if got := value(t, setOf(variables, fm("f", usd, vr("x"))), "f", "x", ""); got != "f:variable_not_known" {
		t.Fatalf("an empty text: %s", got)
	}
	over := setOf(variables, fm("f", number, op(NodeAdd, product, op(NodeDiv, vr("m"), vr("z")))))
	if _, err := must(t, over).Evaluate("f", in(given...)); kind(err) != "limit" {
		t.Fatalf("an overflow on the left of a reason: %v", err)
	}
	behind := setOf(variables, fm("f", number, op(NodeAdd, op(NodeDiv, vr("m"), vr("z")), ref("g"))), fm("g", number, product))
	if _, err := must(t, behind).Evaluate("f", in(given...)); kind(err) != "limit" {
		t.Fatalf("an overflow behind a reference comes before a reason: %v", err)
	}
	two := func(left, right string) Set {
		return setOf(variables, fm("f", usd, op(NodeAdd, ref(left), ref(right))), fm("g", usd, vr("x")), fm("h", usd, op(NodeDiv, vr("cost"), vr("z"))))
	}
	if got := value(t, two("g", "h"), "f", given...); got != "f:variable_not_known g:variable_not_known h:division_by_zero" {
		t.Fatalf("two referenced reasons: %s", got)
	}
	if got := value(t, two("h", "g"), "f", given...); got != "f:division_by_zero g:variable_not_known h:division_by_zero" {
		t.Fatalf("two referenced reasons, the other way: %s", got)
	}
}

func TestLimitsOfAnEvaluation(t *testing.T) {
	variables := vars(decl("big", number), decl("c", number), decl("cost", usd))
	deep := func(kind string) Set {
		return setOf(variables, fm("f", number, op(kind, con("1", number), op(NodeMul, op(NodeMul, vr("big"), vr("big")), vr("c")))))
	}
	for _, operation := range []string{NodeDiv, NodeAdd, NodeSub, NodeMul} {
		if _, err := must(t, deep(operation)).Evaluate("f", in("big", twelve, "c", "999999")); err != nil && operation == NodeDiv {
			t.Fatalf("thirty digits: %v", err)
		}
		if _, err := must(t, deep(operation)).Evaluate("f", in("big", twelve, "c", "9999999")); kind(err) != "limit" {
			t.Fatalf("thirty-one digits under %s: %v", operation, err)
		}
	}
	result, err := must(t, deep(NodeDiv)).Evaluate("f", in("big", twelve, "c", "999999"))
	if err != nil || result.Outcome.Value != "0" || len(strings.TrimPrefix(result.Outcome.Exact, "1/")) != 30 {
		t.Fatalf("a denominator of thirty digits: %v %s", err, result.Outcome.Exact)
	}
	grossed := setOf(variables, fm("f", number, op(NodeGrossUp, op(NodeMul, vr("big"), vr("big")), op(NodeDiv, con("1", number), vr("c")))))
	if _, err := must(t, grossed).Evaluate("f", in("big", twelve, "c", "9999999")); kind(err) != "limit" {
		t.Fatalf("thirty-one digits after a gross-up: %v", err)
	}
	small := testLimits()
	small.Digits = 6
	for _, root := range []Node{op(NodeMax, vr("c"), vr("c")), op(NodeMin, vr("c"), vr("c")), op(NodeMax, vr("c"), con("1", number)), op(NodeMin, con("999999", number), vr("c"))} {
		measured, _, _ := Compile(testProfile(t), small, setOf(variables, fm("f", number, root)))
		if _, err := measured.Evaluate("f", in("c", "123456.789")); kind(err) != "limit" {
			t.Fatalf("a %s of nine digits under a limit of six: %v", root.Kind, err)
		}
		if _, err := measured.Evaluate("f", in("c", "123.456")); err != nil {
			t.Fatalf("a %s of six digits: %v", root.Kind, err)
		}
	}
	held := func(factor string) Set {
		return setOf(variables, fm("f", number, op(NodeMul, vr("big"), con(factor, number))))
	}
	if got := value(t, held("100"), "f", "big", twelve); got != "f=99999999999900(99999999999900)" {
		t.Fatalf("fourteen whole digits: %s", got)
	}
	if _, err := must(t, held("1000")).Evaluate("f", in("big", twelve)); kind(err) != "limit" {
		t.Fatalf("fifteen whole digits: %v", err)
	}
	reached := setOf(variables, fm("f", number, op(NodeMin, ref("g"), con("1", number))), fm("g", number, op(NodeMul, vr("big"), con("1000", number))))
	if _, err := must(t, reached).Evaluate("f", in("big", twelve)); kind(err) != "limit" {
		t.Fatalf("a reached value the profile cannot hold: %v", err)
	}
	unreached := setOf(variables, fm("f", number, con("1", number)), fm("g", number, op(NodeMul, vr("big"), con("1000", number))))
	if got := value(t, unreached, "f", "big", twelve); got != "f=1(1)" {
		t.Fatalf("a formula that is not reached is not computed: %s", got)
	}
}

func TestInputsAndTargets(t *testing.T) {
	compiled := must(t, setOf(common(), fm("f", usd, vr("cost"))))
	cases := []struct {
		name   string
		target string
		inputs []Input
		want   string
	}{
		{"given", "f", in("cost", "1"), "ok"},
		{"nothing given", "f", nil, "ok"},
		{"negative", "f", in("cost", "-1.5"), "ok"},
		{"every variable", "f", in("cost", "1", "rate", "1", "hours", "1", "unit", "1", "a", "1", "b", "1"), "ok"},
		{"more inputs than variables may be", "f", in("cost", "1", "rate", "1", "hours", "1", "unit", "1", "a", "1", "b", "1", "c", "1"), "limit"},
		{"too many before their form", "nowhere", in("x y", "1", "x y", "1", "x y", "1", "x y", "1", "x y", "1", "x y", "1", "x y", "1"), "limit"},
		{"a name the set does not declare", "f", in("price", "1"), "invalid"},
		{"a name twice", "f", in("cost", "1", "cost", "1"), "invalid"},
		{"a name twice, once not known", "f", in("cost", "", "cost", "1"), "invalid"},
		{"a text that is not a number", "f", in("cost", "1,5"), "invalid"},
		{"a text beyond the profile", "f", in("cost", "1234567890123"), "invalid"},
		{"a text too long", "f", in("cost", strings.Repeat("1", 65)), "invalid"},
		{"an input before the target", "nowhere", in("cost", "x"), "invalid"},
		{"a target outside its form", "", in("cost", "1"), "invalid"},
		{"a target the set does not have", "nowhere", in("cost", "1"), "not_found"},
		{"a variable is not a target", "cost", in("cost", "1"), "not_found"},
	}
	for _, c := range cases {
		result, err := compiled.Evaluate(c.target, c.inputs)
		if kind(err) != c.want {
			t.Errorf("%s: %v", c.name, err)
		}
		if err != nil && (result.Semantics != "" || result.Reached != nil) {
			t.Errorf("%s: a result beside an error", c.name)
		}
	}
	result, err := compiled.Evaluate("f", in("cost", "2.5"))
	want := Result{Semantics: "formula_v1", ProfileID: "synthetic", ProfileRevision: "7", Target: "f", Outcome: Outcome{Formula: "f", Type: usd, Value: "2.5", Exact: "5/2"}, Reached: []Outcome{}}
	if err != nil || result.Semantics != want.Semantics || result.ProfileID != want.ProfileID || result.ProfileRevision != want.ProfileRevision ||
		result.Target != want.Target || result.Outcome != want.Outcome || result.Reached == nil || len(result.Reached) != 0 ||
		result.Set != "" || result.Revision != 0 || result.Digest != "" {
		t.Fatalf("result: %v %+v", err, result)
	}
	unknown, err := compiled.Evaluate("f", nil)
	if err != nil || unknown.Outcome != (Outcome{Formula: "f", Type: usd, Reason: "variable_not_known"}) {
		t.Fatalf("no value: %v %+v", err, unknown.Outcome)
	}
}

func TestAChainAtTheLimits(t *testing.T) {
	leaf := func(first Node) Node {
		return op(NodeAdd, op(NodeAdd, op(NodeAdd, first, con("1", number)), con("1", number)), con("1", number))
	}
	set := setOf(vars(decl("a", number)), fm("f1", number, leaf(vr("a"))))
	for _, id := range []string{"f2", "f3", "f4", "f5", "f6"} {
		previous := set.Formulas[len(set.Formulas)-1].ID
		set.Formulas = append(set.Formulas, fm(id, number, leaf(ref(previous))))
	}
	set.Formulas[0], set.Formulas[5] = set.Formulas[5], set.Formulas[0]
	if got := value(t, set, "f6", "a", "0.5"); got != "f6=18.5(37/2) f1=3.5(7/2) f2=6.5(13/2) f3=9.5(19/2) f4=12.5(25/2) f5=15.5(31/2)" {
		t.Fatalf("a chain of six formulas of depth four: %s", got)
	}
}

func TestAValueDoesNotDependOnProfileOrLimits(t *testing.T) {
	set := setOf(common(), fm("f", usd, op(NodeMul, vr("cost"), ref("s"))), fm("s", number, op(NodeDiv, vr("a"), vr("b"))))
	base := value(t, set, "f", "cost", "100", "a", "1", "b", "3")
	wide, found, err := Compile(profileOf(t, 18, 8, 20), Limits{Variables: 1000, Formulas: 1000, Nodes: 10000, Depth: 100, Digits: 10000}, set)
	if err != nil || wide == nil {
		t.Fatalf("compile: %v %v", err, found)
	}
	result, err := wide.Evaluate("f", in("cost", "100", "a", "1", "b", "3"))
	if err != nil || shown(result) != base {
		t.Fatalf("another profile and larger limits: %v %s against %s", err, shown(result), base)
	}
}
