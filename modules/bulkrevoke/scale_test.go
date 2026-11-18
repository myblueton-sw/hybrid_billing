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
	f := setup(t, Limits{MaxTargets: 100000, Pending: 2, Kept: 2, Lifetime: time.Hour})
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
