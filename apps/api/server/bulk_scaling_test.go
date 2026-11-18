package server

import (
	"math/rand/v2"
	"strconv"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/bulkrevoke"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func syntheticJob(n int, number uint64, at time.Time, users func(int) string) bulkrevoke.View {
	items := make([]bulkrevoke.Item, n)
	for i := range items {
		items[i] = bulkrevoke.Item{Index: i, Target: bulkrevoke.Target{User: users(i)}, Outcome: bulkrevoke.OutcomeUnprocessed, State: bulkrevoke.DenyConfirmed,
			Receipt: bulkrevoke.Receipt{Sequence: uint64(i + 2)}}
	}
	return bulkrevoke.View{Number: number, Workspace: SampleWorkspace, Owner: SampleAdmin, Effect: bulkrevoke.SuspendMembership, DenyAt: at, Items: items, Headed: true, Digest: "synthetic",
		Header: bulkrevoke.Receipt{Sequence: 1}}
}

func TestCleanupOfMoreThanTenThousandPeople(t *testing.T) {
	f := setupWith(t, func(c *Config) {
		c.Access = DefaultAccessSettings()
		c.Access.BulkMaxMatching, c.Access.BulkRetainedMax = 10001, 10000000
	})
	f.s.bulk.Restore(0, []bulkrevoke.View{syntheticJob(10001, 500, f.s.clock(), func(i int) string { return "nobody-" + strconv.Itoa(i) })})
	v, _ := f.s.bulk.Get(SampleWorkspace, SampleAdmin, bulkrevoke.ManifestID(500))
	f.s.startCleanup(v)
	if job := f.job(f.as(AdminLogin), v.ID); job["job_state"] != "running" {
		t.Fatalf("the cleanup job was not created %+v", job)
	}
	if n := f.s.drainCleanup(); n != 101 {
		t.Errorf("tasks %d", n)
	}
	got, _ := f.s.bulk.Get(SampleWorkspace, SampleAdmin, v.ID)
	if got.Counts.Succeeded != 10001 || got.Counts.Unprocessed != 0 {
		t.Errorf("counts %+v", got.Counts)
	}
	if cleanupLimits(100000).Tasks != 1000 || cleanupLimits(5000).Tasks != 100 || cleanupLimits(10001).Tasks != 101 || cleanupLimits(1).Tasks != 100 {
		t.Error("task limits")
	}
}

func oldSessionsOf(s *Server, user string) []identity.Verifier {
	login, _ := s.dir.loginOf(user)
	return s.sessions.byLogin(login)
}

func oldPickupsOf(s *Server, user string) int {
	own := map[identity.Verifier]bool{}
	for _, v := range oldSessionsOf(s, user) {
		own[v] = true
	}
	n := 0
	for _, p := range s.dir.pickups {
		if p.target == user || own[p.session] {
			n++
		}
	}
	return n
}

func sideWorld(t *testing.T, seed uint64) (world, []string, []identity.Verifier) {
	f := setup(t, SeedSample).ready()
	rng := rand.New(rand.NewPCG(seed, 323))
	users := []string{SampleAdmin, SampleAdmin2, SampleAdmin3, "nobody", ""}
	logins := []string{AdminLogin, Admin2Login, Admin3Login}
	sessions := []identity.Verifier{}
	for i := 0; i < 3+rng.IntN(8); i++ {
		login := logins[rng.IntN(len(logins))]
		f.advance(time.Duration(rng.IntN(3)) * time.Second)
		f.as(login)
		for v, found := range f.s.sessions.byID {
			if found.login == login && !containsVerifier(sessions, v) {
				sessions = append(sessions, v)
			}
		}
	}
	sessions = append(sessions, verifierOf([]byte("gone")))
	for i := 0; i < 5+rng.IntN(20); i++ {
		f.s.dir.pickups["p"+strconv.Itoa(i)] = pickup{target: users[rng.IntN(len(users))], session: sessions[rng.IntN(len(sessions))], expires: f.s.clock().Add(time.Hour)}
	}
	return f, users, sessions
}

func containsVerifier(list []identity.Verifier, v identity.Verifier) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func TestImpactIndexesEqualThePerPersonScans(t *testing.T) {
	for seed := uint64(0); seed < 40; seed++ {
		f, users, _ := sideWorld(t, seed)
		side := f.s.sideIndex()
		for _, user := range users {
			login, _ := f.s.dir.loginOf(user)
			if got, want := side.pickupsOf(user, login), oldPickupsOf(f.s, user); got != want {
				t.Fatalf("seed %d %s pickups %d; want %d", seed, user, got, want)
			}
			if got, want := len(side.sessions[login]), len(oldSessionsOf(f.s, user)); got != want {
				t.Fatalf("seed %d %s sessions %d; want %d", seed, user, got, want)
			}
		}
		for _, store := range proposalStores {
			tally := store.tally(f.s)
			for _, user := range users {
				if store.count(f.s, user) != tally[user] {
					t.Fatalf("%s %s", store.name, user)
				}
			}
		}
	}
}

func TestCleanupEndsTheSameSessionsAndPickupsAsBefore(t *testing.T) {
	for seed := uint64(0); seed < 40; seed++ {
		f, users, _ := sideWorld(t, seed)
		denyAt := f.s.clock().Add(-time.Second)
		cleaned := map[string]bool{users[seed%3]: true, users[(seed+1)%3]: seed%2 == 0, "nobody": true}
		ended := map[identity.Verifier]bool{}
		for user, on := range cleaned {
			if !on {
				continue
			}
			for _, v := range oldSessionsOf(f.s, user) {
				if found, ok := f.s.sessions.get(v); ok && !found.created.After(denyAt) {
					ended[v] = true
				}
			}
		}
		keep := map[string]bool{}
		for ref, p := range f.s.dir.pickups {
			keep[ref] = !cleaned[p.target] && !ended[p.session]
		}
		side := f.s.cleanupSide()
		for user, on := range cleaned {
			if on {
				f.s.cleanPerson(side, user, denyAt)
			}
		}
		for ref, want := range keep {
			if _, got := f.s.dir.pickups[ref]; got != want {
				t.Fatalf("seed %d pickup %s kept %v; want %v", seed, ref, got, want)
			}
		}
		for v := range f.s.sessions.byID {
			if ended[v] {
				t.Fatalf("seed %d: a session that should have ended is open", seed)
			}
		}
		for v := range ended {
			if _, ok := f.s.sessions.get(v); ok {
				t.Fatalf("seed %d: an ended session remains", seed)
			}
		}
	}
}
