package auth

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

type fixture struct {
	*Registry
	now *time.Time
	t   testing.TB
}

var (
	start  = time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	always = Validity{NotBefore: start.Add(-time.Hour), NotAfter: start.Add(1000 * time.Hour)}
)

func month(t testing.TB, year, number int) Month {
	t.Helper()
	m, err := NewMonth(year, number)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func setup(t testing.TB) fixture {
	t.Helper()
	now := start
	r, err := NewRegistry("synthetic-installation", Catalog{
		Kinds:  []string{"usage_statement", "allocation_pool", "selling_amount"},
		Fields: map[string]FieldClass{"quantity": General, "service": General, "unit_cost": Cost, "margin": Cost},
	}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	f := fixture{r, &now, t}
	for _, id := range []string{"alice", "bob", "carol", "dave", "erin"} {
		f.subject(id, Human)
	}
	f.subject("robot", Service)
	if _, err := r.PutClient(Client{ID: "app", State: Active}, 0); err != nil {
		t.Fatal(err)
	}
	return f
}

func (f fixture) subject(id string, kind SubjectKind) {
	f.t.Helper()
	if _, err := f.PutSubject(Subject{ID: id, Kind: kind, State: Active, Epoch: 1}, 0); err != nil {
		f.t.Fatal(err)
	}
}

func scope(customer Selector) Scope {
	return Scope{WorkspaceID: "w1", Customer: customer, LegalEntity: All(), Contract: All()}
}

func terms(action Action, customer Selector, fields ...string) Terms {
	return Terms{Action: action, Kind: "usage_statement", Scope: scope(customer), Fields: fields, Period: AnyPeriod()}
}

func (f fixture) grant(id, subject string, t Terms) {
	f.t.Helper()
	if err := f.AddGrant(Grant{ID: id, SubjectID: subject, Terms: t, Validity: always}); err != nil {
		f.t.Fatalf("grant %s: %v", id, err)
	}
}

func (f fixture) clientGrant(id string, t Terms) {
	f.t.Helper()
	if err := f.AddGrant(Grant{ID: id, ClientID: "app", Terms: t, Validity: always}); err != nil {
		f.t.Fatalf("client grant %s: %v", id, err)
	}
}

func who(subject string) Principal {
	return Principal{Installation: "synthetic-installation", SubjectID: subject, Epoch: 1}
}

func ask(action Action, customer string, fields ...string) Request {
	return Request{Action: action, Kind: "usage_statement", Resource: Resource{WorkspaceID: "w1", CustomerID: customer}, Fields: fields}
}

func (f fixture) expect(name string, principal Principal, request Request, allowed bool, reason Reason) Decision {
	f.t.Helper()
	got := f.Decide(principal, request)
	if got.Allowed != allowed || got.Reason != reason {
		f.t.Errorf("%s: allowed %v reason %s; want %v %s", name, got.Allowed, got.Reason, allowed, reason)
	}
	if got.Allowed != (got.Code == Allowed) || got.Allowed != got.Stamp.issued {
		f.t.Errorf("%s: inconsistent decision %+v", name, got)
	}
	return got
}

func TestEachActionNeedsItsOwnGrantAndKind(t *testing.T) {
	f := setup(t)
	f.grant("g-view", "alice", terms(View, One("c1"), "quantity"))
	f.grant("g-cost", "alice", terms(ViewCost, One("c1"), "unit_cost"))
	f.grant("g-issue", "alice", terms(Issue, One("c1")))
	f.expect("view by its grant", who("alice"), ask(View, "c1", "quantity"), true, ReasonAllowed)
	f.expect("cost by its grant", who("alice"), ask(ViewCost, "c1", "unit_cost"), true, ReasonAllowed)
	f.expect("issue by its grant", who("alice"), ask(Issue, "c1"), true, ReasonAllowed)
	for _, action := range []Action{ChangeCustomerPolicy, ChangeSellingPolicy, RequestApproval, Deliver} {
		f.expect("no grant for "+string(action), who("alice"), ask(action, "c1"), false, ReasonNoGrant)
	}
	f.expect("download needs its own grant", who("alice"), ask(Download, "c1", "quantity"), false, ReasonNoGrant)
	f.expect("a general field is not a cost field", who("alice"), ask(ViewCost, "c1", "quantity"), false, ReasonInvalidRequest)
	f.expect("a cost field is not viewable by view", who("alice"), ask(View, "c1", "unit_cost"), false, ReasonInvalidRequest)
	f.expect("approve is never decided here", who("alice"), ask(Approve, "c1"), false, ReasonApprovalRequired)
	f.expect("session approval is never decided here", who("alice"), ask(ApproveSupportSession, "c1"), false, ReasonApprovalRequired)
	other := ask(Issue, "c1")
	other.Kind = "allocation_pool"
	f.expect("a grant for one kind does not cover another", who("alice"), other, false, ReasonNoGrant)
	for name, request := range map[string]Request{
		"unknown action":             ask("refund", "c1"),
		"unknown kind":               {Action: Issue, Kind: "invoice", Resource: Resource{WorkspaceID: "w1", CustomerID: "c1"}},
		"field action without field": ask(View, "c1"),
		"fields on another action":   ask(Issue, "c1", "quantity"),
		"unknown field":              ask(View, "c1", "price"),
		"repeated field":             ask(View, "c1", "quantity", "quantity"),
		"no workspace":               {Action: Issue, Kind: "usage_statement", Resource: Resource{CustomerID: "c1"}},
	} {
		f.expect(name, who("alice"), request, false, ReasonInvalidRequest)
	}
}

func TestScopeFieldsPeriodAndValidity(t *testing.T) {
	f := setup(t)
	f.grant("g1", "alice", terms(View, One("c1"), "quantity"))
	f.grant("g2", "alice", terms(View, One("c1"), "service"))
	f.grant("g3", "alice", terms(View, One("c2"), "service"))
	f.grant("g-all", "bob", terms(View, All(), "quantity"))
	contract := terms(Issue, One("c1"))
	contract.Scope.Contract, contract.Scope.LegalEntity = One("k1"), One("le1")
	f.grant("g-contract", "carol", contract)
	bounded := terms(View, One("c1"), "quantity")
	bounded.Period = Months(month(t, 2026, 3), month(t, 2026, 6))
	f.grant("g-period", "dave", bounded)
	f.expect("fields covered by two grants of one scope", who("alice"), ask(View, "c1", "quantity", "service"), true, ReasonAllowed)
	f.expect("a field covered only for another customer", who("alice"), ask(View, "c2", "quantity"), false, ReasonNoGrant)
	f.expect("another customer", who("alice"), ask(View, "c3", "service"), false, ReasonNoGrant)
	f.expect("one customer against a resource without customer", who("alice"), ask(View, "", "quantity"), false, ReasonNoGrant)
	f.expect("all customers matches any", who("bob"), ask(View, "c9", "quantity"), true, ReasonAllowed)
	f.expect("all customers matches a resource without customer", who("bob"), ask(View, "", "quantity"), true, ReasonAllowed)
	elsewhere := ask(View, "c1", "quantity")
	elsewhere.Resource.WorkspaceID = "w2"
	f.expect("another workspace with the same customer identifier", who("bob"), elsewhere, false, ReasonNoGrant)
	exact := ask(Issue, "c1")
	exact.Resource.ContractID, exact.Resource.LegalEntityID = "k1", "le1"
	f.expect("contract and legal entity match", who("carol"), exact, true, ReasonAllowed)
	for name, change := range map[string]func(*Request){
		"another contract":     func(r *Request) { r.Resource.ContractID = "k2" },
		"no contract":          func(r *Request) { r.Resource.ContractID = "" },
		"another legal entity": func(r *Request) { r.Resource.LegalEntityID = "le2" },
	} {
		request := exact
		change(&request)
		f.expect(name, who("carol"), request, false, ReasonNoGrant)
	}
	for name, c := range map[string]struct {
		at      Month
		has     bool
		allowed bool
	}{
		"first month of the period": {month(t, 2026, 3), true, true},
		"last month of the period":  {month(t, 2026, 5), true, true},
		"the exclusive end":         {month(t, 2026, 6), true, false},
		"before the period":         {month(t, 2026, 2), true, false},
		"a resource without month":  {0, false, false},
	} {
		request := ask(View, "c1", "quantity")
		request.Resource.Month, request.Resource.HasMonth = c.at, c.has
		reason := ReasonAllowed
		if !c.allowed {
			reason = ReasonNoGrant
		}
		f.expect(name, who("dave"), request, c.allowed, reason)
	}
	early := terms(View, One("c1"), "service")
	early.Period = Months(Month(0), month(t, 2026, 6))
	f.grant("g-early", "dave", early)
	f.expect("a period starting at month zero against a resource without month", who("dave"), ask(View, "c1", "service"), false, ReasonNoGrant)
	window := Validity{NotBefore: start.Add(time.Hour), NotAfter: start.Add(2 * time.Hour)}
	if err := f.AddGrant(Grant{ID: "g-window", SubjectID: "erin", Terms: terms(Issue, One("c1")), Validity: window}); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name    string
		at      time.Time
		allowed bool
	}{
		{"not yet valid", start, false},
		{"exactly at not-before", start.Add(time.Hour), true},
		{"just before not-after", start.Add(2*time.Hour - time.Nanosecond), true},
		{"exactly at not-after", start.Add(2 * time.Hour), false},
	} {
		*f.now = c.at
		reason := ReasonAllowed
		if !c.allowed {
			reason = ReasonNoGrant
		}
		f.expect(c.name, who("erin"), ask(Issue, "c1"), c.allowed, reason)
	}
}

func TestRegistrationRejectsMalformedRecords(t *testing.T) {
	f := setup(t)
	good := Grant{ID: "g", SubjectID: "alice", Terms: terms(View, One("c1"), "quantity"), Validity: always}
	cases := map[string]func(*Grant){
		"zero customer selector":      func(g *Grant) { g.Terms.Scope.Customer = Selector{} },
		"zero contract selector":      func(g *Grant) { g.Terms.Scope.Contract = Selector{} },
		"zero period":                 func(g *Grant) { g.Terms.Period = Period{} },
		"empty period":                func(g *Grant) { g.Terms.Period = Months(5, 5) },
		"no expiry":                   func(g *Grant) { g.Validity.NotAfter = time.Time{} },
		"inverted validity":           func(g *Grant) { g.Validity = Validity{NotBefore: start, NotAfter: start} },
		"cost field on a view grant":  func(g *Grant) { g.Terms.Fields = []string{"unit_cost"} },
		"general field on cost grant": func(g *Grant) { g.Terms.Action = ViewCost },
		"unknown field":               func(g *Grant) { g.Terms.Fields = []string{"*"} },
		"repeated field":              func(g *Grant) { g.Terms.Fields = []string{"quantity", "quantity"} },
		"view without fields":         func(g *Grant) { g.Terms.Fields = nil },
		"fields on issue":             func(g *Grant) { g.Terms.Action = Issue },
		"unknown action":              func(g *Grant) { g.Terms.Action = "refund" },
		"unknown kind":                func(g *Grant) { g.Terms.Kind = "invoice" },
		"no workspace":                func(g *Grant) { g.Terms.Scope.WorkspaceID = "" },
		"subject and client together": func(g *Grant) { g.ClientID = "app" },
		"neither subject nor client":  func(g *Grant) { g.SubjectID = "" },
		"unknown subject":             func(g *Grant) { g.SubjectID = "nobody" },
		"approve for a service":       func(g *Grant) { g.SubjectID, g.Terms = "robot", terms(Approve, One("c1")) },
		"approve for a client":        func(g *Grant) { g.SubjectID, g.ClientID, g.Terms = "", "app", terms(Approve, One("c1")) },
		"session approval for client": func(g *Grant) { g.SubjectID, g.ClientID, g.Terms = "", "app", terms(ApproveSupportSession, One("c1")) },
	}
	for name, change := range cases {
		grant := good
		grant.Terms.Fields = append([]string(nil), good.Terms.Fields...)
		change(&grant)
		if err := f.AddGrant(grant); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: error %v", name, err)
		}
	}
	if err := f.AddGrant(good); err != nil {
		t.Fatal(err)
	}
	if err := f.AddGrant(good); !errors.Is(err, ErrConflict) {
		t.Errorf("repeated identifier error = %v", err)
	}
	if err := f.RevokeGrant("g", 2); !errors.Is(err, ErrConflict) {
		t.Errorf("stale revocation error = %v", err)
	}
	if err := f.RevokeGrant("g", 1); err != nil {
		t.Fatal(err)
	}
	if err := f.AddGrant(good); !errors.Is(err, ErrConflict) {
		t.Errorf("identifier reuse after revocation error = %v", err)
	}
	if err := f.RevokeGrant("missing", 1); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown grant error = %v", err)
	}
	if _, err := NewMonth(2026, 13); !errors.Is(err, ErrInvalid) {
		t.Errorf("month 13 error = %v", err)
	}
	for name, catalog := range map[string]Catalog{
		"no kinds":       {},
		"repeated kind":  {Kinds: []string{"a", "a"}},
		"unknown class":  {Kinds: []string{"a"}, Fields: map[string]FieldClass{"x": "secret"}},
		"wildcard field": {Kinds: []string{"a"}, Fields: map[string]FieldClass{"*": General}},
	} {
		if _, err := NewRegistry("i", catalog, time.Now); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: error %v", name, err)
		}
	}
	if _, err := NewRegistry("i", Catalog{Kinds: []string{"a"}}, nil); !errors.Is(err, ErrInvalid) {
		t.Errorf("missing clock error = %v", err)
	}
}

func TestPrincipalsMustBeCurrent(t *testing.T) {
	f := setup(t)
	f.grant("g", "alice", terms(Issue, One("c1")))
	f.grant("g-robot", "robot", terms(Issue, One("c1")))
	f.expect("a service subject may hold an ordinary grant", who("robot"), ask(Issue, "c1"), true, ReasonAllowed)
	for name, principal := range map[string]Principal{
		"another installation": {Installation: "other", SubjectID: "alice", Epoch: 1},
		"stale epoch":          {Installation: "synthetic-installation", SubjectID: "alice", Epoch: 2},
		"unknown subject":      {Installation: "synthetic-installation", SubjectID: "nobody", Epoch: 1},
		"client and session":   {Installation: "synthetic-installation", SubjectID: "alice", Epoch: 1, ClientID: "app", SessionID: "s"},
	} {
		f.expect(name, principal, ask(Issue, "c1"), false, ReasonPrincipal)
	}
	if _, err := f.PutSubject(Subject{ID: "alice", Kind: Human, State: Suspended, Epoch: 1}, 1); err != nil {
		t.Fatal(err)
	}
	f.expect("suspended subject", who("alice"), ask(Issue, "c1"), false, ReasonPrincipal)
	for name, subject := range map[string]Subject{
		"reactivation without a new epoch": {ID: "alice", Kind: Human, State: Active, Epoch: 1},
		"changing the kind":                {ID: "alice", Kind: Service, State: Active, Epoch: 2},
		"lower epoch":                      {ID: "alice", Kind: Human, State: Suspended, Epoch: 0},
	} {
		if _, err := f.PutSubject(subject, 2); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: error %v", name, err)
		}
	}
	if _, err := f.PutSubject(Subject{ID: "alice", Kind: Human, State: Active, Epoch: 2}, 2); err != nil {
		t.Fatal(err)
	}
	f.expect("the old epoch after reactivation", who("alice"), ask(Issue, "c1"), false, ReasonPrincipal)
	renewed := who("alice")
	renewed.Epoch = 2
	f.expect("the new epoch", renewed, ask(Issue, "c1"), true, ReasonAllowed)
	f.SetAvailable(false)
	got := f.Decide(renewed, ask(Issue, "c1"))
	if got.Allowed || got.Code != AuthorityUnavailable {
		t.Errorf("unavailable registry: %+v", got)
	}
	if got := f.CheckApproval(renewed, "bob", ask(Approve, "c1")); got.Code != AuthorityUnavailable {
		t.Errorf("unavailable approval check: %+v", got)
	}
	if got := f.Recheck(Stamp{}, renewed, ask(Issue, "c1")); got.Code != AuthorityUnavailable {
		t.Errorf("unavailable recheck: %+v", got)
	}
	if got := f.RecheckApproval(Stamp{}, renewed, "bob", ask(Approve, "c1")); got.Code != AuthorityUnavailable {
		t.Errorf("unavailable approval recheck: %+v", got)
	}
	if err := f.AddSupportSession(SupportSession{}, renewed); !errors.Is(err, ErrUnavailable) {
		t.Errorf("unavailable session registration error = %v", err)
	}
}

func TestStoredTermsDoNotAliasTheCaller(t *testing.T) {
	f := setup(t)
	fields := []string{"quantity"}
	f.grant("g", "alice", terms(View, One("c1"), fields...))
	f.grant("home-bob", "bob", terms(RequestApproval, One("c9")))
	if err := f.delegate("d", "alice", "bob", "", false, terms(View, One("c1"), fields...), always); err != nil {
		t.Fatal(err)
	}
	f.grant("support-approval", "carol", terms(ApproveSupportSession, One("c1")))
	if err := f.AddSupportSession(SupportSession{ID: "s", ActorID: "dave", WorkspaceID: "w1", CustomerID: "c1", Reason: "ticket", Kind: "usage_statement", Fields: fields, Validity: always}, who("carol")); err != nil {
		t.Fatal(err)
	}
	fields[0] = "service"
	in := who("dave")
	in.SessionID = "s"
	for name, principal := range map[string]Principal{"the grant": who("alice"), "the delegation": who("bob"), "the session": in} {
		f.expect(name+" keeps its field", principal, ask(View, "c1", "quantity"), true, ReasonAllowed)
		reason := ReasonNoGrant
		if principal.SessionID != "" {
			reason = ReasonSupportScope
		}
		f.expect(name+" does not gain the caller's later field", principal, ask(View, "c1", "service"), false, reason)
	}
}

func TestDownloadNeedsTheMatchingViewGrant(t *testing.T) {
	f := setup(t)
	f.grant("d", "alice", terms(Download, One("c1"), "quantity", "unit_cost"))
	f.expect("download without any view grant", who("alice"), ask(Download, "c1", "quantity"), false, ReasonNoGrant)
	f.grant("v", "alice", terms(View, One("c1"), "quantity"))
	f.expect("general field with view", who("alice"), ask(Download, "c1", "quantity"), true, ReasonAllowed)
	f.expect("cost field without cost view", who("alice"), ask(Download, "c1", "unit_cost"), false, ReasonNoGrant)
	f.expect("mixed request with one conjunct missing", who("alice"), ask(Download, "c1", "quantity", "unit_cost"), false, ReasonNoGrant)
	f.grant("c", "alice", terms(ViewCost, One("c1"), "unit_cost"))
	f.expect("mixed request with both conjuncts", who("alice"), ask(Download, "c1", "quantity", "unit_cost"), true, ReasonAllowed)
	f.grant("v-only", "bob", terms(View, One("c1"), "quantity"))
	f.expect("view without download", who("bob"), ask(Download, "c1", "quantity"), false, ReasonNoGrant)
}

func TestClientIntersection(t *testing.T) {
	f := setup(t)
	f.grant("u", "alice", terms(View, One("c1"), "quantity", "service"))
	f.clientGrant("c", terms(View, One("c1"), "service"))
	with := who("alice")
	with.ClientID = "app"
	f.expect("both hold the field", with, ask(View, "c1", "service"), true, ReasonAllowed)
	f.expect("only the user holds the field", with, ask(View, "c1", "quantity"), false, ReasonClient)
	f.grant("u2", "bob", terms(View, One("c1"), "quantity"))
	asBob := who("bob")
	asBob.ClientID = "app"
	f.expect("only the client holds the field", asBob, ask(View, "c1", "service"), false, ReasonNoGrant)
	unknown := who("alice")
	unknown.ClientID = "ghost"
	f.expect("unknown client", unknown, ask(View, "c1", "service"), false, ReasonClient)
	f.grant("ud", "alice", terms(Download, One("c1"), "service"))
	f.clientGrant("cd", terms(Download, One("c1"), "service"))
	f.expect("download through a client with both conjuncts on both sides", with, ask(Download, "c1", "service"), true, ReasonAllowed)
	if _, err := f.PutClient(Client{ID: "second", State: Active}, 0); err != nil {
		t.Fatal(err)
	}
	second := who("alice")
	second.ClientID = "second"
	f.expect("a client without grants while another client holds them", second, ask(View, "c1", "service"), false, ReasonClient)
	if _, err := f.PutClient(Client{ID: "app", State: Suspended}, 1); err != nil {
		t.Fatal(err)
	}
	f.expect("suspended client", with, ask(Download, "c1", "service"), false, ReasonClient)
	if _, err := f.PutClient(Client{ID: "app", State: Active}, 2); err != nil {
		t.Fatal(err)
	}
	f.expect("the client is active again", with, ask(Download, "c1", "service"), true, ReasonAllowed)
	if err := f.RevokeGrant("c", 1); err != nil {
		t.Fatal(err)
	}
	f.expect("download through a client that lost its view grant", with, ask(Download, "c1", "service"), false, ReasonClient)
	if got := f.CheckApproval(with, "bob", ask(Approve, "c1")); got.Allowed || got.Reason != ReasonPrincipal {
		t.Errorf("approval through a client: %+v", got)
	}
}

func (f fixture) delegate(id, from, to, parent string, redelegate bool, t Terms, v Validity) error {
	return f.AddDelegation(Delegation{ID: id, DelegatorID: from, DelegateID: to, ParentID: parent, AllowRedelegation: redelegate, Terms: t, Validity: v})
}

func TestDelegationChains(t *testing.T) {
	f := setup(t)
	f.grant("root", "alice", terms(View, One("c1"), "quantity", "service"))
	for _, id := range []string{"bob", "carol", "dave", "erin"} {
		f.grant("home-"+id, id, terms(RequestApproval, One("c9")))
	}
	narrow := terms(View, One("c1"), "quantity")
	if err := f.delegate("d1", "alice", "bob", "", true, narrow, always); err != nil {
		t.Fatal(err)
	}
	got := f.expect("a delegate within the terms", who("bob"), ask(View, "c1", "quantity"), true, ReasonAllowed)
	if len(got.Delegations) != 1 || got.Delegations[0] != "d1" || got.ActorID != "bob" {
		t.Errorf("the decision names the chain and the delegate as actor: %+v", got)
	}
	f.expect("a field outside the delegation", who("bob"), ask(View, "c1", "service"), false, ReasonNoGrant)
	if err := f.delegate("d2", "bob", "carol", "d1", false, narrow, always); err != nil {
		t.Fatal(err)
	}
	f.expect("a re-delegate", who("carol"), ask(View, "c1", "quantity"), true, ReasonAllowed)
	rejected := []struct {
		name string
		err  error
		run  func() error
	}{
		{"re-delegation that the parent does not allow", ErrNotBacked, func() error { return f.delegate("x1", "carol", "dave", "d2", false, narrow, always) }},
		{"a parentless delegation by a delegate without direct grant", ErrNotBacked, func() error { return f.delegate("x2", "bob", "dave", "", false, narrow, always) }},
		{"widened field set", ErrNotBacked, func() error {
			return f.delegate("x3", "bob", "dave", "d1", false, terms(View, One("c1"), "quantity", "service"), always)
		}},
		{"widened scope", ErrNotBacked, func() error {
			return f.delegate("x4", "bob", "dave", "d1", false, terms(View, All(), "quantity"), always)
		}},
		{"another customer", ErrNotBacked, func() error {
			return f.delegate("x5", "bob", "dave", "d1", false, terms(View, One("c2"), "quantity"), always)
		}},
		{"another action than the parent", ErrNotBacked, func() error {
			return f.delegate("x6", "bob", "dave", "d1", false, terms(RequestApproval, One("c1")), always)
		}},
		{"validity beyond the parent", ErrNotBacked, func() error {
			return f.delegate("x7", "bob", "dave", "d1", false, narrow, Validity{NotBefore: always.NotBefore, NotAfter: always.NotAfter.Add(time.Hour)})
		}},
		{"a parent whose delegate is someone else", ErrNotBacked, func() error { return f.delegate("x8", "carol", "dave", "d1", false, narrow, always) }},
		{"parentless beyond the direct grant", ErrNotBacked, func() error {
			return f.delegate("x9", "alice", "dave", "", false, terms(View, All(), "quantity"), always)
		}},
		{"delegator equals delegate", ErrInvalid, func() error { return f.delegate("x10", "alice", "alice", "", false, narrow, always) }},
		{"a subject twice in the chain", ErrInvalid, func() error { return f.delegate("x11", "bob", "alice", "d1", false, narrow, always) }},
		{"delegating approve", ErrInvalid, func() error { return f.delegate("x12", "alice", "dave", "", false, terms(Approve, One("c1")), always) }},
		{"a service delegate", ErrInvalid, func() error { return f.delegate("x13", "alice", "robot", "", false, narrow, always) }},
		{"an existing identifier", ErrConflict, func() error { return f.delegate("d1", "alice", "dave", "", false, narrow, always) }},
	}
	for _, c := range rejected {
		if err := c.run(); !errors.Is(err, c.err) {
			t.Errorf("%s: error %v; want %v", c.name, err, c.err)
		}
	}
	f.grant("home-robot", "robot", terms(RequestApproval, One("c9")))
	if err := f.delegate("x13b", "alice", "robot", "", false, narrow, always); !errors.Is(err, ErrInvalid) {
		t.Errorf("a service delegate that holds a grant in the workspace error = %v", err)
	}
	f.subject("frank", Human)
	if err := f.delegate("x14", "alice", "frank", "", false, narrow, always); !errors.Is(err, ErrInvalid) {
		t.Errorf("a delegate without any grant in the workspace error = %v", err)
	}
	bounded := terms(View, One("c1"), "quantity")
	bounded.Period = Months(month(t, 2026, 1), month(t, 2026, 4))
	if err := f.delegate("x15", "bob", "dave", "d1", false, bounded, always); err != nil {
		t.Errorf("a bounded period under an unbounded parent: %v", err)
	}
	if err := f.delegate("x15b", "alice", "erin", "", true, bounded, always); err != nil {
		t.Fatal(err)
	}
	if err := f.delegate("x15c", "erin", "dave", "x15b", false, terms(View, One("c1"), "quantity"), always); !errors.Is(err, ErrNotBacked) {
		t.Errorf("an unbounded period under a bounded parent error = %v", err)
	}
	f.grant("root-bounded", "erin", bounded)
	if err := f.delegate("x16", "erin", "dave", "", false, terms(View, One("c1"), "quantity"), always); !errors.Is(err, ErrNotBacked) {
		t.Errorf("an unbounded period over a bounded grant error = %v", err)
	}
	if err := f.RevokeGrant("home-carol", 1); err != nil {
		t.Fatal(err)
	}
	f.expect("a delegate that lost its last grant in the workspace", who("carol"), ask(View, "c1", "quantity"), false, ReasonNoGrant)
	f.grant("home-carol-2", "carol", terms(RequestApproval, One("c9")))
	f.expect("the delegate holds a grant in the workspace again", who("carol"), ask(View, "c1", "quantity"), true, ReasonAllowed)
	if err := f.RevokeDelegation("d1", 1); err != nil {
		t.Fatal(err)
	}
	f.expect("the delegate after revocation", who("bob"), ask(View, "c1", "quantity"), false, ReasonNoGrant)
	f.expect("a descendant after its parent is revoked", who("carol"), ask(View, "c1", "quantity"), false, ReasonNoGrant)
}

func TestDelegationEndsWithItsBacking(t *testing.T) {
	short := Validity{NotBefore: start.Add(-time.Hour), NotAfter: start.Add(time.Hour)}
	build := func(t *testing.T) fixture {
		f := setup(t)
		f.grant("root", "alice", terms(View, One("c1"), "quantity"))
		f.grant("home-bob", "bob", terms(RequestApproval, One("c9")))
		f.grant("home-carol", "carol", terms(RequestApproval, One("c9")))
		if err := f.delegate("d1", "alice", "bob", "", true, terms(View, One("c1"), "quantity"), short); err != nil {
			t.Fatal(err)
		}
		if err := f.delegate("d2", "bob", "carol", "d1", false, terms(View, One("c1"), "quantity"), short); err != nil {
			t.Fatal(err)
		}
		f.expect("before the change", who("carol"), ask(View, "c1", "quantity"), true, ReasonAllowed)
		return f
	}
	t.Run("the root delegator loses the direct grant", func(t *testing.T) {
		f := build(t)
		if err := f.RevokeGrant("root", 1); err != nil {
			t.Fatal(err)
		}
		f.expect("the delegate", who("bob"), ask(View, "c1", "quantity"), false, ReasonNoGrant)
		f.expect("the re-delegate", who("carol"), ask(View, "c1", "quantity"), false, ReasonNoGrant)
		f.grant("root-again", "alice", terms(View, One("c1"), "quantity"))
		f.expect("the delegation is live again when the root is granted again", who("carol"), ask(View, "c1", "quantity"), true, ReasonAllowed)
	})
	t.Run("the delegator in the middle is suspended", func(t *testing.T) {
		f := build(t)
		if _, err := f.PutSubject(Subject{ID: "bob", Kind: Human, State: Suspended, Epoch: 1}, 1); err != nil {
			t.Fatal(err)
		}
		f.expect("the re-delegate", who("carol"), ask(View, "c1", "quantity"), false, ReasonNoGrant)
	})
	t.Run("the root delegator is suspended", func(t *testing.T) {
		f := build(t)
		if _, err := f.PutSubject(Subject{ID: "alice", Kind: Human, State: Suspended, Epoch: 1}, 1); err != nil {
			t.Fatal(err)
		}
		f.expect("the delegate", who("bob"), ask(View, "c1", "quantity"), false, ReasonNoGrant)
	})
	t.Run("the delegations expire by the clock", func(t *testing.T) {
		f := build(t)
		*f.now = start.Add(time.Hour)
		f.expect("the delegate", who("bob"), ask(View, "c1", "quantity"), false, ReasonNoGrant)
		f.expect("the root still holds its grant", who("alice"), ask(View, "c1", "quantity"), true, ReasonAllowed)
	})
	t.Run("download through a delegation needs the root's view grant", func(t *testing.T) {
		f := setup(t)
		f.grant("root-download", "alice", terms(Download, One("c1"), "quantity"))
		f.grant("home-bob", "bob", terms(View, One("c1"), "quantity"))
		if err := f.delegate("d", "alice", "bob", "", false, terms(Download, One("c1"), "quantity"), always); err != nil {
			t.Fatal(err)
		}
		f.expect("the root holds download but not view", who("bob"), ask(Download, "c1", "quantity"), false, ReasonNoGrant)
		f.grant("root-view", "alice", terms(View, One("c1"), "quantity"))
		f.expect("the root holds both conjuncts", who("bob"), ask(Download, "c1", "quantity"), true, ReasonAllowed)
	})
	t.Run("fields from a direct grant and from a delegation combine", func(t *testing.T) {
		f := setup(t)
		f.grant("root", "alice", terms(View, One("c1"), "quantity"))
		f.grant("own", "bob", terms(View, One("c1"), "service"))
		if err := f.delegate("d", "alice", "bob", "", false, terms(View, One("c1"), "quantity"), always); err != nil {
			t.Fatal(err)
		}
		f.expect("one field direct and one delegated", who("bob"), ask(View, "c1", "quantity", "service"), true, ReasonAllowed)
	})
	t.Run("a chain is at most four delegations", func(t *testing.T) {
		f := setup(t)
		f.subject("frank", Human)
		f.grant("root", "alice", terms(Issue, One("c1")))
		chain := []string{"alice", "bob", "carol", "dave", "erin", "frank"}
		for _, id := range chain[1:] {
			f.grant("home-"+id, id, terms(RequestApproval, One("c9")))
		}
		parent := ""
		for index := 1; index < len(chain); index++ {
			id := "d" + chain[index]
			err := f.delegate(id, chain[index-1], chain[index], parent, true, terms(Issue, One("c1")), always)
			if index <= 4 && err != nil {
				t.Fatalf("delegation %d: %v", index, err)
			}
			if index == 5 && !errors.Is(err, ErrInvalid) {
				t.Errorf("fifth delegation error = %v", err)
			}
			parent = id
		}
		f.expect("the fourth delegate", who("erin"), ask(Issue, "c1"), true, ReasonAllowed)
	})
}

func (f fixture) session(id, actor, approver string, fields ...string) error {
	return f.AddSupportSession(SupportSession{ID: id, ActorID: actor, WorkspaceID: "w1", CustomerID: "c1", Reason: "ticket-1", Kind: "usage_statement", Fields: fields, Validity: Validity{NotBefore: start.Add(-time.Minute), NotAfter: start.Add(time.Hour)}}, who(approver))
}

func TestSupportSessions(t *testing.T) {
	f := setup(t)
	f.grant("support-approval", "carol", terms(ApproveSupportSession, One("c1")))
	f.grant("own-approve", "alice", terms(Approve, One("c1")))
	f.grant("own-issue", "alice", terms(Issue, One("c1")))
	f.grant("own-cost", "alice", terms(ViewCost, One("c1"), "unit_cost"))
	if err := f.session("s1", "alice", "carol", "quantity"); err != nil {
		t.Fatal(err)
	}
	in := who("alice")
	in.SessionID = "s1"
	got := f.expect("view inside the approved terms", in, ask(View, "c1", "quantity"), true, ReasonAllowed)
	if got.ActorID != "alice" || got.SessionID != "s1" {
		t.Errorf("the decision names the real actor and the session: %+v", got)
	}
	refused := map[string]Request{
		"a field outside the approved set": ask(View, "c1", "service"),
		"another customer":                 ask(View, "c2", "quantity"),
		"a resource without customer":      ask(View, "", "quantity"),
		"issue although held directly":     ask(Issue, "c1"),
		"cost view although held directly": ask(ViewCost, "c1", "unit_cost"),
		"download":                         ask(Download, "c1", "quantity"),
		"deliver":                          ask(Deliver, "c1"),
		"customer policy":                  ask(ChangeCustomerPolicy, "c1"),
		"selling policy":                   ask(ChangeSellingPolicy, "c1"),
		"request approval":                 ask(RequestApproval, "c1"),
	}
	for name, request := range refused {
		got := f.expect(name, in, request, false, ReasonSupportScope)
		if got.ActorID != "alice" || got.SessionID != "s1" {
			t.Errorf("%s: a refusal names the real actor and the session: %+v", name, got)
		}
	}
	otherKind := ask(View, "c1", "quantity")
	otherKind.Kind = "allocation_pool"
	f.expect("another kind", in, otherKind, false, ReasonSupportScope)
	otherWorkspace := ask(View, "c1", "quantity")
	otherWorkspace.Resource.WorkspaceID = "w2"
	f.expect("the same customer identifier in another workspace", in, otherWorkspace, false, ReasonSupportScope)
	f.expect("approve inside a session", in, ask(Approve, "c1"), false, ReasonApprovalRequired)
	if got := f.CheckApproval(in, "bob", ask(Approve, "c1")); got.Allowed || got.Reason != ReasonPrincipal {
		t.Errorf("approval check inside a session: %+v", got)
	}
	if got := f.CheckApproval(who("alice"), "bob", ask(Approve, "c1")); !got.Allowed {
		t.Errorf("the same person approves outside the session under their own grant: %+v", got)
	}
	f.expect("outside the session the own grants apply", who("alice"), ask(Issue, "c1"), true, ReasonAllowed)
	f.expect("the session grants nothing outside it", who("alice"), ask(View, "c1", "quantity"), false, ReasonNoGrant)
	asBob := who("bob")
	asBob.SessionID = "s1"
	f.expect("another subject using the session", asBob, ask(View, "c1", "quantity"), false, ReasonSupportSession)
	withClient := in
	withClient.ClientID = "app"
	f.expect("a session combined with a client", withClient, ask(View, "c1", "quantity"), false, ReasonPrincipal)
	if err := f.RevokeGrant("support-approval", 1); err != nil {
		t.Fatal(err)
	}
	f.expect("the session stays after its approver's grant is revoked", in, ask(View, "c1", "quantity"), true, ReasonAllowed)
	*f.now = start.Add(time.Hour)
	f.expect("after expiry", in, ask(View, "c1", "quantity"), false, ReasonSupportSession)
	*f.now = start
	if err := f.EndSupportSession("s1", 1); err != nil {
		t.Fatal(err)
	}
	f.expect("after ending", in, ask(View, "c1", "quantity"), false, ReasonSupportSession)
	missing := who("alice")
	missing.SessionID = "ghost"
	f.expect("an unknown session", missing, ask(View, "c1", "quantity"), false, ReasonSupportSession)
}

func TestSupportSessionRegistration(t *testing.T) {
	f := setup(t)
	f.grant("support-approval", "carol", terms(ApproveSupportSession, One("c1")))
	f.grant("support-approval-alice", "alice", terms(ApproveSupportSession, One("c1")))
	f.grant("alice-view", "alice", terms(View, One("c1"), "quantity"))
	f.grant("home-dave", "dave", terms(RequestApproval, One("c9")))
	f.grant("support-approval-dave", "dave", terms(ApproveSupportSession, One("c1")))
	if err := f.delegate("d", "alice", "dave", "", false, terms(View, One("c1"), "quantity"), always); err != nil {
		t.Fatal(err)
	}
	narrow := terms(ApproveSupportSession, One("c1"))
	narrow.Scope.Contract = One("k1")
	f.grant("support-approval-contract", "erin", narrow)
	f.grant("support-approval-other", "bob", terms(ApproveSupportSession, One("c2")))
	denied := map[string]func() error{
		"the actor approves the own session":      func() error { return f.session("n1", "alice", "alice", "quantity") },
		"an approver who is the actor's delegate": func() error { return f.session("n2", "alice", "dave", "quantity") },
		"an approver without the grant":           func() error { return f.session("n3", "alice", "bob", "quantity") },
		"an approver whose grant is narrower":     func() error { return f.session("n4", "alice", "erin", "quantity") },
		"a service approver":                      func() error { return f.session("n5", "alice", "robot", "quantity") },
		"an approver with a stale epoch": func() error {
			p := who("carol")
			p.Epoch = 9
			return f.AddSupportSession(SupportSession{ID: "n6", ActorID: "alice", WorkspaceID: "w1", CustomerID: "c1", Reason: "r", Kind: "usage_statement", Fields: []string{"quantity"}, Validity: always}, p)
		},
		"an approver acting through a client": func() error {
			p := who("carol")
			p.ClientID = "app"
			return f.AddSupportSession(SupportSession{ID: "n7", ActorID: "alice", WorkspaceID: "w1", CustomerID: "c1", Reason: "r", Kind: "usage_statement", Fields: []string{"quantity"}, Validity: always}, p)
		},
	}
	for name, run := range denied {
		if err := run(); !errors.Is(err, ErrDenied) {
			t.Errorf("%s: error %v", name, err)
		}
	}
	base := SupportSession{ID: "n", ActorID: "alice", WorkspaceID: "w1", CustomerID: "c1", Reason: "ticket", Kind: "usage_statement", Fields: []string{"quantity"}, Validity: always}
	for name, change := range map[string]func(*SupportSession){
		"a cost field":       func(s *SupportSession) { s.Fields = []string{"unit_cost"} },
		"no field":           func(s *SupportSession) { s.Fields = nil },
		"no customer":        func(s *SupportSession) { s.CustomerID = "" },
		"an unknown kind":    func(s *SupportSession) { s.Kind = "invoice" },
		"no reason":          func(s *SupportSession) { s.Reason = "" },
		"a service actor":    func(s *SupportSession) { s.ActorID = "robot" },
		"no validity":        func(s *SupportSession) { s.Validity = Validity{} },
		"an unknown actor":   func(s *SupportSession) { s.ActorID = "nobody" },
		"an invalid session": func(s *SupportSession) { s.ID = "bad id" },
	} {
		session := base
		change(&session)
		if err := f.AddSupportSession(session, who("carol")); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: error %v", name, err)
		}
	}
	if err := f.AddSupportSession(base, who("carol")); err != nil {
		t.Fatal(err)
	}
	if err := f.AddSupportSession(base, who("carol")); !errors.Is(err, ErrConflict) {
		t.Errorf("an existing session identifier error = %v", err)
	}
	if err := f.EndSupportSession("n", 5); !errors.Is(err, ErrConflict) {
		t.Errorf("stale end error = %v", err)
	}
	if err := f.EndSupportSession("ghost", 1); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown session end error = %v", err)
	}
}

func TestApprovalNeedsAnIndependentHumanWithADirectGrant(t *testing.T) {
	f := setup(t)
	f.grant("alice-approve", "alice", terms(Approve, One("c1")))
	f.grant("bob-approve", "bob", terms(Approve, One("c1")))
	f.grant("carol-other", "carol", terms(Approve, One("c2")))
	cases := []struct {
		name     string
		approver Principal
		author   string
		request  Request
		allowed  bool
		code     Code
		reason   Reason
	}{
		{"an independent approver", who("bob"), "alice", ask(Approve, "c1"), true, Allowed, ReasonAllowed},
		{"the author approving the own work", who("alice"), "alice", ask(Approve, "c1"), false, SelfApprovalDenied, ReasonSelfApproval},
		{"an approver without the grant", who("dave"), "alice", ask(Approve, "c1"), false, PermissionDenied, ReasonNoGrant},
		{"the author without the grant is not told about self approval", who("dave"), "dave", ask(Approve, "c1"), false, PermissionDenied, ReasonNoGrant},
		{"a grant for another customer", who("carol"), "alice", ask(Approve, "c1"), false, PermissionDenied, ReasonNoGrant},
		{"an unknown author", who("bob"), "nobody", ask(Approve, "c1"), false, PermissionDenied, ReasonAuthor},
		{"an empty author", who("bob"), "", ask(Approve, "c1"), false, PermissionDenied, ReasonAuthor},
		{"a service approver", who("robot"), "alice", ask(Approve, "c1"), false, PermissionDenied, ReasonNotHuman},
		{"another action", who("bob"), "alice", ask(Issue, "c1"), false, PermissionDenied, ReasonInvalidRequest},
		{"a stale epoch", Principal{Installation: "synthetic-installation", SubjectID: "bob", Epoch: 7}, "alice", ask(Approve, "c1"), false, PermissionDenied, ReasonPrincipal},
	}
	for _, c := range cases {
		got := f.CheckApproval(c.approver, c.author, c.request)
		if got.Allowed != c.allowed || got.Code != c.code || got.Reason != c.reason {
			t.Errorf("%s: %+v; want %v %s %s", c.name, got, c.allowed, c.code, c.reason)
		}
	}
	otherKind := ask(Approve, "c1")
	otherKind.Kind = "selling_amount"
	if got := f.CheckApproval(who("bob"), "alice", otherKind); got.Allowed {
		t.Errorf("approval of another kind: %+v", got)
	}
	approved := f.CheckApproval(who("bob"), "alice", ask(Approve, "c1"))
	if got := f.RecheckApproval(approved.Stamp, who("bob"), "alice", ask(Approve, "c1")); !got.Allowed {
		t.Errorf("recheck on unchanged state: %+v", got)
	}
	if got := f.RecheckApproval(approved.Stamp, who("bob"), "carol", ask(Approve, "c1")); got.Allowed {
		t.Errorf("an approval stamp reused for another author: %+v", got)
	}
	f.grant("bob-issue", "bob", terms(Issue, One("c1")))
	current := f.CheckApproval(who("bob"), "alice", ask(Approve, "c1"))
	if got := f.Recheck(current.Stamp, who("bob"), ask(Issue, "c1")); got.Allowed || got.Reason != ReasonStamp {
		t.Errorf("an approval stamp presented as an ordinary stamp: %+v", got)
	}
	ordinary := f.Decide(who("bob"), ask(Issue, "c1"))
	if got := f.RecheckApproval(ordinary.Stamp, who("bob"), "alice", ask(Approve, "c1")); !ordinary.Allowed || got.Allowed || got.Reason != ReasonStamp {
		t.Errorf("an ordinary stamp presented as an approval stamp: %+v", got)
	}
	if _, err := f.PutSubject(Subject{ID: "bob", Kind: Human, State: Suspended, Epoch: 1}, 1); err != nil {
		t.Fatal(err)
	}
	if got := f.RecheckApproval(approved.Stamp, who("bob"), "alice", ask(Approve, "c1")); got.Allowed {
		t.Errorf("recheck after the approver is suspended: %+v", got)
	}
}

func TestStampsFailAfterAnyChange(t *testing.T) {
	build := func(t *testing.T) (fixture, Stamp) {
		f := setup(t)
		f.grant("g", "alice", terms(View, One("c1"), "quantity"))
		f.grant("other", "bob", terms(Issue, One("c9")))
		got := f.expect("first decision", who("alice"), ask(View, "c1", "quantity"), true, ReasonAllowed)
		if again := f.Recheck(got.Stamp, who("alice"), ask(View, "c1", "quantity")); !again.Allowed || again.Stamp != got.Stamp {
			t.Fatalf("recheck on unchanged state: %+v", again)
		}
		return f, got.Stamp
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	changes := map[string]func(fixture){
		"the grant is revoked": func(f fixture) { must(f.RevokeGrant("g", 1)) },
		"the subject is suspended": func(f fixture) {
			_, err := f.PutSubject(Subject{ID: "alice", Kind: Human, State: Suspended, Epoch: 1}, 1)
			must(err)
		},
		"an unrelated grant is added":       func(f fixture) { f.grant("more", "carol", terms(Issue, One("c2"))) },
		"an unrelated grant is revoked":     func(f fixture) { must(f.RevokeGrant("other", 1)) },
		"availability is lost and regained": func(f fixture) { f.SetAvailable(false); f.SetAvailable(true) },
		"the clock passes the expiry":       func(f fixture) { *f.now = always.NotAfter },
	}
	for name, change := range changes {
		f, stamp := build(t)
		change(f)
		got := f.Recheck(stamp, who("alice"), ask(View, "c1", "quantity"))
		if got.Allowed {
			t.Errorf("%s: recheck still allowed", name)
		}
		if strings.HasPrefix(name, "an unrelated") && got.Reason != ReasonStamp {
			t.Errorf("%s: reason %s", name, got.Reason)
		}
	}
	f, stamp := build(t)
	for name, request := range map[string]Request{
		"another customer": ask(View, "c2", "quantity"),
		"another action":   ask(Issue, "c1"),
	} {
		if got := f.Recheck(stamp, who("alice"), request); got.Allowed {
			t.Errorf("stamp reused for %s", name)
		}
	}
	f.grant("g-wide", "alice", terms(View, All(), "quantity", "service"))
	fresh := f.Decide(who("alice"), ask(View, "c1", "quantity"))
	for name, request := range map[string]Request{
		"another field of the same generation":     ask(View, "c1", "service"),
		"a wider field set of the same generation": ask(View, "c1", "quantity", "service"),
		"another customer of the same generation":  ask(View, "c2", "quantity"),
	} {
		if !f.Decide(who("alice"), request).Allowed {
			t.Fatalf("%s is expected to be allowed on its own", name)
		}
		if got := f.Recheck(fresh.Stamp, who("alice"), request); got.Allowed {
			t.Errorf("a stamp reused for %s", name)
		}
	}
	if got := f.Recheck(fresh.Stamp, who("alice"), ask(View, "c1", "quantity")); !got.Allowed {
		t.Errorf("the stamp is accepted for its own request: %+v", got)
	}
	if got := f.Recheck(Stamp{}, who("alice"), ask(View, "c1", "service")); got.Allowed {
		t.Errorf("a zero stamp accepted")
	}
	_, first := build(t)
	twin, _ := build(t)
	if got := twin.Recheck(first, who("alice"), ask(View, "c1", "quantity")); got.Allowed {
		t.Errorf("a stamp accepted by another registry instance with the same history")
	}
}

func TestRevocationIsNeverFollowedByAnAllow(t *testing.T) {
	f := setup(t)
	f.grant("g", "alice", terms(Issue, One("c1")))
	var wg sync.WaitGroup
	revoked := make(chan struct{})
	late := make(chan int, 8)
	for reader := 0; reader < 8; reader++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for waiting := true; waiting; {
				select {
				case <-revoked:
					waiting = false
				default:
					f.Decide(who("alice"), ask(Issue, "c1"))
				}
			}
			after := 0
			for index := 0; index < 500; index++ {
				if f.Decide(who("alice"), ask(Issue, "c1")).Allowed {
					after++
				}
			}
			late <- after
		}()
	}
	if err := f.RevokeGrant("g", 1); err != nil {
		t.Fatal(err)
	}
	close(revoked)
	wg.Wait()
	close(late)
	for after := range late {
		if after != 0 {
			t.Errorf("%d allows observed after the revocation returned", after)
		}
	}
}
