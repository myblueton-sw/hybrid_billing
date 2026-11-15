package trust

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNewRegistry(t *testing.T) {
	clock := func() time.Time { return start }
	if _, err := NewRegistry(limits, clock); err != nil {
		t.Fatal(err)
	}
	expect(t, "no clock", second(NewRegistry(limits, nil)), ErrInvalid)
	counts := map[string]struct {
		field func(*Limits) *int
		max   int
	}{
		"workspaces": {func(l *Limits) *int { return &l.Workspaces }, maxWorkspaces}, "connections": {func(l *Limits) *int { return &l.Connections }, maxConnections},
		"retired": {func(l *Limits) *int { return &l.Retired }, maxRetired}, "endpoints": {func(l *Limits) *int { return &l.Endpoints }, maxEndpoints},
		"keys": {func(l *Limits) *int { return &l.Keys }, maxKeys}, "destinations": {func(l *Limits) *int { return &l.Destinations }, maxDestinations},
		"quarantine": {func(l *Limits) *int { return &l.Quarantine }, maxQuarantine}, "allowlist": {func(l *Limits) *int { return &l.Allowlist }, maxAllowlist},
		"redirects": {func(l *Limits) *int { return &l.Redirects }, maxRedirects}, "body": {func(l *Limits) *int { return &l.Body }, maxBody},
	}
	for name, c := range counts {
		for value, ok := range map[int]bool{0: false, 1: true, c.max: true, c.max + 1: false} {
			l := limits
			*c.field(&l) = value
			if _, err := NewRegistry(l, clock); ok != (err == nil) {
				t.Errorf("%s %d: %v", name, value, err)
			}
		}
	}
	for value, ok := range map[int]bool{0: false, 1: true, maxEvents: true, maxEvents + 1: false} {
		l := limits
		l.Events, l.Items = value, 1
		if _, err := NewRegistry(l, clock); ok != (err == nil) {
			t.Errorf("events %d: %v", value, err)
		}
	}
	for value, ok := range map[int]bool{0: false, 1: true, maxItems: true, maxItems + 1: false} {
		l := limits
		l.Items, l.Events = value, maxEvents
		if _, err := NewRegistry(l, clock); ok != (err == nil) {
			t.Errorf("items %d: %v", value, err)
		}
	}
	l := limits
	l.Items, l.Events = 5, 4
	expect(t, "items above events", second(NewRegistry(l, clock)), ErrInvalid)
	l.Items = 4
	if _, err := NewRegistry(l, clock); err != nil {
		t.Errorf("items equal to events: %v", err)
	}
	for name, c := range map[string]struct {
		field    func(*Limits) *time.Duration
		min, max time.Duration
	}{
		"skew": {func(l *Limits) *time.Duration { return &l.Skew }, minSkew, maxSkew}, "key life": {func(l *Limits) *time.Duration { return &l.KeyLife }, minKeyLife, maxKeyLife},
	} {
		for value, ok := range map[time.Duration]bool{c.min - 1: false, c.min: true, c.max: true, c.max + 1: false} {
			l := limits
			*c.field(&l) = value
			if _, err := NewRegistry(l, clock); ok != (err == nil) {
				t.Errorf("%s %v: %v", name, value, err)
			}
		}
	}
	for _, id := range []string{"x", strings.Repeat("a", maxIDBytes), "A.z_0:-9", "Z", "z", "9"} {
		if !validID(id) {
			t.Errorf("%q is refused", id)
		}
	}
	for _, id := range []string{"", strings.Repeat("a", maxIDBytes+1), "a/b", "a b", "é", "a@", "[", "`", "{"} {
		if validID(id) {
			t.Errorf("%q is accepted", id)
		}
	}
	err := error(Refusal{Reason: ReasonStale})
	if err.Error() != ErrRejected.Error()+": stale" || !errors.Is(err, ErrRejected) {
		t.Errorf("refusal %v", err)
	}
}

func TestConnectionForms(t *testing.T) {
	f := setup(t)
	good := Connection{ID: "c1", Fingerprints: []Fingerprint{print("a")}, Endpoints: []string{"usage"}}
	change := func(edit func(*Connection)) Connection {
		c := good
		edit(&c)
		return c
	}
	for name, c := range map[string]Connection{
		"no identifier":          change(func(c *Connection) { c.ID = "" }),
		"no fingerprint":         change(func(c *Connection) { c.Fingerprints = nil }),
		"three fingerprints":     change(func(c *Connection) { c.Fingerprints = []Fingerprint{print("a"), print("b"), print("c")} }),
		"a zero fingerprint":     change(func(c *Connection) { c.Fingerprints = []Fingerprint{{}} }),
		"a repeated fingerprint": change(func(c *Connection) { c.Fingerprints = []Fingerprint{print("a"), print("a")} }),
		"no endpoint":            change(func(c *Connection) { c.Endpoints = nil }),
		"too many endpoints":     change(func(c *Connection) { c.Endpoints = []string{"a", "b", "c", "d"} }),
		"an endpoint as text":    change(func(c *Connection) { c.Endpoints = []string{"a b"} }),
		"a repeated endpoint":    change(func(c *Connection) { c.Endpoints = []string{"a", "a"} }),
	} {
		expect(t, name, second(f.r.ProposeConnection("w1", c, ann)), ErrInvalid)
	}
	expect(t, "no workspace", second(f.r.ProposeConnection("", good, ann)), ErrInvalid)
	expect(t, "an invalid actor", second(f.r.ProposeConnection("w1", good, Actor{ID: "ann", Kind: "robot"})), ErrInvalid)
	expect(t, "an empty actor", second(f.r.ProposeConnection("w1", good, Actor{Kind: KindHuman})), ErrInvalid)
	reason(t, "an assistant proposing", second(f.r.ProposeConnection("w1", good, bot)), ReasonAINotAllowed)
	reason(t, "a system proposing", second(f.r.ProposeConnection("w1", good, sweeper)), ReasonNotHuman)
	if _, err := f.r.ProposeConnection("w1", change(func(c *Connection) {
		c.Endpoints = []string{"a", "b", "c"}
		c.Fingerprints = []Fingerprint{print("a"), print("b")}
	}), ann); err != nil {
		t.Errorf("the maximum endpoints and fingerprints: %v", err)
	}
	for _, call := range []func() error{
		func() error { return second(f.approveConnection("", "c1", bob)) }, func() error { return second(f.approveConnection("w1", "", bob)) },
		func() error { return second(f.r.RevokeConnection("w1", "", bob)) }, func() error { return second(f.r.ConnectionOf("", "c1")) },
		func() error { return second(f.r.ConnectionOf("w1", "")) }, func() error { return second(f.approveConnection("w1", "c1", Actor{ID: "bob"})) },
	} {
		expect(t, "an invalid call", call(), ErrInvalid)
	}
	reason(t, "an assistant approving", second(f.approveConnection("w1", "c1", bot)), ReasonAINotAllowed)
	reason(t, "an assistant revoking", second(f.r.RevokeConnection("w1", "c1", bot)), ReasonAINotAllowed)
	expect(t, "an unknown workspace", second(f.approveConnection("w9", "c1", bob)), ErrNotFound)
	expect(t, "an unknown connection", second(f.approveConnection("w1", "c9", bob)), ErrNotFound)
	expect(t, "revoking an unknown connection", second(f.r.RevokeConnection("w1", "c9", bob)), ErrNotFound)
	expect(t, "reading an unknown connection", second(f.r.ConnectionOf("w1", "c9")), ErrNotFound)
}

func TestConnectionLifecycle(t *testing.T) {
	f := setup(t)
	c := Connection{ID: "c1", Fingerprints: []Fingerprint{print("a")}, Endpoints: []string{"usage", "cost"}}
	view, err := f.r.ProposeConnection("w1", c, ann)
	if err != nil || view.State != StateProposed || view.Revision != 0 || view.Pending == nil || view.Proposer != "ann" || len(view.Active.Fingerprints) != 0 {
		t.Fatalf("proposal %+v %v", view, err)
	}
	view.Pending.Endpoints[0] = "changed"
	c.Endpoints[0] = "changed-too"
	same := Connection{ID: "c1", Fingerprints: []Fingerprint{print("a")}, Endpoints: []string{"cost", "usage"}}
	if again, err := f.r.ProposeConnection("w1", same, ann); err != nil || strings.Join(again.Pending.Endpoints, ",") != "cost,usage" {
		t.Errorf("a repeated proposal %+v %v", again, err)
	}
	expect(t, "another proposer while pending", second(f.r.ProposeConnection("w1", same, cat)), ErrConflict)
	expect(t, "another fingerprint by another person", second(f.r.ProposeConnection("w1", Connection{ID: "c1", Fingerprints: []Fingerprint{print("a"), print("q")}, Endpoints: []string{"cost", "usage"}}, cat)), ErrConflict)
	for name, prints := range map[string][]Fingerprint{"a new fingerprint": {print("q")}, "one more fingerprint": {print("a"), print("q")}} {
		expect(t, name+" while pending", second(f.r.ProposeConnection("w1", Connection{ID: "c1", Fingerprints: prints, Endpoints: []string{"cost", "usage"}}, ann)), ErrConflict)
	}
	expect(t, "other endpoints while pending", second(f.r.ProposeConnection("w1", Connection{ID: "c1", Fingerprints: []Fingerprint{print("a")}, Endpoints: []string{"cost", "other"}}, ann)), ErrConflict)
	for _, call := range []func() error{
		func() error { return second(f.r.WithdrawConnection("", "c1", ann)) }, func() error { return second(f.r.WithdrawConnection("w1", "", ann)) },
	} {
		expect(t, "an invalid withdrawal", call(), ErrInvalid)
	}
	reason(t, "an assistant withdrawing", second(f.r.WithdrawConnection("w1", "c1", bot)), ReasonAINotAllowed)
	expect(t, "withdrawing an unknown connection", second(f.r.WithdrawConnection("w1", "c9", ann)), ErrNotFound)
	if withdrawn, err := f.r.WithdrawConnection("w1", "c1", cat); err != nil || withdrawn.Pending != nil || withdrawn.State != StateProposed {
		t.Errorf("a withdrawal %+v %v", withdrawn, err)
	}
	reason(t, "approving a withdrawn proposal", second(f.approveConnection("w1", "c1", bob)), ReasonNotPending)
	swapped, err := f.r.ProposeConnection("w1", Connection{ID: "c1", Fingerprints: []Fingerprint{print("q")}, Endpoints: []string{"usage"}}, ann)
	if err != nil || swapped.Proposal != 2 {
		t.Fatalf("a new proposal %+v %v", swapped, err)
	}
	reason(t, "approving the reviewed proposal after a swap", second(f.r.ApproveConnection("w1", "c1", 1, bob)), ReasonProposalChanged)
	expect(t, "no proposal number", second(f.r.ApproveConnection("w1", "c1", 0, bob)), ErrInvalid)
	f.must(second(f.r.WithdrawConnection("w1", "c1", ann)))
	f.must(second(f.r.ProposeConnection("w1", same, ann)))
	reason(t, "the proposer approving", second(f.approveConnection("w1", "c1", ann)), ReasonSelfApproval)
	if _, err := f.r.Admit(Presented{Fingerprint: print("a"), Workspace: "w1", Endpoint: "usage"}); err == nil {
		t.Errorf("a proposed connection admitted")
	}
	active, err := f.approveConnection("w1", "c1", bob)
	if err != nil || active.State != StateActive || active.Revision != 1 || active.Pending != nil || active.Approver != "bob" {
		t.Fatalf("approval %+v %v", active, err)
	}
	if again, err := f.approveConnection("w1", "c1", bob); err != nil || again.Revision != 1 {
		t.Errorf("the repeated approval %+v %v", again, err)
	}
	reason(t, "another approver without a proposal", second(f.approveConnection("w1", "c1", cat)), ReasonNotPending)
	if again, err := f.r.ProposeConnection("w1", same, cat); err != nil || again.Pending != nil || again.Revision != 1 {
		t.Errorf("a proposal equal to the active revision %+v %v", again, err)
	}
	admitted, err := f.r.Admit(Presented{Fingerprint: print("a"), Workspace: "w1", Endpoint: "cost"})
	if err != nil || admitted != (Admission{Workspace: "w1", Connection: "c1", Revision: 1}) {
		t.Errorf("admission %+v %v", admitted, err)
	}
	rotated := Connection{ID: "c1", Fingerprints: []Fingerprint{print("a"), print("b")}, Endpoints: []string{"usage"}}
	f.must(second(f.r.ProposeConnection("w1", rotated, cat)))
	if read, _ := f.r.ConnectionOf("w1", "c1"); read.Revision != 1 || read.Pending == nil || len(read.Active.Endpoints) != 2 {
		t.Errorf("a pending replacement changed the active revision %+v", read)
	}
	f.must(second(f.r.Admit(Presented{Fingerprint: print("a"), Workspace: "w1", Endpoint: "cost"})))
	reason(t, "the new fingerprint before approval", second(f.r.Admit(Presented{Fingerprint: print("b"), Workspace: "w1", Endpoint: "usage"})), ReasonUnknownCertificate)
	f.must(second(f.approveConnection("w1", "c1", ann)))
	for _, p := range []Fingerprint{print("a"), print("b")} {
		if got, err := f.r.Admit(Presented{Fingerprint: p, Workspace: "w1", Endpoint: "usage"}); err != nil || got.Revision != 2 {
			t.Errorf("rotation overlap %+v %v", got, err)
		}
	}
	reason(t, "a dropped endpoint", second(f.r.Admit(Presented{Fingerprint: print("a"), Workspace: "w1", Endpoint: "cost"})), ReasonEndpointNotRegistered)
	f.must(second(f.r.ProposeConnection("w1", Connection{ID: "c1", Fingerprints: []Fingerprint{print("b")}, Endpoints: []string{"usage"}}, ann)))
	f.must(second(f.approveConnection("w1", "c1", bob)))
	reason(t, "a replaced fingerprint", second(f.r.Admit(Presented{Fingerprint: print("a"), Workspace: "w1", Endpoint: "usage"})), ReasonUnknownCertificate)
	reason(t, "registering a replaced fingerprint", second(f.r.ProposeConnection("w1", Connection{ID: "c2", Fingerprints: []Fingerprint{print("a")}, Endpoints: []string{"usage"}}, ann)), ReasonFingerprintRetired)
	revoked, err := f.r.RevokeConnection("w1", "c1", cat)
	if err != nil || revoked.State != StateRevoked {
		t.Errorf("revocation %+v %v", revoked, err)
	}
	if again, err := f.r.RevokeConnection("w1", "c1", ann); err != nil || again.State != StateRevoked {
		t.Errorf("revoking again %+v %v", again, err)
	}
	reason(t, "a revoked certificate", second(f.r.Admit(Presented{Fingerprint: print("b"), Workspace: "w1", Endpoint: "usage"})), ReasonUnknownCertificate)
	reason(t, "reusing a revoked identifier", second(f.r.ProposeConnection("w1", Connection{ID: "c1", Fingerprints: []Fingerprint{print("c")}, Endpoints: []string{"usage"}}, ann)), ReasonRevoked)
	reason(t, "registering a revoked fingerprint", second(f.r.ProposeConnection("w2", Connection{ID: "c1", Fingerprints: []Fingerprint{print("b")}, Endpoints: []string{"usage"}}, ann)), ReasonFingerprintRetired)
	reason(t, "approving a revoked connection", second(f.approveConnection("w1", "c1", bob)), ReasonNotPending)
	f.must(second(f.r.ProposeConnection("w1", Connection{ID: "c3", Fingerprints: []Fingerprint{print("z")}, Endpoints: []string{"usage"}}, ann)))
	if gone, err := f.r.RevokeConnection("w1", "c3", ann); err != nil || gone.Pending != nil {
		t.Errorf("revoking a proposal %+v %v", gone, err)
	}
	f.must(second(f.r.ProposeConnection("w2", Connection{ID: "c9", Fingerprints: []Fingerprint{print("z")}, Endpoints: []string{"usage"}}, ann)))
}

func TestFingerprintsAcrossWorkspaces(t *testing.T) {
	f := setup(t)
	shared := Connection{ID: "c1", Fingerprints: []Fingerprint{print("a")}, Endpoints: []string{"usage"}}
	f.must(second(f.r.ProposeConnection("w1", shared, ann)))
	f.must(second(f.r.ProposeConnection("w2", shared, ann)))
	f.must(second(f.approveConnection("w2", "c1", bob)))
	reason(t, "the second approval of a shared fingerprint", second(f.approveConnection("w1", "c1", bob)), ReasonFingerprintInUse)
	reason(t, "a proposal of a used fingerprint", second(f.r.ProposeConnection("w3", shared, ann)), ReasonFingerprintInUse)
	reason(t, "another connection of the same workspace", second(f.r.ProposeConnection("w2", Connection{ID: "c2", Fingerprints: []Fingerprint{print("a")}, Endpoints: []string{"usage"}}, ann)), ReasonFingerprintInUse)
	reason(t, "a certificate of another workspace", second(f.r.Admit(Presented{Fingerprint: print("a"), Workspace: "w1", Endpoint: "usage"})), ReasonWorkspaceMismatch)
	reason(t, "an unknown certificate", second(f.r.Admit(Presented{Fingerprint: print("q"), Workspace: "w1", Endpoint: "usage"})), ReasonUnknownCertificate)
	reason(t, "an unregistered endpoint", second(f.r.Admit(Presented{Fingerprint: print("a"), Workspace: "w2", Endpoint: "cost"})), ReasonEndpointNotRegistered)
	for name, p := range map[string]Presented{"no fingerprint": {Workspace: "w1", Endpoint: "usage"}, "no workspace": {Fingerprint: print("a"), Endpoint: "usage"},
		"no endpoint": {Fingerprint: print("a"), Workspace: "w1"}} {
		expect(t, name, second(f.r.Admit(p)), ErrInvalid)
	}
	counts := f.r.AdmissionRefusals()
	if counts[ReasonWorkspaceMismatch] != 1 || counts[ReasonUnknownCertificate] != 1 || counts[ReasonEndpointNotRegistered] != 1 || len(counts) != 3 {
		t.Errorf("admission counts %+v", counts)
	}
	counts[ReasonStale] = 9
	if len(f.r.AdmissionRefusals()) != 3 {
		t.Errorf("the counts are shared with the caller")
	}
}

func TestConnectionRoom(t *testing.T) {
	l := limits
	l.Connections, l.Workspaces, l.Retired = 1, 2, 3
	f := setupWith(t, l)
	f.connect("w1", "c1", print("a"), print("b"))
	expect(t, "a second connection", second(f.r.ProposeConnection("w1", Connection{ID: "c2", Fingerprints: []Fingerprint{print("c")}, Endpoints: []string{"usage"}}, ann)), ErrLimit)
	f.must(second(f.r.ProposeConnection("w2", Connection{ID: "c9", Fingerprints: []Fingerprint{print("z")}, Endpoints: []string{"usage"}}, ann)))
	expect(t, "a third workspace", second(f.r.ProposeConnection("w3", Connection{ID: "c2", Fingerprints: []Fingerprint{print("c")}, Endpoints: []string{"usage"}}, ann)), ErrLimit)
	f.must(second(f.r.ProposeConnection("w1", Connection{ID: "c1", Fingerprints: []Fingerprint{print("c")}, Endpoints: []string{"usage"}}, ann)))
	f.must(second(f.approveConnection("w1", "c1", bob)))
	expect(t, "an approval without a reserved tombstone", second(f.approveConnection("w2", "c9", bob)), ErrLimit)
	f.must(second(f.r.ProposeConnection("w1", Connection{ID: "c1", Fingerprints: []Fingerprint{print("c"), print("d")}, Endpoints: []string{"usage"}}, ann)))
	expect(t, "a rotation without a reserved tombstone", second(f.approveConnection("w1", "c1", bob)), ErrLimit)
	f.must(second(f.r.WithdrawConnection("w1", "c1", ann)))
	f.must(second(f.r.ProposeConnection("w1", Connection{ID: "c1", Fingerprints: []Fingerprint{print("c")}, Endpoints: []string{"cost"}}, ann)))
	if done, err := f.approveConnection("w1", "c1", bob); err != nil || done.Revision != 3 {
		t.Errorf("a change that keeps the fingerprints at the bound %+v %v", done, err)
	}
	if gone, err := f.r.RevokeConnection("w1", "c1", bob); err != nil || gone.State != StateRevoked {
		t.Errorf("a revocation at the bound %+v %v", gone, err)
	}
	reason(t, "a revoked certificate at the bound", second(f.r.Admit(Presented{Fingerprint: print("c"), Workspace: "w1", Endpoint: "cost"})), ReasonUnknownCertificate)
	if len(f.r.retired) != 3 || len(f.r.owners) != 0 {
		t.Errorf("retired %d owners %d", len(f.r.retired), len(f.r.owners))
	}
}
