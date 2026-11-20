package server

import (
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/webhooks"
)

func TestCredentialPickupReplacementDropsOldReferenceAndPreservesOtherCredential(t *testing.T) {
	d := newDirectory()
	first := pickup{credential: "credential-one", target: "service-one", expires: start.Add(time.Hour)}
	other := pickup{credential: "credential-two", target: "service-two", expires: start.Add(time.Hour)}
	d.putPickup("initial-reference", first)
	d.putPickup("other-reference", other)
	version := d.deliveryVersion
	replacement := first
	replacement.expires = start.Add(2 * time.Hour)
	d.putPickup("replacement-reference", replacement)
	if _, known := d.pickups["initial-reference"]; known {
		t.Fatal("credential pickup replacement retained the old reference")
	}
	current, known := d.credentialPickup(first.credential)
	if !known || current.target != first.target || !current.expires.Equal(replacement.expires) || d.pickupByCredential[first.credential] != "replacement-reference" || d.deliveryVersion == version {
		t.Fatal("credential pickup replacement lost canonical lookup or source invalidation")
	}
	unaffected, known := d.credentialPickup(other.credential)
	if !known || unaffected.target != other.target || !unaffected.expires.Equal(other.expires) || len(d.pickups) != 2 {
		t.Fatal("credential pickup replacement damaged an unrelated pickup")
	}
	version = d.deliveryVersion
	d.dropPickup("initial-reference")
	if current, known := d.credentialPickup(first.credential); !known || !current.expires.Equal(replacement.expires) || d.deliveryVersion != version {
		t.Fatal("old-reference cleanup removed the replacement or changed its source")
	}
}

func TestRevokedHeldRedeliveryCannotAdvertiseReapprovalToIndependentReader(t *testing.T) {
	f := setup(t, SeedSample).ready()
	id, root, _ := f.heldRedelivery(t)
	f.setRights(t, SampleAdmin2, gate.AdministerInstallation)
	if err := f.dispatch(); err != nil {
		t.Fatal("actual redelivery hold premise")
	}
	before := deliveryRowFor(t, f.integrations(root), id)
	if before["kind"] != webhooks.KindRedelivery || before["hold"] != webhooks.ReasonApproverLostRight || before["actions"].(map[string]any)["reapprove"].(map[string]any)["allowed"] != true {
		t.Fatal("independent reader could not reapprove the actual held delivery before revocation")
	}
	got := f.run(root, "integrations.subscription_revoke", subscriptionAt(sampleSubscription), nil)
	if got.status != 200 {
		t.Fatal("actual subscription revocation premise", got.status)
	}
	revision := f.s.Revision()
	snapshot := f.integrations(root)
	after := deliveryRowFor(t, snapshot, id)
	action := after["actions"].(map[string]any)["reapprove"].(map[string]any)
	if subscriptionRowFor(t, snapshot, sampleSubscription)["state"] != webhooks.SubscriptionRevoked || after["kind"] != webhooks.KindRedelivery || after["hold"] != webhooks.ReasonApproverLostRight || after["requester"] == SampleAdmin || after["approver"] == SampleAdmin || action["allowed"] != false || action["blocked"] != webhooks.ReasonRevoked || f.s.Revision() != revision {
		t.Fatal("revoked held delivery advertised reapproval or read changed canonical state", after["kind"], after["hold"], action["allowed"], action["blocked"], f.s.Revision() != revision)
	}
}
