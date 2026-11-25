package approvals

import (
	"maps"
	"reflect"
	"slices"
	"testing"
	"time"
)

func checkUnchanged(t *testing.T, r *Registry, id string, d Decision, actor Actor, want string) {
	t.Helper()
	before := must(r.RequestOf("ws", id))
	req := r.workspaces["ws"].requests[id]
	excluded, deciders, approvers := maps.Clone(req.excluded), maps.Clone(req.deciders), slices.Clone(req.approvers)
	expect(t, want, r.CheckDecision("ws", id, d, actor), "preflight")
	after := must(r.RequestOf("ws", id))
	if !reflect.DeepEqual(before, after) || !reflect.DeepEqual(excluded, req.excluded) || !reflect.DeepEqual(deciders, req.deciders) || !reflect.DeepEqual(approvers, req.approvers) {
		t.Fatal("preflight changed recorded decision state")
	}
}

func TestCheckDecisionCurrentSeparationAndExactCoverage(t *testing.T) {
	for _, amount := range []string{"1000", "1000.00000001", "-1000", "-1000.00000001"} {
		t.Run(amount, func(t *testing.T) {
			r, _ := setup(t, 1)
			must(r.Request("ws", subject("r", "target", amount), human("requester")))
			want := "ok"
			if amount == "1000.00000001" || amount == "-1000.00000001" {
				want = ReasonNoAuthority
			}
			checkUnchanged(t, r, "r", approve, human("ann"), want)
			for _, actor := range []Actor{human("writer"), human("requester")} {
				checkUnchanged(t, r, "r", approve, actor, ReasonSelfApproval)
			}
			checkUnchanged(t, r, "r", approve, Actor{ID: "ann", Kind: KindSystem}, ReasonAINotAllowed)
			checkUnchanged(t, r, "r", Decision{Verdict: VerdictReject, Reason: "other"}, human("unfunded"), "ok")
			checkUnchanged(t, r, "r", Decision{Verdict: VerdictReject, Reason: "invented"}, human("ann"), "invalid")
		})
	}
}

func TestCheckDecisionAuthorityChangesAndClock(t *testing.T) {
	for _, mode := range []string{"revoked", "expired", "future", "currency", "stage"} {
		t.Run(mode, func(t *testing.T) {
			r, c := setup(t, 1)
			must(r.Request("ws", subject("r", "target", "15"), human("requester")))
			checkUnchanged(t, r, "r", approve, human("ann"), "ok")
			must(r.Revoke("ws", "g-ann", human("adm1")))
			if mode != "revoked" {
				p := grant("replacement", "ann", StageIssue, "1000")
				switch mode {
				case "expired":
					p.Authority.To = "2026-11-14"
				case "future":
					p.Authority.From = "2026-11-16"
				case "currency":
					p.Authority.Currency = "EUR"
				case "stage":
					p.Authority.Stage = StageCharge
				}
				administer(t, r, p)
			}
			if mode == "expired" {
				c.advance(48 * time.Hour)
			}
			checkUnchanged(t, r, "r", approve, human("ann"), ReasonNoAuthority)
			if mode == "expired" {
				c.advance(-48 * time.Hour)
				checkUnchanged(t, r, "r", approve, human("ann"), ReasonNoAuthority)
			}
			checkUnchanged(t, r, "r", Decision{Verdict: VerdictReject, Reason: "other"}, human("ann"), "ok")
		})
	}
}

func TestCheckDecisionDoesNotReuseHistoricalReplay(t *testing.T) {
	for _, count := range []int{1, 2} {
		r, _ := setup(t, count)
		must(r.Request("ws", subject("r", "target", "15"), human("requester")))
		checkUnchanged(t, r, "r", approve, human("ann"), "ok")
		recorded := must(r.Decide("ws", "r", approve, human("ann")))
		checkUnchanged(t, r, "r", approve, human("ann"), "conflict")
		must(r.Revoke("ws", "g-ann", human("adm1")))
		if got := must(r.Decide("ws", "r", approve, human("ann"))); !reflect.DeepEqual(got, recorded) {
			t.Fatal("historical replay changed")
		}
		want := ReasonNotPending
		if count == 2 {
			want = "ok"
		}
		checkUnchanged(t, r, "r", approve, human("bob"), want)
	}
	for _, terminal := range []string{"reject", "supersede", "withdraw"} {
		r, _ := setup(t, 1)
		must(r.Request("ws", subject("r", "target", "15"), human("requester")))
		switch terminal {
		case "reject":
			must(r.Decide("ws", "r", Decision{Verdict: VerdictReject, Reason: "other"}, human("ann")))
		case "supersede":
			must(r.Request("ws", subject("new", "target", "15"), human("requester")))
		case "withdraw":
			must(r.Withdraw("ws", "r", human("requester")))
		}
		checkUnchanged(t, r, "r", approve, human("bob"), ReasonNotPending)
	}
}

func TestCheckDecisionExclusionsRemainDetached(t *testing.T) {
	r, _ := setup(t, 2)
	s := subject("r", "target", "1000")
	s.Entries = []Entry{{Item: "a", Scope: "one", Amount: "1500", Digest: digest('c')}, {Item: "b", Scope: "two", Amount: "-500", Digest: digest('d')}}
	must(r.Request("ws", s, human("requester")))
	checkUnchanged(t, r, "r", approve, human("ann"), ReasonNoAuthority)
	d := Decision{Verdict: VerdictApprove, Exclude: []Exclusion{{Item: "a", Reason: "data_missing"}}}
	checkUnchanged(t, r, "r", d, human("ann"), "ok")
	d.Exclude[0].Reason = "blocked_error"
	checkUnchanged(t, r, "r", d, human("ann"), "ok")
	all := Decision{Verdict: VerdictApprove, Exclude: []Exclusion{{Item: "a", Reason: "data_missing"}, {Item: "b", Reason: "data_missing"}}}
	checkUnchanged(t, r, "r", all, human("ann"), ReasonNothingLeft)
	unknown := Decision{Verdict: VerdictApprove, Exclude: []Exclusion{{Item: "missing", Reason: "data_missing"}}}
	checkUnchanged(t, r, "r", unknown, human("ann"), "not_found")
	must(r.Decide("ws", "r", d, human("ann")))
	checkUnchanged(t, r, "r", unknown, human("ann"), "not_found")
	expect(t, "not_found", second(r.Decide("ws", "r", unknown, human("ann"))), "membership before replay")
	checkUnchanged(t, r, "r", approve, human("bob"), "ok")
	expect(t, "not_found", r.CheckDecision("foreign", "r", approve, human("ann")), "workspace")
	expect(t, "invalid", r.CheckDecision("", "r", approve, human("ann")), "invalid workspace")
}
