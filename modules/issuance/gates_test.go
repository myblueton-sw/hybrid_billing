package issuance

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/protected"
	"github.com/myblueton-sw/hybrid_billing/modules/receivables"
)

func TestNewRegistryNeedsEveryDependency(t *testing.T) {
	f := open(t, testLimits())
	full := Config{Fields: f.fields, Terms: f.terms, Details: f.details, Renderer: f.renderer, Zones: time.LoadLocation}
	broken := []Config{full, full, full, full, full}
	broken[0].Fields = nil
	broken[1].Terms = nil
	broken[2].Details = nil
	broken[3].Renderer = nil
	broken[4].Zones = nil
	for index, c := range broken {
		if _, err := NewRegistry(c, testLimits(), f.clock.read); !errors.Is(err, ErrInvalid) {
			t.Errorf("config %d: %v", index, err)
		}
	}
	if _, err := NewRegistry(full, testLimits(), nil); !errors.Is(err, ErrInvalid) {
		t.Error("clock")
	}
	limits := []func(*Limits){
		func(l *Limits) { l.Workspaces = 0 }, func(l *Limits) { l.Workspaces = maxWorkspaces + 1 }, func(l *Limits) { l.Kinds = 0 },
		func(l *Limits) { l.Drafts = 0 }, func(l *Limits) { l.Documents = 0 }, func(l *Limits) { l.Numbers = maxNumbers + 1 },
		func(l *Limits) { l.Attempts = 0 }, func(l *Limits) { l.Downloads = 0 }, func(l *Limits) { l.Downloads = maxDownloads + 1 }, func(l *Limits) { l.Lease = minLease - 1 }, func(l *Limits) { l.Lease = maxLease + 1 },
	}
	for index, change := range limits {
		l := testLimits()
		change(&l)
		if _, err := NewRegistry(full, l, f.clock.read); !errors.Is(err, ErrInvalid) {
			t.Errorf("limits %d: %v", index, err)
		}
	}
	edge := Limits{Workspaces: maxWorkspaces, Kinds: maxKinds, Drafts: maxDrafts, Documents: maxDocuments, Numbers: maxNumbers, Attempts: maxAttempts, Downloads: maxDownloads, Lease: maxLease}
	if _, err := NewRegistry(full, edge, f.clock.read); err != nil {
		t.Error(err)
	}
	edge = Limits{Workspaces: 1, Kinds: 1, Drafts: 1, Documents: 1, Numbers: 1, Attempts: 1, Downloads: 1, Lease: minLease}
	if _, err := NewRegistry(full, edge, f.clock.read); err != nil {
		t.Error(err)
	}
	if (Refusal{Reason: "x"}).Error() != ErrRejected.Error()+": x" || !errors.Is(Refusal{Reason: "x"}, ErrRejected) {
		t.Error("refusal")
	}
}

func TestKindsNeedFormDualControlAndTaxReview(t *testing.T) {
	f := open(t, testLimits())
	base := kinds[0]
	forms := []func(*Kind){
		func(k *Kind) { k.ID = "" }, func(k *Kind) { k.ID = strings.Repeat("k", maxSubject+1) }, func(k *Kind) { k.Class = "other" },
		func(k *Kind) { k.Region = "K" }, func(k *Kind) { k.Region = "kr" }, func(k *Kind) { k.Region = "K1" }, func(k *Kind) { k.Region = "@R" },
		func(k *Kind) { k.Tax = "" }, func(k *Kind) { k.Languages = nil }, func(k *Kind) { k.Languages = []string{"en", "en"} },
		func(k *Kind) { k.Languages = []string{"fr"} }, func(k *Kind) { k.Languages = []string{"en", "ko", "ar", "en"} },
		func(k *Kind) { k.Zone = "" }, func(k *Kind) { k.Zone = "Local" }, func(k *Kind) { k.Zone = "Mars/Base" },
		func(k *Kind) { k.Zone = strings.Repeat("Z", maxIDBytes+1) }, func(k *Kind) { k.Class, k.Statutory = ClassInternal, true },
	}
	for _, change := range forms {
		k := base
		k.Languages = append([]string(nil), base.Languages...)
		change(&k)
		expect(t, "kind form", second(f.r.Define("ws", k, human("definer"))), "invalid")
	}
	expect(t, "workspace", second(f.r.Define("", base, human("definer"))), "invalid")
	expect(t, "actor", second(f.r.Define("ws", base, Actor{ID: "x", Kind: "robot"})), "invalid")
	expect(t, "assistant", second(f.r.Define("ws", base, assistant("bot"))), ReasonAINotAllowed)
	expect(t, "system", second(f.r.Define("ws", base, system("job"))), ReasonNotHuman)
	expect(t, "unknown", second(f.r.ApproveKind("ws", "inv", human("approver"))), "not_found")
	v := must(f.r.Define("ws", base, human("definer")))
	if v.Kind.Languages[0] != "ar" || v.State != StateProposed {
		t.Fatalf("define: %+v", v)
	}
	v.Kind.Languages[0] = "zz"
	if must(f.r.KindOf("ws", "inv")).Kind.Languages[0] != "ar" {
		t.Fatal("a view shares languages")
	}
	sorted := base
	sorted.Languages = []string{"ar", "en", "ko"}
	must(f.r.Define("ws", sorted, human("definer")))
	expect(t, "other definer", second(f.r.Define("ws", sorted, human("other"))), "conflict")
	for _, change := range []func(*Kind){
		func(k *Kind) { k.Class = ClassERP }, func(k *Kind) { k.Region = "US" }, func(k *Kind) { k.Tax = "x" }, func(k *Kind) { k.Statutory = false },
		func(k *Kind) { k.Credit = true }, func(k *Kind) { k.Sequential = true }, func(k *Kind) { k.Zone = "UTC" },
		func(k *Kind) { k.Languages = []string{"ar", "en"} }, func(k *Kind) { k.Languages = []string{"ar", "en", "en"}[:2] },
	} {
		k := sorted
		k.Languages = append([]string(nil), sorted.Languages...)
		change(&k)
		if err := second(f.r.Define("ws", k, human("definer"))); reason(err) != "conflict" && reason(err) != "invalid" {
			t.Fatalf("changed kind: %v", err)
		}
	}
	k := sorted
	k.Languages = []string{"ar", "ko", "en"}
	if must(f.r.Define("ws", k, human("definer"))).Kind.Languages[0] != "ar" {
		t.Fatal("an unsorted repeat")
	}
	expect(t, "tax before approval", second(f.r.VerifyTax("ws", "inv", "ev", human("taxer"))), ReasonKindPending)
	expect(t, "self approval", second(f.r.ApproveKind("ws", "inv", human("definer"))), ReasonSelfApproval)
	expect(t, "approve form", second(f.r.ApproveKind("ws", "", human("approver"))), "invalid")
	expect(t, "approve assistant", second(f.r.ApproveKind("ws", "inv", assistant("bot"))), ReasonAINotAllowed)
	must(f.r.ApproveKind("ws", "inv", human("approver")))
	if must(f.r.ApproveKind("ws", "inv", human("approver"))).Approver != "approver" {
		t.Fatal("repeat approval")
	}
	expect(t, "second approver", second(f.r.ApproveKind("ws", "inv", human("other"))), "conflict")
	expect(t, "tax self", second(f.r.VerifyTax("ws", "inv", "ev", human("definer"))), ReasonSelfReview)
	expect(t, "tax form", second(f.r.VerifyTax("ws", "inv", "bad evidence", human("taxer"))), "invalid")
	expect(t, "tax assistant", second(f.r.VerifyTax("ws", "inv", "ev", assistant("bot"))), ReasonAINotAllowed)
	expect(t, "tax unknown", second(f.r.VerifyTax("ws", "nope", "ev", human("taxer"))), "not_found")
	must(f.r.VerifyTax("ws", "inv", "ev", human("taxer")))
	if must(f.r.VerifyTax("ws", "inv", "ev", human("taxer"))).TaxEvidence != "ev" {
		t.Fatal("repeat tax")
	}
	expect(t, "other evidence", second(f.r.VerifyTax("ws", "inv", "ev2", human("taxer"))), "conflict")
	expect(t, "other reviewer", second(f.r.VerifyTax("ws", "inv", "ev", human("x"))), "conflict")
	pair := Kind{ID: "pair", Class: ClassSelf, Region: "AE", Tax: "ae_vat", Zone: "Asia/Dubai", Languages: []string{"en", "ar"}}
	must(f.r.Define("ws", pair, human("definer")))
	pair.Languages = []string{"ko", "ar"}
	expect(t, "same count other languages", second(f.r.Define("ws", pair, human("definer"))), "conflict")
	plain := kinds[2]
	must(f.r.Define("ws", plain, human("definer")))
	must(f.r.ApproveKind("ws", plain.ID, human("approver")))
	expect(t, "not statutory", second(f.r.VerifyTax("ws", plain.ID, "ev", human("taxer"))), ReasonNotStatutory)
	expect(t, "kindof form", second(f.r.KindOf("", "inv")), "invalid")
	expect(t, "kindof unknown", second(f.r.KindOf("ws", "nope")), "not_found")
	expect(t, "kindof workspace", second(f.r.KindOf("other", "inv")), "not_found")
}

func TestKindAndWorkspaceLimits(t *testing.T) {
	l := testLimits()
	l.Workspaces, l.Kinds = 1, 1
	f := open(t, l)
	must(f.r.Define("ws", kinds[0], human("definer")))
	expect(t, "kinds", second(f.r.Define("ws", kinds[2], human("definer"))), "limit")
	expect(t, "workspaces", second(f.r.Define("ws2", kinds[0], human("definer"))), "limit")
}

func TestSeriesFormsAndPrefixes(t *testing.T) {
	f := open(t, testLimits())
	for _, k := range kinds {
		must(f.r.Define("ws", k, human("definer")))
	}
	ok := Series{ID: "s", Kind: "inv", Prefix: "INV-", Width: 6, First: 1}
	expect(t, "pending kind", second(f.r.ProposeSeries("ws", ok, human("definer"))), ReasonKindPending)
	for _, k := range kinds {
		must(f.r.ApproveKind("ws", k.ID, human("approver")))
	}
	for _, change := range []func(*Series){
		func(v *Series) { v.ID = "" }, func(v *Series) { v.Kind = "" }, func(v *Series) { v.Prefix = "" }, func(v *Series) { v.Prefix = "INV1" },
		func(v *Series) { v.Prefix = "INV0" }, func(v *Series) { v.Prefix = "IN V" }, func(v *Series) { v.Width = 0 }, func(v *Series) { v.Width = maxWidth + 1 },
		func(v *Series) { v.First = 0 }, func(v *Series) { v.Width, v.First = 2, 100 },
	} {
		v := ok
		change(&v)
		expect(t, "series form", second(f.r.ProposeSeries("ws", v, human("definer"))), "invalid")
	}
	expect(t, "workspace", second(f.r.ProposeSeries("", ok, human("definer"))), "invalid")
	expect(t, "assistant", second(f.r.ProposeSeries("ws", ok, assistant("bot"))), ReasonAINotAllowed)
	expect(t, "unknown kind", second(f.r.ProposeSeries("ws", Series{ID: "s", Kind: "nope", Prefix: "N-", Width: 3, First: 1}, human("definer"))), "not_found")
	expect(t, "erp", second(f.r.ProposeSeries("ws", Series{ID: "s", Kind: "erp", Prefix: "E-", Width: 3, First: 1}, human("definer"))), ReasonERPNumbered)
	v := must(f.r.ProposeSeries("ws", ok, human("definer")))
	if v.Next != 1 || v.State != StateProposed {
		t.Fatalf("propose: %+v", v)
	}
	if must(f.r.ProposeSeries("ws", ok, human("definer"))).Proposer != "definer" {
		t.Fatal("repeat")
	}
	expect(t, "other proposer", second(f.r.ProposeSeries("ws", ok, human("other"))), "conflict")
	changed := ok
	changed.Width = 7
	expect(t, "changed", second(f.r.ProposeSeries("ws", changed, human("definer"))), "conflict")
	expect(t, "second series", second(f.r.ProposeSeries("ws", Series{ID: "s2", Kind: "inv", Prefix: "X-", Width: 3, First: 1}, human("definer"))), ReasonSeriesExists)
	expect(t, "prefix", second(f.r.ProposeSeries("ws", Series{ID: "s3", Kind: "int", Prefix: "INV-", Width: 3, First: 1}, human("definer"))), ReasonPrefixInUse)
	expect(t, "approve form", second(f.r.ApproveSeries("ws", "", human("approver"))), "invalid")
	expect(t, "approve assistant", second(f.r.ApproveSeries("ws", "s", assistant("bot"))), ReasonAINotAllowed)
	expect(t, "approve unknown", second(f.r.ApproveSeries("ws", "nope", human("approver"))), "not_found")
	expect(t, "approve workspace", second(f.r.ApproveSeries("other", "s", human("approver"))), "not_found")
	expect(t, "approve self", second(f.r.ApproveSeries("ws", "s", human("definer"))), ReasonSelfApproval)
	must(f.r.ApproveSeries("ws", "s", human("approver")))
	if must(f.r.ApproveSeries("ws", "s", human("approver"))).State != StateActive {
		t.Fatal("repeat approval")
	}
	expect(t, "second approver", second(f.r.ApproveSeries("ws", "s", human("other"))), "conflict")
	expect(t, "numbers form", second(f.r.Numbers("", "s")), "invalid")
	expect(t, "numbers unknown", second(f.r.Numbers("ws", "nope")), "not_found")
	expect(t, "numbers workspace", second(f.r.Numbers("other", "s")), "not_found")
	if format(Series{Prefix: "A-", Width: 3}, 7) != "A-007" || format(Series{Prefix: "A-", Width: 3}, 999) != "A-999" || ceiling(3) != 999 || ceiling(1) != 9 {
		t.Fatal("format")
	}
}

func TestPrefixesCannotCollide(t *testing.T) {
	a := format(Series{Prefix: "A", Width: 4}, 1000)
	if a != "A1000" {
		t.Fatal(a)
	}
	if seriesForm(Series{ID: "s", Kind: "k", Prefix: "A1", Width: 3, First: 1}) {
		t.Fatal("a digit-ending prefix could reproduce A1000")
	}
	for _, prefix := range []string{"A", "A-", "Z.", "a:", "B_"} {
		if !seriesForm(Series{ID: "s", Kind: "k", Prefix: prefix, Width: 3, First: 1}) {
			t.Fatal(prefix)
		}
	}
	if !seriesForm(Series{ID: "s", Kind: "k", Prefix: "A", Width: maxWidth, First: ceiling(maxWidth)}) || seriesForm(Series{ID: "s", Kind: "k", Prefix: "A", Width: maxWidth, First: ceiling(maxWidth) + 1}) {
		t.Fatal("bounds")
	}
}

func TestReservationsAndCancellationsAreNeverReused(t *testing.T) {
	f := standard(t)
	for _, id := range []string{"d1", "d2", "d3"} {
		must(f.r.Submit("ws", draftOf(id, "inv"), human("clerk")))
	}
	expect(t, "form", second(f.r.Reserve("ws", "", human("clerk"))), "invalid")
	expect(t, "assistant", second(f.r.Reserve("ws", "d1", assistant("bot"))), ReasonAINotAllowed)
	expect(t, "system", second(f.r.Reserve("ws", "d1", system("job"))), ReasonNotHuman)
	expect(t, "unknown", second(f.r.Reserve("ws", "nope", human("clerk"))), "not_found")
	first := must(f.r.Reserve("ws", "d1", human("clerk")))
	if first.Number != "INV-000001" || first.State != NumberReserved || must(f.r.DraftOf("ws", "d1")).Number != "INV-000001" {
		t.Fatalf("reserve: %+v", first)
	}
	if must(f.r.Reserve("ws", "d1", human("clerk"))).Number != first.Number {
		t.Fatal("repeat")
	}
	expect(t, "other reserver", second(f.r.Reserve("ws", "d1", human("other"))), ReasonAlreadyReserved)
	expect(t, "cancel form", second(f.r.Cancel("ws", "d1", "bored", human("clerk"))), "invalid")
	expect(t, "cancel assistant", second(f.r.Cancel("ws", "d1", CancelOther, assistant("bot"))), ReasonAINotAllowed)
	expect(t, "cancel unknown", second(f.r.Cancel("ws", "nope", CancelOther, human("clerk"))), "not_found")
	expect(t, "cancel nothing", second(f.r.Cancel("ws", "d2", CancelOther, human("clerk"))), ReasonNotReserved)
	cancelled := must(f.r.Cancel("ws", "d1", CancelOperator, human("lead")))
	if cancelled.State != NumberCancelled || cancelled.Reason != CancelOperator || cancelled.Actor != "lead" || must(f.r.DraftOf("ws", "d1")).Number != "" {
		t.Fatalf("cancel: %+v", cancelled)
	}
	if must(f.r.Cancel("ws", "d1", CancelOperator, human("lead"))).Number != cancelled.Number {
		t.Fatal("repeat cancel")
	}
	expect(t, "other canceller", second(f.r.Cancel("ws", "d1", CancelOperator, human("x"))), ReasonNotReserved)
	expect(t, "other reason", second(f.r.Cancel("ws", "d1", CancelOther, human("lead"))), ReasonNotReserved)
	again := must(f.r.Reserve("ws", "d1", human("clerk")))
	second2 := must(f.r.Reserve("ws", "d2", human("clerk")))
	if again.Number != "INV-000002" || second2.Number != "INV-000003" {
		t.Fatalf("reuse: %s %s", again.Number, second2.Number)
	}
	revised := draftOf("d2", "inv", Line{Text: "Revised", Amount: "7"})
	if must(f.r.Submit("ws", revised, human("clerk"))).Number != "INV-000003" {
		t.Fatal("a revision dropped the reservation")
	}
	f.ready(t, "ws", revised)
	if must(f.r.Issue("ws", "d2", human("issuer"))).Snapshot.Number != "INV-000003" {
		t.Fatal("issue did not consume the reservation")
	}
	expect(t, "reserve issued", second(f.r.Reserve("ws", "d2", human("clerk"))), ReasonAlreadyIssued)
	expect(t, "reserve issued other", second(f.r.Reserve("ws", "d2", human("x"))), ReasonAlreadyIssued)
	expect(t, "cancel issued", second(f.r.Cancel("ws", "d2", CancelOther, human("clerk"))), ReasonAlreadyIssued)
	f.ready(t, "ws", draftOf("d3", "inv"))
	if must(f.r.Issue("ws", "d3", system("job"))).Snapshot.Number != "INV-000004" {
		t.Fatal("issue without a reservation")
	}
	numbers := must(f.r.Numbers("ws", "s-inv"))
	want := []string{NumberCancelled, NumberReserved, NumberIssued, NumberIssued}
	for index, n := range numbers {
		if n.State != want[index] || n.Value != int64(index+1) {
			t.Fatalf("numbers: %+v", numbers)
		}
	}
	seq := draftOf("q1", "seq")
	seq.Language = "ar"
	must(f.r.Submit("ws", seq, human("clerk")))
	expect(t, "sequential", second(f.r.Reserve("ws", "q1", human("clerk"))), ReasonReservationsOff)
	u := draftOf("u1", "inv")
	f.issued(t, u)
	d := s(f).drafts["u1"]
	d.reserved = nil
	expect(t, "issued draft without a reservation", second(f.r.Reserve("ws", "u1", human("clerk"))), ReasonAlreadyIssued)
}

func s(f fixture) *space { return f.r.workspaces["ws"] }

func TestSeriesStatesGateNumbers(t *testing.T) {
	f := open(t, testLimits())
	for _, k := range kinds[:1] {
		must(f.r.Define("ws", k, human("definer")))
		must(f.r.ApproveKind("ws", k.ID, human("approver")))
		must(f.r.VerifyTax("ws", k.ID, "ev", human("taxer")))
	}
	must(f.r.Submit("ws", draftOf("d1", "inv"), human("clerk")))
	expect(t, "no series", second(f.r.Reserve("ws", "d1", human("clerk"))), ReasonNoSeries)
	must(f.r.ProposeSeries("ws", Series{ID: "s", Kind: "inv", Prefix: "I-", Width: 1, First: 8}, human("definer")))
	expect(t, "pending series", second(f.r.Reserve("ws", "d1", human("clerk"))), ReasonSeriesPending)
	must(f.r.ApproveSeries("ws", "s", human("approver")))
	must(f.r.Submit("ws", draftOf("d2", "inv"), human("clerk")))
	must(f.r.Submit("ws", draftOf("d3", "inv"), human("clerk")))
	if must(f.r.Reserve("ws", "d1", human("clerk"))).Number != "I-8" || must(f.r.Reserve("ws", "d2", human("clerk"))).Number != "I-9" {
		t.Fatal("width")
	}
	expect(t, "exhausted", second(f.r.Reserve("ws", "d3", human("clerk"))), "limit")
}

func TestNumberLimitCountsEveryValue(t *testing.T) {
	l := testLimits()
	l.Numbers = 2
	f := open(t, l)
	f.setup(t, "ws")
	for _, id := range []string{"d1", "d2"} {
		f.ready(t, "ws", draftOf(id, "inv"))
	}
	must(f.r.Reserve("ws", "d1", human("clerk")))
	must(f.r.Cancel("ws", "d1", CancelOther, human("clerk")))
	must(f.r.Reserve("ws", "d1", human("clerk")))
	expect(t, "numbers", second(f.r.Reserve("ws", "d2", human("clerk"))), "limit")
	expect(t, "issue", second(f.r.Issue("ws", "d2", human("issuer"))), "limit")
	if len(s(f).documents) != 0 {
		t.Fatal("a refused issue left a document")
	}
}

func TestDraftForms(t *testing.T) {
	f := standard(t)
	forms := []func(*Draft){
		func(d *Draft) { d.ID = "" }, func(d *Draft) { d.Kind = "" }, func(d *Draft) { d.Customer = strings.Repeat("c", maxSubject+1) },
		func(d *Draft) { d.Entity = "" }, func(d *Draft) { d.Contract = "" }, func(d *Draft) { d.Currency = "KR" }, func(d *Draft) { d.Currency = "krw" },
		func(d *Draft) { d.Currency = "K1W" }, func(d *Draft) { d.Scale = -1 }, func(d *Draft) { d.Scale = maxScale + 1 }, func(d *Draft) { d.Language = "fr" },
		func(d *Draft) { d.Lines = nil }, func(d *Draft) { d.Lines = make([]Line, maxLines+1) }, func(d *Draft) { d.Original = "bad id" },
		func(d *Draft) { d.Total = "110.0" }, func(d *Draft) { d.Total = "0" }, func(d *Draft) { d.Lines[0].Amount = "-1" },
		func(d *Draft) { d.Lines[0].Text = "" }, func(d *Draft) { d.Lines[0].Text = "tab\there" }, func(d *Draft) { d.Lines[0].Text = "\xff" },
		func(d *Draft) { d.Lines[0].Text = "a\u202eb" }, func(d *Draft) { d.Lines[0].Text = "a\u202ab" },
		func(d *Draft) { d.Lines[0].Text = "a\u2066b" }, func(d *Draft) { d.Lines[0].Text = "a\u2069b" },
		func(d *Draft) { d.Lines[0].Text = strings.Repeat("x", maxTextBytes+1) },
		func(d *Draft) { d.Detail = &Detail{Artifact: "a", Digest: digestA, Rows: 1, Total: "1.001"} },
		func(d *Draft) { d.Detail = &Detail{Artifact: "", Digest: digestA, Rows: 1, Total: "1"} },
		func(d *Draft) {
			d.Detail = &Detail{Artifact: "a", Digest: strings.ToUpper(digestA), Rows: 1, Total: "1"}
		},
		func(d *Draft) { d.Detail = &Detail{Artifact: "a", Digest: digestA[:63], Rows: 1, Total: "1"} },
		func(d *Draft) { d.Detail = &Detail{Artifact: "a", Digest: digestA[:63] + "g", Rows: 1, Total: "1"} },
		func(d *Draft) { d.Detail = &Detail{Artifact: "a", Digest: digestA, Rows: 0, Total: "1"} },
		func(d *Draft) {
			d.Detail = &Detail{Artifact: "a", Digest: digestA, Rows: maxDetailRows + 1, Total: "1"}
		},
	}
	for index, change := range forms {
		d := draftOf("d1", "inv")
		change(&d)
		if err := second(f.r.Submit("ws", d, human("clerk"))); reason(err) != "invalid" {
			t.Errorf("form %d: %v", index, err)
		}
	}
	must(f.r.Submit("ws", draftOf("m1", "inv", Line{Text: "a\u200fb\u2029c\u2065d\u206ae\u2029", Amount: "1"}), human("clerk")))
	long := draftOf("d1", "inv", Line{Text: strings.Repeat("한", maxTextBytes/3), Amount: "1"})
	must(f.r.Submit("ws", long, human("clerk")))
	edge := draftOf("d2", "inv")
	edge.Detail = &Detail{Artifact: "a", Digest: digestA, Rows: maxDetailRows, Total: "100"}
	f.details.facts["ws|a"] = DetailFacts{Digest: digestA, Rows: maxDetailRows, Total: "100", Currency: "KRW", Scale: 2}
	must(f.r.Submit("ws", edge, human("clerk")))
	lines := make([]Line, maxLines)
	for index := range lines {
		lines[index] = Line{Text: "row", Amount: "1"}
	}
	must(f.r.Submit("ws", draftOf("d3", "inv", lines...), human("clerk")))
	expect(t, "workspace", second(f.r.Submit("", draftOf("d1", "inv"), human("clerk"))), "invalid")
	expect(t, "actor", second(f.r.Submit("ws", draftOf("d1", "inv"), Actor{ID: "", Kind: KindHuman})), "invalid")
	expect(t, "assistant", second(f.r.Submit("ws", draftOf("d9", "inv"), assistant("bot"))), ReasonAINotAllowed)
	expect(t, "system", second(f.r.Submit("ws", draftOf("d9", "inv"), system("job"))), ReasonNotHuman)
	expect(t, "unknown kind", second(f.r.Submit("ws", draftOf("d9", "nope"), human("clerk"))), "not_found")
	expect(t, "unknown workspace", second(f.r.Submit("other", draftOf("d9", "inv"), human("clerk"))), "not_found")
	credit := draftOf("d9", "crn")
	expect(t, "credit without original", second(f.r.Submit("ws", credit, human("clerk"))), "invalid")
	plain := draftOf("d9", "inv")
	plain.Original = "d1"
	expect(t, "original on an invoice", second(f.r.Submit("ws", plain, human("clerk"))), "invalid")
}

func TestAmountsAreCanonical(t *testing.T) {
	for _, text := range []string{"", ".", "1.", ".5", "01", "00", "1.50", "+1", "1e3", "1,0", "0", "0.0", "1.123", "1 ", strings.Repeat("9", maxDigits+1), "1.2.3", "１"} {
		if _, ok := amount(text, 2); ok {
			t.Errorf("%q accepted", text)
		}
	}
	for text, want := range map[string]string{"1": "100", "0.01": "1", "10.5": "1050", strings.Repeat("9", maxDigits-2) + ".99": strings.Repeat("9", maxDigits)} {
		value, ok := amount(text, 2)
		if !ok || value.String() != want {
			t.Errorf("%q: %v %v", text, value, ok)
		}
	}
	if _, ok := amount("1.5", 0); ok {
		t.Error("scale 0")
	}
	if v, ok := amount("7", 0); !ok || v.String() != "7" {
		t.Error("scale 0 whole")
	}
}

func TestDraftRulesAndRevisions(t *testing.T) {
	f := standard(t)
	d := draftOf("d1", "inv")
	d.Total = "111"
	expect(t, "total", second(f.r.Submit("ws", d, human("clerk"))), ReasonTotalMismatch)
	d = draftOf("d1", "inv")
	d.Language = "ar"
	must(f.r.Submit("ws", d, human("clerk")))
	c := draftOf("c1", "crn")
	c.Language, c.Original = "ar", "d0"
	expect(t, "credit language", second(f.r.Submit("ws", c, human("clerk"))), ReasonLanguage)
	pending := kinds[0]
	pending.ID = "pend"
	must(f.r.Define("ws", pending, human("definer")))
	expect(t, "pending kind", second(f.r.Submit("ws", draftOf("p1", "pend"), human("clerk"))), ReasonKindPending)
	d.Detail = &Detail{Artifact: "art", Digest: digestA, Rows: 3, Total: "99"}
	expect(t, "detail total", second(f.r.Submit("ws", d, human("clerk"))), ReasonDetailMismatch)
	d.Detail.Total = "100"
	expect(t, "detail unknown", second(f.r.Submit("ws", d, human("clerk"))), ReasonDetailUnverified)
	right := DetailFacts{Digest: digestA, Rows: 3, Total: "100", Currency: "KRW", Scale: 2}
	for _, change := range []func(*DetailFacts){
		func(x *DetailFacts) { x.Digest = strings.Repeat("b", 64) }, func(x *DetailFacts) { x.Rows = 4 }, func(x *DetailFacts) { x.Total = "100.5" },
		func(x *DetailFacts) { x.Currency = "USD" }, func(x *DetailFacts) { x.Scale = 3 },
	} {
		facts := right
		change(&facts)
		f.details.facts["ws|art"] = facts
		expect(t, "detail facts", second(f.r.Submit("ws", d, human("clerk"))), ReasonDetailUnverified)
	}
	f.details.facts["other|art"] = right
	expect(t, "detail of another workspace", second(f.r.Submit("ws", d, human("clerk"))), ReasonDetailUnverified)
	f.details.facts["ws|art"] = right
	v2 := must(f.r.Submit("ws", d, human("clerk")))
	if v2.Revision != 2 || v2.Approver != "" {
		t.Fatalf("revision: %+v", v2)
	}
	v2.Draft.Lines[0].Text = "mutated"
	v2.Draft.Detail.Rows = 9
	got := must(f.r.DraftOf("ws", "d1"))
	if got.Draft.Lines[0].Text == "mutated" || got.Draft.Detail.Rows != 3 {
		t.Fatal("a view shares the draft")
	}
	if must(f.r.Submit("ws", d, human("clerk"))).Revision != 2 {
		t.Fatal("identical repeat")
	}
	if must(f.r.Submit("ws", d, human("other"))).Revision != 3 {
		t.Fatal("another submitter makes a revision")
	}
	moved := d
	moved.Kind = "int"
	moved.Language = "en"
	expect(t, "kind change", second(f.r.Submit("ws", moved, human("clerk"))), "conflict")
	l := testLimits()
	l.Drafts = 1
	g := open(t, l)
	g.setup(t, "ws")
	must(g.r.Submit("ws", draftOf("a", "inv"), human("clerk")))
	expect(t, "drafts", second(g.r.Submit("ws", draftOf("b", "inv"), human("clerk"))), "limit")
	expect(t, "draftof form", second(f.r.DraftOf("ws", "")), "invalid")
	expect(t, "draftof unknown", second(f.r.DraftOf("ws", "nope")), "not_found")
	expect(t, "draftof workspace", second(f.r.DraftOf("x", "d1")), "not_found")
	if sameDraft(draftOf("a", "inv"), draftOf("b", "inv")) || !sameDraft(draftOf("a", "inv"), draftOf("a", "inv")) {
		t.Fatal("sameDraft")
	}
}

func TestApprovalBindsTheDigestAndTheReferences(t *testing.T) {
	f := standard(t)
	v := must(f.r.Submit("ws", draftOf("d1", "inv"), human("clerk")))
	expect(t, "form", second(f.r.Approve("ws", "d1", "short", human("lead"))), "invalid")
	expect(t, "assistant", second(f.r.Approve("ws", "d1", v.Digest, assistant("bot"))), ReasonAINotAllowed)
	expect(t, "system", second(f.r.Approve("ws", "d1", v.Digest, system("job"))), ReasonNotHuman)
	expect(t, "unknown", second(f.r.Approve("ws", "nope", v.Digest, human("lead"))), "not_found")
	expect(t, "digest", second(f.r.Approve("ws", "d1", digestA, human("lead"))), ReasonDigestChanged)
	expect(t, "self", second(f.r.Approve("ws", "d1", v.Digest, human("clerk"))), ReasonSelfApproval)
	expect(t, "not approved", second(f.r.Issue("ws", "d1", human("issuer"))), ReasonNotApproved)
	f.fields.clear("entity:en1", protected.FieldPaymentAccount)
	expect(t, "unresolved", second(f.r.Approve("ws", "d1", v.Digest, human("lead"))), ReasonAccountUnresolved)
	f.fields.set("entity:en1", protected.FieldPaymentAccount, "acct-1", "account-1", start.Add(-time.Hour))
	a := must(f.r.Approve("ws", "d1", v.Digest, human("lead")))
	if a.Approver != "lead" || len(a.References) != 5 || a.Coverage == nil || a.Coverage.Binding != "b1" {
		t.Fatalf("approve: %+v", a)
	}
	a.References[0].Change = "mutated"
	a.Coverage.Binding = "mutated"
	got := must(f.r.DraftOf("ws", "d1"))
	if got.References[0].Change != "acct-1" || got.Coverage.Binding != "b1" {
		t.Fatal("a view shares the approval")
	}
	expect(t, "second approver", second(f.r.Approve("ws", "d1", v.Digest, human("other"))), "conflict")
	at := must(f.r.Approve("ws", "d1", v.Digest, human("lead"))).At
	f.clock.advance(time.Minute)
	if must(f.r.Approve("ws", "d1", v.Digest, human("lead"))).At != at {
		t.Fatal("an unchanged repeat re-recorded")
	}
	f.fields.set("entity:en1", protected.FieldPaymentAccount, "acct-2", "account-2", f.clock.read())
	expect(t, "reference changed", second(f.r.Issue("ws", "d1", human("issuer"))), ReasonReferenceChanged)
	refreshed := must(f.r.Approve("ws", "d1", v.Digest, human("lead")))
	if refreshed.References[0].Change != "acct-2" {
		t.Fatal("refresh")
	}
	f.terms.put(receivables.Coverage{Contract: "k1", Binding: "b1", Term: "net30", Revision: 1, NetDays: 31, Mode: receivables.ModeCalendar, Zone: "Asia/Seoul"})
	expect(t, "coverage changed", second(f.r.Issue("ws", "d1", human("issuer"))), ReasonReferenceChanged)
	must(f.r.Approve("ws", "d1", v.Digest, human("lead")))
	doc := must(f.r.Issue("ws", "d1", human("issuer")))
	if doc.Snapshot.Due.NetDays != 31 || doc.Snapshot.References[0].Change != "acct-2" {
		t.Fatalf("issued: %+v", doc.Snapshot)
	}
	expect(t, "approve issued", second(f.r.Approve("ws", "d1", v.Digest, human("lead"))), ReasonAlreadyIssued)
	expect(t, "revise issued", second(f.r.Submit("ws", draftOf("d1", "inv", Line{Text: "x", Amount: "1"}), human("clerk"))), ReasonAlreadyIssued)
	if must(f.r.Submit("ws", draftOf("d1", "inv"), human("clerk"))).Revision != 1 {
		t.Fatal("identical repeat after issue")
	}
}

func TestSameCoverageAndReferences(t *testing.T) {
	c := receivables.Coverage{Binding: "b", Term: "t", Revision: 1, NetDays: 3, Zone: "UTC"}
	if !sameCoverage(nil, nil) || sameCoverage(&c, nil) || sameCoverage(nil, &c) || !sameCoverage(&c, &c) {
		t.Fatal("nil")
	}
	for _, change := range []func(*receivables.Coverage){
		func(x *receivables.Coverage) { x.Binding = "x" }, func(x *receivables.Coverage) { x.Term = "x" }, func(x *receivables.Coverage) { x.Revision = 2 },
		func(x *receivables.Coverage) { x.NetDays = 4 }, func(x *receivables.Coverage) { x.Zone = "x" },
	} {
		other := c
		change(&other)
		if sameCoverage(&c, &other) {
			t.Fatal("coverage field ignored")
		}
	}
	r := []Reference{{Field: "a"}}
	if sameReferences(r, nil) || sameReferences(r, []Reference{{Field: "b"}}) || !sameReferences(r, []Reference{{Field: "a"}}) {
		t.Fatal("references")
	}
}
