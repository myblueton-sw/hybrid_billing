package server

import (
	"sort"

	"github.com/myblueton-sw/hybrid_billing/modules/relationships"
)

type span struct {
	target string
	start  relationships.Day
	end    relationships.Day
	open   bool
}

func spanOf(e relationships.Entry) span {
	end, bounded := e.Interval.End()
	return span{target: e.Target, start: e.Interval.Start(), end: end, open: !bounded}
}

func (s span) entry() relationships.Entry {
	if s.open {
		return relationships.Entry{Target: s.target, Interval: relationships.From(s.start)}
	}
	return relationships.Entry{Target: s.target, Interval: relationships.Between(s.start, s.end)}
}

func after(a, b relationships.Day) bool { return b.Before(a) }

func spanCovers(s span, day relationships.Day) bool {
	return !day.Before(s.start) && (s.open || day.Before(s.end))
}

func cut(s span, from relationships.Day) (span, bool) {
	if !s.start.Before(from) {
		return s, false
	}
	if !s.open && !after(s.end, from) {
		return s, true
	}
	return span{target: s.target, start: s.start, end: from}, true
}

func merge(spans []span) []span {
	sort.Slice(spans, func(a, b int) bool {
		if spans[a].target != spans[b].target {
			return spans[a].target < spans[b].target
		}
		return spans[a].start.Before(spans[b].start)
	})
	out := []span{}
	for _, s := range spans {
		if n := len(out); n > 0 && out[n-1].target == s.target && (out[n-1].open || !out[n-1].end.Before(s.start)) {
			last := &out[n-1]
			if s.open {
				last.open = true
			} else if !last.open && after(s.end, last.end) {
				last.end = s.end
			}
			continue
		}
		out = append(out, s)
	}
	sort.SliceStable(out, func(a, b int) bool { return out[a].start.Before(out[b].start) })
	return out
}

func buildTimeline(entries []relationships.Entry, action, target string, from relationships.Day) []relationships.Entry {
	spans := []span{}
	for _, e := range entries {
		s := spanOf(e)
		switch action {
		case "set", "end":
			if kept, ok := cut(s, from); ok {
				spans = append(spans, kept)
			}
		case "leave":
			if s.target != target {
				spans = append(spans, s)
			} else if kept, ok := cut(s, from); ok {
				spans = append(spans, kept)
			}
		default:
			spans = append(spans, s)
		}
	}
	switch action {
	case "set":
		spans = append(spans, span{target: target, start: from, open: true})
	case "join":
		spans = append(spans, span{target: target, start: from, open: true})
	}
	out := []relationships.Entry{}
	for _, s := range merge(spans) {
		out = append(out, s.entry())
	}
	return out
}
