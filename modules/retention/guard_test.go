package retention

import (
	"errors"
	"math/rand"
	"sort"
	"testing"
	"time"
)

func (f world) guard(from, to string) error {
	return f.r.Refer(Reference{From: from, To: to, Kind: Guards}, ingest)
}

func (f world) remove(id string, stores ...string) time.Time {
	f.t.Helper()
	run := f.plan(f.r.artifacts[id].view.Class, 1, id)
	c, err := f.r.Claim(run, 0, worker)
	if err != nil || len(c.Deleting) != 1 {
		f.t.Fatalf("lease of %s: %+v %v", id, c, err)
	}
	for _, s := range stores {
		f.must(f.r.Report(run, 0, c.Token, s, []string{id}, nil, worker))
	}
	return f.at()
}

func TestGuardShape(t *testing.T) {
	f := setup(t)
	f.put("c1", ClassClaim, 1, one)
	f.put("c2", ClassClaim, 1, one)
	f.put("s1", ClassSource, 1, one)
	f.put("f1", ClassFinal, 1, one)
	f.put("far", ClassSource, 1, []string{"t9"})
	f.put("both", ClassSource, 1, []string{"t1", "t9"})
	for name, pair := range map[string][2]string{"from a source": {"s1", "f1"}, "from a final document": {"f1", "s1"}, "to a claim record": {"c1", "c2"}, "to itself": {"c1", "c1"}, "across workspaces": {"c1", "far"}} {
		if err := f.guard(pair[0], pair[1]); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
	expect(t, "an unknown artifact", f.guard("c1", "nobody"), ErrNotFound)
	expect(t, "an unknown record", f.guard("nobody", "s1"), ErrNotFound)
	if err := f.r.Refer(Reference{From: "c1", To: "s1", Kind: Guards}, ann); err == nil {
		t.Error("a person wrote a guard")
	}
	f.must(f.guard("c1", "s1"))
	f.must(f.guard("c1", "s1"))
	f.must(f.guard("c1", "f1"))
	f.must(f.guard("c2", "s1"))
	f.must(f.guard("c1", "both"))
	if got := f.r.guards["c1"]; len(got) != 3 || len(f.r.guards["c2"]) != 1 || f.r.requiredBy["s1"] != 0 || len(f.r.requires["c1"]) != 0 {
		t.Errorf("guards %v required %d", got, f.r.requiredBy["s1"])
	}
	f.policy(ClassSource, 1, 0)
	run := f.plan(ClassSource, 1, "far")
	f.put("c3", ClassClaim, 1, []string{"t9"})
	f.must(second(f.r.Claim(run, 0, worker)))
	reason(t, "an artifact being deleted", f.guard("c3", "far"), ReasonDeleting)
}

func TestGuardKeepsTheRecordUntilItsSourceIsGone(t *testing.T) {
	f := setup(t)
	f.policy(ClassSource, 90, 0)
	f.policy(ClassClaim, 30, 5)
	f.put("s1", ClassSource, 100, one, "db", "obj")
	f.put("c1", ClassClaim, 100, one)
	f.put("c0", ClassClaim, 100, one)
	f.must(f.guard("c1", "s1"))
	look := func(when string, guarding, due bool, dueAt time.Time, eligible string) {
		t.Helper()
		st, err := f.r.Standing("c1")
		f.must(err)
		pv, _ := f.r.Preview(ClassClaim, "", 10)
		ids := ""
		for _, id := range pv.Eligible {
			ids += id + " "
		}
		wantGuarding := 0
		if guarding && due {
			wantGuarding = 1
		}
		if st.Guarding != guarding || st.Due != due || !st.DueAt.Equal(dueAt) || ids != eligible || pv.Guarding != wantGuarding {
			t.Fatalf("%s: standing %+v, preview %+v; want guarding %v due %v at %v eligible %q", when, st, pv, guarding, due, dueAt, eligible)
		}
	}
	look("the source exists", true, false, time.Time{}, "c0 ")
	if st, _ := f.r.Standing("c0"); st.Guarding || !st.Due || !st.DueAt.Equal(f.at().Add(-65*day)) {
		t.Fatalf("a record that guards nothing %+v", st)
	}
	run := f.plan(ClassClaim, 5, "c0", "c1")
	if v := f.run(run); v.Outcomes["c1"] != OutcomeGuarding || v.Outcomes["c0"] != OutcomePending || v.Counts[OutcomeGuarding] != 1 || f.access("c1") != AccessOK {
		t.Fatalf("a plan over a guarding record %+v", v)
	}
	f.must(second(f.r.Stop(run, cat)))

	src := f.plan(ClassSource, 1, "s1")
	lease, err := f.r.Claim(src, 0, worker)
	f.must(err)
	f.must(f.r.Report(src, 0, lease.Token, "db", []string{"s1"}, nil, worker))
	look("the source is damaged", true, false, time.Time{}, "c0 ")
	f.advance(400 * day)
	look("the source stays damaged for long", true, false, time.Time{}, "c0 ")
	f.advance(time.Hour)
	f.must(f.r.Report(src, 0, lease.Token, "obj", []string{"s1"}, nil, worker))
	deleted := f.at()
	at := deleted.Add(35 * day)
	look("the source is deleted and its backups are alive", true, false, at, "c0 ")
	f.advance(34 * day)
	look("a day before the period ends", true, false, at, "c0 ")
	f.advance(day)
	look("the period has ended and the backups are alive", true, true, at, "c0 ")
	iv, _ := f.r.Impact(Policy{Class: ClassClaim, Period: 1})
	if iv.Impact.Count != 0 {
		t.Fatalf("a record that is due already is not newly due %+v", iv)
	}
	held := f.plan(ClassClaim, 1, "c1")
	if v := f.run(held); v.Outcomes["c1"] != OutcomeGuarding {
		t.Fatalf("a plan while the backups are alive %+v", v)
	}
	f.must(f.r.BackupExpired("db", f.at(), cat))
	look("one store's backups are alive", true, true, at, "c0 ")
	f.must(f.r.BackupExpired("obj", deleted, cat))
	look("the other store's backup is not past the deletion", true, true, at, "c0 ")
	f.advance(time.Minute)
	f.must(f.r.BackupExpired("obj", deleted.Add(time.Nanosecond), cat))
	look("the source is gone", false, true, at, "c0 c1 ")
	last := f.plan(ClassClaim, 1, "c1")
	c, err := f.r.Claim(last, 0, worker)
	if err != nil || len(c.Deleting) != 1 {
		t.Fatalf("the record can be deleted now %+v %v", c, err)
	}
	if ok, err := f.r.Proceed(last, 0, c.Token, "c1", worker); !ok || err != nil {
		t.Fatalf("proceed %v %v", ok, err)
	}
	f.r.guards["c1"] = append(f.r.guards["c1"], "c0")
	if ok, err := f.r.Proceed(last, 0, c.Token, "c1", worker); ok || err != nil {
		t.Fatalf("proceed while something guarded is not gone %v %v", ok, err)
	}
	f.r.guards["c1"] = f.r.guards["c1"][:1]
	f.must(f.r.Report(last, 0, c.Token, "db", []string{"c1"}, nil, worker))
	if f.access("c1") != AccessDeleted {
		t.Fatal("the record was not deleted")
	}
	reason(t, "the same identifier again", f.r.Register(Artifact{ID: "c1", Class: ClassClaim, Tenants: one, Anchor: f.at(), Stores: []string{"db"}}, ingest), ReasonDeleted)
	reason(t, "the source's identifier again", f.r.Register(Artifact{ID: "s1", Class: ClassSource, Tenants: one, Anchor: f.at(), Stores: []string{"db", "obj"}}, ingest), ReasonDeleted)
}

func TestGuardPeriodRunsFromTheLastDeletion(t *testing.T) {
	f := setup(t)
	f.policy(ClassSource, 1, 0)
	f.policy(ClassClaim, 30, 0)
	f.put("s1", ClassSource, 10, one)
	f.put("s2", ClassSource, 10, one)
	f.put("c1", ClassClaim, 1000, one)
	f.put("young", ClassClaim, 0, one)
	f.put("s3", ClassSource, 10, one)
	f.must(f.guard("c1", "s1"))
	f.must(f.guard("c1", "s2"))
	f.must(f.guard("young", "s3"))
	first := f.remove("s1", "db")
	if st, _ := f.r.Standing("c1"); st.Due || !st.DueAt.IsZero() || !st.Guarding {
		t.Fatalf("one of two sources deleted %+v", st)
	}
	f.advance(3 * day)
	f.remove("s3", "db")
	if st, _ := f.r.Standing("young"); !st.DueAt.Equal(f.at().Add(30 * day)) {
		t.Fatalf("a record younger than the deletion counts from the deletion %+v", st)
	}
	f.advance(4 * day)
	second := f.remove("s2", "db")
	if !second.After(first) {
		t.Fatal("clock")
	}
	if st, _ := f.r.Standing("c1"); st.Due || !st.DueAt.Equal(second.Add(30*day)) {
		t.Fatalf("the period counts from the later deletion %+v", st)
	}
	f.advance(time.Second)
	f.must(f.r.BackupExpired("db", f.at(), cat))
	if pv, _ := f.r.Preview(ClassClaim, "", 10); len(pv.Eligible) != 0 || pv.Guarding != 0 {
		t.Fatalf("gone but inside the period %+v", pv)
	}
	iv, _ := f.r.Impact(Policy{Class: ClassClaim, Period: 5})
	if iv.Impact.Count != 0 {
		t.Fatalf("five days have not passed since the deletion %+v", iv)
	}
	f.advance(5 * day)
	iv, _ = f.r.Impact(Policy{Class: ClassClaim, Period: 5})
	if iv.Impact.Count != 2 || iv.Guarding != 0 || iv.Held != 0 {
		t.Fatalf("five days after the deletion a shorter period reaches both records %+v", iv)
	}
	f.put("s4", ClassSource, 10, one)
	f.put("c4", ClassClaim, 1000, one)
	f.must(f.guard("c4", "s4"))
	f.remove("s4", "db")
	f.advance(6 * day)
	iv, _ = f.r.Impact(Policy{Class: ClassClaim, Period: 5})
	if iv.Impact.Count != 3 || iv.Guarding != 1 {
		t.Fatalf("a record whose source's backups are alive is counted as guarding %+v", iv)
	}
}

func TestGuardReleasesAPendingRecordAtTheLease(t *testing.T) {
	f := setup(t)
	f.policy(ClassSource, 1, 0)
	f.policy(ClassClaim, 1, 0)
	f.put("c1", ClassClaim, 10, one)
	f.put("c2", ClassClaim, 10, one)
	f.put("s1", ClassSource, 10, one)
	run := f.plan(ClassClaim, 2, "c1", "c2")
	if v := f.run(run); v.Counts[OutcomePending] != 2 {
		t.Fatalf("both pending %+v", v)
	}
	f.must(f.guard("c1", "s1"))
	c, err := f.r.Claim(run, 0, worker)
	if err != nil || len(c.Deleting) != 1 || c.Deleting[0] != "c2" {
		t.Fatalf("the lease %+v %v", c, err)
	}
	if v := f.run(run); v.Outcomes["c1"] != OutcomeGuarding || f.access("c1") != AccessOK {
		t.Fatalf("released as guarding %+v", v)
	}
	reason(t, "proceed for a released record", second(f.r.Proceed(run, 0, c.Token, "c1", worker)), ReasonNotInChunk)

	f.put("c3", ClassClaim, 10, one)
	f.put("s3", ClassSource, 10, one)
	again := f.plan(ClassClaim, 1, "c3")
	lease, err := f.r.Claim(again, 0, worker)
	f.must(err)
	f.must(f.r.Report(again, 0, lease.Token, "db", nil, []string{"c3"}, worker))
	f.advance(2 * time.Hour)
	f.r.guards["c3"] = []string{"s3"}
	retry, err := f.r.Retry(again, 0, worker)
	if err != nil || len(retry.Deleting) != 0 {
		t.Fatalf("a retry of a record that guards an existing source %+v %v", retry, err)
	}
	if v := f.run(again); v.Outcomes["c3"] != OutcomeGuarding {
		t.Fatalf("released at the retry %+v", v)
	}
}

type guardRun struct {
	id       string
	manifest []string
	size     int
	next     int
	token    uint64
	leased   bool
}

func TestGuardProperty(t *testing.T) {
	stores := []string{"db", "obj"}
	guarded, became := 0, 0
	for seed := int64(1); seed <= 900; seed++ {
		rng := rand.New(rand.NewSource(seed))
		f := setup(t)
		period := 1 + rng.Intn(6)
		f.policy(ClassSource, 1+rng.Intn(4), 0)
		f.policy(ClassClaim, period, 0)
		sources, records := []string{}, []string{}
		anchor := map[string]time.Time{}
		for i := 0; i < 6; i++ {
			id := "s" + itoa(i)
			f.put(id, ClassSource, rng.Intn(12), one, stores[:1+rng.Intn(2)]...)
			sources = append(sources, id)
		}
		for i := 0; i < 4; i++ {
			id := "c" + itoa(i)
			age := rng.Intn(12)
			f.put(id, ClassClaim, age, one)
			anchor[id] = f.at().Add(-time.Duration(age) * day)
			records = append(records, id)
		}
		guards := map[string][]string{}
		deletedAt := map[string]map[string]time.Time{}
		backups := map[string]time.Time{}
		gone := func(id string) bool {
			for _, s := range f.r.artifacts[id].view.Stores {
				when, done := deletedAt[id][s]
				if !done || !when.Before(backups[s]) {
					return false
				}
			}
			return true
		}
		authorized := func(what, id string) {
			t.Helper()
			if !contains(records, id) {
				return
			}
			from := anchor[id]
			for _, g := range guards[id] {
				if !gone(g) {
					t.Fatalf("seed %d: %s of %s while %s is not gone", seed, what, id, g)
				}
				for _, when := range deletedAt[g] {
					if when.After(from) {
						from = when
					}
				}
			}
			if f.at().Before(from.Add(time.Duration(period) * day)) {
				t.Fatalf("seed %d: %s of %s before its period ended", seed, what, id)
			}
			if len(guards[id]) > 0 {
				guarded++
			}
		}
		runs := []*guardRun{}
		holds, wrote, allowed := 0, 0, 0
		for step := 0; step < 220; step++ {
			switch op := rng.Intn(12); {
			case op == 0:
				f.advance(time.Duration(1+rng.Intn(120)) * time.Hour)
			case op == 1:
				s := stores[rng.Intn(2)]
				if err := f.r.BackupExpired(s, f.at(), cat); err == nil {
					backups[s] = f.at()
				}
			case op == 2:
				from, to := records[rng.Intn(len(records))], sources[rng.Intn(len(sources))]
				if contains(guards[from], to) {
					continue
				}
				if err := f.guard(from, to); err == nil {
					guards[from] = append(guards[from], to)
					wrote++
				}
			case op == 3 && holds < 3:
				holds++
				f.must(second(f.r.PlaceHold(Hold{ID: "h" + itoa(holds), Owner: "o", Reason: "r", Artifacts: []string{records[rng.Intn(len(records))]}}, cat)))
			case op <= 5:
				class, pool := ClassSource, sources
				if rng.Intn(2) == 0 {
					class, pool = ClassClaim, records
				}
				manifest := []string{}
				for _, id := range pool {
					if a := f.r.artifacts[id]; !a.deleted && a.run == "" && rng.Intn(3) > 0 {
						manifest = append(manifest, id)
					}
				}
				if len(manifest) == 0 || len(runs) > 20 {
					continue
				}
				sort.Strings(manifest)
				size := 1 + rng.Intn(3)
				runs = append(runs, &guardRun{id: f.plan(class, size, manifest...), manifest: manifest, size: size})
			case op == 6 && len(runs) > 0:
				x := runs[rng.Intn(len(runs))]
				f.r.Stop(x.id, cat)
			case len(runs) > 0:
				x := runs[rng.Intn(len(runs))]
				if x.next*x.size >= len(x.manifest) {
					continue
				}
				span := x.manifest[x.next*x.size : min((x.next+1)*x.size, len(x.manifest))]
				if !x.leased || rng.Intn(5) == 0 {
					c, err := f.r.Claim(x.id, x.next, worker)
					if err != nil {
						c, err = f.r.Retry(x.id, x.next, worker)
					}
					if err != nil {
						if rng.Intn(4) == 0 {
							f.advance(2 * time.Hour)
						}
						if v := f.run(x.id); v.Next > x.next {
							x.next, x.leased = v.Next, false
						}
						continue
					}
					x.token, x.leased = c.Token, true
					for _, id := range c.Deleting {
						authorized("a lease", id)
						allowed++
					}
					continue
				}
				store := stores[rng.Intn(2)]
				var done, failed []string
				for _, id := range span {
					a := f.r.artifacts[id]
					if f.run(x.id).Outcomes[id] != OutcomeDeleting || !contains(a.view.Stores, store) || a.stores[store] == StoreDeleted {
						continue
					}
					if ok, err := f.r.Proceed(x.id, x.next, x.token, id, worker); err != nil || !ok {
						continue
					}
					authorized("a go-ahead", id)
					if rng.Intn(5) == 0 {
						failed = append(failed, id)
					} else {
						done = append(done, id)
					}
				}
				if len(done)+len(failed) == 0 {
					if v := f.run(x.id); v.Next > x.next {
						x.next, x.leased = v.Next, false
					}
					continue
				}
				if err := f.r.Report(x.id, x.next, x.token, store, done, failed, worker); err != nil {
					continue
				}
				for _, id := range done {
					if deletedAt[id] == nil {
						deletedAt[id] = map[string]time.Time{}
					}
					deletedAt[id][store] = f.at()
				}
			}
			for _, id := range records {
				st, err := f.r.Standing(id)
				f.must(err)
				keeping := false
				for _, g := range guards[id] {
					keeping = keeping || !gone(g)
				}
				if keeping {
					became++
				}
				if st.Guarding != keeping {
					t.Fatalf("seed %d step %d: %s guarding %v want %v", seed, step, id, st.Guarding, keeping)
				}
			}
		}
		if seed == 900 && (wrote == 0 || allowed == 0) {
			t.Fatalf("the last run wrote %d guards and saw %d go-aheads", wrote, allowed)
		}
	}
	if guarded < 100 || became < 1000 {
		t.Fatalf("too little was exercised: %d go-aheads for a record with a guard, %d observations of a guarding record", guarded, became)
	}
}

func TestGuardReasonOrder(t *testing.T) {
	f := setup(t)
	f.policy(ClassClaim, 1, 0)
	f.put("c1", ClassClaim, 10, one)
	f.put("c2", ClassClaim, 10, one)
	f.put("s1", ClassSource, 10, one)
	f.put("f1", ClassFinal, 1, one)
	for _, id := range []string{"c1", "c2"} {
		f.must(f.guard(id, "s1"))
		f.must(f.r.Refer(Reference{From: "f1", To: id, Kind: Requires}, ingest))
	}
	f.must(second(f.r.PlaceHold(Hold{ID: "h1", Owner: "o", Reason: "r", Artifacts: []string{"c2"}}, cat)))
	run := f.plan(ClassClaim, 2, "c1", "c2")
	if v := f.run(run); v.Outcomes["c1"] != OutcomeGuarding || v.Outcomes["c2"] != OutcomeHeld || v.Counts[OutcomeReferenced] != 0 {
		t.Fatalf("held before guarding before required %+v", v)
	}
}
