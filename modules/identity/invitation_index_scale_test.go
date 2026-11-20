package identity

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestInvitationIndexWorkspaceAndConcurrentQuota(t *testing.T) {
	limits := testLimits
	limits.PendingInvitations = 1
	f := setupWith(t, limits)
	f.user("ann")
	f.seed("w2", "other")
	root, other := f.login("root"), f.login("other")
	f.must(f.w.Invite(root, "w1", "first", Member, v("first"), f.window(), "ann"))
	f.must(f.w.Invite(other, "w2", "second", Member, v("second"), f.window(), "ann"))
	f.refusal("foreign revocation", f.w.RevokeInvitation(other, "w2", "first"), ReasonUnknown)
	assertInvitationIndex(t, f.r, *f.now)
	f.must(f.w.RevokeInvitation(root, "w1", "first"))
	results := contend(16, func(i int) error {
		id := fmt.Sprintf("quota-%02d", i)
		return f.w.Invite(root, "w1", id, Member, v(id), f.window(), "ann")
	})
	success, limited := 0, 0
	for _, err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, ErrLimit) {
			limited++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || limited != 15 || len(f.r.pendingInvitations["w1"]) != 1 || len(f.r.pendingInvitations["w2"]) != 1 {
		t.Fatalf("concurrent quota success=%d limited=%d", success, limited)
	}
	assertInvitationIndex(t, f.r, *f.now)
}

func TestInvitationIndexLongTerminalHistory(t *testing.T) {
	if testing.Short() {
		t.Skip("scale run")
	}
	limits := testLimits
	limits.PendingInvitations = 1
	f := setupWith(t, limits)
	f.user("ann")
	root, ann := f.login("root"), f.login("ann")
	began := time.Now()
	for i := 0; i < 100000; i++ {
		id := fmt.Sprintf("history-%06d", i)
		f.must(f.w.Invite(root, "w1", id, Member, v(id), f.window(), "ann"))
		if i%2 == 0 {
			f.must(f.w.Accept("w1", id, v(id), ann))
			f.must(f.w.RemoveMembership(root, "w1", "ann", f.r.member("w1", "ann").version))
		} else {
			f.must(f.w.RevokeInvitation(root, "w1", id))
		}
	}
	elapsed := time.Since(began)
	assertInvitationIndex(t, f.r, *f.now)
	if len(f.r.invitations) != 100000 || len(f.r.pendingInvitations["w1"]) != 0 {
		t.Fatal("terminal history not retained separately")
	}
	f.want("first historical ID stays reserved", f.w.Invite(root, "w1", "history-000000", Member, v("reused"), f.window(), "ann"), ErrConflict)
	f.must(f.w.Invite(root, "w1", "current", Member, v("current"), f.window(), "ann"))
	f.want("quota still applies after long history", f.w.Invite(root, "w1", "over", Member, v("over"), f.window(), "ann"), ErrLimit)
	assertInvitationIndex(t, f.r, *f.now)
	t.Logf("100000 public invite/accept-or-revoke sequences: %v", elapsed)
}
