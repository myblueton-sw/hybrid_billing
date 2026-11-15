package simulation

import (
	"math/rand"
	"strconv"
	"testing"
)

type modelRun struct {
	scenario string
	engine   int
	rules    string
	output   string
	results  string
	harness  string
	clean    bool
}

type modelCell struct {
	set      bool
	coverage string
	scenario string
	failures int
}

type model struct {
	runs     []modelRun
	revoked  map[string]bool
	checks   map[string][]Check
	ran      map[string]bool
	cell     modelCell
	versions map[string]int
}

func (m *model) verdict(index int) string {
	x := m.runs[index]
	count, marked := 0, false
	var first *modelRun
	for i := range m.runs {
		y := m.runs[i]
		if y.scenario != x.scenario || y.engine != x.engine || y.rules != x.rules {
			continue
		}
		count++
		if first == nil {
			first = &m.runs[i]
		} else if y.output != first.output || y.results != first.results {
			marked = true
		}
	}
	switch {
	case m.revoked[x.harness]:
		return VerdictRevoked
	case marked:
		return VerdictNonReproducible
	case !x.clean:
		return VerdictFail
	case count >= 2:
		return VerdictPass
	}
	return VerdictUnconfirmed
}

func (m *model) marked(index int) int {
	x := m.runs[index]
	first := -1
	for i, y := range m.runs {
		if y.scenario != x.scenario || y.engine != x.engine || y.rules != x.rules {
			continue
		}
		if first < 0 {
			first = i
		} else if y.output != m.runs[first].output || y.results != m.runs[first].results {
			return i + 1
		}
	}
	return 0
}

func (m *model) deciding(scenario string) int {
	top := -1
	for _, x := range m.runs {
		if x.scenario == scenario && x.engine > top {
			top = x.engine
		}
	}
	order := map[string]int{VerdictRevoked: 0, VerdictNonReproducible: 1, VerdictFail: 2, VerdictUnconfirmed: 3, VerdictPass: 4}
	latest := map[string]int{}
	for i, x := range m.runs {
		if x.scenario == scenario && x.engine == top {
			latest[x.rules] = i
		}
	}
	best := -1
	for i := range m.runs {
		x := m.runs[i]
		if x.scenario != scenario || x.engine != top || latest[x.rules] != i {
			continue
		}
		if best < 0 || order[m.verdict(i)] <= order[m.verdict(best)] {
			best = i
		}
	}
	return best + 1
}

func (m *model) current(scenario string) string {
	top := -1
	for _, x := range m.runs {
		if x.scenario == scenario && x.engine > top {
			top = x.engine
		}
	}
	if top < 0 {
		return VerdictNone
	}
	order := map[string]int{VerdictRevoked: 0, VerdictNonReproducible: 1, VerdictFail: 2, VerdictUnconfirmed: 3, VerdictPass: 4}
	latest := map[string]int{}
	for i, x := range m.runs {
		if x.scenario == scenario && x.engine == top {
			latest[x.rules] = i
		}
	}
	worst := ""
	for _, i := range latest {
		if v := m.verdict(i); worst == "" || order[v] < order[worst] {
			worst = v
		}
	}
	return worst
}

func (m *model) status() string {
	c := m.cell
	switch {
	case !c.set:
		return CellOpen
	case c.coverage != CoverageTested:
		return c.coverage
	}
	switch m.current(c.scenario) {
	case VerdictPass:
		return CellPassed
	case VerdictFail, VerdictNonReproducible, VerdictRevoked:
		return CellFailed
	}
	return CellPending
}

func TestSimulationProperty(t *testing.T) {
	runs := 150
	if testing.Short() {
		runs = 30
	}
	seen := map[string]int{}
	for seed := 0; seed < runs; seed++ {
		runSimulationProperty(t, int64(seed), seen)
	}
	for _, name := range []string{VerdictPass, VerdictFail, VerdictUnconfirmed, VerdictNonReproducible, VerdictRevoked, "cell-" + CellPassed, "cell-" + CellFailed, "cell-" + CellPending, "cell-" + CellUndecided, "cell-" + CellUnsupported} {
		if seen[name] == 0 {
			t.Errorf("%s never reached", name)
		}
	}
	t.Logf("reached %v", seen)
}

func runSimulationProperty(t *testing.T, seed int64, seen map[string]int) {
	rng := rand.New(rand.NewSource(seed))
	f := standard(t)
	s2 := billing("s2")
	s2.Events = sum("events-s2-distinct")
	f.approve(Change{Kind: ChangeScenario, Scenario: s2})
	f.approveBy(Change{Kind: ChangeHarness, Harness: Harness{ID: "h2", Identity: other.ID}}, cat, dan)
	m := &model{revoked: map[string]bool{}, checks: map[string][]Check{}, ran: map[string]bool{}, versions: map[string]int{}}
	scenarios := []string{"s1", "s2"}
	engines := []string{"1.0.0", "9.0.0", "10.0.0"}
	for step := 0; step < 40; step++ {
		sc := scenarios[rng.Intn(2)]
		switch rng.Intn(6) {
		case 0:
			checks := []Check{amount("krw-total", strconv.Itoa(rng.Intn(3)), "KRW"), {Name: "lines", Kind: CheckCount, Value: strconv.Itoa(rng.Intn(3))}}
			err := f.try(Change{Kind: ChangeExpectation, Expectation: expectation(sc, m.versions[sc]+1, checks...)})
			if m.ran[sc] {
				reason(t, "after runs", err, ReasonRunsRecorded)
				continue
			}
			f.must(err)
			m.versions[sc]++
			m.checks[sc] = checks
		case 1, 2, 3:
			engine := rng.Intn(3)
			if rng.Intn(2) == 0 {
				engine = 2
			}
			rules := "r" + strconv.Itoa(rng.Intn(2))
			output := "o" + rules
			switch rng.Intn(12) {
			case 0, 1:
				output = "changed"
			case 2:
				output = "sheet-" + sc + strconv.Itoa(m.versions[sc])
			}
			list := results(m.checks[sc])
			clean := true
			switch rng.Intn(10) {
			case 0:
				if len(list) > 0 {
					list[rng.Intn(len(list))].Value = "9"
					clean = false
				}
			case 1:
				if len(list) > 0 {
					list = list[1:]
					clean = false
				}
			case 2:
				list = append(list, Result{Name: "zz", Value: "1"})
				clean = false
			}
			actor, harness := runner, "h1"
			if rng.Intn(3) == 0 {
				actor, harness = other, "h2"
			}
			_, err := f.r.Record(Run{Scenario: sc, Engine: engines[engine], Rules: sum(rules), Output: sum(output), Results: list}, actor)
			switch {
			case m.revoked[harness]:
				reason(t, "revoked", err, ReasonNotHarness)
				continue
			case m.versions[sc] == 0:
				reason(t, "no expectation", err, ReasonNoExpectation)
				continue
			}
			f.must(err)
			m.ran[sc] = true
			copied := output == "sheet-"+sc+strconv.Itoa(m.versions[sc])
			m.runs = append(m.runs, modelRun{scenario: sc, engine: engine, rules: rules, output: output, results: digest(encodeResults(list)), harness: harness, clean: clean && !copied})
		case 4:
			old := m.status()
			var c Cell
			switch rng.Intn(3) {
			case 0:
				c = Cell{Profile: "p1", Event: EventOverage, Coverage: CoverageTested, Scenario: sc, Checks: []string{"krw-total"}}
			case 1:
				c = Cell{Profile: "p1", Event: EventOverage, Coverage: CoverageUndecided, Decision: "d" + strconv.Itoa(step)}
			default:
				c = Cell{Profile: "p1", Event: EventOverage, Coverage: CoverageUnsupported, Dataset: "pub", Evidence: sum(strconv.Itoa(step))}
			}
			err := f.try(Change{Kind: ChangeCell, Cell: c})
			if c.Coverage == CoverageTested && m.versions[sc] == 0 {
				reason(t, "cell without expectation", err, ReasonNoExpectation)
				continue
			}
			f.must(err)
			if m.cell.set && old == CellFailed {
				m.cell.failures++
			}
			m.cell = modelCell{set: true, coverage: c.Coverage, scenario: c.Scenario, failures: m.cell.failures}
		case 5:
			if rng.Intn(4) == 0 {
				f.must(f.r.RevokeHarness("h2", ann))
				m.revoked["h2"] = true
			}
		}
		for index := range m.runs {
			v, err := f.r.Run(uint64(index + 1))
			f.must(err)
			if v.MarkedBy != uint64(m.marked(index)) {
				t.Fatalf("seed %d step %d run %d marked %d, model %d", seed, step, index+1, v.MarkedBy, m.marked(index))
			}
			if want := m.verdict(index); v.Verdict != want {
				t.Fatalf("seed %d step %d run %d verdict %s, model %s", seed, step, index+1, v.Verdict, want)
			}
		}
		for _, name := range scenarios {
			seen[m.current(name)]++
			if c, _ := f.r.Current(name); c.Verdict != m.current(name) || c.Run != uint64(m.deciding(name)) {
				t.Fatalf("seed %d step %d current %s %s, model %s", seed, step, name, c.Verdict, m.current(name))
			}
		}
		seen["cell-"+m.status()]++
		if v := status(t, f, EventOverage); v.Status != m.status() || v.Failures != m.cell.failures {
			t.Fatalf("seed %d step %d cell %s/%d, model %s/%d", seed, step, v.Status, v.Failures, m.status(), m.cell.failures)
		}
		summary := f.r.Summary()
		if summary[m.status()] < 1 || summary[CellOpen] < 17 {
			t.Fatalf("seed %d step %d summary %+v", seed, step, summary)
		}
	}
}
