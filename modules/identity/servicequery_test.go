package identity

import (
	"fmt"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"
)

func assertServiceQueryIndexes(t *testing.T, r *Registry) {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	services, credentials := map[string][]string{}, map[string][]string{}
	for id, found := range r.users {
		if found.kind == Service {
			services[found.workspace] = append(services[found.workspace], id)
		}
	}
	for id, found := range r.credentials {
		credentials[found.user] = append(credentials[found.user], id)
	}
	for _, index := range []map[string][]string{services, credentials} {
		for key := range index {
			sort.Strings(index[key])
		}
	}
	if !reflect.DeepEqual(services, r.serviceIndex) {
		t.Fatalf("service index differs from stored workspace membership: %v / %v", r.serviceIndex, services)
	}
	if !reflect.DeepEqual(credentials, r.credentialIndex) {
		t.Fatalf("credential index differs from stored ownership: %v / %v", r.credentialIndex, credentials)
	}
}

func serviceQueryOracle(r *Registry, workspace, after string, limit int) []ServiceInfo {
	ids := []string{}
	for id, found := range r.users {
		if found.kind == Service && found.workspace == workspace && id > after {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	out := []ServiceInfo{}
	for _, id := range ids[:min(limit, len(ids))] {
		found := r.users[id]
		out = append(out, ServiceInfo{ID: id, State: found.state, Epoch: found.epoch, Creator: found.creator, Owner: found.owner})
	}
	return out
}

func TestServiceQueryPagesAndKnownIDFollowLifecycle(t *testing.T) {
	f := setup(t)
	f.member("ann", Member)
	f.seed("w2", "adm2")
	admin, other := f.login("root"), f.login("adm2")
	for _, id := range []string{"svc-z", "svc-c", "svc-q", "svc-a", "svc-n", "svc-b", "svc-m"} {
		f.must(f.w.CreateServiceUser(admin, "w1", id, "ann"))
		assertServiceQueryIndexes(t, f.r)
	}
	f.must(f.w.CreateServiceUser(other, "w2", "svc-foreign", "adm2"))
	f.must(f.w.SuspendServiceUser(admin, "w1", "svc-c", 1))
	f.must(f.w.RetireServiceUser(admin, "w1", "svc-q", 1))
	for _, after := range []string{"", "svc-a", "svc-h", "svc-q", "svc-z", "zzz"} {
		got, err := f.w.ServiceUsers(admin, "w1", after, 2)
		f.must(err)
		if want := serviceQueryOracle(f.r, "w1", after, 2); !reflect.DeepEqual(got, want) {
			t.Fatalf("page after %q: got %+v want %+v", after, got, want)
		}
	}
	row, err := f.w.ServiceUser(admin, "w1", "svc-c")
	f.must(err)
	if row != (ServiceInfo{ID: "svc-c", State: Suspended, Epoch: 2, Creator: "root", Owner: "ann"}) {
		t.Fatalf("direct metadata: %+v", row)
	}
	row.ID = "changed-copy"
	again, err := f.w.ServiceUser(admin, "w1", "svc-c")
	f.must(err)
	if again.ID != "svc-c" {
		t.Fatal("caller modified stored service metadata")
	}
	assertServiceQueryIndexes(t, f.r)
}

func TestServiceQueryKnownIDAndCredentialPagesRecheckAuthority(t *testing.T) {
	f := setup(t)
	f.member("ann", Member)
	f.seed("w2", "adm2")
	admin, member, other := f.login("root"), f.login("ann"), f.login("adm2")
	f.must(f.w.CreateServiceUser(admin, "w1", "svc", "ann"))
	for _, id := range []string{"missing", "ann"} {
		_, err := f.w.ServiceUser(admin, "w1", id)
		f.refusal("unknown or non-service target", err, ReasonUnknown)
	}
	_, err := f.w.ServiceUser(other, "w2", "svc")
	f.refusal("another workspace target", err, ReasonUnknown)
	_, err = f.w.CredentialPage(other, "w2", "svc", "", 1)
	f.refusal("another workspace credentials", err, ReasonUnknown)
	for _, read := range []func() error{
		func() error { _, err := f.w.ServiceUser(member, "w1", "svc"); return err },
		func() error { _, err := f.w.CredentialPage(member, "w1", "svc", "", 1); return err },
	} {
		f.refusal("member cannot inspect service metadata", read(), ReasonNotAdministrator)
	}
	f.must(f.b.EndSession(admin))
	_, err = f.w.ServiceUser(admin, "w1", "svc")
	f.refusal("ended caller lookup", err, ReasonSession)
	_, err = f.w.CredentialPage(admin, "w1", "svc", "", 1)
	f.refusal("ended caller page", err, ReasonSession)
	assertServiceQueryIndexes(t, f.r)
}

func TestServiceQueryCredentialPagesPreserveRetentionAndAdmission(t *testing.T) {
	f := setup(t)
	admin := f.login("root")
	f.must(f.w.CreateServiceUser(admin, "w1", "svc", "root"))
	f.must(f.w.CreateServiceUser(admin, "w1", "other", "root"))
	window := Validity{NotBefore: start, NotAfter: start.Add(time.Hour)}
	f.must(f.w.AddCredential(admin, "w1", "other", "foreign", v("foreign"), window))
	for i := 24; i >= 0; i-- {
		id := fmt.Sprintf("cred-%02d", i)
		f.must(f.w.AddCredential(admin, "w1", "svc", id, v(id), window))
		f.must(f.w.RevokeCredential(admin, "w1", id))
		assertServiceQueryIndexes(t, f.r)
	}
	f.must(f.w.AddCredential(admin, "w1", "svc", "live-a", v("live-a"), window))
	f.must(f.w.AddCredential(admin, "w1", "svc", "live-b", v("live-b"), window))
	f.want("live credential admission", f.w.AddCredential(admin, "w1", "svc", "too-many", v("too-many"), window), ErrLimit)
	all, err := f.w.Credentials(admin, "w1", "svc")
	f.must(err)
	if len(all) != EndedCredentialsKept+2 || all[0].ID != "cred-00" || all[19].ID != "cred-19" {
		t.Fatalf("retention changed: %+v", all)
	}
	pages := []CredentialInfo{}
	after := ""
	for {
		rows, err := f.w.CredentialPage(admin, "w1", "svc", after, 2)
		f.must(err)
		if len(rows) == 0 {
			break
		}
		if len(rows) > 2 || rows[0].ID <= after {
			t.Fatal("credential page bound or order", rows)
		}
		pages = append(pages, rows...)
		after = rows[len(rows)-1].ID
	}
	if !reflect.DeepEqual(pages, all) {
		t.Fatal("credential pages lost, repeated or disclosed another service row")
	}
	rows, err := f.w.CredentialPage(admin, "w1", "svc", "cred-055", 2)
	f.must(err)
	if len(rows) != 2 || rows[0].ID != "cred-06" || rows[1].ID != "cred-07" {
		t.Fatalf("absent boundary page: %+v", rows)
	}
	f.at(start.Add(time.Hour))
	admin = f.login("root")
	f.must(f.w.AddCredential(admin, "w1", "svc", "new-live", v("new-live"), Validity{NotBefore: *f.now, NotAfter: f.now.Add(time.Hour)}))
	all, err = f.w.Credentials(admin, "w1", "svc")
	f.must(err)
	if len(all) != EndedCredentialsKept+1 || all[len(all)-1].ID != "new-live" {
		t.Fatal("expiry boundary failed to release live admission or prune ended credentials", all)
	}
	assertServiceQueryIndexes(t, f.r)
}

func TestServiceQueryValidationAndUnavailableHandles(t *testing.T) {
	f := setup(t)
	admin := f.login("root")
	f.must(f.w.CreateServiceUser(admin, "w1", "svc", "root"))
	for _, args := range []struct {
		after string
		limit int
	}{{"", 0}, {"", 3}, {"bad id", 1}} {
		_, err := f.w.CredentialPage(admin, "w1", "svc", args.after, args.limit)
		f.want("invalid credential page", err, ErrInvalid)
	}
	_, err := f.w.ServiceUser(admin, "w1", "")
	f.want("invalid direct service ID", err, ErrInvalid)
	for _, unavailable := range []*Workspace{nil, {}, {r: &Registry{}}} {
		_, err = unavailable.ServiceUser(admin, "w1", "svc")
		f.want("unavailable direct service", err, ErrUnavailable)
		_, err = unavailable.CredentialPage(admin, "w1", "svc", "", 1)
		f.want("unavailable credential page", err, ErrUnavailable)
	}
}

func TestServiceQueryIndexesRemainConsistentWithConcurrentReaders(t *testing.T) {
	f := setup(t)
	admin := f.login("root")
	f.must(f.w.CreateServiceUser(admin, "w1", "svc", "root"))
	var readers sync.WaitGroup
	for i := 0; i < 3; i++ {
		readers.Add(1)
		go func() {
			defer readers.Done()
			for n := 0; n < 30; n++ {
				_, _ = f.w.ServiceUsers(admin, "w1", "", 2)
				_, _ = f.w.ServiceUser(admin, "w1", "svc")
				_, _ = f.w.CredentialPage(admin, "w1", "svc", "", 2)
			}
		}()
	}
	for i := 0; i < 30; i++ {
		id := fmt.Sprintf("c%02d", i)
		f.must(f.w.AddCredential(admin, "w1", "svc", id, v(id), Validity{NotBefore: start, NotAfter: start.Add(time.Hour)}))
		f.must(f.w.RevokeCredential(admin, "w1", id))
	}
	readers.Wait()
	assertServiceQueryIndexes(t, f.r)
}

func TestServiceQueryFutureCredentialStillConsumesCapacity(t *testing.T) {
	f := setup(t)
	admin := f.login("root")
	f.must(f.w.CreateServiceUser(admin, "w1", "svc", "root"))
	window := Validity{NotBefore: start.Add(30 * time.Minute), NotAfter: start.Add(time.Hour)}
	for _, id := range []string{"future-a", "future-b"} {
		f.must(f.w.AddCredential(admin, "w1", "svc", id, v(id), window))
	}
	f.want("future credentials reserve live slots", f.w.AddCredential(admin, "w1", "svc", "extra", v("extra"), window), ErrLimit)
	if rows, err := f.w.CredentialPage(admin, "w1", "svc", "", 2); err != nil || len(rows) != 2 || rows[0].Validity.NotBefore != window.NotBefore {
		t.Fatalf("future credential metadata: %+v %v", rows, err)
	}
	assertServiceQueryIndexes(t, f.r)
}

func TestServiceQueryIndexWriterHandlesReplacement(t *testing.T) {
	f := setup(t)
	f.r.mu.Lock()
	f.r.putUser("svc", &user{kind: Service, workspace: "w1", state: Active, epoch: 1})
	f.r.mu.Unlock()
	assertServiceQueryIndexes(t, f.r)
	f.r.mu.Lock()
	f.r.putUser("svc", &user{kind: Service, workspace: "w2", state: Active, epoch: 2})
	f.r.putCredential("c", &credential{user: "svc"})
	f.r.putCredential("c", &credential{user: "other"})
	f.r.mu.Unlock()
	assertServiceQueryIndexes(t, f.r)
	f.r.mu.Lock()
	f.r.removeCredential("c")
	f.r.removeCredential("missing")
	f.r.putUser("svc", &user{kind: Human, state: Active, epoch: 3})
	f.r.mu.Unlock()
	assertServiceQueryIndexes(t, f.r)
}

func TestServiceCountUsesRecordedWorkspaceScopeAndCurrentAuthority(t *testing.T) {
	f := setup(t)
	f.member("ann", Member)
	f.seed("w2", "adm2")
	admin, other := f.login("root"), f.login("adm2")
	f.must(f.w.CreateServiceUser(admin, "w1", "one", "root"))
	f.must(f.w.CreateServiceUser(admin, "w1", "two", "root"))
	f.must(f.w.RetireServiceUser(admin, "w1", "one", 1))
	f.must(f.w.CreateServiceUser(other, "w2", "foreign", "adm2"))
	if count, err := f.w.ServiceCount(admin, "w1"); err != nil || count != 2 {
		t.Fatal("recorded count lost retired rows or crossed workspaces", count, err)
	}
	if count, err := f.w.ServiceCount(other, "w2"); err != nil || count != 1 {
		t.Fatal("foreign own count", count, err)
	}
	f.refusal("member count", second(f.w.ServiceCount(f.login("ann"), "w1")), ReasonNotAdministrator)
	f.must(f.b.EndSession(admin))
	f.refusal("ended count caller", second(f.w.ServiceCount(admin, "w1")), ReasonSession)
	f.want("invalid count workspace", second(f.w.ServiceCount(other, "")), ErrInvalid)
	var absent *Workspace
	f.want("unavailable count", second(absent.ServiceCount(other, "w2")), ErrUnavailable)
}
