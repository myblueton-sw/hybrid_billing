package simulation

import (
	"strconv"
	"testing"
)

func TestRegisterCaps(t *testing.T) {
	cases := []struct {
		name   string
		fill   func(f world, n int)
		cap    int
		change func(f world) Change
	}{
		{"datasets", func(f world, n int) {
			for len(f.r.datasets) < n {
				f.r.datasets["x"+strconv.Itoa(len(f.r.datasets))] = &Dataset{}
			}
		}, maxDatasets, func(world) Change { return Change{Kind: ChangeDataset, Dataset: synthetic("new")} }},
		{"scenarios", func(f world, n int) {
			for len(f.r.scenarios) < n {
				f.r.scenarios["x"+strconv.Itoa(len(f.r.scenarios))] = &scenario{}
			}
		}, maxScenarios, func(world) Change { return Change{Kind: ChangeScenario, Scenario: billing("new")} }},
		{"profiles", func(f world, n int) {
			for len(f.r.profiles) < n {
				f.r.profiles["x"+strconv.Itoa(len(f.r.profiles))] = nil
			}
		}, maxProfiles, func(world) Change { return Change{Kind: ChangeProfile, Profile: Profile{ID: "new"}} }},
		{"targets", func(f world, n int) {
			for len(f.r.targets) < n {
				f.r.targets["x"+strconv.Itoa(len(f.r.targets))] = &target{}
			}
		}, maxTargets, func(world) Change {
			return Change{Kind: ChangeTarget, Target: Target{ID: "new", Kind: TargetCSP, Scenarios: []string{"s1"}}}
		}},
		{"harnesses", func(f world, n int) {
			for len(f.r.harnesses) < n {
				f.r.harnesses["x"+strconv.Itoa(len(f.r.harnesses))] = &harness{}
			}
		}, maxHarnesses, func(world) Change {
			return Change{Kind: ChangeHarness, Harness: Harness{ID: "new", Identity: "new-runner"}}
		}},
	}
	for _, c := range cases {
		f := standard(t)
		c.fill(f, c.cap-1)
		if err := f.try(c.change(f)); err != nil {
			t.Errorf("%s exact fit: %v", c.name, err)
		}
		g := standard(t)
		c.fill(g, c.cap)
		expect(t, c.name, g.try(c.change(g)), ErrLimit)
	}
}

func TestPendingCaps(t *testing.T) {
	f := setup(t)
	for index := 0; index < maxMine; index++ {
		f.must(second(f.r.Propose(Change{Kind: ChangeProfile, Profile: Profile{ID: "a" + strconv.Itoa(index)}}, ann)))
	}
	expect(t, "mine", second(f.r.Propose(Change{Kind: ChangeProfile, Profile: Profile{ID: "a-over"}}, ann)), ErrLimit)
	for _, who := range []Actor{bob, cat} {
		for index := 0; index < maxMine; index++ {
			f.must(second(f.r.Propose(Change{Kind: ChangeProfile, Profile: Profile{ID: who.ID + strconv.Itoa(index)}}, who)))
		}
	}
	eve := Actor{ID: "eve", Kind: KindHuman}
	for index := 0; index < maxPending-3*maxMine; index++ {
		who := dan
		if index%2 == 1 {
			who = eve
		}
		f.must(second(f.r.Propose(Change{Kind: ChangeProfile, Profile: Profile{ID: "d" + strconv.Itoa(index)}}, who)))
	}
	expect(t, "total", second(f.r.Propose(Change{Kind: ChangeProfile, Profile: Profile{ID: "d-over"}}, Actor{ID: "fay", Kind: KindHuman})), ErrLimit)
	if len(f.r.pending) != maxPending {
		t.Errorf("pending %d", len(f.r.pending))
	}
}

func TestDoneHistory(t *testing.T) {
	f := setup(t)
	first := f.approve(Change{Kind: ChangeProfile, Profile: Profile{ID: "p0"}})
	for index := 2; index < maxDone; index++ {
		f.r.done = append(f.r.done, uint64(100000+index))
	}
	f.approve(Change{Kind: ChangeProfile, Profile: Profile{ID: "p-fit"}})
	if _, err := f.r.Proposal(first.Number); err != nil || len(f.r.done) != maxDone {
		t.Fatal("pruned early")
	}
	last := f.approve(Change{Kind: ChangeProfile, Profile: Profile{ID: "p1"}})
	expect(t, "pruned", second(f.r.Proposal(first.Number)), ErrNotFound)
	if _, err := f.r.Proposal(last.Number); err != nil || len(f.r.done) != maxDone {
		t.Errorf("kept %v %d", err, len(f.r.done))
	}
}

func TestRunCaps(t *testing.T) {
	x := Run{Scenario: "s1", Engine: "1.0.0", Rules: sum("r"), Output: sum("o")}
	f := standard(t)
	f.expect("s1", standardChecks()...)
	f.r.scenarios["s1"].runs = make([]uint64, maxRuns-1)
	f.must(second(f.r.Record(x, runner)))
	expect(t, "per scenario", second(f.r.Record(x, runner)), ErrLimit)
	g := standard(t)
	g.expect("s1", standardChecks()...)
	g.r.runs = make([]*run, maxRunsTotal-1)
	g.must(second(g.r.Record(x, runner)))
	expect(t, "total", second(g.r.Record(x, runner)), ErrLimit)
	h := standard(t)
	h.expect("s1", standardChecks()...)
	for len(h.r.keys) < maxKeys-1 {
		h.r.keys[[3]string{"x", strconv.Itoa(len(h.r.keys)), ""}] = &key{}
	}
	h.must(second(h.r.Record(x, runner)))
	h.must(second(h.r.Record(x, runner)))
	other := x
	other.Rules = sum("other")
	expect(t, "keys", second(h.r.Record(other, runner)), ErrLimit)
}
