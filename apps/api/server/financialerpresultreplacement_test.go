package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/erpledger"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func TestFinancialERPResultReplacementHistoryAndRecovery(t *testing.T) {
	for _, interrupted := range []bool{false, true} {
		t.Run(fmt.Sprint(interrupted), func(t *testing.T) {
			w, p := resultWorld(t)
			w.f.s.documentERP = w.a
			history := []financialERPResultView{}
			for revision := 1; revision <= 4; revision++ {
				p.Revision = revision
				for i := range p.Rows {
					p.Rows[i].ID = fmt.Sprintf("revision-%d-row-%d", revision, i)
				}
				v := confirmResult(t, w, proposeResult(t, w, p, fmt.Sprintf("replacement-%d", revision)))
				if interrupted && revision > 1 {
					w.a.afterResultWrite = func(stage string) error {
						if stage == "replacement" {
							return errors.New("lost replacement response")
						}
						return nil
					}
					_, err := applyResult(w, v)
					if !errors.Is(err, errFinancialERPResultUncertain) {
						t.Fatalf("replacement interruption: %v", err)
					}
					_, unknown, err := resultEvidenceCounts(w.f.s)
					w.f.must(err)
					if unknown[SampleAdmin2] != 1 {
						t.Fatal("unknown successor not counted")
					}
					w.a.afterResultWrite = nil
				}
				got, err := applyResult(w, v)
				w.f.must(err)
				history = append(history, got)
				actual, err := w.ledger.ResultOf(SampleWorkspace, p.Preparation)
				w.f.must(err)
				if len(actual.Documents) != 1 || actual.Documents[0].Document.Revision != revision || len(actual.Documents[0].Links) != revision*len(p.Rows) || !actual.Reconciliation.Matched || len(actual.Submission.Commands) != 0 {
					t.Fatal("replacement lost complete history")
				}
				for i, l := range actual.Documents[0].Links {
					if l.Stale != (i < (revision-1)*len(p.Rows)) {
						t.Fatal("incorrect stale prefix")
					}
				}
				replay, err := applyResult(w, v)
				w.f.must(err)
				if replay.ApplicationReceipt != got.ApplicationReceipt {
					t.Fatal("current replay changed receipt")
				}
				for _, old := range history {
					app, owned, err := w.a.resultApplicationOwner(old.Basis, old.Review.Digest)
					w.f.must(err)
					if !owned || app.receipt != old.ApplicationReceipt || app.after == nil {
						t.Fatal("lost historical evidence")
					}
					read, err := resultAs(w, Admin2Login, func(actor string) (financialERPResultView, error) {
						return w.a.inspectResultLocked(SampleWorkspace, old.Basis.ID, old.Review.Digest, actor)
					})
					w.f.must(err)
					if read.ApplicationReceipt != old.ApplicationReceipt {
						t.Fatal("inspection rewrote receipt")
					}
					if old.Basis.ID != v.Basis.ID {
						_, err = applyResult(w, old)
						documentFailure(t, err, commands.EvidenceStale)
					}
				}
				_, unknown, err := resultEvidenceCounts(w.f.s)
				w.f.must(err)
				if len(unknown) != 0 {
					t.Fatal("completed replacement counted unknown")
				}
			}
		})
	}
}

type replacementFailureLedger struct {
	financialERPLedger
	invoked int
	fail    bool
}

func (l *replacementFailureLedger) ReplaceResult(workspace, key string, before erpledger.ResultView, document erpledger.Document, links []erpledger.Link, actor erpledger.Actor) (erpledger.ResultView, error) {
	l.invoked++
	if l.fail {
		return erpledger.ResultView{}, errors.New("before publication")
	}
	return l.financialERPLedger.ReplaceResult(workspace, key, before, document, links, actor)
}

func TestFinancialERPResultReplacementBeforeWriteOwnership(t *testing.T) {
	w, p := resultWorld(t)
	first := confirmResult(t, w, proposeResult(t, w, p, "first"))
	_, err := applyResult(w, first)
	w.f.must(err)
	before, err := w.ledger.ResultOf(SampleWorkspace, p.Preparation)
	w.f.must(err)
	p.Revision = 2
	next := confirmResult(t, w, proposeResult(t, w, p, "second"))
	ledger := &replacementFailureLedger{financialERPLedger: w.a.ledger, fail: true}
	w.a.ledger = ledger
	_, err = applyResult(w, next)
	if !errors.Is(err, errFinancialERPResultUncertain) {
		t.Fatal(err)
	}
	actual, err := w.ledger.ResultOf(SampleWorkspace, p.Preparation)
	w.f.must(err)
	if !reflect.DeepEqual(actual, before) || len(w.a.resultApplications) != 2 {
		t.Fatal("reservation or before state lost")
	}
	ledger.fail = false
	_, err = applyResult(w, next)
	w.f.must(err)
	if ledger.invoked != 2 {
		t.Fatal("before recovery did not retry exactly once")
	}
	_, err = applyResult(w, next)
	w.f.must(err)
	if ledger.invoked != 2 {
		t.Fatal("recorded replay repeated mutation")
	}
}

func TestFinancialERPResultReplacementChainCorruption(t *testing.T) {
	for _, mode := range []string{"missing-after", "altered-after", "parent", "cycle", "before", "reverse", "orphan"} {
		t.Run(mode, func(t *testing.T) {
			w, p := resultWorld(t)
			w.f.s.documentERP = w.a
			first := confirmResult(t, w, proposeResult(t, w, p, "first"))
			_, err := applyResult(w, first)
			w.f.must(err)
			p.Revision = 2
			next := confirmResult(t, w, proposeResult(t, w, p, "second"))
			_, err = applyResult(w, next)
			w.f.must(err)
			key := [2]string{SampleWorkspace, "second"}
			app := w.a.resultApplications[key]
			switch mode {
			case "missing-after":
				app.after = nil
			case "altered-after":
				app.after = cloneResultSnapshot(*app.after)
				app.after.Documents[0].Links[0].Author = "foreign"
			case "parent":
				app.predecessor = "missing"
			case "cycle":
				app.predecessor = "second"
			case "before":
				app.before = cloneResultSnapshot(*app.before)
				app.before.Documents[0].Author = "foreign"
			case "reverse":
				w.a.resultPreparations[[2]string{SampleWorkspace, p.Preparation}] = "first"
			case "orphan":
				delete(w.a.resultExternals, [3]string{SampleWorkspace, p.Connection, p.External})
				delete(w.a.resultPreparations, [2]string{SampleWorkspace, p.Preparation})
			}
			w.a.resultApplications[key] = app
			before, err := w.ledger.ResultOf(SampleWorkspace, p.Preparation)
			w.f.must(err)
			if w.a.validateResultChains() == nil {
				t.Fatal("corrupt chain accepted")
			}
			if _, _, err = resultEvidenceCounts(w.f.s); err == nil {
				t.Fatal("corrupt impact accepted")
			}
			if _, err = applyResult(w, next); err == nil {
				t.Fatal("corrupt apply accepted")
			}
			after, err := w.ledger.ResultOf(SampleWorkspace, p.Preparation)
			w.f.must(err)
			if !reflect.DeepEqual(before, after) {
				t.Fatal("corrupt chain changed ledger")
			}
		})
	}
}

func TestFinancialERPResultReplacementPublicCachedReplay(t *testing.T) {
	w, p, digest := publicResult(t)
	f := w.f
	actor := w.actors[Admin2Login]
	body := chargeBody(f, "replacement-old-key", "public-result", map[string]string{"digest": digest})
	first := f.call("erp_result.apply", body, actor.cookie, actor.csrf)
	f.ok(t, "first", first, "recorded", "")
	p.Revision = 2
	raw, err := json.Marshal(p)
	f.must(err)
	next := f.run(actor, "result_evidence.propose", "public-next", map[string]string{"packet": string(raw)})
	f.ok(t, "next proposal", next, "pending", "")
	nextDigest := field(next, "digest")
	f.ok(t, "next confirmation", f.run(w.actors[Admin3Login], "result_evidence.confirm", "public-next", map[string]string{"digest": nextDigest}), "confirmed", "")
	advice := documentRead(f, actor, "result_evidence.advice", "public-next", map[string]string{"digest": nextDigest})
	if !strings.Contains(field(advice, "readiness"), `"apply":{"state":"ready"`) {
		t.Fatalf("replacement advice: %s", field(advice, "readiness"))
	}
	w.a.afterResultWrite = func(stage string) error {
		if stage == "replacement" {
			return errors.New("response lost")
		}
		return nil
	}
	f.code(t, "unknown replacement", f.run(actor, "erp_result.apply", "public-next", map[string]string{"digest": nextDigest}), 500, "internal_error")
	w.a.afterResultWrite = nil
	recovered := f.run(actor, "erp_result.apply", "public-next", map[string]string{"digest": nextDigest})
	f.ok(t, "recover next", recovered, "recorded", "")
	f.code(t, "old cached success stale", f.call("erp_result.apply", body, actor.cookie, actor.csrf), 409, string(commands.EvidenceStale))
	historical := documentRead(f, actor, "result_evidence.inspect", "public-result", map[string]string{"digest": digest})
	if historical.status != 200 || field(historical, "application_receipt") != field(first, "application_receipt") {
		t.Fatal("historical HTTP receipt lost")
	}
}

func TestFinancialERPResultReplacementUnknownCurrentChecks(t *testing.T) {
	for _, mode := range []string{"foreign-actor", "revoked", "newer", "third-state"} {
		t.Run(mode, func(t *testing.T) {
			w, p := resultWorld(t)
			w.f.s.documentERP = w.a
			first := confirmResult(t, w, proposeResult(t, w, p, "first"))
			_, err := applyResult(w, first)
			w.f.must(err)
			p.Revision = 2
			next := confirmResult(t, w, proposeResult(t, w, p, "second"))
			w.a.afterResultWrite = func(string) error { return errors.New("interrupted") }
			_, err = applyResult(w, next)
			if !errors.Is(err, errFinancialERPResultUncertain) {
				t.Fatal(err)
			}
			w.a.afterResultWrite = nil
			switch mode {
			case "foreign-actor":
				resultGrantTo(t, w.f, "apply", SampleAdmin)
				refreshResultActors(w)
			case "revoked":
				for _, g := range w.f.s.admin.Grants() {
					if g.SubjectID == SampleAdmin2 && g.Terms.Action == gate.ApplyERPResult {
						w.f.must(w.f.s.admin.RevokeGrant(g.ID, g.Version))
					}
				}
			case "newer":
				p.Revision = 3
				newer := confirmResult(t, w, proposeResult(t, w, p, "third"))
				_, err = applyResult(w, newer)
				documentFailure(t, err, commands.EvidenceStale)
			case "third-state":
				doc, e := w.ledger.DocumentOf(SampleWorkspace, p.Connection, p.External)
				w.f.must(e)
				doc.Document.Revision++
				_, e = w.ledger.Document(SampleWorkspace, doc.Document, erpledger.Actor{ID: "foreign", Kind: erpledger.KindHuman})
				w.f.must(e)
			}
			before, err := w.ledger.ResultOf(SampleWorkspace, p.Preparation)
			w.f.must(err)
			if mode == "foreign-actor" {
				_, err = resultAs(w, AdminLogin, func(actor string) (financialERPResultView, error) {
					return w.a.applyResultLocked(SampleWorkspace, next.Basis.ID, next.Review.Digest, actor)
				})
			} else {
				_, err = applyResult(w, next)
			}
			switch mode {
			case "third-state":
				if !errors.Is(err, errFinancialERPResultUncertain) {
					t.Fatalf("third-state outcome: %v", err)
				}
			case "revoked":
				documentFailure(t, err, commands.PermissionDenied)
			default:
				documentFailure(t, err, commands.EvidenceStale)
			}
			after, err := w.ledger.ResultOf(SampleWorkspace, p.Preparation)
			w.f.must(err)
			if !reflect.DeepEqual(before, after) || w.a.resultApplications[[2]string{SampleWorkspace, "second"}].receipt != "" || len(w.a.resultApplications) != 2 {
				t.Fatal("unknown ownership or ledger changed")
			}
			_, unknown, err := resultEvidenceCounts(w.f.s)
			w.f.must(err)
			if unknown[SampleAdmin2] != 1 {
				t.Fatal("stranded replacement disappeared")
			}
		})
	}
}

func TestFinancialERPResultReplacementCapacityBeforeReservation(t *testing.T) {
	w, p := resultWorld(t)
	for revision := 1; revision <= 50; revision++ {
		p.Revision = revision
		v := confirmResult(t, w, proposeResult(t, w, p, fmt.Sprintf("capacity-%d", revision)))
		_, err := applyResult(w, v)
		w.f.must(err)
	}
	before, err := w.ledger.ResultOf(SampleWorkspace, p.Preparation)
	w.f.must(err)
	if len(before.Documents[0].Links) != 100 {
		t.Fatal("fixture did not reach actual ledger capacity")
	}
	p.Revision = 51
	next := confirmResult(t, w, proposeResult(t, w, p, "capacity-next"))
	if _, err = w.a.replacementIntent(next.Basis, SampleAdmin2, next.Review.Digest); err == nil {
		t.Fatal("capacity readiness accepted")
	}
	_, err = applyResult(w, next)
	documentFailure(t, err, commands.PrerequisiteFailed)
	if len(w.a.resultApplications) != 50 || w.a.resultExternals[[3]string{SampleWorkspace, p.Connection, p.External}] != "capacity-50" || w.a.resultPreparations[[2]string{SampleWorkspace, p.Preparation}] != "capacity-50" {
		t.Fatal("capacity refusal reserved ownership")
	}
	after, err := w.ledger.ResultOf(SampleWorkspace, p.Preparation)
	w.f.must(err)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("capacity refusal changed ledger")
	}
}
