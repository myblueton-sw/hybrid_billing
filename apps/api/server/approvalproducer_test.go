package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/rating"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func candidateFixture() chargeInput {
	return chargeInput{Workspace: "workspace", ID: "run", Customer: "customer", Entity: "entity", From: "2026-10-01", To: "2026-10-31",
		Sources: []chargeSource{{ID: "source", Revision: "v1", Digest: strings.Repeat("a", 64)}},
		Profile: decimal.ProfileSpec{ID: "synthetic", Revision: "1", InputIntegerDigits: 12, InputScale: 4, ResultIntegerDigits: 14, ResultScale: 4, IntermediateDigits: 60},
		Limits:  rating.Limits{Rules: 8, Tiers: 4, Discounts: 3, Lines: 50, Intermediate: 60},
		Policy: rating.Policy{ID: "policy", Version: "1", Currency: "USD", CostBasis: rating.Billed, Quantum: "0.01", Mode: "half_even", Path: rating.Resale,
			Eligible: []rating.Class{rating.CSPUsage}, Rules: []rating.Rule{{ID: "default", Level: rating.Default, Type: rating.PassThrough}}},
		Lines: []rating.Line{{ID: "one", Class: rating.CSPUsage, Cost: "10.25", Currency: "USD"}, {ID: "two", Class: rating.CSPUsage, Cost: "4.75", Currency: "USD"}}}
}

func TestChargeCandidateComputedAndIsolated(t *testing.T) {
	var store chargeCandidates
	in := candidateFixture()
	made, err := store.capture(in)
	if err != nil {
		t.Fatal(err)
	}
	result := made.Envelope.Result
	if result.Total != "15" || result.Currency != "USD" || result.ProfileID != "synthetic" || result.ProfileRevision != "1" || result.Algorithm != rating.Algorithm || len(result.Charges) != 2 {
		t.Fatalf("unexpected literal result: %+v", result)
	}
	if result.Charges[0].Amount != "10.25" || result.Charges[1].Amount != "4.75" {
		t.Fatal("charge values changed")
	}
	repeated, err := store.capture(in)
	if err != nil || !reflect.DeepEqual(made, repeated) {
		t.Fatal("identical replay changed", err)
	}
	in.Lines[0].Cost = "99"
	in.Policy.Rules[0].ID = "mutated"
	made.Envelope.Input.Sources[0].Digest = strings.Repeat("b", 64)
	made.Envelope.Result.Charges[0].Amount = "99"
	made.Envelope.Result.Traces[0].Steps[0].Name = "mutated"
	stored, err := store.read("workspace", "run")
	if err != nil || !reflect.DeepEqual(stored, repeated) {
		t.Fatal("alias mutated stored evidence", err)
	}
	if _, err := store.read("foreign", "run"); !errors.Is(err, errCandidateMissing) {
		t.Fatal("foreign read", err)
	}
	foreign := candidateFixture()
	foreign.Workspace = "foreign"
	other, err := store.capture(foreign)
	if err != nil || other.Digest == repeated.Digest {
		t.Fatal("workspace not digest-bound", err)
	}
}

func TestChargeCandidateBindingAndConflict(t *testing.T) {
	for name, change := range map[string]func(*chargeInput){
		"customer":        func(in *chargeInput) { in.Customer = "another" },
		"entity":          func(in *chargeInput) { in.Entity = "another" },
		"period":          func(in *chargeInput) { in.To = "2026-11-01" },
		"source revision": func(in *chargeInput) { in.Sources[0].Revision = "v2" },
		"source digest":   func(in *chargeInput) { in.Sources[0].Digest = strings.Repeat("b", 64) },
		"policy revision": func(in *chargeInput) { in.Policy.Version = "2" },
		"profile":         func(in *chargeInput) { in.Profile.Revision = "2" },
		"line":            func(in *chargeInput) { in.Lines[0].Cost = "10.26" },
		"order":           func(in *chargeInput) { in.Lines[0], in.Lines[1] = in.Lines[1], in.Lines[0] },
	} {
		t.Run(name, func(t *testing.T) {
			var first, second chargeCandidates
			before, err := first.capture(candidateFixture())
			if err != nil {
				t.Fatal(err)
			}
			changed := candidateFixture()
			change(&changed)
			if _, err := first.capture(changed); !errors.Is(err, errCandidateConflict) {
				t.Fatal("changed immutable identity accepted", err)
			}
			after, err := second.capture(changed)
			if err != nil || before.Digest == after.Digest {
				t.Fatal("changed input not bound", err)
			}
		})
	}
}

func TestChargeCandidateInvalidInputs(t *testing.T) {
	for name, change := range map[string]func(*chargeInput){
		"workspace":        func(in *chargeInput) { in.Workspace = "" },
		"date":             func(in *chargeInput) { in.From = "2026-02-30" },
		"reversed":         func(in *chargeInput) { in.To = "2026-09-30" },
		"source":           func(in *chargeInput) { in.Sources = nil },
		"duplicate source": func(in *chargeInput) { in.Sources = append(in.Sources, in.Sources[0]) },
		"digest":           func(in *chargeInput) { in.Sources[0].Digest = strings.Repeat("A", 64) },
		"utf8":             func(in *chargeInput) { in.Lines[0].SKU = string([]byte{255}) },
		"huge profile":     func(in *chargeInput) { in.Profile.InputScale = 1000000000 },
		"huge limit":       func(in *chargeInput) { in.Limits.Intermediate = 1000000000 },
		"lines":            func(in *chargeInput) { in.Lines = make([]rating.Line, 1001) },
		"oversize":         func(in *chargeInput) { in.Lines[0].SKU = strings.Repeat("x", candidateBytes) },
		"computation":      func(in *chargeInput) { in.Policy.Rules[0].Type = "invalid" },
	} {
		t.Run(name, func(t *testing.T) {
			var store chargeCandidates
			in := candidateFixture()
			change(&in)
			if _, err := store.capture(in); err == nil {
				t.Fatal("invalid input accepted")
			}
			if len(store.records) != 0 {
				t.Fatal("failed capture published a record")
			}
			if _, err := store.capture(candidateFixture()); err != nil {
				t.Fatal("failure reserved identity", err)
			}
		})
	}
}

func TestChargeCandidateCapacityAndConcurrency(t *testing.T) {
	var store chargeCandidates
	in := candidateFixture()
	made, err := store.capture(in)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i < 1000; i++ {
		store.records[[2]string{"reserved", string(rune(i))}] = chargeRecord{}
	}
	if replay, err := store.capture(in); err != nil || replay.Digest != made.Digest {
		t.Fatal("capacity blocked replay", err)
	}
	in.ID = "next"
	if _, err := store.capture(in); !errors.Is(err, errCandidateLimit) {
		t.Fatal("capacity bypassed", err)
	}
	var concurrent chargeCandidates
	var group sync.WaitGroup
	results := make(chan error, 20)
	for i := 0; i < 20; i++ {
		group.Add(1)
		go func(i int) {
			defer group.Done()
			in := candidateFixture()
			in.Customer = "a"
			if i%2 == 1 {
				in.Customer = "b"
			}
			_, err := concurrent.capture(in)
			results <- err
		}(i)
	}
	group.Wait()
	close(results)
	succeeded, conflicts := 0, 0
	for err := range results {
		if err == nil {
			succeeded++
		} else if errors.Is(err, errCandidateConflict) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if succeeded != 10 || conflicts != 10 || len(concurrent.records) != 1 {
		t.Fatalf("non-atomic capture: %d %d", succeeded, conflicts)
	}
}

func TestChargeCandidatePreservesExclusion(t *testing.T) {
	var store chargeCandidates
	in := candidateFixture()
	in.Lines[0].Class = rating.Other
	made, err := store.capture(in)
	if err != nil {
		t.Fatal(err)
	}
	result := made.Envelope.Result
	if result.Total != "4.75" || len(result.Excluded) != 1 || result.Excluded[0].Line != "one" || result.Excluded[0].Reason != rating.ClassNotEligible {
		t.Fatalf("excluded source concealed: %+v", result)
	}
}

func TestChargeCandidatePreservesHeldRule(t *testing.T) {
	var store chargeCandidates
	in := candidateFixture()
	in.Lines[0].Cost = ""
	made, err := store.capture(in)
	if err != nil {
		t.Fatal(err)
	}
	result := made.Envelope.Result
	if result.Total != "0" || result.NoRuleHeld || len(result.Excluded) != 2 || len(result.Traces) != 1 || result.Excluded[0].Reason != rating.CostNotKnown || result.Excluded[1].Reason != rating.RuleHeld {
		t.Fatalf("held calculation concealed: %+v", result)
	}
}

func TestChargeCandidateDigestContract(t *testing.T) {
	var store chargeCandidates
	made, err := store.capture(candidateFixture())
	if err != nil {
		t.Fatal(err)
	}
	if made.Envelope.Schema != "hb_charge_candidate_v2" {
		t.Fatal("digest schema changed")
	}
	encoded, err := json.Marshal(made.Envelope)
	if err != nil {
		t.Fatal(err)
	}
	expected := sha256.Sum256(encoded)
	if made.Digest != hex.EncodeToString(expected[:]) {
		t.Fatal("digest excludes or changes complete envelope")
	}
	made.Envelope.Result.Total = "16"
	changed, err := json.Marshal(made.Envelope)
	if err != nil {
		t.Fatal(err)
	}
	other := sha256.Sum256(changed)
	if made.Digest == hex.EncodeToString(other[:]) {
		t.Fatal("computed result is not digest bound")
	}
}

func TestChargeCandidateOutputBound(t *testing.T) {
	var store chargeCandidates
	in := candidateFixture()
	in.Limits.Lines = 1000
	in.Sources = nil
	in.Lines = nil
	for i := 0; i < 1000; i++ {
		id := fmt.Sprintf("%s%04d", strings.Repeat("x", 120), i)
		in.Sources = append(in.Sources, chargeSource{ID: id, Revision: strings.Repeat("v", 128), Digest: strings.Repeat("a", 64)})
		in.Lines = append(in.Lines, rating.Line{ID: id, Class: rating.CSPUsage, SKU: strings.Repeat("s", 128), Service: strings.Repeat("s", 128), Group: strings.Repeat("g", 128), Cost: "10.25", Currency: "USD"})
	}
	encoded, err := json.Marshal(in)
	if err != nil || len(encoded) > candidateBytes || !candidateForm(in) {
		t.Fatal("fixture must fit input bound", len(encoded), err)
	}
	if _, err := store.capture(in); !errors.Is(err, errCandidateLimit) {
		t.Fatal("oversized output not rejected", err)
	}
	if len(store.records) != 0 {
		t.Fatal("oversized output was published")
	}
	if _, err := store.capture(candidateFixture()); err != nil {
		t.Fatal("oversized output reserved identity", err)
	}
}
