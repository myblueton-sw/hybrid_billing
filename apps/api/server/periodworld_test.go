package server

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func (f world) at(d time.Duration) string {
	return f.s.clock().Add(d).Truncate(time.Minute).Format(time.RFC3339)
}

func (f world) period(a actor, target, until string) reply {
	f.t.Helper()
	fields := map[string]string{"end": endNone}
	if until != "" {
		fields = map[string]string{"end": endAt, "role_until": until}
	}
	return f.run(a, "identity.set_role_period", target, fields)
}

func (f world) records(t testing.TB, decision string) []map[string]any {
	t.Helper()
	out := []map[string]any{}
	data, _ := readLedger(f)
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var m map[string]any
		if json.Unmarshal([]byte(line), &m) == nil && (decision == "" || m["decision"] == decision) {
			out = append(out, m)
		}
	}
	return out
}

func (f world) membership(user string) identity.MemberInfo {
	f.t.Helper()
	list, err := f.s.identity.Installation().Members(replayOperator, f.s.bootWorkspace)
	if err != nil {
		f.t.Fatal(err)
	}
	for _, m := range list {
		if m.User == user {
			return m
		}
	}
	return identity.MemberInfo{}
}

func (f world) withMember() actor {
	f.t.Helper()
	f.setPassword(MemberLogin, pw)
	return f.as(MemberLogin)
}

func readLedger(f world) ([]byte, error) { return os.ReadFile(ledgerFile(f)) }

func (f world) sweep() {
	f.s.domain.Lock()
	if f.s.sweepRoles() {
		f.s.counter++
	}
	f.s.domain.Unlock()
}

func osMkdirAll(path string) error { return os.MkdirAll(path, 0o700) }
