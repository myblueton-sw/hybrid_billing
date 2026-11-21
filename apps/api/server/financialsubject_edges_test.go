package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/inputreview"
	"github.com/myblueton-sw/hybrid_billing/modules/pricingpolicy"
	"github.com/myblueton-sw/hybrid_billing/modules/reconcile"
	"reflect"
	"testing"
)

func TestFinancialSubjectRefusesNewMappingAndInputPublication(t *testing.T) {
	for _, mode := range []string{"mapping", "publication"} {
		t.Run(mode, func(t *testing.T) {
			f, maker, _, _, digest := subjectWorld(t)
			p := f.s.chargeProducer
			before := p.candidates.records[[2]string{SampleWorkspace, "subject-candidate"}]
			if mode == "mapping" {
				run, err := p.normal.Run(SampleWorkspace, "sample-normalized-charge")
				f.must(err)
				mapping, err := p.normal.Revision(run.Spec.Mapping, run.Spec.Revision)
				f.must(err)
				for i := range mapping.Entries {
					if mapping.Entries[i].Field == "class" {
						mapping.Entries[i].Source = "service"
					}
				}
				revision, err := p.normal.DeclareMapping(run.Spec.Mapping, run.Spec.Revision, mapping.Entries)
				if err != nil || revision != 2 {
					t.Fatal(revision, err)
				}
			} else {
				err := inputAuthorityAs(f, maker, func() error {
					r := f.s.inputReviews.registry
					record, err := r.Compare(SampleWorkspace, sampleInputReview, "new-publication")
					if err != nil {
						return err
					}
					_, err = r.Confirm(inputreview.Intent{Workspace: SampleWorkspace, ID: record.ID, Actor: SampleMember, Digest: record.Evidence.Digest, ExpectedScope: 1, Rationale: "Synthetic replacement publication"})
					return err
				})
				f.must(err)
			}
			if _, err := readSubject(f, "stale", "subject-candidate", digest); err == nil {
				t.Fatal("new evidence ignored", mode)
			}
			if !reflect.DeepEqual(before, p.candidates.records[[2]string{SampleWorkspace, "subject-candidate"}]) {
				t.Fatal("historical candidate changed")
			}
		})
	}
}

func TestFinancialSubjectTargetScopeAndVersionBoundary(t *testing.T) {
	f, _, _, _, _ := subjectWorld(t)
	v, err := f.s.chargeProducer.candidates.read(SampleWorkspace, "subject-candidate")
	if err != nil {
		t.Fatal(err)
	}
	base, err := financialSubjectTarget(v.Envelope.Input)
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*chargeInput){
		func(i *chargeInput) { i.ID = "other-candidate" },
		func(i *chargeInput) { i.Policy.Currency = "EUR" },
		func(i *chargeInput) {
			i.Normalized.Binding.Version = "2"
			i.Normalized.Binding.ID = "replacement-binding"
			i.Normalized.Run = "replacement-run"
			i.Normalized.Manifest++
		},
		func(i *chargeInput) {
			i.Normalized.MappingRevision++
			i.Normalized.Binding.ContractRevision++
			i.Normalized.Binding.BillingRevision++
		},
	} {
		i := v.Envelope.Input
		change(&i)
		got, err := financialSubjectTarget(i)
		if err != nil || got != base {
			t.Fatal("version bypasses supersession", got, err)
		}
	}
	for _, change := range []func(*chargeInput){
		func(i *chargeInput) { i.Workspace = "other" }, func(i *chargeInput) { i.Customer = "other" }, func(i *chargeInput) { i.Entity = "other" },
		func(i *chargeInput) { i.Normalized.Binding.Contract = "other" }, func(i *chargeInput) { i.Normalized.Binding.BillingProfile = "other" },
		func(i *chargeInput) { i.Normalized.Binding.Provider = "other" }, func(i *chargeInput) { i.Normalized.Binding.Scope = "other" },
		func(i *chargeInput) { i.Normalized.Binding.Period = "other" }, func(i *chargeInput) { i.From = "2026-10-02" }, func(i *chargeInput) { i.To = "2026-11-01" },
	} {
		i := v.Envelope.Input
		change(&i)
		got, err := financialSubjectTarget(i)
		if err != nil || got == base {
			t.Fatal("different economic scope collided", got, err)
		}
	}
}

func TestFinancialSubjectRechecksCurrentProducers(t *testing.T) {
	for _, cause := range []string{"policy-revoked", "target-revision", "document-revision", "voided-basis", "source-bytes", "foreign-review-producer"} {
		t.Run(cause, func(t *testing.T) {
			f, _, checker, fields, digest := subjectWorld(t)

			before := f.s.chargeProducer.candidates.records[[2]string{SampleWorkspace, "subject-candidate"}]
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
			if _, err := readSubject(f, "stale-subject", "subject-candidate", digest); err == nil {
				t.Fatal("stale producer accepted", cause)
			}

			after := f.s.chargeProducer.candidates.records[[2]string{SampleWorkspace, "subject-candidate"}]
			if !reflect.DeepEqual(before, after) {
				t.Fatal("historical bytes changed")
			}
		})
	}
}
