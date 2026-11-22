package server

import (
	"encoding/json"
	"errors"
	"github.com/myblueton-sw/hybrid_billing/modules/inputreview"
	"github.com/myblueton-sw/hybrid_billing/modules/intake"
	"github.com/myblueton-sw/hybrid_billing/modules/normalize"
	"github.com/myblueton-sw/hybrid_billing/modules/pricingpolicy"
	"github.com/myblueton-sw/hybrid_billing/modules/rating"
	"reflect"
	"strconv"
	"testing"
	"time"
)

func captureInvoiceInWorld(t *testing.T, f world, maker, checker actor, prefix, period string, from, to time.Time) (string, string) {
	t.Helper()
	p := f.s.chargeProducer
	contract, err := f.s.orgs.ContractRevision(SampleWorkspace, "sample-contract-01", 2)
	f.must(err)
	key := normalize.MappingKey{Provider: contract.ProviderRef, SourceVersion: "synthetic-source-v1", Target: p.target.ID}
	run := prefix + "-run"
	rows := []normalize.Row{}
	for i, amount := range []string{"10.25", "4.75"} {
		rows = append(rows, normalize.Row{Key: normalize.RowKey{Kind: normalize.EventKeys, Key: prefix + "-line-" + strconv.Itoa(i)}, Cells: map[string]normalize.Cell{
			"billed": {Text: amount}, "currency": {Text: "USD"}, "start": {Text: from.Format(time.RFC3339)}, "end": {Text: to.Format(time.RFC3339)}, "class": {Text: "compute"}, "service": {Text: "virtual-machine"},
		}})
	}
	scope := inputreview.Scope{Workspace: SampleWorkspace, Provider: contract.ProviderRef, Account: "sample-acct-01", Period: period}
	document := candidateDocument{Mode: "synthetic", Scope: scope, ID: prefix + "-provider-document", Revision: 1, Currency: "USD", Field: "billed", Basis: normalize.Billed, Total: "15.0000"}
	body, err := json.Marshal(document)
	f.must(err)
	var review inputreview.Record
	func() {
		f.s.domain.Lock()
		defer f.s.domain.Unlock()
		_, err = p.assembleSourceLocked(sourcePacket{Mode: "synthetic", CostField: "billed", CurrencyField: "currency", Run: normalize.RunSpec{WorkspaceID: SampleWorkspace, ID: run, Mapping: key, Revision: 1, Input: normalize.Input{Scope: scope.Account, Period: period}, Window: &normalize.Window{From: from, To: to}, Expected: map[string]map[string]string{"billed": {"USD": "15"}}}, Artifacts: []sourceArtifact{{ID: prefix + "-source", Document: prefix + "-source-document", Revision: 1, Kind: intake.EventKeys, Sums: map[string]string{"USD": "15"}, Rows: rows}}})
		f.must(err)
		binding := normalizedBinding{SourceMode: "synthetic", ID: prefix + "-binding", Version: "1", Provider: key.Provider, SourceVersion: key.SourceVersion, Scope: scope.Account, Period: period, MappingRevision: 1, Contract: contract.ContractID, ContractRevision: contract.Revision, BillingProfile: "sample-profile-north", BillingRevision: 1, From: from.Format("2006-01-02"), To: to.AddDate(0, 0, -1).Format("2006-01-02"), Class: "class", Service: "service", Classes: []normalizedClass{{Source: "compute", Class: rating.CSPUsage}}}
		billing, _, err := f.s.orgs.BillingProfile(SampleWorkspace, binding.BillingProfile)
		f.must(err)
		binding.Policy.Currency, binding.Policy.Path = billing.Currency, rating.Path(contract.Path)
		f.must(p.retainCaptureBindingLocked(SampleWorkspace, run, binding))
		review, err = f.s.inputReviews.prepareLocked(SampleWorkspace, run, prefix+"-review", body, nil)
		f.must(err)
	}()
	publication, err := f.s.inputReviews.registry.Current(scope)
	if !errors.Is(err, inputreview.ErrNotFound) {
		t.Fatal("new invoice period already published", publication, err)
	}
	confirmed := f.runWith(maker, f.s.Revision(), "input.confirm_publish", review.ID, map[string]string{"digest": review.Evidence.Digest, "scope_revision": "0"})
	f.ok(t, "new invoice input publication", confirmed, inputreview.Eligible, "")
	publication, err = f.s.inputReviews.registry.Current(scope)
	f.must(err)
	spec := policyHTTPSpec()
	spec.From, spec.To, spec.Policy.ID = from, to, prefix+"-pricing"
	specText := policySpecText(t, spec)
	policyID := prefix + "-policy"
	prepared := policyRead(f, maker, "prepare", policyID, map[string]string{"spec": specText})
	proposal := policyMutate(f, maker, "propose", policyID, policyProposalFields(prepared, specText))
	f.ok(t, "new invoice pricing proposal", proposal, pricingpolicy.Pending, "")
	approved := policyMutate(f, checker, "approve", policyID, map[string]string{"digest": field(proposal, "digest")})
	f.ok(t, "new invoice pricing approval", approved, pricingpolicy.Approved, "")
	candidateID := prefix + "-candidate"
	captured := policyMutate(f, maker, "capture", candidateID, map[string]string{"binding": prefix + "-binding", "input_review": review.ID, "input_digest": review.Evidence.Digest, "scope_revision": strconv.FormatUint(publication.Revision, 10), "policy": policyID, "policy_digest": field(proposal, "digest")})
	f.ok(t, "new invoice capture", captured, "captured", "")
	return candidateID, field(captured, "digest")
}

func TestFinancialSecondInvoiceCapturePreservesExistingWorld(t *testing.T) {
	f, _, _, draft, _ := draftApprovalWorld(t)
	producer, adapter, ledger := f.s.chargeProducer, f.s.documentERP, f.s.documentERP.ledger
	before, err := producer.candidates.financialCandidate(SampleWorkspace, "document-candidate")
	f.must(err)
	id, digest := captureInvoiceInWorld(t, f, f.as(MemberLogin), f.as(Admin3Login), "invoice-b", "2026-11", time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC))
	next, err := producer.candidates.financialCandidate(SampleWorkspace, id)
	f.must(err)
	after, err := producer.candidates.financialCandidate(SampleWorkspace, "document-candidate")
	f.must(err)
	origin := next.Envelope.Input.Normalized
	if next.Digest != digest || origin.Run != "invoice-b-run" || origin.InputEligibility == nil || origin.InputEligibility.Review.ID != "invoice-b-review" || origin.InputEligibility.Publication.Revision != 1 || origin.Pricing == nil || origin.Pricing.Proposal.ID != "invoice-b-policy" || origin.Contract.Revision != 2 || origin.Binding.BillingRevision != 1 || next.Envelope.Input.From != "2026-11-01" || next.Envelope.Input.To != "2026-11-30" {
		t.Fatal("second invoice provenance differs from actual new capture")
	}
	if digest == "" || next.Envelope.Input.Normalized.Binding.Period != "2026-11" || next.Envelope.Result.Total != "15" || !reflect.DeepEqual(before, after) || f.s.chargeProducer != producer || f.s.documentERP != adapter || f.s.documentERP.ledger != ledger || !reflect.DeepEqual(f.s.financialIssuance.drafts[draft.Spec.ID], draft) {
		t.Fatal("second invoice capture replaced old world or source")
	}
}
