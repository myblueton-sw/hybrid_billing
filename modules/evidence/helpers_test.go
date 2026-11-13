package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"
)

var start = time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)

const day = 24 * time.Hour

type testClock struct{ at time.Time }

func (c *testClock) read() time.Time { return c.at }

func (c *testClock) advance(d time.Duration) { c.at = c.at.Add(d) }

func testLimits() Limits {
	return Limits{Artifacts: 500, Bundles: 100, Holds: 50, Gaps: 50, Instances: 6, Items: 40, HoldBundles: 10, Restricted: 10, Page: 5, Listed: 3}
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
	r, err := NewRegistry(limits, clock.read)
	if err != nil {
		t.Fatal(err)
	}
	return &harness{t: t, r: r, clock: clock}
}

func digestOf(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])
}

func product(id string) Custody {
	return Custody{Kind: Product, Location: "store:" + id, Receipt: "receipt:" + id}
}

func remote(id string, until time.Time) Custody {
	c := Custody{Kind: Remote, Location: "bucket:" + id}
	if !until.IsZero() {
		c.VerifiedUntil, c.Evidence = until, "check:"+id
	}
	return c
}

func art(id string, kind Kind, custody Custody) ArtifactSpec {
	return ArtifactSpec{WorkspaceID: "ws1", ID: id, Kind: kind, Version: "v1", Digest: digestOf("bytes", id), Size: 10, Custody: custody}
}

func (h *harness) register(spec ArtifactSpec) {
	h.t.Helper()
	if state, err := h.r.Register(spec); err != nil || state != Registered {
		h.t.Fatalf("register %s: %v %v", spec.ID, state, err)
	}
}

var roles = map[string]Kind{"src": SourceOriginal, "man": InputManifest, "map": Mapping, "att": Attribution, "pol": Policy, "fx": FX, "eng": Engine, "mod": Module, "res": Result, "doc": Document}

func (h *harness) set(prefix string, names ...string) []Item {
	h.t.Helper()
	if len(names) == 0 {
		names = []string{"src", "man", "map", "att", "pol", "eng", "res"}
	}
	var items []Item
	for _, name := range names {
		id := prefix + "-" + name
		if h.r.artifact("ws1", id) == nil {
			h.register(art(id, roles[strings.SplitN(name, ".", 2)[0]], product(id)))
		}
		items = append(items, Item{Role: roles[strings.SplitN(name, ".", 2)[0]], Artifact: id})
	}
	return items
}

func bundleSpec(id string, items []Item) BundleSpec {
	return BundleSpec{WorkspaceID: "ws1", ID: id, Customer: "cust-1", Period: "2026-10", RetainUntil: start.Add(365 * day), Items: items}
}

func (h *harness) confirm(spec BundleSpec) Confirmation {
	h.t.Helper()
	out, err := h.r.Confirm(spec)
	if err != nil || out.State != Confirmed {
		h.t.Fatalf("confirm %s: %+v %v", spec.ID, out, err)
	}
	return out
}

func (h *harness) inspect(id string) Inspection {
	h.t.Helper()
	out, err := h.r.Inspect("ws1", id)
	if err != nil {
		h.t.Fatal(err)
	}
	return out
}

func brief(out Inspection) string {
	var broken, risk []string
	for _, item := range out.Broken {
		broken = append(broken, fmt.Sprintf("%s:%s:%s", item.Artifact, item.Cause, item.Reason))
	}
	for _, item := range out.AtRisk {
		risk = append(risk, item.Artifact)
	}
	return fmt.Sprintf("%s%v/%d risk%v/%d %s protected=%v", out.Verdict, broken, out.BrokenTotal, risk, out.AtRiskTotal, out.Rerun, out.Protected)
}

func codes(reasons []Reason) string {
	parts := make([]string, len(reasons))
	for index, reason := range reasons {
		parts[index] = strings.Trim(fmt.Sprintf("%s:%s:%s", reason.Code, reason.Role, reason.Artifact), ":")
	}
	return strings.Join(parts, " ")
}

func (r *Registry) dump() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var lines []string
	for location, held := range r.locations {
		lines = append(lines, fmt.Sprintf("location %s %v", location, held))
	}
	for key, reason := range r.restricted {
		lines = append(lines, "restricted "+key+" "+reason)
	}
	for id, ws := range r.workspaces {
		for name, a := range ws.artifacts {
			line := fmt.Sprintf("artifact %s/%s %s %s %s bundles=%d %s %s", id, name, a.print, a.state, a.reason, len(a.bundles), a.registered.Format(time.RFC3339Nano), a.changed.Format(time.RFC3339Nano))
			for _, held := range a.instances {
				line += fmt.Sprintf(" [%+v %v %s %d]", held.custody, held.lost, held.reason, held.order)
			}
			lines = append(lines, line)
		}
		for name, b := range ws.bundles {
			var holds []string
			for hold := range b.holds {
				holds = append(holds, hold)
			}
			sort.Strings(holds)
			lines = append(lines, fmt.Sprintf("bundle %s/%s %s %s %s %s %v %s %v %v items=%d/%d %s %s", id, name, b.print, b.state, b.reason, b.retained.Format(time.RFC3339Nano), b.counts, b.digest, b.gaps, holds, len(b.entries), len(b.spec.Items),
				b.confirmed.Format(time.RFC3339Nano), b.changed.Format(time.RFC3339Nano)))
		}
		for name, held := range ws.holds {
			lines = append(lines, fmt.Sprintf("hold %s/%s %s %s %s", id, name, held.print, held.state, held.reason))
		}
		for name, held := range ws.gaps {
			lines = append(lines, fmt.Sprintf("gap %s/%s %s %s %s", id, name, held.print, held.state, held.artifact))
		}
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

func (h *harness) unchanged(before, what string) {
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
