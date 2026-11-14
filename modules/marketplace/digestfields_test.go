package marketplace

import "testing"

func TestEveryFactFieldChangesTheStatementDigest(t *testing.T) {
	base := func() Facts { return exampleFacts() }
	variants := []func(*Facts){
		func(f *Facts) { f.Lines[0].Source = "sw-2" }, func(f *Facts) { f.Lines[0].Kind = KindOther },
		func(f *Facts) { f.Lines[0].Account = "acct-2" }, func(f *Facts) { f.Lines[0].Charged = "801" },
		func(f *Facts) { f.Lines[0].List = "1001" }, func(f *Facts) { f.Lines[0].Discount.Amount = "201" },
		func(f *Facts) { f.Lines[0].Discount = notApplied() }, func(f *Facts) { f.Lines[0].Discount.Evidence = evidenceC },
		func(f *Facts) { f.Recognitions[0].Program = "macc-2" }, func(f *Facts) { f.Recognitions[0].Amount = "799" },
		func(f *Facts) { f.Recognitions[0].Evidence = evidenceC }, func(f *Facts) { f.Recognitions[0] = Recognition{Program: "macc-1", State: Unknown} },
		func(f *Facts) { f.Recognitions = nil },
	}
	r := open(t)
	record(t, r, buyerPrivate("tx-1"))
	first, _ := r.RecordStatement("ws", "tx-1", "2026-03", 1, base())
	seen := map[string]bool{first.Digest: true}
	for index, change := range variants {
		q := open(t)
		record(t, q, buyerPrivate("tx-1"))
		facts := base()
		change(&facts)
		state, err := q.RecordStatement("ws", "tx-1", "2026-03", 1, facts)
		if err != nil || seen[state.Digest] {
			t.Errorf("variant %d: %v", index, err)
		}
		seen[state.Digest] = true
	}
	for _, other := range []struct {
		ws, month string
		number    int
	}{{"ws2", "2026-03", 1}, {"ws", "2026-04", 1}} {
		q := open(t)
		q.RecordTransaction(other.ws, buyerPrivate("tx-1"))
		state, err := q.RecordStatement(other.ws, "tx-1", other.month, other.number, base())
		if err != nil || seen[state.Digest] {
			t.Errorf("%+v: %v", other, err)
		}
		seen[state.Digest] = true
	}
	second, _ := r.RecordStatement("ws", "tx-1", "2026-03", 2, base())
	if seen[second.Digest] {
		t.Fatal("the revision number")
	}
	roles := []Transaction{deal("tx-1", RoleReseller, ChannelMarketplace, TypePrivate, CollectorMSP), deal("tx-1", RoleBuyer, ChannelMarketplace, TypePrivate, CollectorMSP)}
	roles[1].Participants.Seller = "isv-1"
	seller := deal("tx-1", RoleSeller, ChannelMarketplace, TypePrivate, CollectorMarketplace)
	for index, tx := range []Transaction{roles[0], seller} {
		q := open(t)
		record(t, q, tx)
		state, _ := q.RecordStatement("ws", "tx-1", "2026-03", 1, base())
		if seen[state.Digest] {
			t.Errorf("role or collector %d", index)
		}
		seen[state.Digest] = true
	}
}

func TestEveryPolicyFieldChangesItsDigest(t *testing.T) {
	r := open(t)
	record(t, r, buyerPrivate("tx-1"))
	record(t, r, buyerPrivate("tx-2"))
	seen := map[string]bool{}
	variants := []func(*PolicyTerms){
		func(p *PolicyTerms) {}, func(p *PolicyTerms) { p.ID = "pol-b" }, func(p *PolicyTerms) { p.Version = "2" },
		func(p *PolicyTerms) { p.Customer = "cust-2" }, func(p *PolicyTerms) { p.Transaction = "tx-2" },
		func(p *PolicyTerms) { p.First = "2026-02" }, func(p *PolicyTerms) { p.Last = "2026-11" }, func(p *PolicyTerms) { p.Rate = "0.3" },
		func(p *PolicyTerms) { p.Scope = []string{KindOther} }, func(p *PolicyTerms) { p.SourceDiscounted, p.SourceBasis = Excluded, "" },
		func(p *PolicyTerms) { p.Evidence = evidenceA }, func(p *PolicyTerms) { p.SourceBasis = evidenceA }, func(p *PolicyTerms) { p.Quantum = "1" },
	}
	for index, change := range variants {
		q := open(t)
		record(t, q, buyerPrivate("tx-1"))
		record(t, q, buyerPrivate("tx-2"))
		p := terms("pol", "1", "tx-1", "0.2", []string{KindSoftware}, Included)
		change(&p)
		state, err := q.RecordPolicy("ws", p, "author-1")
		if err != nil || seen[state.Digest] {
			t.Errorf("variant %d: %v", index, err)
		}
		seen[state.Digest] = true
	}
}

func TestEverySettlementFieldChangesItsDigest(t *testing.T) {
	base := SettlementFacts{State: SettlementPaid, Currency: "USD", Gross: "800", Fee: "24", Refund: "0", Tax: "0", Hold: "0", Net: "776", Report: "rep", Payout: "pay", Received: "776"}
	variants := []func(*SettlementFacts){
		func(f *SettlementFacts) {}, func(f *SettlementFacts) { f.State, f.Payout, f.Received = SettlementReported, "", "" },
		func(f *SettlementFacts) { f.Gross = "801" }, func(f *SettlementFacts) { f.Fee = "25" }, func(f *SettlementFacts) { f.Refund = "1" },
		func(f *SettlementFacts) { f.Tax = "1" }, func(f *SettlementFacts) { f.Hold = "1" }, func(f *SettlementFacts) { f.Net = "775" },
		func(f *SettlementFacts) { f.Report = "rep-2" }, func(f *SettlementFacts) { f.Payout = "pay-2" }, func(f *SettlementFacts) { f.Received = "775" },
	}
	seen := map[string]bool{}
	for index, change := range variants {
		q := open(t)
		record(t, q, deal("tx", RoleSeller, ChannelMarketplace, TypePublic, CollectorMarketplace))
		f := base
		change(&f)
		v, err := q.RecordSettlement("ws", "tx", "s1", 1, f)
		if err != nil || seen[v.Digest] {
			t.Errorf("variant %d: %v", index, err)
		}
		seen[v.Digest] = true
	}
}

func TestEveryChargeFieldChangesItsDigest(t *testing.T) {
	seen := map[string]bool{}
	for index, p := range []PolicyTerms{
		terms("pol", "1", "tx-1", "0.2", []string{KindSoftware}, Excluded),
		terms("pol", "1", "tx-1", "0.2", []string{KindInfrastructure}, Excluded),
		terms("pol", "1", "tx-1", "0.2", []string{KindSoftware}, Included),
		terms("pol", "2", "tx-1", "0.2", []string{KindSoftware}, Excluded),
		terms("pol-b", "1", "tx-1", "0.2", []string{KindSoftware}, Excluded),
	} {
		r, s := example(t)
		approve(t, r, p)
		v := compute(t, s, policyFor(t, r, "tx-1", "2026-03"))
		if seen[v.Digest] {
			t.Errorf("policy %d", index)
		}
		seen[v.Digest] = true
	}
	r, s := example(t)
	p := terms("pol", "1", "tx-1", "0.2", []string{KindSoftware}, Excluded)
	p.Customer = "cust-2"
	approve(t, r, p)
	if v := compute(t, s, policyFor(t, r, "tx-1", "2026-03")); seen[v.Digest] {
		t.Fatal("the customer")
	}
}
