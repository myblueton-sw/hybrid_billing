package exports

import (
	"math/big"
	"strings"
	"testing"
)

func TestValidators(t *testing.T) {
	if !validID(strings.Repeat("a", 128)) || validID(strings.Repeat("a", 129)) || validID("") {
		t.Fatal("id length")
	}
	for _, id := range []string{"A", "Z", "a", "z", "0", "9", ".", "_", ":", "-"} {
		if !validID(id) {
			t.Fatalf("%q", id)
		}
	}
	for _, id := range []string{"@", "[", "`", "{", "/", ";", " a", "!a"} {
		if validID(id) {
			t.Fatalf("%q", id)
		}
	}
	for _, d := range []string{strings.Repeat("0", 64), strings.Repeat("9", 64), strings.Repeat("f", 64), "a" + strings.Repeat("0", 63)} {
		if !validDigest(d) {
			t.Fatal(d)
		}
	}
	for _, d := range []string{"G" + strings.Repeat("0", 63), "/" + strings.Repeat("0", 63), ":" + strings.Repeat("0", 63), "`" + strings.Repeat("0", 63), "g" + strings.Repeat("0", 63), strings.Repeat("0", 63)} {
		if validDigest(d) {
			t.Fatal(d)
		}
	}
	for _, m := range []string{"2006-01", "2010-12", "2090-05", "2019-09", "2000-10"} {
		if !validMonth(m) {
			t.Fatal(m)
		}
	}
	for _, m := range []string{"201a-01", "20a1-01", "20/6-01", "202:-01", "2026-13", "2026-00", "1926-01", "2026_01", "2026-7"} {
		if validMonth(m) {
			t.Fatal(m)
		}
	}
	for _, c := range []string{"AAA", "ZZZ", "AZA"} {
		if !validCurrency(c) {
			t.Fatal(c)
		}
	}
	for _, c := range []string{"aAA", "A@A", "a@A", "AA@", "@AA", "[AA", "A[A", "AA[", "AA", "AAAA"} {
		if validCurrency(c) {
			t.Fatal(c)
		}
	}
	for _, z := range []string{"UTC", "+00:00", "-14:59", "+09:30"} {
		if !validZone(z) {
			t.Fatal(z)
		}
	}
	for _, z := range []string{"+09-00", "x09:00", "+/9:00", "+0/:00", "+09:/0", "+09:0/", "+0900", "+15:00"} {
		if validZone(z) {
			t.Fatal(z)
		}
	}
	for text, ok := range map[string]bool{"0": true, "10": true, "1.5": true, "-1.5": true, "123456789012345678": true, "1234567890123456789": false,
		"01": false, "00": false, "-0": false, "-0.00": false, "1.": false, ".5": false, "1.123456789": false, "": false, "-": false, "1a": false, "1.a": false, "+1": false} {
		if _, got := units(text, 8); got != ok {
			t.Fatalf("%q: %v", text, got)
		}
	}
	if v, _ := units("12.5", 2); v.Cmp(big.NewInt(1250)) != 0 {
		t.Fatal(v)
	}
	if v, _ := units("-7", 0); v.Cmp(big.NewInt(-7)) != 0 {
		t.Fatal(v)
	}
	if show(big.NewInt(5), 1) != "0.5" || show(big.NewInt(5), 0) != "5" || show(big.NewInt(-1234), 2) != "-12.34" || show(big.NewInt(0), 2) != "0.00" {
		t.Fatal("show")
	}
	in := "a\x01b\tc\nd\re f\x7fg￾h￿i�j"
	if got := clean(in); got != "a�b\tc\nd\re f�g�h�i�j" {
		t.Fatalf("clean %q", got)
	}
	if !risky("-1234567890123456789") || risky("1234567890123456789") || risky("-12.5") || !risky("-") || !risky(" =x") || risky("") || risky("   ") {
		t.Fatal("risky")
	}
}
