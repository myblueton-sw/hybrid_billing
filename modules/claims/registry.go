package claims

import (
	"sort"
	"sync"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/exact"
)

type component struct {
	c         Component
	instances map[string]bool
}

type revision struct {
	number      int
	r           Revision
	components  map[string]*component
	effectiveTo string
	author      string
	approver    string
	digest      string
	writtenAt   time.Time
	approvedAt  time.Time
}

type reversal struct {
	r          Reversal
	state      string
	author     string
	approver   string
	digest     string
	writtenAt  time.Time
	approvedAt time.Time
}

type application struct {
	a           Application
	state       string
	expiry      time.Time
	reversals   []*reversal
	digest      string
	reservedBy  string
	reservedAt  time.Time
	confirmedBy string
	confirmedAt time.Time
	releasedBy  string
	releasedAt  time.Time
	replacedBy  string
	replacedAt  time.Time
}

type source struct {
	quantity string
	amount   string
	currency string
	apps     []*application
}

type space struct {
	contracts    map[string][]*revision
	applications map[string]*application
	scopes       map[Scope][]*application
	sources      map[string]*source
	reversals    map[string]*reversal
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

func emptySpace() *space {
	return &space{contracts: map[string][]*revision{}, applications: map[string]*application{}, scopes: map[Scope][]*application{},
		sources: map[string]*source{}, reversals: map[string]*reversal{}}
}

func (r *Registry) space(workspace string) *space {
	s := r.workspaces[workspace]
	if s == nil {
		s = emptySpace()
		r.workspaces[workspace] = s
	}
	return s
}

func (r *Registry) amount(text string) (string, bool) {
	if len(text) > maxNumberBytes {
		return "", false
	}
	value, err := decimal.Parse(r.profile, text)
	if err != nil || value.Sign() < 0 {
		return "", false
	}
	return value.String(), true
}

func (r *Registry) value(text string) exact.Fraction {
	parsed, _ := decimal.Parse(r.profile, text)
	made, _ := exact.Of(parsed)
	return made
}

func same(a, b exact.Fraction) bool { return !a.Less(b) && !b.Less(a) }

func revisionForm(rev Revision) (Revision, bool) {
	if !validDate(rev.From) || !validDate(rev.To) || rev.To <= rev.From || !paths[rev.Path] || len(rev.Components) == 0 {
		return Revision{}, false
	}
	out := Revision{From: rev.From, To: rev.To, Path: rev.Path, Components: make([]Component, 0, len(rev.Components))}
	seen := map[string]bool{}
	for _, c := range rev.Components {
		if !validID(c.ID) || seen[c.ID] || !kinds[c.Kind] || !routes[c.Route] || len(c.Instances) == 0 {
			return Revision{}, false
		}
		if (c.Route == RouteExternal) != (c.Reference != "") || !optionalID(c.Reference) {
			return Revision{}, false
		}
		if rev.Path == PathDirect && c.Kind == KindCSPUsage && c.Billable {
			return Revision{}, false
		}
		instances := append([]string{}, c.Instances...)
		sort.Strings(instances)
		for index, instance := range instances {
			if !validID(instance) || index > 0 && instances[index-1] == instance {
				return Revision{}, false
			}
		}
		seen[c.ID] = true
		c.Instances = instances
		out.Components = append(out.Components, c)
	}
	sort.Slice(out.Components, func(a, b int) bool { return out.Components[a].ID < out.Components[b].ID })
	return out, true
}

func sameRevision(a, b Revision) bool {
	if a.From != b.From || a.To != b.To || a.Path != b.Path || len(a.Components) != len(b.Components) {
		return false
	}
	for index := range a.Components {
		x, y := a.Components[index], b.Components[index]
		if x.ID != y.ID || x.Kind != y.Kind || x.Billable != y.Billable || x.Route != y.Route || x.Reference != y.Reference || len(x.Instances) != len(y.Instances) {
			return false
		}
		for i := range x.Instances {
			if x.Instances[i] != y.Instances[i] {
				return false
			}
		}
	}
	return true
}

func copyRevision(rev Revision) Revision {
	out := rev
	out.Components = make([]Component, 0, len(rev.Components))
	for _, c := range rev.Components {
		c.Instances = append([]string{}, c.Instances...)
		out.Components = append(out.Components, c)
	}
	return out
}

func viewEligibility(contract string, rev *revision) EligibilityView {
	return EligibilityView{Contract: contract, Number: rev.number, Revision: copyRevision(rev.r), EffectiveTo: rev.effectiveTo, Author: rev.author, Approver: rev.approver,
		Digest: rev.digest, WrittenAt: rev.writtenAt, ApprovedAt: rev.approvedAt}
}

func (r *Registry) RecordEligibility(workspace, contract string, number int, rev Revision, author string) (EligibilityView, error) {
	if len(rev.Components) > r.limits.Components {
		return EligibilityView{}, ErrLimit
	}
	for _, c := range rev.Components {
		if len(c.Instances) > maxInstances {
			return EligibilityView{}, ErrLimit
		}
	}
	canonical, formed := revisionForm(rev)
	if !validID(workspace) || !validID(contract) || !validID(author) || number < 1 || number > 1000 || !formed {
		return EligibilityView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	var revisions []*revision
	if s != nil {
		revisions = s.contracts[contract]
	}
	if number <= len(revisions) {
		existing := revisions[number-1]
		if !sameRevision(existing.r, canonical) || existing.author != author {
			return EligibilityView{}, ErrConflict
		}
		return viewEligibility(contract, existing), nil
	}
	if number > len(revisions)+1 {
		return EligibilityView{}, ErrRejected
	}
	if len(revisions) >= r.limits.Revisions || len(revisions) == 0 && s != nil && len(s.contracts) >= r.limits.Contracts {
		return EligibilityView{}, ErrLimit
	}
	s = r.space(workspace)
	stored := &revision{number: number, r: canonical, components: map[string]*component{}, effectiveTo: canonical.To, author: author, writtenAt: now,
		digest: eligibilityDigest(workspace, contract, number, canonical)}
	for _, c := range canonical.Components {
		instances := map[string]bool{}
		for _, instance := range c.Instances {
			instances[instance] = true
		}
		stored.components[c.ID] = &component{c: c, instances: instances}
	}
	s.contracts[contract] = append(revisions, stored)
	return viewEligibility(contract, stored), nil
}

func (r *Registry) ApproveEligibility(workspace, contract string, number int, approver string) (EligibilityView, error) {
	if !validID(workspace) || !validID(contract) || !validID(approver) || number < 1 || number > 1000 {
		return EligibilityView{}, ErrInvalid
	}
	now := r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || number > len(s.contracts[contract]) {
		return EligibilityView{}, ErrNotFound
	}
	rev := s.contracts[contract][number-1]
	if rev.approver != "" {
		if rev.approver != approver {
			return EligibilityView{}, ErrConflict
		}
		return viewEligibility(contract, rev), nil
	}
	if approver == rev.author {
		return EligibilityView{}, ErrRejected
	}
	ended := []*revision{}
	for _, other := range s.contracts[contract] {
		if other.approver == "" || other.r.From >= rev.r.To || rev.r.From >= other.effectiveTo {
			continue
		}
		if other.r.From >= rev.r.From || rev.r.To < other.effectiveTo {
			return EligibilityView{}, ErrRejected
		}
		ended = append(ended, other)
	}
	if len(ended) > 0 {
		for _, app := range s.applications {
			if app.a.Scope.Contract != contract || !r.active(app, now) || app.a.End <= rev.r.From || app.a.Start >= rev.r.To {
				continue
			}
			c := rev.components[app.a.Scope.Component]
			if c == nil || !c.c.Billable || c.c.Route != RoutePlatform || !c.instances[app.a.Scope.Instance] {
				return EligibilityView{}, ErrRejected
			}
		}
	}
	for _, other := range ended {
		other.effectiveTo = rev.r.From
	}
	rev.approver, rev.approvedAt = approver, now
	return viewEligibility(contract, rev), nil
}

func (r *Registry) Eligibility(workspace, contract string, number int) (EligibilityView, error) {
	if !validID(workspace) || !validID(contract) || number < 1 || number > 1000 {
		return EligibilityView{}, ErrInvalid
	}
	r.enter()
	defer r.leave()
	s := r.workspaces[workspace]
	if s == nil || number > len(s.contracts[contract]) {
		return EligibilityView{}, ErrNotFound
	}
	return viewEligibility(contract, s.contracts[contract][number-1]), nil
}

func covering(revisions []*revision, start, end string) *revision {
	for _, rev := range revisions {
		if rev.approver != "" && rev.r.From <= start && end <= rev.effectiveTo {
			return rev
		}
	}
	return nil
}

func eligibilityReason(revisions []*revision, a Application) string {
	rev := covering(revisions, a.Start, a.End)
	if rev == nil {
		return ReasonNoEligibility
	}
	c := rev.components[a.Scope.Component]
	switch {
	case c == nil || !c.c.Billable:
		return ReasonNotEligible
	case !c.instances[a.Scope.Instance]:
		return ReasonUnknownInstance
	case c.c.Route == RouteExternal:
		return ReasonRoutedExternally
	case c.c.Route == RouteUnknown:
		return ReasonRouteUnknown
	}
	return ""
}
