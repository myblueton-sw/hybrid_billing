package server

import (
	"errors"
	"fmt"
	"go/ast"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/perfgate"
	"github.com/myblueton-sw/hybrid_billing/modules/releasetrace"
)

func TestEvidenceReadFaults(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	counts := map[string]int{}
	f.s.fault = func(step string) bool {
		counts[step]++
		return false
	}
	if got := f.evidenceRead(root); got.status != 200 {
		t.Fatal(got.raw)
	}
	if fmt.Sprint(counts) != "map[perf-active_budget:4 perf-hardware:1 perf-profile:5 perf-readiness:1 perf-runs:4 release-evidence-read:1 roster:1 trace-check:5 "+
		"trace-gate:3 trace-path:2 trace-replacements:6 trace-requirement:24 trace-results:15]" {
		t.Errorf("read calls %v", counts)
	}
	for step, total := range counts {
		for n := 1; n <= total; n++ {
			seen := 0
			f.s.fault = func(name string) bool {
				if name == step {
					seen++
				}
				return name == step && seen == n
			}
			f.code(t, fmt.Sprint(step, " ", n), f.evidenceRead(root), 503, "dependency_unavailable")
		}
	}
	f.s.fault = nil
	if got := f.evidenceRead(root); got.status != 200 {
		t.Errorf("read after faults %s", got.raw)
	}
	for _, c := range []func(){
		func() { f.s.ev.paths = append(f.s.ev.paths, "sample-absent") },
		func() { f.s.ev.requirements = append(f.s.ev.requirements, evKey("sample-absent")) },
		func() {
			f.s.ev.results = append(f.s.ev.results, evidenceResultKey{namespace: "sample", test: "t-absent", build: evidenceBuild})
		},
		func() { f.s.ev.hardware = append(f.s.ev.hardware, "sample-absent") },
		func() { f.s.ev.profiles = append(f.s.ev.profiles, "sample-absent") },
	} {
		paths, requirements, results, hardware, profiles := f.s.ev.paths, f.s.ev.requirements, f.s.ev.results, f.s.ev.hardware, f.s.ev.profiles
		c()
		f.code(t, "unknown stored record", f.evidenceRead(root), 503, "dependency_unavailable")
		f.s.ev.paths, f.s.ev.requirements, f.s.ev.results, f.s.ev.hardware, f.s.ev.profiles = paths, requirements, results, hardware, profiles
	}
	f.s.ev.candidate = &releasetrace.Release{Version: "bad version", Build: evidenceBuild}
	f.code(t, "malformed candidate", f.evidenceRead(root), 503, "dependency_unavailable")
}

func TestEvidenceCommandFaults(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	at := ""
	f.s.fault = func(step string) bool { return step == at }
	before, entries := f.evModule(), len(f.s.ev.entries)
	for _, c := range []struct {
		step, kind, target string
		who                actor
		fields             map[string]string
	}{
		{"trace-approve", evidenceApproveKind, traceTarget(19), second, nil},
		{"trace-withdraw", evidenceWithdrawKind, traceTarget(19), root, nil},
		{"perf-approve", evidenceApproveKind, perfTarget(9), root, nil},
		{"perf-withdraw", evidenceWithdrawKind, perfTarget(9), second, nil},
		{"trace-requirement", evidenceRetireReqKind, traceRequirementTarget + "5", root, map[string]string{"reason": "x"}},
		{"trace-propose", evidenceRetireReqKind, traceRequirementTarget + "5", root, map[string]string{"reason": "x"}},
		{"perf-profile", evidenceRetireProfKind, perfProfileTarget + "sample-e2e", root, nil},
		{"perf-propose", evidenceRetireProfKind, perfProfileTarget + "sample-e2e", root, nil},
	} {
		if c.fields == nil {
			c.fields = map[string]string{}
		}
		at = ""
		revision := f.revision(c.who)
		at = c.step
		prior := f.s.Revision()
		f.code(t, c.step+" "+c.kind, f.runWith(c.who, revision, evidenceCommand+c.kind, c.target, c.fields), 500, "internal_error")
		if f.s.Revision() == prior {
			t.Errorf("%s did not move the revision", c.step)
		}
	}
	at = ""
	if f.evModule() != before || len(f.s.ev.entries) != entries || f.s.ev.entry(moduleTrace, 19).state != entryPending || f.s.ev.entry(modulePerf, 9).state != entryPending {
		t.Error("faults changed the modules or the store")
	}
	f.ok(t, "after faults", f.run(root, "release_evidence.retire_profile", perfProfileTarget+"sample-e2e", nil), "pending", "")
}

func evidenceSteps(t *testing.T) map[string]int {
	t.Helper()
	s := seedOnly(t)
	counts := map[string]int{}
	s.fault = func(name string) bool {
		counts[name]++
		return false
	}
	if err := s.seedEvidence(s.clock()); err != nil {
		t.Fatal(err)
	}
	return counts
}

func TestEvidenceSeed(t *testing.T) {
	s := seedOnly(t)
	if err := s.seedEvidence(s.clock()); err != nil {
		t.Fatal(err)
	}
	c := s.ev
	if s := fmt.Sprint(c.seeded, " ", c.requirements, " ", c.builds, " ", c.paths, " ", c.profiles, " ", c.hardware, " ", c.budgets, " ", c.harnesses, " ", len(c.results), " ",
		*c.candidate, " ", len(c.entries), " ", c.applied); s != "true [{sample sample-common-xlsx} {sample sample-approval} {sample sample-recovery} "+
		"{sample sample-erp-template} {sample sample-api-export} {sample sample-legacy-report}] [{0.0.1 "+sampleHash("sample:commit:0")[:40]+"} {0.0.2 "+
		sampleHash("sample:commit:2")[:40]+"}] [sample-path-api sample-path-xlsx] [sample-e2e sample-daily sample-month sample-resume] [sample-hw-small] "+
		"[sample-budget-e2e sample-budget-daily] [{release_trace sample-trace-harness} {perf_gate sample-perf-harness}] 7 "+
		"{sample-candidate-1 0.0.2 [{sample sample-api-export} {sample sample-approval} {sample sample-common-xlsx} {sample sample-recovery}] [sample-path-api]} 28 18" {
		t.Errorf("seed store %s", s)
	}
	for _, e := range c.entries {
		want := releasetrace.ProposalApplied
		if e.number == 19 && e.module == moduleTrace || e.number == 9 && e.module == modulePerf {
			want = entryPending
		}
		if e.state != want || e.rationale != installSeedReason {
			t.Errorf("entry %+v", e)
		}
	}
	if err := s.seedEvidence(s.clock()); err == nil {
		t.Error("second seed")
	}
	counts := evidenceSteps(t)
	if fmt.Sprint(counts) != "map[perf-approve:8 perf-propose:9 perf-record:3 trace-approve:18 trace-propose:19 trace-record:8]" {
		t.Errorf("seed calls %v", counts)
	}
	for step, total := range counts {
		for n := 1; n <= total; n++ {
			s := seedOnly(t)
			seen := 0
			s.fault = func(name string) bool {
				if name == step {
					seen++
				}
				return name == step && seen == n
			}
			err := s.seedEvidence(s.clock())
			if !errors.Is(err, releasetrace.ErrInvalid) && !errors.Is(err, perfgate.ErrInvalid) || s.ev.seeded {
				t.Errorf("%s %d: %v", step, n, err)
			}
		}
	}
	broken := freshInstaller(t)
	broken.fault = func(name string) bool { return name == "perf-record" }
	if err := broken.Start(); !errors.Is(err, perfgate.ErrInvalid) {
		t.Errorf("start with a failing seed %v", err)
	}
	january, err := New(Config{Data: t.TempDir() + "/d", Seed: SeedSample, Clock: func() time.Time { return time.Date(2027, 1, 15, 9, 0, 0, 0, time.UTC) }})
	if err != nil {
		t.Fatal(err)
	}
	if err := january.seedEvidence(january.clock()); err != nil {
		t.Fatal(err)
	}
	if v, err := january.trace.Path("sample-path-api"); err != nil || v.Reconciliations[0].Period != "2026-12" {
		t.Errorf("period across a year %+v %v", v, err)
	}
}

func TestEvidenceStoreAndTargets(t *testing.T) {
	c := newEvidenceStore()
	c.keep(&evidenceEntry{module: moduleTrace, number: 1, state: entryPending})
	for n := uint64(2); n <= installKept+1; n++ {
		c.keep(&evidenceEntry{module: modulePerf, number: n, state: releasetrace.ProposalApplied})
	}
	if len(c.entries) != installKept || c.entry(moduleTrace, 1) == nil || c.entry(modulePerf, 2) != nil || c.entry(modulePerf, installKept+1) == nil ||
		c.entry(moduleTrace, installKept+1) != nil {
		t.Errorf("kept %d", len(c.entries))
	}
	for _, e := range c.entries {
		e.state = entryPending
	}
	c.keep(&evidenceEntry{module: moduleTrace, number: 999, state: entryPending})
	if len(c.entries) != installKept+1 {
		t.Errorf("pending proposals dropped %d", len(c.entries))
	}
	if entryPending != perfgate.ProposalPending || releasetrace.ChangeRetire != perfgate.ChangeRetire || releasetrace.ProposalWithdrawn != perfgate.ProposalWithdrawn ||
		releasetrace.ReasonNotPending != perfgate.ReasonNotPending || releasetrace.ReasonRetired != perfgate.ReasonRetired {
		t.Error("the modules disagree on shared names")
	}
	f := setup(t, SeedSample).ready()
	for target, want := range map[string]string{traceTarget(1): "release_trace 1", traceTarget(19): "release_trace 19", perfTarget(9): "perf_gate 9"} {
		if e := f.s.evidenceEntryOf(target, SampleWorkspace); e == nil || fmt.Sprint(e.module, " ", e.number) != want {
			t.Errorf("%s: %v", target, e)
		}
	}
	for _, target := range []string{"", traceProposalTarget, traceTarget(0), traceTarget(29), perfTarget(10), "trace-proposal-07", "perf-proposal-9x", "x-trace-proposal-1"} {
		if f.s.evidenceEntryOf(target, SampleWorkspace) != nil {
			t.Errorf("%s resolved", target)
		}
	}
	if f.s.evidenceEntryOf(traceTarget(1), "other") != nil || f.s.evidenceProfileOf(perfProfileTarget+"sample-e2e", "other") != "" {
		t.Error("other workspace targets")
	}
	if _, ok := f.s.evidenceRequirementOf(traceRequirementTarget+"1", "other"); ok {
		t.Error("other workspace requirement")
	}
	if k, ok := f.s.evidenceRequirementOf(traceRequirementTarget+"6", SampleWorkspace); !ok || k != evKey("sample-legacy-report") {
		t.Errorf("requirement target %v", k)
	}
	if f.s.ev.requirementIndex(evKey("sample-absent")) != 0 || f.s.ev.hasProfile("sample-absent") || f.s.ev.pendingReplacement(evKey("sample-approval")) {
		t.Error("store lookups")
	}
	f.s.ev.addResult(f.s.ev.results[0])
	if len(f.s.ev.results) != 7 {
		t.Error("duplicate result key stored")
	}
	for n := uint64(100); n < 160; n++ {
		f.s.ev.keep(&evidenceEntry{module: moduleTrace, number: n, kind: releasetrace.ChangeBuild, state: releasetrace.ProposalWithdrawn, proposer: SampleAdmin,
			trace: releasetrace.Change{Kind: releasetrace.ChangeBuild, Build: releasetrace.Build{Build: "9.9.9"}}})
	}
	got := evMap(f.evidenceRead(f.as(AdminLogin)).body, "decisions", "proposals")
	rows := evRows(got)
	waiting, oldest := 0, ""
	for _, row := range rows {
		if row["state"] == entryPending {
			waiting++
		} else {
			oldest = fmt.Sprint(row["number"])
		}
	}
	if len(rows) != evidenceShown || fmtInt(got["total"]) != 88 || got["more"] != true || fmt.Sprint(rows[0]["number"]) != "159" || waiting == 0 || oldest != fmt.Sprint(110+waiting) {
		t.Errorf("page %d %v waiting %d oldest closed %s", len(rows), got["total"], waiting, oldest)
	}
}

func TestEvidenceSummaries(t *testing.T) {
	proof := releasetrace.Change{Kind: "proof", Proof: releasetrace.Proof{Key: evKey("r"), Build: "1.0.0", Environment: "env-secret", Customer: "customer-secret",
		Period: "2026-10", Evidence: sampleHash("p")}}
	activate := releasetrace.Change{Kind: "activate", Activate: releasetrace.Activate{Path: "p", Build: "1.0.0", Proof: sampleHash("p")}}
	retire := releasetrace.Change{Kind: releasetrace.ChangeRetire, Retire: releasetrace.Retire{Key: evKey("a"), Replacement: evKey("b"), Reason: "r"}}
	approval := perfgate.Change{Kind: "approval", Approval: perfgate.Approval{Budget: "b", Distribution: sampleHash("d")}}
	if s := evJSON(traceSummary(proof)) + evJSON(traceSummary(activate)) + evJSON(traceSummary(retire)) + evJSON(perfSummary(approval)); s !=
		`{"build":"1.0.0","label":"sample/r","period":"2026-10"}{"build":"1.0.0","path":"p"}{"label":"sample/a","reason":"r","replacement":"sample/b"}{"budget":"b"}` {
		t.Errorf("summaries %s", s)
	}
	if s := evJSON(traceSummary(releasetrace.Change{Kind: "unknown"})) + evJSON(perfSummary(perfgate.Change{Kind: "unknown"})); s != "{}{}" {
		t.Errorf("unknown kinds %s", s)
	}
	f := setup(t, SeedSample).ready()
	f.s.ev.keep(&evidenceEntry{module: moduleTrace, number: 90, kind: proof.Kind, state: entryPending, proposer: "test-ann", trace: proof, created: start})
	f.s.ev.keep(&evidenceEntry{module: modulePerf, number: 90, kind: approval.Kind, state: entryPending, proposer: "test-ann", perf: approval, created: start})
	got := f.evidenceRead(f.as(AdminLogin))
	if strings.Contains(got.raw, "customer-secret") || strings.Contains(got.raw, "env-secret") || strings.Contains(got.raw, `"customer"`) {
		t.Errorf("proof fields leak %s", got.raw)
	}
	if s := evBlocked(evProposal(t, got, traceTarget(90))) + "|" + evBlocked(evProposal(t, got, perfTarget(90))); s !=
		"false not_console_kind false not_proposer|false not_console_kind false not_proposer" {
		t.Errorf("injected blockers %s", s)
	}
}

func TestMapEvidenceErrors(t *testing.T) {
	trace := map[error]commands.Code{
		releasetrace.Refusal{Reason: releasetrace.ReasonSelfApproval}:           commands.PrerequisiteFailed,
		releasetrace.Refusal{Reason: releasetrace.ReasonNotProposer}:            commands.PrerequisiteFailed,
		releasetrace.Refusal{Reason: releasetrace.ReasonAINotAllowed}:           commands.InternalError,
		releasetrace.Refusal{Reason: releasetrace.ReasonNotHuman}:               commands.InternalError,
		releasetrace.Refusal{Reason: releasetrace.ReasonNotPending}:             commands.InvalidStage,
		releasetrace.Refusal{Reason: releasetrace.ReasonRetired}:                commands.InvalidStage,
		releasetrace.Refusal{Reason: releasetrace.ReasonCommonNeedsReplacement}: commands.InvalidStage,
		releasetrace.Refusal{Reason: releasetrace.ReasonWeakerReplacement}:      commands.InvalidStage,
		releasetrace.ErrNotFound: commands.InvalidStage,
		releasetrace.ErrConflict: commands.InvalidStage,
		releasetrace.ErrLimit:    commands.PrerequisiteFailed,
		releasetrace.ErrInvalid:  commands.InternalError,
		releasetrace.ErrRejected: commands.InternalError,
		errors.New("unrelated"):  commands.InternalError,
	}
	perf := map[error]commands.Code{
		perfgate.Refusal{Reason: perfgate.ReasonSelfApproval}: commands.PrerequisiteFailed,
		perfgate.Refusal{Reason: perfgate.ReasonNotProposer}:  commands.PrerequisiteFailed,
		perfgate.Refusal{Reason: perfgate.ReasonAINotAllowed}: commands.InternalError,
		perfgate.Refusal{Reason: perfgate.ReasonNotHuman}:     commands.InternalError,
		perfgate.Refusal{Reason: perfgate.ReasonNotPending}:   commands.InvalidStage,
		perfgate.Refusal{Reason: perfgate.ReasonRetired}:      commands.InvalidStage,
		perfgate.ErrNotFound:    commands.InvalidStage,
		perfgate.ErrConflict:    commands.InvalidStage,
		perfgate.ErrLimit:       commands.PrerequisiteFailed,
		perfgate.ErrInvalid:     commands.InternalError,
		perfgate.ErrRejected:    commands.InternalError,
		errors.New("unrelated"): commands.InternalError,
	}
	for name, c := range map[string]struct {
		table map[error]commands.Code
		of    func(error) error
	}{"trace": {trace, mapTrace}, "perf": {perf, mapPerf}} {
		for err, want := range c.table {
			var failure *commands.Failure
			if !errors.As(c.of(err), &failure) || failure.Code != want || len(failure.Fields) != 0 {
				t.Errorf("%s %v: %v", name, err, failure)
			}
		}
		if c.of(nil) != nil {
			t.Errorf("%s nil mapped", name)
		}
	}
	for err, want := range map[error]string{nil: "", releasetrace.Refusal{Reason: "x"}: "x", releasetrace.ErrNotFound: "not_found"} {
		if got, failed := traceBlocker(err); got != want || failed != nil {
			t.Errorf("%v: %q %v", err, got, failed)
		}
	}
	for _, err := range []error{releasetrace.ErrInvalid, releasetrace.ErrLimit, releasetrace.ErrConflict, errors.New("other")} {
		if got, failed := traceBlocker(err); got != "" || failed != err {
			t.Errorf("%v became the blocker %q", err, got)
		}
	}
	if !traceGone(releasetrace.ErrNotFound) || !traceGone(releasetrace.Refusal{Reason: releasetrace.ReasonNotPending}) || traceGone(releasetrace.ErrInvalid) ||
		!perfGone(perfgate.ErrNotFound) || !perfGone(perfgate.Refusal{Reason: perfgate.ReasonNotPending}) || perfGone(perfgate.ErrInvalid) {
		t.Error("gone")
	}
}

func TestEvidenceCommandSet(t *testing.T) {
	f := setup(t, SeedSample).ready()
	ids := []string{}
	for id, mutating := range f.s.mutating {
		if strings.HasPrefix(id, evidenceCommand) {
			if !mutating {
				t.Errorf("%s is not mutating", id)
			}
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	if strings.Join(ids, " ") != "release_evidence.approve release_evidence.retire_profile release_evidence.retire_requirement release_evidence.withdraw" {
		t.Errorf("commands %v", ids)
	}
}

func TestEvidenceActorsAndEffectsAreConfined(t *testing.T) {
	modules := map[string]bool{"releasetrace": true, "perfgate": true}
	never := map[string]bool{"RevokeHarness": true, "Deactivate": true}
	forbidden := map[string]bool{"releasetrace.ChangeActivate": true, "releasetrace.ChangeProof": true, "perfgate.ChangeApproval": true}
	seedOnlyNames := map[string]bool{"evidenceReviewer1": true, "evidenceReviewer2": true, "evidenceTraceRunner": true, "evidencePerfRunner": true}
	found := map[string]int{}
	for _, src := range serverSources(t) {
		readOrCommand := src.name == "evidence.go" || src.name == "evidenceload.go" || src.name == "evidencecmds.go"
		for _, decl := range src.file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			seed := fn.Name.Name == "seedEvidence"
			ast.Inspect(fn, func(n ast.Node) bool {
				if ident, ok := n.(*ast.Ident); ok && seedOnlyNames[ident.Name] {
					found["seed names"]++
					if !seed {
						t.Errorf("%s %s uses %s", src.name, fn.Name.Name, ident.Name)
					}
				}
				for pkg := range modules {
					if name, ok := packageSelector(n, pkg); ok {
						if name == "KindSystem" || name == "KindAssistant" {
							found[pkg+"."+name]++
							if !seed {
								t.Errorf("%s %s names %s.%s", src.name, fn.Name.Name, pkg, name)
							}
						}
						if forbidden[pkg+"."+name] {
							t.Errorf("%s %s names %s.%s", src.name, fn.Name.Name, pkg, name)
						}
					}
				}
				if sel, ok := n.(*ast.SelectorExpr); ok && (selectorOn(sel.X, "trace") || selectorOn(sel.X, "perf")) {
					if sel.Sel.Name == "Record" {
						found["Record"]++
						if !seed {
							t.Errorf("%s %s selects Record", src.name, fn.Name.Name)
						}
					}
					if never[sel.Sel.Name] {
						t.Errorf("%s %s selects %s", src.name, fn.Name.Name, sel.Sel.Name)
					}
				}
				lit, ok := n.(*ast.CompositeLit)
				if !ok {
					return true
				}
				for pkg := range modules {
					kind, inPackage := packageSelector(lit.Type, pkg)
					if !inPackage {
						continue
					}
					for _, elt := range lit.Elts {
						kv, ok := elt.(*ast.KeyValueExpr)
						if !ok {
							continue
						}
						key := fmt.Sprint(kv.Key)
						value, _ := packageSelector(kv.Value, pkg)
						if kind == "Change" && key == "Kind" && value != "ChangeRetire" && !seed {
							t.Errorf("%s %s builds a %s change", src.name, fn.Name.Name, value)
						}
						if kind == "Actor" && src.name == "evidencecmds.go" {
							if key == "ID" && fmt.Sprint(kv.Value) != "&{ctx User}" || key == "Kind" && value != "KindHuman" {
								t.Errorf("%s %s builds an actor with %s %v", src.name, fn.Name.Name, key, kv.Value)
							}
							found["command actor fields"]++
						}
					}
				}
				return true
			})
		}
		if readOrCommand {
			ast.Inspect(src.file, func(n ast.Node) bool {
				if lit, ok := n.(*ast.BasicLit); ok && (strings.Contains(lit.Value, "system") || strings.Contains(lit.Value, "assistant")) {
					t.Errorf("%s has the literal %s", src.name, lit.Value)
				}
				return true
			})
		}
	}
	if fmt.Sprint(found) != "map[Record:2 command actor fields:12 perfgate.KindSystem:1 releasetrace.KindSystem:1 seed names:8]" {
		t.Errorf("found %v", found)
	}
}

func TestEvidenceLeaksNothing(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	words := []string{evidenceTraceRunner, evidencePerfRunner, "token", "secret", `"key"`, `"customer"`, "identity"}
	raws := map[string]string{"read": f.evidenceRead(root).raw, "second read": f.evidenceRead(second).raw}
	for _, c := range []struct {
		who          actor
		kind, target string
		fields       map[string]string
	}{
		{second, evidenceApproveKind, traceTarget(19), nil},
		{root, evidenceApproveKind, perfTarget(9), nil},
		{root, evidenceRetireReqKind, traceRequirementTarget + "5", map[string]string{"reason": "x"}},
		{root, evidenceWithdrawKind, traceTarget(20), nil},
		{second, evidenceRetireProfKind, perfProfileTarget + "sample-e2e", nil},
	} {
		got := f.run(c.who, evidenceCommand+c.kind, c.target, c.fields)
		if got.status != 200 {
			t.Fatalf("%s: %s", c.kind, got.raw)
		}
		raws[c.kind+" "+c.target] = got.raw
	}
	raws["final read"] = f.evidenceRead(root).raw
	for name, raw := range raws {
		for _, word := range words {
			if strings.Contains(raw, word) {
				t.Errorf("%s carries %q", name, word)
			}
		}
	}
	for _, reviewer := range []string{evidenceReviewer1, evidenceReviewer2} {
		if !strings.Contains(raws["read"], `{"id":"`+reviewer+`","login":null,"directory":false}`) || strings.Count(raws["read"], reviewer) != strings.Count(raws["read"],
			`{"id":"`+reviewer+`","login":null,"directory":false}`) {
			t.Errorf("reviewer %s shown otherwise", reviewer)
		}
	}
}

func TestEvidenceConcurrentReadsCommandsAndChecks(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				if got := f.evidenceRead(second); got.status != 200 {
					t.Errorf("concurrent read %d", got.status)
				}
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				f.s.trace.Check(retireCheck(evKey("sample-approval"), releasetrace.Key{}), SampleAdmin2, "")
				f.s.perf.Readiness(evidenceBuild)
			}
		}()
	}
	for i := 0; i < 3; i++ {
		for _, c := range []struct{ command, target, fields string }{{"retire_requirement", traceRequirementTarget + fmt.Sprint(i+2), `"reason":"x"`},
			{"retire_profile", perfProfileTarget + "sample-e2e", ""}, {"withdraw", traceTarget(19), ""}} {
			body := fmt.Sprintf(`{"request_id":"c%d%s","target_id":"%s","idempotency_key":"c%d%s","expected_revision":"%s","rationale":"x","body":{%s}}`, i, c.command, c.target, i,
				c.command, f.s.Revision(), c.fields)
			f.call(evidenceCommand+c.command, body, root.cookie, root.csrf)
		}
	}
	wg.Wait()
	f.neverReleased(t, f.evidenceRead(root))
}

func TestEvidenceRowsAreBounded(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	tests := []releasetrace.TestRef{}
	for n := 0; n <= evidenceShown; n++ {
		tests = append(tests, releasetrace.TestRef{ID: fmt.Sprintf("bulk-%02d", n)})
	}
	f.evAddRequirement("test-many-tests", false, false, tests...)
	keys := []releasetrace.Key{}
	for n := 0; n <= evidenceShown; n++ {
		id := fmt.Sprintf("test-bulk-%02d", n)
		f.evAddRequirement(id, false, false, tests[:1+n%2]...)
		keys = append(keys, evKey(id))
	}
	for n := 0; n < evidenceStatesShown; n++ {
		build := fmt.Sprint("0.1.", n)
		f.evApproved(releasetrace.Change{Kind: releasetrace.ChangeBuild, Build: releasetrace.Build{Build: build, Commit: sampleHash(build)[:40]}})
		f.s.ev.builds = append(f.s.ev.builds, evidenceBuildRecord{id: build, commit: sampleHash(build)[:40]})
	}
	for n := 0; n <= evidenceShown; n++ {
		f.s.ev.builds = append(f.s.ev.builds, evidenceBuildRecord{id: fmt.Sprint("0.2.", n)})
		f.s.ev.harnesses = append(f.s.ev.harnesses, evidenceHarnessRecord{module: moduleTrace, id: fmt.Sprint("h-", n)})
	}
	f.evHarness()
	for n := 0; n <= evidenceShown; n++ {
		f.evRecord(tests[0].ID, fmt.Sprint("0.1.", n%evidenceStatesShown), n%2 == 0)
		f.evRecord(tests[n].ID, evidenceBuild, true)
	}
	f.evApproved(releasetrace.Change{Kind: releasetrace.ChangePath, Path: releasetrace.Path{ID: "test-path-wide", Provider: "p", Contract: "c", Role: "r", Version: "v",
		Issuance: releasetrace.IssuanceFile, Requirements: keys}})
	f.evApproved(releasetrace.Change{Kind: releasetrace.ChangeRequirement, Requirement: releasetrace.Requirement{Namespace: "other", ID: "unstored", Source: "s", Owner: "o",
		Tests: tests[:1], Design: sampleHash("u")}})
	f.evApproved(releasetrace.Change{Kind: releasetrace.ChangePath, Path: releasetrace.Path{ID: "test-path-quiet", Provider: "p", Contract: "c", Role: "r", Version: "v",
		Issuance: releasetrace.IssuanceFile, Requirements: []releasetrace.Key{{Namespace: "other", ID: "unstored"}}}})
	for n := 0; n <= evidencePeriods; n++ {
		month := 2025*12 + 9 + n
		f.evApproved(releasetrace.Change{Kind: releasetrace.ChangeReconcile, Reconcile: releasetrace.Reconcile{Path: "test-path-wide",
			Period: fmt.Sprintf("%d-%02d", month/12, month%12+1), Evidence: sampleHash(fmt.Sprint(n)), Matched: true}})
	}
	f.s.ev.paths = append(f.s.ev.paths, "test-path-wide", "test-path-quiet")
	for n := 0; n <= evidenceShown; n++ {
		f.s.ev.paths = append(f.s.ev.paths, "test-path-quiet")
	}
	for n := 0; n <= evidenceShown; n++ {
		id := fmt.Sprintf("hw-%02d", n)
		f.pfApproved(perfgate.Change{Kind: perfgate.ChangeHardware, Hardware: perfgate.Hardware{ID: id, CPUs: 1, MemoryMiB: 1, DiskGiB: 1, Storage: perfgate.StorageHDD,
			Spec: sampleHash(id)}})
		f.s.ev.hardware = append(f.s.ev.hardware, id)
		f.s.ev.profiles = append(f.s.ev.profiles, "sample-resume")
	}
	f.perfHarness()
	for n := 0; n <= evidenceRunsShown; n++ {
		if _, err := f.s.perf.Record(perfgate.Run{Profile: "sample-resume", Hardware: evidenceHardware, Build: evidenceBuild, Dataset: sampleHash("sample:dataset:sample-resume"),
			Output: sampleHash("x"), Rows: int64(n)}, pfRunner); err != nil {
			t.Fatal(err)
		}
	}
	candidate := *f.s.ev.candidate
	for n := 0; n <= evidenceProblems; n++ {
		candidate.Requirements = append(candidate.Requirements, releasetrace.Key{Namespace: "zz", ID: fmt.Sprintf("u%03d", n)})
	}
	f.s.ev.candidate = &candidate
	got := f.evidenceRead(root)
	if got.status != 200 {
		t.Fatal(got.raw)
	}
	size := func(list map[string]any) string {
		return fmt.Sprint(len(evRows(list)), "/", list["total"], "/", list["more"])
	}
	wide, quiet := evPath(t, got, "test-path-wide"), evPath(t, got, "test-path-quiet")
	resume := evProfile(t, got, "sample-resume")
	many := evReq(t, got, "test-many-tests")
	sizes := []string{size(evMap(got.body, "matrix", "paths")), size(evMap(wide, "requirements")), size(evMap(wide, "reconciliations")),
		size(evMap(got.body, "requirements", "requirements")), size(evMap(many, "tests")), size(evMap(evReq(t, got, "sample-approval"), "replacements")),
		fmt.Sprint(len(evMap(many, "states")), many["builds_more"]), size(evMap(got.body, "requirements", "gate", "problems")), size(evMap(got.body, "artifacts", "builds")),
		size(evMap(got.body, "artifacts", "harnesses")), size(evMap(got.body, "artifacts", "results")), size(evMap(got.body, "load", "hardware")),
		size(evMap(got.body, "load", "profiles")), size(evMap(resume, "runs"))}
	if s := strings.Join(sizes, " "); s != "50/55/true 50/51/true 12/13/true 50/58/true 50/51/true 20/56/true 10 true 100/106/true 50/63/true 53/53/false 50/68/true "+
		"50/52/true 50/55/true 20/21/true" {
		t.Errorf("sizes %s", s)
	}
	if s := fmt.Sprint(evRows(evMap(wide, "reconciliations"))[0]["period"], " ", quiet["last_observed"], " ", evRows(evMap(quiet, "requirements"))[0]["target"], " ",
		evRows(evMap(resume, "runs"))[0]["rows"]); s != "2026-10 <nil> <nil> 20" {
		t.Errorf("rows %s", s)
	}
}

func TestEvidenceInjectedModuleFacts(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	p, err := f.s.trace.Propose(releasetrace.Change{Kind: "proof", Proof: releasetrace.Proof{Key: evKey("sample-common-xlsx"), Build: evidenceBuild, Environment: "env-x",
		Customer: "cust-x", Period: "2026-10", Evidence: sampleHash("proof")}}, evAnn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.trace.Approve(p.Number, evBob); err != nil {
		t.Fatal(err)
	}
	a, err := f.s.perf.Propose(perfgate.Change{Kind: "approval", Approval: perfgate.Approval{Budget: "sample-budget-e2e", Distribution: sampleHash("dist")}}, pfAnn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.perf.Approve(a.Number, pfBob); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.trace.Propose(releasetrace.Change{Kind: releasetrace.ChangeBuild, Build: releasetrace.Build{Build: "0.0.3", Commit: sampleHash("b3")[:40]}}, evAnn); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.trace.Approve(p.Number+1, evBob); err != nil {
		t.Fatal(err)
	}
	f.evHarness()
	f.evRecord("t-api-export", "0.0.3", true)
	if err := f.s.trace.RevokeHarness("test-harness", evAnn); err != nil {
		t.Fatal(err)
	}
	got := f.evidenceRead(root)
	api := evPath(t, got, "sample-path-api")
	if s := evJSON(evMap(api, "columns")["customer_proof"]) + " " + fmt.Sprint(evMap(evProfile(t, got, "sample-e2e"), "active_budget")["approved_at"]); s !=
		`{"count":1,"of":3} 2026-11-16T09:00:00Z` {
		t.Errorf("injected proof and approval %s", s)
	}
	if row := evFind(t, evRows(evMap(got.body, "artifacts", "results")), "build", "0.0.3"); row["not_counted_reason"] != notCountedByModule {
		t.Errorf("build outside the store %v", row)
	}
	if strings.Contains(got.raw, "cust-x") || strings.Contains(got.raw, "env-x") {
		t.Error("proof fields leak")
	}
}
