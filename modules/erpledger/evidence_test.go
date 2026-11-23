package erpledger

import (
	"testing"
	"time"
)

func TestPreparationEvidenceReadsCurrentIndependentValues(t *testing.T) {
	f := newFixture(t)
	mapping := must(f.r.MappingOf("w1", "api", "product", "compute"))
	survey := must(f.r.SurveyOf("w1", "api", "acme", "2026-10"))
	count := f.r.workspaces["w1"].mappingCount
	mapping.Mapping.External, mapping.Author = "tampered", "other"
	survey.Survey.Evidence, survey.Author = "tampered", "other"
	got := must(f.r.MappingOf("w1", "api", "product", "compute"))
	if got.Mapping.External != "P-1" || got.Author != "admin" || got.Revision != 1 {
		t.Fatal(got)
	}
	seen := must(f.r.SurveyOf("w1", "api", "acme", "2026-10"))
	if seen.Survey.Evidence != "survey-api" || seen.Author != "admin" {
		t.Fatal(seen)
	}
	if f.r.workspaces["w1"].mappingCount != count || len(f.r.workspaces["w1"].surveys) != 1 || len(f.r.workspaces["w1"].submissions) != 0 {
		t.Fatal("reads mutated evidence collections")
	}
	f.clock.advance(time.Minute)
	changed := must(f.r.Map("w1", Mapping{"api", "product", "compute", "P-NEW"}, human("mapper")))
	latest := must(f.r.MappingOf("w1", "api", "product", "compute"))
	if latest != changed || latest.Revision != 2 || latest.At.Equal(got.At) {
		t.Fatal(latest, changed)
	}
	replaced := must(f.r.Survey("w1", Survey{Connection: "api", Customer: "acme", Period: "2026-10", Outcome: SurveyComplete, Evidence: "new-survey"}, human("surveyor")))
	if latest := must(f.r.SurveyOf("w1", "api", "acme", "2026-10")); latest != replaced || latest.At.Equal(seen.At) {
		t.Fatal(latest)
	}
}

func TestPreparationEvidenceReadsValidateAndIsolateKeys(t *testing.T) {
	f := newFixture(t)
	for _, tc := range []struct{ ws, conn, kind, platform, want string }{
		{"", "api", "product", "compute", "invalid"}, {"w1", "", "product", "compute", "invalid"}, {"w1", "api", "unknown", "compute", "invalid"}, {"w1", "api", "product", "bad id", "invalid"},
		{"foreign", "api", "product", "compute", "not_found"}, {"w1", "missing", "product", "compute", "not_found"}, {"w1", "api", "product", "missing", "not_found"},
	} {
		expect(t, "mapping key", second(f.r.MappingOf(tc.ws, tc.conn, tc.kind, tc.platform)), tc.want)
	}
	for _, tc := range []struct{ ws, conn, customer, period, want string }{
		{"", "api", "acme", "2026-10", "invalid"}, {"w1", "", "acme", "2026-10", "invalid"}, {"w1", "api", "bad id", "2026-10", "invalid"}, {"w1", "api", "acme", "2026-1", "invalid"}, {"w1", "api", "acme", "2026-13", "invalid"},
		{"foreign", "api", "acme", "2026-10", "not_found"}, {"w1", "missing", "acme", "2026-10", "not_found"}, {"w1", "api", "missing", "2026-10", "not_found"}, {"w1", "api", "acme", "2026-11", "not_found"},
	} {
		expect(t, "survey key", second(f.r.SurveyOf(tc.ws, tc.conn, tc.customer, tc.period)), tc.want)
	}
}
