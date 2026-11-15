package incidents

import (
	"testing"
	"time"
)

func TestRecurrenceAfterResolutionIsANewLinkedIncident(t *testing.T) {
	f := open(t, testLimits())
	a := f.raise(t, signal("s1", "late-usage", "k1", SeverityWarning, true, start.Add(-time.Hour)))
	b := f.raise(t, signal("s2", "late-usage", "k1", SeverityCritical, false, start.Add(-2*time.Hour)))
	if a.ID != b.ID || b.Count != 2 || !b.First.Equal(start.Add(-2*time.Hour)) || !b.Last.Equal(start.Add(-time.Hour)) || b.Severity != SeverityCritical || !b.Blocking {
		t.Fatalf("join: %+v", b)
	}
	if blocked := must(f.r.Blocked("ws", "k1", system("workflow"))); len(blocked) != 1 {
		t.Fatal("blocking")
	}
	f.clock.advance(time.Minute)
	resolved := must(f.r.Resolve("ws", a.ID, Resolution{Reason: ReasonFixed, Evidence: "run-9"}, human("ops")))
	if !resolved.Resolved || resolved.Resolver != "ops" || !resolved.ResolvedAt.Equal(start.Add(time.Minute)) {
		t.Fatal(resolved)
	}
	late := f.raise(t, signal("s3", "late-usage", "k1", SeverityCritical, true, start))
	if late.ID != a.ID || late.Late != 1 || late.Count != 2 || late.Severity != SeverityCritical {
		t.Fatalf("a late signal stays on the resolved incident: %+v", late)
	}
	if blocked := must(f.r.Blocked("ws", "k1", system("workflow"))); len(blocked) != 0 {
		t.Fatal("a late signal does not reopen blocking")
	}
	f.clock.advance(time.Minute)
	next := f.raise(t, signal("s4", "late-usage", "k1", SeverityWarning, false, f.clock.read()))
	if next.ID == a.ID || next.Recurs != a.ID || next.Count != 1 {
		t.Fatalf("recurrence: %+v", next)
	}
	before := f.raise(t, signal("s5", "late-usage", "k1", SeverityWarning, false, start.Add(-3*time.Hour)))
	if before.ID != a.ID || before.Late != 2 || !before.First.Equal(start.Add(-3*time.Hour)) {
		t.Fatalf("late rule applies first, even with an open incident: %+v", before)
	}
	if again := must(f.r.IncidentOf("ws", a.ID, system("w"))); !again.Resolved || again.Resolution.Evidence != "run-9" {
		t.Fatal("the earlier resolution is never overwritten")
	}
	f.clock.advance(time.Minute)
	must(f.r.Resolve("ws", next.ID, Resolution{Reason: ReasonOther, Evidence: "e"}, system("auto")))
	f.clock.advance(time.Minute)
	third := f.raise(t, signal("s6", "late-usage", "k1", SeverityInfo, false, f.clock.read()))
	if third.Recurs != next.ID {
		t.Fatal("recurrences chain")
	}
}

func TestQuietHoursSnoozeAndBlocking(t *testing.T) {
	f := open(t, testLimits())
	f.subscribe(t, "ana", "c", SeverityInfo)
	must(f.r.Configure("ws", Settings{Recipient: "ana", Zone: "Asia/Seoul", Quiet: &Quiet{Start: 22 * 60, End: 7 * 60}, Digest: DigestImmediate}, human("ana")))
	f.clock.set(time.Date(2026, 11, 16, 14, 30, 0, 0, time.UTC))
	w := f.raise(t, signal("s1", "c", "k1", SeverityWarning, true, f.clock.read()))
	if got := f.cycle(t); len(got) != 0 {
		t.Fatalf("quiet hours across midnight: %+v", got)
	}
	if len(must(f.r.Blocked("ws", "k1", system("workflow")))) != 1 {
		t.Fatal("blocking holds during quiet hours")
	}
	c := f.raise(t, signal("s2", "c", "k2", SeverityCritical, false, f.clock.read()))
	got := f.cycle(t)
	if len(got) != 1 || len(got[0].Delivered) != 1 || got[0].Delivered[0] != c.ID {
		t.Fatalf("critical bypasses quiet hours: %+v", got)
	}
	f.clock.set(time.Date(2026, 11, 16, 22, 0, 0, 0, time.UTC))
	must(f.r.Snooze("ws", w.ID, f.clock.read().Add(2*time.Hour), human("ana")))
	if got := f.cycle(t); len(got) != 0 {
		t.Fatalf("snoozed: %+v", got)
	}
	if len(must(f.r.Blocked("ws", "k1", system("workflow")))) != 1 {
		t.Fatal("blocking holds during snooze")
	}
	f.clock.advance(2 * time.Hour)
	if got := f.cycle(t); len(got) != 1 || got[0].Delivered[0] != w.ID {
		t.Fatalf("after snooze and quiet hours: %+v", got)
	}
}

func TestPermissionIsCheckedBeforeHandoffAndAtOpening(t *testing.T) {
	f := open(t, testLimits())
	f.subscribe(t, "ana", "c", SeverityInfo)
	inc := f.raise(t, signal("s1", "c", "k1", SeverityWarning, false, start))
	planned := must(f.r.Plan("ws", system("planner")))
	if len(planned) != 1 {
		t.Fatal(planned)
	}
	f.perms.deny("ana", "k1")
	n := must(f.r.Dispatch("ws", planned[0].ID, system("sender")))
	if !n.HandedOff || len(n.Delivered) != 0 {
		t.Fatalf("revoked between planning and handoff: %+v", n)
	}
	if w := must(f.r.Withholdings("ws", system("w"))); len(w) != 1 || w[0].Reason != WithheldRefused {
		t.Fatal(w)
	}
	if again := must(f.r.Dispatch("ws", planned[0].ID, system("sender"))); !again.Handoff.Equal(n.Handoff) {
		t.Fatal("dispatch happens once")
	}
	if got := must(f.r.Plan("ws", system("planner"))); len(got) != 0 {
		t.Fatal("withheld while refused")
	}
	if w := must(f.r.Withholdings("ws", system("w"))); len(w) != 1 {
		t.Fatal("withholding recorded only on change")
	}
	f.perms.allow("ana", "k1")
	got := f.cycle(t)
	if len(got) != 1 || got[0].Delivered[0] != inc.ID {
		t.Fatal(got)
	}
	expect(t, "other person", second(f.r.Open("ws", got[0].ID, inc.ID, human("bob"))), RefusalNotTarget)
	expect(t, "unknown notification", second(f.r.Open("ws", "note-99", inc.ID, human("ana"))), RefusalNotTarget)
	expect(t, "not listed", second(f.r.Open("ws", got[0].ID, "inc-99", human("ana"))), RefusalNotTarget)
	if must(f.r.Open("ws", got[0].ID, inc.ID, human("ana"))).ID != inc.ID {
		t.Fatal("open")
	}
	f.perms.deny("ana", "k1")
	expect(t, "revoked at opening", second(f.r.Open("ws", got[0].ID, inc.ID, human("ana"))), RefusalPermissionRevoked)
	f.perms.failing["ana"] = true
	expect(t, "unavailable at opening", second(f.r.Open("ws", got[0].ID, inc.ID, human("ana"))), RefusalPermissionUnavail)
}

func TestRenotifyEscalationAndDigests(t *testing.T) {
	f := open(t, testLimits())
	f.subscribe(t, "ana", "c", SeverityInfo)
	must(f.r.Configure("ws", Settings{Recipient: "ana", Zone: "UTC", Digest: DigestHourly, Renotify: map[string]time.Duration{SeverityWarning: 30 * time.Minute}}, human("ana")))
	a := f.raise(t, signal("s1", "c", "k1", SeverityWarning, false, start))
	b := f.raise(t, signal("s2", "c", "k2", SeverityInfo, false, start))
	got := f.cycle(t)
	if len(got) != 1 || !got[0].Digest || len(got[0].Delivered) != 2 {
		t.Fatalf("first digest: %+v", got)
	}
	f.clock.advance(31 * time.Minute)
	if got := f.cycle(t); len(got) != 0 {
		t.Fatal("digest cadence holds even when re-notification is due")
	}
	f.clock.advance(30 * time.Minute)
	got = f.cycle(t)
	if len(got) != 1 || len(got[0].Delivered) != 1 || got[0].Delivered[0] != a.ID {
		t.Fatalf("warning re-notified, info never: %+v", got)
	}
	f.raise(t, signal("s3", "c", "k2", SeverityCritical, false, f.clock.read()))
	got = f.cycle(t)
	if len(got) != 1 || got[0].Digest || got[0].Delivered[0] != b.ID {
		t.Fatalf("escalation to critical goes at once: %+v", got)
	}
	if got := f.cycle(t); len(got) != 0 {
		t.Fatal("no storm")
	}
}

func TestPendingNotificationsAndDispatchOrder(t *testing.T) {
	f := open(t, testLimits())
	f.subscribe(t, "ana", "c", SeverityInfo)
	f.raise(t, signal("s1", "c", "k1", SeverityWarning, false, start))
	n1 := must(f.r.Plan("ws", system("planner")))
	if again := must(f.r.Plan("ws", system("planner"))); len(again) != 0 {
		t.Fatal("pending blocks re-planning")
	}
	f.raise(t, signal("s2", "c", "k1", SeverityCritical, false, start))
	n2 := must(f.r.Plan("ws", system("planner")))
	if len(n2) != 1 {
		t.Fatal("critical escalation bypasses the pending notification")
	}
	first := must(f.r.Dispatch("ws", n2[0].ID, system("sender")))
	older := must(f.r.Dispatch("ws", n1[0].ID, system("sender")))
	if len(first.Delivered) != 1 || len(older.Delivered) != 0 {
		t.Fatalf("duplicates dropped whatever the order: %+v %+v", first, older)
	}
	g := open(t, testLimits())
	g.subscribe(t, "ana", "c", SeverityInfo)
	g.raise(t, signal("s1", "c", "k1", SeverityWarning, false, start))
	p1 := must(g.r.Plan("ws", system("planner")))
	g.clock.advance(time.Hour)
	p2 := must(g.r.Plan("ws", system("planner")))
	if len(p2) != 1 {
		t.Fatal("a pending notification goes stale after an hour")
	}
	a := must(g.r.Dispatch("ws", p1[0].ID, system("sender")))
	b := must(g.r.Dispatch("ws", p2[0].ID, system("sender")))
	if len(a.Delivered) != 1 || len(b.Delivered) != 0 {
		t.Fatalf("older first: %+v %+v", a, b)
	}
}

func TestAlternates(t *testing.T) {
	f := open(t, testLimits())
	f.subscribe(t, "ana", "c", SeverityInfo)
	f.subscribe(t, "bob", "c", SeverityInfo)
	must(f.r.Configure("ws", Settings{Recipient: "ana", Zone: "UTC", Digest: DigestImmediate, Alternate: "bob", Away: true}, human("ana")))
	f.raise(t, signal("s1", "c", "k1", SeverityWarning, false, start))
	got := f.cycle(t)
	if len(got) != 1 || got[0].Target != "bob" || len(got[0].Delivered) != 1 {
		t.Fatalf("merged copies for the alternate: %+v", got)
	}
	g := open(t, testLimits())
	g.subscribe(t, "ana", "c", SeverityInfo)
	must(g.r.Configure("ws", Settings{Recipient: "ana", Zone: "UTC", Digest: DigestImmediate, Alternate: "bob", Away: true}, human("ana")))
	must(g.r.Configure("ws", Settings{Recipient: "bob", Zone: "UTC", Digest: DigestImmediate, Alternate: "cy", Away: true}, human("bob")))
	g.raise(t, signal("s1", "c", "k1", SeverityWarning, false, start))
	if got := g.cycle(t); len(got) != 0 {
		t.Fatal("one hop only")
	}
	if w := must(g.r.Withholdings("ws", system("w"))); len(w) != 1 || w[0].Reason != WithheldNoTarget || w[0].Target != "ana" {
		t.Fatal(w)
	}
	h := open(t, testLimits())
	h.subscribe(t, "ana", "c", SeverityInfo)
	must(h.r.Configure("ws", Settings{Recipient: "ana", Zone: "UTC", Digest: DigestImmediate, Alternate: "bob", Away: true}, human("ana")))
	x := h.raise(t, signal("s1", "c", "k1", SeverityWarning, false, start))
	must(h.r.Snooze("ws", x.ID, start.Add(time.Hour), human("bob")))
	if got := h.cycle(t); len(got) != 0 {
		t.Fatal("the alternate snoozed its forwarded copy")
	}
}

func TestZonesAndDST(t *testing.T) {
	ny, _ := time.LoadLocation("America/New_York")
	kolkata, _ := time.LoadLocation("Asia/Kolkata")
	q := Settings{Quiet: &Quiet{Start: 1*60 + 30, End: 3 * 60}}
	if !quiet(q, ny, time.Date(2026, 3, 8, 6, 30, 0, 0, time.UTC)) || quiet(q, ny, time.Date(2026, 3, 8, 7, 0, 0, 0, time.UTC)) {
		t.Fatal("DST gap: 01:30 local counts, 03:00 local does not")
	}
	if !quiet(q, ny, time.Date(2026, 11, 1, 5, 45, 0, 0, time.UTC)) || !quiet(q, ny, time.Date(2026, 11, 1, 6, 45, 0, 0, time.UTC)) {
		t.Fatal("DST overlap: both 01:45 local instants are quiet")
	}
	if !quiet(Settings{Quiet: &Quiet{Start: 22*60 + 30, End: 23 * 60}}, kolkata, time.Date(2026, 11, 16, 17, 0, 0, 0, time.UTC)) {
		t.Fatal("half-hour offset")
	}
	if quiet(Settings{Quiet: &Quiet{Start: 60, End: 120}}, time.UTC, time.Date(2026, 11, 16, 2, 0, 0, 0, time.UTC)) || !quiet(Settings{Quiet: &Quiet{Start: 60, End: 120}}, time.UTC, time.Date(2026, 11, 16, 1, 0, 0, 0, time.UTC)) {
		t.Fatal("half-open window")
	}
	if quiet(Settings{}, time.UTC, start) {
		t.Fatal("no quiet hours")
	}
}
