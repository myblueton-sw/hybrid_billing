package server

import (
	"math"
	"testing"
	"time"
)

func TestServiceDeliveryIndexReplacesRemovesAndPreservesUnrelatedRecords(t *testing.T) {
	d := newDirectory()
	d.putPickup("first", pickup{credential: "one", target: "svc-one", expires: start.Add(time.Hour)})
	original := d.deliveryVersion
	d.putPickup("first", pickup{credential: "two", target: "svc-two", expires: start.Add(time.Hour)})
	if _, ok := d.credentialPickup("one"); ok {
		t.Fatal("replacement retained old credential lookup")
	}
	if p, ok := d.credentialPickup("two"); !ok || p.target != "svc-two" || d.deliveryVersion == original {
		t.Fatal("replacement lost current lookup/version")
	}
	d.putPickup("second", pickup{credential: "three", target: "svc-three", expires: start.Add(time.Hour)})
	d.dropPickupRefs([]string{"first", "missing"})
	if _, ok := d.credentialPickup("two"); ok {
		t.Fatal("removed pickup is still indexed")
	}
	if _, ok := d.credentialPickup("three"); !ok {
		t.Fatal("removing another pickup lost unrelated recovery")
	}
	d.putDelivery("two", delivery{service: "svc-two"})
	d.putDelivery("three", delivery{service: "svc-three"})
	d.forget("svc-two", map[string]bool{})
	if _, ok := d.collected["two"]; ok {
		t.Fatal("forgotten delivery remained recorded")
	}
	if _, ok := d.collected["three"]; !ok {
		t.Fatal("forget lost another service delivery")
	}
	before := d.deliveryVersion
	d.dropPickup("missing")
	d.dropDelivery("missing")
	if before != d.deliveryVersion {
		t.Fatal("no-op removal changed recovery source")
	}
	d.prune(start.Add(time.Hour))
	if len(d.pickupByCredential) != 0 {
		t.Fatal("expiry left recovery lookup entries")
	}
	d.deliveryVersion = math.MaxUint64 - 1
	d.putDelivery("four", delivery{service: "svc-four"})
	d.dropDelivery("four")
	if d.deliveryVersion != math.MaxUint64 {
		t.Fatal("delivery query generation wrapped")
	}
}

func TestServiceDeliveryCollectionAndRevocationFenceActualRecovery(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	svc := f.newService(root, "delivery")
	id, ref, _ := f.issued(root, svc, "1")
	if p, ok := f.s.dir.credentialPickup(id); !ok || p.credential != id || f.s.dir.pickupByCredential[id] != ref {
		t.Fatal("actual issuance omitted recovery lookup")
	}
	before := f.s.dir.deliveryVersion
	got := f.collect(root, ref)
	if got.status != 200 {
		t.Fatal("actual collection failed", got.status, got.body["code"])
	}
	if _, ok := f.s.dir.credentialPickup(id); ok || !f.s.dir.collected[id].collected || f.s.dir.deliveryVersion == before {
		t.Fatal("actual collection omitted recovery source change")
	}
	second, _, _ := f.issued(root, svc, "1")
	before = f.s.dir.deliveryVersion
	revoked := f.run(root, "service.revoke_credential", second, nil)
	if revoked.status != 200 {
		t.Fatal("actual revoke failed", revoked.status, revoked.body["code"])
	}
	if _, ok := f.s.dir.credentialPickup(second); ok || f.s.dir.deliveryVersion == before {
		t.Fatal("actual revocation retained pickup or stale generation")
	}
	if _, ok := f.s.dir.collected[second]; ok {
		t.Fatal("revoked delivery retained metadata")
	}
	if !f.s.dir.collected[id].collected {
		t.Fatal("revoking another credential lost returned-once metadata")
	}
}
