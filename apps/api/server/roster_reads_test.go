package server

import (
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/retention"
)

func TestFurtherReadsAskForTheRosterOnlyWhenTheirRowsNeedIt(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	got := f.retentionRead(root)
	run := retSection(got, "deletion")["rows"].([]any)[0].(map[string]any)["run"].(map[string]any)["target"].(string)
	hold := retSection(got, "holds")["rows"].([]any)[0].(map[string]any)["id"].(string)
	without := []string{
		"/api/v1/retention/artifacts?class=" + retention.ClassSource,
		"/api/v1/retention/references",
		"/api/v1/retention/tombstones",
		"/api/v1/retention/runs",
		"/api/v1/retention/run-artifacts?run=" + run,
		"/api/v1/retention/hold-outcomes?hold=" + hold,
		"/api/v1/release-evidence/paths",
		"/api/v1/release-evidence/results",
	}
	with := []string{
		"/api/v1/retention",
		"/api/v1/retention/holds",
		"/api/v1/retention/proposals?kind=policy",
		"/api/v1/release-evidence",
		"/api/v1/release-evidence/requirements",
		"/api/v1/release-evidence/profiles",
		"/api/v1/release-evidence/proposals",
	}
	before := map[string]string{}
	for _, path := range append(append([]string{}, without...), with...) {
		answer := f.readAs(root, path)
		if answer.status != 200 {
			t.Fatalf("%s: %d %s", path, answer.status, answer.raw[:min(len(answer.raw), 200)])
		}
		before[path] = answer.raw
	}
	reads := 0
	f.s.fault = func(at string) bool {
		if at == "roster" {
			reads++
		}
		return false
	}
	for _, path := range without {
		if answer := f.readAs(root, path); answer.status != 200 || reads != 0 {
			t.Errorf("%s reads the roster %d times (status %d)", path, reads, answer.status)
		}
		reads = 0
	}
	for _, path := range with {
		if answer := f.readAs(root, path); answer.status != 200 || reads == 0 {
			t.Errorf("%s does not read the roster (status %d)", path, answer.status)
		}
		reads = 0
	}
	f.s.fault = func(at string) bool { return at == "roster" }
	for _, path := range without {
		if answer := f.readAs(root, path); answer.status != 200 || answer.raw != before[path] {
			t.Errorf("%s with the roster unavailable: %d, or another answer than before", path, answer.status)
		}
	}
	for _, path := range with {
		if answer := f.readAs(root, path); answer.status != 503 {
			t.Errorf("%s with the roster unavailable: %d", path, answer.status)
		}
	}
	f.s.fault = nil
	light := f.s.retentionFactsWithoutRoster(SampleAdmin, SampleWorkspace)
	if _, err := light.holdRow(f.s.ret.holds[0]); err != errRosterNotRead {
		t.Errorf("a hold row built without the roster: %v", err)
	}
	facts := f.s.evidenceFactsWithoutRoster(SampleAdmin)
	if _, err := facts.profileRow(f.s.ev.profiles[0], nil); err != errRosterNotRead {
		t.Errorf("a profile row built without the roster: %v", err)
	}
	if _, err := facts.requirementRow(0); err != errRosterNotRead {
		t.Errorf("a requirement row built without the roster: %v", err)
	}
}
