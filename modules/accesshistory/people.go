package accesshistory

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"
)

func (h *History) readPeople() error {
	path := filepath.Join(h.dir, peopleFile)
	f, err := h.openOrCreate(path)
	if err != nil {
		return err
	}
	h.people = f
	raw, err := readAll(f)
	if err != nil {
		return fmt.Errorf("%w: %s: %v", ErrConfig, peopleFile, err)
	}
	if n := len(raw); n > 0 && raw[n-1] != '\n' {
		keep := int64(strings.LastIndexByte(string(raw), '\n') + 1)
		if err := f.Truncate(keep); err != nil {
			return fmt.Errorf("%w: %s: %v", ErrConfig, peopleFile, err)
		}
		if err := f.Sync(); err != nil {
			return fmt.Errorf("%w: %s: %v", ErrConfig, peopleFile, err)
		}
		h.o.Log("access_history_people_tail_truncated", "bytes", int64(n)-keep)
		raw = raw[:keep]
	}
	h.peopleSize = int64(len(raw))
	for i, line := range splitLines(raw) {
		var p peopleLine
		if decodeLine(line, &p) != nil || !p.valid() {
			return fmt.Errorf("%w: %w: %s line %d", ErrConfig, ErrDamaged, peopleFile, i+1)
		}
		switch p.Kind {
		case sysIncarnation:
			if p.Floor >= h.floors[p.User] {
				h.floors[p.User], h.floorAt[p.User] = p.Floor, p.At
			}
		default:
			h.audits = append(h.audits, p)
		}
	}
	return nil
}

func (p peopleLine) valid() bool {
	switch p.Kind {
	case sysIncarnation:
		return p.User != "" && len(p.User) <= 128 && p.At > 0
	case "release":
		return p.Marker != "" && p.Snapshot != nil && p.Why != "" && p.At > 0
	case "release_count":
		return p.Year > 0 && p.Count > 0
	}
	return false
}

func (h *History) appendPeople(p peopleLine) error {
	if h.people == nil {
		return ErrClosed
	}
	line, err := encodeLine(p)
	if err != nil {
		return err
	}
	if h.failPeople != nil && h.failPeople() {
		return errInjected
	}
	if _, err := h.people.Seek(0, io.SeekEnd); err != nil {
		return err
	}
	n, err := h.people.Write(line)
	if err != nil {
		if n > 0 {
			_ = h.people.Truncate(h.peopleSize)
		}
		return err
	}
	if err := h.people.Sync(); err != nil {
		_ = h.people.Truncate(h.peopleSize)
		return err
	}
	h.peopleSize += int64(len(line))
	return nil
}

func (h *History) Incarnation(user string) error {
	if user == "" {
		return ErrConfig
	}
	h.stateMu.Lock()
	defer h.stateMu.Unlock()
	h.offerMu.Lock()
	floor := h.seq
	closed := h.closed
	h.offerMu.Unlock()
	if closed {
		return ErrClosed
	}
	at := millis(h.now())
	if err := h.appendPeople(peopleLine{Kind: sysIncarnation, User: user, Floor: floor, At: at}); err != nil {
		h.o.Log("access_history_incarnation_failed", "error", err.Error())
		return err
	}
	h.mu.Lock()
	h.floors[user], h.floorAt[user] = floor, at
	delete(h.persons, user)
	for scope := range h.heldCount {
		if scope.user == user {
			delete(h.heldCount, scope)
		}
	}
	h.offerMu.Lock()
	delete(h.more, user)
	delete(h.moreSeq, user)
	delete(h.refused, user)
	h.offerMu.Unlock()
	h.mu.Unlock()
	h.Offer(Record{Kind: sysIncarnation, User: user})
	return nil
}

func (h *History) floorNeeded(user string, floor uint64, minSeq map[string]uint64) bool {
	if floor > h.writtenSeq {
		return true
	}
	low, ok := minSeq[user]
	return ok && low <= floor
}

func (h *History) compactPeople(now time.Time) error {
	h.stateMu.Lock()
	defer h.stateMu.Unlock()
	minSeq, _ := h.lowestKept(h.cutoff(now))
	keep := []peopleLine{}
	dropped := false
	for user, floor := range h.floors {
		if !h.floorNeeded(user, floor, minSeq) {
			dropped = true
			continue
		}
		keep = append(keep, peopleLine{Kind: sysIncarnation, User: user, Floor: floor, At: h.floorAt[user]})
	}
	limit := now.AddDate(-auditKeepYears, 0, -h.state.Days)
	counts := map[int]int64{}
	audits := []peopleLine{}
	for _, a := range h.audits {
		if a.Kind == "release" && fromMillis(a.At).Before(limit) {
			counts[fromMillis(a.At).Year()]++
			dropped = true
			continue
		}
		if a.Kind == "release_count" {
			counts[a.Year] += a.Count
			continue
		}
		audits = append(audits, a)
	}
	if !dropped {
		return nil
	}
	for year, c := range counts {
		audits = append(audits, peopleLine{Kind: "release_count", Year: year, Count: c, At: millis(now)})
	}
	var raw []byte
	for _, p := range append(keep, audits...) {
		line, err := encodeLine(p)
		if err != nil {
			return err
		}
		raw = append(raw, line...)
	}
	if h.people != nil {
		h.people.Close()
		h.people = nil
	}
	err := h.replaceFile(peopleFile, raw)
	f, oerr := h.sys.openFile(filepath.Join(h.dir, peopleFile), false)
	if oerr != nil {
		return oerr
	}
	h.people = f
	if err != nil {
		return err
	}
	h.peopleSize = int64(len(raw))
	h.audits = audits
	h.mu.Lock()
	for user, floor := range h.floors {
		if !h.floorNeeded(user, floor, minSeq) {
			delete(h.floors, user)
			delete(h.floorAt, user)
		}
	}
	h.mu.Unlock()
	return nil
}

func (h *History) releaseByFlag(now time.Time) error {
	if h.o.ReleaseMarker == "" || h.o.ReleaseReason == "" {
		return fmt.Errorf("%w: release-access-history-marker and release-access-history-reason must be given together", ErrConfig)
	}
	if len(h.state.Markers) == 0 {
		return fmt.Errorf("%w: release-access-history-marker is given but no held marker is pending", ErrConfig)
	}
	for i, m := range h.state.Markers {
		if m.ID != h.o.ReleaseMarker {
			continue
		}
		if err := h.recordRelease(m, h.o.ReleaseReason, now); err != nil {
			return fmt.Errorf("%w: %s: %v", ErrConfig, peopleFile, err)
		}
		h.state.Markers = append(h.state.Markers[:i:i], h.state.Markers[i+1:]...)
		return nil
	}
	return fmt.Errorf("%w: no held marker %s", ErrConfig, h.o.ReleaseMarker)
}

func (h *History) recordRelease(m marker, why string, now time.Time) error {
	snap := m.Snapshot
	line := peopleLine{Kind: "release", Marker: m.ID, Hold: m.Hold, Snapshot: &snap, Why: why, At: millis(now), Run: h.run}
	if err := h.appendPeople(line); err != nil {
		return err
	}
	h.audits = append(h.audits, line)
	h.pendingRecs = append(h.pendingRecs, Record{Kind: sysHoldReleased, Hold: m.Hold, Run: m.ID})
	h.o.Log("access_history_hold_released", "marker", m.ID, "hold", m.Hold, "reason", why)
	return nil
}
