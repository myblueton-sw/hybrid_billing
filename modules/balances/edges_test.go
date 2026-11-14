package balances

import (
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func TestEdgesOfForms(t *testing.T) {
	r, _ := openWith(t, testLimits())
	opened(t, r, cash("cash", "cust-1", "USD"))
	wide, err := decimal.NewProfile(decimal.ProfileSpec{ID: "wide", Revision: "1", InputIntegerDigits: 52, InputScale: 12, ResultIntegerDigits: 60, ResultScale: 12, IntermediateDigits: 200})
	if err != nil {
		t.Fatal(err)
	}
	big, _ := NewRegistry(wide, testLimits(), time.Now)
	big.OpenAccount("ws", cash("cash", "cust-1", "USD"), "c")
	long := strings.Repeat("1", 51) + "." + strings.Repeat("1", 12)
	if _, err := big.RecordIncrease("ws", Increase{ID: "r", Account: "cash", Kind: IncreaseCashReceipt, Amount: long, Reference: "x"}, "c"); err != nil || len(long) != 64 {
		t.Errorf("a number of 64 bytes: %v", err)
	}
	if _, err := big.RecordIncrease("ws", Increase{ID: "r2", Account: "cash", Kind: IncreaseCashReceipt, Amount: "1" + long, Reference: "y"}, "c"); kind(err) != "invalid" {
		t.Errorf("a number of 65 bytes: %v", err)
	}
	funded(t, r, receipt("r", "cash", "5"))
	if v := reserved(t, r, Application{ID: "one", Account: "cash", Target: Target{Document: "d", Line: "1", Amount: "1"}, Requested: "1", Lifetime: 1}); !v.Expiry.Equal(start.Add(time.Second)) {
		t.Errorf("a lifetime of one second: %+v", v)
	}
	if _, err := r.RecordOpening("ws", Opening{ID: "o", Account: "cash", PriorLedger: "l", AsOf: "2026-09-30", Amount: "1", Evidence: "g" + evidence[1:], MigrationKey: "m", IncludedFrom: "2026-09-30"}, "c"); kind(err) != "invalid" {
		t.Errorf("a digest with a bad first character: %v", err)
	}
	if _, err := NewRegistry(testProfile(t), Limits{Accounts: 1, Openings: 1, Increases: 1, Applications: 1, Targets: 1, Reversals: 1, Workspace: 1}, time.Now); err != nil {
		t.Errorf("the bottom limits: %v", err)
	}
	good := Opening{ID: "o", Account: "cash", PriorLedger: "l", AsOf: "2026-09-30", Amount: "1", Evidence: evidence, MigrationKey: "m", IncludedFrom: "2026-09-30"}
	inc := Increase{ID: "i", Account: "cash", Kind: IncreaseCashReceipt, Amount: "1", Reference: "z"}
	app := apply("a", "cash", "d", "1", "1", false)
	for name, call := range map[string]func() error{
		"opening id":       func() error { o := good; o.ID = "a b"; _, err := r.RecordOpening("ws", o, "c"); return err },
		"opening account":  func() error { o := good; o.Account = "a b"; _, err := r.RecordOpening("ws", o, "c"); return err },
		"opening ledger":   func() error { o := good; o.PriorLedger = ""; _, err := r.RecordOpening("ws", o, "c"); return err },
		"opening author":   func() error { _, err := r.RecordOpening("ws", good, "a b"); return err },
		"increase id":      func() error { i := inc; i.ID = "a b"; _, err := r.RecordIncrease("ws", i, "c"); return err },
		"increase account": func() error { i := inc; i.Account = "a b"; _, err := r.RecordIncrease("ws", i, "c"); return err },
		"application id":   func() error { a := app; a.ID = "a b"; _, err := r.Reserve("ws", a, "c"); return err },
		"application acct": func() error { a := app; a.Account = "a b"; _, err := r.Reserve("ws", a, "c"); return err },
		"application doc":  func() error { a := app; a.Target.Document = "a b"; _, err := r.Reserve("ws", a, "c"); return err },
		"reversal author": func() error {
			_, err := r.Reverse("ws", Reversal{ID: "v", Kind: ReverseIncrease, Record: "r", Amount: "1"}, "a b")
			return err
		},
		"approve opening":  func() error { _, err := r.ApproveOpening("ws", "a b", "lead"); return err },
		"withdraw opening": func() error { _, err := r.WithdrawOpening("ws", "a b", "lead"); return err },
		"approve increase": func() error { _, err := r.ApproveIncrease("ws", "a b", "lead"); return err },
		"confirm":          func() error { _, err := r.Confirm("ws", "a b", "lead"); return err },
		"release":          func() error { _, err := r.Release("ws", "a b", "lead"); return err },
		"approve reversal": func() error { _, err := r.ApproveReversal("ws", "a b", "lead"); return err },
	} {
		if kind(call()) != "invalid" {
			t.Errorf("%s accepts a malformed field", name)
		}
	}
}

func TestEdgesOfDates(t *testing.T) {
	r, clock := openWith(t, testLimits())
	opened(t, r, cash("cash", "cust-1", "USD"))
	opened(t, r, credit("credit", "cust-1", "USD"))
	o := Opening{ID: "o", Account: "cash", PriorLedger: "own", AsOf: "2026-09-30", Amount: "1", Evidence: evidence, MigrationKey: "m", IncludedFrom: "2026-09-01"}
	r.RecordOpening("ws", o, "clerk")
	r.ApproveOpening("ws", "o", "lead")
	own := receipt("own", "cash", "1")
	own.Import = Import{Ledger: "different", Transaction: "t1", Occurred: "2026-09-30"}
	if _, err := r.RecordIncrease("ws", own, "clerk"); kind(err) != "rejected" {
		t.Errorf("the account's own opening on its as-of date: %v", err)
	}
	other := Increase{ID: "x", Account: "credit", Kind: IncreaseCreditDocument, Amount: "1", Reference: "cd", Import: Import{Ledger: "own", Transaction: "t2", Occurred: "2026-09-30"}}
	if _, err := r.RecordIncrease("ws", other, "clerk"); kind(err) != "rejected" {
		t.Errorf("another opening of the ledger on its as-of date: %v", err)
	}
	now := receipt("today", "cash", "1")
	now.Import = Import{Ledger: "different", Transaction: "t3", Occurred: "2026-11-14"}
	if _, err := r.RecordIncrease("ws", now, "clerk"); err != nil {
		t.Errorf("an import of today: %v", err)
	}
	last := promotion("last", "credit", "5", "2026-11-15")
	r.RecordIncrease("ws", last, "clerk")
	clock.set(time.Date(2026, 11, 15, 23, 0, 0, 0, time.UTC))
	if _, err := r.ApproveIncrease("ws", "last", "lead"); err != nil {
		t.Errorf("an approval on the expiry date: %v", err)
	}
	fresh, freshClock := openWith(t, testLimits())
	opened(t, fresh, cash("cash", "cust-1", "USD"))
	expiring := o
	expiring.ExpiresOn = "2026-11-15"
	fresh.RecordOpening("ws", expiring, "clerk")
	freshClock.set(time.Date(2026, 11, 15, 23, 0, 0, 0, time.UTC))
	if _, err := fresh.ApproveOpening("ws", "o", "lead"); err != nil {
		t.Errorf("an opening approved on its expiry date: %v", err)
	}
}

func TestLotsAndReturns(t *testing.T) {
	r, clock := openWith(t, testLimits())
	opened(t, r, credit("credit", "cust-1", "USD"))
	funded(t, r, promotion("soon", "credit", "10", "2026-11-20"))
	funded(t, r, Increase{ID: "doc", Account: "credit", Kind: IncreaseCreditDocument, Amount: "50", Reference: "cd"})
	lots, _ := r.Lots("ws", "credit")
	if lots[0].Number != 1 || lots[1].Number != 2 || lots[0].Record != "soon" || lots[1].Source != ReverseIncrease {
		t.Fatalf("numbers: %+v", lots)
	}
	applied(t, r, apply("first", "credit", "d1", "10", "10", false))
	clock.set(time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC))
	later := apply("later", "credit", "d2", "5", "5", false)
	later.Lifetime = 86400
	v := reserved(t, r, later)
	if !v.Expiry.Equal(time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("a used-up lot capped the reservation: %v", v.Expiry)
	}
	r.Release("ws", "later", "runner")
	applied(t, r, apply("split", "credit", "d3", "30", "30", false))
	reversedBy(t, r, Reversal{ID: "v1", Kind: ReverseApplication, Record: "split", Amount: "3"})
	reversedBy(t, r, Reversal{ID: "v2", Kind: ReverseApplication, Record: "split", Amount: "27"})
	lots, _ = r.Lots("ws", "credit")
	if lots[1].Remainder != "50" || lots[1].Available != "50" {
		t.Fatalf("returns: %+v", lots)
	}
	held := reserved(t, r, apply("held", "credit", "d4", "15", "15", false))
	lots, _ = r.Lots("ws", "credit")
	if lots[1].Held != "15" || lots[1].Available != "35" || held.Applied != "15" {
		t.Fatalf("held: %+v", lots)
	}
	fresh, _ := openWith(t, testLimits())
	opened(t, fresh, cash("cash", "cust-1", "USD"))
	fresh.RecordOpening("ws", Opening{ID: "o", Account: "cash", PriorLedger: "l", AsOf: "2026-09-30", Amount: "20", Evidence: evidence, MigrationKey: "m", IncludedFrom: "2026-09-01"}, "clerk")
	if _, err := fresh.Reverse("ws", Reversal{ID: "p", Kind: ReverseOpening, Record: "o", Amount: "1"}, "clerk"); kind(err) != "rejected" {
		t.Fatalf("reversing a pending opening: %v", err)
	}
	fresh.ApproveOpening("ws", "o", "lead")
	funded(t, fresh, receipt("r", "cash", "5"))
	applied(t, fresh, apply("two", "cash", "d", "22", "22", false))
	reversedBy(t, fresh, Reversal{ID: "b1", Kind: ReverseApplication, Record: "two", Amount: "1"})
	reversedBy(t, fresh, Reversal{ID: "b2", Kind: ReverseApplication, Record: "two", Amount: "2"})
	lots, _ = fresh.Lots("ws", "cash")
	if lots[0].Remainder != "1" || lots[1].Remainder != "5" || lots[0].Number != 1 || lots[1].Number != 2 {
		t.Fatalf("a return beyond a hold: %+v", lots)
	}
	reversedBy(t, fresh, Reversal{ID: "ov", Kind: ReverseOpening, Record: "o", Amount: "0.5"})
	st := identity(t, fresh, "ws", "cash")
	if st.LotReversals != "0.5" || st.Closing != "5.5" || st.Opening != "20" {
		t.Fatalf("an opening reversal: %+v", st)
	}
	limits := testLimits()
	limits.Reversals, limits.Workspace = 1, 10
	tight, _ := openWith(t, limits)
	opened(t, tight, cash("cash", "cust-1", "USD"))
	funded(t, tight, receipt("r", "cash", "5"))
	if _, err := tight.Reverse("ws", Reversal{ID: "q1", Kind: ReverseIncrease, Record: "r", Amount: "1"}, "clerk"); err != nil {
		t.Fatal(err)
	}
	if _, err := tight.Reverse("ws", Reversal{ID: "q2", Kind: ReverseIncrease, Record: "r", Amount: "1"}, "clerk"); kind(err) != "limit" {
		t.Fatalf("reversals per record: %v", err)
	}
}
