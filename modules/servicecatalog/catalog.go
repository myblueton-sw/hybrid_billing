package servicecatalog

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

const (
	MaxBytes         = 1 << 20
	MaxResourceBytes = 2048
	MaxAudiences     = 10000
	MaxScopes        = 100000
	PageLimit        = 1000
)

var (
	ErrInvalid     = errors.New("service catalog configuration is invalid")
	ErrNotFound    = errors.New("service catalog entry is not found")
	ErrUnavailable = errors.New("service catalog is unavailable")
)

type SelectorSpec struct {
	Any bool   `json:"any,omitempty"`
	ID  string `json:"id,omitempty"`
}

type PeriodSpec struct {
	Any   bool   `json:"any,omitempty"`
	From  string `json:"from,omitempty"`
	Until string `json:"until,omitempty"`
}

type ScopeSpec struct {
	ID          string       `json:"id"`
	Action      gate.Action  `json:"action"`
	Kind        string       `json:"kind"`
	Fields      []string     `json:"fields"`
	Customer    SelectorSpec `json:"customer"`
	LegalEntity SelectorSpec `json:"legal_entity"`
	Contract    SelectorSpec `json:"contract"`
	Period      PeriodSpec   `json:"period"`
}

type AudienceSpec struct {
	Workspace string      `json:"workspace"`
	ID        string      `json:"id"`
	Resource  string      `json:"resource"`
	Scopes    []ScopeSpec `json:"scopes"`
}

type Config struct {
	Audiences []AudienceSpec `json:"audiences"`
}

type Validator interface {
	DescribesServiceTerms(gate.Terms) bool
}

type Audience struct {
	Workspace  string
	ID         string
	Resource   string
	ScopeCount int
}

type Scope struct {
	ID    string
	Terms gate.Terms
}

type audienceKey struct{ workspace, id string }

type audience struct {
	row    Audience
	scopes map[string]Scope
	ids    []string
}

type Catalog struct {
	version   string
	audiences map[audienceKey]audience
	ids       map[string][]string
}

func identifier(id string) bool {
	if id == "" || len(id) > 128 {
		return false
	}
	for _, c := range id {
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == ':' || c == '-') {
			return false
		}
	}
	return true
}

func recipient(resource string) bool {
	if resource == "" || len(resource) > MaxResourceBytes || !utf8.ValidString(resource) || strings.IndexFunc(resource, func(c rune) bool { return unicode.IsControl(c) || unicode.IsSpace(c) }) >= 0 {
		return false
	}
	u, err := url.Parse(resource)
	if err != nil || u.User != nil || u.RawQuery != "" || u.ForceQuery || strings.Contains(resource, "#") {
		return false
	}
	switch u.Scheme {
	case "https", "http":
		return u.Hostname() != "" && u.Opaque == ""
	case "urn":
		namespace, value, exists := strings.Cut(u.Opaque, ":")
		return exists && identifier(namespace) && !strings.Contains(namespace, ":") && value != ""
	}
	return false
}

func (s SelectorSpec) selector() (gate.Selector, bool) {
	if s.Any {
		return gate.All(), s.ID == ""
	}
	return gate.One(s.ID), identifier(s.ID)
}

func month(raw string) (gate.Month, bool) {
	if len(raw) != 7 || raw[4] != '-' {
		return 0, false
	}
	for i, c := range raw {
		if i != 4 && (c < '0' || c > '9') {
			return 0, false
		}
	}
	year, _ := strconv.Atoi(raw[:4])
	number, _ := strconv.Atoi(raw[5:])
	value, err := gate.NewMonth(year, number)
	return value, err == nil
}

func (p PeriodSpec) period() (gate.Period, bool) {
	if p.Any {
		return gate.AnyPeriod(), p.From == "" && p.Until == ""
	}
	from, validFrom := month(p.From)
	until, validUntil := month(p.Until)
	return gate.Months(from, until), validFrom && validUntil && from < until
}

func (s ScopeSpec) terms(workspace string) (gate.Terms, bool) {
	customer, c := s.Customer.selector()
	legal, l := s.LegalEntity.selector()
	contract, k := s.Contract.selector()
	period, p := s.Period.period()
	return gate.Terms{Action: s.Action, Kind: s.Kind, Fields: append([]string{}, s.Fields...), Scope: gate.Scope{WorkspaceID: workspace, Customer: customer, LegalEntity: legal, Contract: contract}, Period: period}, c && l && k && p
}

func bounded(config Config) bool {
	if len(config.Audiences) > MaxAudiences {
		return false
	}
	scopes, bytes := 0, 16
	for _, a := range config.Audiences {
		if len(a.Scopes) > MaxScopes-scopes || !identifier(a.Workspace) || !identifier(a.ID) || !recipient(a.Resource) {
			return false
		}
		scopes += len(a.Scopes)
		bytes += 64 + len(a.Workspace) + len(a.ID) + len(a.Resource)
		for _, s := range a.Scopes {
			if !identifier(s.ID) || !identifier(string(s.Action)) || !identifier(s.Kind) || len(s.Fields) > PageLimit {
				return false
			}
			bytes += 160 + len(s.ID) + len(s.Action) + len(s.Kind) + len(s.Customer.ID) + len(s.LegalEntity.ID) + len(s.Contract.ID) + len(s.Period.From) + len(s.Period.Until)
			for _, field := range s.Fields {
				if !identifier(field) {
					return false
				}
				bytes += 3 + len(field)
			}
			if bytes > MaxBytes {
				return false
			}
		}
		if bytes > MaxBytes {
			return false
		}
	}
	return true
}

func New(config Config, validator Validator) (*Catalog, error) {
	if validator == nil {
		return nil, ErrUnavailable
	}
	if !bounded(config) {
		return nil, ErrInvalid
	}
	catalog := &Catalog{audiences: map[audienceKey]audience{}, ids: map[string][]string{}}
	normalized := Config{Audiences: make([]AudienceSpec, 0, len(config.Audiences))}
	for _, raw := range config.Audiences {
		key := audienceKey{raw.Workspace, raw.ID}
		if _, duplicate := catalog.audiences[key]; duplicate {
			return nil, ErrInvalid
		}
		entry := audience{row: Audience{Workspace: raw.Workspace, ID: raw.ID, Resource: raw.Resource, ScopeCount: len(raw.Scopes)}, scopes: map[string]Scope{}, ids: make([]string, 0, len(raw.Scopes))}
		copy := AudienceSpec{Workspace: raw.Workspace, ID: raw.ID, Resource: raw.Resource, Scopes: make([]ScopeSpec, 0, len(raw.Scopes))}
		for _, s := range raw.Scopes {
			terms, valid := s.terms(raw.Workspace)
			if _, duplicate := entry.scopes[s.ID]; duplicate || !valid || !validator.DescribesServiceTerms(terms) {
				return nil, ErrInvalid
			}
			s.Fields = append([]string{}, s.Fields...)
			sort.Strings(s.Fields)
			terms.Fields = append([]string{}, s.Fields...)
			entry.scopes[s.ID] = Scope{ID: s.ID, Terms: terms}
			entry.ids = append(entry.ids, s.ID)
			copy.Scopes = append(copy.Scopes, s)
		}
		sort.Strings(entry.ids)
		sort.Slice(copy.Scopes, func(i, j int) bool { return copy.Scopes[i].ID < copy.Scopes[j].ID })
		catalog.audiences[key] = entry
		catalog.ids[raw.Workspace] = append(catalog.ids[raw.Workspace], raw.ID)
		normalized.Audiences = append(normalized.Audiences, copy)
	}
	for workspace := range catalog.ids {
		sort.Strings(catalog.ids[workspace])
	}
	sort.Slice(normalized.Audiences, func(i, j int) bool {
		a, b := normalized.Audiences[i], normalized.Audiences[j]
		return a.Workspace < b.Workspace || a.Workspace == b.Workspace && a.ID < b.ID
	})
	data, err := json.Marshal(normalized)
	if err != nil || len(data) > MaxBytes {
		return nil, ErrInvalid
	}
	digest := sha256.Sum256(data)
	catalog.version = hex.EncodeToString(digest[:])
	return catalog, nil
}

func (c *Catalog) ready() bool {
	return c != nil && c.version != "" && c.audiences != nil && c.ids != nil
}

func (c *Catalog) Version() string {
	if c == nil {
		return ""
	}
	return c.version
}

func (c *Catalog) Audience(workspace, id string) (Audience, error) {
	if !c.ready() {
		return Audience{}, ErrUnavailable
	}
	if !identifier(workspace) || !identifier(id) {
		return Audience{}, ErrInvalid
	}
	found, exists := c.audiences[audienceKey{workspace, id}]
	if !exists {
		return Audience{}, ErrNotFound
	}
	return found.row, nil
}

func copyScope(value Scope) Scope {
	value.Terms.Fields = append([]string{}, value.Terms.Fields...)
	return value
}

func (c *Catalog) Scope(workspace, audience, id string) (Scope, error) {
	if _, err := c.Audience(workspace, audience); err != nil {
		return Scope{}, err
	}
	if !identifier(id) {
		return Scope{}, ErrInvalid
	}
	found, exists := c.audiences[audienceKey{workspace, audience}].scopes[id]
	if !exists {
		return Scope{}, ErrNotFound
	}
	return copyScope(found), nil
}
