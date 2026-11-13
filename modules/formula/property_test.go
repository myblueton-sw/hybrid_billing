package formula

import (
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func TestRoundingAgreesWithThePrimitive(t *testing.T) {
	profile, err := decimal.NewProfile(decimal.ProfileSpec{ID: "wide", Revision: "1", InputIntegerDigits: 9, InputScale: 4, ResultIntegerDigits: 18, ResultScale: 6, IntermediateDigits: 60})
	if err != nil {
		t.Fatal(err)
	}
	random := rand.New(rand.NewSource(206))
	ties, negative := 0, 0
	for round := 0; round < 20000; round++ {
		numerator := fmt.Sprintf("%d.%d", random.Intn(2000)-1000, random.Intn(4)*25)
		denominator := fmt.Sprintf("%d.%d", 1+random.Intn(40), random.Intn(2)*5)
		if random.Intn(4) == 0 {
			denominator = []string{"2", "4", "8", "0.5"}[random.Intn(4)]
		}
		if random.Intn(3) == 0 {
			denominator = "-" + denominator
			negative++
		}
		top, _ := decimal.Parse(profile, numerator)
		bottom, _ := decimal.Parse(profile, denominator)
		value := of(top).over(of(bottom))
		if value.den.Sign() <= 0 || new(big.Int).GCD(nil, nil, new(big.Int).Abs(value.num), value.den).Cmp(bigOne) != 0 {
			t.Fatalf("%s over %s is kept as %s/%s", numerator, denominator, value.num, value.den)
		}
		for scale := 0; scale <= 4; scale++ {
			for name := range modes {
				mode, err := decimal.ParseMode(name)
				if err != nil {
					t.Fatal(err)
				}
				want, err := decimal.DivideRound(profile, top, bottom, scale, mode)
				if err != nil {
					t.Fatal(err)
				}
				got, err := decimal.New(profile, value.quanta(scale, name), scale)
				if err != nil || !got.Equal(want) {
					t.Fatalf("%s over %s at scale %d under %s: %s, the primitive %s (%v)", numerator, denominator, scale, name, got, want, err)
				}
			}
			scaled := new(big.Int).Mul(value.num, power(scale))
			if new(big.Int).Rem(new(big.Int).Mul(scaled, bigTwo), value.den).Sign() == 0 && new(big.Int).Rem(scaled, value.den).Sign() != 0 {
				ties++
			}
		}
	}
	if ties < 2000 || negative < 5000 {
		t.Fatalf("only %d ties and %d negative divisors", ties, negative)
	}
}

var pool = map[Type]string{number: "n", usd: "m", hour: "q", usdHour: "p"}

var productions = map[Type][][3]interface{}{
	number:  {{NodeMul, number, number}, {NodeDiv, number, number}, {NodeDiv, usd, usd}, {NodeDiv, hour, hour}, {NodeDiv, usdHour, usdHour}},
	usd:     {{NodeMul, number, usd}, {NodeMul, usd, number}, {NodeMul, hour, usdHour}, {NodeMul, usdHour, hour}, {NodeDiv, usd, number}},
	hour:    {{NodeMul, number, hour}, {NodeMul, hour, number}, {NodeDiv, hour, number}, {NodeDiv, usd, usdHour}},
	usdHour: {{NodeMul, number, usdHour}, {NodeMul, usdHour, number}, {NodeDiv, usdHour, number}, {NodeDiv, usd, hour}},
}

type grower struct {
	random    *rand.Rand
	depth     int
	reference Type
	refers    bool
	fresh     bool
	declared  []Variable
}

func (g *grower) leaf(kind Type, margin bool) Node {
	if g.fresh {
		name := fmt.Sprintf("v%d", len(g.declared))
		g.declared = append(g.declared, decl(name, kind))
		return vr(name)
	}
	if kind == g.reference && g.random.Intn(4) == 0 {
		g.refers = true
		return ref("g")
	}
	if g.random.Intn(3) == 0 {
		texts := []string{"1", "2", "-3", "0.5", "-0.25", "10", "7.125", "100", "0.001"}
		if margin {
			texts = []string{"0", "0.1", "0.25", "0.5", "0.999"}
		}
		return con(texts[g.random.Intn(len(texts))], kind)
	}
	return vr(fmt.Sprintf("%s%d", pool[kind], 1+g.random.Intn(2)))
}

func (g *grower) node(kind Type, depth int, margin bool) Node {
	if depth >= g.depth || g.random.Intn(5) == 0 {
		return g.leaf(kind, margin)
	}
	switch pick := g.random.Intn(8); {
	case pick < 4:
		return op([]string{NodeAdd, NodeSub, NodeMin, NodeMax}[pick], g.node(kind, depth+1, false), g.node(kind, depth+1, false))
	case pick == 4:
		return op(NodeGrossUp, g.node(kind, depth+1, false), g.node(number, depth+1, true))
	}
	rule := productions[kind][g.random.Intn(len(productions[kind]))]
	return op(rule[0].(string), g.node(rule[1].(Type), depth+1, false), g.node(rule[2].(Type), depth+1, false))
}

type oracle struct {
	values map[string]*big.Rat
	digits int
	seen   *big.Rat
	reason string
}

func (o *oracle) fits(value *big.Rat) (*big.Rat, string, bool) {
	if len(new(big.Int).Abs(value.Num()).String()) > o.digits || len(value.Denom().String()) > o.digits {
		return nil, "", true
	}
	return value, "", false
}

func (o *oracle) eval(node Node) (*big.Rat, string, bool) {
	switch node.Kind {
	case NodeConstant:
		value, _ := new(big.Rat).SetString(node.Number)
		return value, "", false
	case NodeVariable:
		if o.values[node.Name] == nil {
			return nil, "variable_not_known", false
		}
		return o.values[node.Name], "", false
	case NodeReference:
		return o.seen, o.reason, false
	}
	left, reason, over := o.eval(node.Operands[0])
	if over || reason != "" {
		return nil, reason, over
	}
	right, reason, over := o.eval(node.Operands[1])
	if over || reason != "" {
		return nil, reason, over
	}
	one := big.NewRat(1, 1)
	switch node.Kind {
	case NodeAdd:
		return o.fits(new(big.Rat).Add(left, right))
	case NodeSub:
		return o.fits(new(big.Rat).Sub(left, right))
	case NodeMul:
		return o.fits(new(big.Rat).Mul(left, right))
	case NodeDiv:
		if right.Sign() == 0 {
			return nil, "division_by_zero", false
		}
		return o.fits(new(big.Rat).Quo(left, right))
	case NodeMin:
		if left.Cmp(right) <= 0 {
			return o.fits(left)
		}
		return o.fits(right)
	case NodeMax:
		if left.Cmp(right) >= 0 {
			return o.fits(left)
		}
		return o.fits(right)
	}
	if right.Sign() < 0 || right.Cmp(one) >= 0 {
		return nil, "margin_out_of_range", false
	}
	return o.fits(new(big.Rat).Quo(left, new(big.Rat).Sub(one, right)))
}

func roundRat(value *big.Rat, scale int, mode string) *big.Int {
	scaled := new(big.Rat).Mul(value, new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil)))
	floor := new(big.Int).Div(scaled.Num(), scaled.Denom())
	part := new(big.Rat).Sub(scaled, new(big.Rat).SetInt(floor))
	if part.Sign() == 0 {
		return floor
	}
	ceiling := new(big.Int).Add(floor, big.NewInt(1))
	half := part.Cmp(big.NewRat(1, 2))
	positive := scaled.Sign() > 0
	pick := func(up bool) *big.Int {
		if up {
			return ceiling
		}
		return floor
	}
	switch mode {
	case "floor":
		return floor
	case "ceiling":
		return ceiling
	case "toward_zero":
		return pick(!positive)
	case "away_from_zero":
		return pick(positive)
	case "half_up":
		return pick(half > 0 || (half == 0 && positive))
	}
	return pick(half > 0 || (half == 0 && floor.Bit(0) == 1))
}

func ratText(value *big.Rat) string {
	if value.IsInt() {
		return value.Num().String()
	}
	return value.Num().String() + "/" + value.Denom().String()
}

func TestWellTypedTreesAgainstAReference(t *testing.T) {
	profile := testProfile(t)
	random := rand.New(rand.NewSource(2060))
	types := []Type{number, usd, hour, usdHour}
	variables := []Variable{}
	for _, kind := range types {
		variables = append(variables, decl(pool[kind]+"1", kind), decl(pool[kind]+"2", kind))
	}
	texts := []string{"0", "1", "2", "3", "0.5", "-1", "-2.5", "10", "0.25", "7", "100", "0.1", "-0.75", "0.999", "123456.789"}
	quantums := []string{"1", "0.1", "0.01", "0.001", "0.0001"}
	names := []string{"toward_zero", "away_from_zero", "floor", "ceiling", "half_up", "half_even"}
	met := map[string]int{}
	for round := 0; round < 30000; round++ {
		limits := Limits{Variables: 8, Formulas: 2, Nodes: 40, Depth: 5, Digits: []int{6, 12, 40}[random.Intn(3)]}
		output, inner := types[random.Intn(4)], types[random.Intn(4)]
		g := &grower{random: random, depth: 5, reference: inner}
		first := Formula{ID: "f", Target: "f", Rank: 1, Output: output, Quantum: quantums[random.Intn(5)], Mode: names[random.Intn(6)], Root: g.node(output, 1, false)}
		second := Formula{ID: "g", Target: "g", Rank: 1, Output: inner, Quantum: quantums[random.Intn(5)], Mode: names[random.Intn(6)], Root: (&grower{random: random, depth: 4}).node(inner, 1, false)}
		compiled, found, err := Compile(profile, limits, setOf(variables, first, second))
		if err != nil || compiled == nil {
			t.Fatalf("round %d: a tree built to be well typed is refused: %v %v", round, err, found)
		}
		o := &oracle{values: map[string]*big.Rat{}, digits: limits.Digits}
		inputs := []Input{}
		for _, variable := range variables {
			switch random.Intn(8) {
			case 0:
			case 1:
				inputs = append(inputs, Input{Name: variable.Name})
			default:
				text := texts[random.Intn(len(texts))]
				inputs = append(inputs, Input{Name: variable.Name, Value: text})
				o.values[variable.Name], _ = new(big.Rat).SetString(text)
			}
		}
		want, limit := "", false
		finish := func(formula Formula) (string, *big.Rat, string) {
			exact, reason, over := o.eval(formula.Root)
			if over {
				limit = true
				return "", nil, ""
			}
			if reason != "" {
				return formula.ID + ":" + reason, nil, reason
			}
			units := roundRat(exact, quanta[formula.Quantum], formula.Mode)
			shown, err := decimal.New(profile, units, quanta[formula.Quantum])
			if err != nil {
				limit = true
				return "", nil, ""
			}
			seen := exact
			if formula.Output.Kind == KindMoney {
				seen = new(big.Rat).SetFrac(units, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(quanta[formula.Quantum])), nil))
			}
			return formula.ID + "=" + shown.String() + "(" + ratText(exact) + ")", seen, ""
		}
		tail := ""
		if g.refers {
			tail, o.seen, o.reason = finish(second)
			tail = " " + tail
		}
		if !limit {
			head, _, reason := finish(first)
			want = head + tail
			if reason == "" {
				met["value"]++
			} else {
				met[reason]++
			}
			if g.refers && o.reason != "" && reason == o.reason {
				met["a reason through a reference"]++
			}
			if g.refers && o.reason == "" && inner.Kind == KindMoney {
				met["an amount seen rounded"]++
			}
			if g.refers && o.reason == "" && inner.Kind != KindMoney {
				met["a value seen exact"]++
			}
		}
		result, err := compiled.Evaluate("f", inputs)
		if limit {
			met["limit"]++
			if kind(err) != "limit" {
				t.Fatalf("round %d: a limit error expected: %v %s", round, err, shown(result))
			}
			continue
		}
		if err != nil || shown(result) != want {
			t.Fatalf("round %d: %v\n got %s\nwant %s\n%+v\n%+v\n%v", round, err, shown(result), want, first, second, inputs)
		}
	}
	for name, least := range map[string]int{"value": 3000, "variable_not_known": 3000, "division_by_zero": 300, "margin_out_of_range": 1000, "limit": 300,
		"a reason through a reference": 300, "an amount seen rounded": 100, "a value seen exact": 300} {
		if met[name] < least {
			t.Errorf("%s met %d times", name, met[name])
		}
	}
}

func TestOnePlantedDefectIsRefusedWithItsCode(t *testing.T) {
	profile := testProfile(t)
	limits := Limits{Variables: 16, Formulas: 1, Nodes: 40, Depth: 5, Digits: 40}
	random := rand.New(rand.NewSource(2061))
	types := []Type{number, usd, hour, usdHour}
	planted := map[string]int{}
	for round := 0; round < 20000; round++ {
		output := types[random.Intn(4)]
		g := &grower{random: random, depth: 5, fresh: true}
		root := g.node(output, 1, false)
		set := setOf(g.declared, fm("f", output, root))
		if _, found, _ := Compile(profile, limits, set); len(found) != 0 {
			t.Fatalf("round %d: refused before a defect: %v", round, found)
		}
		changed := setOf(append([]Variable{}, g.declared...), fm("f", output, root))
		at := random.Intn(len(changed.Variables))
		want := "type_mismatch@f"
		switch random.Intn(3) {
		case 0:
			changed.Variables[at].Name = "renamed"
			want = "unknown_name@f"
		default:
			other := types[random.Intn(4)]
			for other == changed.Variables[at].Type {
				other = types[random.Intn(4)]
			}
			changed.Variables[at].Type = other
		}
		_, found, _ := Compile(profile, limits, changed)
		if codes(found) != want {
			t.Fatalf("round %d: %s for %s\n%+v\n%+v", round, codes(found), want, root, changed.Variables)
		}
		planted[want]++
		if len(g.declared) > 1 {
			planted["several leaves"]++
		}
	}
	if planted["type_mismatch@f"] < 5000 || planted["unknown_name@f"] < 5000 || planted["several leaves"] < 5000 {
		t.Fatalf("planted: %v", planted)
	}
}

type modelRevision struct {
	state    string
	author   string
	content  string
	impact   *Impact
	approval *[3]string
	history  int
	compiled bool
	reason   string
}

func TestHistoriesAgainstAModel(t *testing.T) {
	random := rand.New(rand.NewSource(2062))
	contents := map[string]Set{"2": plain("2"), "3": plain("3"), "bad": setOf(nil, fm("f", usd, ref("f"))),
		"malformed": setOf(nil, fm("f", usd, con("x", usd))), "deep": setOf(nil, fm("f", number, chain(5)))}
	keys := []string{"2", "2", "2", "3", "3", "bad", "malformed", "deep"}
	met := map[string]bool{}
	note := func(operation, outcome string) { met[operation+" "+outcome] = true }
	for history := 0; history < 4000; history++ {
		clock := &testClock{now: start}
		limits := registryLimits()
		limits.Revisions = 4
		r, _ := NewRegistry(testProfile(t), limits, clock.read)
		model := map[int]*modelRevision{}
		active, epoch, sequence := 0, 0, 0
		digests := map[int]string{}
		for step := 0; step < 60; step++ {
			number := 1 + random.Intn(5)
			m := model[number]
			var err error
			want := "ok"
			operation := []string{"draft", "check", "impact", "impact", "approval", "approval", "activate", "activate", "withdraw", "evaluate"}[random.Intn(10)]
			if step < 12 && random.Intn(2) == 0 {
				operation = []string{"draft", "check", "impact", "approval", "activate"}[step%5]
			}
			before, beforeErr := r.Active("ws", "set")
			switch operation {
			case "draft":
				key, author := keys[random.Intn(len(keys))], []string{"author", "author", "approver"}[random.Intn(3)]
				switch {
				case key == "deep":
					want = "limit"
				case key == "malformed":
					want = "invalid"
				case m != nil && (m.content != key || m.author != author):
					want = "conflict"
				case m != nil:
				case len(model) >= 4:
					want = "limit"
					met["draft without room"] = true
				default:
					model[number] = &modelRevision{state: "draft", author: author, content: key}
					sequence++
				}
				_, err = r.Draft("ws", "set", number, author, contents[key])
			case "check":
				switch {
				case m == nil:
					want = "not_found"
				case m.state == "withdrawn":
					want = "rejected"
				case m.state == "draft":
					m.state, m.compiled = "checked", true
					if m.content == "bad" {
						m.state, m.compiled = "check_failed", false
					}
					sequence++
				}
				_, err = r.RunCheck("ws", "set", number)
			case "impact":
				impact := full(epoch)
				switch random.Intn(10) {
				case 0:
					impact.Epoch = random.Intn(4)
				case 1:
					impact.Coverage = CoverageSample
				case 2:
					impact.WithoutValue = 1
				case 3:
					impact.Failed = 2
				case 4:
					impact.Changed = 11
				case 5:
					impact.Changed = random.Intn(3)
				}
				switch {
				case impact.Changed > impact.Evaluated:
					want = "invalid"
				case m == nil:
					want = "not_found"
				case m.impact != nil && *m.impact == impact:
				case m.impact != nil && m.approval != nil && m.impact.Epoch == epoch:
					want = "conflict"
				case (m.state != "checked" && m.state != "impact_recorded" && m.state != "approval_recorded") || impact.Epoch != epoch:
					want = "rejected"
				case m.impact != nil && m.history >= 2:
					want = "limit"
				default:
					if m.impact != nil {
						m.history++
					}
					m.impact, m.approval, m.state = &impact, nil, "impact_recorded"
					sequence++
				}
				_, err = r.RecordImpact("ws", "set", number, impact)
			case "approval":
				approver := []string{"approver", "approver", "author", "second"}[random.Intn(4)]
				subject := digestA
				if view, viewErr := r.Revision("ws", "set", number); viewErr == nil && view.Impact != nil && random.Intn(6) != 0 {
					subject = view.Impact.Subject
				}
				given := [3]string{approver, "ref-1", subject}
				switch {
				case m == nil:
					want = "not_found"
				case m.impact != nil && m.approval != nil && *m.approval != given:
					want = "conflict"
				case m.impact != nil && m.approval != nil:
				case m.state != "impact_recorded" || m.impact.Epoch != epoch || m.impact.Coverage != CoverageFull || m.impact.WithoutValue != 0 || m.impact.Failed != 0 ||
					subject == digestA || approver == m.author:
					want = "rejected"
				default:
					m.approval, m.state = &given, "approval_recorded"
					sequence++
				}
				_, err = r.RecordApproval("ws", "set", number, approver, "ref-1", subject)
			case "activate":
				switch {
				case m == nil:
					want = "not_found"
				case m.state == "active":
				case m.state != "approval_recorded" || m.impact.Epoch != epoch:
					want = "rejected"
				default:
					if active != 0 {
						model[active].state = "superseded"
					}
					m.state, active = "active", number
					epoch++
					sequence++
				}
				_, err = r.Activate("ws", "set", number)
			case "withdraw":
				reason := []string{"mistake", "mistake", "stopped", "none"}[random.Intn(4)]
				switch {
				case reason == "none":
					want = "invalid"
				case m == nil:
					want = "not_found"
				case m.state == "withdrawn" && m.reason != reason:
					want = "conflict"
				case m.state == "withdrawn":
				default:
					if m.state == "active" {
						active = 0
						epoch++
					}
					m.state, m.reason = "withdrawn", reason
					sequence++
				}
				_, err = r.Withdraw("ws", "set", number, reason)
			case "evaluate":
				switch {
				case m == nil:
					want = "not_found"
				case !m.compiled:
					want = "rejected"
				}
				var result Result
				result, err = r.EvaluateRevision("ws", "set", number, "price", in("cost", "10"))
				if err == nil && shown(result) != map[string]string{"2": "price=20(20)", "3": "price=30(30)"}[m.content] {
					t.Fatalf("history %d step %d: evaluation gives %s", history, step, shown(result))
				}
			}
			if kind(err) != want {
				t.Fatalf("history %d step %d: %s on %d gives %v, the model %s", history, step, operation, number, err, want)
			}
			note(operation, want)
			after, afterErr := r.Active("ws", "set")
			if len(model) == 0 {
				if afterErr == nil {
					t.Fatalf("history %d step %d: a set without a revision", history, step)
				}
				continue
			}
			if afterErr != nil || after.Revision != active || after.Epoch != epoch || after.Sequence != sequence {
				t.Fatalf("history %d step %d after %s: active %+v, the model %d %d %d", history, step, operation, after, active, epoch, sequence)
			}
			if beforeErr == nil {
				if after.Epoch < before.Epoch || (after.Epoch == before.Epoch+1) != (after.Revision != before.Revision) || after.Epoch > before.Epoch+1 {
					t.Fatalf("history %d step %d: the epoch went from %d to %d while the active revision went from %d to %d", history, step, before.Epoch, after.Epoch, before.Revision, after.Revision)
				}
				if operation == "activate" && after.Revision != before.Revision {
					view, _ := r.Revision("ws", "set", number)
					if view.Impact == nil || view.Impact.Impact.Epoch != before.Epoch {
						t.Fatalf("history %d step %d: activated on an impact of another epoch", history, step)
					}
				}
			}
			actives := 0
			for other, expected := range model {
				view, viewErr := r.Revision("ws", "set", other)
				if viewErr != nil || view.State != expected.state || len(view.History) != expected.history || (view.Impact == nil) != (expected.impact == nil) || view.Withdrawal != expected.reason {
					t.Fatalf("history %d step %d after %s: revision %d is %+v, the model %+v", history, step, operation, other, view, expected)
				}
				met["state "+view.State] = true
				if digests[other] == "" {
					digests[other] = view.Digest
				}
				if view.Digest != digests[other] {
					t.Fatalf("history %d step %d: the digest of %d changed", history, step, other)
				}
				if view.Impact != nil {
					if view.Impact.Impact != *expected.impact || (view.Impact.Approval == nil) != (expected.approval == nil) || view.Current != (expected.impact.Epoch == epoch) {
						t.Fatalf("history %d step %d: the impact of %d is %+v, the model %+v", history, step, other, view.Impact, expected.impact)
					}
				}
				if view.State == "active" {
					actives++
					approval := view.Impact.Approval
					if after.Revision != other || after.Digest != view.Digest || approval == nil || approval.Approver == view.Author || approval.Subject != view.Impact.Subject ||
						view.Impact.Impact.Coverage != CoverageFull || view.Impact.Impact.WithoutValue != 0 || view.Impact.Impact.Failed != 0 || view.Impact.Impact.Epoch >= after.Epoch {
						t.Fatalf("history %d step %d: revision %d is active as %+v", history, step, other, view)
					}
				}
			}
			if actives > 1 || (actives == 1) != (after.Revision != 0) {
				t.Fatalf("history %d step %d: %d active revisions", history, step, actives)
			}
		}
	}
	expected := []string{
		"draft ok", "draft conflict", "draft invalid", "draft limit", "draft without room",
		"check ok", "check not_found", "check rejected",
		"impact ok", "impact invalid", "impact not_found", "impact conflict", "impact rejected", "impact limit",
		"approval ok", "approval not_found", "approval conflict", "approval rejected",
		"activate ok", "activate not_found", "activate rejected",
		"withdraw ok", "withdraw invalid", "withdraw not_found", "withdraw conflict",
		"evaluate ok", "evaluate not_found", "evaluate rejected",
		"state draft", "state check_failed", "state checked", "state impact_recorded", "state approval_recorded", "state active", "state superseded", "state withdrawn",
	}
	for _, name := range expected {
		if !met[name] {
			t.Errorf("never met: %s", name)
		}
	}
}

func TestConcurrentActivation(t *testing.T) {
	for round := 0; round < 200; round++ {
		r, _ := open(t)
		approved(t, r, 1, "2")
		approved(t, r, 2, "3")
		gate := make(chan struct{})
		outcomes := make([]string, 2)
		var group sync.WaitGroup
		for index := 0; index < 2; index++ {
			group.Add(1)
			go func(index int) {
				defer group.Done()
				<-gate
				_, err := r.Activate("ws", "set", index+1)
				outcomes[index] = kind(err)
			}(index)
		}
		close(gate)
		group.Wait()
		if !(outcomes[0] == "ok" && outcomes[1] == "rejected") && !(outcomes[0] == "rejected" && outcomes[1] == "ok") {
			t.Fatalf("two activations at once: %v", outcomes)
		}
		active, _ := r.Active("ws", "set")
		if active.Epoch != 1 || active.Revision == 0 {
			t.Fatalf("active: %+v", active)
		}
	}
}

func TestAnApprovalBesideAnActivation(t *testing.T) {
	for round := 0; round < 200; round++ {
		r, _ := open(t)
		approved(t, r, 1, "2")
		subject := impacted(t, r, 2, "3")
		gate := make(chan struct{})
		var group sync.WaitGroup
		var activation, approval error
		group.Add(2)
		go func() {
			defer group.Done()
			<-gate
			_, activation = r.Activate("ws", "set", 1)
		}()
		go func() {
			defer group.Done()
			<-gate
			_, approval = r.RecordApproval("ws", "set", 2, "approver", "ref-1", subject)
		}()
		close(gate)
		group.Wait()
		if activation != nil || (kind(approval) != "ok" && kind(approval) != "rejected") {
			t.Fatalf("activation %v, approval %v", activation, approval)
		}
		if _, err := r.Activate("ws", "set", 2); kind(err) != "rejected" {
			t.Fatalf("the second revision became active on an approval of the earlier epoch: %v", err)
		}
	}
}

func TestEvaluationBesideChanges(t *testing.T) {
	r, _ := open(t)
	activated(t, r, 1, "2")
	approved(t, r, 2, "3")
	gate := make(chan struct{})
	var group sync.WaitGroup
	results := make([]string, 8)
	for index := range results {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			<-gate
			for round := 0; round < 50; round++ {
				result, err := r.EvaluateRevision("ws", "set", 1, "price", in("cost", "10.555"))
				results[index] = kind(err) + " " + shown(result) + " " + result.Digest
			}
		}(index)
	}
	group.Add(1)
	go func() {
		defer group.Done()
		<-gate
		r.Activate("ws", "set", 2)
		r.Withdraw("ws", "set", 1, "replaced")
		r.Withdraw("ws", "set", 2, "stopped")
	}()
	close(gate)
	group.Wait()
	view, _ := r.Revision("ws", "set", 1)
	for index, result := range results {
		if result != "ok price=21.11(2111/100) "+view.Digest {
			t.Fatalf("goroutine %d: %s", index, result)
		}
	}
}

func TestTheSourceUsesNoFloatingPointRunsNothingAndHasNoComments(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	read := 0
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		read++
		for _, word := range []string{"float", "Float", "big.Rat", "complex", "os/exec", "\"net", "\"os\"", "plugin", "reflect", "unsafe", "syscall", "//", "/*"} {
			if strings.Contains(string(body), word) {
				t.Errorf("%s contains %q", file, word)
			}
		}
	}
	if read != 7 {
		t.Fatalf("files read: %d", read)
	}
}
