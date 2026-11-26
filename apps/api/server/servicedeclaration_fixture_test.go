package server

import "encoding/json"

func testServiceCatalog() []byte {
	audiences := []map[string]any{}
	for _, workspace := range []string{SampleWorkspace, EmptyWorkspace} {
		audiences = append(audiences, map[string]any{"workspace": workspace, "id": "test-service-api", "resource": "https://service.example.invalid/api", "scopes": []any{map[string]any{"id": "read", "action": "view", "kind": ReaderKind, "fields": []string{ReaderField}, "customer": map[string]any{"any": true}, "legal_entity": map[string]any{"any": true}, "contract": map[string]any{"any": true}, "period": map[string]any{"any": true}}}})
	}
	data, _ := json.Marshal(map[string]any{"audiences": audiences})
	return data
}

func (f world) declaredFields(fields map[string]string) map[string]string {
	out := map[string]string{"purpose": "Synthetic service processing", "organization": "sample-le-seller", "catalog_version": f.s.serviceCatalog.Version(), "scopes": `[{"audience":"test-service-api","scope":"read"}]`}
	for key, value := range fields {
		out[key] = value
	}
	return out
}
