package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/inputreview"
	"github.com/myblueton-sw/hybrid_billing/modules/pricingpolicy"
	"github.com/myblueton-sw/hybrid_billing/modules/reconcile"
)

func captureHTTPWorld(t *testing.T, confirm bool) (world, actor, actor, map[string]string) {
	t.Helper()
	f, maker, checker, spec, _ := policyHTTPWorld(t)
	provisionPricingCapability(t, f, "use", SampleMember)
	provisionInputCapability(t, f, inputauthority.Confirm)
	p := policyRead(f, maker, "prepare", "capture-policy", map[string]string{"spec": spec})
	proposal := policyMutate(f, maker, "propose", "capture-policy", policyProposalFields(p, spec))
	f.ok(t, "capture proposal", proposal, pricingpolicy.Pending, "")
	f.ok(t, "capture policy approval", policyMutate(f, checker, "approve", "capture-policy", map[string]string{"digest": field(proposal, "digest")}), pricingpolicy.Approved, "")
	review := inputRecord(t, f)
	if confirm {
		f.ok(t, "confirm input", decideInput(f, maker, "confirm_publish", review.Evidence.Digest, "0"), inputreview.Eligible, "")
	}
	return f, maker, checker, map[string]string{"binding": "sample-charge-binding", "input_review": sampleInputReview, "input_digest": review.Evidence.Digest, "scope_revision": "1", "policy": "capture-policy", "policy_digest": field(proposal, "digest")}
}

func TestCandidateCaptureHTTPVerifiedInputAndPolicy(t *testing.T) {
	f, maker, _, fields := captureHTTPWorld(t, true)
	f.code(t, "administrator has no use grant", policyMutate(f, f.as(AdminLogin), "capture", "verified-candidate", fields), 403, string(commands.PermissionDenied))
	got := policyMutate(f, maker, "capture", "verified-candidate", fields)
	f.ok(t, "captured", got, "captured", "")
	if field(got, "receipt") != "historical_capture" || field(got, "rating_total") != "" || len(field(got, "digest")) != 64 {
		t.Fatal("capture receipt", got.raw)
	}
	candidate, err := f.s.chargeProducer.candidates.read(SampleWorkspace, "verified-candidate")
	if err != nil || candidate.Envelope.Schema != "hb_charge_candidate_v7" || candidate.Envelope.Input.Normalized.CaptureAuthor != SampleMember || candidate.Envelope.Result.Total != "15" || candidate.Envelope.Result.Currency != "USD" {
		t.Fatal("exact calculation", candidate, err)
	}
	input := candidate.Envelope.Input.Normalized.InputEligibility
	if input == nil || input.Review.ID != sampleInputReview || input.Publication.Revision != 1 || input.Review.Decision.Actor != SampleMember || candidate.Envelope.Input.Normalized.Pricing.Digest != fields["policy_digest"] {
		t.Fatal("missing immutable prerequisites", input)
	}
	read := f.candidateRead(f.as(AdminLogin), "/verified-candidate")
	if read.status != 200 || read.body["approval_readiness"] != "unverified" {
		t.Fatal(read.raw)
	}
	provenance := read.body["record"].(map[string]any)["provenance"].(map[string]any)
	recorded := provenance["recorded_input"].(map[string]any)
	if recorded["review"] != sampleInputReview || recorded["source_run"] != "sample-normalized-charge" || whole(recorded["scope_revision"]) != 1 || len(recorded) != 9 {
		t.Fatal("input projection", recorded)
	}
	before := append([]byte(nil), f.s.chargeProducer.candidates.records[[2]string{SampleWorkspace, "verified-candidate"}].envelope...)
	input.Review.Decision.Actor = "changed"
	input.Publication.Digest = "changed"
	after, err := f.s.chargeProducer.candidates.read(SampleWorkspace, "verified-candidate")
	if err != nil || after.Envelope.Input.Normalized.InputEligibility.Review.Decision.Actor != SampleMember || !bytes.Equal(before, f.s.chargeProducer.candidates.records[[2]string{SampleWorkspace, "verified-candidate"}].envelope) {
		t.Fatal("returned alias changed history", err)
	}
	f.code(t, "use does not grant cost", f.candidateRead(maker, "/verified-candidate"), 403, string(commands.PermissionDenied))
}

func TestCandidateCaptureHTTPPrerequisitesAndFailedIDReuse(t *testing.T) {
	f, maker, _, fields := captureHTTPWorld(t, false)
	f.code(t, "awaiting input", policyMutate(f, maker, "capture", "reusable", fields), 422, string(commands.PrerequisiteFailed))
	if _, err := f.s.chargeProducer.candidates.read(SampleWorkspace, "reusable"); !errors.Is(err, errCandidateMissing) {
		t.Fatal("failed capture reserved ID", err)
	}
	f.ok(t, "confirm", decideInput(f, maker, "confirm_publish", fields["input_digest"], "0"), inputreview.Eligible, "")
	for _, name := range []string{"scope_revision", "input_digest", "policy_digest", "binding", "policy", "input_review", "Policy", "Profile", "Limits"} {
		bad := map[string]string{}
		for k, v := range fields {
			bad[k] = v
		}
		bad[name] = "other"
		got := policyMutate(f, maker, "capture", "reusable", bad)
		if got.status == 200 {
			t.Fatal("invalid override accepted", name)
		}
		if _, err := f.s.chargeProducer.candidates.read(SampleWorkspace, "reusable"); !errors.Is(err, errCandidateMissing) {
			t.Fatal("invalid capture stored", name, err)
		}
	}
	f.ok(t, "reuse after refusal", policyMutate(f, maker, "capture", "reusable", fields), "captured", "")
}

func TestCandidateCaptureHTTPFreshChecksPrecedeCachedReturn(t *testing.T) {
	for _, cause := range []string{"policy-revoked", "target-revision", "document-revision", "voided-basis", "source-bytes", "foreign-review-producer"} {
		t.Run(cause, func(t *testing.T) {
			f, maker, checker, fields := captureHTTPWorld(t, true)
			first := policyMutate(f, maker, "capture", "cached", fields)
			f.ok(t, "capture", first, "captured", "")
			before := f.s.chargeProducer.candidates.records[[2]string{SampleWorkspace, "cached"}]
			held := f.s.inputReviews.bindings[[2]string{SampleWorkspace, sampleInputReview}]
			switch cause {
			case "policy-revoked":
				f.ok(t, "revoke", policyMutate(f, checker, "revoke", fields["policy"], map[string]string{"digest": fields["policy_digest"]}), pricingpolicy.Revoked, "")
			case "target-revision":
				profile, rev, err := f.s.orgs.BillingProfile(SampleWorkspace, "sample-profile-north")
				if err != nil {
					t.Fatal(err)
				}
				if _, err = f.s.orgs.PutBillingProfile(profile, rev); err != nil {
					t.Fatal(err)
				}
			case "document-revision":
				key := candidateDocumentKey{held.Document.Scope, held.Document.ID}
				value := f.s.inputReviews.documents[key]
				value.Revision++
				f.s.inputReviews.documents[key] = value
			case "voided-basis":
				if err := f.s.inputReviews.basis.Void(SampleWorkspace, held.Spec.ID, reconcile.WrongFigures); err != nil {
					t.Fatal(err)
				}
			case "source-bytes":
				f.s.chargeProducer.sources[[2]string{SampleWorkspace, "sample-normalized-charge"}].Packet[0] = 'x'
			case "foreign-review-producer":
				f.s.inputReviews.producer = &normalizedCharges{}
			}
			got := policyMutate(f, maker, "capture", "cached", fields)
			want := map[string]struct {
				status int
				code   commands.Code
			}{"policy-revoked": {403, commands.PermissionDenied}, "target-revision": {409, commands.EvidenceStale}, "document-revision": {409, commands.EvidenceStale}, "voided-basis": {422, commands.InvalidRequest}, "source-bytes": {503, commands.DependencyUnavailable}, "foreign-review-producer": {501, commands.UnsupportedCapability}}[cause]
			f.code(t, cause, got, want.status, string(want.code))
			after := f.s.chargeProducer.candidates.records[[2]string{SampleWorkspace, "cached"}]
			if !reflect.DeepEqual(before, after) {
				t.Fatal("historical bytes changed")
			}
		})
	}
}

func TestCandidateCaptureHTTPHistoricalReplay(t *testing.T) {
	f, maker, checker, fields := captureHTTPWorld(t, true)
	body, _ := json.Marshal(map[string]any{"request_id": "capture-replay", "target_id": "historical-candidate", "idempotency_key": "capture-replay-key", "expected_revision": f.s.Revision(), "rationale": "sample capture", "body": fields})
	first := f.call("pricing_policy.capture", string(body), maker.cookie, maker.csrf)
	f.ok(t, "capture", first, "captured", "")
	f.ok(t, "revoke", policyMutate(f, checker, "revoke", fields["policy"], map[string]string{"digest": fields["policy_digest"]}), pricingpolicy.Revoked, "")
	before := f.s.counter
	replay := f.call("pricing_policy.capture", string(body), maker.cookie, maker.csrf)
	if replay.status != 200 || field(replay, "digest") != field(first, "digest") || field(replay, "receipt") != "historical_capture" || f.s.counter != before {
		t.Fatal(replay.raw)
	}
	f.code(t, "fresh request after policy revoked", policyMutate(f, maker, "capture", "historical-candidate", fields), 403, string(commands.PermissionDenied))
	for _, grant := range f.s.admin.Grants() {
		if grant.SubjectID == SampleMember && grant.Terms.Action == gate.UsePricingPolicy {
			if err := f.s.admin.RevokeGrant(grant.ID, grant.Version); err != nil {
				t.Fatal(err)
			}
		}
	}
	f.code(t, "revoked use refuses historical replay", f.call("pricing_policy.capture", string(body), maker.cookie, maker.csrf), 403, string(commands.PermissionDenied))
}

func TestCandidateCapturePreservesV5Bytes(t *testing.T) {
	f := setup(t, SeedSample).ready()
	record := f.s.chargeProducer.candidates.records[[2]string{SampleWorkspace, "sample-charge-01"}]
	inputHash, envelopeHash := sha256.Sum256(record.input), sha256.Sum256(record.envelope)
	if hex.EncodeToString(inputHash[:]) != "835018a297fdf605f6009da828182ca479ead64d7317bfea899c38f1de2aa041" || hex.EncodeToString(envelopeHash[:]) != "12faa76aa32eaba3c39a9782f3c290e9177a8c66af0f1c2ee0a4f492f78577d4" {
		t.Fatal("a1d7ce1 sample v5 bytes changed")
	}
	if strings.Contains(string(record.envelope), "InputEligibility") {
		t.Fatal("nil provenance serialized")
	}
}
