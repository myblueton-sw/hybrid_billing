package server

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func TestSessionQueryIndexMatchesIndependentLifecycleModel(t *testing.T) {
	store := newSessions()
	rng := rand.New(rand.NewSource(304))
	owners := []sessionOwner{{"person-a", "workspace", 1}, {"person-b", "workspace", 1}, {"person-a", "workspace", 2}}
	model := map[identity.Verifier]session{}
	for step := 0; step < 400; step++ {
		var verifier identity.Verifier
		verifier[0] = byte(rng.Intn(60) + 1)
		found, exists := model[verifier]
		switch rng.Intn(5) {
		case 0, 1:
			owner := owners[rng.Intn(len(owners))]
			at := start.Add(time.Duration(rng.Intn(20)) * time.Second)
			login := fmt.Sprintf("login-%s-%d", owner.User, owner.Epoch)
			if _, err := store.put(verifier, owner.Workspace, login, at, 1); err != nil {
				t.Fatal(err)
			}
			if !store.bind(verifier, owner, at.Add(sessionIdle)) {
				t.Fatal("canonical bind failed")
			}
			model[verifier] = session{verifier: verifier, owner: owner, handle: handleOf(verifier), created: at, authenticated: at, idleUntil: at.Add(sessionIdle)}
		case 2:
			store.drop(verifier)
			delete(model, verifier)
		case 3:
			if exists {
				at := start.Add(time.Duration(step) * time.Second)
				store.reauthenticated(verifier, at)
				found.authenticated = at
				model[verifier] = found
			}
		case 4:
			if exists {
				at := start.Add(time.Hour + time.Duration(step)*time.Second)
				store.touch(verifier, at)
				if at.After(found.idleUntil) {
					found.idleUntil = at
					model[verifier] = found
				}
			}
		}
		for _, owner := range owners {
			want := []session{}
			for _, row := range model {
				if row.owner == owner {
					want = append(want, row)
				}
			}
			sort.Slice(want, func(i, j int) bool {
				if want[i].created.Equal(want[j].created) {
					return want[i].handle < want[j].handle
				}
				return want[i].created.After(want[j].created)
			})
			state := store.indexState(owner, start, time.Hour)
			if state.Count != len(want) {
				t.Fatalf("step%d owner%v count%d want%d", step, owner, state.Count, len(want))
			}
			if len(want) == 0 {
				continue
			}
			got := []session{}
			after := ""
			for {
				batch, more, ok := store.indexed(owner, state.Version, after, 3)
				if !ok || len(batch) > 3 {
					t.Fatal("invalid bounded index read")
				}
				got = append(got, batch...)
				if !more {
					break
				}
				if len(batch) == 0 {
					t.Fatal("index did not advance")
				}
				after = batch[len(batch)-1].handle
			}
			if len(got) != len(want) {
				t.Fatal("index omitted rows")
			}
			for i := range want {
				if got[i].verifier != want[i].verifier || !got[i].idleUntil.Equal(want[i].idleUntil) || !got[i].authenticated.Equal(want[i].authenticated) {
					t.Fatalf("step%d indexed row differs", step)
				}
			}
			deadline := time.Time{}
			for _, row := range want {
				for _, at := range []time.Time{row.idleUntil, row.authenticated.Add(time.Hour + time.Nanosecond)} {
					if at.After(start) && (deadline.IsZero() || at.Before(deadline)) {
						deadline = at
					}
				}
			}
			if !state.Deadline.Equal(deadline) {
				t.Fatalf("step%d deadline%s want%s", step, state.Deadline, deadline)
			}
		}
	}
	store.prune(start.Add(24 * time.Hour))
	if len(store.owners) != 0 || len(store.byID) != 0 {
		t.Fatal("pruning retained owner index entries")
	}
}

func TestSessionQueryIndexRejectsForeignBindingsAndObsoleteGeneration(t *testing.T) {
	store := newSessions()
	var verifier identity.Verifier
	verifier[0] = 1
	if _, err := store.put(verifier, "workspace", "login", start, 1); err != nil {
		t.Fatal(err)
	}
	if store.bind(verifier, sessionOwner{"person", "foreign", 1}, start.Add(time.Hour)) {
		t.Fatal("foreign workspace bound")
	}
	owner := sessionOwner{"person", "workspace", 1}
	if !store.bind(verifier, owner, start.Add(time.Hour)) {
		t.Fatal("bind failed")
	}
	if store.bind(verifier, sessionOwner{"other", "workspace", 1}, start.Add(time.Hour)) {
		t.Fatal("canonical identity rebound")
	}
	state := store.indexState(owner, start, 0)
	store.reauthenticated(verifier, start.Add(time.Minute))
	if _, _, ok := store.indexed(owner, state.Version, "", 1); ok {
		t.Fatal("obsolete index generation admitted")
	}
	store.take(verifier)
	if len(store.owners) != 0 {
		t.Fatal("empty owner index retained")
	}
}
