package server

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

type documentHistoryRow struct {
	id    string
	value map[string]string
}

func documentPage(rows []documentHistoryRow, fields map[string]string, after, snapshot, revision string) (map[string]string, error) {
	if (after == "") != (snapshot == "") || after != "" && !validRequestID(after) {
		return nil, commands.Fail(commands.InvalidRequest)
	}
	if snapshot != "" && snapshot != revision {
		return nil, commands.Fail(commands.EvidenceStale)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].id < rows[j].id })
	start := 0
	if after != "" {
		start = sort.Search(len(rows), func(i int) bool { return rows[i].id >= after })
		if start == len(rows) || rows[start].id != after {
			return nil, commands.Fail(commands.EvidenceStale)
		}
		start++
	}
	fields["snapshot"], fields["next"] = revision, ""
	end := min(start+10, len(rows))
	for i := start; i < end; i++ {
		raw, err := json.Marshal(rows[i].value)
		if err != nil {
			return nil, commands.Fail(commands.DependencyUnavailable)
		}
		fields[fmt.Sprintf("row_%02d", i-start)] = string(raw)
	}
	if end < len(rows) {
		fields["next"] = rows[end-1].id
	}
	return fields, nil
}

func (a *financialERP) documentHistoryLocked(ctx commands.Context) (map[string]string, error) {
	if err := a.documentPermission(ctx.Workspace, ctx.User, gate.ViewCost); err != nil {
		return nil, err
	}
	kind := ctx.Fields["kind"]
	if kind != "documents" && kind != "preparations" {
		return nil, commands.Fail(commands.InvalidRequest)
	}
	request, _, err := a.server.financialChargeRecord(ctx.Workspace, ctx.Target)
	if err != nil || request.Digest != ctx.Fields["digest"] {
		return nil, commands.Fail(commands.EvidenceStale)
	}
	if len(a.documents) > financialDocumentLimit || len(a.owners) > financialDocumentLimit {
		return nil, commands.Fail(commands.PrerequisiteFailed)
	}
	byDigest := map[string]financialDocumentBasis{}
	var rows []documentHistoryRow
	for key, record := range a.documents {
		if key[0] != ctx.Workspace {
			continue
		}
		checked, b, err := a.governedDocument(ctx.Workspace, key[1], record.review.Digest)
		if err != nil {
			return nil, err
		}
		if _, duplicate := byDigest[checked.review.Digest]; duplicate {
			return nil, commands.Fail(commands.EvidenceStale)
		}
		byDigest[checked.review.Digest] = b
		if b.Request != ctx.Target {
			continue
		}
		if b.RequestDigest != request.Digest {
			return nil, commands.Fail(commands.EvidenceStale)
		}
		if kind == "documents" {
			rows = append(rows, documentHistoryRow{id: b.ID, value: map[string]string{"document": b.ID, "digest": checked.review.Digest, "recorded_state": checked.review.State, "proposed_by": checked.review.Proposer, "proposed_at": checked.review.At.Format(time.RFC3339Nano)}})
		}
	}
	for key, owner := range a.owners {
		if key[0] != ctx.Workspace {
			continue
		}
		b, ok := byDigest[owner.basis]
		if !ok || !validRequestID(key[1]) || owner.submission.Key != key[1] || !validRequestID(owner.actor) || owner.receipt != "" && len(owner.receipt) != 64 {
			return nil, commands.Fail(commands.EvidenceStale)
		}
		associated, _, err := a.server.financialChargeRecord(ctx.Workspace, b.Request)
		if err != nil || associated.Digest != b.RequestDigest {
			return nil, commands.Fail(commands.EvidenceStale)
		}
		if b.Request != ctx.Target {
			continue
		}
		if kind == "preparations" {
			rows = append(rows, documentHistoryRow{id: key[1], value: map[string]string{"preparation": key[1], "document": b.ID, "document_digest": owner.basis, "actor": owner.actor, "recorded_receipt": owner.receipt}})
		}
	}
	fields, err := documentPage(rows, map[string]string{"mode": "synthetic", "request": ctx.Target, "request_digest": request.Digest, "kind": kind, "current_eligibility": "unchecked"}, ctx.Fields["after"], ctx.Fields["snapshot"], ctx.Resolved)
	if err != nil {
		return nil, err
	}
	if err = a.boundedDocumentFields(fields); err != nil {
		return nil, err
	}
	return fields, nil
}
