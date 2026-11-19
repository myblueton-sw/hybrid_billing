package accesshistory

import (
	"sort"
	"time"
)

func (h *History) Read(user string, q Query) Page {
	h.mu.RLock()
	defer h.mu.RUnlock()
	limit := q.Limit
	if limit < 1 || limit > MaxPage {
		limit = DefaultPage
	}
	now := h.now()
	cut := h.cutoff(now)
	page := Page{Days: h.state.Days, Source: h.o.DaysSource, EffectiveDays: h.effective, Events: []Event{}, Notes: []Note{}}
	if len(h.state.History) > 0 {
		page.PeriodSince = parseStamp(h.state.History[len(h.state.History)-1].Since)
	}
	if page.EffectiveDays == 0 || page.EffectiveDays > page.Days {
		page.EffectiveDays = page.Days
	}
	floor := h.floors[user]
	var ws, refused []entry
	if p := h.persons[user]; p != nil {
		if s := p.seqs["w:"+q.Workspace]; s != nil && q.Workspace != "" {
			ws = s.entries
		}
		if s := p.seqs[""]; s != nil {
			refused = s.entries
		}
	}
	visible := func(e entry) bool { return e.seq > floor && e.at >= cut }
	inRange := func(e entry) bool {
		at := fromMillis(e.at)
		return (q.From.IsZero() || !at.Before(q.From)) && (q.To.IsZero() || !at.After(q.To))
	}
	oldest := int64(0)
	for _, list := range [][]entry{ws, refused} {
		for _, e := range list {
			if visible(e) && (oldest == 0 || e.at < oldest) {
				oldest = e.at
			}
		}
	}
	if oldest > 0 {
		page.AvailableFrom = fromMillis(oldest)
	}
	pick := func(list []entry, cursor uint64) []entry {
		out := []entry{}
		for i := len(list) - 1; i >= 0; i-- {
			e := list[i]
			if (cursor == 0 || e.n < cursor) && visible(e) && inRange(e) {
				out = append(out, e)
			}
		}
		return out
	}
	a, b := pick(ws, q.CursorW), pick(refused, q.CursorR)
	merged := append(append([]entry{}, a...), b...)
	sort.SliceStable(merged, func(i, j int) bool { return merged[i].seq > merged[j].seq })
	nextW, nextR := q.CursorW, q.CursorR
	if len(merged) > limit {
		page.More = true
	}
	for i, e := range merged {
		if i == limit {
			break
		}
		page.Events = append(page.Events, Event{At: fromMillis(e.at), Kind: e.kind, Reason: e.reason, Workspace: e.workspace, Handle: e.handle,
			Browser: e.browser, OS: e.os, Count: e.count})
		if e.kind == KindSignInRefused || e.kind == KindSignInRefusedMore {
			nextR = e.n
		} else {
			nextW = e.n
		}
	}
	if page.More {
		page.NextW, page.NextR = cursorOrEnd(nextW, a), cursorOrEnd(nextR, b)
	}
	page.Recording = h.recording()
	page.Notes = h.notesFor(user, q.Workspace, cut, now)
	return page
}

func cursorOrEnd(position uint64, list []entry) uint64 {
	if position == 0 && len(list) > 0 {
		return list[0].n + 1
	}
	return position
}

func (h *History) recording() Recording {
	h.offerMu.Lock()
	defer h.offerMu.Unlock()
	if h.stopped {
		return Recording{State: "stopped", Since: fromMillis(h.stopSince), Reason: h.stopReason}
	}
	return Recording{State: "recording"}
}

func (h *History) notesFor(user, workspace string, cut int64, now time.Time) []Note {
	out := []Note{}
	counted := func(kind string, from, to int64, n int64, known bool) {
		note := Note{Kind: kind}
		if from > 0 {
			note.From = fromMillis(from)
		}
		if to > 0 {
			note.To = fromMillis(to)
		}
		if known {
			c := n
			note.Count = &c
		}
		out = append(out, note)
	}
	boundary := h.floors[user]
	for _, n := range h.notes {
		if n.to != 0 && n.to < cut {
			continue
		}
		switch n.kind {
		case "dropped", "damaged", "early_deletion":
			exact, unknown := attributionCount(n.per, user, workspace, boundary)
			if exact > 0 {
				counted(n.kind, max(n.from, h.floorAt[user]), n.to, exact, true)
			} else if unknown || n.unattributed > 0 {
				counted(n.kind, 0, 0, 0, false)
			}
		default:
			counted(n.kind, n.from, n.to, 0, false)
		}
	}
	h.offerMu.Lock()
	live, unknown := attributionCount(h.dropped.encoded(), user, workspace, boundary)
	all := h.droppedAll
	unattributed := len(h.dropped) == 0
	from, to := h.dropFrom, h.dropTo
	h.offerMu.Unlock()
	if live > 0 {
		counted("dropped", max(from, h.floorAt[user]), to, live, true)
	} else if unknown || (all > 0 && unattributed) {
		counted("dropped", 0, 0, 0, false)
	}
	held, _ := attributionCount(h.heldCount.encoded(), user, workspace, boundary)
	if held > 0 {
		counted("held", 0, millis(now), held, true)
	}
	if h.deferred != "" {
		counted("purge_deferred", 0, millis(now), 0, false)
	}
	if h.overdue > 0 {
		counted("deletion_overdue", h.overdue, millis(now), 0, false)
	}
	if h.futureFile {
		counted("future_file", 0, millis(now), 0, false)
	}
	return out
}
