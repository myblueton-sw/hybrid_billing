package support

import (
	"math"
	"strconv"
	"strings"
	"testing"
	"time"
)

func fields(k Kept) string {
	var out []string
	for _, f := range k.Fields {
		out = append(out, f.Key+"="+f.Class+":"+f.Value)
	}
	return k.Source + "{" + strings.Join(out, ",") + "}"
}

func TestBuildRendersAllowlistedFields(t *testing.T) {
	f := standard(t)
	b := f.bundle("s1")
	want := []string{
		"log{client=address:address-1,elapsed=duration:1500,event=enum:login.failed,operator=user:user-1,retries=integer:3,when=timestamp:2026-11-15T09:00:00Z}",
		"log{contact=email:email-1,event=enum:export.failed,node=host:host-1,operator=user:user-1,subject=identifier:identifier-1}",
		"version{product=version:2.4.0}",
	}
	for index, item := range b.Items {
		if item.Number != index+1 || fields(item) != want[index] {
			t.Errorf("item %d: %s", index, fields(item))
		}
	}
	if b.Summary.Kept != 3 || len(b.Summary.Drops) != 0 || b.State != StateDraft || b.Site != "s1" || b.ID != "bundle-1" || !b.Expiry.Equal(start.Add(time.Hour)) {
		t.Errorf("bundle %+v", b)
	}
	if p := b.Summary.Pseudonyms; p[ClassUser] != 1 || p[ClassAddress] != 1 || p[ClassHost] != 1 || p[ClassEmail] != 1 || p[ClassIdentifier] != 1 || len(p) != 5 {
		t.Errorf("pseudonyms %v", p)
	}
	if sum(b.Bytes) != b.Digest {
		t.Error("digest is not the hash of the bytes")
	}
	for _, secret := range []string{"Jane", "jane", "10.1.2.3", "db01", "acme", "4111111111111111", "ops@"} {
		if strings.Contains(string(b.Bytes), secret) {
			t.Errorf("%s leaked", secret)
		}
	}
	if strings.Contains(string(b.Bytes), "s1") || !strings.Contains(string(b.Bytes), "site-1") {
		t.Error("site id leaked or handle missing")
	}
	other := f.bundle("s2")
	if other.ID != "bundle-1" || !strings.Contains(string(other.Bytes), "site-2") {
		t.Errorf("per-site numbering %s", other.ID)
	}
	again := f.bundle("s1")
	if again.Items[0].Fields[3].Value != "user-1" || again.ID != "bundle-2" {
		t.Error("pseudonyms are not per bundle")
	}
}

func TestBuildDrops(t *testing.T) {
	f := standard(t)
	items := []Item{
		{Source: "trace", Fields: []Field{{Key: "event", Text: "login.failed"}}},
		{Source: SourceLog, Fields: []Field{{Key: "event", Text: "login.failed"}, {Key: "event", Text: "import.done"}}},
		{Source: SourceLog, Fields: []Field{{Key: "password", Text: "hunter2-secret-value"}, {Key: "message", Text: "Authorization: Bearer eyJhbGciOi"}}},
		{Source: SourceLog, Fields: []Field{
			{Key: "event", Text: "Bearer abc"}, {Key: "event2", Text: "x"},
		}},
		{Source: SourceLog, Fields: []Field{
			{Key: "event", Int: 1, Text: "login.failed"},
			{Key: "retries", Int: 1, Text: "x"},
			{Key: "elapsed", Duration: -1},
			{Key: "when", Time: time.Date(1969, 12, 31, 23, 59, 59, 0, time.UTC)},
			{Key: "node", Text: ""},
			{Key: "subject", Text: "a", Int: 2},
			{Key: "operator", Text: "x", Duration: 1},
			{Key: "client", Text: "x", Time: start},
		}},
		{Source: SourceVersion, Fields: []Field{{Key: "product", Text: "2.4"}}},
		{Source: SourceLog, Fields: []Field{{Key: "retries", Int: 7}}},
	}
	b, err := f.r.Create("s1", items, ann)
	f.must(err)
	if b.Summary.Kept != 1 || fields(b.Items[0]) != "log{retries=integer:7}" {
		t.Fatalf("kept %+v", b.Items)
	}
	got := []string{}
	for _, d := range b.Summary.Drops {
		got = append(got, d.Source+"/"+d.Reason+"/"+itoa(d.Count))
	}
	want := "/unknown_source/1,log/duplicate_key/1,log/no_fields/3,log/out_of_range/3,log/undeclared/3,log/wrong_shape/6,version/no_fields/1,version/out_of_range/1"
	if strings.Join(got, ",") != want {
		t.Errorf("drops %s", strings.Join(got, ","))
	}
	for _, leaked := range []string{"trace", "password", "hunter2", "Bearer", "message", "event2", "2.4\x00"} {
		if strings.Contains(string(b.Bytes), leaked) {
			t.Errorf("%q leaked", leaked)
		}
	}
	reason(t, "empty", second(f.r.Create("s1", []Item{{Source: "trace"}}, ann)), ReasonEmpty)
}

func itoa(n int) string { return strconv.Itoa(n) }

func TestClassBoundaries(t *testing.T) {
	b := &builder{seen: map[string]map[string]string{}, drops: map[[2]string]int{}}
	spec := func(class string) FieldSpec { return FieldSpec{Key: "k", Class: class, Values: []string{"a", "c"}} }
	cases := []struct {
		class string
		field Field
		value string
		drop  string
	}{
		{ClassEnum, Field{Text: "a"}, "a", ""},
		{ClassEnum, Field{Text: "c"}, "c", ""},
		{ClassEnum, Field{Text: "b"}, "", DropOutOfRange},
		{ClassEnum, Field{Text: "d"}, "", DropOutOfRange},
		{ClassEnum, Field{Text: "A"}, "", DropOutOfRange},
		{ClassInteger, Field{Int: math.MinInt64}, "-9223372036854775808", ""},
		{ClassInteger, Field{}, "0", ""},
		{ClassDuration, Field{}, "0", ""},
		{ClassDuration, Field{Duration: maxDuration}, "31622400000", ""},
		{ClassDuration, Field{Duration: maxDuration + 1}, "", DropOutOfRange},
		{ClassDuration, Field{Duration: 1999 * time.Microsecond}, "1", ""},
		{ClassTimestamp, Field{Time: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)}, "1970-01-01T00:00:00Z", ""},
		{ClassTimestamp, Field{Time: time.Date(9999, 12, 31, 23, 59, 59, 999, time.FixedZone("x", 3600))}, "9999-12-31T22:59:59Z", ""},
		{ClassTimestamp, Field{Time: time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)}, "", DropOutOfRange},
		{ClassTimestamp, Field{}, "", DropWrongShape},
		{ClassTimestamp, Field{Time: start, Int: 1}, "", DropWrongShape},
		{ClassVersion, Field{Text: "1.0.0"}, "1.0.0", ""},
		{ClassVersion, Field{Text: "1.0.0", Int: 1}, "", DropWrongShape},
		{ClassHost, Field{Text: "a"}, "host-1", ""},
		{ClassHost, Field{Text: " A "}, "host-1", ""},
		{ClassHost, Field{Text: "b"}, "host-2", ""},
		{ClassHost, Field{Int: 1}, "", DropWrongShape},
		{ClassIdentifier, Field{Int: 0}, "identifier-1", ""},
		{ClassIdentifier, Field{Text: "0"}, "identifier-1", ""},
		{ClassIdentifier, Field{Text: "x", Time: start}, "", DropWrongShape},
		{ClassIdentifier, Field{Duration: 1}, "", DropWrongShape},
		{ClassInteger, Field{Duration: 1}, "", DropWrongShape},
		{ClassInteger, Field{Time: start}, "", DropWrongShape},
		{ClassDuration, Field{Text: "x"}, "", DropWrongShape},
		{ClassEnum, Field{Time: start, Text: "a"}, "", DropWrongShape},
	}
	for _, c := range cases {
		value, drop := b.render(spec(c.class), c.field)
		if value != c.value || drop != c.drop {
			t.Errorf("%s %+v: %q %q", c.class, c.field, value, drop)
		}
	}
}

func TestInputForm(t *testing.T) {
	f := standard(t)
	long := strings.Repeat("x", maxValueBytes+1)
	bad := map[string][]Item{
		"none":        nil,
		"many items":  make([]Item, maxItems+1),
		"many fields": {{Source: SourceLog, Fields: make([]Field, maxFields+1)}},
		"long text":   {{Source: SourceLog, Fields: []Field{{Key: "node", Text: long}}}},
		"long key":    {{Source: SourceLog, Fields: []Field{{Key: long}}}},
		"long source": {{Source: long}},
		"utf8":        {{Source: SourceLog, Fields: []Field{{Key: "node", Text: "\xff"}}}},
		"utf8 key":    {{Source: SourceLog, Fields: []Field{{Key: "\xff"}}}},
		"utf8 source": {{Source: "\xff"}},
	}
	for name, items := range bad {
		expect(t, name, second(f.r.Create("s1", items, ann)), ErrInvalid)
	}
	big := []Item{}
	for index := 0; index < maxItems; index++ {
		item := Item{Source: SourceLog}
		for at := 0; at < 65; at++ {
			item.Fields = append(item.Fields, Field{Key: "node", Text: strings.Repeat("y", maxValueBytes)})
		}
		big = append(big, item)
	}
	expect(t, "total", second(f.r.Create("s1", big, ann)), ErrInvalid)
	edge := []Item{{Source: strings.Repeat("s", maxValueBytes), Fields: make([]Field, maxFields)}}
	reason(t, "edge accepted then empty", second(f.r.Create("s1", append(edge, make([]Item, maxItems-1)...), ann)), ReasonEmpty)
	exact := []Item{}
	for index := 0; index < 64; index++ {
		item := Item{Source: SourceLog}
		for at := 0; at < 128; at++ {
			item.Fields = append(item.Fields, Field{Key: "nod", Text: strings.Repeat("y", maxValueBytes)})
		}
		exact = append(exact, item)
	}
	exact[0].Fields[0].Key = ""
	reason(t, "exact total", second(f.r.Create("s1", exact, ann)), ReasonEmpty)
	reason(t, "assistant", second(f.r.Create("s1", sample(), bot)), ReasonAINotAllowed)
	reason(t, "collector", second(f.r.Create("s1", sample(), agent)), ReasonNotHuman)
	expect(t, "site form", second(f.r.Create("", sample(), ann)), ErrInvalid)
	expect(t, "site missing", second(f.r.Create("s9", sample(), ann)), ErrNotFound)
}

const bundleAnswer = "1229e1a135507118978a7385de3362f2794371e60c347b6903b42cb491bdb0a5"

func TestEncodingKnownAnswer(t *testing.T) {
	items := []Kept{{Number: 1, Source: "log", Fields: []Rendered{{Key: "event", Class: "enum", Value: "login.failed"}}}}
	s := Summary{Kept: 1, Pseudonyms: map[string]int{ClassHost: 2}, Drops: []Drop{{Source: "log", Reason: DropUndeclared, Count: 3}}}
	if got := sum(encode("s1", "bundle-1", items, s)); got != bundleAnswer {
		t.Errorf("encoding %s", got)
	}
	base := sum(encode("s1", "bundle-1", items, s))
	changes := map[string]func() []byte{
		"site":   func() []byte { return encode("s2", "bundle-1", items, s) },
		"id":     func() []byte { return encode("s1", "bundle-2", items, s) },
		"number": func() []byte { c := clone(items); c[0].Number = 2; return encode("s1", "bundle-1", c, s) },
		"source": func() []byte { c := clone(items); c[0].Source = "x"; return encode("s1", "bundle-1", c, s) },
		"key":    func() []byte { c := clone(items); c[0].Fields[0].Key = "x"; return encode("s1", "bundle-1", c, s) },
		"class":  func() []byte { c := clone(items); c[0].Fields[0].Class = "x"; return encode("s1", "bundle-1", c, s) },
		"value":  func() []byte { c := clone(items); c[0].Fields[0].Value = "x"; return encode("s1", "bundle-1", c, s) },
		"items":  func() []byte { return encode("s1", "bundle-1", nil, s) },
		"kept":   func() []byte { c := s; c.Kept = 2; return encode("s1", "bundle-1", items, c) },
		"pseudonym": func() []byte {
			c := s
			c.Pseudonyms = map[string]int{ClassHost: 3}
			return encode("s1", "bundle-1", items, c)
		},
		"drops": func() []byte { c := s; c.Drops = nil; return encode("s1", "bundle-1", items, c) },
		"drop src": func() []byte {
			c := s
			c.Drops = []Drop{{Source: "x", Reason: DropUndeclared, Count: 3}}
			return encode("s1", "bundle-1", items, c)
		},
		"drop why": func() []byte {
			c := s
			c.Drops = []Drop{{Source: "log", Reason: "x", Count: 3}}
			return encode("s1", "bundle-1", items, c)
		},
		"drop n": func() []byte {
			c := s
			c.Drops = []Drop{{Source: "log", Reason: DropUndeclared, Count: 4}}
			return encode("s1", "bundle-1", items, c)
		},
	}
	for name, change := range changes {
		if sum(change()) == base {
			t.Errorf("%s does not change the digest", name)
		}
	}
}

func clone(items []Kept) []Kept {
	out := make([]Kept, len(items))
	for index, item := range items {
		item.Fields = append([]Rendered(nil), item.Fields...)
		out[index] = item
	}
	return out
}
