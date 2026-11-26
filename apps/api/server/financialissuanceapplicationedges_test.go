package server

import (
	"reflect"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func TestFinancialApplicationCurrentAdmission(t *testing.T) {
	for _, mode := range []string{"configuration-scope", "configuration-digest", "reverse-owner", "governor", "inspection", "session", "final-digest"} {
		t.Run(mode, func(t *testing.T) {
			f, member, c, digest := applicationWorld(t)
			a := f.s.financialIssuance
			expected := commands.EvidenceStale
			switch mode {
			case "configuration-scope":
				spec := configurationSpec()
				spec.ID = "different-contract-revision"
				spec.ContractRevision = 1
				p, err := proposeConfigurationAs(f, f.as(AdminLogin), spec)
				f.must(err)
				c, err = confirmConfigurationAs(f, f.as(Admin2Login), p)
				f.must(err)
			case "configuration-digest":
				c.Digest = "wrong"
			case "reverse-owner":
				key, err := configurationScopeKey(SampleWorkspace, c.Spec)
				f.must(err)
				a.configurationScopes[key] = "other"
			case "governor", "inspection":
				removed := 0
				for _, g := range f.s.admin.Grants() {
					match := mode == "governor" && g.SubjectID == c.Confirmer && g.Terms.Action == gate.AdministerInstallation
					match = match || mode == "inspection" && g.SubjectID == SampleMember && g.Terms.Kind == draftauthority.ResourceKind && len(g.Terms.Fields) == 1
					if match {
						f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
						removed++
					}
				}
				if removed == 0 {
					t.Fatal("no actual standing removed")
				}
				expected = commands.PermissionDenied
			case "session":
				expected = commands.Unauthenticated
			case "final-digest":
				digest = "wrong"
			}
			before := copyConfiguration(a.configurations[c.Spec.ID])
			moduleBefore, e := a.documents.KindOf(SampleWorkspace, c.Kind.ID)
			f.must(e)
			var err error
			if mode == "session" {
				err = inputAuthorityAs(f, member, func() error {
					f.s.sessions.remove(f.s.caller)
					_, err := a.draftApplication(SampleWorkspace, SampleMember, draftauthority.Submit, c.Spec.ID, c.Digest, "issuance-final", digest)
					return err
				})
			} else {
				_, err = applicationAs(f, member, c, "issuance-final", digest)
			}
			documentFailure(t, err, expected)
			moduleAfter, e := a.documents.KindOf(SampleWorkspace, c.Kind.ID)
			f.must(e)
			if !reflect.DeepEqual(moduleBefore, moduleAfter) || !reflect.DeepEqual(before, a.configurations[c.Spec.ID]) || len(a.drafts) != 0 {
				t.Fatal("failed application changed ownership")
			}
		})
	}
}

func TestFinancialApplicationActivationAndOperationPrecedence(t *testing.T) {
	f, member, _, _ := applicationWorld(t)
	a := f.s.financialIssuance
	err := inputAuthorityAs(f, member, func() error {
		_, err := a.draftApplication(SampleWorkspace, SampleMember, draftauthority.Approve, "missing", "missing", "missing", "missing")
		return err
	})
	documentFailure(t, err, commands.PermissionDenied)
	older := setup(t, SeedSample).ready()
	older.setPassword(MemberLogin, pw)
	earlyMember := older.as(MemberLogin)
	p, e := proposeConfigurationAs(older, older.as(AdminLogin), configurationSpec())
	older.must(e)
	older.advance(2 * time.Minute)
	c, e := confirmConfigurationAs(older, older.as(Admin2Login), p)
	older.must(e)
	older.advance(-time.Minute)
	err = inputAuthorityAs(older, earlyMember, func() error {
		_, err := older.s.financialIssuance.ownedConfirmedConfiguration(SampleWorkspace, c.Spec.ID, c.Digest)
		return err
	})
	documentFailure(t, err, commands.EvidenceStale)
}

func TestFinancialApplicationInnerIntegrityAndApplicability(t *testing.T) {
	f, member, c, digest := applicationWorld(t)
	original, err := applicationAs(f, member, c, "issuance-final", digest)
	f.must(err)
	for _, mode := range []string{"schema", "outer", "source-inner", "configuration-inner", "contract", "billing-version", "document", "currency", "source-schema", "kind", "activation"} {
		t.Run(mode, func(t *testing.T) {
			p := copyIssuanceApplication(original)
			switch mode {
			case "schema":
				p.Schema = "unknown"
			case "outer":
				p.Digest = "wrong"
			case "source-inner":
				p.Source.Basis.Rows[0].Net = "99"
			case "configuration-inner":
				p.Configuration.Reason = "changed"
			case "contract":
				p.Source.Basis.Contract.Revision++
			case "billing-version":
				p.Source.Basis.BillingRevision++
			case "document":
				p.Source.Basis.Document.IssuerLegalEntityID = "other"
			case "currency":
				p.Source.Basis.Subject.Currency = "EUR"
			case "source-schema":
				p.Source.Basis.Schema = "unknown"
			case "kind":
				p.Configuration.Kind.ID = "other"
			case "activation":
				p.Configuration.ActivatedAt = p.Configuration.ActivatedAt.Add(time.Second)
			}
			if mode != "source-inner" {
				p.Source.Digest, err = financialEvidenceDigest("issuance_source", p.Source.Basis)
				f.must(err)
			}
			if mode != "configuration-inner" {
				p.Configuration.Digest, err = configurationDigest(p.Configuration)
				f.must(err)
			}
			if mode != "outer" {
				p.Digest, err = applicationDigest(p)
				f.must(err)
			}
			_, err = issuanceDraftFromApplication(p, "detached", "en")
			documentFailure(t, err, commands.EvidenceStale)
		})
	}
	for _, language := range []string{"", "xx"} {
		_, err = issuanceDraftFromApplication(original, "detached", language)
		documentFailure(t, err, commands.PrerequisiteFailed)
	}
}
