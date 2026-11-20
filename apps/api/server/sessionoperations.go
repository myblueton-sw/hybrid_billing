package server

import (
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func (s *Server) operationChanged(owner sessionOwner, sessionsLocked bool) {
	s.sessionOperationSequence++
	s.sessionOperationVersions[owner] = s.sessionOperationSequence
	if sessionsLocked {
		s.sessions.operationChangedLocked(owner)
	} else {
		s.sessions.operationChanged(owner)
	}
}

func (s *Server) publishSessionOperation(record *sessionRecord) {
	if record.indexed {
		return
	}
	record.indexed = true
	s.sessionOperationRecords[record.owner] = append([]*sessionRecord{record}, s.sessionOperationRecords[record.owner]...)
	s.publishOtherSessionOperation(record)
	s.operationChanged(record.owner, false)
}

func (s *Server) removeSessionOperationIndex(record *sessionRecord, sessionsLocked bool) {
	if !record.indexed {
		return
	}
	list := s.sessionOperationRecords[record.owner]
	kept := list[:0]
	for _, candidate := range list {
		if candidate != record {
			kept = append(kept, candidate)
		}
	}
	clear(list[len(kept):])
	record.indexed = false
	s.forgetOtherSessionOperation(record)
	s.operationChanged(record.owner, sessionsLocked)
	if len(kept) == 0 {
		delete(s.sessionOperationRecords, record.owner)
		delete(s.sessionOperationVersions, record.owner)
	} else {
		s.sessionOperationRecords[record.owner] = kept
	}
}

func (s *Server) readSessionOperation(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	defer s.domain.RUnlock()
	found, answer, ok := s.self(w, r)
	if !ok {
		return
	}
	if found.owner != (sessionOwner{answer.User(), answer.Workspace(), answer.Epoch()}) {
		failCode(w, commands.Unauthenticated, request)
		return
	}
	record := s.sessionOperations[r.PathValue("operation")]
	if record == nil || record.owner != (sessionOwner{answer.User(), answer.Workspace(), answer.Epoch()}) || record.state != "accepted" && record.caller != found.handle {
		failCode(w, commands.UnsupportedCommand, request)
		return
	}
	if r.URL.RawQuery != "" {
		invalidFields(w, request, "query")
		return
	}
	writeJSON(w, http.StatusOK, request, operationSummary(record, false))
}

func (s *Server) operationCursor(found session, version uint64, query string, until time.Time) sessionCursor {
	now := s.clock()
	return sessionCursor{Owner: found.owner, Caller: found.handle, Query: query, Version: version, Reference: now, Expires: earlier(until, now.Add(time.Duration(s.cfg.Access.SessionQuerySeconds)*time.Second))}
}

func (s *Server) validOperationCursor(raw string, expected sessionCursor) (sessionCursor, bool) {
	cursor, valid := s.openSessionCursor(raw)
	if !valid || cursor.Owner != expected.Owner || cursor.Caller != expected.Caller || cursor.Query != expected.Query || cursor.Version != expected.Version || !s.clock().Before(cursor.Expires) {
		return cursor, false
	}
	return cursor, true
}

func (s *Server) listSessionOperations(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	defer s.domain.RUnlock()
	found, answer, ok := s.self(w, r)
	if !ok {
		return
	}
	if found.owner != (sessionOwner{answer.User(), answer.Workspace(), answer.Epoch()}) {
		failCode(w, commands.Unauthenticated, request)
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
	version := s.sessionOperationVersions[found.owner]
	cursor := s.operationCursor(found, version, "operations:"+strconv.Itoa(limit), answer.NotAfter())
	list := s.sessionOperationRecords[found.owner]
	start := 0
	if after != "" {
		prior, valid := s.validOperationCursor(after, cursor)
		record := s.sessionOperations[prior.After]
		if !valid || record == nil || record.owner != found.owner || !record.indexed {
			failRead(w, commands.RevisionConflict, request, "after")
			return
		}
		cursor = prior
		start = sort.Search(len(list), func(i int) bool { return list[i].op.sequence < record.op.sequence })
	}
	end := min(start+limit, len(list))
	rows := make([]sessionOperationSummary, 0, end-start)
	for _, record := range list[start:end] {
		rows = append(rows, operationSummary(record, false))
	}
	next := ""
	if end < len(list) {
		cursor.After = list[end-1].op.id
		next = s.sealSessionCursor(cursor)
	}
	writeJSON(w, http.StatusOK, request, struct {
		Rows       []sessionOperationSummary `json:"rows"`
		Count      int                       `json:"accepted_count"`
		Next       string                    `json:"next"`
		Reference  string                    `json:"reference_at"`
		Generation string                    `json:"generation"`
	}{rows, len(list), next, cursor.Reference.UTC().Format(time.RFC3339Nano), strconv.FormatUint(version, 10)})
}
