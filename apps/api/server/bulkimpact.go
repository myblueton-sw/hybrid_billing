package server

import (
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/accesschange"
	"github.com/myblueton-sw/hybrid_billing/modules/bulkrevoke"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/installer"
	"github.com/myblueton-sw/hybrid_billing/modules/licensemode"
	"github.com/myblueton-sw/hybrid_billing/modules/plugins"
	"github.com/myblueton-sw/hybrid_billing/modules/recovery"
	"github.com/myblueton-sw/hybrid_billing/modules/retention"
	"github.com/myblueton-sw/hybrid_billing/modules/trustchange"
	"github.com/myblueton-sw/hybrid_billing/modules/webhooks"
)

type proposalStore struct {
	name      string
	prefix    string
	rechecked bool
	each      func(s *Server, visit func(proposer string))
}

func (p proposalStore) count(s *Server, user string) int {
	return p.tally(s)[user]
}

func (p proposalStore) tally(s *Server) map[string]int {
	out := map[string]int{}
	p.each(s, func(proposer string) { out[proposer]++ })
	return out
}

var proposalStores = []proposalStore{
	{"trust_changes", "trust", true, func(s *Server, visit func(string)) {
		for _, v := range s.trust.List(s.bootWorkspace) {
			if trustchange.Open(v.State) {
				visit(v.Proposer)
			}
		}
	}},
	{"access_requests", "access", true, func(s *Server, visit func(string)) {
		for _, v := range s.changes.Requests(s.bootWorkspace) {
			if v.State == accesschange.StatePending {
				visit(v.Proposer)
			}
		}
	}},
	{"retention_proposals", "retention", false, func(s *Server, visit func(string)) {
		for _, e := range s.ret.entries {
			if e.view.State == retention.ProposalPending {
				visit(e.view.Proposer)
			}
		}
	}},
	{"recovery_proposals", "recovery", false, func(s *Server, visit func(string)) {
		for _, e := range s.rec.entries {
			if e.view.State == recovery.ProposalPending {
				visit(e.view.Proposer)
			}
		}
	}},
	{"installation_proposals", "installation", false, func(s *Server, visit func(string)) {
		for _, e := range s.install.entries {
			if e.view.State == installer.ProposalPending {
				visit(e.view.Proposer)
			}
		}
	}},
	{"integration_proposals", "integrations", true, func(s *Server, visit func(string)) {
		for _, e := range s.integ.entries {
			if e.view.State == plugins.ProposalPending {
				visit(e.view.Proposer)
			}
		}
		for _, e := range s.integ.subscriptions {
			if v, err := s.subscriptionOf(e.id); err == nil && v.State == webhooks.SubscriptionProposed {
				visit(e.proposer)
			}
			if e.resume != nil {
				visit(e.resume.by)
			}
		}
		for _, requests := range []map[string]letterRequest{s.integ.redeliver, s.integ.dismiss} {
			for _, r := range requests {
				visit(r.by)
			}
		}
	}},
	{"relationship_plans", "relationship", false, func(s *Server, visit func(string)) {
		for _, m := range s.plans.byKey {
			if m.closed == "" && m.activated == 0 {
				visit(m.proposer)
			}
		}
	}},
	{"release_evidence_proposals", "release_evidence", false, func(s *Server, visit func(string)) {
		for _, e := range s.ev.entries {
			if e.state == entryPending {
				visit(e.proposer)
			}
		}
	}},
}

type fallibleProposalStore struct {
	name      string
	prefix    string
	rechecked bool
	proposers func(*Server) ([]string, error)
}

var fallibleProposalStores = []fallibleProposalStore{
	{"license_key_changes", "licenses", false, func(s *Server) ([]string, error) {
		if s.licenseFailed {
			return nil, licensemode.ErrStorage
		}
		if s.license == nil {
			return nil, nil
		}
		return s.license.PendingProposers()
	}},
}

func memberState(m identity.MemberInfo) string {
	if m.State == identity.MembershipActive && !m.Effective {
		return "ended"
	}
	return string(m.State)
}

type roster struct {
	users   map[string]identity.UserInfo
	members map[string]identity.MemberInfo
}

func (s *Server) everyone() (roster, error) {
	members, err := s.roster(s.caller)
	if err != nil {
		return roster{}, err
	}
	out := roster{users: map[string]identity.UserInfo{}, members: members}
	for after := ""; ; {
		page, err := s.users(after)
		if err != nil {
			return roster{}, err
		}
		for _, u := range page {
			out.users[u.ID] = u
		}
		if len(page) < 100 {
			return out, nil
		}
		after = page[len(page)-1].ID
	}
}

func (s *Server) classify(caller string, ids []string, effect bulkrevoke.Effect, r roster, now time.Time, still func(string) bool) ([]bulkrevoke.Target, []bulkrevoke.Exclusion) {
	targets, exclusions := []bulkrevoke.Target{}, []bulkrevoke.Exclusion{}
	for _, id := range ids {
		login, _ := s.dir.loginOf(id)
		u, known := r.users[id]
		m, member := r.members[id]
		reason := ""
		switch {
		case id == caller:
			reason = bulkrevoke.ExcludedSelf
		case !known || u.Kind != identity.Human || u.State == identity.Departed || effect == bulkrevoke.SuspendUser && !member:
			reason, login = bulkrevoke.ExcludedNotFound, ""
		case still != nil && !still(id):
			reason = bulkrevoke.ExcludedNoLongerMatches
		case !member:
			reason = bulkrevoke.ExcludedNoMembership
		case effect == bulkrevoke.SuspendMembership && !m.Effective, effect == bulkrevoke.SuspendUser && u.State != identity.Active:
			reason = bulkrevoke.ExcludedAlreadyRevoked
		}
		if reason != "" {
			exclusions = append(exclusions, bulkrevoke.Exclusion{User: id, Login: login, Reason: reason})
			continue
		}
		targets = append(targets, bulkrevoke.Target{User: id, Login: login, Role: string(m.Role), Until: m.Until, Version: m.Version, Epoch: u.Epoch,
			UserState: string(u.State), MembershipState: memberState(m), Administrator: s.dir.holds(id, now)})
	}
	return targets, exclusions
}

func (s *Server) mayChangeRole(caller string, role string, now time.Time) bool {
	return s.dir.holds(caller, now) && (role == string(identity.Member) || role == string(identity.Administrator))
}

func (s *Server) sessionsOf(user string) []identity.Verifier {
	login, _ := s.dir.loginOf(user)
	return s.sessions.byLogin(login)
}

type sideIndex struct {
	sessions map[string][]identity.Verifier
	owner    map[identity.Verifier]string
	byTarget map[string][]pickup
	byLogin  map[string][]pickup
}

func (s *Server) sideIndex() sideIndex {
	out := sideIndex{sessions: s.sessions.logins(), owner: map[identity.Verifier]string{}, byTarget: map[string][]pickup{}, byLogin: map[string][]pickup{}}
	for login, list := range out.sessions {
		for _, v := range list {
			out.owner[v] = login
		}
	}
	for _, p := range s.dir.pickups {
		out.byTarget[p.target] = append(out.byTarget[p.target], p)
		if login, ok := out.owner[p.session]; ok {
			out.byLogin[login] = append(out.byLogin[login], p)
		}
	}
	return out
}

func (x sideIndex) pickupsOf(user, login string) int {
	n := len(x.byTarget[user])
	for _, p := range x.byLogin[login] {
		if p.target != user {
			n++
		}
	}
	return n
}

func (s *Server) ownedServices(targets map[string]bool) int {
	n := 0
	for after := ""; ; {
		page, _ := s.identity.Workspace().ServiceUsers(s.caller, s.bootWorkspace, after, 100)
		for _, v := range page {
			if targets[v.Owner] && v.State != identity.Departed {
				n++
			}
		}
		if len(page) < 100 {
			return n
		}
		after = page[len(page)-1].ID
	}
}

func (s *Server) impact(targets []bulkrevoke.Target, r roster, now time.Time) (bulkrevoke.Impact, error) {
	out := bulkrevoke.Impact{ComputedAt: now, People: len(targets)}
	side := s.sideIndex()
	leaving := map[string]bool{}
	for _, t := range targets {
		leaving[t.User] = true
		if t.Role == string(identity.Administrator) {
			out.Administrators++
		} else {
			out.Members++
		}
		if t.Administrator {
			out.InstallationAdministrators++
		}
		for _, right := range trustRights {
			if s.holdsRight(t.User, right, now) {
				out.TrustHolders++
				break
			}
		}
		if s.holdsRight(t.User, gate.RevokeAccessInBulk, now) {
			out.BulkHolders++
		}
		login, _ := s.dir.loginOf(t.User)
		out.Sessions += len(side.sessions[login])
		out.Pickups += side.pickupsOf(t.User, login)
	}
	out.ServiceUsers = s.ownedServices(leaving)
	anchors := s.anchors(r.members, now)
	for user := range leaving {
		delete(anchors, user)
	}
	out.AnchorsAfter = len(anchors)
	for _, store := range proposalStores {
		tally, count := store.tally(s), 0
		for _, t := range targets {
			count += tally[t.User]
		}
		out.Pending = append(out.Pending, bulkrevoke.PendingStore{Store: store.name, Count: count, Rechecked: store.rechecked})
	}
	for _, store := range fallibleProposalStores {
		proposers, err := store.proposers(s)
		if err != nil {
			return bulkrevoke.Impact{}, err
		}
		count := 0
		for _, user := range proposers {
			if leaving[user] {
				count++
			}
		}
		out.Pending = append(out.Pending, bulkrevoke.PendingStore{Store: store.name, Count: count, Rechecked: store.rechecked})
	}
	return out, nil
}
