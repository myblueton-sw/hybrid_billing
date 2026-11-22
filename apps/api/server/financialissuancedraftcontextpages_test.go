package server

import (
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func TestFinancialDraftPublicContextCursors(t *testing.T) {
	f, member, reviewer, spec, binding := configuredDraftWorld(t)
	request, digest := spec.Application.FinalRequest, spec.Application.FinalDigest
	first := draftContextRead(f, member, request, digest)
	assertDraftContext(t, first, map[string]string{"binding_count": "1"})
	args := map[string]string{"digest": digest, "after_binding": binding.Spec.ID, "snapshot": field(first, "snapshot")}
	last := documentRead(f, member, "issuance_draft.context", request, args)
	assertDraftContext(t, last, map[string]string{"binding_count": "1", "binding_00": "", "next_binding": "", "snapshot": field(first, "snapshot")})
	for _, test := range []struct {
		after, snapshot string
		status          int
		code            commands.Code
	}{
		{binding.Spec.ID, "", 422, commands.InvalidRequest}, {"", field(first, "snapshot"), 422, commands.InvalidRequest},
		{"not-a-choice", field(first, "snapshot"), 409, commands.EvidenceStale}, {binding.Spec.ID, strings.Repeat("0", 64), 409, commands.EvidenceStale},
	} {
		read := documentRead(f, member, "issuance_draft.context", request, map[string]string{"digest": digest, "after_binding": test.after, "snapshot": test.snapshot})
		f.code(t, "context cursor refuses", read, test.status, string(test.code))
		if read.body["fields"] != nil {
			t.Fatal("cursor refusal disclosed fields")
		}
	}
	other := documentRead(f, reviewer, "issuance_draft.context", request, args)
	f.code(t, "context cursor is caller scoped", other, 409, string(commands.EvidenceStale))
	_, err := submitDraftAs(f, member, spec)
	f.must(err)
	changed := documentRead(f, member, "issuance_draft.context", request, args)
	f.code(t, "new owned draft invalidates prior context", changed, 409, string(commands.EvidenceStale))
	f.s.financialIssuance.documents = nil
	unavailable := draftContextRead(f, member, request, digest)
	assertDraftContext(t, unavailable, map[string]string{"readiness": "unavailable", "submit_action": "evidence_unavailable", "existing_draft": spec.ID})
}

func TestFinancialDraftContextPageProjectionComplete(t *testing.T) {
	f := setup(t, SeedSample).ready()
	p := financialDraftContext{Evidence: financialDraftContextEvidence{Fields: map[string]string{"mode": "synthetic", "current_eligibility": "eligible"}}}
	for i := 0; i < 11; i++ {
		p.Choices = append(p.Choices, map[string]string{"binding": fmt.Sprintf("projection-%02d", i), "binding_digest": strings.Repeat("a", 64), "term_revision": "1", "net_days": "30", "calendar_mode": "calendar", "zone": "UTC", "from": "2026-01-01", "to": "2026-12-31"})
	}
	original := maps.Clone(p.Evidence.Fields)
	first, err := f.s.financialIssuance.draftContextPage(SampleWorkspace, SampleMember, p, "", "")
	f.must(err)
	second, err := f.s.financialIssuance.draftContextPage(SampleWorkspace, SampleMember, p, first["next_binding"], first["snapshot"])
	f.must(err)
	if first["next_binding"] != "projection-09" || second["next_binding"] != "" || first["binding_count"] != "11" || second["binding_count"] != "11" || !maps.Equal(original, p.Evidence.Fields) {
		t.Fatal("projection lost page identity or mutated context")
	}
	seen := map[string]bool{}
	for _, page := range []map[string]string{first, second} {
		for i := 0; i < 10; i++ {
			raw, ok := page[fmt.Sprintf("binding_%02d", i)]
			if !ok {
				continue
			}
			var row map[string]string
			f.must(json.Unmarshal([]byte(raw), &row))
			if seen[row["binding"]] {
				t.Fatal("duplicate page choice")
			}
			seen[row["binding"]] = true
		}
	}
	if len(seen) != 11 {
		t.Fatal("truncated choices")
	}
}
