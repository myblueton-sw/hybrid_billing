package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
)

func subjectPolicy(t *testing.T, f world, id, revision, required string) {
	t.Helper()
	fields := financialFields(revision)
	fields["required"] = required
	p := f.run(f.as(AdminLogin), "financial_policy.propose", id, fields)
	f.ok(t, "policy proposal", p, approvals.StatePending, "")
	f.ok(t, "policy confirmation", f.run(f.as(Admin2Login), "financial_policy.confirm", id, map[string]string{"digest": field(p, "digest")}), approvals.StateConfirmed, "")
}

func subjectWorld(t *testing.T) (world, actor, actor, map[string]string, string) {
	t.Helper()
	f, maker, checker, fields := captureHTTPWorld(t, true)
	got := policyMutate(f, maker, "capture", "subject-candidate", fields)
	f.ok(t, "capture", got, "captured", "")
	subjectPolicy(t, f, "subject-policy", "2", "attribution,input,mapping,result,rule")
	return f, maker, checker, fields, field(got, "digest")
}

func readSubject(f world, request, candidate, digest string) (approvals.Subject, error) {
	f.s.domain.Lock()
	defer f.s.domain.Unlock()
	return f.s.financialSubjectLocked(SampleWorkspace, request, candidate, digest)
}

func TestFinancialSubjectExactCurrentEvidenceAndImmutableAuthor(t *testing.T) {
	f, maker, _, fields, digest := subjectWorld(t)
	before := f.s.chargeProducer.candidates.records[[2]string{SampleWorkspace, "subject-candidate"}]
	count, grants, revision := len(f.s.chargeProducer.candidates.records), len(f.s.admin.Grants()), f.s.Revision()
	a, err := readSubject(f, "financial-one", "subject-candidate", digest)
	if err != nil || a.Amount != "15" || a.Currency != "USD" || a.Author != SampleMember || a.Stage != approvals.StageCharge || len(a.Entries) != 0 || len(a.Digests) != 5 || a.Digests["result"] != digest {
		t.Fatal(a, err)
	}
	if _, err := f.s.approvals.RequestOf(SampleWorkspace, "financial-one"); !errors.Is(err, approvals.ErrNotFound) {
		t.Fatal("builder created request", err)
	}
	if len(f.s.chargeProducer.candidates.records) != count || len(f.s.admin.Grants()) != grants || f.s.Revision() != revision {
		t.Fatal("builder mutated stores")
	}
	a.Digests["input"] = "changed"
	b, err := readSubject(f, "financial-one", "subject-candidate", digest)
	if err != nil || b.Digests["input"] == "changed" {
		t.Fatal("aliased digest", b, err)
	}
	got := policyMutate(f, maker, "capture", "replacement-candidate", fields)
	f.ok(t, "replacement capture", got, "captured", "")
	c, err := readSubject(f, "financial-two", "replacement-candidate", field(got, "digest"))
	if err != nil || c.Target != b.Target || c.Digests["result"] == b.Digests["result"] || c.Author != b.Author {
		t.Fatal("unstable economic target", c, err)
	}
	after := f.s.chargeProducer.candidates.records[[2]string{SampleWorkspace, "subject-candidate"}]
	if !bytes.Equal(before.envelope, after.envelope) {
		t.Fatal("history rewritten")
	}
	first, err := f.s.approvals.Request(SampleWorkspace, b, approvals.Actor{ID: SampleAdmin, Kind: approvals.KindHuman})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.s.approvals.Request(SampleWorkspace, c, approvals.Actor{ID: SampleAdmin2, Kind: approvals.KindHuman})
	if err != nil {
		t.Fatal(err)
	}
	first, err = f.s.approvals.RequestOf(SampleWorkspace, first.Subject.ID)
	if err != nil || first.State != approvals.StateSuperseded || first.Subject.Author != SampleMember {
		t.Fatal("same-target replacement", first, err)
	}
}

func TestFinancialSubjectRefusesCorruptOrUnauthoredCandidates(t *testing.T) {
	for _, mode := range []string{"v5", "v6", "wrong-digest", "author", "result", "input"} {
		t.Run(mode, func(t *testing.T) {
			f, _, _, _, digest := subjectWorld(t)
			store := &f.s.chargeProducer.candidates
			key := [2]string{SampleWorkspace, "subject-candidate"}
			record := store.records[key]
			var envelope chargeEnvelope
			if err := json.Unmarshal(record.envelope, &envelope); err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "v5":
				envelope.Schema = "hb_charge_candidate_v5"
				envelope.Input.Normalized.CaptureAuthor = ""
				envelope.Input.Normalized.InputEligibility = nil
			case "v6":
				envelope.Schema = "hb_charge_candidate_v6"
				envelope.Input.Normalized.CaptureAuthor = ""
			case "author":
				envelope.Input.Normalized.CaptureAuthor = "invalid author"
			case "result":
				envelope.Result.Total = "16"
			case "input":
				envelope.Input.Lines[0].Cost = "16"
			}
			record.envelope, _ = json.Marshal(envelope)
			if mode != "wrong-digest" {
				record.digest, _ = normalizedDigest(envelope)
				digest = record.digest
			} else {
				digest = "wrong"
			}
			store.records[key] = record
			count := len(store.records)
			got, err := readSubject(f, "refused", "subject-candidate", digest)
			if err == nil || !reflect.DeepEqual(got, approvals.Subject{}) {
				t.Fatal("corrupt subject accepted", got, err)
			}
			if len(store.records) != count || !reflect.DeepEqual(store.records[key], record) {
				t.Fatal("refusal changed stored evidence")
			}
			if _, err := f.s.approvals.RequestOf(SampleWorkspace, "refused"); !errors.Is(err, approvals.ErrNotFound) {
				t.Fatal("refusal created request", err)
			}
		})
	}
}

func TestFinancialSubjectRequiresOwnedPolicyAndRealRequiredProducers(t *testing.T) {
	f, _, _, _, digest := subjectWorld(t)
	for _, required := range []string{"fx,result", "erp_draft,result"} {
		current, err := f.s.approvals.PolicyOf(SampleWorkspace, approvals.StageCharge)
		if err != nil {
			t.Fatal(err)
		}
		revision := "3"
		if current.Revision == 3 {
			revision = "4"
		}
		subjectPolicy(t, f, "policy-"+revision, revision, required)
		if _, err := readSubject(f, "missing-evidence", "subject-candidate", digest); err == nil {
			t.Fatal("missing producer accepted", required)
		}
	}
	subjectPolicy(t, f, "policy-five", "5", "input,result")
	if _, err := readSubject(f, "valid", "subject-candidate", digest); err != nil {
		t.Fatal(err)
	}
	binding := f.s.financialPolicies["policy-five"]
	delete(f.s.financialPolicies, "policy-five")
	if _, err := readSubject(f, "unowned", "subject-candidate", digest); err == nil {
		t.Fatal("unbound current policy adopted")
	}
	binding.Digest = "corrupt"
	f.s.financialPolicies["policy-five"] = binding
	if _, err := readSubject(f, "corrupt-policy", "subject-candidate", digest); err == nil {
		t.Fatal("corrupt policy ownership accepted")
	}
}
