package accesshistory

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func Open(o Options) (*History, error) { return open(o, platform{}) }

func open(o Options, sys system) (*History, error) {
	if o.Dir == "" || o.Days < 1 || o.Days > 1095 || o.MaxBytes <= 0 || o.Clock == nil || o.ClockGap <= 0 {
		return nil, fmt.Errorf("%w: options", ErrConfig)
	}
	if o.Log == nil {
		o.Log = func(string, ...any) {}
	}
	if o.Holds == nil {
		o.Holds = func() []Hold { return nil }
	}
	if o.ClockUnverified == nil {
		o.ClockUnverified = func() bool { return false }
	}
	h := &History{o: o, sys: sys, dir: filepath.Join(o.Dir, DirName), queue: make(chan item, queueSize), quit: make(chan struct{}),
		dropped: lossCounts{}, floors: map[string]uint64{}, floorAt: map[string]int64{}, persons: map[string]*person{}, files: map[string]handle{},
		sizes: map[string]int64{}, refused: map[string]int{}, more: map[string]int64{}, moreSeq: map[string]uint64{}, heldCount: lossCounts{}, workspaces: map[string]string{},
		runMarkers: map[string]bool{}}
	h.run = randomHex(8)
	if err := h.prepareDir(); err != nil {
		return nil, err
	}
	ok := false
	defer func() {
		if !ok {
			h.release()
		}
	}()
	if err := h.readState(); err != nil {
		return nil, err
	}
	if err := h.readPeople(); err != nil {
		return nil, err
	}
	now := h.now()
	if err := h.applyFlags(now); err != nil {
		return nil, err
	}
	starts, err := h.scan(now)
	if err != nil {
		return nil, err
	}
	h.writtenSeq = h.seq
	if err := h.checkClockGap(now); err != nil {
		return nil, err
	}
	if err := h.saveState(); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrConfig, retentionFile, err)
	}
	h.unclean(starts, now)
	if err := h.appendDirect(Record{Kind: sysRunStarted, Run: h.run}, now); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrConfig, h.dir, err)
	}
	if h.periodChange != nil {
		h.pendingRecs = append(h.pendingRecs, *h.periodChange)
	}
	h.flushPending()
	if err := h.saveState(); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrConfig, retentionFile, err)
	}
	h.effective = h.coverage()
	if h.effective < h.state.Days {
		h.o.Log("access_history_bound_short", "days", h.state.Days, "effective_days", h.effective)
	}
	h.purgeLocked(now)
	h.wg.Add(1)
	go h.writer()
	ok = true
	return h, nil
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (h *History) now() time.Time { return h.o.Clock().UTC() }

func (h *History) prepareDir() error {
	info, err := os.Lstat(h.dir)
	if errors.Is(err, fs.ErrNotExist) {
		if err := os.Mkdir(h.dir, 0o700); err != nil {
			return fmt.Errorf("%w: %s: %v", ErrConfig, h.dir, err)
		}
		if err := h.sys.syncDir(h.o.Dir); err != nil {
			return fmt.Errorf("%w: %s: %v", ErrConfig, h.o.Dir, err)
		}
		info, err = os.Lstat(h.dir)
	}
	if err != nil {
		return fmt.Errorf("%w: %s: %v", ErrConfig, h.dir, err)
	}
	if err := h.sys.checkDir(info); err != nil {
		return fmt.Errorf("%w: %s: %v", ErrConfig, h.dir, err)
	}
	lock, err := h.openOrCreate(filepath.Join(h.dir, lockFile))
	if err != nil {
		return err
	}
	if err := h.sys.lock(lock); err != nil {
		lock.Close()
		if errors.Is(err, ErrLocked) {
			return fmt.Errorf("%w: %s", ErrLocked, h.dir)
		}
		return fmt.Errorf("%w: %s: %v", ErrConfig, lockFile, err)
	}
	h.lockH = lock
	entries, err := os.ReadDir(h.dir)
	if err != nil {
		return fmt.Errorf("%w: %s: %v", ErrConfig, h.dir, err)
	}
	removed := false
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), tempSuffix) {
			if err := h.sys.remove(filepath.Join(h.dir, e.Name())); err != nil {
				return fmt.Errorf("%w: %s: %v", ErrConfig, e.Name(), err)
			}
			removed = true
		}
	}
	if removed {
		if err := h.sys.syncDir(h.dir); err != nil {
			return fmt.Errorf("%w: %s: %v", ErrConfig, h.dir, err)
		}
	}
	return nil
}

func (h *History) openOrCreate(path string) (handle, error) {
	f, err := h.sys.openFile(path, false)
	if errors.Is(err, fs.ErrNotExist) {
		f, err = h.sys.openFile(path, true)
		if err == nil {
			if serr := h.sys.syncDir(h.dir); serr != nil {
				f.Close()
				return nil, fmt.Errorf("%w: %s: %v", ErrConfig, h.dir, serr)
			}
		}
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrConfig, filepath.Base(path), err)
	}
	return f, nil
}

func readAll(f handle) ([]byte, error) {
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	buf := make([]byte, info.Size())
	if len(buf) == 0 {
		return buf, nil
	}
	n, err := f.ReadAt(buf, 0)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	return buf[:n], nil
}

func (h *History) readState() error {
	path := filepath.Join(h.dir, retentionFile)
	f, err := h.sys.openFile(path, false)
	if errors.Is(err, fs.ErrNotExist) {
		h.state = stateFile{Days: h.o.Days, History: []periodEntry{{Days: h.o.Days, Since: stamp(h.now())}}, Markers: []marker{}}
		return nil
	}
	if err != nil {
		return fmt.Errorf("%w: %s: %v", ErrConfig, retentionFile, err)
	}
	defer f.Close()
	raw, err := readAll(f)
	if err != nil {
		return fmt.Errorf("%w: %s: %v", ErrConfig, retentionFile, err)
	}
	if err := json.Unmarshal(raw, &h.state); err != nil || h.state.Days < 1 {
		return fmt.Errorf("%w: %s: unreadable", ErrConfig, retentionFile)
	}
	if h.state.Markers == nil {
		h.state.Markers = []marker{}
	}
	return nil
}

func stamp(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func parseStamp(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

func (h *History) saveState() error {
	h.offerMu.Lock()
	h.state.Seq = max(h.state.Seq, h.seq)
	h.offerMu.Unlock()
	if h.lastAt > 0 {
		h.state.LastAt = stamp(fromMillis(h.lastAt))
	}
	raw, err := json.Marshal(h.state)
	if err != nil {
		return err
	}
	return h.replaceFile(retentionFile, raw)
}

func (h *History) replaceFile(name string, raw []byte) error {
	temp := filepath.Join(h.dir, name+"."+randomHex(4)+tempSuffix)
	f, err := h.sys.openFile(temp, true)
	if err != nil {
		return err
	}
	if _, err := f.Write(raw); err != nil {
		f.Close()
		_ = h.sys.remove(temp)
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		_ = h.sys.remove(temp)
		return err
	}
	if err := f.Close(); err != nil {
		_ = h.sys.remove(temp)
		return err
	}
	if h.beforeWrite != nil {
		h.beforeWrite()
	}
	if err := h.sys.rename(temp, filepath.Join(h.dir, name)); err != nil {
		_ = h.sys.remove(temp)
		return err
	}
	return h.sys.syncDir(h.dir)
}

func (h *History) applyFlags(now time.Time) error {
	shorter := h.o.Days < h.state.Days
	switch {
	case h.o.AcceptShortening != 0 && !shorter:
		return fmt.Errorf("%w: accept-access-history-shortening is given but no shortening is pending", ErrConfig)
	case shorter && h.o.AcceptShortening != h.o.Days:
		records, bytes, people, held := h.shorteningImpact(now)
		return fmt.Errorf("%w: access_history_days would shorten the access history from %d to %d days and delete %d records (%d bytes) of %d people; %d bytes are kept by legal holds; restart with -accept-access-history-shortening=%d to apply",
			ErrConfig, h.state.Days, h.o.Days, records, bytes, people, held, h.o.Days)
	}
	if h.o.Days != h.state.Days {
		h.o.Log("access_history_period_changed", "from_days", h.state.Days, "to_days", h.o.Days)
		h.state.History = append(h.state.History, periodEntry{Days: h.o.Days, Since: stamp(now)})
		h.periodChange = &Record{Kind: sysPeriodChanged, Days: h.o.Days, Count: int64(h.state.Days)}
		h.state.Days = h.o.Days
	}
	if h.o.ReleaseMarker != "" || h.o.ReleaseReason != "" {
		if err := h.releaseByFlag(now); err != nil {
			return err
		}
	}
	return nil
}

func (h *History) shorteningImpact(now time.Time) (records, bytes int64, people int, held int64) {
	cutoff := millis(now.Add(-time.Duration(h.o.Days) * 24 * time.Hour))
	seen := map[string]bool{}
	_ = h.eachDayFile(func(name string, day time.Time, raw []byte) {
		for _, line := range splitLines(raw) {
			var r Record
			if decodeLine(line, &r) != nil || !Personal(r.Kind) || r.At >= cutoff {
				continue
			}
			if h.heldByMarker(r) {
				held += int64(len(line)) + 1
				continue
			}
			records++
			bytes += int64(len(line)) + 1
			seen[r.User] = true
		}
	})
	return records, bytes, len(seen), held
}

func splitLines(raw []byte) [][]byte {
	out := [][]byte{}
	for len(raw) > 0 {
		i := indexByte(raw, '\n')
		if i < 0 {
			break
		}
		out = append(out, raw[:i])
		raw = raw[i+1:]
	}
	return out
}

func indexByte(b []byte, c byte) int {
	for i, x := range b {
		if x == c {
			return i
		}
	}
	return -1
}

func dayOf(name string) (time.Time, bool) {
	if !strings.HasSuffix(name, daySuffix) {
		return time.Time{}, false
	}
	d, err := time.Parse(dayLayout, strings.TrimSuffix(name, daySuffix))
	if err != nil || d.Format(dayLayout)+daySuffix != name {
		return time.Time{}, false
	}
	return d.UTC(), true
}

func (h *History) dayNames() ([]string, error) {
	entries, err := os.ReadDir(h.dir)
	if err != nil {
		return nil, err
	}
	out := []string{}
	for _, e := range entries {
		if _, ok := dayOf(e.Name()); ok {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out, nil
}

func (h *History) eachDayFile(visit func(name string, day time.Time, raw []byte)) error {
	names, err := h.dayNames()
	if err != nil {
		return err
	}
	for _, name := range names {
		day, _ := dayOf(name)
		f, err := h.sys.openFile(filepath.Join(h.dir, name), false)
		if err != nil {
			return fmt.Errorf("%w: %s: %v", ErrConfig, name, err)
		}
		raw, err := readAll(f)
		f.Close()
		if err != nil {
			return fmt.Errorf("%w: %s: %v", ErrConfig, name, err)
		}
		visit(name, day, raw)
	}
	return nil
}

func (h *History) release() {
	for name, f := range h.files {
		f.Close()
		delete(h.files, name)
	}
	if h.people != nil {
		h.people.Close()
		h.people = nil
	}
	if h.lockH != nil {
		h.lockH.Close()
		h.lockH = nil
	}
}
