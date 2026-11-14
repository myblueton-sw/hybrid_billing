package claims

import (
	"reflect"
	"testing"
	"time"
)

func TestDirectAndResaleEligibility(t *testing.T) {
	r, _ := openWith(t)
	eligible(t, r, "direct-1", 1, direct())
	eligible(t, r, "resale-1", 1, resale())
	if reason := refused(t, r, usage("d-usage", "direct-1", "line-1", "10", "5")); reason != ReasonNotEligible {
		t.Fatalf("CSP usage under a direct contract: %s", reason)
	}
	reserved(t, r, monthly("d-fee", "direct-1", "fee-1", "2026-03-01", "2026-04-01", "100"))
	reserved(t, r, usage("r-usage", "resale-1", "line-2", "10", "5"))
	reserved(t, r, monthly("r-fee", "resale-1", "fee-1", "2026-03-01", "2026-04-01", "100"))
	support := monthly("r-support", "resale-1", "s", "2026-03-01", "2026-04-01", "10")
	support.Scope.Component = "support"
	if reason := refused(t, r, support); reason != ReasonNotEligible {
		t.Fatalf("a component the MSP may not bill: %s", reason)
	}
	missing := monthly("r-none", "resale-1", "x", "2026-03-01", "2026-04-01", "10")
	missing.Scope.Component = "none"
	if reason := refused(t, r, missing); reason != ReasonNotEligible {
		t.Fatalf("an absent component: %s", reason)
	}
	if _, err := r.RecordEligibility("ws", "direct-2", 1, Revision{From: "2026-01-01", To: "2027-01-01", Path: PathDirect,
		Components: []Component{comp("usage", KindCSPUsage, true, RoutePlatform, "main")}}, "author-1"); kind(err) != "invalid" {
		t.Fatalf("billable CSP usage under a direct contract: %v", err)
	}
}

func TestExternalAndUnknownRoutes(t *testing.T) {
	r, _ := openWith(t)
	rev := resale()
	rev.Components[0].Route, rev.Components[0].Reference = RouteExternal, "erp-order-1"
	rev.Components[1].Route = RouteUnknown
	eligible(t, r, "c1", 1, rev)
	for index, id := range []string{"submit-1", "submit-2"} {
		a := usage(id, "c1", "line-1", "10", "5")
		a.Calculation = []string{"calc-1", "calc-2"}[index]
		if reason := refused(t, r, a); reason != ReasonRoutedExternally {
			t.Fatalf("%s: %s", id, reason)
		}
	}
	if reason := refused(t, r, monthly("m", "c1", "fee-1", "2026-03-01", "2026-04-01", "100")); reason != ReasonRouteUnknown {
		t.Fatalf("an unknown route: %s", reason)
	}
	if _, err := r.Application("ws", "submit-1"); kind(err) != "not_found" {
		t.Fatal("a refusal stored")
	}
}

func TestAConfirmedMonthIsNeverBilledAgain(t *testing.T) {
	r, _ := openWith(t)
	eligible(t, r, "c1", 1, resale())
	billed(t, r, monthly("m3", "c1", "fee-1", "2026-03-01", "2026-04-01", "100"))
	again := monthly("m3-again", "c1", "fee-1", "2026-03-15", "2026-04-15", "100")
	again.Calculation = "calc-2"
	if d, err := r.Reserve("ws", again, "another-client"); err != nil || d.Reason != ReasonAlreadyClaimed {
		t.Fatalf("a new key, client and calculation: %v %+v", err, d)
	}
	reserved(t, r, monthly("m4", "c1", "fee-1", "2026-04-01", "2026-05-01", "100"))
	reserved(t, r, monthly("m3-second", "c1", "fee-2", "2026-03-01", "2026-04-01", "100"))
	renamed := monthly("m3-renamed", "c1", "fee-9", "2026-03-01", "2026-04-01", "100")
	if reason := refused(t, r, renamed); reason != ReasonUnknownInstance {
		t.Fatalf("an undeclared instance: %s", reason)
	}
	other := monthly("m5-eur", "c1", "fee-1", "2026-05-01", "2026-06-01", "100")
	other.Currency = "EUR"
	if reason := refused(t, r, other); reason != ReasonCurrencyMismatch {
		t.Fatalf("another currency: %s", reason)
	}
	sourced := usage("as-source", "c1", "line-1", "1", "1")
	sourced.Scope = Scope{Contract: "c1", Component: "fee", Instance: "fee-1"}
	if reason := refused(t, r, sourced); reason != ReasonModeMismatch {
		t.Fatalf("another mode: %s", reason)
	}
	cover, _ := r.Coverage("ws", Scope{"c1", "fee", "fee-1"})
	if len(cover) != 2 || cover[0].Application.ID != "m3" || cover[1].Application.ID != "m4" {
		t.Fatalf("coverage: %+v", cover)
	}
}

func TestPartialConsumptionOfASource(t *testing.T) {
	r, _ := openWith(t)
	eligible(t, r, "c1", 1, resale())
	billed(t, r, usage("u1", "c1", "line-1", "60", "30.00"))
	billed(t, r, usage("u2", "c1", "line-1", "40", "20"))
	if reason := refused(t, r, usage("u3", "c1", "line-1", "0.0001", "0")); reason != ReasonCapacityExceeded {
		t.Fatalf("over capacity: %s", reason)
	}
	reverse(t, r, "rev-1", "u1", "10", "5")
	if reason := refused(t, r, usage("u3", "c1", "line-1", "10.0001", "5")); reason != ReasonCapacityExceeded {
		t.Fatalf("over the freed part: %s", reason)
	}
	rebill := usage("u3", "c1", "line-1", "10", "5")
	rebill.CorrectsKind, rebill.Corrects = CorrectsReversal, "rev-1"
	billed(t, r, rebill)
	src, _ := r.Source("ws", "line-1")
	want := []Remaining{{"u1", Scope{"c1", "usage", "main"}, "50", "25"}, {"u2", Scope{"c1", "usage", "main"}, "40", "20"}, {"u3", Scope{"c1", "usage", "main"}, "10", "5"}}
	if !reflect.DeepEqual(src.Remaining, want) || src.UsedQuantity != "100" || src.UsedAmount != "50" || src.Quantity != "100" {
		t.Fatalf("source: %+v", src)
	}
	if v, _ := r.Application("ws", "u1"); v.State != StateConfirmed || v.ReversedQuantity != "10" || v.ReversedAmount != "5" {
		t.Fatalf("partly reversed: %+v", v)
	}
	if reason := refused(t, r, usage("u4", "c1", "line-1", "1", "1")); reason != ReasonCapacityExceeded {
		t.Fatalf("full again: %s", reason)
	}
	mismatch := usage("u5", "c1", "line-1", "1", "1")
	mismatch.CapQuantity = "200"
	if reason := refused(t, r, mismatch); reason != ReasonCapacityMismatch {
		t.Fatalf("another capacity: %s", reason)
	}
	eligible(t, r, "c2", 1, resale())
	if reason := refused(t, r, usage("u6", "c2", "line-1", "0", "0")); reason != ReasonSourceInOtherScope {
		t.Fatalf("a source in another scope: %s", reason)
	}
	reserved(t, r, usage("u7", "c1", "line-2", "100", "50"))
	two := usage("u8", "c1", "line-3", "1", "1")
	reserved(t, r, two)
}

func TestWholeReversalsInIntervalMode(t *testing.T) {
	r, _ := openWith(t)
	eligible(t, r, "c1", 1, resale())
	billed(t, r, monthly("m3", "c1", "fee-1", "2026-03-01", "2026-04-01", "100"))
	if _, err := r.Reverse("ws", Reversal{ID: "half", Application: "m3", Quantity: "1", Amount: "50"}, "clerk-1"); kind(err) != "rejected" {
		t.Fatalf("a partial reversal of an interval: %v", err)
	}
	if _, err := r.Reverse("ws", Reversal{ID: "w1", Application: "m3", Quantity: "1", Amount: "100"}, "clerk-1"); err != nil {
		t.Fatal(err)
	}
	if reason := refused(t, r, monthly("m3-b", "c1", "fee-1", "2026-03-01", "2026-04-01", "120")); reason != ReasonAlreadyClaimed {
		t.Fatalf("a pending reversal frees nothing: %s", reason)
	}
	if _, err := r.Reverse("ws", Reversal{ID: "w2", Application: "m3", Quantity: "1", Amount: "100"}, "clerk-2"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ApproveReversal("ws", "w1", "clerk-1"); kind(err) != "rejected" {
		t.Fatalf("approved by its author: %v", err)
	}
	if _, err := r.ApproveReversal("ws", "w1", "lead-1"); err != nil {
		t.Fatal(err)
	}
	if v, err := r.ApproveReversal("ws", "w1", "lead-1"); err != nil || v.State != ReversalEffective {
		t.Fatal("a repeat approval")
	}
	if _, err := r.ApproveReversal("ws", "w1", "lead-2"); kind(err) != "conflict" {
		t.Fatal("another approver")
	}
	if _, err := r.ApproveReversal("ws", "w2", "lead-1"); kind(err) != "rejected" {
		t.Fatalf("a second whole reversal: %v", err)
	}
	if v, _ := r.Application("ws", "m3"); v.State != StateReversed {
		t.Fatalf("reversed: %+v", v)
	}
	if _, err := r.Reverse("ws", Reversal{ID: "w3", Application: "m3", Quantity: "1", Amount: "100"}, "clerk-1"); kind(err) != "rejected" {
		t.Fatal("reversing a reversed application")
	}
	correction := monthly("m3-b", "c1", "fee-1", "2026-03-01", "2026-04-01", "120")
	correction.CorrectsKind, correction.Corrects = CorrectsApplication, "m3"
	billed(t, r, correction)
	if v, _ := r.Application("ws", "m3-b"); v.Application.Corrects != "m3" {
		t.Fatal("corrects recorded")
	}
	sourced := usage("as-source", "c1", "line-1", "1", "1")
	sourced.Scope = Scope{Contract: "c1", Component: "fee", Instance: "fee-2"}
	reserved(t, r, sourced)
	revs, _ := r.Reversals("ws", "m3")
	if len(revs) != 2 || revs[0].State != ReversalEffective || revs[1].State != ReversalPending {
		t.Fatalf("reversals: %+v", revs)
	}
}

func TestReplacementExpiryAndRelease(t *testing.T) {
	r, clock := openWith(t)
	eligible(t, r, "c1", 1, resale())
	reserved(t, r, monthly("cand-1", "c1", "fee-1", "2026-03-01", "2026-04-01", "100"))
	next := monthly("cand-2", "c1", "fee-1", "2026-03-01", "2026-04-01", "110")
	next.Replaces, next.Calculation = "cand-1", "calc-2"
	reserved(t, r, next)
	if v, _ := r.Application("ws", "cand-1"); v.State != StateReplaced || v.ReplacedBy != "cand-2" {
		t.Fatalf("replaced: %+v", v)
	}
	if d := reserve(t, r, next); d.Application.State != StateReserved {
		t.Fatal("an identical repeat after the replacement")
	}
	third := monthly("cand-3", "c1", "fee-1", "2026-03-01", "2026-04-01", "120")
	third.Replaces = "cand-1"
	if _, err := r.Reserve("ws", third, "runner-1"); kind(err) != "rejected" {
		t.Fatalf("replacing a replaced candidate: %v", err)
	}
	across := monthly("cand-4", "c1", "fee-2", "2026-03-01", "2026-04-01", "120")
	across.Replaces = "cand-2"
	if _, err := r.Reserve("ws", across, "runner-1"); kind(err) != "rejected" {
		t.Fatalf("replacing across scopes: %v", err)
	}
	confirm(t, r, "cand-2")
	late := monthly("cand-5", "c1", "fee-1", "2026-03-01", "2026-04-01", "120")
	late.Replaces = "cand-2"
	if _, err := r.Reserve("ws", late, "runner-1"); kind(err) != "rejected" {
		t.Fatalf("replacing a confirmed application: %v", err)
	}
	if _, err := r.Release("ws", "cand-2", "runner-1"); kind(err) != "rejected" {
		t.Fatal("releasing a confirmed application")
	}
	reserved(t, r, monthly("apr", "c1", "fee-1", "2026-04-01", "2026-05-01", "100"))
	clock.advance(3599 * time.Second)
	if v, _ := r.Application("ws", "apr"); v.State != StateReserved {
		t.Fatal("before expiry")
	}
	clock.advance(time.Second)
	if v, _ := r.Application("ws", "apr"); v.State != StateExpired || !v.Expiry.Equal(start.Add(time.Hour)) {
		t.Fatalf("at expiry: %+v", v)
	}
	if _, refusal, err := r.Confirm("ws", []string{"apr"}, "runner-1"); kind(err) != "rejected" || refusal.Application != "apr" || refusal.Reason != "" {
		t.Fatalf("confirming an expired reservation: %v %+v", err, refusal)
	}
	if _, err := r.Release("ws", "apr", "runner-1"); kind(err) != "rejected" {
		t.Fatal("releasing an expired reservation")
	}
	expiredTarget := monthly("apr-2", "c1", "fee-1", "2026-04-01", "2026-05-01", "100")
	expiredTarget.Replaces = "apr"
	if _, err := r.Reserve("ws", expiredTarget, "runner-1"); kind(err) != "rejected" {
		t.Fatal("replacing an expired reservation")
	}
	reserved(t, r, monthly("apr-3", "c1", "fee-1", "2026-04-01", "2026-05-01", "100"))
	v, err := r.Release("ws", "apr-3", "runner-9")
	if err != nil || v.State != StateReleased || v.ReleasedBy != "runner-9" {
		t.Fatalf("release: %v %+v", err, v)
	}
	if v, err := r.Release("ws", "apr-3", "runner-1"); err != nil || v.ReleasedBy != "runner-9" {
		t.Fatal("a repeat release")
	}
	reserved(t, r, monthly("apr-4", "c1", "fee-1", "2026-04-01", "2026-05-01", "100"))
}

func TestBatchConfirmationAllOrNone(t *testing.T) {
	r, clock := openWith(t)
	eligible(t, r, "c1", 1, resale())
	reserved(t, r, monthly("a", "c1", "fee-1", "2026-03-01", "2026-04-01", "100"))
	short := monthly("b", "c1", "fee-1", "2026-04-01", "2026-05-01", "100")
	short.Lifetime = 10
	reserved(t, r, short)
	reserved(t, r, monthly("c", "c1", "fee-1", "2026-05-01", "2026-06-01", "100"))
	clock.advance(10 * time.Second)
	if _, refusal, err := r.Confirm("ws", []string{"a", "b", "c"}, "runner-1"); kind(err) != "rejected" || refusal.Application != "b" {
		t.Fatalf("one expired: %v %+v", err, refusal)
	}
	for _, id := range []string{"a", "c"} {
		if v, _ := r.Application("ws", id); v.State != StateReserved {
			t.Fatalf("%s changed", id)
		}
	}
	if _, _, err := r.Confirm("ws", []string{"a", "a"}, "runner-1"); kind(err) != "invalid" {
		t.Fatal("an identifier twice")
	}
	if _, _, err := r.Confirm("ws", []string{"a", "none"}, "runner-1"); kind(err) != "not_found" {
		t.Fatal("an unknown identifier")
	}
	out, _, err := r.Confirm("ws", []string{"a", "c"}, "runner-1")
	if err != nil || len(out) != 2 || out[0].State != StateConfirmed || out[1].ConfirmedBy != "runner-1" {
		t.Fatalf("confirm: %v %+v", err, out)
	}
	if out, _, err := r.Confirm("ws", []string{"c", "a"}, "runner-2"); err != nil || out[0].ConfirmedBy != "runner-1" {
		t.Fatal("a repeat by another actor")
	}
	if _, _, err := r.Confirm("ws", []string{"a", "b", "c", "d", "e"}, "runner-1"); kind(err) != "limit" {
		t.Fatal("batch size")
	}
}

func TestSupersedingRevisions(t *testing.T) {
	r, _ := openWith(t)
	eligible(t, r, "c1", 1, resale())
	billed(t, r, monthly("feb", "c1", "fee-1", "2026-02-01", "2026-03-01", "100"))
	reserved(t, r, monthly("jul", "c1", "fee-1", "2026-07-01", "2026-08-01", "100"))
	renaming := resale()
	renaming.From = "2026-02-01"
	renaming.Components[1].Instances = []string{"fee-9"}
	if _, err := r.RecordEligibility("ws", "c1", 2, renaming, "author-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ApproveEligibility("ws", "c1", 2, "approver-1"); kind(err) != "rejected" {
		t.Fatalf("a revision renaming a billed instance: %v", err)
	}
	external := resale()
	external.From, external.To = "2026-06-01", "2026-12-01"
	external.Components[1].Route, external.Components[1].Reference = RouteExternal, "erp-9"
	if _, err := r.RecordEligibility("ws", "c1", 3, external, "author-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ApproveEligibility("ws", "c1", 3, "approver-1"); kind(err) != "rejected" {
		t.Fatalf("a revision ending before the one it supersedes: %v", err)
	}
	if v, _ := r.Eligibility("ws", "c1", 1); v.EffectiveTo != "2027-01-01" || v.Approver != "approver-1" {
		t.Fatalf("unchanged: %+v", v)
	}
}

func TestSupersessionAndConfirm(t *testing.T) {
	r, _ := openWith(t)
	eligible(t, r, "c1", 1, resale())
	billed(t, r, monthly("feb", "c1", "fee-1", "2026-02-01", "2026-03-01", "100"))
	reserved(t, r, monthly("jul", "c1", "fee-1", "2026-07-01", "2026-08-01", "100"))
	r.Release("ws", "jul", "runner-1")
	reserved(t, r, monthly("jul-b", "c1", "fee-2", "2026-07-01", "2026-08-01", "100"))
	later := resale()
	later.From = "2026-06-01"
	later.Components[1].Instances = []string{"fee-1", "fee-2"}
	later.Components[1].Route, later.Components[1].Reference = RoutePlatform, ""
	later.Components[0].Route, later.Components[0].Reference = RouteExternal, "erp-9"
	eligible(t, r, "c1", 2, later)
	if v, _ := r.Eligibility("ws", "c1", 1); v.EffectiveTo != "2026-06-01" || v.Revision.To != "2027-01-01" {
		t.Fatalf("superseded: %+v", v)
	}
	reserved(t, r, usage("u-mar", "c1", "line-1", "1", "1"))
	jun := usage("u-jun", "c1", "line-2", "1", "1")
	jun.Start, jun.End = "2026-06-01", "2026-07-01"
	if reason := refused(t, r, jun); reason != ReasonRoutedExternally {
		t.Fatalf("after the route changed: %s", reason)
	}
	across := monthly("across", "c1", "fee-1", "2026-05-15", "2026-06-15", "100")
	if reason := refused(t, r, across); reason != ReasonNoEligibility {
		t.Fatalf("an interval across two revisions: %s", reason)
	}
	confirm(t, r, "jul-b")
	moved := resale()
	moved.From = "2026-03-01"
	moved.Components[0].Route, moved.Components[0].Reference = RouteUnknown, ""
	moved.Components[1].Instances = []string{"fee-1", "fee-2"}
	if _, err := r.RecordEligibility("ws", "c1", 3, moved, "author-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ApproveEligibility("ws", "c1", 3, "approver-1"); kind(err) != "rejected" {
		t.Fatalf("an overlap with a later from date: %v", err)
	}
}

func TestConfirmJudgesEligibilityAgain(t *testing.T) {
	r, _ := openWith(t)
	eligible(t, r, "c1", 1, resale())
	reserved(t, r, monthly("feb", "c1", "fee-1", "2026-02-01", "2026-03-01", "100"))
	reserved(t, r, usage("u-jul", "c1", "line-1", "1", "1"))
	later := resale()
	later.From = "2026-02-01"
	later.Components[1].Route, later.Components[1].Reference = RouteExternal, "erp-1"
	if _, err := r.RecordEligibility("ws", "c1", 2, later, "author-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ApproveEligibility("ws", "c1", 2, "approver-1"); kind(err) != "rejected" {
		t.Fatalf("a reservation keeps its route: %v", err)
	}
	r.Release("ws", "feb", "runner-1")
	if _, err := r.ApproveEligibility("ws", "c1", 2, "approver-1"); err != nil {
		t.Fatal(err)
	}
	r2, _ := openWith(t)
	eligible(t, r2, "c1", 1, resale())
	reserved(t, r2, monthly("a", "c1", "fee-1", "2026-01-01", "2026-02-01", "100"))
	reserved(t, r2, monthly("b", "c1", "fee-1", "2026-02-15", "2026-03-15", "100"))
	later2 := resale()
	later2.From = "2026-03-01"
	eligible(t, r2, "c1", 2, later2)
	if _, refusal, err := r2.Confirm("ws", []string{"a", "b"}, "runner-1"); kind(err) != "rejected" || refusal != (Refusal{"b", ReasonNoEligibility}) {
		t.Fatalf("a reservation across the new boundary: %v %+v", err, refusal)
	}
	if v, _ := r2.Application("ws", "a"); v.State != StateReserved {
		t.Fatal("nothing changes")
	}
	confirm(t, r2, "a")
}

func TestEligibilityRecordAndApprove(t *testing.T) {
	r, _ := openWith(t)
	if _, err := r.RecordEligibility("ws", "c1", 2, resale(), "author-1"); kind(err) != "rejected" {
		t.Fatal("a revision beyond the next")
	}
	first, err := r.RecordEligibility("ws", "c1", 1, resale(), "author-1")
	if err != nil || first.Approver != "" {
		t.Fatal(err)
	}
	reordered := resale()
	reordered.Components = []Component{reordered.Components[2], reordered.Components[0], comp("fee", KindMSPService, true, RoutePlatform, "fee-2", "fee-1")}
	if again, err := r.RecordEligibility("ws", "c1", 1, reordered, "author-1"); err != nil || again.Digest != first.Digest {
		t.Fatalf("a repeat in another order: %v", err)
	}
	if _, err := r.RecordEligibility("ws", "c1", 1, resale(), "author-2"); kind(err) != "conflict" {
		t.Fatal("another author")
	}
	changed := resale()
	changed.Components[2].Billable = true
	if _, err := r.RecordEligibility("ws", "c1", 1, changed, "author-1"); kind(err) != "conflict" {
		t.Fatal("another content")
	}
	if reason := refused(t, r, monthly("m", "c1", "fee-1", "2026-03-01", "2026-04-01", "1")); reason != ReasonNoEligibility {
		t.Fatalf("an unapproved revision: %s", reason)
	}
	if _, err := r.ApproveEligibility("ws", "c1", 1, "author-1"); kind(err) != "rejected" {
		t.Fatal("approved by its author")
	}
	if _, err := r.ApproveEligibility("ws", "c1", 1, "approver-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ApproveEligibility("ws", "c1", 1, "approver-2"); kind(err) != "conflict" {
		t.Fatal("another approver")
	}
	if v, err := r.ApproveEligibility("ws", "c1", 1, "approver-1"); err != nil || v.ApprovedAt != start {
		t.Fatal("a repeat approval")
	}
	same := resale()
	if _, err := r.RecordEligibility("ws", "c1", 2, same, "author-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ApproveEligibility("ws", "c1", 2, "approver-1"); kind(err) != "rejected" {
		t.Fatal("an overlap with the same from date")
	}
	if reason := refused(t, r, monthly("nc", "none", "fee-1", "2026-03-01", "2026-04-01", "1")); reason != ReasonNoEligibility {
		t.Fatalf("a contract without revisions: %s", reason)
	}
	if reason := refused(t, r, monthly("late", "c1", "fee-1", "2026-12-15", "2027-01-15", "1")); reason != ReasonNoEligibility {
		t.Fatalf("an interval past the revision: %s", reason)
	}
	edge := monthly("edge", "c1", "fee-1", "2026-12-01", "2027-01-01", "1")
	reserved(t, r, edge)
	jan := monthly("jan", "c1", "fee-1", "2026-01-01", "2026-01-02", "1")
	reserved(t, r, jan)
}

func TestAZeroChargeStillClaimsItsScope(t *testing.T) {
	r, _ := openWith(t)
	eligible(t, r, "c1", 1, resale())
	free := monthly("free", "c1", "fee-1", "2026-03-01", "2026-04-01", "0")
	free.Quantity = "0"
	billed(t, r, free)
	if v, _ := r.Application("ws", "free"); v.State != StateConfirmed {
		t.Fatalf("a zero charge: %+v", v)
	}
	if reason := refused(t, r, monthly("paid", "c1", "fee-1", "2026-03-01", "2026-04-01", "100")); reason != ReasonAlreadyClaimed {
		t.Fatalf("the month billed again: %s", reason)
	}
	none := usage("u0", "c1", "line-1", "0", "0")
	billed(t, r, none)
	eligible(t, r, "c2", 1, resale())
	if reason := refused(t, r, usage("u1", "c2", "line-1", "1", "1")); reason != ReasonSourceInOtherScope {
		t.Fatalf("a zero application keeps its source: %s", reason)
	}
	reverse(t, r, "rv0", "free", "0", "0")
	if v, _ := r.Application("ws", "free"); v.State != StateReversed {
		t.Fatalf("reversed with an effective reversal: %+v", v)
	}
}
