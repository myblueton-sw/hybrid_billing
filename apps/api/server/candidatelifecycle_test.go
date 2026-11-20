package server

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/contracts"
)

func lifecycleTestRegistry(t *testing.T, events []contracts.Event) *contracts.Registry {
	t.Helper()
	r, err := contracts.NewRegistry(contracts.Limits{Contracts: 10, Writes: 32, Reserve: 2, Batch: 12}, func() time.Time { return time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatal(err)
	}
	if len(events) > 0 {
		if _, err := r.Amend("workspace", "contract", 0, "write-1", nil, events, ""); err != nil {
			t.Fatal(err)
		}
	}
	return r
}

func TestCandidateLifecycleWholeInterval(t *testing.T) {
	event := func(kind contracts.Kind, date string, revision uint64) contracts.Event {
		d, err := time.Parse(time.DateOnly, date)
		if err != nil {
			t.Fatal(err)
		}
		day, err := contracts.NewDay(d.Year(), int(d.Month()), d.Day())
		if err != nil {
			t.Fatal(err)
		}
		return contracts.Event{ID: fmt.Sprint(kind, "-", date), Kind: kind, From: day, Revision: revision}
	}
	start := event(contracts.Start, "2026-09-01", 1)
	for _, tc := range []struct {
		name   string
		events []contracts.Event
		want   bool
	}{
		{"whole", []contracts.Event{start}, true},
		{"exact-start", []contracts.Event{event(contracts.Start, "2026-10-01", 1)}, true},
		{"exact-end", []contracts.Event{start, event(contracts.End, "2026-11-01", 0)}, true},
		{"late-start", []contracts.Event{event(contracts.Start, "2026-10-02", 1)}, false},
		{"early-end", []contracts.Event{start, event(contracts.End, "2026-10-31", 0)}, false},
		{"suspend", []contracts.Event{start, event(contracts.Suspend, "2026-10-15", 0)}, false},
		{"gap-resumed", []contracts.Event{start, event(contracts.Suspend, "2026-10-15", 0), event(contracts.Resume, "2026-10-16", 0)}, false},
		{"resumed-before", []contracts.Event{start, event(contracts.Suspend, "2026-09-15", 0), event(contracts.Resume, "2026-09-30", 0)}, true},
		{"revision", []contracts.Event{start, event(contracts.Change, "2026-10-15", 2)}, false},
		{"wrong-revision", []contracts.Event{event(contracts.Start, "2026-09-01", 2)}, false},
		{"renew", []contracts.Event{start, event(contracts.Renew, "2026-10-15", 0)}, true},
		{"missing", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := &normalizedCharges{lifecycles: lifecycleTestRegistry(t, tc.events)}
			b := normalizedBinding{Contract: "contract", ContractRevision: 1, From: "2026-10-01", To: "2026-10-31"}
			out, err := p.lifecycleCoverage("workspace", b)
			if (err == nil) != tc.want {
				t.Fatalf("coverage: %v %v", out, err)
			}
			if !tc.want {
				return
			}
			seq, err := p.lifecycles.Sequence("workspace", "contract")
			if err != nil || out.Sequence != seq || out.Contract != "contract" || out.From != "2026-10-01" || out.To != "2026-11-01" || len(out.Segments) != 1 || out.Segments[0].Revision != 1 || out.Segments[0].State != "active" {
				t.Fatalf("provenance: %+v %v", out, err)
			}
			before, _ := json.Marshal(out)
			out.Segments[0].State = "changed"
			again, err := p.lifecycleCoverage("workspace", b)
			after, _ := json.Marshal(again)
			if err != nil || string(before) != string(after) {
				t.Fatal("aliased provenance", err)
			}
			if _, err := p.lifecycleCoverage("foreign", b); err == nil {
				t.Fatal("foreign history accepted")
			}
		})
	}
}

func TestCandidateLifecycleDateBounds(t *testing.T) {
	start, err := contracts.NewDay(1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	p := &normalizedCharges{lifecycles: lifecycleTestRegistry(t, []contracts.Event{{ID: "start", Kind: contracts.Start, From: start, Revision: 1}})}
	for _, tc := range []struct {
		from, to string
		want     bool
	}{
		{"0001-01-01", "0001-01-01", true},
		{"2024-02-29", "2024-02-29", true},
		{"2023-02-29", "2023-03-01", false},
		{"9999-12-30", "9999-12-30", true},
		{"9999-12-31", "9999-12-31", false},
		{"0000-12-31", "0001-01-01", false},
		{"2026-10-02", "2026-10-01", false},
		{"2026-1-01", "2026-01-02", false},
	} {
		b := normalizedBinding{Contract: "contract", ContractRevision: 1, From: tc.from, To: tc.to}
		if _, err := p.lifecycleCoverage("workspace", b); (err == nil) != tc.want {
			t.Errorf("%s..%s: %v", tc.from, tc.to, err)
		}
	}
	b := normalizedBinding{Contract: "contract", ContractRevision: 1, From: "2026-10-01", To: "2026-10-31"}
	if _, err := (&normalizedCharges{}).lifecycleCoverage("workspace", b); err == nil {
		t.Fatal("missing registry accepted")
	}
	b.ContractRevision = 0
	if _, err := p.lifecycleCoverage("workspace", b); err == nil {
		t.Fatal("zero revision accepted")
	}
}
