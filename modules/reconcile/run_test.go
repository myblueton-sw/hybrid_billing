package reconcile

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

func TestPartsOfTheInternalSide(t *testing.T) {
	h := newHarness(t, nil)
	h.open(doc("d1", 1, "USD", "100", 3))
	good := bill("p1", "40.50")
	cases := []struct {
		name string
		part Part
		want error
	}{
		{"id", Part{ID: "", Amount: "1", Currency: "USD", Kind: Billed, Source: "s"}, ErrInvalid},
		{"amount", Part{ID: "p", Amount: "x", Currency: "USD", Kind: Billed, Source: "s"}, ErrInvalid},
		{"amount beyond the input bound", Part{ID: "p", Amount: "0.00001", Currency: "USD", Kind: Billed, Source: "s"}, ErrInvalid},
		{"currency form", Part{ID: "p", Amount: "1", Currency: "usd", Kind: Billed, Source: "s"}, ErrInvalid},
		{"an unknown kind", Part{ID: "p", Amount: "1", Currency: "USD", Kind: "amortized", Source: "s"}, ErrInvalid},
		{"no kind", Part{ID: "p", Amount: "1", Currency: "USD", Source: "s"}, ErrInvalid},
		{"source", Part{ID: "p", Amount: "1", Currency: "USD", Kind: Billed, Source: ""}, ErrInvalid},
		{"an effective cost", Part{ID: "p", Amount: "1", Currency: "USD", Kind: Effective, Source: "s"}, ErrRejected},
		{"a list cost", Part{ID: "p", Amount: "1", Currency: "USD", Kind: List, Source: "s"}, ErrRejected},
		{"a contracted cost", Part{ID: "p", Amount: "1", Currency: "USD", Kind: Contracted, Source: "s"}, ErrRejected},
		{"a sale amount", Part{ID: "p", Amount: "1", Currency: "USD", Kind: Sale, Source: "s"}, ErrRejected},
		{"another currency", Part{ID: "p", Amount: "1", Currency: "EUR", Kind: Billed, Source: "s"}, ErrRejected},
	}
	for _, c := range cases {
		before := h.r.dump()
		_, err := h.r.Post("ws1", "d1", c.part)
		wantErr(t, err, c.want, c.name)
		h.unchanged(before, c.name)
	}
	for _, c := range []struct {
		ws, id string
		want   error
	}{{"ws2", "d1", ErrNotFound}, {"ws1", "d9", ErrNotFound}, {"", "d1", ErrInvalid}, {"ws1", "", ErrInvalid}} {
		_, err := h.r.Post(c.ws, c.id, good)
		wantErr(t, err, c.want, "post")
	}
	h.post("d1", good)
	before := h.r.dump()
	same := good
	same.Amount = "40.5"
	h.post("d1", same)
	for name, change := range map[string]func(*Part){
		"amount":   func(p *Part) { p.Amount = "40.51" },
		"source":   func(p *Part) { p.Source = "run:other" },
		"kind":     func(p *Part) { p.Kind = Effective },
		"currency": func(p *Part) { p.Currency = "EUR" },
	} {
		other := good
		change(&other)
		_, err := h.r.Post("ws1", "d1", other)
		wantErr(t, err, ErrConflict, name)
	}
	twice := bill("p2", "1")
	twice.Source = good.Source
	_, err := h.r.Post("ws1", "d1", twice)
	wantErr(t, err, ErrConflict, "one source under two parts")
	h.unchanged(before, "repeats and conflicts")
	h.post("d1", bill("p2", "0"), bill("p3", "-0.5"))
	before = h.r.dump()
	_, err = h.r.Post("ws1", "d1", bill("p4", "1"))
	wantErr(t, err, ErrRejected, "a part beyond the declared number")
	h.unchanged(before, "a part too many")
	if view := h.view("d1"); view.Posted != 3 || view.Internal != "" {
		t.Fatalf("%+v", view)
	}
	_, _, err = h.r.Close("ws1", "d1")
	if err != nil {
		t.Fatal(err)
	}
	if got := sums(h.view("d1")); got != "internal=40 explained=0 residual=60 unexplained []" {
		t.Fatal(got)
	}
	before = h.r.dump()
	if state, err := h.r.Post("ws1", "d1", good); err != nil || state != Closed {
		t.Fatal(state, err)
	}
	fresh := bill("p9", "1")
	_, err = h.r.Post("ws1", "d1", fresh)
	wantErr(t, err, ErrRejected, "a part for a closed run")
	fresh.Source = good.Source
	_, err = h.r.Post("ws1", "d1", fresh)
	wantErr(t, err, ErrConflict, "a used source is judged before the state")
	h.unchanged(before, "after closing")
	for _, id := range []string{"i1", "p1"} {
		spec := priced(id)
		if id == "p1" {
			spec = pooled(id)
		}
		h.open(spec)
		_, err := h.r.Post("ws1", id, bill("x", "1"))
		wantErr(t, err, ErrRejected, "a part for a run without a document")
		_, err = h.r.Explain("ws1", id, explain("x", Tax, "1", "i"))
		wantErr(t, err, ErrRejected, "a line for a run without a document")
		if state, verdict, err := h.r.Close("ws1", id); err != nil || state != Closed || verdict != "" {
			t.Fatal(state, verdict, err)
		}
		_, err = h.r.Parts("ws1", id, 0, 1)
		wantErr(t, err, ErrRejected, "parts of a run without a document")
		_, err = h.r.Lines("ws1", id, 0, 1)
		wantErr(t, err, ErrRejected, "lines of a run without a document")
	}
}

func TestExplanationLines(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.Lines = 9 })
	h.open(doc("d1", 1, "USD", "100", 1))
	base := explain("l1", Tax, "10", "line-1")
	cases := []struct {
		name string
		set  func(*Line)
		want error
	}{
		{"id", func(l *Line) { l.ID = "" }, ErrInvalid},
		{"category", func(l *Line) { l.Category = "discount" }, ErrInvalid},
		{"amount", func(l *Line) { l.Amount = "ten" }, ErrInvalid},
		{"a line of zero", func(l *Line) { l.Amount = "0.0" }, ErrInvalid},
		{"currency form", func(l *Line) { l.Currency = "US" }, ErrInvalid},
		{"no evidence", func(l *Line) { l.Evidence = "" }, ErrInvalid},
		{"no item", func(l *Line) { l.Item = "" }, ErrInvalid},
		{"no item type", func(l *Line) { l.ItemType = "" }, ErrInvalid},
		{"an unknown item type", func(l *Line) { l.ItemType = "row" }, ErrInvalid},
		{"rounding with an item", func(l *Line) { l.Category, l.Amount = Rounding, "0.01" }, ErrInvalid},
		{"rounding with an item type", func(l *Line) { l.Category, l.Amount, l.Item = Rounding, "0.01", "" }, ErrInvalid},
		{"rounding without evidence", func(l *Line) { *l = explain("l1", Rounding, "0.01", ""); l.Evidence = "" }, ErrInvalid},
		{"another currency", func(l *Line) { l.Currency = "EUR" }, ErrRejected},
		{"rounding above the allowed amount", func(l *Line) { *l = explain("l1", Rounding, "0.0501", "") }, ErrRejected},
		{"negative rounding above the allowed amount", func(l *Line) { *l = explain("l1", Rounding, "-0.0501", "") }, ErrRejected},
	}
	for _, c := range cases {
		line := base
		c.set(&line)
		before := h.r.dump()
		_, err := h.r.Explain("ws1", "d1", line)
		wantErr(t, err, c.want, c.name)
		h.unchanged(before, c.name)
	}
	for _, c := range []struct {
		ws, id string
		want   error
	}{{"ws2", "d1", ErrNotFound}, {"ws1", "d9", ErrNotFound}, {"", "d1", ErrInvalid}, {"ws1", "", ErrInvalid}} {
		_, err := h.r.Explain(c.ws, c.id, base)
		wantErr(t, err, c.want, "explain")
	}
	for index, category := range []Category{Support, CommitmentPurchase, Marketplace} {
		other := newHarness(t, nil)
		other.open(doc("d1", 1, "USD", "100", 1))
		other.explain("d1", Line{ID: "l1", Category: category, Amount: "1", Currency: "USD", ItemType: InternalItem, Item: fmt.Sprintf("internal-%d", index), Evidence: "memo:1"}, explain("l2", Rounding, "0.01", ""))
	}
	internal := Line{ID: "l5", Category: Credit, Amount: "-5", Currency: "USD", ItemType: InternalItem, Item: "line-1", Evidence: "memo:5"}
	h.explain("d1", base, explain("l2", Support, "3.25", "line-2"), explain("l3", CommitmentPurchase, "20", "line-3"), explain("l4", Marketplace, "0.0001", "line-4"), internal, explain("l6", Rounding, "-0.05", ""))
	before := h.r.dump()
	same := base
	same.Amount = "10.00"
	h.explain("d1", same)
	for name, change := range map[string]func(*Line){
		"amount": func(l *Line) { l.Amount = "11" }, "category": func(l *Line) { l.Category = Support }, "item": func(l *Line) { l.Item = "line-9" },
		"item type": func(l *Line) { l.ItemType = InternalItem }, "evidence": func(l *Line) { l.Evidence = "doc:other" }, "currency": func(l *Line) { l.Currency = "EUR" },
	} {
		other := base
		change(&other)
		_, err := h.r.Explain("ws1", "d1", other)
		wantErr(t, err, ErrConflict, name)
	}
	for name, line := range map[string]Line{
		"an item linked twice in one category":     explain("l7", Tax, "1", "line-1"),
		"an item linked twice in two categories":   explain("l7", Credit, "1", "line-2"),
		"an internal item linked twice":            {ID: "l7", Category: Tax, Amount: "1", Currency: "USD", ItemType: InternalItem, Item: "line-1", Evidence: "e"},
		"a second rounding line":                   explain("l7", Rounding, "0.01", ""),
		"a second rounding line of the other sign": explain("l7", Rounding, "0.05", ""),
	} {
		_, err := h.r.Explain("ws1", "d1", line)
		wantErr(t, err, ErrRejected, name)
	}
	h.unchanged(before, "repeats, conflicts and linked items")
	h.explain("d1", Line{ID: "l7", Category: Tax, Amount: "1", Currency: "USD", ItemType: InternalItem, Item: "line-2", Evidence: "e"}, explain("l8", Credit, "-1", "line-8"), explain("l9", Tax, "2", "line-9"))
	_, err := h.r.Explain("ws1", "d1", explain("l10", Tax, "2", "line-10"))
	wantErr(t, err, ErrLimit, "line limit")
	h.post("d1", bill("p1", "69.7999"))
	if verdict := h.close("d1"); verdict != Explained {
		t.Fatal(sums(h.view("d1")))
	}
	want := "internal=69.7999 explained=30.2001 residual=0 explained [tax:3:13 support:1:3.25 credit:2:-6 commitment_purchase:1:20 marketplace:1:0.0001 rounding:1:-0.05]"
	if got := sums(h.view("d1")); got != want {
		t.Fatalf("\n got %s\nwant %s", got, want)
	}
	if view := h.view("d1"); view.Lines != 9 || view.Posted != 1 {
		t.Fatalf("%+v", view)
	}
	before = h.r.dump()
	if state, err := h.r.Explain("ws1", "d1", base); err != nil || state != Closed {
		t.Fatal(state, err)
	}
	_, err = h.r.Explain("ws1", "d1", explain("l11", Tax, "2", "line-11"))
	wantErr(t, err, ErrRejected, "a line for a closed run")
	h.unchanged(before, "after closing")

	edge := newHarness(t, nil)
	edge.open(doc("d1", 1, "USD", "0", 1))
	edge.explain("d1", explain("l1", Rounding, "0.05", ""))
	below := doc("d0", 1, "USD", "0", 1)
	below.Document = "inv-0"
	edge.open(below)
	edge.explain("d0", explain("l1", Rounding, "-0.0499", ""))
	won := doc("d2", 1, "KRW", "0", 1)
	won.Rounding = "0.5"
	edge.open(won)
	half := explain("l1", Rounding, "-0.5", "")
	half.Currency = "KRW"
	edge.explain("d2", half)
	strict := doc("d3", 1, "EUR", "0", 1)
	strict.Rounding = "0"
	edge.open(strict)
	tiny := explain("l1", Rounding, "0.0001", "")
	tiny.Currency = "EUR"
	_, err = edge.r.Explain("ws1", "d3", tiny)
	wantErr(t, err, ErrRejected, "no rounding where none is allowed")
}

func TestItemsAreLinkedOnceAmongRunsThatCount(t *testing.T) {
	h := newHarness(t, nil)
	credit := Line{ID: "l1", Category: Credit, Amount: "-5", Currency: "USD", ItemType: InternalItem, Item: "credit-77", Evidence: "memo:1"}
	other := doc("d2", 1, "USD", "50", 1)
	other.Document = "inv-200"
	h.open(doc("d1", 1, "USD", "100", 1))
	h.open(other)
	h.explain("d1", credit, explain("l2", Tax, "1", "line-7"))
	_, err := h.r.Explain("ws1", "d2", credit)
	wantErr(t, err, ErrRejected, "one credit explaining two documents")
	h.explain("d2", explain("l2", Tax, "1", "line-7"))
	won := doc("d3", 1, "KRW", "100", 1)
	h.open(won)
	krw := explain("l2", Tax, "1", "line-7")
	krw.Currency = "KRW"
	_, err = h.r.Explain("ws1", "d3", krw)
	wantErr(t, err, ErrRejected, "one document line named in two currencies of a revision")
	far := doc("d1", 1, "USD", "100", 1)
	far.WorkspaceID = "ws2"
	if _, err := h.r.Open(far); err != nil {
		t.Fatal(err)
	}
	if _, err := h.r.Explain("ws2", "d1", credit); err != nil {
		t.Fatalf("items are unique inside a workspace: %v", err)
	}
	if err := h.r.Void("ws1", "d1", WrongFigures); err != nil {
		t.Fatal(err)
	}
	h.explain("d2", credit)
	krw.ID = "l3"
	h.explain("d3", krw)
	before := h.r.dump()
	if state, err := h.r.Explain("ws1", "d1", credit); err != nil || state != Voided {
		t.Fatalf("an identical line of a voided run is a repeat although its item was taken since: %v %v", state, err)
	}
	h.unchanged(before, "a repeat on a voided run")
	second := doc("d4", 2, "USD", "100", 1)
	second.Document = "inv-200"
	h.open(second)
	h.explain("d4", credit, explain("l2", Tax, "1", "line-7"))
	if !h.view("d2").Superseded || h.view("d4").Superseded {
		t.Fatal("supersession")
	}
	if err := h.r.Void("ws1", "d2", VoidSuperseded); err != nil {
		t.Fatal(err)
	}
	third := doc("d5", 1, "USD", "9", 1)
	third.Document = "inv-300"
	h.open(third)
	_, err = h.r.Explain("ws1", "d5", credit)
	wantErr(t, err, ErrRejected, "voiding a superseded run does not free an item that the later revision took")
}

func TestResiduals(t *testing.T) {
	cases := []struct {
		name  string
		total string
		parts []string
		lines []string
		want  string
	}{
		{"nothing to explain", "100", []string{"60", "40"}, nil, "internal=100 explained=0 residual=0 explained []"},
		{"cancelling signs", "0", []string{"70.5", "-70.5"}, nil, "internal=0 explained=0 residual=0 explained []"},
		{"a credit note", "-25", []string{"-30"}, []string{"5"}, "internal=-30 explained=5 residual=0 explained [tax:1:5]"},
		{"the smallest unit short", "100", []string{"99.9999"}, nil, "internal=99.9999 explained=0 residual=0.0001 unexplained []"},
		{"the smallest unit over", "100", []string{"100.0001"}, nil, "internal=100.0001 explained=0 residual=-0.0001 unexplained []"},
		{"mixed scales", "10.5", []string{"3", "0.25", "7.2", "0.0499"}, []string{"0.0001"}, "internal=10.4999 explained=0.0001 residual=0 explained [tax:1:0.0001]"},
		{"lines that cancel", "100", []string{"100"}, []string{"7", "-7"}, "internal=100 explained=0 residual=0 explained [tax:2:0]"},
		{"an explanation too large", "100", []string{"90"}, []string{"11"}, "internal=90 explained=11 residual=-1 unexplained [tax:1:11]"},
		{"a zero part", "5", []string{"0", "5"}, nil, "internal=5 explained=0 residual=0 explained []"},
	}
	for _, c := range cases {
		h := newHarness(t, nil)
		h.open(doc("d1", 1, "USD", c.total, len(c.parts)))
		for index, amount := range c.parts {
			h.post("d1", bill(fmt.Sprintf("p%d", index), amount))
		}
		for index, amount := range c.lines {
			h.explain("d1", explain(fmt.Sprintf("l%d", index), Tax, amount, fmt.Sprintf("line-%d", index)))
		}
		h.clock.advance(time.Minute)
		verdict := h.close("d1")
		view := h.view("d1")
		if got := sums(view); got != c.want || verdict != view.Verdict {
			t.Fatalf("%s:\n got %s\nwant %s", c.name, got, c.want)
		}
		wantReady := "ready[] determined"
		if view.Verdict == Unexplained {
			wantReady = "not_ready[unexplained_residual] undetermined"
		}
		if got := h.ready("d1"); got != wantReady || !view.Changed.Equal(start.Add(time.Minute)) || !view.Opened.Equal(start) {
			t.Fatalf("%s: %s", c.name, got)
		}
	}
	h := newHarness(t, nil)
	h.open(doc("d1", 1, "USD", "10", 3))
	h.post("d1", bill("p1", "5"), bill("p2", "5"))
	before := h.r.dump()
	_, _, err := h.r.Close("ws1", "d1")
	wantErr(t, err, ErrRejected, "closing with a part missing")
	h.unchanged(before, "a refused close")
	if got := h.ready("d1"); got != "not_ready[not_closed] undetermined" {
		t.Fatal(got)
	}
	for _, c := range []struct {
		ws, id string
		want   error
	}{{"ws2", "d1", ErrNotFound}, {"ws1", "d9", ErrNotFound}, {"", "d1", ErrInvalid}, {"ws1", "", ErrInvalid}} {
		_, _, err := h.r.Close(c.ws, c.id)
		wantErr(t, err, c.want, "close")
		_, err = h.r.Readiness(c.ws, c.id)
		wantErr(t, err, c.want, "readiness")
		_, err = h.r.Run(c.ws, c.id)
		wantErr(t, err, c.want, "run")
	}
}

func TestSumsOutsideTheProfile(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.Lines = 300 })
	h.open(doc("d1", 1, "USD", "0", 202))
	for index := 0; index < 101; index++ {
		h.post("d1", bill(fmt.Sprintf("a%03d", index), "999999999999"), bill(fmt.Sprintf("b%03d", index), "-999999999999"))
	}
	if verdict := h.close("d1"); verdict != Explained || h.ready("d1") != "ready[] determined" {
		t.Fatal(h.ready("d1"))
	}
	h.open(func() RunSpec { s := doc("d2", 1, "USD", "0", 101); s.Document = "inv-2"; return s }())
	for index := 0; index < 101; index++ {
		h.post("d2", bill(fmt.Sprintf("a%03d", index), "999999999999"))
	}
	for index := 0; index < 101; index++ {
		h.explain("d2", explain(fmt.Sprintf("l%03d", index), Credit, "-999999999999", fmt.Sprintf("line-%d", index)))
	}
	h.close("d2")
	view := h.view("d2")
	if got := sums(view); got != "internal=100999999999899 explained=-100999999999899 residual=0 explained [credit:101:-100999999999899]" {
		t.Fatal(got)
	}
	if got := h.ready("d2"); got != "not_ready[sum_outside_profile] undetermined" {
		t.Fatalf("an explained run whose sums the profile does not hold: %s", got)
	}
	h.open(func() RunSpec { s := doc("d4", 1, "USD", "0", 101); s.Document = "inv-4"; return s }())
	for index := 0; index < 101; index++ {
		h.post("d4", bill(fmt.Sprintf("a%03d", index), "999999999999"))
	}
	h.close("d4")
	if got := h.ready("d4"); got != "not_ready[unexplained_residual sum_outside_profile] undetermined" {
		t.Fatalf("an internal sum outside the profile: %s", got)
	}
	h.open(func() RunSpec { s := doc("d5", 2, "USD", "0", 1); s.Document = "inv-2"; return s }())
	h.open(func() RunSpec { s := doc("d6", 2, "USD", "0", 1); s.Document = "inv-4"; return s }())
	if a, b := h.ready("d2"), h.ready("d4"); a != "not_ready[superseded sum_outside_profile] undetermined" || b != "not_ready[unexplained_residual superseded sum_outside_profile] undetermined" {
		t.Fatalf("reasons in their order: %s / %s", a, b)
	}
	h.open(func() RunSpec { s := doc("d3", 1, "USD", "0", 2); s.Document = "inv-3"; return s }())
	h.post("d3", bill("a", "5"), bill("b", "-5"))
	for index := 0; index < 101; index++ {
		h.explain("d3", explain(fmt.Sprintf("t%03d", index), Tax, "999999999999", fmt.Sprintf("t-%d", index)), explain(fmt.Sprintf("c%03d", index), Credit, "-999999999999", fmt.Sprintf("c-%d", index)))
	}
	h.close("d3")
	if got := h.ready("d3"); got != "not_ready[sum_outside_profile] undetermined" || h.view("d3").Explained != "0" {
		t.Fatalf("a category sum outside the profile although the lines cancel: %s", got)
	}
}

func euro(id string, revision, parts int) RunSpec {
	spec := doc(id, revision, "EUR", "12", parts)
	spec.Rounding = "0"
	return spec
}

func TestRevisionsSupersede(t *testing.T) {
	h := newHarness(t, nil)
	h.open(doc("r1", 1, "USD", "10", 1))
	h.post("r1", bill("p1", "10"))
	h.close("r1")
	h.open(doc("k1", 1, "KRW", "500", 1))
	if got := h.ready("r1"); got != "ready[] determined" {
		t.Fatal(got)
	}
	current, err := h.r.Current("ws1", "aws", "acct-1", "inv-100")
	want := CurrentView{Revision: 1, Runs: []CurrentRun{{ID: "k1", Currency: "KRW", State: Open}, {ID: "r1", Currency: "USD", State: Closed, Verdict: Explained}}}
	if err != nil || !reflect.DeepEqual(current, want) {
		t.Fatalf("%+v %v", current, err)
	}
	h.open(euro("r3", 3, 1))
	if got := h.ready("r1"); got != "not_ready[superseded] undetermined" {
		t.Fatalf("a later revision supersedes while its own run is still open, in whatever currency: %s", got)
	}
	if got := h.ready("k1"); got != "not_ready[not_closed superseded] undetermined" {
		t.Fatal(got)
	}
	before := h.r.dump()
	krw := bill("p1", "500")
	krw.Currency = "KRW"
	_, err = h.r.Post("ws1", "k1", krw)
	wantErr(t, err, ErrRejected, "a part for a superseded open run")
	line := explain("l1", Tax, "1", "x")
	line.Currency = "KRW"
	_, err = h.r.Explain("ws1", "k1", line)
	wantErr(t, err, ErrRejected, "a line for a superseded open run")
	_, _, err = h.r.Close("ws1", "k1")
	wantErr(t, err, ErrRejected, "closing a superseded open run")
	_, err = h.r.Open(doc("r2", 2, "USD", "11", 1))
	wantErr(t, err, ErrRejected, "a revision below the highest known")
	_, err = h.r.Open(doc("r0", 1, "EUR", "11", 1))
	wantErr(t, err, ErrRejected, "the first revision again in another currency")
	h.unchanged(before, "a superseded run and lower revisions")
	if state, verdict, err := h.r.Close("ws1", "r1"); err != nil || state != Closed || verdict != Explained {
		t.Fatal(state, verdict, err)
	}
	if err := h.r.Void("ws1", "r3", WrongFigures); err != nil {
		t.Fatal(err)
	}
	if got := h.ready("r1"); got != "not_ready[superseded] undetermined" {
		t.Fatalf("voiding the later run does not bring the earlier one back: %s", got)
	}
	current, _ = h.r.Current("ws1", "aws", "acct-1", "inv-100")
	if !reflect.DeepEqual(current, CurrentView{Revision: 3, Superseded: Listed{Total: 2, Items: []string{"k1", "r1"}}}) {
		t.Fatalf("%+v", current)
	}
	_, err = h.r.Open(doc("r2", 2, "USD", "11", 1))
	wantErr(t, err, ErrRejected, "the highest revision is not lowered by a void")
	h.open(euro("r3b", 3, 2))
	h.open(doc("r3c", 3, "USD", "12", 1))
	h.post("r3c", bill("p1", "11"))
	h.close("r3c")
	if got := h.ready("r3c"); got != "not_ready[unexplained_residual] undetermined" {
		t.Fatal(got)
	}
	if got := h.ready("r1"); got != "not_ready[superseded] undetermined" {
		t.Fatal(got)
	}
	h.open(doc("r4", 4, "USD", "12", 1))
	if got := h.ready("r3c"); got != "not_ready[unexplained_residual superseded] undetermined" {
		t.Fatal(got)
	}
	current, _ = h.r.Current("ws1", "aws", "acct-1", "inv-100")
	if !reflect.DeepEqual(current, CurrentView{Revision: 4, Runs: []CurrentRun{{ID: "r4", Currency: "USD", State: Open}}, Superseded: Listed{Total: 4, Items: []string{"k1", "r1", "r3b"}}}) {
		t.Fatalf("%+v", current)
	}
	for _, c := range [][4]string{{"ws1", "aws", "acct-1", "inv-999"}, {"ws2", "aws", "acct-1", "inv-100"}, {"ws1", "gcp", "acct-1", "inv-100"}, {"ws1", "aws", "acct-2", "inv-100"}} {
		_, err := h.r.Current(c[0], c[1], c[2], c[3])
		wantErr(t, err, ErrNotFound, "current")
	}
	for _, c := range [][4]string{{"", "aws", "acct-1", "inv-100"}, {"ws1", "", "acct-1", "inv-100"}, {"ws1", "aws", "", "inv-100"}, {"ws1", "aws", "acct-1", ""}} {
		_, err := h.r.Current(c[0], c[1], c[2], c[3])
		wantErr(t, err, ErrInvalid, "current")
	}
	ordered := newHarness(t, nil)
	ordered.open(doc("z", 1, "USD", "10", 1))
	ordered.open(doc("a", 2, "USD", "10", 1))
	ordered.open(doc("m", 3, "USD", "10", 1))
	if current, _ := ordered.r.Current("ws1", "aws", "acct-1", "inv-100"); !reflect.DeepEqual(current.Superseded, Listed{Total: 2, Items: []string{"z", "a"}}) {
		t.Fatalf("superseded runs are listed by revision first: %+v", current)
	}
	failed := newHarness(t, func(l *Limits) { l.Runs = 1 })
	failed.open(doc("r1", 1, "USD", "10", 1))
	_, err = failed.r.Open(doc("r9", 9, "USD", "10", 1))
	wantErr(t, err, ErrLimit, "no room")
	if failed.view("r1").Superseded {
		t.Fatal("an open that created no run raised the highest revision")
	}
}

func TestReadinessOfTheOtherBasesAndOfVoidedRuns(t *testing.T) {
	h := newHarness(t, nil)
	h.open(priced("complete"))
	incomplete := priced("incomplete")
	incomplete.Charge, incomplete.QuantityComplete = "charge-2", false
	h.open(incomplete)
	costed := priced("costed")
	costed.Charge, costed.Cost, costed.Currency, costed.CostEvidence = "charge-3", "80", "USD", "ledger:4"
	h.open(costed)
	both := priced("both")
	both.Charge, both.QuantityComplete, both.Cost, both.Currency, both.CostEvidence = "charge-4", false, "80", "USD", "ledger:4"
	h.open(both)
	h.open(pooled("pool"))
	h.open(doc("document", 1, "USD", "10", 1))
	h.post("document", bill("p1", "10"))
	h.close("document")
	open := doc("still-open", 1, "USD", "10", 1)
	open.Document = "inv-2"
	h.open(open)
	want := map[string]string{
		"complete":   "ready[] undetermined",
		"incomplete": "not_ready[quantity_incomplete] undetermined",
		"costed":     "ready[] declared",
		"both":       "not_ready[quantity_incomplete] declared",
		"pool":       "ready[] declared",
		"document":   "ready[] determined",
		"still-open": "not_ready[not_closed] undetermined",
	}
	for id, expected := range want {
		if got := h.ready(id); got != expected {
			t.Fatalf("%s: %s, want %s", id, got, expected)
		}
	}
	for _, bad := range []VoidReason{"", "matter_closed", "other"} {
		wantErr(t, h.r.Void("ws1", "pool", bad), ErrInvalid, "void reason")
	}
	for _, c := range []struct {
		ws, id string
		want   error
	}{{"ws2", "pool", ErrNotFound}, {"ws1", "zz", ErrNotFound}, {"", "pool", ErrInvalid}, {"ws1", "", ErrInvalid}} {
		wantErr(t, h.r.Void(c.ws, c.id, VoidedBy), c.want, "void")
	}
	h.clock.advance(time.Hour)
	reasons := []VoidReason{VoidSuperseded, WrongFigures, VoidedBy}
	index := 0
	for _, id := range []string{"both", "complete", "costed", "document", "incomplete", "pool", "still-open"} {
		reason := reasons[index%3]
		index++
		if err := h.r.Void("ws1", id, reason); err != nil {
			t.Fatal(id, err)
		}
		if got := h.ready(id); got != "not_ready[voided] undetermined" {
			t.Fatalf("a voided run makes no positive claim: %s %s", id, got)
		}
		view := h.view(id)
		if view.State != Voided || view.VoidReason != reason || !view.Changed.Equal(start.Add(time.Hour)) || view.WasClosed == (id == "still-open") {
			t.Fatalf("%s: %+v", id, view)
		}
		before := h.r.dump()
		if err := h.r.Void("ws1", id, reason); err != nil {
			t.Fatal(err)
		}
		wantErr(t, h.r.Void("ws1", id, reasons[(index)%3]), ErrConflict, "voided with another reason")
		state, verdict, err := h.r.Close("ws1", id)
		switch {
		case id == "still-open":
			wantErr(t, err, ErrRejected, "closing a run voided while open")
		case err != nil || state != Voided || (id == "document") != (verdict == Explained):
			t.Fatalf("%s: %v %v %v", id, state, verdict, err)
		}
		_, err = h.r.Post("ws1", id, bill("late", "1"))
		wantErr(t, err, ErrRejected, "a part for a voided run")
		h.unchanged(before, "after voiding "+id)
	}
	if got := sums(h.view("document")); got != "internal=10 explained=0 residual=0 explained []" {
		t.Fatalf("a voided run keeps its figures as a record: %s", got)
	}
}

func TestPartsAndLinesAreReadInPages(t *testing.T) {
	h := newHarness(t, nil)
	h.open(doc("d1", 1, "USD", "66", 12))
	for index := 11; index >= 0; index-- {
		h.post("d1", bill(fmt.Sprintf("p%02d", index), fmt.Sprintf("%d.50", index)))
	}
	for _, name := range []string{"b", "c", "a"} {
		h.explain("d1", explain("l-"+name, Tax, "1.0", "line-"+name))
	}
	_, err := h.r.Parts("ws1", "d1", 0, 5)
	wantErr(t, err, ErrRejected, "parts of an open run")
	_, err = h.r.Lines("ws1", "d1", 0, 5)
	wantErr(t, err, ErrRejected, "lines of an open run")
	h.close("d1")
	var ids []string
	for {
		page, err := h.r.Parts("ws1", "d1", len(ids), 5)
		if err != nil {
			t.Fatal(err)
		}
		if len(page) == 0 {
			break
		}
		for _, part := range page {
			ids = append(ids, part.ID+"="+part.Amount)
		}
	}
	if len(ids) != 12 || ids[0] != "p00=0.5" || ids[11] != "p11=11.5" || ids[5] != "p05=5.5" {
		t.Fatal(ids)
	}
	lines, err := h.r.Lines("ws1", "d1", 1, 5)
	if err != nil || len(lines) != 2 || lines[0].ID != "l-b" || lines[1].ID != "l-c" || lines[0].Amount != "1" {
		t.Fatal(lines, err)
	}
	if first, err := h.r.Lines("ws1", "d1", 0, 2); err != nil || len(first) != 2 || first[0].ID != "l-a" || first[1].ID != "l-b" {
		t.Fatal(first, err)
	}
	if last, err := h.r.Lines("ws1", "d1", 2, 2); err != nil || len(last) != 1 || last[0].ID != "l-c" {
		t.Fatal(last, err)
	}
	for _, after := range []int{3, 9} {
		if none, err := h.r.Lines("ws1", "d1", after, 5); err != nil || none != nil {
			t.Fatal(none, err)
		}
	}
	if page, _ := h.r.Parts("ws1", "d1", 10, 1); len(page) != 1 || page[0].ID != "p10" {
		t.Fatal(page)
	}
	for _, after := range []int{12, 40} {
		if page, err := h.r.Parts("ws1", "d1", after, 5); err != nil || page != nil {
			t.Fatal(page, err)
		}
	}
	for _, c := range []struct {
		ws, id       string
		after, limit int
		want         error
	}{{"ws1", "d1", -1, 1, ErrInvalid}, {"ws1", "d1", 0, 0, ErrInvalid}, {"ws1", "d1", 0, 6, ErrInvalid}, {"", "d1", 0, 1, ErrInvalid}, {"ws1", "", 0, 1, ErrInvalid}, {"ws2", "d1", 0, 1, ErrNotFound}, {"ws1", "d9", 0, 1, ErrNotFound}} {
		_, err := h.r.Parts(c.ws, c.id, c.after, c.limit)
		wantErr(t, err, c.want, fmt.Sprint(c))
		_, err = h.r.Lines(c.ws, c.id, c.after, c.limit)
		wantErr(t, err, c.want, fmt.Sprint(c))
	}
	page, _ := h.r.Parts("ws1", "d1", 0, 1)
	page[0].Amount = "999"
	if again, _ := h.r.Parts("ws1", "d1", 0, 1); again[0].Amount != "0.5" {
		t.Fatal("parts share state")
	}
	open := doc("d2", 1, "USD", "1", 2)
	open.Document = "inv-2"
	h.open(open)
	h.post("d2", bill("only", "1"))
	if err := h.r.Void("ws1", "d2", WrongFigures); err != nil {
		t.Fatal(err)
	}
	if page, err := h.r.Parts("ws1", "d2", 0, 5); err != nil || len(page) != 1 {
		t.Fatalf("the parts of a voided run stay readable: %v %v", page, err)
	}
}

func TestRefusalsRepeatNothingAndNoFloatingPoint(t *testing.T) {
	h := newHarness(t, nil)
	h.open(doc("d1", 1, "USD", "10", 1))
	secret := "customer-text"
	spec := doc("d1", 1, "USD", "11", 1)
	spec.Input = secret
	_, conflict := h.r.Open(spec)
	_, invalid := h.r.Open(doc(secret+" x", 1, "USD", "10", 1))
	_, missing := h.r.Post("ws1", secret, bill(secret, "1"))
	odd := bill(secret, "1")
	odd.Kind = Sale
	_, rejected := h.r.Post("ws1", "d1", odd)
	for index, pair := range []struct{ got, want error }{{conflict, ErrConflict}, {invalid, ErrInvalid}, {missing, ErrNotFound}, {rejected, ErrRejected}} {
		if pair.got != pair.want {
			t.Fatalf("%d: %v", index, pair.got)
		}
	}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
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
	if checked != 4 || !banned.MatchString("x := float64(1)") || !banned.MatchString("new(big.Rat)") {
		t.Fatalf("%d source files checked", checked)
	}
}
