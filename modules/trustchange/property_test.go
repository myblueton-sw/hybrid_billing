package trustchange

import (
	"math/rand/v2"
	"testing"
)

type model struct {
	f      world
	base   Base
	people []string
	epochs []uint64
	once   map[uint64]string
}

func (m *model) evidenceFor(v View, rng *rand.Rand) Evidence {
	ev := evidence(string(rune('a' + rng.IntN(3))))
	ev.Blocked = rng.IntN(5) == 0
	return ev
}

func (m *model) step(rng *rand.Rand) {
	f := m.f
	who := m.people[rng.IntN(len(m.people))]
	list := f.r.List("w1")
	var v View
	open := []View{}
	for _, x := range list {
		if Open(x.State) {
			open = append(open, x)
		}
	}
	switch {
	case len(open) > 0 && rng.IntN(5) > 0:
		v = open[rng.IntN(len(open))]
	case len(list) > 0:
		v = list[rng.IntN(len(list))]
	}
	if rng.IntN(5) > 0 {
		for _, p := range m.people {
			if p != v.Proposer && p != v.Approver && rng.IntN(2) == 0 {
				who = p
			}
		}
	}
	switch [10]int{0, 0, 1, 2, 2, 3, 3, 4, 4, 5}[rng.IntN(10)] + rng.IntN(2)*boolInt(rng.IntN(8) == 0) {
	case 0:
		f.r.Propose(Proposal{Workspace: "w1", Route: RouteLocalPassword, Proposer: who, Candidate: candidate(1 + rng.IntN(40)), Base: m.base})
	case 1:
		f.r.Revise(v.ID, "w1", who, v.Revision, candidate(1+rng.IntN(40)), m.base, "")
	case 2:
		f.r.Validate(v.ID, "w1", who, v.Revision, v.ChangeDigest, m.base, m.evidenceFor(v, rng), "")
	case 3:
		if v.Validation != nil {
			digest := v.Validation.Digest
			if rng.IntN(6) == 0 {
				digest = "drift"
			}
			f.r.Approve(Approval{ID: v.ID, Workspace: "w1", Actor: who, ChangeDigest: v.ChangeDigest, EvidenceDigest: v.Validation.Digest, Recomputed: digest, Base: m.base})
		}
	case 4:
		if v.Validation != nil {
			next := m.base.Epoch + 1
			got, err := f.r.Activate(Approval{ID: v.ID, Workspace: "w1", Actor: who, ChangeDigest: v.ChangeDigest, EvidenceDigest: v.Validation.Digest,
				Recomputed: v.Validation.Digest, Base: m.base}, receipt(next))
			if err == nil {
				m.epochs = append(m.epochs, next)
				if prior, twice := m.once[got.OldEpoch]; twice {
					f.t.Fatalf("two activations on epoch %d: %s and %s", got.OldEpoch, prior, got.ID)
				}
				m.once[got.OldEpoch] = got.ID
				m.base = Base{Epoch: next, Policy: m.base.Policy + 1, Digest: got.NewDigest}
			}
		}
	case 5:
		f.r.Reject(v.ID, "w1", who, v.Revision, "")
	default:
		f.r.Cancel(v.ID, "w1", who, v.Revision, "")
	}
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func TestPropertiesOverRandomSequences(t *testing.T) {
	activations := 0
	for seed := uint64(1); seed <= 60; seed++ {
		rng := rand.New(rand.NewPCG(seed, 99))
		f := setup(t)
		n := 3 + rng.IntN(3)
		m := &model{f: f, base: base0, once: map[uint64]string{}}
		for i := 0; i < n; i++ {
			m.people = append(m.people, "admin"+string(rune('a'+i)))
		}
		terminal := map[string]View{}
		for step := 0; step < 300; step++ {
			m.step(rng)
			for _, v := range f.r.List("w1") {
				if before, ok := terminal[v.ID]; ok && (before.State != v.State || before.Revision != v.Revision || before.Approver != v.Approver) {
					t.Fatalf("seed %d: terminal %s changed from %s to %s", seed, v.ID, before.State, v.State)
				}
				if !Open(v.State) {
					terminal[v.ID] = v
				}
				reached := v.State == StateApproved || v.State == StateActivated
				if reached && (v.Validation == nil || v.Validation.Result != ResultValidated) {
					t.Fatalf("seed %d: %s reached %s without a passing validation", seed, v.ID, v.State)
				}
				if v.State == StateActivated && (v.Proposer == v.Approver || v.Proposer == v.Activator || v.Approver == v.Activator || v.Approver == "") {
					t.Fatalf("seed %d: %s activated by %s/%s/%s", seed, v.ID, v.Proposer, v.Approver, v.Activator)
				}
			}
		}
		activations += len(m.epochs)
		for i := 1; i < len(m.epochs); i++ {
			if m.epochs[i] <= m.epochs[i-1] {
				t.Fatalf("seed %d: epochs %v", seed, m.epochs)
			}
		}
	}
	if activations < 60 {
		t.Errorf("only %d activations over all seeds", activations)
	}
}

func TestRefusalText(t *testing.T) {
	f := setup(t)
	if (Refusal{Reason: "x"}).Error() != "trust change refused: x" {
		t.Error("text")
	}
	refused(t, "approve unknown", second(f.r.Approve(Approval{ID: "x", Workspace: "w1"})), ReasonNotFound)
}
