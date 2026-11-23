package server

import "net/url"

func (f world) testSessionSnapshot(a actor) map[string]any {
	f.t.Helper()
	account := f.me(a)
	if account.status != 200 {
		f.t.Fatalf("account status%d", account.status)
	}
	rows := []any{}
	after := ""
	for {
		query := "?limit=100"
		if after != "" {
			query += "&after=" + url.QueryEscape(after)
		}
		page := f.sessionQuery(a, query)
		if page.status != 200 {
			f.t.Fatalf("session query status%d", page.status)
		}
		rows = append(rows, page.body["rows"].([]any)...)
		after = page.body["next"].(string)
		if after == "" {
			break
		}
		if len(rows) > maxSessions {
			f.t.Fatal("session continuation did not finish")
		}
	}
	return map[string]any{"sessions": rows, "sessions_generation": account.body["sessions_generation"]}
}

func (f world) testOperationRows(a actor, operation, view string) []any {
	f.t.Helper()
	rows := []any{}
	after := ""
	for {
		query := "/" + operation + "/items?view=" + view + "&limit=100"
		if after != "" {
			query += "&after=" + url.QueryEscape(after)
		}
		page := f.operationRead(a, query)
		if page.status != 200 {
			f.t.Fatalf("operation items status%d", page.status)
		}
		rows = append(rows, page.body["rows"].([]any)...)
		after = page.body["next"].(string)
		if after == "" {
			break
		}
		if len(rows) > maxSessions {
			f.t.Fatal("operation continuation did not finish")
		}
	}
	return rows
}

func (f world) testLatestLegacyOperation(a actor) map[string]any {
	f.t.Helper()
	page := f.operationRead(a, "?limit=1")
	if page.status != 200 || len(page.body["rows"].([]any)) != 1 {
		f.t.Fatal("no latest accepted operation")
	}
	op := page.body["rows"].([]any)[0].(map[string]any)
	id := op["operation"].(string)
	op["count"] = op["ended_count"]
	op["sessions"] = f.testOperationRows(a, id, "results")
	op["not_included"] = f.testOperationRows(a, id, "not_included")
	return op
}
