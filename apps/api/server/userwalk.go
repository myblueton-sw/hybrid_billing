package server

import (
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

type peopleWalk struct {
	s        *Server
	actor    identity.Verifier
	now      time.Time
	budget   int
	examined int
	last     userCursor
}

func (w *peopleWalk) walk(ids []string, departed bool, visit func(userRow, *identity.MemberInfo) bool) ([]string, error) {
	for len(ids) > 0 {
		size := min(len(ids), identity.PeopleLimit)
		if w.budget > 0 {
			size = min(size, w.budget-w.examined)
		}
		if size == 0 {
			return ids, nil
		}
		found, err := w.s.people(w.actor, ids[:size])
		if err != nil {
			return nil, err
		}
		for i, id := range ids[:size] {
			w.examined++
			w.last = userCursor{departed: departed, user: id}
			p := found[id]
			if p.User.Kind != identity.Human {
				continue
			}
			if visit(w.s.row(p.User, p.Member, w.now), p.Member) {
				return ids[i+1:], nil
			}
		}
		ids = ids[size:]
	}
	return nil, nil
}

type matchWalk struct {
	peopleWalk
	matcher userMatcher
	limit   int
	matched []string
}

func (m *matchWalk) all(login, departed []string) (bool, error) {
	visit := func(row userRow, member *identity.MemberInfo) bool {
		if m.matcher.matches(row, member, m.now) {
			m.matched = append(m.matched, row.UserID)
		}
		return len(m.matched) > m.limit
	}
	rest, err := m.walk(login, false, visit)
	if err != nil || len(rest) > 0 || len(m.matched) > m.limit {
		return len(rest) == 0 && len(departed) == 0, err
	}
	rest, err = m.walk(departed, true, visit)
	return len(rest) == 0, err
}

func (s *Server) matchWalk(actor identity.Verifier, filter userFilter, now time.Time, budget int) *matchWalk {
	return &matchWalk{peopleWalk: peopleWalk{s: s, actor: actor, now: now, budget: budget}, matcher: newMatcher(filter), limit: s.cfg.Access.BulkMaxMatching}
}
