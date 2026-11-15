package releasetrace

import (
	"testing"
	"time"
)

func TestStates(t *testing.T) {
	f := setup(t)
	f.approve(Change{Kind: ChangeRequirement, Requirement: req("R1", true, TestRef{ID: "t1"}, TestRef{ID: "t2", Category: CategoryAmount})})
	f.approve(Change{Kind: ChangeBuild, Build: Build{Build: "0.9.0", Commit: sum("b0")[:40]}})
	f.by(Change{Kind: ChangeHarness, Harness: Harness{ID: "h1", Identity: runner.ID}}, cat, dan)
	state := func(build string) string {
		v, err := f.r.Requirement(k("R1"), build)
		f.must(err)
		return v.State
	}
	if state("1.0.0") != StateDesigned {
		t.Error("designed")
	}
	f.approve(Change{Kind: ChangeImplemented, Implemented: Implemented{Key: k("R1"), Commit: sum("c1")[:40]}})
	f.pass("v1", "t1", "0.9.0", true)
	f.pass("v1", "t2", "0.9.0", true)
	if state("0.9.0") != StateImplemented {
		t.Error("build before implementation counts")
	}
	f.approve(Change{Kind: ChangeBuild, Build: Build{Build: "1.0.0", Commit: sum("b1")[:40]}})
	if state("1.0.0") != StateImplemented {
		t.Error("NOT_RUN counted")
	}
	f.pass("v1", "t1", "1.0.0", true)
	if state("1.0.0") != StateImplemented {
		t.Error("one test missing")
	}
	f.pass("v1", "t2", "1.0.0", false)
	if state("1.0.0") != StateFailing {
		t.Error("failing")
	}
	f.pass("v1", "t2", "1.0.0", true)
	v, _ := f.r.Requirement(k("R1"), "1.0.0")
	if v.State != StateTested || v.Tests[1].Passes != 1 || v.Tests[1].Failures != 1 || !v.Tests[1].Counted || v.Commit != sum("c1")[:40] {
		t.Errorf("tested %+v", v)
	}
	reason(t, "proof wrong build", f.try(Change{Kind: ChangeProof, Proof: Proof{Key: k("R1"), Build: "0.9.0", Environment: "prod", Customer: "acme", Period: "2026-10", Evidence: sum("p")}}), ReasonNotTested)
	f.approve(Change{Kind: ChangeProof, Proof: Proof{Key: k("R1"), Build: "1.0.0", Environment: "prod", Customer: "acme", Period: "2026-10", Evidence: sum("p")}})
	if state("1.0.0") != StateCustomerProven {
		t.Error("proven")
	}
	expect(t, "proof repeat", f.try(Change{Kind: ChangeProof, Proof: Proof{Key: k("R1"), Build: "1.0.0", Environment: "prod", Customer: "acme", Period: "2026-10", Evidence: sum("p")}}), ErrConflict)
	reason(t, "proof unknown build", f.try(Change{Kind: ChangeProof, Proof: Proof{Key: k("R1"), Build: "9.9.9", Environment: "prod", Customer: "acme", Period: "2026-10", Evidence: sum("p")}}), ReasonUnknownBuild)
	expect(t, "proof missing", f.try(Change{Kind: ChangeProof, Proof: Proof{Key: k("R9"), Build: "1.0.0", Environment: "prod", Customer: "acme", Period: "2026-10", Evidence: sum("p")}}), ErrNotFound)
	f.approve(Change{Kind: ChangeImplemented, Implemented: Implemented{Key: k("R1"), Commit: sum("c2")[:40]}})
	if state("1.0.0") != StateImplemented {
		t.Error("reimplementation keeps old passes")
	}
	f.approve(Change{Kind: ChangeBuild, Build: Build{Build: "1.1.0", Commit: sum("b2")[:40]}})
	f.pass("v1", "t1", "1.1.0", false)
	f.pass("v1", "t2", "1.1.0", true)
	f.by(Change{Kind: ChangeHarness, Harness: Harness{ID: "h2", Identity: "runner2"}}, gus, hal)
	f.must(second(f.r.Record(Result{Namespace: "v1", Test: "t1", Build: "1.1.0", Passed: true, Evidence: sum("x")}, Actor{ID: "runner2", Kind: KindSystem})))
	if state("1.1.0") != StateTested {
		t.Fatal("second harness pass")
	}
	f.must(f.r.RevokeHarness("h2", ann))
	if state("1.1.0") != StateImplemented {
		t.Error("revoked latest does not revive older failure or pass")
	}
	if res, _ := f.r.Results("v1", "t1", "1.1.0"); res.Counted || res.Harness != "h2" {
		t.Errorf("result view %+v", res)
	}
	expect(t, "results missing", second(f.r.Results("v1", "t9", "1.1.0")), ErrNotFound)
	expect(t, "requirement missing", second(f.r.Requirement(k("R9"), "1.0.0")), ErrNotFound)
}

func TestRecordRules(t *testing.T) {
	f := standard(t)
	x := Result{Namespace: "v1", Test: "t-amount", Build: "1.0.0", Passed: true, Evidence: sum("e")}
	reason(t, "assistant", second(f.r.Record(x, Actor{ID: runner.ID, Kind: KindAssistant})), ReasonAINotAllowed)
	reason(t, "human", second(f.r.Record(x, Actor{ID: runner.ID, Kind: KindHuman})), ReasonNotHarness)
	reason(t, "unbound", second(f.r.Record(x, Actor{ID: "x", Kind: KindSystem})), ReasonNotHarness)
	expect(t, "actor", second(f.r.Record(x, Actor{Kind: KindSystem})), ErrInvalid)
	for _, mutate := range []func(*Result){func(r *Result) { r.Namespace = "" }, func(r *Result) { r.Test = "" }, func(r *Result) { r.Build = "1" }, func(r *Result) { r.Evidence = "" }} {
		y := x
		mutate(&y)
		expect(t, "form", second(f.r.Record(y, runner)), ErrInvalid)
	}
	other := x
	other.Namespace = "v2"
	reason(t, "other namespace", second(f.r.Record(other, runner)), ReasonUnknownTest)
	unknown := x
	unknown.Build = "9.9.9"
	reason(t, "unknown build", second(f.r.Record(unknown, runner)), ReasonUnknownBuild)
	cases := []struct {
		name string
		h    [2]Actor
		c    Change
	}{
		{"requirement people", [2]Actor{ann, cat}, Change{}},
		{"build people", [2]Actor{eve, fay}, Change{Kind: ChangeBuild, Build: Build{Build: "2.0.0", Commit: sum("b")[:40]}}},
		{"implementer", [2]Actor{gus, hal}, Change{Kind: ChangeImplemented, Implemented: Implemented{Key: k("R1"), Commit: sum("c9")[:40]}}},
	}
	for index, c := range cases {
		g := standard(t)
		build := "1.0.0"
		if c.c.Kind != "" {
			g.by(c.c, c.h[0], c.h[1])
			if c.c.Kind == ChangeBuild {
				build = "2.0.0"
			}
		}
		id := "hx" + itoa(index)
		g.by(Change{Kind: ChangeHarness, Harness: Harness{ID: id, Identity: "r-" + id}}, c.h[0], Actor{ID: "zed", Kind: KindHuman})
		if c.c.Kind == ChangeImplemented {
			g.approve(Change{Kind: ChangeBuild, Build: Build{Build: "3.0.0", Commit: sum("b3")[:40]}})
			build = "3.0.0"
		}
		y := x
		y.Build = build
		reason(t, c.name, second(g.r.Record(y, Actor{ID: "r-" + id, Kind: KindSystem})), ReasonSelfVerification)
	}
	g := standard(t)
	g.by(Change{Kind: ChangeImplemented, Implemented: Implemented{Key: k("R1"), Commit: sum("c9")[:40]}}, cat, gus)
	if v, _ := g.r.Requirement(k("R1"), "1.0.0"); v.Tests[0].Counted {
		t.Error("implementer harness counted on read")
	}
	reason(t, "revoke system", f.r.RevokeHarness("h1", runner), ReasonNotHuman)
	expect(t, "revoke form", f.r.RevokeHarness("", ann), ErrInvalid)
	expect(t, "revoke missing", f.r.RevokeHarness("nope", ann), ErrNotFound)
	f.must(f.r.RevokeHarness("h1", ann))
	reason(t, "revoked", second(f.r.Record(x, runner)), ReasonNotHarness)
	h := standard(t)
	for len(h.r.results) < 1000000 {
		h.r.results[[3]string{"z", itoa(len(h.r.results)), "1.0.0"}] = &result{}
	}
	h.pass("v1", "t-amount", "1.0.0", true)
	h.approve(Change{Kind: ChangeRequirement, Requirement: req("R2", false, TestRef{ID: "new"})})
	expect(t, "results cap", second(h.r.Record(Result{Namespace: "v1", Test: "new", Build: "1.0.0", Passed: true, Evidence: sum("n")}, runner)), ErrLimit)
	_ = time.Second
}
