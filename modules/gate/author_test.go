package gate

import (
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func TestCheckAuthorNeedsAnAdministratorAndTheConsoleRight(t *testing.T) {
	var none *Gate
	modules := Request{Action: AdministerModules, Kind: InstallationKind, Resource: Resource{WorkspaceID: "w1"}}
	if got := none.CheckAuthor(identity.Answer{}, "app", "root", modules); got.Code != AuthorityUnavailable {
		t.Errorf("zero gate %+v", got)
	}
	if got := none.RecheckAuthor(Stamp{}, "app", "root", modules); got.Code != AuthorityUnavailable {
		t.Errorf("zero gate recheck %+v", got)
	}
	f := setupWith(t, Catalog{AuthorClient: "app", Kinds: []string{"usage_statement", InstallationKind}})
	f.must(f.a.SetInstallationWorkspace("w1"))
	second(f.a.PutClient(Client{ID: "app", State: Active}, 0))
	terms := installationTerms("w1")
	terms.Action = AdministerModules
	f.must(f.a.AddGrant(Grant{ID: "console", ClientID: "app", Terms: terms, Validity: always}))
	f.must(f.in.CreateUser(operator, "deputy", triple("deputy"), ""))
	f.join("deputy", identity.Administrator)
	f.member("ann")
	first := f.expect("the author", f.g.CheckAuthor(f.answer("root"), "app", "root", modules), true, ReasonAllowed)
	f.expect("the recheck", f.g.RecheckAuthor(first.Stamp, "app", "root", modules), true, ReasonAllowed)
	f.expect("an unissued stamp", f.g.RecheckAuthor(Stamp{}, "app", "root", modules), false, ReasonStamp)
	f.expect("another administrator", f.g.CheckAuthor(f.answer("deputy"), "app", "root", modules), false, ReasonNotAuthor)
	f.expect("a member author", f.g.CheckAuthor(f.answer("ann"), "app", "ann", modules), false, ReasonInstallationScope)
	other := modules
	other.Resource.WorkspaceID = "w2"
	f.expect("another workspace", f.g.CheckAuthor(f.answer("root"), "app", "root", other), false, ReasonWorkspace)
	earlier := f.expect("the deputy before demotion", f.g.CheckAuthor(f.answer("deputy"), "app", "deputy", modules), true, ReasonAllowed)
	f.must(f.w.ChangeRole(f.login("root"), "w1", "deputy", identity.Member, f.version("deputy")))
	f.expect("a recheck after demotion", f.g.RecheckAuthor(earlier.Stamp, "app", "deputy", modules), false, ReasonIdentity)
}

func TestIntegrationRightsAreReExported(t *testing.T) {
	rights := IntegrationRights()
	if len(rights) != 8 || rights[0] != AdministerAPIClients || rights[7] != ReprocessEvents {
		t.Errorf("rights %v", rights)
	}
	for _, right := range rights {
		if !InstallationAction(right) {
			t.Errorf("%s is not an installation action", right)
		}
	}
}

func TestIntegrationRightNamesAreStable(t *testing.T) {
	for got, want := range map[Action]string{AdministerAPIClients: "administer_api_clients", AdministerModules: "administer_modules", ApproveModules: "approve_modules",
		StopModules: "stop_modules", AdministerChannels: "administer_channels", ApproveChannels: "approve_channels", StopChannels: "stop_channels", ReprocessEvents: "reprocess_events"} {
		if string(got) != want {
			t.Errorf("%s != %s", got, want)
		}
	}
}
