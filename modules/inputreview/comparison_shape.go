package inputreview

import (
	"reflect"

	"github.com/myblueton-sw/hybrid_billing/modules/normalize"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func rowIdentity(key normalize.RowKey) bool {
	return validID(key.Key) && (key.Kind == normalize.EventKeys && key.Document == "" || key.Kind == normalize.LocatorKeys && validID(key.Document))
}

func validateSnapshot(s Snapshot) (decimal.Profile, map[string]normalize.Field, map[string]normalize.Entry, map[string]UnitBinding, error) {
	invalid := func(err error) (decimal.Profile, map[string]normalize.Field, map[string]normalize.Entry, map[string]UnitBinding, error) {
		return decimal.Profile{}, nil, nil, nil, err
	}
	if !validScope(s.Scope) || s.Manifest == 0 || !validDigest(s.Content) || !validDigest(s.Attribution) || !validID(s.Run.Spec.ID) || s.Run.Spec.WorkspaceID != s.Scope.Workspace || s.Run.Spec.Input.Scope != s.Scope.Account || s.Run.Spec.Input.Period != s.Scope.Period || s.Run.Spec.Input.Manifest != s.Manifest || s.Run.Spec.Mapping != s.Mapping.Key || s.Mapping.Key.Provider != s.Scope.Provider || s.Mapping.Key.Target != s.Target.ID || s.Run.Spec.Revision != s.Mapping.Number || s.Mapping.Number == 0 {
		return invalid(ErrInvalid)
	}
	if !validID(s.Target.ID) || !validID(s.Mapping.Key.SourceVersion) || !validID(s.Profile.ID) || !validID(s.Profile.Revision) || s.Profile.ID != s.Run.ProfileID || s.Profile.Revision != s.Run.ProfileRevision {
		return invalid(ErrInvalid)
	}
	for _, size := range []int{s.Profile.InputIntegerDigits, s.Profile.InputScale, s.Profile.ResultIntegerDigits, s.Profile.ResultScale, s.Profile.IntermediateDigits} {
		if size < 1 || size > 80 {
			return invalid(ErrLimit)
		}
	}
	if len(s.Raw) < 1 || len(s.Raw) > 1000 || len(s.Rows) > 1000 || len(s.Target.Fields) < 1 || len(s.Target.Fields) > 20 || len(s.Mapping.Entries) != len(s.Target.Fields) || len(s.Units) > 20 || len(s.Run.Sums) > 20 || len(s.Run.Spec.Expected) > 20 {
		return invalid(ErrLimit)
	}
	for _, sums := range s.Run.Sums {
		if len(sums) > 20 {
			return invalid(ErrLimit)
		}
	}
	for _, sums := range s.Run.Spec.Expected {
		if len(sums) > 20 {
			return invalid(ErrLimit)
		}
	}
	for _, row := range s.Raw {
		if !rowIdentity(row.Key) || len(row.Cells) > 20 {
			return invalid(ErrInvalid)
		}
		for name := range row.Cells {
			if !validID(name) {
				return invalid(ErrInvalid)
			}
		}
	}
	for _, row := range s.Rows {
		if !rowIdentity(row.Key) || len(row.Cells) > 20 || len(row.Lacking) > 20 {
			return invalid(ErrInvalid)
		}
		for name, cell := range row.Cells {
			if !validID(name) || cell.State != normalize.Value && cell.State != normalize.Null && cell.State != normalize.NotProvided && cell.State != normalize.Unmapped {
				return invalid(ErrInvalid)
			}
		}
	}
	if len(s.Mapping.Partial) > 20 || len(s.Mapping.NotApplicable) > 20 || len(s.Mapping.BillingGaps) > 20 || len(s.Run.Unused.Items) > 20 {
		return invalid(ErrLimit)
	}
	if s.Run.Format != nil && len(s.Run.Format.Reasons) > 20 || s.Run.Billing != nil && len(s.Run.Billing.Reasons) > 20 {
		return invalid(ErrLimit)
	}
	remaining := 1 << 20
	if !boundedText(reflect.ValueOf(s), &remaining) {
		return invalid(ErrLimit)
	}
	profile, err := decimal.NewProfile(s.Profile)
	if err != nil {
		return invalid(err)
	}
	fields := make(map[string]normalize.Field)
	for _, field := range s.Target.Fields {
		if !validID(field.Name) {
			return invalid(ErrInvalid)
		}
		if _, exists := fields[field.Name]; exists {
			return invalid(ErrInvalid)
		}
		switch field.Kind {
		case normalize.Cost:
			if field.Cost != normalize.Billed && field.Cost != normalize.Effective && field.Cost != normalize.List && field.Cost != normalize.Contracted {
				return invalid(ErrRejected)
			}
		case normalize.Quantity, normalize.Currency, normalize.Time, normalize.Text:
			if field.Cost != "" {
				return invalid(ErrRejected)
			}
		default:
			return invalid(ErrRejected)
		}
		fields[field.Name] = field
	}
	if fields[s.Target.CostCurrency].Kind != normalize.Currency {
		return invalid(ErrRejected)
	}
	if (s.Target.PeriodStart == "") != (s.Target.PeriodEnd == "") {
		return invalid(ErrRejected)
	}
	if s.Target.PeriodStart != "" && (s.Target.PeriodStart == s.Target.PeriodEnd || fields[s.Target.PeriodStart].Kind != normalize.Time || fields[s.Target.PeriodEnd].Kind != normalize.Time) {
		return invalid(ErrRejected)
	}
	if s.Run.Spec.Window != nil && s.Target.PeriodStart == "" {
		return invalid(ErrRejected)
	}
	entries := make(map[string]normalize.Entry)
	for _, entry := range s.Mapping.Entries {
		field, exists := fields[entry.Field]
		if !exists {
			return invalid(ErrInvalid)
		}
		if _, duplicate := entries[entry.Field]; duplicate {
			return invalid(ErrInvalid)
		}
		if entry.Missing != "" {
			if entry.Missing != normalize.NotProvidedBySource && entry.Missing != normalize.NotSupported && entry.Missing != normalize.NotApplicable {
				return invalid(ErrRejected)
			}
			if entry.Source != "" || entry.SourceKind != "" || entry.SourceCost != "" || entry.Transform != "" || entry.Power != 0 {
				return invalid(ErrRejected)
			}
		} else {
			if !validID(entry.Source) || entry.SourceKind != field.Kind || entry.SourceCost != field.Cost {
				return invalid(ErrRejected)
			}
			switch entry.Transform {
			case normalize.Copy, normalize.Negate:
				if entry.Power != 0 {
					return invalid(ErrRejected)
				}
			case normalize.Scale:
				if entry.Power == 0 || entry.Power < -12 || entry.Power > 12 {
					return invalid(ErrRejected)
				}
			default:
				return invalid(ErrRejected)
			}
			if entry.Transform != normalize.Copy && field.Kind != normalize.Cost && field.Kind != normalize.Quantity {
				return invalid(ErrRejected)
			}
		}
		entries[entry.Field] = entry
	}
	if s.Target.PeriodStart != "" && (entries[s.Target.PeriodStart].Missing != "" || entries[s.Target.PeriodEnd].Missing != "") {
		return invalid(ErrRejected)
	}
	units := make(map[string]UnitBinding)
	for _, binding := range s.Units {
		if fields[binding.Field].Kind != normalize.Quantity || fields[binding.Target].Kind != normalize.Text || !validID(binding.Source) {
			return invalid(ErrRejected)
		}
		entry := entries[binding.Target]
		if entry.Source != binding.Source || entry.Transform != normalize.Copy || entry.Missing != "" {
			return invalid(ErrRejected)
		}
		if _, exists := units[binding.Field]; exists {
			return invalid(ErrInvalid)
		}
		units[binding.Field] = binding
	}
	return profile, fields, entries, units, nil
}
