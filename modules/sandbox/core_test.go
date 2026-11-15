package sandbox

import (
	"errors"
	"strings"
	"testing"
)

func TestConstants(t *testing.T) {
	got := []int{maxEnvironments, maxDestinations, maxSchemas, maxFields, maxValues, maxTargets, maxResets, maxPending, maxMine, maxDone, maxEnvID, maxPathBytes,
		maxSegments, maxSegmentBytes, maxHostBytes, maxLabel, maxKeyBytes, maxIDBytes}
	want := []int{1000, 256, 1000, 128, 64, 64, 10000, 1000, 250, 1000, 32, 200, 16, 64, 253, 63, 64, 128}
	for index := range want {
		if got[index] != want[index] {
			t.Errorf("constant %d = %d", index, got[index])
		}
	}
	expect(t, "clock", second(NewRegistry(nil)), ErrInvalid)
	if !errors.Is(Refusal{Reason: "x"}, ErrRejected) || (Refusal{Reason: "x"}).Error() != ErrRejected.Error()+": x" {
		t.Error("refusal")
	}
}

func TestValidators(t *testing.T) {
	cases := []struct {
		name string
		fn   func(string) bool
		ok   []string
		bad  []string
	}{
		{"env", validEnvID, []string{"a", "0", "demo2", "z9", strings.Repeat("a", 32)}, []string{"", "Demo", "demo-1", "demo_1", "demo.1", strings.Repeat("a", 33)}},
		{"store", validStore, []string{"a", "z", "billing", "demo_db2", strings.Repeat("a", 63)}, []string{"", "1db", "_db", "Billing", "a.b", "a:b", "a-b", strings.Repeat("a", 64)}},
		{"key", validKey, []string{"a", "batch_size", strings.Repeat("a", 64)}, []string{"", "1a", "A", "a-b", strings.Repeat("a", 65)}},
		{"enum", validEnum, []string{"a", "z", "file", "odoo-test", "a_9", strings.Repeat("a", 64)}, []string{"", "1a", "-a", "A", "a.b", "a:b", "a b", strings.Repeat("a", 65)}},
		{"path", validPath, []string{"/a", "/srv/demo", "/a-b/c_d/09", "/" + strings.Repeat("a", 64), "/" + strings.Repeat("a/", 15) + "a"},
			[]string{"", "/", "a/b", "/a/", "//a", "/a//b", "/A", "/a/./b", "/a/../b", "/a b", "/" + strings.Repeat("a", 65), "/" + strings.Repeat("a/", 16) + "a",
				"/" + strings.Repeat("abcdefghi/", 20)}},
		{"prefix", validPrefix, []string{"INV-", "DEMO-", "AB-", "Z9-", "A0-", "09-", strings.Repeat("A", 16) + "-"}, []string{"", "A-", "INV", "inv-", "IN V-", "IN_V-", "INV--", "aB-", "_B-", strings.Repeat("A", 17) + "-"}},
		{"host", validHost, []string{"erp.mock.test", "a.test", "10.0.0.1", "::1", "a-b.example", "x1.invalid", "z.test", "a9.test", "a0.test", "1.a", "1.z", strings.Repeat("a.", 126) + "a"},
			[]string{"", "ERP.mock.test", "erp.mock.test.", ".a.test", "a..test", "-a.test", "a-.test", "a_b.test", "123", "::ffff:127.0.0.1", "fe80::1%eth0", "0:0::1", "a.b-", "1.2", "a.{", "a.`",
				strings.Repeat("a", 64) + ".test", strings.Repeat("a.", 127) + "aa"}},
		{"id", validID, []string{"A", "Z", "a", "z", "0", "9", "a.b_c:d-e"}, []string{"", "!a", " a", "a b", "a@", "a[", "a`", "a{", "a/"}},
		{"dest", validDestID, []string{"a", "z", "0", "9", "a_b-c"}, []string{"", "A", "a.b", "a:b", "a{", "a`", "a/"}},
		{"reserved", isReserved, []string{"a.test", "erp.mock.invalid", "x.example"}, []string{"test", ".test", "test.com", "example.com", "localhost", "a.localhost", "127.0.0.1", "::1", "atest"}},
	}
	for _, c := range cases {
		for _, value := range c.ok {
			if !c.fn(value) {
				t.Errorf("%s refused %q", c.name, value)
			}
		}
		for _, value := range c.bad {
			if c.fn(value) {
				t.Errorf("%s accepted %q", c.name, value)
			}
		}
	}
	for _, key := range []string{"db_password", "erp_endpoint", "smtp_relay", "callback", "api_url", "proxy", "remote_name", "target", "vault", "bucket", "ship_ip"} {
		if !sensitiveKey(key) {
			t.Errorf("sensitive %s", key)
		}
	}
	if sensitiveKey("batch_size") || sensitiveKey("mode") {
		t.Error("plain key refused")
	}
	if !within("/a/b", "/a/b") || !within("/a/b/c", "/a/b") || within("/a/bc", "/a/b") || within("/a", "/a/b") {
		t.Error("within")
	}
}

func TestForms(t *testing.T) {
	for _, e := range []Environment{prod, demo} {
		if !e.valid() {
			t.Fatalf("fixture %s", e.ID)
		}
	}
	envs := map[string]func(*Environment){
		"id": func(e *Environment) { e.ID = "" }, "mode": func(e *Environment) { e.Mode = "staging" }, "db": func(e *Environment) { e.Database = "" },
		"ring": func(e *Environment) { e.KeyRing = "" }, "path": func(e *Environment) { e.EvidencePath = "" }, "prefix": func(e *Environment) { e.NumberPrefix = "" },
	}
	for name, mutate := range envs {
		e := prod
		mutate(&e)
		if e.valid() {
			t.Errorf("environment %s accepted", name)
		}
	}
	if !erpDst.valid() {
		t.Fatal("destination fixture")
	}
	dsts := map[string]func(*Destination){
		"id": func(d *Destination) { d.ID = "" }, "env": func(d *Destination) { d.Environment = "Demo" }, "kind": func(d *Destination) { d.Kind = "ftp" },
		"host": func(d *Destination) { d.Host = "Erp.test" }, "port0": func(d *Destination) { d.Port = 0 }, "port": func(d *Destination) { d.Port = 65536 },
	}
	for name, mutate := range dsts {
		d := erpDst
		mutate(&d)
		if d.valid() {
			t.Errorf("destination %s accepted", name)
		}
	}
	for _, port := range []int{1, 65535} {
		d := erpDst
		d.Port = port
		if !d.valid() {
			t.Errorf("port %d refused", port)
		}
	}
	if !schema().valid() {
		t.Fatal("schema fixture")
	}
	schemas := map[string]func(*Schema){
		"id":         func(s *Schema) { s.ID = "" },
		"empty":      func(s *Schema) { s.Fields = nil },
		"order":      func(s *Schema) { s.Fields[0], s.Fields[1] = s.Fields[1], s.Fields[0] },
		"repeat":     func(s *Schema) { s.Fields[1].Key = s.Fields[0].Key },
		"key":        func(s *Schema) { s.Fields[0].Key = "Batch" },
		"kind":       func(s *Schema) { s.Fields[0].Kind = "text" },
		"int values": func(s *Schema) { s.Fields[0].Values = []string{"a"} },
		"flag vals":  func(s *Schema) { s.Fields[1].Values = []string{"a"} },
		"dst none":   func(s *Schema) { s.Fields[2].Values = nil },
		"dst kind":   func(s *Schema) { s.Fields[2].Values = []string{"ftp"} },
		"dst two":    func(s *Schema) { s.Fields[2].Values = []string{DestinationERP, DestinationWebhook} },
		"enum none":  func(s *Schema) { s.Fields[3].Values = nil },
		"enum value": func(s *Schema) { s.Fields[3].Values = []string{"erp.real.com"} },
		"enum order": func(s *Schema) { s.Fields[3].Values = []string{"z", "a"} },
		"enum dup":   func(s *Schema) { s.Fields[3].Values = []string{"a", "a"} },
	}
	for name, mutate := range schemas {
		s := schema()
		mutate(&s)
		if s.valid() {
			t.Errorf("schema %s accepted", name)
		}
	}
	big := Schema{ID: "big"}
	for len(big.Fields) < maxFields {
		big.Fields = append(big.Fields, FieldSpec{Key: "f" + strings.Repeat("0", 3-len(itoa(len(big.Fields)))) + itoa(len(big.Fields)), Kind: FieldFlag})
	}
	values := []string{}
	for len(values) < maxValues {
		values = append(values, "v"+strings.Repeat("0", 2-len(itoa(len(values))))+itoa(len(values)))
	}
	big.Fields[0] = FieldSpec{Key: big.Fields[0].Key, Kind: FieldEnum, Values: values}
	if !big.valid() {
		t.Error("caps refused")
	}
	big.Fields[0].Values = append(values, "w")
	if big.valid() {
		t.Error("value cap exceeded")
	}
	big.Fields[0].Values = values
	big.Fields = append(big.Fields, FieldSpec{Key: "z", Kind: FieldFlag})
	if big.valid() {
		t.Error("field cap exceeded")
	}
	for _, c := range []Change{{Kind: "other"}, {Kind: ChangeEnvironment}, {Kind: ChangeDestination}, {Kind: ChangeSchema}} {
		if _, ok := clean(c); ok {
			t.Errorf("%s accepted", c.Kind)
		}
	}
	targets := map[string]struct {
		t  Target
		ok bool
	}{
		"evidence":      {Target{Kind: TargetEvidence, Name: "/srv/demo"}, true},
		"evidence bad":  {Target{Kind: TargetEvidence, Name: "srv"}, false},
		"database":      {Target{Kind: TargetDatabase, Name: "x"}, true},
		"database bad":  {Target{Kind: TargetDatabase, Name: ""}, false},
		"kind":          {Target{Kind: "disk", Name: "x"}, false},
		"keyring":       {Target{Kind: TargetKeyRing, Name: "x"}, true},
		"numbers":       {Target{Kind: TargetNumbers, Name: "DEMO-"}, true},
		"connection":    {Target{Kind: TargetConnection, Name: "x"}, true},
		"volume":        {Target{Kind: TargetVolume, Name: "x"}, true},
		"volume spaced": {Target{Kind: TargetVolume, Name: "x y"}, false},
	}
	for name, c := range targets {
		if c.t.valid() != c.ok {
			t.Errorf("target %s", name)
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	out := ""
	for ; n > 0; n /= 10 {
		out = string(rune('0'+n%10)) + out
	}
	return out
}
