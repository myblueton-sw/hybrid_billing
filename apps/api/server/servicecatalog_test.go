package server

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/servicecatalog"
)

func configuredServiceCatalog() []byte {
	return []byte(`{"audiences":[{"workspace":"workspace","id":"usage-api","resource":"https://usage.invalid/api","scopes":[{"id":"read","action":"view","kind":"` + ReaderKind + `","fields":["` + ReaderField + `"],"customer":{"any":true},"legal_entity":{"any":true},"contract":{"any":true},"period":{"any":true}}]}]}`)
}

func TestServiceCatalogConstructorUsesActualGateAndFreezesConfiguration(t *testing.T) {
	raw := configuredServiceCatalog()
	before := append([]byte{}, raw...)
	data := filepath.Join(t.TempDir(), "not-created")
	s, err := New(Config{Data: data, Seed: SeedNone, ServiceCatalog: raw})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	if !bytes.Equal(raw, before) {
		t.Fatal("server overwrote caller catalog bytes")
	}
	if s.cfg.ServiceCatalog != nil {
		t.Fatal("server retained raw catalog configuration")
	}
	if _, err := os.Stat(data); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("constructor initialized installation data", err)
	}
	if len(s.admin.Grants()) != 0 {
		t.Fatal("catalog construction created grants")
	}
	entry, err := s.serviceCatalog.Scope("workspace", "usage-api", "read")
	if err != nil || entry.Terms.Kind != ReaderKind || entry.Terms.Fields[0] != ReaderField {
		t.Fatal("configured catalog was not connected to actual gate vocabulary", entry, err)
	}
	version := s.serviceCatalog.Version()
	for i := range raw {
		raw[i] = 'x'
	}
	entry, err = s.serviceCatalog.Scope("workspace", "usage-api", "read")
	if err != nil || entry.ID != "read" || s.serviceCatalog.Version() != version {
		t.Fatal("caller mutation changed running catalog", err)
	}
	other, err := New(Config{Data: filepath.Join(t.TempDir(), "unused"), Seed: SeedNone, ServiceCatalog: bytes.Replace(before, []byte("https://usage.invalid/api"), []byte("urn:example:changed"), 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	if version == other.serviceCatalog.Version() {
		t.Fatal("changed configuration kept catalog fingerprint")
	}
}

func TestServiceCatalogConstructorRejectsInvalidInputBeforeDataInitialization(t *testing.T) {
	valid := configuredServiceCatalog()
	for name, raw := range map[string][]byte{"non-nil empty": {}, "malformed": []byte("{"), "null": []byte("null"), "unknown vocabulary": bytes.Replace(valid, []byte(ReaderKind), []byte("unknown-kind"), 1), "oversize": bytes.Repeat([]byte(" "), servicecatalog.MaxBytes+1)} {
		t.Run(name, func(t *testing.T) {
			data := filepath.Join(t.TempDir(), "absent")
			s, err := New(Config{Data: data, Seed: SeedNone, ServiceCatalog: raw})
			if s != nil || !errors.Is(err, ErrStart) {
				t.Fatal("invalid configured catalog accepted", name, err)
			}
			if _, err := os.Stat(data); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("invalid catalog initialized data", name, err)
			}
		})
	}
}

func TestServiceCatalogOmissionAndExplicitEmptyAreKnownEmpty(t *testing.T) {
	var version string
	for _, raw := range [][]byte{nil, []byte(`{"audiences":[]}`)} {
		s, err := New(Config{Data: filepath.Join(t.TempDir(), "unused"), Seed: SeedNone, ServiceCatalog: raw})
		if err != nil {
			t.Fatal(err)
		}
		rows, next, err := s.serviceCatalog.AudiencePage("workspace", "", 1)
		if err != nil || rows == nil || len(rows) != 0 || next != "" {
			t.Fatal("omission invented catalog defaults", rows, next, err)
		}
		if version != "" && s.serviceCatalog.Version() != version {
			t.Fatal("equivalent empty catalogs differ")
		}
		version = s.serviceCatalog.Version()
		s.Close()
	}
	if strings.TrimSpace(version) == "" {
		t.Fatal("known empty catalog has no version")
	}
}
