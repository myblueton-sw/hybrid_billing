package simulation

const maxResultBytes = 128

func (x Result) valid() bool {
	if !validID(x.Name) || len(x.Value) > maxResultBytes || x.Currency != "" && !validCurrency(x.Currency) {
		return false
	}
	return validDecimal(x.Value) || validHex(x.Value) || validStatus(x.Value)
}

func (x Run) valid() bool {
	if !validID(x.Scenario) || !semver(x.Engine) || !validHex(x.Rules) || !validHex(x.Output) || len(x.Results) > maxChecks {
		return false
	}
	for index, result := range x.Results {
		if !result.valid() || index > 0 && x.Results[index-1].Name >= result.Name {
			return false
		}
	}
	return true
}

func encodeResults(results []Result) []byte {
	var e encoder
	e.text("hybrid-billing/simulation-results/v1")
	e.count(len(results))
	for _, x := range results {
		for _, value := range []string{x.Name, x.Value, x.Currency} {
			e.text(value)
		}
	}
	return e.out.Bytes()
}

func (r *Registry) Record(x Run, actor Actor) (RunView, error) {
	now := r.enter()
	defer r.leave()
	switch {
	case !actor.valid() || !x.valid():
		return RunView{}, ErrInvalid
	case actor.Kind == KindAssistant:
		return RunView{}, refuse(ReasonAINotAllowed)
	}
	h := r.harnesses[r.identities[actor.ID]]
	if actor.Kind != KindSystem || h == nil || h.revoked {
		return RunView{}, refuse(ReasonNotHarness)
	}
	s := r.scenarios[x.Scenario]
	switch {
	case s == nil:
		return RunView{}, ErrNotFound
	case len(s.expectations) == 0:
		return RunView{}, refuse(ReasonNoExpectation)
	}
	exp := s.expectations[len(s.expectations)-1]
	for _, person := range []string{h.proposer, h.approver} {
		if person == exp.author || person == exp.approver {
			return RunView{}, refuse(ReasonSelfVerification)
		}
	}
	name := [3]string{x.Scenario, x.Engine, x.Rules}
	k := r.keys[name]
	if len(s.runs) >= maxRuns || len(r.runs) >= maxRunsTotal || k == nil && len(r.keys) >= maxKeys {
		return RunView{}, ErrLimit
	}
	number := uint64(len(r.runs) + 1)
	results := digest(encodeResults(x.Results))
	if k == nil {
		k = &key{output: x.Output, results: results}
		r.keys[name] = k
	}
	k.count++
	if (k.output != x.Output || k.results != results) && k.marked == 0 {
		k.marked = number
	}
	got := map[string]Result{}
	for _, result := range x.Results {
		got[result.Name] = result
	}
	view := RunView{Number: number, Scenario: x.Scenario, Engine: x.Engine, Rules: x.Rules, Output: x.Output, Results: results, Harness: r.identities[actor.ID],
		At: now, Expectation: exp.view.Version, Unexpected: len(x.Results), Copied: x.Output == exp.view.Evidence}
	stored := &run{view: view, key: k, harness: h, checks: exp.view.Checks, codes: make([]byte, len(exp.view.Checks))}
	for index, c := range exp.view.Checks {
		result, ok := got[c.Name]
		switch {
		case !ok:
			stored.codes[index] = codeMissing
			continue
		case result.Value == c.Value && result.Currency == c.Currency:
			stored.codes[index] = codeMatch
		default:
			stored.codes[index] = codeMismatch
			if len(stored.values) < maxValues {
				stored.values = append(stored.values, Outcome{Name: c.Name, Outcome: OutcomeMismatch, Value: result.Value, Currency: result.Currency})
			}
		}
		stored.view.Unexpected--
	}
	r.runs = append(r.runs, stored)
	s.runs = append(s.runs, number)
	r.outputs[x.Output] = true
	r.ran[s.fingerprint] = true
	return r.runView(r.runs[number-1]), nil
}

func (r *Registry) RevokeHarness(id string, actor Actor) error {
	r.enter()
	defer r.leave()
	if err := actor.human(); err != nil {
		return err
	}
	if !validID(id) {
		return ErrInvalid
	}
	h := r.harnesses[id]
	if h == nil {
		return ErrNotFound
	}
	h.revoked = true
	return nil
}

func (r *Registry) runView(x *run) RunView {
	v := x.view
	v.MarkedBy, v.Key, v.Verdict = x.key.marked, r.keyState(x.key), r.verdict(x)
	values := x.values
	for index, c := range x.checks {
		o := Outcome{Name: c.Name, Outcome: outcomeNames[x.codes[index]]}
		if x.codes[index] == codeMismatch && len(values) > 0 && values[0].Name == c.Name {
			o, values = values[0], values[1:]
		}
		v.Outcomes = append(v.Outcomes, o)
	}
	return v
}

func (r *Registry) verdict(x *run) string {
	clean := !x.view.Copied && x.view.Unexpected == 0
	for _, code := range x.codes {
		clean = clean && code == codeMatch
	}
	switch key := r.keyState(x.key); {
	case x.harness.revoked:
		return VerdictRevoked
	case key == KeyNonReproducible:
		return VerdictNonReproducible
	case !clean:
		return VerdictFail
	case key == KeyReproduced:
		return VerdictPass
	}
	return VerdictUnconfirmed
}

const (
	codeMatch byte = iota
	codeMismatch
	codeMissing
)

var outcomeNames = []string{OutcomeMatch, OutcomeMismatch, OutcomeMissing}

var rank = map[string]int{VerdictRevoked: 0, VerdictNonReproducible: 1, VerdictFail: 2, VerdictUnconfirmed: 3, VerdictPass: 4}

func (*Registry) keyState(k *key) string {
	switch {
	case k.marked != 0:
		return KeyNonReproducible
	case k.count >= 2:
		return KeyReproduced
	}
	return KeySingle
}

func (r *Registry) current(s *scenario) (RunView, bool) {
	if len(s.runs) == 0 {
		return RunView{Scenario: s.view.ID, Verdict: VerdictNone}, false
	}
	top := r.runs[s.runs[0]-1].view.Engine
	for _, number := range s.runs {
		if engine := r.runs[number-1].view.Engine; semverLess(top, engine) {
			top = engine
		}
	}
	latest := map[string]uint64{}
	for _, number := range s.runs {
		if x := r.runs[number-1].view; x.Engine == top {
			latest[x.Rules] = number
		}
	}
	var worst *run
	verdict := ""
	for _, number := range s.runs {
		x := r.runs[number-1]
		if latest[x.view.Rules] != number {
			continue
		}
		if v := r.verdict(x); worst == nil || rank[v] <= rank[verdict] {
			worst, verdict = x, v
		}
	}
	return r.runView(worst), true
}

type currents map[string]RunView

func (r *Registry) currentOf(cache currents, id string) RunView {
	if v, ok := cache[id]; ok {
		return v
	}
	v, _ := r.current(r.scenarios[id])
	cache[id] = v
	return v
}

func (r *Registry) cellStatus(x *cell, cache currents) string {
	switch {
	case !x.set:
		return CellOpen
	case x.view.Coverage == CoverageUnsupported:
		return CellUnsupported
	case x.view.Coverage == CoverageUndecided:
		return CellUndecided
	}
	s := r.scenarios[x.view.Scenario]
	v := r.currentOf(cache, x.view.Scenario)
	switch v.Verdict {
	case VerdictFail, VerdictNonReproducible, VerdictRevoked:
		return CellFailed
	case VerdictPass:
	default:
		return CellPending
	}
	names := map[string]bool{}
	for _, c := range s.expectations[v.Expectation-1].view.Checks {
		names[c.Name] = true
	}
	for _, name := range x.view.Checks {
		if !names[name] {
			return CellPending
		}
	}
	return CellPassed
}
