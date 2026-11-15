package retention

import (
	"testing"
	"time"
)

func TestDeletionFlow(t *testing.T) {
	f := setup(t)
	f.put("s1", ClassSource, 100, one, "db", "obj")
	f.put("s2", ClassSource, 100, one)
	f.put("s3", ClassSource, 100, []string{"t1", "t2"})
	f.put("s4", ClassSource, 100, one)
	f.put("s5", ClassSource, 10, one)
	f.put("f1", ClassFinal, 1, one)
	f.must(f.r.Refer(Reference{From: "f1", To: "s4", Kind: Requires}, ingest))
	if v, _ := f.r.Preview(ClassSource, "", 10); len(v.Eligible) != 0 {
		t.Fatal("no policy, nothing due")
	}
	f.policy(ClassSource, 90, 0)
	pv, _ := f.r.Preview(ClassSource, "", 10)
	if len(pv.Eligible) != 3 || pv.Referenced != 1 || pv.More {
		t.Fatalf("preview %+v", pv)
	}
	id := f.plan(ClassSource, 2, "s1", "s2", "s3", "s4", "s5")
	v := f.run(id)
	if v.Counts[OutcomePending] != 3 || v.Counts[OutcomeReferenced] != 1 || v.Counts[OutcomeNotDue] != 1 || v.State != RunRunning || v.Revision != 1 || v.Chunks != 3 {
		t.Fatalf("approved %+v", v)
	}
	for _, a := range []string{"s1", "s2", "s3"} {
		if f.access(a) != AccessBlocked {
			t.Errorf("%s not blocked before any chunk", a)
		}
	}
	if f.access("s4") != AccessOK || f.access("s5") != AccessOK {
		t.Error("excluded blocked")
	}
	reason(t, "out of order", second(f.r.Claim(id, 1, worker)), ReasonOutOfOrder)
	c, err := f.r.Claim(id, 0, worker)
	if err != nil || len(c.Deleting) != 2 || c.Deleting[0] != "s1" || !c.Until.Equal(f.at().Add(time.Hour)) {
		t.Fatalf("claim %+v %v", c, err)
	}
	reason(t, "leased", second(f.r.Claim(id, 0, worker)), ReasonLeased)
	reason(t, "next before done", second(f.r.Claim(id, 1, worker)), ReasonOutOfOrder)
	reason(t, "ref to deleting", f.r.Refer(Reference{From: "f1", To: "s1", Kind: Requires}, ingest), ReasonDeleting)
	ok, _ := f.r.Proceed(id, 0, c.Token, "s1", worker)
	if !ok {
		t.Error("proceed")
	}
	f.must(f.r.Report(id, 0, c.Token, "db", []string{"s1", "s2"}, nil, worker))
	if f.r.artifacts["s1"].deleted || !f.r.artifacts["s2"].deleted || f.access("s2") != AccessDeleted || f.access("s1") != AccessBlocked {
		t.Fatal("one store left")
	}
	f.must(f.r.Report(id, 0, c.Token, "db", []string{"s1"}, nil, worker))
	f.must(f.r.Report(id, 0, c.Token, "obj", nil, []string{"s1"}, worker))
	v = f.run(id)
	if v.Counts[OutcomeFailed] != 1 || v.Counts[OutcomeDeleted] != 1 || v.Damaged != 1 || v.Next != 1 || v.Stores["obj"] != (StoreCounts{Target: 1, Failed: 1}) {
		t.Fatalf("after chunk 0 %+v", v)
	}
	reason(t, "deleted then failed", f.r.Report(id, 0, c.Token, "db", nil, []string{"s1"}, worker), ReasonAlreadyDeleted)
	c1, _ := f.r.Claim(id, 1, worker)
	if len(c1.Deleting) != 1 || c1.Deleting[0] != "s3" {
		t.Fatalf("chunk 1 %+v", c1)
	}
	f.must(f.r.Report(id, 1, c1.Token, "db", []string{"s3"}, nil, worker))
	c2, _ := f.r.Claim(id, 2, worker)
	if len(c2.Deleting) != 0 {
		t.Error("chunk 2 has only excluded")
	}
	if v := f.run(id); v.State != RunDone || f.r.unfinished != 0 {
		t.Errorf("done with failure %+v", v.State)
	}
	reason(t, "claim done", second(f.r.Claim(id, 1, worker)), ReasonChunkDone)
	reason(t, "nothing to retry", second(f.r.Retry(id, 1, worker)), ReasonNothingToRetry)
	rc, err := f.r.Retry(id, 0, worker)
	if err != nil || len(rc.Deleting) != 1 {
		t.Fatalf("retry %+v %v", rc, err)
	}
	reason(t, "stale token", f.r.Report(id, 0, c.Token, "obj", nil, []string{"s1"}, worker), ReasonStaleToken)
	f.must(f.r.Report(id, 0, rc.Token, "obj", []string{"s1"}, nil, worker))
	v = f.run(id)
	if v.Counts[OutcomeDeleted] != 3 || v.Damaged != 0 || v.State != RunDone {
		t.Fatalf("retried %+v", v)
	}
	if f.r.Watermark() != 3 || f.r.Restored(2) || !f.r.Restored(3) {
		t.Error("watermark")
	}
	ts, _ := f.r.Tombstones(1, 10)
	if len(ts) != 2 || ts[0].Sequence != 2 || ts[0].Artifact != "s3" || ts[1].Artifact != "s1" || ts[1].Run != id {
		t.Errorf("tombstones %+v", ts)
	}
	f.advance(time.Hour)
	f.must(f.r.BackupExpired("db", f.at(), ann))
	if v := f.run(id); v.State != RunDone {
		t.Error("obj backups not expired")
	}
	f.must(f.r.BackupExpired("obj", f.at(), ann))
	if v := f.run(id); v.State != RunComplete {
		t.Errorf("complete %+v", v.State)
	}
	reason(t, "reregister", f.r.Register(Artifact{ID: "s2", Class: ClassSource, Tenants: one, Anchor: start, Stores: []string{"db"}}, ingest), ReasonDeleted)
	reason(t, "backward", f.r.BackupExpired("db", start, ann), ReasonBackward)
	expect(t, "future backup", f.r.BackupExpired("db", f.at().Add(time.Second), ann), ErrInvalid)
	reason(t, "backup by system", f.r.BackupExpired("db", f.at(), worker), ReasonNotHuman)
	f.put("s6", ClassSource, 100, one)
	f.must(f.r.Refer(Reference{From: "s6", To: "s4", Kind: Requires}, ingest))
	sid := f.plan(ClassSource, 1, "s6")
	sc, _ := f.r.Claim(sid, 0, worker)
	f.must(f.r.Report(sid, 0, sc.Token, "db", []string{"s6"}, nil, worker))
	f.must(f.r.Refer(Reference{From: "f1", To: "s4", Kind: Requires}, ingest))
	if f.r.requiredBy["s4"] != 1 {
		t.Errorf("deleted requirer released %d", f.r.requiredBy["s4"])
	}
}

func TestLeasesAndStaleReports(t *testing.T) {
	f := setup(t)
	f.policy(ClassTemporary, 1, 0)
	f.put("a", ClassTemporary, 5, one, "db", "obj")
	f.put("b", ClassTemporary, 5, one)
	id := f.plan(ClassTemporary, 2, "a", "b")
	c, _ := f.r.Claim(id, 0, worker)
	f.advance(time.Hour)
	_, err := f.r.PlaceHold(Hold{ID: "h1", Owner: "legal", Reason: "case", Artifacts: []string{"a", "b"}}, cat)
	f.must(err)
	c2, err := f.r.Claim(id, 0, worker)
	if err != nil || c2.Token == c.Token || len(c2.Deleting) != 2 {
		t.Fatalf("reclaim keeps deleting %+v %v", c2, err)
	}
	if ok, _ := f.r.Proceed(id, 0, c2.Token, "a", worker); ok {
		t.Error("proceed under hold")
	}
	reason(t, "proceed stale", second(f.r.Proceed(id, 0, c.Token, "a", worker)), ReasonStaleToken)
	reason(t, "proceed other", second(f.r.Proceed(id, 0, c2.Token, "zz", worker)), ReasonNotInChunk)
	f.must(f.r.Report(id, 0, c.Token, "db", []string{"a"}, nil, worker))
	if !f.r.artifacts["a"].damaged {
		t.Error("stale deletion recorded")
	}
	reason(t, "stale failure", f.r.Report(id, 0, c.Token, "obj", nil, []string{"a"}, worker), ReasonStaleToken)
	reason(t, "unknown token", f.r.Report(id, 0, 999, "db", []string{"b"}, nil, worker), ReasonStaleToken)
	f.must(f.r.Report(id, 0, c2.Token, "obj", nil, []string{"a"}, worker))
	f.must(f.r.Report(id, 0, c2.Token, "db", nil, []string{"b"}, worker))
	outs, _ := f.r.HoldOutcomes("h1", "", 10)
	if len(outs) != 1 || outs[0].Artifact != "a" || outs[0].Outcome != HoldPartlyDeleted || outs[0].Stores["db"] != StoreDeleted || outs[0].Stores["obj"] != StoreFailed {
		t.Fatalf("hold outcomes %+v", outs)
	}
	rc, _ := f.r.Retry(id, 0, worker)
	if len(rc.Deleting) != 0 {
		t.Error("held not retried")
	}
	v := f.run(id)
	if v.Counts[OutcomeHeld] != 2 || v.Damaged != 1 || f.access("a") != AccessBlocked || f.access("b") != AccessOK {
		t.Fatalf("held after retry %+v", v)
	}
	reason(t, "ref to damaged", f.r.Refer(Reference{From: "b", To: "a", Kind: Requires}, ingest), ReasonDeleting)
	f.must(f.r.Report(id, 0, c.Token, "obj", []string{"a"}, nil, worker))
	if !f.r.artifacts["a"].deleted || f.run(id).Counts[OutcomeDeleted] != 1 {
		t.Error("late stale deletion completes the artifact")
	}
	hv, _ := f.r.Hold("h1")
	if hv.Counts[HoldNotPreserved] != 1 || len(hv.Counts) != 1 {
		t.Errorf("hold counts %+v", hv.Counts)
	}
	reason(t, "report deleted artifact", f.r.Report(id, 0, rc.Token, "obj", nil, []string{"a"}, worker), ReasonAlreadyDeleted)
	expect(t, "report deleted twice ok", f.r.Report(id, 0, rc.Token, "obj", []string{"a"}, nil, worker), nil)
	reason(t, "report not deleting", f.r.Report(id, 0, rc.Token, "db", nil, []string{"b"}, worker), ReasonNotInChunk)
	f.must(f.r.Report(id, 0, rc.Token, "db", []string{"b"}, nil, worker))
	if !f.r.artifacts["b"].deleted || f.run(id).Counts[OutcomeDeleted] != 2 {
		t.Error("current-token deletion of a held artifact is recorded as fact")
	}
}
