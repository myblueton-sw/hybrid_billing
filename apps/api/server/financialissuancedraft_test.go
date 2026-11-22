package server

import (
	"errors"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
	"reflect"
	"slices"
	"testing"
)

func draftSubmissionWorld(t *testing.T) (world, actor, financialDraftSpec, financialIssuanceProfile) {
	t.Helper()
	f, member, p := termProfileWorld(t)
	grantDraftCapability(f, draftauthority.Submit)
	grantDraftCapability(f, draftauthority.Inspect)
	return f, member, financialDraftSpec{ID: "retained-draft", Profile: p.Spec.ID, ProfileDigest: p.Digest, Language: "en"}, p
}
func submitDraftAs(f world, who actor, spec financialDraftSpec) (financialDraftRecord, error) {
	var out financialDraftRecord
	err := inputAuthorityAs(f, who, func() error {
		var err error
		out, err = f.s.financialIssuance.submitDraft(SampleWorkspace, f.s.answer.User(), spec)
		return err
	})
	return out, err
}
func TestFinancialDraftActualSubmissionAndReplay(t *testing.T) {
	f, member, spec, profile := draftSubmissionWorld(t)
	before, err := f.s.approvals.RequestOf(SampleWorkspace, "issuance-final")
	f.must(err)
	p, err := submitDraftAs(f, member, spec)
	f.must(err)
	if p.State != "submitted" || p.Draft.Total != "16" || len(p.Draft.Lines) != 4 || !p.Draft.Lines[1].Tax || p.Draft.Lines[0].Tax || p.Module.Revision != 1 || p.Module.Approver != "" || p.FinalRequester != before.Requester || !reflect.DeepEqual(p.Profile, profile) {
		t.Fatal("draft differs from actual financial source")
	}
	actual, err := f.s.financialIssuance.documents.DraftOf(SampleWorkspace, spec.ID)
	f.must(err)
	if !reflect.DeepEqual(actual, p.Module) {
		t.Fatal("missing actual module draft")
	}
	replay, err := submitDraftAs(f, member, spec)
	f.must(err)
	if !reflect.DeepEqual(replay, p) {
		t.Fatal("replay revised draft")
	}
	p.Profile.Source.Basis.Rows[0].Net = "999"
	p.Module.Draft.Lines[0].Amount = "999"
	p.Draft.Lines[0].Amount = "999"
	again, err := submitDraftAs(f, member, spec)
	f.must(err)
	if !reflect.DeepEqual(again, replay) {
		t.Fatal("returned draft alias")
	}
	after, err := f.s.approvals.RequestOf(SampleWorkspace, "issuance-final")
	f.must(err)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("submission consumed approval")
	}
	other := spec
	other.ID = "competing-draft"
	_, err = submitDraftAs(f, member, other)
	documentFailure(t, err, commands.PrerequisiteFailed)
}
func TestFinancialDraftSubmissionInterruptions(t *testing.T) {
	for _, point := range []string{"before-submit", "after-intent", "after-submit", "after-submission-publication"} {
		t.Run(point, func(t *testing.T) {
			f, member, spec, _ := draftSubmissionWorld(t)
			a := f.s.financialIssuance
			f.s.fault = func(at string) bool { return at == "issuance-draft-"+point }
			_, err := submitDraftAs(f, member, spec)
			documentFailure(t, err, commands.DependencyUnavailable)
			f.s.fault = nil
			if point == "before-submit" && (len(a.drafts) != 0 || len(a.draftScopes) != 0) {
				t.Fatal("before-write reserved intent")
			}
			if point == "after-intent" {
				retained := a.drafts[spec.ID]
				if retained.State != "submission_unknown" || retained.Module.Digest != "" || a.draftScopes[retained.Profile.Source.Basis.Subject.Target] != spec.ID {
					t.Fatal("missing reserved unknown intent")
				}
				_, absent := a.documents.DraftOf(SampleWorkspace, spec.ID)
				if !errors.Is(absent, issuance.ErrNotFound) {
					t.Fatal("after-intent wrote module draft", absent)
				}
			}
			p, err := submitDraftAs(f, member, spec)
			f.must(err)
			if p.State != "submitted" || p.Module.Revision != 1 || len(a.drafts) != 1 {
				t.Fatal("lost exact recovery")
			}
		})
	}
}
func TestFinancialDraftSubmissionForeignAndChangedEvidence(t *testing.T) {
	for _, mode := range []string{"foreign", "source", "intent", "lookup"} {
		t.Run(mode, func(t *testing.T) {
			f, member, spec, profile := draftSubmissionWorld(t)
			a := f.s.financialIssuance
			expected := commands.EvidenceStale
			switch mode {
			case "foreign":
				d, err := issuanceDraftFromSource(profile, spec.ID, spec.Language)
				f.must(err)
				_, err = a.documents.Submit(SampleWorkspace, d, issuance.Actor{ID: SampleMember, Kind: issuance.KindHuman})
				f.must(err)
			case "source":
				_, err := submitDraftAs(f, member, spec)
				f.must(err)
				key := [2]string{SampleWorkspace, "public-result"}
				app := f.s.documentERP.resultApplications[key]
				app.receipt = "changed"
				f.s.documentERP.resultApplications[key] = app
			case "intent":
				_, err := submitDraftAs(f, member, spec)
				f.must(err)
				spec.Language = "ko"
			case "lookup":
				f.s.fault = func(at string) bool { return at == "issuance-draft-lookup" }
				expected = commands.DependencyUnavailable
			}
			_, err := submitDraftAs(f, member, spec)
			documentFailure(t, err, expected)
			if (mode == "foreign" || mode == "lookup") && len(a.drafts) != 0 {
				t.Fatal("adopted foreign or unavailable module")
			}
		})
	}
}
func TestFinancialDraftSourceMappingBoundaries(t *testing.T) {
	_, _, spec, profile := draftSubmissionWorld(t)
	original, err := issuanceDraftFromSource(profile, spec.ID, spec.Language)
	if err != nil {
		t.Fatal(err)
	}
	reversed := copyIssuanceProfile(profile)
	slices.Reverse(reversed.Source.Basis.Rows)
	same, err := issuanceDraftFromSource(reversed, spec.ID, spec.Language)
	if err != nil || !reflect.DeepEqual(same, original) {
		t.Fatal("row order changed draft", err)
	}
	for _, mode := range []string{"duplicate", "negative", "overflow", "gross", "allocation", "all-zero", "subject", "language"} {
		p := copyIssuanceProfile(profile)
		language := spec.Language
		switch mode {
		case "duplicate":
			p.Source.Basis.Rows[1].ID = p.Source.Basis.Rows[0].ID
		case "negative":
			p.Source.Basis.Rows[0].Net = "-1"
		case "overflow":
			p.Source.Basis.Rows[0].Net = "999999999999999999999999999999999"
		case "gross":
			p.Source.Basis.Subject.Amount = "17"
		case "allocation":
			p.Source.Basis.Net = "14.00"
			p.Source.Basis.Tax = "2.00"
		case "all-zero":
			for i := range p.Source.Basis.Rows {
				p.Source.Basis.Rows[i].Net = "0"
				p.Source.Basis.Rows[i].Tax = "0"
			}
			p.Source.Basis.Net = "0.00"
			p.Source.Basis.Tax = "0.00"
			p.Source.Basis.Subject.Amount = "0"
		case "subject":
			p.Source.Basis.Customer = ""
		case "language":
			language = "fr"
		}
		_, err := issuanceDraftFromSource(p, spec.ID, language)
		documentFailure(t, err, commands.PrerequisiteFailed)
	}
	zero := copyIssuanceProfile(profile)
	zero.Source.Basis.Rows = append(zero.Source.Basis.Rows, financialERPResultRow{ID: "zero-row", Net: "0", Tax: "0"})
	result, err := issuanceDraftFromSource(zero, spec.ID, spec.Language)
	if err != nil || !reflect.DeepEqual(result, original) {
		t.Fatal("zero component emitted or lost totals", err)
	}
}
