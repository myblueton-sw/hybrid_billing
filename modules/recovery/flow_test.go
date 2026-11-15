package recovery

import (
	"testing"
	"time"
)

func TestPermitsAndCalls(t *testing.T) {
	f := standard(t)
	expect(t, "form", second(f.r.Permit("i1", "", erp)), ErrInvalid)
	expect(t, "actor", second(f.r.Permit("i1", "w-erp", Actor{ID: "w-erp"})), ErrInvalid)
	expect(t, "site", second(f.r.Permit("i9", "w-erp", erp)), ErrNotFound)
	expect(t, "site form", second(f.r.Permit("", "w-erp", erp)), ErrInvalid)
	reason(t, "other writer", second(f.r.Permit("i1", "w-iss", erp)), ReasonNotWriter)
	reason(t, "human", second(f.r.Permit("i1", "w-erp", Actor{ID: "w-erp", Kind: KindHuman})), ReasonNotWriter)
	reason(t, "foreign site", second(f.r.Permit("i2", "w-erp", erp)), ReasonNotWriter)
	tok := f.permit("i1", erp)
	if tok.Generation != 1 || tok.Sequence != 1 || !tok.Expiry.Equal(start.Add(time.Minute)) {
		t.Fatalf("token %+v", tok)
	}
	expect(t, "target form", second(f.r.Call("i1", "w-erp", tok, Target{Workspace: "ws"}, erp)), ErrInvalid)
	expect(t, "series without number", second(f.r.Call("i1", "w-erp", tok, Target{Workspace: "ws", Key: "k", Series: "inv"}, erp)), ErrInvalid)
	expect(t, "number without series", second(f.r.Call("i1", "w-erp", tok, Target{Workspace: "ws", Key: "k", Number: 3}, erp)), ErrInvalid)
	expect(t, "call actor", second(f.r.Call("i1", "w-erp", tok, target("k"), Actor{ID: "x"})), ErrInvalid)
	expect(t, "call site", second(f.r.Call("i9", "w-erp", tok, target("k"), erp)), ErrNotFound)
	reason(t, "call other actor", second(f.r.Call("i1", "w-erp", tok, target("k"), iss)), ReasonNotWriter)
	itok := f.permit("i1", iss)
	reason(t, "token of another writer", second(f.r.Call("i1", "w-erp", itok, target("k"), erp)), ReasonNotWriter)
	reason(t, "forged generation", second(f.r.Call("i1", "w-erp", Token{Generation: 2, Sequence: tok.Sequence}, target("k"), erp)), ReasonFenced)
	reason(t, "unknown sequence", second(f.r.Call("i1", "w-erp", Token{Generation: 1, Sequence: 99}, target("k"), erp)), ReasonFenced)
	c, err := f.r.Call("i1", "w-erp", tok, numbered("inv-1", 1), erp)
	f.must(err)
	if c.ID != "call-1" || !c.Deadline.Equal(start.Add(time.Minute)) || c.Outcome != "" {
		t.Fatalf("call %+v", c)
	}
	expect(t, "done form", second(f.r.Done("i1", "", OutcomeConfirmed, erp)), ErrInvalid)
	expect(t, "done outcome", second(f.r.Done("i1", c.ID, "maybe", erp)), ErrInvalid)
	expect(t, "done actor", second(f.r.Done("i1", c.ID, OutcomeConfirmed, Actor{ID: "x"})), ErrInvalid)
	expect(t, "done site", second(f.r.Done("i9", c.ID, OutcomeConfirmed, erp)), ErrNotFound)
	expect(t, "done missing", second(f.r.Done("i1", "call-9", OutcomeConfirmed, erp)), ErrNotFound)
	reason(t, "done other", second(f.r.Done("i1", c.ID, OutcomeConfirmed, iss)), ReasonNotWriter)
	d, err := f.r.Done("i1", c.ID, OutcomeConfirmed, erp)
	if err != nil || d.Outcome != OutcomeConfirmed {
		t.Fatal(d, err)
	}
	f.must(second(f.r.Done("i1", c.ID, OutcomeConfirmed, erp)))
	expect(t, "done conflict", second(f.r.Done("i1", c.ID, OutcomeFailed, erp)), ErrConflict)
	f.advance(time.Minute)
	reason(t, "expired", second(f.r.Call("i1", "w-erp", tok, target("k"), erp)), ReasonExpired)
	tok = f.permit("i1", erp)
	late, _ := f.r.Call("i1", "w-erp", tok, target("late"), erp)
	f.advance(time.Minute)
	reclaimed, err := f.r.Done("i1", late.ID, OutcomeUnknown, erp)
	if err != nil || reclaimed.Outcome != OutcomeUnknown {
		t.Errorf("reclaimed %+v %v", reclaimed, err)
	}
	f.must(f.r.RemoveWriter("i1", "w-iss", ann))
	reason(t, "removed", second(f.r.Call("i1", "w-iss", itok, target("k"), iss)), ReasonNotWriter)
	reason(t, "removed permit", second(f.r.Permit("i1", "w-iss", iss)), ReasonNotWriter)
	if w, err := f.r.HighWater("i1", "ws", "inv"); err != nil || w != 0 {
		t.Errorf("high water %d %v", w, err)
	}
	expect(t, "high form", second(f.r.HighWater("i1", "", "inv")), ErrInvalid)
	expect(t, "high site", second(f.r.HighWater("i9", "ws", "inv")), ErrNotFound)
	expect(t, "fenced form", second(f.r.Fenced("i1", "ws", "")), ErrInvalid)
	expect(t, "fenced site", second(f.r.Fenced("i9", "ws", "k")), ErrNotFound)
}

func TestPermitEvictionAndCallCaps(t *testing.T) {
	f := standard(t)
	first := f.permit("i1", erp)
	other := f.permit("i1", iss)
	for index := 0; index < maxLive; index++ {
		f.permit("i1", erp)
	}
	reason(t, "evicted", second(f.r.Call("i1", "w-erp", first, target("k"), erp)), ReasonFenced)
	f.must(second(f.r.Call("i1", "w-iss", other, target("k"), iss)))
	tok := f.permit("i1", erp)
	s := f.r.installations["i1"]
	s.writers["w-erp"].open = maxOpenPerWriter
	expect(t, "per writer", second(f.r.Call("i1", "w-erp", tok, target("k2"), erp)), ErrLimit)
	s.writers["w-erp"].open = 0
	for index := 0; index < maxOpen; index++ {
		s.open["x"+itoa(index)] = &call{view: CallView{ID: "x" + itoa(index), Writer: "w-iss", Deadline: start.Add(time.Hour)}}
	}
	expect(t, "total", second(f.r.Call("i1", "w-erp", tok, target("k2"), erp)), ErrLimit)
	g := standard(t)
	gt := g.permit("i1", erp)
	for index := 0; index <= maxClosed; index++ {
		c, _ := g.r.Call("i1", "w-erp", gt, target("c"+itoa(index)), erp)
		g.must(second(g.r.Done("i1", c.ID, OutcomeFailed, erp)))
	}
	gs := g.r.installations["i1"]
	if len(gs.closed) != maxClosed || gs.closedByID["call-1"] != nil || gs.closedByID["call-2"] == nil {
		t.Error("closed ring")
	}
}

func TestLockAndPlainUnlock(t *testing.T) {
	f := standard(t)
	tok := f.permit("i1", erp)
	c, _ := f.r.Call("i1", "w-erp", tok, target("k"), erp)
	reason(t, "assistant", f.r.Lock("i1", "suspected", bot), ReasonAINotAllowed)
	expect(t, "reason", f.r.Lock("i1", "", ann), ErrInvalid)
	expect(t, "site", f.r.Lock("i9", "x", ann), ErrNotFound)
	reason(t, "unlock serving", f.unlock("i1"), ReasonNotRecovering)
	f.must(f.r.Lock("i1", "Suspected compromise", ann))
	f.must(f.r.Lock("i1", "again", ann))
	s := f.r.installations["i1"]
	if s.state != StateLocked || s.generation != 2 || !s.stoppedAt.Equal(start) || !s.inflight.Equal(c.Deadline) {
		t.Fatalf("locked %+v", s)
	}
	reason(t, "permit locked", second(f.r.Permit("i1", "w-erp", erp)), ReasonLocked)
	reason(t, "old token", second(f.r.Call("i1", "w-erp", tok, target("k2"), erp)), ReasonFenced)
	s.tokens[tok.Sequence].generation = 2
	reason(t, "locked state", second(f.r.Call("i1", "w-erp", Token{Generation: 2, Sequence: tok.Sequence}, target("k2"), erp)), ReasonLocked)
	reason(t, "open calls", f.unlock("i1"), ReasonOpenCalls)
	f.must(second(f.r.Done("i1", c.ID, OutcomeConfirmed, erp)))
	f.must(f.unlock("i1"))
	if s.state != StateServing || !s.stoppedAt.IsZero() {
		t.Error("unlocked")
	}
	reason(t, "pre-lock token", second(f.r.Call("i1", "w-erp", Token{Generation: 1, Sequence: 99}, target("k3"), erp)), ReasonFenced)
	if other := f.r.installations["i2"]; other.state != StateServing || other.generation != 1 {
		t.Error("lock changed another installation")
	}
}

func TestBeginRules(t *testing.T) {
	f := standard(t)
	p := f.point("i1", start)
	bad := p
	bad.Manifest = manifest()[:8]
	expect(t, "domains", f.try(Change{Kind: ChangeBegin, Begin: bad}), ErrInvalid)
	order := f.point("i1", start)
	order.Manifest[0], order.Manifest[1] = order.Manifest[1], order.Manifest[0]
	expect(t, "order", f.try(Change{Kind: ChangeBegin, Begin: order}), ErrInvalid)
	for name, change := range map[string]func(*RestorePoint){
		"backup":     func(p *RestorePoint) { p.Backup = "x" },
		"digest":     func(p *RestorePoint) { p.ManifestDigest = "x" },
		"asof":       func(p *RestorePoint) { p.AsOf = time.Time{} },
		"generation": func(p *RestorePoint) { p.Generation = 0 },
		"chunks":     func(p *RestorePoint) { p.Manifest[0].Items = 11 },
		"site":       func(p *RestorePoint) { p.Installation = "" },
	} {
		c := f.point("i1", start)
		change(&c)
		expect(t, name, f.try(Change{Kind: ChangeBegin, Begin: c}), ErrInvalid)
	}
	future := f.point("i1", start.Add(time.Hour))
	expect(t, "future", f.try(Change{Kind: ChangeBegin, Begin: future}), ErrInvalid)
	wrong := f.point("i1", start)
	wrong.ManifestDigest = sum("other")
	reason(t, "manifest", f.try(Change{Kind: ChangeBegin, Begin: wrong}), ReasonManifestMismatch)
	stale := f.point("i1", start)
	stale.Generation = 7
	reason(t, "generation", f.try(Change{Kind: ChangeBegin, Begin: stale}), ReasonGeneration)
	pending, err := f.r.Propose(Change{Kind: ChangeBegin, Begin: p}, ann)
	f.must(err)
	other := p
	other.Backup = sum("other")
	expect(t, "one pending", second(f.r.Propose(Change{Kind: ChangeBegin, Begin: other}, cat)), ErrLimit)
	f.must(f.r.Lock("i1", "stop", ann))
	expect(t, "retired", second(f.r.Approve(pending.Number, bob)), ErrNotFound)
	f.begin("i1", start)
	s := f.r.installations["i1"]
	if s.state != StateRecovering || s.generation != 3 || !s.stoppedAt.Equal(start) || s.current.id != "recovery-1" {
		t.Fatalf("recovering %+v", s)
	}
	reason(t, "lock while recovering", f.r.Lock("i1", "x", ann), ReasonRecovering)
	reason(t, "permit while recovering", second(f.r.Permit("i1", "w-erp", erp)), ReasonRecovering)
	reason(t, "second begin", f.try(Change{Kind: ChangeBegin, Begin: f.point("i1", start)}), ReasonOpenRecovery)
	g := setup(t)
	g.installation("i1", "")
	for index := 0; index < maxBegins; index++ {
		g.r.begins = maxBegins
	}
	expect(t, "global begins", second(g.r.Propose(Change{Kind: ChangeBegin, Begin: g.point("i1", start)}, ann)), ErrLimit)
	g.r.begins = 0
	w, _ := g.r.Propose(Change{Kind: ChangeBegin, Begin: g.point("i1", start)}, ann)
	g.must(second(g.r.Withdraw(w.Number, ann)))
	if g.r.begins != 0 {
		t.Error("withdrawn begin counted")
	}
}
