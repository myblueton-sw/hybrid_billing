package server

import (
	"io"
	"log/slog"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

type detailBoundaryClock struct {
	mu           sync.Mutex
	future       time.Time
	armed        bool
	cutoff       int
	observations int
	jumped       bool
}

func (c *detailBoundaryClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.armed {
		c.observations++
		if c.cutoff > 0 && c.observations >= c.cutoff {
			c.jumped = true
		}
	}
	if c.jumped {
		return c.future
	}
	return start
}

func (c *detailBoundaryClock) arm(cutoff int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cutoff, c.observations, c.armed = cutoff, 0, true
}

func (c *detailBoundaryClock) finish() (int, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.armed = false
	return c.observations, c.jumped
}

type detailBoundaryState struct {
	count        int
	service      identity.ServiceInfo
	credentials  []identity.CredentialInfo
	declaration  serviceDeclaration
	pending      pickup
	pickups      int
	delivery     uint64
	declarations int
	revision     string
}

type detailBoundaryFixture struct {
	f          world
	clock      *detailBoundaryClock
	caller     actor
	service    string
	credential string
	kind       string
	before     detailBoundaryState
}

func (d detailBoundaryFixture) state(a actor) detailBoundaryState {
	t := d.f.t
	t.Helper()
	v := a.verifier(d.f)
	count, err := d.f.s.identity.Workspace().ServiceCount(v, SampleWorkspace)
	d.f.must(err)
	service, err := d.f.s.identity.Workspace().ServiceUser(v, SampleWorkspace, d.service)
	d.f.must(err)
	credentials, err := d.f.s.identity.Workspace().Credentials(v, SampleWorkspace, d.service)
	d.f.must(err)
	declaration, known := d.f.s.serviceDeclaration(SampleWorkspace, d.service, 0)
	if !known {
		t.Fatal("actual retained declaration premise")
	}
	pending, known := d.f.s.dir.credentialPickup(d.credential)
	if !known {
		t.Fatal("actual original credential pickup premise")
	}
	return detailBoundaryState{count, service, credentials, declaration, pending, len(d.f.s.dir.pickups), d.f.s.dir.deliveryVersion, len(d.f.s.serviceDeclarations), d.f.s.Revision()}
}

func newDetailBoundaryFixture(t *testing.T, kind string) detailBoundaryFixture {
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
	service := f.newService(caller, "clock-boundary")
	credential, _, _ := f.issued(caller, service, "1")
	d := detailBoundaryFixture{f: f, clock: clock, caller: caller, service: service, credential: credential, kind: kind}
	d.before = d.state(caller)
	if d.before.count != 1 || len(d.before.credentials) != 1 || d.before.declaration.Version != 1 {
		t.Fatal("actual populated detail fixture premise")
	}
	found := capturedAdmissionSession(f, caller)
	future := found.created.Add(sessionAbsolute + time.Minute)
	clock.mu.Lock()
	clock.future = future
	clock.mu.Unlock()
	if !future.After(found.created.Add(sessionAbsolute)) || future.Sub(f.s.clock()) >= f.s.time.tolerance {
		t.Fatal("absolute-expiry jump within wall tolerance premise")
	}
	if got, _, _ := d.read(); got.status != 200 {
		t.Fatal("actual populated detail failed before quiescence", got.status)
	}
	if err := f.s.Close(); err != nil {
		t.Fatal("owned detail fixture failed to quiesce")
	}
	if _, _, err := f.s.identity.Boundary().Peek(found.verifier, SampleWorkspace); err != nil {
		t.Fatal("quiescence unexpectedly revoked module authentication")
	}
	return d
}

func (d detailBoundaryFixture) read() (reply, int, bool) {
	t := d.f.t
	t.Helper()
	id, path := d.service, "/api/v1/identity/services/"+d.service
	if d.kind == "credential" {
		id, path = d.credential, "/api/v1/identity/service-credentials/"+d.credential
	}
	r := serviceWalkRequest(d.caller, path)
	r.SetPathValue("id", id)
	w := httptest.NewRecorder()
	if d.kind == "credential" {
		d.f.s.readServiceCredential(w, r)
	} else {
		d.f.s.readServiceAccount(w, r)
	}
	observations, jumped := d.clock.finish()
	return serviceQueryRecordedFailure(t, w), observations, jumped
}

func (d detailBoundaryFixture) unchanged() {
	t := d.f.t
	t.Helper()
	cookie, csrf, err := d.f.s.openFor(LoginTriple(Admin2Login), SampleWorkspace, false)
	d.f.must(err)
	after := d.state(actor{cookie: cookie, csrf: csrf})
	if !reflect.DeepEqual(d.before, after) {
		t.Fatal("detail clock boundary changed recorded service data or pickup identity")
	}
}
