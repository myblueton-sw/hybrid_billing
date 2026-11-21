package server

import (
	"encoding/json"
	"errors"
	"reflect"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/erpledger"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

var errFinancialERPResultUncertain = errors.New("ERP result application outcome remains unresolved")

type financialERPResultApplication struct {
	actor, digest, receipt string
	document               erpledger.Document
	links                  []erpledger.Link
}

func resultIntent(b financialERPResultBasis, actor, digest string) financialERPResultApplication {
	p := b.Packet
	out := financialERPResultApplication{actor: actor, digest: digest, document: erpledger.Document{Connection: p.Connection, External: p.External, Revision: p.Revision}}
	for _, r := range p.Rows {
		out.document.Rows = append(out.document.Rows, erpledger.Row{ID: r.ID, Product: r.Product, Net: r.Net, Tax: r.Tax, Quantity: r.Quantity})
		out.links = append(out.links, erpledger.Link{Connection: p.Connection, External: p.External, Row: r.ID, Key: p.Preparation, Line: r.Line, Net: r.Net, Tax: r.Tax, Quantity: r.Quantity})
	}
	return out
}

func resultDocumentMatches(v erpledger.DocumentView, app financialERPResultApplication, total erpledger.Amounts) bool {
	if !reflect.DeepEqual(v.Document, app.document) || v.Author != app.actor || v.At.IsZero() || v.Total != total {
		return false
	}
	seen := map[erpledger.Link]bool{}
	for _, actual := range v.Links {
		if actual.Stale || actual.Author != app.actor || seen[actual.Link] {
			return false
		}
		found := false
		for _, want := range app.links {
			if actual.Link == want {
				found = true
				break
			}
		}
		if !found {
			return false
		}
		seen[actual.Link] = true
	}
	return true
}

func resultSnapshotMatches(v erpledger.ResultView, b financialERPResultBasis, app financialERPResultApplication, complete bool) bool {
	if !reflect.DeepEqual(v.Submission, b.Preparation) || len(v.Documents) > 1 {
		return false
	}
	if len(v.Documents) == 1 && !resultDocumentMatches(v.Documents[0], app, b.Total) {
		return false
	}
	if !complete {
		return true
	}
	if len(v.Documents) != 1 || len(v.Documents[0].Links) != len(app.links) || !v.Reconciliation.Matched || v.Reconciliation.Key != b.Packet.Preparation || len(v.Reconciliation.Documents) != 1 || len(v.Reconciliation.Lines) != len(b.Preparation.Submission.Lines) {
		return false
	}
	d := v.Reconciliation.Documents[0]
	if d.Connection != b.Packet.Connection || d.External != b.Packet.External || d.Total != b.Total || d.Attributed != b.Total || b.Total != b.Preparation.Total {
		return false
	}
	seen := map[string]bool{}
	for _, line := range v.Reconciliation.Lines {
		if !line.Matched || seen[line.Line] {
			return false
		}
		seen[line.Line] = true
		found := false
		for _, want := range b.Preparation.Submission.Lines {
			if want.ID == line.Line && line.Linked == resultLineAmounts(want, b.Document.Connection.Connection.Scale) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (a *financialERP) currentResult(b financialERPResultBasis, r financialERPResultRecord, actor string) error {
	if r.review.State != "confirmed" || len(b.Comparison) != 0 || a.server.held() {
		return commands.Fail(commands.PrerequisiteFailed)
	}
	_, watermark, err := a.resultWatermark(b)
	if err != nil {
		return err
	}
	if watermark != r.review.Digest {
		return commands.Fail(commands.EvidenceStale)
	}
	for _, right := range []struct {
		actor  string
		action gate.Action
	}{{r.review.Proposer, gate.ProposeERPResult}, {r.review.Confirmation.Actor, gate.ConfirmERPResult}} {
		if err := a.resultStanding(b.Workspace, right.actor, right.action); err != nil {
			return err
		}
	}
	document, basis, err := a.governedDocument(b.Workspace, b.Document.ID, b.DocumentDigest)
	if err != nil || document.review.State != "confirmed" || !reflect.DeepEqual(basis, b.Document) {
		return commands.Fail(commands.EvidenceStale)
	}
	if _, err = a.currentDocumentLocked(b.Document, actor); err != nil {
		return err
	}
	historical, preparation, digest, err := a.resultHistoricalPreparation(b.Workspace, b.Packet)
	if err != nil {
		return err
	}
	if digest != b.DocumentDigest || !reflect.DeepEqual(historical, b.Document) || !reflect.DeepEqual(preparation, b.Preparation) {
		return commands.Fail(commands.EvidenceStale)
	}
	_, err = a.resultNow()
	return err
}

func (a *financialERP) applyResultLocked(workspace, id, digest, actor string) (financialERPResultView, error) {
	fail := func(err error) (financialERPResultView, error) { return financialERPResultView{}, err }
	if err := a.resultPermission(workspace, actor, gate.ApplyERPResult); err != nil {
		return fail(err)
	}
	r, b, err := a.resultRecord(workspace, id, digest)
	if err != nil {
		return fail(err)
	}
	if err = a.currentResult(b, r, actor); err != nil {
		return fail(err)
	}
	key, external, preparation := [2]string{workspace, id}, [3]string{workspace, b.Packet.Connection, b.Packet.External}, [2]string{workspace, b.Packet.Preparation}
	intent := resultIntent(b, actor, digest)
	app, owned := a.resultApplications[key]
	if owned {
		previous := app
		previous.receipt = ""
		if !reflect.DeepEqual(previous, intent) || a.resultExternals[external] != id || a.resultPreparations[preparation] != id {
			return fail(commands.Fail(commands.EvidenceStale))
		}
	} else {
		if a.resultExternals[external] != "" || a.resultPreparations[preparation] != "" {
			return fail(commands.Fail(commands.EvidenceStale))
		}
		if len(a.resultApplications) >= financialDocumentLimit || len(a.resultExternals) >= financialDocumentLimit || len(a.resultPreparations) >= financialDocumentLimit {
			return fail(commands.Fail(commands.PrerequisiteFailed))
		}
		prospective := struct {
			Basis         financialERPResultBasis
			Document      erpledger.Document
			Links         []erpledger.Link
			Actor, Digest string
		}{b, intent.document, intent.links, actor, digest}
		raw, err := json.Marshal(prospective)
		if err != nil || len(raw) > financialDocumentBytes/2 {
			return fail(commands.Fail(commands.PrerequisiteFailed))
		}
	}
	view, lookup := a.ledger.DocumentOf(workspace, b.Packet.Connection, b.Packet.External)
	if lookup != nil && !errors.Is(lookup, erpledger.ErrNotFound) {
		if owned {
			return fail(errFinancialERPResultUncertain)
		}
		return fail(commands.Fail(commands.DependencyUnavailable))
	}
	if !owned && lookup == nil {
		return fail(commands.Fail(commands.EvidenceStale))
	}
	if owned && lookup == nil && !resultDocumentMatches(view, app, b.Total) {
		return fail(errFinancialERPResultUncertain)
	}
	if owned && lookup != nil && app.receipt != "" {
		return fail(errFinancialERPResultUncertain)
	}
	snapshot, err := a.ledger.ResultOf(workspace, b.Packet.Preparation)
	if err != nil {
		if owned {
			return fail(errFinancialERPResultUncertain)
		}
		return fail(commands.Fail(commands.DependencyUnavailable))
	}
	if !resultSnapshotMatches(snapshot, b, intent, false) || !owned && len(snapshot.Documents) != 0 || lookup != nil && len(snapshot.Documents) != 0 {
		if owned {
			return fail(errFinancialERPResultUncertain)
		}
		return fail(commands.Fail(commands.EvidenceStale))
	}
	if !owned {
		app = intent
		a.resultApplications[key] = app
		a.resultExternals[external] = id
		a.resultPreparations[preparation] = id
	}
	human := erpledger.Actor{ID: actor, Kind: erpledger.KindHuman}
	if lookup != nil {
		view, err = a.ledger.Document(workspace, app.document, human)
		if err != nil {
			return fail(errFinancialERPResultUncertain)
		}
		if a.afterResultWrite != nil && a.afterResultWrite("document") != nil {
			return fail(errFinancialERPResultUncertain)
		}
		if !resultDocumentMatches(view, app, b.Total) {
			return fail(errFinancialERPResultUncertain)
		}
	}
	for _, link := range app.links {
		exists := false
		for _, actual := range view.Links {
			if actual.Link == link {
				exists = true
				break
			}
		}
		if exists {
			continue
		}
		if app.receipt != "" {
			return fail(errFinancialERPResultUncertain)
		}
		view, err = a.ledger.Link(workspace, link, human)
		if err != nil {
			return fail(errFinancialERPResultUncertain)
		}
		if a.afterResultWrite != nil && a.afterResultWrite("link:"+link.Row) != nil {
			return fail(errFinancialERPResultUncertain)
		}
		if !resultDocumentMatches(view, app, b.Total) {
			return fail(errFinancialERPResultUncertain)
		}
	}
	snapshot, err = a.ledger.ResultOf(workspace, b.Packet.Preparation)
	if err != nil || !resultSnapshotMatches(snapshot, b, app, true) {
		return fail(errFinancialERPResultUncertain)
	}
	receipt, err := resultApplicationDigest(digest, actor, snapshot)
	if err != nil || app.receipt != "" && app.receipt != receipt {
		return fail(errFinancialERPResultUncertain)
	}
	app.receipt = receipt
	a.resultApplications[key] = app
	return a.resultView(r, b), nil
}

func resultApplicationDigest(digest, actor string, snapshot erpledger.ResultView) (string, error) {
	return normalizedDigest(struct {
		Schema, Digest, Actor string
		Result                erpledger.ResultView
	}{"hb_erp_result_application_v1", digest, actor, snapshot})
}
