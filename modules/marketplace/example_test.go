package marketplace

import "testing"

func TestTheSourceExample(t *testing.T) {
	r, s := example(t)
	approve(t, r, terms("pol", "1", "tx-1", "0.2", []string{KindSoftware}, Excluded))
	v := compute(t, s, policyFor(t, r, "tx-1", "2026-03"))
	if got := amounts(v); got != "infra-1=200/-0 sw-1=800/-0 | total 1000 charged 1000 margin 0" {
		t.Fatalf("a private offer discounted again: %s", got)
	}
	if len(v.Recognitions) != 1 || v.Recognitions[0].Amount != "800" || v.Recognitions[0].State != Applied {
		t.Fatalf("recognition: %+v", v.Recognitions)
	}

	r2, s2 := example(t)
	approve(t, r2, terms("pol", "1", "tx-1", "0.2", []string{KindSoftware, KindInfrastructure}, Included))
	if got := amounts(compute(t, s2, policyFor(t, r2, "tx-1", "2026-03"))); got != "infra-1=160/-40 sw-1=640/-160 | total 800 charged 1000 margin -200" {
		t.Fatalf("an included contract discount: %s", got)
	}

	r3, s3 := example(t)
	approve(t, r3, terms("pol", "1", "tx-1", "0.2", []string{KindSoftware}, Included))
	if got := amounts(compute(t, s3, policyFor(t, r3, "tx-1", "2026-03"))); got != "infra-1=200/-0 sw-1=640/-160 | total 840 charged 1000 margin -160" {
		t.Fatalf("infrastructure outside the scope: %s", got)
	}
}

func TestARecognitionChangesNoAmount(t *testing.T) {
	r := open(t)
	record(t, r, buyerPrivate("tx-1"))
	facts := exampleFacts()
	write(t, r, "tx-1", "2026-03", 1, facts)
	write(t, r, "tx-1", "2026-04", 1, Facts{Lines: []Line{line("sw-2", KindSoftware, "800", applied("200")), line("infra-2", KindInfrastructure, "200", notApplied())}})
	approve(t, r, terms("pol", "1", "tx-1", "0.2", []string{KindSoftware, KindInfrastructure}, Excluded))
	with := compute(t, sealed(t, r, "tx-1", "2026-03", 1), policyFor(t, r, "tx-1", "2026-03"))
	without := compute(t, sealed(t, r, "tx-1", "2026-04", 1), policyFor(t, r, "tx-1", "2026-04"))
	if with.Total != without.Total || with.Total != "960" || len(without.Recognitions) != 0 {
		t.Fatalf("with %s without %s", amounts(with), amounts(without))
	}
}

func TestTheSettlementOfTheExample(t *testing.T) {
	r := open(t)
	record(t, r, deal("tx-r", RoleReseller, ChannelMarketplace, TypeChannelPrivate, CollectorMSP))
	base := SettlementFacts{State: SettlementExpected, Currency: "USD", Gross: "800", Fee: "24", Refund: "0", Tax: "0", Hold: "0", Net: "776"}
	v, err := r.RecordSettlement("ws", "tx-r", "set-1", 1, base)
	if err != nil || v.Judged != JudgedConsistent || v.Received || v.Facts.State != SettlementExpected {
		t.Fatalf("expected: %v %+v", err, v)
	}
	reported := base
	reported.State, reported.Report = SettlementReported, "rep-1"
	if v, err = r.RecordSettlement("ws", "tx-r", "set-1", 2, reported); err != nil || v.Received {
		t.Fatalf("reported: %v %+v", err, v)
	}
	short := reported
	short.State, short.Payout, short.Received = SettlementPaid, "pay-1", "775"
	if v, err = r.RecordSettlement("ws", "tx-r", "set-1", 3, short); err != nil || v.Received || v.Judged != JudgedHeld || len(v.Reasons) != 1 || v.Reasons[0] != ReasonReceivedDiffers {
		t.Fatalf("short payout: %v %+v", err, v)
	}
	r2 := open(t)
	record(t, r2, deal("tx-r", RoleReseller, ChannelMarketplace, TypeChannelPrivate, CollectorMSP))
	paid := short
	paid.Received = "776"
	if v, err = r2.RecordSettlement("ws", "tx-r", "set-1", 1, paid); err != nil || !v.Received || v.Judged != JudgedConsistent {
		t.Fatalf("paid: %v %+v", err, v)
	}
	if got, _ := r2.Settlement("ws", "tx-r", "set-1"); !got.Received || got.Facts.Net != "776" || got.Facts.Fee != "24" || got.Facts.Gross != "800" {
		t.Fatalf("read back: %+v", got)
	}
}
