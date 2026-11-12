package decimal

import (
	"errors"
	"math/big"
	"testing"
)

func synthetic(t testing.TB) Profile {
	t.Helper()
	p, err := NewProfile(ProfileSpec{ID: "synthetic-test", Revision: "1", InputIntegerDigits: 12, InputScale: 8, ResultIntegerDigits: 18, ResultScale: 12, IntermediateDigits: 80})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func parse(t testing.TB, p Profile, text string) Decimal {
	t.Helper()
	value, err := Parse(p, text)
	if err != nil {
		t.Fatalf("parse %q: %v", text, err)
	}
	return value
}

func TestParseAcceptsOnlyFiniteBaseTenText(t *testing.T) {
	p := synthetic(t)
	for text, want := range map[string]string{
		"0": "0", "-0": "0", "-0.00": "0", "0.10": "0.1", "100.00": "100", "-12.50": "-12.5",
		"0.00000001": "0.00000001", "999999999999.99999999": "999999999999.99999999", "7": "7",
	} {
		got, err := Parse(p, text)
		if err != nil || got.String() != want {
			t.Errorf("Parse(%q) = %v, %v; want %s", text, got, err, want)
		}
	}
	for _, text := range []string{"", "-", "+1", " 1", "1 ", "1e3", "1E3", "NaN", "Infinity", "-Inf", "1,000", ".5", "5.", "00.1", "01", "1.2.3", "0x10", "１２", "1_000", "--1", "1.-2"} {
		if _, err := Parse(p, text); !errors.Is(err, ErrSyntax) {
			t.Errorf("Parse(%q) error = %v; want ErrSyntax", text, err)
		}
	}
	for _, text := range []string{"1234567890123", "0.123456789", "1.100000000", "1234567890123.000000000000000000000001"} {
		if _, err := Parse(p, text); !errors.Is(err, ErrInputBound) {
			t.Errorf("Parse(%q) error = %v; want ErrInputBound", text, err)
		}
	}
}

func TestMissingProfileModeAndOperandsReject(t *testing.T) {
	p := synthetic(t)
	one := parse(t, p, "1")
	if _, err := Parse(Profile{}, "1"); !errors.Is(err, ErrInvalidProfile) {
		t.Errorf("zero profile parse error = %v", err)
	}
	if _, err := Add(Profile{}, one, one); !errors.Is(err, ErrInvalidProfile) {
		t.Errorf("zero profile add error = %v", err)
	}
	if _, err := Add(p, one, Decimal{}); !errors.Is(err, ErrInvalidOperand) {
		t.Errorf("missing operand error = %v", err)
	}
	if (Decimal{}).Equal(Decimal{}) || (Decimal{}).Valid() {
		t.Error("the zero Decimal must not be a value")
	}
	var unset Mode
	if _, err := Round(p, one, 0, unset); !errors.Is(err, ErrInvalidMode) {
		t.Errorf("unset mode error = %v", err)
	}
	if _, err := ParseMode("bankers"); !errors.Is(err, ErrInvalidMode) {
		t.Errorf("unknown mode error = %v", err)
	}
	for _, spec := range []ProfileSpec{
		{},
		{ID: "synthetic", Revision: "1", InputIntegerDigits: 12, InputScale: 8, ResultIntegerDigits: 18, ResultScale: 12},
		{ID: "synthetic", Revision: "1", InputIntegerDigits: 12, InputScale: 8, ResultIntegerDigits: 10, ResultScale: 12, IntermediateDigits: 80},
		{Revision: "1", InputIntegerDigits: 12, InputScale: 8, ResultIntegerDigits: 18, ResultScale: 12, IntermediateDigits: 80},
	} {
		if _, err := NewProfile(spec); !errors.Is(err, ErrInvalidProfile) {
			t.Errorf("NewProfile(%+v) error = %v", spec, err)
		}
	}
}

func TestContractCaseNM01ExactArithmetic(t *testing.T) {
	p := synthetic(t)
	sum, err := Add(p, parse(t, p, "0.1"), parse(t, p, "0.2"))
	if err != nil || sum.String() != "0.3" {
		t.Fatalf("0.1+0.2 = %v, %v", sum, err)
	}
	rest, err := Sub(p, parse(t, p, "1"), parse(t, p, "0.20"))
	if err != nil {
		t.Fatal(err)
	}
	quotient, err := Quo(p, parse(t, p, "100"), rest)
	if err != nil || quotient.String() != "125" {
		t.Fatalf("100/(1-0.20) = %v, %v", quotient, err)
	}
}

func TestContractCaseNM02DivisionNeedsExplicitRounding(t *testing.T) {
	p := synthetic(t)
	one, two, three := parse(t, p, "1"), parse(t, p, "2"), parse(t, p, "3")
	if _, err := Quo(p, one, three); !errors.Is(err, ErrInexactDivision) {
		t.Fatalf("1/3 error = %v", err)
	}
	for _, c := range []struct {
		numerator Decimal
		want      string
	}{{one, "0.3333"}, {two, "0.6667"}} {
		got, err := DivideRound(p, c.numerator, three, 4, HalfUp)
		if err != nil || got.String() != c.want {
			t.Errorf("divide_round = %v, %v; want %s", got, err, c.want)
		}
	}
	if _, err := Quo(p, one, parse(t, p, "0")); !errors.Is(err, ErrDivisionByZero) {
		t.Errorf("division by zero error = %v", err)
	}
}

func TestContractCaseNM03RejectsInsteadOfTruncating(t *testing.T) {
	p := synthetic(t)
	large := parse(t, p, "999999999999")
	product, err := Mul(p, large, large)
	if !errors.Is(err, ErrResultBound) || product.Valid() {
		t.Errorf("overflowing product = %v, %v", product, err)
	}
	edge, err := Mul(p, parse(t, p, "999999999"), parse(t, p, "999999999"))
	if err != nil {
		t.Fatal(err)
	}
	if sum, err := Add(p, edge, edge); !errors.Is(err, ErrResultBound) || sum.Valid() {
		t.Errorf("overflowing sum = %v, %v", sum, err)
	}
	if difference, err := Sub(p, edge, parse(t, p, "0.00000001")); err != nil || difference.String() != "999999998000000000.99999999" {
		t.Errorf("difference at the bound = %v, %v", difference, err)
	}
	small := parse(t, p, "0.00000001")
	if _, err := Mul(p, small, small); !errors.Is(err, ErrResultBound) {
		t.Errorf("excess scale product error = %v", err)
	}
	eighth, err := NewProfile(ProfileSpec{ID: "synthetic-narrow", Revision: "1", InputIntegerDigits: 4, InputScale: 2, ResultIntegerDigits: 4, ResultScale: 2, IntermediateDigits: 12})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Quo(eighth, parse(t, eighth, "1"), parse(t, eighth, "8")); !errors.Is(err, ErrInexactDivision) {
		t.Errorf("1/8 at two places error = %v", err)
	}
	for _, text := range []string{"1e2", "0.1e-3", "1.5E+2", "NaN"} {
		if _, err := Parse(p, text); !errors.Is(err, ErrSyntax) {
			t.Errorf("float text %q error = %v", text, err)
		}
	}
	usd, err := NewMoney(parse(t, p, "10"), "USD")
	if err != nil {
		t.Fatal(err)
	}
	krw, err := NewMoney(parse(t, p, "10"), "KRW")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AddMoney(p, usd, krw); !errors.Is(err, ErrCurrencyMismatch) {
		t.Errorf("mixed currency error = %v", err)
	}
	total, err := AddMoney(p, usd, usd)
	if err != nil || total.Amount().String() != "20" || total.Currency() != "USD" {
		t.Errorf("same currency sum = %v %s, %v", total.Amount(), total.Currency(), err)
	}
	for _, code := range []string{"usd", "US", "USDX", "U$D", ""} {
		if _, err := NewMoney(parse(t, p, "1"), code); !errors.Is(err, ErrInvalidCurrency) {
			t.Errorf("currency %q error = %v", code, err)
		}
	}
	narrow, err := NewProfile(ProfileSpec{ID: "synthetic-intermediate", Revision: "1", InputIntegerDigits: 12, InputScale: 8, ResultIntegerDigits: 30, ResultScale: 16, IntermediateDigits: 46})
	if err != nil {
		t.Fatal(err)
	}
	wide := parse(t, narrow, "999999999999.99999999")
	square, err := Mul(narrow, wide, wide)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Mul(narrow, square, square); !errors.Is(err, ErrIntermediateBound) {
		t.Errorf("intermediate bound error = %v", err)
	}
}

func TestRoundingModesOnTiesAndSigns(t *testing.T) {
	p := synthetic(t)
	cases := []struct {
		value string
		mode  Mode
		want  string
	}{
		{"2.5", TowardZero, "2"}, {"-2.5", TowardZero, "-2"}, {"2.1", AwayFromZero, "3"}, {"-2.1", AwayFromZero, "-3"},
		{"2.9", Floor, "2"}, {"-2.1", Floor, "-3"}, {"2.1", Ceiling, "3"}, {"-2.9", Ceiling, "-2"},
		{"0.5", HalfUp, "1"}, {"-0.5", HalfUp, "-1"}, {"2.5", HalfUp, "3"}, {"-2.5", HalfUp, "-3"}, {"2.4", HalfUp, "2"}, {"-2.6", HalfUp, "-3"},
		{"0.5", HalfEven, "0"}, {"1.5", HalfEven, "2"}, {"2.5", HalfEven, "2"}, {"-2.5", HalfEven, "-2"}, {"-1.5", HalfEven, "-2"}, {"2.51", HalfEven, "3"}, {"-0.4", HalfEven, "0"},
		{"2", Floor, "2"}, {"-2", Ceiling, "-2"}, {"0", AwayFromZero, "0"},
	}
	for _, c := range cases {
		got, err := Round(p, parse(t, p, c.value), 0, c.mode)
		if err != nil || got.String() != c.want {
			t.Errorf("Round(%s, 0, %s) = %v, %v; want %s", c.value, c.mode, got, err, c.want)
		}
	}
	value := parse(t, p, "1.23456")
	if same, err := Round(p, value, 8, HalfUp); err != nil || !same.Equal(value) {
		t.Errorf("rounding to a wider scale = %v, %v", same, err)
	}
	if _, err := Round(p, value, -1, HalfUp); !errors.Is(err, ErrInvalidScale) {
		t.Errorf("negative scale error = %v", err)
	}
	if _, err := Round(p, value, 13, HalfUp); !errors.Is(err, ErrInvalidScale) {
		t.Errorf("excess scale error = %v", err)
	}
	for name, mode := range map[string]Mode{"toward_zero": TowardZero, "away_from_zero": AwayFromZero, "floor": Floor, "ceiling": Ceiling, "half_up": HalfUp, "half_even": HalfEven} {
		got, err := ParseMode(name)
		if err != nil || got != mode || got.String() != name {
			t.Errorf("ParseMode(%s) = %v, %v", name, got, err)
		}
	}
}

func TestRoundToQuantumThatIsNotAPowerOfTen(t *testing.T) {
	p := synthetic(t)
	cases := []struct {
		value, quantum string
		mode           Mode
		want           string
	}{
		{"1.03", "0.05", HalfUp, "1.05"}, {"1.02", "0.05", HalfUp, "1"}, {"1.025", "0.05", HalfUp, "1.05"}, {"1.025", "0.05", HalfEven, "1"},
		{"1.075", "0.05", HalfEven, "1.1"}, {"-1.03", "0.05", Floor, "-1.05"}, {"1.03", "0.05", Floor, "1"}, {"9.99", "0.05", AwayFromZero, "10"},
		{"7", "0.25", Ceiling, "7"}, {"7.01", "0.25", Ceiling, "7.25"}, {"100", "3", TowardZero, "99"}, {"0.004", "0.01", HalfUp, "0"}, {"-0.004", "0.01", HalfUp, "0"},
	}
	for _, c := range cases {
		got, err := RoundToQuantum(p, parse(t, p, c.value), parse(t, p, c.quantum), c.mode)
		if err != nil || got.String() != c.want {
			t.Errorf("RoundToQuantum(%s, %s, %s) = %v, %v; want %s", c.value, c.quantum, c.mode, got, err, c.want)
		}
	}
	for _, quantum := range []string{"0", "-0.01"} {
		if _, err := RoundToQuantum(p, parse(t, p, "1"), parse(t, p, quantum), HalfUp); !errors.Is(err, ErrInvalidQuantum) {
			t.Errorf("quantum %s error = %v", quantum, err)
		}
	}
}

func TestOperandsAreCheckedAgainstThePassedProfile(t *testing.T) {
	wide := synthetic(t)
	narrow, err := NewProfile(ProfileSpec{ID: "synthetic-narrow", Revision: "1", InputIntegerDigits: 4, InputScale: 2, ResultIntegerDigits: 4, ResultScale: 2, IntermediateDigits: 12})
	if err != nil {
		t.Fatal(err)
	}
	outside := parse(t, wide, "123456.789")
	if _, err := Add(narrow, outside, parse(t, narrow, "1")); !errors.Is(err, ErrResultBound) {
		t.Errorf("operand from a wider profile error = %v", err)
	}
	if _, err := New(narrow, new(big.Int).SetInt64(12345), 0); !errors.Is(err, ErrResultBound) {
		t.Errorf("New beyond bounds error = %v", err)
	}
	made, err := New(narrow, new(big.Int).SetInt64(-1250), 2)
	if err != nil || made.String() != "-12.5" {
		t.Errorf("New(-1250, 2) = %v, %v", made, err)
	}
	coefficient := made.Coefficient()
	coefficient.SetInt64(7)
	if made.String() != "-12.5" {
		t.Error("Coefficient must return a copy")
	}
}
