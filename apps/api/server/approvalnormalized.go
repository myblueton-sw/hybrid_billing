package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/myblueton-sw/hybrid_billing/modules/contracts"
	"github.com/myblueton-sw/hybrid_billing/modules/intake"
	"github.com/myblueton-sw/hybrid_billing/modules/normalize"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
	"github.com/myblueton-sw/hybrid_billing/modules/pricingpolicy"
	"github.com/myblueton-sw/hybrid_billing/modules/rating"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

type normalizedClass struct {
	Source string
	Class  rating.Class
}

type normalizedBinding struct {
	SourceMode       string
	SourceVersion    string
	ID               string
	Version          string
	Provider         string
	Scope            string
	Period           string
	MappingRevision  uint64
	Contract         string
	ContractRevision uint64
	BillingProfile   string
	BillingRevision  uint64
	From             string
	To               string
	Class            string
	Classes          []normalizedClass
	Quantity         string
	SKU              string
	Service          string
	Group            string
	Profile          decimal.ProfileSpec
	Limits           rating.Limits
	Policy           rating.Policy
}

type normalizedRowOrigin struct {
	Key      normalize.RowKey
	Chunk    int
	Position int
	Digest   string
	Line     string
}

type normalizedOrigin struct {
	InputEligibility *candidateInputProvenance `json:",omitempty"`
	Pricing          *pricingpolicy.Record
	Source           candidateSource
	Lifecycle        candidateLifecycle
	Binding          normalizedBinding
	Run              string
	Mapping          normalize.MappingKey
	MappingRevision  uint64
	Manifest         uint64
	Profile          string
	ProfileRevision  string
	WindowFrom       string
	WindowTo         string
	TargetDigest     string
	MappingDigest    string
	Contract         organizations.ContractRevision
	BillingProfile   organizations.BillingProfile
	Document         organizations.DocumentKey
	Rows             []normalizedRowOrigin
}

type normalizedCharges struct {
	captureBindings map[[2]string][]byte
	pricingPolicies *pricingpolicy.Registry
	intake          *intake.Registry
	sourceProfile   decimal.Profile
	profileSpec     decimal.ProfileSpec
	sourceChunk     int
	sources         map[[2]string]*sourceAssembly
	sourceArtifacts map[[2]string][2]string
	lifecycles      *contracts.Registry
	domain          *sync.RWMutex
	normal          *normalize.Registry
	orgs            *organizations.Registry
	target          normalize.Target
	page            int
	candidates      chargeCandidates
}

func newNormalizedCharges(orgs *organizations.Registry, lifecycles *contracts.Registry, domain *sync.RWMutex, limits normalize.Limits, spec decimal.ProfileSpec, target normalize.Target, clock func() time.Time) (*normalizedCharges, error) {
	if orgs == nil || lifecycles == nil || domain == nil {
		return nil, errCandidateInvalid
	}
	profile, err := decimal.NewProfile(spec)
	if err != nil {
		return nil, err
	}
	normal, err := normalize.NewRegistry(limits, profile, clock)
	if err != nil {
		return nil, err
	}
	if err := normal.DeclareTarget(target); err != nil {
		return nil, err
	}
	target.Fields = append([]normalize.Field(nil), target.Fields...)
	page := limits.Page
	if page > 100 {
		page = 100
	}
	inputs, err := intake.NewRegistry(intake.Limits{MaxSize: candidateBytes, MaxRatio: 1, Chunks: limits.Chunks, ChunkRecords: limits.ChunkRows, Artifacts: 1000, Records: limits.Rows, Manifests: 1000, Page: page, Listed: 1000}, profile, clock)
	if err != nil {
		return nil, err
	}
	return &normalizedCharges{normal: normal, orgs: orgs, lifecycles: lifecycles, domain: domain, target: target, page: page, intake: inputs, sourceProfile: profile, profileSpec: spec, sourceChunk: limits.ChunkRows, sources: make(map[[2]string]*sourceAssembly), sourceArtifacts: make(map[[2]string][2]string)}, nil
}

func normalizedDigest(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

func (p *normalizedCharges) activeParty(workspace, id string) error {
	party, _, err := p.orgs.Party(workspace, id)
	if err != nil {
		return err
	}
	if party.State != organizations.Active {
		return errCandidateInvalid
	}
	if party.Kind == organizations.InternalUnit {
		parent, _, err := p.orgs.Party(workspace, party.ParentID)
		if err != nil {
			return err
		}
		if parent.Kind != organizations.LegalEntity || parent.State != organizations.Active {
			return errCandidateInvalid
		}
	}
	return nil
}

func (p *normalizedCharges) attribution(workspace string, b normalizedBinding) (organizations.ContractRevision, organizations.BillingProfile, organizations.DocumentKey, error) {
	var contract organizations.ContractRevision
	var profile organizations.BillingProfile
	var key organizations.DocumentKey
	ws, _, err := p.orgs.Workspace(workspace)
	if err != nil {
		return contract, profile, key, err
	}
	if ws.State != organizations.Active {
		return contract, profile, key, errCandidateInvalid
	}
	contract, err = p.orgs.ContractRevision(workspace, b.Contract, b.ContractRevision)
	if err != nil {
		return contract, profile, key, err
	}
	var version uint64
	profile, version, err = p.orgs.BillingProfile(workspace, b.BillingProfile)
	if err != nil {
		return contract, profile, key, err
	}
	if version != b.BillingRevision || profile.State != organizations.Active || profile.WorkspaceID != workspace || contract.WorkspaceID != workspace ||
		contract.CustomerID != profile.BuyerID || contract.SellerID != profile.IssuerID || profile.Currency != b.Policy.Currency || string(contract.Path) != string(b.Policy.Path) {
		return contract, profile, key, errCandidateInvalid
	}
	for _, id := range []string{profile.BuyerID, profile.IssuerID, profile.PayerID, profile.RecipientID} {
		if id != "" {
			if err := p.activeParty(workspace, id); err != nil {
				return contract, profile, key, err
			}
		}
	}
	key, err = p.orgs.DocumentKey(workspace, b.BillingProfile)
	return contract, profile, key, err
}

func (p *normalizedCharges) fields(b normalizedBinding) (string, map[string]rating.Class, error) {
	if b.SourceMode != "synthetic" || !validRequestID(b.ID) || !validRequestID(b.Version) || len(b.Classes) == 0 || len(b.Classes) > 1000 {
		return "", nil, errCandidateInvalid
	}
	fields := make(map[string]normalize.Field)
	cost := ""
	for _, field := range p.target.Fields {
		fields[field.Name] = field
		if field.Kind == normalize.Cost && string(field.Cost) == string(b.Policy.CostBasis) {
			if cost != "" {
				return "", nil, errCandidateInvalid
			}
			cost = field.Name
		}
	}
	if cost == "" || b.Class == "" || fields[b.Class].Kind != normalize.Text || fields[p.target.CostCurrency].Kind != normalize.Currency {
		return "", nil, errCandidateInvalid
	}
	for _, name := range []string{b.SKU, b.Service, b.Group} {
		if name != "" && fields[name].Kind != normalize.Text {
			return "", nil, errCandidateInvalid
		}
	}
	if b.Quantity != "" && fields[b.Quantity].Kind != normalize.Quantity {
		return "", nil, errCandidateInvalid
	}
	for _, rule := range b.Policy.Rules {
		if rule.Level == rating.SKU && b.SKU == "" || rule.Level == rating.Service && b.Service == "" || rule.Level == rating.Group && b.Group == "" {
			return "", nil, errCandidateInvalid
		}
		if (rule.Type == rating.UnitPrice || rule.Type == rating.Graduated || rule.Type == rating.Volume) && b.Quantity == "" {
			return "", nil, errCandidateInvalid
		}
	}
	classes := make(map[string]rating.Class)
	for _, entry := range b.Classes {
		if !validRequestID(entry.Source) || classes[entry.Source] != "" {
			return "", nil, errCandidateInvalid
		}
		switch entry.Class {
		case rating.CSPUsage, rating.ServiceFee, rating.Licence, rating.Other:
		default:
			return "", nil, errCandidateInvalid
		}
		classes[entry.Source] = entry.Class
	}
	return cost, classes, nil
}

func (p *normalizedCharges) capture(workspace, runID, candidateID string, b normalizedBinding) (chargeCandidate, error) {
	p.domain.Lock()
	defer p.domain.Unlock()
	return p.captureLocked(workspace, runID, candidateID, b)
}

func (p *normalizedCharges) captureLocked(workspace, runID, candidateID string, b normalizedBinding) (chargeCandidate, error) {
	return p.captureBoundLocked(workspace, runID, candidateID, b, nil)
}

func (p *normalizedCharges) captureBoundLocked(workspace, runID, candidateID string, b normalizedBinding, pricing *pricingpolicy.Record) (chargeCandidate, error) {
	return p.captureEvidenceLocked(workspace, runID, candidateID, b, pricing, nil)
}

func (p *normalizedCharges) captureEvidenceLocked(workspace, runID, candidateID string, b normalizedBinding, pricing *pricingpolicy.Record, eligibility *candidateInputProvenance) (chargeCandidate, error) {
	cost, classes, err := p.fields(b)
	if err != nil {
		return chargeCandidate{}, err
	}
	run, err := p.normal.Run(workspace, runID)
	if err != nil {
		return chargeCandidate{}, err
	}
	if run.State != normalize.Closed || run.Spec.Rows < 1 || run.Spec.Rows > 1000 || run.Checked != run.Spec.Rows || run.Normal != run.Spec.Rows || run.Errors != 0 || run.Quarantined != 0 || run.Contradiction != 0 ||
		run.Format == nil || run.Format.Result != normalize.Conformant || run.Billing == nil || run.Billing.Result != normalize.Ready ||
		run.Spec.Mapping.Provider != b.Provider || run.Spec.Mapping.SourceVersion != b.SourceVersion || run.Spec.Mapping.Target != p.target.ID || run.Spec.Revision != b.MappingRevision ||
		run.Spec.Input.Scope != b.Scope || run.Spec.Input.Period != b.Period || run.Spec.Input.Manifest == 0 || run.Spec.Window == nil {
		return chargeCandidate{}, errCandidateInvalid
	}
	from, e1 := time.Parse(time.DateOnly, b.From)
	to, e2 := time.Parse(time.DateOnly, b.To)
	if e1 != nil || e2 != nil || from.After(to) || to.AddDate(0, 0, 1).Year() > 9999 || !run.Spec.Window.From.Equal(from) || !run.Spec.Window.To.Equal(to.AddDate(0, 0, 1)) {
		return chargeCandidate{}, errCandidateInvalid
	}
	revision, err := p.normal.Revision(run.Spec.Mapping, run.Spec.Revision)
	if err != nil {
		return chargeCandidate{}, err
	}
	contract, profile, document, err := p.attribution(workspace, b)
	if err != nil {
		return chargeCandidate{}, err
	}
	lifecycle, err := p.lifecycleCoverage(workspace, b)
	if err != nil {
		return chargeCandidate{}, err
	}
	source, err := p.sourceForRun(workspace, runID, run.Spec)
	if err != nil {
		return chargeCandidate{}, err
	}
	origin := normalizedOrigin{InputEligibility: eligibility, Pricing: pricing, Source: source, Lifecycle: lifecycle, Binding: b, Run: runID, Mapping: run.Spec.Mapping, MappingRevision: run.Spec.Revision, Manifest: run.Spec.Input.Manifest,
		Profile: run.ProfileID, ProfileRevision: run.ProfileRevision, WindowFrom: run.Spec.Window.From.UTC().Format(time.RFC3339Nano), WindowTo: run.Spec.Window.To.UTC().Format(time.RFC3339Nano), Contract: contract, BillingProfile: profile, Document: document}
	origin.TargetDigest, err = normalizedDigest(p.target)
	if err != nil {
		return chargeCandidate{}, err
	}
	origin.MappingDigest, err = normalizedDigest(revision)
	if err != nil {
		return chargeCandidate{}, err
	}
	input := chargeInput{SourceMode: b.SourceMode, Workspace: workspace, ID: candidateID, Customer: contract.CustomerID, Entity: document.IssuerLegalEntityID, From: b.From, To: b.To, Profile: b.Profile, Limits: b.Limits, Policy: b.Policy}
	seen := make(map[normalize.RowKey]bool)
	for after := 0; after < run.Spec.Rows; {
		page, err := p.normal.Rows(workspace, runID, after, p.page)
		if err != nil {
			return chargeCandidate{}, err
		}
		if page.Format.Result != normalize.Conformant || page.Billing.Result != normalize.Ready || len(page.Rows) == 0 || len(page.Rows) > p.page || after+len(page.Rows) > run.Spec.Rows {
			return chargeCandidate{}, errCandidateInvalid
		}
		for _, row := range page.Rows {
			if seen[row.Key] || len(row.Lacking) != 0 {
				return chargeCandidate{}, errCandidateInvalid
			}
			seen[row.Key] = true
			for name, cell := range row.Cells {
				if !utf8.ValidString(name) || !utf8.ValidString(cell.Text) {
					return chargeCandidate{}, errCandidateInvalid
				}
			}
			value := func(name string) (string, bool) {
				if name == "" {
					return "", true
				}
				cell, ok := row.Cells[name]
				return cell.Text, ok && cell.State == normalize.Value
			}
			values := make([]string, 7)
			for i, field := range []string{b.Class, cost, p.target.CostCurrency, b.Quantity, b.SKU, b.Service, b.Group} {
				text, ok := value(field)
				if !ok {
					return chargeCandidate{}, errCandidateInvalid
				}
				values[i] = text
			}
			if classes[values[0]] == "" || values[2] != b.Policy.Currency {
				return chargeCandidate{}, errCandidateInvalid
			}
			identity := struct {
				Schema, Workspace, Provider, Scope, Period string
				Key                                        normalize.RowKey
			}{"hb_normalized_line_v1", workspace, b.Provider, b.Scope, b.Period, row.Key}
			id, err := normalizedDigest(identity)
			if err != nil {
				return chargeCandidate{}, err
			}
			rowDigest, err := normalizedDigest(row)
			if err != nil {
				return chargeCandidate{}, err
			}
			input.Lines = append(input.Lines, rating.Line{ID: id, Class: classes[values[0]], Cost: values[1], Currency: values[2], Quantity: values[3], SKU: values[4], Service: values[5], Group: values[6]})
			origin.Rows = append(origin.Rows, normalizedRowOrigin{Key: row.Key, Chunk: row.Chunk, Position: row.Position, Digest: rowDigest, Line: id})
		}
		after += len(page.Rows)
	}
	input.Normalized = origin
	provenance, err := normalizedDigest(origin)
	if err != nil {
		return chargeCandidate{}, err
	}
	input.Sources = []chargeSource{{ID: runID, Revision: b.Version, Digest: provenance}}
	return p.candidates.capture(input)
}

func (p *normalizedCharges) sourceAmount(text string) (string, error) {
	value, err := decimal.Parse(p.sourceProfile, text)
	if err != nil {
		return "", err
	}
	return value.String(), nil
}
