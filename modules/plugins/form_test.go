package plugins

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"
)

func TestNewRegistryForm(t *testing.T) {
	clock := func() time.Time { return start }
	good := Config{Core: "2.4.0", Destinations: &destinations{}, Random: &entropy{}}
	if _, err := NewRegistry(good, limits, clock); err != nil {
		t.Fatal(err)
	}
	for name, c := range map[string]Config{
		"core":         {Core: "2.4", Destinations: good.Destinations, Random: good.Random},
		"destinations": {Core: "2.4.0", Random: good.Random},
		"random":       {Core: "2.4.0", Destinations: good.Destinations},
	} {
		expect(t, name, second(NewRegistry(c, limits, clock)), ErrInvalid)
	}
	expect(t, "clock", second(NewRegistry(good, limits, nil)), ErrInvalid)
	bad := map[string]func(*Limits){
		"publishers low":    func(l *Limits) { l.Publishers = 0 },
		"publishers high":   func(l *Limits) { l.Publishers = maxPublishers + 1 },
		"keys":              func(l *Limits) { l.Keys = maxKeys + 1 },
		"plugins":           func(l *Limits) { l.Plugins = maxPlugins + 1 },
		"versions":          func(l *Limits) { l.Versions = maxVersions + 1 },
		"workspaces":        func(l *Limits) { l.Workspaces = maxWorkspaces + 1 },
		"installations":     func(l *Limits) { l.Installations = maxInstallations + 1 },
		"tokens":            func(l *Limits) { l.Tokens = maxTokens + 1 },
		"fingerprints":      func(l *Limits) { l.Fingerprints = maxFingerprints + 1 },
		"endpoints":         func(l *Limits) { l.Endpoints = maxEndpoints + 1 },
		"proposals":         func(l *Limits) { l.Proposals = maxProposals + 1 },
		"key life low":      func(l *Limits) { l.KeyLife = minKeyLife - 1 },
		"key life high":     func(l *Limits) { l.KeyLife = maxKeyLife + 1 },
		"token life low":    func(l *Limits) { l.TokenLife = minTokenLife - 1 },
		"token life high":   func(l *Limits) { l.TokenLife = maxTokenLife + 1 },
		"proposals at zero": func(l *Limits) { l.Proposals = 0 },
	}
	for name, change := range bad {
		l := limits
		change(&l)
		expect(t, name, second(NewRegistry(good, l, clock)), ErrInvalid)
	}
	edge := Limits{Publishers: maxPublishers, Keys: maxKeys, KeyLife: maxKeyLife, Plugins: maxPlugins, Versions: maxVersions, Workspaces: maxWorkspaces,
		Installations: maxInstallations, Tokens: maxTokens, Fingerprints: maxFingerprints, TokenLife: maxTokenLife, Endpoints: maxEndpoints, Proposals: maxProposals}
	if _, err := NewRegistry(good, edge, clock); err != nil {
		t.Fatal(err)
	}
	low := Limits{Publishers: 1, Keys: 1, KeyLife: minKeyLife, Plugins: 1, Versions: 1, Workspaces: 1, Installations: 1, Tokens: 1, Fingerprints: 1,
		TokenLife: minTokenLife, Endpoints: 1, Proposals: 1}
	if _, err := NewRegistry(good, low, clock); err != nil {
		t.Fatal(err)
	}
}

const manifestAnswer = "f0a3c61026f102daa3f8dcd99d08dd066a8a01592f6aee6f1f35659bc980d888"

func TestSemver(t *testing.T) {
	for _, text := range []string{"0.0.0", "1.2.3", "999999.0.10"} {
		if _, ok := semver(text); !ok {
			t.Errorf("%s refused", text)
		}
	}
	for _, text := range []string{"", "1.2", "1.2.3.4", "01.2.3", "1.02.3", "1.2.03", "1..3", "1.2.x", "1000000.0.0", "1.2.3-rc1", "1.2.3+b", "-1.2.3", "1.2./"} {
		if _, ok := semver(text); ok {
			t.Errorf("%s accepted", text)
		}
	}
	got, _ := semver("12.345.6")
	if got != [3]int{12, 345, 6} {
		t.Errorf("parsed %v", got)
	}
	cases := []struct {
		a, b string
		want int
	}{{"1.2.3", "1.2.3", 0}, {"1.2.3", "1.2.4", -1}, {"1.10.0", "1.9.0", 1}, {"2.0.0", "10.0.0", -1}, {"1.2.10", "1.2.9", 1}, {"0.9.9", "1.0.0", -1}, {"1.0.0", "0.9.9", 1}}
	for _, c := range cases {
		if got := compare(c.a, c.b); got != c.want {
			t.Errorf("compare(%s, %s) = %d", c.a, c.b, got)
		}
	}
}

func TestTextAndIdentifiers(t *testing.T) {
	for _, text := range []string{"a", strings.Repeat("x", maxTextBytes), "Überblick", "<script>", "x y", "~", " "} {
		if !validText(text) {
			t.Errorf("%q refused", text)
		}
	}
	for _, text := range []string{"", strings.Repeat("x", maxTextBytes+1), "a\nb", "\x1f", "\x7f", "\u0080", "\u0085", "\u009f", "\xff"} {
		if validText(text) {
			t.Errorf("%q accepted", text)
		}
	}
	for _, id := range []string{"a", "A.z_0:9-", "Z", "z", "9", strings.Repeat("x", maxIDBytes)} {
		if !validID(id) {
			t.Errorf("%q refused", id)
		}
	}
	for _, id := range []string{"", strings.Repeat("x", maxIDBytes+1), "a b", "a/b", "a@", "[", "`", "{"} {
		if validID(id) {
			t.Errorf("%q accepted", id)
		}
	}
	if !validHex(strings.Repeat("0a", 32)) || !validHex(strings.Repeat("f9", 32)) {
		t.Error("hex refused")
	}
	for _, text := range []string{strings.Repeat("0", 63), strings.Repeat("0", 65), strings.Repeat("A", 64), strings.Repeat("g", 64), "g" + strings.Repeat("0", 63), strings.Repeat("/", 64), strings.Repeat(":", 64), strings.Repeat("`", 64)} {
		if validHex(text) {
			t.Errorf("%q accepted", text)
		}
	}
}

func TestManifestForm(t *testing.T) {
	if !manifest("ledger", "1.0.0").valid() {
		t.Fatal("fixture refused")
	}
	bad := map[string]func(*Manifest){
		"plugin":            func(m *Manifest) { m.Plugin = "a b" },
		"publisher":         func(m *Manifest) { m.Publisher = "" },
		"key":               func(m *Manifest) { m.Key = "" },
		"digest":            func(m *Manifest) { m.Digest = "abc" },
		"version":           func(m *Manifest) { m.Version = "1.0" },
		"core min":          func(m *Manifest) { m.Core.Min = "x" },
		"core max":          func(m *Manifest) { m.Core.Max = "x" },
		"core order":        func(m *Manifest) { m.Core = CoreRange{Min: "3.0.0", Max: "2.0.0"} },
		"no capabilities":   func(m *Manifest) { m.Capabilities, m.Screens = nil, nil },
		"many capabilities": func(m *Manifest) { m.Capabilities = many(maxCapabilities + 1) },
		"capability kind":   func(m *Manifest) { m.Capabilities[0].Kind = "admin" },
		"capability target": func(m *Manifest) { m.Capabilities[0].Target = "" },
		"capability order":  func(m *Manifest) { m.Capabilities[0], m.Capabilities[1] = m.Capabilities[1], m.Capabilities[0] },
		"capability twice":  func(m *Manifest) { m.Screens, m.Capabilities = nil, []Capability{events, events} },
		"kind twice": func(m *Manifest) {
			m.Screens, m.Capabilities = nil, []Capability{events, {Kind: CapabilityReadEvents, Target: "a"}}
		},
		"egress order":     func(m *Manifest) { m.Egress = []string{"b", "a"} },
		"egress twice":     func(m *Manifest) { m.Egress = []string{"a", "a"} },
		"egress id":        func(m *Manifest) { m.Egress = []string{"a b"} },
		"many egress":      func(m *Manifest) { m.Egress = ids(maxEgress + 1) },
		"endpoint order":   func(m *Manifest) { m.Endpoints = []string{"b", "a"} },
		"many endpoints":   func(m *Manifest) { m.Endpoints = ids(maxEndpointIDs + 1) },
		"timeout low":      func(m *Manifest) { m.Limits.Timeout = minTimeout - 1 },
		"timeout high":     func(m *Manifest) { m.Limits.Timeout = maxTimeout + 1 },
		"memory low":       func(m *Manifest) { m.Limits.MemoryMiB = minMemory - 1 },
		"memory high":      func(m *Manifest) { m.Limits.MemoryMiB = maxMemory + 1 },
		"concurrency low":  func(m *Manifest) { m.Limits.Concurrency = 0 },
		"concurrency high": func(m *Manifest) { m.Limits.Concurrency = maxConcurrency + 1 },
		"namespace":        func(m *Manifest) { m.Storage.Namespace = "" },
		"owner":            func(m *Manifest) { m.Storage.Owner = "" },
		"migration":        func(m *Manifest) { m.Storage.Migration = "rewrite" },
		"deletion":         func(m *Manifest) { m.Storage.Deletion = "keep" },
		"retain zero":      func(m *Manifest) { m.Storage.RetentionDays = 0 },
		"retain long":      func(m *Manifest) { m.Storage.RetentionDays = maxRetention + 1 },
		"delete retention": func(m *Manifest) { m.Storage.Deletion = DeleteOnRemoval },
		"many screens":     func(m *Manifest) { m.Screens = make([]Screen, maxScreens+1) },
		"screen id":        func(m *Manifest) { m.Screens[0].ID = "" },
		"screen component": func(m *Manifest) { m.Screens[0].Component = "html" },
		"screen label":     func(m *Manifest) { m.Screens[0].Label = "a\tb" },
		"no fields":        func(m *Manifest) { m.Screens[0].Fields = nil },
		"many fields":      func(m *Manifest) { m.Screens[0].Fields = fields(maxFields+1, m.Storage.Namespace) },
		"no serve":         func(m *Manifest) { m.Capabilities = m.Capabilities[:2] },
		"field name":       func(m *Manifest) { m.Screens[0].Fields[0].Name = "" },
		"field twice":      func(m *Manifest) { m.Screens[0].Fields[1].Name = m.Screens[0].Fields[0].Name },
		"field label":      func(m *Manifest) { m.Screens[0].Fields[0].Label = "" },
		"field source":     func(m *Manifest) { m.Screens[0].Fields[0].Source = "core.credentials" },
		"command source":   func(m *Manifest) { m.Screens[0].Fields[0].Source = "invoice.void" },
		"screen twice":     func(m *Manifest) { m.Screens = []Screen{m.Screens[0], m.Screens[0]} },
		"screen order": func(m *Manifest) {
			m.Capabilities = append(m.Capabilities, Capability{Kind: CapabilityServeScreen, Target: "summary"})
			m.Screens = []Screen{m.Screens[0], m.Screens[0]}
			m.Screens[0].ID = "summary"
		},
	}
	for name, change := range bad {
		m := copyManifest(manifest("ledger", "1.0.0"))
		change(&m)
		if m.valid() {
			t.Errorf("%s accepted", name)
		}
	}
	good := map[string]func(*Manifest){
		"no screens":     func(m *Manifest) { m.Screens = nil },
		"delete rule":    func(m *Manifest) { m.Storage.Deletion, m.Storage.RetentionDays = DeleteOnRemoval, 0 },
		"retain bounds":  func(m *Manifest) { m.Storage.RetentionDays = maxRetention },
		"retain one":     func(m *Manifest) { m.Storage.RetentionDays = 1 },
		"migration none": func(m *Manifest) { m.Storage.Migration = MigrationNone },
		"migration copy": func(m *Manifest) { m.Storage.Migration = MigrationCopyForward },
		"components":     func(m *Manifest) { m.Screens[0].Component = ComponentChart },
		"form":           func(m *Manifest) { m.Screens[0].Component = ComponentForm },
		"detail":         func(m *Manifest) { m.Screens[0].Component = ComponentDetail },
		"limits low":     func(m *Manifest) { m.Limits = RuntimeLimits{Timeout: minTimeout, MemoryMiB: minMemory, Concurrency: 1} },
		"limits high": func(m *Manifest) {
			m.Limits = RuntimeLimits{Timeout: maxTimeout, MemoryMiB: maxMemory, Concurrency: maxConcurrency}
		},
		"lists full":       func(m *Manifest) { m.Egress, m.Endpoints = ids(maxEgress), ids(maxEndpointIDs) },
		"fields full":      func(m *Manifest) { m.Screens[0].Fields = fields(maxFields, m.Storage.Namespace) },
		"capabilities max": func(m *Manifest) { m.Capabilities, m.Screens = many(maxCapabilities), nil },
		"single core":      func(m *Manifest) { m.Core = CoreRange{Min: "2.4.0", Max: "2.4.0"} },
		"empty lists":      func(m *Manifest) { m.Egress, m.Endpoints = nil, nil },
		"screens full":     func(m *Manifest) { m.Capabilities, m.Screens = screens(maxScreens, m.Storage.Namespace) },
	}
	for name, change := range good {
		m := copyManifest(manifest("ledger", "1.0.0"))
		change(&m)
		if !m.valid() {
			t.Errorf("%s refused", name)
		}
	}
}

func many(n int) []Capability {
	out := make([]Capability, n)
	for index := range out {
		out[index] = Capability{Kind: CapabilityReadEvents, Target: "t" + string(rune('A'+index/26)) + string(rune('a'+index%26))}
	}
	return out
}

func screens(n int, source string) ([]Capability, []Screen) {
	var capabilities []Capability
	var out []Screen
	for _, id := range ids(n) {
		capabilities = append(capabilities, Capability{Kind: CapabilityServeScreen, Target: id})
		out = append(out, Screen{ID: id, Component: ComponentTable, Label: "S", Fields: fields(1, source)})
	}
	return capabilities, out
}

func ids(n int) []string {
	out := make([]string, n)
	for index := range out {
		out[index] = "d" + string(rune('A'+index/26)) + string(rune('a'+index%26))
	}
	return out
}

func fields(n int, source string) []Field {
	out := make([]Field, n)
	for index := range out {
		out[index] = Field{Name: "f" + string(rune('A'+index/26)) + string(rune('a'+index%26)), Label: "Field", Source: source}
	}
	return out
}

func TestEncodingKnownAnswers(t *testing.T) {
	sum := func(b []byte) string {
		s := sha256.Sum256(b)
		return hex.EncodeToString(s[:])
	}
	if got := hex.EncodeToString(challenge("acme", "k1")); got != "000000000000001c"+hex.EncodeToString([]byte(challengeTag))+"0000000000000004"+hex.EncodeToString([]byte("acme"))+"0000000000000002"+hex.EncodeToString([]byte("k1")) {
		t.Errorf("challenge %s", got)
	}
	if got := sum(manifest("ledger", "1.0.0").encode()); got != manifestAnswer {
		t.Errorf("manifest %s", got)
	}
	base := manifest("ledger", "1.0.0")
	seen := map[string]string{sum(base.encode()): "base"}
	changes := map[string]func(*Manifest){
		"plugin":      func(m *Manifest) { m.Plugin = "other" },
		"version":     func(m *Manifest) { m.Version = "1.0.1" },
		"publisher":   func(m *Manifest) { m.Publisher = "other" },
		"key":         func(m *Manifest) { m.Key = "k2" },
		"digest":      func(m *Manifest) { m.Digest = strings.Repeat("0", 64) },
		"license":     func(m *Manifest) { m.License = "Apache-2.0" },
		"core min":    func(m *Manifest) { m.Core.Min = "2.0.1" },
		"core max":    func(m *Manifest) { m.Core.Max = "2.9.8" },
		"cap kind":    func(m *Manifest) { m.Capabilities[1].Kind = CapabilityServeScreen },
		"cap target":  func(m *Manifest) { m.Capabilities[1].Target = "other" },
		"cap count":   func(m *Manifest) { m.Capabilities = m.Capabilities[:2] },
		"egress":      func(m *Manifest) { m.Egress = []string{"erp"} },
		"no egress":   func(m *Manifest) { m.Egress = nil },
		"endpoints":   func(m *Manifest) { m.Endpoints = []string{"api", "web"} },
		"timeout":     func(m *Manifest) { m.Limits.Timeout = 2 * time.Second },
		"memory":      func(m *Manifest) { m.Limits.MemoryMiB = 65 },
		"concurrency": func(m *Manifest) { m.Limits.Concurrency = 3 },
		"namespace":   func(m *Manifest) { m.Storage.Namespace = "other" },
		"owner":       func(m *Manifest) { m.Storage.Owner = "other" },
		"retention":   func(m *Manifest) { m.Storage.RetentionDays = 31 },
		"deletion":    func(m *Manifest) { m.Storage.Deletion = DeleteOnRemoval },
		"migration":   func(m *Manifest) { m.Storage.Migration = MigrationNone },
		"screen id":   func(m *Manifest) { m.Screens[0].ID = "other" },
		"component":   func(m *Manifest) { m.Screens[0].Component = ComponentForm },
		"label":       func(m *Manifest) { m.Screens[0].Label = "Other" },
		"field name":  func(m *Manifest) { m.Screens[0].Fields[0].Name = "other" },
		"field label": func(m *Manifest) { m.Screens[0].Fields[0].Label = "Other" },
		"field from":  func(m *Manifest) { m.Screens[0].Fields[0].Source = "other" },
		"field order": func(m *Manifest) {
			m.Screens[0].Fields[0], m.Screens[0].Fields[1] = m.Screens[0].Fields[1], m.Screens[0].Fields[0]
		},
		"no screens": func(m *Manifest) { m.Screens = nil },
		"no fields":  func(m *Manifest) { m.Screens[0].Fields = m.Screens[0].Fields[:1] },
	}
	for name, change := range changes {
		m := copyManifest(base)
		change(&m)
		got := sum(m.encode())
		if other, ok := seen[got]; ok {
			t.Errorf("%s encodes like %s", name, other)
		}
		seen[got] = name
	}
}

func TestConstants(t *testing.T) {
	got := []int64{maxIDBytes, maxTextBytes, maxCapabilities, maxEgress, maxEndpointIDs, maxScreens, maxFields, maxRetention, int64(minTimeout), int64(maxTimeout),
		minMemory, maxMemory, maxConcurrency, maxComponent, maxPublishers, maxKeys, namesPerKey, int64(minKeyLife), int64(maxKeyLife), maxPlugins, maxVersions,
		maxWorkspaces, maxInstallations, maxTokens, int64(minTokenLife), int64(maxTokenLife), maxFingerprints, maxEndpoints, runtimesPer, maxProposals, tokenBytes}
	want := []int64{128, 200, 32, 16, 16, 16, 32, 3650, int64(time.Second), int64(5 * time.Minute), 16, 4096, 64, 6, 1000, 16, 64, int64(time.Hour),
		int64(3650 * 24 * time.Hour), 10000, 100000, 100000, 1000, 1000, int64(time.Second), int64(24 * time.Hour), 10000, 16, 16, 1000, 32}
	for index := range want {
		if got[index] != want[index] {
			t.Errorf("constant %d is %d; want %d", index, got[index], want[index])
		}
	}
	if fieldPrime.String() != "57896044618658097711785492504343953926634992332820282019728792003956564819949" {
		t.Errorf("prime %s", fieldPrime)
	}
	if !same(hexBytes("0a9fAf"[:4]), []byte{0x0a, 0x9f}) || !same(hexBytes("af"), []byte{0xaf}) {
		t.Error("hex decoding")
	}
}

func TestWeakKeys(t *testing.T) {
	if weak(public("strong")) {
		t.Error("strong key refused")
	}
	for _, known := range smallOrder {
		if !weak(hexBytes(known)) {
			t.Errorf("%s accepted", known)
		}
	}
	if !weak(public("strong")[:31]) {
		t.Error("short key accepted")
	}
	prime := hexBytes("edffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7f")
	if !weak(prime) {
		t.Error("non-canonical key accepted")
	}
	below := hexBytes("ebffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7f")
	if weak(below) {
		t.Error("canonical encoding below the prime refused")
	}
	signed := hexBytes("ebffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	if weak(signed) {
		t.Error("sign bit treated as magnitude")
	}
}

func TestCopies(t *testing.T) {
	m := manifest("ledger", "1.0.0")
	c := copyManifest(m)
	c.Capabilities[0].Target, c.Egress[0], c.Endpoints[0], c.Screens[0].Label, c.Screens[0].Fields[0].Name = "x", "x", "x", "x", "x"
	if m.Capabilities[0].Target == "x" || m.Egress[0] == "x" || m.Endpoints[0] == "x" || m.Screens[0].Label == "x" || m.Screens[0].Fields[0].Name == "x" {
		t.Error("copy shares storage")
	}
	k := keyChange("acme", "k1")
	ch := copyChange(Change{Key: k.Key, Signature: []byte{1}})
	ch.Key.Public[0]++
	ch.Key.Proof[0]++
	ch.Signature[0]++
	if ch.Key.Public[0] == k.Key.Public[0] || ch.Key.Proof[0] == k.Key.Proof[0] {
		t.Error("change copy shares keys")
	}
}
