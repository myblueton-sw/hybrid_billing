package identity

import (
	"errors"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

func sameError(a, b error) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	for _, kind := range []error{ErrInvalid, ErrRefused, ErrConflict, ErrLastAdministrator, ErrLimit, ErrUnavailable} {
		if errors.Is(a, kind) != errors.Is(b, kind) {
			return false
		}
	}
	return ReasonOf(a) == ReasonOf(b)
}

func (f fixture) snapshot() (uint64, []MemberInfo) {
	f.t.Helper()
	f.r.mu.Lock()
	defer f.r.mu.Unlock()
	out := []MemberInfo{}
	for _, id := range []string{"root", "ann", "bob", "cat"} {
		if m := f.r.memberships["w1"][id]; m != nil {
			out = append(out, MemberInfo{User: id, Role: m.role, State: m.state, Version: m.version, From: m.from, Until: m.until, SuspendedFor: m.suspendedFor,
				UserState: f.r.users[id].state})
		}
	}
	return f.r.generation, out
}

func TestDryRunsAgreeWithTheMutations(t *testing.T) {
	for seed := int64(1); seed <= 60; seed++ {
		r := rand.New(rand.NewSource(seed))
		f := setup(t)
		f.member("ann", Member)
		f.member("bob", Administrator)
		f.member("cat", Member)
		ids := []string{"root", "ann", "bob", "cat"}
		for step := 0; step < 80; step++ {
			f.at(f.now.Add(time.Duration(r.Intn(40)) * time.Minute))
			actor := v(f.next("try"))
			_ = f.b.OpenSession(triple(ids[r.Intn(2)*2]), actor, 30*time.Minute, 8*time.Hour)
			target := ids[r.Intn(len(ids))]
			version := f.r.memberships["w1"][target].version + uint64(r.Intn(4)/3)
			until := time.Time{}
			if r.Intn(3) > 0 {
				until = f.now.Add(time.Duration(r.Intn(180)-30) * time.Minute)
			}
			at := *f.now
			var check, apply func() error
			switch r.Intn(5) {
			case 0:
				check = func() error { return f.w.CheckSetRolePeriod(actor, "w1", target, until, version, at) }
				apply = func() error { return f.w.SetRolePeriod(actor, "w1", target, until, version, at) }
			case 1:
				role := []Role{Member, Administrator}[r.Intn(2)]
				check = func() error { return f.w.CheckChangeRoleUntil(actor, "w1", target, role, until, version, at) }
				apply = func() error { return f.w.ChangeRoleUntil(actor, "w1", target, role, until, version, at) }
			case 2:
				check = func() error { return f.w.CheckReinstateMembership(actor, "w1", target, version, at) }
				apply = func() error { return f.w.ReinstateMembershipAt(actor, "w1", target, version, at) }
			case 3:
				epoch := f.r.users[target].epoch
				check = func() error { return f.in.CheckReinstateUser(operator, target, epoch) }
				apply = func() error { return f.in.ReinstateUser(operator, target, epoch) }
			default:
				check = func() error { return f.w.SuspendMembership(actor, "w1", target, version) }
				apply = func() error { _, err := f.in.ExpireRoles(operator, "w1", *f.now); return err }
				if r.Intn(2) == 0 {
					apply = func() error { return f.in.SuspendUser(operator, target, f.r.users[target].epoch) }
				}
				_ = check()
				_ = apply()
				continue
			}
			generation, before := f.snapshot()
			dry := check()
			if g, after := f.snapshot(); g != generation || !reflect.DeepEqual(after, before) {
				t.Fatalf("seed %d step %d: a dry run changed state", seed, step)
			}
			if got := apply(); !sameError(dry, got) {
				t.Fatalf("seed %d step %d: dry run %v, mutation %v", seed, step, dry, got)
			}
		}
	}
}

func TestRolePeriodsAgainstAModel(t *testing.T) {
	for seed := int64(1); seed <= 40; seed++ {
		r := rand.New(rand.NewSource(seed))
		f := setup(t)
		ids := []string{"ann", "bob", "cat"}
		for _, id := range ids {
			f.member(id, []Role{Member, Administrator}[r.Intn(2)])
		}
		type issued struct {
			answer Answer
			end    time.Time
		}
		answers := []issued{}
		reached := map[int]bool{}
		for step := 0; step < 120; step++ {
			switch r.Intn(6) {
			case 0, 1:
				f.at(f.now.Add(time.Duration(r.Intn(20)) * time.Minute))
			case 2:
				id := ids[r.Intn(len(ids))]
				until := time.Time{}
				if r.Intn(4) > 0 {
					until = f.now.Add(time.Duration(1+r.Intn(60)) * time.Minute)
				}
				_ = f.w.SetRolePeriod(f.login("root"), "w1", id, until, f.r.memberships["w1"][id].version, *f.now)
			case 3:
				expired, err := f.in.ExpireRoles(operator, "w1", *f.now)
				f.must(err)
				for _, e := range expired {
					m := f.r.memberships["w1"][e.User]
					if e.Blocked || m.state != MembershipSuspended || m.suspendedFor != RolePeriodEnded {
						t.Fatalf("seed %d: swept %+v %+v", seed, e, m)
					}
				}
			default:
				id := ids[r.Intn(len(ids))]
				if a, err := f.b.Access(f.login(id), "w1"); err == nil {
					answers = append(answers, issued{answer: a, end: f.r.memberships["w1"][id].until})
				}
			}
			for i, a := range answers {
				if a.end.IsZero() {
					continue
				}
				if !f.now.Before(a.end) {
					reached[i] = true
				}
				if reached[i] && f.b.Recheck(a.answer) == nil {
					t.Fatalf("seed %d: an answer is valid at or after its end", seed)
				}
			}
			if f.r.administrators("w1", *f.now) == 0 {
				t.Fatalf("seed %d: no administrator", seed)
			}
		}
	}
}

func TestWallStepsNeverReopenAnEnd(t *testing.T) {
	f := setup(t)
	f.bounded("ann", Member, start.Add(10*time.Minute))
	answer, err := f.b.Access(f.login("ann"), "w1")
	f.must(err)
	f.at(start.Add(10 * time.Minute))
	f.must(second(f.in.ExpireRoles(operator, "w1", *f.now)))
	f.at(start)
	if f.b.Recheck(answer) == nil {
		t.Error("a swept end reopened after the clock moved back")
	}
	if _, err := f.b.Access(f.login("ann"), "w1"); err == nil {
		t.Error("access after the sweep with the clock moved back")
	}
}
