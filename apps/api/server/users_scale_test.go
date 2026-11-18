package server

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/bulkrevoke"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func scaleWorld(t testing.TB, n int) (world, *population) {
	t.Helper()
	f := setupWith(t, func(c *Config) {
		c.Identity.PendingInvitations = 10
		c.Access = DefaultAccessSettings()
		c.Access.BulkMaxTargets, c.Access.BulkKept, c.Access.BulkPending, c.Access.BulkRetainedMax = 5000, 100, 200, 2000000
	})
	p := newPopulation(f)
	began := time.Now()
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("user-%016x", uint64(i)*0x9e3779b97f4a7c15)
		login := fmt.Sprintf("person%06d@example.invalid", i)
		p.member(id, login, identity.Member)
	}
	t.Logf("built %d people with memberships in %v", n, time.Since(began))
	return f, p
}

func measure(t testing.TB, label string, runs int, step func()) {
	t.Helper()
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	began := time.Now()
	for i := 0; i < runs; i++ {
		step()
	}
	elapsed := time.Since(began) / time.Duration(runs)
	runtime.ReadMemStats(&after)
	t.Logf("%s: %v per call, %d allocations, %d KiB per call", label, elapsed, (after.Mallocs-before.Mallocs)/uint64(runs), (after.TotalAlloc-before.TotalAlloc)/uint64(runs)/1024)
}

func TestScaleUsersRead(t *testing.T) {
	if testing.Short() {
		t.Skip("scale run")
	}
	for _, n := range []int{10000, 100000} {
		f, p := scaleWorld(t, n)
		root := f.as(AdminLogin)
		deep := "a:" + f.s.dir.order[n*9/10]
		counted := map[string]int{}
		f.s.fault = func(step string) bool { counted[step]++; return false }
		measure(t, fmt.Sprintf("%d first page", n), 20, func() { f.list(root, "?limit=50") })
		measure(t, fmt.Sprintf("%d deep page at 90%%", n), 20, func() { f.list(root, "?limit=50&after="+deep) })
		measure(t, fmt.Sprintf("%d sparse filter exhausting the budget of %d", n, f.s.cfg.Access.UserFilterScan), 20, func() {
			got := f.list(root, "?status=departed")
			if got.body["scan_stopped"] != true {
				t.Fatal("the sparse filter did not stop")
			}
		})
		if counted["roster"] != 0 || counted["users"] != 0 || counted["people"] == 0 {
			t.Errorf("a page called the whole-list path: %v", counted)
		}
		f.s.fault = nil
		f.s.caller = p.admin
		measure(t, fmt.Sprintf("%d roster()", n), 3, func() {
			if _, err := f.s.roster(p.admin); err != nil {
				t.Fatal(err)
			}
		})
		measure(t, fmt.Sprintf("%d everyone()", n), 3, func() {
			if _, err := f.s.everyone(); err != nil {
				t.Fatal(err)
			}
		})
		f.s.caller = identity.Verifier{}
		measure(t, fmt.Sprintf("%d sweepRoles per command", n), 3, func() { f.s.sweepRoles() })
		bulkHistory(t, f, 100, 5000)
		measure(t, fmt.Sprintf("%d first page with 100 executed jobs of 5000 targets", n), 20, func() { f.list(root, "?limit=50") })
	}
}

func bulkHistory(t testing.TB, f world, jobs, size int) {
	t.Helper()
	order := f.s.dir.order
	began := time.Now()
	for j := 0; j < jobs; j++ {
		users, targets := []string{}, []bulkrevoke.Target{}
		for i := 0; i < size; i++ {
			id := order[(j*size+i)%len(order)]
			users = append(users, id)
			targets = append(targets, bulkrevoke.Target{User: id, Login: id, Role: "member", Version: 1, Epoch: 1, UserState: "active", MembershipState: "active"})
		}
		owner := fmt.Sprintf("owner-%d", j)
		draft, err := f.s.bulk.Add(SampleWorkspace, owner, "", users)
		if err != nil {
			t.Fatal(err)
		}
		v, _, err := f.s.bulk.Prepare(SampleWorkspace, owner, draft.ID, bulkrevoke.SuspendMembership, "r1", targets, nil, bulkrevoke.Impact{People: size})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.s.bulk.Execute(SampleWorkspace, v.ID, v.Digest, "executor", 1, nil); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("recorded %d executed jobs of %d targets in %v", jobs, size, time.Since(began))
}

func benchmarkUsers(b *testing.B, n int, query string) {
	f, _ := scaleWorld(b, n)
	root := f.as(AdminLogin)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.list(root, query)
	}
}

func BenchmarkUsersPage10000(b *testing.B)      { benchmarkUsers(b, 10000, "?limit=50") }
func BenchmarkUsersPage100000(b *testing.B)     { benchmarkUsers(b, 100000, "?limit=50") }
func BenchmarkUsersFiltered10000(b *testing.B)  { benchmarkUsers(b, 10000, "?status=departed") }
func BenchmarkUsersFiltered100000(b *testing.B) { benchmarkUsers(b, 100000, "?status=departed") }
