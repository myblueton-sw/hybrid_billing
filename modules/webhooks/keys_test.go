package webhooks

import (
	"encoding/hex"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestKeyRules(t *testing.T) {
	f := setup(t)
	f.subscribe("s1", "d1")
	good := Key{Version: "k2", Public: public("k2"), NotBefore: start, NotAfter: start.Add(time.Hour)}
	for name, k := range map[string]Key{
		"no version":      {Public: good.Public, NotBefore: start, NotAfter: start.Add(time.Hour)},
		"no start":        {Version: "k2", Public: good.Public, NotAfter: start.Add(time.Hour)},
		"an empty window": {Version: "k2", Public: good.Public, NotBefore: start, NotAfter: start},
	} {
		expect(t, name, second(f.r.ProposeKey("w1", "s1", k, ann)), ErrInvalid)
	}
	reason(t, "an assistant proposing", second(f.r.ProposeKey("w1", "s1", good, bot)), ReasonAINotAllowed)
	reason(t, "a system proposing", second(f.r.ProposeKey("w1", "s1", good, relay)), ReasonNotHuman)
	reason(t, "an assistant retiring", second(f.r.RetireKey("w1", "s1", "k1", bot)), ReasonAINotAllowed)
	f.publishKey("s1", "k0")
	expect(t, "an unknown subscription", second(f.r.ProposeKey("w1", "s9", good, ann)), ErrNotFound)
	for index, weakKey := range append([][]byte{nil, make([]byte, 31)}, smallOrder...) {
		reason(t, "a weak key", second(f.r.ProposeKey("w1", "s1", Key{Version: "w" + string(rune('a'+index)), Public: weakKey, NotBefore: start, NotAfter: start.Add(time.Hour)}, ann)), ReasonWeakKey)
	}
	if weak(hexBytes("ecffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7e")) || !weak(hexBytes("edffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7f")) {
		t.Errorf("the canonical boundary")
	}
	reason(t, "a window above the key life", second(f.r.ProposeKey("w1", "s1", Key{Version: "k2", Public: good.Public, NotBefore: start, NotAfter: start.Add(limits.KeyLife + 1)}, ann)), ReasonKeyLife)
	view, err := f.r.ProposeKey("w1", "s1", good, ann)
	if err != nil || view.State != KeyProposed {
		t.Fatalf("proposal %+v %v", view, err)
	}
	view.Key.Public[0] ^= 1
	if again, err := f.r.ProposeKey("w1", "s1", good, ann); err != nil || again.Key.Public[0] != good.Public[0] {
		t.Errorf("a repeated proposal %+v %v", again, err)
	}
	expect(t, "another proposer", second(f.r.ProposeKey("w1", "s1", good, bob)), ErrConflict)
	expect(t, "an approved name", second(f.r.ProposeKey("w1", "s1", Key{Version: "k1", Public: public("k1"), NotBefore: start, NotAfter: start.Add(time.Hour)}, ann)), ErrConflict)
	expect(t, "approve no version", second(f.r.ApproveKey("w1", "s1", "", bob)), ErrInvalid)
	expect(t, "approve an unknown version", second(f.r.ApproveKey("w1", "s1", "k9", bob)), ErrNotFound)
	reason(t, "the proposer approving", second(f.r.ApproveKey("w1", "s1", "k2", ann)), ReasonSelfApproval)
	f.fk.set(func(k *fakes) { k.actual = "other" })
	reason(t, "a signer holding another key", second(f.r.ApproveKey("w1", "s1", "k2", bob)), ReasonKeyMismatch)
	f.fk.set(func(k *fakes) { k.actual, k.signErr = "", boom })
	reason(t, "a signer outage", second(f.r.ApproveKey("w1", "s1", "k2", bob)), ReasonDependency)
	f.fk.set(func(k *fakes) { k.signErr = nil })
	active, err := f.r.ApproveKey("w1", "s1", "k2", bob)
	if err != nil || active.State != KeyActive || active.Approver != "bob" {
		t.Fatalf("approval %+v %v", active, err)
	}
	f.must(second(f.r.ApproveKey("w1", "s1", "k2", bob)))
	reason(t, "another approver", second(f.r.ApproveKey("w1", "s1", "k2", ann)), ReasonNotPending)
	keys, err := f.r.Keys("w1", "s1", relay)
	if err != nil || len(keys) != 3 || keys[0].Key.Version != "k0" || keys[1].Key.Version != "k1" || keys[2].Key.Version != "k2" {
		t.Errorf("published keys %+v %v", keys, err)
	}
	expect(t, "keys of no subscription", second(f.r.Keys("w1", "", relay)), ErrInvalid)
	expect(t, "retire no version", second(f.r.RetireKey("w1", "s1", "", ann)), ErrInvalid)
	expect(t, "retire an unknown version", second(f.r.RetireKey("w1", "s1", "k9", ann)), ErrNotFound)
	if gone, err := f.r.RetireKey("w1", "s1", "k2", ann); err != nil || gone.State != KeyRetired {
		t.Errorf("retirement %+v %v", gone, err)
	}
	expect(t, "a retired name", second(f.r.ProposeKey("w1", "s1", good, ann)), ErrConflict)
	f.must(second(f.r.ProposeKey("w1", "s1", Key{Version: "k3", Public: public("k3"), NotBefore: start, NotAfter: start.Add(time.Hour)}, ann)))
	f.must(second(f.r.RetireKey("w1", "s1", "k3", ann)))
	f.must(second(f.r.ProposeKey("w1", "s1", Key{Version: "k3", Public: public("k3"), NotBefore: start, NotAfter: start.Add(time.Hour)}, ann)))
	if keys, _ := f.r.Keys("w1", "s1", ann); len(keys) != 2 {
		t.Errorf("listing after retirement %+v", keys)
	}
	f.must(second(f.r.RevokeSubscription("w1", "s1", ann)))
	if keys, _ := f.r.Keys("w1", "s1", ann); len(keys) != 0 {
		t.Errorf("keys of a revoked subscription %+v", keys)
	}
	reason(t, "a key for a revoked subscription", second(f.r.ProposeKey("w1", "s1", Key{Version: "k4", Public: public("k4"), NotBefore: start, NotAfter: start.Add(time.Hour)}, ann)), ReasonRevoked)
}

func TestKeyBounds(t *testing.T) {
	l := limits
	l.Keys = 1
	f := setupWith(t, l)
	f.subscribe("s1", "d1")
	f.must(second(f.r.ProposeKey("w1", "s1", Key{Version: "k2", Public: public("k2"), NotBefore: start, NotAfter: start.Add(time.Hour)}, ann)))
	expect(t, "a second outstanding proposal", second(f.r.ProposeKey("w1", "s1", Key{Version: "k3", Public: public("k3"), NotBefore: start, NotAfter: start.Add(time.Hour)}, ann)), ErrLimit)
	expect(t, "a second overlapping key", second(f.r.ApproveKey("w1", "s1", "k2", bob)), ErrLimit)
	f.advance(366 * 24 * time.Hour)
	if active, err := f.r.ApproveKey("w1", "s1", "k2", bob); err != nil || active.State != KeyActive {
		t.Errorf("an expired key still blocks the limit %+v %v", active, err)
	}
	for index := 0; index < namesPerKey; index++ {
		f.r.workspaces["w1"].subscriptions["s1"].names["n"+string(rune('a'+index%26))+string(rune('a'+index/26))] = true
	}
	expect(t, "the name bound", second(f.r.ProposeKey("w1", "s1", Key{Version: "k9", Public: public("k9"), NotBefore: *f.now, NotAfter: f.now.Add(time.Hour)}, ann)), ErrLimit)
	g := setup(t)
	g.subscribe("s1", "d1")
	g.must(second(g.r.ProposeKey("w1", "s1", Key{Version: "k2", Public: public("k2"), NotBefore: start, NotAfter: start.Add(time.Hour)}, ann)))
	for _, step := range []struct {
		name   string
		during func()
	}{
		{"a withdrawal", func() { g.must(second(g.r.RetireKey("w1", "s1", "k2", ann))) }},
		{"a revocation", func() { g.must(second(g.r.RevokeSubscription("w1", "s1", ann))) }},
	} {
		name, during := step.name, step.during
		g.fk.set(func(k *fakes) {
			k.onSign = func() {
				g.fk.set(func(k *fakes) { k.onSign = nil })
				during()
			}
		})
		expect(t, name+" during the challenge", second(g.r.ApproveKey("w1", "s1", "k2", bob)), ErrStale)
		g.fk.set(func(k *fakes) { k.onSign = nil })
		if name == "a withdrawal" {
			g.must(second(g.r.ProposeKey("w1", "s1", Key{Version: "k2", Public: public("k2"), NotBefore: start, NotAfter: start.Add(time.Hour)}, ann)))
		}
	}
}

func TestSigningChecks(t *testing.T) {
	for name, c := range map[string]struct {
		change func(f world)
		want   string
	}{
		"an unpublished version": {func(f world) { f.fk.set(func(k *fakes) { k.version = "k9" }) }, ReasonKeyUnpublished},
		"a retired version":      {func(f world) { f.must(second(f.r.RetireKey("w1", "s1", "k1", ann))) }, ReasonKeyUnpublished},
		"a window not yet open": {func(f world) {
			f.publishKeyAt("s1", "k2", start.Add(time.Hour), start.Add(2*time.Hour))
			f.fk.set(func(k *fakes) { k.version = "k2" })
		}, ReasonKeyUnpublished},
		"a window already closed": {func(f world) {
			f.publishKeyAt("s1", "k2", start.Add(-2*time.Hour), start.Add(-time.Hour))
			f.fk.set(func(k *fakes) { k.version = "k2" })
		}, ReasonKeyUnpublished},
		"a signer holding another key": {func(f world) { f.fk.set(func(k *fakes) { k.actual = "other" }) }, ReasonKeyMismatch},
		"a retirement during signing": {func(f world) {
			f.fk.set(func(k *fakes) {
				k.onSign = func() {
					f.fk.set(func(k *fakes) { k.onSign = nil })
					f.must(second(f.r.RetireKey("w1", "s1", "k1", ann)))
				}
			})
		}, ReasonKeyUnpublished},
	} {
		f := setup(t)
		f.subscribe("s1", "d1")
		f.publish("e1", invoice.Name)
		lease := f.lease()
		c.change(f)
		got, err := f.r.Confirm("w1", lease.Delivery, lease.Token, relay)
		reason(t, name, err, c.want)
		if got.Signature != nil {
			t.Errorf("%s released a signature", name)
		}
		view, _ := f.r.SubscriptionOf("w1", "s1")
		s := f.r.workspaces["w1"].subscriptions["s1"]
		if view.State != SubscriptionSuspended || view.Reason != c.want || s.open.attempts != 0 || s.open.phase != phaseDue {
			t.Errorf("%s: %+v, %d attempts, %s", name, view, s.open.attempts, s.open.phase)
		}
	}
	f := setup(t)
	f.subscribe("s1", "d1")
	f.publish("e1", invoice.Name)
	f.fk.set(func(k *fakes) { k.version = "k9" })
	lease := f.lease()
	reason(t, "the first refusal", second(f.r.Confirm("w1", lease.Delivery, lease.Token, relay)), ReasonKeyUnpublished)
	f.must(second(f.r.RequestResume("w1", "s1", ann)))
	f.must(second(f.r.ApproveResume("w1", "s1", bob)))
	again := f.lease()
	reason(t, "resuming without a key suspends again", second(f.r.Confirm("w1", again.Delivery, again.Token, relay)), ReasonKeyUnpublished)
	edge := setup(t)
	edge.subscribe("s1", "d1")
	edge.publish("e1", invoice.Name)
	edge.publishKeyAt("s1", "k2", start, start.Add(time.Hour))
	edge.fk.set(func(k *fakes) { k.version = "k2" })
	if signed := edge.confirm(edge.lease()); signed.Version != "k2" {
		t.Errorf("a window that opens exactly now %+v", signed)
	}
}

func TestRedirectAfterRetirement(t *testing.T) {
	f := setup(t)
	f.subscribe("s1", "d1")
	f.publish("e1", invoice.Name)
	lease := f.lease()
	f.confirm(lease)
	f.fk.set(func(k *fakes) { k.redirectTo = "d1" })
	f.must(second(f.r.RetireKey("w1", "s1", "k1", ann)))
	reason(t, "a redirect after retirement", third(f.r.Redirect("w1", lease.Delivery, lease.Token, "https://d1/", 0, "/x", relay)), ReasonKeyUnpublished)
	if s := f.r.workspaces["w1"].subscriptions["s1"]; s.open.attempts != 0 || s.open.phase != phaseDue {
		t.Errorf("a refused redirect counted an attempt: %d %s", s.open.attempts, s.open.phase)
	}
	g := setup(t)
	g.subscribe("s1", "d1")
	g.publish("e1", invoice.Name)
	other := g.lease()
	g.confirm(other)
	g.fk.set(func(k *fakes) { k.redirectTo = "d1" })
	g.fk.set(func(k *fakes) {
		k.onCheck = func() {
			g.fk.set(func(k *fakes) { k.onCheck = nil })
			g.must(second(g.r.Report("w1", other.Delivery, other.Token, Outcome{Status: StatusDelivered}, relay)))
		}
	})
	expect(t, "a report during the redirect check", third(g.r.Redirect("w1", other.Delivery, other.Token, "https://d1/", 0, "/x", relay)), ErrStale)
}

func TestHeldDeliveries(t *testing.T) {
	for name, finish := range map[string]func(f world, l Lease){
		"rejected": func(f world, l Lease) { f.confirm(l); f.report(l, StatusRejected, 0) },
		"attempts": func(f world, l Lease) {
			for attempt := 0; attempt < limits.Attempts; attempt++ {
				f.confirm(l)
				f.report(l, StatusFailed, 0)
				f.advance(time.Minute)
				if attempt < limits.Attempts-1 {
					l = f.lease()
				}
			}
		},
		"max age": func(f world, l Lease) {
			f.confirm(l)
			f.report(l, StatusThrottled, time.Minute)
			f.advance(time.Hour)
			f.dispatch()
		},
	} {
		f := setup(t)
		f.subscribe("s1", "d1")
		for index := 0; index < 2; index++ {
			f.publish("x"+string(rune('a'+index)), invoice.Name)
			l := f.lease()
			f.confirm(l)
			f.report(l, StatusRejected, 0)
		}
		f.publish("held", invoice.Name)
		finish(f, f.lease())
		s := f.r.workspaces["w1"].subscriptions["s1"]
		if s.open == nil || !s.open.held || s.view.Reason != ReasonBacklog {
			t.Fatalf("%s: not held %+v", name, s.open)
		}
		reason0 := s.open.reason
		f.advance(2 * time.Hour)
		f.dispatch()
		if s.open.reason != reason0 {
			t.Errorf("%s: the held reason changed to %s", name, s.open.reason)
		}
		f.must(second(f.r.RequestResume("w1", "s1", ann)))
		f.must(second(f.r.ApproveResume("w1", "s1", bob)))
		if leases := f.dispatch(); len(leases) != 0 || s.view.Reason != ReasonBacklog {
			t.Errorf("%s: a held delivery was leased with full lists %+v", name, leases)
		}
		f.must(f.r.RequestDismiss("w1", "s1", s.dead[0].id, ann))
		f.must(f.r.ApproveDismiss("w1", "s1", s.dead[0].id, bob))
		if s.open != nil || len(s.dead) != 2 || s.dead[1].held || s.dead[1].reason != reason0 {
			t.Errorf("%s: the held delivery was not buried after room was freed", name)
		}
	}
	g := setup(t)
	g.subscribe("s1", "d1")
	for index := 0; index < 3; index++ {
		g.publish("y"+string(rune('a'+index)), invoice.Name)
		l := g.lease()
		g.confirm(l)
		g.report(l, StatusRejected, 0)
	}
	s := g.r.workspaces["w1"].subscriptions["s1"]
	g.must(g.r.RequestDismiss("w1", "s1", s.dead[0].id, ann))
	g.must(g.r.ApproveDismiss("w1", "s1", s.dead[0].id, bob))
	g.must(second(g.r.RequestResume("w1", "s1", ann)))
	g.must(second(g.r.ApproveResume("w1", "s1", bob)))
	g.must(g.r.RequestDismiss("w1", "s1", s.dead[0].id, ann))
	g.must(g.r.ApproveDismiss("w1", "s1", s.dead[0].id, bob))
	g.publish("z", invoice.Name)
	if leases := g.dispatch(); len(leases) != 1 || leases[0].EventID != "z" {
		t.Errorf("after the held delivery was buried %+v", leases)
	} else if g.r.workspaces["w1"].subscriptions["s1"].open.reason != "" {
		t.Errorf("a leased delivery carries a reason")
	}
}

func TestKeyReviewGaps(t *testing.T) {
	f := setup(t)
	f.subscribe("s1", "d1")
	f.publish("e1", invoice.Name)
	f.publishKeyAt("s1", "k2", start.Add(-time.Minute), start.Add(time.Second))
	f.fk.set(func(k *fakes) {
		k.version = "k2"
		k.onSign = nil
	})
	lease := f.lease()
	f.fk.set(func(k *fakes) {
		k.onState = func() {
			f.fk.set(func(k *fakes) { k.onState = nil })
			f.advance(time.Second + time.Nanosecond)
		}
	})
	got, err := f.r.Confirm("w1", lease.Delivery, lease.Token, relay)
	reason(t, "an attempt time moved past the window", err, ReasonKeyUnpublished)
	if got.Signature != nil {
		t.Errorf("a signature was released past the window")
	}
	g := setup(t)
	g.subscribe("s1", "d1")
	g.publish("e1", invoice.Name)
	g.publishKeyAt("s1", "k2", start.Add(-time.Minute), start)
	g.fk.set(func(k *fakes) { k.version = "k2" })
	if signed := g.confirm(g.lease()); signed.Version != "k2" {
		t.Errorf("signing at exactly the end of the window %+v", signed)
	}
	exact := setup(t)
	exact.subscribe("s1", "d1")
	if _, err := exact.r.ProposeKey("w1", "s1", Key{Version: "k2", Public: public("k2"), NotBefore: start, NotAfter: start.Add(limits.KeyLife)}, ann); err != nil {
		t.Errorf("a window of exactly the key life: %v", err)
	}
	h := setup(t)
	h.subscribe("s1", "d1")
	_, err = h.r.ProposeSubscription("w2", Subscription{ID: "s1", Subscriber: "acme", Destination: "d1", Types: []string{invoice.Name}}, ann)
	h.must(err)
	if keys, _ := h.r.Keys("w2", "s1", ann); len(keys) != 0 {
		t.Errorf("keys leaked to another workspace %+v", keys)
	}
	expect(t, "approving another workspace's key", second(h.r.ApproveKey("w2", "s1", "k1", bob)), ErrNotFound)
	l := limits
	l.Subscriptions = 1
	p := setupWith(t, l)
	p.subscribe("s1", "d1")
	p.must(second(p.r.RevokeSubscription("w1", "s1", ann)))
	p.subscribe("s2", "d1")
	p.must(second(p.r.RevokeSubscription("w1", "s2", ann)))
	p.must(second(p.r.ProposeSubscription("w1", Subscription{ID: "s3", Subscriber: "acme", Destination: "d1", Types: []string{invoice.Name}}, ann)))
	p.must(second(p.r.RevokeSubscription("w1", "s3", ann)))
	p.must(second(p.r.ProposeSubscription("w1", Subscription{ID: "s1", Subscriber: "acme", Destination: "d1", Types: []string{invoice.Name}}, ann)))
	if keys, _ := p.r.Keys("w1", "s1", ann); len(keys) != 0 {
		t.Errorf("a purged and re-created subscription inherited keys %+v", keys)
	}
	n := setup(t)
	n.subscribe("s1", "d1")
	names := n.r.workspaces["w1"].subscriptions["s1"].names
	for index := len(names); index < limits.Keys*namesPerKey-1; index++ {
		names["n"+strconv.Itoa(index)] = true
	}
	n.must(second(n.r.ProposeKey("w1", "s1", Key{Version: "last", Public: public("last"), NotBefore: start, NotAfter: start.Add(time.Hour)}, ann)))
	n.must(second(n.r.ApproveKey("w1", "s1", "last", bob)))
	expect(t, "one name beyond the bound", second(n.r.ProposeKey("w1", "s1", Key{Version: "over", Public: public("over"), NotBefore: start, NotAfter: start.Add(time.Hour)}, ann)), ErrLimit)
}

func TestConcurrentRetirement(t *testing.T) {
	for round := 0; round < 20; round++ {
		f := setup(t)
		f.subscribe("s1", "d1")
		f.publish("e1", invoice.Name)
		lease := f.lease()
		var wait sync.WaitGroup
		var signed Signed
		var err error
		wait.Add(2)
		go func() {
			defer wait.Done()
			signed, err = f.r.Confirm("w1", lease.Delivery, lease.Token, relay)
		}()
		go func() {
			defer wait.Done()
			_, _ = f.r.RetireKey("w1", "s1", "k1", ann)
		}()
		wait.Wait()
		view, _ := f.r.SubscriptionOf("w1", "s1")
		switch {
		case err == nil && signed.Signature != nil && signed.Version == "k1" && view.State == SubscriptionActive:
		case err != nil && signed.Signature == nil && view.Reason == ReasonKeyUnpublished:
		default:
			t.Fatalf("round %d: %+v %v %+v", round, signed, err, view)
		}
	}
}

func TestWeakKeyLiterals(t *testing.T) {
	for _, text := range []string{
		"0100000000000000000000000000000000000000000000000000000000000000",
		"0100000000000000000000000000000000000000000000000000000000000080",
		"0000000000000000000000000000000000000000000000000000000000000000",
		"0000000000000000000000000000000000000000000000000000000000000080",
		"ecffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7f",
		"ecffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		"26e8958fc2b227b045c3f489f2ef98f0d5dfac05d3c63339b13802886d53fc05",
		"26e8958fc2b227b045c3f489f2ef98f0d5dfac05d3c63339b13802886d53fc85",
		"c7176a703d4dd84fba3c0b760d10670f2a2053fa2c39ccc64ec7fd7792ac037a",
		"c7176a703d4dd84fba3c0b760d10670f2a2053fa2c39ccc64ec7fd7792ac03fa",
	} {
		decoded, err := hex.DecodeString(text)
		if err != nil || !weak(decoded) {
			t.Errorf("%s is not weak", text)
		}
	}
	if weak(public("k1")) {
		t.Errorf("a generated key is weak")
	}
}
