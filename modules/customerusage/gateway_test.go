package customerusage

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	ca "github.com/myblueton-sw/hybrid_billing/modules/customeraccess"
)

func TestBridgeHelper(t *testing.T) {
	if len(os.Args) < 4 || os.Args[len(os.Args)-3] != "usage-helper" {
		return
	}
	mode, marker := os.Args[len(os.Args)-2], os.Args[len(os.Args)-1]
	input := bufio.NewReader(os.Stdin)
	line, _ := input.ReadBytes('\n')
	var req struct {
		Profile string
		Limit   int
		Mode    string
	}
	if json.Unmarshal(line, &req) != nil {
		os.Exit(2)
	}
	var p Profile
	if json.Unmarshal([]byte(req.Profile), &p) != nil {
		os.Exit(2)
	}
	if mode == "oversized" {
		fmt.Println(strings.Repeat("x", 5000))
		os.Exit(0)
	}
	if mode == "malformed" {
		fmt.Println("not-json")
		os.Exit(0)
	}
	if mode == "timeout" {
		time.Sleep(5 * time.Second)
		os.Exit(0)
	}
	if mode == "paused" {
		os.WriteFile(marker, []byte("started"), 0600)
		for {
			if _, err := os.Stat(marker + "-release"); err == nil {
				break
			}
			time.Sleep(time.Millisecond)
		}
	}
	digest := sha256.Sum256([]byte(req.Profile))
	values := map[string]string{"service": "Compute", "quantity": "0.000000000000000000", "unit": "hours", "usage_start": "2026-08-01T00:00:00+00:00", "usage_end": "2026-08-02T00:00:00+00:00", "measurement_state": "source-reported"}
	if mode == "changed" {
		values["quantity"] = "1.000000000000000000"
	}
	if mode == "formula" {
		values["service"] = "=HYPERLINK(\"https://example.invalid\")"
	}
	row := ca.Row{Scope: ca.Scope{CompanyID: p.CompanyID, TenantID: p.TenantID}, ID: strings.Repeat("b", 64), Values: values}
	if mode == "foreign" {
		row.Scope.CompanyID = "foreign"
	}
	if mode == "hidden" {
		row.Values["cost"] = "private"
	}
	revision := hex.EncodeToString(digest[:])
	if mode == "revision" {
		revision = "wrong"
	}
	response := map[string]any{"status": "ready", "revision": revision, "rows": []ca.Row{row}}
	if req.Mode == "export" && mode != "page-marker" {
		response["selection"] = "complete-supported-selection"
		response["row_count"] = 1
	}
	if mode == "wrong-count" {
		response["row_count"] = 2
	}
	json.NewEncoder(os.Stdout).Encode(response)
	if mode == "dead" {
		os.Exit(0)
	}
	line, _ = input.ReadBytes('\n')
	if string(line) != "release\n" {
		os.Exit(2)
	}
	if mode == "bad-ack" {
		fmt.Println(`{"status":"wrong"}`)
		os.Exit(0)
	}
	fmt.Println(`{"status":"released"}`)
	if mode == "trailing" {
		fmt.Println("private trailing")
	}
	if mode == "exit-failure" {
		os.Exit(2)
	}
	os.Exit(0)
}

func testGateway(t *testing.T, mode string, timeout time.Duration) (*Gateway, *ca.Store, ca.Subject, ca.Selection, string) {
	t.Helper()
	p := Profile{Kind: "source-usage-v1", Installation: "00000000-0000-0000-0000-000000000002", Source: "synthetic", Actor: "reader-synthetic", CompanyID: "company-a", TenantID: "00000000-0000-0000-0000-000000000001", Snapshot: strings.Repeat("a", 64), Start: "2026-08-01T00:00:00Z", End: "2026-09-01T00:00:00Z", Grain: "month", Services: map[string]string{"source": "Compute"}, Units: map[string]string{"source": "hours"}}
	store, err := ca.NewStore(p.Installation, Fields(), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.PutCompany(ca.Company{ID: p.CompanyID, Active: true}, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := store.PutTenant(ca.Tenant{ID: p.TenantID, CompanyID: p.CompanyID, Active: true}, 0); err != nil {
		t.Fatal(err)
	}
	scope := ca.Scope{CompanyID: p.CompanyID, TenantID: p.TenantID}
	subject := ca.Subject{ID: "subject", Active: true, Epoch: 1, Grants: []ca.Grant{{Scope: scope, Action: ca.UsageRead, Fields: Fields(), ExpiresAt: time.Now().Add(time.Hour)}}}
	if _, err := store.PutSubject(subject, 0); err != nil {
		t.Fatal(err)
	}
	verify := func(context.Context) (ca.Principal, error) {
		return ca.Principal{Installation: p.Installation, SubjectID: "subject", Epoch: 1, ValidUntil: time.Now().Add(time.Minute)}, nil
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(t.TempDir(), "marker")
	g, err := NewGateway(verify, store, []string{executable, "-test.run=^TestBridgeHelper$", "usage-helper", mode, marker}, p, Budget{Rows: 10, Bytes: 4096, Timeout: timeout})
	if err != nil {
		t.Fatal(err)
	}
	p.Services["source"] = "mutated"
	if g.profile.Services["source"] != "Compute" {
		t.Fatal("profile aliased")
	}
	return g, store, subject, ca.Selection{Scopes: []ca.Scope{scope}, Fields: Fields(), Limit: 2}, marker
}

func TestSuccessfulHeldBridge(t *testing.T) {
	g, _, _, request, _ := testGateway(t, "ok", 5*time.Second)
	page, err := g.Query(context.Background(), request)
	if err != nil || page == nil || len(page.Rows) != 1 {
		t.Fatalf("page=%v err=%v", page, err)
	}
	if page.Rows[0].Values["quantity"] != "0.000000000000000000" || page.AmountBasis != "usage-only" || page.Currency != "not-applicable" || page.ProfileRevision != g.revision || page.Snapshot != g.profile.Snapshot {
		t.Fatal("wrong result basis")
	}
}

func TestBridgeFailuresReturnNoPage(t *testing.T) {
	for _, mode := range []string{"oversized", "malformed", "foreign", "hidden", "revision", "dead", "bad-ack", "trailing", "exit-failure", "timeout"} {
		t.Run(mode, func(t *testing.T) {
			timeout := 5 * time.Second
			if mode == "timeout" {
				timeout = 100 * time.Millisecond
			}
			g, _, _, request, _ := testGateway(t, mode, timeout)
			page, err := g.Query(context.Background(), request)
			if page != nil || err == nil {
				t.Fatalf("page=%v err=%v", page, err)
			}
		})
	}
}

func TestRevocationWhileBridgeIsPaused(t *testing.T) {
	g, store, subject, request, marker := testGateway(t, "paused", 5*time.Second)
	type result struct {
		page *Page
		err  error
	}
	done := make(chan result, 1)
	go func() { p, e := g.Query(context.Background(), request); done <- result{p, e} }()
	deadline := time.Now().Add(3 * time.Second)
	for {
		if _, err := os.Stat(marker); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("child did not start")
		}
		time.Sleep(time.Millisecond)
	}
	subject.Grants = nil
	if _, err := store.PutSubject(subject, 1); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(marker+"-release", []byte("release"), 0600); err != nil {
		t.Fatal(err)
	}
	r := <-done
	if r.page != nil || r.err != ca.ErrDenied {
		t.Fatalf("page=%v err=%v", r.page, r.err)
	}
}

func TestCallerCannotChooseSourceOrHiddenField(t *testing.T) {
	g, _, _, request, _ := testGateway(t, "timeout", time.Second)
	request.Fields = []string{"cost"}
	if p, e := g.Query(context.Background(), request); p != nil || e != ca.ErrDenied {
		t.Fatal("hidden field allowed")
	}
	request.Fields = Fields()
	request.Scopes[0].TenantID = "foreign"
	if p, e := g.Query(context.Background(), request); p != nil || e != ca.ErrDenied {
		t.Fatal("foreign scope allowed")
	}
	request.Scopes[0].TenantID = g.profile.TenantID
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if p, e := g.Query(ctx, request); p != nil || e != context.Canceled {
		t.Fatal("cancelled request allowed")
	}
}
