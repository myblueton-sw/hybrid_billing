package server

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

const (
	listHistory   = "history"
	listOpenCalls = "open_calls"
	listUnknown   = "unknown"
	listPending   = "pending"
)

type recoveryRowsPage struct {
	Revision     string `json:"revision"`
	ServerTime   string `json:"server_time"`
	Installation string `json:"installation"`
	List         string `json:"list"`
	Order        string `json:"order"`
	RowsCanMove  bool   `json:"rows_can_move"`
	Rows         any    `json:"rows"`
	Next         string `json:"next"`
	Total        int    `json:"total"`
}

func nextOf(listed, total, passed int) string {
	if passed+listed < total {
		return strconv.Itoa(passed + listed)
	}
	return ""
}

func historyRows(f recoveryFacts, before, shown int) ([]recoveryHistoryRow, string) {
	out, next := []recoveryHistoryRow{}, ""
	for index := len(f.history) - 1; index >= 0; index-- {
		if before != 0 && index+1 >= before {
			continue
		}
		if len(out) == shown {
			next = strconv.Itoa(index + 2)
			break
		}
		h := f.history[index]
		out = append(out, recoveryHistoryRow{Recovery: h.Recovery, ManifestDigest: h.ManifestDigest, Generation: h.Generation, Outcome: h.Outcome, Lost: h.Lost})
	}
	return out, next
}

func openCallRows(f recoveryFacts, workspace string, passed, shown int) ([]recoveryCallRow, int) {
	out, total := []recoveryCallRow{}, 0
	for _, c := range f.view.OpenCalls {
		if c.Target.Workspace != workspace {
			continue
		}
		if total++; total > passed && len(out) < shown {
			out = append(out, recoveryCallRow{ID: c.ID, Writer: c.Writer, Deadline: stamp(c.Deadline), Key: c.Target.Key, Series: c.Target.Series, Number: c.Target.Number})
		}
	}
	return out, total
}

func (s *Server) unknownRows(f recoveryFacts, workspace string, passed, shown int) ([]recoveryTargetRow, int, error) {
	out, total, errs, id := []recoveryTargetRow{}, 0, []error{}, f.install.id
	for _, t := range f.view.Unknown {
		if t.Workspace != workspace {
			continue
		}
		if total++; total <= passed || len(out) >= shown {
			continue
		}
		target := recoveryTargetRow{Key: t.Key, Series: t.Series, Number: t.Number}
		errs = append(errs, s.recoveryCall("fenced", func() (err error) { target.Fenced, err = s.recovery.Fenced(id, workspace, t.Key); return }))
		if t.Series != "" {
			var high int64
			errs = append(errs, s.recoveryCall("high_water", func() (err error) { high, err = s.recovery.HighWater(id, workspace, t.Series); return }))
			target.HighWater = &high
		}
		out = append(out, target)
	}
	return out, total, errors.Join(errs...)
}

func pendingRows(f recoveryFacts, passed, shown int) []recoveryEventRow {
	out := []recoveryEventRow{}
	for index, e := range f.view.Pending {
		if index >= passed && len(out) < shown {
			out = append(out, recoveryEventRow{Sequence: e.Sequence, Kind: e.Event.Kind, Domain: e.Event.Domain, At: stamp(e.At)})
		}
	}
	return out
}

func (s *Server) listRecoveryRows(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	defer s.domain.RUnlock()
	_, answer, ok := s.admitted(w, r)
	if !ok {
		return
	}
	values := r.URL.Query()
	after, limit, valid := 0, recoveryShown, true
	for name, given := range values {
		switch {
		case len(given) != 1 || given[0] == "":
			valid = false
		case name == "after":
			n, err := strconv.Atoi(given[0])
			after, valid = n, valid && err == nil && n >= 1 && strconv.Itoa(n) == given[0]
		case name == "limit":
			n, err := strconv.Atoi(given[0])
			limit, valid = n, valid && err == nil && n >= 1 && n <= recoveryShown && strconv.Itoa(n) == given[0]
		case name != "installation" && name != "list":
			valid = false
		}
	}
	list := values.Get("list")
	valid = valid && (list == listHistory || list == listOpenCalls || list == listUnknown || list == listPending)
	var install *recoveryInstall
	for _, i := range s.visibleInstalls(answer.Workspace()) {
		if i.id == values.Get("installation") {
			install = i
		}
	}
	if !valid || install == nil {
		failCode(w, commands.InvalidRequest, request)
		return
	}
	if s.faulty("recovery-read") {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	f, err := s.factsOf(install)
	if err != nil {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	page := recoveryRowsPage{Revision: s.Revision(), ServerTime: stamp(s.clock()), Installation: install.id, List: list, Order: requirementsOrder, RowsCanMove: list != listHistory}
	switch {
	case list == listHistory:
		page.Order, page.Total = closedPlansOrder, len(f.history)
		page.Rows, page.Next = historyRows(f, after, limit)
	case f.view == nil:
		page.Rows = []any{}
	case list == listOpenCalls:
		rows, total := openCallRows(f, answer.Workspace(), after, limit)
		page.Rows, page.Total, page.Next = rows, total, nextOf(len(rows), total, after)
	case list == listUnknown:
		rows, total, err := s.unknownRows(f, answer.Workspace(), after, limit)
		if err != nil {
			failCode(w, commands.DependencyUnavailable, request)
			return
		}
		page.Rows, page.Total, page.Next = rows, total, nextOf(len(rows), total, after)
	default:
		rows := pendingRows(f, after, limit)
		page.Rows, page.Total = rows, len(f.view.Pending)
		page.Next = nextOf(len(rows), page.Total, after)
	}
	writeJSON(w, http.StatusOK, request, page)
}
