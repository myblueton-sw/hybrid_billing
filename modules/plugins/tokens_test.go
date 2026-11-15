package plugins

import (
	"crypto/sha256"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestTokens(t *testing.T) {
	f := running(t)
	scopes := []Capability{call, events}
	text, g, err := f.r.IssueToken("w1", "ledger", scopes, time.Minute, run1)
	f.must(err)
	if !strings.HasPrefix(text, tokenPrefix) || len(text) != 47 || g.Workspace != "w1" || g.Version != "1.1.0" || g.Digest != manifest("ledger", "1.1.0").Digest || len(g.Scopes) != 2 {
		t.Fatalf("issued %s %+v", text, g)
	}
	scopes[0] = screen
	checked, err := f.r.CheckToken(text, call)
	f.must(err)
	if checked.Revision != g.Revision || checked.Scopes[0] != call {
		t.Errorf("checked %+v", checked)
	}
	checked.Scopes[0] = screen
	if again, _ := f.r.CheckToken(text, call); again.Scopes[0] != call {
		t.Error("grant shares scopes")
	}
	reason(t, "scope", second(f.r.CheckToken(text, screen)), ReasonScope)
	f.r.mu.Lock()
	for hash := range f.r.tokens {
		for _, b := range [][]byte{hash[:]} {
			if strings.Contains(string(b), text) {
				t.Error("raw token stored")
			}
		}
		if hash == sha256.Sum256([]byte(text)) {
			t.Error("untagged hash stored")
		}
	}
	f.r.mu.Unlock()
	reason(t, "unknown", second(f.r.CheckToken(tokenPrefix+strings.Repeat("A", 43), call)), ReasonUnknownToken)
	expect(t, "admin form", second(f.r.CheckToken("admin-"+strings.Repeat("A", 43), call)), ErrInvalid)
	expect(t, "short", second(f.r.CheckToken(tokenPrefix+strings.Repeat("A", 42), call)), ErrInvalid)
	expect(t, "alphabet", second(f.r.CheckToken(tokenPrefix+strings.Repeat("+", 43), call)), ErrInvalid)
	expect(t, "non-canonical", second(f.r.CheckToken(tokenPrefix+strings.Repeat("A", 42)+"B", call)), ErrInvalid)
	expect(t, "scope form", second(f.r.CheckToken(text, Capability{Kind: "admin", Target: "x"})), ErrInvalid)
	expect(t, "scope target", second(f.r.CheckToken(text, Capability{Kind: CapabilityCallCommand})), ErrInvalid)
	f.advance(time.Minute - time.Nanosecond)
	f.must(second(f.r.CheckToken(text, call)))
	f.advance(time.Nanosecond)
	reason(t, "expired", second(f.r.CheckToken(text, call)), ReasonExpired)
	reason(t, "forgotten", second(f.r.CheckToken(text, call)), ReasonUnknownToken)
}

func TestIssueTokenRules(t *testing.T) {
	f := running(t)
	expect(t, "no scopes", third(f.r.IssueToken("w1", "ledger", nil, time.Minute, run1)), ErrInvalid)
	expect(t, "scope order", third(f.r.IssueToken("w1", "ledger", []Capability{events, call}, time.Minute, run1)), ErrInvalid)
	expect(t, "scope kind", third(f.r.IssueToken("w1", "ledger", []Capability{{Kind: "admin", Target: "x"}}, time.Minute, run1)), ErrInvalid)
	expect(t, "scope target", third(f.r.IssueToken("w1", "ledger", []Capability{{Kind: CapabilityReadEvents}}, time.Minute, run1)), ErrInvalid)
	expect(t, "many scopes", third(f.r.IssueToken("w1", "ledger", many(maxCapabilities+1), time.Minute, run1)), ErrInvalid)
	expect(t, "short", third(f.r.IssueToken("w1", "ledger", []Capability{events}, minTokenLife-1, run1)), ErrInvalid)
	expect(t, "long", third(f.r.IssueToken("w1", "ledger", []Capability{events}, time.Hour+1, run1)), ErrInvalid)
	f.must(third(f.r.IssueToken("w1", "ledger", []Capability{events}, minTokenLife, run1)))
	f.must(third(f.r.IssueToken("w1", "ledger", []Capability{events}, time.Hour, run1)))
	expect(t, "workspace form", third(f.r.IssueToken("", "ledger", []Capability{events}, time.Minute, run1)), ErrInvalid)
	expect(t, "plugin form", third(f.r.IssueToken("w1", "", []Capability{events}, time.Minute, run1)), ErrInvalid)
	expect(t, "actor form", third(f.r.IssueToken("w1", "ledger", []Capability{events}, time.Minute, Actor{ID: "run-1"})), ErrInvalid)
	expect(t, "missing workspace", third(f.r.IssueToken("w9", "ledger", []Capability{events}, time.Minute, run1)), ErrNotFound)
	reason(t, "foreign runtime", third(f.r.IssueToken("w2", "ledger", []Capability{events}, time.Minute, run1)), ReasonNotRuntime)
	reason(t, "human", third(f.r.IssueToken("w1", "ledger", []Capability{events}, time.Minute, Actor{ID: "run-1", Kind: KindHuman})), ReasonNotRuntime)
	expect(t, "missing installation", third(f.r.IssueToken("w1", "other", []Capability{events}, time.Minute, run1)), ErrNotFound)
	reason(t, "undeclared scope", third(f.r.IssueToken("w1", "ledger", []Capability{{Kind: CapabilityCallCommand, Target: "admin.grant"}}, time.Minute, run1)), ReasonScope)
	f.random.err = boom
	reason(t, "entropy", third(f.r.IssueToken("w1", "ledger", []Capability{events}, time.Minute, run1)), ReasonDependency)
	f.random.err = nil
	f.advance(time.Second)
	a := f.token("w1", run1, events)
	b := f.token("w1", run1, events)
	if a == b {
		t.Error("retry returned the same token")
	}
	f.must(second(f.r.Stop("w1", "ledger", ReasonTimeout, run1)))
	reason(t, "stopped", third(f.r.IssueToken("w1", "ledger", []Capability{events}, time.Minute, run1)), ReasonNotEnabled)
	f.must(second(f.r.MarkVulnerable("ledger", "1.1.0", "Flaw", ann)))
	f.r.mu.Lock()
	f.r.workspaces["w2"].installations["ledger"].state = StateEnabled
	f.r.mu.Unlock()
	reason(t, "not runnable", third(f.r.IssueToken("w2", "ledger", []Capability{events}, time.Minute, run2)), ReasonVulnerable)
}

func TestTokenCollision(t *testing.T) {
	f := running(t)
	f.random.repeat = true
	first := f.token("w1", run1, events)
	reason(t, "collision", third(f.r.IssueToken("w2", "ledger", []Capability{events}, time.Minute, run2)), ReasonDependency)
	g, err := f.r.CheckToken(first, events)
	if err != nil || g.Workspace != "w1" {
		t.Errorf("first token %+v %v", g, err)
	}
}

func TestTokenHashKnownAnswer(t *testing.T) {
	sum := digest("hbp_x")
	if hexString(sum[:]) != tokenAnswer {
		t.Errorf("token hash %s", hexString(sum[:]))
	}
}

func TestDefensiveChecks(t *testing.T) {
	f := running(t)
	text := f.token("w1", run1, events)
	other := f.token("w2", run2, events)
	f.r.mu.Lock()
	f.r.plugins["ledger"].versions["1.1.0"].vulnerable = &Mark{By: "ann", Reason: "Unswept"}
	f.r.mu.Unlock()
	reason(t, "unswept mark", second(f.r.CheckToken(text, events)), ReasonStaleToken)
	d := running(t)
	other = d.token("w2", run2, events)
	d.r.mu.Lock()
	d.r.workspaces["w2"].installations["ledger"].version = "1.0.0"
	d.r.mu.Unlock()
	reason(t, "version drift", second(d.r.CheckToken(other, events)), ReasonStaleToken)
	g := running(t)
	var once sync.Once
	g.dest.during = func() {
		once.Do(func() {
			g.r.mu.Lock()
			g.r.workspaces["w1"].installations["ledger"].state = StateStopped
			g.r.mu.Unlock()
		})
	}
	expect(t, "state drift", g.r.CheckEgress("w1", "ledger", "crm", run1), ErrStale)
	h := running(t)
	var twice sync.Once
	h.dest.during = func() {
		twice.Do(func() {
			h.r.mu.Lock()
			h.r.plugins["ledger"].versions["1.1.0"].vulnerable = &Mark{By: "ann", Reason: "Unswept"}
			h.r.mu.Unlock()
		})
	}
	expect(t, "runnable drift", h.r.CheckEgress("w1", "ledger", "crm", run1), ErrStale)
}

func TestFullScopes(t *testing.T) {
	f := standard(t)
	wide := manifest("wide", "1.0.0")
	wide.Capabilities, wide.Screens = many(maxCapabilities), nil
	f.version(wide)
	f.install("w1", "wide", "1.0.0")
	f.enable("w1", "wide")
	f.must(third(f.r.IssueToken("w1", "wide", many(maxCapabilities), time.Minute, run1)))
}

func TestTokenBound(t *testing.T) {
	f := running(t)
	first := f.token("w1", run1, events)
	f.token("w1", run1, events)
	f.token("w1", run1, events)
	expect(t, "bound", third(f.r.IssueToken("w1", "ledger", []Capability{events}, time.Minute, run1)), ErrLimit)
	f.token("w2", run2, events)
	f.must(second(f.r.CheckToken(first, events)))
	f.advance(time.Minute)
	f.token("w1", run1, events)
	f.r.mu.Lock()
	live := len(f.r.workspaces["w1"].installations["ledger"].tokens)
	f.r.mu.Unlock()
	if live != 1 {
		t.Errorf("pruned to %d", live)
	}
	f.must(second(f.r.Stop("w1", "ledger", ReasonOperator, ann)))
	f.enable("w1", "ledger")
	f.token("w1", run1, events)
	f.token("w1", run1, events)
	f.token("w1", run1, events)
}

func TestTokensFollowTheInstallation(t *testing.T) {
	steps := map[string]func(f world){
		"stop":     func(f world) { f.must(second(f.r.Stop("w1", "ledger", ReasonTimeout, run1))) },
		"disable":  func(f world) { f.must(second(f.r.Disable("w1", "ledger", ann))) },
		"upgrade":  func(f world) { f.approve(at(ChangeUpgrade, "w1", "ledger", "1.2.0")) },
		"rollback": func(f world) { f.approve(at(ChangeRollback, "w1", "ledger", "1.0.0")) },
		"runtime": func(f world) {
			f.approve(Change{Kind: ChangeRuntime, Workspace: Workspace{ID: "w1", Runtime: "run-new"}})
		},
		"mark":   func(f world) { f.must(second(f.r.MarkVulnerable("ledger", "1.1.0", "Flaw", ann))) },
		"revoke": func(f world) { f.must(second(f.r.RevokePublisher("acme", "Ended", ann))) },
	}
	for name, step := range steps {
		f := running(t)
		f.version(manifest("ledger", "1.2.0"))
		text := f.token("w1", run1, events)
		other := f.token("w2", run2, events)
		step(f)
		err := second(f.r.CheckToken(text, events))
		reason(t, name, err, ReasonStaleToken)
		if name != "mark" && name != "revoke" {
			if _, err := f.r.CheckToken(other, events); err != nil {
				t.Errorf("%s changed another workspace: %v", name, err)
			}
		}
	}
	f := running(t)
	text := f.token("w1", run1, events)
	f.approve(at(ChangeRemove, "w1", "ledger", ""))
	reason(t, "remove", second(f.r.CheckToken(text, events)), ReasonUnknownToken)
	f.install("w1", "ledger", "1.1.0")
	f.enable("w1", "ledger")
	reason(t, "reinstall", second(f.r.CheckToken(text, events)), ReasonUnknownToken)
	g := running(t)
	old := g.token("v1", runV, events)
	g.must(second(g.r.MarkVulnerable("ledger", "1.1.0", "Flaw", ann)))
	g.must(second(g.r.CheckToken(old, events)))
}

func TestEgress(t *testing.T) {
	f := running(t)
	f.must(f.r.CheckEgress("w1", "ledger", "crm", run1))
	expect(t, "form", f.r.CheckEgress("w1", "ledger", "", run1), ErrInvalid)
	reason(t, "undeclared", f.r.CheckEgress("w1", "ledger", "erp", run1), ReasonNotDeclared)
	reason(t, "foreign", f.r.CheckEgress("w1", "ledger", "crm", run2), ReasonNotRuntime)
	f.dest.active = false
	reason(t, "inactive", f.r.CheckEgress("w1", "ledger", "crm", run1), ReasonDestinationInactive)
	f.dest.active, f.dest.err = true, boom
	reason(t, "dependency", f.r.CheckEgress("w1", "ledger", "crm", run1), ReasonDependency)
	f.dest.err = nil
	var once sync.Once
	f.dest.during = func() {
		once.Do(func() { f.must(second(f.r.Stop("w1", "ledger", ReasonTimeout, run1))) })
	}
	expect(t, "stale", f.r.CheckEgress("w1", "ledger", "crm", run1), ErrStale)
	f.dest.during = nil
	reason(t, "stopped", f.r.CheckEgress("w1", "ledger", "crm", run1), ReasonNotEnabled)
	f.must(f.r.CheckEgress("w2", "ledger", "crm", run2))
}
