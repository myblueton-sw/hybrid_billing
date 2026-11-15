package simulation

import (
	"testing"
	"time"
)

func fixedZone() *time.Location { return time.FixedZone("KST", 9*3600) }

func TestProposalRules(t *testing.T) {
	f := setup(t)
	c := Change{Kind: ChangeProfile, Profile: Profile{ID: "p1"}}
	reason(t, "assistant", second(f.r.Propose(c, bot)), ReasonAINotAllowed)
	reason(t, "system", second(f.r.Propose(c, runner)), ReasonNotHuman)
	expect(t, "actor", second(f.r.Propose(c, Actor{ID: "", Kind: KindHuman})), ErrInvalid)
	expect(t, "actor kind", second(f.r.Propose(c, Actor{ID: "x", Kind: "robot"})), ErrInvalid)
	expect(t, "form", second(f.r.Propose(Change{Kind: ChangeProfile}, ann)), ErrInvalid)
	p, err := f.r.Propose(c, ann)
	f.must(err)
	if again, _ := f.r.Propose(c, ann); again.Number != p.Number {
		t.Error("repeat not idempotent")
	}
	q, err := f.r.Propose(c, cat)
	f.must(err)
	if q.Number == p.Number {
		t.Error("another proposer shares a proposal")
	}
	reason(t, "self", second(f.r.Approve(p.Number, ann)), ReasonSelfApproval)
	reason(t, "approve assistant", second(f.r.Approve(p.Number, bot)), ReasonAINotAllowed)
	expect(t, "approve missing", second(f.r.Approve(99, bob)), ErrNotFound)
	v, err := f.r.Approve(p.Number, bob)
	f.must(err)
	if v.State != ProposalApplied || v.Approver != "bob" || v.Proposer != "ann" {
		t.Errorf("applied %+v", v)
	}
	if again, err := f.r.Approve(p.Number, bob); err != nil || again.State != ProposalApplied {
		t.Errorf("repeat approve %+v %v", again, err)
	}
	reason(t, "approve applied", second(f.r.Approve(p.Number, dan)), ReasonNotPending)
	expect(t, "recheck", second(f.r.Approve(q.Number, bob)), ErrConflict)
	reason(t, "withdraw other", second(f.r.Withdraw(q.Number, ann)), ReasonNotProposer)
	reason(t, "withdraw applied", second(f.r.Withdraw(p.Number, ann)), ReasonNotPending)
	reason(t, "withdraw system", second(f.r.Withdraw(q.Number, runner)), ReasonNotHuman)
	expect(t, "withdraw missing", second(f.r.Withdraw(99, ann)), ErrNotFound)
	w, err := f.r.Withdraw(q.Number, cat)
	if err != nil || w.State != ProposalWithdrawn {
		t.Errorf("withdraw %+v %v", w, err)
	}
	expect(t, "withdrawn read", second(f.r.Proposal(q.Number)), ErrNotFound)
	if got, err := f.r.Proposal(p.Number); err != nil || got.Change.Profile.ID != "p1" {
		t.Errorf("read %+v %v", got, err)
	}
	expect(t, "unscoped expectation", second(f.r.Propose(Change{Kind: ChangeExpectation, Expectation: expectation("nope", 1, standardChecks()...)}, ann)), ErrNotFound)
	expect(t, "unscoped cell", second(f.r.Propose(Change{Kind: ChangeCell, Cell: Cell{Profile: "nope", Event: EventRefund, Coverage: CoverageUndecided, Decision: "d"}}, ann)), ErrNotFound)
	expect(t, "unscoped proof", second(f.r.Propose(Change{Kind: ChangeProof, Proof: Proof{Target: "nope", Environment: "e", Evidence: sum("e")}}, ann)), ErrNotFound)
}

func TestProposalCopies(t *testing.T) {
	f := setup(t)
	d := public("pub")
	p, err := f.r.Propose(Change{Kind: ChangeDataset, Dataset: d}, ann)
	f.must(err)
	d.Coverage[0] = "mutated"
	p.Change.Dataset.Coverage[0] = "mutated"
	got, _ := f.r.Proposal(p.Number)
	if got.Change.Dataset.Coverage[0] != "aws" {
		t.Error("proposal shares memory")
	}
	f.must(second(f.r.Approve(p.Number, bob)))
	read, _ := f.r.Dataset("pub")
	read.Coverage[0] = "mutated"
	if again, _ := f.r.Dataset("pub"); again.Coverage[0] != "aws" {
		t.Error("dataset read shares memory")
	}
}

func TestDatasetRules(t *testing.T) {
	f := setup(t)
	f.approve(Change{Kind: ChangeDataset, Dataset: public("pub")})
	expect(t, "repeat", f.try(Change{Kind: ChangeDataset, Dataset: public("pub")}), ErrConflict)
	changed := public("pub")
	changed.Digest = sum("other")
	expect(t, "repeat changed", f.try(Change{Kind: ChangeDataset, Dataset: changed}), ErrConflict)
	expect(t, "missing parent", f.try(Change{Kind: ChangeDataset, Dataset: converted("c", "nope")}), ErrNotFound)
	f.approve(Change{Kind: ChangeDataset, Dataset: synthetic("syn")})
	reason(t, "synthetic parent", f.try(Change{Kind: ChangeDataset, Dataset: converted("c", "syn")}), ReasonWrongOrigin)
	notice := converted("c", "pub")
	notice.Notice = sum("other")
	reason(t, "notice", f.try(Change{Kind: ChangeDataset, Dataset: notice}), ReasonNoticeChanged)
	license := converted("c", "pub")
	license.License = "MIT"
	reason(t, "license", f.try(Change{Kind: ChangeDataset, Dataset: license}), ReasonNoticeChanged)
	f.approve(Change{Kind: ChangeDataset, Dataset: converted("c1", "pub")})
	f.approve(Change{Kind: ChangeDataset, Dataset: converted("c2", "c1")})
	lineage, err := f.r.Lineage("c2")
	if err != nil || len(lineage) != 3 || lineage[0].ID != "c2" || lineage[1].ID != "c1" || lineage[2].ID != "pub" {
		t.Errorf("lineage %+v %v", lineage, err)
	}
	expect(t, "lineage missing", second(f.r.Lineage("nope")), ErrNotFound)
	expect(t, "dataset missing", second(f.r.Dataset("nope")), ErrNotFound)
	p, err := f.r.Propose(Change{Kind: ChangeDataset, Dataset: converted("c3", "pub")}, cat)
	f.must(err)
	f.approve(Change{Kind: ChangeDataset, Dataset: converted("c3", "pub")})
	expect(t, "approve recheck", second(f.r.Approve(p.Number, dan)), ErrConflict)
}

func TestScenarioRules(t *testing.T) {
	f := setup(t)
	f.approve(Change{Kind: ChangeDataset, Dataset: public("pub")})
	f.approve(Change{Kind: ChangeDataset, Dataset: converted("conv", "pub")})
	f.approve(Change{Kind: ChangeDataset, Dataset: synthetic("syn")})
	expect(t, "profile missing", f.try(Change{Kind: ChangeScenario, Scenario: billing("s1")}), ErrNotFound)
	f.approve(Change{Kind: ChangeProfile, Profile: Profile{ID: "p1"}})
	missing := billing("s1")
	missing.Datasets = []string{"nope"}
	expect(t, "dataset missing", f.try(Change{Kind: ChangeScenario, Scenario: missing}), ErrNotFound)
	for _, id := range []string{"pub", "conv"} {
		s := billing("s1")
		s.Datasets = []string{id, "syn"}
		reason(t, "public answer "+id, f.try(Change{Kind: ChangeScenario, Scenario: s}), ReasonPublicAnswer)
	}
	f.approve(Change{Kind: ChangeScenario, Scenario: billing("s1")})
	expect(t, "repeat", f.try(Change{Kind: ChangeScenario, Scenario: billing("s1")}), ErrConflict)
	formatOK := format("f1")
	formatOK.Datasets = []string{"conv", "pub", "syn"}
	f.approve(Change{Kind: ChangeScenario, Scenario: formatOK})
	local := billing("s2")
	local.Start, local.Events, local.Covers = start.In(fixedZone()), billing("s1").Events, nil
	f.approve(Change{Kind: ChangeScenario, Scenario: local})
	got, err := f.r.Scenario("s2")
	if err != nil || got.Start.Location().String() != "UTC" || !got.Start.Equal(start) {
		t.Errorf("start %v %v", got.Start, err)
	}
	expect(t, "scenario missing", second(f.r.Scenario("nope")), ErrNotFound)
	if f.r.scenarios["s1"].fingerprint != f.r.scenarios["s2"].fingerprint {
		t.Error("same inputs give different fingerprints")
	}
}

func TestExpectationRules(t *testing.T) {
	f := standard(t)
	expect(t, "gap", f.try(Change{Kind: ChangeExpectation, Expectation: expectation("s1", 2, standardChecks()...)}), ErrConflict)
	reason(t, "no amount", f.try(Change{Kind: ChangeExpectation, Expectation: expectation("s1", 1, Check{Name: "lines", Kind: CheckCount, Value: "1"})}), ReasonNoAmount)
	part := amount("krw-may", "10", "KRW")
	part.Period = "2026-05"
	reason(t, "no total", f.try(Change{Kind: ChangeExpectation, Expectation: expectation("s1", 1, part)}), ReasonNoTotal)
	doc := amount("krw-inv", "10", "KRW")
	doc.Document = "inv-1"
	reason(t, "document only", f.try(Change{Kind: ChangeExpectation, Expectation: expectation("s1", 1, doc, amount("usd", "1", "USD"))}), ReasonNoTotal)
	reason(t, "format amount", f.try(Change{Kind: ChangeExpectation, Expectation: expectation("f1", 1, amount("a", "1", "KRW"))}), ReasonWrongPurpose)
	f.approve(Change{Kind: ChangeExpectation, Expectation: expectation("f1", 1, Check{Name: "rows", Kind: CheckCount, Value: "3"})})
	f.approve(Change{Kind: ChangeExpectation, Expectation: expectation("s1", 1, doc, part, amount("krw-total", "10", "KRW"))})
	expect(t, "repeat", f.try(Change{Kind: ChangeExpectation, Expectation: expectation("s1", 1, standardChecks()...)}), ErrConflict)
	expect(t, "skip", second(f.r.Propose(Change{Kind: ChangeExpectation, Expectation: expectation("s1", 3, standardChecks()...)}, ann)), ErrConflict)
	late, err := f.r.Propose(Change{Kind: ChangeExpectation, Expectation: expectation("s1", 2, standardChecks()...)}, ann)
	f.must(err)
	f.record("s1", "1.0.0", "r", "o", nil)
	reason(t, "approve after run", second(f.r.Approve(late.Number, bob)), ReasonRunsRecorded)
	reason(t, "propose after run", second(f.r.Propose(Change{Kind: ChangeExpectation, Expectation: expectation("s1", 2, standardChecks()...)}, cat)), ReasonRunsRecorded)
	copyOf := billing("s9")
	copyOf.FixtureVersion, copyOf.Covers, copyOf.Profile, copyOf.Events = "2.0.0", nil, "", billing("s1").Events
	f.approve(Change{Kind: ChangeScenario, Scenario: copyOf})
	reason(t, "fingerprint", f.try(Change{Kind: ChangeExpectation, Expectation: expectation("s9", 1, standardChecks()...)}), ReasonRunsRecorded)
	fresh := billing("s10")
	fresh.Events = sum("regenerated")
	f.approve(Change{Kind: ChangeScenario, Scenario: fresh})
	f.expect("s10", standardChecks()...)
	got, err := f.r.Expectation("s1", 1)
	if err != nil || got.Version != 1 || len(got.Checks) != 3 {
		t.Errorf("read %+v %v", got, err)
	}
	for _, v := range []int{0, 2} {
		expect(t, "version missing", second(f.r.Expectation("s1", v)), ErrNotFound)
	}
	expect(t, "scenario missing", second(f.r.Expectation("nope", 1)), ErrNotFound)
}

func TestHarnessRules(t *testing.T) {
	f := standard(t)
	expect(t, "repeat", f.try(Change{Kind: ChangeHarness, Harness: Harness{ID: "h1", Identity: "runner-9"}}), ErrConflict)
	reason(t, "identity", f.try(Change{Kind: ChangeHarness, Harness: Harness{ID: "h2", Identity: runner.ID}}), ReasonIdentityTaken)
	f.expect("s1", standardChecks()...)
	x := Run{Scenario: "s1", Engine: "1.0.0", Rules: sum("r"), Output: sum("o")}
	reason(t, "human", second(f.r.Record(x, ann)), ReasonNotHarness)
	reason(t, "assistant", second(f.r.Record(x, bot)), ReasonAINotAllowed)
	reason(t, "unbound", second(f.r.Record(x, other)), ReasonNotHarness)
	expect(t, "actor", second(f.r.Record(x, Actor{Kind: KindSystem})), ErrInvalid)
	expect(t, "form", second(f.r.Record(Run{Scenario: "s1"}, runner)), ErrInvalid)
	missing := x
	missing.Scenario = "nope"
	expect(t, "scenario", second(f.r.Record(missing, runner)), ErrNotFound)
	reason(t, "no expectation", second(f.r.Record(Run{Scenario: "f1", Engine: "1.0.0", Rules: sum("r"), Output: sum("o")}, runner)), ReasonNoExpectation)
	f.approveBy(Change{Kind: ChangeHarness, Harness: Harness{ID: "h2", Identity: other.ID}}, ann, dan)
	reason(t, "self by proposer", second(f.r.Record(x, other)), ReasonSelfVerification)
	f.approveBy(Change{Kind: ChangeHarness, Harness: Harness{ID: "h3", Identity: "runner-3"}}, cat, bob)
	reason(t, "self by approver", second(f.r.Record(x, Actor{ID: "runner-3", Kind: KindSystem})), ReasonSelfVerification)
	f.must(second(f.r.Record(x, runner)))
	reason(t, "revoke system", f.r.RevokeHarness("h1", runner), ReasonNotHuman)
	expect(t, "revoke form", f.r.RevokeHarness("", ann), ErrInvalid)
	expect(t, "revoke missing", f.r.RevokeHarness("nope", ann), ErrNotFound)
	f.must(f.r.RevokeHarness("h1", ann))
	reason(t, "revoked", second(f.r.Record(x, runner)), ReasonNotHarness)
	expect(t, "reuse id", f.try(Change{Kind: ChangeHarness, Harness: Harness{ID: "h1", Identity: "runner-8"}}), ErrConflict)
	reason(t, "reuse identity", f.try(Change{Kind: ChangeHarness, Harness: Harness{ID: "h8", Identity: runner.ID}}), ReasonIdentityTaken)
	if v, _ := f.r.Run(1); v.Verdict != VerdictRevoked || v.Key != KeySingle {
		t.Errorf("revoked run %+v", v)
	}
}
