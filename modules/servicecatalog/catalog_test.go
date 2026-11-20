package servicecatalog

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/auth"
)

func validator(t testing.TB) *auth.Registry {
	t.Helper()
	r, err := auth.NewRegistry("synthetic-installation", auth.Catalog{Kinds: []string{"usage_statement", "allocation_pool", auth.InstallationKind}, Fields: map[string]auth.FieldClass{"quantity": auth.General, "service": auth.General, "unit_cost": auth.Cost}}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func sample() Config {
	return Config{Audiences: []AudienceSpec{{Workspace: "w1", ID: "usage-api", Resource: "https://usage.invalid/api", Scopes: []ScopeSpec{{ID: "read", Action: auth.View, Kind: "usage_statement", Fields: []string{"quantity"}, Customer: SelectorSpec{Any: true}, LegalEntity: SelectorSpec{Any: true}, Contract: SelectorSpec{Any: true}, Period: PeriodSpec{Any: true}}}}}}
}

func TestCatalogExplicitRestrictionsAreNotEffectiveGrants(t *testing.T) {
	v := validator(t)
	before := v.Generation()
	cfg := sample()
	cfg.Audiences[0].Scopes[0].Customer = SelectorSpec{ID: "all"}
	cfg.Audiences[0].Scopes[0].Period = PeriodSpec{From: "2026-01", Until: "2026-03"}
	catalog, err := New(cfg, v)
	if err != nil {
		t.Fatal(err)
	}
	got, err := catalog.Scope("w1", "usage-api", "read")
	if err != nil || got.Terms.Scope.Customer.Any() || got.Terms.Scope.Customer.ID() != "all" || got.Terms.Scope.WorkspaceID != "w1" || got.Terms.Period.Any() || got.Terms.Period.From() >= got.Terms.Period.To() {
		t.Fatal("explicit restriction changed meaning", got, err)
	}
	if len(v.Grants()) != 0 || v.Generation() != before {
		t.Fatal("catalog construction granted authority")
	}
	for _, lookup := range []struct{ workspace, audience, scope string }{{"w2", "usage-api", "read"}, {"w1", "missing", "read"}, {"w1", "usage-api", "missing"}} {
		if _, err := catalog.Scope(lookup.workspace, lookup.audience, lookup.scope); !errors.Is(err, ErrNotFound) {
			t.Fatal("foreign or missing scope disclosed", lookup, err)
		}
	}
	empty, err := New(Config{}, v)
	if err != nil || empty.Version() == "" {
		t.Fatal("empty catalog did not have an explicit fingerprint", err)
	}
	rows, next, err := empty.AudiencePage("w1", "", 1)
	if err != nil || rows == nil || len(rows) != 0 || next != "" {
		t.Fatal("empty catalog invented selectable defaults", rows, next, err)
	}
}

func TestCatalogRejectsIncompleteOrForbiddenDeclarations(t *testing.T) {
	v := validator(t)
	for name, change := range map[string]func(*Config){
		"missing selector": func(c *Config) { c.Audiences[0].Scopes[0].Customer = SelectorSpec{} },
		"mixed selector":   func(c *Config) { c.Audiences[0].Scopes[0].Customer = SelectorSpec{Any: true, ID: "customer"} },
		"missing period":   func(c *Config) { c.Audiences[0].Scopes[0].Period = PeriodSpec{} },
		"mixed period": func(c *Config) {
			c.Audiences[0].Scopes[0].Period = PeriodSpec{Any: true, From: "2026-01", Until: "2026-03"}
		},
		"empty period":       func(c *Config) { c.Audiences[0].Scopes[0].Period = PeriodSpec{From: "2026-01", Until: "2026-01"} },
		"noncanonical month": func(c *Config) { c.Audiences[0].Scopes[0].Period = PeriodSpec{From: "2026-1", Until: "2026-03"} },
		"invalid month":      func(c *Config) { c.Audiences[0].Scopes[0].Period = PeriodSpec{From: "2026-13", Until: "2027-03"} },
		"human approval":     func(c *Config) { c.Audiences[0].Scopes[0].Action = auth.Approve; c.Audiences[0].Scopes[0].Fields = nil },
		"installation right": func(c *Config) {
			c.Audiences[0].Scopes[0].Action = auth.AdministerInstallation
			c.Audiences[0].Scopes[0].Kind = auth.InstallationKind
			c.Audiences[0].Scopes[0].Fields = nil
		},
		"unknown kind":       func(c *Config) { c.Audiences[0].Scopes[0].Kind = "missing" },
		"duplicate fields":   func(c *Config) { c.Audiences[0].Scopes[0].Fields = []string{"quantity", "quantity"} },
		"duplicate scope":    func(c *Config) { c.Audiences[0].Scopes = append(c.Audiences[0].Scopes, c.Audiences[0].Scopes[0]) },
		"duplicate audience": func(c *Config) { c.Audiences = append(c.Audiences, c.Audiences[0]) },
		"invalid workspace":  func(c *Config) { c.Audiences[0].Workspace = "bad workspace" },
		"invalid scope id":   func(c *Config) { c.Audiences[0].Scopes[0].ID = "bad id" },
	} {
		t.Run(name, func(t *testing.T) {
			c := sample()
			change(&c)
			if _, err := New(c, v); !errors.Is(err, ErrInvalid) {
				t.Fatal("invalid declaration accepted", name, err)
			}
		})
	}
	if _, err := New(sample(), nil); err == nil {
		t.Fatal("absent validator accepted")
	}
	var absent *auth.Registry
	if _, err := New(sample(), absent); err == nil {
		t.Fatal("typed nil validator admitted scopes")
	}
	v.SetAvailable(false)
	if _, err := New(sample(), v); err == nil {
		t.Fatal("unavailable validator admitted scopes")
	}
}

func TestCatalogFreezesInputsOutputsAndNormalizesOrder(t *testing.T) {
	v := validator(t)
	cfg := sample()
	cfg.Audiences[0].Scopes[0].Fields = []string{"service", "quantity"}
	other := sample().Audiences[0]
	other.ID = "another"
	cfg.Audiences = append(cfg.Audiences, other)
	before := []string{cfg.Audiences[0].ID, cfg.Audiences[0].Scopes[0].Fields[0]}
	catalog, err := New(cfg, v)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, []string{cfg.Audiences[0].ID, cfg.Audiences[0].Scopes[0].Fields[0]}) {
		t.Fatal("constructor reordered caller configuration")
	}
	reordered := sample()
	reordered.Audiences[0].Scopes[0].Fields = []string{"quantity", "service"}
	reordered.Audiences = append([]AudienceSpec{other}, reordered.Audiences...)
	equivalent, err := New(reordered, v)
	if err != nil || equivalent.Version() != catalog.Version() {
		t.Fatal("equivalent catalog order changed fingerprint", err)
	}
	cfg.Audiences[0].Resource = "https://changed.invalid"
	cfg.Audiences[0].Scopes[0].Fields[0] = "unit_cost"
	row, err := catalog.Audience("w1", "usage-api")
	if err != nil || row.Resource != "https://usage.invalid/api" {
		t.Fatal("input mutation changed catalog", row, err)
	}
	scope, err := catalog.Scope("w1", "usage-api", "read")
	if err != nil {
		t.Fatal(err)
	}
	scope.Terms.Fields[0] = "unit_cost"
	again, _ := catalog.Scope("w1", "usage-api", "read")
	if reflect.DeepEqual(scope.Terms.Fields, again.Terms.Fields) {
		t.Fatal("scope output aliases catalog")
	}
	rows, _, _ := catalog.ScopePage("w1", "usage-api", "", 1)
	rows[0].Terms.Fields[0] = "unit_cost"
	again, _ = catalog.Scope("w1", "usage-api", "read")
	if again.Terms.Fields[0] != "quantity" {
		t.Fatal("page output aliases catalog")
	}
}

func TestCatalogFingerprintCoversEveryDeclarationMeaning(t *testing.T) {
	v := validator(t)
	baseline, err := New(sample(), v)
	if err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(*Config){
		"workspace": func(c *Config) { c.Audiences[0].Workspace = "w2" }, "audience": func(c *Config) { c.Audiences[0].ID = "another" }, "recipient": func(c *Config) { c.Audiences[0].Resource = "urn:example:another" }, "scope": func(c *Config) { c.Audiences[0].Scopes[0].ID = "another" }, "action": func(c *Config) { c.Audiences[0].Scopes[0].Action = auth.Download }, "kind": func(c *Config) { c.Audiences[0].Scopes[0].Kind = "allocation_pool" }, "fields": func(c *Config) { c.Audiences[0].Scopes[0].Fields = []string{"service"} }, "customer": func(c *Config) { c.Audiences[0].Scopes[0].Customer = SelectorSpec{ID: "all"} }, "legal entity": func(c *Config) { c.Audiences[0].Scopes[0].LegalEntity = SelectorSpec{ID: "entity"} }, "contract": func(c *Config) { c.Audiences[0].Scopes[0].Contract = SelectorSpec{ID: "contract"} }, "period": func(c *Config) { c.Audiences[0].Scopes[0].Period = PeriodSpec{From: "2026-01", Until: "2026-03"} },
	} {
		t.Run(name, func(t *testing.T) {
			cfg := sample()
			change(&cfg)
			changed, err := New(cfg, v)
			if err != nil || changed.Version() == baseline.Version() {
				t.Fatal("permission-bearing catalog change kept fingerprint", name, err)
			}
		})
	}
	cfg := sample()
	cfg.Audiences[0].Scopes[0].Period = PeriodSpec{From: "2026-01", Until: "2026-03"}
	a, _ := New(cfg, v)
	cfg.Audiences[0].Scopes[0].Period.Until = "2026-04"
	b, _ := New(cfg, v)
	if a.Version() == b.Version() {
		t.Fatal("exclusive period end omitted from fingerprint")
	}
}

func TestCatalogPagesAndScopesAreIndependentlyBounded(t *testing.T) {
	v := validator(t)
	cfg := sample()
	for _, id := range []string{"z", "a", "m"} {
		row := sample().Audiences[0]
		row.ID = id
		cfg.Audiences = append(cfg.Audiences, row)
	}
	foreign := sample().Audiences[0]
	foreign.Workspace = "w2"
	cfg.Audiences = append(cfg.Audiences, foreign)
	for _, id := range []string{"z", "a", "m"} {
		entry := sample().Audiences[0].Scopes[0]
		entry.ID = id
		cfg.Audiences[0].Scopes = append(cfg.Audiences[0].Scopes, entry)
	}
	catalog, err := New(cfg, v)
	if err != nil {
		t.Fatal(err)
	}
	rows, next, err := catalog.AudiencePage("w1", "", 2)
	if err != nil || len(rows) != 2 || rows[0].ID != "a" || rows[1].ID != "m" || next != "m" {
		t.Fatal("bounded audience page", rows, next, err)
	}
	rows, next, err = catalog.AudiencePage("w1", next, 2)
	if err != nil || len(rows) != 2 || rows[0].ID != "usage-api" || rows[1].ID != "z" || next != "" {
		t.Fatal("audience continuation lost records", rows, next, err)
	}
	scopes, next, err := catalog.ScopePage("w1", "usage-api", "b", 2)
	if err != nil || len(scopes) != 2 || scopes[0].ID != "m" || scopes[1].ID != "read" || next != "read" {
		t.Fatal("absent scope boundary", scopes, next, err)
	}
	scopes, next, err = catalog.ScopePage("w1", "usage-api", next, 2)
	if err != nil || len(scopes) != 1 || scopes[0].ID != "z" || next != "" {
		t.Fatal("scope continuation lost records", scopes, next, err)
	}
	if _, err := catalog.Scope("w2", "usage-api", "a"); !errors.Is(err, ErrNotFound) {
		t.Fatal("scope leaked from another workspace's identically named audience", err)
	}
	row, _ := catalog.Audience("w2", "usage-api")
	if row.ScopeCount != 1 {
		t.Fatal("scope counts crossed audience workspace", row)
	}
	cfg = sample()
	cfg.Audiences[0].Scopes = nil
	emptyAudience, err := New(cfg, v)
	if err != nil {
		t.Fatal(err)
	}
	rows, _, _ = emptyAudience.AudiencePage("w1", "", 1)
	if rows[0].ScopeCount != 0 {
		t.Fatal("scope-less audience invented permissions")
	}
	for _, limit := range []int{0, 1001} {
		if _, _, err := catalog.AudiencePage("w1", "", limit); !errors.Is(err, ErrInvalid) {
			t.Fatal("audience bound ignored", limit, err)
		}
		if _, _, err := catalog.ScopePage("w1", "usage-api", "", limit); !errors.Is(err, ErrInvalid) {
			t.Fatal("scope bound ignored", limit, err)
		}
	}
	if _, _, err := catalog.ScopePage("w1", "missing", "", 1); !errors.Is(err, ErrNotFound) {
		t.Fatal("missing audience invented empty scopes", err)
	}
	if _, _, err := catalog.AudiencePage("w1", "bad id", 1); !errors.Is(err, ErrInvalid) {
		t.Fatal("invalid continuation admitted", err)
	}
	var absent *Catalog
	if absent.Version() != "" {
		t.Fatal("missing catalog has a fingerprint")
	}
	if _, _, err := absent.AudiencePage("w1", "", 1); !errors.Is(err, ErrUnavailable) {
		t.Fatal("missing catalog presented as empty", err)
	}
}

func TestCatalogZeroHandleDoesNotMasqueradeAsConfiguredEmpty(t *testing.T) {
	unbuilt := &Catalog{}
	if _, _, err := unbuilt.AudiencePage("w1", "", 1); !errors.Is(err, ErrUnavailable) {
		t.Fatal("unbuilt catalog reported a valid empty audience page", err)
	}
	if _, err := unbuilt.Audience("w1", "usage-api"); !errors.Is(err, ErrUnavailable) {
		t.Fatal("unbuilt catalog reported entry absence", err)
	}
	if _, _, err := unbuilt.ScopePage("w1", "usage-api", "", 1); !errors.Is(err, ErrUnavailable) {
		t.Fatal("unbuilt catalog reported scope absence", err)
	}
}
