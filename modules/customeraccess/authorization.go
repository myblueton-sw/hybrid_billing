package customeraccess

type authorityStamp struct {
	subject    uint64
	generation uint64
	companies  map[string]uint64
	tenants    map[string]uint64
}

func (s *Store) authorize(principal Principal, request Selection, action Action) (authorityStamp, error) {
	stamp := authorityStamp{}
	if !s.available {
		return stamp, ErrUnavailable
	}
	now := s.clock()
	subject := s.subjects[principal.SubjectID]
	if principal.Installation != s.installation || !validID(principal.SubjectID) || !now.Before(principal.ValidUntil) || subject.version == 0 || !subject.value.Active || principal.Epoch != subject.value.Epoch {
		return stamp, ErrDenied
	}
	stamp.subject = subject.version
	stamp.generation = s.generation
	stamp.companies = make(map[string]uint64, len(request.Scopes))
	stamp.tenants = make(map[string]uint64, len(request.Scopes))
	for _, scope := range request.Scopes {
		company := s.companies[scope.CompanyID]
		tenant := s.tenants[scope.TenantID]
		if company.version == 0 || !company.value.Active || tenant.version == 0 || !tenant.value.Active || tenant.value.CompanyID != scope.CompanyID {
			return authorityStamp{}, ErrDenied
		}
		allowed := make(map[string]struct{})
		for _, grant := range subject.value.Grants {
			if grant.Scope == scope && grant.Action == action && now.Before(grant.ExpiresAt) {
				for _, field := range grant.Fields {
					allowed[field] = struct{}{}
				}
			}
		}
		for _, field := range request.Fields {
			if _, ok := allowed[field]; !ok {
				return authorityStamp{}, ErrDenied
			}
		}
		stamp.companies[scope.CompanyID] = company.version
		stamp.tenants[scope.TenantID] = tenant.version
	}
	return stamp, nil
}

func (a authorityStamp) equals(b authorityStamp) bool {
	if a.subject != b.subject || a.generation != b.generation || len(a.companies) != len(b.companies) || len(a.tenants) != len(b.tenants) {
		return false
	}
	for id, version := range a.companies {
		if b.companies[id] != version {
			return false
		}
	}
	for id, version := range a.tenants {
		if b.tenants[id] != version {
			return false
		}
	}
	return true
}
