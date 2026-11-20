package approvals

import (
	"errors"
	"testing"
	"time"
)

func TestPrepareAuthorityIsPureAndExact(t *testing.T) {
	r, _ := setup(t, 1)
	before := r.now
	calls := 0
	r.clock = func() time.Time { calls++; return before }
	proposals, authorities := len(r.workspaces["ws"].proposals), len(r.workspaces["ws"].authorities)
	for _, amount := range []string{"0", "15", "1234.56789012"} {
		a := Authority{ID: "prepare", Approver: "ann", Stage: StageCharge, Currency: "USD", Limit: amount, From: "2026-01-01", To: "2026-12-31"}
		got, err := r.PrepareAuthority(a)
		if err != nil || got != a {
			t.Fatal(got, err)
		}
	}
	for _, amount := range []string{"-1", "NaN", "1e3", "999999999999999999999999999999999999999999999999999999999999999999999999"} {
		_, err := r.PrepareAuthority(Authority{ID: "bad", Approver: "ann", Stage: StageCharge, Currency: "USD", Limit: amount, From: "2026-01-01"})
		if !errors.Is(err, ErrInvalid) {
			t.Fatal(amount, err)
		}
	}
	if calls != 0 || r.now != before || len(r.workspaces["ws"].proposals) != proposals || len(r.workspaces["ws"].authorities) != authorities {
		t.Fatal("preparation mutated registry")
	}
}

func TestAuthorityAtKeepsEffectiveTimeAfterRollback(t *testing.T) {
	r, c := setup(t, 1)
	a, at, err := r.AuthorityAt("ws", "g-ann")
	if err != nil {
		t.Fatal(err)
	}
	if at.Before(a.Granted) {
		t.Fatal("read before grant")
	}
	c.advance(48 * time.Hour)
	_, future, err := r.AuthorityAt("ws", "g-ann")
	if err != nil {
		t.Fatal(err)
	}
	c.advance(-48 * time.Hour)
	_, again, err := r.AuthorityAt("ws", "g-ann")
	if err != nil || !again.Equal(future) {
		t.Fatal("rollback", again, future, err)
	}
	for _, pair := range [][2]string{{"foreign", "g-ann"}, {"ws", "missing"}, {"", "g-ann"}, {"ws", ""}} {
		if _, _, err := r.AuthorityAt(pair[0], pair[1]); err == nil {
			t.Fatal(pair)
		}
	}
}
