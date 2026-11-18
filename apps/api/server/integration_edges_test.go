package server

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/events"
	"github.com/myblueton-sw/hybrid_billing/modules/plugins"
	"github.com/myblueton-sw/hybrid_billing/modules/webhooks"
)

func TestIntegrationsReadFaults(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	at := ""
	f.s.fault = func(step string) bool { return step == at }
	for _, step := range []string{"integrations-read", "plugins-publisher", "plugins-version", "plugins-installation", "plugins-workspace", "plugins-recheck", "plugins-licenses", "webhooks-subscription",
		"webhooks-keys", "webhooks-deliveries", "events-bounds", "services"} {
		at = step
		f.code(t, step, f.integrations(root), 503, "dependency_unavailable")
	}
	at = ""
	if got := f.integrations(root); got.status != 200 {
		t.Errorf("read after faults %s", got.raw)
	}
	f.s.integ.publishers = append(f.s.integ.publishers, "sample-absent")
	f.code(t, "unknown stored publisher", f.integrations(root), 503, "dependency_unavailable")
}

func TestIntegrationCommandFaults(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	at := ""
	f.s.fault = func(step string) bool { return step == at }
	f.s.integ.redeliver = map[string]letterRequest{}
	before := len(f.s.integ.subscription(sampleSubscription).history)
	for _, c := range []struct {
		step, command, target string
		who                   actor
	}{
		{"plugins-approve", pluginApproveKind, pluginProposalAt(8), root},
		{"plugins-withdraw", pluginWithdrawKind, pluginProposalAt(9), root},
		{"plugins-propose", pluginProposeKind, samplePluginAt, root},
		{"plugins-installation", pluginStopKind, samplePluginAt, root},
		{"plugins-stop", pluginStopKind, samplePluginAt, root},
		{"plugins-disable", pluginDisableKind, samplePluginAt, root},
		{"webhooks-propose", subProposeKind, webhooksTarget, root},
		{"webhooks-subscription", subProposeKind, webhooksTarget, root},
		{"webhooks-subscription", subPauseKind, subscriptionAt(sampleSubscription), root},
		{"webhooks-approve", subApproveKind, subscriptionAt(samplePayments), root},
		{"webhooks-pause", subPauseKind, subscriptionAt(sampleSubscription), root},
		{"webhooks-request_resume", subRequestKind, subscriptionAt(sampleSubscription), root},
		{"webhooks-approve_resume", subResumeKind, subscriptionAt(sampleSubscription), root},
		{"webhooks-revoke", subRevokeKind, subscriptionAt(sampleSubscription), root},
		{"webhooks-request_redeliver", letterRedeliverKind, letterAt(sampleLetter), root},
		{"webhooks-approve_redeliver", letterAgainKind, letterAt(sampleLetter), second},
		{"webhooks-approve_dismiss", letterDropKind, letterAt(sampleLetter), second},
	} {
		at = c.step
		f.code(t, c.step+" "+c.command, f.run(c.who, integrationCommand+c.command, c.target, fieldsFor(c.command)), 500, "internal_error")
	}
	at = ""
	f.s.fault = nil
	if len(f.s.integ.redeliver) != 0 || len(f.s.integ.subscription(sampleSubscription).history) != before || f.s.integ.counter != 0 || len(f.s.integ.entries) != 10 {
		t.Errorf("faults changed the store")
	}
	read := f.integrations(root)
	if s := fmt.Sprint(pluginProposal(t, read, 8)["state"], pluginProposal(t, read, 9)["state"], installedRow(t, read)["state"], subscriptionRowFor(t, read, sampleSubscription)["state"]); s !=
		"pendingpendingenabledactive" {
		t.Errorf("faults changed state %s", s)
	}
}

func seedOnly(t *testing.T) *Server {
	t.Helper()
	s := freshInstaller(t)
	s.bootWorkspace = SampleWorkspace
	return s
}

func TestIntegrationSeed(t *testing.T) {
	s := seedOnly(t)
	if err := s.seedIntegrations(s.clock()); err != nil {
		t.Fatal(err)
	}
	c := s.integ
	if len(c.entries) != 10 || strings.Join(c.publishers, ",") != samplePublisher || len(c.plugins) != 1 || strings.Join(c.plugins[0].versions, ",") != "1.0.0,1.1.0,1.2.0" ||
		c.workspace != SampleWorkspace || len(c.subscriptions) != 2 || fmt.Sprint(c.subscriptions[0].letters) != "[delivery-3]" || len(c.types) != 2 ||
		fmt.Sprint(c.redeliver) != "map[delivery-3:{sample-admin Sample data}]" || len(c.dismiss) != 0 || s.signer.keys != nil {
		t.Fatalf("seed store %+v", c)
	}
	if err := s.seedIntegrations(s.clock()); !errors.Is(err, events.ErrConflict) {
		t.Errorf("second seed %v", err)
	}
	failures := map[string]error{"events-propose": events.ErrInvalid, "events-approve": events.ErrInvalid, "plugins-propose": plugins.ErrInvalid,
		"plugins-approve": plugins.ErrInvalid, "webhooks-propose": webhooks.ErrInvalid, "webhooks-approve": webhooks.ErrInvalid, "webhooks-propose_key": webhooks.ErrInvalid,
		"webhooks-approve_key": webhooks.ErrInvalid, "events-publish": events.ErrInvalid, "webhooks-dispatch": webhooks.ErrInvalid, "webhooks-confirm": webhooks.ErrInvalid,
		"webhooks-report": webhooks.ErrInvalid, "webhooks-request_redeliver": webhooks.ErrInvalid, "events-read": errSeedLease, "events-bounds": webhooks.ErrRejected}
	for step, want := range failures {
		s := seedOnly(t)
		armed := false
		s.fault = func(name string) bool {
			armed = armed || s.signer.keys != nil
			return name == step
		}
		if err := s.seedIntegrations(s.clock()); !errors.Is(err, want) {
			t.Errorf("%s: %v", step, err)
		}
		if s.signer.keys != nil {
			t.Errorf("%s left the signer armed (armed during the seed: %v)", step, armed)
		}
	}
	paused := seedOnly(t)
	paused.fault = func(name string) bool {
		if name == "webhooks-dispatch" {
			paused.fault = nil
			if _, err := paused.webhooks.PauseSubscription(SampleWorkspace, sampleSubscription, sampleAdminWebhook); err != nil {
				t.Error(err)
			}
		}
		return false
	}
	if err := paused.seedIntegrations(paused.clock()); !errors.Is(err, errSeedLease) || paused.signer.keys != nil {
		t.Errorf("no lease %v", err)
	}
	late := seedOnly(t)
	proposals := 0
	late.fault = func(name string) bool {
		if name == "webhooks-propose" {
			proposals++
		}
		return name == "webhooks-propose" && proposals == 2
	}
	if err := late.seedIntegrations(late.clock()); !errors.Is(err, webhooks.ErrInvalid) || len(late.integ.subscriptions) != 1 {
		t.Errorf("payments proposal %v", err)
	}
	broken := freshInstaller(t)
	broken.fault = func(name string) bool { return name == "webhooks-confirm" }
	if err := broken.Start(); !errors.Is(err, webhooks.ErrInvalid) || broken.signer.keys != nil {
		t.Errorf("start with a failing seed %v", err)
	}
}

func TestIntegrationSeedSignerIsDisarmed(t *testing.T) {
	f := setup(t, SeedSample)
	var held ed25519.PrivateKey
	f.s.fault = func(step string) bool {
		if step == "webhooks-approve_key" {
			for _, key := range f.s.signer.keys {
				held = key
			}
		}
		return false
	}
	f = f.ready()
	f.s.fault = nil
	if len(held) != ed25519.PrivateKeySize || f.s.signer.keys != nil {
		t.Fatalf("held %d keys %v", len(held), f.s.signer.keys)
	}
	for _, b := range held {
		if b != 0 {
			t.Fatal("the held key was not zeroed")
		}
	}
	if _, err := f.s.signer.Sign(SampleWorkspace, sampleSubscription, sampleWebhookKey, []byte("m")); !errors.Is(err, errDisarmed) {
		t.Errorf("sign after the seed %v", err)
	}
	if version, err := f.s.signer.Current(SampleWorkspace, sampleSubscription); err != nil || version != sampleWebhookKey {
		t.Errorf("current %q %v", version, err)
	}
	public, _, _ := ed25519.GenerateKey(rand.Reader)
	k := webhooks.Key{Version: "sample-webhook-key-2", Public: public, NotBefore: start, NotAfter: start.Add(sampleKeyLife)}
	if _, err := f.s.webhooks.ProposeKey(SampleWorkspace, sampleSubscription, k, sampleAdminWebhook); err != nil {
		t.Fatal(err)
	}
	var refusal webhooks.Refusal
	if _, err := f.s.webhooks.ApproveKey(SampleWorkspace, sampleSubscription, k.Version, sampleAdmin2Webhook); !errors.As(err, &refusal) || refusal.Reason != webhooks.ReasonDependency {
		t.Errorf("approve key %v", err)
	}
}

func TestSeedSignerScope(t *testing.T) {
	g := newSeedSigner()
	public, private, _ := ed25519.GenerateKey(rand.Reader)
	if _, err := g.Current(SampleWorkspace, sampleSubscription); !errors.Is(err, errDisarmed) {
		t.Errorf("current before arming %v", err)
	}
	g.arm(SampleWorkspace, sampleSubscription, sampleWebhookKey, private)
	signature, err := g.Sign(SampleWorkspace, sampleSubscription, sampleWebhookKey, []byte("m"))
	if err != nil || !ed25519.Verify(public, []byte("m"), signature) {
		t.Fatalf("armed sign %v", err)
	}
	for _, triple := range [][3]string{{"other", sampleSubscription, sampleWebhookKey}, {SampleWorkspace, samplePayments, sampleWebhookKey}, {SampleWorkspace, sampleSubscription, "other"}} {
		if _, err := g.Sign(triple[0], triple[1], triple[2], []byte("m")); !errors.Is(err, errDisarmed) {
			t.Errorf("signed for %v", triple)
		}
	}
	g.disarm()
	if g.keys != nil {
		t.Error("disarm kept the key map")
	}
	for _, b := range private {
		if b != 0 {
			t.Fatal("disarm kept key material")
		}
	}
	if _, err := g.Sign(SampleWorkspace, sampleSubscription, sampleWebhookKey, []byte("m")); !errors.Is(err, errDisarmed) {
		t.Errorf("sign after disarm %v", err)
	}
}

func TestIntegrationStandIns(t *testing.T) {
	f := setup(t, SeedSample).ready()
	d, p := integrationDestinations{s: f.s}, integrationPermits{s: f.s}
	for _, c := range []struct {
		workspace, destination string
		want                   bool
	}{{SampleWorkspace, sampleEgress, true}, {SampleWorkspace, "sample-erp-webhook", false}, {"other", sampleEgress, false}} {
		if got, err := d.Active(c.workspace, c.destination); got != c.want || err != nil {
			t.Errorf("active %v %v", c, err)
		}
	}
	for destination, want := range map[string]string{"sample-erp-webhook": "active", "sample-legacy-webhook": "inactive", "unknown": "inactive"} {
		if got, err := d.State(SampleWorkspace, destination); got != want || err != nil {
			t.Errorf("state %s %s %v", destination, got, err)
		}
	}
	if got, _ := d.State("other", "sample-erp-webhook"); got != webhooks.DestinationInactive {
		t.Errorf("other workspace %s", got)
	}
	event := events.Stored{Event: events.Event{Type: typeInvoice}}
	for _, c := range []struct {
		workspace, subscriber string
		want                  bool
	}{{SampleWorkspace, sampleSubscriber, true}, {"other", sampleSubscriber, false}, {SampleWorkspace, "other", false}} {
		if got, err := p.Allowed(c.workspace, c.subscriber, event); got != c.want || err != nil {
			t.Errorf("allowed %v %v", c, err)
		}
	}
	if got, _ := p.Allowed(SampleWorkspace, sampleSubscriber, events.Stored{Event: events.Event{Type: "invoice.voided"}}); got {
		t.Error("unapproved type allowed")
	}
	f.s.fault = func(step string) bool { return step == "destinations" || step == "permits" || step == "events-read" }
	if _, err := d.Active(SampleWorkspace, sampleEgress); !errors.Is(err, errStandIn) {
		t.Errorf("active fault %v", err)
	}
	if _, err := p.Allowed(SampleWorkspace, sampleSubscriber, event); !errors.Is(err, errStandIn) {
		t.Errorf("permits fault %v", err)
	}
	if _, err := newOutboxReader(f.s).Read(SampleWorkspace, 0, 1); !errors.Is(err, events.ErrInvalid) {
		t.Errorf("outbox fault %v", err)
	}
	f.s.fault = nil
	if page, err := newOutboxReader(f.s).Read(SampleWorkspace, 0, 10); err != nil || len(page.Events) != 3 || page.Head != 3 {
		t.Errorf("outbox %+v %v", page, err)
	}
	if _, _, _, err := (noRedirects{}).Check(SampleWorkspace, "sample-erp-webhook", "", 1, "https://example.invalid"); !errors.Is(err, errRedirect) {
		t.Errorf("redirect %v", err)
	}
}

func TestIntegrationStoreAndTargets(t *testing.T) {
	c := newIntegrationStore()
	c.keep(&pluginEntry{view: plugins.ProposalView{Number: 1, State: plugins.ProposalPending}})
	for n := uint64(2); n <= installKept+1; n++ {
		c.keep(&pluginEntry{view: plugins.ProposalView{Number: n, State: plugins.ProposalApplied}})
	}
	if len(c.entries) != installKept || c.entry(1) == nil || c.entry(2) != nil || c.entry(installKept+1) == nil {
		t.Errorf("kept %d", len(c.entries))
	}
	for index := range c.entries {
		c.entries[index].view.State = plugins.ProposalPending
	}
	c.keep(&pluginEntry{view: plugins.ProposalView{Number: 999, State: plugins.ProposalPending}})
	if len(c.entries) != installKept+1 {
		t.Errorf("pending proposals dropped %d", len(c.entries))
	}
	e := &subscriptionEntry{id: "s", letters: []string{"delivery-1", "delivery-2"}}
	c.subscriptions = []*subscriptionEntry{e}
	c.redeliver["delivery-1"], c.dismiss["delivery-1"] = letterRequest{by: "a"}, letterRequest{by: "b"}
	c.dropLetter(e, "delivery-9")
	c.dropLetter(e, "delivery-1")
	if fmt.Sprint(e.letters) != "[delivery-2]" || len(c.redeliver) != 0 || len(c.dismiss) != 0 || c.letter("delivery-1") != nil || c.letter("delivery-2") != e {
		t.Errorf("letters %v", e.letters)
	}
	f := setup(t, SeedSample).ready()
	for target, want := range map[string]uint64{pluginProposalAt(1): 1, pluginProposalAt(9): 9, pluginProposalAt(10): 10} {
		if e := f.s.pluginEntryOf(target); e == nil || e.view.Number != want {
			t.Errorf("%s: %v", target, e)
		}
	}
	for _, target := range []string{"", "plugin-proposal-", pluginProposalAt(0), pluginProposalAt(11), "plugin-proposal-+9", "plugin-proposal-09", "plugin-proposal-9x",
		"plugin-proposal-18446744073709551616", "x-plugin-proposal-9", "installation-proposal-9"} {
		if f.s.pluginEntryOf(target) != nil {
			t.Errorf("%s resolved", target)
		}
	}
	if f.s.pluginOf(samplePlugin) != nil || f.s.subscriptionEntryOf(sampleSubscription) != nil || f.s.letterEntryOf(sampleLetter) != nil || f.s.letterEntryOf("webhook-letter:") != nil {
		t.Error("unprefixed targets resolved")
	}
	for n := uint64(100); n < 160; n++ {
		f.s.integ.keep(&pluginEntry{view: plugins.ProposalView{Number: n, State: plugins.ProposalWithdrawn, Proposer: SampleAdmin, Change: plugins.Change{Kind: plugins.ChangeEnable}}})
	}
	rows := part(f.integrations(f.as(AdminLogin)), "modules")["proposals"].([]any)
	if len(rows) != installShown || fmt.Sprint(rows[0].(map[string]any)["number"]) != "159" || fmt.Sprint(rows[installShown-1].(map[string]any)["number"]) != "110" {
		t.Errorf("page %d", len(rows))
	}
}

func TestMapIntegrationErrors(t *testing.T) {
	cases := map[error]commands.Code{
		plugins.Refusal{Reason: plugins.ReasonSelfApproval}:          commands.PrerequisiteFailed,
		plugins.Refusal{Reason: plugins.ReasonNotProposer}:           commands.PrerequisiteFailed,
		webhooks.Refusal{Reason: webhooks.ReasonSelfApproval}:        commands.PrerequisiteFailed,
		webhooks.Refusal{Reason: webhooks.ReasonDestinationInactive}: commands.PrerequisiteFailed,
		webhooks.Refusal{Reason: webhooks.ReasonSubscribeRefused}:    commands.PrerequisiteFailed,
		webhooks.Refusal{Reason: webhooks.ReasonDependency}:          commands.DependencyUnavailable,
		plugins.Refusal{Reason: plugins.ReasonAINotAllowed}:          commands.InternalError,
		webhooks.Refusal{Reason: webhooks.ReasonNotHuman}:            commands.InternalError,
		plugins.Refusal{Reason: plugins.ReasonDirection}:             commands.InvalidStage,
		webhooks.Refusal{Reason: webhooks.ReasonNotActive}:           commands.InvalidStage,
		plugins.ErrNotFound:  commands.InvalidStage,
		webhooks.ErrNotFound: commands.InvalidStage,
		plugins.ErrConflict:  commands.InvalidStage,
		webhooks.ErrConflict: commands.InvalidStage,
		plugins.ErrStale:     commands.InvalidStage,
		webhooks.ErrStale:    commands.InvalidStage,
		plugins.ErrLimit:     commands.PrerequisiteFailed,
		webhooks.ErrLimit:    commands.PrerequisiteFailed,
		plugins.ErrInvalid:   commands.InternalError,
		webhooks.ErrInvalid:  commands.InternalError,
		errors.New("other"):  commands.InternalError,
	}
	for err, want := range cases {
		var failure *commands.Failure
		if !errors.As(mapIntegration(err), &failure) || failure.Code != want || len(failure.Fields) != 0 {
			t.Errorf("%v: %v", err, failure)
		}
	}
	if mapIntegration(nil) != nil {
		t.Error("nil mapped")
	}
	for err, want := range map[error]string{nil: "", plugins.Refusal{Reason: "x"}: "x", plugins.ErrNotFound: "not_found", plugins.ErrConflict: "conflict", plugins.ErrLimit: "limit"} {
		if got, rest := recheckReason(err); got != want || rest != nil {
			t.Errorf("%v: %q %v", err, got, rest)
		}
	}
	if _, rest := recheckReason(plugins.ErrInvalid); !errors.Is(rest, plugins.ErrInvalid) {
		t.Error("unexpected recheck error kept")
	}
}

func TestIntegrationCommandSet(t *testing.T) {
	f := setup(t, SeedSample).ready()
	ids := []string{}
	for id, mutating := range f.s.mutating {
		if strings.HasPrefix(id, integrationCommand) {
			if !mutating {
				t.Errorf("%s is not mutating", id)
			}
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	if strings.Join(ids, " ") != "integrations.letter_approve_dismiss integrations.letter_approve_redeliver integrations.letter_request_dismiss integrations.letter_request_redeliver "+
		"integrations.plugin_approve integrations.plugin_disable integrations.plugin_propose integrations.plugin_stop integrations.plugin_withdraw integrations.subscription_approve "+
		"integrations.subscription_approve_resume integrations.subscription_pause integrations.subscription_propose integrations.subscription_request_resume integrations.subscription_revoke" {
		t.Errorf("commands %v", ids)
	}
}

type sourceFile struct {
	name string
	file *ast.File
}

func serverSources(t *testing.T) []sourceFile {
	t.Helper()
	names, _ := filepath.Glob("*.go")
	out := []sourceFile{}
	for _, name := range names {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, sourceFile{name, file})
	}
	return out
}

func receiverName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return ""
	}
	if ident, ok := fn.Recv.List[0].Type.(*ast.Ident); ok {
		return ident.Name
	}
	return ""
}

func TestIntegrationActorsAndEffectsAreConfined(t *testing.T) {
	restricted := map[string]bool{"Dispatch": true, "Confirm": true, "Redirect": true, "IssueToken": true, "CheckToken": true, "CheckEndpoint": true, "CheckEgress": true,
		"ProposeKey": true, "ApproveKey": true, "RetireKey": true, "Publish": true, "Report": true}
	modules := map[string]bool{"plugins": true, "webhooks": true, "events": true}
	checked := 0
	for _, src := range serverSources(t) {
		for _, decl := range src.file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			ast.Inspect(fn, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				name := sel.Sel.Name
				if pkg, ok := sel.X.(*ast.Ident); ok && modules[pkg.Name] && (name == "KindSystem" || name == "KindAssistant") {
					checked++
					if fn.Name.Name != "seedIntegrations" && fn.Name.Name != "newOutboxReader" {
						t.Errorf("%s %s names %s.%s", src.name, fn.Name.Name, pkg.Name, name)
					}
				}
				if name == "system" && !(receiverName(fn) == "outboxReader" && (fn.Name.Name == "Read" || fn.Name.Name == "Bounds")) {
					t.Errorf("%s %s uses the outbox actor", src.name, fn.Name.Name)
				}
				if restricted[name] && fn.Name.Name != "seedIntegrations" && !(name == "Publish" && fn.Name.Name == "publishTrust") {
					owner, _ := sel.X.(*ast.SelectorExpr)
					if name != "Report" || owner == nil || owner.Sel.Name != "installer" && !(owner.Sel.Name == "recovery" && fn.Name.Name == "seedRecovery") &&
						!(owner.Sel.Name == "retention" && fn.Name.Name == "seedRetention") {
						t.Errorf("%s %s calls %s", src.name, fn.Name.Name, name)
					}
				}
				return true
			})
		}
		if src.name == "integrations.go" || src.name == "integrationcmds.go" || src.name == "integrationdeps.go" {
			ast.Inspect(src.file, func(n ast.Node) bool {
				if lit, ok := n.(*ast.BasicLit); ok && (strings.Contains(lit.Value, "system") || strings.Contains(lit.Value, "assistant")) {
					t.Errorf("%s has the literal %s", src.name, lit.Value)
				}
				return true
			})
		}
		if src.name == "integrationdeps.go" {
			for _, spec := range src.file.Imports {
				if path := strings.Trim(spec.Path.Value, `"`); path == "net" || strings.HasPrefix(path, "net/") || path == "crypto/tls" {
					t.Errorf("integrationdeps.go imports %s", path)
				}
			}
		}
	}
	if checked != 3 {
		t.Errorf("non-human actor kinds found %d times", checked)
	}
}

func TestIntegrationCommandsActAsTheHumanReader(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	f.ok(t, "approve", f.run(root, "integrations.plugin_approve", pluginProposalAt(8), nil), "applied", "")
	if v, err := f.s.plugins.Proposal(8); err != nil || v.Approver != SampleAdmin {
		t.Errorf("plugin approver %+v %v", v, err)
	}
	f.ok(t, "approve subscription", f.run(root, "integrations.subscription_approve", subscriptionAt(samplePayments), nil), "active", "")
	if v, err := f.s.webhooks.SubscriptionOf(SampleWorkspace, samplePayments); err != nil || v.Approver != SampleAdmin || v.Proposer != SampleAdmin2 {
		t.Errorf("subscription actors %+v %v", v, err)
	}
	f.ok(t, "request dismiss", f.run(second, "integrations.letter_request_dismiss", letterAt(sampleLetter), nil), "requested", "")
	f.ok(t, "approve dismiss", f.run(root, "integrations.letter_approve_dismiss", letterAt(sampleLetter), nil), "dismissed", "")
}

func TestIntegrationConcurrentReadsAndCommands(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				if got := f.integrations(second); got.status != 200 {
					t.Errorf("concurrent read %d", got.status)
				}
			}
		}()
	}
	for i := 0; i < 3; i++ {
		for _, c := range []struct{ command, target string }{{"plugin_stop", samplePluginAt}, {"plugin_disable", samplePluginAt}, {"subscription_pause", subscriptionAt(sampleSubscription)},
			{"subscription_revoke", subscriptionAt(samplePayments)}} {
			body := fmt.Sprintf(`{"request_id":"c%d%s","target_id":"%s","idempotency_key":"c%d%s","expected_revision":"%s","rationale":"x","body":{}}`, i, c.command, c.target, i, c.command,
				f.s.Revision())
			f.call(integrationCommand+c.command, body, root.cookie, root.csrf)
		}
	}
	wg.Wait()
}
