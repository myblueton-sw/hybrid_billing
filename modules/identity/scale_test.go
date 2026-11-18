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
		f.r.putUser(id, &user{kind: Human, state: Active, epoch: 1})
		f.r.putMembership("w1", id, &membership{role: Member, state: MembershipActive, version: 1, joined: 1, from: start})
		targets = append(targets, Target{User: id, Version: 1})
	}
	f.r.putMembership("w2", "user-0000000000000000", &membership{role: Member, state: MembershipActive, version: 1})
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

func TestScaleListingsOfOneHundredThousand(t *testing.T) {
	if testing.Short() {
		t.Skip("scale run")
	}
	f := setup(t)
	ids := []string{}
	for i := 0; i < 100000; i++ {
		id := fmt.Sprintf("user-%016x", uint64(i)*0x9e3779b97f4a7c15)
		f.r.putUser(id, &user{kind: Human, state: Active, epoch: 1})
		f.r.putMembership("w1", id, &membership{role: Member, state: MembershipActive, version: 1, joined: 1, from: start})
		ids = append(ids, id)
	}
	admin := f.login("root")
	deep := f.r.userIndex[90000]
	timed := func(label string, runs int, step func()) {
		began := time.Now()
		for i := 0; i < runs; i++ {
			step()
		}
		t.Logf("%s: %v per call", label, time.Since(began)/time.Duration(runs))
	}
	timed("indexed Users page of 2 at 90%", 100, func() { f.must(second(f.in.Users(deep, 2))) })
	timed("indexed Members page of 2 at 90%", 100, func() { f.must(second(f.w.Members(admin, "w1", deep, 2))) })
	timed("People of 256", 100, func() { f.must(second(f.w.People(admin, "w1", ids[:PeopleLimit]))) })
	f.r.mu.Lock()
	timed("sorting oracle Users page at 90% (the replaced path)", 5, func() { sortedUsers(f.r, deep, 2) })
	timed("sorting oracle Members page at 90% (the replaced path)", 5, func() { sortedMembers(f.r, "w1", deep, 2) })
	f.r.mu.Unlock()
}
