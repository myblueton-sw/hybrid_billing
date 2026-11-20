package inputreview

import (
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/normalize"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func periodFindings(s Snapshot, cells map[string]normalize.OutCell, key normalize.RowKey) []Finding {
	if s.Target.PeriodStart == "" {
		return nil
	}
	finding := func(code string) []Finding { return []Finding{{code, key, s.Target.PeriodStart}} }
	start, end := cells[s.Target.PeriodStart], cells[s.Target.PeriodEnd]
	if start.State != normalize.Value || end.State != normalize.Value {
		return finding("period_value_missing")
	}
	from, err := time.Parse(time.RFC3339Nano, start.Text)
	if err != nil {
		return finding("period_value_invalid")
	}
	to, err := time.Parse(time.RFC3339Nano, end.Text)
	if err != nil {
		return finding("period_value_invalid")
	}
	if !from.Before(to) {
		return finding("period_inverted")
	}
	window := s.Run.Spec.Window
	if window == nil {
		return finding("period_window_missing")
	}
	if !window.From.Before(window.To) || window.From.UTC().Year() < 0 || window.To.UTC().Year() > 9999 {
		return finding("period_window_invalid")
	}
	if from.Before(window.From) || to.After(window.To) {
		return finding("period_outside_window")
	}
	return nil
}

func declaredTotalFindings(profile decimal.Profile, s Snapshot, fields map[string]normalize.Field, totals map[totalKey]*comparisonTotal) []Finding {
	findings := []Finding{}
	expected, observed := map[string]map[string]string{}, map[string]map[string]string{}
	for key, total := range totals {
		if fields[key.field].Kind != normalize.Cost {
			continue
		}
		for _, actual := range []bool{false, true} {
			value, exists, destination := total.expected, total.hasExpected, expected
			if actual {
				value, exists, destination = total.observed, total.hasObserved, observed
			}
			if !exists {
				continue
			}
			text, err := exactTotal(profile, value)
			if err != nil {
				findings = append(findings, Finding{"total_outside_profile", normalize.RowKey{}, key.field})
				continue
			}
			if destination[key.field] == nil {
				destination[key.field] = map[string]string{}
			}
			destination[key.field][key.unit] = text
		}
	}
	compare := func(stated, derived map[string]map[string]string, required map[string]bool, code string, exactFields bool) {
		keys := map[string]bool{}
		for key := range stated {
			keys[key] = true
		}
		for key := range required {
			keys[key] = true
		}
		for key := range keys {
			left, present := stated[key]
			right, derivedField := derived[key]
			bad := !present || exactFields && !derivedField || fields[key].Kind != normalize.Cost || len(left) != len(right)
			for currency, text := range left {
				a, err := resultNumber(profile, text)
				value, exists := right[currency]
				if err != nil || !exists || !currencyCode(currency) || a.String() != value {
					bad = true
				}
			}
			if bad {
				findings = append(findings, Finding{code, normalize.RowKey{}, key})
			}
		}
	}
	required := map[string]bool{}
	for name, field := range fields {
		if field.Kind == normalize.Cost && field.Billing {
			required[name] = true
		}
	}
	compare(s.Run.Spec.Expected, expected, required, "declared_total_difference", false)
	reported := map[string]bool{}
	for name := range observed {
		reported[name] = true
	}
	compare(s.Run.Sums, observed, reported, "reported_total_difference", true)
	return findings
}
