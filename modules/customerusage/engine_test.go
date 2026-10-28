package customerusage

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	ca "github.com/myblueton-sw/hybrid_billing/modules/customeraccess"
)

func TestEngineGateway(t *testing.T) {
	path := os.Getenv("HB34_USAGE_PROFILE")
	if path == "" {
		t.Skip("dedicated synthetic engine profile not configured")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var profile Profile
	if err := json.Unmarshal(data, &profile); err != nil {
		t.Fatal(err)
	}
	store, err := ca.NewStore(profile.Installation, Fields(), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.PutCompany(ca.Company{ID: profile.CompanyID, Active: true}, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := store.PutTenant(ca.Tenant{ID: profile.TenantID, CompanyID: profile.CompanyID, Active: true}, 0); err != nil {
		t.Fatal(err)
	}
	scope := ca.Scope{CompanyID: profile.CompanyID, TenantID: profile.TenantID}
	subject := ca.Subject{ID: "synthetic-user", Active: true, Epoch: 1, Grants: []ca.Grant{{Scope: scope, Action: ca.UsageRead, Fields: Fields(), ExpiresAt: time.Now().Add(time.Hour)}}}
	if _, err := store.PutSubject(subject, 0); err != nil {
		t.Fatal(err)
	}
	verify := func(context.Context) (ca.Principal, error) {
		return ca.Principal{Installation: profile.Installation, SubjectID: subject.ID, Epoch: 1, ValidUntil: time.Now().Add(time.Minute)}, nil
	}
	command := []string{os.Getenv("HB34_USAGE_PYTHON"), os.Getenv("HB34_USAGE_BRIDGE")}
	request := ca.Selection{Scopes: []ca.Scope{scope}, Fields: Fields(), Limit: 10}
	makeGateway := func(p Profile) *Gateway {
		g, e := NewGateway(verify, store, command, p, Budget{Rows: 10, Bytes: 65536, Timeout: 20 * time.Second})
		if e != nil {
			t.Fatal(e)
		}
		return g
	}
	page, err := makeGateway(profile).Query(context.Background(), request)
	if err != nil || page == nil || len(page.Rows) != 1 {
		t.Fatalf("engine query: page=%v error=%v", page, err)
	}
	values := page.Rows[0].Values
	if values["quantity"] != "2.000000000000000000" || values["unit"] != "hours" || values["service"] != "Compute" || values["measurement_state"] != "source-reported" {
		t.Fatal("independent synthetic expectation mismatch")
	}
	encoded, _ := json.Marshal(page)
	for _, hidden := range []string{"PRIVATE-", "synthetic-shared-payer", "billed_cost", "source_json", "account_context", "Synthetic Shared Compute"} {
		if strings.Contains(string(encoded), hidden) {
			t.Fatal("hidden source data escaped")
		}
	}
	for _, change := range []string{"snapshot", "company", "actor", "mapping"} {
		t.Run(change, func(t *testing.T) {
			p := profile
			switch change {
			case "snapshot":
				p.Snapshot = strings.Repeat("0", 64)
			case "company":
				p.CompanyID = "wrong-company"
			case "actor":
				p.Actor = "reader-msp33-a1-r4"
			case "mapping":
				p.Services = map[string]string{"unmapped": "Compute"}
			}
			g := makeGateway(p)
			if result, e := g.Query(context.Background(), request); result != nil || e == nil {
				t.Fatal("invalid profile returned rows")
			}
		})
	}
	exportGrant := subject.Grants[0]
	exportGrant.Action = ca.UsageExport
	subject.Grants = append(subject.Grants, exportGrant)
	if _, err := store.PutSubject(subject, 1); err != nil {
		t.Fatal(err)
	}
	exports, err := NewExports(makeGateway(profile), ExportLimits{MaxJobs: 2, MaxRows: 1000, MaxBytes: 65536, TTL: time.Minute}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	id, err := exports.Create(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := exports.Run(context.Background(), id); err != nil {
		t.Fatal("engine export generation", err)
	}
	artifact, err := exports.Download(context.Background(), id)
	if err != nil {
		t.Fatal("engine download", err)
	}
	var exported ExportPackage
	if err := json.Unmarshal(artifact.Bytes, &exported); err != nil {
		t.Fatal(err)
	}
	if exported.Manifest.Completion != "complete-supported-selection" || exported.Manifest.RowCount != 1 || len(exported.Rows) != 1 || exported.Rows[0].Values["quantity"] != "2.000000000000000000" || hashBytes(artifact.Bytes) != artifact.SHA256 {
		t.Fatal("engine export manifest/payload mismatch")
	}
	for _, hidden := range []string{"PRIVATE-", "synthetic-shared-payer", "billed_cost", "source_json", "account_context"} {
		if strings.Contains(string(artifact.Bytes), hidden) {
			t.Fatal("hidden source data in artifact")
		}
	}
	subject.Grants = subject.Grants[:1]
	if _, err := store.PutSubject(subject, 2); err != nil {
		t.Fatal(err)
	}
	if data, err := exports.Download(context.Background(), id); data != nil || err != ca.ErrDenied {
		t.Fatal("revoked engine export downloaded")
	}
}
