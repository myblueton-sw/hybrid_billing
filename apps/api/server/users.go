package server

import (
	"errors"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

type memberRow struct {
	Role         string `json:"role"`
	State        string `json:"state"`
	Version      uint64 `json:"version"`
	RoleFrom     string `json:"role_from"`
	RoleUntil    string `json:"role_until"`
	RolePeriod   string `json:"role_period"`
	SuspendedFor string `json:"suspended_for"`
	Ledger       string `json:"ledger"`
}

type revocationRef struct {
	Bulk  string `json:"bulk"`
	State string `json:"state"`
}

type userRow struct {
	UserID            string         `json:"user_id"`
	Login             string         `json:"login"`
	Email             string         `json:"email"`
	State             string         `json:"state"`
	Epoch             uint64         `json:"epoch"`
	Membership        *memberRow     `json:"membership"`
	PasswordSet       bool           `json:"password_set"`
	InvitationExpires string         `json:"invitation_expires"`
	LinkPurpose       string         `json:"link_purpose"`
	LinkExpires       string         `json:"link_expires"`
	Holder            bool           `json:"installation_administrator"`
	Revocation        *revocationRef `json:"revocation"`
}

type authMethods struct {
	Connected    []string `json:"connected"`
	NotConnected []string `json:"not_connected"`
}

type usersPage struct {
	Revision  string    `json:"revision"`
	Next      string    `json:"next"`
	NextLogin string    `json:"next_login"`
	Users     []userRow `json:"users"`
	clockRows
	Settings      []settingRow `json:"settings"`
	LedgerState   string       `json:"ledger_state"`
	CanBulkRevoke bool         `json:"can_bulk_revoke"`
	Filters       userFilter   `json:"filters"`
	ScanStopped   bool         `json:"scan_stopped"`
	ScanBudget    int          `json:"scan_budget"`
	Order         string       `json:"order"`
	ListVersion   string       `json:"list_version"`
	AuthMethods   authMethods  `json:"auth_methods"`
	WarningDays   int          `json:"warning_days"`
}

func stamp(t time.Time) string { return t.UTC().Format(time.RFC3339) }

func stampOr(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return stamp(t)
}

func (s *Server) rolePeriod(m identity.MemberInfo, now time.Time) string {
	warning := time.Duration(s.cfg.Access.RoleWarningDays) * 24 * time.Hour
	switch {
	case m.SuspendedFor == identity.RolePeriodEnded || !m.Until.IsZero() && !now.Before(m.Until):
		return "ended"
	case m.Until.IsZero():
		return "open"
	case m.Until.Sub(now) <= warning:
		return "ends_soon"
	}
	return "bounded"
}

func (s *Server) canBulk(answer identity.Answer) bool {
	return s.gate.Decide(gate.Answer(answer), ConsoleClient, gate.Request{Action: gate.RevokeAccessInBulk, Kind: gate.InstallationKind,
		Resource: gate.Resource{WorkspaceID: answer.Workspace()}}).Allowed
}

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

var (
	errListChanged = errors.New("the cursor person left the login order")
	errBadCursor   = errors.New("the cursor names no listed person")
)

var readRules = map[commands.Code]struct {
	status int
	next   commands.NextAction
}{
	commands.InvalidRequest:        {http.StatusUnprocessableEntity, commands.NextCorrect},
	commands.UnsupportedCapability: {http.StatusNotImplemented, commands.NextNone},
	commands.RevisionConflict:      {http.StatusConflict, commands.NextInspect},
}

func failRead(w http.ResponseWriter, code commands.Code, request string, fields ...string) {
	rule := readRules[code]
	writeJSON(w, rule.status, request, errorBody{Code: string(code), NextAction: string(rule.next), RequestID: request, Fields: fields})
}

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	query, code, _ := parseUsersQuery(r.URL.Query())
	if code != "" {
		failRead(w, code, request)
		return
	}
	s.domain.RLock()
	defer s.domain.RUnlock()
	found, answer, ok := s.admitted(w, r)
	if !ok {
		return
	}
	if answer.Workspace() != s.bootWorkspace {
		failCode(w, commands.PermissionDenied, request)
		return
	}
	page, err := s.usersPage(found.verifier, query)
	switch {
	case errors.Is(err, errListChanged):
		failRead(w, commands.RevisionConflict, request, "after")
		return
	case errors.Is(err, errBadCursor):
		failRead(w, commands.InvalidRequest, request, "after")
		return
	case err != nil:
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	page.CanBulkRevoke = s.canBulk(answer) && len(s.stranded) == 0
	writeJSON(w, http.StatusOK, request, page)
}

func (s *Server) listVersion() string {
	return strconv.FormatUint(s.identity.Subjects().Generation(), 10) + "." + strconv.FormatUint(s.dir.version, 10)
}

func (s *Server) resume(after *userCursor) ([]string, []string, error) {
	d := s.dir
	if after == nil {
		return d.order, d.departed, nil
	}
	if after.departed {
		at := sort.SearchStrings(d.departed, after.user)
		if at == len(d.departed) || d.departed[at] != after.user {
			return nil, nil, errBadCursor
		}
		return nil, d.departed[at+1:], nil
	}
	if login, ok := d.loginOf(after.user); ok {
		return d.order[d.position(after.user, login)+1:], d.departed, nil
	}
	found, ok, err := s.findUser(after.user)
	switch {
	case err != nil:
		return nil, nil, err
	case ok && found.State == identity.Departed:
		return nil, nil, errListChanged
	}
	return nil, nil, errBadCursor
}

func (s *Server) people(actor identity.Verifier, ids []string) (map[string]identity.Person, error) {
	if s.faulty("people") {
		return nil, identity.ErrUnavailable
	}
	found, err := s.identity.Workspace().People(actor, s.bootWorkspace, ids)
	out := make(map[string]identity.Person, len(found))
	for _, p := range found {
		out[p.User.ID] = p
	}
	return out, err
}

type usersScan struct {
	peopleWalk
	query   usersQuery
	matcher userMatcher
	rows    []userRow
}

func (c *usersScan) segment(ids []string, departed bool) (bool, error) {
	rest, err := c.walk(ids, departed, func(row userRow, member *identity.MemberInfo) bool {
		if !c.matcher.matches(row, member, c.now) {
			return false
		}
		c.rows = append(c.rows, row)
		return len(c.rows) == c.query.limit
	})
	return len(rest) > 0, err
}

func (s *Server) usersPage(actor identity.Verifier, query usersQuery) (usersPage, error) {
	login, departed, err := s.resume(query.after)
	if err != nil {
		return usersPage{}, err
	}
	now := s.clock()
	scan := &usersScan{peopleWalk: peopleWalk{s: s, actor: actor, now: now, budget: s.cfg.Access.UserFilterScan}, query: query, matcher: newMatcher(query.filter), rows: []userRow{}}
	more, err := scan.segment(login, false)
	if err == nil && !more && len(scan.rows) < query.limit {
		more, err = scan.segment(departed, true)
	} else if err == nil && !more {
		more = len(departed) > 0
	}
	if err != nil {
		return usersPage{}, err
	}
	out := usersPage{Revision: s.Revision(), Users: scan.rows, clockRows: s.clockRows(now), Settings: s.bulkSettingRows(), LedgerState: s.ledgerRow().State,
		Filters: query.filter, ScanBudget: scan.budget, Order: usersOrder, ListVersion: s.listVersion(),
		AuthMethods: authMethods{Connected: []string{"password"}, NotConnected: []string{"saml", "oidc", "ldap"}}, WarningDays: s.cfg.Access.RoleWarningDays}
	if more {
		out.Next, out.NextLogin = scan.last.String(), s.dir.people[scan.last.user].login
		out.ScanStopped = len(scan.rows) < query.limit
	}
	ids := make([]string, 0, len(scan.rows))
	for _, row := range scan.rows {
		ids = append(ids, row.UserID)
	}
	latest := s.bulk.LatestFor(s.bootWorkspace, ids)
	for i := range out.Users {
		if l, ok := latest[out.Users[i].UserID]; ok {
			out.Users[i].Revocation = &revocationRef{Bulk: l.Bulk, State: l.State}
		}
	}
	return out, nil
}

func (s *Server) row(found identity.UserInfo, m *identity.MemberInfo, now time.Time) userRow {
	row := userRow{UserID: found.ID, State: string(found.State), Epoch: found.Epoch, Holder: s.dir.holds(found.ID, now)}
	if m != nil {
		row.Membership = &memberRow{Role: string(m.Role), State: string(m.State), Version: m.Version, RoleFrom: stampOr(m.From), RoleUntil: stampOr(m.Until),
			RolePeriod: s.rolePeriod(*m, now), SuspendedFor: string(m.SuspendedFor), Ledger: s.memberLedger[found.ID]}
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
	if p.secret != "" {
		writeJSON(w, http.StatusOK, request, map[string]string{"secret": p.secret, "credential": p.credential})
		return
	}
	writeJSON(w, http.StatusOK, request, map[string]string{"url": p.url})
}

func (s *Server) probe(login string) (identity.Verifier, error) {
	v := verifierOf(random(tokenBytes))
	err := s.identity.Boundary().OpenSession(LoginTriple(login), v, time.Minute, time.Minute)
	if errors.Is(err, identity.ErrLimit) {
		s.settleDenied(login)
		err = s.identity.Boundary().OpenSession(LoginTriple(login), v, time.Minute, time.Minute)
	}
	for tries := 0; errors.Is(err, identity.ErrLimit) && tries < maxEvictions; tries++ {
		if !s.evictOldest(login, v) {
			break
		}
		err = s.identity.Boundary().OpenSession(LoginTriple(login), v, time.Minute, time.Minute)
	}
	return v, err
}

func (s *Server) acceptPending(login string, actor identity.Verifier, now time.Time) bool {
	found, live := s.dir.invitation(login, now)
	_, present := s.dir.invites[login]
	s.dir.dropInvite(login)
	if live && s.identity.Workspace().Accept(s.bootWorkspace, found.id, found.verifier, actor) == nil {
		if user, ok := s.dir.userOf(login); ok {
			s.applyAwaiting(user)
		}
	}
	return present
}
