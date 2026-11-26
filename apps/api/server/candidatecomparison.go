package server

import (
	"encoding/json"

	"github.com/myblueton-sw/hybrid_billing/modules/inputreview"
	"github.com/myblueton-sw/hybrid_billing/modules/normalize"
)

func (p *normalizedCharges) comparisonSnapshotLocked(workspace, runID string, units []inputreview.UnitBinding) (inputreview.Snapshot, error) {
	out := inputreview.Snapshot{}
	run, err := p.normal.Run(workspace, runID)
	if err != nil {
		return out, err
	}
	source, err := p.sourceForRun(workspace, runID, run.Spec)
	if err != nil {
		return out, err
	}
	held := p.sources[[2]string{workspace, runID}]
	var packet sourcePacket
	if err := json.Unmarshal(held.Packet, &packet); err != nil {
		return out, err
	}
	prepared, err := p.prepareSource(packet)
	if err != nil {
		return out, err
	}
	mapping, err := p.normal.Revision(run.Spec.Mapping, run.Spec.Revision)
	if err != nil {
		return out, err
	}
	latest, err := p.normal.Latest(run.Spec.Mapping)
	if err != nil {
		return out, err
	}
	if latest.Number != run.Spec.Revision {
		return out, errCandidateConflict
	}
	if run.Spec.Rows < 1 || run.Spec.Rows > 1000 || run.Normal < 0 || run.Normal > run.Spec.Rows {
		return out, errCandidateInvalid
	}
	out = inputreview.Snapshot{Scope: inputreview.Scope{Workspace: workspace, Provider: source.Provider, Account: source.Scope, Period: source.Period}, Manifest: source.Manifest, Content: source.Content, Attribution: source.Attribution, Profile: p.profileSpec, Target: p.target, Mapping: mapping, Run: run, Raw: prepared.rows, Units: append([]inputreview.UnitBinding(nil), units...)}
	out.Target.Fields = append([]normalize.Field(nil), p.target.Fields...)
	for after := 0; after < run.Normal; {
		page, err := p.normal.Rows(workspace, runID, after, p.page)
		if err != nil {
			return inputreview.Snapshot{}, err
		}
		if len(page.Rows) == 0 || len(page.Rows) > p.page || after+len(page.Rows) > run.Normal {
			return inputreview.Snapshot{}, errCandidateInvalid
		}
		out.Rows = append(out.Rows, page.Rows...)
		after += len(page.Rows)
	}
	return out, nil
}

func (p *normalizedCharges) compareSourceLocked(workspace, runID string, units []inputreview.UnitBinding) (inputreview.Comparison, error) {
	snapshot, err := p.comparisonSnapshotLocked(workspace, runID, units)
	if err != nil {
		return inputreview.Comparison{}, err
	}
	return inputreview.CompareSnapshot(snapshot)
}
