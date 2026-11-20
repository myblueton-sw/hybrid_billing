package identity

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestGuardReinstatementPreservesDryRunAndVersions(t *testing.T) {
	f := setup(t)
	f.member("ann", Member)
	root := f.login("root")
	f.must(f.w.SuspendMembership(root, "w1", "ann", 1))
	before := *f.r.member("w1", "ann")
	generation := f.r.generation
	f.r.mu.Lock()
	err := f.r.reinstateMembership(root, "w1", "ann", before.version, *f.now, true)
	f.r.mu.Unlock()
	f.must(err)
	if *f.r.member("w1", "ann") != before || f.r.generation != generation {
		t.Fatal("dry run changed membership or generation")
	}
	f.want("stale reinstatement", f.w.ReinstateMembership(root, "w1", "ann", 1), ErrConflict)
	if *f.r.member("w1", "ann") != before || f.r.generation != generation {
		t.Fatal("refusal changed membership or generation")
	}
	f.must(f.w.ReinstateMembership(root, "w1", "ann", before.version))
	after := f.r.member("w1", "ann")
	if after.state != MembershipActive || after.version != before.version+1 || f.r.generation != generation+1 {
		t.Fatal("successful reinstatement did not advance once")
	}
}

func TestGuardConcurrentAdministratorRemovalRetainsOne(t *testing.T) {
	f := setup(t)
	f.member("ann", Administrator)
	f.member("member", Member)
	ids := []string{"root", "ann"}
	actors := []Verifier{f.login("root"), f.login("ann")}
	results := contend(2, func(i int) error { return f.w.RemoveMembership(actors[i], "w1", ids[i], 1) })
	succeeded, refused := 0, 0
	for _, err := range results {
		if err == nil {
			succeeded++
		} else if errors.Is(err, ErrLastAdministrator) {
			refused++
		} else {
			t.Fatal(err)
		}
	}
	if succeeded != 1 || refused != 1 || len(f.administrators("w1")) != 1 {
		t.Fatalf("succeeded=%d refused=%d", succeeded, refused)
	}
}

func TestGuardScaleReinstateOneHundredThousand(t *testing.T) {
	if testing.Short() {
		t.Skip("scale run")
	}
	f := setup(t)
	const count = 100000
	for i := 0; i < count; i++ {
		id := fmt.Sprintf("guard-%06d", i)
		f.r.putUser(id, &user{kind: Human, state: Active, epoch: 1})
		f.r.putMembership("w1", id, &membership{role: Member, state: MembershipSuspended, version: 1, joined: 1, from: start})
	}
	f.r.putMembership("w2", "guard-000000", &membership{role: Member, state: MembershipSuspended, version: 7})
	actor := f.login("root")
	generation := f.r.generation
	began := time.Now()
	for i := 0; i < count; i++ {
		f.must(f.w.ReinstateMembership(actor, "w1", fmt.Sprintf("guard-%06d", i), 1))
	}
	elapsed := time.Since(began)
	for i := 0; i < count; i++ {
		member := f.r.member("w1", fmt.Sprintf("guard-%06d", i))
		if member.state != MembershipActive || member.version != 2 {
			t.Fatalf("member %d was not reinstated exactly once", i)
		}
	}
	if f.r.generation != generation+count {
		t.Fatal("unexpected generation delta")
	}
	if member := f.r.member("w2", "guard-000000"); member.state != MembershipSuspended || member.version != 7 {
		t.Fatal("foreign workspace changed")
	}
	t.Logf("100000 public reinstatements: %v", elapsed)
}

func BenchmarkGuardOrdinaryMember(b *testing.B) {
	for _, size := range []int{100, 10000, 100000} {
		r := guardWorld(&membership{role: Member, state: MembershipSuspended}, &user{kind: Human, state: Active}, 1, 0)
		for i := 0; i < size; i++ {
			id := fmt.Sprintf("member-%d", i)
			r.users[id] = &user{kind: Human, state: Active}
			r.memberships["w1"][id] = &membership{role: Member, state: MembershipActive}
		}
		after := &membership{role: Member, state: MembershipActive}
		for _, baseline := range []bool{true, false} {
			name := "optimized"
			if baseline {
				name = "original"
			}
			b.Run(fmt.Sprintf("%d/%s", size, name), func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					var refused bool
					if baseline {
						refused = originalStrands(r, "w1", "target", after, start)
					} else {
						refused = r.strands("w1", "target", after, start)
					}
					if refused {
						b.Fatal("ordinary member unexpectedly refused")
					}
				}
			})
		}
	}
}

func TestGuardLastAdministratorRefusalsDoNotWrite(t *testing.T) {
	f := setup(t)
	f.member("ann", Member)
	root := f.login("root")
	before := *f.r.member("w1", "root")
	generation := f.r.generation
	for name, run := range map[string]func() error{
		"demote":  func() error { return f.w.ChangeRole(root, "w1", "root", Member, 1) },
		"suspend": func() error { return f.w.SuspendMembership(root, "w1", "root", 1) },
		"remove":  func() error { return f.w.RemoveMembership(root, "w1", "root", 1) },
	} {
		f.want(name, run(), ErrLastAdministrator)
		if *f.r.member("w1", "root") != before || f.r.generation != generation {
			t.Fatalf("%s changed state on refusal", name)
		}
	}
}
