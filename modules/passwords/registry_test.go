package passwords

import (
	"encoding/hex"
	"errors"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

var start = time.Date(2026, 11, 16, 9, 0, 0, 0, time.UTC)

type world struct {
	r   *Registry
	now *time.Time
	mu  *sync.Mutex
}

func fast() Params {
	p := DefaultParams()
	p.Iterations = 1000
	return p
}

func setup(t testing.TB) world {
	t.Helper()
	now := start
	mu := &sync.Mutex{}
	r, err := New(func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return now
	}, fast())
	if err != nil {
		t.Fatal(err)
	}
	return world{r: r, now: &now, mu: mu}
}

func (w world) advance(d time.Duration) {
	w.mu.Lock()
	*w.now = w.now.Add(d)
	w.mu.Unlock()
}

var ann = Key{Issuer: "prelaunch", Subject: "ann@example.invalid"}

const good = "correct horse battery"

func reason(t testing.TB, name string, err error, want string) {
	t.Helper()
	var r Refusal
	if !errors.As(err, &r) || r.Reason != want {
		t.Errorf("%s: got %v want %s", name, err, want)
	}
}

func TestParams(t *testing.T) {
	d := DefaultParams()
	if d != (Params{Iterations: 600000, SaltBytes: 16, KeyBytes: 32, MinLength: 12, MaxLength: 256, Attempts: 5, Lockout: 15 * time.Minute,
		MinValidity: time.Hour, MaxValidity: 168 * time.Hour, Records: 100000, Links: 10000}) {
		t.Error("defaults")
	}
	if _, err := New(nil, d); err != ErrInvalid {
		t.Error("clock")
	}
	for name, mutate := range map[string]func(*Params){
		"iterations": func(p *Params) { p.Iterations = 0 }, "salt": func(p *Params) { p.SaltBytes = 15 }, "key": func(p *Params) { p.KeyBytes = 15 },
		"min": func(p *Params) { p.MinLength = 0 }, "max": func(p *Params) { p.MaxLength = p.MinLength - 1 }, "attempts": func(p *Params) { p.Attempts = 0 },
		"lockout": func(p *Params) { p.Lockout = 0 }, "minv": func(p *Params) { p.MinValidity = 0 }, "maxv": func(p *Params) { p.MaxValidity = p.MinValidity - 1 },
		"records": func(p *Params) { p.Records = 0 }, "links": func(p *Params) { p.Links = 0 },
	} {
		p := fast()
		mutate(&p)
		if _, err := New(time.Now, p); err != ErrInvalid {
			t.Errorf("%s accepted", name)
		}
	}
	edge := fast()
	edge.SaltBytes, edge.KeyBytes, edge.MinLength, edge.MaxLength, edge.Attempts, edge.Records, edge.Links, edge.Iterations = 16, 16, 1, 1, 1, 1, 1, 1
	edge.MaxValidity = edge.MinValidity
	if _, err := New(time.Now, edge); err != nil {
		t.Error("edge params")
	}
}

func TestHashAndVerify(t *testing.T) {
	w := setup(t)
	salt := make([]byte, 16)
	for i := range salt {
		salt[i] = byte(i)
	}
	if got := hex.EncodeToString(w.r.derive(salt, []byte(good)).Hash); got != "02169e674d80c0fc7292e8610c8f17be140df062af9aa33ea85eaabecffdf08d" {
		t.Fatalf("pbkdf2 kat %s", got)
	}
	reason(t, "short", second(w.r.Hash(ann, strings.Repeat("a", 11))), ReasonPasswordForm)
	reason(t, "long", second(w.r.Hash(ann, strings.Repeat("a", 257))), ReasonPasswordForm)
	reason(t, "utf8", second(w.r.Hash(ann, strings.Repeat("a", 12)+"\xff")), ReasonPasswordForm)
	reason(t, "same", second(w.r.Hash(Key{Issuer: "i", Subject: "abcdefghijklm"}, "abcdefghijklm")), ReasonSameAsLogin)
	for _, k := range []Key{{Subject: "x"}, {Issuer: "i"}, {Issuer: "a b", Subject: "x"}} {
		if _, err := w.r.Hash(k, good); err != ErrInvalid {
			t.Errorf("key %+v", k)
		}
	}
	for _, p := range []string{strings.Repeat("a", 12), strings.Repeat("a", 256)} {
		if _, err := w.r.Hash(ann, p); err != nil {
			t.Errorf("length %d", len(p))
		}
	}
	a, _ := w.r.Hash(ann, good)
	b, _ := w.r.Hash(ann, good)
	if string(a.Salt) == string(b.Salt) || len(a.Salt) != 16 || len(a.Hash) != 32 || a.Iterations != 1000 {
		t.Error("salted")
	}
	if w.r.Verify(ann, good) != ResultWrong || w.r.Has(ann) {
		t.Error("unknown")
	}
	for _, bad := range []Record{{Iterations: 1}, {Iterations: 1000, Salt: make([]byte, 15), Hash: make([]byte, 32)}, {Iterations: 1000, Salt: make([]byte, 16), Hash: make([]byte, 31)}} {
		if w.r.Put(ann, bad) != ErrInvalid {
			t.Error("bad record")
		}
	}
	if w.r.Put(Key{}, a) != ErrInvalid {
		t.Error("bad key")
	}
	if err := w.r.Put(ann, a); err != nil || !w.r.Has(ann) {
		t.Fatal(err)
	}
	a.Hash[0] ^= 1
	if w.r.Verify(ann, good) != ResultOK {
		t.Error("stored copy")
	}
	if w.r.Verify(ann, good+"x") != ResultWrong {
		t.Error("wrong")
	}
	w.r.Remove(ann)
	if w.r.Has(ann) {
		t.Error("removed")
	}
}

func second[T any](_ T, err error) error { return err }

func TestLockout(t *testing.T) {
	w := setup(t)
	rec, _ := w.r.Hash(ann, good)
	w.r.Put(ann, rec)
	for i := 0; i < 4; i++ {
		w.r.Verify(ann, "nope nope nope")
	}
	if w.r.Verify(ann, good) != ResultOK {
		t.Fatal("four failures do not lock")
	}
	for i := 0; i < 4; i++ {
		w.r.Verify(ann, "nope nope nope")
	}
	w.advance(15 * time.Minute)
	w.r.Verify(ann, "nope nope nope")
	if w.r.Verify(ann, good) != ResultOK {
		t.Error("failures outside the window restart the count")
	}
	for i := 0; i < 5; i++ {
		w.r.Verify(ann, "nope nope nope")
	}
	if w.r.Verify(ann, good) != ResultLocked {
		t.Fatal("locked at fifth failure")
	}
	w.advance(15*time.Minute - time.Nanosecond)
	if w.r.Verify(ann, good) != ResultLocked {
		t.Error("still locked")
	}
	w.advance(time.Nanosecond)
	if w.r.Verify(ann, good) != ResultOK {
		t.Error("unlocked")
	}
	for i := 0; i < 5; i++ {
		w.r.Verify(ann, "nope nope nope")
	}
	rec2, _ := w.r.Hash(ann, good)
	w.r.Put(ann, rec2)
	if w.r.Verify(ann, good) != ResultOK {
		t.Error("put clears lock")
	}
	full := fast()
	full.Records = 1
	r, _ := New(time.Now, full)
	r.Put(ann, rec)
	if r.Put(Key{Issuer: "i", Subject: "other"}, rec) != ErrLimit || r.Put(ann, rec) != nil {
		t.Error("records cap")
	}
}

func TestLinks(t *testing.T) {
	w := setup(t)
	for _, c := range []struct {
		k Key
		p string
		v time.Duration
	}{{Key{}, PurposeSet, time.Hour}, {ann, "other", time.Hour}, {ann, PurposeSet, time.Hour - 1}, {ann, PurposeSet, 168*time.Hour + 1}} {
		if _, err := w.r.Issue(c.k, c.p, c.v); err != ErrInvalid {
			t.Errorf("issue %+v", c)
		}
	}
	tok, err := w.r.Issue(ann, PurposeSet, time.Hour)
	if err != nil || len(tok) != 43 {
		t.Fatal(err)
	}
	if l, err := w.r.Peek(tok); err != nil || l.Key != ann || l.Purpose != PurposeSet || !l.Expires.Equal(start.Add(time.Hour)) {
		t.Errorf("peek %+v %v", l, err)
	}
	for _, bad := range []string{"", "!!", "c2hvcnQ", tok + "x"} {
		reason(t, "peek "+bad, second(w.r.Peek(bad)), ReasonLinkInvalid)
		reason(t, "redeem "+bad, second(redeem(w.r, bad, good)), ReasonLinkInvalid)
	}
	reason(t, "redeem form", second(redeem(w.r, tok, "short")), ReasonPasswordForm)
	k, err := redeem(w.r, tok, good)
	if err != nil || k != ann || w.r.Verify(ann, good) != ResultOK {
		t.Fatal(err)
	}
	reason(t, "reuse", second(redeem(w.r, tok, good)), ReasonLinkInvalid)
	old, _ := w.r.Issue(ann, PurposeReset, time.Hour)
	newer, _ := w.r.Issue(ann, PurposeReset, time.Hour)
	reason(t, "replaced", second(w.r.Peek(old)), ReasonLinkInvalid)
	rec, _ := w.r.Hash(ann, good)
	w.r.Put(ann, rec)
	reason(t, "put invalidates", second(w.r.Peek(newer)), ReasonLinkInvalid)
	late, _ := w.r.Issue(ann, PurposeReset, time.Hour)
	w.advance(time.Hour)
	reason(t, "expired", second(w.r.Peek(late)), ReasonLinkInvalid)
	w.r.Remove(ann)
	gone, _ := w.r.Issue(ann, PurposeSet, time.Hour)
	w.r.Remove(ann)
	reason(t, "remove invalidates", second(w.r.Peek(gone)), ReasonLinkInvalid)
	race, _ := w.r.Issue(ann, PurposeSet, time.Hour)
	var wg sync.WaitGroup
	wins := make(chan bool, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := redeem(w.r, race, good)
			wins <- err == nil
		}()
	}
	wg.Wait()
	close(wins)
	n := 0
	for ok := range wins {
		if ok {
			n++
		}
	}
	if n != 1 {
		t.Errorf("concurrent redemptions %d", n)
	}
	cap1 := fast()
	cap1.Links = 1
	r, _ := New(time.Now, cap1)
	if _, err := r.Issue(ann, PurposeSet, time.Hour); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Issue(ann, PurposeSet, time.Hour); err != nil {
		t.Error("replacing own link within cap")
	}
	if _, err := r.Issue(Key{Issuer: "i", Subject: "b"}, PurposeSet, time.Hour); err != ErrLimit {
		t.Error("links cap")
	}
	rc := fast()
	rc.Records = 1
	q, _ := New(time.Now, rc)
	rec3, _ := q.Hash(ann, good)
	q.Put(ann, rec3)
	other := Key{Issuer: "i", Subject: "b"}
	tk, _ := q.Issue(other, PurposeSet, time.Hour)
	if _, err := redeem(q, tk, good); err != ErrLimit {
		t.Error("redeem records cap")
	}
	if _, err := q.Peek(tk); err != nil {
		t.Error("link kept after cap refusal")
	}
	if !errors.Is(Refusal{Reason: "x"}, ErrRejected) || (Refusal{Reason: "x"}).Error() != ErrRejected.Error()+": x" {
		t.Error("refusal")
	}
}

func redeem(r *Registry, tok, pw string) (Key, error) {
	l, err := r.Reserve(tok)
	if err != nil {
		return Key{}, err
	}
	rec, err := r.Hash(l.Key, pw)
	if err != nil {
		r.Release(tok, l.Hold)
		return Key{}, err
	}
	k, err := r.Commit(tok, rec)
	if err != nil {
		r.Release(tok, l.Hold)
	}
	return k, err
}

func TestRedeemSteps(t *testing.T) {
	w := setup(t)
	tok, _ := w.r.Issue(ann, PurposeSet, time.Hour)
	l, err := w.r.Reserve(tok)
	if err != nil || l.Key != ann {
		t.Fatal(err)
	}
	reason(t, "reserved not peekable", second(w.r.Peek(tok)), ReasonLinkInvalid)
	reason(t, "reserved twice", second(w.r.Reserve(tok)), ReasonLinkInvalid)
	rec, _ := w.r.Hash(ann, good)
	if _, err := w.r.Commit(tok, Record{}); err != ErrInvalid {
		t.Error("bad record")
	}
	w.r.Release(tok, l.Hold.Add(time.Second))
	reason(t, "release with another hold", second(w.r.Peek(tok)), ReasonLinkInvalid)
	w.r.Release(tok, l.Hold)
	if _, err := w.r.Peek(tok); err != nil {
		t.Error("released link valid again")
	}
	reason(t, "commit unreserved", second(w.r.Commit(tok, rec)), ReasonLinkInvalid)
	reason(t, "commit garbage", second(w.r.Commit("!!", rec)), ReasonLinkInvalid)
	w.r.Release("!!", time.Time{})
	w.r.Reserve(tok)
	other, _ := w.r.Hash(ann, "another good password")
	w.r.Put(ann, other)
	reason(t, "replaced during hashing", second(w.r.Commit(tok, rec)), ReasonLinkInvalid)
	if w.r.Verify(ann, "another good password") != ResultOK {
		t.Error("concurrent put kept")
	}
	if _, err := w.r.Hash(Key{Issuer: "i", Subject: "a\x01"}, good); err != ErrInvalid {
		t.Error("control character subject")
	}
	w.r.between = func() {
		w.r.between = nil
		again, _ := w.r.Hash(ann, good)
		w.r.Put(ann, again)
	}
	if w.r.Verify(ann, "another good password") != ResultWrong {
		t.Error("stale verify counted as ok")
	}
	if w.r.accounts[ann].failures != 0 || w.r.Verify(ann, good) != ResultOK {
		t.Error("stale outcome not recorded")
	}
}

func TestReservationLapses(t *testing.T) {
	w := setup(t)
	tok, _ := w.r.Issue(ann, PurposeSet, time.Hour)
	w.r.Reserve(tok)
	w.advance(5*time.Minute - time.Nanosecond)
	reason(t, "still reserved", second(w.r.Peek(tok)), ReasonLinkInvalid)
	w.advance(time.Nanosecond)
	if _, err := w.r.Peek(tok); err != nil {
		t.Error("reservation lapsed")
	}
	rec, _ := w.r.Hash(ann, good)
	reason(t, "commit after lapse", second(w.r.Commit(tok, rec)), ReasonLinkInvalid)
	w.r.Reserve(tok)
	w.r.Commit(tok, rec)
	reason(t, "used", second(w.r.Reserve(tok)), ReasonLinkInvalid)
	w.r.Release(tok, time.Time{})
	reason(t, "release used no effect", second(w.r.Peek(tok)), ReasonLinkInvalid)
}

func TestBoundaries(t *testing.T) {
	edge := fast()
	edge.Lockout, edge.MinValidity, edge.MaxValidity = time.Nanosecond, time.Nanosecond, time.Nanosecond
	if _, err := New(time.Now, edge); err != nil {
		t.Error("one nanosecond lockout and validity")
	}
	w := setup(t)
	if _, err := w.r.Issue(ann, PurposeSet, 7*24*time.Hour); err != nil {
		t.Error("exact maximum validity")
	}
	if _, err := w.r.Issue(ann, PurposeSet, time.Hour); err != nil {
		t.Error("exact minimum validity")
	}
}

func TestExpiredLinksPurged(t *testing.T) {
	p := fast()
	p.Links = 1
	now := start
	r, _ := New(func() time.Time { return now }, p)
	r.Issue(ann, PurposeSet, time.Hour)
	other := Key{Issuer: "i", Subject: "other"}
	if _, err := r.Issue(other, PurposeSet, time.Hour); err != ErrLimit {
		t.Fatal("cap")
	}
	now = now.Add(time.Hour)
	tok, err := r.Issue(other, PurposeSet, time.Hour)
	if err != nil || len(r.links) != 1 || len(r.byKey) != 1 {
		t.Fatalf("expired purged %v", err)
	}
	r.Reserve(tok)
	rec, _ := r.Hash(other, good)
	r.Commit(tok, rec)
	if _, err := r.Issue(ann, PurposeSet, time.Hour); err != nil {
		t.Error("used purged")
	}
}

type lock struct {
	failures int
	first    time.Time
	locked   time.Time
}

func lockState(r *Registry, k Key) lock {
	r.mu.Lock()
	defer r.mu.Unlock()
	a := r.accounts[k]
	return lock{a.failures, a.first, a.locked}
}

func TestVerifyHolder(t *testing.T) {
	w := setup(t)
	if w.r.VerifyHolder(ann, good) != ResultWrong {
		t.Error("unknown key")
	}
	rec, _ := w.r.Hash(ann, good)
	w.r.Put(ann, rec)
	if w.r.VerifyHolder(ann, good) != ResultOK || w.r.VerifyHolder(ann, good+"x") != ResultWrong || w.r.VerifyHolder(ann, "") != ResultWrong {
		t.Error("known answer")
	}
	w.r.between = func() {
		w.r.between = nil
		again, _ := w.r.Hash(ann, "another good password")
		w.r.Put(ann, again)
	}
	if w.r.VerifyHolder(ann, good) != ResultWrong {
		t.Error("concurrent put")
	}
	w.r.between = func() {
		w.r.between = nil
		w.r.Remove(ann)
	}
	if w.r.VerifyHolder(ann, "another good password") != ResultWrong {
		t.Error("concurrent remove")
	}
}

func TestVerifyHolderIgnoresLock(t *testing.T) {
	w := setup(t)
	rec, _ := w.r.Hash(ann, good)
	w.r.Put(ann, rec)
	for i := 0; i < 5; i++ {
		w.r.Verify(ann, "nope nope nope")
	}
	before := lockState(w.r, ann)
	if w.r.Verify(ann, good) != ResultLocked || before.failures != 5 || !before.locked.Equal(start.Add(15*time.Minute)) || !before.first.Equal(start) {
		t.Fatalf("not locked %+v", before)
	}
	if w.r.VerifyHolder(ann, good) != ResultOK {
		t.Error("holder refused by a lock set by others")
	}
	for i := 0; i < 20; i++ {
		w.r.VerifyHolder(ann, "nope nope nope")
	}
	if after := lockState(w.r, ann); after != before {
		t.Errorf("lock state changed %+v %+v", before, after)
	}
	w.advance(15 * time.Minute)
	if w.r.Verify(ann, good) != ResultOK {
		t.Error("lock extended")
	}
	for i := 0; i < 10; i++ {
		w.r.VerifyHolder(ann, "nope nope nope")
	}
	if w.r.Verify(ann, good) != ResultOK || lockState(w.r, ann).failures != 0 {
		t.Error("holder failures created a lock")
	}
	for i := 0; i < 3; i++ {
		w.r.Verify(ann, "nope nope nope")
	}
	w.r.VerifyHolder(ann, good)
	if lockState(w.r, ann).failures != 3 {
		t.Error("holder success reset the count")
	}
}

func TestVerifyHolderTiming(t *testing.T) {
	if testing.Short() {
		t.Skip("timing")
	}
	r, _ := New(time.Now, DefaultParams())
	rec, _ := r.Hash(ann, good)
	r.Put(ann, rec)
	cases := []struct {
		k        Key
		password string
		want     string
	}{{ann, good, ResultOK}, {ann, "nope nope nope", ResultWrong}, {Key{Issuer: "prelaunch", Subject: "nobody@example.invalid"}, good, ResultWrong}}
	runs := make([][]time.Duration, len(cases))
	for i := 0; i < 20; i++ {
		for n, c := range cases {
			began := time.Now()
			if got := r.VerifyHolder(c.k, c.password); got != c.want {
				t.Fatalf("result %s", got)
			}
			runs[n] = append(runs[n], time.Since(began))
		}
	}
	median := func(d []time.Duration) time.Duration {
		sort.Slice(d, func(a, b int) bool { return d[a] < d[b] })
		return d[len(d)/2]
	}
	correct, wrong, unknown := median(runs[0]), median(runs[1]), median(runs[2])
	low, high := min(correct, wrong, unknown), max(correct, wrong, unknown)
	if high*2 > low*3 {
		t.Errorf("timing differs: correct %v wrong %v unknown %v", correct, wrong, unknown)
	}
}

func TestVerifyHolderUnknownKeyWithTheDummyPassword(t *testing.T) {
	w := setup(t)
	if got := w.r.VerifyHolder(ann, "dummy password for timing"); got != ResultWrong {
		t.Errorf("unknown key matched the dummy record: %s", got)
	}
	if got := w.r.Verify(ann, "dummy password for timing"); got != ResultWrong {
		t.Errorf("verify: %s", got)
	}
}
