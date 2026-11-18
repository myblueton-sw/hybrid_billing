package server

import (
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/accesschange"
	"github.com/myblueton-sw/hybrid_billing/modules/bulkrevoke"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/installer"
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
	count     func(s *Server, user string) int
}

var proposalStores = []proposalStore{
	{"trust_changes", "trust", true, func(s *Server, user string) int {
		n := 0
		for _, v := range s.trust.List(s.bootWorkspace) {
			if v.Proposer == user && trustchange.Open(v.State) {
				n++
			}
		}
		return n
	}},
	{"access_requests", "access", true, func(s *Server, user string) int {
		n := 0
		for _, v := range s.changes.Requests(s.bootWorkspace) {
			if v.Proposer == user && v.State == accesschange.StatePending {
				n++
			}
		}
		return n
	}},
	{"retention_proposals", "retention", false, func(s *Server, user string) int {
		n := 0
		for _, e := range s.ret.entries {
			if e.view.Proposer == user && e.view.State == retention.ProposalPending {
				n++
			}
		}
		return n
	}},
	{"recovery_proposals", "recovery", false, func(s *Server, user string) int {
		n := 0
		for _, e := range s.rec.entries {
			if e.view.Proposer == user && e.view.State == recovery.ProposalPending {
				n++
			}
		}
		return n
	}},
	{"installation_proposals", "installation", false, func(s *Server, user string) int {
		n := 0
		for _, e := range s.install.entries {
			if e.view.Proposer == user && e.view.State == installer.ProposalPending {
				n++
			}
		}
		return n
	}},
	{"integration_proposals", "integrations", false, func(s *Server, user string) int {
		n := 0
		for _, e := range s.integ.entries {
			if e.view.Proposer == user && e.view.State == plugins.ProposalPending {
				n++
			}
		}
		for _, e := range s.integ.subscriptions {
			if v, err := s.subscriptionOf(e.id); err == nil && e.proposer == user && v.State == webhooks.SubscriptionProposed {
				n++
			}
		}
		for _, requests := range []map[string]letterRequest{s.integ.redeliver, s.integ.dismiss} {
			for _, r := range requests {
				if r.by == user {
					n++
				}
			}
		}
		return n
	}},
	{"relationship_plans", "relationship", false, func(s *Server, user string) int {
		n := 0
		for _, m := range s.plans.byKey {
			if m.proposer == user && m.closed == "" && m.activated == 0 {
				n++
			}
		}
		return n
	}},
	{"release_evidence_proposals", "release_evidence", false, func(s *Server, user string) int {
		n := 0
		for _, e := range s.ev.entries {
			if e.proposer == user && e.state == entryPending {
				n++
			}
		}
		return n
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

func (s *Server) classify(caller string, ids []string, effect bulkrevoke.Effect, r roster, now time.Time) ([]bulkrevoke.Target, []bulkrevoke.Exclusion) {
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

func (s *Server) pickupsOf(user string) int {
	own := map[identity.Verifier]bool{}
	for _, v := range s.sessionsOf(user) {
		own[v] = true
	}
	n := 0
	for _, p := range s.dir.pickups {
		if p.target == user || own[p.session] {
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

func (s *Server) impact(targets []bulkrevoke.Target, r roster, now time.Time) bulkrevoke.Impact {
	out := bulkrevoke.Impact{ComputedAt: now, People: len(targets)}
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
		out.Sessions += len(s.sessionsOf(t.User))
		out.Pickups += s.pickupsOf(t.User)
	}
	out.ServiceUsers = s.ownedServices(leaving)
	anchors := s.anchors(r.members, now)
	for user := range leaving {
		delete(anchors, user)
	}
	out.AnchorsAfter = len(anchors)
	for _, store := range proposalStores {
		count := 0
		for _, t := range targets {
			count += store.count(s, t.User)
		}
		out.Pending = append(out.Pending, bulkrevoke.PendingStore{Store: store.name, Count: count, Rechecked: store.rechecked})
	}
	return out
}
