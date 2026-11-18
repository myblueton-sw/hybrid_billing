package server

import (
	"sync"
	"testing"
)

func TestBulkConcurrency(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.withMember()
	id, digest := f.prepared(t, root, "suspend_membership", SampleMember, SampleAdmin3)
	revision := f.revision(root)
	var wg sync.WaitGroup
	results := make([]reply, 4)
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i] = f.runWith(root, revision, "bulk_revocation.execute", id, map[string]string{"digest": digest})
		}(i)
	}
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			f.bulkRead(root, "")
			f.bulkRead(root, "/"+id+"/items")
			f.list(root, "")
		}()
	}
	wg.Add(2)
	go func() { defer wg.Done(); f.s.drainCleanup() }()
	go func() { defer wg.Done(); f.sweep() }()
	wg.Wait()
	f.s.drainCleanup()
	accepted := 0
	for _, r := range results {
		if r.status == 202 {
			accepted++
		}
	}
	if accepted != 1 || len(f.records(t, "bulk_revocation_job")) != 1 {
		t.Errorf("executions %d", accepted)
	}
	if job := f.job(root, id); job["job_state"] != "succeeded" {
		t.Errorf("job %+v", job)
	}
}
