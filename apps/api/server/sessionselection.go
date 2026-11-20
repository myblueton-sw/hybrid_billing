package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

type selectionView struct {
	Kind       string       `json:"kind"`
	Selection  string       `json:"selection"`
	Operation  string       `json:"operation,omitempty"`
	State      string       `json:"state"`
	Reference  string       `json:"reference_at"`
	Expires    string       `json:"expires_at"`
	Generation string       `json:"generation"`
	Observed   int          `json:"observed"`
	Count      *int         `json:"count"`
	Digest     string       `json:"digest"`
	After      string       `json:"after"`
	Settings   []settingRow `json:"settings"`
}

func (s *Server) selectionToken(record *sessionRecord) string {
	return s.sealSessionCursor(sessionCursor{Owner: record.owner, Caller: record.caller, Query: "selection:" + record.ref, Version: record.version, Revision: record.revision, Reference: record.at, Expires: record.expires, After: record.after, Matched: len(record.targets)})
}

func (s *Server) selectionView(record *sessionRecord) selectionView {
	out := selectionView{Kind: record.kind, Selection: record.ref, State: record.state, Reference: record.at.UTC().Format(time.RFC3339Nano), Expires: record.expires.UTC().Format(time.RFC3339Nano), Generation: record.generation, Observed: len(record.targets), Digest: record.digest, Settings: s.rowsOf(append(SessionQuerySettingList(), sessionSelectionSettings...))}
	if record.op != nil {
		out.Operation = record.op.id
	}
	if record.state == "sealed" || record.state == "accepted" || record.state == "applying" {
		count := len(record.targets)
		out.Count = &count
	} else if record.state == "building" {
		out.After = s.selectionToken(record)
	}
	return out
}

func selectionDigest(record *sessionRecord) string {
	handles := make([]string, 0, len(record.targets))
	for _, target := range record.targets {
		handles = append(handles, target.Row.Handle+":"+hex.EncodeToString(target.verifier[:]))
	}
	base := endDigest(record.kind, record.ref+":"+record.generation, handles)
	body, _ := json.Marshal(struct {
		Base   string
		Owner  sessionOwner
		Caller string
	}{base, record.owner, record.caller})
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func (s *Server) selectionLimit(w http.ResponseWriter, request string) {
	failModule(w, commands.RateLimited, http.StatusTooManyRequests, true, commands.NextRetry, request)
}

func (s *Server) prepareSessionSelection(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.Lock()
	defer s.domain.Unlock()
	found, answer, ok := s.self(w, r)
	if !ok {
		return
	}
	if r.URL.RawQuery != "" {
		invalidFields(w, request, "query")
		return
	}
	values, err := sessionObject(r.Body, "selection", "after", "session")
	if err != nil {
		if tooLarge(err) {
			fail(w, http.StatusRequestEntityTooLarge, CodeBodyTooLarge, request)
			return
		}
		invalidFields(w, request, "selection")
		return
	}
	ref, after := "", ""
	if raw, exists := values["selection"]; exists {
		if json.Unmarshal(raw, &ref) != nil || ref == "" || len(ref) > 64 {
			invalidFields(w, request, "selection")
			return
		}
	}
	if raw, exists := values["after"]; exists {
		if json.Unmarshal(raw, &after) != nil || after == "" || len(after) > 4096 {
			invalidFields(w, request, "after")
			return
		}
	}
	if (ref == "") != (after == "") {
		invalidFields(w, request, "after")
		return
	}
	kind := kindOthers
	var target session
	if raw, exists := values["session"]; exists {
		var handle string
		if len(values) != 1 || json.Unmarshal(raw, &handle) != nil || !isHex(handle, 16) || handle == found.handle {
			invalidFields(w, request, "session")
			return
		}
		var known bool
		target, known = s.sessions.indexedSession(found.owner, handle)
		if !known {
			failCode(w, commands.UnsupportedCommand, request)
			return
		}
		kind = kindOne
	}
	now := s.clock()
	s.expireSessionRecords(now)
	index, expires := s.sessionBasis(found, answer)
	var record *sessionRecord
	if ref != "" {
		record = s.sessionRecords[ref]
		if !s.ownsSessionRecord(record, found) {
			failCode(w, commands.UnsupportedCommand, request)
			return
		}
		cursor, valid := s.openSessionCursor(after)
		if !valid || cursor.Owner != record.owner || cursor.Caller != record.caller || cursor.Query != "selection:"+record.ref || cursor.Version != record.version || cursor.Revision != record.revision || !cursor.Expires.Equal(record.expires) {
			failRead(w, commands.RevisionConflict, request, "after")
			return
		}
		if record.state == "building" && (record.version != index.Version || record.revision != s.Revision() || !now.Before(record.expires)) {
			s.forgetSessionRecord(record)
			failRead(w, commands.RevisionConflict, request, "selection")
			return
		}
		if record.state != "building" || after == record.previous {
			writeJSON(w, http.StatusOK, request, s.selectionView(record))
			return
		}
		if after != s.selectionToken(record) {
			failRead(w, commands.RevisionConflict, request, "after")
			return
		}
	} else {
		pending := 0
		for _, candidate := range s.sessionRecords {
			if s.ownsSessionRecord(candidate, found) && candidate.op == nil && !candidate.legacy {
				pending++
				if candidate.kind == kind && candidate.version == index.Version && candidate.revision == s.Revision() && (kind != kindOne || len(candidate.targets) == 1 && candidate.targets[0].Row.Handle == target.handle && candidate.targets[0].verifier == target.verifier) {
					writeJSON(w, http.StatusOK, request, s.selectionView(candidate))
					return
				}
			}
		}
		if pending >= s.cfg.Access.SessionSelectionPending {
			s.selectionLimit(w, request)
			return
		}
		record = s.newSessionRecord(found, now, expires, index.Version, s.Revision())
		if record == nil {
			s.selectionLimit(w, request)
			return
		}
		record.kind = kind
		record.generation = strconv.FormatUint(index.Version, 10)
	}
	if record.version != index.Version || record.revision != s.Revision() || !now.Before(record.expires) {
		s.forgetSessionRecord(record)
		failRead(w, commands.RevisionConflict, request, "selection")
		return
	}
	if record.kind == kindOne {
		if !s.reserveSessionRecord(record, 1) {
			s.forgetSessionRecord(record)
			s.selectionLimit(w, request)
			return
		}
		record.positions[target.handle] = 0
		record.targets = append(record.targets, frozenSession{Row: s.sessionView(target, found.verifier), verifier: target.verifier})
		record.state = "sealed"
		record.digest = selectionDigest(record)
		writeJSON(w, http.StatusOK, request, s.selectionView(record))
		return
	}
	batch, more, valid := s.sessions.indexed(found.owner, index.Version, record.after, s.cfg.Access.SessionQueryScan)
	if !valid {
		s.forgetSessionRecord(record)
		failRead(w, commands.RevisionConflict, request, "selection")
		return
	}
	required := 0
	for _, candidate := range batch {
		if candidate.verifier != found.verifier {
			required++
		}
	}
	if len(record.targets)+required > s.cfg.Access.SessionSelectionTargets || !s.reserveSessionRecord(record, required) {
		s.forgetSessionRecord(record)
		s.selectionLimit(w, request)
		return
	}
	for _, candidate := range batch {
		record.after = candidate.handle
		if candidate.verifier == found.verifier {
			continue
		}
		record.positions[candidate.handle] = len(record.targets)
		record.targets = append(record.targets, frozenSession{Row: s.sessionView(candidate, found.verifier), verifier: candidate.verifier})
	}
	record.previous = after
	if !more {
		record.state = "sealed"
		record.digest = selectionDigest(record)
	}
	writeJSON(w, http.StatusOK, request, s.selectionView(record))
}
