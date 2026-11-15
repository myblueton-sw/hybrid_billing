package installer

import (
	"crypto/ed25519"
	"strings"
	"testing"
	"time"
)

func TestProposalFlow(t *testing.T) {
	f := setup(t)
	c := keyChange("k1")
	c.Site = Site{ID: "ignored"}
	reason(t, "assistant", second(f.r.Propose(c, bot)), ReasonAINotAllowed)
	reason(t, "system", second(f.r.Propose(c, p1)), ReasonNotHuman)
	expect(t, "actor", second(f.r.Propose(c, Actor{ID: "x", Kind: "robot"})), ErrInvalid)
	expect(t, "kind", second(f.r.Propose(Change{Kind: "admin"}, ann)), ErrInvalid)
	p, err := f.r.Propose(c, ann)
	f.must(err)
	if p.Change.Site.ID != "" || p.State != ProposalPending {
		t.Errorf("proposal %+v", p)
	}
	again, _ := f.r.Propose(c, ann)
	other, _ := f.r.Propose(c, cat)
	if again.Number != p.Number || other.Number == p.Number {
		t.Error("repeat numbering")
	}
	reason(t, "self", second(f.r.Approve(p.Number, ann)), ReasonSelfApproval)
	expect(t, "missing", second(f.r.Approve(99, bob)), ErrNotFound)
	reason(t, "approve assistant", second(f.r.Approve(p.Number, bot)), ReasonAINotAllowed)
	done, err := f.r.Approve(p.Number, bob)
	f.must(err)
	if repeat, err := f.r.Approve(p.Number, bob); err != nil || repeat.Approver != done.Approver {
		t.Error("repeat approval")
	}
	reason(t, "third", second(f.r.Approve(p.Number, cat)), ReasonNotPending)
	expect(t, "stale duplicate", second(f.r.Approve(other.Number, bob)), ErrConflict)
	reason(t, "withdraw applied", second(f.r.Withdraw(p.Number, ann)), ReasonNotPending)
	reason(t, "withdraw other", second(f.r.Withdraw(other.Number, ann)), ReasonNotProposer)
	reason(t, "withdraw assistant", second(f.r.Withdraw(other.Number, bot)), ReasonAINotAllowed)
	expect(t, "withdraw missing", second(f.r.Withdraw(99, cat)), ErrNotFound)
	gone, err := f.r.Withdraw(other.Number, cat)
	if err != nil || gone.State != ProposalWithdrawn {
		t.Errorf("withdrawn %+v %v", gone, err)
	}
	expect(t, "view withdrawn", second(f.r.Proposal(other.Number)), ErrNotFound)
	if view, err := f.r.Proposal(p.Number); err != nil || view.Approver != "bob" {
		t.Errorf("view %+v %v", view, err)
	}
	expect(t, "unknown site scope", second(f.r.Propose(Change{Kind: ChangeProbe, Site: Site{ID: "s9", Probe: "x"}}, ann)), ErrNotFound)
}

func TestProposalCaps(t *testing.T) {
	l := limits
	l.Proposals = 4
	f := setupWith(t, l)
	f.site("s1", "probe-1")
	key := func(name string, actor Actor) error { return second(f.r.Propose(keyChange(name), actor)) }
	f.must(key("a", ann))
	expect(t, "per proposer", key("b", ann), ErrLimit)
	f.must(key("b", bob))
	f.must(key("c", cat))
	f.must(key("d", Actor{ID: "dan", Kind: KindHuman}))
	expect(t, "scope", key("e", Actor{ID: "eve", Kind: KindHuman}), ErrLimit)
	f.must(second(f.r.Propose(Change{Kind: ChangeProbe, Site: Site{ID: "s1", Probe: "probe-x"}}, ann)))
	g := setupWith(t, Limits{Keys: 1, KeyLife: time.Hour * 10000, Releases: 1, Sites: 1, Destinations: 1, Reports: 1, Runs: 1, ObservationLife: time.Minute, BootstrapLife: time.Minute, Proposals: 2})
	var numbers []uint64
	for _, id := range []string{"a", "b", "c", "d"} {
		g.approve(Change{Kind: ChangeKey, Key: keyChange("k" + id).Key})
		g.must(g.r.RetireKey("k"+id, ann))
		numbers = append(numbers, g.r.counter)
	}
	expect(t, "pruned", second(g.r.Proposal(numbers[0])), ErrNotFound)
	expect(t, "pruned next", second(g.r.Proposal(numbers[1])), ErrNotFound)
	g.must(second(g.r.Proposal(numbers[2])))
	g.must(second(g.r.Proposal(numbers[3])))
}

func TestKeys(t *testing.T) {
	l := limits
	l.Keys = 1
	f := setupWith(t, l)
	forms := map[string]func(*ReleaseKey){
		"version": func(k *ReleaseKey) { k.Version = "" },
		"zero":    func(k *ReleaseKey) { k.NotBefore = time.Time{} },
		"order":   func(k *ReleaseKey) { k.NotAfter = k.NotBefore },
		"proof":   func(k *ReleaseKey) { k.Proof = k.Proof[:3] },
	}
	for name, change := range forms {
		c := copyChange(keyChange("k1"))
		change(&c.Key)
		expect(t, name, f.try(c), ErrInvalid)
	}
	w := copyChange(keyChange("k1"))
	w.Key.Public = hexBytes(smallOrder[0])
	reason(t, "weak", f.try(w), ReasonWeakKey)
	long := copyChange(keyChange("k1"))
	long.Key.NotAfter = long.Key.NotBefore.Add(l.KeyLife + time.Second)
	reason(t, "life", f.try(long), ReasonKeyLife)
	wrong := copyChange(keyChange("k1"))
	wrong.Key.Proof = ed25519.Sign(private("other"), challenge("k1"))
	reason(t, "proof", f.try(wrong), ReasonKeyMismatch)
	edge := copyChange(keyChange("k1"))
	edge.Key.NotAfter = edge.Key.NotBefore.Add(l.KeyLife)
	f.approve(edge)
	expect(t, "reuse", f.try(keyChange("k1")), ErrConflict)
	expect(t, "active", f.try(keyChange("k2")), ErrLimit)
	expect(t, "retire form", f.r.RetireKey("", ann), ErrInvalid)
	expect(t, "retire missing", f.r.RetireKey("k9", ann), ErrNotFound)
	reason(t, "retire assistant", f.r.RetireKey("k1", bot), ReasonAINotAllowed)
	f.must(f.r.RetireKey("k1", ann))
	f.approve(keyChange("k2"))
	g := setupWith(t, l)
	for index := 0; index < namesPerKey; index++ {
		name := "k" + strings.Repeat("x", index)
		g.approve(keyChange(name))
		g.must(g.r.RetireKey(name, ann))
	}
	expect(t, "names", g.try(keyChange("last")), ErrLimit)
}

func TestReleases(t *testing.T) {
	l := limits
	l.Releases = 2
	f := setupWith(t, l)
	expect(t, "unknown key", f.try(signed(fixture("1.0.0"))), ErrNotFound)
	f.approve(keyChange("k1"))
	expect(t, "form", f.try(Change{Kind: ChangeRelease, Release: Release{}}), ErrInvalid)
	short := signed(fixture("1.0.0"))
	short.Signature = short.Signature[:3]
	expect(t, "signature form", f.try(short), ErrInvalid)
	tampered := signed(fixture("1.0.0"))
	tampered.Release.Artifacts[0].Digest = sum("evil")
	reason(t, "tampered", f.try(tampered), ReasonSignatureInvalid)
	f.approve(signed(fixture("1.0.0")))
	expect(t, "immutable", f.try(signed(fixture("1.0.0"))), ErrConflict)
	rel, mark, err := f.r.Release("1.0.0")
	if err != nil || mark != nil || rel.Artifacts[0].Digest != sum("api1.0.0") {
		t.Fatalf("release %+v %v %v", rel, mark, err)
	}
	rel.Artifacts[0].Name = "x"
	if again, _, _ := f.r.Release("1.0.0"); again.Artifacts[0].Name != "api" {
		t.Error("view shares storage")
	}
	expect(t, "view form", third(f.r.Release("x")), ErrInvalid)
	expect(t, "view missing", third(f.r.Release("9.0.0")), ErrNotFound)
	f.approve(signed(fixture("1.1.0")))
	expect(t, "room", f.try(signed(fixture("1.2.0"))), ErrLimit)
	g := setup(t)
	later := keyChange("k1")
	later.Key.NotBefore, later.Key.NotAfter = start.Add(time.Hour), start.Add(2*time.Hour)
	g.approve(later)
	reason(t, "before window", g.try(signed(fixture("1.0.0"))), ReasonKeyInactive)
	g.advance(2 * time.Hour)
	g.approve(signed(fixture("1.0.0")))
	g.advance(time.Nanosecond)
	reason(t, "after window", g.try(signed(fixture("1.0.1"))), ReasonKeyInactive)
	h := setup(t)
	h.approve(keyChange("k1"))
	h.must(h.r.RetireKey("k1", ann))
	reason(t, "retired", h.try(signed(fixture("1.0.0"))), ReasonKeyInactive)
}

func TestSitesAndProbes(t *testing.T) {
	l := limits
	l.Sites = 2
	f := setupWith(t, l)
	for name, s := range map[string]Site{"id": {Tenant: "t", Platform: linux, Probe: "p"}, "tenant": {ID: "s", Platform: linux, Probe: "p"},
		"platform": {ID: "s", Tenant: "t", Probe: "p"}, "probe": {ID: "s", Tenant: "t", Platform: linux}} {
		expect(t, name, f.try(Change{Kind: ChangeSite, Site: s}), ErrInvalid)
	}
	f.site("s1", "probe-1")
	expect(t, "twice", f.try(Change{Kind: ChangeSite, Site: Site{ID: "s1", Tenant: "t", Platform: linux, Probe: "probe-9"}}), ErrConflict)
	reason(t, "probe taken", f.try(Change{Kind: ChangeSite, Site: Site{ID: "s2", Tenant: "t", Platform: linux, Probe: "probe-1"}}), ReasonProbeTaken)
	f.site("s2", "probe-2")
	expect(t, "room", f.try(Change{Kind: ChangeSite, Site: Site{ID: "s3", Tenant: "t", Platform: linux, Probe: "probe-3"}}), ErrLimit)
	s, err := f.r.Site("s1")
	if err != nil || s.Tenant != "tenant-s1" || s.Probe != "probe-1" {
		t.Errorf("site %+v %v", s, err)
	}
	expect(t, "view form", second(f.r.Site("")), ErrInvalid)
	expect(t, "view missing", second(f.r.Site("s9")), ErrNotFound)
	rotate := func(probe string) Change {
		return Change{Kind: ChangeProbe, Site: Site{ID: "s1", Probe: probe, Tenant: "evil", Platform: linux}}
	}
	expect(t, "rotate form", f.try(Change{Kind: ChangeProbe, Site: Site{ID: "s1"}}), ErrInvalid)
	reason(t, "own", f.try(rotate("probe-1")), ReasonProbeTaken)
	reason(t, "other", f.try(rotate("probe-2")), ReasonProbeTaken)
	p := f.approve(rotate("probe-3"))
	if p.Change.Site.Tenant != "" {
		t.Error("probe change carried a tenant")
	}
	s, _ = f.r.Site("s1")
	if s.Probe != "probe-3" || s.Tenant != "tenant-s1" {
		t.Errorf("rotated %+v", s)
	}
	reason(t, "tombstone", f.try(rotate("probe-1")), ReasonProbeTaken)
	for index := 3; index <= probesPerSite; index++ {
		f.approve(rotate("probe-" + strings.Repeat("r", index)))
	}
	expect(t, "cap", f.try(rotate("probe-last")), ErrLimit)
}

func TestConfigs(t *testing.T) {
	f := standard(t)
	cfg := func(version string) Change {
		return Change{Kind: ChangeConfig, Config: SiteConfig{Site: "s1", Version: version,
			Ports:   []PortOverride{{Service: "api", Port: Port{Number: 8443, Protocol: ProtocolTCP}, Number: 9443}},
			Volumes: []VolumeOverride{{Volume: "ledger", Host: "ledger-prod"}}}}
	}
	expect(t, "form", f.try(Change{Kind: ChangeConfig, Config: SiteConfig{Site: "s1", Version: "x"}}), ErrInvalid)
	expect(t, "missing release", f.try(cfg("9.0.0")), ErrNotFound)
	bad := cfg("1.0.0")
	bad.Config.Ports[0].Service = "worker"
	reason(t, "undeclared port", f.try(bad), ReasonNotDeclared)
	proto := cfg("1.0.0")
	proto.Config.Ports[0].Port.Protocol = ProtocolUDP
	reason(t, "undeclared protocol", f.try(proto), ReasonNotDeclared)
	vol := cfg("1.0.0")
	vol.Config.Volumes[0].Volume = "other"
	reason(t, "undeclared volume", f.try(vol), ReasonNotDeclared)
	p := f.approve(cfg("1.0.0"))
	p.Change.Config.Ports[0].Number = 1
	if f.r.sites["s1"].config.Ports[0].Number != 9443 || f.r.sites["s1"].revision != 1 {
		t.Error("config stored wrongly")
	}
}

func TestDestinations(t *testing.T) {
	l := limits
	l.Destinations = 2
	f := setupWith(t, l)
	f.site("s1", "probe-1")
	dest := func(id, host string, port int) Change {
		return Change{Kind: ChangeDestination, Destination: Destination{Site: "s1", ID: id, Host: host, Port: port, Ranges: []string{"10.1.0.0/16"}}}
	}
	expect(t, "form", f.try(dest("erp", "ERP", 443)), ErrInvalid)
	f.approve(dest("erp", "erp.internal", 443))
	expect(t, "id twice", f.try(dest("erp", "crm.internal", 443)), ErrConflict)
	expect(t, "target twice", f.try(dest("erp2", "erp.internal", 443)), ErrConflict)
	f.approve(dest("erp2", "erp.internal", 8443))
	expect(t, "room", f.try(dest("crm", "crm.internal", 443)), ErrLimit)
	got, err := f.r.CheckTarget("s1", "erp", p1)
	if err != nil || got.Host != "erp.internal" || got.Ranges[0] != "10.1.0.0/16" {
		t.Errorf("target %+v %v", got, err)
	}
	got.Ranges[0] = "x"
	if again, _ := f.r.CheckTarget("s1", "erp", p1); again.Ranges[0] != "10.1.0.0/16" {
		t.Error("target shares ranges")
	}
}
