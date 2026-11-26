package server

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func TestFinancialERPResultPublicHistoryCursorAndHold(t *testing.T) {
	w, p, digest := publicResult(t)
	f := w.f
	a := w.actors[Admin2Login]
	for i := 0; i < 11; i++ {
		packet := cloneDocumentValue(t, p)
		packet.External = fmt.Sprintf("history-external-%02d", i)
		raw, err := json.Marshal(packet)
		f.must(err)
		f.ok(t, "history proposal", f.run(a, "result_evidence.propose", fmt.Sprintf("history-result-%02d", i), map[string]string{"packet": string(raw)}), "pending", "")
	}
	fields := map[string]string{"digest": p.PreparationDigest}
	first := documentRead(f, a, "result_evidence.history", p.Preparation, fields)
	if first.status != 200 || field(first, "row_09") == "" || field(first, "row_10") != "" || field(first, "next") == "" {
		t.Fatal(first.raw)
	}
	fields["after"], fields["snapshot"] = field(first, "next"), field(first, "snapshot")
	second := documentRead(f, a, "result_evidence.history", p.Preparation, fields)
	if second.status != 200 || field(second, "row_01") == "" || field(second, "row_02") != "" || field(second, "next") != "" {
		t.Fatal(second.raw)
	}
	f.ok(t, "revoke changes revision", f.run(w.actors[Admin3Login], "result_evidence.revoke", "public-result", map[string]string{"digest": digest}), "revoked", "")
	f.code(t, "stale cursor", documentRead(f, a, "result_evidence.history", p.Preparation, fields), 409, string(commands.EvidenceStale))
	f.s.enterHold("result-history-test")
	held := documentRead(f, a, "result_evidence.history", p.Preparation, map[string]string{"digest": p.PreparationDigest})
	if held.status != 200 {
		t.Fatal(held.raw)
	}
	if got := documentRead(f, a, "result_evidence.inspect", "public-result", map[string]string{"digest": digest}); got.status != 200 || field(got, "recorded_state") != "revoked" {
		t.Fatal(got.raw)
	}
}

func TestFinancialERPResultPublicMutationEnvelope(t *testing.T) {
	w, _, digest := publicResult(t)
	f := w.f
	a := w.actors[Admin2Login]
	fields := map[string]string{"digest": digest}
	body := chargeBody(f, "result-envelope", "public-result", fields)
	if got := f.call("erp_result.apply", body, a.cookie, ""); got.status != 403 {
		t.Fatal(got.raw)
	}
	f.code(t, "stale revision", f.runWith(a, "r-0", "erp_result.apply", "public-result", fields), 409, string(commands.RevisionConflict))
	f.advance(16 * time.Minute)
	f.code(t, "stepup", f.run(a, "erp_result.apply", "public-result", fields), 403, CodeReauthentication)
	a = f.as(Admin2Login)
	f.s.enterHold("result-envelope")
	f.code(t, "hold", f.run(a, "erp_result.apply", "public-result", fields), 503, string(commands.DependencyUnavailable))
	if len(w.a.resultApplications) != 0 {
		t.Fatal("envelope refusal applied result")
	}
}
