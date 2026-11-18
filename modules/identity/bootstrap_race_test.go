package identity

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func contend(n int, run func(int) error) []error {
	gate := make(chan struct{})
	out := make([]error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-gate
			out[i] = run(i)
		}()
	}
	close(gate)
	wg.Wait()
	return out
}

func (f fixture) administrators(workspace string) []string {
	f.t.Helper()
	members, err := f.in.Members(operator, workspace)
	f.must(err)
	out := []string{}
	for _, m := range members {
		if m.Role == Administrator && m.Effective {
			out = append(out, m.User)
		}
	}
	return out
}

func (f fixture) armed() {
	f.t.Helper()
	f.must(f.in.ArmBootstrap(v("boot"), Validity{NotBefore: start, NotAfter: start.Add(time.Hour)}, "w1"))
}

func TestConcurrentBootstrapCreatesOneAdministrator(t *testing.T) {
	f := empty(t, testLimits)
	f.armed()
	results := contend(16, func(i int) error {
		id := fmt.Sprintf("u%d", i)
		return f.b.Bootstrap(v("boot"), id, triple(id), id+"@example.invalid")
	})
	won := ""
	for i, err := range results {
		if err == nil {
			if won != "" {
				t.Fatalf("two bootstraps succeeded: %s and u%d", won, i)
			}
			won = fmt.Sprintf("u%d", i)
			continue
		}
		f.refusal(fmt.Sprintf("bootstrap u%d", i), err, ReasonBootstrap)
	}
	if won == "" {
		t.Fatal("no bootstrap succeeded")
	}
	if users, _ := f.in.Users("", 2); len(users) != 1 || users[0].ID != won {
		t.Errorf("users after the race %+v", users)
	}
	if admins := f.administrators("w1"); len(admins) != 1 || admins[0] != won {
		t.Errorf("administrators after the race %v", admins)
	}
	if members, _ := f.in.Members(operator, "w1"); len(members) != 1 {
		t.Errorf("members after the race %+v", members)
	}
	if list, _ := f.in.WithoutAdministrator(); len(list) != 0 {
		t.Errorf("without administrator %v", list)
	}
}

func TestBootstrapRacesUserCreationAndArming(t *testing.T) {
	f := empty(t, testLimits)
	f.refusal("creating the bootstrap user ID before the bootstrap", f.in.CreateUser(operator, "root", triple("early-root"), ""), ReasonBootstrap)
	f.refusal("creating another user before the bootstrap", f.in.CreateUser(operator, "early", triple("early"), ""), ReasonBootstrap)
	f.armed()
	f.refusal("arming again before the bootstrap", f.in.ArmBootstrap(v("other"), Validity{NotBefore: start, NotAfter: start.Add(time.Hour)}, "w2"), ReasonBootstrap)

	results := contend(13, func(i int) error {
		switch {
		case i == 0:
			return f.b.Bootstrap(v("boot"), "root", triple("root"), "root@example.invalid")
		case i <= 4:
			return f.in.CreateUser(operator, "root", triple(fmt.Sprintf("other%d", i)), "")
		case i <= 8:
			id := fmt.Sprintf("plain%d", i)
			return f.in.CreateUser(operator, id, triple(id), "")
		default:
			return f.in.ArmBootstrap(v("other"), Validity{NotBefore: start, NotAfter: start.Add(time.Hour)}, "w2")
		}
	})
	f.must(results[0])
	for i := 1; i <= 4; i++ {
		if err := results[i]; ReasonOf(err) != ReasonBootstrap && !errors.Is(err, ErrConflict) {
			t.Errorf("creating the bootstrap user ID %d: %v", i, err)
		}
	}
	for i := 5; i <= 8; i++ {
		if err := results[i]; err != nil && ReasonOf(err) != ReasonBootstrap {
			t.Errorf("creating plain%d: %v", i, err)
		}
	}
	for i := 9; i < 13; i++ {
		f.refusal(fmt.Sprintf("arming %d", i), results[i], ReasonBootstrap)
	}
	if admins := f.administrators("w1"); len(admins) != 1 || admins[0] != "root" {
		t.Errorf("administrators %v", admins)
	}
	if found, err := f.b.Resolve(triple("root")); err != nil || found.User != "root" {
		t.Errorf("the bootstrap login %+v %v", found, err)
	}
	f.r.mu.Lock()
	for linked, record := range f.r.identities {
		if record.user == "root" && linked != triple("root") {
			t.Errorf("root is also linked to %+v", linked)
		}
	}
	rootLinks := f.r.users["root"].linked
	f.r.mu.Unlock()
	if rootLinks != 1 {
		t.Errorf("root has %d linked identities", rootLinks)
	}
	for i := 1; i <= 8; i++ {
		subject := fmt.Sprintf("other%d", i)
		if i > 4 {
			subject = fmt.Sprintf("plain%d", i)
		}
		found, err := f.b.Resolve(triple(subject))
		if err != nil {
			continue
		}
		if found.User == "root" {
			t.Errorf("login %s resolves to the bootstrap administrator", subject)
		}
		for _, admin := range f.administrators("w1") {
			if admin == found.User {
				t.Errorf("login %s resolves to an administrator", subject)
			}
		}
	}
	if list, _ := f.in.WithoutAdministrator(); len(list) != 0 {
		t.Errorf("without administrator %v", list)
	}
}

type bootState struct {
	users, identities, workspaces, attempts int
	consumed                                bool
}

func (f fixture) bootState() bootState {
	f.r.mu.Lock()
	defer f.r.mu.Unlock()
	return bootState{len(f.r.users), len(f.r.identities), len(f.r.memberships), f.r.boot.attempts, f.r.boot.consumed}
}

func TestBootstrapRefusesAnExistingUserOrLinkedLogin(t *testing.T) {
	existing := empty(t, testLimits)
	existing.armed()
	existing.r.mu.Lock()
	existing.r.users["someone"] = &user{kind: Human, state: Active, epoch: 1}
	existing.r.mu.Unlock()
	before := existing.bootState()
	existing.refusal("a bootstrap with a user present", existing.b.Bootstrap(v("boot"), "root", triple("root"), ""), ReasonBootstrap)
	existing.refusal("a wrong verifier with a user present", existing.b.Bootstrap(v("guess"), "root", triple("root"), ""), ReasonBootstrap)
	if after := existing.bootState(); after != before {
		t.Errorf("state changed with a user present: %+v then %+v", before, after)
	}

	linked := empty(t, testLimits)
	linked.armed()
	linked.r.mu.Lock()
	linked.r.identities[triple("root")] = &identityRecord{user: "ghost", linked: true, order: 1}
	linked.r.mu.Unlock()
	before = linked.bootState()
	linked.refusal("a bootstrap of a linked login", linked.b.Bootstrap(v("boot"), "root", triple("root"), ""), ReasonBootstrap)
	linked.refusal("a wrong verifier for a linked login", linked.b.Bootstrap(v("guess"), "root", triple("root"), ""), ReasonBootstrap)
	if after := linked.bootState(); after != before {
		t.Errorf("state changed with a linked login: %+v then %+v", before, after)
	}
	linked.r.mu.Lock()
	owner := linked.r.identities[triple("root")].user
	linked.r.mu.Unlock()
	if owner != "ghost" {
		t.Errorf("the linked login was taken over by %s", owner)
	}
	linked.must(linked.b.Bootstrap(v("boot"), "root", triple("fresh"), ""))
}

func TestConcurrentAcceptanceOfOneInvitation(t *testing.T) {
	f := setup(t)
	f.user("ann")
	f.must(f.w.Invite(f.login("root"), "w1", "i1", Member, v("i1"), f.window(), "ann"))
	ann := f.login("ann")
	results := contend(16, func(int) error { return f.w.Accept("w1", "i1", v("i1"), ann) })
	accepted := 0
	for i, err := range results {
		switch {
		case err == nil:
			accepted++
		case ReasonOf(err) != ReasonState && ReasonOf(err) != ReasonMember:
			t.Errorf("acceptance %d: %v", i, err)
		}
	}
	if accepted != 1 {
		t.Fatalf("%d acceptances", accepted)
	}
	if version := f.version("ann"); version != 1 {
		t.Errorf("membership version %d", version)
	}
}

func TestAnAcceptedInvitationIsNotReplayedAfterRemoval(t *testing.T) {
	f := setup(t)
	f.user("ann")
	f.must(f.w.Invite(f.login("root"), "w1", "i1", Member, v("i1"), f.window(), "ann"))
	ann := f.login("ann")
	f.must(f.w.Accept("w1", "i1", v("i1"), ann))
	f.must(f.w.RemoveMembership(f.login("root"), "w1", "ann", 1))
	if _, err := f.b.Resolve(triple("ann")); err != nil {
		t.Fatal(err)
	}
	f.refusal("access after the removal", second(f.b.Access(ann, "w1")), ReasonMembership)
	f.r.mu.Lock()
	before := *f.r.memberships["w1"]["ann"]
	f.r.mu.Unlock()
	f.refusal("replaying the accepted invitation", f.w.Accept("w1", "i1", v("i1"), ann), ReasonState)
	f.r.mu.Lock()
	after := *f.r.memberships["w1"]["ann"]
	f.r.mu.Unlock()
	if after != before || after.state != MembershipRemoved {
		t.Errorf("membership changed by the replay: %+v then %+v", before, after)
	}
	f.refusal("access after the replay", second(f.b.Access(ann, "w1")), ReasonMembership)
}

func TestAcceptanceNamingAnotherWorkspaceUnderContention(t *testing.T) {
	f := setup(t)
	f.user("ann")
	f.seed("w2", "adm2")
	f.must(f.w.Invite(f.login("root"), "w1", "i1", Member, v("i1"), f.window(), "ann"))
	ann := f.login("ann")
	results := contend(9, func(i int) error {
		if i == 0 {
			return f.w.Accept("w1", "i1", v("i1"), ann)
		}
		return f.w.Accept("w2", "i1", v("i1"), ann)
	})
	f.must(results[0])
	for i := 1; i < 9; i++ {
		f.refusal(fmt.Sprintf("acceptance naming w2, %d", i), results[i], ReasonUnknown)
	}
	f.r.mu.Lock()
	there := f.r.memberships["w2"]["ann"]
	f.r.mu.Unlock()
	if there != nil {
		t.Errorf("a membership in w2 from w1's invitation %+v", there)
	}
	f.refusal("access in w2", second(f.b.Access(ann, "w2")), ReasonMembership)
}

func TestAUserOfAnotherWorkspaceCannotTakeAnInvitation(t *testing.T) {
	f := setup(t)
	f.user("ann")
	f.user("bob")
	f.seed("w2", "adm2")
	f.must(f.w.Invite(f.login("adm2"), "w2", "j1", Member, v("j1"), f.window(), "bob"))
	f.must(f.w.Accept("w2", "j1", v("j1"), f.login("bob")))
	f.must(f.w.Invite(f.login("root"), "w1", "i1", Administrator, v("i1"), f.window(), "ann"))
	for _, other := range []string{"bob", "adm2"} {
		session := f.login(other)
		f.refusal(other+" with the right verifier", f.w.Accept("w1", "i1", v("i1"), session), ReasonNotInvitee)
		for i := 0; i <= testLimits.Attempts; i++ {
			f.refusal(fmt.Sprintf("%s with a wrong verifier %d", other, i), f.w.Accept("w1", "i1", v("guess"), session), ReasonNotInvitee)
		}
	}
	f.r.mu.Lock()
	found := *f.r.invitations["i1"]
	f.r.mu.Unlock()
	if found.state != invitationPending || found.attempts != 0 {
		t.Errorf("the invitation after other users' attempts %+v", found)
	}
	f.must(f.w.Accept("w1", "i1", v("i1"), f.login("ann")))
	if admins := f.administrators("w1"); len(admins) != 2 {
		t.Errorf("administrators %v", admins)
	}
	for _, other := range []string{"bob", "adm2"} {
		f.refusal(other+" in w1", second(f.b.Access(f.login(other), "w1")), ReasonMembership)
	}
}
