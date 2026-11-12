package gate

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

type mUser struct {
	state   string
	epoch   uint64
	member  string
	joined  uint64
	version uint64
}

type mGrant struct {
	id      string
	subject string
	action  Action
	joined  uint64
	revoked bool
}

type mDelegation struct {
	id                   string
	from, to             string
	fromJoined, toJoined uint64
	revoked              bool
}

type mAnswer struct {
	answer  identity.Answer
	user    string
	epoch   uint64
	version uint64
	ended   *bool
}

type mModel struct {
	users       map[string]*mUser
	grants      []*mGrant
	delegations []*mDelegation
	clientOn    bool
	clientIssue bool
}

func (m *mModel) activeMember(id string, joined uint64) bool {
	found := m.users[id]
	return found != nil && found.state == "active" && found.member == "active" && found.joined == joined
}

func (m *mModel) holds(id string, action Action) bool {
	for _, grant := range m.grants {
		if grant.subject == id && grant.action == action && !grant.revoked && m.activeMember(id, grant.joined) {
			return true
		}
	}
	return false
}

func (m *mModel) anyGrant(id string) bool {
	for _, grant := range m.grants {
		if grant.subject == id && !grant.revoked && m.activeMember(id, grant.joined) {
			return true
		}
	}
	return false
}

func (m *mModel) issues(id string) (bool, bool) {
	if m.holds(id, Issue) {
		return true, false
	}
	for _, delegation := range m.delegations {
		if delegation.to == id && !delegation.revoked && m.activeMember(delegation.from, delegation.fromJoined) && m.activeMember(id, delegation.toJoined) &&
			m.holds(delegation.from, Issue) && m.anyGrant(id) {
			return true, true
		}
	}
	return false, false
}

func (m *mModel) current(held mAnswer) bool {
	found := m.users[held.user]
	return !*held.ended && found.state == "active" && found.epoch == held.epoch && found.member == "active" && found.version == held.version
}

func TestGateDecisionsAgreeWithAReferenceModel(t *testing.T) {
	r := rand.New(rand.NewSource(173))
	reached := map[string]int{}
	people := []string{"u0", "u1", "u2", "u3"}
	for history := 0; history < 80; history++ {
		f := setup(t)
		m := &mModel{users: map[string]*mUser{}}
		for _, id := range people {
			f.member(id)
			m.users[id] = &mUser{state: "active", epoch: 1, member: "active", joined: 1, version: 1}
		}
		f.must(second(f.a.PutClient(Client{ID: "app", State: Active}, 0)))
		m.clientOn = true
		clientVersion := uint64(1)
		answers := []mAnswer{}
		admin := func() identity.Verifier { return f.login("root") }
		for step := 0; step < 120; step++ {
			id := people[r.Intn(len(people))]
			found := m.users[id]
			op := r.Intn(16)
			if step < 45 {
				op = []int{0, 1, 3, 4, 3, 5, 5}[r.Intn(7)]
			}
			switch op {
			case 0, 1, 2:
				if found.state != "active" {
					continue
				}
				session := f.login(id)
				answer, err := f.b.Access(session, "w1")
				if (err == nil) != (found.member == "active") {
					t.Fatalf("history %d: access of %s: %v with membership %s", history, id, err, found.member)
				}
				if err == nil {
					ended := false
					answers = append(answers, mAnswer{answer: answer, user: id, epoch: found.epoch, version: found.version, ended: &ended})
					if r.Intn(6) == 0 {
						f.must(f.b.EndSession(session))
						ended = true
					}
				}
			case 3, 4:
				action := []Action{Issue, Issue, Deliver, RequestApproval, Approve}[r.Intn(5)]
				want := found.state == "active" && found.member == "active"
				grantID := f.next("g")
				err := f.a.AddGrant(Grant{ID: grantID, SubjectID: id, Terms: terms(action), Validity: always})
				if (err == nil) != want {
					t.Fatalf("history %d: grant for %s: %v; model %v", history, id, err, want)
				}
				if want {
					m.grants = append(m.grants, &mGrant{id: grantID, subject: id, action: action, joined: found.joined})
					reached["grant"]++
				} else {
					reached["grant refused"]++
				}
			case 5:
				to := people[r.Intn(len(people))]
				target := m.users[to]
				want := id != to && found.state == "active" && found.member == "active" && target.state == "active" && target.member == "active" &&
					m.holds(id, Issue) && m.anyGrant(to)
				delegationID := f.next("d")
				err := f.a.AddDelegation(Delegation{ID: delegationID, DelegatorID: id, DelegateID: to, Terms: terms(Issue), Validity: always})
				if (err == nil) != want {
					t.Fatalf("history %d: delegation %s to %s: %v; model %v", history, id, to, err, want)
				}
				if want {
					m.delegations = append(m.delegations, &mDelegation{id: delegationID, from: id, to: to, fromJoined: found.joined, toJoined: target.joined})
					reached["delegation"]++
				} else {
					reached["delegation refused"]++
				}
			case 6:
				if found.state == "active" {
					f.must(f.in.SuspendUser(operator, id, found.epoch))
					found.state = "suspended"
				} else if found.state == "suspended" {
					f.must(f.in.ReinstateUser(operator, id, found.epoch))
					found.state = "active"
				} else {
					continue
				}
				found.epoch++
				reached["user state"]++
			case 7:
				if found.state == "departed" || found.member == "removed" {
					continue
				}
				switch found.member {
				case "active":
					if r.Intn(3) == 0 {
						f.must(f.w.RemoveMembership(admin(), "w1", id, found.version))
						found.member = "removed"
						reached["removal"]++
					} else {
						f.must(f.w.SuspendMembership(admin(), "w1", id, found.version))
						found.member = "suspended"
					}
				case "suspended":
					f.must(f.w.ReinstateMembership(admin(), "w1", id, found.version))
					found.member = "active"
				}
				found.version++
				reached["membership state"]++
			case 8:
				if found.state != "active" || found.member != "removed" {
					continue
				}
				f.join(id, identity.Member)
				found.member = "active"
				found.joined++
				found.version++
				reached["invited again"]++
			case 9:
				if found.state == "departed" || r.Intn(8) != 0 {
					continue
				}
				f.must(f.in.DepartUser(operator, id, found.epoch))
				if found.member != "removed" {
					found.version++
				}
				found.state, found.member = "departed", "removed"
				found.epoch++
				reached["departure"]++
			case 10:
				if !m.clientIssue {
					f.must(f.a.AddGrant(Grant{ID: f.next("cg"), ClientID: "app", Terms: terms(Issue), Validity: always}))
					m.clientIssue = true
				} else {
					state := Suspended
					if !m.clientOn {
						state = Active
					}
					next, err := f.a.PutClient(Client{ID: "app", State: state}, clientVersion)
					f.must(err)
					clientVersion, m.clientOn = next, !m.clientOn
				}
			case 11:
				if len(m.grants) > 0 && r.Intn(2) == 0 {
					grant := m.grants[r.Intn(len(m.grants))]
					if !grant.revoked {
						f.must(f.a.RevokeGrant(grant.id, 1))
						grant.revoked = true
						reached["grant revoked"]++
					}
				} else if len(m.delegations) > 0 {
					delegation := m.delegations[r.Intn(len(m.delegations))]
					if !delegation.revoked {
						f.must(f.a.RevokeDelegation(delegation.id, 1))
						delegation.revoked = true
						reached["delegation revoked"]++
					}
				}
			default:
			}
			for index := 0; index < 6 && len(answers) > 0; index++ {
				held := answers[len(answers)-1-r.Intn(min(len(answers), 10))]
				current := m.current(held)
				allowed, delegated := m.issues(held.user)
				got := f.g.Decide(held.answer, "", ask(Issue))
				want := ReasonAllowed
				switch {
				case !current:
					want = ReasonIdentity
				case !allowed:
					want = ReasonNoGrant
				}
				if got.Reason != want || got.Allowed != (want == ReasonAllowed) || (got.Allowed && (len(got.Delegations) > 0) != delegated) {
					t.Fatalf("history %d step %d: decision for %s: %+v; model %s delegated %v", history, step, held.user, got, want, delegated)
				}
				reached["decide "+string(want)]++
				if got.Allowed && delegated {
					reached["delegated allow"]++
				}
				withClient := f.g.Decide(held.answer, "app", ask(Issue))
				wantClient := want
				if want == ReasonAllowed && !(m.clientOn && m.clientIssue) {
					wantClient = ReasonClient
				}
				if withClient.Reason != wantClient || withClient.Allowed != (wantClient == ReasonAllowed) {
					t.Fatalf("history %d step %d: decision through the client for %s: %+v; model %s", history, step, held.user, withClient, wantClient)
				}
				reached["client "+string(wantClient)]++
				author := people[r.Intn(len(people))]
				approval := f.g.CheckApproval(held.answer, author, ask(Approve))
				wantApproval := ReasonAllowed
				switch {
				case !current:
					wantApproval = ReasonIdentity
				case !m.holds(held.user, Approve):
					wantApproval = ReasonNoGrant
				case author == held.user:
					wantApproval = ReasonSelfApproval
				}
				if approval.Reason != wantApproval || approval.Allowed != (wantApproval == ReasonAllowed) {
					t.Fatalf("history %d step %d: approval by %s of %s: %+v; model %s", history, step, held.user, author, approval, wantApproval)
				}
				reached["approval "+string(wantApproval)]++
			}
		}
	}
	for _, category := range []string{"grant", "grant refused", "delegation", "delegation refused", "user state", "membership state", "removal", "invited again", "departure", "grant revoked", "delegation revoked",
		"decide allowed", "decide identity_not_current", "decide no_matching_grant", "delegated allow", "client allowed", "client client_not_allowed",
		"approval allowed", "approval identity_not_current", "approval no_matching_grant", "approval author_is_approver"} {
		if reached[category] < 20 {
			t.Errorf("the generator reached %q only %d times", category, reached[category])
		}
	}
	t.Logf("%v", reached)
}

func TestDecisionsThatStartAfterAChangeNeverAllow(t *testing.T) {
	cases := map[string]struct {
		actor  string
		change func(world)
	}{
		"the acting user is suspended":            {"ann", func(f world) { f.must(f.in.SuspendUser(operator, "ann", 1)) }},
		"the membership is removed":               {"ann", func(f world) { f.must(f.w.RemoveMembership(f.login("root"), "w1", "ann", 1)) }},
		"the delegator is suspended":              {"bob", func(f world) { f.must(f.in.SuspendUser(operator, "ann", 1)) }},
		"the delegator's membership is suspended": {"bob", func(f world) { f.must(f.w.SuspendMembership(f.login("root"), "w1", "ann", 1)) }},
	}
	for name, c := range cases {
		f := setup(t)
		f.member("ann")
		f.member("bob")
		f.grant("ann", Issue)
		f.grant("bob", RequestApproval)
		f.must(f.delegate("d", "ann", "bob"))
		answers := []identity.Answer{}
		for index := 0; index < 6; index++ {
			answers = append(answers, f.answer(c.actor))
		}
		changed := make(chan struct{})
		started := make(chan struct{}, len(answers))
		late := make(chan string, len(answers))
		var wg sync.WaitGroup
		for _, answer := range answers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				first := f.g.Decide(answer, "", ask(Issue))
				started <- struct{}{}
				if !first.Allowed {
					late <- fmt.Sprintf("not allowed before the change: %s", first.Reason)
					return
				}
				for waiting := true; waiting; {
					select {
					case <-changed:
						waiting = false
					default:
						f.g.Decide(answer, "", ask(Issue))
					}
				}
				for index := 0; index < 200; index++ {
					if got := f.g.Decide(answer, "", ask(Issue)); got.Allowed {
						late <- "allowed after the change returned"
						return
					}
				}
				late <- ""
			}()
		}
		for range answers {
			<-started
		}
		c.change(f)
		close(changed)
		wg.Wait()
		close(late)
		for problem := range late {
			if problem != "" {
				t.Errorf("%s: %s", name, problem)
			}
		}
	}
}
