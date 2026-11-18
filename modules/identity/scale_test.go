package identity

import (
	"fmt"
	"testing"
	"time"
)

func TestScaleBulkDenyOfOneHundredThousand(t *testing.T) {
	if testing.Short() {
		t.Skip("scale run")
	}
	f := setup(t)
	targets := []Target{}
	for i := 0; i < 100000; i++ {
		id := fmt.Sprintf("user-%016x", i)
		f.r.users[id] = &user{kind: Human, state: Active, epoch: 1}
		f.r.memberships["w1"][id] = &membership{role: Member, state: MembershipActive, version: 1, joined: 1, from: start}
		targets = append(targets, Target{User: id, Version: 1})
	}
	f.r.memberships["w2"] = map[string]*membership{"user-0000000000000000": {role: Member, state: MembershipActive, version: 1}}
	began := time.Now()
	f.must(f.w.SuspendMemberships(f.login("root"), "w1", targets, BulkRevocation))
	t.Logf("IAM-07 identity deny of 100000 memberships held the registry lock for %v", time.Since(began))
	if f.r.memberships["w2"]["user-0000000000000000"].state != MembershipActive {
		t.Error("another workspace changed")
	}
	for _, target := range targets[:10] {
		if f.r.memberships["w1"][target.User].state != MembershipSuspended {
			t.Fatal("a target was not suspended")
		}
	}
}
