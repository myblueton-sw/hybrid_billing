package inputauthority

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestProposalBoundsAndRequiredPorts(t *testing.T) {
	for _, change := range []string{"id", "actor", "rationale", "control", "workspace", "kind", "recipient", "ceiling", "zero time", "inverted", "maximum", "unsupported year"} {
		t.Run(change, func(t *testing.T) {
			w := setup(t)
			p := w.proposal(Human)
			switch change {
			case "id":
				p.ID = strings.Repeat("x", 129)
			case "actor":
				p.Proposer = ""
			case "rationale":
				p.Rationale = strings.Repeat("x", 1025)
			case "control":
				p.Rationale = "contains\nnewline"
			case "workspace":
				p.Spec.Workspace = ""
			case "kind":
				p.Spec.Kind = "automatic"
			case "recipient":
				p.Spec.Recipient = ""
			case "ceiling":
				p.Spec.CeilingID = ""
			case "zero time":
				p.Spec.NotBefore = time.Time{}
			case "inverted":
				p.Spec.NotAfter = p.Spec.NotBefore
			case "maximum":
				p.Spec.NotAfter = p.Spec.NotBefore.Add(24*time.Hour + time.Nanosecond)
			case "unsupported year":
				p.Spec.NotAfter = time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)
			}
			if _, err := w.registry.Propose(p); !errors.Is(err, ErrInvalid) {
				t.Fatalf("%s: %v", change, err)
			}
		})
	}
	w := setup(t)
	var nilPort *world
	if _, err := NewRegistry(w.registry.limits, nilPort, w, w, func() time.Time { return w.now }); !errors.Is(err, ErrInvalid) {
		t.Fatal("typed nil authority accepted")
	}
	if _, err := NewRegistry(w.registry.limits, w, nil, w, func() time.Time { return w.now }); !errors.Is(err, ErrInvalid) {
		t.Fatal("nil targets accepted")
	}
	if _, err := NewRegistry(w.registry.limits, w, w, nil, func() time.Time { return w.now }); !errors.Is(err, ErrInvalid) {
		t.Fatal("nil effects accepted")
	}
	if _, err := NewRegistry(w.registry.limits, w, w, w, nil); !errors.Is(err, ErrInvalid) {
		t.Fatal("nil clock accepted")
	}
	for _, limits := range []Limits{{}, {Lifetime: time.Hour, MaxGrantValidity: time.Hour, Capacity: 10001}, {Lifetime: -1, MaxGrantValidity: time.Hour, Capacity: 1}} {
		if _, err := NewRegistry(limits, w, w, w, func() time.Time { return w.now }); !errors.Is(err, ErrInvalid) {
			t.Fatal("invalid limits accepted")
		}
	}
}

func TestBoundedFactsCapacityAndUnknownRetention(t *testing.T) {
	for _, source := range []string{"authority", "target"} {
		t.Run(source, func(t *testing.T) {
			w := setup(t)
			if source == "authority" {
				w.allowed["alice"] = strings.Repeat("x", 4097)
			} else {
				w.target = ""
			}
			if _, err := w.registry.Propose(w.proposal(Human)); !errors.Is(err, ErrInvalid) {
				t.Fatal("invalid facts retained", err)
			}
		})
	}
	w := setup(t)
	w.registry.limits.Capacity = 1
	r := propose(t, w, Human)
	w.create = false
	w.applyErr = ErrUnknown
	if _, err := w.registry.Approve(intent(r)); !errors.Is(err, ErrUnknown) {
		t.Fatal(err)
	}
	w.now = w.now.Add(48 * time.Hour)
	if stored(t, w).State != Unknown {
		t.Fatal("unknown expired")
	}
	next := w.proposal(Console)
	next.ID = "p2"
	if _, err := w.registry.Propose(next); !errors.Is(err, ErrLimit) {
		t.Fatal("unknown did not count against capacity", err)
	}
}

func TestCanonicalTimeAndClockRollback(t *testing.T) {
	w := setup(t)
	p := w.proposal(Console)
	p.Spec.NotBefore = p.Spec.NotBefore.In(time.FixedZone("synthetic", 9*60*60))
	p.Spec.NotAfter = p.Spec.NotAfter.In(time.FixedZone("synthetic", 9*60*60))
	r, err := w.registry.Propose(p)
	if err != nil {
		t.Fatal(err)
	}
	if r.Proposal.Spec.NotBefore.Location() != time.UTC {
		t.Fatal("uncanonical retained time")
	}
	w.now = r.ExpiresAt
	if stored(t, w).State != Expired {
		t.Fatal("expiry absent")
	}
	w.now = r.ProposedAt
	if stored(t, w).State != Expired {
		t.Fatal("rollback restored pending")
	}
	if _, err := w.registry.Approve(intent(r)); err == nil {
		t.Fatal("rollback restored approval")
	}
	w.now = time.Time{}
	if _, err := w.registry.Record("w1", "p1"); !errors.Is(err, ErrInvalid) {
		t.Fatal("invalid clock accepted")
	}
}

func TestLookupFailureBeforeAndAfterEffect(t *testing.T) {
	w := setup(t)
	r := propose(t, w, Console)
	w.lookupErr = ErrUnknown
	if _, err := w.registry.Approve(intent(r)); err == nil || w.applied != 0 || stored(t, w).State != Pending {
		t.Fatal("lookup outage treated as absence")
	}
	w.lookupErr = nil
	w.afterApplyLookupError = true
	if _, err := w.registry.Approve(intent(r)); !errors.Is(err, ErrUnknown) || stored(t, w).State != Unknown || w.applied != 1 {
		t.Fatal("post-apply lookup failure lost")
	}
	w.lookupErr = nil
	w.afterApplyLookupError = false
	if _, err := w.registry.Approve(intent(r)); err != nil || w.applied != 1 {
		t.Fatal("lookup reconciliation reapplied", err)
	}
}

func TestUnknownAbsentEffectNeverRestartsAfterExpiry(t *testing.T) {
	w := setup(t)
	r := propose(t, w, Human)
	w.create = false
	w.applyErr = ErrUnknown
	if _, err := w.registry.Approve(intent(r)); !errors.Is(err, ErrUnknown) {
		t.Fatal(err)
	}
	w.now = r.Proposal.Spec.NotAfter
	w.applyErr = nil
	w.create = true
	if _, err := w.registry.Approve(intent(r)); !errors.Is(err, ErrConflict) || w.applied != 1 || stored(t, w).State != Unknown {
		t.Fatal("expired unknown restarted", err)
	}
}

func TestMaximumValidityDoesNotAcceptDurationOverflow(t *testing.T) {
	w := setup(t)
	w.registry.limits.MaxGrantValidity = time.Duration(1<<63 - 1)
	p := w.proposal(Console)
	p.Spec.NotAfter = time.Date(9000, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err := w.registry.Propose(p); !errors.Is(err, ErrInvalid) {
		t.Fatal("duration overflow admitted grant", err)
	}
}

func TestPinnedUnknownMayContinueAfterProposalExpiry(t *testing.T) {
	w := setup(t)
	r := propose(t, w, Human)
	w.create = false
	w.applyErr = ErrUnknown
	if _, err := w.registry.Approve(intent(r)); !errors.Is(err, ErrUnknown) {
		t.Fatal(err)
	}
	first := stored(t, w)
	w.now = r.ExpiresAt.Add(time.Minute)
	w.create = true
	w.applyErr = nil
	got, err := w.registry.Approve(intent(r))
	if err != nil || got.State != Applied || got.Decision.At != first.Decision.At || w.applied != 2 {
		t.Fatal("pinned decision did not continue", err)
	}
}

func TestAppliedReceiptIsHistoricalAfterRevocation(t *testing.T) {
	w := setup(t)
	r := propose(t, w, Console)
	got, err := w.registry.Approve(intent(r))
	if err != nil {
		t.Fatal(err)
	}
	effect := w.effects[got.Decision.GrantID]
	effect.Status = RevokedGrant
	w.effects[got.Decision.GrantID] = effect
	replay, err := w.registry.Approve(intent(r))
	if err != nil || replay.Decision.GrantStatus != ActiveGrant || w.applied != 1 {
		t.Fatal("historical receipt rewritten", err)
	}
	if w.effects[got.Decision.GrantID].Status != RevokedGrant {
		t.Fatal("revoked grant reactivated")
	}
}

func TestFutureGrantWaitsForExactValidityStart(t *testing.T) {
	w := setup(t)
	p := w.proposal(Console)
	p.Spec.NotBefore = w.now.Add(10 * time.Minute)
	r, err := w.registry.Propose(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.registry.Approve(intent(r)); !errors.Is(err, ErrConflict) || stored(t, w).State != Pending || w.applied != 0 {
		t.Fatal("future grant applied early", err)
	}
	w.now = p.Spec.NotBefore
	if _, err := w.registry.Approve(intent(r)); err != nil || w.applied != 1 {
		t.Fatal("grant start boundary refused", err)
	}
}

func TestChangedRefusalReturnsRecordOnlyOnce(t *testing.T) {
	for _, cause := range []string{Expired, Superseded} {
		t.Run(cause, func(t *testing.T) {
			w := setup(t)
			r := propose(t, w, Console)
			if cause == Expired {
				w.now = r.ExpiresAt
			} else {
				w.target = "changed"
			}
			changed, err := w.registry.Approve(intent(r))
			if !errors.Is(err, ErrConflict) || changed.Proposal.ID != "p1" || changed.State != cause {
				t.Fatal("changed refusal missing receipt", err)
			}
			unchanged, err := w.registry.Approve(intent(r))
			if !errors.Is(err, ErrConflict) || unchanged.Proposal.ID != "" {
				t.Fatal("unchanged refusal returned change receipt", err)
			}
		})
	}
}

func TestRecordsAreSortedDetachedBoundedAndFallible(t *testing.T) {
	w := setup(t)
	for _, id := range []string{"z", "a", "m"} {
		p := w.proposal(Console)
		p.ID = id
		if _, err := w.registry.Propose(p); err != nil {
			t.Fatal(err)
		}
	}
	records, err := w.registry.Records("w1")
	if err != nil || len(records) != 3 || records[0].Proposal.ID != "a" || records[2].Proposal.ID != "z" {
		t.Fatal("unordered records", err)
	}
	records[0].Proposal.Proposer = "changed"
	again, err := w.registry.Records("w1")
	if err != nil || again[0].Proposal.Proposer != "alice" {
		t.Fatal("record alias", err)
	}
	foreign, err := w.registry.Records("w2")
	if err != nil || len(foreign) != 0 {
		t.Fatal("foreign records exposed", err)
	}
	w.now = time.Time{}
	if records, err := w.registry.Records("w1"); !errors.Is(err, ErrInvalid) || records != nil {
		t.Fatal("enumeration failure masqueraded as records", err)
	}
}

func TestCapabilityIsClosedAndBoundToProposalAndEffect(t *testing.T) {
	for _, capability := range []string{"", "approve", "view", "CONFIRM"} {
		w := setup(t)
		p := w.proposal(Console)
		p.Spec.Capability = capability
		if _, err := w.registry.Propose(p); !errors.Is(err, ErrInvalid) {
			t.Fatal("invalid capability accepted", capability, err)
		}
	}
	w := setup(t)
	p := w.proposal(Console)
	first, err := w.registry.Propose(p)
	if err != nil {
		t.Fatal(err)
	}
	p.Spec.Capability = Inspect
	if _, err := w.registry.Propose(p); !errors.Is(err, ErrConflict) {
		t.Fatal("same proposal changed capability", err)
	}
	p.ID = "inspection"
	second, err := w.registry.Propose(p)
	if err != nil || second.Digest == first.Digest {
		t.Fatal("capability not bound", err)
	}
	w.applyErr = ErrUnknown
	if _, err := w.registry.Approve(intent(second)); !errors.Is(err, ErrUnknown) {
		t.Fatal(err)
	}
	id := "input-grant-" + second.Digest
	effect := w.effects[id]
	effect.Spec.Capability = Confirm
	w.effects[id] = effect
	w.applyErr = nil
	if _, err := w.registry.Approve(intent(second)); !errors.Is(err, ErrUnknown) {
		t.Fatal("other capability reconciled", err)
	}
}
