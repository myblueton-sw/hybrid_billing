package identity

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

type fixture struct {
	r     *Registry
	in    *Installation
	b     *Boundary
	w     *Workspace
	now   *time.Time
	t     testing.TB
	count *int
}

var (
	start      = time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	testLimits = Limits{BootstrapValidity: time.Hour, InvitationValidity: 72 * time.Hour, CredentialValidity: 720 * time.Hour,
		SessionIdle: 30 * time.Minute, SessionAbsolute: 8 * time.Hour, Reauthentication: 5 * time.Minute,
		Attempts: 3, PendingInvitations: 3, SessionsPerUser: 1000, CredentialsPerServiceUser: 2, PageSize: 2}
	operator = Operator{Reference: "synthetic-operator", Reason: "synthetic-reason"}
)

func v(label string) Verifier { return Verifier(sha256.Sum256([]byte("synthetic-secret-" + label))) }

func triple(subject string) Triple {
	return Triple{Connection: "synthetic-connection", Issuer: "https://issuer.invalid", Subject: subject}
}

func empty(t testing.TB, limits Limits) fixture {
	t.Helper()
	now := start
	r, err := New("synthetic-installation", limits, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	return fixture{r: r, in: r.Installation(), b: r.Boundary(), w: r.Workspace(), now: &now, t: t, count: &count}
}

func setupWith(t testing.TB, limits Limits) fixture {
	t.Helper()
	f := empty(t, limits)
	f.must(f.in.ArmBootstrap(v("boot"), Validity{NotBefore: start, NotAfter: start.Add(time.Hour)}, "w1"))
	f.must(f.b.Bootstrap(v("boot"), "root", triple("root"), "root@example.invalid"))
	return f
}

func setup(t testing.TB) fixture { return setupWith(t, testLimits) }

func (f fixture) must(err error) {
	f.t.Helper()
	if err != nil {
		f.t.Fatal(err)
	}
}

func (f fixture) want(name string, err, want error) {
	f.t.Helper()
	if want == nil && err != nil || want != nil && !errors.Is(err, want) {
		f.t.Errorf("%s: error %v; want %v", name, err, want)
	}
}

func (f fixture) refusal(name string, err error, reason Reason) {
	f.t.Helper()
	if !errors.Is(err, ErrRefused) || ReasonOf(err) != reason {
		f.t.Errorf("%s: error %v reason %q; want refusal %q", name, err, ReasonOf(err), reason)
	}
	if err != nil && err.Error() != ErrRefused.Error() {
		f.t.Errorf("%s: a refusal shows more than the common text: %q", name, err.Error())
	}
}

func (f fixture) next(label string) string {
	*f.count++
	return fmt.Sprintf("%s-%d", label, *f.count)
}

func (f fixture) user(id string) {
	f.t.Helper()
	f.must(f.in.CreateUser(operator, id, triple(id), id+"@example.invalid"))
}

func (f fixture) login(id string) Verifier {
	f.t.Helper()
	verifier := v(f.next("session-" + id))
	f.must(f.b.OpenSession(triple(id), verifier, 30*time.Minute, 8*time.Hour))
	return verifier
}

func (f fixture) window() Validity {
	return Validity{NotBefore: *f.now, NotAfter: f.now.Add(48 * time.Hour)}
}

func (f fixture) join(id string, role Role) {
	f.t.Helper()
	invitation := f.next("invitation")
	f.must(f.w.Invite(f.login("root"), "w1", invitation, role, v(invitation), f.window(), id))
	f.must(f.w.Accept("w1", invitation, v(invitation), f.login(id)))
}

func (f fixture) member(id string, role Role) {
	f.t.Helper()
	f.user(id)
	f.join(id, role)
}

func (f fixture) epoch(id string) uint64 {
	f.t.Helper()
	for after := ""; ; {
		users, err := f.in.Users(after, 2)
		f.must(err)
		for _, found := range users {
			if found.ID == id {
				return found.Epoch
			}
		}
		if len(users) < 2 {
			f.t.Fatalf("user %s is not listed", id)
		}
		after = users[len(users)-1].ID
	}
}

func (f fixture) version(id string) uint64 {
	f.t.Helper()
	admin := f.login("root")
	for after := ""; ; {
		members, err := f.w.Members(admin, "w1", after, 2)
		f.must(err)
		for _, found := range members {
			if found.User == id {
				return found.Version
			}
		}
		if len(members) < 2 {
			f.t.Fatalf("member %s is not listed", id)
		}
		after = members[len(members)-1].User
	}
}

func TestIdentifiersVerifiersAndLimits(t *testing.T) {
	for id, valid := range map[string]bool{"a": true, "A.b_c:d-9": true, strings.Repeat("x", 128): true, strings.Repeat("x", 129): false, "": false, "a b": false, "a/b": false, "a\x00": false} {
		if validID(id) != valid {
			t.Errorf("identifier %q valid = %v", id, !valid)
		}
	}
	if (Verifier{}).valid() || Verifier(sha256.Sum256(nil)).valid() || !v("x").valid() {
		t.Errorf("verifier validity")
	}
	long := strings.Repeat("s", 513)
	for name, bad := range map[string]Triple{"no connection": {"", "i", "s"}, "no issuer": {"c", "", "s"}, "no subject": {"c", "i", ""}, "long subject": {"c", "i", long}} {
		if bad.valid() {
			t.Errorf("triple with %s is valid", name)
		}
	}
	fields := reflect.TypeOf(testLimits).NumField()
	for index := 0; index < fields; index++ {
		limits := testLimits
		reflect.ValueOf(&limits).Elem().Field(index).SetInt(0)
		if _, err := New("i", limits, time.Now); !errors.Is(err, ErrInvalid) {
			t.Errorf("limit %s zero: error %v", reflect.TypeOf(limits).Field(index).Name, err)
		}
	}
	if _, err := New("i", testLimits, nil); !errors.Is(err, ErrInvalid) {
		t.Errorf("missing clock: %v", err)
	}
	if _, err := New("bad id", testLimits, time.Now); !errors.Is(err, ErrInvalid) {
		t.Errorf("invalid installation: %v", err)
	}
}

func TestBootstrap(t *testing.T) {
	hour := Validity{NotBefore: start, NotAfter: start.Add(time.Hour)}
	f := empty(t, testLimits)
	f.refusal("before arming", f.b.Bootstrap(v("boot"), "root", triple("root"), ""), ReasonBootstrap)
	f.refusal("creating a user before the bootstrap", f.in.CreateUser(operator, "early", triple("early"), ""), ReasonBootstrap)
	for name, err := range map[string]error{
		"an invalid verifier":                f.in.ArmBootstrap(Verifier{}, hour, "w1"),
		"the empty-secret verifier":          f.in.ArmBootstrap(Verifier(sha256.Sum256(nil)), hour, "w1"),
		"an invalid workspace":               f.in.ArmBootstrap(v("boot"), hour, ""),
		"a validity above the limit":         f.in.ArmBootstrap(v("boot"), Validity{NotBefore: start, NotAfter: start.Add(time.Hour + time.Nanosecond)}, "w1"),
		"an inverted validity":               f.in.ArmBootstrap(v("boot"), Validity{NotBefore: start, NotAfter: start}, "w1"),
		"a validity without start":           f.in.ArmBootstrap(v("boot"), Validity{NotAfter: start.Add(time.Minute)}, "w1"),
		"a validity already over":            f.in.ArmBootstrap(v("boot"), Validity{NotBefore: start.Add(-time.Hour), NotAfter: start}, "w1"),
		"an inverted validity in the future": f.in.ArmBootstrap(v("boot"), Validity{NotBefore: start.Add(2 * time.Hour), NotAfter: start.Add(time.Hour)}, "w1"),
	} {
		f.want("arming with "+name, err, ErrInvalid)
	}
	f.must(f.in.ArmBootstrap(v("boot"), hour, "w1"))
	f.refusal("arming twice", f.in.ArmBootstrap(v("other"), hour, "w2"), ReasonBootstrap)
	for name, err := range map[string]error{
		"an invalid user":     f.b.Bootstrap(v("boot"), "", triple("root"), ""),
		"an invalid triple":   f.b.Bootstrap(v("boot"), "root", Triple{}, ""),
		"an invalid verifier": f.b.Bootstrap(Verifier{}, "root", triple("root"), ""),
		"a long email":        f.b.Bootstrap(v("boot"), "root", triple("root"), strings.Repeat("e", 321)),
	} {
		f.want("bootstrap with "+name, err, ErrInvalid)
	}
	f.refusal("a wrong verifier", f.b.Bootstrap(v("guess-1"), "root", triple("root"), ""), ReasonVerifier)
	f.refusal("a second wrong verifier", f.b.Bootstrap(v("guess-2"), "root", triple("root"), ""), ReasonVerifier)
	f.must(f.b.Bootstrap(v("boot"), "root", triple("root"), "root@example.invalid"))
	f.refusal("a second bootstrap", f.b.Bootstrap(v("boot"), "second", triple("second"), ""), ReasonBootstrap)
	f.refusal("arming after consumption", f.in.ArmBootstrap(v("again"), hour, "w1"), ReasonBootstrap)
	if answer, err := f.b.Access(f.login("root"), "w1"); err != nil || answer.Role() != Administrator || answer.User() != "root" || answer.MembershipVersion() != 1 {
		t.Errorf("the first administrator: %+v %v", answer, err)
	}
	if _, err := f.b.Access(f.login("root"), "w2"); ReasonOf(err) != ReasonMembership {
		t.Errorf("the first administrator in another workspace: %v", err)
	}

	locked := empty(t, testLimits)
	locked.must(locked.in.ArmBootstrap(v("boot"), hour, "w1"))
	for attempt := 1; attempt <= 3; attempt++ {
		locked.refusal("a wrong verifier", locked.b.Bootstrap(v("guess"), "root", triple("root"), ""), ReasonVerifier)
	}
	locked.refusal("the right verifier after the lock", locked.b.Bootstrap(v("boot"), "root", triple("root"), ""), ReasonBootstrap)

	last := empty(t, testLimits)
	last.must(last.in.ArmBootstrap(v("boot"), hour, "w1"))
	last.refusal("first wrong", last.b.Bootstrap(v("guess"), "root", triple("root"), ""), ReasonVerifier)
	last.refusal("second wrong", last.b.Bootstrap(v("guess"), "root", triple("root"), ""), ReasonVerifier)
	last.must(last.b.Bootstrap(v("boot"), "root", triple("root"), ""))

	expired := empty(t, testLimits)
	expired.must(expired.in.ArmBootstrap(v("boot"), Validity{NotBefore: start.Add(time.Minute), NotAfter: start.Add(time.Hour)}, "w1"))
	expired.refusal("before the validity starts", expired.b.Bootstrap(v("boot"), "root", triple("root"), ""), ReasonExpired)
	*expired.now = start.Add(time.Hour)
	expired.refusal("exactly at the end", expired.b.Bootstrap(v("boot"), "root", triple("root"), ""), ReasonExpired)
	*expired.now = start.Add(time.Hour - time.Nanosecond)
	expired.must(expired.b.Bootstrap(v("boot"), "root", triple("root"), ""))

	race := empty(t, testLimits)
	race.must(race.in.ArmBootstrap(v("boot"), hour, "w1"))
	var wg sync.WaitGroup
	results := make(chan error, 8)
	for index := 0; index < 8; index++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id := fmt.Sprintf("root-%d", index)
			results <- race.b.Bootstrap(v("boot"), id, triple(id), "")
		}()
	}
	wg.Wait()
	close(results)
	succeeded := 0
	for err := range results {
		if err == nil {
			succeeded++
		} else if ReasonOf(err) != ReasonBootstrap {
			t.Errorf("a losing concurrent bootstrap: %v", err)
		}
	}
	users, _ := race.in.Users("", 2)
	if succeeded != 1 || len(users) != 1 {
		t.Errorf("%d concurrent bootstraps succeeded and %d users exist", succeeded, len(users))
	}
}

func TestUserStatesAndEpochs(t *testing.T) {
	f := setup(t)
	f.member("ann", Member)
	session := f.login("ann")
	answer, err := f.b.Access(session, "w1")
	f.must(err)
	for name, err := range map[string]error{
		"no operator reference":      f.in.SuspendUser(Operator{Reason: "r"}, "ann", 1),
		"no operator reason":         f.in.SuspendUser(Operator{Reference: "o"}, "ann", 1),
		"an invalid user":            f.in.SuspendUser(operator, "", 1),
		"creation without operator":  f.in.CreateUser(Operator{}, "zed", triple("zed"), ""),
		"a long operator reference":  f.in.SuspendUser(Operator{Reference: strings.Repeat("o", 257), Reason: "r"}, "ann", 1),
		"a long operator reason":     f.in.SuspendUser(Operator{Reference: "o", Reason: strings.Repeat("r", 257)}, "ann", 1),
		"creation with a long email": f.in.CreateUser(operator, "zed", triple("zed"), strings.Repeat("e", 321)),
	} {
		f.want(name, err, ErrInvalid)
	}
	f.refusal("an unknown user", f.in.SuspendUser(operator, "ghost", 1), ReasonUnknown)
	f.want("a stale epoch", f.in.SuspendUser(operator, "ann", 2), ErrConflict)
	f.want("reinstating an active user", f.in.ReinstateUser(operator, "ann", 1), ErrConflict)
	f.must(f.in.SuspendUser(operator, "ann", 1))
	f.want("suspending twice with the old epoch", f.in.SuspendUser(operator, "ann", 1), ErrConflict)
	f.want("suspending twice with the new epoch", f.in.SuspendUser(operator, "ann", 2), ErrConflict)
	if got := f.epoch("ann"); got != 2 {
		t.Errorf("epoch after suspension %d", got)
	}
	f.refusal("access during suspension", second(f.b.Access(session, "w1")), ReasonSession)
	f.refusal("recheck during suspension", f.b.Recheck(answer), ReasonAnswer)
	f.refusal("login during suspension", f.b.OpenSession(triple("ann"), v("during"), time.Minute, time.Hour), ReasonUser)
	f.want("a stale reinstatement", f.in.ReinstateUser(operator, "ann", 1), ErrConflict)
	f.must(f.in.ReinstateUser(operator, "ann", 2))
	f.refusal("the old session after reinstatement", second(f.b.Access(session, "w1")), ReasonSession)
	f.refusal("the old answer after reinstatement", f.b.Recheck(answer), ReasonAnswer)
	fresh := f.login("ann")
	renewed, err := f.b.Access(fresh, "w1")
	if err != nil || renewed.Epoch() != 3 {
		t.Fatalf("a new session after reinstatement: %+v %v", renewed, err)
	}
	f.want("a stale departure", f.in.DepartUser(operator, "ann", 2), ErrConflict)
	f.must(f.in.DepartUser(operator, "ann", 3))
	f.refusal("the session of a departed user", second(f.b.Access(fresh, "w1")), ReasonSession)
	f.refusal("the answer of a departed user", f.b.Recheck(renewed), ReasonAnswer)
	f.refusal("the identity of a departed user", second(f.b.Resolve(triple("ann"))), ReasonIdentity)
	f.refusal("login of a departed user", f.b.OpenSession(triple("ann"), v("after"), time.Minute, time.Hour), ReasonIdentity)
	for name, err := range map[string]error{
		"reinstating a departed user": f.in.ReinstateUser(operator, "ann", 4),
		"suspending a departed user":  f.in.SuspendUser(operator, "ann", 4),
		"departing twice":             f.in.DepartUser(operator, "ann", 4),
		"reusing the identifier":      f.in.CreateUser(operator, "ann", triple("ann-2"), ""),
		"reusing the triple":          f.in.CreateUser(operator, "ann-2", triple("ann"), ""),
	} {
		f.want(name, err, ErrConflict)
	}
	members, _ := f.w.Members(f.login("root"), "w1", "", 2)
	if len(members) != 1 || members[0].User != "root" {
		t.Errorf("members after the departure %+v", members)
	}
	invitation := f.next("invitation")
	f.must(f.w.Invite(f.login("root"), "w1", invitation, Member, v(invitation), f.window(), "ann"))
	if users, _ := f.in.Users("", 2); len(users) != 2 || users[0] != (UserInfo{ID: "ann", Kind: Human, State: Departed, Epoch: 4}) {
		t.Errorf("users %+v", users)
	}
}

func second[T any](_ T, err error) error { return err }

func TestIdentitiesAreNeverFoundByEmail(t *testing.T) {
	f := setup(t)
	f.user("ann")
	resolved, err := f.b.Resolve(triple("ann"))
	if err != nil || resolved != (Resolved{User: "ann", Email: "ann@example.invalid"}) {
		t.Fatalf("resolve %+v %v", resolved, err)
	}
	f.refusal("an unknown subject", second(f.b.Resolve(triple("ann-reassigned"))), ReasonIdentity)
	f.refusal("login with an unknown subject", f.b.OpenSession(triple("ann-reassigned"), v("s"), time.Minute, time.Hour), ReasonIdentity)
	f.must(f.in.CreateUser(operator, "newcomer", triple("ann-reassigned"), "ann@example.invalid"))
	resolved, _ = f.b.Resolve(triple("ann-reassigned"))
	if resolved.User != "newcomer" {
		t.Errorf("a new subject with a known email resolves to %q", resolved.User)
	}
	other := triple("ann")
	other.Connection = "another-connection"
	f.refusal("the same issuer and subject through another connection", second(f.b.Resolve(other)), ReasonIdentity)
	f.want("an invalid triple", second(f.b.Resolve(Triple{})), ErrInvalid)
}

func TestLinkingAndUnlinking(t *testing.T) {
	f := setup(t)
	f.member("ann", Member)
	f.user("bob")
	session := f.login("ann")
	answer, err := f.b.Access(session, "w1")
	f.must(err)
	added, third := triple("ann-second"), triple("ann-third")
	f.want("no proof", f.b.LinkIdentity(session, added, "", ""), ErrInvalid)
	f.want("a long proof", f.b.LinkIdentity(session, added, "", strings.Repeat("p", 257)), ErrInvalid)
	f.want("a long email", f.b.LinkIdentity(session, added, strings.Repeat("e", 321), "proof"), ErrInvalid)
	ended := f.login("ann")
	f.must(f.b.EndSession(ended))
	f.refusal("an ended session", f.b.LinkIdentity(ended, added, "", "proof"), ReasonSession)
	f.want("an invalid triple", f.b.LinkIdentity(session, Triple{}, "", "proof"), ErrInvalid)
	f.refusal("a dead session", f.b.LinkIdentity(v("nothing"), added, "", "proof"), ReasonSession)
	f.want("a triple of another user", f.b.LinkIdentity(session, triple("bob"), "", "proof"), ErrConflict)
	*f.now = start.Add(5*time.Minute + time.Nanosecond)
	f.refusal("a session older than the window", f.b.LinkIdentity(session, added, "", "proof"), ReasonReauthentication)
	*f.now = start.Add(5 * time.Minute)
	f.must(f.b.LinkIdentity(session, added, "second@example.invalid", "proof"))
	if got := f.epoch("ann"); got != 2 {
		t.Errorf("epoch after linking %d", got)
	}
	f.refusal("the session used for linking", f.b.Recheck(answer), ReasonAnswer)
	f.refusal("the session used for linking, as an actor", f.b.LinkIdentity(session, third, "", "proof"), ReasonSession)
	if resolved, _ := f.b.Resolve(added); resolved.User != "ann" {
		t.Errorf("the linked identity resolves to %q", resolved.User)
	}
	viaSecond := v("via-second")
	f.must(f.b.OpenSession(added, viaSecond, time.Minute, time.Hour))
	f.refusal("unlinking an older identity through a later one", f.b.UnlinkIdentity(viaSecond, triple("ann")), ReasonLaterIdentity)
	f.refusal("unlinking another user's identity", f.b.UnlinkIdentity(viaSecond, triple("bob")), ReasonIdentity)
	viaFirst := f.login("ann")
	f.must(f.b.UnlinkIdentity(viaFirst, added))
	if got := f.epoch("ann"); got != 3 {
		t.Errorf("epoch after unlinking %d", got)
	}
	f.refusal("the unlinked identity", second(f.b.Resolve(added)), ReasonIdentity)
	f.refusal("a session of the unlinked identity", second(f.b.Access(viaSecond, "w1")), ReasonSession)
	f.want("linking the same triple again", f.b.LinkIdentity(f.login("ann"), added, "", "proof"), ErrConflict)
	f.refusal("unlinking the last identity", f.b.UnlinkIdentity(f.login("ann"), triple("ann")), ReasonLastIdentity)
	f.refusal("unlinking an unlinked identity", f.b.UnlinkIdentity(f.login("ann"), added), ReasonIdentity)
	own := f.login("ann")
	f.must(f.b.LinkIdentity(own, third, "", "proof"))
	viaThird := v("via-third")
	f.must(f.b.OpenSession(third, viaThird, time.Minute, time.Hour))
	f.must(f.b.UnlinkIdentity(viaThird, third))
	f.refusal("an identity that unlinked itself", second(f.b.Resolve(third)), ReasonIdentity)
}

func TestSessionPredicateAndIdleRenewal(t *testing.T) {
	f := setup(t)
	f.member("ann", Member)
	for name, err := range map[string]error{
		"a zero idle length":         f.b.OpenSession(triple("ann"), v("a"), 0, time.Hour),
		"a zero absolute length":     f.b.OpenSession(triple("ann"), v("a"), time.Minute, 0),
		"idle above absolute":        f.b.OpenSession(triple("ann"), v("a"), 20*time.Minute, 10*time.Minute),
		"idle above its maximum":     f.b.OpenSession(triple("ann"), v("a"), 30*time.Minute+1, 8*time.Hour),
		"absolute above its maximum": f.b.OpenSession(triple("ann"), v("a"), time.Minute, 8*time.Hour+1),
		"an invalid verifier":        f.b.OpenSession(triple("ann"), Verifier{}, time.Minute, time.Hour),
	} {
		f.want(name, err, ErrInvalid)
	}
	session := v("ann-session")
	f.must(f.b.OpenSession(triple("ann"), session, 10*time.Minute, 25*time.Minute))
	f.want("a verifier that names a session already", f.b.OpenSession(triple("ann"), session, time.Minute, time.Hour), ErrConflict)
	f.want("an invalid workspace", second(f.b.Access(session, "")), ErrInvalid)
	f.refusal("an unknown verifier", second(f.b.Access(v("unknown"), "w1")), ReasonSession)
	f.refusal("the zero verifier", second(f.b.Access(Verifier{}, "w1")), ReasonSession)
	f.refusal("another workspace", second(f.b.Access(session, "w2")), ReasonMembership)
	at := func(offset time.Duration) error {
		*f.now = start.Add(offset)
		return second(f.b.Access(session, "w1"))
	}
	f.must(at(0))
	f.refusal("a clock before the opening", at(-time.Second), ReasonSession)
	f.must(at(10*time.Minute - time.Nanosecond))
	f.must(at(18 * time.Minute))
	f.refusal("the refused access above did not extend the idle time", at(28*time.Minute), ReasonSession)
	other := v("ann-other")
	*f.now = start
	f.must(f.b.OpenSession(triple("ann"), other, 10*time.Minute, 25*time.Minute))
	step := func(offset time.Duration) error {
		*f.now = start.Add(offset)
		return second(f.b.Access(other, "w1"))
	}
	f.must(step(9 * time.Minute))
	f.must(step(5 * time.Minute))
	f.must(step(18 * time.Minute))
	f.refusal("an earlier access moved the idle expiry backwards", step(28*time.Minute), ReasonSession)
	*f.now = start
	third := v("ann-third")
	f.must(f.b.OpenSession(triple("ann"), third, 10*time.Minute, 25*time.Minute))
	renew := func(offset time.Duration) error {
		*f.now = start.Add(offset)
		return second(f.b.Access(third, "w1"))
	}
	f.must(renew(9 * time.Minute))
	f.must(renew(18 * time.Minute))
	f.must(renew(25*time.Minute - time.Nanosecond))
	f.refusal("exactly at the absolute expiry", renew(25*time.Minute), ReasonSession)
	*f.now = start
	exact := v("ann-exact")
	f.must(f.b.OpenSession(triple("ann"), exact, 10*time.Minute, 25*time.Minute))
	*f.now = start.Add(10 * time.Minute)
	f.refusal("exactly at the idle expiry", second(f.b.Access(exact, "w1")), ReasonSession)
	*f.now = start
	ended := f.login("ann")
	f.must(f.b.EndSession(ended))
	f.refusal("an ended session", second(f.b.Access(ended, "w1")), ReasonSession)
	f.refusal("ending twice", f.b.EndSession(ended), ReasonSession)
	f.must(f.b.OpenSession(triple("root"), ended, time.Minute, time.Hour))
	if answer, err := f.b.Access(ended, "w1"); err != nil || answer.User() != "root" {
		t.Errorf("a verifier whose record was deleted names only the new session: %+v %v", answer, err)
	}
	f.must(f.b.EndSession(ended))
	f.refusal("ending an unknown session", f.b.EndSession(v("unknown")), ReasonSession)
}

func TestSessionLimitCountsCurrentSessionsOnly(t *testing.T) {
	limits := testLimits
	limits.SessionsPerUser = 2
	f := setupWith(t, limits)
	first, second := f.login("root"), f.login("root")
	f.want("a third session", f.b.OpenSession(triple("root"), v("third"), time.Minute, time.Hour), ErrLimit)
	f.must(f.b.EndSession(first))
	third := f.login("root")
	f.want("a session beyond the limit again", f.b.OpenSession(triple("root"), v("fourth"), time.Minute, time.Hour), ErrLimit)
	*f.now = start.Add(31 * time.Minute)
	f.must(f.b.OpenSession(triple("root"), v("after-idle"), time.Minute, time.Hour))
	*f.now = start
	f.user("ann")
	one, two := f.login("ann"), f.login("ann")
	f.must(f.in.SuspendUser(operator, "ann", 1))
	f.must(f.in.ReinstateUser(operator, "ann", 2))
	f.must(f.b.OpenSession(triple("ann"), v("ann-new-1"), time.Minute, time.Hour))
	f.must(f.b.OpenSession(triple("ann"), v("ann-new-2"), time.Minute, time.Hour))
	f.want("the limit after reinstatement", f.b.OpenSession(triple("ann"), v("ann-new-3"), time.Minute, time.Hour), ErrLimit)
	*f.now = start.Add(9 * time.Hour)
	f.must(f.b.OpenSession(triple("ann"), v("ann-late"), time.Minute, time.Hour))
	if kept := len(f.r.byUser["ann"]); kept != 1 {
		t.Errorf("%d session records of the user are kept after their absolute expiry", kept)
	}
	_, _, _, _ = second, third, one, two
}

func TestRecheck(t *testing.T) {
	type world struct {
		fixture
		session Verifier
		answer  Answer
	}
	build := func(t *testing.T) world {
		f := setup(t)
		f.member("ann", Member)
		session := f.login("ann")
		answer, err := f.b.Access(session, "w1")
		f.must(err)
		f.must(f.b.Recheck(answer))
		if answer.Installation() != "synthetic-installation" || answer.Workspace() != "w1" || answer.User() != "ann" || answer.Kind() != Human ||
			answer.Epoch() != 1 || answer.Role() != Member || answer.Connection() != "synthetic-connection" || !answer.NotAfter().Equal(start.Add(8*time.Hour)) {
			t.Fatalf("answer %+v", answer)
		}
		return world{f, session, answer}
	}
	admin := func(w world) Verifier { return w.login("root") }
	changes := map[string]func(world){
		"the user is suspended": func(w world) { w.must(w.in.SuspendUser(operator, "ann", 1)) },
		"the user is suspended and reinstated": func(w world) {
			w.must(w.in.SuspendUser(operator, "ann", 1))
			w.must(w.in.ReinstateUser(operator, "ann", 2))
		},
		"the user departs": func(w world) { w.must(w.in.DepartUser(operator, "ann", 1)) },
		"the role changes": func(w world) { w.must(w.w.ChangeRole(admin(w), "w1", "ann", Administrator, 1)) },
		"the role changes and changes back": func(w world) {
			w.must(w.w.ChangeRole(admin(w), "w1", "ann", Administrator, 1))
			w.must(w.w.ChangeRole(admin(w), "w1", "ann", Member, 2))
		},
		"the membership is suspended": func(w world) { w.must(w.w.SuspendMembership(admin(w), "w1", "ann", 1)) },
		"the membership is suspended and back": func(w world) {
			w.must(w.w.SuspendMembership(admin(w), "w1", "ann", 1))
			w.must(w.w.ReinstateMembership(admin(w), "w1", "ann", 2))
		},
		"the membership is removed":               func(w world) { w.must(w.w.RemoveMembership(admin(w), "w1", "ann", 1)) },
		"the member is removed and invited again": func(w world) { w.must(w.w.RemoveMembership(admin(w), "w1", "ann", 1)); w.join("ann", Member) },
		"the session is ended":                    func(w world) { w.must(w.b.EndSession(w.session)) },
		"an identity is linked":                   func(w world) { w.must(w.b.LinkIdentity(w.session, triple("ann-2"), "", "proof")) },
		"the absolute expiry is reached":          func(w world) { *w.now = start.Add(8 * time.Hour) },
		"the clock is before the opening":         func(w world) { *w.now = start.Add(-time.Second) },
	}
	for name, change := range changes {
		w := build(t)
		change(w)
		w.refusal(name, w.b.Recheck(w.answer), ReasonAnswer)
	}
	w := build(t)
	*w.now = start.Add(8*time.Hour - time.Nanosecond)
	w.must(w.b.Recheck(w.answer))
	w.refusal("the zero answer", w.b.Recheck(Answer{}), ReasonAnswer)
	twin := build(t)
	twin.refusal("an answer of another registry with the same history", twin.b.Recheck(w.answer), ReasonAnswer)
	unlinked := build(t)
	unlinked.must(unlinked.b.LinkIdentity(unlinked.session, triple("ann-2"), "", "proof"))
	fresh := unlinked.login("ann")
	viaFirst, err := unlinked.b.Access(fresh, "w1")
	unlinked.must(err)
	second := v("via-second")
	unlinked.must(unlinked.b.OpenSession(triple("ann-2"), second, time.Minute, time.Hour))
	viaSecond, err := unlinked.b.Access(second, "w1")
	unlinked.must(err)
	unlinked.must(unlinked.b.UnlinkIdentity(fresh, triple("ann-2")))
	unlinked.refusal("an answer of the unlinking session", unlinked.b.Recheck(viaFirst), ReasonAnswer)
	unlinked.refusal("an answer through the unlinked identity", unlinked.b.Recheck(viaSecond), ReasonAnswer)
}

func TestAccessStopsWithASuspension(t *testing.T) {
	f := setup(t)
	f.member("ann", Member)
	sessions := []Verifier{}
	for index := 0; index < 6; index++ {
		sessions = append(sessions, f.login("ann"))
	}
	suspended := make(chan struct{})
	started := make(chan struct{}, len(sessions))
	var wg sync.WaitGroup
	late := make(chan int, len(sessions))
	for _, session := range sessions {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := f.b.Access(session, "w1"); err != nil {
				t.Errorf("access before the suspension: %v", err)
			}
			started <- struct{}{}
			for waiting := true; waiting; {
				select {
				case <-suspended:
					waiting = false
				default:
					f.b.Access(session, "w1")
				}
			}
			after := 0
			for index := 0; index < 300; index++ {
				if _, err := f.b.Access(session, "w1"); err == nil {
					after++
				}
			}
			late <- after
		}()
	}
	for range sessions {
		<-started
	}
	f.must(f.in.SuspendUser(operator, "ann", 1))
	close(suspended)
	wg.Wait()
	close(late)
	for after := range late {
		if after != 0 {
			t.Errorf("%d answers after the suspension returned", after)
		}
	}
}

func TestAnUnconstructedRegistryRefusesEverything(t *testing.T) {
	var r Registry
	var nilInstallation *Installation
	var nilBoundary *Boundary
	var nilWorkspace *Workspace
	hour := Validity{NotBefore: start, NotAfter: start.Add(time.Hour)}
	for _, handles := range []struct {
		in *Installation
		b  *Boundary
		w  *Workspace
	}{{r.Installation(), r.Boundary(), r.Workspace()}, {nilInstallation, nilBoundary, nilWorkspace}, {&Installation{}, &Boundary{}, &Workspace{}}} {
		in, b, w := handles.in, handles.b, handles.w
		calls := map[string]error{
			"ArmBootstrap": in.ArmBootstrap(v("x"), hour, "w1"), "CreateUser": in.CreateUser(operator, "u", triple("u"), ""),
			"SuspendUser": in.SuspendUser(operator, "u", 1), "ReinstateUser": in.ReinstateUser(operator, "u", 1), "DepartUser": in.DepartUser(operator, "u", 1),
			"Users": second(in.Users("", 1)), "WithoutAdministrator": second(in.WithoutAdministrator()),
			"Bootstrap": b.Bootstrap(v("x"), "u", triple("u"), ""), "Resolve": second(b.Resolve(triple("u"))), "OpenSession": b.OpenSession(triple("u"), v("s"), time.Minute, time.Hour),
			"EndSession": b.EndSession(v("s")), "Access": second(b.Access(v("s"), "w1")), "AccessByCredential": second(b.AccessByCredential("c", v("c"))),
			"Recheck": b.Recheck(Answer{}), "LinkIdentity": b.LinkIdentity(v("s"), triple("n"), "", "proof"), "UnlinkIdentity": b.UnlinkIdentity(v("s"), triple("n")),
			"Invite": w.Invite(v("s"), "w1", "i", Member, v("i"), hour, "u"), "RevokeInvitation": w.RevokeInvitation(v("s"), "w1", "i"), "Accept": w.Accept("w1", "i", v("i"), v("s")),
			"ChangeRole": w.ChangeRole(v("s"), "w1", "u", Member, 1), "SuspendMembership": w.SuspendMembership(v("s"), "w1", "u", 1),
			"ReinstateMembership": w.ReinstateMembership(v("s"), "w1", "u", 1), "RemoveMembership": w.RemoveMembership(v("s"), "w1", "u", 1),
			"Members": second(w.Members(v("s"), "w1", "", 1)), "CreateServiceUser": w.CreateServiceUser(v("s"), "w1", "svc", "u"),
			"SuspendServiceUser": w.SuspendServiceUser(v("s"), "w1", "svc", 1), "ReinstateServiceUser": w.ReinstateServiceUser(v("s"), "w1", "svc", 1),
			"RetireServiceUser": w.RetireServiceUser(v("s"), "w1", "svc", 1), "ServiceUsers": second(w.ServiceUsers(v("s"), "w1", "", 1)),
			"AddCredential": w.AddCredential(v("s"), "w1", "svc", "c", v("c"), hour), "RevokeCredential": w.RevokeCredential(v("s"), "w1", "c"),
		}
		if len(calls) != 31 {
			t.Fatalf("%d operations are covered", len(calls))
		}
		for name, err := range calls {
			if !errors.Is(err, ErrUnavailable) {
				t.Errorf("%s on an unconstructed registry: %v", name, err)
			}
		}
	}
}

func TestDeadSessionRecordsAreDropped(t *testing.T) {
	f := setup(t)
	f.member("ann", Member)
	held := func() int {
		f.r.mu.Lock()
		defer f.r.mu.Unlock()
		count := 0
		for _, found := range f.r.sessions {
			if found.user == "ann" {
				count++
			}
		}
		if count != len(f.r.byUser["ann"]) {
			t.Errorf("%d sessions and %d index entries", count, len(f.r.byUser["ann"]))
		}
		indexed := 0
		for _, found := range f.r.serials {
			if found.user == "ann" {
				indexed++
			}
		}
		if indexed != count {
			t.Errorf("%d sessions and %d serial entries", count, indexed)
		}
		return count
	}
	steps := []struct {
		name string
		run  func()
	}{
		{"suspension", func() { f.must(f.in.SuspendUser(operator, "ann", 1)); f.must(f.in.ReinstateUser(operator, "ann", 2)) }},
		{"ending", func() {
			session := f.login("ann")
			f.must(f.b.EndSession(session))
		}},
		{"linking", func() { f.must(f.b.LinkIdentity(f.login("ann"), triple("ann-2"), "", "proof")) }},
		{"unlinking", func() { f.must(f.b.UnlinkIdentity(f.login("ann"), triple("ann-2"))) }},
		{"departure", func() { f.must(f.in.DepartUser(operator, "ann", f.epoch("ann"))) }},
	}
	for _, step := range steps {
		if step.name != "ending" {
			f.login("ann")
			f.login("ann")
		}
		step.run()
		if count := held(); count != 0 {
			t.Errorf("%d session records are kept after %s", count, step.name)
		}
	}
}
