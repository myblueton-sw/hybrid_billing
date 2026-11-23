package server

import (
	"errors"
	"maps"
	"reflect"
	"sort"
	"strconv"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/receivables"
	"github.com/myblueton-sw/hybrid_billing/modules/termsauthority"
)

type termWriteStep struct {
	Kind, Capability string
	Action           gate.Action
	Rationale, Self  bool
	Fields, Optional []string
}

var termWriteSteps = []termWriteStep{
	{Kind: "record_term", Capability: termsauthority.RecordTerm, Action: gate.RecordIssuanceTerm, Rationale: true, Fields: []string{"term", "net_days"}},
	{Kind: "approve_term", Capability: termsauthority.ApproveTerm, Action: gate.ApproveIssuanceTerm, Self: true, Fields: []string{"term", "term_digest"}},
	{Kind: "bind_term", Capability: termsauthority.BindTerm, Action: gate.BindIssuanceTerm, Fields: []string{"binding", "term", "term_digest", "from", "reason"}, Optional: []string{"to"}},
	{Kind: "approve_term_binding", Capability: termsauthority.ApproveBinding, Action: gate.ApproveIssuanceTermBinding, Self: true, Fields: []string{"binding", "binding_digest"}},
}

func termWriteCommand(kind string) (termWriteStep, bool) {
	for _, step := range termWriteSteps {
		if step.Kind == kind {
			return step, true
		}
	}
	return termWriteStep{}, false
}

func (s *Server) termSetupDefinitions() []commands.Definition {
	scope := []commands.Field{{Name: "configuration", Type: commands.Text, Required: true}, {Name: "digest", Type: commands.Text, Required: true}}
	read := commands.Permission{Action: gate.ViewCost, Kind: termsauthority.ResourceKind, Fields: []string{termsauthority.EvidenceField}}
	out := []commands.Definition{{ID: "issuance_setup.terms", Permission: read, Fields: append(append([]commands.Field{}, scope...), commands.Field{Name: "after", Type: commands.Text}, commands.Field{Name: "snapshot", Type: commands.Text}), Refusals: []commands.Code{commands.EvidenceStale}, Handler: issuanceSetupContextCommand{s: s, kind: "terms"}}}
	for _, step := range termWriteSteps {
		fields := append([]commands.Field{}, scope...)
		for _, name := range step.Fields {
			fields = append(fields, commands.Field{Name: name, Type: commands.Text, Required: true})
		}
		for _, name := range step.Optional {
			fields = append(fields, commands.Field{Name: name, Type: commands.Text})
		}
		refusals := []commands.Code{commands.EvidenceStale}
		if step.Self {
			refusals = append(refusals, commands.SelfApprovalDenied)
		}
		out = append(out, commands.Definition{ID: "issuance_setup." + step.Kind, Permission: commands.Permission{Action: step.Action, Kind: termsauthority.ResourceKind}, Mutating: true, Rationale: step.Rationale, Fields: fields, Refusals: refusals, Handler: issuanceSetupContextCommand{s: s, kind: step.Kind}})
	}
	return out
}

func termWriteError(self bool, err error) error {
	var failure *commands.Failure
	if errors.As(err, &failure) {
		switch failure.Code {
		case commands.InvalidRequest, commands.PermissionDenied, commands.PrerequisiteFailed, commands.DependencyUnavailable, commands.UnsupportedCapability, commands.AuthorityUnavailable, commands.EvidenceStale, commands.RevisionConflict, commands.InvalidStage:
			return err
		case commands.SelfApprovalDenied:
			if self {
				return err
			}
		}
		return commands.Fail(commands.PermissionDenied)
	}
	switch {
	case errors.Is(err, receivables.ErrConflict), errors.Is(err, receivables.ErrInvalid):
		return commands.Fail(commands.EvidenceStale)
	case errors.Is(err, receivables.ErrRejected), errors.Is(err, receivables.ErrLimit), errors.Is(err, receivables.ErrNotFound):
		return commands.Fail(commands.PrerequisiteFailed)
	}
	return commands.Fail(commands.DependencyUnavailable)
}

func termOfConfiguration(p financialTermRecord, c financialIssuanceConfiguration) bool {
	return p.Configuration != nil && reflect.DeepEqual(*p.Configuration, c)
}

func termModuleID(id string) string {
	digest, _ := financialEvidenceDigest("issuance_term", id)
	return "issued-term-" + digest
}

func bindingModuleID(id string) string {
	digest, _ := financialEvidenceDigest("issuance_binding", id)
	return "issued-binding-" + digest
}

type termWriteObservation struct {
	Term    *financialTermRecord
	Scopes  map[string]string
	Binding *financialTermBinding
	Module  any
	Missing bool
}

func (a *financialIssuance) observeTermWrite(workspace, family, id string) termWriteObservation {
	var out termWriteObservation
	var err error
	if family == "term" {
		if p, ok := a.termIntents[id]; ok {
			p = copyFinancialTerm(p)
			out.Term = &p
		}
		out.Scopes = maps.Clone(a.termScopes)
		out.Module, err = a.terms.Term(workspace, termModuleID(id), 1)
	} else {
		if p, ok := a.bindingIntents[id]; ok {
			p = copyFinancialBinding(p)
			out.Binding = &p
		}
		out.Module, err = a.terms.BindingOf(workspace, bindingModuleID(id))
	}
	out.Missing = err != nil
	return out
}

func (a *financialIssuance) applyTermWrite(kind string, ctx commands.Context) (commands.Outcome, error) {
	step, ok := termWriteCommand(kind)
	if !ok {
		return commands.Outcome{}, commands.Fail(commands.UnsupportedCapability)
	}
	workspace, actor, contract, f := ctx.Workspace, ctx.User, ctx.Target, ctx.Fields
	if err := a.termCaller(workspace, actor, step.Capability); err != nil {
		return commands.Outcome{}, termWriteError(step.Self, err)
	}
	invalid := func() (commands.Outcome, error) { return commands.Outcome{}, commands.Fail(commands.InvalidRequest) }
	stale := func() (commands.Outcome, error) { return commands.Outcome{}, commands.Fail(commands.EvidenceStale) }
	identifier := func(text string) bool { return validRequestID(text) && len(text) <= 128 }
	if !validRequestID(contract) || !validRequestID(f["configuration"]) || !validPlanHash(f["digest"]) || step.Rationale && !documentReason(ctx.Rationale) {
		return invalid()
	}
	family, id := "term", f["term"]
	if kind == "bind_term" || kind == "approve_term_binding" {
		family, id = "binding", f["binding"]
	}
	netDays := 0
	switch kind {
	case "record_term":
		n, err := strconv.Atoi(f["net_days"])
		if !identifier(id) || err != nil || strconv.Itoa(n) != f["net_days"] || n < 0 || n > 365 {
			return invalid()
		}
		netDays = n
	case "approve_term":
		if !identifier(id) || !validPlanHash(f["term_digest"]) {
			return invalid()
		}
	case "bind_term":
		reason := f["reason"]
		if !identifier(id) || !identifier(f["term"]) || !validPlanHash(f["term_digest"]) || !termBindingDate(f["from"]) || f["to"] != "" && (!termBindingDate(f["to"]) || f["to"] < f["from"]) || reason != "initial" && reason != "renegotiation" && reason != "correction" {
			return invalid()
		}
	default:
		if !identifier(id) || !validPlanHash(f["binding_digest"]) {
			return invalid()
		}
	}
	c, err := a.confirmedSetupConfiguration(workspace, contract, f["configuration"], f["digest"])
	if err != nil {
		return commands.Outcome{}, termWriteError(step.Self, err)
	}
	reference := configurationReference(c)
	head := map[string]string{"contract": contract, "configuration": c.Spec.ID, "current_eligibility": "unchecked"}
	termName := id
	if kind == "bind_term" {
		termName = f["term"]
	}
	if family == "term" || kind == "bind_term" {
		if old, exists := a.termIntents[termName]; exists && !termOfConfiguration(old, c) {
			return stale()
		}
	}
	if family == "binding" {
		old, exists := a.bindingIntents[id]
		if exists && !termOfConfiguration(old.Term, c) {
			return stale()
		}
		if kind == "bind_term" && !exists {
			if len(c.Spec.Zone) > 64 {
				return commands.Outcome{}, commands.Fail(commands.PrerequisiteFailed)
			}
			for _, other := range a.bindingIntents {
				if other.Binding.Contract == c.Scope.Contract.ContractID && other.Binding.Zone != c.Spec.Zone {
					return commands.Outcome{}, commands.Fail(commands.PrerequisiteFailed)
				}
			}
		}
	}
	termReceipt := func(p financialTermRecord, err error) (string, error) {
		head["schema"], head["term"], head["term_digest"], head["state"] = "hb_issuance_term_receipt_v1", p.Spec.ID, p.Digest, p.State
		return p.State, err
	}
	bindingReceipt := func(p financialTermBinding, err error) (string, error) {
		head["schema"], head["binding"], head["binding_digest"], head["state"] = "hb_issuance_term_binding_receipt_v1", p.Spec.ID, p.Digest, p.State
		return p.State, err
	}
	before := a.observeTermWrite(workspace, family, id)
	var state string
	switch kind {
	case "record_term":
		state, err = termReceipt(a.recordTerm(workspace, actor, financialTermSpec{Configuration: &reference, ID: id, Mode: receivables.ModeCalendar, Reason: ctx.Rationale, Revision: 1, NetDays: netDays}))
	case "approve_term":
		state, err = termReceipt(a.approveTerm(workspace, id, f["term_digest"], actor))
	case "bind_term":
		state, err = bindingReceipt(a.bindTerm(workspace, actor, financialTermBindingSpec{ID: id, Term: f["term"], TermDigest: f["term_digest"], From: f["from"], To: f["to"], Zone: c.Spec.Zone, Reason: f["reason"]}))
	default:
		state, err = bindingReceipt(a.approveTermBinding(workspace, id, f["binding_digest"], actor))
	}
	changed := !reflect.DeepEqual(before, a.observeTermWrite(workspace, family, id))
	if err != nil {
		if changed {
			return commands.Outcome{}, errConfigurationUncertain
		}
		return commands.Outcome{}, termWriteError(step.Self, err)
	}
	if err := a.boundedDraftFields(head); err != nil || state == "" {
		if changed {
			return commands.Outcome{}, errConfigurationUncertain
		}
		return commands.Outcome{}, commands.Fail(commands.DependencyUnavailable)
	}
	return commands.Outcome{Fields: head, BusinessState: state, ResultRevision: a.server.Next()}, nil
}

func termObservation(state, moduleState, approver, pinned, author string, absent, empty bool) (string, bool) {
	if pinned != "" && (!validRequestID(pinned) || pinned == author) {
		return "", false
	}
	if absent {
		return "absent", state == "registration_unknown" && empty && pinned == ""
	}
	if empty {
		return "", false
	}
	switch {
	case moduleState == receivables.StatePending && approver == "":
		return "pending", state == "registration_unknown" && pinned == "" || state == "pending" && pinned == "" || state == "approval_unknown" && pinned != ""
	case moduleState == receivables.StateApproved && approver != "" && approver == pinned:
		return "approved", state == "approval_unknown" || state == "approved"
	}
	return "", false
}

func (a *financialIssuance) setupTermFields(workspace, actor, contract string, input map[string]string) (map[string]string, error) {
	if err := a.termReadCaller(workspace, actor); err != nil {
		return nil, err
	}
	after, snapshot := input["after"], input["snapshot"]
	if !validRequestID(contract) || !validRequestID(input["configuration"]) || !validPlanHash(input["digest"]) || after != "" && !validRequestID(after) {
		return nil, commands.Fail(commands.InvalidRequest)
	}
	if err := draftCursor(after, snapshot); err != nil {
		return nil, err
	}
	c, err := a.confirmedSetupConfiguration(workspace, contract, input["configuration"], input["digest"])
	if err != nil {
		return nil, err
	}
	bad := func() (map[string]string, error) { return nil, commands.Fail(commands.EvidenceStale) }
	lookup := func() error {
		if a.terms == nil || a.server.faulty("issuance-setup-terms-lookup") {
			return commands.Fail(commands.DependencyUnavailable)
		}
		return nil
	}
	if len(a.termIntents) > financialIssuanceLimit || len(a.bindingIntents) > financialIssuanceLimit {
		return nil, commands.Fail(commands.PrerequisiteFailed)
	}
	head := map[string]string{"schema": "hb_issuance_setup_terms_v1", "contract": contract, "configuration": c.Spec.ID, "digest": c.Digest, "zone": c.Spec.Zone, "term_state": "none", "term_observation": "absent"}
	for _, name := range []string{"term", "term_digest", "term_revision", "net_days", "mode", "term_author", "term_approver", "term_pending_approver", "term_written_at", "term_approved_at"} {
		head[name] = ""
	}
	var owned *financialTermRecord
	for id, p := range a.termIntents {
		if !termOfConfiguration(p, c) {
			continue
		}
		if owned != nil || p.Spec.ID != id {
			return bad()
		}
		p = copyFinancialTerm(p)
		owned = &p
	}
	scope, err := financialEvidenceDigest("issuance_configuration_term_scope_v1", struct{ Workspace, ID, Digest string }{c.Workspace, c.Spec.ID, c.Digest})
	if err != nil || owned == nil && a.termScopes["configured-term-"+scope] != "" {
		return bad()
	}
	if owned != nil {
		p := *owned
		digest, err := termIntentDigest(p)
		if err != nil || digest != p.Digest || !validRequestID(p.Author) || a.termScopeOwned(p) != nil {
			return bad()
		}
		if err := lookup(); err != nil {
			return nil, err
		}
		v, err := a.terms.Term(workspace, p.Term.ID, p.Term.Revision)
		absent := errors.Is(err, receivables.ErrNotFound)
		if err != nil && !absent {
			return nil, commands.Fail(commands.DependencyUnavailable)
		}
		if !absent && (v != p.Module || v.Term != p.Term || v.Author != p.Author) {
			return bad()
		}
		observation, valid := termObservation(p.State, v.State, v.Approver, p.Approver, p.Author, absent, p.Module == (receivables.TermView{}))
		if !valid {
			return bad()
		}
		head["term_state"], head["term_observation"], head["term"], head["term_digest"], head["term_revision"], head["net_days"], head["mode"], head["term_author"] = p.State, observation, p.Spec.ID, p.Digest, strconv.Itoa(p.Term.Revision), strconv.Itoa(p.Term.NetDays), p.Term.Mode, p.Author
		head["term_approver"], head["term_written_at"], head["term_approved_at"] = v.Approver, setupInstant(v.Written), setupInstant(v.ApprovedAt)
		if v.Approver == "" {
			head["term_pending_approver"] = p.Approver
		}
	}
	ids := []string{}
	for id, p := range a.bindingIntents {
		if p.Binding.Contract == c.Scope.Contract.ContractID {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	page := struct {
		Workspace, Actor, Contract string
		Configuration              financialConfigurationReference
		Head                       map[string]string
		Rows                       []string
		Records                    []financialTermBinding
	}{Workspace: workspace, Actor: actor, Contract: contract, Configuration: configurationReference(c), Head: maps.Clone(head)}
	for _, id := range ids {
		p := a.bindingIntents[id]
		digest, err := bindingIntentDigest(p)
		term, known := a.termIntents[p.Spec.Term]
		if err != nil || digest != p.Digest || p.Spec.ID != id || !validRequestID(p.Author) || !known || term.State != "approved" || !reflect.DeepEqual(term, p.Term) {
			return bad()
		}
		if err := lookup(); err != nil {
			return nil, err
		}
		v, err := a.terms.BindingOf(workspace, p.Binding.ID)
		absent := errors.Is(err, receivables.ErrNotFound)
		if err != nil && !absent {
			return nil, commands.Fail(commands.DependencyUnavailable)
		}
		empty := reflect.DeepEqual(p.Module, receivables.BindingView{})
		if !absent && (!reflect.DeepEqual(v, p.Module) || v.Binding != p.Binding || v.Author != p.Author || v.WithdrawnBy != "" || v.PendingEnd != "" || v.EndAuthor != "" || len(v.Ends) != 0) {
			return bad()
		}
		observation, valid := termObservation(p.State, v.State, v.Approver, p.Approver, p.Author, absent, empty)
		if !valid {
			return bad()
		}
		row := map[string]string{
			"binding": id, "digest": p.Digest, "origin": "profile", "configuration": "", "term": p.Spec.Term, "term_revision": strconv.Itoa(p.Binding.Revision), "net_days": strconv.Itoa(p.Term.Term.NetDays), "mode": p.Term.Term.Mode,
			"zone": p.Binding.Zone, "from": p.Binding.From, "to": p.Binding.To, "reason": p.Binding.Reason, "state": p.State, "observation": observation, "author": p.Author, "approver": v.Approver, "pending_approver": "",
			"written_at": setupInstant(v.Written), "approved_at": setupInstant(v.ApprovedAt),
		}
		if p.Term.Configuration != nil {
			row["origin"], row["configuration"] = "configuration", p.Term.Configuration.Spec.ID
		}
		if v.Approver == "" {
			row["pending_approver"] = p.Approver
		}
		raw, err := draftJSON(row)
		if err != nil {
			return nil, err
		}
		if len(raw) > 4096 {
			return nil, commands.Fail(commands.PrerequisiteFailed)
		}
		page.Rows, page.Records = append(page.Rows, raw), append(page.Records, copyFinancialBinding(p))
	}
	digest, err := financialEvidenceDigest("issuance_setup_terms_page_v1", page)
	if err != nil {
		return nil, commands.Fail(commands.DependencyUnavailable)
	}
	return a.setupProtectedPage(head, ids, page.Rows, after, snapshot, digest)
}
