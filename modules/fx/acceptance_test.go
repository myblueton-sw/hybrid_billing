package fx

import (
	"testing"
)

func TestTheFiguresOfTheAcceptance(t *testing.T) {
	r, _ := open(t)
	policyVersion(t, r, "msp-usd", attrs(LevelMSP, "", "USD", "KRW"), dated(1, "2026-01-01", ConstructionDirect))
	policyVersion(t, r, "con-1-usd", attrs(LevelContract, "con-1", "USD", "KRW"), VersionInput{Version: 1, Effective: "2026-01-01", Author: "author", Method: MethodDefaultAllowed})
	feed(t, r, key("USD", "KRW", "2026-11-10"), 1, 1, "1350")
	first := prepare(t, r, "run-1", need("n1", "USD", "KRW", "2026-11-10"))
	if got := entryOf(first); got != "n1:msp_default/msp/msp-usd quote 1350 rate 1350" {
		t.Fatalf("entry: %s", got)
	}
	if e := first.Entries[0]; e.AllowingPolicy != "con-1-usd" || e.AllowingVersion != 1 || len(e.Observations) != 1 || e.Observations[0].Revision != 1 {
		t.Fatalf("selection kept: %+v", e)
	}
	if got := shown(convert(t, r, "run-1", line("l1", "n1", "900", "USD", StateSource))); got != "l1=1215000 KRW fee 0 total 1215000 quoted 1215000" {
		t.Fatalf("900 at 1350: %s", got)
	}
	feed(t, r, key("USD", "KRW", "2026-11-10"), 2, 1, "1360")
	second := prepare(t, r, "run-2", need("n1", "USD", "KRW", "2026-11-10"))
	if got := shown(convert(t, r, "run-2", line("l1", "n1", "900", "USD", StateSource))); got != "l1=1224000 KRW fee 0 total 1224000 quoted 1224000" {
		t.Fatalf("900 at 1360: %s", got)
	}
	if got := shown(convert(t, r, "run-1", line("l1", "n1", "900", "USD", StateSource))); got != "l1=1215000 KRW fee 0 total 1215000 quoted 1215000" {
		t.Fatalf("the first snapshot after the revision: %s", got)
	}
	if first.ContentDigest == second.ContentDigest || first.Digest == second.Digest {
		t.Fatal("a changed rate kept the digest")
	}
	again := prepare(t, r, "run-1", need("n1", "USD", "KRW", "2026-11-10"))
	if again.ContentDigest != first.ContentDigest || again.Digest != first.Digest || entryOf(again) != entryOf(first) {
		t.Fatal("a repeat of a prepared snapshot resolved again")
	}
	third := prepare(t, r, "run-3", need("n1", "USD", "KRW", "2026-11-10"))
	if third.ContentDigest != second.ContentDigest || third.Digest == second.Digest {
		t.Fatal("equal content under another identifier")
	}
}

func TestQuotedUnitsInversesAndLargeAmounts(t *testing.T) {
	r, _ := open(t)
	jpy := dated(1, "2026-01-01", ConstructionDirect)
	policyVersion(t, r, "jpy", attrs(LevelContract, "con-1", "JPY", "KRW"), jpy)
	policyVersion(t, r, "krw-usd", attrs(LevelContract, "con-1", "KRW", "USD"), dated(1, "2026-01-01", ConstructionInverse))
	precise := dated(1, "2026-01-01", ConstructionInverse)
	precise.QuoteScale, precise.Quantum = 0, "0.01"
	policyVersion(t, r, "krw-eur", attrs(LevelContract, "con-1", "KRW", "EUR"), precise)
	feed(t, r, key("JPY", "KRW", "2026-11-10"), 1, 100, "905.25")
	feed(t, r, key("USD", "KRW", "2026-11-10"), 1, 1, "1350")
	feed(t, r, key("EUR", "KRW", "2026-11-10"), 1, 1, "1450.4")
	snap := prepare(t, r, "run", need("jpy", "JPY", "KRW", "2026-11-10"), need("usd", "KRW", "USD", "2026-11-10"), need("eur", "KRW", "EUR", "2026-11-10"))
	if got := entryOf(snap); got != "jpy:contract/contract/jpy quote 905.25 rate 3621/400; usd:contract/contract/krw-usd quote 1350 rate 1/1350; eur:held:rate_precision_lost" {
		t.Fatalf("entries: %s", got)
	}
	if e := snap.Entries[0]; e.Unit != 100 || e.QuotedRate != "3621/4" || e.PerSource != "3621/400" {
		t.Fatalf("per 100: %+v", e)
	}
	if e := snap.Entries[2]; e.SalesQuote != "1450" || e.Difference != "1/3625" || e.ExactQuote != "7252/5" {
		t.Fatalf("precision: %+v", e)
	}
	out := convert(t, r, "run", line("a", "jpy", "1000000000", "JPY", StateSource), line("b", "usd", "1215000000", "KRW", StateSource), line("c", "usd", "1000", "KRW", StateSource), line("d", "eur", "100", "KRW", StateSource))
	if got := shown(out); got != "a=9052500000 KRW fee 0 total 9052500000 quoted 9052500000; b=900000 USD fee 0 total 900000 quoted 900000; c=1 USD fee 0 total 1 quoted 1; d:held:rate_precision_lost" {
		t.Fatalf("conversions: %s", got)
	}
	if out.Held != 1 || out.Label != "fx_v1" || out.Digest != snap.Digest || out.ContentDigest != snap.ContentDigest {
		t.Fatalf("result: %+v", out)
	}
}

func TestSpreadFeeAndRounding(t *testing.T) {
	r, _ := open(t)
	v := dated(1, "2026-01-01", ConstructionDirect)
	v.Spread, v.Fee, v.QuoteScale, v.Quantum, v.AmountMode = "0.005", "0.1", 1, "1", "half_up"
	policyVersion(t, r, "usd", attrs(LevelContract, "con-1", "USD", "KRW"), v)
	inverse := dated(1, "2026-01-01", ConstructionInverse)
	inverse.Spread, inverse.QuoteScale, inverse.QuoteMode, inverse.Quantum, inverse.Tolerance = "0.01", 4, "half_even", "0.01", "0.01"
	policyVersion(t, r, "krw", attrs(LevelContract, "con-1", "KRW", "USD"), inverse)
	feed(t, r, key("USD", "KRW", "2026-11-10"), 1, 1, "999.5")
	snap := prepare(t, r, "run", need("n", "USD", "KRW", "2026-11-10"), need("m", "KRW", "USD", "2026-11-10"))
	e := snap.Entries[0]
	if e.ExactQuote != "401799/400" || e.SalesQuote != "1004.5" || e.SalesRate != "2009/2" || e.Difference != "1/401799" {
		t.Fatalf("spread on a direct quote: %+v", e)
	}
	if got := shown(convert(t, r, "run", line("a", "n", "1", "USD", StateSource))); got != "a=1005 KRW fee 100 total 1105 quoted 1000" {
		t.Fatalf("the fee from the exact amount: %s", got)
	}
	if got := shown(convert(t, r, "run", line("a", "n", "-1", "USD", StateSource), line("b", "n", "0", "USD", StateSource))); got != "a=-1005 KRW fee -100 total -1105 quoted -1000; b=0 KRW fee 0 total 0 quoted 0" {
		t.Fatalf("a credit: %s", got)
	}
	m := snap.Entries[1]
	if m.ExactQuote != "99950/101" || m.SalesQuote != "989.604" {
		t.Fatalf("spread on an inverse quote: %+v", m)
	}
	if m.SalesRate != "250/247401" {
		t.Fatalf("inverse sales rate: %s", m.SalesRate)
	}
	if got := shown(convert(t, r, "run", line("a", "m", "1000000", "KRW", StateSource))); got != "a=1010.51 USD fee 0 total 1010.51 quoted 1000.5" {
		t.Fatalf("an inverse with a spread: %s", got)
	}
}
