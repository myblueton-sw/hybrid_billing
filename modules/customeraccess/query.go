package customeraccess

import "context"

type Service struct {
	verify VerifyFunc
	store  *Store
	read   ReadFunc
	limits Limits
}

func NewService(verify VerifyFunc, store *Store, read ReadFunc, limits Limits) (*Service, error) {
	if verify == nil || store == nil || read == nil || limits.MaxRows <= 0 || limits.MaxScopes <= 0 || limits.MaxFields <= 0 {
		return nil, ErrInvalidPolicy
	}
	store.mu.RLock()
	defer store.mu.RUnlock()
	if store.clock == nil {
		return nil, ErrInvalidPolicy
	}
	return &Service{verify: verify, store: store, read: read, limits: limits}, nil
}

func (s *Service) Query(ctx context.Context, request Selection) ([]Row, error) {
	if ctx == nil || s == nil || s.store == nil {
		return nil, ErrDenied
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !s.validSelection(request) {
		return nil, ErrDenied
	}
	request = copySelection(request)
	principal, err := s.verify(ctx)
	if err != nil {
		return nil, ErrDenied
	}
	s.store.mu.RLock()
	before, err := s.store.authorize(principal, request)
	s.store.mu.RUnlock()
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rows, err := s.read(ctx, copySelection(request))
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, ErrUnavailable
	}
	s.store.mu.RLock()
	defer s.store.mu.RUnlock()
	after, err := s.store.authorize(principal, request)
	if err != nil {
		return nil, err
	}
	if !before.equals(after) {
		return nil, ErrDenied
	}
	result, err := validateRows(rows, request)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if _, err := s.store.authorize(principal, request); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Service) validSelection(request Selection) bool {
	if request.Limit <= 0 || request.Limit > s.limits.MaxRows || len(request.Scopes) == 0 || len(request.Scopes) > s.limits.MaxScopes || len(request.Fields) == 0 || len(request.Fields) > s.limits.MaxFields {
		return false
	}
	scopes := make(map[Scope]struct{}, len(request.Scopes))
	for _, scope := range request.Scopes {
		if !validID(scope.CompanyID) || !validID(scope.TenantID) {
			return false
		}
		if _, exists := scopes[scope]; exists {
			return false
		}
		scopes[scope] = struct{}{}
	}
	fields := make(map[string]struct{}, len(request.Fields))
	for _, field := range request.Fields {
		if _, known := s.store.fields[field]; !known {
			return false
		}
		if _, exists := fields[field]; exists {
			return false
		}
		fields[field] = struct{}{}
	}
	return true
}

func validateRows(rows []Row, request Selection) ([]Row, error) {
	if len(rows) > request.Limit {
		return nil, ErrUnavailable
	}
	scopes := make(map[Scope]struct{}, len(request.Scopes))
	for _, scope := range request.Scopes {
		scopes[scope] = struct{}{}
	}
	type rowKey struct {
		scope Scope
		id    string
	}
	seen := make(map[rowKey]struct{}, len(rows))
	result := make([]Row, 0, len(rows))
	for _, row := range rows {
		if _, ok := scopes[row.Scope]; !ok || !validID(row.ID) || len(row.Values) != len(request.Fields) {
			return nil, ErrUnavailable
		}
		key := rowKey{row.Scope, row.ID}
		if _, exists := seen[key]; exists {
			return nil, ErrUnavailable
		}
		seen[key] = struct{}{}
		values := make(map[string]string, len(request.Fields))
		for _, field := range request.Fields {
			value, exists := row.Values[field]
			if !exists {
				return nil, ErrUnavailable
			}
			values[field] = value
		}
		result = append(result, Row{Scope: row.Scope, ID: row.ID, Values: values})
	}
	return result, nil
}
