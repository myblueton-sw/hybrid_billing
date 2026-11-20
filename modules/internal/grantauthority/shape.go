package grantauthority

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"
)

func missing(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func, reflect.Map, reflect.Slice, reflect.Chan:
		return r.IsNil()
	}
	return false
}

func validID(s string) bool {
	if len(s) < 1 || len(s) > 128 {
		return false
	}
	for _, c := range []byte(s) {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("._:-", rune(c))) {
			return false
		}
	}
	return true
}

func validText(s string, limit int) bool {
	if len(s) < 1 || len(s) > limit || !utf8.ValidString(s) || strings.TrimSpace(s) == "" {
		return false
	}
	for _, c := range s {
		if c < 32 || c == 127 {
			return false
		}
	}
	return true
}

func validTime(t time.Time) bool      { return !t.IsZero() && t.Year() >= 1 && t.Year() <= 9999 }
func canonical(t time.Time) time.Time { return t.UTC().Round(0) }

func normalizeSpec(s Spec) Spec {
	s.NotBefore = canonical(s.NotBefore)
	s.NotAfter = canonical(s.NotAfter)
	return s
}
func (r *Registry) validSpec(s Spec, maximum time.Duration) bool {
	return r.capabilities[s.Capability] && validID(s.Workspace) && validID(s.Recipient) && validTime(s.NotBefore) && validTime(s.NotAfter) && s.NotBefore.Before(s.NotAfter) && !s.NotAfter.After(s.NotBefore.Add(maximum)) &&
		((s.Kind == Console && s.CeilingID == "") || (s.Kind == Human && validID(s.CeilingID)))
}

func validIntent(i Intent) bool {
	if !validID(i.Workspace) || !validID(i.ID) || !validID(i.Actor) || !validText(i.Rationale, 1024) || len(i.Digest) != 64 {
		return false
	}
	decoded, err := hex.DecodeString(i.Digest)
	return err == nil && len(decoded) == 32 && strings.ToLower(i.Digest) == i.Digest
}

func hash(v any) string {
	body, _ := json.Marshal(v)
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}
func clone(r Record) Record {
	if r.Decision != nil {
		d := *r.Decision
		r.Decision = &d
	}
	return r
}
func sameDecision(r Record, i Intent) bool {
	return r.Decision != nil && r.Decision.Actor == i.Actor && r.Decision.Rationale == i.Rationale && r.Digest == i.Digest
}
func exactEffect(e Effect, s Spec) bool {
	return e.Found && e.Spec == s && (e.Status == ActiveGrant || e.Status == RevokedGrant || e.Status == ExpiredGrant)
}
