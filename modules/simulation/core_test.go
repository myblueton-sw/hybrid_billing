package simulation

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestConstants(t *testing.T) {
	got := []int{maxDatasets, maxScenarios, maxVersions, maxRuns, maxRunsTotal, maxKeys, maxHarnesses, maxProfiles, maxTargets, maxPending, maxMine, maxDone,
		maxList, maxChecks, maxLinks, maxPage, maxEventCount, maxDecimalBytes, maxCountDigits, maxZoneBytes, maxHistory, maxValues, maxResultBytes, maxIDBytes, maxURLBytes}
	want := []int{10000, 10000, 16, 1000, 200000, 200000, 100, 1000, 1000, 1000, 250, 1000, 64, 256, 32, 1000, 10000000, 40, 19, 64, 16, 8, 128, 128, 200}
	for index := range want {
		if got[index] != want[index] {
			t.Errorf("constant %d = %d", index, got[index])
		}
	}
	if len(Events) != 18 {
		t.Fatalf("events %d", len(Events))
	}
	expect(t, "clock", second(NewRegistry(nil)), ErrInvalid)
	if !errors.Is(Refusal{Reason: "x"}, ErrRejected) || (Refusal{Reason: "x"}).Error() != ErrRejected.Error()+": x" {
		t.Error("refusal text")
	}
}

func TestValidators(t *testing.T) {
	cases := []struct {
		name string
		fn   func(string) bool
		ok   []string
		bad  []string
	}{
		{"semver", semver, []string{"0.0.0", "1.20.3", "999999999.0.1"}, []string{"", "1.2", "1.2.3.4", "01.2.3", "1.2.x", "1..3", "1000000000.0.0"}},
		{"seed", validSeed, []string{"0", "7", "18446744073709551615"}, []string{"", "-1", "07", "18446744073709551616", "1a"}},
		{"decimal", validDecimal, []string{"0", "0.9", "-1", "10.5", "0.01", "-0.5", strings.Repeat("9", 40)},
			[]string{"", "-0", "-", "00", "01", "1.", "1.50", ".5", "1.9a", "1.:", "1.2.3", "+1", "1e3", "1.a", strings.Repeat("9", 41), "--1"}},
		{"hex", validHex, []string{sum("a")}, []string{"", strings.ToUpper(sum("a")), sum("a")[:63] + "g", sum("a") + "0"}},
		{"currency", validCurrency, []string{"KRW", "USD", "AUD", "ZAR"}, []string{"", "krw", "KR", "KRWX", "K1W", "1RW", "KR@", "KR["}},
		{"status", validStatus, []string{"ok", "a-1", "a", "zone"}, []string{"", "Ok", "1ok", "123.45", "-ok", "_ok", "a b"}},
		{"period", validPeriod, []string{"2026-01", "2026-09", "2026-12", "1999-10"}, []string{"", "2026-00", "2026-13", "2026-1", "2026/01", "0999-01", "20a6-01", "2026-1a", "2026-a1", "2026-0:", "2026-0/", "2026-1/", "2026-1:", "2026-:1", "2026-/1"}},
		{"zone", validZone, []string{"UTC", "Asia/Seoul", "Etc/GMT+9", "America/Port_of_Spain", "Z", "z", "GMT0", strings.Repeat("a", 64)},
			[]string{"", "Asia Seoul", "Asia.Seoul", ".UTC", strings.Repeat("a", 65)}},
		{"source", validSource, []string{"https://example.org/a.csv", "https://e" + strings.Repeat("x", 191)},
			[]string{"", "http://example.org", "https://", "https://u@example.org", "https://e.org/?q", "https://e.org/#f", "https://e.org/a b", "https://e.org/\x7f", "https:// x", "https://\x7fx",
				"https://e" + strings.Repeat("x", 192)}},
	}
	cases = append(cases, struct {
		name string
		fn   func(string) bool
		ok   []string
		bad  []string
	}{"id", validID, []string{"A", "Z", "a", "z", "0", "9", "a.b_c:d-e"}, []string{"", "!a", " a", "a b", "a@", "a[", "a`", "a{", "a/", strings.Repeat("a", 129)}},
		struct {
			name string
			fn   func(string) bool
			ok   []string
			bad  []string
		}{"hex start", validHex, []string{}, []string{"g" + sum("a")[1:], "G" + sum("a")[1:]}})
	for _, c := range cases {
		for _, value := range c.ok {
			if !c.fn(value) {
				t.Errorf("%s refused %q", c.name, value)
			}
		}
		for _, value := range c.bad {
			if c.fn(value) {
				t.Errorf("%s accepted %q", c.name, value)
			}
		}
	}
	less := [][2]string{{"1.0.0", "2.0.0"}, {"9.0.0", "10.0.0"}, {"1.2.3", "1.2.4"}, {"1.9.0", "1.10.0"}}
	for _, pair := range less {
		if !semverLess(pair[0], pair[1]) || semverLess(pair[1], pair[0]) {
			t.Errorf("order %v", pair)
		}
	}
	if semverLess("1.2.3", "1.2.3") {
		t.Error("equal versions ordered")
	}
	if !ascending([]string{"a", "b"}, 1, 2, validID) || ascending([]string{"b", "a"}, 1, 2, validID) || ascending([]string{"a", "a"}, 1, 2, validID) ||
		ascending(nil, 1, 2, validID) || ascending([]string{"a", "b", "c"}, 1, 2, validID) || ascending([]string{"a", "b c"}, 1, 2, validID) || !ascending(nil, 0, 2, validID) {
		t.Error("ascending")
	}
}

func TestDatasetForms(t *testing.T) {
	for name, d := range map[string]Dataset{"public": public("a"), "converted": converted("b", "a"), "synthetic": synthetic("c")} {
		if !d.valid() {
			t.Errorf("%s fixture refused", name)
		}
	}
	bad := map[string]func(*Dataset){
		"id":       func(d *Dataset) { d.ID = "" },
		"version":  func(d *Dataset) { d.Version = "1.0" },
		"digest":   func(d *Dataset) { d.Digest = "x" },
		"coverage": func(d *Dataset) { d.Coverage = nil },
		"origin":   func(d *Dataset) { d.Origin = "scraped" },
	}
	for name, mutate := range bad {
		d := public("a")
		mutate(&d)
		if d.valid() {
			t.Errorf("%s accepted", name)
		}
	}
	origin := map[string]func() Dataset{
		"public source":     func() Dataset { d := public("a"); d.Source = "ftp://x"; return d },
		"public license":    func() Dataset { d := public("a"); d.License = ""; return d },
		"public notice":     func() Dataset { d := public("a"); d.Notice = ""; return d },
		"public parent":     func() Dataset { d := public("a"); d.Parent = "x"; return d },
		"public tool":       func() Dataset { d := public("a"); d.Tool = "x"; return d },
		"public toolver":    func() Dataset { d := public("a"); d.ToolVersion = "1.0.0"; return d },
		"public seed":       func() Dataset { d := public("a"); d.Seed = "1"; return d },
		"converted parent":  func() Dataset { d := converted("b", "a"); d.Parent = ""; return d },
		"converted tool":    func() Dataset { d := converted("b", "a"); d.Tool = ""; return d },
		"converted toolver": func() Dataset { d := converted("b", "a"); d.ToolVersion = ""; return d },
		"converted license": func() Dataset { d := converted("b", "a"); d.License = ""; return d },
		"converted notice":  func() Dataset { d := converted("b", "a"); d.Notice = ""; return d },
		"converted seed":    func() Dataset { d := converted("b", "a"); d.Seed = "1"; return d },
		"converted source":  func() Dataset { d := converted("b", "a"); d.Source = "https://x"; return d },
		"synthetic tool":    func() Dataset { d := synthetic("c"); d.Tool = ""; return d },
		"synthetic toolver": func() Dataset { d := synthetic("c"); d.ToolVersion = "x"; return d },
		"synthetic seed":    func() Dataset { d := synthetic("c"); d.Seed = "-1"; return d },
		"synthetic source":  func() Dataset { d := synthetic("c"); d.Source = "https://x"; return d },
		"synthetic license": func() Dataset { d := synthetic("c"); d.License = "x"; return d },
		"synthetic notice":  func() Dataset { d := synthetic("c"); d.Notice = sum("n"); return d },
		"synthetic parent":  func() Dataset { d := synthetic("c"); d.Parent = "a"; return d },
	}
	for name, build := range origin {
		if build().valid() {
			t.Errorf("%s accepted", name)
		}
	}
}

func TestScenarioForms(t *testing.T) {
	if !billing("s").valid() || !format("f").valid() {
		t.Fatal("fixture refused")
	}
	bad := map[string]func(*Scenario){
		"id":          func(s *Scenario) { s.ID = "" },
		"fixture":     func(s *Scenario) { s.FixtureVersion = "1" },
		"purpose":     func(s *Scenario) { s.Purpose = "demo" },
		"topology":    func(s *Scenario) { s.Topology = "x" },
		"deployment":  func(s *Scenario) { s.Deployment = "x" },
		"seed":        func(s *Scenario) { s.Seed = "" },
		"zero start":  func(s *Scenario) { s.Start = time.Time{} },
		"early start": func(s *Scenario) { s.Start = time.Date(1999, 12, 31, 23, 0, 0, 0, time.UTC) },
		"late start":  func(s *Scenario) { s.Start = time.Date(2200, 1, 1, 0, 0, 0, 0, time.UTC) },
		"zone":        func(s *Scenario) { s.Zone = "" },
		"relations":   func(s *Scenario) { s.Relations = "" },
		"policy":      func(s *Scenario) { s.Policy = "" },
		"fx":          func(s *Scenario) { s.FX = "" },
		"datasets":    func(s *Scenario) { s.Datasets = nil },
		"events":      func(s *Scenario) { s.Events = "" },
		"negative":    func(s *Scenario) { s.EventCount = -1 },
		"many":        func(s *Scenario) { s.EventCount = maxEventCount + 1 },
		"covers only": func(s *Scenario) { s.Profile = "" },
		"profile":     func(s *Scenario) { s.Profile = "a b" },
		"format prof": func(s *Scenario) { s.Purpose = PurposeFormat },
		"covers":      func(s *Scenario) { s.Covers = []string{"overage", "bogus"} },
		"covers ord":  func(s *Scenario) { s.Covers = []string{EventRefund, EventOverage} },
	}
	for name, mutate := range bad {
		s := billing("s")
		mutate(&s)
		if s.valid() {
			t.Errorf("%s accepted", name)
		}
	}
	edge := billing("s")
	edge.Start, edge.EventCount, edge.Covers = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), maxEventCount, nil
	late := billing("s")
	late.Start, late.EventCount, late.Covers = time.Date(2199, 12, 31, 23, 0, 0, 0, time.UTC), 0, append([]string(nil), Events...)
	sorted := append([]string(nil), Events...)
	for i := range sorted {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j] < sorted[i] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	late.Covers = sorted
	if !edge.valid() || !late.valid() {
		t.Error("edges refused")
	}
}

func TestExpectationAndCheckForms(t *testing.T) {
	good := expectation("s1", 1, standardChecks()...)
	if !good.valid() {
		t.Fatal("fixture refused")
	}
	bad := map[string]func(*Expectation){
		"scenario": func(e *Expectation) { e.Scenario = "" },
		"zero":     func(e *Expectation) { e.Version = 0 },
		"high":     func(e *Expectation) { e.Version = maxVersions + 1 },
		"method":   func(e *Expectation) { e.Method = "engine" },
		"evidence": func(e *Expectation) { e.Evidence = "" },
		"empty":    func(e *Expectation) { e.Checks = nil },
		"order":    func(e *Expectation) { e.Checks[0], e.Checks[1] = e.Checks[1], e.Checks[0] },
		"repeat":   func(e *Expectation) { e.Checks[1] = e.Checks[0] },
		"check":    func(e *Expectation) { e.Checks[2].Currency = "" },
	}
	for name, mutate := range bad {
		e := expectation("s1", 1, standardChecks()...)
		mutate(&e)
		if e.valid() {
			t.Errorf("%s accepted", name)
		}
	}
	many := expectation("s1", maxVersions, Check{Name: "a", Kind: CheckCount, Value: "1"})
	many.Method = MethodIndependent
	for len(many.Checks) < maxChecks {
		many.Checks = append(many.Checks, Check{Name: "c" + strings.Repeat("0", 3-len(itoa(len(many.Checks)))) + itoa(len(many.Checks)), Kind: CheckCount, Value: "1"})
	}
	if !many.valid() {
		t.Error("exact caps refused")
	}
	many.Checks = append(many.Checks, Check{Name: "z", Kind: CheckCount, Value: "1"})
	if many.valid() {
		t.Error("check cap exceeded")
	}
	checks := map[string]struct {
		c  Check
		ok bool
	}{
		"amount":       {amount("a", "1", "KRW"), true},
		"amount cur":   {amount("a", "1", ""), false},
		"amount value": {amount("a", "1.0", "KRW"), false},
		"count":        {Check{Name: "a", Kind: CheckCount, Value: strings.Repeat("9", 19)}, true},
		"count long":   {Check{Name: "a", Kind: CheckCount, Value: strings.Repeat("9", 20)}, false},
		"count neg":    {Check{Name: "a", Kind: CheckCount, Value: "-1"}, false},
		"count cur":    {Check{Name: "a", Kind: CheckCount, Value: "1", Currency: "KRW"}, false},
		"digest":       {Check{Name: "a", Kind: CheckDigest, Value: sum("x")}, true},
		"digest bad":   {Check{Name: "a", Kind: CheckDigest, Value: "x"}, false},
		"digest cur":   {Check{Name: "a", Kind: CheckDigest, Value: sum("x"), Currency: "KRW"}, false},
		"status":       {Check{Name: "a", Kind: CheckStatus, Value: "issued"}, true},
		"status num":   {Check{Name: "a", Kind: CheckStatus, Value: "123.45"}, false},
		"status cur":   {Check{Name: "a", Kind: CheckStatus, Value: "ok", Currency: "KRW"}, false},
		"kind":         {Check{Name: "a", Kind: "ratio", Value: "1"}, false},
		"name":         {Check{Name: "", Kind: CheckCount, Value: "1"}, false},
		"document":     {Check{Name: "a", Kind: CheckCount, Value: "1", Document: "inv-1"}, true},
		"document bad": {Check{Name: "a", Kind: CheckCount, Value: "1", Document: "inv 1"}, false},
		"period":       {Check{Name: "a", Kind: CheckCount, Value: "1", Period: "2026-11"}, true},
		"period bad":   {Check{Name: "a", Kind: CheckCount, Value: "1", Period: "2026-13"}, false},
	}
	for name, c := range checks {
		if c.c.valid() != c.ok {
			t.Errorf("%s valid = %v", name, !c.ok)
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	out := ""
	for ; n > 0; n /= 10 {
		out = string(rune('0'+n%10)) + out
	}
	return out
}

func TestCellAndOtherForms(t *testing.T) {
	cells := map[string]struct {
		c  Cell
		ok bool
	}{
		"tested":             {Cell{Profile: "p", Event: EventRefund, Coverage: CoverageTested, Scenario: "s", Checks: []string{"a"}}, true},
		"tested no checks":   {Cell{Profile: "p", Event: EventRefund, Coverage: CoverageTested, Scenario: "s"}, false},
		"tested no scenario": {Cell{Profile: "p", Event: EventRefund, Coverage: CoverageTested, Checks: []string{"a"}}, false},
		"tested dataset":     {Cell{Profile: "p", Event: EventRefund, Coverage: CoverageTested, Scenario: "s", Checks: []string{"a"}, Dataset: "d"}, false},
		"tested evidence":    {Cell{Profile: "p", Event: EventRefund, Coverage: CoverageTested, Scenario: "s", Checks: []string{"a"}, Evidence: sum("e")}, false},
		"tested decision":    {Cell{Profile: "p", Event: EventRefund, Coverage: CoverageTested, Scenario: "s", Checks: []string{"a"}, Decision: "d"}, false},
		"unsupported":        {Cell{Profile: "p", Event: EventRefund, Coverage: CoverageUnsupported, Dataset: "d", Evidence: sum("e")}, true},
		"unsup no dataset":   {Cell{Profile: "p", Event: EventRefund, Coverage: CoverageUnsupported, Evidence: sum("e")}, false},
		"unsup no evidence":  {Cell{Profile: "p", Event: EventRefund, Coverage: CoverageUnsupported, Dataset: "d"}, false},
		"unsup scenario":     {Cell{Profile: "p", Event: EventRefund, Coverage: CoverageUnsupported, Dataset: "d", Evidence: sum("e"), Scenario: "s"}, false},
		"unsup checks":       {Cell{Profile: "p", Event: EventRefund, Coverage: CoverageUnsupported, Dataset: "d", Evidence: sum("e"), Checks: []string{"a"}}, false},
		"unsup decision":     {Cell{Profile: "p", Event: EventRefund, Coverage: CoverageUnsupported, Dataset: "d", Evidence: sum("e"), Decision: "x"}, false},
		"undecided":          {Cell{Profile: "p", Event: EventRefund, Coverage: CoverageUndecided, Decision: "d"}, true},
		"undecided linked":   {Cell{Profile: "p", Event: EventRefund, Coverage: CoverageUndecided, Decision: "d", Scenario: "s", Checks: []string{"a"}}, true},
		"undecided half":     {Cell{Profile: "p", Event: EventRefund, Coverage: CoverageUndecided, Decision: "d", Scenario: "s"}, false},
		"undecided checks":   {Cell{Profile: "p", Event: EventRefund, Coverage: CoverageUndecided, Decision: "d", Checks: []string{"a"}}, false},
		"undecided none":     {Cell{Profile: "p", Event: EventRefund, Coverage: CoverageUndecided}, false},
		"undecided dataset":  {Cell{Profile: "p", Event: EventRefund, Coverage: CoverageUndecided, Decision: "d", Dataset: "x"}, false},
		"undecided evidence": {Cell{Profile: "p", Event: EventRefund, Coverage: CoverageUndecided, Decision: "d", Evidence: sum("e")}, false},
		"coverage":           {Cell{Profile: "p", Event: EventRefund, Coverage: "skipped", Decision: "d"}, false},
		"event":              {Cell{Profile: "p", Event: "bogus", Coverage: CoverageUndecided, Decision: "d"}, false},
		"profile":            {Cell{Event: EventRefund, Coverage: CoverageUndecided, Decision: "d"}, false},
	}
	for name, c := range cells {
		if c.c.valid() != c.ok {
			t.Errorf("%s valid = %v", name, !c.ok)
		}
	}
	changes := map[string]Change{
		"kind":           {Kind: "other"},
		"profile":        {Kind: ChangeProfile},
		"target id":      {Kind: ChangeTarget, Target: Target{Kind: TargetCSP, Scenarios: []string{"s"}}},
		"target kind":    {Kind: ChangeTarget, Target: Target{ID: "t", Kind: "bank", Scenarios: []string{"s"}}},
		"target list":    {Kind: ChangeTarget, Target: Target{ID: "t", Kind: TargetERP}},
		"proof target":   {Kind: ChangeProof, Proof: Proof{Environment: "e", Evidence: sum("e")}},
		"proof env":      {Kind: ChangeProof, Proof: Proof{Target: "t", Evidence: sum("e")}},
		"proof evidence": {Kind: ChangeProof, Proof: Proof{Target: "t", Environment: "e"}},
		"harness id":     {Kind: ChangeHarness, Harness: Harness{Identity: "x"}},
		"harness ident":  {Kind: ChangeHarness, Harness: Harness{ID: "x"}},
		"dataset":        {Kind: ChangeDataset},
		"scenario":       {Kind: ChangeScenario},
		"expectation":    {Kind: ChangeExpectation},
		"cell":           {Kind: ChangeCell},
	}
	for name, c := range changes {
		if _, ok := clean(c); ok {
			t.Errorf("%s accepted", name)
		}
	}
	for _, c := range []Change{{Kind: ChangeTarget, Target: Target{ID: "t", Kind: TargetProduct, Scenarios: []string{"s"}}}, {Kind: ChangeProof, Proof: Proof{Target: "t", Environment: "e", Evidence: sum("e")}}} {
		if _, ok := clean(c); !ok {
			t.Errorf("%s refused", c.Kind)
		}
	}
	run := Run{Scenario: "s", Engine: "1.0.0", Rules: sum("r"), Output: sum("o"), Results: []Result{{Name: "a", Value: "1"}, {Name: "b", Value: sum("x")}, {Name: "c", Value: "ok"}, {Name: "d", Value: "1.5", Currency: "KRW"}}}
	if !run.valid() {
		t.Fatal("run refused")
	}
	runs := map[string]func(*Run){
		"scenario": func(x *Run) { x.Scenario = "" },
		"engine":   func(x *Run) { x.Engine = "1" },
		"rules":    func(x *Run) { x.Rules = "" },
		"output":   func(x *Run) { x.Output = "" },
		"order":    func(x *Run) { x.Results[0], x.Results[1] = x.Results[1], x.Results[0] },
		"name":     func(x *Run) { x.Results[0].Name = "" },
		"value":    func(x *Run) { x.Results[0].Value = "" },
		"form":     func(x *Run) { x.Results[0].Value = "1.0" },
		"currency": func(x *Run) { x.Results[3].Currency = "krw" },
		"long":     func(x *Run) { x.Results[2].Value = "a" + strings.Repeat("b", maxResultBytes) },
		"many": func(x *Run) {
			for len(x.Results) <= maxChecks {
				x.Results = append(x.Results, Result{Name: "z" + itoa(1000+len(x.Results)), Value: "1"})
			}
		},
	}
	for name, mutate := range runs {
		x := run
		x.Results = append([]Result(nil), run.Results...)
		mutate(&x)
		if x.valid() {
			t.Errorf("run %s accepted", name)
		}
	}
	full := run
	full.Results = nil
	for len(full.Results) < maxChecks {
		full.Results = append(full.Results, Result{Name: "r" + itoa(1000+len(full.Results)), Value: "1"})
	}
	if !full.valid() {
		t.Error("results at cap refused")
	}
	dup := run
	dup.Results = []Result{{Name: "a", Value: "1"}, {Name: "a", Value: "2"}}
	if dup.valid() {
		t.Error("duplicate result names accepted")
	}
	edge := run
	edge.Results = []Result{{Name: "a", Value: "a" + strings.Repeat("b", maxResultBytes-1)}}
	if !edge.valid() {
		t.Error("result at size cap refused")
	}
}
