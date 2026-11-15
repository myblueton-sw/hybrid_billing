package installer

import (
	"encoding/hex"
	"strings"
	"testing"
	"time"
)

const previewAnswer = "5657d39979122cf087325ebb57ae71ded6a37fadee357ec937378934727c7e2b"

func TestHashKnownAnswers(t *testing.T) {
	sum := codeHash(codePrefix + strings.Repeat("A", 43))
	if hex.EncodeToString(sum[:]) != "c0452cfd8c46411136583ed093a0aff576e4b825c340dddb1d40e65b6adcadf6" {
		t.Errorf("code hash %x", sum)
	}
	f := standard(t)
	f.observe("s1", p1, empty())
	v := f.preview("s1", "1.0.0")
	if v.Hash != previewAnswer {
		t.Errorf("preview hash %s", v.Hash)
	}
	s, rel := f.r.sites["s1"], f.r.releases["1.0.0"]
	changed := map[string]func(*PreviewView){
		"data":        func(p *PreviewView) { p.Data = nil },
		"permissions": func(p *PreviewView) { p.Permissions = nil },
		"effect":      func(p *PreviewView) { p.Data[0].Effect = EffectUntouched },
		"permission":  func(p *PreviewView) { p.Permissions[0] = "x" },
		"services":    func(p *PreviewView) { p.Services = nil },
		"action":      func(p *PreviewView) { p.Ports[0].Action = "x" },
		"conflicts":   func(p *PreviewView) { p.Conflicts = []Item{{Kind: "config"}} },
		"volumes":     func(p *PreviewView) { p.Volumes[0].Target = "x" },
	}
	for name, change := range changed {
		c := v
		c.Data = append([]DataEffect(nil), v.Data...)
		c.Permissions = append([]string(nil), v.Permissions...)
		c.Ports = append([]Item(nil), v.Ports...)
		c.Volumes = append([]Item(nil), v.Volumes...)
		change(&c)
		if f.r.hash(s, rel, c) == v.Hash {
			t.Errorf("%s does not change the hash", name)
		}
	}
}

func TestResumeNeedsNewObservation(t *testing.T) {
	f := standard(t)
	run := f.started("s1", p1)
	f.must(second(f.r.Interrupt("s1", run.ID, "network lost", ann)))
	reason(t, "old observation", second(f.r.Resume("s1", run.ID, ann)), ReasonObservationStale)
	if v, _ := f.r.Run("s1", run.ID); v.State != RunInterrupted || v.Note != "network lost" {
		t.Errorf("run %+v", v)
	}
	f.observe("s1", p1, empty())
	f.must(second(f.r.Resume("s1", run.ID, ann)))
	f.must(second(f.r.Interrupt("s1", run.ID, "again", ann)))
	if v, _ := f.r.Cancel("s1", run.ID, ann); v.Note != "" {
		t.Errorf("cancel kept the note %+v", v)
	}
	g := standard(t)
	r := g.started("s1", p1)
	g.must(second(g.r.Interrupt("s1", r.ID, "power", ann)))
	g.approve(Change{Kind: ChangeConfig, Config: SiteConfig{Site: "s1", Version: "1.0.0"}})
	if v, _ := g.r.Run("s1", r.ID); v.Note != "" || v.State != RunInvalidated {
		t.Errorf("invalidation kept the note %+v", v)
	}
	expect(t, "control text", second(f.r.Interrupt("s1", run.ID, "a\nb", ann)), ErrInvalid)
}

func TestTranslatedAddresses(t *testing.T) {
	for _, text := range []string{"::/48", "::ffff:0:0/96", "64:ff9b::/96", "::/96", "0:0:0:0:0:ffff::/96", "64:ff9b:1::/48", "::8000:0/97", "64:ff9b::8000:0/97", "64:ff9b:1:8000::/49", "::ffff:8000:0/97"} {
		if _, ok := validRange(text); ok {
			t.Errorf("%s accepted", text)
		}
	}
	if _, ok := validRange("2001:db8::/48"); !ok {
		t.Error("ordinary range refused")
	}
	f := standard(t)
	f.approve(Change{Kind: ChangeDestination, Destination: Destination{Site: "s1", ID: "erp", Host: "erp.internal", Port: 443, Ranges: []string{"2001:db8::/48"}}})
	expect(t, "mapped", second(f.r.Report("s1", "erp", Result{Kind: DiagnoseTLS, Failure: "expired", Address: "::ffff:10.9.9.9"}, p1)), ErrInvalid)
	f.must(second(f.r.Report("s1", "erp", Result{Kind: DiagnoseTLS, Failure: "expired", Address: "2001:db8::5"}, p1)))
	expect(t, "proxy without via", second(f.r.Report("s1", "erp", Result{Kind: DiagnoseProxy, Failure: "proxy_auth", Address: "2001:db8::5"}, p1)), ErrInvalid)
}

func TestEveryConflictListed(t *testing.T) {
	f := standard(t)
	f.approve(Change{Kind: ChangeConfig, Config: SiteConfig{Site: "s1", Version: "1.0.0",
		Ports:   []PortOverride{{Service: "worker", Port: Port{Number: 9000, Protocol: ProtocolTCP}, Number: 8443}},
		Volumes: []VolumeOverride{{Volume: "config", Host: "ledger"}}}})
	o := empty()
	o.Ports = []Port{{Number: 8443, Protocol: ProtocolTCP}}
	o.Volumes = []string{"ledger"}
	f.observe("s1", p1, o)
	v := f.preview("s1", "1.0.0")
	want := "volume:ledger:create:existing_volume,volume:ledger:create:existing_volume,port:api:8443/tcp:bind:port_duplicate,port:worker:8443/tcp:bind:port_duplicate," +
		"port:api:8443/tcp:bind:port_in_use,port:worker:8443/tcp:bind:port_in_use,volume:ledger:create:volume_duplicate,volume:ledger:create:volume_duplicate"
	if items(v.Conflicts) != want {
		t.Errorf("conflicts %s", items(v.Conflicts))
	}
	if len(v.Data) != 2 || v.Data[0].Effect != EffectUntouched || v.Data[1].Effect != EffectUntouched {
		t.Errorf("data %+v", v.Data)
	}
	if items(v.Ports) != "port:api:8443/tcp:bind:port_duplicate,port:worker:8443/tcp:bind:port_duplicate,port:worker:9000/udp:bind:" {
		t.Errorf("ports %s", items(v.Ports))
	}
}

func TestOverrideMatchesServiceAndPort(t *testing.T) {
	f := standard(t)
	f.approve(Change{Kind: ChangeConfig, Config: SiteConfig{Site: "s1", Version: "1.0.0",
		Ports: []PortOverride{{Service: "worker", Port: Port{Number: 9000, Protocol: ProtocolTCP}, Number: 9100}}}})
	f.observe("s1", p1, empty())
	if got := items(f.preview("s1", "1.0.0").Ports); got != "port:api:8443/tcp:bind:,port:worker:9100/tcp:bind:,port:worker:9000/udp:bind:" {
		t.Errorf("ports %s", got)
	}
}

func TestScopedRetirement(t *testing.T) {
	f := standard(t)
	f.observe("s1", p1, empty())
	f.observe("s2", p2, empty())
	f.accept("s1", "1.0.0")
	kept := f.accept("s2", "1.0.0")
	f.approve(Change{Kind: ChangeProbe, Site: Site{ID: "s1", Probe: "probe-new"}})
	f.must(second(f.r.StartRun("s2", kept.Number, ann)))
}

func TestMarksAffectOnlyTheirRelease(t *testing.T) {
	f := standard(t)
	f.approve(signed(fixture("1.1.0")))
	run := f.started("s1", p1)
	f.observe("s2", p2, empty())
	other := f.accept("s2", "1.0.0")
	f.must(second(f.r.RevokeRelease("1.1.0", "Bad", ann)))
	if v, _ := f.r.Run("s1", run.ID); v.State != RunOpen {
		t.Errorf("other release run %+v", v)
	}
	f.must(second(f.r.StartRun("s2", other.Number, ann)))
	g := standard(t)
	done := g.started("s1", p1)
	g.must(second(g.r.Cancel("s1", done.ID, ann)))
	g.must(second(g.r.RevokeRelease("1.0.0", "Bad", ann)))
	if v, _ := g.r.Run("s1", done.ID); v.State != RunCancelled {
		t.Errorf("closed run changed %+v", v)
	}
}

func TestFormBoundaries(t *testing.T) {
	for _, host := range []string{"z", "x0.z", "zz.a", "a0b.c"} {
		if !validHost(host) {
			t.Errorf("%q refused", host)
		}
	}
	expect(t, "prefix", second(standard(t).r.RegisterAdmin("hbx_"+strings.Repeat("A", 43), cat)), ErrInvalid)
	full := fixture("1.0.0")
	full.Services[0].Ports[0].Number = 65535
	if !full.valid() {
		t.Error("port 65535 refused")
	}
	cases := map[string]func(*Release){
		"many artifacts":  func(r *Release) { r.Artifacts = append(r.Artifacts, artifacts(maxArtifacts-1)...) },
		"artifact twice":  func(r *Release) { r.Artifacts = []Artifact{r.Artifacts[0], r.Artifacts[0], r.Artifacts[1]} },
		"volume twice":    func(r *Release) { r.Volumes = []Volume{r.Volumes[0], r.Volumes[0], r.Volumes[1]} },
		"no services":     func(r *Release) { r.Services, r.Volumes = nil, nil },
		"many services":   func(r *Release) { r.Services = append(r.Services, extraServices(maxServices-1)...) },
		"many volumes":    func(r *Release) { r.Services, r.Volumes = packed(5), volumes(maxVolumes+1) },
		"arch tie broken": func(r *Release) { r.Platforms = []Platform{{OS: "linux", Arch: "b"}, {OS: "linux", Arch: "b"}} },
	}
	for name, change := range cases {
		rel := copyRelease(fixture("1.0.0"))
		change(&rel)
		if rel.valid() {
			t.Errorf("%s accepted", name)
		}
	}
	good := copyRelease(fixture("1.0.0"))
	good.Platforms = []Platform{{OS: "linux", Arch: "amd64"}, {OS: "linux", Arch: "arm64"}}
	good.Services, good.Volumes = packed(4), volumes(maxVolumes)
	if !good.valid() {
		t.Error("full volumes or arch order refused")
	}
	one := Release{Services: []Service{{Name: "a"}}, Volumes: []Volume{{Name: "v"}}}
	if strings.Join(one.permissions(), ",") != "create_volume,run_service,write_config" {
		t.Errorf("one volume %v", one.permissions())
	}
	c := SiteConfig{Site: "s1", Version: "1.0.0", Volumes: []VolumeOverride{{Volume: "a", Host: "x"}, {Volume: "a", Host: "y"}}}
	if c.valid() {
		t.Error("duplicate volume override accepted")
	}
	ports := SiteConfig{Site: "s1", Version: "1.0.0"}
	for index := 0; index <= maxOverrides; index++ {
		ports.Ports = append(ports.Ports, PortOverride{Service: "api", Port: Port{Number: index + 1, Protocol: ProtocolTCP}, Number: 1})
	}
	if ports.valid() {
		t.Error("too many port overrides accepted")
	}
	vols := SiteConfig{Site: "s1", Version: "1.0.0", Volumes: []VolumeOverride{}}
	for _, name := range names(maxOverrides + 1) {
		vols.Volumes = append(vols.Volumes, VolumeOverride{Volume: name, Host: "h"})
	}
	if vols.valid() {
		t.Error("too many volume overrides accepted")
	}
	f := standard(t)
	for name, o := range map[string]Observation{"form": {}, "platform": {Platform: Platform{OS: "x", Arch: "y"}, Complete: true}, "partial": {Platform: linux}} {
		if seq, _ := f.r.Observe("s1", o, p1); seq != 0 {
			t.Errorf("%s returned %d", name, seq)
		}
	}
	if seq, _ := f.r.Observe("s1", empty(), p2); seq != 0 {
		t.Error("foreign probe returned a sequence")
	}
	g := setupWith(t, Limits{Keys: 2, KeyLife: limits.KeyLife, Releases: 2, Sites: 2, Destinations: 1, Reports: 1, Runs: 1, ObservationLife: time.Hour, BootstrapLife: time.Hour, Proposals: 5})
	g.must(second(g.r.Propose(keyChange("a"), ann)))
	g.must(second(g.r.Propose(keyChange("b"), ann)))
	expect(t, "per proposer of five", second(g.r.Propose(keyChange("c"), ann)), ErrLimit)
}

func extraServices(n int) []Service {
	out := []Service{}
	for _, name := range names(n) {
		out = append(out, Service{Name: "x" + name, Artifact: "api"})
	}
	return out
}

func packed(n int) []Service {
	out := []Service{}
	all := names(maxVolumes + 1)
	for index, name := range names(n) {
		end := (index + 1) * maxServiceVolume
		if end > len(all) {
			end = len(all)
		}
		begin := index * maxServiceVolume
		if begin > end {
			begin = end
		}
		out = append(out, Service{Name: name, Artifact: "api", Volumes: append([]string(nil), all[begin:end]...)})
	}
	return out
}
