package disclosure

import (
	"strings"
	"testing"
	"time"
)

func TestNoLeaks(t *testing.T) {
	f := newFixture(t)
	p := must(f.r.Project("w1", Request{Policy: "std", Recipient: "acme-portal", Rows: rows()}, system("portal")))
	if strings.Join(p.Fields, ",") != "amount,customer,service,source" || len(p.Rows) != 2 {
		t.Fatalf("projection %+v", p)
	}
	for _, row := range p.Rows {
		joined := strings.Join(row, "|")
		for _, leak := range []string{"beta", "999.00", "7777.77", "3333.33", "s3://", "newfield", "x|"} {
			if strings.Contains(joined, leak) {
				t.Fatalf("leak %s in %v", leak, row)
			}
		}
	}
	if !strings.HasPrefix(p.Rows[0][3], "ref:") && !strings.HasPrefix(p.Rows[1][3], "ref:") {
		t.Fatalf("no reference %v", p.Rows)
	}
	audit := must(f.r.ProjectionOf("w1", p.Digest, human("auditor-staff")))
	if audit.DroppedRows != 1 || strings.Join(audit.DroppedFields, ",") != "cost,margin,newfield" {
		t.Fatalf("audit %+v", audit)
	}
	pkg := must(f.r.Package("w1", p.Digest, system("portal")))
	if !strings.HasPrefix(pkg.File, "pkg-") || !strings.HasSuffix(pkg.File, ".csv") || strings.Contains(pkg.File, "acme") {
		t.Fatalf("file name %s", pkg.File)
	}
	body := string(pkg.Bytes)
	if strings.Contains(body, "beta") || strings.Contains(body, "s3://") || !strings.Contains(body, "'=HYPERLINK") || !strings.Contains(body, "100.00") {
		t.Fatalf("package body %q", body)
	}
	expect(t, "recipient internal preview", second(f.r.Preview("w1", p.Digest, 1, 10, true, human("acme-portal"))), ReasonRecipientActor)
	pv := must(f.r.Preview("w1", p.Digest, 1, 10, false, human("acme-portal")))
	if strings.Contains(pv, "dropped") {
		t.Fatalf("recipient preview leaks counts %s", pv)
	}
}

func TestRolesSeeOnlyTheirAllowlist(t *testing.T) {
	f := newFixture(t)
	f.grant(t, Recipient{ID: "reseller", Role: RolePartner, Customers: []string{"acme", "beta"}, Policy: "std"})
	f.grant(t, Recipient{ID: "audit-firm", Role: RoleAuditor, Customers: []string{"beta"}, Policy: "std"})
	partner := must(f.r.Project("w1", Request{Policy: "std", Recipient: "reseller", Rows: rows()}, system("api")))
	if strings.Join(partner.Fields, ",") != "customer,service" || len(partner.Rows) != 3 {
		t.Fatalf("partner %+v", partner)
	}
	auditor := must(f.r.Project("w1", Request{Policy: "std", Recipient: "audit-firm", Rows: rows()}, system("api")))
	if strings.Join(auditor.Fields, ",") != "amount,cost,customer,service" || len(auditor.Rows) != 1 || auditor.Rows[0][1] != "1" {
		t.Fatalf("auditor %+v", auditor)
	}
}

func TestInferenceCombinations(t *testing.T) {
	f := newFixture(t)
	must(f.r.ProposeCombination("w1", Combination{ID: "geo", Fields: []string{"region", "amount"}}, human("risk")))
	expect(t, "self confirm", second(f.r.ConfirmCombination("w1", "geo", human("risk"))), ReasonSelfApproval)
	must(f.r.ConfirmCombination("w1", "geo", human("risk2")))
	expect(t, "propose violation", second(f.r.Propose("w1", Policy{ID: "geo", Roles: map[string][]string{RoleCustomer: {"region", "amount"}}}, human("author"))), ReasonInferenceConflict)
	v := must(f.r.Propose("w1", Policy{ID: "std", Roles: map[string][]string{RoleCustomer: {"customer", "region", "service"}}}, human("author")))
	must(f.r.ProposeCombination("w1", Combination{ID: "svc", Fields: []string{"region", "service"}}, human("risk")))
	must(f.r.ConfirmCombination("w1", "svc", human("risk2")))
	expect(t, "confirm violation", second(f.r.Confirm("w1", "std", v.Revision, human("approver"))), ReasonInferenceConflict)
	p := must(f.r.Project("w1", Request{Policy: "std", Recipient: "acme-portal", Rows: rows()}, system("portal")))
	pkg := must(f.r.Package("w1", p.Digest, system("portal")))
	token, _, err := f.r.Link("w1", pkg.ID, time.Hour, human("ops"))
	if err != nil {
		t.Fatal(err)
	}
	must(f.r.ProposeCombination("w1", Combination{ID: "amt", Fields: []string{"amount", "service"}}, human("risk")))
	must(f.r.ConfirmCombination("w1", "amt", human("risk2")))
	if pol := must(f.r.PolicyOf("w1", "std", 1)); pol.State != "suspended" {
		t.Fatalf("not suspended %+v", pol)
	}
	expect(t, "project suspended", second(f.r.Project("w1", Request{Policy: "std", Recipient: "acme-portal", Rows: rows()}, system("portal"))), ReasonPolicySuspended)
	expect(t, "access suspended", second(f.r.Access(token, system("acme-portal"))), ReasonPolicySuspended)
}

func TestAccessStops(t *testing.T) {
	f := newFixture(t)
	p := must(f.r.Project("w1", Request{Policy: "std", Recipient: "acme-portal", Rows: rows()}, system("portal")))
	pkg := must(f.r.Package("w1", p.Digest, system("portal")))
	token, link, _ := f.r.Link("w1", pkg.ID, time.Hour, human("ops"))
	token2, link2, _ := f.r.Link("w1", pkg.ID, 2*time.Hour, human("ops"))
	if token == token2 || link.Digest == link2.Digest || link.Expires.Equal(link2.Expires) {
		t.Fatal("links share tokens")
	}
	got := must(f.r.Access(token, human("acme-portal")))
	if got.ID != pkg.ID || string(got.Bytes) != string(pkg.Bytes) || got.FileDigest != pkg.FileDigest {
		t.Fatal("access")
	}
	f.grant(t, Recipient{ID: "other-portal", Role: RoleCustomer, Customers: []string{"beta"}, Policy: "std"})
	expect(t, "other recipient", second(f.r.Access(token, human("other-portal"))), ReasonRecipientMismatch)
	expect(t, "other preview", second(f.r.Preview("w1", p.Digest, 1, 10, false, human("other-portal"))), ReasonRecipientMismatch)
	f.clock.advance(time.Hour - 1)
	must(f.r.Access(token, human("acme-portal")))
	f.clock.advance(1)
	expect(t, "expired", second(f.r.Access(token, human("acme-portal"))), ReasonLinkExpired)
	must(f.r.RevokeLink("w1", link2.Digest, human("ops")))
	expect(t, "revoked link", second(f.r.Access(token2, human("acme-portal"))), ReasonLinkRevoked)
	token3, _, _ := f.r.Link("w1", pkg.ID, time.Hour, human("ops"))
	f.policy(t, Policy{ID: "std", Roles: map[string][]string{RoleCustomer: {"customer", "service"}}})
	expect(t, "policy change", second(f.r.Access(token3, human("acme-portal"))), ReasonPolicySuperseded)
	p2 := must(f.r.Project("w1", Request{Policy: "std", Recipient: "acme-portal", Rows: rows()}, system("portal")))
	pkg2 := must(f.r.Package("w1", p2.Digest, system("portal")))
	token4, _, _ := f.r.Link("w1", pkg2.ID, time.Hour, human("ops"))
	f.grant(t, Recipient{ID: "acme-portal", Role: RoleCustomer, Customers: []string{"acme", "gamma"}, Policy: "std"})
	expect(t, "grant changed", second(f.r.Access(token4, human("acme-portal"))), ReasonRecipientChanged)
	expect(t, "package after change", second(f.r.Package("w1", p2.Digest, system("portal"))), ReasonRecipientChanged)
	p3 := must(f.r.Project("w1", Request{Policy: "std", Recipient: "acme-portal", Rows: rows()}, system("portal")))
	pkg3 := must(f.r.Package("w1", p3.Digest, system("portal")))
	token5, _, _ := f.r.Link("w1", pkg3.ID, time.Hour, human("ops"))
	must(f.r.RevokeRecipient("w1", "acme-portal", human("ops")))
	expect(t, "recipient revoked", second(f.r.Access(token5, human("acme-portal"))), ReasonRecipientRevoked)
	expect(t, "regrant revoked", second(f.r.Grant("w1", Recipient{ID: "acme-portal", Role: RoleCustomer, Customers: []string{"acme"}, Policy: "std"}, human("author"))), ReasonRecipientRevoked)
	expect(t, "unknown token", second(f.r.Access("nope", human("acme-portal"))), "not_found")
}

func TestReferences(t *testing.T) {
	f := newFixture(t)
	f.grant(t, Recipient{ID: "acme-api", Role: RoleCustomer, Customers: []string{"acme"}, Policy: "std"})
	a := must(f.r.Project("w1", Request{Policy: "std", Recipient: "acme-portal", Rows: rows()[:1]}, system("x")))
	b := must(f.r.Project("w1", Request{Policy: "std", Recipient: "acme-api", Rows: rows()[:1]}, system("x")))
	ra, rb := a.Rows[0][3], b.Rows[0][3]
	if ra == rb || !strings.HasPrefix(ra, "ref:") || len(ra) != 28 {
		t.Fatalf("linkable %s %s", ra, rb)
	}
	f.policy(t, Policy{ID: "std", Roles: map[string][]string{RoleCustomer: {"customer", "service", "amount", "source", "region"}}})
	c := must(f.r.Project("w1", Request{Policy: "std", Recipient: "acme-portal", Rows: rows()[:1]}, system("x")))
	if c.Rows[0][4] != ra {
		t.Fatalf("not stable across revisions %v", c.Rows[0])
	}
	if v := must(f.r.Resolve("w1", ra, human("support"))); v != "s3://bucket/acme/2026-10.csv" {
		t.Fatalf("resolve %s", v)
	}
	expect(t, "recipient resolves", second(f.r.Resolve("w1", ra, human("acme-portal"))), ReasonRecipientActor)
	other := must(NewRegistry(Config{Keys: keys, Random: &counter{}}, testLimits(), f.clock.read))
	_ = other
}

func TestChannelParityAndOrder(t *testing.T) {
	f := newFixture(t)
	in := rows()
	var digest string
	for _, channel := range []string{"portal", "api", "ai", "pdf", "mail", "erp"} {
		p := must(f.r.Project("w1", Request{Policy: "std", Recipient: "acme-portal", Rows: in}, system(channel)))
		if digest != "" && p.Digest != digest {
			t.Fatal("channel changed projection")
		}
		digest = p.Digest
		in = append([]Row{in[len(in)-1]}, in[:len(in)-1]...)
	}
	expect(t, "unrecorded projection", second(f.r.Package("w1", strings.Repeat("a", 64), system("erp"))), "not_found")
}

func TestPreviewEscapes(t *testing.T) {
	f := newFixture(t)
	evil := []Row{{"customer": "acme", "service": "<script>alert(1)</script>", "amount": "<img src=x onerror=a>", "source": ""},
		{"customer": "acme", "service": "‮evil\u0007", "amount": "&amp;\"'", "source": ""}}
	p := must(f.r.Project("w1", Request{Policy: "std", Recipient: "acme-portal", Rows: evil}, system("x")))
	out := must(f.r.Preview("w1", p.Digest, 1, 1, true, human("staff")))
	page2 := must(f.r.Preview("w1", p.Digest, 2, 1, false, human("acme-portal")))
	both := out + page2
	for _, banned := range []string{"<script", "<img", "<style", "<link", "<a ", "<iframe", "<form", "\u202e", "\u0007", "=\""} {
		if strings.Contains(both, banned) {
			t.Fatalf("preview contains %q: %s", banned, both)
		}
	}
	if !strings.Contains(out, "dropped rows: 0") || !strings.Contains(out, "rows: 2") || strings.Count(out, "<tr>") != 2 || strings.Contains(page2, "dropped") {
		t.Fatalf("preview %s", out)
	}
	for _, want := range []string{"&lt;script&gt;", "&lt;img src=x onerror=a&gt;", "\ufffdevil\ufffd", "&amp;amp;&#34;&#39;"} {
		if !strings.Contains(both, want) {
			t.Fatalf("missing %q in %s", want, both)
		}
	}
}
