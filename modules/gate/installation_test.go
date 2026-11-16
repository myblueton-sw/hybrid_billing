package gate

import (
	"errors"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func installationTerms(workspace string) Terms {
	return Terms{Action: AdministerInstallation, Kind: InstallationKind, Period: AnyPeriod(),
		Scope: Scope{WorkspaceID: workspace, Customer: All(), LegalEntity: All(), Contract: All()}}
}

func administer() Request {
	return Request{Action: AdministerInstallation, Kind: InstallationKind, Resource: Resource{WorkspaceID: "w1"}}
}

func TestInstallationAdministrationNeedsTheAdministratorRole(t *testing.T) {
	f := setupWith(t, Catalog{Kinds: []string{"usage_statement", InstallationKind}, Fields: map[string]FieldClass{"quantity": General}})
	if err := f.a.SetInstallationWorkspace("bad id"); !errors.Is(err, ErrInvalid) {
		t.Errorf("an invalid workspace: %v", err)
	}
	f.must(f.a.SetInstallationWorkspace("w1"))
	if err := f.a.SetInstallationWorkspace("w2"); !errors.Is(err, ErrConflict) {
		t.Errorf("a second workspace: %v", err)
	}
	if !f.g.Describes(AdministerInstallation, InstallationKind, nil) || f.g.Describes(AdministerInstallation, "usage_statement", nil) {
		t.Error("the action is described only with its kind")
	}
	second(f.a.PutClient(Client{ID: "app", State: Active}, 0))
	f.must(f.a.AddGrant(Grant{ID: "console", ClientID: "app", Terms: installationTerms("w1"), Validity: always}))
	f.must(f.in.CreateUser(operator, "deputy", triple("deputy"), ""))
	f.join("deputy", identity.Administrator)
	f.member("ann")
	for _, id := range []string{"root", "deputy", "ann"} {
		f.must(f.a.AddGrant(Grant{ID: "admin-" + id, SubjectID: id, Terms: installationTerms("w1"), Validity: always}))
	}
	if err := f.a.AddGrant(Grant{ID: "elsewhere", SubjectID: "root", Terms: installationTerms("w2"), Validity: always}); !errors.Is(err, ErrInvalid) {
		t.Errorf("a grant in another workspace: %v", err)
	}
	f.expect("the bootstrap administrator", f.g.Decide(f.answer("root"), "app", administer()), true, ReasonAllowed)
	f.expect("a member who holds a grant", f.g.Decide(f.answer("ann"), "app", administer()), false, ReasonInstallationScope)
	f.expect("in a support session", f.g.DecideInSupport(f.answer("root"), "s", administer()), false, ReasonInstallationScope)
	other := administer()
	other.Resource.WorkspaceID = "w2"
	f.expect("another workspace", f.g.Decide(f.answer("root"), "app", other), false, ReasonWorkspace)
	earlier := f.expect("a second administrator", f.g.Decide(f.answer("deputy"), "app", administer()), true, ReasonAllowed)
	f.expect("a recheck before the demotion", f.g.Recheck(earlier.Stamp, "app", administer()), true, ReasonAllowed)
	f.must(f.w.ChangeRole(f.login("root"), "w1", "deputy", identity.Member, f.version("deputy")))
	f.expect("a recheck after the demotion", f.g.Recheck(earlier.Stamp, "app", administer()), false, ReasonIdentity)
	f.expect("a fresh answer after the demotion", f.g.Decide(f.answer("deputy"), "app", administer()), false, ReasonInstallationScope)
	invitation := f.next("invitation")
	if err := f.w.Invite(f.login("root"), "w2", invitation, identity.Administrator, v(invitation),
		identity.Validity{NotBefore: *f.now, NotAfter: f.now.Add(48 * time.Hour)}, "ann"); err == nil {
		t.Error("an invitation into another workspace")
	}
}

func TestAdministrationListingAndExplain(t *testing.T) {
	var none *Administration
	if none.Grants() != nil || none.Delegations() != nil || none.Explain("x", ask(Issue)).Code != AuthorityUnavailable {
		t.Error("zero administration")
	}
	f := setup(t)
	f.member("ann")
	id := f.grant("ann", Issue)
	views := f.a.Grants()
	if len(views) != 1 || views[0].ID != id || !views[0].Effective {
		t.Errorf("grants %+v", views)
	}
	if len(f.a.Delegations()) != 0 {
		t.Error("delegations")
	}
	if got := f.a.Explain("ann", ask(Issue)); !got.Allowed {
		t.Errorf("explain %+v", got)
	}
	if got := f.a.Explain("root", ask(Issue)); got.Allowed || got.Reason != ReasonNoGrant {
		t.Errorf("explain without grant %+v", got)
	}
}
