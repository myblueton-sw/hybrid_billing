package licensemode

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDurableStatusFailureDoesNotExposeActiveOrExpiredResult(t *testing.T) {
	f, d, dir := durableWorld(t)
	f.must(d.SelectMode(ModeEvaluation, ann))
	_, err := d.Activate(f.sign(evaluation("g1")), Acceptance{Actor: ann.ID, Terms: "eval-terms-v1"}, ann)
	f.must(err)
	f.advance(336 * time.Hour)
	d.store.fault = func(s string) bool { return s == "directory_sync" }
	view, err := d.Status()
	if !errors.Is(err, ErrStorage) || view.State != "" || view.Activation != nil {
		t.Fatal("status returned after uncertain commit")
	}
	d = reopenDurable(t, f, d, dir)
	if !d.registry.grants["g1"].expired {
		t.Fatal("committed expiry disappeared")
	}
	if d.registry.grants["g1"] != d.registry.installations["i1"].activation {
		t.Fatal("grant reference detached")
	}
}
func TestDurableSaveFailureOverridesDomainRefusal(t *testing.T) {
	_, d, _ := durableWorld(t)
	d.store.fault = func(s string) bool { return s == "write" }
	if err := d.SelectMode(ModeEvaluation, bot); !errors.Is(err, ErrStorage) {
		t.Fatal(err)
	}
	if _, err := d.Key("k1"); !errors.Is(err, ErrStorage) {
		t.Fatal("read bypassed failed save")
	}
}
func TestDurableRejectsChangedReferencesAndIntervals(t *testing.T) {
	changes := map[string]func(*checkpoint){
		"interval":            func(c *checkpoint) { g := c.Grants["g1"]; g.View.End = g.View.End.Add(time.Second); c.Grants["g1"] = g },
		"unknown key":         func(c *checkpoint) { g := c.Grants["g1"]; g.View.KeyID = "missing"; c.Grants["g1"] = g },
		"duplicate scope":     func(c *checkpoint) { g := c.Grants["g1"]; g.View.Scope = []string{"core", "core"}; c.Grants["g1"] = g },
		"missing grant":       func(c *checkpoint) { c.State.Grant = "missing" },
		"foreign claim":       func(c *checkpoint) { e := c.Documents["e1"]; e.Claims.Installation = "foreign"; c.Documents["e1"] = e },
		"duplicate documents": func(c *checkpoint) { c.State.Documents = append(c.State.Documents, "e1") },
		"order":               func(c *checkpoint) { e := c.Documents["e1"]; e.Order = 0; c.Documents["e1"] = e },
		"counter":             func(c *checkpoint) { c.Counter = 0 },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			f, d, dir := durableWorld(t)
			f.must(d.SelectMode(ModeEvaluation, ann))
			_, err := d.Activate(f.sign(evaluation("g1")), Acceptance{Actor: ann.ID, Terms: "eval-terms-v1"}, ann)
			f.must(err)
			f.must(d.SelectMode(ModeCommercial, ann))
			_, err = d.Register(f.sign(commercial("e1", "agreement", start.Add(-time.Hour), start.Add(time.Hour))), ann)
			f.must(err)
			f.must(d.Close())
			mutateCheckpoint(t, dir, change)
			if _, err = OpenDurable(dir, "i1", "hybrid-billing", Config{Authority: f.auth}, f.r.clock); !errors.Is(err, ErrCheckpoint) {
				t.Fatal(err)
			}
		})
	}
}
func TestDurableRejectsUnknownFieldsDuplicatesTrailingAndDamage(t *testing.T) {
	for _, kind := range []string{"unknown", "duplicate", "trailing", "damage"} {
		t.Run(kind, func(t *testing.T) {
			f, d, dir := durableWorld(t)
			f.must(d.Close())
			path := filepath.Join(dir, "state.json")
			data, err := os.ReadFile(path)
			f.must(err)
			switch kind {
			case "unknown":
				data = append([]byte(`{"unknown":1,`), data[1:]...)
			case "duplicate":
				data = append([]byte(`{"SHA256":"ignored",`), data[1:]...)
			case "trailing":
				data = append(data, []byte("{}")...)
			case "damage":
				data = bytes.Replace(data, []byte("hybrid-billing"), []byte("hybrid-billinx"), 1)
			}
			f.must(os.WriteFile(path, data, 0600))
			if _, err = OpenDurable(dir, "i1", "hybrid-billing", Config{Authority: f.auth}, f.r.clock); !errors.Is(err, ErrCheckpoint) {
				t.Fatal(err)
			}
		})
	}
}
func TestDurableFileBoundsAndSymlinks(t *testing.T) {
	f, d, dir := durableWorld(t)
	f.must(d.Close())
	path := filepath.Join(dir, "state.json")
	original, err := os.ReadFile(path)
	f.must(err)
	f.must(os.Chmod(path, 0644))
	if _, err = OpenDurable(dir, "i1", "hybrid-billing", Config{Authority: f.auth}, f.r.clock); !errors.Is(err, ErrStorage) {
		t.Fatal(err)
	}
	f.must(os.Chmod(path, 0600))
	f.must(os.Remove(path))
	target := filepath.Join(t.TempDir(), "original")
	f.must(os.WriteFile(target, original, 0600))
	f.must(os.Symlink(target, path))
	if _, err = OpenDurable(dir, "i1", "hybrid-billing", Config{Authority: f.auth}, f.r.clock); !errors.Is(err, ErrStorage) {
		t.Fatal(err)
	}
	f.must(os.Remove(path))
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0600)
	f.must(err)
	f.must(file.Truncate(maxCheckpointBytes + 1))
	f.must(file.Close())
	if _, err = OpenDurable(dir, "i1", "hybrid-billing", Config{Authority: f.auth}, f.r.clock); !errors.Is(err, ErrCheckpoint) {
		t.Fatal(err)
	}
}
func TestDurableStaleOwnerIsNeverDeleted(t *testing.T) {
	f, d, dir := durableWorld(t)
	f.must(d.Close())
	owner := filepath.Join(dir, "owner")
	f.must(os.WriteFile(owner, []byte("stale owner"), 0600))
	if _, err := OpenDurable(dir, "i1", "hybrid-billing", Config{Authority: f.auth}, f.r.clock); !errors.Is(err, ErrOwned) {
		t.Fatal(err)
	}
	data, err := os.ReadFile(owner)
	f.must(err)
	if string(data) != "stale owner" {
		t.Fatal("ownership marker modified")
	}
}

func TestDurableDeletedCheckpointIsNotRecreated(t *testing.T) {
	f, d, dir := durableWorld(t)
	path := filepath.Join(dir, "state.json")
	f.must(os.Remove(path))
	if err := d.SelectMode(ModeNoncommercial, ann); !errors.Is(err, ErrStorage) {
		t.Fatal(err)
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("deleted state was reconstructed")
	}
}
func TestDurableExhaustedProposalCounterCannotWrap(t *testing.T) {
	f, d, dir := durableWorld(t)
	f.must(d.Close())
	mutateCheckpoint(t, dir, func(c *checkpoint) { c.Counter = ^uint64(0) })
	d = reopenDurable(t, f, d, dir)
	p, err := d.Propose(Change{Kind: ChangeCompromise, Compromise: "k1"}, ann)
	if !errors.Is(err, ErrLimit) || p.Number != 0 {
		t.Fatal("counter wrapped")
	}
	d = reopenDurable(t, f, d, dir)
	if d.registry.counter != ^uint64(0) {
		t.Fatal("counter changed")
	}
}
