package server

import (
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
)

func TestIssuanceSetupRecordsThatCannotBeReleased(t *testing.T) {
	w := stepImpactWorld(t)
	f, a := w.f, w.f.s.financialIssuance
	change := func(id, field, kind string) financialChangeSpec {
		spec := w.change
		spec.ID, spec.Field = id, field
		spec.Channel, spec.ChannelDigest = w.channels[kind].Spec.ID, w.channels[kind].Digest
		return spec
	}

	_, err := requestChangeAs(f, w.requester, change("stuck-open", protected.FieldPaymentAccount, "entity"))
	f.must(err)
	_, err = requestChangeAs(f, w.requester, change("stuck-open-next", protected.FieldPaymentAccount, "entity"))
	documentFailure(t, err, commands.PrerequisiteFailed)

	f.s.fault = func(at string) bool { return at == "issuance-change-after-request" }
	_, err = requestChangeAs(f, w.requester, change("stuck-unknown", protected.FieldDocumentRecipient, "customer"))
	documentFailure(t, err, commands.DependencyUnavailable)
	f.s.fault = nil
	if got := a.changes["stuck-unknown"].Phase; got != "request_unknown" {
		t.Fatal("fixture phase " + got)
	}
	_, err = requestChangeAs(f, w.requester, change("stuck-unknown-next", protected.FieldDocumentRecipient, "customer"))
	documentFailure(t, err, commands.PrerequisiteFailed)

	rules := *a.protectedRules
	if rules.State != "confirmed" {
		t.Fatal("fixture rules " + rules.State)
	}
	_, err = proposeRulesAs(f, f.as(AdminLogin), "replacement-rules", issuanceRules())
	documentFailure(t, err, commands.EvidenceStale)
	if a.protectedRules.ID != rules.ID || len(a.changes) != 2 {
		t.Fatal("a refused request changed retained records")
	}
}
