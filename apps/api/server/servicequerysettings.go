package server

var serviceQuerySettings = []AccessSetting{
	{"service_query_page_size", "service-query-page-size", "rows", 50, 1, 100, func(s *AccessSettings) *int { return &s.ServiceQueryPage }},
	{"service_query_scan", "service-query-scan", "work units", 1000, 100, 10000, func(s *AccessSettings) *int { return &s.ServiceQueryScan }},
	{"service_query_lifetime_seconds", "service-query-lifetime-seconds", "seconds", 300, 30, 3600, func(s *AccessSettings) *int { return &s.ServiceQuerySeconds }},
	{"credential_warning_days", "credential-warning-days", "days", 14, 1, 90, func(s *AccessSettings) *int { return &s.CredentialWarningDays }},
}

func ServiceQuerySettingList() []AccessSetting {
	return append([]AccessSetting(nil), serviceQuerySettings...)
}
