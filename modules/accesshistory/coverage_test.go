package accesshistory

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestShorterPeriodRequiresExactOneTimeAcknowledgment(t *testing.T) {
	r := newRig(t)
	h := r.open()
	h.Offer(signIn("u1", "ws1", hA))
	h.Flush()
	path := filepath.Join(r.dir, DirName, r.clock.now().Format(dayLayout)+daySuffix)
	r.clock.add(5 * 24 * time.Hour)
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	r.opts.Days = 1
	for _, ack := range []int{0, 2} {
		r.opts.AcceptShortening = ack
		if err := r.openErr(); !errors.Is(err, ErrConfig) || !strings.Contains(err.Error(), "delete 1 records") {
			t.Fatalf("shortening acknowledgment %d: %v", ack, err)
		}
		after, err := os.ReadFile(path)
		if err != nil || string(after) != string(before) {
			t.Fatalf("refused shortening changed original evidence: %v", err)
		}
	}
	r.opts.AcceptShortening = 1
	h = r.open()
	if page := h.Read("u1", Query{Workspace: "ws1"}); len(page.Events) != 0 || page.Days != 1 {
		t.Fatalf("accepted shortening not applied: %+v", page)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("accepted unheld expired day not deleted: %v", err)
	}
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
	if err := r.openErr(); !errors.Is(err, ErrConfig) || !strings.Contains(err.Error(), "no shortening is pending") {
		t.Fatalf("stale one-time shortening flag accepted: %v", err)
	}
}

func TestClockGapAcknowledgmentMatchesPersistedReference(t *testing.T) {
	r := newRig(t)
	h := r.open()
	h.Offer(signIn("u1", "ws1", hA))
	h.Flush()
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
	r.clock.add(8 * 24 * time.Hour)
	h = r.open()
	if h.state.Deferral == nil {
		t.Fatal("large unaccepted clock gap was not persisted")
	}
	logged := h.state.Deferral.Logged
	if _, ok := noteOf(h.Read("u1", Query{Workspace: "ws1"}), "purge_deferred"); !ok {
		t.Fatal("unaccepted clock gap not disclosed")
	}
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
	r.clock.add(time.Hour)
	for _, ack := range []string{"not-a-time", stamp(parseStamp(logged).Add(time.Minute))} {
		r.opts.AcceptClock = ack
		if err := r.openErr(); !errors.Is(err, ErrConfig) {
			t.Fatalf("mismatched clock acknowledgement accepted: %v", err)
		}
	}
	r.opts.AcceptClock = parseStamp(logged).Truncate(time.Minute).Format(time.RFC3339)
	h = r.open()
	if h.state.Deferral != nil || h.deferred != "" {
		t.Fatal("exact persisted clock acknowledgement did not clear deferral")
	}
	if page := h.Read("u1", Query{Workspace: "ws1"}); len(page.Events) != 1 || !page.Events[0].At.Equal(time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("clock acceptance reset event basis: %+v", page.Events)
	}
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
	if err := r.openErr(); !errors.Is(err, ErrConfig) || !strings.Contains(err.Error(), "no deletion is deferred") {
		t.Fatalf("stale one-time clock flag accepted: %v", err)
	}
}

func TestPriorRunHoldReleaseRequiresExplicitMarkerAndAudit(t *testing.T) {
	r := newRig(t)
	r.opts.Days = 1
	h := r.open()
	h.Offer(signIn("u1", "ws1", hA))
	h.Flush()
	path := filepath.Join(r.dir, DirName, r.clock.now().Format(dayLayout)+daySuffix)
	r.mu.Lock()
	r.holds = []Hold{{ID: "legal-hold", Owner: "owner", Reason: "preserve evidence", Tenants: []string{"ws1"}, Classes: []string{ClassIdentity}, PlacedAt: r.clock.now(), Covers: func(ws string) bool { return ws == "ws1" }}}
	r.mu.Unlock()
	r.clock.add(48 * time.Hour)
	h.Tick()
	if len(h.state.Markers) != 1 {
		t.Fatal("observed hold did not create exactly one marker")
	}
	id := h.state.Markers[0].ID
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
	r.mu.Lock()
	r.holds = nil
	r.mu.Unlock()
	r.opts.ReleaseMarker = id
	if err := r.openErr(); !errors.Is(err, ErrConfig) {
		t.Fatalf("reason-free hold release accepted: %v", err)
	}
	r.opts.ReleaseMarker = "hm-not-present"
	r.opts.ReleaseReason = "reviewed release"
	if err := r.openErr(); !errors.Is(err, ErrConfig) {
		t.Fatalf("unknown hold marker release accepted: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("refused release removed held file: %v", err)
	}
	r.opts.ReleaseMarker = id
	h = r.open()
	if len(h.state.Markers) != 0 {
		t.Fatal("explicit prior-run hold release retained marker")
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("released expired evidence remained: %v", err)
	}
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
	r.opts.ReleaseMarker, r.opts.ReleaseReason = "", ""
	h = r.open()
	defer h.Close()
	found := false
	for _, line := range h.audits {
		if line.Kind == "release" && line.Marker == id && line.Why == "reviewed release" && line.Snapshot != nil && line.Snapshot.Owner == "owner" {
			found = true
		}
	}
	if !found {
		t.Fatal("hold release audit did not survive restart")
	}
}
