package migration

import (
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"testing"
)

type modelState struct {
	history    map[string]map[string]Document
	numbers    map[int64]bool
	displays   map[string]bool
	external   map[string]bool
	high       int64
	printed    map[string]bool
	keys       map[string]bool
	quarantine map[string]map[string]bool
}

func TestMigrationProperty(t *testing.T) {
	runs := 150
	if testing.Short() {
		runs = 30
	}
	for seed := 0; seed < runs; seed++ {
		runMigrationProperty(t, int64(seed))
	}
}

func runMigrationProperty(t *testing.T, seed int64) {
	rng := rand.New(rand.NewSource(seed))
	f := standard(t)
	f.approve(Change{Kind: ChangeSource, Source: Source{ID: "src-3", Tenant: "t1", Importer: "imp-3", Workspaces: []string{"ws1"}}})
	f.approve(Change{Kind: ChangeMapping, Mapping: Mapping{Source: "src-3", SourceSeries: "LEGACY", Workspace: "ws1", Series: "inv"}})
	m := modelState{history: map[string]map[string]Document{"src-1": {}, "src-3": {}}, numbers: map[int64]bool{}, displays: map[string]bool{}, external: map[string]bool{},
		printed: map[string]bool{}, keys: map[string]bool{}, quarantine: map[string]map[string]bool{"src-1": {}, "src-3": {}}}
	for round := 0; round < 12; round++ {
		src := []string{"src-1", "src-3"}[rng.Intn(2)]
		picked := map[string]Document{}
		for n := 0; n < 1+rng.Intn(6); n++ {
			id := "d" + strconv.Itoa(rng.Intn(8))
			d := doc(id, int64(1+rng.Intn(12)))
			if rng.Intn(3) == 0 {
				d.Display = "ALT-" + strconv.Itoa(rng.Intn(6))
			}
			if rng.Intn(3) == 0 {
				d.Delivered = true
			}
			if rng.Intn(4) == 0 {
				d.Submitted, d.ExternalKey = true, "erp-"+strconv.Itoa(rng.Intn(5))
			}
			if rng.Intn(4) == 0 {
				d.Total += int64(rng.Intn(3))
			}
			picked[id] = d
		}
		ids := []string{}
		for id := range picked {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		items := []Item{}
		for _, id := range ids {
			d := picked[id]
			items = append(items, Item{Document: &d})
		}
		if rng.Intn(5) == 0 {
			id, _ := f.batch(src, items, "t1")
			f.must(f.r.Reject(src, id, "Wrong export", ann))
			continue
		}
		localNumbers, localDisplays, localExternal := map[int64]bool{}, map[string]bool{}, map[string]bool{}
		for _, id := range ids {
			d := picked[id]
			stored, known := m.history[src][id]
			class := "new"
			switch {
			case known && stored == d:
				class = "duplicate"
			case known:
				class = "conflict"
			case m.numbers[d.Number] || localNumbers[d.Number] || m.displays[d.Display] || localDisplays[d.Display]:
				class = "number_conflict"
			case d.ExternalKey != "" && (m.external[d.ExternalKey] || localExternal[d.ExternalKey]):
				class = "external_conflict"
			}
			if class == "new" {
				localNumbers[d.Number], localDisplays[d.Display] = true, true
				if d.ExternalKey != "" {
					localExternal[d.ExternalKey] = true
				}
				m.history[src][id] = d
			} else if class != "duplicate" {
				m.quarantine[src][fmt.Sprintf("%s|%s|%+v|%+v|%v", id, class, d, stored, known)] = true
			}
			if d.Number > m.high {
				m.high = d.Number
			}
			m.printed[fenceKey("dsp-", "t1", d.Display)] = true
			if d.Delivered || d.Submitted || known && (stored.Delivered || stored.Submitted) {
				m.keys[fenceKey("src-", "t1", src, id)] = true
				for _, x := range []string{d.ExternalKey, stored.ExternalKey} {
					if x != "" {
						m.keys[fenceKey("ext-", "t1", x)] = true
					}
				}
			}
		}
		for n := range localNumbers {
			m.numbers[n] = true
		}
		for d := range localDisplays {
			m.displays[d] = true
		}
		for x := range localExternal {
			m.external[x] = true
		}
		f.accept(src, "t1", items)
		for source, docs := range m.history {
			for id, want := range docs {
				got, err := f.r.Document("t1", source, id)
				if err != nil || got != want {
					t.Fatalf("seed %d round %d: %s/%s %+v want %+v", seed, round, source, id, got, want)
				}
			}
			if len(f.r.sources[source].history) != len(docs) {
				t.Fatalf("seed %d: history size", seed)
			}
			q, _ := f.r.Quarantine("t1", source)
			if len(q) != len(m.quarantine[source]) {
				t.Fatalf("seed %d round %d: quarantine %d want %d", seed, round, len(q), len(m.quarantine[source]))
			}
		}
		e, _ := f.r.Fences("t1", "ws1")
		if len(e.HighWater) != 1 || e.HighWater[0].Number != m.high || !sameSet(e.Keys, m.keys) || !sameSet(e.Printed, m.printed) {
			t.Fatalf("seed %d round %d: fences %+v", seed, round, e)
		}
	}
}

func sameSet(list []string, set map[string]bool) bool {
	if len(list) != len(set) {
		return false
	}
	for _, v := range list {
		if !set[v] {
			return false
		}
	}
	return true
}
