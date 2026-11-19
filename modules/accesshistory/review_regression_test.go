package accesshistory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestClockDeferralBlocksSizeDeletionAtRestart(t *testing.T) {
	r := newRig(t)
	h := r.open()
	h.Offer(signIn("u1", "ws1", hA))
	h.Flush()
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(r.dir, DirName, r.clock.now().Format(dayLayout)+daySuffix)
	before, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	r.opts.MaxBytes = h.total
	r.clock.add(8 * 24 * time.Hour)
	if next, err := open(r.opts, r.sys); err == nil {
		next.Close()
	}
	after, err := os.ReadFile(name)
	if err != nil || string(after) != string(before) {
		t.Fatalf("unaccepted clock jump deleted or rewrote old data: %v", err)
	}
}

func TestObservedHoldIsDurableBeforeRewrite(t *testing.T) {
	r := newRig(t)
	r.opts.Days = 1
	fs := &faultSys{}
	r.sys = fs
	h := r.open()
	defer h.Close()
	h.Offer(signIn("u1", "ws1", hA))
	h.Offer(signIn("u2", "ws2", hB))
	h.Flush()
	checked, durable := false, false
	fs.mu.Lock()
	fs.renameErr = func(to string) bool {
		if _, ok := dayOf(filepath.Base(to)); ok {
			checked = true
			raw, err := os.ReadFile(filepath.Join(r.dir, DirName, retentionFile))
			var state stateFile
			durable = err == nil && json.Unmarshal(raw, &state) == nil && len(state.Markers) > 0
		}
		return false
	}
	fs.mu.Unlock()
	r.mu.Lock()
	r.holds = []Hold{{ID: "hold-1", Owner: "operator", Reason: "sample", Tenants: []string{"ws1"}, Classes: []string{ClassIdentity}, Covers: func(ws string) bool { return ws == "ws1" }}}
	r.mu.Unlock()
	r.clock.add(48 * time.Hour)
	h.Tick()
	if !checked || !durable {
		t.Fatalf("day rewrite before durable hold marker: checked=%v durable=%v", checked, durable)
	}
}

func TestFailedHoldSnapshotPreventsDestructiveCleanup(t *testing.T) {
	r := newRig(t)
	r.opts.Days = 1
	fs := &faultSys{}
	r.sys = fs
	h := r.open()
	defer h.Close()
	h.Offer(signIn("u1", "ws1", hA))
	h.Offer(signIn("u2", "ws2", hB))
	h.Flush()
	name := filepath.Join(r.dir, DirName, r.clock.now().Format(dayLayout)+daySuffix)
	before, _ := os.ReadFile(name)
	r.mu.Lock()
	r.holds = []Hold{{ID: "hold-1", Tenants: []string{"ws1"}, Classes: []string{ClassIdentity}, Covers: func(ws string) bool { return ws == "ws1" }}}
	r.mu.Unlock()
	fs.mu.Lock()
	fs.renameErr = func(to string) bool { return strings.HasSuffix(to, retentionFile) }
	fs.mu.Unlock()
	r.clock.add(48 * time.Hour)
	h.Tick()
	after, err := os.ReadFile(name)
	if err != nil || string(after) != string(before) {
		t.Fatalf("hold snapshot failure changed original day: %v", err)
	}
}

func TestLossCountsRespectWorkspaceAndIncarnation(t *testing.T) {
	r := newRig(t)
	fs := &faultSys{}
	r.sys = fs
	h := r.open()
	h.Offer(signIn("u1", "ws1", hA))
	h.Flush()
	fs.mu.Lock()
	fs.writeErr = func(p string) bool { _, ok := dayOf(filepath.Base(p)); return ok }
	fs.mu.Unlock()
	h.Offer(signIn("u1", "ws2", hB))
	h.Flush()
	if n, ok := noteOf(h.Read("u1", Query{Workspace: "ws1"}), "dropped"); ok && n.Count != nil {
		t.Fatalf("other workspace exact losses exposed: %+v", n)
	}
	if n, ok := noteOf(h.Read("u1", Query{Workspace: "ws2"}), "dropped"); !ok || n.Count == nil || *n.Count != 1 {
		t.Fatalf("own workspace loss not counted: %+v", n)
	}
	if err := h.Incarnation("u1"); err != nil {
		t.Fatal(err)
	}
	if n, ok := noteOf(h.Read("u1", Query{Workspace: "ws2"}), "dropped"); ok && n.Count != nil {
		t.Fatalf("predecessor exact losses exposed: %+v", n)
	}
	fs.mu.Lock()
	fs.writeErr = nil
	fs.mu.Unlock()
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
	h = r.open()
	defer h.Close()
	if n, ok := noteOf(h.Read("u1", Query{Workspace: "ws2"}), "dropped"); ok && n.Count != nil {
		t.Fatalf("predecessor losses exposed after restart: %+v", n)
	}
}

func TestConcurrentHistorySnapshotsAndMaintenance(t *testing.T) {
	r := newRig(t)
	h := r.open()
	defer h.Close()
	var wg sync.WaitGroup
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 60; i++ {
				switch g {
				case 0:
					h.Offer(signIn("u1", "ws1", hA))
				case 1:
					h.Read("u1", Query{Workspace: "ws1"})
				case 2:
					if err := h.Incarnation("u1"); err != nil {
						t.Error(err)
					}
				case 3:
					r.clock.add(time.Hour)
					h.Tick()
				}
			}
		}(g)
	}
	wg.Wait()
	h.Flush()
}

func TestPersistedLossCountsKeepTheirOriginalScope(t *testing.T) {
	r := newRig(t)
	h := r.open()
	source := Record{User: "u1", Workspace: "ws2", Seq: h.nextSeq()}
	counts := lossCounts{}
	counts.add(source, 3)
	h.control(func() {
		h.writeSystem(Record{Kind: sysDropped, Count: 3, From: millis(h.now()), To: millis(h.now()), Per: counts.encoded()})
	})
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
	h = r.open()
	for _, ws := range []string{"ws1", "ws2"} {
		note, ok := noteOf(h.Read("u1", Query{Workspace: ws}), "dropped")
		if !ok || (ws == "ws1" && note.Count != nil) || (ws == "ws2" && (note.Count == nil || *note.Count != 3)) {
			t.Fatalf("scope %s after restart: %+v", ws, note)
		}
	}
	if err := h.Incarnation("u1"); err != nil {
		t.Fatal(err)
	}
	h.Tick()
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
	h = r.open()
	defer h.Close()
	if note, ok := noteOf(h.Read("u1", Query{Workspace: "ws2"}), "dropped"); ok && note.Count != nil {
		t.Fatalf("persisted predecessor count exposed: %+v", note)
	}
}

func TestObservedHoldSurvivesCrashAfterDayRewrite(t *testing.T) {
	r := newRig(t)
	r.opts.Days = 1
	fs := &faultSys{}
	r.sys = fs
	h := r.open()
	h.Offer(signIn("u1", "ws1", hA))
	h.Offer(signIn("u2", "ws2", hB))
	h.Flush()
	r.mu.Lock()
	r.holds = []Hold{{ID: "h1", Tenants: []string{"ws1"}, Classes: []string{ClassIdentity}, Covers: func(ws string) bool { return ws == "ws1" }}}
	r.mu.Unlock()
	r.clock.add(48 * time.Hour)
	rewritten := false
	fs.mu.Lock()
	fs.renameErr = func(to string) bool {
		if _, ok := dayOf(filepath.Base(to)); ok {
			rewritten = true
		}
		if rewritten && strings.HasSuffix(to, retentionFile) {
			panic("crash after day rewrite")
		}
		return false
	}
	fs.mu.Unlock()
	h.control(func() { defer func() { _ = recover() }(); h.purgeLocked(h.now()) })
	close(h.quit)
	h.wg.Wait()
	h.release()
	fs.mu.Lock()
	fs.renameErr = nil
	fs.mu.Unlock()
	r.mu.Lock()
	r.holds = nil
	r.mu.Unlock()
	h = r.open()
	defer h.Close()
	if len(h.state.Markers) != 1 {
		t.Fatalf("observed hold lost after crash: %+v", h.state.Markers)
	}
	raw, err := os.ReadFile(filepath.Join(r.dir, DirName, "2026-03-10"+daySuffix))
	if err != nil || !strings.Contains(string(raw), `"u":"u1"`) || strings.Contains(string(raw), `"u":"u2"`) {
		t.Fatalf("held record not protected across restart: %v %s", err, raw)
	}
}

func TestContinuousReadOfferAndStateSnapshots(t *testing.T) {
	r := newRig(t)
	r.opts.Days = 2
	h := r.open()
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				h.Read("u1", Query{Workspace: "ws1"})
			}
		}
	}()
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				h.Offer(signIn("u1", "ws1", hA))
			}
		}
	}()
	for i := 0; i < 15; i++ {
		r.clock.add(24 * time.Hour)
		if err := h.Incarnation("u1"); err != nil {
			t.Fatal(err)
		}
		h.Tick()
	}
	close(done)
	wg.Wait()
	h.Flush()
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestQueuedPredecessorRefusalsDoNotBecomeCurrentSummary(t *testing.T) {
	r := newRig(t)
	h := r.open()
	defer h.Close()
	entered, release := make(chan struct{}), make(chan struct{})
	go h.control(func() { close(entered); <-release })
	<-entered
	for i := 0; i < refusedCapPerDay+5; i++ {
		h.Offer(refused("u1"))
	}
	if err := h.Incarnation("u1"); err != nil {
		t.Fatal(err)
	}
	close(release)
	h.Flush()
	r.clock.add(24 * time.Hour)
	h.Tick()
	if page := h.Read("u1", Query{Workspace: "ws1"}); len(page.Events) != 0 {
		t.Fatalf("predecessor refusal summary exposed: %+v", page.Events)
	}
	for i := 0; i < refusedCapPerDay+5; i++ {
		h.Offer(refused("u1"))
	}
	h.Flush()
	r.clock.add(24 * time.Hour)
	h.Tick()
	check := func() {
		page := h.Read("u1", Query{Workspace: "ws1", Limit: MaxPage})
		summaries := 0
		for _, event := range page.Events {
			if event.Kind == KindSignInRefusedMore {
				summaries++
				if event.Count != 5 {
					t.Fatalf("new incarnation summary count %d", event.Count)
				}
			}
		}
		if summaries != 1 {
			t.Fatalf("own summary missing: %+v", page.Events)
		}
	}
	check()
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
	h = r.open()
	defer h.Close()
	check()
}

func TestLateRefusalSummaryDoesNotCrossIncarnationAsEventOrLoss(t *testing.T) {
	r := newRig(t)
	h := r.open()
	old := h.nextSeq()
	if err := h.Incarnation("u1"); err != nil {
		t.Fatal(err)
	}
	h.control(func() {
		h.writeSystem(Record{Kind: KindSignInRefusedMore, User: "u1", Count: 5, SourceSeq: old})
		h.offerMu.Lock()
		h.countDrop(Record{Kind: KindSignInRefusedMore, User: "u1", Seq: h.seq + 1, SourceSeq: old, At: millis(h.now())})
		h.seq++
		h.offerMu.Unlock()
	})
	page := h.Read("u1", Query{Workspace: "ws1"})
	if len(page.Events) != 0 {
		t.Fatalf("late predecessor summary event visible: %+v", page.Events)
	}
	if note, ok := noteOf(page, "dropped"); ok && note.Count != nil {
		t.Fatalf("late predecessor loss count visible: %+v", note)
	}
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
	h = r.open()
	defer h.Close()
	page = h.Read("u1", Query{Workspace: "ws1"})
	if len(page.Events) != 0 {
		t.Fatalf("late predecessor summary visible after restart: %+v", page.Events)
	}
	if note, ok := noteOf(page, "dropped"); ok && note.Count != nil {
		t.Fatalf("late predecessor loss visible after restart: %+v", note)
	}
}

func TestPredecessorQueueDoesNotShiftSuccessorCursor(t *testing.T) {
	r := newRig(t)
	h := r.open()
	for i := 0; i < 5; i++ {
		h.Offer(signIn("u1", "ws1", hA))
	}
	h.Flush()
	entered, release := make(chan struct{}), make(chan struct{})
	go h.control(func() { close(entered); <-release })
	<-entered
	for i := 0; i < 5; i++ {
		h.Offer(signIn("u1", "ws1", hA))
	}
	if err := h.Incarnation("u1"); err != nil {
		t.Fatal(err)
	}
	close(release)
	h.Flush()
	for i := 0; i < 2; i++ {
		h.Offer(signIn("u1", "ws1", hB))
	}
	h.Flush()
	check := func(want uint64) {
		p := h.Read("u1", Query{Workspace: "ws1", Limit: 1})
		if !p.More || p.NextW != want {
			t.Fatalf("cursor discloses predecessor sequence: %+v", p)
		}
	}
	check(2)
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
	h = r.open()
	defer h.Close()
	check(2)
	h.Offer(signIn("u1", "ws1", hB))
	h.Flush()
	check(3)
}
