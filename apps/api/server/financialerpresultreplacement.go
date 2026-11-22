package server

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/erpledger"
)

func cloneResultSnapshot(v erpledger.ResultView) *erpledger.ResultView {
	v.Submission.Submission.Lines = slices.Clone(v.Submission.Submission.Lines)
	v.Submission.Mappings = slices.Clone(v.Submission.Mappings)
	v.Submission.Commands = slices.Clone(v.Submission.Commands)
	if v.Submission.Resolution != nil {
		c := *v.Submission.Resolution
		v.Submission.Resolution = &c
	}
	if v.Submission.Watch != nil {
		c := *v.Submission.Watch
		v.Submission.Watch = &c
	}
	v.Reconciliation.Lines = slices.Clone(v.Reconciliation.Lines)
	v.Reconciliation.Documents = slices.Clone(v.Reconciliation.Documents)
	v.Documents = slices.Clone(v.Documents)
	for i := range v.Documents {
		v.Documents[i].Document.Rows = slices.Clone(v.Documents[i].Document.Rows)
		v.Documents[i].Links = slices.Clone(v.Documents[i].Links)
	}
	return &v
}

func applicationIntentMatches(app financialERPResultApplication, b financialERPResultBasis) bool {
	plain := app
	plain.receipt, plain.predecessor = "", ""
	plain.before, plain.after, plain.reservedAt = nil, nil, time.Time{}
	return validRequestID(app.actor) && reflect.DeepEqual(plain, resultIntent(b, app.actor, app.digest))
}

func replacementSnapshotMatches(v erpledger.ResultView, b financialERPResultBasis, app financialERPResultApplication, observed time.Time) bool {
	if app.before == nil || len(app.before.Documents) != 1 || len(v.Documents) != 1 || app.reservedAt.IsZero() {
		return false
	}
	old, next := app.before.Documents[0], v.Documents[0]
	if next.At.IsZero() || next.At.Before(app.reservedAt) || next.At.Before(old.At) || next.At.After(observed) || len(next.Links) != len(old.Links)+len(app.links) {
		return false
	}
	for i, link := range old.Links {
		link.Stale = true
		if next.Links[i] != link {
			return false
		}
	}
	for i, link := range app.links {
		if next.Links[len(old.Links)+i] != (erpledger.LinkView{Link: link, Author: app.actor}) {
			return false
		}
	}
	trimmed := *cloneResultSnapshot(v)
	trimmed.Documents[0].Links = trimmed.Documents[0].Links[len(old.Links):]
	return reflect.DeepEqual(v.Submission, app.before.Submission) && resultSnapshotMatches(trimmed, b, app, true)
}

func retainedResultValid(b financialERPResultBasis, app financialERPResultApplication) bool {
	if app.receipt == "" {
		return app.after == nil
	}
	if !validPlanHash(app.receipt) || app.after == nil {
		return false
	}
	if app.predecessor == "" {
		if !resultSnapshotMatches(*app.after, b, app, true) {
			return false
		}
	} else if len(app.after.Documents) != 1 || !replacementSnapshotMatches(*app.after, b, app, app.after.Documents[0].At) {
		return false
	}
	digest, err := resultApplicationDigest(app.digest, app.actor, *app.after)
	return err == nil && digest == app.receipt
}

func resultRetainedSize(app financialERPResultApplication) (int, error) {
	raw, err := json.Marshal(struct {
		Actor, Digest, Receipt, Predecessor string
		Document                            erpledger.Document
		Links                               []erpledger.Link
		Before, After                       *erpledger.ResultView
		ReservedAt                          time.Time
	}{app.actor, app.digest, app.receipt, app.predecessor, app.document, app.links, app.before, app.after, app.reservedAt})
	if err != nil || len(raw) > financialDocumentBytes {
		return 0, commands.Fail(commands.PrerequisiteFailed)
	}
	return len(raw), nil
}

func (a *financialERP) validateResultChains() error {
	fail := func() error { return commands.Fail(commands.EvidenceStale) }
	if len(a.resultApplications) > financialDocumentLimit || len(a.resultExternals) != len(a.resultPreparations) || len(a.resultExternals) > len(a.resultApplications) {
		return fail()
	}
	bases := map[[2]string]financialERPResultBasis{}
	for key, app := range a.resultApplications {
		_, b, err := a.resultRecord(key[0], key[1], app.digest)
		if err != nil || !applicationIntentMatches(app, b) || !retainedResultValid(b, app) {
			return fail()
		}
		if _, err := resultRetainedSize(app); err != nil {
			return fail()
		}
		if app.predecessor == "" && (app.before != nil || !app.reservedAt.IsZero()) {
			return fail()
		}
		bases[key] = b
	}
	visited := map[[2]string]bool{}
	preparations := map[[2]string]bool{}
	for external, id := range a.resultExternals {
		key := [2]string{external[0], id}
		head, exists := bases[key]
		if !exists || external != ([3]string{head.Workspace, head.Packet.Connection, head.Packet.External}) {
			return fail()
		}
		preparation := [2]string{head.Workspace, head.Packet.Preparation}
		if preparations[preparation] || a.resultPreparations[preparation] != id {
			return fail()
		}
		preparations[preparation] = true
		for {
			if visited[key] {
				return fail()
			}
			visited[key] = true
			app := a.resultApplications[key]
			b := bases[key]
			if b.Workspace != head.Workspace || b.Packet.Connection != head.Packet.Connection || b.Packet.External != head.Packet.External || b.Packet.Preparation != head.Packet.Preparation {
				return fail()
			}
			if app.predecessor == "" {
				break
			}
			parentKey := [2]string{key[0], app.predecessor}
			parent, ok := a.resultApplications[parentKey]
			previous, found := bases[parentKey]
			if !ok || !found || parent.receipt == "" || parent.after == nil || app.before == nil || !reflect.DeepEqual(*app.before, *parent.after) || previous.Packet.Revision >= b.Packet.Revision || app.reservedAt.IsZero() || len(app.before.Documents) != 1 || app.reservedAt.Before(app.before.Documents[0].At) {
				return fail()
			}
			key = parentKey
		}
	}
	if len(visited) != len(a.resultApplications) || len(preparations) != len(a.resultPreparations) {
		return fail()
	}
	return nil
}

func (a *financialERP) replacementIntent(b financialERPResultBasis, actor, digest string) (financialERPResultApplication, error) {
	fail := func(err error) (financialERPResultApplication, error) { return financialERPResultApplication{}, err }
	if err := a.validateResultChains(); err != nil {
		return fail(err)
	}
	external, preparation := [3]string{b.Workspace, b.Packet.Connection, b.Packet.External}, [2]string{b.Workspace, b.Packet.Preparation}
	predecessor := a.resultExternals[external]
	if predecessor == "" || a.resultPreparations[preparation] != predecessor || predecessor == b.ID {
		return fail(commands.Fail(commands.EvidenceStale))
	}
	parent, ok := a.resultApplications[[2]string{b.Workspace, predecessor}]
	if !ok || parent.receipt == "" || parent.after == nil || parent.document.Revision >= b.Packet.Revision {
		return fail(commands.Fail(commands.EvidenceStale))
	}
	if len(a.resultApplications) >= financialDocumentLimit {
		return fail(commands.Fail(commands.PrerequisiteFailed))
	}
	actual, err := a.ledger.ResultOf(b.Workspace, b.Packet.Preparation)
	if err != nil {
		return fail(commands.Fail(commands.DependencyUnavailable))
	}
	if !reflect.DeepEqual(actual, *parent.after) {
		return fail(commands.Fail(commands.EvidenceStale))
	}
	at, err := a.resultNow()
	if err != nil {
		return fail(err)
	}
	if len(actual.Documents) != 1 || at.Before(actual.Documents[0].At) {
		return fail(commands.Fail(commands.EvidenceStale))
	}
	app := resultIntent(b, actor, digest)
	app.predecessor, app.before, app.reservedAt = predecessor, cloneResultSnapshot(actual), at
	if err := resultApplicationAdmission(b, app); err != nil {
		return fail(err)
	}
	prospective := app
	prospective.after = cloneResultSnapshot(actual)
	last := &prospective.after.Documents[0]
	last.Document = app.document
	last.Author = app.actor
	last.At = time.Date(9999, 12, 31, 23, 59, 59, 999999999, time.UTC)
	for i := range last.Links {
		last.Links[i].Stale = true
	}
	for _, link := range app.links {
		last.Links = append(last.Links, erpledger.LinkView{Link: link, Author: app.actor})
	}
	prospective.receipt = strings.Repeat("d", 64)
	prospective.reservedAt = last.At
	if _, err := resultRetainedSize(prospective); err != nil {
		return fail(err)
	}
	if err := a.ledger.CheckResultReplacement(b.Workspace, b.Packet.Preparation, *app.before, app.document, app.links, erpledger.Actor{ID: actor, Kind: erpledger.KindHuman}); err != nil {
		return fail(commands.Fail(commands.PrerequisiteFailed))
	}
	return app, nil
}

func (a *financialERP) replacementObservation(b financialERPResultBasis, app financialERPResultApplication) error {
	snapshot, err := a.ledger.ResultOf(b.Workspace, b.Packet.Preparation)
	if err != nil {
		return commands.Fail(commands.DependencyUnavailable)
	}
	now, err := a.resultNow()
	if err != nil {
		return err
	}
	if app.before == nil || !reflect.DeepEqual(snapshot, *app.before) && !replacementSnapshotMatches(snapshot, b, app, now) {
		return commands.Fail(commands.EvidenceStale)
	}
	return nil
}

func (a *financialERP) applyReplacement(b financialERPResultBasis, r financialERPResultRecord, app financialERPResultApplication, actor string) (financialERPResultView, error) {
	fail := func(err error) (financialERPResultView, error) { return financialERPResultView{}, err }
	if app.actor != actor || a.resultExternals[[3]string{b.Workspace, b.Packet.Connection, b.Packet.External}] != b.ID || a.resultPreparations[[2]string{b.Workspace, b.Packet.Preparation}] != b.ID {
		return fail(commands.Fail(commands.EvidenceStale))
	}
	if app.receipt != "" {
		if err := a.checkResultReceipt(b, app); err != nil {
			return fail(err)
		}
		return a.resultView(r, b), nil
	}
	snapshot, err := a.ledger.ResultOf(b.Workspace, b.Packet.Preparation)
	if err != nil || app.before == nil {
		return fail(errFinancialERPResultUncertain)
	}
	if reflect.DeepEqual(snapshot, *app.before) {
		_, err = a.ledger.ReplaceResult(b.Workspace, b.Packet.Preparation, *app.before, app.document, app.links, erpledger.Actor{ID: actor, Kind: erpledger.KindHuman})
		if err != nil {
			return fail(errFinancialERPResultUncertain)
		}
		if a.afterResultWrite != nil && a.afterResultWrite("replacement") != nil {
			return fail(errFinancialERPResultUncertain)
		}
		snapshot, err = a.ledger.ResultOf(b.Workspace, b.Packet.Preparation)
		if err != nil {
			return fail(errFinancialERPResultUncertain)
		}
	}
	now, err := a.resultNow()
	if err != nil || !replacementSnapshotMatches(snapshot, b, app, now) {
		return fail(errFinancialERPResultUncertain)
	}
	receipt, err := resultApplicationDigest(app.digest, actor, snapshot)
	if err != nil {
		return fail(errFinancialERPResultUncertain)
	}
	app.after, app.receipt = cloneResultSnapshot(snapshot), receipt
	if _, err := resultRetainedSize(app); err != nil {
		return fail(errFinancialERPResultUncertain)
	}
	a.resultApplications[[2]string{b.Workspace, b.ID}] = app
	a.resultTime = now
	return a.resultView(r, b), nil
}
