package plugins

import (
	"crypto/ed25519"
	"strings"
	"testing"
	"time"
)

func TestProposalFlow(t *testing.T) {
	f := setup(t)
	c := Change{Kind: ChangePublisher, Publisher: Publisher{ID: "acme", Name: "Acme"}, Plugin: "ignored"}
	reason(t, "assistant", second(f.r.Propose(c, bot)), ReasonAINotAllowed)
	reason(t, "system", second(f.r.Propose(c, run1)), ReasonNotHuman)
	expect(t, "actor", second(f.r.Propose(c, Actor{ID: "x", Kind: "robot"})), ErrInvalid)
	expect(t, "actor id", second(f.r.Propose(c, Actor{Kind: KindHuman})), ErrInvalid)
	expect(t, "kind", second(f.r.Propose(Change{Kind: "admin"}, ann)), ErrInvalid)
	p, err := f.r.Propose(c, ann)
	f.must(err)
	if p.Number != 1 || p.State != ProposalPending || p.Proposer != "ann" || p.Change.Plugin != "" {
		t.Fatalf("proposal %+v", p)
	}
	again, err := f.r.Propose(c, ann)
	f.must(err)
	if again.Number != p.Number {
		t.Errorf("repeat numbered %d", again.Number)
	}
	other, err := f.r.Propose(c, cat)
	f.must(err)
	if other.Number == p.Number {
		t.Error("another proposer shared a proposal")
	}
	reason(t, "self", second(f.r.Approve(p.Number, ann)), ReasonSelfApproval)
	reason(t, "approve assistant", second(f.r.Approve(p.Number, bot)), ReasonAINotAllowed)
	expect(t, "approve missing", second(f.r.Approve(99, bob)), ErrNotFound)
	done, err := f.r.Approve(p.Number, bob)
	f.must(err)
	if done.State != ProposalApplied || done.Approver != "bob" {
		t.Fatalf("approved %+v", done)
	}
	if again, err := f.r.Approve(p.Number, bob); err != nil || again.State != ProposalApplied {
		t.Errorf("repeat approval %+v %v", again, err)
	}
	reason(t, "other approver", second(f.r.Approve(p.Number, cat)), ReasonNotPending)
	expect(t, "stale duplicate", second(f.r.Approve(other.Number, bob)), ErrConflict)
	reason(t, "withdraw applied", second(f.r.Withdraw(p.Number, ann)), ReasonNotPending)
	reason(t, "withdraw other", second(f.r.Withdraw(other.Number, ann)), ReasonNotProposer)
	reason(t, "withdraw assistant", second(f.r.Withdraw(other.Number, bot)), ReasonAINotAllowed)
	expect(t, "withdraw missing", second(f.r.Withdraw(99, cat)), ErrNotFound)
	gone, err := f.r.Withdraw(other.Number, cat)
	f.must(err)
	if gone.State != ProposalWithdrawn {
		t.Errorf("withdrawn %+v", gone)
	}
	expect(t, "approve withdrawn", second(f.r.Approve(other.Number, bob)), ErrNotFound)
	expect(t, "view withdrawn", second(f.r.Proposal(other.Number)), ErrNotFound)
	view, err := f.r.Proposal(p.Number)
	f.must(err)
	if view.Approver != "bob" {
		t.Errorf("view %+v", view)
	}
	n := Change{Kind: ChangePublisher, Publisher: Publisher{ID: "beta", Name: "Beta"}}
	first, _ := f.r.Propose(n, ann)
	f.must(second(f.r.Withdraw(first.Number, ann)))
	next, _ := f.r.Propose(n, ann)
	if next.Number == first.Number {
		t.Error("re-proposal reused a number")
	}
	expect(t, "unregistered workspace", second(f.r.Propose(at(ChangeEnable, "w9", "ledger", ""), ann)), ErrNotFound)
}

func TestProposalCaps(t *testing.T) {
	l := limits
	l.Proposals, l.Publishers = 4, 10
	f := setupWith(t, l)
	f.workspace("w1", "run-1", false)
	pub := func(id string, actor Actor) error {
		return second(f.r.Propose(Change{Kind: ChangePublisher, Publisher: Publisher{ID: id, Name: "N"}}, actor))
	}
	f.must(pub("a", ann))
	expect(t, "per proposer", pub("b", ann), ErrLimit)
	f.must(pub("b", bob))
	f.must(pub("c", cat))
	f.must(pub("d", Actor{ID: "dan", Kind: KindHuman}))
	expect(t, "scope", pub("e", Actor{ID: "eve", Kind: KindHuman}), ErrLimit)
	f.must(second(f.r.Propose(Change{Kind: ChangeRuntime, Workspace: Workspace{ID: "w1", Runtime: "run-x"}}, ann)))
	l.Proposals = 1
	g := setupWith(t, l)
	g.must(second(g.r.Propose(Change{Kind: ChangePublisher, Publisher: Publisher{ID: "a", Name: "N"}}, ann)))
	expect(t, "single", second(g.r.Propose(Change{Kind: ChangePublisher, Publisher: Publisher{ID: "b", Name: "N"}}, bob)), ErrLimit)
}

func TestAppliedHistoryBound(t *testing.T) {
	l := limits
	l.Proposals, l.Publishers = 2, 5
	f := setupWith(t, l)
	var numbers []uint64
	for _, id := range []string{"a", "b", "c", "d"} {
		numbers = append(numbers, f.approve(Change{Kind: ChangePublisher, Publisher: Publisher{ID: id, Name: "N"}}).Number)
	}
	expect(t, "oldest pruned", second(f.r.Proposal(numbers[0])), ErrNotFound)
	expect(t, "next pruned", second(f.r.Proposal(numbers[1])), ErrNotFound)
	f.must(second(f.r.Proposal(numbers[2])))
	f.must(second(f.r.Proposal(numbers[3])))
}

func TestPublishersAndKeys(t *testing.T) {
	l := limits
	l.Publishers = 1
	f := setupWith(t, l)
	expect(t, "name", f.try(Change{Kind: ChangePublisher, Publisher: Publisher{ID: "acme", Name: ""}}), ErrInvalid)
	expect(t, "id", f.try(Change{Kind: ChangePublisher, Publisher: Publisher{ID: "a b", Name: "A"}}), ErrInvalid)
	f.approve(Change{Kind: ChangePublisher, Publisher: Publisher{ID: "acme", Name: "Acme"}})
	expect(t, "twice", f.try(Change{Kind: ChangePublisher, Publisher: Publisher{ID: "acme", Name: "Acme"}}), ErrConflict)
	expect(t, "room", f.try(Change{Kind: ChangePublisher, Publisher: Publisher{ID: "beta", Name: "Beta"}}), ErrLimit)
	k := keyChange("acme", "k1")
	forms := map[string]func(*PublisherKey){
		"publisher": func(k *PublisherKey) { k.Publisher = "" },
		"version":   func(k *PublisherKey) { k.Version = "" },
		"zero":      func(k *PublisherKey) { k.NotBefore = time.Time{} },
		"order":     func(k *PublisherKey) { k.NotAfter = k.NotBefore },
		"proof":     func(k *PublisherKey) { k.Proof = k.Proof[:10] },
	}
	for name, change := range forms {
		c := copyChange(k)
		change(&c.Key)
		expect(t, name, f.try(c), ErrInvalid)
	}
	missing := keyChange("beta", "k1")
	expect(t, "unknown publisher", f.try(missing), ErrNotFound)
	weakKey := copyChange(k)
	weakKey.Key.Public = hexBytes(smallOrder[0])
	reason(t, "weak", f.try(weakKey), ReasonWeakKey)
	long := copyChange(k)
	long.Key.NotAfter = long.Key.NotBefore.Add(l.KeyLife + time.Second)
	reason(t, "life", f.try(long), ReasonKeyLife)
	edge := keyChange("acme", "k9")
	edge.Key.NotAfter = edge.Key.NotBefore.Add(l.KeyLife)
	wrong := copyChange(k)
	wrong.Key.Proof = ed25519.Sign(private("other"), challenge("acme", "k1"))
	reason(t, "proof", f.try(wrong), ReasonKeyMismatch)
	f.approve(k)
	expect(t, "name reuse", f.try(k), ErrConflict)
	f.approve(edge)
	expect(t, "active keys", f.try(keyChange("acme", "k3")), ErrLimit)
	f.must(second(f.r.RetireKey("acme", "k9", ann)))
	f.approve(keyChange("acme", "k3"))
	view, err := f.r.Publisher("acme")
	f.must(err)
	if len(view.Keys) != 3 || view.Keys[0].Version != "k1" || !view.Keys[2].Retired || view.Revocation != nil || !same(view.Keys[0].Public, k.Key.Public) {
		t.Fatalf("publisher %+v", view)
	}
	view.Keys[0].Public[0]++
	again, _ := f.r.Publisher("acme")
	if !same(again.Keys[0].Public, k.Key.Public) {
		t.Error("view shares key bytes")
	}
	expect(t, "view form", second(f.r.Publisher("")), ErrInvalid)
	expect(t, "view missing", second(f.r.Publisher("beta")), ErrNotFound)
	expect(t, "retire form", second(f.r.RetireKey("acme", "", ann)), ErrInvalid)
	expect(t, "retire missing", second(f.r.RetireKey("acme", "k7", ann)), ErrNotFound)
	expect(t, "retire publisher", second(f.r.RetireKey("beta", "k1", ann)), ErrNotFound)
	reason(t, "retire assistant", second(f.r.RetireKey("acme", "k1", bot)), ReasonAINotAllowed)
	f.must(second(f.r.RevokePublisher("acme", "Compromised account", ann)))
	reason(t, "revoked", f.try(keyChange("acme", "k5")), ReasonRevoked)
}

func TestKeyNameBound(t *testing.T) {
	l := limits
	l.Keys = 1
	f := setupWith(t, l)
	f.approve(Change{Kind: ChangePublisher, Publisher: Publisher{ID: "acme", Name: "Acme"}})
	for index := 0; index < namesPerKey; index++ {
		name := "k" + strings.Repeat("x", index)
		f.approve(keyChange("acme", name))
		f.must(second(f.r.RetireKey("acme", name, ann)))
	}
	expect(t, "names", f.try(keyChange("acme", "last")), ErrLimit)
}

func TestVersions(t *testing.T) {
	f := setup(t)
	f.publisher("acme")
	f.publisher("beta")
	m := manifest("ledger", "1.0.0")
	expect(t, "form", f.try(Change{Kind: ChangeVersion, Manifest: Manifest{}}), ErrInvalid)
	short := signed(m)
	short.Signature = short.Signature[:5]
	expect(t, "signature form", f.try(short), ErrInvalid)
	unknown := copyManifest(m)
	unknown.Key = "k7"
	expect(t, "unknown key", f.try(signed(unknown)), ErrNotFound)
	stranger := copyManifest(m)
	stranger.Publisher = "zeta"
	expect(t, "unknown publisher", f.try(signed(stranger)), ErrNotFound)
	tampered := signed(m)
	tampered.Manifest.Digest = strings.Repeat("0", 64)
	reason(t, "tampered", f.try(tampered), ReasonSignatureInvalid)
	foreign := signed(m)
	foreign.Signature = ed25519.Sign(private("betak1"), m.encode())
	reason(t, "other key", f.try(foreign), ReasonSignatureInvalid)
	old := copyManifest(m)
	old.Core = CoreRange{Min: "1.0.0", Max: "2.3.9"}
	reason(t, "too old", f.try(signed(old)), ReasonIncompatible)
	young := copyManifest(m)
	young.Core = CoreRange{Min: "2.4.1", Max: "3.0.0"}
	reason(t, "too new", f.try(signed(young)), ReasonIncompatible)
	exact := copyManifest(m)
	exact.Version, exact.Core = "0.1.0", CoreRange{Min: "2.4.0", Max: "2.4.0"}
	f.version(exact)
	f.version(m)
	expect(t, "same version", f.try(signed(m)), ErrConflict)
	changed := copyManifest(m)
	changed.Digest = strings.Repeat("1", 64)
	expect(t, "other digest", f.try(signed(changed)), ErrConflict)
	takeover := copyManifest(m)
	takeover.Version, takeover.Publisher = "9.0.0", "beta"
	reason(t, "takeover", f.try(signed(takeover)), ReasonPublisherMismatch)
	view, err := f.r.Version("ledger", "1.0.0")
	f.must(err)
	if view.Manifest.Digest != m.Digest || view.Vulnerable != nil || !ed25519.Verify(public("acmek1"), view.Manifest.encode(), view.Signature) {
		t.Fatalf("version %+v", view)
	}
	view.Manifest.Capabilities[0].Target = "x"
	again, _ := f.r.Version("ledger", "1.0.0")
	if again.Manifest.Capabilities[0].Target == "x" {
		t.Error("view shares manifest")
	}
	expect(t, "view form", second(f.r.Version("ledger", "1")), ErrInvalid)
	expect(t, "view plugin form", second(f.r.Version("", "1.0.0")), ErrInvalid)
	expect(t, "view missing", second(f.r.Version("ledger", "3.0.0")), ErrNotFound)
	expect(t, "view plugin missing", second(f.r.Version("other", "1.0.0")), ErrNotFound)
	f.must(second(f.r.RetireKey("acme", "k1", ann)))
	retired := manifest("ledger", "1.2.0")
	reason(t, "retired key", f.try(signed(retired)), ReasonKeyInactive)
	f.approve(keyChange("acme", "k2"))
	late := manifest("ledger", "1.2.0")
	late.Key = "k2"
	p, err := f.r.Propose(signed(late), ann)
	f.must(err)
	f.must(second(f.r.RevokePublisher("acme", "Stop publishing", cat)))
	reason(t, "revoked before approval", second(f.r.Approve(p.Number, bob)), ReasonPublisherInactive)
	if _, err := f.r.Version("ledger", "1.0.0"); err != nil {
		t.Error("revoked versions must stay readable")
	}
}

func TestVersionKeyWindow(t *testing.T) {
	f := setup(t)
	f.approve(Change{Kind: ChangePublisher, Publisher: Publisher{ID: "acme", Name: "Acme"}})
	later := keyChange("acme", "k1")
	later.Key.NotBefore, later.Key.NotAfter = start.Add(time.Hour), start.Add(2*time.Hour)
	later.Key.Proof = ed25519.Sign(private("acmek1"), challenge("acme", "k1"))
	f.approve(later)
	m := manifest("ledger", "1.0.0")
	reason(t, "before window", f.try(signed(m)), ReasonKeyInactive)
	f.advance(time.Hour)
	p, err := f.r.Propose(signed(m), ann)
	f.must(err)
	f.advance(time.Hour)
	f.must(second(f.r.Approve(p.Number, bob)))
	m.Version = "1.0.1"
	p, err = f.r.Propose(signed(m), ann)
	f.must(err)
	f.advance(time.Nanosecond)
	reason(t, "after window", second(f.r.Approve(p.Number, bob)), ReasonKeyInactive)
}

func TestVersionRoom(t *testing.T) {
	l := limits
	l.Plugins, l.Versions = 1, 2
	f := setupWith(t, l)
	f.publisher("acme")
	f.version(manifest("ledger", "1.0.0"))
	expect(t, "plugins", f.try(signed(manifest("other", "1.0.0"))), ErrLimit)
	f.version(manifest("ledger", "1.0.1"))
	expect(t, "versions", f.try(signed(manifest("ledger", "1.0.2"))), ErrLimit)
}

func TestWorkspacesAndRuntimes(t *testing.T) {
	l := limits
	l.Workspaces = 2
	f := setupWith(t, l)
	expect(t, "form", f.try(Change{Kind: ChangeWorkspace, Workspace: Workspace{ID: "w1"}}), ErrInvalid)
	expect(t, "id form", f.try(Change{Kind: ChangeWorkspace, Workspace: Workspace{Runtime: "run-1"}}), ErrInvalid)
	f.workspace("w1", "run-1", false)
	expect(t, "twice", f.try(Change{Kind: ChangeWorkspace, Workspace: Workspace{ID: "w1", Runtime: "run-9"}}), ErrConflict)
	reason(t, "runtime taken", f.try(Change{Kind: ChangeWorkspace, Workspace: Workspace{ID: "w2", Runtime: "run-1"}}), ReasonRuntimeTaken)
	f.workspace("v1", "run-v", true)
	expect(t, "room", f.try(Change{Kind: ChangeWorkspace, Workspace: Workspace{ID: "w3", Runtime: "run-3"}}), ErrLimit)
	w, err := f.r.Workspace("v1")
	f.must(err)
	if !w.Verification || w.Runtime != "run-v" {
		t.Errorf("workspace %+v", w)
	}
	expect(t, "view form", second(f.r.Workspace("")), ErrInvalid)
	expect(t, "view missing", second(f.r.Workspace("w9")), ErrNotFound)
	rotate := func(runtime string) Change {
		return Change{Kind: ChangeRuntime, Workspace: Workspace{ID: "w1", Runtime: runtime, Verification: true}}
	}
	expect(t, "rotate form", f.try(Change{Kind: ChangeRuntime, Workspace: Workspace{ID: "w1"}}), ErrInvalid)
	reason(t, "rotate to own", f.try(rotate("run-1")), ReasonRuntimeTaken)
	reason(t, "rotate to other", f.try(rotate("run-v")), ReasonRuntimeTaken)
	p := f.approve(rotate("run-2"))
	if p.Change.Workspace.Verification {
		t.Error("runtime change carried a verification flag")
	}
	w, _ = f.r.Workspace("w1")
	if w.Runtime != "run-2" || w.Verification {
		t.Errorf("rotated %+v", w)
	}
	reason(t, "tombstone", f.try(rotate("run-1")), ReasonRuntimeTaken)
	for index := 3; index <= runtimesPer; index++ {
		f.approve(rotate("run-" + strings.Repeat("r", index)))
	}
	expect(t, "rotation cap", f.try(rotate("run-last")), ErrLimit)
}
