package balances

import (
	"strings"
	"testing"
)

func TestDigestsArePrinted(t *testing.T) {
	r, _ := openWith(t, testLimits())
	a := cash("cash", "cust-1", "USD")
	acct, _ := r.OpenAccount("ws", a, "clerk-1")
	if acct.Digest != printed(accountLabel, "ws", "cash", "cust-1", "entity-1", "USD", KindCash, AuthorityPlatform, "") {
		t.Errorf("account digest")
	}
	ext := Account{ID: "ext", Customer: "c", Entity: "e", Currency: "EUR", Kind: KindCredit, Authority: AuthorityExternal, Ledger: "erp"}
	extView, _ := r.OpenAccount("ws", ext, "clerk-1")
	if extView.Digest != printed(accountLabel, "ws", "ext", "c", "e", "EUR", KindCredit, AuthorityExternal, "erp") {
		t.Errorf("external account digest")
	}
	o := Opening{ID: "o", Account: "cash", PriorLedger: "legacy", AsOf: "2026-09-30", Amount: "12.50", Evidence: evidence, MigrationKey: "m", IncludedFrom: "2026-01-01", ExpiresOn: "2027-01-31"}
	ov, _ := r.RecordOpening("ws", o, "clerk-1")
	if ov.Digest != printed(openingLabel, "ws", "o", "cash", "legacy", "2026-09-30", "12.5", evidence, "m", "2026-01-01", "2027-01-31") || ov.Opening.Amount != "12.5" {
		t.Errorf("opening digest: %+v", ov)
	}
	r.ApproveOpening("ws", "o", "lead-1")
	inc := receipt("i", "cash", "3.10")
	inc.ExpiresOn = "2026-12-01"
	inc.Import = Import{Ledger: "legacy", Transaction: "t", Occurred: "2026-10-01"}
	iv, _ := r.RecordIncrease("ws", inc, "clerk-1")
	if iv.Digest != printed(increaseLabel, "ws", "i", "cash", IncreaseCashReceipt, "3.1", "ref-i", "2026-12-01", "legacy", "t", "2026-10-01") {
		t.Errorf("increase digest: %+v", iv)
	}
	app := apply("a", "cash", "inv", "7.00", "2.5", true)
	av := reserved(t, r, app)
	if av.Digest != printed(applicationLabel, "ws", "a", "cash", "inv", "1", "7", "2.5", "true", "", "", "") {
		t.Errorf("application digest")
	}
	app.ID, app.Partial = "b", false
	app.Import = Import{Ledger: "legacy", Transaction: "u", Occurred: "2026-10-02"}
	bv := reserved(t, r, app)
	if bv.Digest != printed(applicationLabel, "ws", "b", "cash", "inv", "1", "7", "2.5", "false", "legacy", "u", "2026-10-02") {
		t.Errorf("imported application digest")
	}
	r.Confirm("ws", "a", "runner")
	rv, _ := r.Reverse("ws", Reversal{ID: "v", Kind: ReverseApplication, Record: "a", Amount: "0.50"}, "clerk-1")
	if rv.Digest != printed(reversalLabel, "ws", "v", ReverseApplication, "a", "0.5") || rv.State != StatePending {
		t.Errorf("reversal digest: %+v", rv)
	}
}

func TestFractionsInViews(t *testing.T) {
	r, _ := openWith(t, testLimits())
	opened(t, r, cash("cash", "cust-1", "USD"))
	funded(t, r, receipt("r", "cash", "0.3"))
	cases := []struct{ requested, applied, closing string }{
		{"0.001", "0.001", "0.299"},
		{"0.00000008", "0.00000008", "0.29899992"},
		{"0.12", "0.12", "0.17899992"},
		{"1", "0.17899992", "0"},
	}
	for index, c := range cases {
		id := []string{"a1", "a2", "a3", "a4"}[index]
		v := applied(t, r, apply(id, "cash", "inv-"+id, "5", c.requested, true))
		st := identity(t, r, "ws", "cash")
		if v.Applied != c.applied || st.Closing != c.closing {
			t.Errorf("%s: %+v %+v", id, v, st)
		}
	}
	reversedBy(t, r, Reversal{ID: "v", Kind: ReverseApplication, Record: "a4", Amount: "0.07899992"})
	if v, _ := r.Application("ws", "a4"); v.Reversed != "0.07899992" {
		t.Errorf("reversed: %+v", v)
	}
	if st := identity(t, r, "ws", "cash"); st.Closing != "0.07899992" || st.ApplicationReversals != "0.07899992" {
		t.Errorf("statement: %+v", st)
	}
}

func TestTenantBoundary(t *testing.T) {
	r, _ := openWith(t, testLimits())
	opened(t, r, cash("cash", "cust-1", "USD"))
	funded(t, r, receipt("r", "cash", "10"))
	applied(t, r, apply("a", "cash", "inv", "5", "5", false))
	if _, err := r.Reserve("ws2", apply("b", "cash", "inv", "5", "5", false), "runner"); kind(err) != "not_found" {
		t.Errorf("reserve: %v", err)
	}
	if _, err := r.OpenAccount("ws2", cash("cash", "cust-1", "USD"), "clerk"); err != nil {
		t.Fatal(err)
	}
	if v := refused(t, r, apply("a", "cash", "inv", "5", "5", false)); v != "" {
		t.Errorf("a repeat in ws: %s", v)
	}
	d, err := r.Reserve("ws2", apply("a", "cash", "inv", "5", "5", false), "runner")
	if err != nil || d.Reason != ReasonNothingAvailable {
		t.Errorf("another workspace's balance: %+v %v", d, err)
	}
	for name, call := range map[string]func() error{
		"account":   func() error { _, err := r.Account("ws3", "cash"); return err },
		"increase":  func() error { _, err := r.Increase("ws2", "r"); return err },
		"approve":   func() error { _, err := r.ApproveIncrease("ws2", "r", "lead"); return err },
		"withdraw":  func() error { _, err := r.WithdrawIncrease("ws2", "r", "lead"); return err },
		"opening":   func() error { _, err := r.Opening("ws2", "o"); return err },
		"approveO":  func() error { _, err := r.ApproveOpening("ws2", "o", "lead"); return err },
		"withdrawO": func() error { _, err := r.WithdrawOpening("ws2", "o", "lead"); return err },
		"app":       func() error { _, err := r.Application("ws2", "a"); return err },
		"release":   func() error { _, err := r.Release("ws2", "a", "lead"); return err },
		"reverse": func() error {
			_, err := r.Reverse("ws2", Reversal{ID: "v", Kind: ReverseApplication, Record: "a", Amount: "1"}, "c")
			return err
		},
		"approveR":  func() error { _, err := r.ApproveReversal("ws2", "v", "lead"); return err },
		"reversal":  func() error { _, err := r.Reversal("ws2", "v"); return err },
		"lots":      func() error { _, err := r.Lots("ws3", "cash"); return err },
		"statement": func() error { _, err := r.Statement("ws3", "cash"); return err },
		"recordI":   func() error { _, err := r.RecordIncrease("ws3", receipt("r", "cash", "1"), "c"); return err },
		"recordO": func() error {
			_, err := r.RecordOpening("ws3", Opening{ID: "o", Account: "cash", PriorLedger: "l", AsOf: "2026-09-30", Amount: "1", Evidence: evidence, MigrationKey: "m", IncludedFrom: "2026-09-01"}, "c")
			return err
		},
	} {
		if kind(call()) != "not_found" {
			t.Errorf("%s crosses workspaces", name)
		}
	}
	if st := identity(t, r, "ws2", "cash"); st.Closing != "0" {
		t.Errorf("ws2: %+v", st)
	}
}

func TestFormsAtTheirBounds(t *testing.T) {
	r, _ := openWith(t, testLimits())
	long := strings.Repeat("Z", 128)
	if _, err := r.OpenAccount(strings.Repeat("w", 128), Account{ID: long, Customer: "AZaz09.-_:", Entity: long, Currency: "AZA", Kind: KindCash, Authority: AuthorityPlatform}, long); err != nil {
		t.Fatalf("bounds: %v", err)
	}
	for _, id := range []string{strings.Repeat("a", 129), "/a", "a@", "a[", "a`", "a{", ""} {
		if _, err := r.OpenAccount("ws", Account{ID: id, Customer: "c", Entity: "e", Currency: "USD", Kind: KindCash, Authority: AuthorityPlatform}, "clerk"); kind(err) != "invalid" {
			t.Errorf("id %q: %v", id, err)
		}
		if _, err := r.OpenAccount(id, cash("x", "c", "USD"), "clerk"); kind(err) != "invalid" {
			t.Errorf("workspace %q: %v", id, err)
		}
	}
	for _, code := range []string{"@AA", "A[A", "AA@", "[AA", "A@A", "AA[", "AAAA", "AA"} {
		if _, err := r.OpenAccount("ws", cash("x", "c", code), "clerk"); kind(err) != "invalid" {
			t.Errorf("currency %q: %v", code, err)
		}
	}
	opened(t, r, cash("cash", "cust-1", "ZAZ"))
	for _, date := range []string{"1999-12-31", "2100-01-01", "2026-1-01", "2026-02-29", "2024-02-30", "20260101", "2026/01/01"} {
		o := Opening{ID: "o", Account: "cash", PriorLedger: "l", AsOf: date, Amount: "1", Evidence: evidence, MigrationKey: "m", IncludedFrom: "2000-01-01"}
		if _, err := r.RecordOpening("ws", o, "clerk"); kind(err) != "invalid" {
			t.Errorf("date %q: %v", date, err)
		}
	}
	for _, digest := range []string{strings.Repeat("g", 64), strings.Repeat("a", 65), strings.Repeat("0", 63) + "/", strings.Repeat("0", 63) + ":"} {
		o := Opening{ID: "o", Account: "cash", PriorLedger: "l", AsOf: "2026-09-30", Amount: "1", Evidence: digest, MigrationKey: "m", IncludedFrom: "2026-09-30"}
		if _, err := r.RecordOpening("ws", o, "clerk"); kind(err) != "invalid" {
			t.Errorf("digest %q: %v", digest, err)
		}
	}
	leap := Opening{ID: "o", Account: "cash", PriorLedger: "l", AsOf: "2024-02-29", Amount: "1", Evidence: strings.Repeat("9", 64), MigrationKey: "m", IncludedFrom: "2000-01-01", ExpiresOn: "2099-12-31"}
	if _, err := r.RecordOpening("ws", leap, "clerk"); err != nil {
		t.Fatalf("a leap day: %v", err)
	}
	if v, err := r.Account("ws", "cash"); err != nil || v.Author != "clerk-1" || v.Account.Currency != "ZAZ" {
		t.Errorf("read: %+v %v", v, err)
	}
	if _, err := r.Account("ws", "none"); kind(err) != "not_found" {
		t.Errorf("unknown: %v", err)
	}
	for name, call := range map[string]func() error{
		"account":   func() error { _, err := r.Account("ws", ""); return err },
		"increase":  func() error { _, err := r.Increase("", "r"); return err },
		"approve":   func() error { _, err := r.ApproveIncrease("ws", "r", ""); return err },
		"withdraw":  func() error { _, err := r.WithdrawIncrease("ws", "", "c"); return err },
		"opening":   func() error { _, err := r.Opening("ws", ""); return err },
		"approveO":  func() error { _, err := r.ApproveOpening("ws", "o", ""); return err },
		"withdrawO": func() error { _, err := r.WithdrawOpening("ws", "o", ""); return err },
		"app":       func() error { _, err := r.Application("ws", ""); return err },
		"confirm":   func() error { _, err := r.Confirm("ws", "a", ""); return err },
		"release":   func() error { _, err := r.Release("ws", "a", ""); return err },
		"reverse": func() error {
			_, err := r.Reverse("ws", Reversal{ID: "v", Kind: ReverseApplication, Record: "a", Amount: "0"}, "c")
			return err
		},
		"approveR":  func() error { _, err := r.ApproveReversal("ws", "v", ""); return err },
		"reversal":  func() error { _, err := r.Reversal("ws", ""); return err },
		"lots":      func() error { _, err := r.Lots("ws", ""); return err },
		"statement": func() error { _, err := r.Statement("", "cash"); return err },
		"recordI":   func() error { _, err := r.RecordIncrease("ws", receipt("r", "cash", "1"), ""); return err },
		"open":      func() error { _, err := r.OpenAccount("ws", cash("y", "c", "USD"), ""); return err },
	} {
		if kind(call()) != "invalid" {
			t.Errorf("%s accepts a malformed request", name)
		}
	}
}
