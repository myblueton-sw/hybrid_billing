package accesshistory

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type runSpan struct {
	run     string
	start   uint64
	startAt int64
	stopped bool
}

type scanState struct {
	runs    []runSpan
	stops   map[string]bool
	records []struct {
		seq uint64
		at  int64
	}
}

func (h *History) cutoff(now time.Time) int64 {
	return millis(now.Add(-time.Duration(h.state.Days) * 24 * time.Hour))
}

func (h *History) scan(now time.Time) (scanState, error) {
	st := scanState{stops: map[string]bool{}}
	names, err := h.dayNames()
	if err != nil {
		return st, fmt.Errorf("%w: %s: %v", ErrConfig, h.dir, err)
	}
	cut := h.cutoff(now)
	var maxSeq uint64
	for _, name := range names {
		day, _ := dayOf(name)
		if day.After(now.Add(24 * time.Hour)) {
			h.futureFile = true
		}
		path := filepath.Join(h.dir, name)
		f, err := h.sys.openFile(path, false)
		if err != nil {
			return st, fmt.Errorf("%w: %s: %v", ErrConfig, name, err)
		}
		raw, err := readAll(f)
		if err != nil {
			f.Close()
			return st, fmt.Errorf("%w: %s: %v", ErrConfig, name, err)
		}
		if n := len(raw); n > 0 && raw[n-1] != '\n' {
			keep := int64(strings.LastIndexByte(string(raw), '\n') + 1)
			if err := f.Truncate(keep); err != nil {
				f.Close()
				return st, fmt.Errorf("%w: %s: %v", ErrConfig, name, err)
			}
			if err := f.Sync(); err != nil {
				f.Close()
				return st, fmt.Errorf("%w: %s: %v", ErrConfig, name, err)
			}
			raw = raw[:keep]
			h.pendingRecs = append(h.pendingRecs, Record{Kind: sysRecoveredTail, From: millis(day), To: millis(day.Add(24*time.Hour - time.Millisecond))})
		}
		f.Close()
		h.sizes[name] = int64(len(raw))
		h.total += int64(len(raw))
		for _, line := range splitLines(raw) {
			var r Record
			if decodeLine(line, &r) != nil || !r.valid() {
				per := map[string]int64{}
				if source, ok := bestEffortRecord(line); ok {
					counts := lossCounts{}
					counts.add(source, 1)
					per = counts.encoded()
				}
				h.addLossPer("damaged", millis(day), millis(day.Add(24*time.Hour-time.Millisecond)), per, 1)
				continue
			}
			maxSeq = max(maxSeq, r.Seq)
			h.lastAt = max(h.lastAt, r.At)
			st.records = append(st.records, struct {
				seq uint64
				at  int64
			}{r.Seq, r.At})
			h.loadRecord(r, cut, &st)
		}
	}
	entries, err := os.ReadDir(h.dir)
	if err != nil {
		return st, fmt.Errorf("%w: %s: %v", ErrConfig, h.dir, err)
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), heldDamagedSuffix) {
			continue
		}
		f, err := h.sys.openFile(filepath.Join(h.dir, e.Name()), false)
		if err != nil {
			return st, fmt.Errorf("%w: %s: %v", ErrConfig, e.Name(), err)
		}
		info, err := f.Stat()
		f.Close()
		if err != nil {
			return st, fmt.Errorf("%w: %s: %v", ErrConfig, e.Name(), err)
		}
		h.sizes[e.Name()] = info.Size()
		h.total += info.Size()
	}
	for _, floor := range h.floors {
		maxSeq = max(maxSeq, floor)
	}
	h.seq = max(maxSeq, h.state.Seq)
	if last := parseStamp(h.state.LastAt); !last.IsZero() {
		h.lastAt = max(h.lastAt, millis(last))
	}
	for _, p := range h.persons {
		for _, s := range p.seqs {
			sort.Slice(s.entries, func(a, b int) bool { return s.entries[a].seq < s.entries[b].seq })
		}
	}
	return st, nil
}

func (h *History) loadRecord(r Record, cut int64, st *scanState) {
	switch r.Kind {
	case sysRunStarted:
		st.runs = append(st.runs, runSpan{run: r.Run, start: r.Seq, startAt: r.At})
		return
	case sysRunStopped:
		st.stops[r.Run] = true
		return
	case sysDropped:
		h.addLossPer("dropped", r.From, r.To, r.Per, r.Count)
		return
	case sysEarlyDeletion:
		h.addLossPer("early_deletion", r.From, r.To, r.Per, r.Count)
		return
	case sysRecoveredTail:
		h.addLoss("recovered_tail", r.From, r.To, "", 0)
		return
	}
	if !Personal(r.Kind) {
		return
	}
	if !recordVisible(r, h.floors[r.User]) {
		return
	}
	key := sequenceKey(r)
	p := h.personOf(r.User)
	s := p.seqOf(key)
	s.next = max(s.next, r.N)
	if r.At < cut {
		if h.heldByMarker(r) {
			h.heldCount.add(r, 1)
		}
		return
	}
	s.entries = append(s.entries, h.entryOf(r))
}

func sequenceKey(r Record) string {
	if r.Kind == KindSignInRefused || r.Kind == KindSignInRefusedMore {
		return ""
	}
	return "w:" + r.Workspace
}

func (h *History) personOf(user string) *person {
	p := h.persons[user]
	if p == nil {
		p = &person{seqs: map[string]*sequence{}}
		h.persons[user] = p
	}
	return p
}

func (p *person) seqOf(key string) *sequence {
	s := p.seqs[key]
	if s == nil {
		s = &sequence{}
		p.seqs[key] = s
	}
	return s
}

func (h *History) intern(ws string) string {
	if ws == "" {
		return ""
	}
	if v, ok := h.workspaces[ws]; ok {
		return v
	}
	h.workspaces[ws] = ws
	return ws
}

func (h *History) entryOf(r Record) entry {
	return entry{seq: r.Seq, n: r.N, at: r.At, kind: r.Kind, reason: r.Reason, workspace: h.intern(r.Workspace), handle: r.Handle, browser: r.Browser, os: r.OS, count: r.Count}
}

func (h *History) addLoss(kind string, from, to int64, user string, n int64) {
	if user != "" {
		h.addLossPer(kind, from, to, map[string]int64{user: n}, n)
		return
	}
	h.addLossPer(kind, from, to, nil, n)
}

func (h *History) addLossPer(kind string, from, to int64, per map[string]int64, total int64) {
	for i := range h.notes {
		n := &h.notes[i]
		if n.kind == kind && n.from == from && n.to == to {
			n.merge(per, total)
			return
		}
	}
	n := lossNote{kind: kind, from: from, to: to, per: map[string]int64{}}
	n.merge(per, total)
	h.notes = append(h.notes, n)
}

func (n *lossNote) merge(per map[string]int64, total int64) {
	var attributed int64
	for u, c := range per {
		if c > 0 {
			n.per[u] += c
			attributed += c
		}
	}
	if total > attributed {
		n.unattributed += total - attributed
	}
}

func (h *History) unclean(st scanState, now time.Time) {
	sort.Slice(st.runs, func(a, b int) bool { return st.runs[a].start < st.runs[b].start })
	for i, run := range st.runs {
		if st.stops[run.run] {
			continue
		}
		end := uint64(1<<63 - 1)
		to := millis(now)
		if i+1 < len(st.runs) {
			end = st.runs[i+1].start
			to = st.runs[i+1].startAt
		}
		from := run.startAt
		for _, r := range st.records {
			if r.seq >= run.start && r.seq < end {
				from = max(from, r.at)
			}
		}
		h.addLoss("unclean_stop", from, to, "", 0)
	}
}

func (h *History) checkClockGap(now time.Time) error {
	if h.state.Deferral != nil {
		h.deferred = "clock_jump"
		if h.o.AcceptClock == "" {
			h.o.Log("access_history_purge_deferred", "reason", "clock_jump", "reference", h.state.Deferral.Reference, "logged", h.state.Deferral.Logged)
			return nil
		}
		want, err := time.Parse(time.RFC3339, h.o.AcceptClock)
		logged := parseStamp(h.state.Deferral.Logged)
		if err != nil || !want.UTC().Truncate(time.Minute).Equal(logged.Truncate(time.Minute)) {
			return fmt.Errorf("%w: accept-access-history-clock must equal the logged server time %s", ErrConfig, logged.Truncate(time.Minute).Format(time.RFC3339))
		}
		h.state.Deferral = nil
		h.deferred = ""
		h.o.Log("access_history_clock_accepted", "logged", logged.Format(time.RFC3339))
		return nil
	}
	if h.o.AcceptClock != "" {
		return fmt.Errorf("%w: accept-access-history-clock is given but no deletion is deferred", ErrConfig)
	}
	if h.lastAt > 0 && millis(now)-h.lastAt > h.o.ClockGap.Milliseconds() {
		h.state.Deferral = &deferral{Reference: stamp(fromMillis(h.lastAt)), Logged: stamp(now)}
		h.deferred = "clock_jump"
		h.pendingRecs = append(h.pendingRecs, Record{Kind: sysPurgeDeferred, From: h.lastAt, To: millis(now)})
		h.o.Log("access_history_purge_deferred", "reason", "clock_jump", "reference", h.state.Deferral.Reference, "logged", h.state.Deferral.Logged,
			"accept_with", "-accept-access-history-clock="+now.Truncate(time.Minute).Format(time.RFC3339))
	}
	return nil
}
