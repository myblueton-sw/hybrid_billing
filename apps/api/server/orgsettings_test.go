package server

import (
	"errors"
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
	if len(list) != 1 || list[0].Flag != "org-filter-scan" || list[0].Unit != "timelines" || list[0].Min != 100 || list[0].Max != 10000 || list[0].Default != 1000 {
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
