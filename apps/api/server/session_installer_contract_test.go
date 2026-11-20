package server

import (
	"errors"
	"strings"
	"testing"
)

func TestSessionInstallerRegistryCopiesAndNamedSelectionSetters(t *testing.T) {
	all := AllAccessSettingList()
	names, flags := map[string]bool{}, map[string]bool{}
	for _, row := range all {
		if row.Name == "" || row.Flag == "" || row.Unit == "" || names[row.Name] || flags[row.Flag] || row.Default < row.Min || row.Default > row.Max {
			t.Fatal("public installer registry has missing or ambiguous metadata")
		}
		names[row.Name], flags[row.Flag] = true, true
	}
	if len(all) == 0 {
		t.Fatal("public installer registry is empty")
	}
	first := all[0]
	all[0].Name, all[0].Min = "caller-copy", -1
	if fresh := AllAccessSettingList(); fresh[0].Name != first.Name || fresh[0].Min != first.Min {
		t.Fatal("editing the public registry copy changed a later consumer")
	}
	settings := DefaultAccessSettings()
	selection := SessionSelectionSettingList()
	for _, row := range selection {
		if !names[row.Name] {
			t.Fatal("selection installer setting is absent from the public registry")
		}
		switch row.Name {
		case "session_selection_pending":
			row.Set(&settings, 3)
		case "session_selection_max_targets":
			row.Set(&settings, 99)
		case "session_record_references":
			row.Set(&settings, 1024)
		default:
			t.Fatal("unverified selection installer field mapping")
		}
	}
	if settings.SessionSelectionPending != 3 || settings.SessionSelectionTargets != 99 || settings.SessionRecordBudget != 1024 {
		t.Fatal("named public installer setters changed another setting's field")
	}
	if len(selection) != 3 {
		t.Fatal("selection installer fields are missing")
	}
	name, minimum := selection[0].Name, selection[0].Min
	selection[0].Name, selection[0].Min = "selection-caller-copy", -1
	if fresh := SessionSelectionSettingList(); fresh[0].Name != name || fresh[0].Min != minimum {
		t.Fatal("selection installer rows share mutable descriptor storage")
	}
}

func TestSessionInstallerRejectsInsufficientRecordBudgetBeforeStarting(t *testing.T) {
	settings := DefaultAccessSettings()
	settings.SessionSelectionTargets, settings.SessionRecordBudget = 100, 256
	cfg := Config{ServiceCatalog: testServiceCatalog(), Data: t.TempDir() + "/data", Seed: SeedSample, Passwords: fastPasswords(), Access: settings}
	if _, err := New(cfg); !errors.Is(err, ErrConfig) || !strings.Contains(err.Error(), "session_record_references") {
		t.Fatal("individually valid settings escaped the combined record-budget bound")
	}
	cfg.Access.SessionRecordBudget = 301
	s, err := New(cfg)
	if err != nil {
		t.Fatal("exact combined budget boundary was rejected")
	}
	defer s.Close()
	if s.cfg.Access.SessionSelectionTargets != 100 || s.cfg.Access.SessionRecordBudget != 301 {
		t.Fatal("constructor silently replaced the operator's valid boundary settings")
	}
}
