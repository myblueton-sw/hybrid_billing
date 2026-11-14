package erpledger

import (
	"testing"
	"time"
)

func TestFileFirstReconciliationWithoutAPI(t *testing.T) {
	f := newFixture(t)
	f.connect(t, fileConnection("file"))
	v := must(f.r.ConnectionOf("w1", "file"))
	if v.Verified {
		t.Fatal("template verified by default")
	}
	must(f.r.Prepare("w1", invoice("k1", "file", line("l1", "s1", "100")), human("biller")))
	expect(t, "unverified template", second(f.r.Command("w1", "k1", CommandSubmit, human("biller"))), ReasonTemplateUnverified)
	expect(t, "author verifies", second(f.r.VerifyTemplate("w1", "file", "uat-2026-11", human("admin"))), ReasonSelfApproval)
	must(f.r.VerifyTemplate("w1", "file", "uat-2026-11", human("qa")))
	must(f.r.Command("w1", "k1", CommandSubmit, human("biller")))
	expect(t, "no post capability", second(f.r.Command("w1", "k1", CommandPost, human("biller"))), ReasonUnsupported)
	f.observe(t, "file", "k1", "INV-9", 1, StatePosted, SourceResultFile)
	must(f.r.Document("w1", Document{Connection: "file", External: "INV-9", Revision: 1, Rows: []Row{{ID: "r1", Product: "P-1", Net: "100", Tax: "0", Quantity: "1"}}}, system("connector")))
	must(f.r.Link("w1", Link{Connection: "file", External: "INV-9", Row: "r1", Key: "k1", Line: "l1", Net: "100", Tax: "0", Quantity: "1"}, human("biller")))
	rec := must(f.r.Reconcile("w1", "k1"))
	if !rec.Matched || rec.Documents[0].Attributed.Net != "100" {
		t.Fatalf("reconcile %+v", rec)
	}
	sub := must(f.r.SubmissionOf("w1", "k1"))
	for _, c := range sub.Commands {
		if c.Name != CommandSubmit {
			t.Fatalf("api command used %+v", c)
		}
	}
}

func TestExistingPathsBlockDoubleBilling(t *testing.T) {
	f := newFixture(t)
	must(f.r.Anchor("w1", Anchor{ID: "so-1", Connection: "api", Kind: AnchorSalesOrder, External: "SO-1", Customer: "acme", Sources: []string{"s1"}}, human("admin")))
	must(f.r.Anchor("w1", Anchor{ID: "ap-1", Connection: "api", Kind: AnchorPayable, External: "AP-1", Customer: "acme", Sources: []string{"s2"}}, human("admin")))
	expect(t, "direct over order", second(f.r.Prepare("w1", invoice("k1", "api", line("l1", "s1", "100")), human("b"))), ReasonAlreadyReflected)
	order := invoice("k1", "api", line("l1", "s1", "100"))
	order.Path, order.Anchor = PathOrderBased, "so-1"
	must(f.r.Prepare("w1", order, human("b")))
	payable := invoice("k2", "api", line("l1", "s2", "40"))
	payable.Purpose = PurposePayable
	expect(t, "payable over AP", second(f.r.Prepare("w1", payable, human("b"))), ReasonAlreadyReflected)
	payable.Lines = []Line{line("l1", "s3", "40")}
	must(f.r.Prepare("w1", payable, human("b")))
	expect(t, "invoice over AP ok", second(f.r.Prepare("w1", invoice("k3", "api", line("l1", "s2", "40")), human("b"))), "ok")
	other := invoice("k4", "api", line("l1", "s9", "1"))
	other.Period = "2026-11"
	expect(t, "not surveyed", second(f.r.Prepare("w1", other, human("b"))), ReasonNotSurveyed)
	must(f.r.Survey("w1", Survey{Connection: "api", Customer: "acme", Period: "2026-11", Outcome: SurveyUnknown, Evidence: "erp-down"}, human("admin")))
	expect(t, "ledger unknown", second(f.r.Prepare("w1", other, human("b"))), ReasonLedgerUnknown)
}

func TestAnchorAfterPrepareMarksDiverged(t *testing.T) {
	f := newFixture(t)
	must(f.r.Prepare("w1", invoice("k1", "api", line("l1", "s1", "100")), human("biller")))
	must(f.r.Anchor("w1", Anchor{ID: "pp-1", Connection: "api", Kind: AnchorPrepaid, External: "PP-1", Customer: "acme", Sources: []string{"s1"}}, human("admin")))
	v := must(f.r.SubmissionOf("w1", "k1"))
	if !v.Diverged {
		t.Fatal("not diverged")
	}
	expect(t, "send diverged", second(f.r.Command("w1", "k1", CommandSubmit, human("biller"))), ReasonDiverged)
	expect(t, "keep anchored", second(f.r.Resolve("w1", "k1", Resolution{Outcome: ResolveKeep, Evidence: "x"}, human("lead"))), ReasonAlreadyReflected)
	w := must(f.r.Withdraw("w1", "k1", human("biller")))
	if w.State != StateWithdrawn {
		t.Fatalf("withdraw %+v", w)
	}
}

func TestMergedAndSplitDocuments(t *testing.T) {
	f := newFixture(t)
	sub := invoice("k1", "api", line("l1", "s1", "100"), Line{ID: "l2", Source: "s2", Product: "storage", TaxCode: "vat10", Quantity: "2", Net: "50", Tax: "5"})
	f.send(t, sub)
	f.observe(t, "api", "k1", "INV-1", 1, StatePosted, SourceWebhook)
	must(f.r.Document("w1", Document{Connection: "api", External: "INV-1", Revision: 1, Rows: []Row{
		{ID: "r1", Product: "P-1", Net: "100", Tax: "0", Quantity: "1"},
		{ID: "r2", Product: "SUPPORT", Net: "900", Tax: "90", Quantity: "1"},
		{ID: "r3", Product: "P-2", Net: "30", Tax: "3", Quantity: "1"},
	}}, system("connector")))
	must(f.r.Document("w1", Document{Connection: "api", External: "INV-2", Revision: 1, Rows: []Row{{ID: "r1", Product: "P-2", Net: "20", Tax: "2", Quantity: "1"}}}, system("connector")))
	must(f.r.Link("w1", Link{Connection: "api", External: "INV-1", Row: "r1", Key: "k1", Line: "l1", Net: "100", Tax: "0", Quantity: "1"}, human("b")))
	must(f.r.Link("w1", Link{Connection: "api", External: "INV-1", Row: "r3", Key: "k1", Line: "l2", Net: "30", Tax: "3", Quantity: "1"}, human("b")))
	if rec := must(f.r.Reconcile("w1", "k1")); rec.Matched {
		t.Fatalf("partial matched %+v", rec)
	}
	must(f.r.Link("w1", Link{Connection: "api", External: "INV-2", Row: "r1", Key: "k1", Line: "l2", Net: "20", Tax: "2", Quantity: "1"}, human("b")))
	rec := must(f.r.Reconcile("w1", "k1"))
	if !rec.Matched || len(rec.Documents) != 2 {
		t.Fatalf("split %+v", rec)
	}
	d := rec.Documents[0]
	if d.External != "INV-1" || d.Total != (Amounts{"1030", "93", "3"}) || d.Attributed != (Amounts{"130", "3", "2"}) {
		t.Fatalf("merged totals %+v", d)
	}
	expect(t, "over row", second(f.r.Link("w1", Link{Connection: "api", External: "INV-1", Row: "r2", Key: "k1", Line: "l1", Net: "1", Tax: "0", Quantity: "0"}, human("b"))), ReasonOverAllocated)
}

func TestDatabaseTransportAndUnsupportedCommands(t *testing.T) {
	f := newFixture(t)
	db := apiConnection("db")
	db.Transport = TransportDB
	expect(t, "database", second(f.r.Connect("w1", db, human("admin"))), ReasonDirectDatabase)
	narrow := apiConnection("narrow")
	narrow.Capabilities = []string{CommandPrepare, CommandMapping, CommandSubmit}
	f.connect(t, narrow)
	f.send(t, invoice("k1", "narrow", line("l1", "s1", "1")))
	for _, name := range []string{CommandPost, CommandStatus, CommandCorrect, CommandReadDraft, CommandBalance} {
		expect(t, name, second(f.r.Command("w1", "k1", name, human("b"))), ReasonUnsupported)
	}
	if v := must(f.r.SubmissionOf("w1", "k1")); len(v.Commands) != 1 || v.State != StateSent {
		t.Fatalf("unsupported changed state %+v", v)
	}
}

func TestMissingAndReorderedWebhooksRecoveredByPolling(t *testing.T) {
	f := newFixture(t)
	f.send(t, invoice("k1", "api", line("l1", "s1", "100")))
	if due := must(f.r.Due("w1", start.Add(time.Hour))); len(due) != 1 || due[0] != "k1" {
		t.Fatalf("due %v", due)
	}
	f.observe(t, "api", "k1", "INV-1", 3, StatePosted, SourcePoll)
	stale := f.observe(t, "api", "", "INV-1", 2, StateDraft, SourceWebhook)
	if stale.Outcome != "stale" {
		t.Fatalf("stale %+v", stale)
	}
	v := must(f.r.SubmissionOf("w1", "k1"))
	if v.State != StatePosted || v.External != "INV-1" || v.Revision != 3 || v.Watch.Last != start {
		t.Fatalf("recovered %+v", v)
	}
	orphan := f.observe(t, "api", "", "INV-404", 1, StatePosted, SourceWebhook)
	if orphan.Outcome != "orphan" || len(must(f.r.Orphans("w1"))) != 1 {
		t.Fatalf("orphan %+v", orphan)
	}
	expect(t, "posted source still blocked", second(f.r.Prepare("w1", invoice("k2", "api", line("l1", "s1", "100")), human("b"))), ReasonAlreadySubmitted)
}

func TestTimeoutNeedsConfirmedAbsence(t *testing.T) {
	f := newFixture(t)
	f.send(t, invoice("k1", "api", line("l1", "s1", "100")))
	f.clock.advance(time.Hour)
	v := must(f.r.Checked("w1", "k1", "timeout", system("poller")))
	if v.State != StateUnknown || v.Watch.Retries != 1 {
		t.Fatalf("timeout %+v", v)
	}
	expect(t, "new key while unknown", second(f.r.Prepare("w1", invoice("k2", "api", line("l1", "s1", "100")), human("b"))), ReasonAlreadySubmitted)
	expect(t, "webhook absence", second(f.r.Observe("w1", Observation{ID: "w", Connection: "api", Key: "k1", Revision: 1, State: StateAbsent, Source: SourceWebhook}, system("c"))), "invalid")
	f.observe(t, "api", "k1", "", 1, StateAbsent, SourcePoll)
	must(f.r.Prepare("w1", invoice("k2", "api", line("l1", "s1", "100")), human("b")))
}

func TestPrematureAbsenceMovesToUnknown(t *testing.T) {
	f := newFixture(t)
	f.send(t, invoice("k1", "api", line("l1", "s1", "100")))
	o := f.observe(t, "api", "k1", "", 1, StateAbsent, SourcePoll)
	v := must(f.r.SubmissionOf("w1", "k1"))
	if o.Outcome != "premature" || v.State != StateUnknown {
		t.Fatalf("premature %+v %+v", o, v)
	}
	expect(t, "still blocked", second(f.r.Prepare("w1", invoice("k2", "api", line("l1", "s1", "100")), human("b"))), ReasonAlreadySubmitted)
	f.observe(t, "api", "k1", "", 2, StateAbsent, SourcePoll)
	if v := must(f.r.SubmissionOf("w1", "k1")); v.State != StateAbsent {
		t.Fatalf("confirmed %+v", v)
	}
}

func TestWrongAbsenceAndManualEdit(t *testing.T) {
	f := newFixture(t)
	f.send(t, invoice("k1", "api", line("l1", "s1", "100")))
	f.observe(t, "api", "k1", "", 1, StateUnknown, SourcePoll)
	f.observe(t, "api", "k1", "", 2, StateAbsent, SourcePoll)
	f.send(t, invoice("k2", "api", line("l1", "s1", "100")))
	f.observe(t, "api", "k2", "INV-2", 1, StatePosted, SourceWebhook)
	late := f.observe(t, "api", "k1", "INV-1", 3, StatePosted, SourceWebhook)
	if late.Outcome != "diverged" {
		t.Fatalf("late %+v", late)
	}
	a, b := must(f.r.SubmissionOf("w1", "k1")), must(f.r.SubmissionOf("w1", "k2"))
	if !a.Diverged || !b.Diverged || a.State != StateAbsent {
		t.Fatalf("both diverged %+v %+v", a, b)
	}
	expect(t, "accept original first", second(f.r.Resolve("w1", "k1", Resolution{Outcome: ResolveAccept, Evidence: "erp-check"}, human("lead"))), ReasonAlreadySubmitted)
	expect(t, "keep contradicted absence", second(f.r.Resolve("w1", "k1", Resolution{Outcome: ResolveKeep, Evidence: "erp-check"}, human("lead"))), ReasonWrongState)
	must(f.r.Resolve("w1", "k2", Resolution{Outcome: ResolveSuperseded, Evidence: "credit-note-CN-2"}, human("lead")))
	got := must(f.r.Resolve("w1", "k1", Resolution{Outcome: ResolveAccept, Evidence: "erp-check"}, human("lead")))
	if got.State != StatePosted || got.External != "INV-1" || got.Diverged {
		t.Fatalf("accepted %+v", got)
	}
	f.observe(t, "api", "", "INV-1", 4, StateRejected, SourceWebhook)
	if v := must(f.r.SubmissionOf("w1", "k1")); !v.Diverged || v.State != StatePosted {
		t.Fatalf("manual edit %+v", v)
	}
	expect(t, "post while diverged", second(f.r.Command("w1", "k1", CommandCorrect, human("b"))), ReasonDiverged)
}

func TestExternalSuccessWithLocalSaveFailure(t *testing.T) {
	f := newFixture(t)
	sub := invoice("k1", "api", line("l1", "s1", "100"))
	f.send(t, sub)
	again := must(f.r.Prepare("w1", sub, human("biller")))
	if again.State != StateSent {
		t.Fatalf("repeat %+v", again)
	}
	f.observe(t, "api", "k1", "INV-1", 1, StatePosted, SourcePoll)
	expect(t, "other key", second(f.r.Prepare("w1", invoice("k9", "api", line("l1", "s1", "100")), human("biller"))), ReasonAlreadySubmitted)
}

func TestSwitchFromAPIToFileNeverRebills(t *testing.T) {
	f := newFixture(t)
	f.send(t, invoice("k1", "api", line("l1", "s1", "100")))
	f.connect(t, fileConnection("file"))
	expect(t, "file re-bill", second(f.r.Prepare("w1", invoice("k2", "file", line("l1", "s1", "100")), human("b"))), ReasonAlreadySubmitted)
	must(f.r.Anchor("w1", Anchor{ID: "so", Connection: "api", Kind: AnchorSalesOrder, External: "SO", Customer: "acme", Sources: []string{"s5"}}, human("admin")))
	expect(t, "anchor across connections", second(f.r.Prepare("w1", invoice("k3", "file", line("l1", "s5", "1")), human("b"))), ReasonAlreadyReflected)
}
