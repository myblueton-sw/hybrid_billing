package issuance

import (
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/protected"
	"github.com/myblueton-sw/hybrid_billing/modules/receivables"
)

func TestIdentifierAndCodeBoundaries(t *testing.T) {
	if !validID(strings.Repeat("z", maxIDBytes)) || validID(strings.Repeat("z", maxIDBytes+1)) || validID("!a") || validID("a!") || !validID("Az09._:-") {
		t.Fatal("identifiers")
	}
	for _, code := range []string{"AED", "ZAR", "KRW"} {
		if !validCurrency(code) {
			t.Fatal(code)
		}
	}
	for _, code := range []string{"1RW", "K1W", "KR1", "@RW", "[RW", "KR[", "kRW"} {
		if validCurrency(code) {
			t.Fatal(code)
		}
	}
	if !validText(strings.Repeat("a", maxTextBytes)) || validText(strings.Repeat("a", maxTextBytes+1)) {
		t.Fatal("text")
	}
	if seriesForm(Series{ID: "s", Kind: "k", Prefix: "INV9", Width: 3, First: 1}) || seriesForm(Series{ID: "s", Kind: "k", Prefix: "INV0", Width: 3, First: 1}) || !seriesForm(Series{ID: "s", Kind: "k", Prefix: "INV:", Width: 3, First: 1}) {
		t.Fatal("prefix digits")
	}
	f := open(t, testLimits())
	for _, region := range []string{"ZA", "AZ"} {
		k := kinds[2]
		k.ID, k.Region = "r"+region, region
		must(f.r.Define("ws", k, human("definer")))
	}
	for _, region := range []string{"[A", "A[", "@A", "A@"} {
		k := kinds[2]
		k.ID, k.Region = "bad", region
		expect(t, region, second(f.r.Define("ws", k, human("definer"))), "invalid")
	}
	if !detailForm(&Detail{Artifact: "a", Digest: digestA, Rows: 1, Total: "1"}, 0) {
		t.Fatal("one detail row")
	}
}

func TestScaleBoundaries(t *testing.T) {
	f := standard(t)
	for _, scale := range []int{0, maxScale} {
		d := draftOf("s"+string(rune('0'+scale)), "inv", Line{Text: "x", Amount: "7"})
		d.Scale, d.Total = scale, "7"
		must(f.r.Submit("ws", d, human("clerk")))
	}
}

func TestDigestsArePinned(t *testing.T) {
	d := draftOf("d1", "inv")
	d.Detail = &Detail{Artifact: "a", Digest: digestA, Rows: 2, Total: "100"}
	if got := draftDigest("ws", d); got != goldenDraft {
		t.Fatalf("draft digest %s", got)
	}
	plain := draftOf("d1", "inv")
	if draftDigest("ws", plain) == draftDigest("ws", d) {
		t.Fatal("detail presence")
	}
	v := Snapshot{Workspace: "ws", Number: "INV-000001", Draft: plain, Revision: 1, DraftDigest: goldenDraft, Submitter: "clerk", Approver: "lead", Issuer: "issuer", IssuerKind: KindHuman,
		IssuedAt: start, Class: ClassSelf, Region: "KR", Tax: "kr_vat", Statutory: true, TaxEvidence: "ev", TaxReviewer: "taxer",
		References: []Reference{{Field: protected.FieldPaymentAccount, Subject: "entity:en1", Change: "c", Ref: protected.Ref{ID: "r", Fingerprint: "f"}, Effective: start}},
		Due:        Due{Status: DueApplicable, Zone: "Asia/Seoul", IssueDate: "2026-10-01", Binding: "b1", Term: "net30", Revision: 1, NetDays: 30, Mode: receivables.ModeCalendar, OriginalDue: "2026-10-31", EffectiveDue: "2026-10-31"}}
	if got := encodeSnapshot(v); got != goldenSnapshot {
		t.Fatalf("snapshot digest %s", got)
	}
}

const (
	goldenDraft    = "c9312b7afd40781bb1c7df48ea44d478c999f6ca16316f81ac4239e97fbc3636"
	goldenSnapshot = "f5362117bd0f5318d3bf0746ba5f9acb11dab145b00de16f9042056b39099a1c"
)

func TestCreditMismatches(t *testing.T) {
	f := standard(t)
	for _, k := range []Kind{
		{ID: "icr", Class: ClassInternal, Region: "KR", Tax: "none", Credit: true, Zone: "Asia/Seoul", Languages: []string{"ko"}},
		{ID: "ncr", Class: ClassSelf, Region: "KR", Tax: "kr_vat", Credit: true, Zone: "Asia/Seoul", Languages: []string{"ko"}},
	} {
		must(f.r.Define("ws", k, human("definer")))
		must(f.r.ApproveKind("ws", k.ID, human("approver")))
	}
	f.issued(t, draftOf("o", "inv"))
	base := func(kindID string) Draft {
		d := draftOf("c", kindID, Line{Text: "Credit", Amount: "1"})
		d.Original = "o"
		return d
	}
	changes := map[string]func(*Draft){
		"class":     func(d *Draft) { *d = base("icr") },
		"statutory": func(d *Draft) { *d = base("ncr") },
		"entity":    func(d *Draft) { d.Entity = "en2" },
		"currency":  func(d *Draft) { d.Currency = "USD" },
		"scale":     func(d *Draft) { d.Scale = 3 },
	}
	for label, change := range changes {
		d := base("crn")
		change(&d)
		expect(t, label, second(f.r.Submit("ws", d, human("clerk"))), ReasonCreditMismatch)
	}
	credit := base("crn")
	credit.ID = "c-original"
	f.issued(t, credit)
	nested := base("crn")
	nested.ID, nested.Original = "c-nested", "c-original"
	expect(t, "credit of a credit", second(f.r.Submit("ws", nested, human("clerk"))), ReasonCreditMismatch)
}

func TestWorkspaceFormsOnEveryWrite(t *testing.T) {
	f := standard(t)
	v := must(f.r.Submit("ws", draftOf("d1", "inv"), human("clerk")))
	expect(t, "approve", second(f.r.Approve("", "d1", v.Digest, human("lead"))), "invalid")
	expect(t, "cancel", second(f.r.Cancel("", "d1", CancelOther, human("lead"))), "invalid")
	expect(t, "cancel id", second(f.r.Cancel("ws", "", CancelOther, human("lead"))), "invalid")
	expect(t, "cancel actor", second(f.r.Cancel("ws", "d1", CancelOther, Actor{ID: "x"})), "invalid")
	expect(t, "tax", second(f.r.VerifyTax("", "inv", "ev", human("taxer"))), "invalid")
	expect(t, "tax id", second(f.r.VerifyTax("ws", "", "ev", human("taxer"))), "invalid")
	expect(t, "tax actor", second(f.r.VerifyTax("ws", "inv", "ev", Actor{ID: "x"})), "invalid")
	expect(t, "approve id", second(f.r.Approve("ws", "", v.Digest, human("lead"))), "invalid")
	expect(t, "approve actor", second(f.r.Approve("ws", "d1", v.Digest, Actor{ID: "x"})), "invalid")
}

func TestZeroNetDaysAndAttemptNumbers(t *testing.T) {
	f := standard(t)
	f.terms.put(receivables.Coverage{Contract: "k1", Binding: "b1", Term: "net0", Revision: 1, NetDays: 0, Mode: receivables.ModeCalendar, Zone: "Asia/Seoul"})
	doc := f.issued(t, draftOf("d1", "inv"))
	if doc.Snapshot.Due.OriginalDue != doc.Snapshot.Due.IssueDate {
		t.Fatal(doc.Snapshot.Due)
	}
	f.renderer.use(func(Snapshot) ([]byte, error) { return nil, errFake })
	second(f.r.Render("ws", "d1", system("job")))
	f.renderer.use(good)
	f.clock.advance(time.Second)
	v := must(f.r.Render("ws", "d1", system("job")))
	if v.Attempts[0].Number != 1 || v.Attempts[1].Number != 2 || v.Attempts[1].Started != start.Add(time.Second) || v.Attempts[1].Finished != start.Add(time.Second) {
		t.Fatalf("attempts: %+v", v.Attempts)
	}
	html := must(f.r.Preview("ws", "d1", human("clerk")))
	if !strings.Contains(html, "<dt><bdi>Number</bdi></dt><dd><bdi>INV-000001</bdi></dd>") || !strings.Contains(html, "<dt><bdi>Total</bdi></dt><dd><bdi>110</bdi></dd>") {
		t.Fatal(html)
	}
}
