package pricingpolicy

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestEvidenceIsCurrentWithoutGrantingUse(t *testing.T) {
	f := setup(t)
	p := f.approve(t, "p")
	resolved, err := f.r.Resolve(f.spec.Scope, "p", p.Digest, "consumer", f.spec.From, f.spec.To)
	if err != nil {
		t.Fatal(err)
	}
	f.authority.denied["consumer"] = true
	got, err := f.r.CheckEvidence(f.spec.Scope, "p", p.Digest, f.spec.From, f.spec.To)
	if err != nil || !reflect.DeepEqual(got, resolved) {
		t.Fatal(got, err)
	}
	if _, err := f.r.Resolve(f.spec.Scope, "p", p.Digest, "consumer", f.spec.From, f.spec.To); !errors.Is(err, ErrRejected) {
		t.Fatal("evidence granted use", err)
	}
	got.Proposal.Spec.Policy.Rules[0].ID = "changed"
	got.Proposal.Spec.Policy.Eligible[0] = "changed"
	got.Approval.Actor = "changed"
	again, err := f.r.CheckEvidence(f.spec.Scope, "p", p.Digest, f.spec.From, f.spec.To)
	if err != nil || !reflect.DeepEqual(again, resolved) {
		t.Fatal("aliased evidence", again, err)
	}
}

func TestEvidenceRefusesStaleOrUnavailableFacts(t *testing.T) {
	unavailable := errors.New("synthetic target unavailable")
	for _, name := range []string{"pending", "revoked", "target", "unavailable", "clock", "digest", "scope", "workspace", "from", "to", "invalid_interval", "invalid_digest", "invalid_id", "missing"} {
		t.Run(name, func(t *testing.T) {
			f := setup(t)
			p := f.propose(t, "p")
			if name != "pending" {
				p = f.approve(t, "p")
			}
			scope, id, digest, from, to := f.spec.Scope, "p", p.Digest, f.spec.From, f.spec.To
			want := ErrRejected
			switch name {
			case "revoked":
				if _, err := f.r.Revoke("ws", "p", p.Digest, "revoker", "Stop use"); err != nil {
					t.Fatal(err)
				}
			case "target":
				f.targets.value.Fingerprint = strings.Repeat("b", 64)
				want = ErrConflict
			case "unavailable":
				f.targets.err = unavailable
				want = unavailable
			case "clock":
				f.now = f.now.Add(-time.Second)
			case "digest":
				digest = strings.Repeat("c", 64)
				want = ErrConflict
			case "scope":
				scope.Customer = "other"
				want = ErrConflict
			case "workspace":
				scope.Workspace = "other"
				want = ErrNotFound
			case "from":
				from = from.Add(-time.Second)
			case "to":
				to = to.Add(time.Second)
			case "invalid_interval":
				to = from
				want = ErrInvalid
			case "invalid_digest":
				digest = "invalid"
				want = ErrInvalid
			case "invalid_id":
				id = ""
				want = ErrInvalid
			case "missing":
				id = "missing"
				want = ErrNotFound
			}
			got, err := f.r.CheckEvidence(scope, id, digest, from, to)
			if !errors.Is(err, want) || !reflect.DeepEqual(got, Record{}) {
				t.Fatal(name, got, err, want)
			}
			stored, err := f.r.Record("ws", "p")
			if err != nil || stored.Digest != p.Digest || stored.Proposal.Proposer != p.Proposal.Proposer {
				t.Fatal("refusal changed history", stored, err)
			}
		})
	}
}

func TestEvidenceClockBookkeepingDoesNotRewriteDecision(t *testing.T) {
	f := setup(t)
	p := f.approve(t, "p")
	f.now = f.now.Add(time.Hour)
	got, err := f.r.CheckEvidence(f.spec.Scope, "p", p.Digest, f.spec.From, f.spec.To)
	if err != nil || !reflect.DeepEqual(got, p) {
		t.Fatal(got, err)
	}
	f.now = f.now.Add(-time.Minute)
	if _, err := f.r.CheckEvidence(f.spec.Scope, "p", p.Digest, f.spec.From, f.spec.To); !errors.Is(err, ErrRejected) {
		t.Fatal("rollback accepted", err)
	}
	if _, err := f.r.Resolve(f.spec.Scope, "p", p.Digest, "consumer", f.spec.From, f.spec.To); !errors.Is(err, ErrRejected) {
		t.Fatal("resolve rollback accepted", err)
	}
	stored, err := f.r.Record("ws", "p")
	if err != nil || !reflect.DeepEqual(stored, p) {
		t.Fatal("clock changed decision", stored, err)
	}
}
