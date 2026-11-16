package server

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

type memberRow struct {
	Role    string `json:"role"`
	State   string `json:"state"`
	Version uint64 `json:"version"`
}

type userRow struct {
	UserID            string     `json:"user_id"`
	Login             string     `json:"login"`
	Email             string     `json:"email"`
	State             string     `json:"state"`
	Epoch             uint64     `json:"epoch"`
	Membership        *memberRow `json:"membership"`
	PasswordSet       bool       `json:"password_set"`
	InvitationExpires string     `json:"invitation_expires"`
	LinkPurpose       string     `json:"link_purpose"`
	LinkExpires       string     `json:"link_expires"`
	Holder            bool       `json:"installation_administrator"`
}

type usersPage struct {
	Revision string    `json:"revision"`
	Next     string    `json:"next"`
	Users    []userRow `json:"users"`
}

func stamp(t time.Time) string { return t.UTC().Format(time.RFC3339) }

func (s *Server) admitted(w http.ResponseWriter, r *http.Request) (session, identity.Answer, bool) {
	request := r.Header.Get("X-Request-Id")
	found, ok := s.lookup(r)
	if !ok {
		failCode(w, commands.Unauthenticated, request)
		return found, identity.Answer{}, false
	}
	if !sameToken(r.Header.Get(csrfHeader), found.csrf) {
		fail(w, http.StatusForbidden, CodeCSRF, request)
		return found, identity.Answer{}, false
	}
	answer, code := s.access(found)
	if code != "" {
		failCode(w, code, request)
		return found, answer, false
	}
	decision := s.gate.Decide(gate.Answer(answer), ConsoleClient, gate.Request{Action: gate.AdministerInstallation, Kind: gate.InstallationKind,
		Resource: gate.Resource{WorkspaceID: answer.Workspace()}})
	if !decision.Allowed {
		code := commands.PermissionDenied
		if decision.Code == gate.AuthorityUnavailable {
			code = commands.AuthorityUnavailable
		}
		failCode(w, code, request)
		return found, answer, false
	}
	return found, answer, true
}

func pageLimit(r *http.Request) (string, int, bool) {
	after := r.URL.Query().Get("after")
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 100 {
			return "", 0, false
		}
		limit = n
	}
	if after != "" && !validWorkspace(after) {
		return "", 0, false
	}
	return after, limit, true
}

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	after, limit, ok := pageLimit(r)
	if !ok {
		failCode(w, commands.InvalidRequest, request)
		return
	}
	s.domain.RLock()
	defer s.domain.RUnlock()
	found, _, ok := s.admitted(w, r)
	if !ok {
		return
	}
	page, err := s.usersPage(found.verifier, after, limit)
	if err != nil {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	writeJSON(w, http.StatusOK, request, page)
}

func (s *Server) usersPage(actor identity.Verifier, after string, limit int) (usersPage, error) {
	members, err := s.roster(actor)
	if err != nil {
		return usersPage{}, err
	}
	now := s.clock()
	out := usersPage{Revision: s.Revision(), Users: []userRow{}}
	for {
		batch, err := s.users(after)
		if err != nil {
			return usersPage{}, err
		}
		for _, found := range batch {
			if found.Kind != identity.Human {
				continue
			}
			if len(out.Users) == limit {
				out.Next = out.Users[limit-1].UserID
				return out, nil
			}
			out.Users = append(out.Users, s.row(found, members, now))
		}
		if len(batch) < 100 {
			return out, nil
		}
		after = batch[len(batch)-1].ID
	}
}

func (s *Server) row(found identity.UserInfo, members map[string]identity.MemberInfo, now time.Time) userRow {
	row := userRow{UserID: found.ID, State: string(found.State), Epoch: found.Epoch, Holder: s.dir.holds(found.ID, now)}
	if m, ok := members[found.ID]; ok {
		row.Membership = &memberRow{Role: string(m.Role), State: string(m.State), Version: m.Version}
	}
	if p, ok := s.dir.people[found.ID]; ok {
		row.Login, row.Email = p.login, p.email
		row.PasswordSet = s.passwords.Has(LoginKey(p.login))
		if inv, live := s.dir.invitation(p.login, now); live {
			row.InvitationExpires = stamp(inv.expires)
		}
		if l, live := s.dir.link(p.login, now); live {
			row.LinkPurpose, row.LinkExpires = l.purpose, stamp(l.expires)
		}
	}
	return row
}

func (s *Server) collect(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.Lock()
	defer s.domain.Unlock()
	found, _, ok := s.admitted(w, r)
	if !ok {
		return
	}
	ref := r.PathValue("ref")
	p, exists := s.dir.pickups[ref]
	if exists && !s.clock().Before(p.expires) {
		delete(s.dir.pickups, ref)
		exists = false
	}
	if !exists || p.session != found.verifier {
		failCode(w, commands.PermissionDenied, request)
		return
	}
	delete(s.dir.pickups, ref)
	writeJSON(w, http.StatusOK, request, map[string]string{"url": p.url})
}

func (s *Server) endSessions(login string) {
	for _, v := range s.sessions.byLogin(login) {
		_ = s.identity.Boundary().EndSession(v)
		s.sessions.drop(v)
		s.dir.dropPickups(func(p pickup) bool { return p.session == v })
	}
}

func (s *Server) probe(login string) (identity.Verifier, error) {
	v := verifierOf(random(tokenBytes))
	err := s.identity.Boundary().OpenSession(LoginTriple(login), v, time.Minute, time.Minute)
	for tries := 0; errors.Is(err, identity.ErrLimit) && tries < maxEvictions; tries++ {
		old, ok := s.sessions.oldest(login, v)
		if !ok {
			break
		}
		_ = s.identity.Boundary().EndSession(old)
		s.sessions.drop(old)
		err = s.identity.Boundary().OpenSession(LoginTriple(login), v, time.Minute, time.Minute)
	}
	return v, err
}

func (s *Server) acceptPending(login string, actor identity.Verifier, now time.Time) bool {
	found, live := s.dir.invitation(login, now)
	_, present := s.dir.invites[login]
	delete(s.dir.invites, login)
	if live {
		_ = s.identity.Workspace().Accept(s.bootWorkspace, found.id, found.verifier, actor)
	}
	return present
}
