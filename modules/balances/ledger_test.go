package balances

import (
	"math/big"
	"sync"
	"testing"
	"time"
)

func hundred(t testing.TB) (*Registry, *testClock) {
	t.Helper()
	r, clock := openWith(t, testLimits())
	opened(t, r, cash("cash", "cust-1", "USD"))
	funded(t, r, receipt("r1", "cash", "100"))
	return r, clock
}

func TestSixtyAndFiftyOnAHundred(t *testing.T) {
	for _, order := range [][2]string{{"60", "50"}, {"50", "60"}} {
		r, _ := hundred(t)
		first := applied(t, r, apply("a1", "cash", "inv-"+order[0], order[0], order[0], true))
		second := applied(t, r, apply("a2", "cash", "inv-"+order[1], order[1], order[1], true))
		total := new(ratSum).add(t, first.Applied).add(t, second.Applied)
		if total.text() != "100" || first.Applied != order[0] || second.Applied != map[string]string{"60": "40", "50": "50"}[order[0]] {
			t.Fatalf("order %v: %s and %s", order, first.Applied, second.Applied)
		}
		st := identity(t, r, "ws", "cash")
		if st.Closing != "0" || st.Applications != "100" || st.Available != "0" {
			t.Fatalf("statement: %+v", st)
		}
		if reason := refused(t, r, apply("a3", "cash", "inv-60", "60", "10", true)); order[0] == "50" && reason != ReasonNothingAvailable || order[0] == "60" && reason != ReasonTargetSettled {
			t.Fatalf("a third application: %s", reason)
		}
	}
}

func TestConcurrentApplicationsNeverExceedTheBalance(t *testing.T) {
	for round := 0; round < 20; round++ {
		r, _ := hundred(t)
		var wg sync.WaitGroup
		for index, amount := range []string{"60", "50"} {
			wg.Add(1)
			go func(index int, amount string) {
				defer wg.Done()
				id := []string{"a1", "a2"}[index]
				if _, err := r.Reserve("ws", apply(id, "cash", "inv-"+amount, amount, amount, true), "runner"); err != nil {
					t.Error(err)
				}
				if _, err := r.Confirm("ws", id, "runner"); err != nil {
					t.Error(err)
				}
			}(index, amount)
		}
		wg.Wait()
		a1, _ := r.Application("ws", "a1")
		a2, _ := r.Application("ws", "a2")
		unpaid := new(ratSum).add(t, "110").sub(t, a1.Applied).sub(t, a2.Applied)
		if unpaid.text() != "10" || identity(t, r, "ws", "cash").Closing != "0" {
			t.Fatalf("round %d: %+v %+v", round, a1, a2)
		}
	}
}

func TestOpeningBlocksTheTransactionsItIncludes(t *testing.T) {
	r, _ := openWith(t, testLimits())
	opened(t, r, cash("cash", "cust-1", "USD"))
	opened(t, r, credit("credit", "cust-1", "USD"))
	opened(t, r, credit("other", "cust-2", "USD"))
	o := Opening{ID: "o1", Account: "cash", PriorLedger: "legacy", AsOf: "2026-09-30", Amount: "250.50", Evidence: evidence, MigrationKey: "mig-1", IncludedFrom: "2026-01-01"}
	if _, err := r.RecordOpening("ws", o, "clerk-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.RecordIncrease("ws", receipt("early", "cash", "1"), "clerk-1"); kind(err) != "rejected" {
		t.Fatalf("an increase while the opening is pending: %v", err)
	}
	if reason := refused(t, r, apply("early", "cash", "inv", "1", "1", false)); reason != ReasonOpeningPending {
		t.Fatalf("a reservation while the opening is pending: %s", reason)
	}
	if _, err := r.ApproveOpening("ws", "o1", "clerk-1"); kind(err) != "rejected" {
		t.Fatalf("the author approves: %v", err)
	}
	if _, err := r.ApproveOpening("ws", "o1", "lead-1"); err != nil {
		t.Fatal(err)
	}
	if st := identity(t, r, "ws", "cash"); st.Opening != "250.5" || st.Closing != "250.5" {
		t.Fatalf("statement: %+v", st)
	}
	included := receipt("old", "cash", "40")
	for _, day := range []string{"2026-09-30", "2026-01-01", "2025-12-31"} {
		included.Import = Import{Ledger: "legacy", Transaction: "t-" + day, Occurred: day}
		if _, err := r.RecordIncrease("ws", included, "clerk-1"); kind(err) != "rejected" {
			t.Errorf("an included transaction of %s: %v", day, err)
		}
	}
	other := Increase{ID: "x1", Account: "other", Kind: IncreaseCreditDocument, Amount: "5", Reference: "cd-x1", Import: Import{Ledger: "legacy", Transaction: "t-x", Occurred: "2026-09-01"}}
	if _, err := r.RecordIncrease("ws", other, "clerk-1"); kind(err) != "rejected" {
		t.Errorf("another account's opening of the same ledger: %v", err)
	}
	other.Import.Ledger = "elsewhere"
	if _, err := r.RecordIncrease("ws", other, "clerk-1"); err != nil {
		t.Errorf("a ledger in no opening: %v", err)
	}
	later := receipt("new", "cash", "40")
	later.Import = Import{Ledger: "legacy", Transaction: "t-9", Occurred: "2026-10-01"}
	funded(t, r, later)
	again := receipt("again", "cash", "40")
	again.Import = later.Import
	if _, err := r.RecordIncrease("ws", again, "clerk-1"); kind(err) != "conflict" {
		t.Errorf("a second import of one transaction: %v", err)
	}
	moved := Increase{ID: "moved", Account: "credit", Kind: IncreaseCreditDocument, Amount: "40", Reference: "cd-9", Import: later.Import}
	if _, err := r.RecordIncrease("ws", moved, "clerk-1"); kind(err) != "conflict" {
		t.Errorf("the same import in another account: %v", err)
	}
	past := apply("past", "cash", "inv-old", "30", "30", false)
	past.Import = Import{Ledger: "legacy", Transaction: "t-a", Occurred: "2026-09-15"}
	if _, err := r.Reserve("ws", past, "runner"); kind(err) != "rejected" {
		t.Errorf("an included application: %v", err)
	}
	past.Import = Import{Ledger: "legacy", Transaction: "t-a", Occurred: "2026-11-15"}
	if _, err := r.Reserve("ws", past, "runner"); kind(err) != "rejected" {
		t.Errorf("an import after the registry's date: %v", err)
	}
	past.Import.Occurred = "2026-10-02"
	applied(t, r, past)
	past.Partial = true
	past.ID = "past-2"
	if _, err := r.Reserve("ws", past, "runner"); kind(err) != "invalid" {
		t.Errorf("a partial import: %v", err)
	}
	if st := identity(t, r, "ws", "cash"); st.Closing != "260.5" || st.Increases != "40" {
		t.Fatalf("statement: %+v", st)
	}
	second := o
	second.ID, second.MigrationKey = "o2", "mig-2"
	if _, err := r.RecordOpening("ws", second, "clerk-1"); kind(err) != "rejected" {
		t.Errorf("a second opening: %v", err)
	}
	if _, err := r.WithdrawOpening("ws", "o1", "clerk-1"); kind(err) != "rejected" {
		t.Errorf("withdrawing an approved opening: %v", err)
	}
}

func TestWithdrawalsFreeTheirKeys(t *testing.T) {
	r, _ := openWith(t, testLimits())
	opened(t, r, cash("cash", "cust-1", "USD"))
	o := Opening{ID: "o1", Account: "cash", PriorLedger: "legacy", AsOf: "2026-09-30", Amount: "10", Evidence: evidence, MigrationKey: "mig-1", IncludedFrom: "2026-01-01"}
	if _, err := r.RecordOpening("ws", o, "clerk-1"); err != nil {
		t.Fatal(err)
	}
	duplicate := o
	duplicate.ID = "o-dup"
	if _, err := r.RecordOpening("ws", duplicate, "clerk-1"); kind(err) != "conflict" {
		t.Fatalf("a migration key twice: %v", err)
	}
	for range 2 {
		if v, err := r.WithdrawOpening("ws", "o1", "clerk-2"); err != nil || v.State != StateWithdrawn || v.WithdrawnBy != "clerk-2" {
			t.Fatalf("withdraw: %+v %v", v, err)
		}
	}
	if _, err := r.ApproveOpening("ws", "o1", "lead-1"); kind(err) != "rejected" {
		t.Fatalf("approving a withdrawn opening: %v", err)
	}
	early := receipt("i0", "cash", "1")
	early.Import = Import{Ledger: "legacy", Transaction: "t-1", Occurred: "2026-09-01"}
	if _, err := r.RecordIncrease("ws", early, "clerk-1"); err != nil {
		t.Fatalf("a withdrawn opening still blocks: %v", err)
	}
	if v, err := r.WithdrawIncrease("ws", "i0", "clerk-1"); err != nil || v.State != StateWithdrawn {
		t.Fatalf("withdraw increase: %+v %v", v, err)
	}
	if _, err := r.WithdrawIncrease("ws", "i0", "clerk-3"); err != nil {
		t.Fatalf("a repeat withdrawal: %v", err)
	}
	if _, err := r.ApproveIncrease("ws", "i0", "lead-1"); kind(err) != "rejected" {
		t.Fatalf("approving a withdrawn increase: %v", err)
	}
	if _, err := r.RecordOpening("ws", o, "clerk-1"); err != nil {
		t.Fatalf("the same opening repeated: %v", err)
	}
	if _, err := r.RecordOpening("ws", duplicate, "clerk-1"); kind(err) != "rejected" {
		t.Fatalf("an opening after an increase: %v", err)
	}
	fresh, _ := openWith(t, testLimits())
	opened(t, fresh, cash("cash", "cust-1", "USD"))
	if _, err := fresh.RecordOpening("ws", o, "clerk-1"); err != nil {
		t.Fatal(err)
	}
	fresh.WithdrawOpening("ws", "o1", "clerk-1")
	if _, err := fresh.RecordOpening("ws", duplicate, "clerk-1"); err != nil {
		t.Fatalf("reusing the key of a withdrawn opening: %v", err)
	}
	same := receipt("i1", "cash", "5")
	if _, err := fresh.RecordIncrease("ws", same, "clerk-1"); kind(err) != "rejected" {
		t.Fatalf("an increase under a pending opening: %v", err)
	}
	fresh.ApproveOpening("ws", "o-dup", "lead-1")
	funded(t, fresh, same)
	copied := same
	copied.ID = "i2"
	if _, err := fresh.RecordIncrease("ws", copied, "clerk-1"); kind(err) != "conflict" {
		t.Fatalf("one receipt twice: %v", err)
	}
	other := copied
	other.Reference = "ref-i3"
	if _, err := fresh.RecordIncrease("ws", other, "clerk-1"); err != nil {
		t.Fatal(err)
	}
	fresh.WithdrawIncrease("ws", "i2", "clerk-1")
	third := other
	third.ID = "i3"
	if _, err := fresh.RecordIncrease("ws", third, "clerk-1"); err != nil {
		t.Fatalf("a withdrawn increase still holds its reference: %v", err)
	}
	if _, err := fresh.WithdrawIncrease("ws", "i1", "clerk-1"); kind(err) != "rejected" {
		t.Fatalf("withdrawing an approved increase: %v", err)
	}
	identity(t, fresh, "ws", "cash")
}

func TestExpiryOfLotsAndReservations(t *testing.T) {
	r, clock := openWith(t, testLimits())
	opened(t, r, credit("credit", "cust-1", "USD"))
	funded(t, r, Increase{ID: "doc", Account: "credit", Kind: IncreaseCreditDocument, Amount: "30", Reference: "cd-1"})
	funded(t, r, promotion("late", "credit", "20", "2026-12-31"))
	funded(t, r, promotion("soon", "credit", "50", "2026-11-20"))
	first := applied(t, r, apply("a1", "credit", "inv-1", "40", "40", false))
	if first.Applied != "40" {
		t.Fatalf("applied: %+v", first)
	}
	lots, _ := r.Lots("ws", "credit")
	if lots[0].Remainder != "30" || lots[1].Remainder != "20" || lots[2].Remainder != "10" {
		t.Fatalf("the earliest expiry is used first: %+v", lots)
	}
	held := reserved(t, r, apply("a2", "credit", "inv-2", "15", "15", false))
	instant := time.Date(2026, 11, 21, 0, 0, 0, 0, time.UTC)
	if !held.Expiry.Equal(start.Add(time.Hour)) {
		t.Fatalf("expiry: %v", held.Expiry)
	}
	clock.set(instant.Add(-30 * time.Minute))
	capped := reserved(t, r, apply("a3", "credit", "inv-3", "25", "25", false))
	if !capped.Expiry.Equal(instant) {
		t.Fatalf("a reservation outlives its lot: %v", capped.Expiry)
	}
	clock.set(instant)
	if _, err := r.Confirm("ws", "a3", "runner"); kind(err) != "rejected" {
		t.Fatalf("a confirm at the instant of expiry: %v", err)
	}
	if v, _ := r.Application("ws", "a3"); v.State != StateExpired {
		t.Fatalf("state: %+v", v)
	}
	st := identity(t, r, "ws", "credit")
	if st.Expired != "10" || st.Closing != "50" || st.Available != "50" {
		t.Fatalf("statement: %+v", st)
	}
	lots, _ = r.Lots("ws", "credit")
	if lots[2].Remainder != "0" || lots[2].Expired != "10" || lots[2].Available != "0" {
		t.Fatalf("an expired lot: %+v", lots[2])
	}
	if _, err := r.Reverse("ws", Reversal{ID: "back", Kind: ReverseApplication, Record: "a1", Amount: "40.00000001"}, "clerk-1"); kind(err) != "rejected" {
		t.Fatalf("a reversal beyond the applied amount: %v", err)
	}
	reversedBy(t, r, Reversal{ID: "back", Kind: ReverseApplication, Record: "a1", Amount: "40"})
	st = identity(t, r, "ws", "credit")
	if st.Expired != "50" || st.Closing != "50" || st.ApplicationReversals != "40" {
		t.Fatalf("an amount returned to an expired lot: %+v", st)
	}
	if _, err := r.Reverse("ws", Reversal{ID: "lot", Kind: ReverseIncrease, Record: "soon", Amount: "1"}, "clerk-1"); kind(err) != "rejected" {
		t.Fatalf("reversing an expired lot: %v", err)
	}
}

type ratSum struct{ value *big.Rat }

func (s *ratSum) add(t testing.TB, text string) *ratSum {
	if s.value == nil {
		s.value = new(big.Rat)
	}
	s.value.Add(s.value, rat(t, text))
	return s
}

func (s *ratSum) sub(t testing.TB, text string) *ratSum {
	if s.value == nil {
		s.value = new(big.Rat)
	}
	s.value.Sub(s.value, rat(t, text))
	return s
}

func (s *ratSum) text() string { return s.value.RatString() }

func TestOpeningAfterAnImportOfItsLedger(t *testing.T) {
	r, clock := openWith(t, testLimits())
	opened(t, r, cash("a", "cust-1", "USD"))
	opened(t, r, cash("b", "cust-2", "USD"))
	opened(t, r, cash("c", "cust-3", "USD"))
	early := receipt("i1", "a", "10")
	early.Import = Import{Ledger: "legacy", Transaction: "t1", Occurred: "2026-09-15"}
	funded(t, r, early)
	o := Opening{ID: "o", Account: "b", PriorLedger: "legacy", AsOf: "2026-09-30", Amount: "50", Evidence: evidence, MigrationKey: "m", IncludedFrom: "2026-01-01"}
	if _, err := r.RecordOpening("ws", o, "clerk"); kind(err) != "rejected" {
		t.Fatalf("an opening including an imported transaction: %v", err)
	}
	o.AsOf = "2026-09-14"
	if _, err := r.RecordOpening("ws", o, "clerk"); err != nil {
		t.Fatalf("an opening before the import: %v", err)
	}
	pending := Opening{ID: "o2", Account: "c", PriorLedger: "other", AsOf: "2026-09-30", Amount: "5", Evidence: evidence, MigrationKey: "m2", IncludedFrom: "2026-01-01"}
	if _, err := r.RecordOpening("ws", pending, "clerk"); err != nil {
		t.Fatal(err)
	}
	opened(t, r, cash("d", "cust-4", "USD"))
	funded(t, r, receipt("fund", "d", "10"))
	late := apply("p", "d", "inv", "3", "3", false)
	late.Import = Import{Ledger: "other", Transaction: "t9", Occurred: "2026-09-20"}
	if _, err := r.Reserve("ws", late, "runner"); kind(err) != "rejected" {
		t.Fatalf("an import under a pending opening of its ledger: %v", err)
	}
	r.WithdrawOpening("ws", "o2", "clerk")
	reserved(t, r, late)
	again := pending
	again.ID = "o3"
	if _, err := r.RecordOpening("ws", again, "clerk"); kind(err) != "rejected" {
		t.Fatalf("an opening including a reserved import: %v", err)
	}
	clock.advance(2 * time.Hour)
	if _, err := r.RecordOpening("ws", again, "clerk"); err != nil {
		t.Fatalf("an expired reservation still blocks: %v", err)
	}
	r.WithdrawOpening("ws", "o3", "clerk")
	late.ID = "p2"
	applied(t, r, late)
	fourth := pending
	fourth.ID = "o4"
	fourth.Account = "c"
	if _, err := r.RecordOpening("ws", fourth, "clerk"); kind(err) != "rejected" {
		t.Fatalf("an opening including a confirmed import: %v", err)
	}
	r.WithdrawIncrease("ws", "i1", "clerk")
}

func TestApprovalJudgesImportsAgain(t *testing.T) {
	r, _ := openWith(t, testLimits())
	opened(t, r, cash("a", "cust-1", "USD"))
	opened(t, r, cash("b", "cust-2", "USD"))
	o := Opening{ID: "o", Account: "b", PriorLedger: "legacy", AsOf: "2026-09-30", Amount: "50", Evidence: evidence, MigrationKey: "m", IncludedFrom: "2026-01-01"}
	r.RecordOpening("ws", o, "clerk")
	r.WithdrawOpening("ws", "o", "clerk")
	inc := receipt("i1", "a", "10")
	inc.Import = Import{Ledger: "legacy", Transaction: "t1", Occurred: "2026-09-15"}
	if _, err := r.RecordIncrease("ws", inc, "clerk"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ApproveOpening("ws", "o", "lead"); kind(err) != "rejected" {
		t.Fatalf("approving a withdrawn opening: %v", err)
	}
	if _, err := r.WithdrawIncrease("ws", "i1", "clerk"); err != nil {
		t.Fatal(err)
	}
	o.ID = "o2"
	if _, err := r.RecordOpening("ws", o, "clerk"); err != nil {
		t.Fatalf("a withdrawn import still blocks: %v", err)
	}
}

func TestOpeningOnTheDateOfAnImport(t *testing.T) {
	r, _ := openWith(t, testLimits())
	opened(t, r, cash("a", "cust-1", "USD"))
	opened(t, r, cash("b", "cust-2", "USD"))
	inc := receipt("i1", "a", "10")
	inc.Import = Import{Ledger: "inc-ledger", Transaction: "t1", Occurred: "2026-09-30"}
	funded(t, r, inc)
	app := apply("p", "a", "inv", "3", "3", false)
	app.Import = Import{Ledger: "app-ledger", Transaction: "t2", Occurred: "2026-09-30"}
	reserved(t, r, app)
	for _, ledger := range []string{"inc-ledger", "app-ledger"} {
		o := Opening{ID: "o-" + ledger, Account: "b", PriorLedger: ledger, AsOf: "2026-09-30", Amount: "1", Evidence: evidence, MigrationKey: "m-" + ledger, IncludedFrom: "2026-01-01"}
		if _, err := r.RecordOpening("ws", o, "clerk"); kind(err) != "rejected" {
			t.Errorf("%s on the import's date: %v", ledger, err)
		}
	}
}
