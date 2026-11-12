package gate

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

type world struct {
	t     testing.TB
	now   *time.Time
	armed *int
	hook  *func()
	reg   *identity.Registry
	in    *identity.Installation
	b     *identity.Boundary
	w     *identity.Workspace
	g     *Gate
	a     *Administration
	count *int
}

var (
	start    = time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	always   = Validity{NotBefore: start.Add(-time.Hour), NotAfter: start.Add(1000 * time.Hour)}
	operator = identity.Operator{Reference: "synthetic-operator", Reason: "synthetic-reason"}
	limits   = identity.Limits{BootstrapValidity: time.Hour, InvitationValidity: 72 * time.Hour, CredentialValidity: 720 * time.Hour,
		SessionIdle: 30 * time.Minute, SessionAbsolute: 8 * time.Hour, Reauthentication: 5 * time.Minute,
		Attempts: 3, PendingInvitations: 1000, SessionsPerUser: 100000, CredentialsPerServiceUser: 1000, PageSize: 100}
	catalog = Catalog{Kinds: []string{"usage_statement", "allocation_pool"}, Fields: map[string]FieldClass{"quantity": General, "unit_cost": Cost}}
)

func v(label string) identity.Verifier {
	return identity.Verifier(sha256.Sum256([]byte("synthetic-secret-" + label)))
}

func triple(subject string) identity.Triple {
	return identity.Triple{Connection: "synthetic-connection", Issuer: "https://issuer.invalid", Subject: subject}
}

func setup(t testing.TB) world {
	t.Helper()
	now, armed, count := start, 0, 0
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
	reg, err := identity.New("synthetic-installation", limits, clock)
	if err != nil {
		t.Fatal(err)
	}
	g, a, err := New(reg, catalog)
	if err != nil {
		t.Fatal(err)
	}
	f := world{t: t, now: &now, armed: &armed, hook: &hook, reg: reg, in: reg.Installation(), b: reg.Boundary(), w: reg.Workspace(), g: g, a: a, count: &count}
	f.must(f.in.ArmBootstrap(v("boot"), identity.Validity{NotBefore: start, NotAfter: start.Add(time.Hour)}, "w1"))
	f.must(f.b.Bootstrap(v("boot"), "root", triple("root"), ""))
	return f
}

func (f world) must(err error) {
	f.t.Helper()
	if err != nil {
		f.t.Fatal(err)
	}
}

func (f world) next(label string) string {
	*f.count++
	return fmt.Sprintf("%s-%d", label, *f.count)
}

func (f world) login(id string) identity.Verifier {
	f.t.Helper()
	verifier := v(f.next("session-" + id))
	f.must(f.b.OpenSession(triple(id), verifier, 30*time.Minute, 8*time.Hour))
	return verifier
}

func (f world) answer(id string) identity.Answer {
	f.t.Helper()
	answer, err := f.b.Access(f.login(id), "w1")
	f.must(err)
	return answer
}

func (f world) join(id string, role identity.Role) {
	f.t.Helper()
	invitation := f.next("invitation")
	f.must(f.w.Invite(f.login("root"), "w1", invitation, role, v(invitation), identity.Validity{NotBefore: *f.now, NotAfter: f.now.Add(48 * time.Hour)}, id))
	f.must(f.w.Accept("w1", invitation, v(invitation), f.login(id)))
}

func (f world) member(id string) {
	f.t.Helper()
	f.must(f.in.CreateUser(operator, id, triple(id), ""))
	f.join(id, identity.Member)
}

func terms(action Action, fields ...string) Terms {
	return Terms{Action: action, Kind: "usage_statement", Scope: Scope{WorkspaceID: "w1", Customer: One("c1"), LegalEntity: All(), Contract: All()}, Fields: fields, Period: AnyPeriod()}
}

func (f world) grant(subject string, action Action, fields ...string) string {
	f.t.Helper()
	id := f.next("grant")
	f.must(f.a.AddGrant(Grant{ID: id, SubjectID: subject, Terms: terms(action, fields...), Validity: always}))
	return id
}

func ask(action Action, fields ...string) Request {
	return Request{Action: action, Kind: "usage_statement", Resource: Resource{WorkspaceID: "w1", CustomerID: "c1"}, Fields: fields}
}

func (f world) expect(name string, got Decision, allowed bool, reason Reason) Decision {
	f.t.Helper()
	if got.Allowed != allowed || got.Reason != reason {
		f.t.Errorf("%s: allowed %v reason %s; want %v %s", name, got.Allowed, got.Reason, allowed, reason)
	}
	if got.Allowed != (got.Code == Allowed) || got.Allowed != got.Stamp.issued {
		f.t.Errorf("%s: inconsistent decision %+v", name, got)
	}
	return got
}

func (f world) epoch(id string) uint64 {
	_, _, epoch, _ := f.reg.Subjects().Subject(id)
	return epoch
}

func (f world) version(id string) uint64 {
	f.t.Helper()
	members, err := f.w.Members(f.login("root"), "w1", "", 100)
	f.must(err)
	for _, found := range members {
		if found.User == id {
			return found.Version
		}
	}
	f.t.Fatalf("no member %s", id)
	return 0
}

func TestADecisionNeedsACurrentAnswerOfTheActor(t *testing.T) {
	type held struct {
		world
		session identity.Verifier
		answer  identity.Answer
		stamp   Stamp
	}
	build := func(t *testing.T) held {
		f := setup(t)
		f.member("ann")
		f.grant("ann", Issue)
		session := f.login("ann")
		answer, err := f.b.Access(session, "w1")
		f.must(err)
		first := f.expect("the decision", f.g.Decide(answer, "", ask(Issue)), true, ReasonAllowed)
		if first.ActorID != "ann" {
			t.Fatalf("actor %q", first.ActorID)
		}
		f.expect("the recheck", f.g.Recheck(first.Stamp, "", ask(Issue)), true, ReasonAllowed)
		return held{f, session, answer, first.Stamp}
	}
	admin := func(h held) identity.Verifier { return h.login("root") }
	changes := map[string]func(held){
		"the user is suspended": func(h held) { h.must(h.in.SuspendUser(operator, "ann", 1)) },
		"the user is suspended and reinstated": func(h held) {
			h.must(h.in.SuspendUser(operator, "ann", 1))
			h.must(h.in.ReinstateUser(operator, "ann", 2))
		},
		"the user departs":            func(h held) { h.must(h.in.DepartUser(operator, "ann", 1)) },
		"the membership is suspended": func(h held) { h.must(h.w.SuspendMembership(admin(h), "w1", "ann", 1)) },
		"the membership is removed":   func(h held) { h.must(h.w.RemoveMembership(admin(h), "w1", "ann", 1)) },
		"the role changes":            func(h held) { h.must(h.w.ChangeRole(admin(h), "w1", "ann", identity.Administrator, 1)) },
		"the session is ended":        func(h held) { h.must(h.b.EndSession(h.session)) },
		"an identity is linked":       func(h held) { h.must(h.b.LinkIdentity(h.session, triple("ann-2"), "", "proof")) },
		"an identity is linked and unlinked": func(h held) {
			h.must(h.b.LinkIdentity(h.session, triple("ann-2"), "", "proof"))
			h.must(h.b.UnlinkIdentity(h.login("ann"), triple("ann-2")))
		},
		"the answer expires": func(h held) { *h.now = start.Add(8 * time.Hour) },
	}
	for name, change := range changes {
		h := build(t)
		change(h)
		got := h.expect(name, h.g.Decide(h.answer, "", ask(Issue)), false, ReasonIdentity)
		if got.ActorID != "" || got.Code != PermissionDenied {
			t.Errorf("%s: a denial before the answer is confirmed names %q with %s", name, got.ActorID, got.Code)
		}
		h.expect(name+", recheck", h.g.Recheck(h.stamp, "", ask(Issue)), false, ReasonIdentity)
	}
	h := build(t)
	h.expect("the zero answer", h.g.Decide(identity.Answer{}, "", ask(Issue)), false, ReasonIdentity)
	h.expect("the zero stamp", h.g.Recheck(Stamp{}, "", ask(Issue)), false, ReasonStamp)
	twin := build(t)
	twin.expect("an answer of another identity registry", twin.g.Decide(h.answer, "", ask(Issue)), false, ReasonIdentity)
	twin.expect("a stamp of another gate", twin.g.Recheck(h.stamp, "", ask(Issue)), false, ReasonIdentity)

	h = build(t)
	h.must(h.in.SuspendUser(operator, "ann", 1))
	h.must(h.in.ReinstateUser(operator, "ann", 2))
	h.expect("a new answer after reinstatement keeps the grant", h.g.Decide(h.world.answer("ann"), "", ask(Issue)), true, ReasonAllowed)
	h = build(t)
	h.must(h.w.ChangeRole(admin(h), "w1", "ann", identity.Administrator, 1))
	h.expect("a new answer after a role change keeps the grant", h.g.Decide(h.world.answer("ann"), "", ask(Issue)), true, ReasonAllowed)
	h = build(t)
	h.must(h.w.SuspendMembership(admin(h), "w1", "ann", 1))
	h.must(h.w.ReinstateMembership(admin(h), "w1", "ann", 2))
	h.expect("a new answer after the membership was suspended and reinstated keeps the grant", h.g.Decide(h.world.answer("ann"), "", ask(Issue)), true, ReasonAllowed)
	h = build(t)
	h.must(h.w.RemoveMembership(admin(h), "w1", "ann", 1))
	h.join("ann", identity.Member)
	again := h.world.answer("ann")
	h.expect("a removed and invited member has lost the grant", h.g.Decide(again, "", ask(Issue)), false, ReasonNoGrant)
	h.grant("ann", Issue)
	h.expect("a new grant for the new membership", h.g.Decide(again, "", ask(Issue)), true, ReasonAllowed)
	h.expect("the stamp of the earlier membership", h.g.Recheck(h.stamp, "", ask(Issue)), false, ReasonIdentity)
}

func TestTheWorkspaceOfADecisionIsTheAnswers(t *testing.T) {
	f := setup(t)
	f.member("ann")
	f.grant("ann", Issue)
	answer := f.answer("ann")
	elsewhere := ask(Issue)
	elsewhere.Resource.WorkspaceID = "w2"
	got := f.expect("a request naming another workspace", f.g.Decide(answer, "", elsewhere), false, ReasonWorkspace)
	if got.ActorID != "ann" || got.Code != PermissionDenied {
		t.Errorf("the denial %+v", got)
	}
	missing := ask(Issue)
	missing.Resource.WorkspaceID = ""
	f.expect("a request without workspace", f.g.Decide(answer, "", missing), false, ReasonWorkspace)
	first := f.expect("the decision", f.g.Decide(answer, "", ask(Issue)), true, ReasonAllowed)
	f.expect("a recheck naming another workspace", f.g.Recheck(first.Stamp, "", elsewhere), false, ReasonWorkspace)
	other := ask(Issue)
	other.Resource.CustomerID = "c2"
	f.expect("a recheck for another request", f.g.Recheck(first.Stamp, "", other), false, ReasonNoGrant)
	outside := terms(Issue)
	outside.Scope.WorkspaceID = "w2"
	if err := f.a.AddGrant(Grant{ID: "x", SubjectID: "ann", Terms: outside, Validity: always}); !errors.Is(err, ErrInvalid) {
		t.Errorf("a grant in a workspace the user is not a member of: %v", err)
	}
	if err := f.a.AddGrant(Grant{ID: "y", SubjectID: "nobody", Terms: terms(Issue), Validity: always}); !errors.Is(err, ErrInvalid) {
		t.Errorf("a grant for an unknown user: %v", err)
	}
	f.must(f.in.CreateUser(operator, "out", triple("out"), ""))
	if err := f.a.AddGrant(Grant{ID: "z", SubjectID: "out", Terms: terms(Issue), Validity: always}); !errors.Is(err, ErrInvalid) {
		t.Errorf("a grant for a user who is not a member: %v", err)
	}
}

func TestServiceCredentials(t *testing.T) {
	f := setup(t)
	f.member("ann")
	admin := func() identity.Verifier { return f.login("root") }
	f.must(f.w.CreateServiceUser(admin(), "w1", "svc", "ann"))
	f.must(f.w.AddCredential(admin(), "w1", "svc", "c1", v("c1"), identity.Validity{NotBefore: start, NotAfter: start.Add(time.Hour)}))
	f.grant("svc", Issue)
	if err := f.a.AddGrant(Grant{ID: "x", SubjectID: "svc", Terms: terms(Approve), Validity: always}); !errors.Is(err, ErrInvalid) {
		t.Errorf("an approval grant for a service user: %v", err)
	}
	answer, err := f.b.AccessByCredential("c1", v("c1"))
	f.must(err)
	first := f.expect("a service decision", f.g.Decide(answer, "", ask(Issue)), true, ReasonAllowed)
	f.must(second(f.a.PutClient(Client{ID: "app", State: Active}, 0)))
	got := f.expect("a service answer with a client", f.g.Decide(answer, "app", ask(Issue)), false, ReasonPrincipal)
	if got.ActorID != "svc" {
		t.Errorf("actor %q", got.ActorID)
	}
	f.expect("a human-only action", f.g.Decide(answer, "", ask(Approve)), false, ReasonApprovalRequired)
	f.expect("a service approver", f.g.CheckApproval(answer, "ann", ask(Approve)), false, ReasonNotHuman)
	if got := f.expect("a service answer in a support session", f.g.DecideInSupport(answer, "s", ask(View, "quantity")), false, ReasonSupportSession); got.ActorID != "" || got.SessionID != "" {
		t.Errorf("a refusal before the answer is confirmed names %q and %q", got.ActorID, got.SessionID)
	}
	*f.now = start.Add(time.Hour)
	f.expect("at the credential's end", f.g.Decide(answer, "", ask(Issue)), false, ReasonIdentity)
	*f.now = start
	f.must(f.w.SuspendServiceUser(admin(), "w1", "svc", 1))
	f.expect("a suspended service user", f.g.Recheck(first.Stamp, "", ask(Issue)), false, ReasonIdentity)
	f.must(f.w.ReinstateServiceUser(admin(), "w1", "svc", 2))
	renewed, err := f.b.AccessByCredential("c1", v("c1"))
	f.must(err)
	f.expect("a new answer after reinstatement", f.g.Decide(renewed, "", ask(Issue)), true, ReasonAllowed)
	f.must(f.w.RevokeCredential(admin(), "w1", "c1"))
	f.expect("a revoked credential", f.g.Decide(renewed, "", ask(Issue)), false, ReasonIdentity)
}

func second[T any](_ T, err error) error { return err }

func TestClients(t *testing.T) {
	f := setup(t)
	f.member("ann")
	f.grant("ann", Issue)
	f.grant("ann", Deliver)
	f.must(second(f.a.PutClient(Client{ID: "app", State: Active}, 0)))
	f.must(f.a.AddGrant(Grant{ID: "client-issue", ClientID: "app", Terms: terms(Issue), Validity: always}))
	answer := f.answer("ann")
	first := f.expect("user and client both hold the action", f.g.Decide(answer, "app", ask(Issue)), true, ReasonAllowed)
	if first.ClientID != "app" {
		t.Errorf("client %q", first.ClientID)
	}
	f.expect("only the user holds the action", f.g.Decide(answer, "app", ask(Deliver)), false, ReasonClient)
	f.expect("an unknown client", f.g.Decide(answer, "ghost", ask(Issue)), false, ReasonClient)
	f.expect("a recheck without the client", f.g.Recheck(first.Stamp, "", ask(Issue)), false, ReasonStamp)
	f.expect("a recheck with the client", f.g.Recheck(first.Stamp, "app", ask(Issue)), true, ReasonAllowed)
	f.must(second(f.a.PutClient(Client{ID: "app", State: Suspended}, 1)))
	f.expect("a suspended client", f.g.Decide(answer, "app", ask(Issue)), false, ReasonClient)
	f.expect("the user alone still decides", f.g.Decide(answer, "", ask(Issue)), true, ReasonAllowed)
}

func session(id, actor string) SupportSession {
	return SupportSession{ID: id, ActorID: actor, WorkspaceID: "w1", CustomerID: "c1", Reason: "ticket", Kind: "usage_statement", Fields: []string{"quantity"},
		Validity: Validity{NotBefore: start.Add(-time.Minute), NotAfter: start.Add(time.Hour)}}
}

func TestSupportSessions(t *testing.T) {
	f := setup(t)
	f.member("ann")
	f.member("bob")
	f.grant("root", ApproveSupportSession)
	approver := f.answer("root")
	f.must(f.g.AddSupportSession(session("s", "ann"), approver))
	ann, bob := f.answer("ann"), f.answer("bob")
	first := f.expect("the actor inside the session", f.g.DecideInSupport(ann, "s", ask(View, "quantity")), true, ReasonAllowed)
	if first.SessionID != "s" || first.ActorID != "ann" {
		t.Errorf("decision %+v", first)
	}
	f.expect("an empty session identifier", f.g.DecideInSupport(ann, "", ask(View, "quantity")), false, ReasonSupportSession)
	f.expect("another user's session", f.g.DecideInSupport(bob, "s", ask(View, "quantity")), false, ReasonSupportSession)
	f.expect("the session grants nothing outside it", f.g.Decide(ann, "", ask(View, "quantity")), false, ReasonNoGrant)
	f.expect("the recheck", f.g.RecheckInSupport(first.Stamp, "s", ask(View, "quantity")), true, ReasonAllowed)
	f.expect("a recheck without the session", f.g.Recheck(first.Stamp, "", ask(View, "quantity")), false, ReasonNoGrant)
	f.expect("a recheck with an empty session", f.g.RecheckInSupport(first.Stamp, "", ask(View, "quantity")), false, ReasonSupportSession)
	f.expect("a support recheck with the zero stamp", f.g.RecheckInSupport(Stamp{}, "s", ask(View, "quantity")), false, ReasonStamp)

	other := session("s2", "ann")
	other.WorkspaceID = "w2"
	if err := f.g.AddSupportSession(other, approver); !errors.Is(err, ErrDenied) {
		t.Errorf("a session for another workspace than the approver's answer: %v", err)
	}
	if err := f.g.AddSupportSession(session("s3", "ann"), ann); !errors.Is(err, ErrDenied) {
		t.Errorf("an approver without the grant: %v", err)
	}
	f.must(f.w.CreateServiceUser(f.login("root"), "w1", "svc", "ann"))
	f.must(f.w.AddCredential(f.login("root"), "w1", "svc", "c1", v("c1"), identity.Validity{NotBefore: start, NotAfter: start.Add(time.Hour)}))
	service, err := f.b.AccessByCredential("c1", v("c1"))
	f.must(err)
	if err := f.g.AddSupportSession(session("s4", "ann"), service); !errors.Is(err, ErrDenied) {
		t.Errorf("a service approver: %v", err)
	}
	if err := f.g.AddSupportSession(session("s5", "ann"), identity.Answer{}); !errors.Is(err, ErrDenied) {
		t.Errorf("the zero answer as approver: %v", err)
	}
	ended := f.login("root")
	gone, err := f.b.Access(ended, "w1")
	f.must(err)
	f.must(f.b.EndSession(ended))
	if err := f.g.AddSupportSession(session("early", "ann"), gone); !errors.Is(err, ErrDenied) {
		t.Errorf("an approver whose session had ended before the call: %v", err)
	}
	if err := f.g.AddSupportSession(session("early", "ann"), f.answer("root")); err != nil {
		t.Errorf("the identifier a refused approver named is still free: %v", err)
	}
	f.must(f.in.CreateUser(operator, "out", triple("out"), ""))
	if err := f.g.AddSupportSession(session("s6", "out"), approver); !errors.Is(err, ErrInvalid) {
		t.Errorf("an actor who is not a member: %v", err)
	}

	removed := f.version("ann")
	f.must(f.w.RemoveMembership(f.login("root"), "w1", "ann", removed))
	f.join("ann", identity.Member)
	f.expect("the actor removed and invited again", f.g.DecideInSupport(f.answer("ann"), "s", ask(View, "quantity")), false, ReasonSupportSession)

	approverSession := f.login("root")
	late, err := f.b.Access(approverSession, "w1")
	f.must(err)
	*f.hook = func() { f.must(f.b.EndSession(approverSession)) }
	*f.armed = 2
	if err := f.g.AddSupportSession(session("undone", "bob"), late); !errors.Is(err, ErrDenied) {
		t.Errorf("an approver whose session ended during the registration: %v", err)
	}
	f.expect("the session that was registered and undone", f.g.DecideInSupport(f.answer("bob"), "undone", ask(View, "quantity")), false, ReasonSupportSession)
	if err := f.g.AddSupportSession(session("undone", "bob"), f.answer("root")); !errors.Is(err, ErrConflict) {
		t.Errorf("the identifier of an undone session: %v", err)
	}
	f.must(f.b.EndSession(f.login("root")))
	stale := f.answer("root")
	f.must(f.in.SuspendUser(operator, "root", 1))
	if err := f.g.AddSupportSession(session("s7", "bob"), stale); !errors.Is(err, ErrDenied) {
		t.Errorf("an approver whose answer is not current: %v", err)
	}
}

func TestApprovals(t *testing.T) {
	f := setup(t)
	f.member("ann")
	f.member("bob")
	f.member("carol")
	f.grant("bob", Approve)
	f.grant("root", Approve)
	f.grant("carol", Approve)
	bob := f.answer("bob")
	first := f.expect("an independent approver", f.g.CheckApproval(bob, "ann", ask(Approve)), true, ReasonAllowed)
	self := f.expect("the author approving", f.g.CheckApproval(bob, "bob", ask(Approve)), false, ReasonSelfApproval)
	if self.Code != SelfApprovalDenied {
		t.Errorf("code %s", self.Code)
	}
	f.expect("an unknown author", f.g.CheckApproval(bob, "nobody", ask(Approve)), false, ReasonAuthor)
	f.expect("an approver without the grant", f.g.CheckApproval(f.answer("ann"), "bob", ask(Approve)), false, ReasonNoGrant)
	f.expect("another action", f.g.CheckApproval(bob, "ann", ask(Issue)), false, ReasonInvalidRequest)
	f.expect("approve through the ordinary decision", f.g.Decide(bob, "", ask(Approve)), false, ReasonApprovalRequired)
	f.expect("the approval recheck", f.g.RecheckApproval(first.Stamp, "ann", ask(Approve)), true, ReasonAllowed)
	f.expect("the approval recheck for another author", f.g.RecheckApproval(first.Stamp, "carol", ask(Approve)), false, ReasonStamp)
	f.expect("an approval stamp as an ordinary stamp", f.g.Recheck(first.Stamp, "", ask(Approve)), false, ReasonApprovalRequired)
	f.expect("an approval recheck with the zero stamp", f.g.RecheckApproval(Stamp{}, "ann", ask(Approve)), false, ReasonStamp)
	elsewhere := ask(Approve)
	elsewhere.Resource.WorkspaceID = "w2"
	f.expect("an approval in another workspace", f.g.CheckApproval(bob, "ann", elsewhere), false, ReasonWorkspace)

	f.must(f.w.CreateServiceUser(f.login("root"), "w1", "svc", "bob"))
	owned := f.expect("a service author owned by the approver", f.g.CheckApproval(bob, "svc", ask(Approve)), false, ReasonSelfApproval)
	if owned.Code != SelfApprovalDenied || owned.ActorID != "bob" || owned.Stamp.issued {
		t.Errorf("the denial %+v", owned)
	}
	f.expect("a service author created by the approver", f.g.CheckApproval(f.answer("root"), "svc", ask(Approve)), false, ReasonSelfApproval)
	third := f.expect("a service author of somebody else", f.g.CheckApproval(f.answer("carol"), "svc", ask(Approve)), true, ReasonAllowed)
	f.expect("its recheck", f.g.RecheckApproval(third.Stamp, "svc", ask(Approve)), true, ReasonAllowed)
	f.expect("an approval stamp presented for another author, a service user", f.g.RecheckApproval(first.Stamp, "svc", ask(Approve)), false, ReasonStamp)

	f.must(f.in.DepartUser(operator, "ann", f.epoch("ann")))
	f.expect("a departed author is still a known author", f.g.CheckApproval(f.answer("bob"), "ann", ask(Approve)), true, ReasonAllowed)
	f.expect("the earlier approval after an unrelated departure", f.g.RecheckApproval(first.Stamp, "ann", ask(Approve)), false, ReasonStamp)
}

func (f world) delegate(id, from, to string) error {
	return f.a.AddDelegation(Delegation{ID: id, DelegatorID: from, DelegateID: to, Terms: terms(Issue), Validity: always})
}

func TestDelegationsFollowTheDelegatorsMembership(t *testing.T) {
	build := func(t *testing.T) (world, identity.Answer, Stamp) {
		f := setup(t)
		f.member("alice")
		f.member("bob")
		f.grant("alice", Issue)
		f.grant("bob", RequestApproval)
		f.must(f.delegate("d", "alice", "bob"))
		bob := f.answer("bob")
		first := f.expect("the delegate", f.g.Decide(bob, "", ask(Issue)), true, ReasonAllowed)
		if !reflect.DeepEqual(first.Delegations, []string{"d"}) {
			t.Fatalf("delegations %v", first.Delegations)
		}
		return f, bob, first.Stamp
	}
	admin := func(f world) identity.Verifier { return f.login("root") }
	for name, c := range map[string]struct {
		change func(world)
		decide Reason
		after  bool
	}{
		"the delegator's membership is suspended": {func(f world) { f.must(f.w.SuspendMembership(admin(f), "w1", "alice", 1)) }, ReasonNoGrant, false},
		"the delegator is removed":                {func(f world) { f.must(f.w.RemoveMembership(admin(f), "w1", "alice", 1)) }, ReasonNoGrant, false},
		"the delegator is suspended":              {func(f world) { f.must(f.in.SuspendUser(operator, "alice", 1)) }, ReasonNoGrant, false},
		"the delegator departs":                   {func(f world) { f.must(f.in.DepartUser(operator, "alice", 1)) }, ReasonNoGrant, false},
		"the delegator is removed, invited and granted again": {func(f world) {
			f.must(f.w.RemoveMembership(admin(f), "w1", "alice", 1))
			f.join("alice", identity.Member)
			f.grant("alice", Issue)
		}, ReasonNoGrant, false},
		"the delegator is suspended and reinstated": {func(f world) {
			f.must(f.in.SuspendUser(operator, "alice", 1))
			f.must(f.in.ReinstateUser(operator, "alice", 2))
		}, ReasonAllowed, true},
		"the delegator's membership is suspended and reinstated": {func(f world) {
			f.must(f.w.SuspendMembership(admin(f), "w1", "alice", 1))
			f.must(f.w.ReinstateMembership(admin(f), "w1", "alice", 2))
		}, ReasonAllowed, true},
	} {
		f, bob, stamp := build(t)
		c.change(f)
		f.expect(name, f.g.Decide(bob, "", ask(Issue)), c.after, c.decide)
		reason := c.decide
		if c.after {
			reason = ReasonStamp
		}
		f.expect(name+", the stamp taken before", f.g.Recheck(stamp, "", ask(Issue)), false, reason)
	}
	f, _, stamp := build(t)
	f.must(f.in.DepartUser(operator, "bob", 1))
	f.expect("the delegate departs", f.g.Recheck(stamp, "", ask(Issue)), false, ReasonIdentity)
	f, _, _ = build(t)
	f.must(f.in.CreateUser(operator, "out", triple("out"), ""))
	if err := f.delegate("x", "alice", "out"); !errors.Is(err, ErrInvalid) {
		t.Errorf("a delegation to a user who is not a member: %v", err)
	}
	if err := f.delegate("y", "out", "bob"); !errors.Is(err, ErrInvalid) {
		t.Errorf("a delegation from a user who is not a member: %v", err)
	}
}

func TestChangesDuringADecision(t *testing.T) {
	build := func(t *testing.T) (world, identity.Verifier, identity.Answer) {
		f := setup(t)
		f.member("ann")
		f.grant("ann", Issue)
		session := f.login("ann")
		answer, err := f.b.Access(session, "w1")
		f.must(err)
		return f, session, answer
	}
	during := func(f world, change func()) {
		fired := false
		*f.hook = func() { fired = true; change() }
		*f.armed = 2
		f.t.Cleanup(func() {
			if !fired {
				f.t.Errorf("the change was never made")
			}
		})
	}
	named := func(got Decision, actor string) {
		t.Helper()
		if got.ActorID != actor {
			t.Errorf("a denial by the second confirmation names %q; want %q", got.ActorID, actor)
		}
	}
	f, session, answer := build(t)
	during(f, func() { f.must(f.b.EndSession(session)) })
	named(f.expect("the session ends during the decision", f.g.Decide(answer, "", ask(Issue)), false, ReasonIdentity), "ann")

	f, _, answer = build(t)
	admin := f.login("root")
	during(f, func() { f.must(f.w.ChangeRole(admin, "w1", "ann", identity.Administrator, 1)) })
	named(f.expect("the role changes during the decision", f.g.Decide(answer, "", ask(Issue)), false, ReasonIdentity), "ann")

	f, _, _ = build(t)
	f.must(f.w.CreateServiceUser(f.login("root"), "w1", "svc", "ann"))
	f.must(f.w.AddCredential(f.login("root"), "w1", "svc", "c1", v("c1"), identity.Validity{NotBefore: start, NotAfter: start.Add(time.Hour)}))
	f.grant("svc", Issue)
	service, err := f.b.AccessByCredential("c1", v("c1"))
	f.must(err)
	admin = f.login("root")
	during(f, func() { f.must(f.w.RevokeCredential(admin, "w1", "c1")) })
	named(f.expect("the credential is revoked during the decision", f.g.Decide(service, "", ask(Issue)), false, ReasonIdentity), "svc")

	f, _, answer = build(t)
	during(f, func() { f.must(f.in.SuspendUser(operator, "ann", 1)) })
	f.expect("the user is suspended during the decision", f.g.Decide(answer, "", ask(Issue)), false, ReasonPrincipal)

	f, _, answer = build(t)
	admin = f.login("root")
	during(f, func() { f.must(f.w.RemoveMembership(admin, "w1", "ann", 1)) })
	f.expect("the membership is removed during the decision", f.g.Decide(answer, "", ask(Issue)), false, ReasonNoGrant)

	f, session, answer = build(t)
	first := f.expect("a decision", f.g.Decide(answer, "", ask(Issue)), true, ReasonAllowed)
	during(f, func() { f.must(f.b.EndSession(session)) })
	named(f.expect("the session ends during a recheck", f.g.Recheck(first.Stamp, "", ask(Issue)), false, ReasonIdentity), "ann")

	f, _, _ = build(t)
	f.member("bob")
	f.grant("bob", Approve)
	approverSession := f.login("bob")
	approver, err := f.b.Access(approverSession, "w1")
	f.must(err)
	during(f, func() { f.must(f.b.EndSession(approverSession)) })
	named(f.expect("the approver's session ends during the approval check", f.g.CheckApproval(approver, "ann", ask(Approve)), false, ReasonIdentity), "bob")
}

func TestZeroValuesAndConstruction(t *testing.T) {
	var none *Gate
	var nobody *Administration
	for _, handles := range []struct {
		g *Gate
		a *Administration
	}{{none, nobody}, {&Gate{}, &Administration{}}} {
		g, a := handles.g, handles.a
		for name, got := range map[string]Decision{
			"Decide": g.Decide(identity.Answer{}, "", ask(Issue)), "DecideInSupport": g.DecideInSupport(identity.Answer{}, "s", ask(View, "quantity")),
			"Recheck": g.Recheck(Stamp{}, "", ask(Issue)), "RecheckInSupport": g.RecheckInSupport(Stamp{}, "s", ask(View, "quantity")),
			"CheckApproval": g.CheckApproval(identity.Answer{}, "a", ask(Approve)), "RecheckApproval": g.RecheckApproval(Stamp{}, "a", ask(Approve)),
		} {
			if got.Allowed || got.Code != AuthorityUnavailable || got.Reason != ReasonUnavailable {
				t.Errorf("%s on a zero gate: %+v", name, got)
			}
		}
		for name, err := range map[string]error{
			"AddSupportSession": g.AddSupportSession(session("s", "a"), identity.Answer{}),
			"PutClient":         second(a.PutClient(Client{ID: "app", State: Active}, 0)), "AddGrant": a.AddGrant(Grant{}), "RevokeGrant": a.RevokeGrant("g", 1),
			"AddDelegation": a.AddDelegation(Delegation{}), "RevokeDelegation": a.RevokeDelegation("d", 1),
			"EndSupportSession": a.EndSupportSession("s", 1), "SetAvailable": a.SetAvailable(true),
		} {
			if !errors.Is(err, ErrUnavailable) {
				t.Errorf("%s on a zero value: %v", name, err)
			}
		}
	}
	if _, _, err := New(nil, catalog); !errors.Is(err, ErrInvalid) {
		t.Errorf("a nil identity registry: %v", err)
	}
	if _, _, err := New(&identity.Registry{}, catalog); !errors.Is(err, ErrInvalid) {
		t.Errorf("an unconstructed identity registry: %v", err)
	}
	reg, err := identity.New("synthetic-installation", limits, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := New(reg, Catalog{}); !errors.Is(err, ErrInvalid) {
		t.Errorf("an empty catalogue: %v", err)
	}
	f := setup(t)
	f.member("ann")
	f.grant("ann", Issue)
	answer := f.answer("ann")
	f.must(f.a.SetAvailable(false))
	got := f.g.Decide(answer, "", ask(Issue))
	if got.Allowed || got.Code != AuthorityUnavailable {
		t.Errorf("a decision while the permission registry is unavailable: %+v", got)
	}
	f.must(f.a.SetAvailable(true))
	f.expect("available again", f.g.Decide(answer, "", ask(Issue)), true, ReasonAllowed)
	id := f.grant("ann", Deliver)
	f.expect("a granted action", f.g.Decide(answer, "", ask(Deliver)), true, ReasonAllowed)
	f.must(f.a.RevokeGrant(id, 1))
	f.expect("a revoked grant", f.g.Decide(answer, "", ask(Deliver)), false, ReasonNoGrant)
	f.member("bob")
	f.grant("bob", RequestApproval)
	f.must(f.delegate("d", "ann", "bob"))
	f.expect("a delegate", f.g.Decide(f.answer("bob"), "", ask(Issue)), true, ReasonAllowed)
	f.must(f.a.RevokeDelegation("d", 1))
	f.expect("a revoked delegation", f.g.Decide(f.answer("bob"), "", ask(Issue)), false, ReasonNoGrant)
	f.grant("root", ApproveSupportSession)
	f.must(f.g.AddSupportSession(session("s", "ann"), f.answer("root")))
	f.must(f.a.EndSupportSession("s", 1))
	f.expect("an ended support session", f.g.DecideInSupport(answer, "s", ask(View, "quantity")), false, ReasonSupportSession)
}

func TestNoOtherPackageImportsThePermissionPackage(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	const forbidden = "github.com/myblueton-sw/hybrid_billing/modules/auth"
	exempt := map[string]bool{filepath.Join(root, "modules", "auth"): true, filepath.Join(root, "modules", "gate"): true}
	scanned, own := 0, 0
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if name := entry.Name(); path != root && (strings.HasPrefix(name, ".") || name == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		scanned++
		for _, imported := range file.Imports {
			name := strings.Trim(imported.Path.Value, "\"`")
			if name != forbidden && !strings.HasPrefix(name, forbidden+"/") {
				continue
			}
			if exempt[filepath.Dir(path)] {
				own++
				continue
			}
			t.Errorf("%s imports %s", path, name)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if scanned < 20 || own == 0 {
		t.Errorf("the scan read %d files and found %d imports inside the gate", scanned, own)
	}
}

func TestTheAdapterMapsKindAndState(t *testing.T) {
	f := setup(t)
	f.member("ann")
	f.member("bob")
	f.must(f.w.CreateServiceUser(f.login("root"), "w1", "svc", "ann"))
	f.must(f.in.SuspendUser(operator, "ann", 1))
	f.must(f.in.DepartUser(operator, "bob", 1))
	source := adapter{view: f.reg.Subjects()}
	for id, want := range map[string]struct {
		kind  string
		state string
		epoch uint64
	}{"root": {"human", "active", 1}, "ann": {"human", "suspended", 2}, "bob": {"human", "suspended", 2}, "svc": {"service", "active", 1}} {
		got, found := source.Subject(id)
		if !found || got.ID != id || string(got.Kind) != want.kind || string(got.State) != want.state || got.Epoch != want.epoch {
			t.Errorf("%s: %+v found %v; want %+v", id, got, found, want)
		}
	}
	if got, found := source.Subject("nobody"); found || got.ID != "" {
		t.Errorf("an unknown user: %+v %v", got, found)
	}
	if joined, active := source.Member("w1", "root"); !active || joined != 1 {
		t.Errorf("membership through the adapter: %d %v", joined, active)
	}
	if source.Generation() != f.reg.Subjects().Generation() || source.Generation() == 0 {
		t.Errorf("generation through the adapter: %d", source.Generation())
	}
}

func TestRechecksOfEveryKindFailWithTheAnswer(t *testing.T) {
	f := setup(t)
	f.member("ann")
	f.member("bob")
	f.grant("bob", Approve)
	f.grant("root", ApproveSupportSession)
	f.must(f.g.AddSupportSession(session("s", "ann"), f.answer("root")))
	inSupport := f.expect("the support decision", f.g.DecideInSupport(f.answer("ann"), "s", ask(View, "quantity")), true, ReasonAllowed)
	approval := f.expect("the approval", f.g.CheckApproval(f.answer("bob"), "ann", ask(Approve)), true, ReasonAllowed)
	f.must(f.w.SuspendMembership(f.login("root"), "w1", "bob", 1))
	f.expect("an approval recheck after the approver's membership was suspended", f.g.RecheckApproval(approval.Stamp, "ann", ask(Approve)), false, ReasonIdentity)
	f.must(f.w.ChangeRole(f.login("root"), "w1", "ann", identity.Administrator, 1))
	f.expect("a support recheck after the actor's role changed", f.g.RecheckInSupport(inSupport.Stamp, "s", ask(View, "quantity")), false, ReasonIdentity)
	f.must(f.w.CreateServiceUser(f.login("root"), "w1", "svc", "ann"))
	f.must(f.w.AddCredential(f.login("root"), "w1", "svc", "c1", v("c1"), identity.Validity{NotBefore: start, NotAfter: start.Add(time.Hour)}))
	f.grant("svc", Issue)
	service, err := f.b.AccessByCredential("c1", v("c1"))
	f.must(err)
	stamp := f.expect("a service decision", f.g.Decide(service, "", ask(Issue)), true, ReasonAllowed).Stamp
	f.expect("a service stamp in a support recheck", f.g.RecheckInSupport(stamp, "s", ask(View, "quantity")), false, ReasonSupportSession)

	other, _, err := New(f.reg, catalog)
	f.must(err)
	second := f.expect("a second gate over the same identity registry has no grants", other.Decide(service, "", ask(Issue)), false, ReasonNoGrant)
	if second.Stamp.issued {
		t.Errorf("a denial carries a stamp")
	}
	f.expect("a stamp of this gate presented to the second gate", other.Recheck(stamp, "", ask(Issue)), false, ReasonNoGrant)
}

func TestAdmissionClockAndCatalogue(t *testing.T) {
	var none *Gate
	for _, g := range []*Gate{none, {}} {
		if g.Admit(identity.Answer{}) != AdmitUnavailable || g.Clock() != nil || g.Describes(Issue, "usage_statement", nil) {
			t.Errorf("a gate that is not built admits, tells the time or describes a permission")
		}
	}
	f := setup(t)
	f.member("ann")
	session := f.login("ann")
	answer, err := f.b.Access(session, "w1")
	f.must(err)
	if got := f.g.Admit(answer); got != Admitted {
		t.Errorf("a current answer: %s", got)
	}
	if got := f.g.Admit(identity.Answer{}); got != NotCurrent {
		t.Errorf("an empty answer: %s", got)
	}
	f.must(f.a.SetAvailable(false))
	if got := f.g.Admit(answer); got != Admitted {
		t.Errorf("admission follows the identity, not the permission registry: %s", got)
	}
	f.must(f.a.SetAvailable(true))
	f.must(f.b.EndSession(session))
	if got := f.g.Admit(answer); got != NotCurrent {
		t.Errorf("an answer of an ended session: %s", got)
	}
	if f.g.Clock() == nil || !f.g.Clock()().Equal(*f.now) {
		t.Errorf("the gate's clock is not the identity registry's")
	}
	for name, c := range map[string]struct {
		action Action
		kind   string
		fields []string
		want   bool
	}{
		"an action on a kind":                 {Issue, "usage_statement", nil, true},
		"a field action with a field":         {View, "usage_statement", []string{"quantity"}, true},
		"an unknown kind":                     {Issue, "invoice", nil, false},
		"an unknown action":                   {"erase", "usage_statement", nil, false},
		"a field action without fields":       {View, "usage_statement", nil, false},
		"fields on an action without fields":  {Issue, "usage_statement", []string{"quantity"}, false},
		"a field the catalogue does not hold": {View, "usage_statement", []string{"margin"}, false},
	} {
		if got := f.g.Describes(c.action, c.kind, c.fields); got != c.want {
			t.Errorf("%s: %v", name, got)
		}
	}
}
