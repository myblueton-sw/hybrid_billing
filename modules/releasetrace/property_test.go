package releasetrace

import (
	"math/rand"
	"testing"
)

type modelResult struct {
	passed           bool
	harness          int
	passes, failures int
}

type model struct {
	implSeq  int
	built    map[string]int
	revoked  map[int]bool
	results  map[[2]string]*modelResult
	seq      int
	tests    []string
	implDone bool
}

func (m *model) state(build string) string {
	if !m.implDone {
		return StateDesigned
	}
	all := true
	for _, test := range m.tests {
		res := m.results[[2]string{test, build}]
		ok := res != nil && m.built[build] > m.implSeq && !m.revoked[res.harness]
		if ok && !res.passed {
			return StateFailing
		}
		all = all && ok
	}
	if all {
		return StateTested
	}
	return StateImplemented
}

func TestPropertyStates(t *testing.T) {
	for seed := int64(1); seed <= 60; seed++ {
		rng := rand.New(rand.NewSource(seed))
		f := setup(t)
		tests := []string{"a", "b", "c"}[:1+rng.Intn(3)]
		refs := []TestRef{}
		for _, id := range tests {
			refs = append(refs, TestRef{ID: id})
		}
		f.approve(Change{Kind: ChangeRequirement, Requirement: req("R", false, refs...)})
		m := &model{built: map[string]int{}, revoked: map[int]bool{}, results: map[[2]string]*modelResult{}, tests: tests, seq: 1}
		harnesses := 0
		addHarness := func() {
			id := "h" + itoa(harnesses)
			f.by(Change{Kind: ChangeHarness, Harness: Harness{ID: id, Identity: "r" + id}}, cat, dan)
			harnesses++
		}
		addHarness()
		f.by(Change{Kind: ChangeHarness, Harness: Harness{ID: "self", Identity: "rself"}}, ann, eve)
		builds := []string{}
		for step := 0; step < 80; step++ {
			switch op := rng.Intn(10); {
			case op == 0:
				m.seq++
				f.approve(Change{Kind: ChangeImplemented, Implemented: Implemented{Key: k("R"), Commit: sum(itoa(int(seed)) + itoa(step))[:40]}})
				m.implSeq, m.implDone = m.seq, true
			case op == 1 || len(builds) == 0:
				m.seq++
				b := "1." + itoa(len(builds)) + ".0"
				f.approve(Change{Kind: ChangeBuild, Build: Build{Build: b, Commit: sum(b)[:40]}})
				m.built[b] = m.seq
				builds = append(builds, b)
			case op == 2 && harnesses < 4:
				addHarness()
			case op == 3:
				h := rng.Intn(harnesses)
				f.must(f.r.RevokeHarness("h"+itoa(h), ann))
				m.revoked[h] = true
			case op == 4:
				_, err := f.r.Record(Result{Namespace: "v1", Test: tests[0], Build: builds[rng.Intn(len(builds))], Passed: true, Evidence: sum("s")}, Actor{ID: "rself", Kind: KindSystem})
				reason(t, "self harness", err, ReasonSelfVerification)
			default:
				h := rng.Intn(harnesses)
				test, b, passed := tests[rng.Intn(len(tests))], builds[rng.Intn(len(builds))], rng.Intn(3) > 0
				_, err := f.r.Record(Result{Namespace: "v1", Test: test, Build: b, Passed: passed, Evidence: sum("e")}, Actor{ID: "rh" + itoa(h), Kind: KindSystem})
				if m.revoked[h] {
					reason(t, "revoked", err, ReasonNotHarness)
					continue
				}
				f.must(err)
				key := [2]string{test, b}
				res := m.results[key]
				if res == nil {
					res = &modelResult{}
					m.results[key] = res
				}
				res.passed, res.harness = passed, h
				if passed {
					res.passes++
				} else {
					res.failures++
				}
			}
			for _, b := range builds {
				v, err := f.r.Requirement(k("R"), b)
				f.must(err)
				if want := m.state(b); v.State != want {
					t.Fatalf("seed %d step %d build %s: got %s want %s", seed, step, b, v.State, want)
				}
				for index, test := range tests {
					res := m.results[[2]string{test, b}]
					if res == nil {
						continue
					}
					if tv := v.Tests[index]; tv.Passes != res.passes || tv.Failures != res.failures {
						t.Fatalf("seed %d counts %s %s %+v %+v", seed, test, b, tv, res)
					}
				}
			}
		}
	}
}

func TestPropertyGate(t *testing.T) {
	for seed := int64(1); seed <= 200; seed++ {
		rng := rand.New(rand.NewSource(seed))
		f := setup(t)
		f.by(Change{Kind: ChangeHarness, Harness: Harness{ID: "h", Identity: runner.ID}}, cat, dan)
		type spec struct {
			common, blocking, implemented, retired bool
			outcome                                int
		}
		n := 2 + rng.Intn(6)
		specs := map[string]spec{}
		ids := []string{}
		for i := 0; i < n; i++ {
			id := "Q" + itoa(i)
			s := spec{common: rng.Intn(3) == 0, blocking: rng.Intn(3) == 0, implemented: rng.Intn(4) > 0, outcome: rng.Intn(3)}
			q := req(id, s.common, TestRef{ID: "t" + id})
			q.Blocking = s.blocking
			f.approve(Change{Kind: ChangeRequirement, Requirement: q})
			if s.implemented {
				f.approve(Change{Kind: ChangeImplemented, Implemented: Implemented{Key: k(id), Commit: sum(id)[:40]}})
			}
			specs[id], ids = s, append(ids, id)
		}
		f.approve(Change{Kind: ChangeBuild, Build: Build{Build: "1.0.0", Commit: sum("b")[:40]}})
		for _, id := range ids {
			if s := specs[id]; s.outcome > 0 {
				f.pass("v1", "t"+id, "1.0.0", s.outcome == 1)
			}
		}
		onPath := []Key{}
		for _, id := range ids {
			if rng.Intn(2) == 0 || len(onPath) == 0 && id == ids[len(ids)-1] {
				onPath = append(onPath, k(id))
			}
		}
		f.approve(Change{Kind: ChangePath, Path: Path{ID: "P", Provider: "a", Contract: "b", Role: "c", Version: "d", Issuance: IssuanceFile, Requirements: onPath}})
		for _, id := range ids {
			if s := specs[id]; !s.common && rng.Intn(4) == 0 {
				f.approve(Change{Kind: ChangeRetire, Retire: Retire{Key: k(id), Reason: "x"}})
				s.retired = true
				specs[id] = s
			}
		}
		selected := []Key{}
		chosen := map[string]bool{}
		for _, id := range append(ids, "Q8", "Q9") {
			if rng.Intn(2) == 0 {
				selected = append(selected, k(id))
				chosen[id] = true
			}
		}
		if len(selected) == 0 {
			selected, chosen["Q9"] = []Key{k("Q9")}, true
		}
		paths := []string{"P"}
		if rng.Intn(2) == 0 {
			paths = append(paths, "PZ")
		}
		want := []Problem{}
		add := func(p, s string) { want = append(want, Problem{p, s}) }
		tested := func(s spec) bool { return s.implemented && s.outcome == 1 }
		hasCommon := false
		retiredBlocking := []string{}
		for _, key := range selected {
			s, ok := specs[key.ID]
			switch {
			case !ok:
				add(ProblemUnknown, "v1/"+key.ID)
				continue
			case s.retired:
				add(ProblemRetired, "v1/"+key.ID)
			}
			if !tested(s) {
				add(ProblemNotTested, "v1/"+key.ID)
			}
		}
		for _, id := range ids {
			s := specs[id]
			if s.retired {
				retiredBlocking = append(retiredBlocking, id)
				continue
			}
			if s.common {
				hasCommon = true
				if !chosen[id] {
					add(ProblemCommonMissing, "v1/"+id)
				}
			}
			if s.blocking && !tested(s) {
				add(ProblemBlocking, "v1/"+id)
			}
		}
		if !hasCommon {
			add(ProblemCommonUnregistered, "")
		}
		add(ProblemPathInactive, "P")
		for _, key := range onPath {
			if !chosen[key.ID] {
				add(ProblemPathRequirement, "P>v1/"+key.ID)
			}
		}
		add(ProblemPathEvidence, "P")
		if len(paths) == 2 {
			add(ProblemUnknown, "PZ")
		}
		for i := 1; i < len(want); i++ {
			for j := i; j > 0 && (want[j].Problem < want[j-1].Problem || want[j].Problem == want[j-1].Problem && want[j].Subject < want[j-1].Subject); j-- {
				want[j], want[j-1] = want[j-1], want[j]
			}
		}
		g, err := f.r.Gate(Release{Version: "r", Build: "1.0.0", Requirements: selected, Paths: paths})
		f.must(err)
		if g.Ready || len(g.Problems) != len(want) || len(g.RetiredBlocking) != len(retiredBlocking) || len(g.Templates) != 0 {
			t.Fatalf("seed %d got %+v want %+v", seed, g, want)
		}
		for i := range want {
			if g.Problems[i] != want[i] {
				t.Fatalf("seed %d problem %d got %+v want %+v", seed, i, g.Problems[i], want[i])
			}
		}
		for i, id := range retiredBlocking {
			if g.RetiredBlocking[i] != k(id) {
				t.Fatalf("seed %d retired %+v", seed, g.RetiredBlocking)
			}
		}
		for _, id := range ids {
			s := specs[id]
			state := StateDesigned
			switch {
			case s.implemented && s.outcome == 1:
				state = StateTested
			case s.implemented && s.outcome == 2:
				state = StateFailing
			case s.implemented:
				state = StateImplemented
			}
			if got, ok := g.States[k(id)]; ok != chosen[id] || ok && got != state {
				t.Fatalf("seed %d states %+v", seed, g.States)
			}
		}
	}
}
