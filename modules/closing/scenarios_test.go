package closing

import (
	"testing"
	"time"
)

func seoulAt(y int, m time.Month, d, h, mi int) time.Time {
	location, _ := time.LoadLocation("Asia/Seoul")
	return time.Date(y, m, d, h, mi, 0, 0, location).UTC()
}

func TestDeadlinesFollowTheCalendar(t *testing.T) {
	f := open(t, testLimits())
	v := f.cycle(t, "oct")
	want := []time.Time{seoulAt(2026, 11, 2, 18, 0), seoulAt(2026, 11, 4, 18, 0), seoulAt(2026, 11, 5, 18, 0), seoulAt(2026, 11, 10, 18, 0)}
	for index, st := range v.Stages {
		if !st.OriginalDue.Equal(want[index]) || !st.CurrentDue.Equal(want[index]) {
			t.Errorf("%s: %v want %v", st.Spec.Stage, st.OriginalDue, want[index])
		}
	}
	if v.Sequence != 1 {
		t.Fatal("opening takes sequence 1")
	}
}

func TestDeadlineEdges(t *testing.T) {
	ny, _ := time.LoadLocation("America/New_York")
	day := time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC)
	if got := localInstant(day, 2*60+30, ny); !got.Equal(time.Date(2026, 3, 8, 7, 0, 0, 0, time.UTC)) {
		t.Fatalf("gap moves to the first valid instant: %v", got)
	}
	overlap := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	if got := localInstant(overlap, 60+30, ny); !got.Equal(time.Date(2026, 11, 1, 5, 30, 0, 0, time.UTC)) {
		t.Fatalf("overlap takes the earlier instant: %v", got)
	}
	if got := localInstant(day, endOfDay, time.UTC); !got.Equal(time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("1440 is the end of the date: %v", got)
	}
	cal := &calendar{view: CalendarView{Calendar: Calendar{Close: endOfDay}}, location: time.UTC, weekend: map[time.Weekday]bool{}, holidays: map[string]bool{}}
	cal.from, _ = parseDate("2028-01-01")
	cal.to, _ = parseDate("2028-03-31")
	if got := must(cal.deadline("2028-02", Rule{Days: 1})); !got.Equal(time.Date(2028, 3, 2, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("leap February: %v", got)
	}
	expect(t, "beyond coverage", second(cal.deadline("2028-03", Rule{Days: 1})), RefusalOutsideCalendar)
	expect(t, "before coverage", second(cal.deadline("2027-11", Rule{Days: 1})), RefusalOutsideCalendar)
	if got := must(cal.deadline("2028-02", Rule{Days: 31})); !got.Equal(time.Date(2028, 4, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("the last covered day: %v", got)
	}
	cal.to, _ = parseDate("2028-03-15")
	expect(t, "runs out", second(cal.deadline("2028-02", Rule{Days: 31})), RefusalOutsideCalendar)
	dense := &calendar{view: CalendarView{Calendar: Calendar{Close: 60}}, location: time.UTC, weekend: map[time.Weekday]bool{}, holidays: map[string]bool{}}
	dense.from, _ = parseDate("2026-01-01")
	dense.to, _ = parseDate("2030-12-31")
	for d := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC); d.Year() < 2028; d = d.AddDate(0, 0, 1) {
		dense.holidays[d.Format(time.DateOnly)] = true
	}
	expect(t, "bounded search", second(dense.deadline("2026-01", Rule{Days: 1, Business: true})), RefusalOutsideCalendar)
}

func TestWaiversAndHoldsAreNotDone(t *testing.T) {
	f := open(t, testLimits())
	f.cycle(t, "oct", "a", "b", "c", "d")
	must(f.r.Complete("ws", "oct", "a", StageCollect, evidence(EvidenceFile), system("feed")))
	must(f.r.Hold("ws", "oct", "b", StageCollect, "dispute", human("ops")))
	must(f.r.RequestWaiver("ws", "oct", "c", StageCollect, Waiver{Reason: "no-usage", Evidence: "memo-1"}, human("ops")))
	expect(t, "waiver pending blocks completion", second(f.r.Complete("ws", "oct", "c", StageCollect, evidence(EvidenceFile), system("feed"))), RefusalWaiverPending)
	expect(t, "self approval", second(f.r.ApproveWaiver("ws", "oct", "c", StageCollect, human("ops"))), RefusalSelfApproval)
	must(f.r.ApproveWaiver("ws", "oct", "c", StageCollect, human("lead")))
	p := must(f.r.Progress("ws", "oct", StageCollect, AsOf{}))
	if p.Targets != 4 || p.Denominator != 3 || p.Done != 1 || p.Waived != 1 || p.Held != 1 || p.Open != 1 {
		t.Fatalf("progress: %+v", p)
	}
	before := p.Sequence
	must(f.r.AddTargets("ws", "oct", Addition{Targets: []string{"e"}, Reason: "late-contract"}, human("planner")))
	f.clock.set(seoulAt(2026, 11, 2, 19, 0))
	now := must(f.r.Progress("ws", "oct", StageCollect, AsOf{}))
	old := must(f.r.Progress("ws", "oct", StageCollect, AsOf{Sequence: before}))
	if now.Targets != 5 || now.Denominator != 4 || old.Targets != 4 || old.Denominator != 3 || now.OpenPastDue != 2 || now.HeldPastDue != 1 {
		t.Fatalf("revisions: now %+v old %+v", now, old)
	}
	if first := must(f.r.Progress("ws", "oct", StageCollect, AsOf{Sequence: 1})); first.Targets != 0 {
		t.Fatal("the cycle right after opening")
	}
	expect(t, "beyond", second(f.r.Progress("ws", "oct", StageCollect, AsOf{Sequence: now.Sequence + 1})), "not_found")
}

func TestPostponementsKeepTheOriginalDueAndLateness(t *testing.T) {
	f := open(t, testLimits())
	v := f.cycle(t, "oct", "a")
	original := v.Stages[0].OriginalDue
	newDue := v.Stages[1].CurrentDue
	must(f.r.RequestPostponement("ws", "oct", StageCollect, Postponement{Due: newDue, Reason: "erp-outage"}, human("ops")))
	expect(t, "self", second(f.r.ApprovePostponement("ws", "oct", StageCollect, human("ops"))), RefusalSelfApproval)
	before := must(f.r.CycleOf("ws", "oct")).Sequence
	moved := must(f.r.ApprovePostponement("ws", "oct", StageCollect, human("lead")))
	if !moved.Stages[0].OriginalDue.Equal(original) || !moved.Stages[0].CurrentDue.Equal(newDue) || len(moved.Stages[0].History) != 2 {
		t.Fatalf("postponed: %+v", moved.Stages[0])
	}
	f.clock.set(original.Add(time.Hour))
	if p := must(f.r.Progress("ws", "oct", StageCollect, AsOf{})); p.OpenPastDue != 0 || !p.CurrentDue.Equal(newDue) || !p.OriginalDue.Equal(original) {
		t.Fatalf("overdue follows the current due: %+v", p)
	}
	if p := must(f.r.Progress("ws", "oct", StageCollect, AsOf{Sequence: before})); !p.CurrentDue.Equal(original) || p.OpenPastDue != 1 {
		t.Fatalf("the current due as of an earlier sequence: %+v", p)
	}
	done := must(f.r.Complete("ws", "oct", "a", StageCollect, evidence(EvidenceFile), system("feed")))
	last := done.History[len(done.History)-1]
	if last.LateOriginal != time.Hour || last.LateCurrent != 0 {
		t.Fatalf("lateness: %+v", last)
	}
	if p := must(f.r.Progress("ws", "oct", StageCollect, AsOf{})); p.DoneLate != 1 {
		t.Fatal("late against the original due")
	}
	expect(t, "beyond the next stage", second(f.r.RequestPostponement("ws", "oct", StageCollect, Postponement{Due: v.Stages[1].CurrentDue.Add(time.Hour), Reason: "x"}, human("ops"))), RefusalStageOrder)
}

func TestReopenKeepsHistoryAndCascades(t *testing.T) {
	f := open(t, testLimits())
	f.cycle(t, "oct", "a")
	expect(t, "stage order", second(f.r.Complete("ws", "oct", "a", StageValidate, evidence(EvidenceInternal), human("ana"))), RefusalStageOrder)
	must(f.r.Complete("ws", "oct", "a", StageCollect, evidence(EvidenceFile), system("feed")))
	expect(t, "basis", second(f.r.Complete("ws", "oct", "a", StageValidate, evidence(EvidenceERP), human("ana"))), RefusalBasisMismatch)
	expect(t, "internal needs a human", second(f.r.Complete("ws", "oct", "a", StageValidate, evidence(EvidenceInternal), system("feed"))), RefusalNotHuman)
	must(f.r.Complete("ws", "oct", "a", StageValidate, evidence(EvidenceInternal), human("ana")))
	must(f.r.RequestWaiver("ws", "oct", "a", StageIssue, Waiver{Reason: "manual", Evidence: "m"}, human("ops")))
	must(f.r.ApproveWaiver("ws", "oct", "a", StageIssue, human("lead")))
	must(f.r.Complete("ws", "oct", "a", StageSettle, evidence(EvidenceInternal), human("ana")))
	must(f.r.Reopen("ws", "oct", "a", StageCollect, "wrong-file", human("ops")))
	for _, st := range []string{StageCollect, StageValidate, StageSettle} {
		it := must(f.r.Item("ws", "oct", "a", st, time.Time{}))
		if it.State != ItemOpen || it.History[len(it.History)-1].Kind != EventReopened {
			t.Fatalf("%s: %+v", st, it)
		}
	}
	if it := must(f.r.Item("ws", "oct", "a", StageIssue, time.Time{})); it.State != ItemWaived {
		t.Fatal("later waived stages stay waived")
	}
	again := must(f.r.Complete("ws", "oct", "a", StageCollect, evidence(EvidenceFile), system("feed")))
	completions := 0
	for _, e := range again.History {
		if e.Kind == EventCompleted {
			completions++
		}
	}
	if completions != 2 {
		t.Fatal("every completion stays in the history")
	}
}

func TestClaimsAreSignalsHandedOutOnce(t *testing.T) {
	f := open(t, testLimits())
	v := f.cycle(t, "oct", "b", "a")
	due := v.Stages[0].CurrentDue
	f.clock.set(due.Add(-25 * time.Hour))
	if got := must(f.r.Claim("ws", 10, system("scheduler"))); len(got) != 0 {
		t.Fatal(got)
	}
	f.clock.set(due.Add(-time.Hour))
	got := must(f.r.Claim("ws", 10, system("scheduler")))
	if len(got) != 1 || got[0].Step != StepDueSoon || len(got[0].Targets) != 2 || got[0].Targets[0] != "a" || got[0].Stage != StageCollect {
		t.Fatalf("due soon: %+v", got)
	}
	if again := must(f.r.Claim("ws", 10, system("scheduler"))); len(again) != 0 {
		t.Fatal("once only")
	}
	must(f.r.Complete("ws", "oct", "a", StageCollect, evidence(EvidenceFile), system("feed")))
	f.clock.set(due.Add(25 * time.Hour))
	got = must(f.r.Claim("ws", 10, system("scheduler")))
	var collect *Event
	for index := range got {
		if got[index].Stage == StageCollect {
			collect = &got[index]
		}
	}
	if collect == nil || collect.Step != StepOverdue || len(collect.Targets) != 1 || collect.Targets[0] != "b" || len(collect.Skipped) != 1 {
		t.Fatalf("missed run hands out the latest step and records the skip: %+v", got)
	}
	if item := must(f.r.Item("ws", "oct", "b", StageCollect, time.Time{})); item.State != ItemOpen {
		t.Fatal("claims change nothing")
	}
	f.clock.set(due.Add(26 * time.Hour))
	must(f.r.RequestPostponement("ws", "oct", StageCollect, Postponement{Due: v.Stages[1].CurrentDue, Reason: "x"}, human("ops")))
	must(f.r.ApprovePostponement("ws", "oct", StageCollect, human("lead")))
	limited := must(f.r.Claim("ws", 1, system("scheduler")))
	if len(limited) != 1 || len(limited[0].Targets) != 1 {
		t.Fatalf("limit counts items: %+v", limited)
	}
}
