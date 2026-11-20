package server

import "github.com/myblueton-sw/hybrid_billing/modules/identity"

func (s *Server) publishOtherSessionOperation(record *sessionRecord) {
	if record.kind != kindOthers {
		return
	}
	prior := s.sessionOtherOperations[record.owner]
	record.olderOthers = prior
	if prior != nil {
		prior.newerOthers = record
	}
	s.sessionOtherOperations[record.owner] = record
}

func (s *Server) forgetOtherSessionOperation(record *sessionRecord) {
	if record.kind != kindOthers {
		return
	}
	if record.newerOthers != nil {
		record.newerOthers.olderOthers = record.olderOthers
	} else if s.sessionOtherOperations[record.owner] == record {
		if record.olderOthers == nil {
			delete(s.sessionOtherOperations, record.owner)
		} else {
			s.sessionOtherOperations[record.owner] = record.olderOthers
		}
	}
	if record.olderOthers != nil {
		record.olderOthers.newerOthers = record.newerOthers
	}
	record.olderOthers, record.newerOthers = nil, nil
}

func (s *Server) taggedSessionView(target session, caller identity.Verifier) queriedSession {
	row := s.sessionView(target, caller)
	record := s.sessionOtherOperations[target.owner]
	if record != nil && record.indexed && record.state == "accepted" && record.owner == target.owner {
		if _, held := record.heldPositions[target.handle]; held {
			row.NotIncludedOperation = record.op.id
		}
	}
	return row
}
