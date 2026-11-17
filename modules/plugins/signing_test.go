package plugins

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"testing"
	"time"
)

func TestSigningWrappersMatchTheRules(t *testing.T) {
	f := standard(t)
	public, private, err := ed25519.GenerateKey(rand.Reader)
	f.must(err)
	if !bytes.Equal(KeyChallenge("acme", "k2"), challenge("acme", "k2")) {
		t.Error("challenge wrapper")
	}
	f.approve(Change{Kind: ChangeKey, Key: PublisherKey{Publisher: "acme", Version: "k2", Public: public, NotBefore: start, NotAfter: start.Add(24 * time.Hour),
		Proof: ed25519.Sign(private, KeyChallenge("acme", "k2"))}})
	m := manifest("audit", "1.0.0")
	m.Key = "k2"
	if !bytes.Equal(m.SigningMessage(), m.encode()) {
		t.Error("manifest wrapper")
	}
	f.approve(Change{Kind: ChangeVersion, Manifest: m, Signature: ed25519.Sign(private, m.SigningMessage())})
	if v, err := f.r.Version("audit", "1.0.0"); err != nil || v.Manifest.Key != "k2" {
		t.Errorf("signed version %+v %v", v, err)
	}
	other := manifest("audit", "1.1.0")
	other.Key = "k2"
	reason(t, "signature over another manifest", f.try(Change{Kind: ChangeVersion, Manifest: other, Signature: ed25519.Sign(private, m.SigningMessage())}), ReasonSignatureInvalid)
	reason(t, "proof over another version", f.try(Change{Kind: ChangeKey, Key: PublisherKey{Publisher: "acme", Version: "k3", Public: public, NotBefore: start,
		NotAfter: start.Add(time.Hour), Proof: ed25519.Sign(private, KeyChallenge("acme", "k2"))}}), ReasonKeyMismatch)
}

func (f world) snapshot(number uint64) string {
	view, err := f.r.Proposal(number)
	out := fmt.Sprintf("%+v %v", view, err)
	for _, w := range []string{"w1", "w2", "v1"} {
		for _, p := range []string{"ledger", "audit"} {
			i, err := f.r.Installation(w, p)
			out += fmt.Sprintf(" %+v %v", i, err)
		}
	}
	return out
}

func (f world) agrees(name string, number uint64, approver Actor, want string) {
	f.t.Helper()
	before := f.snapshot(number)
	got := f.r.Recheck(number)
	if after := f.snapshot(number); after != before {
		f.t.Errorf("%s: recheck changed state\n%s\n%s", name, before, after)
	}
	if fmt.Sprint(got) != want {
		f.t.Errorf("%s: recheck %v; want %s", name, got, want)
	}
	_, approved := f.r.Approve(number, approver)
	if fmt.Sprint(approved) != fmt.Sprint(got) {
		f.t.Errorf("%s: approve %v; recheck %v", name, approved, got)
	}
}

func (f world) pending(c Change, by Actor) uint64 {
	f.t.Helper()
	p, err := f.r.Propose(c, by)
	f.must(err)
	return p.Number
}

func refusal(reason string) string { return refuse(reason).Error() }

func TestRecheckMatchesApprove(t *testing.T) {
	f := running(t)
	f.agrees("valid", f.pending(at(ChangeRollback, "w1", "ledger", "1.0.0"), ann), bob, "<nil>")
	f.agrees("unknown", 999, bob, ErrNotFound.Error())
	applied := f.approve(at(ChangeRollback, "w2", "ledger", "1.0.0")).Number
	f.agrees("applied", applied, cat, refusal(ReasonNotPending))
	g := standard(t)
	g.install("w1", "ledger", "1.0.0")
	a := g.pending(at(ChangeUpgrade, "w1", "ledger", "1.1.0"), ann)
	b := g.pending(at(ChangeUpgrade, "w1", "ledger", "1.1.0"), cat)
	g.must(second(g.r.Approve(a, bob)))
	g.agrees("direction", b, bob, refusal(ReasonDirection))
	g.must(second(g.r.Disable("w1", "ledger", ann)))
	enable := g.pending(at(ChangeEnable, "w1", "ledger", ""), ann)
	again := g.pending(at(ChangeEnable, "w1", "ledger", ""), cat)
	g.must(second(g.r.Approve(enable, bob)))
	g.agrees("not allowed state", again, bob, refusal(ReasonNotAllowedState))
	other := manifest("audit", "1.0.0")
	other.Storage.Namespace = "ledger-data"
	other.Screens[0].Fields[0].Source = "ledger-data"
	g.version(other)
	taken := g.pending(at(ChangeInstall, "w2", "audit", "1.0.0"), ann)
	g.install("w2", "ledger", "1.0.0")
	g.agrees("namespace taken", taken, bob, refusal(ReasonNamespaceTaken))
	h := standard(t)
	short := manifest("ledger", "1.2.0")
	short.Storage.RetentionDays = 10
	short.Endpoints = nil
	h.version(short)
	low := manifest("ledger", "0.9.0")
	low.Storage.RetentionDays = 10
	h.version(low)
	h.install("w1", "ledger", "0.9.0")
	shorter := h.pending(at(ChangeUpgrade, "w1", "ledger", "1.2.0"), ann)
	endpoint := h.pending(Change{Kind: ChangeEndpoint, Workspace: Workspace{ID: "w1"}, Plugin: "ledger", Endpoint: Endpoint{ID: "api", Fingerprint: fingerprint("a"),
		Evidence: fingerprint("e")}}, cat)
	h.approve(at(ChangeUpgrade, "w1", "ledger", "1.1.0"))
	h.agrees("storage changed", shorter, bob, refusal(ReasonStorageChanged))
	h.agrees("declared endpoint", endpoint, bob, "<nil>")
	k := standard(t)
	k.install("w1", "ledger", "1.0.0")
	undeclared := k.pending(Change{Kind: ChangeEndpoint, Workspace: Workspace{ID: "w1"}, Plugin: "ledger", Endpoint: Endpoint{ID: "api", Fingerprint: fingerprint("c"),
		Evidence: fingerprint("e")}}, ann)
	none := manifest("ledger", "1.2.0")
	none.Endpoints = nil
	k.version(none)
	k.approve(at(ChangeUpgrade, "w1", "ledger", "1.2.0"))
	k.agrees("not declared", undeclared, bob, refusal(ReasonNotDeclared))
	late := k.pending(signed(manifest("ledger", "1.3.0")), ann)
	k.must(second(k.r.RetireKey("acme", "k1", ann)))
	k.agrees("key inactive", late, bob, refusal(ReasonKeyInactive))
	m := standard(t)
	vulnerable := m.pending(at(ChangeInstall, "w1", "ledger", "1.0.0"), ann)
	m.must(second(m.r.MarkVulnerable("ledger", "1.0.0", "synthetic finding", ann)))
	m.agrees("vulnerable", vulnerable, bob, refusal(ReasonVulnerable))
	revoked := m.pending(at(ChangeInstall, "w1", "ledger", "1.1.0"), ann)
	m.must(second(m.r.RevokePublisher("acme", "synthetic revocation", ann)))
	m.agrees("revoked", revoked, bob, refusal(ReasonRevoked))
	n := standard(t)
	beta := n.pending(Change{Kind: ChangePublisher, Publisher: Publisher{ID: "beta", Name: "Beta"}}, ann)
	twin := n.pending(Change{Kind: ChangePublisher, Publisher: Publisher{ID: "beta", Name: "Beta"}}, cat)
	gamma := n.pending(Change{Kind: ChangePublisher, Publisher: Publisher{ID: "gamma", Name: "Gamma"}}, ann)
	delta := n.pending(Change{Kind: ChangePublisher, Publisher: Publisher{ID: "delta", Name: "Delta"}}, cat)
	n.must(second(n.r.Approve(beta, bob)))
	n.agrees("conflict", twin, bob, ErrConflict.Error())
	n.must(second(n.r.Approve(gamma, bob)))
	n.agrees("limit", delta, bob, ErrLimit.Error())
	withdrawn := n.pending(Change{Kind: ChangeWorkspace, Workspace: Workspace{ID: "w9", Runtime: "run-9"}}, ann)
	n.must(second(n.r.Withdraw(withdrawn, ann)))
	n.agrees("withdrawn", withdrawn, bob, ErrNotFound.Error())
}
