package server

var serviceDeclarationSettings = []AccessSetting{
	{"service_declaration_purpose_bytes", "service-declaration-purpose-bytes", "UTF8 bytes", 512, 1, 2048, func(s *AccessSettings) *int { return &s.ServicePurposeBytes }},
	{"service_declaration_scopes", "service-declaration-scopes", "audience/scope pairs", 16, 1, 64, func(s *AccessSettings) *int { return &s.ServiceDeclarationScopes }},
	{"service_declaration_versions", "service-declaration-versions", "versions per service", 5, 1, 20, func(s *AccessSettings) *int { return &s.ServiceDeclarationVersions }},
}

func ServiceDeclarationSettingList() []AccessSetting {
	return append([]AccessSetting(nil), serviceDeclarationSettings...)
}
