package server

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
)

func TestFinancialConfigurationForeignAndUnavailable(t *testing.T) {
	for _, mode := range []string{"foreign", "lookup", "revision", "mismatch"} {
		t.Run(mode, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			a := f.s.financialIssuance
			spec := configurationSpec()
			who := f.as(AdminLogin)
			kind, err := configurationKind(spec)
			f.must(err)
			code := commands.EvidenceStale
			switch mode {
			case "foreign":
				_, err = a.documents.Define(SampleWorkspace, kind, issuance.Actor{ID: SampleAdmin, Kind: issuance.KindHuman})
				f.must(err)
			case "lookup":
				f.s.fault = func(at string) bool { return at == "issuance-configuration-lookup" }
				code = commands.DependencyUnavailable
			case "revision":
				spec.Revision = 2
				code = commands.InvalidRequest
			case "mismatch":
				spec.BillingProfile = "sample-profile-south"
			}
			_, err = proposeConfigurationAs(f, who, spec)
			documentFailure(t, err, code)
			if len(a.configurations) != 0 || len(a.configurationScopes) != 0 {
				t.Fatal("refusal reserved configuration")
			}
		})
	}
}

func TestFinancialConfigurationMissingAndUnobservedModule(t *testing.T) {
	for _, mode := range []string{"missing-proposed", "missing-active", "foreign-proposed", "unobserved-active", "unavailable-recovery"} {
		t.Run(mode, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			a := f.s.financialIssuance
			spec := configurationSpec()
			proposer, confirmer := f.as(AdminLogin), f.as(Admin2Login)
			point := "issuance-configuration-propose-after-module"
			if mode == "foreign-proposed" {
				point = "issuance-configuration-propose-after-intent"
			}
			f.s.fault = func(at string) bool { return at == point }
			_, err := proposeConfigurationAs(f, proposer, spec)
			documentFailure(t, err, commands.DependencyUnavailable)
			f.s.fault = nil
			p := a.configurations[spec.ID]
			if mode == "missing-active" || mode == "unobserved-active" {
				p, err = proposeConfigurationAs(f, proposer, spec)
				f.must(err)
				point = "issuance-configuration-confirm-after-module"
				if mode == "unobserved-active" {
					point = "issuance-configuration-confirm-after-intent"
				}
				f.s.fault = func(at string) bool { return at == point }
				_, err = confirmConfigurationAs(f, confirmer, p)
				documentFailure(t, err, commands.DependencyUnavailable)
				f.s.fault = nil
			}
			before := copyConfiguration(a.configurations[spec.ID])
			switch mode {
			case "missing-proposed", "missing-active":
				empty, err := newFinancialIssuance(f.s)
				f.must(err)
				a.documents = empty.documents
			case "foreign-proposed":
				_, err = a.documents.Define(SampleWorkspace, p.Kind, issuance.Actor{ID: SampleAdmin, Kind: issuance.KindHuman})
				f.must(err)
			case "unobserved-active":
				_, err = a.documents.ApproveKind(SampleWorkspace, p.Kind.ID, issuance.Actor{ID: SampleAdmin2, Kind: issuance.KindHuman})
				f.must(err)
			case "unavailable-recovery":
				f.s.fault = func(at string) bool { return at == "issuance-configuration-lookup" }
			}
			if mode == "missing-active" || mode == "unobserved-active" {
				_, err = confirmConfigurationAs(f, confirmer, p)
			} else {
				_, err = proposeConfigurationAs(f, proposer, spec)
			}
			switch mode {
			case "missing-proposed", "missing-active":
				if !errors.Is(err, issuance.ErrNotFound) {
					t.Fatalf("missing module: %v", err)
				}
			case "unavailable-recovery":
				documentFailure(t, err, commands.DependencyUnavailable)
			default:
				documentFailure(t, err, commands.EvidenceStale)
			}
			if !reflect.DeepEqual(a.configurations[spec.ID], before) || len(a.configurationScopes) != 1 {
				t.Fatal("refusal changed ownership")
			}
			if mode == "missing-proposed" || mode == "missing-active" {
				_, err = a.documents.KindOf(SampleWorkspace, p.Kind.ID)
				if !errors.Is(err, issuance.ErrNotFound) {
					t.Fatal("missing module recreated")
				}
			}
		})
	}
}

func TestFinancialConfigurationCurrentScopeAndGovernors(t *testing.T) {
	for _, mode := range []string{"party-version", "party-suspended", "profile-version", "workspace-version", "proposer-revoked"} {
		t.Run(mode, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			a := f.s.financialIssuance
			p, err := proposeConfigurationAs(f, f.as(AdminLogin), configurationSpec())
			f.must(err)
			switch mode {
			case "party-version", "party-suspended":
				party, v, e := f.s.orgs.Party(SampleWorkspace, "sample-le-north")
				f.must(e)
				if mode == "party-suspended" {
					party.State = organizations.Suspended
				}
				_, err = f.s.orgs.PutParty(party, v)
				f.must(err)
			case "profile-version":
				b, v, e := f.s.orgs.BillingProfile(SampleWorkspace, p.Spec.BillingProfile)
				f.must(e)
				_, err = f.s.orgs.PutBillingProfile(b, v)
				f.must(err)
			case "workspace-version":
				w, v, e := f.s.orgs.Workspace(SampleWorkspace)
				f.must(e)
				_, err = f.s.orgs.PutWorkspace(w, v)
				f.must(err)
			case "proposer-revoked":
				removed := 0
				for _, g := range f.s.admin.Grants() {
					if g.SubjectID == SampleAdmin && g.Terms.Action == gate.AdministerInstallation {
						f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
						removed++
					}
				}
				if removed == 0 {
					t.Fatal("no actual governor grant revoked")
				}
			}
			_, err = confirmConfigurationAs(f, f.as(Admin2Login), p)
			if mode == "proposer-revoked" {
				if !errors.Is(err, inputauthority.ErrDenied) {
					t.Fatalf("governor refusal: %v", err)
				}
			} else {
				documentFailure(t, err, commands.EvidenceStale)
			}
			actual, err := a.documents.KindOf(SampleWorkspace, p.Kind.ID)
			f.must(err)
			if !reflect.DeepEqual(actual, p.Proposed) || !reflect.DeepEqual(a.configurations[p.Spec.ID], p) {
				t.Fatal("stale confirmation reached mutation")
			}
		})
	}
}

func TestFinancialConfigurationCorruptOwnershipAndSnapshots(t *testing.T) {
	for _, mode := range []string{"reverse-missing", "reverse-extra", "record-alias", "digest", "time", "actor", "active-before-proposal"} {
		t.Run(mode, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			a := f.s.financialIssuance
			proposer, confirmer := f.as(AdminLogin), f.as(Admin2Login)
			p, err := proposeConfigurationAs(f, proposer, configurationSpec())
			f.must(err)
			key, err := configurationScopeKey(SampleWorkspace, p.Spec)
			f.must(err)
			saved := copyConfiguration(p)
			switch mode {
			case "reverse-missing":
				delete(a.configurationScopes, key)
			case "reverse-extra":
				a.configurationScopes["foreign-scope"] = p.Spec.ID
			case "record-alias":
				a.configurations["alias"] = copyConfiguration(p)
			case "digest":
				saved.Reason = "different reason"
				a.configurations[p.Spec.ID] = saved
			case "time":
				saved.Proposed.At = saved.Proposed.At.Add(time.Second)
				a.configurations[p.Spec.ID] = saved
			case "actor":
				saved.Proposed.Definer = SampleAdmin3
				a.configurations[p.Spec.ID] = saved
			case "active-before-proposal":
				f.s.fault = func(at string) bool { return at == "issuance-configuration-confirm-after-module" }
				_, err = confirmConfigurationAs(f, confirmer, p)
				documentFailure(t, err, commands.DependencyUnavailable)
				f.s.fault = nil
				saved = a.configurations[p.Spec.ID]
				saved.Active.At = saved.Proposed.At.Add(-time.Second)
				a.configurations[p.Spec.ID] = saved
			}
			_, err = confirmConfigurationAs(f, confirmer, p)
			documentFailure(t, err, commands.EvidenceStale)
		})
	}
}

func TestFinancialConfigurationBoundedOwnership(t *testing.T) {
	for _, mode := range []string{"records", "reverse", "module"} {
		t.Run(mode, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			a := f.s.financialIssuance
			spec := configurationSpec()
			proposer := f.as(AdminLogin)
			for i := 0; i < financialIssuanceLimit; i++ {
				id := fmt.Sprintf("occupied-%d", i)
				switch mode {
				case "records":
					a.configurations[id] = financialIssuanceConfiguration{}
				case "reverse":
					a.configurationScopes[id] = id
				case "module":
					kind, err := configurationKind(spec)
					f.must(err)
					kind.ID = id
					_, err = a.documents.Define(SampleWorkspace, kind, issuance.Actor{ID: SampleAdmin, Kind: issuance.KindHuman})
					f.must(err)
				}
			}
			_, err := proposeConfigurationAs(f, proposer, spec)
			if mode != "module" {
				documentFailure(t, err, commands.PrerequisiteFailed)
				if _, ok := a.configurations[spec.ID]; ok {
					t.Fatal("capacity refusal reserved record")
				}
				return
			}
			if !errors.Is(err, issuance.ErrLimit) {
				t.Fatalf("module limit: %v", err)
			}
			saved := a.configurations[spec.ID]
			if saved.State != "proposal_unknown" || !reflect.DeepEqual(saved.Proposed, issuance.KindView{}) || len(a.configurationScopes) != 1 {
				t.Fatal("known no-write released reserved intent")
			}
			other := spec
			other.ID = "escape-capacity"
			_, err = proposeConfigurationAs(f, proposer, other)
			documentFailure(t, err, commands.PrerequisiteFailed)
		})
	}
}

func TestFinancialConfigurationRecoveryRequiresCurrentAuthority(t *testing.T) {
	for _, mode := range []string{"confirmer-revoked", "ended-session"} {
		t.Run(mode, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			a := f.s.financialIssuance
			proposer, confirmer := f.as(AdminLogin), f.as(Admin2Login)
			p, err := proposeConfigurationAs(f, proposer, configurationSpec())
			f.must(err)
			f.s.fault = func(at string) bool { return at == "issuance-configuration-confirm-after-module" }
			_, err = confirmConfigurationAs(f, confirmer, p)
			documentFailure(t, err, commands.DependencyUnavailable)
			f.s.fault = nil
			retained := copyConfiguration(a.configurations[p.Spec.ID])
			key, err := configurationScopeKey(SampleWorkspace, p.Spec)
			f.must(err)
			if mode == "confirmer-revoked" {
				removed := 0
				for _, g := range f.s.admin.Grants() {
					if g.SubjectID == SampleAdmin2 && g.Terms.Action == gate.AdministerInstallation {
						f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
						removed++
					}
				}
				if removed == 0 {
					t.Fatal("no actual confirmer grant revoked")
				}
				_, err = confirmConfigurationAs(f, confirmer, p)
			} else {
				err = inputAuthorityAs(f, confirmer, func() error {
					f.s.sessions.remove(f.s.caller)
					_, failure := a.confirmConfiguration(SampleWorkspace, p.Spec.ID, p.Digest, f.s.answer.User())
					return failure
				})
			}
			if mode == "confirmer-revoked" {
				if !errors.Is(err, inputauthority.ErrDenied) {
					t.Fatalf("governor refusal: %v", err)
				}
			} else {
				documentFailure(t, err, commands.Unauthenticated)
			}
			actual, err := a.documents.KindOf(SampleWorkspace, p.Kind.ID)
			f.must(err)
			if !reflect.DeepEqual(a.configurations[p.Spec.ID], retained) || a.configurationScopes[key] != p.Spec.ID || !reflect.DeepEqual(actual, retained.Active) {
				t.Fatal("lost authority changed pending recovery or active module")
			}
		})
	}
}
