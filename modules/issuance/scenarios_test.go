package issuance

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/protected"
	"github.com/myblueton-sw/hybrid_billing/modules/receivables"
)

func TestIssueFixesTheSnapshotBeforeTheFile(t *testing.T) {
	f := standard(t)
	d := draftOf("d1", "inv")
	d.Lines[0].Text = "클라우드 사용료 " + strings.Repeat("가", 1000)
	d.Lines = append(d.Lines, Line{Text: "رسوم الخدمة", Amount: "0.5"})
	d.Total = sumOf(d.Lines, 2)
	doc := f.issued(t, d)
	v := doc.Snapshot
	if v.Number != "INV-000001" || doc.File != FileNone || v.External || !v.Statutory || v.TaxEvidence != "tax-review-inv" || v.TaxReviewer != "taxer" {
		t.Fatalf("snapshot: %+v", v)
	}
	if v.Due != (Due{Status: DueApplicable, Zone: "Asia/Seoul", IssueDate: "2026-10-01", Binding: "b1", Term: "net30", Revision: 1, NetDays: 30, Mode: receivables.ModeCalendar, OriginalDue: "2026-10-31", EffectiveDue: "2026-10-31"}) {
		t.Fatalf("due: %+v", v.Due)
	}
	if len(v.References) != 5 || v.References[0].Field != protected.FieldPaymentAccount || v.References[0].Subject != "entity:en1" ||
		v.References[2].Subject != "entity:en1" || v.References[3].Subject != "customer:cu1" || v.References[4].Subject != "kind:inv" {
		t.Fatalf("references: %+v", v.References)
	}
	if v.Draft.Lines[0].Text != d.Lines[0].Text || v.Draft.Lines[2].Text != "رسوم الخدمة" || v.Submitter != "clerk" || v.Approver != "lead" || v.Issuer != "issuer" || v.Revision != 1 {
		t.Fatalf("draft facts: %+v", v)
	}
	if v.Digest != encodeSnapshot(v) || v.DraftDigest != draftDigest("ws", d) {
		t.Fatal("digests")
	}
	f.fields.set("entity:en1", protected.FieldPaymentAccount, "acct-new", "account-new", f.clock.read())
	f.clock.advance(time.Hour)
	again := must(f.r.DocumentOf("ws", "d1"))
	if again.Snapshot.Digest != v.Digest || again.Snapshot.References[0].Change != "acct-en1" {
		t.Fatal("a later protected change touched the snapshot")
	}
	rendered := must(f.r.Render("ws", "d1", system("artifact-job")))
	if rendered.File != FileRendered || rendered.Digest != digestOf(must(good(v))) || len(rendered.Attempts) != 1 || rendered.Attempts[0].Outcome != OutcomeAccepted {
		t.Fatalf("rendered: %+v", rendered)
	}
	data := must(f.r.Download("ws", "d1", human("clerk")))
	again = must(f.r.DocumentOf("ws", "d1"))
	if !bytes.Equal(data, must(good(v))) || len(again.Downloads) != 1 || again.Downloads[0] != (Download{Actor: "clerk", Kind: KindHuman, At: start.Add(time.Hour)}) || again.Snapshot.Due != v.Due {
		t.Fatal("download")
	}
	again.Downloads[0].Actor = "someone-else"
	if must(f.r.DocumentOf("ws", "d1")).Downloads[0].Actor != "clerk" {
		t.Fatal("a view shares the download records")
	}
	data[0] = 'X'
	if must(f.r.Download("ws", "d1", system("portal")))[0] != '%' {
		t.Fatal("a download shares the stored file")
	}
	numbers := must(f.r.Numbers("ws", "s-inv"))
	if len(numbers) != 1 || numbers[0].State != NumberIssued || numbers[0].Draft != "d1" || numbers[0].Actor != "issuer" {
		t.Fatalf("numbers: %+v", numbers)
	}
}

func TestFileFailureRecoversWithTheSameSnapshot(t *testing.T) {
	f := standard(t)
	f.issued(t, draftOf("d1", "inv"))
	f.renderer.use(func(Snapshot) ([]byte, error) { return nil, errFake })
	expect(t, "failed render", second(f.r.Render("ws", "d1", system("job"))), ReasonRenderFailed)
	f.renderer.use(func(Snapshot) ([]byte, error) { panic("renderer crashed") })
	expect(t, "panic", second(f.r.Render("ws", "d1", system("job"))), ReasonRenderFailed)
	expect(t, "no file yet", second(f.r.Download("ws", "d1", human("clerk"))), ReasonNotRendered)
	if v := must(f.r.DocumentOf("ws", "d1")); v.File != FileNone || len(v.Attempts) != 2 {
		t.Fatalf("after failures: %+v", v)
	}
	f.renderer.use(good)
	first := must(f.r.Render("ws", "d1", system("job")))
	again := must(f.r.Render("ws", "d1", human("clerk")))
	if again.Digest != first.Digest || again.Attempts[3].Outcome != OutcomeVerified || again.Attempts[2].Outcome != OutcomeAccepted {
		t.Fatalf("recovery: %+v", again.Attempts)
	}
	seen := f.renderer.seen
	for _, v := range seen[1:] {
		if v.Digest != seen[0].Digest {
			t.Fatal("renderer saw different snapshots")
		}
	}
	f.renderer.use(func(v Snapshot) ([]byte, error) { return page("BT (changed) Tj ET"), nil })
	expect(t, "non-deterministic", second(f.r.Render("ws", "d1", system("job"))), ReasonDigestMismatch)
	if v := must(f.r.DocumentOf("ws", "d1")); v.Digest != first.Digest || v.File != FileRendered || v.Attempts[4].Outcome != ReasonDigestMismatch || v.Attempts[4].Digest == "" {
		t.Fatalf("mismatch kept the file: %+v", v)
	}
	f.renderer.use(func(Snapshot) ([]byte, error) { return nil, errFake })
	expect(t, "a failure after success", second(f.r.Render("ws", "d1", system("job"))), ReasonRenderFailed)
	if v := must(f.r.DocumentOf("ws", "d1")); v.File != FileRendered {
		t.Fatal("a failure after success dropped the file")
	}
	expect(t, "attempt limit", second(f.r.Render("ws", "d1", system("job"))), "limit")
}

func TestPartialCreditsStayWithinTheOriginal(t *testing.T) {
	f := standard(t)
	f.issued(t, draftOf("d1", "inv"))
	credit := func(id, amount string) Draft {
		d := draftOf(id, "crn", Line{Text: "Partial credit", Amount: amount})
		d.Original = "d1"
		return d
	}
	first := f.issued(t, credit("c1", "60"))
	if first.Snapshot.Due.Status != DueNotApplicable || first.Snapshot.Due.IssueDate != "2026-10-01" || first.Snapshot.Number != "CRN-000001" || len(first.Snapshot.References) != 4 {
		t.Fatalf("credit snapshot: %+v", first.Snapshot)
	}
	expect(t, "over the original", second(f.r.Submit("ws", credit("c2", "50.01"), human("clerk"))), ReasonExceedsOriginal)
	f.ready(t, "ws", credit("c2", "50"))
	f.ready(t, "ws", credit("c3", "0.01"))
	must(f.r.Issue("ws", "c2", human("issuer")))
	expect(t, "room used at issue", second(f.r.Issue("ws", "c3", human("issuer"))), ReasonExceedsOriginal)
	if n := must(f.r.Numbers("ws", "s-crn")); len(n) != 2 {
		t.Fatalf("a refused issue consumed a number: %+v", n)
	}
	other := credit("c4", "1")
	other.Customer = "cu2"
	expect(t, "other customer", second(f.r.Submit("ws", other, human("clerk"))), ReasonCreditMismatch)
	other = credit("c4", "1")
	other.Original = "nothing"
	expect(t, "unknown original", second(f.r.Submit("ws", other, human("clerk"))), ReasonOriginalNotIssued)
}

func TestERPNumbersAreRecordedNotAssigned(t *testing.T) {
	f := standard(t)
	d := draftOf("e1", "erp")
	d.Language, d.Contract, d.Entity = "en", "k2", "en2"
	f.ready(t, "ws", d)
	expect(t, "issue", second(f.r.Issue("ws", "e1", human("issuer"))), ReasonERPNumbered)
	expect(t, "reserve", second(f.r.Reserve("ws", "e1", human("clerk"))), ReasonERPNumbered)
	expect(t, "future", second(f.r.RecordExternal("ws", "e1", External{Number: "INV-000001", IssuedAt: start.Add(time.Minute)}, system("erp-sync"))), ReasonFutureInstant)
	expect(t, "before approval", second(f.r.RecordExternal("ws", "e1", External{Number: "INV-000001", IssuedAt: start.Add(-time.Minute)}, system("erp-sync"))), ReasonBeforeApproval)
	doc := must(f.r.RecordExternal("ws", "e1", External{Number: "INV-000001", IssuedAt: start}, system("erp-sync")))
	if !doc.Snapshot.External || doc.Snapshot.Number != "INV-000001" || doc.Snapshot.Due.Zone != "America/New_York" || doc.Snapshot.Due.IssueDate != "2026-09-30" || doc.Snapshot.Due.OriginalDue != "2026-11-14" {
		t.Fatalf("erp: %+v", doc.Snapshot)
	}
	if must(f.r.RecordExternal("ws", "e1", External{Number: "INV-000001", IssuedAt: start}, system("erp-sync"))).Snapshot.Digest != doc.Snapshot.Digest {
		t.Fatal("repeat")
	}
	expect(t, "render", second(f.r.Render("ws", "e1", system("job"))), ReasonNotSelfIssued)
	self := f.issued(t, draftOf("d1", "inv"))
	if self.Snapshot.Number != "INV-000001" {
		t.Fatal(self.Snapshot.Number)
	}
	if must(f.r.DocumentOf("ws", "e1")).Snapshot.External != true || must(f.r.DocumentOf("ws", "d1")).Snapshot.External {
		t.Fatal("documents are ambiguous")
	}
	d2 := d
	d2.ID = "e2"
	f.ready(t, "ws", d2)
	expect(t, "same erp number", second(f.r.RecordExternal("ws", "e2", External{Number: "INV-000001", IssuedAt: start}, system("erp-sync"))), ReasonExternalInUse)
	expect(t, "not erp", second(f.r.RecordExternal("ws", "d1", External{Number: "X1", IssuedAt: start}, system("erp-sync"))), ReasonAlreadyIssued)
}

func TestInternalSettlementsHaveTheirOwnNumbers(t *testing.T) {
	f := standard(t)
	d := draftOf("i1", "int")
	d.Language = "en"
	doc := f.issued(t, d)
	if doc.Snapshot.Number != "INT-000001" || doc.Snapshot.Statutory || doc.Snapshot.Due.Status != DueNotApplicable || len(doc.Snapshot.References) != 2 {
		t.Fatalf("internal: %+v", doc.Snapshot)
	}
	html := must(f.r.Preview("ws", "i1", human("clerk")))
	if !strings.Contains(html, "Not a statutory document.") || !strings.Contains(html, "<dd><bdi>Not applicable</bdi></dd>") {
		t.Fatal(html)
	}
}

func TestPreviewEscapesAndStaysDistinct(t *testing.T) {
	f := standard(t)
	d := draftOf("d1", "inv", Line{Text: "<script>alert(1)</script> =HYPERLINK(\"http://x\")", Amount: "5"})
	d.Detail = &Detail{Artifact: "art-1", Digest: digestA, Rows: 100000, Total: "5"}
	f.details.facts["ws|art-1"] = DetailFacts{Digest: digestA, Rows: 100000, Total: "5", Currency: "KRW", Scale: 2}
	f.issued(t, d)
	html := must(f.r.Preview("ws", "d1", system("portal")))
	if strings.Contains(html, "<script>") || strings.Contains(html, "=\"") || !strings.Contains(html, "&lt;script&gt;") || !strings.Contains(html, "Viewing copy.") || strings.Contains(html, "Not a statutory") || !strings.Contains(html, "Full detail is attached separately.") {
		t.Fatal(html)
	}
	if strings.Count(html, "<tr>") != 2 {
		t.Fatal("summary rows")
	}
}

func TestRealRegistersFeedTheSnapshot(t *testing.T) {
	clock := &testClock{now: start}
	terms, err := receivables.NewRegistry(receivablesProfile(t), receivables.Limits{Workspaces: 1, Terms: 2, Bindings: 2, Receivables: 1, Allocations: 1, Extensions: 1, Holds: 1, NetDays: 120, Freshness: time.Hour}, clock.read, time.LoadLocation)
	if err != nil {
		t.Fatal(err)
	}
	must(terms.RecordTerm("ws", receivables.Term{ID: "net45", Revision: 1, NetDays: 45, Mode: receivables.ModeCalendar}, "clerk"))
	must(terms.ApproveTerm("ws", "net45", 1, "lead"))
	must(terms.Bind("ws", receivables.Binding{ID: "b", Contract: "k1", Term: "net45", Revision: 1, Zone: "Asia/Seoul", From: "2026-01-01", Reason: "initial"}, "clerk"))
	must(terms.ApproveBinding("ws", "b", "lead"))
	f := open(t, testLimits())
	r, err := NewRegistry(Config{Fields: f.fields, Terms: terms, Details: f.details, Renderer: f.renderer, Zones: time.LoadLocation}, testLimits(), clock.read)
	if err != nil {
		t.Fatal(err)
	}
	f.r, f.clock = r, clock
	f.setup(t, "ws")
	f.terms.contracts = nil
	doc := f.issued(t, draftOf("d1", "inv"))
	if doc.Snapshot.Due.OriginalDue != "2026-11-15" || doc.Snapshot.Due.Binding != "b" || doc.Snapshot.Due.IssueDate != "2026-10-01" {
		t.Fatalf("due: %+v", doc.Snapshot.Due)
	}
}
