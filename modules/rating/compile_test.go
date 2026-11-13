package rating

import (
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/allocation"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func TestRefusalsOfAPolicy(t *testing.T) {
	profile := profileOf(t, 12, 4, 14)
	per := Price{Amount: "10", Per: 1}
	tiers := []Tier{{UpTo: "100", Price: per}, {UpTo: "200", Price: per}, {Price: per}}
	good := policy(
		Rule{ID: "r-sku", Level: SKU, Key: "sku-1", Type: UnitPrice, Price: per, Included: "5", Discounts: []Discount{{Percent: "0.1"}, {Fixed: "10"}}, Minimum: "10", Maximum: "20"},
		Rule{ID: "r-tier", Level: Service, Key: "svc-1", Type: Graduated, Tiers: tiers},
		Rule{ID: "r-flat", Level: Group, Key: "grp-1", Type: Flat, Amount: "100"},
		byDefault(Markup, func(r *Rule) { r.Rate = "0.2" }),
	)
	if _, err := Compile(profile, testLimits(), good); err != nil {
		t.Fatal(err)
	}
	rule := func(index int, apply func(*Rule)) func(*Policy) {
		return func(p *Policy) {
			p.Rules = append([]Rule(nil), p.Rules...)
			apply(&p.Rules[index])
		}
	}
	malformed := map[string]func(*Policy){
		"identifier": func(p *Policy) { p.ID = "" }, "version": func(p *Policy) { p.Version = "v 1" }, "currency": func(p *Policy) { p.Currency = "krw" }, "a cost basis": func(p *Policy) { p.CostBasis = "net" }, "no cost basis": func(p *Policy) { p.CostBasis = "" },
		"a quantum of two": func(p *Policy) { p.Quantum = "2" }, "a quantum written otherwise": func(p *Policy) { p.Quantum = "1.0" }, "no quantum": func(p *Policy) { p.Quantum = "" }, "a mode": func(p *Policy) { p.Mode = "bankers" }, "no mode": func(p *Policy) { p.Mode = "" },
		"a path": func(p *Policy) { p.Path = "reseller" }, "no path": func(p *Policy) { p.Path = "" }, "a direct policy with classes": func(p *Policy) { p.Path = Direct }, "an unknown class": func(p *Policy) { p.Eligible = []Class{"usage"} },
		"a class twice": func(p *Policy) { p.Eligible = []Class{Other, Licence, Other} }, "no rule": func(p *Policy) { p.Rules = nil },
		"a rule without an identifier": rule(0, func(r *Rule) { r.ID = "" }), "a rule twice": rule(1, func(r *Rule) { r.ID = "r-sku" }), "a level": rule(0, func(r *Rule) { r.Level = "contract" }), "no level": rule(0, func(r *Rule) { r.Level = "" }), "a level without a key": rule(0, func(r *Rule) { r.Level, r.Key = "contract", "" }),
		"a keyed level without a key": rule(0, func(r *Rule) { r.Key = "" }), "a malformed key": rule(0, func(r *Rule) { r.Key = "sku 1" }), "a default rule with a key": rule(3, func(r *Rule) { r.Key = "x" }),
		"two rules of one level and key": rule(1, func(r *Rule) { r.Level, r.Key = SKU, "sku-1" }), "two default rules": rule(2, func(r *Rule) { r.Level, r.Key, r.Type, r.Amount, r.Rate = Default, "", Markup, "", "0.1" }),
		"a type": rule(3, func(r *Rule) { r.Type = "cost_plus" }), "no type": rule(3, func(r *Rule) { r.Type, r.Rate = "", "" }), "a markup without a rate": rule(3, func(r *Rule) { r.Rate = "" }), "a negative rate": rule(3, func(r *Rule) { r.Rate = "-0.1" }),
		"a rate that is no number": rule(3, func(r *Rule) { r.Rate = "20%" }), "a rate on another type": rule(0, func(r *Rule) { r.Rate = "0.1" }), "a margin on another type": rule(0, func(r *Rule) { r.Margin = "0.1" }),
		"a margin of one": rule(3, func(r *Rule) { r.Type, r.Rate, r.Margin = TargetMargin, "", "1" }), "a margin above one": rule(3, func(r *Rule) { r.Type, r.Rate, r.Margin = TargetMargin, "", "1.5" }),
		"a negative margin": rule(3, func(r *Rule) { r.Type, r.Rate, r.Margin = TargetMargin, "", "-0.1" }), "a target margin without a margin": rule(3, func(r *Rule) { r.Type, r.Rate = TargetMargin, "" }),
		"a unit price without a price": rule(0, func(r *Rule) { r.Price = Price{} }), "a negative price": rule(0, func(r *Rule) { r.Price.Amount = "-1" }), "a denominator of ten": rule(0, func(r *Rule) { r.Price.Per = 10 }),
		"a denominator of zero": rule(0, func(r *Rule) { r.Price.Per = 0 }), "a price on another type": rule(3, func(r *Rule) { r.Price = per }), "tiers on another type": rule(0, func(r *Rule) { r.Tiers = tiers }),
		"a tiered rule without tiers": rule(1, func(r *Rule) { r.Tiers = nil }), "a last tier with a bound": rule(1, func(r *Rule) { r.Tiers = tiers[:2] }), "a tier without a bound before the last": rule(1, func(r *Rule) { r.Tiers = []Tier{{Price: per}, {Price: per}} }),
		"bounds that do not rise": rule(1, func(r *Rule) { r.Tiers = []Tier{{UpTo: "100", Price: per}, {UpTo: "100", Price: per}, {Price: per}} }), "a bound of zero": rule(1, func(r *Rule) { r.Tiers = []Tier{{UpTo: "0", Price: per}, {Price: per}} }),
		"a negative bound": rule(1, func(r *Rule) { r.Tiers = []Tier{{UpTo: "-5", Price: per}, {Price: per}} }), "a tier with a malformed price": rule(1, func(r *Rule) { r.Tiers = []Tier{{UpTo: "5", Price: Price{Amount: "x", Per: 1}}, {Price: per}} }),
		"an included quantity on a cost-based type": rule(3, func(r *Rule) { r.Included = "5" }), "an included quantity on a flat fee": rule(2, func(r *Rule) { r.Included = "5" }), "a negative included quantity": rule(0, func(r *Rule) { r.Included = "-5" }),
		"a flat fee without an amount": rule(2, func(r *Rule) { r.Amount = "" }), "an amount on another type": rule(3, func(r *Rule) { r.Amount = "5" }), "a flat amount that is no multiple of the quantum": rule(2, func(r *Rule) { r.Amount = "100.5" }),
		"a negative flat amount": rule(2, func(r *Rule) { r.Amount = "-100" }), "a flat fee at the default level": rule(3, func(r *Rule) { r.Type, r.Rate, r.Amount = Flat, "", "100" }),
		"a discount of both kinds": rule(0, func(r *Rule) { r.Discounts = []Discount{{Percent: "0.1", Fixed: "5"}} }), "a discount of neither kind": rule(0, func(r *Rule) { r.Discounts = []Discount{{}} }),
		"a percentage of one": rule(0, func(r *Rule) { r.Discounts = []Discount{{Percent: "1"}} }), "a negative percentage": rule(0, func(r *Rule) { r.Discounts = []Discount{{Percent: "-0.1"}} }),
		"a fixed discount that is no multiple": rule(0, func(r *Rule) { r.Discounts = []Discount{{Fixed: "0.5"}} }), "a negative fixed discount": rule(0, func(r *Rule) { r.Discounts = []Discount{{Fixed: "-5"}} }),
		"a minimum that is no multiple": rule(0, func(r *Rule) { r.Minimum = "10.5" }), "a negative minimum": rule(0, func(r *Rule) { r.Minimum = "-10" }), "a maximum that is no multiple": rule(0, func(r *Rule) { r.Maximum = "20.25" }),
		"a negative maximum": rule(0, func(r *Rule) { r.Maximum = "-20" }), "a minimum above the maximum": rule(0, func(r *Rule) { r.Minimum = "21" }),
	}
	for name, apply := range malformed {
		_, err := Compile(profile, testLimits(), alter(good, apply))
		wantErr(t, err, ErrInvalid, name)
	}
	for name, apply := range map[string]func(*Policy){
		"more rules than the limit": func(p *Policy) {
			for index := 0; index < 5; index++ {
				p.Rules = append(p.Rules, Rule{ID: fmt.Sprintf("x%d", index), Level: SKU, Key: fmt.Sprintf("k%d", index), Type: PassThrough})
			}
		},
		"more tiers than the limit": rule(1, func(r *Rule) {
			r.Tiers = []Tier{{UpTo: "1", Price: per}, {UpTo: "2", Price: per}, {UpTo: "3", Price: per}, {UpTo: "4", Price: per}, {Price: per}}
		}),
		"more discounts than the limit": rule(0, func(r *Rule) { r.Discounts = []Discount{{Fixed: "1"}, {Fixed: "1"}, {Fixed: "1"}, {Fixed: "1"}} }),
	} {
		_, err := Compile(profile, testLimits(), alter(good, apply))
		wantErr(t, err, ErrLimit, name)
	}
	edge := alter(good, func(p *Policy) {
		p.Rules = append([]Rule(nil), p.Rules...)
		p.Rules[1].Tiers = []Tier{{UpTo: "1", Price: per}, {UpTo: "2", Price: per}, {UpTo: "3", Price: per}, {Price: per}}
		p.Rules[0].Discounts = []Discount{{Fixed: "1"}, {Fixed: "1"}, {Percent: "0"}}
		p.Rules[0].Minimum, p.Rules[0].Maximum = "20", "20"
		for index := 0; index < 4; index++ {
			p.Rules = append(p.Rules, Rule{ID: fmt.Sprintf("x%d", index), Level: SKU, Key: fmt.Sprintf("k%d", index), Type: PassThrough})
		}
	})
	if _, err := Compile(profile, testLimits(), edge); err != nil {
		t.Fatalf("a policy at its limits: %v", err)
	}
	for name, limits := range map[string]Limits{"rules": {Tiers: 1, Discounts: 1, Lines: 1, Intermediate: 1}, "tiers": {Rules: 1, Discounts: 1, Lines: 1, Intermediate: 1}, "discounts": {Rules: 1, Tiers: 1, Lines: 1, Intermediate: 1},
		"lines": {Rules: 1, Tiers: 1, Discounts: 1, Intermediate: 1}, "intermediate": {Rules: 1, Tiers: 1, Discounts: 1, Lines: 1}} {
		_, err := Compile(profile, limits, good)
		wantErr(t, err, ErrInvalid, "limits without "+name)
	}
	_, err := Compile(decimal.Profile{}, testLimits(), good)
	wantErr(t, err, ErrInvalid, "no profile")
	coarse, err := decimal.NewProfile(decimal.ProfileSpec{ID: "coarse", Revision: "1", InputIntegerDigits: 12, InputScale: 2, ResultIntegerDigits: 14, ResultScale: 2, IntermediateDigits: 40})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Compile(coarse, testLimits(), alter(good, func(p *Policy) { p.Quantum = "0.001" }))
	wantErr(t, err, ErrInvalid, "a quantum the profile cannot read")
	if _, err := Compile(coarse, testLimits(), alter(good, func(p *Policy) { p.Quantum = "0.01" })); err != nil {
		t.Fatal(err)
	}
	var missing *Compiled
	_, err = missing.Rate(nil)
	wantErr(t, err, ErrInvalid, "no compiled policy")
	_, err = (&Compiled{}).Rate(nil)
	wantErr(t, err, ErrInvalid, "a compiled policy that was not compiled")
	for _, kind := range []error{ErrInvalid, ErrLimit, ErrSelfCheck} {
		if strings.ContainsAny(kind.Error(), "%0123456789") {
			t.Fatalf("%q", kind)
		}
	}
	changed := alter(good, func(p *Policy) {
		p.Rules = append([]Rule(nil), p.Rules...)
		p.Eligible = append([]Class(nil), p.Eligible...)
	})
	changed.Rules[1].Tiers = append([]Tier(nil), tiers...)
	compiled, err := Compile(profile, testLimits(), changed)
	if err != nil {
		t.Fatal(err)
	}
	lines := []Line{{ID: "a", Class: Other, Service: "svc-1", Quantity: "150", Currency: "KRW"}, {ID: "b", Class: Licence, SKU: "sku-1", Quantity: "9", Currency: "KRW"}}
	before, err := compiled.Rate(lines)
	if err != nil {
		t.Fatal(err)
	}
	changed.Rules[1].Tiers[0].Price.Amount, changed.Rules[0].Minimum, changed.Eligible[2], changed.Rules[3].Rate, changed.Version = "999", "0", CSPUsage, "9", "v9"
	changed.Rules[0].Discounts[0].Percent = "0.9"
	after, err := compiled.Rate(lines)
	if err != nil || !reflect.DeepEqual(before, after) || before.Total != "1520" {
		t.Fatalf("a policy changed by the caller after it was compiled changes nothing: %s %s %v", before.Total, after.Total, err)
	}
}

func TestRefusalsOfLinesLimitsAndBounds(t *testing.T) {
	profile := profileOf(t, 12, 4, 14)
	compiled, err := Compile(profile, alter(testLimits(), func(l *Limits) { l.Lines = 3 }), policy(byDefault(PassThrough, nil)))
	if err != nil {
		t.Fatal(err)
	}
	good := Line{ID: "a", Class: Other, SKU: "s", Service: "v", Group: "g", Cost: "1", Quantity: "1", Currency: "KRW"}
	for name, apply := range map[string]func(*Line){
		"identifier": func(l *Line) { l.ID = "" }, "a malformed identifier": func(l *Line) { l.ID = "a b" }, "a class": func(l *Line) { l.Class = "usage" }, "no class": func(l *Line) { l.Class = "" }, "a malformed SKU": func(l *Line) { l.SKU = "s/1" },
		"a malformed service": func(l *Line) { l.Service = "v 1" }, "a malformed group": func(l *Line) { l.Group = "g@1" }, "another currency": func(l *Line) { l.Currency = "USD" }, "no currency": func(l *Line) { l.Currency = "" },
		"a cost that is no number": func(l *Line) { l.Cost = "free" }, "a cost finer than the profile reads": func(l *Line) { l.Cost = "0.00001" }, "a cost beyond the profile": func(l *Line) { l.Cost = "1234567890123" },
		"a quantity that is no number": func(l *Line) { l.Quantity = "1e3" }, "a negative quantity": func(l *Line) { l.Quantity = "-1" }, "a quantity of negative zero": func(l *Line) { l.Quantity = "-0" },
	} {
		_, err := compiled.Rate([]Line{alter(good, apply)})
		wantErr(t, err, ErrInvalid, name)
		_, err = compiled.Rate([]Line{good, alter(alter(good, apply), func(l *Line) {
			if l.ID == "a" {
				l.ID = "b"
			}
		})})
		wantErr(t, err, ErrInvalid, name+" after a good line")
	}
	_, err = compiled.Rate([]Line{good, alter(good, func(l *Line) { l.Cost = "2" })})
	wantErr(t, err, ErrInvalid, "an identifier twice")
	four := []Line{good, alter(good, func(l *Line) { l.ID = "b" }), alter(good, func(l *Line) { l.ID = "c" }), alter(good, func(l *Line) { l.ID = "" })}
	_, err = compiled.Rate(four)
	wantErr(t, err, ErrLimit, "more lines than the limit, whatever they hold")
	if out, err := compiled.Rate(four[:3]); err != nil || out.Total != "3" || out.Lines != 3 {
		t.Fatal(out.Total, err)
	}
	if out, err := compiled.Rate([]Line{alter(good, func(l *Line) { l.SKU, l.Service, l.Group, l.Quantity, l.Cost = "", "", "", "", "-0" })}); err != nil || out.Total != "0" || charges(out) != "default/rated/a=0" {
		t.Fatalf("a cost of negative zero is zero: %v", err)
	}
	narrow := profileOf(t, 3, 4, 3)
	bounded, err := Compile(narrow, testLimits(), policy(byDefault(Markup, func(r *Rule) { r.Rate = "0.5" })))
	if err != nil {
		t.Fatal(err)
	}
	_, err = bounded.Rate([]Line{cost("a", "999")})
	wantErr(t, err, ErrLimit, "an amount the profile cannot hold")
	weighed, err := Compile(narrow, testLimits(), policy(byDefault(UnitPrice, func(r *Rule) { r.Price = Price{Amount: "1", Per: 1000} })))
	if err != nil {
		t.Fatal(err)
	}
	_, err = weighed.Rate([]Line{used("a", "600"), used("b", "600")})
	wantErr(t, err, ErrLimit, "weights whose sum the profile cannot hold, beside an amount it can")
	if out, err := weighed.Rate([]Line{used("a", "600"), used("b", "399")}); err != nil || out.Total != "1" {
		t.Fatal(out.Total, err)
	}
	if out, err := bounded.Rate([]Line{cost("a", "600")}); err != nil || out.Total != "900" {
		t.Fatal(out.Total, err)
	}
	two, err := Compile(narrow, testLimits(), policy(Rule{ID: "s", Level: SKU, Key: "k", Type: PassThrough}, byDefault(PassThrough, nil)))
	if err != nil {
		t.Fatal(err)
	}
	_, err = two.Rate([]Line{cost("a", "600"), alter(cost("b", "600"), func(l *Line) { l.SKU = "k" })})
	wantErr(t, err, ErrLimit, "a grand total the profile cannot hold")
	tight, err := Compile(profile, alter(testLimits(), func(l *Limits) { l.Intermediate = 6 }), policy(byDefault(PassThrough, nil)))
	if err != nil {
		t.Fatal(err)
	}
	_, err = tight.Rate([]Line{cost("a", "700000"), cost("b", "500000")})
	wantErr(t, err, ErrLimit, "the allocation's bound of intermediates")
	one := decimal.Decimal{}
	if one, err = decimal.Parse(profile, "1"); err != nil {
		t.Fatal(err)
	}
	for id, want := range map[string]bool{"a": true, "A.z_0:9-": true, strings.Repeat("a", 128): true, strings.Repeat("a", 129): false, "a b": false, "a/b": false, "kim@example": false} {
		if validID(id) != want {
			t.Errorf("identifier %q", id)
		}
		_, err := allocation.LargestRemainderV1(allocation.Request{Profile: profile, Limits: allocation.Limits{MaxTargets: 2, MaxIntermediateDigits: 40}, Encoding: allocation.EncodingASCIIBytewiseV1, Total: one, Quantum: one, Mode: decimal.HalfEven,
			Targets: []allocation.Target{{ID: id, Weight: one}}})
		if (err == nil) != want {
			t.Errorf("the allocation and this package differ on %q: %v", id, err)
		}
	}
	if validID("") {
		t.Error("an empty identifier")
	}
	for code, want := range map[string]bool{"KRW": true, "krw": false, "KR": false, "KRWW": false, "1RW": false, "K1W": false, "KR1": false, "kRW": false, "KrW": false, "KRw": false} {
		if validCurrency(code) != want {
			t.Errorf("currency %q", code)
		}
	}
}

func TestRoundingAgreesWithThePrimitive(t *testing.T) {
	profile, err := decimal.NewProfile(decimal.ProfileSpec{ID: "wide", Revision: "1", InputIntegerDigits: 9, InputScale: 4, ResultIntegerDigits: 18, ResultScale: 6, IntermediateDigits: 60})
	if err != nil {
		t.Fatal(err)
	}
	random := rand.New(rand.NewSource(204))
	ties := 0
	for round := 0; round < 20000; round++ {
		numerator := fmt.Sprintf("%d.%d", random.Intn(2000)-1000, random.Intn(4)*25)
		denominator := fmt.Sprintf("%d.%d", 1+random.Intn(40), random.Intn(2)*5)
		if random.Intn(4) == 0 {
			denominator = []string{"2", "4", "8", "0.5"}[random.Intn(4)]
		}
		top, _ := decimal.Parse(profile, numerator)
		bottom, _ := decimal.Parse(profile, denominator)
		value := of(top).over(of(bottom))
		for scale := 0; scale <= 4; scale++ {
			for _, name := range []string{"toward_zero", "away_from_zero", "floor", "ceiling", "half_up", "half_even"} {
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
			twice := new(big.Int).Mul(new(big.Int).Mul(value.num, power(scale)), big.NewInt(2))
			if new(big.Int).Rem(twice, value.den).Sign() == 0 && new(big.Int).Rem(new(big.Int).Mul(value.num, power(scale)), value.den).Sign() != 0 {
				ties++
			}
		}
	}
	if ties < 2000 {
		t.Fatalf("only %d ties", ties)
	}
	for text, want := range map[string]string{"10/4": "5/2", "-10/4": "-5/2", "0/7": "0", "8/4": "2", "7/1": "7", "-3/9": "-1/3"} {
		parts := strings.Split(text, "/")
		num, _ := new(big.Int).SetString(parts[0], 10)
		den, _ := new(big.Int).SetString(parts[1], 10)
		if got := (fraction{num: num, den: den}).text(); got != want {
			t.Errorf("%s written as %s", text, got)
		}
	}
}

func TestOrderConcurrencyAndTheSource(t *testing.T) {
	p := policy(
		Rule{ID: "b-rule", Level: SKU, Key: "k", Type: Markup, Rate: "0.1", Discounts: []Discount{{Fixed: "1"}, {Percent: "0.1"}}, Minimum: "500", Maximum: "600"},
		Rule{ID: "a-rule", Level: Group, Key: "g", Type: Flat, Amount: "70"},
		byDefault(UnitPrice, func(r *Rule) { r.Price = Price{Amount: "3", Per: 1} }),
	)
	compiled, err := Compile(profileOf(t, 12, 4, 14), alter(testLimits(), func(l *Limits) { l.Lines = 400 }), p)
	if err != nil {
		t.Fatal(err)
	}
	lines := []Line{
		{ID: "m", Class: Other, SKU: "k", Cost: "100", Currency: "KRW"}, {ID: "c", Class: Other, SKU: "k", Cost: "-7.5", Currency: "KRW"}, {ID: "x", Class: Other, SKU: "k", Cost: "300", Currency: "KRW"},
		{ID: "b", Class: Other, SKU: "k", Cost: "-2.5", Currency: "KRW"}, {ID: "g2", Class: Other, Group: "g", Currency: "KRW"}, {ID: "g1", Class: Other, Group: "g", Cost: "5", Currency: "KRW"},
		{ID: "u2", Class: Other, Quantity: "2", Currency: "KRW"}, {ID: "u1", Class: Other, Quantity: "1", Currency: "KRW"},
	}
	want, err := compiled.Rate(lines)
	if err != nil {
		t.Fatal(err)
	}
	expect := "a-rule/flat/-=70 b-rule/rated/m=110 b-rule/rated/x=330 b-rule/passed_at_cost/b=-3 b-rule/passed_at_cost/c=-7 b-rule/discount/-=-1 b-rule/discount/-=-44 b-rule/minimum_top_up/-=105 default/rated/u1=3 default/rated/u2=6"
	if charges(want) != expect || want.Total != "569" || fmt.Sprint(want.Covered) != "[{g1 a-rule} {g2 a-rule}]" {
		t.Fatalf("\n got %s\nwant %s (%s)", charges(want), expect, want.Total)
	}
	random := rand.New(rand.NewSource(7))
	var group sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		shuffled := append([]Line(nil), lines...)
		random.Shuffle(len(shuffled), func(a, b int) { shuffled[a], shuffled[b] = shuffled[b], shuffled[a] })
		group.Add(1)
		go func() {
			defer group.Done()
			for spin := 0; spin < 50; spin++ {
				got, err := compiled.Rate(shuffled)
				if err != nil || !reflect.DeepEqual(got, want) {
					t.Errorf("the lines in another order, by another goroutine: %v", err)
					return
				}
			}
		}()
	}
	group.Wait()
	many := make([]Line, 400)
	for index := range many {
		many[index] = Line{ID: fmt.Sprintf("l%03d", index), Class: Other, Quantity: fmt.Sprintf("%d.5", index%7), Currency: "KRW"}
	}
	pool, err := compiled.Rate(many)
	if err != nil || pool.Total != "4191" || len(pool.Charges) != 400 || pool.Lines != 400 {
		t.Fatalf("a pool at the limit of lines: %s %d %v", pool.Total, len(pool.Charges), err)
	}
	if got := steps(want, "b-rule"); !strings.HasSuffix(got, "discount(440;1)=-1 discount(439;0.1)=-44 minimum(395;500)=105 maximum(500;600)=0 total(;)=490 | 490") {
		t.Fatalf("the maximum is judged on the amount after the minimum: %s", got)
	}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	banned := regexp.MustCompile(`float32|float64|"math"|math\.|ParseFloat|big\.Float|big\.Rat|time\.|rand\.|\.Quo\(p|decimal\.(Mul|Quo|DivideRound|Round|RoundToQuantum|Quanta|Sub)\(`)
	scanned := 0
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		content, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		scanned++
		for index, line := range strings.Split(string(content), "\n") {
			if found := banned.FindString(line); found != "" {
				t.Errorf("%s:%d uses %s", name, index+1, found)
			}
			if strings.Contains(line, "//") || strings.Contains(line, "/*") {
				t.Errorf("%s:%d has a comment", name, index+1)
			}
		}
	}
	if scanned != 4 {
		t.Fatalf("scanned %d source files", scanned)
	}
}
