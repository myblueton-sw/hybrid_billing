package closing

import (
	"strings"
	"testing"
	"time"
)

func TestClaimsReportTheDueInstantAndOrderByIt(t *testing.T) {
	f := open(t, Limits{Workspaces: 1, Calendars: 2, Cycles: 4, Targets: 10, History: 100, Claims: 1, Lead: 48 * time.Hour})
	f.calendar(t, "ws", seoul())
	must(f.r.OpenCycle("ws", Cycle{ID: "x", Period: "2026-10", Calendar: "kr", Revision: 1, Stages: []StageSpec{{Stage: StageCollect, Rule: Rule{Days: 1, Business: true}, Basis: BasisFile}}}, human("planner")))
	must(f.r.OpenCycle("ws", Cycle{ID: "y", Period: "2026-10", Calendar: "kr", Revision: 1, Stages: []StageSpec{{Stage: StageCollect, Rule: Rule{Days: 2, Business: true}, Basis: BasisFile}}}, human("planner")))
	for _, id := range []string{"x", "y"} {
		must(f.r.AddTargets("ws", id, Addition{Targets: []string{"a"}, Reason: "r"}, human("planner")))
	}
	x := must(f.r.CycleOf("ws", "x")).Stages[0].CurrentDue
	y := must(f.r.CycleOf("ws", "y")).Stages[0].CurrentDue
	f.clock.set(x.Add(overdueAfter))
	got := must(f.r.Claim("ws", 1, system("s")))
	if len(got) != 1 || got[0].Cycle != "x" || got[0].Step != StepOverdue || !got[0].Due.Equal(x) || !got[0].At.Equal(x.Add(overdueAfter)) {
		t.Fatalf("the earlier due date comes first: %+v", got)
	}
	got = must(f.r.Claim("ws", 1, system("s")))
	if len(got) != 1 || got[0].Cycle != "y" || !got[0].Due.Equal(y) || !got[0].At.Equal(y.Add(-48*time.Hour)) {
		t.Fatalf("due-soon event reports the due date: %+v", got)
	}
	f.clock.set(y.Add(overdueAfter))
	must(f.r.Claim("ws", 1, system("s")))
	if again := must(f.r.Claim("ws", 1, system("s"))); len(again) != 0 {
		t.Fatal("handed-out stages are not revisited")
	}
	if st := f.r.workspaces["ws"].cycles["y"].stages[0]; len(st.owed) != 0 {
		t.Fatal("the owed set empties once every step is handed out")
	}
}

func TestRepeatsAreIdempotent(t *testing.T) {
	f := open(t, testLimits())
	v := f.cycle(t, "oct", "a")
	must(f.r.Complete("ws", "oct", "a", StageCollect, evidence(EvidenceFile), system("feed")))
	must(f.r.Complete("ws", "oct", "a", StageValidate, evidence(EvidenceInternal), human("ana")))
	expect(t, "a system repeat of an internal approval", second(f.r.Complete("ws", "oct", "a", StageValidate, evidence(EvidenceInternal), system("feed"))), RefusalNotHuman)
	must(f.r.RequestWaiver("ws", "oct", "a", StageIssue, Waiver{Reason: "r", Evidence: "e"}, human("ops")))
	must(f.r.ApproveWaiver("ws", "oct", "a", StageIssue, human("lead")))
	if again := must(f.r.ApproveWaiver("ws", "oct", "a", StageIssue, human("lead"))); again.State != ItemWaived {
		t.Fatal("waiver approval repeat")
	}
	expect(t, "another approver", second(f.r.ApproveWaiver("ws", "oct", "a", StageIssue, human("boss"))), RefusalNotPending)
	must(f.r.RequestPostponement("ws", "oct", StageCollect, Postponement{Due: v.Stages[0].CurrentDue.Add(time.Hour), Reason: "r"}, human("ops")))
	must(f.r.ApprovePostponement("ws", "oct", StageCollect, human("lead")))
	if again := must(f.r.ApprovePostponement("ws", "oct", StageCollect, human("lead"))); !again.Stages[0].CurrentDue.Equal(v.Stages[0].CurrentDue.Add(time.Hour)) {
		t.Fatal("postponement approval repeat")
	}
	expect(t, "another approver", second(f.r.ApprovePostponement("ws", "oct", StageCollect, human("boss"))), RefusalNotPending)
	must(f.r.RequestPostponement("ws", "oct", StageValidate, Postponement{Due: v.Stages[1].CurrentDue.Add(time.Hour), Reason: "r"}, human("ops")))
	must(f.r.WithdrawPostponement("ws", "oct", StageValidate, human("ops")))
	expect(t, "approve after withdrawal", second(f.r.ApprovePostponement("ws", "oct", StageValidate, human("lead"))), RefusalNotPending)
}

func TestWorkspacesAreIsolated(t *testing.T) {
	f := open(t, testLimits())
	f.cycle(t, "oct", "a")
	f.calendar(t, "ws", seoul())
	f.calendar(t, "ws2", seoul())
	expect(t, "a calendar revision of another workspace", second(f.r.OpenCycle("ws2", Cycle{ID: "oct", Period: "2026-10", Calendar: "kr", Revision: 2, Stages: stages()}, human("planner"))), "not_found")
	must(f.r.OpenCycle("ws2", Cycle{ID: "oct", Period: "2026-10", Calendar: "kr", Revision: 1, Stages: stages()}, human("planner")))
	expect(t, "items of another workspace", second(f.r.Complete("ws2", "oct", "a", StageCollect, evidence(EvidenceFile), system("feed"))), "not_found")
	if p := must(f.r.Progress("ws2", "oct", StageCollect, AsOf{})); p.Targets != 0 {
		t.Fatal(p)
	}
}

func TestCycleOnADSTCalendar(t *testing.T) {
	f := open(t, testLimits())
	us := Calendar{ID: "us", Zone: "America/New_York", Weekend: []time.Weekday{time.Saturday, time.Sunday}, Close: 2*60 + 30, Covers: Range{From: "2026-01-01", To: "2026-12-31"}}
	f.calendar(t, "ws", us)
	v := must(f.r.OpenCycle("ws", Cycle{ID: "feb", Period: "2026-02", Calendar: "us", Revision: 1, Stages: []StageSpec{{Stage: StageCollect, Rule: Rule{Days: 8, Business: false}, Basis: BasisFile}}}, human("planner")))
	if !v.Stages[0].OriginalDue.Equal(time.Date(2026, 3, 8, 7, 0, 0, 0, time.UTC)) {
		t.Fatalf("a close time in the spring gap: %v", v.Stages[0].OriginalDue)
	}
}

func TestStagesSharingADueDate(t *testing.T) {
	f := open(t, testLimits())
	f.calendar(t, "ws", seoul())
	must(f.r.OpenCycle("ws", Cycle{ID: "same", Period: "2026-10", Calendar: "kr", Revision: 1, Stages: []StageSpec{
		{Stage: StageCollect, Rule: Rule{Days: 1, Business: true}, Basis: BasisFile}, {Stage: StageValidate, Rule: Rule{Days: 1, Business: true}, Basis: BasisFile}}}, human("planner")))
	must(f.r.AddTargets("ws", "same", Addition{Targets: []string{"a"}, Reason: "r"}, human("planner")))
	f.clock.set(must(f.r.CycleOf("ws", "same")).Stages[0].CurrentDue)
	got := must(f.r.Claim("ws", 10, system("s")))
	if len(got) != 2 || got[0].Stage != StageCollect || got[1].Stage != StageValidate {
		t.Fatalf("stage order breaks ties: %+v", got)
	}
}

func TestEveryChangeTakesTheNextSequence(t *testing.T) {
	f := open(t, testLimits())
	v := f.cycle(t, "oct", "a")
	seq := must(f.r.CycleOf("ws", "oct")).Sequence
	next := func(label string) {
		t.Helper()
		got := must(f.r.CycleOf("ws", "oct")).Sequence
		if got != seq+1 {
			t.Fatalf("%s: sequence %d after %d", label, got, seq)
		}
		seq = got
	}
	must(f.r.Complete("ws", "oct", "a", StageCollect, evidence(EvidenceFile), system("feed")))
	next("complete")
	must(f.r.Reopen("ws", "oct", "a", StageCollect, "r", human("ops")))
	next("reopen")
	must(f.r.Hold("ws", "oct", "a", StageCollect, "r", human("ops")))
	next("hold")
	must(f.r.Release("ws", "oct", "a", StageCollect, human("ops")))
	next("release")
	must(f.r.RequestWaiver("ws", "oct", "a", StageCollect, Waiver{Reason: "r", Evidence: "e"}, human("ops")))
	next("request waiver")
	must(f.r.WithdrawWaiver("ws", "oct", "a", StageCollect, human("ops")))
	next("withdraw waiver")
	must(f.r.RequestWaiver("ws", "oct", "a", StageCollect, Waiver{Reason: "r", Evidence: "e"}, human("ops")))
	next("request waiver again")
	must(f.r.ApproveWaiver("ws", "oct", "a", StageCollect, human("lead")))
	next("approve waiver")
	must(f.r.RequestPostponement("ws", "oct", StageIssue, Postponement{Due: v.Stages[3].CurrentDue, Reason: "r"}, human("ops")))
	next("request postponement")
	must(f.r.WithdrawPostponement("ws", "oct", StageIssue, human("ops")))
	next("withdraw postponement")
	must(f.r.RequestPostponement("ws", "oct", StageIssue, Postponement{Due: v.Stages[3].CurrentDue, Reason: "r"}, human("ops")))
	next("request postponement again")
	must(f.r.ApprovePostponement("ws", "oct", StageIssue, human("lead")))
	next("approve postponement")
	must(f.r.AddTargets("ws", "oct", Addition{Targets: []string{"b"}, Reason: "r"}, human("planner")))
	next("add targets")
	expect(t, "the stage before the last", second(f.r.RequestPostponement("ws", "oct", StageIssue, Postponement{Due: v.Stages[3].CurrentDue.Add(time.Hour), Reason: "r"}, human("ops"))), RefusalStageOrder)
}

func TestExactHistoryRoom(t *testing.T) {
	for _, op := range []string{"hold", "waiver", "postpone"} {
		f := open(t, Limits{Workspaces: 1, Calendars: 2, Cycles: 2, Targets: 10, History: 6, Claims: 10, Lead: time.Hour})
		v := f.cycle(t, "oct", "a")
		switch op {
		case "hold":
			must(f.r.Hold("ws", "oct", "a", StageCollect, "r", human("ops")))
			must(f.r.Release("ws", "oct", "a", StageCollect, human("ops")))
		case "waiver":
			must(f.r.RequestWaiver("ws", "oct", "a", StageCollect, Waiver{Reason: "r", Evidence: "e"}, human("ops")))
			must(f.r.ApproveWaiver("ws", "oct", "a", StageCollect, human("lead")))
		case "postpone":
			must(f.r.RequestPostponement("ws", "oct", StageCollect, Postponement{Due: v.Stages[0].CurrentDue.Add(time.Hour), Reason: "r"}, human("ops")))
			must(f.r.ApprovePostponement("ws", "oct", StageCollect, human("lead")))
		}
	}
}

func TestAllSixStagesAndFormEdges(t *testing.T) {
	f := open(t, testLimits())
	f.calendar(t, "ws", seoul())
	var all []StageSpec
	for index, st := range []string{StageCollect, StageValidate, StageApprove, StageIssue, StageDeliver, StageSettle} {
		all = append(all, StageSpec{Stage: st, Rule: Rule{Days: index + 1, Business: true}, Basis: BasisFile})
	}
	all[5].Rule = Rule{Days: maxDays}
	v := must(f.r.OpenCycle("ws", Cycle{ID: "all", Period: "2026-10", Calendar: "kr", Revision: 1, Stages: all}, human("planner")))
	if len(v.Stages) != 6 {
		t.Fatal(v.Stages)
	}
	edges := seoul()
	edges.ID, edges.Close = "edges", 1
	edges.Holidays = make([]string, maxHolidays)
	day := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	for index := range edges.Holidays {
		edges.Holidays[index] = day.AddDate(0, 0, index).Format(time.DateOnly)
	}
	must(f.r.ProposeCalendar("ws", edges, human("planner")))
	long := seoul()
	long.ID, long.Zone = "long", strings.Repeat("Z", maxIDBytes)
	expect(t, "a 128-byte zone reaches the loader", second(f.r.ProposeCalendar("ws", long, human("planner"))), RefusalZoneUnavailable)
	g := open(t, testLimits())
	g.r.config.Zones = func(string) (*time.Location, error) { return nil, nil }
	expect(t, "nil zone", second(g.r.ProposeCalendar("ws", seoul(), human("planner"))), RefusalZoneUnavailable)
	h := open(t, testLimits())
	h.cycle(t, "oct", "a")
	expect(t, "decide id", second(h.r.ApprovePostponement("ws", "", StageCollect, human("lead"))), "invalid")
	expect(t, "decide stage", second(h.r.ApprovePostponement("ws", "oct", "x", human("lead"))), "invalid")
	expect(t, "decide actor", second(h.r.ApprovePostponement("ws", "oct", StageCollect, Actor{ID: "x"})), "invalid")
}

func TestLatenessAtEarlierSequences(t *testing.T) {
	f := open(t, testLimits())
	v := f.cycle(t, "oct", "a")
	f.clock.set(v.Stages[0].OriginalDue.Add(time.Nanosecond))
	must(f.r.Complete("ws", "oct", "a", StageCollect, evidence(EvidenceFile), system("feed")))
	if p := must(f.r.Progress("ws", "oct", StageCollect, AsOf{})); p.DoneLate != 1 {
		t.Fatal("one nanosecond late is late")
	}
	must(f.r.Reopen("ws", "oct", "a", StageCollect, "r", human("ops")))
	g := open(t, testLimits())
	w := g.cycle(t, "oct", "a")
	must(g.r.Complete("ws", "oct", "a", StageCollect, evidence(EvidenceFile), system("feed")))
	onTime := must(g.r.CycleOf("ws", "oct")).Sequence
	must(g.r.Reopen("ws", "oct", "a", StageCollect, "r", human("ops")))
	g.clock.set(w.Stages[0].OriginalDue.Add(time.Hour))
	must(g.r.Complete("ws", "oct", "a", StageCollect, evidence(EvidenceFile), system("feed")))
	if p := must(g.r.Progress("ws", "oct", StageCollect, AsOf{Sequence: onTime})); p.DoneLate != 0 || p.Done != 1 {
		t.Fatalf("lateness as of the earlier sequence: %+v", p)
	}
	if p := must(g.r.Progress("ws", "oct", StageCollect, AsOf{})); p.DoneLate != 1 {
		t.Fatal("the latest completion was late")
	}
}
