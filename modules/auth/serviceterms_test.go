package auth

import "testing"

func TestServiceTermsValidateCompleteSyntaxWithoutGrantingAuthority(t *testing.T) {
	f := setup(t)
	before := f.Generation()
	base := func() Terms { return terms(View, One("named-customer"), "quantity") }
	if !f.DescribesServiceTerms(base()) {
		t.Fatal("valid declared service restriction was refused")
	}
	for name, change := range map[string]func(*Terms){
		"unknown action":               func(t *Terms) { t.Action = "unknown" },
		"unknown kind":                 func(t *Terms) { t.Kind = "unknown" },
		"missing workspace":            func(t *Terms) { t.Scope.WorkspaceID = "" },
		"invalid workspace":            func(t *Terms) { t.Scope.WorkspaceID = "bad workspace" },
		"missing customer":             func(t *Terms) { t.Scope.Customer = Selector{} },
		"invalid customer":             func(t *Terms) { t.Scope.Customer = One("bad customer") },
		"missing legal entity":         func(t *Terms) { t.Scope.LegalEntity = Selector{} },
		"missing contract":             func(t *Terms) { t.Scope.Contract = Selector{} },
		"missing period":               func(t *Terms) { t.Period = Period{} },
		"empty period":                 func(t *Terms) { t.Period = Months(month(f.t, 2026, 1), month(f.t, 2026, 1)) },
		"reversed period":              func(t *Terms) { t.Period = Months(month(f.t, 2026, 2), month(f.t, 2026, 1)) },
		"missing fields":               func(t *Terms) { t.Fields = nil },
		"unknown field":                func(t *Terms) { t.Fields = []string{"unknown"} },
		"duplicate fields":             func(t *Terms) { t.Fields = []string{"quantity", "quantity"} },
		"wrong field class":            func(t *Terms) { t.Fields = []string{"unit_cost"} },
		"non-field action with fields": func(t *Terms) { t.Action = RequestApproval },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := base()
			change(&candidate)
			if f.DescribesServiceTerms(candidate) {
				t.Fatal("invalid full service terms accepted", name)
			}
		})
	}
	for _, allowed := range []Terms{terms(ViewCost, All(), "unit_cost"), terms(RequestApproval, One("unresolved-reference")), terms(Download, All(), "quantity"), terms(Issue, All()), terms(Deliver, All())} {
		if !f.DescribesServiceTerms(allowed) {
			t.Fatal("valid service vocabulary rejected", allowed.Action)
		}
	}
	if f.Generation() != before || len(f.Grants()) != 0 {
		t.Fatal("declaration validation changed authority or generation")
	}
}

func TestServiceTermsExcludeHumanAndInstallationActions(t *testing.T) {
	f := setup(t)
	for _, action := range []Action{Approve, ApproveSupportSession, AdministerInstallation, TrustPropose, TrustValidate, TrustApprove, TrustActivate, RevokeAccessInBulk, AdministerAPIClients, AdministerModules, ApproveModules, StopModules, AdministerChannels, ApproveChannels, StopChannels, ReprocessEvents} {
		if f.DescribesServiceTerms(terms(action, All())) {
			t.Fatal("service declaration admitted human or installation action", action)
		}
	}
	for _, unavailable := range []*Registry{nil, {}} {
		if unavailable.DescribesServiceTerms(terms(View, All(), "quantity")) {
			t.Fatal("unavailable registry validated a service declaration")
		}
	}
	f.SetAvailable(false)
	if f.DescribesServiceTerms(terms(View, All(), "quantity")) {
		t.Fatal("unavailable permission catalog validated a declaration")
	}
}

func TestServiceTermsRejectOtherwiseValidConfiguredInstallationRights(t *testing.T) {
	f := setupInstallation(t, true)
	if err := f.SetInstallationWorkspace("w1"); err != nil {
		t.Fatal(err)
	}
	for _, action := range append([]Action{AdministerInstallation, TrustPropose, TrustValidate, TrustApprove, TrustActivate, RevokeAccessInBulk}, IntegrationRights()...) {
		candidate := installationTerms("w1")
		candidate.Action = action
		if !f.validTerms(candidate) {
			t.Fatal("installation restriction premise was not otherwise valid", action)
		}
		if f.DescribesServiceTerms(candidate) {
			t.Fatal("configured installation rights admitted as service declaration", action)
		}
	}
}
