package marketplace

import (
	"reflect"
	"testing"
)

func TestSumsWithinAKindAndAcrossTheYear(t *testing.T) {
	r := open(t)
	long := buyerPrivate("tx-1")
	long.Term.End = "2027-12"
	record(t, r, long)
	write(t, r, "tx-1", "2027-02", 1, Facts{Lines: []Line{line("a", KindOther, "1.5", unknown()), line("b", KindOther, "2.25", unknown())}})
	sealed(t, r, "tx-1", "2027-02", 1)
	v, _ := r.AgreementCharges("ws", "aws", "agr-1", "2027-02")
	if v.Groups[0].Sums != (KindSums{"0", "0", "3.75"}) || v.Entries[0].Sums != (KindSums{"0", "0", "3.75"}) {
		t.Fatalf("one kind twice: %+v", v)
	}
	for _, call := range []func() error{
		func() error { _, err := r.AgreementCharges("", "aws", "agr-1", "2027-02"); return err },
		func() error { _, err := r.AgreementCharges("ws", "", "agr-1", "2027-02"); return err },
		func() error { _, err := r.AgreementCharges("ws", "aws", "a b", "2027-02"); return err },
		func() error { _, err := r.AgreementCharges("ws", "aws", "agr-1", "2027-0a"); return err },
	} {
		if err := call(); kind(err) != "invalid" {
			t.Errorf("invalid: %v", err)
		}
	}
	v1 := terms("pol", "1", "tx-1", "0", []string{KindSoftware}, Excluded)
	v1.Last = "2027-12"
	approve(t, r, v1)
	v2 := terms("pol", "2", "tx-1", "0", []string{KindSoftware}, Excluded)
	v2.First, v2.Last = "2027-01", "2027-12"
	approve(t, r, v2)
	if s, _ := r.PolicyState("ws", "pol", "1"); s.EffectiveLast != "2026-12" {
		t.Fatalf("the month before January: %+v", s)
	}
	changed := v1
	changed.Scope = []string{KindOther}
	if _, err := r.RecordPolicy("ws", changed, "author-1"); kind(err) != "conflict" {
		t.Fatalf("another scope: %v", err)
	}
	changed.Scope = []string{KindOther, KindSoftware}
	if _, err := r.RecordPolicy("ws", changed, "author-1"); kind(err) != "conflict" {
		t.Fatalf("a longer scope: %v", err)
	}
}

func TestRepeatsComparedFieldByField(t *testing.T) {
	r := open(t)
	record(t, r, buyerPrivate("tx-1"))
	facts := exampleFacts()
	write(t, r, "tx-1", "2026-03", 1, facts)
	other := Facts{Lines: facts.Lines, Recognitions: []Recognition{{Program: "macc-1", State: Unknown}}}
	if _, err := r.RecordStatement("ws", "tx-1", "2026-03", 1, other); kind(err) != "conflict" {
		t.Fatalf("a changed recognition: %v", err)
	}
	for _, field := range []func(*PolicyTerms){
		func(p *PolicyTerms) { p.Customer = "cust-2" }, func(p *PolicyTerms) { p.First = "2026-02" }, func(p *PolicyTerms) { p.Last = "2026-11" },
		func(p *PolicyTerms) { p.Rate = "0.3" }, func(p *PolicyTerms) { p.SourceDiscounted, p.SourceBasis = Excluded, "" },
		func(p *PolicyTerms) { p.Evidence = evidenceA }, func(p *PolicyTerms) { p.SourceBasis = evidenceA }, func(p *PolicyTerms) { p.Quantum = "1" },
	} {
		base := terms("pol", "1", "tx-1", "0.2", []string{KindSoftware}, Included)
		if _, err := r.RecordPolicy("ws", base, "author-1"); err != nil {
			t.Fatal(err)
		}
		field(&base)
		if _, err := r.RecordPolicy("ws", base, "author-1"); kind(err) != "conflict" {
			t.Errorf("%+v: %v", base, err)
		}
	}
}

func TestPayoutsOfAnotherProvider(t *testing.T) {
	r := open(t)
	record(t, r, deal("tx", RoleSeller, ChannelMarketplace, TypePublic, CollectorMarketplace))
	gcp := deal("tx-g", RoleSeller, ChannelMarketplace, TypePublic, CollectorMarketplace)
	gcp.Provider = "gcp"
	record(t, r, gcp)
	paid := SettlementFacts{State: SettlementPaid, Currency: "USD", Gross: "10", Fee: "1", Refund: "0", Tax: "0", Hold: "0", Net: "9", Report: "rep", Payout: "pay", Received: "9"}
	r.RecordSettlement("ws", "tx", "s1", 1, paid)
	if v, _ := r.RecordSettlement("ws", "tx-g", "s1", 1, paid); !v.Received {
		t.Fatalf("another provider's payout: %+v", v)
	}
	reported := paid
	reported.State, reported.Payout, reported.Received = SettlementReported, "", ""
	r.RecordSettlement("ws", "tx", "s2", 1, reported)
	if v, _ := r.RecordSettlement("ws", "tx", "s2", 2, paid); !reflect.DeepEqual(v.Reasons, []string{ReasonPayoutReused}) {
		t.Fatalf("a payout of the same provider: %+v", v)
	}
	if v, _ := r.RecordSettlement("ws", "tx", "s1", 2, paid); v.Received != false || !reflect.DeepEqual(v.Reasons, []string{ReasonPayoutReused}) {
		t.Fatalf("judged against a later reuse: %+v", v)
	}
}
