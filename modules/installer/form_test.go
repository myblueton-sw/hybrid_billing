package installer

import (
	"encoding/hex"
	"strings"
	"testing"
	"time"
)

func TestNewRegistryForm(t *testing.T) {
	clock := func() time.Time { return start }
	good := Config{Random: &entropy{}}
	expect(t, "random", second(NewRegistry(Config{}, limits, clock)), ErrInvalid)
	expect(t, "clock", second(NewRegistry(good, limits, nil)), ErrInvalid)
	bad := map[string]func(*Limits){
		"keys":             func(l *Limits) { l.Keys = maxKeys + 1 },
		"keys zero":        func(l *Limits) { l.Keys = 0 },
		"releases":         func(l *Limits) { l.Releases = maxReleases + 1 },
		"sites":            func(l *Limits) { l.Sites = maxSites + 1 },
		"destinations":     func(l *Limits) { l.Destinations = maxDestinations + 1 },
		"reports":          func(l *Limits) { l.Reports = maxReports + 1 },
		"runs":             func(l *Limits) { l.Runs = maxRuns + 1 },
		"proposals":        func(l *Limits) { l.Proposals = maxProposals + 1 },
		"key life low":     func(l *Limits) { l.KeyLife = minKeyLife - 1 },
		"key life high":    func(l *Limits) { l.KeyLife = maxKeyLife + 1 },
		"observation low":  func(l *Limits) { l.ObservationLife = minObservationLife - 1 },
		"observation high": func(l *Limits) { l.ObservationLife = maxObservationLife + 1 },
		"bootstrap low":    func(l *Limits) { l.BootstrapLife = minBootstrapLife - 1 },
		"bootstrap high":   func(l *Limits) { l.BootstrapLife = maxBootstrapLife + 1 },
	}
	for name, change := range bad {
		l := limits
		change(&l)
		expect(t, name, second(NewRegistry(good, l, clock)), ErrInvalid)
	}
	high := Limits{Keys: maxKeys, KeyLife: maxKeyLife, Releases: maxReleases, Sites: maxSites, Destinations: maxDestinations, Reports: maxReports, Runs: maxRuns,
		ObservationLife: maxObservationLife, BootstrapLife: maxBootstrapLife, Proposals: maxProposals}
	low := Limits{Keys: 1, KeyLife: minKeyLife, Releases: 1, Sites: 1, Destinations: 1, Reports: 1, Runs: 1, ObservationLife: minObservationLife, BootstrapLife: minBootstrapLife, Proposals: 1}
	for _, l := range []Limits{high, low} {
		if _, err := NewRegistry(good, l, clock); err != nil {
			t.Error(err)
		}
	}
}

func TestConstants(t *testing.T) {
	got := []int64{maxIDBytes, maxTextBytes, maxComponent, maxKeys, namesPerKey, int64(minKeyLife), int64(maxKeyLife), maxReleases, maxSites, maxDestinations,
		maxReports, maxRuns, int64(minObservationLife), int64(maxObservationLife), int64(minBootstrapLife), int64(maxBootstrapLife), maxProposals, probesPerSite,
		maxPlatforms, maxArtifacts, maxServices, maxServicePorts, maxServiceVolume, maxVolumes, maxOverrides, maxRanges, minIPv4Bits, minIPv6Bits, maxHostBytes,
		maxLabel, maxObserved, maxGranted, codeBytes}
	want := []int64{128, 200, 6, 16, 64, int64(time.Hour), int64(3650 * 24 * time.Hour), 10000, 100000, 64, 1000, 100, int64(time.Minute), int64(24 * time.Hour),
		int64(time.Minute), int64(24 * time.Hour), 1000, 16, 16, 32, 32, 8, 8, 32, 64, 8, 16, 48, 253, 63, 1024, 16, 32}
	for index := range want {
		if got[index] != want[index] {
			t.Errorf("constant %d is %d; want %d", index, got[index], want[index])
		}
	}
	if fieldPrime.String() != "57896044618658097711785492504343953926634992332820282019728792003956564819949" {
		t.Error("prime")
	}
	if !same(hexBytes("0a9f"), []byte{0x0a, 0x9f}) || !same(hexBytes("af"), []byte{0xaf}) {
		t.Error("hex")
	}
}

func same(a, b []byte) bool { return string(a) == string(b) }

func TestPrimitives(t *testing.T) {
	for _, id := range []string{"a", "Z", "z", "9", "A.z_0:9-", strings.Repeat("x", maxIDBytes)} {
		if !validID(id) {
			t.Errorf("%q refused", id)
		}
	}
	for _, id := range []string{"", strings.Repeat("x", maxIDBytes+1), "a b", "a/b", "[", "`", "{", "@"} {
		if validID(id) {
			t.Errorf("%q accepted", id)
		}
	}
	for _, text := range []string{"a", strings.Repeat("x", maxTextBytes), "Übung", " "} {
		if !validText(text) {
			t.Errorf("%q refused", text)
		}
	}
	for _, text := range []string{"", strings.Repeat("x", maxTextBytes+1), "a\nb", "\x1f", "\x7f", "\u0080", "\u009f", "\xff"} {
		if validText(text) {
			t.Errorf("%q accepted", text)
		}
	}
	if !validHex(strings.Repeat("0a", 32)) || !validHex(strings.Repeat("f9", 32)) {
		t.Error("hex refused")
	}
	for _, text := range []string{strings.Repeat("0", 63), strings.Repeat("0", 65), strings.Repeat("A", 64), "g" + strings.Repeat("0", 63), strings.Repeat("/", 64), strings.Repeat(":", 64), strings.Repeat("`", 64)} {
		if validHex(text) {
			t.Errorf("%q accepted", text)
		}
	}
	for _, text := range []string{"0.0.0", "1.2.3", "999999.0.10"} {
		if _, ok := semver(text); !ok {
			t.Errorf("%s refused", text)
		}
	}
	for _, text := range []string{"", "1.2", "1.2.3.4", "01.2.3", "1.2.x", "1000000.0.0", "1.2.3-rc", "1.2./"} {
		if _, ok := semver(text); ok {
			t.Errorf("%s accepted", text)
		}
	}
	if v, _ := semver("12.345.6"); v != [3]int{12, 345, 6} {
		t.Errorf("parsed %v", v)
	}
}

func TestWeakKeys(t *testing.T) {
	if weak(public("strong")) {
		t.Error("strong refused")
	}
	for _, known := range smallOrder {
		if !weak(hexBytes(known)) {
			t.Errorf("%s accepted", known)
		}
	}
	if !weak(public("strong")[:31]) || !weak(hexBytes("edffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7f")) {
		t.Error("short or non-canonical accepted")
	}
	if weak(hexBytes("ebffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7f")) || weak(hexBytes("ebffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")) {
		t.Error("canonical refused")
	}
}

func TestReleaseForm(t *testing.T) {
	if !fixture("1.0.0").valid() {
		t.Fatal("fixture refused")
	}
	bad := map[string]func(*Release){
		"version":          func(r *Release) { r.Version = "1" },
		"key":              func(r *Release) { r.Key = "" },
		"no platforms":     func(r *Release) { r.Platforms = nil },
		"many platforms":   func(r *Release) { r.Platforms = platforms(maxPlatforms + 1) },
		"platform form":    func(r *Release) { r.Platforms[0].OS = "" },
		"platform arch":    func(r *Release) { r.Platforms[0].Arch = "" },
		"platform order":   func(r *Release) { r.Platforms[0], r.Platforms[1] = r.Platforms[1], r.Platforms[0] },
		"platform twice":   func(r *Release) { r.Platforms[1] = r.Platforms[0] },
		"arch order":       func(r *Release) { r.Platforms = []Platform{{OS: "linux", Arch: "b"}, {OS: "linux", Arch: "a"}} },
		"no artifacts":     func(r *Release) { r.Artifacts, r.Services = nil, nil },
		"many artifacts":   func(r *Release) { r.Artifacts = artifacts(maxArtifacts + 1) },
		"artifact name":    func(r *Release) { r.Artifacts[0].Name = "" },
		"artifact order":   func(r *Release) { r.Artifacts[0], r.Artifacts[1] = r.Artifacts[1], r.Artifacts[0] },
		"artifact twice":   func(r *Release) { r.Artifacts[1].Name = r.Artifacts[0].Name },
		"artifact digest":  func(r *Release) { r.Artifacts[0].Digest = "abc" },
		"artifact kind":    func(r *Release) { r.Artifacts[0].Kind = "script" },
		"no services":      func(r *Release) { r.Services = nil },
		"many services":    func(r *Release) { r.Services = services(maxServices + 1) },
		"service name":     func(r *Release) { r.Services[0].Name = "" },
		"service order":    func(r *Release) { r.Services[0], r.Services[1] = r.Services[1], r.Services[0] },
		"service twice":    func(r *Release) { r.Services[1].Name = r.Services[0].Name },
		"service artifact": func(r *Release) { r.Services[0].Artifact = "missing" },
		"port number":      func(r *Release) { r.Services[0].Ports[0].Number = 0 },
		"port high":        func(r *Release) { r.Services[0].Ports[0].Number = 65536 },
		"port protocol":    func(r *Release) { r.Services[0].Ports[0].Protocol = "sctp" },
		"port order": func(r *Release) {
			r.Services[1].Ports[0], r.Services[1].Ports[1] = r.Services[1].Ports[1], r.Services[1].Ports[0]
		},
		"port twice":       func(r *Release) { r.Services[1].Ports[1] = r.Services[1].Ports[0] },
		"many ports":       func(r *Release) { r.Services[0].Ports = ports(maxServicePorts + 1) },
		"service volumes":  func(r *Release) { r.Services[0].Volumes = []string{"ledger", "config"} },
		"many svc volumes": func(r *Release) { r.Services[0].Volumes = names(maxServiceVolume + 1) },
		"unknown volume":   func(r *Release) { r.Services[0].Volumes = []string{"config", "other"} },
		"unused volume":    func(r *Release) { r.Volumes = append(r.Volumes, Volume{Name: "spare"}) },
		"volume order":     func(r *Release) { r.Volumes[0], r.Volumes[1] = r.Volumes[1], r.Volumes[0] },
		"volume name":      func(r *Release) { r.Volumes[0].Name = "" },
		"many volumes":     func(r *Release) { r.Volumes = volumes(maxVolumes + 1) },
	}
	for name, change := range bad {
		rel := copyRelease(fixture("1.0.0"))
		change(&rel)
		if rel.valid() {
			t.Errorf("%s accepted", name)
		}
	}
	full := copyRelease(fixture("1.0.0"))
	full.Platforms = platforms(maxPlatforms)
	full.Artifacts = append([]Artifact{{Name: "api", Kind: ArtifactContainer, Digest: sum("api")}}, artifacts(maxArtifacts-1)...)
	full.Services = services(maxServices)
	full.Volumes = nil
	full.Services[0].Ports = ports(maxServicePorts)
	if !full.valid() {
		t.Error("full lists refused")
	}
	vols := copyRelease(fixture("1.0.0"))
	vols.Volumes = volumes(maxVolumes)
	vols.Services = services(1)
	vols.Services[0].Volumes = names(maxServiceVolume)
	if vols.valid() {
		t.Error("unused volumes accepted")
	}
	used := copyRelease(fixture("1.0.0"))
	used.Volumes = volumes(maxServiceVolume)
	used.Services = services(1)
	used.Services[0].Volumes = names(maxServiceVolume)
	if !used.valid() {
		t.Error("full service volumes refused")
	}
}

func platforms(n int) []Platform {
	out := []Platform{}
	for _, name := range names(n) {
		out = append(out, Platform{OS: name, Arch: "amd64"})
	}
	return out
}

func artifacts(n int) []Artifact {
	out := []Artifact{}
	for _, name := range names(n) {
		out = append(out, Artifact{Name: name, Kind: ArtifactPackage, Digest: sum(name)})
	}
	return out
}

func services(n int) []Service {
	out := []Service{}
	for _, name := range names(n) {
		out = append(out, Service{Name: name, Artifact: "api"})
	}
	return out
}

func ports(n int) []Port {
	out := []Port{}
	for index := 0; index < n; index++ {
		out = append(out, Port{Number: 1000 + index, Protocol: ProtocolTCP})
	}
	return out
}

func volumes(n int) []Volume {
	out := []Volume{}
	for _, name := range names(n) {
		out = append(out, Volume{Name: name})
	}
	return out
}

func names(n int) []string {
	out := make([]string, n)
	for index := range out {
		out[index] = "n" + string(rune('a'+index/26)) + string(rune('a'+index%26))
	}
	return out
}

func TestPermissions(t *testing.T) {
	rel := fixture("1.0.0")
	if got := strings.Join(rel.permissions(), ","); got != "create_volume,listen_port,run_service,write_config" {
		t.Errorf("permissions %s", got)
	}
	plain := Release{Services: []Service{{Name: "a"}, {Name: "b", Ports: []Port{{Number: 1, Protocol: ProtocolTCP}}}}}
	if got := strings.Join(plain.permissions(), ","); got != "listen_port,run_service,write_config" {
		t.Errorf("permissions %s", got)
	}
	bare := Release{Services: []Service{{Name: "a"}}}
	if got := strings.Join(bare.permissions(), ","); got != "run_service,write_config" {
		t.Errorf("permissions %s", got)
	}
}

const releaseAnswer = "5f01b2d66e3a025bb7a22583cb9015829d419439a17dacf12cb56682165dcadb"

func TestReleaseEncoding(t *testing.T) {
	base := fixture("1.0.0")
	if got := digestOf(base.encode()); got != releaseAnswer {
		t.Errorf("release %s", got)
	}
	if got := hex.EncodeToString(challenge("k1")); got != "000000000000001d"+hex.EncodeToString([]byte(challengeTag))+"0000000000000002"+hex.EncodeToString([]byte("k1")) {
		t.Errorf("challenge %s", got)
	}
	seen := map[string]string{digestOf(base.encode()): "base"}
	changes := map[string]func(*Release){
		"version":        func(r *Release) { r.Version = "1.0.1" },
		"key":            func(r *Release) { r.Key = "k2" },
		"os":             func(r *Release) { r.Platforms[0].OS = "other" },
		"arch":           func(r *Release) { r.Platforms[0].Arch = "arm64" },
		"platforms":      func(r *Release) { r.Platforms = r.Platforms[:1] },
		"artifact name":  func(r *Release) { r.Artifacts[0].Name = "other" },
		"artifact kind":  func(r *Release) { r.Artifacts[0].Kind = ArtifactPackage },
		"digest":         func(r *Release) { r.Artifacts[0].Digest = sum("other") },
		"artifacts":      func(r *Release) { r.Artifacts = r.Artifacts[:1] },
		"service":        func(r *Release) { r.Services[0].Name = "other" },
		"service image":  func(r *Release) { r.Services[0].Artifact = "worker" },
		"port":           func(r *Release) { r.Services[0].Ports[0].Number = 8444 },
		"protocol":       func(r *Release) { r.Services[0].Ports[0].Protocol = ProtocolUDP },
		"no ports":       func(r *Release) { r.Services[0].Ports = nil },
		"service volume": func(r *Release) { r.Services[0].Volumes = []string{"config"} },
		"services":       func(r *Release) { r.Services = r.Services[:1] },
		"volume":         func(r *Release) { r.Volumes[0].Name = "other" },
		"data":           func(r *Release) { r.Volumes[0].Data = true },
		"volumes":        func(r *Release) { r.Volumes = r.Volumes[:1] },
	}
	for name, change := range changes {
		rel := copyRelease(base)
		change(&rel)
		got := digestOf(rel.encode())
		if other, ok := seen[got]; ok {
			t.Errorf("%s encodes like %s", name, other)
		}
		seen[got] = name
	}
	c := copyRelease(base)
	c.Platforms[0].OS, c.Artifacts[0].Name, c.Services[0].Ports[0].Number, c.Services[0].Volumes[0], c.Volumes[0].Name = "x", "x", 1, "x", "x"
	if base.Platforms[0].OS == "x" || base.Artifacts[0].Name == "x" || base.Services[0].Ports[0].Number == 1 || base.Services[0].Volumes[0] == "x" || base.Volumes[0].Name == "x" {
		t.Error("copy shares storage")
	}
}

func TestHostsAndRanges(t *testing.T) {
	for _, host := range []string{"10.0.0.5", "2001:db8::1", "ca.internal", "a", "a-b.c9", strings.Repeat("a", maxLabel) + ".b", strings.Repeat("a.", 126) + "a"} {
		if !validHost(host) {
			t.Errorf("%q refused", host)
		}
	}
	for _, host := range []string{"", "010.0.0.5", "::ffff:10.0.0.1", "2001:DB8::1", "fe80::1%eth0", "CA.internal", "a.", ".a", "a..b", "-a", "a-", "a-.b", "a_b",
		strings.Repeat("a", maxLabel+1), strings.Repeat("a.", 127) + "a", "a/b", "1.2", "a.123"} {
		if validHost(host) {
			t.Errorf("%q accepted", host)
		}
	}
	for _, text := range []string{"10.0.0.0/16", "10.0.0.0/32", "2001:db8::/48", "2001:db8::1/128"} {
		if _, ok := validRange(text); !ok {
			t.Errorf("%q refused", text)
		}
	}
	for _, text := range []string{"0.0.0.0/0", "10.0.0.0/15", "2001:db8::/47", "::/0", "10.0.0.1/16", "x", "::ffff:10.0.0.0/112", "10.0.0.0/016"} {
		if _, ok := validRange(text); ok {
			t.Errorf("%q accepted", text)
		}
	}
	d := Destination{Site: "s1", ID: "erp", Host: "10.1.2.3", Port: 443, Ranges: []string{"10.1.0.0/16"}}
	if !d.valid() {
		t.Error("destination refused")
	}
	bad := map[string]func(*Destination){
		"site":      func(d *Destination) { d.Site = "" },
		"id":        func(d *Destination) { d.ID = "" },
		"host":      func(d *Destination) { d.Host = "A" },
		"port low":  func(d *Destination) { d.Port = 0 },
		"port high": func(d *Destination) { d.Port = 65536 },
		"no ranges": func(d *Destination) { d.Ranges = nil },
		"many ranges": func(d *Destination) {
			d.Ranges = []string{"10.0.0.0/16", "10.1.0.0/16", "10.2.0.0/16", "10.3.0.0/16", "10.4.0.0/16", "10.5.0.0/16", "10.6.0.0/16", "10.7.0.0/16", "10.8.0.0/16"}
		},
		"range order":   func(d *Destination) { d.Ranges = []string{"10.2.0.0/16", "10.1.0.0/16"} },
		"range twice":   func(d *Destination) { d.Ranges = []string{"10.1.0.0/16", "10.1.0.0/16"} },
		"range form":    func(d *Destination) { d.Ranges = []string{"10.1.0.0/8"} },
		"outside range": func(d *Destination) { d.Ranges = []string{"10.2.0.0/16"} },
	}
	for name, change := range bad {
		c := d
		c.Ranges = append([]string(nil), d.Ranges...)
		change(&c)
		if c.valid() {
			t.Errorf("%s accepted", name)
		}
	}
	named := Destination{Site: "s1", ID: "erp", Host: "erp.internal", Port: 1, Ranges: []string{"10.1.0.0/16", "10.2.0.0/16"}}
	if !named.valid() {
		t.Error("named destination refused")
	}
	named.Port = 65535
	if !named.valid() {
		t.Error("high port refused")
	}
	many := named
	many.Ranges = []string{"10.0.0.0/16", "10.1.0.0/16", "10.2.0.0/16", "10.3.0.0/16", "10.4.0.0/16", "10.5.0.0/16", "10.6.0.0/16", "10.7.0.0/16"}
	if !many.valid() {
		t.Error("eight ranges refused")
	}
}

func TestConfigForm(t *testing.T) {
	c := SiteConfig{Site: "s1", Version: "1.0.0", Ports: []PortOverride{{Service: "api", Port: Port{Number: 8443, Protocol: ProtocolTCP}, Number: 8444},
		{Service: "worker", Port: Port{Number: 9000, Protocol: ProtocolTCP}, Number: 1}, {Service: "worker", Port: Port{Number: 9000, Protocol: ProtocolUDP}, Number: 65535}},
		Volumes: []VolumeOverride{{Volume: "config", Host: "cfg"}, {Volume: "ledger", Host: "data"}}}
	if !c.valid() {
		t.Fatal("config refused")
	}
	bad := map[string]func(*SiteConfig){
		"site":          func(c *SiteConfig) { c.Site = "" },
		"version":       func(c *SiteConfig) { c.Version = "x" },
		"service":       func(c *SiteConfig) { c.Ports[0].Service = "" },
		"port":          func(c *SiteConfig) { c.Ports[0].Port.Protocol = "x" },
		"target low":    func(c *SiteConfig) { c.Ports[0].Number = 0 },
		"target high":   func(c *SiteConfig) { c.Ports[0].Number = 65536 },
		"service order": func(c *SiteConfig) { c.Ports[0], c.Ports[1] = c.Ports[1], c.Ports[0] },
		"port order":    func(c *SiteConfig) { c.Ports[1], c.Ports[2] = c.Ports[2], c.Ports[1] },
		"port twice":    func(c *SiteConfig) { c.Ports[2] = c.Ports[1] },
		"volume":        func(c *SiteConfig) { c.Volumes[0].Volume = "" },
		"host":          func(c *SiteConfig) { c.Volumes[0].Host = "" },
		"volume order":  func(c *SiteConfig) { c.Volumes[0], c.Volumes[1] = c.Volumes[1], c.Volumes[0] },
		"many ports":    func(c *SiteConfig) { c.Ports = make([]PortOverride, maxOverrides+1) },
		"many volumes":  func(c *SiteConfig) { c.Volumes = make([]VolumeOverride, maxOverrides+1) },
	}
	for name, change := range bad {
		copied := c
		copied.Ports = append([]PortOverride(nil), c.Ports...)
		copied.Volumes = append([]VolumeOverride(nil), c.Volumes...)
		change(&copied)
		if copied.valid() {
			t.Errorf("%s accepted", name)
		}
	}
	full := SiteConfig{Site: "s1", Version: "1.0.0"}
	for index := 0; index < maxOverrides; index++ {
		full.Ports = append(full.Ports, PortOverride{Service: "api", Port: Port{Number: index + 1, Protocol: ProtocolTCP}, Number: 1})
		full.Volumes = append(full.Volumes, VolumeOverride{Volume: names(maxOverrides)[index], Host: "h"})
	}
	if !full.valid() {
		t.Error("full overrides refused")
	}
}

func TestObservationForm(t *testing.T) {
	if !empty().valid() {
		t.Fatal("empty refused")
	}
	bad := map[string]Observation{
		"platform":      {Platform: Platform{OS: "linux"}},
		"ports":         {Platform: linux, Ports: []Port{{Number: 2, Protocol: ProtocolTCP}, {Number: 1, Protocol: ProtocolTCP}}},
		"volumes":       {Platform: linux, Volumes: []string{"b", "a"}},
		"services":      {Platform: linux, Services: []string{"a", "a"}},
		"granted":       {Platform: linux, Granted: []string{"x y"}},
		"many granted":  {Platform: linux, Granted: names(maxGranted + 1)},
		"many volumes":  {Platform: linux, Volumes: many(maxObserved + 1)},
		"many services": {Platform: linux, Services: many(maxObserved + 1)},
	}
	for name, o := range bad {
		if o.valid() {
			t.Errorf("%s accepted", name)
		}
	}
	full := Observation{Platform: linux, Volumes: many(maxObserved), Services: many(maxObserved), Granted: names(maxGranted)}
	if !full.valid() {
		t.Error("full observation refused")
	}
}

func many(n int) []string {
	out := make([]string, n)
	for index := range out {
		out[index] = "v" + string(rune('a'+index/676)) + string(rune('a'+index/26%26)) + string(rune('a'+index%26))
	}
	return out
}

func TestGuidanceIsSafe(t *testing.T) {
	unsafe := []string{"disable", "skip", "insecure", "ignore", "admin", "root", "trust_all", "no_verify", "bypass"}
	for key, g := range guidance {
		for _, step := range g.Steps {
			for _, word := range unsafe {
				if strings.Contains(step, word) {
					t.Errorf("%s step %s", key, step)
				}
			}
		}
	}
	want := map[string]string{
		"tls/unknown_authority": "add_internal_ca_to_product_trust,verify_chain", "tls/hostname_mismatch": "use_certificate_hostname,reissue_certificate",
		"tls/expired": "check_clock,renew_certificate", "tls/not_yet_valid": "check_clock,renew_certificate", "tls/weak_signature": "reissue_certificate",
		"proxy/proxy_auth": "configure_proxy_credentials", "proxy/proxy_refused": "configure_proxy_allowlist", "dns/dns_failure": "check_dns",
		"tcp/refused": "check_firewall,check_service", "tcp/timeout": "check_firewall,check_service", "tcp/ok": "", "tls/unknown": "collect_details",
	}
	for key, steps := range want {
		parts := strings.SplitN(key, "/", 2)
		g, ok := diagnosis(parts[0], parts[1])
		if !ok || strings.Join(g.Steps, ",") != steps || g.Cause == "" {
			t.Errorf("%s: %+v", key, g)
		}
	}
	if len(guidance) != 10 {
		t.Errorf("table has %d entries", len(guidance))
	}
	for _, pair := range [][2]string{{"tls", "refused"}, {"tcp", "expired"}, {"dns", "proxy_auth"}, {"proxy", "dns_failure"}, {"http", "ok"}, {"tls", "disable"}} {
		if _, ok := diagnosis(pair[0], pair[1]); ok {
			t.Errorf("%v accepted", pair)
		}
	}
	g, _ := diagnosis("tls", "expired")
	g.Steps[0] = "x"
	if again, _ := diagnosis("tls", "expired"); again.Steps[0] != "check_clock" {
		t.Error("guidance shares storage")
	}
}
