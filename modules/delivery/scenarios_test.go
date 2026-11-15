package delivery

import (
	"testing"
)

func TestAutomaticDeliveryOnAnOpenChannel(t *testing.T) {
	f := standard(t)
	v := must(f.r.Request("ws", Delivery{ID: "d1", Package: "p1", Channel: "mail-acme"}, system("scheduler")))
	if v.State != StateWaiting || v.Recipient != "acme" || v.FileDigest != "digest-p1" || v.Policy != "std" || v.Revision != 1 || v.Grant != 1 || v.PossibleDuplicate {
		t.Fatalf("request: %+v", v)
	}
	s := must(f.r.Begin("ws", "d1", system("worker")))
	if s.Number != 1 || s.Package != "p1" || s.Recipient != "acme" || s.Channel != "mail-acme" || s.FileDigest != "digest-p1" || len(s.Token) != 32 || s.PossibleDuplicate {
		t.Fatalf("start: %+v", s)
	}
	expect(t, "running", second(f.r.Begin("ws", "d1", system("worker"))), ReasonAttemptRunning)
	done := must(f.r.Report("ws", "d1", s.Token, Outcome{Result: ResultAccepted, Reference: "smtp-250"}, system("worker")))
	if done.State != StateAccepted || done.Attempts[0].Result != ResultAccepted || done.Attempts[0].Late {
		t.Fatalf("accepted: %+v", done)
	}
	confirmed := must(f.r.Confirm("ws", "d1", "read-receipt", system("worker")))
	if confirmed.State != StateConfirmed || confirmed.Receipt != "read-receipt" {
		t.Fatal(confirmed)
	}
	again := must(f.r.Request("ws", Delivery{ID: "d9", Package: "p1", Channel: "mail-acme"}, system("scheduler")))
	if !again.Duplicate || again.Delivery.ID != "d1" {
		t.Fatalf("dedup: %+v", again)
	}
}

func TestRetriesAndHonestOutcomes(t *testing.T) {
	f := standard(t)
	must(f.r.Request("ws", Delivery{ID: "d1", Package: "p1", Channel: "mail-acme"}, system("scheduler")))
	first := must(f.r.Begin("ws", "d1", system("worker")))
	v := must(f.r.Report("ws", "d1", first.Token, Outcome{Result: ResultRejected, Reference: "smtp-550"}, system("worker")))
	if v.State != StateWaiting || v.PossibleDuplicate {
		t.Fatalf("rejected: %+v", v)
	}
	retry := must(f.r.Begin("ws", "d1", system("worker")))
	must(f.r.Report("ws", "d1", retry.Token, Outcome{Result: ResultUnknown, Reference: "timeout"}, system("worker")))
	third := must(f.r.Begin("ws", "d1", system("worker")))
	if !third.PossibleDuplicate || third.Number != 3 {
		t.Fatalf("after unknown: %+v", third)
	}
	f.clock.advance(testLimits().Lease)
	expect(t, "exhausted", second(f.r.Begin("ws", "d1", system("worker"))), "limit")
	v = must(f.r.Of("ws", "d1"))
	if v.State != StateExhausted || !v.Attempts[2].Superseded || !v.PossibleDuplicate {
		t.Fatalf("exhausted: %+v", v)
	}
	late := must(f.r.Report("ws", "d1", third.Token, Outcome{Result: ResultAccepted, Reference: "smtp-250"}, system("worker")))
	if late.State != StateAccepted || !late.Attempts[2].Late {
		t.Fatalf("late acceptance: %+v", late)
	}
	dup := must(f.r.Request("ws", Delivery{ID: "d2", Package: "p1", Channel: "mail-acme"}, system("scheduler")))
	if !dup.Duplicate || dup.Delivery.ID != "d1" {
		t.Fatal("a system request after exhaustion must return the existing delivery")
	}
}

func TestPolicyChangeStopsDelivery(t *testing.T) {
	f := standard(t)
	must(f.r.Request("ws", Delivery{ID: "d1", Package: "p1", Channel: "mail-acme"}, system("scheduler")))
	first := must(f.r.Begin("ws", "d1", system("worker")))
	must(f.r.Report("ws", "d1", first.Token, Outcome{Result: ResultRejected, Reference: "busy"}, system("worker")))
	f.pkgs.lapse("ws", "p1", "policy_superseded")
	expect(t, "stopped", second(f.r.Begin("ws", "d1", system("worker"))), ReasonPackageNotLive)
	v := must(f.r.Of("ws", "d1"))
	if v.State != StateStopped || v.Reason != "policy_superseded" || len(v.Attempts) != 1 {
		t.Fatalf("stopped: %+v", v)
	}
	expect(t, "request lapsed", second(f.r.Request("ws", Delivery{ID: "d2", Package: "p1", Channel: "portal-acme"}, system("scheduler"))), ReasonPackageNotLive)
}

func TestClosedChannelNeedsApprovalAndERPIsManual(t *testing.T) {
	f := standard(t)
	must(f.r.Request("ws", Delivery{ID: "d1", Package: "p1", Channel: "mail-acme"}, human("requester")))
	must(f.r.Close("ws", "mail-acme", human("lead")))
	expect(t, "closed", second(f.r.Begin("ws", "d1", system("worker"))), ReasonNotApproved)
	expect(t, "self approval", second(f.r.Approve("ws", "d1", human("requester"))), ReasonSelfApproval)
	must(f.r.Approve("ws", "d1", human("lead")))
	must(f.r.Begin("ws", "d1", system("worker")))
	expect(t, "erp open", second(f.r.Open("ws", "erp-acme", 1, human("lead"))), ReasonERPGateUnbound)
	must(f.r.Request("ws", Delivery{ID: "e1", Package: "p2", Channel: "erp-acme"}, system("scheduler")))
	expect(t, "erp needs approval", second(f.r.Begin("ws", "e1", system("worker"))), ReasonNotApproved)
	must(f.r.Approve("ws", "e1", human("lead")))
	if must(f.r.Begin("ws", "e1", system("worker"))).Channel != "erp-acme" {
		t.Fatal("approved erp delivery")
	}
}

func TestChannelsAreBoundToTheirRecipient(t *testing.T) {
	f := standard(t)
	expect(t, "other recipient", second(f.r.Request("ws", Delivery{ID: "d1", Package: "p3", Channel: "mail-acme"}, system("scheduler"))), ReasonRecipientMismatch)
	must(f.r.Request("ws", Delivery{ID: "d2", Package: "p3", Channel: "mail-beta"}, system("scheduler")))
}

func TestAttachmentsAddedAutomaticallyBlockOpening(t *testing.T) {
	f := standard(t)
	d := proven()
	d.AddsAttachments = true
	v := must(f.r.Diagnose("ws", "portal-acme", d, human("tester")))
	expect(t, "adds attachments", second(f.r.Open("ws", "portal-acme", v.Revision, human("lead"))), ReasonAddsAttachments)
	v = must(f.r.Diagnose("ws", "portal-acme", Diagnosis{FieldControl: true, Evidence: Evidence{Kind: EvidenceChannelTest, Reference: "run-2"}}, human("tester")))
	expect(t, "attachment control", second(f.r.Open("ws", "portal-acme", v.Revision, human("lead"))), ReasonCapabilityUnproven)
	expect(t, "platform export", second(f.r.Diagnose("ws", "portal-acme", Diagnosis{FieldControl: true, AttachmentControl: true, Evidence: Evidence{Kind: EvidencePlatformExport, Reference: "x"}}, human("tester"))), ReasonExportEvidence)
	v = must(f.r.Diagnose("ws", "portal-acme", proven(), human("tester")))
	expect(t, "stale", second(f.r.Open("ws", "portal-acme", v.Revision-1, human("lead"))), ReasonStaleDiagnosis)
	expect(t, "self", second(f.r.Open("ws", "portal-acme", v.Revision, human("tester"))), ReasonSelfApproval)
	c := must(f.r.Open("ws", "portal-acme", v.Revision, human("lead")))
	if !c.Open || c.Opened != v.Revision {
		t.Fatal(c)
	}
	c = must(f.r.ChannelOf("ws", "mail-acme"))
	must(f.r.Diagnose("ws", "mail-acme", Diagnosis{FieldControl: true, AttachmentControl: true, Evidence: Evidence{Kind: EvidenceChannelTest, Reference: "run-9"}}, human("tester")))
	if must(f.r.ChannelOf("ws", "mail-acme")).Open || !c.Open {
		t.Fatal("a new diagnosis must close automatic delivery")
	}
}

func TestResendNeedsASecondHumanAndCarriesDuplicates(t *testing.T) {
	f := standard(t)
	must(f.r.Request("ws", Delivery{ID: "d1", Package: "p1", Channel: "mail-acme"}, system("scheduler")))
	s := must(f.r.Begin("ws", "d1", system("worker")))
	must(f.r.Report("ws", "d1", s.Token, Outcome{Result: ResultAccepted, Reference: "ok"}, system("worker")))
	expect(t, "resend reusing an identifier", second(f.r.Resend("ws", "d1", Delivery{ID: "d1"}, "customer_request", human("ops"))), "conflict")
	r := must(f.r.Resend("ws", "d1", Delivery{ID: "d2"}, "customer_request", human("ops")))
	if r.Resends != "d1" || !r.PossibleDuplicate || r.State != StateWaiting || r.Delivery.Package != "p1" {
		t.Fatalf("resend: %+v", r)
	}
	expect(t, "resend needs approval", second(f.r.Begin("ws", "d2", system("worker"))), ReasonNotApproved)
	expect(t, "self approval", second(f.r.Approve("ws", "d2", human("ops"))), ReasonSelfApproval)
	must(f.r.Approve("ws", "d2", human("lead")))
	if !must(f.r.Begin("ws", "d2", system("worker"))).PossibleDuplicate {
		t.Fatal("resend attempts are possible duplicates")
	}
	expect(t, "second resend while one is open", second(f.r.Resend("ws", "d1", Delivery{ID: "d3"}, "again", human("ops"))), ReasonNotResendable)
	must(f.r.Cancel("ws", "d2", human("ops")))
	fresh := must(f.r.Request("ws", Delivery{ID: "d4", Package: "p1", Channel: "mail-acme"}, system("scheduler")))
	if fresh.Duplicate || !fresh.PossibleDuplicate {
		t.Fatalf("after cancel: %+v", fresh)
	}
}

func TestLateAcceptanceAfterCancelKeepsThePairTaken(t *testing.T) {
	f := standard(t)
	must(f.r.Request("ws", Delivery{ID: "d1", Package: "p1", Channel: "mail-acme"}, system("scheduler")))
	a := must(f.r.Begin("ws", "d1", system("worker")))
	must(f.r.Cancel("ws", "d1", human("ops")))
	d2 := must(f.r.Request("ws", Delivery{ID: "d2", Package: "p1", Channel: "mail-acme"}, system("scheduler")))
	if !d2.PossibleDuplicate {
		t.Fatal("cancelling an attempt in flight marks the pair")
	}
	b := must(f.r.Begin("ws", "d2", system("worker")))
	if !b.PossibleDuplicate {
		t.Fatal("attempts after a cancelled in-flight attempt are possible duplicates")
	}
	late := must(f.r.Report("ws", "d1", a.Token, Outcome{Result: ResultAccepted, Reference: "late"}, system("worker")))
	if late.State != StateAccepted || !late.Attempts[0].Late {
		t.Fatalf("late acceptance after cancel: %+v", late)
	}
	must(f.r.Report("ws", "d2", b.Token, Outcome{Result: ResultAccepted, Reference: "ok"}, system("worker")))
	if !must(f.r.Of("ws", "d2")).PossibleDuplicate {
		t.Fatal("both accepted deliveries must be flagged")
	}
	g := standard(t)
	must(g.r.Request("ws", Delivery{ID: "d1", Package: "p1", Channel: "mail-acme"}, system("scheduler")))
	c := must(g.r.Begin("ws", "d1", system("worker")))
	must(g.r.Cancel("ws", "d1", human("ops")))
	must(g.r.Report("ws", "d1", c.Token, Outcome{Result: ResultAccepted, Reference: "late"}, system("worker")))
	again := must(g.r.Request("ws", Delivery{ID: "d3", Package: "p1", Channel: "mail-acme"}, system("scheduler")))
	if !again.Duplicate || again.Delivery.ID != "d1" {
		t.Fatalf("a late acceptance must take the pair back: %+v", again)
	}
}

func TestLateReportAfterStop(t *testing.T) {
	f := standard(t)
	must(f.r.Request("ws", Delivery{ID: "d1", Package: "p1", Channel: "mail-acme"}, system("scheduler")))
	a := must(f.r.Begin("ws", "d1", system("worker")))
	f.clock.advance(testLimits().Lease)
	f.pkgs.lapse("ws", "p1", "recipient_revoked")
	expect(t, "stopped after supersession", second(f.r.Begin("ws", "d1", system("worker"))), ReasonPackageNotLive)
	v := must(f.r.Report("ws", "d1", a.Token, Outcome{Result: ResultRejected, Reference: "late"}, system("worker")))
	if v.State != StateStopped || !v.Attempts[0].Late || !v.Attempts[0].Superseded || v.Reason != "recipient_revoked" {
		t.Fatalf("late report after stop: %+v", v)
	}
}
