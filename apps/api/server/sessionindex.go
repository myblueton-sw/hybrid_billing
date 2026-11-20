package server

import (
	"slices"
	"sort"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

type sessionOwner struct {
	User      string
	Workspace string
	Epoch     uint64
}
type sessionIndex struct {
	rows    []*session
	idle    []*session
	auth    []*session
	handles map[string]*session
	version uint64
}

func sessionCompare(a, b *session, kind string) int {
	switch kind {
	case "idle":
		if n := a.idleUntil.Compare(b.idleUntil); n != 0 {
			return n
		}
	case "auth":
		if n := a.authenticated.Compare(b.authenticated); n != 0 {
			return n
		}
	default:
		if n := b.created.Compare(a.created); n != 0 {
			return n
		}
	}
	return compareText(a.handle, b.handle)
}

func compareText(a, b string) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func insertSession(list []*session, found *session, kind string) []*session {
	at := sort.Search(len(list), func(i int) bool { return sessionCompare(list[i], found, kind) >= 0 })
	return slices.Insert(list, at, found)
}

func eraseSession(list []*session, found *session, kind string) []*session {
	at := sort.Search(len(list), func(i int) bool { return sessionCompare(list[i], found, kind) >= 0 })
	if at < len(list) && list[at] == found {
		return slices.Delete(list, at, at+1)
	}
	return list
}

func (s *sessions) indexChanged(index *sessionIndex) { s.indexSeq++; index.version = s.indexSeq }

func (s *sessions) unindex(found *session) {
	index := s.owners[found.owner]
	if index == nil {
		return
	}
	index.rows = eraseSession(index.rows, found, "rows")
	index.idle = eraseSession(index.idle, found, "idle")
	index.auth = eraseSession(index.auth, found, "auth")
	delete(index.handles, found.handle)
	s.indexChanged(index)
	if len(index.rows) == 0 {
		delete(s.owners, found.owner)
	}
}

func (s *sessions) bind(v identity.Verifier, key sessionOwner, idle time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	found := s.byID[v]
	if found == nil || key.User == "" || key.Workspace != found.workspace {
		return false
	}
	if found.owner.User != "" {
		return found.owner == key
	}
	index := s.owners[key]
	if index == nil {
		index = &sessionIndex{handles: map[string]*session{}}
		s.owners[key] = index
	}
	handle := handleOf(v)
	if prior := index.handles[handle]; prior != nil && prior.verifier != v {
		return false
	}
	found.owner, found.handle, found.idleUntil = key, handle, idle
	index.rows = insertSession(index.rows, found, "rows")
	index.idle = insertSession(index.idle, found, "idle")
	index.auth = insertSession(index.auth, found, "auth")
	index.handles[handle] = found
	s.indexChanged(index)
	return true
}

func (s *sessions) touch(v identity.Verifier, idle time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	found := s.byID[v]
	if found == nil || found.owner.User == "" || !idle.After(found.idleUntil) {
		return
	}
	index := s.owners[found.owner]
	index.idle = eraseSession(index.idle, found, "idle")
	found.idleUntil = idle
	index.idle = insertSession(index.idle, found, "idle")
}

type sessionIndexState struct {
	Version  uint64
	Count    int
	Deadline time.Time
}

func (s *sessions) indexState(key sessionOwner, now time.Time, maxAge time.Duration) sessionIndexState {
	s.mu.Lock()
	defer s.mu.Unlock()
	index := s.owners[key]
	if index == nil {
		return sessionIndexState{}
	}
	out := sessionIndexState{Version: index.version, Count: len(index.rows)}
	at := sort.Search(len(index.idle), func(i int) bool { return index.idle[i].idleUntil.After(now) })
	if at < len(index.idle) {
		out.Deadline = index.idle[at].idleUntil
	}
	if maxAge > 0 {
		at = sort.Search(len(index.auth), func(i int) bool { return index.auth[i].authenticated.Add(maxAge).Add(time.Nanosecond).After(now) })
		if at < len(index.auth) {
			out.Deadline = earlier(out.Deadline, index.auth[at].authenticated.Add(maxAge).Add(time.Nanosecond))
		}
	}
	return out
}

func earlier(a, b time.Time) time.Time {
	if a.IsZero() || !b.IsZero() && b.Before(a) {
		return b
	}
	return a
}

func (s *sessions) indexed(key sessionOwner, version uint64, after string, limit int) ([]session, bool, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index := s.owners[key]
	if index == nil || index.version != version {
		return nil, false, false
	}
	start := 0
	if after != "" {
		prior := index.handles[after]
		if prior == nil {
			return nil, false, false
		}
		start = sort.Search(len(index.rows), func(i int) bool { return sessionCompare(index.rows[i], prior, "rows") > 0 })
	}
	end := min(start+limit, len(index.rows))
	out := make([]session, 0, end-start)
	for _, found := range index.rows[start:end] {
		out = append(out, *found)
	}
	return out, end < len(index.rows), true
}
