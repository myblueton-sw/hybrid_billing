package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/inputreview"
	"github.com/myblueton-sw/hybrid_billing/modules/normalize"
	"github.com/myblueton-sw/hybrid_billing/modules/pricingpolicy"
	"github.com/myblueton-sw/hybrid_billing/modules/rating"
)

func TestCandidateCaptureHTTPPublicationSupersession(t *testing.T) {
	f, maker, _, fields := captureHTTPWorld(t, true)
	f.ok(t, "first capture", policyMutate(f, maker, "capture", "old-publication", fields), "captured", "")
	old := f.s.chargeProducer.candidates.records[[2]string{SampleWorkspace, "old-publication"}]
	held := f.s.inputReviews.bindings[[2]string{SampleWorkspace, sampleInputReview}]
	document := held.Document
	document.Revision = 2
	body, _ := json.Marshal(document)
	next, err := f.s.inputReviews.prepareLocked(SampleWorkspace, "sample-normalized-charge", "next-input", body, nil)
	if err != nil {
		t.Fatal(err)
	}
	confirmed := f.runWith(maker, f.s.Revision(), "input.confirm_publish", "next-input", map[string]string{"digest": next.Evidence.Digest, "scope_revision": "1"})
	f.ok(t, "new publication", confirmed, inputreview.Eligible, "")
	f.code(t, "old publication", policyMutate(f, maker, "capture", "old-publication", fields), 409, string(commands.EvidenceStale))
	fields["input_review"], fields["input_digest"], fields["scope_revision"] = "next-input", next.Evidence.Digest, "2"
	f.code(t, "same ID new evidence", policyMutate(f, maker, "capture", "old-publication", fields), 409, string(commands.EvidenceStale))
	f.ok(t, "new evidence new ID", policyMutate(f, maker, "capture", "new-publication", fields), "captured", "")
	current, err := f.s.chargeProducer.candidates.read(SampleWorkspace, "new-publication")
	if err != nil || current.Envelope.Result.Total != "15" || current.Envelope.Input.Normalized.InputEligibility.Publication.Revision != 2 || current.Digest == old.digest {
		t.Fatal("new publication not bound", current, err)
	}
	if !bytes.Equal(old.envelope, f.s.chargeProducer.candidates.records[[2]string{SampleWorkspace, "old-publication"}].envelope) {
		t.Fatal("supersession rewrote history")
	}
	in := current.Envelope.Input
	original, err := normalizedDigest(in)
	if err != nil {
		t.Fatal(err)
	}
	in.Normalized.InputEligibility.Publication.Revision++
	changed, err := normalizedDigest(in)
	if err != nil || original == changed {
		t.Fatal("publication revision not digested")
	}
	if _, err = f.s.chargeProducer.candidates.capture(in); !errors.Is(err, errCandidateInvalid) {
		t.Fatal("inconsistent publication accepted", err)
	}
}

func TestCandidateCaptureBindingsAreOwnedAndImmutable(t *testing.T) {
	f := setup(t, SeedSample).ready()
	p := f.s.chargeProducer
	owned, err := p.captureBinding(SampleWorkspace, "sample-charge-binding")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.ValueOf(owned.Binding.Policy).IsZero() || !reflect.ValueOf(owned.Binding.Profile).IsZero() || !reflect.ValueOf(owned.Binding.Limits).IsZero() {
		t.Fatal("pricing terms retained in source binding")
	}
	owned.Binding.Classes[0].Class = rating.Licence
	fresh, err := p.captureBinding(SampleWorkspace, "sample-charge-binding")
	if err != nil || fresh.Binding.Classes[0].Class != rating.CSPUsage {
		t.Fatal("binding alias escaped", err)
	}
	if _, err = p.captureBinding("foreign", "sample-charge-binding"); !errors.Is(err, errCandidateMissing) {
		t.Fatal("workspace binding leak", err)
	}
	seed, err := p.candidates.read(SampleWorkspace, "sample-charge-01")
	if err != nil {
		t.Fatal(err)
	}
	b := seed.Envelope.Input.Normalized.Binding
	if err := p.retainCaptureBindingLocked(SampleWorkspace, "sample-normalized-charge", b); err != nil {
		t.Fatal("identical retention", err)
	}
	b.Version = "2"
	if err := p.retainCaptureBindingLocked(SampleWorkspace, "sample-normalized-charge", b); !errors.Is(err, errCandidateConflict) {
		t.Fatal("immutable binding overwritten", err)
	}
}

func TestCandidateCaptureHTTPRejectedInputAndChangedMapping(t *testing.T) {
	for _, cause := range []string{"rejected", "mapping"} {
		t.Run(cause, func(t *testing.T) {
			f, maker, _, fields := captureHTTPWorld(t, cause != "rejected")
			if cause == "rejected" {
				f.ok(t, "reject", decideInput(f, maker, "reject", fields["input_digest"], "0"), inputreview.Rejected, "")
			} else {
				p := f.s.chargeProducer
				run, err := p.normal.Run(SampleWorkspace, "sample-normalized-charge")
				if err != nil {
					t.Fatal(err)
				}
				mapping, err := p.normal.Revision(run.Spec.Mapping, run.Spec.Revision)
				if err != nil {
					t.Fatal(err)
				}
				for i := range mapping.Entries {
					if mapping.Entries[i].Field == "class" {
						mapping.Entries[i].Source = "service"
					}
				}
				if _, err = p.normal.DeclareMapping(run.Spec.Mapping, run.Spec.Revision, mapping.Entries); err != nil {
					t.Fatal(err)
				}
			}
			got := policyMutate(f, maker, "capture", "refused-input", fields)
			if cause == "rejected" {
				f.code(t, cause, got, 422, string(commands.PrerequisiteFailed))
			} else {
				f.code(t, cause, got, 409, string(commands.EvidenceStale))
			}
			if _, err := f.s.chargeProducer.candidates.read(SampleWorkspace, "refused-input"); !errors.Is(err, errCandidateMissing) {
				t.Fatal(err)
			}
		})
	}
}

func TestCandidateCaptureHTTPEqualTotalWrongSourceRun(t *testing.T) {
	f, maker, _, fields := captureHTTPWorld(t, true)
	p := f.s.chargeProducer
	var packet sourcePacket
	if err := json.Unmarshal(p.sources[[2]string{SampleWorkspace, "sample-normalized-charge"}].Packet, &packet); err != nil {
		t.Fatal(err)
	}
	next := nextSourcePacket(t, packet, "different-normalization", 2)
	next.Artifacts[0].Rows[0].Cells["service"] = normalize.Cell{Text: "another-service"}
	if _, err := p.assembleSourceLocked(next); err != nil {
		t.Fatal(err)
	}
	document := f.s.inputReviews.bindings[[2]string{SampleWorkspace, sampleInputReview}].Document
	document.Revision = 2
	body, _ := json.Marshal(document)
	review, err := f.s.inputReviews.prepareLocked(SampleWorkspace, "different-normalization", "other-run-review", body, nil)
	if err != nil {
		t.Fatal(err)
	}
	f.ok(t, "confirm different source", f.runWith(maker, f.s.Revision(), "input.confirm_publish", review.ID, map[string]string{"digest": review.Evidence.Digest, "scope_revision": "1"}), inputreview.Eligible, "")
	fresh, _, err := f.s.inputReviews.currentEvidenceLocked(SampleWorkspace, review.Evidence.Run)
	if err != nil || fresh.Digest != review.Evidence.Digest || fresh.Verdict != inputreview.Match || document.Total != "15.0000" {
		t.Fatal("invalid equal-total fixture", fresh, err)
	}
	fields["input_review"], fields["input_digest"], fields["scope_revision"] = review.ID, review.Evidence.Digest, "2"
	f.code(t, "equal total does not bind source", policyMutate(f, maker, "capture", "wrong-run-candidate", fields), 422, string(commands.PrerequisiteFailed))
	if _, err := p.candidates.read(SampleWorkspace, "wrong-run-candidate"); !errors.Is(err, errCandidateMissing) {
		t.Fatal(err)
	}
}

func TestCandidateCaptureHTTPRefusesDifferentCostBasis(t *testing.T) {
	f, maker, checker, fields := captureHTTPWorld(t, true)
	f.ok(t, "revoke first policy", policyMutate(f, checker, "revoke", fields["policy"], map[string]string{"digest": fields["policy_digest"]}), pricingpolicy.Revoked, "")
	spec := policyHTTPSpec()
	spec.Policy.ID = "effective-policy"
	spec.Policy.CostBasis = rating.Effective
	text := policySpecText(t, spec)
	p := policyRead(f, maker, "prepare", "effective-policy", map[string]string{"spec": text})
	if p.status != 200 {
		t.Fatal(p.raw)
	}
	proposed := policyMutate(f, maker, "propose", "effective-policy", policyProposalFields(p, text))
	f.ok(t, "effective proposal", proposed, pricingpolicy.Pending, "")
	f.ok(t, "effective approval", policyMutate(f, checker, "approve", "effective-policy", map[string]string{"digest": field(proposed, "digest")}), pricingpolicy.Approved, "")
	fields["policy"], fields["policy_digest"] = "effective-policy", field(proposed, "digest")
	f.code(t, "billed review cannot justify effective basis", policyMutate(f, maker, "capture", "wrong-basis", fields), 422, string(commands.PrerequisiteFailed))
}

func TestCandidateCaptureHTTPHoldStepUpAndConcurrentPolicyRevoke(t *testing.T) {
	f, maker, checker, fields := captureHTTPWorld(t, true)
	f.advance(16 * time.Minute)
	f.code(t, "stepup", policyMutate(f, maker, "capture", "held-candidate", fields), 403, CodeReauthentication)
	maker = f.as(MemberLogin)
	checker = f.as(Admin3Login)
	f.s.domain.Lock()
	f.s.enterHold("synthetic-test")
	f.s.domain.Unlock()
	f.code(t, "held", policyMutate(f, maker, "capture", "held-candidate", fields), 503, string(commands.DependencyUnavailable))
	f.s.hold = nil
	revision := f.s.Revision()
	var capture, revoke reply
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		capture = f.runWith(maker, revision, "pricing_policy.capture", "racing-candidate", fields)
	}()
	go func() {
		defer wg.Done()
		revoke = f.runWith(checker, revision, "pricing_policy.revoke", fields["policy"], map[string]string{"digest": fields["policy_digest"]})
	}()
	wg.Wait()
	if capture.status == 200 {
		f.code(t, "serialized revoke", revoke, 409, string(commands.RevisionConflict))
		f.ok(t, "subsequent revoke", policyMutate(f, checker, "revoke", fields["policy"], map[string]string{"digest": fields["policy_digest"]}), "revoked", "")
	} else {
		f.ok(t, "serialized revoke", revoke, "revoked", "")
		f.code(t, "serialized capture", capture, 409, string(commands.RevisionConflict))
	}
	f.code(t, "fresh capture after revoke", policyMutate(f, maker, "capture", "racing-candidate", fields), 403, string(commands.PermissionDenied))
}
