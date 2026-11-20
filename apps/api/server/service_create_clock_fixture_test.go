package server

import (
	"io"
	"log/slog"
	"reflect"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

type createBoundaryState struct {
	count        int
	service      identity.ServiceInfo
	credentials  []identity.CredentialInfo
	declaration  serviceDeclaration
	collected    delivery
	collections  int
	pickups      int
	delivery     uint64
	declarations int
	revision     string
}

type createBoundaryFixture struct {
	f          world
	clock      *detailBoundaryClock
	verifier   identity.Verifier
	service    string
	credential string
	proposed   string
	before     createBoundaryState
}

func (d createBoundaryFixture) state(verifier identity.Verifier) createBoundaryState {
	t := d.f.t
	t.Helper()
	count, err := d.f.s.identity.Workspace().ServiceCount(verifier, SampleWorkspace)
	d.f.must(err)
	service, err := d.f.s.identity.Workspace().ServiceUser(verifier, SampleWorkspace, d.service)
	d.f.must(err)
	credentials, err := d.f.s.identity.Workspace().Credentials(verifier, SampleWorkspace, d.service)
	d.f.must(err)
	declaration, known := d.f.s.serviceDeclaration(SampleWorkspace, d.service, 0)
	if !known {
		t.Fatal("actual retained declaration premise")
	}
	collected, known := d.f.s.dir.collected[d.credential]
	if !known || !collected.collected || collected.service != d.service {
		t.Fatal("genuine collected credential marker premise")
	}
	return createBoundaryState{count, service, credentials, declaration, collected, len(d.f.s.dir.collected), len(d.f.s.dir.pickups), d.f.s.dir.deliveryVersion, len(d.f.s.serviceDeclarations), d.f.s.Revision()}
}

func newCreateBoundaryFixture(t *testing.T) createBoundaryFixture {
	t.Helper()
	clock := &detailBoundaryClock{}
	origin := time.Now()
	f := setupWith(t, func(cfg *Config) {
		cfg.Clock = clock.now
		cfg.Elapsed = func() time.Duration { return time.Since(origin) }
		cfg.Log = slog.New(slog.NewTextHandler(io.Discard, nil))
	})
	f.setPassword(AdminLogin, pw)
	caller := f.signedIn(AdminLogin, pw)
	service := f.newService(caller, "create-clock-existing")
	credential, ref, _ := f.issued(caller, service, "1")
	if f.collect(caller, ref).status != 200 {
		t.Fatal("actual original-session credential collection premise")
	}
	d := createBoundaryFixture{f: f, clock: clock, verifier: caller.verifier(f), service: service, credential: credential, proposed: f.s.Next()}
	d.before = d.state(d.verifier)
	if d.before.count != 1 || d.before.pickups != 0 || d.before.collections != 1 || len(d.before.credentials) != 1 || d.before.declaration.Version != 1 {
		t.Fatal("actual populated collected-resource premise")
	}
	found := capturedAdmissionSession(f, caller)
	future := found.created.Add(sessionAbsolute + time.Minute)
	clock.mu.Lock()
	clock.future = future
	clock.mu.Unlock()
	if !future.After(found.created.Add(sessionAbsolute)) || future.Sub(f.s.clock()) >= f.s.time.tolerance {
		t.Fatal("absolute-expiry jump within wall tolerance premise")
	}
	if err := f.s.Close(); err != nil {
		t.Fatal("owned create fixture failed to quiesce")
	}
	if _, _, err := f.s.identity.Boundary().Peek(d.verifier, SampleWorkspace); err != nil {
		t.Fatal("quiescence unexpectedly revoked module authentication")
	}
	return d
}

func (d createBoundaryFixture) apply(cutoff int) (commands.Outcome, error, int, bool) {
	f := d.f
	fields := f.declaredFields(map[string]string{"name": "create-clock-new", "owner": SampleAdmin})
	f.s.domain.Lock()
	previousCaller, previousFault := f.s.caller, f.s.fault
	defer func() {
		f.s.caller, f.s.fault = previousCaller, previousFault
		f.s.domain.Unlock()
	}()
	f.s.caller = d.verifier
	d.clock.arm(cutoff)
	outcome, err := (serviceCommand{s: f.s, kind: "create"}).Apply(commands.Context{Target: servicesTarget, Fields: fields})
	observations, jumped := d.clock.finish()
	return outcome, err, observations, jumped
}

func (d createBoundaryFixture) observe(created bool) {
	t := d.f.t
	t.Helper()
	cookie, csrf, err := d.f.s.openFor(LoginTriple(Admin2Login), SampleWorkspace, false)
	d.f.must(err)
	observer := actor{cookie: cookie, csrf: csrf}.verifier(d.f)
	after := d.state(observer)
	service, err := d.f.s.identity.Workspace().ServiceUser(observer, SampleWorkspace, "svc-create-clock-new")
	declaration, known := d.f.s.serviceDeclaration(SampleWorkspace, "svc-create-clock-new", 0)
	if created {
		d.f.must(err)
		if service.ID != "svc-create-clock-new" || service.Owner != SampleAdmin || service.Creator != SampleAdmin || service.State != identity.Active || service.Epoch != 1 || !known || declaration.Version != 1 || declaration.Organization != "sample-le-seller" || declaration.Purpose != "Synthetic service processing" {
			t.Fatal("healthy direct create lost canonical identity or declaration")
		}
		credentials, err := d.f.s.identity.Workspace().Credentials(observer, SampleWorkspace, service.ID)
		d.f.must(err)
		if len(credentials) != 0 || after.count != d.before.count+1 || after.declarations != d.before.declarations+1 {
			t.Fatal("healthy direct create changed more than one identity and declaration")
		}
		after.count, after.declarations = d.before.count, d.before.declarations
	} else if identity.ReasonOf(err) != identity.ReasonUnknown || known || after.count != d.before.count || after.declarations != d.before.declarations {
		t.Fatal("expired direct create left a service or declaration")
	}
	if !reflect.DeepEqual(d.before, after) {
		t.Fatal("create clock boundary changed existing collected resources or committed revision")
	}
}
