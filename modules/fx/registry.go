package fx

import (
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/exact"
)

type revision struct {
	input      ObservationInput
	state      string
	recordedAt time.Time
	approval   *Approval
	void       *Void
}

type version struct {
	input      VersionInput
	effective  time.Time
	state      string
	recordedAt time.Time
	approval   *Approval
	withdrawal *Withdrawal
}

type place struct {
	level   string
	key     string
	purpose string
	source  string
	target  string
}

type policy struct {
	id       string
	attrs    PolicyAttributes
	versions []*version
}

type stored struct {
	snapshot Snapshot
	needs    []Need
	rates    []rates
}

type space struct {
	keys      map[Key][]*revision
	policies  map[string]*policy
	places    map[place]string
	snapshots map[string]*stored
}

type Registry struct {
	mu         sync.Mutex
	profile    decimal.Profile
	limits     Limits
	clock      func() time.Time
	now        time.Time
	workspaces map[string]*space
}

func NewRegistry(profile decimal.Profile, limits Limits, clock func() time.Time) (*Registry, error) {
	if !profile.Valid() || !limits.valid() || clock == nil {
		return nil, ErrInvalid
	}
	return &Registry{profile: profile, limits: limits, clock: clock, workspaces: map[string]*space{}}, nil
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

func (r *Registry) space(workspace string) *space {
	s, ok := r.workspaces[workspace]
	if !ok {
		s = &space{keys: map[Key][]*revision{}, policies: map[string]*policy{}, places: map[place]string{}, snapshots: map[string]*stored{}}
		r.workspaces[workspace] = s
	}
	return s
}

func (r *Registry) number(text string) (string, exact.Fraction, bool) {
	if len(text) > maxNumberBytes {
		return "", exact.Fraction{}, false
	}
	value, err := decimal.Parse(r.profile, text)
	if err != nil {
		return "", exact.Fraction{}, false
	}
	made, _ := exact.Of(value)
	return value.String(), made, true
}

func (r *Registry) positive(text string) (string, bool) {
	canonical, value, ok := r.number(text)
	return canonical, ok && value.Sign() > 0
}

func (r *Registry) share(text string) (string, bool) {
	canonical, value, ok := r.number(text)
	return canonical, ok && value.Sign() >= 0 && value.Less(exact.Whole(1))
}

func validKey(key Key) bool {
	_, dated := parseDate(key.Date)
	return validID(key.Source) && validID(key.Series) && validCurrency(key.Base) && validCurrency(key.Quote) && key.Base != key.Quote && dated
}

func copyApproval(approval *Approval) *Approval {
	if approval == nil {
		return nil
	}
	made := *approval
	return &made
}

func viewRevision(rev *revision) RevisionView {
	out := RevisionView{Revision: rev.input.Revision, State: rev.state, Unit: rev.input.Unit, Value: rev.input.Value, Origin: rev.input.Origin,
		Evidence: rev.input.Evidence, Recorder: rev.input.Recorder, RecordedAt: rev.recordedAt, Approval: copyApproval(rev.approval)}
	if rev.void != nil {
		made := *rev.void
		made.Approval = copyApproval(rev.void.Approval)
		out.Void = &made
	}
	return out
}

func (r *Registry) RecordObservation(workspace string, in ObservationInput) (RevisionView, error) {
	canonical, formed := r.positive(in.Value)
	evidenced := (in.Origin == OriginFeed) == (in.Evidence == "") && (in.Evidence == "" || validID(in.Evidence))
	if !validID(workspace) || !validKey(in.Key) || in.Revision < 1 || in.Revision > maxRevision || !units[in.Unit] ||
		!formed || !origins[in.Origin] || !evidenced || !validID(in.Recorder) {
		return RevisionView{}, ErrInvalid
	}
	in.Value = canonical
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	var revisions []*revision
	if s != nil {
		revisions = s.keys[in.Key]
	}
	if len(revisions) == 0 && in.Revision != 1 {
		return RevisionView{}, ErrNotFound
	}
	if in.Revision <= len(revisions) {
		if revisions[in.Revision-1].input != in {
			return RevisionView{}, ErrConflict
		}
		return viewRevision(revisions[in.Revision-1]), nil
	}
	if in.Revision > len(revisions)+1 {
		return RevisionView{}, ErrRejected
	}
	if len(revisions) >= r.limits.Revisions || (len(revisions) == 0 && s != nil && len(s.keys) >= r.limits.Keys) {
		return RevisionView{}, ErrLimit
	}
	state := ObservationRecorded
	if in.Origin != OriginFeed {
		state = ObservationAwaiting
	}
	rev := &revision{input: in, state: state, recordedAt: now}
	s = r.space(workspace)
	s.keys[in.Key] = append(revisions, rev)
	return viewRevision(rev), nil
}

func (r *Registry) findRevision(workspace string, key Key, number int) (*revision, error) {
	s := r.workspaces[workspace]
	if s == nil || number > len(s.keys[key]) {
		return nil, ErrNotFound
	}
	return s.keys[key][number-1], nil
}

func (r *Registry) RecordObservationApproval(workspace string, key Key, number int, approver, reference string) (RevisionView, error) {
	if !validID(workspace) || !validKey(key) || number < 1 || number > maxRevision || !validID(approver) || !validID(reference) {
		return RevisionView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	rev, err := r.findRevision(workspace, key, number)
	if err != nil {
		return RevisionView{}, err
	}
	if rev.approval != nil {
		if rev.approval.Approver != approver || rev.approval.Reference != reference {
			return RevisionView{}, ErrConflict
		}
		return viewRevision(rev), nil
	}
	if rev.state != ObservationAwaiting || approver == rev.input.Recorder {
		return RevisionView{}, ErrRejected
	}
	rev.approval = &Approval{Approver: approver, Reference: reference, At: now}
	rev.state = ObservationApproved
	return viewRevision(rev), nil
}

func (r *Registry) VoidObservation(workspace string, key Key, number int, voider, evidence, reason string) (RevisionView, error) {
	if !validID(workspace) || !validKey(key) || number < 1 || number > maxRevision || !validID(voider) || !validID(evidence) || !voidReasons[reason] {
		return RevisionView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	rev, err := r.findRevision(workspace, key, number)
	if err != nil {
		return RevisionView{}, err
	}
	if rev.void != nil {
		if rev.void.Voider != voider || rev.void.Evidence != evidence || rev.void.Reason != reason {
			return RevisionView{}, ErrConflict
		}
		return viewRevision(rev), nil
	}
	rev.void = &Void{Voider: voider, Evidence: evidence, Reason: reason, At: now}
	rev.state = ObservationVoidAwaiting
	return viewRevision(rev), nil
}

func (r *Registry) RecordVoidApproval(workspace string, key Key, number int, approver, reference string) (RevisionView, error) {
	if !validID(workspace) || !validKey(key) || number < 1 || number > maxRevision || !validID(approver) || !validID(reference) {
		return RevisionView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	rev, err := r.findRevision(workspace, key, number)
	if err != nil {
		return RevisionView{}, err
	}
	if rev.void != nil && rev.void.Approval != nil {
		if rev.void.Approval.Approver != approver || rev.void.Approval.Reference != reference {
			return RevisionView{}, ErrConflict
		}
		return viewRevision(rev), nil
	}
	if rev.void == nil || approver == rev.void.Voider {
		return RevisionView{}, ErrRejected
	}
	rev.void.Approval = &Approval{Approver: approver, Reference: reference, At: now}
	rev.state = ObservationVoid
	return viewRevision(rev), nil
}

func validAttributes(attrs PolicyAttributes) bool {
	keyed := attrs.Level == LevelCustomer || attrs.Level == LevelContract
	return levels[attrs.Level] && keyed == (attrs.Key != "") && (!keyed || validID(attrs.Key)) && purposes[attrs.Purpose] &&
		validCurrency(attrs.Source) && validCurrency(attrs.Target) && attrs.Source != attrs.Target
}

func (r *Registry) pricing(in *VersionInput) bool {
	spread, okSpread := r.share(in.Spread)
	fee, okFee := r.share(in.Fee)
	tolerance, okTolerance := r.positive(in.Tolerance)
	_, toleranceValue, _ := r.number(tolerance)
	limit, _ := exact.Whole(1).Over(exact.Whole(100))
	_, scaled := decimal.New(r.profile, big.NewInt(1), in.QuoteScale)
	_, known := quanta[in.Quantum]
	_, quantumRead := decimal.Parse(r.profile, in.Quantum)
	if !okSpread || !okFee || !okTolerance || limit.Less(toleranceValue) || in.QuoteScale < 0 || in.QuoteScale > maxQuoteScale || scaled != nil ||
		!exact.ValidMode(in.QuoteMode) || !known || quantumRead != nil || !exact.ValidMode(in.AmountMode) || in.Order != Order {
		return false
	}
	in.Spread, in.Fee, in.Tolerance = spread, fee, tolerance
	return true
}

func (r *Registry) validVersion(attrs PolicyAttributes, in *VersionInput) bool {
	_, dated := parseDate(in.Effective)
	if in.Version < 1 || in.Version > maxVersion || !dated || !validID(in.Author) {
		return false
	}
	bare := VersionInput{Version: in.Version, Effective: in.Effective, Author: in.Author, Method: in.Method}
	switch in.Method {
	case MethodEnded:
		return attrs.Level != LevelMSP && *in == bare
	case MethodDefaultAllowed:
		return attrs.Level == LevelContract && *in == bare
	case MethodFixed:
		canonical, ok := r.positive(in.Value)
		if !ok || !units[in.Unit] || !validID(in.Evidence) || (in.Construction != ConstructionDirect && in.Construction != ConstructionInverse) ||
			in.RateSource != "" || in.Series != "" || in.Pivot != "" || in.Rule != "" || in.MaxDays != 0 || in.Month != "" || in.Averaging != "" || in.LeastDays != 0 {
			return false
		}
		in.Value = canonical
	case MethodDated, MethodMonthlyAverage:
		crossed := in.Construction == ConstructionCross
		if !validID(in.RateSource) || !validID(in.Series) || in.Unit != 0 || in.Value != "" || in.Evidence != "" ||
			(in.Construction != ConstructionDirect && in.Construction != ConstructionInverse && !crossed) ||
			crossed != (in.Pivot != "") || (crossed && (!validCurrency(in.Pivot) || in.Pivot == attrs.Source || in.Pivot == attrs.Target)) {
			return false
		}
		if in.Method == MethodDated {
			ruled := (in.Rule == RuleHold && in.MaxDays == 0) || (in.Rule == RulePreviousRecorded && in.MaxDays >= 1 && in.MaxDays <= maxDays)
			if !ruled || in.Month != "" || in.Averaging != "" || in.LeastDays != 0 {
				return false
			}
		} else if (in.Month != MonthSame && in.Month != MonthPrevious) || in.Averaging != Averaging || in.LeastDays < 1 || in.LeastDays > maxDays || in.Rule != "" || in.MaxDays != 0 {
			return false
		}
	default:
		return false
	}
	return r.pricing(in)
}

func viewVersion(v *version) VersionView {
	out := VersionView{VersionInput: v.input, State: v.state, RecordedAt: v.recordedAt, Approval: copyApproval(v.approval)}
	if v.withdrawal != nil {
		made := *v.withdrawal
		out.Withdrawal = &made
	}
	return out
}

func (r *Registry) RecordPolicyVersion(workspace, id string, attrs PolicyAttributes, in VersionInput) (VersionView, error) {
	if !validID(workspace) || !validID(id) || !validAttributes(attrs) || !r.validVersion(attrs, &in) {
		return VersionView{}, ErrInvalid
	}
	effective, _ := parseDate(in.Effective)
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	var p *policy
	if s != nil {
		p = s.policies[id]
	}
	if p == nil && in.Version != 1 {
		return VersionView{}, ErrNotFound
	}
	at := place{attrs.Level, attrs.Key, attrs.Purpose, attrs.Source, attrs.Target}
	if p != nil && p.attrs != attrs {
		return VersionView{}, ErrConflict
	}
	if p != nil && in.Version <= len(p.versions) {
		if p.versions[in.Version-1].input != in {
			return VersionView{}, ErrConflict
		}
		return viewVersion(p.versions[in.Version-1]), nil
	}
	if p == nil && s != nil && s.places[at] != "" {
		return VersionView{}, ErrConflict
	}
	if p != nil {
		if in.Version > len(p.versions)+1 {
			return VersionView{}, ErrRejected
		}
		for _, earlier := range p.versions {
			if (earlier.withdrawal == nil && !earlier.effective.Before(effective)) || effective.Before(earlier.effective) {
				return VersionView{}, ErrRejected
			}
		}
	}
	if (p == nil && s != nil && len(s.policies) >= r.limits.Policies) || (p != nil && len(p.versions) >= r.limits.Versions) {
		return VersionView{}, ErrLimit
	}
	state := VersionRecorded
	if in.Method == MethodFixed {
		state = VersionAwaiting
	}
	v := &version{input: in, effective: effective, state: state, recordedAt: now}
	if p == nil {
		s = r.space(workspace)
		p = &policy{id: id, attrs: attrs}
		s.policies[id] = p
		s.places[at] = id
	}
	p.versions = append(p.versions, v)
	return viewVersion(v), nil
}

func (r *Registry) findVersion(workspace, id string, number int) (*version, error) {
	s := r.workspaces[workspace]
	if s == nil || s.policies[id] == nil || number > len(s.policies[id].versions) {
		return nil, ErrNotFound
	}
	return s.policies[id].versions[number-1], nil
}

func (r *Registry) RecordPolicyApproval(workspace, id string, number int, approver, reference string) (VersionView, error) {
	if !validID(workspace) || !validID(id) || number < 1 || number > maxVersion || !validID(approver) || !validID(reference) {
		return VersionView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	v, err := r.findVersion(workspace, id, number)
	if err != nil {
		return VersionView{}, err
	}
	if v.approval != nil {
		if v.approval.Approver != approver || v.approval.Reference != reference {
			return VersionView{}, ErrConflict
		}
		return viewVersion(v), nil
	}
	if v.state != VersionAwaiting || v.withdrawal != nil || approver == v.input.Author {
		return VersionView{}, ErrRejected
	}
	v.approval = &Approval{Approver: approver, Reference: reference, At: now}
	v.state = VersionApproved
	return viewVersion(v), nil
}

func (r *Registry) WithdrawPolicyVersion(workspace, id string, number int, withdrawer, reason string) (VersionView, error) {
	if !validID(workspace) || !validID(id) || number < 1 || number > maxVersion || !validID(withdrawer) || !withdrawReason[reason] {
		return VersionView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	v, err := r.findVersion(workspace, id, number)
	if err != nil {
		return VersionView{}, err
	}
	if v.withdrawal != nil {
		if v.withdrawal.Withdrawer != withdrawer || v.withdrawal.Reason != reason {
			return VersionView{}, ErrConflict
		}
		return viewVersion(v), nil
	}
	v.withdrawal = &Withdrawal{Withdrawer: withdrawer, Reason: reason, At: now}
	return viewVersion(v), nil
}

func (p *policy) on(date time.Time) *version {
	index := sort.Search(len(p.versions), func(i int) bool { return p.versions[i].effective.After(date) })
	if index == 0 {
		return nil
	}
	return p.versions[index-1]
}
