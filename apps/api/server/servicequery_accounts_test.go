package server

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func (f world) serviceQuery(a actor, path string) reply {
	f.t.Helper()
	return f.do("GET", path, "", func(r *http.Request) {
		if a.cookie != nil {
			r.AddCookie(a.cookie)
		}
		r.Header.Set(csrfHeader, a.csrf)
	})
}

func TestServiceQueryAccountBudgetResumesInsideOneLargeCredentialPopulation(t *testing.T) {
	f := setupWith(t, func(c *Config) {
		c.Access = DefaultAccessSettings()
		c.Access.ServiceQueryScan = 100
		c.Identity.CredentialsPerServiceUser = 200
	})
	root := f.as(AdminLogin)
	svc := f.newService(root, "large")
	verifier := root.verifier(f)
	for i := 0; i < 180; i++ {
		id := "credential-" + strings.Repeat("0", 3-len(strconv.Itoa(i))) + strconv.Itoa(i)
		f.must(f.s.identity.Workspace().AddCredential(verifier, SampleWorkspace, svc, id, identity.Verifier{byte(i + 1)}, identity.Validity{NotBefore: start, NotAfter: start.Add(time.Hour)}))
	}
	first := f.serviceQuery(root, "/api/v1/identity/services?limit=1")
	if first.status != 200 {
		t.Fatal("first bounded request", first.status, first.body["code"])
	}
	if len(first.body["rows"].([]any)) != 0 || first.body["scan_stopped"] != true || first.body["next"] == "" {
		t.Fatal("large account was emitted before its aggregate finished")
	}
	count := first.body["count"].(map[string]any)
	if count["value"] != "0" || count["certainty"] != "lower_bound" {
		t.Fatal("unfinished account was declared empty/complete")
	}
	second := f.serviceQuery(root, "/api/v1/identity/services?limit=1&after="+url.QueryEscape(first.body["next"].(string)))
	if second.status != 200 {
		t.Fatal("resume bounded request", second.status, second.body["code"])
	}
	rows := second.body["rows"].([]any)
	if len(rows) != 1 {
		t.Fatal("resuming skipped or repeated account")
	}
	summary := rows[0].(map[string]any)["credential_summary"].(map[string]any)
	if summary["recorded"] != "180" || summary["active"] != "180" || summary["expired"] != "0" || summary["revoked"] != "0" {
		t.Fatal("partial aggregate was not resumed exactly", summary)
	}
	count = second.body["count"].(map[string]any)
	if count["value"] != "1" || count["certainty"] != "exact" || second.body["next"] != "" || first.body["reference_at"] != second.body["reference_at"] {
		t.Fatal("resumed account lost snapshot/count boundary")
	}
}

func TestServiceQueryAccountCursorRejectsScopeCallerProcessAndSourceChanges(t *testing.T) {
	for _, change := range []string{"query", "caller", "process", "credential", "declaration", "tampered"} {
		t.Run(change, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			root := f.as(AdminLogin)
			svc := f.newService(root, "alpha")
			f.newService(root, "bravo")
			first := f.serviceQuery(root, "/api/v1/identity/services?limit=1")
			token := first.body["next"].(string)
			if token == "" {
				t.Fatal("continuation premise")
			}
			query := "/api/v1/identity/services?limit=1&after=" + url.QueryEscape(token)
			switch change {
			case "query":
				query += "&state=active"
			case "caller":
				root = f.as(AdminLogin)
			case "process":
				f.s.serviceQueryKey = random(32)
			case "credential":
				f.must(f.s.identity.Workspace().AddCredential(root.verifier(f), SampleWorkspace, svc, "new-credential", identity.Verifier{7}, identity.Validity{NotBefore: start, NotAfter: start.Add(time.Hour)}))
			case "declaration":
				got := f.run(root, "service.declare", svc, f.declaredFields(map[string]string{"declaration_version": "1", "purpose": "Changed purpose"}))
				if got.status != 200 {
					t.Fatal("declaration change premise", got.status)
				}
			case "tampered":
				query = "/api/v1/identity/services?limit=1&after=" + url.QueryEscape(token+"x")
			}
			f.failed(t, "cursor basis must stay exact", f.serviceQuery(root, query), commands.RevisionConflict, "after")
		})
	}
}

func TestServiceQueryAccountReferenceFreezesExpiryWithoutClaimingCurrentUsability(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	alpha := f.newService(root, "alpha")
	bravo := f.newService(root, "bravo")
	for id, until := range map[string]time.Time{alpha: start.Add(time.Hour), bravo: start.Add(30 * time.Second)} {
		f.must(f.s.identity.Workspace().AddCredential(root.verifier(f), SampleWorkspace, id, id+"-credential", identity.Verifier{byte(len(id))}, identity.Validity{NotBefore: start, NotAfter: until}))
	}
	first := f.serviceQuery(root, "/api/v1/identity/services?limit=1&expiry=valid")
	if first.status != 200 || first.body["next"] == "" {
		t.Fatal("expiry reference premise", first.status)
	}
	f.advance(31 * time.Second)
	second := f.serviceQuery(root, "/api/v1/identity/services?limit=1&expiry=valid&after="+url.QueryEscape(first.body["next"].(string)))
	if second.status != 200 {
		t.Fatal("natural time changed frozen account membership", second.status, second.body["code"])
	}
	rows := second.body["rows"].([]any)
	if len(rows) != 1 || rows[0].(map[string]any)["id"] != bravo || second.body["reference_at"] != first.body["reference_at"] {
		t.Fatal("expiry membership used request time instead of reference")
	}
}

func TestServiceQueryAccountStrictInputsAndLegacyMetadataRemainTruthful(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.newService(root, "recorded")
	f.must(f.s.identity.Workspace().CreateServiceUser(root.verifier(f), SampleWorkspace, "legacy", SampleAdmin))
	got := f.serviceQuery(root, "/api/v1/identity/services?declaration=missing")
	if got.status != 200 {
		t.Fatal("legacy query", got.status, got.body["code"])
	}
	rows := got.body["rows"].([]any)
	if len(rows) != 1 || rows[0].(map[string]any)["id"] != "legacy" {
		t.Fatal("legacy absence was not independently filterable")
	}
	declaration := rows[0].(map[string]any)["declaration"].(map[string]any)
	if declaration["state"] != "missing" || declaration["purpose"] != "" || declaration["organization"] != "" || len(declaration["scopes"].([]any)) != 0 {
		t.Fatal("legacy metadata was invented")
	}
	for _, raw := range []string{"?limit=0", "?limit=01", "?limit=101", "?limit=1&limit=2", "?scope=read", "?workspace=foreign", "?State=active", "?unknown=x", "?q=a%0Ab", "?q=", "?q=%zz"} {
		reply := f.serviceQuery(root, "/api/v1/identity/services"+raw)
		f.failed(t, "strict account query "+raw, reply, commands.InvalidRequest)
	}
	member := f.withMember()
	f.failed(t, "member cannot read service query", f.serviceQuery(member, "/api/v1/identity/services"), commands.PermissionDenied)
	anonymous := f.serviceQuery(actor{}, "/api/v1/identity/services")
	f.failed(t, "anonymous cannot read service query", anonymous, commands.Unauthenticated)
}
