package server

import (
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func capturedAdmissionSession(f world, a actor) session {
	f.t.Helper()
	found, known := f.s.sessions.get(a.verifier(f))
	if !known || found.handle == "" || found.owner.User == "" {
		f.t.Fatal("real authenticated session was not canonically bound")
	}
	return found
}

func TestCanonicalAdmissionRefusesInconsistentCapturedValuesWithoutEndingActualActors(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(AdminLogin, pw)
	f.setPassword(Admin2Login, pw)
	first := f.signedIn(AdminLogin, pw)
	other := f.signedIn(Admin2Login, pw)
	bound := capturedAdmissionSession(f, first)
	peer := capturedAdmissionSession(f, other)
	if bound.owner == peer.owner || bound.handle == peer.handle {
		t.Fatal("independent actual actors were not distinct")
	}
	for _, name := range []string{"peer-owner", "owner-workspace", "owner-epoch", "missing-handle"} {
		t.Run(name, func(t *testing.T) {
			candidate := bound
			switch name {
			case "peer-owner":
				candidate.owner = peer.owner
			case "owner-workspace":
				candidate.owner.Workspace = "different-component-workspace"
			case "owner-epoch":
				candidate.owner.Epoch++
			case "missing-handle":
				candidate.handle = ""
			}
			if _, code := f.s.access(candidate); code != commands.Unauthenticated {
				t.Fatal("inconsistent captured values passed canonical admission", code)
			}
			for _, actual := range []actor{first, other} {
				if got := f.me(actual); got.status != 200 {
					t.Fatal("component refusal ended an unchanged actual actor", got.status)
				}
			}
			stored := capturedAdmissionSession(f, first)
			if stored.owner != bound.owner || stored.workspace != bound.workspace || stored.handle != bound.handle {
				t.Fatal("component refusal rewrote the real session binding")
			}
		})
	}
}

func TestCanonicalAdmissionKeepsSameOwnerCallersDistinctAndRejectsEndedCaptures(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(AdminLogin, pw)
	first, second := f.signedIn(AdminLogin, pw), f.signedIn(AdminLogin, pw)
	left, right := capturedAdmissionSession(f, first), capturedAdmissionSession(f, second)
	if left.owner != right.owner || left.handle == right.handle {
		t.Fatal("same-owner callers lost their independent handles")
	}
	for _, found := range []session{left, right} {
		answer, code := f.s.access(found)
		if code != "" || found.workspace != answer.Workspace() || found.owner != (sessionOwner{answer.User(), answer.Workspace(), answer.Epoch()}) {
			t.Fatal("a canonical bound caller was refused")
		}
	}
	if got := f.sessionQuery(second, "?limit=0"); got.status != 422 || got.body["code"] != string(commands.InvalidRequest) {
		t.Fatal("ordinary malformed-query classification changed", got.status)
	}
	if got := f.logout(first); got.status != 204 {
		t.Fatal("actual sign-out premise", got.status)
	}
	if _, code := f.s.access(left); code != commands.Unauthenticated {
		t.Fatal("an ended captured session regained admission")
	}
	if f.me(first).status != 401 || f.me(second).status != 200 {
		t.Fatal("ending one actual caller changed another caller's ownership")
	}
}

func TestCanonicalAdmissionPreservesInvitationAcceptanceAfterBinding(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	login := "canonical-invitation@example.invalid"
	user, token := f.created(root, login, "member")
	if got := f.run(root, "identity.revoke_invitation", user, nil); got.status != 200 {
		t.Fatal("deferred invitation premise", got.status)
	}
	if got := f.redeem(token, pw); got.status != 204 {
		t.Fatal("actual initial-password premise", got.status)
	}
	if got := f.run(root, "identity.invite", user, map[string]string{"role": "member"}); got.status != 200 {
		t.Fatal("actual pending invitation premise", got.status)
	}
	if row := f.row(root, user); row["membership"] != nil {
		t.Fatal("invitation was accepted before sign-in")
	}
	joined := f.signedIn(login, pw)
	bound := capturedAdmissionSession(f, joined)
	answer, code := f.s.access(bound)
	if code != "" || answer.User() != user || bound.owner != (sessionOwner{answer.User(), answer.Workspace(), answer.Epoch()}) {
		t.Fatal("actual invited caller was not bound before successful admission")
	}
	if got := f.me(joined); got.status != 200 || got.body["user_id"] != user {
		t.Fatal("actual accepted invitation did not reach self read", got.status)
	}
	if f.sessionQuery(joined, "?session=current").status != 200 {
		t.Fatal("accepted invitation could not read its canonical current session")
	}
}
