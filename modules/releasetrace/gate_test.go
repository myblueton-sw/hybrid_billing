package releasetrace

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestActivation(t *testing.T) {
	f := standard(t)
	reason(t, "no reconciliation", f.activate("p1", "1.0.0"), ReasonReconciliationMissing)
	f.reconcile("p1", "2026-08", true)
	reason(t, "one period", f.activate("p1", "1.0.0"), ReasonReconciliationMissing)
	f.reconcile("p1", "2026-10", true)
	reason(t, "not consecutive", f.activate("p1", "1.0.0"), ReasonReconciliationMissing)
	f.reconcile("p1", "2026-09", true)
	reason(t, "unknown build", f.activate("p1", "9.9.9"), ReasonUnknownBuild)
	expect(t, "missing path", f.activate("nope", "1.0.0"), ErrNotFound)
	reason(t, "harness binder approves", f.tryBy(Change{Kind: ChangeActivate, Activate: Activate{Path: "p1", Build: "1.0.0", Proof: sum("p")}}, gus, cat), ReasonSelfVerification)
	reason(t, "harness binder proposes", f.tryBy(Change{Kind: ChangeActivate, Activate: Activate{Path: "p1", Build: "1.0.0", Proof: sum("p")}}, dan, hal), ReasonSelfVerification)
	reason(t, "reconciler approves", f.tryBy(Change{Kind: ChangeActivate, Activate: Activate{Path: "p1", Build: "1.0.0", Proof: sum("p")}}, gus, fay), ReasonSelfVerification)
	f.must(f.activate("p1", "1.0.0"))
	v, err := f.r.Path("p1")
	if err != nil || v.State != PathActive || v.Build != "1.0.0" || len(v.Reconciliations) != 3 || v.Reconciliations[2].Period != "2026-10" {
		t.Errorf("active %+v %v", v, err)
	}
	reason(t, "deactivate system", f.r.Deactivate("p1", "x", runner), ReasonNotHuman)
	expect(t, "deactivate form", f.r.Deactivate("p1", "", ann), ErrInvalid)
	expect(t, "deactivate missing", f.r.Deactivate("nope", "x", ann), ErrNotFound)
	f.must(f.r.Deactivate("p1", "incident", ann))
	if v, _ := f.r.Path("p1"); v.State != PathDesigned || v.Proof != "" || v.Build != "" {
		t.Error("deactivated")
	}
	expect(t, "path missing", second(f.r.Path("nope")), ErrNotFound)
	g := standard(t)
	g.reconcile("p1", "2026-09", true)
	g.reconcile("p1", "2026-10", true)
	g.approve(Change{Kind: ChangeRequirement, Requirement: req("R2", false, TestRef{ID: "u1", Category: CategoryAmount})})
	g.approve(Change{Kind: ChangePath, Path: Path{ID: "p2", Provider: "aws", Contract: "msp", Role: "payer", Version: "v1", Issuance: IssuanceAPI, Requirements: []Key{k("R2")}}})
	g.reconcile("p2", "2026-09", true)
	g.reconcile("p2", "2026-10", true)
	reason(t, "category from untested", g.activate("p2", "1.0.0"), ReasonCategoryMissing)
	g.approve(Change{Kind: ChangeRequirement, Requirement: req("R3", false, allCategories("w")...)})
	g.approve(Change{Kind: ChangePath, Path: Path{ID: "p3", Provider: "aws", Contract: "msp", Role: "payer", Version: "v1", Issuance: IssuanceAPI, Requirements: []Key{k("R1"), k("R3")}}})
	g.reconcile("p3", "2026-09", true)
	g.reconcile("p3", "2026-10", true)
	reason(t, "requirement not tested", g.activate("p3", "1.0.0"), ReasonRequirementNotTested)
	h := standard(t)
	h.reconcile("p1", "2026-06", true)
	h.reconcile("p1", "2026-07", true)
	reason(t, "stale", h.activate("p1", "1.0.0"), ReasonReconciliationMissing)
	h.reconcile("p1", "2026-08", true)
	h.must(h.activate("p1", "1.0.0"))
	expect(t, "repeat period", h.tryBy(Change{Kind: ChangeReconcile, Reconcile: Reconcile{Path: "p1", Period: "2026-08", Evidence: sum("e"), Matched: true}}, eve, fay), ErrConflict)
	reason(t, "current month", h.tryBy(Change{Kind: ChangeReconcile, Reconcile: Reconcile{Path: "p1", Period: "2026-11", Evidence: sum("e"), Matched: true}}, eve, fay), ReasonFuturePeriod)
	expect(t, "reconcile missing", h.tryBy(Change{Kind: ChangeReconcile, Reconcile: Reconcile{Path: "nope", Period: "2026-01", Evidence: sum("e"), Matched: true}}, eve, fay), ErrNotFound)
	for i := 0; i < 120; i++ {
		year, month := 2010+i/12, i%12+1
		period := itoa(year) + "-" + map[bool]string{true: "0", false: ""}[month < 10] + itoa(month)
		if period >= "2026-06" {
			break
		}
		h.reconcile("p1", period, true)
	}
	if v, _ := h.r.Path("p1"); len(v.Reconciliations) != 120 || v.Reconciliations[119].Period != "2026-08" || v.Reconciliations[0].Period != "2010-04" {
		t.Errorf("periods %d", len(v.Reconciliations))
	}
	for _, x := range []Reconcile{{Path: "p1", Period: "2026-01", Evidence: sum("e"), Matched: true, Differences: 1}, {Path: "p1", Period: "2026-01", Evidence: sum("e")},
		{Path: "p1", Period: "2026-13", Evidence: sum("e"), Matched: true}, {Path: "p1", Period: "26-01", Evidence: sum("e"), Matched: true}} {
		expect(t, "reconcile form", second(h.r.Propose(Change{Kind: ChangeReconcile, Reconcile: x}, eve)), ErrInvalid)
	}
}

func TestGate(t *testing.T) {
	f := standard(t)
	f.reconcile("p1", "2026-09", true)
	f.reconcile("p1", "2026-10", true)
	f.must(f.activate("p1", "1.0.0"))
	rel := Release{Version: "r1", Build: "1.0.0", Requirements: []Key{k("R1")}, Paths: []string{"p1"}}
	g, err := f.r.Gate(rel)
	if err != nil || !g.Ready || len(g.Templates) != 1 || g.Templates[0] != "std-xlsx" || g.States[k("R1")] != StateTested {
		t.Fatalf("ready %+v %v", g, err)
	}
	f.approve(Change{Kind: ChangeRequirement, Requirement: req("C2", true, TestRef{ID: "c2"})})
	b := req("B1", false, TestRef{ID: "b1"})
	b.Namespace, b.Blocking = "other", true
	f.approve(Change{Kind: ChangeRequirement, Requirement: b})
	g, _ = f.r.Gate(rel)
	want := []Problem{{ProblemBlocking, "other/B1"}, {ProblemCommonMissing, "v1/C2"}}
	if g.Ready || len(g.Problems) != 2 || g.Problems[0] != want[0] || g.Problems[1] != want[1] {
		t.Errorf("common and blocking %+v", g.Problems)
	}
	f.approve(Change{Kind: ChangeRetire, Retire: Retire{Key: Key{"other", "B1"}, Reason: "path not pursued"}})
	g, _ = f.r.Gate(rel)
	if len(g.RetiredBlocking) != 1 || g.RetiredBlocking[0] != (Key{"other", "B1"}) || len(g.Problems) != 1 {
		t.Errorf("retired blocking %+v", g)
	}
	reason(t, "common without replacement", f.try(Change{Kind: ChangeRetire, Retire: Retire{Key: k("C2"), Reason: "x"}}), ReasonCommonNeedsReplacement)
	bad := Release{Version: "r2", Build: "2.0.0", Requirements: []Key{k("C2"), k("R1"), k("R9")}, Paths: []string{"nope", "p1"}}
	g, _ = f.r.Gate(bad)
	got := map[string]bool{}
	for _, p := range g.Problems {
		got[p.Problem+" "+p.Subject] = true
	}
	for _, w := range []string{"not_tested v1/C2", "not_tested v1/R1", "unknown v1/R9", "unknown nope", "path_build p1"} {
		if !got[w] {
			t.Errorf("missing %s in %+v", w, g.Problems)
		}
	}
	if g.Ready || len(g.Templates) != 0 {
		t.Error("templates on problems")
	}
	f.approve(Change{Kind: ChangeRequirement, Requirement: req("R4", false, TestRef{ID: "r4"})})
	f.approve(Change{Kind: ChangePath, Path: Path{ID: "p2", Provider: "aws", Contract: "msp", Role: "payer", Version: "v1", Issuance: IssuanceAPI, Template: "erp-x", Requirements: []Key{k("R4")}}})
	g, _ = f.r.Gate(Release{Version: "r3", Build: "1.0.0", Requirements: []Key{k("C2"), k("R1")}, Paths: []string{"p1", "p2"}})
	for _, w := range []string{"path_inactive p2", "path_requirement p2>v1/R4"} {
		found := false
		for _, p := range g.Problems {
			found = found || p.Problem+" "+p.Subject == w
		}
		if !found {
			t.Errorf("missing %s", w)
		}
	}
	if len(g.Templates) != 1 || g.Templates[0] != "std-xlsx" {
		t.Errorf("inactive template shown %+v", g.Templates)
	}
	f.reconcile("p1", "2026-08", false)
	f.reconcile("p1", "2026-07", true)
	if g, _ := f.r.Gate(rel); hasProblem(g, ProblemPathReconcile) {
		t.Error("older unmatched period")
	}
	h := standard(t)
	h.reconcile("p1", "2026-09", true)
	h.reconcile("p1", "2026-10", true)
	h.must(h.activate("p1", "1.0.0"))
	h.mu.Lock()
	*h.now = time.Date(2026, 12, 2, 9, 0, 0, 0, time.UTC)
	h.mu.Unlock()
	h.reconcile("p1", "2026-11", false)
	if g, _ := h.r.Gate(rel); !hasProblem(g, ProblemPathReconcile) {
		t.Error("latest unmatched period")
	}
	m := setup(t)
	m.approve(Change{Kind: ChangeRequirement, Requirement: req("X", false, TestRef{ID: "x"})})
	m.approve(Change{Kind: ChangePath, Path: Path{ID: "p", Provider: "a", Contract: "b", Role: "c", Version: "d", Issuance: IssuanceFile, Requirements: []Key{k("X")}}})
	if g, _ := m.r.Gate(Release{Version: "r", Build: "1.0.0", Requirements: []Key{k("X")}, Paths: []string{"p"}}); !hasProblem(g, ProblemCommonUnregistered) {
		t.Error("common unregistered")
	}
	for _, x := range []Release{{Version: "", Build: "1.0.0", Requirements: []Key{k("X")}, Paths: []string{"p"}}, {Version: "r", Build: "1", Requirements: []Key{k("X")}, Paths: []string{"p"}},
		{Version: "r", Build: "1.0.0", Paths: []string{"p"}}, {Version: "r", Build: "1.0.0", Requirements: []Key{k("X")}},
		{Version: "r", Build: "1.0.0", Requirements: []Key{k("X"), k("X")}, Paths: []string{"p"}}, {Version: "r", Build: "1.0.0", Requirements: []Key{k("X")}, Paths: []string{"p", "p"}}} {
		expect(t, "gate form", second(m.r.Gate(x)), ErrInvalid)
	}
}

func hasProblem(g GateView, problem string) bool {
	for _, p := range g.Problems {
		if p.Problem == problem {
			return true
		}
	}
	return false
}

func TestRetirement(t *testing.T) {
	f := standard(t)
	f.approve(Change{Kind: ChangeRequirement, Requirement: req("C2", true, TestRef{ID: "c2"})})
	f.approve(Change{Kind: ChangeRequirement, Requirement: req("N1", false, TestRef{ID: "n1"})})
	reason(t, "weaker", f.try(Change{Kind: ChangeRetire, Retire: Retire{Key: k("C2"), Replacement: k("N1"), Reason: "x"}}), ReasonWeakerReplacement)
	reason(t, "older replacement", f.try(Change{Kind: ChangeRetire, Retire: Retire{Key: k("C2"), Replacement: k("R1"), Reason: "x"}}), ReasonWeakerReplacement)
	f.approve(Change{Kind: ChangeRequirement, Requirement: req("C3", true, TestRef{ID: "c3"})})
	f.approve(Change{Kind: ChangeRetire, Retire: Retire{Key: k("C2"), Replacement: k("C3"), Reason: "renamed"}})
	reason(t, "retired twice", f.try(Change{Kind: ChangeRetire, Retire: Retire{Key: k("C2"), Replacement: k("C3"), Reason: "x"}}), ReasonRetired)
	f.approve(Change{Kind: ChangeRequirement, Requirement: req("C4", true, TestRef{ID: "c4"})})
	reason(t, "replacement used", f.try(Change{Kind: ChangeRetire, Retire: Retire{Key: k("R1"), Replacement: k("C3"), Reason: "x"}}), ReasonWeakerReplacement)
	f.approve(Change{Kind: ChangeRequirement, Requirement: req("C5", true, TestRef{ID: "c5"})})
	f.approve(Change{Kind: ChangeRetire, Retire: Retire{Key: k("C4"), Replacement: k("C5"), Reason: "x"}})
	reason(t, "retired replacement", f.try(Change{Kind: ChangeRetire, Retire: Retire{Key: k("C3"), Replacement: k("C4"), Reason: "x"}}), ReasonWeakerReplacement)
	b := req("B1", false, TestRef{ID: "b1"})
	b.Blocking = true
	f.approve(Change{Kind: ChangeRequirement, Requirement: b})
	f.approve(Change{Kind: ChangeRequirement, Requirement: req("N2", false, TestRef{ID: "n2"})})
	reason(t, "blocking needs blocking", f.try(Change{Kind: ChangeRetire, Retire: Retire{Key: k("B1"), Replacement: k("N2"), Reason: "x"}}), ReasonWeakerReplacement)
	expect(t, "missing", f.try(Change{Kind: ChangeRetire, Retire: Retire{Key: k("Z"), Reason: "x"}}), ErrNotFound)
	expect(t, "missing replacement", f.try(Change{Kind: ChangeRetire, Retire: Retire{Key: k("N2"), Replacement: k("Z"), Reason: "x"}}), ErrNotFound)
	reason(t, "implement retired", f.try(Change{Kind: ChangeImplemented, Implemented: Implemented{Key: k("C2"), Commit: sum("c")[:40]}}), ReasonRetired)
	expect(t, "implement missing", f.try(Change{Kind: ChangeImplemented, Implemented: Implemented{Key: k("Z"), Commit: sum("c")[:40]}}), ErrNotFound)
	g, _ := f.r.Gate(Release{Version: "r", Build: "1.0.0", Requirements: []Key{k("C2"), k("R1")}, Paths: []string{"p1"}})
	if !hasProblem(g, ProblemRetired) || hasProblem(g, ProblemCommonUnregistered) {
		t.Errorf("retired %+v", g.Problems)
	}
	for _, p := range g.Problems {
		if p.Problem == ProblemCommonMissing && p.Subject == "v1/C2" {
			t.Error("retired common still required")
		}
	}
	if v, _ := f.r.Requirement(k("C2"), "1.0.0"); !v.Retired || v.Replacement != k("C3") {
		t.Errorf("retired view %+v", v)
	}
}

func TestFormsAndProposals(t *testing.T) {
	expect(t, "clock", second(NewRegistry(nil)), ErrInvalid)
	if !errors.Is(Refusal{Reason: "x"}, ErrRejected) || (Refusal{Reason: "x"}).Error() != ErrRejected.Error()+": x" {
		t.Error("refusal")
	}
	good := req("R1", true, TestRef{ID: "t1", Category: CategoryAmount}, TestRef{ID: "t2"})
	if !good.valid() {
		t.Fatal("fixture")
	}
	bad := map[string]func(*Requirement){
		"ns": func(q *Requirement) { q.Namespace = "" }, "id": func(q *Requirement) { q.ID = "" }, "source": func(q *Requirement) { q.Source = "" },
		"owner": func(q *Requirement) { q.Owner = "" }, "design": func(q *Requirement) { q.Design = "" }, "tests": func(q *Requirement) { q.Tests = nil },
		"order": func(q *Requirement) { q.Tests[0], q.Tests[1] = q.Tests[1], q.Tests[0] }, "dup": func(q *Requirement) { q.Tests[1].ID = q.Tests[0].ID },
		"cat": func(q *Requirement) { q.Tests[0].Category = "speed" }, "test id": func(q *Requirement) { q.Tests[0].ID = "a b" },
		"iface": func(q *Requirement) { q.Interfaces = []string{"b", "a"} },
		"many tests": func(q *Requirement) {
			q.Tests = nil
			for i := 0; i < 257; i++ {
				q.Tests = append(q.Tests, TestRef{ID: "t" + itoa(1000+i)})
			}
		},
		"many ifaces": func(q *Requirement) {
			q.Interfaces = nil
			for i := 0; i < 65; i++ {
				q.Interfaces = append(q.Interfaces, "i"+itoa(1000+i))
			}
		},
	}
	for name, mutate := range bad {
		q := req("R1", true, TestRef{ID: "t1", Category: CategoryAmount}, TestRef{ID: "t2"})
		mutate(&q)
		if q.valid() {
			t.Errorf("%s accepted", name)
		}
	}
	p := Path{ID: "p", Provider: "a", Contract: "b", Role: "c", Version: "d", Issuance: IssuanceFile, Requirements: []Key{k("R1")}}
	if !p.valid() {
		t.Fatal("path fixture")
	}
	for name, mutate := range map[string]func(*Path){
		"id": func(p *Path) { p.ID = "" }, "provider": func(p *Path) { p.Provider = "" }, "contract": func(p *Path) { p.Contract = "" },
		"role": func(p *Path) { p.Role = "" }, "version": func(p *Path) { p.Version = "" }, "issuance": func(p *Path) { p.Issuance = "fax" },
		"template": func(p *Path) { p.Template = "a b" }, "reqs": func(p *Path) { p.Requirements = nil },
		"req order": func(p *Path) { p.Requirements = []Key{k("B"), k("A")} }, "req ns": func(p *Path) { p.Requirements = []Key{{ID: "A"}} },
	} {
		x := p
		mutate(&x)
		if x.valid() {
			t.Errorf("path %s accepted", name)
		}
	}
	for _, c := range []Change{{Kind: "other"}, {Kind: ChangeRetire, Retire: Retire{Key: k("A"), Replacement: k("A"), Reason: "x"}}, {Kind: ChangeRetire, Retire: Retire{Key: k("A")}},
		{Kind: ChangeImplemented, Implemented: Implemented{Key: k("A"), Commit: "x"}}, {Kind: ChangeBuild, Build: Build{Build: "x", Commit: sum("c")[:40]}},
		{Kind: ChangeHarness, Harness: Harness{ID: "h"}}, {Kind: ChangeProof, Proof: Proof{Key: k("A"), Build: "1.0.0", Environment: "e", Customer: "c", Period: "2026", Evidence: sum("e")}},
		{Kind: ChangeActivate, Activate: Activate{Path: "p", Build: "1.0.0"}}, {Kind: ChangePath}, {Kind: ChangeRequirement}} {
		if _, err := clean(c); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s accepted", c.Kind)
		}
	}
	for _, period := range []string{"2026-00", "2026-13", "2026-1a", "x026-01", "2026/01", "2026-20"} {
		if _, ok := month(period); ok {
			t.Errorf("period %s", period)
		}
	}
	if m, ok := month("2026-12"); !ok || m != 2026*12+11 {
		t.Error("month")
	}
	if !validText(strings.Repeat("a", 200)) || validText(strings.Repeat("a", 201)) || validText("a\x01") || validText("a\u0080") || !validText("a ") || validText("\xff") {
		t.Error("text")
	}
	for _, id := range []string{"A", "Z", "a", "z", "0", "9", strings.Repeat("a", 128)} {
		if !validID(id) {
			t.Errorf("id %q", id)
		}
	}
	for _, id := range []string{"", "!a", "a b", "a@", "a[", "a`", "a{", strings.Repeat("a", 129)} {
		if validID(id) {
			t.Errorf("id %q accepted", id)
		}
	}
	if hexOf("gbcd", 4) || !hexOf("abcd", 4) || hexOf("abc", 4) {
		t.Error("hex")
	}
	f := setup(t)
	c := Change{Kind: ChangeRequirement, Requirement: good}
	reason(t, "assistant", second(f.r.Propose(c, bot)), ReasonAINotAllowed)
	reason(t, "system", second(f.r.Propose(c, runner)), ReasonNotHuman)
	expect(t, "actor", second(f.r.Propose(c, Actor{Kind: KindHuman})), ErrInvalid)
	expect(t, "kind", second(f.r.Propose(c, Actor{ID: "x", Kind: "robot"})), ErrInvalid)
	pr, err := f.r.Propose(c, ann)
	f.must(err)
	if again, _ := f.r.Propose(c, ann); again.Number != pr.Number {
		t.Error("repeat")
	}
	q, _ := f.r.Propose(c, cat)
	reason(t, "self", second(f.r.Approve(pr.Number, ann)), ReasonSelfApproval)
	reason(t, "approve system", second(f.r.Approve(pr.Number, runner)), ReasonNotHuman)
	expect(t, "missing", second(f.r.Approve(99, bob)), ErrNotFound)
	f.must(second(f.r.Approve(pr.Number, bob)))
	if again, err := f.r.Approve(pr.Number, bob); err != nil || again.State != ProposalApplied {
		t.Error("repeat approve")
	}
	reason(t, "applied", second(f.r.Approve(pr.Number, cat)), ReasonNotPending)
	expect(t, "recheck", second(f.r.Approve(q.Number, bob)), ErrConflict)
	reason(t, "withdraw other", second(f.r.Withdraw(q.Number, ann)), ReasonNotProposer)
	reason(t, "withdraw applied", second(f.r.Withdraw(pr.Number, ann)), ReasonNotPending)
	reason(t, "withdraw assistant", second(f.r.Withdraw(q.Number, bot)), ReasonAINotAllowed)
	expect(t, "withdraw missing", second(f.r.Withdraw(99, ann)), ErrNotFound)
	if w, err := f.r.Withdraw(q.Number, cat); err != nil || w.State != ProposalWithdrawn {
		t.Error("withdraw")
	}
	expect(t, "withdrawn", second(f.r.Proposal(q.Number)), ErrNotFound)
	pp, _ := f.r.Propose(Change{Kind: ChangePath, Path: p}, ann)
	pp.Change.Path.Requirements[0] = k("ZZ")
	if got, _ := f.r.Proposal(pp.Number); got.Change.Path.Requirements[0] != k("R1") {
		t.Error("proposal shares memory")
	}
	expect(t, "path requirement missing", f.try(Change{Kind: ChangePath, Path: Path{ID: "p9", Provider: "a", Contract: "b", Role: "c", Version: "d", Issuance: IssuanceFile, Requirements: []Key{k("ZZ")}}}), ErrNotFound)
	expect(t, "requirement repeat", f.try(c), ErrConflict)
	f.approve(Change{Kind: ChangeBuild, Build: Build{Build: "1.0.0", Commit: sum("b")[:40]}})
	expect(t, "build repeat", f.try(Change{Kind: ChangeBuild, Build: Build{Build: "1.0.0", Commit: sum("b")[:40]}}), ErrConflict)
	f.by(Change{Kind: ChangeHarness, Harness: Harness{ID: "h1", Identity: "r1"}}, cat, dan)
	expect(t, "harness repeat", f.try(Change{Kind: ChangeHarness, Harness: Harness{ID: "h1", Identity: "r2"}}), ErrConflict)
	reason(t, "identity", f.try(Change{Kind: ChangeHarness, Harness: Harness{ID: "h2", Identity: "r1"}}), ReasonIdentityTaken)
	f.must(second(f.r.Approve(pp.Number, bob)))
	expect(t, "path repeat", f.try(Change{Kind: ChangePath, Path: p}), ErrConflict)
	caps := []struct {
		name   string
		fill   func(w world, n int)
		n      int
		change Change
	}{
		{"requirements", func(w world, n int) {
			for len(w.r.requirements) < n {
				w.r.requirements[Key{"z", itoa(len(w.r.requirements))}] = &requirement{}
			}
		}, 10000, Change{Kind: ChangeRequirement, Requirement: req("NEW", false, TestRef{ID: "n"})}},
		{"builds", func(w world, n int) {
			for len(w.r.builds) < n {
				w.r.builds["9."+itoa(len(w.r.builds))+".0"] = &build{}
			}
		}, 10000, Change{Kind: ChangeBuild, Build: Build{Build: "5.5.5", Commit: sum("x")[:40]}}},
		{"harnesses", func(w world, n int) {
			for len(w.r.harnesses) < n {
				w.r.harnesses["z"+itoa(len(w.r.harnesses))] = &harness{}
			}
		}, 100, Change{Kind: ChangeHarness, Harness: Harness{ID: "new", Identity: "new"}}},
		{"paths", func(w world, n int) {
			for len(w.r.paths) < n {
				w.r.paths["z"+itoa(len(w.r.paths))] = &path{}
			}
		}, 1000, Change{Kind: ChangePath, Path: Path{ID: "new", Provider: "a", Contract: "b", Role: "c", Version: "d", Issuance: IssuanceFile, Requirements: []Key{k("R1")}}}},
	}
	for _, c := range caps {
		w := standard(t)
		c.fill(w, c.n-1)
		if err := w.try(c.change); err != nil {
			t.Errorf("%s fit %v", c.name, err)
		}
		x := standard(t)
		c.fill(x, c.n)
		expect(t, c.name, x.try(c.change), ErrLimit)
	}
	m := standard(t)
	for i := 0; i < maxImpls+3; i++ {
		m.approve(Change{Kind: ChangeImplemented, Implemented: Implemented{Key: k("R1"), Commit: sum("c" + itoa(i))[:40]}})
	}
	if len(m.r.requirements[k("R1")].impls) != 16 {
		t.Error("impl history")
	}
	h := setup(t)
	people := []Actor{ann, bob, cat, dan}
	for _, who := range people {
		for n := 0; n < 250; n++ {
			h.must(second(h.r.Propose(Change{Kind: ChangeBuild, Build: Build{Build: "1." + itoa(n) + ".0", Commit: sum(who.ID)[:40]}}, who)))
		}
		expect(t, "mine", second(h.r.Propose(Change{Kind: ChangeBuild, Build: Build{Build: "2.0.0", Commit: sum(who.ID + "x")[:40]}}, who)), ErrLimit)
	}
	expect(t, "total", second(h.r.Propose(Change{Kind: ChangeBuild, Build: Build{Build: "3.0.0", Commit: sum("e")[:40]}}, eve)), ErrLimit)
	d := setup(t)
	first := d.approve(Change{Kind: ChangeBuild, Build: Build{Build: "0.0.1", Commit: sum("a")[:40]}})
	for n := 2; n < 1000; n++ {
		d.r.done = append(d.r.done, uint64(100000+n))
	}
	kept := d.approve(Change{Kind: ChangeBuild, Build: Build{Build: "0.0.2", Commit: sum("b")[:40]}})
	if _, err := d.r.Proposal(first.Number); err != nil {
		t.Fatal("pruned early")
	}
	d.approve(Change{Kind: ChangeBuild, Build: Build{Build: "0.0.3", Commit: sum("c")[:40]}})
	expect(t, "pruned", second(d.r.Proposal(first.Number)), ErrNotFound)
	if _, err := d.r.Proposal(kept.Number); err != nil || len(d.r.done) != 1000 {
		t.Error("prune one")
	}
}

func TestConcurrency(t *testing.T) {
	f := standard(t)
	var wg sync.WaitGroup
	for n := 0; n < 8; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for m := 0; m < 20; m++ {
				f.r.Record(Result{Namespace: "v1", Test: "t-amount", Build: "1.0.0", Passed: true, Evidence: sum("e")}, runner)
				f.r.Gate(Release{Version: "r", Build: "1.0.0", Requirements: []Key{k("R1")}, Paths: []string{"p1"}})
				f.r.Requirement(k("R1"), "1.0.0")
				if p, err := f.r.Propose(Change{Kind: ChangeBuild, Build: Build{Build: "8." + itoa(n) + "." + itoa(m), Commit: sum("x")[:40]}}, ann); err == nil {
					f.r.Approve(p.Number, bob)
				}
			}
		}(n)
	}
	wg.Wait()
	if len(f.r.builds) != 161 || f.r.results[[3]string{"v1", "t-amount", "1.0.0"}].passes != 161 {
		t.Errorf("counts %d %d", len(f.r.builds), f.r.results[[3]string{"v1", "t-amount", "1.0.0"}].passes)
	}
	_ = time.Second
}
