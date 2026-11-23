package server

import (
	"strings"
	"testing"
	"time"
)

func TestSessionQueryRejectsUnsupportedPrecisionAndOversizedQuery(t *testing.T) {
	f, _, _, a := queryActors(t)
	for _, query := range []string{"?since=2026-11-16T09%3A00%3A00.1234567891Z", "?q=" + strings.Repeat("a", 9000)} {
		got := f.sessionQuery(a, query)
		if got.status != 422 || got.body["rows"] != nil {
			t.Errorf("invalid precision/size status %d", got.status)
		}
	}
}

func TestSessionQueryFractionalBoundsWithDifferentOffsetsAgree(t *testing.T) {
	f, _, _, a := queryActors(t)
	first := f.sessionQuery(a, "?since=2026-11-16T09%3A00%3A00.5Z")
	second := f.sessionQuery(a, "?since=2026-11-16T10%3A00%3A00.5%2B01%3A00")
	if first.status != 200 || second.status != 200 {
		t.Fatal("valid fractional offset refused")
	}
	if first.body["query"].(map[string]any)["since"] != second.body["query"].(map[string]any)["since"] {
		t.Fatal("equivalent instants normalized differently")
	}
	if len(first.body["rows"].([]any)) != len(second.body["rows"].([]any)) {
		t.Fatal("equivalent instants changed matching rows")
	}
}

func TestSessionQuerySparseScanContinuesWithoutFalseZero(t *testing.T) {
	f, _, _, a := queryActors(t)
	for i := 0; i < 5; i++ {
		f.advance(time.Second)
		f.signedIn(MemberLogin, pw)
	}
	f.s.cfg.Access.SessionQueryScan = 2
	first := f.sessionQuery(a, "?q=not-present&limit=1")
	if first.status != 200 || len(first.body["rows"].([]any)) != 0 || first.body["next"] == "" || first.body["scan_stopped"] != true || first.body["count"].(map[string]any)["certainty"] != "lower_bound" {
		t.Fatalf("sparse query lost continuation: %v", first.body)
	}
}
