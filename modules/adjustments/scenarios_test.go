package adjustments

import (
	"reflect"
	"testing"
)

func TestTargetTotalBecomesDifferenceLine(t *testing.T) {
	r, _ := setup(t)
	v := opened(t, r, sample("d1", false))
	if v.Source != "100" || v.Total != "100" || v.Adjustment != "0" || v.Gross != "0" || v.Revision != 1 || v.State != StateOpen || v.Subject != "" {
		t.Fatalf("opened %+v", v)
	}
	before := append([]SourceLine(nil), v.Draft.Lines...)
	v = must(r.Add("ws", "d1", 1, item("e1", EntryTargetTotal, "", "120", ""), human("clerk")))
	if !reflect.DeepEqual(v.Draft.Lines, before) {
		t.Fatalf("source lines changed: %+v", v.Draft.Lines)
	}
	want := Difference{Entry: "e1", Kind: EntryTargetTotal, Current: "100", Target: "120", Amount: "20"}
	if len(v.Differences) != 1 || v.Differences[0] != want {
		t.Fatalf("differences %+v", v.Differences)
	}
	if v.Lines[0].Adjusted != "25" || v.Lines[1].Adjusted != "75" || v.Total != "120" || v.Adjustment != "20" || v.Gross != "20" || v.Revision != 2 {
		t.Fatalf("view %+v", v)
	}
	if v.Entries[0].Author != human("clerk") || !v.Entries[0].Active || !v.Entries[0].Written.Equal(start) {
		t.Fatalf("entry %+v", v.Entries[0])
	}
}

func TestKindsAndScopes(t *testing.T) {
	r, _ := setup(t)
	v := opened(t, r, sample("d1", false))
	v = must(r.Add("ws", "d1", v.Revision, item("q", EntryQuantity, "l1", "12", ""), human("clerk")))
	v = must(r.Add("ws", "d1", v.Revision, item("f", EntryFixed, "l2", "", "-5"), human("clerk")))
	v = must(r.Add("ws", "d1", v.Revision, item("t", EntryTargetTotal, "l1", "40", ""), human("clerk")))
	v = must(r.Add("ws", "d1", v.Revision, item("w", EntryFixed, "", "", "2.5"), human("clerk")))
	want := []Difference{
		{Entry: "q", Kind: EntryQuantity, Line: "l1", Charge: "ch-1", Current: "10", Target: "12", Quantity: "2", Amount: "5"},
		{Entry: "f", Kind: EntryFixed, Line: "l2", Charge: "ch-2", Current: "75", Target: "70", Amount: "-5"},
		{Entry: "t", Kind: EntryTargetTotal, Line: "l1", Charge: "ch-1", Current: "30", Target: "40", Amount: "10"},
		{Entry: "w", Kind: EntryFixed, Current: "110", Target: "112.5", Amount: "2.5"},
	}
	if !reflect.DeepEqual(v.Differences, want) {
		t.Fatalf("differences %+v", v.Differences)
	}
	if v.Lines[0].Adjusted != "40" || v.Lines[1].Adjusted != "70" || v.Total != "112.5" || v.Adjustment != "12.5" || v.Gross != "22.5" {
		t.Fatalf("view %+v", v)
	}
	v = must(r.Add("ws", "d1", v.Revision, item("q2", EntryQuantity, "l1", "11", ""), human("clerk")))
	last := v.Differences[len(v.Differences)-1]
	if last.Current != "12" || last.Quantity != "-1" || last.Amount != "-2.5" || v.Lines[0].Adjusted != "37.5" {
		t.Fatalf("second quantity %+v %+v", last, v.Lines[0])
	}
}

func TestQuantityRoundingTelescopes(t *testing.T) {
	r, _ := setup(t)
	d := sample("d1", false)
	d.Lines = []SourceLine{{ID: "l1", Charge: "ch-1", Quantity: "1", Price: "0.333", Amount: "0.33"}}
	one := must(r.Add("ws", "d1", opened(t, r, d).Revision, item("e1", EntryQuantity, "l1", "2", ""), human("clerk")))
	d.ID = "d2"
	two := opened(t, r, d)
	two = must(r.Add("ws", "d2", two.Revision, item("e1", EntryQuantity, "l1", "1.5", ""), human("clerk")))
	two = must(r.Add("ws", "d2", two.Revision, item("e2", EntryQuantity, "l1", "2", ""), human("clerk")))
	if one.Total != "0.67" || two.Total != "0.67" || one.Differences[0].Amount != "0.34" || two.Differences[0].Amount != "0.17" || two.Differences[1].Amount != "0.17" {
		t.Fatalf("one %+v two %+v", one.Differences, two.Differences)
	}
}

func TestRemoveRecomputesTargets(t *testing.T) {
	r, clock := setup(t)
	v := opened(t, r, sample("d1", false))
	v = must(r.Add("ws", "d1", v.Revision, item("f", EntryFixed, "", "", "10"), human("clerk")))
	v = must(r.Add("ws", "d1", v.Revision, item("t", EntryTargetTotal, "", "150", ""), human("clerk")))
	if v.Differences[1].Amount != "40" || v.Total != "150" {
		t.Fatalf("before %+v", v.Differences)
	}
	clock.advance(1)
	v = must(r.Remove("ws", "d1", v.Revision, "f", human("lead")))
	if len(v.Differences) != 1 || v.Differences[0].Amount != "50" || v.Differences[0].Current != "100" || v.Total != "150" || v.Revision != 4 {
		t.Fatalf("after %+v", v)
	}
	if v.Entries[0].Active || v.Entries[0].RemovedBy != human("lead") || !v.Entries[0].Removed.Equal(start.Add(1)) || !v.Entries[1].Active {
		t.Fatalf("entries %+v", v.Entries)
	}
	again := must(r.Remove("ws", "d1", 3, "f", human("lead")))
	if again.Revision != 4 {
		t.Fatalf("replay %+v", again)
	}
	expect(t, RefuseEntryRemoved, second(r.Remove("ws", "d1", 4, "f", human("lead"))), "remove again")
	expect(t, RefuseEntryRemoved, second(r.Remove("ws", "d1", 3, "f", human("other"))), "remove by another")
	expect(t, "conflict", second(r.Add("ws", "d1", 4, item("f", EntryFixed, "", "", "10"), human("clerk"))), "re-add removed id")
	expect(t, "not_found", second(r.Remove("ws", "d1", 4, "zz", human("lead"))), "remove unknown")
	expect(t, RefuseStaleRevision, second(r.Remove("ws", "d1", 3, "t", human("lead"))), "remove stale")
}

func TestRemovalCanLeaveZeroOrNegative(t *testing.T) {
	r2, _ := setup(t)
	w := opened(t, r2, sample("d1", false))
	w = must(r2.Add("ws", "d1", w.Revision, item("t", EntryTargetTotal, "", "110", ""), human("clerk")))
	w = must(r2.Add("ws", "d1", w.Revision, item("f", EntryFixed, "", "", "-10"), human("clerk")))
	w = must(r2.Add("ws", "d1", w.Revision, item("u", EntryTargetTotal, "", "110", ""), human("clerk")))
	w = must(r2.Remove("ws", "d1", w.Revision, "f", human("clerk")))
	if w.Differences[1].Amount != "0" {
		t.Fatalf("zero %+v", w.Differences)
	}
	expect(t, RefuseZeroDifference, second(r2.Validate("ws", "d1", check(w, "0", w.Total), system("tax"))), "validate zero")
	n := opened(t, r2, sample("d2", false))
	n = must(r2.Add("ws", "d2", n.Revision, item("a", EntryFixed, "", "", "50"), human("clerk")))
	n = must(r2.Add("ws", "d2", n.Revision, item("b", EntryFixed, "", "", "-120"), human("clerk")))
	n = must(r2.Remove("ws", "d2", n.Revision, "a", human("clerk")))
	if n.Total != "-20" {
		t.Fatalf("negative %+v", n)
	}
	expect(t, RefuseNegativeTotal, second(r2.Validate("ws", "d2", check(n, "0", "0"), system("tax"))), "validate negative")
}

func TestAddRefusals(t *testing.T) {
	r, _ := setup(t)
	v := opened(t, r, sample("d1", false))
	expect(t, RefuseZeroDifference, second(r.Add("ws", "d1", 1, item("e", EntryTargetTotal, "", "100", ""), human("clerk"))), "zero target")
	expect(t, RefuseZeroDifference, second(r.Add("ws", "d1", 1, item("e", EntryQuantity, "l1", "10", ""), human("clerk"))), "zero quantity")
	expect(t, RefuseNegativeTotal, second(r.Add("ws", "d1", 1, item("e", EntryFixed, "", "", "-100.01"), human("clerk"))), "negative")
	expect(t, "ok", second(r.Add("ws", "d1", 1, item("e", EntryFixed, "", "", "-100"), human("clerk"))), "to zero")
	expect(t, RefuseNoUnitPrice, second(r.Add("ws", "d1", 2, item("q", EntryQuantity, "l2", "2", ""), human("clerk"))), "no price")
	outside := item("p", EntryFixed, "", "", "1")
	outside.To = "2026-11-01"
	expect(t, RefuseOutsidePeriod, second(r.Add("ws", "d1", 2, outside, human("clerk"))), "after period")
	outside.From, outside.To = "2026-09-30", "2026-10-31"
	expect(t, RefuseOutsidePeriod, second(r.Add("ws", "d1", 2, outside, human("clerk"))), "before period")
	expect(t, RefuseStaleRevision, second(r.Add("ws", "d1", 1, item("s", EntryFixed, "", "", "1"), human("clerk"))), "stale")
	expect(t, "not_found", second(r.Add("ws", "d1", 2, item("s", EntryFixed, "l9", "", "1"), human("clerk"))), "unknown line")
	expect(t, "not_found", second(r.Add("ws", "nope", 1, item("s", EntryFixed, "", "", "1"), human("clerk"))), "unknown draft")
	expect(t, "not_found", second(r.Add("other", "d1", 1, item("s", EntryFixed, "", "", "1"), human("clerk"))), "other workspace")
	expect(t, "invalid", second(r.Add("ws", "d1", 2, item("s", EntryFixed, "", "", "1.001"), human("clerk"))), "off scale fixed")
	expect(t, "invalid", second(r.Add("ws", "d1", 2, item("s", EntryTargetTotal, "", "1.001", ""), human("clerk"))), "off scale target")
	again := must(r.Add("ws", "d1", 1, item("e", EntryFixed, "", "", "-100"), human("clerk")))
	if again.Revision != 2 {
		t.Fatalf("replay %+v", again)
	}
	expect(t, "conflict", second(r.Add("ws", "d1", 1, item("e", EntryFixed, "", "", "-99"), human("clerk"))), "changed replay")
	expect(t, "conflict", second(r.Add("ws", "d1", 1, item("e", EntryFixed, "", "", "-100"), human("other"))), "replay by another")
	expect(t, "conflict", second(r.Add("ws", "d1", 2, item("e", EntryFixed, "", "", "-100"), human("clerk"))), "replay at another revision")
	_ = v
}

func TestSignedLineTargetAndCredits(t *testing.T) {
	r, _ := setup(t)
	d := sample("d1", false)
	d.Lines = append(d.Lines, SourceLine{ID: "l3", Charge: "ch-3", Quantity: "1", Amount: "-20"})
	v := opened(t, r, d)
	v = must(r.Add("ws", "d1", v.Revision, item("e", EntryTargetTotal, "l3", "-30", ""), human("clerk")))
	if v.Differences[0].Amount != "-10" || v.Lines[2].Adjusted != "-30" || v.Total != "70" || v.Gross != "10" {
		t.Fatalf("credit %+v", v)
	}
}

func TestSourceErrorsStayVisible(t *testing.T) {
	r, _ := setup(t)
	v := opened(t, r, sample("d1", false))
	e := item("e1", EntryFixed, "l1", "", "3")
	e.Reason = ReasonSourceError
	expect(t, "invalid", second(r.Add("ws", "d1", 1, e, human("clerk"))), "source error without finding")
	e.Finding = "finding-7"
	v = must(r.Add("ws", "d1", 1, e, human("clerk")))
	f := item("e2", EntryFixed, "l2", "", "1")
	f.Finding = "finding-7"
	v = must(r.Add("ws", "d1", v.Revision, f, human("clerk")))
	g := item("e3", EntryFixed, "l2", "", "1")
	g.Finding = "finding-8"
	v = must(r.Add("ws", "d1", v.Revision, g, human("clerk")))
	if !reflect.DeepEqual(v.Findings, []string{"finding-7", "finding-8"}) {
		t.Fatalf("findings %+v", v.Findings)
	}
	v = must(r.Remove("ws", "d1", v.Revision, "e3", human("clerk")))
	if !reflect.DeepEqual(v.Findings, []string{"finding-7"}) {
		t.Fatalf("findings after removal %+v", v.Findings)
	}
}

func TestAuditAR07ChangedDraftNeedsNewApproval(t *testing.T) {
	r, _ := setup(t)
	v := ready(t, r, "d1", item("e1", EntryTargetTotal, "", "120", ""))
	if v.Subject == "" || v.Check == nil || v.Check.Actor != system("tax-engine") {
		t.Fatalf("validated %+v", v)
	}
	approved := release("r1", v, human("ann"))
	changed := must(r.Add("ws", "d1", v.Revision, item("e2", EntryFixed, "l2", "", "1"), human("clerk")))
	if changed.Subject != "" || changed.Digest == v.Digest {
		t.Fatalf("changed %+v", changed)
	}
	expect(t, RefuseStaleRevision, second(r.Release("ws", "d1", approved, system("runner"))), "old approval after change")
	moved := approved
	moved.Revision = changed.Revision
	expect(t, RefuseNotValidated, second(r.Release("ws", "d1", moved, system("runner"))), "unvalidated revision")
	revalidated := must(r.Validate("ws", "d1", check(changed, "0", changed.Total), system("tax-engine")))
	if revalidated.Subject == v.Subject || revalidated.Subject == "" {
		t.Fatalf("subject %+v", revalidated)
	}
	expect(t, RefuseDigestChanged, second(r.Release("ws", "d1", moved, system("runner"))), "old subject on new revision")
	out := must(r.Release("ws", "d1", release("r2", revalidated, human("ann")), system("runner")))
	if out.Disposition != DispositionNewRun || out.Run != "run-1" || out.Document != "" || out.Total != "121" || out.Subject != revalidated.Subject || out.Digest != revalidated.Digest {
		t.Fatalf("outcome %+v", out)
	}
	after := must(r.Of("ws", "d1"))
	if after.State != StateReleased || after.Outcome == nil || after.Outcome.Release.Request != "r2" || after.Lines[1].Line.Amount != "75" {
		t.Fatalf("after %+v", after)
	}
}

func TestIssuedBaseBecomesCorrection(t *testing.T) {
	r, _ := setup(t)
	v := opened(t, r, sample("d1", true))
	v = must(r.Add("ws", "d1", v.Revision, item("e1", EntryFixed, "l2", "", "-5"), human("clerk")))
	v = must(r.Validate("ws", "d1", check(v, "-0.5", "94.5"), human("reviewer")))
	out := must(r.Release("ws", "d1", release("r1", v, human("ann")), human("operator")))
	if out.Disposition != DispositionCorrection || out.Document != "INV-1" || out.Run != "run-1" || out.TaxAmount != "-0.5" || out.Final != "94.5" ||
		out.Tax != digest('c') || out.Allocation != digest('d') || out.Gross != "5" || out.Adjustment != "-5" || out.Source != "100" {
		t.Fatalf("outcome %+v", out)
	}
	if !out.Released.Equal(start) || out.Actor != human("operator") || out.Release.Approval != "ap-r1" {
		t.Fatalf("outcome meta %+v", out)
	}
}

func TestBaseIssuedAfterOpenIsRefused(t *testing.T) {
	r, _ := setup(t)
	v := ready(t, r, "d1", item("e1", EntryFixed, "", "", "5"))
	q := release("r1", v, human("ann"))
	q.Base.Issued, q.Base.Document = true, "INV-9"
	expect(t, RefuseBaseChanged, second(r.Release("ws", "d1", q, system("runner"))), "issued since")
	q = release("r1", v, human("ann"))
	q.Base.Digest = digest('e')
	expect(t, RefuseBaseChanged, second(r.Release("ws", "d1", q, system("runner"))), "run digest changed")
	q = release("r1", v, human("ann"))
	q.Base.Run = "run-2"
	expect(t, RefuseBaseChanged, second(r.Release("ws", "d1", q, system("runner"))), "run changed")
	for _, change := range []func(*Base){
		func(b *Base) { b.Currency = "EUR" },
		func(b *Base) { b.Scale = 4 },
		func(b *Base) { b.Mode = "floor" },
		func(b *Base) { b.Document = "INV-2"; b.Issued = true },
	} {
		q = release("r1", v, human("ann"))
		change(&q.Base)
		expect(t, RefuseBaseChanged, second(r.Release("ws", "d1", q, system("runner"))), "base field")
	}
	q = release("r1", v, human("ann"))
	q.Base.Mode = "nearest"
	expect(t, "invalid", second(r.Release("ws", "d1", q, system("runner"))), "base form")
	out := must(r.Release("ws", "d1", release("r1", v, human("ann")), system("runner")))
	if out.Currency != "USD" || out.Scale != 2 {
		t.Fatalf("outcome base %+v", out)
	}
}

func TestOneReleasePerRun(t *testing.T) {
	r, _ := setup(t)
	a := ready(t, r, "d1", item("e1", EntryFixed, "", "", "5"))
	b := ready(t, r, "d2", item("e1", EntryTargetTotal, "", "120", ""))
	must(r.Release("ws", "d1", release("r1", a, human("ann")), system("runner")))
	expect(t, RefuseBaseConsumed, second(r.Release("ws", "d2", release("r2", b, human("ann")), system("runner"))), "second draft same run")
	d := sample("d3", false)
	d.Base.Digest = digest('f')
	c := opened(t, r, d)
	c = must(r.Add("ws", "d3", c.Revision, item("e1", EntryFixed, "", "", "5"), human("clerk")))
	c = must(r.Validate("ws", "d3", check(c, "0", c.Total), system("tax")))
	expect(t, "ok", second(r.Release("ws", "d3", release("r3", c, human("ann")), system("runner"))), "new run digest")
}

func TestWithdraw(t *testing.T) {
	r, clock := setup(t)
	v := ready(t, r, "d1", item("e1", EntryFixed, "", "", "5"))
	expect(t, RefuseAINotAllowed, second(r.Withdraw("ws", "d1", assistant("bot"))), "assistant withdraw")
	clock.advance(5)
	w := must(r.Withdraw("ws", "d1", human("lead")))
	if w.State != StateWithdrawn || w.Withdrawer != human("lead") || !w.Withdrawn.Equal(start.Add(5)) {
		t.Fatalf("withdrawn %+v", w)
	}
	again := must(r.Withdraw("ws", "d1", assistant("bot")))
	if again.Withdrawer != human("lead") {
		t.Fatalf("again %+v", again)
	}
	expect(t, RefuseWithdrawn, second(r.Add("ws", "d1", v.Revision, item("e2", EntryFixed, "", "", "1"), human("clerk"))), "add after withdraw")
	expect(t, RefuseWithdrawn, second(r.Remove("ws", "d1", v.Revision, "e1", human("clerk"))), "remove after withdraw")
	expect(t, RefuseWithdrawn, second(r.Validate("ws", "d1", Check{Revision: v.Revision + 1, Digest: v.Digest, Tax: digest('1'), Allocation: digest('d'), TaxAmount: "0", Final: v.Total}, system("tax"))), "validate after withdraw")
	expect(t, RefuseWithdrawn, second(r.Release("ws", "d1", release("r1", v, human("ann")), system("runner"))), "release after withdraw")
	expect(t, "not_found", second(r.Withdraw("ws", "zz", human("lead"))), "unknown")
}

func TestReleasedDraftIsClosed(t *testing.T) {
	r, _ := setup(t)
	v := ready(t, r, "d1", item("e1", EntryFixed, "", "", "5"))
	q := release("r1", v, human("ann"), human("bob"))
	out := must(r.Release("ws", "d1", q, system("runner")))
	q.Approvers[0] = human("zed")
	if out.Release.Approvers[0] != human("ann") {
		t.Fatal("approvers aliased")
	}
	q = release("r1", v, human("ann"), human("bob"))
	again := must(r.Release("ws", "d1", q, system("runner")))
	if !reflect.DeepEqual(again, out) {
		t.Fatalf("replay %+v", again)
	}
	expect(t, "conflict", second(r.Release("ws", "d1", q, system("other"))), "replay by another actor")
	expect(t, RefuseAlreadyReleased, second(r.Release("ws", "d1", release("r9", v, human("ann")), system("runner"))), "second request")
	expect(t, RefuseReleased, second(r.Add("ws", "d1", v.Revision, item("e2", EntryFixed, "", "", "1"), human("clerk"))), "add")
	expect(t, RefuseReleased, second(r.Remove("ws", "d1", v.Revision, "e1", human("clerk"))), "remove")
	expect(t, RefuseReleased, second(r.Withdraw("ws", "d1", human("lead"))), "withdraw")
	expect(t, RefuseReleased, second(r.Validate("ws", "d1", Check{Revision: v.Revision + 1, Digest: v.Digest, Tax: digest('1'), Allocation: digest('d'), TaxAmount: "0", Final: v.Total}, system("tax"))), "validate")
	view := must(r.Of("ws", "d1"))
	view.Outcome.Differences[0].Amount = "999"
	view.Outcome.Release.Approvers[0] = human("zed")
	view.Entries[0].Entry.Evidence[0] = "changed"
	view.Draft.Lines[0].Amount = "1"
	view.Check.Actor = human("zed")
	fresh := must(r.Of("ws", "d1"))
	if fresh.Outcome.Differences[0].Amount != "5" || fresh.Outcome.Release.Approvers[0] != human("ann") || fresh.Entries[0].Entry.Evidence[0] != "ev-e1" ||
		fresh.Draft.Lines[0].Amount != "25" || fresh.Check.Actor != system("tax-engine") {
		t.Fatalf("views aliased %+v", fresh)
	}
}
