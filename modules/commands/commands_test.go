package commands

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

var (
	start      = time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	always     = gate.Validity{NotBefore: start.Add(-time.Hour), NotAfter: start.Add(1000 * time.Hour)}
	operator   = identity.Operator{Reference: "synthetic-operator", Reason: "synthetic-reason"}
	idLimits   = identity.Limits{BootstrapValidity: time.Hour, InvitationValidity: 72 * time.Hour, CredentialValidity: 720 * time.Hour, SessionIdle: 30 * time.Minute, SessionAbsolute: 8 * time.Hour, Reauthentication: 5 * time.Minute, Attempts: 3, PendingInvitations: 1000, SessionsPerUser: 100000, CredentialsPerServiceUser: 1000, PageSize: 100}
	catalog    = gate.Catalog{Kinds: []string{"statement", "adjustment"}, Fields: map[string]gate.FieldClass{"quantity": gate.General, "unit_cost": gate.Cost}}
	testLimits = Limits{Fields: 6, Value: 40, Rationale: 60, Operations: 1000, Journal: 100000}
)

type stub struct {
	mu          sync.Mutex
	targets     map[string]*Target
	applies     int
	resolves    int
	apply       func(Context, *Target) (Outcome, error)
	unavailable bool
	panics      bool
	resolution  Resolution
}

func (s *stub) Resolve(workspace, target string) (Target, Resolution) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.resolves++
	if s.panics {
		panic("synthetic resolve panic")
	}
	if s.unavailable {
		return Target{}, Unavailable
	}
	if s.resolution != 0 {
		return Target{CustomerID: "c1", Revision: "r1", AuthorID: "ann"}, s.resolution
	}
	found := s.targets[workspace+"/"+target]
	if found == nil {
		return Target{}, NotFound
	}
	return *found, Found
}

func (s *stub) Apply(context Context) (Outcome, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.applies++
	found := s.targets[context.Workspace+"/"+context.Target]
	if s.apply != nil {
		return s.apply(context, found)
	}
	if context.Expected != "" && context.Expected != found.Revision {
		return Outcome{}, Fail(RevisionConflict)
	}
	if context.Expected == "" {
		return Outcome{Fields: map[string]string{"revision_read": found.Revision}}, nil
	}
	found.Revision = fmt.Sprintf("r%d", s.applies+1)
	return Outcome{ResultRevision: found.Revision, BusinessState: "recorded", Fields: map[string]string{"echo": context.Fields["amount"]}}, nil
}

func (s *stub) count() (int, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.resolves, s.applies
}

type world struct {
	t     testing.TB
	now   *time.Time
	armed *int
	hook  *func()
	reg   *identity.Registry
	in    *identity.Installation
	b     *identity.Boundary
	w     *identity.Workspace
	g     *gate.Gate
	a     *gate.Administration
	c     *Registry
	h     *stub
	count *int
}

func v(label string) identity.Verifier {
	return identity.Verifier(sha256.Sum256([]byte("synthetic-secret-" + label)))
}

func triple(subject string) identity.Triple {
	return identity.Triple{Connection: "synthetic-connection", Issuer: "https://issuer.invalid", Subject: subject}
}

func setupWith(t testing.TB, limits Limits) world {
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
	reg, err := identity.New("synthetic-installation", idLimits, clock)
	if err != nil {
		t.Fatal(err)
	}
	g, a, err := gate.New(reg, catalog)
	if err != nil {
		t.Fatal(err)
	}
	c, err := New(g, limits)
	if err != nil {
		t.Fatal(err)
	}
	f := world{t: t, now: &now, armed: &armed, hook: &hook, reg: reg, in: reg.Installation(), b: reg.Boundary(), w: reg.Workspace(), g: g, a: a, c: c, count: &count,
		h: &stub{targets: map[string]*Target{"w1/t1": {CustomerID: "c1", Revision: "r1", AuthorID: "ann"}, "w1/t2": {CustomerID: "c2", Revision: "r1", AuthorID: "bob"}}}}
	f.must(f.in.ArmBootstrap(v("boot"), identity.Validity{NotBefore: start, NotAfter: start.Add(time.Hour)}, "w1"))
	f.must(f.b.Bootstrap(v("boot"), "root", triple("root"), ""))
	for _, definition := range []Definition{
		{ID: "statement.adjust", Permission: Permission{Action: gate.Issue, Kind: "statement"}, Mutating: true, Rationale: true, Handler: f.h,
			Fields: []Field{{Name: "amount", Type: Decimal, Required: true}, {Name: "count", Type: Integer}, {Name: "final", Type: Boolean}, {Name: "note", Type: Text},
				{Name: "at", Type: Timestamp}, {Name: "basis", Type: Enum, Values: []string{"net", "gross"}}}},
		{ID: "statement.read", Permission: Permission{Action: gate.View, Kind: "statement", Fields: []string{"quantity"}}, Handler: f.h, Fields: []Field{{Name: "ref", Type: ID}}},
		{ID: "statement.approve", Permission: Permission{Action: gate.Approve, Kind: "statement"}, Approval: true, Mutating: true, Handler: f.h},
		{ID: "statement.planned", Permission: Permission{Action: gate.Deliver, Kind: "statement"}, Mutating: true},
	} {
		f.must(c.Register(definition))
	}
	return f
}

func setup(t testing.TB) world { return setupWith(t, testLimits) }

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

func (f world) grant(subject string, action gate.Action, customer string, fields ...string) string {
	f.t.Helper()
	id := f.next("grant")
	f.must(f.a.AddGrant(gate.Grant{ID: id, SubjectID: subject, Validity: always, Terms: gate.Terms{Action: action, Kind: "statement",
		Scope: gate.Scope{WorkspaceID: "w1", Customer: gate.One(customer), LegalEntity: gate.All(), Contract: gate.All()}, Fields: fields, Period: gate.AnyPeriod()}}))
	return id
}

func adjust(answer gate.Answer, key, amount string) Call {
	return Call{Answer: answer, Request: "req-1", Command: "statement.adjust", Target: "t1", Key: key, Expected: "r1", Rationale: "synthetic rationale",
		Fields: []Value{{"amount", amount}}}
}

func (f world) code(name string, got Result, want Code) Result {
	f.t.Helper()
	switch {
	case want == "" && got.Err != nil:
		f.t.Errorf("%s: error %s; want success", name, got.Err.Code)
	case want != "" && (got.Err == nil || got.Err.Code != want):
		f.t.Errorf("%s: %+v; want %s", name, got, want)
	}
	if got.Err != nil {
		rule := errorRules[got.Err.Code]
		if got.Err.Status != rule.status || got.Err.Next == "" || got.Status != "" || got.Operation != "" || got.Fields != nil || got.Revision != "" {
			f.t.Errorf("%s: an error result carries more than the error: %+v %+v", name, got, got.Err)
		}
	}
	return got
}

func (f world) last() Record {
	f.t.Helper()
	records := f.c.Journal(0, 1000000)
	if len(records) == 0 {
		f.t.Fatal("the journal is empty")
	}
	return records[len(records)-1]
}

func basic(t testing.TB) (world, gate.Answer) {
	f := setup(t)
	f.member("ann")
	f.member("bob")
	f.grant("ann", gate.Issue, "c1")
	f.grant("ann", gate.View, "c1", "quantity")
	f.grant("bob", gate.Approve, "c1")
	return f, f.answer("ann")
}

func TestRegistration(t *testing.T) {
	f := setup(t)
	good := Definition{ID: "x.do", Permission: Permission{Action: gate.Issue, Kind: "statement"}, Mutating: true, Fields: []Field{{Name: "a", Type: Text}}}
	cases := map[string]func(*Definition){
		"an invalid identifier":             func(d *Definition) { d.ID = "x do" },
		"no action":                         func(d *Definition) { d.Permission.Action = "" },
		"an invalid kind":                   func(d *Definition) { d.Permission.Kind = "" },
		"the approval flag without approve": func(d *Definition) { d.Approval = true },
		"approve without the approval flag": func(d *Definition) { d.Permission.Action = gate.Approve },
		"an approval that does not mutate":  func(d *Definition) { d.Approval, d.Permission.Action, d.Mutating = true, gate.Approve, false },
		"an unknown field type":             func(d *Definition) { d.Fields = []Field{{Name: "a", Type: "money"}} },
		"an enum without values":            func(d *Definition) { d.Fields = []Field{{Name: "a", Type: Enum}} },
		"values on another type":            func(d *Definition) { d.Fields = []Field{{Name: "a", Type: Text, Values: []string{"x"}}} },
		"a repeated enum value":             func(d *Definition) { d.Fields = []Field{{Name: "a", Type: Enum, Values: []string{"x", "x"}}} },
		"a field named twice":               func(d *Definition) { d.Fields = []Field{{Name: "a", Type: Text}, {Name: "a", Type: ID}} },
		"an invalid field name":             func(d *Definition) { d.Fields = []Field{{Name: "a b", Type: Text}} },
		"a repeated permission field": func(d *Definition) {
			d.Permission.Action, d.Permission.Fields = gate.View, []string{"quantity", "quantity"}
		},
		"fields above the limit": func(d *Definition) {
			d.Fields = nil
			for index := 0; index < 7; index++ {
				d.Fields = append(d.Fields, Field{Name: fmt.Sprintf("f%d", index), Type: Text})
			}
		},
	}
	for name, change := range cases {
		definition := good
		change(&definition)
		if err := f.c.Register(definition); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
	f.must(f.c.Register(good))
	changed := good
	changed.Rationale = true
	if err := f.c.Register(changed); !errors.Is(err, ErrConflict) {
		t.Errorf("registering an identifier twice: %v", err)
	}
	good.Fields[0].Name = "tampered"
	entries := f.c.Catalogue()
	ids := []string{}
	for _, entry := range entries {
		ids = append(ids, entry.ID)
		if entry.ID == "x.do" && (entry.Fields[0].Name != "a" || entry.Rationale || entry.Supported) {
			t.Errorf("the stored definition changed: %+v", entry)
		}
	}
	if !reflect.DeepEqual(ids, []string{"statement.adjust", "statement.approve", "statement.planned", "statement.read", "x.do"}) {
		t.Errorf("catalogue order %v", ids)
	}
	entries[0].Fields[0].Name = "tampered"
	if f.c.Catalogue()[0].Fields[0].Name != "amount" {
		t.Errorf("the catalogue aliases the registry")
	}
	for name, limits := range map[string]Limits{"fields": {0, 1, 1, 1, 1}, "value": {1, 0, 1, 1, 1}, "rationale": {1, 1, 0, 1, 1}, "operations": {1, 1, 1, 0, 1}, "journal": {1, 1, 1, 1, 0}} {
		if _, err := New(f.g, limits); !errors.Is(err, ErrInvalid) {
			t.Errorf("a zero %s limit: %v", name, err)
		}
	}
	if _, err := New(nil, testLimits); !errors.Is(err, ErrInvalid) {
		t.Errorf("a nil gate: %v", err)
	}
	if _, err := New(&gate.Gate{}, testLimits); !errors.Is(err, ErrInvalid) {
		t.Errorf("an unconstructed gate: %v", err)
	}
}

func TestACompletedMutation(t *testing.T) {
	f, ann := basic(t)
	got := f.code("the call", f.c.Execute(adjust(ann, "k1", "12.50")), "")
	if got.Status != Completed || got.Target != "t1" || got.Revision != "r2" || got.State != "recorded" || got.Fields["echo"] != "12.50" || got.Replay ||
		!strings.HasPrefix(got.Operation, "op-") || len(got.Operation) != 35 || !got.AsOf.Equal(start) || got.Request != "req-1" || got.Job != "" {
		t.Errorf("result %+v", got)
	}
	record := f.last()
	if record.Sequence == 0 || record.Command != "statement.adjust" || record.User != "ann" || record.Client != "" || record.Workspace != "w1" || record.Target != "t1" ||
		record.Outcome != "completed" || record.Operation != got.Operation || record.Replay || len(record.Digest) != 64 || record.Request != "req-1" || !record.At.Equal(start) {
		t.Errorf("record %+v", record)
	}
	if _, applies := f.h.count(); applies != 1 {
		t.Errorf("%d applies", applies)
	}
	read := f.code("a reading call", f.c.Execute(Call{Answer: ann, Request: "req-2", Command: "statement.read", Target: "t1"}), "")
	if read.Status != Completed || read.Operation != "" || read.Revision != "r2" || read.State != "" || read.Fields["revision_read"] != "r2" {
		t.Errorf("reading result %+v", read)
	}
}

func TestStepsRefuseInOrder(t *testing.T) {
	f, ann := basic(t)
	bob := f.answer("bob")
	stale := f.answer("ann")
	f.must(f.b.EndSession(f.login("ann")))
	ended := f.login("ann")
	gone, err := f.b.Access(ended, "w1")
	f.must(err)
	f.must(f.b.EndSession(ended))
	before := len(f.c.Journal(0, 1000000))
	f.code("an answer that is not current, for an unknown command", f.c.Execute(Call{Answer: gone, Request: "r", Command: "nothing.here", Target: "t1"}), Unauthenticated)
	f.code("the zero answer", f.c.Execute(Call{Request: "r", Command: "statement.read", Target: "t1"}), Unauthenticated)
	if len(f.c.Journal(0, 1000000)) != before {
		t.Errorf("a call that was not admitted is in the journal")
	}
	if resolves, _ := f.h.count(); resolves != 0 {
		t.Errorf("a handler ran for a call that was not admitted")
	}
	f.code("an unknown command with a malformed body", f.c.Execute(Call{Answer: ann, Request: "r", Command: "nothing.here", Target: "bad target", Fields: []Value{{"x", "y"}}}), UnsupportedCommand)
	f.code("a command without a handler and a malformed body", f.c.Execute(Call{Answer: ann, Request: "r", Command: "statement.planned", Target: "bad target"}), UnsupportedCapability)
	malformed := adjust(bob, "k", "1e3")
	malformed.Target = "t2"
	got := f.code("a malformed body for a target the caller may not use", f.c.Execute(malformed), InvalidRequest)
	if !reflect.DeepEqual(got.Err.Fields, []string{"amount"}) {
		t.Errorf("fields %v", got.Err.Fields)
	}
	if resolves, _ := f.h.count(); resolves != 0 {
		t.Errorf("the target was resolved before the shape was checked")
	}
	forbidden := adjust(ann, "k", "1")
	forbidden.Target = "t2"
	missing := adjust(ann, "k", "1")
	missing.Target = "t9"
	first := f.code("a forbidden target", f.c.Execute(forbidden), PermissionDenied)
	recordForbidden := f.last()
	second := f.code("a missing target", f.c.Execute(missing), PermissionDenied)
	recordMissing := f.last()
	first.Err.Request, second.Err.Request = "", ""
	if !reflect.DeepEqual(first, second) {
		t.Errorf("a forbidden and a missing target differ: %+v and %+v", first, second)
	}
	recordForbidden.Sequence, recordForbidden.Target, recordForbidden.Digest = 0, "", ""
	recordMissing.Sequence, recordMissing.Target, recordMissing.Digest = 0, "", ""
	if recordForbidden != recordMissing {
		t.Errorf("their records differ in shape: %+v and %+v", recordForbidden, recordMissing)
	}
	done := f.code("the first use of a key", f.c.Execute(adjust(ann, "k1", "1")), "")
	denied := adjust(bob, "k1", "1")
	f.code("a caller without the permission asking with a used key", f.c.Execute(denied), PermissionDenied)
	conflict := adjust(ann, "k1", "2")
	conflict.Expected = "r0"
	f.code("an idempotency conflict before a stale revision", f.c.Execute(conflict), IdempotencyConflict)
	replay := f.code("a replay after the revision moved", f.c.Execute(adjust(stale, "k1", "1")), "")
	if !replay.Replay || replay.Operation != done.Operation || replay.Revision != done.Revision {
		t.Errorf("replay %+v; want operation %s", replay, done.Operation)
	}
	f.code("a new key with the old revision", f.c.Execute(adjust(ann, "k2", "1")), RevisionConflict)
	if _, applies := f.h.count(); applies != 1 {
		t.Errorf("%d applies", applies)
	}
}

func TestShape(t *testing.T) {
	f, ann := basic(t)
	base := func() Call { return adjust(ann, f.next("key"), "1") }
	valid := map[string][]Value{
		"every type":                   {{"amount", "-12.050"}, {"count", "-7"}, {"final", "true"}, {"note", "free text, any form"}, {"at", "2026-11-13T09:00:00+09:00"}, {"basis", "net"}},
		"zero":                         {{"amount", "0"}, {"count", "0"}, {"final", "false"}, {"at", "2026-11-13T00:00:00Z"}, {"basis", "gross"}},
		"minus zero":                   {{"amount", "-0"}},
		"a fraction":                   {{"amount", "0.001"}},
		"a fraction of a second":       {{"amount", "1"}, {"at", "2026-11-13T09:00:00.123456789-03:30"}},
		"a value exactly at the limit": {{"amount", "1"}, {"note", strings.Repeat("n", 40)}},
		"text counted in bytes":        {{"amount", "1"}, {"note", strings.Repeat("\uac00", 13)}},
	}
	for name, fields := range valid {
		call := base()
		call.Fields = fields
		call.Expected = f.h.targets["w1/t1"].Revision
		f.code(name, f.c.Execute(call), "")
	}
	invalid := map[string]struct {
		field string
		value string
	}{
		"an exponent": {"amount", "1e3"}, "not a number": {"amount", "NaN"}, "infinity": {"amount", "Infinity"}, "a plus sign": {"amount", "+1"},
		"a comma": {"amount", "1,5"}, "a leading dot": {"amount", ".5"}, "a trailing dot": {"amount", "1."}, "a bare minus": {"amount", "-"},
		"leading zeros": {"amount", "01"}, "an empty decimal": {"amount", ""}, "a space": {"amount", " 1"}, "a grouping": {"amount", "1 000"},
		"a decimal integer": {"count", "1.0"}, "an integer with zeros": {"count", "007"}, "an integer with a plus": {"count", "+1"},
		"a capital boolean": {"final", "True"}, "a numeric boolean": {"final", "1"},
		"a timestamp without offset": {"at", "2026-11-13T09:00:00"}, "a date": {"at", "2026-11-13"},
		"an hour of one digit": {"at", "2026-11-13T9:00:00Z"}, "a comma fraction": {"at", "2026-11-13T09:00:00,5Z"}, "an offset of a day": {"at", "2026-11-13T09:00:00+24:00"},
		"a space for the separator": {"at", "2026-11-13 09:00:00Z"}, "a thirteenth month": {"at", "2026-13-13T09:00:00Z"}, "an offset without a colon": {"at", "2026-11-13T09:00:00+0900"},
		"a lowercase separator": {"at", "2026-11-13t09:00:00Z"}, "a lowercase zone": {"at", "2026-11-13T09:00:00z"}, "a leap second": {"at", "2026-12-31T23:59:60Z"},
		"a fraction without digits": {"at", "2026-11-13T09:00:00.Z"}, "the thirtieth of February": {"at", "2026-02-30T09:00:00Z"},
		"an unknown enum value": {"basis", "tax"}, "an enum in capitals": {"basis", "NET"},
		"a long text": {"note", strings.Repeat("n", 41)}, "text of fourteen characters and forty two bytes": {"note", strings.Repeat("\uac00", 14)},
	}
	for name, c := range invalid {
		call := base()
		if c.field == "amount" {
			call.Fields = []Value{{c.field, c.value}}
		} else {
			call.Fields = append(call.Fields, Value{c.field, c.value})
		}
		got := f.code(name, f.c.Execute(call), InvalidRequest)
		if got.Err != nil && !reflect.DeepEqual(got.Err.Fields, []string{c.field}) {
			t.Errorf("%s: fields %v", name, got.Err.Fields)
		}
		if got.Err != nil && strings.Contains(fmt.Sprint(got.Err.Fields), c.value) && c.value != "" && c.value != "1" {
			t.Errorf("%s: the value is in the error", name)
		}
	}
	structural := map[string]struct {
		change func(*Call)
		fields []string
	}{
		"a required field missing":  {func(c *Call) { c.Fields = nil }, []string{"amount"}},
		"an unknown field":          {func(c *Call) { c.Fields = append(c.Fields, Value{"other", "x"}) }, []string{"other"}},
		"a repeated field":          {func(c *Call) { c.Fields = append(c.Fields, Value{"amount", "2"}) }, []string{"amount"}},
		"a field name that is text": {func(c *Call) { c.Fields = append(c.Fields, Value{"free text name", "x"}) }, []string{Marker}},
		"more fields than declared": {func(c *Call) {
			c.Fields = []Value{{"amount", "1"}, {"count", "1"}, {"final", "true"}, {"note", "n"}, {"at", "2026-11-13T00:00:00Z"}, {"basis", "net"}, {"note", "n"}}
		}, []string{Marker}},
		"no key":               {func(c *Call) { c.Key = "" }, []string{"idempotency_key"}},
		"a key that is text":   {func(c *Call) { c.Key = "a key" }, []string{"idempotency_key"}},
		"no expected revision": {func(c *Call) { c.Expected = "" }, []string{"expected_revision"}},
		"no rationale":         {func(c *Call) { c.Rationale = "" }, []string{"rationale"}},
		"a long rationale":     {func(c *Call) { c.Rationale = strings.Repeat("r", 61) }, []string{"rationale"}},
		"a rationale of twenty one characters and sixty three bytes": {func(c *Call) { c.Rationale = strings.Repeat("\uac00", 21) }, []string{"rationale"}},
		"a target that is text":  {func(c *Call) { c.Target = "free text target" }, []string{"target_id"}},
		"a request that is text": {func(c *Call) { c.Request = "free text request" }, []string{"request_id"}},
		"a client that is text":  {func(c *Call) { c.Client = "free text client" }, []string{"client"}},
		"several at once":        {func(c *Call) { c.Key, c.Rationale, c.Fields = "", "", nil }, []string{"amount", "idempotency_key", "rationale"}},
	}
	for name, c := range structural {
		call := base()
		c.change(&call)
		got := f.code(name, f.c.Execute(call), InvalidRequest)
		if got.Err != nil && !reflect.DeepEqual(got.Err.Fields, c.fields) {
			t.Errorf("%s: fields %v; want %v", name, got.Err.Fields, c.fields)
		}
	}
	reading := Call{Answer: ann, Request: "r", Command: "statement.read", Target: "t1"}
	for name, change := range map[string]func(*Call){
		"a key on a reading command":       func(c *Call) { c.Key = "k" },
		"a revision on a reading command":  func(c *Call) { c.Expected = "r1" },
		"a rationale on a reading command": func(c *Call) { c.Rationale = "why" },
		"an invalid identifier field":      func(c *Call) { c.Fields = []Value{{"ref", "not an id"}} },
	} {
		call := reading
		change(&call)
		f.code(name, f.c.Execute(call), InvalidRequest)
	}
	f.code("a reading command with an optional identifier", f.c.Execute(Call{Answer: ann, Request: "r", Command: "statement.read", Target: "t1", Fields: []Value{{"ref", "x-1"}}}), "")
	textRequest := base()
	textRequest.Request = "customer name in a request id"
	got := f.c.Execute(textRequest)
	if got.Request != Marker || got.Err.Request != Marker || f.last().Request != Marker {
		t.Errorf("free text of a request identifier is echoed or recorded: %+v %+v", got, f.last())
	}
	textual := base()
	textual.Target, textual.Client, textual.Command = "customer name as target", "customer name as client", "statement.adjust"
	f.c.Execute(textual)
	if record := f.last(); record.Target != Marker || record.Client != Marker || record.Digest != Marker {
		t.Errorf("free text reached the journal: %+v", record)
	}
	unknown := Call{Answer: ann, Request: "r", Command: "customer name as command", Target: "t1"}
	f.code("free text as a command", f.c.Execute(unknown), UnsupportedCommand)
	if record := f.last(); record.Command != Marker {
		t.Errorf("free text of a command reached the journal: %+v", record)
	}
}

func TestPermissionsThroughTheGate(t *testing.T) {
	f, ann := basic(t)
	f.member("cat")
	f.code("no grant", f.c.Execute(adjust(f.answer("cat"), "k0", "1")), PermissionDenied)
	f.code("a reading call without the field grant", f.c.Execute(Call{Answer: f.answer("bob"), Request: "r", Command: "statement.read", Target: "t1"}), PermissionDenied)
	f.code("a reading call with the field grant", f.c.Execute(Call{Answer: ann, Request: "r", Command: "statement.read", Target: "t1"}), "")
	through := adjust(ann, "k1", "1")
	through.Client = "app"
	f.code("an unknown client", f.c.Execute(through), PermissionDenied)
	_, err := f.a.PutClient(gate.Client{ID: "app", State: gate.Active}, 0)
	f.must(err)
	f.code("a client without the grant", f.c.Execute(through), PermissionDenied)
	f.must(f.a.AddGrant(gate.Grant{ID: "client-grant", ClientID: "app", Validity: always, Terms: gate.Terms{Action: gate.Issue, Kind: "statement",
		Scope: gate.Scope{WorkspaceID: "w1", Customer: gate.One("c1"), LegalEntity: gate.All(), Contract: gate.All()}, Period: gate.AnyPeriod()}}))
	done := f.code("a client with the grant", f.c.Execute(through), "")
	if f.last().Client != "app" {
		t.Errorf("the client is not recorded")
	}
	direct := adjust(ann, "k1", "1")
	direct.Expected = f.h.targets["w1/t1"].Revision
	other := f.code("the same key without the client is another scope", f.c.Execute(direct), "")
	if other.Operation == done.Operation || other.Replay {
		t.Errorf("a key was shared between a client and no client")
	}
	f.must(f.w.SuspendMembership(f.login("root"), "w1", "ann", 1))
	f.code("an answer of a suspended membership", f.c.Execute(adjust(ann, "k3", "1")), Unauthenticated)
	f.must(f.w.ReinstateMembership(f.login("root"), "w1", "ann", 2))
	fresh := f.answer("ann")
	f.must(f.in.SuspendUser(operator, "ann", 1))
	f.code("an answer of a suspended user", f.c.Execute(adjust(fresh, "k4", "1")), Unauthenticated)
}

func approve(answer gate.Answer, key, target string) Call {
	return Call{Answer: answer, Request: "req-a", Command: "statement.approve", Target: target, Key: key, Expected: "r1"}
}

func TestApprovals(t *testing.T) {
	f, ann := basic(t)
	bob := f.answer("bob")
	f.grant("ann", gate.Approve, "c1")
	f.grant("bob", gate.Approve, "c2")
	done := f.code("an independent approver", f.c.Execute(approve(bob, "a1", "t1")), "")
	if done.Operation == "" || done.Revision != "r2" {
		t.Errorf("result %+v", done)
	}
	f.h.targets["w1/t1"].Revision = "r1"
	f.code("the author approving the own work", f.c.Execute(approve(ann, "a2", "t1")), SelfApprovalDenied)
	f.code("bob approving bob's work", f.c.Execute(approve(bob, "a3", "t2")), SelfApprovalDenied)
	f.code("a missing target for an approver", f.c.Execute(approve(bob, "a4", "t9")), PermissionDenied)
	f.member("cat")
	f.code("an approver without the grant", f.c.Execute(approve(f.answer("cat"), "a5", "t1")), PermissionDenied)
	through := approve(bob, "a6", "t1")
	through.Client = "app"
	resolves, _ := f.h.count()
	f.code("an approval through a client", f.c.Execute(through), PermissionDenied)
	if after, _ := f.h.count(); after != resolves {
		t.Errorf("the target was resolved for an approval through a client")
	}
	f.h.targets["w1/t3"] = &Target{CustomerID: "c1", Revision: "r1"}
	f.code("a target without an author", f.c.Execute(approve(bob, "a7", "t3")), PermissionDenied)
	issue := adjust(bob, "a8", "1")
	f.code("an ordinary command is not decided by the approval grant", f.c.Execute(issue), PermissionDenied)
	f.grant("cat", gate.Issue, "c1")
	f.code("an approval command is not decided by an ordinary grant", f.c.Execute(approve(f.answer("cat"), "a9", "t1")), PermissionDenied)
}

func TestIdempotency(t *testing.T) {
	f, ann := basic(t)
	f.grant("bob", gate.Issue, "c1")
	bob := f.answer("bob")
	first := f.code("the first call", f.c.Execute(adjust(ann, "k1", "1.0")), "")
	again := f.code("the same call", f.c.Execute(adjust(f.answer("ann"), "k1", "1.0")), "")
	if !again.Replay || again.Operation != first.Operation || again.Revision != first.Revision || again.Fields["echo"] != "1.0" || !f.last().Replay || f.last().Operation != first.Operation {
		t.Errorf("replay %+v record %+v", again, f.last())
	}
	again.Fields["echo"] = "tampered"
	if third := f.c.Execute(adjust(ann, "k1", "1.0")); third.Fields["echo"] != "1.0" {
		t.Errorf("a returned result aliases the stored operation")
	}
	if _, applies := f.h.count(); applies != 1 {
		t.Errorf("%d applies for one intent", applies)
	}
	changes := map[string]func(*Call){
		"another body":                    func(c *Call) { c.Fields = []Value{{"amount", "2"}} },
		"the same amount in another form": func(c *Call) { c.Fields = []Value{{"amount", "1.00"}} },
		"another target":                  func(c *Call) { c.Target = "t2" },
		"another expected revision":       func(c *Call) { c.Expected = "r9" },
		"another rationale":               func(c *Call) { c.Rationale = "another synthetic rationale" },
		"an added optional field":         func(c *Call) { c.Fields = append(c.Fields, Value{"note", ""}) },
	}
	f.grant("ann", gate.Issue, "c2")
	for name, change := range changes {
		call := adjust(ann, "k1", "1.0")
		change(&call)
		f.code(name, f.c.Execute(call), IdempotencyConflict)
	}
	f.code("the same key from another user", f.c.Execute(adjust(bob, "k1", "1.0")), IdempotencyConflict)
	reordered := adjust(ann, "k5", "3")
	reordered.Expected = "r2"
	reordered.Fields = []Value{{"note", "n"}, {"amount", "3"}}
	one := f.code("a call with two fields", f.c.Execute(reordered), "")
	reordered.Fields = []Value{{"amount", "3"}, {"note", "n"}}
	if two := f.c.Execute(reordered); !two.Replay || two.Operation != one.Operation {
		t.Errorf("the order of fields changes the payload: %+v", two)
	}
	otherCommand := Call{Answer: ann, Request: "r", Command: "statement.approve", Target: "t2", Key: "k1", Expected: "r1"}
	f.grant("ann", gate.Approve, "c2")
	f.code("the same key for another command", f.c.Execute(otherCommand), "")
	f.must(f.w.RemoveMembership(f.login("root"), "w1", "ann", 1))
	f.code("a replay by a caller who is no longer a member", f.c.Execute(adjust(ann, "k1", "1.0")), Unauthenticated)

	f = setup(t)
	f.member("ann")
	issued := f.grant("ann", gate.Issue, "c1")
	ann = f.answer("ann")
	f.code("the call before the grant is revoked", f.c.Execute(adjust(ann, "k1", "1")), "")
	f.must(f.a.RevokeGrant(issued, 1))
	lost := f.code("a replay by a caller who lost the grant", f.c.Execute(adjust(ann, "k1", "1")), PermissionDenied)
	if lost.Operation != "" || lost.Fields != nil || lost.Replay {
		t.Errorf("a denied replay discloses the operation: %+v", lost)
	}

	f, ann = basic(t)
	done := f.c.Execute(adjust(ann, "k1", "1"))
	delete(f.h.targets, "w1/t1")
	f.code("a replay after the target stopped resolving", f.c.Execute(adjust(ann, "k1", "1")), PermissionDenied)
	f.c.mu.Lock()
	for _, stored := range f.c.operations {
		stored.workspace = "w2"
	}
	f.c.mu.Unlock()
	f.h.targets["w1/t1"] = &Target{CustomerID: "c1", Revision: done.Revision, AuthorID: "ann"}
	f.code("an operation stored for another workspace under the same identity", f.c.Execute(adjust(ann, "k1", "1")), IdempotencyConflict)
}

func TestHandlerOutcomes(t *testing.T) {
	named := map[Code]Code{UnsupportedCapability: UnsupportedCapability, InvalidStage: InvalidStage, RevisionConflict: RevisionConflict, PrerequisiteFailed: PrerequisiteFailed,
		DependencyUnavailable: DependencyUnavailable, PermissionDenied: PermissionDenied, InvalidRequest: InvalidRequest}
	for returned, want := range named {
		f, ann := basic(t)
		f.h.apply = func(Context, *Target) (Outcome, error) {
			return Outcome{}, Fail(returned, "amount", "not a field", "12.50")
		}
		got := f.code("a handler returning "+string(returned), f.c.Execute(adjust(ann, "k1", "1")), want)
		wantFields := []string(nil)
		if returned == InvalidRequest {
			wantFields = []string{"amount"}
		}
		if !reflect.DeepEqual(got.Err.Fields, wantFields) || got.Err.Operation != "" {
			t.Errorf("%s: fields %v operation %q", returned, got.Err.Fields, got.Err.Operation)
		}
		if got.Err.Retry != (returned == DependencyUnavailable) {
			t.Errorf("%s: retry %v", returned, got.Err.Retry)
		}
		f.h.apply = nil
		retry := f.code("the same key after a named error", f.c.Execute(adjust(ann, "k1", "2")), "")
		if retry.Replay || retry.Fields["echo"] != "2" {
			t.Errorf("%s: the key was not free: %+v", returned, retry)
		}
	}
	unknown := map[string]func(Context, *Target) (Outcome, error){
		"an error that is not named":            func(Context, *Target) (Outcome, error) { return Outcome{}, errors.New("synthetic secret text") },
		"a named error outside the handler set": func(Context, *Target) (Outcome, error) { return Outcome{}, Fail(Unauthenticated) },
		"a panic":                               func(Context, *Target) (Outcome, error) { panic("synthetic secret text") },
		"an outcome together with an error": func(Context, *Target) (Outcome, error) {
			return Outcome{ResultRevision: "r2", BusinessState: "s"}, Fail(InvalidStage)
		},
		"a completed outcome without revision": func(Context, *Target) (Outcome, error) { return Outcome{BusinessState: "s"}, nil },
		"a completed outcome without state":    func(Context, *Target) (Outcome, error) { return Outcome{ResultRevision: "r2"}, nil },
		"an accepted outcome without a job":    func(Context, *Target) (Outcome, error) { return Outcome{Accepted: true}, nil },
		"a job on a completed outcome": func(Context, *Target) (Outcome, error) {
			return Outcome{ResultRevision: "r2", BusinessState: "s", JobID: "j"}, nil
		},
		"a revision that is text": func(Context, *Target) (Outcome, error) {
			return Outcome{ResultRevision: "free text", BusinessState: "s"}, nil
		},
		"a result value above the limit": func(Context, *Target) (Outcome, error) {
			return Outcome{ResultRevision: "r2", BusinessState: "s", Fields: map[string]string{"x": strings.Repeat("v", 41)}}, nil
		},
		"a result field name that is text": func(Context, *Target) (Outcome, error) {
			return Outcome{ResultRevision: "r2", BusinessState: "s", Fields: map[string]string{"free text": "v"}}, nil
		},
		"result fields above the limit": func(Context, *Target) (Outcome, error) {
			fields := map[string]string{}
			for index := 0; index < 7; index++ {
				fields[fmt.Sprintf("f%d", index)] = "v"
			}
			return Outcome{ResultRevision: "r2", BusinessState: "s", Fields: fields}, nil
		},
	}
	for name, apply := range unknown {
		f, ann := basic(t)
		f.h.apply = apply
		got := f.code(name, f.c.Execute(adjust(ann, "k1", "1")), InternalError)
		if got.Err.Operation == "" || got.Err.Next != NextOperation || got.Err.Retry || strings.Contains(fmt.Sprintf("%+v", got), "secret") {
			t.Errorf("%s: %+v", name, got.Err)
		}
		if record := f.last(); record.Outcome != string(InternalError) || record.Operation != got.Err.Operation {
			t.Errorf("%s: record %+v", name, record)
		}
		f.h.apply = nil
		again := f.code(name+", the same call again", f.c.Execute(adjust(ann, "k1", "1")), InternalError)
		if again.Err.Operation != got.Err.Operation || !again.Replay {
			t.Errorf("%s: the failure is not replayed: %+v", name, again)
		}
		f.code(name+", another body", f.c.Execute(adjust(ann, "k1", "2")), IdempotencyConflict)
		if _, applies := f.h.count(); applies != 1 {
			t.Errorf("%s: %d applies", name, applies)
		}
		f.code(name+", the lock is free for another key", f.c.Execute(adjust(ann, "k2", "1")), "")
	}
	f, ann := basic(t)
	f.h.apply = func(Context, *Target) (Outcome, error) { return Outcome{Accepted: true, JobID: "job-7"}, nil }
	accepted := f.code("an accepted outcome", f.c.Execute(adjust(ann, "k1", "1")), "")
	if accepted.Status != Accepted || accepted.Job != "job-7" || accepted.Operation == "" || accepted.Revision != "" || accepted.State != "" || f.last().Outcome != "accepted" {
		t.Errorf("accepted %+v", accepted)
	}
	if again := f.c.Execute(adjust(ann, "k1", "1")); !again.Replay || again.Job != "job-7" {
		t.Errorf("an accepted operation is not replayed: %+v", again)
	}
	reading := Call{Answer: ann, Request: "r", Command: "statement.read", Target: "t1"}
	for name, apply := range map[string]func(Context, *Target) (Outcome, error){
		"an accepted outcome of a reading command": func(Context, *Target) (Outcome, error) { return Outcome{Accepted: true, JobID: "j"}, nil },
		"a revision from a reading command":        func(Context, *Target) (Outcome, error) { return Outcome{ResultRevision: "r2"}, nil },
		"a panic in a reading command":             func(Context, *Target) (Outcome, error) { panic("synthetic") },
	} {
		f.h.apply = apply
		got := f.code(name, f.c.Execute(reading), InternalError)
		if got.Err.Operation != "" || got.Err.Next != NextNone {
			t.Errorf("%s: %+v", name, got.Err)
		}
	}
	f.h.apply = func(Context, *Target) (Outcome, error) { return Outcome{}, Fail(RevisionConflict) }
	if got := f.code("a reading command whose handler saw the target move", f.c.Execute(reading), RevisionConflict); !got.Err.Retry {
		t.Errorf("a reading conflict is not retryable")
	}
	f.h.apply = func(Context, *Target) (Outcome, error) { return Outcome{}, Fail(RevisionConflict) }
	if got := f.code("a mutating command whose handler saw the target move", f.c.Execute(adjust(ann, "k9", "1")), RevisionConflict); got.Err.Retry {
		t.Errorf("a mutating conflict is retryable")
	}
}

func TestResolveFailures(t *testing.T) {
	f, ann := basic(t)
	f.h.unavailable = true
	f.code("an unavailable target store", f.c.Execute(adjust(ann, "k1", "1")), DependencyUnavailable)
	f.h.unavailable = false
	f.h.panics = true
	got := f.code("a panic while resolving", f.c.Execute(adjust(ann, "k1", "1")), InternalError)
	if got.Err.Operation != "" {
		t.Errorf("a failure before the decision stored an operation: %+v", got.Err)
	}
	f.h.panics = false
	f.h.targets["w1/t1"].Revision = "free text revision"
	f.code("a revision that breaks the identifier rule", f.c.Execute(adjust(ann, "k1", "1")), InternalError)
	f.member("cat")
	f.code("the same broken target for a caller without the permission", f.c.Execute(adjust(f.answer("cat"), "k1", "1")), PermissionDenied)
	f.h.targets["w1/t1"].Revision = "r1"
	done := f.code("the key is free after those failures", f.c.Execute(adjust(ann, "k1", "1")), "")
	if done.Replay {
		t.Errorf("a failed resolution consumed the key")
	}
	if _, applies := f.h.count(); applies != 1 {
		t.Errorf("%d applies", applies)
	}
}

func TestTheRecheckBeforeApply(t *testing.T) {
	f, _ := basic(t)
	session := f.login("ann")
	answer, err := f.b.Access(session, "w1")
	f.must(err)
	*f.hook = func() { f.must(f.b.EndSession(session)) }
	*f.armed = 6
	f.code("the session ends between the decision and the apply", f.c.Execute(adjust(answer, "k1", "1")), Unauthenticated)
	if resolves, applies := f.h.count(); applies != 0 || resolves != 1 || *f.armed != 0 {
		t.Errorf("the session did not end after the decision and before the apply: %d resolves, %d applies, %d reads left", resolves, applies, *f.armed)
	}
	f.grant("bob", gate.Approve, "c1")
	approverSession := f.login("bob")
	approver, err := f.b.Access(approverSession, "w1")
	f.must(err)
	*f.hook = func() { f.must(f.b.EndSession(approverSession)) }
	*f.armed = 6
	f.code("the approver's session ends between the check and the apply", f.c.Execute(approve(approver, "a1", "t1")), Unauthenticated)
	if resolves, applies := f.h.count(); applies != 0 || resolves != 2 || *f.armed != 0 {
		t.Errorf("the approver's session did not end after the check and before the apply: %d resolves, %d applies, %d reads left", resolves, applies, *f.armed)
	}
	f.member("cat")
	ann := f.answer("ann")
	admin := f.login("root")
	*f.hook = func() { f.must(f.w.SuspendMembership(admin, "w1", "cat", 1)) }
	*f.armed = 6
	got := f.code("an unrelated membership changes between the decision and the apply", f.c.Execute(adjust(ann, "k2", "1")), AuthorityUnavailable)
	if !got.Err.Retry {
		t.Errorf("a stale stamp is not retryable")
	}
	if resolves, applies := f.h.count(); applies != 0 || resolves != 3 || *f.armed != 0 {
		t.Errorf("%d resolves, %d applies, %d reads left", resolves, applies, *f.armed)
	}
	f.code("the same call again", f.c.Execute(adjust(ann, "k2", "1")), "")
}

func TestTheJournal(t *testing.T) {
	f, ann := basic(t)
	const secret = "MARKER-secret-value"
	call := adjust(ann, "k1", "1")
	call.Rationale = "MARKER-secret-value rationale"
	call.Fields = append(call.Fields, Value{"note", secret})
	f.code("a call with marked text", f.c.Execute(call), "")
	bad := adjust(ann, "k2", "1")
	bad.Fields = append(bad.Fields, Value{"note", strings.Repeat(secret, 3)})
	f.code("a refused call with marked text", f.c.Execute(bad), InvalidRequest)
	f.h.apply = func(Context, *Target) (Outcome, error) { return Outcome{}, errors.New(secret) }
	third := adjust(ann, "k3", "1")
	third.Expected = "r2"
	f.code("a handler error with marked text", f.c.Execute(third), InternalError)
	f.member("cat")
	f.code("a call the gate refuses for want of a grant", f.c.Execute(adjust(f.answer("cat"), "k4", "1")), PermissionDenied)
	f.code("an approval the gate refuses for the author", f.c.Execute(approve(ann, "k5", "t1")), PermissionDenied)
	through := adjust(ann, "k6", "1")
	through.Client = "unknown-client"
	f.code("a call the gate refuses for the client", f.c.Execute(through), PermissionDenied)
	gone := f.answer("cat")
	f.must(f.w.RemoveMembership(f.login("root"), "w1", "cat", 1))
	f.code("a call of a removed member", f.c.Execute(adjust(gone, "k7", "1")), Unauthenticated)
	reasons := []gate.Reason{gate.ReasonAllowed, gate.ReasonUnavailable, gate.ReasonPrincipal, gate.ReasonStamp, gate.ReasonNoGrant, gate.ReasonClient, gate.ReasonSupportScope,
		gate.ReasonSupportSession, gate.ReasonApprovalRequired, gate.ReasonNotHuman, gate.ReasonAuthor, gate.ReasonSelfApproval, gate.ReasonIdentity, gate.ReasonWorkspace}
	records := f.c.Journal(0, 1000)
	for index, record := range records {
		if record.Sequence != uint64(index)+1 {
			t.Errorf("sequence %d at position %d", record.Sequence, index)
		}
		text := fmt.Sprintf("%+v", record)
		if strings.Contains(text, "MARKER") || strings.Contains(text, "rationale") {
			t.Errorf("a record holds caller text: %s", text)
		}
		for _, reason := range reasons {
			if record.Outcome == string(reason) && reason != gate.ReasonUnavailable && reason != gate.ReasonInvalidRequest {
				t.Errorf("a record holds the gate reason %s: %s", reason, text)
			}
		}
	}
	if page := f.c.Journal(1, 1); len(page) != 1 || page[0].Sequence != 2 {
		t.Errorf("page %+v", page)
	}
	if page := f.c.Journal(uint64(len(records)), 5); len(page) != 0 {
		t.Errorf("a page past the end %+v", page)
	}
	records[0].User = "tampered"
	if f.c.Journal(0, 1)[0].User == "tampered" {
		t.Errorf("the journal aliases what it returns")
	}
	twin, twinAnn := basic(t)
	twin.c.Execute(adjust(twinAnn, "k1", "1"))
	first := f.c.Journal(0, 1)[0]
	if first.Digest == twin.last().Digest {
		t.Errorf("two registries give the same digest for the same payload")
	}
	g, annAgain := basic(t)
	g.c.Execute(adjust(annAgain, "k1", "1"))
	one := g.last().Digest
	g.c.Execute(adjust(annAgain, "k1", "1"))
	if g.last().Digest != one {
		t.Errorf("one registry gives two digests for the same payload")
	}
	plain := sha256.Sum256(payload(adjust(annAgain, "k1", "1"), map[string]string{"amount": "1"}))
	if one == fmt.Sprintf("%x", plain) {
		t.Errorf("the journal holds the plain digest")
	}
}

func TestCapacity(t *testing.T) {
	limits := testLimits
	limits.Journal = 3
	f := setupWith(t, limits)
	f.member("ann")
	f.grant("ann", gate.Issue, "c1")
	f.grant("ann", gate.View, "c1", "quantity")
	ann := f.answer("ann")
	f.code("the first call", f.c.Execute(adjust(ann, "k1", "1")), "")
	f.code("a refused call", f.c.Execute(Call{Answer: ann, Request: "r", Command: "nothing", Target: "t"}), UnsupportedCommand)
	f.code("the last slot", f.c.Execute(Call{Answer: ann, Request: "r", Command: "statement.read", Target: "t1"}), "")
	resolves, applies := f.h.count()
	for name, call := range map[string]Call{
		"a replay":        adjust(ann, "k1", "1"),
		"a mutating call": adjust(ann, "k2", "1"),
		"a reading call":  {Answer: ann, Request: "r", Command: "statement.read", Target: "t1"},
		"a refused call":  {Answer: ann, Request: "r", Command: "nothing", Target: "t"},
	} {
		got := f.code(name+" with a full journal", f.c.Execute(call), DependencyUnavailable)
		if !got.Err.Retry {
			t.Errorf("%s: not retryable", name)
		}
	}
	f.code("a call that is not admitted, with a full journal", f.c.Execute(Call{Request: "r", Command: "statement.read", Target: "t1"}), Unauthenticated)
	if afterResolves, afterApplies := f.h.count(); afterResolves != resolves || afterApplies != applies {
		t.Errorf("a handler ran although the call could not be recorded")
	}
	if len(f.c.Journal(0, 100)) != 3 {
		t.Errorf("%d records", len(f.c.Journal(0, 100)))
	}

	limits = testLimits
	limits.Operations = 2
	f = setupWith(t, limits)
	f.member("ann")
	f.grant("ann", gate.Issue, "c1")
	ann = f.answer("ann")
	first := f.c.Execute(adjust(ann, "k1", "1"))
	second := adjust(ann, "k2", "1")
	second.Expected = "r2"
	f.code("the second operation", f.c.Execute(second), "")
	third := adjust(ann, "k3", "1")
	third.Expected = "r3"
	_, applies = f.h.count()
	f.code("a new key with a full operation store", f.c.Execute(third), DependencyUnavailable)
	third.Expected = "r0"
	f.code("a new key with a full store and a stale revision", f.c.Execute(third), DependencyUnavailable)
	if _, after := f.h.count(); after != applies {
		t.Errorf("the handler ran with a full operation store")
	}
	if replay := f.c.Execute(adjust(ann, "k1", "1")); !replay.Replay || replay.Operation != first.Operation {
		t.Errorf("an existing key does not replay with a full store: %+v", replay)
	}
}

func TestAnUnconstructedRegistry(t *testing.T) {
	var none *Registry
	for _, r := range []*Registry{none, {}} {
		got := r.Execute(Call{Request: "r", Command: "x", Target: "t"})
		if got.Err == nil || got.Err.Code != AuthorityUnavailable || got.Request != "r" {
			t.Errorf("execute %+v", got)
		}
		if err := r.Register(Definition{}); !errors.Is(err, ErrUnavailable) {
			t.Errorf("register %v", err)
		}
		if r.Catalogue() != nil || r.Journal(0, 1) != nil {
			t.Errorf("an unconstructed registry answers")
		}
		if _, err := r.OpenAPI("t", "v"); !errors.Is(err, ErrUnavailable) {
			t.Errorf("openapi %v", err)
		}
	}
	for id, valid := range map[string]bool{"a": true, "A.b_c:d-9": true, strings.Repeat("x", 128): true, strings.Repeat("x", 129): false, "": false, "a b": false, "a/b": false} {
		if validID(id) != valid {
			t.Errorf("identifier %q", id)
		}
	}
}

func TestOneApplyForManyIdenticalCalls(t *testing.T) {
	f, _ := basic(t)
	answers := []gate.Answer{}
	for index := 0; index < 8; index++ {
		answers = append(answers, f.answer("ann"))
	}
	var wg sync.WaitGroup
	results := make(chan Result, len(answers)*10)
	for _, answer := range answers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := 0; index < 10; index++ {
				results <- f.c.Execute(adjust(answer, "k1", "1"))
				f.c.Execute(Call{Answer: answer, Request: "r", Command: "statement.read", Target: "t1"})
			}
		}()
	}
	wg.Wait()
	close(results)
	operation, fresh := "", 0
	for got := range results {
		if got.Err != nil {
			t.Fatalf("an identical call ended with %s", got.Err.Code)
		}
		if operation == "" {
			operation = got.Operation
		}
		if got.Operation != operation {
			t.Errorf("two operations for one intent")
		}
		if !got.Replay {
			fresh++
		}
	}
	if _, applies := f.h.count(); applies != 1+80 || fresh != 1 {
		t.Errorf("%d applies including 80 reads, %d fresh results", applies, fresh)
	}
	if records := f.c.Journal(0, 1000000); len(records) != 160 {
		t.Errorf("%d records", len(records))
	}
}

func TestAMissingTargetIsDeniedAlsoUnderAGrantForEveryCustomer(t *testing.T) {
	f := setup(t)
	f.member("ann")
	f.must(f.a.AddGrant(gate.Grant{ID: "wide", SubjectID: "ann", Validity: always, Terms: gate.Terms{Action: gate.Issue, Kind: "statement",
		Scope: gate.Scope{WorkspaceID: "w1", Customer: gate.All(), LegalEntity: gate.All(), Contract: gate.All()}, Period: gate.AnyPeriod()}}))
	ann := f.answer("ann")
	f.code("an existing target under the wide grant", f.c.Execute(adjust(ann, "k1", "1")), "")
	missing := adjust(ann, "k2", "1")
	missing.Target, missing.Expected = "t9", "r1"
	f.code("a missing target under the wide grant", f.c.Execute(missing), PermissionDenied)
	if _, applies := f.h.count(); applies != 1 {
		t.Errorf("the handler ran for a missing target: %d applies", applies)
	}
	f.h.targets["w1/t9"] = &Target{CustomerID: "c9", Revision: "r1"}
	f.code("the same call once the target exists", f.c.Execute(missing), "")
}

func TestAReservedJournalSlotCannotBeTakenByAnotherCall(t *testing.T) {
	limits := testLimits
	limits.Journal = 2
	f := setupWith(t, limits)
	f.member("ann")
	f.grant("ann", gate.Issue, "c1")
	f.grant("ann", gate.View, "c1", "quantity")
	ann, other := f.answer("ann"), f.answer("ann")
	f.code("the first record", f.c.Execute(Call{Answer: ann, Request: "r", Command: "nothing", Target: "t"}), UnsupportedCommand)
	var during Result
	f.h.apply = func(context Context, found *Target) (Outcome, error) {
		if context.Expected == "" {
			return Outcome{}, nil
		}
		f.h.mu.Unlock()
		during = f.c.Execute(Call{Answer: other, Request: "r", Command: "statement.read", Target: "t1"})
		f.h.mu.Lock()
		return Outcome{ResultRevision: "r2", BusinessState: "recorded"}, nil
	}
	f.code("a mutating call that holds the last slot", f.c.Execute(adjust(ann, "k1", "1")), "")
	if during.Err == nil || during.Err.Code != DependencyUnavailable {
		t.Errorf("a reading call during the apply took the reserved slot: %+v", during)
	}
	records := f.c.Journal(0, 100)
	if len(records) != 2 || records[1].Outcome != "completed" {
		t.Errorf("the action is not the last record of a full journal: %+v", records)
	}
}

type slow struct {
	applies atomic.Int64
	second  chan struct{}
	spawn   func()
}

func (s *slow) Resolve(string, string) (Target, Resolution) {
	return Target{CustomerID: "c1", Revision: "r1"}, Found
}

func (s *slow) Apply(Context) (Outcome, error) {
	if s.applies.Add(1) > 1 {
		close(s.second)
		return Outcome{ResultRevision: "r2", BusinessState: "recorded"}, nil
	}
	s.spawn()
	select {
	case <-s.second:
	case <-time.After(300 * time.Millisecond):
	}
	return Outcome{ResultRevision: "r2", BusinessState: "recorded"}, nil
}

func TestMutatingCallsDoNotInterleave(t *testing.T) {
	f := setup(t)
	f.member("ann")
	f.grant("ann", gate.Deliver, "c1")
	handler := &slow{second: make(chan struct{})}
	f.must(f.c.Register(Definition{ID: "statement.send", Permission: Permission{Action: gate.Deliver, Kind: "statement"}, Mutating: true, Handler: handler}))
	ann, again := f.answer("ann"), f.answer("ann")
	call := func(answer gate.Answer) Call {
		return Call{Answer: answer, Request: "r", Command: "statement.send", Target: "t1", Key: "k1", Expected: "r1"}
	}
	concurrent := make(chan Result, 1)
	handler.spawn = func() { go func() { concurrent <- f.c.Execute(call(again)) }() }
	first := f.code("the first call", f.c.Execute(call(ann)), "")
	second := f.code("the call sent while the first was applying", <-concurrent, "")
	if handler.applies.Load() != 1 || !second.Replay || second.Operation != first.Operation {
		t.Errorf("%d applies; the second result %+v", handler.applies.Load(), second)
	}
}

func TestResultsDoNotAliasTheStoredOperation(t *testing.T) {
	f, ann := basic(t)
	first := f.code("the call", f.c.Execute(adjust(ann, "k1", "7")), "")
	first.Fields["echo"] = "tampered"
	if replay := f.c.Execute(adjust(ann, "k1", "7")); replay.Fields["echo"] != "7" {
		t.Errorf("the first result aliases the stored operation: %+v", replay)
	}
}

func TestTheJournalDigestIsKeyed(t *testing.T) {
	one, annOne := basic(t)
	two, annTwo := basic(t)
	one.code("in the first registry", one.c.Execute(adjust(annOne, "k1", "1")), "")
	two.code("in the second registry", two.c.Execute(adjust(annTwo, "k1", "1")), "")
	if one.last().Digest == two.last().Digest || len(one.last().Digest) != 64 {
		t.Errorf("the same payload has the same journal digest in two registries: %s", one.last().Digest)
	}
}
