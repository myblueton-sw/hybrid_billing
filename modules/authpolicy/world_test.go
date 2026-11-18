package authpolicy

import (
	"errors"
	"sync"
	"testing"
	"time"
)

var start = time.Date(2026, 11, 16, 9, 0, 0, 0, time.UTC)

func admin(user string) Principal {
	return Principal{User: user, Workspace: "w1", Role: RoleAdministrator, Kind: KindHuman}
}

func member(user string) Principal {
	return Principal{User: user, Workspace: "w1", Role: RoleMember, Kind: KindHuman}
}

var (
	ann    = admin("ann")
	bob    = admin("bob")
	cat    = admin("cat")
	dan    = admin("dan")
	worker = Principal{User: "worker", Kind: KindSystem}
)

type world struct {
	t   testing.TB
	r   *Registry
	now *time.Time
	mu  *sync.Mutex
}

func setup(t testing.TB) world {
	t.Helper()
	now := start
	mu := &sync.Mutex{}
	r, err := NewRegistry(func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return now
	})
	if err != nil {
		t.Fatal(err)
	}
	return world{t: t, r: r, now: &now, mu: mu}
}

func (f world) advance(d time.Duration) {
	f.mu.Lock()
	*f.now = f.now.Add(d)
	f.mu.Unlock()
}

func (f world) at() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return *f.now
}

func (f world) must(err error) {
	f.t.Helper()
	if err != nil {
		f.t.Fatal(err)
	}
}

func (f world) strong() Authentication {
	return Authentication{Method: MethodOIDC, Issuer: "idp", Factors: []string{FactorWebAuthn}, At: f.at().Add(-time.Minute)}
}

func (f world) weak() Authentication {
	return Authentication{Method: MethodPassword, At: f.at().Add(-time.Minute)}
}

func (f world) tryBy(c Change, proposer, approver Principal, pa, aa Authentication) (ProposalView, error) {
	p, err := f.r.Propose(c, proposer, pa)
	if err != nil {
		return p, err
	}
	return f.r.Approve(p.Number, approver, aa)
}

func (f world) by(c Change, proposer, approver Principal) ProposalView {
	f.t.Helper()
	v, err := f.tryBy(c, proposer, approver, f.strong(), f.strong())
	f.must(err)
	return v
}

func (f world) try(c Change) error {
	_, err := f.tryBy(c, ann, bob, f.strong(), f.strong())
	return err
}

func open() Revision {
	return Revision{Workspace: "w1", Methods: []string{MethodLDAP, MethodOIDC, MethodPassword, MethodSAML}, MFA: MFANone, MaxAge: 24 * time.Hour}
}

func standard(t testing.TB) world {
	f := setup(t)
	f.revise(open())
	return f
}

func (f world) revise(x Revision) RevisionView {
	f.t.Helper()
	epoch, number := uint64(1), uint64(1)
	if w := f.r.workspaces[x.Workspace]; w != nil {
		epoch, number = w.revisions[len(w.revisions)-1].Epoch+1, w.number+1
	}
	v, err := f.r.Publish(Publication{Revision: x, Epoch: epoch, Number: number, Proposer: "ann", Validator: "ann", Approver: "bob", Activator: "cat"})
	f.must(err)
	return v
}

func planOf(users ...string) Change {
	return Change{Kind: ChangePlan, Plan: Plan{Workspace: "w1", Users: users, Chunk: 1, Reason: "r"}}
}

func (f world) block(users []string, chunk int) string {
	f.t.Helper()
	f.by(Change{Kind: ChangePlan, Plan: Plan{Workspace: "w1", Users: users, Chunk: chunk, Reason: "incident"}}, cat, dan)
	return "w1/" + itoa(int(f.r.workspaces["w1"].plans))
}

func expect(t testing.TB, name string, err, want error) {
	t.Helper()
	if !errors.Is(err, want) || want == nil && err != nil {
		t.Errorf("%s: got %v, want %v", name, err, want)
	}
}

func reason(t testing.TB, name string, err error, want string) {
	t.Helper()
	var refusal Refusal
	if !errors.As(err, &refusal) || refusal.Reason != want {
		t.Errorf("%s: got %v, want %s", name, err, want)
	}
}

func second[T any](_ T, err error) error { return err }

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	out := ""
	for ; n > 0; n /= 10 {
		out = string(rune('0'+n%10)) + out
	}
	return out
}
