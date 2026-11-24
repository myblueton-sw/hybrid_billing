package identity

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func replaceFixture(t *testing.T, perService int) (fixture, func() Verifier) {
	t.Helper()
	limits := testLimits
	limits.CredentialsPerServiceUser = perService
	f := setupWith(t, limits)
	f.member("ann", Member)
	admin := func() Verifier { return f.login("root") }
	f.must(f.w.CreateServiceUser(admin(), "w1", "svc", "root"))
	return f, admin
}

func (f fixture) credential(admin func() Verifier, id string) CredentialInfo {
	f.t.Helper()
	_, info, err := f.w.Credential(admin(), "w1", id)
	f.must(err)
	return info
}

func TestReplaceCredentialAddsTheNewOneAndBoundsTheOldOneInOneStep(t *testing.T) {
	f, admin := replaceFixture(t, 3)
	long := Validity{NotBefore: start, NotAfter: start.Add(100 * time.Hour)}
	f.must(f.w.AddCredential(admin(), "w1", "svc", "old", v("old"), long))
	before, err := f.b.AccessByCredential("old", v("old"))
	f.must(err)
	next := Validity{NotBefore: start, NotAfter: start.Add(200 * time.Hour)}
	end := start.Add(24 * time.Hour)
	f.must(f.w.ReplaceCredential(admin(), "w1", "old", "new", v("new"), next, end))
	if got := f.credential(admin, "old"); got != (CredentialInfo{ID: "old", Validity: Validity{NotBefore: start, NotAfter: end}, ReplacedBy: "new"}) {
		t.Fatalf("old credential %+v", got)
	}
	if got := f.credential(admin, "new"); got != (CredentialInfo{ID: "new", Validity: next, Replaces: "old"}) {
		t.Fatalf("new credential %+v", got)
	}
	list, err := f.w.Credentials(admin(), "w1", "svc")
	if err != nil || len(list) != 2 || list[0].ID != "new" || list[0].Replaces != "old" || list[1].ReplacedBy != "new" {
		t.Fatalf("listed %+v %v", list, err)
	}
	if _, err := f.b.AccessByCredential("new", v("new")); err != nil {
		t.Fatalf("the new credential: %v", err)
	}
	*f.now = end.Add(-time.Nanosecond)
	if _, err := f.b.AccessByCredential("old", v("old")); err != nil {
		t.Fatalf("the old credential just before its new end: %v", err)
	}
	f.must(f.b.Recheck(before))
	*f.now = end
	f.refusal("the old credential at its new end", second(f.b.AccessByCredential("old", v("old"))), ReasonExpired)
	f.refusal("an answer obtained before the rotation, rechecked at the new end although it reports the earlier end", f.b.Recheck(before), ReasonAnswer)
	if _, err := f.b.AccessByCredential("new", v("new")); err != nil {
		t.Fatalf("the new credential after the old one ended: %v", err)
	}
}

func TestReplaceCredentialNeverExtendsTheOldEnd(t *testing.T) {
	f, admin := replaceFixture(t, 3)
	short := Validity{NotBefore: start, NotAfter: start.Add(time.Hour)}
	f.must(f.w.AddCredential(admin(), "w1", "svc", "old", v("old"), short))
	f.must(f.w.ReplaceCredential(admin(), "w1", "old", "new", v("new"), Validity{NotBefore: start, NotAfter: start.Add(50 * time.Hour)}, start.Add(24*time.Hour)))
	if got := f.credential(admin, "old"); got.Validity != short || got.ReplacedBy != "new" {
		t.Fatalf("an old credential with less time left than the overlap: %+v", got)
	}
}

func TestReplaceCredentialRefusalsChangeNothing(t *testing.T) {
	f, admin := replaceFixture(t, 2)
	day := Validity{NotBefore: start, NotAfter: start.Add(24 * time.Hour)}
	end := start.Add(time.Hour)
	f.must(f.w.AddCredential(admin(), "w1", "svc", "old", v("old"), day))
	f.must(f.w.CreateServiceUser(admin(), "w1", "other", "root"))
	f.must(f.w.AddCredential(admin(), "w1", "other", "taken", v("taken"), day))
	snapshot := func() string {
		a, _ := f.w.Credentials(admin(), "w1", "svc")
		b, _ := f.w.Credentials(admin(), "w1", "other")
		c, _ := f.w.Credentials(admin(), "w1", "leaving")
		return fmt.Sprintf("%+v %+v %+v", a, b, c)
	}
	was := snapshot()
	unchanged := func(name string) {
		t.Helper()
		if now := snapshot(); now != was {
			t.Fatalf("%s changed the credentials: %s; was %s", name, now, was)
		}
	}
	for name, err := range map[string]error{
		"an invalid workspace":               f.w.ReplaceCredential(admin(), "", "old", "new", v("new"), day, end),
		"an invalid old identifier":          f.w.ReplaceCredential(admin(), "w1", "", "new", v("new"), day, end),
		"an invalid new identifier":          f.w.ReplaceCredential(admin(), "w1", "old", "", v("new"), day, end),
		"an invalid verifier":                f.w.ReplaceCredential(admin(), "w1", "old", "new", Verifier{}, day, end),
		"a validity above the limit":         f.w.ReplaceCredential(admin(), "w1", "old", "new", v("new"), Validity{NotBefore: start, NotAfter: start.Add(720*time.Hour + 1)}, end),
		"a validity already over":            f.w.ReplaceCredential(admin(), "w1", "old", "new", v("new"), Validity{NotBefore: start.Add(-time.Hour), NotAfter: start}, end),
		"an old end at the present time":     f.w.ReplaceCredential(admin(), "w1", "old", "new", v("new"), day, start),
		"an old end before the present":      f.w.ReplaceCredential(admin(), "w1", "old", "new", v("new"), day, start.Add(-time.Second)),
		"a new credential that starts later": f.w.ReplaceCredential(admin(), "w1", "old", "new", v("new"), Validity{NotBefore: start.Add(time.Second), NotAfter: start.Add(time.Hour)}, end),
		"an old end that is the zero time":   f.w.ReplaceCredential(admin(), "w1", "old", "new", v("new"), day, time.Time{}),
	} {
		f.want(name, err, ErrInvalid)
		unchanged(name)
	}
	f.refusal("a member", f.w.ReplaceCredential(f.login("ann"), "w1", "old", "new", v("new"), day, end), ReasonNotAdministrator)
	f.refusal("another workspace", f.w.ReplaceCredential(admin(), "w2", "old", "new", v("new"), day, end), ReasonNotAdministrator)
	f.refusal("an unknown old credential", f.w.ReplaceCredential(admin(), "w1", "ghost", "new", v("new"), day, end), ReasonUnknown)
	f.seed("w2", "adm2")
	f.refusal("an administrator of another workspace naming this workspace's credential", f.w.ReplaceCredential(f.login("adm2"), "w2", "old", "new", v("new"), day, end), ReasonUnknown)
	f.refusal("the same administrator naming a credential that does not exist", f.w.ReplaceCredential(f.login("adm2"), "w2", "ghost", "new", v("new"), day, end), ReasonUnknown)
	f.want("a new identifier that exists, even on another account", f.w.ReplaceCredential(admin(), "w1", "old", "taken", v("new"), day, end), ErrConflict)
	unchanged("refusals by standing, target and identifier")

	f.must(f.w.AddCredential(admin(), "w1", "svc", "second", v("second"), day))
	was = snapshot()
	f.want("the limit, counted with the old credential", f.w.ReplaceCredential(admin(), "w1", "old", "new", v("new"), day, end), ErrLimit)
	unchanged("the limit")
	f.must(f.w.RevokeCredential(admin(), "w1", "second"))
	was = snapshot()

	f.must(f.w.SuspendServiceUser(admin(), "w1", "svc", 1))
	f.refusal("a suspended account", f.w.ReplaceCredential(admin(), "w1", "old", "new", v("new"), day, end), ReasonState)
	unchanged("a suspended account")
	f.must(f.w.ReinstateServiceUser(admin(), "w1", "svc", 2))

	f.must(f.w.CreateServiceUser(admin(), "w1", "leaving", "root"))
	f.must(f.w.AddCredential(admin(), "w1", "leaving", "kept", v("kept"), day))
	f.must(f.w.RetireServiceUser(admin(), "w1", "leaving", 1))
	if got := f.credential(admin, "kept"); got.Revoked || !got.Validity.at(*f.now) {
		t.Fatalf("the retired account's credential is not live in the module, so the next refusal would not be about the account: %+v", got)
	}
	was = snapshot()
	f.refusal("a live credential of a retired account", f.w.ReplaceCredential(admin(), "w1", "kept", "new", v("new"), day, end), ReasonState)
	unchanged("a retired account")
	f.must(f.w.RevokeCredential(admin(), "w1", "taken"))
	was = snapshot()
	f.refusal("a revoked old credential", f.w.ReplaceCredential(admin(), "w1", "taken", "new", v("new"), day, end), ReasonState)
	unchanged("a revoked old credential and a retired account")
	*f.now = start.Add(24 * time.Hour)
	later := Validity{NotBefore: *f.now, NotAfter: f.now.Add(time.Hour)}
	f.refusal("an expired old credential", f.w.ReplaceCredential(admin(), "w1", "old", "new", v("new"), later, f.now.Add(time.Minute)), ReasonState)
	f.must(f.w.AddCredential(admin(), "w1", "svc", "future", v("future"), Validity{NotBefore: f.now.Add(time.Hour), NotAfter: f.now.Add(2 * time.Hour)}))
	f.refusal("an old credential that has not started", f.w.ReplaceCredential(admin(), "w1", "future", "new", v("new"), later, f.now.Add(time.Minute)), ReasonState)
	if _, _, err := f.w.Credential(admin(), "w1", "new"); ReasonOf(err) != ReasonUnknown {
		t.Fatalf("a refused replacement left a new credential: %v", err)
	}
	was = snapshot()
	f.refusal("an old credential that has not started, again", f.w.ReplaceCredential(admin(), "w1", "future", "new", v("new"), later, f.now.Add(time.Minute)), ReasonState)
	unchanged("refusals for an expired and a not yet started credential")
}

func TestACredentialIsReplacedOnceAndItsReplacementCanBeReplaced(t *testing.T) {
	f, admin := replaceFixture(t, 3)
	long := Validity{NotBefore: start, NotAfter: start.Add(100 * time.Hour)}
	f.must(f.w.AddCredential(admin(), "w1", "svc", "a", v("a"), long))
	f.must(f.w.ReplaceCredential(admin(), "w1", "a", "b", v("b"), long, start.Add(10*time.Hour)))
	f.refusal("a second replacement of the same credential", f.w.ReplaceCredential(admin(), "w1", "a", "x", v("x"), long, start.Add(20*time.Hour)), ReasonState)
	f.must(f.w.ReplaceCredential(admin(), "w1", "b", "c", v("c"), long, start.Add(5*time.Hour)))
	a, b, c := f.credential(admin, "a"), f.credential(admin, "b"), f.credential(admin, "c")
	if a.Validity.NotAfter != start.Add(10*time.Hour) || a.ReplacedBy != "b" || b.Replaces != "a" || b.ReplacedBy != "c" || b.Validity.NotAfter != start.Add(5*time.Hour) || c.Replaces != "b" || c.ReplacedBy != "" {
		t.Fatalf("chain %+v %+v %+v", a, b, c)
	}
	f.want("a fourth live credential by replacement", f.w.ReplaceCredential(admin(), "w1", "c", "d", v("d"), long, start.Add(time.Hour)), ErrLimit)
	f.must(f.w.RevokeCredential(admin(), "w1", "c"))
	if got := f.credential(admin, "b"); got.Validity.NotAfter != start.Add(5*time.Hour) || got.Revoked {
		t.Fatalf("revoking the replacement restored or ended the replaced credential: %+v", got)
	}
	f.must(f.w.RevokeCredential(admin(), "w1", "a"))
	if got := f.credential(admin, "b"); got.Revoked || got.Replaces != "a" {
		t.Fatalf("revoking the replaced credential touched its replacement: %+v", got)
	}
}

func TestPruningKeepsAReplacedCredentialWhileItsReplacementIsLiveAndOmitsADanglingLink(t *testing.T) {
	f, admin := replaceFixture(t, 3)
	long := Validity{NotBefore: start, NotAfter: start.Add(100 * time.Hour)}
	f.must(f.w.AddCredential(admin(), "w1", "svc", "a", v("a"), Validity{NotBefore: start, NotAfter: start.Add(30 * time.Minute)}))
	f.must(f.w.ReplaceCredential(admin(), "w1", "a", "b", v("b"), long, start.Add(10*time.Minute)))
	*f.now = start.Add(time.Hour)
	for i := 0; i < EndedCredentialsKept+3; i++ {
		id := fmt.Sprintf("filler-%02d", i)
		f.must(f.w.AddCredential(admin(), "w1", "svc", id, v(id), Validity{NotBefore: *f.now, NotAfter: f.now.Add(2 * time.Hour)}))
		f.must(f.w.RevokeCredential(admin(), "w1", id))
	}
	list, err := f.w.Credentials(admin(), "w1", "svc")
	f.must(err)
	if len(list) != EndedCredentialsKept+2 {
		t.Fatalf("%d credentials kept; want the bound, the replaced one and its live replacement", len(list))
	}
	if got := f.credential(admin, "a"); got.ReplacedBy != "b" {
		t.Fatalf("the replaced credential, ended earlier than every kept one, while its replacement is live: %+v", got)
	}
	f.must(f.w.RevokeCredential(admin(), "w1", "b"))
	f.must(f.w.AddCredential(admin(), "w1", "svc", "trigger", v("trigger"), Validity{NotBefore: *f.now, NotAfter: f.now.Add(3 * time.Hour)}))
	if _, _, err := f.w.Credential(admin(), "w1", "a"); ReasonOf(err) != ReasonUnknown {
		t.Fatalf("the replaced credential after its replacement ended: %v", err)
	}
	list, err = f.w.Credentials(admin(), "w1", "svc")
	f.must(err)
	ended := 0
	for _, c := range list {
		if c.Revoked {
			ended++
		}
		if c.ID == "b" && c.Replaces != "" {
			t.Fatalf("the replacement still names a credential that no longer exists: %+v", c)
		}
	}
	if ended != EndedCredentialsKept {
		t.Fatalf("%d ended credentials kept; want %d", ended, EndedCredentialsKept)
	}
}

func TestNoOperationExtendsACredentialsEnd(t *testing.T) {
	replaced := 0
	for seed := int64(1); seed <= 40; seed++ {
		random := rand.New(rand.NewSource(seed))
		f, admin := replaceFixture(t, 4)
		ends := map[string]time.Time{}
		next := 0
		for step := 0; step < 60; step++ {
			*f.now = f.now.Add(time.Duration(random.Intn(90)) * time.Minute)
			live := []string{}
			list, err := f.w.Credentials(admin(), "w1", "svc")
			f.must(err)
			for _, c := range list {
				if was, known := ends[c.ID]; known && c.Validity.NotAfter.After(was) {
					t.Fatalf("seed %d step %d: the end of %s moved from %s to %s", seed, step, c.ID, was, c.Validity.NotAfter)
				}
				ends[c.ID] = c.Validity.NotAfter
				if !c.Revoked && c.Validity.at(*f.now) {
					live = append(live, c.ID)
				}
			}
			id := fmt.Sprintf("c%d", next)
			next++
			window := Validity{NotBefore: *f.now, NotAfter: f.now.Add(time.Duration(1+random.Intn(200)) * time.Hour)}
			switch pick := random.Intn(4); {
			case pick == 0 || len(live) == 0:
				_ = f.w.AddCredential(admin(), "w1", "svc", id, v(id), window)
			case pick == 1:
				_ = f.w.RevokeCredential(admin(), "w1", live[random.Intn(len(live))])
			default:
				if f.w.ReplaceCredential(admin(), "w1", live[random.Intn(len(live))], id, v(id), window, f.now.Add(time.Duration(1+random.Intn(400))*time.Hour)) == nil {
					replaced++
				}
			}
		}
		list, err := f.w.Credentials(admin(), "w1", "svc")
		f.must(err)
		for _, c := range list {
			if was, known := ends[c.ID]; known && c.Validity.NotAfter.After(was) {
				t.Fatalf("seed %d: after the last step the end of %s moved from %s to %s", seed, c.ID, was, c.Validity.NotAfter)
			}
		}
	}
	if replaced < 100 {
		t.Fatalf("only %d replacements succeeded over all seeds; the property was hardly exercised", replaced)
	}
}

func TestAReplacedCredentialWhoseReplacementWasPrunedNamesNoneAndStaysReplaced(t *testing.T) {
	f, admin := replaceFixture(t, 3)
	long := Validity{NotBefore: start, NotAfter: start.Add(100 * time.Hour)}
	f.must(f.w.AddCredential(admin(), "w1", "svc", "a", v("a"), long))
	f.must(f.w.ReplaceCredential(admin(), "w1", "a", "b", v("b"), Validity{NotBefore: start, NotAfter: start.Add(time.Hour)}, start.Add(50*time.Hour)))
	f.must(f.w.RevokeCredential(admin(), "w1", "b"))
	for i := 0; i < EndedCredentialsKept+1; i++ {
		id := fmt.Sprintf("filler-%02d", i)
		f.must(f.w.AddCredential(admin(), "w1", "svc", id, v(id), Validity{NotBefore: start, NotAfter: start.Add(2 * time.Hour)}))
		f.must(f.w.RevokeCredential(admin(), "w1", id))
	}
	if _, _, err := f.w.Credential(admin(), "w1", "b"); ReasonOf(err) != ReasonUnknown {
		t.Fatalf("the revoked replacement was not pruned: %v", err)
	}
	if got := f.credential(admin, "a"); got.ReplacedBy != "" || got.Validity.NotAfter != start.Add(50*time.Hour) {
		t.Fatalf("a replaced credential whose replacement no longer exists: %+v", got)
	}
	f.refusal("replacing it again", f.w.ReplaceCredential(admin(), "w1", "a", "c", v("c"), long, start.Add(60*time.Hour)), ReasonState)
}

func TestAnIdentifierReusedAfterPruningIsNotTakenForTheOtherHalfOfAPair(t *testing.T) {
	f, admin := replaceFixture(t, 3)
	long := Validity{NotBefore: start, NotAfter: start.Add(100 * time.Hour)}
	f.must(f.w.AddCredential(admin(), "w1", "svc", "a", v("a"), long))
	f.must(f.w.ReplaceCredential(admin(), "w1", "a", "b", v("b"), Validity{NotBefore: start, NotAfter: start.Add(time.Hour)}, start.Add(50*time.Hour)))
	f.must(f.w.RevokeCredential(admin(), "w1", "b"))
	for i := 0; i < EndedCredentialsKept+1; i++ {
		id := fmt.Sprintf("filler-%02d", i)
		f.must(f.w.AddCredential(admin(), "w1", "svc", id, v(id), Validity{NotBefore: start, NotAfter: start.Add(2 * time.Hour)}))
		f.must(f.w.RevokeCredential(admin(), "w1", id))
	}
	f.must(f.w.AddCredential(admin(), "w1", "svc", "b", v("b2"), long))
	if got := f.credential(admin, "a"); got.ReplacedBy != "" {
		t.Fatalf("a replaced credential names an unrelated credential that reuses its replacement's identifier: %+v", got)
	}
	if got := f.credential(admin, "b"); got.Replaces != "" {
		t.Fatalf("an unrelated credential that reuses an identifier names a credential it did not replace: %+v", got)
	}
	f.must(f.w.RevokeCredential(admin(), "w1", "a"))
	for i := 0; i < EndedCredentialsKept+1; i++ {
		id := fmt.Sprintf("later-%02d", i)
		f.must(f.w.AddCredential(admin(), "w1", "svc", id, v(id), Validity{NotBefore: start, NotAfter: start.Add(60 * time.Hour)}))
		f.must(f.w.RevokeCredential(admin(), "w1", id))
	}
	if _, _, err := f.w.Credential(admin(), "w1", "a"); ReasonOf(err) != ReasonUnknown {
		t.Fatalf("an ended credential is kept from pruning by a live unrelated credential that reuses its replacement's identifier: %v", err)
	}
}

func TestAReplacementDoesNotNameAnUnrelatedCredentialThatReusesTheReplacedIdentifier(t *testing.T) {
	f, admin := replaceFixture(t, 3)
	long := Validity{NotBefore: start, NotAfter: start.Add(100 * time.Hour)}
	f.must(f.w.AddCredential(admin(), "w1", "svc", "a", v("a"), long))
	f.must(f.w.ReplaceCredential(admin(), "w1", "a", "b", v("b"), long, start.Add(30*time.Minute)))
	f.must(f.w.RevokeCredential(admin(), "w1", "b"))
	f.must(f.w.RevokeCredential(admin(), "w1", "a"))
	for i := 0; i < EndedCredentialsKept; i++ {
		id := fmt.Sprintf("filler-%02d", i)
		f.must(f.w.AddCredential(admin(), "w1", "svc", id, v(id), Validity{NotBefore: start, NotAfter: start.Add(2 * time.Hour)}))
		f.must(f.w.RevokeCredential(admin(), "w1", id))
	}
	if _, _, err := f.w.Credential(admin(), "w1", "a"); ReasonOf(err) != ReasonUnknown {
		t.Fatalf("the replaced credential was not pruned: %v", err)
	}
	if got := f.credential(admin, "b"); got.Replaces != "" {
		t.Fatalf("a replacement names a credential that no longer exists: %+v", got)
	}
	f.must(f.w.AddCredential(admin(), "w1", "svc", "a", v("a2"), long))
	if got := f.credential(admin, "b"); got.Replaces != "" {
		t.Fatalf("a replacement names an unrelated credential that reuses the replaced identifier: %+v", got)
	}
}
