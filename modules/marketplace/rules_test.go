package marketplace

import (
	"reflect"
	"strings"
	"testing"
)

func TestWhichLinesAreDiscounted(t *testing.T) {
	cases := []struct {
		name   string
		offer  string
		lines  []Line
		rate   string
		scope  []string
		source string
		want   string
	}{
		{"applied on a public offer", TypePublic, []Line{line("a", KindSoftware, "100", applied("10"))}, "0.1", []string{KindSoftware}, Excluded, "a=100/-0 | total 100 charged 100 margin 0"},
		{"applied on a public offer included", TypePublic, []Line{line("a", KindSoftware, "100", applied("10"))}, "0.1", []string{KindSoftware}, Included, "a=90/-10 | total 90 charged 100 margin -10"},
		{"not applied on a public offer", TypePublic, []Line{line("a", KindSoftware, "100", notApplied())}, "0.1", []string{KindSoftware}, Excluded, "a=90/-10 | total 90 charged 100 margin -10"},
		{"unknown in scope", TypePublic, []Line{line("a", KindSoftware, "100", unknown())}, "0.1", []string{KindSoftware}, Excluded, "hold source_discount_unknown"},
		{"unknown in scope included", TypePublic, []Line{line("a", KindSoftware, "100", unknown())}, "0.1", []string{KindSoftware}, Included, "hold source_discount_unknown"},
		{"unknown outside the scope", TypePublic, []Line{line("a", KindOther, "100", unknown()), line("b", KindSoftware, "50", notApplied())}, "0.1", []string{KindSoftware}, Excluded, "a=100/-0 b=45/-5 | total 145 charged 150 margin -5"},
		{"unknown at rate zero", TypePublic, []Line{line("a", KindSoftware, "100", unknown())}, "0", []string{KindSoftware}, Excluded, "a=100/-0 | total 100 charged 100 margin 0"},
		{"private offer software unknown", TypePrivate, []Line{line("a", KindSoftware, "100", unknown())}, "0.1", []string{KindSoftware}, Excluded, "a=100/-0 | total 100 charged 100 margin 0"},
		{"private offer infrastructure in scope", TypePrivate, []Line{line("a", KindSoftware, "100", notApplied()), line("b", KindInfrastructure, "50", notApplied())}, "0.1", []string{KindSoftware, KindInfrastructure}, Excluded, "a=100/-0 b=45/-5 | total 145 charged 150 margin -5"},
		{"a discount reduced toward zero", TypePublic, []Line{line("a", KindSoftware, "0.333", notApplied())}, "0.1", []string{KindSoftware}, Excluded, "a=0.303/-0.03 | total 0.303 charged 0.333 margin -0.03"},
		{"a discount below one quantum", TypePublic, []Line{line("a", KindSoftware, "0.05", notApplied())}, "0.1", []string{KindSoftware}, Excluded, "a=0.05/-0 | total 0.05 charged 0.05 margin 0"},
	}
	for _, c := range cases {
		r := open(t)
		record(t, r, deal("tx", RoleBuyer, ChannelMarketplace, c.offer, CollectorMSP))
		write(t, r, "tx", "2026-03", 1, Facts{Lines: c.lines})
		approve(t, r, terms("pol", "1", "tx", c.rate, c.scope, c.source))
		if got := amounts(compute(t, sealed(t, r, "tx", "2026-03", 1), policyFor(t, r, "tx", "2026-03"))); got != c.want {
			t.Errorf("%s: %s", c.name, got)
		}
	}
}

func TestHoldsInTheirOrder(t *testing.T) {
	unknownLine := Facts{Lines: []Line{line("a", KindSoftware, "100", unknown())}}
	cases := []struct {
		tx   Transaction
		want string
	}{
		{deal("tx", RoleSeller, ChannelMarketplace, TypePublic, CollectorMarketplace), HoldSellerTransaction},
		{deal("tx", RoleSeller, ChannelDirect, TypePublic, CollectorMSP), HoldSellerTransaction},
		{deal("tx", RoleReseller, ChannelMarketplace, TypePublic, CollectorMarketplace), HoldCollectedElsewhere},
		{deal("tx", RoleReseller, ChannelDistributor, TypeChannelPrivate, CollectorDistributor), HoldCollectedElsewhere},
		{deal("tx", RoleReseller, ChannelMarketplace, TypePublic, CollectorMSP), HoldSourceUnknown},
	}
	for _, c := range cases {
		r := open(t)
		record(t, r, c.tx)
		write(t, r, "tx", "2026-03", 1, unknownLine)
		approve(t, r, terms("pol", "1", "tx", "0.1", []string{KindSoftware}, Excluded))
		charge, err := ComputeCharge(sealed(t, r, "tx", "2026-03", 1), policyFor(t, r, "tx", "2026-03"))
		v := charge.View()
		if err != nil || v.Hold != c.want || len(v.Lines) != 0 || len(v.Recognitions) != 0 || v.Total != "" || v.ChargedTotal != "" || v.Margin != "" || v.Digest == "" {
			t.Errorf("%+v: %v %+v", c.tx, err, v)
		}
		if _, err := ViewForCustomer(charge); kind(err) != "rejected" {
			t.Errorf("a held charge has a view: %v", err)
		}
	}
}

func TestTheCustomerViewLeavesOutWhatACustomerMayNotSee(t *testing.T) {
	r, s := example(t)
	approve(t, r, terms("pol", "1", "tx-1", "0.2", []string{KindInfrastructure}, Excluded))
	charge, _ := ComputeCharge(s, policyFor(t, r, "tx-1", "2026-03"))
	v, err := ViewForCustomer(charge)
	if err != nil {
		t.Fatal(err)
	}
	want := CustomerView{Transaction: "tx-1", Month: "2026-03", Customer: "cust-1", Lines: []CustomerLine{{"infra-1", KindInfrastructure, "160"}, {"sw-1", KindSoftware, "800"}},
		Total: "960", Digest: charge.View().Digest}
	if !reflect.DeepEqual(v, want) {
		t.Fatalf("%+v", v)
	}
	fields := []string{}
	for index := 0; index < reflect.TypeOf(v).NumField(); index++ {
		fields = append(fields, reflect.TypeOf(v).Field(index).Name)
	}
	if strings.Join(fields, ",") != "Transaction,Month,Customer,Lines,Total,Digest" || reflect.TypeOf(CustomerLine{}).NumField() != 3 {
		t.Fatalf("view fields: %v", fields)
	}
	if _, err := ViewForCustomer(Charge{}); kind(err) != "invalid" {
		t.Fatal("a charge not computed here")
	}
}

func TestASettlementNeverChangesACustomerCharge(t *testing.T) {
	r := open(t)
	record(t, r, deal("tx", RoleReseller, ChannelMarketplace, TypeChannelPrivate, CollectorMSP))
	write(t, r, "tx", "2026-03", 1, Facts{Lines: []Line{line("a", KindSoftware, "800", applied("200"))}})
	approve(t, r, terms("pol", "1", "tx", "0", []string{KindSoftware}, Excluded))
	before := compute(t, sealed(t, r, "tx", "2026-03", 1), policyFor(t, r, "tx", "2026-03"))
	late := SettlementFacts{State: SettlementExpected, Currency: "USD", Gross: "800", Fee: "24", Refund: "0", Tax: "0", Hold: "776", Net: "0"}
	if _, err := r.RecordSettlement("ws", "tx", "set-1", 1, late); err != nil {
		t.Fatal(err)
	}
	after := compute(t, sealed(t, r, "tx", "2026-03", 1), policyFor(t, r, "tx", "2026-03"))
	if before.Digest != after.Digest || after.Total != "800" {
		t.Fatalf("before %s after %s", amounts(before), amounts(after))
	}
}

func TestSettlementReasonsAndStates(t *testing.T) {
	r := open(t)
	record(t, r, deal("tx", RoleSeller, ChannelMarketplace, TypePublic, CollectorMarketplace))
	record(t, r, deal("tx-b", RoleBuyer, ChannelMarketplace, TypePublic, CollectorMSP))
	other := deal("tx-o", RoleSeller, ChannelMarketplace, TypePublic, CollectorMarketplace)
	other.Agreement = "agr-2"
	record(t, r, other)
	base := SettlementFacts{State: SettlementPaid, Currency: "USD", Gross: "100", Fee: "3", Refund: "1", Tax: "2", Hold: "4", Net: "90", Report: "rep", Payout: "pay-1", Received: "90"}
	if v, _ := r.RecordSettlement("ws", "tx", "s1", 1, base); v.Judged != JudgedConsistent || !v.Received {
		t.Fatalf("every deduction: %+v", v)
	}
	both := base
	both.Net, both.Payout = "91", "pay-2"
	if v, _ := r.RecordSettlement("ws", "tx", "s2", 1, both); !reflect.DeepEqual(v.Reasons, []string{ReasonNetNotPreserved, ReasonReceivedDiffers}) || v.Received {
		t.Fatalf("both reasons: %+v", v)
	}
	over := base
	over.Gross, over.Net, over.Received, over.Payout = "5", "0", "0", "pay-3"
	if v, _ := r.RecordSettlement("ws", "tx", "s3", 1, over); !reflect.DeepEqual(v.Reasons, []string{ReasonNetNotPreserved}) {
		t.Fatalf("gross below deductions: %+v", v)
	}
	reused := base
	if v, _ := r.RecordSettlement("ws", "tx-o", "s1", 1, reused); !reflect.DeepEqual(v.Reasons, []string{ReasonPayoutReused}) || v.Received {
		t.Fatalf("a payout reused across agreements: %+v", v)
	}
	if _, err := r.RecordSettlement("ws", "tx-b", "s1", 1, base); kind(err) != "rejected" {
		t.Fatalf("a settlement of a buyer: %v", err)
	}
	expected := SettlementFacts{State: SettlementExpected, Currency: "USD", Gross: "1", Fee: "0", Refund: "0", Tax: "0", Hold: "0", Net: "1"}
	if _, err := r.RecordSettlement("ws", "tx", "s1", 2, expected); kind(err) != "rejected" {
		t.Fatalf("a paid settlement moved back: %v", err)
	}
	wrong := expected
	wrong.Currency = "EUR"
	if _, err := r.RecordSettlement("ws", "tx", "s4", 1, wrong); kind(err) != "rejected" {
		t.Fatalf("another currency: %v", err)
	}
	if _, err := r.RecordSettlement("ws", "tx", "s1", 3, base); kind(err) != "rejected" {
		t.Fatalf("a revision beyond the next: %v", err)
	}
	if v, err := r.RecordSettlement("ws", "tx", "s1", 1, base); err != nil || !v.Received || v.Revision != 1 {
		t.Fatalf("a repeat: %v %+v", err, v)
	}
	if _, err := r.RecordSettlement("ws", "tx", "s1", 1, both); kind(err) != "conflict" {
		t.Fatalf("a changed repeat: %v", err)
	}
	fixed := both
	fixed.Net = "90"
	fixed.Received = "90"
	if v, _ := r.RecordSettlement("ws", "tx", "s2", 2, fixed); v.Judged != JudgedConsistent || !v.Received {
		t.Fatalf("a correction judged again: %+v", v)
	}
	if _, err := r.Settlement("ws", "tx", "none"); kind(err) != "not_found" {
		t.Fatalf("no settlement: %v", err)
	}
}

func TestSealsCorrectionsAndCurrent(t *testing.T) {
	r := open(t)
	record(t, r, buyerPrivate("tx-1"))
	record(t, r, deal("tx-2", RoleBuyer, ChannelMarketplace, TypePublic, CollectorMSP))
	write(t, r, "tx-1", "2026-03", 1, Facts{Lines: []Line{line("dup", KindSoftware, "10", notApplied())}})
	sealed(t, r, "tx-1", "2026-03", 1)
	write(t, r, "tx-2", "2026-03", 1, Facts{Lines: []Line{line("dup", KindSoftware, "10", notApplied())}, Recognitions: []Recognition{recognized("11")}})
	both := []string{ReasonDuplicateSource, ReasonRecognitionExceedsCharge}
	if s := seal(t, r, "tx-2", "2026-03", 1); !reflect.DeepEqual(s.Reasons, both) || s.State != StateHeld {
		t.Fatalf("both reasons: %+v", s)
	}
	if cur, _ := r.Current("ws", "tx-2", "2026-03"); cur.State != CurrentCorrectionHeld {
		t.Fatalf("current: %+v", cur)
	}
	if _, err := r.Statement("ws", "tx-2", "2026-03", 1); kind(err) != "rejected" {
		t.Fatalf("a held statement read: %v", err)
	}
	write(t, r, "tx-1", "2026-03", 2, Facts{Lines: []Line{line("own", KindSoftware, "10", notApplied())}})
	if s := seal(t, r, "tx-2", "2026-03", 1); s.State != StateHeld || !reflect.DeepEqual(s.Reasons, both) {
		t.Fatalf("a line sold in a sealed revision stays linked until its correction is sealed: %+v", s)
	}
	if _, err := r.Statement("ws", "tx-1", "2026-03", 1); kind(err) != "rejected" {
		t.Fatalf("a sealed revision below a correction read: %v", err)
	}
	sealed(t, r, "tx-1", "2026-03", 2)
	if s := seal(t, r, "tx-2", "2026-03", 1); !reflect.DeepEqual(s.Reasons, []string{ReasonRecognitionExceedsCharge}) {
		t.Fatalf("judged again: %+v", s)
	}
	write(t, r, "tx-2", "2026-03", 2, Facts{Lines: []Line{line("dup", KindSoftware, "10", notApplied())}, Recognitions: []Recognition{recognized("10")}})
	if cur, _ := r.Current("ws", "tx-2", "2026-03"); cur.State != CurrentNotJudged || cur.Revision != 2 {
		t.Fatalf("not judged: %+v", cur)
	}
	if _, err := r.Seal("ws", "tx-2", "2026-03", 1); kind(err) != "rejected" {
		t.Fatalf("a lower revision sealed: %v", err)
	}
	sealed(t, r, "tx-2", "2026-03", 2)
	if cur, _ := r.Current("ws", "tx-2", "2026-03"); cur.State != CurrentSealed || cur.Revision != 2 {
		t.Fatalf("sealed: %+v", cur)
	}
	revs, _ := r.Revisions("ws", "tx-2", "2026-03")
	if len(revs) != 2 || revs[0].State != StateHeld || revs[1].State != StateSealed {
		t.Fatalf("revisions: %+v", revs)
	}

	q := open(t)
	record(t, q, buyerPrivate("tx-1"))
	record(t, q, deal("tx-2", RoleBuyer, ChannelMarketplace, TypePublic, CollectorMSP))
	write(t, q, "tx-1", "2026-03", 1, Facts{Lines: []Line{line("dup", KindSoftware, "10", notApplied())}})
	write(t, q, "tx-2", "2026-04", 1, Facts{Lines: []Line{line("dup", KindSoftware, "10", notApplied())}})
	if s := seal(t, q, "tx-2", "2026-04", 1); s.State != StateHeld {
		t.Fatal("a duplicate across months")
	}
	write(t, q, "tx-1", "2026-03", 2, Facts{Lines: []Line{}})
	sealed(t, q, "tx-1", "2026-03", 2)
	if s := seal(t, q, "tx-2", "2026-04", 1); s.State != StateSealed || len(s.Reasons) != 0 {
		t.Fatalf("a held revision judged again: %+v", s)
	}
	if s := seal(t, q, "tx-2", "2026-04", 1); s.State != StateSealed {
		t.Fatal("a sealed repeat")
	}
	old, _ := q.Revisions("ws", "tx-1", "2026-03")
	if old[0].State != StateSuperseded {
		t.Fatalf("an unjudged lower revision: %+v", old)
	}

	g := open(t)
	other := deal("tx-g", RoleBuyer, ChannelMarketplace, TypePublic, CollectorMSP)
	other.Provider = "gcp"
	record(t, g, buyerPrivate("tx-1"))
	record(t, g, other)
	write(t, g, "tx-1", "2026-03", 1, Facts{Lines: []Line{line("dup", KindSoftware, "10", notApplied())}})
	write(t, g, "tx-g", "2026-03", 1, Facts{Lines: []Line{line("dup", KindSoftware, "10", notApplied())}})
	if s := seal(t, g, "tx-g", "2026-03", 1); s.State != StateSealed {
		t.Fatal("another provider's line")
	}
	write(t, g, "tx-1", "2026-04", 1, Facts{Lines: []Line{line("x", KindSoftware, "10", notApplied())}, Recognitions: []Recognition{recognized("10"), {Program: "p2", State: Applied, Amount: "10", Evidence: evidenceA}, {Program: "p3", State: Unknown}}})
	if s := seal(t, g, "tx-1", "2026-04", 1); s.State != StateSealed {
		t.Fatalf("recognitions each up to the charge: %+v", s)
	}
}

func TestPoliciesVersionsAndApproval(t *testing.T) {
	r, clock := openWith(t)
	record(t, r, buyerPrivate("tx-1"))
	write(t, r, "tx-1", "2026-03", 1, exampleFacts())
	write(t, r, "tx-1", "2026-07", 1, exampleFacts2())
	march, july := sealed(t, r, "tx-1", "2026-03", 1), sealed(t, r, "tx-1", "2026-07", 1)
	v1 := terms("pol", "1", "tx-1", "0", []string{KindSoftware}, Excluded)
	if _, err := r.RecordPolicy("ws", v1, "author-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.PolicyFor("ws", "tx-1", "2026-03"); kind(err) != "not_found" {
		t.Fatalf("an unapproved policy: %v", err)
	}
	if _, err := r.ApprovePolicy("ws", "pol", "1", "author-1"); kind(err) != "rejected" {
		t.Fatalf("approved by its author: %v", err)
	}
	clock.set(start.Add(1))
	state, err := r.ApprovePolicy("ws", "pol", "1", "approver-1")
	if err != nil || state.Approver != "approver-1" || !state.ApprovedAt.Equal(start.Add(1)) || state.EffectiveLast != "2026-12" {
		t.Fatalf("approve: %v %+v", err, state)
	}
	if again, err := r.ApprovePolicy("ws", "pol", "1", "approver-1"); err != nil || again.ApprovedAt != state.ApprovedAt {
		t.Fatalf("a repeat: %v", err)
	}
	if _, err := r.ApprovePolicy("ws", "pol", "1", "approver-2"); kind(err) != "conflict" {
		t.Fatalf("another approver: %v", err)
	}
	early := policyFor(t, r, "tx-1", "2026-07")
	other := terms("pol-x", "1", "tx-1", "0", []string{KindSoftware}, Excluded)
	other.First, other.Last = "2026-12", "2026-12"
	if _, err := r.RecordPolicy("ws", other, "author-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ApprovePolicy("ws", "pol-x", "1", "approver-1"); kind(err) != "rejected" {
		t.Fatalf("an overlapping policy: %v", err)
	}
	v2 := terms("pol", "2", "tx-1", "0.5", []string{KindInfrastructure}, Excluded)
	v2.First = "2026-06"
	approve(t, r, v2)
	if s, _ := r.PolicyState("ws", "pol", "1"); s.EffectiveLast != "2026-05" || s.Terms.Last != "2026-12" {
		t.Fatalf("superseded: %+v", s)
	}
	v3 := terms("pol", "3", "tx-1", "0", []string{KindSoftware}, Excluded)
	v3.First = "2026-04"
	if _, err := r.RecordPolicy("ws", v3, "author-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ApprovePolicy("ws", "pol", "3", "approver-1"); kind(err) != "rejected" {
		t.Fatalf("a version starting before the newest: %v", err)
	}
	if got := amounts(compute(t, march, policyFor(t, r, "tx-1", "2026-03"))); got != "infra-1=200/-0 sw-1=800/-0 | total 1000 charged 1000 margin 0" {
		t.Fatalf("march under version 1: %s", got)
	}
	current := compute(t, july, policyFor(t, r, "tx-1", "2026-07"))
	if amounts(current) != "infra-7=50/-50 sw-7=400/-0 | total 450 charged 500 margin -50" || current.PolicyVersion != "2" || current.PolicyFirst != "2026-06" || current.PolicyLast != "2026-12" {
		t.Fatalf("july under version 2: %s %+v", amounts(current), current)
	}
	stale := compute(t, july, early)
	if stale.PolicyVersion != "1" || stale.PolicyLast != "2026-12" || stale.Digest == current.Digest {
		t.Fatalf("a value fetched before the new version shows its months: %+v", stale)
	}
	if _, err := ComputeCharge(july, policyFor(t, r, "tx-1", "2026-03")); kind(err) != "invalid" {
		t.Fatalf("a month outside the policy: %v", err)
	}
	outside := terms("pol-o", "1", "tx-1", "0", []string{KindSoftware}, Excluded)
	outside.Last = "2027-01"
	if _, err := r.RecordPolicy("ws", outside, "author-1"); kind(err) != "rejected" {
		t.Fatalf("months outside the term: %v", err)
	}
	if _, err := r.RecordPolicy("ws", v1, "author-2"); kind(err) != "conflict" {
		t.Fatalf("another author: %v", err)
	}
	if s, err := r.RecordPolicy("ws", v1, "author-1"); err != nil || s.Approver != "approver-1" {
		t.Fatalf("a repeat: %v", err)
	}
	if _, err := r.PolicyState("ws", "pol", "9"); kind(err) != "not_found" {
		t.Fatal("no such version")
	}
	if _, err := r.ApprovePolicy("ws", "pol", "9", "approver-1"); kind(err) != "not_found" {
		t.Fatal("no such version to approve")
	}
}

func exampleFacts2() Facts {
	return Facts{Lines: []Line{line("sw-7", KindSoftware, "400", applied("100")), line("infra-7", KindInfrastructure, "100", notApplied())}}
}

func TestAgreementChargesByRoleAndCurrency(t *testing.T) {
	r := open(t)
	record(t, r, buyerPrivate("tx-1"))
	eur := deal("tx-2", RoleBuyer, ChannelMarketplace, TypePublic, CollectorMSP)
	eur.Term.Currency = "EUR"
	record(t, r, eur)
	record(t, r, deal("tx-3", RoleSeller, ChannelMarketplace, TypePublic, CollectorMarketplace))
	later := deal("tx-4", RoleBuyer, ChannelMarketplace, TypePublic, CollectorMSP)
	later.Term.Start, later.Term.End = "2026-05", "2026-06"
	record(t, r, later)
	foreign := deal("tx-5", RoleBuyer, ChannelMarketplace, TypePublic, CollectorMSP)
	foreign.Agreement = "agr-2"
	record(t, r, foreign)
	write(t, r, "tx-1", "2026-03", 1, exampleFacts())
	write(t, r, "tx-2", "2026-03", 1, Facts{Lines: []Line{line("e1", KindOther, "5", unknown())}})
	write(t, r, "tx-3", "2026-03", 1, Facts{Lines: []Line{line("s1", KindSoftware, "7", notApplied())}})
	write(t, r, "tx-5", "2026-03", 1, Facts{Lines: []Line{line("f1", KindSoftware, "9", notApplied())}})
	sealed(t, r, "tx-1", "2026-03", 1)
	sealed(t, r, "tx-5", "2026-03", 1)
	v, err := r.AgreementCharges("ws", "aws", "agr-1", "2026-03")
	if err != nil || v.Complete || len(v.Entries) != 3 || v.Entries[1].State != CurrentNotJudged || v.Entries[2].State != CurrentNotJudged {
		t.Fatalf("incomplete: %v %+v", err, v)
	}
	sealed(t, r, "tx-2", "2026-03", 1)
	sealed(t, r, "tx-3", "2026-03", 1)
	v, _ = r.AgreementCharges("ws", "aws", "agr-1", "2026-03")
	want := []GroupSums{
		{RoleBuyer, "EUR", KindSums{"0", "0", "5"}},
		{RoleBuyer, "USD", KindSums{"800", "200", "0"}},
		{RoleSeller, "USD", KindSums{"7", "0", "0"}},
	}
	if !v.Complete || !reflect.DeepEqual(v.Groups, want) || v.Entries[0].Sums != (KindSums{"800", "200", "0"}) || v.Entries[0].Role != RoleBuyer || v.Entries[1].Currency != "EUR" {
		t.Fatalf("complete: %+v", v)
	}
	if v, _ := r.AgreementCharges("ws", "aws", "agr-1", "2026-05"); v.Complete || len(v.Entries) != 4 || v.Entries[3].State != "" {
		t.Fatalf("a month without statements: %+v", v)
	}
	if _, err := r.AgreementCharges("ws", "aws", "agr-9", "2026-03"); kind(err) != "not_found" {
		t.Fatal("no agreement")
	}
	if _, err := r.AgreementCharges("ws", "aws", "agr-1", "2027-03"); kind(err) != "not_found" {
		t.Fatal("no transaction in the month")
	}
	if _, err := r.AgreementCharges("other", "aws", "agr-1", "2026-03"); kind(err) != "not_found" {
		t.Fatal("another workspace")
	}
}
