package server

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/recovery"
)

func TestRecoveryMemberRowsAreWindowed(t *testing.T) {
	v := recovery.RecoveryView{Phase: recovery.PhaseVerify}
	history := []recovery.History{}
	for n := 0; n < 120; n++ {
		workspace := SampleWorkspace
		if n%10 == 9 {
			workspace = "other-workspace"
		}
		v.OpenCalls = append(v.OpenCalls, recovery.CallView{ID: fmt.Sprint("call-", n), Target: recovery.Target{Workspace: workspace, Key: fmt.Sprint("k-", n)}})
		v.Pending = append(v.Pending, recovery.EventView{Sequence: uint64(n + 1)})
		v.Unknown = append(v.Unknown, recovery.Target{Workspace: workspace, Key: fmt.Sprint("k-", n)})
		history = append(history, recovery.History{Recovery: fmt.Sprint("recovery-", n)})
	}
	f := setup(t, SeedSample).ready()
	facts := recoveryFacts{install: f.s.rec.install(recoveryPrimary), state: recovery.StateRecovering, view: &v, history: history}
	set, iso, resume := setRow(facts), f.s.isolationRow(facts, SampleWorkspace), f.s.resumeRow(facts)
	rec, err := f.s.reconcileRow(facts, SampleWorkspace)
	if err != nil || set.HistoryNext != "71" || iso.OpenNext != "50" || iso.OpenTotal != 108 || rec.UnknownNext != "50" || rec.UnknownTotal != 108 || resume.PendingNext != "50" || !iso.OtherScope || !rec.OtherScope {
		t.Fatalf("the main rows say where they stopped: history %q open %q of %d unknown %q of %d pending %q %v", set.HistoryNext, iso.OpenNext, iso.OpenTotal, rec.UnknownNext, rec.UnknownTotal, resume.PendingNext, err)
	}
	seen, before := []string{}, 0
	for _, h := range set.History {
		seen = append(seen, h.Recovery)
	}
	for next := set.HistoryNext; next != ""; {
		fmt.Sscan(next, &before)
		var rows []recoveryHistoryRow
		rows, next = historyRows(facts, before, 30)
		for _, h := range rows {
			seen = append(seen, h.Recovery)
		}
	}
	if len(seen) != 120 || seen[0] != "recovery-119" || seen[50] != "recovery-69" || seen[119] != "recovery-0" {
		t.Fatalf("the history is newest first without gaps: %d rows, %s %s %s", len(seen), seen[0], seen[50], seen[len(seen)-1])
	}
	facts.history = append(facts.history, recovery.History{Recovery: "recovery-new"})
	if rows, _ := historyRows(facts, 71, 1); len(rows) != 1 || rows[0].Recovery != "recovery-69" {
		t.Errorf("a new history entry moves the rows after the cursor: %v", rows)
	}
	facts.history = facts.history[1:]
	if rows, _ := historyRows(facts, 71, 2); len(rows) != 2 || rows[0].Recovery != "recovery-70" || rows[1].Recovery != "recovery-69" {
		t.Errorf("when the oldest entry is dropped a row is repeated, none is skipped: %v", rows)
	}
	calls, passed := []string{}, 0
	for {
		rows, total := openCallRows(facts, SampleWorkspace, passed, 40)
		for _, c := range rows {
			calls = append(calls, c.ID)
		}
		next := nextOf(len(rows), total, passed)
		if next == "" {
			break
		}
		fmt.Sscan(next, &passed)
	}
	if len(calls) != 108 || calls[0] != "call-0" || calls[9] != "call-10" || calls[107] != "call-118" {
		t.Fatalf("the open calls of the workspace in the module's order: %d, %s %s %s", len(calls), calls[0], calls[9], calls[len(calls)-1])
	}
	unknown, total, err := f.s.unknownRows(facts, SampleWorkspace, 100, 50)
	if err != nil || total != 108 || len(unknown) != 8 || unknown[0].Key != "k-111" || nextOf(len(unknown), total, 100) != "" {
		t.Errorf("the last unknown targets: %d of %d first %v %v", len(unknown), total, unknown, err)
	}
	if past, total, _ := f.s.unknownRows(facts, SampleWorkspace, 500, 50); len(past) != 0 || total != 108 {
		t.Errorf("past the end: %d of %d", len(past), total)
	}
	pending := pendingRows(facts, 100, 50)
	if len(pending) != 20 || pending[0].Sequence != 101 || nextOf(len(pending), 120, 100) != "" || nextOf(50, 120, 50) != "100" {
		t.Errorf("the last pending events: %d first %v", len(pending), pending)
	}
}

func TestRecoveryRowsRead(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	got := f.recoveryRead(root)
	iso, rec, resume, set := sectionRow(t, got, "isolation", recoveryPrimary), sectionRow(t, got, "reconciliation", recoveryPrimary), sectionRow(t, got, "resume", recoveryPrimary), sectionRow(t, got, "sets", recoveryPrimary)
	if s := fmt.Sprintf("%v|%v|%v|%v", iso["open_calls_next"], rec["unknown_next"], resume["pending_next"], set["history_next"]); s != "|||" {
		t.Fatalf("lists listed whole have no cursor: %q", s)
	}
	path := "/api/v1/recovery/rows?installation=" + recoveryPrimary
	for list, want := range map[string]any{"open_calls": iso["open_calls"], "unknown": rec["unknown_targets"], "pending": resume["pending_events"], "history": set["history"]} {
		page := f.readAs(root, path+"&list="+list)
		order, moves := "recorded", true
		if list == "history" {
			order, moves = "newest_first", false
		}
		if page.status != 200 || !reflect.DeepEqual(page.body["rows"], want) || page.body["next"] != "" || whole(page.body["total"]) != len(want.([]any)) || page.body["list"] != list ||
			page.body["installation"] != recoveryPrimary || page.body["order"] != order || page.body["rows_can_move"] != moves {
			t.Errorf("%s: %s", list, page.raw[:min(len(page.raw), 300)])
		}
	}
	if past := f.readAs(root, path+"&list=open_calls&after=5"); past.status != 200 || len(past.body["rows"].([]any)) != 0 || past.body["next"] != "" || whole(past.body["total"]) != 1 {
		t.Errorf("past the end: %s", past.raw[:min(len(past.raw), 200)])
	}
	if idle := f.readAs(root, "/api/v1/recovery/rows?installation="+recoveryStandby+"&list=pending"); idle.status != 200 || len(idle.body["rows"].([]any)) != 0 || whole(idle.body["total"]) != 0 {
		t.Errorf("an installation that is not recovering: %s", idle.raw[:min(len(idle.raw), 200)])
	}
	base := "/api/v1/recovery/rows"
	for _, query := range []string{"", "?installation=" + recoveryPrimary, "?list=history", "?installation=never&list=history", "?installation=" + recoveryPrimary + "&list=locks",
		"?installation=" + recoveryPrimary + "&list=history&after=0", "?installation=" + recoveryPrimary + "&list=history&after=x", "?installation=" + recoveryPrimary + "&list=history&limit=0",
		"?installation=" + recoveryPrimary + "&list=history&limit=51", "?installation=" + recoveryPrimary + "&list=history&q=a", "?installation=" + recoveryPrimary + "&list=history&list=pending",
		"?installation=" + recoveryPrimary + "&installation=" + recoveryPrimary + "&list=history", "?installation=&list=history"} {
		if refused := f.readAs(root, base+query); refused.status != 422 {
			t.Errorf("%q: %d", query, refused.status)
		}
	}
	ok := path + "&list=history"
	f.code(t, "member", f.readAs(f.as(MemberLogin), ok), 403, "permission_denied")
	f.code(t, "anonymous", f.readAs(actor{}, ok), 401, "unauthenticated")
	f.s.fault = func(at string) bool { return at == "recovery-read" }
	if failed := f.readAs(root, ok); failed.status != 503 {
		t.Errorf("a fault: %d", failed.status)
	}
	f.s.fault = nil
	f.s.bootWorkspace = "elsewhere"
	if out := f.readAs(root, ok); out.status != 422 {
		t.Errorf("another workspace: %d", out.status)
	}
}
