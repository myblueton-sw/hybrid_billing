package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/accesshistory"
	"github.com/myblueton-sw/hybrid_billing/modules/authpolicy"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
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

func (s *Server) takeManifest(found session, handles []string, others bool) manifestResult {
	return s.sessions.endManifest(found.login, found.verifier, handles, others)
}

func (s *Server) cleanSession(v identity.Verifier) { s.closeSession(v, accesshistory.CauseBulkRevocation) }

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
