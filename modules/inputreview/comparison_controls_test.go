package inputreview

import (
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/normalize"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func TestComparisonRejectsFalseDeclaredAndReportedTotals(t *testing.T) {
	for name, change := range map[string]func(*Snapshot){
		"missing expected":        func(s *Snapshot) { s.Run.Spec.Expected = nil },
		"wrong expected":          func(s *Snapshot) { s.Run.Spec.Expected["cost"]["USD"] = "16" },
		"missing reported":        func(s *Snapshot) { s.Run.Sums = nil },
		"wrong reported":          func(s *Snapshot) { s.Run.Sums["cost"]["USD"] = "16" },
		"extra currency expected": func(s *Snapshot) { s.Run.Spec.Expected["cost"]["EUR"] = "0" },
		"extra currency reported": func(s *Snapshot) { s.Run.Sums["cost"]["EUR"] = "0" },
		"extra field expected":    func(s *Snapshot) { s.Run.Spec.Expected["unknown"] = map[string]string{"USD": "0"} },
		"extra field reported":    func(s *Snapshot) { s.Run.Sums["unknown"] = map[string]string{"USD": "0"} },
	} {
		t.Run(name, func(t *testing.T) {
			s := comparisonFixture()
			change(&s)
			got, err := CompareSnapshot(s)
			if err != nil || got.Evidence.Verdict != Difference {
				t.Fatal("false totals passed", got.Evidence, got.Findings, err)
			}
		})
	}
}

func periodFixture() Snapshot {
	s := comparisonFixture()
	s.Target.PeriodStart = "start"
	s.Target.PeriodEnd = "end"
	for _, name := range []string{"start", "end"} {
		s.Target.Fields = append(s.Target.Fields, normalize.Field{Name: name, Kind: normalize.Time})
		s.Mapping.Entries = append(s.Mapping.Entries, normalize.Entry{Field: name, Source: name, SourceKind: normalize.Time, Transform: normalize.Copy})
	}
	from := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)
	s.Run.Spec.Window = &normalize.Window{From: from, To: to}
	for i := range s.Raw {
		for name, value := range map[string]string{"start": from.Format(time.RFC3339Nano), "end": to.Format(time.RFC3339Nano)} {
			s.Raw[i].Cells[name] = normalize.Cell{Text: value}
			s.Rows[i].Cells[name] = normalize.OutCell{State: normalize.Value, Text: value}
		}
	}
	return s
}

func TestComparisonIndependentlyChecksPeriods(t *testing.T) {
	if got, err := CompareSnapshot(periodFixture()); err != nil || got.Evidence.Verdict != Match {
		t.Fatal("exact boundaries", got.Evidence, got.Findings, err)
	}
	for name, change := range map[string]func(*Snapshot){
		"window absent": func(s *Snapshot) { s.Run.Spec.Window = nil },
		"window inverted": func(s *Snapshot) {
			s.Run.Spec.Window.From, s.Run.Spec.Window.To = s.Run.Spec.Window.To, s.Run.Spec.Window.From
		},
		"outside": func(s *Snapshot) { s.Run.Spec.Window.From = s.Run.Spec.Window.From.Add(time.Second) },
		"equal": func(s *Snapshot) {
			value := s.Raw[0].Cells["start"]
			s.Raw[0].Cells["end"] = value
			s.Rows[0].Cells["end"] = normalize.OutCell{State: normalize.Value, Text: value.Text}
		},
		"inverted": func(s *Snapshot) {
			s.Raw[0].Cells["start"], s.Raw[0].Cells["end"] = s.Raw[0].Cells["end"], s.Raw[0].Cells["start"]
			s.Rows[0].Cells["start"], s.Rows[0].Cells["end"] = s.Rows[0].Cells["end"], s.Rows[0].Cells["start"]
		},
		"missing value": func(s *Snapshot) {
			delete(s.Raw[0].Cells, "start")
			s.Rows[0].Cells["start"] = normalize.OutCell{State: normalize.NotProvided}
		},
	} {
		t.Run(name, func(t *testing.T) {
			s := periodFixture()
			change(&s)
			got, err := CompareSnapshot(s)
			if err != nil || got.Evidence.Verdict != Difference {
				t.Fatal("period passed", got.Evidence, got.Findings, err)
			}
		})
	}
}

func TestComparisonSupportedPowersAgainstDecimalPointOracle(t *testing.T) {
	profile, err := decimal.NewProfile(decimal.ProfileSpec{ID: "powers", Revision: "1", InputIntegerDigits: 20, InputScale: 20, ResultIntegerDigits: 30, ResultScale: 30, IntermediateDigits: 60})
	if err != nil {
		t.Fatal(err)
	}
	for power := -12; power <= 12; power++ {
		if power == 0 {
			continue
		}
		for _, negative := range []bool{false, true} {
			input := "1.2500"
			expected := ""
			places := 2 - power
			if places <= 0 {
				expected = "125" + strings.Repeat("0", -places)
			} else if places < 3 {
				expected = "125"[:3-places] + "." + "125"[3-places:]
			} else {
				expected = "0." + strings.Repeat("0", places-3) + "125"
			}
			if negative {
				input = "-" + input
				expected = "-" + expected
			}
			cell, err := expectedCell(profile, normalize.Field{Name: "cost", Kind: normalize.Cost, Cost: normalize.Billed}, normalize.Entry{Source: "raw", Transform: normalize.Scale, Power: power}, map[string]normalize.Cell{"raw": {Text: input}})
			if err != nil || cell.State != normalize.Value || cell.Text != expected {
				t.Fatal(power, input, cell, expected, err)
			}
		}
	}
}
