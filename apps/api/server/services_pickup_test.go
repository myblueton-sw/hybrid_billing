package server

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func (f world) pickupOf(a actor, service, credential string) (string, string, bool) {
	f.t.Helper()
	row := f.service(a, service)
	if row == nil {
		f.t.Fatalf("service %s is not listed", service)
	}
	for _, c := range row["credentials"].([]any) {
		m := c.(map[string]any)
		if m["id"] == credential {
			state, has := m["secret_pickup"].(string)
			until, _ := m["secret_pickup_until"].(string)
			return state, until, has
		}
	}
	f.t.Fatalf("credential %s is not listed", credential)
	return "", "", false
}

func (f world) wantPickup(t testing.TB, name string, a actor, service, credential, want, until string) {
	t.Helper()
	state, end, has := f.pickupOf(a, service, credential)
	if want == "" && has || want != "" && state != want || end != until {
		t.Errorf("%s: secret_pickup %q (present %v) until %q, want %q until %q", name, state, has, end, want, until)
	}
}

func (f world) collectedEntries() int {
	f.s.domain.Lock()
	defer f.s.domain.Unlock()
	return len(f.s.dir.collected)
}

func (f world) stretch(ref string, until time.Time) {
	f.s.domain.Lock()
	p := f.s.dir.pickups[ref]
	p.expires = until
	f.s.dir.pickups[ref] = p
	f.s.domain.Unlock()
}

func TestPickupStateFollowsTheSecretPickup(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	svc := f.newService(root, "pickup")
	id, ref, _ := f.issued(root, svc, "30")
	f.wantPickup(t, "issued", root, svc, id, pickupOpen, stamp(start.Add(pickupValidity)))
	if raw := f.servicesRead(root).raw; strings.Contains(raw, ref) || strings.Contains(raw, secretPrefix) {
		t.Error("the read carries the pickup reference or a secret")
	}
	other := f.as(AdminLogin)
	f.wantPickup(t, "another administrator's read", other, svc, id, pickupOpen, stamp(start.Add(pickupValidity)))
	f.code(t, "another session collects", f.collect(other, ref), 403, string(commands.PermissionDenied))
	f.wantPickup(t, "after a refused collect", root, svc, id, pickupOpen, stamp(start.Add(pickupValidity)))
	before := f.s.Revision()
	if got := f.collect(root, ref); got.status != 200 {
		t.Fatalf("collect %+v", got)
	}
	if f.s.Revision() != before {
		t.Error("a collect changed the revision")
	}
	f.wantPickup(t, "collected", root, svc, id, pickupCollected, "")
	f.code(t, "second collect", f.collect(root, ref), 403, string(commands.PermissionDenied))
	f.wantPickup(t, "after a second collect", root, svc, id, pickupCollected, "")

	late, lateRef, _ := f.issued(root, svc, "1")
	f.advance(pickupValidity - time.Nanosecond)
	f.wantPickup(t, "just before the pickup end", root, svc, late, pickupOpen, stamp(start.Add(pickupValidity)))
	f.advance(time.Nanosecond)
	f.wantPickup(t, "at the pickup end", root, svc, late, pickupNever, "")
	f.code(t, "collect at the pickup end", f.collect(root, lateRef), 403, string(commands.PermissionDenied))
	f.newService(root, "prune")
	f.wantPickup(t, "after a service command prunes", root, svc, late, pickupNever, "")
	f.wantPickup(t, "the collected one is unchanged", root, svc, id, pickupCollected, "")

	revoked, _, _ := f.issued(root, svc, "1")
	if got := f.run(root, "service.revoke_credential", revoked, nil); got.status != 200 {
		t.Fatalf("revoke %+v", got)
	}
	f.wantPickup(t, "a revoked credential", root, svc, revoked, "", "")
	f.advance(25 * time.Hour)
	root = f.as(AdminLogin)
	f.wantPickup(t, "an expired credential", root, svc, late, "", "")
	f.wantPickup(t, "the live collected credential", root, svc, id, pickupCollected, "")
}

func TestPickupIsOpenOnlyWhileTheIssuingSessionIsLive(t *testing.T) {
	f := memberWorld(t)
	root := f.as(AdminLogin)
	svc := f.newService(root, "sessions")
	signedOut := f.as(AdminLogin)
	a, aRef, _ := f.issued(signedOut, svc, "1")
	if got := f.logout(signedOut); got.status != 204 {
		t.Fatalf("logout %+v", got)
	}
	f.wantPickup(t, "after sign-out", root, svc, a, pickupNever, "")
	f.code(t, "collect after sign-out", f.collect(signedOut, aRef), 401, string(commands.Unauthenticated))

	ended := f.as(AdminLogin)
	b, _, _ := f.issued(ended, svc, "1")
	if got := f.selfPost(root, "sessions/end", f.othersBody(root)); got.status != 202 && got.status != 200 {
		t.Fatalf("end others %+v", got)
	}
	f.drainSelf()
	f.wantPickup(t, "after an M05 session end", root, svc, b, pickupNever, "")

	idle := f.as(AdminLogin)
	c, cRef, _ := f.issued(idle, svc, "1")
	f.stretch(cRef, start.Add(3*time.Hour))
	f.advance(20 * time.Minute)
	reader := f.as(AdminLogin)
	for i := 0; i < 3; i++ {
		f.wantPickup(t, "idle end as the estimate", reader, svc, c, pickupOpen, stamp(start.Add(30*time.Minute)))
	}
	f.advance(10 * time.Minute)
	f.wantPickup(t, "after the idle end, reads did not extend it", reader, svc, c, pickupNever, "")
	f.code(t, "collect after the idle end", f.collect(idle, cRef), 401, string(commands.Unauthenticated))

	evicted := f.as(AdminLogin)
	d, dRef, _ := f.issued(evicted, svc, "1")
	var latest actor
	for i := 0; i < 2*f.s.cfg.Identity.SessionsPerUser; i++ {
		f.advance(time.Second)
		latest = f.as(AdminLogin)
	}
	f.wantPickup(t, "after eviction", latest, svc, d, pickupNever, "")
	f.code(t, "collect after eviction", f.collect(evicted, dRef), 401, string(commands.Unauthenticated))

	held := f.as(AdminLogin)
	e, _, _ := f.issued(held, svc, "1")
	if got := f.run(latest, "service.suspend", svc, nil); got.status != 200 {
		t.Fatalf("suspend %+v", got)
	}
	if got := f.run(latest, "service.reinstate", svc, nil); got.status != 200 {
		t.Fatalf("reinstate %+v", got)
	}
	f.wantPickup(t, "after suspend and reinstate", latest, svc, e, pickupNever, "")
}

func TestPickupEndsWithTheSignInAge(t *testing.T) {
	f := setup(t, SeedSample).ready()
	issuer := f.as(AdminLogin)
	svc := f.newService(issuer, "age")
	id, ref, _ := f.issued(issuer, svc, "30")
	f.stretch(ref, start.Add(13*time.Hour))
	for steps := 0; ; steps++ {
		f.advance(25 * time.Minute)
		if got := f.me(issuer); got.status != 200 {
			if steps < 2 {
				t.Fatalf("the session ended after %d steps", steps)
			}
			break
		}
		want := f.s.clock().Add(sessionIdle)
		if end := start.Add(sessionAbsolute); end.Before(want) {
			want = end
		}
		f.wantPickup(t, "kept alive", f.as(AdminLogin), svc, id, pickupOpen, stamp(want))
	}
	f.wantPickup(t, "after the session can no longer be admitted", f.as(AdminLogin), svc, id, pickupNever, "")
	f.code(t, "collect after the end", f.collect(issuer, ref), 401, string(commands.Unauthenticated))
	f.prunedAtSignIn(t)
}

func (f world) prunedAtSignIn(t *testing.T) {
	t.Helper()
	root := f.as(AdminLogin)
	svc := f.newService(root, "pruned")
	id, ref, _ := f.issued(root, svc, "30")
	f.stretch(ref, f.s.clock().Add(13*time.Hour))
	f.s.domain.Lock()
	f.s.sessions.prune(f.s.clock().Add(time.Second))
	f.s.domain.Unlock()
	f.wantPickup(t, "after the session records are pruned", f.as(AdminLogin), svc, id, pickupNever, "")
}

func TestPickupFollowsTheRolePeriodAndPermission(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	svc := f.newService(root, "roles")
	until := f.at(5 * time.Minute)
	if got := f.period(root, SampleAdmin2, until); got.status != 200 {
		t.Fatalf("period %s", got.raw)
	}
	bounded := f.as(Admin2Login)
	id, ref, _ := f.issued(bounded, svc, "1")
	f.wantPickup(t, "the role period end is the estimate", root, svc, id, pickupOpen, until)
	end, _ := time.Parse(time.RFC3339, until)
	f.advance(end.Sub(f.s.clock()))
	f.wantPickup(t, "at the role period end", root, svc, id, pickupNever, "")
	f.sweep()
	f.wantPickup(t, "after the role sweep", root, svc, id, pickupNever, "")
	f.s.domain.Lock()
	_, kept := f.s.dir.pickups[ref]
	f.s.domain.Unlock()
	if kept {
		t.Error("the role sweep kept the pickup")
	}

	demoted := f.as(Admin3Login)
	other, otherRef, _ := f.issued(demoted, svc, "1")
	f.wantPickup(t, "before the demotion", root, svc, other, pickupOpen, stamp(f.s.clock().Add(pickupValidity)))
	if got := f.run(root, "identity.change_role", SampleAdmin3, map[string]string{"role": "member"}); got.status != 200 {
		t.Fatalf("demote %s", got.raw)
	}
	f.wantPickup(t, "after the demotion", root, svc, other, pickupNever, "")
	if got := f.collect(demoted, otherRef); got.status == 200 {
		t.Errorf("a demoted administrator collected the secret %+v", got)
	}
}

func TestPickupDroppedByBulkCleanup(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	svc := f.newService(root, "bulk")
	target := f.as(Admin2Login)
	id, ref, _ := f.issued(target, svc, "1")
	bulk, digest := f.prepared(t, root, "suspend_membership", SampleAdmin2)
	if got := f.execute(root, bulk, digest); got.status != 202 {
		t.Fatalf("execute %s", got.raw)
	}
	f.wantPickup(t, "after the deny", root, svc, id, pickupNever, "")
	f.s.drainCleanup()
	f.wantPickup(t, "after the cleanup", root, svc, id, pickupNever, "")
	f.s.domain.Lock()
	_, kept := f.s.dir.pickups[ref]
	f.s.domain.Unlock()
	if kept {
		t.Error("the cleanup kept the pickup")
	}
}

func TestPickupEntriesAreBoundedAndOtherReadsUnchanged(t *testing.T) {
	f := setupWith(t, func(c *Config) {
		c.Commands = CommandLimits()
		c.Commands.Replay = 2 * time.Hour
	})
	root := f.as(AdminLogin)
	if got := f.servicesRead(root).body["limits"].(map[string]any)["replay_seconds"]; got != 7200.0 {
		t.Errorf("replay_seconds %v", got)
	}
	svc := f.newService(root, "bound")
	retired := f.newService(root, "retired")
	for i := 0; i < 3; i++ {
		f.issued(root, svc, "1")
	}
	gone, _, _ := f.issued(root, retired, "1")
	if n := f.collectedEntries(); n != 4 {
		t.Fatalf("entries %d", n)
	}
	if got := f.run(root, "service.retire", retired, nil); got.status != 200 {
		t.Fatalf("retire %+v", got)
	}
	f.wantPickup(t, "a credential revoked by retire", root, retired, gone, "", "")
	if n := f.collectedEntries(); n != 3 {
		t.Errorf("entries after retire %d", n)
	}
	f.advance(25 * time.Hour)
	root = f.as(AdminLogin)
	kept, _, _ := f.issued(root, svc, "2")
	if n := f.collectedEntries(); n != 1 {
		t.Errorf("entries after an issue sweeps expired ones %d", n)
	}
	f.wantPickup(t, "the new credential", root, svc, kept, pickupOpen, stamp(f.s.clock().Add(pickupValidity)))
	if got := f.run(root, "service.revoke_credential", kept, nil); got.status != 200 {
		t.Fatalf("revoke %+v", got)
	}
	if n := f.collectedEntries(); n != 0 {
		t.Errorf("entries after revoke %d", n)
	}
	open, _, _ := f.issued(root, svc, "2")
	if raw := f.integrations(root).raw; strings.Contains(raw, "secret_pickup") || !strings.Contains(raw, svc) {
		t.Errorf("integrations read %s", raw)
	}
	f.wantPickup(t, "unchanged by the integrations read", root, svc, open, pickupOpen, stamp(f.s.clock().Add(pickupValidity)))
	f.s.domain.Lock()
	delete(f.s.dir.collected, open)
	f.s.domain.Unlock()
	f.wantPickup(t, "an active credential with no record makes no claim", root, svc, open, "", "")
	doc := f.mustOpenAPI()
	for _, want := range []string{`"/identity/services"`, `"never_collected"`, `"replay_seconds"`, `"secret_pickup_until"`} {
		if !strings.Contains(string(doc), want) {
			t.Errorf("OpenAPI lacks %s", want)
		}
	}
}

func TestPickupReplayAddsNothing(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	svc := f.newService(root, "replays")
	body := fmt.Sprintf(`{"request_id":"r1","target_id":%q,"idempotency_key":"k1","expected_revision":%q,"rationale":"x","body":{"days":"1"}}`, svc, f.revision(root))
	first := f.call("service.issue_credential", body, root.cookie, root.csrf)
	again := f.call("service.issue_credential", body, root.cookie, root.csrf)
	if first.status != 200 || fmt.Sprint(first.body["fields"]) != fmt.Sprint(again.body["fields"]) || f.collectedEntries() != 1 {
		t.Fatalf("replay %s %s entries %d", first.raw, again.raw, f.collectedEntries())
	}
	fields := first.body["fields"].(map[string]any)
	id, ref := fields["credential"].(string), fields["secret_ref"].(string)
	f.collect(root, ref)
	third := f.call("service.issue_credential", body, root.cookie, root.csrf)
	f.code(t, "replayed reference after a collect", f.collect(root, third.body["fields"].(map[string]any)["secret_ref"].(string)), 403, string(commands.PermissionDenied))
	f.wantPickup(t, "after the replay", root, svc, id, pickupCollected, "")
	if n := len(f.service(root, svc)["credentials"].([]any)); n != 1 || f.collectedEntries() != 1 {
		t.Errorf("credentials %d entries %d", n, f.collectedEntries())
	}
}

func TestPickupReadsRaceWithCollects(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	svc := f.newService(root, "race")
	refs := []string{}
	for i := 0; i < 6; i++ {
		_, ref, _ := f.issued(root, svc, "1")
		refs = append(refs, ref)
	}
	var wg sync.WaitGroup
	for i, ref := range refs {
		wg.Add(2)
		go func(ref string) { defer wg.Done(); f.collect(root, ref) }(ref)
		go func(n int) { defer wg.Done(); f.servicesRead(root) }(i)
	}
	wg.Wait()
	for _, c := range f.service(root, svc)["credentials"].([]any) {
		if m := c.(map[string]any); m["secret_pickup"] != pickupCollected {
			t.Errorf("after concurrent collects %v", m)
		}
	}
}

type pickupWorld struct {
	f        world
	r        *rand.Rand
	reader   actor
	sessions []actor
	services []string
	issued   map[string]string
	issuer   map[string]actor
	refs     map[string]string
	read     map[string]bool
	trace    []string
}

func (p *pickupWorld) states() map[string]string {
	out := map[string]string{}
	p.reader = p.f.as(AdminLogin)
	for _, row := range p.f.servicesRead(p.reader).body["services"].([]any) {
		for _, c := range row.(map[string]any)["credentials"].([]any) {
			m := c.(map[string]any)
			state, _ := m["secret_pickup"].(string)
			if m["state"] == "active" {
				out[m["id"].(string)] = state
				if state == "" {
					out[m["id"].(string)] = "missing"
				}
			}
		}
	}
	return out
}

func (p *pickupWorld) step(t *testing.T) {
	t.Helper()
	f := p.f
	pick := func(n int) int { return p.r.Intn(n) }
	switch op := pick(8); op {
	case 0, 1:
		a := p.sessions[pick(len(p.sessions))]
		svc := p.services[pick(len(p.services))]
		got := f.runWith(a, f.revision(f.as(AdminLogin)), "service.issue_credential", svc, map[string]string{"days": "1"})
		if got.status == 200 {
			fields := got.body["fields"].(map[string]any)
			id := fields["credential"].(string)
			p.issued[id], p.issuer[id], p.refs[id] = svc, a, fields["secret_ref"].(string)
		}
		p.trace = append(p.trace, fmt.Sprintf("issue %s %d", svc, got.status))
	case 2:
		before := p.states()
		for id, state := range before {
			if p.read[id] || p.refs[id] == "" {
				continue
			}
			got := f.collect(p.issuer[id], p.refs[id])
			if (got.status == 200) != (state == pickupOpen) {
				t.Fatalf("collect of %s answered %d while the read said %s; trace %v", id, got.status, state, p.trace)
			}
			if got.status == 200 {
				p.read[id] = true
			}
			p.trace = append(p.trace, fmt.Sprintf("collect %s %d", id, got.status))
			return
		}
	case 3:
		d := time.Duration(1+pick(12)) * time.Minute
		f.advance(d)
		p.trace = append(p.trace, fmt.Sprintf("advance %v", d))
	case 4:
		for id := range p.issued {
			got := f.run(f.as(AdminLogin), "service.revoke_credential", id, nil)
			p.trace = append(p.trace, fmt.Sprintf("revoke %s %d", id, got.status))
			delete(p.issued, id)
			return
		}
	case 5:
		svc := p.services[pick(len(p.services))]
		f.run(f.as(AdminLogin), "service.suspend", svc, nil)
		f.run(f.as(AdminLogin), "service.reinstate", svc, nil)
		p.trace = append(p.trace, "suspend "+svc)
	case 6:
		n := pick(len(p.sessions))
		if pick(2) == 0 {
			f.logout(p.sessions[n])
			p.trace = append(p.trace, fmt.Sprintf("logout %d", n))
		} else {
			from := f.as(AdminLogin)
			f.selfPost(from, "sessions/end", f.othersBody(from))
			f.drainSelf()
			p.trace = append(p.trace, "end others")
		}
		p.sessions[n] = f.as(AdminLogin)
	case 7:
		for i := 0; i < 1+pick(4); i++ {
			p.sessions[pick(len(p.sessions))] = f.as(AdminLogin)
		}
		p.trace = append(p.trace, "sign-ins")
	}
}

func (p *pickupWorld) check(t *testing.T) {
	t.Helper()
	for id, state := range p.states() {
		if state == "missing" {
			t.Fatalf("active %s has no pickup state; trace %v", id, p.trace)
		}
		if state == pickupCollected && !p.read[id] {
			t.Fatalf("%s reads collected without a collect; trace %v", id, p.trace)
		}
		if p.read[id] && state != pickupCollected {
			t.Fatalf("%s was collected but reads %s; trace %v", id, state, p.trace)
		}
	}
}

func TestPickupStateProperty(t *testing.T) {
	runs := 30
	if testing.Short() {
		runs = 6
	}
	for seed := int64(1); seed <= int64(runs); seed++ {
		f := memberWorld(t)
		p := &pickupWorld{f: f, r: rand.New(rand.NewSource(seed)), issued: map[string]string{}, issuer: map[string]actor{}, refs: map[string]string{}, read: map[string]bool{}}
		p.reader = f.as(AdminLogin)
		for i := 0; i < 3; i++ {
			p.sessions = append(p.sessions, f.as(AdminLogin))
		}
		p.services = []string{f.newService(p.reader, "prop-a"), f.newService(p.reader, "prop-b")}
		for i := 0; i < 25; i++ {
			p.step(t)
			p.check(t)
		}
	}
}
