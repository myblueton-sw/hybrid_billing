package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

type factSource interface {
	member(user string) (identity.MemberInfo, bool)
	user(user string) identity.UserInfo
	grant(id string) (gate.GrantView, bool)
	grantsOf(subject string) []gate.GrantView
	clientGrants() []gate.GrantView
	delegationsOf(user string) []gate.DelegationView
	approvers() []string
}

type wholeFacts struct {
	s       *Server
	members map[string]identity.MemberInfo
	users   map[string]identity.UserInfo
	grants  []gate.GrantView
}

func (w wholeFacts) member(user string) (identity.MemberInfo, bool) {
	m, ok := w.members[user]
	return m, ok
}

func (w wholeFacts) user(user string) identity.UserInfo { return w.users[user] }

func (w wholeFacts) grant(id string) (gate.GrantView, bool) {
	for _, g := range w.grants {
		if g.ID == id {
			return g, true
		}
	}
	return gate.GrantView{}, false
}

func (w wholeFacts) grantsOf(subject string) []gate.GrantView {
	out := []gate.GrantView{}
	for _, g := range w.grants {
		if g.SubjectID == subject {
			out = append(out, g)
		}
	}
	return out
}

func (w wholeFacts) clientGrants() []gate.GrantView {
	out := []gate.GrantView{}
	for _, g := range w.grants {
		if g.ClientID == ConsoleClient {
			out = append(out, g)
		}
	}
	return out
}

func (w wholeFacts) delegationsOf(user string) []gate.DelegationView {
	out := []gate.DelegationView{}
	for _, d := range w.s.admin.Delegations() {
		if d.DelegatorID == user || d.DelegateID == user {
			out = append(out, d)
		}
	}
	return out
}

func (w wholeFacts) approvers() []string {
	seen := map[string]bool{}
	out := []string{}
	for _, g := range w.grants {
		if g.SubjectID != "" && gate.InstallationAction(g.Terms.Action) && !seen[g.SubjectID] {
			seen[g.SubjectID] = true
			out = append(out, g.SubjectID)
		}
	}
	return out
}
