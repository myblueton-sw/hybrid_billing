package server

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func TestServiceQueryCredentialCollectionIsGlobalScopedAndDoesNotRevealPickupReferences(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(Admin2Login, pw)
	issuer := f.as(AdminLogin)
	viewer := f.as(Admin2Login)
	alpha := f.newService(issuer, "alpha")
	bravo := f.newService(issuer, "bravo")
	id, ref, _ := f.issued(issuer, alpha, "1")
	other, _, _ := f.issued(issuer, bravo, "1")
	empty := f.serviceQuery(viewer, "/api/v1/identity/services?state=retired")
	if empty.status != 200 || len(empty.body["rows"].([]any)) != 0 {
		t.Fatal("independent account filter premise")
	}
	first := f.serviceQuery(viewer, "/api/v1/identity/service-credentials?collection=uncollected&limit=1")
	if first.status != 200 || first.body["next"] == "" {
		t.Fatal("global credential query", first.status, first.body["code"])
	}
	row := first.body["rows"].([]any)[0].(map[string]any)
	if row["id"] != id || row["service"] != alpha || row["secret_pickup"] != pickupOpen {
		t.Fatal("global collection page lost original-session availability")
	}
	if strings.Contains(first.raw, ref) || strings.Contains(first.raw, `"secret_ref"`) || strings.Contains(first.raw, `"secret":`) {
		t.Fatal("credential metadata exposed secret/pickup reference")
	}
	f.code(t, "another administrator cannot collect", f.collect(viewer, ref), 403, string(commands.PermissionDenied))
	if got := f.collect(issuer, ref); got.status != 200 {
		t.Fatal("issuing-session recovery failed", got.status)
	}
	stale := f.serviceQuery(viewer, "/api/v1/identity/service-credentials?collection=uncollected&limit=1&after="+url.QueryEscape(first.body["next"].(string)))
	f.failed(t, "collection changes signed membership", stale, commands.RevisionConflict, "after")
	fresh := f.serviceQuery(viewer, "/api/v1/identity/service-credentials?collection=uncollected")
	rows := fresh.body["rows"].([]any)
	if fresh.status != 200 || len(rows) != 1 || rows[0].(map[string]any)["id"] != other {
		t.Fatal("fresh collection membership did not exclude returned secret")
	}
	scoped := f.serviceQuery(viewer, "/api/v1/identity/service-credentials?service="+alpha)
	if scoped.status != 200 || len(scoped.body["rows"].([]any)) != 1 || scoped.body["rows"].([]any)[0].(map[string]any)["secret_pickup"] != pickupCollected {
		t.Fatal("service filter lost returned-once truth")
	}
}

func TestServiceQueryPickupAvailabilityIsLiveWhileCollectionMembershipIsFrozen(t *testing.T) {
	f := setupWith(t, func(c *Config) { c.Access = DefaultAccessSettings(); c.Access.ServiceQuerySeconds = 900 })
	f.setPassword(Admin2Login, pw)
	issuer := f.as(AdminLogin)
	alpha := f.newService(issuer, "alpha")
	bravo := f.newService(issuer, "bravo")
	f.issued(issuer, alpha, "1")
	beta, _, _ := f.issued(issuer, bravo, "1")
	f.advance(time.Minute)
	viewer := f.as(Admin2Login)
	first := f.serviceQuery(viewer, "/api/v1/identity/service-credentials?collection=uncollected&limit=1")
	if first.status != 200 || first.body["next"] == "" {
		t.Fatal("live observation premise", first.status)
	}
	firstRow := first.body["rows"].([]any)[0].(map[string]any)
	until, err := time.Parse(time.RFC3339Nano, firstRow["secret_pickup_until"].(string))
	f.must(err)
	before := f.s.dir.deliveryVersion
	f.advance(until.Sub(f.s.clock()) + time.Nanosecond)
	second := f.serviceQuery(viewer, "/api/v1/identity/service-credentials?collection=uncollected&limit=1&after="+url.QueryEscape(first.body["next"].(string)))
	if second.status != 200 {
		t.Fatal("natural pickup expiry invalidated recorded membership", second.status, second.body["code"])
	}
	rows := second.body["rows"].([]any)
	if len(rows) != 1 || rows[0].(map[string]any)["id"] != beta || rows[0].(map[string]any)["secret_pickup"] != pickupNever || first.body["reference_at"] != second.body["reference_at"] || f.s.dir.deliveryVersion != before {
		t.Fatal("live observation and frozen membership were conflated")
	}
	end := f.serviceQuery(viewer, "/api/v1/identity/service-credentials?collection=uncollected&limit=1&after="+url.QueryEscape(second.body["next"].(string)))
	if end.status != 200 || end.body["count"].(map[string]any)["value"] != "2" || end.body["count"].(map[string]any)["certainty"] != "exact" {
		t.Fatal("end-of-source observation lost cumulative count")
	}
}
