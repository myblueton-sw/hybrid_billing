package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

type sessionOperationSummary struct {
	Operation         string      `json:"operation"`
	Selection         string      `json:"selection"`
	Kind              string      `json:"kind"`
	State             string      `json:"state"`
	Recorded          int         `json:"recorded_count"`
	Ended             *int        `json:"ended_count"`
	Gone              *int        `json:"gone_count"`
	NotIncluded       *int        `json:"not_included_count"`
	BlockedAt         *string     `json:"blocked_at"`
	ListGeneration    string      `json:"list_generation"`
	CurrentGeneration *string     `json:"current_generation"`
	GenerationChanged *bool       `json:"generation_changed"`
	Digest            string      `json:"digest"`
	Cleanup           *selfCounts `json:"cleanup"`
	Replay            bool        `json:"replay"`
}

func operationSummary(record *sessionRecord, replay bool) sessionOperationSummary {
	op := record.op
	out := sessionOperationSummary{Operation: op.id, Selection: record.ref, Kind: op.kind, State: "unconfirmed", Recorded: len(record.targets), ListGeneration: op.listed, Digest: op.digest, Replay: replay}
	if record.state != "accepted" {
		return out
	}
	ended, gone, held := op.endedCount, op.goneCount, len(op.notIncluded)
	cleanup := op.cleanup
	out.State = stateComplete
	if cleanup.Pending > 0 {
		out.State = stateRunning
	} else if cleanup.Failed > 0 {
		out.State = stateIncomplete
	}
	blocked := stamp(op.blockedAt)
	changed := op.kind == kindOthers && op.before != op.listed
	out.Ended, out.Gone, out.NotIncluded = &ended, &gone, &held
	out.BlockedAt, out.CurrentGeneration, out.GenerationChanged, out.Cleanup = &blocked, &op.after, &changed, &cleanup
	return out
}

func (s *Server) startSessionOperation(record *sessionRecord, kind string) *selfEnd {
	s.selfSeq++
	op := &selfEnd{id: "session-end-" + strconv.FormatUint(s.selfSeq, 10), kind: kind, blockedAt: s.clock(), listed: record.generation, digest: record.digest, record: record, sequence: s.selfSeq}
	record.op = op
	return op
}

func (s *Server) applySessionRecord(found session, record *sessionRecord, kind string) (manifestResult, int) {
	if s.selfEndHook != nil {
		s.selfEndHook()
	}
	op := s.startSessionOperation(record, kind)
	rejected := http.StatusTooManyRequests
	result, admitted := s.takeSessionRecord(found, record, func(units int) bool {
		if !record.legacy && !s.clock().Before(record.expires) {
			rejected = http.StatusConflict
			return false
		}
		if !s.reserveSessionRecord(record, units) {
			return false
		}
		record.state = "applying"
		s.sessionOperations[op.id] = record
		return true
	})
	if !admitted {
		record.op = nil
		return result, http.StatusUnauthorized
	}
	if !result.ok {
		record.op = nil
		return result, rejected
	}
	record.state = "applying"
	s.sessionOperations[op.id] = record
	if !record.legacy {
		result.before, result.after = result.scopeBefore, result.scopeAfter
	}
	op.before, op.after = result.before, result.after
	s.populateSessionEnd(op, found, result)
	record.state = "accepted"
	s.publishSessionOperation(record)
	s.reconcileSessionRecord(record)
	s.remember(record.owner.User, op)
	if len(result.ended) > 0 {
		s.counter++
	}
	select {
	case s.wake <- struct{}{}:
	default:
	}
	return result, http.StatusAccepted
}

func (s *Server) acceptSessionSelection(w http.ResponseWriter, r *http.Request, found session, answer identity.Answer, body endBody) {
	request := r.Header.Get("X-Request-Id")
	record := s.sessionRecords[*body.Selection]
	if !s.ownsSessionRecord(record, found) || record.legacy || record.owner != (sessionOwner{answer.User(), answer.Workspace(), answer.Epoch()}) {
		failCode(w, commands.UnsupportedCommand, request)
		return
	}
	if record.digest != *body.Digest || len(record.targets) != *body.Count {
		invalidFields(w, request, "selection")
		return
	}
	if record.op != nil {
		if record.state != "accepted" {
			failCode(w, commands.AuthorityUnavailable, request)
			return
		}
		writeJSON(w, http.StatusAccepted, request, operationSummary(record, true))
		return
	}
	now := s.clock()
	if !now.Before(record.expires) || record.state != "sealed" {
		s.forgetSessionRecord(record)
		failRead(w, commands.RevisionConflict, request, "selection")
		return
	}
	s.expireSessionRecords(now)
	_, status := s.applySessionRecord(found, record, record.kind)
	switch status {
	case http.StatusUnauthorized:
		failCode(w, commands.Unauthenticated, request)
	case http.StatusConflict:
		s.forgetSessionRecord(record)
		failRead(w, commands.RevisionConflict, request, "selection")
	case http.StatusTooManyRequests:
		s.selectionLimit(w, request)
	default:
		writeJSON(w, http.StatusAccepted, request, operationSummary(record, false))
	}
}

func (s *Server) legacySessionRecord(found session, kind, listed, digest string, handles []string) *sessionRecord {
	index := s.sessions.indexState(found.owner, s.clock(), 0)
	record := s.newSessionRecord(found, s.clock(), s.clock().Add(time.Duration(s.cfg.Access.SessionQuerySeconds)*time.Second), index.Version, s.Revision())
	if record == nil {
		return nil
	}
	if !s.reserveSessionRecord(record, len(handles)) {
		s.forgetSessionRecord(record)
		return nil
	}
	record.legacy, record.state, record.generation, record.digest, record.kind = true, "sealed", listed, digest, kind
	for _, handle := range handles {
		target := frozenSession{Row: queriedSession{selfSession: selfSession{Handle: handle}, State: "unavailable"}}
		if current, known := s.sessions.indexedSession(found.owner, handle); known {
			target.Row = s.sessionView(current, found.verifier)
			target.verifier = current.verifier
		}
		record.positions[handle] = len(record.targets)
		record.targets = append(record.targets, target)
	}
	return record
}

func (op *selfEnd) countCleanup(state string, change int) {
	switch state {
	case itemDone:
		op.cleanup.Done += change
	case itemCleared:
		op.cleanup.AlreadyEnded += change
	case itemFailed:
		op.cleanup.Failed += change
	default:
		op.cleanup.Pending += change
	}
}
