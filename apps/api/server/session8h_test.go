package server

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func TestEightHourSession(t *testing.T) {
	f, a := probeWorld(t)
	limits := CommandLimits()
	session := int(limits.Retention / time.Second)
	extra := 600
	revs := make([]string, 0, session+extra)
	body := func(key, request, rev string) string {
		return `{"request_id":"` + request + `","target_id":"t1","idempotency_key":"` + key + `","expected_revision":"` + rev + `","body":{"note":"hi"}}`
	}
	tick := 0
	send := func(i int) {
		t.Helper()
		if tick%840 == 0 {
			if got := f.reauth(a, pw); got.status != 204 {
				t.Fatalf("reauth at %d: %d %s", i, got.status, got.raw)
			}
		}
		var rev string
		f.s.Read(func(r string) { rev = r })
		revs = append(revs, rev)
		if got := f.call("probe", body(fmt.Sprint("k-", i), fmt.Sprint("q-", i), rev), a.cookie, a.csrf); got.status != 200 || got.body["replay"] != false {
			t.Fatalf("call %d: %d %s", i, got.status, got.raw)
		}
		f.advance(time.Second)
		tick++
	}
	held := func() []commands.Record {
		out := []commands.Record{}
		for _, r := range f.s.Commands().Journal(0, 10*session) {
			if strings.HasPrefix(r.Request, "q-") {
				out = append(out, r)
			}
		}
		return out
	}
	contiguous := func(records []commands.Record, from, to int) {
		t.Helper()
		if len(records) != to-from+1 {
			t.Fatalf("held %d want %d", len(records), to-from+1)
		}
		for n, r := range records {
			if r.Request != fmt.Sprint("q-", from+n) || r.Command != "probe" || r.Outcome != "completed" {
				t.Fatalf("record %d: %+v", n, r)
			}
			if n > 0 && r.Sequence <= records[n-1].Sequence {
				t.Fatalf("order at %d", n)
			}
		}
	}
	for i := 0; i < session; i++ {
		send(i)
	}
	if session < 28800 {
		t.Fatalf("session %d", session)
	}
	contiguous(held(), 0, session-1)
	before := *f.value
	recent := session - 100
	if got := f.call("probe", body(fmt.Sprint("k-", recent), "c-1", revs[recent]), a.cookie, a.csrf); got.status != 200 || got.body["replay"] != true || *f.value != before {
		t.Fatalf("recent replay %d %s", got.status, got.raw)
	}
	old := session - 2*3600
	f.code(t, "expired key", f.call("probe", body(fmt.Sprint("k-", old), "c-2", revs[old]), a.cookie, a.csrf), 409, string(commands.IdempotencyKeyExpired))
	f.code(t, "oldest kept key", f.call("probe", body("k-1", "c-4", revs[1]), a.cookie, a.csrf), 409, string(commands.IdempotencyKeyExpired))
	for i := session; i < session+extra; i++ {
		send(i)
	}
	contiguous(held(), extra, session+extra-1)
	journal := f.s.Commands().Journal(0, 10*session)
	if last := journal[len(journal)-1]; last.Request != fmt.Sprint("q-", session+extra-1) {
		t.Errorf("last %+v", last)
	}
	checks := 0
	for _, r := range journal {
		if r.Request == "c-1" || r.Request == "c-2" || r.Request == "c-4" {
			checks++
		}
	}
	if checks != 3 {
		t.Errorf("check records %d", checks)
	}
	var rev string
	f.s.Read(func(r string) { rev = r })
	if got := f.call("probe", `{"request_id":"c-3","target_id":"t2","idempotency_key":"k-0","expected_revision":"`+rev+`","body":{"note":"again"}}`, a.cookie, a.csrf); got.status != 200 || got.body["replay"] != false {
		t.Errorf("swept key reused %d %s", got.status, got.raw)
	}
	if u := f.s.Commands().Unrecorded(SampleWorkspace); len(u) != 0 {
		t.Errorf("unrecorded %+v", u)
	}
	if th := f.s.Commands().Throttled(SampleWorkspace); len(th) != 0 {
		t.Errorf("throttled %+v", th)
	}
}
