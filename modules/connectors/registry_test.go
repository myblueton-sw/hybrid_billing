package connectors

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	start      = time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	testLimits = Limits{Capabilities: 12, Collections: 12, Parts: 6, Pages: 6, MaxBytes: 1 << 40, MaxRows: 1 << 30, MaxWait: time.Hour}
	usEast     = Condition{Contract: "direct", Region: "us-east", Source: "cur2"}
)

type bench struct {
	t   testing.TB
	r   *Registry
	now *time.Time
}

func open(t testing.TB, limits Limits) bench {
	t.Helper()
	now := start
	registry, err := NewRegistry(limits, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	b := bench{t: t, r: registry, now: &now}
	for _, operation := range []Operation{FetchCosts, FetchInvoices} {
		for _, kind := range []Kind{ProviderOriginal, ProForma} {
			b.ok("declare", second(registry.Declare(verified(operation, kind))))
		}
	}
	return b
}

func verified(operation Operation, kind Kind) Capability {
	return Capability{Provider: "aws", Version: "v1", Operation: operation, Kind: kind, Condition: usEast, State: Verified, Evidence: "evidence-1"}
}

func subjectOf(workspace string) Subject {
	return Subject{WorkspaceID: workspace, ConnectionID: "conn-1", Operation: FetchCosts, ScopeID: "payer-1", PeriodID: "2026-10", Kind: ProviderOriginal}
}

func files(id string, revision uint64) Spec {
	return Spec{ID: id, Subject: subjectOf("w1"), Provider: "aws", Version: "v1", Condition: usEast, Revision: revision, Shape: Files}
}

func pages(id string, revision uint64) Spec {
	spec := files(id, revision)
	spec.Shape, spec.Cursor = Pages, "c0"
	return spec
}

func sum(n int) string { return fmt.Sprintf("%08x", n) }

func parts(count int) []Part {
	made := []Part{}
	for index := 1; index <= count; index++ {
		made = append(made, Part{ID: fmt.Sprintf("part-%d", index), Bytes: int64(1000 * index), Rows: int64(10 * index), Checksum: sum(index)})
	}
	return made
}

func second[T any](_ T, err error) error { return err }

func (b bench) ok(name string, err error) {
	b.t.Helper()
	if err != nil {
		b.t.Errorf("%s: %v", name, err)
	}
}

func (b bench) no(name string, err, want error) {
	b.t.Helper()
	if !errors.Is(err, want) {
		b.t.Errorf("%s: %v; want %v", name, err, want)
	}
}

func (b bench) state(name string, got State, err error, want State) {
	b.t.Helper()
	if err != nil || got != want {
		b.t.Errorf("%s: %s %v; want %s", name, got, err, want)
	}
}

func (b bench) view(id string) CollectionView {
	b.t.Helper()
	view, err := b.r.Collection("w1", id)
	if err != nil {
		b.t.Fatalf("collection %s: %v", id, err)
	}
	return view
}

func (b bench) gaps(id string) string {
	b.t.Helper()
	missing, err := b.r.Complete("w1", id)
	if err == nil {
		return "completed"
	}
	if !errors.Is(err, ErrRejected) {
		return err.Error()
	}
	texts := []string{}
	for _, gap := range missing {
		text := gap.What
		if gap.Part != "" {
			text += ":" + gap.Part
		}
		texts = append(texts, text)
	}
	return strings.Join(texts, " ")
}

func (b bench) filled(spec Spec, count int) {
	b.t.Helper()
	b.ok("start "+spec.ID, b.r.Start(spec))
	b.ok("manifest", second(b.r.DeclareManifest("w1", spec.ID, SourceListed, parts(count))))
	for _, given := range parts(count) {
		b.ok("part", second(b.r.ReceivePart("w1", spec.ID, given)))
	}
}

func (b bench) standings(about Subject) string {
	b.t.Helper()
	view, err := b.r.Effective(about)
	if err != nil {
		return err.Error()
	}
	text := "none"
	if view.Collection != nil {
		text = view.Collection.Spec.ID
		if view.Stale {
			text += "/stale"
		}
		if view.Collection.Empty {
			text += "/empty"
		}
	}
	for _, other := range view.Others {
		text += fmt.Sprintf(" %s=%s", other.ID, other.Standing)
	}
	return fmt.Sprintf("%s @%d known %d", text, view.Pointer, view.Known)
}

func TestCapabilities(t *testing.T) {
	b := open(t, testLimits)
	answer, err := b.r.Permit("aws", "v1", FetchCosts, ProviderOriginal, usEast)
	if err != nil || answer != (Permission{Answer: Allowed, Revision: 1}) {
		t.Errorf("a verified capability: %+v %v", answer, err)
	}
	for name, condition := range map[string]Condition{
		"another contract": {"reseller", "us-east", "cur2"}, "another region": {"direct", "eu-west", "cur2"}, "another source version": {"direct", "us-east", "cur1"},
	} {
		if answer, _ := b.r.Permit("aws", "v1", FetchCosts, ProviderOriginal, condition); answer.Answer != Unknown {
			t.Errorf("%s: %+v", name, answer)
		}
	}
	for name, answer := range map[string]Permission{
		"another adapter version": first(b.r.Permit("aws", "v2", FetchCosts, ProviderOriginal, usEast)),
		"another provider":        first(b.r.Permit("azure", "v1", FetchCosts, ProviderOriginal, usEast)),
		"another data kind":       first(b.r.Permit("aws", "v1", FetchCosts, DistributorSales, usEast)),
		"another operation":       first(b.r.Permit("aws", "v1", Discover, NoKind, usEast)),
	} {
		if answer.Answer != Unknown || answer.Revision != 0 {
			t.Errorf("%s: %+v", name, answer)
		}
	}
	limited := verified(FetchCosts, DistributorSales)
	limited.State, limited.Reason = Limited, "partial-tags"
	b.ok("a limited capability", second(b.r.Declare(limited)))
	unsupported := Capability{Provider: "aws", Version: "v1", Operation: Discover, Kind: NoKind, Condition: usEast, State: Unsupported, Reason: "no-api", Alternative: "file-import"}
	b.ok("an unsupported capability", second(b.r.Declare(unsupported)))
	if answer, _ := b.r.Permit("aws", "v1", FetchCosts, DistributorSales, usEast); answer != (Permission{Answer: AnswerLimited, Reason: "partial-tags", Revision: 1}) {
		t.Errorf("limited: %+v", answer)
	}
	if answer, _ := b.r.Permit("aws", "v1", Discover, NoKind, usEast); answer != (Permission{Answer: AnswerUnsupported, Reason: "no-api", Alternative: "file-import", Revision: 1}) {
		t.Errorf("unsupported: %+v", answer)
	}
	if revision, err := b.r.Declare(limited); err != nil || revision != 1 {
		t.Errorf("the same declaration again: %d %v", revision, err)
	}
	back := verified(FetchCosts, DistributorSales)
	if revision, err := b.r.Declare(back); err != nil || revision != 2 {
		t.Errorf("a replacement: %d %v", revision, err)
	}
	if revision, err := b.r.Declare(limited); err != nil || revision != 3 {
		t.Errorf("a return to earlier content does not return to its revision: %d %v", revision, err)
	}
	bad := map[string]func(*Capability){
		"no provider": func(c *Capability) { c.Provider = "" }, "a version that is text": func(c *Capability) { c.Version = "version one" },
		"an unknown operation": func(c *Capability) { c.Operation = "delete" }, "a fetch without a data kind": func(c *Capability) { c.Kind = NoKind },
		"a data kind on a discovery": func(c *Capability) { c.Operation = Discover }, "an unknown kind": func(c *Capability) { c.Kind = "estimate" },
		"no contract": func(c *Capability) { c.Condition.Contract = "" }, "no region": func(c *Capability) { c.Condition.Region = "" }, "no source version": func(c *Capability) { c.Condition.Source = "" },
		"an unknown state": func(c *Capability) { c.State = "beta" }, "verified without evidence": func(c *Capability) { c.Evidence = "" },
		"verified with a reason": func(c *Capability) { c.Reason = "x" }, "verified with an alternative": func(c *Capability) { c.Alternative = "x" },
		"limited without a reason":     func(c *Capability) { c.State = Limited },
		"limited without evidence":     func(c *Capability) { c.State, c.Reason, c.Evidence = Limited, "x", "" },
		"unsupported with evidence":    func(c *Capability) { c.State, c.Reason = Unsupported, "x" },
		"unsupported without a reason": func(c *Capability) { c.State, c.Evidence = Unsupported, "" },
		"an alternative that is text":  func(c *Capability) { c.State, c.Evidence, c.Reason, c.Alternative = Unsupported, "", "x", "ask sales" },
		"evidence that is an address":  func(c *Capability) { c.Evidence = "https://example.invalid/e" },
	}
	for name, change := range bad {
		stated := verified(FetchCosts, ProviderOriginal)
		change(&stated)
		b.no(name, second(b.r.Declare(stated)), ErrInvalid)
	}
	unknownOperation := Capability{Provider: "aws", Version: "v1", Operation: "delete", Kind: NoKind, Condition: usEast, State: Verified, Evidence: "e"}
	b.no("an unknown operation without a data kind", second(b.r.Declare(unknownOperation)), ErrInvalid)
	b.no("a kind with a discovery asked", second(b.r.Permit("aws", "v1", Discover, ProviderOriginal, usEast)), ErrInvalid)
	b.no("a fetch without a kind asked", second(b.r.Permit("aws", "v1", FetchCosts, NoKind, usEast)), ErrInvalid)
	for index := 0; index < 6; index++ {
		extra := verified(FetchCosts, ProviderOriginal)
		extra.Version = fmt.Sprintf("v%d", index+10)
		b.ok("filling the limit", second(b.r.Declare(extra)))
	}
	over := verified(FetchCosts, ProviderOriginal)
	over.Version = "v99"
	b.no("a new capability beyond the limit", second(b.r.Declare(over)), ErrLimit)
	withdrawn := verified(FetchCosts, ProviderOriginal)
	withdrawn.State, withdrawn.Evidence, withdrawn.Reason = Unsupported, "", "parser-wrong"
	if revision, err := b.r.Declare(withdrawn); err != nil || revision != 2 {
		t.Errorf("a withdrawal at the limit: %d %v", revision, err)
	}
}

func first[T any](value T, _ error) T { return value }

func TestStarting(t *testing.T) {
	b := open(t, testLimits)
	b.ok("a start", b.r.Start(files("a", 5)))
	b.ok("the same start again", b.r.Start(files("a", 5)))
	b.no("the same identifier at another revision", b.r.Start(files("a", 6)), ErrConflict)
	b.no("the same identifier in another shape", b.r.Start(pages("a", 5)), ErrConflict)
	view := b.view("a")
	if view.State != Collecting || view.Capability != 1 || view.Spec != files("a", 5) || !view.Started.Equal(start) || view.Declared {
		t.Errorf("a new collection: %+v", view)
	}
	for name, change := range map[string]func(*Spec){
		"no identifier": func(s *Spec) { s.ID = "" }, "no workspace": func(s *Spec) { s.Subject.WorkspaceID = "" }, "a connection that is an address": func(s *Spec) { s.Subject.ConnectionID = "db://x" },
		"an operation that fetches nothing": func(s *Spec) { s.Subject.Operation = Discover }, "no data kind": func(s *Spec) { s.Subject.Kind = NoKind },
		"a scope that is text": func(s *Spec) { s.Subject.ScopeID = "payer one" }, "no period": func(s *Spec) { s.Subject.PeriodID = "" },
		"no provider": func(s *Spec) { s.Provider = "" }, "no condition": func(s *Spec) { s.Condition = Condition{} }, "a revision of zero": func(s *Spec) { s.Revision = 0 },
		"an unknown shape": func(s *Spec) { s.Shape = "stream" }, "a cursor on files": func(s *Spec) { s.Cursor = "c0" },
	} {
		spec := files("b", 5)
		change(&spec)
		b.no(name, b.r.Start(spec), ErrInvalid)
	}
	noCursor := pages("b", 5)
	noCursor.Cursor = ""
	b.no("pages without a start cursor", b.r.Start(noCursor), ErrInvalid)
	for name, change := range map[string]func(*Spec){
		"an unknown capability":       func(s *Spec) { s.Version = "v2" },
		"another region":              func(s *Spec) { s.Condition.Region = "eu-west" },
		"a kind that is not declared": func(s *Spec) { s.Subject.Kind = DistributorSales },
	} {
		spec := files("b", 5)
		change(&spec)
		b.no(name, b.r.Start(spec), ErrRejected)
	}
	limited := verified(FetchCosts, DistributorSales)
	limited.State, limited.Reason = Limited, "partial"
	b.r.Declare(limited)
	sales := files("b", 5)
	sales.Subject.Kind = DistributorSales
	b.no("a limited capability", b.r.Start(sales), ErrRejected)
	azure := verified(FetchCosts, ProviderOriginal)
	azure.Provider = "azure"
	b.r.Declare(azure)
	other := files("b", 5)
	other.Provider = "azure"
	other.Subject.ScopeID = "payer-2"
	b.no("a connection started under a second provider", b.r.Start(other), ErrConflict)
	elsewhere := other
	elsewhere.Subject.WorkspaceID = "w2"
	b.ok("the same connection identifier in another workspace under another provider", b.r.Start(elsewhere))
	b.no("a start below the known revision", b.r.Start(files("b", 4)), ErrRejected)
	if _, err := b.r.Collection("w1", "b"); !errors.Is(err, ErrNotFound) {
		t.Errorf("a refused start left a collection: %v", err)
	}
	b.ok("a second collection at the known revision", b.r.Start(files("b", 5)))
	b.ok("a start at a higher revision", b.r.Start(files("c", 6)))
	if b.view("a").State != Superseded || b.view("b").State != Superseded || b.view("c").State != Collecting {
		t.Errorf("a higher revision did not supersede the running collections: %s %s %s", b.view("a").State, b.view("b").State, b.view("c").State)
	}
	b.ok("a repeated start after supersession", b.r.Start(files("a", 5)))
	if _, err := b.r.Collection("w2", "a"); !errors.Is(err, ErrNotFound) {
		t.Errorf("a collection of another workspace: %v", err)
	}
	*b.now = b.now.Add(time.Hour)
	b.r.Collection("w1", "a")
	*b.now = b.now.Add(-2 * time.Hour)
	late := files("late", 6)
	late.Subject.ScopeID = "payer-9"
	b.ok("a start after the clock went back", b.r.Start(late))
	if view := b.view("late"); !view.Started.Equal(start.Add(time.Hour)) {
		t.Errorf("the registry's time went backwards to %s", view.Started)
	}
	if _, err := b.r.Collection("w9", "a"); !errors.Is(err, ErrNotFound) {
		t.Errorf("a collection of an unknown workspace: %v", err)
	}
	small := open(t, Limits{Capabilities: 12, Collections: 2, Parts: 6, Pages: 6, MaxBytes: 1 << 40, MaxRows: 1 << 30, MaxWait: time.Hour})
	small.ok("one", small.r.Start(files("a", 1)))
	small.ok("abandoned", small.r.Abandon("w1", "a", ByOperator))
	small.ok("two", small.r.Start(files("b", 1)))
	small.no("a third collection although one is final", small.r.Start(files("c", 1)), ErrLimit)
	small.ok("a repeated start at the limit", small.r.Start(files("b", 1)))
	inOther := files("c", 1)
	inOther.Subject.WorkspaceID = "w2"
	small.ok("another workspace has its own room", small.r.Start(inOther))
}

func TestFiles(t *testing.T) {
	b := open(t, testLimits)
	b.ok("start", b.r.Start(files("a", 5)))
	b.no("a part before the manifest", second(b.r.ReceivePart("w1", "a", parts(1)[0])), ErrRejected)
	if got := b.gaps("a"); got != "no_manifest" {
		t.Errorf("completing without a manifest: %s", got)
	}
	three := parts(3)
	twice := append(parts(2), parts(1)[0])
	b.no("a manifest that names a part twice", second(b.r.DeclareManifest("w1", "a", SourceListed, twice)), ErrInvalid)
	b.no("an unknown origin", second(b.r.DeclareManifest("w1", "a", "guessed", three)), ErrInvalid)
	b.no("more parts than a manifest may have", second(b.r.DeclareManifest("w1", "a", SourceListed, parts(7))), ErrLimit)
	for name, change := range map[string]func(*Part){
		"a part name that is a path": func(p *Part) { p.ID = "dir/part" }, "a negative byte count": func(p *Part) { p.Bytes = -1 }, "a byte count above the bound": func(p *Part) { p.Bytes = 1<<40 + 1 },
		"a negative row count": func(p *Part) { p.Rows = -1 }, "a row count above the bound": func(p *Part) { p.Rows = 1<<30 + 1 },
		"a checksum in capitals": func(p *Part) { p.Checksum = "ABCDEF12" }, "a checksum in base64": func(p *Part) { p.Checksum = "q83vEg==" }, "a short checksum": func(p *Part) { p.Checksum = "abc" },
	} {
		list := parts(3)
		change(&list[1])
		b.no(name, second(b.r.DeclareManifest("w1", "a", SourceListed, list)), ErrInvalid)
	}
	b.state("the manifest", first(b.r.DeclareManifest("w1", "a", SourceListed, three)), nil, Collecting)
	b.state("the same manifest in another order", first(b.r.DeclareManifest("w1", "a", SourceListed, []Part{three[2], three[0], three[1]})), nil, Collecting)
	b.no("a page on a file collection", second(b.r.ReceivePage("w1", "a", "c0", "", 1, sum(1))), ErrRejected)
	b.no("a total on a file collection", second(b.r.DeclareTotal("w1", "a", 1, 1)), ErrRejected)
	b.no("a part outside the manifest", second(b.r.ReceivePart("w1", "a", parts(4)[3])), ErrRejected)
	b.state("the first part", first(b.r.ReceivePart("w1", "a", three[0])), nil, Collecting)
	b.state("the first part again", first(b.r.ReceivePart("w1", "a", three[0])), nil, Collecting)
	if got := b.gaps("a"); got != "part_missing:part-2 part_missing:part-3" {
		t.Errorf("one split file received of three: %s", got)
	}
	for _, c := range []struct {
		name   string
		change func(*Part)
	}{{"a wrong byte count", func(p *Part) { p.Bytes++ }}, {"a wrong row count", func(p *Part) { p.Rows-- }}, {"a wrong checksum", func(p *Part) { p.Checksum = sum(99) }}} {
		wrong := three[1]
		c.change(&wrong)
		b.state(c.name, first(b.r.ReceivePart("w1", "a", wrong)), nil, Collecting)
	}
	view := b.view("a")
	if view.Parts[1].Verified || view.Parts[1].Mismatches != 3 || view.Parts[1].Last.Checksum != sum(99) || !view.Parts[0].Verified || view.Parts[2].Mismatches != 0 {
		t.Errorf("the record of a mismatched part: %+v", view.Parts)
	}
	b.ok("the third part", second(b.r.ReceivePart("w1", "a", three[2])))
	if got := b.gaps("a"); got != "part_mismatched:part-2" {
		t.Errorf("one split file mismatched: %s", got)
	}
	b.state("the mismatched part received correctly", first(b.r.ReceivePart("w1", "a", three[1])), nil, Collecting)
	if got := b.gaps("a"); got != "completed" {
		t.Errorf("everything received: %s", got)
	}
	done := b.view("a")
	if done.State != Completed || done.Totals != (Totals{Units: 3, Bytes: 6000, Rows: 60}) || done.Empty || done.Completion != 1 || len(done.Digest) != 64 || done.Origin != SourceListed {
		t.Errorf("the completed collection: %+v", done)
	}
	if got := b.gaps("a"); got != "completed" {
		t.Errorf("completing again: %s", got)
	}
	b.state("a verified part again after completion", first(b.r.ReceivePart("w1", "a", three[0])), nil, Completed)
	late := three[0]
	late.Rows++
	b.no("a late contradicting report on a completed collection", second(b.r.ReceivePart("w1", "a", late)), ErrRejected)
	b.state("the manifest again after completion", first(b.r.DeclareManifest("w1", "a", SourceListed, three)), nil, Completed)
	b.no("another manifest after completion", second(b.r.DeclareManifest("w1", "a", SourceListed, parts(2))), ErrRejected)
	if b.view("a").State != Completed {
		t.Errorf("a refused report changed a completed collection")
	}

	b.ok("start", b.r.Start(files("b", 5)))
	b.ok("manifest", second(b.r.DeclareManifest("w1", "b", SourceListed, three)))
	b.ok("the right report", second(b.r.ReceivePart("w1", "b", three[0])))
	wrong := three[0]
	wrong.Bytes++
	b.state("a mismatch after verification", first(b.r.ReceivePart("w1", "b", wrong)), nil, Inconsistent)
	b.state("the delayed right report afterwards", first(b.r.ReceivePart("w1", "b", three[0])), nil, Inconsistent)
	b.no("another part afterwards", second(b.r.ReceivePart("w1", "b", three[1])), ErrRejected)
	if got := b.gaps("b"); got != "" {
		t.Errorf("completing an inconsistent collection: %q", got)
	}
	if view := b.view("b"); view.Contradiction != "part:part-1" || !view.Changed.Equal(start) {
		t.Errorf("the recorded contradiction: %+v", view)
	}
	b.no("abandoning a final collection", b.r.Abandon("w1", "b", ByOperator), ErrRejected)

	b.ok("start", b.r.Start(files("c", 5)))
	b.ok("manifest", second(b.r.DeclareManifest("w1", "c", SourceListed, three)))
	b.ok("a wrong report first", second(b.r.ReceivePart("w1", "c", wrong)))
	b.ok("the right one", second(b.r.ReceivePart("w1", "c", three[0])))
	b.state("a delayed copy of the wrong one", first(b.r.ReceivePart("w1", "c", wrong)), nil, Inconsistent)

	b.ok("start", b.r.Start(files("d", 5)))
	b.ok("manifest", second(b.r.DeclareManifest("w1", "d", SourceListed, three)))
	b.state("a second manifest with other content", first(b.r.DeclareManifest("w1", "d", SourceListed, parts(2))), nil, Inconsistent)
	b.ok("start", b.r.Start(files("e", 5)))
	b.ok("manifest", second(b.r.DeclareManifest("w1", "e", SourceListed, three)))
	b.state("the same parts with another origin", first(b.r.DeclareManifest("w1", "e", AdapterDerived, three)), nil, Inconsistent)
	changed := parts(3)
	changed[2].Rows++
	b.ok("start", b.r.Start(files("f", 5)))
	b.ok("manifest", second(b.r.DeclareManifest("w1", "f", SourceListed, three)))
	b.state("the same part names with another value", first(b.r.DeclareManifest("w1", "f", SourceListed, changed)), nil, Inconsistent)

	b.ok("start", b.r.Start(files("g", 5)))
	b.state("an empty manifest derived by the adapter", first(b.r.DeclareManifest("w1", "g", AdapterDerived, nil)), nil, Collecting)
	if got := b.gaps("g"); got != "completed" {
		t.Errorf("an empty manifest: %s", got)
	}
	if view := b.view("g"); !view.Empty || view.Totals != (Totals{}) || view.Origin != AdapterDerived || view.Completion != 2 {
		t.Errorf("an empty completed collection: %+v", view)
	}
}

func TestPages(t *testing.T) {
	b := open(t, Limits{Capabilities: 12, Collections: 40, Parts: 6, Pages: 6, MaxBytes: 1 << 40, MaxRows: 1 << 30, MaxWait: time.Hour})
	b.ok("start", b.r.Start(pages("a", 5)))
	b.no("a manifest on a paged collection", second(b.r.DeclareManifest("w1", "a", SourceListed, parts(1))), ErrRejected)
	b.no("a part on a paged collection", second(b.r.ReceivePart("w1", "a", parts(1)[0])), ErrRejected)
	b.no("a cursor that is not the awaited one", second(b.r.ReceivePage("w1", "a", "c1", "c2", 10, sum(1))), ErrRejected)
	for name, attempt := range map[string]func() (State, error){
		"a cursor with a slash":   func() (State, error) { return b.r.ReceivePage("w1", "a", "a/b", "c1", 10, sum(1)) },
		"a next cursor in base64": func() (State, error) { return b.r.ReceivePage("w1", "a", "c0", "q83v+g==", 10, sum(1)) },
		"negative rows":           func() (State, error) { return b.r.ReceivePage("w1", "a", "c0", "c1", -1, sum(1)) },
		"rows above the bound":    func() (State, error) { return b.r.ReceivePage("w1", "a", "c0", "c1", 1<<30+1, sum(1)) },
		"a checksum that is text": func() (State, error) { return b.r.ReceivePage("w1", "a", "c0", "c1", 10, "see log") },
	} {
		b.no(name, second(attempt()), ErrInvalid)
	}
	b.state("the first page", first(b.r.ReceivePage("w1", "a", "c0", "c1", 10, sum(1))), nil, Collecting)
	b.state("the first page again", first(b.r.ReceivePage("w1", "a", "c0", "c1", 10, sum(1))), nil, Collecting)
	if got := b.gaps("a"); got != "chain_unfinished" {
		t.Errorf("completing a chain that has not ended: %s", got)
	}
	b.state("the second page", first(b.r.ReceivePage("w1", "a", "c1", "c2", 20, sum(2))), nil, Collecting)
	b.state("the first page again after the chain moved on", first(b.r.ReceivePage("w1", "a", "c0", "c1", 10, sum(1))), nil, Collecting)
	b.state("the last page", first(b.r.ReceivePage("w1", "a", "c2", "", 5, sum(3))), nil, Collecting)
	b.no("a new cursor after the chain ended", second(b.r.ReceivePage("w1", "a", "c3", "", 5, sum(4))), ErrRejected)
	if view := b.view("a"); view.Pages != 3 || !view.Ended || view.Waiting != "" {
		t.Errorf("the chain: %+v", view)
	}
	if got := b.gaps("a"); got != "completed" {
		t.Errorf("a finished chain: %s", got)
	}
	if view := b.view("a"); view.Totals != (Totals{Units: 3, Rows: 35}) || view.Empty {
		t.Errorf("a repeated page was counted: %+v", view.Totals)
	}
	b.state("a received page again after completion", first(b.r.ReceivePage("w1", "a", "c1", "c2", 20, sum(2))), nil, Completed)
	b.no("a page with other content after completion", second(b.r.ReceivePage("w1", "a", "c1", "c2", 21, sum(2))), ErrRejected)

	inconsistent := map[string]func(id string) (State, error){
		"a page repeated with other rows": func(id string) (State, error) {
			b.r.ReceivePage("w1", id, "c0", "c1", 10, sum(1))
			return b.r.ReceivePage("w1", id, "c0", "c1", 11, sum(1))
		},
		"a page repeated with another next": func(id string) (State, error) {
			b.r.ReceivePage("w1", id, "c0", "c1", 10, sum(1))
			return b.r.ReceivePage("w1", id, "c0", "c9", 10, sum(1))
		},
		"a page repeated with another checksum": func(id string) (State, error) {
			b.r.ReceivePage("w1", id, "c0", "c1", 10, sum(1))
			return b.r.ReceivePage("w1", id, "c0", "c1", 10, sum(2))
		},
		"a page whose next is its own cursor": func(id string) (State, error) { return b.r.ReceivePage("w1", id, "c0", "c0", 10, sum(1)) },
		"a page whose next is the start cursor": func(id string) (State, error) {
			b.r.ReceivePage("w1", id, "c0", "c1", 10, sum(1))
			return b.r.ReceivePage("w1", id, "c1", "c0", 10, sum(2))
		},
		"a page whose next is an earlier cursor": func(id string) (State, error) {
			b.r.ReceivePage("w1", id, "c0", "c1", 10, sum(1))
			b.r.ReceivePage("w1", id, "c1", "c2", 10, sum(2))
			return b.r.ReceivePage("w1", id, "c2", "c1", 10, sum(3))
		},
		"the same page under a new cursor": func(id string) (State, error) {
			b.r.ReceivePage("w1", id, "c0", "c1", 10, sum(1))
			return b.r.ReceivePage("w1", id, "c1", "c2", 10, sum(1))
		},
		"another total": func(id string) (State, error) {
			b.r.DeclareTotal("w1", id, 3, 30)
			return b.r.DeclareTotal("w1", id, 3, 31)
		},
		"more pages than the declared total": func(id string) (State, error) {
			b.r.DeclareTotal("w1", id, 1, 30)
			b.r.ReceivePage("w1", id, "c0", "c1", 10, sum(1))
			return b.r.ReceivePage("w1", id, "c1", "c2", 20, sum(2))
		},
		"a chain that ends with other rows than declared": func(id string) (State, error) {
			b.r.DeclareTotal("w1", id, 2, 31)
			b.r.ReceivePage("w1", id, "c0", "c1", 10, sum(1))
			return b.r.ReceivePage("w1", id, "c1", "", 20, sum(2))
		},
		"a chain that ends with fewer pages than declared": func(id string) (State, error) {
			b.r.DeclareTotal("w1", id, 3, 30)
			b.r.ReceivePage("w1", id, "c0", "c1", 10, sum(1))
			return b.r.ReceivePage("w1", id, "c1", "", 20, sum(2))
		},
		"a total declared after a chain that ended differently": func(id string) (State, error) {
			b.r.ReceivePage("w1", id, "c0", "", 10, sum(1))
			return b.r.DeclareTotal("w1", id, 1, 11)
		},
	}
	index := 0
	for name, attempt := range inconsistent {
		index++
		id := fmt.Sprintf("bad-%d", index)
		spec := pages(id, 5)
		spec.Subject.ScopeID = id
		b.ok("start", b.r.Start(spec))
		got, err := attempt(id)
		b.state(name, got, err, Inconsistent)
		b.no(name+", then abandoned", b.r.Abandon("w1", id, ByOperator), ErrRejected)
	}
	mixed := pages("mixed", 5)
	mixed.Subject.ScopeID = "mixed"
	b.ok("start", b.r.Start(mixed))
	b.state("a page with rows", first(b.r.ReceivePage("w1", "mixed", "c0", "c1", 4, sum(7))), nil, Collecting)
	b.state("an empty page with the checksum of a page that had rows", first(b.r.ReceivePage("w1", "mixed", "c1", "c2", 0, sum(7))), nil, Collecting)
	b.state("an empty page with a checksum of its own", first(b.r.ReceivePage("w1", "mixed", "c2", "c3", 0, sum(8))), nil, Collecting)
	b.state("a page with rows and the checksum of an empty page", first(b.r.ReceivePage("w1", "mixed", "c3", "", 2, sum(8))), nil, Collecting)
	same := pages("zero", 5)
	same.Subject.ScopeID = "zero"
	b.ok("start", b.r.Start(same))
	b.state("an empty page", first(b.r.ReceivePage("w1", "zero", "c0", "c1", 0, sum(7))), nil, Collecting)
	b.state("a second empty page with the same checksum", first(b.r.ReceivePage("w1", "zero", "c1", "", 0, sum(7))), nil, Collecting)
	if got, _ := b.r.Complete("w1", "zero"); got != nil || !b.view("zero").Empty {
		t.Errorf("an empty chain: %v %+v", got, b.view("zero"))
	}
	total := pages("total", 5)
	total.Subject.ScopeID = "total"
	b.ok("start", b.r.Start(total))
	b.no("a total of no pages", second(b.r.DeclareTotal("w1", "total", 0, 0)), ErrInvalid)
	b.no("a total of negative rows", second(b.r.DeclareTotal("w1", "total", 1, -1)), ErrInvalid)
	b.state("a total", first(b.r.DeclareTotal("w1", "total", 2, 30)), nil, Collecting)
	b.state("the same total again", first(b.r.DeclareTotal("w1", "total", 2, 30)), nil, Collecting)
	b.r.ReceivePage("w1", "total", "c0", "c1", 10, sum(1))
	b.state("the chain ends as declared", first(b.r.ReceivePage("w1", "total", "c1", "", 20, sum(2))), nil, Collecting)
	if got := b.gaps("total"); got != "completed" {
		t.Errorf("declared totals met: %s", got)
	}
	long := pages("long", 5)
	long.Subject.ScopeID = "long"
	b.ok("start", b.r.Start(long))
	for page := 0; page < 6; page++ {
		b.ok("page", second(b.r.ReceivePage("w1", "long", fmt.Sprintf("c%d", page), fmt.Sprintf("c%d", page+1), 1, sum(page+1))))
	}
	b.no("a page beyond the limit", second(b.r.ReceivePage("w1", "long", "c6", "", 1, sum(9))), ErrLimit)
	b.ok("leaving a collection that reached the page limit", b.r.Abandon("w1", "long", LimitReached))
	if view := b.view("long"); view.State != Abandoned || view.AbandonReason != LimitReached {
		t.Errorf("abandoned: %+v", view)
	}
}

func TestInterruptionsAndSourceChanges(t *testing.T) {
	b := open(t, testLimits)
	three := parts(3)
	b.filled(files("a", 5), 3)
	b.ok("a 429", b.r.Throttled("w1", "a", 30*time.Second))
	b.ok("another", b.r.Throttled("w1", "a", time.Minute))
	b.no("a negative wait", b.r.Throttled("w1", "a", -time.Second), ErrInvalid)
	b.no("a wait above the bound", b.r.Throttled("w1", "a", time.Hour+1), ErrInvalid)
	if view := b.view("a"); view.State != Collecting || view.Throttles != 2 || view.LastWait != time.Minute {
		t.Errorf("throttling changed more than its record: %+v", view)
	}
	b.no("an unknown reason", b.r.AccessLost("w1", "a", 1, "tired"), ErrInvalid)
	b.no("an interruption of number zero", b.r.AccessLost("w1", "a", 0, PermissionWithdrawn), ErrInvalid)
	b.no("an interruption that skips a number", b.r.AccessLost("w1", "a", 2, PermissionWithdrawn), ErrConflict)
	b.ok("access lost", b.r.AccessLost("w1", "a", 1, PermissionWithdrawn))
	b.ok("the same report again", b.r.AccessLost("w1", "a", 1, PermissionWithdrawn))
	b.no("the same interruption with another reason", b.r.AccessLost("w1", "a", 1, CredentialExpired), ErrConflict)
	b.no("a second interruption while interrupted", b.r.AccessLost("w1", "a", 2, CredentialExpired), ErrRejected)
	if view := b.view("a"); view.State != Interrupted || view.Interruption != 1 || view.LossReason != PermissionWithdrawn || !view.Parts[2].Verified {
		t.Errorf("interrupted: %+v", view)
	}
	if got := b.gaps("a"); got != "" {
		t.Errorf("completing while interrupted: %q", got)
	}
	if b.view("a").State != Interrupted {
		t.Errorf("a refused completion changed the state")
	}
	b.no("a 429 while interrupted", b.r.Throttled("w1", "a", time.Second), ErrRejected)
	b.state("a verified part again while interrupted", first(b.r.ReceivePart("w1", "a", three[0])), nil, Interrupted)
	b.no("a resumption of an interruption that does not exist", b.r.Resume("w1", "a", 2), ErrRejected)
	b.no("a resumption of number zero", b.r.Resume("w1", "a", 0), ErrInvalid)
	b.ok("resumed", b.r.Resume("w1", "a", 1))
	b.ok("resumed again", b.r.Resume("w1", "a", 1))
	b.ok("a delayed copy of the first loss report", b.r.AccessLost("w1", "a", 1, PermissionWithdrawn))
	if b.view("a").State != Collecting {
		t.Errorf("a delayed loss report interrupted a resumed collection")
	}
	b.ok("a second loss", b.r.AccessLost("w1", "a", 2, CredentialExpired))
	b.ok("a delayed resumption of the first interruption", b.r.Resume("w1", "a", 1))
	if view := b.view("a"); view.State != Interrupted || view.Interruption != 2 || view.LossReason != CredentialExpired {
		t.Errorf("a delayed resumption cleared a newer interruption: %+v", view)
	}
	b.ok("resumed", b.r.Resume("w1", "a", 2))
	if got := b.gaps("a"); got != "completed" {
		t.Errorf("completing after the resumption: %s", got)
	}
	b.ok("a resumption repeated after completion", b.r.Resume("w1", "a", 2))
	b.ok("a loss report repeated after completion", b.r.AccessLost("w1", "a", 2, CredentialExpired))
	b.no("a new interruption of a completed collection", b.r.AccessLost("w1", "a", 3, CredentialExpired), ErrRejected)

	b.filled(files("b", 5), 3)
	b.filled(files("c", 5), 3)
	b.ok("lost", b.r.AccessLost("w1", "b", 1, ConnectionDisabled))
	b.no("a change to the collection's own revision", b.r.SourceChanged("w1", "b", 5), ErrRejected)
	b.no("a change to a lower revision", b.r.SourceChanged("w1", "b", 4), ErrRejected)
	b.ok("a changed source reported while interrupted", b.r.SourceChanged("w1", "b", 6))
	if b.view("b").State != Superseded || b.view("c").State != Superseded || b.view("a").State != Completed {
		t.Errorf("a changed source: %s %s %s", b.view("b").State, b.view("c").State, b.view("a").State)
	}
	b.ok("the same news again", b.r.SourceChanged("w1", "b", 6))
	b.ok("the same news through the completed collection", b.r.SourceChanged("w1", "a", 6))
	b.no("resuming a superseded collection", b.r.Resume("w1", "b", 1), ErrRejected)
	if got := b.gaps("c"); got == "completed" {
		t.Errorf("a superseded collection completed")
	}
	if got := b.standings(subjectOf("w1")); got != "a/stale @1 known 6" {
		t.Errorf("the effective collection after the source changed: %s", got)
	}
	b.ok("a higher revision through the completed collection", b.r.SourceChanged("w1", "a", 7))
	b.no("a start below the revision that became known", b.r.Start(files("d", 6)), ErrRejected)
	b.filled(files("d", 7), 3)
	b.ok("lost", b.r.AccessLost("w1", "d", 1, PermissionWithdrawn))
	again := verified(FetchCosts, ProviderOriginal)
	again.Evidence = "evidence-2"
	b.ok("the capability declared again with new evidence", second(b.r.Declare(again)))
	b.no("resuming under a capability that was declared again", b.r.Resume("w1", "d", 1), ErrRejected)
	b.ok("leaving it", b.r.Abandon("w1", "d", CapabilityChanged))
	b.ok("the same again", b.r.Abandon("w1", "d", CapabilityChanged))
	b.no("with another reason", b.r.Abandon("w1", "d", ByOperator), ErrConflict)
	b.no("an unknown reason", b.r.Abandon("w1", "d", "bored"), ErrInvalid)
	b.filled(files("e", 7), 3)
	back := verified(FetchCosts, ProviderOriginal)
	b.ok("the capability declared back to its first content", second(b.r.Declare(back)))
	if got := b.gaps("e"); got != "capability_changed" {
		t.Errorf("completing under a capability that was declared again: %s", got)
	}
	b.ok("leaving it", b.r.Abandon("w1", "e", CapabilityChanged))
	b.filled(files("f", 7), 3)
	if got := b.gaps("f"); got != "completed" {
		t.Errorf("a collection started under the present capability: %s", got)
	}
}

func TestContradictionsWhileInterruptedAndOtherExits(t *testing.T) {
	b := open(t, Limits{Capabilities: 12, Collections: 40, Parts: 3, Pages: 6, MaxBytes: 1 << 40, MaxRows: 1 << 30, MaxWait: time.Hour})
	three := parts(3)
	cases := map[string]struct {
		paged  bool
		before func(id string)
		report func(id string) (State, error)
		cause  string
	}{
		"a verified part with other values": {false, func(id string) {
			b.r.DeclareManifest("w1", id, SourceListed, three)
			b.r.ReceivePart("w1", id, three[0])
		}, func(id string) (State, error) {
			wrong := three[0]
			wrong.Bytes++
			return b.r.ReceivePart("w1", id, wrong)
		}, "part:part-1"},
		"another manifest": {false, func(id string) { b.r.DeclareManifest("w1", id, SourceListed, three) },
			func(id string) (State, error) { return b.r.DeclareManifest("w1", id, SourceListed, parts(2)) }, "manifest"},
		"another total": {true, func(id string) { b.r.DeclareTotal("w1", id, 2, 30) },
			func(id string) (State, error) { return b.r.DeclareTotal("w1", id, 2, 31) }, "total"},
		"a received page with other content": {true, func(id string) { b.r.ReceivePage("w1", id, "c0", "c1", 10, sum(1)) },
			func(id string) (State, error) { return b.r.ReceivePage("w1", id, "c0", "c1", 11, sum(1)) }, "page:c0"},
	}
	index := 0
	for name, c := range cases {
		index++
		id := fmt.Sprintf("x%d", index)
		spec := files(id, 5)
		if c.paged {
			spec = pages(id, 5)
		}
		spec.Subject.ScopeID = id
		b.ok("start", b.r.Start(spec))
		c.before(id)
		b.ok("lost", b.r.AccessLost("w1", id, 1, PermissionWithdrawn))
		b.pass(time.Minute)
		got, err := c.report(id)
		b.state(name+" while interrupted", got, err, Inconsistent)
		if view := b.view(id); view.Contradiction != c.cause || !view.Changed.Equal(*b.now) || !view.Started.Before(view.Changed) {
			t.Errorf("%s: the recorded cause and time: %q %s", name, view.Contradiction, view.Changed)
		}
		b.no(name+", then resumed", b.r.Resume("w1", id, 1), ErrRejected)
		b.no(name+", the same report again", second(c.report(id)), ErrRejected)
	}
	plain := files("new-part", 5)
	plain.Subject.ScopeID = "new-part"
	b.ok("start", b.r.Start(plain))
	b.r.DeclareManifest("w1", "new-part", SourceListed, three)
	b.ok("lost", b.r.AccessLost("w1", "new-part", 1, PermissionWithdrawn))
	b.no("a part that was not verified yet, while interrupted", second(b.r.ReceivePart("w1", "new-part", three[0])), ErrRejected)
	b.ok("leaving a collection whose part never matches", b.r.Abandon("w1", "new-part", PartUnverifiable))
	paged := pages("expired", 5)
	paged.Subject.ScopeID = "expired"
	b.ok("start", b.r.Start(paged))
	b.r.ReceivePage("w1", "expired", "c0", "c1", 10, sum(1))
	b.ok("lost", b.r.AccessLost("w1", "expired", 1, CredentialExpired))
	b.ok("leaving a collection whose cursor expired while it was interrupted", b.r.Abandon("w1", "expired", CursorExpired))
	if view := b.view("expired"); view.State != Abandoned || view.AbandonReason != CursorExpired {
		t.Errorf("abandoned: %+v", view)
	}
	big := files("big", 5)
	big.Subject.ScopeID = "big"
	b.ok("start", b.r.Start(big))
	b.r.DeclareManifest("w1", "big", SourceListed, three)
	b.state("a second manifest longer than the limit", first(b.r.DeclareManifest("w1", "big", SourceListed, parts(4))), nil, Inconsistent)
	for index, final := range []string{"big", "expired"} {
		b.no("a change at the collection's own revision on a final collection", b.r.SourceChanged("w1", final, 5), ErrRejected)
		b.ok("a changed source reported through a final collection", b.r.SourceChanged("w1", final, uint64(6+index)))
	}
	sales := verified(FetchCosts, DistributorSales)
	b.ok("declare", second(b.r.Declare(sales)))
	third := files("sales", 2)
	third.Subject.Kind = DistributorSales
	b.filled(third, 1)
	b.gaps("sales")
	b.filled(files("orig", 4), 2)
	b.gaps("orig")
	if got := b.standings(third.Subject); got != "sales @1 known 2" {
		t.Errorf("a distributor's sales data is a subject of its own: %s", got)
	}
	b.ok("retired", b.r.Retire("w1", "sales", RetireByOperator))
	if missing, err := b.r.Complete("w1", "sales"); missing != nil || err != nil {
		t.Errorf("completing a retired collection again: %v %v", missing, err)
	}
	b.ok("a changed source reported through a retired collection", b.r.SourceChanged("w1", "sales", 3))
	if view, _ := b.r.Effective(third.Subject); view.Known != 3 || view.Collection != nil || view.Others[0].Origin != SourceListed || !view.Moved.Equal(*b.now) {
		t.Errorf("after the retirement: %+v", view)
	}
	waiting := files("waiting", 9)
	waiting.Subject.ScopeID = "waiting"
	b.ok("start", b.r.Start(waiting))
	b.ok("a wait exactly at the bound", b.r.Throttled("w1", "waiting", time.Hour))
}

func (b bench) pass(duration time.Duration) { *b.now = b.now.Add(duration) }

func TestWhatCounts(t *testing.T) {
	b := open(t, Limits{Capabilities: 12, Collections: 40, Parts: 6, Pages: 6, MaxBytes: 1 << 40, MaxRows: 1 << 30, MaxWait: time.Hour})
	about := subjectOf("w1")
	if _, err := b.r.Effective(about); !errors.Is(err, ErrNotFound) {
		t.Errorf("a subject nobody started: %v", err)
	}
	b.filled(files("a", 5), 3)
	if got := b.standings(about); got != "none @0 known 5" {
		t.Errorf("before any completion: %s", got)
	}
	b.gaps("a")
	b.filled(files("b", 5), 3)
	b.gaps("b")
	b.ok("start", b.r.Start(files("c", 5)))
	b.ok("manifest", second(b.r.DeclareManifest("w1", "c", SourceListed, parts(2))))
	for _, given := range parts(2) {
		b.r.ReceivePart("w1", "c", given)
	}
	b.gaps("c")
	if got := b.standings(about); got != "a b=duplicate c=conflicting @1 known 5" {
		t.Errorf("three completions of one revision: %s", got)
	}
	b.ok("start", b.r.Start(files("d", 6)))
	b.ok("an empty manifest listed too early", second(b.r.DeclareManifest("w1", "d", AdapterDerived, nil)))
	b.gaps("d")
	if got := b.standings(about); got != "a/stale b=duplicate c=conflicting d=newer @1 known 6" {
		t.Errorf("an empty higher revision does not replace what counts: %s", got)
	}
	if view, _ := b.r.Effective(about); !view.Stale || view.Others[2].Empty != true || view.Collection.Totals.Rows != 60 || view.Present.Answer != Allowed || view.Present.Revision != view.Collection.Capability {
		t.Errorf("the effective view: %+v", view)
	}
	b.no("an adoption with a pointer number that is not the present one", b.r.Adopt("w1", "c", 0), ErrConflict)
	b.ok("adopting the conflicting one", b.r.Adopt("w1", "c", 1))
	b.ok("adopting the first again on purpose", b.r.Adopt("w1", "a", 2))
	b.no("a delayed copy of the first adoption", b.r.Adopt("w1", "c", 1), ErrConflict)
	if got := b.standings(about); got != "a/stale b=duplicate c=conflicting d=newer @3 known 6" {
		t.Errorf("after the delayed adoption: %s", got)
	}
	b.ok("adopting the effective one", b.r.Adopt("w1", "a", 3))
	b.ok("adopting the newer, empty one", b.r.Adopt("w1", "d", 3))
	if got := b.standings(about); got != "d/empty a=older b=older c=older @4 known 6" {
		t.Errorf("after adopting the empty revision: %s", got)
	}
	b.no("adopting an older one over it", b.r.Adopt("w1", "a", 4), ErrRejected)
	b.no("an unknown reason", b.r.Retire("w1", "d", "mistake"), ErrInvalid)
	b.ok("retiring the mistaken adoption", b.r.Retire("w1", "d", WrongSource))
	b.ok("the same again", b.r.Retire("w1", "d", WrongSource))
	b.no("with another reason", b.r.Retire("w1", "d", RetireConflict), ErrConflict)
	if got := b.standings(about); got != "none a=candidate b=candidate c=candidate d=retired @5 known 6" {
		t.Errorf("with nothing effective: %s", got)
	}
	if view, _ := b.r.Effective(about); view.Others[3].RetireReason != WrongSource || view.Stale {
		t.Errorf("the retired collection in the list: %+v", view.Others)
	}
	b.no("adopting a retired collection", b.r.Adopt("w1", "d", 5), ErrRejected)
	b.filled(files("e", 6), 1)
	b.gaps("e")
	if got := b.standings(about); got != "none a=candidate b=candidate c=candidate d=retired e=candidate @5 known 6" {
		t.Errorf("a completion after a retirement waits like any other: %s", got)
	}
	b.ok("adopting the older real data again", b.r.Adopt("w1", "a", 5))
	b.ok("retiring a collection that is not effective", b.r.Retire("w1", "c", RetireConflict))
	if got := b.standings(about); got != "a/stale b=duplicate c=retired d=retired e=newer @6 known 6" {
		t.Errorf("the end: %s", got)
	}
	b.no("retiring a collection that never completed", b.r.Retire("w1", "x", RetireByOperator), ErrNotFound)
	b.ok("start", b.r.Start(files("f", 6)))
	b.no("retiring a running collection", b.r.Retire("w1", "f", RetireByOperator), ErrRejected)
	b.no("adopting a running collection", b.r.Adopt("w1", "f", 6), ErrRejected)
	withdrawn := verified(FetchCosts, ProviderOriginal)
	withdrawn.State, withdrawn.Evidence, withdrawn.Reason = Unsupported, "", "parser-wrong"
	b.r.Declare(withdrawn)
	if view, _ := b.r.Effective(about); view.Present.Answer != AnswerUnsupported || view.Present.Revision != 2 || view.Collection.Capability != 1 || view.Collection.Spec.ID != "a" {
		t.Errorf("a capability withdrawn after completion is not visible: %+v", view.Present)
	}

	invoices := files("i", 3)
	invoices.Subject.Operation = FetchInvoices
	b.filled(invoices, 2)
	b.gaps("i")
	forma := files("p", 9)
	forma.Subject.Kind = ProForma
	b.filled(forma, 1)
	b.gaps("p")
	if got := b.standings(invoices.Subject); got != "i @1 known 3" {
		t.Errorf("invoices are a subject of their own: %s", got)
	}
	if got := b.standings(forma.Subject); got != "p @1 known 9" {
		t.Errorf("a pro forma is a subject of its own: %s", got)
	}
	if got := b.standings(about); got != "a/stale b=duplicate c=retired d=retired e=newer @6 known 6" {
		t.Errorf("another operation or kind changed the costs' subject: %s", got)
	}
	bad := about
	bad.ScopeID = "payer one"
	if _, err := b.r.Effective(bad); !errors.Is(err, ErrInvalid) {
		t.Errorf("a malformed subject: %v", err)
	}
	foreign := about
	foreign.WorkspaceID = "w2"
	if _, err := b.r.Effective(foreign); !errors.Is(err, ErrNotFound) {
		t.Errorf("the same subject in another workspace: %v", err)
	}
	twoPaged := func(id string, split int64) {
		spec := pages(id, 1)
		spec.Subject.ScopeID = "paged"
		b.ok("start", b.r.Start(spec))
		b.r.ReceivePage("w1", id, "c0", "c1", split, sum(int(split)))
		b.r.ReceivePage("w1", id, "c1", "", 30-split, sum(int(100-split)))
		b.gaps(id)
	}
	b.r.Declare(verified(FetchCosts, ProviderOriginal))
	twoPaged("q1", 10)
	twoPaged("q2", 10)
	twoPaged("q3", 15)
	paged := about
	paged.ScopeID = "paged"
	if got := b.standings(paged); got != "q1 q2=duplicate q3=conflicting @1 known 1" {
		t.Errorf("the same chain twice and another pagination: %s", got)
	}
	sums := about
	sums.ScopeID = "sums"
	for index, id := range []string{"s1", "s2"} {
		spec := pages(id, 1)
		spec.Subject = sums
		b.ok("start", b.r.Start(spec))
		b.r.ReceivePage("w1", id, "c0", "", 30, sum(50+index))
		b.gaps(id)
	}
	if got := b.standings(sums); got != "s1 s2=conflicting @1 known 1" {
		t.Errorf("two chains that differ only in a checksum: %s", got)
	}
	bytes := about
	bytes.ScopeID = "bytes"
	for index, id := range []string{"f1", "f2"} {
		spec := files(id, 1)
		spec.Subject = bytes
		b.ok("start", b.r.Start(spec))
		list := parts(1)
		list[0].Checksum = sum(70 + index)
		b.r.DeclareManifest("w1", id, SourceListed, list)
		b.r.ReceivePart("w1", id, list[0])
		b.gaps(id)
	}
	if got := b.standings(bytes); got != "f1 f2=conflicting @1 known 1" {
		t.Errorf("two manifests that differ only in a checksum: %s", got)
	}
}

func TestTheOtherWorkspaceSeesNothing(t *testing.T) {
	b := open(t, testLimits)
	theirs := files("a", 5)
	theirs.Subject.WorkspaceID = "w2"
	b.ok("their start", b.r.Start(theirs))
	b.ok("a collection of our own, so that our workspace exists", b.r.Start(files("ours", 5)))
	b.ok("their manifest", second(b.r.DeclareManifest("w2", "a", SourceListed, parts(1))))
	before, _ := b.r.Collection("w2", "a")
	for name, err := range map[string]error{
		"manifest": second(b.r.DeclareManifest("w1", "a", SourceListed, parts(1))), "part": second(b.r.ReceivePart("w1", "a", parts(1)[0])),
		"page": second(b.r.ReceivePage("w1", "a", "c0", "", 1, sum(1))), "total": second(b.r.DeclareTotal("w1", "a", 1, 1)),
		"throttled": b.r.Throttled("w1", "a", time.Second), "lost": b.r.AccessLost("w1", "a", 1, PermissionWithdrawn), "resume": b.r.Resume("w1", "a", 1),
		"changed": b.r.SourceChanged("w1", "a", 9), "abandon": b.r.Abandon("w1", "a", ByOperator), "complete": second(b.r.Complete("w1", "a")),
		"adopt": b.r.Adopt("w1", "a", 0), "retire": b.r.Retire("w1", "a", RetireByOperator), "read": second(b.r.Collection("w1", "a")),
	} {
		b.no(name+" on the other workspace's collection", err, ErrNotFound)
	}
	if _, err := b.r.Collection("w1", "never"); !errors.Is(err, ErrNotFound) {
		t.Errorf("a collection that does not exist: %v", err)
	}
	after, _ := b.r.Collection("w2", "a")
	if fmt.Sprintf("%+v", before) != fmt.Sprintf("%+v", after) {
		t.Errorf("operations of one workspace changed another's collection")
	}
	b.ok("our own collection of the same identifier", b.r.Start(files("a", 9)))
	if view, _ := b.r.Collection("w2", "a"); view.State != Collecting {
		t.Errorf("a higher revision in one workspace superseded another's collection")
	}
	for name, err := range map[string]error{
		"no workspace": second(b.r.Collection("", "a")), "a collection that is text": second(b.r.Collection("w1", "a b")), "complete": second(b.r.Complete("w 1", "a")),
	} {
		b.no(name, err, ErrInvalid)
	}
}

func TestEachPartAndPageCountsOnceUnderConcurrency(t *testing.T) {
	b := open(t, Limits{Capabilities: 12, Collections: 12, Parts: 40, Pages: 40, MaxBytes: 1 << 40, MaxRows: 1 << 30, MaxWait: time.Hour})
	b.ok("start", b.r.Start(files("f", 1)))
	b.ok("manifest", second(b.r.DeclareManifest("w1", "f", SourceListed, parts(40))))
	paged := pages("p", 1)
	paged.Subject.ScopeID = "paged"
	b.ok("start", b.r.Start(paged))
	var group sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for _, given := range parts(40) {
				b.r.ReceivePart("w1", "f", given)
			}
			for page := 0; page < 40; page++ {
				next := fmt.Sprintf("c%d", page+1)
				if page == 39 {
					next = ""
				}
				for attempt := 0; attempt < 200; attempt++ {
					if _, err := b.r.ReceivePage("w1", "p", fmt.Sprintf("c%d", page), next, int64(page+1), sum(page+1)); err == nil {
						break
					}
				}
			}
			b.r.Complete("w1", "f")
			b.r.Complete("w1", "p")
		}()
	}
	group.Wait()
	if view := b.view("f"); view.State != Completed || view.Totals != (Totals{Units: 40, Bytes: 820000, Rows: 8200}) {
		t.Errorf("files: %s %+v", view.State, view.Totals)
	}
	if view := b.view("p"); view.State != Completed || view.Totals != (Totals{Units: 40, Rows: 820}) {
		t.Errorf("pages: %s %+v", view.State, view.Totals)
	}
}

func TestARegistryThatIsNotBuilt(t *testing.T) {
	for name, limits := range map[string]Limits{"capabilities": {0, 1, 1, 1, 1, 1, 1}, "collections": {1, 0, 1, 1, 1, 1, 1}, "parts": {1, 1, 0, 1, 1, 1, 1}, "pages": {1, 1, 1, 0, 1, 1, 1},
		"bytes": {1, 1, 1, 1, 0, 1, 1}, "rows": {1, 1, 1, 1, 1, 0, 1}, "wait": {1, 1, 1, 1, 1, 1, 0},
		"bytes that could overflow a total": {1, 1, 4, 1, 1 << 62, 1, 1}, "rows that could overflow a total": {1, 1, 1, 4, 1, 1 << 62, 1}} {
		if _, err := NewRegistry(limits, time.Now); !errors.Is(err, ErrInvalid) {
			t.Errorf("limits without %s: %v", name, err)
		}
	}
	if _, err := NewRegistry(testLimits, nil); !errors.Is(err, ErrInvalid) {
		t.Errorf("no clock: %v", err)
	}
	var none *Registry
	for _, r := range []*Registry{none, {}} {
		for name, err := range map[string]error{
			"declare": second(r.Declare(verified(FetchCosts, ProviderOriginal))), "permit": second(r.Permit("aws", "v1", FetchCosts, ProviderOriginal, usEast)), "start": r.Start(files("a", 1)),
			"manifest": second(r.DeclareManifest("w1", "a", SourceListed, nil)), "part": second(r.ReceivePart("w1", "a", parts(1)[0])), "total": second(r.DeclareTotal("w1", "a", 1, 1)),
			"page": second(r.ReceivePage("w1", "a", "c0", "", 1, sum(1))), "throttled": r.Throttled("w1", "a", 0), "lost": r.AccessLost("w1", "a", 1, PermissionWithdrawn),
			"resume": r.Resume("w1", "a", 1), "changed": r.SourceChanged("w1", "a", 2), "abandon": r.Abandon("w1", "a", ByOperator), "complete": second(r.Complete("w1", "a")),
			"adopt": r.Adopt("w1", "a", 0), "retire": r.Retire("w1", "a", RetireByOperator), "collection": second(r.Collection("w1", "a")), "effective": second(r.Effective(subjectOf("w1"))),
		} {
			if !errors.Is(err, ErrInvalid) {
				t.Errorf("%s on a registry that is not built: %v", name, err)
			}
		}
	}
	for text, valid := range map[string]bool{"0123abcd": true, strings.Repeat("a", 128): true, strings.Repeat("a", 129): false, "0123abc": false, "0123ABCD": false, "0123abcg": false, "": false} {
		if validChecksum(text) != valid {
			t.Errorf("checksum %q", text)
		}
	}
	for id, valid := range map[string]bool{"a": true, "A.b_c:d-9": true, strings.Repeat("x", 128): true, strings.Repeat("x", 129): false, "": false, "a b": false, "a/b": false} {
		if validID(id) != valid {
			t.Errorf("identifier %q", id)
		}
	}
}
