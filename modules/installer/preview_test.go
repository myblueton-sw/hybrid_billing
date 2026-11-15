package installer

import (
	"strings"
	"testing"
	"time"
)

func items(list []Item) string {
	var out []string
	for _, i := range list {
		out = append(out, i.Kind+":"+i.Target+":"+i.Action+":"+i.Conflict)
	}
	return strings.Join(out, ",")
}

func TestObserve(t *testing.T) {
	f := standard(t)
	expect(t, "form", second(f.r.Observe("s1", Observation{}, p1)), ErrInvalid)
	expect(t, "actor", second(f.r.Observe("s1", empty(), Actor{ID: "x"})), ErrInvalid)
	expect(t, "site form", second(f.r.Observe("", empty(), p1)), ErrInvalid)
	expect(t, "missing", second(f.r.Observe("s9", empty(), p1)), ErrNotFound)
	reason(t, "foreign probe", second(f.r.Observe("s2", empty(), p1)), ReasonNotProbe)
	reason(t, "human", second(f.r.Observe("s1", empty(), Actor{ID: "probe-1", Kind: KindHuman})), ReasonNotProbe)
	other := empty()
	other.Platform = Platform{OS: "linux", Arch: "arm64"}
	reason(t, "platform", second(f.r.Observe("s1", other, p1)), ReasonPlatformMismatch)
	partial := empty()
	partial.Complete = false
	reason(t, "incomplete", second(f.r.Observe("s1", partial, p1)), ReasonIncomplete)
	o := empty()
	o.Volumes = []string{"a"}
	seq, err := f.r.Observe("s1", o, p1)
	if err != nil || seq != 1 {
		t.Fatal(seq, err)
	}
	o.Volumes[0] = "z"
	if f.r.sites["s1"].observation.Volumes[0] != "a" {
		t.Error("observation shares storage")
	}
	if seq, _ := f.r.Observe("s1", empty(), p1); seq != 2 {
		t.Error("sequence")
	}
}

func TestPreviewClean(t *testing.T) {
	f := standard(t)
	expect(t, "form", second(f.r.Preview("s1", "x")), ErrInvalid)
	expect(t, "site form", second(f.r.Preview("", "1.0.0")), ErrInvalid)
	expect(t, "missing site", second(f.r.Preview("s9", "1.0.0")), ErrNotFound)
	expect(t, "missing release", second(f.r.Preview("s1", "9.0.0")), ErrNotFound)
	expect(t, "no observation", second(f.r.Preview("s1", "1.0.0")), ErrNotFound)
	o := empty()
	o.Granted = []string{PermissionRunService}
	f.observe("s1", p1, o)
	v := f.preview("s1", "1.0.0")
	if items(v.Services) != "service:api:create:,service:worker:create:" ||
		items(v.Ports) != "port:api:8443/tcp:bind:,port:worker:9000/tcp:bind:,port:worker:9000/udp:bind:" ||
		items(v.Volumes) != "volume:config:create:,volume:ledger:create:" || len(v.Conflicts) != 0 {
		t.Fatalf("preview %+v", v)
	}
	if len(v.Data) != 1 || v.Data[0] != (DataEffect{Volume: "ledger", Effect: EffectNew}) {
		t.Errorf("data %+v", v.Data)
	}
	if strings.Join(v.Permissions, ",") != "create_volume,listen_port,write_config" || v.Site != "s1" || v.Version != "1.0.0" || !validHex(v.Hash) {
		t.Errorf("permissions %+v", v)
	}
	f.advance(time.Minute)
	f.observe("s1", p1, o)
	if again := f.preview("s1", "1.0.0"); again.Hash != v.Hash {
		t.Error("an unchanged refresh changed the hash")
	}
	other := f.r.sites["s2"]
	other.observation, other.observed = f.r.sites["s1"].observation, f.r.sites["s1"].observed
	if f.preview("s2", "1.0.0").Hash == v.Hash {
		t.Error("hash ignores the site")
	}
	f.advance(10 * time.Minute)
	f.must(second(f.r.Preview("s1", "1.0.0")))
	f.advance(time.Nanosecond)
	reason(t, "stale", second(f.r.Preview("s1", "1.0.0")), ReasonObservationStale)
}

func TestPreviewHashInputs(t *testing.T) {
	base := func() world {
		f := standard(t)
		f.observe("s1", p1, empty())
		return f
	}
	f := base()
	h := f.preview("s1", "1.0.0").Hash
	steps := map[string]func(f world){
		"observation": func(f world) {
			o := empty()
			o.Granted = []string{PermissionWriteConfig}
			f.observe("s1", p1, o)
		},
		"tenant":   func(f world) { f.r.sites["s1"].view.Tenant = "other" },
		"probe":    func(f world) { f.r.sites["s1"].view.Probe = "other" },
		"revision": func(f world) { f.r.sites["s1"].revision++ },
		"digest":   func(f world) { f.r.releases["1.0.0"].digest = sum("x") },
		"unused":   func(f world) { f.r.sites["s1"].observation.Services = []string{"zz"} },
	}
	for name, step := range steps {
		g := base()
		step(g)
		if g.preview("s1", "1.0.0").Hash == h {
			t.Errorf("%s does not change the hash", name)
		}
	}
}

func TestPreviewConflicts(t *testing.T) {
	f := standard(t)
	o := empty()
	o.Ports = []Port{{Number: 8443, Protocol: ProtocolTCP}, {Number: 9000, Protocol: ProtocolTCP}}
	o.Services = []string{"api"}
	o.Volumes = []string{"ledger"}
	f.observe("s1", p1, o)
	v := f.preview("s1", "1.0.0")
	if items(v.Conflicts) != "volume:ledger:create:existing_volume,port:api:8443/tcp:bind:port_in_use,port:worker:9000/tcp:bind:port_in_use,service:api:create:service_exists" {
		t.Fatalf("conflicts %s", items(v.Conflicts))
	}
	if len(v.Data) != 1 || v.Data[0] != (DataEffect{Volume: "ledger", Effect: EffectUntouched}) {
		t.Errorf("data %+v", v.Data)
	}
	f.approve(Change{Kind: ChangeConfig, Config: SiteConfig{Site: "s1", Version: "1.0.0",
		Ports:   []PortOverride{{Service: "api", Port: Port{Number: 8443, Protocol: ProtocolTCP}, Number: 9000}, {Service: "worker", Port: Port{Number: 9000, Protocol: ProtocolTCP}, Number: 9001}},
		Volumes: []VolumeOverride{{Volume: "config", Host: "shared"}, {Volume: "ledger", Host: "shared"}}}})
	f.observe("s1", p1, o)
	v = f.preview("s1", "1.0.0")
	if items(v.Conflicts) != "port:api:9000/tcp:bind:port_in_use,service:api:create:service_exists,volume:shared:create:volume_duplicate,volume:shared:create:volume_duplicate" {
		t.Fatalf("override conflicts %s", items(v.Conflicts))
	}
	if len(v.Data) != 1 || v.Data[0] != (DataEffect{Volume: "shared", Effect: EffectNew}) {
		t.Errorf("duplicate data %+v", v.Data)
	}
	f.approve(Change{Kind: ChangeConfig, Config: SiteConfig{Site: "s1", Version: "1.0.0",
		Ports: []PortOverride{{Service: "worker", Port: Port{Number: 9000, Protocol: ProtocolTCP}, Number: 8443}}}})
	f.observe("s1", p1, empty())
	v = f.preview("s1", "1.0.0")
	if items(v.Conflicts) != "port:api:8443/tcp:bind:port_duplicate,port:worker:8443/tcp:bind:port_duplicate" {
		t.Fatalf("duplicates %s", items(v.Conflicts))
	}
	f.approve(signed(fixture("1.1.0")))
	f.observe("s1", p1, empty())
	v = f.preview("s1", "1.1.0")
	if items(v.Conflicts) != "config:1.0.0::config_mismatch" {
		t.Fatalf("mismatch %s", items(v.Conflicts))
	}
	f.approve(Change{Kind: ChangeConfig, Config: SiteConfig{Site: "s1", Version: "1.0.0"}})
	f.observe("s1", p1, empty())
	if v := f.preview("s1", "1.1.0"); len(v.Conflicts) != 0 {
		t.Errorf("empty config for another version %s", items(v.Conflicts))
	}
}

func TestPreviewRules(t *testing.T) {
	f := standard(t)
	f.observe("s1", p1, empty())
	odd := fixture("2.0.0")
	odd.Platforms = []Platform{{OS: "linux", Arch: "arm64"}}
	f.approve(signed(odd))
	reason(t, "platform", second(f.r.Preview("s1", "2.0.0")), ReasonUnsupported)
	f.must(second(f.r.RevokeRelease("1.0.0", "Broken build", ann)))
	reason(t, "revoked", second(f.r.Preview("s1", "1.0.0")), ReasonRevoked)
	f.must(second(f.r.ClearMark(Target{Kind: MarkRelease, Version: "1.0.0"}, ann)))
	f.must(second(f.r.Preview("s1", "1.0.0")))
	f.must(second(f.r.CompromiseKey("k1", "Leaked", ann)))
	reason(t, "compromised key", second(f.r.Preview("s1", "1.0.0")), ReasonRevoked)
}

func TestPreviewChangesNothing(t *testing.T) {
	f := standard(t)
	o := empty()
	o.Volumes = []string{"ledger"}
	f.observe("s1", p1, o)
	before := *f.r.sites["s1"]
	f.preview("s1", "1.0.0")
	after := f.r.sites["s1"]
	if after.revision != before.revision || after.sequence != before.sequence || len(after.runs) != 0 || after.observation.Volumes[0] != "ledger" {
		t.Error("preview changed state")
	}
}
