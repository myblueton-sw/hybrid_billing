package identity

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

func (f fixture) indexed(t *testing.T, label string) {
	t.Helper()
	f.r.mu.Lock()
	defer f.r.mu.Unlock()
	users := []string{}
	for id := range f.r.users {
		users = append(users, id)
	}
	sort.Strings(users)
	if !reflect.DeepEqual(users, append([]string{}, f.r.userIndex...)) {
		t.Errorf("%s: user index %v; users %v", label, f.r.userIndex, users)
	}
	for workspace, members := range f.r.memberships {
		ids := []string{}
		for id := range members {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		if !reflect.DeepEqual(ids, append([]string{}, f.r.memberIndex[workspace]...)) {
			t.Errorf("%s: member index of %s %v; memberships %v", label, workspace, f.r.memberIndex[workspace], ids)
		}
	}
	if len(f.r.memberIndex) != len(f.r.memberships) {
		t.Errorf("%s: %d member indexes for %d workspaces", label, len(f.r.memberIndex), len(f.r.memberships))
	}
}

func (f fixture) versions() (uint64, uint64) {
	f.r.mu.Lock()
	defer f.r.mu.Unlock()
	return f.r.usersVersion, f.r.membersVersion["w1"]
}

func TestIndexesFollowEveryInsert(t *testing.T) {
	f := empty(t, testLimits)
	f.indexed(t, "empty")
	f.must(f.in.ArmBootstrap(v("boot"), Validity{NotBefore: start, NotAfter: start.Add(time.Hour)}, "w1"))
	f.must(f.b.Bootstrap(v("boot"), "root", triple("root"), "root@example.invalid"))
	f.indexed(t, "bootstrap")
	if users, members := f.versions(); users != 1 || members != 1 {
		t.Errorf("versions after bootstrap %d %d", users, members)
	}
	f.user("cat")
	f.user("ann")
	f.indexed(t, "create user")
	if users, members := f.versions(); users != 3 || members != 1 {
		t.Errorf("versions after creating users %d %d", users, members)
	}
	admin := f.login("root")
	f.must(f.b.EndSession(admin))
	if users, members := f.versions(); users != 3 || members != 1 {
		t.Errorf("a session changed the versions %d %d", users, members)
	}
	f.join("ann", Member)
	f.indexed(t, "accept")
	if users, members := f.versions(); users != 3 || members != 2 {
		t.Errorf("versions after accepting %d %d", users, members)
	}
	f.must(f.w.CreateServiceUser(f.login("root"), "w1", "bot", "ann"))
	f.indexed(t, "service user")
	if users, _ := f.versions(); users != 4 {
		t.Errorf("version after a service user %d", users)
	}
	f.must(f.w.RemoveMembership(f.login("root"), "w1", "ann", 1))
	before, _ := f.versions()
	f.join("ann", Administrator)
	f.indexed(t, "accept after removal")
	if users, members := f.versions(); users != before || members != 3 {
		t.Errorf("versions after a second acceptance %d %d", users, members)
	}
	if f.r.memberIndex["w1"][0] != "ann" || len(f.r.memberIndex["w1"]) != 2 {
		t.Errorf("a returning member was indexed twice: %v", f.r.memberIndex["w1"])
	}
	f.seed("w2", "adm2")
	f.indexed(t, "second workspace")
	if got := f.r.memberIndex["w2"]; !reflect.DeepEqual(got, []string{"adm2"}) {
		t.Errorf("second workspace index %v", got)
	}
}

func TestInsertSortedKeepsOrderAndSkipsDuplicates(t *testing.T) {
	ids := []string{}
	for _, id := range []string{"m", "c", "x", "c", "a", "m", "z"} {
		ids = insertSorted(ids, id)
	}
	if !reflect.DeepEqual(ids, []string{"a", "c", "m", "x", "z"}) {
		t.Errorf("sorted insert %v", ids)
	}
}

func TestMembersSkipsRemovedMembershipsWithoutShorteningAPage(t *testing.T) {
	f := setup(t)
	for _, id := range []string{"ann", "bob", "cat", "dan"} {
		f.member(id, Member)
	}
	f.must(f.w.RemoveMembership(f.login("root"), "w1", "bob", 1))
	f.must(f.w.RemoveMembership(f.login("root"), "w1", "cat", 1))
	first, err := f.w.Members(f.login("root"), "w1", "", 2)
	f.must(err)
	if len(first) != 2 || first[0].User != "ann" || first[1].User != "dan" {
		t.Errorf("first page %+v", first)
	}
	rest, err := f.w.Members(f.login("root"), "w1", "dan", 2)
	f.must(err)
	if len(rest) != 1 || rest[0].User != "root" {
		t.Errorf("second page %+v", rest)
	}
	users, err := f.in.Users("bob", 2)
	f.must(err)
	if len(users) != 2 || users[0].ID != "cat" || users[1].ID != "dan" {
		t.Errorf("users after a cursor %+v", users)
	}
}

func TestPeople(t *testing.T) {
	f := setup(t)
	f.member("ann", Member)
	f.bounded("bob", Administrator, start.Add(time.Hour))
	f.user("cat")
	f.member("dan", Member)
	f.must(f.w.RemoveMembership(f.login("root"), "w1", "dan", 1))
	f.seed("w2", "adm2")
	admin := f.login("root")
	got, err := f.w.People(admin, "w1", []string{"ann", "ghost", "bob", "cat", "dan", "adm2"})
	f.must(err)
	names := []string{}
	for _, p := range got {
		names = append(names, p.User.ID)
	}
	if !reflect.DeepEqual(names, []string{"ann", "bob", "cat", "dan", "adm2"}) {
		t.Fatalf("people %v", names)
	}
	if got[0].Member == nil || got[0].Member.Role != Member || !got[0].Member.Effective || got[0].User.Kind != Human || got[0].User.Epoch != 1 {
		t.Errorf("ann %+v %+v", got[0].User, got[0].Member)
	}
	if got[1].Member == nil || !got[1].Member.Until.Equal(start.Add(time.Hour)) || !got[1].Member.Effective {
		t.Errorf("bob %+v", got[1].Member)
	}
	if got[2].Member != nil || got[3].Member != nil || got[4].Member != nil {
		t.Errorf("no membership here: %+v %+v %+v", got[2].Member, got[3].Member, got[4].Member)
	}
	there, err := f.w.People(f.login("adm2"), "w2", []string{"adm2", "ann"})
	f.must(err)
	if len(there) != 2 || there[0].Member == nil || there[0].Member.Role != Administrator || there[1].Member != nil {
		t.Errorf("the other workspace %+v", there)
	}
	f.at(start.Add(time.Hour))
	ended, err := f.w.People(f.login("root"), "w1", []string{"bob"})
	f.must(err)
	if ended[0].Member == nil || ended[0].Member.Effective || ended[0].Member.State != MembershipActive {
		t.Errorf("at the period end %+v", ended[0].Member)
	}
	f.at(start)
	full := make([]string, PeopleLimit)
	for i := range full {
		full[i] = fmt.Sprintf("u%d", i)
	}
	if got, err := f.w.People(admin, "w1", full); err != nil || len(got) != 0 {
		t.Errorf("a full list of unknown people %d %v", len(got), err)
	}
	f.want("too many", second(f.w.People(admin, "w1", append(full, "one"))), ErrInvalid)
	f.want("a bad workspace", second(f.w.People(admin, "bad id", nil)), ErrInvalid)
	f.want("a bad person", second(f.w.People(admin, "w1", []string{"ann", ""})), ErrInvalid)
	f.refusal("a member", second(f.w.People(f.login("ann"), "w1", []string{"ann"})), ReasonNotAdministrator)
	f.refusal("an administrator of another workspace", second(f.w.People(f.login("adm2"), "w1", []string{"ann"})), ReasonNotAdministrator)
	f.refusal("no session", second(f.w.People(v("nobody"), "w1", []string{"ann"})), ReasonSession)
	var missing *Workspace
	f.want("no registry", second(missing.People(admin, "w1", nil)), ErrUnavailable)
}

func TestUserLookup(t *testing.T) {
	f := setup(t)
	f.member("ann", Member)
	f.must(f.w.CreateServiceUser(f.login("root"), "w1", "bot", "ann"))
	if got, ok, err := f.in.User("ann"); err != nil || !ok || got != (UserInfo{ID: "ann", Kind: Human, State: Active, Epoch: 1}) {
		t.Errorf("ann %+v %v %v", got, ok, err)
	}
	if got, ok, err := f.in.User("bot"); err != nil || !ok || got.Kind != Service || got.Workspace != "w1" {
		t.Errorf("bot %+v %v %v", got, ok, err)
	}
	if got, ok, err := f.in.User("ghost"); err != nil || ok || got != (UserInfo{}) {
		t.Errorf("ghost %+v %v %v", got, ok, err)
	}
	var missing *Installation
	if _, ok, err := missing.User("ann"); ok || !errors.Is(err, ErrUnavailable) {
		t.Errorf("no registry %v %v", ok, err)
	}
	if _, err := f.in.Users("", 0); !errors.Is(err, ErrInvalid) {
		t.Errorf("an empty page %v", err)
	}
	if _, err := missing.Users("", 1); !errors.Is(err, ErrUnavailable) {
		t.Errorf("no registry for the users list %v", err)
	}
}

func sortedUsers(r *Registry, after string, limit int) []UserInfo {
	ids := []string{}
	for id := range r.users {
		ids = append(ids, id)
	}
	out := []UserInfo{}
	for _, id := range page(ids, after, limit) {
		out = append(out, userInfo(id, r.users[id]))
	}
	return out
}

func sortedMembers(r *Registry, workspace, after string, limit int) []MemberInfo {
	ids := []string{}
	for id := range r.memberships[workspace] {
		if r.member(workspace, id) != nil {
			ids = append(ids, id)
		}
	}
	out := []MemberInfo{}
	for _, id := range page(ids, after, limit) {
		out = append(out, r.memberInfo(id, r.member(workspace, id), r.clock()))
	}
	return out
}

func TestListingsMatchTheSortingOracle(t *testing.T) {
	for seed := int64(1); seed <= 30; seed++ {
		f := setupWith(t, Limits{BootstrapValidity: time.Hour, InvitationValidity: 72 * time.Hour, CredentialValidity: 720 * time.Hour,
			SessionIdle: 30 * time.Minute, SessionAbsolute: 8 * time.Hour, Reauthentication: 5 * time.Minute,
			Attempts: 3, PendingInvitations: 100, SessionsPerUser: 1000, CredentialsPerServiceUser: 2, PageSize: 7})
		state := uint64(seed)*2654435761 + 11
		next := func(n int) int {
			state = state*6364136223846793005 + 1442695040888963407
			return int((state >> 33) % uint64(n))
		}
		people := []string{}
		for step := 0; step < 60; step++ {
			switch next(4) {
			case 0, 1:
				id := fmt.Sprintf("p%03d", next(500))
				if f.in.CreateUser(operator, id, triple(id), id+"@example.invalid") == nil {
					people = append(people, id)
				}
			case 2:
				if len(people) > 0 {
					id := people[next(len(people))]
					invitation := f.next("invitation")
					if f.w.Invite(f.login("root"), "w1", invitation, Member, v(invitation), f.window(), id) == nil {
						_ = f.w.Accept("w1", invitation, v(invitation), f.login(id))
					}
				}
			case 3:
				if len(people) > 0 {
					id := people[next(len(people))]
					if found := f.r.member("w1", id); found != nil {
						_ = f.w.RemoveMembership(f.login("root"), "w1", id, found.version)
					}
				}
			}
		}
		f.indexed(t, fmt.Sprintf("seed %d", seed))
		admin := f.login("root")
		cursors := []string{""}
		for i := 0; i < 520; i += 37 {
			cursors = append(cursors, fmt.Sprintf("p%03d", i))
		}
		for _, after := range cursors {
			for _, limit := range []int{1, 3, 7} {
				got, err := f.in.Users(after, limit)
				f.must(err)
				if want := sortedUsers(f.r, after, limit); !reflect.DeepEqual(got, want) {
					t.Fatalf("seed %d users after %q limit %d: %v; want %v", seed, after, limit, got, want)
				}
				members, err := f.w.Members(admin, "w1", after, limit)
				f.must(err)
				if want := sortedMembers(f.r, "w1", after, limit); !reflect.DeepEqual(members, want) {
					t.Fatalf("seed %d members after %q limit %d: %v; want %v", seed, after, limit, members, want)
				}
			}
		}
	}
}

func TestConcurrentReadersOfAGrowingIndex(t *testing.T) {
	f := setup(t)
	admin := f.login("root")
	var wg sync.WaitGroup
	for reader := 0; reader < 4; reader++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				_, _ = f.in.Users("", 2)
				_, _ = f.w.Members(admin, "w1", "", 2)
				_, _ = f.w.People(admin, "w1", []string{"root", "u1"})
				_, _, _ = f.in.User("u1")
			}
		}()
	}
	for i := 0; i < 50; i++ {
		id := fmt.Sprintf("u%d", i)
		f.must(f.in.CreateUser(operator, id, triple(id), id+"@example.invalid"))
	}
	wg.Wait()
	f.indexed(t, "after concurrent readers")
}

var indexHelpers = map[string]bool{"putUser": true, "putMembership": true}

func TestRegistryMapsAreWrittenOnlyThroughTheIndexHelpers(t *testing.T) {
	files, _ := filepath.Glob("*.go")
	written := 0
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			ast.Inspect(fn, func(n ast.Node) bool {
				targets := []ast.Expr{}
				switch node := n.(type) {
				case *ast.AssignStmt:
					targets = node.Lhs
				case *ast.CallExpr:
					if id, ok := node.Fun.(*ast.Ident); ok && id.Name == "delete" && len(node.Args) > 0 {
						targets = node.Args[:1]
					}
				}
				for _, target := range targets {
					if field := registryMap(target); field != "" {
						written++
						if !indexHelpers[fn.Name.Name] {
							t.Errorf("%s writes r.%s in %s", name, field, fn.Name.Name)
						}
					}
				}
				return true
			})
		}
	}
	if written != 3 {
		t.Errorf("the helpers write the maps %d times", written)
	}
}

func registryMap(target ast.Expr) string {
	for {
		switch node := target.(type) {
		case *ast.IndexExpr:
			target = node.X
			continue
		case *ast.SelectorExpr:
			if node.Sel.Name == "users" || node.Sel.Name == "memberships" {
				return node.Sel.Name
			}
		}
		return ""
	}
}
