package server

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
)

func TestFinancialSetupProtectedChangeStates(t *testing.T) {
	cases := []struct{ point, phase, observation string }{
		{"before-request", "", ""}, {"after-request", "request_unknown", protected.StateRequested}, {"after-request-publication", "recorded", protected.StateRequested},
		{"before-verify", "recorded", protected.StateRequested}, {"after-verify", "verification_unknown", protected.StateConfirmed}, {"after-verification-publication", "recorded", protected.StateConfirmed},
		{"before-decide", "recorded", protected.StateConfirmed}, {"after-decide", "decision_unknown", protected.StateScheduled}, {"after-decision-publication", "recorded", protected.StateScheduled},
	}
	for _, tc := range cases {
		t.Run(tc.point, func(t *testing.T) {
			w := configuredSetup(t)
			f, c, spec := w.f, w.configuration, w.change
			grantSetupInspectionForTest(f, true)
			reader := f.as(MemberLogin)
			a := f.s.financialIssuance
			verify := strings.Contains(tc.point, "verif")
			decide := strings.Contains(tc.point, "decid") || strings.Contains(tc.point, "decision")
			var p financialChangeRecord
			var err error
			if verify || decide {
				p, err = requestChangeAs(f, w.requester, spec)
				f.must(err)
			}
			if decide {
				p, err = verifyChangeAs(f, w.verifier, p)
				f.must(err)
			}
			f.s.fault = func(at string) bool { return at == "issuance-change-"+tc.point }
			if decide {
				_, err = decideChangeAs(f, w.approver, p)
			} else if verify {
				_, err = verifyChangeAs(f, w.verifier, p)
			} else {
				_, err = requestChangeAs(f, w.requester, spec)
			}
			documentFailure(t, err, commands.DependencyUnavailable)
			f.s.fault = nil
			before := captureProtectedSetup(f)
			read := func() (reply, reply) {
				return protectedSetupRead(f, reader, "protected", c, nil), protectedSetupRead(f, reader, "protected_changes", c, map[string]string{"field": spec.Field})
			}
			summary, page := read()
			if tc.phase == "" {
				if page.status != 200 || field(page, "count") != "0" || field(summary, "template_changes") != "0" || len(a.changes) != 0 {
					t.Fatal(page.raw)
				}
				return
			}
			retained := a.changes[spec.ID]
			row := protectedSetupRow(t, page, 0)
			decisions, _ := row["decisions"].([]any)
			pendingVerifier, pendingDecision := "", ""
			if retained.Verification != nil && retained.Module.Verification == nil {
				pendingVerifier = retained.Verification.Actor
			}
			if len(retained.Decisions) > len(retained.Module.Decisions) {
				pendingDecision = retained.Decisions[len(retained.Decisions)-1].Actor
			}
			if row["phase"] != tc.phase || row["phase"] != retained.Phase || row["state"] != retained.Module.State || row["observation"] != tc.observation || row["rules"] != "current" || len(decisions) != len(retained.Module.Decisions) || row["pending_verifier"] != pendingVerifier || row["pending_decision_actor"] != pendingDecision || (row["verifier"] != "") != (retained.Module.Verification != nil) || row["ended_at"] != "" {
				t.Fatalf("row %v", row)
			}
			latest := ""
			if tc.point == "after-decision-publication" {
				latest = spec.ID
			}
			if field(summary, "template_changes") != "1" || field(summary, "template_latest_scheduled") != latest || (field(summary, "template_latest_scheduled_effective") != "") != (latest != "") {
				t.Fatal("uncertain change promoted", summary.raw)
			}
			f.advance(2 * time.Minute)
			_, later := read()
			lapsed := protectedSetupRow(t, later, 0)
			if tc.observation == protected.StateScheduled {
				if lapsed["observation"] != protected.StateScheduled || lapsed["ended_at"] != "" || field(later, "snapshot") != field(page, "snapshot") {
					t.Fatalf("scheduled row changed %v", lapsed)
				}
			} else if lapsed["observation"] != protected.StateLapsed || lapsed["state"] != retained.Module.State || lapsed["phase"] != tc.phase || lapsed["ended_at"] != setupInstant(spec.Effective) || lapsed["ended_by"] != "" || field(later, "snapshot") == field(page, "snapshot") {
				t.Fatalf("lapse projection %v", lapsed)
			}
			after := captureProtectedSetup(f)
			before.stored, after.stored = nil, nil
			if !reflect.DeepEqual(before, after) {
				t.Fatal("change read changed retained state")
			}
		})
	}
}

func TestFinancialSetupProtectedChangeOwnershipAndCorruption(t *testing.T) {
	foreignChange := func(c financialIssuanceConfiguration, id string, effective time.Time) protected.Change {
		return protected.Change{ID: id, Subject: protectedSetupSubject(c, "document_kind"), Field: protected.FieldDocumentTemplate, After: protected.Ref{ID: "module-only-reference", Fingerprint: strings.Repeat("d", 64)}, Provenance: protected.ProvenanceStaff, Instruction: protected.Instruction{Kind: protected.ChannelInPerson, Evidence: "module-only-evidence"}, Effective: effective}
	}
	actor := protected.Actor{ID: "foreign-requester", Kind: protected.KindHuman}
	template := map[string]string{"field": protected.FieldDocumentTemplate}

	t.Run("module only", func(t *testing.T) {
		w := configuredSetup(t)
		f, c := w.f, w.configuration
		grantSetupInspectionForTest(f, true)
		reader := f.as(MemberLogin)
		if _, err := f.s.financialIssuance.fields.Request(SampleWorkspace, foreignChange(c, "module-only-change", f.s.clock().Add(time.Hour)), actor); err != nil {
			t.Fatal(err)
		}
		summary, page := protectedSetupRead(f, reader, "protected", c, nil), protectedSetupRead(f, reader, "protected_changes", c, template)
		if page.status != 200 || field(page, "count") != "0" || field(page, "unowned_module") != "1" || field(page, "row_00") != "" || field(summary, "template_unowned_module") != "1" || field(summary, "template_changes") != "0" || field(summary, "recipient_unowned_module") != "0" {
			t.Fatal(page.raw, summary.raw)
		}
		for _, hidden := range []string{"module-only", "foreign-requester"} {
			if strings.Contains(page.raw, hidden) || strings.Contains(summary.raw, hidden) {
				t.Fatalf("module-only change disclosed: %s", hidden)
			}
		}
		if len(f.s.financialIssuance.changes) != 0 || len(f.s.financialIssuance.changeOrders) != 0 {
			t.Fatal("module-only change adopted")
		}
	})

	for _, mode := range []string{"trailing_unowned", "duplicate", "orphan", "state", "requester", "withdrawn", "lookup"} {
		t.Run(mode, func(t *testing.T) {
			w := configuredSetup(t)
			f, c := w.f, w.configuration
			grantSetupInspectionForTest(f, true)
			reader := f.as(MemberLogin)
			a := f.s.financialIssuance
			changes, _ := completeConfiguredSetup(t, w)
			owned := changes[protected.FieldDocumentTemplate]
			key := issuanceChangeScope(owned.Channel.Subject.Subject, owned.Spec.Field)
			if got := protectedSetupRead(f, reader, "protected_changes", c, template); got.status != 200 || field(got, "count") != "1" {
				t.Fatal(got.raw)
			}
			status, code := 409, string(commands.EvidenceStale)
			switch mode {
			case "trailing_unowned":
				if _, err := a.fields.Request(SampleWorkspace, foreignChange(c, "module-only-successor", owned.Spec.Effective.Add(time.Hour)), actor); err != nil {
					t.Fatal(err)
				}
				got := protectedSetupRead(f, reader, "protected_changes", c, template)
				if protectedSetupRow(t, got, 0)["change"] != owned.Spec.ID || field(got, "count") != "1" || field(got, "unowned_module") != "1" || strings.Contains(got.raw, "module-only") {
					t.Fatal(got.raw)
				}
				return
			case "duplicate":
				a.changeOrders[key] = append(a.changeOrders[key], owned.Spec.ID)
			case "orphan":
				a.changeOrders[key] = nil
			case "state":
				owned.Module.State = protected.StateConfirmed
				a.changes[owned.Spec.ID] = owned
			case "requester":
				owned.Requester = "changed-requester"
				a.changes[owned.Spec.ID] = owned
			case "withdrawn":
				spec := w.change
				spec.ID, spec.After.ID, spec.Effective = "open-successor", "synthetic-template-next", owned.Spec.Effective.Add(time.Hour)
				_, err := requestChangeAs(f, w.requester, spec)
				f.must(err)
				if _, err := a.fields.Withdraw(SampleWorkspace, spec.ID, actor); err != nil {
					t.Fatal(err)
				}
			case "lookup":
				f.s.fault = func(at string) bool { return at == "issuance-setup-protected-lookup" }
				status, code = 503, string(commands.DependencyUnavailable)
			}
			f.code(t, mode, protectedSetupRead(f, reader, "protected_changes", c, template), status, code)
			f.code(t, mode+" summary", protectedSetupRead(f, reader, "protected", c, nil), status, code)
		})
	}
}
