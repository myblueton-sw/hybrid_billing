package webhooks

import "testing"

var cat = Actor{ID: "cat", Kind: KindHuman}

func (f world) letter(id string) string {
	f.t.Helper()
	f.subscribe(id, "d1")
	f.publish("e-"+id, invoice.Name)
	lease := f.lease()
	f.confirm(lease)
	f.report(lease, StatusRejected, 0)
	return lease.Delivery
}

func TestSubscriptionWithdrawAndClose(t *testing.T) {
	f := setup(t)
	_, err := f.r.ProposeSubscription("w1", Subscription{ID: "s1", Subscriber: "acme", Destination: "d1", Types: []string{invoice.Name}}, ann)
	f.must(err)
	reason(t, "another person withdraws", second(f.r.WithdrawSubscription("w1", "s1", bob)), ReasonNotRequester)
	reason(t, "an assistant withdraws", second(f.r.WithdrawSubscription("w1", "s1", Actor{ID: "ai", Kind: KindAssistant})), ReasonAINotAllowed)
	expect(t, "an unknown subscription", second(f.r.WithdrawSubscription("w1", "s9", ann)), ErrNotFound)
	got, err := f.r.WithdrawSubscription("w1", "s1", ann)
	if err != nil || got.State != SubscriptionRevoked || got.Reason != ReasonWithdrawn {
		t.Fatalf("withdrawn %+v %v", got, err)
	}
	reason(t, "withdrawn twice", second(f.r.WithdrawSubscription("w1", "s1", ann)), ReasonNotPending)
	reason(t, "approve after withdrawal", second(f.r.ApproveSubscription("w1", "s1", bob)), ReasonNotPending)
	_, err = f.r.ProposeSubscription("w1", Subscription{ID: "s2", Subscriber: "acme", Destination: "d1", Types: []string{invoice.Name}}, ann)
	f.must(err)
	expect(t, "close for another reason", second(f.r.CloseSubscription("w1", "s2", ReasonWithdrawn)), ErrInvalid)
	expect(t, "close an unknown subscription", second(f.r.CloseSubscription("w1", "s9", ReasonRequesterLostRight)), ErrNotFound)
	got, err = f.r.CloseSubscription("w1", "s2", ReasonRequesterLostRight)
	if err != nil || got.State != SubscriptionRevoked || got.Reason != ReasonRequesterLostRight {
		t.Fatalf("closed %+v %v", got, err)
	}
	reason(t, "closed twice", second(f.r.CloseSubscription("w1", "s2", ReasonRequesterLostRight)), ReasonNotPending)
	f.subscribe("s3", "d1")
	reason(t, "withdraw an active subscription", second(f.r.WithdrawSubscription("w1", "s3", ann)), ReasonNotPending)
	reason(t, "close an active subscription", second(f.r.CloseSubscription("w1", "s3", ReasonRequesterLostRight)), ReasonNotPending)
}

func TestResumeWithdrawAndClose(t *testing.T) {
	f := setup(t)
	f.subscribe("s1", "d1")
	reason(t, "withdraw with no request", second(f.r.WithdrawResume("w1", "s1", ann)), ReasonNotPending)
	_, err := f.r.PauseSubscription("w1", "s1", ann)
	f.must(err)
	reason(t, "close with no request", second(f.r.CloseResume("w1", "s1", ReasonRequesterLostRight)), ReasonNotPending)
	_, err = f.r.RequestResume("w1", "s1", ann)
	f.must(err)
	reason(t, "another person withdraws", second(f.r.WithdrawResume("w1", "s1", bob)), ReasonNotRequester)
	got, err := f.r.WithdrawResume("w1", "s1", ann)
	if err != nil || got.Resumer != "" || got.State != SubscriptionPaused {
		t.Fatalf("withdrawn %+v %v", got, err)
	}
	reason(t, "approve after withdrawal", second(f.r.ApproveResume("w1", "s1", bob)), ReasonNotPending)
	_, err = f.r.RequestResume("w1", "s1", ann)
	f.must(err)
	expect(t, "close for another reason", second(f.r.CloseResume("w1", "s1", ReasonWithdrawn)), ErrInvalid)
	got, err = f.r.CloseResume("w1", "s1", ReasonRequesterLostRight)
	if err != nil || got.Resumer != "" || got.State != SubscriptionPaused {
		t.Fatalf("closed %+v %v", got, err)
	}
	expect(t, "close on an unknown subscription", second(f.r.CloseResume("w1", "s9", ReasonRequesterLostRight)), ErrNotFound)
}

func TestLetterRequestWithdrawAndClose(t *testing.T) {
	f := setup(t)
	letter := f.letter("s1")
	reason(t, "withdraw with no request", f.r.WithdrawRequest("w1", "s1", letter, false, ann), ReasonNotPending)
	f.must(f.r.RequestRedeliver("w1", "s1", letter, ann))
	f.must(f.r.RequestDismiss("w1", "s1", letter, bob))
	reason(t, "withdraw another kind's request", f.r.WithdrawRequest("w1", "s1", letter, true, ann), ReasonNotRequester)
	reason(t, "withdraw another person's request", f.r.WithdrawRequest("w1", "s1", letter, false, bob), ReasonNotRequester)
	reason(t, "an assistant withdraws", f.r.WithdrawRequest("w1", "s1", letter, false, Actor{ID: "ai", Kind: KindAssistant}), ReasonAINotAllowed)
	expect(t, "an invalid actor", f.r.WithdrawRequest("w1", "s1", letter, false, Actor{}), ErrInvalid)
	expect(t, "an unknown letter", f.r.WithdrawRequest("w1", "s1", "x", false, ann), ErrNotFound)
	expect(t, "an invalid letter", f.r.WithdrawRequest("w1", "s1", "", false, ann), ErrInvalid)
	expect(t, "an unknown subscription", f.r.WithdrawRequest("w1", "s9", letter, false, ann), ErrNotFound)
	f.must(f.r.WithdrawRequest("w1", "s1", letter, false, ann))
	reason(t, "approve a withdrawn redelivery", second(f.r.ApproveRedeliver("w1", "s1", letter, bob)), ReasonNotPending)
	expect(t, "close for another reason", f.r.CloseRequest("w1", "s1", letter, true, ReasonWithdrawn), ErrInvalid)
	f.must(f.r.CloseRequest("w1", "s1", letter, true, ReasonRequesterLostRight))
	reason(t, "approve a closed dismissal", f.r.ApproveDismiss("w1", "s1", letter, ann), ReasonNotPending)
	reason(t, "close twice", f.r.CloseRequest("w1", "s1", letter, true, ReasonRequesterLostRight), ReasonNotPending)
}

func TestHeldRedeliveriesAreNotLeased(t *testing.T) {
	f := setup(t)
	letter := f.letter("s1")
	f.must(f.r.RequestRedeliver("w1", "s1", letter, ann))
	fresh, err := f.r.ApproveRedeliver("w1", "s1", letter, bob)
	f.must(err)
	if fresh.Requester != "ann" || fresh.Approver != "bob" {
		t.Errorf("the redelivery lost its people %+v", fresh)
	}
	expect(t, "hold for another reason", second(f.r.HoldRedelivery("w1", "s1", fresh.ID, ReasonRequesterLostRight)), ErrInvalid)
	expect(t, "hold an unknown delivery", second(f.r.HoldRedelivery("w1", "s1", "x", ReasonApproverLostRight)), ErrNotFound)
	expect(t, "hold on an unknown subscription", second(f.r.HoldRedelivery("w1", "s9", fresh.ID, ReasonApproverLostRight)), ErrNotFound)
	expect(t, "hold a dead letter", second(f.r.HoldRedelivery("w1", "s1", letter, ReasonApproverLostRight)), ErrNotFound)
	held, err := f.r.HoldRedelivery("w1", "s1", fresh.ID, ReasonApproverLostRight)
	if err != nil || held.Hold != ReasonApproverLostRight || held.State != JobQueued {
		t.Fatalf("held %+v %v", held, err)
	}
	if leases := f.dispatch(); len(leases) != 0 {
		t.Fatalf("a held redelivery was leased %+v", leases)
	}
	reason(t, "the requester re-approves", second(f.r.ReapproveRedelivery("w1", "s1", fresh.ID, ann)), ReasonSelfApproval)
	reason(t, "the lost approver re-approves", second(f.r.ReapproveRedelivery("w1", "s1", fresh.ID, bob)), ReasonSelfApproval)
	reason(t, "an assistant re-approves", second(f.r.ReapproveRedelivery("w1", "s1", fresh.ID, Actor{ID: "ai", Kind: KindAssistant})), ReasonAINotAllowed)
	expect(t, "an invalid actor", second(f.r.ReapproveRedelivery("w1", "s1", fresh.ID, Actor{})), ErrInvalid)
	again, err := f.r.ReapproveRedelivery("w1", "s1", fresh.ID, cat)
	if err != nil || again.Hold != "" || again.Approver != "cat" {
		t.Fatalf("re-approved %+v %v", again, err)
	}
	reason(t, "re-approve a released redelivery", second(f.r.ReapproveRedelivery("w1", "s1", fresh.ID, cat)), ReasonNotHeld)
	lease := f.lease()
	if lease.Delivery != fresh.ID {
		t.Errorf("the released redelivery was not leased %+v", lease)
	}
	expect(t, "hold a leased redelivery", second(f.r.HoldRedelivery("w1", "s1", fresh.ID, ReasonApproverLostRight)), ErrNotFound)
}

func TestAHeldOpenRedeliveryReturnsToTheQueue(t *testing.T) {
	f := setup(t)
	letter := f.letter("s1")
	f.must(f.r.RequestRedeliver("w1", "s1", letter, ann))
	fresh, err := f.r.ApproveRedeliver("w1", "s1", letter, bob)
	f.must(err)
	s := f.r.workspaces["w1"].subscriptions["s1"]
	s.open, s.queue = s.queue[0], nil
	f.r.workspaces["w1"].deliveries[s.open.id] = s.open
	f.must(second(f.r.HoldRedelivery("w1", "s1", fresh.ID, ReasonApproverLostRight)))
	if s.open != nil || len(s.queue) != 1 || f.r.workspaces["w1"].deliveries[fresh.ID] != nil {
		t.Fatalf("the held redelivery stayed open %+v", s.queue)
	}
	f.publish("e-next", invoice.Name)
	for _, lease := range f.dispatch() {
		if lease.Delivery == fresh.ID {
			t.Errorf("a held redelivery was leased")
		}
	}
}

func TestClosedRedeliveriesAreCancelled(t *testing.T) {
	f := setup(t)
	letter := f.letter("s1")
	f.must(f.r.RequestRedeliver("w1", "s1", letter, ann))
	fresh, err := f.r.ApproveRedeliver("w1", "s1", letter, bob)
	f.must(err)
	expect(t, "close for another reason", second(f.r.CloseRedelivery("w1", "s1", fresh.ID, ReasonApproverLostRight)), ErrInvalid)
	f.must(second(f.r.HoldRedelivery("w1", "s1", fresh.ID, ReasonApproverLostRight)))
	closed, err := f.r.CloseRedelivery("w1", "s1", fresh.ID, ReasonRequesterLostRight)
	if err != nil || closed.Kind != KindCancelled || closed.Reason != ReasonRequesterLostRight || closed.Hold != "" || closed.State != JobCancelled {
		t.Fatalf("closed %+v %v", closed, err)
	}
	if leases := f.dispatch(); len(leases) != 0 {
		t.Fatalf("a closed redelivery was leased %+v", leases)
	}
	expect(t, "close twice", second(f.r.CloseRedelivery("w1", "s1", fresh.ID, ReasonRequesterLostRight)), ErrNotFound)
	views, err := f.r.Deliveries("w1", "s1", 0, 10, ann)
	f.must(err)
	listed := false
	for _, v := range views {
		listed = listed || v.ID == fresh.ID && v.Kind == KindCancelled
	}
	if !listed {
		t.Errorf("the closed redelivery is not listed %+v", views)
	}
}

func TestReapproveOnARevokedSubscription(t *testing.T) {
	f := setup(t)
	letter := f.letter("s1")
	f.must(f.r.RequestRedeliver("w1", "s1", letter, ann))
	fresh, err := f.r.ApproveRedeliver("w1", "s1", letter, bob)
	f.must(err)
	f.must(second(f.r.HoldRedelivery("w1", "s1", fresh.ID, ReasonApproverLostRight)))
	s := f.r.workspaces["w1"].subscriptions["s1"]
	s.view.State = SubscriptionRevoked
	reason(t, "re-approve on a revoked subscription", second(f.r.ReapproveRedelivery("w1", "s1", fresh.ID, cat)), ReasonRevoked)
}

func TestRequestOperationsRefuseBadInput(t *testing.T) {
	f := setup(t)
	letter := f.letter("s1")
	expect(t, "an invalid workspace", second(f.r.CloseSubscription("", "s1", ReasonRequesterLostRight)), ErrInvalid)
	expect(t, "an unknown workspace", second(f.r.CloseResume("w9", "s1", ReasonRequesterLostRight)), ErrNotFound)
	expect(t, "withdraw resume on an unknown subscription", second(f.r.WithdrawResume("w1", "s9", ann)), ErrNotFound)
	expect(t, "an invalid delivery", second(f.r.HoldRedelivery("w1", "s1", "", ReasonApproverLostRight)), ErrInvalid)
	expect(t, "re-approve on an unknown subscription", second(f.r.ReapproveRedelivery("w1", "s9", "x", cat)), ErrNotFound)
	expect(t, "close request on an unknown subscription", f.r.CloseRequest("w1", "s9", letter, false, ReasonRequesterLostRight), ErrNotFound)
}

func TestClosedRedeliveriesKeepABoundedHistory(t *testing.T) {
	f := setup(t)
	letter := f.letter("s1")
	s := f.r.workspaces["w1"].subscriptions["s1"]
	for n := 0; n < limits.Dead+1; n++ {
		s.cancelled = append(s.cancelled, &delivery{id: "old-" + string(rune('a'+n)), sub: s, kind: KindCancelled})
	}
	f.must(f.r.RequestRedeliver("w1", "s1", letter, ann))
	fresh, err := f.r.ApproveRedeliver("w1", "s1", letter, bob)
	f.must(err)
	f.must(second(f.r.CloseRedelivery("w1", "s1", fresh.ID, ReasonRequesterLostRight)))
	if len(s.cancelled) != limits.Dead+1 || s.cancelled[len(s.cancelled)-1].id != fresh.ID {
		t.Errorf("history %d", len(s.cancelled))
	}
}

func TestResumeRequestsNeedAPausedOrSuspendedSubscription(t *testing.T) {
	f := setup(t)
	f.subscribe("s1", "d1")
	s := f.r.workspaces["w1"].subscriptions["s1"]
	s.view.Resumer = "ann"
	reason(t, "an active subscription", second(f.r.WithdrawResume("w1", "s1", ann)), ReasonNotPending)
	reason(t, "close on an active subscription", second(f.r.CloseResume("w1", "s1", ReasonRequesterLostRight)), ReasonNotPending)
	s.view.State = SubscriptionSuspended
	if got, err := f.r.WithdrawResume("w1", "s1", ann); err != nil || got.Resumer != "" {
		t.Errorf("suspended %+v %v", got, err)
	}
}

func TestClosedRedeliveriesTrimOnlyBeyondTheBound(t *testing.T) {
	f := setup(t)
	letter := f.letter("s1")
	s := f.r.workspaces["w1"].subscriptions["s1"]
	for n := 0; n < limits.Dead-1; n++ {
		s.cancelled = append(s.cancelled, &delivery{id: "old-" + string(rune('a'+n)), sub: s, kind: KindCancelled})
	}
	f.must(f.r.RequestRedeliver("w1", "s1", letter, ann))
	fresh, err := f.r.ApproveRedeliver("w1", "s1", letter, bob)
	f.must(err)
	f.must(second(f.r.CloseRedelivery("w1", "s1", fresh.ID, ReasonRequesterLostRight)))
	if len(s.cancelled) != limits.Dead {
		t.Errorf("history %d", len(s.cancelled))
	}
}
