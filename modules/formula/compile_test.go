package formula

import (
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func TestCompileRefusesAProfileOrLimitsThatAreNotValid(t *testing.T) {
	good := testLimits()
	if _, _, err := Compile(decimal.Profile{}, good, plain("2")); kind(err) != "invalid" {
		t.Fatalf("profile: %v", err)
	}
	cases := map[string]func(*Limits){
		"variables zero": func(l *Limits) { l.Variables = 0 }, "variables high": func(l *Limits) { l.Variables = 1001 },
		"formulas zero": func(l *Limits) { l.Formulas = 0 }, "formulas high": func(l *Limits) { l.Formulas = 1001 },
		"nodes zero": func(l *Limits) { l.Nodes = 0 }, "nodes high": func(l *Limits) { l.Nodes = 10001 },
		"depth zero": func(l *Limits) { l.Depth = 0 }, "depth high": func(l *Limits) { l.Depth = 101 },
		"digits zero": func(l *Limits) { l.Digits = 0 }, "digits high": func(l *Limits) { l.Digits = 10001 },
	}
	for name, change := range cases {
		limits := good
		change(&limits)
		if _, _, err := Compile(testProfile(t), limits, plain("2")); kind(err) != "invalid" {
			t.Errorf("%s: %v", name, err)
		}
	}
	top := Limits{Variables: 1000, Formulas: 1000, Nodes: 10000, Depth: 100, Digits: 10000}
	if compiled, _, err := Compile(testProfile(t), top, plain("2")); err != nil || compiled == nil {
		t.Fatalf("limits at their bounds: %v", err)
	}
	low := Limits{Variables: 1, Formulas: 1, Nodes: 3, Depth: 2, Digits: 1}
	if compiled, found, err := Compile(testProfile(t), low, plain("2")); err != nil || compiled == nil {
		t.Fatalf("limits of one: %v %v", err, found)
	}
}

func chain(depth int) Node {
	node := con("1", number)
	for level := 1; level < depth; level++ {
		node = op(NodeAdd, node, con("1", number))
	}
	return node
}

func TestCountsAreJudgedBeforeAnyElement(t *testing.T) {
	many := setOf(nil)
	for index := 0; index < 7; index++ {
		many.Variables = append(many.Variables, decl("bad name", number))
	}
	if got := check(t, many); got != "too_many@" {
		t.Fatalf("variables: %s", got)
	}
	six := setOf(nil)
	for index := 0; index < 6; index++ {
		six.Variables = append(six.Variables, decl(string(rune('a'+index)), number))
		six.Formulas = append(six.Formulas, fm(string(rune('p'+index)), number, con("1", number)))
	}
	if got := check(t, six); got != "" {
		t.Fatalf("six of each: %s", got)
	}
	six.Formulas = append(six.Formulas, Formula{ID: "bad id", Root: chain(9)})
	if got := check(t, six); got != "too_many@" {
		t.Fatalf("formulas: %s", got)
	}
}

func TestTreesAreBoundedByVisits(t *testing.T) {
	if got := check(t, setOf(nil, fm("f", number, chain(4)))); got != "" {
		t.Fatalf("depth four: %s", got)
	}
	if got := check(t, setOf(nil, fm("f", number, chain(5)))); got != "too_deep@f" {
		t.Fatalf("depth five: %s", got)
	}
	pair := op(NodeAdd, con("1", number), con("1", number))
	four := op(NodeAdd, pair, pair)
	fifteen := op(NodeAdd, op(NodeAdd, pair, pair), op(NodeAdd, pair, pair))
	if got := check(t, setOf(nil, fm("f", number, fifteen))); got != "" {
		t.Fatalf("fifteen nodes: %s", got)
	}
	limits := testLimits()
	limits.Nodes = 14
	if _, found, _ := Compile(testProfile(t), limits, setOf(nil, fm("f", number, fifteen))); codes(found) != "too_large@f" {
		t.Fatalf("fifteen nodes under fourteen: %s", codes(found))
	}
	limits.Nodes, limits.Depth = 7, 3
	if _, found, _ := Compile(testProfile(t), limits, setOf(nil, fm("f", number, four))); codes(found) != "" {
		t.Fatalf("seven nodes of one shared pair: %s", codes(found))
	}
	limits.Nodes = 6
	if _, found, _ := Compile(testProfile(t), limits, setOf(nil, fm("f", number, four))); codes(found) != "too_large@f" {
		t.Fatalf("a shared subtree counts each time: %s", codes(found))
	}
	operands := make([]Node, 2)
	operands[0] = Node{Kind: NodeAdd, Operands: operands}
	operands[1] = Node{Kind: NodeAdd, Operands: operands}
	self := Node{Kind: NodeAdd, Operands: operands}
	if got := check(t, setOf(nil, fm("f", number, self))); got != "too_deep@f" {
		t.Fatalf("a tree that contains itself: %s", got)
	}
	limits = testLimits()
	limits.Nodes, limits.Depth = 4, 4
	if _, found, _ := Compile(testProfile(t), limits, setOf(nil, fm("f", number, self))); codes(found) != "too_large@f" {
		t.Fatalf("both too deep and too large, the count first: %s", codes(found))
	}
	limits.Nodes, limits.Depth = 4, 3
	left := op(NodeAdd, op(NodeAdd, chain(2), con("1", number)), con("1", number))
	if _, found, _ := Compile(testProfile(t), limits, setOf(nil, fm("f", number, left))); codes(found) != "too_deep@f" {
		t.Fatalf("the fourth visit is too deep before a fifth is too many: %s", codes(found))
	}
}

func TestMalformedContent(t *testing.T) {
	long := strings.Repeat("a", 129)
	huge := strings.Repeat("9", 1<<20)
	good := func() Formula { return fm("f", usd, op(NodeMul, vr("cost"), con("2", number))) }
	set := func(change func(*Formula)) Set {
		formula := good()
		change(&formula)
		return setOf(vars(decl("cost", usd)), formula)
	}
	cases := map[string]Set{
		"target":               set(func(f *Formula) { f.Target = "" }),
		"target long":          set(func(f *Formula) { f.Target = long }),
		"rank zero":            set(func(f *Formula) { f.Rank = 0 }),
		"rank high":            set(func(f *Formula) { f.Rank = 1000 }),
		"output":               set(func(f *Formula) { f.Output = Type{Kind: "amount"} }),
		"output unit on money": set(func(f *Formula) { f.Output = Type{Kind: KindMoney, Currency: "USD", Unit: "hour"} }),
		"quantum":              set(func(f *Formula) { f.Quantum = "0.00001" }),
		"quantum empty":        set(func(f *Formula) { f.Quantum = "" }),
		"mode":                 set(func(f *Formula) { f.Mode = "half_down" }),
		"kind":                 set(func(f *Formula) { f.Root = Node{Kind: "call"} }),
		"kind empty":           set(func(f *Formula) { f.Root = Node{} }),
		"unknown kind with operands": set(func(f *Formula) {
			f.Root = Node{Kind: "call", Operands: []Node{con("1", number), con("1", number)}}
		}),
		"constant with operands": set(func(f *Formula) {
			f.Root = Node{Kind: NodeConstant, Number: "1", Type: usd, Operands: []Node{con("1", number), con("1", number)}}
		}),
		"variable with operands":  set(func(f *Formula) { f.Root = Node{Kind: NodeVariable, Name: "cost", Operands: []Node{con("1", number)}} }),
		"reference with operands": set(func(f *Formula) { f.Root = Node{Kind: NodeReference, Formula: "f", Operands: []Node{con("1", number)}} }),
		"constant text":           set(func(f *Formula) { f.Root = con("1e3", usd) }),
		"constant empty":          set(func(f *Formula) { f.Root = con("", usd) }),
		"constant huge":           set(func(f *Formula) { f.Root = con(huge, usd) }),
		"constant beyond profile": set(func(f *Formula) { f.Root = con("1234567890123", usd) }),
		"constant type":           set(func(f *Formula) { f.Root = con("1", Type{Kind: KindMoney}) }),
		"constant with name":      set(func(f *Formula) { f.Root = Node{Kind: NodeConstant, Number: "1", Type: usd, Name: "cost"} }),
		"constant with formula":   set(func(f *Formula) { f.Root = Node{Kind: NodeConstant, Number: "1", Type: usd, Formula: "f"} }),
		"variable name":           set(func(f *Formula) { f.Root = vr("co st") }),
		"variable long":           set(func(f *Formula) { f.Root = vr(long) }),
		"variable with number":    set(func(f *Formula) { f.Root = Node{Kind: NodeVariable, Name: "cost", Number: "1"} }),
		"variable with type":      set(func(f *Formula) { f.Root = Node{Kind: NodeVariable, Name: "cost", Type: usd} }),
		"variable with formula":   set(func(f *Formula) { f.Root = Node{Kind: NodeVariable, Name: "cost", Formula: "f"} }),
		"reference id":            set(func(f *Formula) { f.Root = ref("") }),
		"reference with number":   set(func(f *Formula) { f.Root = Node{Kind: NodeReference, Formula: "g", Number: "1"} }),
		"reference with name":     set(func(f *Formula) { f.Root = Node{Kind: NodeReference, Formula: "g", Name: "cost"} }),
		"reference with type":     set(func(f *Formula) { f.Root = Node{Kind: NodeReference, Formula: "g", Type: usd} }),
		"one operand":             set(func(f *Formula) { f.Root = Node{Kind: NodeAdd, Operands: []Node{vr("cost")}} }),
		"no operand":              set(func(f *Formula) { f.Root = Node{Kind: NodeAdd} }),
		"three operands":          set(func(f *Formula) { f.Root = Node{Kind: NodeAdd, Operands: []Node{vr("cost"), vr("cost"), vr("cost")}} }),
		"operation with number":   set(func(f *Formula) { f.Root.Number = "1" }),
		"operation with name":     set(func(f *Formula) { f.Root.Name = "cost" }),
		"operation with formula":  set(func(f *Formula) { f.Root.Formula = "f" }),
		"operation with type":     set(func(f *Formula) { f.Root.Type = usd }),
		"nested":                  set(func(f *Formula) { f.Root = op(NodeAdd, vr("cost"), op(NodeAdd, vr("cost"), con("x", usd))) }),
	}
	for name, content := range cases {
		if got := check(t, content); got != "malformed@f" {
			t.Errorf("%s: %s", name, got)
		}
	}
	whole := map[string]Set{
		"variable name":     setOf(vars(decl("", usd)), good()),
		"variable long":     setOf(vars(decl(long, usd)), good()),
		"variable type":     setOf(vars(decl("cost", Type{Kind: KindPrice, Currency: "USD"})), good()),
		"variable currency": setOf(vars(decl("cost", Type{Kind: KindMoney, Currency: "usd"})), good()),
		"variable number":   setOf(vars(decl("cost", Type{Kind: KindNumber, Unit: "hour"})), good()),
		"variable quantity": setOf(vars(decl("cost", Type{Kind: KindQuantity, Unit: "hour", Currency: "USD"})), good()),
		"variable twice":    setOf(vars(decl("cost", usd), decl("cost", usd)), good()),
		"currency first":    setOf(vars(decl("cost", Type{Kind: KindMoney, Currency: "uSD"})), good()),
		"currency second":   setOf(vars(decl("cost", Type{Kind: KindMoney, Currency: "UsD"})), good()),
		"currency third":    setOf(vars(decl("cost", Type{Kind: KindMoney, Currency: "USd"})), good()),
		"currency long":     setOf(vars(decl("cost", Type{Kind: KindMoney, Currency: "USDX"})), good()),
		"number in money":   setOf(vars(decl("cost", Type{Kind: KindNumber, Currency: "USD"})), good()),
		"quantity alone":    setOf(vars(decl("cost", Type{Kind: KindQuantity})), good()),
		"price of a unit":   setOf(vars(decl("cost", Type{Kind: KindPrice, Unit: "hour"})), good()),
		"formula id":        setOf(vars(decl("cost", usd)), Formula{ID: "f f", Root: chain(9)}),
		"formula id long":   setOf(vars(decl("cost", usd)), Formula{ID: huge}),
	}
	for name, content := range whole {
		if got := check(t, content); got != "malformed@" {
			t.Errorf("%s: %s", name, got)
		}
	}
	twice := setOf(vars(decl("cost", usd)), good(), good())
	if got := check(t, twice); got != "malformed@" {
		t.Fatalf("an identifier twice: %s", got)
	}
	both := setOf(vars(decl("cost", usd)), fm("a", number, chain(5)), fm("b", number, con("x", number)), fm("c", number, chain(5)), fm("d", number, Node{Kind: "call"}))
	both.Formulas[0].Mode = "none"
	if got := check(t, both); got != "too_deep@a too_deep@c malformed@b malformed@d" {
		t.Fatalf("sizes and forms together: %s", got)
	}
	three := Node{Kind: NodeAdd, Operands: []Node{chain(2), chain(2), chain(2)}}
	tight := testLimits()
	tight.Nodes = 4
	if _, found, _ := Compile(testProfile(t), tight, setOf(nil, fm("f", number, three))); codes(found) != "malformed@f" {
		t.Fatalf("a malformed operation is not followed: %s", codes(found))
	}
	wide, err := decimal.NewProfile(decimal.ProfileSpec{ID: "wide", Revision: "1", InputIntegerDigits: 70, InputScale: 2, ResultIntegerDigits: 70, ResultScale: 4, IntermediateDigits: 80})
	if err != nil {
		t.Fatal(err)
	}
	for digits, want := range map[int]string{64: "", 65: "malformed@f"} {
		_, found, _ := Compile(wide, testLimits(), setOf(nil, fm("f", number, con(strings.Repeat("9", digits), number))))
		if codes(found) != want {
			t.Fatalf("a constant of %d bytes: %s", digits, codes(found))
		}
	}
	lengths, _, _ := Compile(wide, testLimits(), setOf(vars(decl("a", number)), fm("f", number, vr("a"))))
	if _, err := lengths.Evaluate("f", in("a", strings.Repeat("9", 64))); err != nil {
		t.Fatalf("an input of 64 bytes: %v", err)
	}
	if _, err := lengths.Evaluate("f", in("a", strings.Repeat("9", 65))); kind(err) != "invalid" {
		t.Fatalf("an input of 65 bytes: %v", err)
	}
	narrow := profileOf(t, 12, 3, 14)
	if _, found, _ := Compile(narrow, testLimits(), setOf(vars(decl("cost", usd)), set(func(f *Formula) { f.Quantum = "0.0001" }).Formulas[0])); codes(found) != "malformed@f" {
		t.Fatalf("a quantum the profile cannot read: %s", codes(found))
	}
	if _, found, _ := Compile(narrow, testLimits(), setOf(vars(decl("cost", usd)), set(func(f *Formula) { f.Quantum = "0.001" }).Formulas[0])); codes(found) != "" {
		t.Fatalf("a quantum at the profile's scale: %s", codes(found))
	}
	for _, quantum := range []string{"1", "0.1", "0.01", "0.001", "0.0001"} {
		for mode := range modes {
			formula := good()
			formula.Quantum, formula.Mode = quantum, mode
			if got := check(t, setOf(vars(decl("cost", usd)), formula)); got != "" {
				t.Errorf("%s %s: %s", quantum, mode, got)
			}
		}
	}
	if len(modes) != 6 {
		t.Fatalf("modes: %d", len(modes))
	}
	edge := good()
	edge.Rank, edge.Target = 999, strings.Repeat("t", 128)
	if got := check(t, setOf(vars(decl("cost", usd)), edge)); got != "" {
		t.Fatalf("values at their bounds: %s", got)
	}
}

func TestSemanticFindings(t *testing.T) {
	cost := vars(decl("cost", usd), decl("rate", number))
	cases := []struct {
		name string
		set  Set
		want string
	}{
		{"an unknown variable", setOf(cost, fm("f", usd, vr("price"))), "unknown_name@f"},
		{"an unknown formula", setOf(cost, fm("f", usd, ref("g"))), "unknown_name@f"},
		{"a name of a variable is not a formula", setOf(cost, fm("f", usd, ref("cost"))), "unknown_name@f"},
		{"a name of a formula is not a variable", setOf(cost, fm("f", usd, vr("f"))), "unknown_name@f"},
		{"an operation refused", setOf(cost, fm("f", usd, op(NodeAdd, vr("cost"), vr("rate")))), "type_mismatch@f"},
		{"a root of another type", setOf(cost, fm("f", krw, vr("cost"))), "type_mismatch@f"},
		{"no finding that follows from an unknown name", setOf(cost, fm("f", krw, op(NodeAdd, vr("price"), vr("rate")))), "unknown_name@f"},
		{"no finding that follows from a mismatch", setOf(cost, fm("f", krw, op(NodeMul, op(NodeAdd, vr("cost"), vr("rate")), vr("cost")))), "type_mismatch@f"},
		{"a constant divisor of zero", setOf(cost, fm("f", usd, op(NodeDiv, vr("cost"), con("0", number)))), "constant_division_by_zero@f"},
		{"zero written otherwise", setOf(cost, fm("f", usd, op(NodeDiv, vr("cost"), con("-0.00", number)))), "constant_division_by_zero@f"},
		{"a constant divisor that is not zero", setOf(cost, fm("f", usd, op(NodeDiv, vr("cost"), con("0.000001", number)))), ""},
		{"a constant dividend of zero", setOf(cost, fm("f", number, op(NodeDiv, con("0", number), vr("rate")))), ""},
		{"nothing is folded", setOf(cost, fm("f", usd, op(NodeDiv, vr("cost"), op(NodeSub, con("1", number), con("1", number))))), ""},
		{"zero as another operand", setOf(cost, fm("f", usd, op(NodeMul, vr("cost"), con("0", number)))), ""},
		{"a margin of one", setOf(cost, fm("f", usd, op(NodeGrossUp, vr("cost"), con("1", number)))), "margin_out_of_range@f"},
		{"a margin of one written otherwise", setOf(cost, fm("f", usd, op(NodeGrossUp, vr("cost"), con("1.000", number)))), "margin_out_of_range@f"},
		{"a margin above one", setOf(cost, fm("f", usd, op(NodeGrossUp, vr("cost"), con("1.5", number)))), "margin_out_of_range@f"},
		{"a margin below zero", setOf(cost, fm("f", usd, op(NodeGrossUp, vr("cost"), con("-0.000001", number)))), "margin_out_of_range@f"},
		{"a margin of zero", setOf(cost, fm("f", usd, op(NodeGrossUp, vr("cost"), con("0", number)))), ""},
		{"a margin just below one", setOf(cost, fm("f", usd, op(NodeGrossUp, vr("cost"), con("0.999999", number)))), ""},
		{"one as a divisor", setOf(cost, fm("f", usd, op(NodeDiv, vr("cost"), con("1", number)))), ""},
		{"zero as a margin's value", setOf(cost, fm("f", number, op(NodeGrossUp, con("0", number), vr("rate")))), ""},
		{"a margin of the wrong type and range", setOf(cost, fm("f", usd, op(NodeGrossUp, vr("cost"), con("2", usd)))), "type_mismatch@f margin_out_of_range@f"},
		{"itself", setOf(cost, fm("f", usd, ref("f"))), "cycle@f"},
		{"two", setOf(cost, fm("f", usd, ref("g")), fm("g", usd, ref("f"))), "cycle@f cycle@g"},
		{"three and one that reaches them and one they reach", setOf(cost,
			fm("a", usd, ref("b")), fm("b", usd, ref("c")), fm("c", usd, op(NodeAdd, ref("a"), ref("z"))),
			fm("r", usd, ref("a")), fm("z", usd, vr("cost"))), "cycle@a cycle@b cycle@c cycle@r"},
		{"a cycle beside an unknown formula", setOf(cost, fm("f", usd, op(NodeAdd, ref("f"), ref("none"))), fm("g", usd, ref("none")), fm("h", usd, ref("g"))), "unknown_name@f unknown_name@g cycle@f"},
		{"a formula named twice in one tree", setOf(cost, fm("f", usd, op(NodeAdd, ref("g"), ref("g"))), fm("g", usd, vr("cost"))), ""},
		{"several", setOf(cost,
			fm("a", usd, op(NodeDiv, vr("none"), con("0", number))),
			fm("b", krw, op(NodeGrossUp, vr("cost"), con("1", number))),
			fm("c", usd, ref("c"))), "unknown_name@a type_mismatch@b constant_division_by_zero@a margin_out_of_range@b cycle@c"},
	}
	for _, c := range cases {
		if got := check(t, c.set); got != c.want {
			t.Errorf("%s: %s", c.name, got)
		}
	}
	ranked := func(id, target string, rank int) Formula {
		formula := fm(id, usd, vr("cost"))
		formula.Target, formula.Rank = target, rank
		return formula
	}
	conflict := setOf(cost, ranked("a", "t", 2), ranked("b", "t", 2), ranked("c", "t", 1), ranked("d", "u", 2), ranked("e", "t", 2))
	if got := check(t, conflict); got != "rank_conflict@a rank_conflict@b rank_conflict@e" {
		t.Fatalf("rank conflict: %s", got)
	}
	if got := check(t, setOf(cost, ranked("a", "t", 2), ranked("b", "t", 2), ranked("c", "u", 2))); got != "rank_conflict@a rank_conflict@b" {
		t.Fatalf("two of one rank: %s", got)
	}
	if got := check(t, setOf(cost, ranked("a", "t", 1), ranked("b", "t", 2), ranked("c", "u", 1))); got != "" {
		t.Fatalf("ranks that differ: %s", got)
	}
}

func TestTypeRules(t *testing.T) {
	types := []Type{number, usd, krw, hour, gb, usdHour, krwHour, usdGB}
	name := func(kind Type) string { return kind.Kind + ":" + kind.Currency + ":" + kind.Unit }
	allowed := map[string]map[[2]Type]Type{
		NodeMul: {
			{number, number}: number, {number, usd}: usd, {number, krw}: krw, {number, hour}: hour, {number, gb}: gb,
			{number, usdHour}: usdHour, {number, krwHour}: krwHour, {number, usdGB}: usdGB,
			{usd, number}: usd, {krw, number}: krw, {hour, number}: hour, {gb, number}: gb,
			{usdHour, number}: usdHour, {krwHour, number}: krwHour, {usdGB, number}: usdGB,
			{hour, usdHour}: usd, {usdHour, hour}: usd, {hour, krwHour}: krw, {krwHour, hour}: krw, {gb, usdGB}: usd, {usdGB, gb}: usd,
		},
		NodeDiv: {
			{number, number}: number, {usd, number}: usd, {krw, number}: krw, {hour, number}: hour, {gb, number}: gb,
			{usdHour, number}: usdHour, {krwHour, number}: krwHour, {usdGB, number}: usdGB,
			{usd, usd}: number, {krw, krw}: number, {hour, hour}: number, {gb, gb}: number,
			{usdHour, usdHour}: number, {krwHour, krwHour}: number, {usdGB, usdGB}: number,
			{usd, hour}: usdHour, {usd, gb}: usdGB, {krw, hour}: krwHour, {krw, gb}: Type{Kind: KindPrice, Currency: "KRW", Unit: "gb"},
			{usd, usdHour}: hour, {usd, usdGB}: gb, {krw, krwHour}: hour,
		},
		NodeGrossUp: {
			{number, number}: number, {usd, number}: usd, {krw, number}: krw, {hour, number}: hour, {gb, number}: gb,
			{usdHour, number}: usdHour, {krwHour, number}: krwHour, {usdGB, number}: usdGB,
		},
	}
	for _, same := range []string{NodeAdd, NodeSub, NodeMin, NodeMax} {
		allowed[same] = map[[2]Type]Type{}
		for _, kind := range types {
			allowed[same][[2]Type{kind, kind}] = kind
		}
	}
	if len(allowed) != len(operations) {
		t.Fatalf("operations: %d", len(allowed))
	}
	counted := 0
	for operation, pairs := range allowed {
		for _, left := range types {
			for _, right := range types {
				variables := vars(decl("l", left), decl("r", right))
				root := op(operation, vr("l"), vr("r"))
				want, ok := pairs[[2]Type{left, right}]
				outputs := append([]Type{Type{Kind: KindPrice, Currency: "KRW", Unit: "gb"}}, types...)
				for _, output := range outputs {
					got := check(t, setOf(variables, fm("f", output, root)))
					expected := "type_mismatch@f"
					if ok && output == want {
						expected = ""
						counted++
					}
					if got != expected {
						t.Errorf("%s %s %s as %s: %q", operation, name(left), name(right), name(output), got)
					}
				}
			}
		}
	}
	if counted != 21+22+8+4*8 {
		t.Fatalf("allowed combinations met: %d", counted)
	}
}

func TestCompiledSharesNothingWithTheCaller(t *testing.T) {
	set := setOf(vars(decl("cost", usd)), fm("f", usd, op(NodeMul, vr("cost"), con("2", number))))
	compiled := must(t, set)
	set.Variables[0].Name = "other"
	set.Formulas[0].Root.Operands[1] = con("3", number)
	set.Formulas[0].Root.Kind = NodeDiv
	set.Formulas[0].Target = "g"
	result, err := compiled.Evaluate("f", in("cost", "10"))
	if err != nil || shown(result) != "f=20(20)" {
		t.Fatalf("after the caller changed its set: %v %s", err, shown(result))
	}
}
