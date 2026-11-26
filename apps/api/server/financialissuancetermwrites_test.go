package server

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/receivables"
	"github.com/myblueton-sw/hybrid_billing/modules/termsauthority"
)

type publicTermWorld struct {
	f                        world
	c                        financialIssuanceConfiguration
	author, approver, reader actor
}

func termWriteBody(f world, key, kind string, c financialIssuanceConfiguration, fields map[string]string) string {
	body := map[string]string{"configuration": c.Spec.ID, "digest": c.Digest}
	for name, value := range fields {
		body[name] = value
	}
	envelope := map[string]any{"request_id": key, "target_id": c.Spec.Contract, "idempotency_key": key, "expected_revision": f.s.Revision(), "body": body}
	if step, _ := termWriteCommand(kind); step.Rationale {
		envelope["rationale"] = "Agreed payment terms for the contract"
	}
	raw, _ := json.Marshal(envelope)
	return string(raw)
}

func termWrite(f world, who actor, kind string, c financialIssuanceConfiguration, fields map[string]string) reply {
	f.t.Helper()
	return f.call("issuance_setup."+kind, termWriteBody(f, fmt.Sprintf("term-write-%d", keys.Add(1)), kind, c, fields), who.cookie, who.csrf)
}

func termsRead(f world, who actor, c financialIssuanceConfiguration, extra map[string]string) reply {
	return protectedSetupRead(f, who, "terms", c, extra)
}

func publicTerms(t *testing.T) *publicTermWorld {
	t.Helper()
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	p, err := proposeConfigurationAs(f, f.as(AdminLogin), configurationSpec())
	f.must(err)
	c, err := confirmConfigurationAs(f, f.as(Admin3Login), p)
	f.must(err)
	author, approver := activateProtectedMember(f, "term-author"), activateProtectedMember(f, "term-approver")
	for _, right := range []struct{ capability, recipient string }{{termsauthority.RecordTerm, "term-author"}, {termsauthority.BindTerm, "term-author"}, {termsauthority.ApproveTerm, "term-approver"}, {termsauthority.ApproveBinding, "term-approver"}} {
		grantTermCapability(t, f, right.capability, right.recipient)
	}
	grantSetupInspectionForTest(f, false)
	return &publicTermWorld{f: f, c: c, author: f.as(author), approver: f.as(approver), reader: f.as(MemberLogin)}
}

func (w *publicTermWorld) approvedTerm(t *testing.T) string {
	t.Helper()
	recorded := termWrite(w.f, w.author, "record_term", w.c, map[string]string{"term": "public-term", "net_days": "30"})
	w.f.ok(t, "record", recorded, "pending", "")
	approved := termWrite(w.f, w.approver, "approve_term", w.c, map[string]string{"term": "public-term", "term_digest": field(recorded, "term_digest")})
	w.f.ok(t, "approve", approved, "approved", "")
	return field(approved, "term_digest")
}

func TestFinancialTermWritesPublicSequence(t *testing.T) {
	w := publicTerms(t)
	f, a := w.f, w.f.s.financialIssuance
	empty := termsRead(f, w.reader, w.c, nil)
	if empty.status != 200 || field(empty, "term_state") != "none" || field(empty, "term_observation") != "absent" || field(empty, "count") != "0" || field(empty, "zone") != w.c.Spec.Zone || field(empty, "net_days") != "" {
		t.Fatal(empty.raw)
	}
	recorded := termWrite(f, w.author, "record_term", w.c, map[string]string{"term": "public-term", "net_days": "45"})
	f.ok(t, "record", recorded, "pending", "")
	digest := field(recorded, "term_digest")
	if field(recorded, "schema") != "hb_issuance_term_receipt_v1" || field(recorded, "current_eligibility") != "unchecked" || a.termIntents["public-term"].Term.NetDays != 45 || a.termIntents["public-term"].Term.Mode != receivables.ModeCalendar || a.termIntents["public-term"].Term.Revision != 1 {
		t.Fatal(recorded.raw)
	}
	pending := termsRead(f, w.reader, w.c, nil)
	if field(pending, "term_state") != "pending" || field(pending, "term_observation") != "pending" || field(pending, "net_days") != "45" || field(pending, "mode") != "calendar" || field(pending, "term_revision") != "1" || field(pending, "term_author") != "term-author" || field(pending, "term_approver") != "" || field(pending, "term_approved_at") != "" || field(pending, "term_written_at") == "" {
		t.Fatal(pending.raw)
	}
	f.code(t, "author without the approval capability", termWrite(f, w.author, "approve_term", w.c, map[string]string{"term": "public-term", "term_digest": digest}), 403, string(commands.PermissionDenied))
	grantTermHuman(f, termsauthority.ApproveTerm, "term-author")
	f.code(t, "author approves own term", termWrite(f, w.author, "approve_term", w.c, map[string]string{"term": "public-term", "term_digest": digest}), 403, string(commands.SelfApprovalDenied))
	if a.termIntents["public-term"].State != "pending" || a.termIntents["public-term"].Approver != "" {
		t.Fatal("self approval reserved")
	}
	f.ok(t, "approve", termWrite(f, w.approver, "approve_term", w.c, map[string]string{"term": "public-term", "term_digest": digest}), "approved", "")
	bind := map[string]string{"binding": "public-binding", "term": "public-term", "term_digest": digest, "from": "2026-11-01", "to": "2027-10-31", "reason": "initial"}
	bound := termWrite(f, w.author, "bind_term", w.c, bind)
	f.ok(t, "bind", bound, "pending", "")
	if field(bound, "schema") != "hb_issuance_term_binding_receipt_v1" || a.bindingIntents["public-binding"].Binding.Zone != w.c.Spec.Zone || a.bindingIntents["public-binding"].Binding.Contract != w.c.Spec.Contract {
		t.Fatal(bound.raw)
	}
	f.ok(t, "identical repeat", termWrite(f, w.author, "bind_term", w.c, bind), "pending", "")
	f.ok(t, "approve binding", termWrite(f, w.approver, "approve_term_binding", w.c, map[string]string{"binding": "public-binding", "binding_digest": field(bound, "binding_digest")}), "approved", "")

	before, revision := len(a.bindingIntents), f.s.Revision()
	got := termsRead(f, w.reader, w.c, nil)
	row := protectedSetupRow(t, got, 0)
	if field(got, "term_state") != "approved" || field(got, "term_observation") != "approved" || field(got, "term_approver") != "term-approver" || field(got, "term_pending_approver") != "" || field(got, "count") != "1" || field(got, "current_eligibility") != "unchecked" || field(got, "schema") != "hb_issuance_setup_terms_v1" {
		t.Fatal(got.raw)
	}
	want := map[string]any{"binding": "public-binding", "origin": "configuration", "configuration": w.c.Spec.ID, "term": "public-term", "term_revision": "1", "net_days": "45", "mode": "calendar", "zone": w.c.Spec.Zone, "from": "2026-11-01", "to": "2027-10-31", "reason": "initial", "state": "approved", "observation": "approved", "author": "term-author", "approver": "term-approver", "pending_approver": ""}
	for name, value := range want {
		if row[name] != value {
			t.Fatalf("%s = %v, want %v", name, row[name], value)
		}
	}
	if len(row) != 19 || row["approved_at"] == "" || strings.Contains(got.raw, "Agreed payment terms") || len(a.bindingIntents) != before || f.s.Revision() != revision {
		t.Fatal("row shape, reason disclosure or read effect", got.raw)
	}
	f.s.enterHold("Term history inspection")
	if held := termsRead(f, w.reader, w.c, nil); held.status != 200 || field(held, "snapshot") != field(got, "snapshot") {
		t.Fatal("held read", held.raw)
	}
	f.s.hold = nil
	binding := a.bindingIntents["public-binding"]
	f.must(inputAuthorityAs(f, w.author, func() error {
		coverage, err := a.configuredCoverage(w.c, binding.Spec.ID, binding.Digest, f.s.clock())
		if err != nil {
			return err
		}
		if coverage.NetDays != 45 || coverage.Mode != receivables.ModeCalendar || coverage.Contract != w.c.Spec.Contract {
			t.Fatalf("coverage %+v", coverage)
		}
		return nil
	}))
}

func TestFinancialTermWritesFaultOutcomes(t *testing.T) {
	cases := []struct {
		kind, fault, state, observation string
		clean                           bool
	}{
		{"record_term", "issuance-term-before-register", "none", "absent", true}, {"record_term", "issuance-term-after-register", "registration_unknown", "pending", false}, {"record_term", "issuance-term-after-register-publication", "pending", "pending", false},
		{"approve_term", "issuance-term-before-approve", "pending", "pending", true}, {"approve_term", "issuance-term-after-approve", "approval_unknown", "approved", false}, {"approve_term", "issuance-term-after-approve-publication", "approved", "approved", false},
		{"bind_term", "issuance-binding-before-register", "", "", true}, {"bind_term", "issuance-binding-after-register", "registration_unknown", "pending", false}, {"bind_term", "issuance-binding-after-register-publication", "pending", "pending", false},
		{"approve_term_binding", "issuance-binding-before-approve", "pending", "pending", true}, {"approve_term_binding", "issuance-binding-after-approve", "approval_unknown", "approved", false}, {"approve_term_binding", "issuance-binding-after-approve-publication", "approved", "approved", false},
	}
	for _, tc := range cases {
		t.Run(tc.kind+"/"+tc.fault, func(t *testing.T) {
			w := publicTerms(t)
			f, a := w.f, w.f.s.financialIssuance
			who, family, id, recovered := w.author, "term", "public-term", "pending"
			fields := map[string]string{"term": "public-term", "net_days": "30"}
			if tc.kind != "record_term" {
				recorded := termWrite(f, w.author, "record_term", w.c, fields)
				f.ok(t, "record", recorded, "pending", "")
				who, recovered, fields = w.approver, "approved", map[string]string{"term": "public-term", "term_digest": field(recorded, "term_digest")}
			}
			if tc.kind == "bind_term" || tc.kind == "approve_term_binding" {
				f.ok(t, "approve", termWrite(f, w.approver, "approve_term", w.c, fields), "approved", "")
				who, family, id, recovered = w.author, "binding", "public-binding", "pending"
				fields = map[string]string{"binding": "public-binding", "term": "public-term", "term_digest": fields["term_digest"], "from": "2026-11-01", "reason": "initial"}
			}
			if tc.kind == "approve_term_binding" {
				bound := termWrite(f, w.author, "bind_term", w.c, fields)
				f.ok(t, "bind", bound, "pending", "")
				who, recovered, fields = w.approver, "approved", map[string]string{"binding": "public-binding", "binding_digest": field(bound, "binding_digest")}
			}
			observe := func() termWriteObservation {
				var out termWriteObservation
				f.must(inputAuthorityAs(f, who, func() error {
					out = a.observeTermWrite(SampleWorkspace, family, id)
					return nil
				}))
				return out
			}
			before, revision := observe(), f.s.Revision()
			f.s.fault = func(at string) bool { return at == tc.fault }
			body := termWriteBody(f, "faulted-term-write", tc.kind, w.c, fields)
			got := f.call("issuance_setup."+tc.kind, body, who.cookie, who.csrf)
			if tc.clean {
				f.code(t, "clean refusal", got, 503, string(commands.DependencyUnavailable))
				if got.body["operation_id"] != nil || f.s.Revision() != revision || !reflect.DeepEqual(before, observe()) {
					t.Fatal("clean refusal changed state or carried an operation", got.raw)
				}
			} else {
				if got.status != 500 || got.body["operation_id"] == nil || f.s.Revision() == revision || reflect.DeepEqual(before, observe()) {
					t.Fatal("effect not reported as uncertain", got.raw)
				}
				if replay := f.call("issuance_setup."+tc.kind, body, who.cookie, who.csrf); replay.status != 500 || replay.body["operation_id"] != got.body["operation_id"] {
					t.Fatal("same key did not replay the stored outcome", replay.raw)
				}
				unchanged, second := observe(), f.s.Revision()
				again := termWrite(f, who, tc.kind, w.c, fields)
				if strings.HasSuffix(tc.fault, "-publication") {
					f.ok(t, "publication already retained", again, recovered, "")
				} else {
					f.code(t, "repeated failure at the same point", again, 503, string(commands.DependencyUnavailable))
					if again.body["operation_id"] != nil || f.s.Revision() != second || !reflect.DeepEqual(unchanged, observe()) {
						t.Fatal("repeated failure was not a clean refusal", again.raw)
					}
				}
			}
			f.s.fault = nil
			read := termsRead(f, w.reader, w.c, nil)
			if family == "term" {
				if read.status != 200 || field(read, "term_state") != tc.state || field(read, "term_observation") != tc.observation {
					t.Fatal("read after the fault", read.raw)
				}
				pinned := map[bool]string{true: "term-approver"}[tc.state == "approval_unknown" && tc.observation == "pending"]
				if field(read, "term_pending_approver") != pinned || (field(read, "term_approver") != "") != (tc.observation == "approved") {
					t.Fatal("approver layers", read.raw)
				}
			} else if tc.state == "" {
				if field(read, "count") != "0" {
					t.Fatal(read.raw)
				}
			} else if row := protectedSetupRow(t, read, 0); row["state"] != tc.state || row["observation"] != tc.observation || (row["approver"] != "") != (tc.observation == "approved") {
				t.Fatalf("row after the fault %v", row)
			}
			f.ok(t, "new identical request", termWrite(f, who, tc.kind, w.c, fields), recovered, "")
		})
	}
}

func TestFinancialTermWritesAuthority(t *testing.T) {
	w := publicTerms(t)
	f := w.f
	digest := strings.Repeat("a", 64)
	valid := map[string]map[string]string{
		"record_term": {"term": "authority-term", "net_days": "30"}, "approve_term": {"term": "authority-term", "term_digest": digest},
		"bind_term": {"binding": "authority-binding", "term": "authority-term", "term_digest": digest, "from": "2026-11-01", "reason": "initial"}, "approve_term_binding": {"binding": "authority-binding", "binding_digest": digest},
	}
	holder := map[string]actor{"record_term": w.author, "approve_term": w.approver, "bind_term": w.author, "approve_term_binding": w.approver}
	protectedOnly := activateProtectedMember(f, "protected-only")
	grantProtectedInspectionForTerms(f, "protected-only")
	callers := map[string]actor{"author": w.author, "approver": w.approver, "terms inspection only": w.reader, "protected inspection only": f.as(protectedOnly), "administrator": f.as(AdminLogin)}
	revision := f.s.Revision()
	for _, step := range termWriteSteps {
		for name, who := range callers {
			if who == holder[step.Kind] {
				continue
			}
			f.code(t, step.Kind+" by "+name, termWrite(f, who, step.Kind, w.c, valid[step.Kind]), 403, string(commands.PermissionDenied))
		}
		terms, err := termAuthorityTerms(SampleWorkspace, step.Capability)
		f.must(err)
		if f.s.admin.AddDelegation(gate.Delegation{ID: "delegated-" + step.Kind, DelegatorID: SampleAdmin, DelegateID: SampleMember, Terms: terms, Validity: gate.Validity{NotBefore: f.s.clock(), NotAfter: f.s.clock().Add(time.Hour)}}) == nil {
			t.Fatalf("%s capability could be delegated", step.Kind)
		}
	}
	for name, who := range map[string]actor{"author": w.author, "approver": w.approver, "protected inspection only": f.as(protectedOnly), "administrator": f.as(AdminLogin)} {
		f.code(t, "read by "+name, termsRead(f, who, w.c, nil), 403, string(commands.PermissionDenied))
	}
	if f.s.Revision() != revision || len(f.s.financialIssuance.termIntents) != 0 {
		t.Fatal("a refused caller changed state")
	}
	f.s.enterHold("Term writes held")
	for _, step := range termWriteSteps {
		f.code(t, step.Kind+" held", termWrite(f, holder[step.Kind], step.Kind, w.c, valid[step.Kind]), 503, string(commands.DependencyUnavailable))
	}
	f.s.hold = nil
	for _, mode := range []bool{false, true} {
		for _, step := range termWriteSteps {
			revoked := 0
			for _, grant := range f.s.admin.Grants() {
				_, ceiling := f.s.termAuthority.ceilings[grant.ID]
				if grant.Active && grant.Terms.Action == step.Action && ceiling == mode {
					f.must(f.s.admin.RevokeGrant(grant.ID, grant.Version))
					revoked++
				}
			}
			if revoked == 0 && !mode {
				t.Fatalf("%s: nothing revoked", step.Kind)
			}
			f.code(t, step.Kind+" revoked", termWrite(f, holder[step.Kind], step.Kind, w.c, valid[step.Kind]), 403, string(commands.PermissionDenied))
		}
	}
	f.s.endProbe(w.reader.verifier(f))
	f.code(t, "ended session", termsRead(f, w.reader, w.c, nil), 401, string(commands.Unauthenticated))
	f.s.cfg.Seed = SeedNone
	f.code(t, "non sample", termsRead(f, f.as(AdminLogin), w.c, nil), 501, string(commands.UnsupportedCapability))
}

func grantProtectedInspectionForTerms(f world, recipient string) {
	spec := protectedPermissionSpec(f, "inspect")
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	p, err := protectedPermissionProposal(f, root, inputauthority.Proposal{ID: "terms-test-ceiling", Spec: spec, Rationale: "Explicit evidence ceiling"})
	f.must(err)
	c, err := protectedPermissionDecision(f, second, p)
	f.must(err)
	spec.Kind, spec.Recipient, spec.CeilingID = inputauthority.Human, recipient, c.Decision.GrantID
	p, err = protectedPermissionProposal(f, root, inputauthority.Proposal{ID: "terms-test-human", Spec: spec, Rationale: "Explicit evidence recipient"})
	f.must(err)
	_, err = protectedPermissionDecision(f, second, p)
	f.must(err)
}

func grantTermHuman(f world, capability, recipient string) {
	spec := termPermissionSpec(f, capability)
	spec.Kind, spec.Recipient = inputauthority.Human, recipient
	for id, ceiling := range f.s.termAuthority.ceilings {
		if ceiling.Capability == capability {
			spec.CeilingID = id
		}
	}
	p, err := termPermissionProposal(f, f.as(AdminLogin), inputauthority.Proposal{ID: capability + "-human-" + recipient, Spec: spec, Rationale: "Explicit synthetic term operator"})
	f.must(err)
	_, err = termPermissionDecision(f, f.as(Admin2Login), p)
	f.must(err)
}
