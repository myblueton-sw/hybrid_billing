package server

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
)

func TestFinancialConfigurationLegacySetupEncoding(t *testing.T) {
	f := setup(t, SeedSample).ready()
	p, err := proposeConfigurationAs(f, f.as(AdminLogin), configurationSpec())
	f.must(err)
	p.Digest, p.State, p.Confirmer = "", "", ""
	p.Proposed, p.Active, p.ActivatedAt = issuance.KindView{}, issuance.KindView{}, time.Time{}
	legacy := struct {
		Schema, Workspace                          string
		Spec                                       financialConfigurationSpec
		Scope                                      financialConfigurationScope
		Kind                                       issuance.Kind
		Proposer, Reason, Digest, State, Confirmer string
		Proposed, Active                           issuance.KindView
		ActivatedAt                                time.Time
	}{p.Schema, p.Workspace, p.Spec, p.Scope, p.Kind, p.Proposer, p.Reason, p.Digest, p.State, p.Confirmer, p.Proposed, p.Active, p.ActivatedAt}
	oldJSON, err := json.Marshal(legacy)
	f.must(err)
	currentJSON, err := json.Marshal(p)
	f.must(err)
	if !bytes.Equal(oldJSON, currentJSON) || bytes.Contains(currentJSON, []byte(`"setup"`)) {
		t.Fatal("nil setup changed legacy serialized evidence")
	}
	oldDigest, err := financialEvidenceDigest("issuance_configuration_v1", legacy)
	f.must(err)
	currentDigest, err := configurationDigest(p)
	f.must(err)
	if oldDigest != currentDigest {
		t.Fatal("legacy digest changed")
	}
	for _, setup := range []*financialConfigurationSetupContext{{}, {On: "2026-11-16"}, {ScopeDigest: oldDigest}, {On: "2026-02-30", ScopeDigest: oldDigest}} {
		if validConfigurationSetup(setup) {
			t.Fatal("partial or malformed setup accepted")
		}
	}
	p.Setup = &financialConfigurationSetupContext{On: "2026-11-16", ScopeDigest: oldDigest}
	withSetup, err := configurationDigest(p)
	f.must(err)
	if withSetup == oldDigest {
		t.Fatal("setup not hash bound")
	}
	clone := copyConfiguration(p)
	clone.Setup.ScopeDigest = "changed"
	if p.Setup.ScopeDigest != oldDigest {
		t.Fatal("setup context aliases retained record")
	}
	effect := &financialConfigurationEffect{}
	f.must(inputAuthorityAs(f, f.as(AdminLogin), func() error {
		_, err = f.s.financialIssuance.proposeConfigurationTracked(SampleWorkspace, SampleAdmin, "Invalid context", configurationSpec(), &financialConfigurationSetupContext{}, effect, nil)
		documentFailure(t, err, commands.InvalidRequest)
		if effect.changed {
			t.Fatal("invalid context changed state")
		}
		return nil
	}))
}
