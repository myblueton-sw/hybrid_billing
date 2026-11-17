package accesschange

import (
	"strconv"
	"time"
)

type revocation struct {
	view RevocationView
}

type committed interface{ CommitReason() string }

func copyItems(items []Item) []Item {
	out := make([]Item, len(items))
	for i, it := range items {
		it.Refs = append([]string(nil), it.Refs...)
		out[i] = it
	}
	return out
}

func copyRevocation(v RevocationView) RevocationView {
	v.History = append([]Transition(nil), v.History...)
	v.Ended = append([]string(nil), v.Ended...)
	v.Cleanup = copyItems(v.Cleanup)
	return v
}

func (in Revocation) valid() bool {
	return in.Workspace != "" && in.Target.ID != "" && (in.Target.Kind == TargetGrant || in.Target.Kind == TargetDelegation) && in.RequestedBy != ""
}

func (r *Registry) Revoke(in Revocation, e Effects) (RevocationView, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.revoke(in, e)
}

func (v *RevocationView) move(state string, at time.Time) {
	v.State = state
	v.History = append(v.History, Transition{State: state, At: at})
}

func (r *Registry) revoke(in Revocation, e Effects) (RevocationView, error) {
	if !in.valid() || e.Deny == nil || e.Ledger == nil || e.Cleanup == nil {
		return RevocationView{}, ErrInvalid
	}
	denyAt := r.now()
	denied, err := e.Deny()
	if err != nil {
		return RevocationView{}, err
	}
	r.nextRevocation++
	rec := &revocation{view: RevocationView{ID: RevocationPrefix + strconv.FormatUint(r.nextRevocation, 10), Origin: OriginProcess, RecordedAt: denyAt,
		Workspace: in.Workspace, Target: in.Target, TargetPresent: true, RequestedBy: in.RequestedBy, ApprovedBy: in.ApprovedBy, Request: in.Request,
		RequestDigest: in.RequestDigest, DenyAt: denyAt, DenyGeneration: denied.Generation, Ended: append([]string(nil), denied.Ended...)}}
	v := &rec.view
	v.move(RevocationReceived, r.now())
	r.revocations = append(r.revocations, rec)
	receipt, err := e.Ledger.Commit(Entry{Workspace: in.Workspace, Target: in.Target, Revocation: v.ID, Request: in.Request, RequestDigest: in.RequestDigest,
		RequestedBy: in.RequestedBy, ApprovedBy: in.ApprovedBy, DenyGeneration: denied.Generation, Ended: append([]string(nil), denied.Ended...)})
	switch {
	case err != nil:
		v.Reason = CommitFailed
		if c, ok := err.(committed); ok {
			v.Reason = c.CommitReason()
		}
	case receipt.Sequence <= r.last.Sequence || receipt.Watermark == r.last.Watermark:
		v.Reason = CommitUnknown
	}
	if v.Reason != "" {
		v.Cleanup = copyItems(e.Planned)
		for i := range v.Cleanup {
			v.Cleanup[i].State = ItemNotStarted
		}
		v.move(RevocationUnconfirmed, r.now())
		r.pruneRevocations()
		return copyRevocation(*v), nil
	}
	r.last, v.Receipt = receipt, receipt
	v.move(RevocationConfirmed, r.now())
	v.move(RevocationCleaning, r.now())
	v.Cleanup = copyItems(e.Cleanup(denied))
	v.move(RevocationComplete, r.now())
	v.CompletedAt = r.now()
	r.pruneRevocations()
	return copyRevocation(*v), nil
}

func (r *Registry) pruneRevocations() {
	for i := 0; len(r.revocations) > r.limits.RevocationsKept && i < len(r.revocations); {
		if r.revocations[i].view.State == RevocationComplete {
			r.revocations = append(r.revocations[:i], r.revocations[i+1:]...)
			continue
		}
		i++
	}
}

func (r *Registry) Restore(records []Restored, last Receipt, requests, revocations uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, x := range records {
		v := RevocationView{ID: x.ID, Origin: OriginReplay, RecordedAt: x.RecordedAt.UTC(), Workspace: x.Revocation.Workspace, Target: x.Revocation.Target,
			TargetPresent: x.Present, Replay: x.Replay, RequestedBy: x.Revocation.RequestedBy, ApprovedBy: x.Revocation.ApprovedBy, Request: x.Revocation.Request,
			RequestDigest: x.Revocation.RequestDigest, DenyAt: x.RecordedAt.UTC(), DenyGeneration: x.Generation, Ended: append([]string(nil), x.Ended...),
			Receipt: x.Receipt, Cleanup: copyItems(x.Cleanup), CompletedAt: x.RecordedAt.UTC()}
		v.move(RevocationComplete, x.RecordedAt.UTC())
		r.revocations = append(r.revocations, &revocation{view: v})
	}
	r.pruneRevocations()
	r.last = last
	r.nextRequest = max(r.nextRequest, requests)
	r.nextRevocation = max(r.nextRevocation, revocations)
}

func (r *Registry) Revocations(workspace string) []RevocationView {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []RevocationView{}
	for i := len(r.revocations) - 1; i >= 0; i-- {
		if v := r.revocations[i].view; v.Workspace == workspace {
			out = append(out, copyRevocation(v))
		}
	}
	return out
}
