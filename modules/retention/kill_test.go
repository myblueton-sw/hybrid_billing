package retention

import (
	"testing"
	"time"
)

func TestBoundaries(t *testing.T) {
	f := setup(t)
	if f.r.Register(Artifact{ID: "d", Class: ClassSource, Tenants: []string{"a", "a"}, Anchor: start, Stores: []string{"db"}}, ingest) == nil {
		t.Error("duplicate tenants")
	}
	f.policy(ClassSource, 10, 0)
	f.must(f.r.Register(Artifact{ID: "edge", Class: ClassSource, Tenants: one, Anchor: start.Add(-240 * time.Hour), Stores: []string{"db"}}, ingest))
	f.must(f.r.Register(Artifact{ID: "early", Class: ClassSource, Tenants: one, Anchor: start.Add(-240*time.Hour + time.Nanosecond), Stores: []string{"db"}}, ingest))
	p, _ := f.r.Preview(ClassSource, "", 1)
	if len(p.Eligible) != 1 || p.Eligible[0] != "edge" || p.More {
		t.Errorf("ten days of 24 hours %+v", p)
	}
	equal := f.r
	pr, err := equal.Propose(Change{Kind: ChangePolicy, Policy: Policy{Class: ClassSource, Period: 5, Grace: 5}}, ann)
	f.must(err)
	if !pr.Change.Impact.AsOf.IsZero() {
		t.Error("equal total is not a shortening")
	}
	f.must(second(equal.Approve(pr.Number, bob, Impact{})))
	id := f.plan(ClassSource, 1, "edge")
	if p, _ := f.r.Preview(ClassSource, "", 10); len(p.Eligible) != 0 {
		t.Error("owned artifact previewed")
	}
	c, _ := f.r.Claim(id, 0, worker)
	f.must(f.r.Report(id, 0, c.Token, "db", []string{"edge"}, nil, worker))
	reason(t, "proceed deleted", second(f.r.Proceed(id, 0, c.Token, "edge", worker)), ReasonNotInChunk)
	if p, _ := f.r.Preview(ClassSource, "", 10); len(p.Eligible) != 0 {
		t.Error("deleted artifact previewed")
	}
	f.advance(time.Hour)
	f.policy(ClassSource, 100, 0)
	f.put("young", ClassSource, 5, one)
	if iv, _ := f.r.Impact(Policy{Class: ClassSource, Period: 1}); iv.Impact.Count != 2 {
		t.Errorf("deleted artifact in impact %+v", iv.Impact)
	}
	q, _ := f.r.Propose(Change{Kind: ChangePolicy, Policy: Policy{Class: ClassSource, Period: 1}}, ann)
	iv, _ := f.r.Impact(Policy{Class: ClassSource, Period: 1})
	f.advance(time.Hour)
	f.must(second(f.r.Approve(q.Number, bob, iv.Impact)))
	if ts, err := f.r.Tombstones(0, 1); err != nil || len(ts) != 1 {
		t.Error("tombstone page of one")
	}
	expect(t, "backup store", f.r.BackupExpired("", start, ann), ErrInvalid)
	f.must(f.r.BackupExpired("db", start, ann))
	f.must(second(f.r.PlaceHold(Hold{ID: "h", Owner: "o", Reason: "r", Artifacts: []string{"edge"}}, cat)))
	if o, err := f.r.HoldOutcomes("h", "", 1); err != nil || len(o) != 1 {
		t.Error("outcome page of one")
	}
	if _, err := f.r.HoldOutcomes("h", "", 1000); err != nil {
		t.Error("outcome page of 1000")
	}
	g := setup(t)
	a, _ := g.r.Propose(Change{Kind: ChangePolicy, Policy: Policy{Class: ClassSource, Period: 1}}, ann)
	b, _ := g.r.Propose(Change{Kind: ChangePolicy, Policy: Policy{Class: ClassSource, Period: 2}}, ann)
	g.must(second(g.r.Withdraw(b.Number, ann)))
	if len(g.r.pending) != 1 || g.r.pending[0] != a.Number {
		t.Errorf("pending %v", g.r.pending)
	}
	d := setup(t)
	d.approve(Change{Kind: ChangePolicy, Policy: Policy{Class: ClassSource, Period: 1}})
	for n := 2; n < 1001; n++ {
		d.r.done = append(d.r.done, uint64(100000+n))
	}
	d.approve(Change{Kind: ChangePolicy, Policy: Policy{Class: ClassSource, Period: 2}})
	if len(d.r.done) != 1000 {
		t.Errorf("done %d", len(d.r.done))
	}
}
