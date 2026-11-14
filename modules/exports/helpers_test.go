package exports

import (
	"archive/zip"

	"bytes"
	"errors"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
	"io"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
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

var start = time.Date(2026, 11, 15, 9, 0, 0, 0, time.UTC)

func human(id string) Actor { return Actor{ID: id, Kind: KindHuman} }

func assistant(id string) Actor { return Actor{ID: id, Kind: KindAssistant} }

func system(id string) Actor { return Actor{ID: id, Kind: KindSystem} }

func kind(err error) string {
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
	case errors.Is(err, ErrRejected):
		return "rejected"
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
	if got := kind(err); got != want {
		t.Fatalf("%s: got %s, want %s", label, got, want)
	}
}

func newRegistry(t *testing.T, generator string) (*Registry, *testClock) {
	t.Helper()
	clock := &testClock{now: start}
	r, err := NewRegistry(Config{Generator: generator, Schema: "settlement-1", Numbers: numbers(t)}, Limits{Workspaces: 3, Artifacts: 6, Lines: 6}, clock.read)
	if err != nil {
		t.Fatal(err)
	}
	return r, clock
}

func numbers(t testing.TB) decimal.Profile {
	t.Helper()
	made, err := decimal.NewProfile(decimal.ProfileSpec{ID: "exports-test", Revision: "1", InputIntegerDigits: 18, InputScale: 8, ResultIntegerDigits: 20, ResultScale: 8, IntermediateDigits: 80})
	if err != nil {
		t.Fatal(err)
	}
	return made
}

const approval = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func spec(lines ...Line) Spec {
	return Spec{Connection: "erp-file", Submission: "sub-1", Approval: approval, Customer: "acme", Period: "2026-10", Currency: "KRW", Scale: 2, Lines: lines}
}

func line(row, net string) Line {
	return Line{Row: row, Source: "usage:" + row, Product: "Compute", TaxCode: "VAT10", Quantity: "1", Net: net, Tax: "0"}
}

func settlement(id string, lines ...Line) Request {
	return Request{ID: id, Kind: KindSettlement, TimeZone: "+09:00", Spec: spec(lines...)}
}

func parts(t *testing.T, data []byte) map[string]string {
	t.Helper()
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, f := range z.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(rc)
		rc.Close()
		out[f.Name] = string(b)
	}
	return out
}

func names(m map[string]string) string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return strings.Join(out, ",")
}

func file(t *testing.T, a Artifact, suffix string) []byte {
	t.Helper()
	for _, f := range a.Files {
		if strings.HasSuffix(f.Name, suffix) {
			return f.Bytes
		}
	}
	t.Fatalf("no file %s", suffix)
	return nil
}
