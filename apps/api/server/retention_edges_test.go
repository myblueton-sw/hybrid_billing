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
	"github.com/myblueton-sw/hybrid_billing/modules/retention"
)

func (f world) retModule() string {
	f.t.Helper()
	out := ""
	for n := uint64(1); n <= 20; n++ {
		v, err := f.s.retention.Proposal(n)
		out += fmt.Sprint(v, err)
	}
	for n := 1; n <= 5; n++ {
		v, err := f.s.retention.Hold(fmt.Sprint(retentionHoldPrefix, n))
		out += fmt.Sprint(v, err)
	}
	for _, id := range []string{retentionDispute, retentionTemporary} {
		out += fmt.Sprint(f.retHoldView(id))
	}
	for _, id := range []string{"run/1", "run/2"} {
		out += fmt.Sprint(f.retRunView(id))
	}
	for _, class := range retention.Classes {
		v, err := f.s.retention.Policy(class)
		out += fmt.Sprint(v, err)
	}
	return out + fmt.Sprint(f.s.retention.Watermark())
}

func TestRetentionReadFaults(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	at := ""
	f.s.fault = func(step string) bool { return step == at }
	for _, step := range []string{"retention-read", "roster", "retention-policy", "retention-artifact", "retention-standing", "retention-hold", "retention-hold_outcomes",
		"retention-run", "retention-watermark", "retention-tombstones", "retention-impact"} {
		at = step
		f.code(t, step, f.retentionRead(root), 503, "dependency_unavailable")
	}
	at = "retention-check"
	got := f.retentionRead(root)
	if got.status != 200 || retProposal(t, got, 7)["approve_blocked"] != "invalid" {
		t.Errorf("check fault %s", got.raw)
	}
	at = ""
	if got := f.retentionRead(root); got.status != 200 {
		t.Errorf("read after faults %s", got.raw)
	}
	f.s.ret.artifacts = append(f.s.ret.artifacts, &retentionArtifact{id: "sample-absent", class: retention.ClassClaim, tenants: []string{SampleWorkspace}})
	f.code(t, "unknown stored artifact", f.retentionRead(root), 503, "dependency_unavailable")
	f.s.ret.artifacts = f.s.ret.artifacts[:len(f.s.ret.artifacts)-1]
	f.s.ret.holds = append(f.s.ret.holds, &retentionHold{id: "sample-absent", workspace: SampleWorkspace})
	f.code(t, "unknown stored hold", f.retentionRead(root), 503, "dependency_unavailable")
	f.s.ret.holds = f.s.ret.holds[:len(f.s.ret.holds)-1]
	f.s.ret.runs = append(f.s.ret.runs, &retentionRun{id: "run/9", workspace: SampleWorkspace})
	f.code(t, "unknown stored run", f.retentionRead(root), 503, "dependency_unavailable")
}

func TestRetentionCommandFaults(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	at := ""
	f.s.fault = func(step string) bool { return step == at }
	before, entries, holds := f.retModule(), len(f.s.ret.entries), len(f.s.ret.holds)
	dispute := retentionHoldTarget + retentionDispute
	for _, c := range []struct {
		step, kind, target string
		who                actor
		fields             map[string]string
	}{
		{"retention-approve", retentionApproveKind, retTarget(6), second, nil},
		{"retention-withdraw", retentionWithdrawKind, retTarget(6), root, nil},
		{"retention-policy", retentionPolicyKind, retentionClassTarget + retention.ClassClaim, root, map[string]string{"period": "1", "grace": "0"}},
		{"retention-propose", retentionPolicyKind, retentionClassTarget + retention.ClassClaim, root, map[string]string{"period": "1", "grace": "0"}},
		{"retention-hold", retentionHoldKind, retentionArtifactTarget + "sample-source-5", root, map[string]string{"owner": SampleAdmin, "reason": "x"}},
		{"retention-place_hold", retentionHoldKind, retentionArtifactTarget + "sample-source-5", root, map[string]string{"owner": SampleAdmin, "reason": "x"}},
		{"retention-hold", retentionReleaseKind, dispute, second, map[string]string{"reason": "x"}},
		{"retention-propose", retentionReleaseKind, dispute, second, map[string]string{"reason": "x"}},
		{"retention-run", retentionStopKind, "retention-run-2", root, nil},
		{"retention-stop", retentionStopKind, "retention-run-2", root, nil},
	} {
		if c.fields == nil {
			c.fields = map[string]string{}
		}
		at = ""
		revision := f.revision(c.who)
		at = c.step
		prior := f.s.Revision()
		f.code(t, c.step+" "+c.kind, f.runWith(c.who, revision, retentionCommand+c.kind, c.target, c.fields), 500, "internal_error")
		if f.s.Revision() == prior {
			t.Errorf("%s did not move the revision", c.step)
		}
	}
	at = ""
	if f.retModule() != before || len(f.s.ret.entries) != entries || len(f.s.ret.holds) != holds || f.s.ret.entry(6).view.State != retention.ProposalPending {
		t.Error("faults changed the module or the store")
	}
	f.ok(t, "after faults", f.run(root, "retention.place_hold", retentionArtifactTarget+"sample-source-5", map[string]string{"owner": SampleAdmin, "reason": "x"}), "held",
		"map[hold:retention-hold-3 not_preserved:0 outcome_pending:0 partially_deleted:0 target:retention-hold:retention-hold-3]")
}

func retentionSteps(t *testing.T) map[string]int {
	t.Helper()
	s := seedOnly(t)
	counts := map[string]int{}
	s.fault = func(name string) bool {
		counts[name]++
		return false
	}
	if err := s.seedRetention(s.clock()); err != nil {
		t.Fatal(err)
	}
	return counts
}

func TestRetentionSeed(t *testing.T) {
	s := seedOnly(t)
	if err := s.seedRetention(s.clock()); err != nil {
		t.Fatal(err)
	}
	c := s.ret
	ids := []string{}
	for _, a := range c.artifacts {
		ids = append(ids, a.id+":"+a.class+":"+strings.Join(a.tenants, ","))
	}
	if len(c.artifacts) != 14 || len(c.entries) != 7 || len(c.holds) != 2 || len(c.runs) != 2 || c.plans != 2 || c.holdSeq != 0 || c.runs[0].id != "run/1" ||
		c.runs[1].workspace != SampleWorkspace || c.holds[0].id != retentionDispute || c.holds[1].rationale != installSeedReason || ids[13] != "sample-projection-1:projection:"+SampleWorkspace {
		t.Fatalf("seed store %v %+v %+v", ids, c.holds, c.runs)
	}
	states := []string{}
	for _, e := range c.entries {
		states = append(states, fmt.Sprint(e.view.Number, e.view.Change.Kind, e.view.State, e.view.Proposer, e.view.Approver, e.closedBy))
	}
	if s := strings.Join(states, "|"); s != "1policyappliedsample-adminsample-admin-2sample-admin-2|2policyappliedsample-adminsample-admin-2sample-admin-2|"+
		"3policyappliedsample-adminsample-admin-2sample-admin-2|4planappliedsample-adminsample-admin-2sample-admin-2|5planappliedsample-adminsample-admin-2sample-admin-2|"+
		"6releasependingsample-admin|7policypendingsample-admin-2" {
		t.Errorf("entries %s", s)
	}
	if v, err := s.retention.Hold(retentionTemporary); err != nil || v.Placer != SampleMember || fmt.Sprint(v.Counts) != "map[not_preserved:1]" {
		t.Errorf("member hold %+v %v", v, err)
	}
	if err := s.seedRetention(s.clock()); err == nil {
		t.Error("second seed")
	}
	counts := retentionSteps(t)
	if fmt.Sprint(counts) != "map[retention-approve:5 retention-claim:3 retention-place_hold:2 retention-proceed:2 retention-propose:7 retention-refer:2 "+
		"retention-register:14 retention-report:3 retention-run:2]" {
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
			if err := s.seedRetention(s.clock()); !errors.Is(err, retention.ErrInvalid) {
				t.Errorf("%s %d: %v", step, n, err)
			}
		}
	}
	mismatch := seedOnly(t)
	approvals := 0
	mismatch.fault = func(name string) bool {
		if name == "retention-approve" {
			approvals++
			if approvals == 5 {
				mismatch.ret.plans = 0
			}
		}
		return false
	}
	if err := mismatch.seedRetention(mismatch.clock()); !errors.Is(err, retention.ErrConflict) {
		t.Errorf("run mismatch %v", err)
	}
	broken := freshInstaller(t)
	broken.fault = func(name string) bool { return name == "retention-report" }
	if err := broken.Start(); !errors.Is(err, retention.ErrInvalid) {
		t.Errorf("start with a failing seed %v", err)
	}
}

func TestRetentionSamePlan(t *testing.T) {
	base := retention.DeletionPlan{Class: retention.ClassSource, Artifacts: []string{"a", "b"}, Chunk: 1, Lease: time.Hour, Reason: "r"}
	if !samePlan(base, base) {
		t.Error("same")
	}
	for name, change := range map[string]func(*retention.DeletionPlan){
		"class":  func(p *retention.DeletionPlan) { p.Class = retention.ClassFinal },
		"chunk":  func(p *retention.DeletionPlan) { p.Chunk = 2 },
		"lease":  func(p *retention.DeletionPlan) { p.Lease = time.Minute },
		"reason": func(p *retention.DeletionPlan) { p.Reason = "x" },
		"length": func(p *retention.DeletionPlan) { p.Artifacts = []string{"a"} },
		"member": func(p *retention.DeletionPlan) { p.Artifacts = []string{"a", "c"} },
	} {
		other := base
		change(&other)
		if samePlan(base, other) {
			t.Errorf("%s ignored", name)
		}
	}
}

func TestRetentionStoreAndTargets(t *testing.T) {
	c := newRetentionStore()
	c.keep(&retentionEntry{view: retention.ProposalView{Number: 1, State: retention.ProposalPending}})
	for n := uint64(2); n <= installKept+1; n++ {
		c.keep(&retentionEntry{view: retention.ProposalView{Number: n, State: retention.ProposalApplied}})
	}
	if len(c.entries) != installKept || c.entry(1) == nil || c.entry(2) != nil || c.entry(installKept+1) == nil {
		t.Errorf("kept %d", len(c.entries))
	}
	for index := range c.entries {
		c.entries[index].view.State = retention.ProposalPending
	}
	c.keep(&retentionEntry{view: retention.ProposalView{Number: 999, State: retention.ProposalPending}})
	if len(c.entries) != installKept+1 {
		t.Errorf("pending proposals dropped %d", len(c.entries))
	}
	if c.artifact("x") != nil || c.hold("x") != nil || c.run("x") != nil || c.visibleArtifact("x", SampleWorkspace) || c.visibleHold("x", SampleWorkspace) ||
		c.visibleRun("x", SampleWorkspace) {
		t.Error("empty store")
	}
	f := setup(t, SeedSample).ready()
	plan := &retentionEntry{workspace: SampleWorkspace, view: retention.ProposalView{Change: retention.Change{Kind: retention.ChangePlan, Plan: retention.DeletionPlan{Artifacts: []string{"sample-source-1",
		"other"}}}}}
	if f.s.ret.visibleEntry(plan, SampleWorkspace) {
		t.Error("plan with an unknown artifact is visible")
	}
	plan.view.Change.Plan.Artifacts = []string{"sample-source-1"}
	if !f.s.ret.visibleEntry(plan, SampleWorkspace) || f.s.ret.visibleEntry(plan, "other") {
		t.Error("plan visibility")
	}
	plan.workspace = "other"
	release := &retentionEntry{workspace: "other", view: retention.ProposalView{Change: retention.Change{Kind: retention.ChangeRelease,
		Release: retention.Release{Hold: retentionDispute}}}}
	if f.s.ret.visibleEntry(plan, SampleWorkspace) || f.s.ret.visibleEntry(release, SampleWorkspace) {
		t.Error("another workspace's plan or release is visible")
	}
	release.workspace = SampleWorkspace
	if !f.s.ret.visibleEntry(release, SampleWorkspace) {
		t.Error("release visibility")
	}
	for target, want := range map[string]uint64{retTarget(1): 1, retTarget(7): 7} {
		if e := f.s.retentionEntryOf(target, SampleWorkspace); e == nil || e.view.Number != want {
			t.Errorf("%s: %v", target, e)
		}
	}
	for _, target := range []string{"", retentionProposalTarget, retTarget(0), retTarget(8), "retention-proposal-07", "retention-proposal-7x", "recovery-proposal-7",
		"x-retention-proposal-7"} {
		if f.s.retentionEntryOf(target, SampleWorkspace) != nil {
			t.Errorf("%s resolved", target)
		}
	}
	if f.s.retentionEntryOf(retTarget(1), "other") != nil || f.s.retentionClassOf(retentionClassTarget+"source", "other") != "" ||
		f.s.retentionHoldOf(retentionHoldTarget+retentionDispute, "other") != nil || f.s.retentionRunOf("retention-run-1", "other") != nil {
		t.Error("other workspace targets")
	}
	if id, tenant := f.s.retentionHoldScopeOf(retentionTenantTarget+SampleWorkspace, "other"); id != "" || tenant != "" {
		t.Error("other workspace scope")
	}
	if f.s.retentionRunOf("retention-run-1", SampleWorkspace).id != "run/1" || f.s.retentionHoldOf(retentionHoldTarget+retentionDispute, SampleWorkspace).id != retentionDispute {
		t.Error("targets")
	}
	for n := uint64(100); n < 160; n++ {
		f.s.ret.keep(&retentionEntry{view: retention.ProposalView{Number: n, State: retention.ProposalWithdrawn, Proposer: SampleAdmin,
			Change: retention.Change{Kind: retention.ChangePolicy, Policy: retention.Policy{Class: retention.ClassClaim, Period: 1}}}})
	}
	got := retSection(f.retentionRead(f.as(AdminLogin)), "proposals")
	rows := retRows(got)
	waiting, oldest := 0, ""
	for _, row := range rows {
		if row["state"] == retention.ProposalPending {
			waiting++
		} else {
			oldest = fmt.Sprint(row["number"])
		}
	}
	if len(rows) != retentionShown || fmtInt(got["total"]) != 67 || got["more"] != true || fmt.Sprint(rows[0]["number"]) != "159" || waiting == 0 ||
		oldest != fmt.Sprint(110+waiting) {
		t.Errorf("page %d %v waiting %d oldest closed %s", len(rows), got["total"], waiting, oldest)
	}
}

func TestRetentionRowsAreBounded(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	bulk := []string{}
	for n := 0; n <= retentionWindow; n++ {
		id := fmt.Sprintf("bulk-%04d", n)
		bulk = append(bulk, id)
		f.retRegister(id, retention.ClassClaim, 10, SampleWorkspace)
	}
	for n := 0; n <= retentionShown; n++ {
		f.retRegister(fmt.Sprintf("conv-%02d", n), retention.ClassConversation, 1, SampleWorkspace)
	}
	for n := 1; n <= retentionShown; n++ {
		if err := f.s.retention.Refer(retention.Reference{From: "conv-00", To: fmt.Sprintf("conv-%02d", n), Kind: retention.Requires}, retIngest); err != nil {
			t.Fatal(err)
		}
		f.retPlan(retention.ClassConversation, SampleWorkspace, fmt.Sprintf("conv-%02d", n))
	}
	p, err := f.s.retention.Propose(retention.Change{Kind: retention.ChangePolicy, Policy: retention.Policy{Class: retention.ClassClaim, Period: 1}}, retAnn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.retention.Approve(p.Number, retBob, retention.Impact{}); err != nil {
		t.Fatal(err)
	}
	plan := retention.DeletionPlan{Class: retention.ClassClaim, Artifacts: bulk, Chunk: retentionWindow, Lease: time.Hour, Reason: "bulk"}
	e, err := f.s.retentionPropose(retention.Change{Kind: retention.ChangePlan, Plan: plan}, retAnn, SampleWorkspace, "bulk")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.s.retentionApprove(e, retBob, retention.Impact{}, "bulk"); err != nil {
		t.Fatal(err)
	}
	f.s.ret.runs = append(f.s.ret.runs, &retentionRun{id: "run/53", workspace: SampleWorkspace})
	f.retDelete("run/53", 0, bulk[:retentionWindow]...)
	f.retDelete("run/53", 1, bulk[retentionWindow])
	for n := 0; n < retentionShown; n++ {
		if _, err := f.s.retentionPlaceHold(retention.Hold{ID: fmt.Sprint("hold-", n), Owner: SampleAdmin, Reason: "bulk", Artifacts: []string{"sample-claim-1"}}, retAnn,
			SampleWorkspace, "bulk"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.s.retentionPlaceHold(retention.Hold{ID: "bulk-hold", Owner: SampleAdmin, Reason: "bulk", Tenants: []string{SampleWorkspace},
		Classes: []string{retention.ClassClaim}}, retAnn, SampleWorkspace, "bulk"); err != nil {
		t.Fatal(err)
	}
	got := f.retentionRead(root)
	if got.status != 200 {
		t.Fatal(got.raw)
	}
	f.retBalanced(t, got)
	claim := retGroup(t, got, retention.ClassClaim)
	holds, deletion, proposals := retSection(got, "holds"), retSection(got, "deletion"), retSection(got, "proposals")
	refs := holds["references"].(map[string]any)
	tomb := retSection(got, "decisions")["tombstones"].(map[string]any)
	finished := f.readAs(root, "/api/v1/retention/runs")
	if rows, _ := finished.body["runs"].([]any); finished.status != 200 || len(rows) == 0 {
		t.Fatalf("the finished runs: %s", finished.raw[:min(len(finished.raw), 300)])
	}
	run := finished.body["runs"].([]any)[0].(map[string]any)
	if run["run"].(map[string]any)["label"] != "run/53" {
		t.Fatalf("the newest finished run is %v", run["run"])
	}
	hold := retHold(t, got, "bulk-hold")
	sizes := fmt.Sprint(len(claim["artifacts"].([]any)), claim["total"], claim["more"], len(retRows(holds)), holds["total"], holds["more"], len(retRows(refs)), refs["total"],
		refs["more"], len(retRows(deletion)), deletion["total"], deletion["more"], len(retRows(proposals)), proposals["total"], proposals["more"],
		len(retRows(tomb)), tomb["total"], tomb["more"], retRows(tomb)[0]["sequence"], len(run["artifacts"].(map[string]any)["rows"].([]any)), run["artifacts"].(map[string]any)["total"],
		run["artifacts"].(map[string]any)["more"], len(hold["outcomes"].([]any)), hold["outcomes_more"], retSection(got, "decisions")["watermark"])
	if sizes != "50 1002 true 53 53 false 50 51 true 51 53 true 50 58 true 50 1000 true 1003 50 1001 true 50 true 1003" {
		t.Errorf("sizes %s", sizes)
	}
	pages := 0
	f.s.fault = func(step string) bool {
		if step == "retention-hold_outcomes" {
			pages++
		}
		return false
	}
	outcomes, err := (&retentionFacts{s: f.s, workspace: SampleWorkspace}).visibleOutcomes("bulk-hold", "", retentionShown)
	f.s.fault = nil
	if len(outcomes) != retentionShown+950 || pages != 1 || err != nil {
		t.Errorf("a full visible page took %d calls for %d outcomes %v", pages, len(outcomes), err)
	}
}

func TestMapRetentionErrors(t *testing.T) {
	cases := map[error]commands.Code{
		retention.Refusal{Reason: retention.ReasonSelfApproval}:   commands.PrerequisiteFailed,
		retention.Refusal{Reason: retention.ReasonNotProposer}:    commands.PrerequisiteFailed,
		retention.Refusal{Reason: retention.ReasonAINotAllowed}:   commands.InternalError,
		retention.Refusal{Reason: retention.ReasonNotHuman}:       commands.InternalError,
		retention.Refusal{Reason: retention.ReasonNotSystem}:      commands.InternalError,
		retention.Refusal{Reason: retention.ReasonNotPending}:     commands.InvalidStage,
		retention.Refusal{Reason: retention.ReasonImpactRequired}: commands.InvalidStage,
		retention.Refusal{Reason: retention.ReasonImpactChanged}:  commands.InvalidStage,
		retention.Refusal{Reason: retention.ReasonPlacer}:         commands.InvalidStage,
		retention.Refusal{Reason: retention.ReasonReleased}:       commands.InvalidStage,
		retention.Refusal{Reason: retention.ReasonInPlan}:         commands.InvalidStage,
		retention.Refusal{Reason: retention.ReasonAlreadyDeleted}: commands.InvalidStage,
		retention.ErrNotFound:   commands.InvalidStage,
		retention.ErrConflict:   commands.InvalidStage,
		retention.ErrLimit:      commands.PrerequisiteFailed,
		retention.ErrInvalid:    commands.InternalError,
		retention.ErrRejected:   commands.InternalError,
		errors.New("unrelated"): commands.InternalError,
	}
	for err, want := range cases {
		var failure *commands.Failure
		if !errors.As(mapRetention(err), &failure) || failure.Code != want || len(failure.Fields) != 0 {
			t.Errorf("%v: %v", err, failure)
		}
	}
	if mapRetention(nil) != nil {
		t.Error("nil mapped")
	}
	for err, want := range map[error]string{nil: "", retention.Refusal{Reason: "x"}: "x", retention.ErrNotFound: "not_found", retention.ErrLimit: "limit",
		retention.ErrConflict: "conflict", retention.ErrInvalid: "invalid", errors.New("other"): "invalid"} {
		if got := retentionReason(err); got != want {
			t.Errorf("%v: %q", err, got)
		}
	}
	if decisionOf(retention.AccessBlocked) != notConnected || decisionOf(retention.AccessOK) != "" || excludedOutcome(retention.OutcomePending) {
		t.Error("helpers")
	}
}

func TestRetentionCommandSet(t *testing.T) {
	f := setup(t, SeedSample).ready()
	ids := []string{}
	for id, mutating := range f.s.mutating {
		if strings.HasPrefix(id, retentionCommand) {
			if !mutating {
				t.Errorf("%s is not mutating", id)
			}
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	if strings.Join(ids, " ") != "retention.approve retention.place_hold retention.propose_policy retention.propose_release retention.stop_run retention.withdraw" {
		t.Errorf("commands %v", ids)
	}
}

func TestRetentionActorsAndEffectsAreConfined(t *testing.T) {
	seedOnlyCalls := map[string]bool{"Register": true, "Refer": true, "Claim": true, "Proceed": true, "Report": true}
	never := map[string]bool{"AddTenant": true, "Retry": true, "BackupExpired": true}
	found := map[string]int{}
	for _, src := range serverSources(t) {
		commandFile := src.name == "retention.go" || src.name == "retentioncmds.go"
		for _, decl := range src.file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			seed := fn.Name.Name == "seedRetention"
			ast.Inspect(fn, func(n ast.Node) bool {
				if name, ok := packageSelector(n, "retention"); ok && (name == "KindSystem" || name == "KindAssistant") {
					found[name]++
					if !seed {
						t.Errorf("%s %s names retention.%s", src.name, fn.Name.Name, name)
					}
				}
				if sel, ok := n.(*ast.SelectorExpr); ok && selectorOn(sel.X, "retention") {
					if seedOnlyCalls[sel.Sel.Name] {
						found[sel.Sel.Name]++
					}
					if seedOnlyCalls[sel.Sel.Name] && !seed || never[sel.Sel.Name] {
						t.Errorf("%s %s calls retention %s", src.name, fn.Name.Name, sel.Sel.Name)
					}
				}
				lit, ok := n.(*ast.CompositeLit)
				if !ok {
					return true
				}
				kind, inPackage := packageSelector(lit.Type, "retention")
				if !inPackage {
					return true
				}
				for _, elt := range lit.Elts {
					kv, ok := elt.(*ast.KeyValueExpr)
					if !ok {
						continue
					}
					key := fmt.Sprint(kv.Key)
					value, _ := packageSelector(kv.Value, "retention")
					if kind == "Change" && key == "Kind" && value == "ChangePlan" && !seed {
						t.Errorf("%s %s builds a plan change", src.name, fn.Name.Name)
					}
					if kind == "Actor" && key == "Kind" && commandFile && value != "KindHuman" {
						t.Errorf("%s %s builds a %s actor", src.name, fn.Name.Name, value)
					}
					if kind == "Actor" && key == "ID" && fmt.Sprint(kv.Value) == "SampleMember" {
						found["member actors"]++
						if !seed {
							t.Errorf("%s %s acts as the sample member", src.name, fn.Name.Name)
						}
					}
					if kind == "Actor" && key == "Kind" && value == "KindHuman" {
						found["human actors"]++
					}
				}
				return true
			})
		}
		if commandFile {
			ast.Inspect(src.file, func(n ast.Node) bool {
				if lit, ok := n.(*ast.BasicLit); ok && (strings.Contains(lit.Value, "system") || strings.Contains(lit.Value, "assistant")) {
					t.Errorf("%s has the literal %s", src.name, lit.Value)
				}
				return true
			})
		}
	}
	if fmt.Sprint(found) != "map[Claim:1 KindSystem:2 Proceed:1 Refer:2 Register:1 Report:1 human actors:4 member actors:1]" {
		t.Errorf("found %v", found)
	}
}

func TestRetentionCommandsActAsTheHumanReader(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	f.ok(t, "approve", f.run(second, "retention.approve", retTarget(6), nil), "applied", "")
	if v, err := f.s.retention.Proposal(6); err != nil || v.Approver != SampleAdmin2 || v.Proposer != SampleAdmin {
		t.Errorf("approver %+v %v", v, err)
	}
	f.ok(t, "hold", f.run(second, "retention.place_hold", retentionArtifactTarget+"sample-source-5", map[string]string{"owner": SampleAdmin, "reason": "x"}), "held", "")
	if v := f.retHoldView("retention-hold-1"); v.Placer != SampleAdmin2 {
		t.Errorf("placer %+v", v)
	}
	f.ok(t, "stop", f.run(root, "retention.stop_run", "retention-run-2", nil), "done", "")
	if v := f.retRunView("run/2"); v.Stopper != SampleAdmin {
		t.Errorf("stopper %+v", v)
	}
	f.ok(t, "propose", f.run(second, "retention.propose_policy", retentionClassTarget+retention.ClassClaim, map[string]string{"period": "3", "grace": "0"}), "pending", "")
	if v, err := f.s.retention.Proposal(8); err != nil || v.Proposer != SampleAdmin2 {
		t.Errorf("proposer %+v %v", v, err)
	}
}

func TestRetentionConcurrentReadsCommandsAndChecks(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				if got := f.retentionRead(second); got.status != 200 {
					t.Errorf("concurrent read %d", got.status)
				}
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				f.s.retention.Check(retention.Change{Kind: retention.ChangeRelease, Release: retention.Release{Hold: retentionDispute, Reason: "x"}}, SampleAdmin2, "", retention.Impact{})
				f.s.retention.Standing("sample-temporary-2")
			}
		}()
	}
	for i := 0; i < 3; i++ {
		for _, c := range []struct{ command, target, fields string }{{"place_hold", retentionArtifactTarget + "sample-source-5", `"owner":"sample-admin","reason":"x"`},
			{"propose_policy", retentionClassTarget + retention.ClassClaim, fmt.Sprintf(`"period":"%d","grace":"0"`, i+1)}, {"stop_run", "retention-run-2", ""}} {
			body := fmt.Sprintf(`{"request_id":"c%d%s","target_id":"%s","idempotency_key":"c%d%s","expected_revision":"%s","rationale":"x","body":{%s}}`, i, c.command, c.target, i,
				c.command, f.s.Revision(), c.fields)
			f.call(retentionCommand+c.command, body, root.cookie, root.csrf)
		}
	}
	wg.Wait()
}

func TestRetentionLeaksNothing(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	f.retRegister("other-only", retention.ClassSource, 400, "other-tenant")
	f.retRegister("shared-a", retention.ClassSource, 400, "other-tenant", SampleWorkspace)
	f.retRegister("shared-b", retention.ClassSource, 400, "other-tenant", SampleWorkspace)
	if err := f.s.retention.Refer(retention.Reference{From: "other-only", To: "sample-source-7", Kind: retention.Requires}, retIngest); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.retentionPlaceHold(retention.Hold{ID: "other-hold", Owner: "other-owner", Reason: "other", Artifacts: []string{"other-only"}}, retAnn, "other-tenant",
		"other"); err != nil {
		t.Fatal(err)
	}
	run := f.retPlan(retention.ClassSource, "other-tenant", "other-only", "shared-a", "shared-b")
	c, err := f.s.retention.Claim(run, 0, retWorker)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.s.retention.Report(run, 0, c.Token, retentionDatabase, []string{"shared-a"}, nil, retWorker); err != nil {
		t.Fatal(err)
	}
	words := []string{"other-tenant", "other-only", "other-hold", "other-owner", `"` + run + `"`, runTarget(run), "token", "deadline", "until", "tenants", retentionIngest,
		retentionWorker, stamp(c.Until), "2026-11-16T10:00:00Z"}
	raws := map[string]string{"read": f.retentionRead(root).raw, "second read": f.retentionRead(second).raw}
	for _, c := range []struct {
		who            actor
		kind, target   string
		fields         map[string]string
		expectedStatus int
	}{
		{second, retentionApproveKind, retTarget(6), nil, 200},
		{root, retentionHoldKind, retentionArtifactTarget + "shared-b", map[string]string{"owner": SampleAdmin, "reason": "x"}, 200},
		{root, retentionHoldKind, retentionTenantTarget + SampleWorkspace, map[string]string{"owner": SampleAdmin, "reason": "x"}, 200},
		{second, retentionReleaseKind, retentionHoldTarget + retentionDispute, map[string]string{"reason": "x"}, 200},
		{root, retentionStopKind, "retention-run-2", nil, 200},
		{root, retentionPolicyKind, retentionClassTarget + retention.ClassClaim, map[string]string{"period": "1", "grace": "0"}, 200},
		{root, retentionStopKind, "retention-run-1", nil, 422},
	} {
		got := f.run(c.who, retentionCommand+c.kind, c.target, c.fields)
		if got.status != c.expectedStatus {
			t.Fatalf("%s: %s", c.kind, got.raw)
		}
		raws[c.kind+" "+c.target] = got.raw
	}
	raws["final read"] = f.retentionRead(root).raw
	for name, raw := range raws {
		for _, word := range words {
			if strings.Contains(raw, word) {
				t.Errorf("%s carries %q", name, word)
			}
		}
	}
	if !strings.Contains(raws["read"], `"workspace_count":2`) || !strings.Contains(raws["read"], `"watermark":3`) {
		t.Error("the installation-wide figures are missing")
	}
}

func TestRetentionHiddenArtifactsInVisibleRuns(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.retRegister("other-d", retention.ClassSource, 400, "other-tenant")
	f.retRegister("other-x", retention.ClassSource, 10, "other-tenant")
	run := f.retPlan(retention.ClassSource, SampleWorkspace, "other-d", "other-x", "sample-source-5")
	f.retDelete(run, 0, "other-d")
	if f.s.ret.run(run) != nil {
		t.Fatal("a run with another tenant's artifacts was stored for this workspace")
	}
	got := f.retentionRead(root)
	deletion := retSection(got, "deletion")
	tomb := retSection(got, "decisions")["tombstones"].(map[string]any)
	if s := fmt.Sprint(deletion["total"], " ", tomb["total"], " ", retSection(got, "decisions")["watermark"], " ", retArtifact(t, got, "sample-source-5")["run"]); s !=
		"2 2 3 <nil>" {
		t.Errorf("hidden %s", s)
	}
	for _, word := range []string{"other-d", "other-x", "other-tenant", `"` + run + `"`, runTarget(run)} {
		if strings.Contains(got.raw, word) {
			t.Errorf("read carries %s", word)
		}
	}
	f.s.ret.runs = append(f.s.ret.runs, &retentionRun{id: run, workspace: SampleWorkspace})
	row := retRun(t, f.retentionRead(root), run)
	if s := fmt.Sprint(row["artifacts"], " ", row["stores"]); s != "map[more:false next: rows:[map[id:sample-source-5 outcome:not_due]] total:1] "+
		"[map[deleted:1 excluded:2 failed:0 pending:0 store:sample-database target:3]]" {
		t.Errorf("injected run %s", s)
	}
	visible := 0
	for _, a := range f.s.ret.artifacts {
		if f.s.ret.visibleArtifact(a.id, SampleWorkspace) {
			visible++
		}
	}
	calls := 0
	f.s.fault = func(step string) bool {
		if step == "retention-artifact" {
			calls++
		}
		return step == "retention-artifact" && calls == visible+1
	}
	f.code(t, "hidden artifact fault", f.retentionRead(root), 503, "dependency_unavailable")
	if calls != visible+1 {
		t.Errorf("artifact calls %d of %d", calls, visible+1)
	}
}

func TestRetentionStoredRunsAreVisibleRuns(t *testing.T) {
	c := newRetentionStore()
	c.artifacts = []*retentionArtifact{{id: "a", tenants: []string{"w1"}}, {id: "b", tenants: []string{"w1", "w2"}}}
	c.recordRun("run/1", "w1", []string{"a", "b"})
	c.recordRun("run/2", "w2", []string{"a", "b"})
	c.recordRun("run/3", "w2", []string{"b"})
	if c.visibleRun("run/1", "w1") != true || c.run("run/2") != nil || !c.visibleRun("run/3", "w2") {
		t.Errorf("runs %+v", c.runs)
	}
}

func TestRetentionOutcomesFilterBeforeTheCap(t *testing.T) {
	f := setup(t, SeedSample).ready()
	ids := []string{}
	for n := 0; n < retentionOutcomePage; n++ {
		id := fmt.Sprintf("hid-%04d", n)
		ids = append(ids, id)
		f.retRegister(id, retention.ClassClaim, 10, "other-tenant")
	}
	for _, id := range []string{"vis-1", "vis-2"} {
		ids = append(ids, id)
		f.retRegister(id, retention.ClassClaim, 10, SampleWorkspace)
	}
	p, err := f.s.retention.Propose(retention.Change{Kind: retention.ChangePolicy, Policy: retention.Policy{Class: retention.ClassClaim, Period: 1}}, retAnn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.retention.Approve(p.Number, retBob, retention.Impact{}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.retention.Propose(retention.Change{Kind: retention.ChangePlan, Plan: retention.DeletionPlan{Class: retention.ClassClaim, Artifacts: ids,
		Chunk: retentionOutcomePage, Lease: time.Hour, Reason: "bulk"}}, retAnn); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.retention.Approve(p.Number+1, retBob, retention.Impact{}); err != nil {
		t.Fatal(err)
	}
	f.retDelete("run/3", 0, ids[:retentionOutcomePage]...)
	f.retDelete("run/3", 1, ids[retentionOutcomePage:]...)
	if _, err := f.s.retentionPlaceHold(retention.Hold{ID: "mixed-hold", Owner: SampleAdmin, Reason: "mixed", Artifacts: ids}, retAnn, SampleWorkspace, "mixed"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.retentionPlaceHold(retention.Hold{ID: "visible-hold", Owner: SampleAdmin, Reason: "visible", Artifacts: []string{"vis-1", "vis-2"}}, retAnn,
		SampleWorkspace, "visible"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.retentionPlaceHold(retention.Hold{ID: "tenant-hold", Owner: SampleAdmin, Reason: "tenant", Tenants: []string{"other-tenant", SampleWorkspace}}, retAnn,
		SampleWorkspace, "tenant"); err != nil {
		t.Fatal(err)
	}
	read := f.retentionRead(f.as(AdminLogin))
	f.retAgrees(t, read)
	h := retHold(t, read, "mixed-hold")
	if s := fmt.Sprint(len(h["outcomes"].([]any)), " ", h["outcomes_more"], " ", h["scope"].(map[string]any)["artifacts"], " ", h["counts"], " ", h["counts_scope"]); s !=
		"2 false [vis-1 vis-2] <nil> other_scope" {
		t.Errorf("mixed %s", s)
	}
	if s := fmt.Sprint(retHold(t, read, "tenant-hold")["counts"], " ", retHold(t, read, "tenant-hold")["counts_scope"], " ",
		retHold(t, read, "tenant-hold")["scope"].(map[string]any)["workspace"]); s != "<nil> other_scope true" {
		t.Errorf("tenant %s", s)
	}
	visible := retHold(t, read, "visible-hold")
	if s := fmt.Sprint(visible["counts"], " ", visible["counts_scope"], " ", len(visible["outcomes"].([]any))); s !=
		"map[not_preserved:2 outcome_pending:0 partially_deleted:0] workspace 2" {
		t.Errorf("visible %s", s)
	}
	if strings.Contains(read.raw, "1002") || strings.Contains(read.raw, "other-tenant") || strings.Contains(read.raw, "hid-") {
		t.Error("the read carries another tenant's counts or identifiers")
	}
}
