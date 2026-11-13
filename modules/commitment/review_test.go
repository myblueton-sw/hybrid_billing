package commitment

import (
	"testing"
	"time"
)

func TestCustomerBearingOnlyUnusedCost(t *testing.T) {
	_, p := example(t)
	if got := sell(t, p, bear(BearCustomer, "C"), share("A", "0.5"), share("B", "0.5"), share("C", "0.5")); got != "A=80 B=37.5 C=15 | total 132.5 cost 100 recovered 0 result 32.5 below false priced true" {
		t.Fatalf("savings share: %s", got)
	}
	if got := sell(t, p, bear(BearCustomer, "C"), costPlus("A", "0.1", true), costPlus("B", "0.1", true), costPlus("C", "0.1", true)); got != "A=66 B=27.5 C=16.5 | total 110 cost 100 recovered 0 result 10 below false priced true" {
		t.Fatalf("cost plus: %s", got)
	}
	r := open(t)
	record(t, r, commitmentOf("c1", PaymentNone, "0", RecoveryCharged))
	write(t, r, "c1", "2026-01", 1, "60", "0", used("l1", "acct-a", "60", "0", "100"))
	full := sealed(t, r, "c1", "2026-01", 1)
	a, _ := Allocate(full, accounts(), bear(BearCustomer, "C"))
	if v := a.View(); len(v.Customers) != 1 || v.Customers[0].Customer != "A" {
		t.Fatalf("no unused cost leaves the named customer out: %+v", v.Customers)
	}
}

func TestRecoveredWithUnusedBorne(t *testing.T) {
	r := open(t)
	record(t, r, commitmentOf("c1", PaymentAll, "1200", RecoveryRecovered))
	write(t, r, "c1", "2026-03", 1, "10", "90", used("l1", "acct-a", "5", "55", "100"), used("l2", "acct-b", "2", "23", "50"), unused("l3", "3", "12"))
	p := sealed(t, r, "c1", "2026-03", 1)
	if got := sell(t, p, bear(BearCustomer, "A"), costPlus("A", "0.1", true), costPlus("B", "0.1", true)); got != "A=8.8 B=2.2 | total 11 cost 10 recovered 90 result 1 below false priced true" {
		t.Fatalf("a customer bearing unused cost: %s", got)
	}
	a, _ := Allocate(p, accounts(), bear(BearProRataUsed, ""))
	v := a.View()
	if v.Customers[0].UnusedRecurring != "2.12" || v.Customers[1].UnusedRecurring != "0.88" || v.Customers[0].UnusedAmortized != "8.47" || v.Customers[1].UnusedAmortized != "3.53" {
		t.Fatalf("pro rata: %+v", v.Customers)
	}
	s, _ := Sell(a, []SalePolicy{costPlus("A", "0.1", true), costPlus("B", "0.1", true)})
	if got := charges(s); got != "A=7.83 B=3.17 | total 11 cost 10 recovered 90 result 1 below false priced true" || s.Charges[0].Exact != "7.832" {
		t.Fatalf("pro rata sold: %s %s", got, s.Charges[0].Exact)
	}
	if s.ProfileID != "commit-test" || s.ProfileRevision != "1" {
		t.Fatal("profile in the sale")
	}
}

func TestMovedLineInTheOtherOrderAndSupersededRevisions(t *testing.T) {
	r := open(t)
	record(t, r, commitmentOf("c1", PaymentNone, "0", RecoveryCharged))
	write(t, r, "c1", "2026-01", 1, "1", "0", used("x", "acct-a", "1", "0", "2"))
	seal(t, r, "c1", "2026-01", 1)
	write(t, r, "c1", "2026-01", 2, "0", "0")
	if _, err := r.Revision("ws", "c1", "2026-01", 1); kind(err) != "rejected" {
		t.Fatal("a revision below a written one")
	}
	seal(t, r, "c1", "2026-01", 2)
	write(t, r, "c1", "2026-02", 1, "1", "0", used("x", "acct-a", "1", "0", "2"))
	if got := reasons(seal(t, r, "c1", "2026-02", 1)); got != "sealed:" {
		t.Fatalf("the new period after the old period's correction: %s", got)
	}
	write(t, r, "c1", "2026-03", 1, "0", "0")
	write(t, r, "c1", "2026-03", 2, "0", "0")
	if _, err := r.Revision("ws", "c1", "2026-03", 1); kind(err) != "rejected" {
		t.Fatal("a superseded revision")
	}
}

func TestTheTenantBoundary(t *testing.T) {
	r := open(t)
	record(t, r, commitmentOf("c1", PaymentNone, "0", RecoveryCharged))
	write(t, r, "c1", "2026-01", 1, "1", "0", used("x", "acct-a", "1", "0", "2"))
	seal(t, r, "c1", "2026-01", 1)
	if _, err := r.Seal("other", "c1", "2026-01", 1); kind(err) != "not_found" {
		t.Fatal("seal")
	}
	if _, err := r.Current("other", "c1", "2026-01"); kind(err) != "not_found" {
		t.Fatal("current")
	}
	if _, err := r.Revision("other", "c1", "2026-01", 1); kind(err) != "not_found" {
		t.Fatal("revision")
	}
	if _, err := r.RecordCommitment("other", commitmentOf("c1", PaymentNone, "0", RecoveryCharged)); err != nil {
		t.Fatal(err)
	}
	if _, err := r.RecordPeriod("other", "c1", "2026-02", 1, Facts{Recurring: "1", Amortized: "0", Lines: []Line{used("x", "acct-a", "1", "0", "2")}}); err != nil {
		t.Fatal(err)
	}
	if state, _ := r.Seal("other", "c1", "2026-02", 1); state.State != StateSealed {
		t.Fatalf("a source of another workspace: %+v", state)
	}
}

func TestLimitsOfThePureFunctions(t *testing.T) {
	_, p := example(t)
	five := Attribution{ID: "attr", Version: "1", Mappings: []Mapping{{"a1", "A"}, {"a2", "A"}, {"a3", "B"}, {"a4", "B"}, {"a5", "B"}}}
	if _, err := Allocate(p, five, bear(BearMSP, "")); kind(err) != "limit" {
		t.Fatalf("too many mappings: %v", err)
	}
	four := Attribution{ID: "attr", Version: "1", Mappings: []Mapping{{"acct-a", "A"}, {"acct-b", "B"}}}
	r := open(t)
	record(t, r, commitmentOf("c1", PaymentNone, "0", RecoveryCharged))
	write(t, r, "c1", "2026-01", 1, "5", "0", used("1", "acct-a", "1", "0", "1"), used("2", "acct-b", "1", "0", "1"), used("3", "acct-c", "1", "0", "1"), used("4", "acct-d", "1", "0", "1"), unused("5", "1", "0"))
	q := sealed(t, r, "c1", "2026-01", 1)
	four.Mappings = append(four.Mappings, Mapping{"acct-c", "C"}, Mapping{"acct-d", "D"})
	if _, err := Allocate(q, four, bear(BearCustomer, "E")); kind(err) != "limit" {
		t.Fatalf("five customers: %v", err)
	}
	limits := testLimits()
	limits.Digits = 2
	small, _ := NewRegistry(testProfile(t), limits, func() time.Time { return start })
	record(t, small, commitmentOf("c1", PaymentNone, "0", RecoveryCharged))
	write(t, small, "c1", "2026-03", 1, "100", "0", used("l1", "acct-a", "60", "0", "100"), used("l2", "acct-b", "25", "0", "50"), unused("l3", "15", "0"))
	big := sealed(t, small, "c1", "2026-03", 1)
	if _, err := Allocate(big, accounts(), bear(BearMSP, "")); kind(err) != "limit" {
		t.Fatalf("digits in an allocation: %v", err)
	}
	limits.Digits = 3
	middle, _ := NewRegistry(testProfile(t), limits, func() time.Time { return start })
	record(t, middle, commitmentOf("c1", PaymentNone, "0", RecoveryCharged))
	write(t, middle, "c1", "2026-03", 1, "100", "0", used("l1", "acct-a", "60", "0", "100"), used("l2", "acct-b", "25", "0", "50"), unused("l3", "15", "0"))
	m := sealed(t, middle, "c1", "2026-03", 1)
	a, err := Allocate(m, accounts(), bear(BearMSP, ""))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Sell(a, []SalePolicy{costPlus("A", "0.123", false), costPlus("B", "0", false)}); kind(err) != "limit" {
		t.Fatalf("digits in a sale: %v", err)
	}
}

func TestThePeriodDigestCoversEveryField(t *testing.T) {
	c := commitmentOf("c1", PaymentPartial, "10", RecoveryCharged)
	facts := Facts{Recurring: "3", Amortized: "1", Lines: []Line{used("a", "acct-a", "1", "1", "5"), unused("b", "2", "0")}}
	base := periodDigest("ws", c, "2026-01", 1, facts)
	seen := map[string]string{base: "base"}
	variants := map[string]func(*Commitment, *Facts, *string, *int, *string){
		"workspace":            func(c *Commitment, f *Facts, m *string, n *int, w *string) { *w = "ws2" },
		"provider":             func(c *Commitment, f *Facts, m *string, n *int, w *string) { c.Provider = "gcp" },
		"payer":                func(c *Commitment, f *Facts, m *string, n *int, w *string) { c.PayerAccount = "p2" },
		"kind":                 func(c *Commitment, f *Facts, m *string, n *int, w *string) { c.Kind = KindReserved },
		"currency":             func(c *Commitment, f *Facts, m *string, n *int, w *string) { c.Currency = "EUR" },
		"first":                func(c *Commitment, f *Facts, m *string, n *int, w *string) { c.First = "2026-02" },
		"last":                 func(c *Commitment, f *Facts, m *string, n *int, w *string) { c.Last = "2027-12" },
		"payment":              func(c *Commitment, f *Facts, m *string, n *int, w *string) { c.Payment = PaymentAll },
		"prepaid":              func(c *Commitment, f *Facts, m *string, n *int, w *string) { c.Prepaid = "11" },
		"quantum":              func(c *Commitment, f *Facts, m *string, n *int, w *string) { c.Quantum = "1" },
		"recovery":             func(c *Commitment, f *Facts, m *string, n *int, w *string) { c.Recovery = RecoveryRecovered },
		"month":                func(c *Commitment, f *Facts, m *string, n *int, w *string) { *m = "2026-02" },
		"revision":             func(c *Commitment, f *Facts, m *string, n *int, w *string) { *n = 2 },
		"recurring":            func(c *Commitment, f *Facts, m *string, n *int, w *string) { f.Recurring = "4" },
		"amortized":            func(c *Commitment, f *Facts, m *string, n *int, w *string) { f.Amortized = "2" },
		"source":               func(c *Commitment, f *Facts, m *string, n *int, w *string) { f.Lines[0].Source = "z" },
		"line kind":            func(c *Commitment, f *Facts, m *string, n *int, w *string) { f.Lines[1].Kind = LineUsed },
		"account":              func(c *Commitment, f *Facts, m *string, n *int, w *string) { f.Lines[0].Account = "acct-b" },
		"line rec":             func(c *Commitment, f *Facts, m *string, n *int, w *string) { f.Lines[0].Recurring = "2" },
		"line am":              func(c *Commitment, f *Facts, m *string, n *int, w *string) { f.Lines[0].Amortized = "0" },
		"list":                 func(c *Commitment, f *Facts, m *string, n *int, w *string) { f.Lines[0].List = "6" },
		"commitment of a line": func(c *Commitment, f *Facts, m *string, n *int, w *string) { f.Lines[0].Commitment = "c2" },
		"a byte moved": func(c *Commitment, f *Facts, m *string, n *int, w *string) {
			f.Lines[0].Source, f.Lines[0].Kind = "au", "sed"
		},
	}
	for name, change := range variants {
		cc, ff, month, number, ws := c, Facts{Recurring: facts.Recurring, Amortized: facts.Amortized, Lines: append([]Line{}, facts.Lines...)}, "2026-01", 1, "ws"
		change(&cc, &ff, &month, &number, &ws)
		got := periodDigest(ws, cc, month, number, ff)
		if earlier, ok := seen[got]; ok {
			t.Errorf("%s has the digest of %s", name, earlier)
		}
		seen[got] = name
	}
}
