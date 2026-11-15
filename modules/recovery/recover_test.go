package recovery

import (
	"strings"
	"testing"
	"time"
)

func TestFullRecovery(t *testing.T) {
	f := standard(t)
	old := f.permit("i1", erp)
	f.must(second(f.r.Call("i1", "w-erp", old, numbered("inv-7", 7), erp)))
	done, _ := f.r.Call("i1", "w-erp", old, target("pay-1"), erp)
	f.must(second(f.r.Done("i1", done.ID, OutcomeConfirmed, erp)))
	del := f.must2(f.r.Record("i1", Event{Kind: EventPurge, Domain: DomainEvidence, Key: "bundle-1"}, erp))
	f.advance(time.Minute)
	rev := f.must2(f.r.Record("i1", Event{Kind: EventRevokeAccess, Domain: DomainDatabase, Key: "user-9"}, ver))
	f.advance(time.Minute)
	f.begin("i1", start.Add(-time.Hour))
	reason(t, "old worker", second(f.r.Call("i1", "w-erp", old, target("x"), erp)), ReasonFenced)
	if fenced, _ := f.r.Fenced("i1", "ws", "pay-1"); !fenced {
		t.Error("confirmed call not auto-fenced")
	}
	if hw, _ := f.r.HighWater("i1", "ws", "inv"); hw != 7 {
		t.Errorf("open call high water %d", hw)
	}
	v, err := f.r.Recovery("i1")
	f.must(err)
	if v.Phase != PhaseVerify || len(v.Pending) != 2 || len(v.Unknown) != 1 || v.Unknown[0].Key != "inv-7" || v.Open != 0 || len(v.Domains) != 9 || len(v.Externals) != 2 {
		t.Fatalf("view %+v", v)
	}
	reason(t, "not verified", second(f.r.Reapplied("i1", del.Sequence, ver)), ReasonPhase)
	reason(t, "unlock early", f.unlock("i1"), ReasonIncomplete)
	f.verifyAll("i1", ver)
	reason(t, "out of order", second(f.r.Reapplied("i1", rev.Sequence, ver)), ReasonOutOfOrder)
	f.must(second(f.r.Reapplied("i1", del.Sequence, ver)))
	f.must(second(f.r.Reapplied("i1", del.Sequence, ver)))
	reason(t, "report after reapply", second(f.r.Report("i1", manifest()[0], ver)), ReasonPhase)
	reason(t, "must apply", second(f.r.Unapplicable("i1", rev.Sequence, "gone", ver)), ReasonMustApply)
	f.must(second(f.r.Reapplied("i1", rev.Sequence, ver)))
	reason(t, "not reconciled", f.unlock("i1"), ReasonIncomplete)
	f.advance(10 * time.Minute)
	f.reconcileAll("i1", ver, start.Add(-2*time.Hour), f.at())
	if v, _ := f.r.Recovery("i1"); v.Phase != PhaseReady || v.Externals[0].Status != ExternalReconciled {
		t.Errorf("ready %+v", v)
	}
	f.must(f.unlock("i1"))
	s := f.r.installations["i1"]
	if s.state != StateServing || s.current != nil || len(s.history) != 1 || s.history[0].Outcome != "completed" {
		t.Fatalf("unlocked %+v", s.history)
	}
	tok := f.permit("i1", erp)
	reason(t, "fenced key", second(f.r.Call("i1", "w-erp", tok, target("pay-1"), erp)), ReasonFencedKey)
	reason(t, "below high water", second(f.r.Call("i1", "w-erp", tok, numbered("inv-8", 7), erp)), ReasonBelowHighWater)
	f.must(second(f.r.Call("i1", "w-erp", tok, numbered("inv-8", 8), erp)))
	expect(t, "no recovery", second(f.r.Recovery("i1")), ErrNotFound)
	h, err := f.r.History("i1")
	if err != nil || len(h) != 1 || h[0].Recovery != "recovery-1" {
		t.Errorf("history %+v", h)
	}
	expect(t, "history site", second(f.r.History("i9")), ErrNotFound)
	expect(t, "view site", second(f.r.Recovery("i9")), ErrNotFound)
}

func (f world) must2(v EventView, err error) EventView {
	f.t.Helper()
	f.must(err)
	return v
}

func TestReports(t *testing.T) {
	f := standard(t)
	reason(t, "not recovering", second(f.r.Report("i1", manifest()[0], ver)), ReasonNotRecovering)
	f.begin("i1", start)
	expect(t, "form", second(f.r.Report("i1", Inventory{Domain: "x"}, ver)), ErrInvalid)
	expect(t, "invalid", second(f.r.Report("i1", Inventory{Domain: DomainKeys, Items: 3}, ver)), ErrInvalid)
	expect(t, "actor", second(f.r.Report("i1", manifest()[0], Actor{ID: "x"})), ErrInvalid)
	expect(t, "site", second(f.r.Report("i9", manifest()[0], ver)), ErrNotFound)
	reason(t, "verifier", second(f.r.Report("i1", manifest()[0], erp)), ReasonNotVerifier)
	bad := manifest()[2]
	bad.Chunks[1].Checksum = sum("tampered")
	st, err := f.r.Report("i1", bad, ver)
	if err != nil || st.Status != DomainMismatch || len(st.Mismatch) != 1 || st.Mismatch[0] != 1 {
		t.Fatalf("mismatch %+v", st)
	}
	short := Inventory{Domain: DomainKeys, Items: 4, Chunks: []Chunk{{Start: 0, End: 4, Checksum: sum("keysa")}}}
	st, _ = f.r.Report("i1", short, ver)
	if st.Status != DomainMismatch || len(st.Mismatch) != 1 || st.Mismatch[0] != 1 {
		t.Errorf("short %+v", st)
	}
	long := manifest()[2]
	long.Items = 11
	long.Chunks = append(long.Chunks, Chunk{Start: 10, End: 11, Checksum: sum("c")})
	st, _ = f.r.Report("i1", long, ver)
	if st.Status != DomainMismatch || len(st.Mismatch) != 1 || st.Mismatch[0] != 2 {
		t.Errorf("long %+v", st)
	}
	split := manifest()[2]
	split.Chunks = []Chunk{{Start: 0, End: 4, Checksum: sum("keysa")}, {Start: 4, End: 10, Checksum: sum("keysb")}}
	st, _ = f.r.Report("i1", split, ver)
	if st.Status != DomainVerified {
		t.Errorf("repair %+v", st)
	}
	st.Mismatch = append(st.Mismatch, 9)
	many := Inventory{Domain: DomainOutbox, Items: 200}
	for index := 0; index < 200; index++ {
		many.Chunks = append(many.Chunks, Chunk{Start: uint64(index), End: uint64(index + 1), Checksum: sum("z")})
	}
	st, _ = f.r.Report("i1", many, ver)
	if len(st.Mismatch) != maxMismatch {
		t.Errorf("mismatch cap %d", len(st.Mismatch))
	}
	empty := setup(t)
	empty.installation("i1", "")
	m := manifest()
	m[8] = Inventory{Domain: DomainDeletions}
	empty.approve(Change{Kind: ChangeBegin, Begin: RestorePoint{Installation: "i1", Generation: 1, Backup: sum("b"), ManifestDigest: manifestDigest(m), AsOf: start, Manifest: m}})
	if st, err := empty.r.Report("i1", Inventory{Domain: DomainDeletions}, ver); err != nil || st.Status != DomainVerified {
		t.Errorf("empty domain %+v %v", st, err)
	}
}

func TestReconcilePages(t *testing.T) {
	f := standard(t)
	reason(t, "not recovering", second(f.r.Reconcile("i1", "erp-x", Page{Number: 1, From: start, Through: start}, ver)), ReasonNotRecovering)
	f.begin("i1", start)
	f.advance(10 * time.Minute)
	now := f.at()
	forms := map[string]Page{
		"number":  {Number: 0, From: start, Through: now},
		"from":    {Number: 1, Through: now},
		"order":   {Number: 1, From: now, Through: start},
		"future":  {Number: 1, From: start, Through: now.Add(time.Second)},
		"item":    {Number: 1, From: start, Through: now, Items: []Item{{Workspace: "ws", Key: "k", Kind: "lost", At: start}}},
		"at":      {Number: 1, From: start, Through: now, Items: []Item{{Workspace: "ws", Key: "k", Kind: ItemPaid}}},
		"at late": {Number: 1, From: start, Through: now, Items: []Item{{Workspace: "ws", Key: "k", Kind: ItemPaid, At: now.Add(time.Second)}}},
		"series":  {Number: 1, From: start, Through: now, Items: []Item{{Workspace: "ws", Key: "k", Kind: ItemIssued, At: start, Series: "inv"}}},
		"many":    {Number: 1, From: start, Through: now, Items: make([]Item, maxItems+1)},
	}
	for name, p := range forms {
		expect(t, name, second(f.r.Reconcile("i1", "erp-x", p, ver)), ErrInvalid)
	}
	expect(t, "external form", second(f.r.Reconcile("i1", "", Page{Number: 1, From: start, Through: now}, ver)), ErrInvalid)
	expect(t, "unknown external", second(f.r.Reconcile("i1", "zz", Page{Number: 1, From: start, Through: now}, ver)), ErrNotFound)
	reason(t, "writer reports", second(f.r.Reconcile("i1", "erp-x", Page{Number: 1, From: start, Through: now}, erp)), ReasonNotVerifier)
	first := Page{Number: 1, From: start.Add(-time.Hour), Through: start.Add(time.Minute), Items: []Item{
		{Workspace: "ws", Key: "inv-50", Kind: ItemIssued, At: start.Add(-2 * time.Hour), Series: "inv", Number: 50},
		{Workspace: "ws", Key: "pay-9", Kind: ItemPaid, At: start, Series: "x", Number: 3},
	}}
	st, err := f.r.Reconcile("i1", "erp-x", first, ver)
	if err != nil || st.Status != ExternalPending {
		t.Fatal(st, err)
	}
	if hw, _ := f.r.HighWater("i1", "ws", "inv"); hw != 50 {
		t.Errorf("early items applied %d", hw)
	}
	if hw, _ := f.r.HighWater("i1", "ws", "x"); hw != 0 {
		t.Error("paid item raised a high water")
	}
	if fenced, _ := f.r.Fenced("i1", "ws", "pay-9"); !fenced {
		t.Error("paid not fenced")
	}
	f.must(second(f.r.Reconcile("i1", "erp-x", first, ver)))
	changed := first
	changed.Through = start.Add(2 * time.Minute)
	expect(t, "changed repeat", second(f.r.Reconcile("i1", "erp-x", changed, ver)), ErrConflict)
	reason(t, "skip", second(f.r.Reconcile("i1", "erp-x", Page{Number: 3, From: start, Through: now}, ver)), ReasonOutOfOrder)
	reason(t, "gap", second(f.r.Reconcile("i1", "erp-x", Page{Number: 2, From: start.Add(5 * time.Minute), Through: now}, ver)), ReasonCoverage)
	reason(t, "short", second(f.r.Reconcile("i1", "erp-x", Page{Number: 2, From: start, Through: start.Add(2 * time.Minute), Final: true, Rotated: true}, ver)), ReasonCoverage)
	reason(t, "not rotated", second(f.r.Reconcile("i1", "erp-x", Page{Number: 2, From: start, Through: now, Final: true}, ver)), ReasonCoverage)
	final := Page{Number: 2, From: start, Through: start.Add(3*time.Minute + time.Second), Final: true, Rotated: true}
	reason(t, "below required", second(f.r.Reconcile("i1", "erp-x", Page{Number: 2, From: start, Through: start.Add(3*time.Minute - time.Nanosecond), Final: true, Rotated: true}, ver)), ReasonCoverage)
	st, err = f.r.Reconcile("i1", "erp-x", final, ver)
	if err != nil || st.Status != ExternalReconciled {
		t.Fatal(st, err)
	}
	if again, err := f.r.Reconcile("i1", "erp-x", final, ver); err != nil || again.Status != ExternalReconciled {
		t.Error("repeat final")
	}
	if again, err := f.r.Reconcile("i1", "erp-x", first, ver); err != nil || again.Status != ExternalReconciled {
		t.Error("repeat first after final")
	}
	reason(t, "after final", second(f.r.Reconcile("i1", "erp-x", Page{Number: 3, From: start, Through: now}, ver)), ReasonNotPending)
	reason(t, "late from", second(f.r.Reconcile("i1", "iss-x", Page{Number: 1, From: start, Through: now, Final: true, Rotated: true}, ver)), ReasonCoverage)
	f.must(second(f.r.Reconcile("i1", "iss-x", Page{Number: 1, From: start.Add(-time.Minute), Through: now, Final: true, Rotated: true}, ver)))
}

func TestReconcileInflightAndCaps(t *testing.T) {
	f := standard(t)
	tok := f.permit("i1", iss)
	f.advance(time.Minute)
	f.must(second(f.r.Call("i1", "w-iss", tok, target("slow"), iss)))
	f.begin("i1", start)
	s := f.r.installations["i1"]
	if !s.inflight.Equal(start.Add(3 * time.Minute)) {
		t.Fatalf("inflight %v", s.inflight)
	}
	s.inflight = start.Add(10 * time.Minute)
	f.advance(20 * time.Minute)
	reason(t, "inflight", second(f.r.Reconcile("i1", "erp-x", Page{Number: 1, From: start.Add(-time.Hour), Through: start.Add(10 * time.Minute), Final: true, Rotated: true}, ver)), ReasonCoverage)
	f.must(second(f.r.Reconcile("i1", "erp-x", Page{Number: 1, From: start.Add(-time.Hour), Through: start.Add(11 * time.Minute), Final: true, Rotated: true}, ver)))
	s.fenced = map[[2]string]bool{}
	for index := 0; index < maxKeys; index++ {
		s.fenced[[2]string{"ws", "k" + itoa(index)}] = true
	}
	page := Page{Number: 1, From: start.Add(-time.Hour), Through: start.Add(time.Minute), Items: []Item{{Workspace: "ws", Key: "new", Kind: ItemDelivered, At: start}}}
	expect(t, "key cap", second(f.r.Reconcile("i1", "iss-x", page, ver)), ErrLimit)
	page.Items[0].Key = "k1"
	f.must(second(f.r.Reconcile("i1", "iss-x", page, ver)))
	s.high = map[[2]string]int64{}
	for index := 0; index < maxSeries; index++ {
		s.high[[2]string{"ws", "s" + itoa(index)}] = 1
	}
	two := Page{Number: 2, From: start, Through: start.Add(2 * time.Minute), Items: []Item{{Workspace: "ws", Key: "k2", Kind: ItemIssued, At: start, Series: "new", Number: 1}}}
	expect(t, "series cap", second(f.r.Reconcile("i1", "iss-x", two, ver)), ErrLimit)
	two.Items[0].Series = "s1"
	f.must(second(f.r.Reconcile("i1", "iss-x", two, ver)))
	pg := s.current.externals["iss-x"]
	pg.next = maxPages + 1
	expect(t, "pages", second(f.r.Reconcile("i1", "iss-x", Page{Number: maxPages + 1, From: start, Through: start.Add(time.Minute)}, ver)), ErrLimit)
	f.r.keys = maxKeysTotal
	if f.r.room(s, 1, 0) {
		t.Error("global key cap")
	}
}

func TestEvents(t *testing.T) {
	f := standard(t)
	expect(t, "kind", second(f.r.Record("i1", Event{Kind: "x", Domain: DomainEvidence, Key: "k"}, erp)), ErrInvalid)
	expect(t, "domain", second(f.r.Record("i1", Event{Kind: EventDelete, Domain: "x", Key: "k"}, erp)), ErrInvalid)
	expect(t, "key", second(f.r.Record("i1", Event{Kind: EventDelete, Domain: DomainEvidence}, erp)), ErrInvalid)
	expect(t, "actor", second(f.r.Record("i1", Event{Kind: EventDelete, Domain: DomainEvidence, Key: "k"}, Actor{ID: "x"})), ErrInvalid)
	expect(t, "site", second(f.r.Record("i9", Event{Kind: EventDelete, Domain: DomainEvidence, Key: "k"}, erp)), ErrNotFound)
	reason(t, "human", second(f.r.Record("i1", Event{Kind: EventDelete, Domain: DomainEvidence, Key: "k"}, ann)), ReasonNotWriter)
	reason(t, "foreign", second(f.r.Record("i1", Event{Kind: EventDelete, Domain: DomainEvidence, Key: "k"}, Actor{ID: "b-w-erp", Kind: KindSystem})), ReasonNotWriter)
	before := f.must2(f.r.Record("i1", Event{Kind: EventHold, Domain: DomainEvidence, Key: "old"}, erp))
	f.advance(2 * time.Hour)
	hold := f.must2(f.r.Record("i1", Event{Kind: EventHold, Domain: DomainEvidence, Key: "b-1"}, erp))
	if !hold.At.Equal(start.Add(2*time.Hour)) || hold.Sequence != 2 {
		t.Errorf("stamped %+v", hold)
	}
	extend := f.must2(f.r.Record("i1", Event{Kind: EventExtend, Domain: DomainEvidence, Key: "b-2"}, erp))
	f.advance(time.Minute)
	f.begin("i1", start.Add(time.Hour+time.Minute))
	during := f.must2(f.r.Record("i1", Event{Kind: EventDelete, Domain: DomainEvidence, Key: "b-3"}, ver))
	v, _ := f.r.Recovery("i1")
	if len(v.Pending) != 2 || v.Pending[0].Sequence != hold.Sequence {
		t.Fatalf("pending %+v", v.Pending)
	}
	f.verifyAll("i1", ver)
	expect(t, "outside window", second(f.r.Reapplied("i1", before.Sequence, ver)), ErrNotFound)
	expect(t, "during", second(f.r.Reapplied("i1", during.Sequence, ver)), ErrNotFound)
	expect(t, "reason", second(f.r.Unapplicable("i1", hold.Sequence, "", ver)), ErrInvalid)
	reason(t, "unapplicable out of order", second(f.r.Unapplicable("i1", extend.Sequence, "bundle missing", ver)), ReasonOutOfOrder)
	f.must(second(f.r.Unapplicable("i1", hold.Sequence, "bundle missing", ver)))
	f.must(second(f.r.Unapplicable("i1", hold.Sequence, "bundle missing", ver)))
	expect(t, "applied after lost", second(f.r.Reapplied("i1", hold.Sequence, ver)), ErrConflict)
	f.must(second(f.r.Reapplied("i1", extend.Sequence, ver)))
	expect(t, "lost after applied", second(f.r.Unapplicable("i1", extend.Sequence, "x", ver)), ErrConflict)
	f.advance(10 * time.Minute)
	f.reconcileAll("i1", ver, start, f.at())
	s := f.r.installations["i1"]
	reason(t, "lost digest", f.try(Change{Kind: ChangeUnlock, Unlock: Unlock{Installation: "i1", Generation: s.generation, Lost: lostDigest(nil)}}), ReasonLost)
	v, _ = f.r.Recovery("i1")
	if len(v.Lost) != 1 || v.Lost[0] != hold.Sequence {
		t.Errorf("lost %+v", v.Lost)
	}
	f.must(f.unlock("i1"))
	if h, _ := f.r.History("i1"); h[0].Lost != 1 {
		t.Error("history lost count")
	}
	g := standard(t)
	g.r.eventsTotal = maxEventsTotal
	expect(t, "total events", second(g.r.Record("i1", Event{Kind: EventDelete, Domain: DomainEvidence, Key: "k"}, erp)), ErrLimit)
	g.r.eventsTotal = 0
	gs := g.r.installations["i1"]
	for index := 0; index < maxEvents; index++ {
		gs.events = append(gs.events, &event{})
	}
	expect(t, "events", second(g.r.Record("i1", Event{Kind: EventDelete, Domain: DomainEvidence, Key: "k"}, erp)), ErrLimit)
}

func TestUnlockConditions(t *testing.T) {
	f := standard(t)
	f.begin("i1", start)
	f.verifyAll("i1", ver)
	f.advance(10 * time.Minute)
	f.reconcileAll("i1", ver, start.Add(-time.Hour), f.at())
	f.approve(Change{Kind: ChangeWriter, Writer: Writer{Installation: "i1", ID: "w-pay", Kind: WriterPayment, MaxCall: time.Minute}})
	reason(t, "kind coverage", f.unlock("i1"), ReasonKindCoverage)
	f.approve(Change{Kind: ChangeExternal, External: External{Installation: "i1", ID: "pay-x", Kind: WriterPayment}})
	reason(t, "new external pending", f.unlock("i1"), ReasonIncomplete)
	f.must(second(f.r.Reconcile("i1", "pay-x", Page{Number: 1, From: start.Add(-time.Hour), Through: f.at(), Final: true, Rotated: true}, ver)))
	s := f.r.installations["i1"]
	reason(t, "generation", f.try(Change{Kind: ChangeUnlock, Unlock: Unlock{Installation: "i1", Generation: 1, Lost: lostDigest(nil)}}), ReasonGeneration)
	s.current.domains[DomainKeys].Status = DomainMismatch
	reason(t, "domain", f.unlock("i1"), ReasonIncomplete)
	s.current.domains[DomainKeys].Status = DomainVerified
	f.must(f.unlock("i1"))
}

func TestAbandonAndPrune(t *testing.T) {
	f := standard(t)
	f.must2(f.r.Record("i1", Event{Kind: EventDelete, Domain: DomainEvidence, Key: "k1"}, erp))
	f.advance(time.Hour)
	reason(t, "abandon nothing", f.try(Change{Kind: ChangeAbandon, Abandon: Abandon{Installation: "i1", Recovery: "recovery-1", Generation: f.r.installations["i1"].generation}}), ReasonNoRestorePoint)
	f.begin("i1", start.Add(-time.Minute))
	reason(t, "prune open", f.try(Change{Kind: ChangePrune, Prune: Prune{Installation: "i1", Before: start}}), ReasonOpenRecovery)
	reason(t, "abandon other", f.try(Change{Kind: ChangeAbandon, Abandon: Abandon{Installation: "i1", Recovery: "recovery-9", Generation: f.r.installations["i1"].generation}}), ReasonNoRestorePoint)
	stopped := f.r.installations["i1"].stoppedAt
	f.approve(Change{Kind: ChangeAbandon, Abandon: Abandon{Installation: "i1", Recovery: "recovery-1", Generation: f.r.installations["i1"].generation}})
	s := f.r.installations["i1"]
	if s.state != StateRecovering || s.current == nil || s.current.point != nil || s.generation != 3 || !s.stoppedAt.Equal(stopped) {
		t.Fatalf("abandoned %+v", s)
	}
	reason(t, "unlock after abandon", f.unlock("i1"), ReasonNoRestorePoint)
	reason(t, "report after abandon", second(f.r.Report("i1", manifest()[0], ver)), ReasonNoRestorePoint)
	v, _ := f.r.Recovery("i1")
	if v.ID != "recovery-1" || len(v.Domains) != 0 || v.Phase != PhaseVerify {
		t.Errorf("abandoned view %+v", v)
	}
	f.advance(time.Minute)
	rev := f.must2(f.r.Record("i1", Event{Kind: EventRevokeAccess, Domain: DomainDatabase, Key: "u"}, ver))
	f.advance(time.Minute)
	f.begin("i1", start.Add(-time.Minute))
	v, _ = f.r.Recovery("i1")
	if v.ID != "recovery-1" || len(v.Pending) != 2 || v.Pending[1].Sequence != rev.Sequence {
		t.Fatalf("second attempt %+v", v.Pending)
	}
	if h, _ := f.r.History("i1"); len(h) != 1 || h[0].Outcome != "abandoned" {
		t.Errorf("history %+v", h)
	}
	g := standard(t)
	g.must2(g.r.Record("i1", Event{Kind: EventDelete, Domain: DomainEvidence, Key: "k1"}, erp))
	g.advance(time.Hour)
	g.must2(g.r.Record("i1", Event{Kind: EventDelete, Domain: DomainEvidence, Key: "k2"}, erp))
	expect(t, "future prune", g.try(Change{Kind: ChangePrune, Prune: Prune{Installation: "i1", Before: g.at().Add(time.Second)}}), ErrInvalid)
	g.approve(Change{Kind: ChangePrune, Prune: Prune{Installation: "i1", Before: start.Add(time.Minute)}})
	gs := g.r.installations["i1"]
	if len(gs.events) != 1 || g.r.eventsTotal != 1 || !gs.pruned.Equal(start.Add(time.Minute)) {
		t.Fatalf("pruned %d %d", len(gs.events), g.r.eventsTotal)
	}
	g.approve(Change{Kind: ChangePrune, Prune: Prune{Installation: "i1", Before: start}})
	if !gs.pruned.Equal(start.Add(time.Minute)) {
		t.Error("pruned boundary moved back")
	}
	reason(t, "pruned begin", g.try(Change{Kind: ChangeBegin, Begin: g.point("i1", start.Add(2*time.Minute))}), ReasonPruned)
	g.begin("i1", start.Add(2*time.Minute+time.Nanosecond))
}

func TestRecoveryHistoryBound(t *testing.T) {
	f := standard(t)
	s := f.r.installations["i1"]
	for index := 0; index < maxHistory; index++ {
		s.history = append(s.history, History{Recovery: "r" + itoa(index)})
	}
	f.begin("i1", start)
	f.approve(Change{Kind: ChangeAbandon, Abandon: Abandon{Installation: "i1", Recovery: "recovery-1", Generation: f.r.installations["i1"].generation}})
	if len(s.history) != maxHistory || s.history[0].Recovery != "r1" || !strings.HasPrefix(s.history[maxHistory-1].Recovery, "recovery") {
		t.Error("history bound")
	}
}
