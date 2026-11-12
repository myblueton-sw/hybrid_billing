package auth

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

var (
	propertyCustomers = []string{"c1", "c2", "c3"}
	propertyFields    = []string{"quantity", "service", "unit_cost", "margin"}
	propertySubjects  = []string{"alice", "bob", "carol", "dave", "erin"}
	propertyKinds     = []string{"usage_statement", "allocation_pool"}
	propertyActions   = []Action{View, ViewCost, Download, Issue, Deliver, RequestApproval, ChangeCustomerPolicy, ChangeSellingPolicy}
)

func classOf(field string) FieldClass {
	if field == "unit_cost" || field == "margin" {
		return Cost
	}
	return General
}

func randomSelector(r *rand.Rand) Selector {
	if r.Intn(3) == 0 {
		return All()
	}
	return One(propertyCustomers[r.Intn(len(propertyCustomers))])
}

func randomFields(r *rand.Rand, action Action) []string {
	var pool []string
	for _, field := range propertyFields {
		switch {
		case action == View && classOf(field) == General, action == ViewCost && classOf(field) == Cost, action == Download:
			pool = append(pool, field)
		}
	}
	r.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	if len(pool) == 0 {
		return nil
	}
	return pool[:1+r.Intn(len(pool))]
}

func randomTerms(r *rand.Rand, workspace string) Terms {
	action := propertyActions[r.Intn(len(propertyActions))]
	return Terms{Action: action, Kind: propertyKinds[r.Intn(len(propertyKinds))], Scope: Scope{WorkspaceID: workspace, Customer: randomSelector(r), LegalEntity: All(), Contract: All()}, Fields: randomFields(r, action), Period: AnyPeriod()}
}

func randomRequest(r *rand.Rand) Request {
	action := propertyActions[r.Intn(len(propertyActions))]
	return Request{Action: action, Kind: propertyKinds[r.Intn(len(propertyKinds))], Resource: Resource{WorkspaceID: "w1", CustomerID: propertyCustomers[r.Intn(len(propertyCustomers))]}, Fields: randomFields(r, action)}
}

type world struct {
	fixture
	grants      []Grant
	delegations []Delegation
}

func randomWorld(t *testing.T, r *rand.Rand) world {
	w := world{fixture: setup(t)}
	grants, delegations := 6+r.Intn(14), 2+r.Intn(8)
	for index := 0; index < grants; index++ {
		workspace := "w1"
		if r.Intn(5) == 0 {
			workspace = "w2"
		}
		grant := Grant{ID: fmt.Sprintf("g%d", index), SubjectID: propertySubjects[r.Intn(len(propertySubjects))], Terms: randomTerms(r, workspace), Validity: always}
		if err := w.AddGrant(grant); err != nil {
			t.Fatalf("grant %+v: %v", grant, err)
		}
		w.grants = append(w.grants, grant)
	}
	for index := 0; index < delegations; index++ {
		delegation := Delegation{ID: fmt.Sprintf("d%d", index), DelegatorID: propertySubjects[r.Intn(len(propertySubjects))], DelegateID: propertySubjects[r.Intn(len(propertySubjects))], AllowRedelegation: r.Intn(2) == 0, Validity: always}
		if len(w.delegations) > 0 && r.Intn(2) == 0 {
			parent := w.delegations[r.Intn(len(w.delegations))]
			delegation.ParentID, delegation.DelegatorID, delegation.Terms = parent.ID, parent.DelegateID, parent.Terms
			if len(parent.Terms.Fields) > 1 && r.Intn(2) == 0 {
				delegation.Terms.Fields = parent.Terms.Fields[:1]
			}
			if parent.Terms.Scope.Customer.all && r.Intn(2) == 0 {
				delegation.Terms.Scope.Customer = One(propertyCustomers[r.Intn(len(propertyCustomers))])
			}
		} else {
			var own []Grant
			for _, grant := range w.grants {
				if grant.SubjectID == delegation.DelegatorID {
					own = append(own, grant)
				}
			}
			if len(own) == 0 {
				continue
			}
			delegation.Terms = own[r.Intn(len(own))].Terms
		}
		if w.AddDelegation(delegation) == nil {
			w.delegations = append(w.delegations, delegation)
		}
	}
	return w
}

func oracleCovers(t Terms, action Action, request Request, field string) bool {
	if t.Action != action || t.Kind != request.Kind || t.Scope.WorkspaceID != request.Resource.WorkspaceID {
		return false
	}
	if !t.Scope.Customer.all && t.Scope.Customer.id != request.Resource.CustomerID {
		return false
	}
	if field == "" {
		return true
	}
	for _, candidate := range t.Fields {
		if candidate == field {
			return true
		}
	}
	return false
}

func (w world) oracleDirect(subject string, action Action, request Request, field string) bool {
	for _, grant := range w.grants {
		if grant.SubjectID == subject && oracleCovers(grant.Terms, action, request, field) {
			return true
		}
	}
	return false
}

func classView(field string) Action {
	if classOf(field) == Cost {
		return ViewCost
	}
	return View
}

func (w world) oracleHolds(subject string, request Request, field string) bool {
	if !w.oracleDirect(subject, request.Action, request, field) {
		return false
	}
	return request.Action != Download || w.oracleDirect(subject, classView(field), request, field)
}

func (w world) oracleRoot(d Delegation, action Action, request Request, field string) (string, bool) {
	for {
		if !oracleCovers(d.Terms, action, request, field) {
			return "", false
		}
		if d.ParentID == "" {
			return d.DelegatorID, true
		}
		for _, parent := range w.delegations {
			if parent.ID == d.ParentID {
				d = parent
				break
			}
		}
	}
}

func (w world) oracleHome(subject string) bool {
	for _, grant := range w.grants {
		if grant.SubjectID == subject && grant.Terms.Scope.WorkspaceID == "w1" {
			return true
		}
	}
	return false
}

func (w world) oracleAction(subject string, action Action, request Request, field string) bool {
	if w.oracleDirect(subject, action, request, field) {
		return true
	}
	if !w.oracleHome(subject) {
		return false
	}
	for _, d := range w.delegations {
		if d.DelegateID != subject {
			continue
		}
		root, live := w.oracleRoot(d, action, request, field)
		if !live || !w.oracleDirect(root, action, request, field) {
			continue
		}
		if action != Download || w.oracleDirect(root, classView(field), request, field) {
			return true
		}
	}
	return false
}

func (w world) oracle(subject string, request Request) bool {
	fields := request.Fields
	if len(fields) == 0 {
		fields = []string{""}
	}
	for _, field := range fields {
		if !w.oracleAction(subject, request.Action, request, field) {
			return false
		}
		if request.Action == Download && !w.oracleAction(subject, classView(field), request, field) {
			return false
		}
	}
	return true
}

func TestDecisionsAgreeWithAnIndependentOracle(t *testing.T) {
	r := rand.New(rand.NewSource(169))
	allowed, delegated := 0, 0
	for round := 0; round < 600; round++ {
		w := randomWorld(t, r)
		for probe := 0; probe < 40; probe++ {
			subject := propertySubjects[r.Intn(len(propertySubjects))]
			request := randomRequest(r)
			got := w.Decide(who(subject), request)
			if want := w.oracle(subject, request); got.Allowed != want {
				t.Fatalf("round %d: %s %+v allowed %v; oracle %v\ngrants %+v\ndelegations %+v", round, subject, request, got.Allowed, want, w.grants, w.delegations)
			}
			if got.Allowed {
				allowed++
				if len(got.Delegations) > 0 {
					delegated++
				}
				for _, field := range request.Fields {
					single := request
					single.Fields = []string{field}
					if !w.Decide(who(subject), single).Allowed {
						t.Fatalf("round %d: a subset of an allowed field set is denied: %s %+v", round, subject, single)
					}
				}
			}
		}
	}
	if allowed < 500 || delegated < 50 {
		t.Fatalf("the generator does not exercise the model: %d allowed, %d through a delegation", allowed, delegated)
	}
	t.Logf("%d allowed, %d through a delegation", allowed, delegated)
}

func TestRemovingAuthorityNeverAddsAnAllow(t *testing.T) {
	r := rand.New(rand.NewSource(1690))
	lost := 0
	for round := 0; round < 300; round++ {
		w := randomWorld(t, r)
		type probe struct {
			subject string
			request Request
			before  bool
		}
		var probes []probe
		for index := 0; index < 40; index++ {
			p := probe{subject: propertySubjects[r.Intn(len(propertySubjects))], request: randomRequest(r)}
			p.before = w.Decide(who(p.subject), p.request).Allowed
			probes = append(probes, p)
		}
		switch r.Intn(3) {
		case 0:
			if err := w.RevokeGrant(w.grants[r.Intn(len(w.grants))].ID, 1); err != nil {
				t.Fatal(err)
			}
		case 1:
			if len(w.delegations) == 0 {
				continue
			}
			if err := w.RevokeDelegation(w.delegations[r.Intn(len(w.delegations))].ID, 1); err != nil {
				t.Fatal(err)
			}
		case 2:
			id := propertySubjects[r.Intn(len(propertySubjects))]
			if _, err := w.PutSubject(Subject{ID: id, Kind: Human, State: Suspended, Epoch: 1}, 1); err != nil {
				t.Fatal(err)
			}
		}
		for _, p := range probes {
			after := w.Decide(who(p.subject), p.request).Allowed
			if after && !p.before {
				t.Fatalf("round %d: removing authority allowed %s %+v", round, p.subject, p.request)
			}
			if p.before && !after {
				lost++
			}
		}
	}
	if lost < 50 {
		t.Fatalf("removals rarely change a decision: %d", lost)
	}
	t.Logf("%d allows lost to a removal", lost)
}

func TestOtherWorkspaceWritesLeaveDecisionsUnchanged(t *testing.T) {
	r := rand.New(rand.NewSource(16900))
	for round := 0; round < 300; round++ {
		w := randomWorld(t, r)
		var requests []Request
		var subjects []string
		var before []bool
		for index := 0; index < 30; index++ {
			subject, request := propertySubjects[r.Intn(len(propertySubjects))], randomRequest(r)
			subjects, requests = append(subjects, subject), append(requests, request)
			before = append(before, w.Decide(who(subject), request).Allowed)
		}
		for index := 0; index < 6; index++ {
			grant := Grant{ID: fmt.Sprintf("foreign%d", index), SubjectID: propertySubjects[r.Intn(len(propertySubjects))], Terms: randomTerms(r, "w2"), Validity: always}
			if err := w.AddGrant(grant); err != nil {
				t.Fatal(err)
			}
		}
		for index := range requests {
			if got := w.Decide(who(subjects[index]), requests[index]).Allowed; got != before[index] {
				t.Fatalf("round %d: a grant in another workspace changed %s %+v from %v to %v", round, subjects[index], requests[index], before[index], got)
			}
		}
	}
}

func TestClientDecisionsAreWithinBothSides(t *testing.T) {
	r := rand.New(rand.NewSource(169000))
	allowed := 0
	for round := 0; round < 300; round++ {
		w := randomWorld(t, r)
		var clientGrants []Grant
		clients := 6 + r.Intn(12)
		for index := 0; index < clients; index++ {
			grant := Grant{ID: fmt.Sprintf("client%d", index), ClientID: "app", Terms: randomTerms(r, "w1"), Validity: always}
			if err := w.AddGrant(grant); err != nil {
				t.Fatal(err)
			}
			grant.SubjectID = "app"
			clientGrants = append(clientGrants, grant)
		}
		clientWorld := world{grants: clientGrants}
		for probe := 0; probe < 40; probe++ {
			subject, request := propertySubjects[r.Intn(len(propertySubjects))], randomRequest(r)
			with := who(subject)
			with.ClientID = "app"
			fields := request.Fields
			if len(fields) == 0 {
				fields = []string{""}
			}
			clientHolds := true
			for _, field := range fields {
				clientHolds = clientHolds && clientWorld.oracleHolds("app", request, field)
			}
			want := w.oracle(subject, request) && clientHolds
			if got := w.Decide(with, request).Allowed; got != want {
				t.Fatalf("round %d: %s through the client %+v allowed %v; want %v", round, subject, request, got, want)
			}
			if want {
				allowed++
			}
		}
	}
	if allowed < 100 {
		t.Fatalf("the generator rarely allows a client request: %d", allowed)
	}
	t.Logf("%d client requests allowed", allowed)
}

func FuzzDecideNeverAllowsWithoutAGrant(f *testing.F) {
	f.Add("alice", "view", "usage_statement", "w1", "c1", "quantity")
	f.Add("", "", "", "", "", "")
	f.Add("alice", "approve", "usage_statement", "w1", "c1", "")
	f.Fuzz(func(t *testing.T, subject, action, kind, workspace, customer, field string) {
		now := start
		r, err := NewRegistry("synthetic-installation", Catalog{Kinds: []string{"usage_statement"}, Fields: map[string]FieldClass{"quantity": General}}, func() time.Time { return now })
		if err != nil {
			t.Fatal(err)
		}
		if _, err := r.PutSubject(Subject{ID: "alice", Kind: Human, State: Active, Epoch: 1}, 0); err != nil {
			t.Fatal(err)
		}
		request := Request{Action: Action(action), Kind: kind, Resource: Resource{WorkspaceID: workspace, CustomerID: customer}}
		if field != "" {
			request.Fields = []string{field}
		}
		if got := r.Decide(who(subject), request); got.Allowed || got.Stamp.issued {
			t.Fatalf("allowed without any grant: %+v", got)
		}
	})
}
