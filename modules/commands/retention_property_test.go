package commands

import (
	"errors"
	"math/rand"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

type heldBucket struct {
	tokens int
	last   time.Time
}

type heldOperation struct {
	user, payload string
	created       time.Time
}

func TestRetentionProperty(t *testing.T) {
	if Fail(InvalidStage).Error() != "invalid_stage" || !errors.As(Fail(InvalidStage), new(*Failure)) {
		t.Errorf("a named handler error")
	}
	l := testLimits
	l.Journal, l.Workspaces, l.Users, l.WorkspaceJournal, l.UserJournal = 6, 1, 2, 6, 3
	l.Operations, l.WorkspaceOperations, l.UserOperations = 4, 4, 2
	l.Burst, l.Refill, l.Replay, l.Retention = 3, time.Second, time.Hour, 2*time.Hour
	seeds := int64(20)
	if testing.Short() {
		seeds = 4
	}
	seen := map[Code]int{}
	for seed := int64(1); seed <= seeds; seed++ {
		random := rand.New(rand.NewSource(seed))
		f := setupWith(t, l)
		users := []string{"ann", "bob", "cat"}
		for _, user := range users {
			f.member(user)
			f.grant(user, gate.Issue, "c1")
			f.grant(user, gate.View, "c1", "quantity")
		}
		buckets := map[string]*heldBucket{}
		var records []struct {
			user string
			at   time.Time
		}
		operations := map[string]*heldOperation{}
		sent := map[string]string{}
		for step := 0; step < 150; step++ {
			f.advance([]time.Duration{0, 300 * time.Millisecond, time.Second, 20 * time.Minute, 70 * time.Minute}[random.Intn(5)])
			now := *f.now
			user := users[random.Intn(len(users))]
			answer := f.answer(user)
			kind := random.Intn(3)
			key, payload := []string{"k1", "k2", "k3", "k4"}[random.Intn(4)], []string{"1", "2"}[random.Intn(2)]
			call, current := refused(answer), ""
			switch kind {
			case 1:
				call = reading(answer)
			case 2:
				call = adjust(answer, key, payload)
				f.h.mu.Lock()
				call.Expected = f.h.targets["w1/t1"].Revision
				current = call.Expected
				f.h.mu.Unlock()
				if old := sent[key]; old != "" && random.Intn(2) == 0 {
					payload, call.Expected = old[:1], old[2:]
					call.Fields = []Value{{"amount", payload}}
				}
				sent[key] = payload + "/" + call.Expected
			}
			_, appliedBefore := f.h.count()
			got := f.c.Execute(call)
			_, appliedAfter := f.h.count()
			replayExpected, applyExpected := false, false
			b := buckets[user]
			if b == nil {
				b = &heldBucket{tokens: l.Burst, last: now}
				buckets[user] = b
			}
			for b.tokens < l.Burst && !b.last.Add(l.Refill).After(now) {
				b.tokens++
				b.last = b.last.Add(l.Refill)
			}
			if b.tokens == l.Burst {
				b.last = now
			}
			want := Code("")
			if b.tokens == 0 {
				want = RateLimited
			} else {
				b.tokens--
				live := records[:0]
				for _, r := range records {
					if now.Before(r.at.Add(l.Retention)) {
						live = append(live, r)
					}
				}
				records = live
				mine, holders := 0, map[string]bool{}
				for _, r := range records {
					holders[r.user] = true
					if r.user == user {
						mine++
					}
				}
				room := mine < l.UserJournal && len(records) < l.WorkspaceJournal && (holders[user] || len(holders) < l.Users)
				switch {
				case !room && kind == 0:
					want = UnsupportedCommand
				case !room:
					want = DependencyUnavailable
				case kind == 0:
					want = UnsupportedCommand
				case kind == 1:
					applyExpected = true
				default:
					for k, op := range operations {
						if !now.Before(op.created.Add(l.Retention)) {
							delete(operations, k)
						}
					}
					op := operations[key]
					ownCount, opHolders := 0, map[string]bool{}
					for _, other := range operations {
						opHolders[other.user] = true
						if other.user == user {
							ownCount++
						}
					}
					switch {
					case op != nil && (op.user != user || op.payload != payload+"/"+call.Expected):
						want = IdempotencyConflict
					case op != nil && !now.Before(op.created.Add(l.Replay)):
						want = IdempotencyKeyExpired
					case op != nil:
						replayExpected = true
					case ownCount >= l.UserOperations || len(operations) >= l.WorkspaceOperations || !opHolders[user] && len(opHolders) >= l.Users:
						want = DependencyUnavailable
					case call.Expected != current:
						want = RevisionConflict
					default:
						applyExpected = true
						operations[key] = &heldOperation{user: user, payload: payload + "/" + call.Expected, created: now}
					}
				}
				if room {
					records = append(records, struct {
						user string
						at   time.Time
					}{user, now})
				}
			}
			if want == "" && got.Err != nil || want != "" && (got.Err == nil || got.Err.Code != want) {
				t.Fatalf("seed %d step %d: %s %d %s: %+v; want %q", seed, step, user, kind, key, got.Err, want)
			}
			seen[want]++
			if got.Replay != replayExpected || (appliedAfter > appliedBefore) != applyExpected {
				t.Fatalf("seed %d step %d: replay %v apply %v; want %v %v", seed, step, got.Replay, appliedAfter > appliedBefore, replayExpected, applyExpected)
			}
		}
		if len(f.c.operations) != len(operations) || f.c.reserved != 0 {
			t.Fatalf("seed %d: %d operations; model %d", seed, len(f.c.operations), len(operations))
		}
		f.c.mu.Lock()
		f.c.sweepJournal(*f.now)
		f.c.sweepOperations(*f.now)
		recordUsers, operationUsers := map[holder]int{}, map[holder]int{}
		for _, r := range records {
			if f.now.Before(r.at.Add(l.Retention)) {
				recordUsers[holder{"w1", r.user}]++
			}
		}
		for _, op := range operations {
			if f.now.Before(op.created.Add(l.Retention)) {
				operationUsers[holder{"w1", op.user}]++
			}
		}
		same := func(a, b map[holder]int) bool {
			if len(a) != len(b) {
				return false
			}
			for k, v := range a {
				if b[k] != v {
					return false
				}
			}
			return true
		}
		if !same(recordUsers, f.c.records.users) || !same(operationUsers, f.c.stored.users) || len(f.c.records.members) > l.Users {
			t.Fatalf("seed %d: held records %v model %v; operations %v model %v", seed, f.c.records.users, recordUsers, f.c.stored.users, operationUsers)
		}
		f.c.mu.Unlock()
	}
	for _, code := range []Code{"", RevisionConflict, RateLimited, UnsupportedCommand, DependencyUnavailable, IdempotencyConflict, IdempotencyKeyExpired} {
		if seen[code] == 0 {
			t.Errorf("the model never reached %q", code)
		}
	}
}
