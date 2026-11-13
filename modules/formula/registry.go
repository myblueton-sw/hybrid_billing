package formula

import (
	"sync"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

type revision struct {
	number       int
	state        string
	author       string
	digest       string
	content      Set
	compiled     *Compiled
	findings     []Finding
	impact       *ImpactRecord
	history      []ImpactRecord
	withdrawal   string
	draftedAt    time.Time
	checkedAt    time.Time
	activatedAt  time.Time
	supersededAt time.Time
	withdrawnAt  time.Time
}

type family struct {
	revisions map[int]*revision
	active    int
	epoch     int
	sequence  int
}

type Registry struct {
	mu         sync.Mutex
	profile    decimal.Profile
	limits     RegistryLimits
	clock      func() time.Time
	now        time.Time
	workspaces map[string]map[string]*family
}

func NewRegistry(profile decimal.Profile, limits RegistryLimits, clock func() time.Time) (*Registry, error) {
	if !profile.Valid() || !limits.valid() || clock == nil {
		return nil, ErrInvalid
	}
	return &Registry{profile: profile, limits: limits, clock: clock, workspaces: map[string]map[string]*family{}}, nil
}

func (r *Registry) enter() time.Time {
	r.mu.Lock()
	now := r.clock().UTC().Round(0)
	if now.After(r.now) {
		r.now = now
	}
	return r.now
}

func (r *Registry) leave() { r.mu.Unlock() }

func validRevision(number int) bool { return number >= 1 && number <= maxRevision }

func (r *Registry) find(workspace, set string, number int) (*family, *revision, bool) {
	f, ok := r.workspaces[workspace][set]
	if !ok {
		return nil, nil, false
	}
	rev, ok := f.revisions[number]
	return f, rev, ok
}

func copyRecord(record ImpactRecord) ImpactRecord {
	if record.Approval != nil {
		approval := *record.Approval
		record.Approval = &approval
	}
	return record
}

func (f *family) view(set string, rev *revision) RevisionView {
	out := RevisionView{
		Set: set, Number: rev.number, State: rev.state, Author: rev.author, Digest: rev.digest,
		Findings: append([]Finding{}, rev.findings...), History: make([]ImpactRecord, 0, len(rev.history)), Withdrawal: rev.withdrawal,
		DraftedAt: rev.draftedAt, CheckedAt: rev.checkedAt, ActivatedAt: rev.activatedAt, SupersededAt: rev.supersededAt, WithdrawnAt: rev.withdrawnAt,
	}
	if rev.impact != nil {
		record := copyRecord(*rev.impact)
		out.Impact, out.Current = &record, rev.impact.Impact.Epoch == f.epoch
	}
	for _, record := range rev.history {
		out.History = append(out.History, copyRecord(record))
	}
	return out
}

func (r *Registry) Draft(workspace, set string, number int, author string, content Set) (RevisionView, error) {
	copied, found := structure(r.profile, r.limits.Limits, content)
	malformed := false
	for _, finding := range found {
		if finding.Code != CodeMalformed {
			return RevisionView{}, ErrLimit
		}
		malformed = true
	}
	if malformed || !validID(workspace) || !validID(set) || !validRevision(number) || !validID(author) {
		return RevisionView{}, ErrInvalid
	}
	digest := contentDigest(copied)
	now := r.enter()
	defer r.leave()
	f, existing, ok := r.find(workspace, set, number)
	if ok {
		if existing.digest != digest || existing.author != author {
			return RevisionView{}, ErrConflict
		}
		return f.view(set, existing), nil
	}
	if f == nil && len(r.workspaces[workspace]) >= r.limits.Sets {
		return RevisionView{}, ErrLimit
	}
	if f != nil && len(f.revisions) >= r.limits.Revisions {
		return RevisionView{}, ErrLimit
	}
	if f == nil {
		if r.workspaces[workspace] == nil {
			r.workspaces[workspace] = map[string]*family{}
		}
		f = &family{revisions: map[int]*revision{}}
		r.workspaces[workspace][set] = f
	}
	rev := &revision{number: number, state: StateDraft, author: author, digest: digest, content: copied, draftedAt: now}
	f.revisions[number] = rev
	f.sequence++
	return f.view(set, rev), nil
}

func (r *Registry) RunCheck(workspace, set string, number int) (RevisionView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(set) || !validRevision(number) {
		return RevisionView{}, ErrInvalid
	}
	f, rev, ok := r.find(workspace, set, number)
	if !ok {
		return RevisionView{}, ErrNotFound
	}
	if rev.state == StateWithdrawn {
		return RevisionView{}, ErrRejected
	}
	if rev.state != StateDraft {
		return f.view(set, rev), nil
	}
	compiled, found, _ := compile(r.profile, r.limits.Limits, rev.content)
	rev.checkedAt = now
	rev.state, rev.compiled, rev.findings = StateChecked, compiled, found
	if compiled == nil {
		rev.state = StateCheckFailed
	}
	f.sequence++
	return f.view(set, rev), nil
}

func validImpact(impact Impact) bool {
	for _, count := range []int{impact.Epoch, impact.Evaluated, impact.Changed, impact.WithoutValue, impact.Failed} {
		if count < 0 || count > maxCount {
			return false
		}
	}
	return validID(impact.Manifest) && validDigest(impact.ManifestDigest) &&
		(impact.Coverage == CoverageFull || impact.Coverage == CoverageSample) &&
		impact.Changed <= impact.Evaluated && impact.WithoutValue+impact.Failed <= impact.Evaluated
}

func (r *Registry) RecordImpact(workspace, set string, number int, impact Impact) (RevisionView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(set) || !validRevision(number) || !validImpact(impact) {
		return RevisionView{}, ErrInvalid
	}
	f, rev, ok := r.find(workspace, set, number)
	if !ok {
		return RevisionView{}, ErrNotFound
	}
	if rev.impact != nil && rev.impact.Impact == impact {
		return f.view(set, rev), nil
	}
	if rev.impact != nil && rev.impact.Approval != nil && rev.impact.Impact.Epoch == f.epoch {
		return RevisionView{}, ErrConflict
	}
	if (rev.state != StateChecked && rev.state != StateImpactRecorded && rev.state != StateApprovalRecorded) || impact.Epoch != f.epoch {
		return RevisionView{}, ErrRejected
	}
	if rev.impact != nil {
		if len(rev.history) >= r.limits.History {
			return RevisionView{}, ErrLimit
		}
		rev.history = append(rev.history, *rev.impact)
	}
	rev.impact = &ImpactRecord{Impact: impact, Subject: subjectDigest(workspace, set, number, rev.digest, impact), At: now}
	rev.state = StateImpactRecorded
	f.sequence++
	return f.view(set, rev), nil
}

func (r *Registry) RecordApproval(workspace, set string, number int, approver, reference, subject string) (RevisionView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(set) || !validRevision(number) || !validID(approver) || !validID(reference) || !validDigest(subject) {
		return RevisionView{}, ErrInvalid
	}
	f, rev, ok := r.find(workspace, set, number)
	if !ok {
		return RevisionView{}, ErrNotFound
	}
	if rev.impact != nil && rev.impact.Approval != nil {
		recorded := rev.impact.Approval
		if recorded.Approver != approver || recorded.Reference != reference || recorded.Subject != subject {
			return RevisionView{}, ErrConflict
		}
		return f.view(set, rev), nil
	}
	if rev.state != StateImpactRecorded {
		return RevisionView{}, ErrRejected
	}
	impact := rev.impact.Impact
	if impact.Epoch != f.epoch || impact.Coverage != CoverageFull || impact.WithoutValue != 0 || impact.Failed != 0 ||
		subject != rev.impact.Subject || approver == rev.author {
		return RevisionView{}, ErrRejected
	}
	rev.impact.Approval = &Approval{Approver: approver, Reference: reference, Subject: subject, At: now}
	rev.state = StateApprovalRecorded
	f.sequence++
	return f.view(set, rev), nil
}

func (r *Registry) Activate(workspace, set string, number int) (RevisionView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(set) || !validRevision(number) {
		return RevisionView{}, ErrInvalid
	}
	f, rev, ok := r.find(workspace, set, number)
	if !ok {
		return RevisionView{}, ErrNotFound
	}
	if rev.state == StateActive {
		return f.view(set, rev), nil
	}
	if rev.state != StateApprovalRecorded || rev.impact.Impact.Epoch != f.epoch {
		return RevisionView{}, ErrRejected
	}
	if previous, ok := f.revisions[f.active]; ok {
		previous.state, previous.supersededAt = StateSuperseded, now
	}
	rev.state, rev.activatedAt = StateActive, now
	f.active = number
	f.epoch++
	f.sequence++
	return f.view(set, rev), nil
}

func (r *Registry) Withdraw(workspace, set string, number int, reason string) (RevisionView, error) {
	now := r.enter()
	defer r.leave()
	if !validID(workspace) || !validID(set) || !validRevision(number) || !withdrawals[reason] {
		return RevisionView{}, ErrInvalid
	}
	f, rev, ok := r.find(workspace, set, number)
	if !ok {
		return RevisionView{}, ErrNotFound
	}
	if rev.state == StateWithdrawn {
		if rev.withdrawal != reason {
			return RevisionView{}, ErrConflict
		}
		return f.view(set, rev), nil
	}
	if rev.state == StateActive {
		f.active = 0
		f.epoch++
	}
	rev.state, rev.withdrawal, rev.withdrawnAt = StateWithdrawn, reason, now
	f.sequence++
	return f.view(set, rev), nil
}
