package commands

import (
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func TestAnUnavailableAuthorityIsOneAnswerForEveryTarget(t *testing.T) {
	f, ann := basic(t)
	bob := f.answer("bob")
	f.must(f.a.SetAvailable(false))
	missing := adjust(ann, "k1", "1")
	missing.Target = "t9"
	calls := map[string]Call{"an existing target": adjust(ann, "k1", "1"), "a missing target": missing,
		"an approval of an existing target": approve(bob, "a1", "t1"), "an approval of a missing target": approve(bob, "a1", "t9")}
	shapes := map[string]Record{}
	for name, call := range calls {
		got := f.code(name, f.c.Execute(call), AuthorityUnavailable)
		if !got.Err.Retry || got.Err.Next != NextRetry {
			t.Errorf("%s: an outage is not retryable: %+v", name, got.Err)
		}
		record := f.last()
		record.Sequence, record.Target, record.Digest = 0, "", ""
		shapes[name] = record
	}
	if shapes["an existing target"] != shapes["a missing target"] || shapes["an approval of an existing target"] != shapes["an approval of a missing target"] {
		t.Errorf("an outage tells an existing target from a missing one: %+v", shapes)
	}
	if _, applies := f.h.count(); applies != 0 {
		t.Errorf("%d applies during an outage", applies)
	}
	f.must(f.a.SetAvailable(true))
	if done := f.code("the same key once the authority is back", f.c.Execute(adjust(ann, "k1", "1")), ""); done.Replay {
		t.Errorf("an outage consumed the key")
	}
}

func TestAStoredFailureIsNotAliasedByWhatIsReturned(t *testing.T) {
	f, ann := basic(t)
	f.h.apply = func(Context, *Target) (Outcome, error) { panic("synthetic") }
	first := f.code("the failing call", f.c.Execute(adjust(ann, "k1", "1")), InternalError)
	operation := first.Err.Operation
	first.Err.Operation, first.Err.Code, first.Err.Next = "", PermissionDenied, NextNone
	again := f.code("the replay after the caller changed the first result", f.c.Execute(adjust(ann, "k1", "1")), InternalError)
	if again.Err.Operation != operation || again.Err.Next != NextOperation || again.Err == first.Err {
		t.Errorf("the returned error aliases the stored operation: %+v", again.Err)
	}
	again.Err.Operation = ""
	if third := f.c.Execute(adjust(ann, "k1", "1")); third.Err == nil || third.Err.Operation != operation {
		t.Errorf("a replayed error aliases the stored operation: %+v", third.Err)
	}
}

func TestRegistrationFollowsThePermissionCatalogue(t *testing.T) {
	f := setup(t)
	for name, permission := range map[string]Permission{
		"a kind the catalogue does not hold":                              {Action: gate.Issue, Kind: "invoice"},
		"an action that does not exist":                                   {Action: "erase", Kind: "statement"},
		"fields on an action without fields":                              {Action: gate.Issue, Kind: "statement", Fields: []string{"quantity"}},
		"a field action without fields":                                   {Action: gate.View, Kind: "statement"},
		"a field the catalogue does not hold":                             {Action: gate.View, Kind: "statement", Fields: []string{"margin"}},
		"an action only the gate's own approval of support sessions uses": {Action: gate.ApproveSupportSession, Kind: "statement"},
	} {
		if err := f.c.Register(Definition{ID: "statement.other", Permission: permission, Handler: f.h}); err != ErrInvalid {
			t.Errorf("%s: %v", name, err)
		}
	}
	var absent *stub
	f.must(f.c.Register(Definition{ID: "statement.absent", Permission: Permission{Action: gate.Issue, Kind: "statement"}, Mutating: true, Handler: absent}))
	for _, entry := range f.c.Catalogue() {
		if entry.ID == "statement.absent" && entry.Supported {
			t.Errorf("a handler that is a nil pointer is listed as supported")
		}
	}
	f.member("ann")
	call := Call{Answer: f.answer("ann"), Request: "r", Command: "statement.absent", Target: "t1", Key: "k", Expected: "r1"}
	f.code("a command whose handler is a nil pointer", f.c.Execute(call), UnsupportedCapability)
}

func TestAServiceUserCallsThroughTheSameSteps(t *testing.T) {
	f, _ := basic(t)
	f.must(f.w.CreateServiceUser(f.login("root"), "w1", "svc", "ann"))
	f.must(f.w.AddCredential(f.login("root"), "w1", "svc", "cred", v("cred"), identity.Validity{NotBefore: start, NotAfter: start.Add(time.Hour)}))
	service, err := f.b.AccessByCredential("cred", v("cred"))
	f.must(err)
	f.code("a service user without a grant", f.c.Execute(adjust(service, "k1", "1")), PermissionDenied)
	f.grant("svc", gate.Issue, "c1")
	done := f.code("a service user with the grant", f.c.Execute(adjust(service, "k1", "1")), "")
	if record := f.last(); record.User != "svc" || record.Operation != done.Operation {
		t.Errorf("the record of a service call: %+v", record)
	}
	f.code("an approval by a service user", f.c.Execute(approve(service, "a1", "t2")), PermissionDenied)
	f.must(f.w.RevokeCredential(f.login("root"), "w1", "cred"))
	f.code("a call after the credential is revoked", f.c.Execute(adjust(service, "k2", "1")), Unauthenticated)
	if _, applies := f.h.count(); applies != 1 {
		t.Errorf("%d applies", applies)
	}
}

func TestAResolutionOutsideTheListIsAMissingTarget(t *testing.T) {
	f, ann := basic(t)
	f.h.resolution = 9
	f.code("a resolution the executor does not know", f.c.Execute(adjust(ann, "k1", "1")), PermissionDenied)
	if _, applies := f.h.count(); applies != 0 {
		t.Errorf("the handler ran for an unknown resolution")
	}
	f.h.resolution = 0
	if done := f.code("the same key afterwards", f.c.Execute(adjust(ann, "k1", "1")), ""); done.Replay {
		t.Errorf("an unknown resolution consumed the key")
	}
}

func TestAFailureOutsideTheHandlerLeavesNothingBehind(t *testing.T) {
	limits := testLimits
	limits.Journal = 3
	f := setupWith(t, limits)
	f.member("ann")
	f.grant("ann", gate.Issue, "c1")
	ann := f.answer("ann")
	*f.hook = func() { panic("synthetic clock failure") }
	*f.armed = 8
	got := f.code("the clock fails after the recheck and before the apply", f.c.Execute(adjust(ann, "k1", "1")), InternalError)
	*f.hook = nil
	if resolves, applies := f.h.count(); resolves != 1 || applies != 0 || *f.armed != 0 || got.Err.Operation != "" {
		t.Errorf("%d resolves, %d applies, %d reads left, operation %q", resolves, applies, *f.armed, got.Err.Operation)
	}
	if record := f.last(); record.Outcome != string(InternalError) || record.Operation != "" {
		t.Errorf("the failure is not recorded: %+v", record)
	}
	done := f.code("the same key afterwards", f.c.Execute(adjust(ann, "k1", "1")), "")
	if done.Replay || len(f.c.Journal(0, 100)) != 2 {
		t.Errorf("the failure kept the key, the lock or a journal slot: %+v", done)
	}
	*f.hook = func() { panic("synthetic clock failure") }
	*f.armed = 9
	next := adjust(ann, "k2", "1")
	next.Expected = done.Revision
	last := f.code("the clock fails while the record is written", f.c.Execute(next), "")
	*f.hook = nil
	records := f.c.Journal(0, 100)
	if *f.armed != 0 || len(records) != 3 || !records[2].At.IsZero() || records[2].Operation != last.Operation || records[1].At.IsZero() {
		t.Errorf("an applied action lost its record or its result: %d reads left, %+v", *f.armed, records)
	}
	f.c.mu.Lock()
	reserved := f.c.reserved
	f.c.mu.Unlock()
	if reserved != 0 {
		t.Errorf("%d journal slots stay reserved", reserved)
	}
	if again := f.c.Execute(next); again.Err == nil || again.Err.Code != DependencyUnavailable {
		t.Errorf("the journal of three records is not full, or the lock is held: %+v", again)
	}
}

type overlapping struct {
	active  atomic.Int64
	overlap atomic.Bool
	applies atomic.Int64
	spawn   func()
}

func (o *overlapping) Resolve(string, string) (Target, Resolution) {
	if o.active.Load() > 0 {
		o.overlap.Store(true)
	}
	return Target{CustomerID: "c1", Revision: "r1"}, Found
}

func (o *overlapping) Apply(Context) (Outcome, error) {
	if o.active.Add(1) > 1 {
		o.overlap.Store(true)
	}
	if o.applies.Add(1) == 1 {
		o.spawn()
		time.Sleep(100 * time.Millisecond)
	}
	o.active.Add(-1)
	return Outcome{ResultRevision: "r2", BusinessState: "recorded"}, nil
}

func TestMutatingCallsWithDifferentKeysDoNotOverlapFromResolveToApply(t *testing.T) {
	f := setup(t)
	f.member("ann")
	f.grant("ann", gate.Deliver, "c1")
	handler := &overlapping{}
	f.must(f.c.Register(Definition{ID: "statement.send", Permission: Permission{Action: gate.Deliver, Kind: "statement"}, Mutating: true, Handler: handler}))
	ann, again := f.answer("ann"), f.answer("ann")
	call := func(answer gate.Answer, key string) Call {
		return Call{Answer: answer, Request: "r", Command: "statement.send", Target: "t1", Key: key, Expected: "r1"}
	}
	concurrent := make(chan Result, 1)
	handler.spawn = func() { go func() { concurrent <- f.c.Execute(call(again, "k2")) }() }
	f.code("the first call", f.c.Execute(call(ann, "k1")), "")
	f.code("the call sent while the first was applying", <-concurrent, "")
	if handler.overlap.Load() || handler.applies.Load() != 2 {
		t.Errorf("a second mutating call resolved or applied during the first: %d applies", handler.applies.Load())
	}
	records := f.c.Journal(0, 10)
	if len(records) != 2 || !reflect.DeepEqual([]uint64{records[0].Sequence, records[1].Sequence}, []uint64{1, 2}) || records[0].Operation == records[1].Operation {
		t.Errorf("records %+v", records)
	}
}

func TestAReplayIsRecordedAfterItsOriginal(t *testing.T) {
	f, ann := basic(t)
	again := f.answer("ann")
	concurrent := make(chan Result, 1)
	*f.hook = func() {
		go func() { concurrent <- f.c.Execute(adjust(again, "k1", "1")) }()
		select {
		case replay := <-concurrent:
			concurrent <- replay
		case <-time.After(200 * time.Millisecond):
		}
	}
	*f.armed = 9
	first := f.code("the first call, whose record is being written", f.c.Execute(adjust(ann, "k1", "1")), "")
	second := f.code("the same call sent while that record is written", <-concurrent, "")
	records := f.c.Journal(0, 10)
	if *f.armed != 0 || first.Replay || !second.Replay || len(records) != 2 || records[0].Replay || !records[1].Replay {
		t.Errorf("the replay is not recorded after its original: %d reads left, %+v", *f.armed, records)
	}
}
