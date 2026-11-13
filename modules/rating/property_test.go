package rating

import (
	"fmt"
	"math/big"
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func ratOf(text string) *big.Rat {
	value, read := new(big.Rat).SetString(text)
	if !read {
		panic(text)
	}
	return value
}

func modelRound(value *big.Rat, mode string) *big.Int {
	floor := new(big.Int).Div(value.Num(), value.Denom())
	if value.IsInt() {
		return floor
	}
	ceiling := new(big.Int).Add(floor, big.NewInt(1))
	negative := value.Sign() < 0
	middle := new(big.Rat).Add(new(big.Rat).SetInt(floor), big.NewRat(1, 2))
	side := value.Cmp(middle)
	switch mode {
	case "floor":
		return floor
	case "ceiling":
		return ceiling
	case "toward_zero":
		if negative {
			return ceiling
		}
		return floor
	case "away_from_zero":
		if negative {
			return floor
		}
		return ceiling
	case "half_up":
		if side > 0 || (side == 0 && !negative) {
			return ceiling
		}
		return floor
	}
	if side > 0 || (side == 0 && floor.Bit(0) == 1) {
		return ceiling
	}
	return floor
}

type modelRule struct {
	amounts  map[string]*big.Rat
	held     bool
	unknown  int
	total    *big.Rat
	rated    []string
	passed   []string
	weights  map[string]*big.Rat
	discount []*big.Rat
	before   []*big.Rat
	bounds   map[string]*big.Rat
}

type model struct {
	rules    map[string]*modelRule
	excluded map[string]string
	covered  map[string]string
	total    *big.Rat
}

func rateModel(p Policy, lines []Line) model {
	out := model{rules: map[string]*modelRule{}, excluded: map[string]string{}, covered: map[string]string{}, total: new(big.Rat)}
	quantum := ratOf(p.Quantum)
	toQuanta := func(value *big.Rat) *big.Rat {
		return new(big.Rat).SetInt(modelRound(new(big.Rat).Quo(value, quantum), p.Mode))
	}
	eligible := map[Class]bool{ServiceFee: p.Path == Direct}
	if p.Path == Resale {
		for _, class := range p.Eligible {
			eligible[class] = true
		}
	}
	find := func(line Line) *Rule {
		for _, level := range []Level{SKU, Service, Group, Default} {
			key := map[Level]string{SKU: line.SKU, Service: line.Service, Group: line.Group, Default: ""}[level]
			if level != Default && key == "" {
				continue
			}
			for index := range p.Rules {
				if p.Rules[index].Level == level && p.Rules[index].Key == key {
					return &p.Rules[index]
				}
			}
		}
		return nil
	}
	members := map[string][]Line{}
	byID := map[string]*Rule{}
	for _, line := range lines {
		rule := find(line)
		switch {
		case !eligible[line.Class]:
			out.excluded[line.ID] = ClassNotEligible
		case rule == nil:
			out.excluded[line.ID] = NoRule
		default:
			members[rule.ID] = append(members[rule.ID], line)
			byID[rule.ID] = rule
		}
	}
	for id, group := range members {
		rule := byID[id]
		made := &modelRule{amounts: map[string]*big.Rat{}, weights: map[string]*big.Rat{}, total: new(big.Rat)}
		out.rules[id] = made
		needsCost, needsQuantity := rule.Type == PassThrough || rule.Type == Markup || rule.Type == TargetMargin, rule.Type == UnitPrice || rule.Type == Graduated || rule.Type == Volume
		for _, line := range group {
			if (needsCost && line.Cost == "") || (needsQuantity && line.Quantity == "") {
				made.unknown++
			}
		}
		if made.unknown > 0 {
			made.held = true
			for _, line := range group {
				switch {
				case needsCost && line.Cost == "":
					out.excluded[line.ID] = CostNotKnown
				case needsQuantity && line.Quantity == "":
					out.excluded[line.ID] = QuantityNotKnown
				default:
					out.excluded[line.ID] = RuleHeld
				}
			}
			continue
		}
		base := new(big.Rat)
		passed := new(big.Rat)
		switch {
		case rule.Type == Flat:
			base = ratOf(rule.Amount)
			for _, line := range group {
				out.covered[line.ID] = id
			}
		case needsCost:
			above := new(big.Rat)
			for _, line := range group {
				value := ratOf(line.Cost)
				if value.Sign() < 0 {
					passed.Add(passed, value)
					made.passed = append(made.passed, line.ID)
					made.weights[line.ID] = new(big.Rat).Neg(value)
					continue
				}
				above.Add(above, value)
				made.rated = append(made.rated, line.ID)
				made.weights[line.ID] = value
			}
			switch rule.Type {
			case PassThrough:
				base = above
			case Markup:
				base = new(big.Rat).Mul(above, new(big.Rat).Add(big.NewRat(1, 1), ratOf(rule.Rate)))
			default:
				base = new(big.Rat).Quo(above, new(big.Rat).Sub(big.NewRat(1, 1), ratOf(rule.Margin)))
			}
		default:
			quantity := new(big.Rat)
			for _, line := range group {
				quantity.Add(quantity, ratOf(line.Quantity))
				made.rated = append(made.rated, line.ID)
				made.weights[line.ID] = ratOf(line.Quantity)
			}
			over := new(big.Rat)
			if rule.Included != "" {
				over.Sub(quantity, ratOf(rule.Included))
			} else {
				over.Set(quantity)
			}
			if over.Sign() < 0 {
				over.SetInt64(0)
			}
			unit := func(price Price) *big.Rat {
				return new(big.Rat).Quo(ratOf(price.Amount), big.NewRat(int64(price.Per), 1))
			}
			switch rule.Type {
			case UnitPrice:
				base.Mul(over, unit(rule.Price))
			case Graduated:
				lower := new(big.Rat)
				for _, each := range rule.Tiers {
					upper := new(big.Rat).Set(over)
					if each.UpTo != "" && ratOf(each.UpTo).Cmp(over) < 0 {
						upper = ratOf(each.UpTo)
					}
					if upper.Cmp(lower) > 0 {
						base.Add(base, new(big.Rat).Mul(new(big.Rat).Sub(upper, lower), unit(each.Price)))
						lower = upper
					}
				}
			default:
				for _, each := range rule.Tiers {
					if each.UpTo == "" || over.Cmp(ratOf(each.UpTo)) <= 0 {
						base.Mul(over, unit(each.Price))
						break
					}
				}
			}
		}
		left := toQuanta(base)
		made.amounts["base"] = new(big.Rat).Mul(left, quantum)
		made.amounts["passed"] = new(big.Rat).Mul(toQuanta(passed), quantum)
		for _, each := range rule.Discounts {
			var size *big.Rat
			if each.Percent != "" {
				size = new(big.Rat).SetInt(modelRound(new(big.Rat).Mul(left, ratOf(each.Percent)), p.Mode))
			} else {
				size = new(big.Rat).Quo(ratOf(each.Fixed), quantum)
			}
			if size.Cmp(left) > 0 {
				size = new(big.Rat).Set(left)
			}
			made.before = append(made.before, new(big.Rat).Mul(left, quantum))
			left = new(big.Rat).Sub(left, size)
			made.discount = append(made.discount, new(big.Rat).Neg(new(big.Rat).Mul(size, quantum)))
		}
		made.amounts["minimum"], made.amounts["maximum"] = new(big.Rat), new(big.Rat)
		made.bounds = map[string]*big.Rat{}
		if rule.Minimum != "" {
			made.bounds[StepMinimum] = new(big.Rat).Mul(left, quantum)
		}
		if rule.Minimum != "" {
			if floor := new(big.Rat).Quo(ratOf(rule.Minimum), quantum); left.Cmp(floor) < 0 {
				made.amounts["minimum"] = new(big.Rat).Mul(new(big.Rat).Sub(floor, left), quantum)
				left = floor
			}
		}
		if rule.Maximum != "" {
			made.bounds[StepMaximum] = new(big.Rat).Mul(left, quantum)
		}
		if rule.Maximum != "" {
			if cap := new(big.Rat).Quo(ratOf(rule.Maximum), quantum); left.Cmp(cap) > 0 {
				made.amounts["maximum"] = new(big.Rat).Mul(new(big.Rat).Sub(cap, left), quantum)
				left = cap
			}
		}
		made.total = new(big.Rat).Add(new(big.Rat).Mul(left, quantum), made.amounts["passed"])
		out.total.Add(out.total, made.total)
	}
	return out
}

type generated struct {
	policy Policy
	lines  []Line
}

func generate(rnd *rand.Rand) generated {
	pick := func(options ...string) string { return options[rnd.Intn(len(options))] }
	price := func() Price {
		return Price{Amount: pick("1", "0.5", "12.5", "3", "2500", "0"), Per: []int{1, 1, 1000, 1000000}[rnd.Intn(4)]}
	}
	p := Policy{ID: "pol", Version: "v1", Currency: "KRW", CostBasis: Billed, Quantum: pick("1", "1", "0.1", "0.01", "0.001", "0.0001"), Mode: pick("toward_zero", "away_from_zero", "floor", "ceiling", "half_up", "half_even"), Path: Resale}
	if rnd.Intn(6) == 0 {
		p.Path = Direct
	} else {
		for _, class := range []Class{CSPUsage, ServiceFee, Licence, Other} {
			if rnd.Intn(5) > 0 {
				p.Eligible = append(p.Eligible, class)
			}
		}
	}
	amount := func() string {
		return map[string][]string{"1": {"0", "10", "250", "40"}, "0.1": {"0", "10.5", "250", "0.1"}, "0.01": {"0", "10.25", "99.99", "3"}, "0.001": {"0", "10.001", "7.5", "0.002"}, "0.0001": {"0", "10.0001", "7.5", "0.0002"}}[p.Quantum][rnd.Intn(4)]
	}
	keys := []struct {
		level Level
		key   string
	}{{SKU, "k1"}, {SKU, "k2"}, {Service, "s1"}, {Group, "g1"}, {Default, ""}}
	for index, at := range keys {
		if rnd.Intn(4) == 0 {
			continue
		}
		rule := Rule{ID: fmt.Sprintf("rule%d", index), Level: at.level, Key: at.key, Type: Type(pick("pass_through", "markup", "target_margin", "unit_price", "graduated", "volume", "flat"))}
		if rule.Type == Flat && at.level == Default {
			rule.Type = Markup
		}
		switch rule.Type {
		case Markup:
			rule.Rate = pick("0", "0.2", "0.175", "1.5")
		case TargetMargin:
			rule.Margin = pick("0", "0.2", "0.3", "0.125", "0.9")
		case UnitPrice:
			rule.Price = price()
		case Graduated, Volume:
			bounds := []string{"1", "2.5", "10", "100"}
			for count, first := rnd.Intn(4), rnd.Intn(2); count > 0 && first < len(bounds); count, first = count-1, first+1 {
				rule.Tiers = append(rule.Tiers, Tier{UpTo: bounds[first], Price: price()})
			}
			rule.Tiers = append(rule.Tiers, Tier{Price: price()})
		case Flat:
			rule.Amount = amount()
		}
		if rule.Type.quantityBased() && rnd.Intn(3) == 0 {
			rule.Included = pick("0", "1", "3.5", "50")
		}
		for count := rnd.Intn(4); count > 0; count-- {
			if rnd.Intn(2) == 0 {
				rule.Discounts = append(rule.Discounts, Discount{Percent: pick("0", "0.1", "0.5", "0.333", "0.99")})
			} else {
				rule.Discounts = append(rule.Discounts, Discount{Fixed: amount()})
			}
		}
		if rnd.Intn(3) == 0 {
			rule.Minimum = amount()
		}
		if rnd.Intn(3) == 0 {
			rule.Maximum = amount()
			if rule.Minimum != "" && ratOf(rule.Minimum).Cmp(ratOf(rule.Maximum)) > 0 {
				rule.Minimum, rule.Maximum = rule.Maximum, rule.Minimum
			}
		}
		p.Rules = append(p.Rules, rule)
	}
	if len(p.Rules) == 0 {
		p.Rules = []Rule{{ID: "only", Level: Default, Type: PassThrough}}
	}
	var lines []Line
	for index, count := 0, rnd.Intn(12); index < count; index++ {
		line := Line{ID: fmt.Sprintf("l%02d", rnd.Intn(1000)*100+index), Class: Class(pick("csp_usage", "csp_usage", "service_fee", "licence", "other")), SKU: pick("", "", "k1", "k2", "k9"), Service: pick("", "s1", "s9"), Group: pick("", "g1"),
			Cost: pick("0", "100", "12.3456", "0.5", "0.5", "-30", "-0.5", "999.9999", "3", "7"), Quantity: pick("0", "1", "0.5", "0.5", "2.5", "10", "150", "1234.5678"), Currency: "KRW"}
		if rnd.Intn(25) == 0 {
			line.Cost = ""
		}
		if rnd.Intn(25) == 0 {
			line.Quantity = ""
		}
		lines = append(lines, line)
	}
	return generated{policy: p, lines: lines}
}

func TestRandomPoliciesAgainstAModelAndInvariants(t *testing.T) {
	profile := profileOf(t, 12, 4, 14)
	reached := map[string]int{}
	for seed := int64(1); seed <= 4000; seed++ {
		rnd := rand.New(rand.NewSource(seed))
		made := generate(rnd)
		compiled, err := Compile(profile, testLimits(), made.policy)
		if err != nil {
			t.Fatalf("seed %d: %v %+v", seed, err, made.policy)
		}
		got, err := compiled.Rate(made.lines)
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		want := rateModel(made.policy, made.lines)
		quantum := ratOf(made.policy.Quantum)
		sums := map[string]map[string]*big.Rat{}
		shares := map[string]map[string]map[string]*big.Rat{}
		discounts := map[string][]*big.Rat{}
		appear := map[string]int{}
		grand := new(big.Rat)
		for index, charge := range got.Charges {
			value := ratOf(charge.Amount)
			if !new(big.Rat).Quo(value, quantum).IsInt() {
				t.Fatalf("seed %d: the charge %+v is no multiple of %s", seed, charge, made.policy.Quantum)
			}
			if sums[charge.Rule] == nil {
				sums[charge.Rule], shares[charge.Rule] = map[string]*big.Rat{}, map[string]map[string]*big.Rat{Rated: {}, PassedAtCost: {}}
			}
			if sums[charge.Rule][charge.Kind] == nil {
				sums[charge.Rule][charge.Kind] = new(big.Rat)
			}
			sums[charge.Rule][charge.Kind].Add(sums[charge.Rule][charge.Kind], value)
			grand.Add(grand, value)
			switch charge.Kind {
			case Rated, PassedAtCost:
				shares[charge.Rule][charge.Kind][charge.Line] = value
				appear[charge.Line]++
			case DiscountCharge:
				discounts[charge.Rule] = append(discounts[charge.Rule], value)
				fallthrough
			default:
				if charge.Line != "" || value.Sign() == 0 {
					t.Fatalf("seed %d: a charge of a rule %+v", seed, charge)
				}
			}
			if index > 0 && got.Charges[index-1].Rule > charge.Rule {
				t.Fatalf("seed %d: charges out of the order of rules", seed)
			}
			reached["kind/"+charge.Kind]++
		}
		for _, each := range got.Covered {
			appear[each.Line]++
			if want.covered[each.Line] != each.Rule {
				t.Fatalf("seed %d: %+v, the model %q", seed, each, want.covered[each.Line])
			}
		}
		if len(got.Covered) != len(want.covered) || len(got.Excluded) != len(want.excluded) {
			t.Fatalf("seed %d: %d covered and %d excluded, the model %d and %d", seed, len(got.Covered), len(got.Excluded), len(want.covered), len(want.excluded))
		}
		for _, each := range got.Excluded {
			appear[each.Line]++
			if want.excluded[each.Line] != each.Reason {
				t.Fatalf("seed %d: %+v, the model %q", seed, each, want.excluded[each.Line])
			}
			reached["reason/"+each.Reason]++
		}
		for _, line := range made.lines {
			if appear[line.ID] != 1 {
				t.Fatalf("seed %d: the line %s appears %d times", seed, line.ID, appear[line.ID])
			}
		}
		if len(got.Traces) != len(want.rules) || got.Lines != len(made.lines) {
			t.Fatalf("seed %d: %d traces, the model %d rules", seed, len(got.Traces), len(want.rules))
		}
		held := false
		for _, trace := range got.Traces {
			rule := want.rules[trace.Rule]
			if rule == nil || ratOf(trace.Total).Cmp(rule.total) != 0 {
				t.Fatalf("seed %d rule %s: total %s, the model %v", seed, trace.Rule, trace.Total, rule)
			}
			if rule.held {
				held = true
				reached["held"]++
				if len(trace.Steps) != 1 || trace.Steps[0].Name != StepHeld || trace.Steps[0].Outcome != fmt.Sprint(rule.unknown) || sums[trace.Rule] != nil {
					t.Fatalf("seed %d rule %s: a held rule with %+v", seed, trace.Rule, trace.Steps)
				}
				continue
			}
			var kind Type
			for _, each := range made.policy.Rules {
				if each.ID == trace.Rule {
					kind = each.Type
				}
			}
			reached["type/"+string(kind)]++
			sum := func(name string) *big.Rat {
				if sums[trace.Rule] == nil || sums[trace.Rule][name] == nil {
					return new(big.Rat)
				}
				return sums[trace.Rule][name]
			}
			base := Rated
			if kind == Flat {
				base = FlatCharge
			}
			for name, value := range map[string]*big.Rat{"base": sum(base), "passed": sum(PassedAtCost), "minimum": sum(MinimumTopUp), "maximum": sum(MaximumCap)} {
				if value.Cmp(rule.amounts[name]) != 0 {
					t.Fatalf("seed %d rule %s (%s): %s is %s, the model %s\n%+v\n%+v", seed, trace.Rule, kind, name, value.FloatString(4), rule.amounts[name].FloatString(4), made.policy, made.lines)
				}
			}
			var modelled []*big.Rat
			for _, each := range rule.discount {
				if each.Sign() != 0 {
					modelled = append(modelled, each)
				}
			}
			if len(modelled) != len(discounts[trace.Rule]) {
				t.Fatalf("seed %d rule %s: %d discounts, the model %d", seed, trace.Rule, len(discounts[trace.Rule]), len(modelled))
			}
			for index, each := range modelled {
				if each.Cmp(discounts[trace.Rule][index]) != 0 {
					t.Fatalf("seed %d rule %s: discount %d", seed, trace.Rule, index)
				}
			}
			for name, ids := range map[string][]string{Rated: rule.rated, PassedAtCost: rule.passed} {
				amount := rule.amounts["base"]
				if name == PassedAtCost {
					amount = new(big.Rat).Neg(rule.amounts["passed"])
				}
				if kind == Flat {
					ids = nil
				}
				weight := new(big.Rat)
				for _, id := range ids {
					weight.Add(weight, rule.weights[id])
				}
				found := 0
				if shares[trace.Rule] != nil {
					found = len(shares[trace.Rule][name])
				}
				if found != len(ids) {
					t.Fatalf("seed %d rule %s: %d %s charges for %d lines", seed, trace.Rule, found, name, len(ids))
				}
				for _, id := range ids {
					share := new(big.Rat).Abs(shares[trace.Rule][name][id])
					if weight.Sign() == 0 {
						if share.Sign() != 0 {
							t.Fatalf("seed %d rule %s: a share without weight", seed, trace.Rule)
						}
						continue
					}
					exact := new(big.Rat).Mul(amount, new(big.Rat).Quo(rule.weights[id], weight))
					if distance := new(big.Rat).Abs(new(big.Rat).Sub(share, exact)); distance.Cmp(quantum) >= 0 || (rule.weights[id].Sign() == 0 && share.Sign() != 0) {
						t.Fatalf("seed %d rule %s line %s: share %s, exact %s", seed, trace.Rule, id, share.FloatString(4), exact.FloatString(6))
					}
				}
			}
			if last := trace.Steps[len(trace.Steps)-1]; last.Name != StepTotal || last.Outcome != trace.Total {
				t.Fatalf("seed %d rule %s: the last step %+v", seed, trace.Rule, last)
			}
			expected := []string{StepBaseAmount}
			switch {
			case kind.costBased():
				expected = []string{StepCostsAbove, StepCostsBelow, StepBase, StepBaseAmount, StepPassed, StepPassedAmount}
			case kind.quantityBased():
				expected = []string{StepQuantity, StepIncluded, StepAboveIncluded, StepBase, StepBaseAmount}
			}
			for range rule.discount {
				expected = append(expected, StepDiscount)
			}
			for _, name := range []string{StepMinimum, StepMaximum} {
				if rule.bounds[name] != nil {
					expected = append(expected, name)
				}
			}
			expected = append(expected, StepTotal)
			var names []string
			taken := 0
			for _, step := range trace.Steps {
				if step.Name == StepTier {
					if kind != Graduated && kind != Volume {
						t.Fatalf("seed %d rule %s: a tier step under %s", seed, trace.Rule, kind)
					}
					continue
				}
				names = append(names, step.Name)
				switch step.Name {
				case StepBaseAmount:
					if ratOf(step.Outcome).Cmp(rule.amounts["base"]) != 0 {
						t.Fatalf("seed %d rule %s: the step of the base amount says %s", seed, trace.Rule, step.Outcome)
					}
				case StepPassedAmount:
					if ratOf(step.Outcome).Cmp(rule.amounts["passed"]) != 0 {
						t.Fatalf("seed %d rule %s: the step of what is passed says %s", seed, trace.Rule, step.Outcome)
					}
				case StepDiscount:
					if ratOf(step.Basis).Cmp(rule.before[taken]) != 0 || ratOf(step.Outcome).Cmp(rule.discount[taken]) != 0 {
						t.Fatalf("seed %d rule %s: discount %d says %s of %s", seed, trace.Rule, taken, step.Outcome, step.Basis)
					}
					taken++
				case StepMinimum, StepMaximum:
					if ratOf(step.Basis).Cmp(rule.bounds[step.Name]) != 0 || ratOf(step.Outcome).Cmp(rule.amounts[step.Name]) != 0 {
						t.Fatalf("seed %d rule %s: the %s says %s on %s, the model %s", seed, trace.Rule, step.Name, step.Outcome, step.Basis, rule.bounds[step.Name].FloatString(4))
					}
				}
			}
			if strings.Join(names, " ") != strings.Join(expected, " ") {
				t.Fatalf("seed %d rule %s (%s): the steps %v, want %v", seed, trace.Rule, kind, names, expected)
			}
		}
		if ratOf(got.Total).Cmp(want.total) != 0 || grand.Cmp(want.total) != 0 || got.NoRuleHeld == held {
			t.Fatalf("seed %d: total %s, charges %s, the model %s; held %v", seed, got.Total, grand.FloatString(4), want.total.FloatString(4), held)
		}
		shuffled := append([]Line(nil), made.lines...)
		rnd.Shuffle(len(shuffled), func(a, b int) { shuffled[a], shuffled[b] = shuffled[b], shuffled[a] })
		again, err := compiled.Rate(shuffled)
		if err != nil || !reflect.DeepEqual(got, again) {
			t.Fatalf("seed %d: the lines in another order give another result", seed)
		}
		if len(made.lines) == 0 {
			continue
		}
		target := made.lines[rnd.Intn(len(made.lines))]
		if target.Cost == "" || target.Quantity == "" {
			continue
		}
		halves := func(text string) (string, string, bool) {
			value := ratOf(text)
			first := new(big.Rat).Quo(value, big.NewRat(4, 1))
			scaled := new(big.Rat).Mul(first, big.NewRat(10000, 1))
			if !scaled.IsInt() {
				return "", "", false
			}
			rest := new(big.Rat).Sub(value, first)
			return strings.TrimRight(strings.TrimRight(first.FloatString(4), "0"), "."), strings.TrimRight(strings.TrimRight(rest.FloatString(4), "0"), "."), true
		}
		costA, costB, exactCost := halves(target.Cost)
		quantityA, quantityB, exactQuantity := halves(target.Quantity)
		if !exactCost || !exactQuantity {
			continue
		}
		var split []Line
		for _, line := range made.lines {
			if line.ID != target.ID {
				split = append(split, line)
				continue
			}
			first, rest := line, line
			first.ID, first.Cost, first.Quantity = line.ID+"-1", costA, quantityA
			rest.ID, rest.Cost, rest.Quantity = line.ID+"-2", costB, quantityB
			split = append(split, first, rest)
		}
		cut, err := compiled.Rate(split)
		if err != nil || cut.Total != got.Total || len(cut.Traces) != len(got.Traces) {
			t.Fatalf("seed %d: a line cut in two changes the total from %s to %s (%v)", seed, got.Total, cut.Total, err)
		}
		for index := range cut.Traces {
			if !reflect.DeepEqual(cut.Traces[index].Steps, got.Traces[index].Steps) && cut.Traces[index].Steps[0].Name != StepHeld {
				t.Fatalf("seed %d rule %s: a line cut in two changes the steps", seed, cut.Traces[index].Rule)
			}
		}
		reached["split"]++
		raised := append([]Line(nil), made.lines...)
		for index := range raised {
			if raised[index].ID == target.ID {
				if ratOf(target.Cost).Sign() >= 0 {
					raised[index].Cost = strings.TrimRight(strings.TrimRight(new(big.Rat).Add(ratOf(target.Cost), big.NewRat(7, 2)).FloatString(4), "0"), ".")
				}
				raised[index].Quantity = strings.TrimRight(strings.TrimRight(new(big.Rat).Add(ratOf(target.Quantity), big.NewRat(7, 2)).FloatString(4), "0"), ".")
			}
		}
		more := rateModel(made.policy, raised)
		for id, rule := range want.rules {
			var kind Type
			for _, each := range made.policy.Rules {
				if each.ID == id {
					kind = each.Type
				}
			}
			if after := more.rules[id]; kind != Volume && !rule.held && after != nil && !after.held && after.amounts["base"].Cmp(rule.amounts["base"]) < 0 {
				t.Fatalf("seed %d rule %s: raising an input lowered the base", seed, id)
			}
		}
		higher, err := compiled.Rate(raised)
		if err != nil || ratOf(higher.Total).Cmp(more.total) != 0 {
			t.Fatalf("seed %d: after raising an input the total is %s, the model %s (%v)", seed, higher.Total, more.total.FloatString(4), err)
		}
		ratedOf := func(result Result) map[string]*big.Rat {
			out := map[string]*big.Rat{}
			for _, charge := range result.Charges {
				if charge.Kind == Rated {
					if out[charge.Rule] == nil {
						out[charge.Rule] = new(big.Rat)
					}
					out[charge.Rule].Add(out[charge.Rule], ratOf(charge.Amount))
				}
			}
			return out
		}
		lower, upper := ratedOf(got), ratedOf(higher)
		for _, each := range made.policy.Rules {
			if each.Type != Volume && lower[each.ID] != nil && upper[each.ID] != nil && upper[each.ID].Cmp(lower[each.ID]) < 0 {
				t.Fatalf("seed %d rule %s: raising an input lowered what the engine rated", seed, each.ID)
			}
		}
		reached["raised"]++
	}
	least := map[string]int{"held": 100, "split": 500, "raised": 500}
	for _, key := range []string{"kind/" + Rated, "kind/" + PassedAtCost, "kind/" + FlatCharge, "kind/" + DiscountCharge, "kind/" + MinimumTopUp, "kind/" + MaximumCap,
		"reason/" + ClassNotEligible, "reason/" + NoRule, "reason/" + CostNotKnown, "reason/" + QuantityNotKnown, "reason/" + RuleHeld,
		"type/pass_through", "type/markup", "type/target_margin", "type/unit_price", "type/graduated", "type/volume", "type/flat"} {
		least[key] = 100
	}
	keys := make([]string, 0, len(least))
	for key := range least {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if reached[key] < least[key] {
			t.Errorf("%s reached %d times, want at least %d", key, reached[key], least[key])
		}
	}
}
