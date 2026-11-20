package identity

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"testing"
	"time"
)

func assertCredentialExpiryOracle(t testing.TB, r *Registry) {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	oracle := map[string][]credentialExpiry{}
	for id, found := range r.credentials {
		if !found.revoked {
			oracle[found.user] = append(oracle[found.user], credentialExpiry{id, found.validity.NotAfter})
		}
	}
	for key := range oracle {
		sort.Slice(oracle[key], func(i, j int) bool {
			a, b := oracle[key][i], oracle[key][j]
			return a.until.Before(b.until) || a.until.Equal(b.until) && a.id < b.id
		})
	}
	if !reflect.DeepEqual(oracle, r.credentialExpiry) {
		t.Fatal("expiry index differs from independent stored-map oracle")
	}
}

func assertServiceLiveOracle(t testing.TB, f fixture, actor Verifier, workspace, service string) {
	t.Helper()
	want := 0
	f.r.mu.Lock()
	for _, found := range f.r.credentials {
		if found.user == service && !found.revoked && f.now.Before(found.validity.NotAfter) {
			want++
		}
	}
	f.r.mu.Unlock()
	controls, err := f.w.ServiceControls(actor, workspace, service)
	if err != nil || controls.LiveCredentials != want || controls.CredentialLimit != f.r.limits.CredentialsPerServiceUser {
		t.Fatal("live capacity differs from independent recorded lifecycle", controls, want, err)
	}
}

func TestServiceControlsExpiryIndexPreservesFutureCapacityTiesPruningAndReuse(t *testing.T) {
	limits := testLimits
	limits.CredentialsPerServiceUser = 3
	f := setupWith(t, limits)
	actor := f.login("root")
	f.must(f.w.CreateServiceUser(actor, "w1", "svc", "root"))
	for id, validity := range map[string]Validity{"future": {start.Add(20 * time.Minute), start.Add(time.Hour)}, "tie-a": {start, start.Add(time.Minute)}, "tie-b": {start, start.Add(time.Minute)}} {
		f.must(f.w.AddCredential(actor, "w1", "svc", id, v(id), validity))
		assertCredentialExpiryOracle(t, f.r)
		assertServiceLiveOracle(t, f, actor, "w1", "svc")
	}
	if err := f.w.AddCredential(actor, "w1", "svc", "over", v("over"), Validity{start, start.Add(time.Hour)}); err != ErrLimit {
		t.Fatal("future validity stopped consuming exact capacity", err)
	}
	*f.now = start.Add(time.Minute)
	assertServiceLiveOracle(t, f, actor, "w1", "svc")
	controls, err := f.w.ServiceControls(actor, "w1", "svc")
	f.must(err)
	if controls.LiveCredentials != 1 {
		t.Fatal("NotAfter equality remained live", controls.LiveCredentials)
	}
	f.must(f.w.AddCredential(actor, "w1", "svc", "new", v("new"), Validity{*f.now, f.now.Add(time.Hour)}))
	f.must(f.w.RevokeCredential(actor, "w1", "future"))
	assertCredentialExpiryOracle(t, f.r)
	assertServiceLiveOracle(t, f, actor, "w1", "svc")
	for i := 0; i < EndedCredentialsKept+5; i++ {
		id := fmt.Sprintf("ended-%03d", i)
		f.must(f.w.AddCredential(actor, "w1", "svc", id, v(id), Validity{*f.now, f.now.Add(time.Hour)}))
		f.must(f.w.RevokeCredential(actor, "w1", id))
		assertCredentialExpiryOracle(t, f.r)
		assertServiceLiveOracle(t, f, actor, "w1", "svc")
	}
	_, _, err = f.w.Credential(actor, "w1", "tie-a")
	f.refusal("pruned credential", err, ReasonUnknown)
	f.must(f.w.AddCredential(actor, "w1", "svc", "tie-a", v("reused"), Validity{*f.now, f.now.Add(time.Hour)}))
	assertCredentialExpiryOracle(t, f.r)
	assertServiceLiveOracle(t, f, actor, "w1", "svc")
}

func TestServiceControlsExpiryReplacementAndCurrentAuthority(t *testing.T) {
	f := setup(t)
	f.member("ann", Member)
	f.seed("w2", "adm2")
	actor, other, member := f.login("root"), f.login("adm2"), f.login("ann")
	f.must(f.w.CreateServiceUser(actor, "w1", "one", "root"))
	f.must(f.w.CreateServiceUser(other, "w2", "two", "adm2"))
	f.must(f.w.AddCredential(actor, "w1", "one", "moving", v("moving"), Validity{start, start.Add(time.Hour)}))
	f.r.mu.Lock()
	replacement := *f.r.credentials["moving"]
	replacement.user = "two"
	replacement.validity.NotAfter = start.Add(2 * time.Hour)
	f.r.putCredential("moving", &replacement)
	f.r.mu.Unlock()
	assertCredentialExpiryOracle(t, f.r)
	assertServiceLiveOracle(t, f, actor, "w1", "one")
	assertServiceLiveOracle(t, f, other, "w2", "two")
	f.must(f.in.SuspendUser(operator, "one", 1))
	controls, err := f.w.ServiceControls(actor, "w1", "one")
	f.must(err)
	if !controls.OperatorBlocked || !controls.CanTransition || !controls.CanAllocateCredential {
		t.Fatal("operator/allocator state is not authoritative", controls)
	}
	_, err = f.w.ServiceControls(member, "w1", "one")
	f.refusal("member controls", err, ReasonNotAdministrator)
	_, err = f.w.ServiceControls(other, "w2", "one")
	f.refusal("foreign controls", err, ReasonUnknown)
	_, err = f.w.ServiceControls(actor, "w1", "missing")
	f.refusal("missing controls", err, ReasonUnknown)
	_, err = f.w.ServiceControls(actor, "bad workspace", "one")
	f.want("malformed controls", err, ErrInvalid)
	f.r.mu.Lock()
	f.r.serial = math.MaxUint64
	f.r.users["one"].epoch = math.MaxUint64
	f.r.mu.Unlock()
	controls, err = f.w.ServiceControls(actor, "w1", "one")
	f.must(err)
	if controls.CanTransition || controls.CanAllocateCredential {
		t.Fatal("allocator/version exhaustion advertised available")
	}
	f.must(f.b.EndSession(actor))
	_, err = f.w.ServiceControls(actor, "w1", "one")
	f.refusal("ended controls caller", err, ReasonSession)
	var absent *Workspace
	_, err = absent.ServiceControls(actor, "w1", "one")
	f.want("unavailable controls", err, ErrUnavailable)
}
