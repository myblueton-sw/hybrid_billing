package erpadapters

import (
	"errors"
	"math/rand"
	"strings"
	"testing"
	"time"
)

func TestRegistryForms(t *testing.T) {
	clock := &testClock{now: start}
	must(NewRegistry(Limits{1, 1, 1, 1}, clock.read))
	must(NewRegistry(Limits{100000, 10000, 10000, 100000}, clock.read))
	for _, l := range []Limits{{0, 1, 1, 1}, {1, 0, 1, 1}, {1, 1, 0, 1}, {1, 1, 1, 0}, {100001, 1, 1, 1}, {1, 10001, 1, 1}, {1, 1, 10001, 1}, {1, 1, 1, 100001}} {
		if _, err := NewRegistry(l, clock.read); err != ErrInvalid {
			t.Fatalf("%+v: %v", l, err)
		}
	}
	if _, err := NewRegistry(Limits{1, 1, 1, 1}, nil); err != ErrInvalid {
		t.Fatal(err)
	}
	err := refuse(ReasonRevoked)
	if !errors.Is(err, ErrRejected) || err.Error() != ErrRejected.Error()+": revoked" {
		t.Fatal(err)
	}
	if !validID(strings.Repeat("a", 128)) || validID(strings.Repeat("a", 129)) || validID("") {
		t.Fatal("id length")
	}
	for _, id := range []string{"A", "Z", "a", "z", "0", "9", ".", "_", ":", "-"} {
		if !validID(id) {
			t.Fatalf("%q", id)
		}
	}
	for _, id := range []string{"@", "[", "`", "{", "/", ";", " a", "!a"} {
		if validID(id) {
			t.Fatalf("%q", id)
		}
	}
}

func TestProfileGates(t *testing.T) {
	f := newFixture(t)
	good := Profile{ID: "p", Vendor: "sap", Product: "ecc", Version: "6.0", Deployment: "on_premise", Modules: []string{"sd", "fi"}}
	bad := []func(*Profile){
		func(p *Profile) { p.ID = "" }, func(p *Profile) { p.Vendor = "oracle" }, func(p *Profile) { p.Product = "netsuite" },
		func(p *Profile) { p.Version = "" }, func(p *Profile) { p.Deployment = "edge" }, func(p *Profile) { p.Modules = []string{"a", "a"} },
		func(p *Profile) { p.Modules = []string{""} },
		func(p *Profile) {
			p.Modules = nil
			for i := 0; i < 33; i++ {
				p.Modules = append(p.Modules, "m"+strings.Repeat("x", i+1))
			}
		},
	}
	for i, change := range bad {
		p := good
		change(&p)
		if got := kind(second(f.r.Profile("w1", p, human("arch")))); got != "invalid" {
			t.Fatalf("case %d: %s", i, got)
		}
	}
	thirty := good
	thirty.ID = "p32"
	for i := 0; i < 32; i++ {
		thirty.Modules = append(thirty.Modules[:0:0], thirty.Modules...)
	}
	thirty.Modules = nil
	for i := 0; i < 32; i++ {
		thirty.Modules = append(thirty.Modules, "m"+strings.Repeat("x", i+1))
	}
	must(f.r.Profile("w1", thirty, human("arch")))
	expect(t, "ws", second(f.r.Profile("", good, human("arch"))), "invalid")
	expect(t, "actor", second(f.r.Profile("w1", good, Actor{ID: "a", Kind: "robot"})), "invalid")
	expect(t, "assistant", second(f.r.Profile("w1", good, assistant("bot"))), ReasonAINotAllowed)
	expect(t, "system", second(f.r.Profile("w1", good, system("job"))), ReasonNotHuman)
	v := must(f.r.Profile("w1", good, human("arch")))
	v.Profile.Modules[0] = "x"
	if again := must(f.r.Profile("w1", good, human("arch"))); again.Profile.Modules[0] != "sd" {
		t.Fatal("aliasing")
	}
	expect(t, "other author", second(f.r.Profile("w1", good, human("b"))), "conflict")
	for _, change := range []func(*Profile){
		func(p *Profile) { p.Product = "s4hana" }, func(p *Profile) { p.Version = "7" }, func(p *Profile) { p.Deployment = "cloud" },
		func(p *Profile) { p.Modules = []string{"sd"} }, func(p *Profile) { p.Vendor, p.Product = "odoo", "odoo" },
	} {
		p := good
		change(&p)
		expect(t, "changed", second(f.r.Profile("w1", p, human("arch"))), "conflict")
	}
	must(f.r.Profile("w2", good, human("arch")))
	must(f.r.Profile("w3", good, human("arch")))
	expect(t, "workspaces", second(f.r.Profile("w4", good, human("arch"))), "limit")
	expect(t, "profile of", second(f.r.ProfileOf("w1", "nope")), "not_found")
	expect(t, "profile of ws", second(f.r.ProfileOf("w9", "p")), "not_found")
	if got := must(f.r.ProfileOf("w1", "p")); got.Author != "arch" {
		t.Fatal("profile of")
	}
}

func TestAdapterGates(t *testing.T) {
	f := newFixture(t)
	good := odooAdapter("a2")
	bad := []func(*Adapter){
		func(a *Adapter) { a.ID = "" }, func(a *Adapter) { a.Profile = "" }, func(a *Adapter) { a.Methods = nil },
		func(a *Adapter) { a.MaxRows = 0 }, func(a *Adapter) { a.MaxRows = 1000001 }, func(a *Adapter) { a.CallsPerMinute = 0 },
		func(a *Adapter) { a.CallsPerMinute = 1000001 },
		func(a *Adapter) { a.Methods = []Method{{Command: "delete", Name: "x", Transport: "api"}} },
		func(a *Adapter) {
			a.Methods = []Method{{Command: "post", Name: "x", Transport: "api"}, {Command: "post", Name: "y", Transport: "api"}}
		},
		func(a *Adapter) { a.Methods = []Method{{Command: "post", Name: "", Transport: "api"}} },
		func(a *Adapter) { a.Methods = []Method{{Command: "post", Name: "x", Transport: "ftp"}} },
	}
	for i, change := range bad {
		a := good
		change(&a)
		if got := kind(second(f.r.Adapter("w1", a, human("arch")))); got != "invalid" {
			t.Fatalf("case %d: %s", i, got)
		}
	}
	edge := Adapter{ID: "edge", Profile: "odoo17", Methods: []Method{{Command: "post", Name: "x", Transport: "file"}}, MaxRows: 1000000, CallsPerMinute: 1000000}
	must(f.r.Adapter("w1", edge, human("arch")))
	low := Adapter{ID: "low", Profile: "odoo17", Methods: []Method{{Command: "post", Name: "x", Transport: "file"}}, MaxRows: 1, CallsPerMinute: 1}
	must(f.r.Adapter("w1", low, human("arch")))
	expect(t, "ws", second(f.r.Adapter("", good, human("arch"))), "invalid")
	expect(t, "actor", second(f.r.Adapter("w1", good, human(""))), "invalid")
	expect(t, "unknown profile", second(f.r.Adapter("w1", Adapter{ID: "x", Profile: "nope", Methods: allMethods(), MaxRows: 1, CallsPerMinute: 1}, human("arch"))), ReasonUnknownProfile)
	expect(t, "unknown ws", second(f.r.Adapter("w9", good, human("arch"))), ReasonUnknownProfile)
	expect(t, "assistant", second(f.r.Adapter("w1", good, assistant("bot"))), ReasonAINotAllowed)
	db := good
	db.Methods = []Method{{Command: "post", Name: "x", Transport: "api"}, {Command: "correct", Name: "sql", Transport: "database"}}
	expect(t, "database", second(f.r.Adapter("w1", db, human("arch"))), ReasonDirectDatabase)
	v := must(f.r.Adapter("w1", good, human("arch")))
	if len(v.Methods) != 9 || v.Methods[0].Evidence.Evidence.Level != LevelDeclared || v.Rows.Evidence.Level != LevelDeclared || v.Calls.Evidence.Level != LevelDeclared {
		t.Fatalf("adapter %+v", v)
	}
	v.Adapter.Methods[0].Name = "x"
	v.Methods[0].Method.Name = "x"
	got := must(f.r.Adapter("w1", good, human("arch")))
	if got.Adapter.Methods[0].Name != "odoo.connect" || got.Methods[0].Method.Name != "odoo.connect" {
		t.Fatal("aliasing")
	}
	for _, change := range []func(*Adapter){
		func(a *Adapter) { a.MaxRows = 1 }, func(a *Adapter) { a.CallsPerMinute = 1 }, func(a *Adapter) { a.Methods = a.Methods[:3] },
		func(a *Adapter) { a.Methods = append([]Method(nil), a.Methods...); a.Methods[0].Name = "y" },
	} {
		a := good
		change(&a)
		expect(t, "changed", second(f.r.Adapter("w1", a, human("arch"))), "conflict")
	}
	expect(t, "other author", second(f.r.Adapter("w1", good, human("b"))), "conflict")
	must(f.r.Adapter("w1", odooAdapter("a3"), human("arch")))
	expect(t, "limit", second(f.r.Adapter("w1", odooAdapter("a4"), human("arch"))), "limit")
	expect(t, "adapter of", second(f.r.AdapterOf("w1", "nope")), "not_found")
	expect(t, "adapter of ws", second(f.r.AdapterOf("w9", "a1")), "not_found")
	if got := must(f.r.AdapterOf("w1", "a1")); got.Author != "arch" {
		t.Fatal("adapter of")
	}
}

func TestEvidenceGates(t *testing.T) {
	f := newFixture(t)
	for i, e := range []Evidence{{Level: "gold", Reference: "r"}, {Level: LevelDeclared, Reference: "r"}, {Level: LevelSandbox}, {Level: LevelSimulated, Reference: "a b"}} {
		if got := kind(second(f.r.Evidence("w1", "a1", "post", e, human("qa")))); got != "invalid" {
			t.Fatalf("case %d: %s", i, got)
		}
	}
	e := fixtureEvidence(LevelSandbox)
	expect(t, "command", second(f.r.Evidence("w1", "a1", "delete", e, human("qa"))), "invalid")
	expect(t, "ws", second(f.r.Evidence("", "a1", "post", e, human("qa"))), "invalid")
	expect(t, "id", second(f.r.Evidence("w1", "", "post", e, human("qa"))), "invalid")
	expect(t, "actor", second(f.r.Evidence("w1", "a1", "post", e, human(""))), "invalid")
	expect(t, "missing", second(f.r.Evidence("w1", "nope", "post", e, human("qa"))), "not_found")
	expect(t, "assistant", second(f.r.Evidence("w1", "a1", "post", e, assistant("bot"))), ReasonAINotAllowed)
	must(f.r.Adapter("w1", Adapter{ID: "narrow", Profile: "odoo17", Methods: []Method{{Command: "post", Name: "p", Transport: "api"}}, MaxRows: 1, CallsPerMinute: 1}, human("arch")))
	expect(t, "not mapped", second(f.r.Evidence("w1", "narrow", "correct", e, human("qa"))), ReasonNotMapped)
	expect(t, "not mapped assistant", second(f.r.Evidence("w1", "narrow", "correct", e, assistant("bot"))), ReasonAINotAllowed)
	sim := must(f.r.Evidence("w1", "a1", "post", Evidence{Level: LevelSimulated}, human("qa")))
	if sim.Methods[5].Evidence.Evidence.Level != LevelSimulated {
		t.Fatalf("human simulated %+v", sim.Methods[5])
	}
	must(f.r.Evidence("w1", "a1", "post", Evidence{Level: LevelSimulated}, human("qa")))
	expect(t, "same level other", second(f.r.Evidence("w1", "a1", "post", Evidence{Level: LevelSimulated}, human("qb"))), ReasonLevelDown)
	f.clock.advance(time.Minute)
	jump := must(f.r.Evidence("w1", "a1", "get_status", fixtureEvidence(LevelProduction), human("qa")))
	if jump.Methods[6].Evidence.Evidence.Level != LevelProduction || !jump.Methods[6].Evidence.At.Equal(start.Add(time.Minute)) {
		t.Fatalf("skip %+v", jump.Methods[6])
	}
	expect(t, "down", second(f.r.Evidence("w1", "a1", "get_status", fixtureEvidence(LevelSandbox), human("qa"))), ReasonLevelDown)
	must(f.r.Evidence("w1", "a1", "connect", Evidence{Level: LevelSimulated}, system("sim")))
	if again := must(f.r.Evidence("w1", "a1", "connect", Evidence{Level: LevelSimulated}, system("sim"))); again.Methods[0].Evidence.Actor != "sim" {
		t.Fatal("system repeat")
	}
	expect(t, "system other", second(f.r.Evidence("w1", "a1", "connect", Evidence{Level: LevelSimulated}, system("sim2"))), ReasonLevelDown)
	expect(t, "limit form", second(f.r.LimitEvidence("w1", "a1", "max_latency", e, human("qa"))), "invalid")
	expect(t, "limit ev form", second(f.r.LimitEvidence("w1", "a1", LimitRows, Evidence{Level: LevelSandbox}, human("qa"))), "invalid")
	expect(t, "limit ws", second(f.r.LimitEvidence("", "a1", LimitRows, e, human("qa"))), "invalid")
	expect(t, "limit id", second(f.r.LimitEvidence("w1", "", LimitRows, e, human("qa"))), "invalid")
	expect(t, "limit actor", second(f.r.LimitEvidence("w1", "a1", LimitRows, e, human(""))), "invalid")
	expect(t, "limit missing", second(f.r.LimitEvidence("w1", "nope", LimitRows, e, human("qa"))), "not_found")
	expect(t, "limit self", second(f.r.LimitEvidence("w1", "a1", LimitCalls, e, human("arch"))), ReasonSelfApproval)
	c := must(f.r.LimitEvidence("w1", "a1", LimitCalls, e, human("qa")))
	if c.Calls.Evidence.Level != LevelSandbox || c.Rows.Evidence.Level != LevelDeclared {
		t.Fatalf("calls only %+v", c)
	}
	for _, x := range []string{"", "a b"} {
		expect(t, "revoke form", second(f.r.Revoke("w1", "a1", x, "r", human("qa"))), "invalid")
		expect(t, "revoke reason", second(f.r.Revoke("w1", "a1", "post", x, human("qa"))), "invalid")
	}
	expect(t, "revoke ws", second(f.r.Revoke("", "a1", "post", "r", human("qa"))), "invalid")
	expect(t, "revoke id", second(f.r.Revoke("w1", "", "post", "r", human("qa"))), "invalid")
	expect(t, "revoke actor", second(f.r.Revoke("w1", "a1", "post", "r", human(""))), "invalid")
	expect(t, "revoke missing", second(f.r.Revoke("w1", "nope", "post", "r", human("qa"))), "not_found")
	expect(t, "revoke unmapped", second(f.r.Revoke("w1", "narrow", "correct", "r", human("qa"))), ReasonNotMapped)
	expect(t, "revoke assistant", second(f.r.Revoke("w1", "a1", "post", "r", assistant("bot"))), ReasonAINotAllowed)
	expect(t, "revoke system", second(f.r.Revoke("w1", "a1", "post", "r", system("job"))), ReasonNotHuman)
	expect(t, "revoke self", second(f.r.Revoke("w1", "a1", "post", "r", human("arch"))), ReasonSelfApproval)
	rv := must(f.r.Revoke("w1", "a1", LimitCalls, "slower", human("qa")))
	if !rv.Calls.Revoked || rv.Calls.Reason != "slower" || rv.Calls.Evidence.Level != LevelSandbox {
		t.Fatalf("revoked %+v", rv.Calls)
	}
	must(f.r.Revoke("w1", "a1", LimitCalls, "slower", human("qa")))
	expect(t, "revoke again", second(f.r.Revoke("w1", "a1", LimitCalls, "other", human("qa"))), ReasonRevoked)
	expect(t, "raise revoked limit", second(f.r.LimitEvidence("w1", "a1", LimitCalls, fixtureEvidence(LevelProduction), human("qb"))), ReasonRevoked)
	expect(t, "cert missing", second(f.r.Certification("w1", "nope")), "not_found")
	if cert := must(f.r.Certification("w1", "narrow")); cert.Level != CertUncertified || len(cert.Blocking) != 6 {
		t.Fatalf("narrow %+v", cert)
	}
	expect(t, "caps form", second(f.r.Capabilities("w1", "a1", LevelSimulated)), "invalid")
	expect(t, "caps missing", second(f.r.Capabilities("w1", "nope", LevelSandbox)), "not_found")
	if caps := must(f.r.Capabilities("w1", "a1", LevelSandbox)); strings.Join(caps, ",") != "get_status" {
		t.Fatalf("caps %v", caps)
	}
	expect(t, "admits form", second(f.r.Admits("w1", "a1", LevelDeclared, nil)), "invalid")
	expect(t, "admits missing", second(f.r.Admits("w1", "nope", LevelSandbox, nil)), "not_found")
	if gaps := must(f.r.Admits("w1", "a1", LevelProduction, []string{"get_status", "post", "unknown"})); strings.Join(gaps, ",") != "post,unknown" {
		t.Fatalf("admits %v", gaps)
	}
}

func TestDeliveryGates(t *testing.T) {
	f := newFixture(t)
	for i, d := range []Delivery{{Attachments: []string{"zip"}}, {Attachments: []string{"invoice_pdf", "invoice_pdf"}}, {RecipientControl: "a b"}, {AttachmentControl: "a b"}} {
		if got := kind(second(f.r.Diagnose("w1", "a1", "t", d, human("ops")))); got != "invalid" {
			t.Fatalf("case %d: %s", i, got)
		}
	}
	d := Delivery{}
	expect(t, "target", second(f.r.Diagnose("w1", "a1", "", d, human("ops"))), "invalid")
	expect(t, "ws", second(f.r.Diagnose("", "a1", "t", d, human("ops"))), "invalid")
	expect(t, "id", second(f.r.Diagnose("w1", "", "t", d, human("ops"))), "invalid")
	expect(t, "actor", second(f.r.Diagnose("w1", "a1", "t", d, human(""))), "invalid")
	expect(t, "missing", second(f.r.Diagnose("w1", "nope", "t", d, human("ops"))), "not_found")
	expect(t, "assistant", second(f.r.Diagnose("w1", "a1", "t", d, assistant("bot"))), ReasonAINotAllowed)
	expect(t, "system", second(f.r.Diagnose("w1", "a1", "t", d, system("job"))), ReasonNotHuman)
	clean := must(f.r.Diagnose("w1", "a1", "t", Delivery{Attachments: []string{"statement"}}, human("ops")))
	if clean.Result != DeliveryManual || len(clean.Reasons) != 0 || clean.Number != 1 || clean.Target != "t" {
		t.Fatalf("manual %+v", clean)
	}
	for _, x := range []Delivery{{AutoAttach: true}, {AutoMail: true}, {Portal: true}} {
		x.RecipientControl, x.AttachmentControl = "rc", "ac"
		if got := must(f.r.Diagnose("w1", "a1", "t", x, human("ops"))); got.Result != DeliveryAllowed {
			t.Fatalf("channel %+v", got)
		}
	}
	only := must(f.r.Diagnose("w1", "a1", "t", Delivery{AutoMail: true, RecipientControl: "rc"}, human("ops")))
	if strings.Join(only.Reasons, ",") != ReasonAttachmentUnproven {
		t.Fatalf("attachment only %+v", only)
	}
	allowed := must(f.r.Diagnose("w1", "a1", "t", Delivery{AutoMail: true, RecipientControl: "rc", AttachmentControl: "ac", Attachments: []string{"invoice_pdf"}}, human("ops")))
	allowed.Delivery.Attachments[0] = "x"
	allowed.Reasons = append(allowed.Reasons, "x")
	if got := must(f.r.TargetOf("w1", "a1", "t")); got.Diagnosis.Delivery.Attachments[0] != "invoice_pdf" || len(got.Diagnosis.Reasons) != 0 {
		t.Fatal("diagnosis aliasing")
	}
	expect(t, "enable form", second(f.r.EnableDelivery("w1", "a1", "t", 0, human("lead"))), "invalid")
	expect(t, "enable target", second(f.r.EnableDelivery("w1", "a1", "", 1, human("lead"))), "invalid")
	expect(t, "enable ws", second(f.r.EnableDelivery("", "a1", "t", 1, human("lead"))), "invalid")
	expect(t, "enable id", second(f.r.EnableDelivery("w1", "", "t", 1, human("lead"))), "invalid")
	expect(t, "enable actor", second(f.r.EnableDelivery("w1", "a1", "t", 1, human(""))), "invalid")
	expect(t, "enable missing", second(f.r.EnableDelivery("w1", "nope", "t", 1, human("lead"))), "not_found")
	expect(t, "enable assistant", second(f.r.EnableDelivery("w1", "a1", "t", allowed.Number, assistant("bot"))), ReasonAINotAllowed)
	v := must(f.r.EnableDelivery("w1", "a1", "t", allowed.Number, human("lead")))
	v.Enabled.Actor = "x"
	v.Diagnosis.Reasons = append(v.Diagnosis.Reasons, "x")
	got := must(f.r.EnableDelivery("w1", "a1", "t", allowed.Number, human("lead")))
	if got.Enabled.Actor != "lead" || len(got.Diagnosis.Reasons) != 0 {
		t.Fatal("enable repeat or aliasing")
	}
	expect(t, "enable other", second(f.r.EnableDelivery("w1", "a1", "t", allowed.Number, human("lead2"))), "conflict")
	expect(t, "target of", second(f.r.TargetOf("w1", "a1", "nope")), "not_found")
	expect(t, "target of adapter", second(f.r.TargetOf("w1", "nope", "t")), "not_found")
	for i := 0; i < 2; i++ {
		must(f.r.Diagnose("w1", "a1", "u", Delivery{}, human("ops")))
	}
	expect(t, "diagnoses limit", second(f.r.Diagnose("w1", "a1", "u", Delivery{}, human("ops"))), "limit")
}

func TestPlanGates(t *testing.T) {
	f := newFixture(t)
	expect(t, "zero", second(f.r.Plan("w1", "a1", 0)), "invalid")
	expect(t, "too many", second(f.r.Plan("w1", "a1", 100000001)), "invalid")
	expect(t, "ws", second(f.r.Plan("", "a1", 1)), "invalid")
	expect(t, "id", second(f.r.Plan("w1", "", 1)), "invalid")
	expect(t, "missing", second(f.r.Plan("w1", "nope", 1)), "not_found")
	must(f.r.Adapter("w1", Adapter{ID: "one", Profile: "odoo17", Methods: allMethods(), MaxRows: 1000, CallsPerMinute: 1}, human("arch")))
	must(f.r.LimitEvidence("w1", "one", LimitRows, fixtureEvidence(LevelSandbox), human("qa")))
	if chunks := must(f.r.Plan("w1", "one", 100000000)); len(chunks) != 100000 || chunks[99999] != (Chunk{99999001, 100000000}) {
		t.Fatalf("max %d", len(chunks))
	}
	must(f.r.Adapter("w1", Adapter{ID: "tiny", Profile: "odoo17", Methods: allMethods(), MaxRows: 999, CallsPerMinute: 1}, human("arch")))
	must(f.r.LimitEvidence("w1", "tiny", LimitRows, fixtureEvidence(LevelSandbox), human("qa")))
	expect(t, "documents", second(f.r.Plan("w1", "tiny", 100000000)), "limit")
}

func TestPropertyPlan(t *testing.T) {
	f := newFixture(t)
	rng := rand.New(rand.NewSource(7))
	for i := 0; i < 300; i++ {
		clock := &testClock{now: start}
		r := must(NewRegistry(Limits{1, 1, 1, 1}, clock.read))
		size := 1 + rng.Intn(50)
		must(r.Profile("w", Profile{ID: "p", Vendor: "odoo", Product: "odoo", Version: "1", Deployment: "cloud"}, human("a")))
		must(r.Adapter("w", Adapter{ID: "a", Profile: "p", Methods: allMethods(), MaxRows: size, CallsPerMinute: 1}, human("a")))
		must(r.LimitEvidence("w", "a", LimitRows, fixtureEvidence(LevelSandbox), human("b")))
		rows := 1 + rng.Intn(500)
		chunks := must(r.Plan("w", "a", rows))
		covered := 0
		for j, c := range chunks {
			if c.First != covered+1 || c.Last < c.First || c.Last-c.First+1 > size || j < len(chunks)-1 && c.Last-c.First+1 != size {
				t.Fatalf("rows %d size %d chunk %d %+v", rows, size, j, c)
			}
			covered = c.Last
		}
		if covered != rows || len(chunks) != (rows+size-1)/size {
			t.Fatalf("rows %d size %d covered %d in %d", rows, size, covered, len(chunks))
		}
	}
	_ = f
}

func TestMutationGuards(t *testing.T) {
	f := newFixture(t)
	f.raiseAll(t, "a1", LevelSandbox)
	must(f.r.Revoke("w1", "a1", "correct", "r", human("qa")))
	if c := must(f.r.Certification("w1", "a1")); c.Level != CertUncertified || strings.Join(c.Blocking, ",") != "correct" {
		t.Fatalf("blocking %+v", c)
	}
	must(f.r.Adapter("w1", Adapter{ID: "edge", Profile: "odoo17", Methods: allMethods(), MaxRows: 999, CallsPerMinute: 1}, human("arch")))
	must(f.r.LimitEvidence("w1", "edge", LimitRows, fixtureEvidence(LevelSandbox), human("qa")))
	expect(t, "one past", second(f.r.Plan("w1", "edge", 99900001)), "limit")
	if chunks := must(f.r.Plan("w1", "edge", 99900000)); len(chunks) != 100000 {
		t.Fatalf("at bound %d", len(chunks))
	}
}
