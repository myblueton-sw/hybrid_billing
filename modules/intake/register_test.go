package intake

import (
	"reflect"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func TestConstruction(t *testing.T) {
	clock := &testClock{at: start}
	fields := []func(*Limits){
		func(l *Limits) { l.MaxSize = 0 }, func(l *Limits) { l.MaxRatio = 0 }, func(l *Limits) { l.Chunks = 0 }, func(l *Limits) { l.ChunkRecords = 0 },
		func(l *Limits) { l.Artifacts = 0 }, func(l *Limits) { l.Records = 0 }, func(l *Limits) { l.Manifests = 0 }, func(l *Limits) { l.Page = 0 }, func(l *Limits) { l.Listed = 0 },
	}
	for index, change := range fields {
		limits := testLimits()
		change(&limits)
		if _, err := NewRegistry(limits, testProfile(t), clock.read); err != ErrInvalid {
			t.Fatalf("limit %d: %v", index, err)
		}
	}
	if _, err := NewRegistry(testLimits(), decimal.Profile{}, clock.read); err != ErrInvalid {
		t.Fatal(err)
	}
	if _, err := NewRegistry(testLimits(), testProfile(t), nil); err != ErrInvalid {
		t.Fatal(err)
	}
	one := Limits{MaxSize: 1, MaxRatio: 1, Chunks: 1, ChunkRecords: 1, Artifacts: 1, Records: 1, Manifests: 1, Page: 1, Listed: 1}
	if _, err := NewRegistry(one, testProfile(t), clock.read); err != nil {
		t.Fatal(err)
	}
	var missing *Registry
	if _, _, err := missing.Register(base("a", "d", 1)); err != ErrInvalid {
		t.Fatal(err)
	}
	unbuilt := &Registry{}
	if _, _, err := unbuilt.Register(base("a", "d", 1)); err != ErrInvalid {
		t.Fatal(err)
	}
	if _, err := unbuilt.Stage("ws1", "a", 1, []Record{rec("k", "1", "USD")}); err != ErrInvalid {
		t.Fatal(err)
	}
	if _, err := unbuilt.Staged("ws1", "a"); err != ErrInvalid {
		t.Fatal(err)
	}
	if err := unbuilt.Verify("ws1", "a", digestOf("x"), "v", "e"); err != ErrInvalid {
		t.Fatal(err)
	}
	if _, _, err := unbuilt.Seal("ws1", "a"); err != ErrInvalid {
		t.Fatal(err)
	}
	if err := unbuilt.Abandon("ws1", "a", Superseded); err != ErrInvalid {
		t.Fatal(err)
	}
	if err := unbuilt.Retire("ws1", "a", Reissued); err != ErrInvalid {
		t.Fatal(err)
	}
	if _, err := unbuilt.Artifact("ws1", "a"); err != ErrInvalid {
		t.Fatal(err)
	}
	if _, err := unbuilt.Publish(october, 0); err != ErrInvalid {
		t.Fatal(err)
	}
	if _, err := unbuilt.Status(october); err != ErrInvalid {
		t.Fatal(err)
	}
	if _, err := unbuilt.Manifest(october, 1); err != ErrInvalid {
		t.Fatal(err)
	}
	if _, err := unbuilt.Records(october, 1, 0, 1); err != ErrInvalid {
		t.Fatal(err)
	}
}

func TestRegistrationRefusals(t *testing.T) {
	two := [][]Record{{rec("k1", "1", "USD"), rec("k2", "2", "USD")}}
	cases := []struct {
		name   string
		change func(*Spec)
		want   error
	}{
		{"size zero", func(s *Spec) { s.Size = 0 }, ErrInvalid},
		{"size negative", func(s *Spec) { s.Size = -1 }, ErrInvalid},
		{"expanded below size", func(s *Spec) { s.Expanded = 99 }, ErrInvalid},
		{"chunks over limit", func(s *Spec) { s.Chunks = 201; s.Rows = 201 }, ErrLimit},
		{"rows over capacity", func(s *Spec) { s.Rows = 1001 }, ErrInvalid},
		{"rows over capacity of two chunks", func(s *Spec) { s.Chunks = 2; s.Rows = 2001 }, ErrInvalid},
		{"fewer rows than chunks", func(s *Spec) { s.Chunks = 3 }, ErrInvalid},
		{"no row", func(s *Spec) { s.Rows = 0; s.Chunks = 0 }, ErrInvalid},
		{"no chunk", func(s *Spec) { s.Chunks = 0 }, ErrInvalid},
		{"no sum", func(s *Spec) { s.Sums = nil }, ErrInvalid},
		{"more currencies than rows", func(s *Spec) { s.Sums = map[string]string{"USD": "1", "EUR": "1", "KRW": "1"} }, ErrInvalid},
		{"negative chunks", func(s *Spec) { s.Chunks = -1 }, ErrInvalid},
		{"negative rows", func(s *Spec) { s.Rows = -1 }, ErrInvalid},
		{"sum outside profile", func(s *Spec) { s.Sums = map[string]string{"USD": "1.00001"} }, ErrInvalid},
		{"sum too long", func(s *Spec) { s.Sums = map[string]string{"USD": "1234567890123"} }, ErrInvalid},
		{"sum not a number", func(s *Spec) { s.Sums = map[string]string{"USD": "x"} }, ErrInvalid},
		{"currency", func(s *Spec) { s.Sums = map[string]string{"usd": "3"} }, ErrInvalid},
		{"totals origin", func(s *Spec) { s.Totals = "" }, ErrInvalid},
		{"totals origin unknown", func(s *Spec) { s.Totals = "guessed" }, ErrInvalid},
		{"id", func(s *Spec) { s.ID = "a b" }, ErrInvalid},
		{"document", func(s *Spec) { s.Document = "" }, ErrInvalid},
		{"workspace", func(s *Spec) { s.Subject.WorkspaceID = "" }, ErrInvalid},
		{"provider", func(s *Spec) { s.Subject.Provider = "" }, ErrInvalid},
		{"scope", func(s *Spec) { s.Subject.Scope = "" }, ErrInvalid},
		{"period", func(s *Spec) { s.Subject.Period = "2026/10" }, ErrInvalid},
		{"kind", func(s *Spec) { s.Kind = "row" }, ErrInvalid},
		{"revision", func(s *Spec) { s.Revision = 0 }, ErrInvalid},
		{"bytes", func(s *Spec) { s.Bytes = "ABCDEF0123" }, ErrInvalid},
		{"bytes short", func(s *Spec) { s.Bytes = "abcdef0" }, ErrInvalid},
		{"location", func(s *Spec) { s.Location = "" }, ErrInvalid},
		{"collection", func(s *Spec) { s.Collection = "a/b" }, ErrInvalid},
		{"role", func(s *Spec) { s.Role = "copy" }, ErrInvalid},
		{"holds", func(s *Spec) { s.Holds = "" }, ErrInvalid},
		{"original names an original", func(s *Spec) { s.Original = "x" }, ErrInvalid},
		{"original names a method", func(s *Spec) { s.Method = "m" }, ErrInvalid},
		{"scan", func(s *Spec) { s.Scan = "" }, ErrInvalid},
		{"syntax", func(s *Spec) { s.Syntax = "ok" }, ErrInvalid},
		{"bodies", func(s *Spec) { s.Bodies = "" }, ErrInvalid},
	}
	for _, c := range cases {
		h := newHarness(t, nil)
		s := original(t, "a1", "d1", 1, two...)
		c.change(&s)
		before := h.r.dump()
		_, _, err := h.r.Register(s)
		wantErr(t, err, c.want, c.name)
		h.unchanged(before, c.name)
	}
	without := []struct {
		name   string
		change func(*Spec)
	}{
		{"chunk", func(s *Spec) { s.Chunks = 1 }},
		{"row", func(s *Spec) { s.Rows = 1 }},
		{"sum", func(s *Spec) { s.Sums = map[string]string{"USD": "0"} }},
		{"totals", func(s *Spec) { s.Totals = SourceStated }},
	}
	for _, holds := range []Holds{Empty, AwaitingExtraction} {
		for _, c := range without {
			h := newHarness(t, nil)
			s := base("a1", "d1", 1)
			s.Holds = holds
			c.change(&s)
			before := h.r.dump()
			_, _, err := h.r.Register(s)
			wantErr(t, err, ErrInvalid, string(holds)+" "+c.name)
			h.unchanged(before, c.name)
		}
	}
	for _, holds := range []Holds{"", "something"} {
		h := newHarness(t, nil)
		s := base("a1", "d1", 1)
		s.Holds = holds
		before := h.r.dump()
		_, _, err := h.r.Register(s)
		wantErr(t, err, ErrInvalid, "an original that does not say what it holds")
		h.unchanged(before, "holds")
	}
	h := newHarness(t, nil)
	h.register(scan("o1", "d1", 1))
	for name, change := range map[string]func(*Spec){
		"holds":    func(s *Spec) { s.Holds = Rows },
		"original": func(s *Spec) { s.Original = "" },
		"method":   func(s *Spec) { s.Method = "" },
	} {
		s := derived(t, "x1", "d1", 1, "o1", two...)
		change(&s)
		before := h.r.dump()
		_, _, err := h.r.Register(s)
		wantErr(t, err, ErrInvalid, "derived "+name)
		h.unchanged(before, name)
	}
}

func TestRegistrationBoundaries(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.ChunkRecords = 3; l.Chunks = 4 })
	full := original(t, "a1", "d1", 1, []Record{rec("k1", "1", "USD")})
	full.Chunks, full.Rows = 4, 12
	if state := h.register(full); state != Staging {
		t.Fatal(state)
	}
	over := full
	over.ID, over.Rows = "a2", 13
	_, _, err := h.r.Register(over)
	wantErr(t, err, ErrInvalid, "one row over capacity")
	single := original(t, "a3", "d1", 1, []Record{rec("k1", "1", "USD")})
	if state := h.register(single); state != Staging {
		t.Fatal(state)
	}
	both := original(t, "a4", "d1", 1, []Record{rec("k1", "1", "USD"), rec("k2", "1", "EUR")})
	if state := h.register(both); state != Staging {
		t.Fatal(state)
	}
	edge := base("a5", "d1", 1)
	edge.Size, edge.Expanded = 1000, 10000
	if state, reasons, _ := h.r.Register(edge); state != Staging || reasons != nil {
		t.Fatal(state, reasons)
	}
	same := base("a6", "d1", 1)
	same.Size, same.Expanded = 7, 7
	if state := h.register(same); state != Staging {
		t.Fatal(state)
	}
	written := original(t, "a7", "d1", 1, []Record{rec("k1", "1.5", "USD")})
	written.Sums = map[string]string{"USD": "1.50"}
	h.register(written)
	if view := h.view("a7"); view.Spec.Sums["USD"] != "1.5" {
		t.Fatalf("%+v", view.Spec.Sums)
	}
}

func TestIdentifierRules(t *testing.T) {
	long := ""
	for len(long) < 128 {
		long += "a"
	}
	for text, want := range map[string]bool{"a": true, "A.z_0:9-": true, long: true, long + "a": false, "": false, "a b": false, "a/b": false, "a@b": false, "a[b": false, "a`b": false, "a{b": false, "\uac00": false, "a\n": false, "a,b": false, "a;b": false} {
		if validID(text) != want {
			t.Errorf("identifier %q", text)
		}
	}
	for text, want := range map[string]bool{"0123abcd": true, "0123abc": false, long: true, long + "a": false, "0123abcg": false, "0123ABCD": false, "0123abc/": false, "0123abc:": false, "0123abc`": false} {
		if validDigest(text) != want {
			t.Errorf("digest %q", text)
		}
	}
	for text, want := range map[string]bool{"USD": true, "AZZ": true, "ZAA": true, "US": false, "USDT": false, "uSD": false, "UsD": false, "USd": false, "@SD": false, "U[D": false, "US1": false, "": false} {
		if validCurrency(text) != want {
			t.Errorf("currency %q", text)
		}
	}
}

func TestRegistrationRejections(t *testing.T) {
	cases := []struct {
		name   string
		change func(*Spec)
		want   []string
	}{
		{"too large", func(s *Spec) { s.Size, s.Expanded = 1001, 1001 }, []string{TooLarge}},
		{"ratio", func(s *Spec) { s.Size, s.Expanded = 100, 1001 }, []string{RatioExceeded}},
		{"ratio far", func(s *Spec) { s.Size, s.Expanded = 100, 1100 }, []string{RatioExceeded}},
		{"scan failed", func(s *Spec) { s.Scan = Failed }, []string{ScanFailed}},
		{"scan not run", func(s *Spec) { s.Scan = NotRun }, []string{ScanNotRun}},
		{"syntax failed", func(s *Spec) { s.Syntax = Failed }, []string{SyntaxFailed}},
		{"syntax not run", func(s *Spec) { s.Syntax = NotRun }, []string{SyntaxNotRun}},
		{"bodies found", func(s *Spec) { s.Bodies = Failed }, []string{BodiesFound}},
		{"bodies not run", func(s *Spec) { s.Bodies = NotRun }, []string{BodiesNotRun}},
		{"all", func(s *Spec) {
			s.Size, s.Expanded = 1001, 10011
			s.Scan, s.Syntax, s.Bodies = Failed, NotRun, Failed
		}, []string{TooLarge, RatioExceeded, ScanFailed, SyntaxNotRun, BodiesFound}},
	}
	for _, c := range cases {
		for _, kind := range []string{"rows", "empty"} {
			h := newHarness(t, nil)
			s := base("a1", "d1", 2)
			if kind == "rows" {
				s = original(t, "a1", "d1", 2, []Record{rec("k1", "1", "USD")})
			}
			c.change(&s)
			state, reasons, err := h.r.Register(s)
			if err != nil || state != Rejected || !reflect.DeepEqual(reasons, c.want) {
				t.Fatalf("%s: %v %v %v", c.name, state, reasons, err)
			}
			view := h.view("a1")
			if view.State != Rejected || !reflect.DeepEqual(view.Reasons, c.want) || !view.Registered.Equal(start) {
				t.Fatalf("%s: %+v", c.name, view)
			}
			before := h.r.dump()
			if _, err := h.r.Stage("ws1", "a1", 1, []Record{rec("k1", "1", "USD")}); kind == "rows" && err != ErrRejected {
				t.Fatalf("%s stage: %v", c.name, err)
			}
			_, _, err = h.r.Seal("ws1", "a1")
			wantErr(t, err, ErrRejected, c.name+" seal")
			wantErr(t, h.r.Abandon("ws1", "a1", Superseded), ErrRejected, c.name+" abandon")
			wantErr(t, h.r.Retire("ws1", "a1", Reissued), ErrRejected, c.name+" retire")
			_, err = h.r.Staged("ws1", "a1")
			wantErr(t, err, ErrRejected, c.name+" staged")
			h.clock.advance(time.Minute)
			again, reasons, err := h.r.Register(s)
			if err != nil || again != Rejected || !reflect.DeepEqual(reasons, c.want) {
				t.Fatalf("%s repeat: %v %v %v", c.name, again, reasons, err)
			}
			h.unchanged(before, c.name)
			other := base("a2", "d1", 1)
			other.Subject = november
			_, _, err = h.r.Register(other)
			wantErr(t, err, ErrConflict, c.name+" fixes the document")
			status := h.status(october)
			if !reflect.DeepEqual(status.Behind, Listed{Total: 1, Items: []string{"d1"}}) || status.BlockerTotal != 0 || status.Differs != Same {
				t.Fatalf("%s: %+v", c.name, status)
			}
		}
	}
}

func TestDocumentFixedByFirstArtifact(t *testing.T) {
	h := newHarness(t, nil)
	refused := base("a0", "d1", 1)
	refused.Size = 0
	if _, _, err := h.r.Register(refused); err != ErrInvalid {
		t.Fatal(err)
	}
	first := base("a1", "d1", 1)
	first.Subject = november
	first.Kind = LocatorKeys
	h.register(first)
	for name, change := range map[string]func(*Spec){
		"period":   func(s *Spec) { s.Subject.Period = "2026-10" },
		"provider": func(s *Spec) { s.Subject.Provider = "gcp" },
		"scope":    func(s *Spec) { s.Subject.Scope = "payer-2" },
		"kind":     func(s *Spec) { s.Kind = EventKeys },
	} {
		s := base("a2", "d1", 2)
		s.Subject = november
		s.Kind = LocatorKeys
		change(&s)
		before := h.r.dump()
		_, _, err := h.r.Register(s)
		wantErr(t, err, ErrConflict, name)
		h.unchanged(before, name)
	}
	s := base("a2", "d1", 2)
	s.Subject = november
	s.Kind = LocatorKeys
	h.register(s)
	if _, err := h.r.Status(october); err != ErrNotFound {
		t.Fatal(err)
	}
	elsewhere := base("a1", "d1", 1)
	elsewhere.Subject.WorkspaceID = "ws2"
	h.register(elsewhere)
	if view, err := h.r.Artifact("ws2", "a1"); err != nil || view.Spec.Subject.Period != "2026-10" || view.Spec.Kind != EventKeys {
		t.Fatal(view, err)
	}
	if h.view("a1").Spec.Kind != LocatorKeys {
		t.Fatal("workspaces met")
	}
}

func TestRegistrationRepeatsAndConflicts(t *testing.T) {
	h := newHarness(t, nil)
	chunk := []Record{rec("k1", "1.5", "USD"), rec("k2", "2", "EUR")}
	s := original(t, "a1", "d1", 3, chunk)
	s.Collection = "col:1"
	h.register(s)
	h.clock.advance(time.Hour)
	before := h.r.dump()
	equal := s
	equal.Sums = map[string]string{"USD": "1.50", "EUR": "2.0"}
	if state, reasons, err := h.r.Register(equal); err != nil || state != Staging || reasons != nil {
		t.Fatal(state, reasons, err)
	}
	h.unchanged(before, "identical registration")
	if view := h.view("a1"); !view.Registered.Equal(start) || view.Spec.Sums["USD"] != "1.5" || view.Spec.Sums["EUR"] != "2" {
		t.Fatalf("%+v", view)
	}
	changes := map[string]func(*Spec){
		"revision":   func(c *Spec) { c.Revision = 4 },
		"bytes":      func(c *Spec) { c.Bytes = digestOf("other") },
		"holds":      func(c *Spec) { *c = base("a1", "d1", 3); c.Collection = "col:1" },
		"chunks":     func(c *Spec) { c.Chunks = 2 },
		"rows":       func(c *Spec) { c.Rows = 3 },
		"sum":        func(c *Spec) { c.Sums = map[string]string{"USD": "1.5", "EUR": "2.1"} },
		"currency":   func(c *Spec) { c.Sums = map[string]string{"USD": "1.5", "KRW": "2"} },
		"fewer sums": func(c *Spec) { c.Sums = map[string]string{"USD": "1.5"} },
		"totals":     func(c *Spec) { c.Totals = CallerComputed },
		"size":       func(c *Spec) { c.Size = 99 },
		"expanded":   func(c *Spec) { c.Expanded = 201 },
		"location":   func(c *Spec) { c.Location = "store:b" },
		"collection": func(c *Spec) { c.Collection = "" },
		"scan":       func(c *Spec) { c.Scan = Failed },
		"syntax":     func(c *Spec) { c.Syntax = NotRun },
		"bodies":     func(c *Spec) { c.Bodies = Failed },
		"document":   func(c *Spec) { c.Document = "d2" },
		"period":     func(c *Spec) { c.Subject.Period = "2026-11" },
		"provider":   func(c *Spec) { c.Subject.Provider = "gcp" },
		"scope":      func(c *Spec) { c.Subject.Scope = "payer-9" },
		"kind":       func(c *Spec) { c.Kind = LocatorKeys },
	}
	for name, change := range changes {
		other := s
		change(&other)
		_, _, err := h.r.Register(other)
		wantErr(t, err, ErrConflict, name)
		h.unchanged(before, name)
	}
	h.seal(scan("o1", "d2", 1))
	h.register(scan("o2", "d2", 1))
	x := derived(t, "x1", "d2", 1, "o1", chunk)
	h.register(x)
	before = h.r.dump()
	for name, change := range map[string]func(*Spec){
		"role":     func(c *Spec) { *c = original(t, "x1", "d2", 1, chunk) },
		"original": func(c *Spec) { c.Original = "o2" },
		"method":   func(c *Spec) { c.Method = "ocr:v2" },
	} {
		other := x
		change(&other)
		_, _, err := h.r.Register(other)
		wantErr(t, err, ErrConflict, name)
		h.unchanged(before, name)
	}
	h.seal(x, chunk)
	if err := h.r.Retire("ws1", "x1", Reissued); err != nil {
		t.Fatal(err)
	}
	before = h.r.dump()
	if state, _, err := h.r.Register(x); err != nil || state != Retired {
		t.Fatal(state, err)
	}
	h.unchanged(before, "repeat after retirement")
}

func TestDerivedNeedsAnAwaitingOriginal(t *testing.T) {
	chunk := []Record{rec("k1", "1", "USD")}
	h := newHarness(t, nil)
	h.sealRows("rows1", "d1", 1, chunk)
	h.register(base("empty1", "d1", 1))
	h.register(scan("o1", "d1", 1))
	h.register(scan("o2", "d1", 2))
	h.register(scan("o3", "d2", 1))
	h.register(derived(t, "x0", "d1", 1, "o1", chunk))
	gone := scan("o4", "d1", 1)
	h.register(gone)
	if err := h.r.Abandon("ws1", "o4", Superseded); err != nil {
		t.Fatal(err)
	}
	bad := scan("o5", "d1", 1)
	bad.Scan = Failed
	h.register(bad)
	retired := scan("o6", "d1", 1)
	h.seal(retired)
	if err := h.r.Retire("ws1", "o6", Reissued); err != nil {
		t.Fatal(err)
	}
	foreign := scan("o7", "d1", 1)
	foreign.Subject.WorkspaceID = "ws2"
	h.register(foreign)
	cases := []struct {
		from string
		want error
	}{
		{"none", ErrNotFound}, {"o7", ErrNotFound}, {"rows1", ErrRejected}, {"empty1", ErrRejected}, {"x0", ErrRejected},
		{"o2", ErrRejected}, {"o3", ErrRejected}, {"o4", ErrRejected}, {"o5", ErrRejected}, {"o6", ErrRejected},
	}
	for _, c := range cases {
		before := h.r.dump()
		_, _, err := h.r.Register(derived(t, "x1", "d1", 1, c.from, chunk))
		wantErr(t, err, c.want, c.from)
		h.unchanged(before, c.from)
	}
	if state := h.register(derived(t, "x1", "d1", 1, "o1", chunk)); state != Staging {
		t.Fatal(state)
	}
	h.seal(scan("o8", "d1", 1))
	if state := h.register(derived(t, "x2", "d1", 1, "o8", chunk)); state != Staging {
		t.Fatal(state)
	}
}

func TestArtifactLimitCountsRejectedOnes(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.Artifacts = 2 })
	bad := base("a1", "d1", 1)
	bad.Scan = NotRun
	h.register(bad)
	h.register(base("a2", "d1", 1))
	before := h.r.dump()
	_, _, err := h.r.Register(base("a3", "d9", 1))
	wantErr(t, err, ErrLimit, "third artifact")
	h.unchanged(before, "limit")
	if state := h.register(base("a2", "d1", 1)); state != Staging {
		t.Fatal(state)
	}
	changed := base("a2", "d1", 2)
	_, _, err = h.r.Register(changed)
	wantErr(t, err, ErrConflict, "conflict before limit")
	elsewhere := base("a3", "d9", 1)
	elsewhere.Subject.WorkspaceID = "ws2"
	if state := h.register(elsewhere); state != Staging {
		t.Fatal(state)
	}
	h.seal(base("a2", "d1", 1))
	if _, err := h.r.Status(Subject{WorkspaceID: "ws1", Provider: "aws", Scope: "payer-1", Period: "d9"}); err != ErrNotFound {
		t.Fatal(err)
	}
}
