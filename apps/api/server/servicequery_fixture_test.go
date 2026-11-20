package server

import (
	"encoding/json"
	"net/url"
)

func (f world) fixtureServicePages(a actor, path string) reply {
	f.t.Helper()
	all := []any{}
	after := ""
	var first reply
	for pages := 0; pages < 10000; pages++ {
		uri, err := url.Parse(path)
		if err != nil {
			f.t.Fatal(err)
		}
		query := uri.Query()
		if after != "" {
			query.Set("after", after)
		}
		uri.RawQuery = query.Encode()
		got := f.serviceQuery(a, uri.String())
		if got.status != 200 {
			return got
		}
		if pages == 0 {
			first = got
		}
		rows, ok := got.body["rows"].([]any)
		if !ok {
			f.t.Fatal("bounded fixture source has no rows array")
		}
		all = append(all, rows...)
		next, ok := got.body["next"].(string)
		if !ok {
			f.t.Fatal("bounded fixture source has no continuation")
		}
		if next == "" {
			first.body["rows"] = all
			return first
		}
		if next == after {
			f.t.Fatal("bounded fixture cursor did not advance")
		}
		after = next
	}
	f.t.Fatal("bounded fixture source did not terminate")
	return reply{}
}

func (f world) legacyServicesFixture(a actor) reply {
	f.t.Helper()
	accounts := f.fixtureServicePages(a, "/api/v1/identity/services?limit=100")
	if accounts.status != 200 {
		return accounts
	}
	rows := accounts.body["rows"].([]any)
	for _, raw := range rows {
		row := raw.(map[string]any)
		credentials := f.fixtureServicePages(a, "/api/v1/identity/service-credentials?limit=100&service="+url.QueryEscape(row["id"].(string)))
		if credentials.status != 200 {
			return credentials
		}
		row["credentials"] = credentials.body["rows"]
	}
	owners := f.fixtureServicePages(a, "/api/v1/identity/service-owners?limit=100")
	if owners.status != 200 {
		return owners
	}
	for _, raw := range owners.body["rows"].([]any) {
		row := raw.(map[string]any)
		row["user_id"] = row["id"]
	}
	accounts.body["services"] = rows
	accounts.body["owners"] = owners.body["rows"]
	accounts.body["now"] = accounts.body["reference_at"]
	raw, err := json.Marshal(accounts.body)
	if err != nil {
		f.t.Fatal(err)
	}
	accounts.raw = string(raw)
	return accounts
}
