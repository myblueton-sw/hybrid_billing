package server

import (
	"sort"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

type snapshot struct {
	token    string
	counter  uint64
	sessions []session
}

type manifestResult struct {
	ok          bool
	ended       []session
	gone        []string
	notIncluded []session
	before      string
	after       string
}

func ordered(list []session) []session {
	sort.Slice(list, func(a, b int) bool {
		if !list[a].created.Equal(list[b].created) {
			return list[a].created.After(list[b].created)
		}
		return handleOf(list[a].verifier) < handleOf(list[b].verifier)
	})
	return list
}

func (s *sessions) snapshot(login string) snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := snapshot{token: s.token[login], counter: s.gen[login]}
	for _, found := range s.byID {
		if found.login == login {
			out.sessions = append(out.sessions, *found)
		}
	}
	ordered(out.sessions)
	return out
}

func (s *sessions) generation(login string) uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.gen[login]
}

func (s *sessions) endManifest(login string, caller identity.Verifier, confirmed []string, others bool) manifestResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := manifestResult{before: s.token[login]}
	if own := s.byID[caller]; own == nil || own.login != login {
		return out
	}
	out.ok = true
	want := map[string]bool{}
	for _, h := range confirmed {
		want[h] = true
	}
	present := map[string]bool{}
	for v, found := range s.byID {
		if found.login != login || v == caller {
			continue
		}
		h := handleOf(v)
		switch {
		case want[h]:
			present[h] = true
			out.ended = append(out.ended, *found)
		case others:
			out.notIncluded = append(out.notIncluded, *found)
		}
	}
	for _, found := range out.ended {
		s.remove(found.verifier)
	}
	if len(out.ended) > 0 {
		s.changed(login)
	}
	for _, h := range confirmed {
		if !present[h] {
			out.gone = append(out.gone, h)
		}
	}
	sort.Strings(out.gone)
	ordered(out.ended)
	ordered(out.notIncluded)
	out.after = s.token[login]
	return out
}

func (s *sessions) deny(v identity.Verifier, login string, at time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, held := s.denied[v]; held {
		return true
	}
	if len(s.denied) >= maxSessions {
		s.overflow[login] = at
		return false
	}
	s.denied[v] = denial{login: login, at: at}
	return true
}

func (s *sessions) isDenied(v identity.Verifier) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, held := s.denied[v]
	return held
}

func (s *sessions) release(v identity.Verifier) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.denied, v)
}

func (s *sessions) deniedOf(login string) []identity.Verifier {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []identity.Verifier{}
	for v, d := range s.denied {
		if d.login == login {
			out = append(out, v)
		}
	}
	sort.Slice(out, func(a, b int) bool { return handleOf(out[a]) < handleOf(out[b]) })
	return out
}

func (s *sessions) overflowed(login string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, full := s.overflow[login]
	return full
}

func (s *sessions) sweep(now time.Time) []identity.Verifier {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []identity.Verifier{}
	for v, d := range s.denied {
		if !now.Before(d.at.Add(sessionAbsolute)) {
			delete(s.denied, v)
			out = append(out, v)
		}
	}
	for login, at := range s.overflow {
		if !now.Before(at.Add(sessionAbsolute)) {
			delete(s.overflow, login)
		}
	}
	return out
}

func (s *sessions) present(v identity.Verifier) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.byID[v] != nil
}
