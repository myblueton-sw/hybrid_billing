package disclosure

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"
)

type failing struct{}

func (failing) Read([]byte) (int, error) { return 0, errors.New("no entropy") }

func TestRegistryForms(t *testing.T) {
	clock := &testClock{now: start}
	one := Limits{1, 1, 1, 1, 1, 1, 1, 1, 1}
	must(NewRegistry(Config{Keys: keys, Random: &counter{}}, one, clock.read))
	must(NewRegistry(Config{Keys: keys, Random: &counter{}}, Limits{100000, 10000, 10000, 100000, 1000000, 1000000, 1000000, 100000, 10000}, clock.read))
	for _, c := range []Config{{Random: &counter{}}, {Keys: keys}} {
		if _, err := NewRegistry(c, one, clock.read); err != ErrInvalid {
			t.Fatal(err)
		}
	}
	if _, err := NewRegistry(Config{Keys: keys, Random: &counter{}}, one, nil); err != ErrInvalid {
		t.Fatal(err)
	}
	maxes := []int{100000, 10000, 10000, 100000, 1000000, 1000000, 1000000, 100000, 10000}
	for i := range maxes {
		for _, v := range []int{0, maxes[i] + 1} {
			l := []int{1, 1, 1, 1, 1, 1, 1, 1, 1}
			l[i] = v
			if _, err := NewRegistry(Config{Keys: keys, Random: &counter{}}, Limits{l[0], l[1], l[2], l[3], l[4], l[5], l[6], l[7], l[8]}, clock.read); err != ErrInvalid {
				t.Fatalf("%d %d", i, v)
			}
		}
	}
	err := refuse(ReasonLinkExpired)
	if !errors.Is(err, ErrRejected) || err.Error() != ErrRejected.Error()+": link_expired" {
		t.Fatal(err)
	}
	if !validID(strings.Repeat("a", 128)) || validID(strings.Repeat("a", 129)) || validID("") {
		t.Fatal("id")
	}
	for _, id := range []string{"A", "Z", "a", "z", "0", "9", ".", "_", ":", "-"} {
		if !validID(id) {
			t.Fatal(id)
		}
	}
	for _, id := range []string{"@", "[", "`", "{", "/", ";", " a"} {
		if validID(id) {
			t.Fatal(id)
		}
	}
	if !validValue(strings.Repeat("a", 4096)) || validValue(strings.Repeat("a", 4097)) || validValue("\xff") {
		t.Fatal("value")
	}
}

func TestCatalogueGates(t *testing.T) {
	f := newFixture(t)
	expect(t, "form name", second(f.r.Classify("w1", Field{Name: "", Class: ClassPublic}, human("s"))), "invalid")
	expect(t, "form class", second(f.r.Classify("w1", Field{Name: "x", Class: "private"}, human("s"))), "invalid")
	expect(t, "form ws", second(f.r.Classify("", Field{Name: "x", Class: ClassPublic}, human("s"))), "invalid")
	expect(t, "form actor", second(f.r.Classify("w1", Field{Name: "x", Class: ClassPublic}, Actor{ID: "s", Kind: "robot"})), "invalid")
	if again := must(f.r.Classify("w1", Field{Name: "customer", Class: ClassCustomer}, human("steward"))); again.Author != "steward" {
		t.Fatal("repeat")
	}
	expect(t, "reclassify", second(f.r.Classify("w1", Field{Name: "customer", Class: ClassPublic}, human("steward"))), "conflict")
	expect(t, "other author", second(f.r.Classify("w1", Field{Name: "customer", Class: ClassCustomer}, human("x"))), "conflict")
	expect(t, "assistant", second(f.r.Classify("w1", Field{Name: "x", Class: ClassPublic}, assistant("bot"))), ReasonAINotAllowed)
	expect(t, "system", second(f.r.Classify("w1", Field{Name: "x", Class: ClassPublic}, system("job"))), ReasonNotHuman)
	expect(t, "recipient", second(f.r.Classify("w1", Field{Name: "x", Class: ClassPublic}, human("acme-portal"))), ReasonRecipientActor)
	for _, c := range []string{ClassInternal, ClassSecret} {
		expect(t, "locator "+c, second(f.r.Classify("w1", Field{Name: "loc", Class: c, Locator: true}, human("s"))), ReasonLocatorClass)
	}
	must(f.r.Classify("w1", Field{Name: "pub", Class: ClassPublic, Locator: true}, human("s")))
	for i := 0; i < 4; i++ {
		must(f.r.Classify("w1", Field{Name: "f" + string(rune('a'+i)), Class: ClassPublic}, human("s")))
	}
	expect(t, "limit", second(f.r.Classify("w1", Field{Name: "fz", Class: ClassPublic}, human("s"))), "limit")
	must(f.r.Classify("w2", Field{Name: "a", Class: ClassPublic}, human("s")))
	must(f.r.Classify("w3", Field{Name: "a", Class: ClassPublic}, human("s")))
	expect(t, "workspaces", second(f.r.Classify("w4", Field{Name: "a", Class: ClassPublic}, human("s"))), "limit")
}

func TestCombinationGates(t *testing.T) {
	f := newFixture(t)
	c := Combination{ID: "c1", Fields: []string{"service", "region"}}
	for i, bad := range []Combination{{ID: "", Fields: c.Fields}, {ID: "c", Fields: []string{"a"}}, {ID: "c", Fields: []string{"a", "a"}},
		{ID: "c", Fields: []string{"a", "b", "c", "d", "e", "f", "g", "h", "i"}}, {ID: "c", Fields: []string{"a", "b c"}}} {
		if got := kind(second(f.r.ProposeCombination("w1", bad, human("risk")))); got != "invalid" {
			t.Fatalf("case %d: %s", i, got)
		}
	}
	expect(t, "ws", second(f.r.ProposeCombination("", c, human("risk"))), "invalid")
	expect(t, "actor", second(f.r.ProposeCombination("w1", c, human(""))), "invalid")
	expect(t, "missing ws", second(f.r.ProposeCombination("w9", c, human("risk"))), "not_found")
	expect(t, "assistant", second(f.r.ProposeCombination("w1", c, assistant("bot"))), ReasonAINotAllowed)
	expect(t, "recipient", second(f.r.ProposeCombination("w1", c, human("acme-portal"))), ReasonRecipientActor)
	expect(t, "unclassified", second(f.r.ProposeCombination("w1", Combination{ID: "c1", Fields: []string{"service", "zzz"}}, human("risk"))), ReasonUnclassifiedField)
	v := must(f.r.ProposeCombination("w1", c, human("risk")))
	if strings.Join(v.Combination.Fields, ",") != "region,service" {
		t.Fatalf("sorted %v", v.Combination.Fields)
	}
	if again := must(f.r.ProposeCombination("w1", Combination{ID: "c1", Fields: []string{"region", "service"}}, human("risk"))); again.Proposer != "risk" {
		t.Fatal("repeat")
	}
	expect(t, "other proposer", second(f.r.ProposeCombination("w1", c, human("x"))), "conflict")
	expect(t, "confirm form", second(f.r.ConfirmCombination("w1", "", human("r2"))), "invalid")
	expect(t, "confirm ws", second(f.r.ConfirmCombination("", "c1", human("r2"))), "invalid")
	expect(t, "confirm actor", second(f.r.ConfirmCombination("w1", "c1", human(""))), "invalid")
	expect(t, "confirm missing ws", second(f.r.ConfirmCombination("w9", "c1", human("r2"))), "not_found")
	expect(t, "confirm missing", second(f.r.ConfirmCombination("w1", "nope", human("r2"))), "not_found")
	expect(t, "confirm assistant", second(f.r.ConfirmCombination("w1", "c1", assistant("bot"))), ReasonAINotAllowed)
	expect(t, "confirm recipient", second(f.r.ConfirmCombination("w1", "c1", human("acme-portal"))), ReasonRecipientActor)
	got := must(f.r.ConfirmCombination("w1", "c1", human("r2")))
	got.Combination.Fields[0] = "x"
	if again := must(f.r.ConfirmCombination("w1", "c1", human("r2"))); again.Combination.Fields[0] != "region" {
		t.Fatal("aliasing")
	}
	expect(t, "confirm other", second(f.r.ConfirmCombination("w1", "c1", human("r3"))), "conflict")
	expect(t, "propose confirmed", second(f.r.ProposeCombination("w1", c, human("risk"))), "conflict")
}

func TestPolicyGates(t *testing.T) {
	f := newFixture(t)
	good := Policy{ID: "p2", Roles: map[string][]string{RoleCustomer: {"service"}}}
	for i, bad := range []Policy{{ID: "", Roles: good.Roles}, {ID: "p", Roles: nil}, {ID: "p", Roles: map[string][]string{"staff": {"service"}}},
		{ID: "p", Roles: map[string][]string{RoleCustomer: {}}}, {ID: "p", Roles: map[string][]string{RoleCustomer: {"a", "a"}}}} {
		if got := kind(second(f.r.Propose("w1", bad, human("author")))); got != "invalid" {
			t.Fatalf("case %d: %s", i, got)
		}
	}
	expect(t, "ws", second(f.r.Propose("", good, human("author"))), "invalid")
	expect(t, "actor", second(f.r.Propose("w1", good, human(""))), "invalid")
	expect(t, "missing ws", second(f.r.Propose("w9", good, human("author"))), "not_found")
	expect(t, "assistant", second(f.r.Propose("w1", good, assistant("bot"))), ReasonAINotAllowed)
	expect(t, "recipient", second(f.r.Propose("w1", good, human("acme-portal"))), ReasonRecipientActor)
	expect(t, "unclassified", second(f.r.Propose("w1", Policy{ID: "p2", Roles: map[string][]string{RoleCustomer: {"zzz"}}}, human("author"))), ReasonUnclassifiedField)
	for role, field := range map[string]string{RoleCustomer: "cost", RolePartner: "cost", RoleAuditor: "margin"} {
		expect(t, "class "+role, second(f.r.Propose("w1", Policy{ID: "p2", Roles: map[string][]string{role: {field}}}, human("author"))), ReasonClassNotAllowed)
	}
	v := must(f.r.Propose("w1", good, human("author")))
	if again := must(f.r.Propose("w1", good, human("author"))); again.Revision != v.Revision {
		t.Fatal("repeat")
	}
	v2 := must(f.r.Propose("w1", Policy{ID: "p2", Roles: map[string][]string{RoleCustomer: {"service", "amount"}}}, human("author2")))
	if v2.Revision != 2 || must(f.r.PolicyOf("w1", "p2", 1)).State != "replaced" {
		t.Fatal("replace")
	}
	expect(t, "confirm form", second(f.r.Confirm("w1", "p2", 0, human("a"))), "invalid")
	expect(t, "confirm ws", second(f.r.Confirm("", "p2", 2, human("a"))), "invalid")
	expect(t, "confirm id", second(f.r.Confirm("w1", "", 2, human("a"))), "invalid")
	expect(t, "confirm actor", second(f.r.Confirm("w1", "p2", 2, human(""))), "invalid")
	expect(t, "confirm missing ws", second(f.r.Confirm("w9", "p2", 2, human("a"))), "not_found")
	expect(t, "confirm missing", second(f.r.Confirm("w1", "nope", 2, human("a"))), "not_found")
	expect(t, "confirm assistant", second(f.r.Confirm("w1", "p2", 2, assistant("bot"))), ReasonAINotAllowed)
	expect(t, "confirm recipient", second(f.r.Confirm("w1", "p2", 2, human("acme-portal"))), ReasonRecipientActor)
	expect(t, "stale", second(f.r.Confirm("w1", "p2", 1, human("a"))), ReasonStaleProposal)
	expect(t, "self", second(f.r.Confirm("w1", "p2", 2, human("author2"))), ReasonSelfApproval)
	a := must(f.r.Confirm("w1", "p2", 2, human("a")))
	a.Policy.Roles[RoleCustomer][0] = "x"
	if again := must(f.r.Confirm("w1", "p2", 2, human("a"))); again.Policy.Roles[RoleCustomer][0] != "amount" {
		t.Fatal("aliasing")
	}
	expect(t, "confirm other", second(f.r.Confirm("w1", "p2", 2, human("b"))), ReasonStaleProposal)
	must(f.r.Propose("w1", Policy{ID: "p3", Roles: map[string][]string{RoleCustomer: {"service"}}}, human("author")))
	expect(t, "policies", second(f.r.Propose("w1", Policy{ID: "p4", Roles: map[string][]string{RoleCustomer: {"service"}}}, human("author"))), "limit")
	expect(t, "policy of", second(f.r.PolicyOf("w1", "p2", 3)), "not_found")
	expect(t, "policy of 0", second(f.r.PolicyOf("w1", "p2", 0)), "not_found")
	expect(t, "policy of missing", second(f.r.PolicyOf("w1", "nope", 1)), "not_found")
	if !samePolicy(Policy{Roles: map[string][]string{RoleCustomer: {"a"}}}, Policy{Roles: map[string][]string{RoleCustomer: {"a"}}}) ||
		samePolicy(Policy{Roles: map[string][]string{RoleCustomer: {"a"}}}, Policy{Roles: map[string][]string{RoleCustomer: {"a"}, RolePartner: {"a"}}}) ||
		samePolicy(Policy{Roles: map[string][]string{RoleCustomer: {"a"}}}, Policy{Roles: map[string][]string{RoleCustomer: {"b"}}}) {
		t.Fatal("same policy")
	}
}

func TestGrantGates(t *testing.T) {
	f := newFixture(t)
	good := Recipient{ID: "r1", Role: RolePartner, Customers: []string{"acme"}, Policy: "std"}
	for i, bad := range []Recipient{{ID: "", Role: RolePartner, Customers: good.Customers, Policy: "std"}, {ID: "r", Role: "staff", Customers: good.Customers, Policy: "std"},
		{ID: "r", Role: RolePartner, Customers: nil, Policy: "std"}, {ID: "r", Role: RolePartner, Customers: []string{"a", "a"}, Policy: "std"},
		{ID: "r", Role: RolePartner, Customers: good.Customers, Policy: ""}} {
		if got := kind(second(f.r.Grant("w1", bad, human("author")))); got != "invalid" {
			t.Fatalf("case %d: %s", i, got)
		}
	}
	expect(t, "ws", second(f.r.Grant("", good, human("author"))), "invalid")
	expect(t, "actor", second(f.r.Grant("w1", good, human(""))), "invalid")
	expect(t, "missing ws", second(f.r.Grant("w9", good, human("author"))), "not_found")
	expect(t, "assistant", second(f.r.Grant("w1", good, assistant("bot"))), ReasonAINotAllowed)
	expect(t, "recipient grants", second(f.r.Grant("w1", good, human("acme-portal"))), ReasonRecipientActor)
	self := good
	self.ID = "author"
	expect(t, "self grant", second(f.r.Grant("w1", self, human("author"))), ReasonSelfApproval)
	v := must(f.r.Grant("w1", good, human("author")))
	if again := must(f.r.Grant("w1", good, human("author"))); again.Revision != v.Revision {
		t.Fatal("repeat")
	}
	wider := good
	wider.Customers = []string{"acme", "beta"}
	v2 := must(f.r.Grant("w1", wider, human("author2")))
	expect(t, "confirm form", second(f.r.ConfirmGrant("w1", "r1", 0, human("a"))), "invalid")
	expect(t, "confirm ws", second(f.r.ConfirmGrant("", "r1", 2, human("a"))), "invalid")
	expect(t, "confirm id", second(f.r.ConfirmGrant("w1", "", 2, human("a"))), "invalid")
	expect(t, "confirm actor", second(f.r.ConfirmGrant("w1", "r1", 2, human(""))), "invalid")
	expect(t, "confirm missing ws", second(f.r.ConfirmGrant("w9", "r1", 2, human("a"))), "not_found")
	expect(t, "confirm missing", second(f.r.ConfirmGrant("w1", "nope", 2, human("a"))), "not_found")
	expect(t, "confirm assistant", second(f.r.ConfirmGrant("w1", "r1", 2, assistant("bot"))), ReasonAINotAllowed)
	expect(t, "confirm recipient", second(f.r.ConfirmGrant("w1", "r1", 2, human("r1"))), ReasonRecipientActor)
	expect(t, "stale", second(f.r.ConfirmGrant("w1", "r1", 1, human("a"))), ReasonStaleProposal)
	expect(t, "self", second(f.r.ConfirmGrant("w1", "r1", v2.Revision, human("author2"))), ReasonSelfApproval)
	c := must(f.r.ConfirmGrant("w1", "r1", 2, human("a")))
	c.Recipient.Customers[0] = "x"
	if again := must(f.r.ConfirmGrant("w1", "r1", 2, human("a"))); again.Recipient.Customers[0] != "acme" {
		t.Fatal("aliasing")
	}
	expect(t, "confirm other", second(f.r.ConfirmGrant("w1", "r1", 2, human("b"))), ReasonStaleProposal)
	must(f.r.Grant("w1", Recipient{ID: "r2", Role: RoleCustomer, Customers: []string{"acme"}, Policy: "std"}, human("author")))
	expect(t, "confirm revoked pending", second(f.r.RevokeRecipient("w1", "r2", human("ops"))), "ok")
	expect(t, "confirm after revoke", second(f.r.ConfirmGrant("w1", "r2", 1, human("a"))), ReasonRecipientRevoked)
	expect(t, "revoke form", second(f.r.RevokeRecipient("w1", "", human("ops"))), "invalid")
	expect(t, "revoke ws", second(f.r.RevokeRecipient("", "r2", human("ops"))), "invalid")
	expect(t, "revoke actor", second(f.r.RevokeRecipient("w1", "r2", human(""))), "invalid")
	expect(t, "revoke missing ws", second(f.r.RevokeRecipient("w9", "r2", human("ops"))), "not_found")
	expect(t, "revoke missing", second(f.r.RevokeRecipient("w1", "nope", human("ops"))), "not_found")
	must(f.r.RevokeRecipient("w1", "r2", human("ops")))
	expect(t, "revoke other", second(f.r.RevokeRecipient("w1", "r2", human("ops2"))), ReasonRecipientRevoked)
	expect(t, "revoke assistant", second(f.r.RevokeRecipient("w1", "r1", assistant("bot"))), ReasonAINotAllowed)
	expect(t, "revoke recipient", second(f.r.RevokeRecipient("w1", "r1", human("acme-portal"))), ReasonRecipientActor)
	must(f.r.Grant("w1", Recipient{ID: "r3", Role: RoleCustomer, Customers: []string{"acme"}, Policy: "std"}, human("author")))
	expect(t, "recipients", second(f.r.Grant("w1", Recipient{ID: "r4", Role: RoleCustomer, Customers: []string{"acme"}, Policy: "std"}, human("author"))), "limit")
}

func TestProjectionGates(t *testing.T) {
	f := newFixture(t)
	req := Request{Policy: "std", Recipient: "acme-portal", Rows: rows()}
	many := make([]Row, 11)
	for i := range many {
		many[i] = Row{"customer": "acme"}
	}
	expect(t, "rows limit", second(f.r.Project("w1", Request{Policy: "std", Recipient: "acme-portal", Rows: many}, system("x"))), "limit")
	for i, bad := range []Request{{Policy: "", Recipient: "acme-portal"}, {Policy: "std", Recipient: ""}, {Policy: "std", Recipient: "acme-portal", Rows: []Row{{"service": "x"}}},
		{Policy: "std", Recipient: "acme-portal", Rows: []Row{{"customer": "acme", "a b": "x"}}}, {Policy: "std", Recipient: "acme-portal", Rows: []Row{{"customer": "acme", "x": "\xff"}}}} {
		if got := kind(second(f.r.Project("w1", bad, system("x")))); got != "invalid" {
			t.Fatalf("case %d: %s", i, got)
		}
	}
	expect(t, "ws", second(f.r.Project("", req, system("x"))), "invalid")
	expect(t, "actor", second(f.r.Project("w1", req, system(""))), "invalid")
	expect(t, "missing ws", second(f.r.Project("w9", req, system("x"))), "not_found")
	expect(t, "assistant", second(f.r.Project("w1", req, assistant("bot"))), ReasonAINotAllowed)
	expect(t, "unknown recipient", second(f.r.Project("w1", Request{Policy: "std", Recipient: "nobody", Rows: rows()}, system("x"))), "not_found")
	must(f.r.Grant("w1", Recipient{ID: "pending", Role: RoleCustomer, Customers: []string{"acme"}, Policy: "std"}, human("author")))
	expect(t, "ungranted", second(f.r.Project("w1", Request{Policy: "std", Recipient: "pending", Rows: rows()}, system("x"))), "not_found")
	must(f.r.Classify("w1", Field{Name: "z", Class: ClassPublic}, human("s")))
	must(f.r.Propose("w1", Policy{ID: "other", Roles: map[string][]string{RoleCustomer: {"customer", "service", "amount", "cost2"[:0] + "region"}}}, human("author")))
	expect(t, "not granted", second(f.r.Project("w1", Request{Policy: "other", Recipient: "acme-portal", Rows: rows()}, system("x"))), ReasonPolicyNotGranted)
	f.grant(t, Recipient{ID: "other-r", Role: RoleCustomer, Customers: []string{"acme"}, Policy: "other"})
	expect(t, "no active", second(f.r.Project("w1", Request{Policy: "other", Recipient: "other-r", Rows: rows()}, system("x"))), ReasonNoActivePolicy)
	expect(t, "key", second(f.r.Project("nokey", req, system("x"))), "not_found")
	clock := &testClock{now: start}
	r := must(NewRegistry(Config{Keys: func(string) ([]byte, error) { return nil, nil }, Random: failing{}}, testLimits(), clock.read))
	g := fixture{r: r, clock: clock}
	must(r.Classify("w1", Field{Name: "customer", Class: ClassCustomer}, human("steward")))
	g.policy(t, Policy{ID: "std", Roles: map[string][]string{RoleCustomer: {"customer"}}})
	g.grant(t, Recipient{ID: "acme-portal", Role: RoleCustomer, Customers: []string{"acme"}, Policy: "std"})
	expect(t, "empty key", second(r.Project("w1", req, system("x"))), ReasonKeyUnavailable)
	p := must(f.r.Project("w1", req, system("x")))
	again := must(f.r.Project("w1", req, system("y")))
	if again.Digest != p.Digest || must(f.r.ProjectionOf("w1", p.Digest, human("staff"))).Actor != "x" {
		t.Fatal("repeat keeps first audit")
	}
	p.Rows[0][0] = "tampered"
	p.Fields[0] = "tampered"
	if a := must(f.r.ProjectionOf("w1", p.Digest, human("staff"))); a.Projection.Rows[0][0] == "tampered" || a.Projection.Fields[0] == "tampered" {
		t.Fatal("aliasing")
	}
	a := must(f.r.ProjectionOf("w1", p.Digest, human("staff")))
	a.DroppedFields[0] = "x"
	if must(f.r.ProjectionOf("w1", p.Digest, human("staff"))).DroppedFields[0] != "cost" {
		t.Fatal("audit aliasing")
	}
	expect(t, "audit form", second(f.r.ProjectionOf("w1", "", human("staff"))), "invalid")
	expect(t, "audit ws", second(f.r.ProjectionOf("", p.Digest, human("staff"))), "invalid")
	expect(t, "audit actor", second(f.r.ProjectionOf("w1", p.Digest, human(""))), "invalid")
	expect(t, "audit missing ws", second(f.r.ProjectionOf("w9", p.Digest, human("staff"))), "not_found")
	expect(t, "audit missing", second(f.r.ProjectionOf("w1", "nope", human("staff"))), "not_found")
	expect(t, "audit system", second(f.r.ProjectionOf("w1", p.Digest, system("job"))), ReasonNotHuman)
	expect(t, "audit recipient", second(f.r.ProjectionOf("w1", p.Digest, human("acme-portal"))), ReasonRecipientActor)
	expect(t, "resolve form", second(f.r.Resolve("w1", "x", human("staff"))), "invalid")
	expect(t, "resolve ws", second(f.r.Resolve("", "ref:x", human("staff"))), "invalid")
	expect(t, "resolve actor", second(f.r.Resolve("w1", "ref:x", human(""))), "invalid")
	expect(t, "resolve missing", second(f.r.Resolve("w1", "ref:000", human("staff"))), "not_found")
	expect(t, "resolve missing ws", second(f.r.Resolve("w9", "ref:000", human("staff"))), "not_found")
	expect(t, "resolve assistant", second(f.r.Resolve("w1", "ref:000", assistant("bot"))), ReasonAINotAllowed)
	h := newFixture(t)
	for i := 0; i < 20; i++ {
		must(h.r.Project("w1", Request{Policy: "std", Recipient: "acme-portal", Rows: []Row{{"customer": "acme", "service": "s" + string(rune('a'+i))}}}, system("x")))
	}
	expect(t, "projections", second(h.r.Project("w1", Request{Policy: "std", Recipient: "acme-portal", Rows: []Row{{"customer": "acme", "service": "zz"}}}, system("x"))), "limit")
}

func TestPackageLinkPreviewGates(t *testing.T) {
	f := newFixture(t)
	p := must(f.r.Project("w1", Request{Policy: "std", Recipient: "acme-portal", Rows: rows()}, system("x")))
	expect(t, "pkg form", second(f.r.Package("w1", "", system("x"))), "invalid")
	expect(t, "pkg ws", second(f.r.Package("", p.Digest, system("x"))), "invalid")
	expect(t, "pkg actor", second(f.r.Package("w1", p.Digest, system(""))), "invalid")
	expect(t, "pkg missing ws", second(f.r.Package("w9", p.Digest, system("x"))), "not_found")
	expect(t, "pkg assistant", second(f.r.Package("w1", p.Digest, assistant("bot"))), ReasonAINotAllowed)
	pkg := must(f.r.Package("w1", p.Digest, system("x")))
	pkg.Bytes[0] = 0
	expect(t, "link form", third(f.r.Link("w1", pkg.ID, time.Minute, human("ops"))), "invalid")
	expect(t, "link max", third(f.r.Link("w1", pkg.ID, 30*24*time.Hour+1, human("ops"))), "invalid")
	expect(t, "link ws", third(f.r.Link("", pkg.ID, time.Hour, human("ops"))), "invalid")
	expect(t, "link id", third(f.r.Link("w1", "", time.Hour, human("ops"))), "invalid")
	expect(t, "link actor", third(f.r.Link("w1", pkg.ID, time.Hour, human(""))), "invalid")
	expect(t, "link missing ws", third(f.r.Link("w9", pkg.ID, time.Hour, human("ops"))), "not_found")
	expect(t, "link missing", third(f.r.Link("w1", "pkg-x", time.Hour, human("ops"))), "not_found")
	expect(t, "link assistant", third(f.r.Link("w1", pkg.ID, time.Hour, assistant("bot"))), ReasonAINotAllowed)
	token, link, err := f.r.Link("w1", pkg.ID, 30*24*time.Hour, system("mailer"))
	if err != nil || len(token) != 32 || !link.Expires.Equal(start.Add(30*24*time.Hour)) {
		t.Fatalf("link %v %+v", err, link)
	}
	if got := must(f.r.Access(token, human("acme-portal"))); got.Bytes[0] == 0 || got.File != pkg.File {
		t.Fatal("package aliasing")
	}
	expect(t, "access form", second(f.r.Access("", human("acme-portal"))), "invalid")
	expect(t, "access long", second(f.r.Access(strings.Repeat("a", 65), human("acme-portal"))), "invalid")
	expect(t, "access actor", second(f.r.Access(token, human(""))), "invalid")
	expect(t, "access assistant", second(f.r.Access(token, assistant("bot"))), ReasonAINotAllowed)
	expect(t, "revoke form", second(f.r.RevokeLink("w1", "", human("ops"))), "invalid")
	expect(t, "revoke ws", second(f.r.RevokeLink("", link.Digest, human("ops"))), "invalid")
	expect(t, "revoke actor", second(f.r.RevokeLink("w1", link.Digest, human(""))), "invalid")
	expect(t, "revoke missing ws", second(f.r.RevokeLink("w9", link.Digest, human("ops"))), "not_found")
	expect(t, "revoke missing", second(f.r.RevokeLink("w1", "nope", human("ops"))), "not_found")
	expect(t, "revoke system", second(f.r.RevokeLink("w1", link.Digest, system("job"))), ReasonNotHuman)
	expect(t, "revoke recipient", second(f.r.RevokeLink("w1", link.Digest, human("acme-portal"))), ReasonRecipientActor)
	must(f.r.RevokeLink("w1", link.Digest, human("ops")))
	must(f.r.RevokeLink("w1", link.Digest, human("ops")))
	expect(t, "revoke other", second(f.r.RevokeLink("w1", link.Digest, human("ops2"))), ReasonLinkRevoked)
	expect(t, "preview form page", second(f.r.Preview("w1", p.Digest, 0, 1, false, human("acme-portal"))), "invalid")
	expect(t, "preview form size", second(f.r.Preview("w1", p.Digest, 1, 0, false, human("acme-portal"))), "invalid")
	expect(t, "preview max size", second(f.r.Preview("w1", p.Digest, 1, 501, false, human("acme-portal"))), "invalid")
	expect(t, "preview size 500", second(f.r.Preview("w1", p.Digest, 1, 500, false, human("acme-portal"))), "ok")
	expect(t, "preview ws", second(f.r.Preview("", p.Digest, 1, 1, false, human("acme-portal"))), "invalid")
	expect(t, "preview id", second(f.r.Preview("w1", "", 1, 1, false, human("acme-portal"))), "invalid")
	expect(t, "preview actor", second(f.r.Preview("w1", p.Digest, 1, 1, false, human(""))), "invalid")
	expect(t, "preview missing ws", second(f.r.Preview("w9", p.Digest, 1, 1, false, human("acme-portal"))), "not_found")
	expect(t, "preview missing", second(f.r.Preview("w1", "nope", 1, 1, false, human("acme-portal"))), "not_found")
	expect(t, "preview assistant", second(f.r.Preview("w1", p.Digest, 1, 1, false, assistant("bot"))), ReasonAINotAllowed)
	expect(t, "preview internal system", second(f.r.Preview("w1", p.Digest, 1, 1, true, system("job"))), ReasonNotHuman)
	expect(t, "preview other system", second(f.r.Preview("w1", p.Digest, 1, 1, false, system("job"))), ReasonNotHuman)
	expect(t, "preview recipient system", second(f.r.Preview("w1", p.Digest, 1, 1, false, system("acme-portal"))), "ok")
	expect(t, "preview staff", second(f.r.Preview("w1", p.Digest, 1, 1, false, human("staff"))), "ok")
	clock := &testClock{now: start}
	r := must(NewRegistry(Config{Keys: keys, Random: failing{}}, testLimits(), clock.read))
	g := fixture{r: r, clock: clock}
	must(r.Classify("w1", Field{Name: "customer", Class: ClassCustomer}, human("steward")))
	g.policy(t, Policy{ID: "std", Roles: map[string][]string{RoleCustomer: {"customer"}}})
	g.grant(t, Recipient{ID: "acme-portal", Role: RoleCustomer, Customers: []string{"acme"}, Policy: "std"})
	q := must(r.Project("w1", Request{Policy: "std", Recipient: "acme-portal", Rows: rows()}, system("x")))
	expect(t, "no entropy", second(r.Package("w1", q.Digest, system("x"))), "invalid")
	h := newFixture(t)
	hp := must(h.r.Project("w1", Request{Policy: "std", Recipient: "acme-portal", Rows: rows()}, system("x")))
	for i := 0; i < 20; i++ {
		must(h.r.Package("w1", hp.Digest, system("x")))
	}
	expect(t, "packages", second(h.r.Package("w1", hp.Digest, system("x"))), "limit")
	k := newFixture(t)
	kp := must(k.r.Project("w1", Request{Policy: "std", Recipient: "acme-portal", Rows: rows()}, system("x")))
	kpkg := must(k.r.Package("w1", kp.Digest, system("x")))
	for i := 0; i < 20; i++ {
		if _, _, err := k.r.Link("w1", kpkg.ID, time.Hour, human("ops")); err != nil {
			t.Fatal(err)
		}
	}
	expect(t, "links", third(k.r.Link("w1", kpkg.ID, time.Hour, human("ops"))), "limit")
	k.policy(t, Policy{ID: "std", Roles: map[string][]string{RoleCustomer: {"customer"}}})
	expect(t, "link superseded", third(k.r.Link("w1", kpkg.ID, time.Hour, human("ops"))), ReasonPolicySuperseded)
	expect(t, "package superseded", second(k.r.Package("w1", kp.Digest, system("x"))), ReasonPolicySuperseded)
}

func TestRandomCollisions(t *testing.T) {
	clock := &testClock{now: start}
	r := must(NewRegistry(Config{Keys: keys, Random: repeating{}}, testLimits(), clock.read))
	g := fixture{r: r, clock: clock}
	must(r.Classify("w1", Field{Name: "customer", Class: ClassCustomer}, human("steward")))
	g.policy(t, Policy{ID: "std", Roles: map[string][]string{RoleCustomer: {"customer"}}})
	g.grant(t, Recipient{ID: "acme-portal", Role: RoleCustomer, Customers: []string{"acme"}, Policy: "std"})
	p := must(r.Project("w1", Request{Policy: "std", Recipient: "acme-portal", Rows: rows()}, system("x")))
	pkg := must(r.Package("w1", p.Digest, system("x")))
	expect(t, "package id collision", second(r.Package("w1", p.Digest, system("x"))), "conflict")
	must2 := func(a string, b LinkView, err error) error { return err }
	if err := must2(r.Link("w1", pkg.ID, time.Hour, human("ops"))); err != nil {
		t.Fatal(err)
	}
	expect(t, "token collision", must2(r.Link("w1", pkg.ID, time.Hour, human("ops"))), "conflict")
}

type limited struct {
	left int
	c    counter
}

func (l *limited) Read(p []byte) (int, error) {
	if l.left == 0 {
		return 0, errors.New("exhausted")
	}
	l.left--
	return l.c.Read(p)
}

func TestRemainingBranches(t *testing.T) {
	f := newFixture(t)
	must(f.r.RevokeRecipient("w1", "acme-portal", human("ops")))
	expect(t, "project revoked", second(f.r.Project("w1", Request{Policy: "std", Recipient: "acme-portal", Rows: rows()}, system("x"))), ReasonRecipientRevoked)
	g := newFixture(t)
	p := must(g.r.Project("w1", Request{Policy: "std", Recipient: "acme-portal", Rows: rows()[:1]}, system("x")))
	ref := p.Rows[0][3]
	g.r.workspaces["w1"].refs[ref] = "someone-else\x00other"
	expect(t, "collision", second(g.r.Project("w1", Request{Policy: "std", Recipient: "acme-portal", Rows: []Row{{"customer": "acme", "service": "other", "source": "s3://bucket/acme/2026-10.csv"}}}, system("x"))), ReasonReferenceCollision)
	clock := &testClock{now: start}
	r := must(NewRegistry(Config{Keys: keys, Random: &limited{left: 1}}, testLimits(), clock.read))
	h := fixture{r: r, clock: clock}
	must(r.Classify("w1", Field{Name: "customer", Class: ClassCustomer}, human("steward")))
	h.policy(t, Policy{ID: "std", Roles: map[string][]string{RoleCustomer: {"customer"}}})
	h.grant(t, Recipient{ID: "acme-portal", Role: RoleCustomer, Customers: []string{"acme"}, Policy: "std"})
	q := must(r.Project("w1", Request{Policy: "std", Recipient: "acme-portal", Rows: rows()}, system("x")))
	pkg := must(r.Package("w1", q.Digest, system("x")))
	expect(t, "link entropy", third(r.Link("w1", pkg.ID, time.Hour, human("ops"))), "invalid")
}

func TestChangeReviewRegressions(t *testing.T) {
	f := newFixture(t)
	p := must(f.r.Project("w1", Request{Policy: "std", Recipient: "acme-portal", Rows: rows()}, system("portal")))
	must(f.r.Preview("w1", p.Digest, 1, 10, false, human("acme-portal")))
	f.policy(t, Policy{ID: "std", Roles: map[string][]string{RoleCustomer: {"customer", "service"}}})
	expect(t, "preview superseded", second(f.r.Preview("w1", p.Digest, 1, 10, false, human("acme-portal"))), ReasonPolicySuperseded)
	expect(t, "staff preview stale ok", second(f.r.Preview("w1", p.Digest, 1, 10, false, human("staff"))), "ok")
	p2 := must(f.r.Project("w1", Request{Policy: "std", Recipient: "acme-portal", Rows: rows()}, system("portal")))
	f.grant(t, Recipient{ID: "acme-portal", Role: RoleCustomer, Customers: []string{"acme", "beta"}, Policy: "std"})
	expect(t, "preview grant changed", second(f.r.Preview("w1", p2.Digest, 1, 10, false, human("acme-portal"))), ReasonRecipientChanged)
	p3 := must(f.r.Project("w1", Request{Policy: "std", Recipient: "acme-portal", Rows: rows()}, system("portal")))
	must(f.r.ProposeCombination("w1", Combination{ID: "c", Fields: []string{"customer", "service"}}, human("risk")))
	must(f.r.ConfirmCombination("w1", "c", human("risk2")))
	expect(t, "preview suspended", second(f.r.Preview("w1", p3.Digest, 1, 10, false, human("acme-portal"))), ReasonPolicySuspended)
	n := newFixture(t)
	np := must(n.r.Project("w1", Request{Policy: "std", Recipient: "acme-portal", Rows: rows()}, system("portal")))
	must(n.r.RevokeRecipient("w1", "acme-portal", human("ops")))
	expect(t, "preview revoked", second(n.r.Preview("w1", np.Digest, 1, 10, false, human("acme-portal"))), ReasonRecipientRevoked)
	g := newFixture(t)
	g.grant(t, Recipient{ID: "beta-portal", Role: RoleCustomer, Customers: []string{"beta"}, Policy: "std"})
	expect(t, "recipient projects", second(g.r.Project("w1", Request{Policy: "std", Recipient: "beta-portal", Rows: rows()}, human("acme-portal"))), ReasonRecipientActor)
	expect(t, "recipient system projects", second(g.r.Project("w1", Request{Policy: "std", Recipient: "beta-portal", Rows: rows()}, system("acme-portal"))), ReasonRecipientActor)
	gp := must(g.r.Project("w1", Request{Policy: "std", Recipient: "acme-portal", Rows: rows()}, system("portal")))
	expect(t, "recipient packages", second(g.r.Package("w1", gp.Digest, human("beta-portal"))), ReasonRecipientActor)
	gpkg := must(g.r.Package("w1", gp.Digest, system("portal")))
	expect(t, "recipient links", third(g.r.Link("w1", gpkg.ID, time.Hour, human("beta-portal"))), ReasonRecipientActor)
	token, _, _ := g.r.Link("w1", gpkg.ID, time.Hour, human("ops"))
	must(g.r.RevokeLink("w1", tokenDigest(token), human("ops")))
	expect(t, "mismatch before link state", second(g.r.Access(token, human("beta-portal"))), ReasonRecipientMismatch)
	must(g.r.Classify("w2", Field{Name: "customer", Class: ClassCustomer}, human("steward")))
	must(g.r.Classify("w2", Field{Name: "service", Class: ClassPublic}, human("steward")))
	must(g.r.Propose("w2", Policy{ID: "p", Roles: map[string][]string{RolePartner: {"service"}}}, human("author")))
	expect(t, "cross workspace recipient", second(g.r.Classify("w2", Field{Name: "x", Class: ClassPublic}, human("acme-portal"))), ReasonRecipientActor)
	must(g.r.Confirm("w2", "p", 1, human("approver")))
	v := must(g.r.Grant("w2", Recipient{ID: "c2", Role: RoleCustomer, Customers: []string{"acme"}, Policy: "p"}, human("author")))
	must(g.r.ConfirmGrant("w2", "c2", v.Revision, human("approver")))
	expect(t, "no fields", second(g.r.Project("w2", Request{Policy: "p", Recipient: "c2", Rows: rows()}, system("x"))), ReasonNoFields)
	must(g.r.Classify("w2", Field{Name: "-A1", Class: ClassPublic}, human("steward")))
	must(g.r.Propose("w2", Policy{ID: "p", Roles: map[string][]string{RoleCustomer: {"customer", "-A1"}}}, human("author")))
	must(g.r.Confirm("w2", "p", 2, human("approver")))
	hp := must(g.r.Project("w2", Request{Policy: "p", Recipient: "c2", Rows: []Row{{"customer": "acme", "-A1": "v"}}}, system("x")))
	hpkg := must(g.r.Package("w2", hp.Digest, system("x")))
	if !strings.HasPrefix(string(hpkg.Bytes), "\ufeff'-A1,customer") {
		t.Fatalf("header %q", hpkg.Bytes)
	}
	k := newFixture(t)
	for i := 0; i < 4; i++ {
		must(k.r.ProposeCombination("w1", Combination{ID: "c" + string(rune('0'+i)), Fields: []string{"customer", "region"}}, human("risk")))
	}
	expect(t, "combinations", second(k.r.ProposeCombination("w1", Combination{ID: "c9", Fields: []string{"customer", "region"}}, human("risk"))), "limit")
	m := newFixture(t)
	m.grant(t, Recipient{ID: "r2", Role: RoleCustomer, Customers: []string{"acme"}, Policy: "std"})
	before := must(m.r.Grant("w1", Recipient{ID: "r2", Role: RoleCustomer, Customers: []string{"acme", "beta"}, Policy: "std"}, human("author")))
	m.clock.advance(time.Hour)
	rv := must(m.r.RevokeRecipient("w1", "r2", human("ops")))
	if !rv.At.Equal(before.At) || rv.RevokedBy != "ops" || !rv.RevokedAt.Equal(start.Add(time.Hour)) {
		t.Fatalf("revocation history %+v", rv)
	}
}

func tokenDigest(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func TestMutationGuards(t *testing.T) {
	for text, ok := range map[string]bool{"0": true, "10": true, "1.5": true, "-1.5": true, "123456789012345678": true, "1234567890123456789": false, "1.12345678": true, "1.123456789": false,
		"01": false, "00": false, "-0": false, "-0.00": false, "-0.01": true, "1.": false, ".5": false, "": false, "-": false, "1a": false, "1.a": false, "+1": false, "9": true, "/": false, ":": false} {
		if canonical(text) != ok {
			t.Fatalf("canonical %q", text)
		}
	}
	if neutral("-5") != "-5" || neutral("-x") != "'-x" || neutral(" =1") != "' =1" || neutral("") != "" || neutral("ok") != "ok" {
		t.Fatal("neutral")
	}
	f := newFixture(t)
	v := must(f.r.Grant("w1", Recipient{ID: "r2", Role: RoleCustomer, Customers: []string{"acme"}, Policy: "std"}, human("author")))
	if v.State != "proposed" {
		t.Fatal(v.State)
	}
	v2 := must(f.r.Grant("w1", Recipient{ID: "r2", Role: RoleCustomer, Customers: []string{"acme", "beta"}, Policy: "std"}, human("author")))
	_ = v2
	c := must(f.r.ConfirmGrant("w1", "r2", 2, human("approver")))
	if c.State != "active" {
		t.Fatal(c.State)
	}
	v3 := must(f.r.Grant("w1", Recipient{ID: "r2", Role: RoleCustomer, Customers: []string{"acme"}, Policy: "std"}, human("author")))
	must(f.r.ConfirmGrant("w1", "r2", v3.Revision, human("approver")))
	v4 := must(f.r.Grant("w1", Recipient{ID: "r2", Role: RoleCustomer, Customers: []string{"beta"}, Policy: "std"}, human("author")))
	rv := must(f.r.RevokeRecipient("w1", "r2", human("ops")))
	if rv.Revision != v4.Revision || rv.State != "revoked" {
		t.Fatalf("revoked %+v", rv)
	}
	states := []string{}
	for i := 1; i <= 4; i++ {
		states = append(states, f.r.workspaces["w1"].grants["r2"].revisions[i-1].State)
	}
	if strings.Join(states, ",") != "replaced,superseded,revoked,revoked" {
		t.Fatalf("grant states %v", states)
	}
	pv := must(f.r.Propose("w1", Policy{ID: "std", Roles: map[string][]string{RoleCustomer: {"customer"}}}, human("author")))
	if pv.State != "proposed" {
		t.Fatal(pv.State)
	}
	if got := must(f.r.PolicyOf("w1", "std", pv.Revision)); got.Revision != pv.Revision {
		t.Fatal("policy of last")
	}
	must(f.r.Classify("w1", Field{Name: "a1", Class: ClassPublic}, human("s")))
	expect(t, "eight fields", second(f.r.ProposeCombination("w1", Combination{ID: "eight", Fields: []string{"customer", "service", "amount", "source", "cost", "margin", "region", "a1"}}, human("risk"))), "ok")
	var customers []string
	for i := 0; i < 1001; i++ {
		customers = append(customers, "c"+itoa(i))
	}
	expect(t, "1001 customers", second(f.r.Grant("w1", Recipient{ID: "big", Role: RoleCustomer, Customers: customers, Policy: "std"}, human("author"))), "invalid")
	expect(t, "1000 customers", second(f.r.Grant("w1", Recipient{ID: "big", Role: RoleCustomer, Customers: customers[:1000], Policy: "std"}, human("author"))), "ok")
	ten := make([]Row, 10)
	for i := range ten {
		ten[i] = Row{"customer": "acme", "service": "s" + itoa(i)}
	}
	expect(t, "ten rows", second(f.r.Project("w1", Request{Policy: "std", Recipient: "acme-portal", Rows: ten}, system("x"))), "ok")
	expect(t, "64-char token", second(f.r.Access(strings.Repeat("a", 64), human("acme-portal"))), "not_found")
	p := must(f.r.Project("w1", Request{Policy: "std", Recipient: "acme-portal", Rows: rows()}, system("x")))
	pkg := must(f.r.Package("w1", p.Digest, system("x")))
	if len(pkg.ID) != 36 {
		t.Fatalf("package id %s", pkg.ID)
	}
	for _, role := range []string{RolePartner, RoleAuditor} {
		g := newFixture(t)
		must(g.r.ProposeCombination("w1", Combination{ID: "c", Fields: []string{"customer", "service"}}, human("risk")))
		must(g.r.ConfirmCombination("w1", "c", human("risk2")))
		expect(t, "violation "+role, second(g.r.Propose("w1", Policy{ID: "x", Roles: map[string][]string{role: {"customer", "service"}}}, human("author"))), ReasonInferenceConflict)
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	out := ""
	for ; i > 0; i /= 10 {
		out = string(rune('0'+i%10)) + out
	}
	return out
}

func TestDigestKnownAnswer(t *testing.T) {
	sum := sha256.New()
	for _, text := range []string{"ab", "c"} {
		sum.Write([]byte{0, 0, 0, 0, 0, 0, 0, byte(len(text))})
		sum.Write([]byte(text))
	}
	if digest("ab", "c") != hex.EncodeToString(sum.Sum(nil)) {
		t.Fatal("digest framing")
	}
}
