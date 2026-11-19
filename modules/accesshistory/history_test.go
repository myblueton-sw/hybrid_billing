package accesshistory

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) add(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

type rig struct {
	t     *testing.T
	dir   string
	clock *fakeClock
	holds []Hold
	logs  []string
	opts  Options
	sys   system
	mu    sync.Mutex
}

func newRig(t *testing.T) *rig {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	r := &rig{t: t, dir: dir, clock: &fakeClock{t: time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)}, sys: platform{}}
	r.opts = Options{Dir: dir, Days: 30, DaysSource: "default", MaxBytes: 64 << 20, ClockGap: 7 * 24 * time.Hour, Clock: r.clock.now, Manual: true,
		Holds: func() []Hold { r.mu.Lock(); defer r.mu.Unlock(); return append([]Hold{}, r.holds...) },
		Log:   func(event string, args ...any) { r.mu.Lock(); r.logs = append(r.logs, event); r.mu.Unlock() }}
	return r
}

func (r *rig) open() *History {
	r.t.Helper()
	h, err := open(r.opts, r.sys)
	if err != nil {
		r.t.Fatalf("open: %v", err)
	}
	return h
}

func (r *rig) openErr() error {
	h, err := open(r.opts, r.sys)
	if err == nil {
		h.Close()
	}
	return err
}

func (r *rig) logged(event string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, l := range r.logs {
		if l == event {
			return true
		}
	}
	return false
}

func signIn(user, ws, handle string) Record {
	return Record{Kind: KindSignedIn, User: user, Workspace: ws, Handle: handle, Browser: "chrome", OS: "macos"}
}

func refused(user string) Record {
	return Record{Kind: KindSignInRefused, User: user, Reason: ReasonWrongPass, Browser: "firefox", OS: "linux"}
}

const (
	hA = "0123456789abcdef"
	hB = "fedcba9876543210"
)

func kinds(p Page) string {
	out := []string{}
	for _, e := range p.Events {
		out = append(out, e.Kind)
	}
	return strings.Join(out, ",")
}

func TestOfferReadRestart(t *testing.T) {
	r := newRig(t)
	h := r.open()
	h.Offer(signIn("u1", "ws1", hA))
	h.Offer(refused("u1"))
	h.Offer(signIn("u2", "ws1", hB))
	h.Offer(Record{Kind: KindSessionEnded, User: "u1", Workspace: "ws2", Handle: hB, Reason: CauseByYou})
	h.Flush()
	p := h.Read("u1", Query{Workspace: "ws1"})
	if kinds(p) != "sign_in_refused,signed_in" {
		t.Fatalf("u1 ws1 page = %s", kinds(p))
	}
	if p := h.Read("u1", Query{Workspace: "ws2"}); kinds(p) != "session_ended,sign_in_refused" {
		t.Fatalf("u1 ws2 page = %s", kinds(p))
	}
	if p := h.Read("u3", Query{Workspace: "ws1"}); len(p.Events) != 0 || !p.AvailableFrom.IsZero() {
		t.Fatalf("u3 sees %v", p.Events)
	}
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
	r.clock.add(time.Hour)
	h = r.open()
	defer h.Close()
	if p := h.Read("u1", Query{Workspace: "ws1"}); kinds(p) != "sign_in_refused,signed_in" || len(p.Notes) != 0 {
		t.Fatalf("after restart = %s notes %v", kinds(p), p.Notes)
	}
	h.Offer(signIn("u1", "ws1", hA))
	h.Flush()
	if p := h.Read("u1", Query{Workspace: "ws1"}); kinds(p) != "signed_in,sign_in_refused,signed_in" {
		t.Fatalf("after new sign-in = %s", kinds(p))
	}
}

func TestPagingCursorOwnPositions(t *testing.T) {
	r := newRig(t)
	h := r.open()
	defer h.Close()
	for i := 0; i < 7; i++ {
		h.Offer(signIn("u1", "ws1", hA))
		h.Offer(refused("u1"))
		h.Offer(signIn("u2", "ws1", hB))
	}
	h.Flush()
	seen := 0
	q := Query{Workspace: "ws1", Limit: 4}
	for i := 0; i < 10; i++ {
		p := h.Read("u1", q)
		seen += len(p.Events)
		if !p.More {
			break
		}
		if p.NextW > 8 || p.NextR > 8 {
			t.Fatalf("cursor exceeds own positions: %d %d", p.NextW, p.NextR)
		}
		q.CursorW, q.CursorR = p.NextW, p.NextR
	}
	if seen != 14 {
		t.Fatalf("paged %d events, want 14", seen)
	}
}

func TestUncleanStopNote(t *testing.T) {
	r := newRig(t)
	h := r.open()
	h.Offer(signIn("u1", "ws1", hA))
	h.Flush()
	close(h.quit)
	h.wg.Wait()
	h.release()
	r.clock.add(time.Minute)
	h = r.open()
	defer h.Close()
	p := h.Read("u1", Query{Workspace: "ws1"})
	found := false
	for _, n := range p.Notes {
		found = found || n.Kind == "unclean_stop" && n.Count == nil
	}
	if !found || len(p.Events) != 1 {
		t.Fatalf("notes %v events %d", p.Notes, len(p.Events))
	}
}

func TestRetentionPurgeFakeClock(t *testing.T) {
	r := newRig(t)
	r.opts.Days = 2
	h := r.open()
	h.Offer(signIn("u1", "ws1", hA))
	h.Flush()
	first := r.clock.now()
	r.clock.add(48 * time.Hour)
	h.Offer(signIn("u1", "ws1", hB))
	h.Flush()
	if p := h.Read("u1", Query{Workspace: "ws1"}); len(p.Events) != 2 {
		t.Fatalf("at exact cutoff want 2 got %d", len(p.Events))
	}
	r.clock.add(time.Millisecond)
	if p := h.Read("u1", Query{Workspace: "ws1"}); len(p.Events) != 1 {
		t.Fatalf("past cutoff want 1 got %d", len(p.Events))
	}
	r.clock.add(24 * time.Hour)
	h.Tick()
	if _, err := os.Stat(filepath.Join(r.dir, DirName, first.Format(dayLayout)+daySuffix)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expired day file still present: %v", err)
	}
	h.Close()
}
