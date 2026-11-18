package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/bulkrevoke"
)

func (f world) stage(a actor, draft string, users ...string) reply {
	f.t.Helper()
	fields := map[string]string{"users": strings.Join(users, "\n")}
	if draft != "" {
		fields["draft"] = draft
	}
	return f.run(a, "bulk_revocation.add_targets", bulkTarget, fields)
}

func (f world) prepare(a actor, draft, effect string) reply {
	f.t.Helper()
	return f.run(a, "bulk_revocation.prepare", draft, map[string]string{"effect": effect})
}

func (f world) execute(a actor, id, digest string) reply {
	f.t.Helper()
	return f.run(a, "bulk_revocation.execute", id, map[string]string{"digest": digest})
}

func field(got reply, name string) string {
	fields, _ := got.body["fields"].(map[string]any)
	value, _ := fields[name].(string)
	return value
}

func (f world) bulkRead(a actor, path string) reply {
	f.t.Helper()
	return f.do("GET", "/api/v1/identity/bulk-revocations"+path, "", func(r *http.Request) {
		if a.cookie != nil {
			r.AddCookie(a.cookie)
		}
		r.Header.Set(csrfHeader, a.csrf)
	})
}

func (f world) prepared(t *testing.T, a actor, effect string, users ...string) (string, string) {
	t.Helper()
	staged := f.stage(a, "", users...)
	if staged.status != 200 {
		t.Fatalf("stage %s", staged.raw)
	}
	got := f.prepare(a, field(staged, "draft"), effect)
	if got.status != 200 {
		t.Fatalf("prepare %s", got.raw)
	}
	return field(got, "bulk"), field(got, "digest")
}

func (f world) job(a actor, id string) map[string]any {
	f.t.Helper()
	got := f.bulkRead(a, "/"+id)
	if got.status != 200 {
		f.t.Fatalf("job %s", got.raw)
	}
	return got.body["job"].(map[string]any)
}

func (f world) items(a actor, id, query string) []map[string]any {
	f.t.Helper()
	got := f.bulkRead(a, "/"+id+"/items"+query)
	if got.status != 200 {
		f.t.Fatalf("items %s", got.raw)
	}
	out := []map[string]any{}
	for _, item := range got.body["items"].([]any) {
		out = append(out, item.(map[string]any))
	}
	return out
}

var _ = bulkrevoke.Draft

func whole(v any) int {
	n, _ := strconv.Atoi(fmt.Sprint(v))
	return n
}
