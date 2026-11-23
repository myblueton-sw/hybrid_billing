package server

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
)

func TestSelfEndRacesSignIns(t *testing.T) {
	for round := 0; round < 8; round++ {
		f := memberWorld(t)
		a := f.signedIn(MemberLogin, pw)
		listed := []actor{f.signedIn(MemberLogin, pw), f.signedIn(MemberLogin, pw)}
		page := f.testSessionSnapshot(a)
		handles := []string{handleOfActor(listed[0]), handleOfActor(listed[1])}
		var wg sync.WaitGroup
		newcomers := make([]actor, 4)
		for i := range newcomers {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				newcomers[i] = f.signedIn(MemberLogin, pw)
			}(i)
		}
		var end reply
		wg.Add(1)
		go func() {
			defer wg.Done()
			end = f.selfPost(a, "sessions/end", endJSON(fmt.Sprint(page["sessions_generation"]), handles))
		}()
		wg.Wait()
		if end.status != 202 || fmt.Sprint(end.body["count"]) != "2" {
			t.Fatalf("round %d %+v", round, end)
		}
		for _, l := range listed {
			if f.me(l).status != 401 {
				t.Fatalf("round %d a confirmed session survived", round)
			}
		}
		included := map[string]bool{}
		for _, h := range notIncludedOf(end.body) {
			included[h] = true
		}
		for _, n := range newcomers {
			if f.me(n).status != 200 {
				t.Fatalf("round %d a newcomer was ended", round)
			}
		}
		if len(included) > len(newcomers) {
			t.Fatalf("round %d not included %v", round, included)
		}
		if (len(included) > 0) != (end.body["generation_changed"] == true) {
			t.Fatalf("round %d generation flag %v with %d newcomers reported", round, end.body["generation_changed"], len(included))
		}
	}
}

func TestSelfEndRacesLockFreeRemovals(t *testing.T) {
	var got seen
	up := upstream(t, &got)
	f := readerWorld(t, up.URL)
	f.setPassword(MemberLogin, pw)
	a := f.as(MemberLogin)
	pool := []actor{}
	for i := 0; i < 8; i++ {
		pool = append(pool, f.as(MemberLogin))
	}
	page := f.testSessionSnapshot(a)
	var wg sync.WaitGroup
	for i, b := range pool {
		wg.Add(1)
		go func(i int, b actor) {
			defer wg.Done()
			with := func(r *http.Request) {
				r.AddCookie(b.cookie)
				r.Header.Set(csrfHeader, b.csrf)
				r.Header.Set(clientHeader, ConsoleClient)
			}
			switch i % 4 {
			case 0:
				f.do("GET", "/api/focus-overview?provider=aws", "", with)
			case 1:
				f.do("GET", "/api/v1/session", "", with)
			case 2:
				f.do("POST", "/api/v1/auth/logout", "", with)
			default:
				f.call("probe.write", mutation("k", "r1"), b.cookie, b.csrf)
			}
		}(i, b)
	}
	var end reply
	wg.Add(1)
	go func() {
		defer wg.Done()
		end = f.selfPost(a, "sessions/end", othersOf(page, nil))
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		f.drainSelf()
		f.me(a)
	}()
	wg.Wait()
	if end.status != 202 {
		t.Fatalf("%+v", end)
	}
	cleanup := end.body["cleanup"].(map[string]any)
	sum := 0
	for _, k := range []string{"done", "already_ended", "failed", "pending"} {
		sum += countOf(cleanup[k])
	}
	ended, gone := 0, 0
	for _, r := range resultsOf(end.body) {
		if r == resultEnded {
			ended++
		} else {
			gone++
		}
	}
	if sum != ended || ended+gone != len(pool) || fmt.Sprint(end.body["count"]) != fmt.Sprint(ended) {
		t.Errorf("counts %v ended %d gone %d", cleanup, ended, gone)
	}
	for _, b := range pool {
		if f.s.sessions.present(verifierOfActor(b)) {
			t.Error("a listed session survived")
		}
	}
}

func TestSelfEndRacesOwnSessions(t *testing.T) {
	for round := 0; round < 8; round++ {
		f := memberWorld(t)
		a := f.signedIn(MemberLogin, pw)
		b := f.signedIn(MemberLogin, pw)
		pa, pb := f.testSessionSnapshot(a), f.testSessionSnapshot(b)
		var ra, rb reply
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); ra = f.selfPost(a, "sessions/end", othersOf(pa, nil)) }()
		go func() { defer wg.Done(); rb = f.selfPost(b, "sessions/end", othersOf(pb, nil)) }()
		wg.Wait()
		statuses := fmt.Sprint(ra.status, rb.status)
		if statuses != "202 401" && statuses != "401 202" {
			t.Fatalf("round %d statuses %s", round, statuses)
		}
	}
}
