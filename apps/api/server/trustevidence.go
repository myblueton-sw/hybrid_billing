package server

import (
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/authpolicy"
	"github.com/myblueton-sw/hybrid_billing/modules/decisionledger"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/passwords"
	"github.com/myblueton-sw/hybrid_billing/modules/trustchange"
)

const (
	itemTestLogin   = "test_login"
	itemRecovery    = "recovery_path"
	itemIssuer      = "issuer_subject"
	itemDiagnostics = "provider_diagnostics"
	itemExternal    = "external_status"
	itemMFA         = "mfa_weakening"
	itemAffected    = "affected_subjects"
	itemSessions    = "session_impact"
	itemRecord      = "record_size"

	resultVerified      = "verified"
	resultBlocked       = "blocked"
	resultNotApplicable = "not_applicable"
	resultInformation   = "information"
	maxAffectedShown    = 100
)

type trustPerson struct {
	user     string
	member   identity.MemberInfo
	info     identity.UserInfo
	password bool
	blocked  bool
	admitted bool
	rights   map[gate.Action]gate.Explanation
	grants   map[gate.Action][]string
}

type trustFacts struct {
	people   []trustPerson
	members  int
	affected []string
	sessions int
	now      time.Time
}

func (p trustPerson) qualifies(right gate.Action) bool {
	m := p.member
	return p.info.Kind == identity.Human && p.info.State == identity.Active && m.State == identity.MembershipActive && m.Role == identity.Administrator &&
		p.password && !p.blocked && p.admitted && p.rights[right].Allowed
}

func (p trustPerson) fact() string {
	parts := []string{p.user, strconv.FormatUint(p.member.Version, 10), string(p.member.State), string(p.member.Role), string(p.info.State),
		strconv.FormatUint(p.info.Epoch, 10), string(p.info.Kind), strconv.FormatBool(p.password), strconv.FormatBool(p.blocked), strconv.FormatBool(p.admitted)}
	for _, right := range trustRights {
		e := p.rights[right]
		parts = append(parts, string(right)+"="+strconv.FormatBool(e.Allowed)+":"+string(e.Reason)+":"+strings.Join(p.grants[right], ","))
	}
	return strings.Join(parts, "/")
}

func (s *Server) usablePassword(login string, now time.Time) bool {
	link, pending := s.dir.link(login, now)
	return s.passwords.Has(LoginKey(login)) && !(pending && link.purpose == passwords.PurposeReset)
}

func (s *Server) grantsFor(user string, right gate.Action, now time.Time) []string {
	out := []string{}
	for _, g := range s.admin.Grants() {
		if g.Terms.Action == right && (g.SubjectID == user || g.ClientID == ConsoleClient) && !g.Closed && !now.Before(g.Validity.NotBefore) && now.Before(g.Validity.NotAfter) {
			out = append(out, g.ID+"@"+strconv.FormatUint(g.Version, 10))
		}
	}
	sort.Strings(out)
	return out
}

func (s *Server) trustFacts(actor identity.Verifier, workspace string, candidate *authpolicy.Revision, now time.Time) (trustFacts, error) {
	f, err := s.accessFacts(actor)
	if err != nil {
		return trustFacts{}, err
	}
	out := trustFacts{members: len(f.members), now: now, affected: []string{}}
	for user, m := range f.members {
		out.affected = append(out.affected, user)
		if m.Role != identity.Administrator {
			continue
		}
		login, _ := s.dir.loginOf(user)
		p := trustPerson{user: user, member: m, info: f.users[user], password: s.usablePassword(login, now), blocked: s.policy.Blocked(workspace, user), admitted: true,
			rights: map[gate.Action]gate.Explanation{}, grants: map[gate.Action][]string{}}
		if candidate != nil {
			p.admitted = authpolicy.Reachable(authpolicy.Principal{User: user, Workspace: workspace, Role: authpolicy.RoleAdministrator, Kind: authpolicy.KindHuman}, *candidate)
		}
		for _, right := range trustRights {
			p.rights[right] = s.admin.ExplainInstallation(user, ConsoleClient, gate.Request{Action: right, Kind: gate.InstallationKind, Resource: gate.Resource{WorkspaceID: workspace}}, now)
			if p.rights[right].Allowed {
				p.grants[right] = s.grantsFor(user, right, now)
			}
		}
		out.people = append(out.people, p)
	}
	sort.Strings(out.affected)
	sort.Slice(out.people, func(a, b int) bool { return out.people[a].user < out.people[b].user })
	out.sessions = len(s.sessions.below(workspace, ^uint64(0)))
	return out, nil
}

func (f trustFacts) qualifying() int {
	n := 0
	for _, p := range f.people {
		for _, right := range trustRights {
			if p.qualifies(right) {
				n++
				break
			}
		}
	}
	return n
}

func (f trustFacts) recovery() (bool, string) {
	missing := []string{}
	for _, right := range trustRights {
		found := false
		for _, p := range f.people {
			found = found || p.qualifies(right)
		}
		if !found {
			missing = append(missing, string(right))
		}
	}
	if len(missing) > 0 {
		return false, "missing:" + strings.Join(missing, ",")
	}
	for _, a := range f.people {
		for _, b := range f.people {
			for _, c := range f.people {
				if a.user != b.user && b.user != c.user && a.user != c.user && a.qualifies(gate.TrustPropose) && b.qualifies(gate.TrustApprove) && c.qualifies(gate.TrustActivate) {
					return true, ""
				}
			}
		}
	}
	return false, "three_distinct_people"
}

func (f trustFacts) digestFacts(base trustchange.Base) []string {
	out := []string{"epoch:" + strconv.FormatUint(base.Epoch, 10), "policy:" + strconv.FormatUint(base.Policy, 10), "members:" + strconv.Itoa(f.members)}
	for _, p := range f.people {
		out = append(out, "person:"+p.fact())
	}
	return out
}

func mfaRank(scope string) int {
	return map[string]int{authpolicy.MFANone: 0, authpolicy.MFAAdministrators: 1, authpolicy.MFAEveryone: 2}[scope]
}

func (s *Server) evidenceFor(v trustchange.View, validator authpolicy.Principal, authenticated time.Time, base trustchange.Base, current authpolicy.Revision) (trustchange.Evidence, trustFacts, error) {
	now := s.clock()
	candidate := revisionFor(v.Workspace, v.Candidate)
	f, err := s.trustFacts(s.caller, v.Workspace, &candidate, now)
	if err != nil {
		return trustchange.Evidence{}, trustFacts{}, err
	}
	ev := trustchange.Evidence{Digest: trustchange.EvidenceDigest(v.ChangeDigest, f.digestFacts(base))}
	login := trustchange.Item{Item: itemTestLogin, Result: resultVerified}
	if err := s.policy.Judge(validator, authpolicy.Authentication{Method: authpolicy.MethodPassword, At: authenticated}, candidate, now); err != nil {
		login.Result, login.Reason, login.Detail = resultBlocked, "test_login_failed", refusalReason(err)
	}
	recovery := trustchange.Item{Item: itemRecovery, Result: resultVerified}
	if ok, detail := f.recovery(); !ok {
		recovery.Result, recovery.Reason, recovery.Detail = resultBlocked, "recovery_unverified", detail
	}
	weakens := "unchanged"
	if mfaRank(v.Candidate.MFA) < mfaRank(current.MFA) {
		weakens = "weakens_mfa"
	}
	record := trustchange.Item{Item: itemRecord, Result: resultVerified}
	probe := v
	probe.Validation, probe.Activator = &trustchange.Validation{Digest: ev.Digest}, validator.User
	probe.Validator, probe.Approver = validator.User, validator.User
	next, ok := nextEpoch(base.Epoch, s.generation())
	switch {
	case !ok:
		record.Result, record.Reason = resultBlocked, "epoch_exhausted"
	case !decisionledger.Fits(s.trustDecision(probe, next, base.Policy+1, f.sessions)):
		record.Result, record.Reason = resultBlocked, "record_limit"
	}
	ev.Items = []trustchange.Item{login, recovery,
		{Item: itemIssuer, Result: resultNotApplicable, Reason: "local_password"},
		{Item: itemDiagnostics, Result: resultNotApplicable, Reason: "no_provider_connection"},
		{Item: itemExternal, Result: resultNotApplicable, Reason: "no_external_identity"},
		{Item: itemMFA, Result: resultInformation, Reason: weakens},
		{Item: itemAffected, Result: resultInformation, Detail: strconv.Itoa(f.members)},
		{Item: itemSessions, Result: resultInformation, Detail: strconv.Itoa(f.sessions)},
		record}
	ev.Blocked = login.Result == resultBlocked || recovery.Result == resultBlocked || record.Result == resultBlocked
	return ev, f, nil
}

func (s *Server) recomputed(v trustchange.View, actor identity.Verifier, base trustchange.Base) (string, error) {
	candidate := revisionFor(v.Workspace, v.Candidate)
	f, err := s.trustFacts(actor, v.Workspace, &candidate, s.clock())
	if err != nil {
		return "", err
	}
	return trustchange.EvidenceDigest(v.ChangeDigest, f.digestFacts(base)), nil
}

func refusalReason(err error) string {
	var refusal authpolicy.Refusal
	if errors.As(err, &refusal) {
		return refusal.Reason
	}
	return "invalid"
}

func (s *Server) holdsRight(user string, right gate.Action, now time.Time) bool {
	request := gate.Request{Action: right, Kind: gate.InstallationKind, Resource: gate.Resource{WorkspaceID: s.bootWorkspace}}
	return s.admin.ExplainInstallation(user, ConsoleClient, request, now).Allowed
}

func (s *Server) rightsOf(user string, now time.Time) rightsRow {
	allowed := func(right gate.Action) bool {
		request := gate.Request{Action: right, Kind: gate.InstallationKind, Resource: gate.Resource{WorkspaceID: s.bootWorkspace}}
		return s.admin.ExplainInstallation(user, ConsoleClient, request, now).Allowed
	}
	return rightsRow{Propose: allowed(gate.TrustPropose), Validate: allowed(gate.TrustValidate), Approve: allowed(gate.TrustApprove), Activate: allowed(gate.TrustActivate)}
}
