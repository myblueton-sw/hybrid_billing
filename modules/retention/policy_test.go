package retention

import (
	"testing"
	"time"
)

func TestPolicies(t *testing.T) {
	f := setup(t)
	expect(t, "no policy", second(f.r.Policy(ClassSource)), ErrNotFound)
	f.put("a", ClassSource, 30, []string{"t1"})
	f.put("b", ClassSource, 20, []string{"t1", "t2"})
	f.put("c", ClassSource, 10, []string{"t2"})
	f.put("x", ClassFinal, 30, one)
	f.policy(ClassSource, 25, 5)
	pv, err := f.r.Policy(ClassSource)
	if err != nil || pv.Number != 1 || pv.Proposer != "ann" || pv.Approver != "bob" || !pv.At.Equal(start) {
		t.Fatalf("policy %+v", pv)
	}
	prev, _ := f.r.Preview(ClassSource, "", 10)
	if len(prev.Eligible) != 1 || prev.Eligible[0] != "a" {
		t.Errorf("due at exact boundary %+v", prev)
	}
	f.advance(-time.Nanosecond)
	if prev, _ := f.r.Preview(ClassSource, "", 10); len(prev.Eligible) != 1 {
		t.Error("clock never moves back")
	}
	short := Change{Kind: ChangePolicy, Policy: Policy{Class: ClassSource, Period: 10, Grace: 0}}
	p, err := f.r.Propose(short, ann)
	f.must(err)
	if p.Change.Impact.Count != 2 || !p.Change.Impact.AsOf.Equal(start) || p.Change.Impact.Digest != digestIDs([]string{"b", "c"}) {
		t.Fatalf("impact %+v", p.Change.Impact)
	}
	if p.Change.Impact.Digest != "1b3536bb5505545b611a9077654d308b0aac52db128afb031b5646c47b308076" {
		t.Errorf("digest KAT %s", p.Change.Impact.Digest)
	}
	reason(t, "no reviewed impact", second(f.r.Approve(p.Number, bob, Impact{})), ReasonImpactRequired)
	iv, _ := f.r.Impact(Policy{Class: ClassSource, Period: 10})
	if iv.Tenants["t1"] != 1 || iv.Tenants["t2"] != 2 || iv.Held != 0 {
		t.Errorf("impact view %+v", iv)
	}
	f.advance(time.Hour + 2*time.Nanosecond)
	reason(t, "too old", second(f.r.Approve(p.Number, bob, iv.Impact)), ReasonImpactRequired)
	future := iv.Impact
	future.AsOf = f.at().Add(time.Second)
	reason(t, "future", second(f.r.Approve(p.Number, bob, future)), ReasonImpactRequired)
	iv, _ = f.r.Impact(Policy{Class: ClassSource, Period: 10})
	f.put("d", ClassSource, 15, []string{"t3"})
	reason(t, "changed", second(f.r.Approve(p.Number, bob, iv.Impact)), ReasonImpactChanged)
	same := iv.Impact
	same.Count = 3
	reason(t, "count changed digest same", second(f.r.Approve(p.Number, bob, same)), ReasonImpactChanged)
	iv, _ = f.r.Impact(Policy{Class: ClassSource, Period: 10})
	f.must(second(f.r.Approve(p.Number, bob, iv.Impact)))
	if pv, _ := f.r.Policy(ClassSource); pv.Number != 2 || pv.Policy.Period != 10 {
		t.Error("shortened")
	}
	long := f.approve(Change{Kind: ChangePolicy, Policy: Policy{Class: ClassSource, Period: 100}})
	if long.Change.Impact.Count != 0 || !long.Change.Impact.AsOf.IsZero() {
		t.Error("lengthening has no impact")
	}
	if prev, _ := f.r.Preview(ClassSource, "", 10); len(prev.Eligible) != 0 {
		t.Error("lengthened")
	}
	g := setup(t)
	g.policy(ClassSource, 10, 0)
	g.put("p1", ClassSource, 20, one)
	g.put("p2", ClassSource, 20, one)
	g.put("p3", ClassSource, 20, one)
	id := g.plan(ClassSource, 1, "p1", "p2", "p3")
	g.approve(Change{Kind: ChangePolicy, Policy: Policy{Class: ClassSource, Period: 30}})
	c, _ := g.r.Claim(id, 0, worker)
	if len(c.Deleting) != 0 {
		t.Error("lengthened before claim")
	}
	if v := g.run(id); v.Counts[OutcomeNotDue] != 1 || v.Revision != 1 || g.access("p1") != AccessOK {
		t.Errorf("not due %+v", v.Counts)
	}
	h := setup(t)
	h.put("q1", ClassSource, 20, one)
	h.put("q2", ClassSource, 20, one)
	h.put("q3", ClassSource, 20, one)
	h.put("f1", ClassFinal, 1, one)
	h.policy(ClassSource, 10, 0)
	if iv, _ := h.r.Impact(Policy{Class: ClassSource, Period: 5}); iv.Impact.Count != 0 {
		t.Error("already due under current is not newly due")
	}
	h.policy(ClassSource, 30, 0)
	_, err = h.r.PlaceHold(Hold{ID: "h", Owner: "o", Reason: "r", Artifacts: []string{"q1"}}, cat)
	h.must(err)
	h.must(h.r.Refer(Reference{From: "f1", To: "q2", Kind: Requires}, ingest))
	if iv, _ := h.r.Impact(Policy{Class: ClassSource, Period: 10}); iv.Impact.Count != 3 || iv.Held != 1 || iv.Referenced != 1 {
		t.Errorf("impact breakdown %+v", iv)
	}
	expect(t, "impact form", second(h.r.Impact(Policy{Class: "x", Period: 1})), ErrInvalid)
}

func TestHolds(t *testing.T) {
	f := setup(t)
	f.policy(ClassSource, 1, 0)
	f.policy(ClassConversation, 1, 0)
	f.put("s1", ClassSource, 5, []string{"t1"})
	f.put("s2", ClassSource, 5, []string{"t2"})
	f.put("c1", ClassConversation, 5, []string{"t1"})
	f.put("e1", ClassFinal, 1, []string{"t1"})
	f.must(f.r.Refer(Reference{From: "e1", To: "c1", Kind: Mentions}, ingest))
	f.must(f.r.Refer(Reference{From: "e1", To: "c1", Kind: Mentions}, ingest))
	hv, err := f.r.PlaceHold(Hold{ID: "h1", Owner: "legal", Reason: "case", Tenants: []string{"t1"}, Classes: []string{ClassSource}}, cat)
	if err != nil || hv.Placer != "cat" || len(hv.Counts) != 0 {
		t.Fatalf("hold %+v %v", hv, err)
	}
	prev, _ := f.r.Preview(ClassSource, "", 10)
	if len(prev.Eligible) != 1 || prev.Eligible[0] != "s2" || prev.Held != 1 {
		t.Errorf("tenant class hold %+v", prev)
	}
	cid := f.plan(ClassConversation, 1, "c1")
	cc, _ := f.r.Claim(cid, 0, worker)
	f.must(f.r.Report(cid, 0, cc.Token, "db", []string{"c1"}, nil, worker))
	if f.access("c1") != AccessDeleted || f.access("e1") != AccessOK {
		t.Error("conversation deleted while evidence mentions it; evidence kept")
	}
	f.must(f.r.AddTenant("s2", "t1", ingest))
	f.must(f.r.AddTenant("s2", "t1", ingest))
	if a, _ := f.r.Artifact("s2"); len(a.Tenants) != 2 || a.Tenants[0] != "t1" {
		t.Errorf("tenant added %+v", a.Tenants)
	}
	if prev, _ := f.r.Preview(ClassSource, "", 10); len(prev.Eligible) != 0 || prev.Held != 2 {
		t.Error("added tenant covered")
	}
	reason(t, "tenant on deleted", f.r.AddTenant("c1", "t9", ingest), ReasonDeleted)
	expect(t, "tenant missing", f.r.AddTenant("zz", "t9", ingest), ErrNotFound)
	expect(t, "tenant form", f.r.AddTenant("s2", "", ingest), ErrInvalid)
	reason(t, "tenant by human", f.r.AddTenant("s2", "t3", ann), ReasonNotSystem)
	hv, err = f.r.PlaceHold(Hold{ID: "h2", Owner: "legal", Reason: "case", Artifacts: []string{"c1"}}, cat)
	if err != nil || hv.Counts[HoldNotPreserved] != 1 {
		t.Errorf("hold on deleted %+v %v", hv, err)
	}
	expect(t, "hold form", second(f.r.PlaceHold(Hold{ID: "h9"}, cat)), ErrInvalid)
	expect(t, "hold repeat", second(f.r.PlaceHold(Hold{ID: "h2", Owner: "legal", Reason: "case", Artifacts: []string{"c1"}}, cat)), ErrConflict)
	expect(t, "hold missing artifact", second(f.r.PlaceHold(Hold{ID: "h3", Owner: "o", Reason: "r", Artifacts: []string{"zz"}}, cat)), ErrNotFound)
	reason(t, "hold by system", second(f.r.PlaceHold(Hold{ID: "h3", Owner: "o", Reason: "r", Artifacts: []string{"s1"}}, worker)), ReasonNotHuman)
	reason(t, "placer proposes", second(f.r.Propose(Change{Kind: ChangeRelease, Release: Release{Hold: "h1", Reason: "closed"}}, cat)), ReasonPlacer)
	q, err := f.r.Propose(Change{Kind: ChangeRelease, Release: Release{Hold: "h1", Reason: "closed"}}, ann)
	f.must(err)
	reason(t, "placer approves", second(f.r.Approve(q.Number, cat, Impact{})), ReasonPlacer)
	f.must(second(f.r.Approve(q.Number, dan, Impact{})))
	reason(t, "released twice", f.try(Change{Kind: ChangeRelease, Release: Release{Hold: "h1", Reason: "x"}}), ReasonReleased)
	expect(t, "release missing", f.try(Change{Kind: ChangeRelease, Release: Release{Hold: "zz", Reason: "x"}}), ErrNotFound)
	if hv, _ := f.r.Hold("h1"); !hv.Released {
		t.Error("released view")
	}
	if prev, _ := f.r.Preview(ClassSource, "", 10); len(prev.Eligible) != 2 {
		t.Error("released hold no longer covers")
	}
	expect(t, "hold read", second(f.r.Hold("zz")), ErrNotFound)
	expect(t, "outcomes missing", second(f.r.HoldOutcomes("zz", "", 1)), ErrNotFound)
	expect(t, "outcomes limit", second(f.r.HoldOutcomes("h2", "", 1001)), ErrInvalid)
	expect(t, "outcomes zero", second(f.r.HoldOutcomes("h2", "", 0)), ErrInvalid)
	if o, _ := f.r.HoldOutcomes("h2", "c1", 1000); len(o) != 0 {
		t.Error("after")
	}
	g := setup(t)
	g.policy(ClassSource, 1, 0)
	for _, id := range []string{"a", "b", "c", "d"} {
		g.put(id, ClassSource, 5, one)
	}
	id := g.plan(ClassSource, 4, "a", "b", "c", "d")
	c, _ := g.r.Claim(id, 0, worker)
	g.must(g.r.Report(id, 0, c.Token, "db", []string{"a"}, nil, worker))
	hv, _ = g.r.PlaceHold(Hold{ID: "hx", Owner: "o", Reason: "r", Tenants: one}, cat)
	if hv.Counts[HoldNotPreserved] != 1 || hv.Counts[HoldOutcomePending] != 3 {
		t.Errorf("pending outcomes %+v", hv.Counts)
	}
	page, _ := g.r.HoldOutcomes("hx", "a", 2)
	if len(page) != 2 || page[0].Artifact != "b" || page[1].Artifact != "c" || page[0].Outcome != HoldOutcomePending {
		t.Errorf("page %+v", page)
	}
	g.must(g.r.Report(id, 0, c.Token, "db", []string{"b"}, []string{"c", "d"}, worker))
	if hv, _ := g.r.Hold("hx"); hv.Counts[HoldNotPreserved] != 2 || len(hv.Counts) != 1 {
		t.Errorf("resolved %+v", hv.Counts)
	}
	rc, _ := g.r.Retry(id, 0, worker)
	if len(rc.Deleting) != 0 || g.run(id).Counts[OutcomeHeld] != 2 {
		t.Error("retry refused by hold placed after failure")
	}
}

func TestStop(t *testing.T) {
	f := setup(t)
	f.policy(ClassSource, 1, 0)
	for _, id := range []string{"a", "b", "c", "d", "e"} {
		f.put(id, ClassSource, 5, one)
	}
	id := f.plan(ClassSource, 2, "a", "b", "c", "d", "e")
	c, _ := f.r.Claim(id, 0, worker)
	reason(t, "stop by system", second(f.r.Stop(id, worker)), ReasonNotHuman)
	expect(t, "stop missing", second(f.r.Stop("run/9", cat)), ErrNotFound)
	v, err := f.r.Stop(id, cat)
	if err != nil || v.State != RunStopped || v.Counts[OutcomeStopped] != 3 || v.Stopper != "cat" || f.access("c") != AccessOK || f.access("a") != AccessBlocked {
		t.Fatalf("stopped %+v %v", v, err)
	}
	if again, _ := f.r.Stop(id, dan); again.Stopper != "cat" {
		t.Error("stop idempotent")
	}
	reason(t, "claim after stop", second(f.r.Claim(id, 1, worker)), ReasonChunkDone)
	f.must(f.r.Report(id, 0, c.Token, "db", []string{"a"}, []string{"b"}, worker))
	if v := f.run(id); v.State != RunDone || f.r.unfinished != 0 {
		t.Errorf("stopped resolved %+v", v.State)
	}
	if _, err := f.r.Retry(id, 0, worker); err != nil {
		t.Error("retry after stop")
	}
	g := setup(t)
	g.policy(ClassSource, 1, 0)
	g.put("a", ClassSource, 5, one)
	gid := g.plan(ClassSource, 1, "a")
	g.must(second(g.r.Stop(gid, cat)))
	reason(t, "claim stopped", second(g.r.Claim(gid, 0, worker)), ReasonChunkDone)
	if v := g.run(gid); v.State != RunComplete {
		t.Errorf("stopped before claim %+v", v.State)
	}
}

func TestReplanClearsFailures(t *testing.T) {
	f := setup(t)
	f.policy(ClassSource, 1, 0)
	f.put("a", ClassSource, 5, one, "db", "obj", "tape")
	id := f.plan(ClassSource, 1, "a")
	c, _ := f.r.Claim(id, 0, worker)
	f.must(f.r.Report(id, 0, c.Token, "db", []string{"a"}, nil, worker))
	f.must(f.r.Report(id, 0, c.Token, "obj", nil, []string{"a"}, worker))
	f.must(f.r.Report(id, 0, c.Token, "tape", nil, []string{"a"}, worker))
	f.must(second(f.r.PlaceHold(Hold{ID: "h", Owner: "o", Reason: "r", Artifacts: []string{"a"}}, cat)))
	f.must(second(f.r.Retry(id, 0, worker)))
	f.by(Change{Kind: ChangeRelease, Release: Release{Hold: "h", Reason: "closed"}}, ann, dan)
	id2 := f.plan(ClassSource, 1, "a")
	c2, _ := f.r.Claim(id2, 0, worker)
	f.must(f.r.Report(id2, 0, c2.Token, "obj", []string{"a"}, nil, worker))
	if v := f.run(id2); v.Counts[OutcomeDeleting] != 1 {
		t.Errorf("old failures reused %+v", v.Counts)
	}
	f.must(f.r.Report(id2, 0, c2.Token, "tape", []string{"a"}, nil, worker))
	if f.access("a") != AccessDeleted {
		t.Error("deleted after replan")
	}
}

func TestStaleReportAcrossRuns(t *testing.T) {
	f := setup(t)
	f.policy(ClassSource, 1, 0)
	f.put("a", ClassSource, 5, one, "db", "obj")
	runA := f.plan(ClassSource, 1, "a")
	c, _ := f.r.Claim(runA, 0, worker)
	f.must(f.r.Report(runA, 0, c.Token, "db", nil, []string{"a"}, worker))
	f.must(f.r.Report(runA, 0, c.Token, "obj", nil, []string{"a"}, worker))
	f.must(second(f.r.PlaceHold(Hold{ID: "h", Owner: "o", Reason: "r", Artifacts: []string{"a"}}, cat)))
	f.must(second(f.r.Retry(runA, 0, worker)))
	f.by(Change{Kind: ChangeRelease, Release: Release{Hold: "h", Reason: "closed"}}, ann, dan)
	runB := f.plan(ClassSource, 1, "a")
	cb, _ := f.r.Claim(runB, 0, worker)
	f.must(f.r.Report(runA, 0, c.Token, "db", []string{"a"}, nil, worker))
	f.must(f.r.Report(runA, 0, c.Token, "obj", []string{"a"}, nil, worker))
	b := f.run(runB)
	if b.Counts[OutcomeDeleted] != 1 || b.State != RunDone || f.r.artifacts["a"].run != "" || f.r.unfinished != 0 {
		t.Fatalf("owner run %+v", b)
	}
	if a := f.run(runA); a.Counts[OutcomeDeleted] != 1 {
		t.Errorf("reporting run %+v", a.Counts)
	}
	ts, _ := f.r.Tombstones(0, 10)
	if len(ts) != 1 || ts[0].Run != runB {
		t.Errorf("tombstone %+v", ts)
	}
	reason(t, "owner reports after", f.r.Report(runB, 0, cb.Token, "db", nil, []string{"a"}, worker), ReasonAlreadyDeleted)
	g := setup(t)
	g.policy(ClassSource, 1, 0)
	g.put("p", ClassSource, 5, one, "db")
	g.put("q", ClassSource, 5, one, "db")
	ra := g.plan(ClassSource, 1, "p")
	ca, _ := g.r.Claim(ra, 0, worker)
	g.must(g.r.Report(ra, 0, ca.Token, "db", nil, []string{"p"}, worker))
	g.must(second(g.r.PlaceHold(Hold{ID: "h", Owner: "o", Reason: "r", Artifacts: []string{"p"}}, cat)))
	g.must(second(g.r.Retry(ra, 0, worker)))
	g.by(Change{Kind: ChangeRelease, Release: Release{Hold: "h", Reason: "closed"}}, ann, dan)
	rb := g.plan(ClassSource, 1, "p", "q")
	g.must(g.r.Report(ra, 0, ca.Token, "db", []string{"p"}, nil, worker))
	if v := g.run(rb); v.Counts[OutcomeDeleted] != 1 || v.Counts[OutcomePending] != 1 || v.State != RunRunning {
		t.Errorf("unclaimed owner chunk %+v", v)
	}
	if v := g.run(rb); v.Next != 0 {
		t.Error("unclaimed owner chunk stays open")
	}
	if c, err := g.r.Claim(rb, 0, worker); err != nil || len(c.Deleting) != 0 {
		t.Errorf("deleted not handed out again %v", err)
	}
	f.put("re", ClassSource, 5, []string{"t1"})
	f.must(f.r.AddTenant("re", "t2", ingest))
	f.must(f.r.Register(Artifact{ID: "re", Class: ClassSource, Tenants: []string{"t1"}, Anchor: f.at().Add(-5 * day), Stores: []string{"db"}}, ingest))
}
