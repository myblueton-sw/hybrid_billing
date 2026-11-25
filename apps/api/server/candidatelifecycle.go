package server

import (
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/contracts"
)

type candidateLifecycleSegment struct {
	From     string `json:"from"`
	To       string `json:"to_exclusive"`
	State    string `json:"state"`
	Revision uint64 `json:"contract_revision"`
}

type candidateLifecycle struct {
	Contract string                      `json:"contract"`
	Sequence uint64                      `json:"journal_sequence"`
	From     string                      `json:"from"`
	To       string                      `json:"to_exclusive"`
	Segments []candidateLifecycleSegment `json:"segments"`
}

func (p *normalizedCharges) lifecycleCoverage(workspace string, b normalizedBinding) (candidateLifecycle, error) {
	out := candidateLifecycle{}
	if p.lifecycles == nil || b.ContractRevision == 0 {
		return out, errCandidateInvalid
	}
	from, err := time.Parse(time.DateOnly, b.From)
	if err != nil || from.Format(time.DateOnly) != b.From {
		return out, errCandidateInvalid
	}
	to, err := time.Parse(time.DateOnly, b.To)
	if err != nil || to.Format(time.DateOnly) != b.To {
		return out, errCandidateInvalid
	}
	start, err := contracts.NewDay(from.Year(), int(from.Month()), from.Day())
	if err != nil {
		return out, err
	}
	last, err := contracts.NewDay(to.Year(), int(to.Month()), to.Day())
	if err != nil {
		return out, err
	}
	end, err := last.Add(1)
	if err != nil {
		return out, err
	}
	period, err := contracts.Between(start, end)
	if err != nil {
		return out, err
	}
	sequence, err := p.lifecycles.Sequence(workspace, b.Contract)
	if err != nil {
		return out, err
	}
	segments, err := p.lifecycles.SegmentsAt(workspace, b.Contract, sequence, period)
	if err != nil {
		return out, err
	}
	if len(segments) == 0 {
		return out, errCandidateInvalid
	}
	out = candidateLifecycle{Contract: b.Contract, Sequence: sequence, From: start.String(), To: end.String(), Segments: []candidateLifecycleSegment{}}
	cursor := start
	for _, segment := range segments {
		if !segment.Interval.Valid() || segment.Interval.Start() != cursor || end.Before(segment.Interval.End()) || segment.State != contracts.Active || segment.Revision != b.ContractRevision {
			return candidateLifecycle{}, errCandidateInvalid
		}
		out.Segments = append(out.Segments, candidateLifecycleSegment{segment.Interval.Start().String(), segment.Interval.End().String(), string(segment.State), segment.Revision})
		cursor = segment.Interval.End()
	}
	if cursor != end {
		return candidateLifecycle{}, errCandidateInvalid
	}
	return out, nil
}
