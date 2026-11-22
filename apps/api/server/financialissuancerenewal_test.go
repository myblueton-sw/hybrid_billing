package server

import (
	"fmt"
	"github.com/myblueton-sw/hybrid_billing/modules/financialpermission"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"testing"
	"time"
)

func renewInvoiceAuthorities(t *testing.T, f world) {
	t.Helper()
	authorities := []*serverGrantAuthority{f.s.pricingAuthority, f.s.inputAuthority, f.s.financialRequestPermission, f.s.financialPermission, f.s.documentAuthority, f.s.resultAuthority, f.s.erpFinalRequestPermission, f.s.erpFinalPermission, f.s.protectedAuthority, f.s.termAuthority, f.s.draftAuthority}
	for domain, a := range authorities {
		records, err := a.registry.Records(SampleWorkspace)
		f.must(err)
		ceilings := map[string]string{}
		for _, kind := range []string{inputauthority.Console, inputauthority.Human, financialpermission.DirectHuman} {
			for index, old := range records {
				if old.State != inputauthority.Applied || old.Proposal.Spec.Kind != kind {
					continue
				}
				spec := old.Proposal.Spec
				spec.NotBefore, spec.NotAfter = f.s.clock(), f.s.clock().Add(12*time.Hour)
				if spec.CeilingID != "" {
					replacement, ok := ceilings[spec.CeilingID]
					if !ok {
						t.Fatal("missing actual renewed ceiling")
					}
					spec.CeilingID = replacement
				}
				logins := []string{}
				for _, entry := range []struct{ id, login string }{{SampleAdmin, AdminLogin}, {SampleAdmin2, Admin2Login}, {SampleAdmin3, Admin3Login}} {
					if entry.id != spec.Recipient {
						logins = append(logins, entry.login)
					}
				}
				first, second := f.as(logins[0]), f.as(logins[1])
				var proposed, approved inputauthority.Record
				f.must(inputAuthorityAs(f, first, func() error {
					var err error
					proposed, err = a.proposeLocked(inputauthority.Proposal{ID: fmt.Sprintf("invoice-renew-%d-%d", domain, index), Spec: spec, Proposer: f.s.answer.User(), Rationale: "Renew existing invoice workflow standing after expiry"})
					return err
				}))
				f.must(inputAuthorityAs(f, second, func() error {
					var err error
					approved, err = a.approveLocked(inputauthority.Intent{Workspace: SampleWorkspace, ID: proposed.Proposal.ID, Digest: proposed.Digest, Actor: f.s.answer.User(), Rationale: "Independent renewal of existing capability"})
					return err
				}))
				if approved.State != inputauthority.Applied || approved.Decision.GrantID == old.Decision.GrantID {
					t.Fatal("renewal lacks new actual grant")
				}
				if kind == inputauthority.Console {
					ceilings[old.Decision.GrantID] = approved.Decision.GrantID
				}
				f.s.endProbe(first.verifier(f))
				f.s.endProbe(second.verifier(f))
			}
		}
	}
}
