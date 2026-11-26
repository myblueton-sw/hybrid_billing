package server

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
)

func applicationWorld(t *testing.T) (world, actor, financialIssuanceConfiguration, string) {
	t.Helper()
	var configured financialIssuanceConfiguration
	f, _, _, digest := issuanceSourceWorld(t, func(f world) {
		p, err := proposeConfigurationAs(f, f.as(AdminLogin), configurationSpec())
		f.must(err)
		configured, err = confirmConfigurationAs(f, f.as(Admin2Login), p)
		f.must(err)
		if _, err = f.s.approvals.RequestOf(SampleWorkspace, "issuance-final"); !errors.Is(err, approvals.ErrNotFound) {
			t.Fatal("configuration must precede final evidence")
		}
	})
	grantDraftCapability(f, draftauthority.Submit)
	grantDraftCapability(f, draftauthority.Inspect)
	return f, f.as(MemberLogin), configured, digest
}

func applicationAs(f world, who actor, c financialIssuanceConfiguration, request, digest string) (financialIssuanceApplication, error) {
	var p financialIssuanceApplication
	err := inputAuthorityAs(f, who, func() error {
		var err error
		p, err = f.s.financialIssuance.draftApplication(SampleWorkspace, f.s.answer.User(), draftauthority.Submit, c.Spec.ID, c.Digest, request, digest)
		return err
	})
	return p, err
}

func TestFinancialApplicationActualSourceAndDetachedCopies(t *testing.T) {
	f, member, c, digest := applicationWorld(t)
	a := f.s.financialIssuance
	before, err := f.s.approvals.RequestOf(SampleWorkspace, "issuance-final")
	f.must(err)
	p, err := applicationAs(f, member, c, "issuance-final", digest)
	f.must(err)
	if p.Schema != "hb_issuance_application_v1" || !reflect.DeepEqual(p.Configuration, c) || p.Source.Basis.Net != "15.00" || p.Source.Basis.Tax != "1.00" || len(p.Source.Basis.Contributors) == 0 {
		t.Fatal("application lacks authoritative evidence")
	}
	d, err := issuanceDraftFromApplication(p, "application-draft", "en")
	f.must(err)
	if d.Kind != c.Kind.ID || d.Total != "16" || d.Currency != "USD" || len(d.Lines) != 4 || d.Lines[0].Amount != "10.25" || d.Lines[1].Amount != "0.25" || d.Lines[2].Amount != "4.75" || d.Lines[3].Amount != "0.75" {
		t.Fatal("wrong source-derived monetary draft")
	}
	_, err = a.documents.DraftOf(SampleWorkspace, d.ID)
	if !errors.Is(err, issuance.ErrNotFound) {
		t.Fatal("detached builder submitted draft")
	}
	copied := copyIssuanceApplication(p)
	copied.Configuration.Scope.Parties[0].Party.ID = "changed"
	copied.Configuration.Kind.Languages[0] = "changed"
	copied.Source.Basis.Rows[0].Net = "999"
	copied.Source.Basis.Subject.Digests["result"] = "changed"
	copied.Source.Basis.Contributors[0] = "changed"
	copied.Source.Basis.Contract.Components[0].ID = "changed"
	copied.Source.Basis.Decision.Decision.Exclude = append(copied.Source.Basis.Decision.Decision.Exclude, approvals.Exclusion{Item: "changed", Reason: "changed"})
	again, err := applicationAs(f, member, c, "issuance-final", digest)
	f.must(err)
	if !reflect.DeepEqual(again, p) {
		t.Fatal("detached result changed source/configuration")
	}
	after, err := f.s.approvals.RequestOf(SampleWorkspace, "issuance-final")
	f.must(err)
	if !reflect.DeepEqual(before, after) || len(a.configurations) != 1 || len(a.profiles) != 0 || len(a.drafts) != 0 || len(a.kindReservations) != 0 {
		t.Fatal("application construction changed ownership or consumed approval")
	}
}

func TestFinancialApplicationNextInvoiceReusesConfigurationAfterExpiry(t *testing.T) {
	f, member, c, digest := applicationWorld(t)
	first, err := applicationAs(f, member, c, "issuance-final", digest)
	f.must(err)
	before, err := f.s.approvals.RequestOf(SampleWorkspace, "issuance-final")
	f.must(err)
	e := approvals.Execution{ID: before.Subject.ID, Request: before.Subject.ID, Stage: before.Subject.Stage, Target: before.Subject.Target, Digests: before.Subject.Digests, Amount: before.Subject.Amount, Currency: before.Subject.Currency}
	f.must(f.s.approvals.CheckCurrent(SampleWorkspace, e, approvals.Actor{ID: SampleMember, Kind: approvals.KindHuman}))
	f.advance(24*time.Hour + time.Minute)
	renewInvoiceAuthorities(t, f)
	err = f.s.approvals.CheckCurrent(SampleWorkspace, e, approvals.Actor{ID: SampleMember, Kind: approvals.KindHuman})
	var refusal approvals.Refusal
	if !errors.As(err, &refusal) || refusal.Reason != approvals.ReasonExpired {
		t.Fatalf("old financial approval has not expired: %v", err)
	}
	_, err = applicationAs(f, f.as(MemberLogin), c, "issuance-final", digest)
	documentFailure(t, err, commands.PrerequisiteFailed)
	from := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	candidate, candidateDigest := captureInvoiceInWorld(t, f, f.as(MemberLogin), f.as(Admin3Login), "application-b", "2026-11", from, to)
	final, finalDigest := finalizeInvoiceInWorld(t, f, "application-b", "2026-11", candidate, candidateDigest)
	second, err := applicationAs(f, f.as(MemberLogin), c, final, finalDigest)
	f.must(err)
	if !reflect.DeepEqual(first.Configuration, second.Configuration) || first.Digest == second.Digest || first.Source.Digest == second.Source.Digest || first.Source.Basis.Subject.Target == second.Source.Basis.Subject.Target || second.Source.Basis.FinalRequest != final || second.Source.Basis.Period != "2026-11" {
		t.Fatal("new invoice did not reuse exact configuration with fresh source")
	}
	a := f.s.financialIssuance
	actual, err := a.documents.KindOf(SampleWorkspace, c.Kind.ID)
	f.must(err)
	if !reflect.DeepEqual(actual, c.Active) || !reflect.DeepEqual(a.configurations[c.Spec.ID], c) || len(a.configurations) != 1 || len(a.configurationScopes) != 1 || len(a.profiles) != 0 || len(a.drafts) != 0 {
		t.Fatal("second invoice rebuilt configuration or issued a draft")
	}
	after, err := f.s.approvals.RequestOf(SampleWorkspace, "issuance-final")
	f.must(err)
	if !reflect.DeepEqual(after, before) {
		t.Fatal("new source changed old history")
	}
	if _, err := issuanceDraftFromApplication(first, "historical-draft", "en"); err != nil {
		t.Fatal("pure historical builder unexpectedly required current approval", err)
	}
}
