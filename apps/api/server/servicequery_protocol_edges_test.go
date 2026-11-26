package server

import (
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func TestServiceQueryProtocolRejectsCrossAreaAndAmbiguousFilters(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	service := f.newService(root, "protocol")
	credential, _, _ := f.issued(root, service, "1")
	lists := []string{"/identity/services", "/identity/service-credentials", "/identity/service-owners", "/identity/service-organizations", "/identity/service-audiences", "/identity/service-audiences/test-service-api/scopes", "/identity/services/" + service + "/declarations"}
	common := []string{"limit=0", "limit=01", "limit=101", "limit=invalid", "limit=1&limit=2", "workspace=foreign", "workspace=" + SampleWorkspace + "&workspace=" + SampleWorkspace, "unknown=x", "after=" + strings.Repeat("x", 4097), "q=", "q=a%00b", "q=%zz", "q=" + strings.Repeat("x", 8193)}
	before := f.s.Revision()
	for _, path := range lists {
		for _, query := range common {
			got := f.serviceQuery(root, "/api/v1"+path+"?"+query)
			f.failed(t, "ambiguous or malformed query on "+path, got, commands.InvalidRequest)
			if got.body["rows"] != nil || got.body["row"] != nil || got.body["count"] != nil {
				t.Fatal("invalid query disclosed metadata", path)
			}
		}
	}
	cases := map[string][]string{
		"/identity/services":                              {"state=revoked", "expiry=unknown", "declaration=unknown", "collection=uncollected", "service=" + service, "version=1", "organization=bad!", "owner=bad!", "audience=bad!", "scope=bad!"},
		"/identity/service-credentials":                   {"state=retired", "collection=unknown", "service=bad!", "owner=" + SampleAdmin, "expiry=valid", "declaration=recorded", "version=1"},
		"/identity/service-owners":                        {"state=active", "owner=" + SampleAdmin, "audience=test-service-api", "expiry=none", "collection=uncollected", "declaration=missing", "version=1"},
		"/identity/services/" + service + "/declarations": {"q=search", "version=0", "version=01", "version=18446744073709551616", "version=1&after=opaque"},
	}
	for path, queries := range cases {
		for _, query := range queries {
			f.failed(t, "cross-area query on "+path, f.serviceQuery(root, "/api/v1"+path+"?"+query), commands.InvalidRequest)
		}
	}
	details := []string{"/identity/services/" + service, "/identity/service-credentials/" + credential, "/identity/service-owners/" + SampleAdmin, "/identity/service-organizations/sample-le-seller", "/identity/service-audiences/test-service-api", "/identity/service-audiences/test-service-api/scopes/read"}
	for _, path := range details {
		for _, query := range []string{"q=search", "workspace=foreign", "workspace=" + SampleWorkspace + "&workspace=" + SampleWorkspace, "q=%zz", "q=" + strings.Repeat("x", 8193)} {
			got := f.serviceQuery(root, "/api/v1"+path+"?"+query)
			f.failed(t, "invalid canonical detail query on "+path, got, commands.InvalidRequest)
			if got.body["row"] != nil || got.body["count"] != nil {
				t.Fatal("invalid canonical query disclosed metadata", path)
			}
		}
	}
	if f.s.Revision() != before {
		t.Fatal("refused reads changed revision")
	}
}

func TestServiceQueryDependencyLossNeverPublishesPartialRows(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	service := f.newService(root, "dependencies")
	credential, _, _ := f.issued(root, service, "1")
	paths := map[string][]string{
		"services":            {"/identity/services/" + service, "/identity/service-credentials/" + credential, "/identity/services/" + service + "/declarations"},
		"credentials":         {"/identity/service-credentials/" + credential},
		"people":              {"/identity/service-owners/" + SampleAdmin},
		"roster":              {"/identity/service-owners/" + SampleAdmin},
		"service-declaration": {"/identity/services/" + service + "/declarations"},
		"organizations":       {"/identity/services/" + service, "/identity/service-organizations/sample-le-seller", "/identity/service-audiences/test-service-api/scopes/read", "/identity/service-organizations", "/identity/service-audiences/test-service-api/scopes"},
	}
	for dependency, affected := range paths {
		f.s.fault = func(at string) bool { return at == dependency }
		before := f.s.Revision()
		for _, path := range affected {
			got := f.serviceQuery(root, "/api/v1"+path)
			f.failed(t, "unavailable "+dependency+" at "+path, got, commands.DependencyUnavailable)
			if got.body["row"] != nil || got.body["rows"] != nil || got.body["count"] != nil {
				t.Fatal("dependency loss disclosed partial projection", path)
			}
		}
		if dependency == "organizations" {
			got := f.serviceQuery(root, "/api/v1/identity/service-audiences/test-service-api")
			if got.status != 200 || got.body["row"] == nil {
				t.Fatal("configured audience falsely depended on business eligibility", got.status)
			}
		}
		if f.s.Revision() != before {
			t.Fatal("dependency refusal changed revision")
		}
	}
	f.s.fault = nil
}

func TestServiceQueryDependencyLossAtPublicationRequiresFreshRead(t *testing.T) {
	for _, path := range []string{"/identity/service-organizations", "/identity/service-organizations/sample-le-seller"} {
		t.Run(path, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			root := f.as(AdminLogin)
			checks := 0
			f.s.fault = func(at string) bool {
				if at == "organizations" {
					checks++
					return checks >= 2
				}
				return false
			}
			got := f.serviceQuery(root, "/api/v1"+path)
			f.failed(t, "dependency lost after initial admission", got, commands.DependencyUnavailable)
			if checks < 2 || got.body["row"] != nil || got.body["rows"] != nil || got.body["count"] != nil {
				t.Fatal("publication did not fence dependency loss")
			}
			f.s.fault = nil
			if fresh := f.serviceQuery(root, "/api/v1"+path); fresh.status != 200 {
				t.Fatal("fresh authoritative read did not recover", fresh.status)
			}
		})
	}
}

func TestServiceQueryEveryRouteRequiresCurrentAdministrator(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	service := f.newService(root, "route-admission")
	credential, _, _ := f.issued(root, service, "1")
	member := f.withMember()
	paths := []string{"/identity/services", "/identity/services/" + service, "/identity/services/" + service + "/declarations", "/identity/service-credentials", "/identity/service-credentials/" + credential, "/identity/service-owners", "/identity/service-owners/" + SampleAdmin, "/identity/service-organizations", "/identity/service-organizations/sample-le-seller", "/identity/service-audiences", "/identity/service-audiences/test-service-api", "/identity/service-audiences/test-service-api/scopes", "/identity/service-audiences/test-service-api/scopes/read"}
	before := f.s.Revision()
	for _, path := range paths {
		for _, caller := range []struct {
			name  string
			actor actor
			code  commands.Code
		}{{"anonymous", actor{}, commands.Unauthenticated}, {"current member", member, commands.PermissionDenied}} {
			got := f.serviceQuery(caller.actor, "/api/v1"+path)
			f.failed(t, caller.name+" must not access "+path, got, caller.code)
			for _, field := range []string{"row", "rows", "count", "admission", "limits", "catalog_version"} {
				if got.body[field] != nil {
					t.Fatal("refused caller received metadata", path, field)
				}
			}
		}
	}
	if f.s.Revision() != before {
		t.Fatal("denied reads changed revision")
	}
}
