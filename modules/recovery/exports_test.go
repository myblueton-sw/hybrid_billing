package recovery

import (
	"fmt"
	"sort"
	"testing"
	"time"
)

func snapshot(r *Registry) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := fmt.Sprint(r.counter, len(r.proposals), r.begins, r.keys, r.eventsTotal, len(r.identities))
	ids := []string{}
	for id := range r.installations {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		s := r.installations[id]
		out += fmt.Sprint("|", id, s.generation, s.state, s.stoppedAt, len(s.history), s.current != nil, len(s.writers), len(s.externals), s.verifier, len(s.events), s.pruned)
		if s.current != nil {
			out += fmt.Sprint(s.current.point != nil, len(s.current.lost), len(s.current.applied))
		}
	}
	numbers := []uint64{}
	for n := range r.proposals {
		numbers = append(numbers, n)
	}
	sort.Slice(numbers, func(a, b int) bool { return numbers[a] < numbers[b] })
	for _, n := range numbers {
		p := r.proposals[n]
		out += fmt.Sprint("#", n, p.view.State, p.view.Approver)
	}
	return out
}

func (f world) agree(name string, c Change, want error) {
	f.t.Helper()
	before := snapshot(f.r)
	got := f.r.Check(c)
	if after := snapshot(f.r); after != before {
		f.t.Errorf("%s: check changed state\n%s\n%s", name, before, after)
	}
	p, proposed := f.r.Propose(c, cat)
	if fmt.Sprint(got) != fmt.Sprint(proposed) || fmt.Sprint(got) != fmt.Sprint(want) {
		f.t.Errorf("%s: check %v propose %v want %s", name, got, proposed, want)
	}
	if proposed == nil {
		f.must(second(f.r.Withdraw(p.Number, cat)))
	}
}

func (f world) agreeApprove(name string, number uint64, approver Actor) error {
	f.t.Helper()
	p, err := f.r.Proposal(number)
	f.must(err)
	before := snapshot(f.r)
	got := f.r.Check(p.Change)
	if after := snapshot(f.r); after != before {
		f.t.Errorf("%s: check changed state", name)
	}
	_, want := f.r.Approve(number, approver)
	if fmt.Sprint(got) != fmt.Sprint(want) {
		f.t.Errorf("%s: check %v approve %v", name, got, want)
	}
	return want
}

func refused(reason string) error { return Refusal{Reason: reason} }

func unlockOf(id string, generation uint64, lost string) Change {
	return Change{Kind: ChangeUnlock, Unlock: Unlock{Installation: id, Generation: generation, Lost: lost}}
}

func (f world) status(id string) string {
	f.t.Helper()
	before := snapshot(f.r)
	state, generation, stopped, err := f.r.Status(id)
	if snapshot(f.r) != before {
		f.t.Errorf("status of %s changed state", id)
	}
	if err != nil {
		return err.Error()
	}
	return fmt.Sprint(state, " ", generation, " ", stopped.Format(time.RFC3339))
}

func TestExportedDigests(t *testing.T) {
	inv := []Inventory{{Domain: DomainKeys, Items: 30, Chunks: []Chunk{{Start: 0, End: 12, Checksum: sum("a")}, {Start: 12, End: 30, Checksum: sum("b")}}}}
	if got := ManifestDigest(inv); got != "7f52cfb0bb222f2250100cebb4f4836506678cbb43ed26e2310d704a23f873c4" {
		t.Errorf("manifest %s", got)
	}
	if got := LostDigest([]uint64{11, 12}); got != "5b48f63d5210359d414fe1febfa1304c85400fb2bc09f9aa3c3f0e7ae247fbba" {
		t.Errorf("lost %s", got)
	}
	if LostDigest(nil) != lostDigest([]uint64{}) || ManifestDigest(manifest()) != manifestDigest(manifest()) {
		t.Error("wrappers differ")
	}
}

func TestCheckMatchesPropose(t *testing.T) {
	f := standard(t)
	gen := func(id string) uint64 { return f.r.installations[id].generation }
	f.agree("malformed", Change{Kind: "other"}, ErrInvalid)
	f.agree("malformed unlock", unlockOf("i1", 1, "x"), ErrInvalid)
	f.agree("unknown writer scope", Change{Kind: ChangeWriter, Writer: Writer{Installation: "i9", ID: "w-new", Kind: WriterERP, MaxCall: time.Minute}}, ErrNotFound)
	f.agree("unknown unlock scope", unlockOf("i9", 1, LostDigest(nil)), ErrNotFound)
	f.agree("taken installation", Change{Kind: ChangeInstallation, Installation: Installation{ID: "i1", Tenant: "t"}}, ErrConflict)
	f.agree("new installation", Change{Kind: ChangeInstallation, Installation: Installation{ID: "i3", Tenant: "t"}}, nil)
	f.agree("writer identity", Change{Kind: ChangeWriter, Writer: Writer{Installation: "i1", ID: "w-erp", Kind: WriterERP, MaxCall: time.Minute}}, refused(ReasonIdentityTaken))
	f.agree("verifier identity", Change{Kind: ChangeVerifier, Verifier: Verifier{Installation: "i1", ID: "b-v-1"}}, refused(ReasonIdentityTaken))
	f.agree("external taken", Change{Kind: ChangeExternal, External: External{Installation: "i1", ID: "erp-x", Kind: WriterERP}}, ErrConflict)
	f.agree("new writer", Change{Kind: ChangeWriter, Writer: Writer{Installation: "i1", ID: "w-new", Kind: WriterERP, MaxCall: time.Minute}}, nil)
	s2 := f.r.installations["i2"]
	for index := 0; len(s2.writers) < maxWriters; index++ {
		s2.writers["x"+itoa(index)] = &writer{view: Writer{Kind: WriterERP}}
	}
	for index := 0; len(s2.externals) < maxExternals; index++ {
		s2.externals["x"+itoa(index)] = External{ID: "x" + itoa(index), Kind: WriterERP}
	}
	f.agree("writer limit", Change{Kind: ChangeWriter, Writer: Writer{Installation: "i2", ID: "w-over", Kind: WriterERP, MaxCall: time.Minute}}, ErrLimit)
	f.agree("external limit", Change{Kind: ChangeExternal, External: External{Installation: "i2", ID: "x-over", Kind: WriterERP}}, ErrLimit)
	point := f.point("i1", start.Add(-time.Hour))
	future := point
	future.AsOf = start.Add(time.Hour)
	f.agree("begin future", Change{Kind: ChangeBegin, Begin: future}, ErrInvalid)
	stale := point
	stale.Generation = 9
	f.agree("begin generation", Change{Kind: ChangeBegin, Begin: stale}, refused(ReasonGeneration))
	forged := point
	forged.ManifestDigest = sum("forged")
	f.agree("begin manifest", Change{Kind: ChangeBegin, Begin: forged}, refused(ReasonManifestMismatch))
	f.agree("prune future", Change{Kind: ChangePrune, Prune: Prune{Installation: "i1", Before: start.Add(time.Hour)}}, ErrInvalid)
	f.agree("prune", Change{Kind: ChangePrune, Prune: Prune{Installation: "i1", Before: start}}, nil)
	f.agree("abandon nothing", Change{Kind: ChangeAbandon, Abandon: Abandon{Installation: "i1", Recovery: "recovery-1", Generation: 1}}, refused(ReasonNoRestorePoint))
	f.agree("unlock serving", unlockOf("i1", 1, LostDigest(nil)), refused(ReasonNotRecovering))
	tok := f.permit("i1", erp)
	f.must(second(f.r.Call("i1", "w-erp", tok, numbered("inv-1", 1), erp)))
	f.must(f.r.Lock("i1", "sample lock", ann))
	f.agree("unlock open calls", unlockOf("i1", gen("i1"), LostDigest(nil)), refused(ReasonOpenCalls))
	f.agree("unlock generation", unlockOf("i1", 1, LostDigest(nil)), refused(ReasonGeneration))
	f.advance(2 * time.Minute)
	f.agree("unlock after reclaim", unlockOf("i1", gen("i1"), LostDigest(nil)), nil)
	f.begin("i1", start.Add(-time.Hour))
	f.agree("begin open recovery", Change{Kind: ChangeBegin, Begin: f.point("i1", start.Add(-time.Hour))}, refused(ReasonOpenRecovery))
	f.agree("prune open recovery", Change{Kind: ChangePrune, Prune: Prune{Installation: "i1", Before: start}}, refused(ReasonOpenRecovery))
	f.agree("unlock incomplete", unlockOf("i1", gen("i1"), LostDigest(nil)), refused(ReasonIncomplete))
	f.agree("abandon generation", Change{Kind: ChangeAbandon, Abandon: Abandon{Installation: "i1", Recovery: "recovery-1", Generation: 1}}, refused(ReasonGeneration))
	f.agree("abandon", Change{Kind: ChangeAbandon, Abandon: Abandon{Installation: "i1", Recovery: "recovery-1", Generation: gen("i1")}}, nil)
	f.approve(Change{Kind: ChangeAbandon, Abandon: Abandon{Installation: "i1", Recovery: "recovery-1", Generation: gen("i1")}})
	f.agree("unlock no restore point", unlockOf("i1", gen("i1"), LostDigest(nil)), refused(ReasonNoRestorePoint))
	f.agree("abandon no restore point", Change{Kind: ChangeAbandon, Abandon: Abandon{Installation: "i1", Recovery: "recovery-1", Generation: gen("i1")}}, refused(ReasonNoRestorePoint))
	g := setup(t)
	g.installation("i1", "")
	g.approve(Change{Kind: ChangePrune, Prune: Prune{Installation: "i1", Before: start}})
	g.agree("begin pruned", Change{Kind: ChangeBegin, Begin: g.point("i1", start.Add(-time.Hour))}, refused(ReasonPruned))
}

func TestCheckMatchesProposeAtTheUnlock(t *testing.T) {
	f := standard(t)
	f.approve(Change{Kind: ChangeWriter, Writer: Writer{Installation: "i2", ID: "w-del", Kind: WriterDelivery, MaxCall: time.Minute}})
	for _, id := range []string{"i1", "i2"} {
		f.begin(id, start.Add(-time.Hour))
	}
	f.verifyAll("i1", ver)
	f.verifyAll("i2", Actor{ID: "b-v-1", Kind: KindSystem})
	f.advance(10 * time.Minute)
	f.reconcileAll("i1", ver, start.Add(-2*time.Hour), f.at())
	f.reconcileAll("i2", Actor{ID: "b-v-1", Kind: KindSystem}, start.Add(-2*time.Hour), f.at())
	f.agree("unlock lost", unlockOf("i1", 2, sum("other")), refused(ReasonLost))
	f.agree("unlock kind coverage", unlockOf("i2", 2, LostDigest(nil)), refused(ReasonKindCoverage))
	f.agree("unlock ready", unlockOf("i1", 2, LostDigest(nil)), nil)
	if got := f.status("i1"); got != "recovering 2 "+start.Format(time.RFC3339) {
		t.Errorf("recovering %s", got)
	}
	f.must(f.unlock("i1"))
	if got := f.status("i1"); got != "serving 2 0001-01-01T00:00:00Z" {
		t.Errorf("resumed %s", got)
	}
}

func TestCheckMatchesApprove(t *testing.T) {
	f := standard(t)
	f.must(f.r.Lock("i1", "sample lock", ann))
	p, err := f.r.Propose(unlockOf("i1", 2, LostDigest(nil)), ann)
	f.must(err)
	q, err := f.r.Propose(unlockOf("i1", 2, LostDigest(nil)), cat)
	f.must(err)
	f.begin("i1", start.Add(-time.Hour))
	expect(t, "stale", f.agreeApprove("stale unlock", p.Number, bob), ErrRejected)
	f.must(f.r.Lock("i2", "sample lock", ann))
	r, err := f.r.Propose(unlockOf("i2", 2, LostDigest(nil)), ann)
	f.must(err)
	expect(t, "valid", f.agreeApprove("valid unlock", r.Number, bob), nil)
	if v, _ := f.r.Proposal(q.Number); v.State != ProposalPending {
		t.Errorf("other pending %+v", v)
	}
}

func TestStatus(t *testing.T) {
	f := setup(t)
	if got := f.status(""); got != ErrInvalid.Error() {
		t.Errorf("malformed %s", got)
	}
	if got := f.status("i9"); got != ErrNotFound.Error() {
		t.Errorf("unknown %s", got)
	}
	f.installation("i1", "")
	zero := "0001-01-01T00:00:00Z"
	if got := f.status("i1"); got != "serving 1 "+zero {
		t.Errorf("new %s", got)
	}
	f.advance(time.Minute)
	f.must(f.r.Lock("i1", "sample lock", ann))
	if got := f.status("i1"); got != "locked 2 "+f.at().Format(time.RFC3339) {
		t.Errorf("locked %s", got)
	}
	f.must(f.unlock("i1"))
	if got := f.status("i1"); got != "serving 2 "+zero {
		t.Errorf("released %s", got)
	}
	f.advance(time.Minute)
	f.begin("i1", start)
	if got := f.status("i1"); got != "recovering 3 "+f.at().Format(time.RFC3339) {
		t.Errorf("recovering %s", got)
	}
}
