package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestServiceQueryWireSecurityCodesAndClosedRestrictions(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	var document map[string]any
	if err := json.Unmarshal(f.mustOpenAPI(), &document); err != nil {
		t.Fatal(err)
	}
	reads := document["x-reads"].(map[string]any)
	for path, raw := range reads {
		if path == "/identity/users" || path == "/identity/users/count" || path == "/identity/bulk-revocations" || path == "/integrations" {
			continue
		}
		responses := raw.(map[string]any)["get"].(map[string]any)["responses"].(map[string]any)
		for status, want := range map[string][]string{"403": {"permission_denied", CodeCSRF}, "503": {"dependency_unavailable", "authority_unavailable"}} {
			codes, _ := responses[status].(map[string]any)["x-codes"].([]any)
			for _, code := range want {
				found := false
				for _, candidate := range codes {
					if candidate == code {
						found = true
					}
				}
				if !found {
					t.Fatal("wire contract omitted actual security boundary code", path, status, code)
				}
			}
		}
	}
	get := reads["/identity/service-audiences/{id}/scopes"].(map[string]any)["get"].(map[string]any)
	schema := get["responses"].(map[string]any)["200"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)["properties"].(map[string]any)["rows"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)["restrictions"].(map[string]any)
	properties, ok := schema["properties"].(map[string]any)
	if !ok || schema["additionalProperties"] != false {
		t.Fatal("restrictions are still an unspecified object")
	}
	for _, name := range []string{"id", "action", "kind", "fields", "customer", "legal_entity", "contract", "period"} {
		if properties[name] == nil {
			t.Fatal("restriction field missing", name)
		}
	}
	for _, name := range []string{"customer", "legal_entity", "contract", "period"} {
		branches, ok := properties[name].(map[string]any)["oneOf"].([]any)
		if !ok || len(branches) != 2 {
			t.Fatal("exclusive selector/period shape absent", name)
		}
		for _, raw := range branches {
			if raw.(map[string]any)["additionalProperties"] != false {
				t.Fatal("selector/period accepts ambiguous extra members", name)
			}
		}
	}
	response := f.serviceQuery(root, "/api/v1/identity/service-audiences/test-service-api/scopes")
	if response.status != 200 || len(response.body["rows"].([]any)) != 1 {
		t.Fatal("real restriction payload premise", response.status)
	}
	if folder := os.Getenv("HB_QUERY_SCHEMA_ARTIFACTS"); folder != "" {
		if err := os.MkdirAll(folder, 0o700); err != nil {
			t.Fatal(err)
		}
		artifacts := map[string]any{"restrictions-schema.json": schema, "restrictions-payload.json": response.body["rows"].([]any)[0].(map[string]any)["restrictions"]}
		for name, value := range artifacts {
			data, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(filepath.Join(folder, name), data, 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
}
