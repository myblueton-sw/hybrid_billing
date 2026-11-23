package server

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
)

func setupInstant(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func protectedSetupField(name string) bool {
	for _, f := range protectedSetupFields {
		if f.Field == name {
			return true
		}
	}
	return false
}

func (a *financialIssuance) protectedSetupConfiguration(workspace, actor, contract, kind string, input map[string]string) (financialIssuanceConfiguration, error) {
	none := financialIssuanceConfiguration{}
	if err := a.protectedReadCaller(workspace, actor); err != nil {
		return none, err
	}
	id, digest, after, snapshot := input["configuration"], input["digest"], input["after"], input["snapshot"]
	if !validRequestID(contract) || !validRequestID(id) || !validPlanHash(digest) || kind == "protected" && (after != "" || snapshot != "") || after != "" && !validRequestID(after) || (kind == "protected_changes") != protectedSetupField(input["field"]) || kind != "protected_changes" && input["field"] != "" {
		return none, commands.Fail(commands.InvalidRequest)
	}
	if err := draftCursor(after, snapshot); err != nil {
		return none, err
	}
	if err := a.configurationHistoryIndex(); err != nil {
		return none, err
	}
	p, ok := a.configurations[id]
	if !ok || p.Workspace != workspace || p.Spec.Contract != contract || p.Spec.ID != id {
		return none, commands.Fail(commands.PermissionDenied)
	}
	if p.Digest != digest {
		return none, commands.Fail(commands.EvidenceStale)
	}
	observation, err := a.observeConfigurationHistory(p)
	if err != nil {
		return none, err
	}
	if p.State != "confirmed" || observation != "active" {
		return none, commands.Fail(commands.PrerequisiteFailed)
	}
	return copyConfiguration(p), nil
}

func (a *financialIssuance) observeProtectedRules(workspace string) (*financialProtectedRules, string, error) {
	bad := func() (*financialProtectedRules, string, error) {
		return nil, "", commands.Fail(commands.EvidenceStale)
	}
	if a.fields == nil || a.server.faulty("issuance-setup-protected-lookup") {
		return nil, "", commands.Fail(commands.DependencyUnavailable)
	}
	state, err := a.fields.RulesOf(workspace)
	missing := errors.Is(err, protected.ErrNotFound)
	if err != nil && !missing {
		return nil, "", commands.Fail(commands.DependencyUnavailable)
	}
	if a.protectedRules == nil {
		if missing {
			return nil, "absent", nil
		}
		return nil, "unowned", nil
	}
	p := *a.protectedRules
	digest, err := protectedRuleDigest(p)
	if err != nil || digest != p.Digest || p.Workspace != workspace || !validRequestID(p.ID) || !validRequestID(p.Proposer) || p.Confirmer == p.Proposer {
		return bad()
	}
	pinned := p.State == "confirmation_unknown" || p.State == "confirmed"
	if pinned != (p.Confirmer != "") || p.Confirmer != "" && !validRequestID(p.Confirmer) {
		return bad()
	}
	if missing {
		if p.State == "registration_unknown" && p.Module == (protected.RulesView{}) {
			return &p, "absent", nil
		}
		return bad()
	}
	m := p.Module
	if len(state.History) != 1 || state.History[0] != m || m.Rules != p.Rules || m.Proposer != p.Proposer || m.Revision != 1 || m.Discarder != "" || !m.Discarded.IsZero() || m.Proposed.IsZero() {
		return bad()
	}
	observation := ""
	switch {
	case state.Pending != nil && state.Active == nil && *state.Pending == m && m.Confirmer == "" && m.Confirmed.IsZero():
		observation = "pending"
	case state.Active != nil && state.Pending == nil && *state.Active == m && m.Confirmer != "" && m.Confirmer == p.Confirmer && !m.Confirmed.IsZero():
		observation = "active"
	default:
		return bad()
	}
	switch p.State {
	case "registration_unknown", "proposed":
		if observation != "pending" {
			return bad()
		}
	case "confirmation_unknown":
	case "confirmed":
		if observation != "active" {
			return bad()
		}
	default:
		return bad()
	}
	return &p, observation, nil
}

var protectedSetupFields = []struct{ Field, Prefix, Subject string }{
	{protected.FieldDocumentRecipient, "recipient", "customer"},
	{protected.FieldDocumentTemplate, "template", "document_kind"},
	{protected.FieldPaymentAccount, "account", "entity"},
}

func protectedSetupSubject(c financialIssuanceConfiguration, kind string) string {
	switch kind {
	case "customer":
		return "customer:" + c.Scope.Contract.CustomerID
	case "entity":
		return "entity:" + c.Scope.Document.IssuerLegalEntityID
	}
	return "kind:" + c.Kind.ID
}

func protectedSetupChannelKind(c financialIssuanceConfiguration, p financialChannelRecord) string {
	s := p.Subject
	if p.Workspace != c.Workspace || !reflect.DeepEqual(s.Workspace, c.Scope.Workspace) || s.WorkspaceVersion != c.Scope.WorkspaceVersion {
		return ""
	}
	switch p.Spec.Selector.Kind {
	case "customer":
		if reflect.DeepEqual(s.Contract, c.Scope.Contract) && s.Subject == protectedSetupSubject(c, "customer") {
			return "customer"
		}
	case "entity":
		if reflect.DeepEqual(s.BillingProfile, c.Scope.BillingProfile) && s.BillingVersion == c.Scope.BillingVersion && s.Subject == protectedSetupSubject(c, "entity") {
			return "entity"
		}
	case "configuration_kind":
		r := p.Spec.Selector.Configuration
		if r != nil && *r == configurationReference(c) && s.Configuration != nil && reflect.DeepEqual(*s.Configuration, c) && s.Subject == protectedSetupSubject(c, "document_kind") {
			return "document_kind"
		}
	}
	return ""
}

func (a *financialIssuance) observeSetupChannel(id string, p financialChannelRecord) (string, protected.ChannelView, error) {
	bad := func() (string, protected.ChannelView, error) {
		return "", protected.ChannelView{}, commands.Fail(commands.EvidenceStale)
	}
	digest, err := channelDigest(p)
	if err != nil || digest != p.Digest || p.Spec.ID != id || !validChannelSpec(p.Spec) || !reflect.DeepEqual(p.Spec.Selector, p.Subject.Selector) || !validRequestID(p.Registrar) {
		return bad()
	}
	if p.Admitter != "" && (p.Admitter == p.Registrar || !validRequestID(p.Admitter) || !validRequestID(p.Admission) || len(p.Admission) > 128) || p.Admitter == "" && p.Admission != "" {
		return bad()
	}
	if a.server.faulty("issuance-setup-protected-lookup") {
		return "", protected.ChannelView{}, commands.Fail(commands.DependencyUnavailable)
	}
	v, err := a.fields.ChannelOf(p.Workspace, id)
	if errors.Is(err, protected.ErrNotFound) {
		if p.State == "registration_unknown" && p.Module == (protected.ChannelView{}) && p.Admitter == "" {
			return "absent", protected.ChannelView{}, nil
		}
		return bad()
	}
	if err != nil {
		return "", protected.ChannelView{}, commands.Fail(commands.DependencyUnavailable)
	}
	live := v
	live.RetiredBy, live.Retired = "", time.Time{}
	if live != p.Module || v.Channel != p.channel() || v.Registrar != p.Registrar || v.Registered.IsZero() || (v.RetiredBy == "") != v.Retired.IsZero() {
		return bad()
	}
	open := v.Admitter == "" && v.Admission == "" && v.Admitted.IsZero()
	pinned := p.Admitter != "" && v.Admitter == p.Admitter && v.Admission == p.Admission && !v.Admitted.IsZero()
	switch p.State {
	case "registration_unknown", "registered":
		if p.Admitter != "" || !open {
			return bad()
		}
	case "admission_unknown":
		if p.Admitter == "" || !open && !pinned {
			return bad()
		}
	case "admitted":
		if !pinned {
			return bad()
		}
	default:
		return bad()
	}
	if v.RetiredBy != "" {
		return "retired", v, nil
	}
	if open {
		return "registered", v, nil
	}
	return "admitted", v, nil
}

type protectedSetupChannels struct {
	IDs            []string
	Rows           []string
	Records        []financialChannelRecord
	Observations   []string
	Other, Unowned int
}

func (a *financialIssuance) setupChannels(c financialIssuanceConfiguration, rules *financialProtectedRules, rulesObservation string) (protectedSetupChannels, error) {
	var out protectedSetupChannels
	if len(a.channels) > financialIssuanceLimit {
		return out, commands.Fail(commands.PrerequisiteFailed)
	}
	subjects := map[string]bool{}
	for _, f := range protectedSetupFields {
		subjects[protectedSetupSubject(c, f.Subject)] = true
	}
	ids := make([]string, 0, len(a.channels))
	for id, p := range a.channels {
		if p.Workspace == c.Workspace && subjects[p.Subject.Subject] {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		p := a.channels[id]
		observation, v, err := a.observeSetupChannel(id, p)
		if err != nil {
			return out, err
		}
		kind := protectedSetupChannelKind(c, p)
		if kind == "" {
			out.Other++
			continue
		}
		row := map[string]string{
			"channel": id, "digest": p.Digest, "subject_kind": kind, "kind": p.Spec.Kind, "state": p.State, "observation": observation,
			"registrar": p.Registrar, "admitter": v.Admitter, "pending_admitter": "", "registered_at": setupInstant(v.Registered), "admitted_at": setupInstant(v.Admitted),
			"retired_by": v.RetiredBy, "retired_at": setupInstant(v.Retired), "rules": "other", "request_not_before": "",
		}
		if v.Admitter == "" {
			row["pending_admitter"] = p.Admitter
		}
		if rules != nil && rules.Digest == p.RulesDigest {
			row["rules"] = "current"
			if rules.State == "confirmed" && rulesObservation == "active" && p.State == "admitted" && observation == "admitted" {
				row["request_not_before"] = setupInstant(v.Admitted.Add(rules.Rules.ChannelAge))
			}
		}
		raw, err := draftJSON(row)
		if err != nil {
			return out, err
		}
		if len(raw) > 4096 {
			return out, commands.Fail(commands.PrerequisiteFailed)
		}
		out.IDs, out.Rows, out.Records, out.Observations = append(out.IDs, id), append(out.Rows, raw), append(out.Records, copyChannelRecord(p)), append(out.Observations, observation)
	}
	names := make([]string, 0, len(subjects))
	for subject := range subjects {
		names = append(names, subject)
	}
	sort.Strings(names)
	for _, subject := range names {
		if len(subject) > 128 {
			continue
		}
		if a.server.faulty("issuance-setup-protected-lookup") {
			return out, commands.Fail(commands.DependencyUnavailable)
		}
		views, err := a.fields.ChannelsOf(c.Workspace, subject)
		if errors.Is(err, protected.ErrNotFound) {
			continue
		}
		if err != nil {
			return out, commands.Fail(commands.DependencyUnavailable)
		}
		for _, v := range views {
			p, ok := a.channels[v.Channel.ID]
			if !ok {
				out.Unowned++
			} else if p.Workspace != c.Workspace || p.Subject.Subject != subject {
				return out, commands.Fail(commands.EvidenceStale)
			}
		}
	}
	return out, nil
}

type protectedSetupChanges struct {
	IDs                  []string
	Rows                 []string
	Records              []financialChangeRecord
	Observations         []string
	Other, Unowned       int
	Scheduled, Effective string
}

func (a *financialIssuance) setupChanges(c financialIssuanceConfiguration, rules *financialProtectedRules, field, subject string) (protectedSetupChanges, error) {
	var out protectedSetupChanges
	bad := func() (protectedSetupChanges, error) {
		return protectedSetupChanges{}, commands.Fail(commands.EvidenceStale)
	}
	if len(a.changes) > financialIssuanceLimit {
		return out, commands.Fail(commands.PrerequisiteFailed)
	}
	ids := a.changeOrders[issuanceChangeScope(subject, field)]
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			return bad()
		}
		seen[id] = true
	}
	for id, p := range a.changes {
		if p.Workspace == c.Workspace && p.Channel.Subject.Subject == subject && p.Spec.Field == field && !seen[id] {
			return bad()
		}
	}
	if a.server.faulty("issuance-setup-protected-lookup") {
		return out, commands.Fail(commands.DependencyUnavailable)
	}
	var trail []protected.View
	var err error
	if len(subject) <= 128 {
		trail, err = a.fields.PeekTrail(c.Workspace, subject, field)
	}
	if err != nil && !errors.Is(err, protected.ErrNotFound) {
		return out, commands.Fail(commands.DependencyUnavailable)
	}
	at := 0
	for index, id := range ids {
		p, ok := a.changes[id]
		digest, e := changeDigest(p)
		if !ok || e != nil || digest != p.Digest || p.Workspace != c.Workspace || p.Spec.ID != id || p.Channel.Subject.Subject != subject || p.Spec.Field != field || !validRequestID(p.Requester) || !validChangeProgress(p) {
			return bad()
		}
		observation, v := "absent", protected.View{}
		if p.Module.Digest == "" {
			if p.Phase != "request_unknown" || index != len(ids)-1 {
				return bad()
			}
			if a.server.faulty("issuance-setup-protected-lookup") {
				return out, commands.Fail(commands.DependencyUnavailable)
			}
			if _, e := a.fields.Peek(c.Workspace, id); !errors.Is(e, protected.ErrNotFound) {
				return bad()
			}
		} else {
			if at == len(trail) || trail[at].Change.ID != id {
				return bad()
			}
			v = trail[at]
			at++
			if v.State == protected.StateWithdrawn || !changeViewEqual(v, p.Module) || v.Change != p.change() || v.Requester != p.Requester || v.Before != p.Before {
				return bad()
			}
			observation = v.State
		}
		if p.Configuration == nil || !reflect.DeepEqual(*p.Configuration, c) {
			out.Other++
			continue
		}
		if observation == protected.StateScheduled && p.Phase == "recorded" && p.Module.State == protected.StateScheduled {
			out.Scheduled, out.Effective = id, setupInstant(v.Change.Effective)
		}
		row := map[string]any{
			"change": id, "digest": p.Digest, "field": field, "phase": p.Phase, "state": p.Module.State, "observation": observation,
			"provenance": p.Spec.Provenance, "instruction_kind": p.Spec.Instruction.Kind, "channel": p.Spec.Channel, "requester": p.Requester,
			"requested_at": setupInstant(v.Requested), "effective": setupInstant(p.Spec.Effective), "approvals_required": "", "verifier": "", "verification_outcome": "",
			"verified_at": "", "pending_verifier": "", "pending_decision_actor": "", "ended_by": v.EndedBy, "ended_at": setupInstant(v.Ended), "rules": "other",
		}
		if rules != nil && rules.Digest == p.RulesDigest {
			row["rules"] = "current"
		}
		if observation != "absent" {
			row["approvals_required"] = strconv.Itoa(v.Rules.Approvals)
		}
		if v.Verification != nil {
			row["verifier"], row["verification_outcome"], row["verified_at"] = v.Verification.Verifier, v.Verification.Verification.Outcome, setupInstant(v.Verification.At)
		} else if p.Verification != nil {
			row["pending_verifier"] = p.Verification.Actor
		}
		decisions := make([]map[string]string, 0, len(v.Decisions))
		for _, d := range v.Decisions {
			decisions = append(decisions, map[string]string{"approver": d.Approver, "verdict": d.Approval.Verdict, "at": setupInstant(d.At)})
		}
		row["decisions"] = decisions
		if len(p.Decisions) > len(v.Decisions) {
			row["pending_decision_actor"] = p.Decisions[len(p.Decisions)-1].Actor
		}
		raw, e := draftJSON(row)
		if e != nil {
			return out, e
		}
		if len(raw) > 4096 {
			return out, commands.Fail(commands.PrerequisiteFailed)
		}
		out.IDs, out.Rows, out.Records, out.Observations = append(out.IDs, id), append(out.Rows, raw), append(out.Records, copyChange(p)), append(out.Observations, observation)
	}
	out.Unowned += len(trail) - at
	return out, nil
}

func (a *financialIssuance) setupProtectedPage(fields map[string]string, ids, rows []string, after, snapshot, digest string) (map[string]string, error) {
	if snapshot != "" && snapshot != digest {
		return nil, commands.Fail(commands.EvidenceStale)
	}
	start := 0
	if after != "" {
		start = slices.Index(ids, after) + 1
		if start == 0 {
			return nil, commands.Fail(commands.EvidenceStale)
		}
	}
	fields["current_eligibility"], fields["snapshot"], fields["count"], fields["next"] = "unchecked", digest, strconv.Itoa(len(rows)), ""
	end := min(start+10, len(rows))
	for i := start; i < end; i++ {
		fields[fmt.Sprintf("row_%02d", i-start)] = rows[i]
	}
	if end < len(rows) {
		fields["next"] = ids[end-1]
	}
	if err := a.boundedDraftFields(fields); err != nil {
		return nil, err
	}
	return fields, nil
}

func (a *financialIssuance) setupProtectedFields(workspace, actor, contract, kind string, input map[string]string) (map[string]string, error) {
	c, err := a.protectedSetupConfiguration(workspace, actor, contract, kind, input)
	if err != nil {
		return nil, err
	}
	after, snapshot := input["after"], input["snapshot"]
	rules, rulesObservation, err := a.observeProtectedRules(workspace)
	if err != nil {
		return nil, err
	}
	head := map[string]string{"contract": contract, "configuration": c.Spec.ID, "digest": c.Digest}
	binding := struct {
		Workspace, Actor, Contract, Configuration, Digest, Section, Field, RulesObservation string
		Rules                                                                               *financialProtectedRules
	}{workspace, actor, contract, c.Spec.ID, c.Digest, kind, input["field"], rulesObservation, rules}
	switch kind {
	case "protected_channels":
		channels, err := a.setupChannels(c, rules, rulesObservation)
		if err != nil {
			return nil, err
		}
		digest, err := financialEvidenceDigest("issuance_setup_protected_channel_page_v1", struct {
			Binding any
			Page    protectedSetupChannels
		}{binding, channels})
		if err != nil {
			return nil, commands.Fail(commands.DependencyUnavailable)
		}
		head["schema"], head["other_provenance"], head["unowned_module"] = "hb_issuance_protected_channel_page_v1", strconv.Itoa(channels.Other), strconv.Itoa(channels.Unowned)
		return a.setupProtectedPage(head, channels.IDs, channels.Rows, after, snapshot, digest)
	case "protected_changes":
		for _, f := range protectedSetupFields {
			if f.Field != input["field"] {
				continue
			}
			changes, err := a.setupChanges(c, rules, f.Field, protectedSetupSubject(c, f.Subject))
			if err != nil {
				return nil, err
			}
			digest, err := financialEvidenceDigest("issuance_setup_protected_change_page_v1", struct {
				Binding any
				Page    protectedSetupChanges
			}{binding, changes})
			if err != nil {
				return nil, commands.Fail(commands.DependencyUnavailable)
			}
			head["schema"], head["field"], head["other_provenance"], head["unowned_module"] = "hb_issuance_protected_change_page_v1", f.Field, strconv.Itoa(changes.Other), strconv.Itoa(changes.Unowned)
			return a.setupProtectedPage(head, changes.IDs, changes.Rows, after, snapshot, digest)
		}
		return nil, commands.Fail(commands.InvalidRequest)
	}
	head["schema"], head["current_eligibility"] = "hb_issuance_protected_setup_v1", "unchecked"
	head["rules_state"], head["rules_observation"] = "none", rulesObservation
	for _, name := range []string{"rules_id", "rules_pending_confirmer", "rules_digest", "rules_approvals", "rules_validity_seconds", "rules_max_lead_seconds", "rules_channel_age_seconds", "rules_proposer", "rules_confirmer", "rules_proposed_at", "rules_confirmed_at"} {
		head[name] = ""
	}
	if rules != nil {
		seconds := func(d time.Duration) string { return strconv.FormatInt(int64(d/time.Second), 10) }
		head["rules_state"], head["rules_id"], head["rules_digest"], head["rules_proposer"], head["rules_confirmer"] = rules.State, rules.ID, rules.Digest, rules.Proposer, rules.Module.Confirmer
		if rules.Module.Confirmer == "" {
			head["rules_pending_confirmer"] = rules.Confirmer
		}
		head["rules_approvals"], head["rules_validity_seconds"], head["rules_max_lead_seconds"], head["rules_channel_age_seconds"] = strconv.Itoa(rules.Rules.Approvals), seconds(rules.Rules.Validity), seconds(rules.Rules.MaxLead), seconds(rules.Rules.ChannelAge)
		head["rules_proposed_at"], head["rules_confirmed_at"] = setupInstant(rules.Module.Proposed), setupInstant(rules.Module.Confirmed)
	}
	channels, err := a.setupChannels(c, rules, rulesObservation)
	if err != nil {
		return nil, err
	}
	head["channels"], head["channels_other_provenance"], head["channels_unowned_module"] = strconv.Itoa(len(channels.Rows)), strconv.Itoa(channels.Other), strconv.Itoa(channels.Unowned)
	for _, f := range protectedSetupFields {
		changes, err := a.setupChanges(c, rules, f.Field, protectedSetupSubject(c, f.Subject))
		if err != nil {
			return nil, err
		}
		head[f.Prefix+"_subject_kind"], head[f.Prefix+"_changes"], head[f.Prefix+"_other_provenance"], head[f.Prefix+"_unowned_module"] = f.Subject, strconv.Itoa(len(changes.Rows)), strconv.Itoa(changes.Other), strconv.Itoa(changes.Unowned)
		head[f.Prefix+"_latest_scheduled"], head[f.Prefix+"_latest_scheduled_effective"] = changes.Scheduled, changes.Effective
	}
	if err := a.boundedDraftFields(head); err != nil {
		return nil, err
	}
	return head, nil
}
