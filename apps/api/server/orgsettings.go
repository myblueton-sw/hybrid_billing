package server

type OrgSettings struct {
	FilterScan int
	Given      map[string]bool
}

type OrgSetting struct {
	Name    string
	Flag    string
	Unit    string
	Default int
	Min     int
	Max     int
	field   func(*OrgSettings) *int
}

func (o OrgSetting) Set(s *OrgSettings, value int) { *o.field(s) = value }

func (o OrgSetting) Value(s OrgSettings) int { return *o.field(&s) }

var orgSettings = []OrgSetting{
	{"org_filter_scan", "org-filter-scan", "timelines", 1000, 100, 10000, func(s *OrgSettings) *int { return &s.FilterScan }},
}

func OrgSettingList() []OrgSetting { return append([]OrgSetting(nil), orgSettings...) }

func DefaultOrgSettings() OrgSettings {
	s := OrgSettings{Given: map[string]bool{}}
	for _, o := range orgSettings {
		o.Set(&s, o.Default)
	}
	return s
}

func (s OrgSettings) withDefaults() OrgSettings {
	for _, o := range orgSettings {
		if o.Value(s) != 0 {
			return s
		}
	}
	return DefaultOrgSettings()
}

func (s OrgSettings) validate() error {
	for _, o := range orgSettings {
		if v := o.Value(s); v < o.Min || v > o.Max {
			return settingError(o.Name)
		}
	}
	return nil
}
