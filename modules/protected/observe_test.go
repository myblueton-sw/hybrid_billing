package protected

import (
	"reflect"
	"testing"
	"time"
)

func stored(f fixture, id string) View { return copyView(f.r.workspaces["w1"].changes[id].view) }

func TestPeekProjectsLapseWithoutSettling(t *testing.T) {
	f := ready(t)
	effective := start.Add(5 * day)
	requested := must(f.r.Request("w1", account("c1", ref("v1", 'a'), effective), human("clerk")))
	other := account("c2", ref("v2", 'a'), start.Add(12*day))
	other.Field = FieldTaxIdentifier
	confirmed := must(f.r.Request("w1", other, human("clerk")))
	confirmed = must(f.r.Verify("w1", "c2", confirm("c2", "ch-phone", confirmed.Digest), human("checker")))
	expiry := confirmed.Verification.At.Add(2 * day)
	if !expiry.Before(other.Effective) || !expiry.Before(effective) {
		t.Fatal("fixture deadlines")
	}

	f.clock.advance(expiry.Sub(f.clock.read()))
	if got := must(f.r.Peek("w1", "c2")); !reflect.DeepEqual(got, confirmed) {
		t.Fatalf("at verification expiry %+v", got)
	}
	f.clock.advance(time.Nanosecond)
	got := must(f.r.Peek("w1", "c2"))
	want := copyView(confirmed)
	want.State, want.Ended = StateLapsed, expiry
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("after verification expiry %+v", got)
	}
	if !reflect.DeepEqual(stored(f, "c2"), confirmed) {
		t.Fatal("peek settled a confirmed change")
	}
	got.Verification.Verifier, got.Decisions = "changed", append(got.Decisions, DecisionView{})
	if !reflect.DeepEqual(stored(f, "c2"), confirmed) {
		t.Fatal("peek returned shared state")
	}

	f.clock.advance(effective.Sub(f.clock.read()))
	if got := must(f.r.Peek("w1", "c1")); !reflect.DeepEqual(got, requested) {
		t.Fatalf("at effective %+v", got)
	}
	f.clock.advance(time.Nanosecond)
	trail := must(f.r.PeekTrail("w1", "acme", FieldPaymentAccount))
	want = copyView(requested)
	want.State, want.Ended = StateLapsed, effective
	if len(trail) != 1 || !reflect.DeepEqual(trail[0], want) {
		t.Fatalf("after effective %+v", trail)
	}
	if !reflect.DeepEqual(stored(f, "c1"), requested) || !reflect.DeepEqual(stored(f, "c2"), confirmed) {
		t.Fatal("trail observation settled stored changes")
	}

	if settled := must(f.r.Of("w1", "c1")); !reflect.DeepEqual(settled, want) || !reflect.DeepEqual(stored(f, "c1"), want) {
		t.Fatalf("first stored settlement %+v", settled)
	}
	if again := must(f.r.Peek("w1", "c1")); !reflect.DeepEqual(again, want) {
		t.Fatalf("peek after settlement %+v", again)
	}
}

func TestPeekMatchesSettledReads(t *testing.T) {
	f := ready(t)
	f.schedule(t, account("c1", ref("v1", 'a'), start.Add(5*day)))
	must(f.r.Request("w1", account("c2", ref("v2", 'a'), start.Add(6*day)), human("clerk")))
	must(f.r.Withdraw("w1", "c2", human("clerk")))
	pending := must(f.r.Request("w1", account("c3", ref("v3", 'a'), start.Add(7*day)), human("clerk")))
	pending = must(f.r.Verify("w1", "c3", confirm("c3", "ch-phone", pending.Digest), human("checker")))
	must(f.r.Approve("w1", "c3", Approval{Verdict: VerdictReject, Reason: "other", Digest: pending.Verification.Digest}, human("boss")))
	must(f.r.Request("w1", account("c4", ref("v4", 'a'), start.Add(8*day)), human("clerk")))
	f.clock.advance(20 * day)

	seen := must(f.r.PeekTrail("w1", "acme", FieldPaymentAccount))
	if len(seen) != 4 || stored(f, "c4").State != StateRequested {
		t.Fatalf("trail %+v", seen)
	}
	for _, v := range seen {
		if one := must(f.r.Peek("w1", v.Change.ID)); !reflect.DeepEqual(one, v) {
			t.Fatalf("peek differs for %s", v.Change.ID)
		}
	}
	if actual := must(f.r.Trail("w1", "acme", FieldPaymentAccount)); !reflect.DeepEqual(actual, seen) {
		t.Fatalf("settled trail differs %+v", actual)
	}
	for _, v := range seen {
		if actual := must(f.r.Of("w1", v.Change.ID)); !reflect.DeepEqual(actual, v) {
			t.Fatalf("settled view differs for %s", v.Change.ID)
		}
	}
	if states := []string{seen[0].State, seen[1].State, seen[2].State, seen[3].State}; !reflect.DeepEqual(states, []string{StateScheduled, StateWithdrawn, StateRejected, StateLapsed}) {
		t.Fatalf("states %v", states)
	}
}

func TestPeekRefusals(t *testing.T) {
	f := ready(t)
	expect(t, "missing change", second(f.r.Peek("w1", "none")), "not_found")
	expect(t, "missing workspace", second(f.r.Peek("w9", "c1")), "not_found")
	expect(t, "trail missing workspace", second(f.r.PeekTrail("w9", "acme", FieldPaymentAccount)), "not_found")
	expect(t, "trail workspace", second(f.r.PeekTrail("", "acme", FieldPaymentAccount)), "invalid")
	expect(t, "trail subject", second(f.r.PeekTrail("w1", "bad subject", FieldPaymentAccount)), "invalid")
	expect(t, "trail field", second(f.r.PeekTrail("w1", "acme", "other")), "invalid")
	if empty := must(f.r.PeekTrail("w1", "nobody", FieldPaymentAccount)); len(empty) != 0 {
		t.Fatal("empty trail")
	}
}
