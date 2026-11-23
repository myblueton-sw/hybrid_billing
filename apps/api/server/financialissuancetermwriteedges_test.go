package server

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/protectedauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/termsauthority"
)

func (w *publicTermWorld) second(t *testing.T, id, zone string) financialIssuanceConfiguration {
	t.Helper()
	f := w.f
	base, err := f.s.orgs.ContractRevision(SampleWorkspace, w.c.Spec.Contract, w.c.Spec.ContractRevision)
	f.must(err)
	base.Revision = w.c.Spec.ContractRevision + uint64(len(f.s.financialIssuance.configurations))
	f.must(f.s.orgs.AppendContractRevision(base))
	spec := configurationSpec()
	spec.ID, spec.ContractRevision, spec.Zone = id, base.Revision, zone
	proposed, err := proposeConfigurationAs(f, f.as(AdminLogin), spec)
	f.must(err)
	other, err := confirmConfigurationAs(f, f.as(Admin3Login), proposed)
	f.must(err)
	return other
}

func TestFinancialTermWritesClosedValuesAndPreconditions(t *testing.T) {
	w := publicTerms(t)
	f, a := w.f, w.f.s.financialIssuance
	for name, days := range map[string]string{"negative": "-1", "above the bound": "366", "leading zero": "030", "fraction": "30.5", "empty": "", "words": "thirty"} {
		f.code(t, "net days "+name, termWrite(f, w.author, "record_term", w.c, map[string]string{"term": "closed-term", "net_days": days}), 422, string(commands.InvalidRequest))
	}
	missing := termWrite(f, w.author, "record_term", w.c, map[string]string{"term": "closed-term"})
	if missing.status != 422 || len(a.termIntents) != 0 {
		t.Fatal("a missing net-days value was defaulted", missing.raw)
	}
	for _, days := range []string{"0", "365"} {
		g := publicTerms(t)
		g.f.ok(t, "net days "+days, termWrite(g.f, g.author, "record_term", g.c, map[string]string{"term": "bound-term", "net_days": days}), "pending", "")
	}
	recorded := termWrite(f, w.author, "record_term", w.c, map[string]string{"term": "public-term", "net_days": "30"})
	f.ok(t, "record", recorded, "pending", "")
	digest := field(recorded, "term_digest")
	f.code(t, "second term for the configuration", termWrite(f, w.author, "record_term", w.c, map[string]string{"term": "another-term", "net_days": "45"}), 422, string(commands.PrerequisiteFailed))
	bind := func(id, from, to string) map[string]string {
		fields := map[string]string{"binding": id, "term": "public-term", "term_digest": digest, "from": from, "reason": "initial"}
		if to != "" {
			fields["to"] = to
		}
		return fields
	}
	f.code(t, "binding an unapproved term", termWrite(f, w.author, "bind_term", w.c, bind("early", "2026-11-01", "")), 422, string(commands.PrerequisiteFailed))
	f.ok(t, "approve", termWrite(f, w.approver, "approve_term", w.c, map[string]string{"term": "public-term", "term_digest": digest}), "approved", "")
	for name, change := range map[string]map[string]string{"invalid date": {"from": "2026-02-30"}, "end before start": {"to": "2026-10-31"}, "unknown reason": {"reason": "preference"}, "date format": {"from": "01/11/2026"}} {
		fields := bind("closed-binding", "2026-11-01", "2026-12-31")
		for k, v := range change {
			fields[k] = v
		}
		f.code(t, name, termWrite(f, w.author, "bind_term", w.c, fields), 422, string(commands.InvalidRequest))
	}
	f.code(t, "unused term", termWrite(f, w.author, "bind_term", w.c, map[string]string{"binding": "closed-binding", "term": "absent-term", "term_digest": digest, "from": "2026-11-01", "reason": "initial"}), 422, string(commands.PrerequisiteFailed))
	if len(a.bindingIntents) != 0 {
		t.Fatal("a refused binding was reserved")
	}
	f.ok(t, "bind", termWrite(f, w.author, "bind_term", w.c, bind("first", "2026-11-01", "2026-12-31")), "pending", "")
	f.code(t, "overlapping a pending binding", termWrite(f, w.author, "bind_term", w.c, bind("overlap", "2026-12-15", "2027-01-31")), 422, string(commands.PrerequisiteFailed))
	f.code(t, "identifier reused with another range", termWrite(f, w.author, "bind_term", w.c, bind("first", "2027-01-01", "2027-01-31")), 409, string(commands.EvidenceStale))
	f.ok(t, "open-ended", termWrite(f, w.author, "bind_term", w.c, bind("open", "2027-01-01", "")), "pending", "")
	f.code(t, "any range after an open-ended binding", termWrite(f, w.author, "bind_term", w.c, bind("after-open", "2030-01-01", "2030-12-31")), 422, string(commands.PrerequisiteFailed))
	if len(a.bindingIntents) != 2 {
		t.Fatal("a refused range was reserved")
	}
}

func TestFinancialTermWritesZoneAndConfigurationBinding(t *testing.T) {
	w := publicTerms(t)
	f, a := w.f, w.f.s.financialIssuance
	digest := w.approvedTerm(t)
	f.ok(t, "bind", termWrite(f, w.author, "bind_term", w.c, map[string]string{"binding": "first", "term": "public-term", "term_digest": digest, "from": "2026-11-01", "to": "2026-12-31", "reason": "initial"}), "pending", "")

	same := w.second(t, "same-zone-configuration", w.c.Spec.Zone)
	other := w.second(t, "other-zone-configuration", "Asia/Seoul")
	if other.Spec.Zone == w.c.Spec.Zone || same.Spec.Contract != w.c.Spec.Contract {
		t.Fatal("fixture zones or contracts differ from the intent")
	}
	stage := func(c financialIssuanceConfiguration, term string) string {
		recorded := termWrite(f, w.author, "record_term", c, map[string]string{"term": term, "net_days": "15"})
		f.ok(t, "record "+term, recorded, "pending", "")
		approved := termWrite(f, w.approver, "approve_term", c, map[string]string{"term": term, "term_digest": field(recorded, "term_digest")})
		f.ok(t, "approve "+term, approved, "approved", "")
		return field(approved, "term_digest")
	}
	sameDigest, otherDigest := stage(same, "same-zone-term"), stage(other, "other-zone-term")
	revision, bindings := f.s.Revision(), len(a.bindingIntents)
	mismatch := map[string]string{"binding": "zone-mismatch", "term": "other-zone-term", "term_digest": otherDigest, "from": "2028-01-01", "to": "2028-12-31", "reason": "initial"}
	f.code(t, "another zone for the same contract", termWrite(f, w.author, "bind_term", other, mismatch), 422, string(commands.PrerequisiteFailed))
	if _, reserved := a.bindingIntents["zone-mismatch"]; reserved || len(a.bindingIntents) != bindings || f.s.Revision() != revision {
		t.Fatal("zone refusal reserved a binding")
	}
	f.code(t, "overlap with another configuration's pending binding", termWrite(f, w.author, "bind_term", same, map[string]string{"binding": "cross-overlap", "term": "same-zone-term", "term_digest": sameDigest, "from": "2026-12-01", "to": "2027-01-31", "reason": "initial"}), 422, string(commands.PrerequisiteFailed))
	f.ok(t, "same zone and disjoint range", termWrite(f, w.author, "bind_term", same, map[string]string{"binding": "second", "term": "same-zone-term", "term_digest": sameDigest, "from": "2027-01-01", "to": "2027-12-31", "reason": "renegotiation"}), "pending", "")

	listed := termsRead(f, w.reader, w.c, nil)
	if listed.status != 200 || field(listed, "count") != "2" || field(listed, "term") != "public-term" {
		t.Fatal(listed.raw)
	}
	configurations := map[string]string{}
	for i := 0; i < 2; i++ {
		row := protectedSetupRow(t, listed, i)
		configurations[row["binding"].(string)] = row["configuration"].(string)
	}
	if configurations["first"] != w.c.Spec.ID || configurations["second"] != same.Spec.ID {
		t.Fatalf("rows do not name their configurations %v", configurations)
	}

	revision = f.s.Revision()
	for name, tc := range map[string]struct {
		kind   string
		who    actor
		fields map[string]string
	}{
		"record naming another configuration's term":  {"record_term", w.author, map[string]string{"term": "public-term", "net_days": "30"}},
		"approve another configuration's term":        {"approve_term", w.approver, map[string]string{"term": "public-term", "term_digest": digest}},
		"bind with another configuration's term":      {"bind_term", w.author, map[string]string{"binding": "cross-term", "term": "public-term", "term_digest": digest, "from": "2029-01-01", "to": "2029-12-31", "reason": "initial"}},
		"bind naming another configuration's binding": {"bind_term", w.author, map[string]string{"binding": "first", "term": "same-zone-term", "term_digest": sameDigest, "from": "2029-01-01", "to": "2029-12-31", "reason": "initial"}},
		"approve another configuration's binding":     {"approve_term_binding", w.approver, map[string]string{"binding": "first", "binding_digest": a.bindingIntents["first"].Digest}},
	} {
		f.code(t, name, termWrite(f, tc.who, tc.kind, same, tc.fields), 409, string(commands.EvidenceStale))
	}
	foreign := same
	foreign.Spec.Contract = "sample-contract-02"
	f.code(t, "foreign contract", termWrite(f, w.approver, "approve_term_binding", foreign, map[string]string{"binding": "first", "binding_digest": a.bindingIntents["first"].Digest}), 403, string(commands.PermissionDenied))
	f.code(t, "missing binding", termWrite(f, w.approver, "approve_term_binding", w.c, map[string]string{"binding": "absent", "binding_digest": digest}), 409, string(commands.EvidenceStale))
	f.code(t, "missing term", termWrite(f, w.approver, "approve_term", w.c, map[string]string{"term": "absent", "term_digest": digest}), 409, string(commands.EvidenceStale))
	if f.s.Revision() != revision || a.bindingIntents["first"].State != "pending" || a.bindingIntents["first"].Approver != "" {
		t.Fatal("a refused cross-configuration request changed state")
	}
	f.ok(t, "approve under its own configuration", termWrite(f, w.approver, "approve_term_binding", w.c, map[string]string{"binding": "first", "binding_digest": a.bindingIntents["first"].Digest}), "approved", "")
}

func TestFinancialTermsReadIntegrityPagingAndStanding(t *testing.T) {
	w := publicTerms(t)
	f, a := w.f, w.f.s.financialIssuance
	digest := w.approvedTerm(t)
	for i := 1; i <= 11; i++ {
		fields := map[string]string{"binding": fmt.Sprintf("month-%02d", i), "term": "public-term", "term_digest": digest, "from": fmt.Sprintf("2027-%02d-01", i), "to": fmt.Sprintf("2027-%02d-28", i), "reason": "initial"}
		f.ok(t, fields["binding"], termWrite(f, w.author, "bind_term", w.c, fields), "pending", "")
	}
	first := termsRead(f, w.reader, w.c, nil)
	if first.status != 200 || field(first, "count") != "11" || field(first, "next") != "month-10" || field(first, "row_10") != "" {
		t.Fatal(first.raw)
	}
	cursor := map[string]string{"after": field(first, "next"), "snapshot": field(first, "snapshot")}
	last := termsRead(f, w.reader, w.c, cursor)
	if protectedSetupRow(t, last, 0)["binding"] != "month-11" || field(last, "next") != "" || field(last, "term_state") != "approved" {
		t.Fatal(last.raw)
	}
	f.code(t, "cursor without snapshot", termsRead(f, w.reader, w.c, map[string]string{"after": "month-10"}), 422, string(commands.InvalidRequest))
	f.code(t, "unknown cursor", termsRead(f, w.reader, w.c, map[string]string{"after": "absent", "snapshot": cursor["snapshot"]}), 409, string(commands.EvidenceStale))
	f.ok(t, "approve a later row", termWrite(f, w.approver, "approve_term_binding", w.c, map[string]string{"binding": "month-11", "binding_digest": a.bindingIntents["month-11"].Digest}), "approved", "")
	f.code(t, "changed snapshot", termsRead(f, w.reader, w.c, cursor), 409, string(commands.EvidenceStale))
	stale := w.c
	stale.Digest = strings.Repeat("0", 64)
	f.code(t, "stale configuration", termsRead(f, w.reader, stale, nil), 409, string(commands.EvidenceStale))
	missing := w.c
	missing.Spec.ID = "missing-configuration"
	f.code(t, "unknown configuration", termsRead(f, w.reader, missing, nil), 403, string(commands.PermissionDenied))

	f.s.fault = func(at string) bool { return at == "issuance-setup-terms-lookup" }
	f.code(t, "lookup fault is not absence", termsRead(f, w.reader, w.c, nil), 503, string(commands.DependencyUnavailable))
	f.s.fault = nil

	for _, grant := range f.s.admin.Grants() {
		if grant.SubjectID == "term-author" && grant.Terms.Kind == termsauthority.ResourceKind {
			f.must(f.s.admin.RevokeGrant(grant.ID, grant.Version))
		}
	}
	f.code(t, "approval after the author lost standing", termWrite(f, w.approver, "approve_term_binding", w.c, map[string]string{"binding": "month-01", "binding_digest": a.bindingIntents["month-01"].Digest}), 403, string(commands.PermissionDenied))
	if historical := termsRead(f, w.reader, w.c, nil); historical.status != 200 || field(historical, "count") != "11" || field(historical, "term_author") != "term-author" {
		t.Fatal("lost standing hid retained history", historical.raw)
	}

	for _, mode := range []string{"binding digest", "embedded term", "term digest", "term rewritten consistently", "withdrawn in the module"} {
		saved, term := a.bindingIntents["month-02"], a.termIntents["public-term"]
		switch mode {
		case "binding digest":
			changed := saved
			changed.Binding.To = "2027-02-27"
			a.bindingIntents["month-02"] = changed
		case "embedded term":
			changed := saved
			changed.Term.Term.NetDays = 60
			a.bindingIntents["month-02"] = changed
		case "term digest":
			changed := term
			changed.Term.NetDays = 60
			a.termIntents["public-term"] = changed
		case "term rewritten consistently":
			changed := term
			changed.Spec.Reason = "Rewritten after the binding was made"
			var err error
			changed.Digest, err = termIntentDigest(changed)
			f.must(err)
			a.termIntents["public-term"] = changed
		case "withdrawn in the module":
			if _, err := a.terms.WithdrawBinding(SampleWorkspace, saved.Binding.ID, saved.Author); err != nil {
				t.Fatal(err)
			}
		}
		f.code(t, mode, termsRead(f, w.reader, w.c, nil), 409, string(commands.EvidenceStale))
		a.bindingIntents["month-02"], a.termIntents["public-term"] = saved, term
	}
}

func TestFinancialTermWritesOwnershipSeparationAndPendingActors(t *testing.T) {
	w := publicTerms(t)
	f, a := w.f, w.f.s.financialIssuance
	record := map[string]string{"term": "public-term", "net_days": "30"}
	f.s.fault = func(at string) bool { return at == "issuance-term-after-register" }
	if got := termWrite(f, w.author, "record_term", w.c, record); got.status != 500 || got.body["operation_id"] == nil {
		t.Fatal(got.raw)
	}
	f.s.fault = nil
	grantTermHuman(f, termsauthority.RecordTerm, "term-approver")
	f.code(t, "another actor recovers the term", termWrite(f, w.approver, "record_term", w.c, record), 409, string(commands.EvidenceStale))
	if a.termIntents["public-term"].State != "registration_unknown" || a.termIntents["public-term"].Author != "term-author" {
		t.Fatal("another actor changed the uncertain term")
	}
	recovered := termWrite(f, w.author, "record_term", w.c, record)
	f.ok(t, "author recovers", recovered, "pending", "")
	digest := field(recovered, "term_digest")

	pinned := a.termIntents["public-term"]
	pinned.State, pinned.Approver = "approval_unknown", "term-approver"
	a.termIntents["public-term"] = pinned
	read := termsRead(f, w.reader, w.c, nil)
	if read.status != 200 || field(read, "term_state") != "approval_unknown" || field(read, "term_observation") != "pending" || field(read, "term_pending_approver") != "term-approver" || field(read, "term_approver") != "" || field(read, "term_approved_at") != "" {
		t.Fatal("pinned approver shown as completed", read.raw)
	}
	for name, change := range map[string]func(*financialTermRecord){"pin on a pending term": func(p *financialTermRecord) { p.State = "pending" }, "pin equal to the author": func(p *financialTermRecord) { p.Approver = "term-author" }} {
		broken := pinned
		change(&broken)
		a.termIntents["public-term"] = broken
		f.code(t, name, termsRead(f, w.reader, w.c, nil), 409, string(commands.EvidenceStale))
	}
	a.termIntents["public-term"] = pinned
	f.ok(t, "pinned approver completes", termWrite(f, w.approver, "approve_term", w.c, map[string]string{"term": "public-term", "term_digest": digest}), "approved", "")

	bind := map[string]string{"binding": "public-binding", "term": "public-term", "term_digest": digest, "from": "2026-11-01", "reason": "initial"}
	f.s.fault = func(at string) bool { return at == "issuance-binding-after-register" }
	if got := termWrite(f, w.author, "bind_term", w.c, bind); got.status != 500 || got.body["operation_id"] == nil {
		t.Fatal(got.raw)
	}
	f.s.fault = nil
	grantTermHuman(f, termsauthority.BindTerm, "term-approver")
	f.code(t, "another actor recovers the binding", termWrite(f, w.approver, "bind_term", w.c, bind), 409, string(commands.EvidenceStale))
	bound := termWrite(f, w.author, "bind_term", w.c, bind)
	f.ok(t, "author recovers", bound, "pending", "")
	grantTermHuman(f, termsauthority.ApproveBinding, "term-author")
	approve := map[string]string{"binding": "public-binding", "binding_digest": field(bound, "binding_digest")}
	f.code(t, "author approves own binding", termWrite(f, w.author, "approve_term_binding", w.c, approve), 403, string(commands.SelfApprovalDenied))
	if a.bindingIntents["public-binding"].State != "pending" || a.bindingIntents["public-binding"].Approver != "" {
		t.Fatal("self approval reserved")
	}
	held := a.bindingIntents["public-binding"]
	held.State, held.Approver = "approval_unknown", "term-approver"
	a.bindingIntents["public-binding"] = held
	row := protectedSetupRow(t, termsRead(f, w.reader, w.c, nil), 0)
	if row["state"] != "approval_unknown" || row["observation"] != "pending" || row["pending_approver"] != "term-approver" || row["approver"] != "" || row["approved_at"] != "" {
		t.Fatalf("pinned binding approver shown as completed %v", row)
	}
	f.ok(t, "pinned approver completes", termWrite(f, w.approver, "approve_term_binding", w.c, approve), "approved", "")

	key, err := termScopeKey(a.termIntents["public-term"])
	f.must(err)
	delete(a.termScopes, key)
	f.code(t, "term missing from the scope index", termsRead(f, w.reader, w.c, nil), 409, string(commands.EvidenceStale))
	a.termScopes[key] = "public-term"
	saved := a.termIntents["public-term"]
	delete(a.termIntents, "public-term")
	delete(a.bindingIntents, "public-binding")
	f.code(t, "scope index names a term that is not retained", termsRead(f, w.reader, w.c, nil), 409, string(commands.EvidenceStale))
	a.termIntents["public-term"] = saved
}

func TestFinancialTermAuthorityEdges(t *testing.T) {
	for _, mode := range []string{"inspect grant", "inspect ceiling"} {
		t.Run(mode, func(t *testing.T) {
			w := publicTerms(t)
			f := w.f
			if got := termsRead(f, w.reader, w.c, nil); got.status != 200 {
				t.Fatal(got.raw)
			}
			revoked := 0
			for _, grant := range f.s.admin.Grants() {
				_, ceiling := f.s.termAuthority.ceilings[grant.ID]
				if grant.Active && grant.Terms.Kind == termsauthority.ResourceKind && len(grant.Terms.Fields) != 0 && ceiling == (mode == "inspect ceiling") {
					f.must(f.s.admin.RevokeGrant(grant.ID, grant.Version))
					revoked++
				}
			}
			if revoked != 1 {
				t.Fatalf("revoked %d grants", revoked)
			}
			f.code(t, "read after revocation", termsRead(f, w.reader, w.c, nil), 403, string(commands.PermissionDenied))
		})
	}
	w := publicTerms(t)
	f := w.f
	terms, err := termAuthorityTerms(SampleWorkspace, termsauthority.Inspect)
	f.must(err)
	validity := gate.Validity{NotBefore: f.s.clock(), NotAfter: f.s.clock().Add(time.Hour)}
	f.must(f.s.admin.AddGrant(gate.Grant{ID: "terms-delegation-ceiling", ClientID: ConsoleClient, Terms: terms, Validity: validity}))
	f.must(f.s.admin.AddGrant(gate.Grant{ID: "terms-delegator", SubjectID: SampleAdmin, Terms: terms, Validity: validity}))
	f.must(f.s.admin.AddDelegation(gate.Delegation{ID: "terms-inspection-delegation", DelegatorID: SampleAdmin, DelegateID: "term-author", Terms: terms, Validity: validity}))
	f.must(inputAuthorityAs(f, w.author, func() error {
		req := gate.Request{Action: gate.ViewCost, Kind: termsauthority.ResourceKind, Fields: []string{termsauthority.EvidenceField}, Resource: gate.Resource{WorkspaceID: SampleWorkspace}}
		if got := f.s.gate.Decide(f.s.answer, ConsoleClient, req); !got.Allowed || len(got.Delegations) == 0 {
			t.Fatal("fixture did not establish actual delegated terms inspection")
		}
		return nil
	}))
	f.code(t, "delegated terms inspection", termsRead(f, w.author, w.c, nil), 403, string(commands.PermissionDenied))

	digest := strings.Repeat("a", 64)
	valid := map[string]map[string]string{
		"record_term": {"term": "edge-term", "net_days": "30"}, "approve_term": {"term": "edge-term", "term_digest": digest},
		"bind_term": {"binding": "edge-binding", "term": "edge-term", "term_digest": digest, "from": "2026-11-01", "reason": "initial"}, "approve_term_binding": {"binding": "edge-binding", "binding_digest": digest},
	}
	holder := map[string]actor{"record_term": w.author, "approve_term": w.approver, "bind_term": w.author, "approve_term_binding": w.approver}
	channel := activateProtectedMember(f, "channel-registrar")
	grantChannelCapability(f, protectedauthority.RegisterChannel, "channel-registrar")
	bindOnly := activateProtectedMember(f, "bind-only")
	grantTermHuman(f, termsauthority.BindTerm, "bind-only")
	for _, step := range termWriteSteps {
		f.code(t, step.Kind+" by a protected capability", termWrite(f, f.as(channel), step.Kind, w.c, valid[step.Kind]), 403, string(commands.PermissionDenied))
		if step.Kind != "bind_term" {
			f.code(t, step.Kind+" by a bind-only holder", termWrite(f, f.as(bindOnly), step.Kind, w.c, valid[step.Kind]), 403, string(commands.PermissionDenied))
		}
	}
	for _, step := range termWriteSteps {
		who := holder[step.Kind]
		f.s.endProbe(who.verifier(f))
		f.code(t, step.Kind+" with an ended session", termWrite(f, who, step.Kind, w.c, valid[step.Kind]), 401, string(commands.Unauthenticated))
	}
	f.s.cfg.Seed = SeedNone
	for _, step := range termWriteSteps {
		f.code(t, step.Kind+" outside the sample installation", termWrite(f, f.as(AdminLogin), step.Kind, w.c, valid[step.Kind]), 501, string(commands.UnsupportedCapability))
	}
}
