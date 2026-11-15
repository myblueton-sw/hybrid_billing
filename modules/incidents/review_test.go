package incidents

import (
	"testing"
	"time"
)

func TestMergedCopySnoozeKeepsTheOwnSubscription(t *testing.T) {
	f := open(t, testLimits())
	f.subscribe(t, "ana", "c", SeverityInfo)
	f.subscribe(t, "bob", "c", SeverityInfo)
	must(f.r.Configure("ws", Settings{Recipient: "ana", Zone: "UTC", Digest: DigestImmediate, Alternate: "bob", Away: true}, human("ana")))
	inc := f.raise(t, signal("s1", "c", "k1", SeverityWarning, false, start))
	p := must(f.r.Plan("ws", system("p")))
	must(f.r.Snooze("ws", inc.ID, start.Add(time.Hour), human("ana")))
	if d := must(f.r.Dispatch("ws", p[0].ID, system("s"))); len(d.Delivered) != 1 {
		t.Fatal("ana's snooze must not hold bob's own copy")
	}
}

func TestDispatchChecksTheRouteIsStillCurrent(t *testing.T) {
	f := open(t, testLimits())
	f.subscribe(t, "ana", "c", SeverityInfo)
	must(f.r.Configure("ws", Settings{Recipient: "ana", Zone: "UTC", Digest: DigestImmediate, Alternate: "bob", Away: true}, human("ana")))
	f.raise(t, signal("s1", "c", "k1", SeverityWarning, false, start))
	p := must(f.r.Plan("ws", system("p")))
	must(f.r.Configure("ws", Settings{Recipient: "ana", Zone: "UTC", Digest: DigestImmediate}, human("ana")))
	if d := must(f.r.Dispatch("ws", p[0].ID, system("s"))); len(d.Delivered) != 0 {
		t.Fatal("the alternate no longer stands in")
	}
	if w := must(f.r.Withholdings("ws", system("w"))); len(w) != 1 || w[0].Reason != WithheldNoTarget || w[0].Target != "bob" {
		t.Fatal(w)
	}
	if got := f.cycle(t); len(got) != 1 || got[0].Target != "ana" {
		t.Fatal("back to the recipient")
	}
}

func TestDigestsNeverPileUp(t *testing.T) {
	f := open(t, testLimits())
	f.subscribe(t, "ana", "c", SeverityInfo)
	must(f.r.Configure("ws", Settings{Recipient: "ana", Zone: "UTC", Digest: DigestHourly}, human("ana")))
	f.raise(t, signal("s1", "c", "k1", SeverityInfo, false, start))
	d1 := must(f.r.Plan("ws", system("p")))
	f.clock.advance(10 * time.Minute)
	f.raise(t, signal("s2", "c", "k2", SeverityInfo, false, f.clock.read()))
	if again := must(f.r.Plan("ws", system("p"))); len(again) != 0 {
		t.Fatal("a pending digest anchors the cadence")
	}
	must(f.r.Dispatch("ws", d1[0].ID, system("s")))
	if again := must(f.r.Plan("ws", system("p"))); len(again) != 0 {
		t.Fatal("cadence after handoff")
	}
	f.clock.advance(time.Hour)
	got := f.cycle(t)
	if len(got) != 1 || len(got[0].Delivered) != 1 {
		t.Fatalf("next digest: %+v", got)
	}
	g := open(t, testLimits())
	g.subscribe(t, "ana", "c", SeverityInfo)
	must(g.r.Configure("ws", Settings{Recipient: "ana", Zone: "UTC", Digest: DigestDaily, Quiet: &Quiet{Start: 13 * 60, End: 14 * 60}}, human("ana")))
	g.raise(t, signal("s1", "c", "k1", SeverityInfo, false, start))
	q := must(g.r.Plan("ws", system("p")))
	g.clock.advance(time.Hour)
	if d := must(g.r.Dispatch("ws", q[0].ID, system("s"))); len(d.Delivered) != 0 {
		t.Fatal("quiet at dispatch")
	}
	g.clock.advance(time.Hour)
	if got := g.cycle(t); len(got) != 1 || len(got[0].Delivered) != 1 {
		t.Fatal("an empty digest does not reset the cadence")
	}
	h := open(t, testLimits())
	h.subscribe(t, "ana", "c", SeverityInfo)
	must(h.r.Configure("ws", Settings{Recipient: "ana", Zone: "UTC", Digest: DigestHourly}, human("ana")))
	h.raise(t, signal("s1", "c", "k1", SeverityInfo, false, start))
	must(h.r.Plan("ws", system("p")))
	h.clock.advance(time.Hour)
	h.raise(t, signal("s2", "c", "k2", SeverityInfo, false, h.clock.read()))
	if again := must(h.r.Plan("ws", system("p"))); len(again) != 1 || len(again[0].Incidents) != 2 {
		t.Fatal("a stale pending digest is superseded")
	}
}

func TestCriticalAndDigestInOneRunForSeveralTargets(t *testing.T) {
	f := open(t, testLimits())
	for _, who := range []string{"cy", "ana", "bob"} {
		f.subscribe(t, who, "c", SeverityInfo)
		must(f.r.Configure("ws", Settings{Recipient: who, Zone: "UTC", Digest: DigestHourly}, human(who)))
	}
	f.raise(t, signal("s1", "c", "k1", SeverityInfo, false, start))
	f.raise(t, signal("s2", "c", "k2", SeverityCritical, false, start))
	p := must(f.r.Plan("ws", system("p")))
	if len(p) != 6 {
		t.Fatal(p)
	}
	for index, want := range []string{"ana", "ana", "bob", "bob", "cy", "cy"} {
		if p[index].Target != want || p[index].Digest != (index%2 == 1) {
			t.Fatalf("order: %+v", p)
		}
	}
}

func TestWithholdingTransitions(t *testing.T) {
	f := open(t, testLimits())
	f.subscribe(t, "ana", "c", SeverityInfo)
	f.raise(t, signal("s1", "c", "k1", SeverityWarning, false, start))
	f.perms.deny("ana", "k1")
	must(f.r.Plan("ws", system("p")))
	f.perms.failing["ana"] = true
	must(f.r.Plan("ws", system("p")))
	delete(f.perms.failing, "ana")
	must(f.r.Plan("ws", system("p")))
	must(f.r.Plan("ws", system("p")))
	w := must(f.r.Withholdings("ws", system("w")))
	if len(w) != 3 || w[0].Reason != WithheldRefused || w[1].Reason != WithheldUnavailable || w[2].Reason != WithheldRefused {
		t.Fatal(w)
	}
	g := open(t, testLimits())
	g.subscribe(t, "ana", "c", SeverityInfo)
	must(g.r.Configure("ws", Settings{Recipient: "ana", Zone: "UTC", Digest: DigestImmediate, Alternate: "bob", Away: true}, human("ana")))
	must(g.r.Configure("ws", Settings{Recipient: "bob", Zone: "UTC", Digest: DigestImmediate, Alternate: "cy", Away: true}, human("bob")))
	inc := g.raise(t, signal("s1", "c", "k1", SeverityWarning, false, start))
	must(g.r.Plan("ws", system("p")))
	must(g.r.Configure("ws", Settings{Recipient: "bob", Zone: "UTC", Digest: DigestImmediate}, human("bob")))
	if got := g.cycle(t); len(got) != 1 || got[0].Target != "bob" {
		t.Fatal(got)
	}
	must(g.r.Configure("ws", Settings{Recipient: "bob", Zone: "UTC", Digest: DigestImmediate, Alternate: "cy", Away: true}, human("bob")))
	must(g.r.Plan("ws", system("p")))
	w = must(g.r.Withholdings("ws", system("w")))
	if len(w) != 2 || w[1].Reason != WithheldNoTarget {
		t.Fatalf("no_target cycle: %+v", w)
	}
	must(g.r.Snooze("ws", inc.ID, start.Add(time.Hour), human("ana")))
	must(g.r.Plan("ws", system("p")))
	if len(must(g.r.Withholdings("ws", system("w")))) != 2 {
		t.Fatal("a snoozed copy records no no_target")
	}
}

func TestRisesWhilePending(t *testing.T) {
	f := open(t, testLimits())
	f.subscribe(t, "ana", "c", SeverityInfo)
	f.raise(t, signal("s1", "c", "k1", SeverityInfo, false, start))
	must(f.r.Plan("ws", system("p")))
	f.raise(t, signal("s2", "c", "k1", SeverityWarning, false, start))
	if again := must(f.r.Plan("ws", system("p"))); len(again) != 0 {
		t.Fatal("a rise to warning waits for the pending notification")
	}
}

func TestLateSignalsFindTheirResolvedIncident(t *testing.T) {
	f := open(t, testLimits())
	a := f.raise(t, signal("s1", "c", "k1", SeverityInfo, false, start.Add(-time.Hour)))
	f.clock.advance(time.Minute)
	must(f.r.Resolve("ws", a.ID, Resolution{Reason: ReasonFixed, Evidence: "e"}, system("auto")))
	f.clock.advance(time.Minute)
	b := f.raise(t, signal("s2", "c", "k1", SeverityInfo, false, f.clock.read()))
	f.clock.advance(time.Minute)
	must(f.r.Resolve("ws", b.ID, Resolution{Reason: ReasonFixed, Evidence: "e"}, system("auto")))
	late := f.raise(t, signal("s3", "c", "k1", SeverityInfo, false, start.Add(90*time.Second)))
	if late.ID != b.ID || late.Late != 1 {
		t.Fatalf("the signal belongs to the second incident: %+v", late)
	}
	if first := must(f.r.IncidentOf("ws", a.ID, system("w"))); first.Late != 0 {
		t.Fatal(first)
	}
}

func TestReadAndLimitGates(t *testing.T) {
	f := open(t, testLimits())
	inc := f.raise(t, signal("s1", "c", "k1", SeverityInfo, false, start))
	expect(t, "incidentof actor", second(f.r.IncidentOf("ws", inc.ID, Actor{ID: "x"})), "invalid")
	expect(t, "incidentof human", second(f.r.IncidentOf("ws", inc.ID, human("x"))), RefusalNotSystem)
	expect(t, "incidentof assistant", second(f.r.IncidentOf("ws", inc.ID, assistant("x"))), RefusalAINotAllowed)
	expect(t, "withholdings actor", second(f.r.Withholdings("ws", Actor{ID: "x"})), "invalid")
	expect(t, "withholdings human", second(f.r.Withholdings("ws", human("x"))), RefusalNotSystem)
	l := testLimits()
	l.Subscriptions = 1
	g := open(t, l)
	g.subscribe(t, "ana", "c", SeverityInfo)
	g.subscribe(t, "ana", "c", SeverityWarning)
	expect(t, "subscriptions", second(g.r.Subscribe("ws", Subscription{Recipient: "ana", Condition: "d", Minimum: SeverityInfo}, human("ana"))), "limit")
	l = testLimits()
	l.Notifications = 1
	h := open(t, l)
	h.subscribe(t, "ana", "c", SeverityInfo)
	h.raise(t, signal("s1", "c", "k1", SeverityWarning, false, start))
	p := must(h.r.Plan("ws", system("p")))
	h.perms.deny("ana", "k1")
	expect(t, "dispatch limit", second(h.r.Dispatch("ws", p[0].ID, system("s"))), "limit")
	h.perms.allow("ana", "k1")
	if d := must(h.r.Dispatch("ws", p[0].ID, system("s"))); !d.HandedOff || len(d.Delivered) != 1 {
		t.Fatal("a refused dispatch leaves the notification pending")
	}
}

func TestDispatchRefusesAnAlternateWhoWentAway(t *testing.T) {
	f := open(t, testLimits())
	f.subscribe(t, "ana", "c", SeverityInfo)
	must(f.r.Configure("ws", Settings{Recipient: "ana", Zone: "UTC", Digest: DigestImmediate, Alternate: "bob", Away: true}, human("ana")))
	f.raise(t, signal("s1", "c", "k1", SeverityWarning, false, start))
	p := must(f.r.Plan("ws", system("p")))
	must(f.r.Configure("ws", Settings{Recipient: "bob", Zone: "UTC", Digest: DigestImmediate, Alternate: "cy", Away: true}, human("bob")))
	if d := must(f.r.Dispatch("ws", p[0].ID, system("s"))); len(d.Delivered) != 0 {
		t.Fatal("an alternate who went away receives nothing")
	}
}

func TestPendingStateBelongsToItsNotification(t *testing.T) {
	f := open(t, testLimits())
	f.subscribe(t, "ana", "c", SeverityInfo)
	f.raise(t, signal("s1", "c", "k1", SeverityWarning, false, start))
	q1 := must(f.r.Plan("ws", system("p")))
	if q1[0].Digest {
		t.Fatal("recipients without settings get immediate notifications")
	}
	f.raise(t, signal("s2", "c", "k1", SeverityCritical, false, start))
	must(f.r.Plan("ws", system("p")))
	f.perms.deny("ana", "k1")
	must(f.r.Dispatch("ws", q1[0].ID, system("s")))
	f.perms.allow("ana", "k1")
	if again := must(f.r.Plan("ws", system("p"))); len(again) != 0 {
		t.Fatal("the critical notification is still pending")
	}
	g := open(t, testLimits())
	g.subscribe(t, "ana", "c", SeverityInfo)
	must(g.r.Configure("ws", Settings{Recipient: "ana", Zone: "UTC", Digest: DigestHourly}, human("ana")))
	g.raise(t, signal("s1", "c", "k1", SeverityInfo, false, start))
	g.raise(t, signal("s2", "c", "k2", SeverityCritical, false, start))
	p := must(g.r.Plan("ws", system("p")))
	must(g.r.Dispatch("ws", p[0].ID, system("s")))
	g.raise(t, signal("s3", "c", "k3", SeverityInfo, false, start))
	if again := must(g.r.Plan("ws", system("p"))); len(again) != 0 {
		t.Fatal("dispatching the immediate notification keeps the digest anchor")
	}
	h := open(t, testLimits())
	h.subscribe(t, "ana", "c", SeverityInfo)
	must(h.r.Configure("ws", Settings{Recipient: "ana", Zone: "UTC", Digest: DigestImmediate, Alternate: "bob", Away: true}, human("ana")))
	h.raise(t, signal("s1", "c", "k1", SeverityWarning, false, start))
	r := must(h.r.Plan("ws", system("p")))
	must(h.r.Configure("ws", Settings{Recipient: "ana", Zone: "UTC", Digest: DigestImmediate, Alternate: "bob"}, human("ana")))
	if d := must(h.r.Dispatch("ws", r[0].ID, system("s"))); len(d.Delivered) != 0 {
		t.Fatal("the recipient is back, so the alternate no longer stands in")
	}
}
