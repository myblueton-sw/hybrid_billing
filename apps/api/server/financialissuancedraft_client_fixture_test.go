package server

import (
	"encoding/json"
	"os"
	"testing"
)

func TestFinancialDraftClientFixture(t *testing.T) {
	path := os.Getenv("HB_DRAFT_CLIENT_FIXTURE")
	if path == "" {
		t.Skip("explicit synthetic client fixture output required")
	}
	f, member, reviewer, spec, binding := configuredDraftWorld(t)
	output := map[string]json.RawMessage{}
	retain := func(name string, result reply) {
		t.Helper()
		if result.status != 200 {
			t.Fatalf("%s status %d: %s", name, result.status, result.raw)
		}
		output[name] = json.RawMessage(result.raw)
	}
	request, digest := spec.Application.FinalRequest, spec.Application.FinalDigest
	retain("createContext", draftContextRead(f, member, request, digest))
	retain("submit", draftRun(f, member, "issuance_draft.submit", spec.ID, draftSubmitArguments(spec)))
	retain("list", documentRead(f, reviewer, "issuance_draft.list", SampleWorkspace, map[string]string{}))
	retain("submitted", documentRead(f, reviewer, "issuance_draft.inspect", spec.ID, map[string]string{}))
	retain("prepareContext", draftContextRead(f, reviewer, request, digest))
	draft := f.s.financialIssuance.drafts[spec.ID]
	retain("prepare", draftRun(f, reviewer, "issuance_draft.prepare_approval", "client-fixture-review", draftPrepareArguments(draft, binding)))
	retain("prepared", documentRead(f, reviewer, "issuance_draft.inspect", spec.ID, map[string]string{}))
	retain("approveContext", draftContextRead(f, reviewer, request, digest))
	review := f.s.financialIssuance.draftReviews["client-fixture-review"]
	retain("approve", draftRun(f, reviewer, "issuance_draft.approve", review.ID, map[string]string{"digest": review.Digest}))
	retain("approved", documentRead(f, reviewer, "issuance_draft.inspect", spec.ID, map[string]string{}))
	meta, err := json.Marshal(map[string]string{"workspace": SampleWorkspace, "submitter": draft.Submitter, "reviewer": review.Actor})
	f.must(err)
	output["meta"] = meta
	raw, err := json.Marshal(output)
	f.must(err)
	f.must(os.WriteFile(path, raw, 0600))
}
