package simulation

import (
	"sort"
	"time"
)

func isEvent(name string) bool { return eventIndex(name) >= 0 }

func eventIndex(name string) int {
	for index, event := range Events {
		if event == name {
			return index
		}
	}
	return -1
}

func oneOf(value string, set ...string) bool {
	for _, item := range set {
		if item == value {
			return true
		}
	}
	return false
}

func (c Check) valid() bool {
	if !validID(c.Name) || c.Document != "" && !validID(c.Document) || c.Period != "" && !validPeriod(c.Period) {
		return false
	}
	switch c.Kind {
	case CheckAmount:
		return validDecimal(c.Value) && validCurrency(c.Currency)
	case CheckCount:
		return digits(c.Value, maxCountDigits) && c.Currency == ""
	case CheckDigest:
		return validHex(c.Value) && c.Currency == ""
	case CheckStatus:
		return validStatus(c.Value) && c.Currency == ""
	}
	return false
}

func (d Dataset) valid() bool {
	if !validID(d.ID) || !semver(d.Version) || !validHex(d.Digest) || !ascending(d.Coverage, 1, maxList, validID) {
		return false
	}
	switch d.Origin {
	case OriginPublic:
		return validSource(d.Source) && validID(d.License) && validHex(d.Notice) && d.Parent == "" && d.Tool == "" && d.ToolVersion == "" && d.Seed == ""
	case OriginConverted:
		return validID(d.Parent) && validID(d.Tool) && semver(d.ToolVersion) && validID(d.License) && validHex(d.Notice) && d.Seed == "" && d.Source == ""
	case OriginSynthetic:
		return validID(d.Tool) && semver(d.ToolVersion) && validSeed(d.Seed) && d.Source == "" && d.License == "" && d.Notice == "" && d.Parent == ""
	}
	return false
}

func (s Scenario) valid() bool {
	year := s.Start.Year()
	switch {
	case !validID(s.ID) || !semver(s.FixtureVersion) || !oneOf(s.Purpose, PurposeFormat, PurposeBilling):
		return false
	case !oneOf(s.Topology, TopologySingle, TopologyGroup, TopologyMSP) || !oneOf(s.Deployment, DeploymentCloud, DeploymentOnprem, DeploymentMixed):
		return false
	case !validSeed(s.Seed) || s.Start.IsZero() || year < 2000 || year > 2199 || !validZone(s.Zone):
		return false
	case !validID(s.Relations) || !validID(s.Policy) || !validID(s.FX) || !ascending(s.Datasets, 1, maxList, validID):
		return false
	case !validHex(s.Events) || s.EventCount < 0 || s.EventCount > maxEventCount:
		return false
	case s.Profile == "":
		return len(s.Covers) == 0
	}
	return validID(s.Profile) && s.Purpose == PurposeBilling && ascending(s.Covers, 0, len(Events), isEvent)
}

func (e Expectation) valid() bool {
	if !validID(e.Scenario) || e.Version < 1 || e.Version > maxVersions || !oneOf(e.Method, MethodManual, MethodIndependent) || !validHex(e.Evidence) {
		return false
	}
	if len(e.Checks) == 0 || len(e.Checks) > maxChecks {
		return false
	}
	for index, c := range e.Checks {
		if !c.valid() || index > 0 && e.Checks[index-1].Name >= c.Name {
			return false
		}
	}
	return true
}

func (c Cell) valid() bool {
	if !validID(c.Profile) || !isEvent(c.Event) {
		return false
	}
	linked := validID(c.Scenario) && ascending(c.Checks, 1, maxLinks, validID)
	switch c.Coverage {
	case CoverageTested:
		return linked && c.Dataset == "" && c.Evidence == "" && c.Decision == ""
	case CoverageUnsupported:
		return validID(c.Dataset) && validHex(c.Evidence) && c.Scenario == "" && len(c.Checks) == 0 && c.Decision == ""
	case CoverageUndecided:
		return validID(c.Decision) && c.Dataset == "" && c.Evidence == "" && (linked || c.Scenario == "" && len(c.Checks) == 0)
	}
	return false
}

func clean(c Change) (Change, bool) {
	out := Change{Kind: c.Kind}
	switch c.Kind {
	case ChangeDataset:
		out.Dataset = c.Dataset
		return copyChange(out), c.Dataset.valid()
	case ChangeScenario:
		out.Scenario = c.Scenario
		out.Scenario.Start = c.Scenario.Start.UTC().Round(0)
		return copyChange(out), out.Scenario.valid()
	case ChangeExpectation:
		out.Expectation = c.Expectation
		return copyChange(out), c.Expectation.valid()
	case ChangeProfile:
		out.Profile = c.Profile
		return out, validID(c.Profile.ID)
	case ChangeCell:
		out.Cell = c.Cell
		return copyChange(out), c.Cell.valid()
	case ChangeTarget:
		out.Target = c.Target
		t := c.Target
		return copyChange(out), validID(t.ID) && oneOf(t.Kind, TargetCSP, TargetERP, TargetProduct) && ascending(t.Scenarios, 1, maxList, validID)
	case ChangeProof:
		out.Proof = c.Proof
		p := c.Proof
		return out, validID(p.Target) && validID(p.Environment) && validHex(p.Evidence)
	case ChangeHarness:
		out.Harness = c.Harness
		return out, validID(c.Harness.ID) && validID(c.Harness.Identity)
	}
	return out, false
}

func copyChange(c Change) Change {
	c.Dataset.Coverage = append([]string(nil), c.Dataset.Coverage...)
	c.Scenario.Datasets = append([]string(nil), c.Scenario.Datasets...)
	c.Scenario.Covers = append([]string(nil), c.Scenario.Covers...)
	c.Expectation.Checks = append([]Check(nil), c.Expectation.Checks...)
	c.Cell.Checks = append([]string(nil), c.Cell.Checks...)
	c.Target.Scenarios = append([]string(nil), c.Target.Scenarios...)
	return c
}

func copyProposal(p ProposalView) ProposalView {
	p.Change = copyChange(p.Change)
	return p
}

func encodeChange(c Change) []byte {
	var e encoder
	d, s, x, l, t := c.Dataset, c.Scenario, c.Expectation, c.Cell, c.Target
	for _, value := range []string{c.Kind, d.ID, d.Version, d.Origin, d.Digest, d.Source, d.License, d.Notice, d.Parent, d.Tool, d.ToolVersion, d.Seed,
		s.ID, s.FixtureVersion, s.Purpose, s.Topology, s.Deployment, s.Seed, s.Zone, s.Relations, s.Policy, s.FX, s.Events, s.Profile,
		x.Scenario, x.Method, x.Evidence, c.Profile.ID, l.Profile, l.Event, l.Coverage, l.Scenario, l.Dataset, l.Evidence, l.Decision,
		t.ID, t.Kind, c.Proof.Target, c.Proof.Environment, c.Proof.Evidence, c.Harness.ID, c.Harness.Identity} {
		e.text(value)
	}
	e.number(s.Start.UnixNano())
	e.number(s.EventCount)
	e.count(x.Version)
	for _, list := range [][]string{d.Coverage, s.Datasets, s.Covers, l.Checks, t.Scenarios} {
		e.count(len(list))
		for _, value := range list {
			e.text(value)
		}
	}
	e.count(len(x.Checks))
	for _, k := range x.Checks {
		for _, value := range []string{k.Name, k.Kind, k.Value, k.Currency, k.Document, k.Period} {
			e.text(value)
		}
	}
	return e.out.Bytes()
}

func (r *Registry) scoped(c Change) bool {
	switch c.Kind {
	case ChangeExpectation:
		return r.scenarios[c.Expectation.Scenario] != nil
	case ChangeCell:
		return r.profiles[c.Cell.Profile] != nil
	case ChangeProof:
		return r.targets[c.Proof.Target] != nil
	}
	return true
}

func (r *Registry) check(c Change) error {
	switch c.Kind {
	case ChangeDataset:
		return r.checkDataset(c.Dataset)
	case ChangeScenario:
		return r.checkScenario(c.Scenario)
	case ChangeExpectation:
		return r.checkExpectation(c.Expectation)
	case ChangeProfile:
		switch {
		case r.profiles[c.Profile.ID] != nil:
			return ErrConflict
		case len(r.profiles) >= maxProfiles:
			return ErrLimit
		}
	case ChangeCell:
		return r.checkCell(c.Cell)
	case ChangeTarget:
		for _, id := range c.Target.Scenarios {
			if r.scenarios[id] == nil {
				return ErrNotFound
			}
		}
		switch {
		case r.targets[c.Target.ID] != nil:
			return ErrConflict
		case len(r.targets) >= maxTargets:
			return ErrLimit
		}
	case ChangeProof:
		if r.outputs[c.Proof.Evidence] || r.evidences[c.Proof.Evidence] {
			return refuse(ReasonRunOutput)
		}
	case ChangeHarness:
		switch {
		case r.harnesses[c.Harness.ID] != nil:
			return ErrConflict
		case r.identities[c.Harness.Identity] != "":
			return refuse(ReasonIdentityTaken)
		case len(r.harnesses) >= maxHarnesses:
			return ErrLimit
		}
	}
	return nil
}

func (r *Registry) checkDataset(d Dataset) error {
	var parent *Dataset
	if d.Origin == OriginConverted {
		if parent = r.datasets[d.Parent]; parent == nil {
			return ErrNotFound
		}
	}
	switch {
	case r.datasets[d.ID] != nil:
		return ErrConflict
	case parent != nil && parent.Origin == OriginSynthetic:
		return refuse(ReasonWrongOrigin)
	case parent != nil && (parent.License != d.License || parent.Notice != d.Notice):
		return refuse(ReasonNoticeChanged)
	case len(r.datasets) >= maxDatasets:
		return ErrLimit
	}
	return nil
}

func (r *Registry) checkScenario(s Scenario) error {
	for _, id := range s.Datasets {
		if r.datasets[id] == nil {
			return ErrNotFound
		}
	}
	if s.Profile != "" && r.profiles[s.Profile] == nil {
		return ErrNotFound
	}
	if r.scenarios[s.ID] != nil {
		return ErrConflict
	}
	for _, id := range s.Datasets {
		if s.Purpose == PurposeBilling && r.datasets[id].Origin != OriginSynthetic {
			return refuse(ReasonPublicAnswer)
		}
	}
	if len(r.scenarios) >= maxScenarios {
		return ErrLimit
	}
	return nil
}

func (r *Registry) checkExpectation(x Expectation) error {
	s := r.scenarios[x.Scenario]
	if x.Version != len(s.expectations)+1 {
		return ErrConflict
	}
	if r.ran[s.fingerprint] {
		return refuse(ReasonRunsRecorded)
	}
	if r.outputs[x.Evidence] {
		return refuse(ReasonRunOutput)
	}
	totals, amounts := map[string]bool{}, map[string]bool{}
	for _, k := range x.Checks {
		if k.Kind != CheckAmount {
			continue
		}
		if s.view.Purpose != PurposeBilling {
			return refuse(ReasonWrongPurpose)
		}
		amounts[k.Currency] = true
		if k.Document == "" && k.Period == "" {
			totals[k.Currency] = true
		}
	}
	switch {
	case s.view.Purpose == PurposeBilling && len(amounts) == 0:
		return refuse(ReasonNoAmount)
	case len(totals) != len(amounts):
		return refuse(ReasonNoTotal)
	}
	return nil
}

func (r *Registry) checkCell(c Cell) error {
	if c.Coverage == CoverageUnsupported {
		d := r.datasets[c.Dataset]
		switch {
		case d == nil:
			return ErrNotFound
		case d.Origin == OriginSynthetic:
			return refuse(ReasonWrongOrigin)
		}
		return nil
	}
	if c.Scenario == "" {
		return nil
	}
	s := r.scenarios[c.Scenario]
	if s == nil {
		return ErrNotFound
	}
	switch {
	case s.view.Purpose != PurposeBilling:
		return refuse(ReasonWrongPurpose)
	case s.view.Profile != c.Profile || !listed(s.view.Covers, c.Event):
		return refuse(ReasonNotCovered)
	case len(s.expectations) == 0:
		return refuse(ReasonNoExpectation)
	}
	kinds := map[string]string{}
	for _, k := range s.expectations[len(s.expectations)-1].view.Checks {
		kinds[k.Name] = k.Kind
	}
	amount := c.Event == EventPartialCSVError
	for _, name := range c.Checks {
		switch kinds[name] {
		case "":
			return refuse(ReasonNotCovered)
		case CheckAmount:
			amount = true
		}
	}
	if !amount {
		return refuse(ReasonNoAmount)
	}
	return nil
}

func listed(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

func (r *Registry) apply(c Change, proposer, approver string, now time.Time) {
	c = copyChange(c)
	switch c.Kind {
	case ChangeDataset:
		d := c.Dataset
		r.datasets[d.ID] = &d
	case ChangeScenario:
		r.scenarios[c.Scenario.ID] = &scenario{view: c.Scenario, fingerprint: fingerprint(c.Scenario, r.digests(c.Scenario.Datasets))}
	case ChangeExpectation:
		s := r.scenarios[c.Expectation.Scenario]
		s.expectations = append(s.expectations, approved{view: c.Expectation, author: proposer, approver: approver})
		r.evidences[c.Expectation.Evidence] = true
	case ChangeProfile:
		cells := make([]*cell, len(Events))
		for index, event := range Events {
			cells[index] = &cell{view: Cell{Profile: c.Profile.ID, Event: event}}
		}
		r.profiles[c.Profile.ID] = cells
	case ChangeCell:
		x := r.profiles[c.Cell.Profile][eventIndex(c.Cell.Event)]
		if x.set {
			status := r.cellStatus(x, currents{})
			if status == CellFailed {
				x.failures++
			}
			x.history = append(x.history, CellChange{Replaced: copyCell(x.view), Status: status})
			if len(x.history) > maxHistory {
				x.history = x.history[1:]
			}
			x.revisions++
		}
		x.view, x.set = c.Cell, true
	case ChangeTarget:
		r.targets[c.Target.ID] = &target{view: c.Target}
	case ChangeProof:
		t := r.targets[c.Proof.Target]
		count := 1
		if t.proof != nil {
			count = t.proof.Count + 1
		}
		t.proof = &ProofView{Environment: c.Proof.Environment, Evidence: c.Proof.Evidence, Approver: approver, At: now, Count: count}
	case ChangeHarness:
		r.harnesses[c.Harness.ID] = &harness{identity: c.Harness.Identity, proposer: proposer, approver: approver}
		r.identities[c.Harness.Identity] = c.Harness.ID
	}
}

func copyCell(c Cell) Cell {
	c.Checks = append([]string(nil), c.Checks...)
	return c
}

func (r *Registry) digests(ids []string) []string {
	out := []string{}
	for _, id := range ids {
		out = append(out, r.datasets[id].Digest)
	}
	sort.Strings(out)
	return out
}

func fingerprint(s Scenario, digests []string) string {
	var e encoder
	for _, value := range []string{"hybrid-billing/simulation-scenario/v1", s.Topology, s.Deployment, s.Seed, s.Zone, s.Relations, s.Policy, s.FX, s.Events} {
		e.text(value)
	}
	e.number(s.Start.UnixNano())
	e.number(s.EventCount)
	e.count(len(digests))
	for _, value := range digests {
		e.text(value)
	}
	return digest(e.out.Bytes())
}
