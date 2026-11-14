package cases

import (
	"testing"
	"time"
)

func TestFindingWithoutDifferenceRunsTheReviewFlow(t *testing.T) {
	r, clock := openWith(t, testLimits())
	f := Finding{ID: "f1", Kind: "new_charge", Target: Target{Customer: "c1", Contract: "k1", Resource: "vm-9"}, PeriodStart: "2026-10-01", PeriodEnd: "2026-10-31",
		Meaning: MeaningNone, Evidence: digest('a'), Severity: "medium", Owner: "ana", Source: SourceManual}
	v := must(r.RecordFinding("ws", f, op("rec")))
	if v.State != FindingInReview || v.Owner != "ana" || v.Author != "rec" || len(v.Digest) != 64 || !v.Written.Equal(start) || len(v.Steps) != 0 {
		t.Fatalf("recorded: %+v", v)
	}
	again := must(r.RecordFinding("ws", f, op("rec")))
	if again.Digest != v.Digest {
		t.Error("an identical repeat is not the same finding")
	}
	expect(t, "conflict", second(r.RecordFinding("ws", f, op("other"))), "another actor repeats")
	g := f
	g.Severity = "low"
	expect(t, "conflict", second(r.RecordFinding("ws", g, op("rec"))), "other content")

	expect(t, "rejected", second(r.ConfirmNormal("ws", "f1", op("rec"), because("s1", "expected_change"))), "the recorder confirms")
	expect(t, "rejected", second(r.Resolve("ws", "f1", op("rev"), Transition{Step: "s0", Evidence: digest('e'), Remedy: Remedy{Kind: "credit_document", Reference: "cd-1"}})), "resolve from review")
	clock.advance(time.Minute)
	v = must(r.RequireAdjustment("ws", "f1", op("rev"), because("s1", "missing_credit")))
	if v.State != FindingAdjustmentNeeded || len(v.Steps) != 1 || v.Steps[0].From != FindingInReview || v.Steps[0].To != FindingAdjustmentNeeded || v.Steps[0].Actor != "rev" || !v.Steps[0].At.Equal(start.Add(time.Minute)) {
		t.Fatalf("adjust: %+v", v)
	}
	if again := must(r.RequireAdjustment("ws", "f1", op("rev"), because("s1", "missing_credit"))); len(again.Steps) != 1 {
		t.Error("a repeated step moved again")
	}
	expect(t, "conflict", second(r.RequireAdjustment("ws", "f1", op("rev"), because("s1", "rate_error"))), "a step with other content")
	expect(t, "conflict", second(r.ConfirmNormal("ws", "f1", op("rev"), because("s1", "expected_change"))), "a step of another kind")
	expect(t, "rejected", second(r.ConfirmNormal("ws", "f1", op("rev"), because("s2", "expected_change"))), "confirm after adjustment")
	expect(t, "rejected", second(r.ReopenFinding("ws", "f1", op("rev"), because("s2", "resolution_failed"))), "resolution_failed before a resolution")

	remedy := Transition{Step: "s2", Evidence: digest('e'), Remedy: Remedy{Kind: "credit_document", Reference: "cd-1"}}
	v = must(r.Resolve("ws", "f1", op("rec"), remedy))
	if v.State != FindingResolved || v.Steps[1].Remedy.Reference != "cd-1" || v.Steps[1].Evidence != digest('e') {
		t.Fatalf("resolved: %+v", v)
	}
	v = must(r.ReopenFinding("ws", "f1", op("third"), because("s3", "resolution_failed")))
	if v.State != FindingInReview {
		t.Fatal(v.State)
	}
	expect(t, "rejected", second(r.ConfirmNormal("ws", "f1", op("third"), because("s4", "already_corrected"))), "the reopener confirms")
	v = must(r.ConfirmNormal("ws", "f1", op("rev"), because("s4", "already_corrected")))
	if v.State != FindingConfirmedNormal || len(v.Steps) != 4 {
		t.Fatalf("confirmed: %+v", v)
	}
	expect(t, "rejected", second(r.ReopenFinding("ws", "f1", op("rev"), because("s5", "resolution_failed"))), "resolution_failed after confirmation")
	v = must(r.ReopenFinding("ws", "f1", op("rev"), because("s5", "decision_revised")))
	v = must(r.RequireAdjustment("ws", "f1", op("fourth"), because("s6", "rate_error")))
	v = must(r.ReopenFinding("ws", "f1", op("rec"), because("s7", "decision_revised")))
	if v.State != FindingInReview {
		t.Fatal("adjustment_needed cannot be revised")
	}
	expect(t, "rejected", second(r.ReopenFinding("ws", "f1", op("rec"), because("s8", "new_evidence"))), "reopen in review")
	digests := map[string]bool{}
	for _, s := range v.Steps {
		if len(s.Digest) != 64 || digests[s.Digest] {
			t.Errorf("step digest %q", s.Digest)
		}
		digests[s.Digest] = true
	}
}

func second[T any](_ T, err error) error { return err }

func TestAssignmentAndStepRoom(t *testing.T) {
	r, _ := openWith(t, testLimits())
	must(r.RecordFinding("ws", spike("f", "c1"), op("rec")))
	for i, owner := range []string{"b", "c", "d"} {
		v := must(r.AssignFinding("ws", "f", op("lead"), Transition{Step: "a" + owner, Owner: owner}))
		if v.Owner != owner || v.Finding.Owner != "owner-1" || len(v.Steps) != i+1 || v.Steps[i].Owner != owner || v.State != FindingInReview {
			t.Fatalf("assign: %+v", v)
		}
	}
	must(r.ConfirmNormal("ws", "f", op("rev"), because("c1", "duplicate")))
	must(r.ReopenFinding("ws", "f", op("lead"), because("r1", "new_evidence")))
	expect(t, "limit", second(r.AssignFinding("ws", "f", op("lead"), Transition{Step: "a5", Owner: "e"})), "assign beyond the room")
	expect(t, "ok", second(r.ConfirmNormal("ws", "f", op("rev"), because("c2", "duplicate"))), "a decision after the room is used")
	expect(t, "limit", second(r.ReopenFinding("ws", "f", op("lead"), because("r2", "new_evidence"))), "reopen beyond the room")
	expect(t, "ok", second(r.AssignFinding("ws", "f", op("lead"), Transition{Step: "ab", Owner: "b"})), "an identical repeat is not counted")
}

func TestFindingForms(t *testing.T) {
	roomy := testLimits()
	roomy.Findings = 10
	r, _ := openWith(t, roomy)
	good := spike("f", "c1")
	bad := map[string]func(*Finding){
		"kind":        func(f *Finding) { f.Kind = "spike" },
		"no target":   func(f *Finding) { f.Target = Target{} },
		"line alone":  func(f *Finding) { f.Target.Line = "l" },
		"bad id":      func(f *Finding) { f.ID = "a b" },
		"period":      func(f *Finding) { f.PeriodEnd = "2026-09-30" },
		"start date":  func(f *Finding) { f.PeriodStart = "2026-02-30" },
		"meaning":     func(f *Finding) { f.Meaning = "price" },
		"no amount":   func(f *Finding) { f.Amount = "" },
		"no currency": func(f *Finding) { f.Currency = "" },
		"currency":    func(f *Finding) { f.Currency = "usd" },
		"negative":    func(f *Finding) { f.Amount = "-1" },
		"lead zero":   func(f *Finding) { f.Amount = "01" },
		"exponent":    func(f *Finding) { f.Amount = "1e3" },
		"evidence":    func(f *Finding) { f.Evidence = "x" },
		"severity":    func(f *Finding) { f.Severity = "urgent" },
		"owner":       func(f *Finding) { f.Owner = "" },
		"source":      func(f *Finding) { f.Source = "auto" },
		"detector":    func(f *Finding) { f.Detector = "d1" },
		"no detector": func(f *Finding) { f.Source = SourceDetector },
		"none amount": func(f *Finding) { f.Meaning = MeaningNone },
		"unattributed customer": func(f *Finding) {
			f.Kind = "unattributed"
		},
	}
	for name, change := range bad {
		f := good
		change(&f)
		expect(t, "invalid", second(r.RecordFinding("ws", f, op("rec"))), name)
	}
	ok := map[string]func(*Finding){
		"difference negative": func(f *Finding) { f.Meaning, f.Amount = MeaningDifference, "-0.25" },
		"detector":            func(f *Finding) { f.Source, f.Detector = SourceDetector, "spike-detector:v1" },
		"unattributed":        func(f *Finding) { f.Kind, f.Target = "unattributed", Target{Resource: "vm-1"} },
		"document line":       func(f *Finding) { f.Target.Document, f.Target.Line = "inv", "1" },
		"none":                func(f *Finding) { f.Meaning, f.Amount, f.Currency = MeaningNone, "", "" },
		"one day":             func(f *Finding) { f.PeriodEnd = f.PeriodStart },
	}
	index := 0
	for name, change := range ok {
		f := good
		f.ID = "ok" + string(rune('a'+index))
		index++
		change(&f)
		expect(t, "ok", second(r.RecordFinding("ws", f, op("rec"))), name)
	}
	expect(t, "invalid", second(r.RecordFinding("", good, op("rec"))), "workspace")
	expect(t, "invalid", second(r.RecordFinding("ws", good, Actor{ID: "x", Side: "admin"})), "side")
	expect(t, "invalid", second(r.RecordFinding("ws", good, Actor{ID: "x", Side: SideOperator, Customer: "c"})), "operator with a customer")
	expect(t, "invalid", second(r.RecordFinding("ws", good, Actor{ID: "x", Side: SideCustomer})), "customer without customer")
	must(r.RecordFinding("ws", good, op("rec")))
	for name, tr := range map[string]Transition{
		"no step":     {Reason: "duplicate", Evidence: digest('b')},
		"reason":      {Step: "s", Reason: "rate_error", Evidence: digest('b')},
		"no evidence": {Step: "s", Reason: "duplicate"},
		"with owner":  {Step: "s", Reason: "duplicate", Evidence: digest('b'), Owner: "o"},
		"with remedy": {Step: "s", Reason: "duplicate", Evidence: digest('b'), Remedy: Remedy{Kind: "credit_document", Reference: "x"}},
	} {
		expect(t, "invalid", second(r.ConfirmNormal("ws", "f", op("rev"), tr)), name)
	}
	for name, tr := range map[string]Transition{
		"no reference": {Step: "s", Evidence: digest('b'), Remedy: Remedy{Kind: "credit_document"}},
		"remedy kind":  {Step: "s", Evidence: digest('b'), Remedy: Remedy{Kind: "refund", Reference: "x"}},
		"reason":       {Step: "s", Reason: "duplicate", Evidence: digest('b'), Remedy: Remedy{Kind: "credit_document", Reference: "x"}},
		"no evidence":  {Step: "s", Remedy: Remedy{Kind: "credit_document", Reference: "x"}},
		"owner":        {Step: "s", Owner: "o", Evidence: digest('b'), Remedy: Remedy{Kind: "credit_document", Reference: "x"}},
	} {
		expect(t, "invalid", second(r.Resolve("ws", "f", op("rev"), tr)), name)
	}
	for name, tr := range map[string]Transition{
		"no owner": {Step: "s"},
		"reason":   {Step: "s", Owner: "o", Reason: "duplicate"},
		"evidence": {Step: "s", Owner: "o", Evidence: digest('b')},
		"remedy":   {Step: "s", Owner: "o", Remedy: Remedy{Kind: "credit_document", Reference: "x"}},
	} {
		expect(t, "invalid", second(r.AssignFinding("ws", "f", op("rev"), tr)), name)
	}
	expect(t, "invalid", second(r.ReopenFinding("ws", "f", op("rev"), because("s", "duplicate"))), "reopen reason")
	expect(t, "invalid", second(r.RequireAdjustment("ws", "f", op("rev"), because("s", "duplicate"))), "adjust reason")
	expect(t, "not_found", second(r.ConfirmNormal("ws", "none", op("rev"), because("s", "duplicate"))), "unknown finding")
	expect(t, "not_found", second(r.ConfirmNormal("other", "f", op("rev"), because("s", "duplicate"))), "other workspace")
}

func TestFindingsAreOperatorOnlyBeforeAnyLookup(t *testing.T) {
	r, _ := openWith(t, testLimits())
	must(r.RecordFinding("ws", spike("f", "c1"), op("rec")))
	c := cust("u", "c1")
	for name, err := range map[string]error{
		"record known":   second(r.RecordFinding("ws", spike("f", "c1"), c)),
		"record bad":     second(r.RecordFinding("ws", Finding{}, c)),
		"read known":     second(r.Finding("ws", "f", c)),
		"read unknown":   second(r.Finding("ws", "zz", c)),
		"read malformed": second(r.Finding("", "", c)),
		"confirm":        second(r.ConfirmNormal("ws", "f", c, because("s", "duplicate"))),
		"confirm bad":    second(r.ConfirmNormal("ws", "zz", c, Transition{})),
		"list":           second(r.Findings("ws", c, FindingFilter{})),
		"list bad":       second(r.Findings("", c, FindingFilter{State: "x"})),
		"disputes":       second(r.Disputes("ws", "c1", "inv", "", c)),
		"disputes bad":   second(r.Disputes("", "", "", "", c)),
	} {
		expect(t, "rejected", err, name)
	}
}

func TestFindingList(t *testing.T) {
	r, _ := openWith(t, testLimits())
	must(r.RecordFinding("ws", spike("b", "c1"), op("rec")))
	must(r.RecordFinding("ws", spike("a", "c2"), op("rec")))
	u := Finding{ID: "c", Kind: "unattributed", Target: Target{Resource: "vm"}, PeriodStart: "2026-10-01", PeriodEnd: "2026-10-01", Meaning: MeaningNone, Evidence: digest('a'), Severity: "low", Owner: "o", Source: SourceManual}
	must(r.RecordFinding("ws", u, op("rec")))
	must(r.RequireAdjustment("ws", "b", op("rev"), because("s", "rate_error")))
	ids := func(f FindingFilter) string {
		out := ""
		for _, v := range must(r.Findings("ws", op("x"), f)) {
			out += v.Finding.ID
		}
		return out
	}
	for want, f := range map[string]FindingFilter{
		"abc": {}, "ac": {State: FindingInReview}, "b": {State: FindingAdjustmentNeeded}, "a": {Customer: "c2"}, "c": {Unattributed: true},
		"ab": {Severity: "high"}, "bc": {After: "a"}, "": {After: "c"},
	} {
		if got := ids(f); got != want {
			t.Errorf("%+v: %q, want %q", f, got, want)
		}
	}
	expect(t, "invalid", second(r.Findings("ws", op("x"), FindingFilter{Unattributed: true, Customer: "c1"})), "both customer filters")
	expect(t, "invalid", second(r.Findings("ws", op("x"), FindingFilter{State: "open"})), "state")
	expect(t, "invalid", second(r.Findings("ws", op("x"), FindingFilter{Severity: "x"})), "severity")
	if v := must(r.Findings("none", op("x"), FindingFilter{})); v == nil || len(v) != 0 {
		t.Error("an unknown workspace lists nothing")
	}
	small := testLimits()
	small.Page = 2
	p, _ := openWith(t, small)
	for _, id := range []string{"z", "y", "x"} {
		must(p.RecordFinding("ws", spike(id, "c"), op("rec")))
	}
	if v := must(p.Findings("ws", op("x"), FindingFilter{})); len(v) != 2 || v[0].Finding.ID != "x" || v[1].Finding.ID != "y" {
		t.Errorf("page: %+v", v)
	}
	v := must(r.Finding("ws", "b", op("x")))
	v.Steps[0].Actor = "mutated"
	if must(r.Finding("ws", "b", op("x"))).Steps[0].Actor != "rev" {
		t.Error("a view shares the history")
	}
}
