package server

import (
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func TestSessionStoreDenialReleaseInvalidatesGenerationWithoutRemovingBinding(t *testing.T) {
	store := newSessions()
	var verifier identity.Verifier
	verifier[0] = 19
	owner := sessionOwner{"stored-person", "workspace", 1}
	if _, err := store.put(verifier, owner.Workspace, "stored-login", start, 1); err != nil || !store.bind(verifier, owner, start.Add(time.Hour)) {
		t.Fatal("normal put and binding premise")
	}
	initial := store.indexState(owner, start, 0)
	if !store.deny(verifier, "stored-login", start) || !store.isDenied(verifier) {
		t.Fatal("normal store denial premise")
	}
	denied := store.indexState(owner, start, 0)
	if denied.Version == initial.Version || denied.Count != 1 {
		t.Fatal("denial failed to invalidate generation or removed binding")
	}
	store.release(verifier)
	current := store.indexState(owner, start, 0)
	if store.isDenied(verifier) || current.Version == denied.Version || current.Count != 1 {
		t.Fatal("release failed to invalidate generation or removed binding")
	}
	for _, obsolete := range []uint64{initial.Version, denied.Version} {
		if _, _, ok := store.indexed(owner, obsolete, "", 1); ok {
			t.Fatal("pre-transition generation remained admissible")
		}
	}
	rows, more, ok := store.indexed(owner, current.Version, "", 1)
	if !ok || more || len(rows) != 1 || rows[0].verifier != verifier || rows[0].owner != owner {
		t.Fatal("released binding lost its bounded current index entry")
	}
	if _, _, ok := store.indexed(owner, current.Version, "absent-handle", 1); ok {
		t.Fatal("unknown index position admitted")
	}
	if !store.bind(verifier, owner, start.Add(time.Hour)) {
		t.Fatal("same canonical owner binding was not idempotent")
	}
}

func TestSessionOrderedEraseDoesNotRemoveAdjacentOrAlreadyRemovedIdentity(t *testing.T) {
	left := &session{handle: "a", created: start}
	middle := &session{handle: "b", created: start}
	right := &session{handle: "c", created: start}
	rows := []*session{left, middle, right}
	rows = eraseSession(rows, middle, "rows")
	for _, absent := range []*session{middle, {handle: "b", created: start}} {
		rows = eraseSession(rows, absent, "rows")
		if len(rows) != 2 || rows[0] != left || rows[1] != right {
			t.Fatal("absent ordered identity removed an adjacent record")
		}
	}
}

func TestSessionCapturedProjectionRefusesForeignOwnerAndUnavailableFreshness(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(AdminLogin, pw)
	f.setPassword(Admin2Login, pw)
	first, second := f.signedIn(AdminLogin, pw), f.signedIn(Admin2Login, pw)
	captured, peer := capturedAdmissionSession(f, first), capturedAdmissionSession(f, second)
	for _, kind := range []string{"foreign-owner", "invalid-authentication-time"} {
		candidate, expected := captured, "revoked"
		if kind == "foreign-owner" {
			candidate.owner = peer.owner
		} else {
			candidate.authenticated = time.Time{}
			expected = "unavailable"
		}
		row := f.s.sessionView(candidate, second.verifier(f))
		if row.State != expected || row.CanEnd || row.Handle != captured.handle || row.SignedIn != captured.created.UTC().Format(time.RFC3339Nano) {
			t.Fatal("inconsistent projection advertised an endable session or changed identity", kind, row.State)
		}
		if f.me(first).status != 200 || f.me(second).status != 200 {
			t.Fatal("local projection changed an actual actor")
		}
	}
}
