package rating

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func profileOf(t testing.TB, whole, scale, result int) decimal.Profile {
	t.Helper()
	made, err := decimal.NewProfile(decimal.ProfileSpec{ID: "synthetic", Revision: "7", InputIntegerDigits: whole, InputScale: scale, ResultIntegerDigits: result, ResultScale: scale, IntermediateDigits: 60})
	if err != nil {
		t.Fatal(err)
	}
	return made
}

func testLimits() Limits {
	return Limits{Rules: 8, Tiers: 4, Discounts: 3, Lines: 50, Intermediate: 60}
}

func policy(rules ...Rule) Policy {
	return Policy{ID: "pol-1", Version: "v1", Currency: "KRW", CostBasis: Billed, Quantum: "1", Mode: "half_even", Path: Resale, Eligible: []Class{CSPUsage, ServiceFee, Licence, Other}, Rules: rules}
}

func byDefault(kind Type, apply func(*Rule)) Rule {
	made := Rule{ID: "default", Level: Default, Type: kind}
	if apply != nil {
		apply(&made)
	}
	return made
}

func cost(id, amount string) Line {
	return Line{ID: id, Class: CSPUsage, Cost: amount, Currency: "KRW"}
}

func used(id, quantity string) Line {
	return Line{ID: id, Class: CSPUsage, Quantity: quantity, Currency: "KRW"}
}

func rate(t testing.TB, p Policy, lines ...Line) Result {
	t.Helper()
	compiled, err := Compile(profileOf(t, 12, 4, 14), testLimits(), p)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	out, err := compiled.Rate(lines)
	if err != nil {
		t.Fatalf("rate: %v", err)
	}
	return out
}

func charges(out Result) string {
	var parts []string
	for _, charge := range out.Charges {
		line := charge.Line
		if line == "" {
			line = "-"
		}
		parts = append(parts, fmt.Sprintf("%s/%s/%s=%s", charge.Rule, charge.Kind, line, charge.Amount))
	}
	return strings.Join(parts, " ")
}

func excluded(out Result) string {
	var parts []string
	for _, each := range out.Excluded {
		parts = append(parts, each.Line+":"+each.Reason)
	}
	return strings.Join(parts, " ")
}

func steps(out Result, rule string) string {
	for _, trace := range out.Traces {
		if trace.Rule != rule {
			continue
		}
		var parts []string
		for _, step := range trace.Steps {
			parts = append(parts, fmt.Sprintf("%s(%s;%s)=%s", step.Name, step.Basis, step.Parameter, step.Outcome))
		}
		return strings.Join(parts, " ") + " | " + trace.Total
	}
	return "no trace"
}

func wantErr(t testing.TB, err, want error, what string) {
	t.Helper()
	if !errors.Is(err, want) {
		t.Fatalf("%s: got %v, want %v", what, err, want)
	}
}

func alter[T any](base T, apply func(*T)) T {
	apply(&base)
	return base
}
