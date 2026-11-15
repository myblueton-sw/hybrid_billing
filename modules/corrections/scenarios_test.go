package corrections

import (
	"errors"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
)

func TestDecreaseIsAppliedThroughAMatchingCredit(t *testing.T) {
	f := open(t, testLimits())
	f.issue(t, "ws", draft("o", "inv", "100", ""))
	v := must(f.r.Request("ws", request("c1", "o", "rev-7", "60.5"), human("requester")))
	if v.Difference != "-39.5" || v.Direction != DirectionCredit || v.Base != (Base{Effective: "100", Credits: 0, Digest: creditDigest(nil)}) || v.State != StateRequested || v.Digest != digestOf("ws", v) {
		t.Fatalf("request: %+v", v)
	}
	if e := must(f.r.EffectiveOf("ws", "o")); e.Effective != "100" || e.Open == nil || e.Open.Correction.ID != "c1" || e.Total != "100" {
		t.Fatalf("open: %+v", e)
	}
	expect(t, "self approval", second(f.r.Approve("ws", "c1", v.Digest, human("requester"))), RefusalSelfApproval)
	a := must(f.r.Approve("ws", "c1", v.Digest, human("checker")))
	if a.State != StateApproved || a.Approver != "checker" {
		t.Fatal(a)
	}
	f.clock.advance(time.Minute)
	f.credit(t, "cr1", "39.5", "o")
	applied := must(f.r.Apply("ws", "c1", "cr1", system("linker")))
	if applied.State != StateApplied || applied.Document != "cr1" || applied.Closer != "linker" || applied.Closed != start.Add(time.Minute) {
		t.Fatalf("apply: %+v", applied)
	}
	e := must(f.r.EffectiveOf("ws", "o"))
	if e.Effective != "60.5" || len(e.Applied) != 1 || e.Open != nil || len(e.Unlinked) != 0 {
		t.Fatalf("effective: %+v", e)
	}
	again := must(f.r.Request("ws", request("c9", "o", "rev-7", "60.5"), human("other")))
	if !again.Duplicate || again.Correction.ID != "c1" || again.State != StateApplied {
		t.Fatalf("reprocessing after application: %+v", again)
	}
	if must(f.r.EffectiveOf("ws", "o")).Effective != "60.5" {
		t.Fatal("reprocessing changed the amount")
	}
	second := must(f.r.Request("ws", request("c2", "o", "rev-8", "0"), human("requester")))
	if second.Base.Effective != "60.5" || second.Base.Credits != 1 || second.Difference != "-60.5" {
		t.Fatalf("second base: %+v", second)
	}
}

func TestReprocessingTheSameSourceIsDeduplicated(t *testing.T) {
	f := open(t, testLimits())
	f.issue(t, "ws", draft("o", "inv", "100", ""))
	first := must(f.r.Request("ws", request("c1", "o", "rev-1", "90"), human("requester")))
	again := must(f.r.Request("ws", request("c2", "o", "rev-1", "90"), human("other")))
	if !again.Duplicate || again.Correction.ID != "c1" || again.Digest != first.Digest {
		t.Fatalf("duplicate: %+v", again)
	}
	if must(f.r.Of("ws", "c1")).Duplicate {
		t.Fatal("the stored view is marked duplicate")
	}
	expect(t, "other target", second(f.r.Request("ws", request("c2", "o", "rev-1", "80"), human("requester"))), RefusalKeyReused)
	other := request("c2", "o", "rev-1", "90")
	other.Evidence = "ev-other"
	expect(t, "other evidence", second(f.r.Request("ws", other, human("requester"))), RefusalKeyReused)
	other = request("c2", "o", "rev-1", "90")
	other.Reason = ReasonPriceError
	expect(t, "other reason", second(f.r.Request("ws", other, human("requester"))), RefusalKeyReused)
	other = request("c2", "o", "rev-1", "90")
	other.Period.To = "2026-09-29"
	expect(t, "other period", second(f.r.Request("ws", other, human("requester"))), RefusalKeyReused)
	if _, err := f.r.Of("ws", "c2"); !errors.Is(err, ErrNotFound) {
		t.Fatal("a duplicate created a correction")
	}
	must(f.r.Withdraw("ws", "c1", human("requester")))
	fresh := must(f.r.Request("ws", request("c3", "o", "rev-1", "80"), human("requester")))
	if fresh.Duplicate || fresh.Correction.ID != "c3" {
		t.Fatal("a withdrawn key stayed reserved")
	}
}

func TestOutsideCreditsAreNeverHidden(t *testing.T) {
	f := open(t, testLimits())
	f.issue(t, "ws", draft("o", "inv", "100", ""))
	f.credit(t, "x1", "10", "o")
	e := must(f.r.EffectiveOf("ws", "o"))
	if e.Effective != "90" || len(e.Unlinked) != 1 || e.Unlinked[0] != "x1" {
		t.Fatalf("outside credit: %+v", e)
	}
	expect(t, "request with unlinked", second(f.r.Request("ws", request("c1", "o", "k", "50"), human("requester"))), RefusalUnlinked)
	expect(t, "self explanation", second(f.r.Explain("ws", "o", "x1", Explanation{Reason: ReasonOther, Evidence: "memo"}, human("issuer"))), RefusalSelfExplain)
	note := must(f.r.Explain("ws", "o", "x1", Explanation{Reason: ReasonOther, Evidence: "memo"}, human("auditor")))
	if note.Approver != "lead" || note.Issuer != "issuer" || note.Credit != "x1" {
		t.Fatalf("note: %+v", note)
	}
	v := must(f.r.Request("ws", request("c1", "o", "k", "50"), human("requester")))
	if v.Base.Effective != "90" || v.Difference != "-40" {
		t.Fatalf("base after explanation: %+v", v)
	}
	f.credit(t, "x2", "5", "o")
	expect(t, "approve after outside credit", second(f.r.Approve("ws", "c1", v.Digest, human("checker"))), RefusalBaseChanged)
	must(f.r.Withdraw("ws", "c1", human("requester")))
	must(f.r.Explain("ws", "o", "x2", Explanation{Reason: ReasonOther, Evidence: "memo2"}, human("auditor")))
	v = must(f.r.Request("ws", request("c2", "o", "k2", "50"), human("requester")))
	must(f.r.Approve("ws", "c2", v.Digest, human("checker")))
	f.credit(t, "x3", "1", "o")
	f.credit(t, "y", "35", "o")
	expect(t, "apply after outside credit", second(f.r.Apply("ws", "c2", "y", system("linker"))), RefusalBaseChanged)
	expect(t, "withdraw with unlinked", second(f.r.Withdraw("ws", "c2", human("requester"))), RefusalCreditIssued)
	for _, id := range []string{"x3", "y"} {
		must(f.r.Explain("ws", "o", id, Explanation{Reason: ReasonOther, Evidence: "memo-" + id}, human("auditor")))
	}
	must(f.r.Withdraw("ws", "c2", human("requester")))
	e = must(f.r.EffectiveOf("ws", "o"))
	if e.Effective != "49" || len(e.Notes) != 4 || e.Open != nil {
		t.Fatalf("after explaining: %+v", e)
	}
	must(f.r.Request("ws", request("c3", "o", "k3", "0"), human("requester")))
}

func TestIncreasesAreRefused(t *testing.T) {
	f := open(t, testLimits())
	f.issue(t, "ws", draft("o", "inv", "100", ""))
	expect(t, "increase", second(f.r.Request("ws", request("c1", "o", "k", "100.01"), human("requester"))), RefusalDebit)
	expect(t, "no change", second(f.r.Request("ws", request("c1", "o", "k", "100"), human("requester"))), RefusalNoChange)
	expect(t, "scale", second(f.r.Request("ws", request("c1", "o", "k", "99.999"), human("requester"))), RefusalScale)
	must(f.r.Request("ws", request("c1", "o", "k", "0"), human("requester")))
	expect(t, "pending", second(f.r.Request("ws", request("c2", "o", "k2", "50"), human("requester"))), RefusalPending)
}

func TestERPCreditsAndFinals(t *testing.T) {
	f := open(t, testLimits())
	f.issue(t, "ws", draft("e", "erp", "200", ""))
	f.approved(t, request("c1", "e", "k", "150"))
	f.issue(t, "ws", draft("ec", "erc", "50", "e"))
	must(f.r.Apply("ws", "c1", "ec", system("erp")))
	expect(t, "future", second(f.r.RecordFinal("ws", "e", Final{Reference: "erp-1", Amount: "150", Observed: start.Add(time.Hour)}, system("erp"))), RefusalFuture)
	matched := must(f.r.RecordFinal("ws", "e", Final{Reference: "erp-1", Amount: "150", Observed: start}, system("erp")))
	if matched.Outcome != OutcomeMatched || matched.Expected != "150" {
		t.Fatal(matched)
	}
	if must(f.r.RecordFinal("ws", "e", Final{Reference: "erp-1", Amount: "150", Observed: start}, system("erp"))).Outcome != OutcomeMatched {
		t.Fatal("repeat")
	}
	expect(t, "same reference other amount", second(f.r.RecordFinal("ws", "e", Final{Reference: "erp-1", Amount: "151", Observed: start}, system("erp"))), "conflict")
	expect(t, "same reference other actor", second(f.r.RecordFinal("ws", "e", Final{Reference: "erp-1", Amount: "150", Observed: start}, human("clerk"))), "conflict")
	expect(t, "scale", second(f.r.RecordFinal("ws", "e", Final{Reference: "erp-2", Amount: "1.001", Observed: start}, system("erp"))), RefusalScale)
	diverged := must(f.r.RecordFinal("ws", "e", Final{Reference: "erp-2", Amount: "149", Observed: start}, human("clerk")))
	if diverged.Outcome != OutcomeDiverged || diverged.Expected != "150" {
		t.Fatal(diverged)
	}
	must(f.r.Request("ws", request("c2", "e", "k2", "149"), human("requester")))
	e := must(f.r.EffectiveOf("ws", "e"))
	if e.Effective != "150" || len(e.Finals) != 2 || e.Finals[0].Outcome != OutcomeMatched || e.Finals[1].Outcome != OutcomeDiverged {
		t.Fatalf("finals: %+v", e.Finals)
	}
}

func TestCreditMatching(t *testing.T) {
	f := open(t, testLimits())
	f.issue(t, "ws", draft("o", "inv", "100", ""))
	f.issue(t, "ws", draft("o2", "inv", "100", ""))
	f.approved(t, request("c1", "o", "k", "70"))
	expect(t, "document of another original", second(f.r.Apply("ws", "c1", "o2", system("linker"))), RefusalBaseChanged)
	f.issue(t, "ws", draft("wrong", "crn", "29", "o"))
	expect(t, "amount", second(f.r.Apply("ws", "c1", "wrong", system("linker"))), RefusalAmount)
	must(f.r.Explain("ws", "o", "wrong", Explanation{Reason: ReasonOther, Evidence: "mistake"}, human("auditor")))
	expect(t, "linked by explanation", second(f.r.Apply("ws", "c1", "wrong", system("linker"))), RefusalLinked)
	must(f.r.Withdraw("ws", "c1", human("requester")))
	f.approved(t, request("c2", "o", "k2", "41"))
	other := draft("region", "ocr", "30", "o")
	f.issue(t, "ws", other)
	expect(t, "region", second(f.r.Apply("ws", "c2", "region", system("linker"))), RefusalMismatch)
	for label, change := range map[string]func(*issuance.Snapshot){
		"class": func(s *issuance.Snapshot) { s.Class = issuance.ClassInternal }, "tax": func(s *issuance.Snapshot) { s.Tax = "x" },
		"statutory": func(s *issuance.Snapshot) { s.Statutory = true }, "external": func(s *issuance.Snapshot) { s.External = true },
		"contract": func(s *issuance.Snapshot) { s.Draft.Contract = "k9" }, "customer": func(s *issuance.Snapshot) { s.Draft.Customer = "x" },
		"entity": func(s *issuance.Snapshot) { s.Draft.Entity = "x" }, "currency": func(s *issuance.Snapshot) { s.Draft.Currency = "USD" },
		"scale": func(s *issuance.Snapshot) { s.Draft.Scale = 3 },
	} {
		d := must(f.is.DocumentOf("ws", "region"))
		d.Snapshot.Class, d.Snapshot.Region, d.Snapshot.Tax = issuance.ClassSelf, "KR", "kr_vat"
		change(&d.Snapshot)
		f.docs.mu.Lock()
		f.docs.swap["region"] = d
		f.docs.mu.Unlock()
		expect(t, label, second(f.r.Apply("ws", "c2", "region", system("linker"))), RefusalMismatch)
	}
	d := must(f.is.DocumentOf("ws", "region"))
	d.Snapshot.Region, d.Snapshot.Tax = "KR", "kr_vat"
	f.docs.mu.Lock()
	f.docs.swap["region"] = d
	f.docs.mu.Unlock()
	if must(f.r.Apply("ws", "c2", "region", system("linker"))).State != StateApplied {
		t.Fatal("a matching credit")
	}
}
