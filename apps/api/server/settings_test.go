package server

import (
	"errors"
	"strings"
	"testing"
)

func TestAccessSettingsValidation(t *testing.T) {
	for name, change := range map[string]func(*AccessSettings){
		"access_request_hours":         func(s *AccessSettings) { s.RequestHours = 721 },
		"access_pending_per_proposer":  func(s *AccessSettings) { s.PendingPerProposer = 0 },
		"access_pending_per_workspace": func(s *AccessSettings) { s.PendingPerProposer, s.PendingPerWorkspace = 30, 20 },
		"access_ended_kept":            func(s *AccessSettings) { s.EndedKept = 49 },
		"access_revocations_kept":      func(s *AccessSettings) { s.RevocationsKept = 10001 },
		"access_ended_shown":           func(s *AccessSettings) { s.EndedKept, s.EndedShown = 60, 61 },
		"decision_ledger_max_mib":      func(s *AccessSettings) { s.LedgerMaxMiB = 4097 },
	} {
		s := DefaultAccessSettings()
		change(&s)
		_, err := New(Config{Data: t.TempDir() + "/d", Seed: SeedSample, Access: s})
		if !errors.Is(err, ErrConfig) || !strings.HasSuffix(err.Error(), name) {
			t.Errorf("%s: %v", name, err)
		}
	}
	zero := AccessSettings{AcceptNewLedger: true, AcceptTrustReset: true, CarryTrustFrom: "x", TrustGeneration: 3}.withDefaults()
	if zero.RequestHours != 168 || zero.LedgerMaxMiB != 256 || !zero.AcceptNewLedger || zero.validate() != nil || !zero.AcceptTrustReset || zero.CarryTrustFrom != "x" ||
		zero.TrustGeneration != 3 {
		t.Errorf("defaults %+v", zero)
	}
	list := AccessSettingList()
	list[0].Name = "changed"
	if AccessSettingList()[0].Name != "access_request_hours" {
		t.Error("the list aliases the settings")
	}
}

func TestSettingRows(t *testing.T) {
	f := setupWith(t, func(c *Config) {
		c.Access = DefaultAccessSettings()
		c.Access.RequestHours = 24
		c.Access.Given["access_request_hours"] = true
	})
	rows := f.accessRead(f.as(AdminLogin), "").body["settings"].([]any)
	first, last := rows[0].(map[string]any), rows[6].(map[string]any)
	if first["name"] != "access_request_hours" || first["value"] != 24.0 || first["default"] != 168.0 || first["source"] != SourceStartFlag || first["unit"] != "hours" ||
		first["min"] != 1.0 || first["max"] != 720.0 {
		t.Errorf("first %+v", first)
	}
	if last["name"] != "decision_ledger_max_mib" || last["source"] != SourceDefault || last["value"] != 256.0 {
		t.Errorf("last %+v", last)
	}
}
