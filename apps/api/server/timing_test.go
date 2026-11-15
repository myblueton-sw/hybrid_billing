package server

import (
	"net/http"
	"sort"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/passwords"
)

func TestFailCodesHaveRules(t *testing.T) {
	for _, code := range []commands.Code{commands.Unauthenticated, commands.InvalidRequest, commands.AuthorityUnavailable, commands.InternalError,
		commands.UnsupportedCommand, commands.DependencyUnavailable} {
		if moduleRules[code].status < 400 {
			t.Errorf("%s has no rule", code)
		}
	}
}

func TestTimingIndistinguishable(t *testing.T) {
	if testing.Short() {
		t.Skip("timing")
	}
	f := setup(t, SeedSample)
	f.s.passwords, _ = passwords.New(f.s.clock, passwords.DefaultParams())
	f = f.ready()
	pre := f.pre()
	rec, _ := f.s.passwords.Hash(LoginKey(AdminLogin), pw)
	f.s.passwords.Put(LoginKey(AdminLogin), rec)
	rec2, _ := f.s.passwords.Hash(LoginKey(Admin2Login), pw)
	f.s.passwords.Put(LoginKey(Admin2Login), rec2)
	lock := func() {
		for i := 0; i < 5; i++ {
			f.s.passwords.Verify(LoginKey(Admin2Login), "wrong password here")
		}
	}
	lock()
	median := func(body string) time.Duration {
		var d []time.Duration
		for i := 0; i < 20; i++ {
			f.s.limit = &limiter{}
			f.s.passwords.Put(LoginKey(AdminLogin), rec)
			began := time.Now()
			if r := f.auth("login", body, pre); r.status != http.StatusUnauthorized {
				t.Fatalf("status %d", r.status)
			}
			d = append(d, time.Since(began))
		}
		sort.Slice(d, func(a, b int) bool { return d[a] < d[b] })
		return d[len(d)/2]
	}
	unknown := median(loginBodyFor("nobody@example.invalid", pw))
	wrong := median(loginBodyFor(AdminLogin, "wrong password here"))
	locked := median(loginBodyFor(Admin2Login, pw))
	low, high := min(unknown, wrong, locked), max(unknown, wrong, locked)
	if high*2 > low*3 {
		t.Errorf("timing differs: unknown %v wrong %v locked %v", unknown, wrong, locked)
	}
}
