package publication

import (
	"testing"
	"time"
)

func TestPublishAndRetention(t *testing.T) {
	f := setup(t)
	expect(t, "no current", second(f.r.Current("t", "d")), ErrNotFound)
	a := f.load("t", "d", "v1", split(rows(5, "g"), 2))
	reason(t, "begin while sealed", second(f.r.Begin(Spec{Tenant: "t", Dataset: "d", Version: "v2", Rows: 1, Digest: digestOf([][]Row{rows(1, "g")})})), ReasonUnpublishedOpen)
	expect(t, "no current yet", second(f.r.Current("t", "d")), ErrNotFound)
	f.advance(time.Minute)
	p, err := f.r.Publish("t", "d", "v1")
	if err != nil || p.State != StatePublished || !p.Published.Equal(start.Add(time.Minute)) || p.Sequence != a.Sequence {
		t.Fatalf("publish %+v %v", p, err)
	}
	if again, err := f.r.Publish("t", "d", "v1"); err != nil || again.State != StatePublished {
		t.Errorf("repeat %v", err)
	}
	reason(t, "abort published", f.r.Abort("t", "d", "v1"), ReasonPublished)
	reason(t, "seal published", second(f.r.Seal("t", "d", "v1")), ReasonPublished)
	f.must(second(f.r.Begin(Spec{Tenant: "t", Dataset: "d", Version: "v2", Rows: 1, Digest: digestOf([][]Row{rows(1, "g")})})))
	reason(t, "publish staging", second(f.r.Publish("t", "d", "v2")), ReasonNotSealed)
	reason(t, "read staging", second(f.r.Rows("t", "d", "v2", "", 10)), ReasonNotSealed)
	f.must(f.r.Abort("t", "d", "v2"))
	page, _ := f.r.Rows("t", "d", "v1", "", 3)
	f.publish("t", "d", "v3", split(rows(2, "h"), 2))
	rest, err := f.r.Rows("t", "d", "v1", page.Next, 10)
	if err != nil || len(page.Rows)+len(rest.Rows) != 5 || rest.Next != "" {
		t.Errorf("pinned reader %+v %v", rest, err)
	}
	if c, _ := f.r.Current("t", "d"); c.Version != "v3" {
		t.Errorf("current %+v", c)
	}
	if v, _ := f.r.State("t", "d", "v1"); v.State != StateRetained {
		t.Errorf("retained %+v", v)
	}
	reason(t, "republish retained", second(f.r.Publish("t", "d", "v1")), ReasonRetained)
	f.publish("t", "d", "v4", split(rows(1, "x"), 1))
	f.publish("t", "d", "v5", split(rows(1, "y"), 1))
	reason(t, "evicted", second(f.r.Rows("t", "d", "v1", "", 1)), ReasonStale)
	reason(t, "publish evicted", second(f.r.Publish("t", "d", "v1")), ReasonStale)
	if len(f.r.datasets[[2]string{"t", "d"}].retained) != 2 || f.r.held != 4 {
		t.Errorf("retention %d %d", len(f.r.datasets[[2]string{"t", "d"}].retained), f.r.held)
	}
	reason(t, "old id", second(f.r.Begin(Spec{Tenant: "t", Dataset: "d", Version: "v1", Rows: 1, Digest: digestOf([][]Row{rows(1, "g")})})), ReasonStale)
	reason(t, "unpadded", second(f.r.Begin(Spec{Tenant: "t", Dataset: "d", Version: "v10", Rows: 1, Digest: digestOf([][]Row{rows(1, "g")})})), ReasonStale)
	expect(t, "above highest", second(f.r.State("t", "d", "v9")), ErrNotFound)
	reason(t, "evict current", f.r.Evict("t", "d", "v5"), ReasonPublished)
	f.must(f.r.Evict("t", "d", "v3"))
	reason(t, "evict again", f.r.Evict("t", "d", "v3"), ReasonStale)
	f.must(second(f.r.Begin(Spec{Tenant: "t", Dataset: "d", Version: "v6", Rows: 1, Digest: digestOf([][]Row{rows(1, "g")})})))
	reason(t, "evict staging", f.r.Evict("t", "d", "v6"), ReasonNotSealed)
	f.audit()
	f.must(f.r.Retire("t", "d"))
	if f.r.held != 0 || f.r.staged != 0 || f.r.tenantHeld["t"] != 0 {
		t.Errorf("retire freed %d %d", f.r.held, f.r.staged)
	}
	f.audit()
	reason(t, "after retire", second(f.r.Rows("t", "d", "v5", "", 1)), ReasonStale)
	expect(t, "current after retire", second(f.r.Current("t", "d")), ErrNotFound)
	f.must(f.r.Retire("t", "d"))
	expect(t, "retire missing", f.r.Retire("t", "nope"), ErrNotFound)
	expect(t, "retire form", f.r.Retire("", "d"), ErrInvalid)
	reason(t, "reuse after retire", second(f.r.Begin(Spec{Tenant: "t", Dataset: "d", Version: "v5", Rows: 1, Digest: digestOf([][]Row{rows(1, "g")})})), ReasonStale)
	f.publish("t", "d", "v7", split(rows(1, "g"), 1))
	expect(t, "current form", second(f.r.Current("t", "")), ErrInvalid)
	expect(t, "seal missing", second(f.r.Seal("t", "d", "v99")), ErrNotFound)
	expect(t, "abort missing", f.r.Abort("t", "nope", "v1"), ErrNotFound)
	expect(t, "evict form", f.r.Evict("t", "d", ""), ErrInvalid)
}

func TestPaging(t *testing.T) {
	f := setup(t)
	list := []Row{row("a", "a-", "KRW", "1"), row("b", "a", "USD", "2"), row("c", "a", "KRW", "3"), row("d", "a-", "KRW", "4.5")}
	f.publish("t", "d", "v1", [][]Row{list[:2], {list[2], list[3]}})
	all, _ := f.r.Rows("t", "d", "v1", "", 4)
	if len(all.Rows) != 4 || all.Next != "" {
		t.Errorf("full page %+v", all)
	}
	first, _ := f.r.Rows("t", "d", "v1", "", 3)
	if len(first.Rows) != 3 || first.Next != "c" {
		t.Errorf("first %+v", first)
	}
	between, _ := f.r.Rows("t", "d", "v1", "bb", 10)
	if len(between.Rows) != 2 || between.Rows[0].Key != "c" {
		t.Errorf("between %+v", between)
	}
	if end, _ := f.r.Rows("t", "d", "v1", "d", 10); len(end.Rows) != 0 || end.Rows == nil || end.Next != "" {
		t.Errorf("end %+v", end)
	}
	all.Rows[0].Key = "mutated"
	if again, _ := f.r.Rows("t", "d", "v1", "", 1); again.Rows[0].Key != "a" {
		t.Error("rows share memory")
	}
	totals, _ := f.r.Totals("t", "d", "v1", nil, 2)
	want := []Total{{Key: TotalKey{Group: "a", Currency: "KRW"}, Amount: "3"}, {Key: TotalKey{Group: "a", Currency: "USD"}, Amount: "2"}}
	if len(totals.Totals) != 2 || totals.Totals[0] != want[0] || totals.Totals[1] != want[1] || totals.Next == nil || *totals.Next != want[1].Key {
		t.Fatalf("totals %+v", totals)
	}
	rest, _ := f.r.Totals("t", "d", "v1", totals.Next, 2)
	if len(rest.Totals) != 1 || rest.Totals[0] != (Total{Key: TotalKey{Group: "a-", Currency: "KRW"}, Amount: "5.5"}) || rest.Next != nil {
		t.Errorf("rest %+v", rest)
	}
	mid, _ := f.r.Totals("t", "d", "v1", &TotalKey{Group: "a", Currency: "L"}, 5)
	if len(mid.Totals) != 2 || mid.Totals[0].Key.Currency != "USD" {
		t.Errorf("mid %+v", mid)
	}
	for _, limit := range []int{0, f.r.limits.Page + 1} {
		expect(t, "rows limit", second(f.r.Rows("t", "d", "v1", "", limit)), ErrInvalid)
		expect(t, "totals limit", second(f.r.Totals("t", "d", "v1", nil, limit)), ErrInvalid)
	}
	f.must(second(f.r.Rows("t", "d", "v1", "", f.r.limits.Page)))
	expect(t, "tenant", second(f.r.Rows("t2", "d", "v1", "", 1)), ErrNotFound)
	expect(t, "version form", second(f.r.Totals("t", "d", "", nil, 1)), ErrInvalid)
	f.publish("t2", "d", "v1", [][]Row{{row("z", "g", "EUR", "9")}})
	if other, _ := f.r.Rows("t2", "d", "v1", "", 10); len(other.Rows) != 1 || other.Rows[0].Key != "z" {
		t.Errorf("isolation %+v", other)
	}
}

func TestReservations(t *testing.T) {
	limits := []struct {
		name string
		l    Limits
	}{
		{"staged", Limits{StagedRows: 10}},
		{"tenant staged", Limits{TenantStagedRows: 10}},
		{"held", Limits{HeldRows: 10}},
		{"tenant held", Limits{TenantHeldRows: 10}},
	}
	for _, c := range limits {
		f := setupWith(t, c.l)
		f.must(second(f.r.Begin(Spec{Tenant: "t", Dataset: "a", Version: "v1", Rows: 6, Digest: digestOf([][]Row{rows(6, "g")})})))
		f.must(second(f.r.Begin(Spec{Tenant: "t", Dataset: "b", Version: "v1", Rows: 4, Digest: digestOf([][]Row{rows(4, "g")})})))
		expect(t, c.name, second(f.r.Begin(Spec{Tenant: "t", Dataset: "c", Version: "v1", Rows: 1, Digest: digestOf([][]Row{rows(1, "g")})})), ErrLimit)
		f.must(f.r.Abort("t", "b", "v1"))
		f.must(second(f.r.Begin(Spec{Tenant: "t", Dataset: "c", Version: "v1", Rows: 4, Digest: digestOf([][]Row{rows(4, "g")})})))
	}
	f := setupWith(t, Limits{TenantHeldRows: 10})
	f.publish("t", "a", "v1", [][]Row{rows(10, "g")})
	expect(t, "held full", second(f.r.Begin(Spec{Tenant: "t", Dataset: "a", Version: "v2", Rows: 1, Digest: digestOf([][]Row{rows(1, "g")})})), ErrLimit)
	f.must(second(f.r.Begin(Spec{Tenant: "u", Dataset: "a", Version: "v1", Rows: 10, Digest: digestOf([][]Row{rows(10, "g")})})))
	f.must(f.r.Retire("t", "a"))
	f.must(second(f.r.Begin(Spec{Tenant: "t", Dataset: "a", Version: "v2", Rows: 10, Digest: digestOf([][]Row{rows(10, "g")})})))
	g := setupWith(t, Limits{TenantDatasets: 1, Datasets: 2})
	g.must(second(g.r.Begin(Spec{Tenant: "t", Dataset: "a", Version: "v1", Rows: 1, Digest: digestOf([][]Row{rows(1, "g")})})))
	expect(t, "tenant datasets", second(g.r.Begin(Spec{Tenant: "t", Dataset: "b", Version: "v1", Rows: 1, Digest: digestOf([][]Row{rows(1, "g")})})), ErrLimit)
	g.must(second(g.r.Begin(Spec{Tenant: "u", Dataset: "a", Version: "v1", Rows: 1, Digest: digestOf([][]Row{rows(1, "g")})})))
	expect(t, "datasets", second(g.r.Begin(Spec{Tenant: "w", Dataset: "a", Version: "v1", Rows: 1, Digest: digestOf([][]Row{rows(1, "g")})})), ErrLimit)
}

func TestExpiry(t *testing.T) {
	f := setupWith(t, Limits{StagedRows: 10, StagingTTL: time.Hour})
	f.must(second(f.r.Begin(Spec{Tenant: "t", Dataset: "a", Version: "v1", Rows: 10, Digest: digestOf([][]Row{rows(10, "g")})})))
	f.advance(30 * time.Minute)
	expect(t, "full", second(f.r.Begin(Spec{Tenant: "t", Dataset: "c", Version: "v1", Rows: 1, Digest: digestOf([][]Row{rows(1, "g")})})), ErrLimit)
	f.advance(30*time.Minute - 1)
	expect(t, "just before", second(f.r.Begin(Spec{Tenant: "t", Dataset: "c", Version: "v1", Rows: 1, Digest: digestOf([][]Row{rows(1, "g")})})), ErrLimit)
	f.advance(1)
	f.must(second(f.r.Begin(Spec{Tenant: "t", Dataset: "c", Version: "v1", Rows: 10, Digest: digestOf([][]Row{rows(10, "g")})})))
	reason(t, "expired read", second(f.r.State("t", "a", "v1")), ReasonStale)
	reason(t, "expired begin", second(f.r.Begin(Spec{Tenant: "t", Dataset: "a", Version: "v1", Rows: 10, Digest: digestOf([][]Row{rows(10, "g")})})), ReasonStale)
	if len(f.r.queue) != 1 {
		t.Errorf("queue %d", len(f.r.queue))
	}
	g := setupWith(t, Limits{StagingTTL: time.Hour})
	g.load("t", "a", "v1", [][]Row{rows(2, "g")})
	g.advance(time.Hour)
	reason(t, "publish after deadline", second(g.r.Publish("t", "a", "v1")), ReasonStale)
	if g.r.held != 0 || g.r.staged != 0 {
		t.Errorf("sealed expiry freed %d %d", g.r.held, g.r.staged)
	}
	g.audit()
	g.load("t", "a", "v2", [][]Row{rows(2, "g")})
	reason(t, "evict sealed", g.r.Evict("t", "a", "v2"), ReasonSealed)
	g.advance(30 * time.Minute)
	g.must(second(g.r.Begin(Spec{Tenant: "t", Dataset: "b", Version: "v1", Rows: 1, Digest: digestOf([][]Row{rows(1, "g")})})))
	g.must(second(g.r.Publish("t", "a", "v2")))
	g.audit()
	h := setupWith(t, Limits{StagingTTL: time.Hour})
	h.load("t", "a", "v1", [][]Row{rows(2, "g")})
	h.advance(time.Hour - 1)
	h.must(second(h.r.Publish("t", "a", "v1")))
	h.advance(2 * time.Hour)
	if c, err := h.r.Current("t", "a"); err != nil || c.Version != "v1" {
		t.Error("published version expired")
	}
	h.must(second(h.r.Begin(Spec{Tenant: "t", Dataset: "a", Version: "v2", Rows: 1, Digest: digestOf([][]Row{rows(1, "g")})})))
	h.audit()
	k := setupWith(t, Limits{StagingTTL: time.Hour})
	k.must(second(k.r.Begin(Spec{Tenant: "t", Dataset: "a", Version: "v1", Rows: 1, Digest: digestOf([][]Row{rows(1, "g")})})))
	k.advance(time.Hour)
	reason(t, "expired on lookup", second(k.r.State("t", "a", "v1")), ReasonStale)
	k.advance(30 * time.Minute)
	k.must(second(k.r.Begin(Spec{Tenant: "t", Dataset: "a", Version: "v2", Rows: 1, Digest: digestOf([][]Row{rows(1, "g")})})))
	k.advance(31 * time.Minute)
	k.must(second(k.r.Begin(Spec{Tenant: "t", Dataset: "b", Version: "v1", Rows: 1, Digest: digestOf([][]Row{rows(1, "g")})})))
	if v, err := k.r.State("t", "a", "v2"); err != nil || v.State != StateStaging {
		t.Errorf("old queue entry dropped the new version %v", err)
	}
	k.audit()
	m := setupWith(t, Limits{StagingTTL: time.Hour})
	one := func(dataset, id string) {
		m.must(second(m.r.Begin(Spec{Tenant: "t", Dataset: dataset, Version: id, Rows: 1, Digest: digestOf([][]Row{rows(1, "g")})})))
	}
	one("z", "v1")
	m.advance(time.Minute)
	one("a", "v1")
	m.must(m.r.Abort("t", "a", "v1"))
	m.advance(30 * time.Minute)
	one("a", "v2")
	m.advance(31 * time.Minute)
	one("b", "v1")
	if v, err := m.r.State("t", "a", "v2"); err != nil || v.State != StateStaging {
		t.Errorf("stale entry dropped a newer version %v", err)
	}
	m.audit()
}

func TestQueueCompaction(t *testing.T) {
	f := setup(t)
	for n := 0; n < 200; n++ {
		id := "v" + pad(n)
		f.must(second(f.r.Begin(Spec{Tenant: "t", Dataset: "d", Version: id, Rows: 1, Digest: digestOf([][]Row{rows(1, "g")})})))
		f.must(f.r.Abort("t", "d", id))
	}
	if len(f.r.queue) > 2*f.r.open+65 || f.r.open != 0 {
		t.Errorf("queue %d open %d", len(f.r.queue), f.r.open)
	}
	f.must(second(f.r.Begin(Spec{Tenant: "t", Dataset: "e", Version: "v1", Rows: 1, Digest: digestOf([][]Row{rows(1, "g")})})))
	f.must(second(f.r.Begin(Spec{Tenant: "t", Dataset: "d", Version: "v999999", Rows: 1, Digest: digestOf([][]Row{rows(1, "g")})})))
	for n := 0; n < 70; n++ {
		id := "w" + pad(n)
		f.must(second(f.r.Begin(Spec{Tenant: "u", Dataset: "x", Version: id, Rows: 1, Digest: digestOf([][]Row{rows(1, "g")})})))
		if len(f.r.queue) > 2*f.r.open+63 {
			t.Fatalf("queue %d with %d open", len(f.r.queue), f.r.open)
		}
		f.must(f.r.Abort("u", "x", id))
	}
	live := 0
	for _, q := range f.r.queue {
		if u := q.dataset.unpublished; u != nil && u.view.Sequence == q.sequence {
			live++
		}
	}
	if len(f.r.queue) > 2*f.r.open+65 {
		t.Errorf("queue not compacted %d", len(f.r.queue))
	}
	if live != 2 || f.r.open != 2 {
		t.Errorf("live %d open %d", live, f.r.open)
	}
	f.audit()
}
