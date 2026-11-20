package server

import (
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/contracts"
	"github.com/myblueton-sw/hybrid_billing/modules/intake"
	"github.com/myblueton-sw/hybrid_billing/modules/normalize"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
	"github.com/myblueton-sw/hybrid_billing/modules/rating"
)

func normalizedFixture(t *testing.T, transform bool, alter func([]normalize.Row)) (*normalizedCharges, normalizedBinding) {
	t.Helper()
	p, b, err := normalizedFixtureResult(t, transform, alter)
	if err != nil {
		t.Fatal(err)
	}
	return p, b
}

func normalizedFixtureResult(t *testing.T, transform bool, alter func([]normalize.Row)) (*normalizedCharges, normalizedBinding, error) {
	t.Helper()
	p, b, packet := normalizedFixtureSetup(t, transform, alter)
	p.domain.Lock()
	defer p.domain.Unlock()
	_, err := p.assembleSourceLocked(packet)
	return p, b, err
}

func normalizedFixtureSetup(t *testing.T, transform bool, alter func([]normalize.Row)) (*normalizedCharges, normalizedBinding, sourcePacket) {
	t.Helper()
	input := candidateFixture()
	target := normalize.Target{ID: "target", CostCurrency: "currency", PeriodStart: "start", PeriodEnd: "end", Fields: []normalize.Field{
		{Name: "billed", Kind: normalize.Cost, Cost: normalize.Billed, Format: true, Billing: true},
		{Name: "effective", Kind: normalize.Cost, Cost: normalize.Effective, Format: true, Billing: true},
		{Name: "currency", Kind: normalize.Currency, Format: true, Billing: true},
		{Name: "class", Kind: normalize.Text, Format: true},
		{Name: "start", Kind: normalize.Time}, {Name: "end", Kind: normalize.Time}, {Name: "note", Kind: normalize.Text},
		{Name: "quantity", Kind: normalize.Quantity}, {Name: "service", Kind: normalize.Text},
	}}
	limits := normalize.Limits{Targets: 2, Fields: 10, Mappings: 10, Revisions: 10, Runs: 10, Chunks: 10, ChunkRows: 10, Rows: 100, Cells: 10, Text: 128, Findings: 100, Unused: 10, Page: 1}
	orgs, err := organizations.NewRegistry("installation")
	if err != nil {
		t.Fatal(err)
	}
	life := lifecycleTestRegistry(t, []contracts.Event{{ID: "start", Kind: contracts.Start, From: contractDay(2026, 10, 1), Revision: 1}})
	p, err := newNormalizedCharges(orgs, life, &sync.RWMutex{}, limits, input.Profile, target, func() time.Time { return time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatal(err)
	}
	p.domain.Lock()
	defer p.domain.Unlock()
	check := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	_, err = p.orgs.PutWorkspace(organizations.Workspace{ID: "workspace", Profile: organizations.ProfileMSP, State: organizations.Active}, 0)
	check(err)
	for _, id := range []string{"seller", "customer"} {
		_, err = p.orgs.PutParty(organizations.Party{ID: id, WorkspaceID: "workspace", Kind: organizations.LegalEntity, State: organizations.Active}, 0)
		check(err)
	}
	_, err = p.orgs.PutBillingProfile(organizations.BillingProfile{ID: "billing", WorkspaceID: "workspace", IssuerID: "seller", BuyerID: "customer", PayerID: "customer", Path: organizations.ExternalInvoice, Currency: "USD", State: organizations.Active}, 0)
	check(err)
	check(p.orgs.AppendContractRevision(organizations.ContractRevision{ContractID: "contract", Revision: 1, WorkspaceID: "workspace", ProviderRef: "provider", CustomerID: "customer", SellerID: "seller", Path: organizations.Resale,
		Components: []organizations.Component{{ID: "usage", Kind: organizations.ProviderCharge, ProviderPaymentResponsibleID: "seller", SellerBillable: true}}}))
	key := normalize.MappingKey{Provider: "provider", SourceVersion: "source-v1", Target: "target"}
	var entries []normalize.Entry
	for _, f := range target.Fields {
		entries = append(entries, normalize.Entry{Field: f.Name, Source: f.Name, SourceKind: f.Kind, SourceCost: f.Cost, Transform: normalize.Copy})
	}
	billed := "5"
	if transform {
		entries[0].Transform = normalize.Scale
		entries[0].Power = 1
		billed = "50"
	}
	_, err = p.normal.DeclareMapping(key, 0, entries)
	check(err)
	rows := []normalize.Row{
		{Key: normalize.RowKey{Kind: normalize.LocatorKeys, Document: "doc-a", Key: "same"}, Cells: map[string]normalize.Cell{"start": {Text: "2026-10-01T00:00:00Z"}, "end": {Text: "2026-11-01T00:00:00Z"}, "note": {Text: "source"}, "billed": {Text: "2"}, "effective": {Text: "1"}, "currency": {Text: "USD"}, "class": {Text: "compute"}, "quantity": {Text: "2"}, "service": {Text: "vm"}}},
		{Key: normalize.RowKey{Kind: normalize.LocatorKeys, Document: "doc-b", Key: "same"}, Cells: map[string]normalize.Cell{"start": {Text: "2026-10-01T00:00:00Z"}, "end": {Text: "2026-11-01T00:00:00Z"}, "note": {Text: "source"}, "billed": {Text: "3"}, "effective": {Text: "1"}, "currency": {Text: "USD"}, "class": {Text: "compute"}, "quantity": {Text: "3"}, "service": {Text: "vm"}}},
	}
	if alter != nil {
		alter(rows)
	}
	b := normalizedBinding{SourceMode: "synthetic", ID: "synthetic-binding", Version: "1", Provider: "provider", SourceVersion: "source-v1", Scope: "scope", Period: "october", MappingRevision: 1, Contract: "contract", ContractRevision: 1, BillingProfile: "billing", BillingRevision: 1,
		From: "2026-10-01", To: "2026-10-31", Class: "class", Classes: []normalizedClass{{Source: "compute", Class: rating.CSPUsage}}, Quantity: "quantity", Service: "service", Profile: input.Profile, Limits: input.Limits, Policy: input.Policy}
	packet := normalizedTestPacket(normalize.RunSpec{WorkspaceID: "workspace", ID: "normalized", Mapping: key, Revision: 1, Input: normalize.Input{Scope: "scope", Period: "october"}, Window: &normalize.Window{From: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)}, Expected: map[string]map[string]string{"billed": {"USD": billed}, "effective": {"USD": "2"}}}, rows, 1, "first")
	return p, b, packet
}

func normalizedTestPacket(spec normalize.RunSpec, rows []normalize.Row, revision uint64, suffix string) sourcePacket {
	spec.Input.Manifest, spec.Rows, spec.Chunks = 0, 0, 0
	return sourcePacket{Mode: "synthetic", CostField: "billed", CurrencyField: "currency", Run: spec, Artifacts: []sourceArtifact{
		{ID: "source-a-" + suffix, Document: "doc-a", Revision: revision, Kind: intake.LocatorKeys, Sums: map[string]string{"USD": "2"}, Rows: rows[:1]},
		{ID: "source-b-" + suffix, Document: "doc-b", Revision: revision, Kind: intake.LocatorKeys, Sums: map[string]string{"USD": "3"}, Rows: rows[1:]},
	}}
}

func TestNormalizedChargesUseActualProducer(t *testing.T) {
	for _, tc := range []struct {
		name        string
		basis       rating.Basis
		transformed bool
		total       string
	}{{"billed", rating.Billed, false, "5"}, {"effective", rating.Effective, false, "2"}, {"transformed", rating.Billed, true, "50"}} {
		t.Run(tc.name, func(t *testing.T) {
			p, b := normalizedFixture(t, tc.transformed, nil)
			b.Policy.CostBasis = tc.basis
			made, err := p.capture("workspace", "normalized", "candidate", b)
			if err != nil {
				t.Fatal(err)
			}
			if made.Envelope.Result.Total != tc.total || made.Envelope.Input.Customer != "customer" || made.Envelope.Input.Entity != "seller" {
				t.Fatalf("producer result mismatch: %+v", made.Envelope)
			}
			origin := made.Envelope.Input.Normalized
			if len(origin.Rows) != 2 || len(made.Envelope.Input.Lines) != 2 || origin.Rows[0].Line == origin.Rows[1].Line || origin.Rows[0].Key.Key != origin.Rows[1].Key.Key || origin.Manifest != 1 || origin.MappingRevision != 1 {
				t.Fatal("paging or economic identity lost")
			}
			if origin.MappingDigest == "" || origin.TargetDigest == "" || origin.Contract.ContractID != "contract" || origin.BillingProfile.ID != "billing" {
				t.Fatal("producer provenance missing")
			}
		})
	}
}

func TestNormalizedChargesQuantity(t *testing.T) {
	p, b := normalizedFixture(t, false, nil)
	b.Policy.Rules = []rating.Rule{{ID: "unit", Level: rating.Default, Type: rating.UnitPrice, Price: rating.Price{Amount: "4", Per: 1}}}
	made, err := p.capture("workspace", "normalized", "candidate", b)
	if err != nil || made.Envelope.Result.Total != "20" {
		t.Fatal("quantity result", made.Envelope.Result.Total, err)
	}
}

func TestNormalizedChargesRefuseIncompleteBindings(t *testing.T) {
	for name, change := range map[string]func(*normalizedBinding){
		"scope":            func(b *normalizedBinding) { b.Scope = "foreign" },
		"provider":         func(b *normalizedBinding) { b.Provider = "foreign" },
		"contract":         func(b *normalizedBinding) { b.Contract = "absent" },
		"profile revision": func(b *normalizedBinding) { b.BillingRevision = 2 },
		"mapping revision": func(b *normalizedBinding) { b.MappingRevision = 2 },
		"class":            func(b *normalizedBinding) { b.Classes = nil },
		"unknown class":    func(b *normalizedBinding) { b.Classes[0].Source = "unknown" },
		"duplicate class":  func(b *normalizedBinding) { b.Classes = append(b.Classes, b.Classes[0]) },
		"quantity kind":    func(b *normalizedBinding) { b.Quantity = "service" },
		"currency":         func(b *normalizedBinding) { b.Policy.Currency = "EUR" },
		"missing policy":   func(b *normalizedBinding) { b.Policy = rating.Policy{} },
		"period":           func(b *normalizedBinding) { b.From = "2026-10-02" },
		"missing sku":      func(b *normalizedBinding) { b.Policy.Rules[0].Level = rating.SKU; b.Policy.Rules[0].Key = "vm" },
	} {
		t.Run(name, func(t *testing.T) {
			p, b := normalizedFixture(t, false, nil)
			change(&b)
			if _, err := p.capture("workspace", "normalized", "candidate", b); err == nil {
				t.Fatal("incomplete binding accepted")
			}
			if len(p.candidates.records) != 0 {
				t.Fatal("refusal published candidate")
			}
		})
	}
}

func TestNormalizedChargesRejectMissingAndForeign(t *testing.T) {
	for name, change := range map[string]func([]normalize.Row){
		"missing quantity": func(rows []normalize.Row) { delete(rows[0].Cells, "quantity") },
		"null quantity":    func(rows []normalize.Row) { rows[0].Cells["quantity"] = normalize.Cell{Null: true} },
		"missing service":  func(rows []normalize.Row) { delete(rows[0].Cells, "service") },
		"mixed currency":   func(rows []normalize.Row) { rows[0].Cells["currency"] = normalize.Cell{Text: "EUR"} },
	} {
		t.Run(name, func(t *testing.T) {
			p, b, sourceErr := normalizedFixtureResult(t, false, change)
			if name == "mixed currency" && sourceErr == nil || name != "mixed currency" && sourceErr != nil {
				t.Fatal("unexpected source assembly result", sourceErr)
			}
			if _, err := p.capture("workspace", "normalized", "candidate", b); err == nil {
				t.Fatal("incomplete rows accepted")
			}
			if len(p.candidates.records) != 0 {
				t.Fatal("refusal published candidate")
			}
		})
	}
	p, b := normalizedFixture(t, false, nil)
	if _, err := p.capture("foreign", "normalized", "candidate", b); err == nil {
		t.Fatal("foreign run accepted")
	}
}

func TestNormalizedChargesRejectUnselectedInvalidText(t *testing.T) {
	p, b, sourceErr := normalizedFixtureResult(t, false, func(rows []normalize.Row) { rows[0].Cells["note"] = normalize.Cell{Text: string([]byte{255})} })
	if sourceErr == nil {
		t.Fatal("invalid source UTF8 accepted")
	}
	if _, err := p.capture("workspace", "normalized", "candidate", b); err == nil {
		t.Fatal("invalid unselected text was hashed")
	}
	if len(p.candidates.records) != 0 {
		t.Fatal("invalid text published candidate")
	}
}

func TestNormalizedChargesCurrentOrganizations(t *testing.T) {
	for _, kind := range []string{"workspace", "profile", "party", "customer mismatch", "issuer mismatch", "path mismatch"} {
		t.Run(kind, func(t *testing.T) {
			p, b := normalizedFixture(t, false, nil)
			p.domain.Lock()
			var err error
			switch kind {
			case "workspace":
				_, err = p.orgs.PutWorkspace(organizations.Workspace{ID: "workspace", Profile: organizations.ProfileMSP, State: organizations.Suspended}, 1)
			case "profile":
				profile, revision, e := p.orgs.BillingProfile("workspace", "billing")
				if e != nil {
					t.Fatal(e)
				}
				profile.State = organizations.Suspended
				b.BillingRevision, err = p.orgs.PutBillingProfile(profile, revision)
			case "party":
				_, err = p.orgs.PutParty(organizations.Party{ID: "seller", WorkspaceID: "workspace", Kind: organizations.LegalEntity, State: organizations.Suspended}, 1)
			default:
				_, err = p.orgs.PutParty(organizations.Party{ID: "other", WorkspaceID: "workspace", Kind: organizations.LegalEntity, State: organizations.Active}, 0)
				if err != nil {
					t.Fatal(err)
				}
				contract, e := p.orgs.ContractRevision("workspace", "contract", 1)
				if e != nil {
					t.Fatal(e)
				}
				contract.ContractID = "other-contract"
				switch kind {
				case "customer mismatch":
					contract.CustomerID = "other"
				case "issuer mismatch":
					contract.SellerID = "other"
					contract.Components[0].ProviderPaymentResponsibleID = "other"
				case "path mismatch":
					contract.Path = organizations.Direct
					contract.Components = []organizations.Component{{ID: "service", Kind: organizations.SellerService, SellerBillable: true}}
				}
				err = p.orgs.AppendContractRevision(contract)
				b.Contract = contract.ContractID
			}
			p.domain.Unlock()
			if err != nil {
				t.Fatal(err)
			}
			if _, err := p.capture("workspace", "normalized", "candidate", b); err == nil {
				t.Fatal("invalid attribution accepted")
			}
			if len(p.candidates.records) != 0 {
				t.Fatal("invalid attribution published candidate")
			}
		})
	}
}

func TestNormalizedChargesActiveParent(t *testing.T) {
	p, _ := normalizedFixture(t, false, nil)
	p.domain.Lock()
	defer p.domain.Unlock()
	_, err := p.orgs.PutParty(organizations.Party{ID: "unit", WorkspaceID: "workspace", Kind: organizations.InternalUnit, ParentID: "customer", State: organizations.Active}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.activeParty("workspace", "unit"); err != nil {
		t.Fatal(err)
	}
	_, err = p.orgs.PutParty(organizations.Party{ID: "customer", WorkspaceID: "workspace", Kind: organizations.LegalEntity, State: organizations.Suspended}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.activeParty("workspace", "unit"); err == nil {
		t.Fatal("suspended parent accepted")
	}
}

func TestNormalizedChargesRefusalDoesNotReserve(t *testing.T) {
	p, b := normalizedFixture(t, false, nil)
	bad := b
	bad.Quantity = ""
	bad.Policy.Rules = []rating.Rule{{ID: "quantity", Level: rating.Default, Type: rating.UnitPrice, Price: rating.Price{Amount: "4", Per: 1}}}
	if _, err := p.capture("workspace", "normalized", "candidate", bad); err == nil {
		t.Fatal("missing quantity binding accepted")
	}
	if _, err := p.capture("workspace", "normalized", "candidate", b); err != nil {
		t.Fatal("refusal reserved candidate", err)
	}
}

func TestNormalizedChargesMappingIdentity(t *testing.T) {
	for _, versionChange := range []bool{false, true} {
		p, b := normalizedFixture(t, false, nil)
		b.Policy.CostBasis = rating.Effective
		before, err := p.capture("workspace", "normalized", "first", b)
		if err != nil {
			t.Fatal(err)
		}
		p.domain.Lock()
		run, err := p.normal.Run("workspace", "normalized")
		if err != nil {
			t.Fatal(err)
		}
		revision, err := p.normal.Revision(run.Spec.Mapping, 1)
		if err != nil {
			t.Fatal(err)
		}
		var raw []normalize.Row
		for after := 0; after < 2; after++ {
			page, err := p.normal.Rows("workspace", "normalized", after, 1)
			if err != nil {
				t.Fatal(err)
			}
			row := normalize.Row{Key: page.Rows[0].Key, Cells: make(map[string]normalize.Cell)}
			for field, cell := range page.Rows[0].Cells {
				row.Cells[field] = normalize.Cell{Text: cell.Text}
			}
			raw = append(raw, row)
		}
		spec := run.Spec
		spec.ID = "second-run"
		expected := uint64(1)
		if versionChange {
			spec.Mapping.SourceVersion = "source-v2"
			expected = 0
		} else {
			for i := range revision.Entries {
				if revision.Entries[i].Field == "billed" {
					revision.Entries[i].Transform = normalize.Scale
					revision.Entries[i].Power = 1
				}
			}
			spec.Expected["billed"]["USD"] = "50"
		}
		spec.Revision, err = p.normal.DeclareMapping(spec.Mapping, expected, revision.Entries)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := p.assembleSourceLocked(normalizedTestPacket(spec, raw, 2, "second")); err != nil {
			t.Fatal(err)
		}
		p.domain.Unlock()
		if _, err := p.capture("workspace", spec.ID, "second", b); err == nil {
			t.Fatal("different mapping accepted under old binding")
		}
		b.MappingRevision = spec.Revision
		b.SourceVersion = spec.Mapping.SourceVersion
		after, err := p.capture("workspace", spec.ID, "second", b)
		if err != nil {
			t.Fatal(err)
		}
		if after.Envelope.Result.Total != "2" || before.Digest == after.Digest || before.Envelope.Input.Normalized.MappingDigest == after.Envelope.Input.Normalized.MappingDigest {
			t.Fatal("changed mapping provenance lost")
		}
		if before.Envelope.Input.Lines[0].ID != after.Envelope.Input.Lines[0].ID {
			t.Fatal("mapping revision changed economic identity")
		}
	}
}
