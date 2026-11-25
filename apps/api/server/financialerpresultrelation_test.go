package server

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func TestFinancialERPResultApplicationContext(t *testing.T) {
	w, p := resultWorld(t)
	first := confirmResult(t, w, proposeResult(t, w, p, "first"))
	read := func(v financialERPResultView) financialERPResultContext {
		raw, err := w.a.resultApplicationContext(v.Basis)
		w.f.must(err)
		var c financialERPResultContext
		w.f.must(json.Unmarshal([]byte(raw), &c))
		return c
	}
	if c := read(first); c.Kind != "initial" || c.Position != "not_started" || c.CurrentState != "none" {
		t.Fatal(c)
	}
	_, err := applyResult(w, first)
	w.f.must(err)
	if c := read(first); c.Position != "current" || c.CurrentState != "recorded" || c.CurrentResult != "first" {
		t.Fatal(c)
	}
	p.Revision = 2
	next := confirmResult(t, w, proposeResult(t, w, p, "second"))
	if c := read(next); c.Kind != "replacement" || c.Predecessor != "first" || c.PredecessorRevision != 1 || !validPlanHash(c.PredecessorReceipt) || c.Position != "not_started" {
		t.Fatal(c)
	}
	w.a.afterResultWrite = func(string) error { return errors.New("interrupted") }
	_, err = applyResult(w, next)
	if !errors.Is(err, errFinancialERPResultUncertain) {
		t.Fatal(err)
	}
	if c := read(next); c.Position != "current" || c.CurrentState != "unknown" {
		t.Fatal(c)
	}
	if c := read(first); c.Position != "historical" || c.CurrentResult != "second" || c.CurrentState != "unknown" {
		t.Fatal(c)
	}
	w.a.afterResultWrite = nil
	_, err = applyResult(w, next)
	w.f.must(err)
	if c := read(first); c.Position != "historical" || c.CurrentState != "recorded" {
		t.Fatal(c)
	}
	if c := read(next); c.Position != "current" || c.CurrentState != "recorded" {
		t.Fatal(c)
	}
	w.a.resultPreparations[[2]string{SampleWorkspace, p.Preparation}] = "first"
	_, err = w.a.resultApplicationContext(next.Basis)
	documentFailure(t, err, commands.EvidenceStale)
}

func TestFinancialERPResultUnknownPredecessorContext(t *testing.T) {
	w, p := resultWorld(t)
	first := confirmResult(t, w, proposeResult(t, w, p, "first"))
	w.a.afterResultWrite = func(string) error { return errors.New("interrupted") }
	_, err := applyResult(w, first)
	if !errors.Is(err, errFinancialERPResultUncertain) {
		t.Fatal(err)
	}
	p.Revision = 2
	next := confirmResult(t, w, proposeResult(t, w, p, "second"))
	raw, err := w.a.resultApplicationContext(next.Basis)
	w.f.must(err)
	var c financialERPResultContext
	w.f.must(json.Unmarshal([]byte(raw), &c))
	if c.Predecessor != "first" || c.PredecessorReceipt != "" || c.CurrentState != "unknown" || c.Position != "not_started" {
		t.Fatal(c)
	}
	_, err = applyResult(w, next)
	documentFailure(t, err, commands.EvidenceStale)
}

func TestFinancialERPResultInterruptedApplyInvalidatesEarlierReview(t *testing.T) {
	w, _, digest := publicResult(t)
	f := w.f
	actor := w.actors[Admin2Login]
	earlierReview := chargeBody(f, "earlier-reviewed-request", "public-result", map[string]string{"digest": digest})
	before := f.s.Revision()
	w.a.afterResultWrite = func(string) error { return errors.New("response lost after write") }
	interruptedBody := chargeBody(f, "interrupted-reviewed-request", "public-result", map[string]string{"digest": digest})
	interrupted := f.call("erp_result.apply", interruptedBody, actor.cookie, actor.csrf)
	f.code(t, "interrupted application", interrupted, 500, "internal_error")
	if operation, ok := interrupted.body["operation_id"].(string); !ok || operation == "" {
		t.Fatal("missing uncertain operation reference")
	}
	afterUnknown := f.s.Revision()
	f.code(t, "identical unknown replay", f.call("erp_result.apply", interruptedBody, actor.cookie, actor.csrf), 500, "internal_error")
	if f.s.Revision() != afterUnknown {
		t.Fatal("identical replay advanced revision")
	}

	w.a.afterResultWrite = nil
	if f.s.Revision() == before {
		t.Fatal("reserved application did not invalidate earlier review revision")
	}
	f.code(t, "stale reviewed request", f.call("erp_result.apply", earlierReview, actor.cookie, actor.csrf), 409, string(commands.RevisionConflict))
	if app := w.a.resultApplications[[2]string{SampleWorkspace, "public-result"}]; app.receipt != "" {
		t.Fatal("stale review settled application")
	}
}
