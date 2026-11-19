package accesshistory

import (
	"errors"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type split struct {
	kept    []byte
	damaged []byte
	held    lossCounts
	removed lossCounts
	days    map[string]bool
	any     bool
}

func snapshotCovers(s snapshot, ws string) bool {
	return contains(s.Tenants, ws) && (len(s.Classes) == 0 || contains(s.Classes, ClassIdentity))
}

func (h *History) liveHolds() []Hold {
	out := []Hold{}
	for _, hold := range h.o.Holds() {
		if !hold.Released {
			out = append(out, hold)
		}
	}
	return out
}

func (h *History) heldByMarker(r Record) bool {
	if len(h.state.Markers) == 0 {
		return false
	}
	if r.Workspace == "" {
		return true
	}
	for _, m := range h.state.Markers {
		if snapshotCovers(m.Snapshot, r.Workspace) {
			return true
		}
	}
	return false
}

func (h *History) covered(r Record, live []Hold) (bool, []string) {
	ids := []string{}
	for _, m := range h.state.Markers {
		if r.Workspace == "" || snapshotCovers(m.Snapshot, r.Workspace) {
			ids = append(ids, m.ID)
		}
	}
	if len(ids) > 0 {
		return true, ids
	}
	for _, hold := range live {
		if r.Workspace == "" || (hold.Covers != nil && hold.Covers(r.Workspace)) {
			return true, ids
		}
	}
	return false, ids
}

func (h *History) splitFile(raw []byte, live []Hold) split {
	out := split{held: lossCounts{}, removed: lossCounts{}, days: map[string]bool{}}
	keepDamaged := len(live) > 0 || len(h.state.Markers) > 0
	for _, line := range splitLines(raw) {
		var r Record
		if decodeLine(line, &r) != nil || !r.valid() {
			if keepDamaged {
				out.damaged = append(append(out.damaged, line...), '\n')
				out.any = true
			}
			continue
		}
		if !Personal(r.Kind) {
			continue
		}
		if ok, ids := h.covered(r, live); ok {
			out.kept = append(append(out.kept, line...), '\n')
			out.held.add(r, 1)
			out.any = true
			for _, id := range ids {
				out.days[id] = true
			}
			continue
		}
		out.removed.add(r, 1)
	}
	return out
}

func (h *History) readDay(name string) ([]byte, error) {
	h.closeFile(name)
	f, err := h.sys.openFile(filepath.Join(h.dir, name), false)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return readAll(f)
}

func (h *History) shrinkDay(name string, live []Hold) (split, error) {
	raw, err := h.readDay(name)
	if err != nil {
		return split{}, err
	}
	sp := h.splitFile(raw, live)
	if err := h.persistSplitMarkers(name, sp); err != nil {
		return sp, err
	}
	if h.deletionDeferred() {
		return sp, errDeletionDeferred
	}
	if !sp.any {
		if err := h.sys.remove(filepath.Join(h.dir, name)); err != nil {
			return sp, err
		}
		if err := h.sys.syncDir(h.dir); err != nil {
			return sp, err
		}
		h.total -= h.sizes[name]
		delete(h.sizes, name)
		return sp, nil
	}
	if len(sp.kept) == len(raw) {
		return sp, nil
	}
	if len(sp.damaged) > 0 {
		dname := strings.TrimSuffix(name, daySuffix) + heldDamagedSuffix
		existing, _ := h.readHeldDamaged(dname)
		if err := h.replaceFile(dname, append(existing, sp.damaged...)); err != nil {
			return sp, err
		}
		h.total += int64(len(sp.damaged))
		h.sizes[dname] = int64(len(existing) + len(sp.damaged))
		for i := range h.state.Markers {
			if !contains(h.state.Markers[i].HeldDamaged, dname) {
				h.state.Markers[i].HeldDamaged = append(h.state.Markers[i].HeldDamaged, dname)
			}
		}
	}
	if err := h.replaceFile(name, sp.kept); err != nil {
		return sp, err
	}
	h.total += int64(len(sp.kept)) - h.sizes[name]
	h.sizes[name] = int64(len(sp.kept))
	return sp, nil
}

func (h *History) readHeldDamaged(name string) ([]byte, error) {
	f, err := h.sys.openFile(filepath.Join(h.dir, name), false)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return readAll(f)
}

func (h *History) purgeLocked(now time.Time) {
	h.lastPurge, h.lastRetry = now.Format(dayLayout), millis(now)
	cut := h.cutoff(now)
	h.hide(cut)
	defer h.flushPending()
	if h.deletionDeferred() {
		return
	}
	live := h.liveHolds()
	if err := h.persistHolds(live, now); err != nil {
		h.publishMaintenance(true, now)
		h.o.Log("access_history_state_failed", "error", errClass(err))
		return
	}
	names, err := h.dayNames()
	failed := err != nil
	deleted := 0
	removed := map[string]int64{}
	for _, name := range names {
		day, _ := dayOf(name)
		if millis(day.Add(24*time.Hour)) > cut {
			continue
		}
		sp, err := h.shrinkDay(name, live)
		if err != nil {
			failed = true
			h.o.Log("access_history_purge_failed", "file", name, "error", errClass(err))
			continue
		}
		deleted++
		for u, c := range sp.removed.encoded() {
			removed[u] += c
		}

	}
	if err := h.dropHeldDamaged(); err != nil {
		failed = true
	}
	if deleted > 0 {
		var total int64
		for _, c := range removed {
			total += c
		}
		h.pendingRecs = append(h.pendingRecs, Record{Kind: sysPurged, To: cut, Count: total, Days: deleted})
	}
	_, held := h.lowestKept(cut)
	h.mu.Lock()
	h.heldCount = held
	h.mu.Unlock()
	h.logMarkers(now)
	if err := h.compactPeople(now); err != nil {
		h.o.Log("access_history_compaction_failed", "error", errClass(err))
	}
	h.stateMu.Lock()
	if err := h.saveState(); err != nil {
		h.o.Log("access_history_state_failed", "error", errClass(err))
	}
	h.stateMu.Unlock()
	h.publishMaintenance(failed, now)
}

func (h *History) hide(cut int64) { h.dropBefore(cut) }

func (h *History) observeHolds(live []Hold, now time.Time) {
	for _, hold := range h.o.Holds() {
		if !hold.Released {
			continue
		}
		for i := 0; i < len(h.state.Markers); i++ {
			m := h.state.Markers[i]
			if m.Run == h.run && m.Hold == hold.ID {
				if err := h.recordRelease(m, "released in P04", now); err != nil {
					continue
				}
				h.state.Markers = append(h.state.Markers[:i:i], h.state.Markers[i+1:]...)
				delete(h.runMarkers, hold.ID)
				i--
			}
		}
	}
	for _, hold := range live {
		if h.runMarkers[hold.ID] {
			continue
		}
		h.runMarkers[hold.ID] = true
		m := marker{ID: "hm-" + randomHex(8), Hold: hold.ID, Run: h.run, FirstSeen: stamp(now), Days: []string{},
			Snapshot: snapshot{Owner: hold.Owner, Reason: hold.Reason, Tenants: append([]string{}, hold.Tenants...), Classes: append([]string{}, hold.Classes...),
				PlacedAt: stamp(hold.PlacedAt)}}
		h.state.Markers = append(h.state.Markers, m)
		h.pendingRecs = append(h.pendingRecs, Record{Kind: sysHeld, Hold: hold.ID, Run: m.ID})
	}
}

func (h *History) dropHeldDamaged() error {
	if h.deletionDeferred() {
		return errDeletionDeferred
	}
	listed := map[string]bool{}
	for _, m := range h.state.Markers {
		for _, f := range m.HeldDamaged {
			listed[f] = true
		}
	}
	var failed error
	for name, size := range h.sizes {
		if !strings.HasSuffix(name, heldDamagedSuffix) || listed[name] {
			continue
		}
		if err := h.sys.remove(filepath.Join(h.dir, name)); err != nil && !errors.Is(err, fs.ErrNotExist) {
			failed = err
			continue
		}
		h.total -= size
		delete(h.sizes, name)
	}
	if failed == nil {
		failed = h.sys.syncDir(h.dir)
	}
	return failed
}

func (h *History) lowestKept(cut int64) (map[string]uint64, lossCounts) {
	h.mu.RLock()
	floors := make(map[string]uint64, len(h.floors))
	for user, seq := range h.floors {
		floors[user] = seq
	}
	h.mu.RUnlock()
	low := map[string]uint64{}
	held := lossCounts{}
	_ = h.eachDayFile(func(name string, day time.Time, raw []byte) {
		for _, line := range splitLines(raw) {
			var r Record
			if decodeLine(line, &r) != nil {
				if source, ok := bestEffortRecord(line); ok {
					if v, exists := low[source.User]; !exists || source.Seq < v {
						low[source.User] = source.Seq
					}
				}
				continue
			}
			if r.Kind == sysDropped || r.Kind == sysEarlyDeletion {
				for key := range r.Per {
					if scope, span, ok := decodeAttribution(key); ok {
						if v, exists := low[scope.user]; !exists || span.first < v {
							low[scope.user] = span.first
						}
					}
				}
			}
			if !Personal(r.Kind) {
				continue
			}
			source := r.Seq
			if r.Kind == KindSignInRefusedMore {
				source = r.SourceSeq
			}
			if v, ok := low[r.User]; !ok || source < v {
				low[r.User] = source
			}
			if r.At < cut && recordVisible(r, floors[r.User]) && h.heldByMarker(r) {
				held.add(r, 1)
			}
		}
	})
	return low, held
}

func (h *History) logMarkers(now time.Time) {
	for _, m := range h.state.Markers {
		var records, bytes int64
		for _, d := range m.Days {
			bytes += h.sizes[d+daySuffix]
		}
		h.mu.RLock()
		for _, c := range h.heldCount {
			records += c.count
		}
		h.mu.RUnlock()
		beyond := 0
		if len(m.Days) > 0 {
			sort.Strings(m.Days)
			if oldest, err := time.Parse(dayLayout, m.Days[0]); err == nil {
				beyond = int(now.Sub(oldest).Hours()/24) - h.state.Days
			}
		}
		h.o.Log("access_history_held", "marker", m.ID, "hold", m.Hold, "owner", m.Snapshot.Owner, "reason", m.Snapshot.Reason,
			"tenants", strings.Join(m.Snapshot.Tenants, ","), "classes", strings.Join(m.Snapshot.Classes, ","), "placed_at", m.Snapshot.PlacedAt,
			"first_observed", m.FirstSeen, "days_beyond_period", beyond, "records", records, "bytes", bytes)
	}
}

func (h *History) heldBytes() int64 {
	var total int64
	for _, m := range h.state.Markers {
		for _, d := range m.Days {
			total += h.sizes[d+daySuffix]
		}
		for _, f := range m.HeldDamaged {
			total += h.sizes[f]
		}
	}
	return total
}

func (h *History) coverage() int {
	days := []string{}
	for name := range h.sizes {
		if _, ok := dayOf(name); ok {
			days = append(days, name)
		}
	}
	sort.Strings(days)
	if len(days) > 7 {
		days = days[len(days)-7:]
	}
	var sum int64
	for _, d := range days {
		sum += h.sizes[d]
	}
	if len(days) == 0 || sum == 0 {
		return h.state.Days
	}
	rate := sum / int64(len(days))
	if rate == 0 {
		return h.state.Days
	}
	return int(min(int64(h.state.Days), h.o.MaxBytes/rate))
}

func (h *History) makeRoom(size int64, byDay map[string][]byte) error {
	newest := ""
	for name := range byDay {
		newest = max(newest, name)
	}
	for h.total+size > h.o.MaxBytes {
		if h.deletionDeferred() {
			return errFull
		}
		if err := h.persistHolds(h.liveHolds(), h.now()); err != nil {
			return err
		}
		names, err := h.dayNames()
		if err != nil {
			return err
		}
		oldest := ""
		for _, n := range names {
			if n < newest {
				oldest = n
				break
			}
		}
		if oldest == "" {
			return errFull
		}
		live := h.liveHolds()
		before := h.sizes[oldest]
		sp, err := h.shrinkDay(oldest, live)
		if err != nil {
			return err
		}
		if _, still := h.sizes[oldest]; still && h.sizes[oldest] >= before {
			return errFullHeld
		}
		day, _ := dayOf(oldest)
		next := millis(day.Add(24 * time.Hour))
		h.dropBefore(next)
		var total int64
		for _, c := range sp.removed {
			total += c.count
		}
		h.pendingRecs = append(h.pendingRecs, Record{Kind: sysEarlyDeletion, From: millis(day), To: next, Per: sp.removed.encoded(), Count: total})
		h.mu.Lock()
		h.addLossPer("early_deletion", millis(day), next, sp.removed.encoded(), total)
		h.mu.Unlock()
		h.o.Log("access_history_early_deletion", "day", strings.TrimSuffix(oldest, daySuffix), "records", total, "held_bytes", h.heldBytes())
	}
	return nil
}

func (h *History) dropBefore(at int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, p := range h.persons {
		for _, s := range p.seqs {
			kept := s.entries[:0]
			for _, e := range s.entries {
				if e.at >= at {
					kept = append(kept, e)
				}
			}
			s.entries = kept
		}
	}
}
