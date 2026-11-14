package marketplace

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func TestFormsAtTheirBounds(t *testing.T) {
	ones := Limits{Transactions: 1, Revisions: 1, Lines: 1, Recognitions: 1, Settlements: 1, Policies: 1, Digits: 1}
	if _, err := NewRegistry(testProfile(t), ones, time.Now); err != nil {
		t.Fatal("limits of one")
	}
	r := open(t)
	tx := buyerPrivate(strings.Repeat("Z", 128))
	tx.Provider, tx.Agreement = "AZaz09.-_:", "agr"
	tx.Term = Term{Accepted: "2000-01", Start: "2000-01", End: "2000-01", Currency: "AZA"}
	if _, err := r.RecordTransaction("ws", tx); err != nil {
		t.Fatalf("identifiers, months and currency at their bounds: %v", err)
	}
	for _, month := range []string{"2099-12", "2026-10", "2026-09"} {
		other := buyerPrivate("m" + month)
		other.Term = Term{Accepted: month, Start: month, End: month, Currency: "ZAZ"}
		if _, err := r.RecordTransaction("ws", other); err != nil {
			t.Fatalf("%s: %v", month, err)
		}
	}
	for _, month := range []string{"a026-01", "2a26-01", "20a6-01", "202a-01", "2026-a1", "2026-0a", "2026-00", "2026-13", "1999-12", "2100-01"} {
		other := buyerPrivate("bad")
		other.Term.Accepted = month
		if _, err := r.RecordTransaction("ws", other); kind(err) != "invalid" {
			t.Errorf("month %s: %v", month, err)
		}
	}
	for _, id := range []string{"/a", "a@", "a[", "a`", "a{", strings.Repeat("a", 129)} {
		if _, err := r.RecordTransaction("ws", buyerPrivate(id)); kind(err) != "invalid" {
			t.Errorf("id %q: %v", id, err)
		}
	}
	for _, code := range []string{"@AA", "A[A", "AA@", "[AA", "A@A", "AA["} {
		other := buyerPrivate("cur")
		other.Term.Currency = code
		if _, err := r.RecordTransaction("ws", other); kind(err) != "invalid" {
			t.Errorf("currency %q: %v", code, err)
		}
	}
	if _, err := r.AgreementCharges("ws", "aws", "agr", "2026-13"); kind(err) != "invalid" {
		t.Fatal("month thirteen")
	}
	reseller := deal("res", RoleReseller, ChannelMarketplace, TypePublic, CollectorMSP)
	reseller.Participants.Reseller = "a b"
	if _, err := r.RecordTransaction("ws", reseller); kind(err) != "invalid" {
		t.Fatal("a malformed reseller")
	}
	early := buyerPrivate("early")
	early.Term.Accepted = "1999-12"
	if _, err := r.RecordTransaction("ws", early); kind(err) != "invalid" {
		t.Fatal("an accepted month before the bounds")
	}
	digests := []string{strings.Repeat("09af", 16), "g" + strings.Repeat("a", 63), "/" + strings.Repeat("a", 63), ":" + strings.Repeat("a", 63), "`" + strings.Repeat("a", 63)}
	for index, digest := range digests {
		q := open(t)
		record(t, q, buyerPrivate("tx"))
		_, err := q.RecordStatement("ws", "tx", "2026-01", 1, Facts{Lines: []Line{line("a", KindSoftware, "1", Benefit{State: NotApplied, Evidence: digest})}})
		if (index == 0) != (err == nil) {
			t.Errorf("digest %q: %v", digest, err)
		}
	}
	wide, _ := decimal.NewProfile(decimal.ProfileSpec{ID: "wide", Revision: "1", InputIntegerDigits: 70, InputScale: 8, ResultIntegerDigits: 80, ResultScale: 8, IntermediateDigits: 200})
	w, _ := NewRegistry(wide, testLimits(), time.Now)
	w.RecordTransaction("ws", buyerPrivate("tx"))
	if _, err := w.RecordStatement("ws", "tx", "2026-01", 1, Facts{Lines: []Line{line("a", KindSoftware, strings.Repeat("9", 64), notApplied())}}); err != nil {
		t.Fatalf("a number of 64 bytes: %v", err)
	}
	if _, err := w.RecordStatement("ws", "tx", "2026-02", 1, Facts{Lines: []Line{line("a", KindSoftware, strings.Repeat("9", 65), notApplied())}}); kind(err) != "invalid" {
		t.Fatalf("a number of 65 bytes: %v", err)
	}
	q := open(t)
	record(t, q, buyerPrivate("tx"))
	write(t, q, "tx", "2026-01", 1, Facts{Lines: []Line{line("a", KindSoftware, "1", notApplied())}})
	write(t, q, "tx", "2026-12", 1, Facts{Lines: []Line{line("b", KindSoftware, "0", notApplied())}})
	if v, err := q.AgreementCharges("ws", "aws", "agr-1", "2026-12"); err != nil || len(v.Entries) != 1 {
		t.Fatalf("the last month of the term: %v", err)
	}
	approve(t, q, terms("pol", "1", "tx", "0.5", []string{KindSoftware}, Excluded))
	for _, month := range []string{"2026-01", "2026-12"} {
		s := sealed(t, q, "tx", month, 1)
		p := policyFor(t, q, "tx", month)
		v, err := ComputeCharge(s, p)
		if err != nil || v.View().Hold != "" {
			t.Fatalf("%s at the policy's bounds: %v", month, err)
		}
	}
	if v := compute(t, sealed(t, q, "tx", "2026-12", 1), policyFor(t, q, "tx", "2026-12")); v.Total != "0" || v.Lines[0].Amount != "0" {
		t.Fatalf("a line charged nothing: %+v", v)
	}
}

func TestRevisionNumbersAtTheirBounds(t *testing.T) {
	r := open(t)
	record(t, r, deal("tx", RoleSeller, ChannelMarketplace, TypePublic, CollectorMarketplace))
	write(t, r, "tx", "2026-01", 1, Facts{})
	expected := SettlementFacts{State: SettlementExpected, Currency: "USD", Gross: "1", Fee: "0", Refund: "0", Tax: "0", Hold: "0", Net: "1"}
	for name, call := range map[string]func() error{
		"statement":   func() error { _, err := r.RecordStatement("ws", "tx", "2026-01", 1000, Facts{}); return err },
		"settlement":  func() error { _, err := r.RecordSettlement("ws", "tx", "s", 1000, expected); return err },
		"seal":        func() error { _, err := r.Seal("ws", "tx", "2026-01", 1000); return err },
		"read":        func() error { _, err := r.Statement("ws", "tx", "2026-01", 1000); return err },
		"seal above":  func() error { _, err := r.Seal("ws", "tx", "2026-01", 1001); return err },
		"read above":  func() error { _, err := r.Statement("ws", "tx", "2026-01", 1001); return err },
		"seal ws":     func() error { _, err := r.Seal("", "tx", "2026-01", 1); return err },
		"seal id":     func() error { _, err := r.Seal("ws", "", "2026-01", 1); return err },
		"seal month":  func() error { _, err := r.Seal("ws", "tx", "", 1); return err },
		"read ws":     func() error { _, err := r.Statement("", "tx", "2026-01", 1); return err },
		"read id":     func() error { _, err := r.Statement("ws", "", "2026-01", 1); return err },
		"read month":  func() error { _, err := r.Statement("ws", "tx", "", 1); return err },
		"read zero":   func() error { _, err := r.Statement("ws", "tx", "2026-01", 0); return err },
		"current ws":  func() error { _, err := r.Current("", "tx", "2026-01"); return err },
		"current id":  func() error { _, err := r.Current("ws", "", "2026-01"); return err },
		"revs ws":     func() error { _, err := r.Revisions("", "tx", "2026-01"); return err },
		"revs month":  func() error { _, err := r.Revisions("ws", "tx", ""); return err },
		"settle ws":   func() error { _, err := r.Settlement("", "tx", "s"); return err },
		"settle id":   func() error { _, err := r.Settlement("ws", "", "s"); return err },
		"tx ws":       func() error { _, err := r.Transaction("", "tx"); return err },
		"approve ws":  func() error { _, err := r.ApprovePolicy("", "p", "1", "a"); return err },
		"approve id":  func() error { _, err := r.ApprovePolicy("ws", "", "1", "a"); return err },
		"approve v":   func() error { _, err := r.ApprovePolicy("ws", "p", "", "a"); return err },
		"for ws":      func() error { _, err := r.PolicyFor("", "tx", "2026-01"); return err },
		"for tx":      func() error { _, err := r.PolicyFor("ws", "", "2026-01"); return err },
		"state ws":    func() error { _, err := r.PolicyState("", "p", "1"); return err },
		"state v":     func() error { _, err := r.PolicyState("ws", "p", ""); return err },
		"agreement p": func() error { _, err := r.AgreementCharges("ws", "aws", "", "2026-01"); return err },
	} {
		err := call()
		above := strings.Contains(name, "above") || strings.Contains(name, " ws") || strings.Contains(name, " id") || strings.Contains(name, " month") ||
			strings.Contains(name, "zero") || strings.Contains(name, " v") || strings.Contains(name, " tx") || strings.Contains(name, " p")
		if above != (kind(err) == "invalid") {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestSettlementStatesNeverMoveBack(t *testing.T) {
	r := open(t)
	record(t, r, deal("tx", RoleSeller, ChannelMarketplace, TypePublic, CollectorMarketplace))
	expected := SettlementFacts{State: SettlementExpected, Currency: "USD", Gross: "1", Fee: "0", Refund: "0", Tax: "0", Hold: "0", Net: "1"}
	reported := expected
	reported.State, reported.Report = SettlementReported, "rep"
	paid := reported
	paid.State, paid.Payout, paid.Received = SettlementPaid, "pay", "1"
	r.RecordSettlement("ws", "tx", "s1", 1, reported)
	if _, err := r.RecordSettlement("ws", "tx", "s1", 2, expected); kind(err) != "rejected" {
		t.Fatalf("reported to expected: %v", err)
	}
	if _, err := r.RecordSettlement("ws", "tx", "s1", 2, reported); err != nil {
		t.Fatalf("reported again: %v", err)
	}
	r.RecordSettlement("ws", "tx", "s1", 3, paid)
	q := open(t)
	record(t, q, deal("tx", RoleSeller, ChannelMarketplace, TypePublic, CollectorMarketplace))
	q.RecordSettlement("ws", "tx", "s1", 1, paid)
	if _, err := q.RecordSettlement("ws", "tx", "s1", 2, reported); kind(err) != "rejected" {
		t.Fatalf("paid to reported: %v", err)
	}
}

func TestPolicyBoundaries(t *testing.T) {
	r := open(t)
	record(t, r, buyerPrivate("tx"))
	late := terms("a", "1", "tx", "0", []string{KindSoftware}, Excluded)
	late.First = "2026-06"
	approve(t, r, late)
	before := terms("b", "1", "tx", "0", []string{KindSoftware}, Excluded)
	before.Last = "2026-05"
	approve(t, r, before)
	touching := terms("c", "1", "tx", "0", []string{KindSoftware}, Excluded)
	touching.First, touching.Last = "2026-06", "2026-06"
	if _, err := r.RecordPolicy("ws", touching, "author-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ApprovePolicy("ws", "c", "1", "approver-1"); kind(err) != "rejected" {
		t.Fatalf("an overlap in its last month: %v", err)
	}
	same := terms("a", "2", "tx", "0", []string{KindSoftware}, Excluded)
	same.First = "2026-06"
	if _, err := r.RecordPolicy("ws", same, "author-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ApprovePolicy("ws", "a", "2", "approver-1"); kind(err) != "rejected" {
		t.Fatalf("a version with the same first month: %v", err)
	}
	next := terms("a", "3", "tx", "0", []string{KindSoftware}, Excluded)
	next.First = "2026-11"
	approve(t, r, next)
	if s, _ := r.PolicyState("ws", "a", "1"); s.EffectiveLast != "2026-10" {
		t.Fatalf("the month before November: %+v", s)
	}
	for month, want := range map[string]string{"2026-05": "b", "2026-06": "a", "2026-10": "a", "2026-11": "a"} {
		if p := policyFor(t, r, "tx", month); p.terms.ID != want {
			t.Errorf("%s: %s", month, p.terms.ID)
		}
	}
	if p := policyFor(t, r, "tx", "2026-10"); p.terms.Version != "1" {
		t.Fatal("the effective last month")
	}
	if p := policyFor(t, r, "tx", "2026-01"); p.terms.ID != "b" {
		t.Fatal("the first month")
	}
}

func TestRatesAtTheirScale(t *testing.T) {
	wide, err := decimal.NewProfile(decimal.ProfileSpec{ID: "wide", Revision: "1", InputIntegerDigits: 14, InputScale: 12, ResultIntegerDigits: 16, ResultScale: 12, IntermediateDigits: 80})
	if err != nil {
		t.Fatal(err)
	}
	r, _ := NewRegistry(wide, testLimits(), time.Now)
	r.RecordTransaction("ws", buyerPrivate("tx"))
	ok := terms("p", "1", "tx", "0.12345678", []string{KindSoftware}, Excluded)
	if _, err := r.RecordPolicy("ws", ok, "a"); err != nil {
		t.Fatalf("eight decimals: %v", err)
	}
	nine := terms("q", "1", "tx", "0.123456789", []string{KindSoftware}, Excluded)
	if _, err := r.RecordPolicy("ws", nine, "a"); kind(err) != "invalid" {
		t.Fatalf("nine decimals: %v", err)
	}
}

func TestDigitsOfTheTotalAndTheChargedTotal(t *testing.T) {
	tiny := Limits{Transactions: 2, Revisions: 1, Lines: 2, Recognitions: 1, Settlements: 1, Policies: 2, Digits: 3}
	run := func(charged []string, rate string) error {
		r, _ := NewRegistry(testProfile(t), tiny, time.Now)
		r.RecordTransaction("ws", deal("tx", RoleBuyer, ChannelMarketplace, TypePublic, CollectorMSP))
		facts := Facts{}
		for index, amount := range charged {
			facts.Lines = append(facts.Lines, line(string(rune('a'+index)), KindSoftware, amount, notApplied()))
		}
		r.RecordStatement("ws", "tx", "2026-03", 1, facts)
		r.Seal("ws", "tx", "2026-03", 1)
		s, _ := r.Statement("ws", "tx", "2026-03", 1)
		r.RecordPolicy("ws", terms("p", "1", "tx", rate, []string{KindSoftware}, Excluded), "a")
		r.ApprovePolicy("ws", "p", "1", "b")
		p, _ := r.PolicyFor("ws", "tx", "2026-03")
		_, err := ComputeCharge(s, p)
		return err
	}
	if err := run([]string{"998", "1"}, "0"); err != nil {
		t.Fatalf("three digits: %v", err)
	}
	if err := run([]string{"999", "1"}, "0.5"); kind(err) != "limit" {
		t.Fatalf("a charged total beyond the digits: %v", err)
	}
}

func TestGroupsInOrder(t *testing.T) {
	for round := 0; round < 30; round++ {
		r := open(t)
		for index, currency := range []string{"USD", "EUR", "KRW", "JPY", "AUD"} {
			tx := deal(string(rune('a'+index)), RoleBuyer, ChannelMarketplace, TypePublic, CollectorMSP)
			if index == 4 {
				tx = deal("e", RoleSeller, ChannelMarketplace, TypePublic, CollectorMarketplace)
			}
			tx.Term.Currency = currency
			record(t, r, tx)
			write(t, r, tx.ID, "2026-03", 1, Facts{Lines: []Line{line("src-"+tx.ID, KindOther, "1", unknown())}})
			if index != 2 {
				sealed(t, r, tx.ID, "2026-03", 1)
			}
		}
		v, _ := r.AgreementCharges("ws", "aws", "agr-1", "2026-03")
		got := []string{}
		for _, g := range v.Groups {
			got = append(got, g.Role+"/"+g.Currency)
		}
		want := []string{"buyer/EUR", "buyer/JPY", "buyer/USD", "seller/AUD"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("groups: %v", got)
		}
		ids := []string{}
		for _, e := range v.Entries {
			ids = append(ids, e.Transaction)
		}
		if strings.Join(ids, "") != "abcde" {
			t.Fatalf("entries: %v", ids)
		}
	}
}
