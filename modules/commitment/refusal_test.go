package commitment

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func TestNewRegistry(t *testing.T) {
	clock := func() time.Time { return start }
	if _, err := NewRegistry(decimal.Profile{}, testLimits(), clock); kind(err) != "invalid" {
		t.Fatal("profile")
	}
	if _, err := NewRegistry(testProfile(t), testLimits(), nil); kind(err) != "invalid" {
		t.Fatal("clock")
	}
	for name, change := range map[string]func(*Limits){
		"commitments": func(l *Limits) { l.Commitments = 0 }, "commitments high": func(l *Limits) { l.Commitments = 100001 },
		"revisions": func(l *Limits) { l.Revisions = 0 }, "revisions high": func(l *Limits) { l.Revisions = 1001 },
		"lines": func(l *Limits) { l.Lines = 0 }, "lines high": func(l *Limits) { l.Lines = 100001 },
		"customers": func(l *Limits) { l.Customers = 0 }, "customers high": func(l *Limits) { l.Customers = 10001 },
		"digits": func(l *Limits) { l.Digits = 0 }, "digits high": func(l *Limits) { l.Digits = 10001 },
	} {
		limits := testLimits()
		change(&limits)
		if _, err := NewRegistry(testProfile(t), limits, clock); kind(err) != "invalid" {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := NewRegistry(testProfile(t), Limits{Commitments: 100000, Revisions: 1000, Lines: 100000, Customers: 10000, Digits: 10000}, clock); err != nil {
		t.Fatal(err)
	}
}

func TestRecordCommitment(t *testing.T) {
	r := open(t)
	good := commitmentOf("c1", PaymentPartial, "12.50", RecoveryCharged)
	if c, err := r.RecordCommitment("ws", good); err != nil || c.Prepaid != "12.5" {
		t.Fatalf("recorded: %v %+v", err, c)
	}
	if _, err := r.RecordCommitment("ws", good); err != nil {
		t.Fatal("repeat")
	}
	change := func(apply func(*Commitment)) Commitment {
		c := good
		apply(&c)
		return c
	}
	for name, c := range map[string]struct {
		in   Commitment
		want string
	}{
		"id":                        {change(func(c *Commitment) { c.ID = "" }), "invalid"},
		"provider":                  {change(func(c *Commitment) { c.Provider = "a ws" }), "invalid"},
		"payer":                     {change(func(c *Commitment) { c.PayerAccount = "" }), "invalid"},
		"kind":                      {change(func(c *Commitment) { c.Kind = "spot" }), "invalid"},
		"currency":                  {change(func(c *Commitment) { c.Currency = "usd" }), "invalid"},
		"first":                     {change(func(c *Commitment) { c.First = "2026-13" }), "invalid"},
		"last before first":         {change(func(c *Commitment) { c.Last = "2025-12" }), "invalid"},
		"payment":                   {change(func(c *Commitment) { c.Payment = "monthly" }), "invalid"},
		"prepaid negative":          {change(func(c *Commitment) { c.Prepaid = "-1" }), "invalid"},
		"prepaid zero upfront":      {change(func(c *Commitment) { c.Prepaid = "0" }), "invalid"},
		"prepaid without upfront":   {change(func(c *Commitment) { c.Payment = PaymentNone }), "invalid"},
		"recovered without upfront": {change(func(c *Commitment) { c.Payment, c.Prepaid, c.Recovery = PaymentNone, "0", RecoveryRecovered }), "invalid"},
		"quantum":                   {change(func(c *Commitment) { c.Quantum = "0.5" }), "invalid"},
		"recovery":                  {change(func(c *Commitment) { c.Recovery = "partly" }), "invalid"},
		"another recovery":          {change(func(c *Commitment) { c.Recovery = RecoveryRecovered }), "conflict"},
		"another prepaid":           {change(func(c *Commitment) { c.Prepaid = "13" }), "conflict"},
	} {
		if _, err := r.RecordCommitment("ws", c.in); kind(err) != c.want {
			t.Errorf("%s: %v", name, err)
		}
	}
	for index := 2; index <= 6; index++ {
		record(t, r, commitmentOf("c"+string(rune('0'+index)), PaymentNone, "0", RecoveryCharged))
	}
	if _, err := r.RecordCommitment("ws", commitmentOf("c7", PaymentNone, "0", RecoveryCharged)); kind(err) != "limit" {
		t.Fatalf("a full workspace: %v", err)
	}
	if _, err := r.RecordCommitment("other", commitmentOf("c7", PaymentNone, "0", RecoveryCharged)); err != nil {
		t.Fatal("another workspace")
	}
	want := good
	want.Prepaid = "12.5"
	if c, err := r.Commitment("ws", "c1"); err != nil || c != want {
		t.Fatal("read")
	}
	if _, err := r.Commitment("other", "c1"); kind(err) != "not_found" {
		t.Fatal("tenant boundary")
	}
}

func TestRecordPeriodAndOrder(t *testing.T) {
	r := open(t)
	record(t, r, commitmentOf("c1", PaymentNone, "0", RecoveryCharged))
	good := Facts{Recurring: "1.50", Amortized: "0", Lines: []Line{used("b", "acct-a", "1", "0", "2"), unused("a", "0.5", "0")}}
	if state, err := r.RecordPeriod("ws", "c1", "2026-01", 1, good); err != nil || state.State != StateWritten || !state.WrittenAt.Equal(start) {
		t.Fatalf("written: %v %+v", err, state)
	}
	nine := Facts{Recurring: "0", Amortized: "0"}
	for index := 0; index < 9; index++ {
		nine.Lines = append(nine.Lines, Line{})
	}
	change := func(apply func(*Facts)) Facts {
		f := Facts{Recurring: good.Recurring, Amortized: good.Amortized, Lines: append([]Line{}, good.Lines...)}
		apply(&f)
		return f
	}
	cases := []struct {
		name      string
		workspace string
		month     string
		number    int
		facts     Facts
		want      string
	}{
		{"lines before their form", "", "2026-01", 1, nine, "limit"},
		{"workspace", "", "2026-01", 1, good, "invalid"},
		{"month", "ws", "2026-1", 1, good, "invalid"},
		{"revision zero", "ws", "2026-01", 0, good, "invalid"},
		{"revision high", "ws", "2026-01", 1001, good, "invalid"},
		{"cost", "ws", "2026-01", 2, change(func(f *Facts) { f.Recurring = "-1" }), "invalid"},
		{"source twice", "ws", "2026-01", 2, change(func(f *Facts) { f.Lines[1].Source = "b" }), "invalid"},
		{"kind", "ws", "2026-01", 2, change(func(f *Facts) { f.Lines[0].Kind = "credit" }), "invalid"},
		{"used without account", "ws", "2026-01", 2, change(func(f *Facts) { f.Lines[0].Account = "" }), "invalid"},
		{"used without list", "ws", "2026-01", 2, change(func(f *Facts) { f.Lines[0].List = "" }), "invalid"},
		{"unused with account", "ws", "2026-01", 2, change(func(f *Facts) { f.Lines[1].Account = "acct-a" }), "invalid"},
		{"unused with list", "ws", "2026-01", 2, change(func(f *Facts) { f.Lines[1].List = "1" }), "invalid"},
		{"line commitment", "ws", "2026-01", 2, change(func(f *Facts) { f.Lines[0].Commitment = "" }), "invalid"},
		{"a period without lines", "ws", "2026-04", 1, Facts{Recurring: "0", Amortized: "0"}, "ok"},
		{"unknown commitment", "ws", "2026-01", 1, Facts{Recurring: "0", Amortized: "0"}, "ok"},
		{"another workspace", "other", "2026-01", 1, good, "not_found"},
		{"a new period not at one", "ws", "2026-02", 2, good, "rejected"},
		{"other facts", "ws", "2026-01", 1, change(func(f *Facts) { f.Recurring = "1.5"; f.Lines[0].List = "3" }), "conflict"},
		{"the same facts in another order", "ws", "2026-01", 1, Facts{Recurring: "1.5", Amortized: "0.00", Lines: []Line{good.Lines[1], good.Lines[0]}}, "ok"},
		{"a gap", "ws", "2026-01", 3, good, "rejected"},
		{"a month outside", "ws", "2027-01", 1, good, "rejected"},
		{"amortized without upfront", "ws", "2026-03", 1, change(func(f *Facts) { f.Amortized = "1" }), "rejected"},
		{"amortized line without upfront", "ws", "2026-03", 1, change(func(f *Facts) { f.Lines[1].Amortized = "0.5" }), "rejected"},
	}
	for _, c := range cases {
		id := "c1"
		if c.name == "unknown commitment" {
			id = "none"
		}
		_, err := r.RecordPeriod(c.workspace, id, c.month, c.number, c.facts)
		want := c.want
		if c.name == "unknown commitment" {
			want = "not_found"
		}
		if kind(err) != want {
			t.Errorf("%s: %v", c.name, err)
		}
	}
	for number := 2; number <= 3; number++ {
		write(t, r, "c1", "2026-01", number, "0", "0")
	}
	if _, err := r.RecordPeriod("ws", "c1", "2026-01", 4, good); kind(err) != "limit" {
		t.Fatalf("a full period: %v", err)
	}
	for name, c := range map[string][4]string{"workspace": {"", "c1", "2026-01", "invalid"}, "month": {"ws", "c1", "x", "invalid"}, "commitment": {"ws", "none", "2026-01", "not_found"}, "period": {"ws", "c1", "2026-09", "not_found"}} {
		if _, err := r.Seal(c[0], c[1], c[2], 1); kind(err) != c[3] {
			t.Errorf("seal %s: %v", name, err)
		}
		if _, err := r.Current(c[0], c[1], c[2]); kind(err) != c[3] {
			t.Errorf("current %s: %v", name, err)
		}
		if _, err := r.Revision(c[0], c[1], c[2], 1); kind(err) != c[3] {
			t.Errorf("revision %s: %v", name, err)
		}
		if _, err := r.Revisions(c[0], c[1], c[2]); kind(err) != c[3] {
			t.Errorf("revisions %s: %v", name, err)
		}
	}
	if _, err := r.Seal("ws", "c1", "2026-01", 9); kind(err) != "not_found" {
		t.Fatal("seal of no revision")
	}
}

func printed(fields ...string) string {
	sum := sha256.New()
	for _, field := range fields {
		sum.Write([]byte{0, 0, 0, 0, 0, 0, 0, byte(len(field))})
		sum.Write([]byte(field))
	}
	return hex.EncodeToString(sum.Sum(nil))
}

func TestDigests(t *testing.T) {
	r, p := example(t)
	want := printed("commitment_period_v1", "ws", "c1", "aws", "payer-1", KindSavings, "USD", "2026-01", "2026-12", PaymentNone, "0", "0.01", RecoveryCharged,
		"2026-03", "1", "100", "0", "3",
		"l1", "used", "c1", "acct-a", "60", "0", "100",
		"l2", "used", "c1", "acct-b", "25", "0", "50",
		"l3", "unused", "c1", "", "15", "0", "")
	if p.View().Digest != want {
		t.Fatalf("period digest: %s", p.View().Digest)
	}
	if cur, _ := r.Current("ws", "c1", "2026-03"); cur.Digest != want {
		t.Fatal("current digest")
	}
	a, _ := Allocate(p, accounts(), bear(BearMSP, ""))
	v := a.View()
	if v.AttributionDigest != printed("commitment_attribution_v1", "attr", "1", "2", "acct-a", "A", "acct-b", "B") || v.PolicyDigest != printed("commitment_allocation_policy_v1", "alloc", "1", "msp", "") {
		t.Fatal("input digests")
	}
	reordered := Attribution{ID: "attr", Version: "1", Mappings: []Mapping{{Account: "acct-b", Customer: "B"}, {Account: "acct-a", Customer: "A"}}}
	b, _ := Allocate(p, reordered, bear(BearMSP, ""))
	if b.View().Digest != v.Digest {
		t.Fatal("an attribution in another order")
	}
	seen := map[string]bool{v.Digest: true}
	for _, other := range []struct {
		attr   Attribution
		policy AllocationPolicy
	}{
		{Attribution{ID: "attr", Version: "2", Mappings: accounts().Mappings}, bear(BearMSP, "")},
		{Attribution{ID: "attr", Version: "1", Mappings: []Mapping{{Account: "acct-a", Customer: "B"}, {Account: "acct-b", Customer: "A"}}}, bear(BearMSP, "")},
		{accounts(), bear(BearUnattributed, "")},
		{accounts(), AllocationPolicy{ID: "alloc", Version: "2", Bearing: BearMSP}},
	} {
		c, _ := Allocate(p, other.attr, other.policy)
		if seen[c.View().Digest] {
			t.Errorf("a changed input kept the digest: %+v", other)
		}
		seen[c.View().Digest] = true
	}
	s1, _ := Sell(a, []SalePolicy{costPlus("A", "0.10", false), costPlus("B", "0.1", false)})
	s2, _ := Sell(a, []SalePolicy{costPlus("A", "0.1", false), costPlus("B", "0.1", false)})
	s3, _ := Sell(a, []SalePolicy{costPlus("A", "0.1", true), costPlus("B", "0.1", false)})
	if s1.Digest != s2.Digest || s1.Digest == s3.Digest || s1.Charges[0].PolicyDigest != printed("commitment_sale_policy_v1", "A", "sale-A", "1", "cost_plus", "0.1", "false", "", "half_even") {
		t.Fatal("sale digests")
	}
	view := p.View()
	view.Facts.Lines[0].Recurring = "999"
	if p.View().Facts.Lines[0].Recurring != "60" {
		t.Fatal("a view shares its lines")
	}
}

func TestSaleRefusals(t *testing.T) {
	_, p := example(t)
	a, _ := Allocate(p, accounts(), bear(BearMSP, ""))
	cases := map[string][]SalePolicy{
		"customer":            {{Customer: "", ID: "s", Version: "1", Method: MethodCostPlus, Fee: "0", Mode: "half_even"}},
		"unknown customer":    {costPlus("Z", "0.1", false)},
		"twice":               {costPlus("A", "0.1", false), costPlus("A", "0.2", false)},
		"method":              {{Customer: "A", ID: "s", Version: "1", Method: "markup", Fee: "0", Mode: "half_even"}},
		"mode":                {{Customer: "A", ID: "s", Version: "1", Method: MethodCostPlus, Fee: "0", Mode: "round"}},
		"fee one":             {costPlus("A", "1", false)},
		"fee negative":        {costPlus("A", "-0.1", false)},
		"fee text":            {costPlus("A", "1e-1", false)},
		"fee scale":           {costPlus("A", "0.123456789", false)},
		"fee with share":      {{Customer: "A", ID: "s", Version: "1", Method: MethodCostPlus, Fee: "0.1", Share: "0.5", Mode: "half_even"}},
		"share above one":     {share("A", "1.01")},
		"share with fee":      {{Customer: "A", ID: "s", Version: "1", Method: MethodSavingsShare, Share: "0.5", Fee: "0.1", Mode: "half_even"}},
		"share with fee flag": {{Customer: "A", ID: "s", Version: "1", Method: MethodSavingsShare, Share: "0.5", FeeOnUnused: true, Mode: "half_even"}},
	}
	for name, policies := range cases {
		if _, err := Sell(a, policies); kind(err) != "invalid" {
			t.Errorf("%s: %v", name, err)
		}
	}
	if s, err := Sell(a, []SalePolicy{share("A", "1"), share("B", "0")}); err != nil || charges(s) != "A=60 B=50 | total 110 cost 100 recovered 0 result 10 below false priced true" {
		t.Fatalf("shares at their bounds: %v %s", err, charges(s))
	}
	if s, err := Sell(a, []SalePolicy{costPlus("A", "0.99999999", false), costPlus("B", "0", false)}); err != nil || s.Charges[0].Amount != "120" {
		t.Fatalf("a fee just below one: %v %+v", err, s.Charges)
	}
	five := []SalePolicy{costPlus("A", "0", false), costPlus("B", "0", false), costPlus("C", "0", false), costPlus("D", "0", false), costPlus("E", "0", false)}
	if _, err := Sell(a, five); kind(err) != "limit" {
		t.Fatalf("too many policies: %v", err)
	}
}

func TestConcurrentSealsAgainstThePrepaidAmount(t *testing.T) {
	for round := 0; round < 100; round++ {
		r := open(t)
		record(t, r, commitmentOf("c1", PaymentAll, "150", RecoveryCharged))
		write(t, r, "c1", "2026-01", 1, "0", "100", used("a", "acct-a", "0", "100", "100"))
		write(t, r, "c1", "2026-02", 1, "0", "100", used("b", "acct-a", "0", "100", "100"))
		gate := make(chan struct{})
		var group sync.WaitGroup
		states := make([]RevisionState, 2)
		for index, month := range []string{"2026-01", "2026-02"} {
			group.Add(1)
			go func(index int, month string) {
				defer group.Done()
				<-gate
				states[index], _ = r.Seal("ws", "c1", month, 1)
			}(index, month)
		}
		close(gate)
		group.Wait()
		if states[0].State != StateHeld || states[1].State != StateHeld {
			t.Fatalf("both written periods count for each other: %+v", states)
		}
	}
}

func TestTheSourceHasNoFloatingPointOrComments(t *testing.T) {
	for _, file := range []string{"types.go", "registry.go", "period.go", "allocate.go", "sell.go"} {
		body := readSource(file)
		if body == "" {
			t.Fatalf("%s not read", file)
		}
		for _, word := range []string{"float", "Float", "big.Rat", "//", "/*", "1e"} {
			if strings.Contains(body, word) {
				t.Errorf("%s contains %q", file, word)
			}
		}
	}
}
