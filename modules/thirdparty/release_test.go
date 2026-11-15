package thirdparty

import (
	"sync"
	"testing"
)

func TestLicenceTable(t *testing.T) {
	want := map[string][3]string{
		"MIT": {"all", "all", "all"}, "Apache-2.0": {"all", "all", "all"}, "BSD-2-Clause": {"all", "all", "all"}, "BSD-3-Clause": {"all", "all", "all"},
		"ISC": {"all", "all", "all"}, "Zlib": {"all", "all", "all"}, "OFL-1.1": {"all", "all", "all"}, "CC-BY-4.0": {"all", "all", "all"},
		"MPL-2.0": {"unmodified", "all", "all"}, "CC-BY-SA-4.0": {"none", "all", "all"}, "LGPL-2.1-only": {"none", "all", "all"},
		"LGPL-3.0-only": {"none", "all", "all"}, "GPL-2.0-only": {"none", "none", "all"}, "GPL-3.0-only": {"none", "none", "all"},
		"AGPL-3.0-only": {"none", "none", "all"}, "PolyForm-Noncommercial-1.0.0": {"none", "none", "none"},
	}
	if len(want) != len(licenses) {
		t.Fatalf("licences %d", len(licenses))
	}
	dists := []string{DistBundled, DistSeparate, DistToolOnly}
	for license, rules := range want {
		for index, dist := range dists {
			for _, mod := range []string{ModUnmodified, ModPatched} {
				expected := rules[index] == "all" || rules[index] == "unmodified" && mod == ModUnmodified
				if got := allowed(license, dist, mod); got != expected {
					t.Errorf("%s %s %s = %v", license, dist, mod, got)
				}
			}
		}
		if !allowed(license, DistNotShipped, ModForked) {
			t.Errorf("%s not shipped", license)
		}
	}
}

func TestAdoption(t *testing.T) {
	f := setup(t)
	cases := map[string]func(*Component){
		ReasonTestsFailed:       func(c *Component) { c.Passed = false },
		ReasonNotOffline:        func(c *Component) { c.Offline = false },
		ReasonTelemetryRequired: func(c *Component) { c.Telemetry = TelemetryRequired },
		ReasonLicenseBoundary:   func(c *Component) { c.License = "GPL-3.0-only" },
	}
	for want, mutate := range cases {
		c := comp("c-"+want, "n-"+want, "1.0.0")
		mutate(&c)
		f.approve(Change{Kind: ChangeComponent, Component: c})
		reason(t, want, f.try(Change{Kind: ChangeAdopt, Decision: Decision{Component: c.ID}}), want)
	}
	separate := comp("sep", "sep", "1.0.0")
	separate.Distribution, separate.Offline = DistSeparate, false
	f.approve(Change{Kind: ChangeComponent, Component: separate})
	reason(t, "separate offline", f.try(Change{Kind: ChangeAdopt, Decision: Decision{Component: "sep"}}), ReasonNotOffline)
	tool := comp("tool", "tool", "1.0.0")
	tool.Distribution, tool.Artifact, tool.Offline, tool.License = DistToolOnly, "", false, "GPL-3.0-only"
	f.adopted(tool)
	notShipped := comp("ns", "ns", "1.0.0")
	notShipped.Distribution, notShipped.Artifact, notShipped.Offline = DistNotShipped, "", false
	f.adopted(notShipped)
	optional := comp("opt", "opt", "1.0.0")
	optional.Telemetry, optional.TelemetryEvidence = TelemetryOptional, sum("off")
	f.adopted(optional)
	reason(t, "adopt twice", f.try(Change{Kind: ChangeAdopt, Decision: Decision{Component: "opt"}}), ReasonNotCandidate)
	expect(t, "adopt missing", f.try(Change{Kind: ChangeAdopt, Decision: Decision{Component: "nope"}}), ErrNotFound)
	reason(t, "reject adopted", f.try(Change{Kind: ChangeReject, Decision: Decision{Component: "opt", Reason: "x"}}), ReasonNotCandidate)
	reason(t, "revoke candidate", f.try(Change{Kind: ChangeRevoke, Decision: Decision{Component: "c-tests_failed", Reason: "x"}}), ReasonNotAdopted)
	parent := comp("p", "p", "1.0.0")
	parent.Dependencies = []string{"tool"}
	reason(t, "tool dependency", f.try(Change{Kind: ChangeComponent, Component: parent}), ReasonToolDependency)
	parent.Dependencies = []string{"ns"}
	reason(t, "not shipped dependency", f.try(Change{Kind: ChangeComponent, Component: parent}), ReasonToolDependency)
	toolParent := comp("tp", "tp", "1.0.0")
	toolParent.Distribution, toolParent.Artifact, toolParent.Dependencies = DistToolOnly, "", []string{"tool"}
	f.adopted(toolParent)
	lgpl := comp("lgpl", "lgpl", "1.0.0")
	lgpl.License, lgpl.Distribution = "LGPL-3.0-only", DistSeparate
	f.approve(Change{Kind: ChangeComponent, Component: lgpl})
	parent.Dependencies = []string{"lgpl"}
	f.approve(Change{Kind: ChangeComponent, Component: parent})
	reason(t, "dependency not adopted", f.try(Change{Kind: ChangeAdopt, Decision: Decision{Component: "p"}}), ReasonDependencyNotAdopted)
	f.approve(Change{Kind: ChangeAdopt, Decision: Decision{Component: "lgpl"}})
	f.approve(Change{Kind: ChangeAdopt, Decision: Decision{Component: "p"}})
	bundledLGPL := comp("blgpl", "blgpl", "1.0.0")
	bundledLGPL.License = "MPL-2.0"
	f.adopted(bundledLGPL)
	patched := comp("mplp", "mplp", "1.0.0")
	patched.License, patched.Modification, patched.Patch = "MPL-2.0", ModPatched, sum("patch")
	patched.Distribution = DistSeparate
	f.adopted(patched)
	q := comp("q", "q", "1.0.0")
	q.Dependencies = []string{"mplp"}
	f.approve(Change{Kind: ChangeComponent, Component: q})
	f.approve(Change{Kind: ChangeAdopt, Decision: Decision{Component: "q"}})
	cc := comp("ccsa", "ccsa", "1.0.0")
	cc.License, cc.Distribution = "CC-BY-SA-4.0", DistSeparate
	f.adopted(cc)
	if v, _ := f.r.Component("ccsa"); v.State != StateAdopted {
		t.Error("separate share-alike")
	}
	inner := comp("inner", "inner", "1.0.0")
	inner.License, inner.Modification, inner.Patch = "MPL-2.0", ModPatched, sum("p2")
	inner.Distribution = DistBundled
	f.approve(Change{Kind: ChangeComponent, Component: inner})
	reason(t, "patched mpl bundled", f.try(Change{Kind: ChangeAdopt, Decision: Decision{Component: "inner"}}), ReasonLicenseBoundary)
	viaParent := comp("vp", "vp", "1.0.0")
	viaParent.Dependencies = []string{"ccsa2"}
	share := comp("ccsa2", "ccsa2", "1.0.0")
	share.License, share.Distribution, share.Artifact = "CC-BY-SA-4.0", DistBundled, sum("a")
	f.approve(Change{Kind: ChangeComponent, Component: share})
	f.approve(Change{Kind: ChangeComponent, Component: viaParent})
	f.r.components["ccsa2"].state = StateAdopted
	reason(t, "dependency boundary", f.try(Change{Kind: ChangeAdopt, Decision: Decision{Component: "vp"}}), ReasonDependencyBoundary)
}

func TestReleases(t *testing.T) {
	f := setup(t)
	a := comp("a", "alpha", "1.0.0")
	b := comp("b", "beta", "2.0.0")
	b.Dependencies = []string{"a"}
	f.adopted(a)
	f.adopted(b)
	rel := Release{Version: "1.0.0", Commit: sum("product")[:40], Components: []string{"a", "b"}}
	c, err := f.r.CheckRelease(rel)
	if err != nil || !c.Ready || len(c.Problems) != 0 || c.Bundle == "" {
		t.Fatalf("ready %+v %v", c, err)
	}
	if c.Bundle != "2081a157f15280a546c1a293d614715847f1796b3c0b0305b7212c608bb6799d" {
		t.Errorf("bundle %s", c.Bundle)
	}
	f.approve(Change{Kind: ChangeComponent, Component: comp("cand", "cand", "1.0.0")})
	rej := comp("rej", "rej", "1.0.0")
	f.approve(Change{Kind: ChangeComponent, Component: rej})
	f.approve(Change{Kind: ChangeReject, Decision: Decision{Component: "rej", Reason: "no"}})
	tool := comp("tool", "tool", "1.0.0")
	tool.Distribution, tool.Artifact = DistToolOnly, ""
	f.adopted(tool)
	ns := comp("ns", "ns", "1.0.0")
	ns.Distribution, ns.Artifact = DistNotShipped, ""
	f.adopted(ns)
	dup := comp("dup", "ALPHA", "1.1.0")
	f.adopted(dup)
	bad := Release{Version: "1.0.1", Commit: sum("product"), Components: []string{"a", "b", "cand", "dup", "ghost", "ns", "rej", "tool"}}
	c, _ = f.r.CheckRelease(bad)
	got := map[string]string{}
	for _, p := range c.Problems {
		got[p.Component] = p.Problem
	}
	want := map[string]string{"cand": ProblemNotAdopted, "dup": ProblemDuplicateName, "ghost": ProblemUnknown, "ns": ProblemNotShipped, "rej": ProblemNotAdopted, "tool": ProblemNotShipped}
	if c.Ready || c.Bundle != "" || len(c.Problems) != len(want) {
		t.Errorf("problems %+v", c)
	}
	for id, p := range want {
		if got[id] != p {
			t.Errorf("%s got %s", id, got[id])
		}
	}
	missing := Release{Version: "1.0.2", Commit: sum("product")[:40], Components: []string{"b"}}
	if c, _ := f.r.CheckRelease(missing); c.Ready || c.Problems[0].Problem != ProblemMissingDependency {
		t.Errorf("missing dependency %+v", c)
	}
	view, err := f.r.RecordRelease(rel, ann)
	if err != nil || view.By != "ann" || !view.At.Equal(start) || view.Bundle == "" {
		t.Fatalf("record %+v %v", view, err)
	}
	f.approve(Change{Kind: ChangeRevoke, Decision: Decision{Component: "a", Reason: "licence wrong"}})
	if c, _ := f.r.CheckRelease(rel); c.Ready || c.Problems[0].Problem != ProblemRevoked {
		t.Errorf("revoked %+v", c)
	}
	if again, err := f.r.RecordRelease(rel, ann); err != nil || again.Bundle != view.Bundle {
		t.Errorf("repeat after revoke %v", err)
	}
	other := rel
	other.Commit = sum("other")[:40]
	expect(t, "conflict", second(f.r.RecordRelease(other, ann)), ErrConflict)
	swapped := rel
	swapped.Components = []string{"a", "c"}
	expect(t, "conflict member", second(f.r.RecordRelease(swapped, ann)), ErrConflict)
	shorter := rel
	shorter.Components = []string{"a"}
	expect(t, "conflict list", second(f.r.RecordRelease(shorter, ann)), ErrConflict)
	reason(t, "not ready", second(f.r.RecordRelease(Release{Version: "2.0.0", Commit: rel.Commit, Components: []string{"a", "b"}}, ann)), ReasonNotReady)
	reason(t, "assistant", second(f.r.RecordRelease(rel, bot)), ReasonAINotAllowed)
	expect(t, "form", second(f.r.RecordRelease(Release{Version: "x", Commit: "y", Components: []string{"a"}}, ann)), ErrInvalid)
	read, _ := f.r.Release("1.0.0")
	read.Release.Components[0] = "zz"
	if again, _ := f.r.Release("1.0.0"); again.Release.Components[0] != "a" {
		t.Error("release read shares memory")
	}
	expect(t, "release missing", second(f.r.Release("nope")), ErrNotFound)
	forms := []Release{
		{Version: "", Commit: rel.Commit, Components: []string{"a"}},
		{Version: "v", Commit: "abc", Components: []string{"a"}},
		{Version: "v", Commit: rel.Commit},
		{Version: "v", Commit: rel.Commit, Components: []string{"b", "a"}},
		{Version: "v", Commit: rel.Commit, Components: []string{"a b"}},
	}
	for index, x := range forms {
		expect(t, "check form "+itoa(index), second(f.r.CheckRelease(x)), ErrInvalid)
	}
	big := Release{Version: "v", Commit: rel.Commit}
	for i := 0; i < 1000; i++ {
		big.Components = append(big.Components, "c"+itoa(1000+i))
	}
	if c, err := f.r.CheckRelease(big); err != nil || len(c.Problems) != 1000 {
		t.Errorf("1000 components %v", err)
	}
	big.Components = append(big.Components, "c9999")
	expect(t, "1001", second(f.r.CheckRelease(big)), ErrInvalid)
	g := setup(t)
	g.adopted(comp("a", "alpha", "1.0.0"))
	for len(g.r.releases) < 9999 {
		g.r.releases["x"+itoa(len(g.r.releases))] = &ReleaseView{}
	}
	g.must(second(g.r.RecordRelease(Release{Version: "fit", Commit: rel.Commit, Components: []string{"a"}}, ann)))
	expect(t, "release cap", second(g.r.RecordRelease(Release{Version: "over", Commit: rel.Commit, Components: []string{"a"}}, ann)), ErrLimit)
}

func TestConcurrency(t *testing.T) {
	f := setup(t)
	f.adopted(comp("a", "alpha", "1.0.0"))
	var wg sync.WaitGroup
	for n := 0; n < 8; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for m := 0; m < 20; m++ {
				id := "c" + itoa(n) + "-" + itoa(m)
				if p, err := f.r.Propose(Change{Kind: ChangeComponent, Component: comp(id, id, "1.0.0")}, ann); err == nil {
					f.r.Approve(p.Number, bob)
				}
				f.r.CheckRelease(Release{Version: "v", Commit: sum("p")[:40], Components: []string{"a"}})
				f.r.ByName("alpha")
			}
		}(n)
	}
	wg.Wait()
	if len(f.r.components) != 161 {
		t.Errorf("components %d", len(f.r.components))
	}
}
