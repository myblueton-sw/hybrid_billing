package balances

import (
	"strings"
	"testing"
	"time"
)

func TestEveryRefusalReason(t *testing.T) {
	r, clock := openWith(t, testLimits())
	opened(t, r, cash("cash", "cust-1", "USD"))
	opened(t, r, cash("eur", "cust-1", "EUR"))
	opened(t, r, cash("cust2", "cust-2", "USD"))
	entity := cash("entity2", "cust-1", "USD")
	entity.Entity = "entity-2"
	opened(t, r, entity)
	external := cash("ext", "cust-3", "USD")
	external.Authority, external.Ledger = AuthorityExternal, "erp-1"
	opened(t, r, external)
	opened(t, r, credit("pending", "cust-1", "USD"))
	if _, err := r.RecordOpening("ws", Opening{ID: "o1", Account: "pending", PriorLedger: "l", AsOf: "2026-09-30", Amount: "5", Evidence: evidence, MigrationKey: "m", IncludedFrom: "2026-09-01"}, "clerk-1"); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"cash", "eur", "cust2", "entity2"} {
		funded(t, r, Increase{ID: "f-" + id, Account: id, Kind: IncreaseCashReceipt, Amount: "100", Reference: "rcpt-" + id})
	}
	reserved(t, r, apply("held", "cash", "inv", "80", "30", false))
	cases := []struct {
		app  Application
		want string
	}{
		{apply("x1", "ext", "inv", "80", "1", false), ReasonExternalAuthority},
		{apply("x2", "pending", "inv", "80", "1", false), ReasonOpeningPending},
		{apply("x3", "cust2", "inv", "80", "1", false), ReasonCustomerMismatch},
		{apply("x4", "entity2", "inv", "80", "1", false), ReasonCustomerMismatch},
		{apply("x5", "eur", "inv", "80", "1", false), ReasonCurrencyMismatch},
		{apply("x6", "cash", "inv", "81", "1", false), ReasonTargetMismatch},
		{apply("x7", "cash", "inv", "80", "50.00000001", false), ReasonExceedsTarget},
		{apply("x8", "cash", "inv-2", "100", "70.00000001", false), ReasonInsufficient},
	}
	for _, c := range cases {
		if got := refused(t, r, c.app); got != c.want {
			t.Errorf("%s: %s, want %s", c.app.ID, got, c.want)
		}
	}
	if v := reserved(t, r, apply("x7", "cash", "inv", "80", "50.00000001", true)); v.Applied != "50" {
		t.Fatalf("partial on the target: %+v", v)
	}
	if got := refused(t, r, apply("x9", "cash", "inv", "80", "1", true)); got != ReasonTargetSettled {
		t.Fatalf("settled: %s", got)
	}
	if v := reserved(t, r, apply("x8", "cash", "inv-2", "100", "70.00000001", true)); v.Applied != "20" {
		t.Fatalf("partial on the balance: %+v", v)
	}
	if got := refused(t, r, apply("x10", "cash", "inv-3", "5", "1", true)); got != ReasonNothingAvailable {
		t.Fatalf("nothing available: %s", got)
	}
	clock.advance(2 * time.Hour)
	if v := reserved(t, r, apply("x3", "cust2", "inv", "60", "60", false)); v.Applied != "60" {
		t.Fatalf("a target is set again once nothing is active: %+v", v)
	}
	r.Confirm("ws", "x3", "runner")
	reversedBy(t, r, Reversal{ID: "rv", Kind: ReverseApplication, Record: "x3", Amount: "60"})
	if got := refused(t, r, apply("x11", "cash", "inv", "60", "1", false)); got != ReasonCustomerMismatch {
		t.Fatalf("a target once confirmed stays fixed: %s", got)
	}
	if v := reserved(t, r, apply("x12", "cust2", "inv", "60", "60", false)); v.Applied != "60" {
		t.Fatalf("a reversed confirmation frees room: %+v", v)
	}
	for _, id := range []string{"cash", "eur", "cust2", "entity2", "pending"} {
		identity(t, r, "ws", id)
	}
}

func TestRefusalsInOrder(t *testing.T) {
	r, _ := openWith(t, testLimits())
	external := cash("ext", "cust-3", "USD")
	external.Authority, external.Ledger = AuthorityExternal, "erp-1"
	opened(t, r, external)
	opened(t, r, cash("cash", "cust-1", "USD"))
	if refused(t, r, apply("a", "ext", "inv", "1", "5", false)) != ReasonExternalAuthority {
		t.Fatal("external first")
	}
	if refused(t, r, apply("a", "cash", "inv", "1", "5", false)) != ReasonExceedsTarget {
		t.Fatal("the target before the balance")
	}
	if refused(t, r, apply("a", "cash", "inv", "10", "5", false)) != ReasonNothingAvailable {
		t.Fatal("nothing available before insufficient")
	}
	funded(t, r, receipt("r", "cash", "4"))
	if refused(t, r, apply("a", "cash", "inv", "10", "5", false)) != ReasonInsufficient {
		t.Fatal("insufficient")
	}
	if v := reserved(t, r, apply("a", "cash", "inv", "10", "5", true)); v.Applied != "4" {
		t.Fatalf("a refused request sent again: %+v", v)
	}
	if v := reserved(t, r, apply("a", "cash", "inv", "10", "5", true)); v.Applied != "4" || v.State != StateReserved {
		t.Fatalf("an identical repeat: %+v", v)
	}
	changed := apply("a", "cash", "inv", "10", "5", true)
	changed.Lifetime = 60
	if _, err := r.Reserve("ws", changed, "runner"); kind(err) != "conflict" {
		t.Fatalf("another lifetime: %v", err)
	}
	if _, err := r.Reserve("ws", apply("a", "cash", "inv", "10", "5", true), "another"); err != nil {
		t.Fatalf("another actor: %v", err)
	}
}

func TestErrorKindsInOrder(t *testing.T) {
	limits := testLimits()
	limits.Applications = 2
	r, _ := openWith(t, limits)
	opened(t, r, cash("cash", "cust-1", "USD"))
	funded(t, r, receipt("r", "cash", "10"))
	bad := apply("a", "nowhere", "inv", "0", "1", false)
	if _, err := r.Reserve("ws", bad, "runner"); kind(err) != "invalid" {
		t.Fatalf("form before lookup: %v", err)
	}
	bad.Target.Amount = "1"
	if _, err := r.Reserve("ws", bad, "runner"); kind(err) != "not_found" {
		t.Fatalf("lookup: %v", err)
	}
	applied(t, r, apply("a", "cash", "inv", "1", "1", false))
	if _, err := r.Reserve("ws", apply("a", "cash", "inv", "2", "1", false), "runner"); kind(err) != "conflict" {
		t.Fatalf("conflict: %v", err)
	}
	reserved(t, r, apply("b", "cash", "inv-b", "1", "1", false))
	if _, err := r.Reserve("ws", apply("c", "cash", "inv-c", "1", "1", false), "runner"); kind(err) != "limit" {
		t.Fatalf("room: %v", err)
	}
	if got := refused(t, r, apply("c", "cash", "inv", "1", "1", false)); got != ReasonTargetSettled {
		t.Fatalf("a refusal before room: %s", got)
	}
	imported := apply("c", "cash", "inv-c", "1", "1", false)
	imported.Import = Import{Ledger: "l", Transaction: "t", Occurred: "2026-11-15"}
	if _, err := r.Reserve("ws", imported, "runner"); kind(err) != "rejected" {
		t.Fatalf("a rule before room: %v", err)
	}
	for _, form := range []Application{
		{ID: "z", Account: "cash", Target: Target{Document: "d", Line: "1", Amount: "1"}, Requested: "0", Lifetime: 1},
		{ID: "z", Account: "cash", Target: Target{Document: "d", Line: "1", Amount: "1"}, Requested: "1", Lifetime: 0},
		{ID: "z", Account: "cash", Target: Target{Document: "d", Line: "1", Amount: "1"}, Requested: "1", Lifetime: 86401},
		{ID: "z", Account: "cash", Target: Target{Document: "d", Line: "", Amount: "1"}, Requested: "1", Lifetime: 1},
		{ID: "z", Account: "cash", Target: Target{Document: "d", Line: "1", Amount: "-1"}, Requested: "1", Lifetime: 1},
		{ID: "z", Account: "cash", Target: Target{Document: "d", Line: "1", Amount: "1"}, Requested: "1", Lifetime: 1, Import: Import{Ledger: "l"}},
		{ID: "z", Account: "cash", Target: Target{Document: "d", Line: "1", Amount: "1"}, Requested: "1", Lifetime: 1, Import: Import{Ledger: "l", Transaction: "t", Occurred: "2026-02-30"}},
		{ID: "z", Account: "cash", Target: Target{Document: "d", Line: "1", Amount: "1"}, Requested: "0.000000001", Lifetime: 1},
		{ID: "z", Account: "cash", Target: Target{Document: "d", Line: "1", Amount: "1"}, Requested: strings.Repeat("1", 65), Lifetime: 1},
	} {
		if _, err := r.Reserve("ws", form, "runner"); kind(err) != "invalid" {
			t.Errorf("form %+v: %v", form, err)
		}
	}
	if _, err := r.Reserve("ws", apply("ok", "cash", "d", "1", "1", false), ""); kind(err) != "invalid" {
		t.Errorf("no actor: %v", err)
	}
	if v, err := r.Reserve("ws", Application{ID: "edge", Account: "cash", Target: Target{Document: "d", Line: "1", Amount: "1"}, Requested: "1", Lifetime: 86400}, "runner"); kind(err) != "limit" {
		t.Errorf("the lifetime bound: %+v %v", v, err)
	}
}

func TestAccountsAndIncreaseRules(t *testing.T) {
	r, clock := openWith(t, testLimits())
	a := cash("cash", "cust-1", "USD")
	opened(t, r, a)
	if _, err := r.OpenAccount("ws", a, "someone"); err != nil {
		t.Fatalf("a repeat: %v", err)
	}
	changed := a
	changed.Entity = "entity-9"
	if _, err := r.OpenAccount("ws", changed, "clerk-1"); kind(err) != "conflict" {
		t.Fatalf("a conflict: %v", err)
	}
	twin := a
	twin.ID = "twin"
	if _, err := r.OpenAccount("ws", twin, "clerk-1"); kind(err) != "rejected" {
		t.Fatalf("two ledgers for one balance: %v", err)
	}
	if _, err := r.OpenAccount("ws2", twin, "clerk-1"); err != nil {
		t.Fatalf("another workspace: %v", err)
	}
	for _, bad := range []Account{
		{ID: "b", Customer: "c", Entity: "e", Currency: "usd", Kind: KindCash, Authority: AuthorityPlatform},
		{ID: "b", Customer: "c", Entity: "e", Currency: "USD", Kind: "deposit", Authority: AuthorityPlatform},
		{ID: "b", Customer: "c", Entity: "e", Currency: "USD", Kind: KindCash, Authority: "erp"},
		{ID: "b", Customer: "c", Entity: "e", Currency: "USD", Kind: KindCash, Authority: AuthorityExternal},
		{ID: "b", Customer: "c", Entity: "e", Currency: "USD", Kind: KindCash, Authority: AuthorityPlatform, Ledger: "x"},
		{ID: "b", Customer: "", Entity: "e", Currency: "USD", Kind: KindCash, Authority: AuthorityPlatform},
		{ID: "b", Customer: "c", Entity: "e e", Currency: "USD", Kind: KindCash, Authority: AuthorityPlatform},
	} {
		if _, err := r.OpenAccount("ws", bad, "clerk-1"); kind(err) != "invalid" {
			t.Errorf("account %+v: %v", bad, err)
		}
	}
	if _, err := r.RecordIncrease("ws", Increase{ID: "p", Account: "cash", Kind: IncreasePromotion, Amount: "1", Reference: "x", ExpiresOn: "2026-12-01"}, "clerk-1"); kind(err) != "rejected" {
		t.Errorf("a credit kind on a cash account: %v", err)
	}
	opened(t, r, credit("credit", "cust-1", "USD"))
	if _, err := r.RecordIncrease("ws", receipt("q", "credit", "1"), "clerk-1"); kind(err) != "rejected" {
		t.Errorf("a cash kind on a credit account: %v", err)
	}
	if _, err := r.RecordIncrease("ws", Increase{ID: "p", Account: "credit", Kind: IncreasePromotion, Amount: "1", Reference: "x"}, "clerk-1"); kind(err) != "invalid" {
		t.Errorf("a promotion without expiry: %v", err)
	}
	for _, bad := range []Increase{
		{ID: "p", Account: "credit", Kind: IncreaseSLACredit, Amount: "0", Reference: "x"},
		{ID: "p", Account: "credit", Kind: "gift", Amount: "1", Reference: "x"},
		{ID: "p", Account: "credit", Kind: IncreaseSLACredit, Amount: "1", Reference: ""},
		{ID: "p", Account: "credit", Kind: IncreaseSLACredit, Amount: "1", Reference: "x", ExpiresOn: "2026-13-01"},
	} {
		if _, err := r.RecordIncrease("ws", bad, "clerk-1"); kind(err) != "invalid" {
			t.Errorf("increase %+v: %v", bad, err)
		}
	}
	past := promotion("old", "credit", "1", "2026-11-13")
	if _, err := r.RecordIncrease("ws", past, "clerk-1"); kind(err) != "rejected" {
		t.Errorf("an expired date at record: %v", err)
	}
	soon := promotion("soon", "credit", "1", "2026-11-14")
	if _, err := r.RecordIncrease("ws", soon, "clerk-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ApproveIncrease("ws", "soon", "clerk-1"); kind(err) != "rejected" {
		t.Errorf("the author approves: %v", err)
	}
	clock.set(time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC))
	if _, err := r.ApproveIncrease("ws", "soon", "lead-1"); kind(err) != "rejected" {
		t.Errorf("an expired date at approval: %v", err)
	}
	sla := Increase{ID: "sla", Account: "credit", Kind: IncreaseSLACredit, Amount: "7", Reference: "inc-1"}
	funded(t, r, sla)
	if _, err := r.ApproveIncrease("ws", "sla", "lead-1"); err != nil {
		t.Errorf("a repeat approval: %v", err)
	}
	if _, err := r.ApproveIncrease("ws", "sla", "lead-2"); kind(err) != "conflict" {
		t.Errorf("another approver: %v", err)
	}
	if _, err := r.RecordIncrease("ws", sla, "clerk-9"); err != nil {
		t.Errorf("an identical repeat: %v", err)
	}
	sla.Amount = "7.0"
	if v, err := r.RecordIncrease("ws", sla, "clerk-9"); err != nil || v.Increase.Amount != "7" {
		t.Errorf("a canonical repeat: %+v %v", v, err)
	}
	sla.Amount = "8"
	if _, err := r.RecordIncrease("ws", sla, "clerk-1"); kind(err) != "conflict" {
		t.Errorf("a changed increase: %v", err)
	}
	ext := cash("ext", "cust-3", "USD")
	ext.Authority, ext.Ledger = AuthorityExternal, "erp-1"
	opened(t, r, ext)
	if _, err := r.RecordIncrease("ws", receipt("e", "ext", "1"), "clerk-1"); kind(err) != "rejected" {
		t.Errorf("an increase on an external account: %v", err)
	}
	if _, err := r.RecordOpening("ws", Opening{ID: "oe", Account: "ext", PriorLedger: "l", AsOf: "2026-09-30", Amount: "5", Evidence: evidence, MigrationKey: "me", IncludedFrom: "2026-09-01"}, "clerk-1"); kind(err) != "rejected" {
		t.Errorf("an opening on an external account: %v", err)
	}
	if _, err := r.Statement("ws", "ext"); kind(err) != "rejected" {
		t.Errorf("a statement of an external account: %v", err)
	}
	if v, err := r.Increase("ws", "sla"); err != nil || v.State != StateApproved || v.Approver != "lead-1" {
		t.Errorf("read: %+v %v", v, err)
	}
}

func TestOpeningRules(t *testing.T) {
	r, clock := openWith(t, testLimits())
	opened(t, r, cash("cash", "cust-1", "USD"))
	base := Opening{ID: "o", Account: "cash", PriorLedger: "l", AsOf: "2026-09-30", Amount: "0", Evidence: evidence, MigrationKey: "m", IncludedFrom: "2026-09-30"}
	for _, bad := range []Opening{
		func() Opening { o := base; o.IncludedFrom = "2026-10-01"; return o }(),
		func() Opening { o := base; o.Evidence = strings.ToUpper(evidence); return o }(),
		func() Opening { o := base; o.Evidence = evidence[:63]; return o }(),
		func() Opening { o := base; o.Amount = "-1"; return o }(),
		func() Opening { o := base; o.MigrationKey = ""; return o }(),
		func() Opening { o := base; o.ExpiresOn = "2026-02-29"; return o }(),
	} {
		if _, err := r.RecordOpening("ws", bad, "clerk-1"); kind(err) != "invalid" {
			t.Errorf("opening %+v: %v", bad, err)
		}
	}
	if _, err := r.RecordOpening("ws", func() Opening { o := base; o.Account = "none"; return o }(), "clerk-1"); kind(err) != "not_found" {
		t.Errorf("an unknown account: %v", err)
	}
	if _, err := r.RecordOpening("ws", func() Opening { o := base; o.ExpiresOn = "2026-11-13"; return o }(), "clerk-1"); kind(err) != "rejected" {
		t.Errorf("an expired opening at record: %v", err)
	}
	expiring := base
	expiring.ExpiresOn = "2026-11-14"
	expiring.Amount = "9"
	if _, err := r.RecordOpening("ws", expiring, "clerk-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.RecordOpening("ws", base, "clerk-1"); kind(err) != "conflict" {
		t.Errorf("another opening under its identifier: %v", err)
	}
	clock.set(time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC))
	if _, err := r.ApproveOpening("ws", "o", "lead-1"); kind(err) != "rejected" {
		t.Errorf("an expired opening at approval: %v", err)
	}
	r.WithdrawOpening("ws", "o", "clerk-1")
	zero := base
	zero.ID = "o2"
	if _, err := r.RecordOpening("ws", zero, "clerk-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ApproveOpening("ws", "o2", "lead-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ApproveOpening("ws", "o2", "lead-2"); kind(err) != "conflict" {
		t.Errorf("another approver: %v", err)
	}
	if lots, _ := r.Lots("ws", "cash"); len(lots) != 0 {
		t.Errorf("a zero opening made a lot: %+v", lots)
	}
	if _, err := r.Reverse("ws", Reversal{ID: "rz", Kind: ReverseOpening, Record: "o2", Amount: "1"}, "clerk-1"); kind(err) != "rejected" {
		t.Errorf("reversing a zero opening: %v", err)
	}
	if v, err := r.Opening("ws", "o2"); err != nil || v.State != StateApproved || v.Approver != "lead-1" {
		t.Errorf("read: %+v %v", v, err)
	}
	if v, _ := r.Opening("ws", "o"); v.State != StateWithdrawn {
		t.Errorf("withdrawn: %+v", v)
	}
	if st := identity(t, r, "ws", "cash"); st.Opening != "0" || st.Closing != "0" {
		t.Errorf("statement: %+v", st)
	}
}

func TestReversalRules(t *testing.T) {
	r, _ := openWith(t, testLimits())
	opened(t, r, cash("cash", "cust-1", "USD"))
	o := Opening{ID: "o", Account: "cash", PriorLedger: "l", AsOf: "2026-09-30", Amount: "20", Evidence: evidence, MigrationKey: "m", IncludedFrom: "2026-09-01"}
	r.RecordOpening("ws", o, "clerk-1")
	r.ApproveOpening("ws", "o", "lead-1")
	funded(t, r, receipt("r1", "cash", "30"))
	applied(t, r, apply("a1", "cash", "inv", "25", "25", false))
	reserved(t, r, apply("a2", "cash", "inv-2", "20", "20", false))
	pending := Reversal{ID: "p1", Kind: ReverseApplication, Record: "a1", Amount: "25"}
	if _, err := r.Reverse("ws", pending, "clerk-1"); err != nil {
		t.Fatal(err)
	}
	if st := identity(t, r, "ws", "cash"); st.ApplicationReversals != "0" || st.Available != "5" {
		t.Fatalf("a pending reversal freed money: %+v", st)
	}
	if _, err := r.Reverse("ws", Reversal{ID: "p2", Kind: ReverseApplication, Record: "a1", Amount: "25"}, "clerk-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ApproveReversal("ws", "p1", "clerk-1"); kind(err) != "rejected" {
		t.Fatalf("the author approves: %v", err)
	}
	if _, err := r.ApproveReversal("ws", "p1", "lead-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ApproveReversal("ws", "p1", "lead-1"); err != nil {
		t.Fatalf("a repeat approval: %v", err)
	}
	if _, err := r.ApproveReversal("ws", "p1", "lead-2"); kind(err) != "conflict" {
		t.Fatalf("another approver: %v", err)
	}
	if _, err := r.ApproveReversal("ws", "p2", "lead-1"); kind(err) != "rejected" {
		t.Fatalf("the same reversal applied twice: %v", err)
	}
	if _, err := r.Reverse("ws", Reversal{ID: "p3", Kind: ReverseApplication, Record: "a1", Amount: "0.00000001"}, "clerk-1"); kind(err) != "rejected" {
		t.Fatalf("beyond what remains: %v", err)
	}
	if _, err := r.Reverse("ws", pending, "someone"); err != nil {
		t.Fatalf("an identical repeat: %v", err)
	}
	changed := pending
	changed.Amount = "24"
	if _, err := r.Reverse("ws", changed, "clerk-1"); kind(err) != "conflict" {
		t.Fatalf("a changed reversal: %v", err)
	}
	if _, err := r.Reverse("ws", Reversal{ID: "p4", Kind: ReverseApplication, Record: "a2", Amount: "1"}, "clerk-1"); kind(err) != "rejected" {
		t.Fatalf("reversing a reservation: %v", err)
	}
	if _, err := r.Reverse("ws", Reversal{ID: "p4", Kind: ReverseIncrease, Record: "missing", Amount: "1"}, "clerk-1"); kind(err) != "not_found" {
		t.Fatalf("an unknown record: %v", err)
	}
	if _, err := r.Reverse("ws", Reversal{ID: "p4", Kind: ReverseOpening, Record: "r1", Amount: "1"}, "clerk-1"); kind(err) != "not_found" {
		t.Fatalf("an increase named as an opening: %v", err)
	}
	if _, err := r.Reverse("ws", Reversal{ID: "p4", Kind: "refund", Record: "r1", Amount: "1"}, "clerk-1"); kind(err) != "invalid" {
		t.Fatalf("an unknown kind: %v", err)
	}
	lots, _ := r.Lots("ws", "cash")
	if lots[0].Remainder != "20" || lots[0].Held != "0" || lots[1].Remainder != "30" || lots[1].Held != "20" {
		t.Fatalf("the reversal returned to the lots it used: %+v", lots)
	}
	if _, err := r.Reverse("ws", Reversal{ID: "o1", Kind: ReverseIncrease, Record: "r1", Amount: "10.00000001"}, "clerk-1"); kind(err) != "rejected" {
		t.Fatalf("reversing held money: %v", err)
	}
	r.Release("ws", "a2", "runner")
	if _, err := r.Reverse("ws", Reversal{ID: "i1", Kind: ReverseIncrease, Record: "r1", Amount: "30"}, "clerk-1"); err != nil {
		t.Fatal(err)
	}
	reserved(t, r, apply("a3", "cash", "inv-3", "21", "21", false))
	if _, err := r.ApproveReversal("ws", "i1", "lead-1"); kind(err) != "rejected" {
		t.Fatalf("an approval judged again: %v", err)
	}
	r.Release("ws", "a3", "runner")
	if _, err := r.ApproveReversal("ws", "i1", "lead-1"); err != nil {
		t.Fatal(err)
	}
	st := identity(t, r, "ws", "cash")
	if st.LotReversals != "30" || st.Closing != "20" || st.Available != "20" || st.Applications != "25" || st.ApplicationReversals != "25" {
		t.Fatalf("statement: %+v", st)
	}
	if v, err := r.Reversal("ws", "i1"); err != nil || v.State != StateEffective || v.Approver != "lead-1" || v.Author != "clerk-1" {
		t.Fatalf("read: %+v %v", v, err)
	}
	if v, _ := r.Application("ws", "a1"); v.Reversed != "25" || v.State != StateConfirmed {
		t.Fatalf("application: %+v", v)
	}
}

func TestConfirmAndRelease(t *testing.T) {
	r, clock := openWith(t, testLimits())
	opened(t, r, cash("cash", "cust-1", "USD"))
	funded(t, r, receipt("r", "cash", "10"))
	reserved(t, r, apply("a", "cash", "inv", "5", "5", false))
	if v, err := r.Confirm("ws", "a", "runner-9"); err != nil || v.ConfirmedBy != "runner-9" {
		t.Fatalf("confirm: %+v %v", v, err)
	}
	if v, err := r.Confirm("ws", "a", "other"); err != nil || v.ConfirmedBy != "runner-9" {
		t.Fatalf("a repeat confirm: %+v %v", v, err)
	}
	if _, err := r.Release("ws", "a", "runner"); kind(err) != "rejected" {
		t.Fatalf("releasing a confirmed application: %v", err)
	}
	reserved(t, r, apply("b", "cash", "inv-b", "5", "5", false))
	if v, err := r.Release("ws", "b", "runner-8"); err != nil || v.State != StateReleased || v.ReleasedBy != "runner-8" {
		t.Fatalf("release: %+v %v", v, err)
	}
	if _, err := r.Release("ws", "b", "runner"); err != nil {
		t.Fatalf("a repeat release: %v", err)
	}
	if _, err := r.Confirm("ws", "b", "runner"); kind(err) != "rejected" {
		t.Fatalf("confirming a released application: %v", err)
	}
	reserved(t, r, apply("c", "cash", "inv-c", "5", "5", false))
	clock.advance(time.Hour)
	if _, err := r.Release("ws", "c", "runner"); kind(err) != "rejected" {
		t.Fatalf("releasing an expired application: %v", err)
	}
	if _, err := r.Confirm("ws", "missing", "runner"); kind(err) != "not_found" {
		t.Fatalf("unknown: %v", err)
	}
	if _, err := r.Confirm("ws2", "a", "runner"); kind(err) != "not_found" {
		t.Fatalf("another workspace: %v", err)
	}
	if st := identity(t, r, "ws", "cash"); st.Available != "5" || st.Applications != "5" {
		t.Fatalf("statement: %+v", st)
	}
}

func TestImportsOfApplicationsReopenWhenInactive(t *testing.T) {
	r, clock := openWith(t, testLimits())
	opened(t, r, cash("cash", "cust-1", "USD"))
	funded(t, r, receipt("r", "cash", "10"))
	past := apply("p1", "cash", "inv", "3", "3", false)
	past.Import = Import{Ledger: "legacy", Transaction: "t1", Occurred: "2026-10-01"}
	reserved(t, r, past)
	second := past
	second.ID = "p2"
	if _, err := r.Reserve("ws", second, "runner"); kind(err) != "conflict" {
		t.Fatalf("an import reserved twice: %v", err)
	}
	clock.advance(time.Hour)
	reserved(t, r, second)
	r.Confirm("ws", "p2", "runner")
	third := past
	third.ID = "p3"
	if _, err := r.Reserve("ws", third, "runner"); kind(err) != "conflict" {
		t.Fatalf("a confirmed import: %v", err)
	}
	inc := receipt("ri", "cash", "3")
	inc.Import = past.Import
	if _, err := r.RecordIncrease("ws", inc, "clerk-1"); kind(err) != "conflict" {
		t.Fatalf("an application's import reused by an increase: %v", err)
	}
}

func TestLimits(t *testing.T) {
	limits := Limits{Accounts: 1, Openings: 1, Increases: 1, Applications: 2, Targets: 1, Reversals: 1, Workspace: 1}
	r, _ := openWith(t, limits)
	opened(t, r, cash("cash", "cust-1", "USD"))
	if _, err := r.OpenAccount("ws", cash("two", "cust-2", "USD"), "clerk-1"); kind(err) != "limit" {
		t.Errorf("accounts: %v", err)
	}
	if _, err := r.RecordOpening("ws", Opening{ID: "o", Account: "cash", PriorLedger: "l", AsOf: "2026-09-30", Amount: "1", Evidence: evidence, MigrationKey: "m", IncludedFrom: "2026-09-01"}, "c"); err != nil {
		t.Fatal(err)
	}
	r.WithdrawOpening("ws", "o", "c")
	if _, err := r.RecordOpening("ws", Opening{ID: "o2", Account: "cash", PriorLedger: "l", AsOf: "2026-09-30", Amount: "1", Evidence: evidence, MigrationKey: "m", IncludedFrom: "2026-09-01"}, "c"); kind(err) != "limit" {
		t.Errorf("openings: %v", err)
	}
	funded(t, r, receipt("r", "cash", "10"))
	if _, err := r.RecordIncrease("ws", receipt("r2", "cash", "1"), "c"); kind(err) != "limit" {
		t.Errorf("increases: %v", err)
	}
	applied(t, r, apply("a", "cash", "inv", "5", "2", false))
	if _, err := r.Reserve("ws", apply("b", "cash", "inv-2", "5", "2", false), "c"); kind(err) != "limit" {
		t.Errorf("targets: %v", err)
	}
	applied(t, r, apply("b", "cash", "inv", "5", "2", false))
	if _, err := r.Reserve("ws", apply("c", "cash", "inv", "5", "1", false), "c"); kind(err) != "limit" {
		t.Errorf("applications: %v", err)
	}
	if _, err := r.Reverse("ws", Reversal{ID: "v1", Kind: ReverseApplication, Record: "a", Amount: "1"}, "c"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Reverse("ws", Reversal{ID: "v2", Kind: ReverseApplication, Record: "a", Amount: "1"}, "c"); kind(err) != "limit" {
		t.Errorf("reversals per record: %v", err)
	}
	if _, err := r.Reverse("ws", Reversal{ID: "v2", Kind: ReverseApplication, Record: "b", Amount: "1"}, "c"); kind(err) != "limit" {
		t.Errorf("reversals per workspace: %v", err)
	}
	for _, bad := range []Limits{
		{Accounts: 0, Openings: 1, Increases: 1, Applications: 1, Targets: 1, Reversals: 1, Workspace: 1},
		{Accounts: 100001, Openings: 1, Increases: 1, Applications: 1, Targets: 1, Reversals: 1, Workspace: 1},
		{Accounts: 1, Openings: 101, Increases: 1, Applications: 1, Targets: 1, Reversals: 1, Workspace: 1},
		{Accounts: 1, Openings: 1, Increases: 100001, Applications: 1, Targets: 1, Reversals: 1, Workspace: 1},
		{Accounts: 1, Openings: 1, Increases: 1, Applications: 1000001, Targets: 1, Reversals: 1, Workspace: 1},
		{Accounts: 1, Openings: 1, Increases: 1, Applications: 1, Targets: 1000001, Reversals: 1, Workspace: 1},
		{Accounts: 1, Openings: 1, Increases: 1, Applications: 1, Targets: 1, Reversals: 1001, Workspace: 1},
		{Accounts: 1, Openings: 1, Increases: 1, Applications: 1, Targets: 1, Reversals: 1, Workspace: 1000001},
		{Accounts: 1, Openings: 0, Increases: 1, Applications: 1, Targets: 1, Reversals: 1, Workspace: 1},
		{Accounts: 1, Openings: 1, Increases: 0, Applications: 1, Targets: 1, Reversals: 1, Workspace: 1},
		{Accounts: 1, Openings: 1, Increases: 1, Applications: 0, Targets: 1, Reversals: 1, Workspace: 1},
		{Accounts: 1, Openings: 1, Increases: 1, Applications: 1, Targets: 0, Reversals: 1, Workspace: 1},
		{Accounts: 1, Openings: 1, Increases: 1, Applications: 1, Targets: 1, Reversals: 0, Workspace: 1},
		{Accounts: 1, Openings: 1, Increases: 1, Applications: 1, Targets: 1, Reversals: 1, Workspace: 0},
	} {
		if _, err := NewRegistry(testProfile(t), bad, time.Now); kind(err) != "invalid" {
			t.Errorf("limits %+v: %v", bad, err)
		}
	}
	top := Limits{Accounts: 100000, Openings: 100, Increases: 100000, Applications: 1000000, Targets: 1000000, Reversals: 1000, Workspace: 1000000}
	if _, err := NewRegistry(testProfile(t), top, time.Now); err != nil {
		t.Errorf("the top limits: %v", err)
	}
	if _, err := NewRegistry(testProfile(t), top, nil); kind(err) != "invalid" {
		t.Errorf("no clock: %v", err)
	}
}
