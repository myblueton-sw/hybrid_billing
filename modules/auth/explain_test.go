package auth

import (
	"testing"
	"time"
)

var trustRights = []Action{TrustPropose, TrustValidate, TrustApprove, TrustActivate}

func termsFor(action Action, workspace string) Terms {
	t := installationTerms(workspace)
	t.Action = action
	return t
}

func requestFor(action Action, workspace string) Request {
	return Request{Action: action, Kind: InstallationKind, Resource: Resource{WorkspaceID: workspace}}
}

func TestInstallationActionsAreExactlyTheFourteen(t *testing.T) {
	named := map[Action]bool{"administer_installation": true, "trust_propose": true, "trust_validate": true, "trust_approve": true, "trust_activate": true,
		"revoke_access_in_bulk": true, "administer_api_clients": true, "administer_modules": true, "approve_modules": true, "stop_modules": true,
		"administer_channels": true, "approve_channels": true, "stop_channels": true, "reprocess_events": true}
	for action := range actions {
		if InstallationAction(action) != named[action] {
			t.Errorf("%s: %v", action, InstallationAction(action))
		}
	}
	for action := range named {
		if _, known := actions[action]; !known {
			t.Errorf("%s is not in the catalogue", action)
		}
	}
	if InstallationAction("trust_unknown") || InstallationAction("administer_integrations") || len(named) != 14 {
		t.Error("an unknown action is an installation action")
	}
	rights := IntegrationRights()
	if len(rights) != 8 || rights[0] != AdministerAPIClients || rights[7] != ReprocessEvents {
		t.Errorf("integration rights %v", rights)
	}
	rights[0] = View
	if IntegrationRights()[0] != AdministerAPIClients {
		t.Error("the integration rights list is shared")
	}
	f := setupInstallation(t, true)
	for _, right := range append(append(trustRights, RevokeAccessInBulk), IntegrationRights()...) {
		if !f.Describes(right, InstallationKind, nil) || f.Describes(right, "usage_statement", nil) || f.Describes(right, InstallationKind, []string{"quantity"}) {
			t.Errorf("%s description", right)
		}
	}
}

func TestTrustRightsAndAdministrationAreSeparate(t *testing.T) {
	f := setupInstallation(t, true)
	if err := f.SetInstallationWorkspace("w1"); err != nil {
		t.Fatal(err)
	}
	f.grant("admin-alice", "alice", installationTerms("w1"))
	f.clientGrant("admin-app", installationTerms("w1"))
	for _, right := range trustRights {
		f.grant(string(right)+"-bob", "bob", termsFor(right, "w1"))
		f.clientGrant(string(right)+"-app", termsFor(right, "w1"))
	}
	for _, right := range trustRights {
		f.expect("administration confers no "+string(right), viaApp("alice"), requestFor(right, "w1"), false, ReasonInstallationScope)
		f.expect("the holder of "+string(right), viaApp("bob"), requestFor(right, "w1"), true, ReasonAllowed)
		f.expect("a service subject", viaApp("robot"), requestFor(right, "w1"), false, ReasonInstallationScope)
	}
	f.expect("trust rights confer no administration", viaApp("bob"), administer("w1"), false, ReasonInstallationScope)
	if err := f.AddGrant(Grant{ID: "robot-trust", SubjectID: "robot", Terms: termsFor(TrustPropose, "w1"), Validity: always}); err == nil {
		t.Error("a service subject received a trust right")
	}
	bad := termsFor(TrustApprove, "w1")
	bad.Kind = "usage_statement"
	if err := f.AddGrant(Grant{ID: "wrong-kind", SubjectID: "carol", Terms: bad, Validity: always}); err == nil {
		t.Error("a trust right on another kind")
	}
	if err := f.AddDelegation(Delegation{ID: "trust-delegation", DelegatorID: "bob", DelegateID: "carol", Terms: termsFor(TrustApprove, "w1"), Validity: always}); err == nil {
		t.Error("a trust right was delegated")
	}
}

func TestExplainInstallationAgreesWithDecide(t *testing.T) {
	f := setupInstallation(t, true)
	source := f.source.(*fakeSource)
	if err := f.SetInstallationWorkspace("w1"); err != nil {
		t.Fatal(err)
	}
	now := *f.now
	soon := Validity{NotBefore: now.Add(time.Hour), NotAfter: now.Add(2 * time.Hour)}
	brief := Validity{NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Minute)}
	for _, right := range append([]Action{AdministerInstallation}, trustRights...) {
		f.grant(string(right)+"-alice", "alice", termsFor(right, "w1"))
		f.clientGrant(string(right)+"-app", termsFor(right, "w1"))
		if err := f.AddGrant(Grant{ID: string(right) + "-bob", SubjectID: "bob", Terms: termsFor(right, "w1"), Validity: soon}); err != nil {
			t.Fatal(err)
		}
		if err := f.AddGrant(Grant{ID: string(right) + "-carol", SubjectID: "carol", Terms: termsFor(right, "w1"), Validity: brief}); err != nil {
			t.Fatal(err)
		}
	}
	f.delegations["forged"] = versioned[Delegation]{value: Delegation{ID: "forged", DelegatorID: "alice", DelegateID: "bob", Terms: termsFor(TrustApprove, "w1"), Validity: always}, version: 1}
	generation := f.Generation()
	for _, at := range []time.Time{now, now.Add(30 * time.Minute), now.Add(90 * time.Minute), now.Add(3 * time.Hour)} {
		*f.now = at
		for _, right := range append([]Action{AdministerInstallation}, trustRights...) {
			for _, subject := range []string{"alice", "bob", "carol", "robot", "ghost"} {
				for _, client := range []string{"", "app", "spare", "ghost"} {
					principal := who(subject)
					principal.ClientID = client
					if s, ok := source.subjects[subject]; ok {
						principal.Epoch = s.Epoch
					}
					want := f.Decide(principal, requestFor(right, "w1"))
					got := f.ExplainInstallation(subject, client, requestFor(right, "w1"), at)
					if got.Allowed != want.Allowed || got.Reason != want.Reason || got.Code != want.Code {
						t.Errorf("%s %s %q at %v: explain %+v decide %+v", right, subject, client, at.Sub(now), got, want)
					}
				}
			}
		}
	}
	*f.now = now
	if got := f.ExplainInstallation("alice", "app", requestFor(TrustApprove, "w1"), now.Add(3*time.Hour)); !got.Allowed {
		t.Errorf("an explicit time beyond the registry clock %+v", got)
	}
	if got := f.ExplainInstallation("carol", "app", requestFor(TrustApprove, "w1"), now.Add(time.Hour)); got.Allowed {
		t.Errorf("an expired grant at the given time %+v", got)
	}
	if got := f.ExplainInstallation("bob", "app", requestFor(TrustApprove, "w1"), now); got.Allowed {
		t.Errorf("a delegation or a future grant was consulted %+v", got)
	}
	if got := f.ExplainInstallation("alice", "app", ask(View, ""), now); got.Allowed || got.Reason != ReasonInvalidRequest {
		t.Errorf("a non-installation action %+v", got)
	}
	if f.Generation() != generation {
		t.Error("explaining changed the registry")
	}
}
