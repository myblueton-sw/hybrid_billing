package evidence

import (
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type simulatedStore struct {
	objects map[string]time.Time
	clock   *testClock
}

var forever = time.Date(9000, 1, 1, 0, 0, 0, 0, time.UTC)

func (s *simulatedStore) key(id string, c Custody) string {
	if c.Kind == Product {
		return "product|" + c.Location
	}
	return id + "|remote|" + c.Location
}

func (s *simulatedStore) put(id string, c Custody) {
	switch {
	case c.Kind == Product:
		s.objects[s.key(id, c)] = forever
	case c.VerifiedUntil.After(s.objects[s.key(id, c)]):
		s.objects[s.key(id, c)] = c.VerifiedUntil
	}
}

func (s *simulatedStore) has(id string, c Custody) bool {
	kept, known := s.objects[s.key(id, c)]
	return known && !kept.Before(s.clock.at)
}

func (s *simulatedStore) remove(id string, c Custody) { delete(s.objects, s.key(id, c)) }

func (h *harness) backed(store *simulatedStore, bundles ...string) {
	h.t.Helper()
	for _, id := range bundles {
		out, err := h.r.Inspect("ws1", id)
		if err != nil || out.Verdict != Held {
			continue
		}
		for after := 0; ; {
			page, err := h.r.Items("ws1", id, after, h.r.limits.Page)
			if err != nil {
				h.t.Fatal(err)
			}
			if len(page) == 0 {
				break
			}
			after += len(page)
			for _, item := range page {
				view, _ := h.r.Artifact("ws1", item.Artifact)
				found := false
				for _, held := range view.Instances {
					found = found || !held.Lost && store.has(item.Artifact, held.Custody)
				}
				if !found {
					h.t.Fatalf("bundle %s is held but no bytes of %s exist", id, item.Artifact)
				}
			}
		}
	}
}

func TestPreservedBillingSurvivesTheLossOfItsSources(t *testing.T) {
	h := newHarness(t, nil)
	store := &simulatedStore{objects: map[string]time.Time{}, clock: h.clock}
	retain := start.Add(5 * 365 * day)
	names := []string{"src", "man", "map", "att", "pol", "eng", "res", "doc"}
	var items []Item
	for _, name := range names {
		id := "inv-" + name
		far := remote(id, time.Time{})
		store.put(id, far)
		h.register(art(id, roles[name], far))
		items = append(items, Item{roles[name], id})
	}
	h.refused(bundleSpec2("b1", retain, items), "not_held:source_original:inv-src not_held:input_manifest:inv-man not_held:mapping:inv-map not_held:attribution:inv-att not_held:policy:inv-pol not_held:engine:inv-eng not_held:result:inv-res not_held:document:inv-doc")
	for _, name := range names {
		id := "inv-" + name
		near := product(id)
		store.put(id, near)
		if _, err := h.r.AddCustody("ws1", id, near); err != nil {
			t.Fatal(err)
		}
	}
	h.confirm(bundleSpec2("b1", retain, items))
	downloads := map[string]Download{}
	for _, item := range items {
		out, err := h.r.Download("ws1", "b1", item.Role, item.Artifact)
		if err != nil {
			t.Fatal(err)
		}
		downloads[item.Artifact] = out
	}
	for index, name := range names {
		id := "inv-" + name
		far := remote(id, time.Time{})
		store.remove(id, far)
		reason := []LossReason{Expired, LostDeleted, AccessRevoked}[index%3]
		if _, err := h.r.ReportLoss("ws1", id, Remote, far.Location, reason); err != nil {
			t.Fatal(err)
		}
		h.clock.advance(100 * day)
		h.backed(store, "b1")
	}
	if got := brief(h.inspect("b1")); got != "held[]/0 risk[]/0 operational protected=true" {
		t.Fatal(got)
	}
	for _, item := range items {
		out, err := h.r.Download("ws1", "b1", item.Role, item.Artifact)
		before := downloads[item.Artifact]
		if err != nil || out.ID != before.ID || out.Version != before.Version || out.Digest != before.Digest || out.Size != before.Size || !reflect.DeepEqual(out.Instances, []Custody{product(item.Artifact)}) {
			t.Fatalf("%s: %+v %v", item.Artifact, out, err)
		}
	}
	for _, item := range items {
		if names, err := h.r.DeleteArtifact("ws1", item.Artifact, DeleteRetentionEnded); err != ErrRejected || names.Total != 1 {
			t.Fatal(names, err)
		}
	}
	h.backed(store, "b1")

	alone := art("far-only", Document, remote("far-only", retain))
	store.put("far-only", alone.Custody)
	h.register(alone)
	second := append(h.set("b2"), Item{Document, "far-only"})
	for _, item := range second[:7] {
		store.put(item.Artifact, product(item.Artifact))
	}
	h.confirm(bundleSpec2("b2", retain, second))
	h.backed(store, "b2")
	store.remove("far-only", alone.Custody)
	if out, err := h.r.ReportLoss("ws1", "far-only", Remote, alone.Custody.Location, AccessRevoked); err != nil || out.Total != 1 {
		t.Fatal(out, err)
	}
	if got := brief(h.inspect("b2")); got != "broken[far-only:lost:access_revoked]/1 risk[]/0 operational protected=true" {
		t.Fatal(got)
	}

	shared := product("engine-v9")
	store.put("engine-a", shared)
	h.register(art("engine-a", Engine, shared))
	theirs := art("engine-b", Engine, shared)
	theirs.WorkspaceID = "ws2"
	if _, err := h.r.Register(theirs); err != ErrConflict {
		t.Fatalf("two artifacts must not rest on one product location: %v", err)
	}
	third := h.set("b3")
	for _, item := range third {
		store.put(item.Artifact, product(item.Artifact))
	}
	third[5] = Item{Engine, "engine-a"}
	h.confirm(bundleSpec2("b3", retain, third))
	if _, err := h.r.DeleteArtifact("ws1", "b3-eng", DeletedBy); err != nil {
		t.Fatal(err)
	}
	store.remove("b3-eng", product("b3-eng"))
	h.backed(store, "b1", "b2", "b3")
	theirs.Custody = product("engine-theirs")
	h.register(theirs)
	store.put("engine-b", theirs.Custody)
	if _, err := h.r.DeleteArtifact("ws2", "engine-b", DeletedBy); err != nil {
		t.Fatal(err)
	}
	store.remove("engine-b", theirs.Custody)
	if !store.has("engine-a", shared) {
		t.Fatal("deleting one artifact removed the bytes of another")
	}
	h.backed(store, "b3")
	if got := brief(h.inspect("b3")); got != "held[]/0 risk[]/0 operational protected=true" {
		t.Fatal(got)
	}
}

func TestDeletionAgainstConfirmation(t *testing.T) {
	for round := 0; round < 200; round++ {
		h := newHarness(t, nil)
		items := h.set("b1")
		var wait sync.WaitGroup
		var confirmed, deleted atomic.Int64
		wait.Add(2)
		go func() {
			defer wait.Done()
			if _, err := h.r.Confirm(bundleSpec("b1", items)); err == nil {
				confirmed.Add(1)
			}
		}()
		go func() {
			defer wait.Done()
			if _, err := h.r.DeleteArtifact("ws1", "b1-pol", DeletedBy); err == nil {
				deleted.Add(1)
			}
		}()
		wait.Wait()
		if confirmed.Load()+deleted.Load() != 1 {
			t.Fatalf("round %d: confirmed %d, deleted %d", round, confirmed.Load(), deleted.Load())
		}
	}
}

func TestHoldAndExtensionAgainstPurge(t *testing.T) {
	for round := 0; round < 200; round++ {
		h := newHarness(t, nil)
		h.confirm(bundleSpec2("b1", start.Add(day), h.set("b1")))
		h.clock.advance(2 * day)
		var wait sync.WaitGroup
		var purged, kept atomic.Int64
		wait.Add(2)
		go func() {
			defer wait.Done()
			if err := h.r.Purge("ws1", "b1", PurgeRetentionEnded); err == nil {
				purged.Add(1)
			}
		}()
		go func() {
			defer wait.Done()
			var err error
			if round%2 == 0 {
				_, err = h.r.PlaceHold(HoldSpec{WorkspaceID: "ws1", ID: "h1", Reason: "case:1", Bundles: []string{"b1"}})
			} else {
				_, err = h.r.Extend("ws1", "b1", start.Add(30*day))
			}
			if err == nil {
				kept.Add(1)
			}
		}()
		wait.Wait()
		view, _ := h.r.Bundle("ws1", "b1")
		if purged.Load()+kept.Load() != 1 || (view.State == Purged) != (purged.Load() == 1) {
			t.Fatalf("round %d: purged %d, kept %d, state %s", round, purged.Load(), kept.Load(), view.State)
		}
	}
}

func TestManyWorkersOnOneWorkspace(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.Artifacts = 5000; l.Bundles = 500 })
	var wait sync.WaitGroup
	var failures atomic.Int64
	for worker := 0; worker < 6; worker++ {
		wait.Add(1)
		go func(worker int) {
			defer wait.Done()
			for round := 0; round < 25; round++ {
				prefix := fmt.Sprintf("w%d-%d", worker, round)
				var items []Item
				for _, name := range []string{"src", "man", "map", "att", "pol", "eng", "res"} {
					id := prefix + "-" + name
					if _, err := h.r.Register(art(id, roles[name], product(id))); err != nil {
						failures.Add(1)
					}
					items = append(items, Item{roles[name], id})
				}
				if _, err := h.r.Confirm(bundleSpec(prefix, items)); err != nil {
					failures.Add(1)
				}
				if _, err := h.r.ReportLoss("ws1", prefix+"-pol", Product, "store:"+prefix+"-pol", Corrupted); err != nil {
					failures.Add(1)
				}
				if out, err := h.r.Inspect("ws1", prefix); err != nil || out.Verdict != Broken {
					failures.Add(1)
				}
				h.r.Exposed("ws1")
				if _, err := h.r.AddCustody("ws1", prefix+"-pol", product(prefix+"-pol-again")); err != nil {
					failures.Add(1)
				}
			}
		}(worker)
	}
	wait.Wait()
	exposed, _ := h.r.Exposed("ws1")
	if failures.Load() != 0 || exposed.Total != 0 || len(h.r.workspaces["ws1"].bundles) != 150 {
		t.Fatal(failures.Load(), exposed.Total)
	}
}
