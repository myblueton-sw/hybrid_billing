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
	bodies := []string{loginBodyFor("nobody@example.invalid", pw), loginBodyFor(AdminLogin, "wrong password here"), loginBodyFor(Admin2Login, pw)}
	runs := make([][]time.Duration, len(bodies))
	for i := 0; i < 41; i++ {
		for k := range bodies {
			n := (i + k) % len(bodies)
			f.s.limit = &limiter{}
			f.s.passwords.Put(LoginKey(AdminLogin), rec)
			began := time.Now()
			if r := f.auth("login", bodies[n], pre); r.status != http.StatusUnauthorized {
				t.Fatalf("status %d", r.status)
			}
			runs[n] = append(runs[n], time.Since(began))
		}
	}
	median := func(d []time.Duration) time.Duration {
		sort.Slice(d, func(a, b int) bool { return d[a] < d[b] })
		return d[len(d)/2]
	}
	unknown, wrong, locked := median(runs[0]), median(runs[1]), median(runs[2])
	low, high := min(unknown, wrong, locked), max(unknown, wrong, locked)
	if high*2 > low*3 {
		t.Errorf("timing differs: unknown %v wrong %v locked %v", unknown, wrong, locked)
	}
}
