package server

import (
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/normalize"
	"github.com/myblueton-sw/hybrid_billing/modules/rating"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

const candidateKind = "charge_candidate"
const candidateField = "charge_evidence"

func candidateProfile() decimal.ProfileSpec {
	return decimal.ProfileSpec{ID: "synthetic-charge", Revision: "1", InputIntegerDigits: 12, InputScale: 4, ResultIntegerDigits: 14, ResultScale: 4, IntermediateDigits: 60}
}

func (s *Server) newChargeProducer() (*normalizedCharges, error) {
	profile, err := decimal.NewProfile(candidateProfile())
	if err != nil {
		return nil, err
	}
	target := normalize.Target{ID: "synthetic-charge-target", CostCurrency: "currency", PeriodStart: "start", PeriodEnd: "end", Fields: []normalize.Field{
		{Name: "billed", Kind: normalize.Cost, Cost: normalize.Billed, Format: true, Billing: true},
		{Name: "currency", Kind: normalize.Currency, Format: true, Billing: true},
		{Name: "start", Kind: normalize.Time, Format: true}, {Name: "end", Kind: normalize.Time, Format: true},
		{Name: "class", Kind: normalize.Text, Format: true}, {Name: "service", Kind: normalize.Text, Format: true},
	}}
	limits := normalize.Limits{Targets: 10, Fields: 20, Mappings: 20, Revisions: 20, Runs: 1000, Chunks: 100, ChunkRows: 1000, Rows: 100000, Cells: 20, Text: 128, Findings: 1000, Unused: 20, Page: 100}
	return newNormalizedCharges(s.orgs, &s.domain, limits, profile, target, s.clock)
}

func (s *Server) seedChargeCandidates(now time.Time) error {
	p := s.chargeProducer
	ws := s.Workspace()
	contract, err := s.orgs.ContractRevision(ws, "sample-contract-01", 2)
	if err != nil {
		return err
	}
	key := normalize.MappingKey{Provider: contract.ProviderRef, SourceVersion: "synthetic-source-v1", Target: p.target.ID}
	var entries []normalize.Entry
	for _, field := range p.target.Fields {
		entries = append(entries, normalize.Entry{Field: field.Name, Source: field.Name, SourceKind: field.Kind, SourceCost: field.Cost, Transform: normalize.Copy})
	}
	if _, err := p.normal.DeclareMapping(key, 0, entries); err != nil {
		return err
	}
	from := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	run := "sample-normalized-charge"
	if _, err := p.normal.Open(normalize.RunSpec{WorkspaceID: ws, ID: run, Mapping: key, Revision: 1, Input: normalize.Input{Scope: "sample-acct-01", Period: "2026-10", Manifest: 1}, Chunks: 1, Rows: 2,
		Window: &normalize.Window{From: from, To: to}, Expected: map[string]map[string]string{"billed": {"USD": "15"}}}); err != nil {
		return err
	}
	var rows []normalize.Row
	for i, amount := range []string{"10.25", "4.75"} {
		id := []string{"sample-line-a", "sample-line-b"}[i]
		rows = append(rows, normalize.Row{Key: normalize.RowKey{Kind: normalize.EventKeys, Key: id}, Cells: map[string]normalize.Cell{
			"billed": {Text: amount}, "currency": {Text: "USD"}, "start": {Text: from.Format(time.RFC3339)}, "end": {Text: to.Format(time.RFC3339)}, "class": {Text: "compute"}, "service": {Text: "virtual-machine"},
		}})
	}
	if _, err := p.normal.Check(ws, run, 1, rows); err != nil {
		return err
	}
	if _, _, err := p.normal.Close(ws, run); err != nil {
		return err
	}
	binding := normalizedBinding{SourceMode: "synthetic", ID: "sample-charge-binding", Version: "1", Provider: key.Provider, SourceVersion: key.SourceVersion, Scope: "sample-acct-01", Period: "2026-10", MappingRevision: 1,
		Contract: contract.ContractID, ContractRevision: contract.Revision, BillingProfile: "sample-profile-north", BillingRevision: 1, From: "2026-10-01", To: "2026-10-31", Class: "class", Service: "service",
		Classes: []normalizedClass{{Source: "compute", Class: rating.CSPUsage}}, Profile: candidateProfile(), Limits: rating.Limits{Rules: 20, Tiers: 20, Discounts: 20, Lines: 1000, Intermediate: 60},
		Policy: rating.Policy{ID: "sample-charge-policy", Version: "1", Currency: "USD", CostBasis: rating.Billed, Quantum: "0.01", Mode: "half_even", Path: rating.Resale, Eligible: []rating.Class{rating.CSPUsage},
			Rules: []rating.Rule{{ID: "sample-pass-through", Level: rating.Default, Type: rating.PassThrough}}}}
	if _, err := p.captureLocked(ws, run, "sample-charge-01", binding); err != nil {
		return err
	}
	terms := gate.Terms{Action: gate.ViewCost, Kind: candidateKind, Fields: []string{candidateField}, Period: gate.AnyPeriod(), Scope: gate.Scope{WorkspaceID: ws, Customer: gate.All(), LegalEntity: gate.All(), Contract: gate.All()}}
	validity := gate.Validity{NotBefore: now, NotAfter: now.Add(readerValidity)}
	grants := []gate.Grant{{ID: "candidate-console", ClientID: ConsoleClient, Terms: terms, Validity: validity}}
	for _, user := range []string{SampleAdmin, SampleAdmin2, SampleAdmin3} {
		grants = append(grants, gate.Grant{ID: "candidate-" + user, SubjectID: user, Terms: terms, Validity: validity})
	}
	for _, grant := range grants {
		if err := s.admin.AddGrant(grant); err != nil {
			return err
		}
	}
	return nil
}
