package server

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/retention"
)

func tombstoneFixture(t *testing.T) world {
	t.Helper()
	f := setup(t, SeedSample).ready()
	p, err := f.s.retention.Propose(retention.Change{Kind: retention.ChangePolicy, Policy: retention.Policy{Class: retention.ClassClaim, Period: 1}}, retAnn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.retention.Approve(p.Number, retBob, retention.Impact{}); err != nil {
		t.Fatal(err)
	}
	return f
}

func (f world) deleteClaims(prefix, workspace string, count int) {
	for start := 0; start < count; start += 100 {
		ids := []string{}
		for n := start; n < min(count, start+100); n++ {
			id := fmt.Sprintf("%s-%04d", prefix, n)
			ids = append(ids, id)
			f.retRegister(id, retention.ClassClaim, 10, workspace)
		}
		run := f.retPlan(retention.ClassClaim, workspace, ids...)
		f.retDelete(run, 0, ids...)
	}
}

func TestTombstoneWindowIsTheWorkspacesOwn(t *testing.T) {
	f := tombstoneFixture(t)
	root := f.as(AdminLogin)
	before := whole(retSection(f.retentionRead(root), "decisions")["tombstones"].(map[string]any)["total"])
	recorded := f.s.retention.Watermark()
	f.deleteClaims("mine", SampleWorkspace, 5)
	f.deleteClaims("hidden", "other-workspace", 1200)
	if f.s.retention.Watermark() != recorded+1205 {
		t.Fatalf("the deletions were not all recorded: %d after %d", f.s.retention.Watermark(), recorded)
	}
	got := f.retentionRead(root)
	tomb := retSection(got, "decisions")["tombstones"].(map[string]any)
	if whole(tomb["total"]) != before+5 {
		t.Fatalf("the workspace's own records left its window when another workspace deleted: %v", tomb["total"])
	}
	for _, word := range []string{`"watermark"`, `"sequence"`, "hidden-"} {
		if strings.Contains(got.raw, word) {
			t.Errorf("the read carries %s", word)
		}
	}
	page := f.readAs(root, "/api/v1/retention/tombstones")
	if page.status != 200 || whole(page.body["total"]) != before+5 || strings.Contains(page.raw, `"watermark"`) || strings.Contains(page.raw, `"sequence"`) {
		t.Fatalf("the further read: %s", page.raw[:min(len(page.raw), 300)])
	}
}

func TestTombstoneCursorIsOnlyASignedArtifact(t *testing.T) {
	f := tombstoneFixture(t)
	root := f.as(AdminLogin)
	f.deleteClaims("mine", SampleWorkspace, 60)
	path := "/api/v1/retention/tombstones"
	first := f.readAs(root, path+"?limit=10")
	next := fmt.Sprint(first.body["next"])
	if first.status != 200 || next == "" {
		t.Fatalf("first page: %s", first.raw[:min(len(first.raw), 300)])
	}
	body, err := base64.RawURLEncoding.DecodeString(strings.Split(next, ".")[0])
	rows := first.body["tombstones"].([]any)
	last := rows[len(rows)-1].(map[string]any)["artifact"]
	if err != nil || string(body) != fmt.Sprintf(`{"a":%q}`, last) {
		t.Fatalf("the cursor holds %q, not only the last artifact %v", body, last)
	}
	for _, bad := range []string{"5", "2071", next + "x", "a.b"} {
		if got := f.readAs(root, path+"?after="+bad); got.status != 422 {
			t.Errorf("after=%s: %d", bad, got.status)
		}
	}
	if got := f.readAs(root, path+"?after="+f.s.tombstoneCursor("never-deleted")); got.status != 409 {
		t.Errorf("an artifact outside the window: %d", got.status)
	}
	second := f.readAs(root, path+"?limit=10&after="+next)
	if second.status != 200 || second.body["tombstones"].([]any)[0].(map[string]any)["artifact"] == last {
		t.Fatalf("the second page: %s", second.raw[:min(len(second.raw), 300)])
	}
}

func TestTombstoneIndexRebuildsWhenItsSourceChanges(t *testing.T) {
	f := tombstoneFixture(t)
	root := f.as(AdminLogin)
	f.deleteClaims("mine", SampleWorkspace, 3)
	f.retentionRead(root)
	x := &f.s.tombstoneWindow
	if x.ws != SampleWorkspace || x.upTo != f.s.retention.Watermark() {
		t.Fatalf("index %s %d of %d", x.ws, x.upTo, f.s.retention.Watermark())
	}
	x.rows, x.upTo = nil, f.s.retention.Watermark()
	x.ws = "elsewhere"
	if total := whole(retSection(f.retentionRead(root), "decisions")["tombstones"].(map[string]any)["total"]); total == 0 || x.ws != SampleWorkspace {
		t.Fatalf("the index did not rebuild for a different workspace: %d", total)
	}
	x.rows, x.upTo, x.by = nil, f.s.retention.Watermark(), nil
	if total := whole(retSection(f.retentionRead(root), "decisions")["tombstones"].(map[string]any)["total"]); total == 0 || x.by != f.s.ret {
		t.Fatalf("the index did not rebuild for a different store: %d", total)
	}
	x.rows, x.upTo, x.of = nil, f.s.retention.Watermark(), nil
	if total := whole(retSection(f.retentionRead(root), "decisions")["tombstones"].(map[string]any)["total"]); total == 0 || x.of != f.s.retention {
		t.Fatalf("the index did not rebuild for a different registry: %d", total)
	}
}
