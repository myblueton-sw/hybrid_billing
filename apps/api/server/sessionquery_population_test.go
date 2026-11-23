package server

import (
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func TestSessionQueryBoundsPopulationPagesAndSparseScans(t *testing.T) {
	f := setupWith(t, func(c *Config) {
		c.Identity.SessionsPerUser = 400
		c.Access = DefaultAccessSettings()
		c.Access.SessionQueryScan = 100
	})
	f.setPassword(MemberLogin, pw)
	a := f.signedIn(MemberLogin, pw)
	handles := []string{handleOfActor(a)}
	for i := 0; i < 170; i++ {
		if cookie, _, err := f.s.openFor(LoginTriple(MemberLogin), SampleWorkspace, false); err != nil {
			t.Fatal(err)
		} else {
			handles = append(handles, handleOfActor(actor{cookie: cookie}))
		}
	}
	query := "?limit=17"
	seen := map[string]bool{}
	for n := 0; n < 20; n++ {
		got := f.sessionQuery(a, query)
		if got.status != 200 {
			t.Fatal(got.status)
		}
		rows := got.body["rows"].([]any)
		if len(rows) > 17 || len(got.raw) > 20000 {
			t.Fatal("query returned an unbounded population")
		}
		for _, raw := range rows {
			handle := raw.(map[string]any)["handle"].(string)
			if seen[handle] {
				t.Fatal("row repeated")
			}
			seen[handle] = true
		}
		next := got.body["next"].(string)
		if next == "" {
			break
		}
		query = "?limit=17&after=" + url.QueryEscape(next)
	}
	if len(seen) != 171 {
		t.Fatalf("bounded pages covered%d, want171", len(seen))
	}
	first := f.sessionQuery(a, "?q=not-present&limit=17")
	if first.status != 200 || first.body["scan_stopped"] != true || first.body["next"] == "" || first.body["count"].(map[string]any)["certainty"] != "lower_bound" {
		t.Fatal("sparse scan asserted a false exact zero")
	}
	sort.Strings(handles)
	cursor, valid := f.s.openSessionCursor(first.body["next"].(string))
	if !valid || cursor.After != handles[99] {
		t.Fatal("query exceeded or undershot configured source scan budget")
	}
	second := f.sessionQuery(a, "?q=not-present&limit=17&after="+url.QueryEscape(first.body["next"].(string)))
	if second.status != 200 || second.body["next"] != "" || second.body["count"].(map[string]any)["certainty"] != "exact" {
		t.Fatal("sparse continuation did not finish truthfully")
	}
}

func TestSessionQueryRejectsMissingCSRFAndUnknownDetailUniformly(t *testing.T) {
	f, _, _, a := queryActors(t)
	missing := f.do(http.MethodGet, "/api/v1/me/sessions", "", func(r *http.Request) { r.AddCookie(a.cookie) })
	if missing.status != 403 || missing.body["rows"] != nil {
		t.Fatal("query admitted missing CSRF")
	}
	for _, suffix := range []string{strings.Repeat("0", 16), strings.Repeat("f", 16)} {
		got := f.do(http.MethodGet, "/api/v1/me/sessions/"+suffix, "", func(r *http.Request) { r.AddCookie(a.cookie); r.Header.Set(csrfHeader, a.csrf) })
		if got.status != 404 || got.body["session"] != nil {
			t.Fatal("unknown detail disclosed existence")
		}
	}
}

func TestSessionQueryIndexConcurrentReadersAndLifecycle(t *testing.T) {
	store := newSessions()
	owner := sessionOwner{"person", "workspace", 1}
	var wait sync.WaitGroup
	for reader := 0; reader < 3; reader++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for n := 0; n < 300; n++ {
				state := store.indexState(owner, start, 0)
				rows, _, ok := store.indexed(owner, state.Version, "", 5)
				if ok {
					for _, row := range rows {
						if row.owner != owner {
							t.Error("foreign owner in concurrent query")
						}
					}
					if len(rows) > 5 {
						t.Error("concurrent page exceeded bound")
					}
				}
			}
		}()
	}
	wait.Add(1)
	go func() {
		defer wait.Done()
		for n := 0; n < 200; n++ {
			var v identity.Verifier
			v[0] = byte(n%50 + 1)
			if _, err := store.put(v, owner.Workspace, "login", start.Add(time.Duration(n)*time.Second), 1); err != nil {
				t.Error(err)
				return
			}
			if !store.bind(v, owner, start.Add(time.Hour)) {
				t.Error("concurrent bind failed")
				return
			}
			store.touch(v, start.Add(2*time.Hour))
			store.reauthenticated(v, start.Add(time.Minute))
			if n%3 == 0 {
				store.drop(v)
			}
		}
	}()
	wait.Wait()
}
