package server

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
	"github.com/myblueton-sw/hybrid_billing/modules/protectedauthority"
)

const (
	financialAttestationLimit   = 1000
	financialAttestationPending = 10
)

type financialAttestationUse struct{ Purpose, Target, TargetDigest string }

type financialAttestation struct {
	ID, Workspace, Purpose, Target, TargetDigest, Actor, Statement, Locator, Digest string
	Context                                                                         financialConfigurationReference
	RecordedAt, WithdrawnAt                                                         time.Time
	Consumed                                                                        *financialAttestationUse
}

type setupAttestationStep struct {
	Purpose, Capability string
	Action              gate.Action
	Existing            bool
}

var setupAttestationSteps = []setupAttestationStep{
	{"channel_registration", protectedauthority.RegisterChannel, gate.RegisterIssuanceChannel, false},
	{"channel_admission", protectedauthority.AdmitChannel, gate.AdmitIssuanceChannel, true},
	{"change_instruction", protectedauthority.RequestChange, gate.RequestIssuanceProtectedChange, false},
	{"change_verification", protectedauthority.VerifyChange, gate.VerifyIssuanceProtectedChange, true},
}

func setupAttestationCommand(kind string) (setupAttestationStep, bool, bool) {
	for _, step := range setupAttestationSteps {
		if kind == "attest_"+step.Purpose {
			return step, false, true
		}
		if kind == "withdraw_"+step.Purpose {
			return step, true, true
		}
	}
	return setupAttestationStep{}, false, false
}

func (a *financialIssuance) attestationFor(workspace, id, digest, purpose, target, targetDigest, actor string, use financialAttestationUse) error {
	p, ok := a.attestations[id]
	actual, err := attestationDigest(p)
	if !ok || err != nil || actual != p.Digest || p.Digest != digest || p.ID != id || p.Workspace != workspace || p.Purpose != purpose || p.Target != target || p.TargetDigest != targetDigest || p.Actor != actor || !p.WithdrawnAt.IsZero() || p.Consumed != nil && *p.Consumed != use {
		return commands.Fail(commands.EvidenceStale)
	}
	return nil
}

func (a *financialIssuance) consumeAttestation(id string, use financialAttestationUse) {
	p := a.attestations[id]
	p.Consumed = &use
	a.attestations[id] = p
}

func (a *financialIssuance) heldAttestation(workspace, id, digest string, use financialAttestationUse) error {
	p, ok := a.attestations[id]
	actual, err := attestationDigest(p)
	if !ok || err != nil || actual != p.Digest || p.Digest != digest || p.Workspace != workspace || p.Purpose != use.Purpose || p.Target != use.Target || p.Consumed == nil || *p.Consumed != use || !p.WithdrawnAt.IsZero() {
		return commands.Fail(commands.EvidenceStale)
	}
	return nil
}

func attestationDigest(p financialAttestation) (string, error) {
	p.Digest, p.WithdrawnAt, p.Consumed = "", time.Time{}, nil
	return financialEvidenceDigest("issuance_setup_attestation_v1", p)
}

func attestationState(p financialAttestation) string {
	if p.Consumed != nil {
		return "consumed"
	}
	if !p.WithdrawnAt.IsZero() {
		return "withdrawn"
	}
	return "unconsumed"
}

func attestationText(text string, limit int, required bool) bool {
	if !utf8.ValidString(text) || len(text) > limit || required && strings.TrimSpace(text) == "" {
		return false
	}
	for _, r := range text {
		if r == utf8.RuneError || unicode.IsControl(r) && r != '\t' && r != '\n' {
			return false
		}
	}
	return true
}

func attestationRow(p financialAttestation) map[string]string {
	return map[string]string{
		"attestation": p.ID, "digest": p.Digest, "purpose": p.Purpose, "target": p.Target, "actor": p.Actor, "recorded_at": setupInstant(p.RecordedAt),
		"statement": p.Statement, "locator": p.Locator, "state": attestationState(p), "assurance": "human_attestation",
	}
}

func attestationAdmitted(p financialAttestation) bool {
	row := attestationRow(p)
	row["actor"], row["digest"], row["recorded_at"], row["state"] = strings.Repeat("x", 128), strings.Repeat("0", 64), strings.Repeat("9", 35), "unconsumed"
	raw, err := json.Marshal(row)
	if err != nil || len(raw) > 4096 {
		return false
	}
	enveloped, err := json.Marshal(string(raw))
	return err == nil && len(enveloped) <= 6000
}

func attestationReceipt(contract string, configuration financialConfigurationReference, p financialAttestation) map[string]string {
	return map[string]string{
		"schema": "hb_issuance_setup_attestation_v1", "contract": contract, "configuration": configuration.ID, "attestation": p.ID, "purpose": p.Purpose,
		"target": p.Target, "actor": p.Actor, "recorded_at": setupInstant(p.RecordedAt), "digest": p.Digest, "state": attestationState(p), "assurance": "human_attestation",
	}
}

func (a *financialIssuance) attestationTarget(step setupAttestationStep, workspace, actor, target, digest string) error {
	stale := commands.Fail(commands.EvidenceStale)
	switch step.Purpose {
	case "channel_registration":
		if _, exists := a.channels[target]; exists {
			return stale
		}
	case "change_instruction":
		if _, exists := a.changes[target]; exists {
			return stale
		}
	case "channel_admission":
		p, ok := a.channels[target]
		if !ok || p.Workspace != workspace || p.Spec.ID != target || p.Digest != digest || p.State != "registered" && p.State != "admission_unknown" {
			return stale
		}
		if p.Registrar == actor {
			return commands.Fail(commands.SelfApprovalDenied)
		}
	case "change_verification":
		p, ok := a.changes[target]
		if !ok || p.Workspace != workspace || p.Spec.ID != target || p.Digest != digest || p.Phase != "recorded" || p.Module.State != protected.StateRequested {
			return stale
		}
		if p.Requester == actor {
			return commands.Fail(commands.SelfApprovalDenied)
		}
	}
	return nil
}

func (a *financialIssuance) applySetupAttestation(kind string, ctx commands.Context) (commands.Outcome, error) {
	step, withdraw, ok := setupAttestationCommand(kind)
	if !ok {
		return commands.Outcome{}, commands.Fail(commands.UnsupportedCapability)
	}
	workspace, actor, contract := ctx.Workspace, ctx.User, ctx.Target
	if err := a.protectedCaller(workspace, actor, step.Capability); err != nil {
		return commands.Outcome{}, err
	}
	f := ctx.Fields
	id, target := f["attestation"], f["target"]
	shape := validRequestID(contract) && validRequestID(f["configuration"]) && validPlanHash(f["digest"]) && validRequestID(id) && len(id) <= 128
	if !withdraw {
		shape = shape && validRequestID(target) && len(target) <= 128 && step.Existing == validPlanHash(f["target_digest"]) && (step.Existing || f["target_digest"] == "") && attestationText(f["statement"], 2000, true) && attestationText(f["locator"], 512, false)
	}
	draft := financialAttestation{ID: id, Workspace: workspace, Purpose: step.Purpose, Target: target, TargetDigest: f["target_digest"], Actor: actor, Statement: f["statement"], Locator: f["locator"]}
	if !shape || !withdraw && !attestationAdmitted(draft) {
		return commands.Outcome{}, commands.Fail(commands.InvalidRequest)
	}
	c, err := a.confirmedSetupConfiguration(workspace, contract, f["configuration"], f["digest"])
	if err != nil {
		return commands.Outcome{}, err
	}
	reference := configurationReference(c)
	finish := func(p financialAttestation, changed bool) (commands.Outcome, error) {
		fields := attestationReceipt(contract, reference, p)
		if err := a.boundedDraftFields(fields); err != nil {
			if changed {
				return commands.Outcome{}, errConfigurationUncertain
			}
			return commands.Outcome{}, err
		}
		state := "recorded"
		if withdraw {
			state = "withdrawn"
		}
		return commands.Outcome{Fields: fields, BusinessState: state, ResultRevision: a.server.Next()}, nil
	}
	old, exists := a.attestations[id]
	if withdraw {
		if !exists || old.Workspace != workspace || old.Actor != actor || old.Purpose != step.Purpose {
			return commands.Outcome{}, commands.Fail(commands.PermissionDenied)
		}
		if old.Consumed != nil {
			return commands.Outcome{}, commands.Fail(commands.PrerequisiteFailed)
		}
		if !old.WithdrawnAt.IsZero() {
			return finish(old, false)
		}
		if a.server.faulty("issuance-attestation-before-withdrawal") {
			return commands.Outcome{}, commands.Fail(commands.DependencyUnavailable)
		}
		old.WithdrawnAt = a.server.clock().UTC().Round(0)
		a.attestations[id] = old
		if a.server.faulty("issuance-attestation-after-withdrawal") {
			return commands.Outcome{}, errConfigurationUncertain
		}
		return finish(old, true)
	}
	p := draft
	p.Context = reference
	if exists {
		same := old
		same.RecordedAt, same.WithdrawnAt, same.Consumed, same.Digest = time.Time{}, time.Time{}, nil, ""
		if !reflect.DeepEqual(same, p) {
			return commands.Outcome{}, commands.Fail(commands.EvidenceStale)
		}
		return finish(old, false)
	}
	if err := a.attestationTarget(step, workspace, actor, target, p.TargetDigest); err != nil {
		return commands.Outcome{}, err
	}
	pending := 0
	for _, other := range a.attestations {
		if other.Actor == actor && attestationState(other) == "unconsumed" {
			pending++
		}
	}
	if len(a.attestations) >= financialAttestationLimit || pending >= financialAttestationPending {
		return commands.Outcome{}, commands.Fail(commands.PrerequisiteFailed)
	}
	if a.server.faulty("issuance-attestation-before-retention") {
		return commands.Outcome{}, commands.Fail(commands.DependencyUnavailable)
	}
	p.RecordedAt = a.server.clock().UTC().Round(0)
	if p.Digest, err = attestationDigest(p); err != nil {
		return commands.Outcome{}, commands.Fail(commands.DependencyUnavailable)
	}
	a.attestations[id] = p
	if a.server.faulty("issuance-attestation-after-retention") {
		return commands.Outcome{}, errConfigurationUncertain
	}
	return finish(p, true)
}

func (a *financialIssuance) attestationInScope(c financialIssuanceConfiguration, p financialAttestation) (bool, error) {
	if p.Consumed == nil {
		return p.Context == configurationReference(c), nil
	}
	use := *p.Consumed
	if use.Purpose != p.Purpose || use.Target != p.Target {
		return false, commands.Fail(commands.EvidenceStale)
	}
	if strings.HasPrefix(p.Purpose, "channel_") {
		channel, ok := a.channels[p.Target]
		if !ok || channel.Workspace != p.Workspace || channel.Digest != use.TargetDigest {
			return false, commands.Fail(commands.EvidenceStale)
		}
		return protectedSetupChannelKind(c, channel) != "", nil
	}
	change, ok := a.changes[p.Target]
	if !ok || change.Workspace != p.Workspace || change.Digest != use.TargetDigest {
		return false, commands.Fail(commands.EvidenceStale)
	}
	return change.Configuration != nil && reflect.DeepEqual(*change.Configuration, c), nil
}

func (a *financialIssuance) setupAttestationFields(workspace, actor, contract string, input map[string]string) (map[string]string, error) {
	c, err := a.protectedSetupConfiguration(workspace, actor, contract, "attestations", input)
	if err != nil {
		return nil, err
	}
	if len(a.attestations) > financialAttestationLimit {
		return nil, commands.Fail(commands.PrerequisiteFailed)
	}
	ids := make([]string, 0, len(a.attestations))
	for id := range a.attestations {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	page := struct {
		Workspace, Actor, Contract string
		Configuration              financialConfigurationReference
		IDs, Rows                  []string
		Records                    []financialAttestation
	}{Workspace: workspace, Actor: actor, Contract: contract, Configuration: configurationReference(c)}
	for _, id := range ids {
		p := a.attestations[id]
		digest, err := attestationDigest(p)
		if err != nil || digest != p.Digest || p.ID != id || p.Workspace != workspace || !validRequestID(p.Actor) || p.RecordedAt.IsZero() || p.Consumed != nil && !p.WithdrawnAt.IsZero() {
			return nil, commands.Fail(commands.EvidenceStale)
		}
		if _, _, known := setupAttestationCommand("attest_" + p.Purpose); !known {
			return nil, commands.Fail(commands.EvidenceStale)
		}
		scoped, err := a.attestationInScope(c, p)
		if err != nil {
			return nil, err
		}
		if !scoped {
			continue
		}
		raw, err := draftJSON(attestationRow(p))
		if err != nil {
			return nil, err
		}
		if len(raw) > 4096 {
			return nil, commands.Fail(commands.PrerequisiteFailed)
		}
		page.IDs, page.Rows, page.Records = append(page.IDs, id), append(page.Rows, raw), append(page.Records, p)
	}
	digest, err := financialEvidenceDigest("issuance_setup_attestation_page_v1", page)
	if err != nil {
		return nil, commands.Fail(commands.DependencyUnavailable)
	}
	head := map[string]string{"schema": "hb_issuance_setup_attestation_page_v1", "contract": contract, "configuration": c.Spec.ID, "digest": c.Digest}
	return a.setupProtectedPage(head, page.IDs, page.Rows, input["after"], input["snapshot"], digest)
}
