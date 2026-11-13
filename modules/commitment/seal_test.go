package commitment

import (
	"strings"
	"testing"
)

func reasons(s RevisionState) string { return s.State + ":" + strings.Join(s.Reasons, ",") }

func TestSealReasons(t *testing.T) {
	r := open(t)
	record(t, r, commitmentOf("c1", PaymentPartial, "150", RecoveryCharged))
	write(t, r, "c1", "2026-01", 1, "10", "100", used("a", "acct-a", "10", "100", "200"))
	if got := reasons(seal(t, r, "c1", "2026-01", 1)); got != "sealed:" {
		t.Fatalf("a period that passes: %s", got)
	}
	foreign := used("b", "acct-a", "5", "0", "9")
	foreign.Commitment = "c9"
	write(t, r, "c1", "2026-02", 1, "6", "60", foreign, used("a", "acct-a", "1", "60", "70"))
	if got := reasons(seal(t, r, "c1", "2026-02", 1)); got != "held:foreign_commitment,duplicate_source,prepaid_exceeded" {
		t.Fatalf("several reasons: %s", got)
	}
	write(t, r, "c1", "2026-03", 1, "1", "0", used("c", "acct-a", "1", "0.001", "2"))
	if got := reasons(seal(t, r, "c1", "2026-03", 1)); got != "held:cost_not_preserved,prepaid_exceeded" {
		t.Fatalf("a gap of the provider's smallest unit: %s", got)
	}
	if again := seal(t, r, "c1", "2026-01", 1); again.State != StateSealed {
		t.Fatal("a repeat seal")
	}
}

func TestCorrectionsAndCurrent(t *testing.T) {
	r := open(t)
	record(t, r, commitmentOf("c1", PaymentAll, "300", RecoveryCharged))
	write(t, r, "c1", "2026-01", 1, "0", "100", used("x", "acct-a", "0", "100", "150"))
	seal(t, r, "c1", "2026-01", 1)
	if cur, _ := r.Current("ws", "c1", "2026-01"); cur.State != CurrentSealed || cur.Revision != 1 {
		t.Fatalf("current: %+v", cur)
	}
	write(t, r, "c1", "2026-01", 2, "0", "0", used("y", "acct-a", "0", "0", "1"))
	if cur, _ := r.Current("ws", "c1", "2026-01"); cur.State != CurrentNotJudged || cur.Revision != 2 {
		t.Fatalf("a correction not judged: %+v", cur)
	}
	if _, err := r.Revision("ws", "c1", "2026-01", 1); kind(err) != "rejected" {
		t.Fatalf("an earlier sealed revision while a correction waits: %v", err)
	}
	write(t, r, "c1", "2026-02", 1, "0", "100", used("x", "acct-b", "0", "100", "150"))
	if got := reasons(seal(t, r, "c1", "2026-02", 1)); got != "held:duplicate_source" {
		t.Fatalf("a line sold in a sealed revision whose correction waits: %s", got)
	}
	write(t, r, "c1", "2026-03", 1, "0", "150", used("z", "acct-b", "0", "150", "150"))
	if got := reasons(seal(t, r, "c1", "2026-03", 1)); got != "held:prepaid_exceeded" {
		t.Fatalf("an amortized part sold stays counted: %s", got)
	}
	if _, err := r.Seal("ws", "c1", "2026-01", 1); kind(err) != "rejected" {
		t.Fatalf("sealing a lower revision: %v", err)
	}
	if got := reasons(seal(t, r, "c1", "2026-01", 2)); got != "sealed:" {
		t.Fatalf("the correction: %s", got)
	}
	states, _ := r.Revisions("ws", "c1", "2026-01")
	if states[0].State != StateSealed || states[1].State != StateSealed {
		t.Fatalf("revisions: %+v", states)
	}
	if got := reasons(seal(t, r, "c1", "2026-02", 1)); got != "sealed:" {
		t.Fatalf("judged again after the old period's correction: %s", got)
	}
	if got := reasons(seal(t, r, "c1", "2026-03", 1)); got != "sealed:" {
		t.Fatalf("judged again after the correction lowered the amortized part: %s", got)
	}
	write(t, r, "c1", "2026-04", 1, "0", "0", used("w", "acct-b", "0", "0", "1"))
	write(t, r, "c1", "2026-04", 2, "0", "0", used("v", "acct-b", "0", "0", "1"))
	if states, _ := r.Revisions("ws", "c1", "2026-04"); states[0].State != StateSuperseded || states[1].State != StateWritten {
		t.Fatalf("superseded: %+v", states)
	}
	if _, err := r.Seal("ws", "c1", "2026-04", 1); kind(err) != "rejected" {
		t.Fatalf("a superseded revision: %v", err)
	}
	write(t, r, "c1", "2026-05", 1, "0", "0", unused("u", "1", "0"))
	if got := reasons(seal(t, r, "c1", "2026-05", 1)); got != "held:cost_not_preserved" {
		t.Fatalf("held: %s", got)
	}
	if cur, _ := r.Current("ws", "c1", "2026-05"); cur.State != CurrentCorrectionHeld {
		t.Fatalf("a held highest revision: %+v", cur)
	}
	for _, month := range []string{"2026-04", "2026-05"} {
		if _, err := r.Revision("ws", "c1", month, len(mustStates(t, r, month))); kind(err) != "rejected" {
			t.Fatalf("revision of %s: %v", month, err)
		}
	}
}

func mustStates(t testing.TB, r *Registry, month string) []RevisionState {
	t.Helper()
	states, err := r.Revisions("ws", "c1", month)
	if err != nil {
		t.Fatal(err)
	}
	return states
}

func TestDuplicatesAcrossPayersAndProviders(t *testing.T) {
	r := open(t)
	record(t, r, commitmentOf("c1", PaymentNone, "0", RecoveryCharged))
	other := commitmentOf("c2", PaymentNone, "0", RecoveryCharged)
	other.PayerAccount = "payer-2"
	record(t, r, other)
	third := commitmentOf("c3", PaymentNone, "0", RecoveryCharged)
	record(t, r, third)
	write(t, r, "c1", "2026-01", 1, "1", "0", used("x", "acct-a", "1", "0", "2"))
	seal(t, r, "c1", "2026-01", 1)
	line := used("x", "acct-a", "1", "0", "2")
	line.Commitment = "c2"
	write(t, r, "c2", "2026-01", 1, "1", "0", line)
	if got := reasons(seal(t, r, "c2", "2026-01", 1)); got != "sealed:" {
		t.Fatalf("another payer account: %s", got)
	}
	line.Commitment = "c3"
	write(t, r, "c3", "2026-01", 1, "1", "0", line)
	if got := reasons(seal(t, r, "c3", "2026-01", 1)); got != "held:duplicate_source" {
		t.Fatalf("another commitment of the same payer: %s", got)
	}
}
