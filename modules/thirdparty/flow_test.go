package thirdparty

import (
	"errors"
	"strings"
	"testing"
)

func TestForms(t *testing.T) {
	expect(t, "clock", second(NewRegistry(nil)), ErrInvalid)
	if !errors.Is(Refusal{Reason: "x"}, ErrRejected) || (Refusal{Reason: "x"}).Error() != ErrRejected.Error()+": x" {
		t.Error("refusal")
	}
	if !comp("a", "a", "1.0.0").valid() {
		t.Fatal("fixture")
	}
	bad := map[string]func(*Component){
		"id": func(c *Component) { c.ID = "" }, "name": func(c *Component) { c.Name = "" }, "version": func(c *Component) { c.Version = "^1.0" },
		"commit": func(c *Component) { c.Commit = c.Commit[:39] }, "commit upper": func(c *Component) { c.Commit = strings.ToUpper(c.Commit) },
		"source": func(c *Component) { c.Source = "http://x" }, "source user": func(c *Component) { c.Source = "https://u@x" },
		"source query": func(c *Component) { c.Source = "https://x/?a" }, "source space": func(c *Component) { c.Source = "https://x y" },
		"source ctl": func(c *Component) { c.Source = "https://x\x7f" }, "source long": func(c *Component) { c.Source = "https://" + strings.Repeat("a", 193) },
		"source empty": func(c *Component) { c.Source = "https://" }, "license": func(c *Component) { c.License = "" },
		"license text": func(c *Component) { c.LicenseText = "" }, "notice": func(c *Component) { c.Notice = "x" }, "sbom": func(c *Component) { c.SBOM = "" },
		"tests": func(c *Component) { c.Tests = "" }, "telemetry": func(c *Component) { c.Telemetry = "on" }, "mod": func(c *Component) { c.Modification = "x" },
		"dist": func(c *Component) { c.Distribution = "x" }, "no artifact": func(c *Component) { c.Artifact = "" },
		"artifact bad": func(c *Component) { c.Artifact = "x" }, "patch unmodified": func(c *Component) { c.Patch = sum("p") },
		"patched no patch": func(c *Component) { c.Modification = ModPatched }, "patch bad": func(c *Component) { c.Modification, c.Patch = ModForked, "x" },
		"optional no evidence": func(c *Component) { c.Telemetry = TelemetryOptional },
		"evidence on none":     func(c *Component) { c.TelemetryEvidence = sum("e") },
		"evidence bad":         func(c *Component) { c.Telemetry, c.TelemetryEvidence = TelemetryOptional, "x" },
		"self dep":             func(c *Component) { c.Dependencies = []string{c.ID} },
		"dep order":            func(c *Component) { c.Dependencies = []string{"b", "a"} },
		"dep bad":              func(c *Component) { c.Dependencies = []string{"a b"} },
		"self supersedes":      func(c *Component) { c.Supersedes = c.ID },
		"supersedes bad":       func(c *Component) { c.Supersedes = "a b" },
		"many deps": func(c *Component) {
			for i := 0; i <= 256; i++ {
				c.Dependencies = append(c.Dependencies, "d"+strings.Repeat("0", 3-len(itoa(i)))+itoa(i))
			}
		},
	}
	for name, mutate := range bad {
		c := comp("x", "x", "1.0.0")
		mutate(&c)
		if c.valid() {
			t.Errorf("%s accepted", name)
		}
	}
	tool := comp("x", "x", "1.0.0")
	tool.Distribution, tool.Artifact = DistToolOnly, ""
	if !tool.valid() {
		t.Error("tool without artifact refused")
	}
	edge := comp("x", "x", "1.0.0")
	edge.Commit = sum("c")
	edge.Source = "https://" + strings.Repeat("a", 192)
	edge.Modification, edge.Patch = ModPatched, sum("p")
	edge.Telemetry, edge.TelemetryEvidence = TelemetryOptional, sum("e")
	for i := 0; i < 256; i++ {
		edge.Dependencies = append(edge.Dependencies, "d"+strings.Repeat("0", 3-len(itoa(i)))+itoa(i))
	}
	if !edge.valid() {
		t.Error("edges refused")
	}
	for _, v := range []string{"1.0.0", "1.0.0-linux", "2024.01", "v3", "x1", "1.x2"} {
		if !pinned(v) {
			t.Errorf("version %s refused", v)
		}
	}
	for _, v := range []string{"1.x", "1.X.0", "x", "latest", "MAIN", "Head", "stable", "nightly", "dev", "trunk", "master"} {
		if pinned(v) {
			t.Errorf("version %s accepted", v)
		}
	}
	f := setup(t)
	unpinnedLabels := comp("x", "x", "latest")
	reason(t, "unpinnedLabels", second(f.r.Propose(Change{Kind: ChangeComponent, Component: unpinnedLabels}, ann)), ReasonUnpinnedVersion)
	unknown := comp("x", "x", "1.0.0")
	unknown.License = "WTFPL"
	reason(t, "unknown license", second(f.r.Propose(Change{Kind: ChangeComponent, Component: unknown}, ann)), ReasonUnknownLicense)
	for _, c := range []Change{{Kind: "other"}, {Kind: ChangeComponent}, {Kind: ChangeAdopt}, {Kind: ChangeAdopt, Decision: Decision{Component: "a", Reason: "x"}},
		{Kind: ChangeReject, Decision: Decision{Component: "a"}}, {Kind: ChangeRevoke, Decision: Decision{Reason: "x"}}} {
		if _, err := clean(c); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s %+v accepted", c.Kind, c.Decision)
		}
	}
	if !validText("ok") || validText("") || validText(strings.Repeat("a", 201)) || validText("a\x01") || validText("\xff") || validText("a\u0085") {
		t.Error("text")
	}
	for _, id := range []string{"A", "Z", "a", "z", "0", "9", ".", strings.Repeat("a", 128)} {
		if !validID(id) {
			t.Errorf("id %q", id)
		}
	}
	for _, id := range []string{"", "!a", "a b", "a@", "a[", "a`", "a{", strings.Repeat("a", 129)} {
		if validID(id) {
			t.Errorf("id %q accepted", id)
		}
	}
	if hexOf("ab", 4) || hexOf("abcg", 4) || !hexOf("abcd", 2, 4) {
		t.Error("hexOf")
	}
}

func TestProposals(t *testing.T) {
	f := setup(t)
	c := Change{Kind: ChangeComponent, Component: comp("a", "a", "1.0.0")}
	reason(t, "assistant", second(f.r.Propose(c, bot)), ReasonAINotAllowed)
	reason(t, "system", second(f.r.Propose(c, sys)), ReasonNotHuman)
	expect(t, "actor", second(f.r.Propose(c, Actor{Kind: KindHuman})), ErrInvalid)
	expect(t, "actor kind", second(f.r.Propose(c, Actor{ID: "x", Kind: "robot"})), ErrInvalid)
	p, err := f.r.Propose(c, ann)
	f.must(err)
	if again, _ := f.r.Propose(c, ann); again.Number != p.Number {
		t.Error("repeat")
	}
	q, _ := f.r.Propose(c, cat)
	reason(t, "self", second(f.r.Approve(p.Number, ann)), ReasonSelfApproval)
	reason(t, "approve system", second(f.r.Approve(p.Number, sys)), ReasonNotHuman)
	expect(t, "missing", second(f.r.Approve(99, bob)), ErrNotFound)
	f.must(second(f.r.Approve(p.Number, bob)))
	if again, err := f.r.Approve(p.Number, bob); err != nil || again.State != ProposalApplied {
		t.Error("repeat approve")
	}
	reason(t, "applied", second(f.r.Approve(p.Number, cat)), ReasonNotPending)
	expect(t, "recheck", second(f.r.Approve(q.Number, bob)), ErrConflict)
	reason(t, "withdraw other", second(f.r.Withdraw(q.Number, ann)), ReasonNotProposer)
	reason(t, "withdraw applied", second(f.r.Withdraw(p.Number, ann)), ReasonNotPending)
	reason(t, "withdraw assistant", second(f.r.Withdraw(q.Number, bot)), ReasonAINotAllowed)
	expect(t, "withdraw missing", second(f.r.Withdraw(99, ann)), ErrNotFound)
	if w, err := f.r.Withdraw(q.Number, cat); err != nil || w.State != ProposalWithdrawn {
		t.Error("withdraw")
	}
	expect(t, "withdrawn", second(f.r.Proposal(q.Number)), ErrNotFound)
	d := comp("b", "b", "1.0.0")
	d.Dependencies = []string{"a"}
	r, err := f.r.Propose(Change{Kind: ChangeComponent, Component: d}, ann)
	f.must(err)
	r.Change.Component.Dependencies[0] = "zz"
	if got, _ := f.r.Proposal(r.Number); got.Change.Component.Dependencies[0] != "a" {
		t.Error("proposal shares memory")
	}
	g := setup(t)
	people := []Actor{ann, bob, cat, {ID: "dan", Kind: KindHuman}}
	for _, who := range people {
		for n := 0; n < 250; n++ {
			g.must(second(g.r.Propose(Change{Kind: ChangeComponent, Component: comp(who.ID+itoa(n), who.ID+itoa(n), "1.0.0")}, who)))
		}
		expect(t, "mine", second(g.r.Propose(Change{Kind: ChangeComponent, Component: comp(who.ID+"-x", "y", "1.0.0")}, who)), ErrLimit)
	}
	expect(t, "total", second(g.r.Propose(Change{Kind: ChangeComponent, Component: comp("e", "e", "1.0.0")}, Actor{ID: "eve", Kind: KindHuman})), ErrLimit)
	h := setup(t)
	first := h.approve(Change{Kind: ChangeComponent, Component: comp("c0", "c0", "1.0.0")})
	for n := 2; n < 1000; n++ {
		h.r.done = append(h.r.done, uint64(100000+n))
	}
	kept := h.approve(Change{Kind: ChangeComponent, Component: comp("c1", "c1", "1.0.0")})
	if _, err := h.r.Proposal(first.Number); err != nil {
		t.Fatal("pruned early")
	}
	h.approve(Change{Kind: ChangeComponent, Component: comp("c2", "c2", "1.0.0")})
	expect(t, "pruned", second(h.r.Proposal(first.Number)), ErrNotFound)
	if _, err := h.r.Proposal(kept.Number); err != nil || len(h.r.done) != 1000 {
		t.Errorf("prune one %v %d", err, len(h.r.done))
	}
}

func TestUniquenessAndSupersedes(t *testing.T) {
	f := setup(t)
	f.approve(Change{Kind: ChangeComponent, Component: comp("a1", "Foo", "1.0.0")})
	expect(t, "id", f.try(Change{Kind: ChangeComponent, Component: comp("a1", "Bar", "2.0.0")}), ErrConflict)
	same := comp("a2", "foo", "1.0.0")
	expect(t, "name version", f.try(Change{Kind: ChangeComponent, Component: same}), ErrConflict)
	commit := comp("a3", "FOO", "1.0.1")
	commit.Commit = comp("a1", "Foo", "1.0.0").Commit
	expect(t, "name commit", f.try(Change{Kind: ChangeComponent, Component: commit}), ErrConflict)
	f.approve(Change{Kind: ChangeReject, Decision: Decision{Component: "a1", Reason: "GPL dependency"}})
	expect(t, "after rejection without link", f.try(Change{Kind: ChangeComponent, Component: same}), ErrConflict)
	same.Supersedes = "nope"
	expect(t, "wrong link", f.try(Change{Kind: ChangeComponent, Component: same}), ErrConflict)
	same.Supersedes = "a1"
	f.approve(Change{Kind: ChangeComponent, Component: same})
	if v, _ := f.r.Component("a2"); v.Component.Supersedes != "a1" || v.State != StateCandidate {
		t.Errorf("link %+v", v)
	}
	if v, _ := f.r.Component("a1"); v.State != StateRejected || v.Reason != "GPL dependency" {
		t.Errorf("rejected %+v", v)
	}
	fresh := comp("b1", "bar", "1.0.0")
	fresh.Supersedes = "a1"
	expect(t, "link without earlier", f.try(Change{Kind: ChangeComponent, Component: fresh}), ErrConflict)
	f.approve(Change{Kind: ChangeAdopt, Decision: Decision{Component: "a2"}})
	f.approve(Change{Kind: ChangeRevoke, Decision: Decision{Component: "a2", Reason: "wrong licence"}})
	third := comp("a4", "foo", "1.0.0")
	third.Commit = comp("a1", "Foo", "1.0.0").Commit
	third.Supersedes = "a1"
	expect(t, "must name latest", f.try(Change{Kind: ChangeComponent, Component: third}), ErrConflict)
	third.Supersedes = "a2"
	f.approve(Change{Kind: ChangeComponent, Component: third})
	if list := f.r.ByName("FOO"); len(list) != 3 || list[0].Component.ID != "a1" || list[2].Component.ID != "a4" {
		t.Errorf("by name %+v", list)
	}
	if list := f.r.ByName("nope"); list == nil || len(list) != 0 {
		t.Error("empty by name")
	}
	expect(t, "component missing", second(f.r.Component("nope")), ErrNotFound)
	read, _ := f.r.Component("a4")
	read.Component.Dependencies = append(read.Component.Dependencies, "x")
	if again, _ := f.r.Component("a4"); len(again.Component.Dependencies) != 0 {
		t.Error("read shares memory")
	}
	dep := comp("d1", "dep", "1.0.0")
	dep.Dependencies = []string{"missing"}
	expect(t, "dep missing", f.try(Change{Kind: ChangeComponent, Component: dep}), ErrNotFound)
	g := setup(t)
	for len(g.r.components) < 9999 {
		g.r.components["x"+itoa(len(g.r.components))] = &component{}
	}
	g.approve(Change{Kind: ChangeComponent, Component: comp("fit", "fit", "1.0.0")})
	expect(t, "cap", g.try(Change{Kind: ChangeComponent, Component: comp("over", "over", "1.0.0")}), ErrLimit)
}
