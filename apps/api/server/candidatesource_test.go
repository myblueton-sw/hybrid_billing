package server

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/intake"
	"github.com/myblueton-sw/hybrid_billing/modules/normalize"
)

func sourcePacketCopy(t *testing.T, packet sourcePacket) sourcePacket {
	t.Helper()
	data, err := json.Marshal(packet)
	if err != nil {
		t.Fatal(err)
	}
	var out sourcePacket
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestSourceAssemblyActualPipelineAndIsolation(t *testing.T) {
	p, b, packet := normalizedFixtureSetup(t, false, nil)
	origin, err := p.assembleSource(packet)
	if err != nil {
		t.Fatal(err)
	}
	if origin.Manifest != 1 || origin.Provider != "provider" || origin.Scope != "scope" || origin.Period != "october" || len(origin.Rows) != 2 || origin.Content == "" || origin.Attribution == "" || origin.Content == origin.Attribution || origin.Admission.Difference != "same" {
		t.Fatalf("origin: %+v", origin)
	}
	run, err := p.normal.Run("workspace", "normalized")
	if err != nil || run.Spec.Input.Manifest != origin.Manifest || run.Spec.Rows != 2 || run.State != normalize.Closed {
		t.Fatal("unbound normalization", run, err)
	}
	first, err := p.capture("workspace", "normalized", "candidate", b)
	if err != nil || first.Envelope.Result.Total != "5" || !reflect.DeepEqual(first.Envelope.Input.Normalized.Source, origin) {
		t.Fatal("unbound candidate", err)
	}
	replayed, err := p.assembleSource(packet)
	if err != nil || !reflect.DeepEqual(replayed, origin) {
		t.Fatal("assembly replay changed", err)
	}
	again, err := p.capture("workspace", "normalized", "candidate", b)
	if err != nil || !reflect.DeepEqual(first, again) {
		t.Fatal("candidate replay changed", err)
	}
	packet.Artifacts[0].Rows[0].Cells["billed"] = normalize.Cell{Text: "99"}
	if _, err := p.assembleSource(packet); !errors.Is(err, errCandidateConflict) {
		t.Fatal("changed packet reused ID", err)
	}
	origin.Rows[0].Digest = "changed"
	first.Envelope.Input.Normalized.Source.Rows[0].Amount = "99"
	fresh, err := p.candidates.read("workspace", "candidate")
	if err != nil || !reflect.DeepEqual(fresh, again) {
		t.Fatal("caller/return alias changed stored record", err)
	}
	if len(p.candidates.records) != 1 {
		t.Fatal("replay published another candidate")
	}
}

func TestSourceAssemblyExactAmountAndRawIdentity(t *testing.T) {
	var origins []candidateSource
	for _, amount := range []string{"2", "2.0000"} {
		p, b, packet := normalizedFixtureSetup(t, false, nil)
		packet.Artifacts[0].Rows[0].Cells["billed"] = normalize.Cell{Text: amount}
		origin, err := p.assembleSource(packet)
		if err != nil || origin.Rows[0].Amount != "2" {
			t.Fatal("canonical amount", origin, err)
		}
		made, err := p.capture("workspace", "normalized", "candidate", b)
		if err != nil || made.Envelope.Result.Total != "5" {
			t.Fatal("equivalent amount changed result", err)
		}
		origins = append(origins, origin)
	}
	if origins[0].Rows[0].Digest == origins[1].Rows[0].Digest || origins[0].Content == origins[1].Content {
		t.Fatal("distinct raw spelling lost")
	}
	for _, amount := range []string{"3", "+2", "2.00000", "999999999999999999999999999999"} {
		p, b, packet := normalizedFixtureSetup(t, false, nil)
		packet.Artifacts[0].Rows[0].Cells["billed"] = normalize.Cell{Text: amount}
		if _, err := p.assembleSource(packet); err == nil {
			t.Fatal("unequal or invalid amount accepted", amount)
		}
		if _, err := p.capture("workspace", "normalized", "candidate", b); err == nil || len(p.candidates.records) != 0 {
			t.Fatal("failed source published candidate")
		}
	}
}

func TestSourceAssemblyPartialRetryAndReservations(t *testing.T) {
	p, b, packet := normalizedFixtureSetup(t, false, nil)
	p.domain.Lock()
	prepared, err := p.prepareSource(packet)
	if err != nil {
		t.Fatal(err)
	}
	held, err := p.reserveSource(prepared)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.stageSourceArtifact(prepared, held, 0); err != nil {
		t.Fatal(err)
	}
	p.domain.Unlock()
	if _, err := p.capture("workspace", "normalized", "candidate", b); err == nil {
		t.Fatal("partial source captured")
	}
	changed := sourcePacketCopy(t, packet)
	changed.Artifacts[1].Rows[0].Cells["note"] = normalize.Cell{Text: "different"}
	if _, err := p.assembleSource(changed); !errors.Is(err, errCandidateConflict) {
		t.Fatal("changed partial retry accepted", err)
	}
	other := sourcePacketCopy(t, packet)
	other.Run.ID = "another-run"
	if _, err := p.assembleSource(other); !errors.Is(err, errCandidateConflict) {
		t.Fatal("reserved artifact adopted", err)
	}
	if _, err := p.intake.Artifact("workspace", packet.Artifacts[1].ID); !errors.Is(err, intake.ErrNotFound) {
		t.Fatal("conflicting retry wrote artifact", err)
	}
	if _, err := p.assembleSource(packet); err != nil {
		t.Fatal("identical retry failed", err)
	}
	if _, err := p.capture("workspace", "normalized", "candidate", b); err != nil {
		t.Fatal("failed capture reserved ID", err)
	}
}

func TestSourceAssemblyPartialNormalizationRetry(t *testing.T) {
	p, b, packet := normalizedFixtureSetup(t, false, nil)
	p.sourceChunk = 1
	p.domain.Lock()
	prepared, err := p.prepareSource(packet)
	if err != nil {
		t.Fatal(err)
	}
	held, err := p.reserveSource(prepared)
	if err != nil {
		t.Fatal(err)
	}
	for i := range prepared.specs {
		if err := p.stageSourceArtifact(prepared, held, i); err != nil {
			t.Fatal(err)
		}
	}
	published, err := p.intake.Publish(prepared.subject, held.Base)
	if err != nil {
		t.Fatal(err)
	}
	held.Manifest = published.Number
	spec, err := p.sourceRun(prepared, held)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.normal.Open(spec); err != nil {
		t.Fatal(err)
	}
	held.Normalized = true
	if _, err := p.normal.Check("workspace", "normalized", 1, prepared.rows[:1]); err != nil {
		t.Fatal(err)
	}
	p.domain.Unlock()
	if _, err := p.capture("workspace", "normalized", "candidate", b); err == nil {
		t.Fatal("partial normalization captured")
	}
	if _, err := p.assembleSource(packet); err != nil {
		t.Fatal("normalization retry", err)
	}
	if _, err := p.capture("workspace", "normalized", "candidate", b); err != nil {
		t.Fatal("retry capture", err)
	}
}

func TestSourceAssemblyRefusesUnrelatedRecords(t *testing.T) {
	for _, stage := range []string{"artifact", "run", "after-reservation"} {
		t.Run(stage, func(t *testing.T) {
			p, b, packet := normalizedFixtureSetup(t, false, nil)
			p.domain.Lock()
			prepared, err := p.prepareSource(packet)
			if err != nil {
				t.Fatal(err)
			}
			if stage == "after-reservation" {
				if _, err := p.reserveSource(prepared); err != nil {
					t.Fatal(err)
				}
			}
			if stage == "run" {
				spec, err := p.sourceRun(prepared, &sourceAssembly{Manifest: 1})
				if err != nil {
					t.Fatal(err)
				}
				if _, err := p.normal.Open(spec); err != nil {
					t.Fatal(err)
				}
			} else {
				if _, _, err := p.intake.Register(prepared.specs[0]); err != nil {
					t.Fatal(err)
				}
			}
			p.domain.Unlock()
			if _, err := p.assembleSource(packet); !errors.Is(err, errCandidateConflict) {
				t.Fatal("unrelated record adopted", err)
			}
			if _, err := p.capture("workspace", "normalized", "candidate", b); err == nil || len(p.candidates.records) != 0 {
				t.Fatal("unowned run captured")
			}
		})
	}
}
