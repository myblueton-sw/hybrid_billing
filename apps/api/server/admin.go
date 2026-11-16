package server

import (
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/passwords"
)

const usersTarget = "users"

var errPartial = errors.New("user administration stopped after a change")

type change struct {
	ctx     commands.Context
	user    identity.UserInfo
	login   string
	known   bool
	members map[string]identity.MemberInfo
	now     time.Time
}

type userCommand struct {
	s      *Server
	create bool
	apply  func(*Server, *change) (string, map[string]string, error)
}

func (h userCommand) Resolve(workspace, target string) (commands.Target, commands.Resolution) {
	s := h.s
	if workspace != s.bootWorkspace {
		return commands.Target{}, commands.NotFound
	}
	if h.create {
		if target != usersTarget {
			return commands.Target{}, commands.NotFound
		}
		return commands.Target{Revision: s.Revision()}, commands.Found
	}
	_, found, err := s.findUser(target)
	if err != nil {
		return commands.Target{}, commands.Unavailable
	}
	if !found {
		return commands.Target{}, commands.NotFound
	}
	return commands.Target{Revision: s.Revision()}, commands.Found
}

func (h userCommand) Apply(ctx commands.Context) (commands.Outcome, error) {
	s := h.s
	if len(ctx.Rationale) > maxRationale {
		return commands.Outcome{}, commands.Fail(commands.InvalidRequest)
	}
	c := &change{ctx: ctx, now: s.clock()}
	s.dir.prune(c.now)
	members, err := s.roster(s.caller)
	if !h.create && err == nil {
		c.user, _, err = s.findUser(ctx.Target)
		c.login, c.known = s.dir.loginOf(ctx.Target)
	}
	if err != nil {
		return commands.Outcome{}, mapIdentity(err)
	}
	c.members = members
	state, fields, err := h.apply(s, c)
	if err != nil {
		return commands.Outcome{}, err
	}
	return commands.Outcome{ResultRevision: s.Next(), BusinessState: state, Fields: fields}, nil
}

func (s *Server) faulty(step string) bool { return s.fault != nil && s.fault(step) }

func mapIdentity(err error) error {
	switch {
	case errors.Is(err, identity.ErrLastAdministrator):
		return commands.Fail(commands.PrerequisiteFailed)
	case errors.Is(err, identity.ErrLimit), errors.Is(err, identity.ErrUnavailable):
		return commands.Fail(commands.DependencyUnavailable)
	case errors.Is(err, identity.ErrInvalid):
		return commands.Fail(commands.InvalidRequest)
	}
	switch identity.ReasonOf(err) {
	case identity.ReasonSession, identity.ReasonNotAdministrator, identity.ReasonAnswer:
		return commands.Fail(commands.PermissionDenied)
	}
	return commands.Fail(commands.InvalidStage)
}

func (s *Server) users(after string) ([]identity.UserInfo, error) {
	if s.faulty("users") {
		return nil, identity.ErrUnavailable
	}
	return s.identity.Installation().Users(after, 100)
}

func (s *Server) findUser(id string) (identity.UserInfo, bool, error) {
	after := ""
	for {
		page, err := s.users(after)
		if err != nil {
			return identity.UserInfo{}, false, err
		}
		for _, found := range page {
			if found.ID == id {
				return found, found.Kind == identity.Human, nil
			}
		}
		if len(page) < 100 {
			return identity.UserInfo{}, false, nil
		}
		after = page[len(page)-1].ID
	}
}

func (s *Server) roster(actor identity.Verifier) (map[string]identity.MemberInfo, error) {
	members := map[string]identity.MemberInfo{}
	after := ""
	for {
		if s.faulty("roster") {
			return nil, identity.ErrUnavailable
		}
		page, err := s.identity.Workspace().Members(actor, s.bootWorkspace, after, 100)
		if err != nil {
			return nil, err
		}
		for _, found := range page {
			members[found.User] = found
		}
		if len(page) < 100 {
			return members, nil
		}
		after = page[len(page)-1].User
	}
}

func activeAdministrators(members map[string]identity.MemberInfo) map[string]bool {
	out := map[string]bool{}
	for user, found := range members {
		if found.Role == identity.Administrator && found.State == identity.MembershipActive && found.UserState == identity.Active {
			out[user] = true
		}
	}
	return out
}

func (s *Server) protect(c *change) error {
	target := c.ctx.Target
	if target == c.ctx.User {
		return commands.Fail(commands.PrerequisiteFailed)
	}
	administrators := activeAdministrators(c.members)
	if !administrators[target] {
		return nil
	}
	delete(administrators, target)
	for user := range administrators {
		if s.dir.holds(user, c.now) {
			return nil
		}
	}
	return commands.Fail(commands.PrerequisiteFailed)
}

func operator(c *change) identity.Operator {
	return identity.Operator{Reference: c.ctx.User, Reason: c.ctx.Rationale}
}

func validEmail(email string) bool {
	return email != "" && len(email) <= 254 && strings.Contains(email, "@") && strings.TrimSpace(email) == email
}

func (s *Server) inviteUser(actor identity.Verifier, login, user string, role identity.Role, now time.Time) error {
	id := "invitation-" + hex.EncodeToString(random(12))
	verifier := verifierOf(random(tokenBytes))
	err := s.identity.Workspace().Invite(actor, s.bootWorkspace, id, role, verifier, identity.Validity{NotBefore: now, NotAfter: now.Add(invitationValidity)}, user)
	if err != nil {
		return err
	}
	s.dir.invites[login] = invite{id: id, verifier: verifier, expires: now.Add(invitationValidity)}
	return nil
}

func (s *Server) recordLink(login, purpose string, now time.Time) (string, error) {
	token, err := s.passwords.Issue(LoginKey(login), purpose, setLinkValidity)
	if err != nil {
		return "", err
	}
	s.dir.links[login] = outstanding{purpose: purpose, expires: now.Add(setLinkValidity)}
	return token, nil
}

func (s *Server) issueLink(c *change, login, user, purpose string) (map[string]string, error) {
	token, err := s.recordLink(login, purpose, c.now)
	if err != nil {
		delete(s.dir.links, login)
		s.dir.dropPickups(func(p pickup) bool { return p.target == user })
		return nil, errPartial
	}
	s.dir.dropPickups(func(p pickup) bool { return p.target == user })
	ref := encode(random(16))
	s.dir.pickups[ref] = pickup{url: s.cfg.Origins[0] + "/activate#" + token, session: s.caller, target: user, expires: c.now.Add(pickupValidity)}
	return map[string]string{"link_ref": ref}, nil
}

func createUser(s *Server, c *change) (string, map[string]string, error) {
	login, ok := NormalizeLogin(c.ctx.Fields["login"])
	if !ok || strings.TrimSpace(login) != login {
		return "", nil, commands.Fail(commands.InvalidRequest, "login")
	}
	email := c.ctx.Fields["email"]
	if !validEmail(email) {
		return "", nil, commands.Fail(commands.InvalidRequest, "email")
	}
	if s.dir.taken[login] {
		return "", nil, commands.Fail(commands.PrerequisiteFailed)
	}
	if _, err := s.identity.Boundary().Resolve(LoginTriple(login)); err == nil {
		return "", nil, commands.Fail(commands.PrerequisiteFailed)
	}
	if s.dir.liveInvites(c.now) >= s.cfg.Identity.PendingInvitations || s.dir.liveLinks(c.now) >= s.cfg.Passwords.Links {
		return "", nil, commands.Fail(commands.DependencyUnavailable)
	}
	user := "user-" + hex.EncodeToString(random(8))
	if err := s.identity.Installation().CreateUser(operator(c), user, LoginTriple(login), email); err != nil {
		return "", nil, mapIdentity(err)
	}
	s.dir.add(user, login, email)
	if s.faulty("invite") || s.inviteUser(s.caller, login, user, identity.Role(c.ctx.Fields["role"]), c.now) != nil {
		return "", nil, errPartial
	}
	fields, err := s.issueLink(c, login, user, passwords.PurposeSet)
	if s.faulty("link") || err != nil {
		return "", nil, errPartial
	}
	fields["user_id"] = user
	return "user_created", fields, nil
}

func inviteExisting(s *Server, c *change) (string, map[string]string, error) {
	if _, member := c.members[c.ctx.Target]; member || !c.known || c.user.State != identity.Active {
		return "", nil, commands.Fail(commands.InvalidStage)
	}
	if _, live := s.dir.invitation(c.login, c.now); live {
		return "", nil, commands.Fail(commands.InvalidStage)
	}
	if err := s.inviteUser(s.caller, c.login, c.ctx.Target, identity.Role(c.ctx.Fields["role"]), c.now); err != nil {
		return "", nil, mapIdentity(err)
	}
	return "invited", nil, nil
}

func issueSetLink(s *Server, c *change) (string, map[string]string, error) {
	if !c.known || c.user.State == identity.Departed || s.passwords.Has(LoginKey(c.login)) {
		return "", nil, commands.Fail(commands.InvalidStage)
	}
	fields, err := s.issueLink(c, c.login, c.ctx.Target, passwords.PurposeSet)
	return "link_issued", fields, err
}

func issueResetLink(s *Server, c *change) (string, map[string]string, error) {
	if c.members[c.ctx.Target].Role == identity.Administrator {
		return "", nil, commands.Fail(commands.PrerequisiteFailed)
	}
	if c.user.State != identity.Active || !s.passwords.Has(LoginKey(c.login)) {
		return "", nil, commands.Fail(commands.InvalidStage)
	}
	fields, err := s.issueLink(c, c.login, c.ctx.Target, passwords.PurposeReset)
	return "link_issued", fields, err
}

func suspendUser(s *Server, c *change) (string, map[string]string, error) {
	if err := s.protect(c); err != nil {
		return "", nil, err
	}
	if err := s.identity.Installation().SuspendUser(operator(c), c.ctx.Target, c.user.Epoch); err != nil {
		return "", nil, mapIdentity(err)
	}
	return "user_suspended", nil, nil
}

func reinstateUser(s *Server, c *change) (string, map[string]string, error) {
	if err := s.identity.Installation().ReinstateUser(operator(c), c.ctx.Target, c.user.Epoch); err != nil {
		return "", nil, mapIdentity(err)
	}
	return "user_active", nil, nil
}

func departUser(s *Server, c *change) (string, map[string]string, error) {
	if err := s.protect(c); err != nil {
		return "", nil, err
	}
	if err := s.identity.Installation().DepartUser(operator(c), c.ctx.Target, c.user.Epoch); err != nil {
		return "", nil, mapIdentity(err)
	}
	if c.known {
		s.passwords.Remove(LoginKey(c.login))
		delete(s.dir.invites, c.login)
		delete(s.dir.links, c.login)
		delete(s.dir.people, c.ctx.Target)
		delete(s.dir.holders, c.ctx.Target)
		s.endSessions(c.login)
	}
	s.dir.dropPickups(func(p pickup) bool { return p.target == c.ctx.Target })
	return "user_departed", nil, nil
}

func changeRole(s *Server, c *change) (string, map[string]string, error) {
	role := identity.Role(c.ctx.Fields["role"])
	if err := s.protect(c); err != nil {
		return "", nil, err
	}
	if _, live := s.dir.link(c.login, c.now); live && role == identity.Administrator {
		return "", nil, commands.Fail(commands.PrerequisiteFailed)
	}
	if err := s.identity.Workspace().ChangeRole(s.caller, s.bootWorkspace, c.ctx.Target, role, c.members[c.ctx.Target].Version); err != nil {
		return "", nil, mapIdentity(err)
	}
	return "role_" + string(role), nil, nil
}

func membershipStep(guarded bool, state string, run func(*Server, *change, uint64) error) func(*Server, *change) (string, map[string]string, error) {
	return func(s *Server, c *change) (string, map[string]string, error) {
		if guarded {
			if err := s.protect(c); err != nil {
				return "", nil, err
			}
		}
		if err := run(s, c, c.members[c.ctx.Target].Version); err != nil {
			return "", nil, mapIdentity(err)
		}
		return state, nil, nil
	}
}

func revokeInvitation(s *Server, c *change) (string, map[string]string, error) {
	found, live := s.dir.invitation(c.login, c.now)
	if !live {
		return "", nil, commands.Fail(commands.InvalidStage)
	}
	if err := s.identity.Workspace().RevokeInvitation(s.caller, s.bootWorkspace, found.id); err != nil {
		return "", nil, mapIdentity(err)
	}
	delete(s.dir.invites, c.login)
	return "invitation_revoked", nil, nil
}

func roleField() commands.Field {
	return commands.Field{Name: "role", Type: commands.Enum, Required: true, Values: []string{string(identity.Administrator), string(identity.Member)}}
}

func (s *Server) registerUsers() error {
	for _, d := range append(append(s.userDefinitions(), s.policyDefinitions()...), s.accessDefinitions()...) {
		if err := s.Register(d); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) userDefinitions() []commands.Definition {
	type entry struct {
		id     string
		fields []commands.Field
		create bool
		apply  func(*Server, *change) (string, map[string]string, error)
	}
	ws := s.identity.Workspace()
	entries := []entry{
		{"identity.create_user", []commands.Field{{Name: "login", Type: commands.Text, Required: true}, {Name: "email", Type: commands.Text, Required: true}, roleField()}, true, createUser},
		{"identity.invite", []commands.Field{roleField()}, false, inviteExisting},
		{"identity.issue_set_link", nil, false, issueSetLink},
		{"identity.issue_reset_link", nil, false, issueResetLink},
		{"identity.suspend_user", nil, false, suspendUser},
		{"identity.reinstate_user", nil, false, reinstateUser},
		{"identity.depart_user", nil, false, departUser},
		{"identity.change_role", []commands.Field{roleField()}, false, changeRole},
		{"identity.suspend_membership", nil, false, membershipStep(true, "membership_suspended", func(s *Server, c *change, v uint64) error {
			return ws.SuspendMembership(s.caller, s.bootWorkspace, c.ctx.Target, v)
		})},
		{"identity.reinstate_membership", nil, false, membershipStep(false, "membership_active", func(s *Server, c *change, v uint64) error {
			return ws.ReinstateMembership(s.caller, s.bootWorkspace, c.ctx.Target, v)
		})},
		{"identity.remove_membership", nil, false, membershipStep(true, "membership_removed", func(s *Server, c *change, v uint64) error {
			return ws.RemoveMembership(s.caller, s.bootWorkspace, c.ctx.Target, v)
		})},
		{"identity.revoke_invitation", nil, false, revokeInvitation},
	}
	out := make([]commands.Definition, 0, len(entries))
	for _, e := range entries {
		out = append(out, commands.Definition{ID: e.id, Permission: commands.Permission{Action: gate.AdministerInstallation, Kind: gate.InstallationKind},
			Mutating: true, Rationale: true, Fields: e.fields, Handler: userCommand{s: s, create: e.create, apply: e.apply}})
	}
	return out
}
