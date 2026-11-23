package server

import (
	"net/http"
	"testing"
)

func TestSessionQueryOpenAPIDeclaresBoundedQueryAndKnownIDRead(t *testing.T) {
	f := setup(t, SeedSample).ready()
	doc := f.do(http.MethodGet, "/api/v1/openapi.json", "", nil)
	if doc.status != 200 {
		t.Fatal(doc.status)
	}
	paths := doc.body["x-self"].(map[string]any)
	list, ok := paths["/me/sessions"].(map[string]any)
	if !ok {
		t.Fatal("bounded session query missing from API contract")
	}
	operation := list["get"].(map[string]any)
	if operation["x-order"] != "current_then_signed_in_desc_handle_asc" {
		t.Fatal("query order contract differs")
	}
	parameters := operation["parameters"].([]any)
	names := map[string]bool{}
	for _, value := range parameters {
		names[value.(map[string]any)["name"].(string)] = true
	}
	for _, name := range []string{"session", "workspace", "since", "until", "q", "limit", "after"} {
		if !names[name] {
			t.Errorf("query parameter %s missing", name)
		}
	}
	responses := operation["responses"].(map[string]any)
	for _, status := range []string{"200", "401", "403", "409", "422", "503"} {
		if responses[status] == nil {
			t.Errorf("query status%s missing", status)
		}
	}
	detail, ok := paths["/me/sessions/{handle}"].(map[string]any)
	if !ok || detail["get"].(map[string]any)["responses"].(map[string]any)["404"] == nil {
		t.Fatal("authorized known-ID lookup contract missing")
	}
}
