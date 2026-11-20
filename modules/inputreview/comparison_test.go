package inputreview

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/normalize"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func comparisonFixture() Snapshot {
	mapping := normalize.MappingKey{Provider: "provider", SourceVersion: "source-v1", Target: "target"}
	target := normalize.Target{ID: "target", CostCurrency: "currency", Fields: []normalize.Field{
		{Name: "cost", Kind: normalize.Cost, Cost: normalize.Billed, Format: true, Billing: true},
		{Name: "currency", Kind: normalize.Currency, Format: true, Billing: true},
		{Name: "quantity", Kind: normalize.Quantity},
		{Name: "unit", Kind: normalize.Text},
		{Name: "note", Kind: normalize.Text},
	}}
	entries := []normalize.Entry{}
	for _, field := range target.Fields {
		entries = append(entries, normalize.Entry{Field: field.Name, Source: field.Name, SourceKind: field.Kind, SourceCost: field.Cost, Transform: normalize.Copy})
	}
	s := Snapshot{Scope: Scope{"ws", "provider", "account", "2026-10"}, Manifest: 1, Content: strings.Repeat("a", 64), Attribution: strings.Repeat("b", 64), Profile: decimal.ProfileSpec{ID: "exact", Revision: "1", InputIntegerDigits: 8, InputScale: 4, ResultIntegerDigits: 12, ResultScale: 8, IntermediateDigits: 32}, Target: target, Mapping: normalize.RevisionView{Key: mapping, Number: 1, Entries: entries}, Units: []UnitBinding{{"quantity", "unit", "unit"}}}
	s.Run = normalize.RunView{Spec: normalize.RunSpec{WorkspaceID: "ws", ID: "run", Mapping: mapping, Revision: 1, Input: normalize.Input{Scope: "account", Period: "2026-10", Manifest: 1}, Rows: 2}, State: normalize.Closed, Checked: 2, Normal: 2, ProfileID: "exact", ProfileRevision: "1", Format: &normalize.Verdict{Result: normalize.Conformant}, Billing: &normalize.Verdict{Result: normalize.Ready}}
	s.Run.Spec.Expected = map[string]map[string]string{"cost": {"USD": "15"}}
	s.Run.Sums = map[string]map[string]string{"cost": {"USD": "15"}}
	s.Raw = []normalize.Row{
		{Key: normalize.RowKey{Kind: normalize.EventKeys, Key: "a"}, Cells: map[string]normalize.Cell{"cost": {Text: "10.0000"}, "currency": {Text: "USD"}, "quantity": {Text: "2"}, "unit": {Text: "hours"}, "note": {Null: true}}},
		{Key: normalize.RowKey{Kind: normalize.EventKeys, Key: "b"}, Cells: map[string]normalize.Cell{"cost": {Text: "5"}, "currency": {Text: "USD"}, "quantity": {Text: "3"}, "unit": {Text: "hours"}}},
	}
	s.Rows = []normalize.OutRow{
		{Key: normalize.RowKey{Kind: normalize.EventKeys, Key: "a"}, Cells: map[string]normalize.OutCell{"cost": {State: normalize.Value, Text: "10"}, "currency": {State: normalize.Value, Text: "USD"}, "quantity": {State: normalize.Value, Text: "2"}, "unit": {State: normalize.Value, Text: "hours"}, "note": {State: normalize.Null}}},
		{Key: normalize.RowKey{Kind: normalize.EventKeys, Key: "b"}, Cells: map[string]normalize.OutCell{"cost": {State: normalize.Value, Text: "5"}, "currency": {State: normalize.Value, Text: "USD"}, "quantity": {State: normalize.Value, Text: "3"}, "unit": {State: normalize.Value, Text: "hours"}, "note": {State: normalize.NotProvided}}},
	}
	return s
}

func TestComparisonLiteralTotalsAndDeterministicEnvelope(t *testing.T) {
	s := comparisonFixture()
	got, err := CompareSnapshot(s)
	if err != nil || got.Evidence.Verdict != Match || len(got.Findings) != 0 || len(got.Totals) != 2 {
		t.Fatal(got.Evidence, got.Findings, got.Totals, err)
	}
	if got.Totals[0] != (Total{"cost", "billed", "USD", "15", "15"}) || got.Totals[1] != (Total{"quantity", "", "hours", "5", "5"}) {
		t.Fatal(got.Totals)
	}
	for i := 0; i < 20; i++ {
		again, err := CompareSnapshot(s)
		if err != nil || !bytes.Equal(got.Envelope, again.Envelope) {
			t.Fatal("nondeterministic", err)
		}
	}
	var envelope comparisonEnvelope
	if err := json.Unmarshal(got.Envelope, &envelope); err != nil || envelope.Snapshot.Raw[0].Cells["cost"].Text != "10.0000" {
		t.Fatal("raw spelling lost", err)
	}
	previous := append([]byte(nil), got.Envelope...)
	s.Raw[0].Cells["cost"] = normalize.Cell{Text: "10"}
	changed, err := CompareSnapshot(s)
	if err != nil || changed.Evidence.Verdict != Match || changed.Evidence.Digest == got.Evidence.Digest || !bytes.Equal(got.Envelope, previous) {
		t.Fatal("raw bytes not bound", err)
	}
}

func TestComparisonOffsettingErrorsDoNotPass(t *testing.T) {
	s := comparisonFixture()
	s.Rows[0].Cells["cost"] = normalize.OutCell{State: normalize.Value, Text: "11"}
	s.Rows[1].Cells["cost"] = normalize.OutCell{State: normalize.Value, Text: "4"}
	got, err := CompareSnapshot(s)
	if err != nil || got.Evidence.Verdict != Difference || len(got.Findings) != 2 || got.Totals[0].Expected != "15" || got.Totals[0].Observed != "15" {
		t.Fatal(got.Evidence, got.Findings, got.Totals, err)
	}
	for _, finding := range got.Findings {
		if finding.Code != "cell_difference" {
			t.Fatal(finding)
		}
	}
}

func TestComparisonIdentityStateAndDimensionFailures(t *testing.T) {
	for name, change := range map[string]func(*Snapshot){
		"missing":          func(s *Snapshot) { s.Rows = s.Rows[:1] },
		"extra":            func(s *Snapshot) { s.Rows[1].Key.Key = "foreign" },
		"duplicate raw":    func(s *Snapshot) { s.Raw[1].Key = s.Raw[0].Key },
		"duplicate output": func(s *Snapshot) { s.Rows[1].Key = s.Rows[0].Key },
		"null absent":      func(s *Snapshot) { s.Rows[0].Cells["note"] = normalize.OutCell{State: normalize.NotProvided} },
		"currency": func(s *Snapshot) {
			s.Rows[0].Cells["currency"] = normalize.OutCell{State: normalize.Value, Text: "EUR"}
		},
		"unit": func(s *Snapshot) {
			s.Rows[0].Cells["unit"] = normalize.OutCell{State: normalize.Value, Text: "minutes"}
		},
		"field missing":  func(s *Snapshot) { delete(s.Rows[0].Cells, "note") },
		"quarantine":     func(s *Snapshot) { s.Run.Quarantined = 1 },
		"invalid source": func(s *Snapshot) { s.Raw[0].Cells["cost"] = normalize.Cell{Text: "1e2"} },
	} {
		t.Run(name, func(t *testing.T) {
			s := comparisonFixture()
			change(&s)
			got, err := CompareSnapshot(s)
			if err != nil || got.Evidence.Verdict != Difference || len(got.Findings) == 0 {
				t.Fatal(got.Evidence, got.Findings, got.Totals, err)
			}
		})
	}
}

func TestComparisonTransformsAndRefusals(t *testing.T) {
	for _, tc := range []struct {
		op                   normalize.Transform
		power                int
		first, second, total string
	}{
		{normalize.Negate, 0, "-10", "-5", "-15"},
		{normalize.Scale, -2, "0.1", "0.05", "0.15"},
		{normalize.Scale, 2, "1000", "500", "1500"},
	} {
		t.Run(string(tc.op)+tc.total, func(t *testing.T) {
			s := comparisonFixture()
			s.Mapping.Entries[0].Transform = tc.op
			s.Mapping.Entries[0].Power = tc.power
			s.Run.Spec.Expected["cost"]["USD"] = tc.total
			s.Run.Sums["cost"]["USD"] = tc.total
			s.Rows[0].Cells["cost"] = normalize.OutCell{State: normalize.Value, Text: tc.first}
			s.Rows[1].Cells["cost"] = normalize.OutCell{State: normalize.Value, Text: tc.second}
			got, err := CompareSnapshot(s)
			if err != nil || got.Evidence.Verdict != Match || got.Totals[0].Expected != tc.total {
				t.Fatal(got.Evidence, got.Findings, got.Totals, err)
			}
		})
	}
	for name, change := range map[string]func(*Snapshot){
		"unsupported":          func(s *Snapshot) { s.Mapping.Entries[0].Transform = "round" },
		"wrong basis":          func(s *Snapshot) { s.Mapping.Entries[0].SourceCost = normalize.Effective },
		"missing unit binding": func(s *Snapshot) { s.Units = nil },
		"foreign manifest":     func(s *Snapshot) { s.Run.Spec.Input.Manifest = 2 },
		"foreign workspace":    func(s *Snapshot) { s.Run.Spec.WorkspaceID = "other" },
		"profile mismatch":     func(s *Snapshot) { s.Profile.Revision = "2" },
	} {
		t.Run(name, func(t *testing.T) {
			s := comparisonFixture()
			change(&s)
			if _, err := CompareSnapshot(s); err == nil {
				t.Fatal("unsupported snapshot accepted")
			}
		})
	}
}

func TestComparisonNumericBoundsAndEvidenceBindings(t *testing.T) {
	base := comparisonFixture()
	original, err := CompareSnapshot(base)
	if err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(*Snapshot){
		"attribution": func(s *Snapshot) { s.Attribution = strings.Repeat("c", 64) },
		"profile":     func(s *Snapshot) { s.Profile.ResultIntegerDigits++ },
		"mapping":     func(s *Snapshot) { s.Mapping.Number++; s.Run.Spec.Revision++ },
	} {
		t.Run(name, func(t *testing.T) {
			s := comparisonFixture()
			change(&s)
			got, err := CompareSnapshot(s)
			if err != nil || got.Evidence.Verdict != Match || got.Evidence.Digest == original.Evidence.Digest {
				t.Fatal("evidence unbound", err)
			}
		})
	}
	s := comparisonFixture()
	s.Raw[0].Cells["cost"] = normalize.Cell{Text: "999999999"}
	if got, err := CompareSnapshot(s); err != nil || got.Evidence.Verdict != Difference {
		t.Fatal("input bound", err)
	}
	s = comparisonFixture()
	s.Mapping.Entries[0].Transform = normalize.Scale
	s.Mapping.Entries[0].Power = 12
	if got, err := CompareSnapshot(s); err != nil || got.Evidence.Verdict != Difference {
		t.Fatal("result bound", err)
	}
	s = comparisonFixture()
	s.Profile.IntermediateDigits = 1000000
	if _, err := CompareSnapshot(s); !errors.Is(err, ErrLimit) {
		t.Fatal("unbounded profile", err)
	}
}
