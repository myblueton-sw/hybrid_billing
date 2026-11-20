package server

import (
	"encoding/json"
	"testing"
)

func TestServiceQueryOpenAPIDescribesBoundedReadsAndCurrentDetails(t *testing.T) {
	f := setup(t, SeedSample).ready()
	var doc map[string]any
	if err := json.Unmarshal(f.mustOpenAPI(), &doc); err != nil {
		t.Fatal(err)
	}
	reads := doc["x-reads"].(map[string]any)
	for _, path := range []string{"/identity/services", "/identity/services/{id}", "/identity/services/{id}/declarations", "/identity/service-credentials", "/identity/service-credentials/{id}", "/identity/service-owners", "/identity/service-owners/{id}", "/identity/service-organizations", "/identity/service-organizations/{id}", "/identity/service-audiences", "/identity/service-audiences/{id}", "/identity/service-audiences/{id}/scopes", "/identity/service-audiences/{id}/scopes/{scope}"} {
		entry, ok := reads[path].(map[string]any)
		if !ok {
			t.Fatal("actual read missing from API contract", path)
		}
		get := entry["get"].(map[string]any)
		if get["x-admission"] != "current_installation_administrator" {
			t.Fatal("read admission absent", path)
		}
		if _, ok := get["responses"].(map[string]any)["200"]; !ok {
			t.Fatal("actual read response absent", path)
		}
	}
	account := reads["/identity/services"].(map[string]any)["get"].(map[string]any)
	if account["x-order"] != "service_id_asc" || account["x-no-embedded-populations"] != true {
		t.Fatal("legacy whole-population contract remained")
	}
	if account["x-count-certainty"] != "cumulative_lower_bound_until_end_of_source" || account["x-admission-count"] != "exact_workspace_recorded_accounts_independent_of_filters" {
		t.Fatal("query and admission count meanings were conflated")
	}
	credential := reads["/identity/service-credentials"].(map[string]any)["get"].(map[string]any)
	if credential["x-collection-membership"] != "recorded_not_returned_and_active_at_reference" || credential["x-pickup-observation"] != "current_original_issuing_session_availability" {
		t.Fatal("collection predicate and live pickup meaning missing")
	}
}
