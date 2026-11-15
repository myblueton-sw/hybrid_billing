package closing

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRegistryNeedsDependenciesAndLimits(t *testing.T) {
	f := open(t, testLimits())
	if _, err := NewRegistry(Config{}, testLimits(), f.clock.read); !errors.Is(err, ErrInvalid) {
		t.Fatal("zones")
	}
	if _, err := NewRegistry(Config{Zones: time.LoadLocation}, testLimits(), nil); !errors.Is(err, ErrInvalid) {
		t.Fatal("clock")
	}
	for _, change := range []func(*Limits){
		func(l *Limits) { l.Workspaces = 0 }, func(l *Limits) { l.Calendars = maxCalendars + 1 }, func(l *Limits) { l.Cycles = 0 }, func(l *Limits) { l.Targets = 0 },
		func(l *Limits) { l.History = maxHistory + 1 }, func(l *Limits) { l.Claims = 0 }, func(l *Limits) { l.Lead = minLead - 1 }, func(l *Limits) { l.Lead = maxLead + 1 },
	} {
		l := testLimits()
		change(&l)
		if _, err := NewRegistry(Config{Zones: time.LoadLocation}, l, f.clock.read); !errors.Is(err, ErrInvalid) {
			t.Fatalf("%+v", l)
		}
	}
	for _, l := range []Limits{{maxWorkspaces, maxCalendars, maxCycles, maxTargets, maxHistory, maxClaims, maxLead}, {1, 1, 1, 1, 1, 1, minLead}} {
		if _, err := NewRegistry(Config{Zones: time.LoadLocation}, l, f.clock.read); err != nil {
			t.Fatal(err)
		}
	}
	if (Refusal{Reason: "x"}).Error() != ErrRejected.Error()+": x" || !errors.Is(Refusal{Reason: "x"}, ErrRejected) {
		t.Fatal("refusal")
	}
	if !validID(strings.Repeat("Z", maxIDBytes)) || validID(strings.Repeat("z", maxIDBytes+1)) || validID("!a") || validID("a!") || validID("") || !validID("Az09._:-") {
		t.Fatal("identifiers")
	}
}

func TestCalendarGates(t *testing.T) {
	f := open(t, testLimits())
	good := seoul()
	for _, change := range []func(*Calendar){
		func(c *Calendar) { c.ID = "" }, func(c *Calendar) { c.Zone = "" }, func(c *Calendar) { c.Zone = "Local" }, func(c *Calendar) { c.Zone = strings.Repeat("Z", maxIDBytes+1) },
		func(c *Calendar) { c.Weekend = []time.Weekday{0, 1, 2, 3, 4, 5, 6} }, func(c *Calendar) { c.Weekend = []time.Weekday{1, 1} },
		func(c *Calendar) { c.Weekend = []time.Weekday{7} }, func(c *Calendar) { c.Weekend = []time.Weekday{-1} },
		func(c *Calendar) { c.Holidays = make([]string, maxHolidays+1) }, func(c *Calendar) { c.Holidays = []string{"2026-02-30"} },
		func(c *Calendar) { c.Close = 0 }, func(c *Calendar) { c.Close = endOfDay + 1 }, func(c *Calendar) { c.Covers.From = "x" },
		func(c *Calendar) { c.Covers.To = "x" }, func(c *Calendar) { c.Covers = Range{From: "2026-02-01", To: "2026-01-31"} },
	} {
		c := seoul()
		change(&c)
		expect(t, "form", second(f.r.ProposeCalendar("ws", c, human("planner"))), "invalid")
	}
	expect(t, "workspace", second(f.r.ProposeCalendar("", good, human("planner"))), "invalid")
	expect(t, "actor", second(f.r.ProposeCalendar("ws", good, Actor{ID: "x"})), "invalid")
	expect(t, "assistant", second(f.r.ProposeCalendar("ws", good, assistant("bot"))), RefusalAINotAllowed)
	expect(t, "system", second(f.r.ProposeCalendar("ws", good, system("job"))), RefusalNotHuman)
	zone := good
	zone.Zone = "Mars/Base"
	expect(t, "zone", second(f.r.ProposeCalendar("ws", zone, human("planner"))), RefusalZoneUnavailable)
	edges := good
	edges.Weekend = []time.Weekday{0, 1, 2, 3, 4, 6}
	edges.Close = endOfDay
	edges.Covers = Range{From: "2026-01-01", To: "2026-01-01"}
	edges.ID = "edge"
	must(f.r.ProposeCalendar("ws", edges, human("planner")))
	v := must(f.r.ProposeCalendar("ws", good, human("planner")))
	v.Calendar.Holidays[0] = "mutated"
	if again := must(f.r.ProposeCalendar("ws", good, human("planner"))); again.Revision != 1 || again.Calendar.Holidays[0] != "2026-11-03" {
		t.Fatal("repeat while pending")
	}
	expect(t, "other proposer while pending", second(f.r.ProposeCalendar("ws", good, human("other"))), RefusalCalendarPending)
	for _, change := range []func(*Calendar){
		func(c *Calendar) { c.Zone = "UTC" }, func(c *Calendar) { c.Close = 60 }, func(c *Calendar) { c.Covers.To = "2027-12-30" },
		func(c *Calendar) { c.Weekend = []time.Weekday{time.Saturday} }, func(c *Calendar) { c.Weekend = []time.Weekday{time.Sunday, time.Saturday} },
		func(c *Calendar) { c.Holidays = nil }, func(c *Calendar) { c.Holidays = []string{"2026-11-04"} },
	} {
		c := seoul()
		change(&c)
		expect(t, "changed while pending", second(f.r.ProposeCalendar("ws", c, human("planner"))), RefusalCalendarPending)
	}
	expect(t, "approve form", second(f.r.ApproveCalendar("ws", "kr", 0, human("lead"))), "invalid")
	expect(t, "approve workspace", second(f.r.ApproveCalendar("", "kr", 1, human("lead"))), "invalid")
	expect(t, "approve id", second(f.r.ApproveCalendar("ws", "", 1, human("lead"))), "invalid")
	expect(t, "approve actor", second(f.r.ApproveCalendar("ws", "kr", 1, Actor{ID: "x"})), "invalid")
	expect(t, "approve system", second(f.r.ApproveCalendar("ws", "kr", 1, system("x"))), RefusalNotHuman)
	expect(t, "approve unknown", second(f.r.ApproveCalendar("ws", "kr", 2, human("lead"))), "not_found")
	expect(t, "approve workspace unknown", second(f.r.ApproveCalendar("ws9", "kr", 1, human("lead"))), "not_found")
	expect(t, "approve self", second(f.r.ApproveCalendar("ws", "kr", 1, human("planner"))), RefusalSelfApproval)
	must(f.r.ApproveCalendar("ws", "kr", 1, human("lead")))
	if must(f.r.ApproveCalendar("ws", "kr", 1, human("lead"))).Approver != "lead" {
		t.Fatal("approve repeat")
	}
	expect(t, "second approver", second(f.r.ApproveCalendar("ws", "kr", 1, human("boss"))), "conflict")
	if next := must(f.r.ProposeCalendar("ws", good, human("planner"))); next.Revision != 2 {
		t.Fatal("next revision")
	}
	l := testLimits()
	l.Calendars, l.Workspaces = 1, 1
	g := open(t, l)
	must(g.r.ProposeCalendar("ws", good, human("planner")))
	other := good
	other.ID = "us"
	expect(t, "calendars", second(g.r.ProposeCalendar("ws", other, human("planner"))), "limit")
	expect(t, "workspaces", second(g.r.ProposeCalendar("ws2", good, human("planner"))), "limit")
}

func TestCycleGates(t *testing.T) {
	f := open(t, testLimits())
	f.calendar(t, "ws", seoul())
	good := Cycle{ID: "oct", Period: "2026-10", Calendar: "kr", Revision: 1, Stages: stages()}
	for _, change := range []func(*Cycle){
		func(c *Cycle) { c.ID = "" }, func(c *Cycle) { c.Period = "2026-13" }, func(c *Cycle) { c.Period = "2026-1" }, func(c *Cycle) { c.Calendar = "" },
		func(c *Cycle) { c.Revision = 0 }, func(c *Cycle) { c.Stages = nil }, func(c *Cycle) { c.Stages = []StageSpec{{Stage: "x", Rule: Rule{Days: 1}, Basis: BasisFile}} },
		func(c *Cycle) {
			c.Stages = []StageSpec{{Stage: StageValidate, Rule: Rule{Days: 1}, Basis: BasisFile}, {Stage: StageCollect, Rule: Rule{Days: 2}, Basis: BasisFile}}
		},
		func(c *Cycle) {
			c.Stages = []StageSpec{{Stage: StageCollect, Rule: Rule{Days: 1}, Basis: BasisFile}, {Stage: StageCollect, Rule: Rule{Days: 2}, Basis: BasisFile}}
		},
		func(c *Cycle) { c.Stages = []StageSpec{{Stage: StageCollect, Rule: Rule{Days: 1}, Basis: "paper"}} },
		func(c *Cycle) { c.Stages = []StageSpec{{Stage: StageCollect, Rule: Rule{Days: 0}, Basis: BasisFile}} },
		func(c *Cycle) {
			c.Stages = []StageSpec{{Stage: StageCollect, Rule: Rule{Days: maxDays + 1}, Basis: BasisFile}}
		},
	} {
		c := good
		c.Stages = stages()
		change(&c)
		expect(t, "form", second(f.r.OpenCycle("ws", c, human("planner"))), "invalid")
	}
	expect(t, "workspace", second(f.r.OpenCycle("", good, human("planner"))), "invalid")
	expect(t, "actor", second(f.r.OpenCycle("ws", good, Actor{ID: "x"})), "invalid")
	expect(t, "system", second(f.r.OpenCycle("ws", good, system("x"))), RefusalNotHuman)
	expect(t, "unknown workspace", second(f.r.OpenCycle("ws2", good, human("planner"))), "not_found")
	missing := good
	missing.Revision = 2
	expect(t, "unknown revision", second(f.r.OpenCycle("ws", missing, human("planner"))), "not_found")
	must(f.r.ProposeCalendar("ws", seoul(), human("planner")))
	pending := good
	pending.Revision, pending.ID = 2, "pending"
	expect(t, "pending calendar", second(f.r.OpenCycle("ws", pending, human("planner"))), RefusalNotApproved)
	backwards := good
	backwards.ID = "back"
	backwards.Stages = []StageSpec{{Stage: StageCollect, Rule: Rule{Days: 5, Business: true}, Basis: BasisFile}, {Stage: StageIssue, Rule: Rule{Days: 1, Business: true}, Basis: BasisERP}}
	expect(t, "stage order", second(f.r.OpenCycle("ws", backwards, human("planner"))), RefusalStageOrder)
	outside := good
	outside.ID, outside.Period = "out", "2027-12"
	expect(t, "outside", second(f.r.OpenCycle("ws", outside, human("planner"))), RefusalOutsideCalendar)
	v := must(f.r.OpenCycle("ws", good, human("planner")))
	v.Stages[0].History = append(v.Stages[0].History, DueChange{})
	if again := must(f.r.OpenCycle("ws", good, human("planner"))); len(again.Stages[0].History) != 0 {
		t.Fatal("repeat")
	}
	expect(t, "other opener", second(f.r.OpenCycle("ws", good, human("other"))), "conflict")
	for _, change := range []func(*Cycle){
		func(c *Cycle) { c.Period = "2026-09" }, func(c *Cycle) { c.Revision = 2 }, func(c *Cycle) { c.Calendar = "us" },
		func(c *Cycle) { c.Stages = c.Stages[:1] }, func(c *Cycle) { c.Stages[0].Rule.Days = 2 },
	} {
		c := good
		c.Stages = stages()
		change(&c)
		expect(t, "changed", second(f.r.OpenCycle("ws", c, human("planner"))), "conflict")
	}
	expect(t, "cycleof form", second(f.r.CycleOf("", "oct")), "invalid")
	expect(t, "cycleof id", second(f.r.CycleOf("ws", "")), "invalid")
	expect(t, "cycleof unknown", second(f.r.CycleOf("ws", "nope")), "not_found")
	l := testLimits()
	l.Cycles = 1
	g := open(t, l)
	g.cycle(t, "a")
	other := good
	other.ID = "b"
	expect(t, "cycles", second(g.r.OpenCycle("ws", other, human("planner"))), "limit")
}

func TestTargetGates(t *testing.T) {
	f := open(t, testLimits())
	f.cycle(t, "oct")
	add := Addition{Targets: []string{"a"}, Reason: "r"}
	for _, bad := range []Addition{{Reason: "r"}, {Targets: []string{"a"}}, {Targets: []string{""}, Reason: "r"}, {Targets: []string{"a", "a"}, Reason: "r"}} {
		expect(t, "form", second(f.r.AddTargets("ws", "oct", bad, human("planner"))), "invalid")
	}
	expect(t, "workspace", second(f.r.AddTargets("", "oct", add, human("planner"))), "invalid")
	expect(t, "id", second(f.r.AddTargets("ws", "", add, human("planner"))), "invalid")
	expect(t, "actor", second(f.r.AddTargets("ws", "oct", add, Actor{ID: "x"})), "invalid")
	expect(t, "system", second(f.r.AddTargets("ws", "oct", add, system("x"))), RefusalNotHuman)
	expect(t, "unknown", second(f.r.AddTargets("ws", "nope", add, human("planner"))), "not_found")
	must(f.r.AddTargets("ws", "oct", add, human("planner")))
	if again := must(f.r.AddTargets("ws", "oct", add, human("planner"))); again.Sequence != 2 || len(again.Targets) != 1 {
		t.Fatal("an identical repeat returns the cycle")
	}
	expect(t, "present", second(f.r.AddTargets("ws", "oct", Addition{Targets: []string{"a"}, Reason: "other"}, human("planner"))), RefusalTargetPresent)
	expect(t, "present for another person", second(f.r.AddTargets("ws", "oct", add, human("other"))), RefusalTargetPresent)
	expect(t, "partly present", second(f.r.AddTargets("ws", "oct", Addition{Targets: []string{"a", "b"}, Reason: "r"}, human("planner"))), RefusalTargetPresent)
	l := testLimits()
	l.Targets = 1
	g := open(t, l)
	g.cycle(t, "oct", "a")
	expect(t, "targets", second(g.r.AddTargets("ws", "oct", Addition{Targets: []string{"b"}, Reason: "r"}, human("planner"))), "limit")
	l = testLimits()
	l.History = 4
	h := open(t, l)
	h.cycle(t, "oct", "a")
	expect(t, "history", second(h.r.AddTargets("ws", "oct", Addition{Targets: []string{"b"}, Reason: "r"}, human("planner"))), "limit")
	expect(t, "history complete", second(h.r.Complete("ws", "oct", "a", StageCollect, evidence(EvidenceFile), system("feed"))), "limit")
}

func TestItemGates(t *testing.T) {
	f := open(t, testLimits())
	f.cycle(t, "oct", "a")
	ev := evidence(EvidenceFile)
	expect(t, "form", second(f.r.Complete("ws", "oct", "a", StageCollect, Evidence{Kind: EvidenceFile}, system("feed"))), "invalid")
	expect(t, "kind", second(f.r.Complete("ws", "oct", "a", StageCollect, Evidence{Reference: "r"}, system("feed"))), "invalid")
	expect(t, "stage", second(f.r.Complete("ws", "oct", "a", "nope", ev, system("feed"))), "invalid")
	expect(t, "workspace", second(f.r.Complete("", "oct", "a", StageCollect, ev, system("feed"))), "invalid")
	expect(t, "target", second(f.r.Complete("ws", "oct", "", StageCollect, ev, system("feed"))), "invalid")
	expect(t, "actor", second(f.r.Complete("ws", "oct", "a", StageCollect, ev, Actor{ID: "x"})), "invalid")
	expect(t, "assistant", second(f.r.Complete("ws", "oct", "a", StageCollect, ev, assistant("bot"))), RefusalAINotAllowed)
	expect(t, "unknown cycle", second(f.r.Complete("ws", "nope", "a", StageCollect, ev, system("feed"))), "not_found")
	expect(t, "unknown target", second(f.r.Complete("ws", "oct", "z", StageCollect, ev, system("feed"))), "not_found")
	expect(t, "stage not in cycle", second(f.r.Complete("ws", "oct", "a", StageApprove, ev, system("feed"))), "not_found")
	must(f.r.Complete("ws", "oct", "a", StageCollect, ev, system("feed")))
	if must(f.r.Complete("ws", "oct", "a", StageCollect, ev, system("other"))).State != ItemDone {
		t.Fatal("repeat")
	}
	expect(t, "other evidence", second(f.r.Complete("ws", "oct", "a", StageCollect, Evidence{Kind: EvidenceFile, Reference: "other"}, system("feed"))), "conflict")
	expect(t, "reopen form", second(f.r.Reopen("ws", "oct", "a", StageCollect, "", human("ops"))), "invalid")
	expect(t, "reopen system", second(f.r.Reopen("ws", "oct", "a", StageCollect, "r", system("x"))), RefusalNotHuman)
	expect(t, "reopen unknown", second(f.r.Reopen("ws", "oct", "z", StageCollect, "r", human("ops"))), "not_found")
	expect(t, "reopen open", second(f.r.Reopen("ws", "oct", "a", StageValidate, "r", human("ops"))), RefusalState)
	expect(t, "hold form", second(f.r.Hold("ws", "oct", "a", StageValidate, "", human("ops"))), "invalid")
	expect(t, "hold form2", second(f.r.Hold("ws", "oct", "", StageValidate, "r", human("ops"))), "invalid")
	expect(t, "hold system", second(f.r.Hold("ws", "oct", "a", StageValidate, "r", system("x"))), RefusalNotHuman)
	expect(t, "hold unknown", second(f.r.Hold("ws", "oct", "z", StageValidate, "r", human("ops"))), "not_found")
	expect(t, "hold done", second(f.r.Hold("ws", "oct", "a", StageCollect, "r", human("ops"))), RefusalState)
	expect(t, "release open", second(f.r.Release("ws", "oct", "a", StageValidate, human("ops"))), RefusalState)
	must(f.r.Hold("ws", "oct", "a", StageValidate, "r", human("ops")))
	expect(t, "complete held", second(f.r.Complete("ws", "oct", "a", StageValidate, evidence(EvidenceInternal), human("ana"))), RefusalState)
	must(f.r.RequestWaiver("ws", "oct", "a", StageValidate, Waiver{Reason: "r", Evidence: "e"}, human("ops")))
	expect(t, "release pending", second(f.r.Release("ws", "oct", "a", StageValidate, human("ops"))), RefusalWaiverPending)
	expect(t, "waiver form", second(f.r.RequestWaiver("ws", "oct", "a", StageValidate, Waiver{Reason: "r"}, human("ops"))), "invalid")
	expect(t, "waiver form2", second(f.r.RequestWaiver("ws", "oct", "a", StageValidate, Waiver{Evidence: "e"}, human("ops"))), "invalid")
	expect(t, "waiver form3", second(f.r.RequestWaiver("", "oct", "a", StageValidate, Waiver{Reason: "r", Evidence: "e"}, human("ops"))), "invalid")
	expect(t, "waiver system", second(f.r.RequestWaiver("ws", "oct", "a", StageValidate, Waiver{Reason: "r", Evidence: "e"}, system("x"))), RefusalNotHuman)
	expect(t, "waiver unknown", second(f.r.RequestWaiver("ws", "oct", "z", StageValidate, Waiver{Reason: "r", Evidence: "e"}, human("ops"))), "not_found")
	if must(f.r.RequestWaiver("ws", "oct", "a", StageValidate, Waiver{Reason: "r", Evidence: "e"}, human("ops"))).Waiver == nil {
		t.Fatal("waiver repeat")
	}
	expect(t, "second waiver", second(f.r.RequestWaiver("ws", "oct", "a", StageValidate, Waiver{Reason: "r", Evidence: "e"}, human("boss"))), RefusalWaiverPending)
	expect(t, "withdraw by other", second(f.r.WithdrawWaiver("ws", "oct", "a", StageValidate, human("boss"))), RefusalNotRequester)
	must(f.r.WithdrawWaiver("ws", "oct", "a", StageValidate, human("ops")))
	expect(t, "nothing pending", second(f.r.ApproveWaiver("ws", "oct", "a", StageValidate, human("lead"))), RefusalNotPending)
	expect(t, "decide form", second(f.r.ApproveWaiver("", "oct", "a", StageValidate, human("lead"))), "invalid")
	expect(t, "decide system", second(f.r.ApproveWaiver("ws", "oct", "a", StageValidate, system("lead"))), RefusalNotHuman)
	expect(t, "decide unknown", second(f.r.ApproveWaiver("ws", "oct", "z", StageValidate, human("lead"))), "not_found")
	expect(t, "waiver on done", second(f.r.RequestWaiver("ws", "oct", "a", StageCollect, Waiver{Reason: "r", Evidence: "e"}, human("ops"))), RefusalState)
	must(f.r.Release("ws", "oct", "a", StageValidate, human("ops")))
	expect(t, "item form", second(f.r.Item("", "oct", "a", StageCollect, time.Time{})), "invalid")
	expect(t, "item target", second(f.r.Item("ws", "oct", "", StageCollect, time.Time{})), "invalid")
	expect(t, "item stage", second(f.r.Item("ws", "oct", "a", "x", time.Time{})), "invalid")
	expect(t, "item id", second(f.r.Item("ws", "", "a", StageCollect, time.Time{})), "invalid")
	expect(t, "item unknown", second(f.r.Item("ws", "oct", "z", StageCollect, time.Time{})), "not_found")
	v := must(f.r.CycleOf("ws", "oct"))
	if it := must(f.r.Item("ws", "oct", "a", StageValidate, v.Stages[1].OriginalDue.Add(2*time.Hour))); it.LateOpen != 2*time.Hour {
		t.Fatal(it.LateOpen)
	}
	if it := must(f.r.Item("ws", "oct", "a", StageValidate, time.Time{})); it.LateOpen != 0 {
		t.Fatal("not late yet")
	}
	if it := must(f.r.Item("ws", "oct", "a", StageCollect, v.Stages[0].OriginalDue.Add(time.Hour))); it.LateOpen != 0 {
		t.Fatal("done items have no open lateness")
	}
	expect(t, "progress form", second(f.r.Progress("", "oct", StageCollect, AsOf{})), "invalid")
	expect(t, "progress id", second(f.r.Progress("ws", "", StageCollect, AsOf{})), "invalid")
	expect(t, "progress stage", second(f.r.Progress("ws", "oct", "x", AsOf{})), "invalid")
	expect(t, "progress negative", second(f.r.Progress("ws", "oct", StageCollect, AsOf{Sequence: -1})), "invalid")
	expect(t, "progress unknown", second(f.r.Progress("ws", "nope", StageCollect, AsOf{})), "not_found")
	expect(t, "progress missing stage", second(f.r.Progress("ws", "oct", StageApprove, AsOf{})), "not_found")
	if p := must(f.r.Progress("ws", "oct", StageValidate, AsOf{At: v.Stages[1].OriginalDue})); p.OpenPastDue != 0 {
		t.Fatal("exactly at the due instant is not past due")
	}
	if p := must(f.r.Progress("ws", "oct", StageCollect, AsOf{})); p.DoneLate != 0 || p.Done != 1 {
		t.Fatal("on time")
	}
}

func TestPostponementGates(t *testing.T) {
	f := open(t, testLimits())
	v := f.cycle(t, "oct", "a")
	due := v.Stages[0].CurrentDue.Add(time.Hour)
	p := Postponement{Due: due, Reason: "r"}
	expect(t, "form", second(f.r.RequestPostponement("ws", "oct", StageCollect, Postponement{Reason: "r"}, human("ops"))), "invalid")
	expect(t, "reason", second(f.r.RequestPostponement("ws", "oct", StageCollect, Postponement{Due: due}, human("ops"))), "invalid")
	expect(t, "stage", second(f.r.RequestPostponement("ws", "oct", "x", p, human("ops"))), "invalid")
	expect(t, "workspace", second(f.r.RequestPostponement("", "oct", StageCollect, p, human("ops"))), "invalid")
	expect(t, "actor", second(f.r.RequestPostponement("ws", "oct", StageCollect, p, Actor{ID: "x"})), "invalid")
	expect(t, "system", second(f.r.RequestPostponement("ws", "oct", StageCollect, p, system("x"))), RefusalNotHuman)
	expect(t, "unknown", second(f.r.RequestPostponement("ws", "nope", StageCollect, p, human("ops"))), "not_found")
	expect(t, "missing stage", second(f.r.RequestPostponement("ws", "oct", StageApprove, p, human("ops"))), "not_found")
	expect(t, "not later", second(f.r.RequestPostponement("ws", "oct", StageCollect, Postponement{Due: v.Stages[0].CurrentDue, Reason: "r"}, human("ops"))), RefusalNotLater)
	expect(t, "last stage has no next", second(f.r.RequestPostponement("ws", "oct", StageSettle, Postponement{Due: v.Stages[3].CurrentDue.Add(time.Hour), Reason: "r"}, human("ops"))), "ok")
	must(f.r.RequestPostponement("ws", "oct", StageCollect, p, human("ops")))
	if must(f.r.RequestPostponement("ws", "oct", StageCollect, p, human("ops"))).Stages[0].Pending == nil {
		t.Fatal("repeat")
	}
	expect(t, "second request", second(f.r.RequestPostponement("ws", "oct", StageCollect, p, human("boss"))), RefusalPostponePending)
	expect(t, "withdraw by other", second(f.r.WithdrawPostponement("ws", "oct", StageCollect, human("boss"))), RefusalNotRequester)
	expect(t, "decide form", second(f.r.ApprovePostponement("", "oct", StageCollect, human("lead"))), "invalid")
	expect(t, "decide system", second(f.r.ApprovePostponement("ws", "oct", StageCollect, system("lead"))), RefusalNotHuman)
	expect(t, "decide unknown", second(f.r.ApprovePostponement("ws", "nope", StageCollect, human("lead"))), "not_found")
	expect(t, "decide missing stage", second(f.r.ApprovePostponement("ws", "oct", StageApprove, human("lead"))), "not_found")
	w := must(f.r.WithdrawPostponement("ws", "oct", StageCollect, human("ops")))
	if w.Stages[0].Pending != nil || !w.Stages[0].CurrentDue.Equal(v.Stages[0].CurrentDue) || w.Stages[0].History[1].Kind != EventPostponeWithdraw {
		t.Fatal(w.Stages[0])
	}
	expect(t, "nothing pending", second(f.r.ApprovePostponement("ws", "oct", StageCollect, human("lead"))), RefusalNotPending)
	g := open(t, testLimits())
	g.cycle(t, "oct", "a")
	must(g.r.RequestPostponement("ws", "oct", StageCollect, p, human("ops")))
	g.clock.set(due.Add(time.Minute))
	expect(t, "approval after the new due", second(g.r.ApprovePostponement("ws", "oct", StageCollect, human("lead"))), RefusalNotLater)
	must(g.r.WithdrawPostponement("ws", "oct", StageCollect, human("ops")))
	expect(t, "in the past", second(g.r.RequestPostponement("ws", "oct", StageCollect, p, human("ops"))), RefusalNotLater)
	l := testLimits()
	l.History = 5
	h := open(t, l)
	h.cycle(t, "oct", "a")
	must(h.r.RequestPostponement("ws", "oct", StageCollect, p, human("ops")))
	expect(t, "history request", second(h.r.RequestPostponement("ws", "oct", StageValidate, Postponement{Due: v.Stages[1].CurrentDue.Add(time.Hour), Reason: "r"}, human("ops"))), "limit")
	expect(t, "history decide", second(h.r.ApprovePostponement("ws", "oct", StageCollect, human("lead"))), "limit")
	expect(t, "history hold", second(h.r.Hold("ws", "oct", "a", StageCollect, "r", human("ops"))), "limit")
	expect(t, "history waiver", second(h.r.RequestWaiver("ws", "oct", "a", StageCollect, Waiver{Reason: "r", Evidence: "e"}, human("ops"))), "limit")
}

func TestClaimGates(t *testing.T) {
	f := open(t, testLimits())
	expect(t, "limit", second(f.r.Claim("ws", 0, system("s"))), "invalid")
	expect(t, "limit high", second(f.r.Claim("ws", 51, system("s"))), "invalid")
	expect(t, "workspace", second(f.r.Claim("", 1, system("s"))), "invalid")
	expect(t, "actor", second(f.r.Claim("ws", 1, Actor{ID: "s"})), "invalid")
	expect(t, "assistant", second(f.r.Claim("ws", 1, assistant("s"))), RefusalAINotAllowed)
	expect(t, "human", second(f.r.Claim("ws", 1, human("s"))), RefusalNotSystem)
	if got := must(f.r.Claim("ws9", 1, system("s"))); got != nil {
		t.Fatal("empty workspace")
	}
	v := f.cycle(t, "oct", "a")
	f.clock.set(v.Stages[0].CurrentDue)
	got := must(f.r.Claim("ws", 10, system("s")))
	if len(got) != 1 || got[0].Step != StepDue || len(got[0].Skipped) != 1 {
		t.Fatalf("at the due instant: %+v", got)
	}
	must(f.r.Hold("ws", "oct", "a", StageCollect, "r", human("ops")))
	f.clock.set(v.Stages[0].CurrentDue.Add(overdueAfter))
	collect := func(events []Event) []Event {
		var out []Event
		for _, e := range events {
			if e.Stage == StageCollect {
				out = append(out, e)
			}
		}
		return out
	}
	if got := collect(must(f.r.Claim("ws", 10, system("s")))); len(got) != 0 {
		t.Fatal("held items are not claimed")
	}
	must(f.r.Release("ws", "oct", "a", StageCollect, human("ops")))
	if got := collect(must(f.r.Claim("ws", 10, system("s")))); len(got) != 1 || got[0].Step != StepOverdue || len(got[0].Skipped) != 0 {
		t.Fatalf("released items return: %+v", got)
	}
	must(f.r.AddTargets("ws", "oct", Addition{Targets: []string{"late"}, Reason: "r"}, human("planner")))
	if got := collect(must(f.r.Claim("ws", 10, system("s")))); len(got) != 1 || got[0].Targets[0] != "late" {
		t.Fatalf("late additions get their own events: %+v", got)
	}
}

func TestHistoryLimitsAndClaimOrdering(t *testing.T) {
	l := testLimits()
	l.History = 5
	f := open(t, l)
	f.cycle(t, "oct", "a")
	must(f.r.Complete("ws", "oct", "a", StageCollect, evidence(EvidenceFile), system("feed")))
	expect(t, "reopen history", second(f.r.Reopen("ws", "oct", "a", StageCollect, "r", human("ops"))), "limit")
	g := open(t, Limits{Workspaces: 1, Calendars: 2, Cycles: 4, Targets: 10, History: 5, Claims: 10, Lead: time.Hour})
	g.cycle(t, "oct", "a")
	must(g.r.RequestWaiver("ws", "oct", "a", StageCollect, Waiver{Reason: "r", Evidence: "e"}, human("ops")))
	expect(t, "waiver decision history", second(g.r.ApproveWaiver("ws", "oct", "a", StageCollect, human("lead"))), "limit")
	h := open(t, testLimits())
	h.calendar(t, "ws", seoul())
	for _, id := range []string{"b-cycle", "a-cycle"} {
		must(h.r.OpenCycle("ws", Cycle{ID: id, Period: "2026-10", Calendar: "kr", Revision: 1, Stages: stages()}, human("planner")))
		must(h.r.AddTargets("ws", id, Addition{Targets: []string{"x", "y"}, Reason: "r"}, human("planner")))
	}
	v := must(h.r.CycleOf("ws", "a-cycle"))
	h.clock.set(v.Stages[1].CurrentDue)
	got := must(h.r.Claim("ws", 5, system("s")))
	var flat [][3]string
	for _, e := range got {
		for _, target := range e.Targets {
			flat = append(flat, [3]string{e.Cycle, e.Stage, target})
		}
	}
	want := [][3]string{{"a-cycle", StageCollect, "x"}, {"a-cycle", StageCollect, "y"}, {"b-cycle", StageCollect, "x"}, {"b-cycle", StageCollect, "y"}, {"a-cycle", StageValidate, "x"}}
	if len(flat) != len(want) {
		t.Fatalf("claims: %+v", got)
	}
	for index := range want {
		if flat[index] != want[index] {
			t.Fatalf("order: %+v", flat)
		}
	}
}
