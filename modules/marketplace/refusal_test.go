package marketplace

import (
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func TestNewRegistry(t *testing.T) {
	good := testLimits()
	if _, err := NewRegistry(decimal.Profile{}, good, time.Now); kind(err) != "invalid" {
		t.Fatal("a profile")
	}
	if _, err := NewRegistry(testProfile(t), good, nil); kind(err) != "invalid" {
		t.Fatal("a clock")
	}
	bad := []func(*Limits){
		func(l *Limits) { l.Transactions = 0 }, func(l *Limits) { l.Transactions = 100001 },
		func(l *Limits) { l.Revisions = 0 }, func(l *Limits) { l.Revisions = 1001 },
		func(l *Limits) { l.Lines = 0 }, func(l *Limits) { l.Lines = 100001 },
		func(l *Limits) { l.Recognitions = 0 }, func(l *Limits) { l.Recognitions = 1001 },
		func(l *Limits) { l.Settlements = 0 }, func(l *Limits) { l.Settlements = 10001 },
		func(l *Limits) { l.Policies = 0 }, func(l *Limits) { l.Policies = 100001 },
		func(l *Limits) { l.Digits = 0 }, func(l *Limits) { l.Digits = 10001 },
	}
	for index, change := range bad {
		limits := good
		change(&limits)
		if _, err := NewRegistry(testProfile(t), limits, time.Now); kind(err) != "invalid" {
			t.Errorf("limit %d", index)
		}
	}
	top := Limits{Transactions: 100000, Revisions: 1000, Lines: 100000, Recognitions: 1000, Settlements: 10000, Policies: 100000, Digits: 10000}
	if _, err := NewRegistry(testProfile(t), top, time.Now); err != nil {
		t.Fatal("limits at their bounds")
	}
}

func TestRecordTransaction(t *testing.T) {
	r := open(t)
	good := buyerPrivate("tx-1")
	bad := map[string]func(*Transaction){
		"id":                    func(x *Transaction) { x.ID = "a b" },
		"provider":              func(x *Transaction) { x.Provider = "" },
		"agreement":             func(x *Transaction) { x.Agreement = strings.Repeat("a", 129) },
		"role":                  func(x *Transaction) { x.Role = "partner" },
		"channel":               func(x *Transaction) { x.Channel = "store" },
		"type":                  func(x *Transaction) { x.Type = "free" },
		"collector":             func(x *Transaction) { x.Collector = "bank" },
		"software provider":     func(x *Transaction) { x.Participants.SoftwareProvider = "" },
		"seller":                func(x *Transaction) { x.Participants.Seller = "" },
		"issuer":                func(x *Transaction) { x.Participants.Issuer = "" },
		"reseller of a buyer":   func(x *Transaction) { x.Participants.Reseller = "r" },
		"payee of a buyer":      func(x *Transaction) { x.Participants.Payee = "p" },
		"product":               func(x *Transaction) { x.References.Product = "" },
		"offer":                 func(x *Transaction) { x.References.Offer = "" },
		"subscription":          func(x *Transaction) { x.References.Subscription = "a/b" },
		"order":                 func(x *Transaction) { x.References.Order = "a b" },
		"accepted":              func(x *Transaction) { x.Term.Accepted = "2026-13" },
		"start before accepted": func(x *Transaction) { x.Term.Accepted = "2026-02" },
		"end before start":      func(x *Transaction) { x.Term.End = "2025-12" },
		"start":                 func(x *Transaction) { x.Term.Start = "1999-12" },
		"end":                   func(x *Transaction) { x.Term.End = "2100-01" },
		"currency":              func(x *Transaction) { x.Term.Currency = "usd" },
		"channel private buyer": func(x *Transaction) { x.Type = TypeChannelPrivate },
		"buyer collected afar":  func(x *Transaction) { x.Collector = CollectorMarketplace },
		"distributor collector": func(x *Transaction) {
			x.Role, x.Participants.Payee, x.Participants.Reseller, x.Collector = RoleReseller, "p", "r", CollectorDistributor
		},
		"marketplace collector": func(x *Transaction) {
			x.Role, x.Participants.Payee, x.Participants.Reseller, x.Channel, x.Collector = RoleReseller, "p", "r", ChannelDistributor, CollectorMarketplace
		},
		"reseller without name": func(x *Transaction) { x.Role, x.Participants.Payee = RoleReseller, "p" },
		"seller without payee":  func(x *Transaction) { x.Role = RoleSeller },
		"channel private direct": func(x *Transaction) {
			x.Role, x.Participants.Payee, x.Participants.Reseller, x.Channel, x.Type = RoleReseller, "p", "r", ChannelDirect, TypeChannelPrivate
		},
	}
	for name, change := range bad {
		tx := good
		change(&tx)
		if _, err := r.RecordTransaction("ws", tx); kind(err) != "invalid" {
			t.Errorf("%s: %v", name, err)
		}
	}
	for _, tx := range []Transaction{
		deal("ok-1", RoleReseller, ChannelDistributor, TypeChannelPrivate, CollectorDistributor),
		deal("ok-2", RoleSeller, ChannelDirect, TypePublic, CollectorMSP),
		deal("ok-3", RoleBuyer, ChannelDistributor, TypePublic, CollectorMSP),
	} {
		if _, err := r.RecordTransaction("ws", tx); err != nil {
			t.Errorf("%s: %v", tx.ID, err)
		}
	}
	full := good
	full.References.Subscription, full.References.Order = "sub-1", "ord-1"
	if _, err := r.RecordTransaction("bad ws", full); kind(err) != "invalid" {
		t.Fatal("workspace")
	}
	if _, err := r.RecordTransaction("ws", full); err != nil {
		t.Fatal(err)
	}
	if got, err := r.RecordTransaction("ws", full); err != nil || got != full {
		t.Fatal("a repeat")
	}
	changed := full
	changed.Collector = CollectorMSP
	changed.Term.End = "2026-11"
	if _, err := r.RecordTransaction("ws", changed); kind(err) != "conflict" {
		t.Fatal("a conflict")
	}
	if got, err := r.Transaction("ws", "tx-1"); err != nil || got != full {
		t.Fatal("read back")
	}
	for _, id := range []string{"t3", "t4"} {
		if _, err := r.RecordTransaction("ws", buyerPrivate(id)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := r.RecordTransaction("ws", buyerPrivate("t5")); kind(err) != "limit" {
		t.Fatalf("room: %v", err)
	}
	if _, err := r.RecordTransaction("ws", full); err != nil {
		t.Fatal("a repeat at the limit")
	}
	if _, err := r.Transaction("ws", "none"); kind(err) != "not_found" {
		t.Fatal("not found")
	}
	if _, err := r.Transaction("ws", ""); kind(err) != "invalid" {
		t.Fatal("invalid read")
	}
}

func TestRecordStatement(t *testing.T) {
	r := open(t)
	record(t, r, buyerPrivate("tx-1"))
	good := exampleFacts()
	bad := map[string]func(*Facts){
		"source":                   func(f *Facts) { f.Lines[0].Source = "" },
		"same source":              func(f *Facts) { f.Lines[1].Source = f.Lines[0].Source },
		"kind":                     func(f *Facts) { f.Lines[0].Kind = "fee" },
		"account":                  func(f *Facts) { f.Lines[0].Account = "" },
		"charged":                  func(f *Facts) { f.Lines[0].Charged = "-1" },
		"charged text":             func(f *Facts) { f.Lines[0].Charged = "8e2" },
		"charged scale":            func(f *Facts) { f.Lines[0].Charged = "0.123456789" },
		"charged size":             func(f *Facts) { f.Lines[0].Charged = strings.Repeat("0", 64) + "1" },
		"list":                     func(f *Facts) { f.Lines[0].List = "x" },
		"discount state":           func(f *Facts) { f.Lines[0].Discount.State = "partial" },
		"applied without amount":   func(f *Facts) { f.Lines[0].Discount.Amount = "" },
		"applied without evidence": func(f *Facts) { f.Lines[0].Discount.Evidence = "" },
		"evidence form":            func(f *Facts) { f.Lines[0].Discount.Evidence = strings.Repeat("A", 64) },
		"not applied with amount":  func(f *Facts) { f.Lines[1].Discount.Amount = "1" },
		"not applied no evidence":  func(f *Facts) { f.Lines[1].Discount.Evidence = "" },
		"unknown with amount":      func(f *Facts) { f.Lines[1].Discount = Benefit{State: Unknown, Amount: "0"} },
		"unknown with evidence":    func(f *Facts) { f.Lines[1].Discount = Benefit{State: Unknown, Evidence: evidenceA} },
		"program":                  func(f *Facts) { f.Recognitions[0].Program = "" },
		"same program":             func(f *Facts) { f.Recognitions = append(f.Recognitions, f.Recognitions[0]) },
		"recognition state":        func(f *Facts) { f.Recognitions[0].State = "drawn" },
		"recognition amount":       func(f *Facts) { f.Recognitions[0].Amount = "-800" },
		"unknown recognition 0":    func(f *Facts) { f.Recognitions[0] = Recognition{Program: "m", State: Unknown, Amount: "0"} },
		"recognition evidence":     func(f *Facts) { f.Recognitions[0].Evidence = "" },
		"not applied rec amount":   func(f *Facts) { f.Recognitions[0].State = NotApplied },
	}
	for name, change := range bad {
		facts := Facts{Lines: append([]Line{}, good.Lines...), Recognitions: append([]Recognition{}, good.Recognitions...)}
		change(&facts)
		if _, err := r.RecordStatement("ws", "tx-1", "2026-03", 1, facts); kind(err) != "invalid" {
			t.Errorf("%s: %v", name, err)
		}
	}
	for name, call := range map[string]func() error{
		"workspace": func() error { _, err := r.RecordStatement("", "tx-1", "2026-03", 1, good); return err },
		"id":        func() error { _, err := r.RecordStatement("ws", "", "2026-03", 1, good); return err },
		"month":     func() error { _, err := r.RecordStatement("ws", "tx-1", "2026-3", 1, good); return err },
		"zero":      func() error { _, err := r.RecordStatement("ws", "tx-1", "2026-03", 0, good); return err },
		"above":     func() error { _, err := r.RecordStatement("ws", "tx-1", "2026-03", 1001, good); return err },
	} {
		if err := call(); kind(err) != "invalid" {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := r.RecordStatement("ws", "none", "2026-03", 1, good); kind(err) != "not_found" {
		t.Fatal("no transaction")
	}
	if _, err := r.RecordStatement("ws", "tx-1", "2027-01", 1, good); kind(err) != "rejected" {
		t.Fatal("outside the term")
	}
	if _, err := r.RecordStatement("ws", "tx-1", "2026-03", 2, good); kind(err) != "rejected" {
		t.Fatal("beyond the next")
	}
	reordered := Facts{Lines: []Line{good.Lines[1], good.Lines[0]}, Recognitions: good.Recognitions}
	first, err := r.RecordStatement("ws", "tx-1", "2026-03", 1, reordered)
	if err != nil || first.State != StateWritten || !first.WrittenAt.Equal(start) {
		t.Fatalf("written: %v %+v", err, first)
	}
	again, err := r.RecordStatement("ws", "tx-1", "2026-03", 1, good)
	if err != nil || again.Digest != first.Digest {
		t.Fatal("a repeat in another order")
	}
	padded := Facts{Lines: append([]Line{}, good.Lines...), Recognitions: good.Recognitions}
	padded.Lines[0].Charged = "800.00"
	if _, err := r.RecordStatement("ws", "tx-1", "2026-03", 1, padded); err != nil {
		t.Fatal("canonical numbers repeat")
	}
	changed := Facts{Lines: append([]Line{}, good.Lines...), Recognitions: good.Recognitions}
	changed.Lines[0].Charged = "801"
	if _, err := r.RecordStatement("ws", "tx-1", "2026-03", 1, changed); kind(err) != "conflict" {
		t.Fatal("a conflict")
	}
	moreRecs := Facts{Lines: good.Lines, Recognitions: append(append([]Recognition{}, good.Recognitions...), Recognition{Program: "z", State: Unknown})}
	if _, err := r.RecordStatement("ws", "tx-1", "2026-03", 1, moreRecs); kind(err) != "conflict" {
		t.Fatal("a conflict in recognitions")
	}
	write(t, r, "tx-1", "2026-03", 2, changed)
	write(t, r, "tx-1", "2026-03", 3, good)
	if _, err := r.RecordStatement("ws", "tx-1", "2026-03", 4, good); kind(err) != "limit" {
		t.Fatal("revisions")
	}
	many := Facts{}
	for index := 0; index < 9; index++ {
		many.Lines = append(many.Lines, Line{Source: "bad source"})
	}
	if _, err := r.RecordStatement("ws", "tx-1", "2026-04", 1, many); kind(err) != "limit" {
		t.Fatal("lines before form")
	}
	recs := Facts{Recognitions: make([]Recognition, 4)}
	if _, err := r.RecordStatement("ws", "tx-1", "2026-04", 1, recs); kind(err) != "limit" {
		t.Fatal("recognitions before form")
	}
	for name, call := range map[string]func() error{
		"seal invalid":       func() error { _, err := r.Seal("ws", "tx-1", "2026-03", 0); return err },
		"current invalid":    func() error { _, err := r.Current("ws", "tx-1", "03"); return err },
		"revisions invalid":  func() error { _, err := r.Revisions("ws", "", "2026-03"); return err },
		"statement invalid":  func() error { _, err := r.Statement("ws", "tx-1", "2026-03", 1001); return err },
		"settlement invalid": func() error { _, err := r.Settlement("ws", "tx-1", "a b"); return err },
	} {
		if err := call(); kind(err) != "invalid" {
			t.Errorf("%s: %v", name, err)
		}
	}
	for name, call := range map[string]func() error{
		"seal month":         func() error { _, err := r.Seal("ws", "tx-1", "2026-05", 1); return err },
		"seal revision":      func() error { _, err := r.Seal("ws", "tx-1", "2026-03", 4); return err },
		"seal transaction":   func() error { _, err := r.Seal("ws", "none", "2026-03", 1); return err },
		"current month":      func() error { _, err := r.Current("ws", "tx-1", "2026-05"); return err },
		"current tx":         func() error { _, err := r.Current("ws", "none", "2026-03"); return err },
		"revisions month":    func() error { _, err := r.Revisions("ws", "tx-1", "2026-05"); return err },
		"revisions tx":       func() error { _, err := r.Revisions("ws", "none", "2026-03"); return err },
		"statement month":    func() error { _, err := r.Statement("ws", "tx-1", "2026-05", 1); return err },
		"statement revision": func() error { _, err := r.Statement("ws", "tx-1", "2026-03", 4); return err },
		"statement tx":       func() error { _, err := r.Statement("ws", "none", "2026-03", 1); return err },
		"settlement tx":      func() error { _, err := r.Settlement("ws", "none", "s"); return err },
	} {
		if err := call(); kind(err) != "not_found" {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := r.Statement("ws", "tx-1", "2026-03", 3); kind(err) != "rejected" {
		t.Fatal("an unsealed statement read")
	}
}

func TestPolicyForms(t *testing.T) {
	r := open(t)
	record(t, r, buyerPrivate("tx-1"))
	good := terms("pol", "1", "tx-1", "0.2", []string{KindSoftware}, Included)
	bad := map[string]func(*PolicyTerms){
		"id":                      func(p *PolicyTerms) { p.ID = "" },
		"version":                 func(p *PolicyTerms) { p.Version = "a b" },
		"customer":                func(p *PolicyTerms) { p.Customer = "" },
		"transaction":             func(p *PolicyTerms) { p.Transaction = "" },
		"first":                   func(p *PolicyTerms) { p.First = "2026" },
		"last":                    func(p *PolicyTerms) { p.Last = "2026-00" },
		"last before first":       func(p *PolicyTerms) { p.First, p.Last = "2026-05", "2026-04" },
		"rate one":                func(p *PolicyTerms) { p.Rate = "1" },
		"rate above one":          func(p *PolicyTerms) { p.Rate = "1.5" },
		"rate negative":           func(p *PolicyTerms) { p.Rate = "-0.1" },
		"rate scale":              func(p *PolicyTerms) { p.Rate = "0.123456789" },
		"rate text":               func(p *PolicyTerms) { p.Rate = "20%" },
		"scope empty":             func(p *PolicyTerms) { p.Scope = nil },
		"scope kind":              func(p *PolicyTerms) { p.Scope = []string{"fee"} },
		"scope twice":             func(p *PolicyTerms) { p.Scope = []string{KindOther, KindOther} },
		"source discounted":       func(p *PolicyTerms) { p.SourceDiscounted = "" },
		"included without basis":  func(p *PolicyTerms) { p.SourceBasis = "" },
		"excluded with basis":     func(p *PolicyTerms) { p.SourceDiscounted = Excluded },
		"included at rate zero":   func(p *PolicyTerms) { p.Rate, p.Evidence = "0", "" },
		"rate without evidence":   func(p *PolicyTerms) { p.Evidence = "" },
		"evidence form":           func(p *PolicyTerms) { p.Evidence = "abc" },
		"zero rate with evidence": func(p *PolicyTerms) { p.Rate, p.SourceDiscounted, p.SourceBasis = "0", Excluded, "" },
		"basis form":              func(p *PolicyTerms) { p.SourceBasis = "x" },
		"quantum":                 func(p *PolicyTerms) { p.Quantum = "0.05" },
	}
	for name, change := range bad {
		p := good
		p.Scope = append([]string{}, good.Scope...)
		change(&p)
		if _, err := r.RecordPolicy("ws", p, "author-1"); kind(err) != "invalid" {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := r.RecordPolicy("ws", good, ""); kind(err) != "invalid" {
		t.Fatal("author")
	}
	if _, err := r.RecordPolicy("", good, "author-1"); kind(err) != "invalid" {
		t.Fatal("workspace")
	}
	missing := good
	missing.Transaction = "none"
	if _, err := r.RecordPolicy("ws", missing, "author-1"); kind(err) != "not_found" {
		t.Fatal("no transaction")
	}
	if _, err := r.RecordPolicy("other", good, "author-1"); kind(err) != "not_found" {
		t.Fatal("another workspace")
	}
	edge := good
	edge.Rate, edge.Scope = "0.99999999", []string{KindOther, KindSoftware, KindInfrastructure}
	state, err := r.RecordPolicy("ws", edge, "author-1")
	if err != nil || strings.Join(state.Terms.Scope, ",") != "infrastructure,other,software" || state.Approver != "" {
		t.Fatalf("a rate just below one: %v %+v", err, state)
	}
	for index := 0; index < 6; index++ {
		p := terms("p"+string(rune('a'+index)), "1", "tx-1", "0", []string{KindSoftware}, Excluded)
		_, err = r.RecordPolicy("ws", p, "author-1")
	}
	if kind(err) != "limit" {
		t.Fatalf("policies: %v", err)
	}
	for name, call := range map[string]func() error{
		"approve":    func() error { _, err := r.ApprovePolicy("ws", "pol", "1", ""); return err },
		"policy for": func() error { _, err := r.PolicyFor("ws", "tx-1", "2026"); return err },
		"state":      func() error { _, err := r.PolicyState("ws", "", "1"); return err },
	} {
		if err := call(); kind(err) != "invalid" {
			t.Errorf("%s: %v", name, err)
		}
	}
	for name, call := range map[string]func() error{
		"approve ws":    func() error { _, err := r.ApprovePolicy("other", "pol", "1", "a"); return err },
		"policy for ws": func() error { _, err := r.PolicyFor("other", "tx-1", "2026-03"); return err },
		"policy for tx": func() error { _, err := r.PolicyFor("ws", "none", "2026-03"); return err },
		"state ws":      func() error { _, err := r.PolicyState("other", "pol", "1"); return err },
	} {
		if err := call(); kind(err) != "not_found" {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestChargeRefusalsAndTheTenantBoundary(t *testing.T) {
	r, s := example(t)
	approve(t, r, terms("pol", "1", "tx-1", "0.2", []string{KindSoftware}, Excluded))
	p := policyFor(t, r, "tx-1", "2026-03")
	q, qs := example(t)
	if _, err := q.RecordTransaction("other", buyerPrivate("tx-1")); err != nil {
		t.Fatal(err)
	}
	if _, err := q.RecordStatement("other", "tx-1", "2026-03", 1, exampleFacts()); err != nil {
		t.Fatal(err)
	}
	if st, _ := q.Seal("other", "tx-1", "2026-03", 1); st.State != StateSealed {
		t.Fatal("the same source in another workspace")
	}
	if _, err := q.RecordPolicy("other", terms("pol", "1", "tx-1", "0.2", []string{KindSoftware}, Excluded), "author-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := q.ApprovePolicy("other", "pol", "1", "approver-1"); err != nil {
		t.Fatal(err)
	}
	foreign, _ := q.PolicyFor("other", "tx-1", "2026-03")
	if _, err := ComputeCharge(qs, foreign); kind(err) != "invalid" {
		t.Fatalf("a policy of another workspace: %v", err)
	}
	if _, err := q.PolicyFor("ws", "tx-1", "2026-03"); kind(err) != "not_found" {
		t.Fatal("a policy seen from another workspace")
	}
	r.RecordTransaction("ws", deal("tx-2", RoleBuyer, ChannelMarketplace, TypePublic, CollectorMSP))
	approve(t, r, terms("pol-2", "1", "tx-2", "0", []string{KindSoftware}, Excluded))
	for name, call := range map[string]func() error{
		"empty statement":   func() error { _, err := ComputeCharge(Statement{}, p); return err },
		"empty policy":      func() error { _, err := ComputeCharge(s, Policy{}); return err },
		"other transaction": func() error { _, err := ComputeCharge(s, policyFor(t, r, "tx-2", "2026-03")); return err },
	} {
		if err := call(); kind(err) != "invalid" {
			t.Errorf("%s: %v", name, err)
		}
	}
	tiny := Limits{Transactions: 2, Revisions: 1, Lines: 2, Recognitions: 1, Settlements: 1, Policies: 2, Digits: 3}
	small, err := NewRegistry(testProfile(t), tiny, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	small.RecordTransaction("ws", buyerPrivate("tx-1"))
	small.RecordStatement("ws", "tx-1", "2026-03", 1, Facts{Lines: []Line{line("a", KindSoftware, "999", notApplied()), line("b", KindSoftware, "1", notApplied())}})
	small.Seal("ws", "tx-1", "2026-03", 1)
	big, _ := small.Statement("ws", "tx-1", "2026-03", 1)
	small.RecordPolicy("ws", terms("pol", "1", "tx-1", "0", []string{KindSoftware}, Excluded), "a")
	small.ApprovePolicy("ws", "pol", "1", "b")
	bp, _ := small.PolicyFor("ws", "tx-1", "2026-03")
	if _, err := ComputeCharge(big, bp); kind(err) != "limit" {
		t.Fatalf("digits: %v", err)
	}
	if _, err := small.RecordSettlement("ws", "tx-1", "s", 1, SettlementFacts{}); kind(err) != "invalid" {
		t.Fatal("an empty settlement")
	}
}

func TestSettlementForms(t *testing.T) {
	r := open(t)
	record(t, r, deal("tx", RoleSeller, ChannelMarketplace, TypePublic, CollectorMarketplace))
	good := SettlementFacts{State: SettlementPaid, Currency: "USD", Gross: "10", Fee: "1", Refund: "0", Tax: "0", Hold: "0", Net: "9", Report: "rep", Payout: "pay", Received: "9"}
	bad := map[string]func(*SettlementFacts){
		"state":               func(f *SettlementFacts) { f.State = "settled" },
		"currency":            func(f *SettlementFacts) { f.Currency = "US" },
		"gross":               func(f *SettlementFacts) { f.Gross = "" },
		"fee":                 func(f *SettlementFacts) { f.Fee = "-1" },
		"refund":              func(f *SettlementFacts) { f.Refund = "x" },
		"tax":                 func(f *SettlementFacts) { f.Tax = "1e1" },
		"hold":                func(f *SettlementFacts) { f.Hold = "0.123456789" },
		"net":                 func(f *SettlementFacts) { f.Net = "-9" },
		"paid without report": func(f *SettlementFacts) { f.Report = "" },
		"report form":         func(f *SettlementFacts) { f.Report = "a b" },
		"paid without payout": func(f *SettlementFacts) { f.Payout = "" },
		"paid without amount": func(f *SettlementFacts) { f.Received = "" },
		"received negative":   func(f *SettlementFacts) { f.Received = "-9" },
		"reported with payout": func(f *SettlementFacts) {
			f.State, f.Received = SettlementReported, ""
		},
		"reported with received": func(f *SettlementFacts) { f.State, f.Payout = SettlementReported, "" },
		"expected with report":   func(f *SettlementFacts) { f.State, f.Payout, f.Received = SettlementExpected, "", "" },
	}
	for name, change := range bad {
		f := good
		change(&f)
		if _, err := r.RecordSettlement("ws", "tx", "s", 1, f); kind(err) != "invalid" {
			t.Errorf("%s: %v", name, err)
		}
	}
	for name, call := range map[string]func() error{
		"workspace":  func() error { _, err := r.RecordSettlement("", "tx", "s", 1, good); return err },
		"id":         func() error { _, err := r.RecordSettlement("ws", "", "s", 1, good); return err },
		"settlement": func() error { _, err := r.RecordSettlement("ws", "tx", "", 1, good); return err },
		"zero":       func() error { _, err := r.RecordSettlement("ws", "tx", "s", 0, good); return err },
		"above":      func() error { _, err := r.RecordSettlement("ws", "tx", "s", 1001, good); return err },
	} {
		if err := call(); kind(err) != "invalid" {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := r.RecordSettlement("ws", "none", "s", 1, good); kind(err) != "not_found" {
		t.Fatal("no transaction")
	}
	padded := good
	padded.Gross = "10.00"
	v, err := r.RecordSettlement("ws", "tx", "s", 1, padded)
	if err != nil || v.Facts.Gross != "10" || !v.Received || !v.WrittenAt.Equal(start) {
		t.Fatalf("canonical: %v %+v", err, v)
	}
	expected := SettlementFacts{State: SettlementExpected, Currency: "USD", Gross: "1", Fee: "0", Refund: "0", Tax: "0", Hold: "0", Net: "1"}
	for _, id := range []string{"s2", "s3"} {
		if _, err := r.RecordSettlement("ws", "tx", id, 1, expected); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := r.RecordSettlement("ws", "tx", "s4", 1, expected); kind(err) != "limit" {
		t.Fatal("settlements")
	}
	reported := expected
	reported.State, reported.Report = SettlementReported, "rep"
	r.RecordSettlement("ws", "tx", "s2", 2, reported)
	r.RecordSettlement("ws", "tx", "s2", 3, reported)
	if _, err := r.RecordSettlement("ws", "tx", "s2", 4, reported); kind(err) != "limit" {
		t.Fatal("revisions")
	}
	if got, _ := r.Settlement("ws", "tx", "s2"); got.Revision != 3 || got.Facts.State != SettlementReported || got.Received {
		t.Fatalf("highest: %+v", got)
	}
}

func TestDigests(t *testing.T) {
	r, s := example(t)
	tx := []string{"tx-1", "aws", "agr-1", RoleBuyer, ChannelMarketplace, TypePrivate, CollectorMSP, "isv-1", "isv-1", "", "aws", "", "prod-1", "offer-1", "", "",
		"2026-01", "2026-01", "2026-12", "USD"}
	fields := append([]string{"marketplace_statement_v1", "ws"}, tx...)
	fields = append(fields, "2026-03", "1", "2",
		"infra-1", KindInfrastructure, "acct-1", "200", "", NotApplied, "", evidenceA,
		"sw-1", KindSoftware, "acct-1", "800", "1000", Applied, "200", evidenceA,
		"1", "macc-1", Applied, "800", evidenceB)
	if s.View().Digest != printed(fields...) {
		t.Fatalf("statement digest: %s", s.View().Digest)
	}
	pt := terms("pol", "1", "tx-1", "0.2", []string{KindSoftware}, Excluded)
	approve(t, r, pt)
	state, _ := r.PolicyState("ws", "pol", "1")
	if state.Digest != printed("marketplace_policy_v1", "ws", "pol", "1", "cust-1", "tx-1", "2026-01", "2026-12", "0.2", "1", "software", Excluded, evidenceC, "", "0.01") {
		t.Fatal("policy digest")
	}
	v := compute(t, s, policyFor(t, r, "tx-1", "2026-03"))
	want := printed("marketplace_charge_v1", "market-test", "1", "tx-1", "2026-03", "1", s.View().Digest, "pol", "1", state.Digest, "2026-01", "2026-12", "cust-1", "",
		"2", "infra-1", KindInfrastructure, "200", "0", "200", "sw-1", KindSoftware, "800", "0", "800",
		"1", "macc-1", Applied, "800", evidenceB, "1000", "1000", "0")
	if v.Digest != want || v.Label != ChargeLabel || v.ProfileID != "market-test" || v.ProfileRev != "1" {
		t.Fatalf("charge digest: %+v", v)
	}
	q := open(t)
	record(t, q, deal("tx", RoleSeller, ChannelMarketplace, TypePublic, CollectorMarketplace))
	f := SettlementFacts{State: SettlementPaid, Currency: "USD", Gross: "800", Fee: "24", Refund: "0", Tax: "0", Hold: "0", Net: "776", Report: "rep", Payout: "pay", Received: "776"}
	sv, _ := q.RecordSettlement("ws", "tx", "s1", 1, f)
	if sv.Digest != printed("marketplace_settlement_v1", "ws", "tx", "aws", "agr-1", "s1", "1", SettlementPaid, "USD", "800", "24", "0", "0", "0", "776", "rep", "pay", "776") {
		t.Fatal("settlement digest")
	}
	seen := map[string]bool{s.View().Digest: true}
	base := buyerPrivate("tx-1")
	variants := []func(*Transaction){
		func(x *Transaction) { x.Provider = "aws2" }, func(x *Transaction) { x.Agreement = "agr-2" },
		func(x *Transaction) { x.Channel = ChannelDirect }, func(x *Transaction) { x.Type = TypePublic },
		func(x *Transaction) { x.Participants.SoftwareProvider = "isv-2" }, func(x *Transaction) { x.Participants.Seller = "isv-2" },
		func(x *Transaction) { x.Participants.Issuer = "isv-2" }, func(x *Transaction) { x.References.Product = "p2" },
		func(x *Transaction) { x.References.Offer = "o2" }, func(x *Transaction) { x.References.Subscription = "s2" },
		func(x *Transaction) { x.References.Order = "o2" }, func(x *Transaction) { x.Term.Accepted = "2025-12" },
		func(x *Transaction) { x.Term.Start = "2026-02" }, func(x *Transaction) { x.Term.End = "2026-11" },
		func(x *Transaction) { x.Term.Currency = "EUR" },
	}
	for index, change := range variants {
		w := open(t)
		tx := base
		change(&tx)
		record(t, w, tx)
		state, err := w.RecordStatement("ws", "tx-1", "2026-03", 1, exampleFacts())
		if err != nil || seen[state.Digest] {
			t.Errorf("variant %d: %v", index, err)
		}
		seen[state.Digest] = true
	}
	view := s.View()
	view.Facts.Lines[0].Charged = "1"
	if s.View().Facts.Lines[0].Charged != "200" {
		t.Fatal("a view shares its lines")
	}
	charge, _ := ComputeCharge(s, policyFor(t, r, "tx-1", "2026-03"))
	cv := charge.View()
	cv.Lines[0].Amount = "1"
	if charge.View().Lines[0].Amount != "200" {
		t.Fatal("a charge view shares its lines")
	}
}

func TestConcurrentSealsOfASharedSource(t *testing.T) {
	for round := 0; round < 100; round++ {
		r := open(t)
		record(t, r, buyerPrivate("tx-1"))
		record(t, r, deal("tx-2", RoleBuyer, ChannelMarketplace, TypePublic, CollectorMSP))
		shared := Facts{Lines: []Line{line("dup", KindSoftware, "10", notApplied())}}
		write(t, r, "tx-1", "2026-03", 1, shared)
		write(t, r, "tx-2", "2026-03", 1, shared)
		gate := make(chan struct{})
		var group sync.WaitGroup
		states := make([]RevisionState, 2)
		for index, id := range []string{"tx-1", "tx-2"} {
			group.Add(1)
			go func(index int, id string) {
				defer group.Done()
				<-gate
				states[index], _ = r.Seal("ws", id, "2026-03", 1)
			}(index, id)
		}
		close(gate)
		group.Wait()
		if states[0].State != StateHeld || states[1].State != StateHeld {
			t.Fatalf("a shared source sealed: %+v", states)
		}
		write(t, r, "tx-1", "2026-03", 2, Facts{Lines: []Line{}})
		sealed(t, r, "tx-1", "2026-03", 2)
		if s := seal(t, r, "tx-2", "2026-03", 1); s.State != StateSealed {
			t.Fatalf("after the correction: %+v", s)
		}
	}
}

func TestTheClockNeverMovesBack(t *testing.T) {
	r, clock := openWith(t)
	record(t, r, buyerPrivate("tx-1"))
	clock.set(start.Add(time.Hour).In(time.FixedZone("x", 3600)))
	write(t, r, "tx-1", "2026-03", 1, exampleFacts())
	clock.set(start.Add(-time.Hour))
	s := seal(t, r, "tx-1", "2026-03", 1)
	if !s.WrittenAt.Equal(start.Add(time.Hour)) || !s.JudgedAt.Equal(start.Add(time.Hour)) || s.JudgedAt.Location() != time.UTC {
		t.Fatalf("times: %+v", s)
	}
}

func TestTheSourceHasNoFloatingPointOrComments(t *testing.T) {
	files, _ := filepath.Glob("*.go")
	scanned := 0
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		scanned++
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
	if scanned < 8 {
		t.Fatalf("scanned %d files", scanned)
	}
}
