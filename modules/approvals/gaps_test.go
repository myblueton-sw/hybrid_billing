package approvals

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"strings"
	"testing"
	"time"
)

func TestValidators(t *testing.T) {
	for _, id := range []string{"Z", "z", "9", "A", "a", "0", ".", "_", ":", "-", strings.Repeat("a", 128)} {
		if !validID(id) {
			t.Errorf("identifier %q", id)
		}
	}
	for _, id := range []string{"", "/a", "a/", "[", "{", "@", "`", " a", strings.Repeat("a", 129)} {
		if validID(id) {
			t.Errorf("identifier %q", id)
		}
	}
	for _, code := range []string{"AAA", "ZZZ"} {
		if !validCurrency(code) {
			t.Errorf("currency %q", code)
		}
	}
	for _, code := range []string{"@AA", "A@A", "AA@", "[AA", "A[A", "AA[", "AAAA"} {
		if validCurrency(code) {
			t.Errorf("currency %q", code)
		}
	}
	for _, date := range []string{"2000-01-01", "2099-12-31", "2024-02-29", "2026-10-10"} {
		if !validDate(date) {
			t.Errorf("date %q", date)
		}
	}
	for _, date := range []string{"1999-12-31", "2100-01-01", "2026-02-29", "2026-13-01", "2026-1-01", "20261001"} {
		if validDate(date) {
			t.Errorf("date %q", date)
		}
	}
	for _, text := range []string{strings.Repeat("0", 64), strings.Repeat("f", 64), strings.Repeat("9", 64), strings.Repeat("a", 64)} {
		if !validDigest(text) {
			t.Errorf("digest %q", text)
		}
	}
	for _, text := range []string{"g" + strings.Repeat("0", 63), "/" + strings.Repeat("0", 63), "`" + strings.Repeat("0", 63), strings.Repeat("0", 63) + ":", strings.Repeat("A", 64), strings.Repeat("0", 63)} {
		if validDigest(text) {
			t.Errorf("digest %q", text)
		}
	}
}

func TestMutationGaps(t *testing.T) {
	r, _ := setup(t, 1)
	administer(t, r, Proposal{Kind: ProposeGrant, Authority: Authority{ID: "g-zero", Approver: "zed", Stage: StageIssue, Currency: "USD", Limit: "0", From: "2026-01-01"}})
	must(r.Request("ws", subject("z", "t-zero", "0"), human("req")))
	expect(t, "ok", second(r.Decide("ws", "z", approve, human("zed"))), "a zero limit covers a zero amount")
	must(r.Request("ws", subject("z1", "t-one", "-0.01"), human("req")))
	expect(t, ReasonNoAuthority, second(r.Decide("ws", "z1", approve, human("zed"))), "a zero limit refuses a credit")
	three := policy(StageCharge, 1, 3, "result")
	three.Policy.Validity = 90 * 24 * time.Hour
	administer(t, r, three)
	if p := must(r.PolicyOf("ws", StageCharge)); p.Approvals != 3 || p.Validity != 90*24*time.Hour {
		t.Error(p)
	}
	charge := Subject{ID: "c", Stage: StageCharge, Target: "run", Author: "writer", Digests: map[string]string{"result": digest('a')}, Amount: "1", Currency: "USD"}
	must(r.Request("ws", charge, human("req")))
	must(r.Request("ws", subject("i1", "inv-9", "1"), human("req")))
	must(r.Request("ws", subject("i2", "inv-8", "1"), human("req")))
	must(r.Decide("ws", "i2", Decision{Verdict: VerdictReject, Reason: "other"}, human("ann")))
	must(r.Request("ws", subject("i3", "inv-7", "1"), human("req")))
	must(r.Withdraw("ws", "i3", human("req")))
	administer(t, r, policy(StageIssue, 2, 1, "result", "erp_draft"))
	for id, state := range map[string]string{"c": StatePending, "i1": StateSuperseded, "i2": StateRejected, "i3": StateWithdrawn} {
		if v := must(r.RequestOf("ws", id)); v.State != state {
			t.Errorf("%s after a policy change: %s", id, v.State)
		}
	}
	full := subject("f", "batch-full", "4")
	for _, id := range []string{"a", "b", "c", "d"} {
		full.Entries = append(full.Entries, Entry{Item: id, Scope: "s", Amount: "1", Digest: digest('1')})
	}
	expect(t, "ok", second(r.Request("ws", full, human("req"))), "entries at the room")
	one := subject("o", "batch-one", "1")
	one.Entries = []Entry{{Item: "only", Scope: "s", Amount: "1", Digest: digest('1')}}
	must(r.Request("ws", one, human("req")))
	expect(t, ReasonNothingLeft, second(r.Decide("ws", "o", Decision{Verdict: VerdictApprove, Exclude: []Exclusion{{Item: "only", Reason: "not_eligible"}}}, human("ann"))), "excluding the only entry")
	two := subject("w", "batch-two", "2")
	two.Entries = []Entry{{Item: "p", Scope: "s", Amount: "1", Digest: digest('1')}, {Item: "q", Scope: "s", Amount: "1", Digest: digest('1')}}
	must(r.Request("ws", two, human("req")))
	must(r.Decide("ws", "w", Decision{Verdict: VerdictApprove, Exclude: []Exclusion{{Item: "p", Reason: "not_eligible"}}}, human("ann")))
	expect(t, "conflict", second(r.Decide("ws", "w", Decision{Verdict: VerdictApprove, Exclude: []Exclusion{{Item: "q", Reason: "not_eligible"}}}, human("ann"))), "another excluded item")
	expect(t, "conflict", second(r.Decide("ws", "w", Decision{Verdict: VerdictApprove, Exclude: []Exclusion{{Item: "p", Reason: "data_missing"}}}, human("ann"))), "another exclusion reason")
	expect(t, "ok", second(r.Decide("ws", "w", Decision{Verdict: VerdictApprove, Exclude: []Exclusion{{Item: "p", Reason: "not_eligible"}}}, human("ann"))), "the same decision")
	e := item("e-q", "w", "q", "1", '1')
	e.Target = "batch-two"
	must(r.Execute("ws", e, human("ops")))
	must(r.Report("ws", "e-q", Outcome{Result: ResultFailed, Reason: "timeout"}, human("ops")))
	expect(t, "conflict", second(r.Report("ws", "e-q", Outcome{Result: ResultFailed, Reason: "external_error"}, human("ops"))), "a failure after a failure")
}

func TestPinnedSubjectDigest(t *testing.T) {
	r, _ := setup(t, 1)
	v := must(r.Request("ws", subject("q", "inv", "10"), human("req")))
	sum := sha256.New()
	fields := []string{subjectLabel, "ws", "q", StageIssue, "inv", "writer", "10", "USD", "1", "2", "erp_draft", digest('b'), "result", digest('a'), "0"}
	for _, field := range fields {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(field)))
		sum.Write(size[:])
		sum.Write([]byte(field))
	}
	if want := hex.EncodeToString(sum.Sum(nil)); v.Digest != want {
		t.Errorf("%s, want %s", v.Digest, want)
	}
}
