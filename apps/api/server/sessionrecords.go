package server

import (
	"encoding/hex"
	"sort"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

type frozenSession struct {
	Row      queriedSession
	verifier identity.Verifier
}

type sessionRecord struct {
	ref             string
	kind            string
	legacy          bool
	indexed         bool
	olderOthers     *sessionRecord
	newerOthers     *sessionRecord
	resultPositions map[string]int
	heldPositions   map[string]int
	owner           sessionOwner
	caller          string
	at              time.Time
	expires         time.Time
	version         uint64
	revision        string
	after           string
	previous        string
	generation      string
	state           string
	targets         []frozenSession
	positions       map[string]int
	digest          string
	op              *selfEnd
	units           int
	retries         int
}

func (s *Server) forgetSessionRecord(record *sessionRecord) {
	s.forgetSessionRecordWithLockState(record, false)
}

func (s *Server) forgetSessionRecordWithLockState(record *sessionRecord, sessionsLocked bool) {
	if record.op != nil && (record.state != "accepted" || record.retries > 0) {
		return
	}
	if s.sessionRecords[record.ref] != record {
		return
	}
	delete(s.sessionRecords, record.ref)
	s.sessionRecordUnits -= record.units
	if record.op != nil {
		delete(s.sessionOperations, record.op.id)
		s.removeSessionOperationIndex(record, sessionsLocked)
		rows := s.selfEnds[record.owner.User]
		kept := rows[:0]
		for _, op := range rows {
			if op != record.op {
				kept = append(kept, op)
			}
		}
		clear(rows[len(kept):])
		if len(kept) == 0 {
			delete(s.selfEnds, record.owner.User)
		} else {
			s.selfEnds[record.owner.User] = kept
		}
	}
}

func (s *Server) expireSessionRecords(now time.Time) {
	for _, record := range s.sessionRecords {
		if record.op == nil && !now.Before(record.expires) {
			s.forgetSessionRecord(record)
		}
	}
}

func (s *Server) reserveSessionRecord(record *sessionRecord, units int) bool {
	return s.reserveSessionRecordWithLockState(record, units, false)
}

func (s *Server) reserveSessionRecordWhileSessionsLocked(record *sessionRecord, units int) bool {
	return s.reserveSessionRecordWithLockState(record, units, true)
}

func (s *Server) reserveSessionRecordWithLockState(record *sessionRecord, units int, sessionsLocked bool) bool {
	if units < 0 || units > s.cfg.Access.SessionRecordBudget {
		return false
	}
	if s.sessionRecordUnits+units > s.cfg.Access.SessionRecordBudget {
		eligible := []*sessionRecord{}
		available := s.cfg.Access.SessionRecordBudget - s.sessionRecordUnits
		for _, candidate := range s.sessionRecords {
			if candidate != record && candidate.op != nil && candidate.state == "accepted" && candidate.retries == 0 {
				eligible = append(eligible, candidate)
				available += candidate.units
			}
		}
		if available < units {
			return false
		}
		sort.Slice(eligible, func(i, j int) bool {
			a, b := eligible[i], eligible[j]
			if a.op.blockedAt.Equal(b.op.blockedAt) {
				return a.op.sequence < b.op.sequence
			}
			return a.op.blockedAt.Before(b.op.blockedAt)
		})
		for _, candidate := range eligible {
			if s.sessionRecordUnits+units <= s.cfg.Access.SessionRecordBudget {
				break
			}
			s.forgetSessionRecordWithLockState(candidate, sessionsLocked)
		}
	}
	s.sessionRecordUnits += units
	if record != nil {
		record.units += units
	}
	return true
}

func (s *Server) newSessionRecord(found session, at, expires time.Time, version uint64, revision string) *sessionRecord {
	s.expireSessionRecords(at)
	ref := "session-selection-" + hex.EncodeToString(random(16))
	if s.sessionRecords[ref] != nil || !s.reserveSessionRecord(nil, 1) {
		return nil
	}
	record := &sessionRecord{ref: ref, owner: found.owner, caller: found.handle, at: at, expires: expires, version: version, revision: revision, state: "building", kind: kindOthers, positions: map[string]int{}, units: 1}
	s.sessionRecords[ref] = record
	return record
}

func (s *Server) ownsSessionRecord(record *sessionRecord, found session) bool {
	return record != nil && record.owner == found.owner && record.caller == found.handle
}

func (s *Server) reconcileSessionRecord(record *sessionRecord) {
	if record == nil {
		return
	}
	actual := 1 + len(record.targets)
	if record.op != nil {
		actual += len(record.op.rows) + len(record.op.notIncluded) + record.retries
	}
	if actual > record.units {
		panic("session record reservation violated")
	}
	s.sessionRecordUnits -= record.units - actual
	record.units = actual
}
