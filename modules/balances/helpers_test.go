package balances

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func testProfile(t testing.TB) decimal.Profile {
	t.Helper()
	made, err := decimal.NewProfile(decimal.ProfileSpec{ID: "balances-test", Revision: "1", InputIntegerDigits: 14, InputScale: 8, ResultIntegerDigits: 16, ResultScale: 8, IntermediateDigits: 80})
	if err != nil {
		t.Fatal(err)
	}
	return made
}

func testLimits() Limits {
	return Limits{Accounts: 6, Openings: 3, Increases: 6, Applications: 12, Targets: 8, Reversals: 3, Workspace: 10}
}

type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *testClock) read() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *testClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func (c *testClock) set(at time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = at
}

var start = time.Date(2026, 11, 14, 3, 0, 0, 0, time.UTC)

const evidence = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func openWith(t testing.TB, limits Limits) (*Registry, *testClock) {
	t.Helper()
	clock := &testClock{now: start}
	r, err := NewRegistry(testProfile(t), limits, clock.read)
	if err != nil {
		t.Fatal(err)
	}
	return r, clock
}

func kind(err error) string {
	switch {
	case err == nil:
		return "ok"
	case errors.Is(err, ErrInvalid):
		return "invalid"
	case errors.Is(err, ErrNotFound):
		return "not_found"
	case errors.Is(err, ErrConflict):
		return "conflict"
	case errors.Is(err, ErrRejected):
		return "rejected"
	case errors.Is(err, ErrLimit):
		return "limit"
	}
	return fmt.Sprintf("other:%v", err)
}

func cash(id, customer, currency string) Account {
	return Account{ID: id, Customer: customer, Entity: "entity-1", Currency: currency, Kind: KindCash, Authority: AuthorityPlatform}
}

func credit(id, customer, currency string) Account {
	return Account{ID: id, Customer: customer, Entity: "entity-1", Currency: currency, Kind: KindCredit, Authority: AuthorityPlatform}
}

func opened(t testing.TB, r *Registry, a Account) {
	t.Helper()
	if _, err := r.OpenAccount("ws", a, "clerk-1"); err != nil {
		t.Fatalf("open %s: %v", a.ID, err)
	}
}

func receipt(id, acct, amount string) Increase {
	return Increase{ID: id, Account: acct, Kind: IncreaseCashReceipt, Amount: amount, Reference: "ref-" + id}
}

func promotion(id, acct, amount, expires string) Increase {
	return Increase{ID: id, Account: acct, Kind: IncreasePromotion, Amount: amount, Reference: "ref-" + id, ExpiresOn: expires}
}

func funded(t testing.TB, r *Registry, i Increase) {
	t.Helper()
	if _, err := r.RecordIncrease("ws", i, "clerk-1"); err != nil {
		t.Fatalf("record %s: %v", i.ID, err)
	}
	if _, err := r.ApproveIncrease("ws", i.ID, "lead-1"); err != nil {
		t.Fatalf("approve %s: %v", i.ID, err)
	}
}

func apply(id, acct, document, amount, requested string, partial bool) Application {
	return Application{ID: id, Account: acct, Target: Target{Document: document, Line: "1", Amount: amount}, Requested: requested, Partial: partial, Lifetime: 3600}
}

func reserve(t testing.TB, r *Registry, a Application) Decision {
	t.Helper()
	d, err := r.Reserve("ws", a, "runner-1")
	if err != nil {
		t.Fatalf("reserve %s: %v", a.ID, err)
	}
	return d
}

func reserved(t testing.TB, r *Registry, a Application) ApplicationView {
	t.Helper()
	d := reserve(t, r, a)
	if d.Reason != "" || d.Application.State != StateReserved {
		t.Fatalf("reserve %s: %+v", a.ID, d)
	}
	return d.Application
}

func applied(t testing.TB, r *Registry, a Application) ApplicationView {
	t.Helper()
	reserved(t, r, a)
	v, err := r.Confirm("ws", a.ID, "runner-1")
	if err != nil {
		t.Fatalf("confirm %s: %v", a.ID, err)
	}
	return v
}

func refused(t testing.TB, r *Registry, a Application) string {
	t.Helper()
	d, err := r.Reserve("ws", a, "runner-2")
	if err != nil {
		t.Fatalf("reserve %s: %v", a.ID, err)
	}
	if d.Reason != "" && d.Application.Digest != "" {
		t.Fatalf("a refusal carries an application: %+v", d)
	}
	return d.Reason
}

func reversedBy(t testing.TB, r *Registry, rev Reversal) {
	t.Helper()
	if _, err := r.Reverse("ws", rev, "clerk-1"); err != nil {
		t.Fatalf("reverse %s: %v", rev.ID, err)
	}
	if _, err := r.ApproveReversal("ws", rev.ID, "lead-1"); err != nil {
		t.Fatalf("approve reversal %s: %v", rev.ID, err)
	}
}

func rat(t testing.TB, text string) *big.Rat {
	t.Helper()
	out, ok := new(big.Rat).SetString(text)
	if !ok {
		t.Fatalf("not a number: %q", text)
	}
	return out
}

func identity(t testing.TB, r *Registry, workspace, id string) Statement {
	t.Helper()
	st, err := r.Statement(workspace, id)
	if err != nil {
		t.Fatalf("statement %s: %v", id, err)
	}
	want := new(big.Rat).Add(rat(t, st.Opening), rat(t, st.Increases))
	want.Sub(want, rat(t, st.Applications))
	want.Sub(want, rat(t, st.Expired))
	want.Add(want, rat(t, st.ApplicationReversals))
	want.Sub(want, rat(t, st.LotReversals))
	if want.Cmp(rat(t, st.Closing)) != 0 {
		t.Fatalf("identity broken for %s: %+v", id, st)
	}
	if new(big.Rat).Sub(rat(t, st.Closing), rat(t, st.Held)).Cmp(rat(t, st.Available)) != 0 || rat(t, st.Available).Sign() < 0 || rat(t, st.Held).Sign() < 0 {
		t.Fatalf("available broken for %s: %+v", id, st)
	}
	lots, err := r.Lots(workspace, id)
	if err != nil {
		t.Fatalf("lots %s: %v", id, err)
	}
	closing, expired := new(big.Rat), new(big.Rat)
	for _, l := range lots {
		closing.Add(closing, rat(t, l.Remainder))
		expired.Add(expired, rat(t, l.Expired))
		if rat(t, l.Held).Cmp(rat(t, l.Remainder)) > 0 || rat(t, l.Available).Sign() < 0 {
			t.Fatalf("lot broken: %+v", l)
		}
	}
	if closing.Cmp(rat(t, st.Closing)) != 0 || expired.Cmp(rat(t, st.Expired)) != 0 {
		t.Fatalf("lots disagree with the statement: %+v %+v", lots, st)
	}
	return st
}

func printed(fields ...string) string {
	sum := sha256.New()
	for _, field := range fields {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(field)))
		sum.Write(size[:])
		sum.Write([]byte(field))
	}
	return hex.EncodeToString(sum.Sum(nil))
}

func readSource(file string) string {
	body, _ := os.ReadFile(file)
	return string(body)
}

func TestSourcesHaveNoFloatsOrComments(t *testing.T) {
	entries, _ := os.ReadDir(".")
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		body := readSource(name)
		for _, banned := range []string{"float" + "32", "float" + "64", "//" + " ", "/" + "*", "1" + "e"} {
			if strings.Contains(body, banned) {
				t.Errorf("%s contains %q", name, banned)
			}
		}
	}
}
