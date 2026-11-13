package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

func (h *harness) refused(spec BundleSpec, want string) {
	h.t.Helper()
	before := h.r.dump()
	out, err := h.r.Confirm(spec)
	if err != ErrRejected || out.State != "" || codes(out.Reasons) != want {
		h.t.Fatalf("confirm %s:\n got %s (%v)\nwant %s", spec.ID, codes(out.Reasons), err, want)
	}
	h.unchanged(before, "a refused confirmation")
}

func TestConfirmListsEveryReason(t *testing.T) {
	retain := start.Add(365 * day)
	cases := []struct {
		name    string
		prepare func(h *harness, spec *BundleSpec)
		want    string
	}{
		{"an unknown artifact", func(h *harness, s *BundleSpec) { s.Items = append(s.Items, Item{Document, "nowhere"}) }, "unknown_artifact:document:nowhere"},
		{"an artifact of another workspace", func(h *harness, s *BundleSpec) {
			other := art("far", Document, product("far"))
			other.WorkspaceID = "ws2"
			h.register(other)
			s.Items = append(s.Items, Item{Document, "far"})
		}, "unknown_artifact:document:far"},
		{"a deleted artifact", func(h *harness, s *BundleSpec) {
			h.register(art("gone", Document, product("gone")))
			h.r.DeleteArtifact("ws1", "gone", WrongRegistration)
			s.Items = append(s.Items, Item{Document, "gone"})
		}, "deleted_artifact:document:gone"},
		{"a deleted artifact under another role gives one reason", func(h *harness, s *BundleSpec) {
			h.register(art("gone", Document, product("gone")))
			h.r.DeleteArtifact("ws1", "gone", WrongRegistration)
			s.Items = append(s.Items, Item{Result, "gone"})
		}, "deleted_artifact:result:gone"},
		{"another kind", func(h *harness, s *BundleSpec) { s.Items[0].Role = Document }, "wrong_kind:document:b1-src role_missing:source_original"},
		{"another kind that is not held gives one reason", func(h *harness, s *BundleSpec) {
			h.register(art("far", Policy, remote("far", time.Time{})))
			s.Items = append(s.Items, Item{Document, "far"})
		}, "wrong_kind:document:far"},
		{"named twice in one role", func(h *harness, s *BundleSpec) { s.Items = append(s.Items, s.Items[2]) }, "named_twice::b1-map"},
		{"several named twice", func(h *harness, s *BundleSpec) {
			s.Items = append(s.Items, s.Items[6], s.Items[2], s.Items[4], s.Items[1], s.Items[3])
		},
			"named_twice::b1-att named_twice::b1-man named_twice::b1-map named_twice::b1-pol named_twice::b1-res"},
		{"named twice in two roles", func(h *harness, s *BundleSpec) { s.Items = append(s.Items, Item{Document, "b1-res"}) }, "wrong_kind:document:b1-res named_twice::b1-res"},
		{"no source", func(h *harness, s *BundleSpec) { s.Items = s.Items[1:] }, "role_missing:source_original"},
		{"no result and no engine", func(h *harness, s *BundleSpec) { s.Items = s.Items[:5] }, "role_missing:engine role_missing:result"},
		{"FX used without an item", func(h *harness, s *BundleSpec) { s.FX = true }, "fx_mismatch:fx"},
		{"an FX item without the flag", func(h *harness, s *BundleSpec) { s.Items = append(s.Items, h.set("b1", "fx")...) }, "fx_mismatch:fx"},
		{"modules used without an item", func(h *harness, s *BundleSpec) { s.Modules = true }, "module_mismatch:module"},
		{"a module item without the flag", func(h *harness, s *BundleSpec) { s.Items = append(s.Items, h.set("b1", "mod")...) }, "module_mismatch:module"},
		{"retention now", func(h *harness, s *BundleSpec) { s.RetainUntil = start }, "retention_not_ahead"},
		{"retention in the past", func(h *harness, s *BundleSpec) { s.RetainUntil = start.Add(-day) }, "retention_not_ahead"},
		{"an unverified remote location", func(h *harness, s *BundleSpec) {
			h.register(art("far", Document, remote("far", time.Time{})))
			s.Items = append(s.Items, Item{Document, "far"})
		}, "not_held:document:far"},
		{"a remote location verified for too short a time", func(h *harness, s *BundleSpec) {
			h.register(art("far", Document, remote("far", retain.Add(-time.Nanosecond))))
			s.Items = append(s.Items, Item{Document, "far"})
		}, "not_held:document:far"},
		{"a lost copy", func(h *harness, s *BundleSpec) {
			h.r.ReportLoss("ws1", "b1-pol", Product, "store:b1-pol", Corrupted)
		}, "not_held:policy:b1-pol"},
		{"everything at once", func(h *harness, s *BundleSpec) {
			h.register(art("gone", Document, product("gone")))
			h.r.DeleteArtifact("ws1", "gone", WrongRegistration)
			h.register(art("far", Result, remote("far", time.Time{})))
			h.register(art("near", Result, remote("near", time.Time{})))
			h.r.ReportLoss("ws1", "b1-att", Product, "store:b1-att", Expired)
			s.Items = []Item{{Result, "near"}, {Document, "gone"}, {Result, "far"}, {Mapping, "nowhere"}, {Attribution, "b1-att"}, {Policy, "b1-man"}, {Document, "b1-eng"}, {Engine, "b1-eng"}, {Document, "absent"}}
			s.FX, s.Modules, s.RetainUntil = true, true, start
		}, "unknown_artifact:mapping:nowhere unknown_artifact:document:absent deleted_artifact:document:gone wrong_kind:policy:b1-man wrong_kind:document:b1-eng named_twice::b1-eng " +
			"role_missing:source_original role_missing:input_manifest fx_mismatch:fx module_mismatch:module retention_not_ahead not_held:attribution:b1-att not_held:result:far not_held:result:near"},
	}
	for _, c := range cases {
		h := newHarness(t, nil)
		spec := bundleSpec("b1", h.set("b1"))
		c.prepare(h, &spec)
		h.refused(spec, c.want)
	}
	h := newHarness(t, nil)
	for name, change := range map[string]func(*BundleSpec){
		"workspace":     func(s *BundleSpec) { s.WorkspaceID = "" },
		"id":            func(s *BundleSpec) { s.ID = "b 1" },
		"customer":      func(s *BundleSpec) { s.Customer = "" },
		"period":        func(s *BundleSpec) { s.Period = "2026/10" },
		"no retention":  func(s *BundleSpec) { s.RetainUntil = time.Time{} },
		"no item":       func(s *BundleSpec) { s.Items = nil },
		"a role":        func(s *BundleSpec) { s.Items[1].Role = "invoice" },
		"an artifact":   func(s *BundleSpec) { s.Items[1].Artifact = "" },
		"the last item": func(s *BundleSpec) { s.Items[6].Artifact = "a b" },
	} {
		spec := bundleSpec("b1", h.set("b1"))
		change(&spec)
		_, err := h.r.Confirm(spec)
		wantErr(t, err, ErrInvalid, name)
	}
}

func TestConfirmedBundleHeaderAndRepeats(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.Bundles = 2; l.Items = 12 })
	retain := start.Add(365 * day)
	items := h.set("b1", "res", "src", "man", "map", "att", "pol", "eng", "fx", "mod", "doc", "src.2")
	h.register(art("edge", Document, remote("edge", retain)))
	items = append(items, Item{Document, "edge"})
	spec := bundleSpec("b1", items)
	spec.FX, spec.Modules = true, true
	for _, id := range []string{"g2", "g1"} {
		if _, err := h.r.RecordGap(GapSpec{WorkspaceID: "ws1", ID: id, Customer: "cust-1", Period: "2026-10", Role: SourceOriginal, Reason: CaptureFailed, Owner: "ops", Path: "ticket:1"}); err != nil {
			t.Fatal(err)
		}
	}
	for index, other := range []GapSpec{{Customer: "cust-2", Period: "2026-10"}, {Customer: "cust-1", Period: "2026-11"}} {
		other.WorkspaceID, other.ID, other.Role, other.Reason, other.Owner, other.Path = "ws1", fmt.Sprintf("x%d", index), Policy, ReceivedAfterEnd, "ops", "ticket:2"
		h.r.RecordGap(other)
	}
	h.r.RecordGap(GapSpec{WorkspaceID: "ws2", ID: "g9", Customer: "cust-1", Period: "2026-10", Role: Policy, Reason: CaptureFailed, Owner: "ops", Path: "t"})
	h.r.RecordGap(GapSpec{WorkspaceID: "ws1", ID: "g0", Customer: "cust-1", Period: "2026-10", Role: Policy, Reason: CaptureFailed, Owner: "ops", Path: "t"})
	if err := h.r.CloseGap("ws1", "g0", "b1-pol"); err != nil {
		t.Fatal(err)
	}
	over := spec
	over.Items = append(append([]Item(nil), items...), Item{Document, "one-more"})
	_, err := h.r.Confirm(over)
	wantErr(t, err, ErrLimit, "items over the limit")
	h.clock.advance(time.Hour)
	out := h.confirm(spec)
	if !reflect.DeepEqual(out.Gaps, Listed{Total: 2, Items: []string{"g1", "g2"}}) || out.Reasons != nil {
		t.Fatalf("%+v", out)
	}
	sorted := []Item{{SourceOriginal, "b1-src"}, {SourceOriginal, "b1-src.2"}, {InputManifest, "b1-man"}, {Mapping, "b1-map"}, {Attribution, "b1-att"}, {Policy, "b1-pol"}, {FX, "b1-fx"},
		{Engine, "b1-eng"}, {Module, "b1-mod"}, {Result, "b1-res"}, {Document, "b1-doc"}, {Document, "edge"}}
	var got []Item
	for {
		page, err := h.r.Items("ws1", "b1", len(got), 5)
		if err != nil {
			t.Fatal(err)
		}
		if len(page) == 0 {
			break
		}
		got = append(got, page...)
	}
	if !reflect.DeepEqual(got, sorted) {
		t.Fatalf("%v", got)
	}
	encoded := secondEncoding("evidence-bundle-v1", "ws1", "b1", "cust-1", "2026-10", "fx", "modules")
	for _, item := range sorted {
		encoded = append(encoded, secondEncoding(string(item.Role), item.Artifact, digestOf("bytes", item.Artifact))...)
	}
	sum := sha256.Sum256(encoded)
	view, err := h.r.Bundle("ws1", "b1")
	wantCounts := map[Kind]int{SourceOriginal: 2, InputManifest: 1, Mapping: 1, Attribution: 1, Policy: 1, FX: 1, Engine: 1, Module: 1, Result: 1, Document: 2}
	if err != nil || view.Digest != hex.EncodeToString(sum[:]) || !reflect.DeepEqual(view.Counts, wantCounts) || !reflect.DeepEqual(view.Gaps, []string{"g1", "g2"}) || view.State != Confirmed ||
		!view.Retained.Equal(retain) || !view.Spec.RetainUntil.Equal(retain) || !view.Confirmed.Equal(start.Add(time.Hour)) || view.Spec.Items != nil || view.Spec.Customer != "cust-1" || !view.Spec.FX {
		t.Fatalf("%+v %v", view, err)
	}
	view.Counts[Policy], view.Gaps[0] = 9, "changed"
	if again, _ := h.r.Bundle("ws1", "b1"); again.Counts[Policy] != 1 || again.Gaps[0] != "g1" {
		t.Fatal("view shares state")
	}
	plain := bundleSpec("b2", h.set("b2"))
	h.confirm(plain)
	other, _ := h.r.Bundle("ws1", "b2")
	second := secondEncoding("evidence-bundle-v1", "ws1", "b2", "cust-1", "2026-10", "no_fx", "no_modules")
	for _, name := range []string{"src", "man", "map", "att", "pol", "eng", "res"} {
		second = append(second, secondEncoding(string(roles[name]), "b2-"+name, digestOf("bytes", "b2-"+name))...)
	}
	sum = sha256.Sum256(second)
	if other.Digest != hex.EncodeToString(sum[:]) || len(other.Gaps) != 2 {
		t.Fatalf("%+v", other)
	}

	h.clock.advance(time.Hour)
	before := h.r.dump()
	shuffled := spec
	shuffled.Items = append([]Item{items[5], items[0]}, append(append([]Item(nil), items[1:5]...), items[6:]...)...)
	if out, err := h.r.Confirm(shuffled); err != nil || out.State != Confirmed || out.Gaps.Total != 2 {
		t.Fatalf("%+v %v", out, err)
	}
	for name, change := range map[string]func(*BundleSpec){
		"customer":                             func(s *BundleSpec) { s.Customer = "cust-2" },
		"period":                               func(s *BundleSpec) { s.Period = "2026-11" },
		"retention":                            func(s *BundleSpec) { s.RetainUntil = retain.Add(time.Second) },
		"fx":                                   func(s *BundleSpec) { s.FX = false },
		"modules":                              func(s *BundleSpec) { s.Modules = false },
		"an item fewer":                        func(s *BundleSpec) { s.Items = s.Items[:11] },
		"another role":                         func(s *BundleSpec) { s.Items = append([]Item{{Result, "b1-doc"}}, s.Items[1:]...) },
		"the same artifact under another role": func(s *BundleSpec) { s.Items = append([]Item{{Document, "b1-res"}}, s.Items[1:]...) },
		"a role that keeps the order of artifacts": func(s *BundleSpec) {
			for index, item := range s.Items {
				if item.Artifact == "b1-map" {
					s.Items[index].Role = InputManifest
				}
			}
		},
		"another item": func(s *BundleSpec) { s.Items = append([]Item{{Result, "b2-res"}}, s.Items[1:]...) },
	} {
		changed := spec
		changed.Items = append([]Item(nil), spec.Items...)
		change(&changed)
		_, err := h.r.Confirm(changed)
		wantErr(t, err, ErrConflict, name)
	}
	_, err = h.r.Confirm(bundleSpec("b3", h.set("b2")))
	wantErr(t, err, ErrLimit, "bundle limit")
	h.unchanged(before, "repeats, conflicts and the limit")
	if _, err := h.r.Extend("ws1", "b1", retain.Add(day)); err != nil {
		t.Fatal(err)
	}
	if out, err := h.r.Confirm(spec); err != nil || out.State != Confirmed {
		t.Fatalf("a repeat is compared with what was confirmed: %+v %v", out, err)
	}
	if view, _ := h.r.Bundle("ws1", "b1"); !view.Retained.Equal(retain.Add(day)) || !view.Spec.RetainUntil.Equal(retain) {
		t.Fatalf("%+v", view)
	}
	for _, c := range []struct {
		ws, id       string
		after, limit int
		want         error
	}{{"ws1", "b1", -1, 1, ErrInvalid}, {"ws1", "b1", 0, 0, ErrInvalid}, {"ws1", "b1", 0, 6, ErrInvalid}, {"ws2", "b1", 0, 1, ErrNotFound}, {"ws1", "b9", 0, 1, ErrNotFound}, {"", "b1", 0, 1, ErrInvalid}, {"ws1", "", 0, 1, ErrInvalid}} {
		_, err := h.r.Items(c.ws, c.id, c.after, c.limit)
		wantErr(t, err, c.want, fmt.Sprint(c))
	}
	if page, err := h.r.Items("ws1", "b1", 11, 5); err != nil || len(page) != 1 || page[0].Artifact != "edge" {
		t.Fatal(page, err)
	}
	if page, err := h.r.Items("ws1", "b1", 99, 5); err != nil || page != nil {
		t.Fatal(page, err)
	}
	if open, err := h.r.Gaps("ws1", "cust-1", "2026-10"); err != nil || !reflect.DeepEqual(open, Listed{Total: 2, Items: []string{"g1", "g2"}}) {
		t.Fatal(open, err)
	}
	page, _ := h.r.Items("ws1", "b1", 0, 1)
	if len(page) != 1 {
		t.Fatal(page)
	}
	page[0].Artifact = "changed"
	if again, _ := h.r.Items("ws1", "b1", 0, 1); again[0].Artifact != "b1-src" {
		t.Fatal("items share state")
	}
	for _, c := range []struct {
		ws, id string
		want   error
	}{{"ws2", "b1", ErrNotFound}, {"ws1", "b9", ErrNotFound}, {"", "b1", ErrInvalid}, {"ws1", "", ErrInvalid}} {
		_, err := h.r.Bundle(c.ws, c.id)
		wantErr(t, err, c.want, "bundle view")
	}
}

func TestDeletionWaitsForPurgeAndPurgeForProtection(t *testing.T) {
	h := newHarness(t, nil)
	retain := start.Add(365 * day)
	shared := h.set("b1")
	h.confirm(bundleSpec("b1", shared))
	second := h.set("b2")
	second[0] = shared[0]
	long := bundleSpec("b2", second)
	long.RetainUntil = retain.Add(365 * day)
	h.confirm(long)
	h.register(art("loose", Document, product("loose")))
	for _, bad := range []DeleteReason{"", "matter_closed", "other"} {
		_, err := h.r.DeleteArtifact("ws1", "loose", bad)
		wantErr(t, err, ErrInvalid, "delete reason")
	}
	for _, bad := range []PurgeReason{"", "wrong_registration", "other"} {
		wantErr(t, h.r.Purge("ws1", "b1", bad), ErrInvalid, "purge reason")
	}
	for _, c := range []struct {
		ws, id string
		want   error
	}{{"ws2", "b1-src", ErrNotFound}, {"ws1", "zz", ErrNotFound}, {"", "b1-src", ErrInvalid}, {"ws1", "", ErrInvalid}} {
		_, err := h.r.DeleteArtifact(c.ws, c.id, DeletedBy)
		wantErr(t, err, c.want, "delete")
		wantErr(t, h.r.Purge(c.ws, strings.Replace(c.id, "b1-src", "b1", 1), PurgedBy), c.want, "purge")
	}
	if names, err := h.r.DeleteArtifact("ws1", "loose", DeleteRetentionEnded); err != nil || names.Total != 0 {
		t.Fatal(names, err)
	}
	if _, err := h.r.DeleteArtifact("ws1", "loose", DeleteRetentionEnded); err != nil {
		t.Fatal(err)
	}
	_, err := h.r.DeleteArtifact("ws1", "loose", DeletedBy)
	wantErr(t, err, ErrConflict, "deleted with another reason")
	refuse := func(id string, want ...string) {
		t.Helper()
		before := h.r.dump()
		names, err := h.r.DeleteArtifact("ws1", id, DeletedBy)
		if err != ErrRejected || !reflect.DeepEqual(names, Listed{Total: len(want), Items: want}) {
			t.Fatalf("delete %s: %+v %v", id, names, err)
		}
		h.unchanged(before, "a refused deletion")
	}
	refuse("b1-src", "b1", "b2")
	refuse("b1-pol", "b1")
	many := newHarness(t, nil)
	common := many.set("c")
	for _, id := range []string{"c4", "c2", "c1", "c3"} {
		many.confirm(bundleSpec(id, common))
	}
	if names, err := many.r.DeleteArtifact("ws1", "c-pol", DeletedBy); err != ErrRejected || !reflect.DeepEqual(names, Listed{Total: 4, Items: []string{"c1", "c2", "c3"}}) {
		t.Fatalf("%+v %v", names, err)
	}
	before := h.r.dump()
	wantErr(t, h.r.Purge("ws1", "b1", PurgeRetentionEnded), ErrRejected, "purge under retention")
	h.unchanged(before, "a refused purge")

	if _, err := h.r.PlaceHold(HoldSpec{WorkspaceID: "ws1", ID: "h0", Reason: "case:0", Bundles: []string{"b1"}}); err != nil {
		t.Fatal(err)
	}
	wantErr(t, h.r.Purge("ws1", "b1", PurgeRetentionEnded), ErrRejected, "purge under retention and a hold")
	if err := h.r.ReleaseHold("ws1", "h0", PlacedInError); err != nil {
		t.Fatal(err)
	}
	h.clock.advance(365*day - time.Nanosecond)
	wantErr(t, h.r.Purge("ws1", "b1", PurgeRetentionEnded), ErrRejected, "a nanosecond before the retention time")
	edge := newHarness(t, nil)
	edge.confirm(bundleSpec("b1", edge.set("b1")))
	edge.clock.advance(365 * day)
	if err := edge.r.Purge("ws1", "b1", PurgedBy); err != nil {
		t.Fatalf("purge at the moment protection ends: %v", err)
	}
	if view, _ := edge.r.Bundle("ws1", "b1"); view.PurgeReason != PurgedBy || view.Changed.Location() != time.UTC {
		t.Fatalf("%+v", view)
	}
	h.clock.advance(time.Nanosecond)
	if got := brief(h.inspect("b1")); got != "held[]/0 risk[]/0 operational protected=false" {
		t.Fatal(got)
	}
	refuse("b1-pol", "b1")
	refuse("b1-src", "b1", "b2")
	hold, err := h.r.PlaceHold(HoldSpec{WorkspaceID: "ws1", ID: "h1", Reason: "case:7", Bundles: []string{"b1"}})
	if err != nil || hold.State != Active || hold.Exposed.Total != 0 {
		t.Fatalf("a hold after the retention time still finds every artifact: %+v %v", hold, err)
	}
	if got := brief(h.inspect("b1")); got != "held[]/0 risk[]/0 operational protected=true" {
		t.Fatal(got)
	}
	wantErr(t, h.r.Purge("ws1", "b1", PurgeRetentionEnded), ErrRejected, "purge under a hold")
	if err := h.r.ReleaseHold("ws1", "h1", MatterClosed); err != nil {
		t.Fatal(err)
	}
	extended, err := h.r.Extend("ws1", "b1", h.clock.at.Add(day))
	if err != nil || !extended.Until.Equal(h.clock.at.Add(day)) || extended.NotHeld != 0 || extended.AtRisk != 0 {
		t.Fatalf("an extension after the lapse: %+v %v", extended, err)
	}
	wantErr(t, h.r.Purge("ws1", "b1", PurgeRetentionEnded), ErrRejected, "purge after an extension")
	h.clock.advance(day)
	h.clock.advance(time.Hour)
	if err := h.r.Purge("ws1", "b1", PurgeRetentionEnded); err != nil {
		t.Fatal(err)
	}
	view, _ := h.r.Bundle("ws1", "b1")
	if view.State != Purged || view.PurgeReason != PurgeRetentionEnded || !view.Changed.Equal(h.clock.at) || view.Digest == "" || view.Counts[Policy] != 1 {
		t.Fatalf("%+v", view)
	}
	before = h.r.dump()
	if err := h.r.Purge("ws1", "b1", PurgeRetentionEnded); err != nil {
		t.Fatal(err)
	}
	wantErr(t, h.r.Purge("ws1", "b1", PurgedBy), ErrConflict, "purged with another reason")
	if out, err := h.r.Confirm(bundleSpec("b1", shared)); err != nil || out.State != Purged {
		t.Fatalf("%+v %v", out, err)
	}
	_, err = h.r.Confirm(bundleSpec("b1", second))
	wantErr(t, err, ErrConflict, "a purged identifier is not used again")
	_, err = h.r.Extend("ws1", "b1", h.clock.at.Add(900*day))
	wantErr(t, err, ErrRejected, "extending a purged bundle")
	_, err = h.r.Extend("ws1", "b1", start)
	wantErr(t, err, ErrRejected, "an earlier time for a purged bundle")
	_, err = h.r.Inspect("ws1", "b1")
	wantErr(t, err, ErrRejected, "inspecting a purged bundle")
	_, err = h.r.Items("ws1", "b1", 0, 1)
	wantErr(t, err, ErrRejected, "items of a purged bundle")
	_, err = h.r.Download("ws1", "b1", Policy, "b1-pol")
	wantErr(t, err, ErrRejected, "download from a purged bundle")
	held, err := h.r.PlaceHold(HoldSpec{WorkspaceID: "ws1", ID: "h2", Reason: "case:8", Bundles: []string{"b2", "b1"}})
	if err != ErrRejected || !reflect.DeepEqual(held.Positions, []int{1}) {
		t.Fatalf("a hold on a purged bundle: %+v %v", held, err)
	}
	both, err := h.r.PlaceHold(HoldSpec{WorkspaceID: "ws1", ID: "h3", Reason: "case:9", Bundles: []string{"b1", "b9", "b2", "b8"}})
	if err != ErrNotFound || !reflect.DeepEqual(both.Positions, []int{1, 3}) {
		t.Fatalf("missing bundles are reported before purged ones: %+v %v", both, err)
	}
	h.unchanged(before, "after the purge")
	refuse("b1-src", "b2")
	if _, err := h.r.DeleteArtifact("ws1", "b1-pol", DeleteRetentionEnded); err != nil {
		t.Fatal(err)
	}
	view2, _ := h.r.Artifact("ws1", "b1-pol")
	if view2.State != Deleted || !view2.Changed.Equal(h.clock.at) || view2.Bundles != 0 {
		t.Fatalf("%+v", view2)
	}
	if got := brief(h.inspect("b2")); got != "held[]/0 risk[]/0 operational protected=true" {
		t.Fatal(got)
	}
}

func TestHoldsAndExtensions(t *testing.T) {
	h := newHarness(t, func(l *Limits) { l.Holds = 2; l.HoldBundles = 2 })
	retain := start.Add(365 * day)
	items := h.set("b1")
	h.register(art("far", Document, remote("far", retain)))
	h.register(art("farther", Document, remote("farther", retain.Add(10*day))))
	h.confirm(bundleSpec("b1", append(items, Item{Document, "far"}, Item{Document, "farther"})))
	h.confirm(bundleSpec("b2", h.set("b2")))
	h.confirm(bundleSpec("b3", h.set("b3")))
	if got := brief(h.inspect("b1")); got != "held[]/0 risk[]/0 operational protected=true" {
		t.Fatal(got)
	}
	for name, spec := range map[string]HoldSpec{
		"workspace":      {ID: "h1", Reason: "case:1", Bundles: []string{"b1"}},
		"id":             {WorkspaceID: "ws1", Reason: "case:1", Bundles: []string{"b1"}},
		"reason":         {WorkspaceID: "ws1", ID: "h1", Reason: "a dispute", Bundles: []string{"b1"}},
		"no bundle":      {WorkspaceID: "ws1", ID: "h1", Reason: "case:1"},
		"a bundle twice": {WorkspaceID: "ws1", ID: "h1", Reason: "case:1", Bundles: []string{"b1", "b1"}},
		"a bundle name":  {WorkspaceID: "ws1", ID: "h1", Reason: "case:1", Bundles: []string{"b1", ""}},
	} {
		_, err := h.r.PlaceHold(spec)
		wantErr(t, err, ErrInvalid, name)
	}
	before := h.r.dump()
	_, err := h.r.PlaceHold(HoldSpec{WorkspaceID: "ws1", ID: "h1", Reason: "case:1", Bundles: []string{"b1", "b2", "b3"}})
	wantErr(t, err, ErrLimit, "bundles per hold")
	out, err := h.r.PlaceHold(HoldSpec{WorkspaceID: "ws1", ID: "h1", Reason: "case:1", Bundles: []string{"b9", "b1"}})
	if err != ErrNotFound || !reflect.DeepEqual(out.Positions, []int{0}) {
		t.Fatalf("%+v %v", out, err)
	}
	out, err = h.r.PlaceHold(HoldSpec{WorkspaceID: "ws9", ID: "h1", Reason: "case:1", Bundles: []string{"b1"}})
	if err != ErrNotFound || !reflect.DeepEqual(out.Positions, []int{0}) {
		t.Fatalf("%+v %v", out, err)
	}
	h.unchanged(before, "refused holds")
	out, err = h.r.PlaceHold(HoldSpec{WorkspaceID: "ws1", ID: "h1", Reason: "case:1", Bundles: []string{"b2", "b1"}})
	if err != nil || out.State != Active || !reflect.DeepEqual(out.Exposed, Listed{Total: 1, Items: []string{"b1"}}) {
		t.Fatalf("%+v %v", out, err)
	}
	if got := brief(h.inspect("b1")); got != "held[]/0 risk[far farther]/2 operational protected=true" {
		t.Fatalf("under a hold every item without a product copy is at risk: %s", got)
	}
	if exposed, err := h.r.Exposed("ws1"); err != nil || !reflect.DeepEqual(exposed, Listed{Total: 1, Items: []string{"b1"}}) {
		t.Fatal(exposed, err)
	}
	before = h.r.dump()
	if again, err := h.r.PlaceHold(HoldSpec{WorkspaceID: "ws1", ID: "h1", Reason: "case:1", Bundles: []string{"b1", "b2"}}); err != nil || again.State != Active || again.Exposed.Total != 1 {
		t.Fatalf("%+v %v", again, err)
	}
	for name, spec := range map[string]HoldSpec{
		"reason":  {WorkspaceID: "ws1", ID: "h1", Reason: "case:2", Bundles: []string{"b1", "b2"}},
		"bundles": {WorkspaceID: "ws1", ID: "h1", Reason: "case:1", Bundles: []string{"b1", "b3"}},
		"fewer":   {WorkspaceID: "ws1", ID: "h1", Reason: "case:1", Bundles: []string{"b1"}},
	} {
		_, err := h.r.PlaceHold(spec)
		wantErr(t, err, ErrConflict, name)
	}
	for _, bad := range []ReleaseReason{"", "retention_ended", "other"} {
		wantErr(t, h.r.ReleaseHold("ws1", "h1", bad), ErrInvalid, "release reason")
	}
	wantErr(t, h.r.ReleaseHold("ws1", "h9", MatterClosed), ErrNotFound, "missing hold")
	wantErr(t, h.r.ReleaseHold("ws2", "h1", MatterClosed), ErrNotFound, "another workspace")
	wantErr(t, h.r.ReleaseHold("", "h1", MatterClosed), ErrInvalid, "workspace")
	wantErr(t, h.r.ReleaseHold("ws1", "", MatterClosed), ErrInvalid, "id")
	h.unchanged(before, "hold repeats and refusals")
	if _, err := h.r.AddCustody("ws1", "far", product("far")); err != nil {
		t.Fatal(err)
	}
	if got := brief(h.inspect("b1")); got != "held[]/0 risk[farther]/1 operational protected=true" {
		t.Fatal(got)
	}
	second, err := h.r.PlaceHold(HoldSpec{WorkspaceID: "ws1", ID: "h2", Reason: "case:2", Bundles: []string{"b1"}})
	if err != nil || second.Exposed.Total != 1 {
		t.Fatal(second, err)
	}
	_, err = h.r.PlaceHold(HoldSpec{WorkspaceID: "ws1", ID: "h3", Reason: "case:3", Bundles: []string{"b1"}})
	wantErr(t, err, ErrLimit, "hold limit")
	far := art("far-pol", Policy, product("far-pol"))
	far.WorkspaceID = "ws2"
	h.register(far)
	if out, err := h.r.PlaceHold(HoldSpec{WorkspaceID: "ws2", ID: "h1", Reason: "case:1", Bundles: []string{"b1"}}); err != ErrNotFound || len(out.Positions) != 1 {
		t.Fatalf("the same hold identifier in another workspace is another hold: %+v %v", out, err)
	}
	for _, reason := range []ReleaseReason{MatterClosed, PlacedInError, ReleasedBy} {
		other := newHarness(t, nil)
		other.confirm(bundleSpec("b1", other.set("b1")))
		other.r.PlaceHold(HoldSpec{WorkspaceID: "ws1", ID: "h1", Reason: "case:1", Bundles: []string{"b1"}})
		if err := other.r.ReleaseHold("ws1", "h1", reason); err != nil {
			t.Fatal(reason, err)
		}
	}
	if err := h.r.ReleaseHold("ws1", "h1", PlacedInError); err != nil {
		t.Fatal(err)
	}
	if view, _ := h.r.Bundle("ws1", "b1"); view.Holds != 1 {
		t.Fatalf("%+v", view)
	}
	if got := brief(h.inspect("b1")); got != "held[]/0 risk[farther]/1 operational protected=true" {
		t.Fatalf("the second hold still protects: %s", got)
	}
	before = h.r.dump()
	if err := h.r.ReleaseHold("ws1", "h1", PlacedInError); err != nil {
		t.Fatal(err)
	}
	wantErr(t, h.r.ReleaseHold("ws1", "h1", MatterClosed), ErrConflict, "released with another reason")
	if again, err := h.r.PlaceHold(HoldSpec{WorkspaceID: "ws1", ID: "h1", Reason: "case:1", Bundles: []string{"b2", "b1"}}); err != nil || again.State != Released {
		t.Fatalf("%+v %v", again, err)
	}
	h.unchanged(before, "after the release")
	if err := h.r.ReleaseHold("ws1", "h2", MatterClosed); err != nil {
		t.Fatal(err)
	}
	if got := brief(h.inspect("b1")); got != "held[]/0 risk[]/0 operational protected=true" {
		t.Fatal(got)
	}

	h.clock.advance(5 * time.Hour)
	for _, step := range []struct {
		name  string
		until time.Time
		want  Extension
	}{
		{"earlier", start, Extension{Until: retain}},
		{"equal", retain, Extension{Until: retain}},
		{"to the end of the second verification", retain.Add(10 * day), Extension{Until: retain.Add(10 * day)}},
		{"beyond it", retain.Add(10*day + time.Nanosecond), Extension{Until: retain.Add(10*day + time.Nanosecond), AtRisk: 1}},
		{"an earlier time again", retain.Add(day), Extension{Until: retain.Add(10*day + time.Nanosecond), AtRisk: 1}},
	} {
		got, err := h.r.Extend("ws1", "b1", step.until)
		if err != nil || got != step.want {
			t.Fatalf("%s: %+v %v", step.name, got, err)
		}
	}
	if got := brief(h.inspect("b1")); got != "held[]/0 risk[farther]/1 operational protected=true" {
		t.Fatal(got)
	}
	for _, c := range []struct {
		ws, id string
		until  time.Time
		want   error
	}{{"ws2", "b1", retain, ErrNotFound}, {"ws1", "b9", retain, ErrNotFound}, {"", "b1", retain, ErrInvalid}, {"ws1", "", retain, ErrInvalid}, {"ws1", "b1", time.Time{}, ErrInvalid}} {
		_, err := h.r.Extend(c.ws, c.id, c.until)
		wantErr(t, err, c.want, "extend")
	}
	if view, _ := h.r.Bundle("ws1", "b1"); !view.Changed.Equal(start.Add(5*time.Hour)) || !view.Confirmed.Equal(start) || view.Retained.Location() != time.UTC {
		t.Fatalf("%+v", view)
	}
}
