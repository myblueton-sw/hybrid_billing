package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/accesshistory"
	"github.com/myblueton-sw/hybrid_billing/modules/authpolicy"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"strconv"
)

func (s *Server) retryModuleEnd(v identity.Verifier) error {
	if s.faulty("self-end-unavailable") {
		return identity.ErrUnavailable
	}
	if s.faulty("self-end-unexpected") {
		return errSelfEndUnexpected
	}
	return s.identity.Boundary().EndSession(v)
}

func (s *Server) dropSession(v identity.Verifier, cause string) (session, bool) {
	found, ok := s.sessions.take(v)
	if ok && cause != "" {
		s.recordEnded(found, cause)
	}
	return found, ok
}

func (s *Server) closeSession(v identity.Verifier, cause string) {
	_ = s.identity.Boundary().EndSession(v)
	s.dropSession(v, cause)
}

func (s *Server) endSession(v identity.Verifier, cause string) {
	s.closeSession(v, cause)
	s.dir.dropPickups(func(p pickup) bool { return p.session == v })
}

func (s *Server) endOne(v identity.Verifier, cause string) {
	err := s.retryModuleEnd(v)
	s.dropSession(v, cause)
	s.settleEnded(v, err)
	s.dir.dropPickups(func(p pickup) bool { return p.session == v })
}

func (s *Server) endSessions(login, cause string) {
	for _, v := range s.sessions.byLogin(login) {
		s.endSession(v, cause)
	}
}

func (s *Server) evictOldest(login string, except identity.Verifier) bool {
	old, ok := s.sessions.oldest(login, except)
	if !ok {
		return false
	}
	s.endSession(old, accesshistory.CauseSessionLimit)
	return true
}

func (s *Server) rollbackOpened(v identity.Verifier, module bool) {
	if module {
		_ = s.identity.Boundary().EndSession(v)
	}
	s.sessions.drop(v)
}

func (s *Server) endProbe(v identity.Verifier) { _ = s.identity.Boundary().EndSession(v) }

func (s *Server) logoutSession(found session) {
	_ = s.identity.Boundary().EndSession(found.verifier)
	if _, ok := s.sessions.take(found.verifier); ok {
		s.recordSession(accesshistory.KindSignedOut, "", found)
	}
	s.dir.dropPickups(func(p pickup) bool { return p.session == found.verifier })
}

func (s *Server) cleanSession(v identity.Verifier) {
	s.closeSession(v, accesshistory.CauseBulkRevocation)
}

func identityCause(reason identity.Reason) string {
	switch reason {
	case identity.ReasonSession:
		return ""
	case identity.ReasonUser, identity.ReasonOperatorBlock:
		return accesshistory.CauseAdministrator
	case identity.ReasonRolePeriod:
		return accesshistory.CauseRolePeriod
	}
	return accesshistory.CauseAccessChanged
}

func policyCause(reason string) string {
	if reason == authpolicy.ReasonStale {
		return ""
	}
	return accesshistory.CauseAccessChanged
}

func (s *Server) takeSessionRecord(found session, record *sessionRecord, begin func(int) bool) (manifestResult, bool) {
	store := s.sessions
	store.mu.Lock()
	defer store.mu.Unlock()
	own := store.byID[found.verifier]
	if own == nil || own.owner != record.owner || own.handle != record.caller {
		return manifestResult{}, false
	}
	index := store.owners[record.owner]
	if index == nil {
		return manifestResult{}, false
	}
	matches := func(row *session) bool {
		position, known := record.positions[row.handle]
		return known && record.targets[position].verifier == row.verifier
	}
	held := 0
	if record.op.kind == kindOthers {
		for _, row := range index.rows {
			if row.verifier != found.verifier && !matches(row) {
				held++
			}
		}
	}
	if !begin(2*len(record.targets) + held) {
		return manifestResult{}, true
	}
	out := manifestResult{ok: true, before: store.token[found.login], scopeBefore: strconv.FormatUint(index.version, 10), ended: make([]session, 0, len(record.targets)), gone: []string{}, notIncluded: make([]session, 0, held)}
	for _, row := range index.rows {
		if row.verifier == found.verifier {
			continue
		}
		if matches(row) {
			out.ended = append(out.ended, *row)
		} else if record.op.kind == kindOthers {
			out.notIncluded = append(out.notIncluded, *row)
		}
	}
	for _, target := range record.targets {
		current := index.handles[target.Row.Handle]
		if current == nil || current.verifier != target.verifier {
			out.gone = append(out.gone, target.Row.Handle)
		}
	}
	for _, row := range out.ended {
		store.remove(row.verifier)
	}
	if len(out.ended) > 0 {
		store.changed(found.login)
	}
	out.after = store.token[found.login]
	if current := store.owners[record.owner]; current != nil {
		out.scopeAfter = strconv.FormatUint(current.version, 10)
	}
	return out, true
}
