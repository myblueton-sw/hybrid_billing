package server

import (
	"net/http"
	"testing"
)

func TestSessionSelectionOpenAPIDescribesFrozenAndBoundedOperations(t *testing.T) {
	f := setup(t, SeedSample).ready()
	doc := f.do(http.MethodGet, "/api/v1/openapi.json", "", nil)
	if doc.status != 200 {
		t.Fatal(doc.status)
	}
	paths := doc.body["x-self"].(map[string]any)
	for _, path := range []string{"/me/sessions/selections", "/me/sessions/selections/{reference}", "/me/sessions/operations", "/me/sessions/operations/{operation}", "/me/sessions/operations/{operation}/items"} {
		if paths[path] == nil {
			t.Errorf("missing selection/result contract %s", path)
		}
	}
	operation := paths["/me/sessions/end"].(map[string]any)["post"].(map[string]any)
	schema := operation["requestBody"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
	if variants := schema["oneOf"].([]any); len(variants) != 3 {
		t.Fatal("exclusive selected termination contract missing")
	}
}

func objectShapeMatches(schema map[string]any, value map[string]any) bool {
	if required, ok := schema["required"].([]string); ok {
		for _, key := range required {
			if _, exists := value[key]; !exists {
				return false
			}
		}
	}
	if schema["additionalProperties"] == false {
		properties := schema["properties"].(map[string]any)
		for key := range value {
			if properties[key] == nil {
				return false
			}
		}
	}
	return true
}

func assertExclusiveObjectShape(t *testing.T, variants []any, value map[string]any) {
	t.Helper()
	matches := 0
	for _, variant := range variants {
		if objectShapeMatches(variant.(map[string]any), value) {
			matches++
		}
	}
	if matches != 1 {
		t.Fatalf("payload matched %d object branches, want exactly one", matches)
	}
}

func TestSessionSelectionOpenAPIHasExclusiveRealPayloadShapes(t *testing.T) {
	f, _, _, a := queryActors(t)
	prepared := f.selfPost(a, "sessions/selections", "{}")
	if _, _, err := f.s.openFor(LoginTriple(MemberLogin), SampleWorkspace, false); err != nil {
		t.Fatal(err)
	}
	selected := f.selfPost(a, "sessions/end", selectionEndBody(prepared))
	if selected.status != 202 {
		t.Fatal(selected.status)
	}
	end := selfPaths()["/me/sessions/end"].(map[string]any)["post"].(map[string]any)
	variants := end["responses"].(map[string]any)["202"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)["oneOf"].([]any)
	t.Run("selected", func(t *testing.T) { assertExclusiveObjectShape(t, variants, selected.body) })
	legacy := f.selfPost(a, "sessions/end", f.othersBody(a))
	if legacy.status != 202 {
		t.Fatal(legacy.status)
	}
	t.Run("legacy", func(t *testing.T) { assertExclusiveObjectShape(t, variants, legacy.body) })
	contract := sessionOperationReadContract("items")["responses"].(map[string]any)["200"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
	rowVariants := contract["properties"].(map[string]any)["rows"].(map[string]any)["items"].(map[string]any)["oneOf"].([]any)
	for _, view := range []string{"results", "not_included"} {
		t.Run(view, func(t *testing.T) {
			page := f.operationRead(a, "/"+selected.body["operation"].(string)+"/items?view="+view)
			if page.status != 200 || len(page.body["rows"].([]any)) == 0 {
				t.Fatal("missing actual operation rows")
			}
			for _, row := range page.body["rows"].([]any) {
				assertExclusiveObjectShape(t, rowVariants, row.(map[string]any))
			}
		})
	}
}
