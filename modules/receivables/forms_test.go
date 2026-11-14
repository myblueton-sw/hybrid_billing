package receivables

import (
	"strings"
	"testing"
	"time"
)

func TestValidators(t *testing.T) {
	for _, id := range []string{"Z", "z", "9", "A", "a", "0", ".", "_", ":", "-", strings.Repeat("a", 128)} {
		if !validID(id) {
			t.Errorf("identifier %q", id)
		}
	}
	for _, id := range []string{"", "/a", "a/", "[", "{", "@", "`", " a", strings.Repeat("a", 129)} {
		if validID(id) {
			t.Errorf("identifier %q", id)
		}
	}
	for _, code := range []string{"AAA", "ZZZ"} {
		if !validCurrency(code) {
			t.Errorf("currency %q", code)
		}
	}
	for _, code := range []string{"@AA", "A@A", "AA@", "[AA", "A[A", "AA[", "AAAA"} {
		if validCurrency(code) {
			t.Errorf("currency %q", code)
		}
	}
	for _, date := range []string{"2000-01-01", "2099-12-31", "2024-02-29", "2026-10-10"} {
		if !validDate(date) {
			t.Errorf("date %q", date)
		}
	}
	for _, date := range []string{"1999-12-31", "2100-01-01", "2026-02-29", "2026-13-01", "2026-1-01", "20261001"} {
		if validDate(date) {
			t.Errorf("date %q", date)
		}
	}
	for _, text := range []string{strings.Repeat("0", 64), strings.Repeat("f", 64), strings.Repeat("9", 64), strings.Repeat("a", 64)} {
		if !validDigest(text) {
			t.Errorf("digest %q", text)
		}
	}
	for _, text := range []string{"g" + strings.Repeat("0", 63), "/" + strings.Repeat("0", 63), "`" + strings.Repeat("0", 63), strings.Repeat("0", 63) + ":", strings.Repeat("A", 64), strings.Repeat("0", 63)} {
		if validDigest(text) {
			t.Errorf("digest %q", text)
		}
	}
	if !zoneForm(strings.Repeat("a", 64)) || zoneForm(strings.Repeat("a", 65)) || zoneForm("") || zoneForm("Local") {
		t.Error("zone names")
	}
}

func TestEveryArgumentIsChecked(t *testing.T) {
	r, clock := openWith(t, testLimits())
	clock.set(time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC))
	setup(t, r, "k", "UTC", 30)
	must(r.Issue("ws", invoice("d", "k", "10", start), "c"))
	must(r.Allocate("ws", receipt("a", "d", "1", "2026-10-02"), "s"))
	must(r.Extend("ws", Extension{ID: "x", Document: "d", Due: "2026-12-01", Reason: "billing_error"}, "c"))
	must(r.Hold("ws", Hold{ID: "h", Document: "d", Kind: "dispute", Reference: "c"}, "c"))
	must(r.Reverse("ws", Reversal{ID: "v", Allocation: "a", Effective: "2026-10-02", Source: SourceAuthority, Ledger: "erp", Revision: "r"}, "s"))
	bad := "a b"
	calls := map[string][]func() error{
		"recordTerm": {
			func() error { return second(r.RecordTerm(bad, Term{ID: "t", Revision: 1, Mode: ModeCalendar}, "c")) },
			func() error { return second(r.RecordTerm("ws", Term{ID: "t", Revision: 1, Mode: ModeCalendar}, bad)) },
			func() error {
				return second(r.RecordTerm("ws", Term{ID: "t", Revision: 1000001, Mode: ModeCalendar}, "c"))
			},
		},
		"approveTerm": {
			func() error { return second(r.ApproveTerm(bad, "net30", 1, "x")) },
			func() error { return second(r.ApproveTerm("ws", bad, 1, "x")) },
			func() error { return second(r.ApproveTerm("ws", "net30", 0, "x")) },
			func() error { return second(r.ApproveTerm("ws", "net30", 1, bad)) },
		},
		"term": {
			func() error { return second(r.Term(bad, "net30", 1)) },
			func() error { return second(r.Term("ws", bad, 1)) },
		},
		"bind": {
			func() error {
				return second(r.Bind(bad, Binding{ID: "b", Contract: "k", Term: "net30", Revision: 1, Zone: "UTC", From: "2030-01-01", Reason: "initial"}, "c"))
			},
			func() error {
				return second(r.Bind("ws", Binding{ID: "b", Contract: "k", Term: "net30", Revision: 1, Zone: "UTC", From: "2030-01-01", Reason: "initial"}, bad))
			},
			func() error {
				return second(r.Bind("ws", Binding{ID: "b", Contract: bad, Term: "net30", Revision: 1, Zone: "UTC", From: "2030-01-01", Reason: "initial"}, "c"))
			},
			func() error {
				return second(r.Bind("ws", Binding{ID: "b", Contract: "k", Term: bad, Revision: 1, Zone: "UTC", From: "2030-01-01", Reason: "initial"}, "c"))
			},
			func() error {
				return second(r.Bind("ws", Binding{ID: bad, Contract: "k", Term: "net30", Revision: 1, Zone: "UTC", From: "2030-01-01", Reason: "initial"}, "c"))
			},
			func() error {
				return second(r.Bind("ws", Binding{ID: "b", Contract: "k", Term: "net30", Revision: 1, Zone: "UTC", From: "2030-01-01", To: "2030-13-01", Reason: "initial"}, "c"))
			},
		},
		"approveBinding":  {func() error { return second(r.ApproveBinding(bad, "k-b", "x")) }, func() error { return second(r.ApproveBinding("ws", "k-b", bad)) }},
		"withdrawBinding": {func() error { return second(r.WithdrawBinding(bad, "k-b", "x")) }, func() error { return second(r.WithdrawBinding("ws", bad, "x")) }},
		"end": {
			func() error { return second(r.EndBinding(bad, "k-b", "2027-01-01", "x")) },
			func() error { return second(r.EndBinding("ws", bad, "2027-01-01", "x")) },
			func() error { return second(r.EndBinding("ws", "k-b", "2027-01-01", bad)) },
		},
		"approveEnd": {func() error { return second(r.ApproveEnd(bad, "k-b", "x")) }, func() error { return second(r.ApproveEnd("ws", "k-b", bad)) }},
		"bindingOf":  {func() error { return second(r.BindingOf("ws", bad)) }},
		"estimate": {
			func() error { return second(r.Estimate(bad, "k", "2026-10-01")) },
			func() error { return second(r.Estimate("ws", bad, "2026-10-01")) },
		},
		"issue":    {func() error { return second(r.Issue(bad, invoice("e", "k", "1", start), "c")) }},
		"document": {func() error { return second(r.Document(bad, "d")) }},
		"extend": {
			func() error {
				return second(r.Extend(bad, Extension{ID: "y", Document: "d", Due: "2026-12-02", Reason: "billing_error"}, "c"))
			},
			func() error {
				return second(r.Extend("ws", Extension{ID: "y", Document: "d", Due: "2026-12-02", Reason: "billing_error"}, bad))
			},
			func() error {
				return second(r.Extend("ws", Extension{ID: bad, Document: "d", Due: "2026-12-02", Reason: "billing_error"}, "c"))
			},
			func() error {
				return second(r.Extend("ws", Extension{ID: "y", Document: bad, Due: "2026-12-02", Reason: "billing_error"}, "c"))
			},
			func() error {
				return second(r.Extend("ws", Extension{ID: "y", Document: "d", Due: "2026-12-32", Reason: "billing_error"}, "c"))
			},
		},
		"approveExtension":  {func() error { return second(r.ApproveExtension(bad, "x", "l")) }, func() error { return second(r.ApproveExtension("ws", "x", bad)) }},
		"withdrawExtension": {func() error { return second(r.WithdrawExtension(bad, "x", "c")) }, func() error { return second(r.WithdrawExtension("ws", bad, "c")) }},
		"hold": {
			func() error {
				return second(r.Hold(bad, Hold{ID: "g", Document: "d", Kind: "dispute", Reference: "c"}, "c"))
			},
			func() error {
				return second(r.Hold("ws", Hold{ID: "g", Document: "d", Kind: "dispute", Reference: "c"}, bad))
			},
			func() error {
				return second(r.Hold("ws", Hold{ID: bad, Document: "d", Kind: "dispute", Reference: "c"}, "c"))
			},
			func() error {
				return second(r.Hold("ws", Hold{ID: "g", Document: bad, Kind: "dispute", Reference: "c"}, "c"))
			},
			func() error {
				return second(r.Hold("ws", Hold{ID: "g", Document: "d", Kind: "dispute", Reference: bad}, "c"))
			},
		},
		"release": {func() error { return second(r.ReleaseHold(bad, "h", "c")) }, func() error { return second(r.ReleaseHold("ws", "h", bad)) }},
		"allocate": {
			func() error { return second(r.Allocate(bad, receipt("b", "d", "1", "2026-10-02"), "s")) },
			func() error { return second(r.Allocate("ws", receipt(bad, "d", "1", "2026-10-02"), "s")) },
			func() error { return second(r.Allocate("ws", receipt("b", bad, "1", "2026-10-02"), "s")) },
			func() error {
				return second(r.Allocate("ws", receipt("b", "d", "1."+strings.Repeat("0", 63), "2026-10-02"), "s"))
			},
		},
		"approveAllocation":  {func() error { return second(r.ApproveAllocation(bad, "a", "l")) }, func() error { return second(r.ApproveAllocation("ws", "a", bad)) }},
		"withdrawAllocation": {func() error { return second(r.WithdrawAllocation("ws", bad, "c")) }, func() error { return second(r.WithdrawAllocation("ws", "a", bad)) }},
		"reverse": {
			func() error {
				return second(r.Reverse(bad, Reversal{ID: "w", Allocation: "a", Effective: "2026-10-02", Source: SourceAuthority, Ledger: "erp", Revision: "r"}, "s"))
			},
			func() error {
				return second(r.Reverse("ws", Reversal{ID: "w", Allocation: "a", Effective: "2026-10-02", Source: SourceAuthority, Ledger: "erp", Revision: "r"}, bad))
			},
			func() error {
				return second(r.Reverse("ws", Reversal{ID: bad, Allocation: "a", Effective: "2026-10-02", Source: SourceAuthority, Ledger: "erp", Revision: "r"}, "s"))
			},
			func() error {
				return second(r.Reverse("ws", Reversal{ID: "w", Allocation: bad, Effective: "2026-10-02", Source: SourceAuthority, Ledger: "erp", Revision: "r"}, "s"))
			},
			func() error {
				return second(r.Reverse("ws", Reversal{ID: "w", Allocation: "a", Effective: "2026-1-02", Source: SourceAuthority, Ledger: "erp", Revision: "r"}, "s"))
			},
		},
		"approveReversal":  {func() error { return second(r.ApproveReversal(bad, "v", "l")) }, func() error { return second(r.ApproveReversal("ws", "v", bad)) }},
		"withdrawReversal": {func() error { return second(r.WithdrawReversal(bad, "v", "c")) }, func() error { return second(r.WithdrawReversal("ws", bad, "c")) }},
		"confirm": {
			func() error { return second(r.Confirm(bad, "en", "USD", start, "s")) },
			func() error { return second(r.Confirm("ws", bad, "USD", start, "s")) },
			func() error { return second(r.Confirm("ws", "en", "USD", start, bad)) },
		},
		"status": {func() error { return second(r.Status(bad, "d")) }},
		"aging": {
			func() error { return second(r.Aging("ws", Filter{Customer: bad})) },
			func() error { return second(r.Aging("ws", Filter{Entity: bad})) },
			func() error { return second(r.Aging("ws", Filter{Contract: bad})) },
		},
	}
	for name, list := range calls {
		for index, call := range list {
			if got := kind(call()); got != "invalid" {
				t.Errorf("%s #%d: %s", name, index, got)
			}
		}
	}
	if v := must(r.Term("ws", "net30", 1)); v.State != StateApproved {
		t.Error("a malformed call changed a term")
	}
	edge := must(r.RecordTerm("ws", Term{ID: "t", Revision: 1000000, NetDays: 0, Mode: ModeCalendar}, "c"))
	if edge.Term.Revision != 1000000 {
		t.Error("the largest revision")
	}
}

func TestBindingEdges(t *testing.T) {
	permissive := func(string) (*time.Location, error) { return time.UTC, nil }
	roomy := testLimits()
	roomy.Bindings = 8
	r, err := NewRegistry(testProfile(t), roomy, func() time.Time { return start }, permissive)
	if err != nil {
		t.Fatal(err)
	}
	must(r.RecordTerm("ws", Term{ID: "n", Revision: 1, Mode: ModeCalendar}, "c"))
	must(r.ApproveTerm("ws", "n", 1, "l"))
	long := strings.Repeat("z", 64)
	expect(t, "ok", second(r.Bind("ws", Binding{ID: "z", Contract: "kz", Term: "n", Revision: 1, Zone: long, From: "2026-01-01", Reason: "initial"}, "c")), "a zone name of 64 bytes")
	expect(t, "invalid", second(r.Bind("ws", Binding{ID: "z2", Contract: "kz", Term: "n", Revision: 1, Zone: long + "z", From: "2026-01-01", Reason: "initial"}, "c")), "a zone name of 65 bytes")
	must(r.Bind("ws", Binding{ID: "b1", Contract: "k", Term: "n", Revision: 1, Zone: "Zone-A", From: "2026-03-01", To: "2026-03-31", Reason: "initial"}, "c"))
	must(r.WithdrawBinding("ws", "b1", "c"))
	expect(t, "ok", second(r.Bind("ws", Binding{ID: "b2", Contract: "k", Term: "n", Revision: 1, Zone: "Zone-B", From: "2026-04-01", To: "2026-04-30", Reason: "initial"}, "c")), "a withdrawn binding does not fix the zone")
	must(r.ApproveBinding("ws", "b2", "l"))
	must(r.Bind("ws", Binding{ID: "b3", Contract: "k", Term: "n", Revision: 1, Zone: "Zone-B", From: "2026-03-01", To: "2026-04-01", Reason: "correction"}, "c"))
	expect(t, "rejected", second(r.ApproveBinding("ws", "b3", "l")), "a binding ending on the first day of another")
	must(r.Bind("ws", Binding{ID: "b4", Contract: "k", Term: "n", Revision: 1, Zone: "Zone-B", From: "2026-04-30", To: "2026-05-31", Reason: "correction"}, "c"))
	expect(t, "rejected", second(r.ApproveBinding("ws", "b4", "l")), "a binding starting on the last day of another")
	must(r.Bind("ws", Binding{ID: "b5", Contract: "k", Term: "n", Revision: 1, Zone: "Zone-B", From: "2026-03-01", To: "2026-03-31", Reason: "correction"}, "c"))
	must(r.WithdrawBinding("ws", "b3", "c"))
	expect(t, "ok", second(r.ApproveBinding("ws", "b5", "l")), "an adjacent binding")
	expect(t, "conflict", second(r.Bind("ws", Binding{ID: "b5", Contract: "k", Term: "n", Revision: 1, Zone: "Zone-B", From: "2026-03-02", To: "2026-03-31", Reason: "correction"}, "c")), "a repeat with other content")
	expect(t, "conflict", second(r.Bind("ws", Binding{ID: "b5", Contract: "k", Term: "n", Revision: 1, Zone: "Zone-B", From: "2026-03-01", To: "2026-03-31", Reason: "correction"}, "x")), "a repeat by another actor")
	expect(t, "rejected", second(r.EndBinding("ws", "b5", "2026-02-28", "c")), "an end before the start")
	expect(t, "ok", second(r.EndBinding("ws", "b5", "2026-03-01", "c")), "an end on the start")
	expect(t, "rejected", second(r.EndBinding("ws", "b2", "2026-04-30", "c")), "an end that does not shorten")
	expect(t, "rejected", second(r.EndBinding("ws", "b4", "2026-05-01", "c")), "an end of a pending binding")
	expect(t, "rejected", second(r.ApproveEnd("ws", "b2", "l")), "approve an end that was never asked")
}

func TestEndRoom(t *testing.T) {
	l := testLimits()
	l.Bindings = 2
	r, clock := openWith(t, l)
	clock.set(time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC))
	setup(t, r, "k", "UTC", 30)
	must(r.EndBinding("ws", "k-b", "2026-12-31", "c"))
	must(r.ApproveEnd("ws", "k-b", "l"))
	must(r.EndBinding("ws", "k-b", "2026-12-30", "c"))
	must(r.ApproveEnd("ws", "k-b", "l"))
	expect(t, "limit", second(r.EndBinding("ws", "k-b", "2026-12-29", "c")), "end room")
}
