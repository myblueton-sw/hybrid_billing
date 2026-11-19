package commands

import (
	"encoding/json"
	"errors"
	"math/rand"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func authorSetup(t testing.TB) world {
	t.Helper()
	now := start
	armed := 0
	var hook func()
	clock := func() time.Time {
		if armed > 0 {
			armed--
			if armed == 0 && hook != nil {
				hook()
			}
		}
		return now
	}
	reg, err := identity.New("synthetic-installation", idLimits, clock)
	if err != nil {
		t.Fatal(err)
	}
	g, a, err := gate.New(reg, gate.Catalog{AuthorClient: "app", Kinds: []string{"statement", gate.InstallationKind}})
	if err != nil {
		t.Fatal(err)
	}
	c, err := New(g, testLimits)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	f := world{t: t, now: &now, armed: &armed, hook: &hook, reg: reg, in: reg.Installation(), b: reg.Boundary(), w: reg.Workspace(), g: g, a: a, c: c, count: &count,
		h: &stub{targets: map[string]*Target{"w1/own": {Revision: "r1", AuthorID: "root"}, "w1/ann": {Revision: "r1", AuthorID: "ann"},
			"w1/none": {Revision: "r1"}, "w1/cat": {Revision: "r1", AuthorID: "cat"}}}}
	f.must(f.in.ArmBootstrap(v("boot"), identity.Validity{NotBefore: start, NotAfter: start.Add(time.Hour)}, "w1"))
	f.must(f.b.Bootstrap(v("boot"), "root", triple("root"), ""))
	f.must(a.SetInstallationWorkspace("w1"))
	for _, client := range []string{"app", "spare"} {
		if _, err := a.PutClient(gate.Client{ID: client, State: gate.Active}, 0); err != nil {
			t.Fatal(err)
		}
	}
	f.must(a.AddGrant(gate.Grant{ID: "app-modules", ClientID: "app", Validity: always, Terms: gate.Terms{Action: gate.AdministerModules, Kind: gate.InstallationKind,
		Period: gate.AnyPeriod(), Scope: gate.Scope{WorkspaceID: "w1", Customer: gate.All(), LegalEntity: gate.All(), Contract: gate.All()}}}))
	f.must(c.Register(Definition{ID: "module.withdraw", Permission: Permission{Action: gate.AdministerModules, Kind: gate.InstallationKind}, Author: true,
		Mutating: true, Rationale: true, Handler: f.h}))
	for id, role := range map[string]identity.Role{"ann": identity.Administrator, "cat": identity.Member} {
		f.must(f.in.CreateUser(operator, id, triple(id), ""))
		invitation := f.next("invitation")
		f.must(f.w.Invite(f.login("root"), "w1", invitation, role, v(invitation), identity.Validity{NotBefore: *f.now, NotAfter: f.now.Add(48 * time.Hour)}, id))
		f.must(f.w.Accept("w1", invitation, v(invitation), f.login(id)))
	}
	return f
}

func withdraw(answer gate.Answer, client, key, target string) Call {
	return Call{Answer: answer, Client: client, Request: "req-w", Command: "module.withdraw", Target: target, Key: key, Expected: "r1", Rationale: "synthetic rationale"}
}

func TestAuthorRegistration(t *testing.T) {
	f := authorSetup(t)
	base := Definition{ID: "module.other", Permission: Permission{Action: gate.AdministerModules, Kind: gate.InstallationKind}, Author: true, Mutating: true, Handler: f.h}
	read := base
	read.ID, read.Mutating = "module.read", false
	approval := base
	approval.ID, approval.Approval, approval.Permission.Action = "module.approval", true, gate.Approve
	ordinary := base
	ordinary.ID, ordinary.Permission = "module.ordinary", Permission{Action: gate.Issue, Kind: "statement"}
	for name, d := range map[string]Definition{"read": read, "approval": approval, "ordinary": ordinary} {
		if err := f.c.Register(d); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
	f.must(f.c.Register(base))
	found := false
	for _, e := range f.c.Catalogue() {
		if e.ID == "module.withdraw" {
			found = e.Author
		}
		if e.ID == "statement.adjust" && e.Author {
			t.Error("an ordinary command is author-only")
		}
	}
	if !found {
		t.Error("the catalogue lost the author flag")
	}
	document, err := f.c.OpenAPI("Synthetic", "0.0.0")
	f.must(err)
	var parsed map[string]any
	f.must(json.Unmarshal(document, &parsed))
	permission := parsed["paths"].(map[string]any)["/commands/module.withdraw"].(map[string]any)["post"].(map[string]any)["x-permission"].(map[string]any)
	if permission["author_only"] != true || permission["right_not_required"] != true || permission["action"] != "administer_modules" {
		t.Errorf("x-permission %v", permission)
	}
}

func TestAuthorCommands(t *testing.T) {
	f := authorSetup(t)
	root, ann, cat := f.answer("root"), f.answer("ann"), f.answer("cat")
	f.code("the author", f.c.Execute(withdraw(root, "app", "k1", "own")), "")
	f.h.targets["w1/own"].Revision = "r1"
	f.code("another administrator", f.c.Execute(withdraw(ann, "app", "k2", "own")), PermissionDenied)
	f.code("no client", f.c.Execute(withdraw(root, "", "k3", "own")), PermissionDenied)
	f.code("a client without the right", f.c.Execute(withdraw(root, "spare", "k4", "own")), PermissionDenied)
	f.code("a member author", f.c.Execute(withdraw(cat, "app", "k5", "cat")), PermissionDenied)
	f.code("an empty author", f.c.Execute(withdraw(root, "app", "k6", "none")), PermissionDenied)
	f.code("an unknown target", f.c.Execute(withdraw(root, "app", "k7", "gone")), PermissionDenied)
	f.code("ann withdraws ann's", f.c.Execute(withdraw(ann, "app", "k8", "ann")), "")
	if _, applies := f.h.count(); applies != 2 {
		t.Errorf("%d applies", applies)
	}
	replay := f.code("a replay", f.c.Execute(withdraw(root, "app", "k1", "own")), "")
	if !replay.Replay {
		t.Errorf("not a replay %+v", replay)
	}
	session := f.login("ann")
	answer, err := f.b.Access(session, "w1")
	f.must(err)
	*f.hook = func() { f.must(f.b.EndSession(session)) }
	*f.armed = 9
	f.h.targets["w1/ann"].Revision = "r1"
	f.code("the session ends between the check and the apply", f.c.Execute(withdraw(answer, "app", "k9", "ann")), Unauthenticated)
	if _, applies := f.h.count(); applies != 2 {
		t.Errorf("the author recheck did not stop the apply: %d applies", applies)
	}
	admin := f.login("root")
	f.must(f.w.SuspendMembership(admin, "w1", "ann", 1))
	f.code("a replay after the author is suspended", f.c.Execute(withdraw(ann, "app", "k8", "ann")), Unauthenticated)
}

func TestAuthorCommandsAgreeWithTheRules(t *testing.T) {
	r := rand.New(rand.NewSource(310))
	f := authorSetup(t)
	answers := map[string]gate.Answer{"root": f.answer("root"), "ann": f.answer("ann"), "cat": f.answer("cat")}
	clients := []string{"", "app", "spare"}
	targets := []string{"own", "ann", "cat", "none", "gone"}
	for round := 0; round < 300; round++ {
		user, client, target := []string{"root", "ann", "cat"}[r.Intn(3)], clients[r.Intn(3)], targets[r.Intn(len(targets))]
		if found := f.h.targets["w1/"+target]; found != nil {
			found.Revision = "r1"
		}
		author := ""
		if found := f.h.targets["w1/"+target]; found != nil {
			author = found.AuthorID
		}
		allowed := client == "app" && author == user && user != "cat"
		got := f.c.Execute(withdraw(answers[user], client, f.next("key"), target))
		if (got.Err == nil) != allowed {
			t.Fatalf("round %d: %s via %q on %s: %+v", round, user, client, target, got.Err)
		}
		if !allowed && got.Err.Code != PermissionDenied {
			t.Fatalf("round %d: %s", round, got.Err.Code)
		}
	}
}

func TestAHandlerMayAnswerAuthorityUnavailable(t *testing.T) {
	f, ann := basic(t)
	f.h.apply = func(Context, *Target) (Outcome, error) { return Outcome{}, Fail(AuthorityUnavailable) }
	got := f.code("a handler that cannot check authority", f.c.Execute(adjust(ann, "k1", "1")), AuthorityUnavailable)
	if !got.Err.Retry || got.Err.Status != 503 {
		t.Errorf("%+v", got.Err)
	}
	f.h.apply = nil
	f.code("the same key again", f.c.Execute(adjust(ann, "k1", "1")), "")
}
