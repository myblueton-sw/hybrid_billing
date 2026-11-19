package accesshistory

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type faultSys struct {
	platform
	mu        sync.Mutex
	writeErr  func(path string) bool
	syncErr   func(path string) bool
	renameErr func(to string) bool
	removeErr func(path string) bool
}

type faultHandle struct {
	handle
	path string
	sys  *faultSys
}

func (f *faultSys) check(fn func(string) bool, path string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return fn != nil && fn(path)
}

func (f *faultSys) openFile(path string, create bool) (handle, error) {
	h, err := f.platform.openFile(path, create)
	if err != nil {
		return nil, err
	}
	return &faultHandle{handle: h, path: path, sys: f}, nil
}

func (f *faultSys) rename(from, to string) error {
	if f.check(f.renameErr, to) {
		return errInjected
	}
	return f.platform.rename(from, to)
}

func (f *faultSys) remove(path string) error {
	if f.check(f.removeErr, path) {
		return errInjected
	}
	return f.platform.remove(path)
}

func (h *faultHandle) Write(b []byte) (int, error) {
	if h.sys.check(h.sys.writeErr, h.path) {
		n, _ := h.handle.Write(b[:len(b)/2])
		return n, errInjected
	}
	return h.handle.Write(b)
}

func (h *faultHandle) Sync() error {
	if h.sys.check(h.sys.syncErr, h.path) {
		return errInjected
	}
	return h.handle.Sync()
}

func dayFile(path string) bool { _, ok := dayOf(filepath.Base(path)); return ok }

func noteOf(p Page, kind string) (Note, bool) {
	for _, n := range p.Notes {
		if n.Kind == kind {
			return n, true
		}
	}
	return Note{}, false
}

func TestWriteAndSyncFailuresStopRecordingAndCountDrops(t *testing.T) {
	for name, set := range map[string]func(*faultSys){
		"write": func(f *faultSys) { f.writeErr = dayFile },
		"sync":  func(f *faultSys) { f.syncErr = dayFile },
	} {
		t.Run(name, func(t *testing.T) {
			r := newRig(t)
			fs := &faultSys{}
			r.sys = fs
			h := r.open()
			h.Offer(signIn("u1", "ws1", hA))
			h.Flush()
			fs.mu.Lock()
			set(fs)
			fs.mu.Unlock()
			h.Offer(signIn("u1", "ws1", hB))
			h.Flush()
			h.Offer(signIn("u1", "ws1", hB))
			h.Offer(signIn("u2", "ws1", hB))
			p := h.Read("u1", Query{Workspace: "ws1"})
			if p.Recording.State != "stopped" || p.Recording.Reason != "write_failed" || len(p.Events) != 1 {
				t.Fatalf("recording %+v events %d", p.Recording, len(p.Events))
			}
			n, ok := noteOf(p, "dropped")
			if !ok || n.Count == nil || *n.Count != 2 {
				t.Fatalf("own dropped note %+v", p.Notes)
			}
			other := h.Read("u3", Query{Workspace: "ws1"})
			if n, ok := noteOf(other, "dropped"); !ok || n.Count != nil {
				t.Fatalf("unattributed note for a bystander %+v", other.Notes)
			}
			if !r.logged("access_history_stopped") {
				t.Fatal("stop not logged")
			}
			h.Close()
		})
	}
}

func TestOfferNeverBlocksAndCountsOverflow(t *testing.T) {
	r := newRig(t)
	h := r.open()
	release, entered := make(chan struct{}), make(chan struct{})
	go h.control(func() { close(entered); <-release })
	<-entered
	start := time.Now()
	for i := 0; i < queueSize+25; i++ {
		h.Offer(refused("u1"))
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Fatalf("offers took %v", d)
	}
	close(release)
	h.Flush()
	p := h.Read("u1", Query{Workspace: "ws1"})
	n, ok := noteOf(p, "dropped")
	if !ok || n.Count == nil || *n.Count < 25 {
		t.Fatalf("dropped note %+v", p.Notes)
	}
	h.Close()
}

func TestTornTailAndDamagedLines(t *testing.T) {
	r := newRig(t)
	h := r.open()
	h.Offer(signIn("u1", "ws1", hA))
	h.Offer(signIn("u2", "ws1", hB))
	h.Flush()
	h.Close()
	path := filepath.Join(r.dir, DirName, r.clock.now().Format(dayLayout)+daySuffix)
	raw, _ := os.ReadFile(path)
	lines := strings.SplitAfter(string(raw), "\n")
	var u2 int
	for i, l := range lines {
		if strings.Contains(l, `"u":"u2"`) {
			u2 = i
		}
	}
	lines[u2] = strings.Replace(lines[u2], `"k":"signed_in"`, `"k":"signed_im"`, 1)
	damaged := strings.Join(lines, "") + `0000 {"v":1,"s":99`
	if err := os.WriteFile(path, []byte(damaged), 0o600); err != nil {
		t.Fatal(err)
	}
	r.clock.add(time.Minute)
	h = r.open()
	defer h.Close()
	p := h.Read("u1", Query{Workspace: "ws1"})
	if len(p.Events) != 1 {
		t.Fatalf("u1 events %d", len(p.Events))
	}
	if _, ok := noteOf(p, "recovered_tail"); !ok {
		t.Fatalf("no recovered tail note %+v", p.Notes)
	}
	if n, ok := noteOf(p, "damaged"); !ok || n.Count != nil {
		t.Fatalf("u1 damaged note should have no count %+v", p.Notes)
	}
	p2 := h.Read("u2", Query{Workspace: "ws1"})
	if n, ok := noteOf(p2, "damaged"); !ok || n.Count == nil || *n.Count != 1 || len(p2.Events) != 0 {
		t.Fatalf("u2 damaged note %+v events %d", p2.Notes, len(p2.Events))
	}
	after, _ := os.ReadFile(path)
	if strings.Contains(string(after), `"s":99`) {
		t.Fatal("torn tail not truncated")
	}
}

func TestPeopleLogDamageRefusesStartAndTornTailIsTruncated(t *testing.T) {
	r := newRig(t)
	h := r.open()
	for _, u := range []string{"a", "b", "c"} {
		h.Offer(signIn(u, "ws1", hA))
		h.Flush()
		if err := h.Incarnation(u); err != nil {
			t.Fatal(err)
		}
	}
	h.Close()
	path := filepath.Join(r.dir, DirName, peopleFile)
	raw, _ := os.ReadFile(path)
	torn := append(append([]byte{}, raw...), []byte(`1234abcd {"k":"incarnation","u":"d"`)...)
	if err := os.WriteFile(path, torn, 0o600); err != nil {
		t.Fatal(err)
	}
	h = r.open()
	if len(h.floors) != 3 {
		t.Fatalf("floors after torn tail %v", h.floors)
	}
	h.Close()
	lines := strings.SplitAfter(string(raw), "\n")
	lines[1] = strings.Replace(lines[1], `"u":"b"`, `"u":"x"`, 1)
	if err := os.WriteFile(path, []byte(strings.Join(lines, "")), 0o600); err != nil {
		t.Fatal(err)
	}
	err := r.openErr()
	if !errors.Is(err, ErrConfig) || !errors.Is(err, ErrDamaged) || !strings.Contains(err.Error(), "people.log line 2") {
		t.Fatalf("damaged people.log: %v", err)
	}
}

func TestIncarnationFloorIsAboveQueuedRecords(t *testing.T) {
	r := newRig(t)
	h := r.open()
	defer h.Close()
	release, entered := make(chan struct{}), make(chan struct{})
	go h.control(func() { close(entered); <-release })
	<-entered
	h.Offer(signIn("u1", "ws1", hA))
	h.Offer(refused("u1"))
	if err := h.Incarnation("u1"); err != nil {
		t.Fatal(err)
	}
	close(release)
	h.Flush()
	if p := h.Read("u1", Query{Workspace: "ws1"}); len(p.Events) != 0 {
		t.Fatalf("queued records of the earlier holder visible: %s", kinds(p))
	}
	h.Offer(signIn("u1", "ws1", hA))
	h.Flush()
	if p := h.Read("u1", Query{Workspace: "ws1"}); len(p.Events) != 1 {
		t.Fatalf("new holder events %d", len(p.Events))
	}
}

func TestIncarnationWritesWithWriterStoppedAndFailsClosed(t *testing.T) {
	r := newRig(t)
	fs := &faultSys{}
	r.sys = fs
	h := r.open()
	h.Offer(signIn("u1", "ws1", hA))
	h.Flush()
	fs.mu.Lock()
	fs.writeErr = dayFile
	fs.mu.Unlock()
	h.Offer(signIn("u1", "ws1", hB))
	h.Flush()
	if err := h.Incarnation("u1"); err != nil {
		t.Fatalf("marker with the writer stopped: %v", err)
	}
	if p := h.Read("u1", Query{Workspace: "ws1"}); len(p.Events) != 0 {
		t.Fatalf("floor not applied: %d", len(p.Events))
	}
	fs.mu.Lock()
	fs.writeErr = func(p string) bool { return strings.HasSuffix(p, peopleFile) }
	fs.mu.Unlock()
	if err := h.Incarnation("u2"); err == nil {
		t.Fatal("marker write failure not reported")
	}
	if _, ok := h.floors["u2"]; ok {
		t.Fatal("floor set despite the failure")
	}
	h.Close()
}

func TestPeopleCompactionRacesAdds(t *testing.T) {
	r := newRig(t)
	h := r.open()
	for i := 0; i < 50; i++ {
		h.Offer(signIn("old"+itoa(i), "ws1", hA))
		_ = h.Incarnation("old" + itoa(i))
	}
	var wg sync.WaitGroup
	var added atomic.Int64
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 40; i++ {
				u := "new" + itoa(g) + "-" + itoa(i)
				h.Offer(signIn(u, "ws1", hA))
				if h.Incarnation(u) == nil {
					added.Add(1)
				}
			}
		}(g)
	}
	for i := 0; i < 20; i++ {
		h.control(func() { _ = h.compactPeople(h.now()) })
	}
	wg.Wait()
	h.Offer(signIn("keep", "ws1", hA))
	h.Flush()
	if err := h.Incarnation("keep"); err != nil {
		t.Fatal(err)
	}
	want := map[string]uint64{}
	for u, f := range h.floors {
		want[u] = f
	}
	h.Close()
	h = r.open()
	defer h.Close()
	for u, f := range want {
		if h.floors[u] != f {
			t.Fatalf("floor of %s lost: %d want %d", u, h.floors[u], f)
		}
	}
	if added.Load() == 0 {
		t.Fatal("no concurrent adds")
	}
}

func TestPerFileChecksRefuseStart(t *testing.T) {
	cases := map[string]func(dir string) error{
		"symlinked day file": func(dir string) error {
			return os.Symlink(filepath.Join(dir, peopleFile), filepath.Join(dir, "2026-03-01"+daySuffix))
		},
		"group readable retention": func(dir string) error { return os.Chmod(filepath.Join(dir, retentionFile), 0o640) },
		"hard linked people": func(dir string) error {
			return os.Link(filepath.Join(dir, peopleFile), filepath.Join(r0(dir), "people-copy"))
		},
		"symlinked held damaged": func(dir string) error {
			return os.Symlink(filepath.Join(dir, peopleFile), filepath.Join(dir, "2026-03-01"+heldDamagedSuffix))
		},
		"group readable lock": func(dir string) error { return os.Chmod(filepath.Join(dir, lockFile), 0o660) },
	}
	for name, damage := range cases {
		t.Run(name, func(t *testing.T) {
			r := newRig(t)
			h := r.open()
			h.Close()
			if err := damage(filepath.Join(r.dir, DirName)); err != nil {
				t.Fatal(err)
			}
			if err := r.openErr(); !errors.Is(err, ErrConfig) {
				t.Fatalf("start not refused: %v", err)
			}
		})
	}
}

func r0(dir string) string { return filepath.Dir(dir) }

func TestSecondProcessIsRefused(t *testing.T) {
	r := newRig(t)
	h := r.open()
	defer h.Close()
	if err := r.openErr(); !errors.Is(err, ErrLocked) {
		t.Fatalf("second open: %v", err)
	}
}

func TestCrashBetweenTempSyncAndRenameKeepsOriginal(t *testing.T) {
	r := newRig(t)
	h := r.open()
	before, _ := os.ReadFile(filepath.Join(r.dir, DirName, retentionFile))
	h.beforeWrite = func() { panic("crash") }
	func() {
		defer func() { _ = recover() }()
		_ = h.saveState()
	}()
	close(h.quit)
	h.wg.Wait()
	h.release()
	temps, _ := filepath.Glob(filepath.Join(r.dir, DirName, "*"+tempSuffix))
	if len(temps) != 1 {
		t.Fatalf("temp files %v", temps)
	}
	after, _ := os.ReadFile(filepath.Join(r.dir, DirName, retentionFile))
	if string(after) != string(before) {
		t.Fatal("original changed before the rename")
	}
	h = r.open()
	defer h.Close()
	if temps, _ := filepath.Glob(filepath.Join(r.dir, DirName, "*"+tempSuffix)); len(temps) != 0 {
		t.Fatalf("leftover temp files %v", temps)
	}
}

func TestSeqContinuesAfterEveryDayFileIsDeleted(t *testing.T) {
	r := newRig(t)
	h := r.open()
	for i := 0; i < 5; i++ {
		h.Offer(signIn("u1", "ws1", hA))
	}
	h.Flush()
	h.Close()
	high := h.seq
	files, _ := filepath.Glob(filepath.Join(r.dir, DirName, "*"+daySuffix))
	for _, f := range files {
		_ = os.Remove(f)
	}
	h = r.open()
	defer h.Close()
	h.Offer(signIn("u1", "ws1", hA))
	h.Flush()
	if h.seq <= high {
		t.Fatalf("seq %d not above %d", h.seq, high)
	}
}

func TestCloseCountsLateRecordsAndLogsOffersAfterClose(t *testing.T) {
	r := newRig(t)
	h := r.open()
	h.Offer(signIn("u1", "ws1", hA))
	h.Close()
	h.Offer(signIn("u1", "ws1", hB))
	if !r.logged("access_history_offer_after_close") || h.afterClose != 1 {
		t.Fatalf("offer after close not counted: %d", h.afterClose)
	}
	raw, _ := os.ReadFile(filepath.Join(r.dir, DirName, r.clock.now().Format(dayLayout)+daySuffix))
	text := string(raw)
	if strings.LastIndex(text, `"k":"signed_in"`) > strings.LastIndex(text, `"k":"run_stopped"`) {
		t.Fatal("record after run_stopped")
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	out := ""
	for i > 0 {
		out = string(rune('0'+i%10)) + out
		i /= 10
	}
	return out
}
