package server

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func TestQueryTokenIsBoundToItsDomainAndContent(t *testing.T) {
	s := &Server{serviceQueryKey: []byte("0123456789abcdef0123456789abcdef")}
	type body struct {
		A string `json:"a"`
		N int    `json:"n"`
	}
	raw, ok := s.sealQueryToken("one-v1", body{A: "x", N: 3})
	if !ok {
		t.Fatal("seal")
	}
	var got body
	if !s.openQueryToken("one-v1", raw, &got) || got != (body{A: "x", N: 3}) {
		t.Fatalf("round trip: %+v", got)
	}
	parts := strings.Split(raw, ".")
	extra := base64.RawURLEncoding.EncodeToString([]byte(`{"a":"x","n":3,"z":1}`))
	trailing := base64.RawURLEncoding.EncodeToString([]byte(`{"a":"x","n":3}{}`))
	other := &Server{serviceQueryKey: []byte("fedcba9876543210fedcba9876543210")}
	for name, bad := range map[string]string{
		"another domain's token": func() string { v, _ := s.sealQueryToken("two-v1", body{A: "x", N: 3}); return v }(),
		"another key's token":    func() string { v, _ := other.sealQueryToken("one-v1", body{A: "x", N: 3}); return v }(),
		"a changed body":         base64.RawURLEncoding.EncodeToString([]byte(`{"a":"y","n":3}`)) + "." + parts[1],
		"a changed signature":    parts[0] + "." + base64.RawURLEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")),
		"one part":               parts[0],
		"three parts":            raw + ".x",
		"not base64":             "!!." + parts[1],
		"empty":                  "",
		"too long":               strings.Repeat("a", queryTokenBytes+1),
	} {
		var out body
		if s.openQueryToken("one-v1", bad, &out) {
			t.Errorf("%s was opened", name)
		}
	}
	for name, payload := range map[string]string{"an unknown field": extra, "trailing data": trailing} {
		body, _ := base64.RawURLEncoding.DecodeString(payload)
		signed := payload + "." + base64.RawURLEncoding.EncodeToString(s.queryMAC("one-v1", body))
		var out struct {
			A string `json:"a"`
			N int    `json:"n"`
		}
		if s.openQueryToken("one-v1", signed, &out) {
			t.Errorf("a correctly signed token with %s was opened", name)
		}
	}
}

func TestQueryBindingContinuesOnlyTheSameQueryWithinItsLifetime(t *testing.T) {
	now := time.Date(2026, 11, 26, 9, 0, 0, 0, time.UTC)
	base := queryBinding{Area: "a", Target: "t", Owner: sessionOwner{}, Caller: "c", Query: "q", Basis: "b", Reference: now.Add(-time.Minute), Expires: now.Add(time.Minute)}
	if !base.continues(base, now) {
		t.Fatal("the same binding within its lifetime")
	}
	for name, change := range map[string]func(*queryBinding){
		"area": func(b *queryBinding) { b.Area = "x" }, "owner": func(b *queryBinding) { b.Owner = sessionOwner{User: "other"} }, "owner epoch": func(b *queryBinding) { b.Owner = sessionOwner{Epoch: 1} }, "target": func(b *queryBinding) { b.Target = "x" }, "caller": func(b *queryBinding) { b.Caller = "x" },
		"query": func(b *queryBinding) { b.Query = "x" }, "basis": func(b *queryBinding) { b.Basis = "x" },
		"expired": func(b *queryBinding) { b.Expires = now }, "a reference in the future": func(b *queryBinding) { b.Reference = now.Add(time.Second) },
	} {
		prior := base
		change(&prior)
		if base.continues(prior, now) {
			t.Errorf("a prior cursor differing in %s was continued", name)
		}
	}
	if !base.continues(queryBinding{Area: "a", Target: "t", Caller: "c", Query: "q", Basis: "b", Reference: now, Expires: now.Add(time.Nanosecond)}, now) {
		t.Error("a reference at now and an expiry just after now")
	}
}

func TestScanBudgetStopsWithoutSpendingWhatItCannotCover(t *testing.T) {
	b := scanBudget{limit: 5}
	if !b.reserve(3) || b.remaining() != 2 || b.stopped {
		t.Fatalf("after three of five: %+v", b)
	}
	if b.reserve(3) || b.remaining() != 2 || !b.stopped {
		t.Fatalf("three more than remain: %+v", b)
	}
	if !b.reserve(2) || b.remaining() != 0 {
		t.Fatalf("the exact remainder: %+v", b)
	}
	now := time.Date(2026, 11, 26, 9, 0, 0, 0, time.UTC)
	if got := queryExpiry(now.Add(time.Hour), time.Time{}, now, 60); !got.Equal(now.Add(time.Minute)) {
		t.Errorf("lifetime shorter than the session: %v", got)
	}
	if got := queryExpiry(now.Add(30*time.Second), time.Time{}, now, 60); !got.Equal(now.Add(30 * time.Second)) {
		t.Errorf("session end first: %v", got)
	}
	if got := queryExpiry(now.Add(time.Hour), now.Add(10*time.Second), now, 60); !got.Equal(now.Add(10 * time.Second)) {
		t.Errorf("idle end first: %v", got)
	}
}
