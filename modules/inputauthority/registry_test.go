package inputauthority

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"
)

type world struct {
	now                   time.Time
	registry              *Registry
	allowed               map[string]string
	target                string
	targetErr             error
	lookupErr             error
	applyErr              error
	afterApplyLookupError bool
	create                bool
	effects               map[string]Effect
	applied               int
}

func (w *world) Check(workspace, actor string) (string, error) {
	if workspace != "w1" || w.allowed[actor] == "" {
		return "", ErrDenied
	}
	return w.allowed[actor], nil
}
func (w *world) Current(s Spec) (string, error) {
	if w.targetErr != nil {
		return "", w.targetErr
	}
	return w.target, nil
}
func (w *world) Lookup(id string) (Effect, error) {
	if w.lookupErr != nil {
		return Effect{}, w.lookupErr
	}
	return w.effects[id], nil
}
func (w *world) Apply(id string, s Spec) error {
	w.applied++
	if w.afterApplyLookupError {
		w.lookupErr = ErrUnknown
	}
	if w.create {
		w.effects[id] = Effect{Found: true, Spec: s, Status: ActiveGrant}
	}
	return w.applyErr
}
func setup(t testing.TB) *world {
	t.Helper()
	w := &world{now: time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC), allowed: map[string]string{"alice": "alice-v1", "bob": "bob-v1", "carol": "carol-v1"}, target: "target-v1", create: true, effects: map[string]Effect{}}
	var err error
	w.registry, err = NewRegistry(Limits{Lifetime: time.Hour, MaxGrantValidity: 24 * time.Hour, Capacity: 20}, w, w, w, func() time.Time { return w.now })
	if err != nil {
		t.Fatal(err)
	}
	return w
}
func (w *world) proposal(kind string) Proposal {
	ceiling := ""
	recipient := "console"
	if kind == Human {
		recipient = "carol"
		ceiling = "approved-console-grant"
	}
	return Proposal{ID: "p1", Spec: Spec{Capability: Confirm, Workspace: "w1", Kind: kind, Recipient: recipient, CeilingID: ceiling, NotBefore: w.now, NotAfter: w.now.Add(12 * time.Hour)}, Proposer: "alice", Rationale: "Synthetic reviewed authority"}
}
func propose(t testing.TB, w *world, kind string) Record {
	t.Helper()
	r, err := w.registry.Propose(w.proposal(kind))
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func intent(r Record) Intent {
	return Intent{Workspace: r.Proposal.Spec.Workspace, ID: r.Proposal.ID, Actor: "bob", Digest: r.Digest, Rationale: "Reviewed exact synthetic grant"}
}
func stored(t testing.TB, w *world) Record {
	t.Helper()
	r, err := w.registry.Record("w1", "p1")
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestSeparateConsoleAndHumanProposals(t *testing.T) {
	for _, kind := range []string{Console, Human} {
		t.Run(kind, func(t *testing.T) {
			w := setup(t)
			r := propose(t, w, kind)
			if r.State != Pending || w.applied != 0 {
				t.Fatal("proposal applied authority")
			}
			got, err := w.registry.Approve(intent(r))
			if err != nil || got.State != Applied || got.Decision.GrantStatus != ActiveGrant || w.applied != 1 {
				t.Fatalf("approval: %s %v", got.State, err)
			}
			if len(w.effects) != 1 || w.effects[got.Decision.GrantID].Spec != r.Proposal.Spec {
				t.Fatal("paired or different grant")
			}
			replay, err := w.registry.Approve(intent(r))
			if err != nil || !reflect.DeepEqual(replay, got) || w.applied != 1 {
				t.Fatal("replay changed decision")
			}
			got.Decision.Actor = "mutated"
			if stored(t, w).Decision.Actor != "bob" {
				t.Fatal("decision alias")
			}
		})
	}
}

func TestGovernorsAndRecipientAreDistinct(t *testing.T) {
	for _, kind := range []string{Console, Human} {
		t.Run(kind, func(t *testing.T) {
			w := setup(t)
			p := w.proposal(kind)
			p.Proposer = "outsider"
			if _, err := w.registry.Propose(p); !errors.Is(err, ErrDenied) {
				t.Fatal(err)
			}
			if kind == Human {
				p = w.proposal(kind)
				p.Proposer = "carol"
				if _, err := w.registry.Propose(p); !errors.Is(err, ErrDenied) {
					t.Fatal("recipient proposed", err)
				}
			}
			r := propose(t, w, kind)
			for _, actor := range []string{"alice", "outsider", "carol"} {
				if actor == "carol" && kind == Console {
					continue
				}
				i := intent(r)
				i.Actor = actor
				if _, err := w.registry.Approve(i); !errors.Is(err, ErrDenied) {
					t.Fatalf("actor %s: %v", actor, err)
				}
			}
			if w.applied != 0 {
				t.Fatal("denied actor applied")
			}
		})
	}
}

func TestProposalBoundariesAndStaleFacts(t *testing.T) {
	for _, change := range []string{"proposer revoked", "approver revoked", "proposer facts", "target facts", "target unavailable", "expired", "digest", "workspace"} {
		t.Run(change, func(t *testing.T) {
			w := setup(t)
			r := propose(t, w, Human)
			i := intent(r)
			switch change {
			case "proposer revoked":
				delete(w.allowed, "alice")
			case "approver revoked":
				delete(w.allowed, "bob")
			case "proposer facts":
				w.allowed["alice"] = "alice-v2"
			case "target facts":
				w.target = "target-v2"
			case "target unavailable":
				w.targetErr = ErrDenied
			case "expired":
				w.now = r.ExpiresAt
			case "digest":
				i.Digest = fmt.Sprintf("%064d", 1)
			case "workspace":
				i.Workspace = "w2"
			}
			if _, err := w.registry.Approve(i); err == nil || w.applied != 0 {
				t.Fatal("stale or foreign proposal applied")
			}
		})
	}
	w := setup(t)
	p := w.proposal(Human)
	r, err := w.registry.Propose(p)
	if err != nil {
		t.Fatal(err)
	}
	again, err := w.registry.Propose(p)
	if err != nil || again.Digest != r.Digest {
		t.Fatal("identical proposal changed")
	}
	p.Rationale = "different"
	if _, err := w.registry.Propose(p); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
}

func TestUnknownEffectReconcilesWithoutRecreating(t *testing.T) {
	for _, status := range []string{ActiveGrant, RevokedGrant, ExpiredGrant} {
		t.Run(status, func(t *testing.T) {
			w := setup(t)
			r := propose(t, w, Human)
			w.applyErr = errors.New("effect created but reply lost")
			if _, err := w.registry.Approve(intent(r)); !errors.Is(err, ErrUnknown) {
				t.Fatal(err)
			}
			q := stored(t, w)
			if q.State != Unknown || w.applied != 1 {
				t.Fatal("uncertainty not retained")
			}
			effect := w.effects[q.Decision.GrantID]
			effect.Status = status
			w.effects[q.Decision.GrantID] = effect
			w.target = "changed-after-effect"
			w.now = w.now.Add(48 * time.Hour)
			got, err := w.registry.Approve(intent(r))
			if err != nil || got.State != Applied || got.Decision.GrantStatus != status || w.applied != 1 {
				t.Fatalf("historical resolve %s: %s %v", status, got.State, err)
			}
		})
	}
}

func TestUnknownAttemptRequiresSameAuthorizedIntent(t *testing.T) {
	for _, change := range []string{"actor", "rationale", "authority", "proposer", "lookup", "effect mismatch", "effect status"} {
		t.Run(change, func(t *testing.T) {
			w := setup(t)
			r := propose(t, w, Console)
			w.applyErr = ErrUnknown
			if _, err := w.registry.Approve(intent(r)); !errors.Is(err, ErrUnknown) {
				t.Fatal(err)
			}
			i := intent(r)
			switch change {
			case "actor":
				i.Actor = "carol"
			case "rationale":
				i.Rationale = "another"
			case "authority":
				delete(w.allowed, "bob")
			case "proposer":
				delete(w.allowed, "alice")
			case "lookup":
				w.lookupErr = ErrDenied
			case "effect mismatch":
				q := stored(t, w)
				e := w.effects[q.Decision.GrantID]
				e.Spec.Recipient = "different"
				w.effects[q.Decision.GrantID] = e
			case "effect status":
				q := stored(t, w)
				e := w.effects[q.Decision.GrantID]
				e.Status = "unknown"
				w.effects[q.Decision.GrantID] = e
			}
			if _, err := w.registry.Approve(i); err == nil {
				t.Fatal("uncertain effect accepted")
			}
			if stored(t, w).State != Unknown || w.applied != 1 {
				t.Fatal("unknown state/effect changed")
			}
		})
	}
}

func TestUnknownAbsentEffectRetryAndForbiddenExits(t *testing.T) {
	w := setup(t)
	r := propose(t, w, Human)
	w.create = false
	w.applyErr = errors.New("failed before effect")
	if _, err := w.registry.Approve(intent(r)); !errors.Is(err, ErrUnknown) {
		t.Fatal(err)
	}
	if _, err := w.registry.Reject(intent(r)); !errors.Is(err, ErrConflict) {
		t.Fatal("unknown rejected", err)
	}
	withdraw := intent(r)
	withdraw.Actor = "alice"
	if _, err := w.registry.Withdraw(withdraw); !errors.Is(err, ErrConflict) {
		t.Fatal("unknown withdrawn", err)
	}
	w.target = "changed"
	if _, err := w.registry.Approve(intent(r)); !errors.Is(err, ErrConflict) || w.applied != 1 || stored(t, w).State != Unknown {
		t.Fatal("changed prerequisite retried", err)
	}
	w.target = "target-v1"
	w.create = true
	w.applyErr = nil
	if _, err := w.registry.Approve(intent(r)); err != nil || w.applied != 2 {
		t.Fatal("unchanged retry failed", err)
	}
}

func TestPreexistingGrantIsNeverAdopted(t *testing.T) {
	w := setup(t)
	r := propose(t, w, Human)
	w.effects["input-grant-"+r.Digest] = Effect{Found: true, Spec: r.Proposal.Spec, Status: ActiveGrant}
	if _, err := w.registry.Approve(intent(r)); !errors.Is(err, ErrConflict) || w.applied != 0 || stored(t, w).State != Pending {
		t.Fatal("adopted unrelated exact grant", err)
	}
}

func TestNilApplyWithoutExactEffectRemainsUnknown(t *testing.T) {
	w := setup(t)
	r := propose(t, w, Console)
	w.create = false
	if _, err := w.registry.Approve(intent(r)); !errors.Is(err, ErrUnknown) || stored(t, w).State != Unknown {
		t.Fatal("nil without effect accepted", err)
	}
}

func TestPendingRejectionWithdrawalAndExpiry(t *testing.T) {
	for _, state := range []string{Rejected, Withdrawn, Expired} {
		t.Run(state, func(t *testing.T) {
			w := setup(t)
			r := propose(t, w, Human)
			i := intent(r)
			var err error
			switch state {
			case Rejected:
				_, err = w.registry.Reject(i)
			case Withdrawn:
				i.Actor = "alice"
				_, err = w.registry.Withdraw(i)
			case Expired:
				w.now = r.ExpiresAt
			}
			if err != nil {
				t.Fatal(err)
			}
			if stored(t, w).State != state || w.applied != 0 {
				t.Fatal("bad terminal state")
			}
			if _, err := w.registry.Approve(intent(r)); err == nil {
				t.Fatal("terminal proposal applied")
			}
		})
	}
}

func TestConcurrentIdenticalApprovalAppliesOnce(t *testing.T) {
	w := setup(t)
	r := propose(t, w, Human)
	var wg sync.WaitGroup
	errs := make(chan error, 12)
	for n := 0; n < 12; n++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := w.registry.Approve(intent(r)); errs <- err }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if w.applied != 1 {
		t.Fatal("duplicate effect")
	}
}
