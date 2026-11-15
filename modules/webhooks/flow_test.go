package webhooks

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"
)

func (f world) lease() Lease {
	f.t.Helper()
	leases := f.dispatch()
	if len(leases) != 1 {
		f.t.Fatalf("leases %+v", leases)
	}
	return leases[0]
}

func TestReports(t *testing.T) {
	f := setup(t)
	f.subscribe("s1", "d1")
	f.publish("e1", invoice.Name)
	lease := f.lease()
	f.confirm(lease)
	expect(t, "an unknown status", second(f.r.Report("w1", lease.Delivery, lease.Token, Outcome{Status: "teapot"}, relay)), ErrInvalid)
	view := f.report(lease, StatusFailed, 0)
	if view.Attempts != 1 || view.State != JobWaiting {
		t.Errorf("a failure %+v", view)
	}
	expect(t, "a stale report", second(f.r.Report("w1", lease.Delivery, lease.Token, Outcome{Status: StatusDelivered}, relay)), ErrStale)
	if leases := f.dispatch(); len(leases) != 0 {
		t.Errorf("leased during backoff %+v", leases)
	}
	f.advance(time.Second)
	lease = f.lease()
	f.confirm(lease)
	for retry, want := range map[time.Duration]time.Duration{0: 0, -time.Second: 0, 500 * time.Millisecond: time.Second, 30 * time.Second: 30 * time.Second, time.Hour: time.Minute} {
		view := f.report(lease, StatusThrottled, retry)
		s := f.r.workspaces["w1"].subscriptions["s1"]
		if want > 0 && !s.throttle.Equal(f.now.Add(want)) || view.Attempts != 1 {
			t.Errorf("throttle %v: until %v, attempts %d", retry, s.throttle.Sub(*f.now), view.Attempts)
		}
		if leases := f.dispatch(); len(leases) != 0 {
			t.Errorf("leased while throttled %+v", leases)
		}
		f.advance(s.throttle.Sub(*f.now))
		lease = f.lease()
		f.confirm(lease)
	}
	f.report(lease, StatusUnknown, 0)
	f.advance(time.Minute)
	lease = f.lease()
	f.confirm(lease)
	if view := f.report(lease, StatusFailed, 0); view.Kind != KindDead || view.State != JobFailed || view.Reason != ReasonAttempts {
		t.Errorf("the attempt budget %+v", view)
	}
	f.publish("e2", invoice.Name)
	lease = f.lease()
	f.confirm(lease)
	if view := f.report(lease, StatusRejected, 0); view.Kind != KindDead || view.Reason != ReasonRejected {
		t.Errorf("a rejection %+v", view)
	}
	f.publish("e3", invoice.Name)
	lease = f.lease()
	f.confirm(lease)
	f.report(lease, StatusRejected, 0)
	if view, _ := f.r.SubscriptionOf("w1", "s1"); view.State != SubscriptionSuspended || view.Reason != ReasonBacklog {
		t.Errorf("a full dead-letter list %+v", view)
	}
	views, err := f.r.Deliveries("w1", "s1", 0, 10, relay)
	if err != nil || len(views) != 3 || views[0].Kind != KindOpen || views[0].State != JobWaiting {
		t.Errorf("listing %+v %v", views, err)
	}
	if page, _ := f.r.Deliveries("w1", "s1", 1, 1, ann); len(page) != 1 || page[0].Kind != KindDead {
		t.Errorf("a page %+v", page)
	}
	expect(t, "a negative page", second(f.r.Deliveries("w1", "s1", -1, 1, ann)), ErrInvalid)
	expect(t, "an empty page", second(f.r.Deliveries("w1", "s1", 0, 0, ann)), ErrInvalid)
	expect(t, "a list of no subscription", second(f.r.Deliveries("w1", "", 0, 1, ann)), ErrInvalid)
	reason(t, "an assistant listing", second(f.r.Deliveries("w1", "s1", 0, 1, bot)), ReasonAINotAllowed)
	expect(t, "an unknown subscription", second(f.r.Deliveries("w1", "s9", 0, 1, ann)), ErrNotFound)
}

func TestBackoff(t *testing.T) {
	f := setup(t)
	d := &delivery{id: "delivery-1"}
	previous := time.Duration(0)
	for step := 0; step < 10; step++ {
		d.step = step
		wait := f.r.backoff(d)
		base := min(limits.Backoff<<step, limits.MaxBackoff)
		if wait > base || wait < base*80/100 {
			t.Errorf("step %d: %v outside [%v, %v]", step, wait, base*80/100, base)
		}
		if step > 0 && base < limits.MaxBackoff && wait <= previous/2 {
			t.Errorf("step %d did not grow: %v after %v", step, wait, previous)
		}
		previous = wait
	}
	seen := map[time.Duration]bool{}
	for index := 0; index < 50; index++ {
		seen[f.r.backoff(&delivery{id: "delivery-" + strconv.Itoa(index), step: 5})] = true
	}
	if len(seen) < 5 {
		t.Errorf("the jitter does not vary: %v", seen)
	}
}

func TestExpiry(t *testing.T) {
	f := setup(t)
	f.subscribe("s1", "d1")
	f.publish("e1", invoice.Name)
	lease := f.lease()
	f.advance(time.Minute)
	expect(t, "confirming an expired lease", second(f.r.Confirm("w1", lease.Delivery, lease.Token, relay)), ErrStale)
	if leases := f.dispatch(); len(leases) != 0 {
		t.Errorf("leased right after an unconfirmed expiry %+v", leases)
	}
	f.advance(time.Second)
	again := f.lease()
	if s := f.r.workspaces["w1"].subscriptions["s1"]; s.open.attempts != 0 {
		t.Errorf("an unconfirmed expiry counted an attempt")
	}
	f.confirm(again)
	f.advance(time.Minute)
	if leases := f.dispatch(); len(leases) != 0 {
		t.Errorf("leased at the booked expiry %+v", leases)
	}
	expect(t, "a late report after booking", second(f.r.Report("w1", again.Delivery, again.Token, Outcome{Status: StatusDelivered}, relay)), ErrStale)
	if s := f.r.workspaces["w1"].subscriptions["s1"]; s.open.attempts != 1 {
		t.Errorf("a confirmed expiry was not booked as unknown: %d", s.open.attempts)
	}
	f.advance(2 * time.Second)
	late := f.lease()
	f.confirm(late)
	f.advance(time.Minute)
	if view := f.report(late, StatusDelivered, 0); view.State != JobSucceeded {
		t.Errorf("a late report before booking %+v", view)
	}
	f.publish("e2", invoice.Name)
	held := f.lease()
	f.confirm(held)
	f.report(held, StatusThrottled, time.Minute)
	f.advance(time.Hour)
	if leases := f.dispatch(); len(leases) != 0 {
		t.Errorf("leased after the age limit %+v", leases)
	}
	views, _ := f.r.Deliveries("w1", "s1", 0, 10, ann)
	if len(views) != 1 || views[0].Reason != ReasonMaxAge {
		t.Errorf("the age limit %+v", views)
	}
	l := limits
	l.Attempts = 1
	g := setupWith(t, l)
	g.subscribe("s1", "d1")
	g.publish("e1", invoice.Name)
	one := g.lease()
	g.confirm(one)
	g.advance(time.Minute)
	g.dispatch()
	if views, _ := g.r.Deliveries("w1", "s1", 0, 10, ann); len(views) != 1 || views[0].Reason != ReasonAttempts {
		t.Errorf("an expiry that used up the budget %+v", views)
	}
}

func TestDeadLetters(t *testing.T) {
	f := setup(t)
	f.subscribe("s1", "d1")
	f.publish("e1", invoice.Name)
	lease := f.lease()
	f.confirm(lease)
	f.report(lease, StatusRejected, 0)
	expect(t, "redeliver no letter", f.r.RequestRedeliver("w1", "s1", "", ann), ErrInvalid)
	expect(t, "redeliver an unknown letter", f.r.RequestRedeliver("w1", "s1", "x", ann), ErrNotFound)
	expect(t, "redeliver of an unknown subscription", f.r.RequestRedeliver("w1", "s9", lease.Delivery, ann), ErrNotFound)
	reason(t, "approving a redelivery never requested", second(f.r.ApproveRedeliver("w1", "s1", lease.Delivery, bob)), ReasonNotPending)
	f.must(f.r.RequestRedeliver("w1", "s1", lease.Delivery, ann))
	reason(t, "the requester approving", second(f.r.ApproveRedeliver("w1", "s1", lease.Delivery, ann)), ReasonSelfApproval)
	expect(t, "approve an unknown letter", second(f.r.ApproveRedeliver("w1", "s1", "x", bob)), ErrNotFound)
	fresh, err := f.r.ApproveRedeliver("w1", "s1", lease.Delivery, bob)
	if err != nil || fresh.ID == lease.Delivery || fresh.Kind != KindRedelivery || fresh.State != JobQueued {
		t.Fatalf("a redelivery %+v %v", fresh, err)
	}
	redelivered := f.lease()
	if redelivered.Delivery != fresh.ID || redelivered.EventID != "e1" {
		t.Errorf("the redelivery was not served first %+v", redelivered)
	}
	f.confirm(redelivered)
	f.report(redelivered, StatusRejected, 0)
	expect(t, "dismiss an unknown letter", f.r.RequestDismiss("w1", "s1", "x", ann), ErrNotFound)
	reason(t, "approving a dismissal never requested", f.r.ApproveDismiss("w1", "s1", fresh.ID, bob), ReasonNotPending)
	f.must(f.r.RequestDismiss("w1", "s1", fresh.ID, ann))
	reason(t, "the requester dismissing", f.r.ApproveDismiss("w1", "s1", fresh.ID, ann), ReasonSelfApproval)
	expect(t, "approve dismissal of an unknown letter", f.r.ApproveDismiss("w1", "s1", "x", bob), ErrNotFound)
	f.must(f.r.ApproveDismiss("w1", "s1", fresh.ID, bob))
	if views, _ := f.r.Deliveries("w1", "s1", 0, 10, ann); len(views) != 0 || len(f.r.workspaces["w1"].deliveries) != 0 {
		t.Errorf("after dismissal %+v", views)
	}
}

func TestGap(t *testing.T) {
	f := setup(t)
	f.subscribe("s1", "d1")
	f.publish("e1", invoice.Name)
	f.advance(time.Hour)
	f.publish("e2", invoice.Name)
	if leases := f.dispatch(); len(leases) != 0 {
		t.Errorf("leased across a gap %+v", leases)
	}
	view, _ := f.r.SubscriptionOf("w1", "s1")
	if view.State != SubscriptionSuspended || view.Reason != ReasonGap {
		t.Fatalf("a gap %+v", view)
	}
	f.publish("e3", invoice.Name)
	f.must(second(f.r.RequestResume("w1", "s1", ann)))
	resumed, err := f.r.ApproveResume("w1", "s1", bob)
	if err != nil || resumed.Cursor != 1 || len(resumed.Gaps) != 1 || resumed.Gaps[0] != (Gap{After: 0, To: 1}) {
		t.Fatalf("a resume after the gap %+v %v", resumed, err)
	}
	for _, want := range []string{"e2", "e3"} {
		lease := f.lease()
		if lease.EventID != want {
			t.Errorf("after the gap %s; want %s", lease.EventID, want)
		}
		f.confirm(lease)
		f.report(lease, StatusDelivered, 0)
	}
}

func TestFairness(t *testing.T) {
	l := limits
	l.Work, l.Batch = 1, 1
	f := setupWith(t, l)
	f.subscribe("s1", "d1", "never")
	f.subscribe("s2", "d1")
	for index := 0; index < 3; index++ {
		f.publish("e"+strconv.Itoa(index), invoice.Name)
	}
	served := map[string]bool{}
	for round := 0; round < 4; round++ {
		leases, err := f.r.Dispatch("w1", 1, relay)
		f.must(err)
		for _, lease := range leases {
			served[lease.Subscription] = true
			f.confirm(lease)
			f.report(lease, StatusDelivered, 0)
		}
	}
	if !served["s2"] {
		t.Errorf("a subscription was starved")
	}
	g := setup(t)
	g.subscribe("s1", "d1")
	g.subscribe("s2", "d2")
	g.publish("e1", invoice.Name)
	leases := g.dispatch()
	if len(leases) != 2 {
		t.Errorf("two subscriptions in one call %+v", leases)
	}
}

func TestConfirmRaces(t *testing.T) {
	f := setup(t)
	f.subscribe("s1", "d1")
	f.publish("e1", invoice.Name)
	lease := f.lease()
	hold, entered := make(chan struct{}), make(chan struct{}, 1)
	f.fk.set(func(k *fakes) { k.hold, k.entered = hold, entered })
	done := make(chan error, 1)
	go func() { done <- second(f.r.Confirm("w1", lease.Delivery, lease.Token, relay)) }()
	<-entered
	expect(t, "a second confirmation while confirming", second(f.r.Confirm("w1", lease.Delivery, lease.Token, relay)), ErrStale)
	expect(t, "a report while confirming", second(f.r.Report("w1", lease.Delivery, lease.Token, Outcome{Status: StatusDelivered}, relay)), ErrStale)
	f.must(second(f.r.ProposeSubscription("w2", Subscription{ID: "x", Subscriber: "a", Destination: "d1", Types: []string{"t"}}, ann)))
	f.must(second(f.r.PauseSubscription("w1", "s1", ann)))
	close(hold)
	expect(t, "a confirmation after a pause", <-done, ErrStale)
	f.fk.set(func(k *fakes) { k.hold = nil })
	f.must(second(f.r.RequestResume("w1", "s1", ann)))
	f.must(second(f.r.ApproveResume("w1", "s1", bob)))
	f.advance(time.Second)
	lease = f.lease()
	hold, entered = make(chan struct{}), make(chan struct{}, 1)
	f.fk.set(func(k *fakes) { k.hold, k.entered = hold, entered })
	go func() { done <- second(f.r.Confirm("w1", lease.Delivery, lease.Token, relay)) }()
	<-entered
	f.advance(time.Minute)
	close(hold)
	expect(t, "a confirmation past its deadline", <-done, ErrStale)
	f.fk.set(func(k *fakes) { k.hold = nil })
	if s := f.r.workspaces["w1"].subscriptions["s1"]; s.open.phase != phaseDue || s.open.attempts != 0 {
		t.Errorf("a late confirmation left %s with %d attempts", s.open.phase, s.open.attempts)
	}
	f.advance(time.Minute)
	lease = f.lease()
	f.must(second(f.r.RevokeSubscription("w1", "s1", ann)))
	expect(t, "a confirmation after revocation", second(f.r.Confirm("w1", lease.Delivery, lease.Token, relay)), ErrStale)
}

func TestStates(t *testing.T) {
	s := &subscription{view: SubscriptionView{State: SubscriptionActive}}
	cases := map[string]struct {
		d    *delivery
		want string
	}{
		"cancelled":           {&delivery{kind: KindCancelled}, JobCancelled},
		"dead":                {&delivery{kind: KindDead}, JobFailed},
		"delivered":           {&delivery{kind: KindOpen, reason: StatusDelivered}, JobSucceeded},
		"leased":              {&delivery{kind: KindOpen, phase: phaseLeased}, JobRunning},
		"a queued redelivery": {&delivery{kind: KindRedelivery, phase: phaseDue}, JobQueued},
		"in backoff":          {&delivery{kind: KindOpen, phase: phaseDue, step: 1}, JobWaiting},
		"due":                 {&delivery{kind: KindOpen, phase: phaseDue}, JobQueued},
	}
	for name, c := range cases {
		c.d.sub = s
		if c.d.kind == KindOpen || c.d.kind == "" {
			s.open = c.d
		}
		if got := state(c.d); got != c.want {
			t.Errorf("%s: %s; want %s", name, got, c.want)
		}
	}
	paused := &subscription{view: SubscriptionView{State: SubscriptionPaused}}
	held := &delivery{kind: KindOpen, phase: phaseDue, sub: paused}
	paused.open = held
	if state(held) != JobWaiting {
		t.Errorf("a held delivery is not waiting")
	}
}

func TestChangesDuringCallbacks(t *testing.T) {
	f := setup(t)
	f.must(second(f.r.ProposeSubscription("w1", Subscription{ID: "s1", Subscriber: "acme", Destination: "d1", Types: []string{invoice.Name}}, ann)))
	f.fk.set(func(k *fakes) {
		k.onState = func() {
			f.fk.set(func(k *fakes) { k.onState = nil })
			f.must(second(f.r.RevokeSubscription("w1", "s1", ann)))
		}
	})
	expect(t, "an approval while the subscription was revoked", second(f.r.ApproveSubscription("w1", "s1", bob)), ErrStale)
	f.subscribe("s2", "d1")
	f.must(second(f.r.PauseSubscription("w1", "s2", ann)))
	f.must(second(f.r.RequestResume("w1", "s2", ann)))
	f.fk.set(func(k *fakes) {
		k.onState = func() {
			f.fk.set(func(k *fakes) { k.onState = nil })
			f.must(second(f.r.RevokeSubscription("w1", "s2", ann)))
		}
	})
	expect(t, "a resume while the subscription was revoked", second(f.r.ApproveResume("w1", "s2", bob)), ErrStale)
	f.subscribe("s3", "d1")
	f.publish("e1", invoice.Name)
	f.out.onRead = func() {
		f.out.onRead = nil
		f.must(second(f.r.PauseSubscription("w1", "s3", ann)))
	}
	if leases := f.dispatch(); len(leases) != 0 {
		t.Errorf("a read for a paused subscription leased %+v", leases)
	}
	if view, _ := f.r.SubscriptionOf("w1", "s3"); view.Cursor != 0 {
		t.Errorf("a stale read moved the cursor to %d", view.Cursor)
	}
	if !errorsIs(Refusal{}, ErrRejected) {
		t.Errorf("a refusal does not unwrap")
	}
}

func TestCancelledHistory(t *testing.T) {
	f := setup(t)
	f.subscribe("s1", "d1")
	f.fk.set(func(k *fakes) { k.allowed = false })
	for index := 0; index < 3; index++ {
		f.publish("e"+strconv.Itoa(index), invoice.Name)
		lease := f.lease()
		reason(t, "a refused permission", second(f.r.Confirm("w1", lease.Delivery, lease.Token, relay)), ReasonPermissionRevoked)
	}
	views, _ := f.r.Deliveries("w1", "s1", 0, 10, ann)
	if len(views) != limits.Dead || views[0].EventID != "e1" {
		t.Errorf("the cancellation history %+v", views)
	}
}

func errorsIs(err, target error) bool { return errors.Is(err, target) }

func TestReviewRegressions(t *testing.T) {
	f := setup(t)
	f.subscribe("s1", "d1")
	f.must(second(f.r.PauseSubscription("w1", "s1", ann)))
	f.must(second(f.r.RequestResume("w1", "s1", ann)))
	f.must(second(f.r.RevokeSubscription("w1", "s1", ann)))
	reason(t, "a stale resume after revocation", second(f.r.ApproveResume("w1", "s1", bob)), ReasonNotPending)
	if view, _ := f.r.SubscriptionOf("w1", "s1"); view.State != SubscriptionRevoked || view.Resumer != "" {
		t.Errorf("a revoked subscription %+v", view)
	}
	l := limits
	l.Work = 1
	g := setupWith(t, l)
	g.subscribe("a", "d1", "never")
	g.subscribe("b", "d1")
	served := 0
	for round := 0; round < 6; round++ {
		for index := 0; index < limits.Batch; index++ {
			g.publish("o"+strconv.Itoa(round)+"-"+strconv.Itoa(index), "other")
		}
		g.publish("e"+strconv.Itoa(round), invoice.Name)
		leases, err := g.r.Dispatch("w1", 5, relay)
		g.must(err)
		for _, lease := range leases {
			served++
			g.confirm(lease)
			g.report(lease, StatusDelivered, 0)
		}
	}
	if served == 0 {
		t.Errorf("a subscription was starved behind the work bound")
	}
	h := setup(t)
	h.subscribe("s1", "d1")
	h.publish("e1", invoice.Name)
	first := h.lease()
	h.confirm(first)
	h.report(first, StatusFailed, 0)
	h.must(second(h.r.RevokeSubscription("w1", "s1", ann)))
	if views, _ := h.r.Deliveries("w1", "s1", 0, 10, ann); len(views) != 1 || !views[0].OutcomeUnknown {
		t.Errorf("a cancelled delivery after a confirmed attempt %+v", views)
	}
	k := setup(t)
	for index := 0; index < limits.Subscriptions; index++ {
		id := "s" + strconv.Itoa(index)
		k.subscribe(id, "d1")
		k.must(second(k.r.RevokeSubscription("w1", id, ann)))
	}
	for index := 0; index < 2*limits.Subscriptions; index++ {
		id := "n" + strconv.Itoa(index)
		if _, err := k.r.ProposeSubscription("w1", Subscription{ID: id, Subscriber: "acme", Destination: "d1", Types: []string{"t"}}, ann); err != nil {
			t.Fatalf("a proposal after revocations: %v", err)
		}
		k.must(second(k.r.RevokeSubscription("w1", id, ann)))
	}
	if len(k.r.workspaces["w1"].subscriptions) > 2*limits.Subscriptions {
		t.Errorf("revoked subscriptions were not purged: %d", len(k.r.workspaces["w1"].subscriptions))
	}
}

func TestMaxAgeWithFullLists(t *testing.T) {
	f := setup(t)
	f.subscribe("s1", "d1")
	for index := 0; index < 2; index++ {
		f.publish("e"+strconv.Itoa(index), invoice.Name)
		lease := f.lease()
		f.confirm(lease)
		f.report(lease, StatusRejected, 0)
	}
	f.publish("e2", invoice.Name)
	held := f.lease()
	f.confirm(held)
	f.report(held, StatusThrottled, time.Minute)
	f.advance(time.Hour)
	f.dispatch()
	view, _ := f.r.SubscriptionOf("w1", "s1")
	views, _ := f.r.Deliveries("w1", "s1", 0, 10, ann)
	if view.Reason != ReasonBacklog || views[0].Kind != KindOpen || views[0].Reason != ReasonMaxAge {
		t.Errorf("an aged delivery with full lists %+v %+v", view, views[0])
	}
	f.must(f.r.RequestDismiss("w1", "s1", views[1].ID, ann))
	f.must(f.r.RequestRedeliver("w1", "s1", views[1].ID, ann))
	f.must(f.r.ApproveDismiss("w1", "s1", views[1].ID, bob))
	if s := f.r.workspaces["w1"].subscriptions["s1"]; len(s.redeliver) != 0 || len(s.dismiss) != 0 {
		t.Errorf("request maps leaked %v %v", s.redeliver, s.dismiss)
	}
	if s := f.r.workspaces["w1"].subscriptions["s1"]; s.open != nil || len(s.dead) != 2 || s.dead[1].reason != ReasonMaxAge {
		t.Errorf("an aged delivery held by a full list was not buried after a dismissal")
	}
}

func TestSignatureVerifier(t *testing.T) {
	f := setup(t)
	f.subscribe("s1", "d1")
	f.publish("e1", invoice.Name)
	signed := f.confirm(f.lease())
	verify := func(h map[string]string, body []byte) bool {
		attempt, _ := strconv.Atoi(h["webhook-attempt"])
		at, _ := strconv.ParseInt(h["webhook-time"], 10, 64)
		d := &delivery{id: h["webhook-delivery"], sub: &subscription{view: SubscriptionView{Subscription: Subscription{ID: h["webhook-subscription"]}}}}
		d.event.Event.Source, d.event.Event.ID = h["webhook-event-source"], h["webhook-event-id"]
		sum := sha256.Sum256(append([]byte(h["webhook-key-version"]+"|"), message("w1", d, attempt, h["webhook-key-version"], time.Unix(0, at), body)...))
		return bytes.Equal(sum[:], signed.Signature)
	}
	if !verify(signed.Headers, signed.Body) {
		t.Fatal("the reference verifier rejects a genuine signature")
	}
	if hex.EncodeToString(signed.Signature) != knownSignature {
		t.Errorf("signature %x", signed.Signature)
	}
	for name := range signed.Headers {
		tampered := map[string]string{}
		for key, value := range signed.Headers {
			tampered[key] = value
		}
		tampered[name] += "x"
		if verify(tampered, signed.Body) {
			t.Errorf("a changed %s still verifies", name)
		}
	}
	if verify(signed.Headers, append(append([]byte(nil), signed.Body...), ' ')) {
		t.Errorf("a changed body still verifies")
	}
}

func TestConcurrentDispatch(t *testing.T) {
	f := setup(t)
	f.subscribe("s1", "d1")
	f.subscribe("s2", "d2")
	for index := 0; index < 5; index++ {
		f.publish("e"+strconv.Itoa(index), invoice.Name)
	}
	var wait sync.WaitGroup
	var mu sync.Mutex
	counts := map[string]int{}
	for worker := 0; worker < 8; worker++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			leases, err := f.r.Dispatch("w1", 5, relay)
			if err != nil {
				return
			}
			mu.Lock()
			for _, lease := range leases {
				counts[lease.Subscription]++
			}
			mu.Unlock()
		}()
	}
	wait.Wait()
	if counts["s1"] != 1 || counts["s2"] != 1 {
		t.Errorf("concurrent dispatches leased %v", counts)
	}
}

const knownSignature = "e180854a0c584f2bc3119c4812f2d61ae6f6d1c1b92060171f922e8ab57573ef"

func TestGapHistoryBound(t *testing.T) {
	l := limits
	l.Dead = 1
	f := setupWith(t, l)
	f.subscribe("s1", "d1")
	for round := 0; round < 2; round++ {
		f.publish("e"+strconv.Itoa(round), invoice.Name)
		f.advance(time.Hour)
		f.publish("f"+strconv.Itoa(round), "other")
		f.dispatch()
		f.must(second(f.r.RequestResume("w1", "s1", ann)))
		f.must(second(f.r.ApproveResume("w1", "s1", bob)))
	}
	if view, _ := f.r.SubscriptionOf("w1", "s1"); len(view.Gaps) != 1 || view.Gaps[0].After == 0 {
		t.Errorf("the gap history %+v", view.Gaps)
	}
}

func TestRoundTwoRegressions(t *testing.T) {
	l := limits
	l.Subscriptions = 1
	f := setupWith(t, l)
	for index := 0; index < 2; index++ {
		id := "s" + strconv.Itoa(index)
		f.subscribe(id, "d1")
		f.publish("e"+strconv.Itoa(index), invoice.Name)
		lease := f.lease()
		f.confirm(lease)
		f.report(lease, StatusRejected, 0)
		f.must(second(f.r.RevokeSubscription("w1", id, ann)))
	}
	expect(t, "a proposal when revoked subscriptions with letters fill the map", second(f.r.ProposeSubscription("w1", Subscription{ID: "s9", Subscriber: "acme", Destination: "d1", Types: []string{"t"}}, ann)), ErrLimit)
	views, _ := f.r.Deliveries("w1", "s0", 0, 10, ann)
	f.must(f.r.RequestRedeliver("w1", "s0", views[0].ID, ann))
	reason(t, "a redelivery on a revoked subscription", second(f.r.ApproveRedeliver("w1", "s0", views[0].ID, bob)), ReasonRevoked)
	g := setup(t)
	g.subscribe("s1", "d1")
	for index := 0; index < 3; index++ {
		g.publish("e"+strconv.Itoa(index), invoice.Name)
		lease := g.lease()
		g.confirm(lease)
		g.report(lease, StatusRejected, 0)
	}
	letters, _ := g.r.Deliveries("w1", "s1", 0, 10, ann)
	g.must(g.r.RequestDismiss("w1", "s1", letters[1].ID, ann))
	g.must(g.r.ApproveDismiss("w1", "s1", letters[1].ID, bob))
	after, _ := g.r.Deliveries("w1", "s1", 0, 10, ann)
	if len(after) != 2 || after[0].Kind != KindDead || after[1].Kind != KindDead || g.r.workspaces["w1"].subscriptions["s1"].open != nil {
		t.Errorf("a held rejected delivery was not moved to the dead letters %+v", after)
	}
	h := setup(t)
	h.subscribe("s1", "d1")
	h.publish("e1", invoice.Name)
	lease := h.lease()
	h.fk.set(func(k *fakes) {
		k.onState = func() {
			h.fk.set(func(k *fakes) { k.onState = nil })
			h.advance(3 * time.Second)
		}
	})
	if signed := h.confirm(lease); !signed.At.Equal(start.Add(3 * time.Second)) {
		t.Errorf("a stale attempt time %v", signed.At)
	}
}

func TestMutationGaps(t *testing.T) {
	f := setup(t)
	f.subscribe("s1", "d1")
	f.subscribe("s2", "d2")
	f.publish("e1", invoice.Name)
	if leases, err := f.r.Dispatch("w1", 1, relay); err != nil || len(leases) != 1 {
		t.Errorf("a dispatch limit of one %+v %v", leases, err)
	}
	expect(t, "confirming an invalid delivery", second(f.r.Confirm("w1", "a b", "t", relay)), ErrInvalid)
	expect(t, "confirming in an invalid workspace", second(f.r.Confirm("a b", "d", "t", relay)), ErrInvalid)
	g := setup(t)
	g.subscribe("s1", "d1")
	g.publish("e1", invoice.Name)
	lease := g.lease()
	g.confirm(lease)
	g.report(lease, StatusThrottled, 2*time.Second)
	if s := g.r.workspaces["w1"].subscriptions["s1"]; s.open.step != 1 {
		t.Errorf("a throttle did not advance the backoff step: %d", s.open.step)
	}
	g.must(second(g.r.PauseSubscription("w1", "s1", ann)))
	g.advance(time.Minute)
	if leases := g.dispatch(); len(leases) != 0 {
		t.Errorf("a due delivery of a paused subscription was leased %+v", leases)
	}
	l := limits
	l.Work = 1
	h := setupWith(t, l)
	h.subscribe("s1", "d1")
	h.subscribe("s2", "d2")
	h.publish("e1", "other")
	reads := h.out.reads
	h.dispatch()
	if h.out.reads-reads != 1 {
		t.Errorf("the work bound allowed %d reads", h.out.reads-reads)
	}
}
