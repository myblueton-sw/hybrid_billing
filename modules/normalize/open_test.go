package normalize

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestOpenRefusals(t *testing.T) {
	cases := []struct {
		name   string
		change func(*RunSpec)
		want   error
	}{
		{"workspace", func(s *RunSpec) { s.WorkspaceID = "" }, ErrInvalid},
		{"id", func(s *RunSpec) { s.ID = "r 1" }, ErrInvalid},
		{"mapping provider", func(s *RunSpec) { s.Mapping.Provider = "" }, ErrInvalid},
		{"revision zero", func(s *RunSpec) { s.Revision = 0 }, ErrInvalid},
		{"scope", func(s *RunSpec) { s.Input.Scope = "" }, ErrInvalid},
		{"period", func(s *RunSpec) { s.Input.Period = "2026/10" }, ErrInvalid},
		{"manifest", func(s *RunSpec) { s.Input.Manifest = 0 }, ErrInvalid},
		{"no chunk", func(s *RunSpec) { s.Chunks = 0 }, ErrInvalid},
		{"fewer rows than chunks", func(s *RunSpec) { s.Rows = 1 }, ErrInvalid},
		{"rows over capacity", func(s *RunSpec) { s.Rows = 7 }, ErrInvalid},
		{"rows far over capacity", func(s *RunSpec) { s.Rows = 80 }, ErrInvalid},
		{"chunks over the limit", func(s *RunSpec) { s.Chunks, s.Rows = 5, 5 }, ErrLimit},
		{"more rows than the workspace may hold", func(s *RunSpec) { s.Chunks, s.Rows = 4, 11 }, ErrLimit},
		{"expected field name without a currency", func(s *RunSpec) { s.Expected = map[string]map[string]string{"bil led": {}} }, ErrInvalid},
		{"window inverted", func(s *RunSpec) { s.Window = &Window{From: window.To, To: window.From} }, ErrInvalid},
		{"window empty", func(s *RunSpec) { s.Window = &Window{From: window.From, To: window.From} }, ErrInvalid},
		{"expected currency", func(s *RunSpec) { s.Expected = map[string]map[string]string{"billed": {"usd": "1"}} }, ErrInvalid},
		{"expected sum", func(s *RunSpec) { s.Expected = map[string]map[string]string{"billed": {"USD": "one"}} }, ErrInvalid},
		{"expected sum over the input bound", func(s *RunSpec) { s.Expected = map[string]map[string]string{"billed": {"USD": "1000000000000"}} }, ErrInvalid},
		{"expected field name", func(s *RunSpec) { s.Expected = map[string]map[string]string{"bil led": {"USD": "1"}} }, ErrInvalid},
		{"unknown mapping", func(s *RunSpec) { s.Mapping.SourceVersion = "focus-9" }, ErrNotFound},
		{"revision ahead", func(s *RunSpec) { s.Revision = 3 }, ErrNotFound},
		{"an earlier revision", func(s *RunSpec) { s.Revision = 1 }, ErrRejected},
		{"expected for an unknown field", func(s *RunSpec) { s.Expected = map[string]map[string]string{"sale": {"USD": "1"}} }, ErrRejected},
		{"expected for an unknown field without a currency", func(s *RunSpec) { s.Expected = map[string]map[string]string{"sale": {}} }, ErrRejected},
		{"expected for a quantity", func(s *RunSpec) { s.Expected = map[string]map[string]string{"qty": {"USD": "1"}} }, ErrRejected},
		{"expected for a missing field", func(s *RunSpec) { s.Expected = map[string]map[string]string{"list": {"USD": "1"}} }, ErrRejected},
	}
	for _, c := range cases {
		h := newHarness(t, func(l *Limits) { l.Chunks = 4; l.ChunkRows = 3; l.Rows = 10 })
		h.declare(fullEntries())
		h.declare(testEntries())
		s := spec("r1", 2, 6)
		s.Revision = 2
		c.change(&s)
		before := h.r.dump()
		_, err := h.r.Open(s)
		wantErr(t, err, c.want, c.name)
		h.unchanged(before, c.name)
	}
	h := newHarness(t, func(l *Limits) { l.Chunks = 4; l.ChunkRows = 3 })
	entries := fullEntries()
	entries[5], entries[6] = missing("start", NotSupported), missing("end", NotSupported)
	h.declare(entries)
	_, err := h.r.Open(spec("r1", 4, 12))
	wantErr(t, err, ErrRejected, "a window without a mapped period")
	s := spec("r1", 4, 12)
	s.Window = nil
	h.open(s)
	h.open(spec2("r2", 1, 1, nil))
	h.open(spec2("r3", 1, 3, nil))
	far := spec2("r4", 1, 1, nil)
	far.Window = nil
	one := newHarness(t, nil)
	one.declare(fullEntries())
	early := spec("r1", 1, 1)
	early.Window = &Window{From: time.Time{}, To: time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)}
	one.open(early)
	other := spec("r1", 1, 1)
	other.Window = &Window{From: time.Time{}.Add(time.Nanosecond), To: early.Window.To}
	_, err = one.r.Open(other)
	wantErr(t, err, ErrConflict, "windows far from the present are told apart")
}

func spec2(id string, chunks, rows int, w *Window) RunSpec {
	s := spec(id, chunks, rows)
	s.Window = w
	return s
}

func TestOpenRepeatsConflictsAndBinding(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.Runs = 2 })
	h.declare(testEntries())
	s := spec("r1", 2, 2)
	s.Window = &Window{From: window.From.In(time.FixedZone("kst", 9*3600)), To: window.To}
	s.Expected = map[string]map[string]string{"billed": {"USD": "3.50"}}
	h.open(s)
	s.Expected["billed"]["USD"] = "changed-by-caller"
	h.check("r1", 1, good("a", "1.5", "USD"))
	if number := h.declare(fullEntries()); number != 2 {
		t.Fatal(number)
	}
	second := spec("r2", 1, 1)
	second.Revision = 2
	h.open(second)
	h.clock.advance(time.Minute)
	before := h.r.dump()
	same := spec("r1", 2, 2)
	same.Expected = map[string]map[string]string{"billed": {"USD": "3.5"}}
	if state, err := h.r.Open(same); err != nil || state != Open {
		t.Fatalf("an identical repeat succeeds after a newer revision and at the run limit: %v %v", state, err)
	}
	changes := map[string]func(*RunSpec){
		"revision":        func(x *RunSpec) { x.Revision = 2 },
		"target":          func(x *RunSpec) { x.Mapping.Target = "t2" },
		"source version":  func(x *RunSpec) { x.Mapping.SourceVersion = "focus-1.2" },
		"provider":        func(x *RunSpec) { x.Mapping.Provider = "gcp" },
		"scope":           func(x *RunSpec) { x.Input.Scope = "payer-2" },
		"period":          func(x *RunSpec) { x.Input.Period = "2026-11" },
		"manifest":        func(x *RunSpec) { x.Input.Manifest = 2 },
		"chunks":          func(x *RunSpec) { x.Chunks = 1 },
		"rows":            func(x *RunSpec) { x.Rows = 3 },
		"no window":       func(x *RunSpec) { x.Window = nil },
		"window start":    func(x *RunSpec) { x.Window = &Window{From: window.From.Add(time.Nanosecond), To: window.To} },
		"window end":      func(x *RunSpec) { x.Window = &Window{From: window.From, To: window.To.Add(time.Hour)} },
		"no expected":     func(x *RunSpec) { x.Expected = nil },
		"expected sum":    func(x *RunSpec) { x.Expected = map[string]map[string]string{"billed": {"USD": "3.51"}} },
		"expected money":  func(x *RunSpec) { x.Expected = map[string]map[string]string{"billed": {"EUR": "3.5"}} },
		"expected more":   func(x *RunSpec) { x.Expected = map[string]map[string]string{"billed": {"USD": "3.5", "EUR": "0"}} },
		"expected empty":  func(x *RunSpec) { x.Expected = map[string]map[string]string{"billed": {}} },
		"expected field":  func(x *RunSpec) { x.Expected = map[string]map[string]string{"effective": {"USD": "3.5"}} },
		"expected fields": func(x *RunSpec) { x.Expected = map[string]map[string]string{"billed": {"USD": "3.5"}, "effective": {}} },
	}
	for name, change := range changes {
		other := spec("r1", 2, 2)
		other.Expected = map[string]map[string]string{"billed": {"USD": "3.5"}}
		change(&other)
		_, err := h.r.Open(other)
		wantErr(t, err, ErrConflict, name)
	}
	_, err := h.r.Open(spec2("r3", 1, 1, window))
	wantErr(t, err, ErrRejected, "the first revision is no longer the latest")
	third := spec("r3", 1, 1)
	third.Revision = 2
	_, err = h.r.Open(third)
	wantErr(t, err, ErrLimit, "run limit")
	ended := newHarness(t, func(l *Limits) { l.Runs = 2 })
	ended.declare(fullEntries())
	ended.open(spec("r1", 1, 1))
	ended.open(spec("r2", 1, 1))
	if err := ended.r.Abandon("ws1", "r1", Superseded); err != nil {
		t.Fatal(err)
	}
	ended.check("r2", 1, good("a", "1", "USD"))
	if _, err := ended.r.Check("ws1", "r2", 1, []Row{good("b", "1", "USD")}); err != nil {
		t.Fatal(err)
	}
	_, err = ended.r.Open(spec("r3", 1, 1))
	wantErr(t, err, ErrLimit, "runs that ended still count")
	h.unchanged(before, "open repeats")
	elsewhere := third
	elsewhere.WorkspaceID, elsewhere.ID = "ws2", "r1"
	if state, err := h.r.Open(elsewhere); err != nil || state != Open {
		t.Fatal(state, err)
	}
	h.check("r1", 2, good("b", "2", "USD"))
	view := h.close("r1")
	if verdicts(view) != "partial[mapping_partial] ready[]" || view.Spec.Revision != 1 {
		t.Fatalf("a run stays bound to the revision it was opened on: %s", verdicts(view))
	}
	if view.Spec.Expected["billed"]["USD"] != "3.5" || !view.Spec.Window.From.Equal(window.From) || view.Spec.Window.From.Location() != time.UTC {
		t.Fatalf("%+v", view.Spec)
	}
	view.Spec.Expected["billed"]["USD"] = "9"
	view.Spec.Window.From = time.Time{}
	view.Sums["billed"]["USD"] = "9"
	view.Billing.Result = "changed"
	view.Unused.Items = append(view.Unused.Items, "x")
	again := h.view("r1")
	if again.Spec.Expected["billed"]["USD"] != "3.5" || again.Spec.Window.From.IsZero() || again.Sums["billed"]["USD"] != "3.5" || again.Billing.Result != Ready {
		t.Fatal("view shares state")
	}
	if cell := h.rows("r1")[0].Cells["list"]; cell != (OutCell{State: Unmapped}) {
		t.Fatalf("%+v", cell)
	}
	if view, err := h.r.Run("ws2", "r1"); err != nil || view.Spec.Revision != 2 || view.Checked != 0 {
		t.Fatalf("%+v %v", view, err)
	}
}

func TestAbandonDiscardAndRowsGivenBack(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.Rows = 5 })
	h.declare(fullEntries())
	three := []Row{good("a", "1", "USD"), good("b", "1", "USD"), good("c", "x", "USD")}
	h.open(spec("r1", 2, 4))
	h.check("r1", 1, three...)
	h.open(spec("r2", 1, 3))
	before := h.r.dump()
	_, err := h.r.Check("ws1", "r2", 1, three)
	wantErr(t, err, ErrLimit, "six of five")
	for _, bad := range []AbandonReason{"", "wrong_input", "other"} {
		wantErr(t, h.r.Abandon("ws1", "r1", bad), ErrInvalid, "abandon reason")
	}
	for _, bad := range []DiscardReason{"", "transfer_failed", "other"} {
		wantErr(t, h.r.Discard("ws1", "r1", bad), ErrInvalid, "discard reason")
	}
	wantErr(t, h.r.Discard("ws1", "r1", WrongInput), ErrRejected, "discarding an open run")
	for _, c := range []struct{ ws, id string }{{"ws2", "r1"}, {"ws1", "r9"}} {
		wantErr(t, h.r.Abandon(c.ws, c.id, Superseded), ErrNotFound, "abandon")
		wantErr(t, h.r.Discard(c.ws, c.id, WrongInput), ErrNotFound, "discard")
		_, err := h.r.Run(c.ws, c.id)
		wantErr(t, err, ErrNotFound, "run")
	}
	for _, c := range []struct{ ws, id string }{{"", "r1"}, {"ws1", ""}} {
		wantErr(t, h.r.Abandon(c.ws, c.id, Superseded), ErrInvalid, "abandon")
		wantErr(t, h.r.Discard(c.ws, c.id, WrongInput), ErrInvalid, "discard")
		_, err := h.r.Run(c.ws, c.id)
		wantErr(t, err, ErrInvalid, "run")
	}
	h.unchanged(before, "refusals")
	h.check("r2", 1, good("x", "1", "USD"), good("y", "1", "USD"))
	if h.r.workspaces["ws1"].rows != 5 {
		t.Fatal("five of five")
	}
	h.clock.advance(time.Minute)
	for _, reason := range []AbandonReason{TransferFailed, Superseded, AbandonedBy} {
		other := newHarness(t, nil)
		other.declare(fullEntries())
		other.open(spec("r1", 2, 4))
		if err := other.r.Abandon("ws1", "r1", reason); err != nil {
			t.Fatal(reason, err)
		}
	}
	if err := h.r.Abandon("ws1", "r1", Superseded); err != nil {
		t.Fatal(err)
	}
	view := h.view("r1")
	if view.State != Abandoned || view.AbandonReason != Superseded || !view.Changed.Equal(start.Add(time.Minute)) || h.r.workspaces["ws1"].rows != 2 || counts(view) != "3=2+1+0" {
		t.Fatalf("%+v", view)
	}
	h.clock.advance(time.Minute)
	before = h.r.dump()
	if err := h.r.Abandon("ws1", "r1", Superseded); err != nil {
		t.Fatal(err)
	}
	wantErr(t, h.r.Abandon("ws1", "r1", TransferFailed), ErrConflict, "another reason")
	if state, err := h.r.Check("ws1", "r1", 1, three); err != nil || state != Abandoned {
		t.Fatal(state, err)
	}
	_, err = h.r.Check("ws1", "r1", 2, three[:1])
	wantErr(t, err, ErrRejected, "a new chunk after abandonment")
	_, _, err = h.r.Close("ws1", "r1")
	wantErr(t, err, ErrRejected, "closing an abandoned run")
	wantErr(t, h.r.Discard("ws1", "r1", WrongInput), ErrRejected, "discarding an abandoned run")
	if state, err := h.r.Open(spec("r1", 2, 4)); err != nil || state != Abandoned {
		t.Fatal(state, err)
	}
	h.unchanged(before, "after abandonment")
	if h.r.workspaces["ws1"].runs["r1"].unused != nil {
		t.Fatal("an abandoned run keeps the names it collected")
	}

	h.check("r2", 1, good("x", "1", "USD"), good("y", "1", "USD"))
	_, _, err = h.r.Close("ws1", "r2")
	wantErr(t, err, ErrRejected, "two of three rows")
	if err := h.r.Abandon("ws1", "r2", TransferFailed); err != nil {
		t.Fatal(err)
	}
	h.open(spec("r3", 1, 3))
	h.check("r3", 1, three...)
	closed := h.close("r3")
	if h.r.workspaces["ws1"].rows != 3 {
		t.Fatal("a closed run keeps its rows")
	}
	for _, reason := range []DiscardReason{DiscardSuperseded, WrongInput, DiscardedBy} {
		other := newHarness(t, nil)
		other.one(fullEntries(), nil, three...)
		if err := other.r.Discard("ws1", "run-1", reason); err != nil {
			t.Fatal(reason, err)
		}
	}
	h.clock.advance(time.Minute)
	if err := h.r.Discard("ws1", "r3", WrongInput); err != nil {
		t.Fatal(err)
	}
	view = h.view("r3")
	if view.State != Discarded || view.DiscardReason != WrongInput || h.r.workspaces["ws1"].rows != 0 || !view.Changed.Equal(start.Add(3*time.Minute)) {
		t.Fatalf("%+v", view)
	}
	if verdicts(view) != verdicts(closed) || counts(view) != counts(closed) || !reflect.DeepEqual(view.Sums, closed.Sums) || view.FindingsTotal != 1 {
		t.Fatalf("a discarded run keeps its verdicts, counts and sums: %+v", view)
	}
	before = h.r.dump()
	if err := h.r.Discard("ws1", "r3", WrongInput); err != nil {
		t.Fatal(err)
	}
	wantErr(t, h.r.Discard("ws1", "r3", DiscardedBy), ErrConflict, "another reason")
	wantErr(t, h.r.Abandon("ws1", "r3", Superseded), ErrRejected, "abandon")
	if state, reasons, err := h.r.Close("ws1", "r3"); err != nil || state != Discarded || reasons != nil {
		t.Fatal(state, reasons, err)
	}
	if state, err := h.r.Check("ws1", "r3", 1, three); err != nil || state != Discarded {
		t.Fatal(state, err)
	}
	for _, read := range []func() error{
		func() error { _, err := h.r.Rows("ws1", "r3", 0, 1); return err },
		func() error { _, err := h.r.Findings("ws1", "r3", 0, 1); return err },
		func() error { _, err := h.r.Quarantined("ws1", "r3", 0, 1); return err },
	} {
		wantErr(t, read(), ErrRejected, "reading a discarded run")
	}
	h.unchanged(before, "after discarding")
	h.open(spec("r4", 2, 5))
	h.check("r4", 1, three...)
	h.check("r4", 2, good("x", "1", "USD"), good("y", "1", "USD"))
}

func TestBoundsPagingAndOrder(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.Findings = 4; l.Unused = 2; l.Page = 3 })
	h.declare(fullEntries())
	h.open(spec("r1", 3, 9))
	h.check("r1", 3, good("g", "x", "USD", "Zeta", "1"), good("h", "1", "USD"), good("dup", "1", "USD"))
	h.check("r1", 1, good("a", "x", "USD", "Usage", "", "Alpha", "1"), good("b", "1", "USD", "Midway", null), good("dup", "1", "USD"))
	h.check("r1", 2, good("d", "1", "USD", "ServiceName", null), good("e", "1", "usd"), good("f", "1", "USD", "PeriodEnd", "x", "Alpha", "2"))
	view := h.close("r1")
	if counts(view) != "9=2+5+2" || view.FindingsTotal != 6 || !reflect.DeepEqual(view.Unused, Listed{Total: 3, Items: []string{"Alpha", "Midway"}}) {
		t.Fatalf("%s %+v", counts(view), view)
	}
	if got := h.findings("r1"); got != "1.1/a/billed:not_decimal 1.1/a/qty:empty 2.1/d/svc:required_null 2.2/e/cur:not_currency" {
		t.Fatalf("the first findings in order are kept: %s", got)
	}
	for _, c := range []struct {
		after, limit int
		want         []string
	}{{0, 3, []string{"a", "a", "d"}}, {3, 3, []string{"e"}}, {1, 2, []string{"a", "d"}}, {4, 3, nil}, {9, 1, nil}, {3, 1, []string{"e"}}} {
		page, err := h.r.Findings("ws1", "r1", c.after, c.limit)
		var keys []string
		for _, finding := range page {
			keys = append(keys, finding.Key.Key)
		}
		if err != nil || !reflect.DeepEqual(keys, c.want) {
			t.Fatalf("%+v: %v %v", c, keys, err)
		}
	}
	rows, err := h.r.Rows("ws1", "r1", 0, 3)
	if err != nil || len(rows.Rows) != 2 || rows.Rows[0].Key != ev("b") || rows.Rows[1].Key != ev("h") || rows.Rows[1].Chunk != 3 || rows.Rows[1].Position != 2 {
		t.Fatalf("%+v %v", rows, err)
	}
	if one, _ := h.r.Rows("ws1", "r1", 1, 1); len(one.Rows) != 1 || one.Rows[0].Key != ev("h") {
		t.Fatalf("%+v", one)
	}
	if one, _ := h.r.Rows("ws1", "r1", 0, 1); len(one.Rows) != 1 || one.Rows[0].Key != ev("b") {
		t.Fatalf("%+v", one)
	}
	apart, err := h.r.Quarantined("ws1", "r1", 1, 3)
	if err != nil || !reflect.DeepEqual(apart, []QuarantinedRow{{3, 3, ev("dup")}}) {
		t.Fatalf("%+v %v", apart, err)
	}
	if first, _ := h.r.Quarantined("ws1", "r1", 0, 1); !reflect.DeepEqual(first, []QuarantinedRow{{1, 3, ev("dup")}}) {
		t.Fatalf("%+v", first)
	}
	if none, err := h.r.Quarantined("ws1", "r1", 2, 3); err != nil || none != nil {
		t.Fatal(none, err)
	}
	for name, read := range map[string]func(ws, id string, after, limit int) error{
		"rows":     func(ws, id string, after, limit int) error { _, err := h.r.Rows(ws, id, after, limit); return err },
		"findings": func(ws, id string, after, limit int) error { _, err := h.r.Findings(ws, id, after, limit); return err },
		"quarantined": func(ws, id string, after, limit int) error {
			_, err := h.r.Quarantined(ws, id, after, limit)
			return err
		},
	} {
		wantErr(t, read("ws1", "r1", -1, 1), ErrInvalid, name+" after")
		wantErr(t, read("ws1", "r1", 0, 0), ErrInvalid, name+" limit zero")
		wantErr(t, read("ws1", "r1", 0, 4), ErrInvalid, name+" limit over the page")
		wantErr(t, read("", "r1", 0, 1), ErrInvalid, name+" workspace")
		wantErr(t, read("ws1", "", 0, 1), ErrInvalid, name+" id")
		wantErr(t, read("ws2", "r1", 0, 1), ErrNotFound, name+" another workspace")
		wantErr(t, read("ws1", "r9", 0, 1), ErrNotFound, name+" missing")
		if err := read("ws1", "r1", 0, 3); err != nil {
			t.Fatal(name, err)
		}
	}
	rows.Rows[0].Cells["billed"] = OutCell{}
	rows.Billing.Reasons[0] = "changed"
	page, _ := h.r.Findings("ws1", "r1", 0, 1)
	page[0].Code = "changed"
	apart[0].Key.Key = "changed"
	if again, _ := h.r.Rows("ws1", "r1", 0, 1); again.Rows[0].Cells["billed"].Text != "1" || again.Billing.Reasons[0] != RowsInError {
		t.Fatal("rows share state")
	}
	if again, _ := h.r.Findings("ws1", "r1", 0, 1); again[0].Code != NotDecimal {
		t.Fatal("findings share state")
	}
	if again, _ := h.r.Quarantined("ws1", "r1", 1, 1); again[0].Key.Key != "dup" {
		t.Fatal("quarantined rows share state")
	}
	few := newHarness(t, nil)
	view = few.one(fullEntries(), nil, good("a", "1", "USD", "Beta", "1", "Alpha", "1"))
	if !reflect.DeepEqual(view.Unused, Listed{Total: 2, Items: []string{"Alpha", "Beta"}}) {
		t.Fatalf("%+v", view.Unused)
	}
}

func TestTenantBoundaryAndTime(t *testing.T) {
	h := newHarness(t, nil)
	h.declare(fullEntries())
	h.clock.advance(time.Hour)
	h.open(spec("r1", 1, 1))
	other := spec("r1", 1, 2)
	other.WorkspaceID = "ws2"
	h.clock.advance(-30 * time.Minute)
	if state, err := h.r.Open(other); err != nil || state != Open {
		t.Fatal(state, err)
	}
	if _, err := h.r.Check("ws2", "r1", 1, []Row{good("a", "7", "EUR"), good("b", "x", "EUR")}); err != nil {
		t.Fatal(err)
	}
	h.check("r1", 1, good("a", "1", "USD"))
	view := h.close("r1")
	if counts(view) != "1=1+0+0" || view.Sums["billed"]["USD"] != "1" || !view.Opened.Equal(start.Add(time.Hour)) || !view.Changed.Equal(start.Add(time.Hour)) {
		t.Fatalf("%+v", view)
	}
	if _, _, err := h.r.Close("ws2", "r1"); err != nil {
		t.Fatal(err)
	}
	second, err := h.r.Run("ws2", "r1")
	if err != nil || second.Checked != 2 || second.Errors != 1 || second.Sums["billed"]["EUR"] != "7" || !second.Opened.Equal(start.Add(time.Hour)) {
		t.Fatalf("%+v %v", second, err)
	}
	if h.r.workspaces["ws1"].rows != 1 || h.r.workspaces["ws2"].rows != 2 {
		t.Fatal("rows are counted per workspace")
	}
}

func TestRefusalsRepeatNothingAndFindingsHoldNoValue(t *testing.T) {
	h := newHarness(t, nil)
	h.declare(fullEntries())
	secret := "customer-text"
	h.open(spec("r1", 1, 1))
	h.check("r1", 1, good("k", secret, "USD", "ServiceName", null))
	h.close("r1")
	page, _ := h.r.Findings("ws1", "r1", 0, 10)
	if len(page) != 2 || strings.Contains(fmt.Sprintf("%+v", page), secret) {
		t.Fatalf("%+v", page)
	}
	_, conflict := h.r.Open(spec("r1", 1, 2))
	_, invalid := h.r.Open(spec(secret+" x", 1, 1))
	_, missing := h.r.Check("ws1", secret, 1, []Row{good(secret, "1", "USD")})
	_, rejected := h.r.Check("ws1", "r1", 1, []Row{good(secret, "1", "USD")})
	_, unknown := h.r.DeclareMapping(MappingKey{Provider: secret, SourceVersion: secret, Target: secret}, 0, fullEntries())
	for index, pair := range []struct{ got, want error }{{conflict, ErrConflict}, {invalid, ErrInvalid}, {missing, ErrNotFound}, {rejected, ErrRejected}, {unknown, ErrNotFound}} {
		if pair.got != pair.want || strings.Contains(pair.got.Error(), "customer") {
			t.Fatalf("%d: %v", index, pair.got)
		}
	}
}

func TestSourceUsesNoFloatingPoint(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) < 5 {
		t.Fatal(files, err)
	}
	banned := regexp.MustCompile(`float32|float64|big\.Float|big\.Rat|ParseFloat|FormatFloat|math\.|%[0-9.]*[fgeFGE]\b|complex`)
	checked := 0
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		checked++
		if found := banned.FindString(strings.ReplaceAll(string(content), `"math/big"`, "")); found != "" {
			t.Fatalf("%s uses %q", file, found)
		}
	}
	if checked != 5 {
		t.Fatalf("%d source files checked", checked)
	}
	if !banned.MatchString("x := float64(1)") || !banned.MatchString(`strconv.ParseFloat(s, 64)`) || !banned.MatchString(`fmt.Sprintf("%.2f", v)`) || !banned.MatchString("new(big.Rat)") {
		t.Fatal("the pattern does not find what it bans")
	}
}
