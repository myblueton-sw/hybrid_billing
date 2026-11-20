package identity

import (
	"fmt"
	"testing"
	"time"
)

func originalStrands(r *Registry, workspace, target string, after *membership, now time.Time) bool {
	before := r.administrators(workspace, now)
	if before == 0 || after != nil && after.role == Administrator && effective(after, now) && r.activeHuman(target) {
		return false
	}
	remaining := before
	if r.activeAdministrator(workspace, target, now) {
		remaining--
	}
	if remaining > 0 {
		return false
	}
	if after != nil && after.state != MembershipRemoved {
		return true
	}
	return r.others(workspace, target)
}

func guardMemberships() []*membership {
	rows := []*membership{nil}
	for _, role := range []Role{Member, Administrator, ServiceRole} {
		for _, state := range []MembershipState{MembershipActive, MembershipSuspended, MembershipRemoved} {
			for _, until := range []time.Time{{}, start.Add(-time.Nanosecond), start, start.Add(time.Nanosecond)} {
				rows = append(rows, &membership{role: role, state: state, until: until, from: start, version: 1})
			}
		}
	}
	return rows
}

func guardWorld(target *membership, account *user, otherAdmins, otherPresence int) *Registry {
	r := &Registry{users: map[string]*user{}, memberships: map[string]map[string]*membership{"w1": {}, "w2": {}}}
	if account != nil {
		r.users["target"] = account
	}
	if target != nil {
		r.memberships["w1"]["target"] = target
	}
	for i := 0; i < otherAdmins; i++ {
		id := fmt.Sprintf("admin-%d", i)
		r.users[id] = &user{kind: Human, state: Active}
		r.memberships["w1"][id] = &membership{role: Administrator, state: MembershipActive}
	}
	switch otherPresence {
	case 1, 2:
		r.users["other"] = &user{kind: Human, state: Active}
		state := MembershipActive
		if otherPresence == 2 {
			state = MembershipRemoved
		}
		r.memberships["w1"]["other"] = &membership{role: Member, state: state}
	case 3, 4, 5:
		state, workspace := Active, "w1"
		if otherPresence == 4 {
			state = Departed
		}
		if otherPresence == 5 {
			workspace = "w2"
		}
		r.users["service"] = &user{kind: Service, state: state, workspace: workspace}
	}
	return r
}

func TestGuardMatchesOriginalPredicate(t *testing.T) {
	members := guardMemberships()
	accounts := []*user{nil}
	for _, kind := range []Kind{Human, Service} {
		for _, state := range []UserState{Active, Suspended, Departed} {
			accounts = append(accounts, &user{kind: kind, state: state})
		}
	}
	cases, refusals := 0, 0
	for bi, before := range members {
		for ui, account := range accounts {
			for admins := 0; admins < 3; admins++ {
				for presence := 0; presence < 6; presence++ {
					r := guardWorld(before, account, admins, presence)
					for ai, after := range members {
						want := originalStrands(r, "w1", "target", after, start)
						got := r.strands("w1", "target", after, start)
						cases++
						if want {
							refusals++
						}
						if got != want {
							t.Fatalf("before=%d account=%d admins=%d presence=%d after=%d: got %v want %v", bi, ui, admins, presence, ai, got, want)
						}
					}
				}
			}
		}
	}
	if cases != 172494 || refusals == 0 || refusals == cases {
		t.Fatalf("matrix cases=%d refusals=%d", cases, refusals)
	}
	t.Logf("compared %d cases including %d refusals", cases, refusals)
}

func TestGuardValidityStartAndExpiryBoundaries(t *testing.T) {
	for _, at := range []time.Time{start.Add(-time.Nanosecond), start, start.Add(time.Nanosecond)} {
		for _, from := range []time.Time{start.Add(-time.Nanosecond), start, start.Add(time.Nanosecond)} {
			for _, until := range []time.Time{{}, start.Add(-time.Nanosecond), start, start.Add(time.Nanosecond)} {
				before := &membership{role: Administrator, state: MembershipActive, from: from, until: until}
				r := guardWorld(before, &user{kind: Human, state: Active}, 0, 1)
				for _, after := range guardMemberships() {
					if got, want := r.strands("w1", "target", after, at), originalStrands(r, "w1", "target", after, at); got != want {
						t.Fatalf("at=%s from=%s until=%s got=%v want=%v", at, from, until, got, want)
					}
				}
			}
		}
	}
}
