package server

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/accesschange"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

type priorFacts struct {
	f           accessFacts
	delegations []gate.DelegationView
}

func (p priorFacts) person(user string) string {
	m, u := p.f.members[user], p.f.users[user]
	return strings.Join([]string{user, strconv.FormatUint(m.Version, 10), string(m.Role), memberState(m), string(m.UserState), strconv.FormatUint(u.Epoch, 10),
		string(u.State), string(u.Kind)}, "/")
}

func (p priorFacts) holdsAdmin(user string) bool {
	for _, g := range p.f.grants {
		if g.SubjectID == user && g.Terms.Action == gate.AdministerInstallation && g.Effective && g.Terms.Scope.WorkspaceID == p.f.s.bootWorkspace {
			return true
		}
	}
	return false
}

func (p priorFacts) eligible(user string) bool {
	m, ok := p.f.members[user]
	return ok && m.Role == identity.Administrator && m.Effective && m.UserState == identity.Active && p.f.users[user].Kind == identity.Human && p.holdsAdmin(user)
}

func (p priorFacts) ceilingFor(c accesschange.Change) (gate.GrantView, bool) {
	if governedGrantTerms(gate.Action(c.Action), c.ResourceKind, c.Fields) {
		return gate.GrantView{}, false
	}
	var found gate.GrantView
	for _, g := range p.f.grants {
		if g.ClientID == ConsoleClient && g.Active && !gate.InstallationAction(g.Terms.Action) && g.Terms.Scope.WorkspaceID == p.f.s.bootWorkspace && fullScope(g.Terms) &&
			string(g.Terms.Action) == c.Action && g.Terms.Kind == c.ResourceKind && len(c.Fields) > 0 == (len(g.Terms.Fields) > 0) && covers(g.Terms.Fields, c.Fields) &&
			g.Validity.NotAfter.After(found.Validity.NotAfter) {
			found = g
		}
	}
	return found, found.ID != ""
}

func (p priorFacts) equivalent(c accesschange.Change) string {
	for _, g := range p.f.grants {
		if g.SubjectID == c.Subject && !g.Closed && p.f.now.Before(g.Validity.NotAfter) && string(g.Terms.Action) == c.Action && g.Terms.Kind == c.ResourceKind &&
			strings.Join(g.Terms.Fields, ",") == strings.Join(c.Fields, ",") && sameCustomer(g.Terms.Scope.Customer, c.Customer) {
			return g.ID
		}
	}
	return noneFact
}

func (p priorFacts) mayEnd(holder string) []string {
	out := []string{}
	for _, d := range p.delegations {
		if d.Effective && !d.Closed && (d.DelegatorID == holder || d.DelegateID == holder) {
			out = append(out, d.ID)
		}
	}
	return out
}

func (p priorFacts) basis(c accesschange.Change, proposer string) string {
	parts := []string{"proposer:" + p.person(proposer) + "/" + strconv.FormatBool(p.holdsAdmin(proposer))}
	if c.Kind == accesschange.ChangeAdd {
		ceiling, ok := p.ceilingFor(c)
		parts = append(parts, "ceiling:"+grantFact(ceiling, ok)+"/"+stamp(ceiling.Validity.NotAfter), "subject:"+p.person(c.Subject), "equivalent:"+p.equivalent(c),
			"covered:"+strconv.FormatBool(p.f.covered(proposer, c)))
	} else {
		g, ok := p.f.s.grantView(c.Grant)
		parts = append(parts, "grant:"+grantFact(g, ok))
	}
	return strings.Join(parts, ";")
}

func (p priorFacts) standing(v accesschange.View, approver string) accesschange.Standing {
	allowed := p.eligible(approver)
	if v.Change.Kind == accesschange.ChangeAdd {
		allowed = allowed && !governedGrantTerms(gate.Action(v.Change.Action), v.Change.ResourceKind, v.Change.Fields) && p.f.covered(approver, v.Change) && p.f.covered(v.Proposer, v.Change)
	} else if g, ok := p.f.s.grantView(v.Change.Grant); ok && protectedGrant(g) {
		allowed = false
	}
	return accesschange.Standing{Basis: p.basis(v.Change, v.Proposer), Allowed: allowed}
}

func (p priorFacts) approvers(v accesschange.View) int {
	n := 0
	for user := range p.f.members {
		if p.eligible(user) && user != v.Proposer && (v.Change.Kind == accesschange.ChangeRevoke || user != v.Change.Subject) {
			n++
		}
	}
	return n
}

func compareFacts(t *testing.T, f world, stage string) {
	t.Helper()
	facts, err := f.s.accessFacts(f.as(AdminLogin).verifier(f))
	if err != nil {
		t.Fatalf("%s: %v", stage, err)
	}
	prior := priorFacts{f: facts, delegations: f.s.admin.Delegations()}
	people := []string{"nobody"}
	for user := range facts.members {
		people = append(people, user)
	}
	for user := range facts.users {
		if _, member := facts.members[user]; !member {
			people = append(people, user)
		}
	}
	changes := []accesschange.Change{}
	for _, g := range facts.grants {
		for _, customer := range []string{"all", "c1"} {
			for _, subject := range people {
				changes = append(changes, accesschange.Change{Kind: accesschange.ChangeAdd, Subject: subject, Action: string(g.Terms.Action), ResourceKind: g.Terms.Kind,
					Fields: append([]string{}, g.Terms.Fields...), Customer: customer})
			}
		}
		fresh, _ := f.s.grantView(g.ID)
		if got, ok := facts.src.grant(g.ID); !ok || fmt.Sprint(got) != fmt.Sprint(fresh) {
			t.Errorf("%s: grant %s by identifier differs", stage, g.ID)
		}
	}
	if _, ok := facts.src.grant("no-such-grant"); ok {
		t.Errorf("%s: an unknown grant was found", stage)
	}
	eligible := 0
	for _, user := range people {
		if facts.person(user) != prior.person(user) || facts.holdsAdmin(user) != prior.holdsAdmin(user) || facts.eligible(user) != prior.eligible(user) {
			t.Errorf("%s: %s: person, administrator grant or eligibility differs", stage, user)
		}
		if strings.Join(facts.mayEnd(user), " ") != strings.Join(prior.mayEnd(user), " ") {
			t.Errorf("%s: %s: delegations that may end: %v want %v", stage, user, facts.mayEnd(user), prior.mayEnd(user))
		}
		if prior.eligible(user) {
			eligible++
		}
	}
	if eligible == 0 {
		t.Fatalf("%s: nobody is eligible; the comparison would prove nothing", stage)
	}
	for _, c := range changes {
		got, ok := facts.ceilingFor(c)
		want, wantOK := prior.ceilingFor(c)
		if ok != wantOK || got.ID != want.ID || facts.equivalent(c) != prior.equivalent(c) {
			t.Errorf("%s: %+v: ceiling %q/%v want %q/%v, equivalent %q want %q", stage, c, got.ID, ok, want.ID, wantOK, facts.equivalent(c), prior.equivalent(c))
		}
	}
	for _, c := range changes[:min(len(changes), 400)] {
		for _, proposer := range people {
			if facts.basis(c, proposer) != prior.basis(c, proposer) {
				t.Errorf("%s: basis of %+v proposed by %s differs", stage, c, proposer)
			}
		}
	}
	you := SampleAdmin
	for _, v := range f.s.changes.Requests(f.s.bootWorkspace) {
		for _, approver := range people {
			if got, want := facts.standing(v, approver), prior.standing(v, approver); got != want {
				t.Errorf("%s: standing of %s for %s: %+v want %+v", stage, approver, v.ID, got, want)
			}
		}
		row := facts.changeRequestRow(v, you)
		if v.State == accesschange.StatePending && row.EligibleApprovers != prior.approvers(v) {
			t.Errorf("%s: %s: %d eligible approvers, want %d", stage, v.ID, row.EligibleApprovers, prior.approvers(v))
		}
		for _, proposer := range people {
			for _, kind := range []string{accesschange.ChangeAdd, accesschange.ChangeRevoke} {
				probe := v
				probe.State, probe.Proposer, probe.Change.Kind = accesschange.StatePending, proposer, kind
				if got := facts.changeRequestRow(probe, you).EligibleApprovers; got != prior.approvers(probe) {
					t.Errorf("%s: %s proposed by %s as %s: %d eligible approvers, want %d", stage, v.ID, proposer, kind, got, prior.approvers(probe))
				}
			}
		}
	}
}

func TestFactsThroughTheSourceEqualTheFactsReadDirectly(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	compareFacts(t, f, "the sample")
	f.manyPermissions()
	compareFacts(t, f, "many grants and delegations")
	asked := f.requestGrant(root, SampleMember, nil)
	if asked.status != 200 {
		t.Fatalf("ask for a grant %s", asked.raw)
	}
	compareFacts(t, f, "a pending grant request")
	approved := f.approveRequest(second, asked)
	if approved.status != 200 {
		t.Fatalf("approve %s", approved.raw)
	}
	if got := f.run(root, "access.request_revoke", fmt.Sprint(approved.body["fields"].(map[string]any)["grant"]), nil); got.status != 200 {
		t.Fatalf("ask to revoke %s", got.raw)
	}
	compareFacts(t, f, "a pending revocation request")
	f.must(f.s.admin.RevokeGrant("q-03", 1))
	f.must(f.s.admin.RevokeDelegation("qd-01", 1))
	compareFacts(t, f, "a revoked grant and delegation")
	f.advance(30 * time.Hour)
	compareFacts(t, f, "after some grants expired")
	if got := f.run(f.as(AdminLogin), "identity.suspend_membership", SampleMember, nil); got.status != 200 {
		t.Fatalf("suspend %s", got.raw)
	}
	compareFacts(t, f, "a suspended member")
}
