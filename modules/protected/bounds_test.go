package protected

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"
)

func TestLimitBoundsLiteral(t *testing.T) {
	clock := &testClock{now: start}
	must(NewRegistry(Limits{1, 1, 1}, clock.read))
	must(NewRegistry(Limits{100000, 1000000, 1000000}, clock.read))
	for _, l := range []Limits{{100001, 1, 1}, {1, 1000001, 1}, {1, 1, 1000001}} {
		if _, err := NewRegistry(l, clock.read); err != ErrInvalid {
			t.Fatalf("%+v: %v", l, err)
		}
	}
}

func TestRuleBoundsLiteral(t *testing.T) {
	f := newFixture(t)
	ok := Rules{Approvals: 3, Validity: 30 * 24 * time.Hour, MaxLead: 366 * 24 * time.Hour, ChannelAge: 90 * 24 * time.Hour}
	expect(t, "upper", second(f.r.Propose("w1", ok, human("a"))), "ok")
	for i, r := range []Rules{
		{Approvals: 4, Validity: time.Hour, MaxLead: time.Hour, ChannelAge: 24 * time.Hour},
		{Approvals: 1, Validity: 30*24*time.Hour + 1, MaxLead: time.Hour, ChannelAge: 24 * time.Hour},
		{Approvals: 1, Validity: time.Hour, MaxLead: 366*24*time.Hour + 1, ChannelAge: 24 * time.Hour},
		{Approvals: 1, Validity: time.Hour, MaxLead: time.Hour, ChannelAge: 90*24*time.Hour + 1},
		{Approvals: 1, Validity: time.Hour - 1, MaxLead: time.Hour, ChannelAge: 24 * time.Hour},
		{Approvals: 1, Validity: time.Hour, MaxLead: time.Hour - 1, ChannelAge: 24 * time.Hour},
		{Approvals: 1, Validity: time.Hour, MaxLead: time.Hour, ChannelAge: 24*time.Hour - 1},
	} {
		if got := kind(second(f.r.Propose("w2", r, human("a")))); got != "invalid" {
			t.Fatalf("case %d: %s", i, got)
		}
	}
}

func TestIdentifierAlphabet(t *testing.T) {
	if !validID(strings.Repeat("a", 128)) || validID(strings.Repeat("a", 129)) {
		t.Fatal("length")
	}
	for _, id := range []string{"A", "Z", "a", "z", "0", "9", ".", "_", ":", "-"} {
		if !validID(id) {
			t.Fatalf("%q refused", id)
		}
	}
	for _, id := range []string{"@", "[", "`", "{", "/", ";", " a", "!a", "é"} {
		if validID(id) {
			t.Fatalf("%q accepted", id)
		}
	}
	if !validDigest(strings.Repeat("0", 32)+strings.Repeat("9", 16)+strings.Repeat("f", 16)) || !validDigest("a"+strings.Repeat("0", 63)) {
		t.Fatal("digest alphabet")
	}
	for _, d := range []string{"G" + strings.Repeat("0", 63), "/" + strings.Repeat("0", 63), ":" + strings.Repeat("0", 63), "`" + strings.Repeat("0", 63), "g" + strings.Repeat("0", 63)} {
		if validDigest(d) {
			t.Fatalf("%q accepted", d)
		}
	}
}

func TestHashKnownAnswer(t *testing.T) {
	sum := sha256.New()
	for _, text := range []string{"ab", "c"} {
		sum.Write([]byte{0, 0, 0, 0, 0, 0, 0, byte(len(text))})
		sum.Write([]byte(text))
	}
	if got := hashOf("ab", "c"); got != hex.EncodeToString(sum.Sum(nil)) {
		t.Fatal(got)
	}
}

func TestDiscardFirstRevisionAndAdmitForms(t *testing.T) {
	f := newFixture(t)
	must(f.r.Propose("w1", testRules(), human("admin")))
	d := must(f.r.Discard("w1", 1, human("other")))
	if d.Revision != 1 || d.Discarder != "other" {
		t.Fatalf("%+v", d)
	}
	expect(t, "admit workspace", second(f.r.Admit("", "ch", "e", human("a"))), "invalid")
	expect(t, "admit channel", second(f.r.Admit("w1", "", "e", human("a"))), "invalid")
	expect(t, "admit actor", second(f.r.Admit("w1", "ch", "e", human(""))), "invalid")
	expect(t, "retire workspace", second(f.r.Retire("", "ch", human("a"))), "invalid")
	expect(t, "retire actor", second(f.r.Retire("w1", "ch", human(""))), "invalid")
	expect(t, "register workspace", second(f.r.Register("", Channel{ID: "c", Subject: "s", Kind: ChannelEmail, Ref: ref("v", 'a'), Evidence: "e"}, human("a"))), "invalid")
	expect(t, "register actor", second(f.r.Register("w1", Channel{ID: "c", Subject: "s", Kind: ChannelEmail, Ref: ref("v", 'a'), Evidence: "e"}, human(""))), "invalid")
	expect(t, "confirm actor", second(f.r.Confirm("w1", 1, human(""))), "invalid")
	expect(t, "confirm workspace", second(f.r.Confirm("", 1, human("a"))), "invalid")
	expect(t, "discard actor", second(f.r.Discard("w1", 1, human(""))), "invalid")
	expect(t, "discard workspace", second(f.r.Discard("", 1, human("a"))), "invalid")
}

func TestProposalCap(t *testing.T) {
	f := newFixture(t)
	for i := 0; i < 1000; i++ {
		r := testRules()
		r.Validity = time.Hour + time.Duration(i)
		must(f.r.Propose("w1", r, human("a")))
	}
	r := testRules()
	expect(t, "cap", second(f.r.Propose("w1", r, human("a"))), "limit")
	if got := must(f.r.RulesOf("w1")); len(got.History) != 1000 || got.Pending.Revision != 1000 {
		t.Fatalf("after cap %d", len(got.History))
	}
}
