package decisionledger

import "strconv"

func (l *Ledger) CommitBatch(batch []Decision) ([]Receipt, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.refusal(); err != nil {
		return nil, err
	}
	if len(batch) == 0 || len(batch) > BatchLimit {
		return nil, &CommitError{Reason: ReasonCommitFailed}
	}
	recs := make([]line, len(batch))
	for i, d := range batch {
		recs[i] = lineFor(d)
		recs[i].Seq = l.seq + uint64(i) + 1
		if recs[i].Trust != nil || !recs[i].shaped() {
			return nil, &CommitError{Reason: ReasonCommitFailed}
		}
	}
	at, first := l.now(), l.seq
	if reason := l.appendLines(recs, at); reason != "" {
		return nil, &CommitError{Reason: reason}
	}
	out := make([]Receipt, len(batch))
	last := l.last
	for i := len(recs) - 1; i >= 0; i-- {
		seq := first + uint64(i) + 1
		mark := strconv.FormatUint(seq, 10) + ":" + last
		out[i] = Receipt{Sequence: seq, Watermark: mark}
		last = recs[i].Prev
	}
	for i, d := range batch {
		d.EndedDelegations = recs[i].EndedDelegations
		d.Clock = recs[i].decision().Clock
		l.decisions = append(l.decisions, Record{Sequence: out[i].Sequence, Watermark: out[i].Watermark, At: at, Decision: d})
	}
	return out, nil
}

func (l *Ledger) Stop() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.status.State != StateIsolated {
		l.status.State = StateStopped
	}
}
