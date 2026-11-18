package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func activationBody(token, login string) string {
	return `{"token":"` + token + `","password":"` + pw + `","login":"` + login + `","email":"` + login + `"}`
}

func noneLogin(login string) string {
	return `{"workspace":"` + EmptyWorkspace + `","login":"` + login + `","password":"` + pw + `"}`
}

func (f world) known(login string) bool {
	f.s.domain.RLock()
	defer f.s.domain.RUnlock()
	_, err := f.s.identity.Boundary().Resolve(LoginTriple(login))
	return err == nil || f.s.passwords.Has(LoginKey(login)) || f.s.dir.taken[login]
}

func (f world) people() []string {
	f.s.domain.RLock()
	defer f.s.domain.RUnlock()
	out := []string{}
	for _, p := range f.s.dir.people {
		out = append(out, p.login)
	}
	return out
}

func (f world) members() []identity.MemberInfo {
	f.t.Helper()
	f.s.domain.RLock()
	defer f.s.domain.RUnlock()
	list, err := f.s.identity.Installation().Members(identity.Operator{Reference: "test", Reason: "test"}, f.s.bootWorkspace)
	if err != nil {
		f.t.Fatal(err)
	}
	return list
}

func (f world) closed() world {
	f.t.Cleanup(func() { _ = f.s.Close() })
	return f
}

func (f world) session(cookie *http.Cookie) reply {
	f.t.Helper()
	return f.do("GET", "/api/v1/session", "", func(r *http.Request) { r.Header.Set(clientHeader, ConsoleClient); r.AddCookie(cookie) })
}

func (f world) firstAdmin(login string) actor {
	f.t.Helper()
	if got := f.auth("redeem", activationBody(f.links()[""], login), f.pre()); got.status != 204 {
		f.t.Fatalf("activate %+v", got)
	}
	cookie, csrf, err := f.s.openFor(LoginTriple(login), EmptyWorkspace, false)
	if err != nil {
		f.t.Fatal(err)
	}
	return actor{cookie, csrf}
}

func TestConcurrentActivationCreatesOneAdministrator(t *testing.T) {
	f := setup(t, SeedNone).ready()
	token := f.links()[""]
	pre := f.pre()
	replies := make([]reply, 8)
	gate := make(chan struct{})
	var wg sync.WaitGroup
	for i := range replies {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-gate
			replies[i] = f.auth("redeem", activationBody(token, fmt.Sprintf("first%d@example.invalid", i)), pre)
		}()
	}
	close(gate)
	wg.Wait()
	winner := ""
	for i, got := range replies {
		login := fmt.Sprintf("first%d@example.invalid", i)
		if got.status == 204 {
			if winner != "" {
				t.Fatalf("two activations succeeded: %s and %s", winner, login)
			}
			winner = login
			continue
		}
		f.code(t, "losing activation "+login, got, 422, CodeLinkInvalid)
	}
	if winner == "" {
		t.Fatal("no activation succeeded")
	}
	if people := f.people(); len(people) != 1 || people[0] != winner {
		t.Errorf("directory %v, winner %s", people, winner)
	}
	if users, _ := f.s.identity.Installation().Users("", 10); len(users) != 1 || users[0].ID != EmptyAdmin {
		t.Errorf("users %+v", users)
	}
	if members := f.members(); len(members) != 1 || members[0].User != EmptyAdmin || members[0].Role != identity.Administrator || !members[0].Effective {
		t.Errorf("boot workspace members %+v", members)
	}
	for i := range replies {
		if login := fmt.Sprintf("first%d@example.invalid", i); login != winner {
			if f.known(login) {
				t.Errorf("losing login %s left a record", login)
			}
			f.code(t, "losing login signs in", f.auth("login", noneLogin(login), pre), 401, string(commands.Unauthenticated))
		}
	}
	if _, err := os.Stat(filepath.Join(f.s.cfg.Data, ActivationFile)); !os.IsNotExist(err) {
		t.Errorf("activation file after activation: %v", err)
	}
	f.code(t, "peek after activation", f.auth("link", `{"token":"`+token+`"}`, pre), 422, CodeLinkInvalid)
	if got := f.auth("login", noneLogin(winner), pre); got.status != 200 || got.body["role"] != "administrator" {
		t.Errorf("winner sign-in %+v", got)
	}
}

func TestActivationThatLostWhileHashingIsRefused(t *testing.T) {
	f := setup(t, SeedNone).ready()
	token := f.links()[""]
	pre := f.pre()
	var calls atomic.Int32
	entered, release := make(chan struct{}), make(chan struct{})
	f.s.pause = func() {
		if calls.Add(1) == 1 {
			close(entered)
			<-release
		}
	}
	var once sync.Once
	t.Cleanup(func() { once.Do(func() { close(release) }) })
	first := make(chan reply, 1)
	go func() { first <- f.auth("redeem", activationBody(token, "slow@example.invalid"), pre) }()
	select {
	case <-entered:
	case got := <-first:
		t.Fatalf("the first activation finished without pausing %+v", got)
	}
	if got := f.auth("redeem", activationBody(token, "fast@example.invalid"), pre); got.status != 204 {
		t.Fatalf("second activation %+v", got)
	}
	once.Do(func() { close(release) })
	f.code(t, "activation that lost while hashing", <-first, 422, CodeLinkInvalid)
	if f.known("slow@example.invalid") {
		t.Error("the losing login left a record")
	}
	if people := f.people(); len(people) != 1 || people[0] != "fast@example.invalid" {
		t.Errorf("directory %v", people)
	}
}

func TestSampleStartLeavesNoActivation(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.s.domain.RLock()
	live := f.s.boot.live
	f.s.domain.RUnlock()
	if live {
		t.Fatal("a live activation after the sample start")
	}
	now := f.s.clock()
	err := f.s.identity.Installation().ArmBootstrap(verifierOf(random(tokenBytes)), identity.Validity{NotBefore: now, NotAfter: now.Add(time.Minute)}, SampleWorkspace)
	if identity.ReasonOf(err) != identity.ReasonBootstrap {
		t.Errorf("arming after the sample start: %v", err)
	}
	before := len(f.people())
	f.code(t, "an activation guess", f.auth("redeem", activationBody(encode(random(tokenBytes)), "intruder@example.invalid"), f.pre()), 422, CodeLinkInvalid)
	if f.known("intruder@example.invalid") || len(f.people()) != before {
		t.Error("an activation guess created a user")
	}
}

func TestSignInToAnotherWorkspaceDoesNotAcceptTheInvitation(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	user, tok := f.created(root, "late@example.invalid", "administrator")
	if got := f.run(root, "identity.revoke_invitation", user, nil); got.status != 200 {
		t.Fatalf("revoke %+v", got)
	}
	if got := f.redeem(tok, pw); got.status != 204 {
		t.Fatalf("redeem %+v", got)
	}
	if got := f.run(root, "identity.invite", user, map[string]string{"role": "administrator"}); got.status != 200 {
		t.Fatalf("invite %+v", got)
	}
	pre := f.pre()
	other := f.auth("login", `{"workspace":"other-workspace","login":"late@example.invalid","password":"`+pw+`"}`, pre)
	wrong := f.auth("login", loginBodyFor("late@example.invalid", "a wrong long password"), pre)
	f.code(t, "sign-in naming another workspace", other, 401, string(commands.Unauthenticated))
	delete(other.body, "request_id")
	delete(wrong.body, "request_id")
	a, _ := json.Marshal(other.body)
	b, _ := json.Marshal(wrong.body)
	if string(a) != string(b) {
		t.Errorf("failure bodies differ: %s and %s", a, b)
	}
	f.s.domain.RLock()
	left := len(f.s.sessions.byLogin("late@example.invalid"))
	f.s.domain.RUnlock()
	if left != 0 {
		t.Errorf("%d sessions left", left)
	}
	if row := f.row(root, user); row == nil || row["membership"] != nil || row["invitation_expires"] == nil || row["invitation_expires"] == "" {
		t.Errorf("row after the other-workspace sign-in %+v", row)
	}
	if got := f.signin("late@example.invalid", pw); got.status != 200 || got.body["role"] != "administrator" {
		t.Errorf("sign-in to the boot workspace %+v", got)
	}
}

func TestLinksAndSessionsDoNotCrossInstallations(t *testing.T) {
	a := setup(t, SeedSample).ready().closed()
	b := setup(t, SeedNone).ready().closed()
	bToken := b.links()[""]
	a.code(t, "B's activation link at A", a.auth("redeem", activationBody(bToken, "first@example.invalid"), a.pre()), 422, CodeLinkInvalid)
	b.firstAdmin("first@example.invalid")
	b.code(t, "A's set link at B", b.redeem(a.links()[AdminLogin], pw), 422, CodeLinkInvalid)
	aSession := a.as(AdminLogin)
	b.code(t, "A's session at B", b.session(aSession.cookie), 401, string(commands.Unauthenticated))
	if b.known(AdminLogin) || len(b.people()) != 1 || a.known("first@example.invalid") {
		t.Error("a cross-installation attempt left a record")
	}

	one := setup(t, SeedNone).ready().closed()
	two := setup(t, SeedNone).ready().closed()
	oneToken := one.links()[""]
	oneAdmin := one.firstAdmin("same@example.invalid")
	two.firstAdmin("same@example.invalid")
	_, setToken := one.created(oneAdmin, "joiner@example.invalid", "member")
	two.code(t, "used activation link of the other installation", two.auth("redeem", activationBody(oneToken, "other@example.invalid"), two.pre()), 422, CodeLinkInvalid)
	two.code(t, "set link of the other installation", two.redeem(setToken, pw), 422, CodeLinkInvalid)
	two.code(t, "session of the other installation", two.session(oneAdmin.cookie), 401, string(commands.Unauthenticated))
	if two.known("joiner@example.invalid") || two.known("other@example.invalid") || len(two.people()) != 1 {
		t.Error("the other installation's links left a record")
	}
	if members := two.members(); len(members) != 1 || members[0].User != EmptyAdmin || members[0].Role != identity.Administrator {
		t.Errorf("members at the other installation %+v", members)
	}
	if got := one.session(oneAdmin.cookie); got.status != 200 {
		t.Errorf("the session at its own installation %+v", got)
	}
}

func TestAnEarlierRunsActivationLinkIsRefusedAfterRestart(t *testing.T) {
	for _, used := range []bool{true, false} {
		f := setup(t, SeedNone).ready().closed()
		earlier := f.links()[""]
		if used {
			f.firstAdmin("first@example.invalid")
		}
		g := f.restart(nil).closed()
		pre := g.pre()
		g.code(t, fmt.Sprintf("peek of the earlier link, used %v", used), g.auth("link", `{"token":"`+earlier+`"}`, pre), 422, CodeLinkInvalid)
		g.code(t, fmt.Sprintf("redeem of the earlier link, used %v", used), g.auth("redeem", activationBody(earlier, "again@example.invalid"), pre), 422, CodeLinkInvalid)
		if g.known("again@example.invalid") {
			t.Error("the earlier link created a user")
		}
		g.firstAdmin("again@example.invalid")
	}
}
