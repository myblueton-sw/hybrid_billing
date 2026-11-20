package licensemode

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func durableWorld(t *testing.T) (world, *Durable, string) {
	t.Helper()
	f := setup(t)
	dir := filepath.Join(t.TempDir(), "license")
	d, err := CreateDurable(dir, "i1", "hybrid-billing", ann, Config{Authority: f.auth}, f.r.clock)
	f.must(err)
	t.Cleanup(func() { _ = d.Close() })
	p, err := d.Propose(Change{Kind: ChangeKey, Key: keySpec(f, "k1", "owner")}, ann)
	f.must(err)
	_, err = d.Approve(p.Number, bob)
	f.must(err)
	return f, d, dir
}
func reopenDurable(t *testing.T, f world, d *Durable, dir string) *Durable {
	t.Helper()
	f.must(d.Close())
	next, err := OpenDurable(dir, "i1", "hybrid-billing", Config{Authority: f.auth}, f.r.clock)
	f.must(err)
	t.Cleanup(func() { _ = next.Close() })
	return next
}
func TestDurableEvaluationRestartAndStatusExpiry(t *testing.T) {
	f, d, dir := durableWorld(t)
	f.must(d.SelectMode(ModeEvaluation, ann))
	grant := f.sign(evaluation("g1"))
	original, err := d.Activate(grant, Acceptance{Actor: ann.ID, Terms: "eval-terms-v1"}, ann)
	f.must(err)
	d = reopenDurable(t, f, d, dir)
	f.advance(336 * time.Hour)
	v, err := d.Status()
	f.must(err)
	if v.State != StateEvaluationExpired {
		t.Fatalf("expiry %s", v.State)
	}
	d = reopenDurable(t, f, d, dir)
	again, err := d.Activate(grant, Acceptance{Actor: ann.ID, Terms: "eval-terms-v1"}, ann)
	f.must(err)
	if !again.Start.Equal(original.Start) || !again.End.Equal(original.End) {
		t.Fatal("evaluation restarted")
	}
	f.auth.set(start.Add(time.Hour), true)
	v, err = d.Status()
	f.must(err)
	if v.State != StateUnverified || v.Reason != OutcomeClockAnomaly || v.LastVerified != StateEvaluationExpired {
		t.Fatalf("watermark %+v", v)
	}
	p, err := d.Propose(Change{Kind: ChangeReset, Reset: Reset{At: start.Add(time.Hour).Unix(), Reason: "synthetic reset"}}, ann)
	f.must(err)
	_, err = d.Approve(p.Number, bob)
	f.must(err)
	d = reopenDurable(t, f, d, dir)
	v, err = d.Status()
	f.must(err)
	if v.State != StateEvaluationExpired {
		t.Fatal("expiry latch lost")
	}
	f.auth.set(time.Time{}, false)
	d = reopenDurable(t, f, d, dir)
	v, err = d.Status()
	f.must(err)
	if v.State != StateUnverified || v.Reason != OutcomeTimeUnavailable {
		t.Fatal("local time used as authority")
	}
}
func TestDurableCommercialAndRejectedInput(t *testing.T) {
	f, d, dir := durableWorld(t)
	f.must(d.SelectMode(ModeCommercial, ann))
	signed := f.sign(commercial("e1", "agreement", start.Add(-time.Hour), start.Add(time.Hour)))
	out, err := d.Register(signed, ann)
	f.must(err)
	if out != OutcomeAccepted {
		t.Fatal(out)
	}
	rejected := []byte("sensitive-rejected-raw-input-never-persist")
	out, err = d.Register(Signed{Body: rejected}, ann)
	f.must(err)
	if out != OutcomeInvalid {
		t.Fatal(out)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "state.json"))
	f.must(err)
	if bytes.Contains(raw, rejected) {
		t.Fatal("rejected input persisted")
	}
	d = reopenDurable(t, f, d, dir)
	out, err = d.Register(signed, ann)
	f.must(err)
	if out != OutcomeRepeat {
		t.Fatal("document idempotency lost")
	}
	f.advance(2 * time.Hour)
	v, err := d.Status()
	f.must(err)
	if v.State != StateCommercialExpired {
		t.Fatal(v.State)
	}
	d = reopenDurable(t, f, d, dir)
	if d.registry.installations["i1"].accepted[0] != d.registry.documents["e1"] {
		t.Fatal("document alias lost")
	}
	modes, submissions, err := d.History()
	f.must(err)
	if len(modes) != 1 || len(submissions) != 3 {
		t.Fatal("history lost")
	}
	p, err := d.Propose(Change{Kind: ChangeCompromise, Compromise: "k1"}, ann)
	f.must(err)
	d = reopenDurable(t, f, d, dir)
	_, err = d.Approve(p.Number, bob)
	f.must(err)
	d = reopenDurable(t, f, d, dir)
	key, err := d.Key("k1")
	f.must(err)
	if !key.Compromised {
		t.Fatal("key state lost")
	}
	v, err = d.Status()
	f.must(err)
	if v.State != StateUnverified || v.Reason != ReasonKeyCompromised {
		t.Fatal("compromised rights active")
	}
}
func TestDurableCommitFailurePoisonsAndReopenResolves(t *testing.T) {
	for _, step := range []string{"create", "write", "sync", "close", "rename", "directory_sync"} {
		t.Run(step, func(t *testing.T) {
			f, d, dir := durableWorld(t)
			d.store.fault = func(s string) bool { return s == step }
			if err := d.SelectMode(ModeNoncommercial, ann); !errors.Is(err, ErrStorage) {
				t.Fatal(err)
			}
			v, err := d.Status()
			if !errors.Is(err, ErrStorage) || v.State != "" {
				t.Fatal("uncommitted state exposed")
			}
			if err = d.SelectMode(ModeEvaluation, ann); !errors.Is(err, ErrStorage) {
				t.Fatal("poison bypass")
			}
			d = reopenDurable(t, f, d, dir)
			v, err = d.Status()
			f.must(err)
			want := StateUnselected
			if step == "directory_sync" {
				want = StateNoncommercial
			}
			if v.State != want {
				t.Fatalf("want %s got %s", want, v.State)
			}
		})
	}
}
func TestDurableOwnershipAndExpectedBinding(t *testing.T) {
	f, d, dir := durableWorld(t)
	if _, err := OpenDurable(dir, "i1", "hybrid-billing", Config{Authority: f.auth}, f.r.clock); !errors.Is(err, ErrOwned) {
		t.Fatal(err)
	}
	if _, err := CreateDurable(dir, "i1", "hybrid-billing", ann, Config{Authority: f.auth}, f.r.clock); err == nil {
		t.Fatal("overwrote store")
	}
	f.must(d.Close())
	if _, err := OpenDurable(dir, "foreign", "hybrid-billing", Config{Authority: f.auth}, f.r.clock); !errors.Is(err, ErrCheckpoint) {
		t.Fatal(err)
	}
	d = reopenDurable(t, f, d, dir)
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := d.Status()
			if err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	f.must(d.Close())
	if _, err := d.Status(); !errors.Is(err, ErrStorage) {
		t.Fatal("closed read")
	}
}
func mutateCheckpoint(t *testing.T, dir string, change func(*checkpoint)) {
	t.Helper()
	path := filepath.Join(dir, "state.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var envelope checkpointEnvelope
	var c checkpoint
	if json.Unmarshal(raw, &envelope) != nil || json.Unmarshal(envelope.State, &c) != nil {
		t.Fatal("decode")
	}
	change(&c)
	envelope.State, err = json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(envelope.State)
	envelope.SHA256 = hex.EncodeToString(sum[:])
	raw, err = json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
}
func TestDurableRejectsCorruptOrInconsistentCheckpoints(t *testing.T) {
	cases := map[string]func(*checkpoint){
		"version": func(c *checkpoint) { c.Version++ },
		"binding": func(c *checkpoint) { c.Product = "other" },
		"key proof": func(c *checkpoint) {
			k := c.Keys["k1"]
			k.Key.Proof = string(bytes.Repeat([]byte("0"), 128))
			c.Keys["k1"] = k
		},
		"missing document": func(c *checkpoint) { c.State.Documents = []string{"missing"} },
		"proposal index":   func(c *checkpoint) { c.Pending = []uint64{999} },
		"self approval":    func(c *checkpoint) { p := c.Proposals[1]; p.Approver = p.Proposer; c.Proposals[1] = p },
		"mode":             func(c *checkpoint) { c.State.Mode = "active" },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			f, d, dir := durableWorld(t)
			f.must(d.Close())
			mutateCheckpoint(t, dir, change)
			if _, err := OpenDurable(dir, "i1", "hybrid-billing", Config{Authority: f.auth}, f.r.clock); !errors.Is(err, ErrCheckpoint) {
				t.Fatal(err)
			}
		})
	}
	for _, raw := range []string{"{}", "{broken", `{"SHA256":"x","SHA256":"y","State":{}}`} {
		f, d, dir := durableWorld(t)
		f.must(d.Close())
		f.must(os.WriteFile(filepath.Join(dir, "state.json"), []byte(raw), 0600))
		if _, err := OpenDurable(dir, "i1", "hybrid-billing", Config{Authority: f.auth}, f.r.clock); !errors.Is(err, ErrCheckpoint) {
			t.Fatal(err)
		}
	}
}
