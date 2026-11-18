package server

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/jobs"
)

func taggedLogin(i int) string {
	tag := "-xall"
	if i < 20000 {
		tag = "-x20-xall"
	}
	if i < 5000 {
		tag = "-x5-x20-xall"
	}
	return fmt.Sprintf("p%06d%s@example.invalid", i, tag)
}

func matchingScaleWorld(t *testing.T, n int) (world, *population) {
	t.Helper()
	f := setupWith(t, func(c *Config) {
		c.Identity.PendingInvitations = 10
		c.Access = DefaultAccessSettings()
		c.Access.BulkMaxTargets, c.Access.BulkMaxMatching, c.Access.BulkKept, c.Access.BulkPending, c.Access.BulkRetainedMax = 5000, 100000, 10, 20, 20000000
		c.Access.BulkMinutes, c.Access.LedgerMaxMiB = 1440, 4096
	})
	p := newPopulation(f)
	began := time.Now()
	for i := 0; i < n; i++ {
		p.member(fmt.Sprintf("user-%016x", uint64(i)*0x9e3779b97f4a7c15), taggedLogin(i), identity.Member)
	}
	t.Logf("built %d people with memberships in %v", n, time.Since(began))
	return f, p
}

func heap() uint64 {
	var m runtime.MemStats
	runtime.GC()
	runtime.GC()
	runtime.ReadMemStats(&m)
	return m.HeapAlloc
}

func dataBytes(dir string) int64 {
	var total int64
	_ = filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			if info, e := d.Info(); e == nil {
				total += info.Size()
			}
		}
		return nil
	})
	return total
}

type stamps struct {
	mu    sync.Mutex
	deny  time.Time
	first time.Time
	syncs int
	clean time.Time
}

func (s *stamps) hook(step string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	switch step {
	case "bulk-deny":
		s.deny, s.first, s.syncs, s.clean = now, time.Time{}, 0, time.Time{}
	case "ledger-sync":
		if !s.deny.IsZero() {
			s.syncs++
			if s.first.IsZero() {
				s.first = now
			}
		}
	case "cleanup-create":
		s.clean = now
	}
	return false
}

func sideData(t *testing.T, f world, ids []string) {
	t.Helper()
	now := f.s.clock()
	for i, id := range ids {
		login, _ := f.s.dir.loginOf(id)
		if i%10 == 0 && len(f.s.sessions.byID) < 9900 {
			if _, _, err := f.s.openFor(LoginTriple(login), SampleWorkspace, false); err != nil {
				t.Fatal(err)
			}
		}
		if i%100 == 0 {
			f.s.dir.pickups["scale-"+strconv.Itoa(i)] = pickup{target: id, expires: now.Add(time.Hour)}
			f.s.plans.byKey["scale-"+strconv.Itoa(i)] = &planMeta{proposer: id}
			f.s.ev.entries = append(f.s.ev.entries, &evidenceEntry{proposer: id, state: entryPending})
		}
	}
}

func (f world) timed(label string, call func() reply) reply {
	f.t.Helper()
	began := time.Now()
	got := call()
	f.t.Logf("%s: %v (status %d)", label, time.Since(began), got.status)
	return got
}

func (f world) drainTimed(label string) {
	f.t.Helper()
	began := time.Now()
	longest, tasks := time.Duration(0), 0
	for {
		at := time.Now()
		f.s.domain.Lock()
		leases, err := f.s.jobs.Lease(cleanupWorker, []string{cleanupKind}, 1)
		if err != nil || len(leases) == 0 {
			f.s.domain.Unlock()
			break
		}
		f.s.cleanTask(leases[0])
		f.s.domain.Unlock()
		longest, tasks = max(longest, time.Since(at)), tasks+1
	}
	f.t.Logf("%s: %d cleanup tasks drained in %v, longest lock hold per task %v", label, tasks, time.Since(began), longest)
}

func TestScaleFilterListsOverHTTP(t *testing.T) {
	if testing.Short() {
		t.Skip("scale run")
	}
	f, p := matchingScaleWorld(t, 100000)
	root := f.as(AdminLogin)
	other := f.as(Admin2Login)
	ids := slices.Clone(f.s.dir.order)
	sideData(t, f, ids)
	t.Logf("side data: %d sessions, %d pickups", len(f.s.sessions.byID), len(f.s.dir.pickups))
	for _, scan := range []int{20000, 100000} {
		f.s.cfg.Access.UserCountScan = scan
		measure(t, fmt.Sprintf("count read of everyone with user_count_scan %d", scan), 3, func() {
			if got := f.count(root, "?q=xall"); got.status != 200 {
				t.Fatal(got.raw)
			}
		})
		done := make(chan time.Duration, 1)
		go func() {
			began := time.Now()
			f.count(root, "?q=xall")
			done <- time.Since(began)
		}()
		time.Sleep(2 * time.Millisecond)
		began := time.Now()
		staged := f.stage(other, "", ids[0])
		write := time.Since(began)
		t.Logf("count with user_count_scan %d took %v; a write command started during it answered after %v (status %d)", scan, <-done, write, staged.status)
		f.discard(other, field(staged, "draft"))
	}
	marks := &stamps{}
	f.s.fault = marks.hook
	for _, c := range []struct {
		size  int
		query string
	}{{5000, "q=x5-"}, {20000, "q=x20-"}, {100000, "q=xall"}} {
		swept(f, p, root)
		base := heap()
		fixed := f.timed(fmt.Sprintf("%d add_matching", c.size), func() reply { return f.match(root, c.query) })
		if fixed.status != 200 || field(fixed, "matched") != strconv.Itoa(c.size) {
			t.Fatalf("fix %s", fixed.raw)
		}
		draftHeap := heap()
		prepared := f.timed(fmt.Sprintf("%d prepare", c.size), func() reply { return f.prepare(root, field(fixed, "draft"), "suspend_membership") })
		if prepared.status != 200 {
			t.Fatalf("prepare %s", prepared.raw)
		}
		openHeap := heap()
		t.Logf("%d bytes per draft ID %d, per open prepared person %d", c.size, int64(draftHeap-base)/int64(c.size), int64(openHeap-base)/int64(c.size))
		if c.size == 100000 {
			f.discard(root, field(prepared, "bulk"))
			t.Logf("%d bytes per ended prepared person %d", c.size, int64(heap()-base)/int64(c.size))
			base = heap()
			swept(f, p, root)
			fixed = f.match(root, c.query)
			prepared = f.prepare(root, field(fixed, "draft"), "suspend_membership")
		}
		ledgerBefore := dataBytes(f.s.cfg.Data)
		read := make(chan time.Duration, 1)
		go func() {
			time.Sleep(20 * time.Millisecond)
			began := time.Now()
			f.list(other, "?limit=50")
			read <- time.Since(began)
		}()
		executed := f.timed(fmt.Sprintf("%d execute", c.size), func() reply { return f.execute(root, field(prepared, "bulk"), field(prepared, "digest")) })
		if executed.status != 202 {
			t.Fatalf("execute %s", executed.raw)
		}
		marks.mu.Lock()
		t.Logf("%d phase A (deny to first ledger sync) %v; phase B (ledger header and items) %v with %d syncs", c.size, marks.first.Sub(marks.deny), marks.clean.Sub(marks.first), marks.syncs)
		marks.mu.Unlock()
		t.Logf("%d a users read issued during execute answered after %v", c.size, <-read)
		t.Logf("%d ledger bytes per item %d", c.size, (dataBytes(f.s.cfg.Data)-ledgerBefore)/int64(c.size))
		job := f.job(root, field(prepared, "bulk"))
		counts := job["counts"].(map[string]any)
		if whole(counts["targets"])+whole(counts["excluded"]) != c.size || whole(counts["targets"]) != whole(counts["unprocessed"])+whole(counts["succeeded"])+whole(counts["held"])+whole(counts["failed"]) {
			t.Fatalf("counts %+v", counts)
		}
		f.drainTimed(fmt.Sprintf("%d", c.size))
		if j, _ := f.s.jobs.Job(SampleWorkspace, field(prepared, "bulk")); j.State != jobs.Succeeded {
			t.Errorf("%d cleanup job %s", c.size, j.State)
		}
		t.Logf("%d bytes per retained executed item after cleanup %d", c.size, int64(heap()-base)/int64(c.size))
		measure(t, fmt.Sprintf("jobs list read with the %d list kept", c.size), 3, func() { f.bulkRead(root, "?limit=50") })
		measure(t, fmt.Sprintf("job detail read of the %d list", c.size), 3, func() { f.bulkRead(root, "/"+field(prepared, "bulk")) })
		measure(t, fmt.Sprintf("users first page with the %d list kept", c.size), 5, func() { f.list(root, "?limit=50") })
		reinstated := 0
		members, err := f.s.roster(p.admin)
		p.must(err)
		for id, m := range members {
			if m.State == identity.MembershipSuspended {
				p.must(f.s.identity.Workspace().ReinstateMembership(p.admin, SampleWorkspace, id, m.Version))
				reinstated++
			}
		}
		p.refresh()
		t.Logf("%d reinstated for the next size: %d", c.size, reinstated)
	}
	f.s.fault = nil
}
