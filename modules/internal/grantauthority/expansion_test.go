package grantauthority

import (
	"errors"
	"testing"
)

func TestInspectionExpansionPreservesExistingIssuanceGrantIdentities(t *testing.T) {
	for _, domain := range []Domain{
		{Version: "synthetic-issuance-term-authority-v1", GrantPrefix: "issuance-term-grant-", Capabilities: []string{"record", "approve", "bind", "approve_binding"}},
		{Version: "synthetic-issuance-protected-authority-v1", GrantPrefix: "issuance-protected-grant-", Capabilities: []string{"register_channel", "admit_channel", "request_change", "verify_change", "approve_change"}},
	} {
		for _, capability := range domain.Capabilities {
			t.Run(domain.Version+"/"+capability, func(t *testing.T) {
				old, err := domainRegistry(domain, &domainPort{effects: map[string]Effect{}})
				if err != nil {
					t.Fatal(err)
				}
				expanded := domain
				expanded.Capabilities = append(append([]string{}, domain.Capabilities...), "inspect")
				current, err := domainRegistry(expanded, &domainPort{effects: map[string]Effect{}})
				if err != nil {
					t.Fatal(err)
				}
				p := domainProposal()
				p.Spec.Capability = capability
				before, err := old.Propose(p)
				if err != nil {
					t.Fatal(err)
				}
				after, err := current.Propose(p)
				if err != nil {
					t.Fatal(err)
				}
				if before.Digest != after.Digest {
					t.Fatal("existing proposal identity changed")
				}
				intent := Intent{Workspace: p.Spec.Workspace, ID: p.ID, Actor: "bob", Digest: before.Digest, Rationale: "Independent existing grant review"}
				first, err := old.Approve(intent)
				if err != nil {
					t.Fatal(err)
				}
				second, err := current.Approve(intent)
				if err != nil {
					t.Fatal(err)
				}
				if first.Decision.GrantID != second.Decision.GrantID {
					t.Fatal("existing effect identity changed")
				}
				p.ID, p.Spec.Capability = "inspection", "inspect"
				if _, err := old.Propose(p); !errors.Is(err, ErrInvalid) {
					t.Fatal("legacy capability boundary changed", err)
				}
				if _, err := current.Propose(p); err != nil {
					t.Fatal("new inspection unavailable", err)
				}
			})
		}
	}
}
