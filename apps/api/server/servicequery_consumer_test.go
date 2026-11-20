package server

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServiceQueryConsumerWireSamples(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	service := f.newService(root, "wire")
	credential, _, _ := f.issued(root, service, "1")
	detail := f.serviceQuery(root, "/api/v1/identity/services/"+service)
	if detail.status != 200 {
		t.Fatal("known service premise", detail.status)
	}
	row := detail.body["row"].(map[string]any)
	declaration := row["declaration"].(map[string]any)
	ref := declaration["scopes"].([]any)[0].(map[string]any)
	audience, scope := ref["audience"].(string), ref["scope"].(string)
	ids := map[string]string{"service": service, "credential": credential, "owner": row["owner"].(string), "organization": declaration["organization"].(string), "audience": audience, "scope": scope, "workspace": SampleWorkspace}
	paths := map[string]string{
		"accounts":      "/identity/services",
		"account":       "/identity/services/" + service,
		"credentials":   "/identity/service-credentials",
		"credential":    "/identity/service-credentials/" + credential,
		"owners":        "/identity/service-owners",
		"owner":         "/identity/service-owners/" + ids["owner"],
		"organizations": "/identity/service-organizations",
		"organization":  "/identity/service-organizations/" + ids["organization"],
		"audiences":     "/identity/service-audiences",
		"scopes":        "/identity/service-audiences/" + audience + "/scopes",
		"scope":         "/identity/service-audiences/" + audience + "/scopes/" + scope,
		"history":       "/identity/services/" + service + "/declarations",
	}
	samples := map[string]any{}
	for name, path := range paths {
		got := f.serviceQuery(root, "/api/v1"+path+"?workspace="+SampleWorkspace)
		if got.status != 200 {
			t.Fatal("consumer wire sample", name, got.status, got.body["code"])
		}
		samples[name] = map[string]any{"path": "/api/v1" + path + "?workspace=" + SampleWorkspace, "body": got.body}
	}
	unicodeQueries := []map[string]any{}
	for _, q := range []string{"ΟΣ", "İ", "\u0085Search\u0085", "\ufeffSearch\ufeff", "한국어", "العربية", strings.Repeat("Ⱥ", 32)} {
		got := f.serviceQuery(root, "/api/v1/identity/service-owners?workspace="+SampleWorkspace+"&q="+url.QueryEscape(q))
		if got.status != 200 {
			t.Fatal("Unicode query premise", got.status)
		}
		unicodeQueries = append(unicodeQueries, map[string]any{"input": q, "body": got.body})
	}
	first := f.serviceQuery(root, "/api/v1/identity/service-owners?workspace="+SampleWorkspace+"&q=ADMIN&limit=1")
	after, _ := first.body["next"].(string)
	if first.status != 200 || after == "" {
		t.Fatal("equivalent normalized continuation premise", first.status)
	}
	continued := f.serviceQuery(root, "/api/v1/identity/service-owners?workspace="+SampleWorkspace+"&q=+admin+&limit=1&after="+url.QueryEscape(after))
	if continued.status != 200 || continued.body["search_input"] != " admin " || continued.body["query"].(map[string]any)["q"] != "admin" || continued.body["reference_at"] != first.body["reference_at"] {
		t.Fatal("request correlation changed normalized cursor scope", continued.status)
	}
	if folder := os.Getenv("HB_SERVICE_QUERY_SAMPLES"); folder != "" {
		if err := os.MkdirAll(folder, 0o700); err != nil {
			t.Fatal(err)
		}
		data, err := json.Marshal(map[string]any{"ids": ids, "samples": samples, "unicode_queries": unicodeQueries, "equivalent_continuation": map[string]any{"query": map[string]any{"workspace": SampleWorkspace, "q": " admin ", "limit": 1, "after": after}, "body": continued.body}})
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(folder, "m04-wire.json"), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}
