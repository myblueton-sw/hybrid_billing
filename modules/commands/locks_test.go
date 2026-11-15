package commands

import (
	"math/rand"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

type thing struct {
	revision string
}

type keyed struct {
	mu       sync.Mutex
	aliases  map[string]string
	objects  map[string]*thing
	gates    map[string]chan struct{}
	entered  chan string
	active   map[string]int
	overlap  bool
	mismatch bool
	pause    time.Duration
	applies  int
	resolve  func(target string, calls int) (string, bool)
	calls    map[string]int
	failure  error
	exit     bool
}

func newKeyed() *keyed {
	return &keyed{aliases: map[string]string{"a": "K1", "A": "K1", "b": "K2", "n": ""}, objects: map[string]*thing{"K1": {"r1"}, "K2": {"r1"}, "": {"r1"}},
		gates: map[string]chan struct{}{}, entered: make(chan string, 10), active: map[string]int{}, calls: map[string]int{}}
}

func (k *keyed) Resolve(workspace, target string) (Target, Resolution) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.calls[target]++
	key, ok := k.aliases[target]
	if k.resolve != nil {
		key, ok = k.resolve(target, k.calls[target])
	}
	if !ok {
		return Target{Key: key}, NotFound
	}
	return Target{CustomerID: "c1", Revision: k.objects[key].revision, AuthorID: "ann", Key: key}, Found
}

func (k *keyed) Apply(context Context) (Outcome, error) {
	k.mu.Lock()
	key := k.aliases[context.Target]
	if context.Expected != "" {
		k.active[key]++
		total := 0
		for _, count := range k.active {
			total += count
		}
		if k.active[key] > 1 || k.active[""] > 0 && total > 1 {
			k.overlap = true
		}
		if context.Key != key {
			k.mismatch = true
		}
		defer func() {
			k.mu.Lock()
			k.active[key]--
			k.mu.Unlock()
		}()
	}
	k.applies++
	wait := k.gates[context.Target]
	if context.Expected == "" {
		wait = nil
	}
	failure, exit, pause := k.failure, k.exit, k.pause
	k.mu.Unlock()
	time.Sleep(pause)
	if wait != nil {
		k.entered <- context.Target
		<-wait
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	if exit {
		runtime.Goexit()
	}
	if failure != nil {
		return Outcome{}, failure
	}
	if context.Expected == "" {
		return Outcome{}, nil
	}
	k.objects[key].revision = context.Expected + "x"
	return Outcome{ResultRevision: k.objects[key].revision, BusinessState: "recorded"}, nil
}

func (k *keyed) hold(target string) chan struct{} {
	k.mu.Lock()
	defer k.mu.Unlock()
	gate := make(chan struct{})
	k.gates[target] = gate
	return gate
}

func (k *keyed) open(target string, gate chan struct{}) {
	k.mu.Lock()
	delete(k.gates, target)
	k.mu.Unlock()
	close(gate)
}

func (k *keyed) revision(key string) string {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.objects[key].revision
}

func keyedWorld(t *testing.T, l Limits) (world, *keyed, gate.Answer) {
	f := setupWith(t, l)
	f.member("ann")
	f.grant("ann", gate.Deliver, "c1")
	f.grant("ann", gate.View, "c1", "quantity")
	k := newKeyed()
	f.must(f.c.Register(Definition{ID: "x.send", Permission: Permission{Action: gate.Deliver, Kind: "statement"}, Mutating: true, Handler: k}))
	f.must(f.c.Register(Definition{ID: "x.read", Permission: Permission{Action: gate.View, Kind: "statement", Fields: []string{"quantity"}}, Handler: k}))
	return f, k, f.answer("ann")
}

func send(answer gate.Answer, target, key, expected string) Call {
	return Call{Answer: answer, Request: "r", Command: "x.send", Target: target, Key: key, Expected: expected}
}

func later(f world, call Call) chan Result {
	done := make(chan Result, 1)
	go func() { done <- f.c.Execute(call) }()
	return done
}

func pending(t *testing.T, done chan Result, name string) {
	t.Helper()
	select {
	case got := <-done:
		t.Fatalf("%s returned while it should wait: %+v", name, got)
	case <-time.After(50 * time.Millisecond):
	}
}

func (f world) idle() {
	f.t.Helper()
	f.c.mu.Lock()
	defer f.c.mu.Unlock()
	if len(f.c.locks) != 0 || f.c.reserved != 0 {
		f.t.Errorf("locks %v, reserved %d", f.c.locks, f.c.reserved)
	}
	for _, op := range f.c.operations {
		if op.pending {
			f.t.Errorf("a reservation stayed pending")
		}
	}
}

func TestLocksByObject(t *testing.T) {
	f, k, ann := keyedWorld(t, testLimits)
	release := k.hold("a")
	first := later(f, send(ann, "a", "k1", "r1"))
	<-k.entered
	f.code("another object during a slow handler", f.c.Execute(send(ann, "b", "k2", "r1")), "")
	f.code("a missing object during a slow handler", f.c.Execute(send(ann, "zz", "k3", "r1")), PermissionDenied)
	f.code("a reading command during a slow handler", f.c.Execute(Call{Answer: ann, Request: "r", Command: "x.read", Target: "a"}), "")
	alias := later(f, send(ann, "A", "k4", "r1"))
	pending(t, alias, "the same object through another spelling")
	keyless := later(f, send(ann, "n", "k5", "r1"))
	pending(t, keyless, "an object without a key")
	k.open("a", release)
	f.code("the slow call", <-first, "")
	f.code("the alias after the first applied", <-alias, RevisionConflict)
	f.code("the keyless call", <-keyless, "")
	if k.overlap || k.mismatch {
		t.Errorf("two applies to one object overlapped, or a handler got another key: %v %v", k.overlap, k.mismatch)
	}
	release = k.hold("n")
	exclusive := later(f, send(ann, "n", "k6", k.revision("")))
	select {
	case <-k.entered:
	case got := <-exclusive:
		t.Fatalf("the keyless call did not reach its handler: %+v %+v", got, got.Err)
	}
	blocked := later(f, send(ann, "b", "k7", k.revision("K2")))
	pending(t, blocked, "a keyed call during a keyless one")
	k.open("n", release)
	f.code("the keyless call", <-exclusive, "")
	f.code("the keyed call afterwards", <-blocked, "")
	f.idle()
}

func TestMovedObjects(t *testing.T) {
	f, k, ann := keyedWorld(t, testLimits)
	for name, moves := range map[string]func(string, int) (string, bool){
		"another key":            func(_ string, calls int) (string, bool) { return []string{"K1", "K2"}[(calls+1)%2], true },
		"a key that is lost":     func(_ string, calls int) (string, bool) { return []string{"K1", ""}[(calls+1)%2], true },
		"an object that is gone": func(_ string, calls int) (string, bool) { return "K1", calls%2 == 1 },
		"an object that appears": func(_ string, calls int) (string, bool) { return "K1", calls%2 == 0 },
	} {
		k.mu.Lock()
		k.resolve, k.calls = moves, map[string]int{}
		k.mu.Unlock()
		if got := f.c.Execute(send(ann, "a", "m-"+strings.ReplaceAll(name, " ", "-"), "r1")); got.Err == nil || got.Err.Code != DependencyUnavailable {
			t.Errorf("%s: %+v", name, got.Err)
		}
	}
	k.mu.Lock()
	k.resolve = func(string, int) (string, bool) { return "K1", false }
	k.mu.Unlock()
	f.code("a missing object that names a key", f.c.Execute(send(ann, "a", "m-missing", "r1")), PermissionDenied)
	k.mu.Lock()
	k.resolve = func(_ string, calls int) (string, bool) { return []string{"", "K1"}[(calls+1)%2], true }
	k.calls = map[string]int{}
	k.mu.Unlock()
	f.code("a keyless first read under the exclusive lock", f.c.Execute(send(ann, "a", "m-excl", "r1")), "")
	if k.applies != 1 {
		t.Errorf("%d applies", k.applies)
	}
	f.idle()
}

func TestPendingReservations(t *testing.T) {
	l := testLimits
	l.Users, l.UserOperations, l.WorkspaceOperations = 1, 2, 2
	l.Replay, l.Retention = time.Hour, 2*time.Hour
	f, k, ann := keyedWorld(t, l)
	release := k.hold("a")
	first := later(f, send(ann, "a", "k1", "r1"))
	<-k.entered
	f.code("the same key on another object", f.c.Execute(send(ann, "b", "k1", "r1")), IdempotencyConflict)
	f.code("a second reservation", f.c.Execute(send(ann, "b", "k2", "r1")), "")
	f.code("a third while one is pending", f.c.Execute(send(ann, "b", "k3", k.revision("K2"))), DependencyUnavailable)
	k.mu.Lock()
	k.aliases["a"] = "K2"
	k.mu.Unlock()
	f.code("an identical retry after the alias moved", f.c.Execute(send(ann, "a", "k1", "r1")), DependencyUnavailable)
	k.mu.Lock()
	k.aliases["a"] = "K1"
	k.mu.Unlock()
	f.advance(10 * l.Retention)
	f.code("a pending reservation after a forward clock step", f.c.Execute(send(f.answer("ann"), "b", "k1", "r1")), IdempotencyConflict)
	k.open("a", release)
	f.code("the first call", <-first, "")
	f.idle()
}

func TestAbandonedReservations(t *testing.T) {
	f, k, ann := keyedWorld(t, testLimits)
	f.code("a revision conflict", f.c.Execute(send(ann, "a", "k1", "r0")), RevisionConflict)
	k.mu.Lock()
	k.failure = Fail(PrerequisiteFailed)
	k.mu.Unlock()
	f.code("a named handler error", f.c.Execute(send(ann, "a", "k1", "r1")), PrerequisiteFailed)
	k.mu.Lock()
	k.failure = nil
	k.mu.Unlock()
	if len(f.c.operations) != 0 || len(f.c.order) != 0 || f.c.stored.users[holder{"w1", "ann"}] != 0 {
		t.Errorf("an abandoned reservation stayed: %d operations, %d ordered", len(f.c.operations), len(f.c.order))
	}
	f.code("the key after every abandonment", f.c.Execute(send(ann, "a", "k1", "r1")), "")
	k.mu.Lock()
	k.exit = true
	k.mu.Unlock()
	done := make(chan struct{})
	go func() {
		defer close(done)
		f.c.Execute(send(ann, "b", "k9", "r1"))
	}()
	<-done
	k.mu.Lock()
	k.exit = false
	k.mu.Unlock()
	f.idle()
	if record := f.last(); record.Outcome != string(InternalError) || record.Operation == "" {
		t.Errorf("a handler that exited is not recorded as an unknown end: %+v", record)
	}
	again := f.code("the same key after a handler exited", f.c.Execute(send(ann, "b", "k9", "r1")), InternalError)
	if !again.Replay || again.Err.Next != NextOperation || k.applies != 3 {
		t.Errorf("a handler that exited ran again: %+v, %d applies", again, k.applies)
	}
	f.idle()
}

func TestLockTable(t *testing.T) {
	f := setup(t)
	one := f.c.acquire("w1", "K")
	done := make(chan struct{})
	go func() {
		two := f.c.acquire("w2", "K")
		two()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("the same key in another workspace waited")
	}
	whole := make(chan struct{})
	go func() {
		release := f.c.acquire("w1", "")
		release()
		close(whole)
	}()
	select {
	case <-whole:
		t.Fatal("the workspace lock was taken while a key was held")
	case <-time.After(50 * time.Millisecond):
	}
	one()
	<-whole
	f.idle()
}

type racing struct {
	keyed
	successes map[string]int
	random    *rand.Rand
}

func (r *racing) Apply(context Context) (Outcome, error) {
	r.mu.Lock()
	fail := r.random.Intn(4) == 0
	r.pause = time.Duration(r.random.Intn(3)) * time.Millisecond
	r.mu.Unlock()
	if context.Expected == "" {
		return Outcome{}, nil
	}
	if fail {
		r.mu.Lock()
		key := r.aliases[context.Target]
		r.active[key]++
		if r.active[key] > 1 {
			r.overlap = true
		}
		r.active[key]--
		r.mu.Unlock()
		return Outcome{}, Fail(PrerequisiteFailed)
	}
	outcome, err := r.keyed.Apply(context)
	if err == nil {
		r.mu.Lock()
		r.successes[context.Request]++
		r.mu.Unlock()
	}
	return outcome, err
}

func TestConcurrentLockProperty(t *testing.T) {
	seeds := int64(10)
	if testing.Short() {
		seeds = 3
	}
	for seed := int64(1); seed <= seeds; seed++ {
		f := setup(t)
		f.member("ann")
		f.grant("ann", gate.Deliver, "c1")
		h := &racing{keyed: *newKeyed(), successes: map[string]int{}, random: rand.New(rand.NewSource(seed))}
		f.must(f.c.Register(Definition{ID: "x.send", Permission: Permission{Action: gate.Deliver, Kind: "statement"}, Mutating: true, Handler: h}))
		ann := f.answer("ann")
		var wait sync.WaitGroup
		for worker := 0; worker < 8; worker++ {
			wait.Add(1)
			go func(worker int) {
				defer wait.Done()
				local := rand.New(rand.NewSource(seed*100 + int64(worker)))
				for step := 0; step < 25; step++ {
					target := []string{"a", "A", "b", "n"}[local.Intn(4)]
					key := []string{"k1", "k2", "k3", "k4", "k5", "k6"}[local.Intn(6)]
					expected := h.revision(map[string]string{"a": "K1", "A": "K1", "b": "K2", "n": ""}[target])
					call := send(ann, target, key, expected)
					call.Request = key
					f.c.Execute(call)
				}
			}(worker)
		}
		wait.Wait()
		if h.overlap || h.mismatch || len(h.successes) == 0 {
			t.Fatalf("seed %d: overlap %v, mismatch %v, %d successes", seed, h.overlap, h.mismatch, len(h.successes))
		}
		for key, count := range h.successes {
			if count > 1 {
				t.Fatalf("seed %d: key %s applied %d times", seed, key, count)
			}
		}
		f.idle()
		f.c.mu.Lock()
		held := 0
		for _, count := range f.c.stored.users {
			held += count
		}
		if held != len(f.c.operations) || len(f.c.order) != len(f.c.operations) {
			f.c.mu.Unlock()
			t.Fatalf("seed %d: held %d, operations %d, ordered %d", seed, held, len(f.c.operations), len(f.c.order))
		}
		f.c.mu.Unlock()
	}
}
