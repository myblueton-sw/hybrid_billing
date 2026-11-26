package server

import (
	"errors"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

var errFactsBeyondLimit = errors.New("request facts exceed the lookup limit")

type boundedFacts struct {
	s       *Server
	actor   identity.Verifier
	at      time.Time
	limit   int
	people  map[string]identity.Person
	known   map[string]bool
	client  []gate.GrantView
	holders []string
	loaded  map[string]bool
	failure *error
}

func (s *Server) boundedFacts(actor identity.Verifier) (accessFacts, *error) {
	failure := new(error)
	now := s.clock()
	return accessFacts{s: s, now: now, src: &boundedFacts{s: s, actor: actor, at: now, limit: s.cfg.Access.ServiceQueryScan, people: map[string]identity.Person{},
		known: map[string]bool{}, loaded: map[string]bool{}, failure: failure}}, failure
}

func (b *boundedFacts) fail(err error) {
	if *b.failure == nil {
		*b.failure = err
	}
}

func (b *boundedFacts) person(user string) (identity.Person, bool) {
	if b.known[user] {
		found, ok := b.people[user]
		return found, ok
	}
	b.known[user] = true
	if b.s.faulty("people") {
		b.fail(identity.ErrUnavailable)
		return identity.Person{}, false
	}
	found, err := b.s.identity.Workspace().People(b.actor, b.s.bootWorkspace, []string{user})
	if err != nil {
		if !errors.Is(err, identity.ErrInvalid) {
			b.fail(err)
		}
		return identity.Person{}, false
	}
	if len(found) == 1 {
		b.people[user] = found[0]
		return found[0], true
	}
	return identity.Person{}, false
}

func (b *boundedFacts) member(user string) (identity.MemberInfo, bool) {
	found, ok := b.person(user)
	if !ok || found.Member == nil {
		return identity.MemberInfo{}, false
	}
	return *found.Member, true
}

func (b *boundedFacts) user(user string) identity.UserInfo {
	found, _ := b.person(user)
	return found.User
}

func (b *boundedFacts) grant(id string) (gate.GrantView, bool) {
	return b.s.admin.GrantAt(id, b.at)
}

func (b *boundedFacts) grantsOf(subject string) []gate.GrantView {
	rows, more := b.s.admin.GrantsOfSubject(subject, b.limit, b.at)
	if more {
		b.fail(errFactsBeyondLimit)
	}
	return rows
}

func (b *boundedFacts) clientGrants() []gate.GrantView {
	if !b.loaded["client"] {
		b.loaded["client"] = true
		rows, more := b.s.admin.GrantsOfClient(ConsoleClient, b.limit, b.at)
		if more {
			b.fail(errFactsBeyondLimit)
		}
		b.client = rows
	}
	return b.client
}

func (b *boundedFacts) delegationsOf(user string) []gate.DelegationView {
	rows, more := b.s.admin.DelegationsOfUser(user, b.limit, b.at)
	if more {
		b.fail(errFactsBeyondLimit)
	}
	return rows
}

func (b *boundedFacts) approvers() []string {
	if !b.loaded["holders"] {
		b.loaded["holders"] = true
		rows, more := b.s.admin.InstallationGrants(gate.AdministerInstallation, b.limit, b.at)
		if more {
			b.fail(errFactsBeyondLimit)
		}
		seen := map[string]bool{}
		for _, g := range rows {
			if g.SubjectID != "" && !seen[g.SubjectID] {
				seen[g.SubjectID] = true
				b.holders = append(b.holders, g.SubjectID)
			}
		}
	}
	return b.holders
}
