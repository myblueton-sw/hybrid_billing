package main

import (
	"crypto/rand"
	"errors"
	"flag"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/apps/api/server"
	"github.com/myblueton-sw/hybrid_billing/modules/decisionledger"
)

func TestRun(t *testing.T) {
	for name, args := range map[string][]string{
		"flag":          {"--nope"},
		"listen":        {"--listen", "0.0.0.0:0", "--data", t.TempDir() + "/d"},
		"seed":          {"--listen", "127.0.0.1:0", "--seed", "bad", "--data", t.TempDir() + "/d"},
		"origin":        {"--listen", "127.0.0.1:0", "--console-origin", "nope", "--data", t.TempDir() + "/d"},
		"busy":          {"--listen", "127.0.0.1:1", "--data", t.TempDir() + "/d"},
		"bad data":      {"--listen", "127.0.0.1:0", "--data", "/dev/null/x"},
		"reader form":   {"--listen", "127.0.0.1:0", "--data", t.TempDir() + "/d", "--reader", "nope"},
		"reader target": {"--listen", "127.0.0.1:0", "--data", t.TempDir() + "/d", "--reader", "/api/focus-overview=http://10.0.0.1:1"},
	} {
		if err := run(args, make(chan os.Signal)); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()
	stop := make(chan os.Signal, 1)
	done := make(chan error, 1)
	go func() {
		done <- run([]string{"--listen", addr, "--data", t.TempDir() + "/d", "--console-origin", "http://127.0.0.1:5173", "--reader", "/api/focus-overview=http://127.0.0.1:1"}, stop)
	}()
	var resp *http.Response
	for deadline := time.Now().Add(30 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if resp, err = http.Get("http://" + addr + "/api/v1/health"); err == nil {
			break
		}
	}
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("health %v", err)
	}
	resp.Body.Close()
	req, _ := http.NewRequest("GET", "http://"+addr+"/api/v1/health", nil)
	req.Header.Set("X-Big", strings.Repeat("a", 40<<10))
	if big, err := http.DefaultClient.Do(req); err == nil {
		if big.StatusCode != 431 {
			t.Errorf("header limit %d", big.StatusCode)
		}
		big.Body.Close()
	}
	stop <- os.Interrupt
	if err := <-done; err != nil {
		t.Errorf("shutdown %v", err)
	}
	var o origins
	o.Set("a")
	o.Set("b")
	if o.String() != "a,b" {
		t.Error("origins flag")
	}
}

func TestAccessSettingFlags(t *testing.T) {
	for name, args := range map[string][]string{
		"out of bound":           {"--access-request-hours", "721"},
		"signed":                 {"--access-request-hours", "+5"},
		"negative":               {"--access-pending-per-proposer", "-5"},
		"hex":                    {"--access-ended-kept", "0x10"},
		"leading zero":           {"--access-ended-shown", "05"},
		"over length":            {"--access-revocations-kept", "100000"},
		"empty":                  {"--decision-ledger-max-mib", ""},
		"workspace below":        {"--access-pending-per-proposer", "30", "--access-pending-per-workspace", "20"},
		"shown above kept":       {"--access-ended-kept", "50", "--access-ended-shown", "51"},
		"nothing to replace":     {"--accept-new-decision-ledger"},
		"zero":                   {"--decision-ledger-max-mib", "0"},
		"ledger bound too small": {"--decision-ledger-max-mib", "4097"},
	} {
		full := append([]string{"--listen", "127.0.0.1:0", "--data", t.TempDir() + "/d"}, args...)
		if err := run(full, make(chan os.Signal)); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	if n, ok := settingNumber("12345"); !ok || n != 12345 {
		t.Error("five digits")
	}
	if n, ok := settingNumber("100000"); !ok || n != 100000 {
		t.Error("six digits")
	}
	if n, ok := settingNumber("20000000"); !ok || n != 20000000 {
		t.Error("eight digits")
	}
	for _, raw := range []string{"", "123456789", "+5", "-5", "05", "0x10", "1.5", " 5"} {
		if n, ok := settingNumber(raw); ok || n != 0 {
			t.Errorf("%q gave %d %v", raw, n, ok)
		}
	}
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	access, given := accessFlags(fs)
	if err := fs.Parse([]string{"--access-request-hours", "+5"}); err == nil || !strings.Contains(err.Error(), "whole number without sign or leading zero") {
		t.Errorf("a refused value is named %v", err)
	}
	fs = flag.NewFlagSet("t", flag.ContinueOnError)
	access, given = accessFlags(fs)
	if err := fs.Parse([]string{"--access-request-hours", "24", "--accept-new-decision-ledger"}); err != nil {
		t.Fatal(err)
	}
	given()
	if access.RequestHours != 24 || !access.Given["access_request_hours"] || access.Given["access_ended_kept"] || !access.AcceptNewLedger || access.EndedKept != 1000 {
		t.Errorf("flags %+v", access)
	}
	fs = flag.NewFlagSet("t", flag.ContinueOnError)
	access, _ = accessFlags(fs)
	if err := fs.Parse([]string{"--accept-new-decision-ledger", "--accept-trust-reset", "--carry-trust-from", "abc", "--trust-generation", "5"}); err != nil {
		t.Fatal(err)
	}
	if !access.AcceptTrustReset || access.CarryTrustFrom != "abc" || access.TrustGeneration != 5 {
		t.Errorf("trust flags %+v", access)
	}
	for raw, ok := range map[string]bool{"4294967294": true, "4294967295": false, "4294967296": false, "18446744073709551615": false, "0": false, "+1": false, "x": false} {
		fs = flag.NewFlagSet("t", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		access, _ = accessFlags(fs)
		err := fs.Parse([]string{"--trust-generation", raw})
		if (err == nil) != ok || err != nil && !strings.Contains(err.Error(), "from 1 to 4294967294") {
			t.Errorf("trust-generation %s: %v", raw, err)
		}
	}
	if _, ok := settingNumber("0"); !ok {
		t.Error("zero is canonical and refused by its bound")
	}
}

func TestAccessSettingFlagsReachTheRead(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()
	stop := make(chan os.Signal, 1)
	done := make(chan error, 1)
	dir := t.TempDir() + "/d"
	go func() {
		done <- run([]string{"--listen", addr, "--data", dir, "--access-request-hours", "24", "--decision-ledger-max-mib", "1"}, stop)
	}()
	for deadline := time.Now().Add(30 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if resp, err := http.Get("http://" + addr + "/api/v1/health"); err == nil {
			resp.Body.Close()
			break
		}
	}
	stop <- os.Interrupt
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(filepath.Join(dir, "decisions", "ledger.jsonl")); err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("ledger %v", err)
	}
	l, err := decisionledger.Open(decisionledger.Options{Dir: dir, MaxBytes: 2 << 20, Clock: time.Now, Random: rand.Reader})
	if err != nil {
		t.Fatal(err)
	}
	big := decisionledger.Decision{Workspace: "other-workspace", Decision: decisionledger.RevokeGrant, Target: "gone", Revocation: "revocation-1", RequestedBy: "user-a"}
	for len(big.EndedDelegations) < 40 {
		big.EndedDelegations = append(big.EndedDelegations, "delegation-with-a-long-identifier")
	}
	for size := int64(0); size <= 1<<20; {
		info, _ := os.Stat(filepath.Join(dir, "decisions", "ledger.jsonl"))
		size = info.Size()
		if _, err := l.Commit(big); err != nil {
			t.Fatal(err)
		}
	}
	l.Close()
	if err := run([]string{"--listen", addr, "--data", dir, "--decision-ledger-max-mib", "1"}, make(chan os.Signal)); err == nil || !strings.Contains(err.Error(), "decision_ledger_max_mib") {
		t.Errorf("a bound below the ledger size %v", err)
	}
}

func TestOrgSettingFlags(t *testing.T) {
	for name, args := range map[string][]string{
		"below":        {"--org-filter-scan", "99"},
		"above":        {"--org-filter-scan", "10001"},
		"signed":       {"--org-filter-scan", "+500"},
		"leading zero": {"--org-filter-scan", "0500"},
		"empty":        {"--org-filter-scan", ""},
	} {
		full := append([]string{"--listen", "127.0.0.1:0", "--data", t.TempDir() + "/d"}, args...)
		if err := run(full, make(chan os.Signal)); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	org, given := orgFlags(fs)
	if err := fs.Parse([]string{"--org-filter-scan", "0x10"}); err == nil || !strings.Contains(err.Error(), "org-filter-scan must be a whole number") {
		t.Errorf("a refused value is named %v", err)
	}
	fs = flag.NewFlagSet("t", flag.ContinueOnError)
	org, given = orgFlags(fs)
	if err := fs.Parse(nil); err != nil {
		t.Fatal(err)
	}
	given()
	if org.FilterScan != 1000 || org.Given["org_filter_scan"] {
		t.Errorf("defaults %+v", org)
	}
	fs = flag.NewFlagSet("t", flag.ContinueOnError)
	org, given = orgFlags(fs)
	if err := fs.Parse([]string{"--org-filter-scan", "250"}); err != nil {
		t.Fatal(err)
	}
	given()
	if org.FilterScan != 250 || !org.Given["org_filter_scan"] {
		t.Errorf("flags %+v", org)
	}
	var usage strings.Builder
	fs.SetOutput(&usage)
	fs.PrintDefaults()
	if !strings.Contains(usage.String(), "org_filter_scan in timelines, 100 to 10000 (default 1000)") {
		t.Errorf("usage %s", usage.String())
	}
}

func TestOrgSettingFlagAtItsLowerBoundStarts(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()
	stop := make(chan os.Signal, 1)
	done := make(chan error, 1)
	go func() {
		done <- run([]string{"--listen", addr, "--data", t.TempDir() + "/d", "--org-filter-scan", "100"}, stop)
	}()
	var resp *http.Response
	for deadline := time.Now().Add(30 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if resp, err = http.Get("http://" + addr + "/api/v1/health"); err == nil {
			resp.Body.Close()
			break
		}
	}
	if err != nil {
		t.Fatal(err)
	}
	stop <- os.Interrupt
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestIdentitySettingFlags(t *testing.T) {
	for name, args := range map[string][]string{
		"longest period above its bound": {"--role-period-max-days", "3661"},
		"tolerance zero":                 {"--clock-skew-tolerance-hours", "0"},
		"warning signed":                 {"--role-period-warning-days", "+5"},
		"targets hex":                    {"--bulk-revocation-max-targets", "0x10"},
		"lifetime leading zero":          {"--bulk-revocation-manifest-minutes", "05"},
		"pending above its bound":        {"--bulk-revocation-pending", "201"},
		"kept below its bound":           {"--bulk-revocation-jobs-kept", "9"},
		"warning above the longest":      {"--role-period-max-days", "10", "--role-period-warning-days", "11"},
		"an unknown clock base":          {"--clock-base", "host"},
		"a clock base without a lead":    {"--clock-base", "wall"},
	} {
		full := append([]string{"--listen", "127.0.0.1:0", "--data", t.TempDir() + "/d"}, args...)
		if err := run(full, make(chan os.Signal)); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	access, given := accessFlags(fs)
	if err := fs.Parse([]string{"--bulk-revocation-max-targets", "5000", "--clock-base", "ledger"}); err != nil {
		t.Fatal(err)
	}
	given()
	if access.BulkMaxTargets != 5000 || !access.Given["bulk_revocation_max_targets"] || access.Given["role_period_max_days"] || access.RolePeriodMaxDays != 366 ||
		access.ClockBase != "ledger" || access.ClockSkewHours != 24 || access.RoleWarningDays != 14 || access.BulkMinutes != 30 || access.BulkPending != 20 || access.BulkKept != 100 {
		t.Errorf("identity flags %+v", access)
	}
}

func TestViewGroupDraftFlags(t *testing.T) {
	for name, args := range map[string][]string{
		"open below":      {"--view-group-drafts-open", "0"},
		"open above":      {"--view-group-drafts-open", "201"},
		"members above":   {"--view-group-draft-members", "10001"},
		"kept below":      {"--view-group-drafts-kept", "49"},
		"activated below": {"--view-group-activated-kept", "99"},
		"activated above": {"--view-group-activated-kept", "100001"},
	} {
		full := append([]string{"--listen", "127.0.0.1:0", "--data", t.TempDir() + "/d"}, args...)
		if err := run(full, make(chan os.Signal)); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	org, given := orgFlags(fs)
	if err := fs.Parse([]string{"--view-group-drafts-open", "200", "--view-group-draft-members", "10000", "--view-group-drafts-kept", "50", "--view-group-activated-kept", "100"}); err != nil {
		t.Fatal(err)
	}
	given()
	if org.DraftsOpen != 200 || org.DraftMembers != 10000 || org.DraftsKept != 50 || org.ActivatedKept != 100 || org.FilterScan != 1000 ||
		!org.Given["view_group_drafts_open"] || !org.Given["view_group_activated_kept"] || org.Given["org_filter_scan"] {
		t.Errorf("flags %+v", org)
	}
	var usage strings.Builder
	fs.SetOutput(&usage)
	fs.PrintDefaults()
	for _, want := range []string{"view_group_drafts_open in drafts, 1 to 200 (default 16)", "view_group_draft_members in members, 1 to 10000 (default 1000)",
		"view_group_drafts_kept in drafts, 50 to 10000 (default 200)", "view_group_activated_kept in drafts, 100 to 100000 (default 1000)"} {
		if !strings.Contains(usage.String(), want) {
			t.Errorf("usage missing %s", want)
		}
	}
}
func TestUserFilterScanFlag(t *testing.T) {
	for name, args := range map[string][]string{
		"below":        {"--user-filter-scan", "99"},
		"above":        {"--user-filter-scan", "10001"},
		"leading zero": {"--user-filter-scan", "0500"},
	} {
		full := append([]string{"--listen", "127.0.0.1:0", "--data", t.TempDir() + "/d"}, args...)
		if err := run(full, make(chan os.Signal)); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	access, given := accessFlags(fs)
	if err := fs.Parse([]string{"--user-filter-scan", "250"}); err != nil {
		t.Fatal(err)
	}
	given()
	if access.UserFilterScan != 250 || !access.Given["user_filter_scan"] {
		t.Errorf("flags %+v", access.UserFilterScan)
	}
	var usage strings.Builder
	fs.SetOutput(&usage)
	fs.PrintDefaults()
	if !strings.Contains(usage.String(), "user_filter_scan in people, 100 to 10000 (default 1000)") {
		t.Errorf("usage %s", usage.String())
	}
}

func TestPluginLicenseFlags(t *testing.T) {
	parse := func(args ...string) *server.PluginSettings {
		t.Helper()
		fs := flag.NewFlagSet("t", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		p, given := pluginFlags(fs)
		if err := fs.Parse(args); err != nil {
			t.Fatal(err)
		}
		given()
		return p
	}
	cases := map[string]struct {
		args []string
		want server.PluginSettings
	}{
		"absent":   {nil, server.PluginSettings{}},
		"empty":    {[]string{"--plugin-licenses="}, server.PluginSettings{Raw: "", Given: true, Uses: 1}},
		"list":     {[]string{"--plugin-licenses", "MIT, ISC"}, server.PluginSettings{Raw: "MIT, ISC", Given: true, Uses: 1}},
		"repeated": {[]string{"--plugin-licenses", "MIT", "--plugin-licenses", "ISC"}, server.PluginSettings{Raw: "ISC", Given: true, Uses: 2}},
	}
	for name, c := range cases {
		if got := parse(c.args...); *got != c.want {
			t.Errorf("%s: %+v", name, *got)
		}
	}
	for name, args := range map[string][]string{
		"space":            {"--plugin-licenses", "Apache-2.0, MIT"},
		"repeated":         {"--plugin-licenses", "Apache-2.0", "--plugin-licenses", "Apache-2.0"},
		"seed needs":       {"--plugin-licenses", "MIT"},
		"empty with seed":  {"--plugin-licenses="},
		"trailing comma":   {"--plugin-licenses", "Apache-2.0,"},
		"malformed member": {"--plugin-licenses", "Apache-2.0,MIT OR ISC"},
	} {
		full := append([]string{"--listen", "127.0.0.1:0", "--data", t.TempDir() + "/d"}, args...)
		err := run(full, make(chan os.Signal))
		if !errors.Is(err, server.ErrConfig) || !strings.HasSuffix(err.Error(), "plugin_licenses") {
			t.Errorf("%s: %v", name, err)
		}
	}
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	pluginFlags(fs)
	var usage strings.Builder
	fs.SetOutput(&usage)
	fs.PrintDefaults()
	if !strings.Contains(usage.String(), "(default Apache-2.0,BSD-2-Clause,BSD-3-Clause,ISC,LGPL-2.1-only,LGPL-3.0-only,MIT,MPL-2.0,Zlib)") {
		t.Errorf("usage %s", usage.String())
	}
}

func TestMatchingSettingFlags(t *testing.T) {
	for name, args := range map[string][]string{
		"matching below":     {"--bulk-revocation-max-matching", "0"},
		"matching above":     {"--bulk-revocation-max-matching", "100001"},
		"matching below max": {"--bulk-revocation-max-targets", "600", "--bulk-revocation-max-matching", "599"},
		"count scan below":   {"--user-count-scan", "999"},
		"count scan above":   {"--user-count-scan", "100001"},
		"retained below":     {"--bulk-revocation-retained-people-max", "9999"},
		"retained above":     {"--bulk-revocation-retained-people-max", "20000001"},
		"retained exceeded":  {"--bulk-revocation-jobs-kept", "1000", "--bulk-revocation-pending", "200", "--bulk-revocation-max-targets", "5000", "--bulk-revocation-max-matching", "5000"},
	} {
		full := append([]string{"--listen", "127.0.0.1:0", "--data", t.TempDir() + "/d"}, args...)
		if err := run(full, make(chan os.Signal)); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	full := []string{"--listen", "127.0.0.1:0", "--data", t.TempDir() + "/d", "--bulk-revocation-jobs-kept", "1000", "--bulk-revocation-pending", "200",
		"--bulk-revocation-max-targets", "5000", "--bulk-revocation-max-matching", "5000"}
	err := run(full, make(chan os.Signal))
	for _, want := range []string{"bulk_revocation_retained_people_max 1100000", "bulk_revocation_jobs_kept 1000", "bulk_revocation_pending 200", "bulk_revocation_max_matching 5000", "= 11000000"} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not name %q: %v", want, err)
		}
	}
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	access, given := accessFlags(fs)
	if err := fs.Parse([]string{"--bulk-revocation-max-matching", "100000", "--user-count-scan", "100000", "--bulk-revocation-retained-people-max", "20000000"}); err != nil {
		t.Fatal(err)
	}
	given()
	if access.BulkMaxMatching != 100000 || access.UserCountScan != 100000 || access.BulkRetainedMax != 20000000 || !access.Given["bulk_revocation_max_matching"] ||
		!access.Given["user_count_scan"] || !access.Given["bulk_revocation_retained_people_max"] {
		t.Errorf("flags %+v", access)
	}
	var usage strings.Builder
	fs.SetOutput(&usage)
	fs.PrintDefaults()
	for _, want := range []string{"bulk_revocation_max_matching in people, 1 to 100000 (default 5000)", "user_count_scan in people, 1000 to 100000 (default 20000)",
		"bulk_revocation_retained_people_max in people, 10000 to 20000000 (default 1100000); start is refused when (2 × bulk_revocation_jobs_kept + bulk_revocation_pending) × bulk_revocation_max_matching exceeds it"} {
		if !strings.Contains(usage.String(), want) {
			t.Errorf("usage missing %s", want)
		}
	}
}
