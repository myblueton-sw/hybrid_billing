package server

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestOrgSettings(t *testing.T) {
	for _, value := range []int{99, 10001, -1} {
		_, err := New(Config{Data: t.TempDir() + "/d", Seed: SeedSample, Org: OrgSettings{FilterScan: value}})
		if !errors.Is(err, ErrConfig) || !strings.HasSuffix(err.Error(), "org_filter_scan") {
			t.Errorf("%d: %v", value, err)
		}
	}
	for _, value := range []int{100, 10000} {
		if _, err := New(Config{Data: t.TempDir() + "/d", Seed: SeedSample, Org: OrgSettings{FilterScan: value}}); err != nil {
			t.Errorf("%d: %v", value, err)
		}
	}
	if zero := (OrgSettings{}).withDefaults(); zero.FilterScan != 1000 || zero.Given == nil || zero.validate() != nil {
		t.Errorf("defaults %+v", zero)
	}
	given := OrgSettings{FilterScan: 200, Given: map[string]bool{"org_filter_scan": true}}.withDefaults()
	if given.FilterScan != 200 || !given.Given["org_filter_scan"] {
		t.Errorf("a given value %+v", given)
	}
	list := OrgSettingList()
	if len(list) != 5 || list[0].Flag != "org-filter-scan" || list[0].Unit != "timelines" || list[0].Min != 100 || list[0].Max != 10000 || list[0].Default != 1000 {
		t.Errorf("list %+v", list)
	}
	list[0].Name = "changed"
	if OrgSettingList()[0].Name != "org_filter_scan" {
		t.Error("the list aliases the settings")
	}
	s := DefaultOrgSettings()
	list[0].Set(&s, 300)
	if s.FilterScan != 300 || list[0].Value(s) != 300 {
		t.Errorf("set %+v", s)
	}
	f := setupWith(t, func(c *Config) { c.Org = OrgSettings{FilterScan: 150} })
	if got := f.orgRead(f.as(AdminLogin), ""); got.body["scan_budget"] != 150.0 {
		t.Errorf("the read reports the setting %v", got.body["scan_budget"])
	}
}

func TestDraftSettingsBoundsAndRelations(t *testing.T) {
	for name, c := range map[string]OrgSettings{
		"view_group_drafts_open":    {DraftsOpen: 201},
		"view_group_draft_members":  {DraftMembers: 10001},
		"view_group_drafts_kept":    {DraftsKept: 49},
		"view_group_activated_kept": {ActivatedKept: 100001},
	} {
		_, err := New(Config{Data: t.TempDir() + "/d", Seed: SeedSample, Org: c})
		if !errors.Is(err, ErrConfig) || !strings.HasSuffix(err.Error(), name) {
			t.Errorf("%s: %v", name, err)
		}
	}
	for name, c := range map[string]OrgSettings{
		"view_group_drafts_open":    {DraftsOpen: -1},
		"view_group_drafts_kept":    {DraftsKept: 10001},
		"view_group_activated_kept": {ActivatedKept: 99},
		"view_group_draft_members":  {DraftMembers: -1},
	} {
		if err := c.withDefaults().validate(); !strings.HasSuffix(fmt.Sprint(err), name) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if err := (OrgSettings{Given: map[string]bool{"view_group_drafts_open": true}}).withDefaults().validate(); !strings.HasSuffix(fmt.Sprint(err), "view_group_drafts_open") {
		t.Errorf("a given zero is kept and refused %v", err)
	}
	lowest := OrgSettings{FilterScan: 100, DraftsOpen: 1, DraftMembers: 1, DraftsKept: 50, ActivatedKept: 100}
	highest := OrgSettings{FilterScan: 10000, DraftsOpen: 200, DraftMembers: RelationshipLimits().TimelinesPerPlan, DraftsKept: 10000, ActivatedKept: 100000}
	for _, c := range []OrgSettings{lowest, highest} {
		if err := c.validate(); err != nil {
			t.Errorf("%+v: %v", c, err)
		}
	}
	scan := highest
	scan.FilterScan = RelationshipLimits().PageSize - 2
	if err := scan.validate(); !strings.HasSuffix(fmt.Sprint(err), "org_filter_scan") {
		t.Errorf("scan below the read page %v", err)
	}
	if d := (OrgSettings{FilterScan: 300}).withDefaults(); d.FilterScan != 300 || d.DraftsOpen != 16 || d.DraftMembers != 1000 || d.DraftsKept != 200 || d.ActivatedKept != 1000 {
		t.Errorf("partial defaults %+v", d)
	}
}
