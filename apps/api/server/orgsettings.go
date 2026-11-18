package server

type OrgSettings struct {
	FilterScan    int
	DraftsOpen    int
	DraftMembers  int
	DraftsKept    int
	ActivatedKept int
	Given         map[string]bool
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
	{"view_group_drafts_open", "view-group-drafts-open", "drafts", 16, 1, 200, func(s *OrgSettings) *int { return &s.DraftsOpen }},
	{"view_group_draft_members", "view-group-draft-members", "members", 1000, 1, 10000, func(s *OrgSettings) *int { return &s.DraftMembers }},
	{"view_group_drafts_kept", "view-group-drafts-kept", "drafts", 200, 50, 10000, func(s *OrgSettings) *int { return &s.DraftsKept }},
	{"view_group_activated_kept", "view-group-activated-kept", "drafts", 1000, 100, 100000, func(s *OrgSettings) *int { return &s.ActivatedKept }},
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
	if s.Given == nil {
		s.Given = map[string]bool{}
	}
	for _, o := range orgSettings {
		if o.Value(s) == 0 && !s.Given[o.Name] {
			o.Set(&s, o.Default)
		}
	}
	return s
}

func (s OrgSettings) validate() error {
	switch {
	case s.DraftMembers > RelationshipLimits().TimelinesPerPlan:
		return settingError("view_group_draft_members")
	case s.FilterScan < RelationshipLimits().PageSize-1:
		return settingError("org_filter_scan")
	}
	for _, o := range orgSettings {
		if v := o.Value(s); v < o.Min || v > o.Max {
			return settingError(o.Name)
		}
	}
	return nil
}

func (s *Server) orgSettingRows() []settingRow {
	out := []settingRow{}
	for _, o := range orgSettings {
		source := SourceDefault
		if s.cfg.Org.Given[o.Name] {
			source = SourceStartFlag
		}
		out = append(out, settingRow{Name: o.Name, Value: o.Value(s.cfg.Org), Unit: o.Unit, Default: o.Default, Min: o.Min, Max: o.Max, Source: source})
	}
	return out
}
