package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func TestServiceQueryUncollectedAllStatesExcludesRealCollectionRevocationAndExpiry(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	service := f.newService(root, "collection-state")
	collected, reference, _ := f.issued(root, service, "1")
	revoked, _, _ := f.issued(root, service, "1")
	expired, _, _ := f.issued(root, service, "1")
	remaining, _, _ := f.issued(root, service, "2")
	taken := f.do(http.MethodGet, "/api/v1/identity/links/"+reference, "", func(r *http.Request) {
		r.AddCookie(root.cookie)
		r.Header.Set(csrfHeader, root.csrf)
	})
	if taken.status != 200 || taken.body["credential"] != collected {
		t.Fatal("actual original-session collection premise", taken.status)
	}
	if result := f.run(root, "service.revoke_credential", revoked, nil); result.status != 200 {
		t.Fatal("actual revocation premise", result.status)
	}
	path := "/api/v1/identity/service-credentials?service=" + service + "&state=all&collection=uncollected"
	f.assertServiceQueryIDs(root, path, []string{expired, remaining})
	observed := f.serviceQuery(root, "/api/v1/identity/service-credentials/"+collected)
	if observed.status != 200 || observed.body["row"].(map[string]any)["state"] != "active" || observed.body["row"].(map[string]any)["secret_pickup"] != "collected" {
		t.Fatal("active recorded collection premise was not preserved")
	}
	f.advance(24 * time.Hour)
	root = f.as(AdminLogin)
	f.assertServiceQueryIDs(root, path, []string{remaining})
	for id, state := range map[string]string{collected: "expired", revoked: "revoked", expired: "expired"} {
		got := f.serviceQuery(root, "/api/v1/identity/service-credentials/"+id)
		if got.status != 200 || got.body["row"].(map[string]any)["state"] != state || got.body["row"].(map[string]any)["current_state"] != state {
			t.Fatal("excluded recorded credential lifecycle was hidden or misclassified", got.status)
		}
	}
}

func TestServiceQueryPublicationEncodingFailureCannotPublishAContinuationOrRows(t *testing.T) {
	for _, field := range []string{"reference", "expires", "aggregate-expiry"} {
		t.Run(field, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			root := f.as(AdminLogin)
			f.newService(root, "alpha")
			f.newService(root, "zeta")
			r := serviceWalkRequest(root, "/api/v1/identity/services?limit=1")
			walk, ok := f.s.beginServiceQuery(httptest.NewRecorder(), r, "accounts", "")
			if !ok {
				t.Fatal("admitted encoder boundary premise")
			}
			rows, more, err := walk.accounts()
			if err != nil || !more || len(rows) != 1 {
				t.Fatal("real incomplete page premise")
			}
			invalid := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)
			switch field {
			case "reference":
				walk.cursor.Reference = invalid
			case "expires":
				walk.cursor.Expires = invalid
			case "aggregate-expiry":
				walk.cursor.Aggregate.Earliest = invalid
			}
			w := httptest.NewRecorder()
			writeServiceQueryPage(w, r, walk, rows, more, "service_id_asc")
			assertServiceQueryFailurePayload(t, serviceQueryRecordedFailure(t, w), 500, commands.InternalError)
		})
	}
}
