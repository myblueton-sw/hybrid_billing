package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestPluginLicenseSettings(t *testing.T) {
	if got, err := (PluginSettings{}).licenses(); err != nil || strings.Join(got, ",") != "Apache-2.0,BSD-2-Clause,BSD-3-Clause,ISC,LGPL-2.1-only,LGPL-3.0-only,MIT,MPL-2.0,Zlib" {
		t.Errorf("default %v %v", got, err)
	}
	first := DefaultPluginLicenses()
	first[0] = "changed"
	if DefaultPluginLicenses()[0] != "Apache-2.0" {
		t.Error("the default list is shared")
	}
	if got, err := (PluginSettings{Raw: "MIT,Apache-2.0,LicenseRef-x", Given: true, Uses: 1}).licenses(); err != nil || strings.Join(got, ",") != "Apache-2.0,LicenseRef-x,MIT" {
		t.Errorf("given %v %v", got, err)
	}
	if got, err := (PluginSettings{Raw: "", Given: true, Uses: 1}).licenses(); err != nil || got == nil || len(got) != 0 {
		t.Errorf("empty %#v %v", got, err)
	}
	if got, err := (PluginSettings{Raw: "MIT", Given: false, Uses: 0}).licenses(); err != nil || len(got) != 9 {
		t.Errorf("raw without given %v %v", got, err)
	}
	full := []string{}
	for index := 0; index < 64; index++ {
		full = append(full, fmt.Sprintf("L%03d", index))
	}
	if got, err := (PluginSettings{Raw: strings.Join(full, ","), Given: true, Uses: 1}).licenses(); err != nil || len(got) != 64 {
		t.Errorf("64 entries %d %v", len(got), err)
	}
	refused := map[string]PluginSettings{
		"space after comma": {Raw: "MIT, ISC", Given: true, Uses: 1},
		"leading space":     {Raw: " MIT", Given: true, Uses: 1},
		"trailing space":    {Raw: "MIT ", Given: true, Uses: 1},
		"tab":               {Raw: "MIT\tISC", Given: true, Uses: 1},
		"empty element":     {Raw: "MIT,,ISC", Given: true, Uses: 1},
		"leading comma":     {Raw: ",MIT", Given: true, Uses: 1},
		"trailing comma":    {Raw: "MIT,", Given: true, Uses: 1},
		"only comma":        {Raw: ",", Given: true, Uses: 1},
		"malformed":         {Raw: "MIT OR ISC", Given: true, Uses: 1},
		"bare reference":    {Raw: "LicenseRef-", Given: true, Uses: 1},
		"case reference":    {Raw: "licenseref-x", Given: true, Uses: 1},
		"duplicate":         {Raw: "MIT,ISC,MIT", Given: true, Uses: 1},
		"over":              {Raw: strings.Join(full, ",") + ",L999", Given: true, Uses: 1},
		"repeated":          {Raw: "MIT", Given: true, Uses: 2},
		"repeated default":  {Given: false, Uses: 2},
	}
	for name, p := range refused {
		_, err := p.licenses()
		if !errors.Is(err, ErrConfig) || !strings.HasSuffix(err.Error(), "plugin_licenses") {
			t.Errorf("%s: %v", name, err)
		}
		_, err = New(Config{Data: t.TempDir() + "/d", Seed: SeedNone, Plugins: p})
		if !errors.Is(err, ErrConfig) || !strings.HasSuffix(err.Error(), "plugin_licenses") {
			t.Errorf("%s at start: %v", name, err)
		}
	}
	if s := (PluginSettings{}).source() + " " + (PluginSettings{Given: true}).source(); s != "default start_flag" {
		t.Errorf("source %s", s)
	}
}

func TestPluginLicensesAndTheSeed(t *testing.T) {
	for name, p := range map[string]PluginSettings{
		"empty":         {Raw: "", Given: true, Uses: 1},
		"without":       {Raw: "MIT", Given: true, Uses: 1},
		"case":          {Raw: "apache-2.0", Given: true, Uses: 1},
		"reference too": {Raw: "LicenseRef-sample-proprietary,MIT", Given: true, Uses: 1},
	} {
		_, err := New(Config{Data: t.TempDir() + "/d", Seed: SeedSample, Plugins: p})
		if !errors.Is(err, ErrConfig) || err.Error() != ErrConfig.Error()+": the sample seed needs Apache-2.0 in plugin_licenses" {
			t.Errorf("%s: %v", name, err)
		}
		if _, err := New(Config{Data: t.TempDir() + "/d", Seed: SeedNone, Plugins: p}); err != nil {
			t.Errorf("%s without the seed: %v", name, err)
		}
	}
}

func TestPluginLicensesInTheRead(t *testing.T) {
	f := setupWith(t, func(c *Config) {
		c.Seed = SeedNone
		c.Plugins = PluginSettings{Raw: "", Given: true, Uses: 1}
	})
	modules, err := f.s.modulesOf(fullViewer(f.s, ""))
	raw, _ := json.Marshal(modules.Licenses)
	if err != nil || string(raw) != `{"permitted":[],"source":"start_flag"}` {
		t.Errorf("empty list without the seed %s %v", raw, err)
	}
}

func TestPluginLicenseSettingInTheRead(t *testing.T) {
	g := setupWith(t, func(c *Config) {
		c.Seed = SeedSample
		c.Plugins = PluginSettings{Raw: "Apache-2.0,LicenseRef-sample-proprietary", Given: true, Uses: 1}
	})
	root, second := g.as(AdminLogin), g.as(Admin2Login)
	got := g.integrations(root)
	if s := fmt.Sprint(part(got, "modules")["licenses"]); s != "map[permitted:[Apache-2.0 LicenseRef-sample-proprietary] source:start_flag]" {
		t.Errorf("setting %s", s)
	}
	for _, v := range part(got, "modules")["plugins"].([]any) {
		if row := v.(map[string]any); row["license_state"] != "permitted" {
			t.Errorf("version %v %v", row["version"], row["license_state"])
		}
	}
	g.ok(t, "upgrade to 1.2.0", g.run(root, "integrations.plugin_propose", samplePluginAt, map[string]string{"kind": "upgrade", "version": "1.2.0"}), "pending",
		"map[proposal:11 target:plugin-proposal-11]")
	g.ok(t, "approve 1.2.0", g.run(second, "integrations.plugin_approve", pluginProposalAt(11), nil), "applied", "")
	if v := installedRow(t, g.integrations(root)); v["version"] != "1.2.0" || v["state"] != "enabled" {
		t.Errorf("installed %v", v)
	}
}

func TestPluginLicenseRefusal(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	fields := map[string]string{"kind": "upgrade", "version": "1.2.0"}
	f.unchanged(t, "upgrade to 1.2.0", f.run(root, "integrations.plugin_propose", samplePluginAt, fields), "invalid_stage")
	body, _ := json.Marshal(map[string]any{"request_id": "req-license", "target_id": samplePluginAt, "idempotency_key": "key-license",
		"expected_revision": f.revision(root), "rationale": "sample reason", "body": fields})
	for _, name := range []string{"first send", "same key again"} {
		f.unchanged(t, name, f.call("integrations.plugin_propose", string(body), root.cookie, root.csrf), "invalid_stage")
	}
	if len(f.s.integ.entries) != 10 {
		t.Errorf("entries %d", len(f.s.integ.entries))
	}
	if _, err := f.s.plugins.Proposal(11); err == nil {
		t.Error("a refused proposal was recorded")
	}
}
