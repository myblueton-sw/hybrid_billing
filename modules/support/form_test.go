package support

import (
	"strings"
	"testing"
	"time"
)

func TestNewRegistryForm(t *testing.T) {
	clock := func() time.Time { return start }
	expect(t, "clock", second(NewRegistry(limits, nil)), ErrInvalid)
	bad := map[string]func(*Limits){
		"sites":        func(l *Limits) { l.Sites = maxSites + 1 },
		"sites zero":   func(l *Limits) { l.Sites = 0 },
		"destinations": func(l *Limits) { l.Destinations = maxDestinations + 1 },
		"bundles":      func(l *Limits) { l.Bundles = maxBundles + 1 },
		"site bytes":   func(l *Limits) { l.SiteBytes = maxSiteBytes + 1 },
		"bytes":        func(l *Limits) { l.Bytes = maxBytes + 1 },
		"proposals":    func(l *Limits) { l.Proposals = maxProposals + 1 },
		"site above":   func(l *Limits) { l.SiteBytes, l.Bytes = 2048, 1024 },
		"life low":     func(l *Limits) { l.BundleLife = minBundleLife - 1 },
		"life high":    func(l *Limits) { l.BundleLife = maxBundleLife + 1 },
	}
	for name, change := range bad {
		l := limits
		change(&l)
		expect(t, name, second(NewRegistry(l, clock)), ErrInvalid)
	}
	for _, l := range []Limits{{Sites: maxSites, Destinations: maxDestinations, Bundles: maxBundles, SiteBytes: maxSiteBytes, Bytes: maxBytes, BundleLife: maxBundleLife, Proposals: maxProposals},
		{Sites: 1, Destinations: 1, Bundles: 1, SiteBytes: 1, Bytes: 1, BundleLife: minBundleLife, Proposals: 1}} {
		if _, err := NewRegistry(l, clock); err != nil {
			t.Error(err)
		}
	}
}

func TestConstants(t *testing.T) {
	got := []int64{maxIDBytes, maxComponent, maxSites, maxDestinations, maxBundles, maxSiteBytes, maxBytes, int64(minBundleLife), int64(maxBundleLife), maxProposals,
		maxSchemaPending, maxHostBytes, maxLabel, maxSpecs, maxValues, maxItems, maxFields, maxValueBytes, maxInputBytes, maxOutput, maxUploads, int64(maxDuration)}
	want := []int64{128, 6, 100000, 64, 100, 64 << 20, 4 << 30, int64(time.Hour), int64(30 * 24 * time.Hour), 1000, 16, 253, 63, 128, 64, 256, 128, 256, 4 << 20, 4 << 20, 16,
		int64(366 * 24 * time.Hour)}
	for index := range want {
		if got[index] != want[index] {
			t.Errorf("constant %d is %d; want %d", index, got[index], want[index])
		}
	}
	if strings.Join(sensitive, ",") != "pass,pwd,secret,token,key,credential,private,auth,cookie,session,signature,sig,pin,otp,cert,dsn,connection,card,pan,iban,account,customer,phone,ssn,tax,invoice" {
		t.Error("sensitive list")
	}
}

func TestPrimitives(t *testing.T) {
	for _, id := range []string{"a", "Z", "z", "9", "A.z_0:9-", strings.Repeat("x", maxIDBytes)} {
		if !validID(id) {
			t.Errorf("%q refused", id)
		}
	}
	for _, id := range []string{"", strings.Repeat("x", maxIDBytes+1), "a b", "[", "`", "{", "@", "/"} {
		if validID(id) {
			t.Errorf("%q accepted", id)
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
	for _, text := range []string{"", "1.2", "1.2.3.4", "01.2.3", "1.2.x", "1000000.0.0", "1.2./"} {
		if _, ok := semver(text); ok {
			t.Errorf("%s accepted", text)
		}
	}
	if v, _ := semver("12.345.6"); v != [3]int{12, 345, 6} {
		t.Error("parsed")
	}
	for _, host := range []string{"10.0.0.5", "2001:db8::1", "support.vendor.example", "z", "x0.z", strings.Repeat("a", maxLabel) + ".b", strings.Repeat("a.", 126) + "a"} {
		if !validHost(host) {
			t.Errorf("%q refused", host)
		}
	}
	for _, host := range []string{"", "010.0.0.5", "::ffff:10.0.0.1", "2001:DB8::1", "fe80::1%eth0", "A", "a.", ".a", "a..b", "-a", "a-", "a-.b", "a_b", "1.2", "a.123",
		strings.Repeat("a", maxLabel+1), strings.Repeat("a.", 127) + "a"} {
		if validHost(host) {
			t.Errorf("%q accepted", host)
		}
	}
	if normal("  Ab\tC ") != "ab\tc" || normal("ÄB") != "Äb" {
		t.Errorf("normal %q %q", normal("  Ab\tC "), normal("ÄB"))
	}
}

func TestSchemaForm(t *testing.T) {
	if !logSchema().valid() {
		t.Fatal("fixture refused")
	}
	bad := map[string]func(*Schema){
		"source":        func(s *Schema) { s.Source = "trace" },
		"no fields":     func(s *Schema) { s.Fields = nil },
		"many fields":   func(s *Schema) { s.Fields = specs(maxSpecs + 1) },
		"key form":      func(s *Schema) { s.Fields[0].Key = "Client" },
		"key digit":     func(s *Schema) { s.Fields[0].Key = "1client" },
		"key long":      func(s *Schema) { s.Fields[0].Key = "a" + strings.Repeat("b", maxKeyBytes) },
		"key order":     func(s *Schema) { s.Fields[0], s.Fields[1] = s.Fields[1], s.Fields[0] },
		"key twice":     func(s *Schema) { s.Fields[1].Key = s.Fields[0].Key },
		"class":         func(s *Schema) { s.Fields[0].Class = "text" },
		"values":        func(s *Schema) { s.Fields[0].Values = []string{"a"} },
		"no values":     func(s *Schema) { s.Fields[3].Values = nil },
		"many values":   func(s *Schema) { s.Fields[3].Values = values(maxValues + 1) },
		"value form":    func(s *Schema) { s.Fields[3].Values = []string{"Export"} },
		"value long":    func(s *Schema) { s.Fields[3].Values = []string{"a" + strings.Repeat("b", maxEnumBytes)} },
		"value order":   func(s *Schema) { s.Fields[3].Values = []string{"b", "a"} },
		"value twice":   func(s *Schema) { s.Fields[3].Values = []string{"a", "a"} },
		"pseudonym":     func(s *Schema) { s.Fields[3].Values = []string{"host-1"} },
		"pseudonym id":  func(s *Schema) { s.Fields[3].Values = []string{"identifier-42"} },
		"value control": func(s *Schema) { s.Fields[3].Values = []string{"a​b"} },
	}
	for name, change := range bad {
		s := copySchema(logSchema())
		change(&s)
		if s.valid() {
			t.Errorf("%s accepted", name)
		}
	}
	full := Schema{Source: SourceMetric, Fields: specs(maxSpecs)}
	full.Fields[0] = FieldSpec{Key: full.Fields[0].Key, Class: ClassEnum, Values: values(maxValues)}
	if !full.valid() {
		t.Error("full schema refused")
	}
	edge := Schema{Source: SourceConfig, Fields: []FieldSpec{{Key: "a" + strings.Repeat("b", maxKeyBytes-1), Class: ClassEnum, Values: []string{"a" + strings.Repeat("b", maxEnumBytes-1), "host-x", "hosts-1"}}}}
	if !edge.valid() {
		t.Error("edge schema refused")
	}
	for _, source := range []string{SourceLog, SourceConfig, SourceDiagnostic, SourceMetric, SourceVersion} {
		if !(Schema{Source: source, Fields: []FieldSpec{{Key: "a", Class: ClassInteger}}}).valid() {
			t.Errorf("%s refused", source)
		}
	}
	for _, key := range []string{"db_password", "passwd", "api_key", "x_token", "pin", "otp_code", "client_cert", "card_no", "iban", "account_id", "customer_number", "phone", "ssn",
		"tax_id", "invoice_no", "signature", "sig", "dsn", "connection", "cookie", "session_id", "auth_mode", "private", "credential", "secret", "pwd", "pan", "keys_total"} {
		if !sensitiveKey(key) {
			t.Errorf("%s not sensitive", key)
		}
	}
	for _, key := range []string{"retries", "event", "node", "elapsed", "when", "operator"} {
		if sensitiveKey(key) {
			t.Errorf("%s sensitive", key)
		}
	}
}

func specs(n int) []FieldSpec {
	out := []FieldSpec{}
	for _, name := range names(n) {
		out = append(out, FieldSpec{Key: name, Class: ClassInteger})
	}
	return out
}

func values(n int) []string { return names(n) }

func names(n int) []string {
	out := make([]string, n)
	for index := range out {
		out[index] = "n" + string(rune('a'+index/26)) + string(rune('a'+index%26))
	}
	return out
}
