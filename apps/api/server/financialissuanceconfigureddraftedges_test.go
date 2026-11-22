package server

import (
	"errors"
	"reflect"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
)

func TestFinancialConfiguredDraftMixedOrigins(t *testing.T) {
	f, member, c, digest := applicationWorld(t)
	a := f.s.financialIssuance
	spec := financialDraftSpec{ID: "origin-draft", Language: "en", Application: &financialDraftApplicationRequest{Schema: "hb_issuance_application_request_v1", Configuration: configurationReference(c), FinalRequest: "issuance-final", FinalDigest: digest}}
	for _, mode := range []string{"missing", "schema", "legacy-id", "legacy-digest", "configuration-schema", "final-digest"} {
		s := spec
		r := *spec.Application
		s.Application = &r
		switch mode {
		case "missing":
			s.Application = nil
		case "schema":
			r.Schema = "unknown"
		case "legacy-id":
			s.Profile = "legacy"
		case "legacy-digest":
			s.ProfileDigest = digest
		case "configuration-schema":
			r.Configuration.Schema = "unknown"
		case "final-digest":
			r.FinalDigest = "invalid"
		}
		_, err := submitDraftAs(f, member, s)
		documentFailure(t, err, commands.InvalidRequest)
		if len(a.drafts) != 0 || len(a.draftScopes) != 0 {
			t.Fatal("invalid origin reserved draft")
		}
	}
	draft, err := submitDraftAs(f, member, spec)
	f.must(err)
	for _, mode := range []string{"missing", "legacy-payload", "request", "configuration", "source", "application-digest"} {
		t.Run(mode, func(t *testing.T) {
			p := copyFinancialDraft(draft)
			switch mode {
			case "missing":
				p.Application = nil
			case "legacy-payload":
				p.Profile.Reason = "legacy"
			case "request":
				p.Spec.Application.FinalRequest = "other-final"
			case "configuration":
				p.Application.Configuration.Reason = "changed"
			case "source":
				p.Application.Source.Basis.Rows[0].Net = "999"
			case "application-digest":
				p.Application.Digest = "invalid"
			}
			if _, err := draftIntentDigest(p); err == nil {
				t.Fatal("corrupt origin received valid intent digest")
			}
			raw := copyFinancialDraft(p)
			raw.Digest, raw.State = "", ""
			raw.Module = issuance.DraftView{}
			p.Digest, err = financialEvidenceDigest("issuance_application_draft_v1", raw)
			f.must(err)
			a.drafts[p.Spec.ID] = p
			_, err = a.ownedDraft(SampleWorkspace, p.Spec.ID, p.Digest)
			documentFailure(t, err, commands.EvidenceStale)
			actual, err := a.documents.DraftOf(SampleWorkspace, p.Spec.ID)
			f.must(err)
			if !reflect.DeepEqual(actual, draft.Module) || !reflect.DeepEqual(a.drafts[p.Spec.ID], p) {
				t.Fatal("corrupt origin changed actual state")
			}
			a.drafts[p.Spec.ID] = copyFinancialDraft(draft)
		})
	}
}

func TestFinancialConfiguredDraftCrossFormatEconomicOwnership(t *testing.T) {
	for _, configuredFirst := range []bool{false, true} {
		for _, phase := range []string{"submitted", "unknown", "missing-reverse"} {
			name := "legacy-first-" + phase
			if configuredFirst {
				name = "configured-first-" + phase
			}
			t.Run(name, func(t *testing.T) {
				f, member, legacy, profile := draftSubmissionWorld(t)
				a := f.s.financialIssuance
				cp, err := proposeConfigurationAs(f, f.as(AdminLogin), configurationSpec())
				f.must(err)
				c, err := confirmConfigurationAs(f, f.as(Admin2Login), cp)
				f.must(err)
				configured := financialDraftSpec{ID: legacy.ID, Language: "en", Application: &financialDraftApplicationRequest{Schema: "hb_issuance_application_request_v1", Configuration: configurationReference(c), FinalRequest: profile.Source.Basis.FinalRequest, FinalDigest: profile.Source.Basis.FinalDigest}}
				first, other := legacy, configured
				if configuredFirst {
					first, other = configured, legacy
				}
				if phase != "submitted" {
					f.s.fault = func(at string) bool { return at == "issuance-draft-after-intent" }
				}
				_, err = submitDraftAs(f, member, first)
				if phase != "submitted" {
					documentFailure(t, err, commands.DependencyUnavailable)
				} else {
					f.must(err)
				}
				f.s.fault = nil
				retained := copyFinancialDraft(a.drafts[first.ID])
				_, err = submitDraftAs(f, member, other)
				documentFailure(t, err, commands.EvidenceStale)
				other.ID = "other-format-draft"
				if phase == "missing-reverse" {
					delete(a.draftScopes, profile.Source.Basis.Subject.Target)
				}
				_, err = submitDraftAs(f, member, other)
				expected := commands.PrerequisiteFailed
				if phase == "missing-reverse" {
					expected = commands.EvidenceStale
				}
				documentFailure(t, err, expected)
				_, err = a.documents.DraftOf(SampleWorkspace, other.ID)
				if !errors.Is(err, issuance.ErrNotFound) || len(a.drafts) != 1 || !reflect.DeepEqual(a.drafts[first.ID], retained) {
					t.Fatal("other format bypassed economic ownership")
				}
				if phase == "missing-reverse" && len(a.draftScopes) != 0 {
					t.Fatal("failed attempt repaired missing ownership")
				}
			})
		}
	}
}

func TestFinancialConfiguredDraftCorruptCompetingIntent(t *testing.T) {
	for _, mode := range []string{"outer-digest", "empty-target"} {
		t.Run(mode, func(t *testing.T) {
			f, member, c, digest := applicationWorld(t)
			a := f.s.financialIssuance
			spec := financialDraftSpec{ID: "owned-draft", Language: "en", Application: &financialDraftApplicationRequest{Schema: "hb_issuance_application_request_v1", Configuration: configurationReference(c), FinalRequest: "issuance-final", FinalDigest: digest}}
			f.s.fault = func(at string) bool { return at == "issuance-draft-after-intent" }
			_, err := submitDraftAs(f, member, spec)
			documentFailure(t, err, commands.DependencyUnavailable)
			f.s.fault = nil
			p := copyFinancialDraft(a.drafts[spec.ID])
			delete(a.draftScopes, p.Application.Source.Basis.Subject.Target)
			switch mode {
			case "outer-digest":
				p.FinalRequester = "different-requester"
			case "empty-target":
				p.Application.Source.Basis.Subject.Target = ""
				p.Application.Source.Digest, err = financialEvidenceDigest("issuance_source", p.Application.Source.Basis)
				f.must(err)
				p.Application.Digest, err = applicationDigest(*p.Application)
				f.must(err)
				p.Digest, err = draftIntentDigest(p)
				f.must(err)
			}
			a.drafts[spec.ID] = p
			spec.ID = "competing-draft"
			_, err = submitDraftAs(f, member, spec)
			documentFailure(t, err, commands.EvidenceStale)
			_, err = a.documents.DraftOf(SampleWorkspace, spec.ID)
			if !errors.Is(err, issuance.ErrNotFound) || len(a.draftScopes) != 0 || len(a.drafts) != 1 || !reflect.DeepEqual(a.drafts[p.Spec.ID], p) {
				t.Fatal("corrupt competitor allowed reservation or module write")
			}
		})
	}
}
