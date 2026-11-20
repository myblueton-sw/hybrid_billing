package server

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestServiceQueryCursorCodecRejectsMalformedSignedStructures(t *testing.T) {
	f := setup(t, SeedSample).ready()
	valid := serviceCursor{Area: "services", Reference: start, Expires: start.Add(time.Minute), After: "svc-last", Aggregate: serviceAggregate{Total: 3, Active: 1, Expired: 1, Revoked: 1}}
	seal := func(body []byte) string {
		return base64.RawURLEncoding.EncodeToString(body) + "." + base64.RawURLEncoding.EncodeToString(f.s.serviceMAC("service-query-v1", body))
	}
	body, err := json.Marshal(valid)
	f.must(err)
	encoded, ok := f.s.sealServiceCursor(valid)
	if !ok {
		t.Fatal("valid cursor could not be encoded")
	}
	for _, raw := range []string{encoded, seal(append(append([]byte{}, body...), []byte(" \n\t")...))} {
		got, accepted := f.s.openServiceCursor(raw)
		if !accepted || got.After != valid.After || got.Aggregate != valid.Aggregate || !got.Reference.Equal(valid.Reference) || !got.Expires.Equal(valid.Expires) {
			t.Fatal("valid cursor round trip lost its fixed query state")
		}
	}
	parts := strings.Split(encoded, ".")
	malformed := map[string]string{
		"over bound":             strings.Repeat("x", 4097),
		"no delimiter":           "opaque",
		"extra delimiter":        encoded + ".extra",
		"bad body alphabet":      "%25." + parts[1],
		"bad signature alphabet": parts[0] + ".%25",
		"signature mismatch":     seal([]byte("{}")) + "x",
		"broken signed JSON":     seal([]byte("{")),
		"unknown signed field":   seal(append(append([]byte{}, body[:len(body)-1]...), []byte(",\"unrecognized\":true}")...)),
		"trailing signed object": seal(append(append([]byte{}, body...), []byte(" {}")...)),
		"trailing signed text":   seal(append(append([]byte{}, body...), []byte(" extra")...)),
		"null signed cursor":     seal([]byte("null")),
	}
	for name, raw := range malformed {
		t.Run(name, func(t *testing.T) {
			if _, accepted := f.s.openServiceCursor(raw); accepted {
				t.Fatal("malformed cursor was accepted")
			}
		})
	}
	cases := []struct {
		name   string
		change func(*serviceCursor)
	}{
		{"invalid after", func(c *serviceCursor) { c.After = "bad!" }},
		{"invalid after service", func(c *serviceCursor) { c.AfterService = "bad!" }},
		{"invalid pending service", func(c *serviceCursor) { c.PendingService = "bad!" }},
		{"invalid after credential", func(c *serviceCursor) { c.AfterCredential = "bad!" }},
		{"active exceeds total", func(c *serviceCursor) { c.Aggregate.Active = 4 }},
		{"expired exceeds total", func(c *serviceCursor) { c.Aggregate.Expired = 4 }},
		{"revoked exceeds total", func(c *serviceCursor) { c.Aggregate.Revoked = 4 }},
		{"missing reference", func(c *serviceCursor) { c.Reference = time.Time{} }},
		{"empty interval", func(c *serviceCursor) { c.Expires = c.Reference }},
		{"reversed interval", func(c *serviceCursor) { c.Expires = c.Reference.Add(-time.Nanosecond) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cursor := valid
			tc.change(&cursor)
			raw, err := json.Marshal(cursor)
			f.must(err)
			if _, accepted := f.s.openServiceCursor(seal(raw)); accepted {
				t.Fatal("signed cursor violated decoder invariants")
			}
		})
	}
}

func TestServiceQueryCursorEncoderRejectsUnrepresentableTime(t *testing.T) {
	f := setup(t, SeedSample).ready()
	for _, change := range []func(*serviceCursor){
		func(c *serviceCursor) { c.Reference = time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC) },
		func(c *serviceCursor) { c.Expires = time.Date(-1, 1, 1, 0, 0, 0, 0, time.UTC) },
		func(c *serviceCursor) { c.Aggregate.Earliest = time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC) },
	} {
		cursor := serviceCursor{Reference: start, Expires: start.Add(time.Minute)}
		change(&cursor)
		if raw, ok := f.s.sealServiceCursor(cursor); ok || raw != "" {
			t.Fatal("unrepresentable timestamp produced a continuation")
		}
	}
}
