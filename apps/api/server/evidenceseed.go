package server

import (
	"errors"
	"strconv"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/perfgate"
	"github.com/myblueton-sw/hybrid_billing/modules/releasetrace"
)

const (
	moduleTrace                 = "release_trace"
	modulePerf                  = "perf_gate"
	entryPending                = releasetrace.ProposalPending
	storeRetired                = "store_retired"
	evidenceNamespace           = "sample"
	evidenceCandidateID         = "sample-candidate-1"
	evidenceOldBuild            = "0.0.1"
	evidenceBuild               = "0.0.2"
	evidenceReviewer1           = "sample-evidence-reviewer-1"
	evidenceReviewer2           = "sample-evidence-reviewer-2"
	evidenceTraceRunner         = "sample-trace-runner"
	evidencePerfRunner          = "sample-perf-runner"
	evidenceTraceHarness        = "sample-trace-harness"
	evidencePerfHarness         = "sample-perf-harness"
	evidenceHardware            = "sample-hw-small"
	evidenceRetireReason        = "Sample scope change"
	noResultRecorded            = "no_result_recorded"
	notCountedByModule          = "not_counted_by_module"
	noImplementation            = "no_implementation"
	buildPredatesImplementation = "build_predates_implementation"
)

type evidenceBuildRecord struct {
	id     string
	commit string
}

type evidenceHarnessRecord struct {
	module string
	id     string
}

type evidenceResultKey struct {
	namespace string
	test      string
	build     string
}

type evidenceEntry struct {
	module    string
	number    uint64
	kind      string
	state     string
	proposer  string
	approver  string
	trace     releasetrace.Change
	perf      perfgate.Change
	created   time.Time
	rationale string
	closedBy  string
	closing   string
	applied   uint64
}

type evidenceStore struct {
	seeded       bool
	applied      uint64
	requirements []releasetrace.Key
	builds       []evidenceBuildRecord
	paths        []string
	profiles     []string
	hardware     []string
	budgets      []string
	harnesses    []evidenceHarnessRecord
	results      []evidenceResultKey
	candidate    *releasetrace.Release
	entries      []*evidenceEntry
}

func newEvidenceStore() *evidenceStore { return &evidenceStore{} }

func (e *evidenceEntry) setTrace(v releasetrace.ProposalView) {
	e.number, e.kind, e.state, e.proposer, e.approver, e.trace = v.Number, v.Change.Kind, v.State, v.Proposer, v.Approver, v.Change
}

func (e *evidenceEntry) setPerf(v perfgate.ProposalView) {
	e.number, e.kind, e.state, e.proposer, e.approver, e.perf = v.Number, v.Change.Kind, v.State, v.Proposer, v.Approver, v.Change
}

func (c *evidenceStore) entry(module string, number uint64) *evidenceEntry {
	for _, e := range c.entries {
		if e.module == module && e.number == number {
			return e
		}
	}
	return nil
}

func (c *evidenceStore) keep(e *evidenceEntry) {
	c.entries = append(c.entries, e)
	if len(c.entries) <= installKept {
		return
	}
	for index, old := range c.entries {
		if old.state != entryPending {
			c.entries = append(c.entries[:index], c.entries[index+1:]...)
			return
		}
	}
}

func (c *evidenceStore) requirementIndex(k releasetrace.Key) int {
	for index, stored := range c.requirements {
		if stored == k {
			return index + 1
		}
	}
	return 0
}

func (c *evidenceStore) hasProfile(id string) bool {
	for _, stored := range c.profiles {
		if stored == id {
			return true
		}
	}
	return false
}

func (c *evidenceStore) pendingRetire(k releasetrace.Key) *evidenceEntry {
	for _, e := range c.entries {
		if e.module == moduleTrace && e.state == entryPending && e.kind == releasetrace.ChangeRetire && e.trace.Retire.Key == k {
			return e
		}
	}
	return nil
}

func (c *evidenceStore) pendingReplacement(k releasetrace.Key) bool {
	for _, e := range c.entries {
		if e.module == moduleTrace && e.state == entryPending && e.kind == releasetrace.ChangeRetire && e.trace.Retire.Replacement == k {
			return true
		}
	}
	return false
}

func (c *evidenceStore) pendingReplacements() map[releasetrace.Key]bool {
	out := map[releasetrace.Key]bool{}
	for _, e := range c.entries {
		if e.module == moduleTrace && e.state == entryPending && e.kind == releasetrace.ChangeRetire {
			out[e.trace.Retire.Replacement] = true
		}
	}
	return out
}

func (c *evidenceStore) pendingProfileRetire(id string) *evidenceEntry {
	for _, e := range c.entries {
		if e.module == modulePerf && e.state == entryPending && e.kind == perfgate.ChangeRetire && e.perf.Retire.Profile == id {
			return e
		}
	}
	return nil
}

func (c *evidenceStore) appliedTrace(kind string, match func(releasetrace.Change) bool) uint64 {
	latest := uint64(0)
	for _, e := range c.entries {
		if e.module == moduleTrace && e.kind == kind && e.state == releasetrace.ProposalApplied && e.applied > latest && match(e.trace) {
			latest = e.applied
		}
	}
	return latest
}

func (c *evidenceStore) notCounted(k releasetrace.Key, build string) string {
	implemented := c.appliedTrace(releasetrace.ChangeImplemented, func(x releasetrace.Change) bool { return x.Implemented.Key == k })
	built := c.appliedTrace(releasetrace.ChangeBuild, func(x releasetrace.Change) bool { return x.Build.Build == build })
	if built != 0 && built < implemented {
		return buildPredatesImplementation
	}
	return notCountedByModule
}

func (c *evidenceStore) testOwners(k evidenceResultKey) []releasetrace.Key {
	owners := []releasetrace.Key{}
	for _, e := range c.entries {
		if e.module != moduleTrace || e.kind != releasetrace.ChangeRequirement || e.state != releasetrace.ProposalApplied {
			continue
		}
		q := e.trace.Requirement
		for _, t := range q.Tests {
			if q.Namespace == k.namespace && t.ID == k.test {
				owners = append(owners, releasetrace.Key{Namespace: q.Namespace, ID: q.ID})
			}
		}
	}
	return owners
}

func (c *evidenceStore) addResult(k evidenceResultKey) {
	for _, stored := range c.results {
		if stored == k {
			return
		}
	}
	c.results = append(c.results, k)
}

func (s *Server) traceCall(name string, call func() error) error {
	return s.moduleCall("trace-"+name, releasetrace.ErrInvalid, call)
}

func (s *Server) perfCall(name string, call func() error) error {
	return s.moduleCall("perf-"+name, perfgate.ErrInvalid, call)
}

func (s *Server) tracePropose(c releasetrace.Change, by releasetrace.Actor, rationale string) (*evidenceEntry, error) {
	var view releasetrace.ProposalView
	if err := s.traceCall("propose", func() (err error) { view, err = s.trace.Propose(c, by); return }); err != nil {
		return nil, err
	}
	e := s.ev.entry(moduleTrace, view.Number)
	if e == nil {
		e = &evidenceEntry{module: moduleTrace, created: s.clock(), rationale: rationale}
		e.setTrace(view)
		s.ev.keep(e)
	}
	return e, nil
}

func (s *Server) perfPropose(c perfgate.Change, by perfgate.Actor, rationale string) (*evidenceEntry, error) {
	var view perfgate.ProposalView
	if err := s.perfCall("propose", func() (err error) { view, err = s.perf.Propose(c, by); return }); err != nil {
		return nil, err
	}
	e := s.ev.entry(modulePerf, view.Number)
	if e == nil {
		e = &evidenceEntry{module: modulePerf, created: s.clock(), rationale: rationale}
		e.setPerf(view)
		s.ev.keep(e)
	}
	return e, nil
}

func retireEntry(e *evidenceEntry, gone bool, err error) error {
	if gone && e.state == entryPending {
		e.state = storeRetired
	}
	return err
}

func traceGone(err error) bool {
	var refusal releasetrace.Refusal
	return errors.Is(err, releasetrace.ErrNotFound) || errors.As(err, &refusal) && refusal.Reason == releasetrace.ReasonNotPending
}

func perfGone(err error) bool {
	var refusal perfgate.Refusal
	return errors.Is(err, perfgate.ErrNotFound) || errors.As(err, &refusal) && refusal.Reason == perfgate.ReasonNotPending
}

func (s *Server) traceApprove(e *evidenceEntry, by releasetrace.Actor, rationale string) error {
	var view releasetrace.ProposalView
	if err := s.traceCall("approve", func() (err error) { view, err = s.trace.Approve(e.number, by); return }); err != nil {
		return retireEntry(e, traceGone(err), err)
	}
	if e.state == entryPending {
		s.ev.applied++
		e.closedBy, e.closing, e.applied = by.ID, rationale, s.ev.applied
	}
	e.setTrace(view)
	return nil
}

func (s *Server) perfApprove(e *evidenceEntry, by perfgate.Actor, rationale string) error {
	var view perfgate.ProposalView
	if err := s.perfCall("approve", func() (err error) { view, err = s.perf.Approve(e.number, by); return }); err != nil {
		return retireEntry(e, perfGone(err), err)
	}
	if e.state == entryPending {
		e.closedBy, e.closing = by.ID, rationale
	}
	e.setPerf(view)
	return nil
}

func (s *Server) traceWithdraw(e *evidenceEntry, by releasetrace.Actor, rationale string) error {
	var view releasetrace.ProposalView
	if err := s.traceCall("withdraw", func() (err error) { view, err = s.trace.Withdraw(e.number, by); return }); err != nil {
		return retireEntry(e, traceGone(err), err)
	}
	e.setTrace(view)
	e.closedBy, e.closing = by.ID, rationale
	return nil
}

func (s *Server) perfWithdraw(e *evidenceEntry, by perfgate.Actor, rationale string) error {
	var view perfgate.ProposalView
	if err := s.perfCall("withdraw", func() (err error) { view, err = s.perf.Withdraw(e.number, by); return }); err != nil {
		return retireEntry(e, perfGone(err), err)
	}
	e.setPerf(view)
	e.closedBy, e.closing = by.ID, rationale
	return nil
}

func (s *Server) traceRequirement(k releasetrace.Key, build string) (releasetrace.RequirementView, error) {
	var view releasetrace.RequirementView
	err := s.traceCall("requirement", func() (err error) { view, err = s.trace.Requirement(k, build); return })
	return view, err
}

func (s *Server) tracePath(id string) (releasetrace.PathView, error) {
	var view releasetrace.PathView
	err := s.traceCall("path", func() (err error) { view, err = s.trace.Path(id); return })
	return view, err
}

func (s *Server) traceResults(k evidenceResultKey) (releasetrace.ResultView, error) {
	var view releasetrace.ResultView
	err := s.traceCall("results", func() (err error) { view, err = s.trace.Results(k.namespace, k.test, k.build); return })
	return view, err
}

func (s *Server) traceGate(x releasetrace.Release) (releasetrace.GateView, error) {
	var view releasetrace.GateView
	err := s.traceCall("gate", func() (err error) { view, err = s.trace.Gate(x); return })
	return view, err
}

func (s *Server) perfProfile(id string) (perfgate.Profile, bool, error) {
	var view perfgate.Profile
	var retired bool
	err := s.perfCall("profile", func() (err error) { view, retired, err = s.perf.Profile(id); return })
	return view, retired, err
}

func (s *Server) perfHardware(id string) (perfgate.Hardware, error) {
	var view perfgate.Hardware
	err := s.perfCall("hardware", func() (err error) { view, err = s.perf.Hardware(id); return })
	return view, err
}

func (s *Server) perfActiveBudget(profile string) (perfgate.BudgetView, error) {
	var view perfgate.BudgetView
	err := s.perfCall("active_budget", func() (err error) { view, err = s.perf.ActiveBudget(profile); return })
	return view, err
}

func (s *Server) perfRuns(profile string) ([]perfgate.RunView, error) {
	var list []perfgate.RunView
	err := s.perfCall("runs", func() (err error) { list, err = s.perf.Runs(profile, 0, evidenceRunPage); return })
	return list, err
}

func (s *Server) perfReadiness(build string) (perfgate.ReadinessView, error) {
	var view perfgate.ReadinessView
	err := s.perfCall("readiness", func() (err error) { view, err = s.perf.Readiness(build); return })
	return view, err
}

type evidenceRequirementSample struct {
	id       string
	common   bool
	blocking bool
	tests    []releasetrace.TestRef
}

func evidenceRequirementSamples() []evidenceRequirementSample {
	test := func(id, category string) releasetrace.TestRef {
		return releasetrace.TestRef{ID: id, Category: category}
	}
	return []evidenceRequirementSample{
		{"sample-common-xlsx", true, false, []releasetrace.TestRef{test("t-xlsx-amount", releasetrace.CategoryAmount), test("t-xlsx-isolation", releasetrace.CategoryIsolation)}},
		{"sample-approval", false, false, []releasetrace.TestRef{test("t-approval", releasetrace.CategoryApproval), test("t-duplicate", releasetrace.CategoryDuplicate)}},
		{"sample-recovery", false, true, []releasetrace.TestRef{test("t-bulk", releasetrace.CategoryBulk), test("t-recovery", releasetrace.CategoryRecovery)}},
		{"sample-erp-template", false, false, []releasetrace.TestRef{test("t-erp-template", "")}},
		{"sample-api-export", false, false, []releasetrace.TestRef{test("t-api-export", "")}},
		{"sample-legacy-report", false, false, []releasetrace.TestRef{test("t-legacy", "")}},
	}
}

type evidenceRunSample struct {
	profile  string
	rows     int64
	duration int64
	metrics  []perfgate.Measure
}

func evidenceMetrics(completion, rss, p95, p99, throughput int64) []perfgate.Measure {
	return []perfgate.Measure{{Metric: perfgate.MetricCompletion, Value: completion}, {Metric: perfgate.MetricDBIO, Value: 1073741824},
		{Metric: perfgate.MetricLockWait, Value: 40}, {Metric: perfgate.MetricMaxRSS, Value: rss}, {Metric: perfgate.MetricP95, Value: p95},
		{Metric: perfgate.MetricP99, Value: p99}, {Metric: perfgate.MetricThroughput, Value: throughput}, {Metric: perfgate.MetricWAL, Value: 536870912}}
}

func evidenceLimits(completion, throughput int64) []perfgate.Limit {
	return []perfgate.Limit{{Metric: perfgate.MetricCompletion, Value: completion}, {Metric: perfgate.MetricMaxRSS, Value: 4294967296},
		{Metric: perfgate.MetricP95, Value: 500}, {Metric: perfgate.MetricP99, Value: 1000}, {Metric: perfgate.MetricThroughput, Value: throughput}}
}

func (s *Server) seedEvidence(now time.Time) error {
	first := releasetrace.Actor{ID: SampleAdmin, Kind: releasetrace.KindHuman}
	second := releasetrace.Actor{ID: SampleAdmin2, Kind: releasetrace.KindHuman}
	reviewer1 := releasetrace.Actor{ID: evidenceReviewer1, Kind: releasetrace.KindHuman}
	reviewer2 := releasetrace.Actor{ID: evidenceReviewer2, Kind: releasetrace.KindHuman}
	traceRunner := releasetrace.Actor{ID: evidenceTraceRunner, Kind: releasetrace.KindSystem}
	perfFirst := perfgate.Actor{ID: SampleAdmin, Kind: perfgate.KindHuman}
	perfSecond := perfgate.Actor{ID: SampleAdmin2, Kind: perfgate.KindHuman}
	perfReviewer1 := perfgate.Actor{ID: evidenceReviewer1, Kind: perfgate.KindHuman}
	perfReviewer2 := perfgate.Actor{ID: evidenceReviewer2, Kind: perfgate.KindHuman}
	perfRunner := perfgate.Actor{ID: evidencePerfRunner, Kind: perfgate.KindSystem}
	traced := func(c releasetrace.Change, by, approver releasetrace.Actor, after func()) func() error {
		return func() error {
			e, err := s.tracePropose(c, by, installSeedReason)
			if err != nil {
				return err
			}
			if err := s.traceApprove(e, approver, installSeedReason); err != nil {
				return err
			}
			after()
			return nil
		}
	}
	measured := func(c perfgate.Change, by, approver perfgate.Actor, after func()) func() error {
		return func() error {
			e, err := s.perfPropose(c, by, installSeedReason)
			if err != nil {
				return err
			}
			if err := s.perfApprove(e, approver, installSeedReason); err != nil {
				return err
			}
			after()
			return nil
		}
	}
	key := func(id string) releasetrace.Key { return releasetrace.Key{Namespace: evidenceNamespace, ID: id} }
	build := func(id, label string) func() error {
		commit := sampleHash(label)[:40]
		return traced(releasetrace.Change{Kind: releasetrace.ChangeBuild, Build: releasetrace.Build{Build: id, Commit: commit}}, first, second, func() {
			s.ev.builds = append(s.ev.builds, evidenceBuildRecord{id: id, commit: commit})
		})
	}
	counts := map[evidenceResultKey]int{}
	result := func(test, build string, passed bool) func() error {
		return func() error {
			k := evidenceResultKey{namespace: evidenceNamespace, test: test, build: build}
			counts[k]++
			x := releasetrace.Result{Namespace: evidenceNamespace, Test: test, Build: build, Passed: passed,
				Evidence: sampleHash("sample:result:" + test + ":" + build + ":" + strconv.Itoa(counts[k]))}
			if err := s.traceCall("record", func() error { _, err := s.trace.Record(x, traceRunner); return err }); err != nil {
				return err
			}
			s.ev.addResult(k)
			return nil
		}
	}
	steps := []func() error{}
	samples := evidenceRequirementSamples()
	for _, q := range samples {
		r := releasetrace.Requirement{Namespace: evidenceNamespace, ID: q.id, Source: "sample-source", Owner: "sample-owner", Interfaces: []string{"sample-api"},
			Tests: q.tests, Common: q.common, Blocking: q.blocking, Design: sampleHash("sample:design:" + q.id)}
		steps = append(steps, traced(releasetrace.Change{Kind: releasetrace.ChangeRequirement, Requirement: r}, first, second, func() {
			s.ev.requirements = append(s.ev.requirements, key(q.id))
		}))
	}
	steps = append(steps, build(evidenceOldBuild, "sample:commit:0"))
	for _, q := range samples {
		if q.id == "sample-erp-template" {
			continue
		}
		steps = append(steps, traced(releasetrace.Change{Kind: releasetrace.ChangeImplemented, Implemented: releasetrace.Implemented{Key: key(q.id),
			Commit: sampleHash("sample:commit:1")[:40]}}, first, second, func() {}))
	}
	period := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, time.UTC).Format("2006-01")
	reconcile := func(path string, differences int) func() error {
		return traced(releasetrace.Change{Kind: releasetrace.ChangeReconcile, Reconcile: releasetrace.Reconcile{Path: path, Period: period,
			Evidence: sampleHash("sample:reconcile:" + path + ":" + period), Matched: differences == 0, Differences: differences}}, first, second, func() {})
	}
	path := func(id, issuance, template string, requirements ...string) func() error {
		p := releasetrace.Path{ID: id, Provider: "sample-provider", Contract: "sample-contract", Role: "sample-role", Version: "sample-1", Issuance: issuance,
			Template: template}
		for _, r := range requirements {
			p.Requirements = append(p.Requirements, key(r))
		}
		return traced(releasetrace.Change{Kind: releasetrace.ChangePath, Path: p}, first, second, func() { s.ev.paths = append(s.ev.paths, id) })
	}
	steps = append(steps, build(evidenceBuild, "sample:commit:2"),
		traced(releasetrace.Change{Kind: releasetrace.ChangeHarness, Harness: releasetrace.Harness{ID: evidenceTraceHarness, Identity: evidenceTraceRunner}}, reviewer1,
			reviewer2, func() {
				s.ev.harnesses = append(s.ev.harnesses, evidenceHarnessRecord{module: moduleTrace, id: evidenceTraceHarness})
			}),
		path("sample-path-api", releasetrace.IssuanceAPI, "", "sample-approval", "sample-common-xlsx", "sample-recovery"),
		path("sample-path-xlsx", releasetrace.IssuanceXLSX, "sample-template-xlsx", "sample-common-xlsx", "sample-erp-template"),
		reconcile("sample-path-api", 0), reconcile("sample-path-xlsx", 3),
		result("t-xlsx-amount", evidenceBuild, true), result("t-xlsx-isolation", evidenceBuild, true), result("t-approval", evidenceBuild, false),
		result("t-approval", evidenceBuild, true), result("t-duplicate", evidenceBuild, true), result("t-recovery", evidenceBuild, true),
		result("t-bulk", evidenceBuild, false), result("t-xlsx-amount", evidenceOldBuild, true),
		func() error {
			_, err := s.tracePropose(releasetrace.Change{Kind: releasetrace.ChangeRetire, Retire: releasetrace.Retire{Key: key("sample-legacy-report"),
				Reason: evidenceRetireReason}}, first, installSeedReason)
			return err
		},
		func() error {
			s.ev.candidate = &releasetrace.Release{Version: evidenceCandidateID, Build: evidenceBuild, Requirements: []releasetrace.Key{key("sample-api-export"),
				key("sample-approval"), key("sample-common-xlsx"), key("sample-recovery")}, Paths: []string{"sample-path-api"}}
			return nil
		},
		measured(perfgate.Change{Kind: perfgate.ChangeHardware, Hardware: perfgate.Hardware{ID: evidenceHardware, CPUs: 8, MemoryMiB: 32768, DiskGiB: 500,
			Storage: perfgate.StorageSSD, Spec: sampleHash("sample:hardware:" + evidenceHardware)}}, perfFirst, perfSecond, func() {
			s.ev.hardware = append(s.ev.hardware, evidenceHardware)
		}),
	)
	for _, p := range []struct {
		id   string
		gate string
		axes perfgate.Axes
	}{
		{"sample-e2e", perfgate.GateE2E, perfgate.Axes{RowsRun: 100000}},
		{"sample-daily", perfgate.GateDailyInflow, perfgate.Axes{RowsDay: 1200000}},
		{"sample-month", perfgate.GateMonthlyClose, perfgate.Axes{RowsMonth: 30000000}},
		{"sample-resume", perfgate.GateInterruptResume, perfgate.Axes{RowsRun: 100000, FailureInjections: 1}},
	} {
		profile := perfgate.Profile{ID: p.id, Gate: p.gate, Axes: p.axes, Dataset: sampleHash("sample:dataset:" + p.id), Expected: sampleHash("sample:expected:" + p.id)}
		steps = append(steps, measured(perfgate.Change{Kind: perfgate.ChangeProfile, Profile: profile}, perfFirst, perfSecond, func() {
			s.ev.profiles = append(s.ev.profiles, p.id)
		}))
	}
	budget := func(id, profile string, completion, throughput int64) func() error {
		return measured(perfgate.Change{Kind: perfgate.ChangeBudget, Budget: perfgate.Budget{ID: id, Profile: profile, Hardware: evidenceHardware,
			Limits: evidenceLimits(completion, throughput)}}, perfFirst, perfSecond, func() { s.ev.budgets = append(s.ev.budgets, id) })
	}
	steps = append(steps, budget("sample-budget-e2e", "sample-e2e", 600000, 200), budget("sample-budget-daily", "sample-daily", 86400000, 20),
		measured(perfgate.Change{Kind: perfgate.ChangeHarness, Harness: perfgate.Harness{ID: evidencePerfHarness, Identity: evidencePerfRunner}}, perfReviewer1,
			perfReviewer2, func() {
				s.ev.harnesses = append(s.ev.harnesses, evidenceHarnessRecord{module: modulePerf, id: evidencePerfHarness})
			}))
	for _, r := range []evidenceRunSample{
		{"sample-e2e", 100000, 420, evidenceMetrics(420000, 2147483648, 300, 800, 240)},
		{"sample-daily", 1200000, 86400, evidenceMetrics(86400000, 3221225472, 400, 1500, 13)},
		{"sample-month", 1000000, 3600, evidenceMetrics(3600000, 3221225472, 400, 900, 278)},
	} {
		x := perfgate.Run{Profile: r.profile, Hardware: evidenceHardware, Build: evidenceBuild, Dataset: sampleHash("sample:dataset:" + r.profile),
			Output: sampleHash("sample:expected:" + r.profile), Rows: r.rows, DurationS: r.duration, Metrics: r.metrics}
		steps = append(steps, func() error {
			return s.perfCall("record", func() error { _, err := s.perf.Record(x, perfRunner); return err })
		})
	}
	steps = append(steps, func() error {
		_, err := s.perfPropose(perfgate.Change{Kind: perfgate.ChangeRetire, Retire: perfgate.Retire{Profile: "sample-month"}}, perfSecond, installSeedReason)
		return err
	}, func() error {
		s.ev.seeded = true
		return nil
	})
	return run(steps...)
}
