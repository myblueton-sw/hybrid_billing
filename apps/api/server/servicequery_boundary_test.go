package server

import (
	"net/http"
	"testing"
)

func TestServiceQueryAccountPageDoesNotDownloadOwnerAndCredentialPopulations(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	svc := f.newService(root, "alpha")
	f.newService(root, "bravo")
	f.issued(root, svc, "1")
	got := f.do("GET", "/api/v1/identity/services?limit=1", "", func(r *http.Request) { r.AddCookie(root.cookie); r.Header.Set(csrfHeader, root.csrf) })
	if got.status != 200 {
		t.Fatal("authorized account query", got.status, got.body["code"])
	}
	if _, embedded := got.body["owners"]; embedded {
		t.Fatal("account query downloaded the owner population")
	}
	if _, embedded := got.body["services"]; embedded {
		t.Fatal("account query downloaded the legacy embedded population")
	}
	rows, ok := got.body["rows"].([]any)
	if !ok || len(rows) != 1 {
		t.Fatal("account query did not honor a one-row page")
	}
	for _, raw := range rows {
		if _, embedded := raw.(map[string]any)["credentials"]; embedded {
			t.Fatal("account summary contains a credential population")
		}
	}
	count, ok := got.body["count"].(map[string]any)
	if !ok || count["certainty"] != "lower_bound" || count["value"] != "1" || got.body["next"] == "" {
		t.Fatal("unfinished account scope falsely counted as complete")
	}
}
