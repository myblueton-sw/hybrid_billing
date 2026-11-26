package server

import (
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/authpolicy"
)

func TestServiceLegacyEnumerationTraversesFullBatchesAndRefusesExpiredCaller(t *testing.T) {
	f := setupWith(t, func(c *Config) { c.Identity.PageSize = 1 })
	root := f.as(AdminLogin)
	first, second := f.newService(root, "legacy-first"), f.newService(root, "legacy-second")
	rows, err := f.s.serviceUsers(root.verifier(f))
	f.must(err)
	if len(rows) != 2 || rows[0].ID != first || rows[1].ID != second {
		t.Fatal("legacy full-batch enumeration omitted, duplicated or reordered actual services")
	}
	f.advance(48 * time.Hour)
	rows, err = f.s.serviceUsers(root.verifier(f))
	if err == nil || rows != nil || f.me(root).status != 401 {
		t.Fatal("expired legacy caller received a partial service population")
	}
}

func TestServicePickupObservationRequiresIssuerAuthenticationFreshness(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(AdminLogin, pw)
	f.setPassword(Admin2Login, pw)
	f.directPolicy(authpolicy.Revision{MaxAge: time.Minute})
	issuer := f.signedIn(AdminLogin, pw)
	service := f.newService(issuer, "freshness-contract")
	credential, _, _ := f.issued(issuer, service, "1")
	pending, known := f.s.dir.credentialPickup(credential)
	if !known || pending.session != issuer.verifier(f) {
		t.Fatal("actual original issuer pickup premise")
	}
	if until, live := f.s.pickupLive(pending); !live || !until.After(f.s.clock()) {
		t.Fatal("fresh original issuer pickup was not open")
	}
	f.advance(2 * time.Minute)
	found := capturedAdmissionSession(f, issuer)
	answer, idle, err := f.s.identity.Boundary().Peek(issuer.verifier(f), SampleWorkspace)
	f.must(err)
	if f.s.stale(found) || !f.s.clock().Before(idle) || !f.s.clock().Before(answer.NotAfter()) || !f.s.clock().Before(pending.expires) {
		t.Fatal("authentication age was not isolated from epoch, identity and pickup deadlines")
	}
	if _, live := f.s.pickupLive(pending); live {
		t.Fatal("stale issuer authentication kept the original pickup open")
	}
	observer := f.signedIn(Admin2Login, pw)
	got := f.serviceQuery(observer, "/api/v1/identity/service-credentials/"+credential)
	if got.status != 200 {
		t.Fatal("fresh observer could not read canonical credential metadata", got.status)
	}
	row := got.body["row"].(map[string]any)
	if row["secret_pickup"] != pickupNever || row["secret_pickup_until"] != nil && row["secret_pickup_until"] != "" {
		t.Fatal("fresh observer advertised an expired-authentication pickup as open")
	}
	still, known := f.s.dir.credentialPickup(credential)
	if !known || still.session != pending.session || !still.expires.Equal(pending.expires) || f.me(observer).status != 200 {
		t.Fatal("pickup observation rewrote the original producer or deadline")
	}
}
