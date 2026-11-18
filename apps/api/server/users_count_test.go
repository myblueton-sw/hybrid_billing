package server

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func (f world) count(a actor, query string) reply {
	f.t.Helper()
	return f.do("GET", "/api/v1/identity/users/count"+query, "", func(r *http.Request) {
		if a.cookie != nil {
			r.AddCookie(a.cookie)
		}
		r.Header.Set(csrfHeader, a.csrf)
	})
}

func counted(got reply) string {
	return fmt.Sprintf("%v %v %v", got.body["count"], got.body["complete"], got.body["over_limit"])
}

func TestUsersCount(t *testing.T) {
	f, _, root := filteredWorld(t)
	everyone, _ := f.walk(root, "")
	members, _ := f.walk(root, "role=member")
	n := len(everyone)
	got := f.count(root, "?role=member")
	if got.status != 200 || counted(got) != fmt.Sprintf("%d true false", len(members)) || got.body["limit"] != 5000.0 || got.body["scan_budget"] != 20000.0 ||
		got.body["server_time"] != stamp(f.s.clock()) || got.body["list_version"] != f.s.listVersion() || len(got.body) != 7 {
		t.Fatalf("count %s", got.raw)
	}
	if got := f.count(root, "?q=nobody-at-all"); counted(got) != "0 true false" {
		t.Errorf("nobody %s", got.raw)
	}
	cases := []struct {
		budget, limit int
		want          string
	}{
		{n, n, fmt.Sprintf("%d true false", n)},
		{n, n - 1, fmt.Sprintf("%d true true", n)},
		{n, n - 2, fmt.Sprintf("%d false true", n-1)},
		{n - 1, n, fmt.Sprintf("%d false false", n-1)},
		{n - 1, n - 2, fmt.Sprintf("%d false true", n-1)},
		{n - 1, n - 1, fmt.Sprintf("%d false false", n-1)},
		{3, n, "3 false false"},
		{n + 5, n, fmt.Sprintf("%d true false", n)},
	}
	for _, c := range cases {
		f.s.cfg.Access.UserCountScan, f.s.cfg.Access.BulkMaxMatching = c.budget, c.limit
		if got := f.count(root, ""); counted(got) != c.want || whole(got.body["limit"]) != c.limit || whole(got.body["scan_budget"]) != c.budget {
			t.Errorf("budget %d limit %d: %s; want %s", c.budget, c.limit, got.raw, c.want)
		}
	}
	f.s.cfg.Access.UserCountScan, f.s.cfg.Access.BulkMaxMatching = 20000, 5000
	for query, want := range map[string]string{"?after=a:u-ann": "invalid_request", "?limit=5": "invalid_request", "?workspace=" + SampleWorkspace: "invalid_request",
		"?status=gone": "invalid_request", "?q=": "invalid_request", "?q=" + strings.Repeat("a", 65): "invalid_request", "?method=password": "unsupported_capability",
		"?role=member&role=member": "invalid_request"} {
		got := f.count(root, query)
		if code(got) != want || got.body["fields"] != nil {
			t.Errorf("%s: %s", query, got.raw)
		}
	}
	if got := f.count(actor{}, ""); got.status != 401 {
		t.Errorf("no session %s", got.raw)
	}
	if got := f.count(f.as(Admin3Login), "?role=member"); got.status != 200 {
		t.Errorf("an installation administrator without the bulk right %s", got.raw)
	}
	f.withMember()
	if got := f.count(f.signedIn(MemberLogin, pw), ""); code(got) != string(commands.PermissionDenied) {
		t.Errorf("a member %s", got.raw)
	}
	f.s.bootWorkspace = "elsewhere"
	if got := f.count(root, ""); code(got) != string(commands.PermissionDenied) {
		t.Errorf("another workspace %s", got.raw)
	}
}

func TestUsersCountFaultsAndLock(t *testing.T) {
	f, _, root := filteredWorld(t)
	f.s.fault = func(at string) bool { return at == "people" }
	if got := f.count(root, ""); code(got) != string(commands.DependencyUnavailable) {
		t.Errorf("people fault %s", got.raw)
	}
	f.s.fault = nil
	f.s.domain.RLock()
	done := make(chan reply, 1)
	go func() { done <- f.count(root, "?role=member") }()
	select {
	case got := <-done:
		if got.status != 200 {
			t.Errorf("count under a held read lock %s", got.raw)
		}
	case <-time.After(5 * time.Second):
		t.Error("the count waited for a reader")
	}
	f.s.domain.RUnlock()
}

func TestUsersReadRefusalsCarryNoFields(t *testing.T) {
	f, _, root := filteredWorld(t)
	for _, query := range []string{"?status=gone", "?q=", "?role=member&role=member", "?limit=0", "?workspace=x", "?colour=red"} {
		got := f.list(root, query)
		if got.status != 422 || code(got) != "invalid_request" || got.body["fields"] != nil {
			t.Errorf("%s: %s", query, got.raw)
		}
	}
}

func TestParserNamesTheRefusedParameter(t *testing.T) {
	for raw, want := range map[string]string{"q=": "q", "status=gone": "status", "role=member&status=x": "status", "after=zz": "after", "limit=0": "limit", "colour=red": "colour",
		"workspace=w": "workspace", "method=x": "method", "method=x&status=gone": "status", "q=%20%20": "q"} {
		values, _ := url.ParseQuery(raw)
		_, got, name := parseUsersQuery(values)
		if got == "" || name != want {
			t.Errorf("%s: %s %s; want %s", raw, got, name, want)
		}
	}
	if _, got, name := parseUsersQuery(nil); got != "" || name != "" {
		t.Errorf("empty %s %s", got, name)
	}
}

func TestUsersCountIsRecordedInTheDocument(t *testing.T) {
	f := setup(t, SeedSample).ready()
	doc := f.do("GET", "/api/v1/openapi.json", "", nil)
	reads := doc.body["x-reads"].(map[string]any)
	read := reads["/identity/users/count"].(map[string]any)["get"].(map[string]any)
	refused := read["x-refused-parameters"].(map[string]any)
	if read["operationId"] != "identity.users.count" || len(read["parameters"].([]any)) != 8 || refused["after"] != "invalid_request" || refused["limit"] != "invalid_request" ||
		refused["method"] != "unsupported_capability" || fmt.Sprint(read["x-stops"]) != "[end_of_list user_count_scan bulk_revocation_max_matching_plus_one]" {
		t.Errorf("count read %+v", read)
	}
	responses := read["responses"].(map[string]any)
	for status, want := range map[string]string{"401": "[unauthenticated]", "403": "[csrf_refused permission_denied]", "422": "[invalid_request]", "501": "[unsupported_capability]",
		"503": "[authority_unavailable dependency_unavailable]"} {
		if got := fmt.Sprint(responses[status].(map[string]any)["x-codes"]); got != want {
			t.Errorf("%s codes %s", status, got)
		}
	}
	if _, conflict := responses["409"]; conflict {
		t.Error("the count has no restart answer")
	}
	bulk := reads["/identity/bulk-revocations"].(map[string]any)["get"].(map[string]any)
	if fmt.Sprint(bulk["x-selection"]) != "[explicit filter not_recorded]" || !strings.Contains(bulk["x-changed"].(string), "not_recorded") ||
		!strings.Contains(bulk["x-search-masked"].(string), "display rule") {
		t.Errorf("bulk reads %+v", bulk)
	}
	command := doc.body["paths"].(map[string]any)["/commands/bulk_revocation.add_matching"].(map[string]any)["post"].(map[string]any)
	if command["x-step-up"] != true {
		t.Errorf("add_matching %+v", command)
	}
}
