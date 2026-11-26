package server

import (
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func selectionPageQuery(values url.Values, defaultLimit int) (int, string, bool) {
	limit, after := defaultLimit, ""
	for key, raw := range values {
		if len(raw) != 1 || raw[0] == "" {
			return 0, "", false
		}
		switch key {
		case "limit":
			n, err := strconv.Atoi(raw[0])
			if err != nil || n < 1 || n > 100 {
				return 0, "", false
			}
			limit = n
		case "after":
			if len(raw[0]) > 4096 {
				return 0, "", false
			}
			after = raw[0]
		default:
			return 0, "", false
		}
	}
	return limit, after, true
}

func (s *Server) previewSessionSelection(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.Lock()
	defer s.domain.Unlock()
	found, _, ok := s.self(w, r)
	if !ok {
		return
	}
	now := s.clock()
	s.expireSessionRecords(now)
	record := s.sessionRecords[r.PathValue("reference")]
	if !s.ownsSessionRecord(record, found) {
		failCode(w, commands.UnsupportedCommand, request)
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
	limit, after, valid := selectionPageQuery(values, s.cfg.Access.SessionQueryPage)
	if !valid {
		invalidFields(w, request, "query")
		return
	}
	out := struct {
		selectionView
		Rows []queriedSession `json:"rows"`
		Next string           `json:"next"`
	}{selectionView: s.selectionView(record), Rows: []queriedSession{}}
	if record.state == "building" {
		if after != "" {
			failRead(w, commands.RevisionConflict, request, "after")
			return
		}
		writeJSON(w, http.StatusOK, request, out)
		return
	}
	query := "preview:" + record.ref + ":" + strconv.Itoa(limit)
	cursor := sessionCursor{Owner: record.owner, Caller: record.caller, Query: query, Version: record.version, Revision: record.digest, Reference: record.at, Expires: now.Add(time.Duration(s.cfg.Access.SessionQuerySeconds) * time.Second)}
	if record.op == nil {
		cursor.Expires = earlier(cursor.Expires, record.expires)
	}
	start := 0
	if after != "" {
		prior, valid := s.openSessionCursor(after)
		if !valid || prior.Owner != cursor.Owner || prior.Caller != cursor.Caller || prior.Query != cursor.Query || prior.Version != cursor.Version || prior.Revision != cursor.Revision || !now.Before(prior.Expires) {
			failRead(w, commands.RevisionConflict, request, "after")
			return
		}
		position, known := record.positions[prior.After]
		if !known {
			failRead(w, commands.RevisionConflict, request, "after")
			return
		}
		start = position + 1
		cursor = prior
	}
	end := min(start+limit, len(record.targets))
	for _, target := range record.targets[start:end] {
		out.Rows = append(out.Rows, target.Row)
	}
	if end < len(record.targets) {
		cursor.After = record.targets[end-1].Row.Handle
		out.Next = s.sealSessionCursor(cursor)
	}
	writeJSON(w, http.StatusOK, request, out)
}
