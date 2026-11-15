package recovery

import (
	"sync"
	"testing"
	"time"
)

func TestAutoFenceByOutcomeTime(t *testing.T) {
	f := standard(t)
	tok := f.permit("i1", iss)
	c, _ := f.r.Call("i1", "w-iss", tok, target("slow"), iss)
	f.advance(90 * time.Second)
	f.must(second(f.r.Done("i1", c.ID, OutcomeConfirmed, iss)))
	f.begin("i1", start.Add(time.Minute))
	if fenced, _ := f.r.Fenced("i1", "ws", "slow"); !fenced {
		t.Error("call confirmed after AsOf − Skew was not fenced")
	}
}

func TestRemovedWritersStillCount(t *testing.T) {
	f := standard(t)
	f.approve(Change{Kind: ChangeWriter, Writer: Writer{Installation: "i1", ID: "w-pay", Kind: WriterPayment, MaxCall: time.Hour}})
	f.must(f.r.Lock("i1", "stop", ann))
	f.must(f.r.RemoveWriter("i1", "w-pay", ann))
	f.begin("i1", start)
	f.verifyAll("i1", ver)
	f.advance(2 * time.Hour)
	reason(t, "short coverage", second(f.r.Reconcile("i1", "erp-x", Page{Number: 1, From: start.Add(-time.Hour), Through: start.Add(time.Hour), Final: true, Rotated: true}, ver)), ReasonCoverage)
	f.must(second(f.r.Reconcile("i1", "erp-x", Page{Number: 1, From: start.Add(-time.Hour), Through: start.Add(time.Hour + time.Minute), Final: true, Rotated: true}, ver)))
	f.must(second(f.r.Reconcile("i1", "iss-x", Page{Number: 1, From: start.Add(-time.Hour), Through: f.at(), Final: true, Rotated: true}, ver)))
	reason(t, "removed kind", f.unlock("i1"), ReasonKindCoverage)
	f.approve(Change{Kind: ChangeExternal, External: External{Installation: "i1", ID: "pay-x", Kind: WriterPayment}})
	f.must(second(f.r.Reconcile("i1", "pay-x", Page{Number: 1, From: start.Add(-time.Hour), Through: f.at(), Final: true, Rotated: true}, ver)))
	f.must(f.unlock("i1"))
	g := standard(t)
	g.begin("i1", start)
	g.approve(Change{Kind: ChangeWriter, Writer: Writer{Installation: "i1", ID: "w-new", Kind: WriterERP, MaxCall: time.Hour}})
	g.advance(2 * time.Hour)
	reason(t, "new longer writer", second(g.r.Reconcile("i1", "erp-x", Page{Number: 1, From: start.Add(-time.Hour), Through: start.Add(time.Hour), Final: true, Rotated: true}, ver)), ReasonCoverage)
}

func TestExactCoverageBoundary(t *testing.T) {
	f := standard(t)
	f.begin("i1", start)
	f.advance(10 * time.Minute)
	f.must(second(f.r.Reconcile("i1", "erp-x", Page{Number: 1, From: start.Add(-time.Minute), Through: start.Add(3 * time.Minute), Final: true, Rotated: true}, ver)))
}

func TestAbandonGeneration(t *testing.T) {
	f := standard(t)
	f.begin("i1", start)
	reason(t, "stale abandon", f.try(Change{Kind: ChangeAbandon, Abandon: Abandon{Installation: "i1", Recovery: "recovery-1", Generation: 1}}), ReasonGeneration)
	expect(t, "abandon form", f.try(Change{Kind: ChangeAbandon, Abandon: Abandon{Installation: "i1", Recovery: "recovery-1"}}), ErrInvalid)
}

func TestRecoveryViewDetails(t *testing.T) {
	f := standard(t)
	tok := f.permit("i1", iss)
	c, _ := f.r.Call("i1", "w-iss", tok, numbered("open", 2), iss)
	f.begin("i1", start)
	v, _ := f.r.Recovery("i1")
	if len(v.OpenCalls) != 1 || v.OpenCalls[0].ID != c.ID || v.ManifestDigest != manifestDigest(manifest()) || v.PointGeneration != 1 || v.Skipped != 0 {
		t.Errorf("view %+v", v)
	}
	g := standard(t)
	gs := g.r.installations["i1"]
	for len(gs.high) < maxSeries {
		gs.high[[2]string{"z", itoa(len(gs.high))}] = 1
	}
	g.must(second(g.r.Call("i1", "w-iss", g.permit("i1", iss), numbered("x", 1), iss)))
	g.must(second(g.r.Call("i1", "w-iss", g.permit("i1", iss), target("plain"), iss)))
	g.begin("i1", start)
	gv, _ := g.r.Recovery("i1")
	if gv.Skipped != 1 || !gs.fenced[[2]string{"ws", "plain"}] {
		t.Errorf("skipped %d", gv.Skipped)
	}
	g.approve(Change{Kind: ChangeAbandon, Abandon: Abandon{Installation: "i1", Recovery: "recovery-1", Generation: gs.generation}})
	g.begin("i1", start)
	if gv, _ := g.r.Recovery("i1"); len(gv.Unknown) != 1 {
		t.Errorf("unknown entries across attempts %d", len(gv.Unknown))
	}
}

func TestCallsRaceBegin(t *testing.T) {
	f := standard(t)
	var wg sync.WaitGroup
	results := make(chan string, 1000)
	for worker := 0; worker < 4; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for step := 0; step < 50; step++ {
				tok, err := f.r.Permit("i1", "w-erp", erp)
				if err != nil {
					continue
				}
				key := "k" + itoa(worker*100+step)
				if _, err := f.r.Call("i1", "w-erp", tok, target(key), erp); err == nil {
					results <- key
				}
			}
		}(worker)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		p, err := f.r.Propose(Change{Kind: ChangeBegin, Begin: f.point("i1", start)}, ann)
		if err == nil {
			_, _ = f.r.Approve(p.Number, bob)
		}
	}()
	wg.Wait()
	close(results)
	s := f.r.installations["i1"]
	if s.state != StateRecovering || s.generation != 2 {
		t.Fatalf("begin not applied: %s %d", s.state, s.generation)
	}
	for key := range results {
		if !s.fenced[[2]string{"ws", key}] {
			t.Fatalf("call %s slipped past begin", key)
		}
	}
}

func TestWriterRemovedBeforeStop(t *testing.T) {
	f := standard(t)
	f.approve(Change{Kind: ChangeWriter, Writer: Writer{Installation: "i1", ID: "w-pay", Kind: WriterPayment, MaxCall: time.Minute}})
	f.must(f.r.RemoveWriter("i1", "w-pay", ann))
	f.must(f.r.Lock("i1", "stop", ann))
	f.begin("i1", start)
	f.verifyAll("i1", ver)
	f.advance(10 * time.Minute)
	f.reconcileAll("i1", ver, start.Add(-time.Hour), f.at())
	reason(t, "removed before stop", f.unlock("i1"), ReasonKindCoverage)
}
