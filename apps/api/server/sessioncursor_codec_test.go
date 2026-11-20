package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestSessionCursorCodecRejectsAuthenticatedMalformedStructure(t *testing.T) {
	f := setup(t, SeedSample).ready()
	valid := sessionCursor{Reference: start, Expires: start.Add(time.Minute), Caller: "test-caller", After: "test-position", Matched: 2}
	body, err := json.Marshal(valid)
	f.must(err)
	seal := func(body []byte) string {
		mac := hmac.New(sha256.New, f.s.sessionCursorKey)
		mac.Write(body)
		return base64.RawURLEncoding.EncodeToString(body) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	}
	encoded := f.s.sealSessionCursor(valid)
	if got, ok := f.s.openSessionCursor(encoded); !ok || got.Caller != valid.Caller || got.After != valid.After || got.Matched != valid.Matched || !got.Reference.Equal(valid.Reference) || !got.Expires.Equal(valid.Expires) {
		t.Fatal("valid session cursor did not round trip")
	}
	parts := strings.Split(encoded, ".")
	for _, raw := range []string{
		strings.Repeat("x", 4097), "%25." + parts[1], parts[0] + ".%25", seal([]byte("{")),
		seal(append(append([]byte{}, body[:len(body)-1]...), []byte(",\"unknown\":true}")...)),
		seal(append(append([]byte{}, body...), []byte(" {}")...)),
		seal(append(append([]byte{}, body...), []byte(" trailing")...)),
		seal([]byte(`{"matched":"many"}`)),
	} {
		if _, ok := f.s.openSessionCursor(raw); ok {
			t.Fatal("malformed session cursor helper input was accepted")
		}
	}
}
