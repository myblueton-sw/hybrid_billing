package commands

import (
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func solo(l Limits) Limits {
	l.Workspaces, l.Users = 1, 1
	l.WorkspaceJournal, l.UserJournal = l.Journal, l.Journal
	l.WorkspaceOperations, l.UserOperations = l.Operations, l.Operations
	return l
}

func refused(answer gate.Answer) Call {
	return Call{Answer: answer, Request: "r", Command: "nothing", Target: "t"}
}

func reading(answer gate.Answer) Call {
	return Call{Answer: answer, Request: "r", Command: "statement.read", Target: "t1"}
}

func TestNewLimits(t *testing.T) {
	f := setup(t)
	for name, c := range map[string]struct {
		change func(*Limits)
		ok     bool
	}{
		"no workspaces":             {func(l *Limits) { l.Workspaces = 0 }, false},
		"no users":                  {func(l *Limits) { l.Users = 0 }, false},
		"no workspace journal":      {func(l *Limits) { l.WorkspaceJournal = 0 }, false},
		"no user journal":           {func(l *Limits) { l.UserJournal = 0 }, false},
		"no workspace operations":   {func(l *Limits) { l.WorkspaceOperations = 0 }, false},
		"no user operations":        {func(l *Limits) { l.UserOperations = 0 }, false},
		"no burst":                  {func(l *Limits) { l.Burst = 0 }, false},
		"the largest burst":         {func(l *Limits) { l.Burst = maxBurst }, true},
		"a burst above the largest": {func(l *Limits) { l.Burst = maxBurst + 1 }, false},
		"workspaces times their journal": {func(l *Limits) {
			l.Workspaces, l.WorkspaceJournal, l.Journal, l.UserJournal, l.WorkspaceOperations, l.UserOperations = 4, 25, 100, 2, 250, 2
		}, true},
		"workspaces above the journal": {func(l *Limits) {
			l.Workspaces, l.WorkspaceJournal, l.Journal, l.UserJournal, l.WorkspaceOperations, l.UserOperations = 4, 26, 100, 2, 250, 2
		}, false},
		"workspaces times operations": {func(l *Limits) {
			l.Workspaces, l.WorkspaceOperations, l.Operations, l.UserOperations, l.WorkspaceJournal, l.UserJournal = 4, 25, 100, 2, 25000, 2
		}, true},
		"workspaces above the operations": {func(l *Limits) {
			l.Workspaces, l.WorkspaceOperations, l.Operations, l.UserOperations, l.WorkspaceJournal, l.UserJournal = 4, 26, 100, 2, 25000, 2
		}, false},
		"users times their journal":         {func(l *Limits) { l.Users, l.UserJournal, l.WorkspaceJournal = 4, 25, 100 }, true},
		"users above the workspace journal": {func(l *Limits) { l.Users, l.UserJournal, l.WorkspaceJournal = 4, 26, 100 }, false},
		"users times operations":            {func(l *Limits) { l.Users, l.UserOperations, l.WorkspaceOperations = 4, 25, 100 }, true},
		"users above workspace operations":  {func(l *Limits) { l.Users, l.UserOperations, l.WorkspaceOperations = 4, 26, 100 }, false},
		"the shortest replay":               {func(l *Limits) { l.Replay = minReplay }, true},
		"a replay below the shortest":       {func(l *Limits) { l.Replay = minReplay - 1 }, false},
		"the longest replay":                {func(l *Limits) { l.Replay = maxReplay }, true},
		"a replay above the longest":        {func(l *Limits) { l.Replay = maxReplay + 1 }, false},
		"the shortest retention":            {func(l *Limits) { l.Replay, l.Retention = minReplay, minRetention }, true},
		"a retention below the shortest":    {func(l *Limits) { l.Replay, l.Retention = minReplay, minRetention-1 }, false},
		"the longest retention":             {func(l *Limits) { l.Retention = maxRetention }, true},
		"a retention above the longest":     {func(l *Limits) { l.Retention = maxRetention + 1 }, false},
		"a retention equal to the replay":   {func(l *Limits) { l.Replay, l.Retention = 2*time.Hour, 2*time.Hour }, true},
		"a retention below the replay":      {func(l *Limits) { l.Replay, l.Retention = 2*time.Hour, 2*time.Hour-1 }, false},
		"the shortest refill":               {func(l *Limits) { l.Refill = minRefill }, true},
		"a refill below the shortest":       {func(l *Limits) { l.Refill = minRefill - 1 }, false},
		"the longest refill":                {func(l *Limits) { l.Refill = maxRefill }, true},
		"a refill above the longest":        {func(l *Limits) { l.Refill = maxRefill + 1 }, false},
	} {
		l := testLimits
		c.change(&l)
		if _, err := New(f.g, l); (err == nil) != c.ok {
			t.Errorf("%s: %v", name, err)
		}
	}
	for retry, want := range map[time.Duration]int64{0: 0, -time.Second: 0, time.Nanosecond: 1, time.Millisecond: 1, time.Millisecond + time.Nanosecond: 2, 1500 * time.Millisecond: 1500} {
		if got := (&Error{RetryAfter: retry}).RetryAfterMillis(); got != want {
			t.Errorf("%v: %d; want %d", retry, got, want)
		}
	}
	var none *Registry
	if none.Throttled("w1") != nil || none.Unrecorded("w1") != nil || none.JournalStart() != 0 {
		t.Errorf("an unconstructed registry answers")
	}
}

func TestRateLimits(t *testing.T) {
	l := testLimits
	l.Burst, l.Refill = 3, time.Second
	f := setupWith(t, l)
	f.member("ann")
	f.member("bob")
	f.grant("ann", gate.Issue, "c1")
	ann, bob := f.answer("ann"), f.answer("bob")
	for index := 0; index < 3; index++ {
		f.code("a call within the burst", f.c.Execute(refused(ann)), UnsupportedCommand)
	}
	records := len(f.c.Journal(0, 100))
	limited := f.code("a call beyond the burst", f.c.Execute(refused(ann)), RateLimited)
	if limited.Err.RetryAfter != time.Second || !limited.Err.Retry || limited.Err.Status != 429 || limited.Err.Next != NextRetry {
		t.Errorf("the refusal %+v", limited.Err)
	}
	if len(f.c.Journal(0, 100)) != records || len(f.c.Unrecorded("w1")) != 0 {
		t.Errorf("a rate-limited call left a record")
	}
	f.code("another user", f.c.Execute(refused(bob)), UnsupportedCommand)
	f.advance(999 * time.Millisecond)
	if again := f.c.Execute(refused(ann)); again.Err.Code != RateLimited || again.Err.RetryAfter != time.Millisecond {
		t.Errorf("just before the refill %+v", again.Err)
	}
	f.advance(time.Millisecond)
	f.code("at the refill", f.c.Execute(refused(ann)), UnsupportedCommand)
	f.code("after the refill", f.c.Execute(refused(ann)), RateLimited)
	f.advance(2500 * time.Millisecond)
	f.code("the first of two tokens", f.c.Execute(refused(ann)), UnsupportedCommand)
	f.code("the second of two tokens", f.c.Execute(refused(ann)), UnsupportedCommand)
	if partial := f.c.Execute(refused(ann)); partial.Err.Code != RateLimited || partial.Err.RetryAfter != 500*time.Millisecond {
		t.Errorf("a partial interval is lost %+v", partial.Err)
	}
	f.advance(time.Hour)
	ann = f.answer("ann")
	first := f.code("a mutation", f.c.Execute(adjust(ann, "k1", "1")), "")
	if replay := f.c.Execute(adjust(ann, "k1", "1")); !replay.Replay || replay.Operation != first.Operation {
		t.Errorf("replay %+v", replay)
	}
	f.code("replays take tokens", f.c.Execute(adjust(ann, "k1", "1")), "")
	f.code("the bucket is empty after replays", f.c.Execute(adjust(ann, "k1", "1")), RateLimited)
	throttled := f.c.Throttled("w1")
	if len(throttled) != 1 || throttled[0].User != "ann" || throttled[0].Count != 5 || !throttled[0].First.Equal(start) || !throttled[0].Last.Equal(*f.now) {
		t.Errorf("throttled %+v", throttled)
	}
	if len(f.c.Throttled("w9")) != 0 {
		t.Errorf("another workspace has throttled calls")
	}
}

func TestBucketClockAndCleanup(t *testing.T) {
	l := testLimits
	l.Burst, l.Refill = 2, time.Second
	f := setupWith(t, l)
	who := holder{"w1", "u1"}
	f.c.clock = func() time.Time { panic("synthetic clock failure") }
	if _, ok := f.c.take(who); !ok {
		t.Fatal("the first token")
	}
	f.c.take(who)
	if wait, ok := f.c.take(who); ok || wait != time.Second {
		t.Errorf("a failing clock %v %v", wait, ok)
	}
	if !f.c.buckets[who].last.IsZero() {
		t.Errorf("a failing clock moved the bucket")
	}
	other := holder{"w1", "u0"}
	f.c.take(other)
	f.c.take(other)
	f.c.take(other)
	if got := f.c.Throttled("w1"); len(got) != 2 || got[0].User != "u0" || !got[0].First.IsZero() {
		t.Errorf("throttled with a failing clock %+v", got)
	}
	now := start
	f.c.clock = func() time.Time { return now }
	back := holder{"w1", "back"}
	f.c.take(back)
	f.c.take(back)
	now = now.Add(-time.Hour)
	if wait, ok := f.c.take(back); ok || wait != time.Second || !f.c.buckets[back].last.Equal(now) {
		t.Errorf("a clock that went back %v %v", wait, ok)
	}
	now = now.Add(time.Second)
	if _, ok := f.c.take(back); !ok {
		t.Errorf("one interval after the clock went back")
	}
	now = start
	f.c.buckets = map[holder]*bucket{}
	f.c.cleaned = 1
	for _, user := range []string{"a", "b", "c", "c"} {
		f.c.take(holder{"w1", user})
	}
	if len(f.c.buckets) != 3 || f.c.cleaned != 2 {
		t.Errorf("%d buckets, cleaned %d", len(f.c.buckets), f.c.cleaned)
	}
	now = now.Add(time.Second)
	f.c.take(holder{"w1", "d"})
	f.c.take(holder{"w1", "e"})
	if len(f.c.buckets) != 3 || f.c.buckets[holder{"w1", "a"}] != nil || f.c.buckets[holder{"w1", "c"}] == nil || f.c.buckets[holder{"w1", "d"}] == nil {
		t.Errorf("cleanup kept %v, cleaned %d", f.c.buckets, f.c.cleaned)
	}
	f.c.buckets[holder{"w1", "f"}] = &bucket{tokens: 0, last: now.Add(time.Hour)}
	f.c.buckets[holder{"w1", "g"}] = &bucket{tokens: 2, last: now}
	f.c.cleaned = 1
	f.c.take(holder{"w1", "h"})
	if f.c.buckets[holder{"w1", "g"}] != nil || f.c.buckets[holder{"w1", "f"}] == nil {
		t.Errorf("cleanup of full and future buckets %v", f.c.buckets)
	}
}

func TestConcurrentBucket(t *testing.T) {
	l := testLimits
	l.Burst, l.Refill = 5, time.Hour
	f := setupWith(t, l)
	f.member("ann")
	ann := f.answer("ann")
	var wait sync.WaitGroup
	var mu sync.Mutex
	passed := 0
	for index := 0; index < 20; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			if got := f.c.Execute(refused(ann)); got.Err.Code != RateLimited {
				mu.Lock()
				passed++
				mu.Unlock()
			}
		}()
	}
	wait.Wait()
	if passed != 5 {
		t.Errorf("%d calls passed", passed)
	}
}

func TestJournalShares(t *testing.T) {
	l := testLimits
	l.Journal, l.Workspaces, l.Users, l.WorkspaceJournal, l.UserJournal = 4, 1, 2, 4, 2
	f := setupWith(t, l)
	f.member("ann")
	f.member("bob")
	f.member("cat")
	f.grant("ann", gate.View, "c1", "quantity")
	f.grant("bob", gate.View, "c1", "quantity")
	ann, bob, cat := f.answer("ann"), f.answer("bob"), f.answer("cat")
	f.code("ann's first record", f.c.Execute(reading(ann)), "")
	f.code("ann's second record", f.c.Execute(refused(ann)), UnsupportedCommand)
	resolves, _ := f.h.count()
	f.code("ann beyond her share", f.c.Execute(reading(ann)), DependencyUnavailable)
	f.code("ann's early refusal beyond her share", f.c.Execute(refused(ann)), UnsupportedCommand)
	if after, _ := f.h.count(); after != resolves {
		t.Errorf("an unrecorded call reached the handler")
	}
	f.code("bob within his share", f.c.Execute(reading(bob)), "")
	f.code("cat beyond the users", f.c.Execute(refused(cat)), UnsupportedCommand)
	f.code("cat's reading beyond the users", f.c.Execute(reading(cat)), DependencyUnavailable)
	f.code("bob's second record", f.c.Execute(refused(bob)), UnsupportedCommand)
	if records := f.c.Journal(0, 100); len(records) != 4 {
		t.Errorf("%d records", len(records))
	}
	unrecorded := f.c.Unrecorded("w1")
	if len(unrecorded) != 4 || unrecorded[0].User != "ann" || unrecorded[0].Code != DependencyUnavailable || unrecorded[1].Code != UnsupportedCommand ||
		unrecorded[2].User != "cat" || unrecorded[0].Count != 1 || !unrecorded[0].First.Equal(start) {
		t.Errorf("unrecorded %+v", unrecorded)
	}
	if len(f.c.Unrecorded("w9")) != 0 {
		t.Errorf("another workspace has unrecorded calls")
	}
	h := newHeld()
	h.add(holder{"w1", "a"})
	if h.room(holder{"w2", "a"}, Limits{Workspaces: 1, Users: 5}, 5, 5) || !h.room(holder{"w1", "b"}, Limits{Workspaces: 1, Users: 5}, 5, 5) {
		t.Errorf("the workspace count")
	}
	h.remove(holder{"w1", "a"})
	if len(h.workspaces) != 0 || len(h.members) != 0 || len(h.users) != 0 {
		t.Errorf("held counts stay %+v", h)
	}
}

func TestJournalRetention(t *testing.T) {
	l := testLimits
	l.Journal, l.Replay, l.Retention = 6, time.Hour, 2*time.Hour
	l = solo(l)
	f := setupWith(t, l)
	f.member("ann")
	ann := f.answer("ann")
	if f.c.JournalStart() != 1 {
		t.Errorf("an empty journal starts at %d", f.c.JournalStart())
	}
	f.code("the first record", f.c.Execute(refused(ann)), UnsupportedCommand)
	f.advance(time.Minute)
	f.code("the second record", f.c.Execute(refused(ann)), UnsupportedCommand)
	f.advance(l.Retention - time.Minute - time.Nanosecond)
	ann = f.answer("ann")
	f.code("a record just before the first expires", f.c.Execute(refused(ann)), UnsupportedCommand)
	if f.c.JournalStart() != 1 {
		t.Errorf("a record was swept before its retention")
	}
	f.advance(time.Nanosecond)
	f.code("a record when the first expires", f.c.Execute(refused(ann)), UnsupportedCommand)
	if f.c.JournalStart() != 2 || len(f.c.Journal(0, 100)) != 3 {
		t.Errorf("start %d, records %d", f.c.JournalStart(), len(f.c.Journal(0, 100)))
	}
	if page := f.c.Journal(2, 1); len(page) != 1 || page[0].Sequence != 3 {
		t.Errorf("a page after an evicted range %+v", page)
	}
	if page := f.c.Journal(0, 1); len(page) != 1 || page[0].Sequence != 2 {
		t.Errorf("the first page %+v", page)
	}
	if page := f.c.Journal(9, 1); len(page) != 0 {
		t.Errorf("a page after the end %+v", page)
	}
	f.advance(time.Minute)
	f.code("the second expires", f.c.Execute(refused(ann)), UnsupportedCommand)
	if f.c.head != 0 || len(f.c.journal) != 3 || f.c.JournalStart() != 3 {
		t.Errorf("the journal was not compacted: head %d, %d slots", f.c.head, len(f.c.journal))
	}
	f.advance(l.Retention)
	ann = f.answer("ann")
	f.code("all earlier records expire", f.c.Execute(refused(ann)), UnsupportedCommand)
	if f.c.JournalStart() != 6 || len(f.c.Journal(0, 100)) != 1 || f.c.reserved != 0 {
		t.Errorf("start %d, reserved %d", f.c.JournalStart(), f.c.reserved)
	}
}

func TestUntimedRecords(t *testing.T) {
	l := testLimits
	l.Replay, l.Retention = time.Hour, 2*time.Hour
	f := setupWith(t, l)
	f.member("ann")
	ann := f.answer("ann")
	broken := func() time.Time { panic("synthetic clock failure") }
	working := f.c.clock
	f.c.clock = broken
	f.code("a record without any time", f.c.Execute(refused(ann)), UnsupportedCommand)
	if !f.c.kept[0].IsZero() {
		t.Errorf("a time was invented")
	}
	f.c.clock = working
	f.advance(time.Minute)
	f.code("a timed record", f.c.Execute(refused(ann)), UnsupportedCommand)
	if !f.c.kept[0].Equal(start.Add(time.Minute)) {
		t.Errorf("the untimed record did not take the next time: %v", f.c.kept[0])
	}
	f.c.clock = broken
	f.code("a record after a timed one", f.c.Execute(refused(ann)), UnsupportedCommand)
	if !f.c.kept[2].Equal(start.Add(time.Minute)) {
		t.Errorf("the record did not take its predecessor's time: %v", f.c.kept[2])
	}
	f.c.lastKept = time.Time{}
	reserved := start.Add(2 * time.Minute)
	f.c.reserved++
	f.c.records.add(holder{"w1", "ann"})
	f.c.record(Record{Workspace: "w1", User: "ann"}, reserved)
	f.c.clock = working
	if !f.c.kept[3].Equal(reserved) {
		t.Errorf("the record did not take its reservation time: %v", f.c.kept[3])
	}
	f.advance(l.Retention + time.Hour)
	ann = f.answer("ann")
	f.code("the sweep passes the untimed records", f.c.Execute(refused(ann)), UnsupportedCommand)
	if f.c.JournalStart() != 5 {
		t.Errorf("start %d", f.c.JournalStart())
	}
}

func TestOperationWindow(t *testing.T) {
	l := testLimits
	l.Replay, l.Retention = time.Hour, 2*time.Hour
	f := setupWith(t, l)
	f.member("ann")
	f.grant("ann", gate.Issue, "c1")
	ann := f.answer("ann")
	first := f.code("the first call", f.c.Execute(adjust(ann, "k1", "1")), "")
	f.advance(l.Replay - time.Nanosecond)
	ann = f.answer("ann")
	if replay := f.c.Execute(adjust(ann, "k1", "1")); !replay.Replay || replay.Operation != first.Operation {
		t.Errorf("just before the replay window ends %+v", replay)
	}
	f.advance(time.Nanosecond)
	expired := f.code("at the end of the replay window", f.c.Execute(adjust(ann, "k1", "1")), IdempotencyKeyExpired)
	if expired.Err.Retry || expired.Err.Status != 409 || expired.Err.Next != NextInspect {
		t.Errorf("the expired key %+v", expired.Err)
	}
	f.code("another payload under an expired key", f.c.Execute(adjust(ann, "k1", "2")), IdempotencyConflict)
	_, applies := f.h.count()
	f.advance(l.Retention - l.Replay - time.Nanosecond)
	ann = f.answer("ann")
	f.code("just before the retention ends", f.c.Execute(adjust(ann, "k1", "1")), IdempotencyKeyExpired)
	f.advance(time.Nanosecond)
	again := adjust(ann, "k1", "1")
	again.Expected = f.h.targets["w1/t1"].Revision
	done := f.code("after the retention", f.c.Execute(again), "")
	if done.Replay || done.Operation == first.Operation {
		t.Errorf("an operation past its retention replayed %+v", done)
	}
	if _, after := f.h.count(); after != applies+1 {
		t.Errorf("%d applies", after-applies)
	}
	if replay := f.c.Execute(again); !replay.Replay || replay.Operation != done.Operation {
		t.Errorf("the new operation does not replay %+v", replay)
	}
	f.h.apply = func(Context, *Target) (Outcome, error) { panic("synthetic handler failure") }
	unknown := adjust(ann, "k9", "1")
	unknown.Expected = f.h.targets["w1/t1"].Revision
	failed := f.code("an unknown effect", f.c.Execute(unknown), InternalError)
	f.advance(l.Replay)
	unknown.Answer = f.answer("ann")
	if got := f.c.Execute(unknown); got.Err == nil || got.Err.Code != IdempotencyKeyExpired || failed.Err.Operation == "" {
		t.Errorf("an unknown effect after the replay window %+v", got)
	}
}

func TestOperationShares(t *testing.T) {
	l := testLimits
	l.Operations, l.Workspaces, l.Users, l.WorkspaceOperations, l.UserOperations = 2, 1, 2, 2, 1
	l.Replay, l.Retention = time.Hour, 2*time.Hour
	f := setupWith(t, l)
	f.member("ann")
	f.member("bob")
	f.member("cat")
	for _, user := range []string{"ann", "bob", "cat"} {
		f.grant(user, gate.Issue, "c1")
	}
	ann, bob, cat := f.answer("ann"), f.answer("bob"), f.answer("cat")
	f.code("ann's operation", f.c.Execute(adjust(ann, "k1", "1")), "")
	second := adjust(ann, "k2", "1")
	second.Expected = f.h.targets["w1/t1"].Revision
	f.code("ann beyond her share", f.c.Execute(second), DependencyUnavailable)
	third := adjust(bob, "k3", "1")
	third.Expected = f.h.targets["w1/t1"].Revision
	f.code("bob within his share", f.c.Execute(third), "")
	fourth := adjust(cat, "k4", "1")
	fourth.Expected = f.h.targets["w1/t1"].Revision
	f.code("cat beyond the users", f.c.Execute(fourth), DependencyUnavailable)
	f.advance(l.Replay)
	second.Answer, second.Expected = f.answer("ann"), f.h.targets["w1/t1"].Revision
	f.code("a tombstone keeps ann's share", f.c.Execute(second), DependencyUnavailable)
	f.advance(l.Retention - l.Replay)
	second.Answer = f.answer("ann")
	f.code("after the retention", f.c.Execute(second), "")
	if len(f.c.operations) != 1 || f.c.stored.users[holder{"w1", "bob"}] != 0 || f.c.stored.users[holder{"w1", "ann"}] != 1 {
		t.Errorf("operations %d, held %+v", len(f.c.operations), f.c.stored)
	}
	f.advance(l.Retention)
	reuse := adjust(f.answer("ann"), "k2", "1")
	reuse.Expected = f.h.targets["w1/t1"].Revision
	recreated := f.code("the same key past its retention", f.c.Execute(reuse), "")
	next := adjust(reuse.Answer, "k5", "1")
	next.Expected = f.h.targets["w1/t1"].Revision
	f.code("a full share", f.c.Execute(next), DependencyUnavailable)
	if replay := f.c.Execute(reuse); !replay.Replay || replay.Operation != recreated.Operation {
		t.Errorf("a sweep removed the re-created operation %+v", replay)
	}
	if f.c.first != 0 || len(f.c.order) != 1 {
		t.Errorf("the order was not compacted: first %d, %d entries", f.c.first, len(f.c.order))
	}
}

func TestUnrecordedReplays(t *testing.T) {
	l := testLimits
	l.Journal = 1
	f := setupWith(t, solo(l))
	f.member("ann")
	f.grant("ann", gate.Issue, "c1")
	ann := f.answer("ann")
	f.code("the only record", f.c.Execute(adjust(ann, "k1", "1")), "")
	f.code("a replay without room", f.c.Execute(adjust(ann, "k1", "1")), DependencyUnavailable)
	f.code("a conflict without room", f.c.Execute(adjust(ann, "k1", "2")), DependencyUnavailable)
	if got := f.c.Unrecorded("w1"); len(got) != 1 || got[0].Count != 2 {
		t.Errorf("unrecorded %+v", got)
	}
}

func TestUntimedOperations(t *testing.T) {
	l := testLimits
	l.Replay, l.Retention = time.Hour, 2*time.Hour
	f := setupWith(t, l)
	op := &operation{workspace: "w1", user: "ann"}
	f.c.store(operationKey{key: "a"}, op, time.Time{})
	if !op.created.IsZero() {
		t.Errorf("a time was invented")
	}
	f.c.store(operationKey{key: "b"}, &operation{workspace: "w1", user: "bob", created: start}, time.Time{})
	if !op.created.Equal(start) {
		t.Errorf("the untimed operation did not take the next time")
	}
	f.c.lastKept = start.Add(time.Minute)
	later := &operation{workspace: "w1", user: "cat"}
	f.c.store(operationKey{key: "c"}, later, time.Time{})
	reservedOnly := &operation{workspace: "w1", user: "dan"}
	f.c.store(operationKey{key: "d"}, reservedOnly, start.Add(2*time.Minute))
	if !reservedOnly.created.Equal(start.Add(2 * time.Minute)) {
		t.Errorf("the untimed operation did not take its reservation time")
	}
	if !later.created.Equal(start.Add(time.Minute)) {
		t.Errorf("the untimed operation did not take the last known time")
	}
	early := operationKey{key: "early"}
	f.c.store(operationKey{key: "late"}, &operation{workspace: "w1", user: "dan", created: start.Add(time.Hour)}, time.Time{})
	f.c.store(early, &operation{workspace: "w1", user: "eve", created: start.Add(-time.Hour)}, time.Time{})
	f.c.lastKept = time.Time{}
	if stored, room := f.c.lookup(early, holder{"w1", "eve"}, start.Add(l.Retention)); stored != nil || !room || f.c.operations[early] != nil {
		t.Errorf("an operation past its retention behind a live one was kept: %v %v", stored, room)
	}
	if f.c.expired(op, time.Time{}, time.Hour) || f.c.expired(&operation{}, start.Add(10*time.Hour), time.Hour) {
		t.Errorf("an unknown time expired an operation")
	}
}

func (f world) advance(d time.Duration) { *f.now = f.now.Add(d) }

func TestLongIdleAndSmallSteps(t *testing.T) {
	l := testLimits
	l.Burst, l.Refill = 2, time.Millisecond
	f := setupWith(t, l)
	now := start
	f.c.clock = func() time.Time { return now }
	who := holder{"w1", "idle"}
	f.c.take(who)
	f.c.take(who)
	now = now.Add(30 * 24 * time.Hour)
	if _, ok := f.c.take(who); !ok || f.c.buckets[who].tokens != 1 || !f.c.buckets[who].last.Equal(now) {
		t.Errorf("a thirty-day idle bucket did not refill: %+v", f.c.buckets[who])
	}
	f.c.take(who)
	now = now.Add(-time.Millisecond / 2)
	if wait, ok := f.c.take(who); ok || wait != time.Millisecond || !f.c.buckets[who].last.Equal(now.Add(time.Millisecond/2)) {
		t.Errorf("a step back smaller than one interval moved the bucket: %v %+v", wait, f.c.buckets[who])
	}
	now = now.Add(time.Millisecond / 2)
	f.c.buckets[holder{"w1", "full"}] = &bucket{tokens: 0, last: now.Add(-30 * 24 * time.Hour)}
	f.c.cleaned = 1
	f.c.take(holder{"w1", "fresh"})
	if f.c.buckets[holder{"w1", "full"}] != nil {
		t.Errorf("cleanup kept a bucket idle for thirty days")
	}
}

func TestWorkspaceIsolation(t *testing.T) {
	l := testLimits
	l.Journal, l.Workspaces, l.Users, l.WorkspaceJournal, l.UserJournal = 4, 2, 1, 2, 2
	l.Operations, l.WorkspaceOperations, l.UserOperations = 4, 2, 2
	l.Burst, l.Refill = 1, time.Hour
	f := setupWith(t, l)
	one, two, three := holder{"w1", "ann"}, holder{"w2", "ann"}, holder{"w3", "ann"}
	if _, ok := f.c.take(one); !ok {
		t.Fatal("w1")
	}
	if _, ok := f.c.take(one); ok {
		t.Errorf("w1's bucket is not empty")
	}
	if _, ok := f.c.take(two); !ok {
		t.Errorf("the same user in another workspace shares the bucket")
	}
	for index := 0; index < 2; index++ {
		if _, ok := f.c.reserve(one); !ok {
			t.Fatal("w1 reservation")
		}
		f.c.record(Record{Workspace: "w1", User: "ann"}, start)
	}
	if _, ok := f.c.reserve(one); ok {
		t.Errorf("w1 beyond its share")
	}
	if _, ok := f.c.reserve(two); !ok {
		t.Errorf("w2 is stopped by w1's full share")
	}
	f.c.record(Record{Workspace: "w2", User: "ann"}, start)
	if _, ok := f.c.reserve(three); ok {
		t.Errorf("a third workspace beyond the workspace count")
	}
	for index, key := range []string{"a", "b"} {
		if _, room := f.c.lookup(operationKey{key: key}, one, start); !room {
			t.Fatalf("w1 operation %d", index)
		}
		f.c.store(operationKey{key: key}, &operation{workspace: "w1", user: "ann", created: start}, start)
	}
	if _, room := f.c.lookup(operationKey{key: "c"}, one, start); room {
		t.Errorf("w1 operations beyond its share")
	}
	if _, room := f.c.lookup(operationKey{key: "c"}, two, start); !room {
		t.Errorf("w2 is stopped by w1's operations")
	}
	f.c.store(operationKey{key: "c"}, &operation{workspace: "w2", user: "ann", created: start}, start)
	if _, room := f.c.lookup(operationKey{key: "d"}, three, start); room {
		t.Errorf("a third workspace holds operations")
	}
	if f.c.records.workspaces["w1"] != 2 || f.c.records.workspaces["w2"] != 1 || f.c.stored.workspaces["w1"] != 2 || f.c.stored.members["w2"] != 1 {
		t.Errorf("held counts %+v %+v", f.c.records, f.c.stored)
	}
	f.advance(l.Retention)
	if _, ok := f.c.reserve(three); !ok {
		t.Errorf("a third workspace after the others' records expired")
	}
	if len(f.c.records.workspaces) != 1 || f.c.records.workspaces["w3"] != 1 {
		t.Errorf("held after expiry %+v", f.c.records)
	}
}

func TestStoreEdges(t *testing.T) {
	l := testLimits
	l.Burst, l.Refill = 1, time.Second
	f := setupWith(t, l)
	for _, c := range []struct {
		change func(*Limits)
		ok     bool
	}{
		{func(l *Limits) { l.Burst = 100000 }, true}, {func(l *Limits) { l.Burst = 100001 }, false},
		{func(l *Limits) { l.Replay, l.Retention = 720*time.Hour, 800*time.Hour }, true}, {func(l *Limits) { l.Replay, l.Retention = 720*time.Hour+1, 800*time.Hour }, false},
		{func(l *Limits) { l.Retention = 87600 * time.Hour }, true}, {func(l *Limits) { l.Retention = 87600*time.Hour + 1 }, false},
	} {
		limits := testLimits
		c.change(&limits)
		if _, err := New(f.g, limits); (err == nil) != c.ok {
			t.Errorf("limits %+v: %v", limits, err)
		}
	}
	now := start
	f.c.clock = func() time.Time { return now }
	who := holder{"w1", "edge"}
	if wait, ok := f.c.take(who); !ok || wait != 0 {
		t.Errorf("a granted token carries a wait %v", wait)
	}
	now = now.Add(-time.Second)
	if _, ok := f.c.take(who); ok || !f.c.buckets[who].last.Equal(now) {
		t.Errorf("a step back of exactly one interval was ignored: %+v", f.c.buckets[who])
	}
	f.c.buckets = map[holder]*bucket{{"w1", "x"}: {tokens: 1, last: now}, {"w1", "y"}: {tokens: 1, last: now}}
	f.c.cleaned = 1
	f.c.take(holder{"w1", "z"})
	if f.c.cleaned != 1 || len(f.c.buckets) != 1 {
		t.Errorf("cleanup of an empty set left the threshold at %d", f.c.cleaned)
	}
	f.c.journal = []Record{{Sequence: 1}, {Sequence: 2}}
	f.c.kept = []time.Time{start, now.Add(time.Hour)}
	f.c.head = 0
	f.c.sweepJournal(start.Add(testLimits.Retention))
	if f.c.head != 0 || len(f.c.journal) != 1 || f.c.journal[0].Sequence != 2 {
		t.Errorf("the journal was not compacted at exactly half: head %d, %d slots", f.c.head, len(f.c.journal))
	}
	f.c.journal = []Record{{Sequence: 1}, {Sequence: 2}, {Sequence: 3}}
	f.c.kept = []time.Time{start, now.Add(time.Hour), now.Add(time.Hour)}
	f.c.sweepJournal(start.Add(testLimits.Retention))
	if f.c.head != 1 || len(f.c.journal) != 3 {
		t.Errorf("the journal was compacted below half: head %d, %d slots", f.c.head, len(f.c.journal))
	}
	old := &operation{created: start}
	young := &operation{created: now.Add(time.Hour)}
	f.c.operations = map[operationKey]*operation{{key: "old"}: old, {key: "young"}: young}
	f.c.order = []ordered{{key: operationKey{key: "old"}, op: old}, {key: operationKey{key: "young"}, op: young}}
	f.c.first = 0
	f.c.stored = newHeld()
	f.c.stored.add(holder{})
	f.c.stored.add(holder{})
	f.c.sweepOperations(start.Add(testLimits.Retention))
	if f.c.first != 0 || len(f.c.order) != 1 {
		t.Errorf("the operations were not compacted at exactly half: first %d, %d entries", f.c.first, len(f.c.order))
	}
	f.c.order = append(f.c.order, ordered{key: operationKey{key: "y2"}, op: young}, ordered{key: operationKey{key: "y3"}, op: young})
	f.c.order[0] = ordered{key: operationKey{key: "old"}, op: old}
	f.c.operations[operationKey{key: "old"}] = old
	f.c.operations[operationKey{key: "y2"}], f.c.operations[operationKey{key: "y3"}] = young, young
	f.c.stored.add(holder{})
	f.c.sweepOperations(start.Add(testLimits.Retention))
	if f.c.first != 1 || len(f.c.order) != 3 {
		t.Errorf("the operations were compacted below half: first %d, %d entries", f.c.first, len(f.c.order))
	}
	f.c.aggregated = map[unrecordedKey]*Unrecorded{}
	for _, user := range []string{"b", "a", "c"} {
		for _, code := range []Code{UnsupportedCommand, DependencyUnavailable, InvalidRequest} {
			f.c.aggregated[unrecordedKey{"w1", user, code}] = &Unrecorded{User: user, Code: code}
		}
	}
	for round := 0; round < 30; round++ {
		got := f.c.Unrecorded("w1")
		for index := 1; index < len(got); index++ {
			if got[index-1].User > got[index].User || got[index-1].User == got[index].User && got[index-1].Code >= got[index].Code {
				t.Fatalf("unrecorded order %+v", got)
			}
		}
	}
}
