package server

import (
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func (s *Server) readSessionOperationItems(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	defer s.domain.RUnlock()
	found, answer, ok := s.self(w, r)
	if !ok {
		return
	}
	record := s.sessionOperations[r.PathValue("operation")]
	if record == nil || record.owner != (sessionOwner{answer.User(), answer.Workspace(), answer.Epoch()}) {
		failCode(w, commands.UnsupportedCommand, request)
		return
	}
	if record.state != "accepted" {
		if record.caller == found.handle {
			failCode(w, commands.AuthorityUnavailable, request)
		} else {
			failCode(w, commands.UnsupportedCommand, request)
		}
		return
	}
	if len(r.URL.RawQuery) > 8192 {
		invalidFields(w, request, "query")
		return
	}
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		invalidFields(w, request, "query")
		return
	}
	view := "results"
	if raw, exists := values["view"]; exists {
		if len(raw) != 1 || (raw[0] != "results" && raw[0] != "not_included") {
			invalidFields(w, request, "view")
			return
		}
		view = raw[0]
		delete(values, "view")
	}
	limit, after, valid := selectionPageQuery(values, s.cfg.Access.SessionQueryPage)
	if !valid {
		invalidFields(w, request, "query")
		return
	}
	cursor := s.operationCursor(found, s.sessionOperationVersions[found.owner], "operation:"+record.op.id+":"+view+":"+strconv.Itoa(limit), answer.NotAfter())
	start := 0
	if after != "" {
		prior, valid := s.validOperationCursor(after, cursor)
		if !valid {
			failRead(w, commands.RevisionConflict, request, "after")
			return
		}
		positions := record.resultPositions
		if view == "not_included" {
			positions = record.heldPositions
		}
		position, known := positions[prior.After]
		if !known {
			failRead(w, commands.RevisionConflict, request, "after")
			return
		}
		start = position + 1
		cursor = prior
	}
	count := len(record.op.rows)
	if view == "not_included" {
		count = len(record.op.notIncluded)
	}
	end := min(start+limit, count)
	rows := make([]any, 0, end-start)
	last := ""
	for position := start; position < end; position++ {
		if view == "not_included" {
			row := record.op.notIncluded[position]
			rows = append(rows, row)
			last = row.Handle
		} else {
			row := record.op.rows[position]
			row.Items = append([]selfItem{}, row.Items...)
			rows = append(rows, row)
			last = row.Handle
		}
	}
	next := ""
	if end < count {
		cursor.After = last
		next = s.sealSessionCursor(cursor)
	}
	writeJSON(w, http.StatusOK, request, struct {
		Operation  string `json:"operation"`
		View       string `json:"view"`
		Rows       []any  `json:"rows"`
		Count      int    `json:"count"`
		Next       string `json:"next"`
		Reference  string `json:"reference_at"`
		Generation string `json:"generation"`
	}{record.op.id, view, rows, count, next, cursor.Reference.UTC().Format(time.RFC3339Nano), strconv.FormatUint(cursor.Version, 10)})
}
