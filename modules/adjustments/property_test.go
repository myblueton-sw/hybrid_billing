package adjustments

import (
	"math/big"
	"math/rand"
	"strconv"
	"testing"
)

var modelModes = []string{"toward_zero", "away_from_zero", "floor", "ceiling", "half_up", "half_even"}

func modelRound(value *big.Rat, scale int, mode string) *big.Rat {
	factor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil)
	scaled := new(big.Rat).Mul(value, new(big.Rat).SetInt(factor))
	num, den := scaled.Num(), scaled.Denom()
	low := new(big.Int).Div(num, den)
	if new(big.Int).Mul(low, den).Cmp(num) > 0 {
		low.Sub(low, big.NewInt(1))
	}
	exactly := new(big.Int).Mul(low, den).Cmp(num) == 0
	high := new(big.Int).Add(low, big.NewInt(1))
	if exactly {
		high = low
	}
	frac := new(big.Rat).Sub(scaled, new(big.Rat).SetInt(low))
	half := big.NewRat(1, 2)
	pick := low
	switch mode {
	case "floor":
	case "ceiling":
		pick = high
	case "toward_zero":
		if scaled.Sign() < 0 {
			pick = high
		}
	case "away_from_zero":
		if scaled.Sign() > 0 {
			pick = high
		}
	case "half_up":
		c := frac.Cmp(half)
		if c > 0 || c == 0 && scaled.Sign() > 0 {
			pick = high
		}
	case "half_even":
		c := frac.Cmp(half)
		if c > 0 || c == 0 && low.Bit(0) == 1 {
			pick = high
		}
	}
	return new(big.Rat).SetFrac(pick, factor)
}

type modelEntry struct {
	id, kind, line string
	value          *big.Rat
	active         bool
}

type modelLine struct {
	id                      string
	quantity, price, amount *big.Rat
	priced                  bool
}

type modelResult struct {
	amounts       []*big.Rat
	currents      []*big.Rat
	targets       []*big.Rat
	quantities    []*big.Rat
	lines         map[string]*big.Rat
	total, adjust *big.Rat
	gross         *big.Rat
}

func modelCompute(lines []modelLine, list []*modelEntry, scale int, mode string) modelResult {
	out := modelResult{lines: map[string]*big.Rat{}, total: new(big.Rat), adjust: new(big.Rat), gross: new(big.Rat)}
	qty := map[string]*big.Rat{}
	byID := map[string]modelLine{}
	for _, l := range lines {
		qty[l.id] = new(big.Rat).Set(l.quantity)
		out.lines[l.id] = new(big.Rat).Set(l.amount)
		out.total.Add(out.total, l.amount)
		byID[l.id] = l
	}
	for _, e := range list {
		if !e.active {
			continue
		}
		var change, current, target, quantity *big.Rat
		switch e.kind {
		case EntryQuantity:
			p := byID[e.line].price
			change = new(big.Rat).Sub(modelRound(new(big.Rat).Mul(e.value, p), scale, mode), modelRound(new(big.Rat).Mul(qty[e.line], p), scale, mode))
			current, target, quantity = qty[e.line], e.value, new(big.Rat).Sub(e.value, qty[e.line])
			qty[e.line] = e.value
		default:
			current = out.total
			if e.line != "" {
				current = out.lines[e.line]
			}
			if e.kind == EntryFixed {
				change, target = new(big.Rat).Set(e.value), new(big.Rat).Add(current, e.value)
			} else {
				change, target = new(big.Rat).Sub(e.value, current), e.value
			}
		}
		out.currents, out.targets, out.quantities = append(out.currents, current), append(out.targets, target), append(out.quantities, quantity)
		if e.line != "" {
			out.lines[e.line] = new(big.Rat).Add(out.lines[e.line], change)
		}
		out.total = new(big.Rat).Add(out.total, change)
		out.adjust.Add(out.adjust, change)
		out.gross.Add(out.gross, new(big.Rat).Abs(change))
		out.amounts = append(out.amounts, change)
	}
	return out
}

func rat(text string) *big.Rat {
	v, ok := new(big.Rat).SetString(text)
	if !ok {
		panic(text)
	}
	return v
}

func amount(rng *rand.Rand, low, high int64, scale int) string {
	factor := int64(1)
	for range scale {
		factor *= 10
	}
	n := low*factor + rng.Int63n((high-low)*factor+1)
	return new(big.Rat).SetFrac64(n, factor).FloatString(scale)
}

func equalText(text string, want *big.Rat) bool { return rat(text).Cmp(want) == 0 }

func TestPropertyAgainstModel(t *testing.T) {
	rng := rand.New(rand.NewSource(222))
	for round := range 400 {
		r, _ := NewRegistry(testProfile(t), Limits{Workspaces: 1, Drafts: 1, Lines: 4, Entries: 40, Evidence: 1}, (&testClock{now: start}).read)
		mode := modelModes[rng.Intn(len(modelModes))]
		scale := rng.Intn(9)
		d := Draft{ID: "d", Customer: "c", Contract: "k", From: "2026-10-01", To: "2026-10-31", Base: Base{Run: "run", Digest: digest('a'), Currency: "USD", Scale: scale, Mode: mode}}
		var lines []modelLine
		for index := range 1 + rng.Intn(3) {
			s := SourceLine{ID: "l" + strconv.Itoa(index), Charge: "ch", Quantity: strconv.Itoa(rng.Intn(20)) + "." + strconv.Itoa(rng.Intn(10)), Amount: amount(rng, -20, 200, scale)}
			m := modelLine{id: s.ID, quantity: rat(s.Quantity), amount: rat(s.Amount)}
			if rng.Intn(4) > 0 {
				s.Price = strconv.Itoa(rng.Intn(5)) + "." + strconv.Itoa(rng.Intn(1000))
				m.price, m.priced = rat(s.Price), true
			}
			d.Lines = append(d.Lines, s)
			lines = append(lines, m)
		}
		v, err := r.Open("ws", d, human("w"))
		if err != nil {
			t.Fatalf("round %d open: %v", round, err)
		}
		var model []*modelEntry
		for step := range 30 {
			if rng.Intn(5) == 0 {
				var active []*modelEntry
				for _, e := range model {
					if e.active {
						active = append(active, e)
					}
				}
				if len(active) == 0 {
					continue
				}
				pick := active[rng.Intn(len(active))]
				v = must(r.Remove("ws", "d", v.Revision, pick.id, human("w")))
				pick.active = false
			} else {
				l := lines[rng.Intn(len(lines))]
				e := Entry{ID: "e" + strconv.Itoa(step), Reason: ReasonOther, Evidence: []string{"x"}, From: "2026-10-01", To: "2026-10-31"}
				m := &modelEntry{id: e.ID, active: true}
				switch rng.Intn(3) {
				case 0:
					if !l.priced {
						continue
					}
					e.Kind, e.Line, e.Target = EntryQuantity, l.id, strconv.Itoa(rng.Intn(25))+"."+strconv.Itoa(rng.Intn(100))
					m.value = rat(e.Target)
				case 1:
					e.Kind, e.Amount = EntryFixed, amount(rng, -30, 30, scale)
					if rng.Intn(2) == 0 {
						e.Line = l.id
					}
					m.value = rat(e.Amount)
					if m.value.Sign() == 0 {
						continue
					}
				default:
					e.Kind, e.Target = EntryTargetTotal, amount(rng, -10, 300, scale)
					if rng.Intn(2) == 0 {
						e.Line = l.id
					}
					m.value = rat(e.Target)
				}
				m.kind, m.line = e.Kind, e.Line
				trial := modelCompute(lines, append(append([]*modelEntry(nil), model...), m), scale, mode)
				want := "ok"
				switch {
				case trial.amounts[len(trial.amounts)-1].Sign() == 0:
					want = RefuseZeroDifference
				case trial.total.Sign() < 0:
					want = RefuseNegativeTotal
				}
				next, err := r.Add("ws", "d", v.Revision, e, human("w"))
				if kind(err) != want {
					t.Fatalf("round %d step %d: want %s got %s", round, step, want, kind(err))
				}
				if err != nil {
					continue
				}
				v = next
				model = append(model, m)
			}
			res := modelCompute(lines, model, scale, mode)
			if len(v.Differences) != len(res.amounts) {
				t.Fatalf("round %d step %d: %d differences, model %d", round, step, len(v.Differences), len(res.amounts))
			}
			for index, diff := range v.Differences {
				explained := diff.Kind == EntryQuantity || new(big.Rat).Sub(rat(diff.Target), rat(diff.Current)).Cmp(rat(diff.Amount)) == 0
				quantity := diff.Quantity == "" && res.quantities[index] == nil || res.quantities[index] != nil && diff.Quantity != "" && equalText(diff.Quantity, res.quantities[index])
				if !equalText(diff.Amount, res.amounts[index]) || !explained || !quantity || !equalText(diff.Current, res.currents[index]) || !equalText(diff.Target, res.targets[index]) {
					t.Fatalf("round %d step %d diff %d: %+v model %s", round, step, index, diff, res.amounts[index].FloatString(4))
				}
			}
			if !equalText(v.Total, res.total) || !equalText(v.Adjustment, res.adjust) || !equalText(v.Gross, res.gross) {
				t.Fatalf("round %d step %d totals %s %s %s model %s %s %s", round, step, v.Total, v.Adjustment, v.Gross, res.total.FloatString(4), res.adjust.FloatString(4), res.gross.FloatString(4))
			}
			for index, l := range v.Lines {
				if l.Line != d.Lines[index] || !equalText(l.Adjusted, res.lines[l.Line.ID]) {
					t.Fatalf("round %d step %d line %+v model %s", round, step, l, res.lines[l.Line.ID].FloatString(4))
				}
			}
		}
	}
}

func TestPropertyQuantityPathIndependence(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	for round := range 300 {
		mode := modelModes[rng.Intn(len(modelModes))]
		price := strconv.Itoa(rng.Intn(3)) + "." + strconv.Itoa(rng.Intn(1000))
		final := strconv.Itoa(rng.Intn(30)) + "." + strconv.Itoa(rng.Intn(10))
		totals := map[string]bool{}
		for path := range 3 {
			r, _ := NewRegistry(testProfile(t), Limits{Workspaces: 1, Drafts: 1, Lines: 1, Entries: 10, Evidence: 1}, (&testClock{now: start}).read)
			d := Draft{ID: "d", Customer: "c", Contract: "k", From: "2026-10-01", To: "2026-10-31", Base: Base{Run: "run", Digest: digest('a'), Currency: "USD", Scale: 2, Mode: mode},
				Lines: []SourceLine{{ID: "l", Charge: "ch", Quantity: "3", Price: price, Amount: "100"}}}
			v := must(r.Open("ws", d, human("w")))
			steps := []string{}
			for range path * 2 {
				steps = append(steps, strconv.Itoa(rng.Intn(30))+"."+strconv.Itoa(rng.Intn(10)))
			}
			steps = append(steps, final)
			for index, target := range steps {
				next, err := r.Add("ws", "d", v.Revision, Entry{ID: "e" + strconv.Itoa(index), Kind: EntryQuantity, Line: "l", Target: target, Reason: ReasonOther, Evidence: []string{"x"}, From: "2026-10-01", To: "2026-10-31"}, human("w"))
				switch kind(err) {
				case "ok":
					v = next
				case RefuseZeroDifference:
				default:
					t.Fatalf("round %d path %d: %v", round, path, err)
				}
			}
			totals[v.Lines[0].Adjusted] = true
			want := new(big.Rat).Add(rat("100"), new(big.Rat).Sub(modelRound(new(big.Rat).Mul(rat(final), rat(price)), 2, mode), modelRound(new(big.Rat).Mul(rat("3"), rat(price)), 2, mode)))
			if !equalText(v.Lines[0].Adjusted, want) {
				t.Fatalf("round %d path %d: %s want %s", round, path, v.Lines[0].Adjusted, want.FloatString(2))
			}
		}
		if len(totals) != 1 {
			t.Fatalf("round %d: path dependent %v", round, totals)
		}
	}
}
