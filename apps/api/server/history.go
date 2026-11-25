package server

import (
	"net/url"
	"strconv"
	"strings"
)

const (
	keptPage    = 50
	keptPageMax = 100
)

type keptFacts struct {
	Total   int    `json:"total"`
	Next    string `json:"next"`
	Kept    int    `json:"kept"`
	Dropped int    `json:"dropped"`
}

type keptQuery struct {
	before uint64
	limit  int
	term   string
}

func parseKeptQuery(values url.Values) (keptQuery, bool) {
	out := keptQuery{limit: keptPage}
	for name, given := range values {
		valid := len(given) == 1 && given[0] != ""
		switch {
		case !valid:
		case name == "after":
			n, err := strconv.ParseUint(given[0], 10, 64)
			out.before, valid = n, err == nil && n > 0 && strconv.FormatUint(n, 10) == given[0]
		case name == "limit":
			n, err := strconv.Atoi(given[0])
			out.limit, valid = n, err == nil && n >= 1 && n <= keptPageMax && strconv.Itoa(n) == given[0]
		case name == "q":
			out.term, valid = parseSearch(given[0])
		default:
			valid = false
		}
		if !valid {
			return keptQuery{}, false
		}
	}
	return out, true
}

func (q keptQuery) matches(texts ...string) bool {
	if q.term == "" {
		return true
	}
	term := fold(q.term)
	for _, text := range texts {
		if text != "" && strings.Contains(fold(text), term) {
			return true
		}
	}
	return false
}

func keptCursor(number uint64) string { return strconv.FormatUint(number, 10) }

type keptWalk struct {
	q        keptQuery
	Matching int
	Next     string
	taken    int
	last     uint64
}

func (w *keptWalk) take(number uint64, matched bool) bool {
	if !matched {
		return false
	}
	w.Matching++
	if w.q.before != 0 && number >= w.q.before {
		return false
	}
	if w.taken == w.q.limit {
		w.Next = keptCursor(w.last)
		return false
	}
	w.taken, w.last = w.taken+1, number
	return true
}

func listKept[T any](entries []T, pending func(T) bool, number func(T) uint64, shown int) ([]T, int, string) {
	open := 0
	for _, e := range entries {
		if pending(e) {
			open++
		}
	}
	listed := []T{}
	closed, taken, last, newest := 0, 0, uint64(0), uint64(0)
	for index := len(entries) - 1; index >= 0; index-- {
		e := entries[index]
		if n := number(e); n > newest {
			newest = n
		}
		if pending(e) {
			listed = append(listed, e)
			continue
		}
		closed++
		if taken < shown-open {
			listed, taken, last = append(listed, e), taken+1, number(e)
		}
	}
	switch {
	case taken == closed:
		return listed, closed, ""
	case taken == 0:
		return listed, closed, keptCursor(newest + 1)
	}
	return listed, closed, keptCursor(last)
}

type logFacts struct {
	Total int    `json:"total"`
	Next  string `json:"next"`
}

func newestOf(total, shown int) (int, logFacts) {
	first := max(0, total-shown)
	facts := logFacts{Total: total}
	if first > 0 {
		facts.Next = keptCursor(uint64(first + 1))
	}
	return first, facts
}
