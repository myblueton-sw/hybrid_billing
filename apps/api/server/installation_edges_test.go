package server

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/installer"
)

func TestInstallationFaults(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	at := ""
	f.s.fault = func(step string) bool { return step == at }
	at = "installer-read"
	f.code(t, "read", f.installation(root), 503, "dependency_unavailable")
	cases := []struct {
		step, command, target string
		who                   actor
		fields                map[string]string
	}{
		{"installer-approve", "installation.approve", proposalAt(6), root, nil},
		{"installer-withdraw", "installation.withdraw", proposalAt(7), root, nil},
		{"installer-propose", "installation.propose_plan", installTarget, second, map[string]string{"version": "1.0.0", "hash": strings.Repeat("a", 64)}},
	}
	for _, c := range cases {
		at = c.step
		f.code(t, c.step, f.run(c.who, c.command, c.target, c.fields), 500, "internal_error")
	}
	at = ""
	f.s.fault = nil
	if s := fmt.Sprintf("%v %v", installRow(t, f.installation(root), 6)["state"], installRow(t, f.installation(root), 7)["state"]); s != "pending pending" {
		t.Errorf("faults changed state %s", s)
	}
	f.s.install.releases = append(f.s.install.releases, "9.9.9")
	f.code(t, "unknown stored release", f.installation(root), 503, "dependency_unavailable")
	f.s.install.releases = f.s.install.releases[:2]
	f.s.install.site = "absent-site"
	f.code(t, "unknown stored site", f.installation(root), 503, "dependency_unavailable")
}

func freshInstaller(t *testing.T) *Server {
	t.Helper()
	s, err := New(Config{Data: t.TempDir() + "/d", Seed: SeedSample, Clock: func() time.Time { return start }})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestInstallationSeedFailures(t *testing.T) {
	s := freshInstaller(t)
	now := s.clock()
	if err := s.seedInstallation(now); err != nil {
		t.Fatal(err)
	}
	if len(s.install.entries) != 7 || s.install.site != installSite || strings.Join(s.install.releases, ",") != "1.0.0,1.1.0" || len(s.install.destinations) != 1 ||
		len(s.install.reports) != 1 || !s.install.observed.Equal(now) {
		t.Fatalf("seed store %+v", s.install)
	}
	if err := s.seedInstallation(now); !errors.Is(err, installer.ErrConflict) {
		t.Errorf("second seed %v", err)
	}
	for _, step := range []string{"installer-propose", "installer-approve", "installer-observe", "installer-report", "installer-preview"} {
		s := freshInstaller(t)
		s.fault = func(name string) bool { return name == step }
		if err := s.seedInstallation(s.clock()); !errors.Is(err, installer.ErrInvalid) {
			t.Errorf("%s: %v", step, err)
		}
	}
	s = freshInstaller(t)
	e, err := s.installPropose(installer.Change{Kind: installer.ChangeSite, Site: installer.Site{ID: installSite, Tenant: SampleWorkspace, Platform: installPlatform,
		Probe: installProbe}}, sampleAdminActor, "r")
	if err != nil {
		t.Fatal(err)
	}
	var refusal installer.Refusal
	if err := s.installApprove(e, sampleAdminActor, "r"); !errors.As(err, &refusal) || refusal.Reason != installer.ReasonSelfApproval {
		t.Errorf("self approval in seed %v", err)
	}
	h := freshInstaller(t)
	if err := h.Start(); err != nil {
		t.Fatal(err)
	}
	broken := freshInstaller(t)
	broken.fault = func(name string) bool { return name == "installer-observe" }
	if err := broken.Start(); !errors.Is(err, installer.ErrInvalid) {
		t.Errorf("start with a failing seed %v", err)
	}
}

func TestInstallStoreKeepsPendingProposals(t *testing.T) {
	c := newInstallStore()
	c.keep(&installEntry{view: installer.ProposalView{Number: 1, State: installer.ProposalPending}})
	for n := uint64(2); n <= installKept+1; n++ {
		c.keep(&installEntry{view: installer.ProposalView{Number: n, State: installer.ProposalApplied}})
	}
	if len(c.entries) != installKept || c.entry(1) == nil || c.entry(2) != nil || c.entry(3) == nil || c.entry(installKept+1) == nil {
		t.Errorf("kept %d first %v", len(c.entries), c.entries[0].view.Number)
	}
	for n := uint64(500); n < 500+installKept; n++ {
		c.entries[n-500].view = installer.ProposalView{Number: n, State: installer.ProposalPending}
	}
	c.keep(&installEntry{view: installer.ProposalView{Number: 999, State: installer.ProposalPending}})
	if len(c.entries) != installKept+1 {
		t.Errorf("pending proposals dropped %d", len(c.entries))
	}
}

func TestInstallationProposalPageIsBounded(t *testing.T) {
	f := setup(t, SeedSample).ready()
	for n := uint64(100); n < 160; n++ {
		f.s.install.keep(&installEntry{view: installer.ProposalView{Number: n, State: installer.ProposalWithdrawn, Proposer: SampleAdmin,
			Change: installer.Change{Kind: installer.ChangeDestination}}})
	}
	rows := f.installation(f.as(AdminLogin)).body["proposals"].([]any)
	if n, first, waiting, oldest := listedShape(rows); n != installShown || first != 159 || waiting == 0 || oldest != 110+waiting {
		t.Errorf("page %d first %d waiting %d oldest closed %d", n, first, waiting, oldest)
	}
}

func TestInstallEntryTargets(t *testing.T) {
	f := setup(t, SeedSample).ready()
	for target, want := range map[string]uint64{proposalAt(1): 1, proposalAt(7): 7} {
		if e := f.s.installEntry(target); e == nil || e.view.Number != want {
			t.Errorf("%s: %v", target, e)
		}
	}
	for _, target := range []string{"", "installation", proposalAt(0), proposalAt(8), "installation-proposal-+7", "installation-proposal-07", "installation-proposal-7x",
		"installation-proposal-18446744073709551616", "x-installation-proposal-7"} {
		if e := f.s.installEntry(target); e != nil {
			t.Errorf("%s resolved", target)
		}
	}
}

func TestMapInstallerErrors(t *testing.T) {
	cases := map[error]commands.Code{
		installer.Refusal{Reason: installer.ReasonSelfApproval}: commands.PrerequisiteFailed,
		installer.Refusal{Reason: installer.ReasonNotProposer}:  commands.PrerequisiteFailed,
		installer.Refusal{Reason: installer.ReasonNotPending}:   commands.InvalidStage,
		installer.Refusal{Reason: installer.ReasonPlanChanged}:  commands.InvalidStage,
		installer.ErrNotFound: commands.InvalidStage,
		installer.ErrConflict: commands.InvalidStage,
		installer.ErrLimit:    commands.PrerequisiteFailed,
		installer.ErrInvalid:  commands.InternalError,
		installer.ErrRejected: commands.InternalError,
		errors.New("other"):   commands.InternalError,
	}
	for err, want := range cases {
		var failure *commands.Failure
		if !errors.As(mapInstaller(err), &failure) || failure.Code != want || len(failure.Fields) != 0 {
			t.Errorf("%v: %v", err, failure)
		}
	}
	if mapInstaller(nil) != nil {
		t.Error("nil mapped")
	}
	if installReason(installer.ErrNotFound) != notObserved || installReason(installer.Refusal{Reason: "x"}) != "x" {
		t.Error("preview reasons")
	}
}

func TestInstallationLimitMapsToPrerequisite(t *testing.T) {
	f := setup(t, SeedSample).ready()
	small := InstallerLimits()
	small.Proposals = 1
	r, err := installer.NewRegistry(installer.Config{Random: strings.NewReader(strings.Repeat("x", 64))}, small, f.s.clock)
	if err != nil {
		t.Fatal(err)
	}
	f.s.installer, f.s.install = r, newInstallStore()
	if err := f.s.seedInstallation(f.s.clock()); !errors.Is(err, installer.ErrLimit) {
		t.Fatalf("small seed %v", err)
	}
	next, err := r.Preview(installSite, "1.1.0")
	if err != nil {
		t.Fatal(err)
	}
	root := f.as(AdminLogin)
	f.code(t, "limit", f.run(root, "installation.propose_plan", installTarget, map[string]string{"version": "1.1.0", "hash": next.Hash}), 422, "prerequisite_failed")
}

func TestInstallationCommandsActAsTheHumanReader(t *testing.T) {
	data, err := parser.ParseFile(token.NewFileSet(), "installation.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	ast.Inspect(data, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok && (sel.Sel.Name == "KindSystem" || sel.Sel.Name == "KindAssistant") {
			t.Errorf("installation.go names %s", sel.Sel.Name)
		}
		return true
	})
	seed, err := parser.ParseFile(token.NewFileSet(), "installseed.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, decl := range seed.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		ast.Inspect(fn, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok && (sel.Sel.Name == "KindSystem" || sel.Sel.Name == "KindAssistant") && fn.Name.Name != "seedInstallation" {
				t.Errorf("%s names %s", fn.Name.Name, sel.Sel.Name)
			}
			return true
		})
	}
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	next, _ := f.s.installer.Preview(installSite, "1.1.0")
	for _, c := range []struct {
		who            actor
		command        string
		target         string
		fields         map[string]string
		number         uint64
		proposer, by   string
		state, approve string
	}{
		{root, "installation.approve", proposalAt(6), nil, 6, SampleAdmin2, SampleAdmin, installer.ProposalApplied, SampleAdmin},
		{second, "installation.approve", proposalAt(7), nil, 7, SampleAdmin, SampleAdmin2, installer.ProposalApplied, SampleAdmin2},
		{second, "installation.propose_plan", installTarget, map[string]string{"version": "1.1.0", "hash": next.Hash}, 8, SampleAdmin2, "", installer.ProposalPending, ""},
	} {
		if got := f.run(c.who, c.command, c.target, c.fields); got.status != 200 {
			t.Fatalf("%s %s", c.command, got.raw)
		}
		view, err := f.s.installer.Proposal(c.number)
		e := f.s.install.entry(c.number)
		if err != nil || view.Proposer != c.proposer || view.Approver != c.approve || view.State != c.state || e.closedBy != c.by || e.view.State != c.state {
			t.Errorf("%s: %+v %+v %v", c.command, view, e, err)
		}
	}
	if view, _ := f.s.installer.Proposal(8); view.Change.Acceptance.Admin != SampleAdmin2 {
		t.Errorf("plan administrator %+v", view)
	}
	if gone := f.run(second, "installation.withdraw", proposalAt(8), nil); gone.status != 200 || f.s.install.entry(8).closedBy != SampleAdmin2 {
		t.Errorf("withdraw %s", gone.raw)
	}
}

func TestInstallationConcurrentReadsAndCommands(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	next, _ := f.s.installer.Preview(installSite, "1.1.0")
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				if got := f.installation(second); got.status != 200 {
					t.Errorf("concurrent read %d", got.status)
				}
			}
		}()
	}
	for i := 0; i < 3; i++ {
		body := fmt.Sprintf(`{"request_id":"p%d","target_id":"installation","idempotency_key":"p%d","expected_revision":"%s","rationale":"x","body":{"version":"1.1.0","hash":"%s"}}`,
			i, i, f.s.Revision(), next.Hash)
		f.call("installation.propose_plan", body, root.cookie, root.csrf)
		body = fmt.Sprintf(`{"request_id":"w%d","target_id":"%s","idempotency_key":"w%d","expected_revision":"%s","rationale":"x","body":{}}`, i, proposalAt(8), i, f.s.Revision())
		f.call("installation.withdraw", body, root.cookie, root.csrf)
	}
	wg.Wait()
}

func TestInstallTargetsAndPlanChoiceBoundaries(t *testing.T) {
	f := setup(t, SeedSample).ready()
	if f.s.installEntry("1") != nil || f.s.installEntry(proposalTarget+"01") != nil || f.s.installEntry(proposalTarget+"x") != nil {
		t.Error("non-canonical target resolved")
	}
	if f.s.installEntry(proposalTarget+"1") == nil {
		t.Error("canonical target not resolved")
	}
	saved := f.s.install.releases
	f.s.install.releases = []string{"1.0.0", "1.1.0"}
	if got := f.s.newestOther("1.1.0"); got != "1.0.0" {
		t.Errorf("newest other %q", got)
	}
	f.s.install.releases = saved
	entries := f.s.install.entries
	plan := &installEntry{view: installer.ProposalView{Number: 99, State: installer.ProposalPending, Change: installer.Change{Kind: installer.ChangePlan,
		Acceptance: installer.Acceptance{Version: "0.9.0"}}}}
	hidden := []*installEntry{plan}
	for n := 0; n < installShown+5; n++ {
		hidden = append(hidden, &installEntry{view: installer.ProposalView{Number: uint64(100 + n), State: installer.ProposalApplied, Change: installer.Change{Kind: installer.ChangePlan,
			Acceptance: installer.Acceptance{Version: "0.8.0"}}}})
	}
	f.s.install.entries = hidden
	if got, basis := f.s.planTarget(""); got != "0.9.0" || basis != "pending_plan" {
		t.Errorf("plan target %q basis %q with the pending plan older than the rows a read shows", got, basis)
	}
	f.s.install.entries = nil
	if got, basis := f.s.planTarget(""); got == "" || basis != "newest_release" {
		t.Errorf("plan target %q basis %q without a pending plan", got, basis)
	}
	f.s.install.releases = nil
	if got, basis := f.s.planTarget(""); got != "" || basis != "" {
		t.Errorf("plan target %q basis %q without releases", got, basis)
	}
	f.s.install.releases = saved
	f.s.install.entries = entries
}
