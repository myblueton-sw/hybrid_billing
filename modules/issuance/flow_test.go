package issuance

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/protected"
	"github.com/myblueton-sw/hybrid_billing/modules/receivables"
)

func TestIssueGates(t *testing.T) {
	f := standard(t)
	f.ready(t, "ws", draftOf("d1", "inv"))
	expect(t, "form", second(f.r.Issue("", "d1", human("issuer"))), "invalid")
	expect(t, "actor", second(f.r.Issue("ws", "d1", Actor{ID: "x"})), "invalid")
	expect(t, "assistant", second(f.r.Issue("ws", "d1", assistant("bot"))), ReasonAINotAllowed)
	expect(t, "unknown", second(f.r.Issue("ws", "nope", human("issuer"))), "not_found")
	expect(t, "workspace", second(f.r.Issue("other", "d1", human("issuer"))), "not_found")
	doc := must(f.r.Issue("ws", "d1", human("issuer")))
	if must(f.r.Issue("ws", "d1", human("issuer"))).Snapshot.Digest != doc.Snapshot.Digest {
		t.Fatal("repeat")
	}
	expect(t, "other issuer", second(f.r.Issue("ws", "d1", system("job"))), ReasonAlreadyIssued)
	doc.Snapshot.Draft.Lines[0].Text = "mutated"
	doc.Snapshot.References[0].Change = "mutated"
	again := must(f.r.DocumentOf("ws", "d1"))
	if again.Snapshot.Draft.Lines[0].Text == "mutated" || again.Snapshot.References[0].Change == "mutated" {
		t.Fatal("a view shares the snapshot")
	}
	expect(t, "documentof form", second(f.r.DocumentOf("ws", "")), "invalid")
	expect(t, "documentof unknown", second(f.r.DocumentOf("ws", "nope")), "not_found")
	expect(t, "documentof workspace", second(f.r.DocumentOf("x", "d1")), "not_found")

	unverified := kinds[0]
	unverified.ID = "raw"
	must(f.r.Define("ws", unverified, human("definer")))
	must(f.r.ApproveKind("ws", "raw", human("approver")))
	must(f.r.ProposeSeries("ws", Series{ID: "s-raw", Kind: "raw", Prefix: "RAW-", Width: 3, First: 1}, human("definer")))
	must(f.r.ApproveSeries("ws", "s-raw", human("approver")))
	f.fields.set("kind:raw", protected.FieldDocumentTemplate, "tpl-raw", "template-raw", start.Add(-time.Hour))
	f.ready(t, "ws", draftOf("r1", "raw"))
	expect(t, "tax", second(f.r.Issue("ws", "r1", human("issuer"))), ReasonTaxUnverified)
	if len(must(f.r.Numbers("ws", "s-raw"))) != 0 {
		t.Fatal("a refused issue consumed a number")
	}

	f.ready(t, "ws", draftOf("d2", "inv"))
	f.details.facts["ws|art"] = DetailFacts{Digest: digestA, Rows: 2, Total: "100", Currency: "KRW", Scale: 2}
	withDetail := draftOf("d3", "inv")
	withDetail.Detail = &Detail{Artifact: "art", Digest: digestA, Rows: 2, Total: "100"}
	f.ready(t, "ws", withDetail)
	delete(f.details.facts, "ws|art")
	expect(t, "detail at issue", second(f.r.Issue("ws", "d3", human("issuer"))), ReasonDetailUnverified)
	f.fields.fail = true
	expect(t, "fields down", second(f.r.Issue("ws", "d2", human("issuer"))), ReasonAccountUnresolved)
	f.fields.fail = false
	l := testLimits()
	l.Documents = 1
	g := open(t, l)
	g.setup(t, "ws")
	g.issued(t, draftOf("a", "inv"))
	g.ready(t, "ws", draftOf("b", "inv"))
	expect(t, "documents", second(g.r.Issue("ws", "b", human("issuer"))), "limit")
	e := draftOf("e", "erp")
	e.Language, e.Contract = "en", "k2"
	g.ready(t, "ws", e)
	expect(t, "documents external", second(g.r.RecordExternal("ws", "e", External{Number: "N1", IssuedAt: start}, system("erp"))), "limit")
}

func TestReferencesAndTermsMustResolve(t *testing.T) {
	f := standard(t)
	cases := []struct {
		subject, field, reason string
	}{
		{"customer:cu1", protected.FieldDocumentRecipient, ReasonRecipientUnresolved},
		{"entity:en1", protected.FieldTaxIdentifier, ReasonTaxIDUnresolved},
		{"customer:cu1", protected.FieldTaxIdentifier, ReasonTaxIDUnresolved},
		{"kind:inv", protected.FieldDocumentTemplate, ReasonTemplateUnresolved},
	}
	v := must(f.r.Submit("ws", draftOf("d1", "inv"), human("clerk")))
	for _, c := range cases {
		saved := f.fields.entries[c.subject+"|"+c.field]
		f.fields.clear(c.subject, c.field)
		expect(t, c.reason, second(f.r.Approve("ws", "d1", v.Digest, human("lead"))), c.reason)
		f.fields.entries[c.subject+"|"+c.field] = saved
	}
	for label, bad := range map[string]protected.Resolution{
		"change":    {Change: "", Ref: protected.Ref{ID: "r"}, Effective: start.Add(-time.Hour)},
		"ref":       {Change: "c", Ref: protected.Ref{ID: "bad ref"}, Effective: start.Add(-time.Hour)},
		"zero time": {Change: "c", Ref: protected.Ref{ID: "r"}},
		"future":    {Change: "c", Ref: protected.Ref{ID: "r"}, Effective: start.Add(time.Hour)},
	} {
		saved := f.fields.entries["entity:en1|"+protected.FieldPaymentAccount]
		f.fields.entries["entity:en1|"+protected.FieldPaymentAccount] = nil
		r := &badFields{fields: f.fields, bad: bad}
		f.r.config.Fields = r
		expect(t, label, second(f.r.Approve("ws", "d1", v.Digest, human("lead"))), ReasonAccountUnresolved)
		f.r.config.Fields = f.fields
		f.fields.entries["entity:en1|"+protected.FieldPaymentAccount] = saved
	}
	for _, c := range []struct {
		want string
		err  error
	}{
		{ReasonTermMissing, receivables.Refusal{Reason: receivables.ReasonTermMissing}},
		{ReasonTermPending, receivables.Refusal{Reason: receivables.ReasonTermPending}},
		{ReasonTermConflict, receivables.Refusal{Reason: receivables.ReasonTermConflict}},
		{ReasonTermUnavailable, receivables.Refusal{Reason: receivables.ReasonZoneMismatch}},
		{ReasonTermUnavailable, errors.New("down")},
	} {
		f.terms.err = c.err
		expect(t, c.want, second(f.r.Approve("ws", "d1", v.Digest, human("lead"))), c.want)
	}
	f.terms.err = nil
	for _, c := range []receivables.Coverage{
		{Contract: "k1", Binding: "b1", Term: "t", Revision: 1, NetDays: 30, Mode: receivables.ModeBusinessDay, Zone: "Asia/Seoul"},
		{Contract: "k9", Binding: "b1", Term: "t", Revision: 1, NetDays: 30, Mode: receivables.ModeCalendar, Zone: "Asia/Seoul"},
		{Contract: "k1", Binding: "b1", Term: "t", Revision: 1, NetDays: -1, Mode: receivables.ModeCalendar, Zone: "Asia/Seoul"},
		{Contract: "k1", Binding: "b1", Term: "t", Revision: 1, NetDays: maxNetDays + 1, Mode: receivables.ModeCalendar, Zone: "Asia/Seoul"},
	} {
		f.terms.contracts["k1"] = c
		expect(t, "coverage", second(f.r.Approve("ws", "d1", v.Digest, human("lead"))), ReasonModeUnsupported)
	}
	f.terms.contracts["k1"] = receivables.Coverage{Contract: "k1", Binding: "b1", Term: "t", Revision: 1, NetDays: 30, Mode: receivables.ModeCalendar, Zone: "Mars/Base"}
	expect(t, "coverage zone", second(f.r.Approve("ws", "d1", v.Digest, human("lead"))), ReasonZoneUnavailable)
	f.terms.contracts["k1"] = receivables.Coverage{Contract: "k1", Binding: "b1", Term: "t", Revision: 1, NetDays: 30, Mode: receivables.ModeCalendar, Zone: "Asia/Seoul"}
	for _, skew := range []int{1, 2} {
		f.terms.skew = skew
		expect(t, "coverage dates", second(f.r.Approve("ws", "d1", v.Digest, human("lead"))), ReasonCoverageMismatch)
	}
	f.terms.skew = 0
	f.terms.contracts["k1"] = receivables.Coverage{Contract: "k1", Binding: "b1", Term: "t", Revision: 1, NetDays: maxNetDays, Mode: receivables.ModeCalendar, Zone: "Asia/Seoul"}
	must(f.r.Approve("ws", "d1", v.Digest, human("lead")))
	if due := must(f.r.Issue("ws", "d1", human("issuer"))).Snapshot.Due; due.OriginalDue != "2027-10-02" {
		t.Fatal(due)
	}
}

type badFields struct {
	fields *fields
	bad    protected.Resolution
}

func (b *badFields) Resolve(workspace, subject, field string, at time.Time) (protected.Resolution, error) {
	if subject == "entity:en1" && field == protected.FieldPaymentAccount {
		return b.bad, nil
	}
	return b.fields.Resolve(workspace, subject, field, at)
}

func TestDueDatesFollowTheContractZone(t *testing.T) {
	cases := []struct {
		at              time.Time
		zone            string
		net             int
		issue, original string
	}{
		{time.Date(2026, 9, 30, 14, 59, 59, 0, time.UTC), "Asia/Seoul", 30, "2026-09-30", "2026-10-30"},
		{time.Date(2026, 9, 30, 15, 0, 0, 0, time.UTC), "Asia/Seoul", 30, "2026-10-01", "2026-10-31"},
		{time.Date(2026, 10, 1, 3, 0, 0, 0, time.UTC), "Asia/Seoul", 45, "2026-10-01", "2026-11-15"},
		{time.Date(2028, 1, 30, 12, 0, 0, 0, time.UTC), "UTC", 30, "2028-01-30", "2028-02-29"},
		{time.Date(2026, 1, 31, 12, 0, 0, 0, time.UTC), "UTC", 28, "2026-01-31", "2026-02-28"},
		{time.Date(2026, 3, 8, 6, 30, 0, 0, time.UTC), "America/New_York", 1, "2026-03-08", "2026-03-09"},
		{time.Date(2026, 11, 1, 4, 30, 0, 0, time.UTC), "America/New_York", 0, "2026-11-01", "2026-11-01"},
		{time.Date(2026, 12, 31, 23, 0, 0, 0, time.UTC), "Asia/Dubai", 1, "2027-01-01", "2027-01-02"},
	}
	for _, c := range cases {
		location, _ := time.LoadLocation(c.zone)
		due := due(Kind{}, &receivables.Coverage{Binding: "b", Term: "t", Revision: 1, NetDays: c.net, Mode: receivables.ModeCalendar, Zone: c.zone}, c.at, location)
		if due.IssueDate != c.issue || due.OriginalDue != c.original || due.EffectiveDue != c.original || due.Status != DueApplicable {
			t.Errorf("%v %s: %+v", c.at, c.zone, due)
		}
	}
	seoul, _ := time.LoadLocation("Asia/Seoul")
	if due := due(Kind{Zone: "Asia/Seoul"}, nil, time.Date(2026, 9, 30, 15, 0, 0, 0, time.UTC), seoul); due != (Due{Status: DueNotApplicable, Zone: "Asia/Seoul", IssueDate: "2026-10-01"}) {
		t.Fatal(due)
	}
}

func TestRecordExternalGates(t *testing.T) {
	f := standard(t)
	e := draftOf("e1", "erp")
	e.Language, e.Contract = "en", "k2"
	must(f.r.Submit("ws", e, human("clerk")))
	expect(t, "number", second(f.r.RecordExternal("ws", "e1", External{Number: "", IssuedAt: start}, system("erp"))), "invalid")
	expect(t, "instant", second(f.r.RecordExternal("ws", "e1", External{Number: "N1"}, system("erp"))), "invalid")
	expect(t, "assistant", second(f.r.RecordExternal("ws", "e1", External{Number: "N1", IssuedAt: start}, assistant("bot"))), ReasonAINotAllowed)
	expect(t, "unknown", second(f.r.RecordExternal("ws", "nope", External{Number: "N1", IssuedAt: start}, system("erp"))), "not_found")
	expect(t, "not approved", second(f.r.RecordExternal("ws", "e1", External{Number: "N1", IssuedAt: start}, system("erp"))), ReasonNotApproved)
	f.ready(t, "ws", e)
	f.ready(t, "ws", draftOf("d1", "inv"))
	expect(t, "self kind", second(f.r.RecordExternal("ws", "d1", External{Number: "N1", IssuedAt: start}, system("erp"))), ReasonNotERP)
	f.fields.set("customer:cu1", protected.FieldDocumentRecipient, "rcpt-new", "recipient-new", start.Add(-time.Minute))
	f.clock.advance(time.Minute)
	expect(t, "changed at the erp instant", second(f.r.RecordExternal("ws", "e1", External{Number: "N1", IssuedAt: start}, system("erp"))), ReasonReferenceChanged)
	f.ready(t, "ws", e)
	doc := must(f.r.RecordExternal("ws", "e1", External{Number: "N1", IssuedAt: start.Add(time.Minute)}, system("erp")))
	for _, x := range []External{{Number: "N2", IssuedAt: start.Add(time.Minute)}, {Number: "N1", IssuedAt: start}} {
		expect(t, "different repeat", second(f.r.RecordExternal("ws", "e1", x, system("erp"))), ReasonAlreadyIssued)
	}
	expect(t, "other recorder", second(f.r.RecordExternal("ws", "e1", External{Number: "N1", IssuedAt: start.Add(time.Minute)}, human("clerk"))), ReasonAlreadyIssued)
	if doc.Snapshot.IssuedAt != start.Add(time.Minute) || doc.Snapshot.Issuer != "erp" {
		t.Fatal(doc.Snapshot)
	}
	f.issued(t, draftOf("d9", "inv"))
	expect(t, "issue then record", second(f.r.RecordExternal("ws", "d9", External{Number: "Z", IssuedAt: start}, system("job"))), ReasonAlreadyIssued)
	expect(t, "record then issue", second(f.r.Issue("ws", "e1", system("erp"))), ReasonAlreadyIssued)
}

func TestRenderGatesAndLeases(t *testing.T) {
	f := standard(t)
	f.issued(t, draftOf("d1", "inv"))
	expect(t, "form", second(f.r.Render("ws", "", system("job"))), "invalid")
	expect(t, "assistant", second(f.r.Render("ws", "d1", assistant("bot"))), ReasonAINotAllowed)
	expect(t, "unknown", second(f.r.Render("ws", "nope", system("job"))), "not_found")
	expect(t, "workspace", second(f.r.Render("other", "d1", system("job"))), "not_found")
	expect(t, "download form", second(f.r.Download("ws", "", system("job"))), "invalid")
	expect(t, "download assistant", second(f.r.Download("ws", "d1", assistant("bot"))), ReasonAINotAllowed)
	expect(t, "download unknown", second(f.r.Download("x", "d1", system("job"))), "not_found")
	expect(t, "preview unknown", second(f.r.Preview("ws", "nope", system("job"))), "not_found")
	release := make(chan struct{})
	entered := make(chan struct{}, 4)
	f.renderer.use(func(v Snapshot) ([]byte, error) {
		entered <- struct{}{}
		<-release
		return good(v)
	})
	var wg sync.WaitGroup
	var slow error
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, slow = f.r.Render("ws", "d1", system("job"))
	}()
	<-entered
	if v := must(f.r.DocumentOf("ws", "d1")); v.File != FileRendering {
		t.Fatal(v.File)
	}
	expect(t, "in progress", second(f.r.Render("ws", "d1", system("job"))), ReasonRenderInProgress)
	expect(t, "download while rendering", second(f.r.Download("ws", "d1", system("job"))), ReasonNotRendered)
	f.clock.advance(testLimits().Lease)
	f.renderer.use(good)
	fresh := must(f.r.Render("ws", "d1", system("job2")))
	if fresh.Attempts[0].Outcome != ReasonAttemptSuperseded || fresh.Attempts[1].Outcome != OutcomeAccepted {
		t.Fatalf("lease: %+v", fresh.Attempts)
	}
	close(release)
	wg.Wait()
	expect(t, "stale completion", slow, ReasonAttemptSuperseded)
	if v := must(f.r.DocumentOf("ws", "d1")); v.Digest != fresh.Digest || len(v.Attempts) != 2 {
		t.Fatalf("stale completion changed the document: %+v", v)
	}
	s(f).documents["d1"].file[0] ^= 1
	expect(t, "damaged file", second(f.r.Download("ws", "d1", human("clerk"))), ReasonDigestMismatch)
	s(f).documents["d1"].file[0] ^= 1
	must(f.r.Download("ws", "d1", human("clerk")))
}

func TestRendererReceivesACopy(t *testing.T) {
	f := standard(t)
	d := draftOf("d1", "inv")
	f.details.facts["ws|art"] = DetailFacts{Digest: digestA, Rows: 2, Total: "100", Currency: "KRW", Scale: 2}
	d.Detail = &Detail{Artifact: "art", Digest: digestA, Rows: 2, Total: "100"}
	f.issued(t, d)
	f.renderer.use(func(v Snapshot) ([]byte, error) {
		v.Draft.Lines[0].Text = "mutated"
		v.Draft.Detail.Rows = 99
		v.References[0].Change = "mutated"
		return good(v)
	})
	must(f.r.Render("ws", "d1", system("job")))
	v := must(f.r.DocumentOf("ws", "d1")).Snapshot
	if v.Draft.Lines[0].Text == "mutated" || v.Draft.Detail.Rows != 2 || v.References[0].Change == "mutated" {
		t.Fatal("the renderer changed the snapshot")
	}
	f.renderer.use(func(Snapshot) ([]byte, error) { return catalog("/URI (http://x)"), nil })
	f.issued(t, draftOf("d2", "inv"))
	expect(t, "unsafe", second(f.r.Render("ws", "d2", system("job"))), ReasonUnsafeOutput)
	f.renderer.use(func(Snapshot) ([]byte, error) { return []byte("not a pdf"), nil })
	expect(t, "malformed", second(f.r.Render("ws", "d2", system("job"))), ReasonMalformedOutput)
	if v := must(f.r.DocumentOf("ws", "d2")); v.Attempts[0].Outcome != ReasonUnsafeOutput || v.Attempts[1].Outcome != ReasonMalformedOutput || v.Attempts[0].Digest != "" || v.File != FileNone {
		t.Fatalf("attempts: %+v", v.Attempts)
	}
}

func TestWorkspacesAreIsolated(t *testing.T) {
	f := standard(t)
	f.setup(t, "ws2")
	a := f.issued(t, draftOf("d1", "inv"))
	f.ready(t, "ws2", draftOf("d1", "inv"))
	b := must(f.r.Issue("ws2", "d1", human("issuer")))
	if a.Snapshot.Number != b.Snapshot.Number || a.Snapshot.Digest == b.Snapshot.Digest || a.Snapshot.DraftDigest == b.Snapshot.DraftDigest {
		t.Fatal("workspace digests")
	}
	c := draftOf("c1", "crn", Line{Text: "x", Amount: "1"})
	c.Original = "d1"
	f.issued(t, c)
	if len(s(f).documents["d1"].credits) != 1 || len(f.r.workspaces["ws2"].documents["d1"].credits) != 0 {
		t.Fatal("credits crossed workspaces")
	}
}

func TestReviewedEdgeCases(t *testing.T) {
	f := standard(t)
	failing := false
	f.r.config.Zones = func(name string) (*time.Location, error) {
		if failing {
			return nil, errors.New("tzdata unavailable")
		}
		return time.LoadLocation(name)
	}
	internal := draftOf("i1", "int")
	internal.Language = "en"
	f.ready(t, "ws", internal)
	failing = true
	expect(t, "zone at issue", second(f.r.Issue("ws", "i1", human("issuer"))), ReasonZoneUnavailable)
	if len(must(f.r.Numbers("ws", "s-int"))) != 0 || len(s(f).documents) != 0 {
		t.Fatal("a zone failure left a number or a document")
	}
	failing = false
	f.issued(t, draftOf("d1", "inv"))
	expect(t, "issuer kind", second(f.r.Issue("ws", "d1", system("issuer"))), ReasonAlreadyIssued)
	if v := must(f.r.DocumentOf("ws", "d1")).Snapshot; v.IssuerKind != KindHuman {
		t.Fatal(v.IssuerKind)
	}
	f.clock.advance(time.Minute)
	f.ready(t, "ws", draftOf("d2", "inv"))
	must(f.r.Reserve("ws", "d2", human("reserver")))
	f.clock.advance(time.Minute)
	must(f.r.Issue("ws", "d2", system("job")))
	numbers := must(f.r.Numbers("ws", "s-inv"))
	if n := numbers[1]; n.Taker != "reserver" || n.Taken != start.Add(time.Minute) || n.Actor != "job" || n.At != start.Add(2*time.Minute) {
		t.Fatalf("reserver lost: %+v", n)
	}
	e := draftOf("e1", "erp")
	e.Language, e.Contract = "en", "k2"
	f.ready(t, "ws", e)
	must(f.r.RecordExternal("ws", "e1", External{Number: "N1", IssuedAt: f.clock.read()}, system("erp")))
	expect(t, "external issuer kind", second(f.r.RecordExternal("ws", "e1", External{Number: "N1", IssuedAt: f.clock.read()}, human("erp"))), ReasonAlreadyIssued)
	for index := 0; index < testLimits().Downloads; index++ {
		must(f.r.Render("ws", "d1", system("job")))
		must(f.r.Download("ws", "d1", human("clerk")))
	}
	expect(t, "downloads", second(f.r.Download("ws", "d1", human("clerk"))), "limit")
}

func TestLeaseExpiryAtTheAttemptLimit(t *testing.T) {
	l := testLimits()
	l.Attempts = 1
	f := open(t, l)
	f.setup(t, "ws")
	f.issued(t, draftOf("d1", "inv"))
	release := make(chan struct{})
	entered := make(chan struct{}, 1)
	f.renderer.use(func(v Snapshot) ([]byte, error) {
		entered <- struct{}{}
		<-release
		return good(v)
	})
	done := make(chan error, 1)
	go func() {
		_, err := f.r.Render("ws", "d1", system("job"))
		done <- err
	}()
	<-entered
	f.clock.advance(l.Lease)
	expect(t, "limit after the lease", second(f.r.Render("ws", "d1", system("job2"))), "limit")
	if v := must(f.r.DocumentOf("ws", "d1")); v.Attempts[0].Outcome != "" {
		t.Fatalf("the limit refusal changed the running attempt: %+v", v.Attempts)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if v := must(f.r.DocumentOf("ws", "d1")); v.File != FileRendered || v.Attempts[0].Outcome != OutcomeAccepted {
		t.Fatalf("the original attempt: %+v", v)
	}
}

func TestStoredFileIsACopyOfTheRendererBuffer(t *testing.T) {
	f := standard(t)
	f.issued(t, draftOf("d1", "inv"))
	var kept []byte
	f.renderer.use(func(v Snapshot) ([]byte, error) {
		out, _ := good(v)
		kept = out
		return out, nil
	})
	must(f.r.Render("ws", "d1", system("job")))
	kept[0] = 'X'
	if must(f.r.Download("ws", "d1", human("clerk")))[0] != '%' {
		t.Fatal("the stored file shares the renderer buffer")
	}
}

func TestCreditsOfListsEveryCredit(t *testing.T) {
	f := standard(t)
	f.issued(t, draftOf("d1", "inv"))
	if got := must(f.r.CreditsOf("ws", "d1")); got == nil || len(got) != 0 {
		t.Fatalf("no credits: %#v", got)
	}
	for _, id := range []string{"c1", "c2"} {
		c := draftOf(id, "crn", Line{Text: "x", Amount: "1"})
		c.Original = "d1"
		f.issued(t, c)
	}
	got := must(f.r.CreditsOf("ws", "d1"))
	if len(got) != 2 || got[0] != "c1" || got[1] != "c2" {
		t.Fatal(got)
	}
	got[0] = "mutated"
	if must(f.r.CreditsOf("ws", "d1"))[0] != "c1" {
		t.Fatal("a view shares the credits")
	}
	expect(t, "form", second(f.r.CreditsOf("", "d1")), "invalid")
	expect(t, "form id", second(f.r.CreditsOf("ws", "")), "invalid")
	expect(t, "unknown", second(f.r.CreditsOf("ws", "nope")), "not_found")
	expect(t, "workspace", second(f.r.CreditsOf("other", "d1")), "not_found")
}
