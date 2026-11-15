package assistant

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

var (
	start     = time.Date(2026, 11, 15, 9, 0, 0, 0, time.UTC)
	always    = gate.Validity{NotBefore: start.Add(-time.Hour), NotAfter: start.Add(100000 * time.Hour)}
	operator  = identity.Operator{Reference: "synthetic-operator", Reason: "synthetic-reason"}
	idLimits  = identity.Limits{BootstrapValidity: time.Hour, InvitationValidity: 72 * time.Hour, CredentialValidity: 720 * time.Hour, SessionIdle: 30 * time.Minute, SessionAbsolute: 8 * time.Hour, Reauthentication: 5 * time.Minute, Attempts: 3, PendingInvitations: 1000, SessionsPerUser: 100000, CredentialsPerServiceUser: 1000, PageSize: 100}
	catalog   = gate.Catalog{Kinds: []string{"statement", "account", "recipient"}, Fields: map[string]gate.FieldClass{"quantity": gate.General, "unit_cost": gate.Cost}}
	cmdLimits = commands.Limits{Fields: 8, Value: 200, Rationale: 200, Operations: 1000, Journal: 100000, Workspaces: 1, Users: 10, WorkspaceJournal: 100000,
		UserJournal: 10000, WorkspaceOperations: 1000, UserOperations: 100, Replay: 24 * time.Hour, Retention: 400 * 24 * time.Hour, Burst: 100000, Refill: time.Millisecond}
	limits    = Limits{Workspaces: 3, Tools: 20, Jobs: 20, Conversations: 20, Cards: 3, Calls: 5, Targets: 3, Duration: 2 * time.Hour, Retention: 24 * time.Hour, Fields: 3, Bytes: 8}
	protected = []string{"account", "recipient"}
	ann       = Actor{ID: "ann", Kind: KindHuman}
	bob       = Actor{ID: "bob", Kind: KindHuman}
	cat       = Actor{ID: "cat", Kind: KindHuman}
	bot       = Actor{ID: "bot", Kind: KindAssistant}
	sweeper   = Actor{ID: "sweeper", Kind: KindSystem}
)

type stub struct {
	mu      sync.Mutex
	targets map[string]*commands.Target
	fields  map[string]string
	panics  bool
	applied []commands.Context
}

func (s *stub) Resolve(workspace, target string) (commands.Target, commands.Resolution) {
	s.mu.Lock()
	defer s.mu.Unlock()
	found := s.targets[workspace+"/"+target]
	if found == nil {
		return commands.Target{}, commands.NotFound
	}
	return *found, commands.Found
}

func (s *stub) Apply(context commands.Context) (commands.Outcome, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.applied = append(s.applied, context)
	if s.panics {
		panic("synthetic apply panic")
	}
	if context.Expected == "" {
		copied := map[string]string{}
		for name, value := range s.fields {
			copied[name] = value
		}
		return commands.Outcome{Fields: copied}, nil
	}
	found := s.targets[context.Workspace+"/"+context.Target]
	found.Revision = fmt.Sprintf("r%d", len(s.applied)+1)
	return commands.Outcome{ResultRevision: found.Revision, BusinessState: "recorded"}, nil
}

func (s *stub) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.applied)
}

func (s *stub) last() commands.Context {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.applied[len(s.applied)-1]
}

func (s *stub) revision(target string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.targets["w1/"+target].Revision
}

type world struct {
	t     testing.TB
	now   *time.Time
	in    *identity.Installation
	b     *identity.Boundary
	w     *identity.Workspace
	a     *gate.Administration
	c     *commands.Registry
	h     *stub
	r     *Registry
	count *int
}

func v(label string) identity.Verifier {
	return identity.Verifier(sha256.Sum256([]byte("synthetic-secret-" + label)))
}

func triple(subject string) identity.Triple {
	return identity.Triple{Connection: "synthetic-connection", Issuer: "https://issuer.invalid", Subject: subject}
}

func setupWith(t testing.TB, l Limits) world {
	t.Helper()
	now, count := start, 0
	clock := func() time.Time { return now }
	reg, err := identity.New("synthetic-installation", idLimits, clock)
	if err != nil {
		t.Fatal(err)
	}
	g, a, err := gate.New(reg, catalog)
	if err != nil {
		t.Fatal(err)
	}
	c, err := commands.New(g, cmdLimits)
	if err != nil {
		t.Fatal(err)
	}
	h := &stub{targets: map[string]*commands.Target{
		"w1/t1": {CustomerID: "c1", Revision: "r1", AuthorID: "ann"},
		"w1/t2": {CustomerID: "c2", Revision: "r1", AuthorID: "ann"},
		"w1/t3": {CustomerID: "c1", Revision: "r1", AuthorID: "ann"},
		"w1/t4": {CustomerID: "c1", Revision: "r1", AuthorID: "ann"},
		"w1/t5": {CustomerID: "c1", Revision: "r1", AuthorID: "ann"},
	}, fields: map[string]string{"name": "가나다라", "total": "1234567890", "yank": "abcdefgh", "zeta": "z", "secret": "s"}}
	r, err := NewRegistry(Config{Commands: c, Client: "asst", Protected: protected}, l, clock)
	if err != nil {
		t.Fatal(err)
	}
	f := world{t: t, now: &now, in: reg.Installation(), b: reg.Boundary(), w: reg.Workspace(), a: a, c: c, h: h, r: r, count: &count}
	f.must(f.in.ArmBootstrap(v("boot"), identity.Validity{NotBefore: start, NotAfter: start.Add(time.Hour)}, "w1"))
	f.must(f.b.Bootstrap(v("boot"), "root", triple("root"), ""))
	for _, d := range []commands.Definition{
		{ID: "statement.read", Permission: commands.Permission{Action: gate.View, Kind: "statement", Fields: []string{"quantity"}}, Handler: h, Fields: []commands.Field{{Name: "ref", Type: commands.ID}}},
		{ID: "statement.cost", Permission: commands.Permission{Action: gate.ViewCost, Kind: "statement", Fields: []string{"unit_cost"}}, Handler: h},
		{ID: "statement.draft", Permission: commands.Permission{Action: gate.RequestApproval, Kind: "statement"}, Mutating: true, Handler: h,
			Fields: []commands.Field{{Name: "note", Type: commands.Text}, {Name: "amount", Type: commands.Decimal}, {Name: "count", Type: commands.Integer}}},
		{ID: "statement.simulate", Permission: commands.Permission{Action: gate.View, Kind: "statement", Fields: []string{"quantity"}}, Mutating: true, Handler: h,
			Fields: []commands.Field{{Name: "count", Type: commands.Integer}}},
		{ID: "statement.issue", Permission: commands.Permission{Action: gate.Issue, Kind: "statement"}, Mutating: true, Rationale: true, Handler: h,
			Fields: []commands.Field{{Name: "count", Type: commands.Integer}, {Name: "amount", Type: commands.Decimal}}},
		{ID: "statement.approve", Permission: commands.Permission{Action: gate.Approve, Kind: "statement"}, Approval: true, Mutating: true, Handler: h},
		{ID: "statement.download", Permission: commands.Permission{Action: gate.Download, Kind: "statement", Fields: []string{"quantity"}}, Handler: h},
		{ID: "statement.planned", Permission: commands.Permission{Action: gate.Deliver, Kind: "statement"}, Mutating: true},
		{ID: "account.change", Permission: commands.Permission{Action: gate.Issue, Kind: "account"}, Mutating: true, Handler: h},
		{ID: "recipient.change", Permission: commands.Permission{Action: gate.Issue, Kind: "recipient"}, Mutating: true, Handler: h},
	} {
		f.must(c.Register(d))
	}
	f.member("ann")
	f.member("bob")
	f.member("cat")
	for _, action := range []gate.Action{gate.View, gate.RequestApproval, gate.Issue, gate.Download} {
		f.grant("ann", "", action, "c1")
	}
	f.grant("bob", "", gate.Approve, "c1")
	f.grant("bob", "", gate.Issue, "c1")
	_, err = f.a.PutClient(gate.Client{ID: "asst", State: gate.Active}, 0)
	f.must(err)
	f.grant("", "asst", gate.View, "c1")
	f.grant("", "asst", gate.RequestApproval, "c1")
	return f
}

func setup(t testing.TB) world { return setupWith(t, limits) }

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

func (f world) answer(id string) gate.Answer {
	f.t.Helper()
	answer, err := f.b.Access(f.login(id), "w1")
	f.must(err)
	return answer
}

func (f world) member(id string) {
	f.t.Helper()
	f.must(f.in.CreateUser(operator, id, triple(id), ""))
	invitation := f.next("invitation")
	f.must(f.w.Invite(f.login("root"), "w1", invitation, identity.Member, v(invitation), identity.Validity{NotBefore: *f.now, NotAfter: f.now.Add(48 * time.Hour)}, id))
	f.must(f.w.Accept("w1", invitation, v(invitation), f.login(id)))
}

func (f world) grant(subject, client string, action gate.Action, customer string) {
	f.t.Helper()
	var fields []string
	if action == gate.View || action == gate.Download {
		fields = []string{"quantity"}
	}
	f.must(f.a.AddGrant(gate.Grant{ID: f.next("grant"), SubjectID: subject, ClientID: client, Validity: always, Terms: gate.Terms{Action: action, Kind: "statement", Fields: fields,
		Scope: gate.Scope{WorkspaceID: "w1", Customer: gate.One(customer), LegalEntity: gate.All(), Contract: gate.All()}, Period: gate.AnyPeriod()}}))
}

func (f world) advance(d time.Duration) { *f.now = f.now.Add(d) }

func (f world) expose(command, stage string, results ...string) ToolView {
	f.t.Helper()
	_, err := f.r.ProposeTool("w1", Tool{Command: command, Stage: stage, Results: results}, ann)
	f.must(err)
	view, err := f.r.ApproveTool("w1", command, bob)
	f.must(err)
	return view
}

func (f world) conversation(id string, owner Actor) {
	f.t.Helper()
	_, err := f.r.OpenConversation("w1", Conversation{ID: id, Owner: owner.ID, Expires: f.now.Add(24 * time.Hour)}, owner)
	f.must(err)
}

func (f world) job(id, conversation string) {
	f.t.Helper()
	_, err := f.r.CreateJob("w1", Job{ID: id, Purpose: "synthetic purpose"}, ann)
	f.must(err)
	f.conversation(conversation, ann)
	_, err = f.r.Attach("w1", id, conversation, ann)
	f.must(err)
}

func (f world) call(conversation, tool, target, expected string, input ...commands.Value) Invocation {
	return Invocation{Conversation: conversation, Tool: tool, Target: target, Expected: expected, Input: input, Answer: f.answer("ann")}
}

func (f world) lastRecord() commands.Record {
	f.t.Helper()
	records := f.c.Journal(0, 1000000)
	if len(records) == 0 {
		f.t.Fatal("the journal is empty")
	}
	return records[len(records)-1]
}

func expect(t testing.TB, name string, err, want error) {
	t.Helper()
	if !errors.Is(err, want) || want == nil && err != nil {
		t.Errorf("%s: %v; want %v", name, err, want)
	}
}

func reason(t testing.TB, name string, err error, want string) {
	t.Helper()
	var refusal Refusal
	if !errors.As(err, &refusal) || refusal.Reason != want {
		t.Errorf("%s: %v; want %s", name, err, want)
	}
}

func second[T any](_ T, err error) error { return err }

type fake struct {
	mu      sync.Mutex
	entries []commands.Entry
	calls   []commands.Call
	result  func(commands.Call) commands.Result
}

func (c *fake) Catalogue() []commands.Entry {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]commands.Entry(nil), c.entries...)
}

func (c *fake) Execute(call commands.Call) commands.Result {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls = append(c.calls, call)
	if c.result != nil {
		return c.result(call)
	}
	return commands.Result{Status: commands.Completed, Operation: "op-" + call.Request}
}
