package accesschange

import (
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"
)

var start = time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC)

type world struct {
	t   testing.TB
	r   *Registry
	now *time.Time
	mu  *sync.Mutex
}

func limits() Limits {
	return Limits{Lifetime: 168 * time.Hour, PerProposer: 3, PerWorkspace: 5, EndedKept: 4, RevocationsKept: 3}
}

func setup(t testing.TB) world {
	t.Helper()
	now, mu := start, &sync.Mutex{}
	r, err := NewRegistry(func() time.Time { mu.Lock(); defer mu.Unlock(); return now }, limits())
	if err != nil {
		t.Fatal(err)
	}
	return world{t, r, &now, mu}
}

func (w world) advance(d time.Duration) {
	w.mu.Lock()
	*w.now = w.now.Add(d)
	w.mu.Unlock()
}

var (
	alice = Actor{User: "alice", Human: true}
	bob   = Actor{User: "bob", Human: true}
	carol = Actor{User: "carol", Human: true}
	robot = Actor{User: "robot"}
)

func add(subject string) Change {
	return Change{Kind: ChangeAdd, Subject: subject, Action: "view", ResourceKind: "focus_data", Fields: []string{"rows"}, Customer: "all", Days: 10}
}

func revoke(grant string) Change { return Change{Kind: ChangeRevoke, Grant: grant} }

func proposal(c Change, by Actor) Proposal {
	p := Proposal{Workspace: "w1", Change: c, Basis: "basis-1", CeilingUntil: start.Add(60 * 24 * time.Hour), Proposer: by, Allowed: true, Rationale: "reason"}
	if c.Kind == ChangeRevoke {
		p.Holder = "dave"
	}
	return p
}

func (w world) propose(p Proposal) View {
	w.t.Helper()
	v, existing, err := w.r.Propose(p)
	if err != nil || existing {
		w.t.Fatalf("propose %v %v", err, existing)
	}
	return v
}

func (w world) propErr(p Proposal) error {
	w.t.Helper()
	v, existing, err := w.r.Propose(p)
	if err != nil && (existing || v.Number != 0) {
		w.t.Errorf("a refused proposal reported a request %v %+v", existing, v)
	}
	return err
}

func refused(t testing.TB, name string, err error, want Reason) {
	t.Helper()
	var r Refusal
	if !errors.As(err, &r) || r.Reason != want {
		t.Errorf("%s: %v want %s", name, err, want)
	}
}

func applied(string) func(Applier) (string, error) {
	return func(Applier) (string, error) { return "grant-x", nil }
}

func TestRequestLifecycle(t *testing.T) {
	w := setup(t)
	v := w.propose(proposal(add("carol"), alice))
	if v.ID != "access-request-1" || v.State != StatePending || v.Proposer != "alice" || !v.ExpiresAt.Equal(start.Add(168*time.Hour)) || len(v.Digest) != 64 || v.Rationale != "reason" {
		t.Fatalf("view %+v", v)
	}
	again, existing, err := w.r.Propose(proposal(add("carol"), alice))
	if err != nil || !existing || again.Number != v.Number {
		t.Errorf("equal proposal %v %v", err, existing)
	}
	err = w.propErr(proposal(add("carol"), bob))
	refused(t, "another proposer for the same target", err, ReasonPendingTarget)
	other := add("carol")
	other.Days = 5
	err = w.propErr(proposal(other, alice))
	refused(t, "a different change for the same target", err, ReasonPendingTarget)
	calls := 0
	got, err := w.r.Approve(v.Number, "w1", bob, v.Digest, Standing{Basis: "basis-1", Allowed: true}, func(Applier) (string, error) { calls++; return "grant-x", nil })
	if err != nil || got.State != StateApplied || got.DecidedBy != "bob" || got.Result != "grant-x" || calls != 1 {
		t.Fatalf("approve %+v %v", got, err)
	}
	_, err = w.r.Approve(v.Number, "w1", bob, v.Digest, Standing{Basis: "basis-1", Allowed: true}, func(Applier) (string, error) { calls++; return "", nil })
	refused(t, "a second approval", err, ReasonNotPending)
	if calls != 1 {
		t.Error("applied twice")
	}
	r := w.propose(proposal(revoke("grant-x"), alice))
	if r.Holder != "dave" || r.Change.Kind != ChangeRevoke {
		t.Errorf("revoke view %+v", r)
	}
	rejected, err := w.r.Reject(r.Number, "w1", bob, true, "not now")
	if err != nil || rejected.State != StateRejected || rejected.DecisionReason != "not now" || rejected.DecidedBy != "bob" {
		t.Errorf("reject %+v %v", rejected, err)
	}
	x := w.propose(proposal(add("dave"), alice))
	if got, err := w.r.Withdraw(x.Number, "w1", alice); err != nil || got.State != StateWithdrawn {
		t.Errorf("withdraw %+v %v", got, err)
	}
	listed := w.r.Requests("w1")
	if len(listed) != 3 || listed[0].Number != 1 || len(w.r.Requests("w2")) != 0 {
		t.Errorf("requests %+v", listed)
	}
	listed[0].Change.Fields[0] = "changed"
	if w.r.Requests("w1")[0].Change.Fields[0] != "rows" {
		t.Error("views alias the registry")
	}
	if _, ok := w.r.Request(1, "w2"); ok {
		t.Error("another workspace")
	}
	if got, ok := w.r.Request(1, "w1"); !ok || got.State != StateApplied {
		t.Error("request")
	}
}

func TestHolderMayApproveButNeverRejectARevocation(t *testing.T) {
	w := setup(t)
	v := w.propose(proposal(revoke("grant-d"), alice))
	_, err := w.r.Reject(v.Number, "w1", Actor{User: "dave", Human: true}, true, "keep it")
	refused(t, "holder reject", err, ReasonHolderReject)
	got, err := w.r.Approve(v.Number, "w1", Actor{User: "dave", Human: true}, v.Digest, Standing{Basis: "basis-1", Allowed: true}, applied(""))
	if err != nil || got.State != StateApplied {
		t.Errorf("holder approve %v", err)
	}
}

func TestProposalRefusals(t *testing.T) {
	w := setup(t)
	for name, c := range map[string]struct {
		p    Proposal
		want Reason
	}{
		"not human":           {proposal(add("carol"), robot), ReasonNotHuman},
		"proposer is subject": {proposal(add("alice"), alice), ReasonProposerSubject},
		"proposer is holder":  {proposal(revoke("g"), Actor{User: "dave", Human: true}), ReasonProposerHolder},
		"not allowed":         {func() Proposal { p := proposal(add("carol"), alice); p.Allowed = false; return p }(), ReasonNotAllowed},
		"expiry passed": {func() Proposal {
			p := proposal(add("carol"), alice)
			p.CeilingUntil = start.Add(10 * 24 * time.Hour)
			return p
		}(), ReasonExpiry},
		"human before subject": {proposal(add("robot"), robot), ReasonNotHuman},
		"subject before allowed": {func() Proposal {
			p := proposal(add("alice"), alice)
			p.Allowed = false
			return p
		}(), ReasonProposerSubject},
	} {
		err := w.propErr(c.p)
		refused(t, name, err, c.want)
	}
	for name, p := range map[string]Proposal{
		"kind":        proposal(Change{Kind: "other"}, alice),
		"add days":    proposal(Change{Kind: ChangeAdd, Subject: "carol", Action: "view", ResourceKind: "k", Customer: "all"}, alice),
		"add grant":   proposal(Change{Kind: ChangeAdd, Subject: "carol", Action: "view", ResourceKind: "k", Customer: "all", Days: 1, Grant: "g"}, alice),
		"revoke days": proposal(Change{Kind: ChangeRevoke, Grant: "g", Days: 1}, alice),
		"workspace":   func() Proposal { p := proposal(add("carol"), alice); p.Workspace = ""; return p }(),
		"holder":      func() Proposal { p := proposal(revoke("g"), alice); p.Holder = ""; return p }(),
		"proposer":    proposal(add("carol"), Actor{Human: true}),
	} {
		if err := w.propErr(p); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if len(w.r.Requests("w1")) != 0 {
		t.Error("a refused proposal was kept")
	}
	exact := proposal(add("carol"), alice)
	exact.CeilingUntil = start.Add(10*24*time.Hour + time.Nanosecond)
	if v := w.propose(exact); !v.ExpiresAt.Equal(start.Add(time.Nanosecond)) {
		t.Errorf("expiry bound by the ceiling %v", v.ExpiresAt)
	}
}

func TestLimits(t *testing.T) {
	w := setup(t)
	for _, s := range []string{"s1", "s2", "s3"} {
		w.propose(proposal(add(s), alice))
	}
	err := w.propErr(proposal(add("s4"), alice))
	refused(t, "per proposer", err, ReasonLimit)
	w.propose(proposal(add("s4"), bob))
	w.propose(proposal(add("s5"), bob))
	err = w.propErr(proposal(add("s6"), carol))
	refused(t, "per workspace", err, ReasonLimit)
	other := proposal(add("s6"), carol)
	other.Workspace = "w2"
	w.propose(other)
	w.advance(168 * time.Hour)
	w.propose(proposal(add("s7"), carol))
	for name, l := range map[string]Limits{
		"lifetime":     {PerProposer: 1, PerWorkspace: 1, EndedKept: 1, RevocationsKept: 1},
		"proposer":     {Lifetime: time.Hour, PerWorkspace: 1, EndedKept: 1, RevocationsKept: 1},
		"workspace":    {Lifetime: time.Hour, PerProposer: 2, PerWorkspace: 1, EndedKept: 1, RevocationsKept: 1},
		"ended":        {Lifetime: time.Hour, PerProposer: 1, PerWorkspace: 1, RevocationsKept: 1},
		"revocations":  {Lifetime: time.Hour, PerProposer: 1, PerWorkspace: 1, EndedKept: 1},
		"missing time": {},
	} {
		if _, err := NewRegistry(func() time.Time { return start }, l); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := NewRegistry(func() time.Time { return start }, Limits{Lifetime: time.Nanosecond, PerProposer: 1, PerWorkspace: 1, EndedKept: 1, RevocationsKept: 1}); err != nil {
		t.Errorf("the smallest valid limits %v", err)
	}
	one := add("dave")
	one.Days = 1
	if v := w.propose(proposal(one, alice)); v.Change.Days != 1 {
		t.Error("one day")
	}
	if _, err := NewRegistry(nil, limits()); !errors.Is(err, ErrInvalid) {
		t.Error("clock")
	}
}

func TestLazyExpiryAtTheBoundary(t *testing.T) {
	w := setup(t)
	v := w.propose(proposal(add("carol"), alice))
	w.advance(168*time.Hour - time.Nanosecond)
	if got, _ := w.r.Request(v.Number, "w1"); got.State != StatePending {
		t.Errorf("1 ns before %s", got.State)
	}
	w.advance(time.Nanosecond)
	got, _ := w.r.Request(v.Number, "w1")
	if got.State != StateExpired || w.r.requests[v.Number].view.State != StatePending {
		t.Errorf("a read shows expired without recording it %s", got.State)
	}
	_, err := w.r.Withdraw(v.Number, "w1", alice)
	refused(t, "action on an expired request", err, ReasonExpired)
	if stored := w.r.requests[v.Number].view; stored.State != StateExpired || !stored.DecidedAt.Equal(v.ExpiresAt) {
		t.Errorf("expired recorded %+v", stored)
	}
	_, err = w.r.Withdraw(v.Number, "w1", alice)
	refused(t, "after expiry", err, ReasonNotPending)
	w.propose(proposal(add("carol"), alice))
}

func TestBasisAndDigest(t *testing.T) {
	w := setup(t)
	v := w.propose(proposal(add("carol"), alice))
	_, err := w.r.Approve(v.Number, "w1", bob, "wrong", Standing{Basis: "basis-1", Allowed: true}, applied(""))
	refused(t, "wrong digest", err, ReasonDigest)
	if got, _ := w.r.Request(v.Number, "w1"); got.State != StatePending {
		t.Error("a wrong digest changed state")
	}
	_, err = w.r.Approve(v.Number, "w1", bob, v.Digest, Standing{Basis: "basis-2", Allowed: true}, applied(""))
	refused(t, "basis changed", err, ReasonSuperseded)
	if got, _ := w.r.Request(v.Number, "w1"); got.State != StateSuperseded {
		t.Errorf("superseded %s", got.State)
	}
	second := w.propose(proposal(add("carol"), alice))
	if second.Digest != v.Digest {
		t.Error("the same change and basis give the same revision")
	}
	changed := proposal(add("carol"), bob)
	changed.Basis = "basis-3"
	if _, err := w.r.Withdraw(second.Number, "w1", alice); err != nil {
		t.Fatal(err)
	}
	if third := w.propose(changed); third.Digest == v.Digest {
		t.Error("a different basis gives a different revision")
	}
}

func TestApplyFailureLeavesThePendingRequest(t *testing.T) {
	w := setup(t)
	v := w.propose(proposal(add("carol"), alice))
	boom := errors.New("deny failed")
	if _, err := w.r.Approve(v.Number, "w1", bob, v.Digest, Standing{Basis: "basis-1", Allowed: true}, func(Applier) (string, error) { return "", boom }); err != boom {
		t.Errorf("apply error %v", err)
	}
	if got, _ := w.r.Request(v.Number, "w1"); got.State != StatePending || got.Result != "" {
		t.Errorf("after a failed apply %+v", got)
	}
}

func TestDecisionRefusalOrder(t *testing.T) {
	w := setup(t)
	ok := Standing{Basis: "basis-1", Allowed: true}
	v := w.propose(proposal(add("carol"), alice))
	cases := []struct {
		name  string
		actor Actor
		dig   string
		st    Standing
		want  Reason
	}{
		{"human before separation", Actor{User: "alice"}, v.Digest, ok, ReasonNotHuman},
		{"separation before allowed", alice, v.Digest, Standing{Basis: "basis-1"}, ReasonSelfDecision},
		{"subject before allowed", carol, v.Digest, Standing{Basis: "basis-1"}, ReasonApproverSubject},
		{"allowed before digest", bob, "wrong", Standing{Basis: "basis-1"}, ReasonNotAllowed},
		{"digest before basis", bob, "wrong", Standing{Basis: "basis-2", Allowed: true}, ReasonDigest},
	}
	before := w.r.Requests("w1")
	for _, c := range cases {
		_, err := w.r.Approve(v.Number, "w1", c.actor, c.dig, c.st, applied(""))
		refused(t, c.name, err, c.want)
	}
	if !reflect.DeepEqual(before, w.r.Requests("w1")) {
		t.Error("a refusal changed state")
	}
	_, err := w.r.Approve(99, "w1", bob, v.Digest, ok, applied(""))
	refused(t, "unknown", err, ReasonNotFound)
	_, err = w.r.Approve(v.Number, "w2", bob, v.Digest, ok, applied(""))
	refused(t, "other workspace", err, ReasonNotFound)
	w.advance(200 * time.Hour)
	_, err = w.r.Approve(v.Number, "w1", bob, "wrong", ok, applied(""))
	refused(t, "state before digest", err, ReasonExpired)
	r := w.propose(proposal(revoke("g"), alice))
	for _, c := range []struct {
		name    string
		actor   Actor
		allowed bool
		want    Reason
	}{
		{"reject human first", Actor{User: "bob"}, true, ReasonNotHuman},
		{"reject self", alice, false, ReasonSelfDecision},
		{"reject holder before allowed", Actor{User: "dave", Human: true}, false, ReasonHolderReject},
		{"reject allowed", bob, false, ReasonNotAllowed},
	} {
		_, err := w.r.Reject(r.Number, "w1", c.actor, c.allowed, "x")
		refused(t, c.name, err, c.want)
	}
	_, err = w.r.Reject(99, "w1", bob, true, "x")
	refused(t, "reject unknown", err, ReasonNotFound)
	_, err = w.r.Withdraw(r.Number, "w1", Actor{User: "alice"})
	refused(t, "withdraw human", err, ReasonNotHuman)
	_, err = w.r.Withdraw(r.Number, "w1", bob)
	refused(t, "withdraw by another", err, ReasonNotProposer)
	_, err = w.r.Withdraw(99, "w1", alice)
	refused(t, "withdraw unknown", err, ReasonNotFound)
	w.r.Withdraw(r.Number, "w1", alice)
	_, err = w.r.Reject(r.Number, "w1", bob, true, "x")
	refused(t, "reject ended", err, ReasonNotPending)
	if (Refusal{ReasonLimit}).Error() != "access change refused: pending_limit" {
		t.Error("error text")
	}
}

func TestEndedRequestsArePruned(t *testing.T) {
	w := setup(t)
	for i := 0; i < 6; i++ {
		v := w.propose(proposal(add("carol"), alice))
		w.r.Withdraw(v.Number, "w1", alice)
	}
	pending := w.propose(proposal(add("dave"), alice))
	got := w.r.Requests("w1")
	if len(got) != 5 || got[0].Number != 3 || got[4].Number != pending.Number {
		t.Errorf("kept %d first %d", len(got), got[0].Number)
	}
}

func TestNumber(t *testing.T) {
	for id, want := range map[string]uint64{"access-request-1": 1, "access-request-42": 42} {
		if n, ok := Number(id, RequestPrefix); !ok || n != want {
			t.Errorf("%s", id)
		}
	}
	for _, id := range []string{"access-request-0", "access-request-01", "access-request-", "access-request-+1", "revocation-1", "access-request-18446744073709551616", "12"} {
		if _, ok := Number(id, RequestPrefix); ok {
			t.Errorf("%s accepted", id)
		}
	}
}

func TestConcurrentDecisionsOnOneRequest(t *testing.T) {
	for round := 0; round < 20; round++ {
		w := setup(t)
		v := w.propose(proposal(add("carol"), alice))
		var wg sync.WaitGroup
		var mu sync.Mutex
		wins, applies := 0, 0
		decide := []func() error{
			func() error {
				_, err := w.r.Approve(v.Number, "w1", bob, v.Digest, Standing{Basis: "basis-1", Allowed: true}, func(Applier) (string, error) {
					applies++
					return "g", nil
				})
				return err
			},
			func() error {
				_, err := w.r.Reject(v.Number, "w1", Actor{User: "erin", Human: true}, true, "no")
				return err
			},
			func() error { _, err := w.r.Withdraw(v.Number, "w1", alice); return err },
		}
		for _, d := range decide {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if d() == nil {
					mu.Lock()
					wins++
					mu.Unlock()
				}
			}()
		}
		wg.Wait()
		if wins != 1 || applies > 1 {
			t.Fatalf("wins %d applies %d", wins, applies)
		}
	}
}

func TestConcurrentProposalsForOneTarget(t *testing.T) {
	w := setup(t)
	var wg sync.WaitGroup
	for _, who := range []Actor{alice, bob, carol, {User: "erin", Human: true}} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.r.Propose(proposal(add("dave"), who))
			w.r.Requests("w1")
		}()
	}
	wg.Wait()
	if got := w.r.Requests("w1"); len(got) != 1 || got[0].State != StatePending {
		t.Errorf("pending %+v", got)
	}
}
