package corrections

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
	"github.com/myblueton-sw/hybrid_billing/modules/receivables"
)

type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *testClock) read() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *testClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

var start = time.Date(2026, 10, 1, 3, 0, 0, 0, time.UTC)

func human(id string) Actor { return Actor{ID: id, Kind: KindHuman} }

func assistant(id string) Actor { return Actor{ID: id, Kind: KindAssistant} }

func system(id string) Actor { return Actor{ID: id, Kind: KindSystem} }

func reason(err error) string {
	var refusal Refusal
	switch {
	case err == nil:
		return "ok"
	case errors.As(err, &refusal):
		return refusal.Reason
	case errors.Is(err, ErrInvalid):
		return "invalid"
	case errors.Is(err, ErrNotFound):
		return "not_found"
	case errors.Is(err, ErrConflict):
		return "conflict"
	case errors.Is(err, ErrLimit):
		return "limit"
	}
	return "other: " + err.Error()
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func second[T any](_ T, err error) error { return err }

func expect(t *testing.T, label string, err error, want string) {
	t.Helper()
	if got := reason(err); got != want {
		t.Fatalf("%s: got %s, want %s", label, got, want)
	}
}

type fields struct{}

func (fields) Resolve(workspace, subject, field string, at time.Time) (protected.Resolution, error) {
	return protected.Resolution{Change: "c-" + field, Ref: protected.Ref{ID: "r-" + field, Fingerprint: "f"}, Effective: start.Add(-time.Hour)}, nil
}

type terms struct{}

func (terms) Cover(workspace, contract string, at time.Time) (receivables.Coverage, error) {
	location, _ := time.LoadLocation("Asia/Seoul")
	issue := at.In(location).Format(time.DateOnly)
	day, _ := time.Parse(time.DateOnly, issue)
	return receivables.Coverage{Contract: contract, Binding: "b", Term: "net30", Revision: 1, NetDays: 30, Mode: receivables.ModeCalendar, Zone: "Asia/Seoul",
		Date: issue, Due: day.AddDate(0, 0, 30).Format(time.DateOnly)}, nil
}

type details struct{}

func (details) Detail(workspace, artifact string) (issuance.DetailFacts, error) {
	return issuance.DetailFacts{}, errors.New("none")
}

type renderer struct{}

func (renderer) Render(issuance.Snapshot) ([]byte, error) { return nil, errors.New("unused") }

type documents struct {
	mu         sync.Mutex
	inner      *issuance.Registry
	failDoc    string
	failCredit bool
	swap       map[string]issuance.DocumentView
}

func (d *documents) DocumentOf(workspace, id string) (issuance.DocumentView, error) {
	d.mu.Lock()
	failDoc, swap := d.failDoc, d.swap[id]
	d.mu.Unlock()
	if failDoc == id {
		return issuance.DocumentView{}, errors.New("unavailable")
	}
	if swap.Snapshot.Number != "" {
		return swap, nil
	}
	return d.inner.DocumentOf(workspace, id)
}

func (d *documents) CreditsOf(workspace, id string) ([]string, error) {
	d.mu.Lock()
	fail := d.failCredit
	d.mu.Unlock()
	if fail {
		return nil, errors.New("unavailable")
	}
	return d.inner.CreditsOf(workspace, id)
}

type fixture struct {
	r     *Registry
	is    *issuance.Registry
	docs  *documents
	clock *testClock
}

func testLimits() Limits { return Limits{Workspaces: 2, Corrections: 20, Records: 20} }

var kinds = []issuance.Kind{
	{ID: "inv", Class: issuance.ClassSelf, Region: "KR", Tax: "kr_vat", Zone: "Asia/Seoul", Languages: []string{"ko"}},
	{ID: "crn", Class: issuance.ClassSelf, Region: "KR", Tax: "kr_vat", Credit: true, Zone: "Asia/Seoul", Languages: []string{"ko"}},
	{ID: "erp", Class: issuance.ClassERP, Region: "KR", Tax: "kr_vat", Zone: "Asia/Seoul", Languages: []string{"ko"}},
	{ID: "erc", Class: issuance.ClassERP, Region: "KR", Tax: "kr_vat", Credit: true, Zone: "Asia/Seoul", Languages: []string{"ko"}},
	{ID: "ocr", Class: issuance.ClassSelf, Region: "US", Tax: "us_sales", Credit: true, Zone: "Asia/Seoul", Languages: []string{"ko"}},
}

func open(t testing.TB, limits Limits) fixture {
	t.Helper()
	clock := &testClock{now: start}
	is, err := issuance.NewRegistry(issuance.Config{Fields: fields{}, Terms: terms{}, Details: details{}, Renderer: renderer{}, Zones: time.LoadLocation},
		issuance.Limits{Workspaces: 3, Kinds: 10, Drafts: 200, Documents: 200, Numbers: 400, Attempts: 1, Downloads: 1, Lease: time.Minute}, clock.read)
	if err != nil {
		t.Fatal(err)
	}
	docs := &documents{inner: is, swap: map[string]issuance.DocumentView{}}
	r, err := NewRegistry(Config{Documents: docs}, limits, clock.read)
	if err != nil {
		t.Fatal(err)
	}
	f := fixture{r: r, is: is, docs: docs, clock: clock}
	for _, ws := range []string{"ws", "ws2"} {
		for _, k := range kinds {
			must(is.Define(ws, k, issuance.Actor{ID: "definer", Kind: issuance.KindHuman}))
			must(is.ApproveKind(ws, k.ID, issuance.Actor{ID: "approver", Kind: issuance.KindHuman}))
			if k.Class != issuance.ClassERP {
				must(is.ProposeSeries(ws, issuance.Series{ID: "s-" + k.ID, Kind: k.ID, Prefix: k.ID + "-", Width: 6, First: 1}, issuance.Actor{ID: "definer", Kind: issuance.KindHuman}))
				must(is.ApproveSeries(ws, "s-"+k.ID, issuance.Actor{ID: "approver", Kind: issuance.KindHuman}))
			}
		}
	}
	return f
}

func draft(id, kind, total, original string) issuance.Draft {
	return issuance.Draft{ID: id, Kind: kind, Customer: "cu", Entity: "en", Contract: "k1", Currency: "KRW", Scale: 2, Language: "ko",
		Lines: []issuance.Line{{Text: "Usage", Amount: total}}, Total: total, Original: original}
}

func (f fixture) issue(t testing.TB, workspace string, d issuance.Draft) issuance.DocumentView {
	t.Helper()
	v := must(f.is.Submit(workspace, d, issuance.Actor{ID: "clerk", Kind: issuance.KindHuman}))
	must(f.is.Approve(workspace, d.ID, v.Digest, issuance.Actor{ID: "lead", Kind: issuance.KindHuman}))
	k := must(f.is.KindOf(workspace, d.Kind))
	if k.Kind.Class == issuance.ClassERP {
		return must(f.is.RecordExternal(workspace, d.ID, issuance.External{Number: "E-" + d.ID, IssuedAt: f.clock.read()}, issuance.Actor{ID: "erp", Kind: issuance.KindSystem}))
	}
	return must(f.is.Issue(workspace, d.ID, issuance.Actor{ID: "issuer", Kind: issuance.KindHuman}))
}

func (f fixture) credit(t testing.TB, id, amount, original string) issuance.DocumentView {
	t.Helper()
	return f.issue(t, "ws", draft(id, "crn", amount, original))
}

func request(id, original, key, target string) Correction {
	return Correction{ID: id, Original: original, Key: key, Target: target, Reason: ReasonLateSource, Evidence: "ev-" + key, Period: Period{From: "2026-09-01", To: "2026-09-30"}}
}

func (f fixture) approved(t testing.TB, c Correction) View {
	t.Helper()
	v := must(f.r.Request("ws", c, human("requester")))
	return must(f.r.Approve("ws", c.ID, v.Digest, human("checker")))
}
