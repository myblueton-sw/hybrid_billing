package accesschange

import (
	"reflect"
	"testing"
	"time"
)

func TestVersionAdvancesOnEveryChangeOfARequestOrRevocation(t *testing.T) {
	w := setup(t)
	last := w.r.Version()
	step := func(name string) {
		t.Helper()
		if now := w.r.Version(); now <= last {
			t.Errorf("%s did not advance the version: %d then %d", name, last, now)
		} else {
			last = now
		}
	}
	a := w.propose(proposal(add("carol"), alice))
	step("a proposal")
	b := w.propose(proposal(add("dave"), alice))
	step("a second proposal")
	c := w.propose(proposal(add("erin"), alice))
	step("a third proposal")
	if _, err := w.r.Approve(a.Number, "w1", bob, a.Digest, Standing{Basis: "basis-1", Allowed: true}, func(Applier) (string, error) { return "grant-x", nil }); err != nil {
		t.Fatal(err)
	}
	step("an approval")
	if _, err := w.r.Reject(b.Number, "w1", bob, true, "not now"); err != nil {
		t.Fatal(err)
	}
	step("a rejection")
	if _, err := w.r.Withdraw(c.Number, "w1", alice); err != nil {
		t.Fatal(err)
	}
	step("a withdrawal")
	e := &effects{ledger: &fakeLedger{seq: 4}}
	w.r.Restore(nil, Receipt{Sequence: 4, Watermark: "mark-e"}, 0, 0)
	step("a restore")
	if _, err := w.r.Revoke(target("g1"), e.build()); err != nil {
		t.Fatal(err)
	}
	step("a revocation")
	before := w.r.Version()
	w.r.Requests("w1")
	w.r.Revocations("w1")
	w.r.Request(a.Number, "w1")
	w.r.RequestPage("w1", 0, 5, start)
	w.r.RevocationPage("w1", "", 5)
	w.r.Counts("w1", start)
	if w.r.Version() != before {
		t.Error("a read advanced the version")
	}
}

func TestRequestPagesAreNewestFirstAndEvaluatedAtTheGivenTime(t *testing.T) {
	w := setup(t)
	var made []View
	for i, who := range []Actor{alice, bob, carol} {
		made = append(made, w.propose(proposal(add([]string{"dave", "erin", "frank"}[i]), who)))
	}
	other := proposal(add("gina"), alice)
	other.Workspace = "w2"
	elsewhere := w.propose(other)
	if _, err := w.r.Withdraw(made[1].Number, "w1", bob); err != nil {
		t.Fatal(err)
	}
	whole := w.r.Requests("w1")
	var walked []View
	for before := uint64(0); ; {
		page, more := w.r.RequestPage("w1", before, 2, start)
		walked = append(walked, page...)
		if !more {
			break
		}
		before = page[len(page)-1].Number
	}
	if len(walked) != len(whole) || len(walked) != 3 {
		t.Fatalf("walked %d of %d", len(walked), len(whole))
	}
	for i := range walked {
		if !reflect.DeepEqual(walked[i], whole[len(whole)-1-i]) {
			t.Errorf("row %d of the pages is not the whole list read newest first", i)
		}
		if walked[i].Number == elsewhere.Number {
			t.Error("another workspace's request is in the pages")
		}
	}
	if page, more := w.r.RequestPage("w1", made[0].Number, 5, start); len(page) != 0 || more {
		t.Errorf("before the oldest: %+v %v", page, more)
	}
	if page, more := w.r.RequestPage("w1", 0, 0, start); page != nil || more {
		t.Errorf("limit 0: %+v %v", page, more)
	}
	lapse := made[0].ExpiresAt
	early, _ := w.r.RequestPage("w1", 0, 5, lapse.Add(-time.Nanosecond))
	late, _ := w.r.RequestPage("w1", 0, 5, lapse)
	if early[2].State != StatePending || late[2].State != StateExpired || late[1].State != StateWithdrawn {
		t.Errorf("a pending request reads as expired from its expiry on, at the given time: %s then %s", early[2].State, late[2].State)
	}
	if now := w.r.Requests("w1"); now[0].State != StatePending {
		t.Error("the whole list is evaluated at the registry's own time")
	}
	if got := w.r.Counts("w1", start); got != (Counts{Pending: 2, Ended: 1}) {
		t.Errorf("counts at the start: %+v", got)
	}
	if got := w.r.Counts("w1", lapse); got != (Counts{Pending: 0, Ended: 3}) {
		t.Errorf("counts once the pending ones have lapsed: %+v", got)
	}
	if got := w.r.Counts("w2", start); got != (Counts{Pending: 1}) {
		t.Errorf("counts of the other workspace: %+v", got)
	}
}

func TestRevocationPagesAreNewestFirstAndRefuseACursorThatIsGone(t *testing.T) {
	w := setup(t)
	e := &effects{ledger: &fakeLedger{seq: 4}}
	w.r.Restore(nil, Receipt{Sequence: 4, Watermark: "mark-e"}, 0, 0)
	for _, id := range []string{"g1", "g2", "g3"} {
		if _, err := w.r.Revoke(target(id), e.build()); err != nil {
			t.Fatal(err)
		}
		w.advance(time.Minute)
	}
	whole := w.r.Revocations("w1")
	first, more, known := w.r.RevocationPage("w1", "", 2)
	if !known || !more || len(first) != 2 || !reflect.DeepEqual(first, whole[:2]) {
		t.Fatalf("the first page: %d rows, more %v", len(first), more)
	}
	rest, more, known := w.r.RevocationPage("w1", first[1].ID, 2)
	if !known || more || len(rest) != 1 || !reflect.DeepEqual(rest[0], whole[2]) {
		t.Fatalf("the second page: %d rows, more %v", len(rest), more)
	}
	if _, _, known := w.r.RevocationPage("w1", "revocation-999", 2); known {
		t.Error("a cursor naming no kept revocation was continued")
	}
	if _, _, known := w.r.RevocationPage("w2", first[0].ID, 2); known {
		t.Error("a cursor of another workspace was continued")
	}
	if page, _, known := w.r.RevocationPage("w2", "", 2); !known || len(page) != 0 {
		t.Errorf("another workspace's first page: %+v %v", page, known)
	}
	first[0].Ended = append(first[0].Ended, "tampered")
	if again, _, _ := w.r.RevocationPage("w1", "", 1); len(again[0].Ended) != len(whole[0].Ended) {
		t.Error("a page shares its rows with the registry")
	}
	oldest := whole[2].ID
	if _, err := w.r.Revoke(target("g4"), e.build()); err != nil {
		t.Fatal(err)
	}
	if kept := w.r.Revocations("w1"); len(kept) != 3 {
		t.Fatalf("kept %d", len(kept))
	}
	if _, _, known := w.r.RevocationPage("w1", oldest, 2); known {
		t.Error("a cursor naming a pruned revocation was continued")
	}
	if got := w.r.Counts("w1", start); got.Revocations != 3 {
		t.Errorf("revocations counted: %+v", got)
	}
}
