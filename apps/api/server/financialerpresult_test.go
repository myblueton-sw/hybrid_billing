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
)

func resultGrantTo(t *testing.T, f world, capability, recipient string) {
	t.Helper()
	governors := []string{}
	for _, u := range []struct{ id, login string }{{SampleAdmin, AdminLogin}, {SampleAdmin2, Admin2Login}, {SampleAdmin3, Admin3Login}} {
		if u.id != recipient {
			governors = append(governors, u.login)
		}
	}
	first, second := f.as(governors[0]), f.as(governors[1])
	defer f.s.endProbe(first.verifier(f))
	defer f.s.endProbe(second.verifier(f))
	id := "result-" + capability + "-" + recipient
	fields := resultPermissionTimes(f)
	fields["capability"] = capability
	p := f.run(first, "result_authority.propose_console", id+"-console", fields)
	f.ok(t, "result ceiling proposal", p, "pending", "")
	ceiling := resultPermissionApprove(f, second, id+"-console", p)
	f.ok(t, "result ceiling", ceiling, "applied", "")
	fields["recipient"], fields["ceiling"] = recipient, field(ceiling, "grant")
	p = f.run(first, "result_authority.propose_human", id+"-human", fields)
	f.ok(t, "result human proposal", p, "pending", "")
	f.ok(t, "result human grant", resultPermissionApprove(f, second, id+"-human", p), "applied", "")
}

func resultWorld(t *testing.T, beforeFinance ...func(world)) (documentWorld, financialERPResultPacket) {
	t.Helper()
	return resultWorldFromDocument(t, explicitDocumentWorld(t, beforeFinance...))
}

func resultWorldFromDocument(t *testing.T, w documentWorld) (documentWorld, financialERPResultPacket) {
	t.Helper()
	sub, err := w.prepare("result-preparation")
	w.f.must(err)
	for _, entry := range []struct{ capability, recipient string }{{"propose", SampleAdmin2}, {"apply", SampleAdmin2}, {"inspect", SampleAdmin2}, {"confirm", SampleAdmin3}, {"revoke", SampleAdmin3}, {"inspect", SampleAdmin3}} {
		resultGrantTo(t, w.f, entry.capability, entry.recipient)
	}
	for login := range w.actors {
		for _, v := range w.f.s.sessions.byLogin(login) {
			w.f.s.endProbe(v)
		}
		w.actors[login] = w.f.as(login)
	}
	w.consumer = w.actors[Admin2Login]
	packet := financialERPResultPacket{Mode: "synthetic", Profile: "synthetic_result_one_to_one_v1", Evidence: "reported-source", Preparation: "result-preparation", PreparationDigest: sub.Digest, Connection: "synthetic-document-erp", External: "reported-document", Currency: "USD", Customer: sub.Submission.Customer, Period: "2026-10", Revision: 1}
	for i, line := range sub.Submission.Lines {
		packet.Rows = append(packet.Rows, financialERPResultRow{ID: fmt.Sprintf("row-%d", i), Line: line.ID, Product: "synthetic-product", Unit: "document-unit", Net: []string{"10.25", "4.75"}[i], Tax: []string{"0.25", "0.75"}[i], Quantity: []string{"2", "3"}[i]})
	}
	return w, packet
}

func resultAs(w documentWorld, login string, run func(string) (financialERPResultView, error)) (financialERPResultView, error) {
	var out financialERPResultView
	err := inputAuthorityAs(w.f, w.actors[login], func() error { var err error; out, err = run(w.f.s.answer.User()); return err })
	return out, err
}
func proposeResult(t *testing.T, w documentWorld, p financialERPResultPacket, id string) financialERPResultView {
	t.Helper()
	raw, err := json.Marshal(p)
	w.f.must(err)
	v, err := resultAs(w, Admin2Login, func(actor string) (financialERPResultView, error) {
		return w.a.proposeResultLocked(SampleWorkspace, id, actor, "Independent reported packet", string(raw))
	})
	w.f.must(err)
	return v
}
func confirmResult(t *testing.T, w documentWorld, v financialERPResultView) financialERPResultView {
	t.Helper()
	out, err := resultAs(w, Admin3Login, func(actor string) (financialERPResultView, error) {
		return w.a.decideResultLocked(SampleWorkspace, v.Basis.ID, v.Review.Digest, actor, "Independent result confirmation", false)
	})
	w.f.must(err)
	return out
}
func applyResult(w documentWorld, v financialERPResultView) (financialERPResultView, error) {
	return resultAs(w, Admin2Login, func(actor string) (financialERPResultView, error) {
		return w.a.applyResultLocked(SampleWorkspace, v.Basis.ID, v.Review.Digest, actor)
	})
}

func TestFinancialERPResultActualApplicationAndBoundaryRecovery(t *testing.T) {
	for _, boundary := range []string{"", "document", "link:row-0", "link:row-1"} {
		t.Run(boundary, func(t *testing.T) {
			w, p := resultWorld(t)
			v := confirmResult(t, w, proposeResult(t, w, p, "reported"))
			if len(v.Basis.Comparison) != 0 {
				t.Fatal(v.Basis.Comparison)
			}
			if boundary != "" {
				w.a.afterResultWrite = func(stage string) error {
					if stage == boundary {
						return errors.New("lost response")
					}
					return nil
				}
				_, err := applyResult(w, v)
				if !errors.Is(err, errFinancialERPResultUncertain) {
					t.Fatalf("got %v", err)
				}
				app := w.a.resultApplications[[2]string{SampleWorkspace, "reported"}]
				if app.receipt != "" || app.actor != SampleAdmin2 {
					t.Fatal("uncertainty ownership lost")
				}
				w.a.afterResultWrite = nil
			}
			got, err := applyResult(w, v)
			w.f.must(err)
			if got.ApplicationState != "recorded" || !validPlanHash(got.ApplicationReceipt) {
				t.Fatal(got.ApplicationState)
			}
			again, err := applyResult(w, v)
			w.f.must(err)
			if again.ApplicationReceipt != got.ApplicationReceipt {
				t.Fatal("receipt changed")
			}
			actual, err := w.ledger.ResultOf(SampleWorkspace, p.Preparation)
			w.f.must(err)
			if !actual.Reconciliation.Matched || len(actual.Documents) != 1 || len(actual.Documents[0].Links) != 2 || actual.Documents[0].Total != (erpledger.Amounts{Net: "15.00", Tax: "1.00", Quantity: "5.00"}) || actual.Submission.State != erpledger.StatePrepared || len(actual.Submission.Commands) != 0 {
				t.Fatal(actual)
			}
			got.Basis.Packet.Rows[0].Net = "999"
			got.Basis.Preparation.Submission.Lines[0].Net = "999"
			read, err := resultAs(w, Admin2Login, func(actor string) (financialERPResultView, error) {
				return w.a.inspectResultLocked(SampleWorkspace, v.Basis.ID, v.Review.Digest, actor)
			})
			w.f.must(err)
			if read.Basis.Packet.Rows[0].Net == "999" || read.Basis.Preparation.Submission.Lines[0].Net == "999" || read.CurrentEligibility != "unchecked" {
				t.Fatal("history mutated")
			}
		})
	}
}

func TestFinancialERPResultMismatchesRetainedWithoutApplication(t *testing.T) {
	w, p := resultWorld(t)
	for _, mode := range []string{"product", "unit", "net", "tax", "quantity", "line", "swap"} {
		t.Run(mode, func(t *testing.T) {
			packet := cloneDocumentValue(t, p)
			packet.External = "mismatch-" + mode
			switch mode {
			case "product":
				packet.Rows[0].Product = "other-product"
			case "unit":
				packet.Rows[0].Unit = "other-unit"
			case "net":
				packet.Rows[0].Net = "11"
			case "tax":
				packet.Rows[0].Tax = "0.5"
			case "quantity":
				packet.Rows[0].Quantity = "4"
			case "line":
				packet.Rows[0].Line = packet.Rows[1].Line
			case "swap":
				packet.Rows[0].Net, packet.Rows[1].Net = packet.Rows[1].Net, packet.Rows[0].Net
			}
			v := confirmResult(t, w, proposeResult(t, w, packet, mode))
			if len(v.Basis.Comparison) == 0 {
				t.Fatal("mismatch hidden")
			}
			_, err := applyResult(w, v)
			documentFailure(t, err, commands.PrerequisiteFailed)
			if _, err = w.ledger.DocumentOf(SampleWorkspace, p.Connection, packet.External); !errors.Is(err, erpledger.ErrNotFound) {
				t.Fatal("mismatch wrote ledger")
			}
		})
	}
	if len(w.a.resultApplications) != 0 {
		t.Fatal("mismatch reserved ownership")
	}
}

func TestFinancialERPResultWatermarkPreservesStrandedOwnership(t *testing.T) {
	w, p := resultWorld(t)
	old := confirmResult(t, w, proposeResult(t, w, p, "older"))
	conflict := proposeResult(t, w, p, "conflict")
	_, err := resultAs(w, Admin3Login, func(actor string) (financialERPResultView, error) {
		return w.a.decideResultLocked(SampleWorkspace, "conflict", conflict.Review.Digest, actor, "Conflicting version", false)
	})
	documentFailure(t, err, commands.EvidenceStale)
	w.a.afterResultWrite = func(string) error { return errors.New("interrupted") }
	_, err = applyResult(w, old)
	if !errors.Is(err, errFinancialERPResultUncertain) {
		t.Fatal(err)
	}
	w.a.afterResultWrite = nil
	p.Revision = 2
	newer := confirmResult(t, w, proposeResult(t, w, p, "newer"))
	_, err = applyResult(w, old)
	documentFailure(t, err, commands.EvidenceStale)
	_, err = applyResult(w, newer)
	documentFailure(t, err, commands.EvidenceStale)
	_, err = resultAs(w, Admin3Login, func(actor string) (financialERPResultView, error) {
		return w.a.decideResultLocked(SampleWorkspace, "newer", newer.Review.Digest, actor, "Withdraw newer evidence", true)
	})
	w.f.must(err)
	_, err = applyResult(w, old)
	documentFailure(t, err, commands.EvidenceStale)
	if len(w.a.resultApplications) != 1 || w.a.resultExternals[[3]string{SampleWorkspace, p.Connection, p.External}] != "older" || w.a.resultApplications[[2]string{SampleWorkspace, "older"}].receipt != "" {
		t.Fatal("unknown ownership released")
	}
}

func TestFinancialERPResultStrictPacketShape(t *testing.T) {
	w, p := resultWorld(t)
	raw, err := json.Marshal(p)
	w.f.must(err)
	parsed, err := parseResultPacket(string(raw))
	w.f.must(err)
	if !reflect.DeepEqual(parsed, p) {
		t.Fatal("packet changed")
	}
	for _, bad := range []string{strings.Replace(string(raw), "\"Mode\":", "\"mode\":", 1), strings.Replace(string(raw), "\"Mode\":", "\"Mode\":\"synthetic\",\"Mode\":", 1), strings.Replace(string(raw), "\"Revision\":1", "\"Revision\":2147483648", 1), strings.Replace(string(raw), "\"Revision\":1", "\"Revision\":1.0", 1), strings.Replace(string(raw), "\"Rows\":[", "\"Unknown\":true,\"Rows\":[", 1), string(raw) + " null", strings.Repeat(" ", 4097), strings.Replace(string(raw), "reported-source", `\ud800`, 1)} {
		if _, err := parseResultPacket(bad); err == nil {
			t.Fatal("invalid packet accepted")
		}
	}
}
