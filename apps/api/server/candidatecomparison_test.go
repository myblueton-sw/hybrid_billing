package server

import (
	"bytes"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/inputreview"
	"github.com/myblueton-sw/hybrid_billing/modules/normalize"
)

func TestCandidateComparisonActualServerSeed(t *testing.T) {
	f := setup(t, SeedSample).ready()
	defer f.s.Close()
	p := f.s.chargeProducer
	f.s.domain.Lock()
	defer f.s.domain.Unlock()
	before := len(p.candidates.records)
	got, err := p.compareSourceLocked(SampleWorkspace, "sample-normalized-charge", nil)
	if err != nil || got.Evidence.Verdict != inputreview.Match || len(got.Totals) != 1 || got.Totals[0].Expected != "15" || got.Totals[0].Observed != "15" {
		t.Fatal(got.Evidence, got.Totals, err)
	}
	if p.profileSpec != candidateProfile() || len(p.candidates.records) != before {
		t.Fatal("profile or capture changed")
	}
	repeat, err := p.compareSourceLocked(SampleWorkspace, "sample-normalized-charge", nil)
	if err != nil || !bytes.Equal(got.Envelope, repeat.Envelope) {
		t.Fatal("comparison was not stable", err)
	}
	if _, err := p.compareSourceLocked("foreign", "sample-normalized-charge", nil); err == nil {
		t.Fatal("foreign producer evidence")
	}
}

func TestCandidateComparisonCompletePagesAndExplicitUnits(t *testing.T) {
	p, _, packet := normalizedFixtureSetup(t, false, func(rows []normalize.Row) {
		for i := range rows {
			rows[i].Cells["note"] = normalize.Cell{Text: "hours"}
		}
	})
	p.domain.Lock()
	defer p.domain.Unlock()
	if _, err := p.assembleSourceLocked(packet); err != nil {
		t.Fatal(err)
	}
	units := []inputreview.UnitBinding{{Field: "quantity", Source: "note", Target: "note"}}
	snapshot, err := p.comparisonSnapshotLocked("workspace", "normalized", units)
	if err != nil || p.page != 1 || len(snapshot.Rows) != 2 || len(snapshot.Raw) != 2 {
		t.Fatal("complete paging", err)
	}
	got, err := inputreview.CompareSnapshot(snapshot)
	if err != nil || got.Evidence.Verdict != inputreview.Match {
		t.Fatal(got.Evidence, err)
	}
	snapshot.Raw[0].Cells["billed"] = normalize.Cell{Text: "100"}
	snapshot.Target.Fields[0].Name = "changed"
	units[0].Source = "changed"
	again, err := p.compareSourceLocked("workspace", "normalized", []inputreview.UnitBinding{{Field: "quantity", Source: "note", Target: "note"}})
	if err != nil || !bytes.Equal(got.Envelope, again.Envelope) {
		t.Fatal("snapshot alias changed producer", err)
	}
	if _, err := p.compareSourceLocked("workspace", "normalized", nil); err == nil {
		t.Fatal("undeclared quantity units")
	}
}
