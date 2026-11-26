package inputreview

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

	"github.com/myblueton-sw/hybrid_billing/modules/normalize"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

const ComparisonMethod = "raw-normalized-v1"

type UnitBinding struct {
	Field  string
	Source string
	Target string
}

type Snapshot struct {
	Scope       Scope
	Manifest    uint64
	Content     string
	Attribution string
	Profile     decimal.ProfileSpec
	Target      normalize.Target
	Mapping     normalize.RevisionView
	Run         normalize.RunView
	Raw         []normalize.Row
	Rows        []normalize.OutRow
	Units       []UnitBinding
}

type Finding struct {
	Code  string
	Key   normalize.RowKey
	Field string
}

type Total struct {
	Field    string
	Basis    string
	Unit     string
	Expected string
	Observed string
}

type Comparison struct {
	Evidence Evidence
	Findings []Finding
	Totals   []Total
	Envelope []byte
}

type comparisonEnvelope struct {
	Method   string
	Snapshot Snapshot
	Findings []Finding
	Totals   []Total
}

func CompareSnapshot(snapshot Snapshot) (Comparison, error) {
	out := Comparison{}
	profile, fields, entries, units, err := validateSnapshot(snapshot)
	if err != nil {
		return out, err
	}
	raw := make(map[normalize.RowKey]normalize.Row)
	observed := make(map[normalize.RowKey]normalize.OutRow)
	findings := []Finding{}
	add := func(code string, key normalize.RowKey, field string) {
		findings = append(findings, Finding{code, key, field})
	}
	for _, row := range snapshot.Raw {
		if _, exists := raw[row.Key]; exists {
			add("duplicate_raw_identity", row.Key, "")
		} else {
			raw[row.Key] = row
		}
	}
	for _, row := range snapshot.Rows {
		if _, exists := observed[row.Key]; exists {
			add("duplicate_observed_identity", row.Key, "")
		} else {
			observed[row.Key] = row
		}
	}
	if snapshot.Run.State != normalize.Closed || snapshot.Run.Errors != 0 || snapshot.Run.Quarantined != 0 || snapshot.Run.Contradiction != 0 || snapshot.Run.Checked != len(snapshot.Raw) || snapshot.Run.Normal != len(snapshot.Raw) || snapshot.Run.Spec.Rows != len(snapshot.Raw) || len(snapshot.Rows) != len(snapshot.Raw) {
		add("incomplete_dispositions", normalize.RowKey{}, "")
	}
	if snapshot.Run.Format == nil || snapshot.Run.Billing == nil || snapshot.Run.Format.Result != normalize.Conformant || snapshot.Run.Billing.Result != normalize.Ready {
		add("normalization_not_ready", normalize.RowKey{}, "")
	}
	totals := make(map[totalKey]*comparisonTotal)
	for key, row := range raw {
		actual, exists := observed[key]
		if !exists {
			add("missing_observed_identity", key, "")
		}
		expected := make(map[string]normalize.OutCell)
		for name, field := range fields {
			cell, err := expectedCell(profile, field, entries[name], row.Cells)
			if err != nil {
				add("source_cell_invalid", key, name)
				continue
			}
			expected[name] = cell
			if (field.Format || field.Billing) && cell.State != normalize.Value {
				add("required_value_missing", key, name)
			}
			if exists && actual.Cells[name] != cell {
				add("cell_difference", key, name)
			}
		}
		for _, finding := range periodFindings(snapshot, expected, key) {
			findings = append(findings, finding)
		}
		if exists {
			for name := range actual.Cells {
				if _, known := fields[name]; !known {
					add("unexpected_observed_field", key, name)
				}
			}
			if len(actual.Cells) != len(fields) {
				add("observed_field_coverage", key, "")
			}
		}
		for name, field := range fields {
			if field.Kind != normalize.Cost && field.Kind != normalize.Quantity {
				continue
			}
			for _, side := range []bool{false, true} {
				cells := expected
				if side {
					if !exists {
						continue
					}
					cells = actual.Cells
				}
				cell, known := cells[name]
				if !known || cell.State != normalize.Value {
					continue
				}
				unit := ""
				if field.Kind == normalize.Cost {
					currency := cells[snapshot.Target.CostCurrency]
					if currency.State != normalize.Value || !currencyCode(currency.Text) {
						add("currency_missing", key, name)
						continue
					}
					unit = currency.Text
				} else {
					binding, known := units[name]
					if !known {
						return Comparison{}, ErrRejected
					}
					unitCell := row.Cells[binding.Source]
					if unitCell.Null || unitCell.Text == "" {
						add("quantity_unit_missing", key, name)
						continue
					}
					unit = unitCell.Text
					if side {
						value := cells[binding.Target]
						if value.State != normalize.Value || value.Text == "" {
							add("quantity_unit_missing", key, name)
							continue
						}
						unit = value.Text
					}
				}
				if err := accumulate(profile, totals, totalKey{name, string(field.Cost), unit}, cell.Text, side); err != nil {
					add("numeric_observation_invalid", key, name)
				}
			}
		}
	}
	for key := range observed {
		if _, exists := raw[key]; !exists {
			add("extra_observed_identity", key, "")
		}
	}
	findings = append(findings, declaredTotalFindings(profile, snapshot, fields, totals)...)
	for key, value := range totals {
		left, err := exactTotal(profile, value.expected)
		if err != nil {
			return Comparison{}, err
		}
		right, err := exactTotal(profile, value.observed)
		if err != nil {
			return Comparison{}, err
		}
		out.Totals = append(out.Totals, Total{key.field, key.basis, key.unit, left, right})
		if value.expected.Cmp(value.observed) != 0 {
			add("total_difference", normalize.RowKey{}, key.field)
		}
	}
	sort.Slice(out.Totals, func(i, j int) bool {
		a, b := out.Totals[i], out.Totals[j]
		if a.Field != b.Field {
			return a.Field < b.Field
		}
		if a.Basis != b.Basis {
			return a.Basis < b.Basis
		}
		return a.Unit < b.Unit
	})
	sort.Slice(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		if a.Key.Kind != b.Key.Kind {
			return a.Key.Kind < b.Key.Kind
		}
		if a.Key.Document != b.Key.Document {
			return a.Key.Document < b.Key.Document
		}
		if a.Key.Key != b.Key.Key {
			return a.Key.Key < b.Key.Key
		}
		if a.Field != b.Field {
			return a.Field < b.Field
		}
		return a.Code < b.Code
	})
	if len(findings) > 1000 {
		return Comparison{}, ErrLimit
	}
	out.Findings = findings
	encoded, err := json.Marshal(comparisonEnvelope{ComparisonMethod, snapshot, findings, out.Totals})
	if err != nil {
		return Comparison{}, err
	}
	if len(encoded) > 1<<20 {
		return Comparison{}, ErrLimit
	}
	sum := sha256.Sum256(encoded)
	out.Envelope = encoded
	out.Evidence = Evidence{Scope: snapshot.Scope, Run: snapshot.Run.Spec.ID, Digest: hex.EncodeToString(sum[:]), Method: ComparisonMethod, Verdict: Match}
	if len(findings) > 0 {
		out.Evidence.Verdict = Difference
		for _, finding := range findings {
			out.Evidence.Findings = append(out.Evidence.Findings, finding.Code)
		}
	}
	return out, nil
}
