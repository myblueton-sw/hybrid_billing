package recovery

import (
	"testing"
	"time"
)

func TestRemainingPaths(t *testing.T) {
	f := standard(t)
	for index := 0; index < maxInstallations; index++ {
		f.r.installations["x"+itoa(index)] = &site{}
	}
	expect(t, "installations", f.try(Change{Kind: ChangeInstallation, Installation: Installation{ID: "new", Tenant: "t"}}), ErrLimit)
	g := setup(t)
	g.approve(Change{Kind: ChangeInstallation, Installation: Installation{ID: "bare", Tenant: "t"}})
	g.must(g.r.Lock("bare", "stop", ann))
	g.must(second(g.r.Propose(Change{Kind: ChangeExternal, External: External{Installation: "bare", ID: "a", Kind: WriterERP}}, ann)))
	g.must(second(g.r.Propose(Change{Kind: ChangeExternal, External: External{Installation: "bare", ID: "b", Kind: WriterERP}}, ann)))
	h := setup(t)
	var first uint64
	for index := 0; index <= maxProposals; index++ {
		p := h.approve(Change{Kind: ChangeInstallation, Installation: Installation{ID: "s" + itoa(index), Tenant: "t"}})
		if index == 0 {
			first = p.Number
		}
	}
	expect(t, "history pruned", second(h.r.Proposal(first)), ErrNotFound)
	k := standard(t)
	k.must(second(k.r.Call("i1", "w-erp", k.permit("i1", erp), target("busy"), erp)))
	ks := k.r.installations["i1"]
	for index := 0; index < maxKeys; index++ {
		ks.fenced[[2]string{"z", "k" + itoa(index)}] = true
	}
	k.begin("i1", start)
	if ks.fenced[[2]string{"ws", "busy"}] {
		t.Error("auto-fence ignored the cap")
	}
	reason(t, "settle verifier", second(k.r.Reapplied("i1", 1, erp)), ReasonNotVerifier)
	reason(t, "unapplicable verifier", second(k.r.Unapplicable("i1", 1, "x", erp)), ReasonNotVerifier)
	m := standard(t)
	ev1 := m.must2(m.r.Record("i1", Event{Kind: EventDelete, Domain: DomainEvidence, Key: "k"}, erp))
	m.must2(m.r.Record("i1", Event{Kind: EventDelete, Domain: DomainEvidence, Key: "k2"}, erp))
	m.advance(time.Minute)
	m.begin("i1", start.Add(-time.Minute))
	m.verifyAll("i1", ver)
	if v, _ := m.r.Recovery("i1"); v.Phase != PhaseVerify {
		t.Errorf("phase before reapply %s", v.Phase)
	}
	m.must(second(m.r.Reapplied("i1", ev1.Sequence, ver)))
	if v, _ := m.r.Recovery("i1"); v.Phase != PhaseReapply {
		t.Errorf("phase %s", v.Phase)
	}
	m.advance(10 * time.Minute)
	m.must(second(m.r.Reconcile("i1", "erp-x", Page{Number: 1, From: start.Add(-time.Hour), Through: m.at()}, ver)))
	m.must(second(m.r.Reconcile("i1", "erp-x", Page{Number: 2, From: start, Through: start.Add(time.Minute), Final: true, Rotated: true}, ver)))
	if pg := m.r.installations["i1"].current.externals["erp-x"]; !pg.through.Equal(m.at()) {
		t.Error("through went backwards")
	}
}
