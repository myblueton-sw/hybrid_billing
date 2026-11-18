package bulkrevoke

import (
	"fmt"
	"testing"
	"time"
)

func TestScaleManifestOfOneHundredThousand(t *testing.T) {
	if testing.Short() {
		t.Skip("scale run")
	}
	f := setup(t, Limits{MaxTargets: 100000, MaxMatching: 100000, Pending: 2, Kept: 2, Lifetime: time.Hour})
	users, targets := []string{}, []Target{}
	for i := 0; i < 100000; i++ {
		id := fmt.Sprintf("user-%016x", i)
		users, targets = append(users, id), append(targets, target(id))
	}
	began := time.Now()
	d := f.add("ann", "", users...)
	v, _, err := f.r.Prepare("w1", "ann", d.ID, SuspendMembership, "r1", targets, nil, Impact{})
	if err != nil {
		t.Fatal(err)
	}
	v, err = f.r.Execute("w1", v.ID, v.Digest, "bob", 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("IAM-07 manifest: stage, prepare and execute of 100000 targets in %v", time.Since(began))
	c := v.Counts
	if c.Targets != 100000 || c.Targets != c.Succeeded+c.Held+c.Failed+c.Unprocessed {
		t.Errorf("counts %+v", c)
	}
}

func executeTimed(t *testing.T, f fixture, owner string, size, offset int) time.Duration {
	users, targets := []string{}, []Target{}
	for i := 0; i < size; i++ {
		id := fmt.Sprintf("user-%06d", (offset+i)%100000)
		users, targets = append(users, id), append(targets, target(id))
	}
	d := f.add(owner, "", users...)
	v, _, err := f.r.Prepare("w1", owner, d.ID, SuspendMembership, "r1", targets, nil, Impact{})
	if err != nil {
		t.Fatal(err)
	}
	began := time.Now()
	v, err = f.r.Execute("w1", v.ID, v.Digest, "bob", 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	took := time.Since(began)
	_ = f.r.Header("w1", v.ID, Receipt{Sequence: 1})
	for _, item := range v.Items {
		_ = f.r.Record("w1", v.ID, []int{item.Index}, []Receipt{{Sequence: 2}})
		_ = f.r.Cleaning("w1", v.ID, item.Index)
		_ = f.r.Cleaned("w1", v.ID, item.Index, nil)
	}
	return took
}

func TestScaleIndexPerExecute(t *testing.T) {
	if testing.Short() {
		t.Skip("scale run")
	}
	f := setup(t, Limits{MaxTargets: 100000, MaxMatching: 100000, Pending: 200, Kept: 100, Lifetime: time.Hour})
	var total time.Duration
	for j := 0; j < 100; j++ {
		total += executeTimed(t, f, fmt.Sprintf("o%d", j), 5000, j*5000)
	}
	t.Logf("100 executes of 5000 with up to 100 kept: %v in total", total)
	t.Logf("execute of 5000 with 100 kept jobs of 5000 (one evicted): %v", executeTimed(t, f, "p1", 5000, 3))
	for i := 0; i < 3; i++ {
		t.Logf("execute of 100000 with 100 kept jobs (run %d): %v", i+1, executeTimed(t, f, fmt.Sprintf("q%d", i), 100000, i))
	}
	t.Logf("execute of 5000 with three 100000-item jobs kept: %v", executeTimed(t, f, "p2", 5000, 11))
}
