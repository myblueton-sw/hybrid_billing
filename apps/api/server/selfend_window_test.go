package server

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/accesshistory"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func (f world) moduleLive(v identity.Verifier) bool {
	f.t.Helper()
	_, err := f.s.identity.Boundary().Access(v, SampleWorkspace)
	return err == nil
}

func (f world) failModuleEnds(kind string) {
	f.s.fault = func(at string) bool { return at == kind }
}

func (f world) drainSelf() int {
	f.s.domain.Lock()
	defer f.s.domain.Unlock()
	return f.s.drainSelfEnds()
}

func (f world) lastEnd(a actor) map[string]any {
	f.t.Helper()
	ends := f.me(a).body["session_ends"].([]any)
	if len(ends) == 0 {
		f.t.Fatal("no session ends")
	}
	return ends[0].(map[string]any)
}

func TestSelfEndStaleCookiePerPath(t *testing.T) {
	var got seen
	up := upstream(t, &got)
	f := readerWorld(t, up.URL)
	f.setPassword(MemberLogin, pw)
	a := f.as(MemberLogin)
	b := f.as(MemberLogin)
	vb := verifierOfActor(b)
	captured, _ := f.s.sessions.get(vb)
	f.failModuleEnds("self-end-unavailable")
	end := f.selfPost(a, "sessions/end", f.othersBody(a))
	f.s.fault = nil
	if end.status != 202 || end.body["state"] != stateRunning || c3Of(end.body, handleOf(vb)) != "pending/retrying" {
		t.Fatalf("%+v", end)
	}
	if !f.s.sessions.isDenied(vb) || !f.moduleLive(vb) {
		t.Fatal("the pending window was not set up: module record must be live and the verifier denied")
	}
	with := func(r *http.Request) {
		r.AddCookie(b.cookie)
		r.Header.Set(csrfHeader, b.csrf)
		r.Header.Set(clientHeader, ConsoleClient)
	}
	f.code(t, "readers.go:112", f.do("GET", "/api/focus-overview?provider=aws", "", with), 401, string(commands.Unauthenticated))
	f.code(t, "boundary.go:119", f.do("GET", "/api/v1/session", "", with), 401, string(commands.Unauthenticated))
	f.code(t, "commands.go:174", f.call("probe.write", mutation("k", "r1"), b.cookie, b.csrf), 401, string(commands.Unauthenticated))
	if out := f.do("POST", "/api/v1/auth/logout", "", with); out.status != 204 || !f.s.sessions.isDenied(vb) {
		t.Errorf("signin.go:293 %+v", out)
	}
	if _, code := f.s.access(captured); code != commands.Unauthenticated {
		t.Errorf("a request resolved before the deny was admitted: %q", code)
	}
	if f.s.sessions.present(vb) {
		t.Error("the refused session was put back")
	}
	if f.me(a).status != 200 {
		t.Error("the current session stopped working")
	}
}

func TestSelfEndModuleRefusesAtOnce(t *testing.T) {
	f := memberWorld(t)
	a := f.as(MemberLogin)
	b := f.as(MemberLogin)
	vb := verifierOfActor(b)
	captured, _ := f.s.sessions.get(vb)
	f.selfPost(a, "sessions/end", f.othersBody(a))
	if f.moduleLive(vb) || f.s.sessions.isDenied(vb) {
		t.Fatal("the module record survived the deny")
	}
	if _, code := f.s.access(captured); code != commands.Unauthenticated {
		t.Errorf("admitted %q", code)
	}
}

func TestAccessRechecksAfterTheModuleAnswer(t *testing.T) {
	f := memberWorld(t)
	b := f.as(MemberLogin)
	vb := verifierOfActor(b)
	captured, _ := f.s.sessions.get(vb)
	f.s.accessHook = func(identity.Verifier) { f.s.sessions.drop(vb) }
	_, code := f.s.access(captured)
	f.s.accessHook = nil
	if code != commands.Unauthenticated {
		t.Errorf("a session removed after the module answer was admitted: %q", code)
	}
	c := f.as(MemberLogin)
	vc := verifierOfActor(c)
	captured, _ = f.s.sessions.get(vc)
	f.s.accessHook = func(identity.Verifier) { f.s.sessions.deny(vc, MemberLogin, f.s.clock()) }
	_, code = f.s.access(captured)
	f.s.accessHook = nil
	if code != commands.Unauthenticated {
		t.Errorf("a session denied after the module answer was admitted: %q", code)
	}
}

func TestSelfEndPickupNotRedeemableAfterDeny(t *testing.T) {
	f := memberWorld(t)
	a := f.as(AdminLogin)
	b := f.as(AdminLogin)
	got := f.run(b, "identity.create_user", usersTarget, map[string]string{"login": "pick2@example.invalid", "email": "pick2@example.invalid", "role": "member"})
	ref := got.body["fields"].(map[string]any)["link_ref"].(string)
	f.failModuleEnds("self-end-unavailable")
	f.selfPost(a, "sessions/end", f.othersBody(a))
	f.s.fault = nil
	f.code(t, "collect", f.do("GET", "/api/v1/identity/links/"+ref, "", func(r *http.Request) { r.AddCookie(b.cookie); r.Header.Set(csrfHeader, b.csrf) }), 401, string(commands.Unauthenticated))
	if _, ok := f.s.dir.pickups[ref]; ok {
		t.Error("pickup kept")
	}
}

func TestSelfEndRetriesWithBackoff(t *testing.T) {
	f := memberWorld(t)
	a := f.as(MemberLogin)
	b := f.as(MemberLogin)
	vb, hb := verifierOfActor(b), handleOfActor(b)
	f.failModuleEnds("self-end-unavailable")
	f.selfPost(a, "sessions/end", f.othersBody(a))
	steps := []struct {
		wait     time.Duration
		worked   int
		attempts int
	}{{999 * time.Millisecond, 0, 1}, {time.Millisecond, 1, 2}, {3999 * time.Millisecond, 0, 2}, {time.Millisecond, 1, 3}, {15999 * time.Millisecond, 0, 3}, {time.Millisecond, 1, 4}}
	for i, step := range steps {
		f.advance(step.wait)
		if n := f.drainSelf(); n != step.worked {
			t.Fatalf("step %d worked %d want %d", i, n, step.worked)
		}
		if r := f.s.retries[vb]; r == nil || r.attempts != step.attempts {
			t.Fatalf("step %d attempts %+v", i, r)
		}
	}
	end := f.lastEnd(a)
	if c3Of(end, hb) != "failed/retry_budget_spent" || end["state"] != stateIncomplete {
		t.Fatalf("after the budget %v", end)
	}
	f.advance(time.Hour)
	if f.drainSelf() != 0 || !f.s.sessions.isDenied(vb) {
		t.Error("a spent retry ran again or released the guard")
	}
	f.s.fault = nil
	f.advance(sessionAbsolute)
	f.drainSelf()
	if f.s.sessions.isDenied(vb) || f.s.retries[vb] != nil {
		t.Error("the sweep did not release the guard after the session lifetime")
	}
}

func TestSelfEndRetrySucceeds(t *testing.T) {
	f := memberWorld(t)
	a := f.as(MemberLogin)
	b := f.as(MemberLogin)
	vb, hb := verifierOfActor(b), handleOfActor(b)
	f.failModuleEnds("self-end-unavailable")
	f.selfPost(a, "sessions/end", f.othersBody(a))
	f.s.fault = nil
	f.advance(time.Second)
	f.drainSelf()
	end := f.lastEnd(a)
	if c3Of(end, hb) != "done/session_ended" || end["state"] != stateComplete || f.s.sessions.isDenied(vb) || f.moduleLive(vb) {
		t.Fatalf("%v", end)
	}
}

func TestSelfEndUnexpectedErrorFailsAtOnce(t *testing.T) {
	f := memberWorld(t)
	a := f.as(MemberLogin)
	b := f.as(MemberLogin)
	vb, hb := verifierOfActor(b), handleOfActor(b)
	f.failModuleEnds("self-end-unexpected")
	end := f.selfPost(a, "sessions/end", f.othersBody(a))
	f.s.fault = nil
	if c3Of(end.body, hb) != "failed/unexpected_error" || end.body["state"] != stateIncomplete || !f.s.sessions.isDenied(vb) {
		t.Fatalf("%+v", end)
	}
	f.advance(time.Minute)
	if f.drainSelf() != 0 {
		t.Error("a failed item was retried")
	}
}

func TestSelfEndAlreadyClosedModuleRecord(t *testing.T) {
	f := memberWorld(t)
	a := f.as(MemberLogin)
	b := f.as(MemberLogin)
	vb, hb := verifierOfActor(b), handleOfActor(b)
	_ = f.s.identity.Boundary().EndSession(vb)
	end := f.selfPost(a, "sessions/end", f.othersBody(a))
	if c3Of(end.body, hb) != "already_ended/already_closed" || end.body["state"] != stateComplete || f.s.sessions.isDenied(vb) {
		t.Fatalf("%+v", end)
	}
	cleanup := end.body["cleanup"].(map[string]any)
	if fmt.Sprint(end.body["count"], cleanup["done"], cleanup["already_ended"]) != "1 0 1" {
		t.Errorf("counts %v", cleanup)
	}
}

func TestSelfEndEvictionSettlesDeniedFirst(t *testing.T) {
	f := memberWorld(t)
	a := f.signedIn(MemberLogin, pw)
	for i := 0; i < 9; i++ {
		f.signedIn(MemberLogin, pw)
	}
	f.failModuleEnds("self-end-unavailable")
	f.selfPost(a, "sessions/end", f.othersBody(a))
	f.s.fault = nil
	if len(f.s.sessions.deniedOf(MemberLogin)) != 9 {
		t.Fatalf("denied %d", len(f.s.sessions.deniedOf(MemberLogin)))
	}
	f.signedIn(MemberLogin, pw)
	if f.me(a).status != 200 {
		t.Fatal("eviction ended the caller's live session instead of a denied record")
	}
	if len(f.s.sessions.deniedOf(MemberLogin)) != 0 || f.lastEnd(a)["state"] != stateComplete {
		t.Errorf("denied records were not settled: %v", f.lastEnd(a)["cleanup"])
	}
}

func TestSelfEndProbeSettlesDeniedFirst(t *testing.T) {
	f := memberWorld(t)
	a := f.signedIn(MemberLogin, pw)
	for i := 0; i < 9; i++ {
		f.signedIn(MemberLogin, pw)
	}
	f.failModuleEnds("self-end-unavailable")
	f.selfPost(a, "sessions/end", f.othersBody(a))
	f.s.fault = nil
	f.s.domain.Lock()
	v, err := f.s.probe(MemberLogin)
	if err == nil {
		_ = f.s.identity.Boundary().EndSession(v)
	}
	f.s.domain.Unlock()
	if err != nil || f.me(a).status != 200 || len(f.s.sessions.deniedOf(MemberLogin)) != 0 {
		t.Errorf("probe %v, caller %d, denied %d", err, f.me(a).status, len(f.s.sessions.deniedOf(MemberLogin)))
	}
}

func TestDeniedSetCapFailsClosed(t *testing.T) {
	f := memberWorld(t)
	a := f.signedIn(MemberLogin, pw)
	f.signedIn(MemberLogin, pw)
	now := f.s.clock()
	for i := 0; len(f.s.sessions.denied) < maxSessions; i++ {
		f.s.sessions.denied[verifierOf(random(tokenBytes))] = denial{login: "filler@example.invalid", at: now}
	}
	f.failModuleEnds("self-end-unavailable")
	end := f.selfPost(a, "sessions/end", f.othersBody(a))
	f.s.fault = nil
	if end.status != 202 || len(f.s.sessions.denied) != maxSessions || !f.s.sessions.overflowed(MemberLogin) {
		t.Fatalf("cap %d overflow %v %+v", len(f.s.sessions.denied), f.s.sessions.overflowed(MemberLogin), end)
	}
	f.code(t, "sign-in while full", f.signin(MemberLogin, pw), 503, string(commands.DependencyUnavailable))
	if f.signin(AdminLogin, pw).status != 200 {
		t.Error("another login was refused")
	}
	f.advance(sessionAbsolute)
	f.drainSelf()
	if f.s.sessions.overflowed(MemberLogin) || len(f.s.sessions.denied) != 0 {
		t.Errorf("sweep left %d entries", len(f.s.sessions.denied))
	}
	if f.signin(MemberLogin, pw).status != 200 {
		t.Error("sign-in still refused after the sweep")
	}
}

func TestSelfEndSweepKeepsEntriesUntilTheLifetime(t *testing.T) {
	f := memberWorld(t)
	a := f.as(MemberLogin)
	b := f.as(MemberLogin)
	vb := verifierOfActor(b)
	f.failModuleEnds("self-end-unexpected")
	f.selfPost(a, "sessions/end", f.othersBody(a))
	f.s.fault = nil
	f.advance(sessionAbsolute - time.Nanosecond)
	f.drainSelf()
	if !f.s.sessions.isDenied(vb) {
		t.Fatal("released before the lifetime")
	}
	f.advance(time.Nanosecond)
	f.drainSelf()
	if f.s.sessions.isDenied(vb) {
		t.Fatal("kept after the lifetime")
	}
}

func TestOtherSessionEndersChangeTheGeneration(t *testing.T) {
	cases := map[string]func(f world, a, b actor){
		"logout": func(f world, a, b actor) { f.logout(b) },
		"password change": func(f world, a, b actor) {
			if got := f.selfPost(a, "password", pwBody(pw, newPw)); got.status != 204 {
				f.t.Fatalf("password %+v", got)
			}
		},
		"fifth wrong password": func(f world, a, b actor) {
			for i := 0; i < selfAttempts; i++ {
				f.selfPost(b, "password", pwBody("wrong password here", newPw))
			}
		},
		"fifth wrong reauthentication": func(f world, a, b actor) {
			for i := 0; i < selfAttempts; i++ {
				f.reauth(b, "wrong password here")
			}
		},
		"administrator suspension": func(f world, a, b actor) {
			f.s.domain.Lock()
			f.s.endSessions(MemberLogin, accesshistory.CauseAdministrator)
			f.s.domain.Unlock()
		},
		"end one": func(f world, a, b actor) {
			f.s.domain.Lock()
			f.s.endOne(verifierOfActor(b), accesshistory.CauseAccessChanged)
			f.s.domain.Unlock()
		},
	}
	for name, run := range cases {
		t.Run(name, func(t *testing.T) {
			f := memberWorld(t)
			a := f.signedIn(MemberLogin, pw)
			b := f.signedIn(MemberLogin, pw)
			listed := f.me(a).body
			before := f.s.sessions.generation(MemberLogin)
			run(f, a, b)
			if f.s.sessions.generation(MemberLogin) == before {
				t.Fatal("generation unchanged")
			}
			if len(f.s.selfEnds) != 0 {
				t.Error("a session end operation was created")
			}
			if f.me(a).status != 200 {
				return
			}
			got := f.selfPost(a, "sessions/end", othersOf(listed, nil))
			if got.status != 202 || got.body["generation_changed"] != true || resultsOf(got.body)[handleOfActor(b)] != resultGone {
				t.Errorf("stale manifest %+v", got)
			}
		})
	}
}

func TestSignInEvictionChangesTheGeneration(t *testing.T) {
	f := memberWorld(t)
	a := f.signedIn(MemberLogin, pw)
	for i := 0; i < 9; i++ {
		f.signedIn(MemberLogin, pw)
	}
	before := f.s.sessions.generation(MemberLogin)
	f.signedIn(MemberLogin, pw)
	if f.s.sessions.generation(MemberLogin) == before || len(f.s.selfEnds) != 0 {
		t.Error("eviction")
	}
	_ = a
}

func TestSelfEndIgnoresLedgerStatesRegressionGuard(t *testing.T) {
	for name, build := range ledgerBlockedWorlds() {
		t.Run(name, func(t *testing.T) {
			f := build(t)
			root := f.as(AdminLogin)
			f.as(AdminLogin)
			if got := f.selfPost(root, "sessions/end", f.othersBody(root)); got.status != 202 || fmt.Sprint(got.body["count"]) != "1" {
				t.Errorf("%+v", got)
			}
		})
	}
	t.Run("operator recovery", func(t *testing.T) {
		f := memberWorld(t)
		root := f.as(AdminLogin)
		f.as(AdminLogin)
		f.s.stranded = []uint64{1}
		if got := f.selfPost(root, "sessions/end", f.othersBody(root)); got.status != 202 {
			t.Errorf("%+v", got)
		}
	})
}

func TestSelfEndKeepsFiveOperations(t *testing.T) {
	f := memberWorld(t)
	a := f.as(MemberLogin)
	for i := 0; i < selfEndsKept+1; i++ {
		f.as(MemberLogin)
		f.selfPost(a, "sessions/end", f.othersBody(a))
	}
	ends := f.me(a).body["session_ends"].([]any)
	if len(ends) != selfEndsKept || ends[0].(map[string]any)["operation"] != fmt.Sprintf("session-end-%d", selfEndsKept+1) {
		t.Errorf("kept %d first %v", len(ends), ends[0])
	}
}

func TestEndOneSettlesADeniedVerifier(t *testing.T) {
	f := memberWorld(t)
	a := f.as(MemberLogin)
	b := f.as(MemberLogin)
	vb, hb := verifierOfActor(b), handleOfActor(b)
	f.failModuleEnds("self-end-unavailable")
	f.selfPost(a, "sessions/end", f.othersBody(a))
	f.s.fault = nil
	if !f.s.sessions.isDenied(vb) || f.s.retries[vb] == nil {
		t.Fatal("setup: the verifier must be denied with a pending retry")
	}
	got := f.s.execute("probe.write", commands.Call{Request: "req-1"}, vb)
	if got.Err == nil || got.Err.Code != commands.Unauthenticated {
		t.Fatalf("execute %+v", got)
	}
	if f.s.sessions.isDenied(vb) || f.s.retries[vb] != nil || f.moduleLive(vb) {
		t.Error("endOne did not settle the denied verifier")
	}
	if end := f.lastEnd(a); c3Of(end, hb) != "done/session_ended" || end["state"] != stateComplete {
		t.Errorf("operation %v", end)
	}
}

func TestEndOneKeepsAPendingRetryWhileTheModuleIsDown(t *testing.T) {
	f := memberWorld(t)
	a := f.as(MemberLogin)
	b := f.as(MemberLogin)
	vb := verifierOfActor(b)
	f.failModuleEnds("self-end-unavailable")
	f.selfPost(a, "sessions/end", f.othersBody(a))
	f.s.execute("probe.write", commands.Call{Request: "req-1"}, vb)
	f.s.fault = nil
	if !f.s.sessions.isDenied(vb) || f.s.retries[vb] == nil {
		t.Error("a failed end released the guard or dropped the retry")
	}
}

func TestDeniedOverflowCreatesNoRetry(t *testing.T) {
	f := memberWorld(t)
	a := f.signedIn(MemberLogin, pw)
	b := f.signedIn(MemberLogin, pw)
	now := f.s.clock()
	for len(f.s.sessions.denied) < maxSessions {
		f.s.sessions.denied[verifierOf(random(tokenBytes))] = denial{login: "filler@example.invalid", at: now}
	}
	f.failModuleEnds("self-end-unavailable")
	end := f.selfPost(a, "sessions/end", f.othersBody(a))
	f.s.fault = nil
	if c3Of(end.body, handleOfActor(b)) != "failed/denied_set_full" || end.body["state"] != stateIncomplete || len(f.s.retries) != 0 {
		t.Fatalf("%+v retries %d", end.body, len(f.s.retries))
	}
}

func TestSelfEndInFlightRequestsAreRefused(t *testing.T) {
	var got seen
	up := upstream(t, &got)
	f := readerWorld(t, up.URL)
	f.setPassword(MemberLogin, pw)
	for _, path := range []string{"/api/focus-overview?provider=aws", "/api/v1/session"} {
		a := f.as(MemberLogin)
		b := f.as(MemberLogin)
		body := f.othersBody(a)
		vb := verifierOfActor(b)
		var out reply
		finished := make(chan struct{})
		go func() {
			out = inFlightAfter(t, f, b, path, func() {
				if end := f.selfPost(a, "sessions/end", body); end.status != 202 {
					t.Errorf("end %+v", end)
				}
			})
			close(finished)
		}()
		<-finished
		f.code(t, "in flight "+path, out, 401, string(commands.Unauthenticated))
		if f.s.sessions.present(vb) {
			t.Errorf("%s put the session back", path)
		}
		f.logout(a)
	}
}

func inFlightAfter(t *testing.T, f world, b actor, path string, during func()) reply {
	t.Helper()
	vb := verifierOfActor(b)
	held, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	f.s.accessHook = func(v identity.Verifier) {
		if v == vb {
			once.Do(func() { close(held); <-release })
		}
	}
	defer func() { f.s.accessHook = nil }()
	done := make(chan reply)
	go func() {
		done <- f.do("GET", path, "", func(r *http.Request) {
			r.AddCookie(b.cookie)
			r.Header.Set(csrfHeader, b.csrf)
			r.Header.Set(clientHeader, ConsoleClient)
		})
	}()
	<-held
	during()
	close(release)
	return <-done
}

func TestEndOneUnexpectedErrorLeavesTheRetryPending(t *testing.T) {
	f := memberWorld(t)
	a := f.as(MemberLogin)
	b := f.as(MemberLogin)
	vb, hb := verifierOfActor(b), handleOfActor(b)
	f.failModuleEnds("self-end-unavailable")
	f.selfPost(a, "sessions/end", f.othersBody(a))
	f.failModuleEnds("self-end-unexpected")
	f.s.execute("probe.write", commands.Call{Request: "req-1"}, vb)
	f.s.fault = nil
	if r := f.s.retries[vb]; r == nil || r.final || !f.s.sessions.isDenied(vb) || c3Of(f.lastEnd(a), hb) != "pending/retrying" {
		t.Errorf("retry %+v c3 %s", r, c3Of(f.lastEnd(a), hb))
	}
}

func TestEndOneSettlesAnAlreadyClosedRecord(t *testing.T) {
	f := memberWorld(t)
	a := f.as(MemberLogin)
	b := f.as(MemberLogin)
	vb, hb := verifierOfActor(b), handleOfActor(b)
	f.failModuleEnds("self-end-unavailable")
	f.selfPost(a, "sessions/end", f.othersBody(a))
	f.s.fault = nil
	_ = f.s.identity.Boundary().EndSession(vb)
	f.s.execute("probe.write", commands.Call{Request: "req-1"}, vb)
	if f.s.sessions.isDenied(vb) || f.s.retries[vb] != nil || c3Of(f.lastEnd(a), hb) != "already_ended/already_closed" {
		t.Errorf("c3 %s", c3Of(f.lastEnd(a), hb))
	}
}

func TestSelfEndInFlightIsRefusedWhenTheDeniedSetIsFull(t *testing.T) {
	var got seen
	up := upstream(t, &got)
	f := readerWorld(t, up.URL)
	f.setPassword(MemberLogin, pw)
	a := f.as(MemberLogin)
	targets := []actor{f.as(MemberLogin), f.as(MemberLogin)}
	now := f.s.clock()
	for len(f.s.sessions.denied) < maxSessions {
		f.s.sessions.denied[verifierOf(random(tokenBytes))] = denial{login: "filler@example.invalid", at: now}
	}
	for i, path := range []string{"/api/focus-overview?provider=aws", "/api/v1/session"} {
		b := targets[i]
		vb := verifierOfActor(b)
		body := endJSON(f.generationOf(a), []string{handleOf(vb)})
		f.failModuleEnds("self-end-unavailable")
		out := inFlightAfter(t, f, b, path, func() {
			if end := f.selfPost(a, "sessions/end", body); end.status != 202 || c3Of(end.body, handleOf(vb)) != "failed/denied_set_full" {
				t.Errorf("end %+v", end)
			}
		})
		f.s.fault = nil
		if f.s.sessions.isDenied(vb) || !f.moduleLive(vb) {
			t.Fatal("setup: the verifier must be outside the denied set with a live module record")
		}
		f.code(t, "in flight with a full denied set "+path, out, 401, string(commands.Unauthenticated))
		if f.s.sessions.present(vb) {
			t.Errorf("%s put the session back", path)
		}
	}
}

func TestEndedSessionsCarryAllEightCleanupItems(t *testing.T) {
	f := memberWorld(t)
	a := f.as(MemberLogin)
	f.as(MemberLogin)
	f.failModuleEnds("self-end-unavailable")
	f.as(MemberLogin)
	f.s.fault = nil
	end := f.selfPost(a, "sessions/end", othersOf(f.me(a).body, nil))
	listed := map[string]bool{}
	for _, row := range end.body["sessions"].([]any) {
		m := row.(map[string]any)
		ids := []string{}
		for _, item := range m["items"].([]any) {
			ids = append(ids, fmt.Sprint(item.(map[string]any)["item"]))
		}
		listed[m["result"].(string)] = true
		if m["result"] == resultEnded && fmt.Sprint(ids) != "[C1 C2 C3 C4 C5 C6 C7 C8]" {
			t.Errorf("ended row items %v", ids)
		}
	}
	gone := f.selfPost(a, "sessions/end", endJSON(f.generationOf(a), []string{"00000000000000ab"}))
	if row := gone.body["sessions"].([]any)[0].(map[string]any); row["result"] != resultGone || len(row["items"].([]any)) != 0 {
		t.Errorf("gone row %v", row)
	}
	if !listed[resultEnded] {
		t.Error("no ended row")
	}
}
