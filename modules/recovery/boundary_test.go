package recovery

import (
	"testing"
	"time"
)

func TestDigestKnownAnswers(t *testing.T) {
	inv := []Inventory{{Domain: DomainKeys, Items: 30, Chunks: []Chunk{{Start: 0, End: 12, Checksum: sum("a")}, {Start: 12, End: 30, Checksum: sum("b")}}}}
	if got := manifestDigest(inv); got != "7f52cfb0bb222f2250100cebb4f4836506678cbb43ed26e2310d704a23f873c4" {
		t.Errorf("manifest %s", got)
	}
	if got := lostDigest([]uint64{11, 12}); got != "5b48f63d5210359d414fe1febfa1304c85400fb2bc09f9aa3c3f0e7ae247fbba" {
		t.Errorf("lost %s", got)
	}
	if maxIDBytes != 128 || maxTextBytes != 200 {
		t.Error("text bounds")
	}
}

func TestCapsAtTheEdge(t *testing.T) {
	f := standard(t)
	for len(f.r.installations) < maxInstallations-1 {
		f.r.installations["x"+itoa(len(f.r.installations))] = &site{}
	}
	f.approve(Change{Kind: ChangeInstallation, Installation: Installation{ID: "last", Tenant: "t"}})
	expect(t, "full", f.try(Change{Kind: ChangeInstallation, Installation: Installation{ID: "over", Tenant: "t"}}), ErrLimit)
	g := standard(t)
	var toks []Token
	for index := 0; index <= maxLive; index++ {
		toks = append(toks, g.permit("i1", erp))
	}
	if w := g.r.installations["i1"].writers["w-erp"]; len(w.live) != maxLive {
		t.Errorf("live %d", len(w.live))
	}
	reason(t, "oldest evicted", second(g.r.Call("i1", "w-erp", toks[0], target("a"), erp)), ReasonFenced)
	g.must(second(g.r.Call("i1", "w-erp", toks[1], target("a"), erp)))
	h := standard(t)
	tok := h.permit("i1", erp)
	var last CallView
	for index := 0; index < maxOpenPerWriter; index++ {
		c, err := h.r.Call("i1", "w-erp", tok, target("k"+itoa(index)), erp)
		h.must(err)
		last = c
	}
	if last.ID != "call-1000" {
		t.Errorf("call id %s", last.ID)
	}
	expect(t, "per writer", second(h.r.Call("i1", "w-erp", tok, target("over"), erp)), ErrLimit)
	k := standard(t)
	ks := k.r.installations["i1"]
	for index := 0; index < maxOpen-1; index++ {
		ks.open["x"+itoa(index)] = &call{view: CallView{ID: "x" + itoa(index), Writer: "w-iss", Deadline: start.Add(time.Hour)}}
	}
	ktok := k.permit("i1", erp)
	k.must(second(k.r.Call("i1", "w-erp", ktok, target("last"), erp)))
	expect(t, "total", second(k.r.Call("i1", "w-erp", ktok, target("over"), erp)), ErrLimit)
	if hw, err := k.r.HighWater("", "ws", "inv"); err == nil || hw != 0 {
		t.Error("high water on error")
	}
	if fenced, err := k.r.Fenced("i9", "ws", "k"); err == nil || fenced {
		t.Error("fenced on error")
	}
	m := standard(t)
	ms := m.r.installations["i1"]
	m.r.keys = maxKeysTotal - 1
	if !m.r.room(ms, 1, 0) || m.r.room(ms, 2, 0) {
		t.Error("total key room")
	}
	m.r.keys = 0
	for len(ms.fenced) < maxKeys-1 {
		ms.fenced[[2]string{"z", itoa(len(ms.fenced))}] = true
	}
	for len(ms.high) < maxSeries-1 {
		ms.high[[2]string{"z", itoa(len(ms.high))}] = 1
	}
	m.must(second(m.r.Call("i1", "w-erp", m.permit("i1", erp), numbered("edge", 3), erp)))
	m.begin("i1", start)
	if !ms.fenced[[2]string{"ws", "edge"}] || ms.high[[2]string{"ws", "inv"}] != 3 || m.r.keys != 1 {
		t.Error("auto-fence at the edge")
	}
}

func TestAutoFenceSelection(t *testing.T) {
	f := standard(t)
	tok := f.permit("i1", erp)
	early, _ := f.r.Call("i1", "w-erp", tok, target("early"), erp)
	f.must(second(f.r.Done("i1", early.ID, OutcomeConfirmed, erp)))
	f.advance(10 * time.Minute)
	tok = f.permit("i1", erp)
	failed, _ := f.r.Call("i1", "w-erp", tok, target("failed"), erp)
	f.must(second(f.r.Done("i1", failed.ID, OutcomeFailed, erp)))
	ok, _ := f.r.Call("i1", "w-erp", tok, target("ok"), erp)
	f.must(second(f.r.Done("i1", ok.ID, OutcomeConfirmed, erp)))
	f.begin("i1", start.Add(5*time.Minute))
	s := f.r.installations["i1"]
	if s.fenced[[2]string{"ws", "early"}] || s.fenced[[2]string{"ws", "failed"}] || !s.fenced[[2]string{"ws", "ok"}] {
		t.Errorf("auto-fence %v", s.fenced)
	}
}

func TestReconcileEdges(t *testing.T) {
	f := standard(t)
	f.approve(Change{Kind: ChangeWriter, Writer: Writer{Installation: "i1", ID: "w-slow", Kind: WriterERP, MaxCall: time.Hour}})
	f.must(f.r.RemoveWriter("i1", "w-slow", ann))
	f.begin("i1", start)
	f.advance(10 * time.Minute)
	page := Page{Number: 1, From: start.Add(-time.Hour), Through: start.Add(3 * time.Minute), Final: true, Rotated: true}
	f.must(second(f.r.Reconcile("i1", "erp-x", page, ver)))
	items := make([]Item, maxItems)
	for index := range items {
		items[index] = Item{Workspace: "ws", Key: "k" + itoa(index), Kind: ItemDelivered, At: start, Series: "s", Number: 1}
	}
	f.must(second(f.r.Reconcile("i1", "iss-x", Page{Number: 1, From: start.Add(-time.Hour), Through: start, Items: items}, ver)))
	s := f.r.installations["i1"]
	if len(s.high) != 0 {
		t.Error("delivered items raised a high water")
	}
	paid := Page{Number: 2, From: start, Through: start, Items: []Item{{Workspace: "ws", Key: "p", Kind: ItemPaid, At: start, Series: "s", Number: 9}}}
	for len(s.high) < maxSeries {
		s.high[[2]string{"z", itoa(len(s.high))}] = 1
	}
	f.must(second(f.r.Reconcile("i1", "iss-x", paid, ver)))
	delete(s.high, [2]string{"z", "0"})
	sub := Page{Number: 3, From: start, Through: start, Items: []Item{{Workspace: "ws", Key: "sub", Kind: ItemSubmitted, At: start, Series: "erp", Number: 4}}}
	f.must(second(f.r.Reconcile("i1", "iss-x", sub, ver)))
	if s.high[[2]string{"ws", "erp"}] != 4 {
		t.Error("submitted item did not raise a high water")
	}
	pg := s.current.externals["iss-x"]
	pg.next = maxPages
	pg.digests = append(pg.digests, make([]string, maxPages-1-len(pg.digests))...)
	f.must(second(f.r.Reconcile("i1", "iss-x", Page{Number: maxPages, From: start, Through: start}, ver)))
}

func TestMustApplyRevokeKey(t *testing.T) {
	f := standard(t)
	ev, _ := f.r.Record("i1", Event{Kind: EventRevokeKey, Domain: DomainKeys, Key: "k"}, erp)
	f.advance(time.Minute)
	f.begin("i1", start)
	f.verifyAll("i1", ver)
	reason(t, "revoke key", second(f.r.Unapplicable("i1", ev.Sequence, "gone", ver)), ReasonMustApply)
}

func TestFirstChunkMismatch(t *testing.T) {
	f := standard(t)
	f.begin("i1", start)
	bad := manifest()[1]
	bad.Chunks[0].Checksum = sum("x")
	st, _ := f.r.Report("i1", bad, ver)
	if len(st.Mismatch) != 1 || st.Mismatch[0] != 0 {
		t.Errorf("first chunk %+v", st)
	}
}

func TestProposalCapsAndHistory(t *testing.T) {
	f := standard(t)
	cap := (maxProposals + 3) / 4
	proposers := []Actor{ann, bob, cat, {ID: "dan", Kind: KindHuman}}
	for _, actor := range proposers {
		for index := 0; index < cap; index++ {
			f.must(second(f.r.Propose(Change{Kind: ChangeExternal, External: External{Installation: "i1", ID: actor.ID + itoa(index), Kind: WriterERP}}, actor)))
		}
	}
	expect(t, "per proposer", second(f.r.Propose(Change{Kind: ChangeExternal, External: External{Installation: "i1", ID: "ann-over", Kind: WriterERP}}, ann)), ErrLimit)
	expect(t, "scope", second(f.r.Propose(Change{Kind: ChangeExternal, External: External{Installation: "i1", ID: "eve-1", Kind: WriterERP}}, Actor{ID: "eve", Kind: KindHuman})), ErrLimit)
	g := setup(t)
	g.approve(Change{Kind: ChangeInstallation, Installation: Installation{ID: "i1", Tenant: "t"}})
	g.must(second(g.r.Propose(Change{Kind: ChangeExternal, External: External{Installation: "i1", ID: "a", Kind: WriterERP}}, ann)))
	g.must(second(g.r.Propose(Change{Kind: ChangeExternal, External: External{Installation: "i1", ID: "b", Kind: WriterERP}}, bob)))
	h := setup(t)
	var numbers []uint64
	for index := 0; index < maxProposals+2; index++ {
		numbers = append(numbers, h.approve(Change{Kind: ChangeInstallation, Installation: Installation{ID: "s" + itoa(index), Tenant: "t"}}).Number)
	}
	expect(t, "first pruned", second(h.r.Proposal(numbers[1])), ErrNotFound)
	h.must(second(h.r.Proposal(numbers[2])))
	k := standard(t)
	ks := k.r.installations["i1"]
	for index := 0; index < maxHistory-1; index++ {
		ks.history = append(ks.history, History{Recovery: "r"})
	}
	k.begin("i1", start)
	k.approve(Change{Kind: ChangeAbandon, Abandon: Abandon{Installation: "i1", Recovery: "recovery-1", Generation: k.r.installations["i1"].generation}})
	if len(ks.history) != maxHistory {
		t.Errorf("history %d", len(ks.history))
	}
}

func TestFinalBoundaries(t *testing.T) {
	f := standard(t)
	limit := (maxProposals + 3) / 4
	for index := 0; index < limit; index++ {
		f.must(second(f.r.Propose(Change{Kind: ChangeExternal, External: External{Installation: "i1", ID: "e" + itoa(index), Kind: WriterERP}}, ann)))
	}
	expect(t, "per proposer edge", second(f.r.Propose(Change{Kind: ChangeExternal, External: External{Installation: "i1", ID: "over", Kind: WriterERP}}, ann)), ErrLimit)
	f.must(second(f.r.Propose(Change{Kind: ChangeExternal, External: External{Installation: "i1", ID: "other", Kind: WriterERP}}, bob)))
	if hw, err := f.r.HighWater("i1", "", "inv"); err == nil || hw != 0 {
		t.Error("high water form")
	}
	g := standard(t)
	g.begin("i1", start)
	g.verifyAll("i1", ver)
	if v, _ := g.r.Recovery("i1"); v.Phase != PhaseReady {
		t.Errorf("nothing to reapply %s", v.Phase)
	}
	h := standard(t)
	h.must2(h.r.Record("i1", Event{Kind: EventDelete, Domain: DomainEvidence, Key: "one"}, erp))
	h.advance(time.Minute)
	h.begin("i1", start.Add(-time.Minute))
	h.verifyAll("i1", ver)
	if v, _ := h.r.Recovery("i1"); v.Phase != PhaseVerify {
		t.Errorf("one pending %s", v.Phase)
	}
	k := standard(t)
	ks := k.r.installations["i1"]
	k.must(second(k.r.Call("i1", "w-erp", k.permit("i1", erp), target("dup"), erp)))
	ks.fenced[[2]string{"ws", "dup"}] = true
	for len(ks.fenced) < maxKeys {
		ks.fenced[[2]string{"z", itoa(len(ks.fenced))}] = true
	}
	k.begin("i1", start)
	if v, _ := k.r.Recovery("i1"); v.Skipped != 0 {
		t.Errorf("already fenced key skipped %d", v.Skipped)
	}
}
