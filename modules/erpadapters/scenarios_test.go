package erpadapters

import (
	"strings"
	"testing"
)

func TestInternalAttachmentBlocksAutomaticDelivery(t *testing.T) {
	f := newFixture(t)
	d := must(f.r.Diagnose("w1", "a1", "conn-1", Delivery{AutoMail: true, Attachments: []string{"invoice_pdf", "internal_cost"}, RecipientControl: "rc-1", AttachmentControl: "ac-1"}, human("ops")))
	if d.Result != DeliveryBlocked || strings.Join(d.Reasons, ",") != ReasonInternalAttachment {
		t.Fatalf("diagnosis %+v", d)
	}
	expect(t, "enable blocked", second(f.r.EnableDelivery("w1", "a1", "conn-1", d.Number, human("lead"))), ReasonDeliveryUnsafe)
	u := must(f.r.Diagnose("w1", "a1", "conn-1", Delivery{Portal: true, Attachments: []string{"invoice_pdf"}}, human("ops")))
	if u.Result != DeliveryBlocked || strings.Join(u.Reasons, ",") != ReasonRecipientUnproven+","+ReasonAttachmentUnproven {
		t.Fatalf("unproven %+v", u)
	}
	m := must(f.r.Diagnose("w1", "a1", "conn-1", Delivery{Attachments: []string{"internal_cost"}}, human("ops")))
	if m.Result != DeliveryManual || strings.Join(m.Reasons, ",") != ReasonInternalAttachment {
		t.Fatalf("manual warns %+v", m)
	}
	ok := must(f.r.Diagnose("w1", "a1", "conn-1", Delivery{AutoAttach: true, Attachments: []string{"invoice_pdf", "invoice_xml"}, RecipientControl: "rc-1", AttachmentControl: "ac-1"}, human("ops")))
	if ok.Result != DeliveryAllowed || len(ok.Reasons) != 0 {
		t.Fatalf("allowed %+v", ok)
	}
	expect(t, "stale", second(f.r.EnableDelivery("w1", "a1", "conn-1", m.Number, human("lead"))), ReasonStaleDiagnosis)
	expect(t, "self", second(f.r.EnableDelivery("w1", "a1", "conn-1", ok.Number, human("ops"))), ReasonSelfApproval)
	v := must(f.r.EnableDelivery("w1", "a1", "conn-1", ok.Number, human("lead")))
	if v.Enabled == nil || v.Enabled.Diagnosis != ok.Number {
		t.Fatalf("enabled %+v", v)
	}
	expect(t, "other target", second(f.r.EnableDelivery("w1", "a1", "conn-2", ok.Number, human("lead"))), "not_found")
	again := must(f.r.Diagnose("w1", "a1", "conn-1", Delivery{AutoAttach: true, Attachments: []string{"invoice_pdf", "invoice_xml"}, RecipientControl: "rc-1", AttachmentControl: "ac-1"}, human("ops")))
	if t2 := must(f.r.TargetOf("w1", "a1", "conn-1")); t2.Enabled == nil || t2.Enabled.Diagnosis != ok.Number || t2.Diagnosis.Number != again.Number {
		t.Fatalf("stays tied %+v", t2)
	}
	wider := must(f.r.Diagnose("w1", "a1", "conn-1", Delivery{AutoAttach: true, Portal: true, Attachments: []string{"invoice_pdf", "invoice_xml"}, RecipientControl: "rc-1", AttachmentControl: "ac-1"}, human("ops")))
	if t3 := must(f.r.TargetOf("w1", "a1", "conn-1")); wider.Result != DeliveryAllowed || t3.Enabled != nil {
		t.Fatalf("widened channels kept enablement %+v", t3)
	}
	must(f.r.EnableDelivery("w1", "a1", "conn-1", wider.Number, human("lead")))
	must(f.r.Diagnose("w1", "a1", "conn-1", Delivery{}, human("ops")))
	if t4 := must(f.r.TargetOf("w1", "a1", "conn-1")); t4.Enabled != nil {
		t.Fatalf("manual only kept enablement %+v", t4)
	}
	for _, change := range []func(*Delivery){
		func(d *Delivery) { d.AutoAttach = false }, func(d *Delivery) { d.AutoMail = true }, func(d *Delivery) { d.Portal = false },
		func(d *Delivery) { d.RecipientControl = "rc-9" }, func(d *Delivery) { d.AttachmentControl = "ac-9" }, func(d *Delivery) { d.Attachments = []string{"invoice_pdf"} },
	} {
		base := Delivery{AutoAttach: true, Portal: true, Attachments: []string{"invoice_pdf", "invoice_xml"}, RecipientControl: "rc-1", AttachmentControl: "ac-1"}
		d := base
		d.Attachments = append([]string(nil), base.Attachments...)
		change(&d)
		if sameDelivery(base, d) {
			t.Fatalf("same delivery ignores a field %+v", d)
		}
	}
	must(f.r.Diagnose("w1", "a1", "conn-1", Delivery{AutoMail: true, Attachments: []string{"internal_cost"}, RecipientControl: "rc", AttachmentControl: "ac"}, human("ops")))
}

func TestProductsStaySeparate(t *testing.T) {
	f := newFixture(t)
	for _, p := range []Profile{
		{ID: "s4", Vendor: "sap", Product: "s4hana", Version: "2023", Deployment: "private_cloud"},
		{ID: "ecc", Vendor: "sap", Product: "ecc", Version: "6.0", Deployment: "on_premise"},
	} {
		must(f.r.Profile("w1", p, human("arch")))
	}
	same := allMethods()
	must(f.r.Adapter("w1", Adapter{ID: "s4-a", Profile: "s4", Methods: same, MaxRows: 999, CallsPerMinute: 10}, human("arch")))
	must(f.r.Adapter("w1", Adapter{ID: "ecc-a", Profile: "ecc", Methods: same, MaxRows: 999, CallsPerMinute: 10}, human("arch")))
	f.raiseAll(t, "s4-a", LevelSandbox)
	if c := must(f.r.Certification("w1", "s4-a")); c.Level != LevelSandbox {
		t.Fatalf("s4 %+v", c)
	}
	if c := must(f.r.Certification("w1", "ecc-a")); c.Level != CertUncertified || len(c.Blocking) != 6 {
		t.Fatalf("ecc %+v", c)
	}
	expect(t, "rebind", second(f.r.Adapter("w1", Adapter{ID: "s4-a", Profile: "ecc", Methods: same, MaxRows: 999, CallsPerMinute: 10}, human("arch"))), "conflict")
	for _, bad := range []Profile{
		{ID: "x", Vendor: "sap", Product: "odoo", Version: "1", Deployment: "cloud"},
		{ID: "x", Vendor: "odoo", Product: "s4hana", Version: "1", Deployment: "cloud"},
		{ID: "x", Vendor: "sap", Product: "other", Version: "1", Deployment: "cloud"},
		{ID: "x", Vendor: "other", Product: "business_one", Version: "1", Deployment: "cloud"},
		{ID: "x", Vendor: "odoo", Product: "other", Version: "1", Deployment: "cloud"},
	} {
		expect(t, bad.Vendor+"/"+bad.Product, second(f.r.Profile("w1", bad, human("arch"))), ReasonVendorMismatch)
	}
	must(f.r.Profile("w1", Profile{ID: "b1", Vendor: "sap", Product: "business_one", Version: "10", Deployment: "cloud"}, human("arch")))
	must(f.r.Profile("w1", Profile{ID: "o", Vendor: "other", Product: "other", Version: "1", Deployment: "cloud"}, human("arch")))
	expect(t, "limit", second(f.r.Profile("w1", Profile{ID: "o2", Vendor: "other", Product: "other", Version: "1", Deployment: "cloud"}, human("arch"))), "limit")
}

func TestSimulationNeverCertifies(t *testing.T) {
	f := newFixture(t)
	for _, c := range lifecycle {
		must(f.r.Evidence("w1", "a1", c, Evidence{Level: LevelSimulated}, system("simulator")))
	}
	c := must(f.r.Certification("w1", "a1"))
	if c.Level != CertUncertified || len(c.Blocking) != len(lifecycle) {
		t.Fatalf("simulated certified %+v", c)
	}
	if caps := must(f.r.Capabilities("w1", "a1", LevelSandbox)); len(caps) != 0 {
		t.Fatalf("simulated capabilities %v", caps)
	}
	expect(t, "system sandbox", second(f.r.Evidence("w1", "a1", "post", fixtureEvidence(LevelSandbox), system("simulator"))), ReasonNotHuman)
	expect(t, "author sandbox", second(f.r.Evidence("w1", "a1", "post", fixtureEvidence(LevelSandbox), human("arch"))), ReasonSelfApproval)
	f.raiseAll(t, "a1", LevelSandbox)
	if c := must(f.r.Certification("w1", "a1")); c.Level != LevelSandbox {
		t.Fatalf("sandbox %+v", c)
	}
	f.raiseAll(t, "a1", LevelProduction)
	if c := must(f.r.Certification("w1", "a1")); c.Level != LevelProduction {
		t.Fatalf("production %+v", c)
	}
	if caps := must(f.r.Capabilities("w1", "a1", LevelProduction)); strings.Join(caps, ",") != "submit_draft,read_draft,post,get_status,correct,get_balance" {
		t.Fatalf("production caps %v", caps)
	}
	revoked := must(f.r.Revoke("w1", "a1", "post", "erp-upgrade-18", human("qb")))
	if m := revoked.Methods[5].Evidence; m.Actor != "qa" || m.RevokedBy != "qb" || !m.Revoked {
		t.Fatalf("raiser lost %+v", m)
	}
	expect(t, "revoker repeats evidence", second(f.r.Evidence("w1", "a1", "post", fixtureEvidence(LevelProduction), human("qa"))), ReasonRevoked)
	expect(t, "revoked assistant", second(f.r.Evidence("w1", "a1", "post", fixtureEvidence(LevelProduction), assistant("bot"))), ReasonAINotAllowed)
	c = must(f.r.Certification("w1", "a1"))
	if c.Level != CertUncertified || strings.Join(c.Blocking, ",") != "post" {
		t.Fatalf("revoked %+v", c)
	}
	if caps := must(f.r.Capabilities("w1", "a1", LevelSandbox)); strings.Contains(strings.Join(caps, ","), "post") {
		t.Fatalf("revoked capability %v", caps)
	}
	expect(t, "raise revoked", second(f.r.Evidence("w1", "a1", "post", Evidence{Level: LevelProduction, Reference: "again"}, human("qb"))), ReasonRevoked)
	gaps := must(f.r.Admits("w1", "a1", LevelSandbox, []string{"prepare", "post", "get_status"}))
	if strings.Join(gaps, ",") != "prepare,post" {
		t.Fatalf("admits %v", gaps)
	}
}

func TestPlanNeedsProvenRowLimit(t *testing.T) {
	f := newFixture(t)
	expect(t, "declared", second(f.r.Plan("w1", "a1", 1200)), ReasonLimitUnproven)
	must(f.r.LimitEvidence("w1", "a1", LimitRows, Evidence{Level: LevelSimulated}, system("sim")))
	expect(t, "simulated", second(f.r.Plan("w1", "a1", 1200)), ReasonLimitUnproven)
	must(f.r.LimitEvidence("w1", "a1", LimitRows, fixtureEvidence(LevelSandbox), human("qa")))
	chunks := must(f.r.Plan("w1", "a1", 1200))
	if len(chunks) != 3 || chunks[0] != (Chunk{1, 500}) || chunks[2] != (Chunk{1001, 1200}) {
		t.Fatalf("chunks %+v", chunks)
	}
	if one := must(f.r.Plan("w1", "a1", 500)); len(one) != 1 || one[0] != (Chunk{1, 500}) {
		t.Fatalf("exact %+v", one)
	}
	must(f.r.Revoke("w1", "a1", LimitRows, "measured-lower", human("qa")))
	expect(t, "revoked limit", second(f.r.Plan("w1", "a1", 10)), ReasonLimitUnproven)
}
