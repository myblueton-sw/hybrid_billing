package server

import (
	"fmt"
	"maps"
	"reflect"
	"sort"
	"strconv"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/issuance"
)

type financialDraftContext struct {
	Evidence   financialDraftContextEvidence
	Choices    []map[string]string
	Drift      bool
	SetupDate  string
	References []issuance.Reference
}

func (a *financialIssuance) draftContext(workspace, actor, request, digest string) (financialDraftContext, error) {
	evidence, err := a.draftContextEvidence(workspace, actor, request, digest)
	if err != nil {
		return financialDraftContext{}, err
	}
	p := financialDraftContext{Evidence: evidence}
	f := p.Evidence.Fields
	f["submit_permission"] = a.draftRecoveryPermission(workspace, actor, actor, draftauthority.Submit)
	f["approve_permission"] = a.draftRecoveryPermission(workspace, actor, actor, draftauthority.Approve)
	f["step_up"] = "satisfied"
	for _, command := range []string{"submit", "prepare_approval", "approve"} {
		code := a.server.stepUpCheck(a.server.caller, actor, "issuance_draft."+command, "")
		if code == commands.PrerequisiteFailed {
			f["step_up"] = "authentication_policy_unsatisfied"
		} else if code != "" && f["step_up"] == "satisfied" {
			f["step_up"] = "reauthentication_required"
		}
	}
	if a.server.held() {
		f["readiness"] = "installation_held"
	} else {
		a.currentDraftContext(workspace, actor, request, digest, &p)
	}
	a.draftContextActions(workspace, actor, &p)
	return p, nil
}

func (a *financialIssuance) currentDraftContext(workspace, actor, request, digest string, p *financialDraftContext) {
	f := p.Evidence.Fields
	f["readiness"] = "unavailable"
	if existing := p.Evidence.Existing; existing != nil && (existing.Module == "conflict" || existing.Module == "unavailable") {
		f["current_eligibility"] = "unavailable"
		return
	}
	source, err := a.server.financialIssuanceSourceLocked(workspace, request, digest, actor)
	if err != nil {
		f["current_eligibility"] = draftContextSourceStatus(err)
		if f["current_eligibility"] == "ineligible" {
			f["readiness"] = "source_ineligible"
		}
		return
	}
	f["current_eligibility"] = "eligible"
	c, readiness, err := a.contextConfiguration(source, p.Evidence.Existing)
	f["readiness"] = readiness
	if err != nil || c == nil {
		return
	}
	if err := draftContextConstructible(*c, source); err != nil {
		f["readiness"], f["current_eligibility"] = "source_ineligible", "ineligible"
		return
	}
	languages, err := draftJSON(c.Spec.Languages)
	if err != nil {
		f["readiness"] = "unavailable"
		return
	}
	f["configuration"], f["configuration_digest"], f["languages"] = c.Spec.ID, c.Digest, languages
	if a.fields == nil || a.terms == nil {
		f["readiness"] = "unavailable"
		return
	}
	at := a.server.clock().UTC().Round(0)
	refs, err := a.configuredReferences(*c, at)
	if err != nil {
		f["readiness"] = "unavailable"
		if draftContextMissingSetup(err) {
			f["readiness"] = "terms_missing"
		}
		return
	}
	choices, err := a.contextBindingChoices(*c, at)
	if err != nil {
		f["readiness"] = "unavailable"
		if draftContextMissingSetup(err) {
			f["readiness"] = "terms_missing"
		}
		return
	}
	if len(choices) == 0 {
		f["readiness"] = "terms_missing"
		return
	}
	location, err := time.LoadLocation(c.Spec.Zone)
	if err != nil {
		f["readiness"] = "unavailable"
		return
	}
	p.SetupDate = at.In(location).Format(time.DateOnly)
	p.References = refs
	p.Choices = choices
	if existing := p.Evidence.Existing; existing != nil {
		f["readiness"] = "existing_draft"
		r := existing.Review
		if existing.HasReview && (r.State == "prepared" || r.State == "approval_unknown") {
			coverage, err := a.configuredCoverage(*c, r.BindingID, r.BindingDigest, at)
			if err != nil {
				f["readiness"], p.Choices = "unavailable", nil
				if draftContextMissingSetup(err) {
					f["readiness"] = "terms_missing"
				}
				return
			}
			p.Drift = !reflect.DeepEqual(refs, r.References) || coverage != r.Coverage
		}
	}
}

func (a *financialIssuance) contextDraftFields(workspace, actor, request, digest, after, snapshot string) (map[string]string, error) {
	if err := a.draftReadCaller(workspace, actor); err != nil {
		return nil, err
	}
	if err := draftCursor(after, snapshot); err != nil {
		return nil, err
	}
	if after != "" && !validRequestID(after) {
		return nil, commands.Fail(commands.InvalidRequest)
	}
	p, err := a.draftContext(workspace, actor, request, digest)
	if err != nil {
		return nil, err
	}
	return a.draftContextPage(workspace, actor, p, after, snapshot)
}

func (a *financialIssuance) draftContextPage(workspace, actor string, p financialDraftContext, after, snapshot string) (map[string]string, error) {
	bound, err := financialEvidenceDigest("issuance_draft_context_page_v1", struct {
		Workspace, Actor string
		Context          financialDraftContext
	}{workspace, actor, p})
	if err != nil {
		return nil, err
	}
	if snapshot != "" && snapshot != bound {
		return nil, commands.Fail(commands.EvidenceStale)
	}
	start := 0
	if after != "" {
		n := sort.Search(len(p.Choices), func(i int) bool { return p.Choices[i]["binding"] >= after })
		if n == len(p.Choices) || p.Choices[n]["binding"] != after {
			return nil, commands.Fail(commands.EvidenceStale)
		}
		start = n + 1
	}
	f := maps.Clone(p.Evidence.Fields)
	f["snapshot"], f["next_binding"], f["binding_count"] = bound, "", strconv.Itoa(len(p.Choices))
	end := min(start+10, len(p.Choices))
	for i := start; i < end; i++ {
		f[fmt.Sprintf("binding_%02d", i-start)], err = draftJSON(p.Choices[i])
		if err != nil {
			return nil, err
		}
	}
	if end < len(p.Choices) {
		f["next_binding"] = p.Choices[end-1]["binding"]
	}
	if err := a.boundedDraftFields(f); err != nil {
		return nil, err
	}
	return f, nil
}
