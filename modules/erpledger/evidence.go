package erpledger

func (r *Registry) MappingOf(workspace, connection, kind, platform string) (MappingView, error) {
	if !validID(workspace) || !validID(connection) || !mappingKinds[kind] || !validID(platform) {
		return MappingView{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.mappings[mappingKey{connection, kind, platform}] == nil {
		return MappingView{}, ErrNotFound
	}
	return *s.mappings[mappingKey{connection, kind, platform}], nil
}

func (r *Registry) SurveyOf(workspace, connection, customer, period string) (SurveyView, error) {
	if !validID(workspace) || !validID(connection) || !validID(customer) || !validMonth(period) {
		return SurveyView{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || s.surveys[surveyKey{connection, customer, period}] == nil {
		return SurveyView{}, ErrNotFound
	}
	return *s.surveys[surveyKey{connection, customer, period}], nil
}
