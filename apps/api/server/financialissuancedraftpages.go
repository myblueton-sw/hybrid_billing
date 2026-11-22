package server

import (
	"fmt"
	"maps"
	"sort"
	"strconv"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func draftCursor(after, snapshot string) error {
	if (after == "") != (snapshot == "") || snapshot != "" && !validPlanHash(snapshot) {
		return commands.Fail(commands.InvalidRequest)
	}
	return nil
}

func (a *financialIssuance) inspectDraftFields(workspace, actor, id, after, snapshot string) (map[string]string, error) {
	if err := a.draftReadCaller(workspace, actor); err != nil {
		return nil, err
	}
	if err := draftCursor(after, snapshot); err != nil {
		return nil, err
	}
	start := 0
	if after != "" {
		n, err := strconv.Atoi(after)
		if err != nil || n < 0 || n >= 200 || strconv.Itoa(n) != after {
			return nil, commands.Fail(commands.InvalidRequest)
		}
		start = n + 1
	}
	p, err := a.inspectDraftRecord(workspace, actor, id)
	if err != nil {
		return nil, err
	}
	f, err := a.draftInspectionFields(workspace, actor, p)
	if err != nil {
		return nil, err
	}
	lines := p.Draft.Draft.Lines
	if len(lines) == 0 || len(lines) > 200 {
		return nil, commands.Fail(commands.EvidenceStale)
	}
	if after != "" && start >= len(lines) {
		return nil, commands.Fail(commands.EvidenceStale)
	}
	digest, err := financialEvidenceDigest("issuance_draft_inspection_page_v1", struct {
		Workspace, Actor string
		Observation      financialDraftObservation
		Fields           map[string]string
	}{workspace, actor, p, f})
	if err != nil {
		return nil, err
	}
	if snapshot != "" && snapshot != digest {
		return nil, commands.Fail(commands.EvidenceStale)
	}
	f["snapshot"], f["next_line"] = digest, ""
	end := min(start+10, len(lines))
	for i := start; i < end; i++ {
		line := lines[i]
		f[fmt.Sprintf("line_%02d", i-start)], err = draftJSON(struct {
			Index  string `json:"index"`
			Text   string `json:"text"`
			Amount string `json:"amount"`
			Tax    bool   `json:"tax"`
		}{strconv.Itoa(i), line.Text, line.Amount, line.Tax})
		if err != nil {
			return nil, err
		}
	}
	if end < len(lines) {
		f["next_line"] = strconv.Itoa(end - 1)
	}
	if err := a.boundedDraftFields(f); err != nil {
		return nil, err
	}
	return f, nil
}

func (a *financialIssuance) listDraftFields(workspace, actor, after, snapshot string) (map[string]string, error) {
	if err := a.draftReadCaller(workspace, actor); err != nil {
		return nil, err
	}
	if err := draftCursor(after, snapshot); err != nil {
		return nil, err
	}
	if after != "" && !validRequestID(after) {
		return nil, commands.Fail(commands.InvalidRequest)
	}
	if len(a.drafts) > financialIssuanceLimit {
		return nil, commands.Fail(commands.PrerequisiteFailed)
	}
	ids := make([]string, 0, len(a.drafts))
	for id := range a.drafts {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var observations []financialDraftObservation
	var rows []map[string]string
	for _, id := range ids {
		owned := a.drafts[id]
		p, err := a.retainedDraft(owned.Workspace, id)
		if err != nil {
			return nil, err
		}
		if owned.Workspace != workspace {
			continue
		}
		p = a.observeDraft(p)
		row, err := draftSummary(p)
		if err != nil {
			return nil, err
		}
		observations = append(observations, p)
		rows = append(rows, row)
	}
	digest, err := financialEvidenceDigest("issuance_draft_list_page_v1", struct {
		Workspace, Actor string
		Records          []financialDraftObservation
	}{workspace, actor, observations})
	if err != nil {
		return nil, err
	}
	if snapshot != "" && snapshot != digest {
		return nil, commands.Fail(commands.EvidenceStale)
	}
	start := 0
	if after != "" {
		n := sort.Search(len(rows), func(i int) bool { return rows[i]["draft"] >= after })
		if n == len(rows) || rows[n]["draft"] != after {
			return nil, commands.Fail(commands.EvidenceStale)
		}
		start = n + 1
	}
	fields := map[string]string{"mode": "synthetic", "current_eligibility": "unchecked", "snapshot": digest, "next": "", "count": strconv.Itoa(len(rows))}
	end := min(start+10, len(rows))
	for i := start; i < end; i++ {
		fields[fmt.Sprintf("row_%02d", i-start)], err = draftJSON(maps.Clone(rows[i]))
		if err != nil {
			return nil, err
		}
	}
	if end < len(rows) {
		fields["next"] = rows[end-1]["draft"]
	}
	if err := a.boundedDraftFields(fields); err != nil {
		return nil, err
	}
	return fields, nil
}
