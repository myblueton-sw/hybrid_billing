package publication

import (
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func TestBeginAndChunks(t *testing.T) {
	f := setup(t)
	parts := split(rows(25, "g"), 10)
	spec := Spec{Tenant: "t1", Dataset: "usage", Version: "v001", Rows: 25, Digest: digestOf(parts)}
	v, err := f.r.Begin(spec)
	if err != nil || v.State != StateStaging || v.Sequence != 1 || v.NextIndex != 0 || !v.Deadline.Equal(start.Add(24*time.Hour)) {
		t.Fatalf("begin %+v %v", v, err)
	}
	if again, err := f.r.Begin(spec); err != nil || !again.Repeat || f.r.staged != 25 {
		t.Errorf("repeat %+v %v %d", again, err, f.r.staged)
	}
	changed := spec
	changed.Rows = 24
	expect(t, "repeat changed", second(f.r.Begin(changed)), ErrConflict)
	next := spec
	next.Version = "v002"
	reason(t, "unpublished open", second(f.r.Begin(next)), ReasonUnpublishedOpen)
	c := Chunk{Tenant: "t1", Dataset: "usage", Version: "v001", Index: 0, Rows: parts[0]}
	if v, err := f.r.AddChunk(c); err != nil || v.NextIndex != 1 || v.ReceivedRows != 10 || v.Repeat {
		t.Errorf("chunk %+v %v", v, err)
	}
	if v, _ := f.r.AddChunk(c); !v.Repeat || v.ReceivedRows != 10 {
		t.Errorf("resend %+v", v)
	}
	other := c
	other.Rows = parts[1]
	expect(t, "conflict", second(f.r.AddChunk(other)), ErrConflict)
	reason(t, "gap", second(f.r.AddChunk(Chunk{Tenant: "t1", Dataset: "usage", Version: "v001", Index: 2, Rows: parts[2]})), ReasonGap)
	reason(t, "count", second(f.r.Seal("t1", "usage", "v001")), ReasonCountMismatch)
	f.must(second(f.r.AddChunk(Chunk{Tenant: "t1", Dataset: "usage", Version: "v001", Index: 1, Rows: parts[1]})))
	extra := Chunk{Tenant: "t1", Dataset: "usage", Version: "v001", Index: 2, Rows: rows(6, "h")}
	expect(t, "over reserved", second(f.r.AddChunk(extra)), ErrLimit)
	state, err := f.r.State("t1", "usage", "v001")
	if err != nil || state.NextIndex != 2 || state.ReceivedRows != 20 {
		t.Errorf("state %+v %v", state, err)
	}
	f.must(second(f.r.AddChunk(Chunk{Tenant: "t1", Dataset: "usage", Version: "v001", Index: 2, Rows: parts[2]})))
	sealed, err := f.r.Seal("t1", "usage", "v001")
	if err != nil || sealed.State != StateSealed || f.r.staged != 0 || f.r.held != 25 {
		t.Fatalf("seal %+v %v %d %d", sealed, err, f.r.staged, f.r.held)
	}
	if again, err := f.r.Seal("t1", "usage", "v001"); err != nil || again.State != StateSealed {
		t.Errorf("seal repeat %v", err)
	}
	reason(t, "chunk after seal", second(f.r.AddChunk(c)), ReasonSealed)
	forms := []Chunk{
		{Tenant: "", Dataset: "usage", Version: "v001", Rows: parts[0]},
		{Tenant: "t1", Dataset: "a b", Version: "v001", Rows: parts[0]},
		{Tenant: "t1", Dataset: "usage", Version: "", Rows: parts[0]},
		{Tenant: "t1", Dataset: "usage", Version: "v001", Index: -1, Rows: parts[0]},
		{Tenant: "t1", Dataset: "usage", Version: "v001"},
		{Tenant: "t1", Dataset: "usage", Version: "v001", Rows: []Row{row("a", "g", "KRW", "1.0")}},
	}
	for index, c := range forms {
		expect(t, "chunk form "+pad(index), second(f.r.AddChunk(c)), ErrInvalid)
	}
	g := setupWith(t, Limits{ChunkRows: 3})
	g.must(second(g.r.Begin(Spec{Tenant: "t", Dataset: "d", Version: "v1", Rows: 4, Digest: digestOf([][]Row{rows(3, "g")})})))
	g.must(second(g.r.AddChunk(Chunk{Tenant: "t", Dataset: "d", Version: "v1", Rows: rows(3, "g")})))
	expect(t, "chunk cap", second(g.r.AddChunk(Chunk{Tenant: "t", Dataset: "d", Version: "v1", Index: 1, Rows: rows(4, "g")})), ErrInvalid)
	expect(t, "missing dataset", second(f.r.AddChunk(Chunk{Tenant: "t9", Dataset: "usage", Version: "v001", Rows: parts[0]})), ErrNotFound)
	expect(t, "missing version", second(f.r.AddChunk(Chunk{Tenant: "t1", Dataset: "usage", Version: "v999", Rows: parts[0]})), ErrNotFound)
}

func TestSealRefusals(t *testing.T) {
	f := setup(t)
	parts := split(rows(4, "g"), 2)
	f.must(second(f.r.Begin(Spec{Tenant: "t", Dataset: "d", Version: "v1", Rows: 4, Digest: digestOf([][]Row{rows(4, "g")})})))
	for index, p := range parts {
		f.must(second(f.r.AddChunk(Chunk{Tenant: "t", Dataset: "d", Version: "v1", Index: index, Rows: p})))
	}
	reason(t, "digest", second(f.r.Seal("t", "d", "v1")), ReasonDigestMismatch)
	f.must(f.r.Abort("t", "d", "v1"))
	reason(t, "aborted read", second(f.r.State("t", "d", "v1")), ReasonStale)
	dup := [][]Row{{row("a", "g", "KRW", "1"), row("b", "g", "KRW", "1")}, {row("a", "h", "KRW", "2")}}
	f.must(second(f.r.Begin(Spec{Tenant: "t", Dataset: "d", Version: "v2", Rows: 3, Digest: digestOf(dup)})))
	for index, p := range dup {
		f.must(second(f.r.AddChunk(Chunk{Tenant: "t", Dataset: "d", Version: "v2", Index: index, Rows: p})))
	}
	reason(t, "duplicate", second(f.r.Seal("t", "d", "v2")), ReasonDuplicateKey)
	f.must(f.r.Abort("t", "d", "v2"))
	big := [][]Row{{row("a", "g", "KRW", "999999"), row("b", "g", "KRW", "999999")}, {row("c", "g", "KRW", "-999999")}}
	v := f.load("t", "d", "v3", big)
	if v.State != StateSealed {
		t.Fatal("running sum split refused")
	}
	if page, _ := f.r.Totals("t", "d", "v3", nil, 10); len(page.Totals) != 1 || page.Totals[0].Amount != "999999" {
		t.Errorf("totals %+v", page)
	}
	f.must(f.r.Abort("t", "d", "v3"))
	over := [][]Row{}
	for index := 0; index < 101; index++ {
		over = append(over, []Row{row("k"+pad(index), "g", "KRW", "999999.9999")})
	}
	f.must(second(f.r.Begin(Spec{Tenant: "t", Dataset: "d", Version: "v4", Rows: 101, Digest: digestOf(over)})))
	for index, p := range over {
		f.must(second(f.r.AddChunk(Chunk{Tenant: "t", Dataset: "d", Version: "v4", Index: index, Rows: p})))
	}
	reason(t, "result bound", second(f.r.Seal("t", "d", "v4")), ReasonOverflow)
	f.must(f.r.Abort("t", "d", "v4"))
	fits := over[:100]
	f.load("t", "d", "v5", fits)
	f.must(f.r.Abort("t", "d", "v5"))
	f.audit()
	crossing := append(append([][]Row{}, over...), []Row{row("z1", "g", "KRW", "-999999.9999")}, []Row{row("z2", "g", "KRW", "-999999.9999")})
	f.load("t", "d", "v6", crossing)
	if page, _ := f.r.Totals("t", "d", "v6", nil, 1); page.Totals[0].Amount != "98999999.9901" {
		t.Errorf("crossing total %+v", page.Totals)
	}
	f.must(f.r.Abort("t", "d", "v6"))
	f.audit()
	tight, err := decimal.NewProfile(decimal.ProfileSpec{ID: "tight", Revision: "1", InputIntegerDigits: 6, InputScale: 4, ResultIntegerDigits: 8, ResultScale: 4, IntermediateDigits: 12})
	f.must(err)
	k, err := NewRegistry(Config{Profile: tight}, func() time.Time { return start })
	f.must(err)
	kw := world{t: t, r: k}
	kw.must(second(k.Begin(Spec{Tenant: "t", Dataset: "d", Version: "v1", Rows: 101, Digest: digestOf(over)})))
	for index, p := range over {
		kw.must(second(k.AddChunk(Chunk{Tenant: "t", Dataset: "d", Version: "v1", Index: index, Rows: p})))
	}
	reason(t, "intermediate bound", second(k.Seal("t", "d", "v1")), ReasonOverflow)
	h := setupWith(t, Limits{Groups: 2})
	groups := [][]Row{{row("a", "g1", "KRW", "1"), row("b", "g2", "KRW", "1")}, {row("c", "g3", "KRW", "1")}}
	h.must(second(h.r.Begin(Spec{Tenant: "t", Dataset: "d", Version: "v1", Rows: 3, Digest: digestOf(groups)})))
	for index, p := range groups {
		h.must(second(h.r.AddChunk(Chunk{Tenant: "t", Dataset: "d", Version: "v1", Index: index, Rows: p})))
	}
	expect(t, "groups", second(h.r.Seal("t", "d", "v1")), ErrLimit)
	if v, _ := h.r.State("t", "d", "v1"); v.State != StateStaging {
		t.Error("refused seal left staging")
	}
}
