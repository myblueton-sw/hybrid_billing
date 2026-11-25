package server

import (
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func (a *financialERP) resultHistoryLocked(ctx commands.Context) (map[string]string, error) {
	if err := a.resultPermission(ctx.Workspace, ctx.User, gate.ViewCost); err != nil {
		return nil, err
	}
	owner, ok := a.owners[[2]string{ctx.Workspace, ctx.Target}]
	if !ok || !validPlanHash(owner.receipt) || owner.receipt != ctx.Fields["digest"] || owner.submission.Key != ctx.Target {
		return nil, commands.Fail(commands.EvidenceStale)
	}
	if len(a.results) > financialDocumentLimit {
		return nil, commands.Fail(commands.PrerequisiteFailed)
	}
	var rows []documentHistoryRow
	for key, record := range a.results {
		if key[0] != ctx.Workspace {
			continue
		}
		r, b, err := a.resultRecord(key[0], key[1], record.review.Digest)
		if err != nil {
			return nil, err
		}
		if b.Packet.Preparation != ctx.Target {
			continue
		}
		if b.Packet.PreparationDigest != owner.receipt || !matchingPreparation(owner, b.Preparation) {
			return nil, commands.Fail(commands.EvidenceStale)
		}
		if _, _, err = a.resultApplicationOwner(b, r.review.Digest); err != nil {
			return nil, err
		}
		v := a.resultView(r, b)
		fields := resultReceipt(v)
		fields["proposed_by"], fields["proposed_at"] = r.review.Proposer, r.review.At.Format(time.RFC3339Nano)
		rows = append(rows, documentHistoryRow{id: b.ID, value: fields})
	}
	fields, err := documentPage(rows, map[string]string{"mode": "synthetic", "preparation": ctx.Target, "digest": owner.receipt, "current_eligibility": "unchecked"}, ctx.Fields["after"], ctx.Fields["snapshot"], ctx.Resolved)
	if err != nil {
		return nil, err
	}
	if err := a.boundedResultFields(fields); err != nil {
		return nil, err
	}
	return fields, nil
}
