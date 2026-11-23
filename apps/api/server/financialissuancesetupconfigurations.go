package server

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func configurationHistoryRow(p financialIssuanceConfiguration, observation string) map[string]string {
	row := map[string]string{
		"configuration": p.Spec.ID, "digest": p.Digest, "state": p.State, "module_observation": observation,
		"revision": strconv.FormatUint(p.Spec.Revision, 10), "contract_revision": strconv.FormatUint(p.Spec.ContractRevision, 10),
		"billing_profile": p.Spec.BillingProfile, "billing_revision": strconv.FormatUint(p.Spec.BillingRevision, 10),
		"mode": p.Spec.Mode, "profile": p.Spec.Profile, "region": p.Spec.Region, "tax": p.Spec.Tax, "zone": p.Spec.Zone,
		"languages": strings.Join(p.Spec.Languages, ","), "proposer": p.Proposer, "confirmer": p.Confirmer,
	}
	if !p.Proposed.At.IsZero() {
		row["proposed_at"] = p.Proposed.At.Format(time.RFC3339Nano)
	}
	if !p.Active.At.IsZero() {
		row["approval_observed_at"] = p.Active.At.Format(time.RFC3339Nano)
	}
	if !p.ActivatedAt.IsZero() {
		row["activated_at"] = p.ActivatedAt.Format(time.RFC3339Nano)
	}
	return row
}

func (a *financialIssuance) setupConfigurationFields(workspace, actor, contract, after, snapshot string) (map[string]string, error) {
	if err := a.protectedReadCaller(workspace, actor); err != nil {
		return nil, err
	}
	if !validRequestID(contract) || after != "" && !validRequestID(after) {
		return nil, commands.Fail(commands.InvalidRequest)
	}
	if err := draftCursor(after, snapshot); err != nil {
		return nil, err
	}
	if err := a.configurationHistoryIndex(); err != nil {
		return nil, err
	}
	ids := []string{}
	for id, p := range a.configurations {
		if p.Workspace == workspace && p.Spec.Contract == contract {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	rows := make([]string, 0, len(ids))
	records := make([]financialIssuanceConfiguration, 0, len(ids))
	for _, id := range ids {
		p := a.configurations[id]
		observation, err := a.observeConfigurationHistory(p)
		if err != nil {
			return nil, err
		}
		raw, err := draftJSON(configurationHistoryRow(p, observation))
		if err != nil {
			return nil, err
		}
		if len(raw) > 4096 {
			return nil, commands.Fail(commands.PrerequisiteFailed)
		}
		rows = append(rows, raw)
		records = append(records, p)
	}
	digest, err := financialEvidenceDigest("issuance_setup_configuration_page_v1", struct {
		Workspace, Actor, Contract string
		Records                    []financialIssuanceConfiguration
		Rows                       []string
	}{workspace, actor, contract, records, rows})
	if err != nil {
		return nil, commands.Fail(commands.DependencyUnavailable)
	}
	if snapshot != "" && snapshot != digest {
		return nil, commands.Fail(commands.EvidenceStale)
	}
	start := 0
	if after != "" {
		index := sort.SearchStrings(ids, after)
		if index == len(ids) || ids[index] != after {
			return nil, commands.Fail(commands.EvidenceStale)
		}
		start = index + 1
	}
	fields := map[string]string{"schema": "hb_issuance_configuration_page_v1", "contract": contract, "current_eligibility": "unchecked", "snapshot": digest, "count": strconv.Itoa(len(rows)), "next": ""}
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
