package server

import (
	"encoding/json"
	"errors"

	"github.com/myblueton-sw/hybrid_billing/modules/intake"
	"github.com/myblueton-sw/hybrid_billing/modules/normalize"
)

func sourceObserve(status intake.StatusView, expected uint64) (sourceAdmission, error) {
	out := sourceAdmission{Difference: status.Differs, Latest: status.Latest, Staging: status.Staging, Behind: status.Behind.Total, SameBytes: status.SameBytes.Total, Blockers: status.BlockerTotal}
	if out.Difference == intake.Same {
		out.Difference = "same"
	}
	reason := ""
	switch {
	case out.Latest != expected:
		reason = "manifest_changed"
	case out.Blockers != 0:
		reason = "intake_blockers"
	case out.Staging != 0:
		reason = "staged_artifacts"
	case out.Behind != 0:
		reason = "behind_revision"
	case out.SameBytes != 0:
		reason = "duplicate_byte_signal"
	}
	if reason != "" {
		return out, sourceCondition{reason, out, expected}
	}
	return out, nil
}

func (p *normalizedCharges) reserveSource(prepared sourcePrepared) (*sourceAssembly, error) {
	key := [2]string{prepared.packet.Run.WorkspaceID, prepared.packet.Run.ID}
	if held := p.sources[key]; held != nil {
		if held.Print != prepared.print {
			return nil, errCandidateConflict
		}
		return held, nil
	}
	if len(p.sources) >= 1000 || len(p.sourceArtifacts)+len(prepared.specs) > 1000 {
		return nil, errCandidateLimit
	}
	if _, err := p.normal.Run(key[0], key[1]); err == nil {
		return nil, errCandidateConflict
	} else if !errors.Is(err, normalize.ErrNotFound) {
		return nil, err
	}
	for _, spec := range prepared.specs {
		if _, exists := p.sourceArtifacts[[2]string{key[0], spec.ID}]; exists {
			return nil, errCandidateConflict
		}
		if _, err := p.intake.Artifact(key[0], spec.ID); err == nil {
			return nil, errCandidateConflict
		} else if !sourceAbsent(err) {
			return nil, err
		}
	}
	status, err := p.intake.Status(prepared.subject)
	if err != nil && !sourceAbsent(err) {
		return nil, err
	}
	held := &sourceAssembly{Print: prepared.print, Packet: append([]byte(nil), prepared.bytes...), Base: status.Latest, Artifacts: make(map[string]bool)}
	p.sources[key] = held
	for _, spec := range prepared.specs {
		p.sourceArtifacts[[2]string{key[0], spec.ID}] = key
	}
	return held, nil
}

func (p *normalizedCharges) stageSourceArtifact(prepared sourcePrepared, held *sourceAssembly, index int) error {
	spec := prepared.specs[index]
	if !held.Artifacts[spec.ID] {
		if _, err := p.intake.Artifact(spec.Subject.WorkspaceID, spec.ID); err == nil {
			return errCandidateConflict
		} else if !sourceAbsent(err) {
			return err
		}
	}
	state, _, err := p.intake.Register(spec)
	if err != nil {
		return err
	}
	if state != intake.Staging && state != intake.Sealed {
		return errCandidateInvalid
	}
	held.Artifacts[spec.ID] = true
	rows := prepared.records[index]
	for after, chunk := 0, 1; after < len(rows); chunk++ {
		end := min(after+p.sourceChunk, len(rows))
		state, err = p.intake.Stage(spec.Subject.WorkspaceID, spec.ID, chunk, rows[after:end])
		if err != nil {
			return err
		}
		if state != intake.Staging && state != intake.Sealed {
			return errCandidateInvalid
		}
		after = end
	}
	state, gaps, err := p.intake.Seal(spec.Subject.WorkspaceID, spec.ID)
	if err != nil {
		return err
	}
	if state != intake.Sealed || len(gaps) != 0 {
		return errCandidateInvalid
	}
	return nil
}

func (p *normalizedCharges) verifySource(prepared sourcePrepared, held *sourceAssembly) (candidateSource, error) {
	out := candidateSource{}
	status, err := p.intake.Status(prepared.subject)
	if err != nil {
		return out, err
	}
	admission, err := sourceObserve(status, held.Manifest)
	if err != nil {
		return out, err
	}
	if status.Differs != intake.Same {
		return out, sourceCondition{Reason: "unpublished_" + status.Differs, Observed: admission, Expected: held.Manifest}
	}
	manifest, err := p.intake.Manifest(prepared.subject, held.Manifest)
	if err != nil {
		return out, err
	}
	if manifest.Subject != prepared.subject || manifest.Number != held.Manifest || manifest.Records != len(prepared.origins) {
		return out, errCandidateInvalid
	}
	out = candidateSource{Workspace: prepared.subject.WorkspaceID, Provider: prepared.subject.Provider, Scope: prepared.subject.Scope, Period: prepared.subject.Period, Version: prepared.packet.Run.Mapping.SourceVersion, CostField: prepared.packet.CostField, Currency: prepared.packet.CurrencyField, Manifest: manifest.Number, Content: manifest.Content, Attribution: manifest.Attribution, Admission: admission, Rows: []candidateSourceRow{}}
	seen := make(map[normalize.RowKey]bool)
	for after := 0; after < manifest.Records; {
		rows, err := p.intake.Records(prepared.subject, held.Manifest, after, p.page)
		if err != nil {
			return candidateSource{}, err
		}
		if len(rows) == 0 || len(rows) > p.page || after+len(rows) > manifest.Records {
			return candidateSource{}, errCandidateInvalid
		}
		for _, row := range rows {
			key := normalize.RowKey{Kind: normalize.KeyKind(row.Kind), Document: row.Document, Key: row.Key}
			expected, exists := prepared.origins[key]
			if !exists || seen[key] || expected.Artifact != row.Artifact || expected.Digest != row.Digest || expected.Amount != row.Amount || expected.Currency != row.Currency {
				return candidateSource{}, errCandidateInvalid
			}
			seen[key] = true
			out.Rows = append(out.Rows, expected)
		}
		after += len(rows)
	}
	if len(seen) != len(prepared.origins) {
		return candidateSource{}, errCandidateInvalid
	}
	return out, nil
}

func (p *normalizedCharges) sourceRun(prepared sourcePrepared, held *sourceAssembly) (normalize.RunSpec, error) {
	run := prepared.packet.Run
	run.Input.Manifest = held.Manifest
	run.Rows = len(prepared.rows)
	run.Chunks = (run.Rows + p.sourceChunk - 1) / p.sourceChunk
	if run.Window != nil {
		run.Window = &normalize.Window{From: run.Window.From.UTC(), To: run.Window.To.UTC()}
	}
	run.Expected = make(map[string]map[string]string)
	for field, sums := range prepared.packet.Run.Expected {
		run.Expected[field] = make(map[string]string)
		for currency, amount := range sums {
			canonical, err := p.sourceAmount(amount)
			if err != nil {
				return normalize.RunSpec{}, err
			}
			run.Expected[field][currency] = canonical
		}
	}
	return run, nil
}

func (p *normalizedCharges) assembleSource(packet sourcePacket) (candidateSource, error) {
	p.domain.Lock()
	defer p.domain.Unlock()
	return p.assembleSourceLocked(packet)
}

func (p *normalizedCharges) assembleSourceLocked(packet sourcePacket) (candidateSource, error) {
	prepared, err := p.prepareSource(packet)
	if err != nil {
		return candidateSource{}, err
	}
	held, err := p.reserveSource(prepared)
	if err != nil {
		return candidateSource{}, err
	}
	for index := range prepared.specs {
		if err := p.stageSourceArtifact(prepared, held, index); err != nil {
			return candidateSource{}, err
		}
	}
	if held.Manifest == 0 {
		status, err := p.intake.Status(prepared.subject)
		if err != nil {
			return candidateSource{}, err
		}
		if _, err := sourceObserve(status, held.Base); err != nil {
			return candidateSource{}, err
		}
		published, err := p.intake.Publish(prepared.subject, held.Base)
		if err != nil {
			return candidateSource{}, err
		}
		if published.Number == 0 || published.BlockerTotal != 0 {
			return candidateSource{}, errCandidateInvalid
		}
		held.Manifest = published.Number
	}
	origin, err := p.verifySource(prepared, held)
	if err != nil {
		return candidateSource{}, err
	}
	run, err := p.sourceRun(prepared, held)
	if err != nil {
		return candidateSource{}, err
	}
	if !held.Normalized {
		if _, err := p.normal.Run(run.WorkspaceID, run.ID); err == nil {
			return candidateSource{}, errCandidateConflict
		} else if !errors.Is(err, normalize.ErrNotFound) {
			return candidateSource{}, err
		}
	}
	state, err := p.normal.Open(run)
	if err != nil {
		return candidateSource{}, err
	}
	if state != normalize.Open && state != normalize.Closed {
		return candidateSource{}, errCandidateInvalid
	}
	held.Normalized = true
	for after, chunk := 0, 1; after < len(prepared.rows); chunk++ {
		end := min(after+p.sourceChunk, len(prepared.rows))
		state, err = p.normal.Check(run.WorkspaceID, run.ID, chunk, prepared.rows[after:end])
		if err != nil {
			return candidateSource{}, err
		}
		if state != normalize.Open && state != normalize.Closed {
			return candidateSource{}, errCandidateInvalid
		}
		after = end
	}
	state, _, err = p.normal.Close(run.WorkspaceID, run.ID)
	if err != nil {
		return candidateSource{}, err
	}
	if state != normalize.Closed {
		return candidateSource{}, errCandidateInvalid
	}
	held.Complete = true
	return origin, nil
}

func (p *normalizedCharges) sourceForRun(workspace, id string, run normalize.RunSpec) (candidateSource, error) {
	held := p.sources[[2]string{workspace, id}]
	if held == nil || !held.Complete {
		return candidateSource{}, errCandidateInvalid
	}
	var packet sourcePacket
	if err := json.Unmarshal(held.Packet, &packet); err != nil {
		return candidateSource{}, err
	}
	prepared, err := p.prepareSource(packet)
	if err != nil {
		return candidateSource{}, err
	}
	if prepared.print != held.Print || packet.Run.WorkspaceID != workspace || packet.Run.ID != id {
		return candidateSource{}, errCandidateInvalid
	}
	expected, err := p.sourceRun(prepared, held)
	if err != nil {
		return candidateSource{}, err
	}
	actualDigest, err := normalizedDigest(run)
	if err != nil {
		return candidateSource{}, err
	}
	expectedDigest, err := normalizedDigest(expected)
	if err != nil {
		return candidateSource{}, err
	}
	if actualDigest != expectedDigest {
		return candidateSource{}, errCandidateInvalid
	}
	return p.verifySource(prepared, held)
}
