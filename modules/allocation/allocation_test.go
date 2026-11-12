package allocation

import (
	"errors"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func synthetic(t testing.TB) decimal.Profile {
	t.Helper()
	p, err := decimal.NewProfile(decimal.ProfileSpec{ID: "synthetic-allocation", Revision: "1", InputIntegerDigits: 12, InputScale: 8, ResultIntegerDigits: 18, ResultScale: 12, IntermediateDigits: 80})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

var limits = Limits{MaxTargets: 64, MaxIntermediateDigits: 60}

func value(t testing.TB, p decimal.Profile, text string) decimal.Decimal {
	t.Helper()
	parsed, err := decimal.Parse(p, text)
	if err != nil {
		t.Fatalf("parse %q: %v", text, err)
	}
	return parsed
}

func targets(t testing.TB, p decimal.Profile, pairs ...string) []Target {
	t.Helper()
	list := make([]Target, 0, len(pairs)/2)
	for index := 0; index < len(pairs); index += 2 {
		list = append(list, Target{ID: pairs[index], Weight: value(t, p, pairs[index+1])})
	}
	return list
}

func request(t testing.TB, p decimal.Profile, total, quantum string, mode decimal.Mode, list []Target) Request {
	t.Helper()
	return Request{Profile: p, Limits: limits, Encoding: EncodingASCIIBytewiseV1, Total: value(t, p, total), Quantum: value(t, p, quantum), Mode: mode, Targets: list}
}

func amounts(result Result) string {
	parts := make([]string, len(result.Shares))
	for index, share := range result.Shares {
		parts[index] = share.ID + "=" + share.Amount.String()
	}
	return strings.Join(parts, " ")
}

func TestContractCalculatorCases(t *testing.T) {
	p := synthetic(t)
	cases := []struct {
		name, total, quantum string
		mode                 decimal.Mode
		pairs                []string
		shares, quantized    string
		component, weight    string
	}{
		{"AP-T01 weights 600 and 400", "100", "0.01", decimal.HalfUp, []string{"A", "600", "B", "400"}, "A=60 B=40", "100", "0", "1000"},
		{"AP-T02 and NM-04 equal thirds of five quanta", "0.05", "0.01", decimal.HalfUp, []string{"A", "1", "B", "1", "C", "1"}, "A=0.02 B=0.02 C=0.01", "0.05", "0", "3"},
		{"AP-T02 permuted input", "0.05", "0.01", decimal.HalfUp, []string{"C", "1", "A", "1", "B", "1"}, "A=0.02 B=0.02 C=0.01", "0.05", "0", "3"},
		{"AP-T02 and NM-04 negative total", "-0.05", "0.01", decimal.HalfUp, []string{"B", "1", "C", "1", "A", "1"}, "A=-0.02 B=-0.02 C=-0.01", "-0.05", "0", "3"},
		{"AP-T03 quantum 0.05 half up", "1.03", "0.05", decimal.HalfUp, []string{"A", "1", "B", "1"}, "A=0.55 B=0.5", "1.05", "-0.02", "2"},
		{"AP-T04 zero total", "0", "0.01", decimal.HalfUp, []string{"A", "3", "B", "7"}, "A=0 B=0", "0", "0", "10"},
		{"AP-T04 individual zero weight", "10", "0.01", decimal.HalfUp, []string{"A", "0", "B", "5", "C", "5"}, "A=0 B=5 C=5", "10", "0", "10"},
		{"AP-T05 capacity with idle bearer", "1000", "0.01", decimal.HalfUp, []string{"A", "30", "B", "50", "idle", "20"}, "A=300 B=500 idle=200", "1000", "0", "100"},
		{"AP-T05 usage basis as a separate pool", "1000", "0.01", decimal.HalfUp, []string{"A", "30", "B", "50"}, "A=375 B=625", "1000", "0", "80"},
		{"H11 style decimal weights", "100", "0.01", decimal.HalfUp, []string{"A", "0.5", "B", "0.25", "C", "0.25"}, "A=50 B=25 C=25", "100", "0", "1"},
		{"total rounds to zero", "0.004", "0.01", decimal.HalfUp, []string{"A", "1", "B", "1"}, "A=0 B=0", "0", "0.004", "2"},
		{"negative total rounds to zero", "-0.004", "0.01", decimal.HalfUp, []string{"A", "1", "B", "1"}, "A=0 B=0", "0", "-0.004", "2"},
		{"floor of a negative total", "-1.03", "0.05", decimal.Floor, []string{"A", "1", "B", "1"}, "A=-0.55 B=-0.5", "-1.05", "0.02", "2"},
		{"ties by identifier bytes", "0.01", "0.01", decimal.HalfUp, []string{"b", "1", "B", "1", "a", "1"}, "B=0.01 a=0 b=0", "0.01", "0", "3"},
		{"larger remainder wins against the smaller identifier", "0.03", "0.01", decimal.HalfUp, []string{"A", "3", "B", "1"}, "A=0.02 B=0.01", "0.03", "0", "4"},
		{"ceiling is the mode applied", "1.02", "0.05", decimal.Ceiling, []string{"A", "1", "B", "1"}, "A=0.55 B=0.5", "1.05", "-0.03", "2"},
		{"away from zero is the mode applied", "1.02", "0.05", decimal.AwayFromZero, []string{"A", "1", "B", "1"}, "A=0.55 B=0.5", "1.05", "-0.03", "2"},
		{"floor is the mode applied", "1.04", "0.05", decimal.Floor, []string{"A", "1", "B", "1"}, "A=0.5 B=0.5", "1", "0.04", "2"},
		{"toward zero is the mode applied", "-1.04", "0.05", decimal.TowardZero, []string{"A", "1", "B", "1"}, "A=-0.5 B=-0.5", "-1", "-0.04", "2"},
		{"half even is the mode applied", "1.025", "0.05", decimal.HalfEven, []string{"A", "1", "B", "1"}, "A=0.5 B=0.5", "1", "0.025", "2"},
		{"half up on the same tie", "1.025", "0.05", decimal.HalfUp, []string{"A", "1", "B", "1"}, "A=0.55 B=0.5", "1.05", "-0.025", "2"},
		{"unequal weights and a quantum of 0.03", "10", "0.03", decimal.HalfEven, []string{"A", "0.7", "B", "1.25", "C", "3"}, "A=1.41 B=2.52 C=6.06", "9.99", "0.01", "4.95"},
	}
	for _, c := range cases {
		result, err := LargestRemainderV1(request(t, p, c.total, c.quantum, c.mode, targets(t, p, c.pairs...)))
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if got := amounts(result); got != c.shares {
			t.Errorf("%s: shares %s; want %s", c.name, got, c.shares)
		}
		if result.Quantized.String() != c.quantized || result.RoundingComponent.String() != c.component || result.Denominator.String() != c.weight {
			t.Errorf("%s: quantized %s component %s denominator %s", c.name, result.Quantized, result.RoundingComponent, result.Denominator)
		}
		if result.Algorithm != "largest_remainder_v1" || result.Encoding != "ascii_bytewise_v1" || result.ProfileID != "synthetic-allocation" || result.ProfileRevision != "1" || result.Mode != c.mode || result.Original.String() != value(t, p, c.total).String() {
			t.Errorf("%s: result does not echo its inputs: %+v", c.name, result)
		}
	}
}

func TestInvalidPoolsRejectBeforeAnyResult(t *testing.T) {
	p := synthetic(t)
	base := func() Request {
		return request(t, p, "100", "0.01", decimal.HalfUp, targets(t, p, "A", "1", "B", "1"))
	}
	cases := []struct {
		name   string
		change func(*Request)
		want   error
	}{
		{"NM-05 duplicate target", func(r *Request) { r.Targets = targets(t, p, "A", "1", "A", "2") }, ErrDuplicateTargetID},
		{"NM-05 zero total weight", func(r *Request) { r.Targets = targets(t, p, "A", "0", "B", "0") }, ErrZeroTotalWeight},
		{"NM-05 negative weight", func(r *Request) { r.Targets = targets(t, p, "A", "-1", "B", "3") }, ErrNegativeWeight},
		{"AP-T04 missing weight", func(r *Request) { r.Targets = []Target{{ID: "A"}, {ID: "B", Weight: value(t, p, "1")}} }, decimal.ErrInvalidOperand},
		{"no targets", func(r *Request) { r.Targets = nil }, ErrNoTargets},
		{"too many targets", func(r *Request) { r.Limits.MaxTargets = 1 }, ErrTooManyTargets},
		{"missing limits", func(r *Request) { r.Limits = Limits{} }, ErrInvalidLimits},
		{"missing profile", func(r *Request) { r.Profile = decimal.Profile{} }, decimal.ErrInvalidProfile},
		{"missing mode", func(r *Request) { r.Mode = 0 }, decimal.ErrInvalidMode},
		{"missing total", func(r *Request) { r.Total = decimal.Decimal{} }, decimal.ErrInvalidOperand},
		{"zero quantum", func(r *Request) { r.Quantum = value(t, p, "0") }, decimal.ErrInvalidQuantum},
		{"negative quantum", func(r *Request) { r.Quantum = value(t, p, "-0.01") }, decimal.ErrInvalidQuantum},
		{"undeclared encoding", func(r *Request) { r.Encoding = "" }, ErrInvalidEncoding},
		{"unknown encoding", func(r *Request) { r.Encoding = "utf8_nfc_v1" }, ErrInvalidEncoding},
		{"empty identifier", func(r *Request) { r.Targets[0].ID = "" }, ErrInvalidTargetID},
		{"identifier with a space", func(r *Request) { r.Targets[0].ID = "A B" }, ErrInvalidTargetID},
		{"identifier outside ASCII", func(r *Request) { r.Targets[0].ID = "고객" }, ErrInvalidTargetID},
		{"identifier too long", func(r *Request) { r.Targets[0].ID = strings.Repeat("a", 129) }, ErrInvalidTargetID},
		{"intermediate limit", func(r *Request) { r.Limits.MaxIntermediateDigits = 4 }, ErrIntermediateBound},
	}
	for _, c := range cases {
		r := base()
		c.change(&r)
		result, err := LargestRemainderV1(r)
		if !errors.Is(err, c.want) || result.Shares != nil {
			t.Errorf("%s: result %v error %v; want %v", c.name, result.Shares, err, c.want)
		}
	}
	for _, text := range []string{"1e2", "0.000000001", "1234567890123"} {
		if _, err := decimal.Parse(p, text); err == nil {
			t.Errorf("AP-T04 raw input %q must be rejected before it can reach the calculator", text)
		}
	}
	narrow, err := decimal.NewProfile(decimal.ProfileSpec{ID: "synthetic-narrow", Revision: "1", InputIntegerDigits: 3, InputScale: 2, ResultIntegerDigits: 3, ResultScale: 2, IntermediateDigits: 12})
	if err != nil {
		t.Fatal(err)
	}
	overflow := Request{Profile: narrow, Limits: limits, Encoding: EncodingASCIIBytewiseV1, Total: value(t, narrow, "999.99"), Quantum: value(t, narrow, "0.05"), Mode: decimal.AwayFromZero, Targets: targets(t, narrow, "A", "1")}
	if _, err := LargestRemainderV1(overflow); !errors.Is(err, decimal.ErrResultBound) {
		t.Errorf("quantized total beyond the result bound error = %v", err)
	}
	denominator := Request{Profile: narrow, Limits: limits, Encoding: EncodingASCIIBytewiseV1, Total: value(t, narrow, "1"), Quantum: value(t, narrow, "0.01"), Mode: decimal.HalfUp, Targets: targets(t, narrow, "A", "999", "B", "999")}
	if _, err := LargestRemainderV1(denominator); !errors.Is(err, decimal.ErrResultBound) {
		t.Errorf("denominator beyond the result bound error = %v", err)
	}
}

func TestIdentifierAtTheLengthLimitIsAccepted(t *testing.T) {
	p := synthetic(t)
	longest := strings.Repeat("z", 128)
	result, err := LargestRemainderV1(request(t, p, "1", "0.01", decimal.HalfUp, targets(t, p, longest, "1", "A.b_c:d-9", "1")))
	if err != nil || amounts(result) != "A.b_c:d-9=0.5 "+longest+"=0.5" {
		t.Errorf("result %s error %v", amounts(result), err)
	}
}
