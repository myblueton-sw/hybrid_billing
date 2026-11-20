package server

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func TestFirstAdministratorHTTPActivationRetainsExactRightsAfterRollbackFailure(t *testing.T) {
	f := setup(t, SeedNone).ready()
	token := f.links()[""]
	login := "rollback-admin@example.invalid"
	failures, rollbacks := 0, 0
	activated := func() reply {
		previous := f.s.fault
		defer func() { f.s.fault = previous }()
		f.s.fault = func(at string) bool {
			switch at {
			case "integration-grant-3":
				failures++
				return true
			case "integration-rollback":
				rollbacks++
				return true
			}
			return false
		}
		return f.auth("redeem", activationBody(token, login), f.pre())
	}()
	if activated.status != 204 || failures != 1 || rollbacks != 4 || f.s.boot.live {
		t.Fatal("incomplete grant rollback prevented genuine bootstrap completion")
	}
	if _, err := os.Stat(filepath.Join(f.s.cfg.Data, ActivationFile)); !os.IsNotExist(err) {
		t.Fatal("completed activation retained its activation file")
	}
	expected := map[gate.Action]bool{
		gate.AdministerAPIClients: true, gate.AdministerModules: true, gate.ApproveModules: true, gate.StopModules: true,
		gate.AdministerChannels: false, gate.ApproveChannels: false, gate.StopChannels: false, gate.ReprocessEvents: false,
	}
	clients, subjects := map[gate.Action]bool{}, map[gate.Action]bool{}
	for _, grant := range f.s.admin.Grants() {
		if !strings.HasPrefix(grant.ID, integrationGrantID) {
			continue
		}
		wanted, known := expected[grant.Terms.Action]
		if !known || grant.Closed || grant.Terms.Scope.WorkspaceID != EmptyWorkspace {
			t.Fatal("activation rollback changed or invented integration grants")
		}
		switch {
		case grant.ClientID == ConsoleClient && grant.SubjectID == "":
			if grant.ID != integrationGrantID+string(grant.Terms.Action)+"-console" || clients[grant.Terms.Action] {
				t.Fatal("activation client integration grant identity changed")
			}
			clients[grant.Terms.Action] = true
		case grant.SubjectID == EmptyAdmin && grant.ClientID == "":
			if !wanted || grant.ID != integrationGrantID+string(grant.Terms.Action)+"-"+EmptyAdmin || subjects[grant.Terms.Action] {
				t.Fatal("activation retained unexpected subject integration authority")
			}
			subjects[grant.Terms.Action] = true
		default:
			t.Fatal("activation integration grant has another principal")
		}
	}
	if len(clients) != 8 || len(subjects) != 4 {
		t.Fatal("activation failed to preserve the exact partial grant set")
	}
	for action, wanted := range expected {
		if !clients[action] || subjects[action] != wanted {
			t.Fatal("activation partial authority differs from explicit expected rights")
		}
	}
	before, err := f.s.identity.Installation().Users("", f.s.cfg.Identity.PageSize)
	f.must(err)
	if len(before) != 1 || before[0].ID != EmptyAdmin {
		t.Fatal("actual first administrator identity premise")
	}
	f.code(t, "activation token is consumed", f.auth("redeem", activationBody(token, "second@example.invalid"), f.pre()), 422, CodeLinkInvalid)
	after, err := f.s.identity.Installation().Users("", f.s.cfg.Identity.PageSize)
	f.must(err)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("consumed activation token changed registered users")
	}
	got := f.auth("login", `{"workspace":"`+EmptyWorkspace+`","login":"`+login+`","password":"`+pw+`"}`, f.pre())
	if got.status != 200 || got.body["user"] != EmptyAdmin {
		t.Fatal("partial integration rollback prevented actual password sign-in")
	}
	caller := actor{}
	for _, cookie := range readCookies(got.header) {
		if cookie.Name == CookieName {
			caller = actor{cookie: cookie, csrf: got.body["csrf_token"].(string)}
		}
	}
	if caller.cookie == nil || f.me(caller).status != 200 {
		t.Fatal("activated administrator cannot use its actual authenticated session")
	}
	integrations := f.integrations(caller)
	rights, ok := integrations.body["rights"].(map[string]any)
	if integrations.status != 200 || !ok || len(rights) != 8 {
		t.Fatal("partial integration authority lacks its exact rights map")
	}
	for action, wanted := range expected {
		if rights[string(action)] != wanted {
			t.Fatal("installation administrator inherited an absent integration right")
		}
	}
	if areaState(integrations, "clients") != "shown" || areaState(integrations, "modules") != "shown" || areaState(integrations, "channels") != "withheld" || areaState(integrations, "events") != "withheld" {
		t.Fatal("integration areas do not reflect retained and withheld activation rights")
	}
}
