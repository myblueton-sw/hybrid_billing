package server

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func TestFinancialERPResultAdviceOutputAdmission(t *testing.T) {
	w, p := resultWorld(t)
	w.f.s.documentERP = w.a
	f := w.f
	raw, err := json.Marshal(p)
	f.must(err)
	source := string(raw) + strings.Repeat(" ", 2100-len(raw))
	proposed := f.run(w.actors[Admin2Login], "result_evidence.propose", "bounded-advice", map[string]string{"packet": source})
	f.ok(t, "bounded proposal", proposed, "pending", "")
	v := financialERPResultView{Basis: financialERPResultBasis{ID: "bounded-advice"}, Review: financialDocumentReview{Digest: field(proposed, "digest")}}
	before := f.s.Revision()
	f.s.cfg.Commands.Value = 2000
	_, ready := resultAdvice(t, w, Admin3Login, v)
	for _, action := range []string{"confirm", "revoke"} {
		if ready[action].State != "blocked" || ready[action].Code != commands.PrerequisiteFailed {
			t.Fatal(ready)
		}
	}
	if f.s.Revision() != before || len(w.a.resultApplications) != 0 {
		t.Fatal("advice mutated state")
	}
}

func TestFinancialERPResultAdviceOtherAuthorizedActorCannotRecover(t *testing.T) {
	w, p := resultWorld(t)
	w.f.s.documentERP = w.a
	v := confirmResult(t, w, proposeResult(t, w, p, "actor-advice"))
	w.a.afterResultWrite = func(string) error { return errors.New("interrupted") }
	_, err := applyResult(w, v)
	if !errors.Is(err, errFinancialERPResultUncertain) {
		t.Fatal(err)
	}
	w.a.afterResultWrite = nil
	resultGrantTo(t, w.f, "apply", SampleAdmin)
	resultGrantTo(t, w.f, "inspect", SampleAdmin)
	refreshResultActors(w)
	before := w.f.s.Revision()
	app := w.a.resultApplications[[2]string{SampleWorkspace, v.Basis.ID}]
	_, ready := resultAdvice(t, w, AdminLogin, v)
	if ready["recover"].State != "blocked" || ready["recover"].Code != commands.EvidenceStale {
		t.Fatal(ready)
	}
	after := w.a.resultApplications[[2]string{SampleWorkspace, v.Basis.ID}]
	if w.f.s.Revision() != before || after.actor != app.actor || after.receipt != app.receipt || after.digest != app.digest {
		t.Fatal("advice altered ownership")
	}
}

func resultAdvice(t *testing.T, w documentWorld, login string, v financialERPResultView) (reply, map[string]documentAdviceState) {
	t.Helper()
	got := documentRead(w.f, w.actors[login], "result_evidence.advice", v.Basis.ID, map[string]string{"digest": v.Review.Digest})
	if got.status != 200 {
		t.Fatal(got.raw)
	}
	var readiness map[string]documentAdviceState
	w.f.must(json.Unmarshal([]byte(field(got, "readiness")), &readiness))
	return got, readiness
}

func TestFinancialERPResultAdviceUnknownRecoveryIsReadOnly(t *testing.T) {
	w, p := resultWorld(t)
	w.f.s.documentERP = w.a
	v := confirmResult(t, w, proposeResult(t, w, p, "advice"))
	_, before := resultAdvice(t, w, Admin2Login, v)
	if before["apply"].State != "ready" || before["recover"].State != "inapplicable" || len(w.a.resultApplications) != 0 {
		t.Fatal(before)
	}
	w.a.afterResultWrite = func(string) error { return errors.New("interrupted") }
	_, err := applyResult(w, v)
	if !errors.Is(err, errFinancialERPResultUncertain) {
		t.Fatal(err)
	}
	w.a.afterResultWrite = nil
	read, ready := resultAdvice(t, w, Admin2Login, v)
	if ready["recover"].State != "ready" || ready["apply"].State != "inapplicable" || field(read, "application_state") != "unknown" || field(read, "applying_actor") != SampleAdmin2 {
		t.Fatal(ready)
	}
	if w.a.resultApplications[[2]string{SampleWorkspace, v.Basis.ID}].receipt != "" {
		t.Fatal("advice settled uncertainty")
	}
	_, err = applyResult(w, v)
	w.f.must(err)
	_, done := resultAdvice(t, w, Admin2Login, v)
	if done["recover"].State != "inapplicable" || done["apply"].State != "inapplicable" {
		t.Fatal(done)
	}
}

func TestFinancialERPResultAdviceMismatchConfirmationAndHeldRevoke(t *testing.T) {
	w, p := resultWorld(t)
	w.f.s.documentERP = w.a
	p.Rows[0].Net = "99"
	v := proposeResult(t, w, p, "mismatch-advice")
	_, err := decideDocument(w, Admin3Login, financialDocumentView{Basis: w.basis, Review: financialDocumentReview{Digest: w.digest}}, true, "Withdraw document")
	w.f.must(err)
	_, pending := resultAdvice(t, w, Admin3Login, v)
	if pending["confirm"].State != "ready" {
		t.Fatal(pending)
	}
	v = confirmResult(t, w, v)
	_, blocked := resultAdvice(t, w, Admin2Login, v)
	if blocked["apply"].State != "blocked" {
		t.Fatal(blocked)
	}
	w.f.s.enterHold("advice")
	w.f.s.chargeProducer = nil
	_, held := resultAdvice(t, w, Admin3Login, v)
	if held["revoke"].State != "ready" || held["confirm"].State != "inapplicable" {
		t.Fatal(held)
	}
	if len(w.a.resultApplications) != 0 {
		t.Fatal("advice created effects")
	}
}

func TestFinancialERPResultAdviceRejectsConflictingRecovery(t *testing.T) {
	w, p := resultWorld(t)
	w.f.s.documentERP = w.a
	v := confirmResult(t, w, proposeResult(t, w, p, "conflict-advice"))
	w.a.afterResultWrite = func(string) error { return errors.New("interrupted") }
	_, err := applyResult(w, v)
	if !errors.Is(err, errFinancialERPResultUncertain) {
		t.Fatal(err)
	}
	w.a.afterResultWrite = nil
	w.a.ledger = resultLookupLedger{financialERPLedger: w.a.ledger, unavailable: true}
	_, unavailable := resultAdvice(t, w, Admin2Login, v)
	if unavailable["recover"].State != "unavailable" || unavailable["recover"].Code != commands.DependencyUnavailable {
		t.Fatal(unavailable)
	}
	w.a.ledger = w.ledger
	p.Revision = 2
	confirmResult(t, w, proposeResult(t, w, p, "newer-advice"))
	_, stale := resultAdvice(t, w, Admin2Login, v)
	if stale["recover"].State != "blocked" || stale["recover"].Code != commands.EvidenceStale {
		t.Fatal(stale)
	}
}
