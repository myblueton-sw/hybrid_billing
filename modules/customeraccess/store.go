package customeraccess

import (
	"math"
	"sync"
	"time"
)

type record[T any] struct {
	value   T
	version uint64
}

type Store struct {
	mu           sync.RWMutex
	installation string
	fields       map[string]struct{}
	clock        func() time.Time
	companies    map[string]record[Company]
	tenants      map[string]record[Tenant]
	subjects     map[string]record[Subject]
	available    bool
	generation   uint64
}

func NewStore(installation string, fields []string, clock func() time.Time) (*Store, error) {
	if !validID(installation) || len(fields) == 0 || clock == nil {
		return nil, ErrInvalidPolicy
	}
	catalog := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		if !validID(field) {
			return nil, ErrInvalidPolicy
		}
		if _, exists := catalog[field]; exists {
			return nil, ErrInvalidPolicy
		}
		catalog[field] = struct{}{}
	}
	return &Store{
		installation: installation, fields: catalog, clock: clock,
		companies: make(map[string]record[Company]), tenants: make(map[string]record[Tenant]),
		subjects: make(map[string]record[Subject]), available: true, generation: 1,
	}, nil
}

func nextVersion(current, expected uint64) (uint64, error) {
	if current != expected || current == math.MaxUint64 {
		return 0, ErrConflict
	}
	return current + 1, nil
}

func (s *Store) PutCompany(company Company, expected uint64) (uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.clock == nil || !validID(company.ID) {
		return 0, ErrInvalidPolicy
	}
	version, err := nextVersion(s.companies[company.ID].version, expected)
	if err != nil {
		return 0, err
	}
	s.companies[company.ID] = record[Company]{company, version}
	return version, nil
}

func (s *Store) PutTenant(tenant Tenant, expected uint64) (uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.clock == nil || !validID(tenant.ID) || !validID(tenant.CompanyID) || s.companies[tenant.CompanyID].version == 0 {
		return 0, ErrInvalidPolicy
	}
	version, err := nextVersion(s.tenants[tenant.ID].version, expected)
	if err != nil {
		return 0, err
	}
	s.tenants[tenant.ID] = record[Tenant]{tenant, version}
	return version, nil
}

func (s *Store) PutSubject(subject Subject, expected uint64) (uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.clock == nil || !validID(subject.ID) || subject.Epoch == 0 {
		return 0, ErrInvalidPolicy
	}
	old := s.subjects[subject.ID]
	version, err := nextVersion(old.version, expected)
	if err != nil {
		return 0, err
	}
	if old.version != 0 && (subject.Epoch < old.value.Epoch || (!old.value.Active && subject.Active && subject.Epoch <= old.value.Epoch)) {
		return 0, ErrInvalidPolicy
	}
	grants := make([]Grant, len(subject.Grants))
	for i, grant := range subject.Grants {
		tenant := s.tenants[grant.Scope.TenantID]
		if tenant.version == 0 || tenant.value.CompanyID != grant.Scope.CompanyID || grant.ExpiresAt.IsZero() || len(grant.Fields) == 0 || (grant.Action != UsageRead && grant.Action != UsageExport) {
			return 0, ErrInvalidPolicy
		}
		seen := make(map[string]struct{}, len(grant.Fields))
		for _, field := range grant.Fields {
			if _, ok := s.fields[field]; !ok {
				return 0, ErrInvalidPolicy
			}
			if _, exists := seen[field]; exists {
				return 0, ErrInvalidPolicy
			}
			seen[field] = struct{}{}
		}
		grants[i] = grant
		grants[i].Fields = append([]string(nil), grant.Fields...)
	}
	subject.Grants = grants
	s.subjects[subject.ID] = record[Subject]{subject, version}
	return version, nil
}

func (s *Store) SetAvailable(available bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.clock == nil || s.generation == math.MaxUint64 {
		return ErrInvalidPolicy
	}
	if s.available != available {
		s.available = available
		s.generation++
	}
	return nil
}
