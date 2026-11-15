package thirdparty

import (
	"strings"
	"testing"
)

func TestEdges(t *testing.T) {
	if !validText(strings.Repeat("a", 200)) || !validText("a ") || validText("a\u0080") {
		t.Error("text edges")
	}
	if hexOf("gbcd", 4) {
		t.Error("hex first character")
	}
	for _, ok := range []string{"https://www.npmjs.com/package/@scope/pkg", "https://proxy.golang.org/x/@v/v1.2.3.zip", "https://e.org:443/a"} {
		if !validSource(ok) {
			t.Errorf("source %s refused", ok)
		}
	}
	for _, bad := range []string{"https:// x", "https:///x", "https://:443", "https://u@e.org/x", "https://u:p@e.org", "https://e.org/x?", "https://e.org/x#", "https://e.org/%zz", "https://e.org/a b", "https://e.org/a\x7f", "https://e.org/\x80", "https://e.org/\x01"} {
		if validSource(bad) {
			t.Errorf("source %s accepted", bad)
		}
	}
	stray := Change{Kind: ChangeComponent, Component: comp("x", "x", "1.0.0"), Decision: Decision{Component: "y"}}
	if _, err := clean(stray); err == nil {
		t.Error("stray decision accepted")
	}
	g := setup(t)
	pf := comp("pf", "pf", "1.0.0")
	pf.License, pf.Distribution, pf.Artifact = "PolyForm-Noncommercial-1.0.0", DistNotShipped, ""
	g.adopted(pf)
	tp := comp("tp", "tp", "1.0.0")
	tp.Distribution, tp.Artifact, tp.Dependencies = DistToolOnly, "", []string{"pf"}
	g.approve(Change{Kind: ChangeComponent, Component: tp})
	reason(t, "dependency boundary through the API", g.try(Change{Kind: ChangeAdopt, Decision: Decision{Component: "tp"}}), ReasonDependencyBoundary)
	if c, _ := g.r.CheckRelease(Release{Version: "v", Commit: sum("p")[:40], Components: []string{"tp"}}); len(c.Problems) != 2 || c.Problems[0].Problem != ProblemNotAdopted || c.Problems[1].Problem != ProblemNotShipped {
		t.Errorf("tool parent missing dependency %+v", c.Problems)
	}
	dep := comp("dep", "dep", "1.0.0")
	g.adopted(dep)
	parent := comp("par", "par", "1.0.0")
	parent.Dependencies = []string{"dep"}
	g.adopted(parent)
	read, _ := g.r.Component("par")
	read.Component.Dependencies[0] = "zz"
	if again, _ := g.r.Component("par"); again.Component.Dependencies[0] != "dep" {
		t.Error("component read shares memory")
	}
	list := g.r.ByName("par")
	list[0].Component.Dependencies[0] = "zz"
	if again := g.r.ByName("par"); again[0].Component.Dependencies[0] != "dep" {
		t.Error("by name shares memory")
	}
	c := comp("x", "x", "1.0.0")
	c.Dependencies = []string{"a", "a"}
	if c.valid() {
		t.Error("duplicate dependencies")
	}
	f := setup(t)
	expect(t, "duplicate components", second(f.r.CheckRelease(Release{Version: "v", Commit: sum("p")[:40], Components: []string{"a", "a"}})), ErrInvalid)
}
