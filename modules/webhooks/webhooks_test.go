package webhooks

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestNewRegistry(t *testing.T) {
	f := setup(t)
	clock := func() time.Time { return start }
	good := Config{Outbox: f.out, Destinations: f.fk, Permits: f.fk, Signer: f.fk, Redirects: f.fk}
	for name, c := range map[string]Config{
		"no outbox": {Destinations: f.fk, Permits: f.fk, Signer: f.fk, Redirects: f.fk}, "no destinations": {Outbox: f.out, Permits: f.fk, Signer: f.fk, Redirects: f.fk},
		"no permits": {Outbox: f.out, Destinations: f.fk, Signer: f.fk, Redirects: f.fk}, "no signer": {Outbox: f.out, Destinations: f.fk, Permits: f.fk, Redirects: f.fk},
		"no redirects": {Outbox: f.out, Destinations: f.fk, Permits: f.fk, Signer: f.fk},
	} {
		expect(t, name, second(NewRegistry(c, limits, clock)), ErrInvalid)
	}
	expect(t, "no clock", second(NewRegistry(good, limits, nil)), ErrInvalid)
	counts := map[string]struct {
		field func(*Limits) *int
		max   int
	}{
		"workspaces": {func(l *Limits) *int { return &l.Workspaces }, maxWorkspaces}, "subscriptions": {func(l *Limits) *int { return &l.Subscriptions }, maxSubscription},
		"types": {func(l *Limits) *int { return &l.Types }, maxTypes}, "attempts": {func(l *Limits) *int { return &l.Attempts }, maxAttempts},
		"dead": {func(l *Limits) *int { return &l.Dead }, maxDead}, "batch": {func(l *Limits) *int { return &l.Batch }, maxBatch}, "work": {func(l *Limits) *int { return &l.Work }, maxWork},
	}
	for name, c := range counts {
		for value, ok := range map[int]bool{0: false, 1: true, c.max: true, c.max + 1: false} {
			l := limits
			*c.field(&l) = value
			if _, err := NewRegistry(good, l, clock); ok != (err == nil) {
				t.Errorf("%s %d: %v", name, value, err)
			}
		}
	}
	for name, change := range map[string]func(*Limits){
		"a short lease": func(l *Limits) { l.Lease = minLease - 1 }, "a long lease": func(l *Limits) { l.Lease = maxLease + 1 },
		"a short backoff": func(l *Limits) { l.Backoff = minBackoff - 1 }, "a backoff above the maximum": func(l *Limits) { l.Backoff = 2 * time.Minute },
		"a long maximum backoff": func(l *Limits) { l.MaxBackoff, l.MaxAge = maxMaxBackoff+1, maxMaxAge }, "an age below the backoff": func(l *Limits) { l.MaxAge = time.Second - 1 },
		"a long age": func(l *Limits) { l.MaxAge = maxMaxAge + 1 },
	} {
		l := limits
		change(&l)
		expect(t, name, second(NewRegistry(good, l, clock)), ErrInvalid)
	}
	for name, change := range map[string]func(*Limits){
		"the shortest lease": func(l *Limits) { l.Lease = minLease }, "the longest lease": func(l *Limits) { l.Lease = maxLease },
		"the shortest backoff": func(l *Limits) { l.Backoff = minBackoff }, "a backoff at the maximum": func(l *Limits) { l.Backoff = time.Minute },
		"the longest maximum backoff": func(l *Limits) { l.MaxBackoff = maxMaxBackoff }, "an age at the backoff": func(l *Limits) { l.MaxAge = time.Second },
		"the longest age": func(l *Limits) { l.MaxAge = maxMaxAge },
	} {
		l := limits
		change(&l)
		if _, err := NewRegistry(good, l, clock); err != nil {
			t.Errorf("%s: %v", name, err)
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
	if err := error(Refusal{Reason: "x"}); err.Error() != ErrRejected.Error()+": x" {
		t.Errorf("refusal text %v", err)
	}
}

func TestSubscriptions(t *testing.T) {
	f := setup(t)
	good := Subscription{ID: "s1", Subscriber: "acme", Destination: "d1", Types: []string{invoice.Name}}
	for name, s := range map[string]Subscription{
		"no identifier": {Subscriber: "acme", Destination: "d1", Types: []string{"t"}}, "no subscriber": {ID: "s1", Destination: "d1", Types: []string{"t"}},
		"no destination": {ID: "s1", Subscriber: "acme", Types: []string{"t"}}, "no types": {ID: "s1", Subscriber: "acme", Destination: "d1"},
		"too many types":  {ID: "s1", Subscriber: "acme", Destination: "d1", Types: []string{"a", "b", "c"}},
		"a repeated type": {ID: "s1", Subscriber: "acme", Destination: "d1", Types: []string{"a", "a"}},
		"a type as text":  {ID: "s1", Subscriber: "acme", Destination: "d1", Types: []string{"a b"}},
	} {
		expect(t, name, second(f.r.ProposeSubscription("w1", s, ann)), ErrInvalid)
	}
	expect(t, "no workspace", second(f.r.ProposeSubscription("", good, ann)), ErrInvalid)
	expect(t, "an invalid actor", second(f.r.ProposeSubscription("w1", good, Actor{ID: "ann"})), ErrInvalid)
	reason(t, "an assistant proposing", second(f.r.ProposeSubscription("w1", good, bot)), ReasonAINotAllowed)
	reason(t, "a system proposing", second(f.r.ProposeSubscription("w1", good, relay)), ReasonNotHuman)
	view, err := f.r.ProposeSubscription("w1", Subscription{ID: "s1", Subscriber: "acme", Destination: "d1", Types: []string{"b", "a"}}, ann)
	if err != nil || view.State != SubscriptionProposed || view.Subscription.Types[0] != "a" {
		t.Fatalf("proposal %+v %v", view, err)
	}
	f.must(second(f.r.ProposeSubscription("w1", Subscription{ID: "s1", Subscriber: "acme", Destination: "d1", Types: []string{"a", "b"}}, ann)))
	for name, s := range map[string]Subscription{
		"another subscriber":  {ID: "s1", Subscriber: "other", Destination: "d1", Types: []string{"a", "b"}},
		"another destination": {ID: "s1", Subscriber: "acme", Destination: "d2", Types: []string{"a", "b"}},
		"other types":         {ID: "s1", Subscriber: "acme", Destination: "d1", Types: []string{"a", "c"}},
		"fewer types":         {ID: "s1", Subscriber: "acme", Destination: "d1", Types: []string{"a"}},
	} {
		expect(t, name, second(f.r.ProposeSubscription("w1", s, ann)), ErrConflict)
	}
	expect(t, "another proposer", second(f.r.ProposeSubscription("w1", Subscription{ID: "s1", Subscriber: "acme", Destination: "d1", Types: []string{"a", "b"}}, bob)), ErrConflict)
	expect(t, "approve no workspace", second(f.r.ApproveSubscription("", "s1", bob)), ErrInvalid)
	expect(t, "approve no identifier", second(f.r.ApproveSubscription("w1", "", bob)), ErrInvalid)
	expect(t, "approve an invalid actor", second(f.r.ApproveSubscription("w1", "s1", Actor{ID: "bob"})), ErrInvalid)
	reason(t, "an assistant approving", second(f.r.ApproveSubscription("w1", "s1", bot)), ReasonAINotAllowed)
	expect(t, "an unknown workspace", second(f.r.ApproveSubscription("w9", "s1", bob)), ErrNotFound)
	expect(t, "an unknown subscription", second(f.r.ApproveSubscription("w1", "s9", bob)), ErrNotFound)
	reason(t, "the proposer approving", second(f.r.ApproveSubscription("w1", "s1", ann)), ReasonSelfApproval)
	f.fk.set(func(k *fakes) { k.states["d1"] = DestinationInactive })
	reason(t, "an inactive destination", second(f.r.ApproveSubscription("w1", "s1", bob)), ReasonDestinationInactive)
	f.fk.set(func(k *fakes) { k.states["d1"], k.stateErr = DestinationActive, boom })
	reason(t, "a destination outage", second(f.r.ApproveSubscription("w1", "s1", bob)), ReasonDependency)
	f.fk.set(func(k *fakes) { k.stateErr, k.subscribe = nil, false })
	reason(t, "a refused subscription", second(f.r.ApproveSubscription("w1", "s1", bob)), ReasonSubscribeRefused)
	f.fk.set(func(k *fakes) { k.subscribe, k.subErr = true, boom })
	reason(t, "a permit outage", second(f.r.ApproveSubscription("w1", "s1", bob)), ReasonDependency)
	f.fk.set(func(k *fakes) { k.subErr = nil })
	f.out.boundErr = boom
	reason(t, "an outbox outage", second(f.r.ApproveSubscription("w1", "s1", bob)), ReasonDependency)
	f.out.boundErr = nil
	f.publish("e1", invoice.Name)
	active, err := f.r.ApproveSubscription("w1", "s1", bob)
	if err != nil || active.State != SubscriptionActive || active.Cursor != 1 || active.Approver != "bob" {
		t.Fatalf("approval %+v %v", active, err)
	}
	f.must(second(f.r.ApproveSubscription("w1", "s1", bob)))
	reason(t, "another approver", second(f.r.ApproveSubscription("w1", "s1", ann)), ReasonNotPending)
	if read, err := f.r.SubscriptionOf("w1", "s1"); err != nil || read.State != SubscriptionActive {
		t.Errorf("reading %+v %v", read, err)
	}
	expect(t, "read no workspace", second(f.r.SubscriptionOf("", "s1")), ErrInvalid)
	expect(t, "read no identifier", second(f.r.SubscriptionOf("w1", "")), ErrInvalid)
	expect(t, "read an unknown workspace", second(f.r.SubscriptionOf("w9", "s1")), ErrNotFound)
	expect(t, "read an unknown subscription", second(f.r.SubscriptionOf("w1", "s9")), ErrNotFound)
	f.must(second(f.r.ProposeSubscription("w1", Subscription{ID: "s2", Subscriber: "acme", Destination: "d1", Types: []string{"a"}}, ann)))
	f.must(second(f.r.ProposeSubscription("w1", Subscription{ID: "s3", Subscriber: "acme", Destination: "d1", Types: []string{"a"}}, ann)))
	expect(t, "a fourth subscription", second(f.r.ProposeSubscription("w1", Subscription{ID: "s4", Subscriber: "acme", Destination: "d1", Types: []string{"a"}}, ann)), ErrLimit)
	f.must(second(f.r.ProposeSubscription("w2", Subscription{ID: "s1", Subscriber: "acme", Destination: "d1", Types: []string{"a"}}, ann)))
	expect(t, "a third workspace", second(f.r.ProposeSubscription("w3", Subscription{ID: "s1", Subscriber: "acme", Destination: "d1", Types: []string{"a"}}, ann)), ErrLimit)
}

func TestLifecycle(t *testing.T) {
	f := setup(t)
	f.subscribe("s1", "d1")
	reason(t, "resuming an active subscription", second(f.r.RequestResume("w1", "s1", ann)), ReasonNotPending)
	reason(t, "approving a resume never requested", second(f.r.ApproveResume("w1", "s1", bob)), ReasonNotPending)
	paused, err := f.r.PauseSubscription("w1", "s1", ann)
	if err != nil || paused.State != SubscriptionPaused || paused.Reason != ReasonPaused {
		t.Fatalf("pause %+v %v", paused, err)
	}
	reason(t, "pausing twice", second(f.r.PauseSubscription("w1", "s1", ann)), ReasonNotActive)
	f.publish("e1", invoice.Name)
	if leases := f.dispatch(); len(leases) != 0 {
		t.Errorf("a paused subscription was dispatched: %+v", leases)
	}
	f.must(second(f.r.RequestResume("w1", "s1", ann)))
	reason(t, "the requester approving", second(f.r.ApproveResume("w1", "s1", ann)), ReasonSelfApproval)
	f.fk.set(func(k *fakes) { k.subscribe = false })
	reason(t, "a resume refused by the permits", second(f.r.ApproveResume("w1", "s1", bob)), ReasonSubscribeRefused)
	f.fk.set(func(k *fakes) { k.subscribe = true })
	f.out.boundErr = boom
	reason(t, "a resume during an outbox outage", second(f.r.ApproveResume("w1", "s1", bob)), ReasonDependency)
	f.out.boundErr = nil
	resumed, err := f.r.ApproveResume("w1", "s1", bob)
	if err != nil || resumed.State != SubscriptionActive || resumed.Resumer != "" || resumed.Cursor != 0 {
		t.Fatalf("resume %+v %v", resumed, err)
	}
	expect(t, "invalid actions", second(f.r.PauseSubscription("w1", "", ann)), ErrInvalid)
	expect(t, "unknown subscriptions", second(f.r.RequestResume("w1", "s9", ann)), ErrNotFound)
	expect(t, "approve resume of an unknown subscription", second(f.r.ApproveResume("w1", "s9", ann)), ErrNotFound)
	expect(t, "revoke an unknown subscription", second(f.r.RevokeSubscription("w1", "s9", ann)), ErrNotFound)
	if leases := f.dispatch(); len(leases) != 1 {
		t.Fatalf("after resume %+v", leases)
	}
	gone, err := f.r.RevokeSubscription("w1", "s1", ann)
	if err != nil || gone.State != SubscriptionRevoked || gone.Reason != ReasonSubscriptionRevoked {
		t.Errorf("revoke %+v %v", gone, err)
	}
	f.must(second(f.r.RevokeSubscription("w1", "s1", ann)))
	views, _ := f.r.Deliveries("w1", "s1", 0, 10, ann)
	if len(views) != 1 || views[0].State != JobCancelled || views[0].OutcomeUnknown {
		t.Errorf("a cancelled delivery %+v", views)
	}
}

func TestDispatchAndCursor(t *testing.T) {
	f := setup(t)
	expect(t, "no workspace", second(f.r.Dispatch("", 1, relay)), ErrInvalid)
	expect(t, "no limit", second(f.r.Dispatch("w1", 0, relay)), ErrInvalid)
	expect(t, "a limit above the batch", second(f.r.Dispatch("w1", limits.Batch+1, relay)), ErrInvalid)
	expect(t, "an invalid actor", second(f.r.Dispatch("w1", 1, Actor{})), ErrInvalid)
	reason(t, "a human dispatching", second(f.r.Dispatch("w1", 1, ann)), ReasonNotSystem)
	if leases := f.dispatch(); len(leases) != 0 {
		t.Errorf("an unknown workspace %+v", leases)
	}
	f.subscribe("s1", "d1")
	if leases := f.dispatch(); len(leases) != 0 {
		t.Errorf("a subscription at the head %+v", leases)
	}
	f.publish("o1", "other")
	f.publish("e1", invoice.Name)
	leases := f.dispatch()
	if len(leases) != 1 || leases[0].EventID != "e1" || leases[0].Sequence != 2 || leases[0].Destination != "d1" || leases[0].Subscription != "s1" {
		t.Fatalf("a lease %+v", leases)
	}
	if again := f.dispatch(); len(again) != 0 {
		t.Errorf("one open delivery per subscription %+v", again)
	}
	f.confirm(leases[0])
	if view := f.report(leases[0], StatusDelivered, 0); view.State != JobSucceeded {
		t.Errorf("delivered %+v", view)
	}
	f.out.boundErr = boom
	reason(t, "an outbox outage", second(f.r.Dispatch("w1", 1, relay)), ReasonDependency)
	f.out.boundErr = nil
	f.publish("e2", invoice.Name)
	f.out.readErr = boom
	if leases := f.dispatch(); len(leases) != 0 {
		t.Errorf("a read outage leased %+v", leases)
	}
	f.out.readErr = nil
	if leases := f.dispatch(); len(leases) != 1 || leases[0].EventID != "e2" {
		t.Errorf("after the read outage %+v", leases)
	}
}

func TestConfirmAndSignature(t *testing.T) {
	f := setup(t)
	f.subscribe("s1", "d1")
	f.publish("e1", invoice.Name)
	lease := f.dispatch()[0]
	expect(t, "no token", second(f.r.Confirm("w1", lease.Delivery, "", relay)), ErrInvalid)
	reason(t, "a human confirming", second(f.r.Confirm("w1", lease.Delivery, lease.Token, ann)), ReasonNotSystem)
	expect(t, "a wrong token", second(f.r.Confirm("w1", lease.Delivery, "nope", relay)), ErrStale)
	expect(t, "an unknown workspace", second(f.r.Confirm("w9", lease.Delivery, lease.Token, relay)), ErrStale)
	expect(t, "a report before confirmation", second(f.r.Report("w1", lease.Delivery, lease.Token, Outcome{Status: StatusDelivered}, relay)), ErrStale)
	signed := f.confirm(lease)
	if signed.Attempt != 1 || signed.Version != "k1" || signed.Headers["webhook-delivery"] != lease.Delivery || signed.Headers["webhook-event-source"] != "urn:billing" {
		t.Fatalf("signed %+v", signed)
	}
	var want bytes.Buffer
	digest := sha256.Sum256(signed.Body)
	for _, value := range []string{signatureTag, "w1", "s1", lease.Delivery, "urn:billing", "e1", "1", "k1", strconv.FormatInt(start.UnixNano(), 10), string(digest[:])} {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(value)))
		want.Write(size[:])
		want.WriteString(value)
	}
	sum := sha256.Sum256(append([]byte("k1|"), want.Bytes()...))
	if !bytes.Equal(signed.Signature, sum[:]) {
		t.Errorf("the signed message differs from the known answer")
	}
	expect(t, "a second confirmation", second(f.r.Confirm("w1", lease.Delivery, lease.Token, relay)), ErrStale)
	if url, hop, err := f.r.Redirect("w1", lease.Delivery, lease.Token, "https://d1/", 0, "/next", relay); err == nil || url != "" || hop != 0 {
		t.Errorf("a redirect to another destination %q %d %v", url, hop, err)
	}
	f.fk.set(func(k *fakes) { k.redirectTo = "d1" })
	if url, hop, err := f.r.Redirect("w1", lease.Delivery, lease.Token, "https://d1/", 0, "/next", relay); err != nil || url != "/next" || hop != 1 {
		t.Errorf("a redirect to the own destination %q %d %v", url, hop, err)
	}
	f.fk.set(func(k *fakes) { k.redirectErr = boom })
	reason(t, "a refused redirect", third(f.r.Redirect("w1", lease.Delivery, lease.Token, "https://d1/", 0, "/next", relay)), ReasonUnapprovedRedirect)
	expect(t, "a redirect with a stale token", third(f.r.Redirect("w1", lease.Delivery, "nope", "https://d1/", 0, "/x", relay)), ErrStale)
	f.report(lease, StatusFailed, 0)
	f.advance(time.Minute)
	f.fk.set(func(k *fakes) { k.version = "k2" })
	retry := f.dispatch()[0]
	expect(t, "a redirect before confirmation", third(f.r.Redirect("w1", retry.Delivery, retry.Token, "https://d1/", 0, "/x", relay)), ErrStale)
	if again := f.confirm(retry); again.Attempt != 2 || again.Version != "k2" {
		t.Errorf("a retry after a key change %+v", again)
	}
}

func TestConfirmOutcomes(t *testing.T) {
	cases := map[string]struct {
		change func(*fakes)
		want   string
		state  string
	}{
		"a destination outage": {func(k *fakes) { k.stateErr = boom }, ReasonDependency, SubscriptionActive},
		"a permit outage":      {func(k *fakes) { k.allowErr = boom }, ReasonDependency, SubscriptionActive},
		"a version outage":     {func(k *fakes) { k.versionErr = boom }, ReasonDependency, SubscriptionActive},
		"a signer outage":      {func(k *fakes) { k.signErr = boom }, ReasonDependency, SubscriptionActive},
		"an inactive target":   {func(k *fakes) { k.states["d1"] = DestinationInactive }, ReasonDestinationInactive, SubscriptionSuspended},
		"a revoked target":     {func(k *fakes) { k.states["d1"] = DestinationRevoked }, ReasonDestinationRevoked, SubscriptionRevoked},
		"a revoked permission": {func(k *fakes) { k.allowed = false }, ReasonPermissionRevoked, SubscriptionActive},
	}
	for name, c := range cases {
		f := setup(t)
		f.subscribe("s1", "d1")
		f.publish("e1", invoice.Name)
		lease := f.dispatch()[0]
		f.fk.set(c.change)
		reason(t, name, second(f.r.Confirm("w1", lease.Delivery, lease.Token, relay)), c.want)
		if view, _ := f.r.SubscriptionOf("w1", "s1"); view.State != c.state {
			t.Errorf("%s: subscription %s", name, view.State)
		}
	}
}
