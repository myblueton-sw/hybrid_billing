package releasetrace

import "testing"

func keys(n int) []Key {
	out := []Key{}
	for i := 0; i < n; i++ {
		out = append(out, Key{Namespace: "v1", ID: "k" + itoa(100000+i)})
	}
	return out
}

func TestExactLimits(t *testing.T) {
	q := req("R1", false, TestRef{ID: "t"})
	q.Interfaces = nil
	if !q.valid() {
		t.Error("no interfaces")
	}
	q.Tests = nil
	for i := 0; i < 256; i++ {
		q.Tests = append(q.Tests, TestRef{ID: "t" + itoa(1000+i)})
	}
	for i := 0; i < 64; i++ {
		q.Interfaces = append(q.Interfaces, "i"+itoa(1000+i))
	}
	if !q.valid() {
		t.Error("256 tests and 64 interfaces")
	}
	f := setup(t)
	path := Path{ID: "p", Provider: "a", Contract: "b", Role: "c", Version: "d", Issuance: IssuanceFile, Requirements: keys(256)}
	expect(t, "256 path requirements", f.try(Change{Kind: ChangePath, Path: path}), ErrNotFound)
	path.Requirements = keys(257)
	expect(t, "257 path requirements", f.try(Change{Kind: ChangePath, Path: path}), ErrInvalid)
	paths := []string{}
	for i := 0; i < 1000; i++ {
		paths = append(paths, "p"+itoa(10000+i))
	}
	expect(t, "1000 paths", second(f.r.Gate(Release{Version: "r", Build: "1.0.0", Requirements: keys(10000), Paths: paths})), nil)
	expect(t, "10001 requirements", second(f.r.Gate(Release{Version: "r", Build: "1.0.0", Requirements: keys(10001), Paths: paths})), ErrInvalid)
	expect(t, "1001 paths", second(f.r.Gate(Release{Version: "r", Build: "1.0.0", Requirements: keys(1), Paths: append(paths, "p99999")})), ErrInvalid)
	for _, period := range []string{"2026-1/", "2026-0:", "2026-00", "2026-13", "2026-20", "2026-+1", "2026--1", "2026-/9", "2026-:0"} {
		if _, ok := month(period); ok {
			t.Errorf("period %s", period)
		}
	}
	for period, want := range map[string]int{"2026-01": 24312, "2026-09": 24320, "2026-10": 24321, "1000-12": 12011} {
		if m, ok := month(period); !ok || m != want {
			t.Errorf("period %s %d", period, m)
		}
	}
	m := standard(t)
	for i := 0; i < 20; i++ {
		m.approve(Change{Kind: ChangeImplemented, Implemented: Implemented{Key: k("R1"), Commit: sum("c" + itoa(i))[:40]}})
		if n := len(m.r.requirements[k("R1")].impls); n != min(i+2, 16) {
			t.Fatalf("impl %d kept %d", i, n)
		}
	}
	if m.r.requirements[k("R1")].impls[0].commit != sum("c4")[:40] {
		t.Error("oldest kept")
	}
}

func TestReadSideAndOrdering(t *testing.T) {
	f := standard(t)
	if v, _ := f.r.Requirement(k("R1"), "1.0.0"); v.State != StateTested {
		t.Fatal("fixture")
	}
	f.by(Change{Kind: ChangeRequirement, Requirement: req("R2", false, TestRef{ID: "t-amount"})}, cat, ann)
	if v, _ := f.r.Requirement(k("R1"), "1.0.0"); v.State != StateImplemented || v.Tests[0].Counted {
		t.Errorf("binder registered requirement %+v", v.State)
	}
	if res, _ := f.r.Results("v1", "t-amount", "1.0.0"); res.Counted {
		t.Error("result view counted")
	}
	if res, _ := f.r.Results("v1", "t-"+CategoryBulk, "1.0.0"); !res.Counted {
		t.Error("other result")
	}
	g := standard(t)
	g.approve(Change{Kind: ChangeImplemented, Implemented: Implemented{Key: k("R1"), Commit: sum("c2")[:40]}})
	if res, _ := g.r.Results("v1", "t-amount", "1.0.0"); res.Counted {
		t.Error("result before implementation counted")
	}
	s := setup(t)
	s.approve(Change{Kind: ChangeRequirement, Requirement: req("R1", false, TestRef{ID: "t1"})})
	s.by(Change{Kind: ChangeBuild, Build: Build{Build: "0.1.0", Commit: sum("b0")[:40]}}, eve, fay)
	s.by(Change{Kind: ChangeImplemented, Implemented: Implemented{Key: k("R1"), Commit: sum("c")[:40]}}, gus, hal)
	s.by(Change{Kind: ChangeBuild, Build: Build{Build: "1.0.0", Commit: sum("b")[:40]}}, eve, fay)
	s.by(Change{Kind: ChangeHarness, Harness: Harness{ID: "h", Identity: "r"}}, hal, dan)
	reason(t, "single implementation", second(s.r.Record(Result{Namespace: "v1", Test: "t1", Build: "1.0.0", Passed: true, Evidence: sum("e")}, Actor{ID: "r", Kind: KindSystem})), ReasonSelfVerification)
	p := standard(t)
	for _, period := range []string{"2026-09", "2026-10", "2026-08", "2026-06", "2026-07"} {
		p.reconcile("p1", period, true)
	}
	v, _ := p.r.Path("p1")
	got := ""
	for _, x := range v.Reconciliations {
		got += x.Period + " "
	}
	if got != "2026-06 2026-07 2026-08 2026-09 2026-10 " {
		t.Errorf("order %s", got)
	}
	one := standard(t)
	one.reconcile("p1", "2026-10", false)
	if g, _ := one.r.Gate(Release{Version: "r", Build: "1.0.0", Requirements: []Key{k("R1")}, Paths: []string{"p1"}}); !hasProblem(g, ProblemPathReconcile) {
		t.Error("single unmatched period")
	}
	c := setup(t)
	refs := allCategories("t")
	c.approve(Change{Kind: ChangeRequirement, Requirement: req("R1", true, refs...)})
	c.approve(Change{Kind: ChangeImplemented, Implemented: Implemented{Key: k("R1"), Commit: sum("c")[:40]}})
	c.approve(Change{Kind: ChangeBuild, Build: Build{Build: "1.0.0", Commit: sum("b")[:40]}})
	c.by(Change{Kind: ChangeHarness, Harness: Harness{ID: "h1", Identity: runner.ID}}, cat, dan)
	for index, tr := range refs {
		c.pass("v1", tr.ID, "1.0.0", index != 0)
	}
	c.approve(Change{Kind: ChangePath, Path: Path{ID: "p1", Provider: "a", Contract: "b", Role: "c", Version: "d", Issuance: IssuanceFile, Requirements: []Key{k("R1")}}})
	c.reconcile("p1", "2026-09", true)
	c.reconcile("p1", "2026-10", true)
	reason(t, "failing category", c.activate("p1", "1.0.0"), ReasonCategoryMissing)
	o := setup(t)
	for _, id := range []string{"C3", "C1", "C5", "C2", "C4", "A1", "B1"} {
		o.approve(Change{Kind: ChangeRequirement, Requirement: req(id, id[0] == 'C', TestRef{ID: "x" + id})})
	}
	gv, _ := o.r.Gate(Release{Version: "r", Build: "1.0.0", Requirements: []Key{k("A0"), k("A1"), k("B0"), k("B1")}, Paths: []string{"q1", "q2"}})
	want := []Problem{{ProblemCommonMissing, "v1/C1"}, {ProblemCommonMissing, "v1/C2"}, {ProblemCommonMissing, "v1/C3"}, {ProblemCommonMissing, "v1/C4"}, {ProblemCommonMissing, "v1/C5"},
		{ProblemNotTested, "v1/A1"}, {ProblemNotTested, "v1/B1"}, {ProblemUnknown, "q1"}, {ProblemUnknown, "q2"}, {ProblemUnknown, "v1/A0"}, {ProblemUnknown, "v1/B0"}}
	if len(gv.Problems) != len(want) {
		t.Fatalf("problems %+v", gv.Problems)
	}
	for i := range want {
		if gv.Problems[i] != want[i] {
			t.Errorf("problem %d %+v", i, gv.Problems[i])
		}
	}
}

func TestTemplatesNeedEvidence(t *testing.T) {
	f := standard(t)
	f.reconcile("p1", "2026-09", true)
	f.reconcile("p1", "2026-10", true)
	f.must(f.activate("p1", "1.0.0"))
	if v, _ := f.r.Path("p1"); v.Proof != sum("proof") {
		t.Error("proof kept")
	}
	rel := Release{Version: "r", Build: "1.0.0", Requirements: []Key{k("R1")}, Paths: []string{"p1"}}
	f.must(f.r.RevokeHarness("h1", ann))
	g, _ := f.r.Gate(rel)
	if len(g.Templates) != 0 || !hasProblem(g, ProblemPathEvidence) {
		t.Errorf("revoked evidence %+v", g)
	}
	r := standard(t)
	r.reconcile("p1", "2026-09", true)
	r.reconcile("p1", "2026-10", true)
	r.must(r.activate("p1", "1.0.0"))
	r.approve(Change{Kind: ChangeRequirement, Requirement: req("R2", true, TestRef{ID: "x"})})
	r.approve(Change{Kind: ChangeRetire, Retire: Retire{Key: k("R1"), Replacement: k("R2"), Reason: "split"}})
	g, _ = r.r.Gate(rel)
	if len(g.Templates) != 0 || !hasProblem(g, ProblemPathEvidence) {
		t.Errorf("retired path requirement %+v", g)
	}
	u := standard(t)
	u.approve(Change{Kind: ChangeRequirement, Requirement: req("U1", false, TestRef{ID: "u1"})})
	u.approve(Change{Kind: ChangeImplemented, Implemented: Implemented{Key: k("U1"), Commit: sum("u")[:40]}})
	u.approve(Change{Kind: ChangeBuild, Build: Build{Build: "2.0.0", Commit: sum("b2")[:40]}})
	u.pass("v1", "u1", "2.0.0", true)
	u.approve(Change{Kind: ChangePath, Path: Path{ID: "p2", Provider: "a", Contract: "b", Role: "c", Version: "d", Issuance: IssuanceFile, Requirements: []Key{k("U1")}}})
	if g, _ := u.r.Gate(Release{Version: "r", Build: "2.0.0", Requirements: []Key{k("R1"), k("U1")}, Paths: []string{"p2"}}); !hasProblem(g, ProblemPathEvidence) || !hasProblem(g, ProblemPathInactive) {
		t.Errorf("inactive untagged %+v", g.Problems)
	}
	n := standard(t)
	n.approve(Change{Kind: ChangeRequirement, Requirement: req("X1", false, TestRef{ID: "x1"})})
	n.approve(Change{Kind: ChangeRetire, Retire: Retire{Key: k("X1"), Reason: "dropped"}})
	g, _ = n.r.Gate(Release{Version: "r", Build: "1.0.0", Requirements: []Key{k("R1"), k("X1")}, Paths: []string{"p1"}})
	found := false
	for _, p := range g.Problems {
		found = found || p == Problem{ProblemRetired, "v1/X1"}
	}
	if !found {
		t.Errorf("retired subject %+v", g.Problems)
	}
}
