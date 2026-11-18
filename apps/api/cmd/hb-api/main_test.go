package main

import (
	"crypto/rand"
	"flag"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	for _, raw := range []string{"", "123456", "+5", "-5", "05", "0x10", "1.5", " 5"} {
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
