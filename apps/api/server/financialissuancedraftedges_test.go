package server

import (
	"errors"
	"fmt"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
	"reflect"
	"testing"
)

func grantOtherDraftOperator(f world, recipient string) {
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	defer f.s.endProbe(root.verifier(f))
	defer f.s.endProbe(second.verifier(f))
	for _, capability := range []string{draftauthority.Submit, draftauthority.Inspect} {
		spec := draftPermissionSpec(f, capability)
		p, err := draftPermissionProposal(f, root, inputauthority.Proposal{ID: "other-" + capability + "-ceiling", Spec: spec, Rationale: "Independent second draft operator"})
		f.must(err)
		ceiling, err := draftPermissionDecision(f, second, p)
		f.must(err)
		spec.Kind, spec.Recipient, spec.CeilingID = inputauthority.Human, recipient, ceiling.Decision.GrantID
		p, err = draftPermissionProposal(f, root, inputauthority.Proposal{ID: "other-" + capability + "-human", Spec: spec, Rationale: "Explicit second draft operator"})
		f.must(err)
		_, err = draftPermissionDecision(f, second, p)
		f.must(err)
	}
}
func TestFinancialDraftSubmissionWrongAuthorizedRecoveryActor(t *testing.T) {
	f, member, spec, _ := draftSubmissionWorld(t)
	grantOtherDraftOperator(f, SampleAdmin3)
	other := f.as(Admin3Login)
	f.must(inputAuthorityAs(f, other, func() error {
		return f.s.financialIssuance.draftOperation(SampleWorkspace, SampleAdmin3, draftauthority.Submit)
	}))
	f.s.fault = func(at string) bool { return at == "issuance-draft-after-submit" }
	_, err := submitDraftAs(f, member, spec)
	documentFailure(t, err, commands.DependencyUnavailable)
	f.s.fault = nil
	saved := copyFinancialDraft(f.s.financialIssuance.drafts[spec.ID])
	_, err = submitDraftAs(f, other, spec)
	documentFailure(t, err, commands.EvidenceStale)
	if !reflect.DeepEqual(saved, f.s.financialIssuance.drafts[spec.ID]) {
		t.Fatal("wrong actor changed recovery owner")
	}
	_, err = submitDraftAs(f, member, spec)
	f.must(err)
}
func TestFinancialDraftSubmissionLostRightsAndForeignRevision(t *testing.T) {
	for _, mode := range []string{"right", "revision", "scope"} {
		t.Run(mode, func(t *testing.T) {
			f, member, spec, _ := draftSubmissionWorld(t)
			a := f.s.financialIssuance
			p, err := submitDraftAs(f, member, spec)
			f.must(err)
			expected := commands.EvidenceStale
			switch mode {
			case "right":
				expected = commands.PermissionDenied
				revoked := 0
				for _, g := range f.s.admin.Grants() {
					if g.SubjectID == SampleMember && g.Terms.Action == gate.SubmitIssuanceDraft {
						f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
						revoked++
					}
				}
				if revoked == 0 {
					t.Fatal("no actual submit grant revoked")
				}
			case "revision":
				changed := copyModuleDraft(p.Draft)
				changed.Lines[0].Text = "foreign revision"
				_, err = a.documents.Submit(SampleWorkspace, changed, issuance.Actor{ID: SampleAdmin3, Kind: issuance.KindHuman})
				f.must(err)
			case "scope":
				delete(a.draftScopes, p.Profile.Source.Basis.Subject.Target)
			}
			_, err = submitDraftAs(f, member, spec)
			documentFailure(t, err, expected)
			if !reflect.DeepEqual(a.drafts[spec.ID], p) {
				t.Fatal("refusal rewrote retained draft")
			}
		})
	}
}
func TestFinancialDraftSubmissionActualModuleCapacityRefusal(t *testing.T) {
	f, member, spec, profile := draftSubmissionWorld(t)
	a := f.s.financialIssuance
	d, err := issuanceDraftFromSource(profile, "capacity", spec.Language)
	f.must(err)
	for i := 0; i < 100; i++ {
		d.ID = fmt.Sprintf("existing-draft-%d", i)
		_, err = a.documents.Submit(SampleWorkspace, d, issuance.Actor{ID: SampleMember, Kind: issuance.KindHuman})
		f.must(err)
	}
	_, err = submitDraftAs(f, member, spec)
	if !errors.Is(err, issuance.ErrLimit) {
		t.Fatal("actual module capacity not refused", err)
	}
	saved := copyFinancialDraft(a.drafts[spec.ID])
	if saved.State != "submission_refused" || saved.Module.Digest != "" || a.draftScopes[profile.Source.Basis.Subject.Target] != spec.ID {
		t.Fatal("known no-effect refusal misclassified")
	}
	_, err = a.documents.DraftOf(SampleWorkspace, spec.ID)
	if !errors.Is(err, issuance.ErrNotFound) {
		t.Fatal("capacity failure created module draft")
	}
	competing := spec
	competing.ID = "capacity-escape"
	_, err = submitDraftAs(f, member, competing)
	documentFailure(t, err, commands.PrerequisiteFailed)
	_, err = submitDraftAs(f, member, spec)
	if !errors.Is(err, issuance.ErrLimit) || !reflect.DeepEqual(saved, a.drafts[spec.ID]) {
		t.Fatal("exact known-refusal retry changed intent", err)
	}
}

func TestFinancialDraftMissingRecordedModuleRefuses(t *testing.T) {
	f, member, spec, profile := draftSubmissionWorld(t)
	a := f.s.financialIssuance
	p, err := submitDraftAs(f, member, spec)
	f.must(err)
	fresh, err := newFinancialIssuance(f.s)
	f.must(err)
	_, err = fresh.documents.Define(SampleWorkspace, profile.Kind, issuance.Actor{ID: profile.Proposer, Kind: issuance.KindHuman})
	f.must(err)
	_, err = fresh.documents.ApproveKind(SampleWorkspace, profile.Kind.ID, issuance.Actor{ID: profile.Confirmer, Kind: issuance.KindHuman})
	f.must(err)
	a.documents = fresh.documents
	_, err = submitDraftAs(f, member, spec)
	documentFailure(t, err, commands.EvidenceStale)
	if !reflect.DeepEqual(a.drafts[spec.ID], p) {
		t.Fatal("missing observed module rewrote intent")
	}
	_, err = a.documents.DraftOf(SampleWorkspace, spec.ID)
	if !errors.Is(err, issuance.ErrNotFound) {
		t.Fatal("missing observed module silently recreated")
	}
}

func TestFinancialDraftRecoveryRechecksCurrentEvidence(t *testing.T) {
	for _, mode := range []string{"source", "submit", "inspect"} {
		t.Run(mode, func(t *testing.T) {
			f, member, spec, _ := draftSubmissionWorld(t)
			a := f.s.financialIssuance
			f.s.fault = func(at string) bool { return at == "issuance-draft-after-submit" }
			_, err := submitDraftAs(f, member, spec)
			documentFailure(t, err, commands.DependencyUnavailable)
			f.s.fault = nil
			saved := copyFinancialDraft(a.drafts[spec.ID])
			expected := commands.PermissionDenied
			if mode == "source" {
				expected = commands.EvidenceStale
				key := [2]string{SampleWorkspace, "public-result"}
				app := f.s.documentERP.resultApplications[key]
				app.receipt = "changed"
				f.s.documentERP.resultApplications[key] = app
			} else {
				action := gate.SubmitIssuanceDraft
				if mode == "inspect" {
					action = gate.ViewCost
				}
				count := 0
				for _, g := range f.s.admin.Grants() {
					if g.SubjectID == SampleMember && g.Terms.Kind == draftauthority.ResourceKind && g.Terms.Action == action {
						f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
						count++
					}
				}
				if count == 0 {
					t.Fatal("no actual recovery right revoked")
				}
			}
			_, err = submitDraftAs(f, member, spec)
			documentFailure(t, err, expected)
			if !reflect.DeepEqual(saved, a.drafts[spec.ID]) {
				t.Fatal("failed recovery changed unknown ownership")
			}
			actual, err := a.documents.DraftOf(SampleWorkspace, spec.ID)
			f.must(err)
			if !reflect.DeepEqual(saved.Module, actual) {
				t.Fatal("failed recovery revised actual module")
			}
		})
	}
}
