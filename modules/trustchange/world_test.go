package trustchange

import (
	"errors"
	"testing"
	"time"
)

var start = time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC)

type world struct {
	t   testing.TB
	r   *Registry
	now *time.Time
}

func setup(t testing.TB) world {
	t.Helper()
	now := start
	r, err := New("0a1b2c3d", func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	return world{t: t, r: r, now: &now}
}

func candidate(minutes int) Candidate {
	return Candidate{Methods: []string{"password"}, MFA: "none", MaxAge: time.Duration(minutes) * time.Minute}
}

var base0 = Base{Epoch: 1, Policy: 1, Digest: ConfigDigest(candidate(720))}

func (f world) propose(by string, minutes int) View {
	f.t.Helper()
	v, err := f.r.Propose(Proposal{Workspace: "w1", Route: RouteLocalPassword, Proposer: by, Candidate: candidate(minutes), Base: base0, Rationale: "why"})
	if err != nil {
		f.t.Fatal(err)
	}
	return v
}

func evidence(tag string) Evidence {
	return Evidence{Digest: EvidenceDigest("x", []string{tag}), Items: []Item{{Item: "test_login", Result: "verified"}}}
}

func (f world) validated(by string, v View) View {
	f.t.Helper()
	got, err := f.r.Validate(v.ID, "w1", by, v.Revision, v.ChangeDigest, base0, evidence("e"), "checked")
	if err != nil {
		f.t.Fatal(err)
	}
	return got
}

func approval(v View, actor string) Approval {
	return Approval{ID: v.ID, Workspace: "w1", Actor: actor, ChangeDigest: v.ChangeDigest, EvidenceDigest: v.Validation.Digest, Recomputed: v.Validation.Digest, Base: base0, Rationale: "ok"}
}

func (f world) approved(by string, v View) View {
	f.t.Helper()
	got, err := f.r.Approve(approval(v, by))
	if err != nil {
		f.t.Fatal(err)
	}
	return got
}

func receipt(epoch uint64) func(View) Outcome {
	return func(View) Outcome {
		return Outcome{Activation: Activation{Epoch: epoch, Sequence: 4, Watermark: "4:x", At: start}}
	}
}

func refused(t testing.TB, name string, err error, reason string) {
	t.Helper()
	var r Refusal
	if !errors.As(err, &r) || r.Reason != reason {
		t.Errorf("%s: %v; want %s", name, err, reason)
	}
}

func second[T any](_ T, err error) error { return err }
