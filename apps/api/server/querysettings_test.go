package server

import (
	"fmt"
	"testing"
)

func settingsOf(t *testing.T, got reply, field string) []map[string]any {
	t.Helper()
	if got.status != 200 {
		t.Fatalf("read %s", got.raw)
	}
	out := []map[string]any{}
	for _, raw := range got.body[field].([]any) {
		out = append(out, raw.(map[string]any))
	}
	return out
}

func TestTheEntryReadsCarryTheListQuerySettings(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	for name, got := range map[string]reply{"access": f.accessWire(root, ""), "policy": f.policyWire(root)} {
		rows := settingsOf(t, got, "query_settings")
		if len(rows) != 3 {
			t.Fatalf("%s: %d rows", name, len(rows))
		}
		want := [][]any{{"service_query_page_size", "rows", 50, 1, 100}, {"service_query_scan", "work units", 1000, 100, 10000}, {"service_query_lifetime_seconds", "seconds", 300, 30, 3600}}
		for i, row := range rows {
			line := fmt.Sprint(row["name"], row["unit"], row["value"], row["default"], row["min"], row["max"], row["source"])
			if line != fmt.Sprint(want[i][0], want[i][1], want[i][2], want[i][2], want[i][3], want[i][4], SourceDefault) || len(row) != 7 {
				t.Errorf("%s row %d: %v", name, i, row)
			}
		}
	}
	if rows := settingsOf(t, f.accessWire(root, ""), "settings"); len(rows) != 7 || rows[0]["name"] != "access_request_hours" {
		t.Errorf("the access settings changed: %v", rows)
	}
	given := setupWith(t, func(c *Config) {
		c.Access = DefaultAccessSettings()
		c.Access.ServiceQueryPage = 10
		c.Access.Given["service_query_page_size"] = true
	})
	other := given.as(AdminLogin)
	for name, got := range map[string]reply{"access": given.accessWire(other, ""), "policy": given.policyWire(other)} {
		rows := settingsOf(t, got, "query_settings")
		if fmt.Sprint(rows[0]["value"]) != "10" || rows[0]["source"] != SourceStartFlag || fmt.Sprint(rows[0]["default"]) != "50" || rows[1]["source"] != SourceDefault {
			t.Errorf("%s under a start flag: %v", name, rows)
		}
	}
	f.setPassword(MemberLogin, pw)
	member := f.as(MemberLogin)
	for name, got := range map[string]reply{"access": f.accessWire(member, ""), "policy": f.policyWire(member)} {
		if got.status != 403 || got.body["query_settings"] != nil {
			t.Errorf("%s for a member: %s", name, got.raw)
		}
	}
	if got := given.readAs(other, trustChangesPath+"?limit=11"); got.status != 422 || fmt.Sprint(got.body["fields"]) != "[limit]" {
		t.Errorf("a size above the limit the read states: %s", got.raw)
	}
	if got := given.readAs(other, trustChangesPath+"?limit=10"); got.status != 200 {
		t.Errorf("a size at the limit the read states: %s", got.raw)
	}
	if got := given.readAs(other, trustChangesPath); got.status != 200 || fmt.Sprint(got.body["query"].(map[string]any)["limit"]) != "10" {
		t.Errorf("no size under a limit of ten: %s", got.raw)
	}
}
