package identity

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"
)

func assertInvitationIndex(t testing.TB, r *Registry, now time.Time) {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	want := map[string]map[string]bool{}
	counts := map[string]int{}
	for id, record := range r.invitations {
		if record.state == invitationPending {
			if want[record.workspace] == nil {
				want[record.workspace] = map[string]bool{}
			}
			want[record.workspace][id] = true
		}
		if r.countable(record, now) {
			counts[record.workspace]++
		}
	}
	for workspace, ids := range r.pendingInvitations {
		actual := 0
		for id := range ids {
			record := r.invitations[id]
			if record == nil || record.workspace != workspace || record.state != invitationPending {
				t.Fatalf("invalid indexed invitation %s/%s", workspace, id)
			}
			if !want[workspace][id] {
				t.Fatalf("unexpected indexed invitation %s/%s", workspace, id)
			}
			delete(want[workspace], id)
			if r.countable(record, now) {
				actual++
			}
		}
		if actual != counts[workspace] {
			t.Fatalf("%s indexed count=%d canonical=%d", workspace, actual, counts[workspace])
		}
	}
	for workspace, ids := range want {
		if len(ids) > 0 {
			t.Fatalf("missing pending invitations in %s: %d", workspace, len(ids))
		}
	}
}

func TestInvitationIndexPublicTerminalSequences(t *testing.T) {
	f := setup(t)
	f.user("ann")
	root, ann := f.login("root"), f.login("ann")
	random := rand.New(rand.NewSource(339))
	outcomes := [3]int{}
	for i := 0; i < 300; i++ {
		id := fmt.Sprintf("indexed-%04d", i)
		f.must(f.w.Invite(root, "w1", id, Member, v(id), f.window(), "ann"))
		assertInvitationIndex(t, f.r, *f.now)
		outcome := random.Intn(3)
		outcomes[outcome]++
		switch outcome {
		case 0:
			f.must(f.w.Accept("w1", id, v(id), ann))
			assertInvitationIndex(t, f.r, *f.now)
			member := f.r.member("w1", "ann")
			f.must(f.w.RemoveMembership(root, "w1", "ann", member.version))
		case 1:
			f.must(f.w.RevokeInvitation(root, "w1", id))
		case 2:
			for attempt := 0; attempt < testLimits.Attempts; attempt++ {
				f.refusal("wrong verifier", f.w.Accept("w1", id, v("wrong"), ann), ReasonVerifier)
				assertInvitationIndex(t, f.r, *f.now)
			}
		}
		assertInvitationIndex(t, f.r, *f.now)
		f.want("historical ID reuse", f.w.Invite(root, "w1", id, Member, v(id), f.window(), "ann"), ErrConflict)
	}
	for _, count := range outcomes {
		if count == 0 {
			t.Fatal("a terminal outcome was not exercised")
		}
	}
	if len(f.r.invitations) != 300 || len(f.r.pendingInvitations["w1"]) != 0 {
		t.Fatal("terminal history was deleted or remained indexed")
	}
	t.Logf("accepted/revoked/locked sequences: %v", outcomes)
}

func TestInvitationIndexQuotaTimeAndAuthority(t *testing.T) {
	limits := testLimits
	limits.PendingInvitations = 1
	f := setupWith(t, limits)
	f.user("ann")
	root := f.login("root")
	future := Validity{NotBefore: start.Add(time.Minute), NotAfter: start.Add(time.Hour)}
	f.must(f.w.Invite(root, "w1", "future", Member, v("future"), future, "ann"))
	f.want("future pending consumes quota", f.w.Invite(root, "w1", "second", Member, v("second"), f.window(), "ann"), ErrLimit)
	f.refusal("early acceptance", f.w.Accept("w1", "future", v("future"), f.login("ann")), ReasonExpired)
	assertInvitationIndex(t, f.r, *f.now)
	*f.now = start.Add(time.Hour)
	root = f.login("root")
	f.must(f.w.Invite(root, "w1", "second", Member, v("second"), f.window(), "ann"))
	assertInvitationIndex(t, f.r, *f.now)
	if len(f.r.pendingInvitations["w1"]) != 2 {
		t.Fatal("expired state-pending invitation was pruned")
	}
	f.must(f.w.RevokeInvitation(root, "w1", "second"))
	*f.now = start
	f.want("clock rollback restores original quota decision", f.w.Invite(f.login("root"), "w1", "third", Member, v("third"), f.window(), "ann"), ErrLimit)
	assertInvitationIndex(t, f.r, *f.now)
	f.must(f.in.SuspendUser(operator, "root", 1))
	assertInvitationIndex(t, f.r, *f.now)
	if f.r.countable(f.r.invitations["future"], *f.now) {
		t.Fatal("inactive inviter counted")
	}
	if len(f.r.pendingInvitations["w1"]) != 1 {
		t.Fatal("authority change pruned state-pending invitation")
	}
}

func TestInvitationIndexFailedAcceptVersionOverflow(t *testing.T) {
	f := setup(t)
	f.user("ann")
	root, ann := f.login("root"), f.login("ann")
	f.r.putMembership("w1", "ann", &membership{role: Member, state: MembershipRemoved, version: math.MaxUint64})
	f.must(f.w.Invite(root, "w1", "overflow", Member, v("overflow"), f.window(), "ann"))
	f.want("version overflow", f.w.Accept("w1", "overflow", v("overflow"), ann), ErrConflict)
	assertInvitationIndex(t, f.r, *f.now)
	if len(f.r.pendingInvitations["w1"]) != 1 || f.r.invitations["overflow"].state != invitationPending {
		t.Fatal("failed acceptance removed pending state")
	}
}
