package intake

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

var start = time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)

type testClock struct{ at time.Time }

func (c *testClock) read() time.Time { return c.at }

func (c *testClock) advance(d time.Duration) { c.at = c.at.Add(d) }

func testProfile(t testing.TB) decimal.Profile {
	t.Helper()
	made, err := decimal.NewProfile(decimal.ProfileSpec{ID: "synthetic", Revision: "1", InputIntegerDigits: 12, InputScale: 4, ResultIntegerDigits: 14, ResultScale: 4, IntermediateDigits: 40})
	if err != nil {
		t.Fatal(err)
	}
	return made
}

func testLimits() Limits {
	return Limits{MaxSize: 1000, MaxRatio: 10, Chunks: 200, ChunkRecords: 1000, Artifacts: 1000, Records: 1000000, Manifests: 50, Page: 100, Listed: 3}
}

type harness struct {
	t     testing.TB
	r     *Registry
	clock *testClock
}

func newHarness(t testing.TB, change func(*Limits)) *harness {
	t.Helper()
	limits := testLimits()
	if change != nil {
		change(&limits)
	}
	clock := &testClock{at: start}
	r, err := NewRegistry(limits, testProfile(t), clock.read)
	if err != nil {
		t.Fatal(err)
	}
	return &harness{t: t, r: r, clock: clock}
}

var october = Subject{WorkspaceID: "ws1", Provider: "aws", Scope: "payer-1", Period: "2026-10"}
var november = Subject{WorkspaceID: "ws1", Provider: "aws", Scope: "payer-1", Period: "2026-11"}

func digestOf(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])
}

func rec(key, amount, currency string) Record {
	return Record{Key: key, Digest: digestOf(key, amount, currency)[:32], Amount: amount, Currency: currency}
}

func scaled(t testing.TB, text string) int64 {
	t.Helper()
	negative := strings.HasPrefix(text, "-")
	text = strings.TrimPrefix(text, "-")
	whole, fraction, _ := strings.Cut(text, ".")
	for len(fraction) < 4 {
		fraction += "0"
	}
	var value int64
	for _, c := range whole + fraction {
		if c < '0' || c > '9' {
			t.Fatalf("test amount %q", text)
		}
		value = value*10 + int64(c-'0')
	}
	if negative {
		return -value
	}
	return value
}

func unscaled(value int64) string {
	sign := ""
	if value < 0 {
		sign = "-"
		value = -value
	}
	text := fmt.Sprintf("%d.%04d", value/10000, value%10000)
	text = strings.TrimRight(text, "0")
	text = strings.TrimSuffix(text, ".")
	return sign + text
}

func totals(t testing.TB, chunks ...[]Record) (int, map[string]string) {
	t.Helper()
	sums := map[string]int64{}
	rows := 0
	for _, chunk := range chunks {
		for _, record := range chunk {
			sums[record.Currency] += scaled(t, record.Amount)
			rows++
		}
	}
	out := map[string]string{}
	for currency, sum := range sums {
		out[currency] = unscaled(sum)
	}
	return rows, out
}

func base(id, document string, revision uint64) Spec {
	return Spec{ID: id, Subject: october, Document: document, Kind: EventKeys, Revision: revision, Bytes: digestOf("bytes", id), Role: Original, Holds: Empty,
		Size: 100, Expanded: 200, Location: "store:" + id, Scan: Passed, Syntax: Passed, Bodies: Passed}
}

func original(t testing.TB, id, document string, revision uint64, chunks ...[]Record) Spec {
	s := base(id, document, revision)
	s.Holds = Rows
	s.Chunks = len(chunks)
	s.Rows, s.Sums = totals(t, chunks...)
	s.Totals = SourceStated
	return s
}

func scan(id, document string, revision uint64) Spec {
	s := base(id, document, revision)
	s.Holds = AwaitingExtraction
	return s
}

func derived(t testing.TB, id, document string, revision uint64, from string, chunks ...[]Record) Spec {
	s := original(t, id, document, revision, chunks...)
	s.Role = Derived
	s.Holds = ""
	s.Original = from
	s.Method = "ocr:v1"
	s.Totals = CallerComputed
	return s
}

func (h *harness) register(s Spec) State {
	h.t.Helper()
	state, _, err := h.r.Register(s)
	if err != nil {
		h.t.Fatalf("register %s: %v", s.ID, err)
	}
	return state
}

func (h *harness) stage(s Spec, chunks ...[]Record) {
	h.t.Helper()
	for index, chunk := range chunks {
		if state, err := h.r.Stage(s.Subject.WorkspaceID, s.ID, index+1, chunk); err != nil || state != Staging {
			h.t.Fatalf("stage %s chunk %d: %v %v", s.ID, index+1, state, err)
		}
	}
}

func (h *harness) seal(s Spec, chunks ...[]Record) {
	h.t.Helper()
	if state := h.register(s); state != Staging {
		h.t.Fatalf("register %s: %v", s.ID, state)
	}
	h.stage(s, chunks...)
	if s.Role == Derived {
		staged, err := h.r.Staged(s.Subject.WorkspaceID, s.ID)
		if err != nil {
			h.t.Fatal(err)
		}
		if err := h.r.Verify(s.Subject.WorkspaceID, s.ID, staged.Digest, "verifier:1", "evidence:1"); err != nil {
			h.t.Fatal(err)
		}
	}
	if state, gaps, err := h.r.Seal(s.Subject.WorkspaceID, s.ID); err != nil || state != Sealed {
		h.t.Fatalf("seal %s: %v %v %v", s.ID, state, gaps, err)
	}
}

func (h *harness) sealRows(id, document string, revision uint64, chunks ...[]Record) Spec {
	h.t.Helper()
	s := original(h.t, id, document, revision, chunks...)
	h.seal(s, chunks...)
	return s
}

func (h *harness) publish(at Subject, expected uint64) Published {
	h.t.Helper()
	out, err := h.r.Publish(at, expected)
	if err != nil {
		h.t.Fatalf("publish: %v %+v", err, out)
	}
	return out
}

func (h *harness) view(id string) ArtifactView {
	h.t.Helper()
	view, err := h.r.Artifact("ws1", id)
	if err != nil {
		h.t.Fatal(err)
	}
	return view
}

func (h *harness) status(at Subject) StatusView {
	h.t.Helper()
	view, err := h.r.Status(at)
	if err != nil {
		h.t.Fatal(err)
	}
	return view
}

func (h *harness) all(at Subject, number uint64) []ManifestRecord {
	h.t.Helper()
	var out []ManifestRecord
	for {
		page, err := h.r.Records(at, number, len(out), h.r.limits.Page)
		if err != nil {
			h.t.Fatal(err)
		}
		if len(page) == 0 {
			return out
		}
		out = append(out, page...)
	}
}

func brief(records []ManifestRecord) string {
	parts := make([]string, len(records))
	for index, record := range records {
		parts[index] = fmt.Sprintf("%s/%s/%s=%s%s@%s", record.Kind, record.Document, record.Key, record.Amount, record.Currency, record.Artifact)
	}
	return strings.Join(parts, " ")
}

func (r *Registry) dump() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out bytes.Buffer
	spaces := make([]string, 0, len(r.workspaces))
	for id := range r.workspaces {
		spaces = append(spaces, id)
	}
	sort.Strings(spaces)
	for _, id := range spaces {
		ws := r.workspaces[id]
		fmt.Fprintf(&out, "ws %s records=%d sequence=%d\n", id, ws.records, ws.sequence)
		ids := make([]string, 0, len(ws.artifacts))
		for artifactID := range ws.artifacts {
			ids = append(ids, artifactID)
		}
		sort.Strings(ids)
		for _, artifactID := range ids {
			a := ws.artifacts[artifactID]
			chunks := make([]string, 0, len(a.chunks))
			for number, sum := range a.chunks {
				chunks = append(chunks, fmt.Sprintf("%06d:%s", number, sum))
			}
			sort.Strings(chunks)
			rows := "dropped"
			if a.rows != nil {
				rows = recordDigest(a.rows)
			}
			fmt.Fprintf(&out, " a %s %q %s %v %s %s chunks=%v rows=%s count=%d held=%d sums=%v digest=%s seq=%d ver=%v contra=%d %s %s\n", artifactID, a.print, a.state, a.reasons, a.abandon, a.retire,
				chunks, rows, a.count, a.held, texts(a.sums), a.digest, a.sequence, a.verified, a.contradiction, a.registered.Format(time.RFC3339Nano), a.changed.Format(time.RFC3339Nano))
		}
		documents := make([]string, 0, len(ws.documents))
		for documentID := range ws.documents {
			documents = append(documents, documentID)
		}
		sort.Strings(documents)
		for _, documentID := range documents {
			d := ws.documents[documentID]
			fmt.Fprintf(&out, " d %s %v %s highest=%d n=%d\n", documentID, d.subject, d.kind, d.highest, len(d.artifacts))
		}
		subjects := make([]string, 0, len(ws.subjects))
		for at, sub := range ws.subjects {
			line := fmt.Sprintf(" s %v docs=%d", at, len(sub.documents))
			for _, m := range sub.manifests {
				line += fmt.Sprintf(" m%d:%s:%s:%d:%v:%v:%v:%d:%s", m.view.Number, m.view.Content, m.view.Attribution, m.view.Records, m.sums, m.view.Behind, m.view.SameBytes, m.view.Staging, m.view.At.Format(time.RFC3339Nano))
			}
			subjects = append(subjects, line)
		}
		sort.Strings(subjects)
		out.WriteString(strings.Join(subjects, "\n") + "\n")
	}
	return out.String()
}

func (h *harness) unchanged(before string, what string) {
	h.t.Helper()
	if after := h.r.dump(); after != before {
		h.t.Fatalf("%s wrote:\n%s\n---\n%s", what, before, after)
	}
}

func wantErr(t testing.TB, err, want error, what string) {
	t.Helper()
	if !errors.Is(err, want) {
		t.Fatalf("%s: got %v, want %v", what, err, want)
	}
}

func secondEncoding(fields ...string) []byte {
	var out []byte
	for _, field := range fields {
		size := len(field)
		out = append(out, 0, 0, 0, 0, byte(size>>24), byte(size>>16), byte(size>>8), byte(size))
		out = append(out, field...)
	}
	return out
}
