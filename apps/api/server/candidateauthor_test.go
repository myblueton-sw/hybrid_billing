package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/pricingauthority"
)

func TestCandidateAuthorPreservesV6Bytes(t *testing.T) {
	f, maker, _, fields := captureHTTPWorld(t, true)
	p := f.s.chargeProducer
	var old chargeCandidate
	err := inputAuthorityAs(f, maker, func() error {
		owned, err := p.captureBinding(SampleWorkspace, fields["binding"])
		if err != nil {
			return err
		}
		policy, err := p.pricingPolicies.Record(SampleWorkspace, fields["policy"])
		if err != nil {
			return err
		}
		input, err := f.s.eligibleCaptureInput(commands.Context{Workspace: SampleWorkspace, Fields: fields}, owned, policy)
		if err != nil {
			return err
		}
		old, err = p.captureGovernedInputLocked(p.pricingPolicies, owned.Scope, fields["policy"], fields["policy_digest"], SampleMember, owned.SourceRun, "author-baseline", owned.Binding, input, "")
		return err
	})
	if err != nil || old.Envelope.Schema != "hb_charge_candidate_v6" {
		t.Fatal(old, err)
	}
	record := p.candidates.records[[2]string{SampleWorkspace, "author-baseline"}]
	input, envelope := sha256.Sum256(record.input), sha256.Sum256(record.envelope)
	if hex.EncodeToString(input[:]) != "a1f1117164ba0a0e48f0cc863faa0e66e33375de0bc18ff61807fe06b6678772" || hex.EncodeToString(envelope[:]) != "386085a342fa0a3d0089d5f9fc391235e6585d73d2057e42f7c169ac2dfc2199" {
		t.Fatalf("9420c53 v6 fixture bytes changed: input=%x envelope=%x", input, envelope)
	}
	if _, err := summarizeCandidate(old, SampleWorkspace, "author-baseline"); err != nil {
		t.Fatal(err)
	}
	f.code(t, "no historical backfill", policyMutate(f, maker, "capture", "author-baseline", fields), 409, string(commands.EvidenceStale))
	if !bytes.Equal(record.envelope, f.s.chargeProducer.candidates.records[[2]string{SampleWorkspace, "author-baseline"}].envelope) {
		t.Fatal("historical v6 changed")
	}
}

func TestCandidateAuthorSchemaMatrix(t *testing.T) {
	f, maker, _, fields := captureHTTPWorld(t, true)
	f.ok(t, "capture", policyMutate(f, maker, "capture", "authored", fields), "captured", "")
	for _, variant := range []string{"v5", "v6", "v7", "invalid-author", "author-without-input"} {
		t.Run(variant, func(t *testing.T) {
			value, err := f.s.chargeProducer.candidates.read(SampleWorkspace, "authored")
			if err != nil {
				t.Fatal(err)
			}
			switch variant {
			case "v5":
				value.Envelope.Input.Normalized.InputEligibility = nil
				value.Envelope.Input.Normalized.CaptureAuthor = ""
			case "v6":
				value.Envelope.Input.Normalized.CaptureAuthor = ""
			case "invalid-author":
				value.Envelope.Input.Normalized.CaptureAuthor = "invalid author"
			case "author-without-input":
				value.Envelope.Input.Normalized.InputEligibility = nil
			}
			invalid := variant == "invalid-author" || variant == "author-without-input"
			store := &chargeCandidates{}
			_, err = store.capture(value.Envelope.Input)
			if invalid {
				if !errors.Is(err, errCandidateInvalid) || len(store.records) != 0 {
					t.Fatal("invalid author stored", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			for _, schema := range []string{"v5", "v6", "v7", "v8"} {
				value.Envelope.Schema = "hb_charge_candidate_" + schema
				_, err := summarizeCandidate(value, SampleWorkspace, "authored")
				if (err == nil) != (!invalid && schema == variant) {
					t.Fatal(variant, schema, err)
				}
			}
		})
	}
}

func TestCandidateAuthorHTTPIdentityAndReplay(t *testing.T) {
	f, maker, checker, fields := captureHTTPWorld(t, true)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	ceiling := pricingInspectHTTP(f, root, "policy-console-use")
	if ceiling.status != 200 {
		t.Fatal(ceiling.raw)
	}
	grantFields := pricingTimes(f)
	grantFields["capability"], grantFields["recipient"], grantFields["ceiling"] = "use", SampleAdmin3, field(ceiling, "grant")
	proposed := f.run(root, "pricing_authority.propose_human", "other-capture-human", grantFields)
	f.ok(t, "another user use proposal", proposed, pricingauthority.Pending, "")
	f.ok(t, "another user use approval", pricingApproveHTTP(f, second, "other-capture-human", proposed), pricingauthority.Applied, "")
	fields["author"] = SampleAdmin3
	f.code(t, "author override", policyMutate(f, maker, "capture", "authored", fields), 422, string(commands.InvalidRequest))
	delete(fields, "author")
	body, _ := json.Marshal(map[string]any{"request_id": "author-request", "target_id": "authored", "idempotency_key": "author-key", "expected_revision": f.s.Revision(), "rationale": "sample authorship", "body": fields})
	first := f.call("pricing_policy.capture", string(body), maker.cookie, maker.csrf)
	f.ok(t, "capture", first, "captured", "")
	replay := f.call("pricing_policy.capture", string(body), maker.cookie, maker.csrf)
	f.ok(t, "same author replay", replay, "captured", "")
	if field(first, "digest") != field(replay, "digest") {
		t.Fatal("replay changed capture")
	}
	f.code(t, "another user replay", f.call("pricing_policy.capture", string(body), checker.cookie, checker.csrf), 409, string(commands.IdempotencyConflict))
	f.code(t, "another author same candidate", policyMutate(f, checker, "capture", "authored", fields), 409, string(commands.EvidenceStale))
	f.ok(t, "other author own candidate", policyMutate(f, checker, "capture", "other-authored", fields), "captured", "")
	for id, author := range map[string]string{"authored": SampleMember, "other-authored": SampleAdmin3} {
		value, err := f.s.chargeProducer.candidates.read(SampleWorkspace, id)
		if err != nil || value.Envelope.Input.Normalized.CaptureAuthor != author {
			t.Fatal(id, value, err)
		}
	}
	before := f.s.chargeProducer.candidates.records[[2]string{SampleWorkspace, "authored"}]
	u, _, _ := f.s.findUser(SampleMember)
	if err := f.s.identity.Installation().SuspendUser(seedOperator, SampleMember, u.Epoch); err != nil {
		t.Fatal(err)
	}
	read := f.candidateRead(f.as(AdminLogin), "/authored")
	if read.status != 200 || read.body["approval_readiness"] != "unverified" {
		t.Fatal(read.raw)
	}
	after, err := f.s.chargeProducer.candidates.read(SampleWorkspace, "authored")
	if err != nil || after.Envelope.Input.Normalized.CaptureAuthor != SampleMember || !bytes.Equal(before.envelope, f.s.chargeProducer.candidates.records[[2]string{SampleWorkspace, "authored"}].envelope) {
		t.Fatal("suspension rewrote author", err)
	}
}
