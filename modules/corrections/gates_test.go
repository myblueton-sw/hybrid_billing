package corrections

import (
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
)

func TestRegistryNeedsDependenciesAndLimits(t *testing.T) {
	f := open(t, testLimits())
	if _, err := NewRegistry(Config{}, testLimits(), f.clock.read); !errors.Is(err, ErrInvalid) {
		t.Fatal("documents")
	}
	if _, err := NewRegistry(Config{Documents: f.docs}, testLimits(), nil); !errors.Is(err, ErrInvalid) {
		t.Fatal("clock")
	}
	for _, l := range []Limits{{0, 1, 1}, {1, 0, 1}, {1, 1, 0}, {maxWorkspaces + 1, 1, 1}, {1, maxCorrections + 1, 1}, {1, 1, maxRecords + 1}} {
		if _, err := NewRegistry(Config{Documents: f.docs}, l, f.clock.read); !errors.Is(err, ErrInvalid) {
			t.Fatal(l)
		}
	}
	if _, err := NewRegistry(Config{Documents: f.docs}, Limits{maxWorkspaces, maxCorrections, maxRecords}, f.clock.read); err != nil {
		t.Fatal(err)
	}
	if (Refusal{Reason: "x"}).Error() != ErrRejected.Error()+": x" || !errors.Is(Refusal{Reason: "x"}, ErrRejected) {
		t.Fatal("refusal")
	}
}

func TestFormsAndHelpers(t *testing.T) {
	if !validID(strings.Repeat("z", maxIDBytes)) || validID(strings.Repeat("z", maxIDBytes+1)) || validID("") || validID("a b") || validID("!a") || !validID("Az09._:-") {
		t.Fatal("identifiers")
	}
	for _, text := range []string{"0", "1", "10.5", "0.01", strings.Repeat("9", maxDigits)} {
		if !canonical(text) {
			t.Fatal(text)
		}
	}
	for _, text := range []string{"", ".", "1.", ".5", "01", "1.50", "-1", "+1", "1a", "1.a", strings.Repeat("9", maxDigits+1), "9.:"} {
		if canonical(text) {
			t.Fatal(text)
		}
	}
	for value, want := range map[int64]string{0: "0", -1: "-0.01", 150: "1.5", -3950: "-39.5", 100: "1", 7: "0.07"} {
		if got := format(new(big.Int).SetInt64(value), 2); got != want {
			t.Fatalf("%d: %s", value, got)
		}
	}
	if got := format(new(big.Int).SetInt64(12), 0); got != "12" {
		t.Fatal(got)
	}
	for _, d := range []string{"2026-02-30", "2026-1-01", "x", ""} {
		if validDate(d) {
			t.Fatal(d)
		}
	}
	base := request("c", "o", "k", "1")
	for _, change := range []func(*Correction){
		func(c *Correction) { c.ID = "" }, func(c *Correction) { c.Original = "" }, func(c *Correction) { c.Key = "" }, func(c *Correction) { c.Target = "-1" },
		func(c *Correction) { c.Reason = "bored" }, func(c *Correction) { c.Evidence = "" }, func(c *Correction) { c.Period.From = "x" },
		func(c *Correction) { c.Period.To = "x" }, func(c *Correction) { c.Period.From = "2026-10-01" },
	} {
		c := base
		change(&c)
		if form(c) {
			t.Fatalf("form %+v", c)
		}
	}
	same := base
	same.Period = Period{From: "2026-09-30", To: "2026-09-30"}
	if !form(base) || !form(same) {
		t.Fatal("valid forms")
	}
	for _, r := range []string{ReasonLateSource, ReasonRetroactive, ReasonQuantityError, ReasonPriceError, ReasonOther} {
		c := base
		c.Reason = r
		if !form(c) {
			t.Fatal(r)
		}
	}
}

func TestRequestGates(t *testing.T) {
	f := open(t, testLimits())
	f.issue(t, "ws", draft("o", "inv", "100", ""))
	expect(t, "workspace", second(f.r.Request("", request("c", "o", "k", "1"), human("requester"))), "invalid")
	expect(t, "actor", second(f.r.Request("ws", request("c", "o", "k", "1"), Actor{ID: "x"})), "invalid")
	expect(t, "form", second(f.r.Request("ws", request("", "o", "k", "1"), human("requester"))), "invalid")
	expect(t, "assistant", second(f.r.Request("ws", request("c", "o", "k", "1"), assistant("bot"))), RefusalAINotAllowed)
	expect(t, "system", second(f.r.Request("ws", request("c", "o", "k", "1"), system("job"))), RefusalNotHuman)
	expect(t, "unknown original", second(f.r.Request("ws", request("c", "nope", "k", "1"), human("requester"))), "not_found")
	expect(t, "other workspace", second(f.r.Request("ws2", request("c", "o", "k", "1"), human("requester"))), "not_found")
	f.docs.failDoc = "o"
	expect(t, "documents down", second(f.r.Request("ws", request("c", "o", "k", "1"), human("requester"))), RefusalUnavailable)
	f.docs.failDoc = ""
	f.docs.failCredit = true
	expect(t, "credits down", second(f.r.Request("ws", request("c", "o", "k", "1"), human("requester"))), RefusalUnavailable)
	f.docs.failCredit = false
	f.credit(t, "cr", "1", "o")
	expect(t, "credit as original", second(f.r.Request("ws", request("c", "cr", "k", "0"), human("requester"))), RefusalNotOriginal)
	must(f.r.Explain("ws", "o", "cr", Explanation{Reason: ReasonOther, Evidence: "e"}, human("auditor")))
	v := must(f.r.Request("ws", request("c", "o", "k", "1"), human("requester")))
	if must(f.r.Request("ws", request("c", "o", "k", "1"), human("requester"))).Digest != v.Digest {
		t.Fatal("repeat")
	}
	expect(t, "other requester", second(f.r.Request("ws", request("c", "o", "k", "1"), human("other"))), "conflict")
	expect(t, "other content", second(f.r.Request("ws", request("c", "o", "k", "2"), human("requester"))), "conflict")
	l := testLimits()
	l.Corrections, l.Workspaces = 1, 1
	g := open(t, l)
	g.issue(t, "ws", draft("o", "inv", "100", ""))
	g.issue(t, "ws", draft("p", "inv", "100", ""))
	g.issue(t, "ws2", draft("o", "inv", "100", ""))
	must(g.r.Request("ws", request("c", "o", "k", "1"), human("requester")))
	expect(t, "corrections", second(g.r.Request("ws", request("d", "p", "k", "1"), human("requester"))), "limit")
	expect(t, "workspaces", second(g.r.Request("ws2", request("c", "o", "k", "1"), human("requester"))), "limit")
}

func TestOriginalSnapshotChecks(t *testing.T) {
	f := open(t, testLimits())
	f.issue(t, "ws", draft("o", "inv", "100", ""))
	d := must(f.is.DocumentOf("ws", "o"))
	d.Snapshot.Workspace = "ws2"
	f.docs.swap["o"] = d
	expect(t, "foreign snapshot", second(f.r.EffectiveOf("ws", "o")), RefusalUnavailable)
	d.Snapshot.Workspace, d.Snapshot.Draft.Total = "ws", "100.0"
	f.docs.swap["o"] = d
	expect(t, "non-canonical total", second(f.r.EffectiveOf("ws", "o")), RefusalUnavailable)
	d.Snapshot.Draft.Total = "1.001"
	f.docs.swap["o"] = d
	expect(t, "total beyond scale", second(f.r.EffectiveOf("ws", "o")), RefusalUnavailable)
	delete(f.docs.swap, "o")
	f.credit(t, "cr", "1", "o")
	c := must(f.is.DocumentOf("ws", "cr"))
	c.Snapshot.Draft.Total = "0.001"
	f.docs.swap["cr"] = c
	expect(t, "credit beyond scale", second(f.r.EffectiveOf("ws", "o")), RefusalUnavailable)
	c.Snapshot.Draft.Total = "-5"
	f.docs.swap["cr"] = c
	expect(t, "signed credit total", second(f.r.EffectiveOf("ws", "o")), RefusalUnavailable)
	c.Snapshot.Draft.Total, c.Snapshot.Draft.Original = "1", "other"
	f.docs.swap["cr"] = c
	expect(t, "credit of another original", second(f.r.EffectiveOf("ws", "o")), RefusalUnavailable)
	delete(f.docs.swap, "cr")
	f.docs.failDoc = "cr"
	expect(t, "credit lookup", second(f.r.EffectiveOf("ws", "o")), RefusalUnavailable)
	f.docs.failDoc = ""
	expect(t, "effective form", second(f.r.EffectiveOf("", "o")), "invalid")
	expect(t, "effective unknown", second(f.r.EffectiveOf("ws", "nope")), "not_found")
	f.docs.failCredit = true
	expect(t, "effective credits", second(f.r.EffectiveOf("ws", "o")), RefusalUnavailable)
}

func TestApproveAndWithdrawGates(t *testing.T) {
	f := open(t, testLimits())
	f.issue(t, "ws", draft("o", "inv", "100", ""))
	v := must(f.r.Request("ws", request("c", "o", "k", "50"), human("requester")))
	expect(t, "form", second(f.r.Approve("ws", "c", "short", human("checker"))), "invalid")
	expect(t, "form workspace", second(f.r.Approve("", "c", v.Digest, human("checker"))), "invalid")
	expect(t, "form id", second(f.r.Approve("ws", "", v.Digest, human("checker"))), "invalid")
	expect(t, "form actor", second(f.r.Approve("ws", "c", v.Digest, Actor{ID: "x"})), "invalid")
	expect(t, "assistant", second(f.r.Approve("ws", "c", v.Digest, assistant("bot"))), RefusalAINotAllowed)
	expect(t, "unknown", second(f.r.Approve("ws", "nope", v.Digest, human("checker"))), "not_found")
	expect(t, "unknown workspace", second(f.r.Approve("ws2", "c", v.Digest, human("checker"))), "not_found")
	expect(t, "digest", second(f.r.Approve("ws", "c", strings.Repeat("a", 64), human("checker"))), RefusalDigestChanged)
	f.docs.failCredit = true
	expect(t, "credits down", second(f.r.Approve("ws", "c", v.Digest, human("checker"))), RefusalUnavailable)
	f.docs.failCredit = false
	must(f.r.Approve("ws", "c", v.Digest, human("checker")))
	if must(f.r.Approve("ws", "c", v.Digest, human("checker"))).Approver != "checker" {
		t.Fatal("repeat")
	}
	expect(t, "second approver", second(f.r.Approve("ws", "c", v.Digest, human("other"))), "conflict")
	expect(t, "approved other digest", second(f.r.Approve("ws", "c", strings.Repeat("a", 64), human("checker"))), RefusalNotRequested)
	expect(t, "withdraw form", second(f.r.Withdraw("ws", "", human("requester"))), "invalid")
	expect(t, "withdraw assistant", second(f.r.Withdraw("ws", "c", assistant("bot"))), RefusalAINotAllowed)
	expect(t, "withdraw system", second(f.r.Withdraw("ws", "c", system("job"))), RefusalNotHuman)
	expect(t, "withdraw unknown", second(f.r.Withdraw("ws", "nope", human("requester"))), "not_found")
	f.docs.failCredit = true
	expect(t, "withdraw credits down", second(f.r.Withdraw("ws", "c", human("requester"))), RefusalUnavailable)
	f.docs.failCredit = false
	w := must(f.r.Withdraw("ws", "c", human("requester")))
	if w.State != StateWithdrawn || w.Closer != "requester" {
		t.Fatal(w)
	}
	if must(f.r.Withdraw("ws", "c", human("requester"))).State != StateWithdrawn {
		t.Fatal("repeat withdraw")
	}
	expect(t, "other withdrawer", second(f.r.Withdraw("ws", "c", human("other"))), "conflict")
	expect(t, "approve withdrawn", second(f.r.Approve("ws", "c", v.Digest, human("checker"))), RefusalNotRequested)
	expect(t, "apply withdrawn", second(f.r.Apply("ws", "c", "x", system("linker"))), RefusalNotApproved)
	f.approved(t, request("d", "o", "k2", "50"))
	f.credit(t, "cr", "50", "o")
	must(f.r.Apply("ws", "d", "cr", system("linker")))
	expect(t, "withdraw applied", second(f.r.Withdraw("ws", "d", human("requester"))), RefusalClosed)
	must(f.r.Request("ws", request("e", "o", "k3", "10"), human("requester")))
	f.credit(t, "x", "1", "o")
	if must(f.r.Withdraw("ws", "e", human("requester"))).State != StateWithdrawn {
		t.Fatal("a requested correction is always withdrawable")
	}
}

func TestApplyGates(t *testing.T) {
	f := open(t, testLimits())
	f.issue(t, "ws", draft("o", "inv", "100", ""))
	f.approved(t, request("c", "o", "k", "60"))
	expect(t, "form", second(f.r.Apply("ws", "c", "", system("linker"))), "invalid")
	expect(t, "form workspace", second(f.r.Apply("", "c", "cr", system("linker"))), "invalid")
	expect(t, "form id", second(f.r.Apply("ws", "", "cr", system("linker"))), "invalid")
	expect(t, "form actor", second(f.r.Apply("ws", "c", "cr", Actor{ID: "x"})), "invalid")
	expect(t, "assistant", second(f.r.Apply("ws", "c", "cr", assistant("bot"))), RefusalAINotAllowed)
	expect(t, "unknown", second(f.r.Apply("ws", "nope", "cr", system("linker"))), "not_found")
	expect(t, "no credit yet", second(f.r.Apply("ws", "c", "cr", system("linker"))), RefusalBaseChanged)
	f.credit(t, "cr", "40", "o")
	f.docs.failCredit = true
	expect(t, "credits down", second(f.r.Apply("ws", "c", "cr", system("linker"))), RefusalUnavailable)
	f.docs.failCredit = false
	expect(t, "other document named", second(f.r.Apply("ws", "c", "o", system("linker"))), RefusalBaseChanged)
	f.docs.failDoc = "cr"
	expect(t, "document down", second(f.r.Apply("ws", "c", "cr", system("linker"))), RefusalUnavailable)
	f.docs.failDoc = ""
	v := must(f.r.Apply("ws", "c", "cr", human("linker")))
	if must(f.r.Apply("ws", "c", "cr", human("linker"))).Document != v.Document {
		t.Fatal("repeat")
	}
	expect(t, "other actor", second(f.r.Apply("ws", "c", "cr", system("linker2"))), RefusalClosed)
	expect(t, "same id other kind", second(f.r.Apply("ws", "c", "cr", system("linker"))), RefusalClosed)
	expect(t, "other document", second(f.r.Apply("ws", "c", "cr2", human("linker"))), RefusalClosed)
	expect(t, "of form", second(f.r.Of("", "c")), "invalid")
	expect(t, "of unknown", second(f.r.Of("ws", "nope")), "not_found")
	f.approved(t, request("c2", "o", "k2", "10"))
	f.credit(t, "cr2", "50", "o")
	expect(t, "already linked", second(f.r.Apply("ws", "c2", "cr", system("linker"))), RefusalLinked)
}

func TestExplainGates(t *testing.T) {
	f := open(t, testLimits())
	f.issue(t, "ws", draft("o", "inv", "100", ""))
	e := Explanation{Reason: ReasonOther, Evidence: "memo"}
	for label, call := range map[string]func() error{
		"workspace": func() error { return second(f.r.Explain("", "o", "x", e, human("a"))) },
		"original":  func() error { return second(f.r.Explain("ws", "", "x", e, human("a"))) },
		"credit":    func() error { return second(f.r.Explain("ws", "o", "", e, human("a"))) },
		"reason": func() error {
			return second(f.r.Explain("ws", "o", "x", Explanation{Reason: "x", Evidence: "m"}, human("a")))
		},
		"evidence": func() error { return second(f.r.Explain("ws", "o", "x", Explanation{Reason: ReasonOther}, human("a"))) },
		"actor":    func() error { return second(f.r.Explain("ws", "o", "x", e, Actor{ID: "a"})) },
	} {
		expect(t, label, call(), "invalid")
	}
	expect(t, "assistant", second(f.r.Explain("ws", "o", "x", e, assistant("bot"))), RefusalAINotAllowed)
	expect(t, "system", second(f.r.Explain("ws", "o", "x", e, system("job"))), RefusalNotHuman)
	expect(t, "unknown original", second(f.r.Explain("ws", "nope", "x", e, human("a"))), "not_found")
	expect(t, "not unlinked", second(f.r.Explain("ws", "o", "x", e, human("a"))), RefusalNotUnlinked)
	f.credit(t, "cr", "1", "o")
	f.docs.failCredit = true
	expect(t, "credits down", second(f.r.Explain("ws", "o", "cr", e, human("a"))), RefusalUnavailable)
	f.docs.failCredit = false
	n := must(f.r.Explain("ws", "o", "cr", e, human("a")))
	if must(f.r.Explain("ws", "o", "cr", e, human("a"))) != n {
		t.Fatal("repeat")
	}
	expect(t, "other explainer", second(f.r.Explain("ws", "o", "cr", e, human("b"))), "conflict")
	expect(t, "other explanation", second(f.r.Explain("ws", "o", "cr", Explanation{Reason: ReasonOther, Evidence: "x"}, human("a"))), "conflict")
	f.approved(t, request("c", "o", "k", "50"))
	f.credit(t, "y", "49", "o")
	must(f.r.Apply("ws", "c", "y", system("linker")))
	expect(t, "linked by correction", second(f.r.Explain("ws", "o", "y", e, human("a"))), RefusalLinked)
	sys := draft("sysc", "crn", "1", "o")
	v := must(f.is.Submit("ws", sys, issuance.Actor{ID: "clerk", Kind: issuance.KindHuman}))
	must(f.is.Approve("ws", "sysc", v.Digest, issuance.Actor{ID: "lead", Kind: issuance.KindHuman}))
	must(f.is.Issue("ws", "sysc", issuance.Actor{ID: "auditor", Kind: issuance.KindSystem}))
	must(f.r.Explain("ws", "o", "sysc", e, human("auditor")))
	l := testLimits()
	l.Records = 1
	g := open(t, l)
	g.issue(t, "ws", draft("o", "inv", "100", ""))
	g.credit(t, "a1", "1", "o")
	g.credit(t, "a2", "1", "o")
	must(g.r.Explain("ws", "o", "a1", e, human("a")))
	expect(t, "records", second(g.r.Explain("ws", "o", "a2", e, human("a"))), "limit")
	expect(t, "records final", second(g.r.RecordFinal("ws", "o", Final{Reference: "r", Amount: "1", Observed: start}, system("erp"))), "limit")
	l.Workspaces = 1
	h := open(t, l)
	h.issue(t, "ws", draft("o", "inv", "100", ""))
	h.issue(t, "ws2", draft("o", "inv", "100", ""))
	h.issue(t, "ws2", draft("cr", "crn", "1", "o"))
	must(h.r.RecordFinal("ws", "o", Final{Reference: "r", Amount: "1", Observed: start}, system("erp")))
	expect(t, "workspace limit explain", second(h.r.Explain("ws2", "o", "cr", e, human("a"))), "limit")
	expect(t, "workspace limit final", second(h.r.RecordFinal("ws2", "o", Final{Reference: "r", Amount: "1", Observed: start}, system("erp"))), "limit")
}

func TestFinalGates(t *testing.T) {
	f := open(t, testLimits())
	f.issue(t, "ws", draft("o", "inv", "100", ""))
	good := Final{Reference: "r", Amount: "100", Observed: start}
	for label, call := range map[string]func() error{
		"workspace": func() error { return second(f.r.RecordFinal("", "o", good, system("erp"))) },
		"original":  func() error { return second(f.r.RecordFinal("ws", "", good, system("erp"))) },
		"reference": func() error {
			return second(f.r.RecordFinal("ws", "o", Final{Amount: "1", Observed: start}, system("erp")))
		},
		"amount": func() error {
			return second(f.r.RecordFinal("ws", "o", Final{Reference: "r", Amount: "-1", Observed: start}, system("erp")))
		},
		"observed": func() error {
			return second(f.r.RecordFinal("ws", "o", Final{Reference: "r", Amount: "1"}, system("erp")))
		},
		"actor": func() error { return second(f.r.RecordFinal("ws", "o", good, Actor{ID: "x"})) },
	} {
		expect(t, label, call(), "invalid")
	}
	expect(t, "assistant", second(f.r.RecordFinal("ws", "o", good, assistant("bot"))), RefusalAINotAllowed)
	expect(t, "unknown", second(f.r.RecordFinal("ws", "nope", good, system("erp"))), "not_found")
	f.docs.failCredit = true
	expect(t, "credits down", second(f.r.RecordFinal("ws", "o", good, system("erp"))), RefusalUnavailable)
	f.docs.failCredit = false
	local := Final{Reference: "r", Amount: "100", Observed: start.In(time.FixedZone("x", 3600))}
	must(f.r.RecordFinal("ws", "o", local, system("erp")))
	if e := must(f.r.EffectiveOf("ws", "o")); e.Finals[0].Final.Observed.Location() != time.UTC {
		t.Fatal("observation zone")
	}
}

func TestFinalsCountTowardTheRecordLimit(t *testing.T) {
	if !validID("Z") || !validID("ZZ") {
		t.Fatal("upper-case Z")
	}
	l := testLimits()
	l.Records = 1
	f := open(t, l)
	f.issue(t, "ws", draft("o", "inv", "100", ""))
	must(f.r.RecordFinal("ws", "o", Final{Reference: "r1", Amount: "100", Observed: start}, system("erp")))
	expect(t, "records", second(f.r.RecordFinal("ws", "o", Final{Reference: "r2", Amount: "100", Observed: start}, system("erp"))), "limit")
}
